package planner

import (
	"context"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// SPEC-ANKICARD-002 §3.3 — the back end validates a card-bearing entry
// through a single gate, placed before the render call and reached from both
// the ordinary synchronization path and the migration path.

// clozeBody carries a marker, so an entry using it reaches the gate's
// downstream neighbours rather than tripping cloze_marker_missing first.
const clozeBody = "A {{c1::cloze}} body."

// runOne drives one entry through an ordinary sync run and returns what the
// run reported for it, plus the client so a caller can assert on the request
// log. The registry starts empty and the entry carries no identifier, so an
// accepted entry takes the add branch — which is what makes "no request was
// issued" a meaningful assertion for a rejected one.
func runOne(t *testing.T, entry protocol.Entry) ([]protocol.Result, []protocol.Error, *fakeClient) {
	t.Helper()
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	results, errs := Run(context.Background(), protocol.Request{
		Config:  baseConfig(),
		Entries: []protocol.Entry{entry},
	}, reg, client)
	return results, errs, client
}

// optionEntry is a cloze-style sync target carrying the given options.
func optionEntry(direction, incremental, swift *string) protocol.Entry {
	return protocol.Entry{
		Key:         "a.org::0",
		NoteID:      nil,
		NoteType:    "imoogi-Cloze",
		SourcePath:  "a.org",
		Title:       "A heading",
		Body:        clozeBody,
		Direction:   direction,
		Incremental: incremental,
		Swift:       swift,
	}
}

// expectSkippedWithCode asserts the one-diagnostic-per-rejected-entry shape:
// skipped, exactly one error, and that error carrying the expected code.
func expectSkippedWithCode(t *testing.T, results []protocol.Result, errs []protocol.Error, code string) {
	t.Helper()
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Action != protocol.ActionSkipped {
		t.Errorf("action = %q, want %q", results[0].Action, protocol.ActionSkipped)
	}
	if len(errs) != 1 {
		t.Fatalf("errs = %+v, want exactly one", errs)
	}
	if errs[0].Code != code {
		t.Errorf("code = %q, want %q", errs[0].Code, code)
	}
}

func expectAccepted(t *testing.T, results []protocol.Result, errs []protocol.Error) {
	t.Helper()
	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 || results[0].Action != protocol.ActionAdded {
		t.Fatalf("results = %+v, want one added", results)
	}
}

// AC-OPT-007: the cloze-style predicate answers for exactly two names, and is
// DERIVED from the declared-type-to-renderer mapping rather than from a second
// literal list — which is what keeps it from drifting away from the renderer's
// own dispatch the first time a third name appears.
func TestIsClozeStyleAnswersForExactlyTwoNames(t *testing.T) {
	for _, tc := range []struct {
		declared string
		want     bool
	}{
		{"Cloze", true},
		{"imoogi-Cloze", true},
		{"Basic", false},
		{"imoogi-Basic", false},
		{"imoogi-Other", false},
		{"MyCloze", false},
		{"cloze", false},
		{"", false},
	} {
		if got := isClozeStyle(tc.declared); got != tc.want {
			t.Errorf("isClozeStyle(%q) = %v, want %v", tc.declared, got, tc.want)
		}
		// The anti-duplication half: the predicate must agree with the
		// mapping it is derived from, for every name. A second literal list
		// would be free to disagree here.
		if got, viaMapping := isClozeStyle(tc.declared), renderType(tc.declared) == "Cloze"; got != viaMapping {
			t.Errorf("isClozeStyle(%q) = %v but renderType says %v — the predicate has drifted from the mapping",
				tc.declared, got, viaMapping)
		}
	}
}

// AC-OPT-009: the recognized value set, exactly.
func TestCardOptionValueRecognition(t *testing.T) {
	for _, tc := range []struct {
		name       string
		property   string
		value      string
		wantReject bool
	}{
		{"direction right arrow", "direction", "->", false},
		{"direction left arrow", "direction", "<-", false},
		{"direction both arrows", "direction", "<->", false},
		{"direction arrow with surrounding space", "direction", "  ->  ", false},
		{"direction falsy", "direction", "nil", false},
		{"direction falsy upper case", "direction", "NIL", false},
		{"direction double dash", "direction", "-->", true},
		{"direction fat arrow", "direction", "<=>", true},
		{"direction empty string on the wire", "direction", "", true},
		{"incremental truthy", "incremental", "t", false},
		{"incremental truthy upper case", "incremental", "T", false},
		{"incremental falsy", "incremental", "nil", false},
		{"incremental yes", "incremental", "yes", true},
		{"incremental one", "incremental", "1", true},
		{"swift truthy", "swift", "t", false},
		{"swift falsy with surrounding space", "swift", " nil ", false},
		{"swift true", "swift", "true", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var entry protocol.Entry
			switch tc.property {
			case "direction":
				entry = optionEntry(strPtr(tc.value), nil, nil)
			case "incremental":
				entry = optionEntry(nil, strPtr(tc.value), nil)
			case "swift":
				entry = optionEntry(nil, nil, strPtr(tc.value))
			}

			results, errs, client := runOne(t, entry)

			if !tc.wantReject {
				expectAccepted(t, results, errs)
				return
			}

			expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionInvalid)
			// The diagnostic's machine-oriented detail names BOTH the
			// offending property and its actual value, so the user can find
			// the drawer line that carries it.
			if !strings.Contains(errs[0].Message, tc.property) {
				t.Errorf("message %q does not name the property %q", errs[0].Message, tc.property)
			}
			if !strings.Contains(errs[0].Message, tc.value) {
				t.Errorf("message %q does not name the offending value %q", errs[0].Message, tc.value)
			}
			if len(client.addCalls) != 0 || len(client.writeSequence) != 0 {
				t.Errorf("a rejected entry issued requests %v, want none", client.writeSequence)
			}
		})
	}
}

