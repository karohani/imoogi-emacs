package protocol

import (
	"encoding/json"
	"time"
)

// SchemaVersion is the only protocol version this host speaks.
const SchemaVersion = 1

// Request-content limits from PROTOCOL.md § Limits.
const (
	MaxTitleRunes = 1000
	MaxURLRunes   = 4096
	MaxTextBytes  = 4 << 20
	MaxNodes      = 50000
	MaxDepth      = 12
	MaxAssets     = 30
)

// Message types.
const (
	TypeSaveClip = "saveClip"
	TypeStatus   = "status"
	TypePing     = "ping"
)

// Error codes.
const (
	CodeInvalidRequest   = "invalid_request"
	CodeUnknownCategory  = "unknown_category"
	CodeNotConfigured    = "not_configured"
	CodeRootUnavailable  = "root_unavailable"
	CodePermissionDenied = "permission_denied"
	CodeDiskFull         = "disk_full"
	CodeTooLarge         = "too_large"
	CodeInternal         = "internal"
)

// Result statuses.
const (
	StatusSaved      = "saved"
	StatusPartial    = "partial"
	StatusFailed     = "failed"
	StatusInProgress = "in_progress"
	StatusUnknown    = "unknown"
)

// Asset failure reasons.
const (
	ReasonNotHTTPS          = "not_https"
	ReasonUnsupportedScheme = "unsupported_scheme"
	ReasonBlockedAddress    = "blocked_address"
	ReasonDNSFailed         = "dns_failed"
	ReasonTooManyRedirects  = "too_many_redirects"
	ReasonHTTPStatus        = "http_status"
	ReasonTimeout           = "timeout"
	ReasonTooLarge          = "too_large"
	ReasonTotalLimit        = "total_limit"
	ReasonCountLimit        = "count_limit"
	ReasonUnsupportedType   = "unsupported_type"
	ReasonBadSignature      = "bad_signature"
	ReasonTooManyPixels     = "too_many_pixels"
	ReasonDecodeFailed      = "decode_failed"
	ReasonNetwork           = "network"
	ReasonSkipped           = "skipped"
)

var categories = map[string]string{
	"articles":    "00. Inbox/01. Articles",
	"papers":      "00. Inbox/02. Papers",
	"transcripts": "00. Inbox/03. Transcripts",
	"clippings":   "00. Inbox/04. Clippings",
	"ai-research": "00. Inbox/05. AI Research",
}

// CategoryFolder maps a category ID to its root-relative folder (slash
// separated). Only the fixed table is ever consulted.
func CategoryFolder(id string) (string, bool) {
	f, ok := categories[id]
	return f, ok
}

// Inline is one inline AST node. Which fields are meaningful depends on T.
type Inline struct {
	T     string
	V     string   // text, code
	Href  string   // link
	Asset string   // image
	C     []Inline // bold, italic, strike, link
}

// Block is one block AST node. Which fields are meaningful depends on T.
type Block struct {
	T       string
	Level   int          // heading
	Inl     []Inline     // heading, para
	Ordered bool         // list
	Items   [][]Block    // list
	Blocks  []Block      // quote
	Header  bool         // table
	Rows    [][][]Inline // table: row -> cell -> inlines
	Text    string       // code
	Asset   string       // image
}

// Asset is one downloadable image reference.
type Asset struct {
	ID  string
	URL string
}

// SaveClip is a validated saveClip request.
type SaveClip struct {
	RequestID      string
	Category       string
	Title          string // normalised: NFC, single line, trimmed
	SourceURL      string
	ClippedAt      time.Time
	DownloadAssets bool
	Blocks         []Block
	Assets         []Asset
}

// Request is any validated incoming message.
type Request struct {
	Type      string
	RequestID string // saveClip, status
	Save      *SaveClip
}

// Error is a request-level failure carrying a protocol error code. Message is
// always host-authored text and never echoes request content.
type Error struct {
	Code      string
	Message   string
	RequestID string // set only when the request carried a syntactically valid id
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// FailedAsset names one image that was not stored.
type FailedAsset struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// ErrorInfo is the wire form of a request failure.
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Result is the `result` response.
type Result struct {
	Type         string        `json:"type"`
	RequestID    string        `json:"requestId,omitempty"`
	Status       string        `json:"status"`
	Path         string        `json:"path,omitempty"`
	AssetsSaved  int           `json:"assetsSaved"`
	FailedAssets []FailedAsset `json:"failedAssets"`
	Duplicate    bool          `json:"duplicate"`
	Error        *ErrorInfo    `json:"error,omitempty"`
}

// Ack is the `ack` response.
type Ack struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
}

// Pong is the `ping` response. It never carries the root path.
type Pong struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schemaVersion"`
	Configured    bool   `json:"configured"`
	Version       string `json:"version"`
}

// FailedResult builds a `result` with status failed.
func FailedResult(requestID, code, message string) Result {
	return Result{Type: "result", RequestID: requestID, Status: StatusFailed,
		Error: &ErrorInfo{Code: code, Message: message}}
}

// EncodeResponse marshals v and guarantees the output fits MaxOutgoing. For a
// Result that is too large, failedAssets entries are dropped from the end and
// then the error message is truncated; nothing else is altered.
func EncodeResponse(v any) ([]byte, error) {
	if r, ok := v.(Result); ok {
		if r.FailedAssets == nil {
			r.FailedAssets = []FailedAsset{}
		}
		for {
			b, err := json.Marshal(r)
			if err != nil {
				return nil, err
			}
			if len(b) <= MaxOutgoing {
				return b, nil
			}
			switch {
			case len(r.FailedAssets) > 0:
				drop := len(r.FailedAssets) / 2
				if drop == 0 {
					drop = 1
				}
				r.FailedAssets = r.FailedAssets[:len(r.FailedAssets)-drop]
			case r.Error != nil && r.Error.Message != "":
				e := *r.Error
				e.Message = truncateRunes(e.Message, len(e.Message)/2)
				r.Error = &e
			default:
				return nil, &Error{Code: CodeInternal, Message: "response exceeds size limit"}
			}
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(b) > MaxOutgoing {
		return nil, &Error{Code: CodeInternal, Message: "response exceeds size limit"}
	}
	return b, nil
}

// truncateRunes cuts s to at most n bytes without splitting a UTF-8 sequence.
func truncateRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && n < len(s) && s[n]&0xC0 == 0x80 {
		n--
	}
	return s[:n]
}
