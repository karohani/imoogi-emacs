package npmbundle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunNpmRunsArgvInDirAndReportsOutputOnFailure(t *testing.T) {
	bin := t.TempDir()
	node := filepath.Join(bin, "node")
	script := "#!/bin/sh\npwd > ran-here\nprintf '%s ' \"$@\" >> ran-here\n[ \"$1\" = fail ] && { echo 'npm ERR! 404 Not Found' >&2; exit 1; }\nexit 0\n"
	if err := os.WriteFile(node, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := runNpm(context.Background(), dir, []string{node, "ok", "ci"}); err != nil {
		t.Fatalf("runNpm() = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "ran-here"))
	if err != nil || !strings.Contains(string(got), "ok ci") {
		t.Fatalf("argv not run in dir: %q %v", got, err)
	}
	err = runNpm(context.Background(), dir, []string{node, "fail"})
	if err == nil || !strings.Contains(err.Error(), "404 Not Found") {
		t.Fatalf("runNpm() failure = %v, want npm output in the error", err)
	}
}

func TestTailKeepsTheEnd(t *testing.T) {
	if got := tail(strings.Repeat("a", 10)+"END", 5); got != "…aaEND" {
		t.Fatalf("tail() = %q", got)
	}
	if got := tail("  short \n", 50); got != "short" {
		t.Fatalf("tail() = %q", got)
	}
}

func TestCommandNamesTheLockfileAndFlags(t *testing.T) {
	got := Command("toolchains/acp/x/package-lock.json")
	if !strings.Contains(got, "toolchains/acp/x/package-lock.json") || !strings.Contains(got, strings.Join(CIArgs, " ")) {
		t.Fatalf("Command() = %q", got)
	}
}

func TestCheckPinRejectsLooseManifests(t *testing.T) {
	lock := []byte(`{"packages":{"node_modules/@scope/adapter":{"version":"1.2.3"}}}`)
	tests := map[string]struct{ pkg, lock string }{
		"extra dependency": {`{"dependencies":{"@scope/adapter":"1.2.3","left-pad":"1.0.0"}}`, string(lock)},
		"range not pin":    {`{"dependencies":{"@scope/adapter":"^1.2.3"}}`, string(lock)},
		"lock resolves":    {`{"dependencies":{"@scope/adapter":"1.2.3"}}`, `{"packages":{"node_modules/@scope/adapter":{"version":"1.2.4"}}}`},
		"bad package.json": {`{`, string(lock)},
		"bad lockfile":     {`{"dependencies":{"@scope/adapter":"1.2.3"}}`, `{`},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if err := checkPin([]byte(tc.pkg), []byte(tc.lock), testPkg, testVersion); err == nil {
				t.Fatal("checkPin() accepted it")
			}
		})
	}
}

func TestBinEntryAndLicenseRejectIncompleteInstalls(t *testing.T) {
	dir := t.TempDir()
	if _, err := BinEntry(dir, testPkg, testVersion); err == nil {
		t.Fatal("BinEntry() accepted a missing package.json")
	}
	writeTree(t, dir, map[string]string{"package.json": `{"name":"@scope/adapter","version":"1.2.3","bin":{"a":"a.js","b":"b.js"}}`})
	if _, err := BinEntry(dir, testPkg, testVersion); err == nil || !strings.Contains(err.Error(), "2 bin") {
		t.Fatalf("BinEntry() = %v, want two-bin rejection", err)
	}
	writeTree(t, dir, map[string]string{"package.json": `{"name":"@scope/adapter","version":"1.2.3","bin":{"a":"missing.js"}}`})
	if _, err := BinEntry(dir, testPkg, testVersion); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("BinEntry() = %v, want missing bin rejection", err)
	}
	if err := copyLicense(dir, filepath.Join(t.TempDir(), "LICENSE")); err == nil {
		t.Fatal("copyLicense() accepted a package without a license file")
	}
	writeTree(t, dir, map[string]string{"LICENSE.md": "md license\n"})
	out := filepath.Join(t.TempDir(), "x", "LICENSE")
	if err := copyLicense(dir, out); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(out); string(data) != "md license\n" {
		t.Fatalf("license = %q", data)
	}
}
