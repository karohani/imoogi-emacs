package orgpreview

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type AssetResolver struct {
	AllowedRoots []string
	ExactAssets  map[string]string
}

func NewAssetResolver(roots []string) (AssetResolver, error) {
	return NewAssetResolverWithMap(roots, nil)
}

func NewAssetResolverWithMap(roots []string, exact map[string]string) (AssetResolver, error) {
	resolved := make([]string, 0, len(roots))
	for _, root := range roots {
		if root == "" {
			continue
		}
		clean, err := canonical(root)
		if err != nil {
			return AssetResolver{}, err
		}
		resolved = append(resolved, clean)
	}
	assets := make(map[string]string, len(exact))
	for id, path := range exact {
		if !strings.HasPrefix(id, "imoogi-asset:") {
			continue
		}
		clean, err := canonical(path)
		if err != nil {
			return AssetResolver{}, fmt.Errorf("canonical exact asset: %w", err)
		}
		assets[id] = clean
	}
	return AssetResolver{AllowedRoots: resolved, ExactAssets: assets}, nil
}

func (r AssetResolver) Resolve(baseFile, target string) (string, error) {
	if target == "" {
		return "", errors.New("empty target")
	}
	if exact, ok := r.ExactAssets[target]; ok {
		return exact, nil
	}
	decoded, err := url.PathUnescape(target)
	if err != nil {
		return "", fmt.Errorf("decode target: %w", err)
	}
	parsed, err := url.Parse(decoded)
	if err == nil && parsed.Scheme != "" && parsed.Scheme != "file" {
		return "", fmt.Errorf("scheme %q is not allowed", parsed.Scheme)
	}
	if parsed != nil && parsed.Scheme == "file" {
		return "", errors.New("file scheme is not allowed")
	}
	var candidate string
	if filepath.IsAbs(decoded) {
		candidate = decoded
	} else {
		base := "."
		if baseFile != "" {
			base = filepath.Dir(baseFile)
		}
		candidate = filepath.Join(base, decoded)
	}
	clean, err := canonical(candidate)
	if err != nil {
		return "", err
	}
	for _, root := range r.AllowedRoots {
		if within(root, clean) {
			return clean, nil
		}
	}
	return "", errors.New("target is outside allowed roots")
}

func canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return filepath.Clean(abs), nil
		}
		return "", err
	}
	return filepath.Clean(eval), nil
}

func within(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
