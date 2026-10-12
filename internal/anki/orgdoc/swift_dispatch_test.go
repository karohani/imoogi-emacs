package orgdoc_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
)

// SPEC-ANKICARD-004 AC-SW-009 — the swift option reaches the renderer through
// the EXISTING option-taking entry point, whose dispatch is extended rather
// than duplicated, and is total.
func TestSwiftDispatch(t *testing.T) {
	t.Run("exactly_two_entry_points", func(t *testing.T) {
		// A third door would reopen the drift REQ-ML-010.1's delegation
		// closed, so the exported surface is enumerated, not assumed.
		sources, err := filepath.Glob("*.go")
		if err != nil {
			t.Fatal(err)
		}
		var exported []string
		for _, path := range sources {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.IsExported() {
					exported = append(exported, fn.Name.Name)
				}
			}
		}
		sort.Strings(exported)
		if strings.Join(exported, ",") != "Render,RenderWithOptions" {
			t.Errorf("exported functions = %v, want exactly Render and RenderWithOptions", exported)
		}
	})

	t.Run("no_option_is_byte_identical_to_the_unwidened_entry_point", func(t *testing.T) {
		for _, c := range goldenCorpus {
			old, oldErr := orgdoc.Render(c.NoteType, c.Title, c.Body)
			fresh, freshErr := orgdoc.RenderWithOptions(c.NoteType, c.Title, c.Body, orgdoc.CardOptions{})
			if (oldErr == nil) != (freshErr == nil) {
				t.Fatalf("%s: error disagreement: old = %v, new = %v", c.Name, oldErr, freshErr)
			}
			for field, want := range old {
				if fresh[field] != want {
					t.Errorf("%s: field %q differs\n new: %q\n old: %q", c.Name, field, fresh[field], want)
				}
			}
		}
	})

	t.Run("all_three_options_falsy_render_as_no_option", func(t *testing.T) {
		const title, body = "A {{c1::cloze}} heading", "도쿄 :-> 일본\n\n- one\n- two\n"
		none, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeCloze, title, body, orgdoc.CardOptions{})
		if err != nil {
			t.Fatal(err)
		}
		falsy, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeCloze, title, body,
			orgdoc.CardOptions{Direction: orgdoc.DirectionNone, Incremental: false, Swift: false})
		if err != nil {
			t.Fatal(err)
		}
		for field, want := range none {
			if falsy[field] != want {
				t.Errorf("field %q differs\n falsy: %q\n  none: %q", field, falsy[field], want)
			}
			for _, absent := range []string{"children-list", "swift-arrow", "<br>"} {
				if strings.Contains(want, absent) {
					t.Errorf("field %q carries %q with no option on: %q", field, absent, want)
				}
			}
		}
	})

	t.Run("swift_beside_a_multiline_option_composes_as_swift", func(t *testing.T) {
		// Unreachable from production — the gate rejects the combination —
		// so it is asserted at the renderer's own boundary.
		fields, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeCloze, "Cities", "도쿄 :-> 일본\n\n- Tokyo\n",
			orgdoc.CardOptions{Direction: orgdoc.DirectionRightward, Incremental: true, Swift: true})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(fields["Text"], "swift-arrow") || strings.Contains(fields["Text"], "children-list") {
			t.Errorf("Text = %q, want the swift card and no answer-list container", fields["Text"])
		}
	})

	t.Run("swift_on_a_non_cloze_type_delegates", func(t *testing.T) {
		got, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeBasic, "Q", "도쿄 :-> 일본", orgdoc.CardOptions{Swift: true})
		if err != nil {
			t.Fatal(err)
		}
		want, _ := orgdoc.Render(orgdoc.NoteTypeBasic, "Q", "도쿄 :-> 일본")
		if len(got) != len(want) || got["Front"] != want["Front"] || got["Back"] != want["Back"] {
			t.Errorf("fields = %q, want the unwidened entry point's %q", got, want)
		}
	})

	t.Run("a_swift_entry_with_no_arrow_line_is_reported", func(t *testing.T) {
		_, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeCloze, "Q {{c1::x}}", "평범한 본문", orgdoc.CardOptions{Swift: true})
		var missing *orgdoc.SwiftArrowMissingError
		if !errors.As(err, &missing) {
			t.Errorf("err = %v, want *SwiftArrowMissingError — a hand-written marker elsewhere does not excuse it", err)
		}
	})
}
