package planner

import (
	"context"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// Two headings carrying the same ANKI_NOTE_ID — the shape a copy-pasted
// heading produces. Observed before the guard: both were dispatched to the
// one note, the later heading's content silently replaced the earlier one's,
// the run reported zero errors, and because the registry holds one hash for
// two differing headings every later run re-issued an update (REQ-011
// violated permanently). The identifier is the note's identity, so a
// duplicated identifier is not two notes but one note claimed twice — the
// only safe move is to write nothing for either claimant and say so.

func TestRun_DuplicateNoteID_FailsBothAndWritesNothing(t *testing.T) {
	reg := newTestRegistry(t)
	oldHash := renderAndHash(t, "Basic", "원본 카드", "원본 내용.", "DeckA", nil)
	reg.Put(registry.Entry{NoteID: 7, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "DeckA", ContentHash: oldHash})
	client := newFakeClient()
	client.seedNote(7, "Basic", map[string]string{"Front": "원본 카드", "Back": "원본 내용."}, nil, []int{70})

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: intPtr(7), NoteType: "Basic", SourcePath: "a.org", Deck: strPtr("DeckA"), Title: "원본 카드", Body: "원본 내용."},
			{Key: "a.org::1", NoteID: intPtr(7), NoteType: "Basic", SourcePath: "a.org", Deck: strPtr("DeckA"), Title: "복사한 카드", Body: "복사본 내용."},
		},
		Census: []protocol.CensusEntry{{NoteID: 7, SourcePath: "a.org"}, {NoteID: 7, SourcePath: "a.org"}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(results) != 2 {
		t.Fatalf("results = %+v, want one per claimant", results)
	}
	for _, r := range results {
		if r.Action != protocol.ActionFailed {
			t.Errorf("%s: action = %q, want %q", *r.Key, r.Action, protocol.ActionFailed)
		}
	}
	if len(errs) != 2 {
		t.Fatalf("errs = %+v, want one per claimant", errs)
	}
	for _, e := range errs {
		if e.Code != protocol.CodeNoteIDDuplicated {
			t.Errorf("code = %q, want %q", e.Code, protocol.CodeNoteIDDuplicated)
		}
		if e.Key == nil {
			t.Errorf("error carries no key; the user must learn WHICH headings collide")
		}
	}
	if n := len(client.updateFieldsCalls) + len(client.addCalls); n != 0 {
		t.Errorf("AnkiConnect writes = %d, want 0 — a duplicated identifier must write nothing", n)
	}
	if got, _ := reg.Lookup(7); got.ContentHash != oldHash {
		t.Errorf("registry hash advanced to %q; it must stay at the pre-run value so the collision keeps being reported", got.ContentHash)
	}
	if len(client.deleteCalls) != 0 {
		t.Errorf("deleteCalls = %v, want none — the note is still claimed, just ambiguously", client.deleteCalls)
	}
}

func TestRun_DuplicateNoteID_DoesNotBlockUnrelatedEntries(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: intPtr(7), NoteType: "Basic", SourcePath: "a.org", Title: "A", Body: "a"},
			{Key: "a.org::1", NoteID: intPtr(7), NoteType: "Basic", SourcePath: "a.org", Title: "A copy", Body: "a2"},
			{Key: "b.org::0", NoteID: nil, NoteType: "Basic", SourcePath: "b.org", Title: "Fresh", Body: "new"},
		},
	}

	results, _ := Run(context.Background(), req, reg, client)

	var added int
	for _, r := range results {
		if r.Action == protocol.ActionAdded {
			added++
		}
	}
	if added != 1 {
		t.Errorf("added = %d, want 1 — the unrelated heading must still sync", added)
	}
	if len(client.addCalls) != 1 || client.addCalls[0].fields["Front"] == "" {
		t.Errorf("addCalls = %+v, want exactly the fresh heading", client.addCalls)
	}
}
