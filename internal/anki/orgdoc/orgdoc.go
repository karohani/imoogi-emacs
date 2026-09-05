// Package orgdoc renders a sync target's title and body into Anki field
// values via go-org (plan.md D-1). It is a pure function of its inputs: no
// registry, no AnkiConnect client, no side effects (plan.md §D module
// table).
package orgdoc

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/niklasfasching/go-org/org"
)

// Note types this SPEC's MVP scope covers (spec.md REQ-003, REQ-004).
const (
	NoteTypeBasic = "Basic"
	NoteTypeCloze = "Cloze"
)

// clozeMarkerPattern matches the opening shape of an Anki cloze marker,
// {{cN::...}}. It deliberately matches only the discriminating prefix
// ({{c<digits>::) rather than the whole marker (including its closing }}),
// since REQ-019's diagnostic fires on the ABSENCE of this shape anywhere in
// the source text — a partial match on the prefix is sufficient evidence a
// marker is present; matching the full construct is unnecessary for a
// presence check.
var clozeMarkerPattern = regexp.MustCompile(`\{\{c\d+::`)

// ClozeMarkerMissingError is returned by Render when a Cloze entry's title
// and body together contain no {{cN:: marker anywhere (spec.md REQ-019,
// plan.md D-5's cloze_marker_missing code). The caller (the M3 planner)
// reports this as a skipped entry and creates no note for it.
type ClozeMarkerMissingError struct{}

func (e *ClozeMarkerMissingError) Error() string {
	return "cloze note type declared but no {{cN:: marker found in title or body"
}

// Render renders a sync target's title and body into the note-type-
// appropriate Anki field shape.
//
//   - Basic (spec.md REQ-003): Front = the rendered title, Back = the
//     rendered body.
//   - Cloze (spec.md REQ-004): a single Text field carrying the rendered
//     combination of title and body, with every {{cN::answer}} marker
//     preserved byte-for-byte. go-org treats a cloze marker as literal
//     inline text it does not specially parse (research.md §2) — the
//     verification obligation this preservation imposes is on THIS
//     function's own rendering path, not on go-org's Org-semantic coverage.
//     When neither the title nor the body contains a {{cN:: marker anywhere,
//     Render returns a *ClozeMarkerMissingError instead of rendering
//     (spec.md REQ-019).
//
// An empty body renders to an empty field rather than crashing
// (acceptance.md §D.7).
func Render(noteType, title, body string) (map[string]string, error) {
	switch noteType {
	case NoteTypeBasic:
		return map[string]string{
			"Front": renderFragment(title),
			"Back":  renderFragment(body),
		}, nil
	case NoteTypeCloze:
		if !hasClozeMarker(title) && !hasClozeMarker(body) {
			return nil, &ClozeMarkerMissingError{}
		}
		return map[string]string{
			"Text": renderFragment(combineTitleAndBody(title, body)),
		}, nil
	default:
		return nil, fmt.Errorf("orgdoc: unrecognized note type %q", noteType)
	}
}

// hasClozeMarker reports whether s contains the discriminating {{cN:: prefix
// of an Anki cloze marker anywhere. Checked against the raw Org source
// (before rendering) — cloze markers pass through go-org's HTML writer
// unchanged (research.md §2), so a raw-text check and a rendered-output
// check are equivalent, and the raw-text check avoids paying a render just
// to detect absence.
func hasClozeMarker(s string) bool {
	return clozeMarkerPattern.MatchString(s)
}

// combineTitleAndBody joins a heading's title and body into one Org source
// fragment for Cloze rendering (spec.md REQ-004: "the Go binary shall render
// the heading title and body into the note's single Text field"). An empty
// body yields the title alone, so a title-only Cloze entry does not carry a
// spurious blank paragraph.
func combineTitleAndBody(title, body string) string {
	if body == "" {
		return title
	}
	return title + "\n\n" + body
}

