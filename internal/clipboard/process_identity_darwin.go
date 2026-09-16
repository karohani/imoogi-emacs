//go:build darwin

package clipboard

/*
#include <libproc.h>
#include <stdio.h>

static int imoogi_process_start(int pid, char *buffer, int size) {
    struct proc_bsdinfo info;
    int result = proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, &info, sizeof(info));
    if (result != sizeof(info)) {
        return 0;
    }
    return snprintf(buffer, size, "%llu:%llu",
                    info.pbi_start_tvsec, info.pbi_start_tvusec);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func processIdentity(pid int) (string, bool, error) {
	if pid <= 0 {
		return "", false, nil
	}
	buffer := make([]byte, 64)
	written := C.imoogi_process_start(C.int(pid), (*C.char)(unsafe.Pointer(&buffer[0])), C.int(len(buffer)))
	if written <= 0 {
		return "", false, nil
	}
	if int(written) >= len(buffer) {
		return "", false, fmt.Errorf("process identity exceeds buffer")
	}
	return C.GoString((*C.char)(unsafe.Pointer(&buffer[0]))), true, nil
}
