package orgdoc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// SPEC-ANKICARD-004 — the swift arrow card. These tests drive the scanner and
// the composition directly; the dispatch through RenderWithOptions is pinned in
// swift_dispatch_test.go.

// swiftTitle is the heading title every scanner case is composed under. It
// carries no marker, so the numbering base is zero unless a body says
// otherwise.
const swiftTitle = "제목"

// swiftCorpusRow is one row of acceptance.md § Corpus: a remaining body and
// the (left, right) sides of every arrow line the scanner must find in it, in
// document order. A row with no sides is a row the scanner must find nothing
// in.
type swiftCorpusRow struct {
	row   int
	body  string
	sides [][2]string
}

// swiftCorpus is acceptance.md's forty-two-row corpus, verbatim.
var swiftCorpus = []swiftCorpusRow{
	{1, "도쿄 :-> 일본", [][2]string{{"도쿄", "일본"}}},
	{2, "도쿄 :-> 일본\n서울 :-> 한국", [][2]string{{"도쿄", "일본"}, {"서울", "한국"}}},
	{3, "A :<- B", [][2]string{{"A", "B"}}},
	{4, "A :<-> B", [][2]string{{"A", "B"}}},
	{5, "A :-> B :-> C", [][2]string{{"A", "B :-> C"}}},
	{6, "A:->B", [][2]string{{"A", "B"}}},
	{7, "A :->", nil},
	{8, ":-> B", nil},
	{9, "A :->   ", nil},
	{10, "- 도쿄 :-> 일본", nil},
	{11, "- Tokyo :: Japan :-> x", nil},
	{12, "#+BEGIN_SRC go\nx :-> y\n#+END_SRC", nil},
	{13, "#+BEGIN_QUOTE\nx :-> y\n#+END_QUOTE", nil},
	{14, "문맥 한 줄\n도쿄 :-> 일본", [][2]string{{"도쿄", "일본"}}},
	{15, "A :-> B\n\ntext\n\nC :-> D", [][2]string{{"A", "B"}, {"C", "D"}}},
	{16, "{{c1::A}} :-> B", [][2]string{{"{{c1::A}}", "B"}}},
	{17, "도쿄 :-> 일본\n\n- 참고 사항", [][2]string{{"도쿄", "일본"}}},
	{18, "{{c1::도쿄 :-> 일본}}", nil},
	{19, "{{c1::도쿄 :-> 일본", nil},
	{20, ":-> B :-> C", nil},
	{21, "- 수도\n  도쿄 :-> 일본", nil},
	{22, "| A :-> B |", nil},
	{23, "도쿄 :-> 일본\n참고: 성립 조건", [][2]string{{"도쿄", "일본"}}},
	{24, "{{c1::메모\n\n도쿄 :-> 일본", nil},
	{25, ": A :-> B", nil},
	{26, "[fn:1] A :-> B", nil},
	{27, "* A :-> B", nil},
	{28, "#+TITLE: A :-> B", nil},
	{29, "# A :-> B", nil},
	{30, "\\begin{align}\nA :-> B\n\\end{align}", nil},
	{31, "~도쿄 :-> 일본~", [][2]string{{"~도쿄", "일본~"}}},
	{32, "[fn::도쿄 :-> 일본]", [][2]string{{"[fn::도쿄", "일본]"}}},
	{33, "[fn:1] 메모\n도쿄 :-> 일본", nil},
	{34, "[fn:1] 메모\n\n\n도쿄 :-> 일본", [][2]string{{"도쿄", "일본"}}},
	{35, "\\begin{align}\n도쿄 :-> 일본", [][2]string{{"도쿄", "일본"}}},
	{36, "문맥\n:PROPERTIES:\n:X: 도쿄 :-> 일본\n:END:", nil},
	{37, "\\begin{align} A :-> B \\end{align}", [][2]string{{"\\begin{align} A", "B \\end{align}"}}},
	{38, ":NOTES:\nA :-> B\n:END:", [][2]string{{"A", "B"}}},
	{39, "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n\n#+RESULTS:\n도쿄 :-> 일본", nil},
	{40, "#+BEGIN_SRC sh\necho\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본", [][2]string{{"도쿄", "일본"}}},
	{41, "[fn:1] 메모\n#+BEGIN_SRC go\n\n\n#+END_SRC\n도쿄 :-> 일본", nil},
	{42, "문맥\n:PROPERTIES:\n도쿄 :-> 일본\n:END:", [][2]string{{"도쿄", "일본"}}},
}

