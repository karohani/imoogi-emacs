package emacsctl

import (
	"path/filepath"
	"testing"
)

func TestTaskCreateEnvelope(t *testing.T) {
	for _, withFile := range []bool{false, true} {
		tr := &fakeTransport{response: Result{Code: 0}}
		args := []string{"task-create", "Fix login", "--external-system", "internal tracker", "--external-id", "PROJ-123", "--external-url", "https://tracker.example/PROJ-123"}
		if withFile {
			args = append(args, "--file", "tasks.org")
		}
		code, _, stderr := runMain(t, nil, tr, args...)
		if code != 0 {
			t.Fatalf("code=%d stderr=%s", code, stderr)
		}
		event := decodeEvent(t, tr.event)
		p := event["payload"].(map[string]any)
		if event["type"] != "task-create" || p["title"] != "Fix login" || p["externalSystem"] != "internal tracker" || p["externalId"] != "PROJ-123" || p["externalUrl"] != "https://tracker.example/PROJ-123" {
			t.Fatalf("event=%v", event)
		}
		if withFile && p["path"] != filepath.Join(testWorkDir, "tasks.org") {
			t.Fatalf("path=%v", p["path"])
		}
		if !withFile {
			if _, ok := p["path"]; ok {
				t.Fatalf("default path must be resolved by Emacs: %v", p)
			}
		}
	}
}

func TestTaskCreateWithoutExternalMetadata(t *testing.T) {
	tr := &fakeTransport{response: Result{Code: 0}}
	if code, _, err := runMain(t, nil, tr, "task-create", "Read notes"); code != 0 {
		t.Fatalf("code=%d err=%s", code, err)
	}
	if p := decodeEvent(t, tr.event)["payload"].(map[string]any); len(p) != 1 || p["title"] != "Read notes" {
		t.Fatalf("payload=%v", p)
	}
}

func TestTaskCreateRejectsInvalidInputBeforeDelivery(t *testing.T) {
	cases := [][]string{
		{"task-create"}, {"task-create", " "}, {"task-create", "a\n* injected"}, {"task-create", "a\x00b"},
		{"task-create", "a", "--external-system", "jira"},
		{"task-create", "a", "--external-id", "A-1"},
		{"task-create", "a", "--external-url", "https://example.com"},
		{"task-create", "a", "--external-system", "jira", "--external-id", "A-1\n:ID: injected"},
		{"task-create", "a", "--external-system", "", "--external-id", "A-1"},
		{"task-create", "a", "--external-system", "jira", "--external-id", "A-1", "--external-url", "file:///tmp/x"},
		{"task-create", "a", "--external-system", "jira", "--external-id", "A-1", "--external-url", "https:///path"},
		{"task-create", "a", "--external-system", "jira", "--external-id", "A-1", "--external-url", "https://example.com/a b"},
	}
	for _, args := range cases {
		tr := &fakeTransport{}
		code, _, _ := runMain(t, nil, tr, args...)
		if code != 2 || tr.calls != 0 {
			t.Errorf("%q: code=%d calls=%d", args, code, tr.calls)
		}
	}
}

func TestTaskCreateReviewValidationParity(t *testing.T) {
	for _, value := range []string{
		"https://example.com:bad", "https://[bad", "https://example.com/%zz",
		"https://example.com/?q=%zz", "https://example.com/%", "https://example.com/#%zz",
	} {
		tr := &fakeTransport{}
		code, _, _ := runMain(t, nil, tr, "task-create", "Task", "--external-system", "jira", "--external-id", "A-1", "--external-url", value)
		if code != 2 || tr.calls != 0 {
			t.Errorf("URL %q: code=%d calls=%d", value, code, tr.calls)
		}
	}
	for _, char := range []rune{127, 128, 133, 159, 8232, 8233, 160, 8199, 8239, 12288} {
		tr := &fakeTransport{}
		code, _, _ := runMain(t, nil, tr, "task-create", string(char))
		if code != 2 || tr.calls != 0 {
			t.Errorf("title U+%04X: code=%d calls=%d", char, code, tr.calls)
		}
	}
	for _, value := range []string{"https://example.com:443/a%20b?q=x%2Fy%25%20#part", "http://[::1]:8080/a?q=1", "HTTPS://example.com/a"} {
		tr := &fakeTransport{}
		code, _, stderr := runMain(t, nil, tr, "task-create", "Task", "--external-system", "jira", "--external-id", "A-1", "--external-url", value)
		if code != 0 || tr.calls != 1 {
			t.Errorf("URL %q: code=%d calls=%d stderr=%s", value, code, tr.calls, stderr)
		}
	}
}
