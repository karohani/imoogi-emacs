package clojure

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/config"
)

func TestSupportsComponentByNameOrKind(t *testing.T) {
	if !SupportsComponent(config.LockComponent{Name: "clojure-lsp"}) || !SupportsComponent(config.LockComponent{Name: "x", Kind: "clojure-language-server"}) {
		t.Fatal("clojure-lsp not supported")
	}
	if SupportsComponent(config.LockComponent{Name: "jdtls", Kind: "java-language-server"}) {
		t.Fatal("jdtls claimed by clojure provider")
	}
}

func TestClojureLspProviderInstallsNativeBinary(t *testing.T) {
	repo := t.TempDir()
	binary := "#!/bin/sh\necho clojure-lsp fake\n"
	component := component(t, repo, zipBytes(t, map[string]string{"clojure-lsp": binary}))
	staging := filepath.Join(repo, ".local", ".staging")

	probes, err := New(component).Materialize(repo, staging)
	if err != nil {
		t.Fatalf("Materialize failed: %v", err)
	}
	installed := filepath.Join(staging, "bin", "clojure-lsp")
	if len(probes) != 1 || probes[0].Command != installed || strings.Join(probes[0].Args, " ") != "--version" {
		t.Fatalf("probes = %+v", probes)
	}
	data, err := os.ReadFile(installed)
	if err != nil || string(data) != binary {
		t.Fatalf("installed binary = %q, %v", data, err)
	}
	info, err := os.Stat(installed)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("installed mode = %v, %v", info, err)
	}
	if entries, _ := os.ReadDir(filepath.Join(staging, "lib")); len(entries) != 0 {
		t.Fatalf("extraction temp left in lib/: %v", entries)
	}
}

func TestClojureLspProviderRejectsArchiveWithoutBinary(t *testing.T) {
	repo := t.TempDir()
	component := component(t, repo, zipBytes(t, map[string]string{"README": "x"}))
	if _, err := New(component).Materialize(repo, filepath.Join(repo, ".staging")); err == nil || !strings.Contains(err.Error(), "clojure-lsp") {
		t.Fatalf("Materialize error = %v", err)
	}
}

func TestClojureLspProviderRejectsForeignComponent(t *testing.T) {
	if _, err := New(config.LockComponent{Name: "gopls", Kind: "go-language-server"}).Materialize(t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("Materialize accepted gopls")
	}
}

func component(t *testing.T, repo string, body []byte) config.LockComponent {
	t.Helper()
	rel := "vendor/toolchains/clojure/clojure-lsp/2026.07.06-14.34.19/darwin-arm64/clojure-lsp-native-macos-aarch64.zip"
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	return config.LockComponent{
		Name:            "clojure-lsp",
		Kind:            "clojure-language-server",
		UpstreamVersion: "2026.07.06-14.34.19",
		Artifact:        config.Artifact{Path: rel, Size: int64(len(body)), SHA256: hex.EncodeToString(sum[:])},
		Probe:           config.Probe{Command: "bin/clojure-lsp", Args: []string{"--version"}},
	}
}

func zipBytes(t *testing.T, files map[string]string) []byte {
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
