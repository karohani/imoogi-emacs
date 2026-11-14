package node

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/config"
)

func nodeComponent(t *testing.T, repo string) config.LockComponent {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{
		"node-v24.19.0-darwin-arm64/bin/node":  "#!/bin/sh\necho node\n",
		"node-v24.19.0-darwin-arm64/README.md": "readme\n",
	} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	rel := "vendor/toolchains/node/v24.19.0/darwin-arm64/node.tar.gz"
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return config.LockComponent{Name: "node", Kind: "node-runtime", UpstreamVersion: "v24.19.0",
		Artifact: config.Artifact{Path: rel, Size: int64(buf.Len()), SHA256: hex.EncodeToString(sum[:])}}
}

func TestEnsureExtractsOnceAndIsSharedByLaterCallers(t *testing.T) {
	repo := t.TempDir()
	component := nodeComponent(t, repo)
	staging := t.TempDir()

	first, err := Ensure(repo, staging, component)
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if filepath.Base(first) != "node-v24.19.0-darwin-arm64" {
		t.Fatalf("Ensure() = %s", first)
	}
	info, err := os.Stat(filepath.Join(first, "bin", "node"))
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("bin/node mode = %v, %v", info, err)
	}
	marker := filepath.Join(first, "marker")
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := Ensure(repo, staging, component)
	if err != nil || second != first {
		t.Fatalf("second Ensure() = %s, %v", second, err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("second Ensure() re-extracted over the first")
	}
}

func TestEnsureRejectsDigestMismatchAndForeignStagingContent(t *testing.T) {
	repo := t.TempDir()
	component := nodeComponent(t, repo)
	bad := component
	bad.Artifact.SHA256 = strings.Repeat("0", 64)
	if _, err := Ensure(repo, t.TempDir(), bad); err == nil {
		t.Fatal("Ensure() accepted a digest mismatch")
	}

	staging := t.TempDir()
	if err := os.MkdirAll(filepath.Join(staging, "lib", "node", "node-v99.0.0-darwin-arm64", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Ensure(repo, staging, component); err == nil || !strings.Contains(err.Error(), "v99") {
		t.Fatalf("Ensure() = %v, want rejection of a different node already staged", err)
	}
}
