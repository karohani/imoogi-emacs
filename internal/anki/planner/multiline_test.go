package planner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/hashing"
	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// SPEC-ANKICARD-003 §3.3 — the card options reach the renderer on BOTH call
// sites, and the composition step's own diagnostic is mapped here.

// multilineEntry is a cloze-style sync target whose body carries an answer
// list and NO hand-written marker, so composition is what satisfies the
// cloze-marker gate for it (REQ-ML-009.3).
func multilineEntry(direction, incremental *string) protocol.Entry {
	return protocol.Entry{
		Key:         "a.org::0",
		NoteType:    "imoogi-Cloze",
		SourcePath:  "a.org",
		Title:       "Capital of Japan",
		Body:        "- Tokyo\n- Osaka\n",
		Direction:   direction,
		Incremental: incremental,
	}
}

// addedText returns the Text field the run sent to AnkiConnect for the one
// entry it processed.
func addedText(t *testing.T, client *fakeClient) string {
	t.Helper()
	if len(client.addCalls) != 1 {
		t.Fatalf("addCalls = %d, want 1", len(client.addCalls))
	}
	return client.addCalls[0].fields["Text"]
}

// AC-ML-002 — an entry with no answer item is skipped with the new code.
func TestMultilineAnswerMissing(t *testing.T) {
	t.Run("a_rightward_with_no_list", func(t *testing.T) {
		entry := multilineEntry(strPtr("->"), nil)
		entry.Body = "A body with prose but no list at all.\n"
		results, errs, client := runOne(t, entry)
		expectSkippedWithCode(t, results, errs, protocol.CodeMultilineAnswerMissing)
		if len(client.addCalls) != 0 {
			t.Errorf("addCalls = %d, want 0 — no note is created or updated for it", len(client.addCalls))
		}
	})

	t.Run("b_leftward_fails_identically", func(t *testing.T) {
		// The title alone does not rescue it: a leftward card on a body with no
		// list hides the only content it has.
		entry := multilineEntry(strPtr("<-"), nil)
		entry.Body = "A body with prose but no list at all.\n"
		results, errs, _ := runOne(t, entry)
		expectSkippedWithCode(t, results, errs, protocol.CodeMultilineAnswerMissing)
	})

	t.Run("c_the_skip_carries_the_existing_identifier", func(t *testing.T) {
		entry := multilineEntry(strPtr("->"), nil)
		entry.Body = "A body with prose but no list at all.\n"
		entry.NoteID = intPtr(77)
		results, errs, _ := runOne(t, entry)
		expectSkippedWithCode(t, results, errs, protocol.CodeMultilineAnswerMissing)
		if results[0].NoteID == nil || *results[0].NoteID != 77 {
			t.Errorf("result note_id = %v, want the existing 77", results[0].NoteID)
		}
	})

	t.Run("d_the_gates_codes_still_win", func(t *testing.T) {
		// The validation gate runs BEFORE composition, so a malformed option
		// value is reported against the drawer line the user wrote rather than
		// as a missing answer list. The new code is fourth by construction
		// rather than by a fourth entry in the gate's fixed order.
		entry := multilineEntry(strPtr("sideways"), nil)
		entry.Body = "A body with prose but no list at all.\n"
		results, errs, _ := runOne(t, entry)
		expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionInvalid)
	})

	t.Run("the_reported_message_names_the_problem", func(t *testing.T) {
		entry := multilineEntry(strPtr("->"), nil)
		entry.Body = "A body with prose but no list at all.\n"
		_, errs, _ := runOne(t, entry)
		if len(errs) != 1 || !strings.Contains(errs[0].Message, "answer") {
			t.Errorf("errs = %+v, want one naming the missing answer list", errs)
		}
	})
}

