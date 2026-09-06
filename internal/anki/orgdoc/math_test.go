package orgdoc

import (
	"strings"
	"testing"
)

// renderBody is the shared arrangement for the math table tests: a Basic entry
// whose Back field carries the fragment under test. Asserting on Back rather
// than on a hand-built HTML string keeps every case exercising the real
// go-org-then-transform pipeline the planner hashes (spec.md REQ-C-016).
func renderBody(t *testing.T, body string) string {
	t.Helper()
	fields, err := Render(NoteTypeBasic, "T", body)
	if err != nil {
		t.Fatalf("Render(Basic, %q, %q) returned unexpected error: %v", "T", body, err)
	}
	return fields["Back"]
}

// TestRender_Math_ConvertsTheFourOrgSyntaxesToAnkiDelimiters is AC-C-008: the
// delimiter table of spec.md REQ-C-010.1, asserted as byte-exact output rather
// than substring containment so a stray leftover `$` or a doubled delimiter
// cannot pass. The final sub-assertion carries AC-C-008's "no other delimiter
// form appears anywhere in the output" clause: after conversion no `$` may
// remain, because every `$` form the table admits has been rewritten.
func TestRender_Math_ConvertsTheFourOrgSyntaxesToAnkiDelimiters(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "single dollar becomes the inline MathJax pair",
			body: `$E = mc^2$`,
			want: "<p>\\(E = mc^2\\)</p>\n",
		},
		{
			name: "double dollar becomes the display MathJax pair",
			body: `$$E = mc^2$$`,
			want: "<p>\\[E = mc^2\\]</p>\n",
		},
		{
			name: "an already-inline fragment is emitted unchanged",
			body: `\(E = mc^2\)`,
			want: "<p>\\(E = mc^2\\)</p>\n",
		},
		{
			name: "an already-display fragment is emitted unchanged",
			body: `\[E = mc^2\]`,
			want: "<p>\\[E = mc^2\\]</p>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBody(t, tt.body)
			if got != tt.want {
				t.Errorf("Back = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "$") {
				t.Errorf("Back = %q, want no `$` delimiter left in the output (AC-C-008: no other delimiter form appears)", got)
			}
		})
	}
}

// TestRender_Math_InlineHeuristicRejectsNonMathDollars is AC-C-009a plus the
// remaining rejection grounds of the Org inline-math heuristic (spec.md § 2,
// consumed by REQ-C-010.2). Each row asserts the rendered output is byte-equal
// to what the renderer alone produces — the transform must be a no-op, not a
// partial rewrite. The rule is applied as written: none of these rows is
// rejected because a character "looks like currency".
func TestRender_Math_InlineHeuristicRejectsNonMathDollars(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			// AC-C-009a. Two independent grounds: the enclosed text `5 and `
			// is not attached to the closing `$`, and that `$` is followed by
			// `7` rather than by whitespace or non-dash punctuation.
			name: "currency amounts stay verbatim and produce no MathJax pair",
			body: "costs $5 and $7 total",
			want: "<p>costs $5 and $7 total</p>\n",
		},
		{
			name: "a closing dollar followed by a dash is rejected",
			body: "a $x$- b",
			want: "<p>a $x$- b</p>\n",
		},
		{
			name: "more than two line breaks inside the candidate is rejected",
			body: "$a\nb\nc\nd$",
			want: "<p>$a\nb\nc\nd$</p>\n",
		},
		{
			name: "text detached from the opening dollar is rejected",
			body: "$ x$",
			want: "<p>$ x$</p>\n",
		},
		{
			name: "text detached from the closing dollar is rejected",
			body: "$x $",
			want: "<p>$x $</p>\n",
		},
		{
			name: "an unclosed display-dollar run stays verbatim",
			body: "$$x unclosed",
			want: "<p>$$x unclosed</p>\n",
		},
		{
			name: "an unclosed inline fragment stays verbatim",
			body: `\(x unclosed`,
			want: "<p>\\(x unclosed</p>\n",
		},
		{
			name: "an unclosed display fragment stays verbatim",
			body: `\[x unclosed`,
			want: "<p>\\[x unclosed</p>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBody(t, tt.body)
			if got != tt.want {
				t.Errorf("Back = %q, want %q (the transform must leave a rejected candidate byte-unchanged)", got, tt.want)
			}
		})
	}
}

