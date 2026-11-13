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
	for _, path := range []string{"/static/reader-pagination.js", "/static/reader-view.js", "/static/reader.css"} {
		t.Run(path, func(t *testing.T) {
			r := httptest.NewRecorder()
			s.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
			if r.Code != 200 || r.Body.Len() == 0 || r.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("GET %s: %d %s", path, r.Code, r.Body.String())
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
