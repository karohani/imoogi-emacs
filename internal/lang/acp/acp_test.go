package acp

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
	"github.com/karohani/imoogi-emacs/internal/npmbundle"
)

const fakeNodeScript = `#!/bin/sh
printf '%s|%s|claude=%s|codex=%s\n' "$0" "$*" "${CLAUDE_CODE_EXECUTABLE-unset}" "${CODEX_PATH-unset}" >>"$IMOOGI_FAKE_NODE_LOG"
`

func put(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}

func lockComponent(t *testing.T, repo, name, kind, source, version, rel string, data []byte) config.LockComponent {
	t.Helper()
	put(t, filepath.Join(repo, filepath.FromSlash(rel)), data, 0o644)
	sum := sha256.Sum256(data)
	return config.LockComponent{
		Name: name, Kind: kind, Source: source, UpstreamVersion: version,
		Artifact: config.Artifact{Path: rel, Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])},
		Probe:    config.Probe{Command: "bin/" + name},
	}
}

func nodeArchive(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte(fakeNodeScript)
	if err := tw.WriteHeader(&tar.Header{Name: "node-v24.19.0-darwin-arm64/bin/node", Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// adapterBundle packs a node_modules tree the way fetch does.
func adapterBundle(t *testing.T, pkg, version string) []byte {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "node_modules", filepath.FromSlash(pkg))
	put(t, filepath.Join(dir, "package.json"), []byte(`{"name":"`+pkg+`","version":"`+version+`","bin":{"x":"dist/index.js"}}`), 0o644)
	put(t, filepath.Join(dir, "dist", "index.js"), []byte("// adapter\n"), 0o644)
	var buf bytes.Buffer
	if err := npmbundle.Pack(root, &buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type fixture struct {
	repo, staging string
	node          config.LockComponent
	adapters      []config.LockComponent
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo with spaces")
	f := fixture{repo: repo, staging: filepath.Join(repo, ".local", "toolchains", "2026.11.14.1")}
	f.node = lockComponent(t, repo, "node", "node-runtime", "nodejs.org/dist", "v24.19.0", "vendor/toolchains/node/node.tar.gz", nodeArchive(t))
	for _, a := range []struct{ name, version string }{{"claude-agent-acp", "0.99.1"}, {"codex-acp", "2.5.0"}} {
		pkg := "@agentclientprotocol/" + a.name
		f.adapters = append(f.adapters, lockComponent(t, repo, a.name, config.KindACPAdapter, "npm:"+pkg, a.version,
			"vendor/toolchains/acp/"+a.name+"/"+a.version+"/darwin-arm64/bundle.tgz", adapterBundle(t, pkg, a.version)))
	}
	return f
}

func run(t *testing.T, command string, env []string) {
	t.Helper()
	cmd := exec.Command(command)
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", command, err, out)
	}
}

func TestProviderInstallsLaunchersOnTheBundledNode(t *testing.T) {
	f := newFixture(t)
	probes, err := New(f.adapters, f.node).Materialize(f.repo, f.staging)
	if err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}
	if len(probes) != 2 || probes[0].Command != filepath.Join(f.staging, "bin", "claude-agent-acp") || len(probes[0].Args) != 0 {
		t.Fatalf("probes = %+v", probes)
	}
	for _, name := range []string{"claude-agent-acp", "codex-acp"} {
		info, err := os.Stat(filepath.Join(f.staging, "bin", name))
		if err != nil || info.Mode().Perm() != 0o755 {
			t.Fatalf("launcher %s: %v %v", name, info, err)
		}
	}
	if _, err := os.Stat(filepath.Join(f.staging, "lib", "acp", "codex-acp", "node_modules", "@agentclientprotocol", "codex-acp", "dist", "index.js")); err != nil {
		t.Fatal(err)
	}

	// Through the activation symlink, as Emacs runs it.
	binLink := filepath.Join(f.repo, ".local", "bin")
	if err := os.Symlink(filepath.Join("toolchains", "2026.11.14.1", "bin"), binLink); err != nil {
		t.Fatal(err)
	}
	clis := t.TempDir()
	put(t, filepath.Join(clis, "claude"), []byte("#!/bin/sh\n"), 0o755)
	logFile := filepath.Join(t.TempDir(), "node.log")
	base := []string{"IMOOGI_FAKE_NODE_LOG=" + logFile}

	run(t, filepath.Join(binLink, "claude-agent-acp"), append(base, "PATH="+clis+":/usr/bin:/bin"))
	run(t, filepath.Join(binLink, "claude-agent-acp"), append(base, "PATH=/usr/bin:/bin"))
	run(t, filepath.Join(binLink, "claude-agent-acp"), append(base, "PATH="+clis+":/usr/bin:/bin", "CLAUDE_CODE_EXECUTABLE=/opt/mine/claude"))
	put(t, filepath.Join(clis, "codex"), []byte("#!/bin/sh\n"), 0o755)
	run(t, filepath.Join(binLink, "codex-acp"), append(base, "PATH="+clis+":/usr/bin:/bin"))

	lines := strings.Split(strings.TrimSpace(string(mustRead(t, logFile))), "\n")
	physical, err := filepath.EvalSymlinks(f.staging) // launchers resolve with cd -P
	if err != nil {
		t.Fatal(err)
	}
	nodeBin := filepath.Join(physical, "lib", "node", "node-v24.19.0-darwin-arm64", "bin", "node")
	claudeEntry := filepath.Join(physical, "lib", "acp", "claude-agent-acp", "node_modules", "@agentclientprotocol", "claude-agent-acp", "dist", "index.js")
	want := []string{
		nodeBin + "|" + claudeEntry + "|claude=" + filepath.Join(clis, "claude") + "|codex=unset",
		nodeBin + "|" + claudeEntry + "|claude=unset|codex=unset",
		nodeBin + "|" + claudeEntry + "|claude=/opt/mine/claude|codex=unset",
	}
	for i, w := range want {
		if lines[i] != w {
			t.Fatalf("run %d:\n got %s\nwant %s", i, lines[i], w)
		}
	}
	if !strings.HasSuffix(lines[3], "codex-acp/dist/index.js|claude=unset|codex="+filepath.Join(clis, "codex")) {
		t.Fatalf("codex run = %s", lines[3])
	}
}

func TestProviderRejectsBadInputs(t *testing.T) {
	tests := map[string]func(*fixture){
		"no node":          func(f *fixture) { f.node = config.LockComponent{} },
		"unknown adapter":  func(f *fixture) { f.adapters[0].Name = "gemini-acp" },
		"artifact digest":  func(f *fixture) { f.adapters[1].Artifact.SHA256 = strings.Repeat("0", 64) },
		"bundle version":   func(f *fixture) { f.adapters[0].UpstreamVersion = "0.99.2" },
		"node engine":      func(f *fixture) { f.adapters[0].Runtime.NodeEngine = ">=26.0.0" },
		"non-npm source":   func(f *fixture) { f.adapters[0].Source = "github.com/x" },
		"probe escapes":    func(f *fixture) { f.adapters[0].Probe.Command = "../x" },
		"wrong probe name": func(f *fixture) { f.adapters[0].Probe.Command = "bin/codex-acp" },
	}
	for name, edit := range tests {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			edit(&f)
			if _, err := New(f.adapters, f.node).Materialize(f.repo, f.staging); err == nil {
				t.Fatal("Materialize() succeeded")
			}
		})
	}
}

func TestProviderReusesNodeAnotherProviderStaged(t *testing.T) {
	f := newFixture(t)
	if _, err := New(f.adapters[:1], f.node).Materialize(f.repo, f.staging); err != nil {
		t.Fatal(err)
	}
	if _, err := New(f.adapters[1:], f.node).Materialize(f.repo, f.staging); err != nil {
		t.Fatalf("second provider on the same staging tree: %v", err)
	}
}

func TestSupportsComponentAndAdapters(t *testing.T) {
	if !SupportsComponent(config.LockComponent{Kind: config.KindACPAdapter}) || SupportsComponent(config.LockComponent{Kind: "node-runtime"}) {
		t.Fatal("SupportsComponent misclassifies")
	}
	a, ok := Lookup("codex-acp")
	if !ok || a.CLI != "codex" || a.EnvVar != "CODEX_PATH" {
		t.Fatalf("Lookup(codex-acp) = %+v, %v", a, ok)
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
