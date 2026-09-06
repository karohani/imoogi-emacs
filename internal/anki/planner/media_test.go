package planner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/hashing"
	"github.com/karohani/imoogi-emacs/internal/anki/media"
	"github.com/karohani/imoogi-emacs/internal/anki/orgdoc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

// ---- helpers ----

// mediaRoot returns a fresh sync root plus a writeAsset closure that plants
// media beside a note file. Entry source paths are RELATIVE to the sync
// root, which is what the Elisp front end sends (imoogi-scan.el records a
// relative path), so the base directory is the joined path's directory.
func mediaRoot(t *testing.T) (string, func(relDir, name string, content []byte) string) {
	t.Helper()
	root := t.TempDir()
	return root, func(relDir, name string, content []byte) string {
		t.Helper()
		dir := filepath.Join(root, relDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, content, 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
		return p
	}
}

// storedNameFor re-derives design.md §9.3's stored name independently of
// the implementation under test.
func storedNameFor(base, ext string, content []byte) string {
	sum := sha256.Sum256(content)
	return base + "-" + hex.EncodeToString(sum[:])[:12] + ext
}

// mediaConfig is baseConfig with a real sync root, so the media pass has a
// directory to resolve against.
func mediaConfig(root string) protocol.Config {
	c := baseConfig()
	c.SyncRoot = root
	return c
}

// renderMediaAndHash is renderAndHash with the media pass interposed — the
// pipeline order REQ-C-016 fixes. Tests seeding a PRIOR registry hash for a
// media-bearing entry must use this, not renderAndHash.
func renderMediaAndHash(t *testing.T, root, sourcePath, noteType, title, body, deck string, tags []string) string {
	t.Helper()
	fields, err := orgdoc.Render(noteType, title, body)
	if err != nil {
		t.Fatalf("renderMediaAndHash: render: %v", err)
	}
	baseDir := filepath.Dir(filepath.Join(root, sourcePath))
	rewritten, _, err := media.Rewrite(baseDir, root, fields)
	if err != nil {
		t.Fatalf("renderMediaAndHash: media: %v", err)
	}
	return hashing.Hash(noteType, rewritten, deck, tags)
}

// ---- AC-C-016a: the load-bearing ordering guard ----

// The field map the client is handed and the field map the registry hashes
// must be the SAME bytes. Prose sequencing in planner.go is not the
// guarantee; this recomputation is — it is the very computation
// confirmAndDelete's ownership predicate performs on a notesInfo response.
func TestRun_Media_StoredBytesEqualHashedBytes(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	content := []byte("DIAGRAM-BYTES")
	writeAsset("notes", "diagram.png", content)
	stored := storedNameFor("diagram", ".png", content)

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	req := protocol.Request{
		Config: mediaConfig(root),
		Entries: []protocol.Entry{{
			Key:        "notes/a.org::0",
			NoteID:     nil,
			NoteType:   "Basic",
			SourcePath: "notes/a.org",
			Tags:       []string{"z", "a"},
			Title:      "Euler",
			// Both transforms this SPEC introduces, in one entry.
			Body: "$e^{i\\pi}+1=0$\n\n[[file:diagram.png]]",
		}},
	}

	results, errs := Run(context.Background(), req, reg, client)
	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if results[0].Action != protocol.ActionAdded {
		t.Fatalf("action = %q, want added", results[0].Action)
	}
	if len(client.addCalls) != 1 {
		t.Fatalf("addCalls = %d, want 1", len(client.addCalls))
	}

	sent := client.addCalls[0].fields

	// The transforms actually reached the dispatched text — without this
	// the byte-equality below would hold vacuously on an untransformed map.
	if !strings.Contains(sent["Back"], `src="`+stored+`"`) {
		t.Errorf("dispatched Back does not carry the stored media name %q:\n%s", stored, sent["Back"])
	}
	if strings.Contains(sent["Back"], `src="diagram.png"`) {
		t.Errorf("dispatched Back still carries the unrewritten reference:\n%s", sent["Back"])
	}
	if !strings.Contains(sent["Back"], `\(`) {
		t.Errorf("dispatched Back does not carry the MathJax delimiter transform: %s", sent["Back"])
	}

	entry, ok := reg.Lookup(*results[0].NoteID)
	if !ok {
		t.Fatalf("registry has no entry for note %d", *results[0].NoteID)
	}

	// The guard: recomputing the hash from what the CLIENT was handed
	// reproduces the hash the REGISTRY recorded, exactly.
	recomputed := hashing.Hash(client.addCalls[0].modelName, sent, client.addCalls[0].deck, client.addCalls[0].tags)
	if recomputed != entry.ContentHash {
		t.Errorf("hash recomputed from the dispatched field map = %s\nregistry recorded            = %s\n"+
			"the bytes sent to AnkiConnect are not the bytes that were hashed", recomputed, entry.ContentHash)
	}
}

// AC-C-016c: the same recomputation, run through a notesInfo response, is
// what the orphan-deletion ownership predicate performs — and it must
// still match after the referenced image's bytes change.
func TestRun_Media_OwnershipPredicateSurvivesAMediaEdit(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	writeAsset("notes", "diagram.png", []byte("V1"))

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	entryOf := func() protocol.Entry {
		return protocol.Entry{
			Key:        "notes/a.org::0",
			NoteType:   "Basic",
			SourcePath: "notes/a.org",
			Title:      "T",
			Body:       "[[file:diagram.png]]",
		}
	}

	results, _ := Run(context.Background(), protocol.Request{
		Config: mediaConfig(root), Entries: []protocol.Entry{entryOf()},
	}, reg, client)
	noteID := *results[0].NoteID

	// Edit the image's bytes and re-sync the same heading. The census names
	// the note, without which the delete path would (correctly) read it as
	// an orphan and reap it.
	writeAsset("notes", "diagram.png", []byte("V2-DIFFERENT"))
	second := entryOf()
	second.NoteID = &noteID
	Run(context.Background(), protocol.Request{
		Config:  mediaConfig(root),
		Census:  []protocol.CensusEntry{{NoteID: noteID, SourcePath: "notes/a.org"}},
		Entries: []protocol.Entry{second},
	}, reg, client)

	regEntry, _ := reg.Lookup(noteID)
	infos, err := client.NotesInfo(context.Background(), []int{noteID})
	if err != nil {
		t.Fatalf("NotesInfo: %v", err)
	}
	info := infos[0]
	predicate := hashing.Hash(info.ModelName, fieldValuesToStrings(info.Fields), regEntry.ResolvedDeck, info.Tags)
	if info.ModelName != regEntry.NoteType || predicate != regEntry.ContentHash {
		t.Errorf("ownership predicate no longer matches after a media edit:\n"+
			"recomputed = %s\nregistry   = %s", predicate, regEntry.ContentHash)
	}
}

// AC-C-016b: an image byte change produces an update; a byte-identical
// image produces a no-op. Image bytes reach the hash SOLELY through the
// content-derived stored filename written into the field text.
func TestRun_Media_ImageEditUpdates_ByteIdenticalIsNoOp(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	writeAsset("notes", "diagram.png", []byte("V1"))

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	entryOf := func(id *int) protocol.Entry {
		return protocol.Entry{
			Key: "notes/a.org::0", NoteID: id, NoteType: "Basic",
			SourcePath: "notes/a.org", Title: "T", Body: "[[file:diagram.png]]",
		}
	}
	run := func(e protocol.Entry) []protocol.Result {
		// A heading already carrying an identifier must also appear in the
		// census, or the delete path reads it as an orphan and reaps it.
		var census []protocol.CensusEntry
		if e.NoteID != nil {
			census = []protocol.CensusEntry{{NoteID: *e.NoteID, SourcePath: e.SourcePath}}
		}
		res, errs := Run(context.Background(), protocol.Request{
			Config: mediaConfig(root), Census: census, Entries: []protocol.Entry{e},
		}, reg, client)
		if len(errs) != 0 {
			t.Fatalf("errs = %+v, want none", errs)
		}
		return res
	}

	first := run(entryOf(nil))
	noteID := *first[0].NoteID
	hashAfterAdd := mustLookup(t, reg, noteID).ContentHash

	// Rewriting the SAME bytes: content-identical, so no-op.
	writeAsset("notes", "diagram.png", []byte("V1"))
	res := run(entryOf(&noteID))
	if res[0].Action != protocol.ActionSkipped {
		t.Errorf("action = %q, want skipped for a byte-identical image", res[0].Action)
	}
	if len(client.updateFieldsCalls) != 0 {
		t.Errorf("updateFieldsCalls = %d, want 0 for a byte-identical image", len(client.updateFieldsCalls))
	}
	if got := mustLookup(t, reg, noteID).ContentHash; got != hashAfterAdd {
		t.Errorf("hash changed on a byte-identical image: %s -> %s", hashAfterAdd, got)
	}

	// Different bytes: the stored name changes, so the field text changes,
	// so the hash changes, so the note is updated.
	writeAsset("notes", "diagram.png", []byte("V2-DIFFERENT"))
	res = run(entryOf(&noteID))
	if res[0].Action != protocol.ActionUpdated {
		t.Errorf("action = %q, want updated after an image edit", res[0].Action)
	}
	if len(client.updateFieldsCalls) != 1 {
		t.Fatalf("updateFieldsCalls = %d, want 1", len(client.updateFieldsCalls))
	}
	if got := mustLookup(t, reg, noteID).ContentHash; got == hashAfterAdd {
		t.Errorf("hash unchanged after an image edit: %s", got)
	}
}

// AC-C-016b, the signature half: the hash's input set stays exactly the
// four the parent SPEC's D-6 fixes. A new parameter — image bytes, a file
// path, a media list — breaks this compile-time assertion.
func TestHashInputSetIsUnchangedByTheMediaPass(t *testing.T) {
	// Named so the assertion reads as the contract it is: adding a fifth
	// parameter — image bytes, a file path, a media list — stops compiling
	// here rather than silently widening the hash's input set.
	type parentSPECHashInputs = func(noteType string, fields map[string]string, deck string, tags []string) string
	_ = parentSPECHashInputs(hashing.Hash)
}

// ---- AC-C-014: gating and dedupe ----

// AC-C-014a, the media analogue of AC-015: an unchanged re-sync issues no
// storeMediaFile request at all. Upload rides the add/update paths, not the
// rewrite, so a no-op entry costs nothing.
func TestRun_Media_UnchangedResyncIssuesZeroStoreMediaFileCalls(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	writeAsset("notes", "diagram.png", []byte("STEADY"))

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	entryOf := func(id *int) protocol.Entry {
		return protocol.Entry{
			Key: "notes/a.org::0", NoteID: id, NoteType: "Basic",
			SourcePath: "notes/a.org", Title: "T", Body: "[[file:diagram.png]]",
		}
	}
	first, _ := Run(context.Background(), protocol.Request{
		Config: mediaConfig(root), Entries: []protocol.Entry{entryOf(nil)},
	}, reg, client)
	noteID := *first[0].NoteID
	if len(client.storeMediaFileCalls) != 1 {
		t.Fatalf("first run storeMediaFileCalls = %d, want 1", len(client.storeMediaFileCalls))
	}

	// Second run over an unchanged tree.
	client.storeMediaFileCalls = nil
	client.addCalls = nil
	client.updateFieldsCalls = nil
	client.updateTagsCalls = nil

	e := entryOf(&noteID)
	census := []protocol.CensusEntry{{NoteID: noteID, SourcePath: "notes/a.org"}}
	res, errs := Run(context.Background(), protocol.Request{
		Config: mediaConfig(root), Census: census, Entries: []protocol.Entry{e},
	}, reg, client)
	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if res[0].Action != protocol.ActionSkipped {
		t.Fatalf("action = %q, want skipped", res[0].Action)
	}
	if len(client.storeMediaFileCalls) != 0 {
		t.Errorf("storeMediaFileCalls = %d, want exactly 0 on an unchanged re-sync", len(client.storeMediaFileCalls))
	}
	if len(client.addCalls)+len(client.updateFieldsCalls)+len(client.updateTagsCalls) != 0 {
		t.Errorf("note-mutating calls fired on an unchanged re-sync")
	}
}

// AC-C-014b: three entries referencing one image upload it once.
func TestRun_Media_SharedImageUploadsExactlyOncePerRun(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	content := []byte("SHARED")
	writeAsset("notes", "shared.png", content)
	stored := storedNameFor("shared", ".png", content)

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	var entries []protocol.Entry
	for _, k := range []string{"a", "b", "c"} {
		entries = append(entries, protocol.Entry{
			Key: "notes/" + k + ".org::0", NoteType: "Basic",
			SourcePath: "notes/" + k + ".org", Title: k, Body: "[[file:shared.png]]",
		})
	}

	_, errs := Run(context.Background(), protocol.Request{
		Config: mediaConfig(root), Entries: entries,
	}, reg, client)
	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(client.addCalls) != 3 {
		t.Fatalf("addCalls = %d, want 3", len(client.addCalls))
	}
	if len(client.storeMediaFileCalls) != 1 {
		t.Errorf("storeMediaFileCalls = %d, want 1 after per-run dedupe by stored name", len(client.storeMediaFileCalls))
	}
	if client.storeMediaFileCalls[0].filename != stored {
		t.Errorf("uploaded filename = %q, want %q", client.storeMediaFileCalls[0].filename, stored)
	}
}

// ---- AC-C-015: unresolvable and unuploadable media are contained skips ----

func TestRun_Media_MissingFileSkipsWithCodeAndLeavesRegistryHashUnchanged(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	writeAsset("notes", "present.png", []byte("PRESENT"))

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	// A previously synchronized entry whose image has since vanished.
	priorHash := "0000000000000000000000000000000000000000000000000000000000000000"
	reg.Put(registry.Entry{
		NoteID: 41, SourcePath: "notes/a.org", NoteType: "Basic",
		ResolvedDeck: "Inbox", ContentHash: priorHash,
	})

	req := protocol.Request{
		Config: mediaConfig(root),
		Entries: []protocol.Entry{
			{
				Key: "notes/a.org::0", NoteID: intPtr(41), NoteType: "Basic",
				SourcePath: "notes/a.org", Title: "Broken", Body: "[[file:gone.png]]",
			},
			// A healthy sibling: the run must keep processing.
			{
				Key: "notes/b.org::0", NoteType: "Basic",
				SourcePath: "notes/b.org", Title: "Fine", Body: "[[file:present.png]]",
			},
		},
		Census: []protocol.CensusEntry{{NoteID: 41, SourcePath: "notes/a.org"}},
	}
	// notes/b.org resolves against notes/, where present.png lives.

	results, errs := Run(context.Background(), req, reg, client)

	var broken *protocol.Result
	for i := range results {
		if results[i].Key != nil && *results[i].Key == "notes/a.org::0" {
			broken = &results[i]
		}
	}
	if broken == nil {
		t.Fatalf("no result for the broken entry; results = %+v", results)
	}
	if broken.Action != protocol.ActionSkipped {
		t.Errorf("action = %q, want skipped", broken.Action)
	}

	var found bool
	for _, e := range errs {
		if e.Code == protocol.CodeMediaFileNotFound && e.Key != nil && *e.Key == "notes/a.org::0" {
			found = true
			if !strings.Contains(e.Message, "gone.png") {
				t.Errorf("diagnostic %q does not name the reference", e.Message)
			}
		}
	}
	if !found {
		t.Errorf("no %s diagnostic for the broken entry; errs = %+v", protocol.CodeMediaFileNotFound, errs)
	}

	// No note write for the broken entry.
	for _, c := range client.updateFieldsCalls {
		if c.noteID == 41 {
			t.Errorf("updateNoteFields issued for the skipped entry")
		}
	}
	// Every synchronization-bearing field of the skipped entry must survive
	// the run, not merely its hash. SourcePath is deliberately excluded:
	// step 10a refreshes it from the census for every reported identifier,
	// skip or not, and that is a local registry write REQ-C-015 does not
	// reach.
	after := mustLookup(t, reg, 41)
	if after.ContentHash != priorHash {
		t.Errorf("registry hash changed for the skipped entry: %s -> %s", priorHash, after.ContentHash)
	}
	if after.NoteType != "Basic" || after.ResolvedDeck != "Inbox" {
		t.Errorf("registry entry mutated for the skipped entry: %+v", after)
	}

	// The healthy sibling was still processed.
	if len(client.addCalls) != 1 {
		t.Errorf("addCalls = %d, want 1 — the run must continue past a skipped entry", len(client.addCalls))
	}
}

// REQ-C-015's upload half: a storeMediaFile failure carries
// media_upload_failed, and — the part that matters — no note is written,
// because the upload runs BEFORE the note write on every dispatching path.
func TestRun_Media_UploadFailureReportsCodeAndWritesNoNote(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	writeAsset("notes", "diagram.png", []byte("UPLOAD-FAIL"))

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true
	client.storeMediaFileErr = errFakeTransport

	results, errs := Run(context.Background(), protocol.Request{
		Config: mediaConfig(root),
		Entries: []protocol.Entry{{
			Key: "notes/a.org::0", NoteType: "Basic",
			SourcePath: "notes/a.org", Title: "T", Body: "[[file:diagram.png]]",
		}},
	}, reg, client)

	if len(results) != 1 || results[0].Action == protocol.ActionAdded {
		t.Errorf("results = %+v, want a non-added result", results)
	}
	var found bool
	for _, e := range errs {
		if e.Code == protocol.CodeMediaUploadFailed {
			found = true
		}
	}
	if !found {
		t.Errorf("no %s diagnostic; errs = %+v", protocol.CodeMediaUploadFailed, errs)
	}
	if len(client.addCalls) != 0 {
		t.Errorf("addCalls = %d, want 0 — the upload failed, so no note may be written", len(client.addCalls))
	}
	if len(reg.All()) != 0 {
		t.Errorf("registry gained %d entries after a failed upload, want 0", len(reg.All()))
	}
}

// The same upload failure on the UPDATE path. This is where REQ-C-015's
// "leave that entry's existing registry hash unchanged" clause has teeth:
// an already-synchronized note must not have its recorded hash advanced by
// a run that wrote nothing.
func TestRun_Media_UploadFailureOnUpdateLeavesRegistryHashUnchanged(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	writeAsset("notes", "diagram.png", []byte("V1"))

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	entryOf := func(id *int) protocol.Entry {
		return protocol.Entry{
			Key: "notes/a.org::0", NoteID: id, NoteType: "Basic",
			SourcePath: "notes/a.org", Title: "T", Body: "[[file:diagram.png]]",
		}
	}
	first, _ := Run(context.Background(), protocol.Request{
		Config: mediaConfig(root), Entries: []protocol.Entry{entryOf(nil)},
	}, reg, client)
	noteID := *first[0].NoteID
	hashAfterAdd := mustLookup(t, reg, noteID).ContentHash

	// The image changes (so the entry would be updated) but the upload fails.
	writeAsset("notes", "diagram.png", []byte("V2-DIFFERENT"))
	client.storeMediaFileErr = errFakeTransport
	client.updateFieldsCalls = nil

	results, errs := Run(context.Background(), protocol.Request{
		Config:  mediaConfig(root),
		Census:  []protocol.CensusEntry{{NoteID: noteID, SourcePath: "notes/a.org"}},
		Entries: []protocol.Entry{entryOf(&noteID)},
	}, reg, client)

	if results[0].Action != protocol.ActionSkipped {
		t.Errorf("action = %q, want skipped", results[0].Action)
	}
	if len(errs) == 0 || errs[0].Code != protocol.CodeMediaUploadFailed {
		t.Errorf("errs = %+v, want %s", errs, protocol.CodeMediaUploadFailed)
	}
	if len(client.updateFieldsCalls) != 0 {
		t.Errorf("updateNoteFields issued after a failed upload")
	}
	if got := mustLookup(t, reg, noteID).ContentHash; got != hashAfterAdd {
		t.Errorf("registry hash advanced despite writing nothing: %s -> %s", hashAfterAdd, got)
	}
}

// And on the reconcile path, where the identifier is real but the registry
// has lost its record. Adoption must not proceed on a failed upload either.
func TestRun_Media_UploadFailureOnReconcileWritesNothing(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	writeAsset("notes", "diagram.png", []byte("RECONCILE"))

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true
	client.storeMediaFileErr = errFakeTransport
	// The collection holds the note; the registry does not.
	client.seedNote(55, "Basic", map[string]string{"Front": "T", "Back": "old"}, nil, []int{5501})

	results, errs := Run(context.Background(), protocol.Request{
		Config:  mediaConfig(root),
		Census:  []protocol.CensusEntry{{NoteID: 55, SourcePath: "notes/a.org"}},
		Entries: []protocol.Entry{{Key: "notes/a.org::0", NoteID: intPtr(55), NoteType: "Basic", SourcePath: "notes/a.org", Title: "T", Body: "[[file:diagram.png]]"}},
	}, reg, client)

	if results[0].Action != protocol.ActionSkipped {
		t.Errorf("action = %q, want skipped", results[0].Action)
	}
	var found bool
	for _, e := range errs {
		if e.Code == protocol.CodeMediaUploadFailed {
			found = true
		}
	}
	if !found {
		t.Errorf("errs = %+v, want %s", errs, protocol.CodeMediaUploadFailed)
	}
	if len(client.updateFieldsCalls) != 0 || len(client.addCalls) != 0 {
		t.Errorf("a note was written despite the failed upload")
	}
	if _, ok := reg.Lookup(55); ok {
		t.Errorf("note 55 was adopted into the registry despite the failed upload")
	}
}

// AC-C-010b through the planner: an escape above the sync root is a
// confinement rejection reported as media_file_not_found, with no upload.
func TestRun_Media_EscapeAboveSyncRootIsSkipped(t *testing.T) {
	root, _ := mediaRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside.png")
	if err := os.WriteFile(outside, []byte("OUTSIDE"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	results, errs := Run(context.Background(), protocol.Request{
		Config: mediaConfig(root),
		Entries: []protocol.Entry{{
			Key: "notes/a.org::0", NoteType: "Basic",
			SourcePath: "notes/a.org", Title: "T", Body: "[[file:../../outside.png]]",
		}},
	}, reg, client)

	if results[0].Action != protocol.ActionSkipped {
		t.Errorf("action = %q, want skipped", results[0].Action)
	}
	if len(errs) == 0 || errs[0].Code != protocol.CodeMediaFileNotFound {
		t.Errorf("errs = %+v, want %s", errs, protocol.CodeMediaFileNotFound)
	}
	if len(client.storeMediaFileCalls) != 0 {
		t.Errorf("storeMediaFileCalls = %d, want 0", len(client.storeMediaFileCalls))
	}
	if len(client.addCalls) != 0 {
		t.Errorf("addCalls = %d, want 0", len(client.addCalls))
	}
}

// ---- AC-C-012a end to end: the two-part link through the planner ----

func TestRun_Media_TwoPartLinkReachesAnkiAsAnImg(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	content := []byte("TWO-PART-E2E")
	writeAsset("notes", "diagram.png", content)
	stored := storedNameFor("diagram", ".png", content)

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	_, errs := Run(context.Background(), protocol.Request{
		Config: mediaConfig(root),
		Entries: []protocol.Entry{{
			Key: "notes/a.org::0", NoteType: "Basic",
			SourcePath: "notes/a.org", Title: "T",
			Body: "[[file:diagram.png][My diagram]]",
		}},
	}, reg, client)
	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	back := client.addCalls[0].fields["Back"]
	if !strings.Contains(back, `<img src="`+stored+`" alt="My diagram"`) {
		t.Errorf("two-part link did not reach Anki as an img: %s", back)
	}
	if len(client.storeMediaFileCalls) != 1 {
		t.Errorf("storeMediaFileCalls = %d, want 1", len(client.storeMediaFileCalls))
	}
}

// AC-C-015b: no uploaded filename may begin with "_", in any run.
func TestRun_Media_NoUploadedFilenameBeginsWithUnderscore(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	writeAsset("notes", "_private.png", []byte("UNDERSCORE"))

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	_, errs := Run(context.Background(), protocol.Request{
		Config: mediaConfig(root),
		Entries: []protocol.Entry{{
			Key: "notes/a.org::0", NoteType: "Basic",
			SourcePath: "notes/a.org", Title: "T", Body: "[[file:_private.png]]",
		}},
	}, reg, client)
	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if len(client.storeMediaFileCalls) != 1 {
		t.Fatalf("storeMediaFileCalls = %d, want 1", len(client.storeMediaFileCalls))
	}
	if strings.HasPrefix(client.storeMediaFileCalls[0].filename, "_") {
		t.Errorf("uploaded filename %q begins with _", client.storeMediaFileCalls[0].filename)
	}
}

// ---- AC-C-017: the first run after the rendering change ----

// AC-C-017a: a note synchronized under the PRE-M4 rendering (an unrewritten
// src) is updated in a single pass, and the run reports success.
func TestRun_FirstRunAfterRenderingChange_UpdatesInOnePass(t *testing.T) {
	root, writeAsset := mediaRoot(t)
	writeAsset("notes", "diagram.png", []byte("PRE-M4"))

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	// The pre-M4 hash: rendered, but with no media pass interposed.
	preM4Hash := renderAndHash(t, "Basic", "T", "[[file:diagram.png]]", "Inbox", nil)
	reg.Put(registry.Entry{
		NoteID: 77, SourcePath: "notes/a.org", NoteType: "Basic",
		ResolvedDeck: "Inbox", ContentHash: preM4Hash,
	})
	client.seedNote(77, "Basic", map[string]string{"Front": "T", "Back": "x"}, nil, []int{7701})

	postM4Hash := renderMediaAndHash(t, root, "notes/a.org", "Basic", "T", "[[file:diagram.png]]", "Inbox", nil)
	if postM4Hash == preM4Hash {
		t.Fatalf("the media pass did not change the hash; the first-run invalidation cannot be observed")
	}

	results, errs := Run(context.Background(), protocol.Request{
		Config:  mediaConfig(root),
		Census:  []protocol.CensusEntry{{NoteID: 77, SourcePath: "notes/a.org"}},
		Entries: []protocol.Entry{{Key: "notes/a.org::0", NoteID: intPtr(77), NoteType: "Basic", SourcePath: "notes/a.org", Title: "T", Body: "[[file:diagram.png]]"}},
	}, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v — the invalidation is specified behavior, not an error", errs)
	}
	if results[0].Action != protocol.ActionUpdated {
		t.Errorf("action = %q, want updated", results[0].Action)
	}
	if len(client.updateFieldsCalls) != 1 {
		t.Errorf("updateFieldsCalls = %d, want exactly 1 (a single pass)", len(client.updateFieldsCalls))
	}
	if got := mustLookup(t, reg, 77).ContentHash; got != postM4Hash {
		t.Errorf("recorded hash = %s, want the post-transform hash %s", got, postM4Hash)
	}
}

// AC-C-017b: a note carrying neither math nor a local image is byte-
// identical under the new transforms, so no request is issued for it.
func TestRun_FirstRunAfterRenderingChange_PlainNoteIssuesNoRequest(t *testing.T) {
	root, _ := mediaRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	reg := newTestRegistry(t)
	client := newFakeClient()
	client.decks["Inbox"] = true

	body := "Just prose, no math and no image."
	hash := renderAndHash(t, "Basic", "T", body, "Inbox", nil)
	reg.Put(registry.Entry{
		NoteID: 78, SourcePath: "notes/p.org", NoteType: "Basic",
		ResolvedDeck: "Inbox", ContentHash: hash,
	})

	results, errs := Run(context.Background(), protocol.Request{
		Config:  mediaConfig(root),
		Census:  []protocol.CensusEntry{{NoteID: 78, SourcePath: "notes/p.org"}},
		Entries: []protocol.Entry{{Key: "notes/p.org::0", NoteID: intPtr(78), NoteType: "Basic", SourcePath: "notes/p.org", Title: "T", Body: body}},
	}, reg, client)

	if len(errs) != 0 {
		t.Fatalf("errs = %+v, want none", errs)
	}
	if results[0].Action != protocol.ActionSkipped {
		t.Errorf("action = %q, want skipped", results[0].Action)
	}
	if n := len(client.addCalls) + len(client.updateFieldsCalls) + len(client.updateTagsCalls) + len(client.storeMediaFileCalls); n != 0 {
		t.Errorf("%d requests issued for an unchanged plain note, want 0", n)
	}
	if got := mustLookup(t, reg, 78).ContentHash; got != hash {
		t.Errorf("hash changed for an unchanged plain note")
	}
}

// ---- base-directory derivation ----

func TestMediaBaseDir(t *testing.T) {
	cases := []struct {
		name       string
		root       string
		sourcePath string
		want       string
	}{
		{"nested source path", filepath.Join("/tmp", "notes"), filepath.Join("deep", "a.org"), filepath.Join("/tmp", "notes", "deep")},
		{"root-level source path", filepath.Join("/tmp", "notes"), "a.org", filepath.Join("/tmp", "notes")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mediaBaseDir(tc.root, tc.sourcePath); got != tc.want {
				t.Errorf("mediaBaseDir(%q, %q) = %q, want %q", tc.root, tc.sourcePath, got, tc.want)
			}
		})
	}
}

// The front end passes sync_root as the user configured it, and a defcustom
// holding "~/notes" is entirely ordinary. Expanding it here is defensive:
// an unexpanded "~" would make every reference fail confinement.
func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory available: %v", err)
	}
	cases := []struct{ in, want string }{
		{"~", home},
		{filepath.Join("~", "notes"), filepath.Join(home, "notes")},
		{filepath.Join("/abs", "notes"), filepath.Join("/abs", "notes")},
		{"", ""},
		{"~user/notes", "~user/notes"}, // another user's home: not ours to expand
	}
	for _, tc := range cases {
		if got := expandTilde(tc.in); got != tc.want {
			t.Errorf("expandTilde(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func mustLookup(t *testing.T, reg *registry.Registry, noteID int) registry.Entry {
	t.Helper()
	e, ok := reg.Lookup(noteID)
	if !ok {
		t.Fatalf("registry has no entry for note %d", noteID)
	}
	return e
}
