package fetch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/image/webp"

	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
)

// Limits from PROTOCOL.md § Limits.
const (
	MaxImageBytes = 10 << 20
	MaxTotalBytes = 50 << 20
	MaxPixels     = 50_000_000
	MaxSide       = 20_000
	MaxRedirects  = 3
)

const userAgent = "Mozilla/5.0 (compatible; OrgClipper/1.0)"

// Options configures a Downloader. The zero value is not usable; start from
// DefaultOptions.
type Options struct {
	// CheckIP decides whether an address may be connected to. It is applied
	// to IP-literal hosts up front and to every resolved address at connect
	// time (each redirect hop included). Production: IsPublic.
	CheckIP func(netip.Addr) bool
	// RootCAs is the TLS trust store; nil means the system pool.
	RootCAs *x509.CertPool
	// Port is the only TCP port allowed. Production: "443".
	Port string
	// Resolver overrides DNS resolution (tests).
	Resolver *net.Resolver

	PerImageTimeout time.Duration
	HeaderTimeout   time.Duration
	Concurrency     int
	MaxCount        int
	MaxImageBytes   int64
	MaxTotalBytes   int64
	MaxPixels       int64
	MaxSide         int
}

// DefaultOptions returns the production configuration.
func DefaultOptions() Options {
	return Options{
		CheckIP:         IsPublic,
		Port:            "443",
		PerImageTimeout: 20 * time.Second,
		HeaderTimeout:   10 * time.Second,
		Concurrency:     4,
		MaxCount:        protocol.MaxAssets,
		MaxImageBytes:   MaxImageBytes,
		MaxTotalBytes:   MaxTotalBytes,
		MaxPixels:       MaxPixels,
		MaxSide:         MaxSide,
	}
}

// Result is the outcome for one asset. Reason is empty on success, in which
// case Data holds the received bytes and Ext the detected type.
type Result struct {
	ID     string
	Data   []byte
	Ext    string // png, jpg, webp
	Reason string
}

// Downloader fetches images. It is safe for concurrent use.
type Downloader struct {
	opts   Options
	client *http.Client
}

// reasonError carries a protocol asset-failure reason through net/http.
type reasonError struct{ reason string }

func (e *reasonError) Error() string { return "refused: " + e.reason }

// New builds a Downloader whose transport refuses non-public peers at
// connect time. No proxy, no cookie jar, no Referer, fixed User-Agent.
func New(o Options) *Downloader {
	d := &Downloader{opts: o}
	dialer := &net.Dialer{
		Timeout:  o.HeaderTimeout,
		Resolver: o.Resolver,
		Control:  d.control,
	}
	tr := &http.Transport{
		Proxy:                  nil, // never honour HTTPS_PROXY and friends
		DialContext:            dialer.DialContext,
		TLSClientConfig:        &tls.Config{RootCAs: o.RootCAs, MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:    o.HeaderTimeout,
		ResponseHeaderTimeout:  o.HeaderTimeout,
		MaxResponseHeaderBytes: 64 << 10,
		ForceAttemptHTTP2:      true,
		MaxIdleConnsPerHost:    o.Concurrency,
		IdleConnTimeout:        30 * time.Second,
	}
	d.client = &http.Client{
		Transport: tr,
		Jar:       nil,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > MaxRedirects {
				return &reasonError{protocol.ReasonTooManyRedirects}
			}
			if reason := d.checkURL(req.URL); reason != "" {
				return &reasonError{reason}
			}
			req.Header.Del("Referer") // net/http adds it on https->https hops
			req.Header.Del("Authorization")
			req.Header.Del("Cookie")
			return nil
		},
	}
	return d
}

// control runs on the actual socket address after DNS resolution, so a name
// that re-resolves to a private address (DNS rebinding) is still refused.
//
// @MX:WARN: [AUTO] security gate executed inside net.Dialer for every connection
// @MX:REASON: removing it (or dialing through a proxy) re-opens SSRF to loopback, LAN and cloud metadata
func (d *Downloader) control(network, address string, _ syscall.RawConn) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return &reasonError{protocol.ReasonBlockedAddress}
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || port != d.opts.Port || !d.opts.CheckIP(ip) {
		return &reasonError{protocol.ReasonBlockedAddress}
	}
	return nil
}

