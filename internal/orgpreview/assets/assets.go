package assets

import (
	"path/filepath"

	"github.com/karohani/imoogi-emacs/internal/orgpreview"
)

type Resolver struct {
	roots []string
}

func NewResolver(roots []string) Resolver {
	return Resolver{roots: roots}
}

func (r Resolver) Resolve(target string) (string, error) {
	resolver, err := orgpreview.NewAssetResolver(r.roots)
	if err != nil {
		return "", err
	}
	base := ""
	if len(r.roots) > 0 {
		base = filepath.Join(r.roots[0], "buffer.org")
	}
	return resolver.Resolve(base, target)
}
