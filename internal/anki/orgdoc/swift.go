package orgdoc

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/niklasfasching/go-org/org"
)

// Swift arrow card composition (spec.md SPEC-ANKICARD-004).
//
// A heading marked ANKI_SWIFT has its remaining body read line by line for an
// arrow. Each arrow line becomes a pair of positionally numbered cloze sides,
// so one heading yields one cloze note carrying as many cards as its arrow
// lines earn. The heading title stays unwrapped above them as context.
//
// Composition is SOURCE-LEVEL and PRE-RENDER (plan.md DD-1): go-org escapes
// both angle brackets, so after rendering the tokens read `:-&gt;` and no
// longer stand in a known relation to the source a marker must be written
// into. Markers and the emphasis are written into the Org source instead.
//
// Which lines can carry an arrow is DEFINITIONAL (REQ-SW-001.5): a line the
// parser reads as ordinary paragraph text rendered in place. It is decided in
// two halves (REQ-SW-001.6). The context-free half asks go-org itself
// (isPlainTextLine). The context-dependent half mirrors, rule by rule, the
// parser's opening-and-continuation rule for each construct that can make a
// LATER line something other than in-place paragraph text — and, for the
// source-block result, the HTML writer's emission rule — composed in the fixed
// order nonParagraphLines states.

// SwiftArrowMissingError is returned by RenderWithOptions when a swift entry's
// remaining body yields no arrow line (spec.md REQ-SW-002). The caller reports
// it as a skipped entry carrying the swift_arrow_missing code, and writes
// nothing for it.
//
// It binds whatever the body does contain: a body whose every arrow sits in
// list content, inside a block, on a line with an empty side, or inside a
// hand-written marker's span yields no arrow line and is reported — a heading
// that asks for a swift card and produces none is the case this exists for.
type SwiftArrowMissingError struct{}

func (e *SwiftArrowMissingError) Error() string {
	return "swift card option is on but the entry's body carries no arrow line"
}

// --- the fixed snippet table (REQ-SW-007.3) ----------------------------------

// Composition places inside an Org export snippet ONLY a value from this table
// — the three arrow forms and the line break — and never assembles a snippet
// from a line's own text. The snippet bypasses go-org's escaping entirely, so
// text placed there would reach the card as markup; and an unrecognized
// backend name renders a snippet to nothing at all. Neither failure is visible
// in the Org buffer, which is why the bound is absolute. TestSwiftSnippetBound
// asserts these four are the package's only HTML export-snippet literals.
//
// The token inside each is HTML-escaped (REQ-SW-007.2), and the emphasis
// element carries the class the base stylesheet targets (REQ-SW-011.2).
const swiftLineBreak = "@@html:<br>@@"

var swiftArrowSnippets = map[string]string{
	":->":  `@@html:<b class="swift-arrow">:-&gt;</b>@@`,
	":<-":  `@@html:<b class="swift-arrow">:&lt;-</b>@@`,
	":<->": `@@html:<b class="swift-arrow">:&lt;-&gt;</b>@@`,
}

// --- recognizing an arrow line (REQ-SW-001.1-.4) ------------------------------

// arrowLinePattern is REQ-SW-001.1's expression. The alternation is ordered
// longest-first, so `:<->` is never read as `:<-` followed by a stray `>`; the
// order is the rule. arrowTokenPattern finds the line's FIRST token by the same
// ordering, which is what REQ-SW-001.3 compares the expression's choice with.
var (
	arrowLinePattern  = regexp.MustCompile(`(.+?)\s*(:<->|:->|:<-)\s*(.+)`)
	arrowTokenPattern = regexp.MustCompile(`:<->|:->|:<-`)
)

// arrowLine is one arrow line, as byte offsets into the remaining body. The
// two sides are already trimmed.
type arrowLine struct {
	line                 int // index into splitLines(body)
	lineEnd              int // offset of the line's end, excluding its newline
	token                string
	leftStart, leftEnd   int
	tokenStart, tokenEnd int
	rightStart, rightEnd int
}

