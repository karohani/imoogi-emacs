package artifact

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestVerifyFileStreamsExactSizeAndSHA256(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "artifact.bin")
	data := []byte("verified bytes")
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := digestFor(data)

	if err := VerifyFile(file, digest); err != nil {
		t.Fatalf("VerifyFile failed: %v", err)
	}
	if err := VerifyFile(file, Digest{Size: digest.Size + 1, SHA256: digest.SHA256}); err == nil {
		t.Fatal("VerifyFile accepted size mismatch")
	}
	if err := VerifyFile(file, Digest{Size: digest.Size, SHA256: strings.Repeat("0", 64)}); err == nil {
		t.Fatal("VerifyFile accepted hash mismatch")
	}
	if err := VerifyFile(file, Digest{Size: digest.Size, SHA256: strings.ToUpper(digest.SHA256)}); err == nil {
		t.Fatal("VerifyFile accepted non-lowercase SHA-256")
	}
}

func TestExtractTarGzipValidArchiveUsesExplicitModesAndIgnoresMetadata(t *testing.T) {
	root := t.TempDir()
	archive := writeArchive(t, root, []tarEntry{
		dirEntry("bin", 0o777),
		fileEntry("bin/gopls", []byte("#!/bin/sh\nexit 0\n"), 0o777),
		fileEntry("README.txt", []byte("hello\n"), 0o600),
	})
	dest := filepath.Join(root, "out")

	result, err := ExtractTarGzip(archive, dest, digestFile(t, archive), Limits{})
	if err != nil {
		t.Fatalf("ExtractTarGzip failed: %v", err)
	}
	if result.Entries != 3 || result.Bytes != int64(len("#!/bin/sh\nexit 0\n")+len("hello\n")) {
		t.Fatalf("result = %+v", result)
	}
	assertFile(t, filepath.Join(dest, "bin", "gopls"), "#!/bin/sh\nexit 0\n")
	assertFile(t, filepath.Join(dest, "README.txt"), "hello\n")
	assertMode(t, filepath.Join(dest, "bin"), 0o755)
	assertMode(t, filepath.Join(dest, "bin", "gopls"), 0o644)
	assertMode(t, filepath.Join(dest, "README.txt"), 0o644)
}

func TestExtractTarGzipAllowsSymlinkAncestorOutsideStagingRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink ancestor regression is POSIX-specific")
	}
	root := t.TempDir()
	realParent := filepath.Join(root, "real-parent")
	if err := os.Mkdir(realParent, 0o755); err != nil {
		t.Fatal(err)
	}
	linkedParent := filepath.Join(root, "linked-parent")
	if err := os.Symlink(realParent, linkedParent); err != nil {
		t.Fatal(err)
	}
	archive := writeArchive(t, root, []tarEntry{fileEntry("bin/tool", []byte("ok"), 0o644)})
	dest := filepath.Join(linkedParent, "out")

	if _, err := ExtractTarGzip(archive, dest, digestFile(t, archive), Limits{}); err != nil {
		t.Fatalf("ExtractTarGzip failed through symlink ancestor: %v", err)
	}
	assertFile(t, filepath.Join(realParent, "out", "bin", "tool"), "ok")
}

func TestExtractTarGzipDefaultRejectsNodeShapedSymlinks(t *testing.T) {
	root := t.TempDir()
	archive := writeArchive(t, root, []tarEntry{
		dirEntry("node-v24.19.0-darwin-arm64/bin", 0o755),
		dirEntry("node-v24.19.0-darwin-arm64/lib/node_modules/npm/bin", 0o755),
		fileEntry("node-v24.19.0-darwin-arm64/bin/node", []byte("node"), 0o755),
		fileEntry("node-v24.19.0-darwin-arm64/lib/node_modules/npm/bin/npm-cli.js", []byte("npm"), 0o644),
		linkEntry("node-v24.19.0-darwin-arm64/bin/npm", tar.TypeSymlink, "../lib/node_modules/npm/bin/npm-cli.js"),
	})
	dest := filepath.Join(root, "out")

	if err := extractError(archive, dest, digestFile(t, archive), Limits{}); err == nil {
		t.Fatal("ExtractTarGzip accepted symlink without opt-in")
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		t.Fatalf("destination exists after default symlink rejection; err=%v", err)
	}
}

