package npmbundle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/karohani/imoogi-emacs/internal/artifact"
)

// CIArgs is the npm invocation every build uses. --omit=optional drops the
// per-platform native packages the adapters list as optional; --ignore-scripts
// runs no install scripts; --no-bin-links keeps symlinks out of the tree;
// --install-strategy=hoisted pins the layout regardless of user config.
var CIArgs = []string{
	"ci",
	"--omit=optional",
	"--ignore-scripts",
	"--no-audit",
	"--no-fund",
	"--no-bin-links",
	"--install-strategy=hoisted",
}

var (
	// ErrInputs marks committed inputs that do not match the pin (lockfile
	// digest, package.json, lockfile resolution): a configuration fault.
	ErrInputs = errors.New("bundle inputs do not match the pin")
	// ErrNPM marks an npm ci that did not complete with valid inputs: offline,
	// a registry error, or another npm failure.
	ErrNPM = errors.New("npm install did not complete")
)

// Command describes a build for lock provenance.
func Command(lockfile string) string {
	return "npm " + strings.Join(CIArgs, " ") + " (" + lockfile + ", npm bundled with the locked node) then deterministic tar.gz of node_modules"
}

// Runner runs argv (argv[0] is the node binary) in dir.
type Runner func(ctx context.Context, dir string, argv []string) error

type BuildOptions struct {
	PackageDir    string // holds the committed package.json and package-lock.json
	LockSHA256    string // expected SHA-256 of package-lock.json
	Package       string // npm package the bundle pins, e.g. @scope/name
	Version       string // exact version of Package
	NodeArchive   string // locked node distribution tar.gz; its npm runs the install
	NodeDigest    artifact.Digest
	Output        string // tar.gz to create; must not exist
	LicenseOutput string // optional: where to copy the package's own license
	Run           Runner // nil runs the real npm
}

// Build installs Package from the committed lockfile with the npm that ships
// inside the locked node distribution, checks the result, and packs it to
// Output. Inputs are validated before npm runs; on any failure Output is not
// left behind.
//
// @MX:WARN: [AUTO] runs npm, which reaches the registry and reads the user's ~/.npmrc
// @MX:REASON: user npmrc carries registry, proxy and min-release-age settings that must apply; only the cache is isolated
func Build(ctx context.Context, opts BuildOptions) (artifact.Digest, error) {
	if opts.Run == nil {
		opts.Run = runNpm
	}
	if _, err := os.Lstat(opts.Output); err == nil {
		return artifact.Digest{}, fmt.Errorf("build output %s already exists", opts.Output)
	} else if !errors.Is(err, os.ErrNotExist) {
		return artifact.Digest{}, err
	}
	lockData, err := os.ReadFile(filepath.Join(opts.PackageDir, "package-lock.json"))
	if err != nil {
		return artifact.Digest{}, err
	}
	sum := sha256.Sum256(lockData)
	if got := hex.EncodeToString(sum[:]); got != opts.LockSHA256 {
		return artifact.Digest{}, fmt.Errorf("%w: package-lock.json sha256 %s does not match the locked revision %s", ErrInputs, got, opts.LockSHA256)
	}
	pkgData, err := os.ReadFile(filepath.Join(opts.PackageDir, "package.json"))
	if err != nil {
		return artifact.Digest{}, err
	}
	if err := checkPin(pkgData, lockData, opts.Package, opts.Version); err != nil {
		return artifact.Digest{}, fmt.Errorf("%w: %w", ErrInputs, err)
	}

	work, err := os.MkdirTemp("", "imoogi-npm-build-*")
	if err != nil {
		return artifact.Digest{}, err
	}
	defer os.RemoveAll(work)
	nodeDir := filepath.Join(work, "node")
	if _, err := artifact.ExtractTarGzipWithOptions(opts.NodeArchive, nodeDir, opts.NodeDigest,
		artifact.Limits{MaxEntries: 20_000, MaxFileBytes: 512 << 20, MaxOutputBytes: 1 << 30},
		artifact.ExtractOptions{AllowRelativeSymlinks: true}); err != nil {
		return artifact.Digest{}, fmt.Errorf("extract locked node: %w", err)
	}
	dists, err := os.ReadDir(nodeDir)
	if err != nil {
		return artifact.Digest{}, err
	}
	if len(dists) != 1 || !dists[0].IsDir() {
		return artifact.Digest{}, errors.New("locked node archive must contain one distribution directory")
	}
	dist := filepath.Join(nodeDir, dists[0].Name())
	nodeBin := filepath.Join(dist, "bin", "node")
	npmCLI := filepath.Join(dist, "lib", "node_modules", "npm", "bin", "npm-cli.js")
	for _, required := range []string{nodeBin, npmCLI} {
		if info, err := os.Lstat(required); err != nil || !info.Mode().IsRegular() {
			return artifact.Digest{}, fmt.Errorf("locked node distribution has no %s", filepath.Base(required))
		}
	}
	// The extractor writes every file 0644 regardless of the archive's modes.
	if err := os.Chmod(nodeBin, 0o755); err != nil {
		return artifact.Digest{}, err
	}

	project := filepath.Join(work, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		return artifact.Digest{}, err
	}
	for name, data := range map[string][]byte{"package.json": pkgData, "package-lock.json": lockData} {
		if err := os.WriteFile(filepath.Join(project, name), data, 0o644); err != nil {
			return artifact.Digest{}, err
		}
	}
	argv := append([]string{nodeBin, npmCLI}, CIArgs...)
	argv = append(argv, "--cache", filepath.Join(work, "npm-cache"))
	if err := opts.Run(ctx, project, argv); err != nil {
		return artifact.Digest{}, fmt.Errorf("%w: npm ci for %s@%s: %w", ErrNPM, opts.Package, opts.Version, err)
	}

	installed := filepath.Join(project, "node_modules", filepath.FromSlash(opts.Package))
	if _, err := BinEntry(installed, opts.Package, opts.Version); err != nil {
		return artifact.Digest{}, err
	}
	if opts.LicenseOutput != "" {
		if err := copyLicense(installed, opts.LicenseOutput); err != nil {
			return artifact.Digest{}, err
		}
	}
	return packTo(project, opts.Output)
}

