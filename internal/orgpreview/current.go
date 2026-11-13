package orgpreview

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
)

// CurrentEvent tells every browser tab of an Emacs session which buffer the
// selected Emacs window shows. Tabs opened with follow=1 switch to its URL.
type CurrentEvent struct {
	Envelope
	Type  string `json:"type"`
	Title string `json:"title"`
	Path  string `json:"path"`
	URL   string `json:"url"`
}

// handleEmacsCurrent records the session's current buffer. The announcement
// waits for the buffer's first committed revision when its preview URL does
// not exist yet (Emacs debounces that revision).
func (s *Server) handleEmacsCurrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var env Envelope
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&env); err != nil {
		writeError(w, http.StatusBadRequest, "malformed json")
		return
	}
	if err := env.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if env.Origin != OriginEmacs {
		writeError(w, http.StatusBadRequest, "current origin must be emacs")
		return
	}
	if ev := s.setCurrent(env.SessionID, env.BufferID); ev != nil {
		s.broadcastSession(env.SessionID, *ev)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) setCurrent(sessionID, bufferID string) *CurrentEvent {
	state := s.state(sessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	state.current = bufferID
	ev := currentEventLocked(sessionID, state)
	state.currentSent = ev != nil
	return ev
}

// pendingCurrent returns the announcement held back for bufferID until its
// first revision committed, at most once per setCurrent.
func (s *Server) pendingCurrent(sessionID, bufferID string) *CurrentEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.sessions[sessionID]
	if state == nil || state.current != bufferID || state.currentSent {
		return nil
	}
	ev := currentEventLocked(sessionID, state)
	state.currentSent = ev != nil
	return ev
}

// currentFor returns the current buffer for a tab that opened on bufferID,
// or nil when the tab already shows it or nothing is current.
func (s *Server) currentFor(sessionID, bufferID string) *CurrentEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.sessions[sessionID]
	if state == nil || state.current == "" || state.current == bufferID {
		return nil
	}
	return currentEventLocked(sessionID, state)
}

func currentEventLocked(sessionID string, state *sessionState) *CurrentEvent {
	buffer := state.buffers[state.current]
	if buffer == nil || buffer.browserView == "" {
		return nil
	}
	title := ""
	if buffer.path != "" {
		title = filepath.Base(buffer.path)
	}
	return &CurrentEvent{
		Envelope: NewEnvelope(sessionID, state.current, buffer.revision, OriginServer, "current"),
		Type:     "current",
		Title:    title,
		Path:     buffer.path,
		URL:      previewURL(sessionID, state.current, buffer),
	}
}

func (s *Server) broadcastSession(sessionID string, payload any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.sessions[sessionID]
	if state == nil {
		return
	}
	for client := range state.browsers {
		select {
		case client.send <- payload:
		default:
		}
	}
}
