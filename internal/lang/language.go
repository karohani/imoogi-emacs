package lang

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karohani/imoogi-emacs/internal/config"
)

const ExecutableMode os.FileMode = 0o755

type Probe struct {
	Command string
	Args    []string
}

type Provider interface {
	Materialize(repoRoot, stagingRoot string) ([]Probe, error)
}

func ArtifactPath(repoRoot string, component config.LockComponent) string {
	return filepath.Join(repoRoot, filepath.FromSlash(component.Artifact.Path))
}

func AbsoluteProbe(stagingRoot string, probe config.Probe) (Probe, error) {
	command, err := ContainedStagePath(stagingRoot, probe.Command)
	if err != nil {
		return Probe{}, fmt.Errorf("probe command: %w", err)
	}
	return Probe{Command: command, Args: append([]string(nil), probe.Args...)}, nil
}

func ContainedStagePath(stagingRoot, rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("empty relative path")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("%q is absolute", rel)
	}
	if strings.Contains(rel, `\`) {
		return "", fmt.Errorf("%q contains backslash", rel)
	}
	if strings.Contains(rel, ":") {
		return "", fmt.Errorf("%q contains drive or scheme separator", rel)
	}
	if filepath.Clean(rel) != filepath.FromSlash(rel) {
		return "", fmt.Errorf("%q is not clean", rel)
	}
	rootAbs, err := filepath.Abs(stagingRoot)
	if err != nil {
		return "", fmt.Errorf("resolve staging root: %w", err)
	}
	target := filepath.Join(rootAbs, filepath.FromSlash(rel))
	relToRoot, err := filepath.Rel(rootAbs, target)
	if err != nil {
		return "", fmt.Errorf("check staging containment: %w", err)
	}
	if relToRoot == "." || relToRoot == ".." || filepath.IsAbs(relToRoot) || hasDotDotPrefix(relToRoot) {
		return "", fmt.Errorf("%q escapes staging root", rel)
	}
	return target, nil
}

func EnsureExecutable(file string) error {
	info, err := os.Lstat(file)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", file)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", file)
	}
	return os.Chmod(file, ExecutableMode)
}

func WriteExecutable(file string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(file, data, ExecutableMode); err != nil {
		return err
	}
	return os.Chmod(file, ExecutableMode)
}

func hasDotDotPrefix(rel string) bool {
	return len(rel) > 3 && rel[:3] == ".."+string(os.PathSeparator)
}
