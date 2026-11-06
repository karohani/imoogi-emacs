package planner

import (
	"context"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/hashing"
	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// SPEC-ANKISRC-001 DD-2: the exact anchor appended to the back field.
func wantSourceAnchor(encodedID string) string {
	return `<div class="imoogi-src"><a href="imoogi-anki://open?id=` + encodedID +
		`">Emacs에서 원본 열기</a></div>`
}

// AC-ASRC-006: the link lands at the END of Back (Basic shape) or Back Extra
// (Cloze shape), for imoogi-owned and stock types alike; the front field is
// unchanged; an entry without org_id is byte-identical to the baseline; an
// org_id outside the character set adds no link, still adds the note, and
// reports org_id_invalid against its key.
func TestRun_SourceLink_AppendedToBackFieldOnly(t *testing.T) {
	reg := newTestRegistry(t)
	client := newFakeClient()

	uuid := "6F1C8E0A-3B2D-4C5E-9F10-123456789ABC"
	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteType: "imoogi-Basic", SourcePath: "a.org", Title: "Q1", Body: "A1", OrgID: uuid},
			{Key: "a.org::1", NoteType: "imoogi-Cloze", SourcePath: "a.org", Title: "C", Body: "The {{c1::sun}} rises.", OrgID: "imoogi:abc"},
			{Key: "a.org::2", NoteType: "Basic", SourcePath: "a.org", Title: "Q3", Body: "A3", OrgID: "x@host.example"},
			{Key: "a.org::3", NoteType: "Basic", SourcePath: "a.org", Title: "Q4", Body: "A4"},
			{Key: "a.org::4", NoteType: "Basic", SourcePath: "a.org", Title: "Q5", Body: "A5", OrgID: "a b"},
			{Key: "a.org::5", NoteType: "Cloze", SourcePath: "a.org", Title: "C6", Body: "The {{c1::moon}}.", OrgID: "ts.1"},
		},
	}

	results, errs := Run(context.Background(), req, reg, client)

	if len(results) != 6 {
		t.Fatalf("results = %+v, want 6", results)
	}
	for _, r := range results {
		if r.Action != protocol.ActionAdded {
			t.Errorf("result %s = %q, want added (an invalid org_id never blocks the note)", *r.Key, r.Action)
		}
	}
	if len(errs) != 1 || errs[0].Code != protocol.CodeOrgIDInvalid || errs[0].Key == nil || *errs[0].Key != "a.org::4" {
		t.Fatalf("errs = %+v, want exactly one org_id_invalid keyed a.org::4", errs)
	}
	if len(client.addCalls) != 6 {
		t.Fatalf("addCalls = %d, want 6", len(client.addCalls))
	}

	basicFront, _ := orgdoc.Render("Basic", "Q1", "A1")
	add := client.addCalls[0].fields
	if add["Front"] != basicFront["Front"] {
		t.Errorf("Front = %q, want the unlinked render %q", add["Front"], basicFront["Front"])
	}
	if add["Back"] != basicFront["Back"]+wantSourceAnchor("6F1C8E0A-3B2D-4C5E-9F10-123456789ABC") {
		t.Errorf("Back = %q, want render + anchor", add["Back"])
	}

	cloze, _ := orgdoc.Render("Cloze", "C", "The {{c1::sun}} rises.")
	add = client.addCalls[1].fields
	if add["Text"] != cloze["Text"] {
		t.Errorf("Text = %q, want unchanged %q", add["Text"], cloze["Text"])
	}
	if add["Back Extra"] != cloze["Back Extra"]+wantSourceAnchor("imoogi%3Aabc") {
		t.Errorf("Back Extra = %q, want render + percent-encoded anchor", add["Back Extra"])
	}

	stock, _ := orgdoc.Render("Basic", "Q3", "A3")
	if got := client.addCalls[2].fields["Back"]; got != stock["Back"]+wantSourceAnchor("x%40host.example") {
		t.Errorf("stock Basic Back = %q, want render + anchor", got)
	}

	plain, _ := orgdoc.Render("Basic", "Q4", "A4")
	for name, value := range plain {
		if client.addCalls[3].fields[name] != value {
			t.Errorf("no-org_id %s = %q, want baseline %q", name, client.addCalls[3].fields[name], value)
		}
	}

	invalid, _ := orgdoc.Render("Basic", "Q5", "A5")
	for name, value := range invalid {
		if client.addCalls[4].fields[name] != value {
			t.Errorf("invalid-org_id %s = %q, want no link %q", name, client.addCalls[4].fields[name], value)
		}
	}

	stockCloze, _ := orgdoc.Render("Cloze", "C6", "The {{c1::moon}}.")
	if got := client.addCalls[5].fields["Extra"]; got != stockCloze["Back Extra"]+wantSourceAnchor("ts.1") {
		t.Errorf("stock Cloze Extra = %q, want render + anchor", got)
	}

	for _, call := range client.addCalls {
		for name, value := range call.fields {
			if strings.Contains(value, "a.org") || strings.Contains(value, "/tmp/notes") {
				t.Errorf("field %s carries a file path: %q (REQ-ASRC-005.2)", name, value)
			}
		}
	}
}

