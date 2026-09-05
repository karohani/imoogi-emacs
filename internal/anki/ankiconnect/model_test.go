package ankiconnect

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// normalizeJSON round-trips a value through decode-and-re-encode so that both
// the params the client actually sent and the expected literal below are
// rendered the same canonical way (json.Marshal sorts map keys, so object
// field order — which carries no JSON meaning — cannot make an otherwise
// correct shape fail). What survives normalization is exactly what matters:
// the key set, the values, and array-vs-object nesting.
func normalizeJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	return string(out)
}

// paramsJSON renders the params the client actually sent, canonically.
func paramsJSON(t *testing.T, req rpcRequest) string {
	t.Helper()
	return normalizeJSON(t, req.Params)
}

// wantJSON renders an expected wire literal — written in design.md §8.1's own
// field order for readability — through the same canonical form.
func wantJSON(t *testing.T, literal string) string {
	t.Helper()
	var decoded any
	if err := json.Unmarshal([]byte(literal), &decoded); err != nil {
		t.Fatalf("expected literal is not valid JSON: %v", err)
	}
	return normalizeJSON(t, decoded)
}

// recordingServer answers every request with result/error and captures the
// decoded request the client sent.
func recordingServer(t *testing.T, result any, apiErr *string) (*httptest.Server, *rpcRequest) {
	t.Helper()
	var got rpcRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = decodeRequest(t, r)
		writeJSON(t, w, map[string]any{"result": result, "error": apiErr})
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

// ---- ModelNames: the install probe (REQ-C-002) ----

func TestModelNames_SendsModelNamesActionAndParsesTheNameList(t *testing.T) {
	srv, got := recordingServer(t, []string{"Basic", "Cloze", "imoogi-basic"}, nil)

	c := NewClient(srv.URL, nil)
	names, err := c.ModelNames(context.Background())
	if err != nil {
		t.Fatalf("ModelNames: %v", err)
	}

	if got.Action != "modelNames" {
		t.Errorf("action = %q, want modelNames", got.Action)
	}
	if got.Version != Version {
		t.Errorf("version = %d, want %d", got.Version, Version)
	}
	if got.Params != nil {
		t.Errorf("params = %v, want none — modelNames takes no parameters", got.Params)
	}
	want := []string{"Basic", "Cloze", "imoogi-basic"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestModelNames_MalformedResultIsProtocolError(t *testing.T) {
	srv, _ := recordingServer(t, "not-an-array", nil)

	c := NewClient(srv.URL, nil)
	_, err := c.ModelNames(context.Background())
	var protoErr *ProtocolError
	if !errors.As(err, &protoErr) {
		t.Fatalf("error = %T (%v), want *ProtocolError", err, err)
	}
}

// ---- CreateModel: flat params, cardTemplates is an ARRAY ----

func TestCreateModel_SendsFlatParamsWithCardTemplateArray(t *testing.T) {
	srv, got := recordingServer(t, map[string]any{"id": 1}, nil)

	c := NewClient(srv.URL, nil)
	err := c.CreateModel(context.Background(), "imoogi-basic",
		[]string{"Front", "Back"},
		".card { color: red; }",
		false,
		[]CardTemplate{{Name: "Card 1", Front: "{{Front}}", Back: "{{Back}}"}},
	)
	if err != nil {
		t.Fatalf("CreateModel: %v", err)
	}

	if got.Action != "createModel" {
		t.Errorf("action = %q, want createModel", got.Action)
	}
	const want = `{"cardTemplates":[{"Name":"Card 1","Front":"{{Front}}","Back":"{{Back}}"}],` +
		`"css":".card { color: red; }","inOrderFields":["Front","Back"],` +
		`"isCloze":false,"modelName":"imoogi-basic"}`
	if actual := paramsJSON(t, *got); actual != wantJSON(t, want) {
		t.Errorf("params =\n  %s\nwant\n  %s", actual, want)
	}
}

func TestCreateModel_ClozeFlagIsCarriedOnTheWire(t *testing.T) {
	srv, got := recordingServer(t, map[string]any{"id": 2}, nil)

	c := NewClient(srv.URL, nil)
	err := c.CreateModel(context.Background(), "imoogi-cloze",
		[]string{"Text", "Extra"}, "", true,
		[]CardTemplate{{Name: "Cloze", Front: "{{cloze:Text}}", Back: "{{cloze:Text}}<br>{{Extra}}"}},
	)
	if err != nil {
		t.Fatalf("CreateModel: %v", err)
	}

	var params map[string]any
	b, err := json.Marshal(got.Params)
	if err != nil {
		t.Fatalf("re-marshal params: %v", err)
	}
	if err := json.Unmarshal(b, &params); err != nil {
		t.Fatalf("decode params: %v", err)
	}
	if params["isCloze"] != true {
		t.Errorf("isCloze = %v, want true", params["isCloze"])
	}
}

func TestCreateModel_APIErrorSurfacesAsAPIError(t *testing.T) {
	msg := "Model name already exists"
	srv, _ := recordingServer(t, nil, &msg)

	c := NewClient(srv.URL, nil)
	err := c.CreateModel(context.Background(), "imoogi-basic", []string{"Front"}, "", false, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T (%v), want *APIError", err, err)
	}
	if apiErr.Action != "createModel" {
		t.Errorf("APIError.Action = %q, want createModel", apiErr.Action)
	}
}

// ---- UpdateModelStyling: params.model = {name, css} ----

func TestUpdateModelStyling_SendsNestedModelWithNameAndCSS(t *testing.T) {
	srv, got := recordingServer(t, nil, nil)

	c := NewClient(srv.URL, nil)
	if err := c.UpdateModelStyling(context.Background(), "imoogi-basic", ".card { font-size: 20px; }"); err != nil {
		t.Fatalf("UpdateModelStyling: %v", err)
	}

	if got.Action != "updateModelStyling" {
		t.Errorf("action = %q, want updateModelStyling", got.Action)
	}
	const want = `{"model":{"name":"imoogi-basic","css":".card { font-size: 20px; }"}}`
	if actual := paramsJSON(t, *got); actual != wantJSON(t, want) {
		t.Errorf("params =\n  %s\nwant\n  %s", actual, want)
	}
}

func TestUpdateModelStyling_APIErrorSurfacesAsAPIError(t *testing.T) {
	msg := "model was not found: imoogi-basic"
	srv, _ := recordingServer(t, nil, &msg)

	c := NewClient(srv.URL, nil)
	err := c.UpdateModelStyling(context.Background(), "imoogi-basic", "")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T (%v), want *APIError", err, err)
	}
}

// ---- UpdateModelTemplates: params.model.templates is a MAP keyed by card name ----

func TestUpdateModelTemplates_SendsTemplateMapKeyedByCardName(t *testing.T) {
	srv, got := recordingServer(t, nil, nil)

	c := NewClient(srv.URL, nil)
	err := c.UpdateModelTemplates(context.Background(), "imoogi-basic",
		[]CardTemplate{{Name: "Card 1", Front: "{{Front}}", Back: "{{Back}}"}},
	)
	if err != nil {
		t.Fatalf("UpdateModelTemplates: %v", err)
	}

	if got.Action != "updateModelTemplates" {
		t.Errorf("action = %q, want updateModelTemplates", got.Action)
	}
	// Deliberately distinct from createModel's cardTemplates ARRAY: here the
	// templates are an OBJECT keyed by card name, and each value carries only
	// Front/Back (design.md §8.1).
	const want = `{"model":{"name":"imoogi-basic","templates":{"Card 1":{"Front":"{{Front}}","Back":"{{Back}}"}}}}`
	if actual := paramsJSON(t, *got); actual != wantJSON(t, want) {
		t.Errorf("params =\n  %s\nwant\n  %s", actual, want)
	}
}

func TestUpdateModelTemplates_MultipleCardsEachKeyedByItsOwnName(t *testing.T) {
	srv, got := recordingServer(t, nil, nil)

	c := NewClient(srv.URL, nil)
	err := c.UpdateModelTemplates(context.Background(), "imoogi-basic", []CardTemplate{
		{Name: "Card 1", Front: "F1", Back: "B1"},
		{Name: "Card 2", Front: "F2", Back: "B2"},
	})
	if err != nil {
		t.Fatalf("UpdateModelTemplates: %v", err)
	}

	const want = `{"model":{"name":"imoogi-basic","templates":` +
		`{"Card 1":{"Front":"F1","Back":"B1"},"Card 2":{"Front":"F2","Back":"B2"}}}}`
	if actual := paramsJSON(t, *got); actual != wantJSON(t, want) {
		t.Errorf("params =\n  %s\nwant\n  %s", actual, want)
	}
}

func TestUpdateModelTemplates_APIErrorSurfacesAsAPIError(t *testing.T) {
	msg := "model was not found: imoogi-basic"
	srv, _ := recordingServer(t, nil, &msg)

	c := NewClient(srv.URL, nil)
	err := c.UpdateModelTemplates(context.Background(), "imoogi-basic", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T (%v), want *APIError", err, err)
	}
}

// ---- StoreMediaFile: filename + local absolute path, returns stored name ----

func TestStoreMediaFile_SendsFilenameAndPathAndReturnsStoredName(t *testing.T) {
	srv, got := recordingServer(t, "imoogi-abc123.png", nil)

	c := NewClient(srv.URL, nil)
	stored, err := c.StoreMediaFile(context.Background(), "imoogi-abc123.png", "/home/u/notes/img/diagram.png")
	if err != nil {
		t.Fatalf("StoreMediaFile: %v", err)
	}

	if got.Action != "storeMediaFile" {
		t.Errorf("action = %q, want storeMediaFile", got.Action)
	}
	// "path" is used rather than "data", so no base64 encoding is needed for a
	// local file (design.md §8.1).
	const want = `{"filename":"imoogi-abc123.png","path":"/home/u/notes/img/diagram.png"}`
	if actual := paramsJSON(t, *got); actual != wantJSON(t, want) {
		t.Errorf("params =\n  %s\nwant\n  %s", actual, want)
	}
	if stored != "imoogi-abc123.png" {
		t.Errorf("stored name = %q, want imoogi-abc123.png", stored)
	}
}

func TestStoreMediaFile_AnkiRenamedTheFileAndTheNewNameIsReturned(t *testing.T) {
	// AnkiConnect answers with the name it actually stored, which need not
	// equal the requested one when the collection already holds a different
	// file under that name.
	srv, _ := recordingServer(t, "imoogi-abc123-2.png", nil)

	c := NewClient(srv.URL, nil)
	stored, err := c.StoreMediaFile(context.Background(), "imoogi-abc123.png", "/tmp/x.png")
	if err != nil {
		t.Fatalf("StoreMediaFile: %v", err)
	}
	if stored != "imoogi-abc123-2.png" {
		t.Errorf("stored name = %q, want the name Anki reported back", stored)
	}
}

func TestStoreMediaFile_MalformedResultIsProtocolError(t *testing.T) {
	srv, _ := recordingServer(t, []string{"not", "a", "string"}, nil)

	c := NewClient(srv.URL, nil)
	_, err := c.StoreMediaFile(context.Background(), "f.png", "/tmp/f.png")
	var protoErr *ProtocolError
	if !errors.As(err, &protoErr) {
		t.Fatalf("error = %T (%v), want *ProtocolError", err, err)
	}
}

func TestStoreMediaFile_TransportErrorWhenNothingIsListening(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing is listening anymore — connection refused

	c := NewClient(url, nil)
	_, err := c.StoreMediaFile(context.Background(), "f.png", "/tmp/f.png")
	var transportErr *TransportError
	if !errors.As(err, &transportErr) {
		t.Fatalf("error = %T (%v), want *TransportError", err, err)
	}
}

// ---- The interface surface itself ----

func TestAnkiConnectorInterfaceCarriesTheFiveNewModelAndMediaMethods(t *testing.T) {
	// The five additions must be reachable through the INTERFACE, not merely
	// on the concrete client — the planner and the install step both hold an
	// AnkiConnector, never a *Client. Each call below fails to compile if a
	// signature drifts from design.md §8.1, and the recorded action names
	// prove the dispatch actually reached the wire.
	var actions []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		actions = append(actions, req.Action)
		// storeMediaFile and modelNames are the two with a typed result; a
		// string satisfies the former and an empty array the latter, so one
		// handler serves every action here.
		if req.Action == "storeMediaFile" {
			writeJSON(t, w, map[string]any{"result": "f.png", "error": nil})
			return
		}
		writeJSON(t, w, map[string]any{"result": []string{}, "error": nil})
	}))
	defer srv.Close()

	var conn AnkiConnector = NewClient(srv.URL, nil)
	ctx := context.Background()

	if _, err := conn.ModelNames(ctx); err != nil {
		t.Fatalf("ModelNames via interface: %v", err)
	}
	if err := conn.CreateModel(ctx, "imoogi-basic", []string{"Front"}, "", false, nil); err != nil {
		t.Fatalf("CreateModel via interface: %v", err)
	}
	if err := conn.UpdateModelStyling(ctx, "imoogi-basic", ""); err != nil {
		t.Fatalf("UpdateModelStyling via interface: %v", err)
	}
	if err := conn.UpdateModelTemplates(ctx, "imoogi-basic", nil); err != nil {
		t.Fatalf("UpdateModelTemplates via interface: %v", err)
	}
	if _, err := conn.StoreMediaFile(ctx, "f.png", "/tmp/f.png"); err != nil {
		t.Fatalf("StoreMediaFile via interface: %v", err)
	}

	want := []string{"modelNames", "createModel", "updateModelStyling", "updateModelTemplates", "storeMediaFile"}
	if len(actions) != len(want) {
		t.Fatalf("actions = %v, want %v", actions, want)
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Errorf("actions[%d] = %q, want %q", i, actions[i], want[i])
		}
	}
}