// renderFragment renders one Org source fragment to HTML via go-org, then
// applies the MathJax delimiter transform (spec.md REQ-C-010, REQ-C-011). An
// empty fragment renders to an empty string (go-org's own behavior on empty
// input) rather than erroring — the empty-body edge case (acceptance.md
// §D.7) relies on this.
//
// The transform runs HERE, on the rendered output, so the field values Render
// returns are already transformed. That ordering is what makes REQ-C-016 hold
// by construction: the planner hashes what this function returns, so a math
// edit changes the hash and a re-render of unchanged math does not.
func renderFragment(s string) string {
	if s == "" {
		return ""
	}
	html, err := org.New().Parse(strings.NewReader(s), "./").Write(org.NewHTMLWriter())
	if err != nil {
		// go-org's HTML writer does not fail on well-formed plain-text/prose
		// input in practice (research.md §2's documented 80/20 subset covers
		// this SPEC's card-body scope); render the raw text verbatim rather
		// than surface a Go error the caller has no typed diagnostic for.
		// The transform still applies: the delimiter contract is a property of
		// the field value the planner hashes, not of which render path
		// produced it. Raw Org is not HTML, so transformMath's markup handling
		// treats any `<` there conservatively (see transformMath).
		return transformMath(s)
	}
	return transformMath(html)
}

// --- MathJax delimiter transform (spec.md REQ-C-010, REQ-C-011) -------------
//
// A post-render pass, sibling to the cloze-marker guard above: it reads the
// rendered HTML and rewrites Org's math delimiters into the two forms Anki's
// bundled MathJax accepts. Anki's cloze logic has special handling that a
// custom delimiter breaks, so exactly these two pairs are ever emitted
// (design.md § 10.2).
const (
	mathInlineOpen   = `\(`
	mathInlineClose  = `\)`
	mathDisplayOpen  = `\[`
	mathDisplayClose = `\]`

	// maxInlineMathLineBreaks is the Org inline-math heuristic's line-break
	// bound (spec.md § 2): a single-`$` candidate enclosing more than this many
	// line breaks is prose, not math.
	maxInlineMathLineBreaks = 2
)

// mathLineBreaks replaces the line breaks inside a converted fragment with
// <br> (REQ-C-010.3). A bare newline is unrenderable by Anki's MathJax, so it
// cannot simply be left alone. CRLF is collapsed to a single <br> rather than
// leaving a stray carriage return behind.
var mathLineBreaks = strings.NewReplacer("\r\n", "<br>", "\n", "<br>")

// transformMath rewrites the math delimiters in rendered HTML.
//
// It walks the input as an alternation of markup spans (`<`…`>`) and text
// spans, and converts only text spans that sit outside a <pre> or <code>
// region. That single rule discharges all four of REQ-C-011's protected
// shapes: <pre> and <code> regions by the depth counter, and HTML attribute
// values and link targets because a markup span is copied through opaquely —
// an href or an alt is inside `<`…`>` and is therefore never examined.
//
// The `<` discriminator is sound because go-org escapes literal `<` in text to
// &lt;, so a bare `<` in rendered output is always markup. On the raw-text
// fallback path (renderFragment's error branch) the input is Org source rather
// than HTML, and a `<` there is treated as markup anyway — conversion is
// suppressed rather than risked, which is the safe direction: content is
// always preserved byte-for-byte, only the rewrite is skipped.
func transformMath(html string) string {
	var b strings.Builder
	depth, start := 0, 0
	inTag := false
	for i := 0; i < len(html); i++ {
		switch {
		case !inTag && html[i] == '<':
			b.WriteString(convertUnprotected(html[start:i], depth))
			start, inTag = i, true
		case inTag && html[i] == '>':
			tag := html[start : i+1]
			b.WriteString(tag)
			depth += mathProtectionDelta(tag)
			start, inTag = i+1, false
		}
	}
	if inTag {
		// An unterminated `<`: everything from it on is markup-shaped and is
		// copied through without conversion.
		b.WriteString(html[start:])
	} else {
		b.WriteString(convertUnprotected(html[start:], depth))
	}
	return b.String()
}

// convertUnprotected converts a text span unless it sits inside a protected
// region (REQ-C-011). A negative depth — reachable only from malformed input
// on the raw-text fallback path — reads as unprotected, matching depth zero.
func convertUnprotected(text string, depth int) string {
	if depth > 0 {
		return text
	}
	return convertMathDelimiters(text)
}

// mathProtectionDelta reports how a markup span moves the protected-region
// depth: +1 for an opening <pre>/<code>, -1 for its closing tag, 0 for every
// other tag. Attribute-bearing forms (<pre class="example">) and the
// self-closing form both reduce to the leading name.
func mathProtectionDelta(tag string) int {
	name := strings.TrimSuffix(strings.TrimPrefix(tag, "<"), ">")
	closing := strings.HasPrefix(name, "/")
	if closing {
		name = name[1:]
	}
	if cut := strings.IndexAny(name, " \t\n/"); cut >= 0 {
		name = name[:cut]
	}
	if name != "pre" && name != "code" {
		return 0
	}
	if closing {
		return -1
	}
	return 1
}

