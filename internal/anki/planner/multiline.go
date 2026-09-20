package planner

import (
	"strings"

	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// readCardOptions maps an entry's resolved option VALUES onto the renderer's
// own enum (SPEC-ANKICARD-003 REQ-ML-010).
//
// This is the one place the wire's vocabulary and the renderer's meet, and it
// adds no second parser for either: the trimming and case rules stay in
// `readDirection` and `readBoolean`, which SPEC-ANKICARD-002 owns. A second
// copy of those rules would drift the first time one of them was edited.
//
// A value this function cannot classify is impossible here, because
// `validateCardOptions` runs before the render on both paths and rejects an
// unrecognized value with `card_option_invalid` — so an unrecognized direction
// reaching this point would already have been reported against the drawer line
// the user wrote.
func readCardOptions(entry protocol.Entry) orgdoc.CardOptions {
	opts := orgdoc.CardOptions{}
	if incremental, _ := readBoolean(entry.Incremental); incremental {
		opts.Incremental = true
	}
	if on, _ := readDirection(entry.Direction); !on {
		return opts
	}
	switch strings.TrimSpace(*entry.Direction) {
	case directionLeftward:
		opts.Direction = orgdoc.DirectionLeftward
	case directionBoth:
		opts.Direction = orgdoc.DirectionBoth
	default:
		opts.Direction = orgdoc.DirectionRightward
	}
	return opts
}

// renderError maps a render failure onto the diagnostic code that names it and
// says whether the entry is SKIPPED or FAILED, shared by the ordinary
// synchronization path and the migration path so the two cannot report the
// same failure differently.
//
// A skip is a condition of the entry's own content that the user can correct
// in the Org buffer — a missing marker, a missing answer list — and it leaves
// the collection, the registry, and the heading untouched. Anything else is a
// rendering failure.
//
// The multiline code joins the one the renderer could already return. It is
// reported only for an entry that already passed the validation gate, which is
// what keeps it out of that gate's fixed order (REQ-ML-011).
func renderError(err error) (code string, skip bool) {
	switch err.(type) {
	case *orgdoc.ClozeMarkerMissingError:
		return protocol.CodeClozeMarkerMissing, true
	case *orgdoc.MultilineAnswerMissingError:
		return protocol.CodeMultilineAnswerMissing, true
	default:
		return protocol.CodeOrgParseError, false
	}
}
