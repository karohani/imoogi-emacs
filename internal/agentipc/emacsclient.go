package agentipc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// evalExpr is the only expression ever sent to Emacs. It pops the event file
// path from server-eval-args-left before anything else runs, so the path is
// never evaluated as Lisp even when the receiver is not loaded
// (REQ-AIPC-012.3). It never embeds the event or its path.
const evalExpr = `(let ((f (pop server-eval-args-left))) (if (fboundp 'imoogi-agent-receive-file) (imoogi-agent-receive-file f) "error:not-loaded"))`

// outputLimit bounds how much emacsclient stdout and stderr is kept.
const outputLimit = 64 << 10

// waitDelay bounds pipe draining after emacsclient is killed, so the CLI
// ends within the time limit plus one second (REQ-AIPC-014.4).
const waitDelay = 500 * time.Millisecond

// appBundleCandidates are the macOS application-bundle locations tried after
// $EMACSCLIENT and PATH, in order. They are copied verbatim from
// scripts/imoogi-editor find_emacsclient (lines 5-37) and must stay in the
// same order there; TestDiscoveryCandidatesMatchImoogiEditor watches for
// drift. ${EMACS_VERSION:-31.1} and the * glob are expanded as bash would.
// Tests replace this list to point at a temporary directory.
var appBundleCandidates = []string{
	"/Applications/Emacs-${EMACS_VERSION:-31.1}.app/Contents/MacOS/bin/emacsclient",
	"/Applications/Emacs.app/Contents/MacOS/bin/emacsclient",
	"/Applications/Emacs-*.app/Contents/MacOS/bin/emacsclient",
}

// Emacsclient delivers events through the emacsclient program found by
// findEmacsclient, reading its configuration from the process environment.
type Emacsclient struct{}

// Deliver writes the event to an owner-only file inside a fresh owner-only
// temporary directory, runs `emacsclient --eval EXPR PATH` without a shell,
// classifies the outcome, and removes the file and the directory on every
// path.
//
// @MX:ANCHOR: [AUTO] Deliver is the transport entry point: discovery, event directory lifetime, time limit, classification.
// @MX:REASON: Every exit code 0/1/3 of spec.md § 2.4 originates here; REQ-AIPC-012..014 and SPEC-AGENTIPC-002 REQ-AIPH-006 all depend on this sequence.
func (Emacsclient) Deliver(event []byte, timeout time.Duration) Result {
	client, err := findEmacsclient()
	if err != nil {
		return Result{Code: ExitNotSent, Diag: err.Error()}
	}
	dir, path, err := writeEventFile(event)
	if dir != "" {
		// Removal failure has no remaining channel to report on; the
		// directory is owner-only and the next run uses a fresh name.
		defer func() { _ = os.RemoveAll(dir) }()
	}
	if err != nil {
		return Result{Code: ExitNotSent, Diag: err.Error()}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// @MX:WARN: [AUTO] On timeout exec.CommandContext kills emacsclient (SIGKILL); Emacs may still process the request later.
	// @MX:REASON: emacsclient --timeout cannot interrupt a slow or prompting evaluation (plan.md D-9); a late server read finds the file gone and logs unreadable (D-10).
	cmd := exec.CommandContext(ctx, client, "--eval", evalExpr, path)
	cmd.WaitDelay = waitDelay
	stdout := &cappedBuffer{limit: outputLimit}
	stderr := &cappedBuffer{limit: outputLimit}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return Result{Code: ExitNotSent, Diag: fmt.Sprintf("emacsclient timed out after %v", timeout)}
	}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return classify(stdout.buf, stderr.buf, 0)
	case errors.As(err, &exitErr):
		return classify(stdout.buf, stderr.buf, exitErr.ExitCode())
	default:
		return Result{Code: ExitNotSent, Diag: fmt.Sprintf("cannot run %s: %v", client, err)}
	}
}

// writeEventFile creates a fresh directory with mode 0700 under the system
// temporary directory and, inside it, the event file with mode 0600
// regardless of umask. It returns the directory whenever one was created,
// even on a later failure, so the caller can remove it with everything in it;
// the file path is absolute.
func writeEventFile(event []byte) (dir, path string, err error) {
	dir, err = os.MkdirTemp("", "imoogi-agent-*")
	if err != nil {
		return "", "", fmt.Errorf("cannot create event directory: %w", err)
	}
	if abs, absErr := filepath.Abs(dir); absErr == nil {
		dir = abs
	} else {
		return dir, "", fmt.Errorf("cannot create event directory: %w", absErr)
	}
	path = filepath.Join(dir, "event.json")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return dir, "", fmt.Errorf("cannot create event file: %w", err)
	}
	err = f.Chmod(0o600)
	if err == nil {
		_, err = f.Write(event)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return dir, path, fmt.Errorf("cannot write event file: %w", err)
	}
	return dir, path, nil
}

