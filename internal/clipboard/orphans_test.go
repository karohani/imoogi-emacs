package clipboard

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

type orphanVault struct {
	root     string
	assets   string
	document string
	owner    ResolvedOwner
	store    Store
}

func newOrphanVault(t *testing.T) orphanVault {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	document := filepath.Join(root, "00. Inbox", "a.org")
	writeTestFile(t, document, "")
	owner, err := ResolveOwner(Owner{Kind: OwnerProjectNotes, Document: document, Root: root, AssetDir: "50. Assets"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(owner.AssetRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	return orphanVault{root: root, assets: owner.AssetRoot, document: document, owner: owner, store: Store{Base: t.TempDir()}}
}

// asset writes an aged asset file so the grace window does not hide it.
func (v orphanVault) asset(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(v.assets, name)
	writeTestFile(t, path, "asset:"+name)
	ageTestFile(t, path)
	return path
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func ageTestFile(t *testing.T, path string) {
	t.Helper()
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
}

func orphanNames(orphans []Orphan) []string {
	names := make([]string, 0, len(orphans))
	for _, orphan := range orphans {
		names = append(names, filepath.Base(orphan.Path))
	}
	sort.Strings(names)
	return names
}

func mustFindOrphans(t *testing.T, owner ResolvedOwner, store Store, opts OrphanOptions) []string {
	t.Helper()
	orphans, err := FindOrphans(owner, store, opts)
	if err != nil {
		t.Fatal(err)
	}
	return orphanNames(orphans)
}

func TestFindOrphansProjectVault(t *testing.T) {
	v := newOrphanVault(t)
	v.asset(t, "a.png")
	v.asset(t, "b.png")
	v.asset(t, "c x.png")
	v.asset(t, "d.png")
	v.asset(t, ".hidden.png")
	writeTestFile(t, filepath.Join(v.assets, assetIndexFilename), `{"version":1}`)
	ageTestFile(t, filepath.Join(v.assets, assetIndexFilename))
	if err := os.MkdirAll(filepath.Join(v.assets, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, v.document, "* note\n[[file:../50. Assets/a.png]]\n[[file:../50. Assets/d.png::*Heading][desc]]\n")
	writeTestFile(t, filepath.Join(v.root, "10. Notes", "m.md"), "![c](../50.%20Assets/c%20x.png)\n")
	// A document in a hidden directory is not part of the vault.
	writeTestFile(t, filepath.Join(v.root, ".git", "x.org"), "[[file:../50. Assets/b.png]]")

	got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{})
	if want := []string{"b.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphans = %v, want %v", got, want)
	}
	orphans, err := FindOrphans(v.owner, v.store, OrphanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if orphans[0].Path != filepath.Join(v.assets, "b.png") || orphans[0].Size != int64(len("asset:b.png")) || orphans[0].ModTime.IsZero() {
		t.Fatalf("orphan = %#v", orphans[0])
	}
}

// Each link below reaches its asset only through a differently named symlink
// outside the asset root, so the basename guard cannot be what keeps it.
func TestFindOrphansResolvesLinksByFileIdentity(t *testing.T) {
	v := newOrphanVault(t)
	links := filepath.Join(v.root, "links")
	if err := os.MkdirAll(links, 0o755); err != nil {
		t.Fatal(err)
	}
	for index, alias := range []string{"alias1.png", "alias2.png", "alias3.png", "alias 4.png", "alias[5].png"} {
		target := v.asset(t, "real-"+string(rune('a'+index))+".png")
		if err := os.Symlink(target, filepath.Join(links, alias)); err != nil {
			t.Fatal(err)
		}
	}
	v.asset(t, "real-unlinked.png")
	writeTestFile(t, v.document, strings.Join([]string{
		"[[file:../links/alias1.png]]",
		"[[file:../links/alias2.png::search text][d]]",
		"[[file:../links/alias%5B5%5D.png][org-encoded]]",
	}, "\n"))
	writeTestFile(t, filepath.Join(v.root, "m.md"), strings.Join([]string{
		`[t](file:links/alias3.png "a title")`,
		"![e](links/alias%204.png)",
	}, "\n"))

	got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{})
	if want := []string{"real-unlinked.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphans = %v, want %v", got, want)
	}
}

func TestFindOrphansUnicodeNormalization(t *testing.T) {
	v := newOrphanVault(t)
	screenshotNFC := "스크린샷.png"
	photoNFC := "사진.png"
	pictureNFC := "그림.png"
	v.asset(t, hangulNFD(screenshotNFC))
	v.asset(t, photoNFC)
	v.asset(t, hangulNFD(pictureNFC))
	v.asset(t, "unused.png")
	writeTestFile(t, filepath.Join(v.root, "n.md"), "![s](50.%20Assets/"+screenshotNFC+")\n![p](50.%20Assets/"+hangulNFD(photoNFC)+")\n")
	// Only the basename guard sees an HTML tag; NFC text against an NFD name.
	writeTestFile(t, filepath.Join(v.root, "h.md"), `<img src="50. Assets/`+pictureNFC+`">`)

	got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{})
	if want := []string{"unused.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphans = %v, want %v", got, want)
	}
}

