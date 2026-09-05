package planner

import (
	"context"
	"errors"

	"github.com/karohani/imoogi-emacs/internal/anki/ankiconnect"
)

// fakeClient is a test double for ankiconnect.AnkiConnector, standing in
// for the recording httptest server acceptance.md's harness assumptions
// describe ("Criteria mentioning 'the AnkiConnect stub' are checked against
// a recording httptest server ... so that request sets can be asserted
// exactly"). A plain in-memory fake gives the same call-order and
// exact-request-set assertions without a real HTTP round trip, which
// acceptance.md explicitly leaves as the implementer's choice.
type fakeClient struct {
	// decks is the stub collection's existing deck set, seeded by the test
	// before Run() so AC-009-style "no deck-create request" assertions are
	// possible (acceptance.md's own harness note: "the stub's collection
	// starts containing the configured default deck, so that a deck-create
	// request never fires incidentally").
	decks map[string]bool

	// notes is the stub collection's existing notes, keyed by note ID.
	// Seeded directly by seedNote for delete-path tests, or populated by
	// AddNote during a run.
	notes  map[int]*fakeNote
	nextID int

	// Call logs, one per AnkiConnect action, in call order.
	addCalls          []addCall
	createDeckCalls   []string
	updateFieldsCalls []updateFieldsCall
	updateTagsCalls   []updateTagsCall
	changeDeckCalls   []changeDeckCall
	deleteCalls       [][]int
	notesInfoCalls    [][]int

	// Configurable failures for the suppression-path tests (AC-022).
	notesInfoErr   error
	deleteNotesErr error

	// modelFields is the stub collection's note-type -> field-name map. A
	// real Anki profile's field names are whatever the user made them, and
	// a customized "Basic" carrying lowercase front/back is the observed
	// case this exists to reproduce. Unset entries fall back to the Anki
	// stock names, so every pre-existing test keeps its old behavior.
	modelFields    map[string][]string
	modelFieldsErr error
}

type fakeNote struct {
	modelName string
	fields    map[string]string
	tags      []string
	cards     []int
}

type addCall struct {
	deck      string
	modelName string
	fields    map[string]string
	tags      []string
}

type updateFieldsCall struct {
	noteID int
	fields map[string]string
}

type updateTagsCall struct {
	noteID int
	tags   []string
}

type changeDeckCall struct {
	cardIDs []int
	deck    string
}

var _ ankiconnect.AnkiConnector = (*fakeClient)(nil)

func newFakeClient() *fakeClient {
	return &fakeClient{
		decks: map[string]bool{},
		notes: map[int]*fakeNote{},
	}
}

// seedNote plants a note directly in the stub collection, bypassing AddNote
// — the shape delete-path tests need: a registry already claims the note,
// and the stub already holds it, with no add/update request in between.
func (f *fakeClient) seedNote(id int, modelName string, fields map[string]string, tags []string, cards []int) {
	f.notes[id] = &fakeNote{modelName: modelName, fields: cloneStringMap(fields), tags: cloneStringSlice(tags), cards: cards}
}

func (f *fakeClient) Handshake(ctx context.Context) error { return nil }

// ModelFieldNames answers with the seeded per-model field names, falling
// back to Anki's stock names for a model the test did not customize.
func (f *fakeClient) ModelFieldNames(ctx context.Context, modelName string) ([]string, error) {
	if f.modelFieldsErr != nil {
		return nil, f.modelFieldsErr
	}
	if names, ok := f.modelFields[modelName]; ok {
		return append([]string(nil), names...), nil
	}
	switch modelName {
	case "Basic":
		return []string{"Front", "Back"}, nil
	case "Cloze":
		return []string{"Text", "Extra"}, nil
	default:
		return nil, errors.New("fake: unknown model " + modelName)
	}
}

func (f *fakeClient) DeckNames(ctx context.Context) ([]string, error) {
	names := make([]string, 0, len(f.decks))
	for d := range f.decks {
		names = append(names, d)
	}
	return names, nil
}

func (f *fakeClient) CreateDeck(ctx context.Context, name string) error {
	f.createDeckCalls = append(f.createDeckCalls, name)
	f.decks[name] = true
	return nil
}

func (f *fakeClient) AddNote(ctx context.Context, deck, modelName string, fields map[string]string, tags []string) (int, error) {
	f.nextID++
	id := f.nextID
	f.addCalls = append(f.addCalls, addCall{deck: deck, modelName: modelName, fields: cloneStringMap(fields), tags: cloneStringSlice(tags)})
	f.notes[id] = &fakeNote{
		modelName: modelName,
		fields:    cloneStringMap(fields),
		tags:      cloneStringSlice(tags),
		cards:     []int{id*1000 + 1},
	}
	return id, nil
}

func (f *fakeClient) UpdateNoteFields(ctx context.Context, noteID int, fields map[string]string) error {
	f.updateFieldsCalls = append(f.updateFieldsCalls, updateFieldsCall{noteID: noteID, fields: cloneStringMap(fields)})
	if n, ok := f.notes[noteID]; ok {
		n.fields = cloneStringMap(fields)
	}
	return nil
}

func (f *fakeClient) UpdateNoteTags(ctx context.Context, noteID int, tags []string) error {
	f.updateTagsCalls = append(f.updateTagsCalls, updateTagsCall{noteID: noteID, tags: cloneStringSlice(tags)})
	if n, ok := f.notes[noteID]; ok {
		n.tags = cloneStringSlice(tags)
	}
	return nil
}

func (f *fakeClient) NotesInfo(ctx context.Context, noteIDs []int) ([]ankiconnect.NoteInfo, error) {
	f.notesInfoCalls = append(f.notesInfoCalls, append([]int(nil), noteIDs...))
	if f.notesInfoErr != nil {
		return nil, f.notesInfoErr
	}
	out := make([]ankiconnect.NoteInfo, len(noteIDs))
	for i, id := range noteIDs {
		n, ok := f.notes[id]
		if !ok {
			out[i] = ankiconnect.NoteInfo{NoteID: id, Exists: false}
			continue
		}
		fv := make(map[string]ankiconnect.FieldValue, len(n.fields))
		order := 0
		for name, val := range n.fields {
			fv[name] = ankiconnect.FieldValue{Value: val, Order: order}
			order++
		}
		out[i] = ankiconnect.NoteInfo{
			NoteID:    id,
			Exists:    true,
			ModelName: n.modelName,
			Fields:    fv,
			Tags:      cloneStringSlice(n.tags),
			Cards:     append([]int(nil), n.cards...),
		}
	}
	return out, nil
}

func (f *fakeClient) ChangeDeck(ctx context.Context, cardIDs []int, deck string) error {
	f.changeDeckCalls = append(f.changeDeckCalls, changeDeckCall{cardIDs: append([]int(nil), cardIDs...), deck: deck})
	return nil
}

func (f *fakeClient) DeleteNotes(ctx context.Context, noteIDs []int) error {
	f.deleteCalls = append(f.deleteCalls, append([]int(nil), noteIDs...))
	if f.deleteNotesErr != nil {
		return f.deleteNotesErr
	}
	for _, id := range noteIDs {
		delete(f.notes, id)
	}
	return nil
}

var errFakeTransport = errors.New("fake: simulated transport failure")

func cloneStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}
