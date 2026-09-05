package planner

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/hashing"
	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// renderAndHash mirrors exactly what processEntry computes internally, so
// a test can construct a registry's PRIOR content_hash that faithfully
// represents "this was the old state" rather than an opaque sentinel.
func renderAndHash(t *testing.T, noteType, title, body, deck string, tags []string) string {
	t.Helper()
	fields, err := orgdoc.Render(noteType, title, body)
	if err != nil {
		t.Fatalf("renderAndHash: %v", err)
	}
	return hashing.Hash(noteType, fields, deck, tags)
}

func newTestRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	reg, err := registry.Load(filepath.Join(t.TempDir(), "registry.json"))
	if err != nil {
		t.Fatalf("newTestRegistry: %v", err)
	}
	return reg
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func baseConfig() protocol.Config {
	return protocol.Config{
		DefaultDeck:     "Inbox",
		AnkiConnectURL:  "http://127.0.0.1:8765",
		RegistryPath:    "/tmp/registry.json",
		SyncRoot:        "/tmp/notes",
		ExcludePatterns: nil,
		ScanComplete:    true,
	}
}

// ---- REQ-009: add branch ----

func TestRun_AddBranch_NoNoteID_CreatesNoteAndRegistersIt(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{
				Key:        "a.org::0",
				NoteID:     nil,
				NoteType:   "Basic",
				SourcePath: "a.org",
				Deck:       nil,
				Tags:       nil,
				Title:      "Capital of France",
				Body:       "Paris.",
			},
		},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Action != protocol.ActionAdded {
		t.Errorf("action = %q, want %q", results[0].Action, protocol.ActionAdded)
	}
	if len(client.addCalls) != 1 {
		t.Fatalf("addCalls = %d, want 1", len(client.addCalls))
	}
	if client.addCalls[0].deck != "Inbox" {
		t.Errorf("add deck = %q, want fallback default %q", client.addCalls[0].deck, "Inbox")
	}
	if client.addCalls[0].modelName != "Basic" {
		t.Errorf("add modelName = %q, want Basic", client.addCalls[0].modelName)
	}
	if got := client.addCalls[0].fields["Front"]; got == "" {
		t.Errorf("add Front field empty, want rendered title")
	}

	e, ok := reg.Lookup(*results[0].NoteID)
	if !ok {
		t.Fatalf("registry has no entry for the newly added note")
	}
	if e.SourcePath != "a.org" {
		t.Errorf("registry source_path = %q, want a.org", e.SourcePath)
	}
	if e.ResolvedDeck != "Inbox" {
		t.Errorf("registry resolved_deck = %q, want Inbox", e.ResolvedDeck)
	}
}

// AC-009: the stub's collection already contains the default deck, so no
// deck-create request must fire.
func TestRun_AddBranch_DeckAlreadyExists_NoCreateDeckRequest(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: nil, NoteType: "Basic", SourcePath: "a.org", Title: "T", Body: "B"},
		},
	}

	_, errs := Run(context.Background(), req, reg, client)
	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(client.createDeckCalls) != 0 {
		t.Errorf("createDeckCalls = %v, want none — the deck already existed", client.createDeckCalls)
	}
}

// AC-013: two decks, neither existing, both get created.
func TestRun_AddBranch_MissingDeck_IsCreated(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: nil, NoteType: "Basic", SourcePath: "a.org", Deck: strPtr("Geography::Europe"), Title: "T", Body: "B"},
		},
	}

	_, errs := Run(context.Background(), req, reg, client)
	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(client.createDeckCalls) != 1 || client.createDeckCalls[0] != "Geography::Europe" {
		t.Errorf("createDeckCalls = %v, want [Geography::Europe]", client.createDeckCalls)
	}
}

// AC-014: an entry with no ANKI_TAGS anywhere in its chain carries an empty
// tag set, not a null/absent one.
func TestRun_AddBranch_NoTags_SendsEmptySet(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: nil, NoteType: "Basic", SourcePath: "a.org", Tags: nil, Title: "T", Body: "B"},
		},
	}
	Run(context.Background(), req, reg, client)

	// The wire-level nil→[] conversion (REQ-006's "empty set, not an absent
	// field") is ankiconnect.Client's own job, already regression-covered in
	// M3a (TestAddNote_MalformedResultIsProtocolError's siblings). The
	// planner's obligation is only to pass through an empty tag set, not to
	// re-normalize nil vs non-nil-empty at the Go-slice level.
	if len(client.addCalls[0].tags) != 0 {
		t.Errorf("add tags = %v, want empty", client.addCalls[0].tags)
	}
}

