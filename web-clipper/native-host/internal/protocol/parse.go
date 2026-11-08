package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var (
	requestIDRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	assetIDRe   = regexp.MustCompile(`^a[0-9]{1,3}$`)
)

// ValidRequestID reports whether s is a lowercase UUID v4.
func ValidRequestID(s string) bool { return requestIDRe.MatchString(s) }

// Field sets per node type. A node must carry exactly these keys.
var (
	blockFields = map[string][]string{
		"heading": {"t", "level", "inl"},
		"para":    {"t", "inl"},
		"list":    {"t", "ordered", "items"},
		"quote":   {"t", "blocks"},
		"table":   {"t", "header", "rows"},
		"code":    {"t", "text"},
		"image":   {"t", "asset"},
		"hr":      {"t"},
	}
	inlineFields = map[string][]string{
		"text":   {"t", "v"},
		"bold":   {"t", "c"},
		"italic": {"t", "c"},
		"strike": {"t", "c"},
		"code":   {"t", "v"},
		"link":   {"t", "href", "c"},
		"image":  {"t", "asset"},
		"br":     {"t"},
	}
	topFields = map[string][]string{
		TypeSaveClip: {"schemaVersion", "type", "requestId", "category", "title", "sourceUrl",
			"clippedAt", "downloadAssets", "blocks", "assets"},
		TypeStatus: {"schemaVersion", "type", "requestId"},
		TypePing:   {"schemaVersion", "type"},
	}
)

// errInvalid is the internal parse failure; Parse wraps it as an *Error.
type errInvalid string

func (e errInvalid) Error() string { return string(e) }

// parser is a single-pass, token-streaming validator. It never skips an
// unknown value: an unexpected key or token aborts immediately, so the cost of
// a hostile payload is bounded by the bytes actually consumed.
type parser struct {
	dec       *json.Decoder
	nodes     int
	textBytes int
	assetRefs []string
}

// Parse validates one message body and returns the decoded request. Every
// failure is an *Error with code invalid_request or unknown_category.
//
// @MX:ANCHOR: [AUTO] trust boundary: every byte the extension sends is validated here
// @MX:REASON: renderer, store and downloader assume the invariants established by this function
func Parse(body []byte) (*Request, error) {
	if !utf8.Valid(body) {
		return nil, &Error{Code: CodeInvalidRequest, Message: "message is not valid UTF-8"}
	}
	p := &parser{dec: json.NewDecoder(bytes.NewReader(body))}
	p.dec.UseNumber()
	top, err := p.parseTop()
	if err != nil {
		code, msg := CodeInvalidRequest, err.Error()
		var pe *Error
		if errors.As(err, &pe) {
			code, msg = pe.Code, pe.Message
		}
		id := ""
		if top != nil && ValidRequestID(top.requestID) {
			id = top.requestID
		}
		return nil, &Error{Code: code, Message: msg, RequestID: id}
	}
	return top.req, nil
}

type topState struct {
	requestID string
	req       *Request
}

func (p *parser) parseTop() (*topState, error) {
	st := &topState{}
	if err := p.expectDelim('{'); err != nil {
		return st, err
	}
	seen := map[string]bool{}
	var (
		version              int
		typ, category, title string
		sourceURL, clipped   string
		download             bool
		blocks               []Block
		assets               []Asset
	)
	for p.dec.More() {
		key, err := p.key(seen)
		if err != nil {
			return st, err
		}
		switch key {
		case "schemaVersion":
			version, err = p.int()
		case "type":
			typ, err = p.str()
		case "requestId":
			st.requestID, err = p.str()
		case "category":
			category, err = p.str()
		case "title":
			title, err = p.str()
		case "sourceUrl":
			sourceURL, err = p.str()
		case "clippedAt":
			clipped, err = p.str()
		case "downloadAssets":
			download, err = p.bool()
		case "blocks":
			blocks, err = p.blockList(1)
		case "assets":
			assets, err = p.assets()
		default:
			return st, errInvalid("unknown field")
		}
		if err != nil {
			return st, err
		}
	}
	if err := p.expectDelim('}'); err != nil {
		return st, err
	}
	if _, err := p.dec.Token(); err != io.EOF {
		return st, errInvalid("trailing data after message")
	}
	if !seen["schemaVersion"] || version != SchemaVersion {
		return st, errInvalid("unsupported schemaVersion")
	}
	want, ok := topFields[typ]
	if !ok {
		return st, errInvalid("unknown message type")
	}
	if err := exactFields(seen, want); err != nil {
		return st, err
	}
	switch typ {
	case TypePing:
		st.req = &Request{Type: TypePing}
		return st, nil
	case TypeStatus:
		if !ValidRequestID(st.requestID) {
			return st, errInvalid("requestId must be a lowercase UUID v4")
		}
		st.req = &Request{Type: TypeStatus, RequestID: st.requestID}
		return st, nil
	}

	// saveClip
	if !ValidRequestID(st.requestID) {
		return st, errInvalid("requestId must be a lowercase UUID v4")
	}
	if _, ok := CategoryFolder(category); !ok {
		return st, &Error{Code: CodeUnknownCategory, Message: "unknown category"}
	}
	title = NormalizeLine(title)
	if n := utf8.RuneCountInString(title); n < 1 || n > MaxTitleRunes {
		return st, errInvalid("title must be 1..1000 characters")
	}
	if err := checkSourceURL(sourceURL); err != nil {
		return st, err
	}
	at, err := time.Parse(time.RFC3339, clipped)
	if err != nil {
		return st, errInvalid("clippedAt must be an RFC 3339 timestamp")
	}
	ids := make(map[string]bool, len(assets))
	for _, a := range assets {
		if !assetIDRe.MatchString(a.ID) {
			return st, errInvalid("invalid asset id")
		}
		if ids[a.ID] {
			return st, errInvalid("duplicate asset id")
		}
		ids[a.ID] = true
		if a.URL == "" || utf8.RuneCountInString(a.URL) > MaxURLRunes {
			return st, errInvalid("invalid asset url length")
		}
	}
	for _, ref := range p.assetRefs {
		if !ids[ref] {
			return st, errInvalid("image references an unknown asset")
		}
	}
	st.req = &Request{Type: TypeSaveClip, RequestID: st.requestID, Save: &SaveClip{
		RequestID: st.requestID, Category: category, Title: title, SourceURL: sourceURL,
		ClippedAt: at, DownloadAssets: download, Blocks: blocks, Assets: assets,
	}}
	return st, nil
}

