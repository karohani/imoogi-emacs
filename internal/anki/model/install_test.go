package model_test

import (
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/model"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// AC-C-001 — a collection holding neither imoogi-owned type takes the absent
// branch: exactly one createModel per type, zero updates, and each request
// carries that type's field list, its templates, and a non-empty css.
func TestInstallCreatesBothTypesWhenTheCollectionHasNeither(t *testing.T) {
	fake := newFakeClient("Basic", "Cloze")

	results, errs := model.Install(context.Background(), fake, "", io.Discard)

	if len(errs) != 0 {
		t.Fatalf("install reported %d errors on a clean run: %+v", len(errs), errs)
	}
	if fake.modelNamesCalls != 1 {
		t.Errorf("modelNames was called %d times, want exactly 1 (one probe per invocation)", fake.modelNamesCalls)
	}
	if len(fake.createModelCalls) != 2 {
		t.Fatalf("createModel called %d times, want exactly 2", len(fake.createModelCalls))
	}
	if n := len(fake.updateModelStylingCalls); n != 0 {
		t.Errorf("updateModelStyling called %d times on the absent branch, want 0", n)
	}
	if n := len(fake.updateModelTemplatesCalls); n != 0 {
		t.Errorf("updateModelTemplates called %d times on the absent branch, want 0", n)
	}

	wantShape := map[string]struct {
		fields  []string
		isCloze bool
	}{
		"imoogi-Basic": {[]string{"Front", "Back"}, false},
		"imoogi-Cloze": {[]string{"Text", "Back Extra"}, true},
	}
	seen := map[string]bool{}
	for _, call := range fake.createModelCalls {
		want, ok := wantShape[call.name]
		if !ok {
			t.Errorf("createModel named the unexpected model %q", call.name)
			continue
		}
		if seen[call.name] {
			t.Errorf("createModel named %q more than once", call.name)
		}
		seen[call.name] = true
		if !reflect.DeepEqual(call.inOrderFields, want.fields) {
			t.Errorf("%s inOrderFields = %v, want %v", call.name, call.inOrderFields, want.fields)
		}
		if call.isCloze != want.isCloze {
			t.Errorf("%s isCloze = %v, want %v", call.name, call.isCloze, want.isCloze)
		}
		if len(call.templates) == 0 {
			t.Errorf("%s carried no card templates", call.name)
		}
		if call.css == "" {
			t.Errorf("%s carried an empty css value", call.name)
		}
	}

	// One result per type, action "added", note_id null (REQ-C-002; the
	// existing Response shape carries the outcome, so no wire field is added).
	assertResults(t, results, map[string]string{
		"imoogi-Basic": protocol.ActionAdded,
		"imoogi-Cloze": protocol.ActionAdded,
	})
}

// AC-C-002 — a collection already holding both types takes the present
// branch: one updateModelStyling and one updateModelTemplates per type, and
// EXACTLY zero createModel. This is design.md §6's probe-then-act: the branch
// whose behavior on an existing name is undocumented is never reached.
func TestInstallUpdatesBothTypesWhenTheCollectionAlreadyHasThem(t *testing.T) {
	fake := newFakeClient("Basic", "Cloze", "imoogi-Basic", "imoogi-Cloze")

	results, errs := model.Install(context.Background(), fake, "", io.Discard)

	if len(errs) != 0 {
		t.Fatalf("install reported %d errors on a clean run: %+v", len(errs), errs)
	}
	if n := len(fake.createModelCalls); n != 0 {
		t.Errorf("createModel called %d times on the present branch, want exactly 0", n)
	}
	if n := len(fake.updateModelStylingCalls); n != 2 {
		t.Errorf("updateModelStyling called %d times, want 2", n)
	}
	if n := len(fake.updateModelTemplatesCalls); n != 2 {
		t.Errorf("updateModelTemplates called %d times, want 2", n)
	}
	assertResults(t, results, map[string]string{
		"imoogi-Basic": protocol.ActionUpdated,
		"imoogi-Cloze": protocol.ActionUpdated,
	})
}

// AC-C-002's second clause — a second consecutive invocation produces a
// request log byte-identical to the first.
func TestInstallIsIdempotentAcrossConsecutiveInvocations(t *testing.T) {
	first := newFakeClient("imoogi-Basic", "imoogi-Cloze")
	model.Install(context.Background(), first, "/* user */\n", io.Discard)

	second := newFakeClient("imoogi-Basic", "imoogi-Cloze")
	model.Install(context.Background(), second, "/* user */\n", io.Discard)

	if !reflect.DeepEqual(first.createModelCalls, second.createModelCalls) {
		t.Error("the createModel log differed between two identical invocations")
	}
	if !reflect.DeepEqual(first.updateModelStylingCalls, second.updateModelStylingCalls) {
		t.Errorf("the updateModelStyling log differed between two identical invocations:\n%+v\n%+v",
			first.updateModelStylingCalls, second.updateModelStylingCalls)
	}
	if !reflect.DeepEqual(first.updateModelTemplatesCalls, second.updateModelTemplatesCalls) {
		t.Error("the updateModelTemplates log differed between two identical invocations")
	}
}

// The absent branch settles into the present branch: install against a bare
// collection twice and the second run updates rather than creates, because
// the first run's createModel put the names into the collection the probe
// reads. This is the property "repeated invocations of imoogi-anki-setup are
// idempotent" (REQ-C-002.2) actually means in the field, where the first
// setup is the one that installs.
func TestInstallSwitchesToTheUpdateBranchOnItsSecondRunAgainstOneCollection(t *testing.T) {
	fake := newFakeClient("Basic", "Cloze")

	model.Install(context.Background(), fake, "", io.Discard)
	if len(fake.createModelCalls) != 2 {
		t.Fatalf("first run made %d createModel calls, want 2", len(fake.createModelCalls))
	}

	model.Install(context.Background(), fake, "", io.Discard)
	if n := len(fake.createModelCalls); n != 2 {
		t.Errorf("createModel total after the second run = %d, want 2 (the second run must not create)", n)
	}
	if n := len(fake.updateModelStylingCalls); n != 2 {
		t.Errorf("updateModelStyling total after the second run = %d, want 2", n)
	}
	if n := len(fake.updateModelTemplatesCalls); n != 2 {
		t.Errorf("updateModelTemplates total after the second run = %d, want 2", n)
	}
	if fake.modelNamesCalls != 2 {
		t.Errorf("modelNames was called %d times across two invocations, want 2", fake.modelNamesCalls)
	}
}

// AC-C-003b — the install path is ownership-scoped. The collection also holds
// the stock types, a user-authored type, and an imoogi-prefixed name that is
// NOT one of the two recognized types (REQ-C-001.1's passthrough clause: it
// is neither created, updated, nor reported, and raises no diagnostic).
func TestInstallWritesOnlyPrefixMatchingModels(t *testing.T) {
	fake := newFakeClient("Basic", "Cloze", "My Custom Type", "imoogi-Other", "imoogi-Basic")

	results, errs := model.Install(context.Background(), fake, "", io.Discard)

	if len(errs) != 0 {
		t.Fatalf("install reported %d errors: %+v", len(errs), errs)
	}
	written := fake.writtenModelNames()
	if len(written) == 0 {
		t.Fatal("install issued no model-write request at all")
	}
	for _, name := range written {
		if !model.IsOwned(name) {
			t.Errorf("a model-write request named the foreign model %q", name)
		}
		if name != "imoogi-Basic" && name != "imoogi-Cloze" {
			t.Errorf("a model-write request named %q, which is not one of the two recognized types", name)
		}
	}
	for _, r := range results {
		if r.Key != nil && *r.Key == "imoogi-Other" {
			t.Error("imoogi-Other was reported; REQ-C-001.1 says it passes through untouched")
		}
	}
	// The recognized pair is still handled: imoogi-Basic was present so it
	// updates, imoogi-Cloze was absent so it is created.
	if n := len(fake.createModelCalls); n != 1 || fake.createModelCalls[0].name != "imoogi-Cloze" {
		t.Errorf("createModel log = %+v, want exactly one call naming imoogi-Cloze", fake.createModelCalls)
	}
	if n := len(fake.updateModelStylingCalls); n != 1 || fake.updateModelStylingCalls[0].name != "imoogi-Basic" {
		t.Errorf("updateModelStyling log = %+v, want exactly one call naming imoogi-Basic", fake.updateModelStylingCalls)
	}
}

// AC-C-006a — the css value in EVERY model-write request is the base
// stylesheet followed verbatim by the user stylesheet, in that order.
func TestInstallUploadsBaseFollowedByUserStylesheet(t *testing.T) {
	const marker = "/* USER MARKER */ .card { letter-spacing: 0.02em; }\n"
	want := model.BaseCSS() + marker

	// Both branches carry css: createModel takes it as a parameter, and
	// updateModelStyling is the present branch's carrier. Both are checked.
	absent := newFakeClient()
	model.Install(context.Background(), absent, marker, io.Discard)
	for _, call := range absent.createModelCalls {
		if call.css != want {
			t.Errorf("createModel(%s) css is not base+user verbatim (got %d bytes, want %d)", call.name, len(call.css), len(want))
		}
	}

	present := newFakeClient("imoogi-Basic", "imoogi-Cloze")
	model.Install(context.Background(), present, marker, io.Discard)
	for _, call := range present.updateModelStylingCalls {
		if call.css != want {
			t.Errorf("updateModelStyling(%s) css is not base+user verbatim (got %d bytes, want %d)", call.name, len(call.css), len(want))
		}
	}
}

// AC-C-006b — no user stylesheet means the base alone, and no diagnostic.
func TestInstallWithNoUserStylesheetUploadsTheBaseAloneAndReportsNothing(t *testing.T) {
	fake := newFakeClient()
	_, errs := model.Install(context.Background(), fake, "", io.Discard)
	if len(errs) != 0 {
		t.Errorf("an absent user stylesheet produced %d diagnostics: %+v", len(errs), errs)
	}
	for _, call := range fake.createModelCalls {
		if call.css != model.BaseCSS() {
			t.Errorf("createModel(%s) css is not the base stylesheet alone", call.name)
		}
	}
}

// AC-C-006c — the uploaded base portion does not vary with the collection.
// Two collections whose decks differ entirely are modelled here as two
// entirely different existing-model sets, since the install step reads no deck
// at all — which is itself the point: no deck name reaches the stylesheet
// because no deck name reaches this code path.
func TestInstallUploadsAByteIdenticalBaseAcrossDifferentCollections(t *testing.T) {
	one := newFakeClient("Basic", "Deck A Type")
	two := newFakeClient("Cloze", "Deck B Type", "Something Else")
	model.Install(context.Background(), one, "", io.Discard)
	model.Install(context.Background(), two, "", io.Discard)

	if len(one.createModelCalls) == 0 || len(two.createModelCalls) == 0 {
		t.Fatal("one of the two collections issued no createModel")
	}
	if one.createModelCalls[0].css != two.createModelCalls[0].css {
		t.Error("the uploaded stylesheet differed between two collections")
	}
}

// AC-C-007 — wholesale replacement, announced BEFORE the first write. The
// announcement writer records how many model-write requests had been issued
// at the moment it was written to; the assertion is that the count was zero.
func TestInstallAnnouncesOwnershipBeforeTheFirstModelWrite(t *testing.T) {
	fake := newFakeClient("imoogi-Basic", "imoogi-Cloze")
	announce := &orderingWriter{writesAt: func() int { return fake.modelWrites() }}

	model.Install(context.Background(), fake, "", announce)

	if len(announce.snapshots) == 0 {
		t.Fatal("install issued no ownership announcement")
	}
	for i, at := range announce.snapshots {
		if at != 0 {
			t.Errorf("announcement chunk %d was written after %d model-write requests, want 0", i, at)
		}
	}
	text := announce.buf.String()
	// The message must say WHAT is owned and WHAT is lost — a bare "installing
	// models" line would satisfy a non-empty check and warn the user of
	// nothing.
	for _, want := range []string{"imoogi-Basic", "imoogi-Cloze", "replace"} {
		if !strings.Contains(text, want) {
			t.Errorf("the ownership announcement does not mention %q: %q", want, text)
		}
	}
}

// AC-C-007's first clause — the update path sends the FULL concatenation, not
// a merge with whatever the type already carried. The fake's stub collection
// holds a hand edit only in the sense that the request is what is asserted:
// the css sent equals base+user exactly, so nothing of a prior value survives.
func TestInstallReplacesStylingWholesaleRatherThanMerging(t *testing.T) {
	const marker = "/* USER */\n"
	fake := newFakeClient("imoogi-Basic", "imoogi-Cloze")
	model.Install(context.Background(), fake, marker, io.Discard)

	want := model.BaseCSS() + marker
	for _, call := range fake.updateModelStylingCalls {
		if call.css != want {
			t.Errorf("updateModelStyling(%s) sent %d bytes, want the full %d-byte concatenation", call.name, len(call.css), len(want))
		}
		if strings.Count(call.css, marker) != 1 {
			t.Errorf("updateModelStyling(%s) css carries the user stylesheet %d times, want once", call.name, strings.Count(call.css, marker))
		}
	}
}

// A failing probe cannot be recovered from: no type's branch is decidable, so
// nothing is installed and nothing is written. The run-level diagnostic is
// ankiconnect_error rather than model_install_failed, because no model
// install was attempted.
func TestInstallReportsARunLevelErrorWhenTheProbeFails(t *testing.T) {
	fake := newFakeClient("imoogi-Basic")
	fake.modelNamesErr = errFakeInstall

	results, errs := model.Install(context.Background(), fake, "", io.Discard)

	if len(results) != 0 {
		t.Errorf("a failed probe produced %d results, want 0", len(results))
	}
	if len(errs) != 1 {
		t.Fatalf("a failed probe produced %d errors, want exactly 1: %+v", len(errs), errs)
	}
	if errs[0].Code != protocol.CodeAnkiConnectError {
		t.Errorf("probe failure code = %q, want %q", errs[0].Code, protocol.CodeAnkiConnectError)
	}
	if errs[0].Key != nil {
		t.Errorf("probe failure is run-level, so its key must be nil; got %q", *errs[0].Key)
	}
	if fake.modelWrites() != 0 {
		t.Errorf("a failed probe still issued %d model-write requests, want 0", fake.modelWrites())
	}
}

// One type failing does not take the other down: the survivor is still
// reported, and the failure is keyed by the model name it belongs to
// (model_install_failed, design.md §5).
func TestInstallReportsAPerTypeFailureAndStillInstallsTheOther(t *testing.T) {
	fake := newFakeClient()
	fake.failModel["imoogi-Cloze"] = errFakeInstall

	results, errs := model.Install(context.Background(), fake, "", io.Discard)

	if len(results) != 1 {
		t.Fatalf("one failing type produced %d results, want 1: %+v", len(results), results)
	}
	if results[0].Key == nil || *results[0].Key != "imoogi-Basic" {
		t.Errorf("the surviving result is not imoogi-Basic: %+v", results[0])
	}
	if len(errs) != 1 {
		t.Fatalf("one failing type produced %d errors, want 1: %+v", len(errs), errs)
	}
	if errs[0].Code != protocol.CodeModelInstallFailed {
		t.Errorf("failure code = %q, want %q", errs[0].Code, protocol.CodeModelInstallFailed)
	}
	if errs[0].Key == nil || *errs[0].Key != "imoogi-Cloze" {
		t.Errorf("the failure is not keyed by the failing model name: %+v", errs[0])
	}
}

// A present type whose styling write fails does not get its templates written
// either — the two writes are one install for one type, and a half-applied
// type is a worse outcome than an unapplied one.
func TestInstallStopsAfterAFailedStylingWriteForThatType(t *testing.T) {
	fake := newFakeClient("imoogi-Basic", "imoogi-Cloze")
	fake.failModel["imoogi-Basic"] = errFakeInstall

	_, errs := model.Install(context.Background(), fake, "", io.Discard)

	if len(errs) != 1 {
		t.Fatalf("produced %d errors, want 1: %+v", len(errs), errs)
	}
	for _, call := range fake.updateModelTemplatesCalls {
		if call.name == "imoogi-Basic" {
			t.Error("updateModelTemplates was issued for imoogi-Basic after its styling write failed")
		}
	}
	if n := len(fake.updateModelTemplatesCalls); n != 1 {
		t.Errorf("updateModelTemplates called %d times, want 1 (the surviving type only)", n)
	}
}

// Every type failing means nothing was installed — the one case in which the
// caller reports ok:false.
func TestInstallProducesNoResultsWhenEveryTypeFails(t *testing.T) {
	fake := newFakeClient()
	fake.failModel["imoogi-Basic"] = errFakeInstall
	fake.failModel["imoogi-Cloze"] = errFakeInstall

	results, errs := model.Install(context.Background(), fake, "", io.Discard)

	if len(results) != 0 {
		t.Errorf("every type failed but %d results were reported: %+v", len(results), results)
	}
	if len(errs) != 2 {
		t.Errorf("every type failed but %d errors were reported, want 2: %+v", len(errs), errs)
	}
}

// A nil announcement writer is not a crash. The Elisp front end is the only
// caller today and always supplies one, but a nil writer is the shape a later
// caller reaches for when it wants the announcement suppressed.
func TestInstallToleratesANilAnnouncementWriter(t *testing.T) {
	fake := newFakeClient()
	results, errs := model.Install(context.Background(), fake, "", nil)
	if len(errs) != 0 || len(results) != 2 {
		t.Errorf("install with a nil announcement writer: %d results, %d errors; want 2 and 0", len(results), len(errs))
	}
}

// assertResults checks the one-result-per-type shape REQ-C-002 fixes: key is
// the model name, action is added or updated, and note_id is null (a note type
// has no note identifier, and the field is not repurposed).
func assertResults(t *testing.T, results []protocol.Result, want map[string]string) {
	t.Helper()
	if len(results) != len(want) {
		t.Fatalf("install produced %d results, want %d: %+v", len(results), len(want), results)
	}
	for _, r := range results {
		if r.Key == nil {
			t.Errorf("a result carries a nil key; the install step keys results by model name: %+v", r)
			continue
		}
		wantAction, ok := want[*r.Key]
		if !ok {
			t.Errorf("a result names the unexpected model %q", *r.Key)
			continue
		}
		if r.Action != wantAction {
			t.Errorf("result for %q has action %q, want %q", *r.Key, r.Action, wantAction)
		}
		if r.NoteID != nil {
			t.Errorf("result for %q carries note_id %d; a note type has none", *r.Key, *r.NoteID)
		}
	}
}

// orderingWriter records, for each Write, how many model-write requests the
// fake client had logged at that moment. It is what turns AC-C-007's "before
// the write" from prose into an assertion.
type orderingWriter struct {
	writesAt  func() int
	snapshots []int
	buf       strings.Builder
}

func (w *orderingWriter) Write(p []byte) (int, error) {
	w.snapshots = append(w.snapshots, w.writesAt())
	return w.buf.Write(p)
}
