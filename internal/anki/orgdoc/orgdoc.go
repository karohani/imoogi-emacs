// Package orgdoc renders a sync target's title and body into Anki field
// values via go-org (plan.md D-1). It is a pure function of its inputs: no
// registry, no AnkiConnect client, no side effects (plan.md §D module
// table).
package orgdoc

import (
	"fmt"
	"regexp"
	"strings"

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

// renderFragment renders one Org source fragment to HTML via go-org. An
// empty fragment renders to an empty string (go-org's own behavior on empty
// input) rather than erroring — the empty-body edge case (acceptance.md
// §D.7) relies on this.
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
		return s
	}
	return html
}
