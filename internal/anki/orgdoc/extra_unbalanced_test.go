package orgdoc

import (
	"errors"
	"strings"
	"testing"
)

// SPEC-ANKICARD-005 — an EXTRA marker line the supplementary split leaves
// behind is reported rather than rendered. Every body below is the PRE-split
// body, so the whole pipeline runs: split, check, then whatever would have
// come next.

// unpairedExtraBodies are the shapes whose split leaves an EXTRA marker line in
// the remaining body (REQ-AKX-001) or an opening one in the extracted
// supplementary content (REQ-AKX-002). Each carries an answer list, so the
// multiline entry point would otherwise compose a card from it.
var unpairedExtraBodies = []struct {
	name string
	body string
}{
	{"a_stray_closing_marker_after_a_pair", "- A\n#+BEGIN_EXTRA\nnote\n#+END_EXTRA\n#+END_EXTRA\n- B\n"},
	{"a_lone_closing_marker", "- A\n#+END_EXTRA\n- B\n"},
	{"an_unterminated_opening_mid_body", "- A\n#+BEGIN_EXTRA\nnote\n- B\n"},
	// No trailing newline: the split pattern needs one after the opening
	// line, so this line is never consumed and must still be caught.
	{"an_opening_on_the_last_line_without_a_newline", "- A\n- B\n#+BEGIN_EXTRA"},
	// Balanced input, damaged output: the first opening pairs with the FIRST
	// closing, exactly as Org itself reads it.
	{"balanced_nested_blocks", "- A\n#+BEGIN_EXTRA\nouter\n#+BEGIN_EXTRA\ninner\n#+END_EXTRA\ntail\n#+END_EXTRA\n- B\n"},
	// The check reads the split's own grammar: indentation and case do not
	// hide a marker from it.
	{"an_indented_lowercase_closing_marker", "- A\n  #+end_extra\n- B\n"},
	// AC-AKX-002: the remaining body is clean, and the residue sits in the
	// extracted content instead.
	{"two_openings_one_closing_leave_it_in_the_extracted_content", "- A\n#+BEGIN_EXTRA\nfirst\n#+BEGIN_EXTRA\nsecond\n#+END_EXTRA\n- B\n"},
}

// AC-AKX-001, AC-AKX-002 — both entry points that perform the split reject
// every shape with the dedicated error and no fields.
func TestRenderRejectsUnpairedExtraMarker(t *testing.T) {
	for _, c := range unpairedExtraBodies {
		t.Run("cloze/"+c.name, func(t *testing.T) {
			fields, err := Render(NoteTypeCloze, "Q {{c1::x}}", c.body)
			expectExtraBlockUnbalanced(t, fields, err)
		})
		t.Run("multiline/"+c.name, func(t *testing.T) {
			fields, err := RenderWithOptions(NoteTypeCloze, "Q", c.body, CardOptions{Direction: DirectionRightward})
			expectExtraBlockUnbalanced(t, fields, err)
		})
	}

	t.Run("the_message_names_the_markers", func(t *testing.T) {
		msg := (&ExtraBlockUnbalancedError{}).Error()
		if !strings.Contains(msg, "#+BEGIN_EXTRA") || !strings.Contains(msg, "#+END_EXTRA") {
			t.Errorf("Error() = %q, want it to name both markers", msg)
		}
	})
}

func expectExtraBlockUnbalanced(t *testing.T, fields map[string]string, err error) {
	t.Helper()
	var want *ExtraBlockUnbalancedError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v (%T), want *ExtraBlockUnbalancedError", err, err)
	}
	if fields != nil {
		t.Errorf("fields = %v, want none for a rejected entry", fields)
	}
}

// AC-AKX-003 — what is NOT a marker line passes through untouched, and the
// documented block shapes keep rendering as before. This is the check's
// false-positive guard; every row renders identically before and after it.
func TestRenderAcceptsExtraMarkerLookalikes(t *testing.T) {
	for _, c := range []struct {
		name      string
		body      string
		wantText  []string
		wantExtra []string
	}{
		{
			// Org's in-block escape: the leading comma keeps the line from
			// being read as a marker.
			name:      "a_comma_escaped_marker_inside_a_block",
			body:      "body\n#+BEGIN_EXTRA\n,#+BEGIN_EXTRA\n#+END_EXTRA\n",
			wantExtra: []string{",#+BEGIN_EXTRA"},
		},
		{
			name:     "a_marker_in_the_middle_of_a_line",
			body:     "see #+END_EXTRA here\n",
			wantText: []string{"see #+END_EXTRA here"},
		},
		{
			name:      "a_verbatim_marker_inside_a_block",
			body:      "body\n#+BEGIN_EXTRA\n=#+END_EXTRA=\n#+END_EXTRA\n",
			wantExtra: []string{"#+END_EXTRA"},
		},
		{
			name:      "one_block",
			body:      "body\n#+BEGIN_EXTRA\nnote\n#+END_EXTRA\n",
			wantText:  []string{"body"},
			wantExtra: []string{"note"},
		},
		{
			name:      "two_sequential_blocks",
			body:      "body\n#+BEGIN_EXTRA\none\n#+END_EXTRA\n#+BEGIN_EXTRA\ntwo\n#+END_EXTRA\n",
			wantText:  []string{"body"},
			wantExtra: []string{"one", "two"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			fields, err := Render(NoteTypeCloze, "Q {{c1::x}}", c.body)
			if err != nil {
				t.Fatalf("err = %v, want none", err)
			}
			for _, want := range c.wantText {
				if !strings.Contains(fields["Text"], want) {
					t.Errorf("Text = %q, missing %q", fields["Text"], want)
				}
			}
			for _, want := range c.wantExtra {
				if !strings.Contains(fields["Back Extra"], want) {
					t.Errorf("Back Extra = %q, missing %q", fields["Back Extra"], want)
				}
			}
		})
	}
}
