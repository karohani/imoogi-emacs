package fetch

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 4, 3)), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// A 1x1 lossless WebP.
var webp1x1, _ = base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")

// pngBomb is a tiny, well-formed PNG header that declares w x h pixels.
func pngBomb(w, h uint32) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6 // bit depth 8, RGBA
	chunk := append([]byte("IHDR"), ihdr...)
	binary.Write(&b, binary.BigEndian, uint32(13))
	b.Write(chunk)
	binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

// jpegBomb is SOI + APP0/JFIF + a baseline SOF0 header declaring w x h.
func jpegBomb(w, h uint16) []byte {
	b := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00}
	b = append(b, 0xFF, 0xC0, 0x00, 0x0B, 0x08)
	b = binary.BigEndian.AppendUint16(b, h)
	b = binary.BigEndian.AppendUint16(b, w)
	return append(b, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xD9)
}

type harness struct {
	srv  *httptest.Server
	base string
	d    *Downloader
	mux  *http.ServeMux
}

// newHarness starts a TLS test server on 127.0.0.1 and a downloader whose IP
// policy and trust store are swapped for the test (production blocks
// loopback and trusts only the system pool).
func newHarness(t *testing.T, mut func(*Options)) *harness {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewUnstartedServer(mux)
	srv.Config.ErrorLog = log.New(io.Discard, "", 0) // refused handshakes are expected
	srv.StartTLS()
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	pool := srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	opts := DefaultOptions()
	opts.CheckIP = func(a netip.Addr) bool { return a == netip.MustParseAddr("127.0.0.1") }
	opts.RootCAs = pool
	opts.Port = u.Port()
	if mut != nil {
		mut(&opts)
	}
	return &harness{srv: srv, base: srv.URL, d: New(opts), mux: mux}
}

func (h *harness) fetch(urls ...string) []Result {
	var as []protocol.Asset
	for i, u := range urls {
		as = append(as, protocol.Asset{ID: fmt.Sprintf("a%d", i+1), URL: u})
	}
	return h.d.FetchAll(context.Background(), as)
}

func serve(ct string, body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ct)
		w.Write(body)
	}
}

func TestFetchValidImages(t *testing.T) {
	h := newHarness(t, nil)
	p := pngBytes(t, 3, 2)
	h.mux.Handle("/a.png", serve("image/png", p))
	h.mux.Handle("/b", serve("application/octet-stream", jpegBytes(t)))
	h.mux.Handle("/c.webp", serve("image/webp", webp1x1))
	res := h.fetch(h.base+"/a.png", h.base+"/b", h.base+"/c.webp")
	for i, want := range []string{"png", "jpg", "webp"} {
		if res[i].Reason != "" || res[i].Ext != want {
			t.Fatalf("asset %d: %+v", i, res[i])
		}
	}
	if !bytes.Equal(res[0].Data, p) || res[0].ID != "a1" {
		t.Fatal("data/id mismatch")
	}
	if n := AssetName(res[0].Data, res[0].Ext); len(n) != 36 || !strings.HasSuffix(n, ".png") {
		t.Fatalf("asset name %q", n)
	}
}

func TestFetchURLPolicy(t *testing.T) {
	h := newHarness(t, nil)
	cases := map[string]string{
		"http://example.com/x.png":                  protocol.ReasonNotHTTPS,
		"ftp://example.com/x.png":                   protocol.ReasonNotHTTPS,
		"HTTP://example.com/x.png":                  protocol.ReasonNotHTTPS,
		"data:image/png;base64,AAAA":                protocol.ReasonUnsupportedScheme,
		"blob:https://example.com/uuid":             protocol.ReasonUnsupportedScheme,
		"file:///etc/passwd":                        protocol.ReasonUnsupportedScheme,
		"https://user:pw@127.0.0.1/x":               protocol.ReasonBlockedAddress,
		"https://10.0.0.1/x.png":                    protocol.ReasonBlockedAddress,
		"https://[::1]/x.png":                       protocol.ReasonBlockedAddress,
		"https://169.254.169.254/latest/meta-data/": protocol.ReasonBlockedAddress,
		"https://127.0.0.1:1/x.png":                 protocol.ReasonBlockedAddress, // wrong port
		"https:opaque":                              protocol.ReasonNotHTTPS,
		"::not a url":                               protocol.ReasonNotHTTPS,
	}
	for raw, want := range cases {
		if got := h.fetch(raw)[0].Reason; got != want {
			t.Errorf("%s: got %q want %q", raw, got, want)
		}
	}
}

