package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeMarkerEnv names the file the fake emacsclient creates when it runs.
// When set, this test binary behaves as that fake instead of running tests.
const fakeMarkerEnv = "IMOOGI_AGENT_TEST_FAKE_MARKER"

func TestMain(m *testing.M) {
	if marker := os.Getenv(fakeMarkerEnv); marker != "" {
		os.Exit(fakeEmacsclient(marker))
	}
	os.Exit(m.Run())
}

// fakeEmacsclient records its arguments and the event file contents in the
// marker file, then accepts the event the way emacsclient 30.2 prints it.
func fakeEmacsclient(marker string) int {
	args := os.Args[1:]
	var record strings.Builder
	record.WriteString(strings.Join(args, "\x00"))
	if len(args) == 3 {
		if data, err := os.ReadFile(args[2]); err == nil {
			record.WriteString("\x01")
			record.Write(data)
		}
	}
	if err := os.WriteFile(marker, []byte(record.String()), 0o600); err != nil {
		return 1
	}
	_, _ = os.Stdout.WriteString("\"ok\"\n")
	return 0
}

// useFakeEmacsclient points EMACSCLIENT at this test binary and returns the
// marker path the fake writes when it is invoked.
func useFakeEmacsclient(t *testing.T) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "called")
	t.Setenv("EMACSCLIENT", self)
	t.Setenv(fakeMarkerEnv, marker)
	return marker
}

func TestAC12CLIRejectsBeforeSending(t *testing.T) {
	marker := useFakeEmacsclient(t)
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Setenv("IMOOGI_AGENT_TIMEOUT", "")
	cases := []struct {
		name string
		env  string
		args []string
	}{
		{"no arguments", "", nil},
		{"frobnicate", "", []string{"frobnicate"}},
		{"message without text", "", []string{"message"}},
		{"empty text", "", []string{"message", ""}},
		{"two texts", "", []string{"message", "a", "b"}},
		{"goto without line", "", []string{"goto", "f.go"}},
		{"goto line 0", "", []string{"goto", "f.go", "0"}},
		{"goto line x", "", []string{"goto", "f.go", "x"}},
		{"goto column 0", "", []string{"goto", "f.go", "3", "0"}},
		{"goto too many", "", []string{"goto", "f.go", "3", "4", "5"}},
		{"finish done", "", []string{"finish", "done"}},
		{"unknown option", "", []string{"message", "hi", "--bogus"}},
		{"missing option value", "", []string{"message", "hi", "--project"}},
		{"timeout 0", "", []string{"message", "hi", "--timeout", "0"}},
		{"environment timeout abc", "abc", []string{"message", "hi"}},
		{"text over 1 MiB", "", []string{"message", strings.Repeat("x", 1<<20+1)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("IMOOGI_AGENT_TIMEOUT", tc.env)
			var stdout, stderr bytes.Buffer
			code := run(tc.args, strings.NewReader(""), &stdout, &stderr)
			if code != 2 {
				t.Errorf("code = %d, want 2", code)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if got := stderr.String(); !strings.HasSuffix(got, "\n") || strings.Count(got, "\n") != 1 {
				t.Errorf("stderr = %q, want exactly one line", got)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Errorf("fake emacsclient was invoked (marker stat err=%v)", err)
			}
			if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
				t.Errorf("TMPDIR holds %d entries, want none", len(entries))
			}
		})
	}
}

func TestAC11CLIVersion(t *testing.T) {
	marker := useFakeEmacsclient(t)
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || stdout.String() != "imoogi-agent dev\n" || stderr.Len() != 0 {
		t.Errorf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("--version invoked emacsclient")
	}
}

func TestMainExitsWithRunCode(t *testing.T) {
	savedArgs, savedExit := os.Args, exit
	t.Cleanup(func() { os.Args, exit = savedArgs, savedExit })
	got := -1
	exit = func(code int) { got = code }
	os.Args = []string{"imoogi-agent", "frobnicate"}
	main()
	if got != 2 {
		t.Errorf("main exited with %d, want 2", got)
	}
}
