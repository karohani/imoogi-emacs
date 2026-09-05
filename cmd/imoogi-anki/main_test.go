package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// invoke drives run() the way the front end drives the binary: arguments,
// one JSON document on stdin, one on stdout.
func invoke(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

// newAnkiConnectStub answers AnkiConnect's own recommended first call
// (requestPermission, research.md §3.5) with a granted permission — enough
// for ankiconnect.Client.Handshake to succeed, so tests exercising the full
// sync pipeline don't need a real Anki + AnkiConnect installation. Every
// request it does not recognize (an actual add/update/delete etc.) answers
// with a generic empty-result envelope, since these end-to-end main_test.go
// tests only need the handshake to succeed — the decision logic itself is
// internal/planner's own test suite.
func newAnkiConnectStub(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope struct {
			Action string `json:"action"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &envelope)
		w.Header().Set("Content-Type", "application/json")
		switch envelope.Action {
		case "requestPermission":
			_, _ = fmt.Fprint(w, `{"result": {"permission": "granted", "requireApiKey": false, "version": 6}, "error": null}`)
		case "modelFieldNames":
			_, _ = fmt.Fprint(w, `{"result": ["Front", "Back", "Text", "Extra"], "error": null}`)
		default:
			_, _ = fmt.Fprint(w, `{"result": null, "error": null}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// buildRequest assembles a minimal, well-formed request document targeting
// the given AnkiConnect URL and registry path, so each test gets its own
// isolated stub server and temp registry file rather than sharing the
// fixed, unreachable address the old minimalRequest constant hard-coded.
func buildRequest(t *testing.T, ankiConnectURL string) string {
	t.Helper()
	registryPath := filepath.Join(t.TempDir(), "registry.json")
	req := protocol.Request{
		ProtocolVersion: protocol.Version,
		Config: protocol.Config{
			DefaultDeck:     "Inbox",
			AnkiConnectURL:  ankiConnectURL,
			RegistryPath:    registryPath,
			SyncRoot:        t.TempDir(),
			ExcludePatterns: []string{},
			ScanComplete:    true,
		},
		Census:  []protocol.CensusEntry{},
		Entries: []protocol.Entry{},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	return string(data)
}

// minimalRequest is a static, well-formed request body used ONLY by tests
// that never reach the AnkiConnect handshake at all (a malformed body, a
// protocol-version mismatch caught before decode, an unknown subcommand) —
// tests that DO reach the handshake use buildRequest against a live stub.
const minimalRequest = `{
  "protocol_version": 1,
  "config": {
    "default_deck": "Inbox",
    "anki_connect_url": "http://127.0.0.1:8765",
    "registry_path": "/tmp/registry.json",
    "sync_root": "/tmp/notes",
    "exclude_patterns": [],
    "scan_complete": true
  },
  "census": [],
  "entries": []
}`

func TestVersionFlagPrintsNonEmptyStringAndExitsZero(t *testing.T) {
	// AC-001's second clause: invoking the binary with --version exits 0 and
	// prints a non-empty version string.
	code, stdout, _ := invoke(t, "", "--version")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Error("--version printed nothing; want a non-empty version string")
	}
	if !strings.Contains(stdout, "imoogi") {
		t.Errorf("--version output %q does not name the binary", stdout)
	}
}

func TestSyncDecodesRequestAndEmitsValidResponse(t *testing.T) {
	stub := newAnkiConnectStub(t)
	code, stdout, _ := invoke(t, buildRequest(t, stub.URL), "sync")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout: %s", code, stdout)
	}

	var resp protocol.Response
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("stdout is not a valid response document: %v\ngot: %s", err, stdout)
	}

	if resp.ProtocolVersion != protocol.Version {
		t.Errorf("response protocol_version = %d, want %d", resp.ProtocolVersion, protocol.Version)
	}
	if !resp.OK {
		t.Error("ok = false, want true for a well-formed request")
	}
	if len(resp.Results) != 0 {
		t.Errorf("len(results) = %d, want 0 — the request carries no entries", len(resp.Results))
	}
	if len(resp.Errors) != 0 {
		t.Errorf("len(errors) = %d, want 0", len(resp.Errors))
	}
}

func TestSyncEmitsEmptyArraysNotNullOnTheWire(t *testing.T) {
	// A nil slice marshals to null, which is a different wire value from []
	// and parses differently on the Elisp side.
	stub := newAnkiConnectStub(t)
	_, stdout, _ := invoke(t, buildRequest(t, stub.URL), "sync")

	if !strings.Contains(stdout, `"results":[]`) {
		t.Errorf(`response must carry "results":[] — got: %s`, stdout)
	}
	if !strings.Contains(stdout, `"errors":[]`) {
		t.Errorf(`response must carry "errors":[] — got: %s`, stdout)
	}
	if strings.Contains(stdout, `null`) {
		t.Errorf("an empty response must carry no null: %s", stdout)
	}
}

// AC-005: nothing listening at the configured AnkiConnect URL is reported
// as anki_unreachable, distinctly from a host that answers but is not
// AnkiConnect (AC-006 below).
func TestSyncReportsAnkiUnreachableWhenNothingListens(t *testing.T) {
	// Bind and immediately close a listener to get a genuinely unreachable
	// local address — connection refused, not a slow timeout.
	closedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	unreachableURL := closedSrv.URL
	closedSrv.Close()

	code, stdout, _ := invoke(t, buildRequest(t, unreachableURL), "sync")

	if code == 0 {
		t.Error("exit code = 0, want non-zero — a run-level failure before any entry is processed")
	}
	var resp protocol.Response
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("stdout is not a valid response document: %v\ngot: %s", err, stdout)
	}
	if resp.OK {
		t.Error("ok = true, want false")
	}
	if len(resp.Errors) != 1 || resp.Errors[0].Code != protocol.CodeAnkiUnreachable {
		t.Errorf("errors = %+v, want one anki_unreachable", resp.Errors)
	}
	if len(resp.Results) != 0 {
		t.Errorf("len(results) = %d, want 0 — nothing was processed", len(resp.Results))
	}
}