func TestExtractTarGzipWithOptionsAllowsNodeShapedRelativeSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixture is POSIX-specific")
	}
	root := t.TempDir()
	archive := writeArchive(t, root, []tarEntry{
		dirEntry("node-v24.19.0-darwin-arm64/bin", 0o755),
		dirEntry("node-v24.19.0-darwin-arm64/lib/node_modules/npm/bin", 0o755),
		dirEntry("node-v24.19.0-darwin-arm64/lib/node_modules/corepack/dist", 0o755),
		fileEntry("node-v24.19.0-darwin-arm64/bin/node", []byte("node"), 0o755),
		fileEntry("node-v24.19.0-darwin-arm64/lib/node_modules/npm/bin/npm-cli.js", []byte("npm"), 0o644),
		fileEntry("node-v24.19.0-darwin-arm64/lib/node_modules/corepack/dist/corepack.js", []byte("corepack"), 0o644),
		linkEntry("node-v24.19.0-darwin-arm64/bin/npm", tar.TypeSymlink, "../lib/node_modules/npm/bin/npm-cli.js"),
		linkEntry("node-v24.19.0-darwin-arm64/bin/corepack", tar.TypeSymlink, "../lib/node_modules/corepack/dist/corepack.js"),
	})
	dest := filepath.Join(root, "out")

	result, err := ExtractTarGzipWithOptions(archive, dest, digestFile(t, archive), Limits{}, ExtractOptions{AllowRelativeSymlinks: true})
	if err != nil {
		t.Fatalf("ExtractTarGzipWithOptions failed: %v", err)
	}
	if result.Entries != 8 || result.Bytes != int64(len("node")+len("npm")+len("corepack")) {
		t.Fatalf("result = %+v", result)
	}
	assertFile(t, filepath.Join(dest, "node-v24.19.0-darwin-arm64", "bin", "node"), "node")
	assertSymlink(t, filepath.Join(dest, "node-v24.19.0-darwin-arm64", "bin", "npm"), "../lib/node_modules/npm/bin/npm-cli.js")
	assertSymlink(t, filepath.Join(dest, "node-v24.19.0-darwin-arm64", "bin", "corepack"), "../lib/node_modules/corepack/dist/corepack.js")
	assertFile(t, filepath.Join(dest, "node-v24.19.0-darwin-arm64", "bin", "npm"), "npm")
	assertFile(t, filepath.Join(dest, "node-v24.19.0-darwin-arm64", "bin", "corepack"), "corepack")
}