func TestFindOrphansBasenameGuard(t *testing.T) {
	v := newOrphanVault(t)
	for _, name := range []string{"y.png", "z.png", "bare.pdf", "Upper.PNG", "q&a+1.png", "free.png"} {
		v.asset(t, name)
	}
	writeTestFile(t, v.document, strings.Join([]string{
		`<img src="50. Assets/y.png">`,
		"[[./50. Assets/z.png]]",
		"see bare.pdf for details",
		"[[file:upper.png]]",
		"emacs-hexified: q%26a%2b1.png",
	}, "\n"))

	got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{})
	if want := []string{"free.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphans = %v, want %v", got, want)
	}
}

func TestFindOrphansGraceWindow(t *testing.T) {
	v := newOrphanVault(t)
	path := filepath.Join(v.assets, "fresh.png")
	writeTestFile(t, path, "fresh")
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	modified := now.Add(-time.Hour)
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
	clock := func() time.Time { return now }
	if got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{Now: clock}); len(got) != 0 {
		t.Fatalf("default grace reported %v", got)
	}
	if got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{Now: clock, Grace: 30 * time.Minute}); !reflect.DeepEqual(got, []string{"fresh.png"}) {
		t.Fatalf("short grace orphans = %v", got)
	}
}

func TestFindOrphansProtectedPaths(t *testing.T) {
	v := newOrphanVault(t)
	v.asset(t, "kept.png")
	v.asset(t, "free.png")
	// Reach the protected file through a symlinked spelling of the root.
	alias := filepath.Join(t.TempDir(), "vault")
	if err := os.Symlink(v.root, alias); err != nil {
		t.Fatal(err)
	}
	protected := []string{filepath.Join(alias, "50. Assets", "kept.png"), filepath.Join(v.assets, "missing.png")}
	got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{Protected: protected})
	if want := []string{"free.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphans = %v, want %v", got, want)
	}
}