func TestProductionDefaultsBlockLoopbackAndPort(t *testing.T) {
	// With the production policy, the test server (127.0.0.1, random port) is
	// unreachable on both counts.
	srv := httptest.NewTLSServer(serve("image/png", []byte("x")))
	defer srv.Close()
	d := New(DefaultOptions())
	res := d.FetchAll(context.Background(), []protocol.Asset{{ID: "a1", URL: srv.URL + "/x.png"}})
	if res[0].Reason != protocol.ReasonBlockedAddress {
		t.Fatalf("got %+v", res[0])
	}
	u, _ := url.Parse(srv.URL)
	opts := DefaultOptions()
	opts.Port = u.Port()
	res = New(opts).FetchAll(context.Background(), []protocol.Asset{{ID: "a1", URL: "https://localhost:" + u.Port() + "/x.png"}})
	if res[0].Reason != protocol.ReasonBlockedAddress {
		t.Fatalf("hostname resolving to loopback not blocked at connect time: %+v", res[0])
	}
}

func TestDialTimeCheckDefeatsHostnames(t *testing.T) {
	// The hostname passes the up-front literal check; the connect-time check
	// on the resolved address must still refuse it.
	var hits atomic.Int32
	h := newHarness(t, func(o *Options) {
		o.CheckIP = func(netip.Addr) bool { hits.Add(1); return false }
	})
	u, _ := url.Parse(h.base)
	res := h.fetch("https://localhost:" + u.Port() + "/x.png")
	if res[0].Reason != protocol.ReasonBlockedAddress || hits.Load() == 0 {
		t.Fatalf("got %+v hits=%d", res[0], hits.Load())
	}
}

func TestRedirects(t *testing.T) {
	var referers, cookies []string
	var mu sync.Mutex
	h := newHarness(t, nil)
	p := pngBytes(t, 1, 1)
	h.mux.HandleFunc("/hop/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		referers = append(referers, r.Header.Get("Referer"))
		cookies = append(cookies, r.Header.Get("Cookie"))
		mu.Unlock()
		if ua := r.Header.Get("User-Agent"); ua != userAgent {
			t.Errorf("user agent %q", ua)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("auth header sent")
		}
		n := strings.TrimPrefix(r.URL.Path, "/hop/")
		if n == "0" {
			w.Header().Set("Content-Type", "image/png")
			w.Write(p)
			return
		}
		var next int
		fmt.Sscan(n, &next)
		http.SetCookie(w, &http.Cookie{Name: "s", Value: "secret"})
		http.Redirect(w, r, fmt.Sprintf("/hop/%d", next-1), http.StatusFound)
	})
	h.mux.HandleFunc("/to-http", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1/x", http.StatusFound)
	})
	h.mux.HandleFunc("/to-private", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://192.168.0.1/x", http.StatusFound)
	})
	h.mux.HandleFunc("/to-file", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "file:///etc/passwd", http.StatusFound)
	})
	h.mux.HandleFunc("/to-userinfo", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, strings.Replace(h.base, "https://", "https://u:p@", 1)+"/hop/0", http.StatusFound)
	})
	res := h.fetch(h.base+"/hop/3", h.base+"/hop/4", h.base+"/to-http", h.base+"/to-private", h.base+"/to-file", h.base+"/to-userinfo")
	want := []string{"", protocol.ReasonTooManyRedirects, protocol.ReasonNotHTTPS, protocol.ReasonBlockedAddress,
		protocol.ReasonUnsupportedScheme, protocol.ReasonBlockedAddress}
	for i, w := range want {
		if res[i].Reason != w {
			t.Errorf("case %d: got %q want %q", i, res[i].Reason, w)
		}
	}
	for _, r := range referers {
		if r != "" {
			t.Errorf("Referer leaked: %q", r)
		}
	}
	for _, c := range cookies {
		if c != "" {
			t.Errorf("cookie replayed: %q", c)
		}
	}
}

