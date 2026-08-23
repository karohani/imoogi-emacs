package setup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang"
	"github.com/karohani/imoogi-emacs/internal/lang/golang"
	"github.com/karohani/imoogi-emacs/internal/lang/typescript"
)

const (
	metadataSchema      = "imoogi-toolchain-install/v2"
	installMetadataName = "install.json"
)

type Kind string

const (
	KindConfigIntegrity Kind = "config-integrity"
	KindBusy            Kind = "setup-busy"
	KindProbe           Kind = "setup-probe"
	KindSetup           Kind = "setup"
)

type Error struct {
	Kind Kind
	Err  error
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return string(e.Kind)
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type Options struct {
	Workdir         string
	Stdout          io.Writer
	Stderr          io.Writer
	ProviderFactory ProviderFactory
	RunProbe        ProbeRunner
	Failpoints      Failpoints
}

type ProviderFactory func(config.ResolvedLock) ([]lang.Provider, error)
type ProbeRunner func(context.Context, lang.Probe, string, string) error

type Failpoints struct {
	AfterStaging   func() error
	AfterProbes    func() error
	AfterPublish   func() error
	BeforeActivate func() error
}

type metadata struct {
	Schema       string              `json:"schema"`
	CLI          string              `json:"cli_version"`
	Bundle       string              `json:"bundle"`
	Target       config.Target       `json:"target"`
	LockSHA256   string              `json:"lock_sha256"`
	BundleSHA256 string              `json:"bundle_sha256"`
	Components   []metadataComponent `json:"components"`
}

type metadataComponent struct {
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	Source          string `json:"source"`
	UpstreamVersion string `json:"upstream_version"`
	Revision        string `json:"revision,omitempty"`
	ArtifactSHA256  string `json:"artifact_sha256"`
}

func Run(ctx context.Context, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	opts = opts.withDefaults()

	repoRoot, err := FindRoot(opts.Workdir)
	if err != nil {
		return classify(KindConfigIntegrity, err)
	}
	desiredPath := filepath.Join(repoRoot, "toolchains.json")
	lockPath := filepath.Join(repoRoot, "toolchains.lock.json")
	if err := requireRegularFile(desiredPath); err != nil {
		return classify(KindConfigIntegrity, err)
	}
	if err := requireRegularFile(lockPath); err != nil {
		return classify(KindConfigIntegrity, err)
	}
	desired, err := config.LoadDesired(desiredPath)
	if err != nil {
		return classify(KindConfigIntegrity, err)
	}
	lock, err := config.LoadLock(lockPath)
	if err != nil {
		return classify(KindConfigIntegrity, err)
	}
	if err := desired.ValidateLock(*lock); err != nil {
		return classify(KindConfigIntegrity, err)
	}
	lockDigest, err := fileSHA256(lockPath)
	if err != nil {
		return classify(KindConfigIntegrity, err)
	}
	if err := verifyArtifacts(repoRoot, lock.Components); err != nil {
		return classify(KindConfigIntegrity, err)
	}

	localRoot := filepath.Join(repoRoot, ".local")
	if err := ensureRealDir(localRoot); err != nil {
		return classify(KindSetup, fmt.Errorf("prepare .local: %w", err))
	}
	toolchainsRoot := filepath.Join(localRoot, "toolchains")
	if err := ensureRealDir(toolchainsRoot); err != nil {
		return classify(KindSetup, fmt.Errorf("prepare .local/toolchains: %w", err))
	}
	release, err := acquireLock(filepath.Join(localRoot, "setup.lock"))
	if err != nil {
		return classify(KindBusy, err)
	}
	defer release()

	bundleRoot := filepath.Join(localRoot, "toolchains", lock.Bundle)
	if same, err := installedMatches(bundleRoot, lockDigest); err != nil {
		return classify(KindConfigIntegrity, err)
	} else if same {
		if err := runLockProbes(ctx, opts, *lock, bundleRoot, localRoot); err != nil {
			return err
		}
		if err := activate(localRoot, lock.Bundle); err != nil {
			return classify(KindSetup, err)
		}
		fmt.Fprintf(opts.Stdout, "setup reused %s\n", lock.Bundle)
		return nil
	}

	providers, err := opts.ProviderFactory(*lock)
	if err != nil {
		return classify(KindConfigIntegrity, err)
	}

	staging, err := os.MkdirTemp(localRoot, ".staging-"+lock.Bundle+"-*")
	if err != nil {
		return classify(KindSetup, fmt.Errorf("create setup staging: %w", err))
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(staging)
		}
	}()

	var probes []lang.Probe
	for _, provider := range providers {
		providerProbes, err := provider.Materialize(repoRoot, staging)
		if err != nil {
			return classify(KindSetup, err)
		}
		probes = append(probes, providerProbes...)
	}
	if err := writeMetadata(staging, *lock, lockDigest); err != nil {
		return classify(KindSetup, err)
	}
	if opts.Failpoints.AfterStaging != nil {
		if err := opts.Failpoints.AfterStaging(); err != nil {
			return classify(KindSetup, err)
		}
	}
	if err := runProbes(ctx, opts, probes, staging, localRoot); err != nil {
		return err
	}
	if opts.Failpoints.AfterProbes != nil {
		if err := opts.Failpoints.AfterProbes(); err != nil {
			return classify(KindSetup, err)
		}
	}
	if err := ensureRealDir(filepath.Dir(bundleRoot)); err != nil {
		return classify(KindSetup, err)
	}
	if err := publish(staging, bundleRoot, lockDigest); err != nil {
		return classify(KindSetup, err)
	}
	published = true
	if opts.Failpoints.AfterPublish != nil {
		if err := opts.Failpoints.AfterPublish(); err != nil {
			return classify(KindSetup, err)
		}
	}
	if opts.Failpoints.BeforeActivate != nil {
		if err := opts.Failpoints.BeforeActivate(); err != nil {
			return classify(KindSetup, err)
		}
	}
	if err := activate(localRoot, lock.Bundle); err != nil {
		return classify(KindSetup, err)
	}
	fmt.Fprintf(opts.Stdout, "setup activated %s\n", lock.Bundle)
	return nil
}

