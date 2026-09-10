package orgpreview

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixture = `* Title
Paragraph with *bold*, /italic/, ~code~, [[file:doc.txt][doc]], and [[file:image.png][image]].

- first
- second

#+begin_src go
fmt.Println("<escaped>")
#+end_src

| Name | Value |
|------+-------|
| A    | B     |
`

func TestFallbackParserV1Constructs(t *testing.T) {
	doc, err := FallbackParser{}.Parse(fixture)
	if err != nil {
		t.Fatal(err)
	}
	types := flattenTypes(doc.Nodes)
	for _, want := range []string{"heading", "paragraph", "strong", "emphasis", "code", "link", "image", "list", "list_item", "code_block", "table", "table_row", "table_cell"} {
		if !contains(types, want) {
			t.Fatalf("missing node type %q in %#v", want, types)
		}
	}
	seen := map[string]struct{}{}
	var checkIDs func([]Node)
	checkIDs = func(nodes []Node) {
		for _, node := range nodes {
			if node.ID == "" {
				t.Fatalf("node %s has empty id", node.Type)
			}
			if _, ok := seen[node.ID]; ok {
				t.Fatalf("duplicate id %q", node.ID)
			}
			seen[node.ID] = struct{}{}
			if !node.Range.ValidFor(len(fixture)) {
				t.Fatalf("bad range for %s: %#v", node.Type, node.Range)
			}
			checkIDs(node.Children)
		}
	}
	checkIDs(doc.Nodes)
}

func TestRendererEscapesHTMLAndMapsElements(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "doc.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "image.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := FallbackParser{}.Parse(fixture)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := NewAssetResolver([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	out := Renderer{
		Assets:         resolver,
		BaseFile:       filepath.Join(root, "note.org"),
		AssetToken:     "test-token",
		AssetSessionID: "s1",
		AssetBufferID:  "b1",
	}.Render(doc)
	for _, want := range []string{"data-org-id=", "<strong", "<em", ">code</code>", "&lt;escaped&gt;", "<table", "<img"} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered html missing %q:\n%s", want, out)
		}
	}
	for _, want := range []string{"token=test-token", "session=s1", "buffer=b1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("asset URL missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "<escaped>") {
		t.Fatalf("raw code HTML was not escaped:\n%s", out)
	}
}

func TestProtocolValidation(t *testing.T) {
	valid := NewEnvelope("session-1", "buffer-1", 1, OriginEmacs, "event-1")
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid envelope rejected: %v", err)
	}
	invalid := valid
	invalid.Version = "old"
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid version accepted")
	}
	invalid = valid
	invalid.Origin = "loop"
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid origin accepted")
	}
	invalid = valid
	invalid.SessionID = "../escape"
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid session accepted")
	}
}

func TestAssetResolverRejectsTraversalAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}
	resolver, err := NewAssetResolver([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, "note.org")
	if _, err := resolver.Resolve(base, "file://"+filepath.Join(root, "ok.txt")); err == nil {
		t.Fatal("file:// target accepted")
	}
	if _, err := resolver.Resolve(base, "../secret.txt"); err == nil {
		t.Fatal("path traversal accepted")
	}
	if _, err := resolver.Resolve(base, "link.txt"); err == nil {
		t.Fatal("symlink escape accepted")
	}
}

func TestServerRevisionAuthAndNewestWins(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token", Parser: FallbackParser{}})
	if err != nil {
		t.Fatal(err)
	}

	req := RevisionRequest{Envelope: NewEnvelope("s1", "b1", 2, OriginEmacs, "e2"), Text: "* New\n"}
	resp := postRevision(t, server.Handler(), "test-token", req)
	if !resp.Committed || resp.Revision != 2 || !strings.Contains(resp.HTML, "New") {
		t.Fatalf("unexpected revision response: %#v", resp)
	}
	staleReq := RevisionRequest{Envelope: NewEnvelope("s1", "b1", 1, OriginEmacs, "e1"), Text: "* Old\n"}
	stale := postRevision(t, server.Handler(), "test-token", staleReq)
	if !stale.Stale || stale.Committed {
		t.Fatalf("stale revision committed: %#v", stale)
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/emacs/revisions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httpReq)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated revision status = %d", rec.Code)
	}
}

