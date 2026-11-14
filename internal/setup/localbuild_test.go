package setup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang/acp"
	"github.com/karohani/imoogi-emacs/internal/lang/typescript"
	"github.com/karohani/imoogi-emacs/internal/npmbundle"
)

const (
	adapterName    = "claude-agent-acp"
	adapterPkg     = "@agentclientprotocol/claude-agent-acp"
	adapterVersion = "0.99.1"
	adapterLock    = `{"lockfileVersion":3,"packages":{"":{"dependencies":{"@agentclientprotocol/claude-agent-acp":"0.99.1"}},"node_modules/@agentclientprotocol/claude-agent-acp":{"version":"0.99.1"}}}`
)

// fakeNPM lays out what npm ci would install, writing marker into the adapter.
func fakeNPM(calls *int, marker string) npmbundle.Runner {
	return func(_ context.Context, dir string, _ []string) error {
		*calls++
		pkgDir := filepath.Join(dir, "node_modules", filepath.FromSlash(adapterPkg))
		if err := os.MkdirAll(filepath.Join(pkgDir, "dist"), 0o755); err != nil {
			return err
		}
		manifest := `{"name":"` + adapterPkg + `","version":"` + adapterVersion + `","bin":{"claude-agent-acp":"dist/index.js"}}`
		if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(manifest), 0o644); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(pkgDir, "dist", "index.js"), []byte(marker), 0o644)
	}
}

