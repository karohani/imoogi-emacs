package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeMarkerEnv names the file the fake emacsclient records into when it
// runs; fakeModeEnv selects its answer. When fakeMarkerEnv is set, this test
// binary behaves as that fake instead of running tests.
const (
	fakeMarkerEnv = "IMOOGI_AGENT_TEST_FAKE_MARKER"
	fakeModeEnv   = "IMOOGI_AGENT_TEST_FAKE_MODE"
)

type fakeRecord struct {
	Args    []string `json:"args"`
	Perm    string   `json:"perm"`
	Content string   `json:"content"`
}

func TestMain(m *testing.M) {
	if marker := os.Getenv(fakeMarkerEnv); marker != "" {
		os.Exit(fakeEmacsclient(marker, os.Getenv(fakeModeEnv)))
	}
	os.Exit(m.Run())
}

// fakeEmacsclient records its arguments and the event file's permission bits
// and contents, then answers the way emacsclient 30.2 was measured to
// (acceptance.md § 0.3). Mode "" accepts.
func fakeEmacsclient(marker, mode string) int {
	record := fakeRecord{Args: os.Args[1:]}
	if n := len(record.Args); n > 0 {
		if info, err := os.Stat(record.Args[n-1]); err == nil {
			record.Perm = strconv.FormatUint(uint64(info.Mode().Perm()), 8)
		}
		if data, err := os.ReadFile(record.Args[n-1]); err == nil {
			record.Content = string(data)
		}
	}
	data, _ := json.Marshal(record)
	if err := os.WriteFile(marker, data, 0o600); err != nil {
		return 97
	}
	if secs, ok := strings.CutPrefix(mode, "sleep:"); ok {
		n, _ := strconv.Atoi(secs)
		time.Sleep(time.Duration(n) * time.Second)
		mode = ""
	}
	switch mode {
	case "":
		_, _ = os.Stdout.WriteString("\"ok\"\n")
		return 0
	case "reject":
		_, _ = os.Stdout.WriteString("\"error:bad-path\"\n")
		return 0
	case "elisp-error":
		_, _ = os.Stderr.WriteString("*ERROR*: boom")
		return 1
	case "connect-fail":
		_, _ = os.Stderr.WriteString("emacsclient: can't find socket; have you started the server?\n" +
			"emacsclient: To start the server in Emacs, type \"M-x server-start\".\n" +
			"emacsclient: No socket or alternate editor.  Please use:\n")
		return 1
	case "nil":
		_, _ = os.Stdout.WriteString("nil\n")
		return 0
	case "empty":
		return 0
	}
	return 98
}

// useFakeEmacsclient points EMACSCLIENT at this test binary answering in
// mode (none or "" accepts) and returns the marker path the fake writes when
// it is invoked.
func useFakeEmacsclient(t *testing.T, mode ...string) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "called")
	t.Setenv("EMACSCLIENT", self)
	t.Setenv(fakeMarkerEnv, marker)
	t.Setenv(fakeModeEnv, strings.Join(mode, ""))
	t.Setenv("IMOOGI_AGENT_TIMEOUT", "")
	return marker
}

func readRecord(t *testing.T, marker string) fakeRecord {
	t.Helper()
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("fake emacsclient was not invoked: %v", err)
	}
	var record fakeRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	return record
}

