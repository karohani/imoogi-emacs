package protocol_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// SPEC-ANKISRC-001 AC-ASRC-008: org_id is optional. Present, it round-trips;
// absent, it decodes as the empty string and the encoded entry carries no
// org_id key at all, and Version stays 2.
func TestEntryOrgIDIsOptionalOnTheWire(t *testing.T) {
	if protocol.Version != 2 {
		t.Fatalf("Version = %d, want 2 (org_id must not bump the version)", protocol.Version)
	}

	var with protocol.Entry
	if err := json.Unmarshal([]byte(`{"key":"a.org::0","org_id":"6F1C-uuid"}`), &with); err != nil {
		t.Fatalf("decode with org_id: %v", err)
	}
	if with.OrgID != "6F1C-uuid" {
		t.Errorf("OrgID = %q, want the decoded value", with.OrgID)
	}
	encoded, err := json.Marshal(with)
	if err != nil {
		t.Fatalf("encode with org_id: %v", err)
	}
	if !strings.Contains(string(encoded), `"org_id":"6F1C-uuid"`) {
		t.Errorf("encoded entry %s lost org_id", encoded)
	}

	var without protocol.Entry
	if err := json.Unmarshal([]byte(`{"key":"a.org::0"}`), &without); err != nil {
		t.Fatalf("decode without org_id: %v", err)
	}
	if without.OrgID != "" {
		t.Errorf("OrgID = %q, want absent (empty)", without.OrgID)
	}
	encoded, err = json.Marshal(without)
	if err != nil {
		t.Fatalf("encode without org_id: %v", err)
	}
	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &keyed); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	if _, ok := keyed["org_id"]; ok {
		t.Errorf("encoded entry %s carries org_id although none was set", encoded)
	}
}

// DD-3: one character set shared by the planner, the open handler, and the
// Elisp receiver.
func TestValidOrgID(t *testing.T) {
	accepted := []string{
		"6F1C8E0A-3B2D-4C5E-9F10-123456789ABC", // uuid
		"imoogi:20261012T101500-abc",           // org method prefix:random
		"20261012T101500.123456",               // ts
		"20261012T101500.123456@host.example",  // org-id-include-domain
		strings.Repeat("a", 128),
	}
	for _, id := range accepted {
		if !protocol.ValidOrgID(id) {
			t.Errorf("ValidOrgID(%q) = false, want true", id)
		}
	}
	rejected := []string{
		"", "a b", "a%b", "a&b", "a=b", "a/b", `a"b`, "a\\b", "a\nb", "a\tb",
		strings.Repeat("a", 129), "한글",
	}
	for _, id := range rejected {
		if protocol.ValidOrgID(id) {
			t.Errorf("ValidOrgID(%q) = true, want false", id)
		}
	}
}

func TestOrgIDInvalidCodeIsDeclared(t *testing.T) {
	if protocol.CodeOrgIDInvalid != "org_id_invalid" {
		t.Errorf("CodeOrgIDInvalid = %q, want org_id_invalid", protocol.CodeOrgIDInvalid)
	}
}
