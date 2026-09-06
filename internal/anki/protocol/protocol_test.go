package protocol_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

const requestFixture = `{
  "protocol_version": 1,
  "config": {
    "default_deck": "Inbox",
    "anki_connect_url": "http://127.0.0.1:8765",
    "registry_path": "/home/u/notes/.imoogi-registry.json",
    "sync_root": "/home/u/notes",
    "exclude_patterns": ["drafts/", "archive/"],
    "scan_complete": true
  },
  "census": [
    {"note_id": 1001, "source_path": "a.org"},
    {"note_id": 1005, "source_path": "archive/moved.org"}
  ],
  "entries": [
    {
      "key": "a.org::0",
      "note_id": 1001,
      "note_type": "Basic",
      "source_path": "a.org",
      "deck": "Geography::Europe",
      "tags": ["geography", "europe"],
      "title": "Capital of France",
      "body": "Paris."
    },
    {
      "key": "b.org::0",
      "note_id": null,
      "note_type": "Cloze",
      "source_path": "b.org",
      "deck": null,
      "tags": [],
      "title": "France",
      "body": "The capital of France is {{c1::Paris}}."
    }
  ]
}`

func TestRequestDecodesWireFixture(t *testing.T) {
	var req protocol.Request
	if err := json.Unmarshal([]byte(requestFixture), &req); err != nil {
		t.Fatalf("decoding request fixture: %v", err)
	}

	if req.ProtocolVersion != 1 {
		t.Errorf("protocol_version = %d, want 1", req.ProtocolVersion)
	}
	if req.Config.DefaultDeck != "Inbox" {
		t.Errorf("config.default_deck = %q, want %q", req.Config.DefaultDeck, "Inbox")
	}
	if req.Config.AnkiConnectURL != "http://127.0.0.1:8765" {
		t.Errorf("config.anki_connect_url = %q", req.Config.AnkiConnectURL)
	}
	if req.Config.RegistryPath != "/home/u/notes/.imoogi-registry.json" {
		t.Errorf("config.registry_path = %q", req.Config.RegistryPath)
	}
	if req.Config.SyncRoot != "/home/u/notes" {
		t.Errorf("config.sync_root = %q", req.Config.SyncRoot)
	}
	if got, want := req.Config.ExcludePatterns, []string{"drafts/", "archive/"}; !equalStrings(got, want) {
		t.Errorf("config.exclude_patterns = %v, want %v", got, want)
	}
	if !req.Config.ScanComplete {
		t.Error("config.scan_complete = false, want true")
	}

	if len(req.Census) != 2 {
		t.Fatalf("len(census) = %d, want 2", len(req.Census))
	}
	if req.Census[0].NoteID != 1001 || req.Census[0].SourcePath != "a.org" {
		t.Errorf("census[0] = %+v", req.Census[0])
	}
	if req.Census[1].NoteID != 1005 || req.Census[1].SourcePath != "archive/moved.org" {
		t.Errorf("census[1] = %+v", req.Census[1])
	}

	if len(req.Entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(req.Entries))
	}

	first := req.Entries[0]
	if first.Key != "a.org::0" {
		t.Errorf("entries[0].key = %q", first.Key)
	}
	if first.NoteID == nil || *first.NoteID != 1001 {
		t.Errorf("entries[0].note_id = %v, want 1001", first.NoteID)
	}
	if first.NoteType != "Basic" {
		t.Errorf("entries[0].note_type = %q", first.NoteType)
	}
	if first.SourcePath != "a.org" {
		t.Errorf("entries[0].source_path = %q", first.SourcePath)
	}
	if first.Deck == nil || *first.Deck != "Geography::Europe" {
		t.Errorf("entries[0].deck = %v", first.Deck)
	}
	if got, want := first.Tags, []string{"geography", "europe"}; !equalStrings(got, want) {
		t.Errorf("entries[0].tags = %v, want %v", got, want)
	}
	if first.Title != "Capital of France" {
		t.Errorf("entries[0].title = %q", first.Title)
	}
	if first.Body != "Paris." {
		t.Errorf("entries[0].body = %q", first.Body)
	}

	// A null note_id is the REQ-009 add trigger and a null deck is the REQ-005
	// fallback trigger; both must decode as nil rather than a zero value.
	second := req.Entries[1]
	if second.NoteID != nil {
		t.Errorf("entries[1].note_id = %v, want nil", *second.NoteID)
	}
	if second.Deck != nil {
		t.Errorf("entries[1].deck = %v, want nil", *second.Deck)
	}
	if second.Body != "The capital of France is {{c1::Paris}}." {
		t.Errorf("entries[1].body = %q — cloze markers must survive decoding byte-for-byte", second.Body)
	}
}

