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

// replaceBundle swaps an installed bundle that skipped a component for the
// complete staging tree. The old tree is moved aside first and put back if the
// swap fails, then removed.
func replaceBundle(staging, bundleRoot string) error {
	aside := filepath.Join(filepath.Dir(bundleRoot), fmt.Sprintf(".replaced-%s-%d", filepath.Base(bundleRoot), os.Getpid()))
	if err := os.RemoveAll(aside); err != nil {
		return err
	}
	if err := os.Rename(bundleRoot, aside); err != nil {
		return fmt.Errorf("move incomplete bundle aside: %w", err)
	}
	if err := os.Rename(staging, bundleRoot); err != nil {
		if backErr := os.Rename(aside, bundleRoot); backErr != nil {
			return fmt.Errorf("publish bundle: %w (putting the previous bundle back also failed: %v)", err, backErr)
		}
		return fmt.Errorf("publish bundle: %w", err)
	}
	return os.RemoveAll(aside)
}