// TestRender_Math_LineBreakInsideFragmentBecomesBr is AC-C-009b (REQ-C-010.3):
// a bare newline is unrenderable by Anki's bundled MathJax, so every line break
// a converted fragment carries is emitted as `<br>`. The two-line-break row sits
// exactly on the heuristic's bound, which the rejection table's three-line-break
// row is the other side of.
func TestRender_Math_LineBreakInsideFragmentBecomesBr(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "one line break inside an inline fragment",
			body: "a $x\ny$ b",
			want: "<p>a \\(x<br>y\\) b</p>\n",
		},
		{
			name: "two line breaks sit on the heuristic bound and still convert",
			body: "$a\nb\nc$",
			want: "<p>\\(a<br>b<br>c\\)</p>\n",
		},
		{
			name: "a line break inside a display fragment",
			body: "$$a\nb$$",
			want: "<p>\\[a<br>b\\]</p>\n",
		},
		// The rows above all enter through a `$`-delimited source form. A
		// fragment the author already wrote as `\(..\)` or `\[..\]` takes a
		// different branch of the transform, and REQ-C-010.3 makes no
		// distinction between the two source forms -- so the normalization
		// is asserted on both, not inferred from the `$` case.
		{
			name: "one line break inside an already-parenthesized inline fragment",
			body: "a \\(x\ny\\) b",
			want: "<p>a \\(x<br>y\\) b</p>\n",
		},
		{
			name: "a line break inside an already-bracketed display fragment",
			body: "\\[a\nb\\]",
			want: "<p>\\[a<br>b\\]</p>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBody(t, tt.body)
			if got != tt.want {
				t.Errorf("Back = %q, want %q", got, tt.want)
			}
			if !strings.Contains(got, "<br>") {
				t.Errorf("Back = %q, want it to carry a <br> in place of the fragment's line break", got)
			}
		})
	}
}

// TestRender_Math_ProtectedRegionsKeepDollarsByteUnchanged is AC-C-009c
// (REQ-C-011). Every `$` below sits in a pair that WOULD satisfy the inline
// heuristic in ordinary prose, so each row fails loudly if the region guard is
// missing rather than passing by accident on an unconvertible candidate. The
// four protected shapes are a `<code>` region, an HTML attribute value that is
// not a link target (`alt`), a link target (`href`), and a `<pre>` region.
func TestRender_Math_ProtectedRegionsKeepDollarsByteUnchanged(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string // the byte-exact span that must survive
	}{
		{
			name: "inside a code region",
			body: "Inline ~code $x$ here~ and prose.",
			want: "<code>code $x$ here</code>",
		},
		{
			name: "inside a link target",
			body: "A [[https://e.com/?q=$y$][link]] here.",
			want: `href="https://e.com/?q=$y$"`,
		},
		{
			name: "inside a non-link attribute value",
			body: "#+ATTR_HTML: :alt costs $w$ here\n[[file:a.png]]",
			want: `alt="costs $w$ here"`,
		},
		{
			name: "inside a pre region",
			body: "#+BEGIN_EXAMPLE\n$z$\n#+END_EXAMPLE",
			want: "<pre class=\"example\">\n$z$\n</pre>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBody(t, tt.body)
			if !strings.Contains(got, tt.want) {
				t.Errorf("Back = %q, want it to contain the byte-unchanged span %q", got, tt.want)
			}
			for _, delim := range []string{`\(`, `\)`, `\[`, `\]`} {
				if strings.Contains(got, delim) {
					t.Errorf("Back = %q, want no %q — a protected `$` must not be converted", got, delim)
				}
			}
			// AC-C-009c's second clause: the transform emits no reference to a
			// network-hosted MathJax or stylesheet resource. Asserted on the
			// markup that would carry one rather than on "http", because the
			// link-target row legitimately contains an https URL of its own.
			for _, forbidden := range []string{"mathjax", "<script", "<link"} {
				if strings.Contains(strings.ToLower(got), forbidden) {
					t.Errorf("Back = %q, want no %q — the transform references no network-hosted resource", got, forbidden)
				}
			}
		})
	}
}

