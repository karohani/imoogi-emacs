package orgdoc

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/niklasfasching/go-org/org"
)

// Multiline question-and-answer composition (spec.md SPEC-ANKICARD-003).
//
// A heading carrying a multiline option becomes one question-and-answer card:
// the title is the question, the first top-level list of the remaining body
// holds the answers, and cloze markers are placed according to the resolved
// direction.
//
// Composition is SOURCE-LEVEL and PRE-RENDER (plan.md DD-1). Two measured
// facts make HTML-level wrapping wrong rather than merely awkward: a nested
// list renders INSIDE its parent item, so finding a depth-one <li> means
// writing an HTML parser; and a description list emits no <li> at all, so an
// <li> scanner misses that shape SILENTLY, producing an unwrapped card with no
// signal. A marker written into Org source, by contrast, passes through go-org
// untouched in every list form.
//
// The mechanism is a source-level scanner whose bullet grammar mirrors
// go-org's own, because org.ListItem carries no source position — the AST
// cannot map an item back to the line that produced it. The parse is still
// used for the two things the scanner must not guess: the list's KIND (the
// descriptive form is a distinct AST type) and its item COUNT, which
// TestMultilineAnswerList asserts the scanner agrees with over the whole
// corpus. That equality is what makes the hand-rolled scanner falsifiable.

// Direction is the resolved ANKI_DIRECTION, deciding which spans composition
// wraps in a generated marker (spec.md REQ-ML-003).
type Direction int

// The four states ANKI_DIRECTION resolves to. DirectionNone is both the
// absent case and the explicitly falsy one, which SPEC-ANKICARD-002 recognizes
// as "no direction" rather than as an error; composition treats it as `->`
// (spec.md REQ-ML-004).
const (
	DirectionNone Direction = iota
	DirectionRightward
	DirectionLeftward
	DirectionBoth
)

// CardOptions carries the resolved multiline pair. It deliberately does NOT
// carry ANKI_SWIFT: swift selects a different card kind, and an entry bearing
// swift beside a multiline option is rejected by the planner's validation gate
// before it can reach composition (spec.md § 3 preamble).
type CardOptions struct {
	Direction   Direction
	Incremental bool
}

// multiline reports whether at least one multiline option is ON — the
// definition of a MULTILINE ENTRY (spec.md § 2). An entry for which this is
// false takes the existing rendering path byte-for-byte (REQ-ML-014.1).
func (o CardOptions) multiline() bool {
	return o.Direction != DirectionNone || o.Incremental
}

// wrapsAnswers and wrapsTitle read the direction table of REQ-ML-003, with
// REQ-ML-004's default folded in: an unresolved direction composes as `->`.
func (o CardOptions) wrapsAnswers() bool {
	return o.Direction != DirectionLeftward
}

func (o CardOptions) wrapsTitle() bool {
	return o.Direction == DirectionLeftward || o.Direction == DirectionBoth
}

// MultilineAnswerMissingError is returned by RenderWithOptions when a
// multiline entry's remaining body yields no answer item (spec.md REQ-ML-002).
// The caller reports this as a skipped entry carrying the
// multiline_answer_missing code, and creates no note for it.
//
// The obligation holds for EVERY direction, `<-` included: on a body with no
// list, a leftward card hides the only content it has and leaves a prompt with
// no visible cue — a worse card than the rightward case, not an exempt one.
type MultilineAnswerMissingError struct{}

func (e *MultilineAnswerMissingError) Error() string {
	return "multiline card option is on but the entry's body carries no answer list"
}

// --- the source-level scanner ----------------------------------------------