// AC-006: a host that answers but does not speak AnkiConnect's own protocol
// is distinguished from a stopped Anki — ankiconnect_missing, not
// anki_unreachable.
func TestSyncReportsAnkiConnectMissingWhenHostAnswersButIsNotAnkiConnect(t *testing.T) {
	notAnkiConnect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "not json at all")
	}))
	t.Cleanup(notAnkiConnect.Close)

	code, stdout, _ := invoke(t, buildRequest(t, notAnkiConnect.URL), "sync")

	if code == 0 {
		t.Error("exit code = 0, want non-zero")
	}
	var resp protocol.Response
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("stdout is not a valid response document: %v\ngot: %s", err, stdout)
	}
	if len(resp.Errors) != 1 || resp.Errors[0].Code != protocol.CodeAnkiConnectMissing {
		t.Errorf("errors = %+v, want one ankiconnect_missing — distinct from anki_unreachable", resp.Errors)
	}
}

// acceptance.md §D.7: a registry file that exists but is unparseable must
// report state_unreadable, and never silently start a fresh registry.
func TestSyncReportsStateUnreadableForCorruptRegistry(t *testing.T) {
	stub := newAnkiConnectStub(t)
	registryPath := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(registryPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	req := protocol.Request{
		ProtocolVersion: protocol.Version,
		Config: protocol.Config{
			DefaultDeck:    "Inbox",
			AnkiConnectURL: stub.URL,
			RegistryPath:   registryPath,
			SyncRoot:       t.TempDir(),
			ScanComplete:   true,
		},
		Census:  []protocol.CensusEntry{},
		Entries: []protocol.Entry{},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	code, stdout, _ := invoke(t, string(data), "sync")

	if code == 0 {
		t.Error("exit code = 0, want non-zero")
	}
	var resp protocol.Response
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("stdout is not a valid response document: %v\ngot: %s", err, stdout)
	}
	if len(resp.Errors) != 1 || resp.Errors[0].Code != protocol.CodeStateUnreadable {
		t.Errorf("errors = %+v, want one state_unreadable", resp.Errors)
	}
}

// End-to-end wiring: a request carrying one Basic sync target with no
// identifier is added, and the front end's write-back trigger (a non-nil
// note_id in an "added" result) is present in the response.
func TestSyncEndToEnd_AddsANoteThroughThePlanner(t *testing.T) {
	stub := newAnkiConnectStub(t)
	registryPath := filepath.Join(t.TempDir(), "registry.json")
	req := protocol.Request{
		ProtocolVersion: protocol.Version,
		Config: protocol.Config{
			DefaultDeck:    "Inbox",
			AnkiConnectURL: stub.URL,
			RegistryPath:   registryPath,
			SyncRoot:       t.TempDir(),
			ScanComplete:   true,
		},
		Census: []protocol.CensusEntry{},
		Entries: []protocol.Entry{
			{Key: "a.org::0", NoteID: nil, NoteType: "Basic", SourcePath: "a.org", Title: "T", Body: "B"},
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	code, stdout, _ := invoke(t, string(data), "sync")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout: %s", code, stdout)
	}

	var resp protocol.Response
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("stdout is not a valid response document: %v\ngot: %s", err, stdout)
	}
	if !resp.OK {
		t.Error("ok = false, want true")
	}
	if len(resp.Results) != 1 || resp.Results[0].Action != protocol.ActionAdded {
		t.Fatalf("results = %+v, want one added", resp.Results)
	}
	if resp.Results[0].NoteID == nil {
		t.Error("added result carries no note_id; the front end cannot write back what it never received")
	}

	// design.md §3 step 12: the registry is persisted to disk before the
	// response is written, reflecting the add.
	persisted, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("registry was not persisted to %s: %v", registryPath, err)
	}
	if !strings.Contains(string(persisted), fmt.Sprintf("%d", *resp.Results[0].NoteID)) {
		t.Errorf("persisted registry does not carry the added note's id: %s", persisted)
	}
}

