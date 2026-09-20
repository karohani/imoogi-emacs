package orgdoc

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

// text renders a multiline entry and returns its Text field, failing the test
// on any error. Every criterion in this file is stated against the rendered
// Text field, which is what the planner hashes and what Anki receives.
func text(t *testing.T, title, body string, opts CardOptions) string {
	t.Helper()
	fields, err := RenderWithOptions(NoteTypeCloze, title, body, opts)
	if err != nil {
		t.Fatalf("RenderWithOptions(%q, %q, %+v): %v", title, body, opts, err)
	}
	return fields["Text"]
}

func mustContain(t *testing.T, got, want, why string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("%s\n  rendered: %s\n  missing : %s", why, got, want)
	}
}

func mustNotContain(t *testing.T, got, unwanted, why string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Errorf("%s\n  rendered: %s\n  contains: %s", why, got, unwanted)
	}
}

// AC-ML-003 — the resolved direction selects what composition wraps.
func TestMultilineDirection(t *testing.T) {
	const title, body = "Capital of Japan", "- Tokyo\n"

	t.Run("a_rightward_wraps_the_answers", func(t *testing.T) {
		got := text(t, title, body, CardOptions{Direction: DirectionRightward})
		mustContain(t, got, "{{c1::Tokyo}}", "`->` wraps each answer item")
		mustContain(t, got, "<p>Capital of Japan</p>", "`->` leaves the title unwrapped")
	})

	t.Run("b_leftward_wraps_the_title", func(t *testing.T) {
		// The NUMBER is part of the assertion: the design record fixes the
		// leftward title at c2, and numbering is a hash input, so leaving it
		// loose would let it drift silently.
		got := text(t, title, body, CardOptions{Direction: DirectionLeftward})
		mustContain(t, got, "{{c2::Capital of Japan}}", "`<-` wraps the title at c2")
		mustContain(t, got, ">Tokyo<", "`<-` leaves the answer items unwrapped")
		mustNotContain(t, got, "{{c1::", "`<-` wraps no answer")
	})

	t.Run("c_both_wraps_both_with_different_numbers", func(t *testing.T) {
		got := text(t, title, body, CardOptions{Direction: DirectionBoth})
		mustContain(t, got, "{{c1::Tokyo}}", "`<->` wraps the answer")
		mustContain(t, got, "{{c2::Capital of Japan}}", "`<->` wraps the title under a different number")
	})
}

// AC-ML-004 — a multiline entry whose direction did not resolve to an arrow
// composes as though `->` had been written.
func TestMultilineDefaultDirection(t *testing.T) {
	const title, body = "Capital of Japan", "- Tokyo\n- Osaka\n"

	// Incremental alone is the shape this default exists for: it is what makes
	// ANKI_INCREMENTAL usable without ANKI_DIRECTION.
	absent := text(t, title, body, CardOptions{Direction: DirectionNone, Incremental: true})
	mustContain(t, absent, "{{c1::Tokyo}}", "an unresolved direction wraps the answers")
	mustContain(t, absent, "{{c2::Osaka}}", "an unresolved direction wraps the answers")
	mustContain(t, absent, "<p>Capital of Japan</p>", "an unresolved direction leaves the title unwrapped")

	// The default binds the explicitly falsy case too, which SPEC-ANKICARD-002
	// recognizes as "no direction" rather than as an error. The two spellings
	// are distinct INPUTS only at the planner's boundary — `readDirection`
	// resolves both to the same state before the renderer sees them — so the
	// byte-identity half of this criterion is asserted there, where the inputs
	// genuinely differ, rather than here where it would be tautological. See
	// TestMultilineDefaultDirection in internal/anki/planner.
	explicit := text(t, title, body, CardOptions{Direction: DirectionRightward, Incremental: true})
	if explicit != absent {
		t.Errorf("an unresolved direction does not compose as `->`\n got: %s\nwant: %s", absent, explicit)
	}
}

