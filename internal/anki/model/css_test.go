package model_test

import (
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/model"
)

// Every literal below is transcribed from REQ-C-007.2's own table and from
// AC-C-005's prose, NOT read back off the stylesheet. A test that quotes the
// asset it checks asserts nothing.

// AC-C-005, clause 1: the two night-mode selector forms, the image rule, and
// the wrapping rule.
func TestBaseStylesheetCarriesTheStructuralRules(t *testing.T) {
	css := model.BaseCSS()
	for _, want := range []string{
		".card.nightMode",
		".card.night_mode",
		"max-width: 100%",
		"max-height: none",
		"word-wrap: break-word",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("base stylesheet is missing %q", want)
		}
	}
	// AnkiDroid's form has NO space between the class names. A stylesheet
	// carrying `.card .night_mode` instead would select a descendant and
	// silently theme nothing, which the substring above cannot distinguish
	// on its own — so the descendant form is excluded explicitly.
	if strings.Contains(css, ".card .night_mode") {
		t.Error("base stylesheet carries the descendant form `.card .night_mode`; AnkiDroid's form has no space")
	}
	if strings.Contains(css, ".card .nightMode") {
		t.Error("base stylesheet carries the descendant form `.card .nightMode`")
	}
}

// AC-C-005, clause 2, light half: the three themed properties carry their
// light values on `.card`.
func TestBaseStylesheetDeclaresTheLightCustomProperties(t *testing.T) {
	css := model.BaseCSS()
	for _, want := range []string{
		"--imoogi-bg: #f7f3e9",
		"--imoogi-fg: #1c1a17",
		"--imoogi-emphasis-fg: #0b0a09",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("base stylesheet is missing the light declaration %q", want)
		}
	}
}

// AC-C-005, clause 2, night half: the three themed properties carry their
// night values, and do so under BOTH selector forms. A selector list naming
// both forms satisfies this; two separate blocks would too. What must not
// happen is the values landing under only one form, so the assertion is that
// each night value appears within a block whose selector names both.
func TestBaseStylesheetDeclaresTheNightCustomPropertiesUnderBothSelectors(t *testing.T) {
	css := model.BaseCSS()
	nightValues := []string{
		"--imoogi-bg: #1a1a1a",
		"--imoogi-fg: #c8c4bc",
		"--imoogi-emphasis-fg: #f5f2ec",
	}
	for _, want := range nightValues {
		if !strings.Contains(css, want) {
			t.Errorf("base stylesheet is missing the night declaration %q", want)
		}
	}
	for _, selector := range []string{".card.nightMode", ".card.night_mode"} {
		for _, want := range nightValues {
			if !declaredUnder(css, selector, want) {
				t.Errorf("night declaration %q is not in force under %q", want, selector)
			}
		}
	}
}

// AC-C-005, clause 2, remainder: the four unthemed properties are declared
// exactly ONCE each, with the literal values REQ-C-007.2 fixes. "Once each"
// is asserted as a count, because a second declaration is how a value drifts
// between the light and night halves without any single substring failing.
func TestBaseStylesheetDeclaresTheUnthemedCustomPropertiesExactlyOnce(t *testing.T) {
	css := model.BaseCSS()
	cases := []struct {
		property string
		literal  string
	}{
		{"--imoogi-measure", "--imoogi-measure: 44rem"},
		{"--imoogi-line-height", "--imoogi-line-height: 1.6"},
		{"--imoogi-serif", "--imoogi-serif: Iowan Old Style, Palatino Linotype, Palatino, Georgia, serif"},
		{"--imoogi-sans", "--imoogi-sans: -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, Helvetica Neue, Arial, sans-serif"},
	}
	for _, tc := range cases {
		if !strings.Contains(css, tc.literal) {
			t.Errorf("base stylesheet is missing %q", tc.literal)
		}
		// A declaration is `--name:`; a consumption is `var(--name)`. Only
		// the former is counted.
		if got := strings.Count(css, tc.property+":"); got != 1 {
			t.Errorf("%s is declared %d times, want exactly 1", tc.property, got)
		}
	}
}

// AC-C-005, clause 3: each custom property is consumed exactly where
// REQ-C-007.2 says it is. A property declared and never consumed themes
// nothing, so the declaration assertions above are not sufficient on their
// own.
func TestBaseStylesheetConsumesEachCustomPropertyAtItsNamedSite(t *testing.T) {
	css := model.BaseCSS()
	cases := []struct {
		site        string
		selector    string
		declaration string
	}{
		{"card ground", ".card", "background: var(--imoogi-bg)"},
		{"card ink", ".card", "color: var(--imoogi-fg)"},
		{"card body face", ".card", "font-family: var(--imoogi-sans)"},
		{"card rhythm", ".card", "line-height: var(--imoogi-line-height)"},
		{"heading face", "h6", "font-family: var(--imoogi-serif)"},
		{"deck wrapper measure", ".imoogi-deck", "max-width: var(--imoogi-measure)"},
		{"math and code contrast", "code", "color: var(--imoogi-emphasis-fg)"},
	}
	for _, tc := range cases {
		if !strings.Contains(css, tc.declaration) {
			t.Errorf("%s: base stylesheet is missing %q", tc.site, tc.declaration)
			continue
		}
		if !declaredUnder(css, tc.selector, tc.declaration) {
			t.Errorf("%s: %q is not in force under a rule naming %q", tc.site, tc.declaration, tc.selector)
		}
	}
	// The heading rule spans h1 through h6 (REQ-C-007.2 names the range, not
	// a single level), so every level is asserted rather than just the one
	// the site table above pins.
	for _, h := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		if !declaredUnder(css, h, "font-family: var(--imoogi-serif)") {
			t.Errorf("the serif face is not in force for %s", h)
		}
	}
}

