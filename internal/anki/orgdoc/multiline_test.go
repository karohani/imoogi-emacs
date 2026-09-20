package orgdoc

import (
	"strings"
	"testing"
)

// The M1 corpus (plan.md § F M1). Every row is a REQUIRED member: each
// stresses a different axis of the scanner, and dropping one removes the only
// falsifier for that axis (plan.md § G anti-pattern 9).
//
// `items` is the number of answer items the FIRST top-level list of the
// collapsed body carries, as go-org itself parses it. `found` is whether that
// list exists at all. `spans` — when non-nil — asserts the exact wrapped text
// per item; those are the rows the count-equality property is blind to by
// construction, because the scanner and the parse agree on how many items
// there are and disagree only on what part of each is the answer.
var multilineCorpus = []struct {
	name  string
	body  string
	found bool
	items int
	spans []string
}{
	// --- block interiors are not the answer list -----------------------------
	{
		name:  "src_block_bullet_is_not_an_item",
		body:  "#+BEGIN_SRC text\n- x\n#+END_SRC\n- real\n",
		found: true, items: 1,
		spans: []string{"real"},
	},
	{
		name:  "example_block_yields_no_list_at_all",
		body:  "#+BEGIN_EXAMPLE\n- x\n#+END_EXAMPLE\n",
		found: false,
	},
	{
		name:  "quote_block_list_is_real_but_not_top_level",
		body:  "#+BEGIN_QUOTE\n- inner\n#+END_QUOTE\n",
		found: false,
	},

	// --- blank-line runs -----------------------------------------------------
	{
		name:  "one_blank_line_keeps_one_list",
		body:  "- A\n\n- B\n",
		found: true, items: 2,
		spans: []string{"A", "B"},
	},
	{
		name:  "two_blank_lines_read_as_one_list_after_the_collapse",
		body:  "- A\n\n\n- B\n",
		found: true, items: 2,
		spans: []string{"A", "B"},
	},
	{
		name:  "genuine_content_still_separates",
		body:  "- A\n\nNote.\n\n- B\n",
		found: true, items: 1,
		spans: []string{"A"},
	},

	// --- span assertions: counts agree, spans differ -------------------------
	{
		name:  "span_checkbox_status_is_consumed",
		body:  "- [ ] A\n- [X] B\n",
		found: true, items: 2,
		spans: []string{"A", "B"},
	},
	{
		name:  "span_plain_sibling_in_a_description_list",
		body:  "- term :: def\n- plain\n",
		found: true, items: 2,
		spans: []string{"def", "plain"},
	},
	{
		name:  "span_ordered_counter_cookie_is_consumed",
		body:  "1. [@5] A\n2. B\n",
		found: true, items: 2,
		spans: []string{"A", "B"},
	},
	{
		name:  "span_position_blind_status_removes_four_bytes_with_no_leading_token",
		body:  "- Tokyo [ ] is big\n",
		found: true, items: 1,
		// The status expression is unanchored and the parser then removes a
		// FIXED four bytes from position zero, so the answer content begins at
		// byte 4 — `o [ ] is big` — with no leading token anywhere. A composer
		// that searched for one would find nothing and wrap whole.
		spans: []string{"o [ ] is big"},
	},

	// --- list shapes ---------------------------------------------------------
	{
		name:  "paren_terminator_is_an_ordered_list",
		body:  "1) A\n2) B\n",
		found: true, items: 2,
		spans: []string{"A", "B"},
	},
	{
		name:  "indented_first_bullet_is_top_level",
		body:  "  - A\n  - B\n",
		found: true, items: 2,
		spans: []string{"A", "B"},
	},
	{
		name:  "a_bullet_change_merges_into_one_list",
		body:  "- A\n+ B\n",
		found: true, items: 2,
		spans: []string{"A", "B"},
	},

	// --- the collapse's boundaries -------------------------------------------
	{
		name:  "the_collapse_does_not_reach_inside_a_block",
		body:  "#+BEGIN_SRC text\nx\n\n\ny\n#+END_SRC\n\n- A\n",
		found: true, items: 1,
		spans: []string{"A"},
	},
	{
		name:  "two_author_written_lists_merge_into_one",
		body:  "- A\n- B\n\n\n- C\n- D\n",
		found: true, items: 4,
		spans: []string{"A", "B", "C", "D"},
	},
	{
		name:  "an_unterminated_block_makes_the_collapse_decline",
		body:  "#+BEGIN_SRC text\n- A\n\n\n- B\n",
		found: true, items: 1,
		spans: []string{"A"},
	},

	// --- multi-byte: the offset is in BYTES ----------------------------------
	{
		name:  "multi_byte_position_blind_slice",
		body:  "- 서울특별시 [ ] 큼\n",
		found: true, items: 1,
		// Four BYTES, not four runes: three of `서` plus one of `울`. A
		// rune-rounded composer places its opening somewhere the parser does
		// not start content, which destroys the marker (plan.md DD-3).
		spans: []string{"- 서울특별시 [ ] 큼"[len("- ")+4:]},
	},
	{
		name:  "multi_byte_status_item",
		body:  "- [ ] 서울특별시\n",
		found: true, items: 1,
		spans: []string{"서울특별시"},
	},

	// --- a dangling supplementary marker in the remaining body ---------------
	//
	// The supplementary split can leave an unpaired `#+BEGIN_EXTRA` or
	// `#+END_EXTRA` behind, and the bodies below are its measured residue. They
	// are here to PIN what composition does with one, which is: nothing
	// special. A dangling marker is ordinary content, and it ends the answer
	// list exactly as a paragraph does — REQ-ML-001.4 and REQ-ML-001.5 applied
	// unchanged, and the same outcome AC-ML-009e already fixes for a paragraph.
	//
	// The consequence is user-visible and is NOT silent-by-choice: the answers
	// after the marker are dropped from the card and the marker itself renders
	// as a literal paragraph on it. Reporting that would need a diagnostic code
	// this SPEC does not define; see the note on composeMultiline.
	{
		// `- A / #+BEGIN_EXTRA note #+END_EXTRA / #+END_EXTRA / - B` — the pair
		// is TERMINATED, so no unterminated-block rule is engaged and the
		// collapse has nothing to decline. The stray closing marker alone ends
		// the list.
		name:  "dangling_end_marker_ends_the_answer_list",
		body:  "- A\n#+END_EXTRA\n- B",
		found: true, items: 1,
		spans: []string{"A"},
	},
	{
		// The same residue when the author separated the block with blank
		// lines, which is the shape Org authoring actually produces.
		name:  "dangling_end_marker_survives_the_collapse",
		body:  "- A\n\n\n#+END_EXTRA\n\n- B",
		found: true, items: 1,
		spans: []string{"A"},
	},
	{
		// A lone closing marker with no opening: identical residue, identical
		// outcome. It opens no block interior, here or in go-org.
		name:  "lone_end_marker_with_no_opening",
		body:  "- A\n#+END_EXTRA\n- B\n",
		found: true, items: 1,
		spans: []string{"A"},
	},
	{
		// An unterminated `#+BEGIN_EXTRA` is worse: go-org swallows the
		// following bullet into the marker's own paragraph, so the later answer
		// stops being a list item at all. The scanner agrees with the parse at
		// one item, which is why the DD-1 cross-check cannot catch this — it is
		// not a scanner defect.
		name:  "dangling_begin_marker_swallows_the_later_answer",
		body:  "- A\n#+BEGIN_EXTRA\nnote\n- B\n",
		found: true, items: 1,
		spans: []string{"A"},
	},
	{
		// The row that matters most, because its markers are BALANCED. From
		// `- A / #+BEGIN_EXTRA outer #+BEGIN_EXTRA inner #+END_EXTRA tail
		// #+END_EXTRA / - B` — two openings, two closings, properly nested —
		// the split's non-greedy match runs from the FIRST opening to the
		// FIRST closing, so the outer block's tail and its closing marker
		// survive into the remaining body. Balanced input, damaged output.
		name:  "balanced_nested_blocks_leave_a_stray_closing_marker",
		body:  "- A\n\ntail\n#+END_EXTRA\n\n- B",
		found: true, items: 1,
		spans: []string{"A"},
	},
	{
		// The positive control, and the reason the rows above are not a claim
		// that every supplementary block breaks the card. Two SEQUENTIAL
		// blocks — the shape `splitExtraBlocks` documents as supported — leave
		// only a wide blank-line run, which the collapse closes, so BOTH
		// answers survive. Pre-split body:
		// `- A / #+BEGIN_EXTRA one #+END_EXTRA / #+BEGIN_EXTRA two #+END_EXTRA / - B`.
		name:  "sequential_blocks_leave_only_a_gap_the_collapse_closes",
		body:  "- A\n\n\n\n- B",
		found: true, items: 2,
		spans: []string{"A", "B"},
	},
}