// The bullet grammar, mirroring go-org v1.9.1's org/list.go verbatim. Group 4
// (unordered) and group 5 (ordered) are the item's CONTENT — the offset at
// which the renderer's own parser begins reading it.
var (
	unorderedBulletPattern = regexp.MustCompile(`^(\s*)([+*-])(\s+(.*)|$)`)
	orderedBulletPattern   = regexp.MustCompile(`^(\s*)(([0-9]+|[a-zA-Z])[.)])(\s+(.*)|$)`)

	// The three consumptions the parser performs ahead of an item's answer
	// content, in the order it performs them. Both of the first two are
	// UNANCHORED and the parser then removes a fixed BYTE count from position
	// zero regardless of where the match occurred — which is why the offset is
	// mirrored arithmetically here and never found by searching for a leading
	// token (spec.md REQ-ML-001.3, plan.md § G anti-patterns 11 and 12).
	counterCookiePattern    = regexp.MustCompile(`\[@(\d+)\]\s`)
	itemStatusPattern       = regexp.MustCompile(`\[( |X|-)\]\s`)
	descriptiveSplitPattern = regexp.MustCompile(`\s::(\s|$)`)

	beginBlockLinePattern = regexp.MustCompile(`(?i)^\s*#\+BEGIN_(\w+)`)
	endBlockLinePattern   = regexp.MustCompile(`(?i)^\s*#\+END_(\w+)`)
)

// The list kinds go-org's parser assigns. A DESCRIPTIVE list is neither
// ordered nor unordered for consumption purposes: the counter cookie is gated
// on the kind being ordered, so one `::` in the list's FIRST item leaves the
// cookie in the term (plan.md DD-3).
const (
	listKindOrdered     = "ordered"
	listKindDescriptive = "descriptive"
)

// answerItem is one answer item's wrappable span, as byte offsets into the
// body the scanner was given. An item whose content is empty has start == end
// and contributes nothing to wrap.
type answerItem struct{ start, end int }

// answerList is what the scanner found: where the container line is inserted,
// and one span per answer item in document order.
type answerList struct {
	containerAt int
	items       []answerItem
}

// sourceLine is one line of a body with its byte extent, so a span can be
// expressed as offsets into the original string rather than as a copy.
type sourceLine struct {
	start, end int // end excludes the newline
	text       string
	hasNewline bool
	indent     int
	blank      bool
}

func splitLines(body string) []sourceLine {
	lines := make([]sourceLine, 0, strings.Count(body, "\n")+1)
	for at := 0; at < len(body); {
		end := strings.IndexByte(body[at:], '\n')
		nl := end >= 0
		if !nl {
			end = len(body) - at
		}
		text := body[at : at+end]
		lines = append(lines, sourceLine{
			start:      at,
			end:        at + end,
			text:       text,
			hasNewline: nl,
			indent:     len(text) - len(strings.TrimLeft(text, " \t")),
			blank:      strings.TrimSpace(text) == "",
		})
		at += end
		if nl {
			at++
		}
	}
	return lines
}

// blockRegions reports which lines sit inside a block interior, and the index
// of the first `#+BEGIN_` line that is never terminated.
//
// The pairing rule is go-org's own: a block closes at the next `#+END_` whose
// NAME matches, compared case-insensitively, so `#+END_EXAMPLE` does not close
// a `#+BEGIN_SRC`. A `#+BEGIN_` with no matching close opens nothing — the
// parser backtracks and renders the line as plain text — and an orphan
// `#+END_` likewise opens nothing, which matters because the supplementary
// split leaves exactly that shape behind when the author nested one
// `#+BEGIN_EXTRA` block inside another.
//
// `undefinedFrom` is what REQ-ML-009.2's decline keys on: from an unterminated
// opening to the end of the body the extent of the block is undefined, so the
// collapse declines rather than guessing where it ends.
func blockRegions(lines []sourceLine) (interior []bool, undefinedFrom int) {
	interior = make([]bool, len(lines))
	undefinedFrom = -1
	for i := 0; i < len(lines); {
		m := beginBlockLinePattern.FindStringSubmatch(lines[i].text)
		if m == nil {
			i++
			continue
		}
		name := strings.ToUpper(m[1])
		end := -1
		for j := i + 1; j < len(lines); j++ {
			if e := endBlockLinePattern.FindStringSubmatch(lines[j].text); e != nil &&
				strings.ToUpper(e[1]) == name {
				end = j
				break
			}
		}
		if end < 0 {
			if undefinedFrom < 0 {
				undefinedFrom = i
			}
			i++
			continue
		}
		for j := i + 1; j < end; j++ {
			interior[j] = true
		}
		i = end + 1
	}
	return interior, undefinedFrom
}

