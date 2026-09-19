package provenance

import (
	"bytes"
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
	cmd := exec.Command("git", "-C", repoRoot, "ls-files", "--others", "--ignored", "--exclude-standard", "-z")
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
