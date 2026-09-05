package registry

import (
	"errors"
	"strings"
	"testing"
)

func TestCorruptError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("boom")
	e := &CorruptError{Path: "/tmp/registry.json", Err: inner}

	if !strings.Contains(e.Error(), "/tmp/registry.json") || !strings.Contains(e.Error(), "boom") {
		t.Fatalf("Error() = %q, want it to mention path and inner error", e.Error())
	}
	if !errors.Is(e, inner) {
		t.Fatalf("errors.Is(e, inner) = false, want true (Unwrap must expose inner error)")
	}
}
