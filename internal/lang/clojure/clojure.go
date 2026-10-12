// Package clojure installs clojure-lsp from its GraalVM native build.
//
// The native binary needs no Java runtime, so Clojure LSP works on a target
// without a JDK — unlike the JVM-hosted servers in package jvm. It is the
// only file in the upstream zip and is installed as bin/clojure-lsp, the same
// shape as gopls.
package clojure

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang"
)

// SupportsComponent reports whether this provider installs component.
func SupportsComponent(component config.LockComponent) bool {
	return component.Name == "clojure-lsp" || component.Kind == "clojure-language-server"
}

type Provider struct {
	Component config.LockComponent
}

func New(component config.LockComponent) Provider {
	return Provider{Component: component}
}

func (p Provider) Materialize(repoRoot, stagingRoot string) ([]lang.Probe, error) {
	component := p.Component
	if !SupportsComponent(component) {
		return nil, fmt.Errorf("clojure provider cannot materialize component %q kind %q", component.Name, component.Kind)
	}
	extract, err := lang.ContainedStagePath(stagingRoot, "lib/.extract-clojure-lsp")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(extract), 0o755); err != nil {
		return nil, err
	}
	if _, err := artifact.ExtractZip(lang.ArtifactPath(repoRoot, component), extract,
		artifact.Digest{Size: component.Artifact.Size, SHA256: component.Artifact.SHA256},
		artifact.Limits{MaxEntries: 16}); err != nil {
		return nil, fmt.Errorf("extract clojure-lsp artifact: %w", err)
	}
	defer os.RemoveAll(extract)

	source := filepath.Join(extract, "clojure-lsp")
	if info, err := os.Lstat(source); err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("clojure-lsp archive has no clojure-lsp binary at its root")
	}
	destination, err := lang.ContainedStagePath(stagingRoot, "bin/clojure-lsp")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(source, destination); err != nil {
		return nil, fmt.Errorf("install clojure-lsp: %w", err)
	}
	if err := lang.EnsureExecutable(destination); err != nil {
		return nil, err
	}
	probe, err := lang.AbsoluteProbe(stagingRoot, component.Probe)
	if err != nil {
		return nil, err
	}
	return []lang.Probe{probe}, nil
}
