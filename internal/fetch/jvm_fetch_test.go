package fetch

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/config"
)

const (
	jdtlsCommit   = "08eafe6ff60c7159ef88571d47b6a9ef82fef94e"
	kotlinCommit  = "f532bcc9c6578f5a35009a8b5f281f93d4d131ba"
	clojureCommit = "de66293956c057b6ef02601594f196d048684683"
)

func TestRunFetchesJVMAndClojureServersWithLicenses(t *testing.T) {
	repo := t.TempDir()
	jdtls := tarGz(t, map[string]string{"plugins/org.eclipse.equinox.launcher_1.jar": "jar"})
	kotlin := fetchZip(t, map[string]string{"server/bin/kotlin-language-server": "#!/bin/sh\n", "server/licenseReport.html": "report\n"})
	clojure := fetchZip(t, map[string]string{"clojure-lsp": "native\n"})
	jdtlsPath := "/jdtls/milestones/1.61.0/jdt-language-server-1.61.0-202609031315.tar.gz"
	requests := map[string]int{}
	server := fixtureServer(t, requests, map[string][]byte{
		jdtlsPath:             jdtls,
		jdtlsPath + ".sha256": []byte(shaLine(jdtls, "")),
		"/fwcd/kotlin-language-server/releases/download/1.3.13/server.zip":                                    kotlin,
		"/clojure-lsp/clojure-lsp/releases/download/2026.07.06-14.34.19/clojure-lsp-native-macos-aarch64.zip": clojure,
		"/raw/eclipse-jdtls/eclipse.jdt.ls/" + jdtlsCommit + "/LICENSE":                                       []byte("EPL-2.0\n"),
		"/raw/fwcd/kotlin-language-server/" + kotlinCommit + "/LICENSE.txt":                                   []byte("MIT kotlin\n"),
		"/raw/clojure-lsp/clojure-lsp/" + clojureCommit + "/LICENSE":                                          []byte("MIT clojure\n"),
	})
	defer server.Close()
	writeJVMManifest(t, repo)
	goBin := writeFakeGo(t, repo, "cli-bin\n")
	opts := Options{
		Workdir: repo,
		Client:  server.Client(),
		Now:     fixedTime("2026-10-12T00:00:00Z"),
		Sources: Sources{EclipseDownloads: server.URL, GitHub: server.URL, GitHubRaw: server.URL + "/raw"},
		Go:      goBin,
	}

	if err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertFile(t, repo, "vendor/toolchains/java/jdtls/1.61.0-202609031315/darwin-arm64/jdt-language-server.tar.gz", string(jdtls))
	assertFile(t, repo, "vendor/toolchains/kotlin/kotlin-language-server/1.3.13/darwin-arm64/server.zip", string(kotlin))
	assertFile(t, repo, "vendor/toolchains/clojure/clojure-lsp/2026.07.06-14.34.19/darwin-arm64/clojure-lsp-native-macos-aarch64.zip", string(clojure))
	assertFile(t, repo, "vendor/toolchains/licenses/jdtls-LICENSE", "EPL-2.0\n")
	assertFile(t, repo, "vendor/toolchains/licenses/kotlin-language-server-LICENSE", "MIT kotlin\n")
	assertFile(t, repo, "vendor/toolchains/licenses/kotlin-language-server-1.3.13-licenseReport.html", "report\n")
	assertFile(t, repo, "vendor/toolchains/licenses/clojure-lsp-LICENSE", "MIT clojure\n")

	lock, err := config.LoadLock(filepath.Join(repo, "toolchains.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]config.LockComponent{}
	for _, component := range lock.Components {
		byName[component.Name] = component
	}
	checks := map[string]struct {
		url, probe string
		args       []string
	}{
		"jdtls":                  {server.URL + jdtlsPath, "bin/jdtls", []string{"--imoogi-probe"}},
		"kotlin-language-server": {server.URL + "/fwcd/kotlin-language-server/releases/download/1.3.13/server.zip", "bin/kotlin-language-server", []string{"--imoogi-probe"}},
		"clojure-lsp":            {server.URL + "/clojure-lsp/clojure-lsp/releases/download/2026.07.06-14.34.19/clojure-lsp-native-macos-aarch64.zip", "bin/clojure-lsp", []string{"--version"}},
	}
	for name, want := range checks {
		got, ok := byName[name]
		if !ok {
			t.Fatalf("lock is missing %s", name)
		}
		if got.Artifact.SourceURL != want.url || got.Probe.Command != want.probe || strings.Join(got.Probe.Args, " ") != strings.Join(want.args, " ") {
			t.Fatalf("%s lock = %+v", name, got)
		}
	}
	if byName["kotlin-language-server"].License.Notice != "vendor/toolchains/licenses/kotlin-language-server-1.3.13-licenseReport.html" {
		t.Fatalf("kotlin notice = %q", byName["kotlin-language-server"].License.Notice)
	}

	// A re-run with the same pins reuses everything without the network.
	before, _ := os.ReadFile(filepath.Join(repo, "toolchains.lock.json"))
	server.Close()
	opts.Now = fixedTime("2026-10-13T00:00:00Z")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatalf("offline re-run error = %v", err)
	}
	after, _ := os.ReadFile(filepath.Join(repo, "toolchains.lock.json"))
	if !bytes.Equal(before, after) {
		t.Fatalf("re-run changed the lock\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestRunRejectsJdtlsChecksumMismatch(t *testing.T) {
	repo := t.TempDir()
	jdtls := tarGz(t, map[string]string{"plugins/x.jar": "jar"})
	jdtlsPath := "/jdtls/milestones/1.61.0/jdt-language-server-1.61.0-202609031315.tar.gz"
	server := fixtureServer(t, map[string]int{}, map[string][]byte{
		jdtlsPath:             jdtls,
		jdtlsPath + ".sha256": []byte(strings.Repeat("0", 64) + "\n"),
	})
	defer server.Close()
	writeDesired(t, repo, `{"name": "jdtls", "kind": "java-language-server", "source": "download.eclipse.org/jdtls", "upstream_version": "1.61.0-202609031315", "revision": "`+jdtlsCommit+`"}`)

	err := Run(context.Background(), Options{
		Workdir: repo, Client: server.Client(), Now: fixedTime("2026-10-12T00:00:00Z"),
		Sources: Sources{EclipseDownloads: server.URL, GitHub: server.URL, GitHubRaw: server.URL},
		Go:      writeFakeGo(t, repo, "cli\n"),
	})
	if err == nil || !strings.Contains(err.Error(), "jdtls checksum mismatch") {
		t.Fatalf("Run() error = %v", err)
	}
	assertMissing(t, repo, "vendor/toolchains/java/jdtls/1.61.0-202609031315/darwin-arm64/jdt-language-server.tar.gz")
	assertMissing(t, repo, "toolchains.lock.json")
}

func TestRunRejectsJdtlsVersionWithoutBuildStamp(t *testing.T) {
	repo := t.TempDir()
	writeDesired(t, repo, `{"name": "jdtls", "kind": "java-language-server", "source": "download.eclipse.org/jdtls", "upstream_version": "1.61.0", "revision": "`+jdtlsCommit+`"}`)
	err := Run(context.Background(), Options{Workdir: repo, Now: fixedTime("2026-10-12T00:00:00Z"), Go: writeFakeGo(t, repo, "cli\n")})
	if err == nil || !strings.Contains(err.Error(), "build stamp") {
		t.Fatalf("Run() error = %v", err)
	}
}

func writeJVMManifest(t *testing.T, repo string) {
	t.Helper()
	writeDesired(t, repo, strings.Join([]string{
		`{"name": "jdtls", "kind": "java-language-server", "source": "download.eclipse.org/jdtls", "upstream_version": "1.61.0-202609031315", "revision": "` + jdtlsCommit + `"}`,
		`{"name": "kotlin-language-server", "kind": "kotlin-language-server", "source": "github.com/fwcd/kotlin-language-server", "upstream_version": "1.3.13", "revision": "` + kotlinCommit + `"}`,
		`{"name": "clojure-lsp", "kind": "clojure-language-server", "source": "github.com/clojure-lsp/clojure-lsp", "upstream_version": "2026.07.06-14.34.19", "revision": "` + clojureCommit + `"}`,
	}, ",\n    "))
}

func writeDesired(t *testing.T, repo, components string) {
	t.Helper()
	manifest := `{
  "schema": "imoogi-toolchains-desired/v1",
  "cli_version": "1.0.0",
  "bundle": "2026.10.12.1",
  "target": {"os": "darwin", "arch": "arm64"},
  "components": [
    ` + components + `
  ]
}
`
	var probe map[string]any
	if err := json.Unmarshal([]byte(manifest), &probe); err != nil {
		t.Fatalf("fixture manifest is not JSON: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "toolchains.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "cmd/imoogi-toolchain"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func fetchZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
