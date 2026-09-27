// Package agentipc implements the imoogi-agent CLI: it turns one command-line
// invocation into one protocol version 1 event (SPEC-AGENTIPC-001 § 2) and
// hands it to the running Emacs through emacsclient.
package agentipc

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Exit codes (spec.md § 2.4).
const (
	ExitOK        = 0 // the receiver returned "ok"
	ExitNotSent   = 1 // delivery not confirmed: no emacsclient, no server, timeout, event file failure
	ExitUsage     = 2 // invalid invocation, rejected before sending
	ExitRejected  = 3 // Emacs was reached but refused the event
	programPrefix = "imoogi-agent: "
)

// Result is the outcome of one delivery attempt: an exit code and, for any
// non-zero code, a one-line diagnostic without the program prefix.
type Result struct {
	Code int
	Diag string
}

// Transport delivers one encoded event within the given time limit.
type Transport interface {
	Deliver(event []byte, timeout time.Duration) Result
}

// Deps carries everything Main reads from the process environment, so tests
// can substitute each piece.
type Deps struct {
	Version   string
	Getenv    func(string) string
	Getwd     func() (string, error)
	Now       func() time.Time
	Transport Transport
}

// Main runs one imoogi-agent invocation and returns its exit code. On success
// it writes nothing; on failure it writes exactly one diagnostic line to
// stderr. Writes to stdout/stderr discard their errors: they are the only
// channels left to report on.
//
// @MX:ANCHOR: [AUTO] Main is the single entry point shared by cmd/imoogi-agent run and all CLI tests.
// @MX:REASON: The exit-code contract of spec.md § 2.4 is decided here; every caller depends on it.
func Main(args []string, stdout, stderr io.Writer, deps Deps) int {
	if len(args) == 1 && args[0] == "--version" {
		_, _ = fmt.Fprintf(stdout, "imoogi-agent %s\n", deps.Version)
		return ExitOK
	}
	req, err := parseRequest(args, deps.Getenv, deps.Getwd, deps.Now())
	if err != nil {
		var usage *usageError
		if errors.As(err, &usage) {
			return fail(stderr, Result{Code: ExitUsage, Diag: usage.msg})
		}
		return fail(stderr, Result{Code: ExitNotSent, Diag: err.Error()})
	}
	result := deps.Transport.Deliver(req.event, req.timeout)
	if result.Code == ExitOK {
		return ExitOK
	}
	return fail(stderr, result)
}

func fail(stderr io.Writer, result Result) int {
	_, _ = fmt.Fprintf(stderr, "%s%s\n", programPrefix, oneLine(result.Diag))
	return result.Code
}

// oneLine folds any line breaks so a diagnostic is always a single line.
func oneLine(s string) string {
	s = strings.TrimSpace(s)
	return strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s)
}
