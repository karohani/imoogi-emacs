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
	`(?ims)^[ \t]*#\+begin_extra[^\n]*\n(.*?)^[ \t]*#\+end_extra[^\n]*\n?`)

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
