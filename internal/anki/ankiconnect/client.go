// Package ankiconnect is the only component in this binary permitted to
// reach AnkiConnect (spec.md C-6). Every exported method returns typed,
// distinguishable errors (TransportError / ProtocolError / APIError) so a
// caller — internal/planner, added in a later milestone — can map failures
// to the plan.md D-5 diagnostic-code table without re-deriving transport
// detail.
//
// Action names and parameter shapes below are verified against two primary
// sources fetched directly for this milestone, the same way research.md's
// authors fetched changeDeck/notesInfo/deleteNotes/addNote:
//
//   - git.sr.ht ~foosoft/anki-connect, blob master/README.md (§Note Actions)
//     — confirms updateNoteFields and updateNoteTags exist as separate
//     actions with the parameter shapes used below (research.md §3 does not
//     cover either; both are independently verified here).
//   - git.sr.ht ~foosoft/anki-connect, blob master/plugin/__init__.py,
//     the notesInfo() handler — resolves research.md §3.2's flagged
//     unresolved question: a note ID absent from the collection raises
//     Anki's own NotFoundError internally, which the handler catches and
//     appends an EMPTY {} object to the result array in that note's
//     position (handler's own comment: "Best behavior is probably to add
//     an 'empty card' to the returned result, so that the items of the
//     input and return lists correspond."). See NotesInfo below.
package ankiconnect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Version is the AnkiConnect JSON-RPC protocol version this client speaks,
// matching every sample request in the add-on's own README (research.md
// §3).
const Version = 6

// AnkiConnector is the surface internal/planner (a later milestone) drives.
// *Client implements it against a real AnkiConnect endpoint; a test double
// implements it against nothing at all, per this milestone's requirement
// that the planner be testable without real Anki or real network calls.
type AnkiConnector interface {
	Handshake(ctx context.Context) error
	DeckNames(ctx context.Context) ([]string, error)
	CreateDeck(ctx context.Context, name string) error
	ModelFieldNames(ctx context.Context, modelName string) ([]string, error)
	AddNote(ctx context.Context, deck, modelName string, fields map[string]string, tags []string) (int, error)
	UpdateNoteFields(ctx context.Context, noteID int, fields map[string]string) error
	UpdateNoteTags(ctx context.Context, noteID int, tags []string) error
	NotesInfo(ctx context.Context, noteIDs []int) ([]NoteInfo, error)
	ChangeDeck(ctx context.Context, cardIDs []int, deck string) error
	DeleteNotes(ctx context.Context, noteIDs []int) error
}

var _ AnkiConnector = (*Client)(nil)

// Client is an HTTP client for one AnkiConnect endpoint (config.anki_connect_url,
// design.md §2.1).
type Client struct {
	url        string
	httpClient *http.Client
}

// NewClient returns a Client for the given AnkiConnect URL. httpClient is
// optional; a nil value gets a default with a bounded timeout so a hung
// request cannot stall a sync run indefinitely.
func NewClient(url string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{url: url, httpClient: httpClient}
}

