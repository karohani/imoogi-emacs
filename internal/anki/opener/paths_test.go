package opener

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// ExecRunner is the only seam that touches real processes. It is exercised
// with harmless system binaries; the sleeping child is bounded by its context
// (exec.CommandContext kills it), so no process outlives the test.
func TestExecRunnerReportsExitStatusOutputAndTimeout(t *testing.T) {
	for _, p := range []string{"/usr/bin/true", "/usr/bin/false", "/bin/echo", "/bin/sleep"} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("%s missing on this host", p)
		}
	}
	ok := ExecRunner(context.Background(), "/usr/bin/true")
	if ok.Err != nil || ok.ExitCode != 0 || ok.TimedOut {
		t.Errorf("true: %+v, want a clean exit", ok)
	}

	failed := ExecRunner(context.Background(), "/usr/bin/false")
	if failed.Err != nil || failed.ExitCode == 0 {
		t.Errorf("false: %+v, want a non-zero exit code and no Err", failed)
	}

	echoed := ExecRunner(context.Background(), "/bin/echo", `"ok"`)
	if got := strings.TrimSpace(string(echoed.Stdout)); got != `"ok"` {
		t.Errorf("echo stdout = %q, want %q", got, `"ok"`)
	}

	missing := ExecRunner(context.Background(), filepath.Join(t.TempDir(), "no-such-binary"))
	if missing.Err == nil {
		t.Errorf("missing binary: %+v, want Err", missing)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	slow := ExecRunner(ctx, "/bin/sleep", "5")
	if !slow.TimedOut {
		t.Errorf("sleep past its limit: %+v, want TimedOut", slow)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("timed-out call took %v, want it killed near its limit", elapsed)
	}
}

func TestCappedBufferKeepsTheLimitAndReportsFullWrites(t *testing.T) {
	b := &cappedBuffer{limit: 4}
	if n, err := b.Write([]byte("abcdef")); n != 6 || err != nil {
		t.Errorf("Write = %d, %v; want 6, nil", n, err)
	}
	if n, err := b.Write([]byte("gh")); n != 2 || err != nil {
		t.Errorf("second Write = %d, %v; want 2, nil", n, err)
	}
	if b.String() != "abcd" {
		t.Errorf("kept %q, want %q", b.String(), "abcd")
	}
}

// REQ-ASRC-010: a failed launch is not silent either.
func TestOpenNotifiesWhenEmacsCannotBeLaunched(t *testing.T) {
	for name, launch := range map[string]CmdResult{
		"exit":  {ExitCode: 1, Stderr: []byte("LSOpenURLsWithRole() failed")},
		"error": {Err: errors.New("exec: no such file")},
	} {
		env := newFakeEnv()
		env.probe = func(int) CmdResult { return CmdResult{ExitCode: 1} }
		env.launch = launch
		if code := Open(linkFor("abc"), env.deps()); code == ExitOK {
			t.Errorf("%s: exit 0, want non-zero", name)
		}
		if env.count(isNotify) != 1 || env.count(isEval) != 0 {
			t.Errorf("%s: notify=%d eval=%d, want 1/0", name, env.count(isNotify), env.count(isEval))
		}
	}
}

// A probe that cannot even start emacsclient fails with a notice and does
// not try to launch Emacs.
func TestOpenNotifiesWhenEmacsclientCannotRun(t *testing.T) {
	env := newFakeEnv()
	env.probe = func(int) CmdResult { return CmdResult{Err: errors.New("permission denied")} }
	if code := Open(linkFor("abc"), env.deps()); code == ExitOK {
		t.Fatalf("exit 0, want non-zero")
	}
	if env.count(isLaunch) != 0 || env.count(isNotify) != 1 {
		t.Errorf("launch=%d notify=%d, want 0/1", env.count(isLaunch), env.count(isNotify))
	}
}

// A failed activation after a normal result changes nothing: exit 0, no
// notification, a line on stderr only.
func TestOpenIgnoresAFailedActivation(t *testing.T) {
	env := newFakeEnv()
	env.launch = CmdResult{ExitCode: 1, Stderr: []byte("cannot activate")}
	deps := env.deps()
	stderr := &bytes.Buffer{}
	deps.Stderr = stderr
	if code := Open(linkFor("abc"), deps); code != ExitOK {
		t.Fatalf("exit %d, want 0", code)
	}
	if env.count(isNotify) != 0 {
		t.Errorf("notified about a failed activation")
	}
	if !strings.Contains(stderr.String(), "could not activate") {
		t.Errorf("stderr = %q, want the activation failure", stderr.String())
	}
}

func TestLaunchArgsFallBackToTheEmacsApp(t *testing.T) {
	if got := launchArgs("/opt/homebrew/bin/emacsclient"); len(got) != 2 || got[1] != "Emacs" {
		t.Errorf("launchArgs = %q, want [-a Emacs]", got)
	}
}

