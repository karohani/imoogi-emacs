package opener

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AC-ASRC-016: the URL-handler app is generated without Launch Services.
// Both the destination and the binary path carry spaces; the script must
// quote each on its own and look nothing up on PATH.
func TestInstallGeneratesAppWithoutRegistering(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "with space")
	binary := "/Users/someone/my tools/imoogi-anki"
	var calls [][]string
	var script string
	run := func(ctx context.Context, name string, args ...string) CmdResult {
		calls = append(calls, append([]string{name}, args...))
		if name == osacompileCommand {
			data, err := os.ReadFile(args[len(args)-1])
			if err != nil {
				t.Fatalf("osacompile source unreadable: %v", err)
			}
			script = string(data)
		}
		return CmdResult{}
	}

	app, err := Install(InstallOptions{Dest: dest, BinaryPath: binary, Register: false, Run: run})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if app != filepath.Join(dest, AppName) {
		t.Errorf("app = %s, want %s", app, filepath.Join(dest, AppName))
	}

	plist, err := os.ReadFile(filepath.Join(app, "Contents", "Info.plist"))
	if err != nil {
		t.Fatalf("Info.plist: %v", err)
	}
	for _, want := range []string{
		"<key>CFBundleURLTypes</key>",
		"<key>CFBundleURLSchemes</key>",
		"<string>imoogi-anki</string>",
		"<key>CFBundleIdentifier</key>",
		"<key>CFBundleExecutable</key>",
	} {
		if !strings.Contains(string(plist), want) {
			t.Errorf("Info.plist lacks %s", want)
		}
	}

	for _, want := range []string{
		"on open location theURL",
		`(quoted form of "/Users/someone/my tools/imoogi-anki")`,
		"(quoted form of theURL)",
		`" open "`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script lacks %s:\n%s", want, script)
		}
	}
	if script != AppleScriptSource(binary) {
		t.Errorf("osacompile got a different script than AppleScriptSource")
	}
	for _, forbidden := range []string{"PATH", "which ", "command -v"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("script consults %q", forbidden)
		}
	}

	var compiled, signed bool
	for _, c := range calls {
		switch c[0] {
		case lsregisterCommand:
			t.Errorf("lsregister called with --no-register: %q", c)
		case osacompileCommand:
			compiled = c[1] == "-o" && c[2] == app
		case codesignCommand:
			signed = c[len(c)-1] == app
		}
	}
	if !compiled || !signed {
		t.Errorf("compiled=%v signed=%v, want both (calls %q)", compiled, signed, calls)
	}
}

func TestInstallRegistersWhenAsked(t *testing.T) {
	var registered []string
	run := func(ctx context.Context, name string, args ...string) CmdResult {
		if name == lsregisterCommand {
			registered = args
		}
		return CmdResult{}
	}
	app, err := Install(InstallOptions{Dest: t.TempDir(), BinaryPath: "/bin/imoogi-anki", Register: true, Run: run})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(registered) != 2 || registered[0] != "-f" || registered[1] != app {
		t.Errorf("lsregister args = %q, want [-f %s]", registered, app)
	}
}

func TestInstallRefusesToReplaceAForeignApp(t *testing.T) {
	dest := t.TempDir()
	foreign := filepath.Join(dest, AppName, "Contents")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "Info.plist"), []byte("<plist>someone else</plist>"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(ctx context.Context, name string, args ...string) CmdResult { return CmdResult{} }
	if _, err := Install(InstallOptions{Dest: dest, BinaryPath: "/bin/imoogi-anki", Run: run}); err == nil {
		t.Errorf("Install replaced an app it did not create")
	}
}

func TestAppleScriptSourceEscapesTheBinaryPath(t *testing.T) {
	src := AppleScriptSource(`/odd "dir"\x/imoogi-anki`)
	if !strings.Contains(src, `(quoted form of "/odd \"dir\"\\x/imoogi-anki")`) {
		t.Errorf("path not escaped as an AppleScript string:\n%s", src)
	}
}