type rpcRequest struct {
	Action  string `json:"action"`
	Version int    `json:"version"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *string         `json:"error"`
}

// call issues one JSON-RPC request and returns the raw "result" payload.
// Every exported method builds on this so the three error tiers are
// produced in exactly one place.
func (c *Client) call(ctx context.Context, action string, params any) (json.RawMessage, error) {
	reqBody, err := json.Marshal(rpcRequest{Action: action, Version: Version, Params: params})
	if err != nil {
		return nil, fmt.Errorf("ankiconnect: %s: encode request: %w", action, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("ankiconnect: %s: build request: %w", action, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &TransportError{Op: action, URL: c.url, Err: err}
	}
	// Closing a response body that has already been read to completion can
	// only fail in ways this client cannot act on, and the read below is
	// where a genuine transport failure surfaces as a *TransportError.
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &TransportError{Op: action, URL: c.url, Err: err}
	}

	var envelope rpcResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, &ProtocolError{Op: action, Body: string(body), Err: err}
	}

	if envelope.Error != nil {
		return nil, &APIError{Action: action, Message: *envelope.Error}
	}

	return envelope.Result, nil
}

type permissionResult struct {
	Permission string `json:"permission"`
	Version    int    `json:"version"`
}

// Handshake performs AnkiConnect's own recommended first call
// (requestPermission, research.md §3.5) so the two distinct failure modes
// plan.md D-5 names can be told apart before any other request is
// attempted: TransportError (anki_unreachable — nothing is listening) vs
// ProtocolError (ankiconnect_missing — something answered, but not
// AnkiConnect).
func (c *Client) Handshake(ctx context.Context) error {
	raw, err := c.call(ctx, "requestPermission", nil)
	if err != nil {
		return err // already *TransportError, *ProtocolError, or *APIError
	}
	var result permissionResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return &ProtocolError{Op: "requestPermission", Body: string(raw), Err: err}
	}
	if result.Permission != "granted" {
		return &PermissionDeniedError{}
	}
	return nil
}

// DeckNames lists every deck in the collection (action "deckNames").
func (c *Client) DeckNames(ctx context.Context) ([]string, error) {
	raw, err := c.call(ctx, "deckNames", nil)
	if err != nil {
		return nil, err
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil, &ProtocolError{Op: "deckNames", Body: string(raw), Err: err}
	}
	return names, nil
}

// CreateDeck creates name if it does not already exist (action
// "createDeck"). Documented as a no-op on an already-existing deck
// (research.md §3.5), so callers do not need to pre-check existence.
func (c *Client) CreateDeck(ctx context.Context, name string) error {
	_, err := c.call(ctx, "createDeck", map[string]any{"deck": name})
	return err
}

type modelFieldNamesParams struct {
	ModelName string `json:"modelName"`
}

// ModelFieldNames returns the field names a note type actually carries
// (action "modelFieldNames"). Field names are user-editable in Anki, so a
// profile's "Basic" may well be front/back rather than the stock
// Front/Back; the planner resolves rendered field names against this
// before dispatching, because AnkiConnect's updateNoteFields accepts an
// unknown field name, answers error:null, and silently changes nothing.
func (c *Client) ModelFieldNames(ctx context.Context, modelName string) ([]string, error) {
	raw, err := c.call(ctx, "modelFieldNames", modelFieldNamesParams{ModelName: modelName})
	if err != nil {
		return nil, err
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil, &ProtocolError{Op: "modelFieldNames", Body: string(raw), Err: err}
	}
	return names, nil
}

type addNoteParams struct {
	Note addNoteBody `json:"note"`
}

type addNoteBody struct {
	DeckName  string            `json:"deckName"`
	ModelName string            `json:"modelName"`
	Fields    map[string]string `json:"fields"`
	Tags      []string          `json:"tags"`
}

// AddNote creates a note with the given deck, note type, field values, and
// tags (action "addNote", research.md §3.4), returning the identifier
// AnkiConnect assigned. A nil tags slice is sent as an empty array, never
// as null, matching REQ-006's "shall assign no tags when that value
// resolves empty or absent" — an empty set, not an absent field.
func (c *Client) AddNote(ctx context.Context, deck, modelName string, fields map[string]string, tags []string) (int, error) {
	if tags == nil {
		tags = []string{}
	}
	raw, err := c.call(ctx, "addNote", addNoteParams{Note: addNoteBody{
		DeckName:  deck,
		ModelName: modelName,
		Fields:    fields,
		Tags:      tags,
	}})
	if err != nil {
		return 0, err
	}
	var noteID int
	if err := json.Unmarshal(raw, &noteID); err != nil {
		return 0, &ProtocolError{Op: "addNote", Body: string(raw), Err: err}
	}
	return noteID, nil
}

type updateNoteFieldsParams struct {
	Note updateNoteFieldsBody `json:"note"`
}

type updateNoteFieldsBody struct {
	ID     int               `json:"id"`
	Fields map[string]string `json:"fields"`
}

// UpdateNoteFields sets a note's field values by note ID (action
// "updateNoteFields"; params.note = {id, fields}).
func (c *Client) UpdateNoteFields(ctx context.Context, noteID int, fields map[string]string) error {
	_, err := c.call(ctx, "updateNoteFields", updateNoteFieldsParams{Note: updateNoteFieldsBody{ID: noteID, Fields: fields}})
	return err
}

type updateNoteTagsParams struct {
	Note int      `json:"note"`
	Tags []string `json:"tags"`
}

// UpdateNoteTags sets a note's tags by note ID (action "updateNoteTags";
// params = {note: <id>, tags: [...]}) — old tags are removed and replaced
// with the given set, not merged with it.
func (c *Client) UpdateNoteTags(ctx context.Context, noteID int, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	_, err := c.call(ctx, "updateNoteTags", updateNoteTagsParams{Note: noteID, Tags: tags})
	return err
}

// FieldValue is one field's rendered content and display order, as
// notesInfo returns it (research.md §3.2).
type FieldValue struct {
	Value string `json:"value"`
	Order int    `json:"order"`
}

// NoteInfo is one notesInfo result, positionally aligned with the
// requested note ID. NoteID always echoes the REQUESTED id — not the
// response's own noteId field, which AnkiConnect omits (leaves at its zero
// value) for a note that does not exist. See NotesInfo for the absent-ID
// shape this is built to tolerate.
type NoteInfo struct {
	NoteID    int
	Exists    bool
	ModelName string
	Fields    map[string]FieldValue
	Tags      []string
	Cards     []int
}

type rawNoteInfo struct {
	NoteID    int                   `json:"noteId"`
	ModelName string                `json:"modelName"`
	Tags      []string              `json:"tags"`
	Fields    map[string]FieldValue `json:"fields"`
	Cards     []int                 `json:"cards"`
}

// NotesInfo queries a set of note identifiers in one call (action
// "notesInfo", research.md §3.2, plan.md D-9's ownership predicate). An
// empty noteIDs makes no round trip at all and returns (nil, nil).
//
// Absent-ID behavior (see the package doc comment for the source): the
// add-on's own notesInfo() handler appends an EMPTY {} object — no noteId,
// no modelName, no fields — to the result array, in the absent note's
// request position, when the note ID does not exist in the collection.
// This method computes Exists from whichever discriminating field the
// response actually carries (ModelName, Fields, or a non-zero NoteID),
// tolerating that exact shape as well as any of the alternatives the
// unresolved research question considered (a zero-value noteId, or
// omission padded by a caller upstream) — any of those collapses to the
// same all-zero-value struct on decode.
func (c *Client) NotesInfo(ctx context.Context, noteIDs []int) ([]NoteInfo, error) {
	if len(noteIDs) == 0 {
		return nil, nil
	}
	raw, err := c.call(ctx, "notesInfo", map[string]any{"notes": noteIDs})
	if err != nil {
		return nil, err
	}
	var rawResults []rawNoteInfo
	if err := json.Unmarshal(raw, &rawResults); err != nil {
		return nil, &ProtocolError{Op: "notesInfo", Body: string(raw), Err: err}
	}
	if len(rawResults) != len(noteIDs) {
		return nil, &ProtocolError{
			Op:   "notesInfo",
			Body: string(raw),
			Err:  fmt.Errorf("expected %d results, got %d", len(noteIDs), len(rawResults)),
		}
	}

	out := make([]NoteInfo, len(noteIDs))
	for i, r := range rawResults {
		exists := r.ModelName != "" || len(r.Fields) > 0 || r.NoteID != 0
		out[i] = NoteInfo{
			NoteID:    noteIDs[i],
			Exists:    exists,
			ModelName: r.ModelName,
			Fields:    r.Fields,
			Tags:      r.Tags,
			Cards:     r.Cards,
		}
	}
	return out, nil
}

// ChangeDeck moves the given CARD identifiers — never note identifiers —
// into deck, creating the deck if it does not already exist (action
// "changeDeck", research.md §3.1). AnkiConnect has no note-scoped deck
// change: a deck is a property of a card, not of a note (plan.md D-10).
func (c *Client) ChangeDeck(ctx context.Context, cardIDs []int, deck string) error {
	_, err := c.call(ctx, "changeDeck", map[string]any{"cards": cardIDs, "deck": deck})
	return err
}

// DeleteNotes deletes the given note identifiers explicitly (action
// "deleteNotes", research.md §3.3). AnkiConnect has no query-, tag-, or
// pattern-scoped delete action for this method to accidentally reach for
// (spec.md REQ-016).
func (c *Client) DeleteNotes(ctx context.Context, noteIDs []int) error {
	_, err := c.call(ctx, "deleteNotes", map[string]any{"notes": noteIDs})
	return err
}
