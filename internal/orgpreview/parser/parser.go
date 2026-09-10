package parser

import "github.com/karohani/imoogi-emacs/internal/orgpreview"

type Adapter string

const (
	AdapterFallback   Adapter = "fallback"
	AdapterTreeSitter Adapter = "tree-sitter"
)

type Options struct {
	Adapter  Adapter
	BufferID string
}

func Parse(source []byte, opts Options) (orgpreview.Document, error) {
	if opts.Adapter == AdapterTreeSitter {
		return orgpreview.TreeSitterParser{}.Parse(string(source))
	}
	return orgpreview.FallbackParser{}.Parse(string(source))
}

func TreeSitterAvailable() bool {
	return false
}