// AC-ML-005 — the incremental setting selects how many distinct numbers the
// answers carry.
func TestMultilineIncremental(t *testing.T) {
	const title, body = "Capital of Japan", "- Tokyo\n- Osaka\n"

	t.Run("a_off_the_answers_share_one_number", func(t *testing.T) {
		got := text(t, title, body, CardOptions{Direction: DirectionRightward})
		mustContain(t, got, "{{c1::Tokyo}}", "both answers share one number")
		mustContain(t, got, "{{c1::Osaka}}", "both answers share one number")
	})

	t.Run("b_on_each_answer_carries_its_own_number_in_document_order", func(t *testing.T) {
		got := text(t, title, body, CardOptions{Direction: DirectionRightward, Incremental: true})
		mustContain(t, got, "{{c1::Tokyo}}", "the first answer takes the first number")
		mustContain(t, got, "{{c2::Osaka}}", "the second answer takes the second number")
		if strings.Index(got, "{{c1::") > strings.Index(got, "{{c2::") {
			t.Errorf("numbers are not assigned in document order: %s", got)
		}
	})

	t.Run("c_both_with_incremental_keeps_the_title_distinct", func(t *testing.T) {
		got := text(t, title, body, CardOptions{Direction: DirectionBoth, Incremental: true})
		mustContain(t, got, "{{c1::Tokyo}}", "the answers consume consecutive numbers")
		mustContain(t, got, "{{c2::Osaka}}", "the answers consume consecutive numbers")
		mustContain(t, got, "{{c3::Capital of Japan}}", "the title's number differs from every answer's")
	})
}

// AC-ML-006 — generated numbers do not collide with a hand-written marker, and
// a span already carrying one is not wrapped.
func TestMultilineNumbering(t *testing.T) {
	t.Run("a_an_item_carrying_a_hand_written_marker_is_not_wrapped", func(t *testing.T) {
		got := text(t, "Cities", "- Tokyo is {{c4::big}}\n- Osaka\n",
			CardOptions{Direction: DirectionRightward})
		// Anki's cloze pattern closes at the first `}}` after an opening
		// marker, so a generated marker placed around a hand-written one is
		// ended by it. Not wrapping is what delivers the byte-for-byte survival
		// the criterion asks for; wrapping would have separated that `}}` into
		// `} }` and destroyed the author's existing card.
		mustContain(t, got, "Tokyo is {{c4::big}}", "the hand-written item survives byte-for-byte")
		mustNotContain(t, got, "{{c4::big} }", "the brace rule did not reach an unwrapped item")
		mustContain(t, got, "{{c5::Osaka}}", "generated numbering begins above the highest hand-written one")
		if nested := regexp.MustCompile(`\{\{c\d+::[^}]*\{\{c\d+::`).FindString(got); nested != "" {
			t.Errorf("a generated marker nests inside another: %q in %s", nested, got)
		}
	})

	t.Run("b_the_ordinary_case_is_the_design_records_numbering", func(t *testing.T) {
		got := text(t, "Capital of Japan", "- Tokyo\n- Osaka\n", CardOptions{Direction: DirectionBoth})
		mustContain(t, got, "{{c1::Tokyo}}", "the answers carry c1")
		mustContain(t, got, "{{c1::Osaka}}", "the answers carry c1")
		mustContain(t, got, "{{c2::Capital of Japan}}", "the title carries c2")
	})

	t.Run("c_a_marker_in_the_title_counts_toward_the_offset", func(t *testing.T) {
		got := text(t, "Capital of {{c7::Japan}}", "- Tokyo\n",
			CardOptions{Direction: DirectionRightward})
		mustContain(t, got, "{{c8::Tokyo}}", "generated numbering begins above the title's marker")
		mustContain(t, got, "Capital of {{c7::Japan}}", "the title reaches the output unchanged")
	})

	t.Run("d_an_entry_whose_every_answer_is_hand_clozed_still_renders", func(t *testing.T) {
		got := text(t, "Cities", "- {{c1::Tokyo}}\n- {{c2::Osaka}}\n",
			CardOptions{Direction: DirectionRightward})
		mustContain(t, got, "{{c1::Tokyo}}", "the author's marker reaches the output unchanged")
		mustContain(t, got, "{{c2::Osaka}}", "the author's marker reaches the output unchanged")
		mustNotContain(t, got, "{{c3::", "nothing is wrapped")
	})

	t.Run("e_the_not_wrap_rule_governs_incremental", func(t *testing.T) {
		// The shared number is the point of the fixture. Per-answer numbering
		// binds only the spans composition wraps, so with none wrapped the
		// entry yields one card — the specified outcome, not an unmet
		// obligation, since composition cannot renumber an author's marker
		// without destroying the card that marker already makes.
		got := text(t, "Cities", "- {{c1::Tokyo}}\n- {{c1::Osaka}}\n",
			CardOptions{Direction: DirectionRightward, Incremental: true})
		mustContain(t, got, "{{c1::Tokyo}}", "both markers keep the number 1")
		mustContain(t, got, "{{c1::Osaka}}", "both markers keep the number 1")
		mustNotContain(t, got, "{{c2::", "nothing is renumbered")
	})
}

