package artifact

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type zipEntry struct {
	name string
	body []byte
	mode os.FileMode
}

func writeZip(t *testing.T, root string, entries []zipEntry) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetMode(entry.mode)
		w, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "archive.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractZipValidArchiveUsesExplicitModes(t *testing.T) {
	root := t.TempDir()
	archive := writeZip(t, root, []zipEntry{
		{name: "server/", mode: os.ModeDir | 0o777},
		{name: "server/bin/kotlin-language-server", body: []byte("#!/bin/sh\n"), mode: 0o777},
		{name: "server/lib/server.jar", body: []byte("jar"), mode: 0o600},
	})
	dest := filepath.Join(root, "out")

	result, err := ExtractZip(archive, dest, digestFile(t, archive), Limits{})
	if err != nil {
		t.Fatalf("ExtractZip failed: %v", err)
	}
	if result.Entries != 3 || result.Bytes != int64(len("#!/bin/sh\n")+len("jar")) {
		t.Fatalf("result = %+v", result)
	}
	assertFile(t, filepath.Join(dest, "server", "bin", "kotlin-language-server"), "#!/bin/sh\n")
	assertFile(t, filepath.Join(dest, "server", "lib", "server.jar"), "jar")
	assertMode(t, filepath.Join(dest, "server", "bin"), 0o755)
	assertMode(t, filepath.Join(dest, "server", "bin", "kotlin-language-server"), 0o644)
}

func TestExtractZipVerifiesBeforeDestinationMutation(t *testing.T) {
	root := t.TempDir()
	archive := writeZip(t, root, []zipEntry{{name: "clojure-lsp", body: []byte("bin"), mode: 0o755}})
	dest := filepath.Join(root, "out")

	_, err := ExtractZip(archive, dest, digestFor([]byte("other")), Limits{})
	if err == nil {
		t.Fatal("ExtractZip accepted a digest mismatch")
	}
	if _, statErr := os.Lstat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("destination was created after digest mismatch: %v", statErr)
	}
}

func TestExtractZipRejectsHostileEntries(t *testing.T) {
	tests := []struct {
		name    string
		entries []zipEntry
		want    string
	}{
		{name: "parent traversal", entries: []zipEntry{{name: "../escape", body: []byte("x"), mode: 0o644}}, want: "path traversal"},
		{name: "absolute path", entries: []zipEntry{{name: "/etc/passwd", body: []byte("x"), mode: 0o644}}, want: "absolute path"},
		{name: "backslash", entries: []zipEntry{{name: `dir\file`, body: []byte("x"), mode: 0o644}}, want: "backslash"},
		{name: "symlink", entries: []zipEntry{{name: "link", body: []byte("target"), mode: os.ModeSymlink | 0o777}}, want: "symlinks are not supported"},
		{name: "duplicate", entries: []zipEntry{
			{name: "a", body: []byte("1"), mode: 0o644},
			{name: "a", body: []byte("2"), mode: 0o644},
		}, want: "duplicated"},
		{name: "file parent conflict", entries: []zipEntry{
			{name: "a", body: []byte("1"), mode: 0o644},
			{name: "a/b", body: []byte("2"), mode: 0o644},
		}, want: "conflicts with file parent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			archive := writeZip(t, root, tt.entries)
			dest := filepath.Join(root, "out")
			_, err := ExtractZip(archive, dest, digestFile(t, archive), Limits{})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ExtractZip error = %v, want %q", err, tt.want)
			}
			if _, statErr := os.Lstat(dest); !os.IsNotExist(statErr) {
				t.Fatalf("destination exists after rejected archive: %v", statErr)
			}
			if _, statErr := os.Lstat(filepath.Join(filepath.Dir(root), "escape")); !os.IsNotExist(statErr) {
				t.Fatalf("traversal entry escaped extraction root: %v", statErr)
			}
		})
	}
}

func TestExtractZipRejectsEntryAndOutputLimits(t *testing.T) {
	root := t.TempDir()
	archive := writeZip(t, root, []zipEntry{
		{name: "a", body: []byte("12345"), mode: 0o644},
		{name: "b", body: []byte("12345"), mode: 0o644},
	})
	digest := digestFile(t, archive)

	if _, err := ExtractZip(archive, filepath.Join(root, "entries"), digest, Limits{MaxEntries: 1}); err == nil || !strings.Contains(err.Error(), "too many entries") {
		t.Fatalf("entry limit error = %v", err)
	}
	if _, err := ExtractZip(archive, filepath.Join(root, "file"), digest, Limits{MaxFileBytes: 4}); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("file limit error = %v", err)
	}
	if _, err := ExtractZip(archive, filepath.Join(root, "output"), digest, Limits{MaxOutputBytes: 9}); err == nil || !strings.Contains(err.Error(), "output is too large") {
		t.Fatalf("output limit error = %v", err)
	}
}

func TestExtractZipRejectsNonEmptyDestination(t *testing.T) {
	root := t.TempDir()
	archive := writeZip(t, root, []zipEntry{{name: "a", body: []byte("1"), mode: 0o644}})
	dest := filepath.Join(root, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "keep"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ExtractZip(archive, dest, digestFile(t, archive), Limits{}); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("ExtractZip error = %v", err)
	}
	assertFile(t, filepath.Join(dest, "keep"), "x")
}
