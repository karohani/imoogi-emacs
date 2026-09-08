package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/model"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// installRequest builds the install request document (protocol.InstallRequest)
// the front end writes to install-models' stdin.
func installRequest(t *testing.T, ankiConnectURL, userCSS string) string {
	t.Helper()
	data, err := json.Marshal(protocol.InstallRequest{
		ProtocolVersion: protocol.Version,
		AnkiConnectURL:  ankiConnectURL,
		UserCSS:         userCSS,
	})
	if err != nil {
		t.Fatalf("installRequest: %v", err)
	}
	return string(data)
}

// decodeResponse reads the one response document the binary writes to stdout.
func decodeResponse(t *testing.T, stdout string) protocol.Response {
	t.Helper()
	var resp protocol.Response
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &resp); err != nil {
		t.Fatalf("stdout is not a response document: %v\ngot: %q", err, stdout)
	}
	return resp
}

// AC-C-001 end to end, through genuine AnkiConnect wire JSON: a collection
// holding only the stock types takes the absent branch, and the exact action
// sequence is asserted rather than a count — the handshake first, one probe,
// then one createModel per type and nothing else.
func TestInstallModelsEndToEnd_CreatesBothTypesThroughRealWireShapes(t *testing.T) {
	rec := newWireRecorder(0)
	rec.models = []string{"Basic", "Cloze"}
	stub := rec.server(t)

	code, stdout, stderr := invoke(t, installRequest(t, stub.URL, ""), "install-models")
	if code != 0 {
		t.Fatalf("install-models exited %d; stderr: %s", code, stderr)
	}

	actions, params := rec.snapshot()
	want := []string{"requestPermission", "modelNames", "createModel", "createModel"}
	if !equalActions(actions, want) {
		t.Errorf("action sequence = %v, want %v", actions, want)
	}

	// The two createModel requests carry the shape AC-C-001 fixes, read off
	// the wire rather than off the Go struct that produced it.
	var seen []string
	for _, raw := range params["createModel"] {
		var p struct {
			ModelName     string   `json:"modelName"`
			InOrderFields []string `json:"inOrderFields"`
			CSS           string   `json:"css"`
			IsCloze       bool     `json:"isCloze"`
			CardTemplates []struct {
				Name  string `json:"Name"`
				Front string `json:"Front"`
				Back  string `json:"Back"`
			} `json:"cardTemplates"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatalf("createModel params are not the documented shape: %v\ngot: %s", err, raw)
		}
		seen = append(seen, p.ModelName)
		if p.CSS == "" {
			t.Errorf("createModel(%s) carried an empty css value", p.ModelName)
		}
		if len(p.CardTemplates) == 0 {
			t.Errorf("createModel(%s) carried no cardTemplates", p.ModelName)
		}
		// The deck hook reaches the wire as the stable wrapper class plus
		// the {{Deck}} carrier the template script normalizes at review
		// time (REQ-C-006); the raw `deck-{{Deck}}` form never does.
		for _, tpl := range p.CardTemplates {
			if !strings.Contains(tpl.Front, `class="imoogi-deck"`) || !strings.Contains(tpl.Front, `{{Deck}}`) {
				t.Errorf("createModel(%s) template %q front carries no deck wrapper", p.ModelName, tpl.Name)
			}
			if strings.Contains(tpl.Front, `class="deck-{{Deck}}"`) {
				t.Errorf("createModel(%s) template %q front still carries the raw deck wrapper", p.ModelName, tpl.Name)
			}
		}
		switch p.ModelName {
		case "imoogi-Basic":
			if strings.Join(p.InOrderFields, ",") != "Front,Back" {
				t.Errorf("imoogi-Basic inOrderFields = %v, want [Front Back]", p.InOrderFields)
			}
			if p.IsCloze {
				t.Error("imoogi-Basic was created with isCloze true")
			}
		case "imoogi-Cloze":
			if strings.Join(p.InOrderFields, ",") != "Text,Back Extra" {
				t.Errorf("imoogi-Cloze inOrderFields = %v, want [Text Back Extra]", p.InOrderFields)
			}
			if !p.IsCloze {
				t.Error("imoogi-Cloze was created with isCloze false")
			}
		default:
			t.Errorf("createModel named the unexpected model %q", p.ModelName)
		}
	}
	if strings.Join(seen, ",") != "imoogi-Basic,imoogi-Cloze" {
		t.Errorf("createModel order = %v, want imoogi-Basic then imoogi-Cloze", seen)
	}

	resp := decodeResponse(t, stdout)
	if !resp.OK {
		t.Errorf("response ok = false on a clean install: %+v", resp)
	}
	if resp.ProtocolVersion != protocol.Version {
		t.Errorf("response protocol_version = %d, want %d", resp.ProtocolVersion, protocol.Version)
	}
	if len(resp.Errors) != 0 {
		t.Errorf("response carries %d errors: %+v", len(resp.Errors), resp.Errors)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("response carries %d results, want 2: %+v", len(resp.Results), resp.Results)
	}
	for _, r := range resp.Results {
		if r.Action != protocol.ActionAdded {
			t.Errorf("result %+v action = %q, want %q", r, r.Action, protocol.ActionAdded)
		}
		if r.NoteID != nil {
			t.Errorf("result %+v carries a note_id; a note type has none", r)
		}
		if r.Key == nil || !model.IsOwned(*r.Key) {
			t.Errorf("result %+v is not keyed by an imoogi-owned model name", r)
		}
	}
}

// AC-C-002 end to end: a collection already holding both types takes the
// present branch — styling then templates per type, and zero createModel.
func TestInstallModelsEndToEnd_UpdatesPresentTypes(t *testing.T) {
	rec := newWireRecorder(0)
	rec.models = []string{"Basic", "Cloze", "imoogi-Basic", "imoogi-Cloze"}
	stub := rec.server(t)

	code, stdout, stderr := invoke(t, installRequest(t, stub.URL, ""), "install-models")
	if code != 0 {
		t.Fatalf("install-models exited %d; stderr: %s", code, stderr)
	}

	actions, _ := rec.snapshot()
	want := []string{
		"requestPermission", "modelNames",
		"updateModelStyling", "updateModelTemplates",
		"updateModelStyling", "updateModelTemplates",
	}
	if !equalActions(actions, want) {
		t.Errorf("action sequence = %v, want %v", actions, want)
	}

	resp := decodeResponse(t, stdout)
	for _, r := range resp.Results {
		if r.Action != protocol.ActionUpdated {
			t.Errorf("result %+v action = %q, want %q", r, r.Action, protocol.ActionUpdated)
		}
	}
}

// AC-C-002's idempotence clause at the process boundary: a second invocation
// against the same collection issues no createModel, because the first
// invocation's creations are what the second one's probe now sees.
func TestInstallModelsIsIdempotentAcrossTwoInvocations(t *testing.T) {
	rec := newWireRecorder(0)
	rec.models = []string{"Basic", "Cloze"}
	stub := rec.server(t)

	if code, _, stderr := invoke(t, installRequest(t, stub.URL, ""), "install-models"); code != 0 {
		t.Fatalf("first invocation exited %d; stderr: %s", code, stderr)
	}
	rec.reset()

	code, stdout, stderr := invoke(t, installRequest(t, stub.URL, ""), "install-models")
	if code != 0 {
		t.Fatalf("second invocation exited %d; stderr: %s", code, stderr)
	}

	actions, _ := rec.snapshot()
	want := []string{
		"requestPermission", "modelNames",
		"updateModelStyling", "updateModelTemplates",
		"updateModelStyling", "updateModelTemplates",
	}
	if !equalActions(actions, want) {
		t.Errorf("second invocation action sequence = %v, want %v", actions, want)
	}
	for _, r := range decodeResponse(t, stdout).Results {
		if r.Action != protocol.ActionUpdated {
			t.Errorf("second invocation reported %q, want %q", r.Action, protocol.ActionUpdated)
		}
	}
}

// AC-C-006a on the wire: the css the binary actually sends is the base
// stylesheet followed verbatim by the user stylesheet the request carried.
func TestInstallModelsCarriesTheUserStylesheetVerbatimOnTheWire(t *testing.T) {
	const marker = "/* USER MARKER */ .card { letter-spacing: 0.03em; }\n"
	rec := newWireRecorder(0)
	stub := rec.server(t)

	if code, _, stderr := invoke(t, installRequest(t, stub.URL, marker), "install-models"); code != 0 {
		t.Fatalf("install-models exited %d; stderr: %s", code, stderr)
	}

	_, params := rec.snapshot()
	if len(params["createModel"]) == 0 {
		t.Fatal("no createModel request was issued")
	}
	want := model.BaseCSS() + marker
	for _, raw := range params["createModel"] {
		var p struct {
			ModelName string `json:"modelName"`
			CSS       string `json:"css"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatalf("createModel params: %v", err)
		}
		if p.CSS != want {
			t.Errorf("createModel(%s) css is not base+user verbatim (%d bytes on the wire, %d expected)",
				p.ModelName, len(p.CSS), len(want))
		}
	}
}