// arrowSides reads back the (left, right) text of every arrow line the scanner
// found, as substrings of the body it was given.
func arrowSides(body string, arrows []arrowLine) [][2]string {
	var sides [][2]string
	for _, a := range arrows {
		sides = append(sides, [2]string{body[a.leftStart:a.leftEnd], body[a.rightStart:a.rightEnd]})
	}
	return sides
}

// corpusRow returns one row of the corpus by its number.
func corpusRow(t *testing.T, n int) swiftCorpusRow {
	t.Helper()
	for _, r := range swiftCorpus {
		if r.row == n {
			return r
		}
	}
	t.Fatalf("corpus has no row %d", n)
	return swiftCorpusRow{}
}

// composeRow scans and composes one body under swiftTitle, returning the
// composed body (which is the input byte-for-byte when nothing was found).
func composeRow(body string) (string, []arrowLine) {
	arrows := scanArrowLines(swiftTitle, body)
	return composeArrowLines(swiftTitle, body, arrows), arrows
}

// AC-SW-001 — every corpus row, by count and by sides; and every zero row
// composes to its input byte-for-byte, because a count alone passes on a
// scanner that found zero for the wrong reason (AC-SW-001e3, Definition of
// Done).
func TestSwiftArrowLines(t *testing.T) {
	if len(swiftCorpus) != 42 {
		t.Fatalf("corpus holds %d rows, acceptance.md records 42", len(swiftCorpus))
	}
	for _, r := range swiftCorpus {
		t.Run(fmt.Sprintf("row_%02d", r.row), func(t *testing.T) {
			composed, arrows := composeRow(r.body)
			got := arrowSides(r.body, arrows)
			if len(got) != len(r.sides) || (len(got) > 0 && !reflect.DeepEqual(got, r.sides)) {
				t.Fatalf("arrow lines = %q, want %q", got, r.sides)
			}
			if len(r.sides) == 0 && composed != r.body {
				t.Errorf("a body with no arrow line was changed by composition\n got: %q\nwant: %q", composed, r.body)
			}
			if len(r.sides) > 0 && composed == r.body {
				t.Errorf("a body with %d arrow line(s) was composed to itself", len(r.sides))
			}
		})
	}

	t.Run("a_the_three_tokens_are_each_recognized", func(t *testing.T) {
		for row, want := range map[int]string{1: ":->", 3: ":<-", 4: ":<->"} {
			r := corpusRow(t, row)
			arrows := scanArrowLines(swiftTitle, r.body)
			if len(arrows) != 1 || arrows[0].token != want {
				t.Errorf("row %d: arrows = %+v, want one %q", row, arrows, want)
			}
		}
	})

	t.Run("b_a_second_token_is_right_side_text_and_unemphasized", func(t *testing.T) {
		composed, _ := composeRow(corpusRow(t, 5).body)
		html := renderFragment(composed)
		if strings.Count(html, "<b ") != 1 {
			t.Errorf("rendered = %q, want exactly one emphasis element", html)
		}
		if !strings.Contains(html, "B :-&gt; C}}") {
			t.Errorf("rendered = %q, want the second token as escaped right-side text inside the marker", html)
		}
	})

	t.Run("e_list_content_renders_as_one_intact_list", func(t *testing.T) {
		for _, row := range []int{10, 11, 21} {
			composed, _ := composeRow(corpusRow(t, row).body)
			html := renderFragment(composed)
			if strings.Count(html, "<li>")+strings.Count(html, "<dt>") != 1 {
				t.Errorf("row %d: rendered = %q, want one intact list item", row, html)
			}
		}
		html := renderFragment(corpusRow(t, 21).body)
		if !strings.Contains(html, "<li>수도\n도쿄 :-&gt; 일본</li>") {
			t.Errorf("row 21: rendered = %q, want one <li> holding both lines", html)
		}
	})

	t.Run("f_context_renders_unchanged_outside_every_marker", func(t *testing.T) {
		composed, _ := composeRow(corpusRow(t, 14).body)
		if !strings.HasPrefix(composed, "문맥 한 줄\n") {
			t.Errorf("row 14: composed = %q, want the prose line first and unchanged", composed)
		}
		composed, _ = composeRow(corpusRow(t, 17).body)
		if !strings.HasSuffix(composed, "\n\n- 참고 사항") {
			t.Errorf("row 17: composed = %q, want the list below unchanged", composed)
		}
		html := renderFragment(composed)
		if strings.Contains(html, "children-list") || !strings.Contains(html, "<li>참고 사항</li>") {
			t.Errorf("row 17: rendered = %q, want an ordinary list carrying no marker and no container", html)
		}
	})

	// § 5 — pinned, not repaired. The measured broken output is asserted so it
	// cannot change silently; a later card repairing the class changes these
	// two rows deliberately.
	t.Run("g_inline_constructs_are_pinned_broken", func(t *testing.T) {
		composed, _ := composeRow(corpusRow(t, 31).body)
		html := renderFragment(composed)
		if !regexp.MustCompile(`<code[^>]*>[^<]*@@html:`).MatchString(html) {
			t.Errorf("row 31: rendered = %q, want the literal @@html: inside a <code> element", html)
		}

		composed, _ = composeRow(corpusRow(t, 32).body)
		html = renderFragment(composed)
		if !strings.Contains(html, "{{c1::") || !strings.Contains(html, "}}") {
			t.Fatalf("row 32: rendered = %q, want both halves of the marker present", html)
		}
		for _, element := range regexp.MustCompile(`(?s)<p>.*?</p>`).FindAllString(html, -1) {
			open := strings.Index(element, "{{c1::")
			if open >= 0 && strings.Contains(element[open:], "}}") {
				t.Errorf("row 32: element %q pairs the marker, want its halves in different block elements", element)
			}
		}
	})
}

