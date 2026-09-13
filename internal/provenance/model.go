package provenance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const (
	IndexSchema  = "imoogi-vendor-index/v1"
	DomainSchema = "imoogi-vendor-domain/v1"
)

type Index struct {
	Schema     string      `json:"schema"`
	Boundaries []string    `json:"boundaries"`
	Roots      []string    `json:"roots"`
	Manifests  []string    `json:"manifests"`
	Excludes   []Exclusion `json:"excludes,omitempty"`
}

type Exclusion struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type Domain struct {
	Schema     string      `json:"schema"`
	Domain     string      `json:"domain"`
	Components []Component `json:"components"`
}

type Component struct {
	ID       string    `json:"id"`
	Kind     string    `json:"kind"`
	Version  string    `json:"version"`
	Source   Source    `json:"source"`
	Platform *Platform `json:"platform,omitempty"`
	Workflow string    `json:"workflow"`
	Files    []File    `json:"files"`
}

type Source struct {
	Type        string `json:"type"`
	URL         string `json:"url"`
	ArtifactURL string `json:"artifact_url,omitempty"`
	Commit      string `json:"commit,omitempty"`
	Ref         string `json:"ref,omitempty"`
}

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Role   string `json:"role,omitempty"`
}

func loadStrict(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return fmt.Errorf("decode %s: trailing data: %w", path, err)
		}
		return fmt.Errorf("decode %s: trailing JSON value", path)
	}
	return nil
}
