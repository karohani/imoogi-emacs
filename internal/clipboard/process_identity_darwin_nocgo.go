//go:build darwin && !cgo

package clipboard

import "errors"

func processIdentity(int) (string, bool, error) {
	return "", false, errors.New("process identity validation requires cgo on macOS")
}
