package model

import (
	_ "embed"
	"strings"
)

// deckClassJS is the review-time half of REQ-C-006: the script every card
// template carries, which reads the deck path Anki expanded into the
// template's hidden carrier and adds the normalized `deck-<token>` class to
// the wrapper. NormalizeDeckClass is the reference implementation; the
// mirror test runs this asset under node over the acceptance rows so the two
// cannot drift apart silently.
//
//go:embed assets/deckclass.js
var deckClassJS string

// deckClassPlaceholder marks, in each raw template asset, where the script
// goes. There is one script source and four template sides; substituting at
// embed time is what keeps the four from becoming hand-copied duplicates.
// It is an HTML comment rather than a {{…}} form because Anki would try to
// expand the latter as a field reference.
const deckClassPlaceholder = "<!-- imoogi:deck-class-script -->"

// @MX:NOTE: [AUTO] the four installed template bodies are derived from the
// raw assets here, never uploaded raw — a raw asset still carries the
// placeholder and no script.
//
// withDeckClassScript returns a raw template asset with its placeholder
// replaced by the inline script tag. The replacement is unconditional: a
// template that carried no placeholder would come back unchanged, and the
// carriage test is what asserts that never happens.
func withDeckClassScript(raw string) string {
	return strings.Replace(raw, deckClassPlaceholder, "<script>"+deckClassJS+"</script>", 1)
}

// DeckClassScript returns the embedded template script verbatim — the bytes
// each installed template side carries between its script tags.
func DeckClassScript() string { return deckClassJS }
