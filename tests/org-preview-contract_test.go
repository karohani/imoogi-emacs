package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/karohani/imoogi-emacs/internal/orgpreview"
	"github.com/karohani/imoogi-emacs/internal/orgpreview/assets"
	"github.com/karohani/imoogi-emacs/internal/orgpreview/parser"
	"github.com/karohani/imoogi-emacs/internal/orgpreview/protocol"
	"github.com/karohani/imoogi-emacs/internal/orgpreview/render"
	"github.com/karohani/imoogi-emacs/internal/orgpreview/server"
	"github.com/karohani/imoogi-emacs/internal/orgpreview/session"
)

func TestOrgPreviewFallbackParserReturnsStableSemanticRangesForUnsavedBuffer(t *testing.T) {
	source := readOrgPreviewFixture(t, "v1-surface.org")

	doc, err := parser.Parse(source, parser.Options{
		Adapter:  parser.AdapterFallback,
		BufferID: "buffer-a",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assertNodeCoveringText(t, doc, orgpreview.KindHeading, "Heading One")
	assertNodeCoveringText(t, doc, orgpreview.KindParagraph, "Korean text")
	assertNodeCoveringText(t, doc, orgpreview.KindListItem, "First item")
	assertNodeCoveringText(t, doc, orgpreview.KindCodeBlock, "fmt.Println")
	assertNodeCoveringText(t, doc, orgpreview.KindTable, "Column A")
	assertNodeCoveringText(t, doc, orgpreview.KindImage, "diagram.png")
	assertNodeCoveringText(t, doc, orgpreview.KindFileLink, "notes.org")
}

func TestOrgPreviewTreeSitterAndFallbackAdaptersEmitEquivalentIRForV1Fixture(t *testing.T) {
	if !parser.TreeSitterAvailable() {
		t.Skip("tree-sitter Org adapter is optional until the vendored grammar feasibility gate passes")
	}
	source := readOrgPreviewFixture(t, "v1-surface.org")

	fallback, err := parser.Parse(source, parser.Options{Adapter: parser.AdapterFallback, BufferID: "buffer-a"})
	if err != nil {
		t.Fatalf("fallback Parse() error = %v", err)
	}
	treeSitter, err := parser.Parse(source, parser.Options{Adapter: parser.AdapterTreeSitter, BufferID: "buffer-a"})
	if err != nil {
		t.Fatalf("tree-sitter Parse() error = %v", err)
	}

	if diff := orgpreview.DiffNormalizedIR(fallback, treeSitter); diff != "" {
		t.Fatalf("normalized IR differs between fallback and tree-sitter adapters:\n%s", diff)
	}
}

func TestOrgPreviewRendererEscapesHTMLAndEmitsSemanticMappings(t *testing.T) {
	source := readOrgPreviewFixture(t, "hostile.org")
	doc, err := parser.Parse(source, parser.Options{Adapter: parser.AdapterFallback, BufferID: "buffer-a"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	page, err := render.HTML(doc)
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}
	if strings.Contains(page.Markup, "<script>") || strings.Contains(page.Markup, "onerror=") {
		t.Fatalf("rendered markup contains executable hostile HTML:\n%s", page.Markup)
	}
	if !strings.Contains(page.Markup, "data-org-preview-id=") {
		t.Fatalf("rendered markup missing semantic IDs:\n%s", page.Markup)
	}
	if len(page.Mappings) == 0 {
		t.Fatal("rendered page has no source mappings")
	}
}

func TestOrgPreviewProtocolRejectsStaleRevisionAndReflectedNavigation(t *testing.T) {
	arbiter := protocol.NewRevisionArbiter()
	if !arbiter.Accepts("session-a", "buffer-a", 41) {
		t.Fatal("first revision was rejected")
	}
	if arbiter.Accepts("session-a", "buffer-a", 40) {
		t.Fatal("stale revision was accepted")
	}

	suppressor := protocol.NewLoopSuppressor()
	event := protocol.NavigationEvent{
		Version:   protocol.Version,
		SessionID: "session-a",
		BufferID:  "buffer-a",
		Revision:  41,
		EventID:   "event-1",
		Origin:    protocol.OriginEmacs,
		ElementID: "org-preview-buffer-a-0001",
	}
	if suppressor.SeenOrMark(event) {
		t.Fatal("new navigation event was reported as reflected")
	}
	reflected := event
	reflected.Origin = protocol.OriginBrowser
	if !suppressor.SeenOrMark(reflected) {
		t.Fatal("reflected navigation event was not suppressed")
	}
}

func TestOrgPreviewSessionRegistryIsolatesBuffersAndCleansStateOnClose(t *testing.T) {
	registry := session.NewRegistry()
	first := registry.Open(session.OpenRequest{BufferID: "buffer-a"})
	second := registry.Open(session.OpenRequest{BufferID: "buffer-b"})
	if first.ID == second.ID {
		t.Fatalf("two buffers share session ID %q", first.ID)
	}
	if got := registry.Lookup(first.ID).BufferID; got != "buffer-a" {
		t.Fatalf("first session buffer = %q, want buffer-a", got)
	}
	registry.Close(first.ID)
	if registry.Lookup(first.ID).OK {
		t.Fatal("closed session is still registered")
	}
	if !registry.Lookup(second.ID).OK {
		t.Fatal("closing one session removed another buffer session")
	}
}

func TestOrgPreviewAssetResolverRejectsTraversalSymlinkEscapeAndRemoteSchemes(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "diagram.png"), []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.png"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.png"), filepath.Join(root, "assets", "escape.png")); err != nil {
		t.Fatal(err)
	}

	resolver := assets.NewResolver([]string{root})
	if _, err := resolver.Resolve("assets/diagram.png"); err != nil {
		t.Fatalf("in-root asset rejected: %v", err)
	}
	rejections := []string{
		"../secret.png",
		"%2e%2e/secret.png",
		filepath.Join(root, "assets", "escape.png"),
		"file:///etc/passwd",
		"https://example.com/remote.png",
	}
	for _, candidate := range rejections {
		t.Run(candidate, func(t *testing.T) {
			if _, err := resolver.Resolve(candidate); err == nil {
				t.Fatal("unsafe asset path was accepted")
			}
		})
	}
}

