// Package planner is the pure decision layer design.md §3 steps 9-13
// describe: entries plus census plus registry become add / update / move /
// skip / delete decisions (spec.md REQ-009 … REQ-016, REQ-021, REQ-022).
//
// This is the highest-risk code in SPEC-ANKI-001. The deletion path in
// particular is identifier-based, content-confirmed, and never inferred
// from absence alone (plan.md D-9) — every branch below cites the
// requirement or design decision it implements so a reader can check this
// file against the SPEC rather than trust it.
package planner

import (
	"context"
	"fmt"
	"strings"

	"github.com/karohani/imoogi-emacs/internal/anki/ankiconnect"
	"github.com/karohani/imoogi-emacs/internal/anki/hashing"
	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// Run executes design.md §3 steps 9-11 over one request: a per-entry
// decision for every sync target (step 9), census reconciliation and
// candidate computation (step 10), and orphan confirmation plus deletion
// (step 11). reg is mutated in place — every add, update, adoption, and
// deletion is reflected in its in-memory state; the caller persists it via
// reg.Save() once Run returns (step 12 is deliberately the caller's job,
// not this function's, so Run performs no file I/O of its own and stays a
// pure decision layer over the client and registry it is handed).
func Run(ctx context.Context, req protocol.Request, reg *registry.Registry, client ankiconnect.AnkiConnector) ([]protocol.Result, []protocol.Error) {
	r := &runner{ctx: ctx, client: client, reg: reg, defaultDeck: req.Config.DefaultDeck}

	// Snapshot which identifiers the registry ALREADY held before this run
	// touches anything. A note added (or reconcile-adopted) THIS run is
	// assigned its identifier by AnkiConnect during this very call — it can
	// never appear in this run's census, because the front end's scan (and
	// therefore the census) was captured before that identifier existed.
	// Without this snapshot, step 10b would see a brand-new registry entry
	// with no matching census occurrence and misread "too new to be
	// scanned yet" as "no longer exists", deleting the note in the same
	// run that created it. Candidate eligibility is therefore restricted to
	// identifiers that predate this run (plan.md D-9 step 1's "the
	// registry's identifiers" means the registry AS OF THIS RUN'S START).
	preRunIDs := make(map[int]bool)
	for _, e := range reg.All() {
		preRunIDs[e.NoteID] = true
	}

	var results []protocol.Result
	var errs []protocol.Error

	// An identifier is a note's identity, so two headings carrying the same
	// ANKI_NOTE_ID are not two notes but ONE note claimed twice — the shape a
	// copy-pasted heading produces. Dispatching both would make them take
	// turns overwriting that one note (the later heading silently replacing
	// the earlier one's content, with zero errors reported) and, because the
	// registry keeps one hash for two differing headings, re-issue an update
	// on every later run for as long as the duplicate lives. Neither claimant
	// can be trusted, so neither is written: both fail with a code naming the
	// collision, and the registry entry stays as it was so the next run
	// reports it again. Unrelated entries are unaffected.
	claimants := make(map[int][]string)
	for _, entry := range req.Entries {
		if entry.NoteID != nil {
			claimants[*entry.NoteID] = append(claimants[*entry.NoteID], entry.Key)
		}
	}

	for _, entry := range req.Entries {
		if entry.NoteID != nil && len(claimants[*entry.NoteID]) > 1 {
			key := entry.Key
			noteID := *entry.NoteID
			results = append(results, protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: &noteID})
			errs = append(errs, protocol.Error{
				Code: protocol.CodeNoteIDDuplicated,
				Message: fmt.Sprintf("note %d is claimed by %d headings (%s); remove ANKI_NOTE_ID from all but one",
					noteID, len(claimants[noteID]), strings.Join(claimants[noteID], ", ")),
				Key: &key,
			})
			continue
		}
		res, entryErrs := r.processEntry(entry)
		if res != nil {
			results = append(results, *res)
		}
		errs = append(errs, entryErrs...)
	}

	// design.md §3 step 10a: refresh recorded source paths from the census,
	// for every registry identifier the census reports — regardless of
	// which branch above (if any) touched that identifier this run.
	reconcileSourcePaths(req.Census, reg)

	// design.md §3 step 10b: candidates = registry identifiers (as of this
	// run's start) minus every identifier the census reports anywhere,
	// minus every identifier the census reports nowhere but whose
	// (just-refreshed) recorded source path matches a current exclusion
	// pattern (plan.md D-9 step 1).
	candidates := computeCandidates(req.Census, reg, req.Config.ExcludePatterns, preRunIDs)

	// design.md §3 step 11: orphan confirmation and deletion.
	delResults, delErrs := r.confirmAndDelete(candidates, req.Config.ScanComplete)
	results = append(results, delResults...)
	errs = append(errs, delErrs...)

	return results, errs
}

