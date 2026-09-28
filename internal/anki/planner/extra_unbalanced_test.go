package planner

import (
	"context"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// SPEC-ANKICARD-005 — an entry whose supplementary split leaves an unpaired
// EXTRA marker is skipped with extra_block_unbalanced, on both call sites.

// AC-AKX-004 — the check runs before every other render-time diagnostic, and
// after the card-option validation gate. Rows (a) and (b) carry bodies that
// produce a competing code without the check, so they pass only when it runs
// FIRST.
func TestExtraBlockUnbalancedPrecedence(t *testing.T) {
	t.Run("a_before_the_cloze_marker_gate", func(t *testing.T) {
		// Without the check: cloze_marker_missing.
		entry := protocol.Entry{
			Key: "a.org::0", NoteType: "Cloze", SourcePath: "a.org",
			Title: "no marker here", Body: "plain text\n#+END_EXTRA\n",
		}
		results, errs, client := runOne(t, entry)
		expectSkippedWithCode(t, results, errs, protocol.CodeExtraBlockUnbalanced)
		if len(client.addCalls) != 0 {
			t.Errorf("addCalls = %d, want 0", len(client.addCalls))
		}
	})

	t.Run("b_before_composition", func(t *testing.T) {
		// Without the check: multiline_answer_missing.
		entry := multilineEntry(strPtr("->"), nil)
		entry.Body = "#+BEGIN_EXTRA\nnote\n"
		results, errs, _ := runOne(t, entry)
		expectSkippedWithCode(t, results, errs, protocol.CodeExtraBlockUnbalanced)
	})

	t.Run("c_the_validation_gate_still_wins", func(t *testing.T) {
		entry := multilineEntry(strPtr("sideways"), nil)
		entry.Body = "- Tokyo\n#+END_EXTRA\n- Osaka\n"
		results, errs, _ := runOne(t, entry)
		expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionInvalid)
	})
}

// AC-AKX-005 — the rejected entry is skipped carrying its identifier, nothing
// is written for it, and the run goes on to the next entry.
func TestExtraBlockUnbalancedSkipsOnlyThatEntry(t *testing.T) {
	const priorTitle, priorBody = "The capital of France is {{c1::Paris}}", "Since 987."
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{
		NoteID: 1001, SourcePath: "a.org", NoteType: "Cloze", ResolvedDeck: "Inbox",
		ContentHash: renderAndHash(t, "Cloze", priorTitle, priorBody, "Inbox", nil),
	})
	client := newFakeClient()
	client.decks["Inbox"] = true
	client.seedNote(1001, "Cloze", renderedFields(t, "Cloze", priorTitle, priorBody), nil, []int{1001001})
	before, _ := reg.Lookup(1001)

	req := protocol.Request{
		Config: baseConfig(),
		Census: []protocol.CensusEntry{{NoteID: 1001, SourcePath: "a.org"}},
		Entries: []protocol.Entry{
			{
				Key: "a.org::0", NoteID: intPtr(1001), NoteType: "Cloze", SourcePath: "a.org",
				Title: priorTitle, Body: "Since 987.\n#+END_EXTRA\n",
			},
			{
				Key: "a.org::1", NoteType: "Cloze", SourcePath: "a.org",
				Title: "The capital of Italy is {{c1::Rome}}", Body: "Since 1871.",
			},
		},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Action != protocol.ActionSkipped {
		t.Errorf("results[0].action = %q, want skipped", results[0].Action)
	}
	if results[0].NoteID == nil || *results[0].NoteID != 1001 {
		t.Errorf("results[0].note_id = %v, want the existing 1001", results[0].NoteID)
	}
	if results[1].Action != protocol.ActionAdded {
		t.Errorf("results[1].action = %q, want added", results[1].Action)
	}
	if len(errs) != 1 || errs[0].Code != protocol.CodeExtraBlockUnbalanced {
		t.Fatalf("errs = %+v, want exactly one %s", errs, protocol.CodeExtraBlockUnbalanced)
	}
	if errs[0].Key == nil || *errs[0].Key != "a.org::0" {
		t.Errorf("errs[0].key = %v, want a.org::0", errs[0].Key)
	}
	for _, action := range client.writeSequence {
		if action == "updateNoteFields" || action == "deleteNotes" {
			t.Errorf("writeSequence = %v, want no %s — the skipped note is left as it was", client.writeSequence, action)
		}
	}
	if len(client.addCalls) != 1 {
		t.Errorf("addCalls = %d, want 1 — the second entry only", len(client.addCalls))
	}
	if after, _ := reg.Lookup(1001); after != before {
		t.Errorf("registry entry changed:\n before = %+v\n  after = %+v", before, after)
	}
}

// AC-AKX-006 (a) — the migration path reports the same skip and leaves the
// original note and its registry record exactly as they were.
func TestMigrate_UnpairedExtraMarker_IsSkippedAndLeavesTheOriginal(t *testing.T) {
	const title = "The capital of France is {{c1::Paris}}"
	client, reg, entry := migrateFixture(t, 41, "a.org::0", title, "Since 987.")
	reg.Put(registry.Entry{
		NoteID: 41, SourcePath: "a.org", NoteType: "Cloze",
		ResolvedDeck: "Inbox", ContentHash: "recorded-under-cloze",
	})
	entry.NoteType = "Cloze" // recorded Cloze too, so this IS a candidate...
	entry.Body = "Since 987.\n#+END_EXTRA\n"

	before := reg.All()
	results, errs := Migrate(context.Background(), migrateRequest(entry), reg, client, false)

	if len(results) != 1 || results[0].Action != protocol.ActionSkipped {
		t.Fatalf("results = %+v, want exactly one skipped entry", results)
	}
	if len(errs) != 1 || errs[0].Code != protocol.CodeExtraBlockUnbalanced {
		t.Fatalf("errs = %+v, want exactly one %s", errs, protocol.CodeExtraBlockUnbalanced)
	}
	if len(client.writeSequence) != 0 {
		t.Errorf("writeSequence = %v, want none — the original note stays", client.writeSequence)
	}
	if after := reg.All(); len(after) != len(before) || after[0] != before[0] {
		t.Errorf("registry changed:\n before = %+v\n  after = %+v", before, after)
	}
}
