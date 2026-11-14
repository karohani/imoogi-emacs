// Package acp installs Agent Client Protocol adapters into the setup bundle.
// Each adapter artifact is a packed node_modules tree; the provider unpacks it
// under lib/acp/<name> and writes bin/<name>, a launcher that runs the
// adapter's bin script on the bundled node.
package acp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang"
	"github.com/karohani/imoogi-emacs/internal/lang/node"
	"github.com/karohani/imoogi-emacs/internal/npmbundle"
)

// Adapter is what an adapter needs from outside the bundle: the agent CLI it
// drives, and the variable through which it is told where that CLI is.
type Adapter struct {
	Name   string
	CLI    string
	EnvVar string
}

// Adapters lists every adapter the provider knows how to launch.
var Adapters = []Adapter{
	{Name: "claude-agent-acp", CLI: "claude", EnvVar: "CLAUDE_CODE_EXECUTABLE"},
	{Name: "codex-acp", CLI: "codex", EnvVar: "CODEX_PATH"},
}

// Lookup returns the adapter named name.
func Lookup(name string) (Adapter, bool) {
	for _, a := range Adapters {
		if a.Name == name {
			return a, true
		}
	}
	return Adapter{}, false
}

// InstallDir is where an adapter's node_modules tree lives in the bundle.
func InstallDir(name string) string { return "lib/acp/" + name }

// SupportsComponent reports whether this provider installs component.
func SupportsComponent(component config.LockComponent) bool {
	return component.Kind == config.KindACPAdapter
}

type Provider struct {
	Components []config.LockComponent
	Node       config.LockComponent
}

func New(components []config.LockComponent, nodeComponent config.LockComponent) Provider {
	return Provider{Components: append([]config.LockComponent(nil), components...), Node: nodeComponent}
}

// scriptSafe bounds what is interpolated into the sh launcher.
var scriptSafe = regexp.MustCompile(`^[A-Za-z0-9@._/+-]+$`)

func (p Provider) Materialize(repoRoot, stagingRoot string) ([]lang.Probe, error) {
	if p.Node.Name == "" {
		return nil, fmt.Errorf("acp adapters need the node component")
	}
	nodeVersion := strings.TrimPrefix(p.Node.UpstreamVersion, "v")
	for _, c := range p.Components {
		if c.Runtime.NodeEngine == "" {
			continue
		}
		ok, err := config.CheckNodeEngine(nodeVersion, c.Runtime.NodeEngine)
		if err != nil {
			return nil, fmt.Errorf("%s node_engine: %w", c.Name, err)
		}
		if !ok {
			return nil, fmt.Errorf("%s requires node_engine %q but node is %s", c.Name, c.Runtime.NodeEngine, nodeVersion)
		}
	}
	dist, err := node.Ensure(repoRoot, stagingRoot, p.Node)
	if err != nil {
		return nil, err
	}
	probes := make([]lang.Probe, 0, len(p.Components))
	for _, c := range p.Components {
		probe, err := install(repoRoot, stagingRoot, filepath.Base(dist), c)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.Name, err)
		}
		probes = append(probes, probe)
	}
	return probes, nil
}

func install(repoRoot, stagingRoot, nodeDist string, c config.LockComponent) (lang.Probe, error) {
	adapter, ok := Lookup(c.Name)
	if !ok {
		return lang.Probe{}, fmt.Errorf("unknown ACP adapter")
	}
	pkg := config.NPMPackageName(c.Source)
	if pkg == "" || !scriptSafe.MatchString(pkg) {
		return lang.Probe{}, fmt.Errorf("source %q is not an npm package", c.Source)
	}
	launcherRel := "bin/" + c.Name
	if c.Probe.Command != launcherRel || len(c.Probe.Args) != 0 {
		return lang.Probe{}, fmt.Errorf("probe must be %s with no arguments", launcherRel)
	}
	dest, err := lang.ContainedStagePath(stagingRoot, InstallDir(c.Name))
	if err != nil {
		return lang.Probe{}, err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return lang.Probe{}, err
	}
	// The claude adapter's closure is ~9,000 entries; extractNode's 20,000 cap
	// is the precedent. The digest check before extraction is the real control.
	if _, err := artifact.ExtractTarGzip(lang.ArtifactPath(repoRoot, c), dest,
		artifact.Digest{Size: c.Artifact.Size, SHA256: c.Artifact.SHA256},
		artifact.Limits{MaxEntries: 20_000, MaxFileBytes: 512 << 20, MaxOutputBytes: 1 << 30}); err != nil {
		return lang.Probe{}, fmt.Errorf("extract artifact: %w", err)
	}
	entry, err := npmbundle.BinEntry(filepath.Join(dest, "node_modules", filepath.FromSlash(pkg)), pkg, c.UpstreamVersion)
	if err != nil {
		return lang.Probe{}, err
	}
	if !scriptSafe.MatchString(entry) || !scriptSafe.MatchString(nodeDist) {
		return lang.Probe{}, fmt.Errorf("bin path %q is not safe to embed in a launcher", entry)
	}
	launcher, err := lang.ContainedStagePath(stagingRoot, launcherRel)
	if err != nil {
		return lang.Probe{}, err
	}
	script := launcherScript(adapter, nodeDist, InstallDir(c.Name)+"/node_modules/"+pkg+"/"+entry)
	if err := lang.WriteExecutable(launcher, []byte(script)); err != nil {
		return lang.Probe{}, err
	}
	return lang.AbsoluteProbe(stagingRoot, c.Probe)
}

// launcherScript resolves the bundle through the activation symlink (cd -P,
// like the TypeScript launchers), points the adapter at its agent CLI when the
// caller has not, and runs the adapter on the bundled node. A CLI missing from
// PATH leaves the variable unset; the adapter then reports it on first use.
func launcherScript(a Adapter, nodeDist, script string) string {
	return fmt.Sprintf(`#!/bin/sh
# %[1]s: Agent Client Protocol adapter on the bundled node (imoogi-toolchain).
set -eu
bin_dir=$(CDPATH= cd -P -- "$(dirname -- "$0")" && pwd)
bundle_dir=$(CDPATH= cd -P -- "$bin_dir/.." && pwd)
if [ -z "${%[2]s:-}" ]; then
  cli=$(command -v %[3]s 2>/dev/null || true)
  case "$cli" in
    /*) %[2]s=$cli; export %[2]s ;;
  esac
fi
exec "$bundle_dir/lib/node/%[4]s/bin/node" "$bundle_dir/%[5]s" "$@"
`, a.Name, a.EnvVar, a.CLI, nodeDist, script)
}
