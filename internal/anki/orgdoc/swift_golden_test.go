package orgdoc_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
)

// SPEC-ANKICARD-004 REQ-SW-012.3: the swift cases live in a golden file of
// their own, DISTINCT from render-golden.json and multiline-golden.json, with
// their own recording environment variable — the split SPEC-ANKICARD-003
// REQ-ML-014.3 established. A regenerated golden proves nothing about the
// cases it already held, so the two earlier files are never re-recorded; the
// swift cases get theirs:
//
//	UPDATE_SWIFT_GOLDEN=1 go test ./internal/anki/orgdoc -run TestSwiftGoldenCorpus
var swiftGoldenCorpus = []struct {
	Name  string
	Title string
	Body  string
}{
	{Name: "one_korean_line", Title: "일본", Body: "도쿄 :-> 일본\n"},
	{Name: "consecutive_lines_break_apart", Title: "수도", Body: "도쿄 :-> 일본\n서울 :-> 한국\n"},
	{Name: "leftward", Title: "수도", Body: "도쿄 :<- 일본\n"},
	{Name: "both_ways", Title: "수도", Body: "도쿄 :<-> 일본\n"},
	{Name: "a_second_token_is_text", Title: "Chain", Body: "A :-> B :-> C\n"},
	{Name: "prose_above_and_below", Title: "수도", Body: "문맥 한 줄\n도쿄 :-> 일본\n참고: 성립 조건\n"},
	{Name: "blank_separated_runs", Title: "Pairs", Body: "A :-> B\n\ntext\n\nC :-> D\n"},
	{Name: "hand_written_marker_on_one_side", Title: "Pairs", Body: "{{c1::A}} :-> B\n"},
	{Name: "title_marker_raises_the_base", Title: "{{c3::수도}}", Body: "도쿄 :-> 일본\n"},
	{Name: "latex_braces_are_separated", Title: "Formula", Body: "root :-> \\sqrt{a^{2}}\n"},
	{Name: "an_arrow_free_list_is_context", Title: "수도", Body: "도쿄 :-> 일본\n\n- 참고 사항\n"},
	{Name: "a_list_arrow_beside_a_plain_one", Title: "수도", Body: "도쿄 :-> 일본\n\n- 서울 :-> 한국\n"},
	{Name: "supplementary_block", Title: "수도", Body: "도쿄 :-> 일본\n\n#+BEGIN_EXTRA\n메모\n#+END_EXTRA\n"},
	{Name: "a_source_block_beside_the_arrow", Title: "Shell", Body: "#+BEGIN_SRC sh\nx :-> y\n#+END_SRC\n\nrun :-> echo\n"},
}

const swiftGoldenPath = "testdata/swift-golden.json"

func TestSwiftGoldenCorpus(t *testing.T) {
	produced := make(map[string]map[string]string, len(swiftGoldenCorpus))
	for _, c := range swiftGoldenCorpus {
		fields, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeCloze, c.Title, c.Body, orgdoc.CardOptions{Swift: true})
		if err != nil {
			t.Fatalf("%s: RenderWithOptions returned an error the corpus does not expect: %v", c.Name, err)
		}
		produced[c.Name] = fields
	}

	if os.Getenv("UPDATE_SWIFT_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(swiftGoldenPath), 0o755); err != nil {
			t.Fatalf("creating testdata directory: %v", err)
		}
		encoded, err := json.MarshalIndent(produced, "", "  ")
		if err != nil {
			t.Fatalf("encoding golden: %v", err)
		}
		if err := os.WriteFile(swiftGoldenPath, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("writing golden: %v", err)
		}
		t.Logf("recorded %d golden cases to %s", len(produced), swiftGoldenPath)
		return
	}

	raw, err := os.ReadFile(swiftGoldenPath)
	if err != nil {
		t.Fatalf("reading golden %s: %v (record it with UPDATE_SWIFT_GOLDEN=1)", swiftGoldenPath, err)
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