// AC-C-003b / AC-C-022b on the install path, asserted over the real request
// log: no model-WRITE request names a model outside the imoogi- namespace,
// even though the collection holds the stock types, a user-authored type, and
// an imoogi-prefixed name that is not one of the two recognized ones.
func TestInstallModelsWritesNoForeignModelOnTheWire(t *testing.T) {
	rec := newWireRecorder(0)
	rec.models = []string{"Basic", "Cloze", "My Custom Type", "imoogi-Other"}
	stub := rec.server(t)

	if code, _, stderr := invoke(t, installRequest(t, stub.URL, ""), "install-models"); code != 0 {
		t.Fatalf("install-models exited %d; stderr: %s", code, stderr)
	}

	_, params := rec.snapshot()
	writes := 0
	for _, action := range []string{"createModel", "updateModelStyling", "updateModelTemplates"} {
		for _, raw := range params[action] {
			writes++
			name := modelNameOnTheWire(t, action, raw)
			if !model.IsOwned(name) {
				t.Errorf("%s named the foreign model %q", action, name)
			}
			if name != "imoogi-Basic" && name != "imoogi-Cloze" {
				t.Errorf("%s named %q, which is not one of the two recognized types", action, name)
			}
		}
	}
	if writes == 0 {
		t.Fatal("the install step issued no model-write request at all")
	}
}

