package planner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/hashing"
	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// SPEC-ANKICARD-002 AC-OPT-012c: an entry carrying no card option issues no
// request this SPEC caused, and reports exactly what it reported before.
//
// The two logs below are recorded at the SPEC's base commit and compared
// against thereafter. As with the renderer golden, recording is gated behind
// an environment variable rather than an "update if missing" fallback, so the
// run that breaks the baseline cannot also re-bless it:
//
//	UPDATE_PLANNER_GOLDEN=1 go test ./internal/anki/planner -run TestOptionFreeRequestLogGolden
//
// The empty errors slice is the load-bearing half of the no-op fixture. A
// rejected entry is ALSO reported as `skipped`, so the action alone cannot
// distinguish an untouched entry from one this SPEC's gate rejected — only
// the absence of an error can.

const plannerGoldenPath = "testdata/request-log-golden.json"

// requestLog is the serializable projection of everything the fake client was
// asked to do, plus the run's own reported outcome. Unexported call structs
// carry no JSON tags, so each is flattened here into an exported shape;
// map keys serialize in sorted order, which is what makes the comparison a
// byte comparison rather than a set comparison.
type requestLog struct {
	WriteSequence []string            `json:"write_sequence"`
	AddCalls      []loggedAdd         `json:"add_calls"`
	CreateDecks   []string            `json:"create_deck_calls"`
	UpdateFields  []loggedUpdateField `json:"update_fields_calls"`
	UpdateTags    []loggedUpdateTags  `json:"update_tags_calls"`
	ChangeDecks   []loggedChangeDeck  `json:"change_deck_calls"`
	DeleteCalls   [][]int             `json:"delete_calls"`
	NotesInfo     [][]int             `json:"notes_info_calls"`
	StoreMedia    []loggedStoreMedia  `json:"store_media_calls"`
	Results       []protocol.Result   `json:"results"`
	Errors        []protocol.Error    `json:"errors"`
}

type loggedAdd struct {
	Deck      string            `json:"deck"`
	ModelName string            `json:"model_name"`
	Fields    map[string]string `json:"fields"`
	Tags      []string          `json:"tags"`
}

type loggedUpdateField struct {
	NoteID int               `json:"note_id"`
	Fields map[string]string `json:"fields"`
}

type loggedUpdateTags struct {
	NoteID int      `json:"note_id"`
	Tags   []string `json:"tags"`
}

type loggedChangeDeck struct {
	CardIDs []int  `json:"card_ids"`
	Deck    string `json:"deck"`
}

type loggedStoreMedia struct {
	Filename string `json:"filename"`
	Path     string `json:"path"`
}

func captureRequestLog(c *fakeClient, results []protocol.Result, errs []protocol.Error) requestLog {
	log := requestLog{
		WriteSequence: c.writeSequence,
		CreateDecks:   c.createDeckCalls,
		DeleteCalls:   c.deleteCalls,
		NotesInfo:     c.notesInfoCalls,
		Results:       results,
		Errors:        errs,
	}
	for _, a := range c.addCalls {
		log.AddCalls = append(log.AddCalls, loggedAdd{Deck: a.deck, ModelName: a.modelName, Fields: a.fields, Tags: a.tags})
	}
	for _, u := range c.updateFieldsCalls {
		log.UpdateFields = append(log.UpdateFields, loggedUpdateField{NoteID: u.noteID, Fields: u.fields})
	}
	for _, u := range c.updateTagsCalls {
		log.UpdateTags = append(log.UpdateTags, loggedUpdateTags{NoteID: u.noteID, Tags: u.tags})
	}
	for _, d := range c.changeDeckCalls {
		log.ChangeDecks = append(log.ChangeDecks, loggedChangeDeck{CardIDs: d.cardIDs, Deck: d.deck})
	}
	for _, m := range c.storeMediaFileCalls {
		log.StoreMedia = append(log.StoreMedia, loggedStoreMedia{Filename: m.filename, Path: m.path})
	}
	return log
}

