package planner

import (
	"context"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// AC-C-020b — the sync-path complement. A heading hand-edited from the
// stock type its registry entry records to that type's imoogi- counterpart
// is REQ-C-021's case, not REQ-C-020's: an ORDINARY sync run must skip it
// with note_type_change_unsupported and write no field.
//
// This is the parent SPEC's REQ-021 behavior, and the reason it needs a
// test of its own is that the declared name reaches the renderer BEFORE the
// registry comparison. A renderer that does not recognize the counterpart
// name reports the entry as an org_parse_error failure instead — a
// different code, a different action, and a diagnostic that sends the user
// looking at their Org markup rather than at imoogi-anki-setup.
func TestRun_HandEditedImoogiNoteType_SkipsWithNoteTypeChangeUnsupported(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.seedNote(41, "Basic", map[string]string{"Front": "Capital of France", "Back": "Paris."}, nil, []int{4101})

	reg.Put(registry.Entry{
		NoteID:       41,
		SourcePath:   "a.org",
		NoteType:     "Basic",
		ResolvedDeck: "Inbox",
		ContentHash:  renderAndHash(t, "Basic", "Capital of France", "Paris.", "Inbox", nil),
	})

	req := protocol.Request{
		Config: baseConfig(),
		Census: []protocol.CensusEntry{{NoteID: 41, SourcePath: "a.org"}},
		Entries: []protocol.Entry{{
			Key:        "a.org::0",
			NoteID:     intPtr(41),
			NoteType:   "imoogi-Basic", // hand-edited; the registry still records Basic
			SourcePath: "a.org",
			Title:      "Capital of France",
			Body:       "Paris.",
		}},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Action != protocol.ActionSkipped {
		t.Errorf("action = %q, want %q", results[0].Action, protocol.ActionSkipped)
	}
	if len(errs) != 1 {
		t.Fatalf("errs = %+v, want exactly one", errs)
	}
	if errs[0].Code != protocol.CodeNoteTypeChangeUnsupported {
		t.Errorf("code = %q, want %q", errs[0].Code, protocol.CodeNoteTypeChangeUnsupported)
	}
	if len(client.updateFieldsCalls) != 0 {
		t.Errorf("updateNoteFields fired %d times, want 0 — a skipped entry writes no field", len(client.updateFieldsCalls))
	}
}

// migrateFixture plants one already-synced entry on a stock note type: the
// stub collection holds the note, and the registry records it with the hash
// the renderer actually produces for that content. It is the shape every
// migration candidate has by definition — an entry imoogi created before the
// imoogi-owned types existed.
func migrateFixture(t *testing.T, id int, key, title, body string) (*fakeClient, *registry.Registry, protocol.Entry) {
	t.Helper()
	client := newFakeClient()
	client.decks["Inbox"] = true
	client.seedNote(id, "Basic", renderedFields(t, "Basic", title, body), nil, []int{id*1000 + 1})

	reg := newTestRegistry(t)
	reg.Put(registry.Entry{
		NoteID:       id,
		SourcePath:   "a.org",
		NoteType:     "Basic",
		ResolvedDeck: "Inbox",
		ContentHash:  renderAndHash(t, "Basic", title, body, "Inbox", nil),
	})

	return client, reg, protocol.Entry{
		Key:        key,
		NoteID:     intPtr(id),
		NoteType:   "Basic",
		SourcePath: "a.org",
		Title:      title,
		Body:       body,
	}
}

// seedSecondCandidate adds a second already-synced stock entry to an
// existing fixture, so a test can assert that the run CONTINUES past
// whatever happens to the first one.
func seedSecondCandidate(t *testing.T, client *fakeClient, reg *registry.Registry, id int, key, title, body string) protocol.Entry {
	t.Helper()
	client.seedNote(id, "Basic", renderedFields(t, "Basic", title, body), nil, []int{id*1000 + 1})
	reg.Put(registry.Entry{
		NoteID:       id,
		SourcePath:   "a.org",
		NoteType:     "Basic",
		ResolvedDeck: "Inbox",
		ContentHash:  renderAndHash(t, "Basic", title, body, "Inbox", nil),
	})
	return protocol.Entry{
		Key: key, NoteID: intPtr(id), NoteType: "Basic",
		SourcePath: "a.org", Title: title, Body: body,
	}
}

func renderedFields(t *testing.T, noteType, title, body string) map[string]string {
	t.Helper()
	fields, err := orgdoc.Render(noteType, title, body)
	if err != nil {
		t.Fatalf("renderedFields: %v", err)
	}
	return fields
}

func migrateRequest(entries ...protocol.Entry) protocol.Request {
	return protocol.Request{Config: baseConfig(), Entries: entries}
}

// AC-C-018b — the dry run reports every candidate through results[] and
// records ZERO AnkiConnect writes.
//
// The zero-writes assertion is what makes design.md §7.1's ordering
// defensible: the count is obtained BEFORE the user is asked, so the prompt
// can name it, and obtaining it changes nothing that consent would have
// protected. Scoped to writes rather than to all calls, because the
// handshake read is permitted (acceptance.md's own wording).
func TestMigrate_DryRun_ReportsEveryCandidateAndWritesNothing(t *testing.T) {
	client, reg, first := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")
	second := seedSecondCandidate(t, client, reg, 42, "a.org::1", "Capital of Italy", "Rome.")

	results, errs := Migrate(context.Background(), migrateRequest(first, second), reg, client, true)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2 — the candidate count IS len(results)", len(results))
	}
	for i, want := range []int{41, 42} {
		if results[i].Action != protocol.ActionMigrateCandidate {
			t.Errorf("results[%d].Action = %q, want %q", i, results[i].Action, protocol.ActionMigrateCandidate)
		}
		if results[i].NoteID == nil || *results[i].NoteID != want {
			t.Errorf("results[%d].NoteID = %v, want the EXISTING identifier %d", i, results[i].NoteID, want)
		}
		if results[i].Key == nil {
			t.Errorf("results[%d].Key is nil, want the request entry's key echoed back", i)
		}
	}

	if len(client.writeSequence) != 0 {
		t.Errorf("dry run issued write requests %v, want none — REQ-C-019's consent rests on this", client.writeSequence)
	}
	assertNoModelWrites(t, client)
}

// AC-C-018c (Go half) — the dry run leaves the registry byte-unchanged. The
// Elisp half (the declined prompt issuing no writing request at all) is M6's.
func TestMigrate_DryRun_LeavesTheRegistryUntouched(t *testing.T) {
	client, reg, entry := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")
	before := reg.All()

	if _, errs := Migrate(context.Background(), migrateRequest(entry), reg, client, true); len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}

	after := reg.All()
	if len(after) != len(before) {
		t.Fatalf("registry holds %d entries after a dry run, want the original %d", len(after), len(before))
	}
	if after[0] != before[0] {
		t.Errorf("registry entry changed under a dry run:\n before = %+v\n  after = %+v", before[0], after[0])
	}
}

