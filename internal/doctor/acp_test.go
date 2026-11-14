package doctor

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

const (
	bundle     = "2026.11.14.1"
	claudeSHA  = "1111111111111111111111111111111111111111111111111111111111111111"
	codexSHA   = "2222222222222222222222222222222222222222222222222222222222222222"
	claudeBin  = root + "/.local/bin/claude-agent-acp"
	codexBin   = root + "/.local/bin/codex-acp"
	lockPath   = root + "/toolchains.lock.json"
	installRec = root + "/.local/toolchains/" + bundle + "/install.json"
)

const lockJSON = `{"bundle":"` + bundle + `","components":[
 {"name":"node","kind":"node-runtime","upstream_version":"v24.19.0","artifact":{"sha256":"3333333333333333333333333333333333333333333333333333333333333333"}},
 {"name":"claude-agent-acp","kind":"acp-adapter","upstream_version":"0.99.1","artifact":{"sha256":"` + claudeSHA + `"}},
 {"name":"codex-acp","kind":"acp-adapter","upstream_version":"2.5.0","artifact":{"sha256":"` + codexSHA + `"}}]}`

func lockDigest() string {
	sum := sha256.Sum256([]byte(lockJSON))
	return hex.EncodeToString(sum[:])
}

func installJSON(lockSum, claude string) []byte {
	return []byte(`{"schema":"imoogi-toolchain-install/v2","bundle":"` + bundle + `","lock_sha256":"` + lockSum + `","components":[
 {"name":"claude-agent-acp","artifact_sha256":"` + claude + `"},{"name":"codex-acp","artifact_sha256":"` + codexSHA + `"}]}`)
}

// withAdapters is healthy() plus both adapters installed, answering, and their CLIs on PATH.
func withAdapters() fakeEnv {
	env := healthy()
	env.files = map[string][]byte{lockPath: []byte(lockJSON), installRec: installJSON(lockDigest(), claudeSHA)}
	env.links[root+"/.local/bin"] = "toolchains/" + bundle + "/bin"
	env.exists[claudeBin], env.exists[codexBin] = true, true
	env.answers = map[string]error{claudeBin: nil, codexBin: nil}
	env.path = map[string]string{"claude": "/home/u/.local/bin/claude", "codex": "/opt/homebrew/bin/codex"}
	return env
}

func find(t *testing.T, findings []Finding, name string) Finding {
	t.Helper()
	for _, f := range findings {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no finding %q in %+v", name, findings)
	return Finding{}
}

func TestAdaptersInstalledAndAnsweringAreOK(t *testing.T) {
	findings := Check(withAdapters())
	for _, f := range findings {
		if f.Status != StatusOK {
			t.Fatalf("unexpected finding: %+v", f)
		}
	}
	if len(findings) != 9 {
		t.Fatalf("want 5 base + 2 adapters + 2 CLIs, got %d: %+v", len(findings), findings)
	}
	if f := find(t, findings, "claude-agent-acp"); f.Kind != "acp" || !strings.Contains(f.Detail, "initialize") {
		t.Fatalf("adapter finding = %+v", f)
	}
	if f := find(t, findings, "claude"); f.Kind != "cli" || !strings.Contains(f.Detail, "/home/u/.local/bin/claude") {
		t.Fatalf("cli finding = %+v", f)
	}
}

func TestNoAdapterFindingsWithoutAdaptersInTheLock(t *testing.T) {
	env := healthy()
	env.files = map[string][]byte{lockPath: []byte(`{"bundle":"b","components":[{"name":"gopls","kind":"go-language-server"}]}`)}
	if got := len(Check(env)); got != 5 {
		t.Fatalf("findings = %d, want the 5 base checks", got)
	}
}

func TestAdapterInstallProblemsAreClassified(t *testing.T) {
	tests := map[string]struct {
		edit   func(*fakeEnv)
		status string
		detail string
	}{
		"bundle not installed": {func(e *fakeEnv) { delete(e.files, installRec) }, StatusMissing, bundle},
		"other lock installed": {func(e *fakeEnv) { e.files[installRec] = installJSON(strings.Repeat("9", 64), claudeSHA) }, StatusStale, "toolchains.lock.json"},
		"other artifact":       {func(e *fakeEnv) { e.files[installRec] = installJSON(lockDigest(), codexSHA) }, StatusStale, "artifact"},
		"other bundle active":  {func(e *fakeEnv) { e.links[root+"/.local/bin"] = "toolchains/2026.10.12.1/bin" }, StatusStale, "2026.10.12.1"},
		"launcher missing":     {func(e *fakeEnv) { e.exists[claudeBin] = false }, StatusMissing, "launcher"},
		"no handshake":         {func(e *fakeEnv) { e.answers[claudeBin] = errors.New("exit status 1: boom") }, StatusFailed, "boom"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			env := withAdapters()
			tc.edit(&env)
			findings := Check(env)
			f := find(t, findings, "claude-agent-acp")
			if f.Status != tc.status || !strings.Contains(f.Detail, tc.detail) || !strings.Contains(f.Fix, "make toolchain-setup") {
				t.Fatalf("finding = %+v, want status %s detail containing %q", f, tc.status, tc.detail)
			}
			if Healthy(findings) {
				t.Fatal("an adapter problem must fail the doctor")
			}
		})
	}
}