func TestSyncReportsProtocolMismatchRatherThanProceeding(t *testing.T) {
	mismatched := strings.Replace(
		minimalRequest,
		`"protocol_version": 1`,
		`"protocol_version": 99`,
		1,
	)

	code, stdout, _ := invoke(t, mismatched, "sync")

	if code == 0 {
		t.Errorf("exit code = 0; a run-level failure exits non-zero so the front end's own taxonomy applies")
	}

	var resp protocol.Response
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("a mismatch must still emit a readable response document: %v\ngot: %s", err, stdout)
	}

	if resp.OK {
		t.Error("ok = true, want false on a protocol mismatch")
	}
	// The response echoes the version this binary speaks, not the incoming one:
	// the point of the document is to name what the binary can actually parse.
	if resp.ProtocolVersion != protocol.Version {
		t.Errorf("response protocol_version = %d, want this binary's own %d",
			resp.ProtocolVersion, protocol.Version)
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("len(errors) = %d, want 1", len(resp.Errors))
	}
	if resp.Errors[0].Code != protocol.CodeBinaryIncompatible {
		t.Errorf("errors[0].code = %q, want %q", resp.Errors[0].Code, protocol.CodeBinaryIncompatible)
	}
	if resp.Errors[0].Key != nil {
		t.Errorf("errors[0].key = %q, want nil for a run-level error", *resp.Errors[0].Key)
	}
	// The message is machine-oriented detail, but it must name both versions so
	// the mismatch is diagnosable by hand.
	if !strings.Contains(resp.Errors[0].Message, "99") || !strings.Contains(resp.Errors[0].Message, "1") {
		t.Errorf("errors[0].message = %q; want both the request and binary versions named",
			resp.Errors[0].Message)
	}
	if len(resp.Results) != 0 {
		t.Errorf("len(results) = %d, want 0 — nothing is processed on a mismatch", len(resp.Results))
	}
}

