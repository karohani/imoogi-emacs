package model

import "strings"

// classPrefix is the literal every deck class begins with. The wrapper the
// card templates emit is `class="deck-{{Deck}}"`, so this constant and the
// templates' literal are two spellings of one convention; the templates are
// the wire form and this is the Go form.
const classPrefix = "deck-"

// unnamedToken is REQ-C-006.2's empty-token sentinel. A deck made entirely of
// separators or punctuation normalizes to nothing, and the bare `deck-` class
// that would result is a valid CSS identifier but is also the exact prefix
// every other deck class begins with — a user rule written as `.deck-` would
// be indistinguishable from a typo. The sentinel makes the empty case
// nameable and selectable. It is not an error and raises no diagnostic.
const unnamedToken = "unnamed"

// NormalizeDeckClass maps Anki's {{Deck}} value — the full deck path, `::`
// separators included — to a CSS-identifier-safe token, as the total function
// REQ-C-006.2 and design.md §3.3 fix:
//
//  1. ASCII case fold
//  2. each `::` separator becomes ONE hyphen
//  3. every remaining rune outside [a-z0-9_-] becomes a hyphen
//  4. hyphen runs collapse to one
//  5. leading and trailing hyphens are stripped
//  6. a digit-leading result takes the `_` guard character
//  7. an empty result becomes the `unnamed` sentinel
//
// Step 2 runs before step 3 so a `::` yields one hyphen rather than two; step
// 4 then keeps every other run to one as well.
//
// Step 6 is defensive rather than load-bearing HERE: the emitted class is
// `deck-` + token, which already begins with a letter, so a digit-leading
// token could not produce an invalid class in this SPEC's own usage. It is
// implemented anyway so the function is correct standalone and stays correct
// if a later SPEC consumes the token without the prefix.
func NormalizeDeckClass(deck string) string {
	// Step 1 — ASCII case fold, done rune-wise rather than through
	// strings.ToLower so the fold is exactly the ASCII one design.md §3.3
	// names. Unicode folding would be indistinguishable in the result (step 3
	// maps every non-ASCII rune to a hyphen regardless) but would introduce a
	// locale-shaped behavior into a function specified as byte-mechanical.
	var folded strings.Builder
	folded.Grow(len(deck))
	for _, r := range deck {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		folded.WriteRune(r)
	}

	// Step 2 — each separator becomes one hyphen.
	s := strings.ReplaceAll(folded.String(), "::", "-")

	// Steps 3 and 4 — map every rune outside the permitted class to a hyphen,
	// collapsing runs as they are produced rather than in a second pass.
	var mapped strings.Builder
	mapped.Grow(len(s))
	lastWasHyphen := false
	for _, r := range s {
		permitted := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_'
		switch {
		case permitted:
			mapped.WriteRune(r)
			lastWasHyphen = false
		case lastWasHyphen:
			// Inside a hyphen run — step 4 collapses it, so emit nothing.
		default:
			mapped.WriteByte('-')
			lastWasHyphen = true
		}
	}

	// Step 5 — strip the edges.
	token := strings.Trim(mapped.String(), "-")

	// Step 7, hoisted above step 6 because an empty token has no first byte to
	// test. The two are disjoint (the sentinel begins with a letter), so the
	// order between them is free.
	if token == "" {
		return unnamedToken
	}

	// Step 6 — the digit guard. `_` is inside the permitted class and is
	// distinct from the `-` separator, so it cannot be confused with a
	// stripped edge.
	if token[0] >= '0' && token[0] <= '9' {
		return "_" + token
	}
	return token
}

// DeckClass returns the full CSS class the card-template wrapper carries for
// a deck: the `deck-` prefix followed by the normalized token. It never
// returns the bare prefix, because NormalizeDeckClass never returns the empty
// string (REQ-C-006.2).
func DeckClass(deck string) string {
	return classPrefix + NormalizeDeckClass(deck)
}