// optionFreeNoOpFixture is AC-OPT-012c's own fixture: every entry's recorded
// hash already matches what the run recomputes, so every entry takes the
// no-op branch and no AnkiConnect request is issued at all.
func optionFreeNoOpFixture(t *testing.T) (protocol.Request, *registry.Registry, *fakeClient) {
	t.Helper()
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	entries := []protocol.Entry{
		{
			Key: "a.org::0", NoteID: intPtr(101), NoteType: "imoogi-Basic", SourcePath: "a.org",
			Deck: nil, Tags: []string{"geo"}, Title: "Capital of France", Body: "Paris.",
		},
		{
			Key: "a.org::1", NoteID: intPtr(102), NoteType: "imoogi-Cloze", SourcePath: "a.org",
			Deck: strPtr("World"), Tags: nil, Title: "The capital of {{c1::France}}.", Body: "Worth knowing.",
		},
	}
	decks := []string{"Inbox", "World"}
	for i, e := range entries {
		client.decks[decks[i]] = true
		client.seedNote(*e.NoteID, e.NoteType, map[string]string{"Front": "x"}, e.Tags, []int{*e.NoteID * 10})
		// processEntry RENDERS under renderType but HASHES under the
		// DECLARED name — the note type is a hash input, and the two
		// imoogi-owned names must hash differently from their stock
		// counterparts. Reproducing both halves here is what makes the
		// recorded hash a genuine "this was the old state" rather than a
		// value that merely happens to differ.
		fields, err := orgdoc.Render(renderType(e.NoteType), e.Title, e.Body)
		if err != nil {
			t.Fatalf("seeding the no-op fixture: %v", err)
		}
		reg.Put(registry.Entry{
			NoteID:       *e.NoteID,
			SourcePath:   e.SourcePath,
			NoteType:     e.NoteType,
			ResolvedDeck: decks[i],
			ContentHash:  hashing.Hash(e.NoteType, fields, decks[i], e.Tags),
		})
	}

	return protocol.Request{
		Config: baseConfig(),
		Census: []protocol.CensusEntry{
			{NoteID: 101, SourcePath: "a.org"},
			{NoteID: 102, SourcePath: "a.org"},
		},
		Entries: entries,
	}, reg, client
}

// optionFreeMixedFixture exercises the branches the no-op fixture cannot: an
// add (no identifier at all) and an update (recorded hash differs). Its log is
// the substantive half of the non-interference baseline — a fixture that
// issues no request would be byte-identical to itself for trivial reasons.
func optionFreeMixedFixture(t *testing.T) (protocol.Request, *registry.Registry, *fakeClient) {
	t.Helper()
	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	client.seedNote(201, "imoogi-Cloze", map[string]string{"Text": "stale"}, nil, []int{2010})
	reg.Put(registry.Entry{
		NoteID:       201,
		SourcePath:   "b.org",
		NoteType:     "imoogi-Cloze",
		ResolvedDeck: "Inbox",
		ContentHash:  "a-hash-that-no-longer-matches",
	})

	return protocol.Request{
			Config: baseConfig(),
			Census: []protocol.CensusEntry{{NoteID: 201, SourcePath: "b.org"}},
			Entries: []protocol.Entry{
				{
					Key: "b.org::0", NoteID: nil, NoteType: "imoogi-Basic", SourcePath: "b.org",
					Deck: nil, Tags: []string{"new", "alpha"}, Title: "A brand new heading", Body: "With a body.",
				},
				{
					Key: "b.org::1", NoteID: intPtr(201), NoteType: "imoogi-Cloze", SourcePath: "b.org",
					Deck: nil, Tags: nil, Title: "Edited {{c1::cloze}} heading", Body: "New body text.",
				},
			},
		},
		reg, client
}

func TestOptionFreeRequestLogGolden(t *testing.T) {
	produced := map[string]requestLog{}

	for name, build := range map[string]func(*testing.T) (protocol.Request, *registry.Registry, *fakeClient){
		"no_op": optionFreeNoOpFixture,
		"mixed": optionFreeMixedFixture,
	} {
		req, reg, client := build(t)
		results, errs := Run(context.Background(), req, reg, client)
		produced[name] = captureRequestLog(client, results, errs)
	}

	// AC-OPT-012c's own assertions, checked directly rather than only through
	// the golden: for the no-op fixture every action is `skipped` carrying the
	// existing identifier, and the errors slice is EMPTY.
	noOp := produced["no_op"]
	if len(noOp.Errors) != 0 {
		t.Errorf("no-op fixture reported %d errors, want none: %+v", len(noOp.Errors), noOp.Errors)
	}
	if len(noOp.WriteSequence) != 0 {
		t.Errorf("no-op fixture issued writes %v, want none", noOp.WriteSequence)
	}
	for _, r := range noOp.Results {
		if r.Action != protocol.ActionSkipped {
			t.Errorf("no-op fixture action = %q, want %q", r.Action, protocol.ActionSkipped)
		}
		if r.NoteID == nil {
			t.Errorf("no-op fixture result carries no note identifier")
		}
	}

	if os.Getenv("UPDATE_PLANNER_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(plannerGoldenPath), 0o755); err != nil {
			t.Fatalf("creating testdata directory: %v", err)
		}
		encoded, err := json.MarshalIndent(produced, "", "  ")
		if err != nil {
			t.Fatalf("encoding golden: %v", err)
		}
		if err := os.WriteFile(plannerGoldenPath, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("writing golden: %v", err)
		}
		t.Logf("recorded %d request logs to %s", len(produced), plannerGoldenPath)
		return
	}

	raw, err := os.ReadFile(plannerGoldenPath)
	if err != nil {
		t.Fatalf("reading golden %s: %v (record it with UPDATE_PLANNER_GOLDEN=1)", plannerGoldenPath, err)
	}
	gotEncoded, err := json.MarshalIndent(produced, "", "  ")
	if err != nil {
		t.Fatalf("encoding produced logs: %v", err)
	}
	if string(append(gotEncoded, '\n')) != string(raw) {
		t.Errorf("the request log is not byte-identical to the baseline recorded at the SPEC's base commit\n got:\n%s\nwant:\n%s",
			gotEncoded, raw)
	}
}