// AC-C-019a — add before delete, under the counterpart type, with fields
// RENDERED rather than copied from the original.
//
// The ordering is the whole point (design.md §7.2): delete-first leaves a
// window where the registry names an identifier Anki no longer holds, and a
// crash there strands the entry. Add-first leaves a duplicate instead —
// visible, recoverable, never a loss. A per-action count cannot express
// that, which is why writeSequence exists.
func TestMigrate_AddsUnderTheCounterpartBeforeDeletingTheOriginal(t *testing.T) {
	client, reg, entry := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")

	results, errs := Migrate(context.Background(), migrateRequest(entry), reg, client, false)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	addAt, delAt := indexOf(client.writeSequence, "addNote"), indexOf(client.writeSequence, "deleteNotes")
	if addAt < 0 || delAt < 0 {
		t.Fatalf("writeSequence = %v, want both an addNote and a deleteNotes", client.writeSequence)
	}
	if addAt > delAt {
		t.Errorf("writeSequence = %v — deleteNotes preceded addNote; design.md §7.2 requires the add first", client.writeSequence)
	}

	if len(client.addCalls) != 1 {
		t.Fatalf("addCalls = %d, want 1", len(client.addCalls))
	}
	if client.addCalls[0].modelName != "imoogi-Basic" {
		t.Errorf("add modelName = %q, want the imoogi- counterpart", client.addCalls[0].modelName)
	}
	if got, want := client.addCalls[0].fields["Front"], renderedFields(t, "Basic", "Capital of France", "Paris.")["Front"]; got != want {
		t.Errorf("add Front = %q, want the RENDERED value %q — the fields are re-rendered, not copied off the original", got, want)
	}
	if len(client.deleteCalls) != 1 || len(client.deleteCalls[0]) != 1 || client.deleteCalls[0][0] != 41 {
		t.Errorf("deleteCalls = %v, want exactly one delete naming the original note 41", client.deleteCalls)
	}

	assertNoModelWrites(t, client)
}