func TestProtocolMismatchIsNamedEvenWhenTheBodyShapeIsUnparseable(t *testing.T) {
	// design.md §2.1: protocol_version is checked before anything else is
	// parsed. The case the field exists for is a future request whose *shape*
	// this binary does not know, so a version mismatch must be named rather
	// than surfacing as a decode error against the current shape.
	future := `{"protocol_version": 99, "config": "not an object", "entries": 42}`

	code, stdout, _ := invoke(t, future, "sync")

	if code == 0 {
		t.Errorf("exit code = 0, want non-zero")
	}

	var resp protocol.Response
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("a version mismatch must emit a readable response, not a parse failure: %v\ngot: %s", err, stdout)
	}
	if resp.OK {
		t.Error("ok = true, want false")
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("len(errors) = %d, want 1", len(resp.Errors))
	}
	if resp.Errors[0].Code != protocol.CodeBinaryIncompatible {
		t.Errorf("errors[0].code = %q, want %q — the mismatch must be named, not reported as a decode error",
			resp.Errors[0].Code, protocol.CodeBinaryIncompatible)
	}
}

func TestSyncOnMalformedStdinFailsWithoutPanicking(t *testing.T) {
	code, stdout, stderr := invoke(t, "not json at all", "sync")

	if code == 0 {
		t.Errorf("exit code = 0, want non-zero for an unparseable request")
	}
	if strings.Contains(stdout, "panic:") || strings.Contains(stderr, "panic:") {
		t.Errorf("a malformed request must not panic\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("an unparseable request must leave a diagnostic on stderr")
	}
}

func TestUnknownSubcommandFailsWithGuidanceOnStderr(t *testing.T) {
	code, _, stderr := invoke(t, "", "frobnicate")

	if code == 0 {
		t.Errorf("exit code = 0, want non-zero for an unknown subcommand")
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("an unknown subcommand must leave a diagnostic on stderr")
	}
}

func TestNoArgumentsFailsWithGuidanceOnStderr(t *testing.T) {
	code, _, stderr := invoke(t, "")

	if code == 0 {
		t.Errorf("exit code = 0, want non-zero when no subcommand is given")
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("an empty invocation must leave a diagnostic on stderr")
	}
}

func TestDiagnosticsNeverContaminateStdout(t *testing.T) {
	// stdout carries the response document and nothing else; stderr is reserved
	// for diagnostics that are never shown verbatim to the user.
	for _, tc := range []struct {
		name  string
		stdin string
		args  []string
	}{
		{"unknown subcommand", "", []string{"frobnicate"}},
		{"no arguments", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, stdout, _ := invoke(t, tc.stdin, tc.args...)
			if strings.TrimSpace(stdout) != "" {
				t.Errorf("stdout carried non-response output: %q", stdout)
			}
		})
	}
}

// --- The delete path, end to end, through real AnkiConnect wire bytes ---
//
// plan.md D-9 step 2's ownership predicate is a chain: the values sent in
// an addNote request must come back out of a notesInfo response and
// recompute, through hashing.Hash, to the hash stored at add time. Two
// existing suites cover the ends of that chain and neither covers the
// join:
//
//   - internal/ankiconnect/client_test.go pins the notesInfo decode
//     against its own hardcoded response literals;
//   - internal/planner/delete_test.go pins the decision logic against an
//     in-process fake, seeding hand-built NoteInfo structs ("F1"/"B1")
//     and computing the registry hash from those SAME literals.
//
// Each side is therefore self-consistent by construction. Nothing asserts
// that the values one side WRITES are the values the other side READS —
// so a transform applied on the way out (or on the way back) keeps both
// suites green while silently making every candidate look "unowned",
// disabling deletion entirely. Verified: adding a strings.TrimSpace over
// the outgoing addNote field values leaves all six internal/... packages
// passing and fails only this test (go-org renders a trailing newline, so
// the stored value stops matching the hashed one).
//
// This test closes that join by running the real client against a
// recording httptest server: run 1 adds and the stub REMEMBERS what it
// received; run 2 replays those exact bytes back as a genuine notesInfo
// response. acceptance.md's own harness assumption ("a recording httptest
// server standing in for AnkiConnect, so that request sets can be
// asserted exactly") is what this finally supplies for the delete path.

