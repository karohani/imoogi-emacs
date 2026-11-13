package orgpreview

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReaderStaticAssets(t *testing.T) {
	s, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/static/reader-pagination.js", "/static/reader-view.js", "/static/reader.css", "/static/mermaid-canvas.js"} {
		t.Run(path, func(t *testing.T) {
			r := httptest.NewRecorder()
			s.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
			if r.Code != 200 || r.Body.Len() == 0 || r.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("GET %s: %d %s", path, r.Code, r.Body.String())
			}
			// A rebuilt binary must reach an open tab without a hard reload.
			if got := r.Header().Get("Cache-Control"); got != "no-cache" {
				t.Fatalf("GET %s Cache-Control = %q", path, got)
			}
			if !strings.Contains(browserShell, path) {
				t.Fatalf("shell missing %s", path)
			}
			r = httptest.NewRecorder()
			s.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodPost, path, nil))
			if r.Code != 405 {
				t.Fatalf("POST: %d", r.Code)
			}
		})
	}
}

func TestBrowserShellPutsRenderedMermaidOnACanvas(t *testing.T) {
	for _, want := range []string{
		"const mermaidCanvas = OrgMermaidCanvas.create({document, root});",
		"rendered:node => mermaidCanvas.attach(node),",
		".mermaid-viewport{",
		"body[data-reader-mode=book] .mermaid-toolbar",
	} {
		if !strings.Contains(browserShell, want) {
			t.Fatalf("shell missing %q", want)
		}
	}
}

func TestPreviewPageAndDiagnosticsRevalidate(t *testing.T) {
	s, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/preview", "/static/mermaid-diagnostics.js"} {
		r := httptest.NewRecorder()
		s.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != 200 || r.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("GET %s: %d Cache-Control=%q", path, r.Code, r.Header().Get("Cache-Control"))
		}
	}
}
