package clipboard

import "context"

type Inspection struct {
	ClipboardID string
	Formats     []string
	Kind        string
	Capability  string
}

type ClipboardPayload struct {
	Inspection Inspection
	Paths      []string
	Image      []byte
	ImageMIME  string
}

type Adapter interface {
	Inspect(context.Context) (Inspection, error)
	Read(context.Context, string) (ClipboardPayload, error)
}
