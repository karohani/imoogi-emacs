package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
)

func TestLookupRejectsBadLedgers(t *testing.T) {
	s, dir := openTemp(t)
	reqs := filepath.Join(dir, ".org-clipper", "requests")
	os.MkdirAll(reqs, 0o700)
	p := filepath.Join(reqs, rid+".json")
	for name, body := range map[string]string{
		"corrupt":  "{",
		"mismatch": `{"type":"result","requestId":"00000000-0000-4000-8000-000000000000","status":"saved"}`,
		"oversize": `{"x":"` + strings.Repeat("a", maxLedgerBytes) + `"}`,
	} {
		os.WriteFile(p, []byte(body), 0o600)
		if r, err := s.Lookup(rid); err == nil || r != nil {
			t.Errorf("%s: accepted %+v", name, r)
		}
	}
	os.Remove(p)
	os.Mkdir(p, 0o700) // a directory where the ledger file should be
	if _, err := s.Lookup(rid); err == nil {
		t.Error("directory accepted as ledger")
	}
}

func TestInFlight(t *testing.T) {
	s, dir := openTemp(t)
	if s.InFlight(rid) || s.InFlight("bad") {
		t.Fatal("no lock yet")
	}
	release, _, _ := s.Acquire(rid)
	if !s.InFlight(rid) {
		t.Fatal("fresh lock not seen")
	}
	release()
	lock := filepath.Join(dir, ".org-clipper", "requests", rid+".lock")
	os.WriteFile(lock, nil, 0o600)
	old := time.Now().Add(-time.Hour)
	os.Chtimes(lock, old, old)
	if s.InFlight(rid) {
		t.Fatal("stale lock reported in flight")
	}
}

func TestBadIDsEverywhere(t *testing.T) {
	s, _ := openTemp(t)
	if _, _, err := s.Acquire("x"); err == nil {
		t.Error("acquire")
	}
	if err := s.WriteLedger(protocol.Result{RequestID: "x"}); err == nil {
		t.Error("ledger")
	}
	st, _ := s.Stage(rid)
	defer st.Discard()
	if _, err := st.Commit("00. Inbox/../x", "d"); err == nil {
		t.Error("traversing category accepted")
	}
	if _, err := st.Commit("", "d"); err == nil {
		t.Error("empty category accepted")
	}
}

func TestDiscardIdempotentAndAfterCommit(t *testing.T) {
	s, dir := openTemp(t)
	st, _ := s.Stage(rid)
	st.WriteDoc("x")
	if _, err := st.Commit("00. Inbox/01. Articles", "d"); err != nil {
		t.Fatal(err)
	}
	st.Discard()
	st.Discard()
	if _, err := os.Stat(filepath.Join(dir, "00. Inbox", "01. Articles", "d.org")); err != nil {
		t.Fatal("discard after commit removed the clip")
	}
	if err := st.WriteDoc("again"); err == nil {
		t.Fatal("write after commit succeeded")
	}
}

func TestWriteDocTwiceFails(t *testing.T) {
	s, _ := openTemp(t)
	st, _ := s.Stage(rid)
	defer st.Discard()
	st.WriteDoc("a")
	if err := st.WriteDoc("b"); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("O_EXCL not enforced: %v", err)
	}
}

func TestLedgerWriteFailsOnReadOnlyRequests(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	s, dir := openTemp(t)
	reqs := filepath.Join(dir, ".org-clipper", "requests")
	os.MkdirAll(reqs, 0o700)
	os.Chmod(reqs, 0o500)
	defer os.Chmod(reqs, 0o700)
	err := s.WriteLedger(protocol.Result{Type: "result", RequestID: rid, Status: protocol.StatusSaved})
	if Classify(err) != protocol.CodePermissionDenied {
		t.Fatalf("got %v", err)
	}
	if _, _, err := s.Acquire(rid); Classify(err) != protocol.CodePermissionDenied {
		t.Fatalf("acquire: %v", err)
	}
}

func TestClassifyMore(t *testing.T) {
	cases := map[string]struct {
		err  error
		want string
	}{
		"nil":      {nil, ""},
		"protocol": {&protocol.Error{Code: protocol.CodeRootUnavailable}, protocol.CodeRootUnavailable},
		"edquot":   {&fs.PathError{Err: syscall.EDQUOT}, protocol.CodeDiskFull},
		"eloop":    {&fs.PathError{Err: syscall.ELOOP}, protocol.CodePermissionDenied},
		"escape":   {errors.New("openat x: path escapes from parent"), protocol.CodePermissionDenied},
	}
	for name, tc := range cases {
		if got := Classify(tc.err); got != tc.want {
			t.Errorf("%s: %q want %q", name, got, tc.want)
		}
	}
}

func TestMessagesNeverCarryPaths(t *testing.T) {
	for _, code := range []string{protocol.CodeDiskFull, protocol.CodePermissionDenied, protocol.CodeRootUnavailable,
		protocol.CodeNotConfigured, protocol.CodeInternal, "anything"} {
		m := Message(code)
		if m == "" || strings.Contains(m, "/") && code != protocol.CodeNotConfigured {
			t.Errorf("%s: %q", code, m)
		}
	}
}
