package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	toolconfig "github.com/karohani/imoogi-emacs/internal/config"
	toolsetup "github.com/karohani/imoogi-emacs/internal/setup"
)

const (
	ExitOK              = 0
	ExitError           = 1
	ExitUsage           = 2
	ExitConfigIntegrity = 10
	ExitSetupBusy       = 11
	ExitSetupProbe      = 12
	ExitSetup           = 13
	defaultVersion      = "dev"
	notConfigured       = "not configured"
	commandName         = "imoogi-toolchain"
	defaultHelpText     = `imoogi-toolchain manages imoogi-emacs toolchain artifacts.

Usage:
  imoogi-toolchain <command> [options]

Commands:
  fetch    Fetch artifacts on an online build machine
  setup    Set up artifacts in an offline target environment
  version  Report CLI, desired, available, and active versions

Options:
  -h, --help  Show help
`
)

type Config struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Version string
	Workdir string
	Hooks   Hooks
}

type Hooks struct {
	Fetch Action
	Setup Action
}

type Action func(context.Context, IO) error

type IO struct {
	Stdout io.Writer
	Stderr io.Writer
}

func Run(args []string, cfg Config) int {
	r := runner{
		stdout:  cfg.Stdout,
		stderr:  cfg.Stderr,
		version: cfg.Version,
		workdir: cfg.Workdir,
		hooks:   cfg.Hooks,
	}
	r.setDefaults()
	return r.run(args)
}

type runner struct {
	stdout  io.Writer
	stderr  io.Writer
	version string
	workdir string
	hooks   Hooks
}

func (r *runner) setDefaults() {
	if r.stdout == nil {
		r.stdout = io.Discard
	}
	if r.stderr == nil {
		r.stderr = io.Discard
	}
	if r.version == "" {
		r.version = defaultVersion
	}
	if r.workdir == "" {
		r.workdir = "."
	}
}

func (r runner) run(args []string) int {
	if len(args) == 0 || isHelp(args[0]) {
		return r.writeHelp(defaultHelpText)
	}

	switch args[0] {
	case "fetch":
		return r.runFetch(args[1:])
	case "setup":
		return r.runAction("setup", args[1:], r.hooks.Setup)
	case "version":
		return r.runVersion(args[1:])
	default:
		fmt.Fprintf(r.stderr, "%s: unknown command %q\n", commandName, args[0])
		fmt.Fprintf(r.stderr, "Run '%s --help' for usage.\n", commandName)
		return ExitUsage
	}
}

func (r runner) runFetch(args []string) int {
	if len(args) > 0 {
		if isHelp(args[0]) {
			return r.writeHelp(actionHelp("fetch"))
		}
		if args[0] == "--dry-run" {
			if len(args) > 1 {
				return r.unexpectedArg("fetch", args[1])
			}
			return r.runFetchDryRun()
		}
		return r.unexpectedArg("fetch", args[0])
	}
	return r.runAction("fetch", args, r.hooks.Fetch)
}

func (r runner) runFetchDryRun() int {
	manifest, err := toolconfig.LoadDesired(filepath.Join(r.workdir, "toolchains.json"))
	if err != nil {
		fmt.Fprintf(r.stderr, "%s fetch: %v\n", commandName, err)
		return ExitError
	}
	lock, err := toolconfig.LoadLock(filepath.Join(r.workdir, "toolchains.lock.json"))
	if err != nil {
		fmt.Fprintf(r.stderr, "%s fetch: %v\n", commandName, err)
		return ExitError
	}
	plan, err := toolconfig.BuildPlan(*manifest, *lock)
	if err != nil {
		fmt.Fprintf(r.stderr, "%s fetch: %v\n", commandName, err)
		return ExitError
	}
	writeFetchDryRunPlan(r.stdout, plan)
	return ExitOK
}

func (r runner) runVersion(args []string) int {
	if len(args) > 0 {
		if isHelp(args[0]) {
			return r.writeHelp(versionHelp())
		}
		return r.unexpectedArg("version", args[0])
	}
	if err := r.writeVersionReport(); err != nil {
		fmt.Fprintf(r.stderr, "%s version: %v\n", commandName, err)
		return ExitConfigIntegrity
	}
	return ExitOK
}

func (r runner) runAction(name string, args []string, action Action) int {
	if len(args) > 0 {
		if isHelp(args[0]) {
			return r.writeHelp(actionHelp(name))
		}
		return r.unexpectedArg(name, args[0])
	}
	if action == nil {
		fmt.Fprintf(r.stderr, "%s %s: %s\n", commandName, name, notConfigured)
		return ExitError
	}
	if err := action(context.Background(), IO{Stdout: r.stdout, Stderr: r.stderr}); err != nil {
		fmt.Fprintf(r.stderr, "%s %s: %v\n", commandName, name, err)
		return exitCodeFor(err)
	}
	return ExitOK
}