// runner threads the pieces a per-entry decision needs (client, registry,
// default deck) plus a run-scoped deck-existence cache, so ensureDeck
// issues at most one deckNames query per run and never a redundant
// createDeck for a deck this run already confirmed or created
// (acceptance.md AC-009).
type runner struct {
	ctx         context.Context
	client      ankiconnect.AnkiConnector
	reg         *registry.Registry
	defaultDeck string

	decksLoaded bool
	knownDecks  map[string]bool

	// modelFields caches each note type's real field names for the run, so
	// modelFieldNames is queried at most once per note type.
	modelFields map[string][]string
}

// resolveFields maps orgdoc's canonical field names (Front/Back/Text) onto
// the note type's REAL field names before dispatch.
//
// Anki's field names are user-editable, so a profile's "Basic" may carry
// front/back rather than the stock Front/Back. The two AnkiConnect actions
// disagree about that: addNote matches names case-insensitively, so adds
// appear to work, while updateNoteFields accepts an unknown name, answers
// error:null, and changes nothing. Left unresolved, an edit is reported as
// "updated" with zero errors while Anki keeps the old text — and because
// the registry's content hash advances anyway, the next run skips the entry
// and the divergence becomes permanent with no signal anywhere.
//
// A rendered field with no counterpart in the model is an error rather than
// a dropped key: writing part of a note and calling it a success is the
// failure mode this whole function exists to remove.
func (r *runner) resolveFields(noteType string, fields map[string]string, key string) (map[string]string, *protocol.Error) {
	names, ok := r.modelFields[noteType]
	if !ok {
		fetched, err := r.client.ModelFieldNames(r.ctx, noteType)
		if err != nil {
			return nil, &protocol.Error{
				Code:    protocol.CodeAnkiConnectError,
				Message: fmt.Sprintf("note type %q: field-name lookup failed: %v", noteType, err),
				Key:     &key,
			}
		}
		if r.modelFields == nil {
			r.modelFields = map[string][]string{}
		}
		r.modelFields[noteType] = fetched
		names = fetched
	}

	byLower := make(map[string]string, len(names))
	for _, name := range names {
		byLower[strings.ToLower(name)] = name
	}

	resolved := make(map[string]string, len(fields))
	for rendered, value := range fields {
		actual, found := byLower[strings.ToLower(rendered)]
		if !found {
			return nil, &protocol.Error{
				Code: protocol.CodeNoteFieldMissing,
				Message: fmt.Sprintf(
					"note type %q has no field matching %q (its fields are %s); rename the field in Anki or use a note type that carries it",
					noteType, rendered, strings.Join(names, ", ")),
				Key: &key,
			}
		}
		resolved[actual] = value
	}
	return resolved, nil
}

// resolveDeck applies REQ-005's fallback: a nil or empty resolved deck
// selects the configured default. entries[].deck arrives already resolved
// through the front end's REQ-002 inheritance chain — this function applies
// only the last, back-end-owned fallback step.
func resolveDeck(deck *string, defaultDeck string) string {
	if deck != nil && *deck != "" {
		return *deck
	}
	return defaultDeck
}