// swiftMirrorCase is a body beyond the corpus that pins one mirrored parser
// rule, each measured against go-org v1.9.1.
type swiftMirrorCase struct {
	name  string
	body  string
	count int
}

// The mirrored rules of REQ-SW-001.5/.6, each with its converse. Every case
// here agrees with go-org: TestSwiftScannerAgreesWithTheParser checks that
// claim mechanically by rendering the uncomposed body.
var swiftMirrorCases = []swiftMirrorCase{
	// The footnote definition (plan.md DD-4, R4-1).
	{"fn_def_line_inside_a_nested_src_block_does_not_end_it", "[fn:1] 메모\n#+BEGIN_SRC org\n[fn:2] x\n#+END_SRC\n도쿄 :-> 일본", 0},
	{"fn_def_line_inside_a_nested_example_block_does_not_end_it", "[fn:1] 메모\n#+BEGIN_EXAMPLE\n[fn:2] x\n#+END_EXAMPLE\n도쿄 :-> 일본", 0},
	{"fn_def_line_inside_a_block_opens_nothing", "#+BEGIN_SRC org\n[fn:2] x\n#+END_SRC\n도쿄 :-> 일본", 1},
	{"blank_pair_inside_a_nested_quote_does_not_end_it", "[fn:1] 메모\n#+BEGIN_QUOTE\n\n\n#+END_QUOTE\n도쿄 :-> 일본", 0},
	{"blank_pair_inside_a_nested_environment_does_not_end_it", "[fn:1] 메모\n\\begin{align}\n\n\n\\end{align}\n도쿄 :-> 일본", 0},
	{"blank_pair_inside_a_nested_list_ends_it", "[fn:1] 메모\n- a\n\n\n도쿄 :-> 일본", 1},
	{"blank_pair_inside_a_nested_drawer_ends_it", "[fn:1] 메모\n:NOTES:\n\n\n:END:\n도쿄 :-> 일본", 1},
	{"one_blank_line_does_not_end_it", "[fn:1] 메모\n\n도쿄 :-> 일본", 0},
	{"a_bare_label_line_and_one_blank", "[fn:1]\n\n도쿄 :-> 일본", 0},
	{"a_bare_label_line_and_two_blanks", "[fn:1]\n\n\n도쿄 :-> 일본", 1},
	{"another_definition_continues_the_hiding", "[fn:1] a\n[fn:2] b\n도쿄 :-> 일본", 0},
	{"a_korean_label_is_not_a_definition", "[fn:메모] 노트\n도쿄 :-> 일본", 1},
	{"no_space_after_the_label_is_not_a_definition", "[fn:1]노트\n도쿄 :-> 일본", 1},
	{"an_indented_label_is_not_a_definition", " [fn:1] 노트\n도쿄 :-> 일본", 1},
	{"a_hyphen_underscore_label_alone_is_a_definition", "[fn:a-b_1]\n도쿄 :-> 일본", 0},

	// The result attached to a source block.
	{"exports_none_suppresses", "#+BEGIN_SRC sh :exports none\necho\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본", 0},
	{"one_blank_after_results_still_suppresses", "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n#+RESULTS:\n\n도쿄 :-> 일본", 0},
	{"two_blanks_after_results_do_not", "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n#+RESULTS:\n\n\n도쿄 :-> 일본", 1},
	{"a_closed_results_drawer_is_suppressed_whole", "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n#+RESULTS:\n:RESULTS:\n도쿄 :-> 일본\n:END:\n서울 :-> 한국", 1},
	{"an_unclosed_results_drawer_runs_to_the_end", "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n#+RESULTS:\n:RESULTS:\n도쿄 :-> 일본\n\n\n서울 :-> 한국", 0},
	{"a_non_blank_line_before_results_detaches_it", "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n메모\n#+RESULTS:\n도쿄 :-> 일본", 1},
	{"an_example_block_has_no_result", "#+BEGIN_EXAMPLE :exports code\necho\n#+END_EXAMPLE\n#+RESULTS:\n도쿄 :-> 일본", 1},
	{"uppercase_value_is_not_code", "#+BEGIN_SRC sh :exports CODE\necho\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본", 1},
	{"uppercase_key_is_not_exports", "#+BEGIN_SRC sh :EXPORTS code\necho\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본", 1},
	{"a_header_with_no_language_token", "#+BEGIN_SRC :exports code\necho\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본", 1},
	{"exports_results_renders", "#+BEGIN_SRC sh :exports results\necho\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본", 1},
	{"exports_both_renders", "#+BEGIN_SRC sh :exports both\necho\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본", 1},
	{"lowercase_delimiters_suppress", "#+begin_src sh :exports code\necho\n#+end_src\n#+results:\n도쿄 :-> 일본", 0},
	{"an_empty_source_block_still_carries_a_result", "#+BEGIN_SRC sh :exports code\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본", 0},
	{"an_affiliated_keyword_joins_the_result", "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n#+RESULTS:\n#+NAME: r\n도쿄 :-> 일본", 0},
	{"a_nested_source_block_as_the_result", "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n#+RESULTS:\n#+BEGIN_SRC sh\necho\n#+END_SRC\n#+RESULTS:\n도쿄 :-> 일본", 0},
	{"a_list_as_the_result_ends_at_the_list", "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n#+RESULTS:\n- a\n도쿄 :-> 일본", 1},
	{"a_quote_block_as_the_result_ends_at_its_close", "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n#+RESULTS:\n#+BEGIN_QUOTE\nx\n#+END_QUOTE\n도쿄 :-> 일본", 1},
	{"an_environment_as_the_result_ends_at_its_close", "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n#+RESULTS:\n\\begin{align}\nx\n\\end{align}\n도쿄 :-> 일본", 1},
	{"results_as_the_last_line_attaches_nothing", "도쿄 :-> 일본\n#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n#+RESULTS:", 1},

	// The LaTeX environment.
	{"uppercase_keywords_open_an_environment", "\\BEGIN{align}\n도쿄 :-> 일본\n\\END{align}", 0},
	{"the_closing_name_compares_without_case", "\\begin{align}\n도쿄 :-> 일본\n\\end{ALIGN}", 0},
	{"a_different_closing_name_opens_nothing", "\\begin{align}\n도쿄 :-> 일본\n\\end{equation}", 1},
	{"a_starred_name_is_a_name", "\\begin{align*}\n도쿄 :-> 일본\n\\end{align*}", 0},

	// The property drawer.
	{"a_lowercase_property_drawer_is_suppressed", "문맥\n:properties:\n:X: 도쿄 :-> 일본\n:end:", 0},
	{"a_blank_line_inside_opens_nothing", "문맥\n:PROPERTIES:\n\n:X: 도쿄 :-> 일본\n:END:", 1},
	{"a_missing_end_opens_nothing", "문맥\n:PROPERTIES:\n:X: 도쿄 :-> 일본", 1},

	// Context-free shapes that open a construct over the next line. Read on
	// its own each of these is an ordinary paragraph — the parse fails and
	// falls back to text — while in context its arrow renders to nothing.
	{"a_results_keyword_carrying_the_arrow", "#+RESULTS: a :-> b\nfoo", 0},
	{"a_name_keyword_carrying_the_arrow", "#+NAME: A :-> B\nfoo", 0},
	{"a_caption_keyword_carrying_the_arrow", "#+CAPTION: A :-> B\nfoo", 0},
	{"a_block_header_carrying_the_arrow", "#+BEGIN_SRC go :-> x\ny\n#+END_SRC", 0},
}