// AC-ML-007 — a generated marker cannot close early.
func TestMultilineBraceSafety(t *testing.T) {
	// betweenMarker returns what sits between a generated marker's opening and
	// its own closing `}}` — the span the criterion states its property over.
	betweenMarker := func(t *testing.T, got string) string {
		t.Helper()
		open := strings.Index(got, "{{c1::")
		if open < 0 {
			t.Fatalf("no generated marker in %s", got)
		}
		rest := got[open+len("{{c1::"):]
		close := strings.LastIndex(rest, "}}")
		if close < 0 {
			t.Fatalf("generated marker never closes in %s", got)
		}
		return rest[:close]
	}

	t.Run("a_a_brace_run_inside_wrapped_content_is_separated", func(t *testing.T) {
		got := text(t, "Formula", "- \\sqrt{a^{2}}\n", CardOptions{Direction: DirectionRightward})
		if inner := betweenMarker(t, got); strings.Contains(inner, "}}") {
			t.Errorf("`}}` survives inside the generated marker — the blank would close early\n  inner: %q\n  full : %s", inner, got)
		}
	})

	t.Run("b_content_ending_in_a_brace_is_padded", func(t *testing.T) {
		got := text(t, "Formula", "- f(x}\n", CardOptions{Direction: DirectionRightward})
		mustContain(t, got, "{{c1::f(x} }}", "a space separates the trailing brace from the closing `}}`")
	})
}

// AC-ML-007c — the Go composer and the editor helper produce the same form for
// every brace-hazard input, read from ONE shared fixture rather than from
// literals duplicated in each suite. A divergence means an author sees one
// result when they cloze by hand and another when the same text is wrapped by
// composition; duplicated literals would let the two drift unnoticed.
func TestMultilineBraceFixtureMatchesTheEditor(t *testing.T) {
	for _, c := range loadBraceFixture(t) {
		if got := clozeSafeText(c.In); got != c.Want {
			t.Errorf("clozeSafeText(%q)\n got: %q\nwant: %q", c.In, got, c.Want)
		}
	}
}

// AC-ML-008 — incremental with `<-` is inert and raises no diagnostic.
func TestMultilineLeftwardIncremental(t *testing.T) {
	const title, body = "Capital of Japan", "- Tokyo\n- Osaka\n"
	on := text(t, title, body, CardOptions{Direction: DirectionLeftward, Incremental: true})
	off := text(t, title, body, CardOptions{Direction: DirectionLeftward})
	if on != off {
		t.Errorf("incremental is not inert under `<-`\n  on: %s\n off: %s", on, off)
	}
	// `<-` wraps only the title, so there is nothing for per-item numbering to
	// number. Reporting the combination would cost a diagnostic code to
	// describe a card that is correct.
	mustContain(t, on, "{{c2::Capital of Japan}}", "the leftward title keeps its fixed number")
}