func TestExtractTarGzipWithOptionsRejectsHostileSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixture is POSIX-specific")
	}
	root := t.TempDir()
	tests := map[string][]tarEntry{
		"absolute target": {
			dirEntry("pkg/bin", 0o755),
			fileEntry("pkg/lib/tool", []byte("ok"), 0o644),
			linkEntry("pkg/bin/tool", tar.TypeSymlink, "/tmp/tool"),
		},
		"target escapes archive root": {
			dirEntry("pkg/bin", 0o755),
			fileEntry("pkg/lib/tool", []byte("ok"), 0o644),
			linkEntry("pkg/bin/tool", tar.TypeSymlink, "../../outside"),
		},
		"dangling internal target": {
			dirEntry("pkg/bin", 0o755),
			linkEntry("pkg/bin/tool", tar.TypeSymlink, "../lib/missing"),
		},
		"symlink chain": {
			dirEntry("pkg/bin", 0o755),
			dirEntry("pkg/lib", 0o755),
			fileEntry("pkg/lib/tool", []byte("ok"), 0o644),
			linkEntry("pkg/bin/first", tar.TypeSymlink, "second"),
			linkEntry("pkg/bin/second", tar.TypeSymlink, "../lib/tool"),
		},
		"symlink parent traversal": {
			dirEntry("pkg/bin", 0o755),
			fileEntry("pkg/lib/tool", []byte("ok"), 0o644),
			linkEntry("pkg/bin", tar.TypeSymlink, "../lib/tool"),
			fileEntry("pkg/bin/tool", []byte("bad"), 0o644),
		},
		"duplicate symlink destination": {
			dirEntry("pkg/bin", 0o755),
			fileEntry("pkg/lib/tool", []byte("ok"), 0o644),
			linkEntry("pkg/bin/tool", tar.TypeSymlink, "../lib/tool"),
			linkEntry("pkg/bin/tool", tar.TypeSymlink, "../lib/tool"),
		},
		"absolute link path": {
			fileEntry("pkg/lib/tool", []byte("ok"), 0o644),
			linkEntry("/pkg/bin/tool", tar.TypeSymlink, "../lib/tool"),
		},
		"unclean link target": {
			dirEntry("pkg/bin", 0o755),
			fileEntry("pkg/lib/tool", []byte("ok"), 0o644),
			linkEntry("pkg/bin/tool", tar.TypeSymlink, "../lib/./tool"),
		},
	}
	for name, entries := range tests {
		t.Run(name, func(t *testing.T) {
			archive := writeArchive(t, root, entries)
			dest := filepath.Join(root, strings.ReplaceAll(name, " ", "-"))

			_, err := ExtractTarGzipWithOptions(archive, dest, digestFile(t, archive), Limits{}, ExtractOptions{AllowRelativeSymlinks: true})
			if err == nil {
				t.Fatal("ExtractTarGzipWithOptions succeeded, want rejection")
			}
			if _, statErr := os.Lstat(dest); !os.IsNotExist(statErr) {
				t.Fatalf("destination exists after rejection; err=%v", statErr)
			}
		})
	}
}

func TestMkdirNoSymlinkBelowRejectsSymlinkInsideRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink rejection regression is POSIX-specific")
	}
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	injected := filepath.Join(root, "injected")
	if err := os.Symlink(outside, injected); err != nil {
		t.Fatal(err)
	}

	err := mkdirNoSymlinkBelow(root, filepath.Join(injected, "child"))
	if err == nil {
		t.Fatal("mkdirNoSymlinkBelow accepted symlink inside root")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error = %v, want symlink rejection", err)
	}
}

func TestExtractTarGzipVerifiesBeforeDestinationMutation(t *testing.T) {
	root := t.TempDir()
	archive := writeArchive(t, root, []tarEntry{fileEntry("ok.txt", []byte("ok"), 0o644)})
	dest := filepath.Join(root, "out")

	err := extractError(archive, dest, Digest{Size: digestFile(t, archive).Size + 1, SHA256: digestFile(t, archive).SHA256}, Limits{})
	if err == nil {
		t.Fatal("ExtractTarGzip accepted bad digest")
	}
	if _, statErr := os.Lstat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("destination was mutated before verification; stat err=%v", statErr)
	}
}

func TestExtractTarGzipFailureLeavesEmptyDestinationUnchanged(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "out")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "escape")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	archive := writeArchive(t, root, []tarEntry{fileEntry("../escape", []byte("bad"), 0o644)})

	err := extractError(archive, dest, digestFile(t, archive), Limits{})
	if err == nil {
		t.Fatal("ExtractTarGzip accepted hostile archive")
	}
	entries, readErr := os.ReadDir(dest)
	if readErr != nil {
		t.Fatalf("destination missing after failure: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("destination entries after failure = %v, want empty", entries)
	}
	assertFile(t, outside, "keep")
}

