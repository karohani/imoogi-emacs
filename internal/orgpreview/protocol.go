package orgpreview

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

type Origin string

const (
	OriginEmacs   Origin = "emacs"
	OriginBrowser Origin = "browser"
	OriginServer  Origin = "server"
)

var idRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type Envelope struct {
	Version   string `json:"version"`
	SessionID string `json:"session_id"`
	BufferID  string `json:"buffer_id"`
	Revision  int64  `json:"revision"`
	EventID   string `json:"event_id"`
	Origin    Origin `json:"origin"`
}

func (r *RevisionRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		Version           string   `json:"version"`
		SessionID         string   `json:"session_id"`
		SessionIDCamel    string   `json:"sessionId"`
		BufferID          string   `json:"buffer_id"`
		BufferIDCamel     string   `json:"bufferId"`
		Revision          int64    `json:"revision"`
		EventID           string   `json:"event_id"`
		EventIDCamel      string   `json:"eventId"`
		Origin            Origin   `json:"origin"`
		Path              string   `json:"path"`
		Text              string   `json:"text"`
		CursorByte        int      `json:"cursor_byte"`
		CursorByteCamel   int      `json:"cursorByte"`
		AllowedRoots      []string `json:"allowed_roots"`
		AllowedRootsCamel []string `json:"allowedRoots"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Envelope = Envelope{
		Version:   raw.Version,
		SessionID: firstNonEmpty(raw.SessionID, raw.SessionIDCamel),
		BufferID:  firstNonEmpty(raw.BufferID, raw.BufferIDCamel),
		Revision:  raw.Revision,
		EventID:   firstNonEmpty(raw.EventID, raw.EventIDCamel),
		Origin:    raw.Origin,
	}
	r.Path = raw.Path
	r.Text = raw.Text
	r.CursorByte = raw.CursorByte
	if raw.CursorByteCamel != 0 {
		r.CursorByte = raw.CursorByteCamel
	}
	r.AllowedRoots = raw.AllowedRoots
	if len(raw.AllowedRootsCamel) > 0 {
		r.AllowedRoots = raw.AllowedRootsCamel
	}
	return nil
}

func (e *NavigationEvent) UnmarshalJSON(data []byte) error {
	var raw struct {
		Version         string      `json:"version"`
		SessionID       string      `json:"session_id"`
		SessionIDCamel  string      `json:"sessionId"`
		BufferID        string      `json:"buffer_id"`
		BufferIDCamel   string      `json:"bufferId"`
		Revision        int64       `json:"revision"`
		EventID         string      `json:"event_id"`
		EventIDCamel    string      `json:"eventId"`
		Origin          Origin      `json:"origin"`
		ElementID       string      `json:"element_id"`
		ElementIDCamel  string      `json:"elementId"`
		CursorByte      int         `json:"cursor_byte"`
		CursorByteCamel int         `json:"cursorByte"`
		Range           SourceRange `json:"range"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.Envelope = Envelope{
		Version:   raw.Version,
		SessionID: firstNonEmpty(raw.SessionID, raw.SessionIDCamel),
		BufferID:  firstNonEmpty(raw.BufferID, raw.BufferIDCamel),
		Revision:  raw.Revision,
		EventID:   firstNonEmpty(raw.EventID, raw.EventIDCamel),
		Origin:    raw.Origin,
	}
	e.ElementID = firstNonEmpty(raw.ElementID, raw.ElementIDCamel)
	e.CursorByte = raw.CursorByte
	if raw.CursorByteCamel != 0 {
		e.CursorByte = raw.CursorByteCamel
	}
	e.Range = raw.Range
	return nil
}

type RevisionRequest struct {
	Envelope
	Path         string   `json:"path,omitempty"`
	Text         string   `json:"text"`
	CursorByte   int      `json:"cursor_byte,omitempty"`
	AllowedRoots []string `json:"allowed_roots,omitempty"`
}

type RevisionResponse struct {
	Envelope
	Committed bool     `json:"committed"`
	Stale     bool     `json:"stale,omitempty"`
	Document  Document `json:"document,omitempty"`
	HTML      string   `json:"html,omitempty"`
}

type NavigationEvent struct {
	Envelope
	ElementID  string      `json:"element_id"`
	CursorByte int         `json:"cursor_byte,omitempty"`
	Range      SourceRange `json:"range,omitempty"`
}

type RenderEvent struct {
	Envelope
	Document Document `json:"document"`
	HTML     string   `json:"html"`
}

type ErrorResponse struct {
	Version string `json:"version"`
	Error   string `json:"error"`
}

func NewEnvelope(sessionID, bufferID string, revision int64, origin Origin, eventID string) Envelope {
	if eventID == "" {
		eventID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return Envelope{
		Version:   ProtocolVersion,
		SessionID: sessionID,
		BufferID:  bufferID,
		Revision:  revision,
		EventID:   eventID,
		Origin:    origin,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (e Envelope) Validate() error {
	if e.Version != ProtocolVersion {
		return fmt.Errorf("unsupported protocol version %q", e.Version)
	}
	if !idRE.MatchString(e.SessionID) {
		return errors.New("invalid session_id")
	}
	if !idRE.MatchString(e.BufferID) {
		return errors.New("invalid buffer_id")
	}
	if e.Revision < 0 {
		return errors.New("revision must be non-negative")
	}
	if !idRE.MatchString(e.EventID) {
		return errors.New("invalid event_id")
	}
	switch e.Origin {
	case OriginEmacs, OriginBrowser, OriginServer:
		return nil
	default:
		return fmt.Errorf("invalid origin %q", e.Origin)
	}
}
