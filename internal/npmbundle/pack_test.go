package npmbundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/karohani/imoogi-emacs/internal/artifact"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func packBytes(t *testing.T, root string) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := Pack(root, &buf); err != nil {
		t.Fatalf("Pack() error = %v", err)
	}
	return buf.Bytes()
}

func sampleTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"node_modules/@scope/pkg/package.json":  `{"name":"@scope/pkg","version":"1.0.0","bin":{"pkg":"dist/index.js"}}`,
		"node_modules/@scope/pkg/dist/index.js": "#!/usr/bin/env node\n",
		"node_modules/dep/index.js":             "module.exports = 1\n",
		"node_modules/.package-lock.json":       `{"hidden":"lockfile"}`,
		"node_modules/.bin/pkg":                 "shim\n",
		"node_modules/dep/node_modules/.bin/x":  "shim\n",
		"package.json":                          `{"name":"wrapper"}`,
	})
	if err := os.Chmod(filepath.Join(root, "node_modules/@scope/pkg/dist/index.js"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPackIsByteIdenticalAcrossMetadataChanges(t *testing.T) {
	root := sampleTree(t)
	first := packBytes(t, root)

	old := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	_ = filepath.Walk(root, func(path string, _ os.FileInfo, _ error) error {
		return os.Chtimes(path, old, old)
	})
	if err := os.Chmod(filepath.Join(root, "node_modules/dep/index.js"), 0o600); err != nil {
		t.Fatal(err)
	}
	second := packBytes(t, root)
	if !bytes.Equal(first, second) {
		t.Fatal("Pack output changed with mtimes and non-executable permission bits")
	}

	writeTree(t, root, map[string]string{"node_modules/dep/index.js": "module.exports = 2\n"})
	if bytes.Equal(first, packBytes(t, root)) {
		t.Fatal("Pack output did not change with file content")
	}
}

func TestPackWritesNormalizedSortedHeaders(t *testing.T) {
	data := packBytes(t, sampleTree(t))
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if gz.Name != "" || !gz.ModTime.IsZero() || gz.Comment != "" {
		t.Fatalf("gzip header carries name/mtime/comment: %+v", gz.Header)
	}
	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
		if hdr.Uid != 0 || hdr.Gid != 0 || hdr.Uname != "" || hdr.Gname != "" {
			t.Fatalf("%s carries ownership: %+v", hdr.Name, hdr)
		}
		if !hdr.ModTime.Equal(time.Unix(0, 0)) {
			t.Fatalf("%s mtime = %v", hdr.Name, hdr.ModTime)
		}
		want := int64(0o644)
		switch {
		case hdr.Typeflag == tar.TypeDir:
			want = 0o755
		case strings.HasSuffix(hdr.Name, "dist/index.js"):
			want = 0o755
		}
		if hdr.Mode != want {
			t.Fatalf("%s mode = %o, want %o", hdr.Name, hdr.Mode, want)
		}
	}
	want := []string{
		"node_modules/",
		"node_modules/@scope/",
		"node_modules/@scope/pkg/",
		"node_modules/@scope/pkg/dist/",
		"node_modules/@scope/pkg/dist/index.js",
		"node_modules/@scope/pkg/package.json",
		"node_modules/dep/",
		"node_modules/dep/index.js",
		"node_modules/dep/node_modules/",
	}
	if strings.Join(names, "\n") != strings.Join(want, "\n") {
		t.Fatalf("entries:\n%s\nwant:\n%s", strings.Join(names, "\n"), strings.Join(want, "\n"))
	}
}

func TestPackRejectsSymlinks(t *testing.T) {
	root := sampleTree(t)
	if err := os.Symlink("index.js", filepath.Join(root, "node_modules/dep/alias.js")); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := Pack(root, &buf); err == nil || !strings.Contains(err.Error(), "alias.js") {
		t.Fatalf("Pack() error = %v, want symlink rejection naming the entry", err)
	}
}

func TestPackOutputExtractsWithTheSetupExtractor(t *testing.T) {
	data := packBytes(t, sampleTree(t))
	archive := filepath.Join(t.TempDir(), "bundle.tgz")
	if err := os.WriteFile(archive, data, 0o644); err != nil {
		t.Fatal(err)
	}
	digest, err := DigestFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "out")
	if _, err := artifact.ExtractTarGzip(archive, dest, digest, artifact.Limits{}); err != nil {
		t.Fatalf("ExtractTarGzip() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "node_modules/@scope/pkg/dist/index.js")); err != nil {
		t.Fatal(err)
	}
}