// ensureDeck creates deck if this run has not already seen it exist,
// caching the collection's deck list on first use so a deck already
// present in the collection (acceptance.md AC-009) never provokes a
// createDeck request, while a genuinely missing deck (AC-013) does
// (spec.md REQ-005).
func (r *runner) ensureDeck(deck string) *protocol.Error {
	if !r.decksLoaded {
		names, err := r.client.DeckNames(r.ctx)
		r.knownDecks = make(map[string]bool, len(names))
		if err == nil {
			for _, n := range names {
				r.knownDecks[n] = true
			}
		}
		r.decksLoaded = true
	}
	if r.knownDecks[deck] {
		return nil
	}
	if err := r.client.CreateDeck(r.ctx, deck); err != nil {
		return &protocol.Error{Code: protocol.CodeDeckCreateFailed, Message: err.Error()}
	}
	r.knownDecks[deck] = true
	return nil
}

// processEntry implements design.md §3 step 9's five-branch partition. Every
// entry lands in exactly one branch, and the branches are mutually
// exclusive by construction: REQ-010's own text requires the registry's
// recorded note type to MATCH, which is what keeps it disjoint from
// REQ-021 (a type mismatch always implies a hash mismatch, since note type
// is itself a hash input — design.md §2.4).
func (r *runner) processEntry(entry protocol.Entry) (*protocol.Result, []protocol.Error) {
	key := entry.Key
	resolvedDeck := resolveDeck(entry.Deck, r.defaultDeck)

	fields, err := orgdoc.Render(entry.NoteType, entry.Title, entry.Body)
	if err != nil {
		if _, ok := err.(*orgdoc.ClozeMarkerMissingError); ok {
			// REQ-019 / AC-025: skip, create no note, continue processing.
			return &protocol.Result{Key: &key, Action: protocol.ActionSkipped, NoteID: entry.NoteID},
				[]protocol.Error{{Code: protocol.CodeClozeMarkerMissing, Message: err.Error(), Key: &key}}
		}
		return &protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: entry.NoteID},
			[]protocol.Error{{Code: protocol.CodeOrgParseError, Message: err.Error(), Key: &key}}
	}

	contentHash := hashing.Hash(entry.NoteType, fields, resolvedDeck, entry.Tags)

	if entry.NoteID == nil {
		// add (REQ-009): no identifier on this heading at all.
		resolved, fieldErr := r.resolveFields(entry.NoteType, fields, key)
		if fieldErr != nil {
			return &protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: nil}, []protocol.Error{*fieldErr}
		}
		return r.addNewNote(entry, resolved, resolvedDeck, contentHash, key)
	}

	noteID := *entry.NoteID
	regEntry, found := r.reg.Lookup(noteID)
	if !found {
		// reconcile (REQ-012): an identifier the registry has no record of.
		resolved, fieldErr := r.resolveFields(entry.NoteType, fields, key)
		if fieldErr != nil {
			return &protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: &noteID}, []protocol.Error{*fieldErr}
		}
		return r.reconcile(entry, resolved, resolvedDeck, contentHash, key, noteID)
	}

	if regEntry.NoteType != entry.NoteType {
		// note-type-change skip (REQ-021, plan.md D-12): update no field,
		// move no card, leave the registry entry untouched.
		return &protocol.Result{Key: &key, Action: protocol.ActionSkipped, NoteID: &noteID},
			[]protocol.Error{{
				Code: protocol.CodeNoteTypeChangeUnsupported,
				Message: fmt.Sprintf(
					"note %d: registry-recorded note type %q differs from declared %q",
					noteID, regEntry.NoteType, entry.NoteType),
				Key: &key,
			}}
	}

	if regEntry.ContentHash == contentHash {
		// no-op (REQ-011): issue no AnkiConnect request at all.
		return &protocol.Result{Key: &key, Action: protocol.ActionSkipped, NoteID: &noteID}, nil
	}

	// update (REQ-010): note type matches, hash differs. Field resolution
	// happens HERE, after the no-op check above, so REQ-011's "issue no
	// AnkiConnect request at all when nothing changed" still holds.
	resolved, fieldErr := r.resolveFields(entry.NoteType, fields, key)
	if fieldErr != nil {
		return &protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: &noteID}, []protocol.Error{*fieldErr}
	}
	return r.updateNote(entry, resolved, resolvedDeck, contentHash, key, noteID, regEntry)
}

