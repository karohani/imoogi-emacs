// Command imoogi-agent lets a coding agent notify the running Emacs: it builds
// one event from its arguments and delivers it through emacsclient
// (SPEC-AGENTIPC-001). Exit codes: 0 accepted, 1 not delivered, 2 invalid
// invocation, 3 rejected by Emacs.
package main

import (
	"io"
	"os"
	"time"

	"github.com/karohani/imoogi-emacs/internal/agentipc"
)

var version = "dev"

// exit is os.Exit, replaceable so tests can observe main's exit code.
var exit = os.Exit

func main() {
	exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is main's testable body. stdin is unused: events come from arguments.
//
// @MX:ANCHOR: [AUTO] run wires the real process environment into agentipc.Main.
// @MX:REASON: main and every CLI-level test enter through run; its signature follows cmd/imoogi-anki.
func run(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	return agentipc.Main(args, stdout, stderr, agentipc.Deps{
		Version:   version,
		Getenv:    os.Getenv,
		Getwd:     os.Getwd,
		Now:       time.Now,
		Transport: agentipc.Emacsclient{},
	})
}
