package orgpreview

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed web/mermaid.min.js
var mermaidJS []byte

type ServerConfig struct {
	Token        string
	AllowedRoots []string
	Parser       Parser
}

type Server struct {
	token    string
	parser   Parser
	mux      *http.ServeMux
	mu       sync.Mutex
	sessions map[string]*sessionState
	roots    []string
}

type sessionState struct {
	buffers    map[string]*bufferState
	seenEvents map[string]struct{}
	emacs      map[chan NavigationEvent]struct{}
	browsers   map[*browserClient]struct{}
}

type bufferState struct {
	revision int64
	document Document
	html     string
	path     string
	roots    []string
}

type browserClient struct {
	sessionID string
	bufferID  string
	conn      net.Conn
	send      chan any
}

func NewServer(cfg ServerConfig) (*Server, error) {
	token := cfg.Token
	if token == "" {
		generated, err := GenerateToken()
		if err != nil {
			return nil, err
		}
		token = generated
	}
	parser := cfg.Parser
	if parser == nil {
		parser = FallbackParser{}
	}
	s := &Server{
		token:    token,
		parser:   parser,
		mux:      http.NewServeMux(),
		sessions: map[string]*sessionState{},
		roots:    cfg.AllowedRoots,
	}
	s.routes()
	return s, nil
}

func GenerateToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf[:]), nil
}

func (s *Server) Token() string { return s.token }

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) Listen(ctx context.Context, addr string) (*http.Server, net.Listener, error) {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	if host, _, err := net.SplitHostPort(ln.Addr().String()); err == nil && host != "127.0.0.1" && host != "localhost" {
		_ = ln.Close()
		return nil, nil, fmt.Errorf("refusing non-loopback listener %s", ln.Addr())
	}
	httpServer := &http.Server{Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		_ = httpServer.Shutdown(context.Background())
	}()
	go func() {
		if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("org preview server stopped: %v", err)
		}
	}()
	return httpServer, ln, nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/api/emacs/revisions", s.handleRevision)
	s.mux.HandleFunc("/api/emacs/events", s.handleEvents)
	s.mux.HandleFunc("/api/emacs/navigation", s.handleEmacsNavigation)
	s.mux.HandleFunc("/api/sessions/", s.handleSessionRender)
	s.mux.HandleFunc("/ws/browser", s.handleBrowserWebSocket)
	s.mux.HandleFunc("/asset", s.handleAsset)
	s.mux.HandleFunc("/static/mermaid.min.js", s.handleMermaidJS)
}

func (s *Server) handleMermaidJS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(mermaidJS)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"version": ProtocolVersion})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/preview" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, browserShell)
}

func (s *Server) handleRevision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req RevisionRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 16<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed json")
		return
	}
	if req.Origin == "" {
		req.Origin = OriginEmacs
	}
	if req.EventID == "" {
		req.EventID = fmt.Sprintf("revision-%d", req.Revision)
	}
	if err := req.Envelope.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Origin != OriginEmacs {
		writeError(w, http.StatusBadRequest, "revision origin must be emacs")
		return
	}
	state := s.state(req.SessionID)
	stale := false
	s.mu.Lock()
	buf := state.buffers[req.BufferID]
	if buf != nil && req.Revision < buf.revision {
		stale = true
	}
	s.mu.Unlock()
	if stale {
		resp := RevisionResponse{Envelope: req.Envelope, Committed: false, Stale: true}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	parser := s.parser
	if req.Syntax == "markdown" {
		parser = MarkdownParser{}
	}
	doc, err := parser.Parse(req.Text)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	roots := append([]string{}, s.roots...)
	roots = append(roots, req.AllowedRoots...)
	if req.Path != "" {
		roots = append(roots, filepath.Dir(req.Path))
	}
	resolver, err := NewAssetResolver(roots)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	html := Renderer{
		Assets:         resolver,
		BaseFile:       req.Path,
		AssetToken:     s.token,
		AssetSessionID: req.SessionID,
		AssetBufferID:  req.BufferID,
	}.Render(doc)
	renderEvent := RenderEvent{Envelope: req.Envelope, Document: doc, HTML: html}
	committed := s.commitRevision(req, doc, html, roots)
	resp := RevisionResponse{Envelope: req.Envelope, Committed: committed, Stale: !committed, Document: doc, HTML: html}
	writeJSON(w, http.StatusAccepted, resp)
	if committed {
		s.broadcastBrowser(req.SessionID, req.BufferID, renderEvent)
		if node := FindNodeAt(doc.Nodes, req.CursorByte); node != nil {
			nav := NavigationEvent{
				Envelope:  NewEnvelope(req.SessionID, req.BufferID, req.Revision, OriginEmacs, req.EventID+"-cursor"),
				ElementID: node.ID,
				Range:     node.Range,
			}
			s.broadcastBrowser(req.SessionID, req.BufferID, nav)
		}
	}
}

