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
	"github.com/karohani/imoogi-emacs/internal/anki/model"
	"github.com/karohani/imoogi-emacs/internal/anki/planner"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
	"github.com/karohani/imoogi-emacs/internal/anki/registry"
)

const version = "imoogi version 0.1.0-dev"

// install-models is deliberately listed here but is not a documented user
// entry point (design.md §6). Installation rides imoogi-anki-setup, so a user
// following the existing setup instructions ends up with the models
// installed; the Go binary still needs a verb to dispatch on, and it is named
// for what it does.
const usage = `usage: imoogi <command>

commands:
  sync             read a JSON request document on stdin, write a JSON response on stdout
  install-models   read an install request document on stdin, install imoogi's note types
  migrate          re-home stock-note-type entries onto imoogi's own note types
    --dry-run      report the candidates and their count; write nothing
  --version        print the version and exit
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
	case "install-models":
		return runInstall(stdin, stdout, stderr)
	case "migrate":
		return runMigrate(args[1:], stdin, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "imoogi: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// runSync is the `sync` subcommand: the full per-entry decision pass plus
// census reconciliation and orphan deletion.
func runSync(stdin io.Reader, stdout, stderr io.Writer) int {
	return runOverSyncRequest(stdin, stdout, stderr, true, planner.Run)
}

// runMigrate is the `migrate` subcommand (design.md §7.1). It reads the SAME
// request document `sync` reads and writes the SAME response schema — the
// only thing that distinguishes the two on the wire is which verb was
// invoked, which is REQ-C-018's wire-stability clause.
//
// `--dry-run` is the ONLY flag it accepts, and anything else is a usage
// error rather than a silently ignored argument: a mistyped flag that fell
// through to the writing path would migrate a collection the user meant to
// inspect, and that is not a mistake this command can afford to absorb.
func runMigrate(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	dryRun := false
	for _, arg := range args {
		if arg != "--dry-run" {
			_, _ = fmt.Fprintf(stderr, "imoogi: unknown migrate flag %q\n\n%s", arg, usage)
			return 2
		}
		dryRun = true
	}

	// The registry is persisted only on the writing run. A dry run must
	// leave it byte-unchanged (AC-C-018c) — it decided nothing, so it has
	// nothing to record, and rewriting the same content would still churn
	// the file's mtime for a command that promised to write nothing.
	return runOverSyncRequest(stdin, stdout, stderr, !dryRun,
		func(ctx context.Context, req protocol.Request, reg *registry.Registry, client ankiconnect.AnkiConnector) ([]protocol.Result, []protocol.Error) {
			return planner.Migrate(ctx, req, reg, client, dryRun)
		})
}

// planPass is the decision layer one subcommand runs over a sync request:
// planner.Run for `sync`, planner.Migrate for `migrate`. Both take the same
// inputs and return the same pair, which is what lets the two subcommands
// share the process boundary below instead of forking it.
type planPass func(context.Context, protocol.Request, *registry.Registry, ankiconnect.AnkiConnector) ([]protocol.Result, []protocol.Error)

// runOverSyncRequest is the process boundary both sync-request subcommands
// share: read stdin, probe the protocol version, decode, handshake, load the
// registry, run the pass, optionally persist, write one response.
//
// It is one function rather than two near-identical ones because every step
// before and after the pass is a contract the two subcommands must agree on
// exactly — the same two handshake codes, the same state_unreadable path,
// the same best-effort save diagnosis, the same response shape. Two copies
// are how those silently drift apart.
func runOverSyncRequest(stdin io.Reader, stdout, stderr io.Writer, persist bool, pass planPass) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "imoogi: request could not be read: %v\n", err)
		return 1
	}

	if code, ok := probeProtocolVersion(raw, stdout, stderr); !ok {
		return code
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

	results, errs := pass(ctx, req, reg, client)

	if persist {
		if err := reg.Save(); err != nil {
			// The registry write is best-effort diagnosed on stderr only: no
			// plan.md D-5 code is reserved for a SAVE failure specifically (only
			// state_unreadable, which names a READ failure), and inventing one
			// here would leave it absent from the front end's own code table —
			// exactly the defect D-5's own rationale warns against. The results
			// already computed this run are still reported.
			_, _ = fmt.Fprintf(stderr, "imoogi: registry could not be saved: %v\n", err)
		}
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

// probeProtocolVersion reads only the protocol_version field of a request
// document and compares it against the version this binary speaks. It returns
// the exit code to use and whether the caller may proceed.
//
// The version is probed before the body is decoded, because the case this
// field exists for is a request whose shape this binary does not know: a full
// decode would fail on the shape and never reach the comparison. Both
// subcommands' request documents carry the field at the same place and for
// the same reason, so both probe through here — a second copy of this logic
// is how the two would silently drift apart.
func probeProtocolVersion(raw []byte, stdout, stderr io.Writer) (int, bool) {
	var probe struct {
		ProtocolVersion int `json:"protocol_version"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		_, _ = fmt.Fprintf(stderr, "imoogi: request could not be decoded: %v\n", err)
		return 1, false
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
		return 1, false
	}
	return 0, true
}

// runInstall is the install step's process boundary (design.md §6): read one
// install request document on stdin, install imoogi's two note types, write
// one response document on stdout.
//
// It reads no stylesheet from disk. The user stylesheet arrives as text in
// the request; the front end is its only reader (REQ-C-008).
//
// The ownership announcement (REQ-C-002.3) goes to STDERR, because stdout
// carries the response document and nothing else — the same invariant every
// diagnostic in this file observes.
func runInstall(stdin io.Reader, stdout, stderr io.Writer) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "imoogi: request could not be read: %v\n", err)
		return 1
	}

	if code, ok := probeProtocolVersion(raw, stdout, stderr); !ok {
		return code
	}

	var req protocol.InstallRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		_, _ = fmt.Fprintf(stderr, "imoogi: request could not be decoded: %v\n", err)
		return 1
	}

	ctx := context.Background()

	// The same handshake runSync performs, for the same reason and with the
	// same two codes: an unreachable endpoint and a host that answers without
	// being AnkiConnect are distinguishable only here, before any other
	// request is attempted. The install step is not where a second
	// unreachability taxonomy gets invented.
	client := ankiconnect.NewClient(req.AnkiConnectURL, nil)
	if err := client.Handshake(ctx); err != nil {
		return failRun(stdout, stderr, handshakeErrorCode(err), err.Error())
	}

	results, errs := model.Install(ctx, client, req.UserCSS, stderr)

	// ok is false only when NOTHING could be installed. One type failing
	// while the other succeeds is a partial success the front end can act on,
	// not a failed run.
	resp := newResponse(len(results) > 0)
	resp.Results = append(resp.Results, results...)
	resp.Errors = append(resp.Errors, errs...)
	if err := writeResponse(stdout, resp); err != nil {
		_, _ = fmt.Fprintf(stderr, "imoogi: response could not be written: %v\n", err)
		return 1
	}
	if !resp.OK {
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