// AC-C-019b (Go half) — the registry entry is REPLACED: new identifier, the
// imoogi-owned type, and a hash taken under that type. The hash must be
// taken under the counterpart, or the very next ordinary sync recomputes a
// different one and reissues an update for a note nothing changed on.
func TestMigrate_ReplacesTheRegistryEntryWithTheNewIdentifierAndType(t *testing.T) {
	client, reg, entry := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")

	results, _ := Migrate(context.Background(), migrateRequest(entry), reg, client, false)

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Action != protocol.ActionAdded {
		t.Errorf("action = %q, want %q — the front end's existing write-back acts on `added`",
			results[0].Action, protocol.ActionAdded)
	}
	newID := results[0].NoteID
	if newID == nil || *newID == 41 {
		t.Fatalf("results[0].NoteID = %v, want the NEW identifier (not the original 41)", newID)
	}

	if _, stillThere := reg.Lookup(41); stillThere {
		t.Errorf("registry still holds the original identifier 41 after a successful migration")
	}
	e, ok := reg.Lookup(*newID)
	if !ok {
		t.Fatalf("registry has no entry for the new identifier %d", *newID)
	}
	if e.NoteType != "imoogi-Basic" {
		t.Errorf("registry note_type = %q, want imoogi-Basic", e.NoteType)
	}
	if want := renderAndHash(t, "Basic", "Capital of France", "Paris.", "Inbox", nil); e.ContentHash == want {
		t.Errorf("registry content_hash was taken under the STOCK type; it must be taken under the counterpart, "+
			"or the next ordinary sync sees a mismatch and reissues an update (hash = %q)", e.ContentHash)
	}
	if e.SourcePath != "a.org" || e.ResolvedDeck != "Inbox" {
		t.Errorf("registry entry = %+v, want source_path a.org and resolved_deck Inbox carried over", e)
	}
}

// AC-C-019c — the declared-type residue. An entry declaring a type that is
// neither its recorded stock type nor that type's counterpart is REQ-C-021's
// case even inside a confirmed migration, and the partition is in
// requirement text rather than in branch ordering (design.md §7.4).
func TestMigrate_DeclaredTypeIsNeitherRecordedNorCounterpart_SkipsUnsupported(t *testing.T) {
	client, reg, entry := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")
	entry.NoteType = "Cloze" // neither the recorded Basic nor imoogi-Basic

	results, errs := Migrate(context.Background(), migrateRequest(entry), reg, client, false)

	if len(results) != 1 || results[0].Action != protocol.ActionSkipped {
		t.Fatalf("results = %+v, want exactly one skipped entry", results)
	}
	if len(errs) != 1 || errs[0].Code != protocol.CodeNoteTypeChangeUnsupported {
		t.Fatalf("errs = %+v, want exactly one %s", errs, protocol.CodeNoteTypeChangeUnsupported)
	}
	if len(client.writeSequence) != 0 {
		t.Errorf("writeSequence = %v, want none — nothing is added or deleted for a non-candidate", client.writeSequence)
	}
}