// The cases whose scanner outcome deliberately DIVERGES from go-org, each in
// the over-excluding direction: a withheld card, never a hidden marker.
var swiftDisclosedDivergences = []swiftMirrorCase{
	// spec.md § 5, the three interleavings under REQ-SW-001.6's union.
	{"environment_first_interleaving_over_excludes", "\\begin{align}\n#+BEGIN_QUOTE\n\\end{align}\nA :-> B\n#+END_QUOTE", 0},
	{"source_first_interleaving_over_excludes", "#+BEGIN_SRC go\n\\begin{align}\n#+END_SRC\nA :-> B\n\\end{align}", 0},
	// The quote-first shape AGREES with go-org (the arrow line is raw
	// environment interior there too); listed here beside its two siblings.
	{"quote_first_interleaving_agrees", "#+BEGIN_QUOTE\n\\begin{align}\n#+END_QUOTE\nA :-> B\n\\end{align}", 0},
	// The property-drawer rule is computed on its own (REQ-SW-001.6), so a
	// `:PROPERTIES:` line inside an ordinary drawer still opens one here,
	// while go-org renders it as text.
	{"a_property_drawer_inside_an_ordinary_drawer_over_excludes", ":NOTES:\n:PROPERTIES:\n:X: 도쿄 :-> 일본\n:END:", 0},
	// The list walk is SPEC-ANKICARD-003's, which reads a column-zero `*` as
	// a bullet; a body never holds a headline, because the scan cuts it at the
	// next one, so this costs nothing reachable.
	{"a_line_after_a_headline_shaped_line_over_excludes", "* A\n  B :-> C", 0},
	// An opener the parser rejects falls back to text in place, but the
	// line's own token is not plain text, so it is not admitted.
	{"an_unclosed_block_header_carrying_the_arrow", "#+BEGIN_SRC go :-> x", 0},
}

