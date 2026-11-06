package fetch

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/karohani/imoogi-emacs/internal/config"
)

// The JVM-hosted servers (jdtls, kotlin-language-server) and clojure-lsp are
// downloaded as published; no JVM is fetched — the target is assumed to
// provide Java, and the launchers written at setup handle its absence.
//
// None of the three archives carries its own LICENSE at a stable path, so the
// license is downloaded from the upstream repository at the locked revision
// (a commit, not a tag, so the bytes cannot move under the pin).

// jdtls milestones are named by version plus a build stamp
// (jdt-language-server-1.61.0-202609031315.tar.gz); the stamp is not
// derivable from the version, so upstream_version carries both.
func fetchJdtls(ctx context.Context, opts Options, repoRoot, staging string, previous *config.ResolvedLock, component config.DesiredComponent, target config.Target, retrievedAt string) (config.LockComponent, []stagedArtifact, error) {
	version := component.UpstreamVersion
	release, _, ok := strings.Cut(version, "-")
	if !ok || release == "" {
		return config.LockComponent{}, nil, fmt.Errorf("jdtls upstream_version %q must be <release>-<build stamp>, as in the milestone file name", version)
	}
	if component.Revision == "" {
		return config.LockComponent{}, nil, errors.New("jdtls revision (upstream commit) is required to pin its license")
	}
	name := fmt.Sprintf("jdt-language-server-%s.tar.gz", version)
	url := fmt.Sprintf("%s/jdtls/milestones/%s/%s", strings.TrimRight(opts.Sources.EclipseDownloads, "/"), release, name)
	rel := fmt.Sprintf("vendor/toolchains/java/jdtls/%s/darwin-arm64/jdt-language-server.tar.gz", version)
	d, artifact, reused, err := fetchDownloadedArtifact(ctx, opts.Client, repoRoot, staging, previous, component, rel, url, 0o644, func(stage string, got digest) error {
		want, err := fetchSingleSHA256(ctx, opts.Client, url+".sha256")
		if err != nil {
			return err
		}
		if got.SHA256 != want {
			return fmt.Errorf("jdtls checksum mismatch for %s: got %s want %s", name, got.SHA256, want)
		}
		return nil
	})
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	artifacts := appendArtifact(nil, artifact)
	license := "vendor/toolchains/licenses/jdtls-LICENSE"
	artifacts, err = stageUpstreamFile(ctx, opts, repoRoot, staging, license, rawURL(opts, "eclipse-jdtls/eclipse.jdt.ls", component.Revision, "LICENSE"), reused.Name != "", artifacts)
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	return lock(component, target, rel, url, reuseRetrievedAt(retrievedAt, reused), d, license, "", "curl "+url, []config.InstallEntry{
		{Path: "lib/jdtls", Mode: "directory"},
		{Path: "bin/jdtls", Mode: "executable"},
	}, config.Probe{Command: "bin/jdtls", Args: []string{"--imoogi-probe"}}), artifacts, nil
}

// kotlin-language-server publishes no checksum for server.zip; like the npm
// tarballs, the digest is recorded on first fetch and enforced afterwards.
func fetchKotlinLanguageServer(ctx context.Context, opts Options, repoRoot, staging string, previous *config.ResolvedLock, component config.DesiredComponent, target config.Target, retrievedAt string) (config.LockComponent, []stagedArtifact, error) {
	version := component.UpstreamVersion
	if component.Revision == "" {
		return config.LockComponent{}, nil, errors.New("kotlin-language-server revision (upstream commit) is required to pin its license")
	}
	url := githubReleaseURL(opts, "fwcd/kotlin-language-server", version, "server.zip")
	rel := fmt.Sprintf("vendor/toolchains/kotlin/kotlin-language-server/%s/darwin-arm64/server.zip", version)
	d, artifact, reused, err := fetchDownloadedArtifact(ctx, opts.Client, repoRoot, staging, previous, component, rel, url, 0o644, func(stage string, _ digest) error {
		return requireZipEntry(stage, "server/bin/kotlin-language-server")
	})
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	artifacts := appendArtifact(nil, artifact)
	archivePath, err := downloadedArtifactPath(repoRoot, rel, artifact)
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	license := "vendor/toolchains/licenses/kotlin-language-server-LICENSE"
	artifacts, err = stageUpstreamFile(ctx, opts, repoRoot, staging, license, rawURL(opts, "fwcd/kotlin-language-server", component.Revision, "LICENSE.txt"), reused.Name != "", artifacts)
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	// The zip's own third-party license report covers the bundled jars.
	notice := fmt.Sprintf("vendor/toolchains/licenses/kotlin-language-server-%s-licenseReport.html", version)
	artifacts, err = stageFromZip(repoRoot, staging, archivePath, notice, "server/licenseReport.html", artifacts)
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	return lock(component, target, rel, url, reuseRetrievedAt(retrievedAt, reused), d, license, notice, "curl -L "+url, []config.InstallEntry{
		{Path: "lib/kotlin-language-server", Mode: "directory"},
		{Path: "bin/kotlin-language-server", Mode: "executable"},
	}, config.Probe{Command: "bin/kotlin-language-server", Args: []string{"--imoogi-probe"}}), artifacts, nil
}