func checkSourceURL(s string) error {
	if s == "" || utf8.RuneCountInString(s) > MaxURLRunes {
		return errInvalid("sourceUrl length out of range")
	}
	u, err := url.Parse(s)
	if err != nil {
		return errInvalid("sourceUrl does not parse")
	}
	scheme := strings.ToLower(u.Scheme)
	if (scheme != "http" && scheme != "https") || u.Host == "" {
		return errInvalid("sourceUrl must be an http or https URL")
	}
	return nil
}

// NormalizeLine NFC-normalises s, turns every control or line-separator rune
// into a space, collapses whitespace runs and trims. The result is safe to
// place on a single line.
func NormalizeLine(s string) string {
	s = norm.NFC.String(s)
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		if unicode.IsControl(r) || unicode.IsSpace(r) || r == ' ' || r == ' ' {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	return b.String()
}

func exactFields(seen map[string]bool, want []string) error {
	if len(seen) != len(want) {
		if len(seen) > len(want) {
			return errInvalid("unexpected field for node type")
		}
		return errInvalid("missing required field")
	}
	for _, f := range want {
		if !seen[f] {
			return errInvalid("unexpected field for node type")
		}
	}
	return nil
}

func (p *parser) countNode(depth int) error {
	if depth > MaxDepth {
		return errInvalid("nesting depth exceeds limit")
	}
	p.nodes++
	if p.nodes > MaxNodes {
		return errInvalid("node count exceeds limit")
	}
	return nil
}

func (p *parser) addText(s string) error {
	p.textBytes += len(s)
	if p.textBytes > MaxTextBytes {
		return errInvalid("total text exceeds limit")
	}
	return nil
}

func (p *parser) blockList(depth int) ([]Block, error) {
	if err := p.expectDelim('['); err != nil {
		return nil, err
	}
	out := []Block{}
	for p.dec.More() {
		b, err := p.block(depth)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, p.expectDelim(']')
}

func (p *parser) block(depth int) (Block, error) {
	var b Block
	if err := p.countNode(depth); err != nil {
		return b, err
	}
	if err := p.expectDelim('{'); err != nil {
		return b, err
	}
	seen := map[string]bool{}
	for p.dec.More() {
		key, err := p.key(seen)
		if err != nil {
			return b, err
		}
		switch key {
		case "t":
			b.T, err = p.str()
		case "level":
			b.Level, err = p.int()
		case "inl":
			b.Inl, err = p.inlineList(depth + 1)
		case "ordered":
			b.Ordered, err = p.bool()
		case "items":
			b.Items, err = p.items(depth + 1)
		case "blocks":
			b.Blocks, err = p.blockList(depth + 1)
		case "header":
			b.Header, err = p.bool()
		case "rows":
			b.Rows, err = p.rows(depth + 1)
		case "text":
			if b.Text, err = p.str(); err == nil {
				err = p.addText(b.Text)
			}
		case "asset":
			b.Asset, err = p.str()
		default:
			return b, errInvalid("unknown block field")
		}
		if err != nil {
			return b, err
		}
	}
	if err := p.expectDelim('}'); err != nil {
		return b, err
	}
	want, ok := blockFields[b.T]
	if !ok {
		return b, errInvalid("unknown block type")
	}
	if err := exactFields(seen, want); err != nil {
		return b, err
	}
	switch b.T {
	case "heading":
		if b.Level < 1 || b.Level > 6 {
			return b, errInvalid("heading level must be 1..6")
		}
	case "image":
		p.assetRefs = append(p.assetRefs, b.Asset)
	}
	return b, nil
}

func (p *parser) items(depth int) ([][]Block, error) {
	if err := p.expectDelim('['); err != nil {
		return nil, err
	}
	out := [][]Block{}
	for p.dec.More() {
		item, err := p.blockList(depth)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, p.expectDelim(']')
}

func (p *parser) rows(depth int) ([][][]Inline, error) {
	if err := p.expectDelim('['); err != nil {
		return nil, err
	}
	out := [][][]Inline{}
	for p.dec.More() {
		if err := p.expectDelim('['); err != nil {
			return nil, err
		}
		row := [][]Inline{}
		for p.dec.More() {
			cell, err := p.inlineList(depth)
			if err != nil {
				return nil, err
			}
			row = append(row, cell)
		}
		if err := p.expectDelim(']'); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, p.expectDelim(']')
}

func (p *parser) inlineList(depth int) ([]Inline, error) {
	if err := p.expectDelim('['); err != nil {
		return nil, err
	}
	out := []Inline{}
	for p.dec.More() {
		in, err := p.inline(depth)
		if err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, p.expectDelim(']')
}

func (p *parser) inline(depth int) (Inline, error) {
	var in Inline
	if err := p.countNode(depth); err != nil {
		return in, err
	}
	if err := p.expectDelim('{'); err != nil {
		return in, err
	}
	seen := map[string]bool{}
	for p.dec.More() {
		key, err := p.key(seen)
		if err != nil {
			return in, err
		}
		switch key {
		case "t":
			in.T, err = p.str()
		case "v":
			if in.V, err = p.str(); err == nil {
				err = p.addText(in.V)
			}
		case "href":
			if in.Href, err = p.str(); err == nil {
				err = p.addText(in.Href)
			}
		case "asset":
			in.Asset, err = p.str()
		case "c":
			in.C, err = p.inlineList(depth + 1)
		default:
			return in, errInvalid("unknown inline field")
		}
		if err != nil {
			return in, err
		}
	}
	if err := p.expectDelim('}'); err != nil {
		return in, err
	}
	want, ok := inlineFields[in.T]
	if !ok {
		return in, errInvalid("unknown inline type")
	}
	if err := exactFields(seen, want); err != nil {
		return in, err
	}
	if in.T == "image" {
		p.assetRefs = append(p.assetRefs, in.Asset)
	}
	return in, nil
}

func (p *parser) assets() ([]Asset, error) {
	if err := p.expectDelim('['); err != nil {
		return nil, err
	}
	out := []Asset{}
	for p.dec.More() {
		if len(out) == MaxAssets {
			return nil, errInvalid("too many assets")
		}
		if err := p.expectDelim('{'); err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		var a Asset
		for p.dec.More() {
			key, err := p.key(seen)
			if err != nil {
				return nil, err
			}
			switch key {
			case "id":
				a.ID, err = p.str()
			case "url":
				a.URL, err = p.str()
			default:
				return nil, errInvalid("unknown asset field")
			}
			if err != nil {
				return nil, err
			}
		}
		if err := p.expectDelim('}'); err != nil {
			return nil, err
		}
		if err := exactFields(seen, []string{"id", "url"}); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, p.expectDelim(']')
}

// key reads an object key and rejects duplicates.
func (p *parser) key(seen map[string]bool) (string, error) {
	tok, err := p.dec.Token()
	if err != nil {
		return "", errInvalid("malformed JSON")
	}
	k, ok := tok.(string)
	if !ok {
		return "", errInvalid("malformed JSON")
	}
	if seen[k] {
		return "", errInvalid("duplicate field")
	}
	seen[k] = true
	return k, nil
}

func (p *parser) expectDelim(d json.Delim) error {
	tok, err := p.dec.Token()
	if err != nil {
		return errInvalid("malformed JSON")
	}
	if got, ok := tok.(json.Delim); !ok || got != d {
		return errInvalid("wrong JSON type")
	}
	return nil
}

func (p *parser) str() (string, error) {
	tok, err := p.dec.Token()
	if err != nil {
		return "", errInvalid("malformed JSON")
	}
	s, ok := tok.(string)
	if !ok {
		return "", errInvalid("expected a string")
	}
	return s, nil
}

func (p *parser) bool() (bool, error) {
	tok, err := p.dec.Token()
	if err != nil {
		return false, errInvalid("malformed JSON")
	}
	v, ok := tok.(bool)
	if !ok {
		return false, errInvalid("expected a boolean")
	}
	return v, nil
}

func (p *parser) int() (int, error) {
	tok, err := p.dec.Token()
	if err != nil {
		return 0, errInvalid("malformed JSON")
	}
	n, ok := tok.(json.Number)
	if !ok {
		return 0, errInvalid("expected an integer")
	}
	v, err := strconv.Atoi(string(n))
	if err != nil {
		return 0, errInvalid("expected an integer")
	}
	return v, nil
}
