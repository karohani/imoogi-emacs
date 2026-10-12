package opener

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

const fakeClient = "/Applications/Emacs.app/Contents/MacOS/bin/emacsclient"

// call is one command the handler asked the fake runner to execute.
type call struct {
	name     string
	args     []string
	deadline time.Duration // the per-call limit the handler attached
}

// fakeEnv records every command and answers from a per-kind script. The
// clock is fake: Sleep advances it, so the launch wait is exercised without
// real waiting.
type fakeEnv struct {
	calls  []call
	now    time.Time
	probe  func(n int) CmdResult // n counts probes from 1
	eval   func() CmdResult
	launch CmdResult
	probes int
}

func newFakeEnv() *fakeEnv {
	return &fakeEnv{
		now:    time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC),
		probe:  func(int) CmdResult { return CmdResult{} },
		eval:   func() CmdResult { return CmdResult{Stdout: []byte("\"ok\"\n")} },
		launch: CmdResult{},
	}
}

func (f *fakeEnv) deps() Deps {
	return Deps{
		FindEmacsclient: func() (string, error) { return fakeClient, nil },
		Run: func(ctx context.Context, name string, args ...string) CmdResult {
			c := call{name: name, args: append([]string(nil), args...)}
			if d, ok := ctx.Deadline(); ok {
				c.deadline = time.Until(d).Round(100 * time.Millisecond)
			}
			f.calls = append(f.calls, c)
			switch {
			case name == fakeClient && len(args) == 2 && args[1] == probeExpr:
				f.probes++
				return f.probe(f.probes)
			case name == fakeClient:
				return f.eval()
			case name == openCommand:
				return f.launch
			default:
				return CmdResult{}
			}
		},
		Now:           func() time.Time { return f.now },
		Sleep:         func(d time.Duration) { f.now = f.now.Add(d) },
		ProbeTimeout:  2 * time.Second,
		EvalTimeout:   10 * time.Second,
		LaunchTimeout: 10 * time.Second,
		LaunchWait:    3 * time.Second,
		PollInterval:  500 * time.Millisecond,
		Stderr:        &bytes.Buffer{},
	}
}

func (f *fakeEnv) count(pred func(call) bool) int {
	n := 0
	for _, c := range f.calls {
		if pred(c) {
			n++
		}
	}
	return n
}

func isClient(c call) bool { return c.name == fakeClient }
func isNotify(c call) bool { return c.name == osascriptCommand }
func isLaunch(c call) bool { return c.name == openCommand && len(c.args) > 0 && c.args[0] == "-a" }
func isEval(c call) bool   { return isClient(c) && len(c.args) == 2 && c.args[1] != probeExpr }

func linkFor(id string) string {
	return "imoogi-anki://open?id=" + url.QueryEscape(id)
}

// AC-ASRC-014: every malformed or hostile link is refused before Emacs is
// contacted, with a visible notification and a non-zero exit.
func TestOpenRejectsBadLinksBeforeCallingEmacs(t *testing.T) {
	bad := []string{
		`imoogi-anki://open?id=x%22)(shell-command%20%22touch%20pwn%22`,
		"imoogi-anki://open?id=a%20b",
		"imoogi-anki://open?id=",
		"imoogi-anki://open?id=" + strings.Repeat("a", 129),
		"imoogi-anki://open?id=a&id=b",
		"other://open?id=abc",
		"imoogi-anki://close?id=abc",
		"imoogi-anki://open",
		"imoogi-anki://open?id=abc&x=1",
		"imoogi-anki:open?id=abc",
	}
	for _, raw := range bad {
		env := newFakeEnv()
		code := Open(raw, env.deps())
		if code == ExitOK {
			t.Errorf("%s: exit 0, want non-zero", raw)
		}
		if n := env.count(isClient); n != 0 {
			t.Errorf("%s: emacsclient called %d times, want 0", raw, n)
		}
		if n := env.count(isNotify); n != 1 {
			t.Errorf("%s: notifications = %d, want 1", raw, n)
		}
	}
}

