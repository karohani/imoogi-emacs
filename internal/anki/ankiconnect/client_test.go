package ankiconnect

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// decodeRequest is a small test helper: it reads and decodes the request
// body the client sent, so tests can assert on the exact action + params
// shape without hardcoding raw JSON on both sides of the fixture.
func decodeRequest(t *testing.T, r *http.Request) rpcRequest {
	t.Helper()
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return req
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestHandshake_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "requestPermission" {
			t.Fatalf("action = %q, want requestPermission", req.Action)
		}
		writeJSON(t, w, map[string]any{
			"result": map[string]any{"permission": "granted", "requireApiKey": false, "version": 6},
			"error":  nil,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	if err := c.Handshake(context.Background()); err != nil {
		t.Fatalf("Handshake: %v", err)
	}
}

func TestHandshake_TransportError_NothingListening(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing is listening anymore — connection refused

	c := NewClient(url, nil)
	err := c.Handshake(context.Background())
	if err == nil {
		t.Fatalf("Handshake against a closed listener returned nil error")
	}
	var transportErr *TransportError
	if !errors.As(err, &transportErr) {
		t.Fatalf("Handshake error = %T (%v), want *TransportError (maps to anki_unreachable)", err, err)
	}
}

func TestHandshake_ProtocolError_RespondsButNotAnkiConnect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>not anki-connect</html>"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	err := c.Handshake(context.Background())
	if err == nil {
		t.Fatalf("Handshake against a non-AnkiConnect responder returned nil error")
	}
	var protoErr *ProtocolError
	if !errors.As(err, &protoErr) {
		t.Fatalf("Handshake error = %T (%v), want *ProtocolError (maps to ankiconnect_missing)", err, err)
	}
}

func TestHandshake_PermissionDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"result": map[string]any{"permission": "denied"},
			"error":  nil,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	err := c.Handshake(context.Background())
	var denied *PermissionDeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("Handshake error = %T (%v), want *PermissionDeniedError", err, err)
	}
}

func TestAPIError_SurfacesDistinctType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"result": nil, "error": "cannot create note because it is a duplicate"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	_, err := c.AddNote(context.Background(), "Default", "Basic", map[string]string{"Front": "f", "Back": "b"}, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("AddNote error = %T (%v), want *APIError (maps to ankiconnect_error)", err, err)
	}
	if apiErr.Message == "" {
		t.Fatalf("APIError.Message is empty")
	}
}

func TestDeckNames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "deckNames" {
			t.Fatalf("action = %q, want deckNames", req.Action)
		}
		writeJSON(t, w, map[string]any{"result": []string{"Default", "Geography::Europe"}, "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	names, err := c.DeckNames(context.Background())
	if err != nil {
		t.Fatalf("DeckNames: %v", err)
	}
	if len(names) != 2 || names[1] != "Geography::Europe" {
		t.Fatalf("DeckNames = %v, want [Default Geography::Europe]", names)
	}
}

func TestDeckNames_MalformedResultIsProtocolError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"result": "not-an-array", "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	_, err := c.DeckNames(context.Background())
	var protoErr *ProtocolError
	if !errors.As(err, &protoErr) {
		t.Fatalf("DeckNames malformed-result error = %T (%v), want *ProtocolError", err, err)
	}
}

func TestAddNote_MalformedResultIsProtocolError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"result": "not-a-number", "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	_, err := c.AddNote(context.Background(), "Default", "Basic", map[string]string{"Front": "f"}, nil)
	var protoErr *ProtocolError
	if !errors.As(err, &protoErr) {
		t.Fatalf("AddNote malformed-result error = %T (%v), want *ProtocolError", err, err)
	}
}

func TestUpdateNoteTags_NilTagsSentAsEmptyArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		params, ok := req.Params.(map[string]any)
		if !ok {
			t.Fatalf("params not a map: %#v", req.Params)
		}
		tags, ok := params["tags"].([]any)
		if !ok {
			t.Fatalf("tags = %#v, want an empty array (never null)", params["tags"])
		}
		if len(tags) != 0 {
			t.Fatalf("tags = %v, want empty", tags)
		}
		writeJSON(t, w, map[string]any{"result": nil, "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	if err := c.UpdateNoteTags(context.Background(), 1, nil); err != nil {
		t.Fatalf("UpdateNoteTags: %v", err)
	}
}