func TestServerMapsEmacsNavigationCursorToRenderedSemanticID(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token", Parser: FallbackParser{}})
	if err != nil {
		t.Fatal(err)
	}
	source := "* Title\nBody text\n"
	cursor := strings.Index(source, "Body")
	req := RevisionRequest{
		Envelope:   NewEnvelope("s1", "b1", 1, OriginEmacs, "revision-1"),
		Text:       source,
		CursorByte: cursor,
	}
	resp := postRevision(t, server.Handler(), "test-token", req)
	want := FindNodeAt(resp.Document.Nodes, cursor)
	if want == nil {
		t.Fatal("fixture did not produce a node at cursor")
	}

	ev := NavigationEvent{
		Envelope:   NewEnvelope("s1", "b1", 1, OriginEmacs, "nav-1"),
		CursorByte: cursor,
		ElementID:  "emacs-org-element-id",
	}
	mapped, ok := server.mapEmacsNavigation(ev)
	if !ok {
		t.Fatal("cursor navigation was not mapped")
	}
	if mapped.ElementID != want.ID || mapped.Range != want.Range {
		t.Fatalf("mapped navigation = %#v, want id=%q range=%#v", mapped, want.ID, want.Range)
	}

	body, err := json.Marshal(NavigationEvent{
		Envelope:   NewEnvelope("s1", "b1", 1, OriginEmacs, "nav-2"),
		CursorByte: cursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/emacs/navigation", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Org-Preview-Token", "test-token")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httpReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("cursor-only navigation status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestServerAssetRequiresTokenAndMatchingSessionBufferRoot(t *testing.T) {
	root := t.TempDir()
	assetPath := filepath.Join(root, "image.png")
	if err := os.WriteFile(assetPath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(ServerConfig{Token: "test-token", Parser: FallbackParser{}})
	if err != nil {
		t.Fatal(err)
	}
	req := RevisionRequest{
		Envelope:     NewEnvelope("s1", "b1", 1, OriginEmacs, "e1"),
		Text:         "[[file:image.png][image]]",
		Path:         filepath.Join(root, "note.org"),
		AllowedRoots: []string{root},
	}
	resp := postRevision(t, server.Handler(), "test-token", req)
	if !strings.Contains(resp.HTML, "token=test-token") || !strings.Contains(resp.HTML, "session=s1") || !strings.Contains(resp.HTML, "buffer=b1") {
		t.Fatalf("rendered asset URL is not session authenticated:\n%s", resp.HTML)
	}

	request := httptest.NewRequest(http.MethodGet, "/asset?path="+assetPath+"&session=s1&buffer=b1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, request)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated asset status = %d", rec.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/asset?path="+assetPath+"&session=s1&buffer=other&token=test-token", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, request)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("wrong buffer asset status = %d", rec.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/asset?path="+assetPath+"&session=s1&buffer=b1&token=test-token", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, request)
	if rec.Code != http.StatusOK || rec.Body.String() != "png" {
		t.Fatalf("authorized asset status/body = %d/%q", rec.Code, rec.Body.String())
	}
}

func TestServerServesPreviewShellAndBrowserWebSocketQueryContract(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/preview?session_id=s1&buffer_id=b1&token=test-token", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, request)

	if rec.Code != http.StatusOK {
		t.Fatalf("preview shell status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"session_id",
		"buffer_id",
		"/ws/browser",
		`id="toc"`,
		`id="overview"`,
		"--red:#ff8c92",
		"--blue:#82b7ff",
		"--green:#a5d67d",
		"--yellow:#f2d479",
		"function markElement",
		"function rebuildSidebars",
		"window.scrollTo(x, y)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("preview shell missing %q:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{"window.onscroll", "block:'center'"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("preview shell should not auto-scroll on navigation; found %q:\n%s", unwanted, body)
		}
	}
}

func TestServerDeduplicatesOriginEvents(t *testing.T) {
	server, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	ev := NavigationEvent{Envelope: NewEnvelope("s1", "b1", 3, OriginBrowser, "same-event"), ElementID: "heading-1"}
	if !server.markSeen(ev.SessionID, ev.EventID) {
		t.Fatal("first event should be accepted")
	}
	if server.markSeen(ev.SessionID, ev.EventID) {
		t.Fatal("duplicate event should be suppressed")
	}
}

func postRevision(t *testing.T, handler http.Handler, token string, req RevisionRequest) RevisionResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/emacs/revisions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Org-Preview-Token", token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httpReq)
	data, _ := io.ReadAll(rec.Result().Body)
	if rec.Code != http.StatusAccepted && rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, data)
	}
	var out RevisionResponse
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func flattenTypes(nodes []Node) []string {
	var out []string
	var walk func([]Node)
	walk = func(list []Node) {
		for _, node := range list {
			out = append(out, node.Type)
			walk(node.Children)
		}
	}
	walk(nodes)
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