func (s *Server) handleSessionRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "render" {
		http.NotFound(w, r)
		return
	}
	sessionID := parts[0]
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.sessions[sessionID]
	if state == nil {
		http.NotFound(w, r)
		return
	}
	var latest *bufferState
	for _, buffer := range state.buffers {
		if latest == nil || buffer.revision > latest.revision {
			latest = buffer
		}
	}
	if latest == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, latest.html)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sessionID := r.URL.Query().Get("session")
	bufferID := r.URL.Query().Get("buffer")
	env := NewEnvelope(sessionID, bufferID, 0, OriginServer, "sse-open")
	if err := env.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	ch := make(chan NavigationEvent, 16)
	s.addEmacsSubscriber(sessionID, ch)
	defer s.removeEmacsSubscriber(sessionID, ch)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			if ev.BufferID != bufferID {
				continue
			}
			data, _ := json.Marshal(ev)
			_, _ = fmt.Fprintf(w, "event: navigation\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Server) handleEmacsNavigation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var ev NavigationEvent
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&ev); err != nil {
		writeError(w, http.StatusBadRequest, "malformed json")
		return
	}
	if err := ev.Envelope.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if ev.Origin != OriginEmacs {
		writeError(w, http.StatusBadRequest, "navigation origin must be emacs")
		return
	}
	mapped, ok := s.mapEmacsNavigation(ev)
	if !ok {
		writeError(w, http.StatusBadRequest, "navigation target is required")
		return
	}
	ev = mapped
	if ev.ElementID == "" {
		writeError(w, http.StatusBadRequest, "element_id is required")
		return
	}
	if s.markSeen(ev.SessionID, ev.EventID) {
		s.broadcastBrowser(ev.SessionID, ev.BufferID, ev)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) mapEmacsNavigation(ev NavigationEvent) (NavigationEvent, bool) {
	s.mu.Lock()
	state := s.sessions[ev.SessionID]
	var buf *bufferState
	if state != nil {
		buf = state.buffers[ev.BufferID]
	}
	s.mu.Unlock()
	if buf == nil {
		return ev, ev.ElementID != "" || ev.Range.End > ev.Range.Start
	}
	if node := FindNodeAt(buf.document.Nodes, ev.CursorByte); node != nil {
		ev.ElementID = node.ID
		ev.Range = node.Range
		return ev, true
	}
	if ev.Range.End > ev.Range.Start {
		if node := FindNodeAt(buf.document.Nodes, ev.Range.Start); node != nil {
			ev.ElementID = node.ID
			ev.Range = node.Range
			return ev, true
		}
	}
	return ev, ev.ElementID != ""
}

func (s *Server) handleBrowserWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sessionID := r.URL.Query().Get("session")
	bufferID := r.URL.Query().Get("buffer")
	env := NewEnvelope(sessionID, bufferID, 0, OriginBrowser, "ws-open")
	if err := env.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	conn, brw, err := hijackWebSocket(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	client := &browserClient{sessionID: sessionID, bufferID: bufferID, conn: conn, send: make(chan any, 16)}
	s.addBrowser(client)
	defer s.removeBrowser(client)
	if latest := s.latestRender(sessionID, bufferID); latest != nil {
		client.send <- *latest
	}
	errCh := make(chan error, 2)
	go func() { errCh <- client.writeLoop() }()
	go func() { errCh <- s.browserReadLoop(client, brw) }()
	<-errCh
}

func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	path := r.URL.Query().Get("path")
	clean, err := canonical(path)
	if err != nil {
		http.Error(w, "bad asset", http.StatusBadRequest)
		return
	}
	sessionID := r.URL.Query().Get("session")
	bufferID := r.URL.Query().Get("buffer")
	allowed := s.assetAllowed(sessionID, bufferID, clean)
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	f, err := os.Open(clean)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	if contentType := mime.TypeByExtension(filepath.Ext(clean)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = io.Copy(w, f)
}

