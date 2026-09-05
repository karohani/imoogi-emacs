package model

import _ "embed"

// baseCSS is the imoogi base stylesheet, embedded at build time rather than
// read from disk (design.md §4.1). Two properties follow: the air-gap rule
// cannot be violated by a missing file at runtime, and the asset stays a real
// .css file that an editor and a linter can both read — which a Go string
// constant of this size would not.
//
//go:embed assets/base.css
var baseCSS string

// BaseCSS returns the embedded base stylesheet.
//
// It takes no argument, which is REQ-C-009's "byte-identical for every deck"
// enforced by signature rather than by assertion: there is nothing a deck
// name could be passed as.
func BaseCSS() string { return baseCSS }

// UploadCSS returns the CSS the install step uploads: the base stylesheet
// followed VERBATIM by the user stylesheet, in that order (REQ-C-008).
//
// Nothing is inserted between the two and nothing is rewritten. A separator —
// even a newline — would be a byte the user did not write appearing inside
// their own stylesheet's byte range, and REQ-C-008 says verbatim. The base
// asset already ends in a newline, so the concatenation is well-formed CSS
// without one.
//
// An absent user stylesheet arrives as the empty string (the front end is the
// only reader of that file; this package reads no stylesheet from disk), and
// the result is then the base alone, byte-identical.
func UploadCSS(userCSS string) string { return baseCSS + userCSS }