// AC-C-003a's E2E mirror and the M1 fence at the process boundary: a plain
// sync run reaches none of the three model-write endpoints, and does not even
// probe modelNames.
func TestSyncEndToEnd_IssuesNoModelRequestOfAnyKind(t *testing.T) {
	rec := newWireRecorder(4242)
	stub := rec.server(t)

	code, _, stderr := invoke(t, buildRequest(t, stub.URL), "sync")
	if code != 0 {
		t.Fatalf("sync exited %d; stderr: %s", code, stderr)
	}

	actions, _ := rec.snapshot()
	for _, action := range actions {
		switch action {
		case "createModel", "updateModelStyling", "updateModelTemplates", "modelNames":
			t.Errorf("a synchronization run issued %q; the install surface is unreachable from sync", action)
		}
	}
}

// AC-C-007's announcement, and the standing invariant it must not break: the
// ownership notice reaches stderr and stdout stays a single response document.
func TestInstallModelsAnnouncesOwnershipOnStderrAndKeepsStdoutClean(t *testing.T) {
	rec := newWireRecorder(0)
	stub := rec.server(t)

	code, stdout, stderr := invoke(t, installRequest(t, stub.URL, ""), "install-models")
	if code != 0 {
		t.Fatalf("install-models exited %d; stderr: %s", code, stderr)
	}

	for _, want := range []string{"imoogi-Basic", "imoogi-Cloze", "replace"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the ownership announcement on stderr does not mention %q: %q", want, stderr)
		}
	}
	if strings.Contains(stdout, "replace") {
		t.Errorf("the ownership announcement contaminated stdout: %q", stdout)
	}
	decodeResponse(t, stdout) // stdout is exactly one response document
}

// The version probe is the same one runSync performs, and for the same
// reason: the case the field exists for is a request whose shape this binary
// does not know, and a full decode would fail on the shape before reaching
// the comparison.
func TestInstallModelsReportsProtocolMismatchRatherThanProceeding(t *testing.T) {
	rec := newWireRecorder(0)
	stub := rec.server(t)

	stdin := `{"protocol_version": 999, "anki_connect_url": "` + stub.URL + `", "user_css": ""}`
	code, stdout, _ := invoke(t, stdin, "install-models")
	if code == 0 {
		t.Errorf("a protocol mismatch exited 0, want non-zero")
	}

	resp := decodeResponse(t, stdout)
	if resp.OK {
		t.Error("a protocol mismatch reported ok true")
	}
	if len(resp.Errors) != 1 || resp.Errors[0].Code != protocol.CodeBinaryIncompatible {
		t.Fatalf("errors = %+v, want exactly one %s", resp.Errors, protocol.CodeBinaryIncompatible)
	}

	actions, _ := rec.snapshot()
	if len(actions) != 0 {
		t.Errorf("a protocol mismatch still reached AnkiConnect: %v", actions)
	}
}