// convertMathDelimiters applies the REQ-C-010.1 delimiter table to one
// unprotected text span, left to right.
//
// The `$$` case is tried before the single-`$` case so a display fragment is
// never mistaken for two inline ones. The already-delimited `\(`…`\)` and
// `\[`…`\]` forms are recognized too, even though their delimiters are already
// correct: recognizing them is what lets REQ-C-010.3's <br> rule reach their
// interiors, and what stops a `$` inside one from being reinterpreted.
//
// A candidate whose closing delimiter is absent, or whose single-`$` form
// fails the heuristic, is emitted verbatim and the scan resumes one byte on —
// so the rejected `$` becomes the next candidate's opening, which is what
// makes `costs $5 and $7 total` come through untouched (AC-C-009a).
func convertMathDelimiters(text string) string {
	var b strings.Builder
	for i := 0; i < len(text); {
		switch {
		case strings.HasPrefix(text[i:], "$$"):
			if out, next, ok := pairedFragment(text, i, "$$", mathDisplayOpen, mathDisplayClose); ok {
				b.WriteString(out)
				i = next
				continue
			}
		case strings.HasPrefix(text[i:], mathInlineOpen):
			if out, next, ok := pairedFragment(text, i, mathInlineClose, mathInlineOpen, mathInlineClose); ok {
				b.WriteString(out)
				i = next
				continue
			}
		case strings.HasPrefix(text[i:], mathDisplayOpen):
			if out, next, ok := pairedFragment(text, i, mathDisplayClose, mathDisplayOpen, mathDisplayClose); ok {
				b.WriteString(out)
				i = next
				continue
			}
		case text[i] == '$':
			if j, ok := inlineDollarClose(text, i); ok {
				b.WriteString(mathInlineOpen + mathLineBreaks.Replace(text[i+1:j]) + mathInlineClose)
				i = j + 1
				continue
			}
		}
		b.WriteByte(text[i])
		i++
	}
	return b.String()
}

// pairedFragment converts a two-byte-delimited fragment opening at i, whose
// closing delimiter is closeMark, into open + interior + close, and reports the
// index to resume scanning at. It returns false when the closing delimiter is
// absent, leaving the caller to emit the opening bytes verbatim.
//
// All three two-byte forms ($$…$$, \(…\), \[…\]) share this one implementation
// so the index arithmetic — 2 opening bytes plus 2 closing bytes past the
// interior — is written once rather than three times.
func pairedFragment(text string, i int, closeMark, open, close string) (string, int, bool) {
	j := strings.Index(text[i+2:], closeMark)
	if j < 0 {
		return "", 0, false
	}
	return open + mathLineBreaks.Replace(text[i+2:i+2+j]) + close, i + j + 4, true
}

// inlineDollarClose applies the Org inline-math heuristic (spec.md § 2,
// consumed by REQ-C-010.2) to the single-`$` candidate opening at i, returning
// the index of its closing `$`.
//
// The candidate's interior never spans a `$`, matching Org's own rule, so the
// closing candidate is the very next `$` and nothing further. That is why
// `costs $5 and $7 total` is rejected on the first candidate (the interior
// `5 and ` ends in whitespace) rather than being stretched to a later `$`.
func inlineDollarClose(text string, i int) (int, bool) {
	rel := strings.IndexByte(text[i+1:], '$')
	if rel <= 0 {
		// No closing `$`, or an empty interior — neither is a fragment.
		return 0, false
	}
	j := i + 1 + rel
	interior := text[i+1 : j]
	if isASCIISpace(interior[0]) ||
		isASCIISpace(interior[len(interior)-1]) ||
		strings.Count(interior, "\n") > maxInlineMathLineBreaks ||
		!isClosingDollarBoundary(text[j+1:]) {
		return 0, false
	}
	return j, true
}

// isClosingDollarBoundary reports whether what follows a candidate's closing
// `$` satisfies the heuristic: whitespace, or punctuation other than a dash.
// The end of the text span counts as a boundary — a fragment that fills its
// paragraph has nothing after it (AC-C-008's first row is exactly this shape).
//
// The dash exclusion is spec.md § 2's wording, which is narrower than Org's own
// follow-set; the spec is followed as written.
func isClosingDollarBoundary(rest string) bool {
	if rest == "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(rest)
	return unicode.IsSpace(r) || (unicode.IsPunct(r) && r != '-')
}

// isASCIISpace reports whether b is one of the whitespace bytes the "attached
// to both `$` characters" clause of the heuristic rejects.
func isASCIISpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
