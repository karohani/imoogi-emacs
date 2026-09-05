package planner

import (
	"context"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/hashing"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// AC-020 — orphan deletion is directory-wide, exclusion-exempt, exemption
// survives a file move, and deletion is confirmed by identifier and
// content. The richest single criterion in the SPEC: 5 notes, 8 Then
// clauses, all asserted here as one test with 8 independent sub-assertions.
//
// Given: registry records
//
//	1001 sourced from a.org, heading since deleted
//	1002 sourced from b.org, heading still exists (not a candidate at all —
//	     it is a sync target this run, i.e. present in entries[])
//	1003 sourced from archive/old.org, heading still exists there (excluded
//	     dir, but census finds it — first exemption term)
//	1004 sourced from a.org, heading deleted, note already absent from the
//	     stub (already-absent skip)
//	1005 recorded as sourced from notes/moved.org (stale path), but the
//	     file was moved intact to archive/moved.org, where the heading
//	     still carries the identifier — census finds it at the NEW
//	     location, so both no-delete AND a source-path refresh are owed.
//
// The stub reports 1001 and 1005 as fully matching the registry's record
// (type/fields/tags), so a wrongly-admitted candidate would be confirmed
// and deleted rather than skipped as absent/unowned — the harness's own
// discipline against a false PASS.
func TestRun_AC020_OrphanDeletion_AllEightClauses(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{NoteID: 1001, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: hashOf(t, "Basic", map[string]string{"Front": "F1", "Back": "B1"}, "Inbox", nil)})
	reg.Put(registry.Entry{NoteID: 1002, SourcePath: "b.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: "irrelevant-1002"})
	reg.Put(registry.Entry{NoteID: 1003, SourcePath: "archive/old.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: "irrelevant-1003"})
	reg.Put(registry.Entry{NoteID: 1004, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: "irrelevant-1004"})
	reg.Put(registry.Entry{NoteID: 1005, SourcePath: "notes/moved.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: hashOf(t, "Basic", map[string]string{"Front": "F5", "Back": "B5"}, "Inbox", nil)})

	client := newFakeClient()
	client.seedNote(1001, "Basic", map[string]string{"Front": "F1", "Back": "B1"}, nil, []int{9001})
	// 1002 not seeded: it must never be queried or deleted regardless.
	// 1003 not seeded: same — it must never enter the candidate set at all.
	// 1004 deliberately NOT seeded: "already absent from the stub's collection".
	client.seedNote(1005, "Basic", map[string]string{"Front": "F5", "Back": "B5"}, nil, []int{9005})

	req := protocol.Request{
		Config: protocol.Config{
			DefaultDeck:     "Inbox",
			AnkiConnectURL:  "http://127.0.0.1:8765",
			RegistryPath:    "/tmp/registry.json",
			SyncRoot:        "/tmp/notes",
			ExcludePatterns: []string{"archive/"},
			ScanComplete:    true,
		},
		// 1002's heading still exists (found by the census below) but is not
		// exercised as a sync target this run — AC-020's own rationale for
		// its non-deletion is "the whole tree was scanned rather than one
		// file" (plan.md D16), a census fact, not an entries[]/no-op fact.
		Entries: nil,
		// The identifier census: whole tree, excluded files included.
		// 1001 and 1004 are absent from the census entirely (their headings
		// were deleted). 1002 is found at b.org. 1003 is found at its
		// existing, still-excluded location. 1005 is found at its NEW
		// location, archive/moved.org, not the stale recorded notes/moved.org.
		Census: []protocol.CensusEntry{
			{NoteID: 1002, SourcePath: "b.org"},
			{NoteID: 1003, SourcePath: "archive/old.org"},
			{NoteID: 1005, SourcePath: "archive/moved.org"},
		},
	}

	results, errs := Run(context.Background(), req, reg, client)

	// Clause 1: a confirmation query is recorded covering exactly {1001, 1004}.
	if len(client.notesInfoCalls) != 1 {
		t.Fatalf("notesInfoCalls = %d, want exactly 1 confirmation query", len(client.notesInfoCalls))
	}
	gotCandidates := intSet(client.notesInfoCalls[0])
	wantCandidates := intSet([]int{1001, 1004})
	if !gotCandidates.equal(wantCandidates) {
		t.Errorf("confirmation query candidates = %v, want {1001, 1004}", client.notesInfoCalls[0])
	}

	// Clause 2: exactly one delete request is recorded, naming 1001.
	if len(client.deleteCalls) != 1 {
		t.Fatalf("deleteCalls = %v, want exactly one call", client.deleteCalls)
	}
	if len(client.deleteCalls[0]) != 1 || client.deleteCalls[0][0] != 1001 {
		t.Errorf("delete request = %v, want [1001]", client.deleteCalls[0])
	}
	// Clause 1 (order): confirmation precedes every delete request.
	// (only one delete call here; ordering is enforced by construction —
	// confirmAndDelete always confirms before it deletes.)

	// Clause 3: no delete request for 1002 (whole-tree scan, not per-file).
	for _, batch := range client.deleteCalls {
		for _, id := range batch {
			if id == 1002 {
				t.Error("1002 was deleted; its heading still exists")
			}
		}
	}

	// Clause 4: 1003 never enters the candidate set (found in the excluded
	// but still-read archive/old.org).
	if gotCandidates.has(1003) {
		t.Error("1003 entered the candidate set; the census should have exempted it")
	}

	// Clause 5: 1005 never enters the candidate set (found at its new
	// location), and no add/update request touches it (excluded from
	// sync-target eligibility).
	if gotCandidates.has(1005) {
		t.Error("1005 entered the candidate set; the census should have found it at its new location")
	}
	for _, c := range client.updateFieldsCalls {
		if c.noteID == 1005 {
			t.Error("1005 received an update request; it is excluded from sync-target eligibility")
		}
	}
	if len(client.addCalls) != 0 {
		t.Errorf("addCalls = %d, want 0 — no entry in this run needs an add", len(client.addCalls))
	}

	// Clause 6: the registry now records 1005's refreshed source path.
	e1005, ok := reg.Lookup(1005)
	if !ok {
		t.Fatal("1005's registry entry was dropped; it must survive with a refreshed path")
	}
	if e1005.SourcePath != "archive/moved.org" {
		t.Errorf("1005 source_path = %q, want archive/moved.org (refreshed from the census)", e1005.SourcePath)
	}

	// Clause 7: no delete request for 1004, and no error reported for it —
	// confirmation showed it already absent.
	for _, batch := range client.deleteCalls {
		for _, id := range batch {
			if id == 1004 {
				t.Error("1004 was deleted; it must be skipped silently as already absent")
			}
		}
	}
	for _, e := range errs {
		if e.Message != "" && containsSubstr(e.Message, "1004") {
			t.Errorf("an already-absent candidate must not be reported as an error: %+v", e)
		}
	}

	// Clause 8: no delete request in the run is scoped by anything other
	// than an explicit identifier list — enforced structurally by the fake
	// client's DeleteNotes signature (noteIDs []int, no query/tag param);
	// re-assert every recorded batch is non-empty and identifier-only.
	for _, batch := range client.deleteCalls {
		if len(batch) == 0 {
			t.Error("an empty delete batch was recorded")
		}
	}

	// Sanity: 1001 was actually dropped from the registry post-delete.
	if _, ok := reg.Lookup(1001); ok {
		t.Error("1001's registry entry survived a confirmed delete")
	}
	// Sanity: exactly one deleted result reported, key nil, matching note id.
	var deletedResults []protocol.Result
	for _, r := range results {
		if r.Action == protocol.ActionDeleted {
			deletedResults = append(deletedResults, r)
		}
	}
	if len(deletedResults) != 1 {
		t.Fatalf("deleted results = %+v, want exactly one", deletedResults)
	}
	if deletedResults[0].Key != nil {
		t.Errorf("deleted result key = %v, want nil (no corresponding request entry)", *deletedResults[0].Key)
	}
	if deletedResults[0].NoteID == nil || *deletedResults[0].NoteID != 1001 {
		t.Errorf("deleted result note_id = %v, want 1001", deletedResults[0].NoteID)
	}
}

// AC-021: an incomplete scan suppresses every deletion, including for a
// candidate that would otherwise have confirmed; readable files still
// proceed through their own add/update path.
func TestRun_AC021_IncompleteScan_SuppressesAllDeletion(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{NoteID: 1001, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: hashOf(t, "Basic", map[string]string{"Front": "F1", "Back": "B1"}, "Inbox", nil)})
	client := newFakeClient()
	client.seedNote(1001, "Basic", map[string]string{"Front": "F1", "Back": "B1"}, nil, []int{9001})

	req := protocol.Request{
		Config: protocol.Config{
			DefaultDeck:     "Inbox",
			AnkiConnectURL:  "http://127.0.0.1:8765",
			RegistryPath:    "/tmp/registry.json",
			SyncRoot:        "/tmp/notes",
			ExcludePatterns: []string{"archive/"},
			ScanComplete:    false, // c.org and archive/locked.org could not be read
		},
		Entries: []protocol.Entry{
			{Key: "readable.org::0", NoteID: nil, NoteType: "Basic", SourcePath: "readable.org", Title: "still processed", Body: "yes"},
		},
		Census: []protocol.CensusEntry{},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(client.deleteCalls) != 0 {
		t.Errorf("deleteCalls = %v, want none — an incomplete scan suppresses deletion for candidate 1001 too", client.deleteCalls)
	}
	if len(client.notesInfoCalls) != 0 {
		t.Errorf("notesInfoCalls = %v, want none — no confirmation query on an incomplete scan", client.notesInfoCalls)
	}
	foundSuppressed := false
	var suppressedMessage string
	for _, e := range errs {
		if e.Code == protocol.CodeDeleteSuppressed {
			foundSuppressed = true
			suppressedMessage = e.Message
		}
	}
	if !foundSuppressed {
		t.Errorf("errs = %+v, want a delete_suppressed entry", errs)
	}
	// AC-021's suppression reason is an incomplete scan — distinct from
	// AC-022's confirmation-query-failure reason, even though both share
	// CodeDeleteSuppressed. The message is what makes the two reasons
	// independently distinguishable in the test suite.
	if !containsSubstr(suppressedMessage, "incomplete") {
		t.Errorf("suppressed message = %q, want it to reference an incomplete scan", suppressedMessage)
	}
	// The readable file's add still proceeds.
	foundAdded := false
	for _, r := range results {
		if r.Action == protocol.ActionAdded {
			foundAdded = true
		}
	}
	if !foundAdded {
		t.Errorf("results = %+v, want the readable entry's add to still succeed", results)
	}
	if _, ok := reg.Lookup(1001); !ok {
		t.Error("1001's registry entry was dropped despite deletion being suppressed")
	}
}

// AC-022: an unanswered confirmation (transport error) suppresses every
// deletion, and prior successful add/update requests in the same run are
// not reverted.
func TestRun_AC022_UnansweredConfirmation_SuppressesAllDeletion(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{NoteID: 1001, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: hashOf(t, "Basic", map[string]string{"Front": "F1", "Back": "B1"}, "Inbox", nil)})
	client := newFakeClient()
	client.seedNote(1001, "Basic", map[string]string{"Front": "F1", "Back": "B1"}, nil, []int{9001})
	client.notesInfoErr = errFakeTransport

	req := protocol.Request{
		Config: protocol.Config{
			DefaultDeck:     "Inbox",
			AnkiConnectURL:  "http://127.0.0.1:8765",
			RegistryPath:    "/tmp/registry.json",
			SyncRoot:        "/tmp/notes",
			ExcludePatterns: nil,
			ScanComplete:    true,
		},
		Entries: []protocol.Entry{
			{Key: "b.org::0", NoteID: nil, NoteType: "Basic", SourcePath: "b.org", Title: "already succeeded", Body: "yes"},
		},
		Census: []protocol.CensusEntry{}, // 1001 absent from census -> candidate
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(client.deleteCalls) != 0 {
		t.Errorf("deleteCalls = %v, want none", client.deleteCalls)
	}
	foundSuppressed := false
	var suppressedMessage string
	for _, e := range errs {
		if e.Code == protocol.CodeDeleteSuppressed {
			foundSuppressed = true
			suppressedMessage = e.Message
		}
	}
	if !foundSuppressed {
		t.Errorf("errs = %+v, want a delete_suppressed entry", errs)
	}
	// AC-022's suppression reason is a failed confirmation query — distinct
	// from AC-021's incomplete-scan reason, even though both share
	// CodeDeleteSuppressed. The message is what makes the two reasons
	// independently distinguishable in the test suite.
	if !containsSubstr(suppressedMessage, "confirmation query failed") {
		t.Errorf("suppressed message = %q, want it to reference the failed confirmation query", suppressedMessage)
	}
	// The add that already succeeded this run is not reverted.
	if len(client.addCalls) != 1 {
		t.Errorf("addCalls = %d, want 1 — the prior success must not be reverted", len(client.addCalls))
	}
	foundAdded := false
	for _, r := range results {
		if r.Action == protocol.ActionAdded {
			foundAdded = true
		}
	}
	if !foundAdded {
		t.Errorf("results = %+v, want the add result preserved", results)
	}
	if _, ok := reg.Lookup(1001); !ok {
		t.Error("1001 was dropped despite the confirmation query failing")
	}
}

// AC-023: a candidate whose content no longer matches (an Anki-side edit)
// is skipped, not deleted, and its registry entry is retained.
func TestRun_AC023_ContentMismatch_SkippedNotDeleted(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{NoteID: 1001, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: hashOf(t, "Basic", map[string]string{"Front": "F1", "Back": "B1"}, "Inbox", nil)})
	reg.Put(registry.Entry{NoteID: 1002, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: hashOf(t, "Basic", map[string]string{"Front": "F2", "Back": "B2"}, "Inbox", nil)})
	client := newFakeClient()
	client.seedNote(1001, "Basic", map[string]string{"Front": "F1", "Back": "B1"}, nil, []int{9001})
	client.seedNote(1002, "Basic", map[string]string{"Front": "F2", "Back": "EDITED ON ANKI SIDE"}, nil, []int{9002})

	req := protocol.Request{
		Config: protocol.Config{
			DefaultDeck:     "Inbox",
			AnkiConnectURL:  "http://127.0.0.1:8765",
			RegistryPath:    "/tmp/registry.json",
			SyncRoot:        "/tmp/notes",
			ExcludePatterns: nil,
			ScanComplete:    true,
		},
		Entries: nil,
		Census:  []protocol.CensusEntry{}, // both headings deleted; neither in census
	}

	_, errs := Run(context.Background(), req, reg, client)

	if len(client.deleteCalls) != 1 || len(client.deleteCalls[0]) != 1 || client.deleteCalls[0][0] != 1001 {
		t.Fatalf("deleteCalls = %v, want exactly [1001]", client.deleteCalls)
	}
	foundUnowned := false
	for _, e := range errs {
		if e.Code == protocol.CodeDeleteCandidateUnowned {
			foundUnowned = true
		}
	}
	if !foundUnowned {
		t.Errorf("errs = %+v, want a delete_candidate_unowned entry for 1002", errs)
	}
	if _, ok := reg.Lookup(1002); !ok {
		t.Error("1002's registry entry was dropped; it must be retained on an unowned skip")
	}
	if _, ok := reg.Lookup(1001); ok {
		t.Error("1001 should have been deleted and dropped")
	}
}

// AC-020's Given note: a candidate reported as existing but with a
// differing NOTE TYPE (not just field content) must also route to the
// unowned skip, per REQ-014's "note type and rendered field content both
// match" conjunction and acceptance.md §D.7's edge case.
func TestRun_ConfirmationNoteTypeMismatch_SkippedAsUnowned(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{NoteID: 1001, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: hashOf(t, "Basic", map[string]string{"Front": "F1", "Back": "B1"}, "Inbox", nil)})
	client := newFakeClient()
	client.seedNote(1001, "Cloze", map[string]string{"Text": "{{c1::x}}"}, nil, []int{9001})

	req := protocol.Request{
		Config: protocol.Config{
			DefaultDeck:  "Inbox",
			SyncRoot:     "/tmp/notes",
			ScanComplete: true,
		},
		Census: []protocol.CensusEntry{},
	}

	_, errs := Run(context.Background(), req, reg, client)

	if len(client.deleteCalls) != 0 {
		t.Errorf("deleteCalls = %v, want none — note type mismatch is not ownership", client.deleteCalls)
	}
	found := false
	for _, e := range errs {
		if e.Code == protocol.CodeDeleteCandidateUnowned {
			found = true
		}
	}
	if !found {
		t.Errorf("errs = %+v, want delete_candidate_unowned", errs)
	}
	if _, ok := reg.Lookup(1001); !ok {
		t.Error("registry entry must be retained on a note-type-mismatch skip")
	}
}

// §D.7 edge case: no candidates at all -> zero confirmation queries (the
// steady-state cost of an unchanged tree must not include an idle
// notesInfo round trip).
func TestRun_NoCandidates_IssuesNoConfirmationQuery(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()

	req := protocol.Request{
		Config: protocol.Config{DefaultDeck: "Inbox", SyncRoot: "/tmp/notes", ScanComplete: true},
	}
	Run(context.Background(), req, reg, client)

	if len(client.notesInfoCalls) != 0 {
		t.Errorf("notesInfoCalls = %d, want 0 — an empty candidate set issues no query", len(client.notesInfoCalls))
	}
}

// TestRun_FreshlyAddedNote_NeverConsideredOrphanCandidateSameRun is a
// dedicated regression test for the pre-run identifier snapshot (planner.go
// preRunIDs, ~line 34-48): a note added THIS run is assigned its identifier
// by AnkiConnect during this very call, so it can never have appeared in
// this run's census — without restricting candidate eligibility to
// identifiers that predate the run, a brand-new note would misread as an
// orphan and be deleted in the same run that created it.
//
// This guarantee was previously only incidentally exercised by the add-
// branch test (which asserts nothing about deletion) and the AC-020
// reconcile/candidate test (which seeds a pre-existing registry, never a
// same-run add). This test isolates the guarantee directly: add a brand-new
// note with no prior registry entry and no note_id on the entry, then assert
// the orphan-confirmation query was never issued for it.
func TestRun_FreshlyAddedNote_NeverConsideredOrphanCandidateSameRun(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()

	req := protocol.Request{
		Config: protocol.Config{DefaultDeck: "Inbox", SyncRoot: "/tmp/notes", ScanComplete: true},
		Entries: []protocol.Entry{
			// No NoteID at all -- the add branch (REQ-009). No prior registry
			// entry exists for it either.
			{Key: "fresh.org::0", NoteID: nil, NoteType: "Basic", SourcePath: "fresh.org", Title: "Brand new", Body: "content"},
		},
		// The census was captured before this run's scan, so it cannot
		// possibly report an identifier AnkiConnect has not assigned yet.
		Census: []protocol.CensusEntry{},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 || results[0].Action != protocol.ActionAdded || results[0].NoteID == nil {
		t.Fatalf("results = %+v, want exactly one added note with a note id", results)
	}
	newID := *results[0].NoteID

	if len(client.notesInfoCalls) != 0 {
		t.Errorf("notesInfoCalls = %v, want none — a freshly added note must never be treated as an orphan candidate in the same run", client.notesInfoCalls)
	}
	for _, batch := range client.notesInfoCalls {
		for _, id := range batch {
			if id == newID {
				t.Errorf("the freshly assigned note id %d was included in an orphan-confirmation query this run", newID)
			}
		}
	}
	if len(client.deleteCalls) != 0 {
		t.Errorf("deleteCalls = %v, want none — nothing was deleted", client.deleteCalls)
	}
	if _, ok := reg.Lookup(newID); !ok {
		t.Errorf("registry entry for the freshly added note %d was dropped", newID)
	}
}

// §D.7 edge case: the same identifier present at two census locations, one
// of them excluded — exempt from candidacy regardless, and the registry
// records the FIRST occurrence in scan order as the refreshed path.
func TestRun_CensusDuplicateIdentifier_ExemptAndFirstOccurrenceWins(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{NoteID: 42, SourcePath: "stale.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: "irrelevant"})
	client := newFakeClient()

	req := protocol.Request{
		Config: protocol.Config{
			DefaultDeck:     "Inbox",
			SyncRoot:        "/tmp/notes",
			ExcludePatterns: []string{"archive/"},
			ScanComplete:    true,
		},
		Census: []protocol.CensusEntry{
			{NoteID: 42, SourcePath: "first.org"},          // first occurrence in scan order
			{NoteID: 42, SourcePath: "archive/second.org"}, // duplicate, excluded
		},
	}

	Run(context.Background(), req, reg, client)

	if len(client.notesInfoCalls) != 0 {
		t.Errorf("notesInfoCalls = %d, want 0 — 42 is exempt via the census regardless of duplication", len(client.notesInfoCalls))
	}
	e, ok := reg.Lookup(42)
	if !ok {
		t.Fatal("42's registry entry was dropped; it must survive as a duplicate-exempt identifier")
	}
	if e.SourcePath != "first.org" {
		t.Errorf("source_path = %q, want first.org (first occurrence in scan order)", e.SourcePath)
	}
}

// §D.7 edge case: a heading keeps ANKI_NOTE_ID but loses ANKI_NOTE_TYPE. It
// is no longer a sync target (absent from entries[]) but the census still
// finds its identifier, so the note must survive — no add, no update, no
// delete.
func TestRun_UnmarkedHeading_KeepsIdentifierLosesType_NoteSurvives(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{NoteID: 77, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: "irrelevant"})
	client := newFakeClient()
	client.seedNote(77, "Basic", map[string]string{"Front": "F", "Back": "B"}, nil, []int{9077})

	req := protocol.Request{
		Config: protocol.Config{DefaultDeck: "Inbox", SyncRoot: "/tmp/notes", ScanComplete: true},
		// no entries[] at all -- the heading is no longer a sync target
		Census: []protocol.CensusEntry{{NoteID: 77, SourcePath: "a.org"}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(results) != 0 {
		t.Errorf("results = %+v, want none — the heading is not a sync target and not a delete candidate", results)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %+v, want none", errs)
	}
	if len(client.deleteCalls) != 0 || len(client.addCalls) != 0 || len(client.updateFieldsCalls) != 0 {
		t.Error("an unmarked heading must produce no add, update, or delete request")
	}
	if _, ok := reg.Lookup(77); !ok {
		t.Error("77's registry entry must survive; the census still finds its identifier")
	}
}

// §D.7 edge case: a registry identifier whose heading was deleted from
// inside an already-excluded file. The census reports it nowhere, so the
// first exemption term does not apply; the recorded source path still
// matches the exclusion pattern, so the second (residue) term does, and
// nothing is deleted.
func TestRun_HeadingDeletedInsideExcludedSubtree_ResidueTermExempts(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{NoteID: 55, SourcePath: "archive/gone.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: "irrelevant"})
	client := newFakeClient()
	client.seedNote(55, "Basic", map[string]string{"Front": "F", "Back": "B"}, nil, []int{9055})

	req := protocol.Request{
		Config: protocol.Config{
			DefaultDeck:     "Inbox",
			SyncRoot:        "/tmp/notes",
			ExcludePatterns: []string{"archive/"},
			ScanComplete:    true,
		},
		Census: []protocol.CensusEntry{}, // 55 found nowhere at all
	}

	Run(context.Background(), req, reg, client)

	if len(client.notesInfoCalls) != 0 {
		t.Errorf("notesInfoCalls = %d, want 0 — 55 is exempted by the residue term, never queried", len(client.notesInfoCalls))
	}
	if len(client.deleteCalls) != 0 {
		t.Errorf("deleteCalls = %v, want none", client.deleteCalls)
	}
	if _, ok := reg.Lookup(55); !ok {
		t.Error("55's registry entry must survive under the residue exemption")
	}
}

// A registry identifier whose heading is genuinely gone from a
// non-excluded location, with no residue exemption available, IS a
// candidate and — if confirmed — gets deleted. This is the contrast case
// to the residue-term test above: same shape, but the recorded path does
// not match any exclusion pattern.
func TestRun_HeadingDeletedFromNonExcludedLocation_NoResidueExemption_IsDeleted(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{NoteID: 88, SourcePath: "notes/gone.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: hashOf(t, "Basic", map[string]string{"Front": "F", "Back": "B"}, "Inbox", nil)})
	client := newFakeClient()
	client.seedNote(88, "Basic", map[string]string{"Front": "F", "Back": "B"}, nil, []int{9088})

	req := protocol.Request{
		Config: protocol.Config{
			DefaultDeck:     "Inbox",
			SyncRoot:        "/tmp/notes",
			ExcludePatterns: []string{"archive/"},
			ScanComplete:    true,
		},
		Census: []protocol.CensusEntry{},
	}

	Run(context.Background(), req, reg, client)

	if len(client.deleteCalls) != 1 || len(client.deleteCalls[0]) != 1 || client.deleteCalls[0][0] != 88 {
		t.Errorf("deleteCalls = %v, want exactly [88]", client.deleteCalls)
	}
	if _, ok := reg.Lookup(88); ok {
		t.Error("88 must be dropped from the registry after a confirmed delete")
	}
}

// ---- test helpers ----

func hashOf(t *testing.T, noteType string, fields map[string]string, deck string, tags []string) string {
	t.Helper()
	return hashing.Hash(noteType, fields, deck, tags)
}

type intSetT map[int]bool

func intSet(ids []int) intSetT {
	s := make(intSetT, len(ids))
	for _, id := range ids {
		s[id] = true
	}
	return s
}

func (s intSetT) has(id int) bool { return s[id] }

func (s intSetT) equal(other intSetT) bool {
	if len(s) != len(other) {
		return false
	}
	for k := range s {
		if !other[k] {
			return false
		}
	}
	return true
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && (func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}