// addNewNote implements REQ-009's add branch: ensure the deck exists
// (REQ-005), create the note, and record it in the registry (REQ-008). Also
// the fallback path for REQ-012's reconciliation when the identifier does
// not resolve to an owned, type-matching note in the collection — in that
// case the STALE identifier is simply discarded; a fresh one is assigned.
func (r *runner) addNewNote(entry protocol.Entry, fields map[string]string, resolvedDeck, contentHash, key string) (*protocol.Result, []protocol.Error) {
	if deckErr := r.ensureDeck(resolvedDeck); deckErr != nil {
		deckErr.Key = &key
		return &protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: nil}, []protocol.Error{*deckErr}
	}

	noteID, err := r.client.AddNote(r.ctx, resolvedDeck, entry.NoteType, fields, entry.Tags)
	if err != nil {
		return &protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: nil},
			[]protocol.Error{{Code: protocol.CodeAnkiConnectError, Message: err.Error(), Key: &key}}
	}

	r.reg.Put(registry.Entry{
		NoteID:       noteID,
		SourcePath:   entry.SourcePath,
		NoteType:     entry.NoteType,
		ResolvedDeck: resolvedDeck,
		ContentHash:  contentHash,
	})
	return &protocol.Result{Key: &key, Action: protocol.ActionAdded, NoteID: &noteID}, nil
}

// reconcile implements REQ-012: query AnkiConnect for an identifier the
// registry does not hold. When it names an existing note whose note type
// matches the target's, adopt it into the registry and treat the update as
// though both the recorded hash AND the recorded resolved deck had
// differed — so the reconciling update re-asserts the target's resolved
// deck placement as well as its content (plan.md D-3's registry-loss
// recovery story). Otherwise, treat the target as unsynchronized and add a
// new note: a lost registry costs a redundant update, never a duplicate.
func (r *runner) reconcile(entry protocol.Entry, fields map[string]string, resolvedDeck, contentHash, key string, staleNoteID int) (*protocol.Result, []protocol.Error) {
	infos, err := r.client.NotesInfo(r.ctx, []int{staleNoteID})
	if err != nil {
		// The identifier cannot even be checked; treat conservatively as
		// unsynchronized rather than block the whole run on a transient
		// query failure — the add path is always available as a fallback.
		return r.addNewNote(entry, fields, resolvedDeck, contentHash, key)
	}
	info := infos[0]
	if !info.Exists || info.ModelName != entry.NoteType {
		return r.addNewNote(entry, fields, resolvedDeck, contentHash, key)
	}

	// Matched: adopt and force both the content update and the deck move.
	if err := r.client.UpdateNoteFields(r.ctx, staleNoteID, fields); err != nil {
		return &protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: &staleNoteID},
			[]protocol.Error{{Code: protocol.CodeAnkiConnectError, Message: err.Error(), Key: &key}}
	}
	if err := r.client.UpdateNoteTags(r.ctx, staleNoteID, entry.Tags); err != nil {
		return &protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: &staleNoteID},
			[]protocol.Error{{Code: protocol.CodeAnkiConnectError, Message: err.Error(), Key: &key}}
	}

	var errs []protocol.Error
	if err := r.client.ChangeDeck(r.ctx, info.Cards, resolvedDeck); err != nil {
		errs = append(errs, protocol.Error{
			Code:    protocol.CodeDeckMoveFailed,
			Message: fmt.Sprintf("note %d: %v", staleNoteID, err),
			Key:     &key,
		})
	}

	r.reg.Put(registry.Entry{
		NoteID:       staleNoteID,
		SourcePath:   entry.SourcePath,
		NoteType:     entry.NoteType,
		ResolvedDeck: resolvedDeck,
		ContentHash:  contentHash,
	})
	return &protocol.Result{Key: &key, Action: protocol.ActionUpdated, NoteID: &staleNoteID}, errs
}

