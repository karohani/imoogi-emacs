package orgpreview

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
)

// recentFilesLimit bounds the sidebar list so a long Emacs session stays readable.
const recentFilesLimit = 20

type recentFile struct {
	Title   string `json:"title"`
	Path    string `json:"path"`
	URL     string `json:"url"`
	Current bool   `json:"current"`
}

type recentFilesResponse struct {
	Files []recentFile `json:"files"`
}

// handleRecentFiles lists the files previewed in the caller's Emacs session,
// most recently updated first, each with its own live preview URL.
//
// @MX:NOTE The caller authenticates as one buffer but receives the browser
// view of every file buffer in the same session (never another session):
// the sidebar must open their live previews. Same Emacs, same user.
func (s *Server) handleRecentFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.browserAuthorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sessionID := r.URL.Query().Get("session")
	currentID := r.URL.Query().Get("buffer")

	type candidate struct {
		file    recentFile
		touched uint64
	}
	var candidates []candidate
	s.mu.Lock()
	if state := s.sessions[sessionID]; state != nil {
		for bufferID, buffer := range state.buffers {
			if buffer.path == "" || buffer.browserView == "" {
				continue
			}
			values := url.Values{
				"session": {sessionID}, "buffer": {bufferID},
				"generation": {fmt.Sprint(buffer.generation)}, "view": {buffer.browserView},
			}
			candidates = append(candidates, candidate{
				file: recentFile{
					Title:   filepath.Base(buffer.path),
					Path:    buffer.path,
					URL:     "/preview?" + values.Encode(),
					Current: bufferID == currentID,
				},
				touched: buffer.touched,
			})
		}
	}
	s.mu.Unlock()

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].touched > candidates[j].touched })
	if len(candidates) > recentFilesLimit {
		candidates = candidates[:recentFilesLimit]
	}
	resp := recentFilesResponse{Files: make([]recentFile, 0, len(candidates))}
	for _, c := range candidates {
		resp.Files = append(resp.Files, c.file)
	}
	writeJSON(w, http.StatusOK, resp)
}
