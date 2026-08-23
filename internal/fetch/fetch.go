package fetch

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/karohani/imoogi-emacs/internal/activation"
	"github.com/karohani/imoogi-emacs/internal/config"
)

const generatedBy = "imoogi-toolchain fetch"

type Options struct {
	Workdir string
	Stdout  io.Writer
	Stderr  io.Writer
	Client  *http.Client
	Now     func() time.Time
	Sources Sources
	Go      string
}

type Sources struct {
	NodeDistBase string
	NPMRegistry  string
	GoProxy      string
}

func Run(ctx context.Context, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Workdir == "" {
		opts.Workdir = "."
	}
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.Client == nil {
		opts.Client = http.DefaultClient
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Sources.NodeDistBase == "" {
		opts.Sources.NodeDistBase = "https://nodejs.org/dist"
	}
	if opts.Sources.NPMRegistry == "" {
		opts.Sources.NPMRegistry = "https://registry.npmjs.org"
	}
	if opts.Sources.GoProxy == "" {
		opts.Sources.GoProxy = "https://proxy.golang.org"
	}
	if opts.Go == "" {
		opts.Go = "go"
	}
	repoRoot, err := findRoot(opts.Workdir)
	if err != nil {
		return err
	}
	desired, err := config.LoadDesired(filepath.Join(repoRoot, "toolchains.json"))
	if err != nil {
		return err
	}
	if runtime.GOOS != config.TargetOS || runtime.GOARCH != config.TargetArch {
		return fmt.Errorf("fetch must run on %s/%s build machine, got %s/%s", config.TargetOS, config.TargetArch, runtime.GOOS, runtime.GOARCH)
	}

	resolved := config.ResolvedLock{
		Schema:      config.LockSchema,
		CLI:         desired.CLI,
		Bundle:      desired.Bundle,
		Target:      desired.Target,
		GeneratedBy: generatedBy,
	}
	previous, err := config.LoadLock(filepath.Join(repoRoot, "toolchains.lock.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load existing lock: %w", err)
	}
	localDir := filepath.Join(repoRoot, ".local")
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("create .local: %w", err)
	}
	staging, err := os.MkdirTemp(localDir, ".fetch-staging-*")
	if err != nil {
		return fmt.Errorf("create fetch staging: %w", err)
	}
	defer os.RemoveAll(staging)
	var staged []stagedArtifact
	for _, component := range desired.Components {
		locked, artifacts, err := fetchComponent(ctx, opts, repoRoot, staging, previous, component)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", component.Name, err)
		}
		resolved.Components = append(resolved.Components, locked)
		staged = append(staged, artifacts...)
	}
	bootstrapArtifacts, err := buildBootstrap(ctx, opts, repoRoot, staging, desired.CLI)
	if err != nil {
		return fmt.Errorf("fetch bootstrap CLI: %w", err)
	}
	staged = append(staged, bootstrapArtifacts...)
	if err := desired.ValidateLock(resolved); err != nil {
		return err
	}
	if previous != nil {
		if err := config.DetectDrift(*previous, resolved); err != nil {
			return err
		}
	}
	for _, artifact := range staged {
		if err := publishArtifact(artifact); err != nil {
			return err
		}
	}

	lockPath := filepath.Join(repoRoot, "toolchains.lock.json")
	tempPath := lockPath + ".tmp"
	out, err := os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	writeErr := config.WriteLock(out, resolved)
	closeErr := out.Close()
	if writeErr != nil {
		_ = os.Remove(tempPath)
		return writeErr
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return closeErr
	}
	if err := os.Rename(tempPath, lockPath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	fmt.Fprintf(opts.Stdout, "fetch wrote %s\n", filepath.Base(lockPath))
	return nil
}

func findRoot(start string) (string, error) {
	if start == "" {
		start = "."
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve workdir: %w", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("inspect workdir: %w", err)
	}
	if !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		if info, err := os.Lstat(filepath.Join(dir, "toolchains.json")); err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("toolchains.json not found from %s", start)
		}
		dir = parent
	}
}