// AC-ASRC-007: a note registered under the pre-link hash is updated ONCE, in
// place, when its entry first carries org_id; the next run is a no-op; and
// the orphan-confirmation rehash of the stored fields equals the recorded
// hash, so the note stays owned.
func TestRun_SourceLink_UpdatesExistingNoteOnceAndHashStaysConsistent(t *testing.T) {
	reg := newTestRegistry(t)
	oldHash := renderAndHash(t, "Basic", "T", "B", "Inbox", nil)
	reg.Put(registry.Entry{NoteID: 1001, SourcePath: "a.org", NoteType: "Basic", ResolvedDeck: "Inbox", ContentHash: oldHash})
	oldFields, _ := orgdoc.Render("Basic", "T", "B")
	client := newFakeClient()
	client.decks["Inbox"] = true
	client.seedNote(1001, "Basic", oldFields, nil, []int{5001})

	req := protocol.Request{
		Config: baseConfig(),
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: intPtr(1001), NoteType: "Basic", SourcePath: "a.org", Title: "T", Body: "B", OrgID: "id-1"},
		},
		Census: []protocol.CensusEntry{{NoteID: 1001, SourcePath: "a.org"}},
	}

	results, errs := Run(context.Background(), req, reg, client)
	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(results) != 1 || results[0].Action != protocol.ActionUpdated || *results[0].NoteID != 1001 {
		t.Fatalf("results = %+v, want one updated for 1001", results)
	}
	if len(client.updateFieldsCalls) != 1 || len(client.addCalls) != 0 || len(client.deleteCalls) != 0 {
		t.Fatalf("update=%d add=%d delete=%d, want 1/0/0",
			len(client.updateFieldsCalls), len(client.addCalls), len(client.deleteCalls))
	}

	before := len(client.writeSequence)
	results, errs = Run(context.Background(), req, reg, client)
	if len(errs) != 0 || len(results) != 1 || results[0].Action != protocol.ActionSkipped {
		t.Fatalf("second run results=%+v errs=%+v, want one skipped no-op", results, errs)
	}
	if len(client.writeSequence) != before {
		t.Errorf("second run issued writes %v, want none", client.writeSequence[before:])
	}

	recorded, _ := reg.Lookup(1001)
	infos, _ := client.NotesInfo(context.Background(), []int{1001})
	rehash := hashing.Hash(infos[0].ModelName, fieldValuesToStrings(infos[0].Fields), recorded.ResolvedDeck, infos[0].Tags)
	if rehash != recorded.ContentHash {
		t.Errorf("orphan-confirmation rehash %s != recorded %s", rehash, recorded.ContentHash)
	}
}
