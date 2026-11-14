package doctor

import (
	"bytes"
	"debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// SystemEnv inspects a real checkout with go and git.
type SystemEnv struct {
	RootDir, HomeDir string
}

// NewSystemEnv locates the repository that contains dir.
func NewSystemEnv(dir string) (SystemEnv, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return SystemEnv{}, fmt.Errorf("not inside the imoogi-emacs repository: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return SystemEnv{}, err
	}
	return SystemEnv{RootDir: strings.TrimSpace(string(out)), HomeDir: home}, nil
}

func (e SystemEnv) Root() string { return e.RootDir }
func (e SystemEnv) Home() string { return e.HomeDir }

// The doctor itself runs with `go run`, so it is never a bin/ tool.
const self = "imoogi-doctor"

func (e SystemEnv) Tools() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(e.RootDir, "cmd"))
	if err != nil {
		return nil, err
	}
	var tools []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != self {
			tools = append(tools, entry.Name())
		}
	}
	return tools, nil
}

func (e SystemEnv) BinEntries() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(e.RootDir, "bin"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

var toolchainRE = regexp.MustCompile(`(?m)^toolchain\s+(go\S+)`)
var goLineRE = regexp.MustCompile(`(?m)^go\s+(\S+)`)

// RequiredGo is the toolchain go.work pins (workspace mode reads go.work), then go.mod.
func (e SystemEnv) RequiredGo() (string, error) {
	for _, file := range []string{"go.work", "go.mod"} {
		data, err := os.ReadFile(filepath.Join(e.RootDir, file))
		if err != nil {
			continue
		}
		if m := toolchainRE.FindSubmatch(data); m != nil {
			return string(m[1]), nil
		}
		if m := goLineRE.FindSubmatch(data); m != nil {
			return "go" + string(m[1]), nil
		}
	}
	return "", fmt.Errorf("no go or toolchain line in go.work or go.mod")
}

func (e SystemEnv) ReadBuildInfo(path string) (BuildInfo, error) {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return BuildInfo{}, err
	}
	out := BuildInfo{GoVersion: info.GoVersion, Main: info.Path}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			out.Revision = s.Value
		case "vcs.modified":
			out.Modified = s.Value == "true"
		}
	}
	return out, nil
}

func (e SystemEnv) Deps(tool string) ([]string, error) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{if .Module}}{{if .Module.Main}}{{.Dir}}{{end}}{{end}}", "./cmd/"+tool)
	cmd.Dir = e.RootDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list %s: %v %s", tool, err, strings.TrimSpace(stderr.String()))
	}
	var dirs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if rel, err := filepath.Rel(e.RootDir, line); err == nil && line != "" {
			dirs = append(dirs, rel)
		}
	}
	return dirs, nil
}

// ChangedSince lists tracked files that differ from revision in the working tree,
// plus new untracked files, under paths.
func (e SystemEnv) ChangedSince(revision string, paths []string) ([]string, error) {
	diff := exec.Command("git", append([]string{"-C", e.RootDir, "diff", "--name-only", revision, "--"}, paths...)...)
	var stderr bytes.Buffer
	diff.Stderr = &stderr
	out, err := diff.Output()
	if err != nil {
		return nil, fmt.Errorf("compare with %s: %s", short(revision), strings.TrimSpace(stderr.String()))
	}
	untracked, err := exec.Command("git", append([]string{"-C", e.RootDir, "ls-files", "--others", "--exclude-standard", "--"}, paths...)...).Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(string(out)+string(untracked), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func (e SystemEnv) Readlink(path string) (string, error) { return os.Readlink(path) }

func (e SystemEnv) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
