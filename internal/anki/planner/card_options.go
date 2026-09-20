package planner

import (
	"fmt"
	"strings"

	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// The recognized card-option values (spec.md REQ-OPT-009).
//
// `t` and `nil` rather than `true`/`false` or `yes`/`no` because the values
// are written in Org by an Emacs user, and these are the spellings the
// surroundings use. The falsy spelling is recognized on ALL THREE properties,
// including direction, where it means "no direction" rather than naming one:
// an explicit opt-out that worked on two of three inheriting properties and
// errored on the third would be a trap, since the failure is invisible on a
// drawer line and the user has no way to predict which property behaves which
// way.
const (
	directionRightward = "->"
	directionLeftward  = "<-"
	directionBoth      = "<->"

	optionTruthy = "t"
	optionFalsy  = "nil"
)

// isClozeStyle reports whether a DECLARED note type is one the cloze renderer
// handles — exactly the stock `Cloze` and the imoogi-owned `imoogi-Cloze`.
//
// Derived from renderType, the existing declared-type-to-renderer mapping,
// rather than from a second list of note-type names. A literal
// []string{"Cloze", "imoogi-Cloze"} would answer identically today and be
// free to drift from the renderer's own dispatch the first time a third name
// appears — which is the whole failure mode this derivation removes.
//
// It lives in the planner rather than in internal/anki/model, the package
// that owns note-type identity, because model's package doc states that
// nothing in it is reachable from an ordinary synchronization run, and that
// isolation is what keeps the per-run "no note-type write" blanket
// verifiable. A predicate the sync hot path calls on every option-bearing
// entry would break that stated invariant.
func isClozeStyle(declaredNoteType string) bool {
	return renderType(declaredNoteType) == orgdoc.NoteTypeCloze
}

// readDirection classifies a direction value. `on` reports whether the option
// is ON — that is, whether the value names one of the three arrows; `ok`
// reports whether the value was recognized at all.
//
// A nil pointer is "the chain resolved to no value", which is off and
// recognized. Arrows are compared after trimming surrounding whitespace; the
// falsy spelling is compared trimmed AND without regard to letter case.
func readDirection(value *string) (on, ok bool) {
	if value == nil {
		return false, true
	}
	trimmed := strings.TrimSpace(*value)
	switch trimmed {
	case directionRightward, directionLeftward, directionBoth:
		return true, true
	}
	if strings.EqualFold(trimmed, optionFalsy) {
		return false, true
	}
	return false, false
}

// readBoolean classifies an incremental or swift value, whose recognized set
// is exactly the truthy and falsy spellings. Both are compared after trimming
// and without regard to case: case-insensitivity costs one comparison and
// removes a whole class of skips whose cause (`T` written for `t`) is
// invisible on a drawer line.
//
// The empty string is deliberately NOT recognized. The front end resolves a
// present-but-empty property to null, so an empty string can only reach here
// from a hand-built or third-party request; rejecting it keeps this side's
// recognized set closed rather than trusting the front end to have normalized.
func readBoolean(value *string) (on, ok bool) {
	if value == nil {
		return false, true
	}
	trimmed := strings.TrimSpace(*value)
	switch {
	case strings.EqualFold(trimmed, optionTruthy):
		return true, true
	case strings.EqualFold(trimmed, optionFalsy):
		return false, true
	default:
		return false, false
	}
}

// validateCardOptions is the single gate spec.md REQ-OPT-008 describes. It
// returns the ONE diagnostic a rejected entry earns, or nil when the entry may
// proceed.
//
// Both the ordinary synchronization path and the migration path reach it, from
// this one implementation, and both call it BEFORE rendering — so a
// mis-specified option is reported against the properties the user wrote
// rather than surfacing later as a rendering or field-resolution failure.
//
// Exactly one diagnostic is emitted per rejected entry, in this fixed order:
//
//	card_option_invalid → card_option_conflict → card_option_needs_cloze
//
// Without a stated order, an entry offending against all three rules would
// have an outcome that depended on evaluation order here — untestable, and apt
// to change under refactoring. The order runs cheapest-and-most-local first: a
// malformed value is a defect in one drawer line and must be reported even
// when the other two rules would also fire, because the conflict rule cannot
// correctly classify a value it does not recognize. The note-type rule runs
// last because it is the one the user is most likely to have intended
// differently, and reporting it while a value is still malformed would send
// them to the wrong line.
//
// An entry for which no card-option property resolved to a value passes
// through unchanged in behavior: no diagnostic, no skip, and no request the
// binary would not otherwise have issued.
//
// declaredNoteType is read from the entry on BOTH paths. On the migration path
// that is the declared name rather than the migration counterpart, which is
// safe because the two always agree on cloze-style-ness: a type and its
// imoogi-owned counterpart are cloze-style together or not at all, by the
// counterpart relation itself.
func validateCardOptions(entry protocol.Entry, key string) *protocol.Error {
	reject := func(code, message string) *protocol.Error {
		return &protocol.Error{Code: code, Message: message, Key: &key}
	}

	directionOn, directionOK := readDirection(entry.Direction)
	incrementalOn, incrementalOK := readBoolean(entry.Incremental)
	swiftOn, swiftOK := readBoolean(entry.Swift)

	// Rule 1 — an unrecognized value. Named with its property and its actual
	// text so the user can find the drawer line that carries it. Checked in a
	// fixed property order so an entry with two malformed values also has one
	// defined outcome.
	for _, candidate := range []struct {
		property string
		value    *string
		ok       bool
		accepts  string
	}{
		{"direction", entry.Direction, directionOK, `"->", "<-", "<->", or "nil"`},
		{"incremental", entry.Incremental, incrementalOK, `"t" or "nil"`},
		{"swift", entry.Swift, swiftOK, `"t" or "nil"`},
	} {
		if !candidate.ok {
			return reject(protocol.CodeCardOptionInvalid, fmt.Sprintf(
				"card option %s has unrecognized value %q; it accepts %s",
				candidate.property, *candidate.value, candidate.accepts))
		}
	}

	// Rule 2 — swift together with a multiline option. The two groups select
	// mutually exclusive card kinds and no precedence between them is defined
	// anywhere; silently choosing one would make the resulting card differ
	// from what the user wrote with no signal.
	//
	// An explicitly falsy value is OFF and therefore not option-bearing, so it
	// does not conflict — which is what keeps the inheritance opt-out usable:
	// because all three properties inherit, a file-level swift plus a
	// heading-level direction conflicts on that heading, and the escape hatch
	// is writing the falsy spelling (or an empty value) in the heading's own
	// drawer.
	if swiftOn && (directionOn || incrementalOn) {
		return reject(protocol.CodeCardOptionConflict, fmt.Sprintf(
			"card option swift is on together with %s; they select different card kinds, so remove one of them",
			multilineOptionNames(directionOn, incrementalOn)))
	}

	// Rule 3 — an option on a note type that cannot carry it. Every card kind
	// these options select is built on a cloze-style note type, so an option
	// on any other type describes a card that cannot be produced. Reporting it
	// here, rather than letting the entry synchronize as though the options
	// were absent, is what keeps the user's stated intent from being silently
	// discarded.
	//
	// Bound to option-BEARING entries: a heading carrying only falsy options
	// has declined them, and must not be told to change its note type for
	// doing so.
	if (directionOn || incrementalOn || swiftOn) && !isClozeStyle(entry.NoteType) {
		return reject(protocol.CodeCardOptionNeedsCloze, fmt.Sprintf(
			"note type %q is not a cloze type, but this entry carries a card option; "+
				"change the note type to a cloze type, or remove the option",
			entry.NoteType))
	}

	return nil
}

// multilineOptionNames names whichever multiline options are on, for the
// conflict diagnostic's detail.
func multilineOptionNames(directionOn, incrementalOn bool) string {
	switch {
	case directionOn && incrementalOn:
		return "direction and incremental"
	case directionOn:
		return "direction"
	default:
		return "incremental"
	}
}
