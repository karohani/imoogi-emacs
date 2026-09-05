// Package registry persists the derived per-sync-root state design.md §2.3
// defines: one JSON file per sync root, holding an array of entries keyed
// by Anki note identifier. Identity (ANKI_NOTE_ID) lives in the user's Org
// file; everything recomputable lives here (plan.md D-3).
//
// The registry is rewritten wholesale on every Save — atomic replace via a
// temp file plus rename, never a partial in-place patch — so a crash
// mid-write cannot leave a torn file that silently drops entries
// (design.md §2.3, plan.md §D module table).
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Entry is one registry record (design.md §2.3). ContentHash is produced by
// internal/hashing.Hash over the note type, rendered fields, resolved deck,
// and sorted tag set.
type Entry struct {
	NoteID       int    `json:"note_id"`
	SourcePath   string `json:"source_path"`
	NoteType     string `json:"note_type"`
	ResolvedDeck string `json:"resolved_deck"`
	ContentHash  string `json:"content_hash"`
}

// CorruptError distinguishes a registry file that EXISTS but could not be
// read or parsed from one that is simply absent (plan.md D-5's
// state_unreadable). Load returns this — never a bare error and never a
// silent empty registry — so callers cannot mistake corruption for "first
// run" and orphan every existing note by starting over
// (acceptance.md §D.7).
type CorruptError struct {
	Path string
	Err  error
}

func (e *CorruptError) Error() string {
	return fmt.Sprintf("registry: %s: state unreadable: %v", e.Path, e.Err)
}

func (e *CorruptError) Unwrap() error { return e.Err }

// Registry is the in-memory view of one sync root's persisted state, keyed
// by note identifier for O(1) lookup (design.md §3 step 9's per-entry
// registry lookup).
type Registry struct {
	path    string
	entries map[int]Entry
}

// Load reads the registry file at path. A missing file is NOT an error —
// it returns an empty Registry, matching "first run has no registry"
// (design.md §2.3, acceptance.md §D.7: "must be created; must not be
// treated as an error; must not delete anything, since its candidate set
// is empty"). A file that exists but cannot be read or parsed returns
// *CorruptError; the caller MUST report state_unreadable and MUST NOT
// proceed as though the registry were empty (acceptance.md §D.7).
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Registry{path: path, entries: map[int]Entry{}}, nil
		}
		return nil, &CorruptError{Path: path, Err: err}
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, &CorruptError{Path: path, Err: err}
	}

	m := make(map[int]Entry, len(entries))
	for _, e := range entries {
		m[e.NoteID] = e
	}
	return &Registry{path: path, entries: m}, nil
}

// Lookup returns the entry recorded for noteID, if any (design.md §3 step
// 9's registry lookup).
func (r *Registry) Lookup(noteID int) (Entry, bool) {
	e, ok := r.entries[noteID]
	return e, ok
}

// All enumerates every entry, sorted by NoteID for deterministic output
// (the planner's candidate computation, design.md §3 step 10, needs the
// full identifier set; a stable order keeps Save's on-disk output and any
// derived report reproducible across runs with identical state).
func (r *Registry) All() []Entry {
	out := make([]Entry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NoteID < out[j].NoteID })
	return out
}

// Put adds a new entry or overwrites the existing one for e.NoteID —
// design.md §2.3's "add/update" operation. Both an add (REQ-008) and an
// update's refreshed note type/deck/hash use this single method; there is
// no behavioral difference between the two at the registry layer.
func (r *Registry) Put(e Entry) {
	r.entries[e.NoteID] = e
}

// Remove drops the entry for noteID, if present — used when a confirmed
// orphan candidate is deleted (design.md §3 step 11, REQ-014).
func (r *Registry) Remove(noteID int) {
	delete(r.entries, noteID)
}

// Save performs an atomic replace of the registry file: marshal every
// entry, write it to a temp file in the same directory, then rename the
// temp file over the target path. Rename is the atomicity boundary — a
// crash at any point before it leaves, at worst, a stray temp file next to
// an untouched, still-valid target; a crash can never produce a torn
// target file, because the target is never opened for writing directly
// (design.md §2.3, plan.md §D module table).
func (r *Registry) Save() error {
	entries := r.All()
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("registry: marshal: %w", err)
	}

	dir := filepath.Dir(r.path)
	tmp, err := os.CreateTemp(dir, ".registry-*.tmp")
	if err != nil {
		return fmt.Errorf("registry: create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	renamed := false
	defer func() {
		if !renamed {
			// Cleanup of a temp file this function is abandoning: the
			// caller is already being handed the error that caused the
			// abandonment, and a stray temp file next to an untouched,
			// still-valid target is explicitly the tolerated worst case
			// of this function's atomicity contract (see the doc comment
			// above). Nothing actionable is discarded here.
			_ = os.Remove(tmpPath)
		}
	}()

	// The Close on each of the two error paths below is best-effort: the
	// Write or Sync error is the one that describes what went wrong, and
	// returning a Close error in its place would MASK the real cause. The
	// success-path Close, by contrast, is checked and returned (below) —
	// a Close failure there can mean the data never reached the disk, and
	// renaming an unflushed temp file over a valid registry would defeat
	// the atomic replace this whole function exists to provide.
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("registry: write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("registry: sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("registry: close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, r.path); err != nil {
		return fmt.Errorf("registry: rename temp file into place: %w", err)
	}
	renamed = true
	return nil
}