func containedPath(root, rel string) (string, error) {
	if err := config.ValidateRepoPath(rel); err != nil {
		return "", fmt.Errorf("generated path %q: %w", rel, err)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve containment root: %w", err)
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	contained, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("check generated path %q: %w", rel, err)
	}
	contained = filepath.ToSlash(contained)
	if contained == ".." || strings.HasPrefix(contained, "../") {
		return "", fmt.Errorf("generated path %q escapes %s", rel, root)
	}
	return path, nil
}

type stagedArtifact struct {
	StagePath string
	FinalPath string
	Mode      os.FileMode
}

func fetchComponent(ctx context.Context, opts Options, repoRoot, staging string, previous *config.ResolvedLock, component config.DesiredComponent) (config.LockComponent, []stagedArtifact, error) {
	target := config.Target{OS: config.TargetOS, Arch: config.TargetArch}
	retrievedAt := opts.Now().UTC().Format(time.RFC3339)
	switch component.Name {
	case "node":
		version := component.UpstreamVersion
		name := fmt.Sprintf("node-%s-darwin-arm64.tar.gz", version)
		base := strings.TrimRight(opts.Sources.NodeDistBase, "/")
		url := fmt.Sprintf("%s/%s/%s", base, version, name)
		shasumsURL := fmt.Sprintf("%s/%s/SHASUMS256.txt", base, version)
		rel := fmt.Sprintf("vendor/toolchains/node/%s/darwin-arm64/%s", version, name)
		digest, artifact, reused, err := fetchDownloadedArtifact(ctx, opts.Client, repoRoot, staging, previous, component, rel, url, 0o644, func(stage string, got digest) error {
			want, err := fetchNodeSHA256(ctx, opts.Client, shasumsURL, name)
			if err != nil {
				return err
			}
			if got.SHA256 != want {
				return fmt.Errorf("node checksum mismatch for %s: got %s want %s", name, got.SHA256, want)
			}
			return nil
		})
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		retrievedAt = reuseRetrievedAt(retrievedAt, reused)
		artifacts := appendArtifact(nil, artifact)
		archivePath, err := downloadedArtifactPath(repoRoot, rel, artifact)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		nodeLicense := fmt.Sprintf("vendor/toolchains/licenses/node-%s-LICENSE", version)
		artifacts, err = stageFromTarGz(repoRoot, staging, archivePath, nodeLicense, []string{fmt.Sprintf("node-%s-darwin-arm64/LICENSE", version), "LICENSE"}, artifacts)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		return lock(component, target, rel, url, retrievedAt, digest, nodeLicense, "", "curl "+url, []config.InstallEntry{
			{Path: "lib/node", Mode: "directory"},
			{Path: "bin/node", Mode: "executable"},
		}, config.Probe{Command: "bin/node", Args: []string{"--version"}}), artifacts, nil
	case "typescript":
		version := component.UpstreamVersion
		url := npmTarballURL(opts.Sources.NPMRegistry, "typescript", version)
		rel := fmt.Sprintf("vendor/toolchains/typescript/typescript/%s/darwin-arm64/typescript.tgz", version)
		digest, artifact, reused, err := fetchDownloadedArtifact(ctx, opts.Client, repoRoot, staging, previous, component, rel, url, 0o644, func(stage string, got digest) error {
			if ok, err := tarGzHasPath(stage, "package/lib/tsserver.js"); err != nil {
				return err
			} else if !ok {
				return errors.New("typescript tarball is missing package/lib/tsserver.js required by typescript-language-server 6")
			}
			return nil
		})
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		retrievedAt = reuseRetrievedAt(retrievedAt, reused)
		artifacts := appendArtifact(nil, artifact)
		archivePath, err := downloadedArtifactPath(repoRoot, rel, artifact)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		tsLicense := fmt.Sprintf("vendor/toolchains/licenses/typescript-%s-LICENSE.txt", version)
		artifacts, err = stageFromTarGz(repoRoot, staging, archivePath, tsLicense, []string{"package/LICENSE.txt", "package/LICENSE"}, artifacts)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		tsNotice := fmt.Sprintf("vendor/toolchains/licenses/typescript-%s-ThirdPartyNoticeText.txt", version)
		artifacts, err = stageFromTarGz(repoRoot, staging, archivePath, tsNotice, []string{"package/ThirdPartyNoticeText.txt"}, artifacts)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		return lock(component, target, rel, url, retrievedAt, digest, tsLicense, tsNotice, "curl "+url, []config.InstallEntry{
			{Path: "lib/node_modules/typescript", Mode: "directory"},
			{Path: "bin/tsc", Mode: "executable"},
		}, config.Probe{Command: "bin/tsc", Args: []string{"--version"}}), artifacts, nil
	case "typescript-language-server":
		version := component.UpstreamVersion
		url := npmTarballURL(opts.Sources.NPMRegistry, "typescript-language-server", version)
		rel := fmt.Sprintf("vendor/toolchains/typescript/typescript-language-server/%s/darwin-arm64/server.tgz", version)
		digest, artifact, reused, err := fetchDownloadedArtifact(ctx, opts.Client, repoRoot, staging, previous, component, rel, url, 0o644, nil)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		retrievedAt = reuseRetrievedAt(retrievedAt, reused)
		artifacts := appendArtifact(nil, artifact)
		archivePath, err := downloadedArtifactPath(repoRoot, rel, artifact)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		artifacts, err = stageFromTarGz(repoRoot, staging, archivePath, "vendor/toolchains/licenses/typescript-language-server-LICENSE", []string{"package/LICENSE", "package/LICENSE.txt"}, artifacts)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		return lock(component, target, rel, url, retrievedAt, digest, "vendor/toolchains/licenses/typescript-language-server-LICENSE", "", "curl "+url, []config.InstallEntry{
			{Path: "lib/node_modules/typescript-language-server", Mode: "directory"},
			{Path: "bin/typescript-language-server", Mode: "executable"},
		}, config.Probe{Command: "bin/typescript-language-server", Args: []string{"--help"}}), artifacts, nil
	case "basedpyright":
		// npm 배포물이라 typescript-language-server 와 같은 경로를 탄다.
		// 실행은 node 가 하므로 별도 런타임을 받지 않는다.
		version := component.UpstreamVersion
		url := npmTarballURL(opts.Sources.NPMRegistry, "basedpyright", version)
		rel := fmt.Sprintf("vendor/toolchains/python/basedpyright/%s/darwin-arm64/package.tgz", version)
		digest, artifact, reused, err := fetchDownloadedArtifact(ctx, opts.Client, repoRoot, staging, previous, component, rel, url, 0o644, nil)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		retrievedAt = reuseRetrievedAt(retrievedAt, reused)
		artifacts := appendArtifact(nil, artifact)
		archivePath, err := downloadedArtifactPath(repoRoot, rel, artifact)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		license := "vendor/toolchains/licenses/basedpyright-LICENSE"
		artifacts, err = stageFromTarGz(repoRoot, staging, archivePath, license, []string{"package/LICENSE.txt", "package/LICENSE"}, artifacts)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		return lock(component, target, rel, url, retrievedAt, digest, license, "", "curl "+url, []config.InstallEntry{
			{Path: "lib/node_modules/basedpyright", Mode: "directory"},
			{Path: "bin/basedpyright-langserver", Mode: "executable"},
			{Path: "bin/basedpyright", Mode: "executable"},
		}, config.Probe{Command: "bin/basedpyright", Args: []string{"--version"}}), artifacts, nil
	case "gopls":
		version := component.UpstreamVersion
		rel := fmt.Sprintf("vendor/toolchains/go/gopls/%s/darwin-arm64/gopls", version)
		finalPath, err := containedPath(repoRoot, rel)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		if previousComponent, ok := matchingPrevious(previous, component, rel); ok {
			if digest, err := fileDigest(finalPath); err == nil && digest.Size == previousComponent.Artifact.Size && digest.SHA256 == previousComponent.Artifact.SHA256 {
				if err := validateLicenseFiles(repoRoot, previousComponent.License); err != nil {
					return config.LockComponent{}, nil, err
				}
				return lock(component, target, rel, opts.Sources.GoProxy+"/golang.org/x/tools/gopls/@v/"+version+".zip", previousComponent.Artifact.RetrievedAt, digest, "vendor/toolchains/licenses/gopls-LICENSE", "", goplsCommand(version), []config.InstallEntry{
					{Path: "bin/gopls", Mode: "executable"},
				}, config.Probe{Command: "bin/gopls", Args: []string{"version"}}), nil, nil
			}
		}
		stagePath, err := containedPath(staging, rel)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		goplsLicenses, err := goplsLicenseTargets(repoRoot, staging)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		if err := buildGopls(ctx, opts.Go, repoRoot, stagePath, version, goplsLicenses); err != nil {
			return config.LockComponent{}, nil, err
		}
		digest, err := fileDigest(stagePath)
		if err != nil {
			return config.LockComponent{}, nil, err
		}
		artifacts := []stagedArtifact{{StagePath: stagePath, FinalPath: finalPath, Mode: 0o755}}
		artifacts = append(artifacts, goplsLicenses...)
		return lock(component, target, rel, strings.TrimRight(opts.Sources.GoProxy, "/")+"/golang.org/x/tools/gopls/@v/"+version+".zip", retrievedAt, digest, "vendor/toolchains/licenses/gopls-LICENSE", "", goplsCommand(version), []config.InstallEntry{
			{Path: "bin/gopls", Mode: "executable"},
		}, config.Probe{Command: "bin/gopls", Args: []string{"version"}}), artifacts, nil
	default:
		return config.LockComponent{}, nil, fmt.Errorf("unsupported component %q", component.Name)
	}
}