// AC-C-020a — a failed add is a no-op on the original, and the run
// continues. The invariant underneath is REQ-C-022's closing clause: no
// entry ever reaches a state where the registry names an identifier the
// collection no longer holds. Add-before-delete is what makes it hold for
// free — the delete is simply never reached.
func TestMigrate_FailedAdd_LeavesOriginalAndRegistryIntactAndContinues(t *testing.T) {
	client, reg, first := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")
	second := seedSecondCandidate(t, client, reg, 42, "a.org::1", "Capital of Italy", "Rome.")
	before := reg.All()
	client.addNoteErr = errFakeTransport

	results, errs := Migrate(context.Background(), migrateRequest(first, second), reg, client, false)

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2 — the run continues past a failed add", len(results))
	}
	for i, r := range results {
		if r.Action != protocol.ActionSkipped {
			t.Errorf("results[%d].Action = %q, want %q", i, r.Action, protocol.ActionSkipped)
		}
	}
	if len(errs) != 2 {
		t.Fatalf("errs = %+v, want one per failed entry", errs)
	}
	for i, e := range errs {
		if e.Code != protocol.CodeMigrationAddFailed {
			t.Errorf("errs[%d].Code = %q, want %q", i, e.Code, protocol.CodeMigrationAddFailed)
		}
		if e.Key == nil {
			t.Errorf("errs[%d].Key is nil, want the per-entry key", i)
		}
	}

	if len(client.deleteCalls) != 0 {
		t.Errorf("deleteCalls = %v, want none — the delete is never reached when the add failed", client.deleteCalls)
	}
	for _, id := range []int{41, 42} {
		if _, ok := client.notes[id]; !ok {
			t.Errorf("original note %d no longer exists in the collection", id)
		}
	}
	after := reg.All()
	if len(after) != len(before) {
		t.Fatalf("registry holds %d entries, want the original %d", len(after), len(before))
	}
	for i := range before {
		if after[i] != before[i] {
			t.Errorf("registry entry %d changed:\n before = %+v\n  after = %+v", i, before[i], after[i])
		}
	}
}

// The counterpart-declaring entry is a candidate too: REQ-C-020's gate
// accepts EITHER the recorded stock type or its counterpart as the declared
// type, which is what makes a re-run after a partial write-back converge
// rather than fall into REQ-C-021's skip.
func TestMigrate_DeclaredCounterpartType_IsStillACandidate(t *testing.T) {
	client, reg, entry := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")
	entry.NoteType = "imoogi-Basic"

	results, errs := Migrate(context.Background(), migrateRequest(entry), reg, client, false)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 || results[0].Action != protocol.ActionAdded {
		t.Fatalf("results = %+v, want one added entry", results)
	}
	if len(client.addCalls) != 1 || client.addCalls[0].modelName != "imoogi-Basic" {
		t.Errorf("addCalls = %+v, want one add under imoogi-Basic", client.addCalls)
	}
}

// An entry whose registry record already names an imoogi-owned type has
// nothing to migrate. It is not a candidate, and — because its declared and
// recorded types agree — it is not REQ-C-021's mismatch either, so it is
// reported not at all rather than as a skip.
func TestMigrate_AlreadyOnTheOwnedType_IsNotACandidate(t *testing.T) {
	client, reg, entry := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")
	reg.Put(registry.Entry{
		NoteID: 41, SourcePath: "a.org", NoteType: "imoogi-Basic",
		ResolvedDeck: "Inbox", ContentHash: "whatever",
	})
	entry.NoteType = "imoogi-Basic"

	for _, dryRun := range []bool{true, false} {
		results, errs := Migrate(context.Background(), migrateRequest(entry), reg, client, dryRun)
		if len(results) != 0 || len(errs) != 0 {
			t.Errorf("dryRun=%v: results = %+v, errs = %+v, want both empty", dryRun, results, errs)
		}
	}
	if len(client.writeSequence) != 0 {
		t.Errorf("writeSequence = %v, want none", client.writeSequence)
	}
}

