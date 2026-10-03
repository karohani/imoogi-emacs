package notecache

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenStoreCancelledContextDoesNotDeleteExistingDatabase(t *testing.T) {
	cacheDir := t.TempDir()
	scopeKey := "cancelled"
	store, err := openStore(context.Background(), cacheDir, scopeKey)
	if err != nil {
		t.Fatalf("initial open failed: %v", err)
	}
	if err := store.close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cacheDir, scopeKey+".sqlite")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = openStore(ctx, cacheDir, scopeKey)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("openStore error = %v, want context.Canceled", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("database was removed after cancellation: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("database changed after cancellation")
	}
}

func TestOpenStoreRecoversNotADBDatabase(t *testing.T) {
	cacheDir := t.TempDir()
	scopeKey := "notadb"
	path := filepath.Join(cacheDir, scopeKey+".sqlite")
	if err := os.WriteFile(path, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := openStore(context.Background(), cacheDir, scopeKey)
	if err != nil {
		t.Fatalf("openStore failed to recover notadb file: %v", err)
	}
	if err := store.close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "not a sqlite database") {
		t.Fatalf("notadb sentinel remained after recovery")
	}
}

func TestOpenStoreRejectsDatabaseSymlink(t *testing.T) {
	cacheDir := t.TempDir()
	scopeKey := "symlink"
	target := filepath.Join(cacheDir, "target.sqlite")
	if err := os.WriteFile(target, []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(cacheDir, scopeKey+".sqlite")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	_, err := openStore(context.Background(), cacheDir, scopeKey)
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("openStore error = %v, want symlink rejection", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "sentinel" {
		t.Fatalf("symlink target was modified: %q", data)
	}
}