func downloadedArtifactPath(repoRoot, rel string, artifact stagedArtifact) (string, error) {
	if artifact.StagePath != "" {
		return artifact.StagePath, nil
	}
	return containedPath(repoRoot, rel)
}

func validateLicenseFiles(repoRoot string, license config.License) error {
	for _, rel := range []string{license.Path, license.Notice} {
		if rel == "" {
			continue
		}
		path, err := containedPath(repoRoot, rel)
		if err != nil {
			return err
		}
		exists, err := regularFileExists(path)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("recorded license file is missing: %s", rel)
		}
	}
	return nil
}

func npmTarballURL(registry, pkg, version string) string {
	base := strings.TrimRight(registry, "/")
	name := pkg
	if slash := strings.LastIndex(pkg, "/"); slash >= 0 {
		name = pkg[slash+1:]
	}
	return fmt.Sprintf("%s/%s/-/%s-%s.tgz", base, pkg, name, version)
}

func fetchDownloadedArtifact(ctx context.Context, client *http.Client, repoRoot, staging string, previous *config.ResolvedLock, component config.DesiredComponent, rel, url string, mode os.FileMode, validate func(string, digest) error) (digest, stagedArtifact, config.LockComponent, error) {
	finalPath, err := containedPath(repoRoot, rel)
	if err != nil {
		return digest{}, stagedArtifact{}, config.LockComponent{}, err
	}
	if previousComponent, ok := matchingPrevious(previous, component, rel); ok {
		if d, err := fileDigest(finalPath); err == nil && d.Size == previousComponent.Artifact.Size && d.SHA256 == previousComponent.Artifact.SHA256 {
			return d, stagedArtifact{}, previousComponent, nil
		}
	}
	stagePath, err := containedPath(staging, rel)
	if err != nil {
		return digest{}, stagedArtifact{}, config.LockComponent{}, err
	}
	d, err := download(ctx, client, url, stagePath, mode)
	if err != nil {
		return digest{}, stagedArtifact{}, config.LockComponent{}, err
	}
	if validate != nil {
		if err := validate(stagePath, d); err != nil {
			return digest{}, stagedArtifact{}, config.LockComponent{}, err
		}
	}
	return d, stagedArtifact{StagePath: stagePath, FinalPath: finalPath, Mode: mode}, config.LockComponent{}, nil
}