// REQ-019 / AC-025: a Cloze entry with no marker is skipped, not added, and
// the run continues to the next entry.
func TestRun_ClozeMissingMarker_SkipsAndContinues(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: nil, NoteType: "Cloze", SourcePath: "a.org", Title: "no markers here", Body: "plain text"},
			{Key: "a.org::1", NoteID: nil, NoteType: "Basic", SourcePath: "a.org", Title: "Valid", Body: "Body"},
		},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Action != protocol.ActionSkipped {
		t.Errorf("results[0].action = %q, want skipped", results[0].Action)
	}
	if results[1].Action != protocol.ActionAdded {
		t.Errorf("results[1].action = %q, want added", results[1].Action)
	}
	if len(client.addCalls) != 1 {
		t.Errorf("addCalls = %d, want 1 — only the valid Basic entry", len(client.addCalls))
	}
	if len(errs) != 1 || errs[0].Code != protocol.CodeClozeMarkerMissing {
		t.Errorf("errs = %+v, want one cloze_marker_missing", errs)
	}
	if errs[0].Key == nil || *errs[0].Key != "a.org::0" {
		t.Errorf("errs[0].key = %v, want a.org::0", errs[0].Key)
	}
}

// ---- REQ-011: no-op ----

// AC-015: an unchanged re-sync issues zero note-mutating / card-moving
// requests.
func TestRun_NoOp_HashUnchanged_IssuesNoRequests(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()

	// First run: adds the note.
	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: nil, NoteType: "Basic", SourcePath: "a.org", Title: "T", Body: "B"},
		},
		Census: []protocol.CensusEntry{},
	}
	results, _ := Run(context.Background(), req, reg, client)
	newID := *results[0].NoteID

	// Second run: identical content, now carrying the assigned identifier.
	client2calls := len(client.addCalls) + len(client.updateFieldsCalls) + len(client.updateTagsCalls) + len(client.changeDeckCalls)
	_ = client2calls

	req2 := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: intPtr(newID), NoteType: "Basic", SourcePath: "a.org", Title: "T", Body: "B"},
		},
		Census: []protocol.CensusEntry{{NoteID: newID, SourcePath: "a.org"}},
	}
	beforeAdd, beforeUF, beforeUT, beforeCD := len(client.addCalls), len(client.updateFieldsCalls), len(client.updateTagsCalls), len(client.changeDeckCalls)
	results2, errs2 := Run(context.Background(), req2, reg, client)

	if len(errs2) != 0 {
		t.Fatalf("errs2 = %+v, want none", errs2)
	}
	if len(results2) != 1 || results2[0].Action != protocol.ActionSkipped {
		t.Fatalf("results2 = %+v, want one skipped result", results2)
	}
	if len(client.addCalls) != beforeAdd || len(client.updateFieldsCalls) != beforeUF ||
		len(client.updateTagsCalls) != beforeUT || len(client.changeDeckCalls) != beforeCD {
		t.Errorf("second run issued a note-mutating or card-moving request; want zero")
	}
}

// ---- REQ-010: update ----

