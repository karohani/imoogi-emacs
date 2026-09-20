package orgdoc_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
)

// SPEC-ANKICARD-002 AC-OPT-012a: the renderer's output is byte-identical
// across this SPEC.
//
// The corpus below is recorded at the SPEC's base commit — the post-t12 tree,
// which already carries the supplementary `Back Extra` field — and compared
// against on every later run. The recording step is deliberately separated
// from the comparison step by an environment variable rather than by an
// "update if missing" fallback: a golden that regenerates itself the moment
// it disappears proves nothing, because the run that broke it would also be
// the run that re-blessed it (plan.md §H anti-pattern 3).
//
// Regenerate deliberately, and only when a change to the rendered output is
// itself the intent:
//
//	UPDATE_ORGDOC_GOLDEN=1 go test ./internal/anki/orgdoc -run TestRenderGoldenCorpus
//
// This file adds no production code to the package. It is the measurement
// half of REQ-OPT-012.1, which is why it lives here rather than in the
// planner: the claim is about `Render` itself, not about a caller's use of it.
var goldenCorpus = []struct {
	Name     string `json:"name"`
	NoteType string `json:"note_type"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}{
	{
		Name:     "basic",
		NoteType: orgdoc.NoteTypeBasic,
		Title:    "What is a monad?",
		Body:     "A monoid in the category of endofunctors.\n\n- associative\n- has a unit\n",
	},
	{
		Name:     "cloze",
		NoteType: orgdoc.NoteTypeCloze,
		Title:    "The capital of {{c1::France}} is {{c2::Paris}}.",
		Body:     "Both are worth remembering.\n",
	},
	{
		Name:     "cloze_with_extra_block",
		NoteType: orgdoc.NoteTypeCloze,
		Title:    "Kanji {{c1::水}} means water.",
		Body: "It is a pictograph of flowing water.\n\n" +
			"#+BEGIN_EXTRA\nOn-reading: スイ\nKun-reading: みず\n#+END_EXTRA\n",
	},
	{
		Name:     "math",
		NoteType: orgdoc.NoteTypeCloze,
		Title:    "Euler's identity is {{c1::$e^{i\\pi} + 1 = 0$}}.",
		Body:     "Derived from $$e^{i\\theta} = \\cos\\theta + i\\sin\\theta$$.\n",
	},
	{
		Name:     "empty_body",
		NoteType: orgdoc.NoteTypeCloze,
		Title:    "A title carrying its own {{c1::marker}} and nothing else.",
		Body:     "",
	},
}

const goldenPath = "testdata/render-golden.json"

func TestRenderGoldenCorpus(t *testing.T) {
	produced := make(map[string]map[string]string, len(goldenCorpus))
	for _, c := range goldenCorpus {
		fields, err := orgdoc.Render(c.NoteType, c.Title, c.Body)
		if err != nil {
			t.Fatalf("%s: Render returned an error the corpus does not expect: %v", c.Name, err)
		}
		produced[c.Name] = fields
	}

	if os.Getenv("UPDATE_ORGDOC_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("creating testdata directory: %v", err)
		}
		encoded, err := json.MarshalIndent(produced, "", "  ")
		if err != nil {
			t.Fatalf("encoding golden: %v", err)
		}
		if err := os.WriteFile(goldenPath, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("writing golden: %v", err)
		}
		t.Logf("recorded %d golden cases to %s", len(produced), goldenPath)
		return
	}

	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden %s: %v (record it with UPDATE_ORGDOC_GOLDEN=1)", goldenPath, err)
	}
	var want map[string]map[string]string
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("decoding golden: %v", err)
	}

	if len(want) != len(produced) {
		t.Fatalf("golden holds %d cases, corpus produced %d", len(want), len(produced))
	}
	for name, wantFields := range want {
		gotFields, ok := produced[name]
		if !ok {
			t.Errorf("golden case %q is absent from the corpus", name)
			continue
		}
		if len(gotFields) != len(wantFields) {
			t.Errorf("%s: field count %d, golden records %d", name, len(gotFields), len(wantFields))
		}
		for field, wantValue := range wantFields {
			gotValue, ok := gotFields[field]
			if !ok {
				t.Errorf("%s: field %q is absent from the rendered output", name, field)
				continue
			}
			if gotValue != wantValue {
				t.Errorf("%s: field %q is not byte-identical to the golden\n got: %q\nwant: %q",
					name, field, gotValue, wantValue)
			}
		}
	}
}

// TestRenderSignatureIsUnchanged is the compile-time half of AC-OPT-012a: the
// renderer's entry point gained no card-option parameter. A parameter added to
// Render makes this assignment fail to compile, which is the assertion — not a
// reviewer reading a signature (plan.md §H anti-pattern 4).
func TestRenderSignatureIsUnchanged(t *testing.T) {
	var _ func(noteType, title, body string) (map[string]string, error) = orgdoc.Render
}
