package orgpreview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

type recentTestBuffer struct {
	session, buffer, path string
	generation            uint64
	view                  string
}

func postRecentBuffer(t *testing.T, server *Server, revision int64, b *recentTestBuffer) {
	t.Helper()
	resp := postRevision(t, server.Handler(), "test-token", RevisionRequest{
		Envelope:   NewEnvelope(b.session, b.buffer, revision, OriginEmacs, fmt.Sprintf("%s-%d", b.buffer, revision)),
		Generation: b.generation,
		Text:       "* " + b.buffer + "\n",
		Path:       b.path,
	})
	b.view = resp.BrowserView
}

func getRecent(t *testing.T, server *Server, b *recentTestBuffer, view string) (int, recentFilesResponse) {
	t.Helper()
	query := url.Values{
		"session": {b.session}, "buffer": {b.buffer},
		"generation": {fmt.Sprint(b.generation)}, "view": {view},
	}
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/recent?"+query.Encode(), nil))
	var out recentFilesResponse
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code, out
}

func newRecentTestServer(t *testing.T) *Server {
	t.Helper()
	server, err := NewServer(ServerConfig{Token: "test-token", Parser: FallbackParser{}})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func TestServerRecentFilesRequiresBrowserCredentials(t *testing.T) {
	server := newRecentTestServer(t)
	a := &recentTestBuffer{session: "s1", buffer: "a", path: filepath.Join(t.TempDir(), "a.org"), generation: 1}
	postRecentBuffer(t, server, 1, a)

	if code, _ := getRecent(t, server, a, ""); code != http.StatusUnauthorized {
		t.Fatalf("missing view status = %d", code)
	}
	if code, _ := getRecent(t, server, a, "wrong-view"); code != http.StatusUnauthorized {
		t.Fatalf("wrong view status = %d", code)
	}
	stale := *a
	stale.generation = 2
	if code, _ := getRecent(t, server, &stale, a.view); code != http.StatusUnauthorized {
		t.Fatalf("stale generation status = %d", code)
	}
}

func TestServerRecentFilesListsSessionFilesNewestFirst(t *testing.T) {
	server := newRecentTestServer(t)
	dir := t.TempDir()
	a := &recentTestBuffer{session: "s1", buffer: "a", path: filepath.Join(dir, "a.org"), generation: 1}
	b := &recentTestBuffer{session: "s1", buffer: "b", path: filepath.Join(dir, "notes", "b.md"), generation: 3}
	unsaved := &recentTestBuffer{session: "s1", buffer: "scratch", generation: 1}
	other := &recentTestBuffer{session: "s2", buffer: "d", path: filepath.Join(dir, "d.org"), generation: 1}
	postRecentBuffer(t, server, 1, a)
	postRecentBuffer(t, server, 1, b)
	postRecentBuffer(t, server, 1, unsaved)
	postRecentBuffer(t, server, 1, other)

	code, got := getRecent(t, server, a, a.view)
	if code != http.StatusOK {
		t.Fatalf("recent status = %d", code)
	}
	if len(got.Files) != 2 {
		t.Fatalf("want 2 session files (no pathless buffer, no other session), got %+v", got.Files)
	}
	if got.Files[0].Title != "b.md" || got.Files[1].Title != "a.org" {
		t.Fatalf("want newest first [b.md a.org], got %+v", got.Files)
	}
	if got.Files[0].Current || !got.Files[1].Current {
		t.Fatalf("current flag wrong: %+v", got.Files)
	}
	if got.Files[0].Path != b.path {
		t.Fatalf("path = %q, want %q", got.Files[0].Path, b.path)
	}

	// The entry URL is the other buffer's own live preview address.
	entry, err := url.Parse(got.Files[0].URL)
	if err != nil {
		t.Fatal(err)
	}
	q := entry.Query()
	if entry.Path != "/preview" || q.Get("session") != "s1" || q.Get("buffer") != "b" ||
		q.Get("view") != b.view || q.Get("generation") != "3" {
		t.Fatalf("entry url = %q", got.Files[0].URL)
	}
	if code, _ := getRecent(t, server, b, q.Get("view")); code != http.StatusOK {
		t.Fatalf("entry credentials do not authorize buffer b: %d", code)
	}

	// A new revision moves the file to the top.
	postRecentBuffer(t, server, 2, a)
	_, got = getRecent(t, server, a, a.view)
	if got.Files[0].Title != "a.org" {
		t.Fatalf("want a.org first after its update, got %+v", got.Files)
	}
}

func TestServerRecentFilesCapsTheList(t *testing.T) {
	server := newRecentTestServer(t)
	dir := t.TempDir()
	var last *recentTestBuffer
	for i := 0; i < recentFilesLimit+5; i++ {
		last = &recentTestBuffer{session: "s1", buffer: fmt.Sprintf("b%d", i), path: filepath.Join(dir, fmt.Sprintf("f%d.org", i)), generation: 1}
		postRecentBuffer(t, server, 1, last)
	}
	_, got := getRecent(t, server, last, last.view)
	if len(got.Files) != recentFilesLimit {
		t.Fatalf("want %d files, got %d", recentFilesLimit, len(got.Files))
	}
	if !got.Files[0].Current {
		t.Fatalf("newest (current) file should lead: %+v", got.Files[0])
	}
}

func TestBrowserShellHasRecentFilesTab(t *testing.T) {
	for _, want := range []string{
		`data-panel="recent"`,
		`id="recent"`,
		"/api/recent",
		"function refreshRecent(",
	} {
		if !strings.Contains(browserShell, want) {
			t.Fatalf("preview shell missing %q", want)
		}
	}
	if !strings.Contains(functionBody(t, browserShell, "function renderRevision("), "refreshRecent()") {
		t.Fatal("renderRevision must refresh the recent files list")
	}
}
