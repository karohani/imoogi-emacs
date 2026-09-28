package protocol_test

import (
	"testing"

	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// AC-AKX-006 (c) — the new diagnostic is one more `code` value on the existing
// errors[] field, so the wire string is the whole contract and the protocol
// version does not move.
func TestExtraBlockUnbalancedCodeMatchesWireString(t *testing.T) {
	if protocol.CodeExtraBlockUnbalanced != "extra_block_unbalanced" {
		t.Errorf("CodeExtraBlockUnbalanced = %q, want %q", protocol.CodeExtraBlockUnbalanced, "extra_block_unbalanced")
	}
	if protocol.Version != 2 {
		t.Errorf("Version = %d, want 2 — a new code value is not a wire-shape change", protocol.Version)
	}
}
