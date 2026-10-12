package planner

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/hashing"
	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// SPEC-ANKICARD-004 — the swift option reaches the renderer on BOTH call sites,
// and the composition step's own diagnostic is mapped here.

// swiftEntry is a cloze-style sync target carrying ANKI_SWIFT: t whose body
// carries one arrow line and NO hand-written marker, so composition is what
// satisfies the cloze-marker gate for it (REQ-SW-008.3).
func swiftEntry(body string) protocol.Entry {
	return protocol.Entry{
		Key:        "a.org::0",
		NoteType:   "imoogi-Cloze",
		SourcePath: "a.org",
		Title:      "일본",
		Body:       body,
		Swift:      strPtr("t"),
	}
}

// swiftZeroArrowBodies are acceptance.md's corpus rows whose arrow-line count
// is zero, by row number (rows 7-13, 18-22, 24-30, 33, 36, 39, 41).
var swiftZeroArrowBodies = map[int]string{
	7:  "A :->",
	8:  ":-> B",
	9:  "A :->   ",
	10: "- 도쿄 :-> 일본",
	11: "- Tokyo :: Japan :-> x",
	12: "#+BEGIN_SRC go\nx :-> y\n#+END_SRC",
	13: "#+BEGIN_QUOTE\nx :-> y\n#+END_QUOTE",
	18: "{{c1::도쿄 :-> 일본}}",
	19: "{{c1::도쿄 :-> 일본",
	20: ":-> B :-> C",
	21: "- 수도\n  도쿄 :-> 일본",
	22: "| A :-> B |",
	24: "{{c1::메모\n\n도쿄 :-> 일본",
	25: ": A :-> B",
	26: "[fn:1] A :-> B",
	27: "* A :-> B",
	28: "#+TITLE: A :-> B",
	29: "# A :-> B",
	30: "\\begin{align}\nA :-> B\n\\end{align}",
	33: "[fn:1] 메모\n도쿄 :-> 일본",
	36: "문맥\n:PROPERTIES:\n:X: 도쿄 :-> 일본\n:END:",
	39: "#+BEGIN_SRC sh :exports code\necho\n#+END_SRC\n\n#+RESULTS:\n도쿄 :-> 일본",
	41: "[fn:1] 메모\n#+BEGIN_SRC go\n\n\n#+END_SRC\n도쿄 :-> 일본",
}

// AC-SW-002 — an entry with no arrow line is skipped with the new code, carries
// its existing identifier, issues no write, and leaves its registry record.
func TestSwiftArrowMissing(t *testing.T) {
	if len(swiftZeroArrowBodies) != 23 {
		t.Fatalf("the zero-arrow table holds %d rows, acceptance.md names 23", len(swiftZeroArrowBodies))
	}
	for row, body := range swiftZeroArrowBodies {
		t.Run(fmt.Sprintf("row_%02d", row), func(t *testing.T) {
			reg := newTestRegistry(t)
			client := newFakeClient()
			client.decks["Inbox"] = true
			client.seedNote(77, "imoogi-Cloze", map[string]string{"Text": "x", "Back Extra": ""}, nil, []int{770})
			before := registry.Entry{
				NoteID: 77, SourcePath: "a.org", NoteType: "imoogi-Cloze",
				ResolvedDeck: "Inbox", ContentHash: "the-recorded-hash",
			}
			reg.Put(before)

			entry := swiftEntry(body)
			entry.NoteID = intPtr(77)
			results, errs := Run(context.Background(), protocol.Request{
				Config:  baseConfig(),
				Census:  []protocol.CensusEntry{{NoteID: 77, SourcePath: "a.org"}},
				Entries: []protocol.Entry{entry},
			}, reg, client)

			expectSkippedWithCode(t, results, errs, protocol.CodeSwiftArrowMissing)
			if results[0].NoteID == nil || *results[0].NoteID != 77 {
				t.Errorf("result note_id = %v, want the existing 77", results[0].NoteID)
			}
			if len(client.writeSequence) != 0 {
				t.Errorf("requests issued: %v, want none", client.writeSequence)
			}
			if after, found := reg.Lookup(77); !found || after != before {
				t.Errorf("registry record = %+v, want it unchanged at %+v", after, before)
			}
		})
	}

	t.Run("the_reported_message_names_the_problem", func(t *testing.T) {
		_, errs, _ := runOne(t, swiftEntry("평범한 본문"))
		if len(errs) != 1 || !strings.Contains(errs[0].Message, "arrow") {
			t.Errorf("errs = %+v, want one naming the missing arrow line", errs)
		}
	})

	t.Run("row_12_under_a_multiline_option_is_the_other_code", func(t *testing.T) {
		// The two diagnostics are distinguishable on one body. Rows 10 and 11
		// are lists, which a multiline entry takes as its answers, so they
		// would compose rather than cross-check.
		entry := multilineEntry(strPtr("->"), nil)
		entry.Body = swiftZeroArrowBodies[12]
		results, errs, _ := runOne(t, entry)
		expectSkippedWithCode(t, results, errs, protocol.CodeMultilineAnswerMissing)
	})
}