func TestSwiftMirroredRules(t *testing.T) {
	for _, c := range append(append([]swiftMirrorCase{}, swiftMirrorCases...), swiftDisclosedDivergences...) {
		t.Run(c.name, func(t *testing.T) {
			composed, arrows := composeRow(c.body)
			if len(arrows) != c.count {
				t.Fatalf("arrow lines = %q, want %d", arrowSides(c.body, arrows), c.count)
			}
			if c.count == 0 && composed != c.body {
				t.Errorf("composition changed a body with no arrow line\n got: %q", composed)
			}
		})
	}

	t.Run("a_headline_ends_a_footnote_definition", func(t *testing.T) {
		// Unreachable from a real body, which the scan cuts at the next
		// headline; mirrored because it is one of go-org's three end
		// conditions. The line below the headline is not list content
		// either, being less indented than the bullet-shaped headline's text.
		if arrows := scanArrowLines(swiftTitle, "[fn:1] 메모\n* 제목\n도쿄 :-> 일본"); len(arrows) != 1 {
			t.Errorf("arrows = %+v, want the line below the headline", arrows)
		}
	})

	t.Run("a_non_breaking_space_side_is_empty", func(t *testing.T) {
		// The expression's `\s` is ASCII-only and admits it; the Unicode
		// trim does not (REQ-SW-001.4).
		if arrows := scanArrowLines(swiftTitle, "A :->\u00a0"); len(arrows) != 0 {
			t.Errorf("arrows = %+v, want none", arrows)
		}
	})
}

// renderedInPlace reports whether go-org renders a line's text inside a
// top-level paragraph, at its own position — the property REQ-SW-001.5
// defines an arrow line by. It is the test's independent oracle: the scanner
// mirrors go-org's rules, and this renders the body with go-org itself.
func renderedInPlace(body, line string) bool {
	html := renderFragment(body)
	escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(strings.TrimSpace(line))
	at := strings.Index(html, escaped)
	if at < 0 {
		return false
	}
	var stack []string
	for _, tag := range regexp.MustCompile(`<(/?)([a-z0-9]+)[^>]*>`).FindAllStringSubmatch(html[:at], -1) {
		if tag[1] == "" {
			stack = append(stack, tag[2])
		} else if len(stack) > 0 {
			stack = stack[:len(stack)-1]
		}
	}
	return len(stack) == 1 && stack[0] == "p"
}

