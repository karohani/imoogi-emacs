package opener

import (
	"context"
	"errors"
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

	// The app is compiled and signed in a staging bundle beside the final
	// one, then moved into place; nothing is left behind.
	var built string
	var signed bool
	for _, c := range calls {
		switch c[0] {
		case lsregisterCommand:
			t.Errorf("lsregister called with --no-register: %q", c)
		case osacompileCommand:
			if c[1] == "-o" {
				built = c[2]
			}
		case codesignCommand:
			signed = built != "" && c[len(c)-1] == built
		}
	}
	if built == "" || built == app || filepath.Base(built) != AppName || !signed {
		t.Errorf("built=%q signed=%v, want a staged %s, signed (calls %q)", built, signed, AppName, calls)
	}
	assertOnlyApp(t, dest)
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

// writeOwnApp puts an app Install made at dest, with a marker file that a
// rebuild would not produce.
func writeOwnApp(t *testing.T, dest string) string {
	t.Helper()
	contents := filepath.Join(dest, AppName, "Contents")
	if err := os.MkdirAll(contents, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte("<plist>"+BundleID+" old</plist>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contents, "old-marker"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dest, AppName)
}

func assertOldApp(t *testing.T, app string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(app, "Contents", "old-marker")); err != nil {
		t.Errorf("the old app is gone: %v", err)
	}
	plist, err := os.ReadFile(filepath.Join(app, "Contents", "Info.plist"))
	if err != nil || string(plist) != "<plist>"+BundleID+" old</plist>" {
		t.Errorf("the old Info.plist changed: %q %v", plist, err)
	}
}

// assertOnlyApp fails when dest holds anything but the app: a staging
// directory or a set-aside copy left behind.
func assertOnlyApp(t *testing.T, dest string) {
	t.Helper()
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 1 || names[0] != AppName {
		t.Errorf("dest holds %q, want only %s", names, AppName)
	}
}

// Card review finding 3: a reinstall builds the replacement first; a failed
// build leaves the working handler exactly as it was.
func TestInstallKeepsTheOldAppWhenTheRebuildFails(t *testing.T) {
	cases := map[string]func(name string, args []string) bool{
		"osacompile": func(name string, _ []string) bool { return name == osacompileCommand },
		"codesign":   func(name string, args []string) bool { return name == codesignCommand && args[0] != "--verify" },
		"verify":     func(name string, args []string) bool { return name == codesignCommand && args[0] == "--verify" },
	}
	for label, fails := range cases {
		dest := t.TempDir()
		app := writeOwnApp(t, dest)
		run := func(ctx context.Context, name string, args ...string) CmdResult {
			if fails(name, args) {
				return CmdResult{ExitCode: 1, Stderr: []byte("boom")}
			}
			return CmdResult{}
		}
		if _, err := Install(InstallOptions{Dest: dest, BinaryPath: "/bin/imoogi-anki", Run: run}); err == nil {
			t.Errorf("%s: Install succeeded although the build failed", label)
		}
		assertOldApp(t, app)
		assertOnlyApp(t, dest)
	}
}

func TestInstallReplacesItsOwnAppAfterASuccessfulBuild(t *testing.T) {
	dest := t.TempDir()
	app := writeOwnApp(t, dest)
	run := func(ctx context.Context, name string, args ...string) CmdResult { return CmdResult{} }
	got, err := Install(InstallOptions{Dest: dest, BinaryPath: "/bin/imoogi-anki", Run: run})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if got != app {
		t.Errorf("app = %s, want %s", got, app)
	}
	if _, err := os.Stat(filepath.Join(app, "Contents", "old-marker")); !os.IsNotExist(err) {
		t.Errorf("the old app's files survived the replacement: %v", err)
	}
	plist, err := os.ReadFile(filepath.Join(app, "Contents", "Info.plist"))
	if err != nil || string(plist) != InfoPlist() {
		t.Errorf("Info.plist = %q %v, want the generated one", plist, err)
	}
	assertOnlyApp(t, dest)
}

func TestInstallRestoresTheOldAppWhenTheSwapFails(t *testing.T) {
	dest := t.TempDir()
	app := writeOwnApp(t, dest)
	intoPlace := 0
	rename := func(from, to string) error {
		if to == app {
			intoPlace++
			if intoPlace == 1 {
				return errors.New("rename refused")
			}
		}
		return os.Rename(from, to)
	}
	run := func(ctx context.Context, name string, args ...string) CmdResult { return CmdResult{} }
	if _, err := Install(InstallOptions{Dest: dest, BinaryPath: "/bin/imoogi-anki", Run: run, Rename: rename}); err == nil {
		t.Errorf("Install succeeded although the new app could not be moved into place")
	}
	assertOldApp(t, app)
	assertOnlyApp(t, dest)
}