func TestAdapterSkippedBySetupNeedsAttentionWithTheRemedy(t *testing.T) {
	env := withAdapters()
	record := strings.Replace(string(installJSON(lockDigest(), claudeSHA)), `"components"`, `"skipped":["claude-agent-acp"],"components"`, 1)
	env.files[installRec] = []byte(record)
	env.exists[claudeBin] = false
	lock := strings.Replace(lockJSON, `"upstream_version":"0.99.1",`, `"upstream_version":"0.99.1","artifact_policy":"local-build",`, 1)
	lock = strings.Replace(lock, `"artifact":{"sha256":"`+claudeSHA+`"}`, `"artifact":{"path":"vendor/toolchains/acp/claude-agent-acp/0.99.1/darwin-arm64/bundle.tgz","sha256":"`+claudeSHA+`"}`, 1)
	env.files[lockPath] = []byte(lock)
	sum := sha256.Sum256([]byte(lock))
	env.files[installRec] = []byte(strings.Replace(record, lockDigest(), hex.EncodeToString(sum[:]), 1))

	findings := Check(env)
	f := find(t, findings, "claude-agent-acp")
	if f.Status != StatusMissing || !strings.Contains(f.Detail, "skipped") ||
		!strings.Contains(f.Fix, "make toolchain-setup") || !strings.Contains(f.Fix, "vendor/toolchains/acp/claude-agent-acp/0.99.1/darwin-arm64/bundle.tgz") {
		t.Fatalf("finding = %+v", f)
	}
	if find(t, findings, "codex-acp").Status != StatusOK {
		t.Fatal("the other adapter must still be checked normally")
	}
	if Healthy(findings) {
		t.Fatal("a skipped adapter needs attention")
	}
}

func TestMissingAgentCLIIsAWarning(t *testing.T) {
	env := withAdapters()
	delete(env.path, "codex")
	findings := Check(env)
	f := find(t, findings, "codex")
	if f.Status != StatusWarn || !strings.Contains(f.Detail, "codex-acp") || !strings.Contains(f.Fix, "CODEX_PATH") {
		t.Fatalf("finding = %+v", f)
	}
	if !Healthy(findings) {
		t.Fatal("a missing agent CLI is a warning, not a failure")
	}

	env.vars = map[string]string{"CODEX_PATH": "/opt/codex/bin/codex"}
	if f := find(t, Check(env), "codex"); f.Status != StatusWarn || !strings.Contains(f.Detail, "/opt/codex/bin/codex") {
		t.Fatalf("CODEX_PATH to a missing file: %+v", f)
	}
	env.exists["/opt/codex/bin/codex"] = true
	if f := find(t, Check(env), "codex"); f.Status != StatusOK || !strings.Contains(f.Detail, "CODEX_PATH") {
		t.Fatalf("CODEX_PATH to an existing file: %+v", f)
	}
}

func TestUnreadableLockIsUnknown(t *testing.T) {
	env := withAdapters()
	env.files[lockPath] = []byte("{not json")
	if f := find(t, Check(env), "toolchains.lock.json"); f.Status != StatusUnknown {
		t.Fatalf("finding = %+v", f)
	}
}
