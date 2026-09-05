package protocol_test

import (
	"encoding/json"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// The install request document (spec.md §2 "Install request document",
// design.md §6) is a NEW wire document, not a field added to an existing
// one — which is what keeps REQ-C-018's wire-stability clause true.
//
// Key spelling decision, recorded here because a test is the only place a
// wire spelling stays honest: design.md §6 writes the endpoint key as
// `ankiconnect_url`, while the shipped sync `Config` has spelled the very
// same value `anki_connect_url` since the parent SPEC. Two spellings for
// one value on one wire is a defect, and the front end already builds the
// sync spelling; design.md §6's form is therefore read as a typo and the
// existing `anki_connect_url` is authoritative.
const installRequestFixture = `{
  "protocol_version": 1,
  "anki_connect_url": "127.0.0.1:8765",
  "user_css": ".card { letter-spacing: 0.01em; }"
}`

func TestInstallRequestDecodesTheThreeDocumentedFields(t *testing.T) {
	var req protocol.InstallRequest
	if err := json.Unmarshal([]byte(installRequestFixture), &req); err != nil {
		t.Fatalf("install request fixture did not decode: %v", err)
	}
	if req.ProtocolVersion != 1 {
		t.Errorf("protocol_version = %d, want 1", req.ProtocolVersion)
	}
	if req.AnkiConnectURL != "127.0.0.1:8765" {
		t.Errorf("anki_connect_url = %q, want %q", req.AnkiConnectURL, "127.0.0.1:8765")
	}
	if want := ".card { letter-spacing: 0.01em; }"; req.UserCSS != want {
		t.Errorf("user_css = %q, want %q", req.UserCSS, want)
	}
}

// The install document's key spelling is asserted on the ENCODE side too,
// because the front end (M6) writes these keys and a silent rename here
// would break it with no compile error anywhere.
func TestInstallRequestCarriesExactlyThreeKeysWithTheSyncSpelling(t *testing.T) {
	encoded, err := json.Marshal(protocol.InstallRequest{
		ProtocolVersion: protocol.Version,
		AnkiConnectURL:  "127.0.0.1:8765",
		UserCSS:         "",
	})
	if err != nil {
		t.Fatalf("install request did not encode: %v", err)
	}

	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &keyed); err != nil {
		t.Fatalf("encoded install request did not decode as an object: %v", err)
	}

	want := []string{"protocol_version", "anki_connect_url", "user_css"}
	if len(keyed) != len(want) {
		t.Errorf("install request carries %d keys (%s), want exactly %d", len(keyed), encoded, len(want))
	}
	for _, key := range want {
		if _, ok := keyed[key]; !ok {
			t.Errorf("install request is missing key %q; encoded as %s", key, encoded)
		}
	}
	// The typo guard: the design-document spelling must NOT appear.
	if _, ok := keyed["ankiconnect_url"]; ok {
		t.Errorf("install request carries the design.md §6 spelling ankiconnect_url; the sync spelling anki_connect_url is authoritative")
	}
}

// An absent user stylesheet is carried as the empty string, never as a
// dropped key (REQ-C-008's "Where no such file exists" branch). No field in
// this package carries omitempty, and this asserts the install document
// keeps that property.
func TestInstallRequestKeepsUserCSSPresentWhenEmpty(t *testing.T) {
	encoded, err := json.Marshal(protocol.InstallRequest{ProtocolVersion: protocol.Version})
	if err != nil {
		t.Fatalf("install request did not encode: %v", err)
	}
	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &keyed); err != nil {
		t.Fatalf("encoded install request did not decode as an object: %v", err)
	}
	raw, ok := keyed["user_css"]
	if !ok {
		t.Fatalf("user_css key was dropped when empty; encoded as %s", encoded)
	}
	if string(raw) != `""` {
		t.Errorf("user_css = %s, want an empty JSON string", raw)
	}
}