// The scanner is falsifiable: for every corpus row and mirror case whose
// outcome is decided by PLACEMENT, the count it reports matches what go-org
// renders in place. The rows left out are decided by another rule and are
// measured elsewhere: 7, 8, 9, and 20 by the sides (REQ-SW-001.3, .4); 18, 19,
// and 24 by the marker extent (REQ-SW-005.1); 13 is the stated cost of the
// one block rule; 31 and 32 are § 5's inline exclusion; and the two-arrow rows
// have no single line to locate.
var swiftRowsNotDecidedByPlacement = map[int]bool{
	7: true, 8: true, 9: true, 13: true, 18: true, 19: true, 20: true, 24: true, 31: true, 32: true,
}

func TestSwiftScannerAgreesWithTheParser(t *testing.T) {
	check := func(t *testing.T, body string) {
		t.Helper()
		count := len(scanArrowLines(swiftTitle, body))
		lines := splitLines(body)
		var arrowText string
		for _, ln := range lines {
			if strings.Contains(ln.text, ":-") || strings.Contains(ln.text, ":<") {
				arrowText = ln.text
			}
		}
		if inPlace := renderedInPlace(body, arrowText); inPlace != (count > 0) {
			t.Errorf("go-org renders %q in place = %v, scanner count = %d", arrowText, inPlace, count)
		}
	}
	for _, r := range swiftCorpus {
		if swiftRowsNotDecidedByPlacement[r.row] || len(r.sides) > 1 {
			continue
		}
		t.Run(fmt.Sprintf("row_%02d", r.row), func(t *testing.T) { check(t, r.body) })
	}
	for _, c := range swiftMirrorCases {
		t.Run(c.name, func(t *testing.T) { check(t, c.body) })
	}
}

// AC-SW-003 — positional numbering, restarted per entry, counted over arrow
// lines only.
func TestSwiftNumbering(t *testing.T) {
	t.Run("two_consecutive_lines", func(t *testing.T) {
		composed, _ := composeRow(corpusRow(t, 2).body)
		lines := strings.Split(composed, "\n")
		if !strings.Contains(lines[0], "{{c1::일본}}") || !strings.Contains(lines[1], "{{c3::한국}}") {
			t.Errorf("composed = %q, want c1 on line 1 and c3 on line 2", composed)
		}
		if strings.Contains(composed, "{{c2::") || strings.Contains(composed, "{{c4::") {
			t.Errorf("composed = %q, want c2 and c4 consumed by the unwrapped left sides", composed)
		}
	})

	t.Run("the_count_restarts_per_entry", func(t *testing.T) {
		first, _ := composeRow(corpusRow(t, 2).body)
		second, _ := composeRow(corpusRow(t, 2).body)
		if first != second || !strings.Contains(second, "{{c1::") {
			t.Errorf("second entry = %q, want it to begin at c1 again", second)
		}
	})

	t.Run("only_arrow_lines_count", func(t *testing.T) {
		composed, _ := composeRow("도쿄 :-> 일본\n중간의 평범한 문장\n서울 :-> 한국")
		if !strings.Contains(composed, "{{c3::한국}}") {
			t.Errorf("composed = %q, want the second arrow line numbered c3", composed)
		}
	})
}

// AC-SW-004 — the token selects which sides are wrapped.
func TestSwiftDirectionTable(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{"A :-> B", `A @@html:<b class="swift-arrow">:-&gt;</b>@@ {{c1::B}}`},
		{"A :<- B", `{{c2::A}} @@html:<b class="swift-arrow">:&lt;-</b>@@ B`},
		{"A :<-> B", `{{c2::A}} @@html:<b class="swift-arrow">:&lt;-&gt;</b>@@ {{c1::B}}`},
	} {
		if got, _ := composeRow(c.body); got != c.want {
			t.Errorf("compose(%q)\n got: %q\nwant: %q", c.body, got, c.want)
		}
	}
}