// TestRender_Cloze_MathDelimiterCollisionKeepsMarkerByteForByte is AC-C-023,
// the highest-risk interaction in this milestone: the cloze marker's braces and
// a MathJax fragment's braces sit adjacent, so an off-by-one in the transform
// corrupts the marker and Anki silently stops generating the card. Brace
// balance is asserted by count rather than by eye.
func TestRender_Cloze_MathDelimiterCollisionKeepsMarkerByteForByte(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "a single-dollar fragment inside a cloze marker converts without disturbing the marker",
			body: `{{c1::$x^{2}$}}`,
			want: `{{c1::\(x^{2}\)}}`,
		},
		{
			name: "an already-converted fragment inside a cloze marker survives byte-for-byte",
			body: `{{c1::\(x^{2}\)}}`,
			want: `{{c1::\(x^{2}\)}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, err := Render(NoteTypeCloze, "Math", tt.body)
			if err != nil {
				t.Fatalf("Render(Cloze, %q, %q) returned unexpected error: %v", "Math", tt.body, err)
			}
			text := fields["Text"]
			if !strings.Contains(text, tt.want) {
				t.Errorf("Text = %q, want it to contain %q", text, tt.want)
			}
			if !strings.Contains(text, "{{c1::") {
				t.Errorf("Text = %q, want the cloze marker prefix {{c1:: preserved byte-for-byte", text)
			}
			if open, closed := strings.Count(text, "{"), strings.Count(text, "}"); open != closed {
				t.Errorf("Text = %q, has %d `{` and %d `}` — braces must stay balanced", text, open, closed)
			}
			if strings.Contains(text, "$") {
				t.Errorf("Text = %q, want no `$` delimiter — Anki's cloze logic breaks on a non-standard delimiter", text)
			}
		})
	}
}

// TestRender_Math_SubSuperscriptInsideFragmentSurvives is AC-C-024, carried as
// a REGRESSION TEST rather than as an assertion of upstream behavior: research.md
// C2 records that no lens verified go-org's fix version, and the pinned
// renderer's source reading is the only evidence that `_` and `^` inside a
// fragment interior are parsed as raw text. If a future renderer bump breaks
// this, the failure here is the signal — not a claim that upstream guarantees it.
func TestRender_Math_SubSuperscriptInsideFragmentSurvives(t *testing.T) {
	const body = `\(\sum_{i=1}^n a_n\)`
	got := renderBody(t, body)
	if want := "<p>\\(\\sum_{i=1}^n a_n\\)</p>\n"; got != want {
		t.Errorf("Back = %q, want %q", got, want)
	}
	for _, el := range []string{"<sub", "<sup"} {
		if strings.Contains(got, el) {
			t.Errorf("Back = %q, want no %q element inside the fragment — the TeX interior must reach Anki intact", got, el)
		}
	}
}

// TestRender_Math_EscapedEntityInsideFragmentStaysEscaped is design.md § 10.3's
// first regression (plan.md R11): the renderer HTML-escapes raw text, so `a < b`
// reaches Anki as `a &lt; b`. MathJax reads DOM text nodes where `&lt;` has
// already been decoded, and unescaping in the source would emit invalid HTML —
// so the escaping is deliberately KEPT through the transform. Recorded as a
// regression test; the on-card rendering is verified separately.
func TestRender_Math_EscapedEntityInsideFragmentStaysEscaped(t *testing.T) {
	got := renderBody(t, "$a < b$")
	if want := "<p>\\(a &lt; b\\)</p>\n"; got != want {
		t.Errorf("Back = %q, want %q (the entity stays escaped inside the converted fragment)", got, want)
	}
}

// TestRender_Math_RawTextFallbackTreatsMarkupShapedBytesConservatively covers
// the transform on renderFragment's go-org-error fallback, which hands back raw
// Org source rather than HTML. Reached through the same documented stdlib
// boundary the existing oversized-line test uses: go-org tokenizes with a
// default bufio.Scanner whose 64KiB per-line ceiling makes a longer single line
// fail. Raw Org is not HTML, so markup-shaped bytes there are treated
// conservatively — a `<code>` run and an unterminated `<` both suppress
// conversion rather than risk rewriting something that is not prose.
func TestRender_Math_RawTextFallbackTreatsMarkupShapedBytesConservatively(t *testing.T) {
	pad := strings.Repeat("a", 70*1024) // one line, past bufio.Scanner's 64KiB limit
	tests := []struct {
		name string
		tail string
	}{
		{name: "a code region suppresses conversion", tail: "<code>$x$</code>"},
		{name: "an unterminated tag suppresses conversion", tail: "<b $x$"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBody(t, pad+tt.tail)
			if !strings.HasSuffix(got, tt.tail) {
				t.Errorf("Back tail = %q, want the raw suffix %q returned verbatim", got[len(got)-len(tt.tail):], tt.tail)
			}
			if strings.Contains(got, `\(`) {
				t.Errorf("Back = ...%q, want no inline MathJax pair in markup-shaped fallback text", got[len(got)-len(tt.tail):])
			}
		})
	}
}