func TestRequestRoundTripPreservesFields(t *testing.T) {
	var first protocol.Request
	if err := json.Unmarshal([]byte(requestFixture), &first); err != nil {
		t.Fatalf("decoding request fixture: %v", err)
	}

	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("encoding request: %v", err)
	}

	var second protocol.Request
	if err := json.Unmarshal(encoded, &second); err != nil {
		t.Fatalf("re-decoding request: %v", err)
	}

	if len(second.Entries) != len(first.Entries) || len(second.Census) != len(first.Census) {
		t.Fatalf("round trip changed collection lengths: %+v", second)
	}
	if second.Entries[0].NoteID == nil || *second.Entries[0].NoteID != 1001 {
		t.Errorf("round trip lost entries[0].note_id")
	}
	if second.Entries[1].NoteID != nil {
		t.Errorf("round trip turned a null note_id into %v", *second.Entries[1].NoteID)
	}
	if second.Config.SyncRoot != first.Config.SyncRoot {
		t.Errorf("round trip changed sync_root")
	}
}

func TestRequestEncodesSnakeCaseWireNames(t *testing.T) {
	var req protocol.Request
	if err := json.Unmarshal([]byte(requestFixture), &req); err != nil {
		t.Fatalf("decoding request fixture: %v", err)
	}

	encoded, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("encoding request: %v", err)
	}
	wire := string(encoded)

	for _, name := range []string{
		`"protocol_version"`, `"default_deck"`, `"anki_connect_url"`, `"registry_path"`,
		`"sync_root"`, `"exclude_patterns"`, `"scan_complete"`, `"census"`, `"note_id"`,
		`"source_path"`, `"entries"`, `"key"`, `"note_type"`, `"deck"`, `"tags"`,
		`"title"`, `"body"`,
	} {
		if !strings.Contains(wire, name) {
			t.Errorf("encoded request is missing wire field %s\ngot: %s", name, wire)
		}
	}

	// entries[1] carries a null note_id and a null deck. The keys must be
	// present carrying null — omitempty would drop them and the front end
	// cannot distinguish an absent key from an absent identifier.
	if !strings.Contains(wire, `"note_id":null`) {
		t.Errorf("a null note_id must encode as an explicit null, not a dropped key\ngot: %s", wire)
	}
	if !strings.Contains(wire, `"deck":null`) {
		t.Errorf("a null deck must encode as an explicit null, not a dropped key\ngot: %s", wire)
	}
	// entries[1] carries an empty tag set (REQ-006 / AC-014): it must encode as
	// [] rather than null, which a nil slice would produce.
	if !strings.Contains(wire, `"tags":[]`) {
		t.Errorf("an empty tag set must encode as [], not null\ngot: %s", wire)
	}
}

const responseFixture = `{
  "protocol_version": 1,
  "ok": true,
  "results": [
    {"key": "a.org::0", "action": "updated", "note_id": 1001},
    {"key": "b.org::0", "action": "added", "note_id": 1002},
    {"key": null, "action": "deleted", "note_id": 1003},
    {"key": "c.org::0", "action": "skipped", "note_id": null}
  ],
  "errors": [
    {"code": "anki_unreachable", "message": "no listener at 127.0.0.1:8765", "key": null},
    {"code": "cloze_marker_missing", "message": "no {{cN:: marker", "key": "c.org::0"}
  ]
}`