// readArrowLine applies REQ-SW-001.1, .3, and .4 to one line.
//
// Non-greediness minimises the left side but does not pin the first token —
// `:-> B :-> C` selects the SECOND token — so a line whose selected token is
// not its first is rejected rather than read literally (REQ-SW-001.3). Each
// side is then trimmed of UNICODE whitespace: the expression's `\s` is ASCII
// only, and a side holding nothing but a non-breaking space would otherwise
// be wrapped as a blank with no answer behind it (REQ-SW-001.4).
func readArrowLine(ln sourceLine) (arrowLine, bool) {
	m := arrowLinePattern.FindStringSubmatchIndex(ln.text)
	if m == nil || arrowTokenPattern.FindStringIndex(ln.text)[0] != m[4] {
		return arrowLine{}, false
	}
	leftStart, leftEnd := trimmedSpan(ln.text, m[2], m[3])
	rightStart, rightEnd := trimmedSpan(ln.text, m[6], m[7])
	if leftStart == leftEnd || rightStart == rightEnd {
		return arrowLine{}, false
	}
	return arrowLine{
		lineEnd:    ln.end,
		token:      ln.text[m[4]:m[5]],
		leftStart:  ln.start + leftStart,
		leftEnd:    ln.start + leftEnd,
		tokenStart: ln.start + m[4],
		tokenEnd:   ln.start + m[5],
		rightStart: ln.start + rightStart,
		rightEnd:   ln.start + rightEnd,
	}, true
}

// trimmedSpan narrows text[start:end] to exclude surrounding Unicode
// whitespace, returning the narrowed offsets.
func trimmedSpan(text string, start, end int) (int, int) {
	span := text[start:end]
	trimmedLeft := strings.TrimLeftFunc(span, unicode.IsSpace)
	start += len(span) - len(trimmedLeft)
	return start, start + len(strings.TrimRightFunc(trimmedLeft, unicode.IsSpace))
}

// --- the hand-written marker's extent (REQ-SW-005.1, plan.md DD-9) -----------

// markerSpans reports the byte extent of every hand-written cloze marker in s.
//
// A span opens at `{{cN::` and closes at the first `}}` after it — Anki's own
// non-greedy rule, not a second one. An UNCLOSED opening spans to the end of
// s: on its own it makes no card, but a generated marker below it would supply
// the `}}` it lacks and bring it to life around content the author never
// marked.
//
// This is an EXTENT scan on purpose. hasClozeMarker answers "is there a marker
// somewhere", which cannot decide "does this offset sit inside one".
func markerSpans(s string) [][2]int {
	var spans [][2]int
	for at := 0; ; {
		open := clozeMarkerPattern.FindStringIndex(s[at:])
		if open == nil {
			return spans
		}
		closeAt := strings.Index(s[at+open[1]:], "}}")
		if closeAt < 0 {
			return append(spans, [2]int{at + open[0], len(s)})
		}
		end := at + open[1] + closeAt + len("}}")
		spans = append(spans, [2]int{at + open[0], end})
		at = end
	}
}

func insideAnySpan(spans [][2]int, offset int) bool {
	for _, span := range spans {
		if span[0] <= offset && offset < span[1] {
			return true
		}
	}
	return false
}

// --- the context-free half (REQ-SW-001.6) -------------------------------------

// isPlainTextLine reports whether go-org lexes a line as plain paragraph text.
//
// The parser itself answers: the line is placed under a fixed text line, and it
// is plain text exactly when the parser continues that paragraph through it. A
// table row, a heading, a keyword, a comment, a fixed-width line, a footnote
// definition, a list bullet, or a construct opener each stops the paragraph and
// yields a second node.
//
// Parsing the line ALONE is not enough, and the difference is measured: a lone
// opener the parser then fails to complete falls back to a paragraph, so
// `#+RESULTS: A :-> B`, `#+NAME: A :-> B`, or `#+BEGIN_SRC A :-> B` read alone
// are paragraphs — while followed by a line, the first two attach to it and
// render their own text to nothing, and the third becomes a block header. The
// fall-back case is withheld too, which over-excludes and never hides a marker.
func isPlainTextLine(text string) bool {
	doc := org.New().Silent().Parse(strings.NewReader("x\n"+text), "./")
	return len(doc.Nodes) == 1
}

// --- the context-dependent half (REQ-SW-001.6) --------------------------------

// The delimiter grammar of each mirrored construct, verbatim from go-org
// v1.9.1: org/block.go (the LaTeX environment, the source-block result),
// org/footnote.go, org/drawer.go, and org/headline.go.
var (
	beginLatexLinePattern         = regexp.MustCompile(`(?i)^(\s*)\\begin{([^}]+)}(\s*)$`)
	endLatexLinePattern           = regexp.MustCompile(`(?i)^(\s*)\\end{([^}]+)}(\s*)$`)
	footnoteDefinitionLinePattern = regexp.MustCompile(`^\[fn:([\w-]+)\](\s+(.+)|\s*$)`)
	headlineLinePattern           = regexp.MustCompile(`^([*]+)\s+(.*)`)
	beginDrawerLinePattern        = regexp.MustCompile(`^(\s*):(\S+):\s*$`)
	endDrawerLinePattern          = regexp.MustCompile(`(?i)^(\s*):END:\s*$`)
	propertyLinePattern           = regexp.MustCompile(`^(\s*):(\S+):(\s+(.*)$|$)`)
	resultLinePattern             = regexp.MustCompile(`(?i)^(\s*)#\+RESULTS:`)
)

