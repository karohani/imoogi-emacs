package typescript

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/config"
)

func TestProviderMaterializesLauncherWithVendoredNodeAndRelocation(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo with spaces")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	components := writeFakeArtifacts(t, repo, "v22.14.0", ">=20.0.0 <23.0.0")
	staging := filepath.Join(repo, ".local", "toolchains", "2026.08.15.1")

	probes, err := New(components).Materialize(repo, staging)
	if err != nil {
		t.Fatalf("Materialize failed: %v", err)
	}
	if got, want := len(probes), 3; got != want {
		t.Fatalf("probes = %d, want %d", got, want)
	}
	assertMode(t, filepath.Join(staging, "bin", "node"), 0o755)
	assertMode(t, filepath.Join(staging, "bin", "tsc"), 0o755)
	assertMode(t, filepath.Join(staging, "bin", "typescript-language-server"), 0o755)
	assertMode(t, filepath.Join(staging, "lib", "node", "node-v22.14.0-darwin-arm64", "bin", "node"), 0o755)
	if _, err := os.Stat(filepath.Join(staging, "lib", "node_modules", "typescript", "package.json")); err != nil {
		t.Fatalf("typescript sibling package is not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(staging, "lib", "node_modules", "typescript-language-server", "package.json")); err != nil {
		t.Fatalf("typescript-language-server package is not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(staging, "lib", "node", "node-v22.14.0-darwin-arm64", "README.md")); err != nil {
		t.Fatalf("node distribution was not preserved: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".local", "bin")); !os.IsNotExist(err) {
		t.Fatalf("provider created activation path; err=%v", err)
	}

	binLink := filepath.Join(repo, ".local", "bin")
	if err := os.Symlink(filepath.Join("toolchains", "2026.08.15.1", "bin"), binLink); err != nil {
		t.Fatal(err)
	}
	sentinelDir := filepath.Join(repo, "sentinel")
	writeFile(t, filepath.Join(sentinelDir, "node"), []byte("#!/bin/sh\necho system node sentinel >&2\nexit 44\n"), 0o755)
	logFile := filepath.Join(repo, "node.log")
	pathWithSentinel := "PATH=" + sentinelDir + string(os.PathListSeparator) + os.Getenv("PATH")
	runWithEnv(t, filepath.Join(binLink, "node"), []string{"--version"}, append(os.Environ(), pathWithSentinel, "IMOOGI_FAKE_NODE_LOG="+logFile))
	runWithEnv(t, filepath.Join(binLink, "typescript-language-server"), []string{"--help"}, append(os.Environ(), pathWithSentinel, "IMOOGI_FAKE_NODE_LOG="+logFile))
	log := mustRead(t, logFile)
	if strings.Contains(string(log), "system node sentinel") {
		t.Fatalf("system node sentinel was used: %s", log)
	}
	if !strings.Contains(string(log), filepath.Join(staging, "lib", "node", "node-v22.14.0-darwin-arm64", "bin", "node")+"|") {
		t.Fatalf("log does not show physical vendored node path: %s", log)
	}
	if !strings.Contains(string(log), "node_modules/typescript-language-server/lib/cli.mjs --help ") {
		t.Fatalf("launcher did not invoke vendored TLS CLI directly: %s", log)
	}
	runWithEnv(t, filepath.Join(binLink, "tsc"), []string{"--version"}, append(os.Environ(), pathWithSentinel, "IMOOGI_FAKE_NODE_LOG="+logFile))
	if !strings.Contains(string(mustRead(t, logFile)), "node_modules/typescript/lib/tsc.js --version ") {
		t.Fatalf("tsc launcher did not invoke vendored TypeScript bin: %s", mustRead(t, logFile))
	}
	if strings.Contains(string(log), "--tsserver-path") {
		t.Fatalf("launcher used unsupported --tsserver-path flag: %s", log)
	}

	relocated := filepath.Join(t.TempDir(), "relocated repo with spaces")
	if err := os.Rename(repo, relocated); err != nil {
		t.Fatal(err)
	}
	relocatedLog := filepath.Join(relocated, "node.log")
	runWithEnv(t, filepath.Join(relocated, ".local", "bin", "typescript-language-server"), []string{"--help"}, append(os.Environ(), pathWithSentinel, "IMOOGI_FAKE_NODE_LOG="+relocatedLog))
	if !strings.Contains(string(mustRead(t, relocatedLog)), filepath.Join(relocated, ".local", "toolchains", "2026.08.15.1", "lib", "node")) {
		t.Fatalf("relocated launcher did not resolve relocated physical bundle: %s", mustRead(t, relocatedLog))
	}
}

func TestProviderRejectsIntegrityFailureAndEngineMismatch(t *testing.T) {
	t.Run("integrity", func(t *testing.T) {
		repo := t.TempDir()
		components := writeFakeArtifacts(t, repo, "v22.14.0", ">=20.0.0 <23.0.0")
		for i := range components {
			if components[i].Name == "typescript" {
				components[i].Artifact.SHA256 = strings.Repeat("0", 64)
			}
		}
		staging := filepath.Join(repo, ".local", "toolchains", "2026.08.15.1")
		if _, err := New(components).Materialize(repo, staging); err == nil {
			t.Fatal("Materialize succeeded with bad TypeScript digest")
		}
		if _, err := os.Lstat(filepath.Join(staging, "bin", "typescript-language-server")); !os.IsNotExist(err) {
			t.Fatalf("launcher exists after integrity failure; err=%v", err)
		}
	})

	t.Run("engine", func(t *testing.T) {
		repo := t.TempDir()
		components := writeFakeArtifacts(t, repo, "v18.19.0", ">=20.0.0 <23.0.0")
		if _, err := New(components).Materialize(repo, filepath.Join(repo, ".local", "toolchains", "2026.08.15.1")); err == nil {
			t.Fatal("Materialize succeeded with incompatible Node engine")
		}
	})
}

func TestProviderRejectsMissingAuthority(t *testing.T) {
	repo := t.TempDir()
	components := writeFakeArtifacts(t, repo, "v22.14.0", ">=20.0.0 <23.0.0")
	components = components[:2]
	if _, err := New(components).Materialize(repo, filepath.Join(repo, ".local", "toolchains", "2026.08.15.1")); err == nil {
		t.Fatal("Materialize succeeded without typescript-language-server component")
	}
}

type tarEntry struct {
	name string
	mode int64
	body []byte
	dir  bool
}

func writeFakeArtifacts(t *testing.T, repo, nodeVersion, engine string) []config.LockComponent {
	t.Helper()
	nodeDist := "node-" + nodeVersion + "-darwin-arm64"
	nodeScript := []byte("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo " + nodeVersion + "; fi\ncase \"${1:-}\" in\n  *node_modules/typescript-language-server/lib/cli.mjs)\n    if [ ! -f \"$(dirname \"$1\")/../../typescript/lib/tsserver.js\" ]; then\n      echo \"missing sibling typescript tsserver\" >&2\n      exit 45\n    fi\n    ;;\nesac\nif [ -n \"${IMOOGI_FAKE_NODE_LOG:-}\" ]; then printf '%s|' \"$0\" >> \"$IMOOGI_FAKE_NODE_LOG\"; printf '%s ' \"$@\" >> \"$IMOOGI_FAKE_NODE_LOG\"; printf '\\n' >> \"$IMOOGI_FAKE_NODE_LOG\"; fi\nexit 0\n")
	nodeArchive := writeTGZ(t, repo, []tarEntry{
		{name: nodeDist, dir: true, mode: 0o755},
		{name: nodeDist + "/bin", dir: true, mode: 0o755},
		{name: nodeDist + "/bin/node", body: nodeScript, mode: 0o755},
		{name: nodeDist + "/README.md", body: []byte("full distribution marker\n"), mode: 0o644},
	})
	tsArchive := writeTGZ(t, repo, []tarEntry{
		{name: "package", dir: true, mode: 0o755},
		{name: "package/package.json", body: []byte(`{"name":"typescript"}`), mode: 0o644},
		{name: "package/lib", dir: true, mode: 0o755},
		{name: "package/lib/tsc.js", body: []byte("// fake tsc\n"), mode: 0o644},
		{name: "package/lib/tsserver.js", body: []byte("// fake tsserver\n"), mode: 0o644},
	})
	tlsArchive := writeTGZ(t, repo, []tarEntry{
		{name: "package", dir: true, mode: 0o755},
		{name: "package/package.json", body: []byte(`{"name":"typescript-language-server"}`), mode: 0o644},
		{name: "package/lib", dir: true, mode: 0o755},
		{name: "package/lib/cli.mjs", body: []byte("// fake tls cli\n"), mode: 0o644},
	})

	return []config.LockComponent{
		component(t, "node", "node-runtime", "nodejs.org/dist", nodeVersion, repo, "vendor/toolchains/typescript/node/"+nodeVersion+"/darwin-arm64/node.tar.gz", nodeArchive, config.Probe{Command: "bin/node", Args: []string{"--version"}}, config.RuntimeConstraint{}),
		component(t, "typescript", "typescript-sdk", "npm:typescript", "6.0.3", repo, "vendor/toolchains/typescript/typescript/6.0.3/darwin-arm64/typescript.tgz", tsArchive, config.Probe{Command: "bin/tsc", Args: []string{"--version"}}, config.RuntimeConstraint{}),
		component(t, "typescript-language-server", "typescript-language-server", "npm:typescript-language-server", "4.3.3", repo, "vendor/toolchains/typescript/typescript-language-server/4.3.3/darwin-arm64/server.tgz", tlsArchive, config.Probe{Command: "bin/typescript-language-server", Args: []string{"--help"}}, config.RuntimeConstraint{NodeEngine: engine}),
	}
}

func component(t *testing.T, name, kind, source, version, repo, relPath string, archive []byte, probe config.Probe, runtime config.RuntimeConstraint) config.LockComponent {
	t.Helper()
	sum := sha256.Sum256(archive)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(relPath)), archive, 0o600)
	return config.LockComponent{
		Name:            name,
		Kind:            kind,
		Source:          source,
		UpstreamVersion: version,
		Artifact: config.Artifact{
			Path:   relPath,
			Size:   int64(len(archive)),
			SHA256: hex.EncodeToString(sum[:]),
		},
		Runtime: runtime,
		Probe:   probe,
	}
}

func writeTGZ(t *testing.T, root string, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, entry := range entries {
		typeflag := byte(tar.TypeReg)
		size := int64(len(entry.body))
		if entry.dir {
			typeflag = tar.TypeDir
			size = 0
		}
		hdr := tar.Header{Name: entry.name, Typeflag: typeflag, Mode: entry.mode, Size: size}
		if err := tw.WriteHeader(&hdr); err != nil {
			t.Fatalf("WriteHeader(%q): %v", entry.name, err)
		}
		if !entry.dir {
			if _, err := tw.Write(entry.body); err != nil {
				t.Fatalf("Write(%q): %v", entry.name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeFile(t interface {
	Helper()
	Fatal(...any)
}, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}

func runWithEnv(t *testing.T, command string, args []string, env []string) {
	t.Helper()
	cmd := exec.Command(command, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", command, args, err, out)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %o, want %o", path, got, want)
	}
}
