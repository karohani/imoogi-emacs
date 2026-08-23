package setup

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
	"testing"

	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang"
)

const lockHelperEnv = "IMOOGI_SETUP_LOCK_HELPER"

func TestSetupActivatesRelativeLinkFromNestedDirectoryAndIsIdempotent(t *testing.T) {
	root := makeRepo(t, "repo with spaces", "one", "two")
	nested := filepath.Join(root, "a", "b")
	mustMkdir(t, nested)
	var probes []lang.Probe

	for i := 0; i < 2; i++ {
		var stdout bytes.Buffer
		err := Run(context.Background(), Options{
			Workdir:         nested,
			Stdout:          &stdout,
			ProviderFactory: fakeProviderFactory(t, nil),
			RunProbe: func(_ context.Context, probe lang.Probe, bundleRoot, probeRoot string) error {
				if !filepath.IsAbs(probe.Command) {
					t.Fatalf("probe command is not absolute: %q", probe.Command)
				}
				if !strings.HasPrefix(probe.Command, bundleRoot+string(os.PathSeparator)) {
					t.Fatalf("probe %q outside bundle root %q", probe.Command, bundleRoot)
				}
				if strings.HasPrefix(probeRoot, bundleRoot+string(os.PathSeparator)) {
					t.Fatalf("probe temp %q must be outside bundle root %q", probeRoot, bundleRoot)
				}
				probes = append(probes, probe)
				return nil
			},
		})
		if err != nil {
			t.Fatalf("setup %d failed: %v", i+1, err)
		}
		if i == 0 && !strings.Contains(stdout.String(), "setup activated 2026.08.22.1") {
			t.Fatalf("stdout = %q", stdout.String())
		}
		if i == 1 && !strings.Contains(stdout.String(), "setup reused 2026.08.22.1") {
			t.Fatalf("stdout = %q", stdout.String())
		}
	}

	link := filepath.Join(root, ".local", "bin")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("read activation link: %v", err)
	}
	if target != "toolchains/2026.08.22.1/bin" {
		t.Fatalf("activation link = %q", target)
	}
	if len(probes) != 4 {
		t.Fatalf("probe count = %d, want 4", len(probes))
	}
	assertNoTransactionTemp(t, root)
}

func TestCorruptLaterArtifactPreventsProviderCallsAndMutations(t *testing.T) {
	root := makeRepo(t, "repo", "one", "two")
	if err := os.WriteFile(filepath.Join(root, "vendor", "two.bin"), []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	var calls int
	err := Run(context.Background(), Options{
		Workdir: root,
		ProviderFactory: func(config.ResolvedLock) ([]lang.Provider, error) {
			calls++
			return fakeProviderFactory(t, nil)(config.ResolvedLock{})
		},
	})
	if kindOf(err) != KindConfigIntegrity {
		t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindConfigIntegrity)
	}
	if calls != 0 {
		t.Fatalf("provider calls = %d, want 0", calls)
	}
	if _, err := os.Lstat(filepath.Join(root, ".local", "bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("activation exists after corrupt artifact: %v", err)
	}
	assertNoTransactionTemp(t, root)
}

func TestBusyLockHasStableErrorAndNoMutation(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	local := filepath.Join(root, ".local")
	mustMkdir(t, local)
	release, err := acquireLock(filepath.Join(local, "setup.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	err = Run(context.Background(), Options{
		Workdir: root,
		ProviderFactory: func(config.ResolvedLock) ([]lang.Provider, error) {
			t.Fatal("provider must not run while lock is busy")
			return nil, nil
		},
	})
	if kindOf(err) != KindBusy {
		t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindBusy)
	}
	if !strings.Contains(err.Error(), "setup already running") {
		t.Fatalf("error = %q, want stable busy text", err.Error())
	}
	if _, err := os.Lstat(filepath.Join(local, "bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("activation mutated while busy: %v", err)
	}
	assertNoTransactionTemp(t, root)
}

func TestFailpointsPreservePreviousActivationAndCleanupStaging(t *testing.T) {
	failpoints := map[string]func(*Failpoints){
		"after-staging": func(fp *Failpoints) { fp.AfterStaging = func() error { return errors.New("boom") } },
		"after-probes":  func(fp *Failpoints) { fp.AfterProbes = func() error { return errors.New("boom") } },
		"after-publish": func(fp *Failpoints) { fp.AfterPublish = func() error { return errors.New("boom") } },
	}
	for name, configure := range failpoints {
		t.Run(name, func(t *testing.T) {
			root := makeRepo(t, "repo", "one")
			installPrevious(t, root)
			var fp Failpoints
			configure(&fp)
			err := Run(context.Background(), Options{
				Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe, Failpoints: fp,
			})
			if kindOf(err) != KindSetup {
				t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindSetup)
			}
			assertPreviousActivation(t, root)
			assertNoTransactionTemp(t, root)
		})
	}
}

