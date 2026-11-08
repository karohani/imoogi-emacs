package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func useTempConfig(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "cfg")
	t.Setenv(EnvConfigDir, dir)
	return dir
}

func TestLoadMissing(t *testing.T) {
	useTempConfig(t)
	c, err := Load()
	if !errors.Is(err, ErrNotConfigured) || c == nil || c.Root != "" {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := useTempConfig(t)
	root := t.TempDir()
	if err := Save(&Config{Root: root, AllowedExtensionIDs: []string{strings.Repeat("a", 32)}}); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil || c.Root != root || len(c.AllowedExtensionIDs) != 1 {
		t.Fatalf("%+v %v", c, err)
	}
	fi, _ := os.Stat(filepath.Join(dir, "config.json"))
	di, _ := os.Stat(dir)
	if fi.Mode().Perm() != 0o600 || di.Mode().Perm() != 0o700 {
		t.Fatalf("modes file=%v dir=%v", fi.Mode().Perm(), di.Mode().Perm())
	}
}

func TestLoadRejects(t *testing.T) {
	dir := useTempConfig(t)
	os.MkdirAll(dir, 0o700)
	p := filepath.Join(dir, "config.json")
	for name, body := range map[string]string{
		"unknown field": `{"root":"/tmp","x":1}`,
		"relative root": `{"root":"rel/path"}`,
		"unclean root":  `{"root":"/tmp/../etc"}`,
		"bad id":        `{"root":"/tmp","allowedExtensionIds":["nope"]}`,
		"malformed":     `{`,
		"too large":     `{"root":"/tmp","allowedExtensionIds":["` + strings.Repeat("a", 70000) + `"]}`,
	} {
		os.WriteFile(p, []byte(body), 0o600)
		if _, err := Load(); err == nil || errors.Is(err, ErrNotConfigured) {
			t.Errorf("%s: accepted (%v)", name, err)
		}
	}
	os.WriteFile(p, []byte(`{"root":"/tmp"}`), 0o600)
	os.Chmod(p, 0o666)
	if _, err := Load(); err == nil {
		t.Error("world-writable config accepted")
	}
	os.Chmod(p, 0o600)
	os.WriteFile(p, []byte(`{"root":"","allowedExtensionIds":[]}`), 0o600)
	if _, err := Load(); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("empty root: %v", err)
	}
}

func TestValidateRoot(t *testing.T) {
	d := t.TempDir()
	f := filepath.Join(d, "file")
	os.WriteFile(f, nil, 0o600)
	if got, err := ValidateRoot(d + "/./"); err != nil || got != d {
		t.Fatalf("%q %v", got, err)
	}
	for _, bad := range []string{"", "relative", "~/x", f, filepath.Join(d, "missing")} {
		if _, err := ValidateRoot(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestEnvMustBeAbsolute(t *testing.T) {
	t.Setenv(EnvConfigDir, "relative")
	if _, err := Dir(); err == nil {
		t.Fatal("relative env accepted")
	}
	t.Setenv(EnvLogDir, "/abs/logs")
	if d, err := LogDir(); err != nil || d != "/abs/logs" {
		t.Fatalf("%q %v", d, err)
	}
	t.Setenv(EnvLogDir, "")
	if d, err := LogDir(); err != nil || !strings.HasSuffix(d, filepath.Join("Library", "Logs", "org-clipper")) {
		t.Fatalf("%q %v", d, err)
	}
}