// TestMultilineAnswerList is AC-ML-001's deciding test. It carries the M1
// corpus (plan.md § F M1) and the DD-1 scanner-versus-parse cross-check.
func TestMultilineAnswerList(t *testing.T) {
	for _, c := range multilineCorpus {
		t.Run("corpus/"+c.name, func(t *testing.T) {
			collapsed := collapseBlankRuns(c.body)

			// DD-1's falsifiable cross-check: the scanner and the renderer must
			// read the same document. A disagreement on the item count means
			// the hand-rolled scanner has diverged from go-org, which is
			// exactly the class of bug a hand-rolled scanner invites.
			kind, parsed, parsedOK := parseFirstList(collapsed)
			if parsedOK != c.found {
				t.Fatalf("parse found=%v, corpus records %v (collapsed %q)", parsedOK, c.found, collapsed)
			}
			list, scanOK := scanAnswerList(collapsed, kind)
			if scanOK != parsedOK {
				t.Fatalf("scanner found=%v but parse found=%v (collapsed %q)", scanOK, parsedOK, collapsed)
			}
			if !c.found {
				return
			}
			if parsed != c.items {
				t.Fatalf("parse counted %d items, corpus records %d (collapsed %q)", parsed, c.items, collapsed)
			}
			if len(list.items) != parsed {
				t.Fatalf("scanner counted %d items, parse counted %d (collapsed %q)",
					len(list.items), parsed, collapsed)
			}

			if c.spans == nil {
				return
			}
			got := make([]string, 0, len(list.items))
			for _, it := range list.items {
				got = append(got, collapsed[it.start:it.end])
			}
			// The span equality IS the byte-offset assertion. On the multi-byte
			// corruption row the correct span is deliberately NOT valid UTF-8:
			// the parser cuts four BYTES from position zero and lands inside a
			// character, and mirroring that cut is the obligation. A
			// rune-rounded composer produces a different span and fails here.
			// The valid-UTF-8 requirement of AC-ML-001j is about the RENDERED
			// output of a correctly composed entry, and is asserted there.
			if strings.Join(got, "\x00") != strings.Join(c.spans, "\x00") {
				t.Errorf("wrapped spans\n got: %q\nwant: %q", got, c.spans)
			}
		})
	}
}

