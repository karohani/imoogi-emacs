// Package opener is the open side of SPEC-ANKISRC-001: `imoogi-anki open
// <url>` turns a card's "Emacs에서 원본 열기" link into one fixed Emacs call,
// and `imoogi-anki install-url-handler` builds the small macOS app that hands
// imoogi-anki:// links to this binary.
//
// Every external program runs through Deps.Run without a shell, and every
// failure ends in a visible macOS notification plus a non-zero exit: the
// process is launched by Launch Services with no terminal attached, so a
// failure that only printed to stderr would be a silent one (REQ-ASRC-010).
package opener

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/karohani/imoogi-emacs/internal/agentipc"
	"github.com/karohani/imoogi-emacs/internal/anki/protocol"
)

// Exit codes of `imoogi-anki open`.
const (
	ExitOK     = 0 // the receiver returned "ok", "not-found", "duplicate" or "incomplete"
	ExitFailed = 1 // Emacs unreachable, timed out, or the receiver failed
	ExitUsage  = 2 // the link itself was rejected before Emacs was contacted
)

const (
	// Scheme and Host are the only link shape accepted (REQ-ASRC-009).
	Scheme = "imoogi-anki"
	Host   = "open"

	// probeExpr is the liveness probe: the cheapest expression a running
	// server evaluates.
	probeExpr = "t"

	openCommand      = "/usr/bin/open"
	osascriptCommand = "/usr/bin/osascript"

	notificationTitle = "imoogi-anki"
	waitDelay         = 500 * time.Millisecond
	outputLimit       = 64 << 10
)

// normalResults are the receiver's normal outcomes (REQ-ASRC-010.2).
// "not-found", "duplicate" and "incomplete" are normal: their explanation is
// already on the Emacs frame the receiver raised, so the handler adds nothing.
// "incomplete" means a target could not be read, so the receiver listed it
// and moved nowhere (a deviation from the SPEC's three values; see
// progress.md §E.2).
var normalResults = map[string]bool{"ok": true, "not-found": true, "duplicate": true, "incomplete": true}

// CmdResult is what one external command produced.
type CmdResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	// Err is set when the command could not be run at all, or ended in a way
	// other than a normal exit.
	Err error
	// TimedOut is set when the call outlived its per-call limit and was killed.
	TimedOut bool
}

// Runner executes one command without a shell. ctx carries the per-call limit.
type Runner func(ctx context.Context, name string, args ...string) CmdResult

// Deps are the handler's seams. DefaultDeps fills them for real use; tests
// replace the runner, the clock, and the limits.
type Deps struct {
	FindEmacsclient func() (string, error)
	Run             Runner
	Now             func() time.Time
	Sleep           func(time.Duration)

	// ProbeTimeout and EvalTimeout bound each emacsclient call; LaunchTimeout
	// bounds `open -a`. LaunchWait bounds the whole wait for a freshly
	// launched Emacs, probed every PollInterval. The handler's worst case is
	// therefore LaunchWait plus one ProbeTimeout plus EvalTimeout plus the
	// notification (REQ-ASRC-010.3).
	ProbeTimeout  time.Duration
	EvalTimeout   time.Duration
	LaunchTimeout time.Duration
	LaunchWait    time.Duration
	PollInterval  time.Duration

	Stderr io.Writer
}

// DefaultDeps returns the real seams with plan.md DD-6's limits.
func DefaultDeps(stderr io.Writer) Deps {
	return Deps{
		FindEmacsclient: agentipc.FindEmacsclient,
		Run:             ExecRunner,
		Now:             time.Now,
		Sleep:           time.Sleep,
		ProbeTimeout:    2 * time.Second,
		EvalTimeout:     10 * time.Second,
		LaunchTimeout:   10 * time.Second,
		LaunchWait:      30 * time.Second,
		PollInterval:    500 * time.Millisecond,
		Stderr:          stderr,
	}
}

