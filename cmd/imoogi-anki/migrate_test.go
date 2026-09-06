package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// collectionRecorder is a recording AnkiConnect stub that holds MORE THAN
// ONE note. wireRecorder deliberately remembers a single add, which is all
// the sync delete path needs; the migration path needs two notes alive at
// once — the stock original under confirmation and the counterpart
// replacement just added — so the ownership check reads the original's real
// fields rather than the replacement's.
//
// It speaks genuine wire JSON and answers from its own collection, so the
// hash recompute the delete path performs is a real round trip.
type collectionRecorder struct {
	mu      sync.Mutex
	actions []string

	notes  map[int]*recordedNote
	nextID int
}

type recordedNote struct {
	modelName string
	fields    map[string]string
	tags      []string
}

func newCollectionRecorder() *collectionRecorder {
	return &collectionRecorder{notes: map[int]*recordedNote{}, nextID: 1000}
}

func (c *collectionRecorder) snapshotActions() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.actions...)
}

func (c *collectionRecorder) noteIDs() []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	ids := make([]int, 0, len(c.notes))
	for id := range c.notes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func (c *collectionRecorder) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var envelope struct {
			Action string          `json:"action"`
			Params json.RawMessage `json:"params"`
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("stub: reading request body: %v", err)
			return
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			t.Errorf("stub: request is not AnkiConnect wire JSON: %v\ngot: %s", err, body)
			return
		}

		c.mu.Lock()
		c.actions = append(c.actions, envelope.Action)
		result := c.answer(t, envelope.Action, envelope.Params)
		c.mu.Unlock()

		rw.Header().Set("Content-Type", "application/json")
		encoded, err := json.Marshal(struct {
			Result any     `json:"result"`
			Error  *string `json:"error"`
		}{Result: result, Error: nil})
		if err != nil {
			t.Errorf("stub: encoding response for %s: %v", envelope.Action, err)
			return
		}
		if _, err := rw.Write(encoded); err != nil {
			t.Errorf("stub: writing response for %s: %v", envelope.Action, err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// answer produces the "result" payload for one action. The caller holds c.mu.
func (c *collectionRecorder) answer(t *testing.T, action string, params json.RawMessage) any {
	t.Helper()
	switch action {
	case "requestPermission":
		return map[string]any{"permission": "granted", "requireApiKey": false, "version": 6}

	case "deckNames":
		return []string{"Inbox"}

	case "modelFieldNames":
		var p struct {
			ModelName string `json:"modelName"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			t.Fatalf("modelFieldNames params unmarshal: %v", err)
		}
		switch p.ModelName {
		case "Basic", "imoogi-Basic":
			return []string{"Front", "Back"}
		case "Cloze":
			return []string{"Text", "Extra"}
		case "imoogi-Cloze":
			return []string{"Text", "Back Extra"}
		default:
			t.Fatalf("modelFieldNames: unexpected model %q", p.ModelName)
			return nil
		}

	case "addNote":
		var p struct {
			Note struct {
				ModelName string            `json:"modelName"`
				Fields    map[string]string `json:"fields"`
				Tags      []string          `json:"tags"`
			} `json:"note"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			t.Errorf("stub: addNote params are not the documented shape: %v\ngot: %s", err, params)
			return nil
		}
		c.nextID++
		c.notes[c.nextID] = &recordedNote{modelName: p.Note.ModelName, fields: p.Note.Fields, tags: p.Note.Tags}
		return c.nextID

	case "notesInfo":
		var p struct {
			Notes []int `json:"notes"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			t.Errorf("stub: notesInfo params are not the documented shape: %v\ngot: %s", err, params)
			return nil
		}
		out := make([]map[string]any, 0, len(p.Notes))
		for _, id := range p.Notes {
			n, ok := c.notes[id]
			if !ok {
				out = append(out, map[string]any{})
				continue
			}
			names := make([]string, 0, len(n.fields))
			for name := range n.fields {
				names = append(names, name)
			}
			sort.Strings(names)
			fields := map[string]any{}
			for i, name := range names {
				fields[name] = map[string]any{"value": n.fields[name], "order": i}
			}
			tags := n.tags
			if tags == nil {
				tags = []string{}
			}
			out = append(out, map[string]any{
				"noteId": id, "modelName": n.modelName, "tags": tags,
				"fields": fields, "cards": []int{id * 10},
			})
		}
		return out

	case "deleteNotes":
		var p struct {
			Notes []int `json:"notes"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			t.Errorf("stub: deleteNotes params are not the documented shape: %v\ngot: %s", err, params)
			return nil
		}
		for _, id := range p.Notes {
			delete(c.notes, id)
		}
		return nil

	default:
		return nil
	}
}

// migrateFixtureRun performs the setup every migration end-to-end test needs:
// one ordinary `sync` that adds a stock-Basic note, leaving a registry
// recording it and a collection holding it. That is what a pre-SPEC
// collection looks like, and building it through the real sync path rather
// than by hand is what makes the migration a genuine round trip.
func migrateFixtureRun(t *testing.T, rec *collectionRecorder, url, registryPath string) int {
	t.Helper()
	entry := protocol.Entry{
		Key: "a.org::0", NoteID: nil, NoteType: "Basic",
		SourcePath: "a.org", Title: "Capital of France", Body: "Paris.",
	}
	var stdout, stderr strings.Builder
	if code := run([]string{"sync"},
		strings.NewReader(syncRequest(t, url, registryPath, nil, []protocol.Entry{entry})),
		&stdout, &stderr); code != 0 {
		t.Fatalf("fixture sync: exit %d\nstderr: %s", code, stderr.String())
	}
	resp := decodeResponse(t, stdout.String())
	if len(resp.Results) != 1 || resp.Results[0].NoteID == nil {
		t.Fatalf("fixture sync: results = %+v, want one carrying a note id", resp.Results)
	}
	return *resp.Results[0].NoteID
}

// migrateEntry is the same heading the fixture sync added, now carrying the
// identifier that sync assigned it — the shape the front end sends on the
// migrate run.
func migrateEntry(noteID int) protocol.Entry {
	return protocol.Entry{
		Key: "a.org::0", NoteID: &noteID, NoteType: "Basic",
		SourcePath: "a.org", Title: "Capital of France", Body: "Paris.",
	}
}

// AC-C-018a + AC-C-018b end to end: `migrate --dry-run` reads the sync
// request document, answers with the sync response schema, reports the
// candidate through results[], and issues no write action on the wire.
func TestMigrateDryRunEndToEnd_ReportsTheCandidateAndWritesNothing(t *testing.T) {
	rec := newCollectionRecorder()
	srv := rec.server(t)
	registryPath := filepath.Join(t.TempDir(), "registry.json")

	noteID := migrateFixtureRun(t, rec, srv.URL, registryPath)
	before, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("reading the registry the fixture wrote: %v", err)
	}
	rec.mu.Lock()
	rec.actions = nil
	rec.mu.Unlock()

	var stdout, stderr strings.Builder
	code := run([]string{"migrate", "--dry-run"},
		strings.NewReader(syncRequest(t, srv.URL, registryPath, nil, []protocol.Entry{migrateEntry(noteID)})),
		&stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, want 0\nstderr: %s", code, stderr.String())
	}

	resp := decodeResponse(t, stdout.String())
	if resp.ProtocolVersion != protocol.Version {
		t.Errorf("response protocol_version = %d, want %d", resp.ProtocolVersion, protocol.Version)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results = %+v, want exactly one candidate", resp.Results)
	}
	if resp.Results[0].Action != protocol.ActionMigrateCandidate {
		t.Errorf("action = %q, want %q", resp.Results[0].Action, protocol.ActionMigrateCandidate)
	}
	if resp.Results[0].NoteID == nil || *resp.Results[0].NoteID != noteID {
		t.Errorf("note_id = %v, want the existing identifier %d", resp.Results[0].NoteID, noteID)
	}

	// AC-C-018b: every write action, counted on the wire itself. The
	// handshake read is permitted and is why this is a write allowlist
	// rather than an emptiness check on the whole log.
	for _, action := range rec.snapshotActions() {
		switch action {
		case "addNote", "deleteNotes", "updateNoteFields", "updateNoteTags",
			"changeDeck", "createDeck", "storeMediaFile",
			"createModel", "updateModelStyling", "updateModelTemplates":
			t.Errorf("dry run issued the write action %q; the recorded log was %v", action, rec.snapshotActions())
		}
	}

	after, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("reading the registry after the dry run: %v", err)
	}
	if string(after) != string(before) {
		t.Errorf("dry run rewrote the registry:\n before = %s\n  after = %s", before, after)
	}
}

