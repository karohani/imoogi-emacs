package clipboard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ResolvedOwner struct {
	Owner           Owner
	AssetRoot       string
	ContainmentRoot string
}

func ResolveOwner(owner Owner, stagingBase string) (ResolvedOwner, error) {
	if err := owner.ValidateShape(); err != nil {
		return ResolvedOwner{}, err
	}
	switch owner.Kind {
	case OwnerProjectNotes:
		document, err := canonicalForContainment(owner.Document)
		if err != nil {
			return ResolvedOwner{}, fmt.Errorf("canonical document: %w", err)
		}
		root, err := canonicalForContainment(owner.Root)
		if err != nil {
			return ResolvedOwner{}, fmt.Errorf("canonical project root: %w", err)
		}
		if !pathWithin(root, document) || document == root {
			return ResolvedOwner{}, errors.New("project document is outside project root")
		}
		assetRoot, err := canonicalForContainment(filepath.Join(root, "assets"))
		if err != nil {
			return ResolvedOwner{}, fmt.Errorf("canonical project asset root: %w", err)
		}
		if !pathWithin(root, assetRoot) || assetRoot == root {
			return ResolvedOwner{}, fmt.Errorf("%s: project asset root is outside project root", CodePathEscape)
		}
		return ResolvedOwner{Owner: owner, AssetRoot: assetRoot, ContainmentRoot: root}, nil
	case OwnerStandalone:
		document, err := canonicalForContainment(owner.Document)
		if err != nil {
			return ResolvedOwner{}, fmt.Errorf("canonical document: %w", err)
		}
		expected := strings.TrimSuffix(document, filepath.Ext(document)) + ".assets"
		root, err := canonicalForContainment(owner.Root)
		if err != nil {
			return ResolvedOwner{}, fmt.Errorf("canonical standalone root: %w", err)
		}
		if root != expected {
			return ResolvedOwner{}, fmt.Errorf("standalone root %q does not match %q", root, expected)
		}
		return ResolvedOwner{Owner: owner, AssetRoot: root, ContainmentRoot: filepath.Dir(expected)}, nil
	case OwnerStaging:
		base, err := filepath.Abs(stagingBase)
		if err != nil {
			return ResolvedOwner{}, err
		}
		generationRoot := filepath.Join(base, "sessions", safeComponent(owner.Session), safeComponent(owner.Buffer), fmt.Sprint(owner.Generation))
		return ResolvedOwner{
			Owner:           owner,
			AssetRoot:       filepath.Join(generationRoot, "assets"),
			ContainmentRoot: generationRoot,
		}, nil
	default:
		return ResolvedOwner{}, fmt.Errorf("unsupported owner kind %q", owner.Kind)
	}
}

func canonicalForContainment(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	current := abs
	var suffix []string
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for index := len(suffix) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, suffix[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return abs, nil
		}
		suffix = append(suffix, filepath.Base(current))
		current = parent
	}
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func safeComponent(value string) string {
	value = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return '-'
		}
		return r
	}, value)
	if value == "" || value == "." || value == ".." {
		return "invalid"
	}
	return value
}