func TestFailureAfterPublishAllowsReuseOnRerun(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	installPrevious(t, root)
	err := Run(context.Background(), Options{
		Workdir:         root,
		ProviderFactory: fakeProviderFactory(t, nil),
		RunProbe:        okProbe,
		Failpoints: Failpoints{BeforeActivate: func() error {
			return errors.New("stop before activation")
		}},
	})
	if kindOf(err) != KindSetup {
		t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindSetup)
	}
	if _, err := os.Stat(filepath.Join(root, ".local", "toolchains", "2026.08.22.1", "install.json")); err != nil {
		t.Fatalf("published bundle not usable: %v", err)
	}
	assertPreviousActivation(t, root)
	err = Run(context.Background(), Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe})
	if err != nil {
		t.Fatalf("rerun after publish failed: %v", err)
	}
	assertActiveBundle(t, root, "2026.08.22.1")
}

func TestDefaultProbeRunnerDoesNotMutateBundleOnReuse(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	err := Run(context.Background(), Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil)})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	bundle := filepath.Join(root, ".local", "toolchains", "2026.08.22.1")
	before := snapshotTree(t, bundle)
	err = Run(context.Background(), Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil)})
	if err != nil {
		t.Fatalf("reuse setup failed: %v", err)
	}
	after := snapshotTree(t, bundle)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("bundle mutated on reuse:\nbefore=%v\nafter=%v", before, after)
	}
	assertNoTransactionTemp(t, root)
}

func TestInstalledBundleReuseStillVerifiesCommittedArtifacts(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	err := Run(context.Background(), Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe})
	if err != nil {
		t.Fatalf("initial setup failed: %v", err)
	}
	assertActiveBundle(t, root, "2026.08.22.1")
	if err := os.WriteFile(filepath.Join(root, "vendor", "one.bin"), []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = Run(context.Background(), Options{
		Workdir: root,
		ProviderFactory: func(config.ResolvedLock) ([]lang.Provider, error) {
			t.Fatal("provider must not run when committed artifact is corrupt")
			return nil, nil
		},
		RunProbe: okProbe,
	})
	if kindOf(err) != KindConfigIntegrity {
		t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindConfigIntegrity)
	}
	assertActiveBundle(t, root, "2026.08.22.1")
}

func TestInstalledBundleReuseRejectsTreeMutation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{
			name: "executable content",
			mutate: func(t *testing.T, bundle string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(bundle, "bin", "one"), []byte("#!/bin/sh\nexit 7\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "executable mode",
			mutate: func(t *testing.T, bundle string) {
				t.Helper()
				if err := os.Chmod(filepath.Join(bundle, "bin", "one"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "executable symlink",
			mutate: func(t *testing.T, bundle string) {
				t.Helper()
				path := filepath.Join(bundle, "bin", "one")
				outside := filepath.Join(t.TempDir(), "outside")
				if err := os.WriteFile(outside, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "added directory content",
			mutate: func(t *testing.T, bundle string) {
				t.Helper()
				path := filepath.Join(bundle, "lib", "one", "data.txt")
				mustMkdir(t, filepath.Dir(path))
				if err := os.WriteFile(path, []byte("tampered\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := makeRepo(t, "repo", "one")
			if err := Run(context.Background(), Options{
				Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe,
			}); err != nil {
				t.Fatalf("initial setup failed: %v", err)
			}
			bundle := filepath.Join(root, ".local", "toolchains", "2026.08.22.1")
			tc.mutate(t, bundle)

			err := Run(context.Background(), Options{
				Workdir: root,
				ProviderFactory: func(config.ResolvedLock) ([]lang.Provider, error) {
					t.Fatal("provider must not run for a tampered installed bundle")
					return nil, nil
				},
				RunProbe: func(context.Context, lang.Probe, string, string) error {
					t.Fatal("probe must not run for a tampered installed bundle")
					return nil
				},
			})
			if kindOf(err) != KindConfigIntegrity || !strings.Contains(err.Error(), "content integrity") {
				t.Fatalf("error = %v, kind %q; want content integrity failure", err, kindOf(err))
			}
			assertActiveBundle(t, root, "2026.08.22.1")
		})
	}
}

func TestRejectsLocalSymlinkWithoutOutsideWrites(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	outside := filepath.Join(t.TempDir(), "outside")
	mustMkdir(t, outside)
	if err := os.Symlink(outside, filepath.Join(root, ".local")); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe})
	if kindOf(err) != KindSetup {
		t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindSetup)
	}
	assertEmptyDir(t, outside)
}

func TestRejectsToolchainsSymlinkWithoutOutsideWrites(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	outside := filepath.Join(t.TempDir(), "outside")
	mustMkdir(t, outside)
	local := filepath.Join(root, ".local")
	mustMkdir(t, local)
	if err := os.Symlink(outside, filepath.Join(local, "toolchains")); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe})
	if kindOf(err) != KindSetup {
		t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindSetup)
	}
	assertEmptyDir(t, outside)
}

func TestRejectsSetupLockSymlinkWithoutTruncatingOutsideTarget(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	local := filepath.Join(root, ".local")
	mustMkdir(t, local)
	outside := filepath.Join(t.TempDir(), "outside-lock")
	const sentinel = "do-not-truncate"
	if err := os.WriteFile(outside, []byte(sentinel), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(local, "setup.lock")); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe})
	if kindOf(err) != KindSetup {
		t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindSetup)
	}
	data, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != sentinel {
		t.Fatalf("outside lock target mutated: %q", data)
	}
}

