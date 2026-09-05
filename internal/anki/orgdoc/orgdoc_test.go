package orgdoc

import (
	"errors"
	"strings"
	"testing"
)

func TestRender_Basic_TitleAndBodyMapToFrontAndBack(t *testing.T) {
	fields, err := Render(NoteTypeBasic, "Capital of France", "Paris.")
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
	if !strings.Contains(fields["Front"], "Capital of France") {
		t.Errorf("Front = %q, want it to contain %q", fields["Front"], "Capital of France")
	}
	if !strings.Contains(fields["Back"], "Paris") {
		t.Errorf("Back = %q, want it to contain %q", fields["Back"], "Paris")
	}
	if _, ok := fields["Text"]; ok {
		t.Errorf("Basic rendering must not populate a Text field, got fields=%v", fields)
	}
}

func TestRender_Basic_EmptyBody_RendersEmptyFieldWithoutCrashing(t *testing.T) {
	fields, err := Render(NoteTypeBasic, "A heading with no body", "")
	if err != nil {
		t.Fatalf("Render returned unexpected error on empty body: %v", err)
	}
	if fields["Back"] != "" {
		t.Errorf("Back = %q, want empty string for an empty body", fields["Back"])
	}
	if !strings.Contains(fields["Front"], "A heading with no body") {
		t.Errorf("Front = %q, want it to contain the title", fields["Front"])
	}
}

// TestRender_Cloze_PreservesMarkerByteForByte is the highest-risk behavior in
// this milestone (Section D): go-org must not be the source of any
// reformatting here (research.md §2 confirms it treats the marker as literal
// inline text), but imoogi's OWN rendering path must also not touch it. This
// asserts byte-exact preservation, not mere substring containment, by
// checking the marker's exact byte sequence appears unchanged in Text.
func TestRender_Cloze_PreservesMarkerByteForByte(t *testing.T) {
	body := "The capital of France is {{c1::Paris}}."
	fields, err := Render(NoteTypeCloze, "Geography", body)
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
	marker := "{{c1::Paris}}"
	if !strings.Contains(fields["Text"], marker) {
		t.Fatalf("Text = %q, want it to contain the byte-exact marker %q", fields["Text"], marker)
	}
	if _, ok := fields["Front"]; ok {
		t.Errorf("Cloze rendering must not populate Front/Back fields, got fields=%v", fields)
	}
	if _, ok := fields["Back"]; ok {
		t.Errorf("Cloze rendering must not populate Front/Back fields, got fields=%v", fields)
	}
}

func TestRender_Cloze_PreservesMultipleMarkersByteForByte(t *testing.T) {
	body := "First {{c1::alpha}} and second {{c2::beta}} and third {{c3::gamma::hint}}."
	fields, err := Render(NoteTypeCloze, "Multi", body)
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
	for _, marker := range []string{"{{c1::alpha}}", "{{c2::beta}}", "{{c3::gamma::hint}}"} {
		if !strings.Contains(fields["Text"], marker) {
			t.Errorf("Text = %q, want it to contain byte-exact marker %q", fields["Text"], marker)
		}
	}
}

func TestRender_Cloze_MarkerInTitleIsSufficient(t *testing.T) {
	// The marker may appear in the title alone (title + body are combined into
	// one Text field, spec.md REQ-004), so a body-only presence check would be
	// wrong.
	fields, err := Render(NoteTypeCloze, "The capital is {{c1::Paris}}", "No marker here.")
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
	if !strings.Contains(fields["Text"], "{{c1::Paris}}") {
		t.Errorf("Text = %q, want it to contain the title's marker", fields["Text"])
	}
}

// TestRender_Cloze_MissingMarker_ReturnsTypedDiagnostic is the second
// highest-risk behavior in this milestone: REQ-019 requires a typed
// diagnostic (cloze_marker_missing), not a panic and not silently rendered
// garbage.
func TestRender_Cloze_MissingMarker_ReturnsTypedDiagnostic(t *testing.T) {
	fields, err := Render(NoteTypeCloze, "No markers anywhere", "Just plain prose, no braces at all.")
	if err == nil {
		t.Fatalf("Render returned no error for a Cloze entry with no {{cN:: marker; fields=%v", fields)
	}
	var missing *ClozeMarkerMissingError
	if !errors.As(err, &missing) {
		t.Fatalf("Render returned error %v (type %T), want a *ClozeMarkerMissingError", err, err)
	}
	if fields != nil {
		t.Errorf("Render returned non-nil fields alongside a missing-marker error: %v", fields)
	}
}

