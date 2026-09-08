package model_test

import (
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/model"
)

// REQ-C-001.1: exactly two imoogi-owned types exist, and the ownership
// predicate is the literal name prefix — decidable from a single modelNames
// response with no local state.
func TestOwnedIsExactlyTheTwoNamedTypesInAFixedOrder(t *testing.T) {
	owned := model.Owned()
	if len(owned) != 2 {
		t.Fatalf("Owned() returned %d types, want exactly 2", len(owned))
	}
	if owned[0].Name != "imoogi-Basic" {
		t.Errorf("Owned()[0].Name = %q, want %q", owned[0].Name, "imoogi-Basic")
	}
	if owned[1].Name != "imoogi-Cloze" {
		t.Errorf("Owned()[1].Name = %q, want %q", owned[1].Name, "imoogi-Cloze")
	}
	for _, spec := range owned {
		if !strings.HasPrefix(spec.Name, model.OwnedPrefix) {
			t.Errorf("%q does not carry the ownership prefix %q", spec.Name, model.OwnedPrefix)
		}
	}
}

// Owned() returns a fresh slice per call. The install step iterates it and a
// caller could sort or truncate it; a shared backing array would let one
// invocation corrupt the next, which is exactly what AC-C-002's idempotence
// assertion would then catch as a mystery.
func TestOwnedReturnsAnIndependentSlicePerCall(t *testing.T) {
	first := model.Owned()
	first[0].Name = "mutated"
	first[0].InOrderFields[0] = "mutated"
	first[0].Templates[0].Front = "mutated"

	second := model.Owned()
	if second[0].Name != "imoogi-Basic" {
		t.Errorf("Owned()[0].Name = %q after a caller mutated an earlier result", second[0].Name)
	}
	if second[0].InOrderFields[0] != "Front" {
		t.Errorf("Owned()[0].InOrderFields[0] = %q after a caller mutated an earlier result", second[0].InOrderFields[0])
	}
	if second[0].Templates[0].Front == "mutated" {
		t.Error("Owned()[0].Templates[0].Front was mutated through an earlier result")
	}
}

// REQ-C-001.2 and design.md §4.2: the field names mirror the stock types
// exactly, so the renderer's output map shape is unchanged and the planner's
// field-resolution layer needs no modification.
func TestOwnedTypesMirrorTheStockFieldNames(t *testing.T) {
	cases := []struct {
		model   string
		fields  []string
		isCloze bool
	}{
		{"imoogi-Basic", []string{"Front", "Back"}, false},
		{"imoogi-Cloze", []string{"Text", "Back Extra"}, true},
	}
	byName := map[string]model.Spec{}
	for _, spec := range model.Owned() {
		byName[spec.Name] = spec
	}
	for _, tc := range cases {
		spec, ok := byName[tc.model]
		if !ok {
			t.Errorf("Owned() carries no %q", tc.model)
			continue
		}
		if len(spec.InOrderFields) != len(tc.fields) {
			t.Errorf("%s has %d fields, want %d", tc.model, len(spec.InOrderFields), len(tc.fields))
			continue
		}
		for i, want := range tc.fields {
			if spec.InOrderFields[i] != want {
				t.Errorf("%s field %d = %q, want %q", tc.model, i, spec.InOrderFields[i], want)
			}
		}
		if spec.IsCloze != tc.isCloze {
			t.Errorf("%s IsCloze = %v, want %v", tc.model, spec.IsCloze, tc.isCloze)
		}
	}
}

