package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	toolsetup "github.com/karohani/imoogi-emacs/internal/setup"
)

func TestHelpCommandsExitZeroWithoutStderr(t *testing.T) {
	tests := [][]string{
		nil,
		{"--help"},
		{"-h"},
		{"help"},
		{"fetch", "--help"},
		{"setup", "--help"},
		{"version", "--help"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(args, Config{Stdout: &stdout, Stderr: &stderr})
			if code != ExitOK {
				t.Fatalf("exit code = %d, want %d", code, ExitOK)
			}
			if stdout.Len() == 0 {
				t.Fatal("stdout is empty")
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRootHelpDescribesFetchAndSetupEnvironments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--help"}, Config{Stdout: &stdout, Stderr: &stderr})

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	want := `imoogi-toolchain manages imoogi-emacs toolchain artifacts.

Usage:
  imoogi-toolchain <command> [options]

Commands:
  fetch    Fetch artifacts on an online build machine
  setup    Set up artifacts in an offline target environment
  version  Report CLI, desired, available, and active versions

Options:
  -h, --help  Show help
`
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestFetchAndSetupHelpDescribeEnvironmentContracts(t *testing.T) {
	tests := map[string]string{
		"fetch": `Usage:
  imoogi-toolchain fetch [options]

Fetch toolchain artifacts on an online build machine. This command is not for offline target environments.

Showing this help does not read manifests, inspect tools, or access the network.

Options:
      --dry-run  Validate manifests and print planned fetch mutations without network or build actions
  -h, --help     Show help
`,
		"setup": `Usage:
  imoogi-toolchain setup [options]

Set up previously fetched toolchain artifacts in an offline target environment.

Showing this help does not read manifests, inspect tools, or access the network.

Options:
  -h, --help  Show help
`,
	}

	for command, want := range tests {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run([]string{command, "--help"}, Config{Stdout: &stdout, Stderr: &stderr})

			if code != ExitOK {
				t.Fatalf("exit code = %d, want %d", code, ExitOK)
			}
			if got := stdout.String(); got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestFetchDryRunPrintsOfflinePlanFromManifestAndLock(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, filepath.Join("..", "config", "testdata", "toolchains.json"), filepath.Join(root, "toolchains.json"))
	copyFixture(t, filepath.Join("..", "config", "testdata", "toolchains.lock.json"), filepath.Join(root, "toolchains.lock.json"))

	var stdout, stderr bytes.Buffer
	code := Run([]string{"fetch", "--dry-run"}, Config{
		Stdout:  &stdout,
		Stderr:  &stderr,
		Workdir: root,
		Hooks: Hooks{
			Fetch: func(context.Context, IO) error {
				t.Fatal("fetch hook must not run during dry-run")
				return nil
			},
		},
	})

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	wantContains := []string{
		"fetch dry-run plan\n",
		"cli_version: 1.2.3\n",
		"bundle: 2026.08.15.1\n",
		"target: darwin/arm64\n",
		"network: disabled in dry-run\n",
		"build: disabled in dry-run\n",
		"activation: none\n",
		"  - gopls go-language-server v0.20.0 vendor/toolchains/go/gopls/v0.20.0/darwin-arm64/gopls.tar.gz size=123 sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n",
		"  - typescript-language-server typescript-language-server 4.3.3 vendor/toolchains/typescript/typescript-language-server/4.3.3/darwin-arm64/server.tgz size=321 sha256=dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd\n",
	}
	for _, want := range wantContains {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestFetchDryRunRejectsInvalidManifestLockPair(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, filepath.Join("..", "config", "testdata", "toolchains.json"), filepath.Join(root, "toolchains.json"))
	copyFixture(t, filepath.Join("..", "config", "testdata", "toolchains.lock.json"), filepath.Join(root, "toolchains.lock.json"))
	data, err := os.ReadFile(filepath.Join(root, "toolchains.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.ReplaceAll(data, []byte(`"bundle": "2026.08.15.1"`), []byte(`"bundle": "2026.08.16.1"`))
	if err := os.WriteFile(filepath.Join(root, "toolchains.lock.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"fetch", "--dry-run"}, Config{Stdout: &stdout, Stderr: &stderr, Workdir: root})

	if code != ExitError {
		t.Fatalf("exit code = %d, want %d", code, ExitError)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "bundle mismatch") {
		t.Fatalf("stderr = %q, want bundle mismatch", stderr.String())
	}
}

func TestVersionPrintsConfiguredVersionAndRepositoryState(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, filepath.Join("..", "config", "testdata", "toolchains.json"), filepath.Join(root, "toolchains.json"))
	copyFixture(t, filepath.Join("..", "config", "testdata", "toolchains.lock.json"), filepath.Join(root, "toolchains.lock.json"))
	if err := os.MkdirAll(filepath.Join(root, ".local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.FromSlash("toolchains/2026.08.15.1/bin"), filepath.Join(root, ".local", "bin")); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, Config{
		Stdout:  &stdout,
		Stderr:  &stderr,
		Version: "test-version",
		Workdir: root,
	})

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	wantContains := []string{
		"cli_version: test-version\n",
		"desired_bundle: 2026.08.15.1\n",
		"available_bundle: 2026.08.15.1\n",
		"active_bundle: 2026.08.15.1\n",
		"  - gopls go-language-server v0.20.0\n",
		"  - typescript-language-server typescript-language-server 4.3.3\n",
	}
	for _, want := range wantContains {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestVersionWorksBeforeManifestsExist(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, Config{
		Stdout:  &stdout,
		Stderr:  &stderr,
		Version: "1.0.0",
		Workdir: t.TempDir(),
	})

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	want := "cli_version: 1.0.0\ndesired_bundle: not present\navailable_bundle: not present\ncomponents: not present\nactive_bundle: not active\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestVersionRejectsMalformedLockWithoutMutation(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, filepath.Join("..", "config", "testdata", "toolchains.json"), filepath.Join(root, "toolchains.json"))
	if err := os.WriteFile(filepath.Join(root, "toolchains.lock.json"), []byte(`{"schema":"bad"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, Config{Stdout: &stdout, Stderr: &stderr, Workdir: root})

	if code != ExitConfigIntegrity {
		t.Fatalf("exit code = %d, want %d", code, ExitConfigIntegrity)
	}
	if !strings.Contains(stdout.String(), "cli_version: dev\n") || !strings.Contains(stdout.String(), "desired_bundle: 2026.08.15.1\n") {
		t.Fatalf("stdout = %q, want partial read-only report", stdout.String())
	}
	if !strings.Contains(stderr.String(), "imoogi-toolchain version:") || !strings.Contains(stderr.String(), "unsupported") {
		t.Fatalf("stderr = %q, want malformed lock error", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".local")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("version created .local or unexpected stat error: %v", err)
	}
}

func TestUnknownCommandUsesStableUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"bogus"}, Config{Stdout: &stdout, Stderr: &stderr})

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	want := "imoogi-toolchain: unknown command \"bogus\"\nRun 'imoogi-toolchain --help' for usage.\n"
	if got := stderr.String(); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestFetchAndSetupDefaultToNotConfigured(t *testing.T) {
	for _, command := range []string{"fetch", "setup"} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run([]string{command}, Config{Stdout: &stdout, Stderr: &stderr})

			if code != ExitError {
				t.Fatalf("exit code = %d, want %d", code, ExitError)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			want := "imoogi-toolchain " + command + ": not configured\n"
			if got := stderr.String(); got != want {
				t.Fatalf("stderr = %q, want %q", got, want)
			}
		})
	}
}

func TestFetchAndSetupUseInjectedHooks(t *testing.T) {
	tests := map[string]struct {
		args []string
		cfg  Config
	}{
		"fetch": {
			args: []string{"fetch"},
			cfg: Config{Hooks: Hooks{
				Fetch: func(_ context.Context, io IO) error {
					_, err := io.Stdout.Write([]byte("fetch ok\n"))
					return err
				},
			}},
		},
		"setup": {
			args: []string{"setup"},
			cfg: Config{Hooks: Hooks{
				Setup: func(_ context.Context, io IO) error {
					_, err := io.Stdout.Write([]byte("setup ok\n"))
					return err
				},
			}},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			tt.cfg.Stdout = &stdout
			tt.cfg.Stderr = &stderr
			code := Run(tt.args, tt.cfg)
			if code != ExitOK {
				t.Fatalf("exit code = %d, want %d", code, ExitOK)
			}
			if got, want := stdout.String(), name+" ok\n"; got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestHookErrorsUseStableCommandError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"fetch"}, Config{
		Stdout: &stdout,
		Stderr: &stderr,
		Hooks: Hooks{
			Fetch: func(context.Context, IO) error {
				return errors.New("fixture failure")
			},
		},
	})

	if code != ExitError {
		t.Fatalf("exit code = %d, want %d", code, ExitError)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	want := "imoogi-toolchain fetch: fixture failure\n"
	if got := stderr.String(); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestSetupErrorsUseClassifiedExitCodes(t *testing.T) {
	tests := map[string]struct {
		err  error
		code int
	}{
		"config": {err: &toolsetup.Error{Kind: toolsetup.KindConfigIntegrity, Err: errors.New("bad lock")}, code: ExitConfigIntegrity},
		"busy":   {err: &toolsetup.Error{Kind: toolsetup.KindBusy, Err: errors.New("setup already running")}, code: ExitSetupBusy},
		"probe":  {err: &toolsetup.Error{Kind: toolsetup.KindProbe, Err: errors.New("probe failed")}, code: ExitSetupProbe},
		"setup":  {err: &toolsetup.Error{Kind: toolsetup.KindSetup, Err: errors.New("setup failed")}, code: ExitSetup},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run([]string{"setup"}, Config{
				Stdout: &stdout,
				Stderr: &stderr,
				Hooks: Hooks{Setup: func(context.Context, IO) error {
					return tt.err
				}},
			})
			if code != tt.code {
				t.Fatalf("exit code = %d, want %d", code, tt.code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.err.Error()) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.err.Error())
			}
		})
	}
}

func TestDarwinArm64BuildIsConfigured(t *testing.T) {
	root := repoRoot(t)
	output := filepath.Join(t.TempDir(), "imoogi-toolchain")
	cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags=-buildid=", "-o", output, "./cmd/imoogi-toolchain")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH=arm64", "CGO_ENABLED=0")

	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, combined)
	}
	if info, err := os.Stat(output); err != nil {
		t.Fatalf("built binary missing: %v", err)
	} else if info.Size() == 0 {
		t.Fatal("built binary is empty")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func copyFixture(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
