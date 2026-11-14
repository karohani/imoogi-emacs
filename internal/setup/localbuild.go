package setup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang"
	"github.com/karohani/imoogi-emacs/internal/npmbundle"
)

// verifyACPLockfiles checks every ACP adapter's committed package-lock.json
// against the lock revision. It runs before setup decides to build, install or
// reuse, so a present artifact or an installed bundle cannot hide a lockfile
// that changed while the revision did not.
func verifyACPLockfiles(repoRoot string, lock config.ResolvedLock) error {
	for _, c := range lock.Components {
		if c.Kind != config.KindACPAdapter {
			continue
		}
		rel := config.ACPPackageDir(c.Name) + "/package-lock.json"
		if err := config.ValidateRepoPath(rel); err != nil {
			return fmt.Errorf("%s package-lock.json: %w", c.Name, err)
		}
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("%s package-lock.json: %w", c.Name, err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != c.Revision {
			return fmt.Errorf("%s: %s sha256 %s does not match the locked revision %s", c.Name, rel, got, c.Revision)
		}
	}
	return nil
}

// missingLocalBuilds lists local-build components whose artifact is absent.
// A present artifact is left to verifyArtifacts like any committed one.
func missingLocalBuilds(repoRoot string, lock config.ResolvedLock) ([]config.LockComponent, error) {
	var out []config.LockComponent
	for _, c := range lock.Components {
		if c.ArtifactPolicy != config.ArtifactPolicyLocalBuild {
			continue
		}
		_, err := os.Lstat(lang.ArtifactPath(repoRoot, c))
		switch {
		case err == nil:
		case errors.Is(err, os.ErrNotExist):
			out = append(out, c)
		default:
			return nil, fmt.Errorf("inspect %s artifact: %w", c.Name, err)
		}
	}
	return out, nil
}

// buildLocalArtifact rebuilds a non-committed artifact from its committed
// package-lock.json with the locked node's npm, and publishes it only after it
// matched the locked size and SHA-256.
//
// An npm install that does not complete (offline, registry error, npm exit
// status) returns a skip reason instead of an error: setup then installs
// everything else without this component. Everything else stays a hard error:
// a built artifact whose digest differs from the lock, committed inputs that
// do not match the pin, a missing or corrupt node. The lock is never rewritten.
//
// @MX:WARN: [AUTO] reaches the npm registry during setup
// @MX:REASON: the only setup step that needs the network; offline it skips this one component rather than failing every language server
func buildLocalArtifact(ctx context.Context, opts Options, repoRoot string, lock config.ResolvedLock, c config.LockComponent) (skip error, err error) {
	var nodeComponent config.LockComponent
	for _, candidate := range lock.Components {
		if candidate.Name == "node" || candidate.Kind == "node-runtime" {
			nodeComponent = candidate
			break
		}
	}
	if nodeComponent.Name == "" {
		return nil, classify(KindConfigIntegrity, fmt.Errorf("%s is built with the locked node, but the lock has none", c.Name))
	}
	pkgDir := config.ACPPackageDir(c.Name)
	if err := config.ValidateRepoPath(pkgDir); err != nil {
		return nil, classify(KindConfigIntegrity, fmt.Errorf("%s package directory: %w", c.Name, err))
	}
	final := lang.ArtifactPath(repoRoot, c)
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return nil, classify(KindSetup, err)
	}
	temp, err := os.MkdirTemp(filepath.Dir(final), ".build-*")
	if err != nil {
		return nil, classify(KindSetup, err)
	}
	defer os.RemoveAll(temp)

	fmt.Fprintf(opts.Stdout, "setup building %s %s from %s (artifact is built locally, not committed)\n", c.Name, c.UpstreamVersion, pkgDir+"/package-lock.json")
	built := filepath.Join(temp, filepath.Base(final))
	got, err := npmbundle.Build(ctx, npmbundle.BuildOptions{
		PackageDir:  filepath.Join(repoRoot, filepath.FromSlash(pkgDir)),
		LockSHA256:  c.Revision,
		Package:     config.NPMPackageName(c.Source),
		Version:     c.UpstreamVersion,
		NodeArchive: lang.ArtifactPath(repoRoot, nodeComponent),
		NodeDigest:  artifact.Digest{Size: nodeComponent.Artifact.Size, SHA256: nodeComponent.Artifact.SHA256},
		Output:      built,
		Run:         opts.NPMRunner,
	})
	switch {
	case err == nil:
	case errors.Is(err, npmbundle.ErrNPM):
		return err, nil
	case errors.Is(err, npmbundle.ErrInputs):
		return nil, classify(KindConfigIntegrity, fmt.Errorf("build %s: %w", c.Name, err))
	default:
		return nil, classify(KindSetup, fmt.Errorf("build %s: %w", c.Name, err))
	}
	if got.Size != c.Artifact.Size || got.SHA256 != c.Artifact.SHA256 {
		return nil, classify(KindConfigIntegrity, fmt.Errorf(
			"locally built %s is sha256 %s (%d bytes) but the lock pins sha256 %s (%d bytes); the lock was not changed",
			c.Name, got.SHA256, got.Size, c.Artifact.SHA256, c.Artifact.Size))
	}
	if err := os.Rename(built, final); err != nil {
		return nil, classify(KindSetup, fmt.Errorf("publish %s artifact: %w", c.Name, err))
	}
	return nil, nil
}