// TestMultilineBlankRunCollapse pins the collapse's own behaviour, separately
// from what the scanner then makes of it (spec.md REQ-ML-009.2).
func TestMultilineBlankRunCollapse(t *testing.T) {
	for _, c := range []struct {
		name string
		body string
		want string
	}{
		{"a_single_blank_is_left_alone", "- A\n\n- B\n", "- A\n\n- B\n"},
		{"a_run_collapses_to_one", "- A\n\n\n\n- B\n", "- A\n\n- B\n"},
		{"no_blank_run_is_byte_identical", "- A\n- B\n", "- A\n- B\n"},
		{"an_empty_body_is_left_alone", "", ""},
		{"a_body_with_no_trailing_newline_keeps_none", "- A\n\n\n- B", "- A\n\n- B"},
		{
			// A block's interior is content the author wrote. Collapsing it
			// reformats a code sample with no diagnostic — a silent content
			// change, which is the failure this exclusion exists to prevent.
			"a_block_interior_is_untouched",
			"#+BEGIN_SRC text\nx\n\n\ny\n#+END_SRC\n\n\n- A\n",
			"#+BEGIN_SRC text\nx\n\n\ny\n#+END_SRC\n\n- A\n",
		},
		{
			"an_example_block_interior_is_untouched",
			"#+BEGIN_EXAMPLE\nx\n\n\ny\n#+END_EXAMPLE\n\n\n- A\n",
			"#+BEGIN_EXAMPLE\nx\n\n\ny\n#+END_EXAMPLE\n\n- A\n",
		},
		{
			// A mismatched END does not close the block, so the BEGIN is
			// unterminated and the collapse declines from that line on.
			"a_mismatched_end_does_not_close_the_block",
			"#+BEGIN_SRC text\nx\n#+END_EXAMPLE\n\n\n- A\n",
			"#+BEGIN_SRC text\nx\n#+END_EXAMPLE\n\n\n- A\n",
		},
		{
			// The extent of an unterminated block is undefined, so the collapse
			// declines rather than guessing where it ends (REQ-ML-009.2).
			"an_unterminated_block_declines_from_its_line_on",
			"#+BEGIN_SRC text\n- A\n\n\n- B\n",
			"#+BEGIN_SRC text\n- A\n\n\n- B\n",
		},
		{
			// Only the undefined region declines. A run BEFORE the unterminated
			// opening has a defined extent and still collapses.
			"a_run_before_the_unterminated_opening_still_collapses",
			"- A\n\n\n- B\n#+BEGIN_SRC text\nx\n\n\ny\n",
			"- A\n\n- B\n#+BEGIN_SRC text\nx\n\n\ny\n",
		},
		{
			// An orphan #+END_ opens nothing, so nothing is undefined and the
			// collapse runs normally. Reachable from well-formed author input:
			// the #+BEGIN_EXTRA split leaves exactly this shape behind when the
			// author nested one EXTRA block inside another.
			"an_orphan_end_marker_opens_nothing",
			"- A\n#+END_EXTRA\n\n\n- B\n",
			"- A\n#+END_EXTRA\n\n- B\n",
		},
		{
			"a_lowercase_block_keyword_is_the_same_block",
			"#+begin_src text\nx\n\n\ny\n#+end_src\n\n\n- A\n",
			"#+begin_src text\nx\n\n\ny\n#+end_src\n\n- A\n",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := collapseBlankRuns(c.body); got != c.want {
				t.Errorf("collapseBlankRuns\n got: %q\nwant: %q", got, c.want)
			}
		})
	}
}