// AC-016: a changed body updates in place; no changeDeck fires when the
// resolved deck already matches the registry's recorded value.
func TestRun_Update_BodyChanged_DeckUnchanged_UpdatesFieldsOnly(t *testing.T) {
	reg := newTestRegistry(t)
	oldHash := renderAndHash(t, "Basic", "T", "old body", "DeckA", nil)
	reg.Put(registry.Entry{NoteID: 1001, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "DeckA", ContentHash: oldHash})
	client := newFakeClient()
	client.seedNote(1001, "Basic", map[string]string{"Front": "old", "Back": "old"}, nil, []int{5001})

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: intPtr(1001), NoteType: "Basic", SourcePath: "a.org", Deck: strPtr("DeckA"), Title: "T", Body: "new body"},
		},
		// The heading still exists and is a sync target this run, so the
		// census (a superset of entries[]'s identifiers) reports it too —
		// without this, 1001 would look like an orphan candidate.
		Census: []protocol.CensusEntry{{NoteID: 1001, SourcePath: "a.org"}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 || results[0].Action != protocol.ActionUpdated {
		t.Fatalf("results = %+v, want one updated", results)
	}
	if len(client.updateFieldsCalls) != 1 || client.updateFieldsCalls[0].noteID != 1001 {
		t.Fatalf("updateFieldsCalls = %v, want one call for note 1001", client.updateFieldsCalls)
	}
	// AC-016: the update request must carry the NEW, changed field content —
	// not a stale value carried over from the note's prior state. The entry's
	// body was edited from "old body" to "new body", so the rendered "Back"
	// field sent to AnkiConnect must reflect the new body and must not still
	// read the old one.
	gotBack := client.updateFieldsCalls[0].fields["Back"]
	if !strings.Contains(gotBack, "new body") {
		t.Errorf("updateFieldsCalls[0].fields[Back] = %q, want it to contain the new body content %q", gotBack, "new body")
	}
	if strings.Contains(gotBack, "old body") {
		t.Errorf("updateFieldsCalls[0].fields[Back] = %q, still carries the stale old body content", gotBack)
	}
	if len(client.updateTagsCalls) != 1 {
		t.Errorf("updateTagsCalls = %d, want 1", len(client.updateTagsCalls))
	}
	if len(client.addCalls) != 0 {
		t.Errorf("addCalls = %d, want 0 — this is an update, not an add", len(client.addCalls))
	}
	if len(client.deleteCalls) != 0 {
		t.Errorf("deleteCalls = %v, want none", client.deleteCalls)
	}
	if len(client.changeDeckCalls) != 0 {
		t.Errorf("changeDeckCalls = %v, want none — resolved deck matches the registry's recorded value", client.changeDeckCalls)
	}
}

// AC-017: a changed deck moves the note's cards, never the note itself, and
// the comparison is against the registry's recorded deck — never a live
// query of where the cards currently sit.
func TestRun_Update_DeckChanged_MovesCardsNotNote(t *testing.T) {
	reg := newTestRegistry(t)
	// Same title/body as the entry below; only the deck differs — and
	// because the resolved deck is itself a hash input (design.md §2.4),
	// that alone is enough to make the hash differ too.
	oldHash := renderAndHash(t, "Basic", "T", "B", "DeckA", nil)
	reg.Put(registry.Entry{NoteID: 1001, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "DeckA", ContentHash: oldHash})
	client := newFakeClient()
	client.seedNote(1001, "Basic", map[string]string{"Front": "T", "Back": "B"}, nil, []int{5001, 5002})

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			// body unchanged, deck now resolves to DeckB
			{Key: "a.org::0", NoteID: intPtr(1001), NoteType: "Basic", SourcePath: "a.org", Deck: strPtr("DeckB"), Title: "T", Body: "B"},
		},
		Census: []protocol.CensusEntry{{NoteID: 1001, SourcePath: "a.org"}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 || results[0].Action != protocol.ActionUpdated {
		t.Fatalf("results = %+v, want one updated", results)
	}
	if len(client.notesInfoCalls) != 1 || len(client.notesInfoCalls[0]) != 1 || client.notesInfoCalls[0][0] != 1001 {
		t.Errorf("notesInfoCalls = %v, want one call for [1001]", client.notesInfoCalls)
	}
	if len(client.changeDeckCalls) != 1 {
		t.Fatalf("changeDeckCalls = %d, want 1", len(client.changeDeckCalls))
	}
	if client.changeDeckCalls[0].deck != "DeckB" {
		t.Errorf("changeDeck target = %q, want DeckB", client.changeDeckCalls[0].deck)
	}
	gotCards := client.changeDeckCalls[0].cardIDs
	if len(gotCards) != 2 || gotCards[0] != 5001 || gotCards[1] != 5002 {
		t.Errorf("changeDeck cards = %v, want [5001 5002]", gotCards)
	}

	e, ok := reg.Lookup(1001)
	if !ok {
		t.Fatal("registry lost entry 1001")
	}
	if e.ResolvedDeck != "DeckB" {
		t.Errorf("registry resolved_deck = %q, want DeckB after the move", e.ResolvedDeck)
	}
}

// ---- REQ-021: note-type-change skip, disjoint from REQ-010 ----