// AC-OPT-010: swift together with a multiline option selects two mutually
// exclusive card kinds, and no precedence between them is defined anywhere.
// A falsy value on either side is OFF and therefore not option-bearing, so it
// does not conflict — which is what keeps the inheritance opt-out usable.
func TestSwiftWithMultilineOptionIsRejected(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		swift, direction, incremental *string
		wantConflict                  bool
	}{
		{"swift with an arrow", strPtr("t"), strPtr("->"), nil, true},
		{"swift with incremental", strPtr("t"), nil, strPtr("t"), true},
		{"swift with both", strPtr("t"), strPtr("<->"), strPtr("t"), true},
		{"swift with falsy incremental", strPtr("t"), nil, strPtr("nil"), false},
		{"swift with falsy direction", strPtr("t"), strPtr("nil"), nil, false},
		{"falsy swift with multiline options", strPtr("nil"), strPtr("->"), strPtr("t"), false},
		{"swift alone", strPtr("t"), nil, nil, false},
		{"multiline options with no swift", nil, strPtr("->"), strPtr("t"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			results, errs, _ := runOne(t, optionEntry(tc.direction, tc.incremental, tc.swift))
			if tc.wantConflict {
				expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionConflict)
				return
			}
			expectAccepted(t, results, errs)
		})
	}
}

// AC-OPT-008: a card option on a non-cloze note type describes a card that
// cannot be produced, and is reported rather than silently discarded.
func TestCardOptionOnNonClozeTypeIsRejected(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		noteType                      string
		direction, incremental, swift *string
		wantReject                    bool
	}{
		{"swift on imoogi-Basic", "imoogi-Basic", nil, nil, strPtr("t"), true},
		{"direction on imoogi-Basic", "imoogi-Basic", strPtr("->"), nil, nil, true},
		{"incremental on imoogi-Basic", "imoogi-Basic", nil, strPtr("t"), nil, true},
		{"swift on stock Basic", "Basic", nil, nil, strPtr("t"), true},
		// The case that makes the inheritance opt-out usable: a basic
		// heading under a file-level ANKI_SWIFT may write the falsy
		// spelling in its own drawer without being told to change its
		// note type for declining the option.
		{"falsy swift on imoogi-Basic", "imoogi-Basic", nil, nil, strPtr("nil"), false},
		{"falsy direction on imoogi-Basic", "imoogi-Basic", strPtr("nil"), nil, nil, false},
		{"no options at all on imoogi-Basic", "imoogi-Basic", nil, nil, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := protocol.Entry{
				Key: "a.org::0", NoteType: tc.noteType, SourcePath: "a.org",
				Title: "A heading", Body: "A plain body.",
				Direction: tc.direction, Incremental: tc.incremental, Swift: tc.swift,
			}
			results, errs, client := runOne(t, entry)

			if !tc.wantReject {
				expectAccepted(t, results, errs)
				return
			}
			expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionNeedsCloze)
			if len(client.writeSequence) != 0 {
				t.Errorf("a rejected entry issued requests %v, want none", client.writeSequence)
			}
		})
	}
}