// nonParagraphLines reports, for every line of a remaining body, whether a
// construct opened on an earlier line makes it something other than in-place
// paragraph text.
//
// The mirrored rules compose in the one fixed order REQ-SW-001.6 states:
//
//  1. Block regions (the package's one `#+BEGIN_` pairing) and LaTeX
//     environment regions are each computed independently over the whole body.
//  2. The footnote-definition rule and the source-block-result rule then run
//     over the body, reading those regions.
//  3. The property-drawer and list rules are computed on their own.
//
// A line is excluded when ANY rule excludes it. Where a block and an
// environment interleave rather than nest, the union over-excludes against
// go-org — a withheld card, never a hidden marker (spec.md § 5).
func nonParagraphLines(lines []sourceLine) []bool {
	excluded := make([]bool, len(lines))
	blocks, _ := pairedBlocks(lines)
	environments := latexEnvironments(lines)

	region := make([]bool, len(lines))
	for _, span := range append(append([]blockSpan{}, blocks...), environments...) {
		markLines(region, span.begin, span.end)
		markLines(excluded, span.begin, span.end)
	}

	excludeFootnoteDefinitions(lines, region, excluded)
	for _, block := range blocks {
		if block.name == "SRC" && exportsSuppressResult(lines[block.begin].text) {
			excludeAttachedResult(lines, block, blocks, environments, region, excluded)
		}
	}

	excludePropertyDrawers(lines, excluded)
	excludeListContent(lines, excluded)
	return excluded
}

func markLines(marks []bool, from, to int) {
	for i := from; i <= to; i++ {
		marks[i] = true
	}
}

// latexEnvironments pairs `\begin{NAME}` and `\end{NAME}` lines by go-org's own
// rule: each delimiter alone on its line, the whole line matched without
// regard to case, closed by the next `\end` of the same NAME. An opening with
// no matching close opens nothing and renders as text.
//
// This is a second DELIMITER, not a second copy of the `#+BEGIN_` rule — the
// delimiters and the name capture differ — so no line is paired by two rules
// that could disagree about it.
func latexEnvironments(lines []sourceLine) []blockSpan {
	var spans []blockSpan
	for i := 0; i < len(lines); {
		m := beginLatexLinePattern.FindStringSubmatch(lines[i].text)
		if m == nil {
			i++
			continue
		}
		name, end := strings.ToUpper(m[2]), -1
		for j := i + 1; j < len(lines); j++ {
			if e := endLatexLinePattern.FindStringSubmatch(lines[j].text); e != nil && strings.ToUpper(e[2]) == name {
				end = j
				break
			}
		}
		if end < 0 {
			i++
			continue
		}
		spans = append(spans, blockSpan{begin: i, end: end, name: name})
		i = end + 1
	}
	return spans
}

// excludeFootnoteDefinitions mirrors org/footnote.go. A definition opens at
// column zero with an ASCII label followed by whitespace or the line's end,
// and continues until the second of two consecutive blank lines, a headline,
// or another definition line — which opens the next one.
//
// go-org tests that end condition only BETWEEN the constructs a definition
// holds: a nested `#+BEGIN_` block or LaTeX environment runs to its own close
// whatever it contains. So every end condition is suspended inside a region,
// and no definition opens inside one (plan-audit r4, R4-1). A list or an
// ordinary drawer honours the stop, so a blank pair inside one does end it.
func excludeFootnoteDefinitions(lines []sourceLine, region, excluded []bool) {
	for i := 0; i < len(lines); {
		if region[i] || !footnoteDefinitionLinePattern.MatchString(lines[i].text) {
			i++
			continue
		}
		excluded[i] = true
		j, blankRun := i+1, 0
	continuation:
		for ; j < len(lines); j++ {
			switch {
			case region[j]:
				blankRun = 0
			case lines[j].blank:
				if blankRun++; blankRun == 2 {
					break continuation
				}
			case footnoteDefinitionLinePattern.MatchString(lines[j].text),
				headlineLinePattern.MatchString(lines[j].text):
				break continuation
			default:
				blankRun = 0
			}
			excluded[j] = true
		}
		i = j
	}
}