func matchingPrevious(previous *config.ResolvedLock, desired config.DesiredComponent, rel string) (config.LockComponent, bool) {
	if previous == nil {
		return config.LockComponent{}, false
	}
	for _, component := range previous.Components {
		if component.Name == desired.Name &&
			component.Kind == desired.Kind &&
			component.Source == desired.Source &&
			component.UpstreamVersion == desired.UpstreamVersion &&
			component.Revision == desired.Revision &&
			component.Target.OS == config.TargetOS &&
			component.Target.Arch == config.TargetArch &&
			component.Artifact.Path == rel {
			return component, true
		}
	}
	return config.LockComponent{}, false
}

func reuseRetrievedAt(fallback string, reused config.LockComponent) string {
	if reused.Artifact.RetrievedAt != "" {
		return reused.Artifact.RetrievedAt
	}
	return fallback
}

func download(ctx context.Context, client *http.Client, url, path string, mode os.FileMode) (digest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return digest{}, err
	}
	res, err := client.Do(req)
	if err != nil {
		return digest{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return digest{}, fmt.Errorf("%s returned %s", url, res.Status)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return digest{}, err
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return digest{}, err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(out, h), res.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return digest{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return digest{}, closeErr
	}
	return digest{Size: n, SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}

func fetchNodeSHA256(ctx context.Context, client *http.Client, shasumsURL, archive string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shasumsURL, nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s returned %s", shasumsURL, res.Status)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == archive {
			if len(fields[0]) != 64 {
				return "", fmt.Errorf("node checksum for %s is not SHA-256", archive)
			}
			return strings.ToLower(fields[0]), nil
		}
	}
	return "", fmt.Errorf("node checksum for %s not found in SHASUMS256.txt", archive)
}