func invoke(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(""), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestAC12CLIRejectsBeforeSending(t *testing.T) {
	marker := useFakeEmacsclient(t)
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
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
			code, stdout, stderr := invoke(tc.args...)
			if code != 2 {
				t.Errorf("code = %d, want 2", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if !strings.HasSuffix(stderr, "\n") || strings.Count(stderr, "\n") != 1 {
				t.Errorf("stderr = %q, want exactly one line", stderr)
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
	code, stdout, stderr := invoke("--version")
	if code != 0 || stdout != "imoogi-agent dev\n" || stderr != "" {
		t.Errorf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
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

func TestAC11CLIEventsReachEmacsclient(t *testing.T) {
	work := t.TempDir()
	t.Chdir(work)
	cases := []struct {
		args        []string
		wantType    string
		wantPayload map[string]any
		wantEnv     bool
	}{
		{[]string{"message", "안녕 %s"}, "message", map[string]any{"text": "안녕 %s"}, false},
		{[]string{"open-file", "notes/a.md"}, "open-file", map[string]any{"path": filepath.Join(work, "notes/a.md")}, false},
		{[]string{"goto", "src/x.go", "12", "5", "--project", "p1", "--session", "s1"}, "goto-location",
			map[string]any{"path": filepath.Join(work, "src/x.go"), "line": float64(12), "column": float64(5)}, true},
		{[]string{"goto", "src/x.go", "12"}, "goto-location",
			map[string]any{"path": filepath.Join(work, "src/x.go"), "line": float64(12)}, false},
		{[]string{"artifact", "out/r.md", "--type", "report", "--title", "주간 보고"}, "artifact-created",
			map[string]any{"path": filepath.Join(work, "out/r.md"), "artifactType": "report", "title": "주간 보고"}, false},
		{[]string{"finish", "failed", "빌드 실패", "--artifact", "log.txt"}, "task-finished",
			map[string]any{"status": "failed", "summary": "빌드 실패", "artifact": filepath.Join(work, "log.txt")}, false},
		{[]string{"message", "--", "--not-a-flag"}, "message", map[string]any{"text": "--not-a-flag"}, false},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			marker := useFakeEmacsclient(t)
			code, stdout, stderr := invoke(tc.args...)
			if code != 0 || stdout != "" || stderr != "" {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			var event map[string]any
			if err := json.Unmarshal([]byte(readRecord(t, marker).Content), &event); err != nil {
				t.Fatal(err)
			}
			if event["version"] != "1" || event["type"] != tc.wantType {
				t.Errorf("version=%v type=%v, want 1 and %s", event["version"], event["type"], tc.wantType)
			}
			ts, _ := event["timestamp"].(string)
			if _, err := time.Parse(time.RFC3339, ts); err != nil {
				t.Errorf("timestamp: %v", err)
			}
			_, hasProject := event["project"]
			_, hasSession := event["session"]
			if hasProject != tc.wantEnv || hasSession != tc.wantEnv {
				t.Errorf("project/session present = %v/%v, want %v", hasProject, hasSession, tc.wantEnv)
			}
			payload, _ := event["payload"].(map[string]any)
			if len(payload) != len(tc.wantPayload) {
				t.Errorf("payload = %v, want %v", payload, tc.wantPayload)
			}
			for key, want := range tc.wantPayload {
				if payload[key] != want {
					t.Errorf("payload[%s] = %v, want %v", key, payload[key], want)
				}
			}
		})
	}
}

func TestAC13CLIEventFileLifecycle(t *testing.T) {
	for _, mode := range []string{"", "reject", "elisp-error", "connect-fail", "sleep:30"} {
		t.Run("mode="+mode, func(t *testing.T) {
			marker := useFakeEmacsclient(t, mode)
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			invoke("message", "hi", "--timeout", "1")
			record := readRecord(t, marker)
			if len(record.Args) != 3 || record.Args[0] != "--eval" {
				t.Fatalf("args = %q, want --eval EXPR PATH", record.Args)
			}
			expr, path := record.Args[1], record.Args[2]
			if strings.Contains(expr, path) || strings.Contains(expr, "hi") {
				t.Errorf("evaluation expression embeds the event or its path: %q", expr)
			}
			if !filepath.IsAbs(path) || record.Perm != "600" {
				t.Errorf("path=%q perm=%s, want an absolute 0600 file", path, record.Perm)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("event file %s survived the CLI", path)
			}
			if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
				t.Errorf("TMPDIR holds %d entries after the CLI finished", len(entries))
			}
		})
	}
}

func TestAC13CLIFixedExpressionIsByteIdentical(t *testing.T) {
	var exprs []string
	for _, args := range [][]string{{"message", "one"}, {"open-file", "/x"}, {"finish", "success"}} {
		marker := useFakeEmacsclient(t)
		invoke(args...)
		record := readRecord(t, marker)
		if len(record.Args) != 3 {
			t.Fatalf("args = %q", record.Args)
		}
		exprs = append(exprs, record.Args[1])
	}
	if exprs[0] != exprs[1] || exprs[1] != exprs[2] {
		t.Errorf("evaluation expressions differ between calls: %q", exprs)
	}
}

func TestAC14CLIExitCodes(t *testing.T) {
	cases := []struct {
		mode     string
		wantCode int
		wantDiag string
	}{
		{"", 0, ""}, {"reject", 3, "bad-path"}, {"elisp-error", 3, "boom"},
		{"connect-fail", 1, "can't find socket"}, {"nil", 3, "nil"}, {"empty", 3, "no output"},
	}
	for _, tc := range cases {
		t.Run("mode="+tc.mode, func(t *testing.T) {
			useFakeEmacsclient(t, tc.mode)
			code, stdout, stderr := invoke("message", "hi")
			if code != tc.wantCode || stdout != "" {
				t.Errorf("code=%d stdout=%q stderr=%q, want %d and empty stdout", code, stdout, stderr, tc.wantCode)
			}
			if tc.wantCode != 0 && (!strings.Contains(stderr, tc.wantDiag) || strings.Count(stderr, "\n") != 1) {
				t.Errorf("stderr = %q, want one line containing %q", stderr, tc.wantDiag)
			}
		})
	}
}

func TestAC14CLITimeLimits(t *testing.T) {
	cases := []struct {
		name     string
		env      string
		args     []string
		sleep    string
		wantCode int
		within   time.Duration
	}{
		{"--timeout 1 kills a hung emacsclient", "", []string{"--timeout", "1"}, "sleep:30", 1, 2 * time.Second},
		{"--timeout 3 beats IMOOGI_AGENT_TIMEOUT=1", "1", []string{"--timeout", "3"}, "sleep:2", 0, 3 * time.Second},
		{"default limit is 5 seconds", "", nil, "sleep:7", 1, 6 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useFakeEmacsclient(t, tc.sleep)
			t.Setenv("IMOOGI_AGENT_TIMEOUT", tc.env)
			start := time.Now()
			code, _, stderr := invoke(append([]string{"message", "hi"}, tc.args...)...)
			elapsed := time.Since(start)
			if code != tc.wantCode {
				t.Errorf("code = %d (stderr %q), want %d", code, stderr, tc.wantCode)
			}
			if elapsed > tc.within {
				t.Errorf("CLI took %v, want under %v", elapsed, tc.within)
			}
		})
	}
}

func TestAC14CLIUnwritableTempDir(t *testing.T) {
	marker := useFakeEmacsclient(t)
	dir := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", dir)
	if code, _, stderr := invoke("message", "hi"); code != 1 || strings.Count(stderr, "\n") != 1 {
		t.Errorf("code=%d stderr=%q, want 1 and one line", code, stderr)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("emacsclient was invoked without an event file")
	}
}

func TestAC14CLINonExecutableEmacsclient(t *testing.T) {
	marker := useFakeEmacsclient(t)
	plain := filepath.Join(t.TempDir(), "emacsclient")
	if err := os.WriteFile(plain, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EMACSCLIENT", plain)
	if code, _, stderr := invoke("message", "hi"); code != 1 || !strings.Contains(stderr, "not executable") {
		t.Errorf("code=%d stderr=%q, want 1 naming the non-executable EMACSCLIENT", code, stderr)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("a fallback emacsclient was invoked")
	}
}
