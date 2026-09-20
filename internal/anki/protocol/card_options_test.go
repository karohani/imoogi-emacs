package protocol_test

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// SPEC-ANKICARD-002 §3.2 — the wire contract for the three card-option
// properties. They travel as nullable STRINGS, always present, never
// interpreted on the way (REQ-OPT-004).

// AC-OPT-004a: the three keys are emitted even when every value is null. A
// missing key fails, because this contract holds that a dropped key and a
// null value are not interchangeable.
func TestEntryAlwaysEmitsTheThreeCardOptionKeys(t *testing.T) {
	entry := protocol.Entry{
		Key:        "a.org::0",
		NoteID:     nil,
		NoteType:   "imoogi-Cloze",
		SourcePath: "a.org",
		Title:      "A {{c1::cloze}} heading",
		Body:       "",
	}

	encoded, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("encoding entry: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("decoding the encoded entry: %v", err)
	}

	for _, key := range []string{"direction", "incremental", "swift"} {
		value, present := raw[key]
		if !present {
			t.Errorf("key %q is absent from the serialized entry; a dropped key and a null value are not interchangeable here", key)
			continue
		}
		if string(value) != "null" {
			t.Errorf("key %q = %s, want null for an unresolved option", key, value)
		}
	}
}

// AC-OPT-004b: a resolved value is a JSON string — including the falsy
// spelling, which is the case a boolean-typed wire would silently collapse
// into the same value as "no option at all".
func TestResolvedCardOptionsEncodeAsJSONStrings(t *testing.T) {
	incremental := "t"
	swift := "nil"
	entry := protocol.Entry{
		Key:         "a.org::0",
		NoteType:    "imoogi-Cloze",
		SourcePath:  "a.org",
		Incremental: &incremental,
		Swift:       &swift,
	}

	encoded, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("encoding entry: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("decoding the encoded entry: %v", err)
	}

	if got := string(raw["incremental"]); got != `"t"` {
		t.Errorf("incremental = %s, want the JSON string \"t\"", got)
	}
	if got := string(raw["swift"]); got != `"nil"` {
		t.Errorf("swift = %s, want the JSON string \"nil\" — a boolean wire would collapse this into the null case", got)
	}
	if got := string(raw["direction"]); got != "null" {
		t.Errorf("direction = %s, want null", got)
	}
}

// AC-OPT-004c: the shape round-trips with its null-ness intact.
func TestCardOptionsRoundTripPreservesNullness(t *testing.T) {
	const fixture = `{
	  "key": "a.org::0",
	  "note_id": null,
	  "note_type": "imoogi-Cloze",
	  "source_path": "a.org",
	  "deck": null,
	  "tags": [],
	  "title": "A {{c1::cloze}} heading",
	  "body": "",
	  "direction": null,
	  "incremental": "t",
	  "swift": "nil"
	}`

	var entry protocol.Entry
	if err := json.Unmarshal([]byte(fixture), &entry); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}

	if entry.Direction != nil {
		t.Errorf("direction = %q, want nil", *entry.Direction)
	}
	if entry.Incremental == nil || *entry.Incremental != "t" {
		t.Errorf("incremental = %v, want %q", entry.Incremental, "t")
	}
	if entry.Swift == nil || *entry.Swift != "nil" {
		t.Errorf("swift = %v, want the string %q, not a nil pointer", entry.Swift, "nil")
	}

	reencoded, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("re-encoding entry: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(reencoded, &raw); err != nil {
		t.Fatalf("decoding the re-encoded entry: %v", err)
	}
	if string(raw["direction"]) != "null" {
		t.Errorf("round-tripped direction = %s, want null", raw["direction"])
	}
	if string(raw["incremental"]) != `"t"` {
		t.Errorf("round-tripped incremental = %s, want \"t\"", raw["incremental"])
	}
	if string(raw["swift"]) != `"nil"` {
		t.Errorf("round-tripped swift = %s, want \"nil\"", raw["swift"])
	}
}

// AC-OPT-004d: the response document gains nothing. Pinned as an exact key
// set so a field added to Response fails here rather than being noticed in
// review.
func TestResponseKeySetIsUnchangedFromVersionOne(t *testing.T) {
	encoded, err := json.Marshal(protocol.Response{
		ProtocolVersion: protocol.Version,
		OK:              true,
		Results:         []protocol.Result{},
		Errors:          []protocol.Error{},
	})
	if err != nil {
		t.Fatalf("encoding response: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("decoding the encoded response: %v", err)
	}

	got := make([]string, 0, len(raw))
	for k := range raw {
		got = append(got, k)
	}
	sort.Strings(got)

	want := []string{"errors", "ok", "protocol_version", "results"}
	if len(got) != len(want) {
		t.Fatalf("response key set = %v, want the version-1 set %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("response key set = %v, want the version-1 set %v", got, want)
		}
	}
}

// AC-OPT-005's constant assertion lives in protocol_test.go's
// TestVersionConstantIsTwo, which already owned that pin — one pin, one test,
// rather than two that could be un-asserted separately.

// The three diagnostic codes this SPEC introduces. They are declared with the
// wire contract rather than with the gate that raises them, matching the
// const block's own convention: the front end's message table and its
// contract test key on the constant set, so a code added later than its table
// entry breaks the pairing either way.
func TestCardOptionErrorCodeConstantsMatchWireStrings(t *testing.T) {
	for _, tc := range []struct{ got, want string }{
		{protocol.CodeCardOptionInvalid, "card_option_invalid"},
		{protocol.CodeCardOptionConflict, "card_option_conflict"},
		{protocol.CodeCardOptionNeedsCloze, "card_option_needs_cloze"},
	} {
		if tc.got != tc.want {
			t.Errorf("code constant = %q, want %q", tc.got, tc.want)
		}
	}
}