func exitCodeFor(err error) int {
	var setupErr *toolsetup.Error
	if !errors.As(err, &setupErr) {
		return ExitError
	}
	switch setupErr.Kind {
	case toolsetup.KindConfigIntegrity:
		return ExitConfigIntegrity
	case toolsetup.KindBusy:
		return ExitSetupBusy
	case toolsetup.KindProbe:
		return ExitSetupProbe
	case toolsetup.KindSetup:
		return ExitSetup
	default:
		return ExitError
	}
}

func (r runner) unexpectedArg(command, arg string) int {
	fmt.Fprintf(r.stderr, "%s %s: unexpected argument %q\n", commandName, command, arg)
	fmt.Fprintf(r.stderr, "Run '%s %s --help' for usage.\n", commandName, command)
	return ExitUsage
}

func (r runner) writeHelp(text string) int {
	fmt.Fprint(r.stdout, text)
	return ExitOK
}

func writeFetchDryRunPlan(w io.Writer, plan toolconfig.Plan) {
	fmt.Fprintln(w, "fetch dry-run plan")
	fmt.Fprintf(w, "cli_version: %s\n", plan.CLI)
	fmt.Fprintf(w, "bundle: %s\n", plan.Bundle)
	fmt.Fprintf(w, "target: %s/%s\n", plan.Target.OS, plan.Target.Arch)
	fmt.Fprintln(w, "planned_mutations:")
	fmt.Fprintln(w, "  - validate toolchains.json")
	fmt.Fprintln(w, "  - validate toolchains.lock.json")
	fmt.Fprintln(w, "  - prepare vendor/toolchains artifacts")
	fmt.Fprintln(w, "  - update deterministic lock/provenance/license records")
	fmt.Fprintln(w, "network: disabled in dry-run")
	fmt.Fprintln(w, "build: disabled in dry-run")
	fmt.Fprintln(w, "activation: none")
	fmt.Fprintln(w, "components:")
	for _, component := range plan.Components {
		fmt.Fprintf(w, "  - %s %s %s %s size=%d sha256=%s\n",
			component.Name,
			component.Kind,
			component.UpstreamVersion,
			component.ArtifactPath,
			component.Size,
			component.SHA256,
		)
	}
}

func (r runner) writeVersionReport() error {
	fmt.Fprintf(r.stdout, "cli_version: %s\n", r.version)

	desiredPath := filepath.Join(r.workdir, "toolchains.json")
	desired, desiredErr := toolconfig.LoadDesired(desiredPath)
	switch {
	case desiredErr == nil:
		fmt.Fprintf(r.stdout, "desired_bundle: %s\n", desired.Bundle)
	case errors.Is(desiredErr, os.ErrNotExist):
		fmt.Fprintln(r.stdout, "desired_bundle: not present")
	default:
		return desiredErr
	}

	lockPath := filepath.Join(r.workdir, "toolchains.lock.json")
	lock, lockErr := toolconfig.LoadLock(lockPath)
	switch {
	case lockErr == nil:
		fmt.Fprintf(r.stdout, "available_bundle: %s\n", lock.Bundle)
		fmt.Fprintln(r.stdout, "components:")
		for _, component := range lock.Components {
			fmt.Fprintf(r.stdout, "  - %s %s %s\n", component.Name, component.Kind, component.UpstreamVersion)
		}
	case errors.Is(lockErr, os.ErrNotExist):
		fmt.Fprintln(r.stdout, "available_bundle: not present")
		fmt.Fprintln(r.stdout, "components: not present")
	default:
		return lockErr
	}

	active, err := activeBundle(filepath.Join(r.workdir, ".local", "bin"))
	if err != nil {
		return err
	}
	fmt.Fprintf(r.stdout, "active_bundle: %s\n", active)
	return nil
}

func activeBundle(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "not active", nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return "not active", nil
	}
	target, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	clean := filepath.ToSlash(filepath.Clean(target))
	if filepath.IsAbs(target) || clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "not active", nil
	}
	parts := strings.Split(clean, "/")
	if len(parts) != 3 || parts[0] != "toolchains" || parts[1] == "" || parts[2] != "bin" {
		return "not active", nil
	}
	return parts[1], nil
}

func isHelp(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}

func actionHelp(name string) string {
	switch name {
	case "fetch":
		return fmt.Sprintf(`Usage:
  %s fetch [options]

Fetch toolchain artifacts on an online build machine. This command is not for offline target environments.

Showing this help does not read manifests, inspect tools, or access the network.

Options:
      --dry-run  Validate manifests and print planned fetch mutations without network or build actions
  -h, --help     Show help
`, commandName)
	case "setup":
		return fmt.Sprintf(`Usage:
  %s setup [options]

Set up previously fetched toolchain artifacts in an offline target environment.

Showing this help does not read manifests, inspect tools, or access the network.

Options:
  -h, --help  Show help
`, commandName)
	}
	return fmt.Sprintf(`Usage:
  %s %s [options]

Showing this help does not read manifests, inspect tools, or access the network.

Options:
  -h, --help  Show help
`, commandName, name)
}

func versionHelp() string {
	return fmt.Sprintf(`Usage:
  %s version [options]

Print CLI, available bundle, active bundle, and upstream component versions.

Showing this help does not read manifests, inspect tools, or access the network.

Options:
  -h, --help  Show help
`, commandName)
}
