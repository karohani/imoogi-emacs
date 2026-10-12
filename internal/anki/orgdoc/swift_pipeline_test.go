package orgdoc

import (
	"errors"
	"strings"
	"testing"
)

// SPEC-ANKICARD-004 AC-SW-008 — the swift branch runs split, check, compose,
// gate, render, in that order (REQ-SW-008).
func TestSwiftPipelineOrder(t *testing.T) {
	swift := CardOptions{Swift: true}

	t.Run("supplementary_content_never_becomes_an_arrow_line", func(t *testing.T) {
		const body = "#+BEGIN_EXTRA\n도쿄 :-> 일본\n#+END_EXTRA\n"
		_, err := RenderWithOptions(NoteTypeCloze, "제목", body, swift)
		var missing *SwiftArrowMissingError
		if !errors.As(err, &missing) {
			t.Fatalf("err = %v, want *SwiftArrowMissingError — the split ran first", err)
		}
		rest, extra, err := splitExtraBlocksChecked(body)
		if err != nil || !strings.Contains(extra, "도쿄 :-> 일본") || strings.Contains(rest, ":->") {
			t.Errorf("checked split = (rest %q, extra %q, err %v), want the arrow line in extra alone", rest, extra, err)
		}
	})

	t.Run("an_orphan_marker_is_reported_before_composition", func(t *testing.T) {
		_, err := RenderWithOptions(NoteTypeCloze, "제목", "도쿄 :-> 일본\n#+END_EXTRA\n서울 :-> 한국", swift)
		var unbalanced *ExtraBlockUnbalancedError
		if !errors.As(err, &unbalanced) {
			t.Errorf("err = %v, want *ExtraBlockUnbalancedError from the checked split", err)
		}
	})

	t.Run("a_generated_marker_satisfies_the_gate", func(t *testing.T) {
		fields, err := RenderWithOptions(NoteTypeCloze, "제목", "도쿄 :-> 일본", swift)
		if err != nil {
			t.Fatalf("err = %v, want the entry rendered rather than skipped as unmarked", err)
		}
		if !strings.Contains(fields["Text"], "{{c1::일본}}") {
			t.Errorf("Text = %q, want the generated marker", fields["Text"])
		}
		if _, ok := fields["Back Extra"]; !ok {
			t.Error("Back Extra is absent, want it emitted even when empty")
		}
	})

	t.Run("a_blank_line_run_reaches_the_renderer_uncollapsed", func(t *testing.T) {
		// Corpus row 15 renders as three body paragraphs below the title's
		// own: the gaps are kept (plan.md DD-7).
		fields, err := RenderWithOptions(NoteTypeCloze, "제목", "A :-> B\n\ntext\n\nC :-> D", swift)
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(fields["Text"], "<p>"); n != 4 {
			t.Errorf("Text = %q carries %d paragraphs, want the title's and three", fields["Text"], n)
		}
	})

	t.Run("the_extra_field_carries_the_supplementary_content", func(t *testing.T) {
		fields, err := RenderWithOptions(NoteTypeCloze, "제목", "도쿄 :-> 일본\n\n#+BEGIN_EXTRA\n메모\n#+END_EXTRA", swift)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(fields["Back Extra"], "메모") || strings.Contains(fields["Text"], "메모") {
			t.Errorf("fields = %q, want the extra content in Back Extra alone", fields)
		}
	})
}
