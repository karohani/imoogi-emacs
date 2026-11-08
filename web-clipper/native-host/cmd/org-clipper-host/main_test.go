package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/obsidian-clipper/native-host/internal/config"
	"github.com/karohani/obsidian-clipper/native-host/internal/install"
	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
)

var idA = strings.Repeat("b", 32)

// isolate points every path the binary touches at a temp dir. No test here
// may reach the real HOME.
func isolate(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("HOME", filepath.Join(base, "home"))
	t.Setenv(config.EnvConfigDir, filepath.Join(base, "cfg"))
	t.Setenv(config.EnvLogDir, filepath.Join(base, "logs"))
	t.Setenv(install.EnvChromeHostsDir, filepath.Join(base, "hosts"))
	return base
}

func cli(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(""), &out, &errb)
	return code, out.String(), errb.String()
}

func TestVersionAndUsage(t *testing.T) {
	isolate(t)
	if code, out, _ := cli(t, "--version"); code != 0 || !strings.Contains(out, version) {
		t.Fatalf("%d %q", code, out)
	}
	if code, out, _ := cli(t, "--help"); code != 0 || !strings.Contains(out, "setup --root") {
		t.Fatalf("%d %q", code, out)
	}
	if code, _, errs := cli(t, "bogus"); code != 2 || !strings.Contains(errs, "unknown command") {
		t.Fatalf("%d %q", code, errs)
	}
}

func TestSetupInstallStatusUninstall(t *testing.T) {
	base := isolate(t)
	root := filepath.Join(base, "clips")
	os.Mkdir(root, 0o700)

	for _, bad := range [][]string{{"setup"}, {"setup", "--root", "relative"}, {"setup", "--root", filepath.Join(base, "missing")}, {"setup", "--root", root, "extra"}} {
		if code, _, _ := cli(t, bad...); code == 0 {
			t.Errorf("%v accepted", bad)
		}
	}
	if code, out, errs := cli(t, "setup", "--root", root+"/"); code != 0 || !strings.Contains(out, root) {
		t.Fatalf("setup %d %q %q", code, out, errs)
	}
	if code, _, _ := cli(t, "install", "--extension-id", "BAD"); code == 0 {
		t.Fatal("bad id accepted")
	}
	if code, _, _ := cli(t, "install"); code != 2 {
		t.Fatal("install without ids accepted")
	}
	code, out, errs := cli(t, "install", "--extension-id", idA)
	if code != 0 || !strings.Contains(out, "chrome-extension://"+idA+"/") {
		t.Fatalf("install %d %q %q", code, out, errs)
	}
	bin := filepath.Join(base, "cfg", "bin", "org-clipper-host")
	if fi, err := os.Stat(bin); err != nil || fi.Mode().Perm() != 0o755 {
		t.Fatalf("binary %v %v", fi, err)
	}
	_, out, _ = cli(t, "status")
	for _, want := range []string{root, "present", idA} {
		if !strings.Contains(out, want) {
			t.Errorf("status missing %q:\n%s", want, out)
		}
	}
	if code, out, _ := cli(t, "uninstall"); code != 0 || strings.Count(out, "removed ") != 2 {
		t.Fatalf("uninstall %d %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(base, "cfg", "config.json")); err != nil {
		t.Fatal("config removed without --purge-config")
	}
	if code, out, _ := cli(t, "uninstall", "--purge-config"); code != 0 || !strings.Contains(out, "config.json") {
		t.Fatalf("purge %d %q", code, out)
	}
	_, out, _ = cli(t, "status")
	if !strings.Contains(out, "not configured") || !strings.Contains(out, "absent") {
		t.Fatalf("status after purge:\n%s", out)
	}
	if entries, _ := os.ReadDir(filepath.Join(base, "home")); len(entries) != 0 {
		t.Fatalf("wrote into HOME: %v", entries)
	}
}

func pingFrame() []byte {
	b, _ := json.Marshal(map[string]any{"schemaVersion": 1, "type": "ping"})
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], uint32(len(b)))
	return append(hdr[:], b...)
}

func TestHostModeAndOriginCheck(t *testing.T) {
	base := isolate(t)
	root := filepath.Join(base, "clips")
	os.Mkdir(root, 0o700)
	config.Save(&config.Config{Root: root, AllowedExtensionIDs: []string{idA}})

	var out bytes.Buffer
	if code := run(nil, bytes.NewReader(pingFrame()), &out, &bytes.Buffer{}); code != 0 {
		t.Fatalf("host mode exit %d", code)
	}
	msg, err := protocol.ReadMessage(&out)
	if err != nil || !strings.Contains(string(msg), `"configured":true`) || strings.Contains(string(msg), root) {
		t.Fatalf("pong %s %v", msg, err)
	}

	out.Reset()
	if code := run([]string{"chrome-extension://" + idA + "/"}, bytes.NewReader(pingFrame()), &out, &bytes.Buffer{}); code != 0 || out.Len() == 0 {
		t.Fatalf("allowed origin refused: %d", code)
	}
	out.Reset()
	other := "chrome-extension://" + strings.Repeat("c", 32) + "/"
	if code := run([]string{other}, bytes.NewReader(pingFrame()), &out, &bytes.Buffer{}); code != 1 || out.Len() != 0 {
		t.Fatalf("foreign origin served: %d %q", code, out.String())
	}
	logs, _ := os.ReadFile(filepath.Join(base, "logs", "host.log"))
	if !strings.Contains(string(logs), "type=origin ") || !strings.Contains(string(logs), "status=refused") {
		t.Fatalf("log %q", logs)
	}
	if fi, _ := os.Stat(filepath.Join(base, "logs", "host.log")); fi.Mode().Perm() != 0o600 {
		t.Fatalf("log mode %v", fi.Mode().Perm())
	}
}

func TestOriginAllowed(t *testing.T) {
	ids := []string{idA}
	for origin, want := range map[string]bool{
		"chrome-extension://" + idA + "/":                     true,
		"chrome-extension://" + idA:                           true,
		"chrome-extension://" + strings.Repeat("c", 32) + "/": false,
		"https://" + idA + "/":                                false,
		"chrome-extension://../":                              false,
	} {
		if got := originAllowed(origin, ids); got != want {
			t.Errorf("%q: %v", origin, got)
		}
	}
}
