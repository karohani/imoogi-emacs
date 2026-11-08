// Package protocol implements the Chrome Native Messaging framing and the
// schemaVersion 1 message schema described in PROTOCOL.md.
package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Size limits from PROTOCOL.md § Limits.
const (
	MaxIncoming = 8 << 20  // extension -> host, checked from the prefix
	MaxOutgoing = 64 << 10 // host -> extension
)

// ErrMessageTooLarge is returned when a length prefix exceeds MaxIncoming.
// The body has not been read, so the stream cannot be resynchronised.
var ErrMessageTooLarge = errors.New("incoming message exceeds size limit")

// ReadMessage reads one length-prefixed message. It returns io.EOF only on a
// clean end of stream (no prefix bytes at all) and io.ErrUnexpectedEOF when the
// stream ends inside a prefix or body.
//
// @MX:ANCHOR: [AUTO] the only reader of the untrusted stdin byte stream
// @MX:REASON: the size check must precede the body allocation; moving it would allow an 4 GiB allocation from one prefix
func ReadMessage(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err // io.EOF (clean) or io.ErrUnexpectedEOF (partial prefix)
	}
	n := binary.LittleEndian.Uint32(hdr[:])
	if n > MaxIncoming {
		return nil, ErrMessageTooLarge
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return body, nil
}

// WriteMessage writes one length-prefixed message, refusing anything larger
// than MaxOutgoing without writing a byte.
func WriteMessage(w io.Writer, body []byte) error {
	if len(body) > MaxOutgoing {
		return fmt.Errorf("outgoing message is %d bytes, limit %d", len(body), MaxOutgoing)
	}
	buf := make([]byte, 4+len(body))
	binary.LittleEndian.PutUint32(buf, uint32(len(body)))
	copy(buf[4:], body)
	_, err := w.Write(buf)
	return err
}
