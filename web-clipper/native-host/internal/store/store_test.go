package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
)

const rid = "8c3f0f8e-1f8a-4c55-9a59-0e5a3f8f9d21"

func openTemp(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, dir
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"원문 제목":                  "원문-제목",
		"Hello, World!":          "hello-world",
		"  ../../etc/passwd  ":   "etc-passwd",
		"a\\b/c\x00d\ne":         "a-b-c-d-e",
		".hidden":                "hidden",
		"...":                    "untitled",
		"":                       "untitled",
		"日本語のタイトル — test":        "日本語のタイトル-test",
		"école":                 "école", // NFC
		"tab\tand​zwsp":          "tab-and-zwsp",
		"C++ & Go: 2026 edition": "c-go-2026-edition",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
	long := Slug(strings.Repeat("가나다 ", 40))
	if n := utf8.RuneCountInString(long); n > 60 || strings.HasSuffix(long, "-") {
		t.Errorf("long slug %q (%d runes)", long, n)
	}
}

func TestDocBaseName(t *testing.T) {
	at := time.Date(2026, 11, 8, 1, 0, 0, 0, time.FixedZone("KST", 9*3600))
	if got := DocBaseName(at, "원문 제목"); got != "2026-11-08-원문-제목" {
		t.Fatal(got)
	}
}

func TestCommitRequiresDocument(t *testing.T) {
	s, _ := openTemp(t)
	st, _ := s.Stage(rid)
	defer st.Discard()
	if _, err := st.Commit("00. Inbox/04. Clippings", "d"); err == nil {
		t.Fatal("commit without a document succeeded")
	}
}

func TestFreeName(t *testing.T) {
	s, dir := openTemp(t)
	cat := filepath.Join(dir, "00. Inbox", "04. Clippings")
	os.MkdirAll(filepath.Join(cat, "assets", "x-2"), 0o700)
	os.WriteFile(filepath.Join(cat, "x.org"), []byte("a"), 0o600)
	got, err := s.FreeName("00. Inbox/04. Clippings", "x")
	if err != nil || got != "x-3" {
		t.Fatalf("got %q %v", got, err)
	}
	if got, _ := s.FreeName("00. Inbox/04. Clippings", "y"); got != "y" {
		t.Fatalf("free base: %q", got)
	}
	if _, err := s.FreeName("00. Inbox/assets", "y"); err == nil {
		t.Fatal("category named assets accepted")
	}
}

func TestCommitWithoutAssetsCreatesNoAssetDir(t *testing.T) {
	s, dir := openTemp(t)
	st, _ := s.Stage(rid)
	st.WriteDoc("text only")
	rel, err := st.Commit("00. Inbox/04. Clippings", "t")
	if err != nil || rel != "00. Inbox/04. Clippings/t.org" {
		t.Fatalf("%q %v", rel, err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "00. Inbox", "04. Clippings", "assets")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("assets dir created for a clip without images: %v", err)
	}
}