// exportsSuppressResult reports whether a source block's header makes go-org's
// HTML writer drop the block's attached result: its parsed `:exports` value is
// exactly `code` or `none` (org/html_writer.go, WriteBlock). The header is read
// by go-org's own parameter parsing rather than by a second copy of it.
func exportsSuppressResult(header string) bool {
	doc := org.New().Silent().Parse(strings.NewReader(header+"\n#+END_SRC\n"), "./")
	// The header was paired as a block opener, so the node is the block; a
	// zero Block would carry no parameters and read as not suppressed.
	block, _ := doc.Nodes[0].(org.Block)
	exports := block.ParameterMap()[":exports"]
	return exports == "code" || exports == "none"
}

// attachedResult reports the index of the `#+RESULTS:` line go-org attaches to
// a source block, or -1. The parser skips blank lines only after the block's
// close, and attaches a result only when a line follows it to be read as the
// result's one construct (org/block.go, parseSrcBlockResult and parseResult).
func attachedResult(lines []sourceLine, block blockSpan) int {
	j := block.end + 1
	for j < len(lines) && lines[j].blank {
		j++
	}
	if j+1 >= len(lines) || !resultLinePattern.MatchString(lines[j].text) {
		return -1
	}
	return j
}

// excludeAttachedResult excludes a suppressed source block's attached result:
// the `#+RESULTS:` line and the ONE construct the parser reads next.
func excludeAttachedResult(lines []sourceLine, block blockSpan, blocks, environments []blockSpan, region, excluded []bool) {
	result := attachedResult(lines, block)
	if result < 0 {
		return
	}
	markLines(excluded, result, constructEnd(lines, result+1, blocks, environments, region))
}

// constructEnd reports the last line of the one construct go-org's parseOne
// reads starting at line j.
//
// A drawer runs to its `:END:` or, unclosed, to the end of the body; a block
// or environment to its close, a source block carrying its own result with it;
// a list to where its continuation rule ends it. Anything else is read as a
// paragraph — which is go-org's own fall-back for an opener it fails to
// complete — running on through the plain-text lines below it, and starting
// even on a blank line. Reading an unlisted construct as a paragraph can only
// widen the exclusion, so it never hides a marker.
func constructEnd(lines []sourceLine, j int, blocks, environments []blockSpan, region []bool) int {
	text := lines[j].text
	if m := beginDrawerLinePattern.FindStringSubmatch(text); m != nil &&
		!endDrawerLinePattern.MatchString(text) && strings.ToUpper(m[2]) != "PROPERTIES" {
		for k := j + 1; k < len(lines); k++ {
			if !region[k] && endDrawerLinePattern.MatchString(lines[k].text) {
				return k
			}
		}
		return len(lines) - 1
	}
	for _, block := range blocks {
		if block.begin == j {
			if result := attachedResult(lines, block); block.name == "SRC" && result >= 0 {
				return constructEnd(lines, result+1, blocks, environments, region)
			}
			return block.end
		}
	}
	for _, environment := range environments {
		if environment.begin == j {
			return environment.end
		}
	}
	if bullet, ok := readBullet(text); ok {
		return walkList(lines, j, bullet, func(int, listLineRole, bulletLine) {}) - 1
	}
	k := j + 1
	for k < len(lines) && !lines[k].blank && isPlainTextLine(lines[k].text) {
		k++
	}
	return k - 1
}

// excludePropertyDrawers mirrors org/drawer.go's parsePropertyDrawer. A line
// holding only `:PROPERTIES:` opens a property drawer only when every line
// after it, up to a line holding only `:END:`, is a property line — so a blank
// line fails it — and that `:END:` is reached. Such a drawer renders nothing.
// Otherwise the opener is plain text and nothing is suppressed.
//
// It is computed on its own (REQ-SW-001.6), so a `:PROPERTIES:` line inside an
// ordinary drawer, which go-org renders as text, is still read as an opener
// here: an over-exclusion, disclosed in the run-phase record.
func excludePropertyDrawers(lines []sourceLine, excluded []bool) {
	for i := 0; i < len(lines); i++ {
		m := beginDrawerLinePattern.FindStringSubmatch(lines[i].text)
		if m == nil || strings.ToUpper(m[2]) != "PROPERTIES" {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if endDrawerLinePattern.MatchString(lines[j].text) {
				markLines(excluded, i, j)
				i = j
				break
			}
			if !propertyLinePattern.MatchString(lines[j].text) {
				break
			}
		}
	}
}