func packTo(project, output string) (artifact.Digest, error) {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return artifact.Digest{}, err
	}
	out, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return artifact.Digest{}, err
	}
	packErr := Pack(project, out)
	closeErr := out.Close()
	if packErr != nil || closeErr != nil {
		_ = os.Remove(output)
		return artifact.Digest{}, errors.Join(packErr, closeErr)
	}
	return DigestFile(output)
}

// checkPin requires package.json to depend on exactly Package@Version and the
// lockfile to resolve that version.
func checkPin(pkgData, lockData []byte, pkg, version string) error {
	var manifest struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(pkgData, &manifest); err != nil {
		return fmt.Errorf("package.json: %w", err)
	}
	if len(manifest.Dependencies) != 1 || manifest.Dependencies[pkg] != version {
		return fmt.Errorf("package.json must depend on exactly %s %s, has %v", pkg, version, manifest.Dependencies)
	}
	var lock struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(lockData, &lock); err != nil {
		return fmt.Errorf("package-lock.json: %w", err)
	}
	if got := lock.Packages["node_modules/"+pkg].Version; got != version {
		return fmt.Errorf("package-lock.json resolves %s to %q, want %s", pkg, got, version)
	}
	return nil
}

// BinEntry checks that dir is the installed Package@Version and returns its
// single bin script as a clean path relative to dir.
func BinEntry(dir, pkg, version string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return "", fmt.Errorf("installed %s: %w", pkg, err)
	}
	var manifest struct {
		Name    string            `json:"name"`
		Version string            `json:"version"`
		Bin     map[string]string `json:"bin"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("installed %s package.json: %w", pkg, err)
	}
	if manifest.Name != pkg || manifest.Version != version {
		return "", fmt.Errorf("installed package is %s@%s, want %s@%s", manifest.Name, manifest.Version, pkg, version)
	}
	if len(manifest.Bin) != 1 {
		return "", fmt.Errorf("%s declares %d bin entries, want 1", pkg, len(manifest.Bin))
	}
	for _, script := range manifest.Bin {
		rel, ok := cleanRelative(script)
		if !ok {
			return "", fmt.Errorf("%s bin %q is not a clean relative path", pkg, script)
		}
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			return "", fmt.Errorf("%s bin %s is missing", pkg, rel)
		}
		return rel, nil
	}
	return "", nil
}

func copyLicense(dir, output string) error {
	for _, name := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return err
		}
		return os.WriteFile(output, data, 0o644)
	}
	return fmt.Errorf("%s has no LICENSE file", dir)
}

func runNpm(ctx context.Context, dir string, argv []string) error {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(argv[0])+string(os.PathListSeparator)+os.Getenv("PATH"))
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, tail(output.String(), 2000))
	}
	return nil
}

func tail(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return "…" + s[len(s)-n:]
	}
	return s
}