// AC-OPT-008's identifier clause: a rejected entry carries its EXISTING note
// identifier, and its registry record is left untouched.
func TestRejectedEntryKeepsItsIdentifierAndRegistryRecord(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true
	client.seedNote(77, "imoogi-Basic", map[string]string{"Front": "x", "Back": "y"}, nil, []int{770})
	before := registry.Entry{
		NoteID: 77, SourcePath: "a.org", NoteType: "imoogi-Basic",
		ResolvedDeck: "Inbox", ContentHash: "the-recorded-hash",
	}
	reg.Put(before)

	results, errs := Run(context.Background(), protocol.Request{
		Config: baseConfig(),
		Census: []protocol.CensusEntry{{NoteID: 77, SourcePath: "a.org"}},
		Entries: []protocol.Entry{{
			Key: "a.org::0", NoteID: intPtr(77), NoteType: "imoogi-Basic", SourcePath: "a.org",
			Title: "A heading", Body: "A plain body.", Swift: strPtr("t"),
		}},
	}, reg, client)

	expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionNeedsCloze)
	if results[0].NoteID == nil || *results[0].NoteID != 77 {
		t.Errorf("result note_id = %v, want the entry's existing 77", results[0].NoteID)
	}
	after, found := reg.Lookup(77)
	if !found || after != before {
		t.Errorf("registry record = %+v, want it unchanged at %+v", after, before)
	}
	if len(client.writeSequence) != 0 {
		t.Errorf("requests issued: %v, want none", client.writeSequence)
	}
}

// AC-OPT-011a: exactly one diagnostic per rejected entry, in a fixed order.
// Without a stated order, an entry offending against all three rules would
// have an outcome that depended on evaluation order in the implementation.
func TestOneDiagnosticPerEntryInFixedOrder(t *testing.T) {
	// All three rules fire: a non-cloze type, a conflict, and a malformed
	// value. The malformed value wins — the conflict rule cannot correctly
	// classify a value it does not recognize.
	results, errs, _ := runOne(t, protocol.Entry{
		Key: "a.org::0", NoteType: "imoogi-Basic", SourcePath: "a.org",
		Title: "A heading", Body: "A plain body.",
		Swift: strPtr("t"), Direction: strPtr("bogus"),
	})
	expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionInvalid)

	// Direction corrected; the conflict and the non-cloze type remain.
	results, errs, _ = runOne(t, protocol.Entry{
		Key: "a.org::0", NoteType: "imoogi-Basic", SourcePath: "a.org",
		Title: "A heading", Body: "A plain body.",
		Swift: strPtr("t"), Direction: strPtr("->"),
	})
	expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionConflict)

	// Swift removed; only the non-cloze type remains.
	results, errs, _ = runOne(t, protocol.Entry{
		Key: "a.org::0", NoteType: "imoogi-Basic", SourcePath: "a.org",
		Title: "A heading", Body: "A plain body.",
		Direction: strPtr("->"),
	})
	expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionNeedsCloze)
}

// AC-OPT-011b: the gate runs BEFORE the render. The renderer fails on exactly
// two conditions, and each gives one usable probe: the suppressed code is one
// only the renderer can raise, so seeing the option code instead is evidence
// the gate ran first.
func TestGatePrecedesRendering(t *testing.T) {
	// Probe one: a note type the renderer rejects outright.
	results, errs, _ := runOne(t, protocol.Entry{
		Key: "a.org::0", NoteType: "Widget", SourcePath: "a.org",
		Title: "A heading", Body: "A plain body.", Swift: strPtr("t"),
	})
	expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionNeedsCloze)

	// Probe two: a cloze entry with no marker anywhere, whose option value
	// is also malformed.
	results, errs, _ = runOne(t, protocol.Entry{
		Key: "a.org::0", NoteType: "imoogi-Cloze", SourcePath: "a.org",
		Title: "A heading with no marker", Body: "A plain body.",
		Direction: strPtr("bogus"),
	})
	expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionInvalid)
}

// AC-OPT-013: the marker-missing baseline this SPEC pins for the multiline
// card that comes later. A valid option on a valid type passes the gate and is
// then reported by the EXISTING code — this SPEC introduces no marker the
// options generate. The test is expected to be UPDATED by the multiline card,
// not to keep passing forever.
func TestValidOptionStillReachesClozeMarkerMissing(t *testing.T) {
	results, errs, client := runOne(t, protocol.Entry{
		Key: "a.org::0", NoteType: "imoogi-Cloze", SourcePath: "a.org",
		Title: "A heading with no marker", Body: "A plain body.",
		Direction: strPtr("->"),
	})
	expectSkippedWithCode(t, results, errs, protocol.CodeClozeMarkerMissing)
	if len(client.addCalls) != 0 {
		t.Errorf("addCalls = %d, want 0 — no note is created for it", len(client.addCalls))
	}
}

// --- The migration path ------------------------------------------------

// migrateOne drives one entry through a migration run over a registry that
// already records it on a stock type, which is what makes it a candidate.
func migrateOneEntry(t *testing.T, entry protocol.Entry, recordedType string, dryRun bool) ([]protocol.Result, []protocol.Error, *fakeClient, *registry.Registry) {
	t.Helper()
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true
	noteID := *entry.NoteID
	client.seedNote(noteID, recordedType, map[string]string{"Text": clozeBody, "Extra": ""}, nil, []int{noteID * 10})
	reg.Put(registry.Entry{
		NoteID: noteID, SourcePath: entry.SourcePath, NoteType: recordedType,
		ResolvedDeck: "Inbox", ContentHash: "a-recorded-hash",
	})

	results, errs := Migrate(context.Background(), protocol.Request{
		Config:  baseConfig(),
		Entries: []protocol.Entry{entry},
	}, reg, client, dryRun)
	return results, errs, client, reg
}

