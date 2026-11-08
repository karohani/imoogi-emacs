package host

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Logger appends one line per request. It records only the timestamp,
// request id, category, status, asset counts, error code and the source
// URL's host. Titles, bodies, full URLs, queries and headers are never
// logged.
type Logger struct {
	mu sync.Mutex
	w  io.Writer
}

// OpenLog opens <dir>/host.log for append (dir 0700, file 0600). On any
// failure it returns a Logger that discards, so logging can never break the
// protocol.
func OpenLog(dir string) (*Logger, io.Closer) {
	if dir == "" {
		return &Logger{w: io.Discard}, io.NopCloser(nil)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return &Logger{w: io.Discard}, io.NopCloser(nil)
	}
	f, err := os.OpenFile(filepath.Join(dir, "host.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return &Logger{w: io.Discard}, io.NopCloser(nil)
	}
	return &Logger{w: f}, f
}

// NewLogger logs to w (tests).
func NewLogger(w io.Writer) *Logger { return &Logger{w: w} }

// Entry is one log record.
type Entry struct {
	RequestID   string
	Type        string
	Category    string
	Status      string
	AssetsSaved int
	AssetsFail  int
	ErrorCode   string
	SourceURL   string // reduced to its host before writing
}

// Log writes e.
func (l *Logger) Log(e Entry) {
	if l == nil {
		return
	}
	line := fmt.Sprintf("%s type=%s request=%s category=%s status=%s assets_saved=%d assets_failed=%d error=%s host=%s\n",
		time.Now().UTC().Format(time.RFC3339), token(e.Type), token(e.RequestID), token(e.Category),
		token(e.Status), e.AssetsSaved, e.AssetsFail, token(e.ErrorCode), hostOnly(e.SourceURL))
	l.mu.Lock()
	defer l.mu.Unlock()
	io.WriteString(l.w, line)
}

// token keeps log fields to a safe alphabet so nothing request-controlled can
// forge a log line.
func token(s string) string {
	if s == "" {
		return "-"
	}
	var b strings.Builder
	for _, r := range s {
		if r < 0x80 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			b.WriteRune(r)
		} else {
			b.WriteByte('?')
		}
		if b.Len() >= 64 {
			break
		}
	}
	return b.String()
}

func hostOnly(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "-"
	}
	return token(strings.ToLower(u.Hostname()))
}