// AC-ML-011a and AC-ML-011d — the answer list renders inside a container the
// stylesheet can target, on every list form and under every direction.
func TestMultilineContainer(t *testing.T) {
	for _, c := range []struct{ name, body, element string }{
		{"unordered", "- Tokyo\n- Osaka\n", "ul"},
		{"ordered", "1. Tokyo\n2. Osaka\n", "ol"},
		{"description", "- Tokyo :: the capital\n- Osaka :: the second\n", "dl"},
	} {
		t.Run("a_"+c.name, func(t *testing.T) {
			got := text(t, "Cities", c.body, CardOptions{Direction: DirectionRightward})
			mustContain(t, got, `<`+c.element+` class="children-list">`,
				"the container class is emitted on this list form")
			if n := strings.Count(got, "children-list"); n != 1 {
				t.Errorf("the Text field carries %d elements bearing the class, want exactly 1: %s", n, got)
			}
		})
	}

	t.Run("d_the_container_is_present_under_leftward_too", func(t *testing.T) {
		// An implementation attaching the container as part of the
		// answer-wrapping step would omit it here, where no answer is wrapped.
		got := text(t, "Cities", "- Tokyo\n- Osaka\n", CardOptions{Direction: DirectionLeftward})
		mustContain(t, got, `class="children-list"`, "the container does not depend on any answer being wrapped")
		mustContain(t, got, ">Tokyo<", "both items sit inside the container")
		mustContain(t, got, ">Osaka<", "both items sit inside the container")
	})
}