// ParseURL accepts exactly imoogi-anki://open?id=<percent-encoded ID> and
// returns the decoded ID once it passes the shared character set. Anything
// else — another scheme or host, a path, credentials, a second parameter, a
// repeated id — is an error.
func ParseURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("not a URL: %v", err)
	}
	if u.Scheme != Scheme || u.Opaque != "" || u.User != nil || !strings.EqualFold(u.Host, Host) {
		return "", fmt.Errorf("not an %s://%s link", Scheme, Host)
	}
	if u.Path != "" && u.Path != "/" {
		return "", errors.New("unexpected path in the link")
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", fmt.Errorf("malformed query: %v", err)
	}
	ids := query["id"]
	if len(query) != 1 || len(ids) != 1 {
		return "", errors.New("the link must carry exactly one id parameter")
	}
	if !protocol.ValidOrgID(ids[0]) {
		return "", errors.New("the id is outside the Org ID character set")
	}
	return ids[0], nil
}

// lispString quotes s as an Emacs Lisp string literal. The character set
// already excludes both escaped characters; quoting is the second line of
// defence (REQ-ASRC-009.3).
func lispString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// receiverExpr is the only expression evaluated with a link's ID.
func receiverExpr(id string) string {
	return "(imoogi-anki-open-id " + lispString(id) + ")"
}

// bundleFor returns the .app bundle emacsclient was found inside, or "" when
// it is not inside one. Launching and activating that bundle — rather than
// whatever "Emacs" Launch Services prefers — keeps the server and the window
// in one installation when several Emacs builds are present.
func bundleFor(client string) string {
	if i := strings.Index(client, ".app/Contents/"); i >= 0 {
		return client[:i+len(".app")]
	}
	return ""
}

// Open handles one imoogi-anki://open link and returns the exit code.
//
// @MX:ANCHOR: [AUTO] Open is the URL handler's single entry: validation, emacsclient discovery, launch-and-wait, eval, classification, notification.
// @MX:REASON: Every exit code of REQ-ASRC-009/010 and every visible failure notice originates here; the shim app and cmd/imoogi-anki both depend on it.
func Open(rawURL string, d Deps) int {
	id, err := ParseURL(rawURL)
	if err != nil {
		return d.fail(ExitUsage, "링크를 열 수 없습니다: "+err.Error())
	}
	client, err := d.FindEmacsclient()
	if err != nil {
		return d.fail(ExitFailed, "emacsclient 를 찾지 못했습니다: "+err.Error())
	}

	probe := d.call(d.ProbeTimeout, client, "--eval", probeExpr)
	switch {
	case probe.TimedOut:
		return d.fail(ExitFailed, fmt.Sprintf("Emacs 가 %v 안에 응답하지 않았습니다.", d.ProbeTimeout))
	case probe.Err != nil:
		return d.fail(ExitFailed, fmt.Sprintf("emacsclient 를 실행하지 못했습니다: %v", probe.Err))
	case probe.ExitCode != 0:
		if code, ok := d.launchAndWait(client); !ok {
			return code
		}
	}

	result := d.call(d.EvalTimeout, client, "--eval", receiverExpr(id))
	if result.TimedOut {
		return d.fail(ExitFailed, fmt.Sprintf("Emacs 가 %v 안에 원본 열기를 끝내지 못했습니다.", d.EvalTimeout))
	}
	if problem := classify(result); problem != "" {
		return d.fail(ExitFailed, problem)
	}
	// The receiver already raised its frame; bringing the app forward too
	// covers a click that arrives while another app is frontmost. A failure
	// here changes nothing about the outcome, so it is reported on stderr
	// only.
	if activate := d.call(d.LaunchTimeout, openCommand, launchArgs(client)...); activate.Err != nil || activate.ExitCode != 0 {
		_, _ = fmt.Fprintf(d.Stderr, "imoogi-anki: could not activate Emacs: %v %s\n", activate.Err, firstLine(activate.Stderr))
	}
	return ExitOK
}

// launchAndWait starts Emacs and polls emacsclient until it answers or
// LaunchWait elapses. ok is false when the caller must return code.
func (d Deps) launchAndWait(client string) (code int, ok bool) {
	launch := d.call(d.LaunchTimeout, openCommand, launchArgs(client)...)
	if launch.TimedOut || launch.Err != nil || launch.ExitCode != 0 {
		return d.fail(ExitFailed, "Emacs 를 실행하지 못했습니다: "+describe(launch)), false
	}
	deadline := d.Now().Add(d.LaunchWait)
	for {
		d.Sleep(d.PollInterval)
		probe := d.call(d.ProbeTimeout, client, "--eval", probeExpr)
		if !probe.TimedOut && probe.Err == nil && probe.ExitCode == 0 {
			return 0, true
		}
		if !d.Now().Before(deadline) {
			return d.fail(ExitFailed, fmt.Sprintf(
				"Emacs 를 실행했지만 %v 안에 서버가 준비되지 않았습니다. Emacs 가 다 뜬 뒤 링크를 다시 누르세요.",
				d.LaunchWait)), false
		}
	}
}

