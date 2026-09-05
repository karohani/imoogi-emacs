package registry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_MissingFileCreatesEmptyRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load on missing file returned error: %v", err)
	}
	if got := len(reg.All()); got != 0 {
		t.Fatalf("expected 0 entries on a fresh registry, got %d", got)
	}
	if _, ok := reg.Lookup(1); ok {
		t.Fatalf("Lookup on a fresh registry unexpectedly found an entry")
	}
}

func TestLoad_CorruptFileReturnsDistinctError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatalf("Load on an unparseable file returned nil error")
	}
	var corrupt *CorruptError
	if !errors.As(err, &corrupt) {
		t.Fatalf("Load on an unparseable file returned %T, want *CorruptError", err)
	}

	// A corrupt file must be distinguishable from an absent one — the two
	// error paths must never collapse into the same signal (acceptance.md
	// §D.7: "must not silently start a fresh registry").
	missingPath := filepath.Join(dir, "does-not-exist.json")
	_, missingErr := Load(missingPath)
	if missingErr != nil {
		t.Fatalf("Load on a missing file must not error, got: %v", missingErr)
	}
}

func TestPut_AddsAndOverwritesByNoteID(t *testing.T) {
	dir := t.TempDir()
	reg, err := Load(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	reg.Put(Entry{NoteID: 1, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Default", ContentHash: "h1"})
	if e, ok := reg.Lookup(1); !ok || e.SourcePath != "a.org" {
		t.Fatalf("Lookup(1) = %+v, %v; want a.org entry", e, ok)
	}

	reg.Put(Entry{NoteID: 1, SourcePath: "b.org", NoteType: "Basic", ResolvedDeck: "Default", ContentHash: "h2"})
	if e, ok := reg.Lookup(1); !ok || e.SourcePath != "b.org" || e.ContentHash != "h2" {
		t.Fatalf("Lookup(1) after overwrite = %+v, %v; want b.org/h2", e, ok)
	}
	if got := len(reg.All()); got != 1 {
		t.Fatalf("expected 1 entry after overwrite (not 2), got %d", got)
	}
}

func TestRemove_DropsEntry(t *testing.T) {
	dir := t.TempDir()
	reg, err := Load(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	reg.Put(Entry{NoteID: 5, SourcePath: "x.org", NoteType: "Cloze", ResolvedDeck: "Deck", ContentHash: "h"})
	reg.Remove(5)
	if _, ok := reg.Lookup(5); ok {
		t.Fatalf("Lookup(5) found an entry after Remove(5)")
	}
	if got := len(reg.All()); got != 0 {
		t.Fatalf("expected 0 entries after Remove, got %d", got)
	}
}

func TestAll_SortedByNoteIDForDeterministicOutput(t *testing.T) {
	dir := t.TempDir()
	reg, err := Load(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	reg.Put(Entry{NoteID: 30, SourcePath: "c.org"})
	reg.Put(Entry{NoteID: 10, SourcePath: "a.org"})
	reg.Put(Entry{NoteID: 20, SourcePath: "b.org"})

	all := reg.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(all))
	}
	for i, want := range []int{10, 20, 30} {
		if all[i].NoteID != want {
			t.Fatalf("All()[%d].NoteID = %d, want %d (want ascending NoteID order)", i, all[i].NoteID, want)
		}
	}
}

func TestSave_RoundTripsThroughLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	reg.Put(Entry{NoteID: 1502298033753, SourcePath: "deck/topic.org", NoteType: "Basic", ResolvedDeck: "Geography::Europe", ContentHash: "abc123"})
	reg.Put(Entry{NoteID: 42, SourcePath: "other.org", NoteType: "Cloze", ResolvedDeck: "Default", ContentHash: "def456"})

	if err := reg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload after Save: %v", err)
	}
	e, ok := reloaded.Lookup(1502298033753)
	if !ok {
		t.Fatalf("reloaded registry missing note 1502298033753")
	}
	if e.SourcePath != "deck/topic.org" || e.NoteType != "Basic" || e.ResolvedDeck != "Geography::Europe" || e.ContentHash != "abc123" {
		t.Fatalf("reloaded entry = %+v, fields did not round-trip", e)
	}
	if got := len(reloaded.All()); got != 2 {
		t.Fatalf("reloaded registry has %d entries, want 2", got)
	}
}

// TestSave_UsesTempFileAndRename is the RED-captured verification of the
// atomic-write mechanism plan.md §D and design.md §2.3 require: Save MUST
// write to a temp file and rename it into place, never write the target
// path directly. This is tested at the unit level per the milestone's
// instruction — by inspecting the write mechanism's artifacts rather than
// by killing a process mid-write.
func TestSave_UsesTempFileAndRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	reg.Put(Entry{NoteID: 1, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Default", ContentHash: "h"})

	if err := reg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 1 || names[0] != "registry.json" {
		t.Fatalf("directory contents after Save = %v; want exactly [registry.json] (no leftover temp file)", names)
	}
}

// TestLoad_IgnoresLeftoverTempFileFromInterruptedWrite is the RED-captured
// crash-survival test: a temp file left behind by a Save that crashed
// BEFORE the rename step must never be picked up as the registry, and the
// pre-existing good file must still be read correctly. This is the unit-
// level proxy for "a crash mid-write cannot leave a torn file" — the
// rename is the atomicity boundary, so a partial write that never reached
// rename can only ever leave a stray temp file, never a corrupted target.
func TestLoad_IgnoresLeftoverTempFileFromInterruptedWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	reg.Put(Entry{NoteID: 7, SourcePath: "good.org", NoteType: "Basic", ResolvedDeck: "Default", ContentHash: "good-hash"})
	if err := reg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Simulate a Save that crashed after CreateTemp+Write but before
	// Rename: a stray temp file sitting next to the already-good target.
	strayTemp := filepath.Join(dir, ".registry-crashed.tmp")
	if err := os.WriteFile(strayTemp, []byte("{ garbage, not even valid json"), 0o644); err != nil {
		t.Fatalf("setup stray temp file: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load must ignore the stray temp file and read the good target, got error: %v", err)
	}
	e, ok := reloaded.Lookup(7)
	if !ok || e.SourcePath != "good.org" {
		t.Fatalf("reloaded registry = %+v; the pre-existing good file must survive an interrupted sibling write", reloaded.All())
	}

	// The target file's own bytes must be exactly what Save wrote — the
	// stray temp file must not have overwritten or been merged into it.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}
	if strings.Contains(string(raw), "garbage") {
		t.Fatalf("target registry file was corrupted by the stray temp file's content")
	}
}

func TestSave_FailsWhenTargetDirectoryDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing-subdir", "registry.json")

	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	reg.Put(Entry{NoteID: 1, SourcePath: "a.org"})

	if err := reg.Save(); err == nil {
		t.Fatalf("Save into a nonexistent directory returned nil error")
	}
}

func TestSave_FailsWhenTargetPathIsADirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// A non-empty directory at the target path makes the final rename fail
	// (POSIX: cannot rename a file over a non-empty directory), exercising
	// Save's rename-error branch without needing to kill a process mid-write.
	if err := os.WriteFile(filepath.Join(path, "occupied"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	reg := &Registry{path: path, entries: map[int]Entry{1: {NoteID: 1}}}
	if err := reg.Save(); err == nil {
		t.Fatalf("Save onto an occupied directory path returned nil error")
	}

	// The temp file created before the failed rename must be cleaned up —
	// Save must not leak temp files on a failure path.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "registry.json" {
			t.Fatalf("leftover temp file after failed Save: %s", e.Name())
		}
	}
}

func TestLoad_UnreadablePathIsReportedAsCorrupt(t *testing.T) {
	dir := t.TempDir()
	// A directory where a file is expected forces a read failure distinct
	// from "does not exist" (state_unreadable per plan.md D-5).
	path := filepath.Join(dir, "registry.json")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatalf("Load on a directory (not a file) returned nil error")
	}
	var corrupt *CorruptError
	if !errors.As(err, &corrupt) {
		t.Fatalf("Load on an unreadable path returned %T, want *CorruptError", err)
	}
}
