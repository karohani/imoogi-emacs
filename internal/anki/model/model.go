// Package model owns imoogi's two note types: their names, their field
// lists, their embedded card templates, the embedded base stylesheet, the
// deck-class normalizer and its review-time script counterpart the
// templates carry, and the probe-then-act install step that puts all of it
// into a collection.
//
// Nothing in this package is reachable from an ordinary synchronization run.
// That isolation is the whole point: the parent SPEC's per-run "no note-type
// write" blanket survives verbatim on the hot path (AC-C-003a) and is lifted
// only on a command the user explicitly invokes (AC-C-003b).
//
// The package reads no file from disk. The templates, the deck-class script
// they carry, and the base stylesheet are embedded at build time (design.md
// §4.1) — the air-gap rule forbids a
// network fetch, and a stylesheet of this size as a Go string constant would
// be neither readable nor lintable. The USER stylesheet is not embedded and
// is not read here either: it arrives as text in the install request, read by
// the front end (REQ-C-008).
package model

import (
	_ "embed"

	"github.com/karohani/imoogi-emacs/internal/anki/ankiconnect"
)

// OwnedPrefix is imoogi's note-type ownership predicate (REQ-C-001.1). It is
// name-based rather than registry-based on purpose: it is decidable from a
// single modelNames response with no local state, which is what makes the
// assertion "this request targeted an imoogi-owned model" checkable from a
// request log alone.
const OwnedPrefix = "imoogi-"

// The two — and only two — imoogi-owned note types.
const (
	BasicName = OwnedPrefix + "Basic"
	ClozeName = OwnedPrefix + "Cloze"
)

// IsOwned reports whether a model name is imoogi's. Any other name is
// foreign, and no code path in this package writes a foreign model
// (REQ-C-004).
//
// A name that carries the prefix but is neither BasicName nor ClozeName —
// `imoogi-Other`, say — is owned by this predicate yet is still neither
// created, updated, nor reported: Owned() enumerates exactly two types and
// the install step iterates that enumeration, so REQ-C-001.1's passthrough
// clause holds by construction rather than by a guard.
func IsOwned(modelName string) bool {
	return len(modelName) >= len(OwnedPrefix) && modelName[:len(OwnedPrefix)] == OwnedPrefix
}

// Spec is one imoogi-owned note type's complete definition — everything
// createModel needs except the CSS, which is per-invocation (it carries the
// user stylesheet) rather than per-type.
type Spec struct {
	Name string
	// InOrderFields mirrors the stock counterpart's field names exactly
	// (REQ-C-001.2), so the renderer's output map shape is unchanged and the
	// planner's field-resolution layer needs no modification.
	InOrderFields []string
	// IsCloze selects createModel's cloze behavior. Same action, no separate
	// endpoint (design.md §4.2).
	IsCloze   bool
	Templates []ankiconnect.CardTemplate
}

//go:embed assets/basic-front.html
var basicFront string

//go:embed assets/basic-back.html
var basicBack string

//go:embed assets/cloze-front.html
var clozeFront string

//go:embed assets/cloze-back.html
var clozeBack string

// Owned returns the two imoogi-owned note types in a FIXED order —
// imoogi-Basic then imoogi-Cloze. The order is load-bearing: the install step
// iterates it, so a map's iteration order would make AC-C-002's
// byte-identical-request-log assertion fail at random.
//
// A fresh slice is built per call, backed by fresh sub-slices, so a caller
// that sorts or edits its result cannot corrupt the next invocation.
func Owned() []Spec {
	return []Spec{
		{
			Name:          BasicName,
			InOrderFields: []string{"Front", "Back"},
			IsCloze:       false,
			Templates: []ankiconnect.CardTemplate{
				{Name: "Card 1", Front: withDeckClassScript(basicFront), Back: withDeckClassScript(basicBack)},
			},
		},
		{
			Name:          ClozeName,
			InOrderFields: []string{"Text", "Back Extra"},
			IsCloze:       true,
			Templates: []ankiconnect.CardTemplate{
				{Name: "Cloze", Front: withDeckClassScript(clozeFront), Back: withDeckClassScript(clozeBack)},
			},
		},
	}
}