func TestOrgPreviewServerRejectsUnauthenticatedRevisionPost(t *testing.T) {
	instance := startOrgPreviewServer(t)

	response := postJSON(t, instance.URL+"/api/emacs/revisions", "", map[string]any{
		"version":   protocol.Version,
		"sessionId": "session-a",
		"bufferId":  "buffer-a",
		"revision":  1,
		"text":      "* Secret\n",
	})
	defer response.Body.Close()

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
}

func TestOrgPreviewServerRendersUnsavedRevisionWithoutReadingBufferFile(t *testing.T) {
	instance := startOrgPreviewServer(t)

	response := postJSON(t, instance.URL+"/api/emacs/revisions", "test-token", map[string]any{
		"version":   protocol.Version,
		"sessionId": "session-a",
		"bufferId":  "buffer-a",
		"revision":  1,
		"text":      "* Unsaved Heading\nUnsaved body",
	})
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want %d; body=%s", response.StatusCode, http.StatusAccepted, body)
	}

	unauthenticatedRender, err := http.Get(instance.URL + "/api/sessions/session-a/render?revision=1")
	if err != nil {
		t.Fatal(err)
	}
	defer unauthenticatedRender.Body.Close()
	if unauthenticatedRender.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated render status = %d, want %d", unauthenticatedRender.StatusCode, http.StatusUnauthorized)
	}

	pageRequest, err := http.NewRequest(http.MethodGet, instance.URL+"/api/sessions/session-a/render?revision=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	pageRequest.Header.Set("Authorization", "Bearer test-token")
	pageResponse, err := http.DefaultClient.Do(pageRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer pageResponse.Body.Close()
	body, err := io.ReadAll(pageResponse.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Unsaved Heading") || !strings.Contains(string(body), "Unsaved body") {
		t.Fatalf("rendered page does not contain unsaved buffer text:\n%s", body)
	}
}

func TestOrgPreviewServerListensOnLoopbackAddress(t *testing.T) {
	instance := startOrgPreviewServer(t)
	if !strings.HasPrefix(instance.URL, "http://127.0.0.1:") {
		t.Fatalf("server URL = %q, want loopback 127.0.0.1", instance.URL)
	}
}

func readOrgPreviewFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "org-preview", name))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func assertNodeCoveringText(t *testing.T, doc orgpreview.Document, kind orgpreview.Kind, needle string) {
	t.Helper()
	for _, node := range orgpreview.Flatten(doc) {
		if node.Kind == kind && strings.Contains(node.Text, needle) {
			if node.ID == "" {
				t.Fatalf("%s node covering %q has empty ID", kind, needle)
			}
			if node.StartByte >= node.EndByte {
				t.Fatalf("%s node covering %q has invalid range [%d,%d)", kind, needle, node.StartByte, node.EndByte)
			}
			return
		}
	}
	t.Fatalf("no %s node covers %q in normalized IR", kind, needle)
}

func startOrgPreviewServer(t *testing.T) *server.Instance {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	instance, err := server.Start(ctx, server.Config{
		Addr:         "127.0.0.1:0",
		Token:        "test-token",
		AllowedRoots: []string{t.TempDir()},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := instance.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return instance
}

func postJSON(t *testing.T, url string, token string, payload map[string]any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