// AC-SW-008's planner half and AC-SW-010's third clause — an unpaired
// supplementary marker is reported by the check on the split, before
// composition, and wins over a missing arrow line.
func TestSwiftUnpairedExtraMarkerIsReported(t *testing.T) {
	for _, c := range []struct{ name, body string }{
		{"an_orphan_closing_marker_between_two_arrow_lines", "도쿄 :-> 일본\n#+END_EXTRA\n서울 :-> 한국"},
		{"an_orphan_closing_marker_and_no_arrow_line", "평범한 본문\n#+END_EXTRA\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			results, errs, client := runOne(t, swiftEntry(c.body))
			expectSkippedWithCode(t, results, errs, protocol.CodeExtraBlockUnbalanced)
			if len(client.writeSequence) != 0 {
				t.Errorf("requests issued: %v, want none", client.writeSequence)
			}
		})
	}
}

// REQ-SW-009 — the option reader maps the resolved swift value, through the
// SPEC-ANKICARD-002 parser rather than a second one.
func TestSwiftOptionReading(t *testing.T) {
	for _, c := range []struct {
		name  string
		swift *string
		want  bool
	}{
		{"absent", nil, false},
		{"truthy", strPtr("t"), true},
		{"truthy with case and space", strPtr(" T "), true},
		{"falsy", strPtr("nil"), false},
		{"empty", strPtr(""), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := readCardOptions(protocol.Entry{Swift: c.swift}).Swift; got != c.want {
				t.Errorf("Swift = %v, want %v", got, c.want)
			}
		})
	}
}

// REQ-SW-009 — a swift entry is composed on the ordinary path, and the
// migration path renders the same text, so the sync after a migration is a
// no-op rather than an update.
func TestSwiftBothPaths(t *testing.T) {
	t.Run("the_ordinary_path_composes", func(t *testing.T) {
		_, errs, client := runOne(t, swiftEntry("도쿄 :-> 일본"))
		if len(errs) != 0 {
			t.Fatalf("errs = %+v, want none", errs)
		}
		if got := addedText(t, client); !strings.Contains(got, "{{c1::일본}}") || !strings.Contains(got, "swift-arrow") {
			t.Errorf("Text = %q — the option was dropped at the sync call site", got)
		}
	})

	t.Run("the_migration_path_renders_the_same_text", func(t *testing.T) {
		entry := swiftEntry("도쿄 :-> 일본")
		entry.NoteType = "Cloze"
		entry.NoteID = intPtr(55)
		original, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeCloze, entry.Title, entry.Body, readCardOptions(entry))
		if err != nil {
			t.Fatalf("rendering the original: %v", err)
		}
		reg := newTestRegistry(t)
		client := newFakeClient()
		client.decks["Inbox"] = true
		client.seedNote(55, "Cloze", original, nil, []int{550})
		reg.Put(registry.Entry{
			NoteID: 55, SourcePath: "a.org", NoteType: "Cloze",
			ResolvedDeck: "Inbox", ContentHash: hashing.Hash("Cloze", original, "Inbox", nil),
		})
		_, errs := Migrate(context.Background(), protocol.Request{
			Config:  baseConfig(),
			Entries: []protocol.Entry{entry},
		}, reg, client, false)
		if len(errs) != 0 {
			t.Fatalf("migration errs = %+v, want none", errs)
		}
		if migrated := addedText(t, client); migrated != original["Text"] {
			t.Errorf("migrated Text %q differs from the sync path's %q", migrated, original["Text"])
		}
	})
}
