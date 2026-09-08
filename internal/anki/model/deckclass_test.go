package model_test

import (
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/model"
	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
)

// deckClassTable is AC-C-004's table verbatim, plus the rationale rows
// design.md §3.3 names but the table does not enumerate. It is package-level
// rather than local to one test because two tests consume it: the Go
// normalizer's own assertion below, and the Go-vs-JS mirror in
// deckclass_script_test.go. The Elisp mirror (tests/anki-notetype-test.el)
// reads these rows off this file by regexp, so each row stays on one line in
// the {"name", "deck", "token", "class"} shape.
var deckClassTable = []struct {
	name  string
	deck  string
	token string
	class string
}{
	// The nine AC-C-004 rows.
	{"parenthesized path, the user's real deck", "(PROGRAMMER)::(GO)", "programmer-go", "deck-programmer-go"},
	{"ordinary two-level path", "Geography::Europe", "geography-europe", "deck-geography-europe"},
	{"consecutive separators collapse to one hyphen", "A::::B", "a-b", "deck-a-b"},
	{"digit-leading token takes the underscore guard", "2026 Review", "_2026-review", "deck-_2026-review"},
	{"surrounding and interior spaces", "  Spaced  Name  ", "spaced-name", "deck-spaced-name"},
	{"underscore is inside the permitted class", "Math_Notes", "math_notes", "deck-math_notes"},
	{"separator-only deck yields the sentinel", "::", "unnamed", "deck-unnamed"},
	{"hyphen-only deck yields the sentinel", "---", "unnamed", "deck-unnamed"},
	{"punctuation-only deck yields the sentinel", "!!!", "unnamed", "deck-unnamed"},

	// Rows design.md §3.3 reasons about explicitly.
	{"the empty deck is a sentinel, not a bare prefix", "", "unnamed", "deck-unnamed"},
	{"a single separator level is a plain hyphen", "A::B", "a-b", "deck-a-b"},
	{"non-ASCII runes are outside the permitted class", "한국어", "unnamed", "deck-unnamed"},
	{"mixed script keeps only the permitted runes", "Go::한국어::Notes", "go-notes", "deck-go-notes"},
	{"uppercase folds to lower", "GO", "go", "deck-go"},
}

// deckClassAdversarialInputs are the inputs outside the table that the
// permitted-token property is asserted over: control characters, a quote,
// a non-BMP rune, digit- and underscore-leading names. The mirror test feeds
// these to the JS normalizer too, since a carrier that survives a quote or a
// newline is exactly what review time has to prove.
var deckClassAdversarialInputs = []string{
	"(PROGRAMMER)::(GO)", "2026 Review", "::", "---", "!!!", "",
	"a.b/c", "Tab\tSeparated", "New\nLine", `Quote"Mark`, "semi;colon",
	"한국어", "emoji-🐉", "9lives", "__leading", "trailing__",
}

// The normalizer is a total function, so every row asserts a value rather
// than an error.
func TestNormalizeDeckClassMatchesTheAcceptanceTable(t *testing.T) {
	for _, tc := range deckClassTable {
		t.Run(tc.name, func(t *testing.T) {
			if got := model.NormalizeDeckClass(tc.deck); got != tc.token {
				t.Errorf("NormalizeDeckClass(%q) = %q, want %q", tc.deck, got, tc.token)
			}
			if got := model.DeckClass(tc.deck); got != tc.class {
				t.Errorf("DeckClass(%q) = %q, want %q", tc.deck, got, tc.class)
			}
		})
	}
}

// REQ-C-006.2's own closing clause: the emitted class is never the bare
// `deck-`, because that string is the exact prefix every real deck class
// begins with and a user rule written as `.deck-` would be indistinguishable
// from a typo.
func TestDeckClassIsNeverTheBarePrefix(t *testing.T) {
	for _, deck := range []string{"", "::", "---", "!!!", "   ", "::::", "-", "?"} {
		if got := model.DeckClass(deck); got == "deck-" {
			t.Errorf("DeckClass(%q) = %q, want the sentinel class rather than the bare prefix", deck, got)
		}
	}
}

