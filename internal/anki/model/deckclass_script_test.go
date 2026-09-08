package model_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/model"
)

// deckClassPlaceholder is the marker the raw template assets carry where the
// script goes. It is an HTML comment rather than a {{…}} form because Anki
// would try to expand the latter as a field reference.
const deckClassPlaceholder = "<!-- imoogi:deck-class-script -->"

// mirrorHarness is the node program the mirror test runs. It executes the
// embedded script — the very bytes the card runs — inside a vm context whose
// `document` is a stub of the one thing the script touches: a query for the
// deck-name carriers, each with a textContent and a parent wrapper whose
// classList records what was added. The stub's classList de-duplicates like
// the real one, and the script is run twice per deck so a second render adds
// nothing new. Inputs arrive as JSON on stdin; results leave as JSON on
// stdout. Nothing from Go is interpolated into the program text.
const mirrorHarness = `"use strict";
const fs = require("fs");
const vm = require("vm");
const input = JSON.parse(fs.readFileSync(0, "utf8"));
const results = input.decks.map(function (deck) {
  const added = [];
  const wrapper = { classList: { add: function (c) { if (added.indexOf(c) < 0) { added.push(c); } } } };
  const carrier = { textContent: deck, parentNode: wrapper };
  const document = { querySelectorAll: function (sel) { return sel === ".imoogi-deck-name" ? [carrier] : []; } };
  vm.runInNewContext(input.script, { document: document });
  vm.runInNewContext(input.script, { document: document });
  return added;
});
process.stdout.write(JSON.stringify(results));
`

// AC-C-004 at the surface that matters: the class the card actually gets at
// review time. The JS normalizer inside the template script must agree with
// NormalizeDeckClass — the Go function is the reference — on every row of
// the acceptance table and on every adversarial input the Go property test
// uses, including the quote, the tab, the newline, and the non-BMP rune.
//
// node is required to run the script; when it is absent the test skips
// loudly rather than passing vacuously.
func TestDeckClassScriptMirrorsTheGoNormalizer(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on PATH; the Go-vs-JS deck-class mirror cannot run here")
	}

	var decks []string
	for _, row := range deckClassTable {
		decks = append(decks, row.deck)
	}
	decks = append(decks, deckClassAdversarialInputs...)

	harness := filepath.Join(t.TempDir(), "mirror.js")
	if err := os.WriteFile(harness, []byte(mirrorHarness), 0o600); err != nil {
		t.Fatalf("writing the harness: %v", err)
	}
	stdin, err := json.Marshal(map[string]any{"script": model.DeckClassScript(), "decks": decks})
	if err != nil {
		t.Fatalf("encoding the inputs: %v", err)
	}

	cmd := exec.Command(node, harness)
	cmd.Stdin = bytes.NewReader(stdin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("node failed: %v\nstderr:\n%s", err, stderr.String())
	}
	var got [][]string
	if err := json.Unmarshal(stdout, &got); err != nil {
		t.Fatalf("decoding node's output %q: %v", stdout, err)
	}
	if len(got) != len(decks) {
		t.Fatalf("node returned %d results for %d decks", len(got), len(decks))
	}

	for i, deck := range decks {
		want := []string{model.DeckClass(deck)}
		if len(got[i]) != 1 || got[i][0] != want[0] {
			t.Errorf("deck %q: the script added %q, Go says %q", deck, got[i], want)
		}
	}
	t.Logf("mirrored %d deck names through node", len(decks))
}

// The script is one asset carried by every template side exactly once. The
// four installed bodies are produced by substituting the placeholder in the
// raw assets, so no hand-copied duplicate can drift from the source.
func TestEveryTemplateSideCarriesTheDeckClassScriptExactlyOnce(t *testing.T) {
	js := model.DeckClassScript()
	if strings.TrimSpace(js) == "" {
		t.Fatal("DeckClassScript() is empty")
	}
	tag := "<script>" + js + "</script>"
	for _, spec := range model.Owned() {
		for _, tpl := range spec.Templates {
			for side, body := range map[string]string{"front": tpl.Front, "back": tpl.Back} {
				if n := strings.Count(body, tag); n != 1 {
					t.Errorf("%s template %q %s carries the deck-class script %d times, want exactly 1", spec.Name, tpl.Name, side, n)
				}
				if strings.Contains(body, deckClassPlaceholder) {
					t.Errorf("%s template %q %s still carries the placeholder; the script was not substituted", spec.Name, tpl.Name, side)
				}
			}
		}
	}
}

// The snippet source is a single file under assets/, and the raw template
// assets carry the placeholder — never a script of their own — so the one
// file is the only place the normalizer can live.
func TestDeckClassScriptSourceIsASingleAsset(t *testing.T) {
	entries, err := os.ReadDir("assets")
	if err != nil {
		t.Fatalf("reading assets/: %v", err)
	}
	var scripts, templates []string
	for _, e := range entries {
		switch filepath.Ext(e.Name()) {
		case ".js":
			scripts = append(scripts, e.Name())
		case ".html":
			templates = append(templates, e.Name())
		}
	}
	if len(scripts) != 1 {
		t.Fatalf("assets/ carries %d .js files %v, want exactly 1", len(scripts), scripts)
	}
	src, err := os.ReadFile(filepath.Join("assets", scripts[0]))
	if err != nil {
		t.Fatal(err)
	}
	if string(src) != model.DeckClassScript() {
		t.Errorf("DeckClassScript() is not the bytes of assets/%s", scripts[0])
	}
	if len(templates) != 4 {
		t.Fatalf("assets/ carries %d .html templates, want 4", len(templates))
	}
	for _, name := range templates {
		raw, err := os.ReadFile(filepath.Join("assets", name))
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(raw), deckClassPlaceholder); n != 1 {
			t.Errorf("assets/%s carries the placeholder %d times, want exactly 1", name, n)
		}
		if strings.Contains(string(raw), "<script") {
			t.Errorf("assets/%s carries a script of its own; the deck-class script has one source", name)
		}
	}
}

// The runtime facts the script is written against: it is injected into an
// existing page each time a card is shown, so it must not wait for a load
// event, and it must not locate itself through currentScript, which is not
// guaranteed to survive the injection. Both are text properties of the
// asset, so both are pinned as text.
func TestDeckClassScriptAssumesNoLoadEventAndNoCurrentScript(t *testing.T) {
	// Code only: the asset's comments name these constructs to say why they
	// are avoided, and a comment is not a reliance.
	js := regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(model.DeckClassScript(), "")
	for _, forbidden := range []string{"DOMContentLoaded", "currentScript", "addEventListener(\"load\"", "onload"} {
		if strings.Contains(js, forbidden) {
			t.Errorf("deck-class script relies on %q, which Anki's card injection does not provide", forbidden)
		}
	}
	if strings.Contains(js, "toLowerCase") {
		t.Error("deck-class script uses toLowerCase, which folds non-ASCII letters the Go normalizer maps to hyphens")
	}
}
