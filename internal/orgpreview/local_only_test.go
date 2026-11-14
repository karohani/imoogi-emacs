package orgpreview

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A preview server is only for this machine's browser: a DNS-rebinding host name
// or a page from another origin is refused even when it carries a valid token.
func TestListenRefusesForeignHostAndOrigin(t *testing.T) {
	s, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, ln, err := s.Listen(ctx, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	self := "127.0.0.1:" + port
	for _, tc := range []struct {
		name, host, origin string
		want               int
	}{
		{"own address", self, "", http.StatusOK},
		{"localhost name", "localhost:" + port, "", http.StatusOK},
		{"same-origin page", self, "http://" + self, http.StatusOK},
		{"rebound host name", "evil.example:" + port, "", http.StatusForbidden},
		{"other port", "127.0.0.1:1", "", http.StatusForbidden},
		{"foreign origin", self, "http://evil.example", http.StatusForbidden},
		{"opaque origin", self, "null", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "http://"+ln.Addr().String()+"/health", nil)
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Fatalf("Host %q Origin %q: %d, want %d", tc.host, tc.origin, resp.StatusCode, tc.want)
			}
		})
	}
}

// Responses that carry document text, file lists or assets must never be stored.
func TestDocumentResponsesAreNotStored(t *testing.T) {
	s, err := NewServer(ServerConfig{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/asset?session=s&buffer=b", "/api/file-preview?session=s&buffer=b", "/api/recent?session=s&buffer=b"} {
		r := httptest.NewRecorder()
		s.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if got := r.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("GET %s (%d): Cache-Control = %q", path, r.Code, got)
		}
	}
}
