package setup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang"
	"github.com/karohani/imoogi-emacs/internal/npmbundle"
)

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
// package-lock.json with the locked node's npm. The result must match the
// locked size and SHA-256; anything else is a hard error and the lock is never
// rewritten. The artifact appears at its final path only after it verified.
//
// @MX:WARN: [AUTO] reaches the npm registry during setup
// @MX:REASON: the only setup step that needs the network; offline it fails with npm's error instead of installing an unverified tree
func buildLocalArtifact(ctx context.Context, opts Options, repoRoot string, lock config.ResolvedLock, c config.LockComponent) error {
	var nodeComponent config.LockComponent
	for _, candidate := range lock.Components {
		if candidate.Name == "node" || candidate.Kind == "node-runtime" {
			nodeComponent = candidate
			break
		}
	}
	if nodeComponent.Name == "" {
		return classify(KindConfigIntegrity, fmt.Errorf("%s is built with the locked node, but the lock has none", c.Name))
	}
	pkgDir := config.ACPPackageDir(c.Name)
	if err := config.ValidateRepoPath(pkgDir); err != nil {
		return classify(KindConfigIntegrity, fmt.Errorf("%s package directory: %w", c.Name, err))
	}
	final := lang.ArtifactPath(repoRoot, c)
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return classify(KindSetup, err)
	}
	temp, err := os.MkdirTemp(filepath.Dir(final), ".build-*")
	if err != nil {
		return classify(KindSetup, err)
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
	if err != nil {
		return classify(KindSetup, fmt.Errorf("build %s: %w", c.Name, err))
	}
	if got.Size != c.Artifact.Size || got.SHA256 != c.Artifact.SHA256 {
		return classify(KindConfigIntegrity, fmt.Errorf(
			"locally built %s is sha256 %s (%d bytes) but the lock pins sha256 %s (%d bytes); the lock was not changed",
			c.Name, got.SHA256, got.Size, c.Artifact.SHA256, c.Artifact.Size))
	}
	if err := os.Rename(built, final); err != nil {
		return classify(KindSetup, fmt.Errorf("publish %s artifact: %w", c.Name, err))
	}
	return nil
}