func TestResponseDecodesWireFixture(t *testing.T) {
	var resp protocol.Response
	if err := json.Unmarshal([]byte(responseFixture), &resp); err != nil {
		t.Fatalf("decoding response fixture: %v", err)
	}

	if resp.ProtocolVersion != 1 {
		t.Errorf("protocol_version = %d, want 1", resp.ProtocolVersion)
	}
	if !resp.OK {
		t.Error("ok = false, want true")
	}
	if len(resp.Results) != 4 {
		t.Fatalf("len(results) = %d, want 4", len(resp.Results))
	}
	if len(resp.Errors) != 2 {
		t.Fatalf("len(errors) = %d, want 2", len(resp.Errors))
	}

	if resp.Results[0].Key == nil || *resp.Results[0].Key != "a.org::0" {
		t.Errorf("results[0].key = %v", resp.Results[0].Key)
	}
	if resp.Results[0].Action != protocol.ActionUpdated {
		t.Errorf("results[0].action = %q, want %q", resp.Results[0].Action, protocol.ActionUpdated)
	}
	if resp.Results[1].Action != protocol.ActionAdded {
		t.Errorf("results[1].action = %q, want %q", resp.Results[1].Action, protocol.ActionAdded)
	}

	// design.md §2.2: a deleted result has no request-side entries[] item to
	// echo a key from, so results[].key is string-or-null, mirroring errors[].key.
	deleted := resp.Results[2]
	if deleted.Key != nil {
		t.Errorf("results[2].key = %q, want nil for a deleted result", *deleted.Key)
	}
	if deleted.Action != protocol.ActionDeleted {
		t.Errorf("results[2].action = %q, want %q", deleted.Action, protocol.ActionDeleted)
	}
	if deleted.NoteID == nil || *deleted.NoteID != 1003 {
		t.Errorf("results[2].note_id = %v — the deleted identifier is its only identifying field", deleted.NoteID)
	}

	skipped := resp.Results[3]
	if skipped.Action != protocol.ActionSkipped {
		t.Errorf("results[3].action = %q, want %q", skipped.Action, protocol.ActionSkipped)
	}
	if skipped.NoteID != nil {
		t.Errorf("results[3].note_id = %v, want nil", *skipped.NoteID)
	}

	if resp.Errors[0].Code != "anki_unreachable" {
		t.Errorf("errors[0].code = %q", resp.Errors[0].Code)
	}
	if resp.Errors[0].Message != "no listener at 127.0.0.1:8765" {
		t.Errorf("errors[0].message = %q", resp.Errors[0].Message)
	}
	if resp.Errors[0].Key != nil {
		t.Errorf("errors[0].key = %q, want nil for a run-level error", *resp.Errors[0].Key)
	}
	if resp.Errors[1].Key == nil || *resp.Errors[1].Key != "c.org::0" {
		t.Errorf("errors[1].key = %v, want the per-entry key", resp.Errors[1].Key)
	}
}

func TestResponseRoundTripPreservesFields(t *testing.T) {
	var first protocol.Response
	if err := json.Unmarshal([]byte(responseFixture), &first); err != nil {
		t.Fatalf("decoding response fixture: %v", err)
	}

	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("encoding response: %v", err)
	}

	var second protocol.Response
	if err := json.Unmarshal(encoded, &second); err != nil {
		t.Fatalf("re-decoding response: %v", err)
	}

	if len(second.Results) != 4 || len(second.Errors) != 2 {
		t.Fatalf("round trip changed collection lengths: %+v", second)
	}
	if second.Results[2].Key != nil {
		t.Errorf("round trip turned a null result key into %q", *second.Results[2].Key)
	}
	if second.Results[2].NoteID == nil || *second.Results[2].NoteID != 1003 {
		t.Errorf("round trip lost the deleted result's note_id")
	}
	if second.Errors[1].Key == nil || *second.Errors[1].Key != "c.org::0" {
		t.Errorf("round trip lost errors[1].key")
	}
}

