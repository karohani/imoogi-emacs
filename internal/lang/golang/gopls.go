package golang

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang"
)

type GoplsProvider struct {
	Component config.LockComponent
}

func New(component config.LockComponent) GoplsProvider {
	return GoplsProvider{Component: component}
}

func (p GoplsProvider) Materialize(repoRoot, stagingRoot string) ([]lang.Probe, error) {
	component := p.Component
	if component.Name != "gopls" && component.Kind != "go-language-server" {
		return nil, fmt.Errorf("gopls provider cannot materialize component %q kind %q", component.Name, component.Kind)
	}
	source := lang.ArtifactPath(repoRoot, component)
	if err := artifact.VerifyFile(source, artifact.Digest{Size: component.Artifact.Size, SHA256: component.Artifact.SHA256}); err != nil {
		return nil, fmt.Errorf("verify gopls artifact: %w", err)
	}
	destination, err := lang.ContainedStagePath(stagingRoot, "bin/gopls")
	if err != nil {
		return nil, err
	}
	if err := copyExecutable(source, destination); err != nil {
		return nil, fmt.Errorf("install gopls: %w", err)
	}
	probe, err := lang.AbsoluteProbe(stagingRoot, component.Probe)
	if err != nil {
		return nil, err
	}
	return []lang.Probe{probe}, nil
}

func copyExecutable(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, lang.ExecutableMode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(lang.ExecutableMode)
}