// wireRecorder is a recording AnkiConnect stub speaking genuine wire JSON.
// It records every action it is asked for, and remembers what the addNote
// request actually carried, so the later notesInfo answer echoes those
// exact values back — which is what makes the hash recompute a real round
// trip rather than a restatement of the expected answer.
type wireRecorder struct {
	mu      sync.Mutex
	actions []string
	params  map[string][]json.RawMessage

	assignedID int
	modelName  string
	deckName   string
	fields     map[string]string
	tags       []string

	// models is the stub collection's note-type name set, answered by
	// modelNames. It is seeded per test so the install step's probe-then-act
	// branch can be driven from either side.
	//
	// It must be answered EXPLICITLY rather than left to the default arm: the
	// default returns a null result, which unmarshals into a []string as nil
	// with no error, so an untaught stub would silently look like an empty
	// collection and the absent branch would pass by accident.
	models []string
}

func newWireRecorder(assignedID int) *wireRecorder {
	return &wireRecorder{assignedID: assignedID, params: map[string][]json.RawMessage{}}
}

// reset clears the action log between runs, keeping what was recorded from
// the addNote request — run 2 asserts over its OWN request set.
func (w *wireRecorder) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.actions = nil
	w.params = map[string][]json.RawMessage{}
}

func (w *wireRecorder) snapshot() ([]string, map[string][]json.RawMessage) {
	w.mu.Lock()
	defer w.mu.Unlock()
	actions := append([]string(nil), w.actions...)
	params := map[string][]json.RawMessage{}
	for k, v := range w.params {
		params[k] = append([]json.RawMessage(nil), v...)
	}
	return actions, params
}

