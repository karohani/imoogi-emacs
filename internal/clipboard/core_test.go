package clipboard

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveProjectOwnerContainsDocument(t *testing.T) {
	root := t.TempDir()
	document := filepath.Join(root, "notes", "a.org")
	if err := os.MkdirAll(filepath.Dir(document), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(document, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveOwner(Owner{Kind: OwnerProjectNotes, Document: document, Root: root}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.AssetRoot != filepath.Join(canonicalRoot, "assets") {
		t.Fatalf("asset root = %q", resolved.AssetRoot)
	}
	outside := filepath.Join(t.TempDir(), "outside.org")
	if _, err := ResolveOwner(Owner{Kind: OwnerProjectNotes, Document: outside, Root: root}, t.TempDir()); err == nil {
		t.Fatal("outside document accepted")
	}
}

func TestResolveProjectOwnerRejectsEscapingAssetSymlink(t *testing.T) {
	root := t.TempDir()
	document := filepath.Join(root, "note.org")
	if err := os.WriteFile(document, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "assets")); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveOwner(Owner{Kind: OwnerProjectNotes, Document: document, Root: root}, t.TempDir())
	if err == nil {
		t.Fatal("escaping assets symlink accepted")
	}
}

func TestResolveStandaloneRequiresDerivedSibling(t *testing.T) {
	document := filepath.Join(t.TempDir(), "memo.org")
	root := stringsTrimExtension(document) + ".assets"
	if _, err := ResolveOwner(Owner{Kind: OwnerStandalone, Document: document, Root: root}, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveOwner(Owner{Kind: OwnerStandalone, Document: document, Root: root + "-wrong"}, t.TempDir()); err == nil {
		t.Fatal("mismatched standalone root accepted")
	}
}

func TestPublisherRejectsDirectoryBeforePublishing(t *testing.T) {
	sourceRoot := t.TempDir()
	file := filepath.Join(sourceRoot, "a.txt")
	if err := os.WriteFile(file, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "assets")
	_, err := (Publisher{}).Publish([]string{file, sourceRoot}, destination)
	if err == nil {
		t.Fatal("directory batch accepted")
	}
	if entries, readErr := os.ReadDir(destination); !os.IsNotExist(readErr) && len(entries) != 0 {
		t.Fatalf("destination changed: %v, %v", entries, readErr)
	}
}

func TestPublisherReusesIdenticalAssetWithoutOverwrite(t *testing.T) {
	source := filepath.Join(t.TempDir(), "same.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	first, err := (Publisher{}).Publish([]string{source}, destination)
	if err != nil {
		t.Fatal(err)
	}
	second, err := (Publisher{}).Publish([]string{source}, destination)
	if err != nil {
		t.Fatal(err)
	}
	if first.Assets[0].Path != second.Assets[0].Path {
		t.Fatalf("publisher did not reuse existing file: %q != %q", first.Assets[0].Path, second.Assets[0].Path)
	}
	if first.Assets[0].SHA256 != second.Assets[0].SHA256 {
		t.Fatal("hash changed")
	}
	if first.Assets[0].ID == second.Assets[0].ID {
		t.Fatal("opaque asset id was reused")
	}
	if len(second.CreatedFiles) != 0 {
		t.Fatalf("duplicate publish created files: %v", second.CreatedFiles)
	}
	if _, err := os.Stat(filepath.Join(destination, assetIndexFilename)); err != nil {
		t.Fatalf("asset cache was not written: %v", err)
	}
	if err := os.WriteFile(source, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	third, err := (Publisher{}).Publish([]string{source}, destination)
	if err != nil {
		t.Fatal(err)
	}
	if third.Assets[0].Path == first.Assets[0].Path {
		t.Fatal("changed source reused stale asset")
	}
	if len(third.CreatedFiles) != 1 {
		t.Fatalf("changed source did not create asset: %v", third.CreatedFiles)
	}
}

func TestStoreLifecycleAndToken(t *testing.T) {
	now := time.Date(2026, 9, 16, 1, 2, 3, 0, time.UTC)
	store := Store{Base: t.TempDir(), Now: func() time.Time { return now }}
	identity := Identity{Session: "s", Buffer: "b", Generation: 1}
	owner := Owner{Kind: OwnerStaging, Session: "s", Buffer: "b", Generation: 1}
	manifest, directory, err := store.Create(identity, owner, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(directory, "wrong", StatePrepared, nil, nil, nil); err == nil {
		t.Fatal("wrong token accepted")
	}
	for _, state := range []TransactionState{StatePrepared, StateDocumentSaved, StateCommitted} {
		manifest, err = store.Transition(directory, manifest.TransactionToken, state, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Transition(directory, manifest.TransactionToken, StatePrepared, nil, nil, nil); err == nil {
		t.Fatal("terminal transition accepted")
	}
}

func TestPruneKeepsLiveProcessAndRemovesDeadProcess(t *testing.T) {
	now := time.Date(2026, 9, 16, 1, 2, 3, 0, time.UTC)
	store := Store{Base: t.TempDir(), Now: func() time.Time { return now }}
	liveStart, alive, err := processIdentity(os.Getpid())
	if err != nil || !alive {
		t.Fatalf("current process identity = %q, %v, %v", liveStart, alive, err)
	}
	live := Identity{Session: "live", Buffer: "buffer", Generation: 1}
	dead := Identity{Session: "dead", Buffer: "buffer", Generation: 1}
	if err := store.RenewLease(live, Lease{PID: os.Getpid(), ProcessStart: liveStart}); err != nil {
		t.Fatal(err)
	}
	if err := store.RenewLease(dead, Lease{PID: 99999999, ProcessStart: "old-process"}); err != nil {
		t.Fatal(err)
	}
	store.Now = func() time.Time { return now.Add(48 * time.Hour) }
	removed, err := store.Prune(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if _, err := os.Stat(filepath.Join(store.Base, "sessions", "live", "buffer", "lease.json")); err != nil {
		t.Fatalf("live lease removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Base, "sessions", "dead", "buffer")); !os.IsNotExist(err) {
		t.Fatalf("dead session remains: %v", err)
	}
}

func TestPruneKeepsExpiredSessionWithoutValidLeaseIdentity(t *testing.T) {
	now := time.Date(2026, 9, 16, 1, 2, 3, 0, time.UTC)
	store := Store{Base: t.TempDir(), Now: func() time.Time { return now }}
	directory := filepath.Join(store.Base, "sessions", "unknown", "buffer")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	old := now.Add(-48 * time.Hour)
	if err := os.Chtimes(directory, old, old); err != nil {
		t.Fatal(err)
	}
	removed, err := store.Prune(24 * time.Hour)
	if err != nil || removed != 0 {
		t.Fatalf("prune unknown identity = %d, %v", removed, err)
	}
	if _, err := os.Stat(directory); err != nil {
		t.Fatalf("session without proven-dead identity was removed: %v", err)
	}
}

func stringsTrimExtension(path string) string {
	return path[:len(path)-len(filepath.Ext(path))]
}
