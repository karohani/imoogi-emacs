package model_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/karohani/imoogi-emacs/internal/anki/ankiconnect"
)

// fakeClient is this package's test double for ankiconnect.AnkiConnector,
// carrying one call log per action exactly as the planner's own double does
// (design.md §8.2): every negative assertion in acceptance.md is a count on a
// SPECIFIC log, which a shared log could not express.
//
// It is a second double rather than a reuse of the planner's because that one
// lives in a _test file of package planner and is unexported — Go gives no way
// to import it. The two are deliberately not factored into a shared helper:
// the planner's exists to prove the sync path never reaches these endpoints,
// this one exists to prove the install path drives them exactly once each, and
// a shared double would have to serve both intents at once.
type fakeClient struct {
	// models is the stub collection's note-type name set, seeded by a test so
	// the probe-then-act branch can be driven from either side. It is also
	// mutated by CreateModel, so a second Install in the same test observes
	// what the first one created — the shape design.md §6's probe-then-act
	// claim actually rests on.
	models []string

	modelNamesCalls           int
	createModelCalls          []createModelCall
	updateModelStylingCalls   []updateModelStylingCall
	updateModelTemplatesCalls []updateModelTemplatesCall

	// Configurable failures. modelNamesErr fails the probe outright;
	// failModel fails every write naming that model, which is how the
	// partial-failure and total-failure paths are driven.
	modelNamesErr error
	failModel     map[string]error
}

type createModelCall struct {
	name          string
	inOrderFields []string
	css           string
	isCloze       bool
	templates     []ankiconnect.CardTemplate
}

type updateModelStylingCall struct {
	name string
	css  string
}

type updateModelTemplatesCall struct {
	name      string
	templates []ankiconnect.CardTemplate
}

var _ ankiconnect.AnkiConnector = (*fakeClient)(nil)

func newFakeClient(existingModels ...string) *fakeClient {
	return &fakeClient{
		models:    append([]string(nil), existingModels...),
		failModel: map[string]error{},
	}
}

// modelWrites is the total count across the three model-WRITE logs. AC-C-003b
// and AC-C-007 both reason about "before the first model-write request", and
// the announcement-ordering test reads this at announce time.
func (f *fakeClient) modelWrites() int {
	return len(f.createModelCalls) + len(f.updateModelStylingCalls) + len(f.updateModelTemplatesCalls)
}

// writtenModelNames is every model name the three write logs name, in call
// order — the exact set AC-C-003b and AC-C-022b scope their assertion to.
func (f *fakeClient) writtenModelNames() []string {
	var names []string
	for _, c := range f.createModelCalls {
		names = append(names, c.name)
	}
	for _, c := range f.updateModelStylingCalls {
		names = append(names, c.name)
	}
	for _, c := range f.updateModelTemplatesCalls {
		names = append(names, c.name)
	}
	return names
}

func (f *fakeClient) ModelNames(ctx context.Context) ([]string, error) {
	f.modelNamesCalls++
	if f.modelNamesErr != nil {
		return nil, f.modelNamesErr
	}
	return append([]string(nil), f.models...), nil
}

func (f *fakeClient) CreateModel(ctx context.Context, name string, inOrderFields []string, css string, isCloze bool, templates []ankiconnect.CardTemplate) error {
	f.createModelCalls = append(f.createModelCalls, createModelCall{
		name:          name,
		inOrderFields: append([]string(nil), inOrderFields...),
		css:           css,
		isCloze:       isCloze,
		templates:     append([]ankiconnect.CardTemplate(nil), templates...),
	})
	if err := f.failModel[name]; err != nil {
		return err
	}
	f.models = append(f.models, name)
	return nil
}

func (f *fakeClient) UpdateModelStyling(ctx context.Context, name, css string) error {
	f.updateModelStylingCalls = append(f.updateModelStylingCalls, updateModelStylingCall{name: name, css: css})
	return f.failModel[name]
}

func (f *fakeClient) UpdateModelTemplates(ctx context.Context, name string, templates []ankiconnect.CardTemplate) error {
	f.updateModelTemplatesCalls = append(f.updateModelTemplatesCalls, updateModelTemplatesCall{
		name:      name,
		templates: append([]ankiconnect.CardTemplate(nil), templates...),
	})
	return f.failModel[name]
}

// ---- The synchronization surface, which the install step must never reach ----
//
// Every method below panics rather than recording. AC-C-003a is the sync
// path's fence and lives in the planner's tests; this is its mirror image —
// the install path touching a note, a deck, or a media file is a defect no
// count assertion would have to be written to catch, because the test simply
// fails where it happens.

func (f *fakeClient) Handshake(ctx context.Context) error { return nil }

func (f *fakeClient) DeckNames(ctx context.Context) ([]string, error) {
	panic("install step reached DeckNames")
}

func (f *fakeClient) CreateDeck(ctx context.Context, name string) error {
	panic("install step reached CreateDeck: " + name)
}

func (f *fakeClient) ModelFieldNames(ctx context.Context, modelName string) ([]string, error) {
	panic("install step reached ModelFieldNames: " + modelName)
}

func (f *fakeClient) AddNote(ctx context.Context, deck, modelName string, fields map[string]string, tags []string) (int, error) {
	panic("install step reached AddNote")
}

func (f *fakeClient) UpdateNoteFields(ctx context.Context, noteID int, fields map[string]string) error {
	panic(fmt.Sprintf("install step reached UpdateNoteFields: %d", noteID))
}

func (f *fakeClient) UpdateNoteTags(ctx context.Context, noteID int, tags []string) error {
	panic(fmt.Sprintf("install step reached UpdateNoteTags: %d", noteID))
}

func (f *fakeClient) NotesInfo(ctx context.Context, noteIDs []int) ([]ankiconnect.NoteInfo, error) {
	panic("install step reached NotesInfo")
}

func (f *fakeClient) ChangeDeck(ctx context.Context, cardIDs []int, deck string) error {
	panic("install step reached ChangeDeck: " + deck)
}

func (f *fakeClient) DeleteNotes(ctx context.Context, noteIDs []int) error {
	panic("install step reached DeleteNotes")
}

func (f *fakeClient) StoreMediaFile(ctx context.Context, filename, absPath string) (string, error) {
	panic("install step reached StoreMediaFile: " + filename)
}

var errFakeInstall = errors.New("fake: simulated AnkiConnect failure")