// The Cloze half of the candidate set. Basic → imoogi-Basic is the case
// every other test here exercises; this one keeps the second stock type from
// being an untested arm, since a mapping that handled only Basic would pass
// all of them while silently leaving every Cloze entry unmigrated.
func TestMigrate_ClozeEntryMigratesOntoTheClozeCounterpart(t *testing.T) {
	const title, body = "The capital of France is {{c1::Paris}}", "Since 987."
	client := newFakeClient()
	client.decks["Inbox"] = true
	client.seedNote(41, "Cloze", renderedFields(t, "Cloze", title, body), nil, []int{41001})

	reg := newTestRegistry(t)
	reg.Put(registry.Entry{
		NoteID: 41, SourcePath: "a.org", NoteType: "Cloze",
		ResolvedDeck: "Inbox", ContentHash: renderAndHash(t, "Cloze", title, body, "Inbox", nil),
	})
	entry := protocol.Entry{
		Key: "a.org::0", NoteID: intPtr(41), NoteType: "Cloze",
		SourcePath: "a.org", Title: title, Body: body,
	}

	results, errs := Migrate(context.Background(), migrateRequest(entry), reg, client, false)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 || results[0].Action != protocol.ActionAdded {
		t.Fatalf("results = %+v, want one added entry", results)
	}
	if len(client.addCalls) != 1 || client.addCalls[0].modelName != "imoogi-Cloze" {
		t.Fatalf("addCalls = %+v, want one add under imoogi-Cloze", client.addCalls)
	}
	if _, ok := client.addCalls[0].fields["Text"]; !ok {
		t.Errorf("add fields = %v, want the Cloze Text field", client.addCalls[0].fields)
	}
	e, ok := reg.Lookup(*results[0].NoteID)
	if !ok || e.NoteType != "imoogi-Cloze" {
		t.Errorf("registry entry = %+v (found=%v), want it recorded on imoogi-Cloze", e, ok)
	}
}

// design.md §7.3, "Render fails for a candidate": skip that entry with its
// render diagnostic, leave the original untouched, continue. The diagnostic
// is the RENDER's own code, not a migration code — the entry's problem is
// its Org markup, and reporting migration_add_failed would send the user
// looking in the wrong place.
func TestMigrate_RenderFailure_SkipsWithTheRenderDiagnosticAndTouchesNothing(t *testing.T) {
	client, reg, entry := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")
	reg.Put(registry.Entry{
		NoteID: 41, SourcePath: "a.org", NoteType: "Cloze",
		ResolvedDeck: "Inbox", ContentHash: "recorded-under-cloze",
	})
	entry.NoteType = "Cloze"                // recorded Cloze too, so this IS a candidate...
	entry.Body = "no cloze marker anywhere" // ...whose content cannot render

	before := reg.All()
	results, errs := Migrate(context.Background(), migrateRequest(entry), reg, client, false)

	if len(results) != 1 || results[0].Action != protocol.ActionSkipped {
		t.Fatalf("results = %+v, want exactly one skipped entry", results)
	}
	if len(errs) != 1 || errs[0].Code != protocol.CodeClozeMarkerMissing {
		t.Fatalf("errs = %+v, want exactly one %s", errs, protocol.CodeClozeMarkerMissing)
	}
	if len(client.writeSequence) != 0 {
		t.Errorf("writeSequence = %v, want none — an unrenderable candidate writes nothing", client.writeSequence)
	}
	if after := reg.All(); len(after) != len(before) || after[0] != before[0] {
		t.Errorf("registry changed:\n before = %+v\n  after = %+v", before, after)
	}
}

