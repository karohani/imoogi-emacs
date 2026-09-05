package model

import (
	"context"
	"fmt"
	"io"

	"github.com/karohani/imoogi-emacs/internal/anki/ankiconnect"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// OwnershipNotice is the message REQ-C-002.3 requires the install step to
// emit BEFORE its first model-write request of an invocation: imoogi owns
// these two types' styling outright, and a hand edit made inside Anki is
// replaced rather than merged.
//
// It is emitted once per invocation on the announcement writer, never on
// stdout — stdout carries the response document and nothing else.
const OwnershipNotice = "imoogi owns the styling and card templates of imoogi-Basic and imoogi-Cloze outright. " +
	"This step will replace each type's entire CSS and templates; any hand edit made to them inside Anki is discarded.\n"

// Install is the install step (REQ-C-002, design.md §6): probe modelNames
// once, then per imoogi-owned type create it if absent or update its styling
// and templates if present.
//
// Probe-then-act, not create-and-catch. Whether createModel on an existing
// name errors, no-ops, or overwrites is undocumented; probing first means
// that branch is never reached, so the unknown is unreachable rather than
// merely unlikely.
//
// It returns one Result per type it installed — key = model name, action =
// added or updated, note_id nil — and one Error per type it could not, keyed
// by that type's name. No new wire shape is introduced: this is the existing
// Response's own vocabulary, which is what keeps REQ-C-018's wire-stability
// clause true.
//
// A failed probe is run-level: no type's branch is decidable, so nothing is
// attempted and the single error carries a nil key. It is ankiconnect_error
// rather than model_install_failed because no model install was attempted.
//
// announce may be nil, which suppresses the notice; every other caller passes
// the process's stderr.
func Install(ctx context.Context, client ankiconnect.AnkiConnector, userCSS string, announce io.Writer) ([]protocol.Result, []protocol.Error) {
	results := []protocol.Result{}
	errs := []protocol.Error{}

	existing, err := client.ModelNames(ctx)
	if err != nil {
		return nil, append(errs, protocol.Error{
			Code:    protocol.CodeAnkiConnectError,
			Message: fmt.Sprintf("note-type probe failed: %v", err),
			Key:     nil,
		})
	}

	present := make(map[string]bool, len(existing))
	for _, name := range existing {
		present[name] = true
	}

	css := UploadCSS(userCSS)
	announced := false

	for _, spec := range Owned() {
		// The notice precedes the FIRST model-write request of the
		// invocation, not each one — announcing per type would say the same
		// thing twice. It is emitted inside the loop rather than before it so
		// that a run which writes nothing (a failed probe returns above)
		// never claims to be about to write.
		if !announced {
			if announce != nil {
				_, _ = io.WriteString(announce, OwnershipNotice)
			}
			announced = true
		}

		action, err := installOne(ctx, client, spec, css, present[spec.Name])
		if err != nil {
			key := spec.Name
			errs = append(errs, protocol.Error{
				Code:    protocol.CodeModelInstallFailed,
				Message: fmt.Sprintf("%s: %v", spec.Name, err),
				Key:     &key,
			})
			continue
		}
		key := spec.Name
		results = append(results, protocol.Result{Key: &key, Action: action, NoteID: nil})
	}

	return results, errs
}

// installOne applies one note type and reports which branch it took. The
// present branch issues styling before templates and stops on the first
// failure: the two writes are one install for one type, and a type carrying
// new CSS with old templates is a worse outcome than one left untouched.
func installOne(ctx context.Context, client ankiconnect.AnkiConnector, spec Spec, css string, alreadyPresent bool) (string, error) {
	if !alreadyPresent {
		if err := client.CreateModel(ctx, spec.Name, spec.InOrderFields, css, spec.IsCloze, spec.Templates); err != nil {
			return "", err
		}
		return protocol.ActionAdded, nil
	}
	if err := client.UpdateModelStyling(ctx, spec.Name, css); err != nil {
		return "", err
	}
	if err := client.UpdateModelTemplates(ctx, spec.Name, spec.Templates); err != nil {
		return "", err
	}
	return protocol.ActionUpdated, nil
}
