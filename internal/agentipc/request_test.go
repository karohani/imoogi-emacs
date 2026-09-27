package agentipc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeTransport records every event handed to it instead of delivering it.
type fakeTransport struct {
	calls    int
	event    []byte
	timeout  time.Duration
	response Result
}

func (f *fakeTransport) Deliver(event []byte, timeout time.Duration) Result {
	f.calls++
	f.event = append([]byte(nil), event...)
	f.timeout = timeout
	return f.response
}

const testWorkDir = "/work/dir"

func testDeps(env map[string]string, transport Transport) Deps {
	return Deps{
		Version:   "test-1.2.3",
		Getenv:    func(key string) string { return env[key] },
		Getwd:     func() (string, error) { return testWorkDir, nil },
		Now:       func() time.Time { return time.Date(2026, 9, 27, 10, 47, 0, 0, time.FixedZone("KST", 9*3600)) },
		Transport: transport,
	}
}

func runMain(t *testing.T, env map[string]string, transport Transport, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Main(args, &stdout, &stderr, testDeps(env, transport))
	return code, stdout.String(), stderr.String()
}

// assertNoDuplicateKeys walks the JSON token stream and fails when any object
// carries the same key twice (AC-AIPC-011: checked at the token level, since
// decoding into a map would silently keep only the last value).
func assertNoDuplicateKeys(t *testing.T, data []byte) {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch tok {
		case json.Delim('{'):
			seen := map[string]bool{}
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return err
				}
				key := keyTok.(string)
				if seen[key] {
					return fmt.Errorf("duplicate key %q", key)
				}
				seen[key] = true
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		case json.Delim('['):
			for dec.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		}
		return nil
	}
	if err := walk(); err != nil {
		t.Fatalf("token walk of %s: %v", data, err)
	}
}

func decodeEvent(t *testing.T, data []byte) map[string]any {
	t.Helper()
	assertNoDuplicateKeys(t, data)
	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("event is not JSON: %v\n%s", err, data)
	}
	if event["version"] != "1" {
		t.Errorf("version = %v, want \"1\"", event["version"])
	}
	ts, _ := event["timestamp"].(string)
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Errorf("timestamp %q is not RFC 3339: %v", ts, err)
	}
	return event
}

func TestAC11SubcommandsBuildEvents(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantType    string
		wantPayload map[string]any
		wantProject string
		wantSession string
	}{
		{"a message keeps percent", []string{"message", "안녕 %s"}, "message",
			map[string]any{"text": "안녕 %s"}, "", ""},
		{"b open-file absolutizes", []string{"open-file", "notes/a.md"}, "open-file",
			map[string]any{"path": filepath.Join(testWorkDir, "notes/a.md")}, "", ""},
		{"c goto with column and envelope options", []string{"goto", "src/x.go", "12", "5", "--project", "p1", "--session", "s1"}, "goto-location",
			map[string]any{"path": filepath.Join(testWorkDir, "src/x.go"), "line": float64(12), "column": float64(5)}, "p1", "s1"},
		{"d goto without column", []string{"goto", "src/x.go", "12"}, "goto-location",
			map[string]any{"path": filepath.Join(testWorkDir, "src/x.go"), "line": float64(12)}, "", ""},
		{"e artifact with type and title", []string{"artifact", "out/r.md", "--type", "report", "--title", "주간 보고"}, "artifact-created",
			map[string]any{"path": filepath.Join(testWorkDir, "out/r.md"), "artifactType": "report", "title": "주간 보고"}, "", ""},
		{"f finish failed with summary and artifact", []string{"finish", "failed", "빌드 실패", "--artifact", "log.txt"}, "task-finished",
			map[string]any{"status": "failed", "summary": "빌드 실패", "artifact": filepath.Join(testWorkDir, "log.txt")}, "", ""},
		{"g double dash makes a flag-like text positional", []string{"message", "--", "--not-a-flag"}, "message",
			map[string]any{"text": "--not-a-flag"}, "", ""},
		{"options before the subcommand", []string{"--session", "s9", "finish", "success"}, "task-finished",
			map[string]any{"status": "success"}, "", "s9"},
		{"absolute path kept as given", []string{"open-file", "/abs/f.txt"}, "open-file",
			map[string]any{"path": "/abs/f.txt"}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &fakeTransport{response: Result{Code: 0}}
			code, stdout, stderr := runMain(t, nil, transport, tc.args...)
			if code != 0 || stdout != "" || stderr != "" {
				t.Fatalf("code=%d stdout=%q stderr=%q, want 0 and empty output", code, stdout, stderr)
			}
			if transport.calls != 1 {
				t.Fatalf("transport called %d times, want 1", transport.calls)
			}
			event := decodeEvent(t, transport.event)
			if event["type"] != tc.wantType {
				t.Errorf("type = %v, want %s", event["type"], tc.wantType)
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
			for key, want := range map[string]string{"project": tc.wantProject, "session": tc.wantSession} {
				got, present := event[key]
				if want == "" && present {
					t.Errorf("%s present (%v), want absent", key, got)
				}
				if want != "" && got != want {
					t.Errorf("%s = %v, want %s", key, got, want)
				}
			}
		})
	}
}

