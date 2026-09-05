package ankiconnect

import "fmt"

// TransportError means the request never reached anything that speaks
// HTTP at the configured URL — dial failure, connection refused, or a
// timeout. Nothing is listening: a caller (internal/planner, a later
// milestone) maps this to plan.md D-5's anki_unreachable ("no listener at
// the AnkiConnect URL — Anki is very likely not running").
type TransportError struct {
	Op  string
	URL string
	Err error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("ankiconnect: %s %s: %v", e.Op, e.URL, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }

// ProtocolError means an HTTP response was received at the configured URL,
// but its body does not carry a valid AnkiConnect JSON-RPC envelope —
// unparseable JSON, or a shape that does not resemble one (including a
// notesInfo response array whose length does not match the request).
// Something is listening, but it does not appear to be AnkiConnect: a
// caller maps this to plan.md D-5's ankiconnect_missing when it surfaces
// from Handshake ("host responds but does not answer the AnkiConnect
// handshake — the add-on is absent or unresponsive").
type ProtocolError struct {
	Op   string
	Body string
	Err  error
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("ankiconnect: %s: unexpected response: %v (body: %.200q)", e.Op, e.Err, e.Body)
}

func (e *ProtocolError) Unwrap() error { return e.Err }

// APIError wraps a non-null "error" string AnkiConnect's own envelope
// carried for a request that otherwise round-tripped correctly — the
// request reached AnkiConnect and AnkiConnect itself rejected it. Maps to
// plan.md D-5's ankiconnect_error.
type APIError struct {
	Action  string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ankiconnect: %s: %s", e.Action, e.Message)
}

// PermissionDeniedError means AnkiConnect answered the handshake but
// declined permission for this origin (research.md §3.5 residual finding:
// "requestPermission can return permission: denied on an untrusted
// origin"). localhost is trusted by default, so this is not expected in
// normal operation; no plan.md D-5 code is reserved for it specifically —
// a caller that observes it is free to map it alongside
// ankiconnect_missing, since the add-on responded but is unusable either
// way.
type PermissionDeniedError struct{}

func (e *PermissionDeniedError) Error() string {
	return "ankiconnect: permission denied for this origin"
}