func TestStageCommitLedger(t *testing.T) {
	s, dir := openTemp(t)
	st, err := s.Stage(rid)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.WriteAsset("0123456789abcdef0123456789abcdef.png", []byte("img")); err != nil {
		t.Fatal(err)
	}
	// Identical content within a clip dedupes silently.
	if err := st.WriteAsset("0123456789abcdef0123456789abcdef.png", []byte("img")); err != nil {
		t.Fatalf("dedupe: %v", err)
	}
	if err := st.WriteDoc("#+TITLE: x\n"); err != nil {
		t.Fatal(err)
	}
	rel, err := st.Commit("00. Inbox/04. Clippings", "2026-11-08-x")
	if err != nil {
		t.Fatal(err)
	}
	if rel != "00. Inbox/04. Clippings/2026-11-08-x.org" {
		t.Fatalf("rel %q", rel)
	}
	cat := filepath.Join(dir, "00. Inbox", "04. Clippings")
	b, err := os.ReadFile(filepath.Join(cat, "2026-11-08-x.org"))
	if err != nil || string(b) != "#+TITLE: x\n" {
		t.Fatalf("doc: %q %v", b, err)
	}
	for path, want := range map[string]fs.FileMode{
		filepath.Join(cat, "2026-11-08-x.org"):                                               0o600,
		filepath.Join(cat, "assets", "2026-11-08-x", "0123456789abcdef0123456789abcdef.png"): 0o600,
		filepath.Join(cat, "assets", "2026-11-08-x"):                                         0o700,
		filepath.Join(cat, "assets"):                                                         0o700,
		filepath.Join(dir, "00. Inbox"):                                                      0o700,
		filepath.Join(dir, ".org-clipper", "tmp"):                                            0o700,
	} {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != want {
			t.Errorf("%s mode %v, want %v", path, fi.Mode().Perm(), want)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(dir, ".org-clipper", "tmp"))
	if len(entries) != 0 {
		t.Fatalf("staging left behind: %v", entries)
	}

	res := protocol.Result{Type: "result", RequestID: rid, Status: protocol.StatusSaved, Path: rel, AssetsSaved: 1}
	if err := s.WriteLedger(res); err != nil {
		t.Fatal(err)
	}
	got, err := s.Lookup(rid)
	if err != nil || got == nil || got.Path != rel || got.Status != protocol.StatusSaved {
		t.Fatalf("lookup %+v %v", got, err)
	}
	if fi, _ := os.Stat(filepath.Join(dir, ".org-clipper", "requests", rid+".json")); fi.Mode().Perm() != 0o600 {
		t.Errorf("ledger mode %v", fi.Mode().Perm())
	}
	if fi, _ := os.Stat(filepath.Join(dir, ".org-clipper", "requests")); fi.Mode().Perm() != 0o700 {
		t.Errorf("requests dir mode %v", fi.Mode().Perm())
	}
	if none, err := s.Lookup("00000000-0000-4000-8000-000000000000"); none != nil || err != nil {
		t.Fatalf("missing ledger: %+v %v", none, err)
	}
}

func TestCommitNeverOverwrites(t *testing.T) {
	s, dir := openTemp(t)
	cat := filepath.Join(dir, "00. Inbox", "04. Clippings")
	os.MkdirAll(filepath.Join(cat, "assets", "a"), 0o700)
	os.WriteFile(filepath.Join(cat, "taken.org"), []byte("original"), 0o600)
	for _, name := range []string{"taken", "a"} {
		st, _ := s.Stage(rid)
		st.WriteAsset("0123456789abcdef0123456789abcdef.png", []byte("img"))
		st.WriteDoc("new")
		if _, err := st.Commit("00. Inbox/04. Clippings", name); !errors.Is(err, ErrExists) {
			t.Fatalf("%s: want ErrExists, got %v", name, err)
		}
		st.Discard()
	}
	b, _ := os.ReadFile(filepath.Join(cat, "taken.org"))
	if string(b) != "original" {
		t.Fatal("existing clip overwritten")
	}
	if _, err := os.Lstat(filepath.Join(cat, "assets", "taken")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("assets moved for a refused commit")
	}
}

func TestSymlinkedCategoryRefused(t *testing.T) {
	s, dir := openTemp(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "00. Inbox")); err != nil {
		t.Fatal(err)
	}
	st, err := s.Stage(rid)
	if err != nil {
		t.Fatal(err)
	}
	st.WriteDoc("x")
	_, err = st.Commit("00. Inbox/04. Clippings", "d")
	if err == nil {
		t.Fatal("commit through an escaping symlink succeeded")
	}
	if code := Classify(err); code != protocol.CodePermissionDenied {
		t.Errorf("classify %q (%v)", code, err)
	}
	st.Discard()
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("wrote outside root: %v", entries)
	}
}

func TestSymlinkedAssetsDirRefused(t *testing.T) {
	s, dir := openTemp(t)
	outside := t.TempDir()
	st, err := s.Stage(rid)
	if err != nil {
		t.Fatal(err)
	}
	assets := filepath.Join(dir, filepath.FromSlash(st.Dir()), "assets")
	if err := os.Remove(assets); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, assets); err != nil {
		t.Fatal(err)
	}
	if err := st.WriteAsset("0123456789abcdef0123456789abcdef.png", []byte("x")); err == nil {
		t.Fatal("asset written through escaping symlink")
	}
	st.Discard()
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("wrote outside root: %v", entries)
	}
}

func TestSymlinkedStateDirRefused(t *testing.T) {
	s, dir := openTemp(t)
	outside := t.TempDir()
	os.Symlink(outside, filepath.Join(dir, ".org-clipper"))
	if _, err := s.Stage(rid); err == nil {
		t.Fatal("staging through escaping symlink succeeded")
	}
	if _, _, err := s.Acquire(rid); err == nil {
		t.Fatal("lock through escaping symlink succeeded")
	}
	s.CleanTmp(0)
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("wrote outside root: %v", entries)
	}
}