func TestExtractTarGzipRejectsHostileArchiveClasses(t *testing.T) {
	root := t.TempDir()
	tests := map[string][]tarEntry{
		"absolute path":       {fileEntry("/abs", []byte("bad"), 0o644)},
		"drive path":          {fileEntry("C:/tool", []byte("bad"), 0o644)},
		"backslash path":      {fileEntry(`bin\tool`, []byte("bad"), 0o644)},
		"dot path":            {fileEntry(".", []byte("bad"), 0o644)},
		"traversal path":      {fileEntry("../tool", []byte("bad"), 0o644)},
		"unclean path":        {fileEntry("bin/./tool", []byte("bad"), 0o644)},
		"duplicate file":      {fileEntry("bin/tool", []byte("one"), 0o644), fileEntry("bin/tool", []byte("two"), 0o644)},
		"file parent":         {fileEntry("bin", []byte("bad"), 0o644), fileEntry("bin/tool", []byte("bad"), 0o644)},
		"directory after kid": {fileEntry("bin/tool", []byte("bad"), 0o644), dirEntry("bin", 0o755)},
		"symlink":             {linkEntry("bin/tool", tar.TypeSymlink, "target")},
		"hardlink":            {linkEntry("bin/tool", tar.TypeLink, "target")},
		"character device":    {specialEntry("dev/tty", tar.TypeChar)},
		"block device":        {specialEntry("dev/disk", tar.TypeBlock)},
		"fifo":                {specialEntry("tmp/pipe", tar.TypeFifo)},
		"socket":              {specialEntry("tmp/socket", 's')},
		"pax global":          {paxGlobalEntry()},
		"sparse":              {specialEntry("sparse", 'S')},
	}
	for name, entries := range tests {
		t.Run(name, func(t *testing.T) {
			archive := writeArchive(t, root, entries)
			dest := filepath.Join(root, strings.ReplaceAll(name, " ", "-"))

			if err := extractError(archive, dest, digestFile(t, archive), Limits{}); err == nil {
				t.Fatal("ExtractTarGzip succeeded, want rejection")
			}
			if _, err := os.Lstat(dest); !os.IsNotExist(err) {
				t.Fatalf("destination exists after rejection; err=%v", err)
			}
		})
	}
}

func TestExtractTarGzipRejectsEntryAndOutputLimits(t *testing.T) {
	root := t.TempDir()
	tests := map[string]struct {
		entries []tarEntry
		limits  Limits
	}{
		"too many entries": {
			entries: []tarEntry{fileEntry("one", []byte("1"), 0o644), fileEntry("two", []byte("2"), 0o644)},
			limits:  Limits{MaxEntries: 1, MaxFileBytes: 10, MaxOutputBytes: 10},
		},
		"file too large": {
			entries: []tarEntry{fileEntry("one", []byte("12345"), 0o644)},
			limits:  Limits{MaxEntries: 10, MaxFileBytes: 4, MaxOutputBytes: 10},
		},
		"aggregate too large": {
			entries: []tarEntry{fileEntry("one", []byte("123"), 0o644), fileEntry("two", []byte("456"), 0o644)},
			limits:  Limits{MaxEntries: 10, MaxFileBytes: 10, MaxOutputBytes: 5},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			archive := writeArchive(t, root, tt.entries)
			dest := filepath.Join(root, strings.ReplaceAll(name, " ", "-"))
			if err := extractError(archive, dest, digestFile(t, archive), tt.limits); err == nil {
				t.Fatal("ExtractTarGzip succeeded, want limit rejection")
			}
			if _, err := os.Lstat(dest); !os.IsNotExist(err) {
				t.Fatalf("destination exists after rejection; err=%v", err)
			}
		})
	}
}

func TestExtractTarGzipRejectsTruncatedAndCorruptStreams(t *testing.T) {
	root := t.TempDir()
	valid := mustRead(t, writeArchive(t, root, []tarEntry{fileEntry("ok", []byte("ok"), 0o644)}))
	tests := map[string][]byte{
		"truncated gzip": valid[:len(valid)/2],
		"corrupt gzip":   []byte("not gzip"),
		"corrupt tar":    gzipBytes(t, []byte("not a tar stream")),
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			archive := filepath.Join(root, strings.ReplaceAll(name, " ", "-")+".tgz")
			if err := os.WriteFile(archive, data, 0o600); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(root, strings.ReplaceAll(name, " ", "-")+"-out")
			if err := extractError(archive, dest, digestFor(data), Limits{}); err == nil {
				t.Fatal("ExtractTarGzip succeeded, want corrupt stream rejection")
			}
			if _, err := os.Lstat(dest); !os.IsNotExist(err) {
				t.Fatalf("destination exists after corrupt stream; err=%v", err)
			}
		})
	}
}

