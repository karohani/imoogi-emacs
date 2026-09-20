package orgdoc_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
)

// SPEC-ANKICARD-003 AC-ML-013b: the multiline byte-identity cases live in a
// golden file DISTINCT from render-golden.json, with its own recording
// environment variable.
//
// Keeping them apart is the point. Adding these cases to the existing corpus
// would require regenerating it, and a regenerated golden proves nothing about
// the cases it already held — the run that would have caught a regression is
// the run that re-blesses it. The existing file therefore stays byte-identical
// to its content at this SPEC's base commit, verified by `git diff --exit-code`
// (AC-ML-013a), and the new cases get their own file:
//
//	UPDATE_MULTILINE_GOLDEN=1 go test ./internal/anki/orgdoc -run TestMultilineGoldenCorpus
var multilineGoldenCorpus = []struct {
	Name        string
	Title       string
	Body        string
	Direction   orgdoc.Direction
	Incremental bool
}{
	{
		Name: "rightward", Title: "Capital of Japan",
		Body: "- Tokyo\n- Osaka\n", Direction: orgdoc.DirectionRightward,
	},
	{
		Name: "leftward", Title: "Capital of Japan",
		Body: "- Tokyo\n- Osaka\n", Direction: orgdoc.DirectionLeftward,
	},
	{
		Name: "both_incremental", Title: "Capital of Japan",
		Body: "- Tokyo\n- Osaka\n", Direction: orgdoc.DirectionBoth, Incremental: true,
	},
	{
		Name: "incremental_alone_defaults_to_rightward", Title: "Capital of Japan",
		Body: "- Tokyo\n- Osaka\n", Incremental: true,
	},
	{
		Name: "description_list", Title: "Cities",
		Body: "- Tokyo :: the capital\n- Osaka :: the second city\n", Direction: orgdoc.DirectionRightward,
	},
	{
		Name: "checkbox_and_nested_child", Title: "Cities",
		Body:      "- [ ] Tokyo\n  - Kanto\n- [X] Osaka\n",
		Direction: orgdoc.DirectionRightward,
	},
	{
		Name: "korean_answers_with_a_status", Title: "대한민국의 수도",
		Body: "- [ ] 서울특별시\n- [X] 부산\n", Direction: orgdoc.DirectionRightward,
	},
	{
		Name: "hand_written_marker_is_left_alone", Title: "Cities",
		Body: "- Tokyo is {{c4::big}}\n- Osaka\n", Direction: orgdoc.DirectionRightward,
	},
	{
		Name: "latex_braces", Title: "Formula",
		Body: "- \\sqrt{a^{2}}\n- f(x}\n", Direction: orgdoc.DirectionRightward,
	},
	{
		Name: "supplementary_block_between_two_answers", Title: "Cities",
		Body:      "- Tokyo\n\n#+BEGIN_EXTRA\nnote\n#+END_EXTRA\n\n- Osaka\n",
		Direction: orgdoc.DirectionRightward,
	},
}

const multilineGoldenPath = "testdata/multiline-golden.json"

func TestMultilineGoldenCorpus(t *testing.T) {
	produced := make(map[string]map[string]string, len(multilineGoldenCorpus))
	for _, c := range multilineGoldenCorpus {
		fields, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeCloze, c.Title, c.Body,
			orgdoc.CardOptions{Direction: c.Direction, Incremental: c.Incremental})
		if err != nil {
			t.Fatalf("%s: RenderWithOptions returned an error the corpus does not expect: %v", c.Name, err)
		}
		produced[c.Name] = fields
	}

	if os.Getenv("UPDATE_MULTILINE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(multilineGoldenPath), 0o755); err != nil {
			t.Fatalf("creating testdata directory: %v", err)
		}
		encoded, err := json.MarshalIndent(produced, "", "  ")
		if err != nil {
			t.Fatalf("encoding golden: %v", err)
		}
		if err := os.WriteFile(multilineGoldenPath, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("writing golden: %v", err)
		}
		t.Logf("recorded %d golden cases to %s", len(produced), multilineGoldenPath)
		return
	}

	raw, err := os.ReadFile(multilineGoldenPath)
	if err != nil {
		t.Fatalf("reading golden %s: %v (record it with UPDATE_MULTILINE_GOLDEN=1)", multilineGoldenPath, err)
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
		for field, wantValue := range wantFields {
			if gotFields[field] != wantValue {
				t.Errorf("%s: field %q is not byte-identical to the golden\n got: %q\nwant: %q",
					name, field, gotFields[field], wantValue)
			}
		}
	}
}

// AC-ML-013e: the NEW entry point with no card option set produces exactly what
// the existing one produces, for every case in the existing byte-identity
// corpus.
//
// This is the criterion that actually guards production. Once both call sites
// move to the option-taking entry point, the existing one has no production
// caller, so a corpus exercising it alone proves nothing about what users get.
// The delegation REQ-ML-010.1 requires is what makes this hold structurally;
// this criterion checks the delegation is there.
func TestMultilineEntryPointDelegatesForNonMultilineEntries(t *testing.T) {
	for _, c := range goldenCorpus {
		t.Run(c.Name, func(t *testing.T) {
			old, oldErr := orgdoc.Render(c.NoteType, c.Title, c.Body)
			fresh, freshErr := orgdoc.RenderWithOptions(c.NoteType, c.Title, c.Body, orgdoc.CardOptions{})
			if (oldErr == nil) != (freshErr == nil) {
				t.Fatalf("error disagreement: old = %v, new = %v", oldErr, freshErr)
			}
			if len(old) != len(fresh) {
				t.Fatalf("field count %d, existing entry point produced %d", len(fresh), len(old))
			}
			for field, want := range old {
				if fresh[field] != want {
					t.Errorf("field %q is not byte-identical\n new: %q\n old: %q", field, fresh[field], want)
				}
			}
		})
	}
}

// AC-ML-013c: an entry whose options all carry the FALSY spelling renders
// byte-identically to the same entry with all three resolved to no value.
//
// Both resolve to the zero CardOptions before the renderer sees them, so this
// is asserted here as the renderer-side property and in the planner as the
// wire-side one, where the two spellings are genuinely different inputs.
func TestMultilineFalsyOptionsAreByteIdenticalToNoOptions(t *testing.T) {
	const title, body = "A {{c1::cloze}} heading", "With a body.\n\n- one\n- two\n"
	none, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeCloze, title, body, orgdoc.CardOptions{})
	if err != nil {
		t.Fatalf("rendering with no options: %v", err)
	}
	falsy, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeCloze, title, body,
		orgdoc.CardOptions{Direction: orgdoc.DirectionNone, Incremental: false})
	if err != nil {
		t.Fatalf("rendering with falsy options: %v", err)
	}
	for field, want := range none {
		if falsy[field] != want {
			t.Errorf("field %q differs\n falsy: %q\n  none: %q", field, falsy[field], want)
		}
	}
	// And neither carries a generated marker or a container: an entry with no
	// multiline option on is not a multiline entry.
	for _, v := range none {
		if strings.Contains(v, "children-list") {
			t.Errorf("a non-multiline entry gained the answer-list container: %q", v)
		}
	}
}