// design.md §7.3, "DeleteNotes fails after a successful add": the new note
// and the updated registry entry STAND, the original survives as a visible
// duplicate, and the failure is reported. Never a lost note.
//
// Abandoning the registry swap here would be the tempting-looking rollback
// and the wrong one: the new note exists, so a registry still naming the old
// identifier is exactly the "names an identifier the collection no longer
// holds" state REQ-C-022 forbids, reached from the opposite direction.
func TestMigrate_DeleteFailureAfterSuccessfulAdd_KeepsTheNewNoteAndReports(t *testing.T) {
	client, reg, entry := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")
	client.deleteNotesErr = errFakeTransport

	results, errs := Migrate(context.Background(), migrateRequest(entry), reg, client, false)

	if len(results) != 1 || results[0].Action != protocol.ActionAdded {
		t.Fatalf("results = %+v, want the migration still reported as added", results)
	}
	newID := results[0].NoteID
	if newID == nil {
		t.Fatal("results[0].NoteID is nil, want the new identifier")
	}
	if len(errs) != 1 || errs[0].Code != protocol.CodeAnkiConnectError {
		t.Fatalf("errs = %+v, want exactly one %s naming the failed delete", errs, protocol.CodeAnkiConnectError)
	}
	if errs[0].Key == nil {
		t.Error("the delete error carries no key; it belongs to the entry being migrated")
	}

	if _, ok := client.notes[41]; !ok {
		t.Error("the original note is gone even though the delete failed")
	}
	if _, stillThere := reg.Lookup(41); stillThere {
		t.Error("registry still names the original identifier; the swap must stand so no entry names a note that no longer exists")
	}
	e, ok := reg.Lookup(*newID)
	if !ok || e.NoteType != "imoogi-Basic" {
		t.Errorf("registry entry for the new note = %+v (found=%v), want it recorded on imoogi-Basic", e, ok)
	}
}

// A field-resolution failure is a skip, not a failure, and it happens BEFORE
// the add — so, like every other early return on this path, the original
// note and its registry entry are untouched.
func TestMigrate_FieldResolutionFailure_SkipsBeforeAddingAnything(t *testing.T) {
	client, reg, entry := migrateFixture(t, 41, "a.org::0", "Capital of France", "Paris.")
	client.modelFields = map[string][]string{"imoogi-Basic": {"Question", "Answer"}}
	before := reg.All()

	results, errs := Migrate(context.Background(), migrateRequest(entry), reg, client, false)

	if len(results) != 1 || results[0].Action != protocol.ActionSkipped {
		t.Fatalf("results = %+v, want exactly one skipped entry", results)
	}
	if len(errs) != 1 || errs[0].Code != protocol.CodeNoteFieldMissing {
		t.Fatalf("errs = %+v, want exactly one %s", errs, protocol.CodeNoteFieldMissing)
	}
	if len(client.addCalls) != 0 || len(client.deleteCalls) != 0 {
		t.Errorf("addCalls = %+v, deleteCalls = %v, want neither", client.addCalls, client.deleteCalls)
	}
	if after := reg.All(); len(after) != len(before) || after[0] != before[0] {
		t.Errorf("registry changed:\n before = %+v\n  after = %+v", before, after)
	}
}

// AC-C-022b mechanism — no request on a model-WRITE endpoint, on either
// migrate mode. The read endpoints are deliberately out of scope:
// modelFieldNames("Basic") is a required call on the stock path.
func assertNoModelWrites(t *testing.T, client *fakeClient) {
	t.Helper()
	if n := len(client.createModelCalls); n != 0 {
		t.Errorf("createModel fired %d times, want 0", n)
	}
	if n := len(client.updateModelStylingCalls); n != 0 {
		t.Errorf("updateModelStyling fired %d times, want 0", n)
	}
	if n := len(client.updateModelTemplatesCalls); n != 0 {
		t.Errorf("updateModelTemplates fired %d times, want 0", n)
	}
}

func indexOf(haystack []string, needle string) int {
	for i, s := range haystack {
		if s == needle {
			return i
		}
	}
	return -1
}
