package agentipc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The helper-process fake: when fakeModeEnv is set this test binary acts as
// emacsclient. It records what it received, then answers with the byte
// shapes emacsclient 30.2 produced in the plan-phase measurements
// (acceptance.md § 0.3).
const (
	fakeModeEnv   = "IMOOGI_AGENT_FAKE_MODE"
	fakeRecordEnv = "IMOOGI_AGENT_FAKE_RECORD"
)

type fakeRecord struct {
	Argv0   string   `json:"argv0"`
	Args    []string `json:"args"`
	Perm    string   `json:"perm"`
	Content string   `json:"content"`
}

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeModeEnv); mode != "" {
		os.Exit(fakeEmacsclient(mode))
	}
	os.Exit(m.Run())
}

func fakeEmacsclient(mode string) int {
	record := fakeRecord{Argv0: os.Args[0], Args: os.Args[1:]}
	if n := len(record.Args); n > 0 {
		path := record.Args[n-1]
		if info, err := os.Stat(path); err == nil {
			record.Perm = strconv.FormatUint(uint64(info.Mode().Perm()), 8)
		}
		if data, err := os.ReadFile(path); err == nil {
			record.Content = string(data)
		}
	}
	if recordPath := os.Getenv(fakeRecordEnv); recordPath != "" {
		data, _ := json.Marshal(record)
		if err := os.WriteFile(recordPath, data, 0o600); err != nil {
			return 97
		}
	}
	if secs, ok := strings.CutPrefix(mode, "sleep:"); ok {
		n, _ := strconv.Atoi(secs)
		time.Sleep(time.Duration(n) * time.Second)
		mode = "ok"
	}
	switch mode {
	case "ok":
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

// useFake points EMACSCLIENT at this test binary in the given mode and
// returns the path the fake writes its record to.
func useFake(t *testing.T, mode string) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(t.TempDir(), "record.json")
	t.Setenv("EMACSCLIENT", self)
	t.Setenv(fakeModeEnv, mode)
	t.Setenv(fakeRecordEnv, record)
	return record
}

func readRecord(t *testing.T, path string) fakeRecord {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fake emacsclient left no record: %v", err)
	}
	var record fakeRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	return record
}

func TestAC13DeliveryArgumentsPermissionsAndCleanup(t *testing.T) {
	event := []byte(`{"version":"1","type":"message","timestamp":"t","payload":{"text":"hi"}}` + "\n")
	for _, mode := range []string{"ok", "reject", "elisp-error", "connect-fail", "sleep:30"} {
		t.Run(mode, func(t *testing.T) {
			recordPath := useFake(t, mode)
			t.Setenv("TMPDIR", t.TempDir())
			start := time.Now()
			Emacsclient{}.Deliver(event, time.Second)
			if elapsed := time.Since(start); elapsed > 2*time.Second {
				t.Errorf("Deliver took %v, want under 2s", elapsed)
			}
			record := readRecord(t, recordPath)
			if len(record.Args) != 3 || record.Args[0] != "--eval" || record.Args[1] != evalExpr {
				t.Fatalf("args = %q, want [--eval <evalExpr> PATH]", record.Args)
			}
			path := record.Args[2]
			if !filepath.IsAbs(path) {
				t.Errorf("event path %q is not absolute", path)
			}
			if record.Perm != "600" {
				t.Errorf("event file permissions = %s, want 600", record.Perm)
			}
			if record.Content != string(event) {
				t.Errorf("event file content = %q, want %q", record.Content, event)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("event file %s still exists after Deliver (err=%v)", path, err)
			}
		})
	}
}

func TestAC13EvalExpressionConsumesPathFirst(t *testing.T) {
	pop := strings.Index(evalExpr, "(pop server-eval-args-left)")
	fbound := strings.Index(evalExpr, "fboundp")
	receiver := strings.Index(evalExpr, "imoogi-agent-receive-file")
	if pop < 0 || fbound < 0 || receiver < 0 {
		t.Fatalf("evalExpr %q lacks pop, fboundp or the receiver name", evalExpr)
	}
	if pop > fbound || pop > receiver {
		t.Errorf("evalExpr pops the path after checking the receiver: %q", evalExpr)
	}
	if !strings.Contains(evalExpr, `"error:not-loaded"`) {
		t.Errorf("evalExpr does not answer error:not-loaded: %q", evalExpr)
	}
	if strings.Contains(evalExpr, "\n") {
		t.Errorf("evalExpr spans several lines: %q", evalExpr)
	}
}

