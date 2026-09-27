package agentipc

import "time"

// Emacsclient delivers events through the emacsclient program.
type Emacsclient struct {
	Getenv func(string) string
}

// Deliver is not implemented yet (SPEC-AGENTIPC-001 M4).
func (e Emacsclient) Deliver(event []byte, timeout time.Duration) Result {
	return Result{Code: ExitNotSent, Diag: "delivery is not implemented yet"}
}