func launchArgs(client string) []string {
	if bundle := bundleFor(client); bundle != "" {
		return []string{"-a", bundle}
	}
	return []string{"-a", "Emacs"}
}

// classify returns "" for a normal receiver result, otherwise a one-line
// description of what went wrong (REQ-ASRC-010.2). It is this package's own
// classifier: agentipc's accepts only "ok" and error:* and would call
// "not-found" an unrecognized response.
func classify(r CmdResult) string {
	status, parsed := agentipc.ParseLispString(r.Stdout)
	if r.Err == nil && r.ExitCode == 0 && parsed && normalResults[status] {
		return ""
	}
	for _, stream := range [][]byte{r.Stderr, r.Stdout} {
		if _, detail, found := strings.Cut(string(stream), "*ERROR*:"); found {
			return "Emacs 오류: " + firstLine([]byte(detail))
		}
	}
	if r.Err != nil {
		return fmt.Sprintf("emacsclient 를 실행하지 못했습니다: %v", r.Err)
	}
	if r.ExitCode != 0 {
		return fmt.Sprintf("emacsclient 가 실패했습니다(종료 코드 %d): %s", r.ExitCode, firstLine(r.Stderr))
	}
	if parsed {
		return "Emacs 가 예상하지 못한 결과를 돌려주었습니다: " + status
	}
	if out := firstLine(r.Stdout); out != "" {
		return "Emacs 의 응답을 해석할 수 없습니다: " + out
	}
	return "Emacs 의 응답을 해석할 수 없습니다: 출력 없음"
}

// call runs one command under its own time limit.
func (d Deps) call(limit time.Duration, name string, args ...string) CmdResult {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	r := d.Run(ctx, name, args...)
	if ctx.Err() == context.DeadlineExceeded {
		r.TimedOut = true
	}
	return r
}

// fail reports message on stderr and as a macOS notification, then returns
// code. A notification that cannot be shown has no further channel; stderr
// still carries the message.
func (d Deps) fail(code int, message string) int {
	_, _ = fmt.Fprintf(d.Stderr, "imoogi-anki: %s\n", message)
	d.notify(message)
	return code
}

func (d Deps) notify(message string) {
	script := "display notification " + appleScriptString(truncate(oneLine(message), 240)) +
		" with title " + appleScriptString(notificationTitle)
	d.call(5*time.Second, osascriptCommand, "-e", script)
}

// appleScriptString quotes s as an AppleScript string literal.
func appleScriptString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func describe(r CmdResult) string {
	switch {
	case r.TimedOut:
		return "시간 초과"
	case r.Err != nil:
		return r.Err.Error()
	default:
		return fmt.Sprintf("종료 코드 %d %s", r.ExitCode, firstLine(r.Stderr))
	}
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func firstLine(b []byte) string {
	line, _, _ := strings.Cut(strings.TrimSpace(string(b)), "\n")
	return strings.TrimSpace(line)
}

// ExecRunner runs name with args, without a shell, killed when ctx ends.
func ExecRunner(ctx context.Context, name string, args ...string) CmdResult {
	// @MX:WARN: [AUTO] On timeout exec.CommandContext kills the child; for emacsclient, Emacs may still finish the request later.
	// @MX:REASON: A slow receiver (plan.md R-7) can therefore still open the heading after the timeout notice was shown.
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = waitDelay
	stdout := &cappedBuffer{limit: outputLimit}
	stderr := &cappedBuffer{limit: outputLimit}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	result := CmdResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		return result
	}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		result.ExitCode = exitErr.ExitCode()
	default:
		result.Err = err
	}
	return result
}

// cappedBuffer keeps at most limit bytes and reports every write as complete,
// so a chatty child never turns into a spurious failure.
type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.Len(); room > 0 {
		b.Buffer.Write(p[:min(room, len(p))])
	}
	return len(p), nil
}

// executablePath returns this binary's absolute, symlink-resolved path.
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return resolveSymlinks(exe)
}
