package provenance

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitIgnored returns the repository-relative paths of untracked files that
// git ignores below repoRoot. Such files are local noise (Finder metadata,
// regenerable caches) rather than vendored content, so neither the generator
// nor the verifier treats them as part of a component. When git is missing
// or repoRoot is not a repository the set is empty and behaviour is
// unchanged.
func gitIgnored(repoRoot string) map[string]struct{} {
	ignored := map[string]struct{}{}
	cmd := gitCommand(repoRoot, "ls-files", "--others", "--ignored", "--exclude-standard", "-z")
	out, err := cmd.Output()
	if err != nil {
		return ignored
	}
	for _, entry := range bytes.Split(out, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		ignored[filepath.ToSlash(strings.TrimSuffix(string(entry), "/"))] = struct{}{}
	}
	return ignored
}

// gitRepoEnv lists the variables git hooks export to pin git to the hook's
// repository. Inherited by a child git, they override its working directory,
// so a command meant for dir would read or write the hook's repository instead.
var gitRepoEnv = []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY", "GIT_PREFIX"}

// gitCommand returns a git command that runs in dir and resolves its
// repository from dir alone, even when called from inside a git hook.
func gitCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		keep := true
		for _, drop := range gitRepoEnv {
			if name == drop {
				keep = false
				break
			}
		}
		if keep {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	return cmd
}