// design.md §4.2's template table: each type carries one card, named as that
// table names it, and each side references the fields that side is meant to
// show. A template whose sides referenced no field would render a blank card
// and would still satisfy the wrapper assertion in deckclass_test.go.
func TestTemplatesReferenceTheirOwnFields(t *testing.T) {
	cases := []struct {
		model     string
		card      string
		frontRefs []string
		backRefs  []string
	}{
		{
			model:     "imoogi-Basic",
			card:      "Card 1",
			frontRefs: []string{"{{Front}}"},
			backRefs:  []string{"{{FrontSide}}", "{{Back}}"},
		},
		{
			model:     "imoogi-Cloze",
			card:      "Cloze",
			frontRefs: []string{"{{cloze:Text}}"},
			backRefs:  []string{"{{cloze:Text}}", "{{Back Extra}}"},
		},
	}
	byName := map[string]model.Spec{}
	for _, spec := range model.Owned() {
		byName[spec.Name] = spec
	}
	for _, tc := range cases {
		spec := byName[tc.model]
		if len(spec.Templates) != 1 {
			t.Errorf("%s carries %d templates, want exactly 1", tc.model, len(spec.Templates))
			continue
		}
		tpl := spec.Templates[0]
		if tpl.Name != tc.card {
			t.Errorf("%s template name = %q, want %q", tc.model, tpl.Name, tc.card)
		}
		for _, ref := range tc.frontRefs {
			if !strings.Contains(tpl.Front, ref) {
				t.Errorf("%s front does not reference %s", tc.model, ref)
			}
		}
		for _, ref := range tc.backRefs {
			if !strings.Contains(tpl.Back, ref) {
				t.Errorf("%s back does not reference %s", tc.model, ref)
			}
		}
	}
}

// The templates are embedded assets, and an embed of a missing-but-globbed
// or empty file compiles cleanly. Non-emptiness is therefore asserted rather
// than assumed.
func TestEveryTemplateSideIsNonEmpty(t *testing.T) {
	for _, spec := range model.Owned() {
		for _, tpl := range spec.Templates {
			if strings.TrimSpace(tpl.Front) == "" {
				t.Errorf("%s template %q has an empty front", spec.Name, tpl.Name)
			}
			if strings.TrimSpace(tpl.Back) == "" {
				t.Errorf("%s template %q has an empty back", spec.Name, tpl.Name)
			}
		}
	}
}

// REQ-C-007's air-gap clause reaches the templates too: a card template can
// pull a remote script or stylesheet just as a stylesheet can, and Anki
// renders it inside a real web view.
//
// The templates DO carry one script — REQ-C-006's review-time deck-class
// hook, inline and source-less — so the bare `<script>` tag is permitted
// and every construct that would make a script or the page reach the
// network is forbidden instead: a src attribute, a URL scheme, a fetch,
// a dynamic import, a request object.
func TestTemplatesReachNoNetworkResource(t *testing.T) {
	forbidden := []string{"htt" + "p", "url" + "(", "//", "src" + "=", "<scr" + "ipt ", "fet" + "ch(", "imp" + "ort(", "XMLHttp" + "Request", "send" + "Beacon"}
	for _, spec := range model.Owned() {
		for _, tpl := range spec.Templates {
			for side, text := range map[string]string{"front": tpl.Front, "back": tpl.Back} {
				for _, needle := range forbidden {
					if strings.Contains(text, needle) {
						t.Errorf("%s template %q %s carries %q", spec.Name, tpl.Name, side, needle)
					}
				}
				// Every script tag is the bare, attribute-less one.
				if bare, all := strings.Count(text, "<scr"+"ipt>"), strings.Count(text, "<scr"+"ipt"); bare != all {
					t.Errorf("%s template %q %s carries %d script tags but only %d bare ones", spec.Name, tpl.Name, side, all, bare)
				}
			}
		}
	}
}

// REQ-C-005.2 and design.md §4.2: the stock names are NOT imoogi's own. The
// prefix predicate is the only ownership test in the system, so the two stock
// names failing it is the property every negative assertion downstream rests
// on.
func TestIsOwnedIsTheLiteralPrefixPredicate(t *testing.T) {
	cases := []struct {
		name  string
		owned bool
	}{
		{"imoogi-Basic", true},
		{"imoogi-Cloze", true},
		{"imoogi-Other", true},
		{"imoogi-", true},
		{"Basic", false},
		{"Cloze", false},
		{"Basic (and reversed card)", false},
		{"My imoogi-Basic", false},
		{"IMOOGI-Basic", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := model.IsOwned(tc.name); got != tc.owned {
			t.Errorf("IsOwned(%q) = %v, want %v", tc.name, got, tc.owned)
		}
	}
}