// AC-SW-005 — a hand-written marker is never disturbed.
func TestSwiftHandWrittenMarker(t *testing.T) {
	t.Run("a_marker_spanning_the_arrow", func(t *testing.T) {
		for _, row := range []int{18, 19, 24} {
			body := corpusRow(t, row).body
			composed, err := composeSwift(swiftTitle, body)
			var missing *SwiftArrowMissingError
			if !errors.As(err, &missing) {
				t.Errorf("row %d: err = %v, want *SwiftArrowMissingError", row, err)
			}
			if strings.Contains(composed, "{{c2::") {
				t.Errorf("row %d: composed = %q, want no generated marker anywhere", row, composed)
			}
		}
	})

	t.Run("a_marker_wholly_on_one_side", func(t *testing.T) {
		composed, _ := composeRow(corpusRow(t, 16).body)
		want := `{{c1::A}} @@html:<b class="swift-arrow">:-&gt;</b>@@ {{c2::B}}`
		if composed != want {
			t.Errorf("composed = %q, want %q", composed, want)
		}
	})

	t.Run("the_offset_applies_to_the_whole_formula", func(t *testing.T) {
		composed, _ := composeRow("메모 {{c5::x}}\n도쿄 :-> 일본\n서울 :-> 한국")
		if !strings.Contains(composed, "{{c6::일본}}") || !strings.Contains(composed, "{{c8::한국}}") {
			t.Errorf("composed = %q, want c6 then c8", composed)
		}
	})

	t.Run("a_marker_in_the_title_raises_the_base", func(t *testing.T) {
		arrows := scanArrowLines("{{c3::제목}}", "도쿄 :-> 일본")
		if got := composeArrowLines("{{c3::제목}}", "도쿄 :-> 일본", arrows); !strings.Contains(got, "{{c4::일본}}") {
			t.Errorf("composed = %q, want c4", got)
		}
	})

	t.Run("an_unclosed_marker_in_the_title_spans_the_body", func(t *testing.T) {
		if arrows := scanArrowLines("{{c1::제목", "도쿄 :-> 일본"); len(arrows) != 0 {
			t.Errorf("arrows = %+v, want none — a generated }} would close the title's opening", arrows)
		}
	})

	t.Run("every_selected_side_already_marked", func(t *testing.T) {
		body := "도쿄 :-> {{c1::일본}}\n{{c4::서울}} :<-> {{c3::한국}}"
		composed, err := composeSwift(swiftTitle, body)
		if err != nil {
			t.Fatalf("err = %v, want none — the entry is not skipped", err)
		}
		for _, marker := range []string{"{{c1::일본}}", "{{c4::서울}}", "{{c3::한국}}"} {
			if !strings.Contains(composed, marker) {
				t.Errorf("composed = %q, want the author's %s unchanged", composed, marker)
			}
		}
		if n := len(clozeNumberPattern.FindAllString(composed, -1)); n != 3 {
			t.Errorf("composed = %q carries %d markers, want the author's 3 alone", composed, n)
		}
	})
}

// AC-SW-006 — a generated marker cannot close early, by the one shared rule.
func TestSwiftBraceSafety(t *testing.T) {
	composed, _ := composeRow(`x :-> \sqrt{a^{2}}`)
	if !strings.HasSuffix(composed, `{{c1::\sqrt{a^{2} } }}`) {
		t.Errorf("composed = %q, want the separated form with one space before the close", composed)
	}

	// The same fixture the editor is compared against (AC-ML-007c): the swift
	// path reaches the rule through its one implementation.
	for _, c := range loadBraceFixture(t) {
		if strings.TrimSpace(c.In) != c.In || hasClozeMarker(c.In) {
			continue
		}
		got, _ := composeRow("Q :-> " + c.In)
		if want := "{{c1::" + c.Want + "}}"; !strings.HasSuffix(got, want) {
			t.Errorf("compose(%q)\n got: %q\nwant suffix: %q", c.In, got, want)
		}
	}
}

// AC-SW-007 — the arrow renders as a real, classed <b>.
func TestSwiftArrowEmphasis(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{"A :-> B", `<b class="swift-arrow">:-&gt;</b>`},
		{"A :<- B", `<b class="swift-arrow">:&lt;-</b>`},
		{"A :<-> B", `<b class="swift-arrow">:&lt;-&gt;</b>`},
	} {
		composed, _ := composeRow(c.body)
		html := renderFragment(composed)
		if !strings.Contains(html, c.want) || strings.Contains(html, "&lt;b") {
			t.Errorf("render(%q) = %q, want %q and no escaped tag", c.body, html, c.want)
		}
	}
}