// excludeListContent excludes every line of every list: each bullet line, and
// every line the list-continuation rule attaches to one. A continuation line
// carries no bullet, so only that rule identifies it (REQ-SW-001.5). Bullets
// inside a block interior are not list items, as for the multiline scan.
func excludeListContent(lines []sourceLine, excluded []bool) {
	interior, _ := blockRegions(lines)
	for i := 0; i < len(lines); {
		bullet, ok := readBullet(lines[i].text)
		if interior[i] || !ok {
			i++
			continue
		}
		end := walkList(lines, i, bullet, func(int, listLineRole, bulletLine) {})
		markLines(excluded, i, end-1)
		i = end
	}
}

// --- scanning and composing ---------------------------------------------------

// scanArrowLines returns a swift entry's arrow lines in document order.
//
// A hand-written marker's extent is measured over the title and the body as
// they are rendered — the title, a blank line, the body — so an unclosed
// opening in the title spans the whole body, exactly as the generated `}}`
// below it would otherwise close it.
func scanArrowLines(title, body string) []arrowLine {
	lines := splitLines(body)
	excluded := nonParagraphLines(lines)
	spans := markerSpans(combineTitleAndBody(title, body))
	shift := len(title) + len("\n\n")

	var arrows []arrowLine
	for i, ln := range lines {
		if excluded[i] {
			continue
		}
		arrow, ok := readArrowLine(ln)
		if !ok || insideAnySpan(spans, shift+arrow.tokenStart) || !isPlainTextLine(ln.text) {
			continue
		}
		arrow.line = i
		arrows = append(arrows, arrow)
	}
	return arrows
}

// composeArrowLines rewrites each arrow line in place and leaves every other
// byte of the body as it was (REQ-SW-001.7).
//
// The i-th arrow line numbers its right side c(base+2i−1) and its left side
// c(base+2i), where base is the highest hand-written number in the entry
// (REQ-SW-003, REQ-SW-005.3). Numbering is POSITIONAL: a side the token leaves
// unwrapped still consumes its number, so inserting a `:<-` line above another
// renumbers nothing below it and Anki keeps every card's review history.
//
// Composition inserts no line of its own (REQ-SW-011.3). The emphasis replaces
// the token; the break is appended to an arrow line if and only if the very
// next line is itself an arrow line, so a prose line below an arrow line
// renders exactly as it does today (REQ-SW-011.1).
func composeArrowLines(title, body string, arrows []arrowLine) string {
	if len(arrows) == 0 {
		return body
	}
	base := highestClozeNumber(title, body)

	var b strings.Builder
	b.Grow(len(body) + 64*len(arrows))
	at := 0
	for n, arrow := range arrows {
		i := n + 1
		b.WriteString(body[at:arrow.leftStart])
		writeSide(&b, body[arrow.leftStart:arrow.leftEnd], base+2*i, arrow.token != ":->")
		b.WriteString(body[arrow.leftEnd:arrow.tokenStart])
		b.WriteString(swiftArrowSnippets[arrow.token])
		b.WriteString(body[arrow.tokenEnd:arrow.rightStart])
		writeSide(&b, body[arrow.rightStart:arrow.rightEnd], base+2*i-1, arrow.token != ":<-")
		b.WriteString(body[arrow.rightEnd:arrow.lineEnd])
		if n+1 < len(arrows) && arrows[n+1].line == arrow.line+1 {
			b.WriteString(swiftLineBreak)
		}
		at = arrow.lineEnd
	}
	b.WriteString(body[at:])
	return b.String()
}

// writeSide writes one side, wrapped in a generated marker numbered number when
// the token selects it and the side carries no hand-written marker of its own
// (REQ-SW-004, REQ-SW-005.2). The wrapped content takes the one brace-safety
// rule the multiline card uses (REQ-SW-006).
func writeSide(b *strings.Builder, side string, number int, selected bool) {
	if !selected || !wrappable(side) {
		b.WriteString(side)
		return
	}
	fmt.Fprintf(b, "{{c%d::%s}}", number, clozeSafeText(side))
}

// composeSwift reads a swift entry's title and remaining body and returns the
// composed body, or a *SwiftArrowMissingError when the body yields no arrow
// line. The title is never wrapped: it stays above the arrow lines as context.
//
// It reads the remaining body as the checked split left it and does NOT
// collapse blank-line runs: swift reads independent lines and has no list to
// keep together, so a wider gap only yields more paragraphs (REQ-SW-008.2,
// plan.md DD-7).
func composeSwift(title, body string) (string, error) {
	arrows := scanArrowLines(title, body)
	if len(arrows) == 0 {
		return "", &SwiftArrowMissingError{}
	}
	return composeArrowLines(title, body, arrows), nil
}