// The rendered half of AC-ML-001: the span rules of REQ-ML-001.3 stated as
// properties of the output a user actually receives. The corpus half lives in
// multiline_test.go; both run under `-run TestMultilineAnswerList`.
func TestMultilineAnswerListRendering(t *testing.T) {
	right := CardOptions{Direction: DirectionRightward}

	t.Run("a_a_plain_list_supplies_the_answers", func(t *testing.T) {
		got := text(t, "Capital of Japan", "- Tokyo\n- Osaka\n", right)
		mustContain(t, got, "{{c1::Tokyo}}", "each item is wrapped")
		mustContain(t, got, "{{c1::Osaka}}", "each item is wrapped")
		mustContain(t, got, "<p>Capital of Japan</p>", "the title is outside every marker")
	})

	t.Run("b_a_nested_sub_item_is_not_an_answer_item", func(t *testing.T) {
		got := text(t, "Cities", "- Tokyo\n  - Kanto\n- Osaka\n", right)
		mustContain(t, got, "{{c1::Tokyo}}", "the parent item is wrapped")
		mustContain(t, got, "{{c1::Osaka}}", "the sibling item is wrapped")
		mustContain(t, got, "<li>Kanto</li>", "the nested child stays visible outside every marker")
	})

	t.Run("c_an_ordered_list_counts", func(t *testing.T) {
		got := text(t, "Cities", "1. A\n2. B\n", right)
		mustContain(t, got, "{{c1::A}}", "an ordered item is wrapped like an unordered one")
		mustContain(t, got, "{{c1::B}}", "an ordered item is wrapped like an unordered one")
	})

	t.Run("d_a_description_item_is_wrapped_on_its_definition_half", func(t *testing.T) {
		got := text(t, "Cities", "- Tokyo :: the capital\n", right)
		mustContain(t, got, "<dd>{{c1::the capital}}</dd>", "the definition half is wrapped")
		// go-org emits a description term across its own lines, so the term is
		// matched against its closing tag rather than as a one-line element.
		mustContain(t, got, "Tokyo\n</dt>", "the term half is not wrapped")
		// A marker torn across the term and the definition is the specific
		// corruption this rejects: go-org splits the item at the first ` :: `,
		// so a whole-item wrap puts the marker's own `::` on the wrong side.
		mustNotContain(t, got, "<dt>{{c1::Tokyo", "the marker is not split across the term and the definition")
	})

	t.Run("f_a_checkbox_item_is_wrapped_after_its_checkbox", func(t *testing.T) {
		got := text(t, "Cities", "- [ ] Tokyo\n- [X] Osaka\n", right)
		mustContain(t, got, `<li class="unchecked">{{c1::Tokyo}}</li>`, "the marker opens after the checkbox")
		mustContain(t, got, `<li class="checked">{{c1::Osaka}}</li>`, "the marker opens after the checkbox")
		// `::[` is the signature of a marker whose opening the status parser
		// consumed — go-org reads the status before the item content.
		mustNotContain(t, got, "::[", "the checkbox parser did not consume the marker's opening")
	})

	t.Run("g_a_plain_sibling_in_a_description_list_is_wrapped_whole", func(t *testing.T) {
		got := text(t, "Cities", "- Tokyo :: the capital\n- Osaka\n", right)
		mustContain(t, got, "<dd>{{c1::the capital}}</dd>", "the description item gives up its definition half")
		mustContain(t, got, "<dd>{{c1::Osaka}}</dd>", "the plain sibling has no definition half, so it is wrapped whole")
		mustNotContain(t, got, "<dt>{{c1::", "neither marker is split across the term and the definition")
	})

	t.Run("h_an_ordered_items_counter_cookie_stays_outside_the_marker", func(t *testing.T) {
		got := text(t, "Cities", "1. [@5] Tokyo\n2. Osaka\n", right)
		mustContain(t, got, `<li value="5">{{c1::Tokyo}}</li>`, "the cookie is consumed ahead of the marker")
		// `:[@` differs from the checkbox signature by one character, which is
		// why the checkbox criterion does not catch this shape.
		mustNotContain(t, got, ":[@", "the counter parser did not consume the marker's opening")
	})

	t.Run("i_a_mid_item_bracket_is_handled_by_mirroring_the_parser", func(t *testing.T) {
		// This is the criterion that fails a composer built by searching for a
		// leading token. The status match is unanchored and removes a fixed
		// count from the item's start, so it consumes four bytes here with no
		// leading token present. The item renders lossily either way — that
		// truncation is pre-existing go-org behaviour this SPEC does not fix;
		// what is required is that composition does not ALSO destroy the marker.
		got := text(t, "Cities", "- Tokyo [ ] is big\n", right)
		mustContain(t, got, "{{c1::", "the generated marker's opening survives")
		mustNotContain(t, got, "::Tokyo", "the composer did not wrap whole by searching for a leading token")
	})

	t.Run("j_the_offset_is_a_byte_offset_on_multi_byte_content", func(t *testing.T) {
		got := text(t, "도시", "- [ ] 서울특별시\n- [X] 부산\n", right)
		mustContain(t, got, "{{c1::서울특별시}}", "the marker opens immediately after the status marker")
		mustContain(t, got, "{{c1::부산}}", "the marker opens immediately after the status marker")
		// The falsifier a rune-rounded composer fails: the parser removes a
		// fixed BYTE count and does not round, so a composer that rounded would
		// place its opening at a different offset.
		if !utf8.ValidString(got) {
			t.Errorf("the rendered output is not valid UTF-8: %q", got)
		}
	})

	t.Run("e_a_lead_paragraph_and_a_later_list_are_not_answers", func(t *testing.T) {
		got := text(t, "Cities",
			"Think first.\n\n- Tokyo\n\nAlso worth noting.\n\n- Unrelated\n", right)
		mustContain(t, got, "{{c1::Tokyo}}", "the first list supplies the answers")
		if n := strings.Count(got, "{{c"); n != 1 {
			t.Errorf("the output carries %d markers, want exactly 1: %s", n, got)
		}
		for _, outside := range []string{"Think first.", "Also worth noting.", "Unrelated"} {
			mustContain(t, got, outside, "content outside the answer list still renders")
		}
	})
}

// braceFixtureCase is one row of the shared brace-hazard fixture. The fixture
// is read by BOTH this suite and tests/anki-commands-test.el rather than
// duplicated as literals in each, so the Go composer and the editor helper
// cannot drift without one of them failing.
type braceFixtureCase struct {
	Why  string `json:"why"`
	In   string `json:"in"`
	Want string `json:"want"`
}

const braceFixturePath = "testdata/cloze-brace-fixture.json"

func loadBraceFixture(t *testing.T) []braceFixtureCase {
	t.Helper()
	raw, err := os.ReadFile(braceFixturePath)
	if err != nil {
		t.Fatalf("reading the shared brace fixture %s: %v", braceFixturePath, err)
	}
	var cases []braceFixtureCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("decoding the shared brace fixture: %v", err)
	}
	if len(cases) == 0 {
		t.Fatalf("the shared brace fixture is empty")
	}
	return cases
}