func TestRender_Cloze_MissingMarker_PartialLikeTextDoesNotFalselyMatch(t *testing.T) {
	// A brace pair that is not the {{cN:: shape must not be mistaken for a
	// cloze marker.
	_, err := Render(NoteTypeCloze, "Title", "Some {curly} text and {{not a marker}} too.")
	if err == nil {
		t.Fatalf("Render returned no error for text with braces but no {{cN:: marker")
	}
	var missing *ClozeMarkerMissingError
	if !errors.As(err, &missing) {
		t.Fatalf("Render returned error %v (type %T), want a *ClozeMarkerMissingError", err, err)
	}
}

func TestRender_UnknownNoteType_ReturnsError(t *testing.T) {
	_, err := Render("Reversed", "Title", "Body")
	if err == nil {
		t.Fatal("Render returned no error for an unrecognized note type")
	}
}

// TestClozeMarkerMissingError_MessageNamesTheProblem exercises the diagnostic
// text a caller surfaces to the user. REQ-019's value is not just that Render
// returns *some* typed error but that the error explains what is missing, so
// the message is asserted through the error the public API actually hands back
// rather than a hand-constructed value.
func TestClozeMarkerMissingError_MessageNamesTheProblem(t *testing.T) {
	_, err := Render(NoteTypeCloze, "No markers anywhere", "Just plain prose.")
	if err == nil {
		t.Fatal("Render returned no error for a Cloze entry with no {{cN:: marker")
	}
	msg := err.Error()
	if msg == "" {
		t.Fatal("ClozeMarkerMissingError.Error() returned an empty string; a diagnostic with no text tells the caller nothing")
	}
	// The two things a reader needs: which note type is at fault, and that the
	// missing thing is a cloze marker.
	for _, want := range []string{"cloze", "marker"} {
		if !strings.Contains(strings.ToLower(msg), want) {
			t.Errorf("Error() = %q, want it to mention %q", msg, want)
		}
	}
}

func TestCombineTitleAndBody(t *testing.T) {
	tests := []struct {
		name  string
		title string
		body  string
		want  string
	}{
		{
			name:  "empty body yields the title alone, with no trailing blank line",
			title: "A title",
			body:  "",
			want:  "A title",
		},
		{
			name:  "non-empty body is separated from the title by a blank line",
			title: "A title",
			body:  "A body.",
			want:  "A title\n\nA body.",
		},
		{
			name:  "empty title with a body keeps the separator so the body stays a paragraph",
			title: "",
			body:  "A body.",
			want:  "\n\nA body.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := combineTitleAndBody(tt.title, tt.body); got != tt.want {
				t.Errorf("combineTitleAndBody(%q, %q) = %q, want %q", tt.title, tt.body, got, tt.want)
			}
		})
	}
}

// TestRender_Cloze_EmptyBody_NoSpuriousBlankParagraph is the empty-body edge
// (acceptance.md §D.7) on the Cloze path: a title-only Cloze entry must render
// exactly as the title would on its own. Comparing against renderFragment(title)
// asserts the absence of the extra paragraph a naive "title + \n\n + body" join
// would introduce — a plain substring check on the marker would pass either way.
func TestRender_Cloze_EmptyBody_NoSpuriousBlankParagraph(t *testing.T) {
	const title = "The capital is {{c1::Paris}}"
	fields, err := Render(NoteTypeCloze, title, "")
	if err != nil {
		t.Fatalf("Render returned unexpected error on an empty Cloze body: %v", err)
	}
	if !strings.Contains(fields["Text"], "{{c1::Paris}}") {
		t.Errorf("Text = %q, want it to contain the title's marker", fields["Text"])
	}
	if want := renderFragment(title); fields["Text"] != want {
		t.Errorf("Text = %q, want %q (a title-only Cloze entry must render as the bare title)", fields["Text"], want)
	}
}

// TestRender_Basic_OversizedSingleLine_FallsBackToRawText covers renderFragment's
// go-org-error fallback. The path is reachable through a documented stdlib
// boundary rather than a synthetic one: go-org tokenizes with a default
// bufio.Scanner, whose 64KiB per-line ceiling makes any longer single line fail
// with bufio.ErrTooLong. The contract under test is that such a fragment is
// returned verbatim — content is never silently dropped and no error escapes to
// a caller that has no typed diagnostic for it.
func TestRender_Basic_OversizedSingleLine_FallsBackToRawText(t *testing.T) {
	body := strings.Repeat("a", 70*1024) // one line, past bufio.Scanner's 64KiB limit
	fields, err := Render(NoteTypeBasic, "Oversized", body)
	if err != nil {
		t.Fatalf("Render returned an error for an oversized body; the fallback must absorb it: %v", err)
	}
	if fields["Back"] != body {
		t.Errorf("Back length = %d, want the %d-byte body returned verbatim", len(fields["Back"]), len(body))
	}
}
