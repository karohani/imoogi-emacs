package planner

import (
	"context"
	"fmt"

	"github.com/karohani/imoogi-emacs/internal/anki/ankiconnect"
	"github.com/karohani/imoogi-emacs/internal/anki/hashing"
	"github.com/karohani/imoogi-emacs/internal/anki/media"
	"github.com/karohani/imoogi-emacs/internal/anki/model"
	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// Migrate re-homes registry entries recorded on a stock note type onto that
// type's imoogi- counterpart — design.md §7.1 steps 5-10, spec.md REQ-C-020.
//
// It reads the SAME request document `sync` reads and writes the SAME
// response schema, which is REQ-C-018's wire-stability clause: no existing
// document gains a field, and the only addition anywhere is one new `action`
// VALUE, emitted by the dry run alone.
//
// dryRun is the consent gate's mechanism (REQ-C-019). In dry-run mode this
// function issues NO AnkiConnect request of any kind and mutates nothing: it
// reads the registry, reports one `migrate_candidate` per candidate carrying
// that candidate's EXISTING identifier, and returns. The front end's prompt
// needs a count before it can ask an informed question, and this is how the
// count is obtained without acting on the answer it has not received yet.
//
// Like Run, it mutates reg in place and performs no file I/O — persisting is
// the caller's job, and the caller deliberately does NOT persist after a dry
// run (AC-C-018c: the registry is byte-unchanged).
//
// What it deliberately does NOT do is the orphan pass. Census reconciliation
// and orphan deletion belong to `sync` (design.md §3 steps 10-11); §7.1's
// migrate sequence is steps 5-10 and nothing else. Keeping the orphan pass
// out is also what makes the dry run's zero-write guarantee structural
// rather than incidental.
func Migrate(ctx context.Context, req protocol.Request, reg *registry.Registry, client ankiconnect.AnkiConnector, dryRun bool) ([]protocol.Result, []protocol.Error) {
	r := &runner{
		ctx:         ctx,
		client:      client,
		reg:         reg,
		defaultDeck: req.Config.DefaultDeck,
		syncRoot:    expandTilde(req.Config.SyncRoot),
	}

	var results []protocol.Result
	var errs []protocol.Error

	for _, entry := range req.Entries {
		counterpart, isCandidate := migrationTarget(entry, reg)
		if !isCandidate {
			res, err := nonCandidate(entry, reg)
			if res != nil {
				results = append(results, *res)
			}
			if err != nil {
				errs = append(errs, *err)
			}
			continue
		}

		key := entry.Key
		noteID := *entry.NoteID

		if dryRun {
			// design.md §5.1: the existing identifier, not a new one —
			// nothing has been created, and the front end counts this
			// result rather than writing anything back from it.
			id := noteID
			results = append(results, protocol.Result{
				Key:    &key,
				Action: protocol.ActionMigrateCandidate,
				NoteID: &id,
			})
			continue
		}

		res, entryErrs := r.migrateOne(entry, counterpart)
		if res != nil {
			results = append(results, *res)
		}
		errs = append(errs, entryErrs...)
	}

	return results, errs
}

// migrationTarget applies REQ-C-020's gate and returns the counterpart type
// to re-home onto. Both halves must hold: the registry's RECORDED type is a
// stock type, and the DECLARED type is either that same stock type or its
// imoogi- counterpart.
//
// The declared-type half accepts the counterpart as well as the stock name
// so a re-run converges. A migration whose Go side succeeded but whose Elisp
// write-back did not finish leaves headings already declaring the
// counterpart while the registry still records the stock type; without this
// arm, the retry would fall into REQ-C-021's skip and the entry could never
// be repaired.
func migrationTarget(entry protocol.Entry, reg *registry.Registry) (string, bool) {
	if entry.NoteID == nil {
		return "", false
	}
	regEntry, found := reg.Lookup(*entry.NoteID)
	if !found {
		return "", false
	}
	counterpart, recordedIsStock := stockCounterpart(regEntry.NoteType)
	if !recordedIsStock {
		return "", false
	}
	if entry.NoteType != regEntry.NoteType && entry.NoteType != counterpart {
		return "", false
	}
	return counterpart, true
}

// stockCounterpart maps a stock note type to the imoogi-owned type that
// re-homes it, reporting whether the name was a stock type at all. It is the
// candidate-set predicate of design.md §7.5: every registry entry recording
// a stock type is a candidate, and no other entry is.
func stockCounterpart(recorded string) (string, bool) {
	switch recorded {
	case orgdoc.NoteTypeBasic:
		return model.BasicName, true
	case orgdoc.NoteTypeCloze:
		return model.ClozeName, true
	default:
		return "", false
	}
}

// nonCandidate decides what a non-candidate entry is reported as — the
// residue REQ-C-021 governs (design.md §7.4).
//
// An entry whose registry-recorded type DIFFERS from its declared type and
// which the gate above rejected is a genuine note-type mismatch: it is
// skipped with note_type_change_unsupported, exactly as an ordinary sync
// would report it. Everything else — a heading with no identifier, an
// identifier the registry does not hold, or an entry already recorded on an
// imoogi-owned type — has nothing to migrate and nothing to complain about,
// so it is reported not at all. Reporting it as a skip would inflate the
// dry run's count with entries that were never candidates.
func nonCandidate(entry protocol.Entry, reg *registry.Registry) (*protocol.Result, *protocol.Error) {
	if entry.NoteID == nil {
		return nil, nil
	}
	regEntry, found := reg.Lookup(*entry.NoteID)
	if !found || regEntry.NoteType == entry.NoteType {
		return nil, nil
	}

	key := entry.Key
	noteID := *entry.NoteID
	return &protocol.Result{Key: &key, Action: protocol.ActionSkipped, NoteID: &noteID},
		&protocol.Error{
			Code: protocol.CodeNoteTypeChangeUnsupported,
			Message: fmt.Sprintf(
				"note %d: registry-recorded note type %q differs from declared %q, and %q is not its imoogi- counterpart",
				noteID, regEntry.NoteType, entry.NoteType, entry.NoteType),
			Key: &key,
		}
}

// migrateOne executes design.md §7.1 steps 6-10 for one confirmed candidate:
// render under the counterpart type, add, delete the original only after the
// add succeeded, and replace the registry record.
//
// Every early return leaves the collection and the registry exactly as they
// were and reports the entry as skipped, which is REQ-C-022's invariant: no
// entry ever reaches a state where the registry names an identifier the
// collection no longer holds. Add-before-delete is what makes that hold for
// free rather than by careful unwinding — when the add fails, the delete is
// simply never reached (design.md §7.2).
func (r *runner) migrateOne(entry protocol.Entry, counterpart string) (*protocol.Result, []protocol.Error) {
	key := entry.Key
	noteID := *entry.NoteID
	resolvedDeck := resolveDeck(entry.Deck, r.defaultDeck)
	skipped := &protocol.Result{Key: &key, Action: protocol.ActionSkipped, NoteID: &noteID}

	// The full §2 pipeline, in the order the sync path runs it: render,
	// then media, then hash. Rendering under the counterpart is rendering
	// under its stock mirror (renderType), because the two carry the same
	// field names by REQ-C-001.2 — the counterpart is what the note is
	// ADDED under, not a second output shape.
	fields, err := orgdoc.Render(renderType(counterpart), entry.Title, entry.Body)
	if err != nil {
		code := protocol.CodeOrgParseError
		if _, ok := err.(*orgdoc.ClozeMarkerMissingError); ok {
			code = protocol.CodeClozeMarkerMissing
		}
		return skipped, []protocol.Error{{Code: code, Message: err.Error(), Key: &key}}
	}

	fields, uploads, mediaErr := media.Rewrite(mediaBaseDir(r.syncRoot, entry.SourcePath), r.syncRoot, fields)
	if mediaErr != nil {
		return skipped, []protocol.Error{{Code: protocol.CodeMediaFileNotFound, Message: mediaErr.Error(), Key: &key}}
	}

	// Hashed under the COUNTERPART, because that is the type the registry
	// is about to record. Note type is a hash input (design.md §2.4), so a
	// hash taken under the stock name would mismatch on the very next
	// ordinary sync and reissue an update for a note nothing changed on.
	contentHash := hashing.Hash(counterpart, fields, resolvedDeck, entry.Tags)

	resolved, fieldErr := r.resolveFields(counterpart, fields, key)
	if fieldErr != nil {
		return skipped, []protocol.Error{*fieldErr}
	}
	if upErr := r.uploadMedia(uploads, key); upErr != nil {
		return skipped, []protocol.Error{*upErr}
	}
	if deckErr := r.ensureDeck(resolvedDeck); deckErr != nil {
		deckErr.Key = &key
		return skipped, []protocol.Error{*deckErr}
	}

	newID, addErr := r.client.AddNote(r.ctx, resolvedDeck, counterpart, resolved, entry.Tags)
	if addErr != nil {
		// REQ-C-022: the original note, its registry entry, and its heading
		// properties are all untouched, and the run continues.
		return skipped, []protocol.Error{{
			Code:    protocol.CodeMigrationAddFailed,
			Message: fmt.Sprintf("note %d: adding its %s replacement failed: %v", noteID, counterpart, addErr),
			Key:     &key,
		}}
	}

	// The original goes through the EXISTING ownership-confirmed delete
	// path (REQ-C-020.1) rather than a bare deleteNotes: one notesInfo
	// confirms the note's type and rendered content still match what the
	// registry recorded, and a note that fails that check is retained with
	// delete_candidate_unowned. Running it BEFORE the registry swap is
	// load-bearing — the check reads the OLD entry's hash, which the swap
	// is about to replace.
	//
	// Its `deleted` results are dropped: this entry already has a result
	// carrying the new identifier, and a second one naming the old note
	// would be counted by the front end as a separate outcome. Its errors
	// pass through, so a failed or unowned delete is reported.
	_, delErrs := r.confirmAndDelete([]int{noteID}, true)
	for i := range delErrs {
		delErrs[i].Key = &key
	}

	// The swap happens whether or not the delete landed. A surviving
	// original is a visible duplicate the next ordinary sync collects as an
	// orphan; abandoning the swap instead would leave the registry naming
	// the OLD identifier while the new note exists unrecorded — the one
	// state REQ-C-022 forbids, arrived at from the other direction.
	r.reg.Remove(noteID)
	r.reg.Put(registry.Entry{
		NoteID:       newID,
		SourcePath:   entry.SourcePath,
		NoteType:     counterpart,
		ResolvedDeck: resolvedDeck,
		ContentHash:  contentHash,
	})

	// `added`, not a migration-specific value: design.md §5.1 adds exactly
	// one action value and scopes it to the dry run. The front end's
	// existing write-back already acts on `added` by writing the reported
	// identifier into ANKI_NOTE_ID, which is precisely REQ-C-020.3's first
	// half; the ANKI_NOTE_TYPE half is derived locally there.
	return &protocol.Result{Key: &key, Action: protocol.ActionAdded, NoteID: &newID}, delErrs
}