func (o Options) withDefaults() Options {
	if o.Workdir == "" {
		o.Workdir = "."
	}
	if o.Stdout == nil {
		o.Stdout = io.Discard
	}
	if o.Stderr == nil {
		o.Stderr = io.Discard
	}
	if o.ProviderFactory == nil {
		o.ProviderFactory = DefaultProviderFactory
	}
	if o.RunProbe == nil {
		o.RunProbe = runProbe
	}
	return o
}

func FindRoot(start string) (string, error) {
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
		if regularFileExists(filepath.Join(dir, "toolchains.json")) && regularFileExists(filepath.Join(dir, "toolchains.lock.json")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("toolchains.json and toolchains.lock.json not found from %s", start)
		}
		dir = parent
	}
}

func DefaultProviderFactory(lock config.ResolvedLock) ([]lang.Provider, error) {
	var providers []lang.Provider
	processed := map[string]struct{}{}
	var tsComponents []config.LockComponent
	for _, component := range lock.Components {
		switch {
		case component.Name == "gopls" || component.Kind == "go-language-server":
			providers = append(providers, golang.New(component))
			processed[component.Name] = struct{}{}
		case isTypeScriptComponent(component):
			tsComponents = append(tsComponents, component)
			processed[component.Name] = struct{}{}
		}
	}
	if len(tsComponents) > 0 {
		providers = append(providers, typescript.New(tsComponents))
	}
	for _, component := range lock.Components {
		if _, ok := processed[component.Name]; !ok {
			return nil, fmt.Errorf("no provider for component %q kind %q", component.Name, component.Kind)
		}
	}
	return providers, nil
}

func isTypeScriptComponent(component config.LockComponent) bool {
	switch {
	case component.Name == "node" || component.Kind == "node-runtime":
		return true
	case component.Name == "typescript" || component.Kind == "typescript-sdk":
		return true
	case component.Name == "typescript-language-server" || component.Kind == "typescript-language-server":
		return true
	// node 런타임을 공유하므로 같은 provider 가 설치한다(typescript 와 무관한
	// 서버지만, 묶는 기준은 언어가 아니라 런타임이다).
	case component.Name == "basedpyright" || component.Kind == "python-language-server":
		return true
	default:
		return false
	}
}

