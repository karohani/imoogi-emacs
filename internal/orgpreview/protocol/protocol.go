package protocol

import "github.com/karohani/imoogi-emacs/internal/orgpreview"

const Version = orgpreview.ProtocolVersion

type Origin = orgpreview.Origin

const (
	OriginEmacs   = orgpreview.OriginEmacs
	OriginBrowser = orgpreview.OriginBrowser
	OriginServer  = orgpreview.OriginServer
)

type NavigationEvent struct {
	Version    string                 `json:"version"`
	SessionID  string                 `json:"sessionId"`
	BufferID   string                 `json:"bufferId"`
	Revision   int64                  `json:"revision"`
	EventID    string                 `json:"eventId"`
	Origin     Origin                 `json:"origin"`
	ElementID  string                 `json:"elementId"`
	CursorByte int                    `json:"cursorByte,omitempty"`
	Range      orgpreview.SourceRange `json:"range,omitempty"`
}

type RevisionArbiter struct {
	latest map[string]int64
}

func NewRevisionArbiter() *RevisionArbiter {
	return &RevisionArbiter{latest: map[string]int64{}}
}

func (a *RevisionArbiter) Accepts(sessionID, bufferID string, revision int64) bool {
	key := sessionID + "\x00" + bufferID
	if current, ok := a.latest[key]; ok && revision < current {
		return false
	}
	a.latest[key] = revision
	return true
}

type LoopSuppressor struct {
	seen map[string]struct{}
}

func NewLoopSuppressor() *LoopSuppressor {
	return &LoopSuppressor{seen: map[string]struct{}{}}
}

func (s *LoopSuppressor) SeenOrMark(ev NavigationEvent) bool {
	key := ev.SessionID + "\x00" + ev.BufferID + "\x00" + ev.EventID
	if _, ok := s.seen[key]; ok {
		return true
	}
	s.seen[key] = struct{}{}
	return false
}