// TestHashIsUnchangedByCardOptions is AC-OPT-012b's second half. It lives in
// the planner rather than in `hashing` because `hashing.Hash` takes no
// card-option parameter and so cannot be handed the two variants to compare —
// only the planner sees both an entry's options and the hash recorded for it.
//
// At the SPEC's base commit the two variants are constructed identically,
// because the option fields do not exist yet; the assertion still holds and
// becomes discriminating the moment M2 adds them.
func TestHashIsUnchangedByCardOptions(t *testing.T) {
	run := func(withOptions bool) string {
		reg := newTestRegistry(t)
		client := newFakeClient()
		client.decks["Inbox"] = true

		entry := protocol.Entry{
			Key: "h.org::0", NoteID: nil, NoteType: "imoogi-Cloze", SourcePath: "h.org",
			Deck: nil, Tags: []string{"z", "a"},
			Title: "A {{c1::cloze}} heading", Body: "With a body.",
		}
		applyHashProbeOptions(&entry, withOptions)

		results, errs := Run(context.Background(), protocol.Request{
			Config:  baseConfig(),
			Entries: []protocol.Entry{entry},
		}, reg, client)

		if len(errs) != 0 {
			t.Fatalf("withOptions=%v: errs = %+v, want none", withOptions, errs)
		}
		if len(results) != 1 || results[0].Action != protocol.ActionAdded {
			t.Fatalf("withOptions=%v: results = %+v, want one added", withOptions, results)
		}
		rec, found := reg.Lookup(*results[0].NoteID)
		if !found {
			t.Fatalf("withOptions=%v: the added note was not recorded in the registry", withOptions)
		}
		return rec.ContentHash
	}

	without := run(false)
	with := run(true)
	if without != with {
		t.Errorf("the recorded content hash differs with card options set\nwithout: %s\n   with: %s", without, with)
	}
}

// TestHashCallSiteTakesFourArguments is AC-OPT-012b's first half: a compiled
// call site invoking hashing.Hash with exactly the four existing arguments.
// A parameter added for a card option makes this fail to compile, which IS
// the assertion — a build-time check, not a reviewer reading a signature.
func TestHashCallSiteTakesFourArguments(t *testing.T) {
	var _ func(string, map[string]string, string, []string) string = hashing.Hash
	_ = hashing.Hash("imoogi-Cloze", map[string]string{"Text": "x"}, "Inbox", []string{"a"})
}

// applyHashProbeOptions sets the entry's card-option fields when the probe
// wants them, and is the ONE place TestHashIsUnchangedByCardOptions needs to
// change as the wire contract grows.
//
// The option set here is deliberately VALID and non-conflicting, so the entry
// passes the gate and reaches the add branch: the probe is about the hash the
// registry records, and a rejected entry records none.
//
// It must ALSO be an option with no RENDERING meaning, and that is the
// requirement SPEC-ANKICARD-003 introduced. The probe's claim is that setting a
// card option leaves the recorded hash alone; an option that changes the
// rendered field value changes the hash through it, because the rendered value
// is one of the four hash inputs and always was.
//
// `direction` and `incremental` can no longer serve. That SPEC made them
// render: they wrap the heading's title and its answer list in cloze markers,
// so the Text field genuinely differs and the hash differs with it — correctly,
// since a note whose options changed SHOULD be reported as updated. Worse for
// this probe, REQ-ML-002 rejects a multiline entry whose body carries no answer
// list, and this entry's body is a bare sentence, so the run never reached the
// add branch at all.
//
// `swift` is the remaining option with no rendering meaning: backlog card t15
// owns the card kind it selects, and until that lands the validation gate reads
// its value and nothing else consumes it. WHEN t15 GIVES SWIFT A RENDERING
// MEANING, THIS PROBE NEEDS THE SAME TREATMENT AGAIN — and at that point no
// card option will be rendering-free, so the probe will need a different shape
// rather than a different option.
//
// What the probe still asserts is unchanged and still discriminating: a
// card-option field carried on the entry does not reach `hashing.Hash` as an
// input of its own. Its compile-time half, TestHashCallSiteTakesFourArguments
// above, holds the same claim from the other side.
func applyHashProbeOptions(entry *protocol.Entry, withOptions bool) {
	if !withOptions {
		return
	}
	swift := "t"
	entry.Swift = &swift
}
