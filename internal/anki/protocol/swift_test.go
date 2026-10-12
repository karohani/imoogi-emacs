package protocol_test

import (
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// SPEC-ANKICARD-004 REQ-SW-002 — the new diagnostic is one more `code` value
// on the existing errors[] field, so the wire string is the whole contract and
// the protocol version does not move: this SPEC adds no field in either
// direction.
func TestSwiftArrowMissingCodeMatchesWireString(t *testing.T) {
	if protocol.CodeSwiftArrowMissing != "swift_arrow_missing" {
		t.Errorf("CodeSwiftArrowMissing = %q, want %q", protocol.CodeSwiftArrowMissing, "swift_arrow_missing")
	}
	if protocol.Version != 2 {
		t.Errorf("Version = %d, want 2 — a new code value is not a wire-shape change", protocol.Version)
	}
}