func fakeNodeArchive(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, name := range []string{"node-v24.19.0-darwin-arm64/bin/node", "node-v24.19.0-darwin-arm64/lib/node_modules/npm/bin/npm-cli.js"} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: 1, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// localBuildRepo is makeRepo plus a node component and a local-build adapter
// whose locked digest is what the fake npm produces with marker "v1".
func localBuildRepo(t *testing.T) (string, config.LockComponent) {
	t.Helper()
	root := makeRepo(t, "repo", "one")
	lock, err := config.LoadLock(filepath.Join(root, "toolchains.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	desired, err := config.LoadDesired(filepath.Join(root, "toolchains.json"))
	if err != nil {
		t.Fatal(err)
	}
	base := lock.Components[0]

	nodeData := fakeNodeArchive(t)
	nodeRel := "vendor/node.tar.gz"
	if err := os.WriteFile(filepath.Join(root, nodeRel), nodeData, 0o644); err != nil {
		t.Fatal(err)
	}
	node := base
	node.Name, node.Kind, node.Source, node.UpstreamVersion = "node", "node-runtime", "nodejs.org/dist", "v24.19.0"
	node.Artifact.Path, node.Artifact.Size, node.Artifact.SHA256 = nodeRel, int64(len(nodeData)), sha(nodeData)
	node.Probe = config.Probe{Command: "bin/node"}
	node.Install = []config.InstallEntry{{Path: "bin/node", Mode: "executable"}}

	pkgDir := filepath.Join(root, filepath.FromSlash(config.ACPPackageDir(adapterName)))
	mustMkdir(t, pkgDir)
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"dependencies":{"`+adapterPkg+`":"`+adapterVersion+`"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package-lock.json"), []byte(adapterLock), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls int
	expected := filepath.Join(t.TempDir(), "expected.tgz")
	digest, err := npmbundle.Build(context.Background(), npmbundle.BuildOptions{
		PackageDir: pkgDir, LockSHA256: sha([]byte(adapterLock)), Package: adapterPkg, Version: adapterVersion,
		NodeArchive: filepath.Join(root, nodeRel), NodeDigest: artifact.Digest{Size: node.Artifact.Size, SHA256: node.Artifact.SHA256},
		Output: expected, Run: fakeNPM(&calls, "v1"),
	})
	if err != nil {
		t.Fatal(err)
	}

	adapter := base
	adapter.Name, adapter.Kind, adapter.Source, adapter.UpstreamVersion = adapterName, config.KindACPAdapter, "npm:"+adapterPkg, adapterVersion
	adapter.Revision, adapter.ArtifactPolicy = sha([]byte(adapterLock)), config.ArtifactPolicyLocalBuild
	adapter.Artifact.Path = "vendor/toolchains/acp/" + adapterName + "/" + adapterVersion + "/darwin-arm64/bundle.tgz"
	adapter.Artifact.Size, adapter.Artifact.SHA256 = digest.Size, digest.SHA256
	adapter.Probe = config.Probe{Command: "bin/" + adapterName}
	adapter.Install = []config.InstallEntry{{Path: "bin/" + adapterName, Mode: "executable"}}

	lock.Components = append(lock.Components, node, adapter)
	for _, c := range []config.LockComponent{node, adapter} {
		desired.Components = append(desired.Components, config.DesiredComponent{
			Name: c.Name, Kind: c.Kind, Source: c.Source, UpstreamVersion: c.UpstreamVersion,
			Revision: c.Revision, ArtifactPolicy: c.ArtifactPolicy,
		})
	}
	writeJSON(t, filepath.Join(root, "toolchains.json"), desired)
	writeJSON(t, filepath.Join(root, "toolchains.lock.json"), lock)
	return root, adapter
}

func sha(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestSetupBuildsAMissingLocalBuildArtifactAndVerifiesIt(t *testing.T) {
	root, adapter := localBuildRepo(t)
	var calls int
	var stdout bytes.Buffer
	opts := Options{Workdir: root, Stdout: &stdout, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe, NPMRunner: fakeNPM(&calls, "v1")}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("npm runs = %d, want 1", calls)
	}
	if err := artifact.VerifyFile(filepath.Join(root, filepath.FromSlash(adapter.Artifact.Path)),
		artifact.Digest{Size: adapter.Artifact.Size, SHA256: adapter.Artifact.SHA256}); err != nil {
		t.Fatalf("built artifact: %v", err)
	}
	if !strings.Contains(stdout.String(), "building "+adapterName) {
		t.Fatalf("stdout does not announce the local build: %q", stdout.String())
	}
	assertActiveBundle(t, root, "2026.08.22.1")
	assertNoBuildTemp(t, root, adapter)

	// Present artifact: no rebuild.
	if err := Run(context.Background(), opts); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("npm ran again for a present artifact (%d runs)", calls)
	}
}

func TestSetupRejectsALocalBuildThatDoesNotMatchTheLock(t *testing.T) {
	root, adapter := localBuildRepo(t)
	lockBefore, err := os.ReadFile(filepath.Join(root, "toolchains.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	err = Run(context.Background(), Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe, NPMRunner: fakeNPM(&calls, "drifted")})
	if err == nil || kindOf(err) != KindConfigIntegrity || !strings.Contains(err.Error(), adapter.Artifact.SHA256) {
		t.Fatalf("Run() error = %v (kind %q), want config-integrity naming the locked digest", err, kindOf(err))
	}
	if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(adapter.Artifact.Path))); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("a mismatching build was published")
	}
	lockAfter, err := os.ReadFile(filepath.Join(root, "toolchains.lock.json"))
	if err != nil || !bytes.Equal(lockBefore, lockAfter) {
		t.Fatal("the lock changed after a mismatching build")
	}
	assertNoBuildTemp(t, root, adapter)
	if _, err := os.Lstat(filepath.Join(root, ".local", "bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("setup activated after a failed local build")
	}
}

func offlineNPM(calls *int) npmbundle.Runner {
	return func(context.Context, string, []string) error {
		*calls++
		return errors.New("getaddrinfo ENOTFOUND registry.npmjs.org")
	}
}

func activeBin(root, name string) string {
	return filepath.Join(root, ".local", "bin", name)
}

func readInstallRecord(t *testing.T, root string) metadata {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".local", "toolchains", "2026.08.22.1", installMetadataName))
	if err != nil {
		t.Fatal(err)
	}
	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatal(err)
	}
	return meta
}