// warnSkipped tells the operator which component is missing and how to get it.
func warnSkipped(w io.Writer, c config.LockComponent, reason error) {
	fmt.Fprintf(w, "warning: skipped %s %s: its bundle is built locally and the build did not complete (%s).\n", c.Name, c.UpstreamVersion, summarizeNpmError(reason))
	fmt.Fprintf(w, "warning: every other toolchain component is installed. To add %s, run `make toolchain-setup` again once the npm registry is reachable, or copy a bundle built elsewhere to %s.\n", c.Name, c.Artifact.Path)
}

// summarizeNpmError keeps the first line of npm's failure and the line naming
// the request that failed, dropping stack frames and the path of a debug log
// that lived in the (already removed) build directory.
func summarizeNpmError(err error) string {
	lines := strings.Split(strings.TrimSpace(err.Error()), "\n")
	summary := strings.TrimSpace(lines[0])
	for _, line := range lines[1:] {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "npm error"))
		if strings.Contains(line, "request to ") || strings.Contains(line, "reason:") {
			return summary + "; " + line
		}
	}
	return summary
}

// withoutComponents returns lock without the named components.
func withoutComponents(lock config.ResolvedLock, names map[string]bool) config.ResolvedLock {
	if len(names) == 0 {
		return lock
	}
	out := lock
	out.Components = nil
	for _, c := range lock.Components {
		if !names[c.Name] {
			out.Components = append(out.Components, c)
		}
	}
	return out
}

func subsetOf(names []string, set map[string]bool) bool {
	for _, name := range names {
		if !set[name] {
			return false
		}
	}
	return true
}

func sortedNames(set map[string]bool) []string {
	if len(set) == 0 {
		return nil
	}
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func skippedNote(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return " (without " + strings.Join(names, ", ") + ")"
}

// installedDir names the directory of bundle that setup treats as installed:
// the one .local/bin points into when it is an existing directory of this
// bundle, otherwise the bundle id itself.
func installedDir(localRoot, bundle string) string {
	target, err := os.Readlink(filepath.Join(localRoot, "bin"))
	if err != nil {
		return bundle
	}
	dir, ok := config.ActiveBundleDir(target, bundle)
	if !ok {
		return bundle
	}
	if _, err := os.Lstat(filepath.Join(localRoot, "toolchains", dir)); err != nil {
		return bundle
	}
	return dir
}

// publishReplacement publishes the complete staging tree for an installed
// bundle that skipped a component. It goes into a directory of its own, named
// after its content, so the installed tree stays whole and active until
// activate switches .local/bin to the new one in a single rename.
//
// @MX:NOTE: [AUTO] readers of .local/bin always see a whole bundle: new tree first, then one atomic link rename, then the old tree is retired
func publishReplacement(staging, toolchains, bundle, installed, treeSHA256 string, fp Failpoints) (string, error) {
	name := config.ReplacementBundleDir(bundle, treeSHA256)
	if name == installed {
		return "", fmt.Errorf("replacement for bundle %s has the installed tree's content", bundle)
	}
	// Left by a run interrupted before it switched .local/bin; never active.
	if err := retireDir(toolchains, name, fp); err != nil {
		return "", err
	}
	if err := os.Rename(staging, filepath.Join(toolchains, name)); err != nil {
		return "", fmt.Errorf("publish bundle: %w", err)
	}
	return name, fp.step("replacement-published")
}

// retireInactive removes every directory of bundle except active. Run only
// after .local/bin points at active.
func retireInactive(toolchains, bundle, active string, fp Failpoints) error {
	entries, err := os.ReadDir(toolchains)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() != active && config.IsBundleDir(e.Name(), bundle) {
			if err := retireDir(toolchains, e.Name(), fp); err != nil {
				return err
			}
		}
	}
	return nil
}

// retireDir renames a bundle directory to a dot name before deleting it, so a
// half-deleted tree never sits where it could be taken for an installed bundle.
func retireDir(toolchains, name string, fp Failpoints) error {
	path := filepath.Join(toolchains, name)
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	retired := filepath.Join(toolchains, fmt.Sprintf(".retired-%s-%d", name, os.Getpid()))
	if err := os.RemoveAll(retired); err != nil {
		return err
	}
	if err := os.Rename(path, retired); err != nil {
		return fmt.Errorf("retire bundle directory %s: %w", name, err)
	}
	if err := fp.step("bundle-retired"); err != nil {
		return err
	}
	return os.RemoveAll(retired)
}

// sweepDebris removes what an interrupted setup can leave behind: staging and
// probe trees, temporary activation links, retired bundle directories, and the
// ".replaced-*" trees of earlier setup versions. It runs under the setup lock,
// when no other setup can own them, and is best effort: debris never blocks.
func sweepDebris(localRoot string) {
	for _, pattern := range []string{
		filepath.Join(localRoot, ".staging-*"),
		filepath.Join(localRoot, ".probe-*"),
		filepath.Join(localRoot, ".bin-*.tmp"),
		filepath.Join(localRoot, "toolchains", ".retired-*"),
		filepath.Join(localRoot, "toolchains", ".replaced-*"),
	} {
		matches, _ := filepath.Glob(pattern)
		for _, path := range matches {
			_ = os.RemoveAll(path)
		}
	}
}
