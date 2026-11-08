package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/obsidian-clipper/native-host/internal/config"
)

var (
	idA = strings.Repeat("a", 32)
	idB = strings.Repeat("p", 32)
)

func env(t *testing.T) (cfgDir, hostsDir string) {
	t.Helper()
	base := t.TempDir()
	cfgDir, hostsDir = filepath.Join(base, "cfg"), filepath.Join(base, "hosts")
	t.Setenv(config.EnvConfigDir, cfgDir)
	t.Setenv(EnvChromeHostsDir, hostsDir)
	return
}

func fakeBinary(t *testing.T) string {
	p := filepath.Join(t.TempDir(), "src-bin")
	os.WriteFile(p, []byte("#!binary"), 0o700)
	return p
}

func TestInstallUninstall(t *testing.T) {
	cfgDir, hostsDir := env(t)
	root := t.TempDir()
	if err := config.Save(&config.Config{Root: root}); err != nil {
		t.Fatal(err)
	}
	res, err := Install(fakeBinary(t), []string{idA, idB, idA})
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(cfgDir, "bin", "org-clipper-host")
	if res.Binary != bin {
		t.Fatalf("binary %q", res.Binary)
	}
	fi, err := os.Stat(bin)
	if err != nil || fi.Mode().Perm() != 0o755 {
		t.Fatalf("binary %v %v", fi, err)
	}
	b, err := os.ReadFile(filepath.Join(hostsDir, HostName+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["name"] != HostName || m["type"] != "stdio" || m["path"] != bin || len(m) != 5 {
		t.Fatalf("manifest %s", b)
	}
	origins := m["allowed_origins"].([]any)
	if len(origins) != 2 || origins[0] != "chrome-extension://"+idA+"/" || origins[1] != "chrome-extension://"+idB+"/" {
		t.Fatalf("origins %v", origins)
	}
	c, err := config.Load()
	if err != nil || c.Root != root || len(c.AllowedExtensionIDs) != 2 {
		t.Fatalf("config after install %+v %v", c, err)
	}

	removed, err := Uninstall(false)
	if err != nil || len(removed) != 2 {
		t.Fatalf("uninstall %v %v", removed, err)
	}
	if _, err := config.Load(); err != nil {
		t.Fatalf("config removed without --purge-config: %v", err)
	}
	removed, err = Uninstall(true)
	if err != nil || len(removed) != 1 || !strings.HasSuffix(removed[0], "config.json") {
		t.Fatalf("purge %v %v", removed, err)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatal("uninstall touched the save root")
	}
}

func TestInstallRejectsBadIDs(t *testing.T) {
	env(t)
	for _, ids := range [][]string{nil, {"short"}, {strings.Repeat("q", 32)}, {strings.Repeat("A", 32)}, {idA, "../x"}} {
		if _, err := Install(fakeBinary(t), ids); err == nil {
			t.Errorf("%v accepted", ids)
		}
	}
	if _, _, err := ReadManifest(); err == nil {
		t.Error("manifest written despite rejection")
	}
}

func TestInstallWithoutRootAndSelfCopy(t *testing.T) {
	env(t)
	src := fakeBinary(t)
	if _, err := Install(src, []string{idA}); err != nil {
		t.Fatal(err)
	}
	bin, _ := BinPath()
	// Re-running install from the installed binary is a no-op copy.
	if _, err := Install(bin, []string{idB}); err != nil {
		t.Fatal(err)
	}
	m, _, err := ReadManifest()
	if err != nil || len(m.AllowedOrigins) != 1 || !strings.Contains(m.AllowedOrigins[0], idB) {
		t.Fatalf("%+v %v", m, err)
	}
}

func TestRelativeHostsDirRejected(t *testing.T) {
	t.Setenv(EnvChromeHostsDir, "rel")
	if _, err := ManifestPath(); err == nil {
		t.Fatal("relative hosts dir accepted")
	}
}
