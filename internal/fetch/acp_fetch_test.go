package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/npmbundle"
)

const acpLockfile = `{"lockfileVersion":3,"packages":{"":{"dependencies":{"@agentclientprotocol/codex-acp":"2.5.0"}},"node_modules/@agentclientprotocol/codex-acp":{"version":"2.5.0"}}}`

func acpRevision() string {
	sum := sha256.Sum256([]byte(acpLockfile))
	return hex.EncodeToString(sum[:])
}

// writeACPManifest adds a codex-acp adapter (with the given policy) to the
// standard fixture manifest and commits its package inputs.
func writeACPManifest(t *testing.T, repo, policy string) {
	t.Helper()
	writeManifest(t, repo)
	path := filepath.Join(repo, "toolchains.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	entry := `{"name": "codex-acp", "kind": "acp-adapter", "source": "npm:@agentclientprotocol/codex-acp", "upstream_version": "2.5.0", "revision": "` + acpRevision() + `"`
	if policy != "" {
		entry += `, "artifact_policy": "` + policy + `"`
	}
	entry += "},\n    "
	updated := strings.Replace(string(data), `{"name": "gopls"`, entry+`{"name": "gopls"`, 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(repo, "toolchains", "acp", "codex-acp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"@agentclientprotocol/codex-acp":"2.5.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(acpLockfile), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fakeACPInstall(calls *int, marker string) npmbundle.Runner {
	return func(_ context.Context, dir string, argv []string) error {
		*calls++
		if !strings.HasSuffix(argv[0], "node-v24.19.0-darwin-arm64/bin/node") {
			return errors.New("npm not run with the fetched node: " + argv[0])
		}
		pkg := filepath.Join(dir, "node_modules", "@agentclientprotocol", "codex-acp")
		if err := os.MkdirAll(filepath.Join(pkg, "dist"), 0o755); err != nil {
			return err
		}
		for name, body := range map[string]string{
			"package.json":  `{"name":"@agentclientprotocol/codex-acp","version":"2.5.0","bin":{"codex-acp":"dist/index.js"}}`,
			"dist/index.js": marker,
			"LICENSE":       "codex-acp license\n",
		} {
			if err := os.WriteFile(filepath.Join(pkg, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
				return err
			}
		}
		return nil
	}
}

func acpServer(t *testing.T) *httptest.Server {
	t.Helper()
	node := tarGz(t, map[string]string{
		"node-v24.19.0-darwin-arm64/LICENSE":                             "node license\n",
		"node-v24.19.0-darwin-arm64/bin/node":                            "node\n",
		"node-v24.19.0-darwin-arm64/lib/node_modules/npm/bin/npm-cli.js": "npm\n",
	})
	ts := tarGz(t, map[string]string{
		"package/LICENSE.txt":              "typescript license\n",
		"package/ThirdPartyNoticeText.txt": "typescript notice\n",
		"package/lib/tsserver.js":          "server\n",
	})
	tls := tarGz(t, map[string]string{"package/LICENSE": "tls license\n"})
	server := fixtureServer(t, map[string]int{}, map[string][]byte{
		"/dist/v24.19.0/node-v24.19.0-darwin-arm64.tar.gz":                   node,
		"/dist/v24.19.0/SHASUMS256.txt":                                      []byte(shaLine(node, "node-v24.19.0-darwin-arm64.tar.gz")),
		"/typescript/-/typescript-6.0.3.tgz":                                 ts,
		"/typescript-language-server/-/typescript-language-server-6.0.0.tgz": tls,
	})
	t.Cleanup(server.Close)
	return server
}

func runACPFetch(t *testing.T, repo string, server *httptest.Server, run npmbundle.Runner) error {
	t.Helper()
	return Run(context.Background(), Options{
		Workdir:   repo,
		Client:    server.Client(),
		Now:       fixedTime("2026-11-14T00:00:00Z"),
		Sources:   Sources{NodeDistBase: server.URL + "/dist", NPMRegistry: server.URL, GoProxy: server.URL + "/proxy"},
		Go:        writeFakeGo(t, repo, "cli-bin\n"),
		NPMRunner: run,
	})
}

const acpArtifact = "vendor/toolchains/acp/codex-acp/2.5.0/darwin-arm64/bundle.tgz"

func TestRunBuildsACPAdapterFromTheCommittedLockfile(t *testing.T) {
	repo := t.TempDir()
	writeACPManifest(t, repo, "")
	server := acpServer(t)
	var calls int
	if err := runACPFetch(t, repo, server, fakeACPInstall(&calls, "v1")); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("npm runs = %d", calls)
	}
	assertFile(t, repo, "vendor/toolchains/licenses/codex-acp-LICENSE", "codex-acp license\n")
	lock, err := config.LoadLock(filepath.Join(repo, "toolchains.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got config.LockComponent
	for _, c := range lock.Components {
		if c.Name == "codex-acp" {
			got = c
		}
	}
	data, err := os.ReadFile(filepath.Join(repo, acpArtifact))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if got.Artifact.Path != acpArtifact || got.Artifact.SHA256 != hex.EncodeToString(sum[:]) || got.Revision != acpRevision() {
		t.Fatalf("locked component = %+v", got)
	}
	if got.Artifact.SourceURL != "" && !strings.HasSuffix(got.Artifact.SourceURL, "/@agentclientprotocol/codex-acp/-/codex-acp-2.5.0.tgz") {
		t.Fatalf("source_url = %s", got.Artifact.SourceURL)
	}
	if !strings.Contains(got.Provenance.Command, "toolchains/acp/codex-acp/package-lock.json") || !strings.Contains(got.Provenance.Command, "npm ci") {
		t.Fatalf("provenance command = %q", got.Provenance.Command)
	}
	if got.Probe.Command != "bin/codex-acp" || len(got.Install) != 2 {
		t.Fatalf("probe/install = %+v %+v", got.Probe, got.Install)
	}

	// A committed artifact that matches the lock is reused without npm.
	if err := runACPFetch(t, repo, server, fakeACPInstall(&calls, "v1")); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("npm ran again for a matching artifact (%d)", calls)
	}
}

func TestRunRebuildsALocalBuildArtifactAndCatchesDrift(t *testing.T) {
	repo := t.TempDir()
	writeACPManifest(t, repo, config.ArtifactPolicyLocalBuild)
	server := acpServer(t)
	var calls int
	if err := runACPFetch(t, repo, server, fakeACPInstall(&calls, "v1")); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	lockBefore, err := os.ReadFile(filepath.Join(repo, "toolchains.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(lockBefore), `"artifact_policy": "local-build"`) {
		t.Fatalf("lock lost the artifact policy:\n%s", lockBefore)
	}

	// The local artifact is gone (not committed): fetch rebuilds the same bytes.
	if err := os.Remove(filepath.Join(repo, acpArtifact)); err != nil {
		t.Fatal(err)
	}
	if err := runACPFetch(t, repo, server, fakeACPInstall(&calls, "v1")); err != nil {
		t.Fatalf("rebuild Run() error = %v", err)
	}
	lockAfter, err := os.ReadFile(filepath.Join(repo, "toolchains.lock.json"))
	if err != nil || string(lockAfter) != string(lockBefore) {
		t.Fatalf("a deterministic rebuild changed the lock")
	}

	// A rebuild with different bytes under the same identity is drift.
	if err := os.Remove(filepath.Join(repo, acpArtifact)); err != nil {
		t.Fatal(err)
	}
	err = runACPFetch(t, repo, server, fakeACPInstall(&calls, "v2"))
	if err == nil || !strings.Contains(err.Error(), "different artifact digest") {
		t.Fatalf("Run() error = %v, want drift", err)
	}
	if _, statErr := os.Stat(filepath.Join(repo, acpArtifact)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("drifted artifact was published")
	}
}

func TestRunRejectsAStaleACPRevision(t *testing.T) {
	repo := t.TempDir()
	writeACPManifest(t, repo, "")
	if err := os.WriteFile(filepath.Join(repo, "toolchains", "acp", "codex-acp", "package-lock.json"), []byte(acpLockfile+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	server := acpServer(t)
	var calls int
	err := runACPFetch(t, repo, server, fakeACPInstall(&calls, "v1"))
	if err == nil || !strings.Contains(err.Error(), "package-lock.json") {
		t.Fatalf("Run() error = %v, want stale revision rejection", err)
	}
	if calls != 0 {
		t.Fatal("npm ran with a stale revision")
	}
}