// server returns an httptest.Server answering in AnkiConnect's own envelope
// shape. Responses are produced with json.Marshal rather than string
// concatenation: the field values are go-org-rendered HTML, and a
// hand-escaped literal would corrupt them in ways that surface only as an
// unexplained hash mismatch.
func (w *wireRecorder) server(t *testing.T) *httptest.Server {
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

		w.mu.Lock()
		w.actions = append(w.actions, envelope.Action)
		w.params[envelope.Action] = append(w.params[envelope.Action], envelope.Params)
		result := w.answer(t, envelope.Action, envelope.Params)
		w.mu.Unlock()

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

// answer produces the "result" payload for one action. The caller holds w.mu.
func (w *wireRecorder) answer(t *testing.T, action string, params json.RawMessage) any {
	t.Helper()
	switch action {
	case "requestPermission":
		return map[string]any{"permission": "granted", "requireApiKey": false, "version": 6}

	case "deckNames":
		// acceptance.md §D preamble: the stub's collection starts holding
		// the configured default deck, so createDeck never fires
		// incidentally and inflates a request count.
		return []string{"Inbox"}

	case "modelFieldNames":
		// The planner resolves rendered field names against the note type's
		// real fields before dispatch. This stub answers with Anki's stock
		// names; the lowercase-field case a customized profile produces is
		// covered in internal/anki/planner/fieldnames_test.go.
		var p struct {
			ModelName string `json:"modelName"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			t.Fatalf("modelFieldNames params unmarshal: %v", err)
		}
		switch p.ModelName {
		case "Basic":
			return []string{"Front", "Back"}
		case "Cloze":
			return []string{"Text", "Extra"}
		default:
			t.Fatalf("modelFieldNames: unexpected model %q", p.ModelName)
			return nil
		}

	case "modelNames":
		if w.models == nil {
			return []string{}
		}
		return append([]string(nil), w.models...)

	case "createModel":
		// The created type joins the collection, so a second install against
		// this same stub observes it and takes the update branch — the shape
		// REQ-C-002.2's idempotence claim actually rests on.
		var p struct {
			ModelName string `json:"modelName"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			t.Errorf("stub: createModel params are not the documented shape: %v\ngot: %s", err, params)
			return nil
		}
		w.models = append(w.models, p.ModelName)
		return map[string]any{"id": 1234567890, "name": p.ModelName}

	case "updateModelStyling", "updateModelTemplates":
		return nil

	case "addNote":
		var p struct {
			Note struct {
				DeckName  string            `json:"deckName"`
				ModelName string            `json:"modelName"`
				Fields    map[string]string `json:"fields"`
				Tags      []string          `json:"tags"`
			} `json:"note"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			t.Errorf("stub: addNote params are not the documented shape: %v\ngot: %s", err, params)
			return nil
		}
		w.deckName = p.Note.DeckName
		w.modelName = p.Note.ModelName
		w.fields = p.Note.Fields
		w.tags = p.Note.Tags
		return w.assignedID

	case "notesInfo":
		var p struct {
			Notes []int `json:"notes"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			t.Errorf("stub: notesInfo params are not the documented shape: %v\ngot: %s", err, params)
			return nil
		}
		// The result array is positionally aligned with the request, and an
		// absent note is an empty object — the add-on's own documented
		// behavior (internal/ankiconnect's package comment).
		out := make([]map[string]any, 0, len(p.Notes))
		for _, id := range p.Notes {
			if id != w.assignedID {
				out = append(out, map[string]any{})
				continue
			}
			names := make([]string, 0, len(w.fields))
			for name := range w.fields {
				names = append(names, name)
			}
			sort.Strings(names)
			fields := map[string]any{}
			for i, name := range names {
				fields[name] = map[string]any{"value": w.fields[name], "order": i}
			}
			tags := w.tags
			if tags == nil {
				tags = []string{}
			}
			out = append(out, map[string]any{
				"noteId":    id,
				"modelName": w.modelName,
				"tags":      tags,
				"fields":    fields,
				"cards":     []int{5001},
			})
		}
		return out

	default:
		return nil
	}
}

// syncRequest builds a request document against a FIXED registry path, so
// run 2 reads exactly the registry run 1 persisted (buildRequest mints a
// fresh t.TempDir per call and so cannot be reused across runs).
func syncRequest(t *testing.T, ankiConnectURL, registryPath string, census []protocol.CensusEntry, entries []protocol.Entry) string {
	t.Helper()
	req := protocol.Request{
		ProtocolVersion: protocol.Version,
		Config: protocol.Config{
			DefaultDeck:     "Inbox",
			AnkiConnectURL:  ankiConnectURL,
			RegistryPath:    registryPath,
			SyncRoot:        t.TempDir(),
			ExcludePatterns: []string{},
			ScanComplete:    true,
		},
		Census:  census,
		Entries: entries,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("syncRequest: %v", err)
	}
	return string(data)
}

func equalActions(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestSyncEndToEnd_DeletesAnOrphanThroughRealNotesInfoWireShape(t *testing.T) {
	const noteID = 1001
	rec := newWireRecorder(noteID)
	stub := rec.server(t)
	registryPath := filepath.Join(t.TempDir(), "registry.json")

	// --- Run 1: add the note; the stub records exactly what addNote carried.
	entry := protocol.Entry{
		Key:        "a.org::0",
		NoteID:     nil,
		NoteType:   "Basic",
		SourcePath: "a.org",
		Title:      "Capital of France",
		Body:       "Paris.",
	}
	code, stdout, stderr := invoke(t, syncRequest(t, stub.URL, registryPath, nil, []protocol.Entry{entry}), "sync")
	if code != 0 {
		t.Fatalf("run 1: exit code = %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var addResp protocol.Response
	if err := json.Unmarshal([]byte(stdout), &addResp); err != nil {
		t.Fatalf("run 1: stdout is not a valid response document: %v\ngot: %s", err, stdout)
	}
	if len(addResp.Results) != 1 || addResp.Results[0].Action != protocol.ActionAdded {
		t.Fatalf("run 1: results = %+v, want one added", addResp.Results)
	}
	if addResp.Results[0].NoteID == nil || *addResp.Results[0].NoteID != noteID {
		t.Fatalf("run 1: added result note_id = %v, want %d", addResp.Results[0].NoteID, noteID)
	}
	if len(rec.fields) == 0 {
		t.Fatal("run 1: the stub recorded no addNote field values; run 2 would have nothing to echo back")
	}

	// --- Run 2: the heading is gone — no entry, and no census occurrence.
	rec.reset()
	code, stdout, stderr = invoke(t, syncRequest(t, stub.URL, registryPath, []protocol.CensusEntry{}, []protocol.Entry{}), "sync")
	if code != 0 {
		t.Fatalf("run 2: exit code = %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var delResp protocol.Response
	if err := json.Unmarshal([]byte(stdout), &delResp); err != nil {
		t.Fatalf("run 2: stdout is not a valid response document: %v\ngot: %s", err, stdout)
	}

	actions, params := rec.snapshot()

	// The load-bearing assertion: the write -> read -> recompute -> confirm
	// chain survived the real wire bytes. See this block's header comment
	// for why neither unit suite can fail in its place.
	deletes := params["deleteNotes"]
	if len(deletes) != 1 {
		t.Fatalf("run 2: recorded %d deleteNotes requests, want exactly 1\n"+
			"actions: %v\nresponse: %+v\n"+
			"addNote recorded deck %q, note type %q, fields %v, tags %v\n"+
			"A confirmation reporting the candidate unowned means the hash recomputed from "+
			"the notesInfo wire response does not match the hash stored at add time.",
			len(deletes), actions, delResp, rec.deckName, rec.modelName, rec.fields, rec.tags)
	}
	var deletePayload struct {
		Notes []int `json:"notes"`
	}
	if err := json.Unmarshal(deletes[0], &deletePayload); err != nil {
		t.Fatalf("run 2: deleteNotes params are not the documented shape: %v\ngot: %s", err, deletes[0])
	}
	if len(deletePayload.Notes) != 1 || deletePayload.Notes[0] != noteID {
		t.Errorf("run 2: deleteNotes carried %v, want exactly [%d] — REQ-016 requires the "+
			"identifier named explicitly, never a tag, query, or pattern scope",
			deletePayload.Notes, noteID)
	}

	// The confirmation query covered exactly the candidate set, and preceded
	// the delete (plan.md D-9: never delete on absence alone).
	if len(params["notesInfo"]) != 1 {
		t.Fatalf("run 2: recorded %d notesInfo requests, want exactly 1: %v", len(params["notesInfo"]), actions)
	}
	var infoPayload struct {
		Notes []int `json:"notes"`
	}
	if err := json.Unmarshal(params["notesInfo"][0], &infoPayload); err != nil {
		t.Fatalf("run 2: notesInfo params are not the documented shape: %v", err)
	}
	if len(infoPayload.Notes) != 1 || infoPayload.Notes[0] != noteID {
		t.Errorf("run 2: notesInfo covered %v, want exactly the candidate set [%d]", infoPayload.Notes, noteID)
	}
	if want := []string{"requestPermission", "notesInfo", "deleteNotes"}; !equalActions(actions, want) {
		t.Errorf("run 2: recorded actions = %v, want exactly %v — the confirmation must precede "+
			"the delete, and no note-mutating or deck-moving request belongs in a run whose "+
			"only work is one confirmed deletion", actions, want)
	}

	if len(delResp.Results) != 1 || delResp.Results[0].Action != protocol.ActionDeleted {
		t.Errorf("run 2: results = %+v, want one deleted", delResp.Results)
	}
	if len(delResp.Errors) != 0 {
		t.Errorf("run 2: errors = %+v, want none", delResp.Errors)
	}

	// design.md §3 step 12: the deletion is reflected in the persisted
	// registry, so a third run would find no candidate at all.
	persisted, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("run 2: registry not readable: %v", err)
	}
	if strings.Contains(string(persisted), fmt.Sprintf("%d", noteID)) {
		t.Errorf("run 2: the deleted note is still recorded in the registry: %s", persisted)
	}
}