func appendArtifact(artifacts []stagedArtifact, artifact stagedArtifact) []stagedArtifact {
	if artifact.StagePath == "" {
		return artifacts
	}
	return append(artifacts, artifact)
}

func stageFromTarGz(repoRoot, staging, archivePath, finalRel string, candidates []string, artifacts []stagedArtifact) ([]stagedArtifact, error) {
	if archivePath == "" {
		return artifacts, nil
	}
	stagePath, err := containedPath(staging, finalRel)
	if err != nil {
		return nil, err
	}
	finalPath, err := containedPath(repoRoot, finalRel)
	if err != nil {
		return nil, err
	}
	if err := extractTarGzFile(archivePath, candidates, stagePath); err != nil {
		return nil, err
	}
	return append(artifacts, stagedArtifact{StagePath: stagePath, FinalPath: finalPath, Mode: 0o644}), nil
}

func tarGzHasPath(archivePath, wanted string) (bool, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return false, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return false, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if header.Typeflag == tar.TypeReg && filepath.ToSlash(filepath.Clean(header.Name)) == wanted {
			return true, nil
		}
	}
}

func extractTarGzFile(archivePath string, candidates []string, destination string) error {
	const maxExtractedMetadataBytes = 4 << 20
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	wanted := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		wanted[filepath.ToSlash(filepath.Clean(candidate))] = struct{}{}
	}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s missing %s", filepath.Base(archivePath), strings.Join(candidates, " or "))
		}
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Clean(header.Name))
		if _, ok := wanted[name]; !ok {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			return fmt.Errorf("%s is not a regular file in %s", name, filepath.Base(archivePath))
		}
		if header.Size < 0 || header.Size > maxExtractedMetadataBytes {
			return fmt.Errorf("%s exceeds the %d-byte metadata limit", name, maxExtractedMetadataBytes)
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			_ = os.Remove(destination)
			return copyErr
		}
		if closeErr != nil {
			_ = os.Remove(destination)
			return closeErr
		}
		return nil
	}
}

func goplsLicenseTargets(repoRoot, staging string) ([]stagedArtifact, error) {
	rels := []string{
		"vendor/toolchains/licenses/gopls-LICENSE",
	}
	artifacts := make([]stagedArtifact, 0, len(rels))
	for _, rel := range rels {
		stagePath, err := containedPath(staging, rel)
		if err != nil {
			return nil, err
		}
		finalPath, err := containedPath(repoRoot, rel)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, stagedArtifact{
			StagePath: stagePath,
			FinalPath: finalPath,
			Mode:      0o644,
		})
	}
	return artifacts, nil
}