func TestEmptyResponseEncodesEmptyArraysNotNull(t *testing.T) {
	// The minimal valid response. A nil slice marshals to null, which is a
	// different wire value from [] and parses differently on the Elisp side.
	resp := protocol.Response{
		ProtocolVersion: protocol.Version,
		OK:              true,
		Results:         []protocol.Result{},
		Errors:          []protocol.Error{},
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("encoding response: %v", err)
	}
	wire := string(encoded)

	if !strings.Contains(wire, `"results":[]`) {
		t.Errorf(`encoded response must contain "results":[] — got: %s`, wire)
	}
	if !strings.Contains(wire, `"errors":[]`) {
		t.Errorf(`encoded response must contain "errors":[] — got: %s`, wire)
	}
	if strings.Contains(wire, `null`) {
		t.Errorf("an empty response must carry no null: %s", wire)
	}
	if !strings.Contains(wire, `"ok":true`) {
		t.Errorf(`encoded response must contain "ok":true — got: %s`, wire)
	}
}

func TestVersionConstantIsOne(t *testing.T) {
	if protocol.Version != 1 {
		t.Errorf("protocol.Version = %d, want 1 (plan.md D-2: present from the first release)", protocol.Version)
	}
}

func TestActionConstantsMatchWireEnum(t *testing.T) {
	// design.md §2.2 results[].action enum.
	for wire, got := range map[string]string{
		"added":   protocol.ActionAdded,
		"updated": protocol.ActionUpdated,
		"skipped": protocol.ActionSkipped,
		"deleted": protocol.ActionDeleted,
		"failed":  protocol.ActionFailed,
	} {
		if got != wire {
			t.Errorf("action constant for %q = %q", wire, got)
		}
	}
}

// AC-C-018a — the migrate subcommand's wire stability. The card-styling SPEC
// adds exactly ONE thing to the wire: a sixth `action` VALUE. Not a field, not
// a document, not a version bump.
//
// The three assertions below are the three halves of that clause that can be
// checked mechanically from the Go side. The fourth — that the Elisp pinned
// version constant is likewise unchanged — is the contract test's, in M6.
func TestMigrateCandidateIsTheOnlyWireAddition(t *testing.T) {
	if protocol.ActionMigrateCandidate != "migrate_candidate" {
		t.Errorf("ActionMigrateCandidate = %q, want %q", protocol.ActionMigrateCandidate, "migrate_candidate")
	}

	// The version constant is unchanged. TestVersionConstantIsOne asserts
	// the same value for the parent SPEC's reason; this restates it under
	// REQ-C-018's own clause, because the two would have to be un-asserted
	// separately for the wire contract to drift silently.
	if protocol.Version != 1 {
		t.Errorf("protocol.Version = %d, want 1 — REQ-C-018 leaves it unchanged", protocol.Version)
	}

	// Result gained no field. A `migrate_candidate` result is an ordinary
	// result: {key, action, note_id} and nothing else, which is what lets
	// the candidate count be len(results) rather than a new response field.
	var got []string
	rt := reflect.TypeOf(protocol.Result{})
	for i := 0; i < rt.NumField(); i++ {
		got = append(got, rt.Field(i).Tag.Get("json"))
	}
	if want := []string{"key", "action", "note_id"}; !equalStrings(got, want) {
		t.Errorf("Result wire fields = %v, want exactly %v", got, want)
	}
}

func TestErrorCodeConstantsMatchWireStrings(t *testing.T) {
	// plan.md D-5 codes the Go back end emits. M1 needs only the
	// protocol-mismatch code; the rest arrive with their milestones.
	if protocol.CodeBinaryIncompatible != "binary_incompatible" {
		t.Errorf("CodeBinaryIncompatible = %q", protocol.CodeBinaryIncompatible)
	}
}

// design.md §5 — the four codes the note-type install, media, and migration
// paths emit. The constants land in M1 with the client surface they describe,
// ahead of the milestones that raise them, because the Elisp table and its
// contract test are keyed on the constant set rather than on its use.
func TestCardSPECErrorCodeConstantsMatchWireStrings(t *testing.T) {
	for _, tt := range []struct {
		got  string
		want string
	}{
		{protocol.CodeModelInstallFailed, "model_install_failed"},
		{protocol.CodeMediaFileNotFound, "media_file_not_found"},
		{protocol.CodeMediaUploadFailed, "media_upload_failed"},
		{protocol.CodeMigrationAddFailed, "migration_add_failed"},
	} {
		if tt.got != tt.want {
			t.Errorf("code = %q, want %q", tt.got, tt.want)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
