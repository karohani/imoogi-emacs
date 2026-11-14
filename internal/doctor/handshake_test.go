package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func script(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "adapter")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHandshakeReadsTheInitializeResult(t *testing.T) {
	// Logs on stdout before the response must not confuse the reader.
	launcher := script(t, `read line
case "$line" in *'"method":"initialize"'*) ;; *) echo "bad request: $line" >&2; exit 3 ;; esac
echo 'starting adapter'
echo '{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1}}'
cat >/dev/null
`)
	if err := (SystemEnv{}).Handshake(launcher); err != nil {
		t.Fatalf("Handshake() = %v", err)
	}
}

func TestHandshakeReportsErrorsExitsAndSilence(t *testing.T) {
	old := handshakeTimeout
	defer func() { handshakeTimeout = old }()

	// Only the silence case needs a short timeout. The others answer or exit
	// on their own, so a generous timeout keeps them from racing process
	// start-up when the whole suite runs in parallel.
	tests := map[string]struct {
		body, want string
		timeout    time.Duration
	}{
		"rpc error": {`read line; echo '{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"no auth"}}'`, "no auth", 5 * time.Second},
		"exit":      {`echo 'cannot find module' >&2; exit 1`, "cannot find module", 5 * time.Second},
		"silence":   {`sleep 30`, "no initialize response", 500 * time.Millisecond},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			handshakeTimeout = tc.timeout
			start := time.Now()
			err := (SystemEnv{}).Handshake(script(t, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Handshake() = %v, want %q", err, tc.want)
			}
			if time.Since(start) > tc.timeout+3*time.Second {
				t.Fatalf("Handshake took %s; the timeout or process cleanup did not apply", time.Since(start))
			}
		})
	}
}