// AC-ML-009b and AC-ML-009c — a generated marker satisfies the marker gate,
// and the gate still fires for everyone else.
func TestMultilineMarkerGate(t *testing.T) {
	t.Run("b_a_generated_marker_satisfies_the_gate", func(t *testing.T) {
		results, errs, client := runOne(t, multilineEntry(strPtr("->"), nil))
		expectAccepted(t, results, errs)
		if got := addedText(t, client); !strings.Contains(got, "{{c1::Tokyo}}") {
			t.Errorf("Text = %q, want the generated markers", got)
		}
	})

	t.Run("c_the_gate_still_fires_for_everyone_else", func(t *testing.T) {
		// An entry declaring a cloze-style note type, carrying NO multiline
		// option on, and carrying no marker is skipped exactly as it is today.
		entry := multilineEntry(nil, strPtr("nil"))
		entry.Body = "- Tokyo\n- Osaka\n"
		results, errs, _ := runOne(t, entry)
		expectSkippedWithCode(t, results, errs, protocol.CodeClozeMarkerMissing)
	})

	t.Run("a_supplementary_content_never_becomes_an_answer", func(t *testing.T) {
		entry := multilineEntry(strPtr("->"), nil)
		entry.Body = "- Tokyo\n\n#+BEGIN_EXTRA\n- not an answer\n#+END_EXTRA\n"
		_, _, client := runOne(t, entry)
		if len(client.addCalls) != 1 {
			t.Fatalf("addCalls = %d, want 1", len(client.addCalls))
		}
		fields := client.addCalls[0].fields
		if !strings.Contains(fields["Text"], "{{c1::Tokyo}}") {
			t.Errorf("Text = %q, want the answer wrapped", fields["Text"])
		}
		if strings.Contains(fields["Text"], "not an answer") {
			t.Errorf("Text = %q, want the supplementary content out of the question", fields["Text"])
		}
		if !strings.Contains(fields["Back Extra"], "not an answer") ||
			strings.Contains(fields["Back Extra"], "{{c") {
			t.Errorf("Back Extra = %q, want the supplementary content carrying no marker", fields["Back Extra"])
		}
	})

	t.Run("d_a_supplementary_block_between_two_answers_loses_neither", func(t *testing.T) {
		// Removing the block leaves a two-blank-line gap that would otherwise
		// split the authored list in two and drop every answer after it. The
		// collapse of REQ-ML-009.2 closes that gap.
		entry := multilineEntry(strPtr("->"), nil)
		entry.Body = "- Tokyo\n\n#+BEGIN_EXTRA\nnote\n#+END_EXTRA\n\n- Osaka\n"
		_, _, client := runOne(t, entry)
		fields := client.addCalls[0].fields
		for _, want := range []string{"{{c1::Tokyo}}", "{{c1::Osaka}}"} {
			if !strings.Contains(fields["Text"], want) {
				t.Errorf("Text = %q, missing %s", fields["Text"], want)
			}
		}
		if n := strings.Count(fields["Text"], "children-list"); n != 1 {
			t.Errorf("Text carries %d containers, want exactly 1 holding both answers: %q", n, fields["Text"])
		}
		if !strings.Contains(fields["Back Extra"], "note") {
			t.Errorf("Back Extra = %q, want the block's interior", fields["Back Extra"])
		}
	})

	t.Run("e_the_collapse_does_not_merge_across_genuine_content", func(t *testing.T) {
		// A paragraph still separates the lists, and only the first supplies
		// answers. Paired with d, this makes the collapse falsifiable in both
		// directions: one that did nothing fails d, one that swallowed
		// intervening prose fails this.
		entry := multilineEntry(strPtr("->"), nil)
		entry.Body = "- Tokyo\n\nNote.\n\n- Osaka\n"
		_, _, client := runOne(t, entry)
		text := client.addCalls[0].fields["Text"]
		if !strings.Contains(text, "{{c1::Tokyo}}") {
			t.Errorf("Text = %q, want the first list wrapped", text)
		}
		if strings.Contains(text, "{{c1::Osaka}}") {
			t.Errorf("Text = %q, want the later list left as ordinary content", text)
		}
	})

	t.Run("f_the_collapse_does_not_reach_inside_a_block", func(t *testing.T) {
		// An author's code sample reformatted with no diagnostic is a silent
		// content change. Each block form is asserted by its own case: the two
		// take different render paths, so one passing is not evidence for the
		// other.
		for _, block := range []string{"SRC text", "EXAMPLE"} {
			body := "#+BEGIN_" + block + "\nline1\n\n\nline2\n#+END_" + strings.Fields(block)[0] + "\n\n- Tokyo\n"
			entry := multilineEntry(strPtr("->"), nil)
			entry.Body = body
			_, _, client := runOne(t, entry)
			withCollapse := client.addCalls[0].fields["Text"]

			// The same block rendered from a body that never passed through the
			// collapse, taken through the untouched entry point.
			plain, err := orgdoc.Render(orgdoc.NoteTypeCloze, "Capital of Japan",
				strings.Replace(body, "- Tokyo\n", "- {{c1::Tokyo}}\n", 1))
			if err != nil {
				t.Fatalf("%s: Render: %v", block, err)
			}
			if interior(withCollapse) != interior(plain["Text"]) {
				t.Errorf("%s: the block's rendered interior changed\n with collapse: %q\n  without    : %q",
					block, interior(withCollapse), interior(plain["Text"]))
			}
		}
	})

	t.Run("h_an_unterminated_block_makes_the_collapse_decline", func(t *testing.T) {
		// The extent of an unterminated block is undefined, so the collapse
		// declines rather than guessing where it ends: the two lists stay
		// separate and only the first supplies answers. The document still
		// renders — go-org treats the unterminated opening as plain text.
		entry := multilineEntry(strPtr("->"), nil)
		entry.Body = "#+BEGIN_SRC text\n- Tokyo\n\n\n- Osaka\n"
		results, errs, client := runOne(t, entry)
		expectAccepted(t, results, errs)
		text := addedText(t, client)
		if !strings.Contains(text, "{{c1::Tokyo}}") {
			t.Errorf("Text = %q, want the first list wrapped", text)
		}
		if strings.Contains(text, "{{c1::Osaka}}") {
			t.Errorf("Text = %q, want the second list left split and unwrapped", text)
		}
	})

	t.Run("g_two_author_written_lists_merge_and_that_is_pinned", func(t *testing.T) {
		// The cost the collapse takes, asserted rather than merely disclosed:
		// a blank-line run is the only way Org lets an author write two
		// adjacent lists with no prose between them, so a deliberately-authored
		// two-list body now yields one merged answer list. The merged outcome
		// is visible on the card, where the dropped-answer outcome it replaces
		// was silent.
		entry := multilineEntry(strPtr("->"), nil)
		entry.Body = "- Tokyo\n- Osaka\n\n\n- Kyoto\n- Nara\n"
		_, _, client := runOne(t, entry)
		text := addedText(t, client)
		for _, want := range []string{"{{c1::Tokyo}}", "{{c1::Osaka}}", "{{c1::Kyoto}}", "{{c1::Nara}}"} {
			if !strings.Contains(text, want) {
				t.Errorf("Text = %q, missing %s", text, want)
			}
		}
		if n := strings.Count(text, "children-list"); n != 1 {
			t.Errorf("Text carries %d containers, want all four items inside one: %q", n, text)
		}
	})
}

