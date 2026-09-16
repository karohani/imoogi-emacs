//go:build !darwin || !cgo

package clipboard

import (
	"context"
	"fmt"
)

type unsupportedAdapter struct{}

func NewAdapter() Adapter { return unsupportedAdapter{} }

func (unsupportedAdapter) Inspect(context.Context) (Inspection, error) {
	return Inspection{Capability: string(CodeUnsupportedCapability)}, fmt.Errorf("%s", CodeUnsupportedCapability)
}

func (unsupportedAdapter) Read(context.Context, string) (ClipboardPayload, error) {
	return ClipboardPayload{}, fmt.Errorf("%s", CodeUnsupportedCapability)
}
