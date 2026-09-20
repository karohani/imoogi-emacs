package orgdoc

import (
	"errors"
	"strings"
	"testing"
)

// The card's byte-invariance requirement, made mechanical: a Cloze entry with
// no EXTRA block must render the SAME Text bytes before and after the split
// lands. Anything else re-hashes every existing Cloze note and produces a
// sync-wide `updated` storm that carries no content change.
func TestRender_Cloze_WithoutExtraBlock_TextIsByteInvariant(t *testing.T) {
	const want = "<p>Geography</p>\n<p>\nThe capital of France is {{c1::Paris}}.</p>\n<ul>\n<li>Note one</li>\n<li>Note two</li>\n</ul>\n"
	fields, err := Render(NoteTypeCloze, "Geography",
		"The capital of France is {{c1::Paris}}.\n\n- Note one\n- Note two")
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
	if fields["Text"] != want {
		t.Errorf("Text bytes changed.\n got: %q\nwant: %q", fields["Text"], want)
	}
}

// The field is emitted even when empty. Omitting it when absent would look
// tidier, but then DELETING an EXTRA block from a card that had one would send
// an update whose payload never mentions the field — and AnkiConnect leaves a
// field it is not given, so the stale extra would survive forever.
func TestRender_Cloze_WithoutExtraBlock_StillEmitsEmptyBackExtra(t *testing.T) {
	fields, err := Render(NoteTypeCloze, "Geography", "Paris is {{c1::the capital}}.")
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
	got, ok := fields["Back Extra"]
	if !ok {
		t.Fatalf("Back Extra key absent; fields=%v", fields)
	}
	if got != "" {
		t.Errorf("Back Extra = %q, want the empty string", got)
	}
}

func TestRender_Cloze_ExtraBlockMovesOutOfTextIntoBackExtra(t *testing.T) {
	fields, err := Render(NoteTypeCloze, "Geography",
		"The capital of France is {{c1::Paris}}.\n#+BEGIN_EXTRA\nPopulation about 2.1 million.\n#+END_EXTRA")
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
	if !strings.Contains(fields["Text"], "{{c1::Paris}}") {
		t.Errorf("Text = %q, want it to keep the cloze marker", fields["Text"])
	}
	if strings.Contains(fields["Text"], "2.1 million") {
		t.Errorf("Text = %q, want the EXTRA content removed from it", fields["Text"])
	}
	if !strings.Contains(fields["Back Extra"], "2.1 million") {
		t.Errorf("Back Extra = %q, want it to carry the EXTRA content", fields["Back Extra"])
	}
	if strings.Contains(fields["Back Extra"], "extra-block") {
		t.Errorf("Back Extra = %q, want the wrapper div gone with the block markers", fields["Back Extra"])
	}
}

// Org accepts its keywords in either case, so a body written in lowercase is
// the same document — the split must not depend on how the author shouts.
func TestRender_Cloze_LowercaseExtraBlockIsRecognized(t *testing.T) {
	fields, err := Render(NoteTypeCloze, "Geography",
		"Paris is {{c1::the capital}}.\n#+begin_extra\nA side note.\n#+end_extra")
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
	if !strings.Contains(fields["Back Extra"], "A side note") {
		t.Errorf("Back Extra = %q, want the lowercase block recognized", fields["Back Extra"])
	}
}

func TestRender_Cloze_MultipleExtraBlocksConcatenate(t *testing.T) {
	fields, err := Render(NoteTypeCloze, "Geography",
		"Paris is {{c1::the capital}}.\n#+BEGIN_EXTRA\nFirst note.\n#+END_EXTRA\nMore body.\n#+BEGIN_EXTRA\nSecond note.\n#+END_EXTRA")
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
	for _, want := range []string{"First note", "Second note"} {
		if !strings.Contains(fields["Back Extra"], want) {
			t.Errorf("Back Extra = %q, want it to contain %q", fields["Back Extra"], want)
		}
	}
	if !strings.Contains(fields["Text"], "More body") {
		t.Errorf("Text = %q, want the body between the two blocks preserved", fields["Text"])
	}
}

// Anki's cloze template reads {{cloze:Text}} only, so a marker that lives
// solely inside the EXTRA content produces a note with no cloze deletion and
// Anki refuses it. The gate therefore runs on the Text fragment, not the
// whole body.
func TestRender_Cloze_MarkerOnlyInsideExtra_ReportsMissingMarker(t *testing.T) {
	_, err := Render(NoteTypeCloze, "Geography",
		"Plain body.\n#+BEGIN_EXTRA\nA hidden {{c1::marker}}.\n#+END_EXTRA")
	var missing *ClozeMarkerMissingError
	if !errors.As(err, &missing) {
		t.Fatalf("err = %v, want *ClozeMarkerMissingError", err)
	}
}

// imoogi-Basic has no Back Extra field, so emitting that key there would fail
// the planner's field resolution. Basic keeps the block inline in Back, which
// is what it already does.
func TestRender_Basic_ExtraBlockStaysInlineAndEmitsNoBackExtra(t *testing.T) {
	const want = "<p>A body.</p>\n<div class=\"extra-block\">\n<p>Side note.</p>\n</div>\n"
	fields, err := Render(NoteTypeBasic, "Q", "A body.\n#+BEGIN_EXTRA\nSide note.\n#+END_EXTRA")
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
	if fields["Back"] != want {
		t.Errorf("Back bytes changed.\n got: %q\nwant: %q", fields["Back"], want)
	}
	if _, ok := fields["Back Extra"]; ok {
		t.Errorf("Basic must not emit a Back Extra field, got fields=%v", fields)
	}
}