func TestInvalidNamesRejected(t *testing.T) {
	s, _ := openTemp(t)
	st, _ := s.Stage(rid)
	defer st.Discard()
	for _, bad := range []string{"../x.png", "a/b.png", ".png", "x.gif", "", "0123456789abcdef0123456789abcdeg.png"} {
		if err := st.WriteAsset(bad, []byte("x")); err == nil {
			t.Errorf("asset name %q accepted", bad)
		}
	}
	for _, bad := range []string{"..", ".x", "a/b", "a\\b", "", "a\x00b"} {
		if _, err := st.Commit("00. Inbox/04. Clippings", bad); err == nil {
			t.Errorf("dir name %q accepted", bad)
		}
	}
	if _, err := s.Stage("../../x"); err == nil {
		t.Error("invalid request id accepted for staging")
	}
	if _, err := s.Lookup("../x"); err == nil {
		t.Error("invalid request id accepted for lookup")
	}
}

func TestLock(t *testing.T) {
	s, dir := openTemp(t)
	release, busy, err := s.Acquire(rid)
	if err != nil || busy {
		t.Fatalf("first acquire: busy=%v err=%v", busy, err)
	}
	if _, busy, err := s.Acquire(rid); err != nil || !busy {
		t.Fatalf("second acquire: busy=%v err=%v", busy, err)
	}
	release()
	lock := filepath.Join(dir, ".org-clipper", "requests", rid+".lock")
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatal("lock not removed")
	}
	// A stale lock (older than 10 min) is taken over.
	os.WriteFile(lock, nil, 0o600)
	old := time.Now().Add(-11 * time.Minute)
	os.Chtimes(lock, old, old)
	release, busy, err = s.Acquire(rid)
	if err != nil || busy {
		t.Fatalf("stale takeover: busy=%v err=%v", busy, err)
	}
	release()
	// A fresh lock is respected.
	os.WriteFile(lock, nil, 0o600)
	if _, busy, _ := s.Acquire(rid); !busy {
		t.Fatal("fresh foreign lock ignored")
	}
}

func TestCleanTmp(t *testing.T) {
	s, dir := openTemp(t)
	tmp := filepath.Join(dir, ".org-clipper", "tmp")
	os.MkdirAll(tmp, 0o700)
	outside := t.TempDir()
	victim := filepath.Join(outside, "keep.txt")
	os.WriteFile(victim, []byte("keep"), 0o600)

	oldDir := filepath.Join(tmp, "old")
	os.MkdirAll(oldDir, 0o700)
	os.Symlink(outside, filepath.Join(oldDir, "link"))
	os.WriteFile(filepath.Join(oldDir, "f"), []byte("x"), 0o600)
	os.MkdirAll(filepath.Join(tmp, "fresh"), 0o700)
	os.WriteFile(filepath.Join(dir, "unrelated-old.txt"), []byte("x"), 0o600)
	old := time.Now().Add(-25 * time.Hour)
	os.Chtimes(oldDir, old, old)
	os.Chtimes(filepath.Join(dir, "unrelated-old.txt"), old, old)

	s.CleanTmp(24 * time.Hour)

	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("old staging dir not removed")
	}
	if _, err := os.Stat(filepath.Join(tmp, "fresh")); err != nil {
		t.Error("fresh staging dir removed")
	}
	if _, err := os.Stat(victim); err != nil {
		t.Error("symlink target outside tmp was removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "unrelated-old.txt")); err != nil {
		t.Error("file outside tmp was removed")
	}
}

func TestOpenErrors(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "missing"))
	var pe *protocol.Error
	if !errors.As(err, &pe) || pe.Code != protocol.CodeRootUnavailable {
		t.Fatalf("got %v", err)
	}
	f := filepath.Join(t.TempDir(), "file")
	os.WriteFile(f, nil, 0o600)
	if _, err := Open(f); !errors.As(err, &pe) || pe.Code != protocol.CodeRootUnavailable {
		t.Fatalf("file as root: %v", err)
	}
}

func TestClassify(t *testing.T) {
	cases := map[error]string{
		&fs.PathError{Op: "write", Path: "x", Err: syscall.ENOSPC}: protocol.CodeDiskFull,
		&fs.PathError{Op: "open", Path: "x", Err: syscall.EACCES}:  protocol.CodePermissionDenied,
		&fs.PathError{Op: "open", Path: "x", Err: syscall.EPERM}:   protocol.CodePermissionDenied,
		&fs.PathError{Op: "open", Path: "x", Err: syscall.ENOENT}:  protocol.CodeRootUnavailable,
		ErrExists:           protocol.CodeInternal,
		errors.New("other"): protocol.CodeInternal,
	}
	for err, want := range cases {
		if got := Classify(err); got != want {
			t.Errorf("Classify(%v) = %q, want %q", err, got, want)
		}
	}
}

func TestPermissionDeniedOnReadOnlyRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	s, dir := openTemp(t)
	os.Chmod(dir, 0o500)
	defer os.Chmod(dir, 0o700)
	_, err := s.Stage(rid)
	if Classify(err) != protocol.CodePermissionDenied {
		t.Fatalf("got %v", err)
	}
}
