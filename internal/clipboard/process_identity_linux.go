//go:build linux

package clipboard

import (
	"fmt"
	"os"
	"strings"
)

func processIdentity(pid int) (string, bool, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 22 {
		return "", false, fmt.Errorf("malformed process stat")
	}
	return fields[21], true, nil
}