func TestExtractTarGzipRejectsExistingDestinationSymlinkAndNonEmpty(t *testing.T) {
	root := t.TempDir()
	archive := writeArchive(t, root, []tarEntry{fileEntry("ok", []byte("ok"), 0o644)})

	nonEmpty := filepath.Join(root, "non-empty")
	if err := os.Mkdir(nonEmpty, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonEmpty, "keep"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := extractError(archive, nonEmpty, digestFile(t, archive), Limits{}); err == nil {
		t.Fatal("ExtractTarGzip accepted non-empty destination")
	}
	assertFile(t, filepath.Join(nonEmpty, "keep"), "keep")

	if runtime.GOOS != "windows" {
		link := filepath.Join(root, "link")
		if err := os.Symlink(nonEmpty, link); err != nil {
			t.Fatal(err)
		}
		if err := extractError(archive, link, digestFile(t, archive), Limits{}); err == nil {
			t.Fatal("ExtractTarGzip accepted symlink destination")
		}
	}
}

type tarEntry struct {
	header tar.Header
	body   []byte
}

func fileEntry(name string, body []byte, mode int64) tarEntry {
	return tarEntry{
		header: tar.Header{
			Name:       name,
			Typeflag:   tar.TypeReg,
			Mode:       mode,
			Size:       int64(len(body)),
			Uid:        1234,
			Gid:        5678,
			Uname:      "ignored",
			Gname:      "ignored",
			ModTime:    time.Unix(123456, 0),
			AccessTime: time.Unix(123456, 0),
			ChangeTime: time.Unix(123456, 0),
			PAXRecords: map[string]string{"SCHILY.xattr.user.test": "ignored"},
		},
		body: body,
	}
}

func dirEntry(name string, mode int64) tarEntry {
	return tarEntry{header: tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: mode}}
}

func linkEntry(name string, typ byte, target string) tarEntry {
	return tarEntry{header: tar.Header{Name: name, Typeflag: typ, Linkname: target, Mode: 0o777}}
}

func specialEntry(name string, typ byte) tarEntry {
	return tarEntry{header: tar.Header{Name: name, Typeflag: typ, Devmajor: 1, Devminor: 2}}
}

func paxGlobalEntry() tarEntry {
	return tarEntry{header: tar.Header{
		Typeflag:   tar.TypeXGlobalHeader,
		PAXRecords: map[string]string{"comment": "unsupported"},
	}}
}

func writeArchive(t *testing.T, root string, entries []tarEntry) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, entry := range entries {
		hdr := entry.header
		if err := tw.WriteHeader(&hdr); err != nil {
			t.Fatalf("WriteHeader(%q) failed: %v", hdr.Name, err)
		}
		if len(entry.body) > 0 {
			if _, err := tw.Write(entry.body); err != nil {
				t.Fatalf("Write(%q) failed: %v", hdr.Name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(root, "fixture-*.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Write(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	return file.Name()
}

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func extractError(archive, dest string, digest Digest, limits Limits) error {
	_, err := ExtractTarGzip(archive, dest, digest, limits)
	return err
}

func digestFile(t *testing.T, file string) Digest {
	t.Helper()
	return digestFor(mustRead(t, file))
}

func digestFor(data []byte) Digest {
	sum := sha256.Sum256(data)
	return Digest{Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])}
}

func mustRead(t *testing.T, file string) []byte {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertFile(t *testing.T, file, want string) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", file, data, want)
	}
}

func assertMode(t *testing.T, file string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %o, want %o", file, got, want)
	}
}

func assertSymlink(t *testing.T, file, want string) {
	t.Helper()
	got, err := os.Readlink(file)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s target = %q, want %q", file, got, want)
	}
}