// A dead endpoint is anki_unreachable, exactly as it is for sync — the same
// handshake, the same typed error, the same code. The install step is not
// where a second unreachability taxonomy gets invented.
func TestInstallModelsReportsAnkiUnreachableWhenNothingListens(t *testing.T) {
	// A port on the loopback interface with nothing bound to it. The sync
	// path's own unreachability test uses the same construction.
	code, stdout, _ := invoke(t, installRequest(t, "http://127.0.0.1:1", ""), "install-models")
	if code == 0 {
		t.Error("an unreachable endpoint exited 0, want non-zero")
	}
	resp := decodeResponse(t, stdout)
	if resp.OK {
		t.Error("an unreachable endpoint reported ok true")
	}
	if len(resp.Errors) != 1 || resp.Errors[0].Code != protocol.CodeAnkiUnreachable {
		t.Fatalf("errors = %+v, want exactly one %s", resp.Errors, protocol.CodeAnkiUnreachable)
	}
	if len(resp.Results) != 0 {
		t.Errorf("an unreachable endpoint still produced %d results", len(resp.Results))
	}
}

// Malformed stdin fails without panicking, and without reaching Anki.
func TestInstallModelsOnMalformedStdinFailsWithoutPanicking(t *testing.T) {
	code, _, stderr := invoke(t, "{not json", "install-models")
	if code == 0 {
		t.Error("malformed stdin exited 0, want non-zero")
	}
	if stderr == "" {
		t.Error("malformed stdin produced no diagnostic on stderr")
	}
}

// A host that answers without being AnkiConnect is ankiconnect_missing, not
// anki_unreachable — the same two-code distinction the sync path draws, drawn
// once in the shared handshake rather than twice.
func TestInstallModelsReportsAnkiConnectMissingWhenHostAnswersButIsNotAnkiConnect(t *testing.T) {
	notAnkiConnect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "not json at all")
	}))
	t.Cleanup(notAnkiConnect.Close)

	code, stdout, _ := invoke(t, installRequest(t, notAnkiConnect.URL, ""), "install-models")
	if code == 0 {
		t.Error("exit code = 0, want non-zero")
	}
	resp := decodeResponse(t, stdout)
	if len(resp.Errors) != 1 || resp.Errors[0].Code != protocol.CodeAnkiConnectMissing {
		t.Errorf("errors = %+v, want one %s", resp.Errors, protocol.CodeAnkiConnectMissing)
	}
}