func TestSetupSkipsALocalBuildThatCannotBuildAndActivatesTheRest(t *testing.T) {
	root, adapter := localBuildRepo(t)
	var calls int
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), Options{Workdir: root, Stdout: &stdout, Stderr: &stderr,
		ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe, NPMRunner: offlineNPM(&calls)})
	if err != nil {
		t.Fatalf("Run() error = %v, want setup to finish without the local-build component", err)
	}
	assertActiveBundle(t, root, "2026.08.22.1")
	for _, name := range []string{"one", "node"} {
		if _, err := os.Stat(activeBin(root, name)); err != nil {
			t.Fatalf("%s was not installed: %v", name, err)
		}
	}
	if _, err := os.Lstat(activeBin(root, adapterName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a launcher was installed for the skipped %s (%v)", adapterName, err)
	}
	warning := stderr.String()
	for _, want := range []string{adapterName, "ENOTFOUND", "make toolchain-setup", adapter.Artifact.Path} {
		if !strings.Contains(warning, want) {
			t.Fatalf("warning %q does not mention %q", warning, want)
		}
	}
	if meta := readInstallRecord(t, root); len(meta.Skipped) != 1 || meta.Skipped[0] != adapterName {
		t.Fatalf("install.json skipped = %v", meta.Skipped)
	}
	assertNoBuildTemp(t, root, adapter)

	// Still offline: the incomplete bundle is reused and the warning repeats.
	stdout.Reset()
	stderr.Reset()
	if err := Run(context.Background(), Options{Workdir: root, Stdout: &stdout, Stderr: &stderr,
		ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe, NPMRunner: offlineNPM(&calls)}); err != nil {
		t.Fatalf("offline rerun: %v", err)
	}
	if !strings.Contains(stdout.String(), "setup reused") || !strings.Contains(stderr.String(), adapterName) {
		t.Fatalf("offline rerun stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	// Online again: the build succeeds and the bundle is completed, not reused.
	stdout.Reset()
	if err := Run(context.Background(), Options{Workdir: root, Stdout: &stdout, Stderr: &stderr,
		ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe, NPMRunner: fakeNPM(&calls, "v1")}); err != nil {
		t.Fatalf("online rerun: %v", err)
	}
	if strings.Contains(stdout.String(), "setup reused") {
		t.Fatalf("a bundle that skipped %s was reused after the build succeeded: %q", adapterName, stdout.String())
	}
	if _, err := os.Stat(activeBin(root, adapterName)); err != nil {
		t.Fatalf("%s was not installed once its build succeeded: %v", adapterName, err)
	}
	if meta := readInstallRecord(t, root); len(meta.Skipped) != 0 {
		t.Fatalf("install.json still records skipped %v", meta.Skipped)
	}
	assertActiveBundle(t, root, "2026.08.22.1")
	assertNoTransactionTemp(t, root)
	entries, err := os.ReadDir(filepath.Join(root, ".local", "toolchains"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "2026.08.22.1" {
			t.Fatalf("replacement left %s behind", e.Name())
		}
	}

	// Complete now: a further run reuses it.
	stdout.Reset()
	if err := Run(context.Background(), Options{Workdir: root, Stdout: &stdout,
		ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe, NPMRunner: offlineNPM(&calls)}); err != nil || !strings.Contains(stdout.String(), "setup reused") {
		t.Fatalf("complete rerun: %v %q", err, stdout.String())
	}
}

func TestSkipWarningKeepsTheCauseAndDropsNpmNoise(t *testing.T) {
	reason := errors.New(`npm install did not complete: npm ci for @x/y@1.0.0: exit status 1: npm error code ECONNREFUSED
npm error syscall connect
npm error FetchError: request to http://127.0.0.1:9/z/-/z-1.tgz failed, reason: connect ECONNREFUSED 127.0.0.1:9
npm error     at ClientRequest.<anonymous> (/tmp/x/minipass-fetch/lib/index.js:130:14)
npm error }
npm error A complete log of this run can be found in: /tmp/imoogi-npm-build-1/npm-cache/_logs/debug-0.log`)
	var out bytes.Buffer
	warnSkipped(&out, config.LockComponent{Name: "claude-agent-acp", UpstreamVersion: "0.99.1", Artifact: config.Artifact{Path: "vendor/x/bundle.tgz"}}, reason)
	got := out.String()
	if lines := strings.Count(got, "\n"); lines != 2 {
		t.Fatalf("warning has %d lines, want 2:\n%s", lines, got)
	}
	for _, want := range []string{"ECONNREFUSED", "request to http://127.0.0.1:9/z/-/z-1.tgz failed", "make toolchain-setup", "vendor/x/bundle.tgz"} {
		if !strings.Contains(got, want) {
			t.Fatalf("warning lacks %q:\n%s", want, got)
		}
	}
	for _, noise := range []string{"ClientRequest", "complete log", "imoogi-npm-build"} {
		if strings.Contains(got, noise) {
			t.Fatalf("warning keeps npm noise %q:\n%s", noise, got)
		}
	}
}

func TestSetupStillFailsHardOnAStaleLockfileRevision(t *testing.T) {
	root, adapter := localBuildRepo(t)
	lockfile := filepath.Join(root, filepath.FromSlash(config.ACPPackageDir(adapterName)), "package-lock.json")
	if err := os.WriteFile(lockfile, []byte(adapterLock+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls int
	err := Run(context.Background(), Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe, NPMRunner: fakeNPM(&calls, "v1")})
	if err == nil || kindOf(err) != KindConfigIntegrity || !strings.Contains(err.Error(), "package-lock.json") {
		t.Fatalf("Run() error = %v (kind %q), want config-integrity on the lockfile revision", err, kindOf(err))
	}
	if calls != 0 {
		t.Fatal("npm ran with a stale lockfile")
	}
	if _, err := os.Lstat(filepath.Join(root, ".local", "bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("setup activated despite a stale lockfile revision")
	}
	_ = adapter
}

// TestStaleLockfileFailsEvenWhenNothingIsBuilt: the committed package-lock.json
// is checked against the lock revision before setup decides whether to build,
// install or reuse, so a present artifact or an installed bundle does not hide
// a lockfile that no longer matches.
func TestStaleLockfileFailsEvenWhenNothingIsBuilt(t *testing.T) {
	for _, name := range []string{"artifact present", "bundle installed"} {
		t.Run(name, func(t *testing.T) {
			root, _ := localBuildRepo(t)
			var calls int
			opts := Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe, NPMRunner: fakeNPM(&calls, "v1")}
			if err := Run(context.Background(), opts); err != nil {
				t.Fatalf("first Run() error = %v", err)
			}
			if name == "artifact present" {
				if err := os.RemoveAll(filepath.Join(root, ".local")); err != nil {
					t.Fatal(err)
				}
			}
			lockfile := filepath.Join(root, filepath.FromSlash(config.ACPPackageDir(adapterName)), "package-lock.json")
			if err := os.WriteFile(lockfile, []byte(adapterLock+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			var stdout bytes.Buffer
			opts.Stdout = &stdout
			err := Run(context.Background(), opts)
			if err == nil || kindOf(err) != KindConfigIntegrity || !strings.Contains(err.Error(), "package-lock.json") {
				t.Fatalf("Run() error = %v (kind %q), want config-integrity on the lockfile revision", err, kindOf(err))
			}
			if out := stdout.String(); strings.Contains(out, "setup reused") || strings.Contains(out, "setup activated") {
				t.Fatalf("setup went on despite a stale lockfile: %q", out)
			}
			if calls != 1 {
				t.Fatalf("npm runs = %d, want only the first run's build", calls)
			}
			if name == "artifact present" {
				if _, err := os.Lstat(filepath.Join(root, ".local", "bin")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("setup activated despite a stale lockfile revision")
				}
			}
		})
	}
}

func TestDefaultProviderFactoryRoutesACPAdaptersWithTheNodeComponent(t *testing.T) {
	providers, err := DefaultProviderFactory(config.ResolvedLock{Components: []config.LockComponent{
		providerComponent("codex-acp", config.KindACPAdapter),
		providerComponent("node", "node-runtime"),
		providerComponent("typescript", "typescript-sdk"),
		providerComponent("typescript-language-server", "typescript-language-server"),
		providerComponent("claude-agent-acp", config.KindACPAdapter),
	}})
	if err != nil {
		t.Fatalf("DefaultProviderFactory() error = %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("providers = %d", len(providers))
	}
	if _, ok := providers[0].(typescript.Provider); !ok {
		t.Fatalf("first provider = %T, want the typescript provider that stages node first", providers[0])
	}
	p, ok := providers[1].(acp.Provider)
	if !ok || p.Node.Name != "node" || len(p.Components) != 2 || p.Components[0].Name != "codex-acp" {
		t.Fatalf("acp provider = %#v", providers[1])
	}

	if _, err := DefaultProviderFactory(config.ResolvedLock{Components: []config.LockComponent{
		providerComponent("codex-acp", config.KindACPAdapter),
	}}); err == nil || !strings.Contains(err.Error(), "node") {
		t.Fatalf("DefaultProviderFactory() without node = %v", err)
	}
}

func assertNoBuildTemp(t *testing.T, root string, adapter config.LockComponent) {
	t.Helper()
	dir := filepath.Dir(filepath.Join(root, filepath.FromSlash(adapter.Artifact.Path)))
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			t.Fatalf("build temp leaked: %s", e.Name())
		}
	}
}