// AC-ASRC-014, the accepting half: the four ID shapes Org produces arrive
// percent-encoded, are decoded, and reach Emacs as exactly one fixed call.
func TestOpenPassesDecodedIDToTheFixedReceiver(t *testing.T) {
	for _, id := range []string{
		"6F1C8E0A-3B2D-4C5E-9F10-123456789ABC",
		"imoogi:20261012T101500-abc",
		"20261012T101500.123456",
		"20261012T101500.123456@host.example",
	} {
		env := newFakeEnv()
		if code := Open(linkFor(id), env.deps()); code != ExitOK {
			t.Fatalf("%s: exit %d, want 0; stderr calls %+v", id, code, env.calls)
		}
		var evals []call
		for _, c := range env.calls {
			if isEval(c) {
				evals = append(evals, c)
			}
		}
		if len(evals) != 1 {
			t.Fatalf("%s: evals = %d, want 1", id, len(evals))
		}
		want := `(imoogi-anki-open-id "` + id + `")`
		if evals[0].args[0] != "--eval" || evals[0].args[1] != want {
			t.Errorf("%s: eval args = %q, want [--eval %q]", id, evals[0].args, want)
		}
		if env.count(isNotify) != 0 {
			t.Errorf("%s: a successful open notified", id)
		}
	}
}

// AC-ASRC-015 (a): Emacs never answers — one launch, probes until the
// bound, one notification, non-zero.
func TestOpenLaunchesEmacsAndGivesUpAtTheBound(t *testing.T) {
	env := newFakeEnv()
	env.probe = func(int) CmdResult { return CmdResult{ExitCode: 1, Stderr: []byte("can't find socket")} }
	code := Open(linkFor("abc"), env.deps())
	if code == ExitOK {
		t.Fatalf("exit 0, want non-zero")
	}
	if n := env.count(isLaunch); n != 1 {
		t.Errorf("launches = %d, want 1", n)
	}
	if env.probes < 6 {
		t.Errorf("probes = %d, want polling up to the 3s bound at 0.5s (>= 6)", env.probes)
	}
	if n := env.count(isNotify); n != 1 {
		t.Errorf("notifications = %d, want 1", n)
	}
	if env.count(isEval) != 0 {
		t.Errorf("evaluated although Emacs never answered")
	}
	for _, c := range env.calls {
		if isLaunch(c) && (len(c.args) != 2 || c.args[1] != "/Applications/Emacs.app") {
			t.Errorf("launch args = %q, want the bundle emacsclient was found in", c.args)
		}
	}
}

// AC-ASRC-015 (b): the second probe after the launch succeeds.
func TestOpenLaunchesEmacsThenEvaluates(t *testing.T) {
	env := newFakeEnv()
	env.probe = func(n int) CmdResult {
		if n < 3 {
			return CmdResult{ExitCode: 1}
		}
		return CmdResult{}
	}
	if code := Open(linkFor("abc"), env.deps()); code != ExitOK {
		t.Fatalf("exit %d, want 0", code)
	}
	if env.launchesBeforeEval() != 1 || env.count(isEval) != 1 || env.count(isNotify) != 0 {
		t.Errorf("launch=%d eval=%d notify=%d, want 1/1/0",
			env.launchesBeforeEval(), env.count(isEval), env.count(isNotify))
	}
	// After a normal result the same bundle is activated once more.
	last := env.calls[len(env.calls)-1]
	if !isLaunch(last) || last.args[1] != "/Applications/Emacs.app" {
		t.Errorf("last call = %+v, want the bundle activation", last)
	}
}

// launchesBeforeEval counts `open -a` calls issued before the receiver eval;
// the one after it is the activation, not a launch.
func (f *fakeEnv) launchesBeforeEval() int {
	n := 0
	for _, c := range f.calls {
		if isEval(c) {
			break
		}
		if isLaunch(c) {
			n++
		}
	}
	return n
}

// AC-ASRC-015 (c): no emacsclient anywhere — nothing is run but the notice.
func TestOpenWithoutEmacsclientOnlyNotifies(t *testing.T) {
	env := newFakeEnv()
	deps := env.deps()
	deps.FindEmacsclient = func() (string, error) { return "", errors.New("emacsclient not found") }
	if code := Open(linkFor("abc"), deps); code == ExitOK {
		t.Fatalf("exit 0, want non-zero")
	}
	if env.count(isClient) != 0 || env.count(isLaunch) != 0 || env.count(isNotify) != 1 {
		t.Errorf("client=%d launch=%d notify=%d, want 0/0/1",
			env.count(isClient), env.count(isLaunch), env.count(isNotify))
	}
}