// interior returns what sits between the first <pre> and its closing tag, or
// the whole string when there is none — the block body AC-ML-009f asserts is
// unchanged.
func interior(html string) string {
	open := strings.Index(html, "<pre")
	if open < 0 {
		return html
	}
	close := strings.Index(html[open:], "</pre>")
	if close < 0 {
		return html[open:]
	}
	return html[open : open+close]
}

// AC-ML-010 — both call sites pass the options.
func TestMultilineBothPaths(t *testing.T) {
	t.Run("c_a_multiline_entry_on_the_ordinary_path_is_wrapped", func(t *testing.T) {
		_, _, client := runOne(t, multilineEntry(strPtr("->"), nil))
		if got := addedText(t, client); !strings.Contains(got, "{{c1::Tokyo}}") {
			t.Errorf("Text = %q — the options were dropped at the sync call site", got)
		}
	})

	t.Run("b_the_migration_path_renders_the_same_text_as_the_sync_path", func(t *testing.T) {
		// isClozeStyle accepts stock `Cloze`, so a stock-Cloze heading can be a
		// valid multiline entry. A migration that rendered it WITHOUT its
		// options would carry an unwrapped Text while the ordinary path
		// computes a wrapped one — and the very next synchronization would
		// report an update for content the user never changed.
		entry := multilineEntry(strPtr("->"), nil)
		entry.NoteType = "Cloze"
		entry.NoteID = intPtr(55)

		// The original note is seeded with what the ordinary path renders for
		// it TODAY — wrapped, because the options already reached the renderer
		// there — and the registry with that content's hash, so the migration's
		// ownership-confirmed delete of the original succeeds and the run
		// carries no unrelated diagnostic.
		opts := readCardOptions(entry)
		original, err := orgdoc.RenderWithOptions(orgdoc.NoteTypeCloze, entry.Title, entry.Body, opts)
		if err != nil {
			t.Fatalf("rendering the original: %v", err)
		}
		reg := newTestRegistry(t)
		migrateClient := newFakeClient()
		migrateClient.decks["Inbox"] = true
		migrateClient.seedNote(55, "Cloze", original, nil, []int{550})
		reg.Put(registry.Entry{
			NoteID: 55, SourcePath: "a.org", NoteType: "Cloze",
			ResolvedDeck: "Inbox", ContentHash: hashing.Hash("Cloze", original, "Inbox", nil),
		})

		_, migrateErrs := Migrate(context.Background(), protocol.Request{
			Config:  baseConfig(),
			Entries: []protocol.Entry{entry},
		}, reg, migrateClient, false)
		if len(migrateErrs) != 0 {
			t.Fatalf("migration errs = %+v, want none", migrateErrs)
		}
		migrated := addedText(t, migrateClient)
		if migrated != original["Text"] {
			t.Errorf("the migration path's Text differs from the sync path's: migrated %q, sync %q",
				migrated, original["Text"])
		}
		if !strings.Contains(migrated, "{{c1::Tokyo}}") {
			t.Errorf("migrated Text = %q — the options were dropped at the migration call site", migrated)
		}

		// The ordinary path then plans the same entry against the registry the
		// migration wrote, and reports a no-op rather than an update.
		synced := entry
		synced.NoteType = "imoogi-Cloze"
		for _, rec := range reg.All() {
			synced.NoteID = intPtr(rec.NoteID)
		}
		syncClient := newFakeClient()
		syncClient.decks["Inbox"] = true
		results, errs := Run(context.Background(), protocol.Request{
			Config:  baseConfig(),
			Entries: []protocol.Entry{synced},
		}, reg, syncClient)
		if len(errs) != 0 {
			t.Fatalf("sync errs = %+v, want none", errs)
		}
		if len(results) != 1 || results[0].Action != protocol.ActionSkipped {
			t.Errorf("results = %+v, want one no-op skip rather than an update", results)
		}
		if len(syncClient.updateFieldsCalls) != 0 {
			t.Errorf("the ordinary path issued %d field updates, want none",
				len(syncClient.updateFieldsCalls))
		}
	})
}

