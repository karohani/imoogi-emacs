package orgpreview

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func postCurrent(t *testing.T, server *Server, token, sessionID, bufferID string) int {
	t.Helper()
	body, err := json.Marshal(NewEnvelope(sessionID, bufferID, 0, OriginEmacs, "current-"+bufferID))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/emacs/current", bytes.NewReader(body))
	if token != "" {
		req.Header.Set("X-Org-Preview-Token", token)
	}
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	return rec.Code
}

func watchBrowser(server *Server, sessionID, bufferID string) *browserClient {
	client := &browserClient{sessionID: sessionID, bufferID: bufferID, send: make(chan any, 8)}
	server.addBrowser(client)
	return client
}

// watchFollower is a tab opened with follow=1, the only kind told about other buffers.
func watchFollower(server *Server, sessionID, bufferID string) *browserClient {
	client := &browserClient{sessionID: sessionID, bufferID: bufferID, follow: true, send: make(chan any, 8)}
	server.addBrowser(client)
	return client
}

func drainCurrent(t *testing.T, client *browserClient) []CurrentEvent {
	t.Helper()
	var out []CurrentEvent
	for {
		select {
		case msg := <-client.send:
			if ev, ok := msg.(CurrentEvent); ok {
				out = append(out, ev)
			}
		default:
			return out
		}
	}
}

func TestServerCurrentRequiresEmacsToken(t *testing.T) {
	server := newRecentTestServer(t)
	if code := postCurrent(t, server, "", "s1", "a"); code != http.StatusUnauthorized {
		t.Fatalf("missing token: %d", code)
	}
	if code := postCurrent(t, server, "wrong", "s1", "a"); code != http.StatusUnauthorized {
		t.Fatalf("wrong token: %d", code)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/emacs/current", nil)
	req.Header.Set("X-Org-Preview-Token", "test-token")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", rec.Code)
	}
}

func TestServerCurrentTellsOnlyFollowTabsTheTargetURL(t *testing.T) {
	server := newRecentTestServer(t)
	a := &recentTestBuffer{session: "s1", buffer: "a", path: "/notes/a.org", generation: 1}
	b := &recentTestBuffer{session: "s1", buffer: "b", path: "/notes/b.md", generation: 3}
	postRecentBuffer(t, server, 1, a)
	postRecentBuffer(t, server, 1, b)
	onA, onB, other := watchFollower(server, "s1", "a"), watchFollower(server, "s1", "b"), watchFollower(server, "s2", "a")
	plain := watchBrowser(server, "s1", "a")
	drainCurrent(t, onA)
	drainCurrent(t, onB)
	drainCurrent(t, plain)

	if code := postCurrent(t, server, "test-token", "s1", "b"); code != http.StatusOK {
		t.Fatalf("current: %d", code)
	}
	for name, client := range map[string]*browserClient{"tab on a": onA, "tab on b": onB} {
		got := drainCurrent(t, client)
		if len(got) != 1 {
			t.Fatalf("%s: %d current events", name, len(got))
		}
		ev := got[0]
		if ev.Type != "current" || ev.BufferID != "b" || ev.Title != "b.md" || ev.Path != "/notes/b.md" {
			t.Fatalf("%s: %#v", name, ev)
		}
		u, err := url.Parse(ev.URL)
		if err != nil || u.Path != "/preview" {
			t.Fatalf("%s: url %q", name, ev.URL)
		}
		q := u.Query()
		if q.Get("session") != "s1" || q.Get("buffer") != "b" || q.Get("generation") != "3" || q.Get("view") != b.view {
			t.Fatalf("%s: url query %v (view %s)", name, q, b.view)
		}
	}
	if got := drainCurrent(t, other); len(got) != 0 {
		t.Fatalf("another session must not hear about this Emacs: %#v", got)
	}
	// A plain tab holds one buffer's view token; it must not learn other buffers' tokens.
	if got := drainCurrent(t, plain); len(got) != 0 {
		t.Fatalf("a tab without follow=1 must not receive other buffers' URLs: %#v", got)
	}
}

func TestServerCurrentSkipsBuffersWithoutAFile(t *testing.T) {
	server := newRecentTestServer(t)
	a := &recentTestBuffer{session: "s1", buffer: "a", path: "/notes/a.org", generation: 1}
	scratch := &recentTestBuffer{session: "s1", buffer: "scratch", generation: 1}
	postRecentBuffer(t, server, 1, a)
	postRecentBuffer(t, server, 1, scratch)
	tab := watchFollower(server, "s1", "a")
	drainCurrent(t, tab)
	postCurrent(t, server, "test-token", "s1", "scratch")
	if got := drainCurrent(t, tab); len(got) != 0 {
		t.Fatalf("a buffer that /api/recent hides must not be announced either: %#v", got)
	}
	if ev := server.currentFor("s1", "a"); ev != nil {
		t.Fatalf("a connecting tab must not learn a pathless buffer: %#v", ev)
	}
}

func TestServerCurrentWaitsForTheFirstRevision(t *testing.T) {
	server := newRecentTestServer(t)
	a := &recentTestBuffer{session: "s1", buffer: "a", path: "/notes/a.org", generation: 1}
	postRecentBuffer(t, server, 1, a)
	tab := watchFollower(server, "s1", "a")
	drainCurrent(t, tab)

	// Emacs announces a buffer whose first revision is still debounced.
	if code := postCurrent(t, server, "test-token", "s1", "c"); code != http.StatusOK {
		t.Fatalf("current before revision: %d", code)
	}
	if got := drainCurrent(t, tab); len(got) != 0 {
		t.Fatalf("no URL exists yet, nothing to send: %#v", got)
	}
	c := &recentTestBuffer{session: "s1", buffer: "c", path: "/notes/c.org", generation: 1}
	postRecentBuffer(t, server, 1, c)
	got := drainCurrent(t, tab)
	if len(got) != 1 || got[0].BufferID != "c" || !strings.Contains(got[0].URL, "view="+c.view) {
		t.Fatalf("first revision must announce the waiting current buffer: %#v", got)
	}
	postRecentBuffer(t, server, 2, c)
	if got := drainCurrent(t, tab); len(got) != 0 {
		t.Fatalf("later edits must not re-announce: %#v", got)
	}
}

func TestServerCurrentGreetsANewTabOnAnotherBuffer(t *testing.T) {
	server := newRecentTestServer(t)
	a := &recentTestBuffer{session: "s1", buffer: "a", path: "/notes/a.org", generation: 1}
	b := &recentTestBuffer{session: "s1", buffer: "b", path: "/notes/b.org", generation: 1}
	postRecentBuffer(t, server, 1, a)
	postRecentBuffer(t, server, 1, b)
	if ev := server.currentFor("s1", "a"); ev != nil {
		t.Fatalf("no current buffer yet: %#v", ev)
	}
	postCurrent(t, server, "test-token", "s1", "b")
	if ev := server.currentFor("s1", "a"); ev == nil || ev.BufferID != "b" {
		t.Fatalf("tab opened on a must learn about b: %#v", ev)
	}
	if ev := server.currentFor("s1", "b"); ev != nil {
		t.Fatalf("tab already on the current buffer: %#v", ev)
	}
}

func TestBrowserShellFollowsCurrentBuffer(t *testing.T) {
	for _, want := range []string{
		"const follow = params.get('follow') === '1';",
		"if (msg.type === 'current') { reader.follow(msg); return; }",
		"function followURL(",
		"if (follow && !linkedFileID) u.searchParams.set('follow', '1');",
	} {
		if !strings.Contains(browserShell, want) {
			t.Fatalf("preview shell missing %q", want)
		}
	}
}
