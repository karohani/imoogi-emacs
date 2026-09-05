package planner

import (
	"context"
	"errors"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// A real Anki profile's field names are whatever the user made them.
// The observed case: a customized "Basic" note type whose fields are
// lowercase front/back, against an orgdoc.Render that emits Front/Back.
//
// AnkiConnect's addNote matches field names case-insensitively, so adds
// looked fine; updateNoteFields does not — it accepts the unknown names,
// answers error:null, and changes nothing. The run reported action
// "updated" with zero errors while Anki kept the old text, and the
// registry's content hash advanced so the next run skipped the entry.
// Org and Anki then diverged permanently with no signal anywhere.
//
// These tests pin the fix: field names are resolved against the note
// type's real fields before dispatch, and a name that cannot be resolved
// is an error rather than a silent no-op.

func TestRun_AddBranch_ModelWithLowercaseFields_SendsTheModelsOwnNames(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.modelFields = map[string][]string{"Basic": {"front", "back"}}

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{{
			Key: "a.org::0", NoteType: "Basic", SourcePath: "a.org",
			Title: "Capital of France", Body: "Paris.",
		}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if results[0].Action != protocol.ActionAdded {
		t.Fatalf("action = %q, want %q", results[0].Action, protocol.ActionAdded)
	}
	if len(client.addCalls) != 1 {
		t.Fatalf("addCalls = %d, want 1", len(client.addCalls))
	}
	got := client.addCalls[0].fields
	if _, ok := got["Front"]; ok {
		t.Errorf("add sent capitalized %q; the model has no such field", "Front")
	}
	if got["front"] == "" {
		t.Errorf("add fields = %v, want the model's own lowercase %q populated", got, "front")
	}
	if got["back"] == "" {
		t.Errorf("add fields = %v, want the model's own lowercase %q populated", got, "back")
	}
}

func TestRun_UpdateBranch_ModelWithLowercaseFields_SendsTheModelsOwnNames(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.modelFields = map[string][]string{"Basic": {"front", "back"}}
	client.seedNote(41, "Basic", map[string]string{"front": "<p>Capital of France</p>", "back": "<p>Paris.</p>"}, nil, []int{1})

	oldHash := renderAndHash(t, "Basic", "Capital of France", "Paris.", "DeckA", nil)
	reg.Put(registry.Entry{NoteID: 41, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "DeckA", ContentHash: oldHash})

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{{
			Key: "a.org::0", NoteID: intPtr(41), NoteType: "Basic", SourcePath: "a.org",
			Deck: strPtr("DeckA"), Title: "Capital of France", Body: "Paris, on the river Seine.",
		}},
		Census: []protocol.CensusEntry{{NoteID: 41, SourcePath: "a.org"}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if results[0].Action != protocol.ActionUpdated {
		t.Fatalf("action = %q, want %q", results[0].Action, protocol.ActionUpdated)
	}
	if len(client.updateFieldsCalls) != 1 {
		t.Fatalf("updateFieldsCalls = %d, want 1", len(client.updateFieldsCalls))
	}
	got := client.updateFieldsCalls[0].fields
	if _, ok := got["Back"]; ok {
		t.Errorf("update sent capitalized %q; AnkiConnect would accept it and change nothing", "Back")
	}
	if got["back"] == "" {
		t.Errorf("update fields = %v, want the model's own lowercase %q populated", got, "back")
	}
}

func TestRun_ModelMissingARenderedField_FailsLoudlyInsteadOfSilently(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	// A "Basic" whose second field was renamed to something orgdoc never emits.
	client.modelFields = map[string][]string{"Basic": {"front", "answer"}}

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{{
			Key: "a.org::0", NoteType: "Basic", SourcePath: "a.org",
			Title: "Capital of France", Body: "Paris.",
		}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) == 0 {
		t.Fatalf("errs = none, want a note_field_missing error rather than a silent partial write")
	}
	if errs[0].Code != protocol.CodeNoteFieldMissing {
		t.Errorf("code = %q, want %q", errs[0].Code, protocol.CodeNoteFieldMissing)
	}
	if results[0].Action != protocol.ActionFailed {
		t.Errorf("action = %q, want %q", results[0].Action, protocol.ActionFailed)
	}
	if len(client.addCalls) != 0 {
		t.Errorf("addCalls = %d, want 0 — nothing may be written when a field cannot be resolved", len(client.addCalls))
	}
}

func TestRun_ModelFieldLookupFails_ReportsAnkiConnectError(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.modelFieldsErr = errors.New("boom")

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{{
			Key: "a.org::0", NoteType: "Basic", SourcePath: "a.org",
			Title: "Capital of France", Body: "Paris.",
		}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) == 0 {
		t.Fatalf("errs = none, want the lookup failure surfaced")
	}
	if errs[0].Code != protocol.CodeAnkiConnectError {
		t.Errorf("code = %q, want %q", errs[0].Code, protocol.CodeAnkiConnectError)
	}
	if results[0].Action != protocol.ActionFailed {
		t.Errorf("action = %q, want %q", results[0].Action, protocol.ActionFailed)
	}
}