// TestMultilineComposeEdges covers the composition branches the AC fixtures
// reach only incidentally.
func TestMultilineComposeEdges(t *testing.T) {
	t.Run("an_item_with_no_content_is_counted_but_not_wrapped", func(t *testing.T) {
		// An empty bullet is still an item to the parser, so the entry is not
		// reported as answer-missing; there is simply nothing to wrap in it.
		got := text(t, "Cities", "- \n- Osaka\n", CardOptions{Direction: DirectionRightward})
		mustContain(t, got, "{{c1::Osaka}}", "the item that has content is wrapped")
		mustNotContain(t, got, "{{c1::}}", "the empty item contributes no marker")
	})

	t.Run("a_title_already_carrying_a_marker_is_not_wrapped", func(t *testing.T) {
		// The not-wrap rule binds the title exactly as it binds an answer item.
		got := text(t, "Capital of {{c7::Japan}}", "- Tokyo\n", CardOptions{Direction: DirectionBoth})
		mustContain(t, got, "Capital of {{c7::Japan}}", "the title survives byte-for-byte")
		mustNotContain(t, got, "{{c9::Capital", "no generated marker is placed around it")
		mustContain(t, got, "{{c8::Tokyo}}", "the answers still number above the author's marker")
	})

	t.Run("a_non_cloze_note_type_takes_the_ordinary_path", func(t *testing.T) {
		// Unreachable from production — the planner's validation gate rejects
		// an option on a non-cloze type before the render — but the entry point
		// is total rather than partial.
		fields, err := RenderWithOptions(NoteTypeBasic, "Q", "- A\n", CardOptions{Direction: DirectionRightward})
		if err != nil {
			t.Fatalf("RenderWithOptions on a Basic note type: %v", err)
		}
		plain, err := Render(NoteTypeBasic, "Q", "- A\n")
		if err != nil {
			t.Fatalf("Render on a Basic note type: %v", err)
		}
		if fields["Front"] != plain["Front"] || fields["Back"] != plain["Back"] {
			t.Errorf("a non-cloze note type did not delegate\n got: %v\nwant: %v", fields, plain)
		}
	})

	t.Run("a_multiline_entry_with_no_answer_list_is_rejected", func(t *testing.T) {
		// The obligation holds for EVERY direction, `<-` included: on a body
		// with no list a leftward card hides the only content it has and
		// leaves a prompt with no visible cue.
		for _, direction := range []Direction{DirectionRightward, DirectionLeftward, DirectionBoth} {
			_, err := RenderWithOptions(NoteTypeCloze, "Q", "Just a paragraph.\n",
				CardOptions{Direction: direction})
			if _, ok := err.(*MultilineAnswerMissingError); !ok {
				t.Fatalf("direction %v: got %v, want *MultilineAnswerMissingError", direction, err)
			}
			// The message names the problem rather than restating the type.
			// The planner carries it into the diagnostic the user reads.
			if msg := err.Error(); !strings.Contains(msg, "answer") {
				t.Errorf("direction %v: the error message does not name the problem: %q", direction, msg)
			}
		}
	})

	t.Run("a_composition_that_generates_no_marker_still_meets_the_gate", func(t *testing.T) {
		// The gate is SATISFIED by composition (REQ-ML-009.3), not bypassed by
		// it. An answer list of empty items generates nothing, so an entry
		// carrying no hand-written marker is skipped exactly as any other
		// unmarked cloze entry is.
		for _, c := range []struct {
			name  string
			title string
			opts  CardOptions
		}{
			{"rightward_with_an_empty_answer", "Cities", CardOptions{Direction: DirectionRightward}},
			{"leftward_with_an_empty_title", "", CardOptions{Direction: DirectionLeftward}},
		} {
			_, err := RenderWithOptions(NoteTypeCloze, c.title, "- \n", c.opts)
			if _, ok := err.(*ClozeMarkerMissingError); !ok {
				t.Errorf("%s: got %v, want *ClozeMarkerMissingError", c.name, err)
			}
		}
	})
}