func TestRejectsSymlinkedRootManifests(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	outsideManifest := filepath.Join(t.TempDir(), "toolchains.json")
	data, err := os.ReadFile(filepath.Join(root, "toolchains.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outsideManifest, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "toolchains.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideManifest, filepath.Join(root, "toolchains.json")); err != nil {
		t.Fatal(err)
	}
	err = Run(context.Background(), Options{Workdir: root, ProviderFactory: fakeProviderFactory(t, nil), RunProbe: okProbe})
	if kindOf(err) != KindConfigIntegrity {
		t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindConfigIntegrity)
	}
}

func TestLockFileIsReusableAfterRelease(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	local := filepath.Join(root, ".local")
	mustMkdir(t, local)
	lockPath := filepath.Join(local, "setup.lock")
	release, err := acquireLock(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	release()
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("persistent lock file missing: %v", err)
	}
	release, err = acquireLock(lockPath)
	if err != nil {
		t.Fatalf("lock did not recover after release: %v", err)
	}
	release()
}

func TestFlockRecoversAfterSubprocessExitWithoutRelease(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	local := filepath.Join(root, ".local")
	mustMkdir(t, local)
	lockPath := filepath.Join(local, "setup.lock")
	cmd := exec.Command(os.Args[0], "-test.run=TestFlockChildHelper")
	cmd.Env = append(os.Environ(), lockHelperEnv+"="+lockPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("lock helper failed: %v\n%s", err, output)
	}
	release, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("parent could not reacquire lock after child exit: %v", err)
	}
	release()
}

