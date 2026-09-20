package planner

import (
	"context"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// The renderer emits one name for the cloze note's supplementary field, but
// the two cloze-style models in play do not agree on what that field is
// called: imoogi-Cloze carries "Back Extra" while a stock "Cloze" profile can
// carry plain "Extra". Field resolution matches by exact name, so without an
// alias a hand-written stock-Cloze heading — which this project explicitly
// keeps synchronizable — would start failing with note_field_missing the
// moment the renderer began populating that field at all.
func TestRun_AddBranch_StockClozeWithExtraField_ResolvesTheRenderedBackExtra(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.modelFields = map[string][]string{"Cloze": {"Text", "Extra"}}

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{{
			Key: "a.org::0", NoteType: "Cloze", SourcePath: "a.org",
			Title: "Geography",
			Body:  "Paris is {{c1::the capital}}.\n#+BEGIN_EXTRA\nA side note.\n#+END_EXTRA",
		}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if results[0].Action != protocol.ActionAdded {
		t.Fatalf("action = %q, want %q", results[0].Action, protocol.ActionAdded)
	}
	got := client.addCalls[0].fields
	if _, ok := got["Back Extra"]; ok {
		t.Errorf("add sent %q; this model has no such field", "Back Extra")
	}
	if got["Extra"] == "" {
		t.Errorf("add fields = %v, want the model's own %q populated", got, "Extra")
	}
}

// The alias must not turn a genuinely missing field into a silent success:
// a model carrying neither name still fails loudly.
func TestRun_AddBranch_ClozeModelWithNoExtraField_StillReportsFieldMissing(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.modelFields = map[string][]string{"Cloze": {"Text"}}

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{{
			Key: "a.org::0", NoteType: "Cloze", SourcePath: "a.org",
			Title: "Geography", Body: "Paris is {{c1::the capital}}.",
		}},
	}

	_, errs := Run(context.Background(), req, reg, client)

	if len(errs) != 1 || errs[0].Code != protocol.CodeNoteFieldMissing {
		t.Fatalf("errs = %+v, want one %s", errs, protocol.CodeNoteFieldMissing)
	}
}