// updateNote implements REQ-010: update fields and tags by identifier, then
// — only if the target's resolved deck differs from the registry's
// recorded last-synced deck — retrieve the note's card identifiers and
// move those cards. Where the two decks already agree, neither notesInfo
// nor changeDeck is issued at all (acceptance.md AC-016).
func (r *runner) updateNote(entry protocol.Entry, fields map[string]string, resolvedDeck, contentHash, key string, noteID int, prior registry.Entry) (*protocol.Result, []protocol.Error) {
	if err := r.client.UpdateNoteFields(r.ctx, noteID, fields); err != nil {
		return &protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: &noteID},
			[]protocol.Error{{Code: protocol.CodeAnkiConnectError, Message: err.Error(), Key: &key}}
	}
	if err := r.client.UpdateNoteTags(r.ctx, noteID, entry.Tags); err != nil {
		return &protocol.Result{Key: &key, Action: protocol.ActionFailed, NoteID: &noteID},
			[]protocol.Error{{Code: protocol.CodeAnkiConnectError, Message: err.Error(), Key: &key}}
	}

	newDeck := prior.ResolvedDeck
	var errs []protocol.Error
	if resolvedDeck != prior.ResolvedDeck {
		infos, err := r.client.NotesInfo(r.ctx, []int{noteID})
		if err != nil {
			errs = append(errs, protocol.Error{
				Code:    protocol.CodeDeckMoveFailed,
				Message: fmt.Sprintf("note %d: card lookup failed: %v", noteID, err),
				Key:     &key,
			})
		} else if err := r.client.ChangeDeck(r.ctx, infos[0].Cards, resolvedDeck); err != nil {
			errs = append(errs, protocol.Error{
				Code:    protocol.CodeDeckMoveFailed,
				Message: fmt.Sprintf("note %d: %v", noteID, err),
				Key:     &key,
			})
		} else {
			newDeck = resolvedDeck
		}
	}

	r.reg.Put(registry.Entry{
		NoteID:       noteID,
		SourcePath:   prior.SourcePath, // refreshed separately in step 10a
		NoteType:     entry.NoteType,
		ResolvedDeck: newDeck,
		ContentHash:  contentHash,
	})
	return &protocol.Result{Key: &key, Action: protocol.ActionUpdated, NoteID: &noteID}, errs
}

// reconcileSourcePaths is design.md §3 step 10a: for every registry
// identifier the census reports, refresh its recorded source path — for
// EVERY reported identifier, regardless of which branch (if any) step 9
// put its entry in, since this is a local registry write and REQ-011's
// AnkiConnect-request prohibition does not reach it. A census occurrence
// naming an identifier the registry does not hold is ignored here; that is
// REQ-012's reconcile branch's job (already run, above), or nothing at all.
//
// A duplicated identifier — the census reporting the same note_id at more
// than one location — takes the FIRST occurrence in scan order (the order
// census[] itself carries, REQ-022) as its refreshed path.
func reconcileSourcePaths(census []protocol.CensusEntry, reg *registry.Registry) {
	seen := make(map[int]bool, len(census))
	for _, c := range census {
		if seen[c.NoteID] {
			continue
		}
		seen[c.NoteID] = true
		if e, ok := reg.Lookup(c.NoteID); ok {
			e.SourcePath = c.SourcePath
			reg.Put(e)
		}
	}
}

