//go:build !darwin && !linux

package clipboard

import "errors"

func processIdentity(int) (string, bool, error) {
	return "", false, errors.New("process identity validation is unsupported on this platform")
}