func TestRun_NoteTypeChanged_SkipsWithDiagnosticAndTouchesNothing(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Put(registry.Entry{NoteID: 1001, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: "whatever"})
	client := newFakeClient()
	client.seedNote(1001, "Basic", map[string]string{"Front": "T", "Back": "B"}, nil, []int{5001})

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			// declared type now Cloze but registry recorded Basic
			{Key: "a.org::0", NoteID: intPtr(1001), NoteType: "Cloze", SourcePath: "a.org", Title: "T {{c1::x}}", Body: ""},
		},
		Census: []protocol.CensusEntry{{NoteID: 1001, SourcePath: "a.org"}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(results) != 1 || results[0].Action != protocol.ActionSkipped {
		t.Fatalf("results = %+v, want one skipped", results)
	}
	if len(errs) != 1 || errs[0].Code != protocol.CodeNoteTypeChangeUnsupported {
		t.Fatalf("errs = %+v, want one note_type_change_unsupported", errs)
	}
	if len(client.updateFieldsCalls) != 0 || len(client.updateTagsCalls) != 0 || len(client.changeDeckCalls) != 0 {
		t.Error("note-type-change skip must touch no field and move no card")
	}
	e, _ := reg.Lookup(1001)
	if e.NoteType != "Basic" {
		t.Errorf("registry entry mutated; note type = %q, want unchanged Basic", e.NoteType)
	}
}

// ---- REQ-012: reconcile ----

// AC-018: an identifier with no registry entry is reconciled against Anki
// (matched → update, not duplicated), or added fresh (unmatched → add,
// never left unsynchronized).
func TestRun_Reconcile_MatchedInCollection_UpdatesRatherThanDuplicates(t *testing.T) {
	reg := newTestRegistry(t) // empty registry — this is the "lost registry" case
	client := newFakeClient()
	client.seedNote(1001, "Basic", map[string]string{"Front": "old", "Back": "old"}, nil, []int{5001})

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: intPtr(1001), NoteType: "Basic", SourcePath: "a.org", Deck: strPtr("DeckA"), Title: "T", Body: "new"},
		},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 || results[0].Action != protocol.ActionUpdated {
		t.Fatalf("results = %+v, want one updated (reconciled), never a second add", results)
	}
	if *results[0].NoteID != 1001 {
		t.Errorf("reconciled result note_id = %d, want 1001 unchanged", *results[0].NoteID)
	}
	if len(client.addCalls) != 0 {
		t.Errorf("addCalls = %d, want 0 — no duplicate note", len(client.addCalls))
	}
	// D-3 / §D.7: adopted as though the recorded deck had differed too, so
	// the reconciling update re-asserts deck placement even though DeckA is
	// the target's own first-seen resolution.
	if len(client.changeDeckCalls) != 1 || client.changeDeckCalls[0].deck != "DeckA" {
		t.Errorf("changeDeckCalls = %v, want one forced move to DeckA", client.changeDeckCalls)
	}
	e, ok := reg.Lookup(1001)
	if !ok {
		t.Fatal("registry did not adopt the reconciled entry")
	}
	if e.SourcePath != "a.org" || e.ResolvedDeck != "DeckA" {
		t.Errorf("adopted registry entry = %+v", e)
	}
}

func TestRun_Reconcile_NotInCollection_AddsFreshNote(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	// note 9999 does not exist in the stub collection at all.

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "b.org::0", NoteID: intPtr(9999), NoteType: "Basic", SourcePath: "b.org", Title: "Q", Body: "Body"},
		},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 || results[0].Action != protocol.ActionAdded {
		t.Fatalf("results = %+v, want one added", results)
	}
	if *results[0].NoteID == 9999 {
		t.Error("result note_id must be the freshly assigned id, not the stale 9999")
	}
	if len(client.addCalls) != 1 {
		t.Errorf("addCalls = %d, want 1", len(client.addCalls))
	}
	if _, ok := reg.Lookup(9999); ok {
		t.Error("registry must not carry an entry for the stale, nonexistent identifier")
	}
}

// Reconcile where the collection has the identifier but under a different
// note type — must also fall through to the add path (D-9's own-drawer type
// match discipline extends to REQ-012's adoption test).
func TestRun_Reconcile_TypeMismatch_AddsFreshNote(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.seedNote(1001, "Cloze", map[string]string{"Text": "{{c1::x}}"}, nil, []int{5001})

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: intPtr(1001), NoteType: "Basic", SourcePath: "a.org", Title: "T", Body: "B"},
		},
	}

	results, errs := Run(context.Background(), req, reg, client)
	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 || results[0].Action != protocol.ActionAdded {
		t.Fatalf("results = %+v, want add on type mismatch", results)
	}
	if len(client.addCalls) != 1 {
		t.Errorf("addCalls = %d, want 1", len(client.addCalls))
	}
}
