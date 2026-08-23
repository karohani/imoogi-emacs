package golang

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/config"
)

func TestGoplsProviderVerifiesCopiesAndProbes(t *testing.T) {
	repo := t.TempDir()
	artifactPath := filepath.Join("vendor", "toolchains", "go", "gopls", "v0.20.0", "darwin-arm64", "gopls")
	body := []byte("#!/bin/sh\necho gopls fake\n")
	writeFile(t, filepath.Join(repo, artifactPath), body, 0o600)
	component := goplsComponent(artifactPath, body)
	staging := filepath.Join(repo, ".local", ".staging")

	probes, err := New(component).Materialize(repo, staging)
	if err != nil {
		t.Fatalf("Materialize failed: %v", err)
	}
	if got, want := len(probes), 1; got != want {
		t.Fatalf("probes = %d, want %d", got, want)
	}
	wantProbe := filepath.Join(staging, "bin", "gopls")
	if probes[0].Command != wantProbe || strings.Join(probes[0].Args, " ") != "version" {
		t.Fatalf("probe = %+v", probes[0])
	}
	assertFile(t, wantProbe, string(body))
	assertMode(t, wantProbe, 0o755)
	if _, err := os.Lstat(filepath.Join(repo, ".local", "bin")); !os.IsNotExist(err) {
		t.Fatalf("provider created activation path; err=%v", err)
	}
}

func TestGoplsProviderIntegrityFailureDoesNotMaterialize(t *testing.T) {
	repo := t.TempDir()
	artifactPath := filepath.Join("vendor", "toolchains", "go", "gopls", "v0.20.0", "darwin-arm64", "gopls")
	body := []byte("bad")
	writeFile(t, filepath.Join(repo, artifactPath), body, 0o600)
	component := goplsComponent(artifactPath, []byte("expected"))
	staging := filepath.Join(repo, ".local", ".staging")

	if _, err := New(component).Materialize(repo, staging); err == nil {
		t.Fatal("Materialize succeeded with mismatched artifact")
	}
	if _, err := os.Lstat(filepath.Join(staging, "bin", "gopls")); !os.IsNotExist(err) {
		t.Fatalf("gopls was materialized after integrity failure; err=%v", err)
	}
}

func TestGoplsProviderRejectsWrongComponentAuthority(t *testing.T) {
	component := goplsComponent("vendor/toolchains/go/gopls/gopls", []byte("x"))
	component.Name = "node"
	component.Kind = "node-runtime"

	if _, err := New(component).Materialize(t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("Materialize accepted non-gopls component")
	}
}

func goplsComponent(path string, data []byte) config.LockComponent {
	sum := sha256.Sum256(data)
	return config.LockComponent{
		Name:            "gopls",
		Kind:            "go-language-server",
		Source:          "golang.org/x/tools/gopls",
		UpstreamVersion: "v0.20.0",
		Artifact: config.Artifact{
			Path:   filepath.ToSlash(path),
			Size:   int64(len(data)),
			SHA256: hex.EncodeToString(sum[:]),
		},
		Probe: config.Probe{Command: "bin/gopls", Args: []string{"version"}},
	}
}

func writeFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, data, want)
	}
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