// AC-ML-004b — the explicitly falsy direction spelling renders byte-identically
// to an absent one. The two are distinct INPUTS only here, at the boundary
// where `readDirection` resolves them, which is why the criterion is asserted
// on this side rather than inside the renderer.
func TestMultilineDefaultDirection(t *testing.T) {
	_, _, absent := runOne(t, multilineEntry(nil, strPtr("t")))
	_, _, falsy := runOne(t, multilineEntry(strPtr("nil"), strPtr("t")))

	absentText, falsyText := addedText(t, absent), addedText(t, falsy)
	if absentText != falsyText {
		t.Errorf("an explicitly falsy direction does not render identically to an absent one\n falsy : %q\n absent: %q",
			falsyText, absentText)
	}
	// Both compose as `->`: the answers are wrapped and the title is not.
	if !strings.Contains(absentText, "{{c1::Tokyo}}") ||
		strings.Contains(absentText, "{{c3::Capital of Japan}}") {
		t.Errorf("Text = %q, want the `->` shape", absentText)
	}
}

// TestMultilineOptionReading pins the mapping from the wire's option strings
// onto the renderer's own enum — the one place the two vocabularies meet.
func TestMultilineOptionReading(t *testing.T) {
	for _, c := range []struct {
		name        string
		direction   *string
		incremental *string
		want        orgdoc.CardOptions
	}{
		{"no option at all", nil, nil, orgdoc.CardOptions{}},
		{"rightward", strPtr("->"), nil, orgdoc.CardOptions{Direction: orgdoc.DirectionRightward}},
		{"leftward", strPtr("<-"), nil, orgdoc.CardOptions{Direction: orgdoc.DirectionLeftward}},
		{"both", strPtr("<->"), nil, orgdoc.CardOptions{Direction: orgdoc.DirectionBoth}},
		{"surrounding whitespace is trimmed", strPtr("  ->  "), nil, orgdoc.CardOptions{Direction: orgdoc.DirectionRightward}},
		{"the falsy spelling is no direction", strPtr("nil"), nil, orgdoc.CardOptions{}},
		{"incremental alone", nil, strPtr("t"), orgdoc.CardOptions{Incremental: true}},
		{"incremental case does not matter", nil, strPtr("T"), orgdoc.CardOptions{Incremental: true}},
		{"falsy incremental is off", nil, strPtr("nil"), orgdoc.CardOptions{}},
		{
			"both options together",
			strPtr("<->"), strPtr("t"),
			orgdoc.CardOptions{Direction: orgdoc.DirectionBoth, Incremental: true},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			entry := protocol.Entry{Direction: c.direction, Incremental: c.incremental}
			if got := readCardOptions(entry); got != c.want {
				t.Errorf("readCardOptions = %+v, want %+v", got, c.want)
			}
		})
	}
}