func TestFindOrphansProtectsInFlightTransactions(t *testing.T) {
	v := newOrphanVault(t)
	dedup := v.asset(t, "dedup.png")
	created := v.asset(t, "created.png")
	committed := v.asset(t, "committed.png")
	aborted := v.asset(t, "aborted.png")
	owner := v.owner.Owner
	identity := Identity{Session: "s", Buffer: "b", Generation: 1}

	// A dedup hit reuses an existing asset: it is in Assets, not CreatedFiles.
	if _, _, err := v.store.Create(identity, owner, []Asset{{Path: dedup}}, nil); err != nil {
		t.Fatal(err)
	}
	if _, _, err := v.store.Create(identity, owner, nil, []string{created}); err != nil {
		t.Fatal(err)
	}
	done, directory, err := v.store.Create(identity, owner, []Asset{{Path: committed}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.store.Prepare(directory, done.TransactionToken, owner, []Asset{{Path: committed}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, state := range []TransactionState{StateDocumentSaved, StateCommitted} {
		if _, err := v.store.Transition(directory, done.TransactionToken, state, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	cancelled, directory, err := v.store.Create(identity, owner, []Asset{{Path: aborted}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.store.Transition(directory, cancelled.TransactionToken, StateAborted, nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{})
	if want := []string{"aborted.png", "committed.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphans = %v, want %v", got, want)
	}
}

func TestFindOrphansRejectsUndecodableManifest(t *testing.T) {
	v := newOrphanVault(t)
	v.asset(t, "a.png")
	writeTestFile(t, filepath.Join(v.store.Base, "sessions", "s", "b", "1", "transactions", "t1", manifestFilename), "{not json")
	if _, err := FindOrphans(v.owner, v.store, OrphanOptions{}); err == nil {
		t.Fatal("undecodable manifest ignored")
	}
}

func TestFindOrphansRequiresStoreBase(t *testing.T) {
	v := newOrphanVault(t)
	v.asset(t, "a.png")
	if _, err := FindOrphans(v.owner, Store{}, OrphanOptions{}); err == nil {
		t.Fatal("empty store base accepted")
	}
}

func TestFindOrphansIgnoresSymlinkedAssets(t *testing.T) {
	v := newOrphanVault(t)
	outside := filepath.Join(t.TempDir(), "outside.png")
	writeTestFile(t, outside, "outside")
	ageTestFile(t, outside)
	link := filepath.Join(v.assets, "link.png")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{}); len(got) != 0 {
		t.Fatalf("orphans = %v", got)
	}
	trashed, err := TrashOrphans(v.owner, v.store, OrphanOptions{}, []string{link}, failingTrash(t))
	if err != nil || len(trashed) != 0 {
		t.Fatalf("trashed = %v, err = %v", trashed, err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal(err)
	}
}

func TestFindOrphansAbortsOnUnreadableDocument(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every file")
	}
	v := newOrphanVault(t)
	v.asset(t, "a.png")
	locked := filepath.Join(v.root, "locked.org")
	writeTestFile(t, locked, "[[file:50. Assets/a.png]]")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o644) })
	if _, err := FindOrphans(v.owner, v.store, OrphanOptions{}); err == nil {
		t.Fatal("unreadable document treated as having no references")
	}
}

func TestFindOrphansAbortsOnDanglingDocumentSymlink(t *testing.T) {
	v := newOrphanVault(t)
	v.asset(t, "a.png")
	if err := os.Symlink(filepath.Join(v.root, "gone.org"), filepath.Join(v.root, "elsewhere.org")); err != nil {
		t.Fatal(err)
	}
	if _, err := FindOrphans(v.owner, v.store, OrphanOptions{}); err == nil {
		t.Fatal("dangling document symlink treated as having no references")
	}
}

func TestFindOrphansAbortsOnSymlinkedDirectory(t *testing.T) {
	v := newOrphanVault(t)
	v.asset(t, "a.png")
	elsewhere := t.TempDir()
	writeTestFile(t, filepath.Join(elsewhere, "hidden.org"), "[[file:a.png]]")
	if err := os.Symlink(elsewhere, filepath.Join(v.root, "linked notes")); err != nil {
		t.Fatal(err)
	}
	if _, err := FindOrphans(v.owner, v.store, OrphanOptions{}); err == nil {
		t.Fatal("documents behind a symlinked directory treated as absent")
	}
}

func TestFindOrphansMissingProtectedPathMatchesName(t *testing.T) {
	v := newOrphanVault(t)
	v.asset(t, hangulNFD("사진.png"))
	v.asset(t, "free.png")
	// A composed spelling from Emacs may not resolve on a non-APFS volume.
	protected := []string{filepath.Join(t.TempDir(), "elsewhere", "사진.PNG")}
	if got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{Protected: protected}); !reflect.DeepEqual(got, []string{"free.png"}) {
		t.Fatalf("orphans = %v", got)
	}
}

func TestFindOrphansSkipsEmacsLockFiles(t *testing.T) {
	v := newOrphanVault(t)
	v.asset(t, "a.png")
	// Emacs marks a modified buffer with a dangling symlink named .#<file>.
	if err := os.Symlink("jay@host.1234:1700000000", filepath.Join(filepath.Dir(v.document), ".#a.org")); err != nil {
		t.Fatal(err)
	}
	if got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{}); !reflect.DeepEqual(got, []string{"a.png"}) {
		t.Fatalf("orphans = %v", got)
	}
}

func TestFindOrphansMissingAssetRoot(t *testing.T) {
	v := newOrphanVault(t)
	if err := os.Remove(v.assets); err != nil {
		t.Fatal(err)
	}
	orphans, err := FindOrphans(v.owner, v.store, OrphanOptions{})
	if err != nil || len(orphans) != 0 {
		t.Fatalf("orphans = %v, err = %v", orphans, err)
	}
}

func TestFindOrphansStandaloneScansOnlyItsDocument(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	document := filepath.Join(directory, "a.org")
	root := filepath.Join(directory, "a.assets")
	owner, err := ResolveOwner(Owner{Kind: OwnerStandalone, Document: document, Root: root}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"mine.png", "theirs.png"} {
		writeTestFile(t, filepath.Join(root, name), name)
		ageTestFile(t, filepath.Join(root, name))
	}
	store := Store{Base: t.TempDir()}
	if _, err := FindOrphans(owner, store, OrphanOptions{}); err == nil {
		t.Fatal("missing standalone document treated as having no references")
	}
	writeTestFile(t, document, "[[file:a.assets/mine.png]]")
	writeTestFile(t, filepath.Join(directory, "b.org"), "[[file:a.assets/theirs.png]]")
	if got := mustFindOrphans(t, owner, store, OrphanOptions{}); !reflect.DeepEqual(got, []string{"theirs.png"}) {
		t.Fatalf("orphans = %v", got)
	}
}

func TestFindOrphansRejectsStagingOwner(t *testing.T) {
	staging := t.TempDir()
	owner, err := ResolveOwner(Owner{Kind: OwnerStaging, Session: "s", Buffer: "b", Generation: 1}, staging)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FindOrphans(owner, Store{Base: staging}, OrphanOptions{}); err == nil {
		t.Fatal("staging owner accepted")
	}
}

// fakeTrash moves files into a temporary directory and records what moved.
func fakeTrash(t *testing.T) (func(string) error, string) {
	t.Helper()
	bin := t.TempDir()
	return func(path string) error {
		return os.Rename(path, filepath.Join(bin, filepath.Base(path)))
	}, bin
}

func failingTrash(t *testing.T) func(string) error {
	return func(path string) error {
		t.Errorf("trash called for %q", path)
		return nil
	}
}

func TestTrashOrphansMovesOnlyFreshOrphans(t *testing.T) {
	v := newOrphanVault(t)
	orphan := v.asset(t, "orphan.png")
	referenced := v.asset(t, "referenced.png")
	writeTestFile(t, v.document, "[[file:../50. Assets/referenced.png]]")
	outside := filepath.Join(v.root, "outside.png")
	writeTestFile(t, outside, "outside")
	ageTestFile(t, outside)
	nested := filepath.Join(v.assets, "nested", "deep.png")
	writeTestFile(t, nested, "deep")
	ageTestFile(t, nested)
	trash, bin := fakeTrash(t)

	requested := []string{orphan, referenced, outside, nested, filepath.Join(v.assets, "missing.png")}
	trashed, err := TrashOrphans(v.owner, v.store, OrphanOptions{}, requested, trash)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{orphan}; !reflect.DeepEqual(trashed, want) {
		t.Fatalf("trashed = %v, want %v", trashed, want)
	}
	if _, err := os.Stat(filepath.Join(bin, "orphan.png")); err != nil {
		t.Fatalf("orphan not in trash: %v", err)
	}
	for _, kept := range []string{referenced, outside, nested} {
		if _, err := os.Stat(kept); err != nil {
			t.Fatalf("%s was moved: %v", kept, err)
		}
	}
}

func TestTrashOrphansRescansBeforeMoving(t *testing.T) {
	v := newOrphanVault(t)
	path := v.asset(t, "late.png")
	if got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{}); !reflect.DeepEqual(got, []string{"late.png"}) {
		t.Fatalf("orphans = %v", got)
	}
	// The user links the file again between listing and trashing.
	writeTestFile(t, v.document, "[[file:../50. Assets/late.png]]")
	trashed, err := TrashOrphans(v.owner, v.store, OrphanOptions{}, []string{path}, failingTrash(t))
	if err != nil || len(trashed) != 0 {
		t.Fatalf("trashed = %v, err = %v", trashed, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestTrashOrphansReportsTrashFailure(t *testing.T) {
	v := newOrphanVault(t)
	path := v.asset(t, "stuck.png")
	trashed, err := TrashOrphans(v.owner, v.store, OrphanOptions{}, []string{path}, func(string) error { return os.ErrPermission })
	if err == nil || len(trashed) != 0 {
		t.Fatalf("trashed = %v, err = %v", trashed, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestTrashOrphansRequiresFileToLeave(t *testing.T) {
	v := newOrphanVault(t)
	path := v.asset(t, "stays.png")
	trashed, err := TrashOrphans(v.owner, v.store, OrphanOptions{}, []string{path}, func(string) error { return nil })
	if err == nil || len(trashed) != 0 {
		t.Fatalf("trashed = %v, err = %v", trashed, err)
	}
}

func TestDecodePercentLenient(t *testing.T) {
	tests := map[string]string{
		"a%20b":          "a b",
		"%E2%9C%93":      "✓",
		"100%":           "100%",
		"%zz%2":          "%zz%2",
		"a+b":            "a+b",
		"%5b%5D":         "[]",
		"plain/path.png": "plain/path.png",
	}
	for input, want := range tests {
		if got := decodePercentLenient(input); got != want {
			t.Errorf("decodePercentLenient(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestHangulNormalization(t *testing.T) {
	composed := "한"     // 한
	decomposed := "한" // ᄒ ᅡ ᆫ
	if got := hangulNFD(composed + ".png"); got != decomposed+".png" {
		t.Fatalf("hangulNFD = %q", got)
	}
	if got := hangulNFC(decomposed + ".png"); got != composed+".png" {
		t.Fatalf("hangulNFC = %q", got)
	}
	if got := hangulNFC("가"); got != "가" {
		t.Fatalf("hangulNFC(LV) = %q", got)
	}
}

func TestFindOrphansMarkdownBackslashEscapes(t *testing.T) {
	v := newOrphanVault(t)
	v.asset(t, "photo(1).png")
	v.asset(t, "a_b[2].png")
	v.asset(t, "free.png")
	writeTestFile(t, v.document, strings.Join([]string{
		`![image](50.\ Assets/photo\(1\).png)`,
		`see a\_b\[2\].png`,
	}, "\n"))

	got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{})
	if want := []string{"free.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphans = %v, want %v", got, want)
	}
}

func TestFindOrphansUnescapesMarkdownTargetsBeforeResolving(t *testing.T) {
	v := newOrphanVault(t)
	real := v.asset(t, "real.png")
	links := filepath.Join(v.root, "links")
	if err := os.MkdirAll(links, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(links, "alias_1.png")); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, v.document, `![image](../links/alias\_1.png)`)

	if got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{}); len(got) != 0 {
		t.Fatalf("orphans = %v, want none", got)
	}
}

func TestTrashOrphansMatchesTheRequestedNameNotASharedInode(t *testing.T) {
	v := newOrphanVault(t)
	first := v.asset(t, "a.png")
	second := filepath.Join(v.assets, "b.png")
	if err := os.Link(first, second); err != nil {
		t.Fatal(err)
	}
	trash, _ := fakeTrash(t)

	trashed, err := TrashOrphans(v.owner, v.store, OrphanOptions{}, []string{second}, trash)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{second}; !reflect.DeepEqual(trashed, want) {
		t.Fatalf("trashed = %v, want %v", trashed, want)
	}
	if _, err := os.Lstat(first); err != nil {
		t.Fatalf("sibling hard link was moved: %v", err)
	}
}

func TestFindOrphansProtectsAssetsAliasedBySymlinks(t *testing.T) {
	v := newOrphanVault(t)
	real := v.asset(t, "real.png")
	v.asset(t, "free.png")
	links := filepath.Join(v.root, "links")
	if err := os.MkdirAll(links, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(links, "alias.png")); err != nil {
		t.Fatal(err)
	}
	// A reference-style Markdown link the parser does not resolve.
	writeTestFile(t, v.document, "![image][photo]\n\n[photo]: ../links/alias.png\n")

	got := mustFindOrphans(t, v.owner, v.store, OrphanOptions{})
	if want := []string{"free.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphans = %v, want %v", got, want)
	}
}