// ok is false ONLY when nothing could be installed. Both types failing is
// that case, and it is the one that exits non-zero: the front end then falls
// back to its own error taxonomy rather than reading a half-truthful report.
func TestInstallModelsReportsNotOKWhenEveryTypeFails(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope struct {
			Action string `json:"action"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &envelope)
		w.Header().Set("Content-Type", "application/json")
		switch envelope.Action {
		case "requestPermission":
			_, _ = fmt.Fprint(w, `{"result": {"permission": "granted", "version": 6}, "error": null}`)
		case "modelNames":
			_, _ = fmt.Fprint(w, `{"result": [], "error": null}`)
		default:
			// AnkiConnect reports an action failure in the envelope's error
			// field with a null result, not as an HTTP status.
			_, _ = fmt.Fprint(w, `{"result": null, "error": "collection is read-only"}`)
		}
	}))
	t.Cleanup(failing.Close)

	code, stdout, _ := invoke(t, installRequest(t, failing.URL, ""), "install-models")
	if code == 0 {
		t.Error("exit code = 0 when nothing could be installed, want non-zero")
	}
	resp := decodeResponse(t, stdout)
	if resp.OK {
		t.Error("ok = true when nothing could be installed")
	}
	if len(resp.Results) != 0 {
		t.Errorf("results = %+v, want none", resp.Results)
	}
	if len(resp.Errors) != 2 {
		t.Fatalf("errors = %+v, want one per type", resp.Errors)
	}
	for _, e := range resp.Errors {
		if e.Code != protocol.CodeModelInstallFailed {
			t.Errorf("error %+v code = %q, want %q", e, e.Code, protocol.CodeModelInstallFailed)
		}
		if e.Key == nil || !model.IsOwned(*e.Key) {
			t.Errorf("error %+v is not keyed by an imoogi-owned model name", e)
		}
	}
}

// One type failing while the other succeeds is a PARTIAL success: ok stays
// true, the survivor is reported, and the exit code is 0 — the front end has
// something actionable, so it must not be told the run failed outright.
func TestInstallModelsReportsPartialSuccessAsOK(t *testing.T) {
	partial := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope struct {
			Action string          `json:"action"`
			Params json.RawMessage `json:"params"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &envelope)
		w.Header().Set("Content-Type", "application/json")
		switch envelope.Action {
		case "requestPermission":
			_, _ = fmt.Fprint(w, `{"result": {"permission": "granted", "version": 6}, "error": null}`)
		case "modelNames":
			_, _ = fmt.Fprint(w, `{"result": [], "error": null}`)
		case "createModel":
			if strings.Contains(string(envelope.Params), "imoogi-Cloze") {
				_, _ = fmt.Fprint(w, `{"result": null, "error": "cloze creation refused"}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"result": {"id": 1}, "error": null}`)
		default:
			_, _ = fmt.Fprint(w, `{"result": null, "error": null}`)
		}
	}))
	t.Cleanup(partial.Close)

	code, stdout, _ := invoke(t, installRequest(t, partial.URL, ""), "install-models")
	if code != 0 {
		t.Errorf("exit code = %d on a partial success, want 0", code)
	}
	resp := decodeResponse(t, stdout)
	if !resp.OK {
		t.Error("ok = false on a partial success; something WAS installed")
	}
	if len(resp.Results) != 1 || resp.Results[0].Key == nil || *resp.Results[0].Key != "imoogi-Basic" {
		t.Errorf("results = %+v, want exactly imoogi-Basic", resp.Results)
	}
	if len(resp.Errors) != 1 || resp.Errors[0].Key == nil || *resp.Errors[0].Key != "imoogi-Cloze" {
		t.Errorf("errors = %+v, want exactly imoogi-Cloze", resp.Errors)
	}
}

// An unreadable stdin is a failure of the transport, not of the request: no
// response document can be written because no request was received. Both
// subcommands take the same path, so both are exercised here.
func TestUnreadableStdinFailsWithoutPanicking(t *testing.T) {
	for _, subcommand := range []string{"sync", "install-models"} {
		t.Run(subcommand, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := run([]string{subcommand}, failingReader{}, &out, &errOut)
			if code == 0 {
				t.Errorf("%s exited 0 on an unreadable stdin, want non-zero", subcommand)
			}
			if !strings.Contains(errOut.String(), "could not be read") {
				t.Errorf("%s produced no read diagnostic on stderr: %q", subcommand, errOut.String())
			}
			if out.String() != "" {
				t.Errorf("%s wrote %q to stdout with no request to answer", subcommand, out.String())
			}
		})
	}
}

// failingReader is an io.Reader that cannot be read. It stands in for a
// closed pipe or a severed process boundary — the case the ReadAll error
// branch exists for, and one no string fixture can produce.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("stdin is not readable") }

// modelNameOnTheWire extracts the model name from a model-write request's
// params. The three actions genuinely differ in shape — createModel is flat,
// the two updates nest under "model" — and design.md §8.1 is explicit that
// the shapes must not be unified behind one serializer, so the test reads
// each as it actually is.
func modelNameOnTheWire(t *testing.T, action string, raw json.RawMessage) string {
	t.Helper()
	if action == "createModel" {
		var p struct {
			ModelName string `json:"modelName"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatalf("%s params: %v\ngot: %s", action, err, raw)
		}
		return p.ModelName
	}
	var p struct {
		Model struct {
			Name string `json:"name"`
		} `json:"model"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("%s params: %v\ngot: %s", action, err, raw)
	}
	return p.Model.Name
}
