// Package node installs the locked Node.js distribution into a setup staging
// bundle. Several providers run on the same node (TypeScript and Python
// language servers, the ACP adapters); whichever runs first extracts it and the
// rest reuse that copy.
package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang"
)

// StageDir is where the node distribution lives inside a bundle.
const StageDir = "lib/node"

// DistName is the directory the official darwin-arm64 archive unpacks to.
func DistName(component config.LockComponent) string {
	return "node-" + component.UpstreamVersion + "-darwin-arm64"
}

// Ensure extracts the node artifact into stagingRoot/lib/node unless an
// earlier provider already did, and returns the distribution directory. The
// artifact is verified against the lock digest on extraction; a staged node of
// another version is an error, never silently reused.
//
// @MX:ANCHOR: [AUTO] single node extraction point shared by the node-based providers
// @MX:REASON: typescript and acp providers both call it in one staging tree; a second extraction would fail on the non-empty destination
func Ensure(repoRoot, stagingRoot string, component config.LockComponent) (string, error) {
	destination, err := lang.ContainedStagePath(stagingRoot, StageDir)
	if err != nil {
		return "", err
	}
	want := DistName(component)
	if entries, err := os.ReadDir(destination); err == nil {
		if len(entries) != 1 || !entries[0].IsDir() || entries[0].Name() != want {
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				names = append(names, e.Name())
			}
			return "", fmt.Errorf("staged node is %s, want %s", strings.Join(names, ", "), want)
		}
		dist := filepath.Join(destination, want)
		if err := lang.EnsureExecutable(filepath.Join(dist, "bin", "node")); err != nil {
			return "", fmt.Errorf("staged node executable: %w", err)
		}
		return dist, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", err
	}
	if _, err := artifact.ExtractTarGzipWithOptions(
		lang.ArtifactPath(repoRoot, component),
		destination,
		artifact.Digest{Size: component.Artifact.Size, SHA256: component.Artifact.SHA256},
		artifact.Limits{
			MaxEntries:     20_000,
			MaxFileBytes:   512 << 20,
			MaxOutputBytes: 1 << 30,
		},
		artifact.ExtractOptions{AllowRelativeSymlinks: true},
	); err != nil {
		return "", fmt.Errorf("extract node artifact: %w", err)
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		return "", err
	}
	if len(entries) != 1 || !entries[0].IsDir() || entries[0].Name() != want {
		return "", fmt.Errorf("node archive must contain the one distribution directory %s", want)
	}
	dist := filepath.Join(destination, entries[0].Name())
	if err := lang.EnsureExecutable(filepath.Join(dist, "bin", "node")); err != nil {
		return "", fmt.Errorf("node executable: %w", err)
	}
	return dist, nil
}