// The token is a valid CSS identifier tail for every input, which is the
// property step 3 (rune map), step 6 (digit guard), and step 7 (sentinel)
// exist to establish. Asserted over the acceptance inputs plus adversarial
// ones rather than only the table.
func TestNormalizeDeckClassAlwaysProducesAPermittedToken(t *testing.T) {
	for _, deck := range deckClassAdversarialInputs {
		token := model.NormalizeDeckClass(deck)
		if token == "" {
			t.Errorf("NormalizeDeckClass(%q) produced the empty token; step 7's sentinel did not fire", deck)
			continue
		}
		if c := token[0]; c >= '0' && c <= '9' {
			t.Errorf("NormalizeDeckClass(%q) = %q begins with a digit; step 6's guard did not fire", deck, token)
		}
		if strings.HasPrefix(token, "-") || strings.HasSuffix(token, "-") {
			t.Errorf("NormalizeDeckClass(%q) = %q carries an edge hyphen; step 5's trim did not fire", deck, token)
		}
		if strings.Contains(token, "--") {
			t.Errorf("NormalizeDeckClass(%q) = %q carries a hyphen run; step 4's collapse did not fire", deck, token)
		}
		for _, r := range token {
			permitted := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
			if !permitted {
				t.Errorf("NormalizeDeckClass(%q) = %q carries the forbidden rune %q", deck, token, r)
			}
		}
	}
}

// AC-C-004's second clause: the wrapper lives in the card template and
// NOWHERE else. Anki expands {{Deck}} at review time only inside a template,
// so a wrapper that reached stored field content would both fail to expand
// and enter the content hash — design.md §3.1's "invisible to the hash".
//
// The wrapper's own class is the stable `imoogi-deck`, NOT `deck-{{Deck}}`:
// Anki expands {{Deck}} verbatim, so a class attribute built from it carries
// the deck's punctuation and spaces and the normalized class REQ-C-006.2
// promises would never exist (sync-audit F1). The deck path instead travels
// as the text of a carrier element the template script reads, and the
// normalized `deck-<token>` is ADDED to the wrapper at review time — see
// deckclass_script_test.go for the script's own contract.
func TestDeckWrapperAppearsInEveryTemplateSide(t *testing.T) {
	const (
		wrapper = `class="imoogi-deck"`
		carrier = `class="imoogi-deck-name" style="display:none">{{Deck}}</span>`
		rawForm = `class="deck-{{Deck}}"`
	)
	for _, spec := range model.Owned() {
		for _, tpl := range spec.Templates {
			for side, body := range map[string]string{"front": tpl.Front, "back": tpl.Back} {
				if n := strings.Count(body, wrapper); n != 1 {
					t.Errorf("%s template %q %s carries %d wrappers %s, want exactly 1", spec.Name, tpl.Name, side, n, wrapper)
				}
				if n := strings.Count(body, carrier); n != 1 {
					t.Errorf("%s template %q %s carries %d deck-name carriers, want exactly 1", spec.Name, tpl.Name, side, n)
				}
				if strings.Contains(body, rawForm) {
					t.Errorf("%s template %q %s still emits the raw %s wrapper; the class never normalizes at review time", spec.Name, tpl.Name, side, rawForm)
				}
			}
		}
	}
}

// The complement of the assertion above, and the one AC-C-004 states as a
// grep: no field value the renderer produces carries the wrapper. The
// renderer is exercised over both note types and over a body carrying the
// constructs most likely to smuggle a div through — a block, a list, and
// inline markup.
func TestRenderedFieldValuesCarryNoDeckWrapper(t *testing.T) {
	const body = "Some *bold* text.\n\n- a list item\n- another\n\n#+begin_quote\nquoted\n#+end_quote\n"
	cases := []struct{ noteType, title, body string }{
		{orgdoc.NoteTypeBasic, "A title", body},
		{orgdoc.NoteTypeCloze, "A title", "The {{c1::answer}} is here.\n\n" + body},
	}
	for _, tc := range cases {
		fields, err := orgdoc.Render(tc.noteType, tc.title, tc.body)
		if err != nil {
			t.Fatalf("Render(%q) failed: %v", tc.noteType, err)
		}
		if len(fields) == 0 {
			t.Fatalf("Render(%q) produced no fields", tc.noteType)
		}
		for name, value := range fields {
			if strings.Contains(value, `class="deck-`) || strings.Contains(value, "imoogi-deck") {
				t.Errorf("rendered field %q of %s carries the deck wrapper: %q", name, tc.noteType, value)
			}
		}
	}
}
