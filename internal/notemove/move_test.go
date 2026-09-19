package notemove

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoveCopiesVerifiesAndRemovesSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "local", "26.01-os")
	destination := filepath.Join(root, "external", "26.01-os")
	if err := os.MkdirAll(filepath.Join(source, "concepts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("#+TITLE: Context switch\n")
	if err := os.WriteFile(filepath.Join(source, "concepts", "C01.org"), content, 0o640); err != nil {
		t.Fatal(err)
	}

	response := Move(Request{Operation: "move", Source: source, Destination: destination})
	if !response.OK {
		t.Fatalf("Move failed: %#v", response)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists or stat failed unexpectedly: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "concepts", "C01.org"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch: %q", got)
	}
	if response.Files != 1 || response.Bytes != int64(len(content)) {
		t.Fatalf("unexpected accounting: %#v", response)
	}
}

func TestMoveNeverOverwritesDestination(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "local", "notes")
	destination := filepath.Join(root, "external", "notes")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "keep"), []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	response := Move(Request{Operation: "move", Source: source, Destination: destination})
	if response.OK || response.Code != "invalid_path" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source was changed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "keep"))
	if err != nil || string(got) != "existing" {
		t.Fatalf("destination was changed: %q, %v", got, err)
	}
}

func TestMoveRejectsNestedDestination(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "notes")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	response := Move(Request{
		Operation:   "move",
		Source:      source,
		Destination: filepath.Join(source, "nested", "copy"),
	})
	if response.OK || response.Code != "invalid_path" {
		t.Fatalf("unexpected response: %#v", response)
	}
}