// AC-C-019a + AC-C-019b end to end: the writing run adds under the
// counterpart type BEFORE deleting the original, and the persisted registry
// carries the new identifier and the imoogi-owned type.
func TestMigrateEndToEnd_AddsUnderTheCounterpartThenDeletesTheOriginal(t *testing.T) {
	rec := newCollectionRecorder()
	srv := rec.server(t)
	registryPath := filepath.Join(t.TempDir(), "registry.json")

	noteID := migrateFixtureRun(t, rec, srv.URL, registryPath)
	rec.mu.Lock()
	rec.actions = nil
	rec.mu.Unlock()

	var stdout, stderr strings.Builder
	code := run([]string{"migrate"},
		strings.NewReader(syncRequest(t, srv.URL, registryPath, nil, []protocol.Entry{migrateEntry(noteID)})),
		&stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, want 0\nstderr: %s", code, stderr.String())
	}

	resp := decodeResponse(t, stdout.String())
	if len(resp.Errors) != 0 {
		t.Fatalf("errors = %+v, want none", resp.Errors)
	}
	if len(resp.Results) != 1 || resp.Results[0].Action != protocol.ActionAdded {
		t.Fatalf("results = %+v, want one `added` result", resp.Results)
	}
	newID := resp.Results[0].NoteID
	if newID == nil || *newID == noteID {
		t.Fatalf("note_id = %v, want a NEW identifier for the front end to write back", newID)
	}

	actions := rec.snapshotActions()
	addAt, delAt := indexOfAction(actions, "addNote"), indexOfAction(actions, "deleteNotes")
	if addAt < 0 || delAt < 0 {
		t.Fatalf("actions = %v, want both an addNote and a deleteNotes", actions)
	}
	if addAt > delAt {
		t.Errorf("actions = %v — the delete preceded the add; design.md §7.2 requires add-before-delete", actions)
	}

	// The collection now holds exactly the replacement, under the
	// counterpart type: the original was deleted, and only after the add.
	if ids := rec.noteIDs(); len(ids) != 1 || ids[0] != *newID {
		t.Errorf("collection holds %v, want exactly the new note %d", ids, *newID)
	}
	rec.mu.Lock()
	replacement := rec.notes[*newID]
	rec.mu.Unlock()
	if replacement == nil || replacement.modelName != "imoogi-Basic" {
		t.Errorf("replacement note = %+v, want it under imoogi-Basic", replacement)
	}

	// The registry the run PERSISTED names the new identifier and type —
	// the dry-run test asserts the complementary no-write, so between them
	// the save is attributed to the writing run alone.
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("reading the persisted registry: %v", err)
	}
	if !strings.Contains(string(raw), "imoogi-Basic") {
		t.Errorf("persisted registry does not record imoogi-Basic:\n%s", raw)
	}
	if strings.Contains(string(raw), `"note_id":`+strconv.Itoa(noteID)) {
		t.Errorf("persisted registry still names the original identifier %d:\n%s", noteID, raw)
	}
}

// A mistyped flag is a usage error, not an argument the writing path
// silently absorbs — `migrate --dryrun` must NOT migrate the collection.
func TestMigrateRejectsAnUnknownFlagRatherThanMigrating(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run([]string{"migrate", "--dryrun"}, strings.NewReader("{}"), &stdout, &stderr)

	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if stdout.String() != "" {
		t.Errorf("stdout = %q, want empty — a usage error writes no response document", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--dryrun") {
		t.Errorf("stderr = %q, want it to name the offending flag", stderr.String())
	}
}

func indexOfAction(haystack []string, needle string) int {
	for i, s := range haystack {
		if s == needle {
			return i
		}
	}
	return -1
}