// clojure-lsp ships a GraalVM native binary, so Clojure LSP needs no Java.
// The macOS aarch64 zip has no published .sha256 (the other platforms do);
// the digest is recorded on first fetch and enforced afterwards.
func fetchClojureLsp(ctx context.Context, opts Options, repoRoot, staging string, previous *config.ResolvedLock, component config.DesiredComponent, target config.Target, retrievedAt string) (config.LockComponent, []stagedArtifact, error) {
	version := component.UpstreamVersion
	if component.Revision == "" {
		return config.LockComponent{}, nil, errors.New("clojure-lsp revision (upstream commit) is required to pin its license")
	}
	name := "clojure-lsp-native-macos-aarch64.zip"
	url := githubReleaseURL(opts, "clojure-lsp/clojure-lsp", version, name)
	rel := fmt.Sprintf("vendor/toolchains/clojure/clojure-lsp/%s/darwin-arm64/%s", version, name)
	d, artifact, reused, err := fetchDownloadedArtifact(ctx, opts.Client, repoRoot, staging, previous, component, rel, url, 0o644, func(stage string, _ digest) error {
		return requireZipEntry(stage, "clojure-lsp")
	})
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	artifacts := appendArtifact(nil, artifact)
	license := "vendor/toolchains/licenses/clojure-lsp-LICENSE"
	artifacts, err = stageUpstreamFile(ctx, opts, repoRoot, staging, license, rawURL(opts, "clojure-lsp/clojure-lsp", component.Revision, "LICENSE"), reused.Name != "", artifacts)
	if err != nil {
		return config.LockComponent{}, nil, err
	}
	return lock(component, target, rel, url, reuseRetrievedAt(retrievedAt, reused), d, license, "", "curl -L "+url, []config.InstallEntry{
		{Path: "bin/clojure-lsp", Mode: "executable"},
	}, config.Probe{Command: "bin/clojure-lsp", Args: []string{"--version"}}), artifacts, nil
}

func githubReleaseURL(opts Options, repo, tag, asset string) string {
	return fmt.Sprintf("%s/%s/releases/download/%s/%s", strings.TrimRight(opts.Sources.GitHub, "/"), repo, tag, asset)
}

func rawURL(opts Options, repo, revision, file string) string {
	return fmt.Sprintf("%s/%s/%s/%s", strings.TrimRight(opts.Sources.GitHubRaw, "/"), repo, revision, file)
}

// stageUpstreamFile downloads url into rel. When the artifact was reused
// from the lock and rel already exists, nothing is downloaded, so a re-fetch
// with unchanged pins stays offline.
func stageUpstreamFile(ctx context.Context, opts Options, repoRoot, staging, rel, url string, reused bool, artifacts []stagedArtifact) ([]stagedArtifact, error) {
	finalPath, err := containedPath(repoRoot, rel)
	if err != nil {
		return nil, err
	}
	if reused {
		if exists, err := regularFileExists(finalPath); err != nil {
			return nil, err
		} else if exists {
			return artifacts, nil
		}
	}
	stagePath, err := containedPath(staging, rel)
	if err != nil {
		return nil, err
	}
	if _, err := download(ctx, opts.Client, url, stagePath, 0o644); err != nil {
		return nil, err
	}
	return append(artifacts, stagedArtifact{StagePath: stagePath, FinalPath: finalPath, Mode: 0o644}), nil
}

func fetchSingleSHA256(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s returned %s", url, res.Status)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, 4<<10))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 || len(fields[0]) != 64 {
		return "", fmt.Errorf("%s does not hold a SHA-256 digest", url)
	}
	return strings.ToLower(fields[0]), nil
}

func requireZipEntry(archivePath, name string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, file := range zr.File {
		if file.Name == name && file.Mode().IsRegular() {
			return nil
		}
	}
	return fmt.Errorf("%s is missing %s", filepath.Base(archivePath), name)
}

func stageFromZip(repoRoot, staging, archivePath, finalRel, entry string, artifacts []stagedArtifact) ([]stagedArtifact, error) {
	const maxExtractedMetadataBytes = 4 << 20
	stagePath, err := containedPath(staging, finalRel)
	if err != nil {
		return nil, err
	}
	finalPath, err := containedPath(repoRoot, finalRel)
	if err != nil {
		return nil, err
	}
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	for _, file := range zr.File {
		if file.Name != entry {
			continue
		}
		if !file.Mode().IsRegular() || file.UncompressedSize64 > maxExtractedMetadataBytes {
			return nil, fmt.Errorf("%s in %s is not a small regular file", entry, filepath.Base(archivePath))
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		if err := os.MkdirAll(filepath.Dir(stagePath), 0o755); err != nil {
			return nil, err
		}
		out, err := os.OpenFile(stagePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		_, copyErr := io.Copy(out, io.LimitReader(rc, maxExtractedMetadataBytes))
		closeErr := out.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(stagePath)
			return nil, errors.Join(copyErr, closeErr)
		}
		return append(artifacts, stagedArtifact{StagePath: stagePath, FinalPath: finalPath, Mode: 0o644}), nil
	}
	return nil, fmt.Errorf("%s is missing %s", filepath.Base(archivePath), entry)
}