func verifyArtifacts(repoRoot string, components []config.LockComponent) error {
	for _, component := range components {
		path := lang.ArtifactPath(repoRoot, component)
		digest := artifact.Digest{Size: component.Artifact.Size, SHA256: component.Artifact.SHA256}
		if err := artifact.VerifyFile(path, digest); err != nil {
			return fmt.Errorf("verify %s artifact: %w", component.Name, err)
		}
	}
	return nil
}

func runLockProbes(ctx context.Context, opts Options, lock config.ResolvedLock, bundleRoot, localRoot string) error {
	probes := make([]lang.Probe, 0, len(lock.Components))
	for _, component := range lock.Components {
		probe, err := lang.AbsoluteProbe(bundleRoot, component.Probe)
		if err != nil {
			return classify(KindConfigIntegrity, fmt.Errorf("%s probe: %w", component.Name, err))
		}
		probes = append(probes, probe)
	}
	return runProbes(ctx, opts, probes, bundleRoot, localRoot)
}

func runProbes(ctx context.Context, opts Options, probes []lang.Probe, bundleRoot, localRoot string) error {
	probeRoot, err := os.MkdirTemp(localRoot, ".probe-*")
	if err != nil {
		return classify(KindSetup, fmt.Errorf("create probe temp: %w", err))
	}
	defer os.RemoveAll(probeRoot)
	for _, probe := range probes {
		if !filepath.IsAbs(probe.Command) {
			return classify(KindProbe, fmt.Errorf("probe command %q is not absolute", probe.Command))
		}
		if err := opts.RunProbe(ctx, probe, bundleRoot, probeRoot); err != nil {
			return classify(KindProbe, fmt.Errorf("%s %s: %w", probe.Command, strings.Join(probe.Args, " "), err))
		}
	}
	return nil
}

func runProbe(ctx context.Context, probe lang.Probe, bundleRoot, probeRoot string) error {
	cmd := exec.CommandContext(ctx, probe.Command, probe.Args...)
	cmd.Dir = bundleRoot
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + probeRoot,
		"TMPDIR=" + probeRoot,
		"NO_PROXY=*",
		"no_proxy=*",
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(output.String()))
	}
	return nil
}

func acquireLock(path string) (func(), error) {
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_CREAT|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, &Error{Kind: KindSetup, Err: fmt.Errorf("setup lock is a symlink: %s", path)}
		}
		return nil, fmt.Errorf("open setup lock: %w", err)
	}
	f := os.NewFile(uintptr(fd), path)
	if f == nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("open setup lock: invalid file descriptor")
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("setup already running: %s", path)
		}
		return nil, fmt.Errorf("lock setup: %w", err)
	}
	release := func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
	if err := f.Truncate(0); err != nil {
		release()
		return nil, fmt.Errorf("write setup lock: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		release()
		return nil, fmt.Errorf("write setup lock: %w", err)
	}
	if _, err := fmt.Fprintf(f, "pid=%d\n", os.Getpid()); err != nil {
		release()
		return nil, fmt.Errorf("write setup lock: %w", err)
	}
	return release, nil
}

func publish(staging, bundleRoot, lockDigest string) error {
	if same, err := installedMatches(bundleRoot, lockDigest); err != nil {
		return err
	} else if same {
		if err := os.RemoveAll(staging); err != nil {
			return fmt.Errorf("remove duplicate staging: %w", err)
		}
		return nil
	}
	if err := os.Rename(staging, bundleRoot); err != nil {
		if same, matchErr := installedMatches(bundleRoot, lockDigest); matchErr == nil && same {
			if removeErr := os.RemoveAll(staging); removeErr != nil {
				return fmt.Errorf("remove duplicate staging after publish race: %w", removeErr)
			}
			return nil
		}
		return fmt.Errorf("publish bundle: %w", err)
	}
	return nil
}

func activate(localRoot, bundle string) error {
	linkTarget := filepath.ToSlash(filepath.Join("toolchains", bundle, "bin"))
	tempLink := filepath.Join(localRoot, ".bin-"+bundle+".tmp")
	_ = os.Remove(tempLink)
	if err := os.Symlink(linkTarget, tempLink); err != nil {
		return fmt.Errorf("create temporary activation link: %w", err)
	}
	if err := os.Rename(tempLink, filepath.Join(localRoot, "bin")); err != nil {
		_ = os.Remove(tempLink)
		return fmt.Errorf("activate toolchain link: %w", err)
	}
	return nil
}