func TestAC11EnvelopeFieldOrder(t *testing.T) {
	transport := &fakeTransport{}
	if code, _, stderr := runMain(t, nil, transport, "message", "hi", "--project", "p", "--session", "s"); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	want := `{"version":"1","type":"message","timestamp":"2026-09-27T10:47:00+09:00","project":"p","session":"s","payload":{"text":"hi"}}` + "\n"
	if string(transport.event) != want {
		t.Errorf("event bytes\n got %s\nwant %s", transport.event, want)
	}
}

func TestAC11EventIsNotHTMLEscaped(t *testing.T) {
	transport := &fakeTransport{}
	runMain(t, nil, transport, "message", "<a&b>")
	if !bytes.Contains(transport.event, []byte(`"text":"<a&b>"`)) {
		t.Errorf("event %s escapes HTML characters", transport.event)
	}
}

func TestAC11Version(t *testing.T) {
	transport := &fakeTransport{}
	code, stdout, stderr := runMain(t, nil, transport, "--version")
	if code != 0 || stdout != "imoogi-agent test-1.2.3\n" || stderr != "" {
		t.Errorf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if transport.calls != 0 {
		t.Errorf("--version called the transport")
	}
}

func TestAC12InvalidInvocationsExit2(t *testing.T) {
	huge := strings.Repeat("x", 1<<20)
	cases := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{"no arguments", nil, nil},
		{"unknown subcommand", nil, []string{"frobnicate"}},
		{"message without text", nil, []string{"message"}},
		{"empty text", nil, []string{"message", ""}},
		{"too many texts", nil, []string{"message", "a", "b"}},
		{"goto without line", nil, []string{"goto", "f.go"}},
		{"goto line zero", nil, []string{"goto", "f.go", "0"}},
		{"goto line not a number", nil, []string{"goto", "f.go", "x"}},
		{"goto column zero", nil, []string{"goto", "f.go", "3", "0"}},
		{"goto too many", nil, []string{"goto", "f.go", "3", "4", "5"}},
		{"goto signed line", nil, []string{"goto", "f.go", "+3"}},
		{"goto overflowing line", nil, []string{"goto", "f.go", "99999999999999999999999"}},
		{"finish unknown status", nil, []string{"finish", "done"}},
		{"finish too many", nil, []string{"finish", "success", "s", "extra"}},
		{"unknown option", nil, []string{"message", "hi", "--bogus"}},
		{"option missing value", nil, []string{"message", "hi", "--project"}},
		{"option given twice", nil, []string{"message", "hi", "--project", "a", "--project", "b"}},
		{"timeout zero", nil, []string{"message", "hi", "--timeout", "0"}},
		{"timeout negative looks like option", nil, []string{"message", "hi", "--timeout", "-1"}},
		{"timeout overflow", nil, []string{"message", "hi", "--timeout", "99999999999"}},
		{"env timeout not a number", map[string]string{"IMOOGI_AGENT_TIMEOUT": "abc"}, []string{"message", "hi"}},
		{"type option on message", nil, []string{"message", "hi", "--type", "t"}},
		{"artifact option on goto", nil, []string{"goto", "f.go", "1", "--artifact", "x"}},
		{"empty path", nil, []string{"open-file", ""}},
		{"empty artifact path", nil, []string{"finish", "success", "--artifact", ""}},
		{"open-file without path", nil, []string{"open-file"}},
		{"artifact too many", nil, []string{"artifact", "a", "b"}},
		{"version with arguments", nil, []string{"--version", "message"}},
		{"event larger than 1 MiB", nil, []string{"message", huge}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &fakeTransport{}
			code, stdout, stderr := runMain(t, tc.env, transport, tc.args...)
			if code != 2 {
				t.Errorf("code = %d, want 2", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if !strings.HasSuffix(stderr, "\n") || strings.Count(stderr, "\n") != 1 {
				t.Errorf("stderr = %q, want exactly one line", stderr)
			}
			if transport.calls != 0 {
				t.Errorf("transport called for an invalid invocation")
			}
		})
	}
}

func TestAC12EventExactlyAtLimitIsAccepted(t *testing.T) {
	transport := &fakeTransport{}
	// Measure the envelope around an empty-ish text, then pad the text so the
	// encoded event (trailing newline included) is exactly 1 MiB.
	runMain(t, nil, transport, "message", "x")
	overhead := len(transport.event) - 1
	text := strings.Repeat("x", maxEventBytes-overhead)
	code, _, stderr := runMain(t, nil, transport, "message", text)
	if code != 0 {
		t.Fatalf("code = %d stderr=%q, want 0 for an event of exactly %d bytes", code, stderr, maxEventBytes)
	}
	if len(transport.event) != maxEventBytes {
		t.Errorf("event size = %d, want %d", len(transport.event), maxEventBytes)
	}
	code, _, _ = runMain(t, nil, transport, "message", text+"x")
	if code != 2 {
		t.Errorf("one byte over the limit: code = %d, want 2", code)
	}
}

func TestTimeoutPrecedence(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		args []string
		want time.Duration
	}{
		{"default", nil, nil, 5 * time.Second},
		{"environment", map[string]string{"IMOOGI_AGENT_TIMEOUT": "7"}, nil, 7 * time.Second},
		{"empty environment is unset", map[string]string{"IMOOGI_AGENT_TIMEOUT": ""}, nil, 5 * time.Second},
		{"option beats environment", map[string]string{"IMOOGI_AGENT_TIMEOUT": "7"}, []string{"--timeout", "3"}, 3 * time.Second},
		{"option ignores invalid environment", map[string]string{"IMOOGI_AGENT_TIMEOUT": "abc"}, []string{"--timeout", "2"}, 2 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &fakeTransport{}
			args := append([]string{"message", "hi"}, tc.args...)
			if code, _, stderr := runMain(t, tc.env, transport, args...); code != 0 {
				t.Fatalf("code=%d stderr=%q", code, stderr)
			}
			if transport.timeout != tc.want {
				t.Errorf("timeout = %v, want %v", transport.timeout, tc.want)
			}
		})
	}
}

func TestTransportResultBecomesExitCode(t *testing.T) {
	transport := &fakeTransport{response: Result{Code: 3, Diag: "rejected by Emacs: bad-path"}}
	code, stdout, stderr := runMain(t, nil, transport, "message", "hi")
	if code != 3 || stdout != "" || stderr != "imoogi-agent: rejected by Emacs: bad-path\n" {
		t.Errorf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestWorkingDirectoryFailureIsNotDelivered(t *testing.T) {
	transport := &fakeTransport{}
	deps := testDeps(nil, transport)
	deps.Getwd = func() (string, error) { return "", errors.New("gone") }
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"open-file", "rel.txt"}, &stdout, &stderr, deps); code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if transport.calls != 0 || strings.Count(stderr.String(), "\n") != 1 {
		t.Errorf("calls=%d stderr=%q", transport.calls, stderr.String())
	}
}