func TestSizeLimits(t *testing.T) {
	h := newHarness(t, nil)
	big := append(pngBytes(t, 1, 1), bytes.Repeat([]byte{0}, MaxImageBytes)...)
	h.mux.HandleFunc("/chunked", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		// No Content-Length: the body is streamed in chunks.
		for off := 0; off < len(big); off += 1 << 20 {
			w.Write(big[off:min(off+1<<20, len(big))])
			w.(http.Flusher).Flush()
		}
	})
	h.mux.HandleFunc("/declared", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", fmt.Sprint(len(big)))
		w.Write(big)
	})
	res := h.fetch(h.base+"/chunked", h.base+"/declared")
	for i, r := range res {
		if r.Reason != protocol.ReasonTooLarge {
			t.Errorf("case %d: got %q", i, r.Reason)
		}
	}
}

func TestLyingContentLengthIsNotTrusted(t *testing.T) {
	// Raw responses whose Content-Length disagrees with the bytes on the wire.
	h := newHarness(t, func(o *Options) { o.MaxImageBytes = 1024 })
	p := pngBytes(t, 1, 1)
	raw := func(header string, body []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			conn, buf, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			buf.WriteString("HTTP/1.1 200 OK\r\nContent-Type: image/png\r\nConnection: close\r\n" + header + "\r\n")
			buf.Write(body)
			buf.Flush()
		}
	}
	long := append(append([]byte{}, p...), bytes.Repeat([]byte{1}, 8192)...)
	// Declares 100 bytes, sends 8 KiB: never more than the declared 100 are read.
	h.mux.Handle("/under", raw("Content-Length: 100\r\n", long))
	// Declares nothing, streams past the limit until the server closes.
	h.mux.Handle("/none", raw("", long))
	// Declares a size within the limit but is a chunked stream past it.
	h.mux.Handle("/chunked", raw("Transfer-Encoding: chunked\r\n", []byte(fmt.Sprintf("%x\r\n%s\r\n0\r\n\r\n", len(long), long))))
	res := h.fetch(h.base+"/under", h.base+"/none", h.base+"/chunked")
	if res[0].Reason != "" || len(res[0].Data) != 100 {
		t.Errorf("under: reason %q, %d bytes", res[0].Reason, len(res[0].Data))
	}
	if res[1].Reason != protocol.ReasonTooLarge || res[2].Reason != protocol.ReasonTooLarge {
		t.Errorf("none=%q chunked=%q, want too_large", res[1].Reason, res[2].Reason)
	}
}

func TestTotalLimit(t *testing.T) {
	h := newHarness(t, func(o *Options) {
		o.MaxImageBytes = 100 << 10
		o.MaxTotalBytes = 250 << 10
	})
	body := append(pngBytes(t, 1, 1), bytes.Repeat([]byte{0}, 90<<10)...)
	h.mux.Handle("/x", serve("image/png", body))
	var urls []string
	for range 4 {
		urls = append(urls, h.base+"/x")
	}
	res := h.fetch(urls...)
	ok, total := 0, 0
	for _, r := range res {
		switch r.Reason {
		case "":
			ok++
		case protocol.ReasonTotalLimit:
			total++
		default:
			t.Errorf("unexpected %q", r.Reason)
		}
	}
	if ok > 2 || total < 2 {
		t.Fatalf("ok=%d total_limit=%d; 4x90KiB must exceed a 250KiB budget", ok, total)
	}
}

func TestCountLimit(t *testing.T) {
	h := newHarness(t, func(o *Options) { o.MaxCount = 2 })
	h.mux.Handle("/x", serve("image/png", pngBytes(t, 1, 1)))
	res := h.fetch(h.base+"/x", h.base+"/x", h.base+"/x")
	if res[0].Reason != "" || res[1].Reason != "" || res[2].Reason != protocol.ReasonCountLimit {
		t.Fatalf("%+v", res)
	}
}