// AC-OPT-011c: the migration path shares the gate, from one implementation.
func TestMigrationPathSharesTheGate(t *testing.T) {
	entry := protocol.Entry{
		Key: "a.org::0", NoteID: intPtr(55), NoteType: "Cloze", SourcePath: "a.org",
		Title: "A heading", Body: clozeBody,
		Swift: strPtr("t"), Direction: strPtr("->"),
	}
	results, errs, client, reg := migrateOneEntry(t, entry, "Cloze", false)

	expectSkippedWithCode(t, results, errs, protocol.CodeCardOptionConflict)
	if results[0].NoteID == nil || *results[0].NoteID != 55 {
		t.Errorf("result note_id = %v, want the original 55", results[0].NoteID)
	}
	if len(client.addCalls) != 0 || len(client.deleteCalls) != 0 {
		t.Errorf("migration issued addNote %d / deleteNotes %d for a rejected entry, want none",
			len(client.addCalls), len(client.deleteCalls))
	}
	if rec, found := reg.Lookup(55); !found || rec.NoteType != "Cloze" {
		t.Errorf("registry record = %+v, want it unchanged on the stock type", rec)
	}
}

// AC-OPT-011c's two inherited bypass arms, pinned rather than left unnoticed.
// Both return BEFORE the per-entry pipeline, so no position for the gate
// inside that pipeline can reach them — the bypass is structural, not a
// placement choice. Neither loses coverage: an entry not migrated is still
// processed by the ordinary path, where the gate does run.
func TestMigrationBypassArmsRaiseNoOptionDiagnostic(t *testing.T) {
	conflicting := protocol.Entry{
		Key: "a.org::0", NoteID: intPtr(55), NoteType: "Cloze", SourcePath: "a.org",
		Title: "A heading", Body: clozeBody,
		Swift: strPtr("t"), Direction: strPtr("->"),
	}

	t.Run("dry run", func(t *testing.T) {
		results, errs, _, _ := migrateOneEntry(t, conflicting, "Cloze", true)
		if len(errs) != 0 {
			t.Errorf("errs = %+v, want none — the dry run returns before the per-entry pipeline", errs)
		}
		if len(results) != 1 || results[0].Action != protocol.ActionMigrateCandidate {
			t.Fatalf("results = %+v, want one migrate_candidate", results)
		}
	})

	t.Run("non-candidate", func(t *testing.T) {
		// Recorded on an imoogi-owned type, so the candidate gate rejects
		// it before the pipeline. Its existing handling is unchanged, and
		// no option diagnostic is raised.
		results, errs, _, _ := migrateOneEntry(t, conflicting, "imoogi-Cloze", false)
		for _, e := range errs {
			if strings.HasPrefix(e.Code, "card_option_") {
				t.Errorf("non-candidate raised %q, want no option diagnostic", e.Code)
			}
		}
		// "whatever the existing non-candidate handling already reports for
		// it, unchanged": a declared type differing from the recorded one,
		// which is not its counterpart, is a genuine note-type mismatch.
		if len(errs) != 1 || errs[0].Code != protocol.CodeNoteTypeChangeUnsupported {
			t.Fatalf("errs = %+v, want exactly one %s", errs, protocol.CodeNoteTypeChangeUnsupported)
		}
		if len(results) != 1 || results[0].Action != protocol.ActionSkipped {
			t.Errorf("results = %+v, want one skipped", results)
		}
	})
}

// AC-OPT-011d: the gate reads the DECLARED note type on both paths. A
// candidate's declared type and its migration counterpart always agree on
// cloze-style-ness — a type and its imoogi-owned counterpart are cloze-style
// together or not at all — which is the property the one-signature gate
// relies on.
func TestMigrationGateReadsTheDeclaredType(t *testing.T) {
	entry := protocol.Entry{
		Key: "a.org::0", NoteID: intPtr(56), NoteType: "Cloze", SourcePath: "a.org",
		Title: "A heading", Body: clozeBody,
		Direction: strPtr("->"),
	}
	results, errs, _, _ := migrateOneEntry(t, entry, "Cloze", false)

	for _, e := range errs {
		if strings.HasPrefix(e.Code, "card_option_") {
			t.Errorf("a valid option on a cloze-style candidate raised %q, want none", e.Code)
		}
	}
	if len(results) != 1 || results[0].Action != protocol.ActionAdded {
		t.Fatalf("results = %+v, want one added — the migration proceeded", results)
	}
}