// collapseBlankRuns reduces a run of consecutive blank lines to a single blank
// line (spec.md REQ-ML-009.2). It runs for a multiline entry and for no other,
// so it re-hashes nothing outside this SPEC's own surface.
//
// Without it, a supplementary block sitting between two answers leaves a
// two-blank-line gap once the split has removed it, and go-org reads that gap
// as ending the list — so every answer after the block is silently dropped
// from the card.
//
// Two exclusions bound it, and both exist to prevent a SILENT content change.
// A blank-line run inside a block is content the author wrote, so collapsing it
// would reformat a code sample with no diagnostic. And where a `#+BEGIN_` is
// never terminated the block's extent is undefined, so the collapse declines
// from that line on rather than guess; the consequence is that the answer list
// stays split, which is what the tree does today.
func collapseBlankRuns(body string) string {
	lines := splitLines(body)
	interior, undefinedFrom := blockRegions(lines)

	var b strings.Builder
	b.Grow(len(body))
	blankRun := 0
	for i, ln := range lines {
		collapsible := ln.blank && !interior[i] && (undefinedFrom < 0 || i < undefinedFrom)
		if collapsible {
			blankRun++
			if blankRun > 1 {
				continue
			}
		} else {
			blankRun = 0
		}
		b.WriteString(ln.text)
		if ln.hasNewline {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// parseFirstList reports the kind and item count of the first top-level list
// go-org itself finds in a body (plan.md DD-1).
//
// These are the two facts the scanner must not guess. The kind decides which
// consumptions apply, and the count is the cross-check that makes the scanner
// falsifiable — asserted as a property over the corpus rather than at runtime,
// because at runtime there is no correct action to take on disagreement and
// rendering unwrapped is the silent failure DD-1 rejects post-render for.
func parseFirstList(body string) (kind string, items int, ok bool) {
	doc := org.New().Silent().Parse(strings.NewReader(body), "./")
	for _, node := range doc.Nodes {
		if list, isList := node.(org.List); isList {
			return list.Kind, len(list.Items), true
		}
	}
	return "", 0, false
}

// bulletLine is one line read as a list bullet.
type bulletLine struct {
	indent       int
	mainKind     string // "ordered" or "unordered" — the axis a bullet change does NOT cross
	minIndent    int    // indent + len(bullet); a continuation must reach it
	contentStart int    // byte offset WITHIN the line at which content begins
}

// readBullet reads one line as a list bullet, mirroring go-org's lexList.
func readBullet(text string) (bulletLine, bool) {
	if m := unorderedBulletPattern.FindStringSubmatchIndex(text); m != nil {
		return bulletFrom(text, m, "unordered", 2, 4), true
	}
	if m := orderedBulletPattern.FindStringSubmatchIndex(text); m != nil {
		return bulletFrom(text, m, listKindOrdered, 2, 5), true
	}
	return bulletLine{}, false
}

// bulletFrom assembles a bulletLine from a submatch index pair. An unset
// content group — the bullet-only form `-` with nothing after it — reads as
// empty content beginning at the end of the line, which is what the parser
// does with it.
func bulletFrom(text string, m []int, mainKind string, bulletGroup, contentGroup int) bulletLine {
	indent := m[3] - m[2] // group 1 is the leading whitespace; its width is the indent
	contentStart := len(text)
	if m[2*contentGroup] >= 0 {
		contentStart = m[2*contentGroup]
	}
	return bulletLine{
		indent:       indent,
		mainKind:     mainKind,
		minIndent:    indent + (m[2*bulletGroup+1] - m[2*bulletGroup]),
		contentStart: contentStart,
	}
}

// answerContentOffset reports how many BYTES the parser consumes from the
// start of an item's content before the answer begins, mirroring go-org's own
// arithmetic in order (spec.md REQ-ML-001.3).
//
// The order matters: the status expression runs against the already-sliced
// post-cookie content, and the descriptive split runs against what is left of
// that. Each slice is a fixed byte count from position zero except the
// descriptive term, which alone slices at its own match index.
func answerContentOffset(content, kind string) int {
	off := 0
	if kind == listKindOrdered {
		if m := counterCookiePattern.FindStringSubmatch(content); m != nil {
			off += len("[@] ") + len(m[1])
		}
	}
	if itemStatusPattern.MatchString(content[off:]) {
		off += len("[ ] ")
	}
	if kind == listKindDescriptive {
		if m := descriptiveSplitPattern.FindStringIndex(content[off:]); m != nil {
			off += m[1]
		}
	}
	return off
}

// scanAnswerList finds the answer list — the first list go-org recognizes at
// the top level of the body — and returns one span per answer item.
//
// The four rules the bare bullet grammar does not carry, each measured against
// go-org v1.9.1: a bullet line inside a block interior is not a list item; the
// base indentation comes from the list's FIRST bullet rather than from column
// zero; a continuation line must reach the item's own minimum indent, which is
// the bullet's indent plus the bullet's own width; and two consecutive blank
// lines end the list where one does not.
//
// A span covers the item's own content and stops before its nested children:
// at the first nested bullet, the first blank line, or the first line indented
// less than the item's content (plan.md DD-2). Extending it over a nested list
// would leave the marker spanning intervening markup, whose behaviour inside
// an Anki cloze is unverified here.
func scanAnswerList(body, kind string) (answerList, bool) {
	lines := splitLines(body)
	interior, _ := blockRegions(lines)

	first := -1
	var base bulletLine
	for i, ln := range lines {
		if interior[i] {
			continue
		}
		if b, ok := readBullet(ln.text); ok {
			first, base = i, b
			break
		}
	}
	if first < 0 {
		return answerList{}, false
	}

	list := answerList{containerAt: lines[first].start}
	cur := base
	spanOpen := false
	blankRun := 0
	for i := first; i < len(lines); i++ {
		ln := lines[i]
		if ln.blank {
			blankRun++
			if blankRun >= 2 {
				break
			}
			spanOpen = false
			continue
		}
		b, isBullet := readBullet(ln.text)
		if isBullet && b.indent == base.indent && b.mainKind == base.mainKind {
			blankRun = 0
			cur = b
			start := ln.start + b.contentStart + answerContentOffset(ln.text[b.contentStart:], kind)
			list.items = append(list.items, answerItem{start: start, end: ln.end})
			spanOpen = true
			continue
		}
		if ln.indent < cur.minIndent {
			break
		}
		// A continuation of the current item. It extends the span only while
		// the span is still open and the line is not itself a bullet — a
		// nested bullet is a child, and a blank line has already closed it.
		blankRun = 0
		if spanOpen && !isBullet {
			list.items[len(list.items)-1].end = ln.end
		} else {
			spanOpen = false
		}
	}
	return list, true
}

// --- composition ------------------------------------------------------------

// containerClassLine is the one prepended Org line that puts the answer list
// inside an element the card stylesheet can target (spec.md REQ-ML-012).
//
// It rides the same pre-render mechanism as the markers, and measured, it
// attaches to ALL THREE list forms — the <ul>, the <ol>, and the <dl> alike —
// which is what keeps a description-style answer list from rendering unstyled.
// A `#+BEGIN_…` special block was measured as the alternative and rejected:
// go-org truncates the block name at a hyphen and appends its own suffix, so
// `#+BEGIN_CHILDREN-LIST` yields the wrong class inside an extra wrapper.
const containerClassLine = "#+ATTR_HTML: :class children-list\n"

var (
	braceRunPattern    = regexp.MustCompile(`\}{2,}`)
	clozeNumberPattern = regexp.MustCompile(`\{\{c(\d+)::`)
)

// clozeSafeText returns text in the form that can sit inside a generated
// marker without closing it early (spec.md REQ-ML-007).
//
// Anki's cloze pattern is non-greedy: it closes at the first `}}` after the
// opening marker. So a run of two or more closing braces inside the wrapped
// content is separated by a single space each, and content ending in `}` gains
// one space so the boundary with the closing `}}` does not itself form a third
// consecutive brace. LaTeX ignores whitespace between braces, so the meaning of
// a formula is unchanged.
//
// Separation and pad are ONE operation here because they are one operation in
// the editor too (REQ-ML-007.3), which is what lets both implementations be
// compared against the single shared fixture in testdata.
func clozeSafeText(text string) string {
	safe := braceRunPattern.ReplaceAllStringFunc(text, func(run string) string {
		return strings.Join(strings.Split(run, ""), " ")
	})
	if strings.HasSuffix(safe, "}") {
		safe += " "
	}
	return safe
}

// highestClozeNumber reports the largest number any hand-written marker in the
// given fragments carries, or zero when there is none.
//
// Generated numbering begins above it, so a generated marker can never collide
// with one the author wrote elsewhere in the entry — the same convention
// `imoogi-anki--next-cloze-number` uses, so both sides of the system count
// alike (spec.md REQ-ML-006.2).
func highestClozeNumber(fragments ...string) int {
	highest := 0
	for _, fragment := range fragments {
		for _, m := range clozeNumberPattern.FindAllStringSubmatch(fragment, -1) {
			if n, _ := strconv.Atoi(m[1]); n > highest {
				highest = n
			}
		}
	}
	return highest
}

// composeMultiline reads a multiline entry's title and remaining body and
// produces the Org source fragments the renderer then renders (spec.md § 3.2).
//
// It returns a *MultilineAnswerMissingError when the remaining body yields no
// answer item, for EVERY direction — a card that asks a question it states no
// answer to is not a card, and admitting one direction would make the
// diagnostic's meaning depend on which arrow was written.
//
// A DANGLING SUPPLEMENTARY MARKER gets no special handling, and that is a
// decision rather than an oversight. The split can leave an unpaired
// `#+BEGIN_EXTRA` or `#+END_EXTRA` in the remaining body; such a line is
// ORDINARY CONTENT here, and it ends the answer list exactly as a paragraph
// does — REQ-ML-001.4 and REQ-ML-001.5 applied unchanged. The consequence is
// real and user-visible: answers after the marker are dropped from the card,
// and on an unterminated opening go-org swallows the following bullet into the
// marker's own paragraph so it stops being a list item at all.
//
// Three alternatives were measured and rejected. Declining to collapse — the
// rule REQ-ML-009.2 applies to an unterminated block — is a NO-OP on every one
// of these shapes: the separating line is non-blank, so there is no blank-line
// run to decline on, and rendering is byte-identical with the collapse and
// without it. Removing the marker is the silent content change REQ-ML-009.2
// exists to forbid. Reading the answer list ACROSS it would wrap answers that
// go-org renders in a second list, outside the `children-list` container,
// contradicting REQ-ML-012.1.
//
// What remains is to report it, and no diagnostic code in this SPEC covers the
// shape: REQ-ML-002's fires only when there is no answer item, and here there
// is one. Making the outcome non-silent therefore needs a new code and a new
// requirement — a SPEC change, deliberately not made here. The behaviour is
// pinned by TestMultilineAnswerList's dangling-marker corpus rows and by
// TestMultilineDanglingExtraMarker in the planner.
func composeMultiline(title, body string, opts CardOptions) (string, string, error) {
	collapsed := collapseBlankRuns(body)

	// The parse supplies the list's KIND; the scanner supplies the spans.
	// Their agreement on how many items there are is asserted over the corpus
	// (TestMultilineAnswerList) rather than checked here: at runtime there is
	// no correct action to take on a disagreement, and silently rendering
	// unwrapped is the outcome DD-1 rejects post-render for.
	kind, _, _ := parseFirstList(collapsed)
	list, found := scanAnswerList(collapsed, kind)
	if !found {
		return "", "", &MultilineAnswerMissingError{}
	}

	base := highestClozeNumber(title, collapsed)

	// `<-` wraps only the title, so there is nothing for per-item numbering to
	// number: incremental is inert there and raises no diagnostic, because
	// reporting it would cost a diagnostic code to describe a card that is
	// correct (spec.md REQ-ML-008).
	incremental := opts.Incremental && opts.Direction != DirectionLeftward

	var b strings.Builder
	b.Grow(len(collapsed) + len(containerClassLine))
	b.WriteString(collapsed[:list.containerAt])
	b.WriteString(containerClassLine)

	at := list.containerAt
	number := base + 1
	for _, item := range list.items {
		span := collapsed[item.start:item.end]
		if !opts.wrapsAnswers() || !wrappable(span) {
			continue
		}
		b.WriteString(collapsed[at:item.start])
		fmt.Fprintf(&b, "{{c%d::%s}}", number, clozeSafeText(span))
		at = item.end
		if incremental {
			number++
		}
	}
	b.WriteString(collapsed[at:])

	// With incremental off the answers share one number and the title takes
	// the next, which is the design record's fixed c1/c2 pair on an entry
	// carrying no hand-written marker. With incremental on the answers have
	// consumed consecutive numbers and the title sits above them all
	// (spec.md REQ-ML-005.3, REQ-ML-006.3).
	titleNumber := base + 2
	if incremental {
		titleNumber = number
	}

	composedTitle := title
	if opts.wrapsTitle() && wrappable(title) {
		composedTitle = fmt.Sprintf("{{c%d::%s}}", titleNumber, clozeSafeText(title))
	}
	return composedTitle, b.String(), nil
}

// wrappable reports whether a span is one composition may place a generated
// marker around. It binds an answer item and the title alike.
//
// A span already carrying a HAND-WRITTEN marker is not wrapped, and reaches the
// renderer byte-for-byte as the author wrote it. Anki cannot nest cloze markers
// — its pattern is non-greedy, so the outer marker would close at the inner
// one's braces — and wrapping anyway would destroy the card that marker already
// makes (spec.md REQ-ML-006.1). A span with no content has nothing to wrap, and
// an empty generated marker is a blank with no answer behind it.
func wrappable(span string) bool {
	return span != "" && !hasClozeMarker(span)
}

// RenderWithOptions is the option-taking entry point (spec.md REQ-ML-010). Both
// the ordinary synchronization path and the migration path reach the renderer
// through it.
//
// For an entry that is not a multiline entry it DELEGATES to Render rather than
// reimplement its behaviour, which is what makes REQ-ML-014.1's byte-identity
// structural instead of merely tested: once both call sites move here, a second
// rendering path for the no-option case would be free to drift, and the corpus
// that would catch the drift exercises only the path no production caller
// takes. Render itself keeps its signature and its output, so
// SPEC-ANKICARD-002's compile-time signature assertion goes on holding.
//
// The note-type guard is the same delegation. Every card kind these options
// select is built on a cloze-style note type, and the planner's validation gate
// rejects an option on any other type before the render — so the guard is
// unreachable from production and exists to keep this function total.
//
// The pipeline order is fixed: split, compose, gate, render. Composition reads
// the REMAINING body, so supplementary content can never become an answer item;
// and it runs BEFORE the marker gate, so a multiline entry carrying no
// hand-written marker is satisfied by the markers composition generated rather
// than skipped as unmarked (spec.md REQ-ML-009).
func RenderWithOptions(noteType, title, body string, opts CardOptions) (map[string]string, error) {
	if !opts.multiline() || noteType != NoteTypeCloze {
		return Render(noteType, title, body)
	}

	rest, extra := splitExtraBlocks(body)
	composedTitle, composedRest, err := composeMultiline(title, rest, opts)
	if err != nil {
		return nil, err
	}
	if !hasClozeMarker(composedTitle) && !hasClozeMarker(composedRest) {
		return nil, &ClozeMarkerMissingError{}
	}
	return map[string]string{
		"Text":       renderFragment(combineTitleAndBody(composedTitle, composedRest)),
		"Back Extra": renderFragment(extra),
	}, nil
}