func TestTypeChecks(t *testing.T) {
	h := newHarness(t, nil)
	h.mux.Handle("/html-as-png", serve("image/png", []byte("<!doctype html><script>alert(1)</script>")))
	h.mux.Handle("/svg", serve("image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`)))
	h.mux.Handle("/gif", serve("image/gif", []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")))
	h.mux.Handle("/truncated", serve("image/png", pngBytes(t, 2, 2)[:20]))
	h.mux.Handle("/png-bomb", serve("image/png", pngBomb(30000, 30000)))
	h.mux.Handle("/png-50mp", serve("image/png", pngBomb(10000, 6000)))
	h.mux.Handle("/png-side", serve("image/png", pngBomb(20001, 10)))
	h.mux.Handle("/jpeg-bomb", serve("image/jpeg", jpegBomb(60000, 60000)))
	h.mux.Handle("/png-ok-max", serve("image/png", pngBomb(20000, 2500)))
	h.mux.Handle("/status", http.NotFoundHandler())
	res := h.fetch(h.base+"/html-as-png", h.base+"/svg", h.base+"/gif", h.base+"/truncated",
		h.base+"/png-bomb", h.base+"/png-50mp", h.base+"/png-side", h.base+"/jpeg-bomb", h.base+"/png-ok-max", h.base+"/status")
	want := []string{protocol.ReasonBadSignature, protocol.ReasonUnsupportedType, protocol.ReasonUnsupportedType,
		protocol.ReasonDecodeFailed, protocol.ReasonTooManyPixels, protocol.ReasonTooManyPixels,
		protocol.ReasonTooManyPixels, protocol.ReasonTooManyPixels, "", protocol.ReasonHTTPStatus}
	for i, w := range want {
		if res[i].Reason != w {
			t.Errorf("case %d: got %q want %q", i, res[i].Reason, w)
		}
	}
}

func TestTimeouts(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	h := newHarness(t, func(o *Options) {
		o.HeaderTimeout = 200 * time.Millisecond
		o.PerImageTimeout = 400 * time.Millisecond
	})
	h.mux.HandleFunc("/slow-header", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	h.mux.HandleFunc("/slow-body", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngBytes(t, 1, 1)[:10])
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	start := time.Now()
	res := h.fetch(h.base+"/slow-header", h.base+"/slow-body")
	for i, r := range res {
		if r.Reason != protocol.ReasonTimeout {
			t.Errorf("case %d: got %q", i, r.Reason)
		}
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("timeouts not enforced")
	}
}

func TestConcurrencyBound(t *testing.T) {
	var cur, peak atomic.Int32
	h := newHarness(t, nil)
	p := pngBytes(t, 1, 1)
	h.mux.HandleFunc("/x", func(w http.ResponseWriter, r *http.Request) {
		n := cur.Add(1)
		for {
			old := peak.Load()
			if n <= old || peak.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		cur.Add(-1)
		w.Header().Set("Content-Type", "image/png")
		w.Write(p)
	})
	var urls []string
	for range 12 {
		urls = append(urls, h.base+"/x")
	}
	for _, r := range h.fetch(urls...) {
		if r.Reason != "" {
			t.Fatalf("%+v", r)
		}
	}
	if peak.Load() > 4 {
		t.Fatalf("peak concurrency %d > 4", peak.Load())
	}
}

func TestDNSFailureAndNetwork(t *testing.T) {
	h := newHarness(t, func(o *Options) {
		o.Resolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return nil, errors.New("no dns in tests")
		}}
	})
	u, _ := url.Parse(h.base)
	if r := h.fetch("https://does-not-exist.invalid:" + u.Port() + "/x")[0]; r.Reason != protocol.ReasonDNSFailed {
		t.Fatalf("dns: %+v", r)
	}
	// Untrusted certificate -> network.
	h2 := newHarness(t, func(o *Options) { o.RootCAs = nil })
	if r := h2.fetch(h2.base + "/x")[0]; r.Reason != protocol.ReasonNetwork {
		t.Fatalf("tls: %+v", r)
	}
}

func TestCancelledContext(t *testing.T) {
	h := newHarness(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res := h.d.FetchAll(ctx, []protocol.Asset{{ID: "a1", URL: h.base + "/x"}})
	if res[0].Reason == "" {
		t.Fatal("cancelled fetch succeeded")
	}
}
