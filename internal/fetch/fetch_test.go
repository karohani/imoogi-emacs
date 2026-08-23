package fetch

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunFetchesArtifactsLicensesBootstrapAndReusesLock(t *testing.T) {
	repo := t.TempDir()
	node := tarGz(t, map[string]string{"node-v24.19.0-darwin-arm64/LICENSE": "node license\n"})
	ts := tarGz(t, map[string]string{
		"package/LICENSE.txt":              "typescript license\n",
		"package/ThirdPartyNoticeText.txt": "typescript notice\n",
		"package/lib/tsserver.js":          "server\n",
	})
	tls := tarGz(t, map[string]string{"package/LICENSE": "tls license\n"})
	requests := map[string]int{}
	server := fixtureServer(t, requests, map[string][]byte{
		"/dist/v24.19.0/node-v24.19.0-darwin-arm64.tar.gz":                   node,
		"/dist/v24.19.0/SHASUMS256.txt":                                      []byte(shaLine(node, "node-v24.19.0-darwin-arm64.tar.gz")),
		"/typescript/-/typescript-6.0.3.tgz":                                 ts,
		"/typescript-language-server/-/typescript-language-server-6.0.0.tgz": tls,
	})
	writeManifest(t, repo)
	goBin := writeFakeGo(t, repo, "cli-bin\n")
	now := fixedTime("2026-08-22T00:00:00Z")

	if err := Run(context.Background(), Options{
		Workdir: repo,
		Client:  server.Client(),
		Now:     now,
		Sources: Sources{NodeDistBase: server.URL + "/dist", NPMRegistry: server.URL, GoProxy: server.URL + "/proxy"},
		Go:      goBin,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertFile(t, repo, "vendor/toolchains/node/v24.19.0/darwin-arm64/node-v24.19.0-darwin-arm64.tar.gz", string(node))
	assertFile(t, repo, "vendor/toolchains/licenses/node-v24.19.0-LICENSE", "node license\n")
	assertFile(t, repo, "vendor/toolchains/licenses/typescript-6.0.3-LICENSE.txt", "typescript license\n")
	assertFile(t, repo, "vendor/toolchains/licenses/typescript-6.0.3-ThirdPartyNoticeText.txt", "typescript notice\n")
	assertFile(t, repo, "vendor/toolchains/licenses/typescript-language-server-LICENSE", "tls license\n")
	assertFile(t, repo, "vendor/toolchains/licenses/gopls-LICENSE", "gopls license\n")
	assertFile(t, repo, "vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain", "cli-bin\n")

	lockData, err := os.ReadFile(filepath.Join(repo, "toolchains.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(lockData, []byte(`"command": "go install -trimpath -ldflags=-buildid= golang.org/x/tools/gopls@v0.23.0"`)) {
		t.Fatalf("lock does not contain reproducible gopls command:\n%s", lockData)
	}
	if !bytes.Contains(lockData, []byte(`"notice": "vendor/toolchains/licenses/typescript-6.0.3-ThirdPartyNoticeText.txt"`)) {
		t.Fatalf("lock does not contain TypeScript NOTICE path:\n%s", lockData)
	}
	provenance := readBootstrapProvenance(t, repo)
	if provenance["cli_version"] != "1.0.0" || provenance["built_at"] != "2026-08-22T00:00:00Z" {
		t.Fatalf("unexpected bootstrap provenance: %#v", provenance)
	}

	server.Close()
	before := string(lockData)
	if err := Run(context.Background(), Options{
		Workdir: repo,
		Client:  server.Client(),
		Now:     fixedTime("2026-08-23T00:00:00Z"),
		Sources: Sources{NodeDistBase: server.URL + "/dist", NPMRegistry: server.URL, GoProxy: server.URL + "/proxy"},
		Go:      goBin,
	}); err != nil {
		t.Fatalf("idempotent Run() error = %v", err)
	}
	afterData, err := os.ReadFile(filepath.Join(repo, "toolchains.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(afterData) != before {
		t.Fatalf("idempotent run changed lock\nbefore:\n%s\nafter:\n%s", before, afterData)
	}
	if !bytes.Contains(afterData, []byte(`"toolchain": "`+runtime.Version()+`"`)) {
		t.Fatalf("lock does not record the actual Go toolchain:\n%s", afterData)
	}

	bootstrapPath := filepath.Join(repo, "vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain")
	if err := os.WriteFile(bootstrapPath, []byte("corrupt\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	err = Run(context.Background(), Options{
		Workdir: repo,
		Client:  server.Client(),
		Now:     fixedTime("2026-08-24T00:00:00Z"),
		Sources: Sources{NodeDistBase: server.URL + "/dist", NPMRegistry: server.URL, GoProxy: server.URL + "/proxy"},
		Go:      goBin,
	})
	if err == nil || !strings.Contains(err.Error(), "does not match its provenance digest") {
		t.Fatalf("Run() after bootstrap corruption error = %v", err)
	}
	unchangedLock, readErr := os.ReadFile(filepath.Join(repo, "toolchains.lock.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(unchangedLock) != before {
		t.Fatal("bootstrap corruption changed the lock")
	}
}

func TestRunRejectsMalformedExistingLockBeforeFetching(t *testing.T) {
	repo := t.TempDir()
	writeManifest(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "toolchains.lock.json"), []byte("{not-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Run(context.Background(), Options{Workdir: repo, Go: writeFakeGo(t, repo, "cli-bin\n")})
	if err == nil || !strings.Contains(err.Error(), "load existing lock") {
		t.Fatalf("Run() error = %v, want malformed existing lock rejection", err)
	}
	lockData, readErr := os.ReadFile(filepath.Join(repo, "toolchains.lock.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(lockData) != "{not-json\n" {
		t.Fatal("malformed existing lock was modified")
	}
}

func TestRunRejectsPathLikeVersionBeforeCreatingLocalState(t *testing.T) {
	repo := t.TempDir()
	writeManifest(t, repo)
	manifestPath := filepath.Join(repo, "toolchains.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte(`"upstream_version": "v0.23.0"`), []byte(`"upstream_version": "../../outside"`), 1)
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	err = Run(context.Background(), Options{Workdir: repo})
	if err == nil || !strings.Contains(err.Error(), "path-safe version token") {
		t.Fatalf("Run() error = %v, want path-safe version rejection", err)
	}
	if _, statErr := os.Lstat(filepath.Join(repo, ".local")); !os.IsNotExist(statErr) {
		t.Fatalf("fetch created .local before rejecting the manifest: %v", statErr)
	}
}

func TestContainedPathRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"../outside", "/tmp/outside", `vendor\outside`, "vendor/../outside"} {
		if _, err := containedPath(root, rel); err == nil {
			t.Fatalf("containedPath(%q) succeeded", rel)
		}
	}
	want := filepath.Join(root, "vendor", "toolchains", "artifact")
	got, err := containedPath(root, "vendor/toolchains/artifact")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("containedPath = %q, want %q", got, want)
	}
}

func TestRunRejectsNodeChecksumWithoutPartialPublish(t *testing.T) {
	repo := t.TempDir()
	node := tarGz(t, map[string]string{"node-v24.19.0-darwin-arm64/LICENSE": "node license\n"})
	requests := map[string]int{}
	server := fixtureServer(t, requests, map[string][]byte{
		"/dist/v24.19.0/node-v24.19.0-darwin-arm64.tar.gz": node,
		"/dist/v24.19.0/SHASUMS256.txt":                    []byte(strings.Repeat("0", 64) + "  node-v24.19.0-darwin-arm64.tar.gz\n"),
	})
	writeManifest(t, repo)

	err := Run(context.Background(), Options{
		Workdir: repo,
		Client:  server.Client(),
		Now:     fixedTime("2026-08-22T00:00:00Z"),
		Sources: Sources{NodeDistBase: server.URL + "/dist", NPMRegistry: server.URL},
		Go:      writeFakeGo(t, repo, "cli-bin\n"),
	})
	if err == nil || !strings.Contains(err.Error(), "node checksum mismatch") {
		t.Fatalf("Run() error = %v, want checksum mismatch", err)
	}
	assertMissing(t, repo, "vendor/toolchains/node/v24.19.0/darwin-arm64/node-v24.19.0-darwin-arm64.tar.gz")
	assertMissing(t, repo, "toolchains.lock.json")
}

func TestRunRejectsTypeScriptWithoutTSServer(t *testing.T) {
	repo := t.TempDir()
	node := tarGz(t, map[string]string{"node-v24.19.0-darwin-arm64/LICENSE": "node license\n"})
	ts := tarGz(t, map[string]string{"package/LICENSE.txt": "typescript license\n"})
	requests := map[string]int{}
	server := fixtureServer(t, requests, map[string][]byte{
		"/dist/v24.19.0/node-v24.19.0-darwin-arm64.tar.gz": node,
		"/dist/v24.19.0/SHASUMS256.txt":                    []byte(shaLine(node, "node-v24.19.0-darwin-arm64.tar.gz")),
		"/typescript/-/typescript-6.0.3.tgz":               ts,
	})
	writeManifest(t, repo)

	err := Run(context.Background(), Options{
		Workdir: repo,
		Client:  server.Client(),
		Now:     fixedTime("2026-08-22T00:00:00Z"),
		Sources: Sources{NodeDistBase: server.URL + "/dist", NPMRegistry: server.URL},
		Go:      writeFakeGo(t, repo, "cli-bin\n"),
	})
	if err == nil || !strings.Contains(err.Error(), "missing package/lib/tsserver.js") {
		t.Fatalf("Run() error = %v, want tsserver rejection", err)
	}
	assertMissing(t, repo, "vendor/toolchains/node/v24.19.0/darwin-arm64/node-v24.19.0-darwin-arm64.tar.gz")
	assertMissing(t, repo, "toolchains.lock.json")
}

func TestRunRejectsSymlinkLicenseWithoutPartialPublish(t *testing.T) {
	repo := t.TempDir()
	node := tarGz(t, map[string]string{"node-v24.19.0-darwin-arm64/LICENSE": "node license\n"})
	ts := tarGzWithSymlink(t,
		map[string]string{
			"package/ThirdPartyNoticeText.txt": "notice\n",
			"package/lib/tsserver.js":          "server\n",
		},
		"package/LICENSE.txt",
		"../outside",
	)
	requests := map[string]int{}
	server := fixtureServer(t, requests, map[string][]byte{
		"/dist/v24.19.0/node-v24.19.0-darwin-arm64.tar.gz": node,
		"/dist/v24.19.0/SHASUMS256.txt":                    []byte(shaLine(node, "node-v24.19.0-darwin-arm64.tar.gz")),
		"/typescript/-/typescript-6.0.3.tgz":               ts,
	})
	writeManifest(t, repo)

	err := Run(context.Background(), Options{
		Workdir: repo,
		Client:  server.Client(),
		Now:     fixedTime("2026-08-22T00:00:00Z"),
		Sources: Sources{NodeDistBase: server.URL + "/dist", NPMRegistry: server.URL},
		Go:      writeFakeGo(t, repo, "cli-bin\n"),
	})
	if err == nil || !strings.Contains(err.Error(), "is not a regular file") {
		t.Fatalf("Run() error = %v, want symlink license rejection", err)
	}
	assertMissing(t, repo, "vendor/toolchains/node/v24.19.0/darwin-arm64/node-v24.19.0-darwin-arm64.tar.gz")
	assertMissing(t, repo, "toolchains.lock.json")
}

func TestRunRejectsDriftBeforeOverwrite(t *testing.T) {
	repo := t.TempDir()
	nodeV1 := tarGz(t, map[string]string{"node-v24.19.0-darwin-arm64/LICENSE": "node license\n"})
	ts := tarGz(t, map[string]string{
		"package/LICENSE.txt":              "typescript license\n",
		"package/ThirdPartyNoticeText.txt": "typescript notice\n",
		"package/lib/tsserver.js":          "server\n",
	})
	tls := tarGz(t, map[string]string{"package/LICENSE": "tls license\n"})
	requests := map[string]int{}
	objects := map[string][]byte{
		"/dist/v24.19.0/node-v24.19.0-darwin-arm64.tar.gz":                   nodeV1,
		"/dist/v24.19.0/SHASUMS256.txt":                                      []byte(shaLine(nodeV1, "node-v24.19.0-darwin-arm64.tar.gz")),
		"/typescript/-/typescript-6.0.3.tgz":                                 ts,
		"/typescript-language-server/-/typescript-language-server-6.0.0.tgz": tls,
	}
	server := fixtureServer(t, requests, objects)
	writeManifest(t, repo)
	goBin := writeFakeGo(t, repo, "cli-bin\n")
	opts := Options{
		Workdir: repo,
		Client:  server.Client(),
		Now:     fixedTime("2026-08-22T00:00:00Z"),
		Sources: Sources{NodeDistBase: server.URL + "/dist", NPMRegistry: server.URL, GoProxy: server.URL + "/proxy"},
		Go:      goBin,
	}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	lockBefore, err := os.ReadFile(filepath.Join(repo, "toolchains.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	nodePath := filepath.Join(repo, "vendor/toolchains/node/v24.19.0/darwin-arm64/node-v24.19.0-darwin-arm64.tar.gz")
	if err := os.Remove(nodePath); err != nil {
		t.Fatal(err)
	}
	nodeV2 := tarGz(t, map[string]string{"node-v24.19.0-darwin-arm64/LICENSE": "changed\n"})
	objects["/dist/v24.19.0/node-v24.19.0-darwin-arm64.tar.gz"] = nodeV2
	objects["/dist/v24.19.0/SHASUMS256.txt"] = []byte(shaLine(nodeV2, "node-v24.19.0-darwin-arm64.tar.gz"))

	err = Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "same identity but different artifact") {
		t.Fatalf("Run() error = %v, want drift rejection", err)
	}
	lockAfter, err := os.ReadFile(filepath.Join(repo, "toolchains.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("failed fetch changed existing lock")
	}
	assertMissing(t, repo, "vendor/toolchains/node/v24.19.0/darwin-arm64/node-v24.19.0-darwin-arm64.tar.gz")
}

func TestRunRejectsUnsupportedComponent(t *testing.T) {
	repo := t.TempDir()
	writeManifest(t, repo)
	data, err := os.ReadFile(filepath.Join(repo, "toolchains.json"))
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte(`"name": "typescript"`), []byte(`"name": "unknown"`), 1)
	data = bytes.Replace(data, []byte(`"kind": "typescript-sdk"`), []byte(`"kind": "unknown-kind"`), 1)
	data = bytes.Replace(data, []byte(`"source": "npm:typescript"`), []byte(`"source": "unknown"`), 1)
	if err := os.WriteFile(filepath.Join(repo, "toolchains.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	node := tarGz(t, map[string]string{"node-v24.19.0-darwin-arm64/LICENSE": "node license\n"})
	server := fixtureServer(t, map[string]int{}, map[string][]byte{
		"/dist/v24.19.0/node-v24.19.0-darwin-arm64.tar.gz": node,
		"/dist/v24.19.0/SHASUMS256.txt":                    []byte(shaLine(node, "node-v24.19.0-darwin-arm64.tar.gz")),
	})
	err = Run(context.Background(), Options{
		Workdir: repo,
		Client:  server.Client(),
		Now:     fixedTime("2026-08-22T00:00:00Z"),
		Sources: Sources{NodeDistBase: server.URL + "/dist", NPMRegistry: server.URL},
		Go:      writeFakeGo(t, repo, "cli-bin\n"),
	})
	if err == nil || !strings.Contains(err.Error(), `unsupported component "unknown"`) {
		t.Fatalf("Run() error = %v, want unsupported component", err)
	}
}

func fixtureServer(t *testing.T, requests map[string]int, objects map[string][]byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		data, ok := objects[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
}

func writeManifest(t *testing.T, repo string) {
	t.Helper()
	manifest := `{
  "schema": "imoogi-toolchains-desired/v1",
  "cli_version": "1.0.0",
  "bundle": "2026.08.22.1",
  "target": {"os": "darwin", "arch": "arm64"},
  "runtime_policy": {"node_version": "v24.19.0"},
  "components": [
    {"name": "gopls", "kind": "go-language-server", "source": "golang.org/x/tools/gopls", "upstream_version": "v0.23.0", "revision": "014f87ff5c01915bc90f4f11a6bb8aea3e0edbd7"},
    {"name": "node", "kind": "node-runtime", "source": "nodejs.org/dist", "upstream_version": "v24.19.0"},
    {"name": "typescript", "kind": "typescript-sdk", "source": "npm:typescript", "upstream_version": "6.0.3"},
    {"name": "typescript-language-server", "kind": "typescript-language-server", "source": "npm:typescript-language-server", "upstream_version": "6.0.0", "revision": "1c0f224eb44c626d96dae07aaf5d78654de0e1f2", "runtime": {"node_engine": ">=22.22.2"}}
  ]
}
`
	if err := os.WriteFile(filepath.Join(repo, "toolchains.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "cmd/imoogi-toolchain"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFakeGo(t *testing.T, repo, cliContent string) string {
	t.Helper()
	path := filepath.Join(repo, "fake-go")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
case "$1" in
  install)
    mkdir -p "$GOBIN"
    printf 'gopls-bin\n' > "$GOBIN/gopls"
    chmod +x "$GOBIN/gopls"
    module="$GOMODCACHE/golang.org/x/tools/gopls@v0.23.0"
    mkdir -p "$module"
    printf 'gopls license\n' > "$module/LICENSE"
    printf 'gopls patents\n' > "$module/PATENTS"
    ;;
  build)
    out=""
    while [ "$#" -gt 0 ]; do
      if [ "$1" = "-o" ]; then
        shift
        out="$1"
      fi
      shift || true
    done
    mkdir -p "$(dirname "$out")"
    printf %s > "$out"
    chmod +x "$out"
    ;;
  *)
    echo "unexpected fake go command: $*" >&2
    exit 2
    ;;
esac
`, shellQuote(cliContent))
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func fixedTime(value string) func() time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return func() time.Time { return parsed }
}

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		data := []byte(body)
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
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

func tarGzWithSymlink(t *testing.T, files map[string]string, symlinkName, symlinkTarget string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		data := []byte(body)
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.WriteHeader(&tar.Header{Name: symlinkName, Mode: 0o777, Typeflag: tar.TypeSymlink, Linkname: symlinkTarget}); err != nil {
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

func shaLine(data []byte, name string) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]) + "  " + name + "\n"
}

func assertFile(t *testing.T, repo, rel, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", rel, got, want)
	}
}

func assertMissing(t *testing.T, repo, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(rel))); !os.IsNotExist(err) {
		t.Fatalf("%s exists or stat failed with non-ENOENT: %v", rel, err)
	}
}

func readBootstrapProvenance(t *testing.T, repo string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, "vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain.provenance.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
