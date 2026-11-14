package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang/acp"
	"github.com/karohani/imoogi-emacs/internal/npmbundle"
)

// fetchACPAdapter builds an adapter bundle from its committed package-lock.json
// with the npm inside the node artifact this run locked (nodeArchive is that
// artifact's staged or published file). A committed artifact that still
// matches the previous lock is reused. A local-build artifact is rebuilt when
// absent, and DetectDrift then requires the same bytes as before.
func fetchACPAdapter(ctx context.Context, opts Options, repoRoot, staging string, previous *config.ResolvedLock, component config.DesiredComponent, nodeLock config.LockComponent, nodeArchive string) (config.LockComponent, []stagedArtifact, error) {
	target := config.Target{OS: config.TargetOS, Arch: config.TargetArch}
	pkgDir := config.ACPPackageDir(component.Name)
	lockfile := pkgDir + "/package-lock.json"
	lockPath, err := containedPath(repoRoot, lockfile)
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		return config.LockComponent{}, nil, fmt.Errorf("read committed %s: %w", lockfile, err)
	}
	sum := sha256.Sum256(lockData)
	if got := hex.EncodeToString(sum[:]); got != component.Revision {
		return config.LockComponent{}, nil, fmt.Errorf("%s sha256 is %s but toolchains.json revision is %s; set the revision to the new lockfile digest", lockfile, got, component.Revision)
	}
	if _, ok := acp.Lookup(component.Name); !ok {
		return config.LockComponent{}, nil, fmt.Errorf("no ACP launcher for adapter %q", component.Name)
	}

	version := component.UpstreamVersion
	pkg := config.NPMPackageName(component.Source)
	rel := fmt.Sprintf("vendor/toolchains/acp/%s/%s/darwin-arm64/bundle.tgz", component.Name, version)
	licenseRel := fmt.Sprintf("vendor/toolchains/licenses/%s-LICENSE", component.Name)
	sourceURL := npmTarballURL(opts.Sources.NPMRegistry, pkg, version)
	install := []config.InstallEntry{
		{Path: "bin/" + component.Name, Mode: "executable"},
		{Path: acp.InstallDir(component.Name), Mode: "directory"},
	}
	probe := config.Probe{Command: "bin/" + component.Name}
	command := npmbundle.Command(lockfile)
	finalPath, err := containedPath(repoRoot, rel)
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	retrievedAt := opts.Now().UTC().Format(time.RFC3339)
	previousComponent, matched := matchingPrevious(previous, component, rel)
	if matched {
		if d, err := fileDigest(finalPath); err == nil && d.Size == previousComponent.Artifact.Size && d.SHA256 == previousComponent.Artifact.SHA256 {
			if err := validateLicenseFiles(repoRoot, previousComponent.License); err != nil {
				return config.LockComponent{}, nil, err
			}
			return lock(component, target, rel, sourceURL, previousComponent.Artifact.RetrievedAt, d, licenseRel, "", command, install, probe), nil, nil
		}
		retrievedAt = previousComponent.Artifact.RetrievedAt
	}

	stagePath, err := containedPath(staging, rel)
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	licenseStage, err := containedPath(staging, licenseRel)
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	licenseFinal, err := containedPath(repoRoot, licenseRel)
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	built, err := npmbundle.Build(ctx, npmbundle.BuildOptions{
		PackageDir:    filepath.Dir(lockPath),
		LockSHA256:    component.Revision,
		Package:       pkg,
		Version:       version,
		NodeArchive:   nodeArchive,
		NodeDigest:    artifact.Digest{Size: nodeLock.Artifact.Size, SHA256: nodeLock.Artifact.SHA256},
		Output:        stagePath,
		LicenseOutput: licenseStage,
		Run:           opts.NPMRunner,
	})
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	fmt.Fprintf(opts.Stdout, "fetch built %s %s (%d bytes, sha256 %s)\n", component.Name, version, built.Size, built.SHA256)
	return lock(component, target, rel, sourceURL, retrievedAt, digest{Size: built.Size, SHA256: built.SHA256}, licenseRel, "", command, install, probe),
		[]stagedArtifact{
			{StagePath: stagePath, FinalPath: finalPath, Mode: 0o644},
			{StagePath: licenseStage, FinalPath: licenseFinal, Mode: 0o644},
		}, nil
}