// TestRenderErrorMapping pins the three arms both call sites share, so a
// failure cannot be reported as one code on the sync path and another on the
// migration path.
func TestRenderErrorMapping(t *testing.T) {
	for _, c := range []struct {
		name     string
		err      error
		wantCode string
		wantSkip bool
	}{
		{
			"a missing marker is a skip",
			&orgdoc.ClozeMarkerMissingError{},
			protocol.CodeClozeMarkerMissing, true,
		},
		{
			"a missing answer list is a skip",
			&orgdoc.MultilineAnswerMissingError{},
			protocol.CodeMultilineAnswerMissing, true,
		},
		{
			// Anything else is a rendering failure rather than a condition of
			// the entry's content that the user can correct in the buffer.
			"an unrecognized note type is a failure",
			errors.New("orgdoc: unrecognized note type \"Weird\""),
			protocol.CodeOrgParseError, false,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, skip := renderError(c.err)
			if code != c.wantCode || skip != c.wantSkip {
				t.Errorf("renderError = (%q, %v), want (%q, %v)", code, skip, c.wantCode, c.wantSkip)
			}
		})
	}
}

// TestMultilineDanglingExtraMarker pins what a multiline entry renders to when
// the supplementary split leaves an UNPAIRED `#+BEGIN_EXTRA` or `#+END_EXTRA`
// in the remaining body.
//
// The residue is pre-existing: `splitExtraBlocks` produces it, and it does so
// whether or not this SPEC's options are on. What this test fixes is the
// multiline reading of that residue, driven from the ORIGINAL body so the
// whole pipeline — split, collapse, scan, compose, gate, render — is exercised
// rather than the scanner alone.
//
// The decision it pins: a dangling marker is ORDINARY CONTENT. Composition
// gives it no special handling, and it ends the answer list exactly as a
// paragraph does. Two consequences follow, and both are asserted here rather
// than left to be discovered — the answers after the marker are dropped from
// the card, and the marker renders on it as a literal paragraph.
//
// Neither is reported. Reporting would need a diagnostic code this SPEC does
// not define, and no other resolution is available inside its scope: removing
// the marker is the silent content change REQ-ML-009.2 forbids, and having the
// scanner read across it would wrap answers that go-org renders in a second
// list outside the `children-list` container, contradicting AC-ML-011a.
func TestMultilineDanglingExtraMarker(t *testing.T) {
	for _, c := range []struct {
		name        string
		body        string
		wantWrapped []string
		wantDropped []string
		wantLiteral string
	}{
		{
			// The pair is TERMINATED, so no unterminated-block rule fires and
			// the collapse has nothing to decline. Measured: declining and
			// collapsing render identically on every shape in this table, so a
			// decline rule here would change no output at all.
			name:        "a_stray_closing_marker_after_a_valid_pair",
			body:        "- Tokyo\n\n#+BEGIN_EXTRA\nnote\n#+END_EXTRA\n\n#+END_EXTRA\n\n- Osaka\n",
			wantWrapped: []string{"Tokyo"},
			wantDropped: []string{"Osaka"},
			wantLiteral: "#+END_EXTRA",
		},
		{
			name:        "a_lone_closing_marker_with_no_opening",
			body:        "- Tokyo\n\n#+END_EXTRA\n\n- Osaka\n",
			wantWrapped: []string{"Tokyo"},
			wantDropped: []string{"Osaka"},
			wantLiteral: "#+END_EXTRA",
		},
		{
			// Worse than the closing case: go-org swallows the later bullet
			// into the marker's own paragraph, so that answer stops being a
			// list item at all rather than merely going unwrapped.
			name:        "an_unterminated_opening_marker",
			body:        "- Tokyo\n\n#+BEGIN_EXTRA\nnote\n\n- Osaka\n",
			wantWrapped: []string{"Tokyo"},
			wantDropped: []string{"Osaka"},
			wantLiteral: "#+BEGIN_EXTRA",
		},
		{
			// The case that settles REACHABILITY, and the reason this table is
			// not merely about malformed authoring: the markers here are
			// BALANCED — two openings, two closings, properly nested. The
			// split's non-greedy match runs from the first opening to the
			// FIRST closing, so the outer block's tail and its closing marker
			// survive into the remaining body and damage the question side.
			name:        "balanced_but_nested_supplementary_blocks",
			body:        "- Tokyo\n\n#+BEGIN_EXTRA\nouter\n#+BEGIN_EXTRA\ninner\n#+END_EXTRA\ntail\n#+END_EXTRA\n\n- Osaka\n",
			wantWrapped: []string{"Tokyo"},
			wantDropped: []string{"Osaka"},
			wantLiteral: "#+END_EXTRA",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			entry := multilineEntry(strPtr("->"), nil)
			entry.Body = c.body
			results, errs, client := runOne(t, entry)

			// The entry is NOT rejected: an answer list is present, so
			// REQ-ML-002's diagnostic does not fire, and no other code covers
			// this shape.
			expectAccepted(t, results, errs)
			text := addedText(t, client)

			for _, want := range c.wantWrapped {
				if !strings.Contains(text, "{{c1::"+want+"}}") {
					t.Errorf("Text = %q, want %s wrapped", text, want)
				}
			}
			for _, dropped := range c.wantDropped {
				if strings.Contains(text, "{{c1::"+dropped+"}}") {
					t.Errorf("Text = %q, want %s left out of the answers", text, dropped)
				}
			}
			if !strings.Contains(text, c.wantLiteral) {
				t.Errorf("Text = %q, want the dangling marker %s visible as literal text — "+
					"it is what makes this outcome reviewable on the card rather than invisible",
					text, c.wantLiteral)
			}
		})
	}
}

