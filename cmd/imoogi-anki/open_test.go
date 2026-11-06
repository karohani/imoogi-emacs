package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/karohani/imoogi-emacs/internal/anki/opener"
)

// fakeOpenSeams swaps the open side's seams for recording fakes for one test.
func fakeOpenSeams(t *testing.T) *[][]string {
	t.Helper()
	t.Setenv(logPathEnvironment, "")
	var calls [][]string
	record := func(ctx context.Context, name string, args ...string) opener.CmdResult {
		calls = append(calls, append([]string{name}, args...))
		if strings.HasSuffix(name, "emacsclient") && len(args) == 2 && args[1] != "t" {
			return opener.CmdResult{Stdout: []byte("\"ok\"\n")}
		}
		return opener.CmdResult{}
	}
	prevDeps, prevRunner, prevBinary := openDeps, installRunner, currentBinary
	openDeps = func(stderr io.Writer) opener.Deps {
		d := opener.DefaultDeps(stderr)
		d.FindEmacsclient = func() (string, error) { return "/Applications/Emacs.app/Contents/MacOS/bin/emacsclient", nil }
		d.Run = record
		d.Sleep = func(time.Duration) {}
		return d
	}
	installRunner = record
	currentBinary = func() (string, error) { return "/opt/imoogi tools/imoogi-anki", nil }
	t.Cleanup(func() { openDeps, installRunner, currentBinary = prevDeps, prevRunner, prevBinary })
	return &calls
}

func TestOpenSubcommandDispatchesTheLink(t *testing.T) {
	calls := fakeOpenSeams(t)
	var out, errOut bytes.Buffer
	if code := run([]string{"open", "imoogi-anki://open?id=abc%3A1"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %s", code, errOut.String())
	}
	var evaluated bool
	for _, c := range *calls {
		if strings.HasSuffix(c[0], "emacsclient") && c[len(c)-1] == `(imoogi-anki-open-id "abc:1")` {
			evaluated = true
		}
	}
	if !evaluated {
		t.Errorf("receiver not evaluated; calls %q", *calls)
	}
}

func TestOpenSubcommandRejectsMissingOrBadURL(t *testing.T) {
	calls := fakeOpenSeams(t)
	var out, errOut bytes.Buffer
	if code := run([]string{"open"}, strings.NewReader(""), &out, &errOut); code != 2 {
		t.Errorf("open without URL: exit %d, want 2", code)
	}
	if code := run([]string{"open", "imoogi-anki://open?id=a%20b"}, strings.NewReader(""), &out, &errOut); code == 0 {
		t.Errorf("open with a bad id: exit 0")
	}
	for _, c := range *calls {
		if strings.HasSuffix(c[0], "emacsclient") {
			t.Errorf("emacsclient called for a rejected link: %q", c)
		}
	}
}

func TestInstallURLHandlerSubcommandGeneratesWithoutRegistering(t *testing.T) {
	calls := fakeOpenSeams(t)
	dest := filepath.Join(t.TempDir(), "my apps")
	var out, errOut bytes.Buffer
	code := run([]string{"install-url-handler", "--dest", dest, "--no-register"}, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("exit %d, stderr %s", code, errOut.String())
	}
	app := filepath.Join(dest, opener.AppName)
	if strings.TrimSpace(out.String()) != app {
		t.Errorf("stdout = %q, want %q", out.String(), app)
	}
	if _, err := os.Stat(filepath.Join(app, "Contents", "Info.plist")); err != nil {
		t.Errorf("Info.plist missing: %v", err)
	}
	for _, c := range *calls {
		if strings.HasSuffix(c[0], "lsregister") {
			t.Errorf("lsregister ran under --no-register")
		}
	}
	if code := run([]string{"install-url-handler", "--bogus"}, strings.NewReader(""), &out, &errOut); code != 2 {
		t.Errorf("unknown flag: exit %d, want 2", code)
	}
}