func TestCreateDeck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "createDeck" {
			t.Fatalf("action = %q, want createDeck", req.Action)
		}
		params, ok := req.Params.(map[string]any)
		if !ok || params["deck"] != "Geography::Europe" {
			t.Fatalf("params = %#v, want deck=Geography::Europe", req.Params)
		}
		writeJSON(t, w, map[string]any{"result": 1234567890, "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	if err := c.CreateDeck(context.Background(), "Geography::Europe"); err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
}

func TestAddNote_ReturnsAssignedNoteID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "addNote" {
			t.Fatalf("action = %q, want addNote", req.Action)
		}
		writeJSON(t, w, map[string]any{"result": 1502298033753, "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	id, err := c.AddNote(context.Background(), "Default", "Basic", map[string]string{"Front": "Q", "Back": "A"}, []string{"tag1"})
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}
	if id != 1502298033753 {
		t.Fatalf("AddNote id = %d, want 1502298033753", id)
	}
}

func TestUpdateNoteFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "updateNoteFields" {
			t.Fatalf("action = %q, want updateNoteFields", req.Action)
		}
		writeJSON(t, w, map[string]any{"result": nil, "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	if err := c.UpdateNoteFields(context.Background(), 1514547547030, map[string]string{"Front": "new"}); err != nil {
		t.Fatalf("UpdateNoteFields: %v", err)
	}
}

func TestUpdateNoteTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "updateNoteTags" {
			t.Fatalf("action = %q, want updateNoteTags", req.Action)
		}
		writeJSON(t, w, map[string]any{"result": nil, "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	if err := c.UpdateNoteTags(context.Background(), 1483959289817, []string{"european-languages"}); err != nil {
		t.Fatalf("UpdateNoteTags: %v", err)
	}
}

func TestChangeDeck_TakesCardIDsNotNoteIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "changeDeck" {
			t.Fatalf("action = %q, want changeDeck", req.Action)
		}
		params, ok := req.Params.(map[string]any)
		if !ok {
			t.Fatalf("params not a map: %#v", req.Params)
		}
		if _, hasCards := params["cards"]; !hasCards {
			t.Fatalf("changeDeck params missing 'cards' key: %#v", params)
		}
		if _, hasNotes := params["notes"]; hasNotes {
			t.Fatalf("changeDeck params must not carry a 'notes' key: %#v", params)
		}
		writeJSON(t, w, map[string]any{"result": nil, "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	if err := c.ChangeDeck(context.Background(), []int{1502098034045}, "Japanese::JLPT N3"); err != nil {
		t.Fatalf("ChangeDeck: %v", err)
	}
}

func TestDeleteNotes_ExplicitIDsOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "deleteNotes" {
			t.Fatalf("action = %q, want deleteNotes", req.Action)
		}
		params, ok := req.Params.(map[string]any)
		if !ok {
			t.Fatalf("params not a map: %#v", req.Params)
		}
		if _, hasNotes := params["notes"]; !hasNotes {
			t.Fatalf("deleteNotes params missing 'notes' key: %#v", params)
		}
		writeJSON(t, w, map[string]any{"result": nil, "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	if err := c.DeleteNotes(context.Background(), []int{1502298033753}); err != nil {
		t.Fatalf("DeleteNotes: %v", err)
	}
}

func TestNotesInfo_ExistingNote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "notesInfo" {
			t.Fatalf("action = %q, want notesInfo", req.Action)
		}
		writeJSON(t, w, map[string]any{
			"result": []map[string]any{
				{
					"noteId":    1502298033753,
					"profile":   "User_1",
					"modelName": "Basic",
					"tags":      []string{"tag", "another_tag"},
					"fields": map[string]any{
						"Front": map[string]any{"value": "front content", "order": 0},
						"Back":  map[string]any{"value": "back content", "order": 1},
					},
					"mod":   1718377864,
					"cards": []int{1498938915662},
				},
			},
			"error": nil,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	infos, err := c.NotesInfo(context.Background(), []int{1502298033753})
	if err != nil {
		t.Fatalf("NotesInfo: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("len(infos) = %d, want 1", len(infos))
	}
	got := infos[0]
	if !got.Exists {
		t.Fatalf("existing note reported Exists=false: %+v", got)
	}
	if got.NoteID != 1502298033753 {
		t.Fatalf("NoteID = %d, want 1502298033753", got.NoteID)
	}
	if got.ModelName != "Basic" {
		t.Fatalf("ModelName = %q, want Basic", got.ModelName)
	}
	if got.Fields["Front"].Value != "front content" {
		t.Fatalf("Fields[Front].Value = %q, want %q", got.Fields["Front"].Value, "front content")
	}
	if len(got.Cards) != 1 || got.Cards[0] != 1498938915662 {
		t.Fatalf("Cards = %v, want [1498938915662]", got.Cards)
	}
}

// TestNotesInfo_AbsentNoteReportsExistsFalse is the RED-captured
// verification of the run-phase research finding: notesInfo returns an
// EMPTY {} object — not an omission, not a null, not a sentinel noteId —
// for a note ID absent from the collection, confirmed by reading the
// add-on's own plugin/__init__.py notesInfo() handler (git.sr.ht
// ~foosoft/anki-connect, blob master/plugin/__init__.py). The response
// array's length and order match the request positionally, so this test
// asserts positional correspondence as well as absence detection.
func TestNotesInfo_AbsentNoteReportsExistsFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mirrors AnkiConnect's own response shape verbatim: one existing
		// note, one absent note represented as an empty object, in request
		// order (1004 does not exist — the AC-020 scenario from acceptance.md).
		writeJSON(t, w, map[string]any{
			"result": []map[string]any{
				{
					"noteId":    1001,
					"modelName": "Basic",
					"tags":      []string{},
					"fields": map[string]any{
						"Front": map[string]any{"value": "f", "order": 0},
						"Back":  map[string]any{"value": "b", "order": 1},
					},
					"mod":   1,
					"cards": []int{2001},
				},
				{}, // absent note ID 1004
			},
			"error": nil,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	infos, err := c.NotesInfo(context.Background(), []int{1001, 1004})
	if err != nil {
		t.Fatalf("NotesInfo: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("len(infos) = %d, want 2", len(infos))
	}
	if !infos[0].Exists || infos[0].NoteID != 1001 {
		t.Fatalf("infos[0] = %+v, want Exists=true NoteID=1001", infos[0])
	}
	if infos[1].Exists {
		t.Fatalf("infos[1] (absent note) reported Exists=true: %+v", infos[1])
	}
	if infos[1].NoteID != 1004 {
		t.Fatalf("infos[1].NoteID = %d, want 1004 (echoes the REQUESTED id, not the empty response object)", infos[1].NoteID)
	}
}

func TestNotesInfo_EmptyInputMakesNoCall(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	infos, err := c.NotesInfo(context.Background(), nil)
	if err != nil {
		t.Fatalf("NotesInfo with no IDs: %v", err)
	}
	if len(infos) != 0 {
		t.Fatalf("infos = %v, want empty", infos)
	}
	if called {
		t.Fatalf("NotesInfo with no IDs must not issue a round trip")
	}
}

func TestNotesInfo_ResultLengthMismatchIsProtocolError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"result": []map[string]any{{}}, "error": nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	_, err := c.NotesInfo(context.Background(), []int{1, 2, 3})
	var protoErr *ProtocolError
	if !errors.As(err, &protoErr) {
		t.Fatalf("NotesInfo length-mismatch error = %T (%v), want *ProtocolError", err, err)
	}
}

// TestClientImplementsAnkiConnector is a compile-time-shaped check that
// *Client satisfies the AnkiConnector interface internal/planner (a later
// milestone) will depend on.
func TestClientImplementsAnkiConnector(t *testing.T) {
	var _ AnkiConnector = (*Client)(nil)
}
