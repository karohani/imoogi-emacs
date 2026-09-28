package orgdoc

import (
	"regexp"
	"strings"
)

// extraBlockPattern matches one Org `#+BEGIN_EXTRA … #+END_EXTRA` block and
// captures its interior.
//
// Case-insensitive because Org keywords are: a body written `#+begin_extra`
// is the same document, and a split that depended on the author's shouting
// would be a silent authoring trap. Non-greedy interior so two blocks in one
// body are two matches rather than one match swallowing the text between
// them. Both delimiters are line-anchored, so the words appearing inside a
// sentence do not open or close a block.
var extraBlockPattern = regexp.MustCompile(
	`(?ims)` + extraOpenLine + `[^\n]*\n(.*?)` + extraCloseLine + `[^\n]*\n?`)

// extraOpenLine and extraCloseLine are the line-head grammar of the two block
// delimiters, shared by the split above and the unpaired-marker check below so
// the two can never count different lines as markers (SPEC-ANKICARD-005
// REQ-AKX-001.1).
const (
	extraOpenLine  = `^[ \t]*#\+begin_extra`
	extraCloseLine = `^[ \t]*#\+end_extra`
)

// extraMarkerLinePattern matches any line the split would read as a block
// delimiter. No trailing newline is required: an opening marker on a body's
// last line is never consumed by the split, and is exactly what the check must
// still catch.
var extraMarkerLinePattern = regexp.MustCompile(`(?im)` + extraOpenLine + `|` + extraCloseLine)

// ExtraBlockUnbalancedError is returned by the cloze-style renderers when the
// supplementary split leaves an EXTRA marker line behind — in the remaining
// body, or as an opening marker inside the extracted content (SPEC-ANKICARD-005
// REQ-AKX-001, REQ-AKX-002). The caller reports it as a skipped entry carrying
// the extra_block_unbalanced code, and writes nothing for it.
type ExtraBlockUnbalancedError struct{}

func (e *ExtraBlockUnbalancedError) Error() string {
	return "a #+BEGIN_EXTRA or #+END_EXTRA line has no partner after the supplementary split"
}

// splitExtraBlocks separates a heading body into the part that stays in the
// question and the supplementary part destined for the note's extra field.
//
// A body carrying no block is returned BYTE-IDENTICAL — not trimmed, not
// normalized. That is load-bearing rather than incidental: the planner hashes
// the rendered field values, so any reformatting here would re-hash every
// existing note and produce a sync-wide run of `updated` results that carry
// no content change at all.
//
// Several blocks concatenate in document order, separated by a blank line so
// they render as separate paragraphs rather than running together.
//
// Blocks do not nest: an opening pairs with the NEXT closing, as in Org itself.
// Every renderer calls this through splitExtraBlocksChecked, which rejects a
// body whose split leaves a marker unpaired.
func splitExtraBlocks(body string) (rest, extra string) {
	matches := extraBlockPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return body, ""
	}

	interiors := make([]string, 0, len(matches))
	for _, m := range matches {
		if interior := strings.Trim(m[1], "\n"); interior != "" {
			interiors = append(interiors, interior)
		}
	}

	rest = strings.TrimSpace(extraBlockPattern.ReplaceAllString(body, ""))
	return rest, strings.Join(interiors, "\n\n")
}

// splitExtraBlocksChecked is the split every cloze-style renderer runs, followed
// at once by the unpaired-marker check, so the check sits before every other
// render-time diagnostic on every path that splits (SPEC-ANKICARD-005
// REQ-AKX-001.4, REQ-AKX-004).
//
// A marker line left in the remaining body, or one inside the extracted content,
// means the split paired the markers differently from how the author laid them
// out. The entry is rejected rather than repaired: guessing a pairing, or
// deleting the stray line, would be a silent content change.
//
// Checking both sides with the one pattern is deliberate. The extracted content
// can only carry an OPENING marker — a closing one would have ended the match —
// so the closing half of the pattern is inert there, and one predicate stays
// the single definition of a marker line.
func splitExtraBlocksChecked(body string) (rest, extra string, err error) {
	rest, extra = splitExtraBlocks(body)
	if extraMarkerLinePattern.MatchString(rest) || extraMarkerLinePattern.MatchString(extra) {
		return "", "", &ExtraBlockUnbalancedError{}
	}
	return rest, extra, nil
}