func TestClassifyFallbacks(t *testing.T) {
	cases := map[string]struct {
		result CmdResult
		want   string
	}{
		"run error":   {CmdResult{Err: errors.New("broken pipe")}, "broken pipe"},
		"unparseable": {CmdResult{Stdout: []byte("nil\n")}, "nil"},
		"silent":      {CmdResult{}, "출력 없음"},
		"stdout err":  {CmdResult{Stdout: []byte("*ERROR*: wrong-type-argument\n")}, "wrong-type-argument"},
	}
	for name, tc := range cases {
		got := classify(tc.result)
		if got == "" || !strings.Contains(got, tc.want) {
			t.Errorf("%s: classify = %q, want a failure mentioning %q", name, got, tc.want)
		}
	}
}

func TestNotificationIsOneBoundedLine(t *testing.T) {
	long := strings.Repeat("가", 300) + "\nsecond line"
	env := newFakeEnv()
	env.deps().notify(long)
	if len(env.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(env.calls))
	}
	script := env.calls[0].args[1]
	if strings.Contains(script, "\n") {
		t.Errorf("notification script spans lines: %q", script)
	}
	if !strings.Contains(script, "…") || utf8.RuneCountInString(script) > 300 {
		t.Errorf("message not truncated: %d runes", utf8.RuneCountInString(script))
	}
}

func TestDefaultDepsCarryThePlanLimits(t *testing.T) {
	d := DefaultDeps(&bytes.Buffer{})
	if d.ProbeTimeout != 2*time.Second || d.EvalTimeout != 10*time.Second ||
		d.LaunchWait != 30*time.Second || d.PollInterval != 500*time.Millisecond {
		t.Errorf("limits = probe %v eval %v wait %v poll %v, want 2s/10s/30s/0.5s (plan DD-6)",
			d.ProbeTimeout, d.EvalTimeout, d.LaunchWait, d.PollInterval)
	}
	if d.FindEmacsclient == nil || d.Run == nil || d.Now == nil || d.Sleep == nil {
		t.Errorf("DefaultDeps left a seam nil")
	}
}

func TestInstallRejectsARelativeBinaryAndReportsToolFailures(t *testing.T) {
	ok := func(context.Context, string, ...string) CmdResult { return CmdResult{} }
	if _, err := Install(InstallOptions{Dest: t.TempDir(), BinaryPath: "imoogi-anki", Run: ok}); err == nil {
		t.Errorf("a relative binary path was accepted")
	}
	for name, result := range map[string]CmdResult{
		"exit":    {ExitCode: 1, Stderr: []byte("syntax error")},
		"error":   {Err: errors.New("not found")},
		"timeout": {TimedOut: true},
	} {
		run := func(_ context.Context, cmd string, _ ...string) CmdResult {
			if cmd == osacompileCommand {
				return result
			}
			return CmdResult{}
		}
		if _, err := Install(InstallOptions{Dest: t.TempDir(), BinaryPath: "/bin/imoogi-anki", Run: run}); err == nil {
			t.Errorf("%s: osacompile failure was not reported", name)
		}
	}
}

// Re-installing over an app Install made itself replaces it.
func TestInstallReplacesItsOwnApp(t *testing.T) {
	dest := t.TempDir()
	run := func(context.Context, string, ...string) CmdResult { return CmdResult{} }
	first, err := Install(InstallOptions{Dest: dest, BinaryPath: "/bin/imoogi-anki", Run: run})
	if err != nil {
		t.Fatalf("first Install: %v", err)
	}
	if _, err := Install(InstallOptions{Dest: dest, BinaryPath: "/bin/imoogi-anki", Run: run}); err != nil {
		t.Fatalf("second Install over its own app: %v", err)
	}
	if _, err := os.Stat(filepath.Join(first, "Contents", "Info.plist")); err != nil {
		t.Errorf("Info.plist missing after reinstall: %v", err)
	}
}

func TestBinaryPathHelpers(t *testing.T) {
	if _, err := os.UserHomeDir(); err != nil {
		t.Skipf("no home directory on this host: %v", err)
	}
	exe, err := CurrentBinary()
	if err != nil || !filepath.IsAbs(exe) {
		t.Errorf("CurrentBinary = %q, %v; want an absolute path", exe, err)
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	if err := os.WriteFile(target, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	got, err := resolveSymlinks(link)
	want, _ := filepath.EvalSymlinks(target)
	if err != nil || got != want {
		t.Errorf("resolveSymlinks = %q, %v; want %q", got, err, want)
	}
	home, err := DefaultInstallDest()
	if err != nil || filepath.Base(home) != "Applications" {
		t.Errorf("DefaultInstallDest = %q, %v", home, err)
	}
}