func TestAC14ClassifiesEmacsclientOutput(t *testing.T) {
	cases := []struct {
		name     string
		stdout   string
		stderr   string
		exit     int
		wantCode int
		wantDiag string
	}{
		{"accepted", "\"ok\"\n", "", 0, 0, ""},
		{"receiver rejection", "\"error:bad-path\"\n", "", 0, 3, "bad-path"},
		{"not loaded", "\"error:not-loaded\"\n", "", 0, 3, "not-loaded"},
		{"escaped reason", "\"error:handler \\\"x\\\" \\\\\"\n", "", 0, 3, `handler "x" \`},
		{"elisp error", "", "*ERROR*: boom", 1, 3, "boom"},
		{"elisp error with exit 0", "", "*ERROR*: boom", 0, 3, "boom"},
		{"connection failure", "", "emacsclient: can't find socket; have you started the server?\nline 2\nline 3\n", 1, 1, "can't find socket"},
		{"silent failure", "", "", 1, 1, "exit status 1"},
		{"exit 0 with nil", "nil\n", "", 0, 3, "nil"},
		{"exit 0 with nothing", "", "", 0, 3, "no output"},
		{"exit 0 with other string", "\"okay\"\n", "", 0, 3, "okay"},
		{"unterminated string", "\"ok\n", "", 0, 3, ""},
		{"inner unescaped quote", "\"o\"k\"\n", "", 0, 3, ""},
		{"ok with failure exit", "\"ok\"\n", "", 1, 1, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classify([]byte(tc.stdout), []byte(tc.stderr), tc.exit)
			if got.Code != tc.wantCode {
				t.Errorf("code = %d, want %d (diag %q)", got.Code, tc.wantCode, got.Diag)
			}
			if !strings.Contains(got.Diag, tc.wantDiag) {
				t.Errorf("diag = %q, want it to contain %q", got.Diag, tc.wantDiag)
			}
			if tc.wantCode != 0 && got.Diag == "" {
				t.Errorf("failure without a diagnostic")
			}
		})
	}
}

func TestAC14DeliveryExitCodes(t *testing.T) {
	cases := []struct {
		mode     string
		wantCode int
	}{
		{"ok", 0}, {"reject", 3}, {"elisp-error", 3}, {"connect-fail", 1}, {"nil", 3}, {"empty", 3},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			useFake(t, tc.mode)
			if got := (Emacsclient{}).Deliver([]byte("{}\n"), 5*time.Second); got.Code != tc.wantCode {
				t.Errorf("code = %d (%s), want %d", got.Code, got.Diag, tc.wantCode)
			}
		})
	}
}

func TestAC14TimeoutKillsEmacsclient(t *testing.T) {
	useFake(t, "sleep:30")
	start := time.Now()
	got := Emacsclient{}.Deliver([]byte("{}\n"), time.Second)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("Deliver took %v, want under 2s", elapsed)
	}
	if got.Code != 1 || !strings.Contains(got.Diag, "timed out") {
		t.Errorf("result = %+v, want exit 1 with a timeout diagnostic", got)
	}
}

func TestAC14UnwritableTempDirIsNotDelivered(t *testing.T) {
	recordPath := useFake(t, "ok")
	dir := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", dir)
	got := Emacsclient{}.Deliver([]byte("{}\n"), time.Second)
	if got.Code != 1 {
		t.Errorf("code = %d (%s), want 1", got.Code, got.Diag)
	}
	if _, err := os.Stat(recordPath); !os.IsNotExist(err) {
		t.Errorf("emacsclient was invoked although the event file could not be created")
	}
}

// installExecutable creates an executable regular file at path.
func installExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// useApplicationsDir replaces the app-bundle candidates with the same
// patterns rooted in a temporary directory, restoring them afterwards.
func useApplicationsDir(t *testing.T) string {
	t.Helper()
	apps := t.TempDir()
	saved := appBundleCandidates
	t.Cleanup(func() { appBundleCandidates = saved })
	appBundleCandidates = nil
	for _, candidate := range saved {
		appBundleCandidates = append(appBundleCandidates,
			filepath.Join(apps, strings.TrimPrefix(candidate, "/Applications/")))
	}
	return apps
}

func bundle(apps, name string) string {
	return filepath.Join(apps, name, "Contents/MacOS/bin/emacsclient")
}

func TestAC14DiscoveryOrder(t *testing.T) {
	cases := []struct {
		name          string
		emacsclient   string // "exec", "noexec", or "" (unset)
		onPath        bool
		bundles       []string
		emacsVersion  string
		want          string // "EMACSCLIENT", "PATH", a bundle name, or "" for failure
		wantErrSubstr string
	}{
		{"a executable EMACSCLIENT wins", "exec", true, []string{"Emacs.app"}, "", "EMACSCLIENT", ""},
		{"b non-executable EMACSCLIENT fails", "noexec", true, nil, "", "", "not executable"},
		{"c PATH before bundles", "", true, []string{"Emacs-31.1.app", "Emacs.app"}, "", "PATH", ""},
		{"d default version bundle first", "", false, []string{"Emacs-31.1.app", "Emacs.app"}, "", "Emacs-31.1.app", ""},
		{"e missing EMACS_VERSION bundle falls back to Emacs.app", "", false, []string{"Emacs-31.1.app", "Emacs.app"}, "30.2", "Emacs.app", ""},
		{"e2 EMACS_VERSION selects its bundle", "", false, []string{"Emacs-30.2.app", "Emacs.app"}, "30.2", "Emacs-30.2.app", ""},
		{"f glob finds any versioned bundle", "", false, []string{"Emacs-29.4.app"}, "", "Emacs-29.4.app", ""},
		{"g nothing found", "", false, nil, "", "", "not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apps := useApplicationsDir(t)
			for _, name := range tc.bundles {
				installExecutable(t, bundle(apps, name))
			}
			pathDir := t.TempDir()
			if tc.onPath {
				installExecutable(t, filepath.Join(pathDir, "emacsclient"))
			}
			t.Setenv("PATH", pathDir)
			t.Setenv("EMACS_VERSION", tc.emacsVersion)
			explicit := filepath.Join(t.TempDir(), "my-emacsclient")
			switch tc.emacsclient {
			case "exec":
				installExecutable(t, explicit)
				t.Setenv("EMACSCLIENT", explicit)
			case "noexec":
				if err := os.WriteFile(explicit, nil, 0o644); err != nil {
					t.Fatal(err)
				}
				t.Setenv("EMACSCLIENT", explicit)
			default:
				t.Setenv("EMACSCLIENT", "")
			}

			got, err := findEmacsclient()
			if tc.want == "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("findEmacsclient() = %q, %v; want error containing %q", got, err, tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("findEmacsclient() error: %v", err)
			}
			want := map[string]string{
				"EMACSCLIENT": explicit,
				"PATH":        filepath.Join(pathDir, "emacsclient"),
			}[tc.want]
			if want == "" {
				want = bundle(apps, tc.want)
			}
			if got != want {
				t.Errorf("findEmacsclient() = %q, want %q", got, want)
			}
		})
	}
}

func TestAC14DiscoveryIgnoresDirectoriesAndRelativePath(t *testing.T) {
	apps := useApplicationsDir(t)
	if err := os.MkdirAll(bundle(apps, "Emacs.app"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	t.Setenv("EMACSCLIENT", "")
	if got, err := findEmacsclient(); err == nil {
		t.Errorf("a directory named emacsclient was accepted: %q", got)
	}

	dir := t.TempDir()
	installExecutable(t, filepath.Join(dir, "ec"))
	t.Chdir(dir)
	t.Setenv("EMACSCLIENT", "ec")
	got, err := findEmacsclient()
	if err != nil || !filepath.IsAbs(got) {
		t.Errorf("relative EMACSCLIENT resolved to %q, %v; want an absolute path", got, err)
	}
}

func TestAC14DeliveryWithoutEmacsclient(t *testing.T) {
	useApplicationsDir(t)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("EMACSCLIENT", "")
	got := Emacsclient{}.Deliver([]byte("{}\n"), time.Second)
	if got.Code != 1 || !strings.Contains(got.Diag, "not found") {
		t.Errorf("result = %+v, want exit 1 with not found", got)
	}
}

// TestDiscoveryCandidatesMatchImoogiEditor watches for drift between the Go
// candidate list and scripts/imoogi-editor find_emacsclient (plan.md R-2):
// every Go candidate must appear there, in the same order.
func TestDiscoveryCandidatesMatchImoogiEditor(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("..", "..", "scripts", "imoogi-editor"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(script)
	envAt := strings.Index(text, `"${EMACSCLIENT:-}"`)
	pathAt := strings.Index(text, "command -v emacsclient")
	if envAt < 0 || pathAt < 0 || envAt > pathAt {
		t.Fatalf("imoogi-editor no longer checks EMACSCLIENT before PATH")
	}
	last := pathAt
	for _, candidate := range appBundleCandidates {
		at := strings.Index(text[last:], candidate)
		if at < 0 {
			t.Fatalf("candidate %q not found after position %d in scripts/imoogi-editor", candidate, last)
		}
		last += at + len(candidate)
	}
}