// findEmacsclient mirrors scripts/imoogi-editor find_emacsclient:
// $EMACSCLIENT, then PATH, then appBundleCandidates in order. A bare
// $EMACSCLIENT name (no slash) is looked up in PATH only; a name with a
// slash must be executable. Either way there is no fallback.
func findEmacsclient() (string, error) {
	if explicit := os.Getenv("EMACSCLIENT"); explicit != "" {
		if !strings.ContainsRune(explicit, '/') {
			// Never resolve a bare name against the current directory
			// (SPEC-AGENTIPC-002 REQ-AIPH-007): LookPath stops at the
			// first PATH match and reports a relative entry as ErrDot, and
			// the IsAbs test still holds when GODEBUG=execerrdot=0.
			found, err := exec.LookPath(explicit)
			if err != nil || !filepath.IsAbs(found) {
				return "", fmt.Errorf("EMACSCLIENT not found in PATH: %s", explicit)
			}
			return found, nil
		}
		if !isExecutable(explicit) {
			return "", fmt.Errorf("EMACSCLIENT is not executable: %s", explicit)
		}
		return filepath.Abs(explicit)
	}
	// exec.LookPath refuses PATH entries relative to the current directory.
	if found, err := exec.LookPath("emacsclient"); err == nil {
		return found, nil
	}
	version := os.Getenv("EMACS_VERSION")
	if version == "" {
		version = "31.1"
	}
	for _, candidate := range appBundleCandidates {
		candidate = strings.ReplaceAll(candidate, "${EMACS_VERSION:-31.1}", version)
		matches := []string{candidate}
		if strings.Contains(candidate, "*") {
			matches, _ = filepath.Glob(candidate)
		}
		for _, match := range matches {
			if isExecutable(match) {
				return match, nil
			}
		}
	}
	return "", errors.New("emacsclient not found; install Emacs or set EMACSCLIENT")
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0
}

// classify maps an emacsclient run that finished within the time limit to an
// exit code (plan.md § 3.2).
func classify(stdout, stderr []byte, exitCode int) Result {
	status, parsed := parseLispString(stdout)
	switch {
	case exitCode == 0 && parsed && status == "ok":
		return Result{Code: ExitOK}
	case parsed && strings.HasPrefix(status, "error:"):
		return Result{Code: ExitRejected, Diag: "rejected by Emacs: " + strings.TrimPrefix(status, "error:")}
	}
	if _, detail, found := strings.Cut(string(stderr), "*ERROR*:"); found {
		return Result{Code: ExitRejected, Diag: "Emacs error: " + firstLine(detail)}
	}
	if exitCode != 0 {
		diag := fmt.Sprintf("emacsclient failed (exit status %d)", exitCode)
		if line := firstLine(string(stderr)); line != "" {
			diag += ": " + line
		}
		return Result{Code: ExitNotSent, Diag: diag}
	}
	if out := strings.TrimSpace(string(stdout)); out != "" {
		return Result{Code: ExitRejected, Diag: "unrecognized response from Emacs: " + firstLine(out)}
	}
	return Result{Code: ExitRejected, Diag: "unrecognized response from Emacs: no output"}
}

// parseLispString reads stdout as one printed Lisp string: a trailing newline,
// surrounding double quotes, and only \" and \\ unescaped. Any other shape
// (nil, bare text, an unescaped inner quote) is not a status string.
func parseLispString(stdout []byte) (string, bool) {
	s := strings.TrimSuffix(string(stdout), "\n")
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", false
	}
	var out strings.Builder
	body := s[1 : len(s)-1]
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case c == '\\' && i+1 < len(body) && (body[i+1] == '"' || body[i+1] == '\\'):
			out.WriteByte(body[i+1])
			i++
		case c == '"':
			return "", false
		default:
			out.WriteByte(c)
		}
	}
	return out.String(), true
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}

// cappedBuffer keeps at most limit bytes and silently discards the rest. It
// always reports a full write: an error would stop exec's copy goroutine and
// turn a clean emacsclient exit into a spurious failure.
type cappedBuffer struct {
	buf   []byte
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - len(b.buf); room > 0 {
		b.buf = append(b.buf, p[:min(room, len(p))]...)
	}
	return len(p), nil
}
