//go:build darwin && cgo

package clipboard

/*
#cgo LDFLAGS: -framework AppKit
#include "pasteboard_darwin.h"
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"unsafe"
)

type nativeAdapter struct{}

func NewAdapter() Adapter { return nativeAdapter{} }

func (nativeAdapter) Inspect(_ context.Context) (Inspection, error) {
	identity := fmt.Sprintf("macos:%d", int64(C.imoogi_pasteboard_change_count()))
	value := C.imoogi_pasteboard_inspect_json()
	if value == nil {
		return Inspection{}, fmt.Errorf("inspect pasteboard types")
	}
	defer C.imoogi_pasteboard_free(unsafe.Pointer(value))
	var formats []string
	if err := json.Unmarshal([]byte(C.GoString(value)), &formats); err != nil {
		return Inspection{}, err
	}
	kind := "text"
	for _, format := range formats {
		switch format {
		case "public.file-url", "NSFilenamesPboardType":
			kind = "files"
		case "public.png", "public.tiff":
			if kind != "files" {
				kind = "image"
			}
		}
	}
	return Inspection{ClipboardID: identity, Formats: formats, Kind: kind, Capability: "native"}, nil
}

func (adapter nativeAdapter) Read(ctx context.Context, expectedID string) (ClipboardPayload, error) {
	inspection, err := adapter.Inspect(ctx)
	if err != nil {
		return ClipboardPayload{}, err
	}
	if expectedID == "" || inspection.ClipboardID != expectedID {
		return ClipboardPayload{}, fmt.Errorf("%s", CodeClipboardChanged)
	}
	payload := ClipboardPayload{Inspection: inspection}
	if inspection.Kind == "files" {
		value := C.imoogi_pasteboard_files_json()
		if value == nil {
			return ClipboardPayload{}, fmt.Errorf("read pasteboard files")
		}
		defer C.imoogi_pasteboard_free(unsafe.Pointer(value))
		if err := json.Unmarshal([]byte(C.GoString(value)), &payload.Paths); err != nil {
			return ClipboardPayload{}, err
		}
		if fmt.Sprintf("macos:%d", int64(C.imoogi_pasteboard_change_count())) != expectedID {
			return ClipboardPayload{}, fmt.Errorf("%s", CodeClipboardChanged)
		}
		return payload, nil
	}
	if inspection.Kind == "image" {
		var length C.size_t
		value := C.imoogi_pasteboard_png(&length)
		if value == nil || length == 0 {
			return ClipboardPayload{}, fmt.Errorf("read pasteboard image")
		}
		defer C.imoogi_pasteboard_free(unsafe.Pointer(value))
		payload.Image = C.GoBytes(unsafe.Pointer(value), C.int(length))
		payload.ImageMIME = "image/png"
		if fmt.Sprintf("macos:%d", int64(C.imoogi_pasteboard_change_count())) != expectedID {
			return ClipboardPayload{}, fmt.Errorf("%s", CodeClipboardChanged)
		}
		return payload, nil
	}
	return payload, nil
}
