// Package config reads and writes the host configuration. The file is only
// ever written by CLI subcommands; nothing a message says can change it.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// Environment overrides (used by tests; never set them in production).
const (
	EnvConfigDir = "ORG_CLIPPER_CONFIG_DIR"
	EnvLogDir    = "ORG_CLIPPER_LOG_DIR"
)

const (
	fileName = "config.json"
	maxSize  = 64 << 10
)

// ErrNotConfigured means no config file exists, or it has no root.
var ErrNotConfigured = errors.New("not configured")

// ExtensionIDRe matches a Chrome extension ID.
var ExtensionIDRe = regexp.MustCompile(`^[a-p]{32}$`)

// Config is the persisted host configuration.
type Config struct {
	Root                string   `json:"root"`
	AllowedExtensionIDs []string `json:"allowedExtensionIds"`
}

func home() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(h) {
		return "", fmt.Errorf("cannot determine home directory")
	}
	return h, nil
}

func absEnv(name, fallback string) (string, error) {
	if v := os.Getenv(name); v != "" {
		if !filepath.IsAbs(v) {
			return "", fmt.Errorf("%s must be an absolute path", name)
		}
		return filepath.Clean(v), nil
	}
	return fallback, nil
}

// Dir is the configuration directory.
func Dir() (string, error) {
	h, err := home()
	if err != nil && os.Getenv(EnvConfigDir) == "" {
		return "", err
	}
	return absEnv(EnvConfigDir, filepath.Join(h, "Library", "Application Support", "org-clipper"))
}

// LogDir is the log directory.
func LogDir() (string, error) {
	h, err := home()
	if err != nil && os.Getenv(EnvLogDir) == "" {
		return "", err
	}
	return absEnv(EnvLogDir, filepath.Join(h, "Library", "Logs", "org-clipper"))
}

// Path is the configuration file path.
func Path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, fileName), nil
}

// ValidateRoot checks a candidate save root: absolute, an existing directory.
// It returns the cleaned path. It never guesses or creates paths.
func ValidateRoot(p string) (string, error) {
	if p == "" || !filepath.IsAbs(p) {
		return "", fmt.Errorf("root must be an absolute path")
	}
	p = filepath.Clean(p)
	fi, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("root does not exist or is not accessible")
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("root is not a directory")
	}
	return p, nil
}

// Load reads the configuration. A missing file returns ErrNotConfigured with
// an empty Config. A group- or world-writable file is refused: whoever can
// write it decides where clips are written.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, ErrNotConfigured
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Mode().Perm()&0o022 != 0 {
		return nil, fmt.Errorf("config file is group/world writable; refusing to use it")
	}
	b, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxSize {
		return nil, fmt.Errorf("config file too large")
	}
	var c Config
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("config file is malformed: %w", err)
	}
	for _, id := range c.AllowedExtensionIDs {
		if !ExtensionIDRe.MatchString(id) {
			return nil, fmt.Errorf("config file has an invalid extension id")
		}
	}
	if c.Root == "" {
		return &c, ErrNotConfigured
	}
	if !filepath.IsAbs(c.Root) || filepath.Clean(c.Root) != c.Root {
		return nil, fmt.Errorf("config root must be an absolute, clean path")
	}
	return &c, nil
}

// Save writes the configuration atomically (dir 0700, file 0600).
func Save(c *Config) error {
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(d, 0o700); err != nil {
		return err
	}
	if c.AllowedExtensionIDs == nil {
		c.AllowedExtensionIDs = []string{}
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(filepath.Join(d, fileName), append(b, '\n'), 0o600)
}

// WriteFileAtomic writes data to a temp file in the target directory, fsyncs
// it, sets perm, and renames it over path.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(perm); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}
