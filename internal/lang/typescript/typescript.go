package typescript

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang"
)

type Provider struct {
	Components []config.LockComponent
}

func New(components []config.LockComponent) Provider {
	return Provider{Components: append([]config.LockComponent(nil), components...)}
}

func (p Provider) Materialize(repoRoot, stagingRoot string) ([]lang.Probe, error) {
	components, err := selectComponents(p.Components)
	if err != nil {
		return nil, err
	}
	nodeVersion := strings.TrimPrefix(components.node.UpstreamVersion, "v")
	ok, err := config.CheckNodeEngine(nodeVersion, components.tls.Runtime.NodeEngine)
	if err != nil {
		return nil, fmt.Errorf("typescript-language-server node_engine: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("typescript-language-server requires node_engine %q but node is %s", components.tls.Runtime.NodeEngine, nodeVersion)
	}

	nodeRoot, err := extractNode(repoRoot, stagingRoot, components.node)
	if err != nil {
		return nil, err
	}
	tsRoot, err := extractPackage(repoRoot, stagingRoot, components.typescript, "typescript")
	if err != nil {
		return nil, err
	}
	tlsRoot, err := extractPackage(repoRoot, stagingRoot, components.tls, "typescript-language-server")
	if err != nil {
		return nil, err
	}

	nodeDist := filepath.Base(nodeRoot)
	nodeBin := filepath.Join(nodeRoot, "bin", "node")
	if err := lang.EnsureExecutable(nodeBin); err != nil {
		return nil, fmt.Errorf("node executable: %w", err)
	}
	if err := writeNodeShim(stagingRoot, nodeDist); err != nil {
		return nil, err
	}
	if err := writeTscLauncher(stagingRoot, nodeDist); err != nil {
		return nil, err
	}
	if err := writeLauncher(stagingRoot, nodeDist); err != nil {
		return nil, err
	}
	if err := validatePackage(tlsRoot, "typescript-language-server"); err != nil {
		return nil, err
	}
	if err := validatePackage(tsRoot, "typescript"); err != nil {
		return nil, err
	}
	if err := validatePackageFile(tsRoot, "typescript", "lib/tsc.js"); err != nil {
		return nil, err
	}
	if err := validatePackageFile(tsRoot, "typescript", "lib/tsserver.js"); err != nil {
		return nil, err
	}
	if err := validatePackageFile(tlsRoot, "typescript-language-server", "lib/cli.mjs"); err != nil {
		return nil, err
	}

	probes := make([]lang.Probe, 0, 3)
	nodeProbe, err := lang.AbsoluteProbe(stagingRoot, components.node.Probe)
	if err != nil {
		return nil, err
	}
	tsProbe, err := lang.AbsoluteProbe(stagingRoot, components.typescript.Probe)
	if err != nil {
		return nil, err
	}
	tlsProbe, err := lang.AbsoluteProbe(stagingRoot, components.tls.Probe)
	if err != nil {
		return nil, err
	}
	probes = append(probes, nodeProbe, tsProbe, tlsProbe)
	return probes, nil
}

type selected struct {
	node       config.LockComponent
	typescript config.LockComponent
	tls        config.LockComponent
}

func selectComponents(components []config.LockComponent) (selected, error) {
	var out selected
	for _, component := range components {
		switch {
		case component.Name == "node" || component.Kind == "node-runtime":
			out.node = component
		case component.Name == "typescript" || component.Kind == "typescript-sdk":
			out.typescript = component
		case component.Name == "typescript-language-server" || component.Kind == "typescript-language-server":
			out.tls = component
		}
	}
	if out.node.Name == "" {
		return selected{}, fmt.Errorf("typescript provider requires node component")
	}
	if out.typescript.Name == "" {
		return selected{}, fmt.Errorf("typescript provider requires typescript component")
	}
	if out.tls.Name == "" {
		return selected{}, fmt.Errorf("typescript provider requires typescript-language-server component")
	}
	if out.tls.Runtime.NodeEngine == "" {
		return selected{}, fmt.Errorf("typescript-language-server component has no node_engine")
	}
	return out, nil
}

func extractNode(repoRoot, stagingRoot string, component config.LockComponent) (string, error) {
	destination, err := lang.ContainedStagePath(stagingRoot, "lib/node")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", err
	}
	if _, err := artifact.ExtractTarGzipWithOptions(
		lang.ArtifactPath(repoRoot, component),
		destination,
		digest(component),
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
	if len(entries) != 1 || !entries[0].IsDir() {
		return "", fmt.Errorf("node archive must contain one distribution directory")
	}
	return filepath.Join(destination, entries[0].Name()), nil
}

func extractPackage(repoRoot, stagingRoot string, component config.LockComponent, packageName string) (string, error) {
	extractDestination, err := lang.ContainedStagePath(stagingRoot, "lib/.extract-"+packageName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(extractDestination), 0o755); err != nil {
		return "", err
	}
	if _, err := artifact.ExtractTarGzip(lang.ArtifactPath(repoRoot, component), extractDestination, digest(component), artifact.Limits{}); err != nil {
		return "", fmt.Errorf("extract %s artifact: %w", component.Name, err)
	}
	defer os.RemoveAll(extractDestination)

	packageRoot := filepath.Join(extractDestination, "package")
	info, err := os.Stat(packageRoot)
	if err != nil {
		return "", fmt.Errorf("%s package root: %w", component.Name, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s package root is not a directory", component.Name)
	}
	if err := validatePackage(packageRoot, packageName); err != nil {
		return "", err
	}

	packageDestination, err := nodePackageDestination(stagingRoot, packageName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(packageDestination), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Lstat(packageDestination); err == nil {
		return "", fmt.Errorf("%s package destination already exists", component.Name)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("%s package destination: %w", component.Name, err)
	}
	if err := os.Rename(packageRoot, packageDestination); err != nil {
		return "", fmt.Errorf("install %s package: %w", component.Name, err)
	}
	return packageDestination, nil
}

func writeNodeShim(stagingRoot, nodeDist string) error {
	shim, err := lang.ContainedStagePath(stagingRoot, "bin/node")
	if err != nil {
		return err
	}
	script := fmt.Sprintf(`#!/bin/sh
set -eu
bin_dir=$(CDPATH= cd -P -- "$(dirname -- "$0")" && pwd)
bundle_dir=$(CDPATH= cd -P -- "$bin_dir/.." && pwd)
exec "$bundle_dir/lib/node/%s/bin/node" "$@"
`, nodeDist)
	return lang.WriteExecutable(shim, []byte(script))
}

func writeLauncher(stagingRoot, nodeDist string) error {
	launcher, err := lang.ContainedStagePath(stagingRoot, "bin/typescript-language-server")
	if err != nil {
		return err
	}
	script := fmt.Sprintf(`#!/bin/sh
set -eu
bin_dir=$(CDPATH= cd -P -- "$(dirname -- "$0")" && pwd)
bundle_dir=$(CDPATH= cd -P -- "$bin_dir/.." && pwd)
node="$bundle_dir/lib/node/%s/bin/node"
tls_cli="$bundle_dir/lib/node_modules/typescript-language-server/lib/cli.mjs"
exec "$node" "$tls_cli" "$@"
`, nodeDist)
	return lang.WriteExecutable(launcher, []byte(script))
}

func writeTscLauncher(stagingRoot, nodeDist string) error {
	launcher, err := lang.ContainedStagePath(stagingRoot, "bin/tsc")
	if err != nil {
		return err
	}
	script := fmt.Sprintf(`#!/bin/sh
set -eu
bin_dir=$(CDPATH= cd -P -- "$(dirname -- "$0")" && pwd)
bundle_dir=$(CDPATH= cd -P -- "$bin_dir/.." && pwd)
exec "$bundle_dir/lib/node/%s/bin/node" "$bundle_dir/lib/node_modules/typescript/lib/tsc.js" "$@"
`, nodeDist)
	return lang.WriteExecutable(launcher, []byte(script))
}

func nodePackageDestination(stagingRoot, packageName string) (string, error) {
	if !validNpmPackageName(packageName) {
		return "", fmt.Errorf("unsupported npm package name %q", packageName)
	}
	return lang.ContainedStagePath(stagingRoot, filepath.ToSlash(filepath.Join("lib/node_modules", packageName)))
}

func validNpmPackageName(name string) bool {
	if name == "" || strings.Contains(name, `\`) || strings.Contains(name, "..") {
		return false
	}
	parts := strings.Split(name, "/")
	if strings.HasPrefix(name, "@") {
		if len(parts) != 2 || parts[0] == "@" || parts[1] == "" {
			return false
		}
	} else if len(parts) != 1 {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func validatePackage(packageRoot, name string) error {
	data, err := os.ReadFile(filepath.Join(packageRoot, "package.json"))
	if err != nil {
		return fmt.Errorf("%s package.json: %w", name, err)
	}
	var parsed struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("%s package.json: %w", name, err)
	}
	if parsed.Name != name {
		return fmt.Errorf("package %s has name %q", name, parsed.Name)
	}
	return nil
}

func validatePackageFile(packageRoot, packageName, relPath string) error {
	if relPath == "" || filepath.IsAbs(relPath) || strings.Contains(relPath, "..") {
		return fmt.Errorf("%s invalid required path %q", packageName, relPath)
	}
	info, err := os.Stat(filepath.Join(packageRoot, filepath.FromSlash(relPath)))
	if err != nil {
		return fmt.Errorf("%s required file %s: %w", packageName, relPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s required file %s is a directory", packageName, relPath)
	}
	return nil
}

func digest(component config.LockComponent) artifact.Digest {
	return artifact.Digest{Size: component.Artifact.Size, SHA256: component.Artifact.SHA256}
}