// TestMultilineUnbalancedOpeningLeavesTheQuestionClean is the fourth residue
// shape, and the one that does NOT damage the card's question side: two
// openings with one closing leave the answer list intact and put the stray
// marker in the Back Extra field instead.
func TestMultilineUnbalancedOpeningLeavesTheQuestionClean(t *testing.T) {
	entry := multilineEntry(strPtr("->"), nil)
	entry.Body = "- Tokyo\n#+BEGIN_EXTRA\nfirst\n#+BEGIN_EXTRA\nsecond\n#+END_EXTRA\n- Osaka\n"
	results, errs, client := runOne(t, entry)
	expectAccepted(t, results, errs)

	fields := client.addCalls[0].fields
	for _, want := range []string{"{{c1::Tokyo}}", "{{c1::Osaka}}"} {
		if !strings.Contains(fields["Text"], want) {
			t.Errorf("Text = %q, missing %s — both answers survive on this shape", fields["Text"], want)
		}
	}
	if strings.Contains(fields["Text"], "#+BEGIN_EXTRA") {
		t.Errorf("Text = %q, want no marker on the question side", fields["Text"])
	}
	// The residue lands here instead. Pre-existing and out of this SPEC's
	// scope to repair; pinned so a later card inherits a measurement rather
	// than a suspicion.
	if !strings.Contains(fields["Back Extra"], "#+BEGIN_EXTRA") {
		t.Errorf("Back Extra = %q, want the stray opening marker recorded here", fields["Back Extra"])
	}
}