func TestFlockChildHelper(t *testing.T) {
	lockPath := os.Getenv(lockHelperEnv)
	if lockPath == "" {
		return
	}
	_, err := acquireLock(lockPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func TestExistingBundleConflictFailsClosed(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	bundle := filepath.Join(root, ".local", "toolchains", "2026.08.22.1")
	mustMkdir(t, bundle)
	if err := os.WriteFile(filepath.Join(bundle, "install.json"), []byte(`{"schema":"imoogi-toolchain-install/v2","lock_sha256":"wrong","bundle_sha256":"0000000000000000000000000000000000000000000000000000000000000000"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), Options{Workdir: root})
	if kindOf(err) != KindConfigIntegrity {
		t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindConfigIntegrity)
	}
	if _, err := os.Lstat(filepath.Join(root, ".local", "bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("activation exists after conflict: %v", err)
	}
}

func TestProbeFailureDoesNotActivate(t *testing.T) {
	root := makeRepo(t, "repo", "one")
	err := Run(context.Background(), Options{
		Workdir:         root,
		ProviderFactory: fakeProviderFactory(t, nil),
		RunProbe:        func(context.Context, lang.Probe, string, string) error { return errors.New("probe failed") },
	})
	if kindOf(err) != KindProbe {
		t.Fatalf("error = %v, kind %q; want %s", err, kindOf(err), KindProbe)
	}
	if _, err := os.Lstat(filepath.Join(root, ".local", "bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("activation exists after probe failure: %v", err)
	}
	assertNoTransactionTemp(t, root)
}

type fakeProvider struct {
	components []config.LockComponent
	onRun      func()
	t          *testing.T
}

func (p fakeProvider) Materialize(_ string, stagingRoot string) ([]lang.Probe, error) {
	if p.onRun != nil {
		p.onRun()
	}
	probes := make([]lang.Probe, 0, len(p.components))
	for _, component := range p.components {
		command, err := lang.ContainedStagePath(stagingRoot, component.Probe.Command)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(command), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(command, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			return nil, err
		}
		probes = append(probes, lang.Probe{Command: command, Args: append([]string(nil), component.Probe.Args...)})
	}
	return probes, nil
}

func fakeProviderFactory(t *testing.T, onRun func()) ProviderFactory {
	t.Helper()
	return func(lock config.ResolvedLock) ([]lang.Provider, error) {
		return []lang.Provider{fakeProvider{components: lock.Components, onRun: onRun, t: t}}, nil
	}
}

func okProbe(context.Context, lang.Probe, string, string) error { return nil }

func makeRepo(t *testing.T, name string, components ...string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	mustMkdir(t, filepath.Join(root, "vendor"))
	lockComponents := make([]config.LockComponent, 0, len(components))
	desiredComponents := make([]config.DesiredComponent, 0, len(components))
	for _, name := range components {
		artifactPath := filepath.ToSlash(filepath.Join("vendor", name+".bin"))
		data := []byte("artifact-" + name)
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(artifactPath)), data, 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		component := config.LockComponent{
			Name: name, Kind: "fixture", Source: "fixture", UpstreamVersion: "1.0.0",
			Target: config.Target{OS: "darwin", Arch: "arm64"},
			Artifact: config.Artifact{
				Path: artifactPath, Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:]),
				SourceURL: "https://example.invalid/" + name, RetrievedAt: "2026-08-22T00:00:00Z",
			},
			License:    config.License{Path: artifactPath},
			Provenance: config.Provenance{Builder: "test", Toolchain: "test", Command: "test"},
			Install:    []config.InstallEntry{{Path: "bin/" + name, Mode: "executable"}},
			Probe:      config.Probe{Command: "bin/" + name, Args: []string{"--version"}},
		}
		lockComponents = append(lockComponents, component)
		desiredComponents = append(desiredComponents, config.DesiredComponent{
			Name: name, Kind: component.Kind, Source: component.Source, UpstreamVersion: component.UpstreamVersion,
		})
	}
	writeJSON(t, filepath.Join(root, "toolchains.json"), config.DesiredManifest{
		Schema: config.DesiredSchema, CLI: "1.0.0", Bundle: "2026.08.22.1",
		Target: config.Target{OS: "darwin", Arch: "arm64"}, Components: desiredComponents,
	})
	writeJSON(t, filepath.Join(root, "toolchains.lock.json"), config.ResolvedLock{
		Schema: config.LockSchema, CLI: "1.0.0", Bundle: "2026.08.22.1",
		Target: config.Target{OS: "darwin", Arch: "arm64"}, Components: lockComponents,
	})
	return root
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func installPrevious(t *testing.T, root string) {
	t.Helper()
	prevBin := filepath.Join(root, ".local", "toolchains", "previous", "bin")
	mustMkdir(t, prevBin)
	if err := os.WriteFile(filepath.Join(prevBin, "sentinel"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(root, ".local")
	if err := os.Symlink("toolchains/previous/bin", filepath.Join(local, "bin")); err != nil {
		t.Fatal(err)
	}
}

func assertPreviousActivation(t *testing.T, root string) {
	t.Helper()
	target, err := os.Readlink(filepath.Join(root, ".local", "bin"))
	if err != nil {
		t.Fatalf("read previous activation: %v", err)
	}
	if target != "toolchains/previous/bin" {
		t.Fatalf("activation = %q, want previous", target)
	}
	if _, err := os.Stat(filepath.Join(root, ".local", "bin", "sentinel")); err != nil {
		t.Fatalf("previous executable unavailable: %v", err)
	}
}

func assertActiveBundle(t *testing.T, root, bundle string) {
	t.Helper()
	target, err := os.Readlink(filepath.Join(root, ".local", "bin"))
	if err != nil {
		t.Fatalf("read activation: %v", err)
	}
	want := fmt.Sprintf("toolchains/%s/bin", bundle)
	if target != want {
		t.Fatalf("activation = %q, want %q", target, want)
	}
}

func snapshotTree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, fmt.Sprintf("%s %s %d", filepath.ToSlash(rel), info.Mode().String(), info.Size()))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func assertNoTransactionTemp(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, ".local"))
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".staging-") || strings.HasPrefix(entry.Name(), ".probe-") {
			t.Fatalf("transaction temp leaked: %s", entry.Name())
		}
	}
}

func assertEmptyDir(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("%s not empty: %v", path, entries)
	}
}

func kindOf(err error) Kind {
	var setupErr *Error
	if errors.As(err, &setupErr) {
		return setupErr.Kind
	}
	return ""
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
