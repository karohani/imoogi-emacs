// Package install places the host binary and the Chrome Native Messaging
// manifest, and removes them again.
package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/karohani/obsidian-clipper/native-host/internal/config"
)

// HostName is the Native Messaging host name the extension connects to.
const HostName = "com.karohani.org_clipper"

// EnvChromeHostsDir overrides the Chrome manifest directory (tests).
const EnvChromeHostsDir = "ORG_CLIPPER_CHROME_HOSTS_DIR"

// Manifest is Chrome's native messaging host manifest.
type Manifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

// ChromeHostsDir is where Chrome looks for per-user host manifests.
func ChromeHostsDir() (string, error) {
	if v := os.Getenv(EnvChromeHostsDir); v != "" {
		if !filepath.IsAbs(v) {
			return "", fmt.Errorf("%s must be an absolute path", EnvChromeHostsDir)
		}
		return filepath.Clean(v), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts"), nil
}

// ManifestPath is the host manifest file.
func ManifestPath() (string, error) {
	d, err := ChromeHostsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, HostName+".json"), nil
}

// BinPath is where the host binary is installed.
func BinPath() (string, error) {
	d, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "bin", "org-clipper-host"), nil
}

// Result describes what Install wrote.
type Result struct {
	Binary   string
	Manifest string
	IDs      []string
}

// Install copies srcBinary to BinPath (0755), writes the manifest allowing
// exactly the given extension IDs, and records the IDs in the config.
func Install(srcBinary string, ids []string) (*Result, error) {
	if len(ids) == 0 {
		return nil, errors.New("at least one --extension-id is required")
	}
	var clean []string
	for _, id := range ids {
		if !config.ExtensionIDRe.MatchString(id) {
			return nil, fmt.Errorf("invalid extension id %q (want 32 characters a-p)", id)
		}
		if !slices.Contains(clean, id) {
			clean = append(clean, id)
		}
	}
	bin, err := BinPath()
	if err != nil {
		return nil, err
	}
	man, err := ManifestPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(bin), 0o700); err != nil {
		return nil, err
	}
	if err := copyBinary(srcBinary, bin); err != nil {
		return nil, fmt.Errorf("copy binary: %w", err)
	}
	m := Manifest{
		Name:        HostName,
		Description: "Org Clipper: saves web clips as Org files",
		Path:        bin,
		Type:        "stdio",
	}
	for _, id := range clean {
		m.AllowedOrigins = append(m.AllowedOrigins, "chrome-extension://"+id+"/")
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(man), 0o755); err != nil {
		return nil, err
	}
	if err := config.WriteFileAtomic(man, append(b, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}
	c, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotConfigured) {
		return nil, fmt.Errorf("existing config unreadable: %w", err)
	}
	c.AllowedExtensionIDs = clean
	if err := config.Save(c); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	return &Result{Binary: bin, Manifest: man, IDs: clean}, nil
}

func copyBinary(src, dst string) error {
	if same, err := sameFile(src, dst); err == nil && same {
		return nil // already running from the installed location
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".org-clipper-host.tmp-*")
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmp.Name())
		}
	}()
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return err
	}
	ok = true
	return nil
}

func sameFile(a, b string) (bool, error) {
	fa, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	fb, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(fa, fb), nil
}

// Uninstall removes the manifest and the installed binary; with purgeConfig
// it also removes config.json. Clips and logs are never touched. It returns
// the paths actually removed.
func Uninstall(purgeConfig bool) ([]string, error) {
	var targets []string
	for _, f := range []func() (string, error){ManifestPath, BinPath} {
		p, err := f()
		if err != nil {
			return nil, err
		}
		targets = append(targets, p)
	}
	if purgeConfig {
		p, err := config.Path()
		if err != nil {
			return nil, err
		}
		targets = append(targets, p)
	}
	var removed []string
	for _, p := range targets {
		err := os.Remove(p)
		switch {
		case err == nil:
			removed = append(removed, p)
		case errors.Is(err, os.ErrNotExist):
		default:
			return removed, err
		}
	}
	return removed, nil
}

// ReadManifest loads the installed manifest, if present.
func ReadManifest() (*Manifest, string, error) {
	p, err := ManifestPath()
	if err != nil {
		return nil, "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, p, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, p, err
	}
	return &m, p, nil
}