// computeCandidates is design.md §3 step 10b: candidates = registry
// identifiers minus every identifier the census reports (at any location,
// excluded or not) minus every identifier the census reports NOWHERE but
// whose (just-refreshed) recorded source path matches a current exclusion
// pattern. The first subtraction answers "does this heading still exist?";
// the second is a residue term for a heading that exists nowhere and was
// last seen at an excluded location (plan.md D-13).
func computeCandidates(census []protocol.CensusEntry, reg *registry.Registry, excludePatterns []string, eligible map[int]bool) []int {
	inCensus := make(map[int]bool, len(census))
	for _, c := range census {
		inCensus[c.NoteID] = true
	}

	var candidates []int
	for _, e := range reg.All() {
		if !eligible[e.NoteID] {
			continue // registered during this very run — never itself a candidate
		}
		if inCensus[e.NoteID] {
			continue
		}
		if matchesAnyExclusion(e.SourcePath, excludePatterns) {
			continue // residue term: last seen at an excluded location
		}
		candidates = append(candidates, e.NoteID)
	}
	return candidates
}

// matchesAnyExclusion reports whether path falls under any of the
// configured exclusion patterns. Pattern syntax richness (globs, anchors)
// is the front end's scan-time concern (imoogi-exclude-patterns, plan.md
// D-8); the back end only needs to re-apply the same patterns against a
// RECORDED path for the residue term above, so a directory-prefix
// substring match is sufficient for that narrow purpose.
func matchesAnyExclusion(path string, patterns []string) bool {
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if containsSubstring(path, p) {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// confirmAndDelete is design.md §3 step 11 (plan.md D-9 steps 2-4). On an
// incomplete scan, the WHOLE delete path is skipped with no confirmation
// query at all (REQ-015). On an empty candidate set, nothing is skipped —
// there is simply nothing to confirm, and no query is issued either
// (acceptance.md AC-015's steady-state guarantee). Otherwise, one notesInfo
// call confirms every candidate by identifier, note type, AND rendered
// field content against the registry's record; confirmed candidates are
// deleted in one batched deleteNotes call naming them explicitly (REQ-016).
func (r *runner) confirmAndDelete(candidates []int, scanComplete bool) ([]protocol.Result, []protocol.Error) {
	if !scanComplete {
		return nil, []protocol.Error{{
			Code:    protocol.CodeDeleteSuppressed,
			Message: "scan of the sync root was incomplete; deletion suppressed for this run",
		}}
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	infos, err := r.client.NotesInfo(r.ctx, candidates)
	if err != nil {
		return nil, []protocol.Error{{
			Code:    protocol.CodeDeleteSuppressed,
			Message: fmt.Sprintf("orphan confirmation query failed: %v", err),
		}}
	}

	var confirmed []int
	var errs []protocol.Error
	for _, info := range infos {
		if !info.Exists {
			continue // already-absent: skip silently, not an error (REQ-014)
		}
		e, ok := r.reg.Lookup(info.NoteID)
		if !ok {
			continue // defensive: not a real candidate without a registry entry
		}
		recomputedHash := hashing.Hash(info.ModelName, fieldValuesToStrings(info.Fields), e.ResolvedDeck, info.Tags)
		if info.ModelName != e.NoteType || recomputedHash != e.ContentHash {
			errs = append(errs, protocol.Error{
				Code:    protocol.CodeDeleteCandidateUnowned,
				Message: fmt.Sprintf("note %d exists but its note type or field content does not match the registry's record", info.NoteID),
			})
			continue // unowned: skip, registry entry retained (REQ-014)
		}
		confirmed = append(confirmed, info.NoteID)
	}

	var results []protocol.Result
	if len(confirmed) > 0 {
		if err := r.client.DeleteNotes(r.ctx, confirmed); err != nil {
			errs = append(errs, protocol.Error{
				Code:    protocol.CodeAnkiConnectError,
				Message: fmt.Sprintf("deleteNotes failed for %v: %v", confirmed, err),
			})
		} else {
			for _, id := range confirmed {
				r.reg.Remove(id)
				noteID := id
				results = append(results, protocol.Result{Key: nil, Action: protocol.ActionDeleted, NoteID: &noteID})
			}
		}
	}
	return results, errs
}

func fieldValuesToStrings(fields map[string]ankiconnect.FieldValue) map[string]string {
	out := make(map[string]string, len(fields))
	for name, v := range fields {
		out[name] = v.Value
	}
	return out
}