// checkURL validates a URL before any request (and on every redirect hop).
func (d *Downloader) checkURL(u *url.URL) string {
	switch u.Scheme { // url.Parse lowercases the scheme
	case "https":
	case "data", "blob", "file":
		return protocol.ReasonUnsupportedScheme
	default:
		return protocol.ReasonNotHTTPS
	}
	if u.Opaque != "" || u.Host == "" {
		return protocol.ReasonNotHTTPS
	}
	if u.User != nil {
		return protocol.ReasonBlockedAddress
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	if port != d.opts.Port {
		return protocol.ReasonBlockedAddress
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && !d.opts.CheckIP(ip) {
		return protocol.ReasonBlockedAddress
	}
	return ""
}

// FetchAll downloads every asset with bounded concurrency and returns one
// Result per asset, in input order. Assets beyond MaxCount fail with
// count_limit; the clip-wide byte budget is shared across all downloads.
func (d *Downloader) FetchAll(ctx context.Context, assets []protocol.Asset) []Result {
	out := make([]Result, len(assets))
	var total atomic.Int64
	sem := make(chan struct{}, max(1, d.opts.Concurrency))
	var wg sync.WaitGroup
	for i, a := range assets {
		out[i].ID = a.ID
		if i >= d.opts.MaxCount {
			out[i].Reason = protocol.ReasonCountLimit
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			data, ext, reason := d.fetchOne(ctx, a.URL, &total)
			out[i].Data, out[i].Ext, out[i].Reason = data, ext, reason
		}()
	}
	wg.Wait()
	return out
}

var errTotalBudget = errors.New("clip byte budget exhausted")

// budgetReader charges every byte read against the clip-wide total.
type budgetReader struct {
	r     io.Reader
	total *atomic.Int64
	max   int64
}

func (b *budgetReader) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	if n > 0 && b.total.Add(int64(n)) > b.max {
		return n, errTotalBudget
	}
	return n, err
}

func (d *Downloader) fetchOne(ctx context.Context, raw string, total *atomic.Int64) ([]byte, string, string) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, "", protocol.ReasonNotHTTPS
	}
	if reason := d.checkURL(u); reason != "" {
		return nil, "", reason
	}
	ctx, cancel := context.WithTimeout(ctx, d.opts.PerImageTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", protocol.ReasonNetwork
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "image/png,image/jpeg,image/webp")
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", classify(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", protocol.ReasonHTTPStatus
	}
	if resp.ContentLength > d.opts.MaxImageBytes {
		return nil, "", protocol.ReasonTooLarge
	}
	// Limits are measured on bytes actually read, never on Content-Length.
	body := io.LimitReader(&budgetReader{r: resp.Body, total: total, max: d.opts.MaxTotalBytes}, d.opts.MaxImageBytes+1)
	data, err := io.ReadAll(body)
	switch {
	case errors.Is(err, errTotalBudget):
		return nil, "", protocol.ReasonTotalLimit
	case err != nil:
		return nil, "", classify(err)
	case int64(len(data)) > d.opts.MaxImageBytes:
		return nil, "", protocol.ReasonTooLarge
	}
	ext, reason := d.validate(data, resp.Header.Get("Content-Type"))
	if reason != "" {
		return nil, "", reason
	}
	return data, ext, ""
}

// validate identifies the type by magic bytes only, then decodes just the
// header to bound the pixel count. The image is never fully decoded.
func (d *Downloader) validate(data []byte, contentType string) (string, string) {
	ext := sniff(data)
	if ext == "" {
		mt, _, _ := mime.ParseMediaType(contentType)
		switch strings.ToLower(mt) {
		case "image/png", "image/jpeg", "image/jpg", "image/pjpeg", "image/webp":
			return "", protocol.ReasonBadSignature
		}
		return "", protocol.ReasonUnsupportedType
	}
	r := bytes.NewReader(data)
	var w, h int
	switch ext {
	case "png":
		c, err := png.DecodeConfig(r)
		if err != nil {
			return "", protocol.ReasonDecodeFailed
		}
		w, h = c.Width, c.Height
	case "jpg":
		c, err := jpeg.DecodeConfig(r)
		if err != nil {
			return "", protocol.ReasonDecodeFailed
		}
		w, h = c.Width, c.Height
	case "webp":
		c, err := webp.DecodeConfig(r)
		if err != nil {
			return "", protocol.ReasonDecodeFailed
		}
		w, h = c.Width, c.Height
	}
	if w <= 0 || h <= 0 {
		return "", protocol.ReasonDecodeFailed
	}
	if w > d.opts.MaxSide || h > d.opts.MaxSide || int64(w)*int64(h) > d.opts.MaxPixels {
		return "", protocol.ReasonTooManyPixels
	}
	return ext, ""
}

func sniff(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return "png"
	case bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}):
		return "jpg"
	case len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return "webp"
	}
	return ""
}

func classify(err error) string {
	var re *reasonError
	if errors.As(err, &re) {
		return re.reason
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return protocol.ReasonDNSFailed
	}
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
		return protocol.ReasonTimeout
	}
	return protocol.ReasonNetwork
}

// AssetName is the on-disk name for image bytes: the first 32 hex digits of
// their SHA-256 plus the detected extension. Remote names are never used.
func AssetName(data []byte, ext string) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:32] + "." + ext
}