// AC-SW-007's bound, decided by a test rather than by reading a grep: the
// snippet table is the only producer of an `@@html:` string in the package,
// its entries are exactly the four constants, and no corpus composition emits
// an `@@html:` outside them (REQ-SW-007.3).
func TestSwiftSnippetBound(t *testing.T) {
	want := map[string]bool{
		`@@html:<b class="swift-arrow">:-&gt;</b>@@`:     true,
		`@@html:<b class="swift-arrow">:&lt;-</b>@@`:     true,
		`@@html:<b class="swift-arrow">:&lt;-&gt;</b>@@`: true,
		`@@html:<br>@@`: true,
	}
	got := map[string]bool{swiftLineBreak: true}
	for _, snippet := range swiftArrowSnippets {
		got[snippet] = true
	}
	if !reflect.DeepEqual(got, want) || len(swiftArrowSnippets) != 3 {
		t.Fatalf("snippet table = %q + %q, want exactly the four constants", swiftArrowSnippets, swiftLineBreak)
	}

	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	occurrences := 0
	for _, path := range sources {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		n := strings.Count(string(raw), "@@html:")
		if n > 0 && path != "swift.go" {
			t.Errorf("%s carries %d `@@html:` literal(s); only the swift table may", path, n)
		}
		occurrences += n
	}
	if occurrences != 4 {
		t.Errorf("the package's sources carry %d `@@html:` literals, want the table's 4", occurrences)
	}

	strip := strings.NewReplacer(
		`@@html:<b class="swift-arrow">:-&gt;</b>@@`, "",
		`@@html:<b class="swift-arrow">:&lt;-</b>@@`, "",
		`@@html:<b class="swift-arrow">:&lt;-&gt;</b>@@`, "",
		`@@html:<br>@@`, "")
	for _, r := range swiftCorpus {
		composed, _ := composeRow(r.body)
		if strings.Contains(strip.Replace(composed), "@@html:") {
			t.Errorf("row %d composed an `@@html:` outside the table: %q", r.row, composed)
		}
	}
}

// AC-SW-011 — a break between consecutive arrow lines and nowhere else, and
// no line inserted by composition.
func TestSwiftLineBreak(t *testing.T) {
	t.Run("consecutive_arrow_lines", func(t *testing.T) {
		composed, _ := composeRow(corpusRow(t, 2).body)
		html := renderFragment(composed)
		if !strings.Contains(html, "{{c1::일본}}<br>\n서울") {
			t.Errorf("rendered = %q, want a <br> between the two arrow lines", html)
		}
		if strings.Count(composed, swiftLineBreak) != 1 {
			t.Errorf("composed = %q, want exactly one break, after the first line", composed)
		}
	})

	t.Run("prose_below_an_arrow_line", func(t *testing.T) {
		composed, _ := composeRow(corpusRow(t, 23).body)
		if strings.Contains(composed, swiftLineBreak) || !strings.HasSuffix(composed, "\n참고: 성립 조건") {
			t.Errorf("composed = %q, want no break and the prose unchanged", composed)
		}
		html := renderFragment(composed)
		if !strings.Contains(html, "{{c1::일본}}\n참고: 성립 조건</p>") {
			t.Errorf("rendered = %q, want the prose in the same paragraph, unclassed", html)
		}
	})

	t.Run("blank_separated_arrow_lines", func(t *testing.T) {
		composed, _ := composeRow(corpusRow(t, 15).body)
		if strings.Contains(composed, swiftLineBreak) {
			t.Errorf("composed = %q, want no break across a blank line", composed)
		}
	})

	t.Run("composition_inserts_no_line", func(t *testing.T) {
		for _, r := range swiftCorpus {
			composed, _ := composeRow(r.body)
			if strings.Count(composed, "\n") != strings.Count(r.body, "\n") || strings.Contains(composed, "#+ATTR_HTML") {
				t.Errorf("row %d: composed = %q, want the same lines and no attribute line", r.row, composed)
			}
		}
	})
}

// AC-SW-011 — the emphasis element carries the class the stylesheet targets.
func TestSwiftArrowClass(t *testing.T) {
	for token, snippet := range swiftArrowSnippets {
		if !strings.Contains(snippet, `<b class="swift-arrow">`) {
			t.Errorf("snippet for %q = %q, want the swift-arrow class", token, snippet)
		}
	}
}

// REQ-SW-002 — an entry whose body yields no arrow line is reported.
func TestSwiftComposeReportsAMissingArrow(t *testing.T) {
	for _, body := range []string{"", "평범한 본문", "- 도쿄 :-> 일본"} {
		_, err := composeSwift(swiftTitle, body)
		var missing *SwiftArrowMissingError
		if !errors.As(err, &missing) {
			t.Errorf("composeSwift(%q) err = %v, want *SwiftArrowMissingError", body, err)
		}
	}
	if (&SwiftArrowMissingError{}).Error() == "" {
		t.Error("the error carries no message")
	}
}