// AC-C-005's air-gap clause, and the one assertion in this file that is a
// security property rather than an appearance one: the embedded stylesheet
// reaches no network resource, by construction and mechanically. Comments
// count — a commented-out font import is still a string in the asset, and a
// later edit that uncomments it would pass a laxer check.
func TestBaseStylesheetReachesNoNetworkResource(t *testing.T) {
	css := model.BaseCSS()
	// Assembled at runtime so this test file itself does not contain the
	// forbidden literals — otherwise a grep of the package for them, which
	// is how a reviewer checks this property by hand, returns the test.
	forbidden := []string{"htt" + "p", "@" + "import", "url" + "(", "//fonts", "src:"}
	for _, needle := range forbidden {
		if strings.Contains(css, needle) {
			t.Errorf("base stylesheet carries the network-fetch construct %q", needle)
		}
	}
}

// REQ-C-009: no rule of imoogi's own targets a NAMED deck. The generic
// wrapper rule required by REQ-C-007.2 selects the stable `imoogi-deck` class
// every template emits and is deck-agnostic, so it is not a violation; a
// `.deck-<something>` class selector would be.
//
// The stable class is what keeps the measure rule in force even where the
// template script does not run; the normalized per-deck class is added to
// the same element at review time and is for the user stylesheet alone.
func TestBaseStylesheetCarriesNoPerDeckRule(t *testing.T) {
	css := model.BaseCSS()
	if !strings.Contains(css, ".card .imoogi-deck") {
		t.Error("base stylesheet carries no generic deck-wrapper rule on .imoogi-deck")
	}
	if strings.Contains(css, `[class^="deck-"]`) {
		t.Error("base stylesheet still carries the attribute-prefix rule that assumed an unnormalized class")
	}
	if strings.Contains(css, ".deck-") {
		t.Error("base stylesheet carries a class selector naming a specific deck; REQ-C-009 forbids per-deck styling")
	}
}

// AC-C-006a: a supplied user stylesheet is appended VERBATIM after the base,
// in that order, with nothing inserted between and nothing rewritten.
func TestUploadCSSAppendsTheUserStylesheetVerbatim(t *testing.T) {
	const marker = "/* MARKER */ .card { letter-spacing: 0.01em; }\n"
	got := model.UploadCSS(marker)
	want := model.BaseCSS() + marker
	if got != want {
		t.Errorf("UploadCSS did not produce base+user verbatim\n got: %q\nwant: %q", got, want)
	}
	if !strings.HasPrefix(got, model.BaseCSS()) {
		t.Error("uploaded CSS does not begin with the base stylesheet")
	}
	if !strings.HasSuffix(got, marker) {
		t.Error("uploaded CSS does not end with the user stylesheet")
	}
}

// AC-C-006b: an absent user stylesheet arrives as the empty string, and the
// uploaded CSS is then the base alone — byte-identical, with no trailing
// separator introduced by the concatenation.
func TestUploadCSSWithNoUserStylesheetIsTheBaseAlone(t *testing.T) {
	if got := model.UploadCSS(""); got != model.BaseCSS() {
		t.Errorf("UploadCSS(\"\") = %d bytes, want the base stylesheet's %d unchanged", len(got), len(model.BaseCSS()))
	}
}

// AC-C-006c: the base portion does not vary with anything. BaseCSS takes no
// argument at all, so the property holds by signature; this asserts the
// weaker but checkable form — repeated reads are byte-identical, and the base
// prefix of two uploads built from different user stylesheets is the same.
func TestBaseStylesheetIsInvariantAcrossReadsAndUploads(t *testing.T) {
	if model.BaseCSS() != model.BaseCSS() {
		t.Fatal("BaseCSS is not stable across calls")
	}
	base := model.BaseCSS()
	for _, user := range []string{"", "/* deck A */\n", "/* an entirely different collection */\n"} {
		if got := model.UploadCSS(user)[:len(base)]; got != base {
			t.Errorf("the base portion of the upload varied for user CSS %q", user)
		}
	}
}

// The stylesheet is a real asset rather than an accidentally-empty embed. An
// empty file embeds without error and would pass a `strings.Contains("")`
// style check, so its non-emptiness is asserted directly.
func TestBaseStylesheetIsNotEmpty(t *testing.T) {
	if len(strings.TrimSpace(model.BaseCSS())) == 0 {
		t.Fatal("the embedded base stylesheet is empty")
	}
}

// declaredUnder reports whether declaration appears inside a CSS rule whose
// selector names selector. The stylesheet is imoogi's own asset with no
// nesting and no at-rules, so a brace-delimited scan is exact here rather
// than an approximation of a CSS parser.
func declaredUnder(css, selector, declaration string) bool {
	rest := css
	for {
		open := strings.Index(rest, "{")
		if open < 0 {
			return false
		}
		close := strings.Index(rest[open:], "}")
		if close < 0 {
			return false
		}
		close += open
		if strings.Contains(rest[:open], selector) && strings.Contains(rest[open:close], declaration) {
			return true
		}
		rest = rest[close+1:]
	}
}