func buildGopls(ctx context.Context, goBinary, repoRoot, output, version string, licenses []stagedArtifact) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "imoogi-gopls-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	cmd := exec.CommandContext(ctx, goBinary, "install", "-trimpath", "-ldflags=-buildid=", "golang.org/x/tools/gopls@"+version)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"GOOS=darwin",
		"GOARCH=arm64",
		"CGO_ENABLED=0",
		"GOBIN="+tmp,
		"GOCACHE="+filepath.Join(tmp, "gocache"),
		"GOMODCACHE="+filepath.Join(tmp, "gomodcache"),
	)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(outputBytes)))
	}
	if err := copyExecutable(filepath.Join(tmp, "gopls"), output); err != nil {
		return err
	}
	moduleDir := filepath.Join(tmp, "gomodcache", "golang.org", "x", "tools", "gopls@"+version)
	licenseSources := map[string]string{
		"gopls-LICENSE": "LICENSE",
	}
	for _, artifact := range licenses {
		if err := os.MkdirAll(filepath.Dir(artifact.StagePath), 0o755); err != nil {
			return err
		}
		sourceName, ok := licenseSources[filepath.Base(artifact.FinalPath)]
		if !ok {
			return fmt.Errorf("unknown gopls license target %s", artifact.FinalPath)
		}
		if err := copyFile(filepath.Join(moduleDir, sourceName), artifact.StagePath, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func goplsCommand(version string) string {
	return "go install -trimpath -ldflags=-buildid= golang.org/x/tools/gopls@" + version
}

type bootstrapProvenance struct {
	CLI       string `json:"cli_version"`
	Builder   string `json:"builder"`
	Go        string `json:"go_version"`
	Command   string `json:"command"`
	BuiltAt   string `json:"built_at"`
	Artifact  string `json:"artifact"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	Generated string `json:"generated_by"`
}

func buildBootstrap(ctx context.Context, opts Options, repoRoot, staging, cliVersion string) ([]stagedArtifact, error) {
	rel := fmt.Sprintf("vendor/toolchains/cli/%s/darwin-arm64/imoogi-toolchain", cliVersion)
	finalPath, err := containedPath(repoRoot, rel)
	if err != nil {
		return nil, err
	}
	provenanceRel := fmt.Sprintf("vendor/toolchains/cli/%s/darwin-arm64/imoogi-toolchain.provenance.json", cliVersion)
	provenanceFinalPath, err := containedPath(repoRoot, provenanceRel)
	if err != nil {
		return nil, err
	}
	binaryExists, err := regularFileExists(finalPath)
	if err != nil {
		return nil, err
	}
	provenanceExists, err := regularFileExists(provenanceFinalPath)
	if err != nil {
		return nil, err
	}
	if binaryExists != provenanceExists {
		return nil, errors.New("bootstrap binary and provenance must either both exist or both be absent")
	}
	if binaryExists {
		if err := validateBootstrapProvenance(finalPath, provenanceFinalPath, rel, cliVersion); err != nil {
			return nil, err
		}
		return nil, nil
	}
	stagePath, err := containedPath(staging, rel)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(stagePath), 0o755); err != nil {
		return nil, err
	}
	spec := activation.BootstrapSpec{Version: cliVersion, Output: stagePath}
	args := spec.Args()
	command := activation.BootstrapSpec{Version: cliVersion, Output: filepath.ToSlash(rel)}.Provenance()
	cmd := exec.CommandContext(ctx, opts.Go, args...)
	cmd.Dir = repoRoot
	cmd.Env = append(append(os.Environ(), spec.Env()...),
		"GOCACHE="+filepath.Join(staging, ".bootstrap-gocache"),
	)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(outputBytes)))
	}
	if err := os.Chmod(stagePath, 0o755); err != nil {
		return nil, err
	}
	d, err := fileDigest(stagePath)
	if err != nil {
		return nil, err
	}
	provenance := bootstrapProvenance{
		CLI:       cliVersion,
		Builder:   runtime.GOOS + "/" + runtime.GOARCH,
		Go:        runtime.Version(),
		Command:   command,
		BuiltAt:   opts.Now().UTC().Format(time.RFC3339),
		Artifact:  rel,
		Size:      d.Size,
		SHA256:    d.SHA256,
		Generated: generatedBy,
	}
	provenanceStagePath, err := containedPath(staging, provenanceRel)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(provenanceStagePath), 0o755); err != nil {
		return nil, err
	}
	out, err := os.OpenFile(provenanceStagePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	encodeErr := enc.Encode(provenance)
	closeErr := out.Close()
	if encodeErr != nil {
		_ = os.Remove(provenanceStagePath)
		return nil, encodeErr
	}
	if closeErr != nil {
		_ = os.Remove(provenanceStagePath)
		return nil, closeErr
	}
	return []stagedArtifact{
		{StagePath: stagePath, FinalPath: finalPath, Mode: 0o755},
		{StagePath: provenanceStagePath, FinalPath: provenanceFinalPath, Mode: 0o644},
	}, nil
}

func regularFileExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("expected regular file: %s", path)
	}
	return true, nil
}

func validateBootstrapProvenance(binaryPath, provenancePath, artifactRel, cliVersion string) error {
	data, err := os.ReadFile(provenancePath)
	if err != nil {
		return err
	}
	var provenance bootstrapProvenance
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&provenance); err != nil {
		return fmt.Errorf("decode bootstrap provenance: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode bootstrap provenance: multiple JSON values")
		}
		return fmt.Errorf("decode bootstrap provenance: %w", err)
	}
	if provenance.CLI != cliVersion || provenance.Artifact != artifactRel || provenance.Generated != generatedBy {
		return errors.New("bootstrap provenance identity does not match the desired CLI")
	}
	d, err := fileDigest(binaryPath)
	if err != nil {
		return err
	}
	if provenance.Size != d.Size || provenance.SHA256 != d.SHA256 {
		return errors.New("bootstrap binary does not match its provenance digest")
	}
	return nil
}

func publishArtifact(artifact stagedArtifact) error {
	if err := os.MkdirAll(filepath.Dir(artifact.FinalPath), 0o755); err != nil {
		return err
	}
	if existing, err := fileDigest(artifact.FinalPath); err == nil {
		next, nextErr := fileDigest(artifact.StagePath)
		if nextErr != nil {
			return nextErr
		}
		if existing == next {
			return nil
		}
		return fmt.Errorf("refuse to overwrite existing artifact with different bytes: %s", artifact.FinalPath)
	} else if !os.IsNotExist(err) {
		return err
	}
	temp := artifact.FinalPath + ".tmp"
	in, err := os.Open(artifact.StagePath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, artifact.Mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(temp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(temp)
		return closeErr
	}
	if err := os.Chmod(temp, artifact.Mode); err != nil {
		_ = os.Remove(temp)
		return err
	}
	if err := os.Rename(temp, artifact.FinalPath); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

func copyExecutable(source, destination string) error {
	return copyFile(source, destination, 0o755)
}

func copyFile(source, destination string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	temp := destination + ".tmp"
	out, err := os.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(temp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(temp)
		return closeErr
	}
	if err := os.Chmod(temp, mode); err != nil {
		_ = os.Remove(temp)
		return err
	}
	if err := os.Rename(temp, destination); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

type digest struct {
	Size   int64
	SHA256 string
}

func fileDigest(path string) (digest, error) {
	f, err := os.Open(path)
	if err != nil {
		return digest{}, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return digest{}, err
	}
	return digest{Size: n, SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}

func lock(component config.DesiredComponent, target config.Target, rel, sourceURL, retrievedAt string, d digest, license, notice, command string, install []config.InstallEntry, probe config.Probe) config.LockComponent {
	return config.LockComponent{
		Name:            component.Name,
		Kind:            component.Kind,
		Source:          component.Source,
		UpstreamVersion: component.UpstreamVersion,
		Revision:        component.Revision,
		Target:          target,
		Artifact: config.Artifact{
			Path:        rel,
			Size:        d.Size,
			SHA256:      d.SHA256,
			SourceURL:   sourceURL,
			RetrievedAt: retrievedAt,
		},
		License: config.License{
			Path:   license,
			Notice: notice,
		},
		Provenance: config.Provenance{
			Builder:   runtime.GOOS + "/" + runtime.GOARCH,
			Toolchain: runtime.Version(),
			Command:   command,
		},
		Install: install,
		Runtime: component.Runtime,
		Probe:   probe,
	}
}