// TestMultilineSequentialExtraBlocksKeepEveryAnswer is the positive control for
// TestMultilineDanglingExtraMarker, and it bounds how far that test's finding
// reaches.
//
// Two SEQUENTIAL supplementary blocks are the shape `splitExtraBlocks`
// documents as supported ("several blocks concatenate in document order"). They
// leave no stray marker — only a wider blank-line run than a single block does
// — and the collapse of REQ-ML-009.2 closes it. Every answer survives, and both
// block interiors reach the Back Extra field.
//
// Without this case the dangling-marker table would read as a claim that
// supplementary blocks break multiline cards generally. They do not: the
// damage needs unbalanced markers or NESTED blocks, never the documented ones.
func TestMultilineSequentialExtraBlocksKeepEveryAnswer(t *testing.T) {
	entry := multilineEntry(strPtr("->"), nil)
	entry.Body = "- Tokyo\n\n#+BEGIN_EXTRA\none\n#+END_EXTRA\n\n#+BEGIN_EXTRA\ntwo\n#+END_EXTRA\n\n- Osaka\n"
	results, errs, client := runOne(t, entry)
	expectAccepted(t, results, errs)

	fields := client.addCalls[0].fields
	for _, want := range []string{"{{c1::Tokyo}}", "{{c1::Osaka}}"} {
		if !strings.Contains(fields["Text"], want) {
			t.Errorf("Text = %q, missing %s — the collapse must close the gap two blocks leave",
				fields["Text"], want)
		}
	}
	if strings.Contains(fields["Text"], "#+BEGIN_EXTRA") || strings.Contains(fields["Text"], "#+END_EXTRA") {
		t.Errorf("Text = %q, want no marker on the question side", fields["Text"])
	}
	if n := strings.Count(fields["Text"], "children-list"); n != 1 {
		t.Errorf("Text carries %d containers, want both answers inside one: %q", n, fields["Text"])
	}
	for _, want := range []string{"one", "two"} {
		if !strings.Contains(fields["Back Extra"], want) {
			t.Errorf("Back Extra = %q, missing the interior %q", fields["Back Extra"], want)
		}
	}
}