// AC-ASRC-015 (d) and (f): only "ok", "not-found" and "duplicate" are
// normal; anything else notifies with the error text and exits non-zero.
func TestOpenClassifiesTheReceiverResult(t *testing.T) {
	cases := []struct {
		name   string
		result CmdResult
		ok     bool
		detail string
	}{
		{"not-found", CmdResult{Stdout: []byte("\"not-found\"\n")}, true, ""},
		{"duplicate", CmdResult{Stdout: []byte("\"duplicate\"\n")}, true, ""},
		{"nonzero", CmdResult{ExitCode: 1, Stderr: []byte("server went away")}, false, "server went away"},
		{"void-function", CmdResult{ExitCode: 1,
			Stderr: []byte("*ERROR*: Symbol’s function definition is void: imoogi-anki-open-id\n")}, false, "imoogi-anki-open-id"},
		{"weird", CmdResult{Stdout: []byte("\"weird\"\n")}, false, "weird"},
	}
	for _, tc := range cases {
		env := newFakeEnv()
		env.eval = func() CmdResult { return tc.result }
		code := Open(linkFor("abc"), env.deps())
		if (code == ExitOK) != tc.ok {
			t.Errorf("%s: exit %d, want ok=%v", tc.name, code, tc.ok)
		}
		notes := env.count(isNotify)
		if tc.ok && notes != 0 {
			t.Errorf("%s: notified on a normal result", tc.name)
		}
		if !tc.ok {
			if notes != 1 {
				t.Errorf("%s: notifications = %d, want 1", tc.name, notes)
			}
			for _, c := range env.calls {
				if isNotify(c) && !strings.Contains(strings.Join(c.args, " "), tc.detail) {
					t.Errorf("%s: notification %q lacks %q", tc.name, c.args, tc.detail)
				}
			}
		}
	}
}

// AC-ASRC-015 (e): a probe or an eval that outlives its own limit is
// abandoned, notified, and fails. The fake blocks until the handler's own
// per-call deadline fires, so the limit is really enforced by Open.
func TestOpenEnforcesPerCallTimeouts(t *testing.T) {
	for _, stage := range []string{"probe", "eval"} {
		env := newFakeEnv()
		block := func() CmdResult { return CmdResult{} }
		deps := env.deps()
		deps.ProbeTimeout = 30 * time.Millisecond
		deps.EvalTimeout = 30 * time.Millisecond
		inner := deps.Run
		deps.Run = func(ctx context.Context, name string, args ...string) CmdResult {
			isProbe := name == fakeClient && len(args) == 2 && args[1] == probeExpr
			if (stage == "probe" && isProbe) || (stage == "eval" && name == fakeClient && !isProbe) {
				env.calls = append(env.calls, call{name: name, args: args})
				<-ctx.Done()
				return block()
			}
			return inner(ctx, name, args...)
		}
		start := time.Now()
		code := Open(linkFor("abc"), deps)
		if code == ExitOK {
			t.Errorf("%s: exit 0, want non-zero", stage)
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("%s: took %v, want bounded by the per-call limit", stage, elapsed)
		}
		if env.count(isNotify) != 1 {
			t.Errorf("%s: notifications = %d, want 1", stage, env.count(isNotify))
		}
		if env.count(isLaunch) != 0 {
			t.Errorf("%s: a timed-out call launched Emacs", stage)
		}
	}
}

// REQ-ASRC-010.3: every emacsclient call carries its own limit.
func TestOpenAttachesPerCallLimits(t *testing.T) {
	env := newFakeEnv()
	if code := Open(linkFor("abc"), env.deps()); code != ExitOK {
		t.Fatalf("exit %d, want 0", code)
	}
	for _, c := range env.calls {
		if !isClient(c) {
			continue
		}
		want := 2 * time.Second
		if isEval(c) {
			want = 10 * time.Second
		}
		if c.deadline != want {
			t.Errorf("%q deadline = %v, want %v", c.args, c.deadline, want)
		}
	}
}

func TestLispStringQuotesBackslashAndQuote(t *testing.T) {
	if got := lispString(`a"b\c`); got != `"a\"b\\c"` {
		t.Errorf("lispString = %s", got)
	}
}

func TestBundleForEmacsclient(t *testing.T) {
	cases := map[string]string{
		"/Applications/Emacs.app/Contents/MacOS/bin/emacsclient":      "/Applications/Emacs.app",
		"/Applications/Emacs-31.1.app/Contents/MacOS/bin/emacsclient": "/Applications/Emacs-31.1.app",
		"/opt/homebrew/bin/emacsclient":                               "",
	}
	for client, want := range cases {
		if got := bundleFor(client); got != want {
			t.Errorf("bundleFor(%s) = %q, want %q", client, got, want)
		}
	}
}