func installedMatches(bundleRoot, lockDigest string) (bool, error) {
	info, err := os.Lstat(bundleRoot)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect existing bundle: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("existing bundle %s is a symlink", bundleRoot)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("existing bundle %s is not a directory", bundleRoot)
	}
	metadataPath := filepath.Join(bundleRoot, installMetadataName)
	if err := requireRegularFile(metadataPath); err != nil {
		return false, fmt.Errorf("existing bundle %s has invalid install metadata: %w", bundleRoot, err)
	}
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return false, fmt.Errorf("existing bundle %s has no readable install metadata", bundleRoot)
	}
	var meta metadata
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&meta); err != nil {
		return false, fmt.Errorf("existing bundle install metadata is invalid: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return false, errors.New("existing bundle install metadata has trailing JSON values")
		}
		return false, fmt.Errorf("existing bundle install metadata is invalid: %w", err)
	}
	if meta.Schema != metadataSchema || meta.LockSHA256 != lockDigest {
		return false, fmt.Errorf("existing bundle %s conflicts with lock digest", bundleRoot)
	}
	bundleDigest, err := bundleTreeSHA256(bundleRoot)
	if err != nil {
		return false, fmt.Errorf("verify existing bundle %s: %w", bundleRoot, err)
	}
	if meta.BundleSHA256 != bundleDigest {
		return false, fmt.Errorf("existing bundle %s failed content integrity verification", bundleRoot)
	}
	return true, nil
}

func writeMetadata(root string, lock config.ResolvedLock, lockDigest string) error {
	bundleDigest, err := bundleTreeSHA256(root)
	if err != nil {
		return fmt.Errorf("hash installed bundle: %w", err)
	}
	meta := metadata{
		Schema:       metadataSchema,
		CLI:          lock.CLI,
		Bundle:       lock.Bundle,
		Target:       lock.Target,
		LockSHA256:   lockDigest,
		BundleSHA256: bundleDigest,
		Components:   make([]metadataComponent, 0, len(lock.Components)),
	}
	for _, component := range lock.Components {
		meta.Components = append(meta.Components, metadataComponent{
			Name:            component.Name,
			Kind:            component.Kind,
			Source:          component.Source,
			UpstreamVersion: component.UpstreamVersion,
			Revision:        component.Revision,
			ArtifactSHA256:  component.Artifact.SHA256,
		})
	}
	sort.Slice(meta.Components, func(i, j int) bool {
		return meta.Components[i].Name < meta.Components[j].Name
	})
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(root, installMetadataName), data, 0o644)
}

func bundleTreeSHA256(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." || filepath.ToSlash(rel) == installMetadataName {
			return nil
		}
		rel = filepath.ToSlash(rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if err := writeHashField(h, []byte(rel)); err != nil {
			return err
		}
		if err := binary.Write(h, binary.BigEndian, uint32(info.Mode().Perm())); err != nil {
			return err
		}
		switch {
		case info.IsDir():
			_, err = h.Write([]byte{'d'})
			return err
		case info.Mode().IsRegular():
			if _, err := h.Write([]byte{'f'}); err != nil {
				return err
			}
			if err := binary.Write(h, binary.BigEndian, uint64(info.Size())); err != nil {
				return err
			}
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(h, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		case info.Mode()&os.ModeSymlink != 0:
			if _, err := h.Write([]byte{'l'}); err != nil {
				return err
			}
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return writeHashField(h, []byte(target))
		default:
			return fmt.Errorf("unsupported installed bundle entry type: %s", rel)
		}
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeHashField(w io.Writer, value []byte) error {
	if err := binary.Write(w, binary.BigEndian, uint64(len(value))); err != nil {
		return err
	}
	_, err := w.Write(value)
	return err
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read lock digest: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func regularFileExists(path string) bool {
	return requireRegularFile(path) == nil
}

func requireRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", filepath.Base(path))
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	return nil
}

func ensureRealDir(path string) error {
	if err := os.Mkdir(path, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func classify(kind Kind, err error) error {
	if err == nil {
		return nil
	}
	var setupErr *Error
	if errors.As(err, &setupErr) {
		return err
	}
	return &Error{Kind: kind, Err: err}
}
