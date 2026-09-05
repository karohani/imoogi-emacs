// Command imoogi is the back end of the imoogi Org-to-Anki sync. It reads one
// JSON request document from stdin and writes one JSON response document to
// stdout, once per sync run. It performs no user interaction.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/karohani/imoogi-emacs/internal/anki/ankiconnect"
	"github.com/karohani/imoogi-emacs/internal/anki/planner"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

const version = "imoogi version 0.1.0-dev"

const usage = `usage: imoogi <command>

commands:
  sync        read a JSON request document on stdin, write a JSON response on stdout
  --version   print the version and exit
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is main's testable body. Exit 0 means the response document on stdout is
// valid and should be read; non-zero means the front end falls back to its own
// error taxonomy.
//
// Every write to stdout and stderr in this file discards its error explicitly
// (`_, _ =`). This is a CLI process whose only two output streams ARE these
// writers: if a write to stderr fails, there is no second channel left to
// report that failure on, and no recovery available beyond the exit code the
// surrounding branch already returns. Discarding is stated rather than
// implied so the omission reads as a decision, not an oversight.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}

	switch args[0] {
	case "--version", "-version", "version":
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	case "sync":
		return runSync(stdin, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "imoogi: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func runSync(stdin io.Reader, stdout, stderr io.Writer) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "imoogi: request could not be read: %v\n", err)
		return 1
	}

	// The version is probed before the body is decoded, because the case this
	// field exists for is a request whose shape this binary does not know: a
	// full decode would fail on the shape and never reach the comparison.
	var probe struct {
		ProtocolVersion int `json:"protocol_version"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		_, _ = fmt.Fprintf(stderr, "imoogi: request could not be decoded: %v\n", err)
		return 1
	}

	if probe.ProtocolVersion != protocol.Version {
		resp := newResponse(false)
		resp.Errors = append(resp.Errors, protocol.Error{
			Code: protocol.CodeBinaryIncompatible,
			Message: fmt.Sprintf(
				"request protocol_version %d, binary speaks %d",
				probe.ProtocolVersion, protocol.Version),
			Key: nil,
		})
		if err := writeResponse(stdout, resp); err != nil {
			_, _ = fmt.Fprintf(stderr, "imoogi: response could not be written: %v\n", err)
		}
		return 1
	}

	var req protocol.Request
	if err := json.Unmarshal(raw, &req); err != nil {
		_, _ = fmt.Fprintf(stderr, "imoogi: request could not be decoded: %v\n", err)
		return 1
	}

	ctx := context.Background()

	// design.md §3 step 7's protocol check is above; the handshake is the
	// FIRST AnkiConnect interaction of the run, before any entry is
	// processed (spec.md REQ-018's two distinct codes). requestPermission
	// is what distinguishes "nothing is listening" (anki_unreachable) from
	// "something answered, but not AnkiConnect" (ankiconnect_missing) —
	// ankiconnect.Client.Handshake already produces the right typed error
	// for each case (plan.md D-5).
	client := ankiconnect.NewClient(req.Config.AnkiConnectURL, nil)
	if err := client.Handshake(ctx); err != nil {
		return failRun(stdout, stderr, handshakeErrorCode(err), err.Error())
	}

	reg, err := registry.Load(req.Config.RegistryPath)
	if err != nil {
		return failRun(stdout, stderr, protocol.CodeStateUnreadable, err.Error())
	}

	results, errs := planner.Run(ctx, req, reg, client)

	if err := reg.Save(); err != nil {
		// The registry write is best-effort diagnosed on stderr only: no
		// plan.md D-5 code is reserved for a SAVE failure specifically (only
		// state_unreadable, which names a READ failure), and inventing one
		// here would leave it absent from the front end's own code table —
		// exactly the defect D-5's own rationale warns against. The results
		// already computed this run are still reported.
		_, _ = fmt.Fprintf(stderr, "imoogi: registry could not be saved: %v\n", err)
	}

	resp := newResponse(true)
	resp.Results = append(resp.Results, results...)
	resp.Errors = append(resp.Errors, errs...)
	if err := writeResponse(stdout, resp); err != nil {
		_, _ = fmt.Fprintf(stderr, "imoogi: response could not be written: %v\n", err)
		return 1
	}
	return 0
}

// failRun writes a run-level failure response (ok: false, one run-scoped
// error, no results — nothing could be processed) and returns the non-zero
// exit the front end's own error taxonomy expects (plan.md D-2).
func failRun(stdout, stderr io.Writer, code, message string) int {
	resp := newResponse(false)
	resp.Errors = append(resp.Errors, protocol.Error{Code: code, Message: message, Key: nil})
	if err := writeResponse(stdout, resp); err != nil {
		_, _ = fmt.Fprintf(stderr, "imoogi: response could not be written: %v\n", err)
	}
	return 1
}

// handshakeErrorCode maps ankiconnect.Client.Handshake's typed errors to
// plan.md D-5's two distinct codes: a *TransportError means nothing is
// listening at all (anki_unreachable); anything else that reached the
// handshake and still failed (a *ProtocolError or a *PermissionDeniedError)
// means something answered but is not usable AnkiConnect
// (ankiconnect_missing).
func handshakeErrorCode(err error) string {
	var transportErr *ankiconnect.TransportError
	if errors.As(err, &transportErr) {
		return protocol.CodeAnkiUnreachable
	}
	return protocol.CodeAnkiConnectMissing
}

// newResponse builds a response whose collections are empty rather than nil, so
// they encode as [] rather than null.
func newResponse(ok bool) protocol.Response {
	return protocol.Response{
		ProtocolVersion: protocol.Version,
		OK:              ok,
		Results:         []protocol.Result{},
		Errors:          []protocol.Error{},
	}
}

func writeResponse(stdout io.Writer, resp protocol.Response) error {
	encoded, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, string(encoded))
	return err
}
