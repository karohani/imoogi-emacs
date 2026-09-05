package ankiconnect

import (
	"errors"
	"strings"
	"testing"
)

func TestTransportError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("connection refused")
	e := &TransportError{Op: "requestPermission", URL: "http://127.0.0.1:8765", Err: inner}

	if !strings.Contains(e.Error(), "requestPermission") || !strings.Contains(e.Error(), "connection refused") {
		t.Fatalf("Error() = %q, want it to mention op and inner error", e.Error())
	}
	if !errors.Is(e, inner) {
		t.Fatalf("errors.Is(e, inner) = false, want true")
	}
}

func TestProtocolError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("invalid character")
	e := &ProtocolError{Op: "notesInfo", Body: "<html>", Err: inner}

	if !strings.Contains(e.Error(), "notesInfo") || !strings.Contains(e.Error(), "invalid character") {
		t.Fatalf("Error() = %q, want it to mention op and inner error", e.Error())
	}
	if !errors.Is(e, inner) {
		t.Fatalf("errors.Is(e, inner) = false, want true")
	}
}

func TestAPIError_Error(t *testing.T) {
	e := &APIError{Action: "addNote", Message: "cannot create note because it is a duplicate"}
	if !strings.Contains(e.Error(), "addNote") || !strings.Contains(e.Error(), "duplicate") {
		t.Fatalf("Error() = %q, want it to mention action and message", e.Error())
	}
}

func TestPermissionDeniedError_Error(t *testing.T) {
	e := &PermissionDeniedError{}
	if !strings.Contains(e.Error(), "permission") {
		t.Fatalf("Error() = %q, want it to mention permission", e.Error())
	}
}