func (s *Server) assetAllowed(sessionID, bufferID, clean string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sessionID != "" || bufferID != "" {
		state := s.sessions[sessionID]
		if state == nil {
			return false
		}
		return bufferAllowsAsset(state.buffers[bufferID], clean)
	}
	for _, state := range s.sessions {
		for _, buffer := range state.buffers {
			if bufferAllowsAsset(buffer, clean) {
				return true
			}
		}
	}
	return false
}

func bufferAllowsAsset(buffer *bufferState, clean string) bool {
	if buffer == nil {
		return false
	}
	for _, root := range buffer.roots {
		rootClean, err := canonical(root)
		if err == nil && within(rootClean, clean) {
			return true
		}
	}
	return false
}

func (s *Server) authorized(r *http.Request) bool {
	token := r.Header.Get("X-Org-Preview-Token")
	if token == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	return token != "" && token == s.token
}

func (s *Server) state(sessionID string) *sessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.sessions[sessionID]
	if state == nil {
		state = &sessionState{
			buffers:    map[string]*bufferState{},
			seenEvents: map[string]struct{}{},
			emacs:      map[chan NavigationEvent]struct{}{},
			browsers:   map[*browserClient]struct{}{},
		}
		s.sessions[sessionID] = state
	}
	return state
}

func (s *Server) commitRevision(req RevisionRequest, doc Document, html string, roots []string) bool {
	state := s.state(req.SessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	buf := state.buffers[req.BufferID]
	if buf != nil && req.Revision < buf.revision {
		return false
	}
	state.buffers[req.BufferID] = &bufferState{revision: req.Revision, document: doc, html: html, path: req.Path, roots: roots}
	return true
}

func (s *Server) addEmacsSubscriber(sessionID string, ch chan NavigationEvent) {
	state := s.state(sessionID)
	s.mu.Lock()
	state.emacs[ch] = struct{}{}
	s.mu.Unlock()
}

func (s *Server) removeEmacsSubscriber(sessionID string, ch chan NavigationEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state := s.sessions[sessionID]; state != nil {
		delete(state.emacs, ch)
	}
	close(ch)
}

func (s *Server) addBrowser(client *browserClient) {
	state := s.state(client.sessionID)
	s.mu.Lock()
	state.browsers[client] = struct{}{}
	s.mu.Unlock()
}

func (s *Server) removeBrowser(client *browserClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state := s.sessions[client.sessionID]; state != nil {
		delete(state.browsers, client)
	}
	_ = client.conn.Close()
	close(client.send)
}

func (s *Server) markSeen(sessionID, eventID string) bool {
	state := s.state(sessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := state.seenEvents[eventID]; ok {
		return false
	}
	state.seenEvents[eventID] = struct{}{}
	return true
}

func (s *Server) broadcastBrowser(sessionID, bufferID string, payload any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.sessions[sessionID]
	if state == nil {
		return
	}
	for client := range state.browsers {
		if client.bufferID != bufferID {
			continue
		}
		select {
		case client.send <- payload:
		default:
		}
	}
}

func (s *Server) broadcastEmacs(ev NavigationEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.sessions[ev.SessionID]
	if state == nil {
		return
	}
	for ch := range state.emacs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (s *Server) latestRender(sessionID, bufferID string) *RenderEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.sessions[sessionID]
	if state == nil {
		return nil
	}
	buf := state.buffers[bufferID]
	if buf == nil {
		return nil
	}
	env := NewEnvelope(sessionID, bufferID, buf.revision, OriginServer, "latest")
	return &RenderEvent{Envelope: env, Document: buf.document, HTML: buf.html}
}

func (s *Server) browserReadLoop(client *browserClient, rw *bufio.ReadWriter) error {
	for {
		payload, err := readWebSocketText(rw.Reader)
		if err != nil {
			return err
		}
		var ev NavigationEvent
		if err := json.Unmarshal(payload, &ev); err != nil {
			continue
		}
		if err := ev.Envelope.Validate(); err != nil {
			continue
		}
		if ev.Origin != OriginBrowser || ev.SessionID != client.sessionID || ev.BufferID != client.bufferID {
			continue
		}
		if ev.ElementID == "" {
			continue
		}
		if s.markSeen(ev.SessionID, ev.EventID) {
			s.broadcastEmacs(ev)
		}
	}
}

func (c *browserClient) writeLoop() error {
	for payload := range c.send {
		data, err := json.Marshal(payload)
		if err != nil {
			continue
		}
		if err := writeWebSocketText(c.conn, data); err != nil {
			return err
		}
	}
	return nil
}

func hijackWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, *bufio.ReadWriter, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, nil, errors.New("missing websocket upgrade")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, nil, errors.New("missing websocket key")
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijacking unsupported")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, nil, err
	}
	accept := websocketAccept(key)
	_, err = fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", accept)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	return conn, rw, nil
}

func websocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func readWebSocketText(r *bufio.Reader) ([]byte, error) {
	first, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	opcode := first & 0x0f
	if opcode == 0x8 {
		return nil, io.EOF
	}
	if opcode != 0x1 {
		return nil, errors.New("unsupported websocket opcode")
	}
	second, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	masked := second&0x80 != 0
	length := uint64(second & 0x7f)
	switch length {
	case 126:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		length = uint64(binary.BigEndian.Uint16(buf[:]))
	case 127:
		var buf [8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		length = binary.BigEndian.Uint64(buf[:])
	}
	if length > 1<<20 {
		return nil, errors.New("websocket payload too large")
	}
	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(r, mask[:]); err != nil {
			return nil, err
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return payload, nil
}

func writeWebSocketText(w io.Writer, payload []byte) error {
	header := []byte{0x81}
	switch {
	case len(payload) < 126:
		header = append(header, byte(len(payload)))
	case len(payload) <= 0xffff:
		header = append(header, 126, byte(len(payload)>>8), byte(len(payload)))
	default:
		header = append(header, 127)
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(len(payload)))
		header = append(header, buf[:]...)
	}
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Version: ProtocolVersion, Error: message})
}

const browserShell = `<!doctype html>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Org Preview</title>
<style>
:root{color-scheme:dark;--bg:#171b22;--panel:#20252d;--panel2:#252b35;--text:#d8dee9;--muted:#8792a2;--line:#333b48;--red:#ff8c92;--blue:#82b7ff;--green:#a5d67d;--yellow:#f2d479;--red-bg:#3b2930;--blue-bg:#253449;--green-bg:#2c392b;--yellow-bg:#3d3726;--active:#89ddff;--shadow:0 20px 60px rgba(0,0,0,.25)}
*{box-sizing:border-box}
body{margin:0;font:16px/1.65 ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:var(--text);background:radial-gradient(circle at top left,#273140 0,#171b22 34rem);letter-spacing:0}
#app{display:grid;grid-template-columns:280px minmax(0,1fr);gap:28px;max-width:1360px;margin:0 auto;padding:36px 28px 56px}
#root{min-width:0;max-width:900px;width:100%;margin:0 auto}
.preview-sidebar{position:sticky;top:20px;align-self:start;max-height:calc(100vh - 40px);overflow:hidden;border:1px solid rgba(255,255,255,.08);border-radius:8px;background:rgba(32,37,45,.92);box-shadow:var(--shadow);backdrop-filter:blur(14px)}
.preview-sidebar header{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px 14px;border-bottom:1px solid var(--line)}
#status{font-size:12px;color:var(--muted);white-space:nowrap}
.tabs{display:flex;gap:4px;padding:6px;border-bottom:1px solid var(--line);background:rgba(0,0,0,.12)}
.tab{appearance:none;border:0;border-radius:6px;padding:6px 10px;background:transparent;color:var(--muted);font:600 12px/1 system-ui,sans-serif;cursor:pointer}
.tab.active{background:var(--panel2);color:var(--text)}
.panel{display:none;max-height:calc(100vh - 124px);overflow:auto;padding:10px}
.panel.active{display:block}
.empty{color:var(--muted);font-size:13px;padding:10px}
.toc-item,.overview-item{width:100%;border:0;text-align:left;color:var(--text);background:transparent;cursor:pointer}
.toc-item{display:grid;grid-template-columns:8px minmax(0,1fr);gap:9px;align-items:start;border-radius:6px;padding:7px 8px;font-size:13px;line-height:1.35}
.toc-item:hover,.overview-item:hover{background:rgba(255,255,255,.055)}
.toc-swatch{height:22px;border-radius:3px;background:currentColor}
.toc-title{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.toc-l2{padding-left:18px}.toc-l3{padding-left:34px}.toc-l4,.toc-l5,.toc-l6{padding-left:50px}
.overview-item{display:block;border:1px solid rgba(255,255,255,.06);border-left:3px solid var(--section-color,var(--blue));border-radius:8px;margin:0 0 8px;padding:10px;background:rgba(255,255,255,.025)}
.overview-kind{display:block;margin-bottom:4px;color:var(--muted);font-size:11px;text-transform:uppercase;letter-spacing:.08em}
.overview-text{display:-webkit-box;-webkit-line-clamp:3;-webkit-box-orient:vertical;overflow:hidden;font-size:13px;line-height:1.42}
.palette-red{color:var(--red)}.palette-blue{color:var(--blue)}.palette-green{color:var(--green)}.palette-yellow{color:var(--yellow)}
.org-preview{background:rgba(32,37,45,.68);border:1px solid rgba(255,255,255,.08);border-radius:8px;padding:34px 42px;box-shadow:var(--shadow)}
.org-preview h1,.org-preview h2,.org-preview h3,.org-preview h4,.org-preview h5,.org-preview h6{position:relative;margin:1.7em -18px .7em;padding:8px 18px;border-radius:7px;line-height:1.25;overflow-wrap:anywhere;background:rgba(255,255,255,.045)}
.org-preview h1{margin-top:.1em}.org-preview .palette-red{color:var(--red);background:var(--red-bg)}.org-preview .palette-blue{color:var(--blue);background:var(--blue-bg)}.org-preview .palette-green{color:var(--green);background:var(--green-bg)}.org-preview .palette-yellow{color:var(--yellow);background:var(--yellow-bg)}
.org-preview .document-title{font-size:1.4rem;font-weight:650;letter-spacing:-.025em;color:var(--text);margin:0 0 1.5rem}
.org-preview .org-heading a,.org-preview .org-heading code{color:inherit;background:transparent}
.org-preview p{margin:1em 0;color:#d5dbe6}.org-preview a{color:var(--active)}.org-preview img{max-width:100%;border-radius:8px;border:1px solid var(--line)}
.org-preview ul{padding-left:1.35rem}.org-preview li{margin:.35em 0}
pre,code{border-radius:6px;background:#11151b}code{padding:2px 5px;color:#cdd6e3}pre{padding:14px 16px;overflow:auto;border:1px solid var(--line)}pre code{padding:0;background:transparent}
.mermaid{margin:1.25em 0;padding:18px;overflow:auto;border:1px solid var(--line);border-radius:8px;background:#11151b;text-align:center}.mermaid svg{max-width:100%;height:auto}
table{width:100%;border-collapse:collapse;margin:1.1em 0;overflow:hidden;border-radius:7px}td{border:1px solid var(--line);padding:7px 10px}
[data-org-id].active-block{outline:2px solid var(--active);outline-offset:5px;background-color:rgba(137,221,255,.07)}
.toc-item.active,.overview-item.active{background:rgba(137,221,255,.10);box-shadow:inset 2px 0 0 var(--active)}
@media (max-width:980px){#app{display:block;padding:18px 14px 42px}.preview-sidebar{position:relative;top:auto;margin:0 0 18px;max-height:none}.panel{max-height:240px}.org-preview{padding:24px 22px}.org-preview h1,.org-preview h2,.org-preview h3,.org-preview h4,.org-preview h5,.org-preview h6{margin-left:-10px;margin-right:-10px;padding-left:10px;padding-right:10px}}
</style>
<div id="app">
  <aside class="preview-sidebar" aria-label="Document overview">
    <header><strong>Org Preview</strong><span id="status">disconnected</span></header>
    <nav class="tabs" aria-label="Preview navigation">
      <button class="tab active" data-panel="toc" type="button">목차</button>
      <button class="tab" data-panel="overview" type="button">개요</button>
    </nav>
    <section id="toc" class="panel active" aria-label="Table of contents"></section>
    <section id="overview" class="panel" aria-label="Document overview"></section>
  </aside>
  <div id="root"></div>
</div>
<script src="/static/mermaid.min.js"></script>
<script>
mermaid.initialize({startOnLoad:false,securityLevel:'strict',theme:'dark'});
const params = new URLSearchParams(location.search);
const statusEl = document.getElementById('status');
const root = document.getElementById('root');
const toc = document.getElementById('toc');
const overview = document.getElementById('overview');
const sessionID = params.get('session_id') || params.get('session');
const bufferID = params.get('buffer_id') || params.get('buffer');
let revision = -1;
let activeTarget = null;
const blockKinds = new Set(['heading','paragraph','list','list_item','code_block','table','image']);
const palette = ['red','blue','green','yellow'];
function wsURL(){const u=new URL('/ws/browser', location.href);u.protocol=location.protocol==='https:'?'wss:':'ws:';u.searchParams.set('session', sessionID || '');u.searchParams.set('buffer', bufferID || '');u.searchParams.set('token', params.get('token') || '');return u;}
function connect(){
  const ws = new WebSocket(wsURL());
  ws.onopen = () => statusEl.textContent = 'connected';
  ws.onclose = () => { statusEl.textContent = 'reconnecting'; setTimeout(connect, 500); };
  ws.onmessage = event => {
    const msg = JSON.parse(event.data);
    if (msg.html) renderRevision(msg);
    markFromMessage(msg);
  };
  root.onclick = event => {
    const el = blockFor(event.target.closest('[data-org-id]'));
    if (!el || !sessionID || !bufferID || ws.readyState !== WebSocket.OPEN) return;
    sendNavigation(ws, el);
  };
}
function sendNavigation(ws, el){
  ws.send(JSON.stringify({version:'org-preview/v1',session_id:sessionID,buffer_id:bufferID,revision:revision,event_id:String(Date.now()),origin:'browser',element_id:el.dataset.orgId,range:{start:Number(el.dataset.orgRangeStart||0),end:Number(el.dataset.orgRangeEnd||0)}}));
}
function renderRevision(msg){
  if (typeof msg.revision === 'number' && msg.revision < revision) return;
  const x = window.scrollX;
  const y = window.scrollY;
  revision = msg.revision;
  root.innerHTML = msg.html;
  decorateDocument();
  rebuildSidebars();
  restoreActive();
  renderMermaid(revision);
  requestAnimationFrame(() => window.scrollTo(x, y));
}
async function renderMermaid(targetRevision){
  const nodes = root.querySelectorAll('.mermaid:not([data-processed])');
  if (!nodes.length) return;
  try {
    await mermaid.run({nodes:nodes,suppressErrors:true});
  } catch (_) {
    nodes.forEach(node => node.classList.add('mermaid-error'));
  }
  if (targetRevision === revision) restoreActive();
}
function markFromMessage(msg){
  if (!msg.element_id && !msg.range) return;
  if (typeof msg.revision === 'number' && msg.revision < revision) return;
  const byID = msg.element_id ? root.querySelector('[data-org-id="'+CSS.escape(msg.element_id)+'"]') : null;
  const byRange = msg.range ? root.querySelector('[data-org-range-start="'+Number(msg.range.start||0)+'"][data-org-range-end="'+Number(msg.range.end||0)+'"]') : null;
  markElement(byID || byRange);
}
function markElement(el){
  el = blockFor(el);
  if (!el) return;
  activeTarget = {id:el.dataset.orgId,start:el.dataset.orgRangeStart,end:el.dataset.orgRangeEnd};
  root.querySelectorAll('.active-block').forEach(n => n.classList.remove('active-block'));
  document.querySelectorAll('.toc-item.active,.overview-item.active').forEach(n => n.classList.remove('active'));
  el.classList.add('active-block');
  const selector = sidebarSelector(el);
  document.querySelectorAll(selector).forEach(n => n.classList.add('active'));
}
function restoreActive(){
  if (!activeTarget) return;
  const el = root.querySelector('[data-org-id="'+CSS.escape(activeTarget.id)+'"]') || root.querySelector('[data-org-range-start="'+CSS.escape(activeTarget.start)+'"][data-org-range-end="'+CSS.escape(activeTarget.end)+'"]');
  markElement(el);
}
function blockFor(el){
  while (el && el !== root) {
    if (blockKinds.has(el.dataset.orgKind)) return el;
    el = el.parentElement && el.parentElement.closest('[data-org-id]');
  }
  return null;
}
function decorateDocument(){
  root.querySelectorAll('p').forEach(el => {
    const title = el.textContent.match(/^#\+TITLE:\s*(.+)$/i);
    if (title) { el.textContent = title[1]; el.classList.add('document-title'); document.title = title[1] + ' · Org Preview'; }
  });
  let color = 'red';
  root.querySelectorAll('[data-org-id]').forEach(el => {
    if (el.matches('h1,h2,h3,h4,h5,h6')) color = colorForLevel(levelOf(el));
    el.dataset.sectionColor = color;
  });
  root.querySelectorAll('h1,h2,h3,h4,h5,h6').forEach(h => {
    h.classList.add('palette-'+colorForLevel(levelOf(h)));
  });
}
function rebuildSidebars(){
  const headings = Array.from(root.querySelectorAll('h1[data-org-id],h2[data-org-id],h3[data-org-id],h4[data-org-id],h5[data-org-id],h6[data-org-id]'));
  fillPanel(toc, headings, 'toc');
  const blocks = Array.from(root.querySelectorAll('[data-org-id]')).map(blockFor).filter(uniqueBlock).filter(el => blockKinds.has(el.dataset.orgKind));
  fillPanel(overview, blocks, 'overview');
}
function fillPanel(panel, nodes, mode){
  panel.replaceChildren();
  if (!nodes.length) {
    const empty = document.createElement('div');
    empty.className = 'empty';
    empty.textContent = mode === 'toc' ? 'No headings yet' : 'No blocks yet';
    panel.appendChild(empty);
    return;
  }
  nodes.forEach(el => panel.appendChild(mode === 'toc' ? tocButton(el) : overviewButton(el)));
}
function tocButton(el){
  const level = levelOf(el);
  const button = baseButton(el, 'toc-item toc-l'+Math.min(level, 6));
  const swatch = document.createElement('span');
  swatch.className = 'toc-swatch palette-'+colorForLevel(level);
  const title = document.createElement('span');
  title.className = 'toc-title';
  title.textContent = cleanText(el);
  button.append(swatch, title);
  return button;
}
function overviewButton(el){
  const button = baseButton(el, 'overview-item');
  button.style.setProperty('--section-color', 'var(--'+colorForElement(el)+')');
  const kind = document.createElement('span');
  kind.className = 'overview-kind palette-'+colorForElement(el);
  kind.textContent = labelForKind(el);
  const text = document.createElement('span');
  text.className = 'overview-text';
  text.textContent = cleanText(el);
  button.append(kind, text);
  return button;
}
function baseButton(el, className){
  const button = document.createElement('button');
  button.type = 'button';
  button.className = className;
  button.dataset.targetOrgId = el.dataset.orgId;
  button.dataset.targetOrgRangeStart = el.dataset.orgRangeStart || '0';
  button.dataset.targetOrgRangeEnd = el.dataset.orgRangeEnd || '0';
  button.addEventListener('click', () => jumpTo(el.dataset.orgId));
  return button;
}
function jumpTo(id){
  const el = root.querySelector('[data-org-id="'+CSS.escape(id)+'"]');
  if (!el) return;
  markElement(el);
  el.scrollIntoView({block:'start',behavior:'smooth'});
}
function sidebarSelector(el){
  const id = CSS.escape(el.dataset.orgId || '');
  const start = CSS.escape(el.dataset.orgRangeStart || '0');
  const end = CSS.escape(el.dataset.orgRangeEnd || '0');
  return '[data-target-org-id="'+id+'"],[data-target-org-range-start="'+start+'"][data-target-org-range-end="'+end+'"]';
}
function uniqueBlock(el, index, list){return el && list.indexOf(el) === index;}
function levelOf(el){return Number(el.dataset.orgLevel) || Number((el.tagName || 'H1').replace('H','')) || 1;}
function colorForLevel(level){return palette[(Math.max(1, level)-1)%palette.length];}
function colorForElement(el){return el.matches('h1,h2,h3,h4,h5,h6') ? colorForLevel(levelOf(el)) : (el.dataset.sectionColor || 'red');}
function cleanText(el){return (el.textContent || '').replace(/\s+/g,' ').trim() || labelForKind(el);}
function labelForKind(el){return ({heading:'제목',paragraph:'문단',list:'목록',list_item:'항목',code_block:'코드',table:'표',image:'이미지'})[el.dataset.orgKind] || '문단';}
document.querySelectorAll('.tab').forEach(tab => {
  tab.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach(n => n.classList.toggle('active', n === tab));
    document.querySelectorAll('.panel').forEach(n => n.classList.toggle('active', n.id === tab.dataset.panel));
  });
});
function bootEmpty(){
  root.innerHTML = '<main class="org-preview"><p>Waiting for Emacs buffer...</p></main>';
  rebuildSidebars();
}
bootEmpty();
connect();
</script>`