// TestMultilineScannerBoundaries pins where the scanner stops, which the
// corpus above exercises only incidentally.
func TestMultilineScannerBoundaries(t *testing.T) {
	for _, c := range []struct {
		name  string
		body  string
		spans []string
	}{
		{
			// A nested sub-item is not an answer item, and the generated marker
			// stops before it (spec.md REQ-ML-001.1, plan.md DD-2).
			"a_nested_sub_item_is_not_an_answer_item",
			"- Tokyo\n  - Kanto\n- Osaka\n",
			[]string{"Tokyo", "Osaka"},
		},
		{
			// A continuation line is the item's OWN content — it renders into
			// the same list item — so the span covers it. It stops at the first
			// nested bullet, the first blank line, or the first line indented
			// less than the item's own content.
			"a_continuation_line_is_part_of_the_span",
			"- Tokyo\n  and more\n- Osaka\n",
			[]string{"Tokyo\n  and more", "Osaka"},
		},
		{
			"a_blank_line_ends_the_span_but_not_the_item",
			"- Tokyo\n\n  and more\n- Osaka\n",
			[]string{"Tokyo", "Osaka"},
		},
		{
			"a_lead_paragraph_and_a_later_list_are_not_answers",
			"Think first.\n\n- Tokyo\n\nAlso worth noting.\n\n- Unrelated\n",
			[]string{"Tokyo"},
		},
		{
			// An item with no content is still an item to the parser, so the
			// scanner counts it; it contributes an empty span, which
			// composition then declines to wrap.
			"an_empty_item_counts_but_spans_nothing",
			"- \n- B\n",
			[]string{"", "B"},
		},
		{
			"a_trailing_paragraph_ends_the_list",
			"- A\n- B\nNot an item.\n",
			[]string{"A", "B"},
		},
		{
			"an_ordered_list_of_two_digits_takes_its_own_min_indent",
			"9. A\n10. B\n   still B\n",
			[]string{"A", "B\n   still B"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			collapsed := collapseBlankRuns(c.body)
			kind, _, ok := parseFirstList(collapsed)
			if !ok {
				t.Fatalf("parse found no list in %q", collapsed)
			}
			list, ok := scanAnswerList(collapsed, kind)
			if !ok {
				t.Fatalf("scanner found no list in %q", collapsed)
			}
			got := make([]string, 0, len(list.items))
			for _, it := range list.items {
				got = append(got, collapsed[it.start:it.end])
			}
			if strings.Join(got, "\x00") != strings.Join(c.spans, "\x00") {
				t.Errorf("spans\n got: %q\nwant: %q", got, c.spans)
			}
		})
	}
}

// TestMultilineScannerFindsNothing covers the shapes that yield no answer
// list at all — the condition REQ-ML-002's diagnostic fires on.
func TestMultilineScannerFindsNothing(t *testing.T) {
	for _, body := range []string{
		"",
		"Just a paragraph.\n",
		"#+BEGIN_EXAMPLE\n- x\n#+END_EXAMPLE\n",
	} {
		collapsed := collapseBlankRuns(body)
		kind, _, parsedOK := parseFirstList(collapsed)
		_, scanOK := scanAnswerList(collapsed, kind)
		if parsedOK || scanOK {
			t.Errorf("%q: parse found=%v scanner found=%v, both should be false", body, parsedOK, scanOK)
		}
	}
}
