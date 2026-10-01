package typescript

import (
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/config"
)

func TestSelectComponentsClassifiesByNameOrKind(t *testing.T) {
	components, err := selectComponents([]config.LockComponent{
		selectorComponent("runtime", "node-runtime", ""),
		selectorComponent("sdk", "typescript-sdk", ""),
		selectorComponent("server", "typescript-language-server", ">=20.0.0"),
		selectorComponent("py", "python-language-server", ""),
	})
	if err != nil {
		t.Fatalf("selectComponents failed: %v", err)
	}
	if components.node.Name != "runtime" {
		t.Fatalf("node = %q, want runtime", components.node.Name)
	}
	if components.typescript.Name != "sdk" {
		t.Fatalf("typescript = %q, want sdk", components.typescript.Name)
	}
	if components.tls.Name != "server" {
		t.Fatalf("tls = %q, want server", components.tls.Name)
	}
	if components.pyright.Name != "py" {
		t.Fatalf("pyright = %q, want py", components.pyright.Name)
	}
}

func TestSelectComponentsPreservesClassifierOrderAndLastWins(t *testing.T) {
	components, err := selectComponents([]config.LockComponent{
		selectorComponent("node", "typescript-sdk", ""),
		selectorComponent("typescript", "typescript-language-server", ""),
		selectorComponent("typescript-language-server", "node-runtime", ">=18.0.0"),
		selectorComponent("runtime", "node-runtime", ""),
		selectorComponent("server", "typescript-language-server", ">=20.0.0"),
		selectorComponent("py-one", "python-language-server", ""),
		selectorComponent("basedpyright", "custom-python-server", ""),
	})
	if err != nil {
		t.Fatalf("selectComponents failed: %v", err)
	}
	if components.node.Name != "runtime" {
		t.Fatalf("node = %q, want runtime", components.node.Name)
	}
	if components.typescript.Name != "typescript" {
		t.Fatalf("typescript = %q, want typescript", components.typescript.Name)
	}
	if components.tls.Name != "server" {
		t.Fatalf("tls = %q, want server", components.tls.Name)
	}
	if components.pyright.Name != "basedpyright" {
		t.Fatalf("pyright = %q, want basedpyright", components.pyright.Name)
	}
}

func TestSelectComponentsValidationOrder(t *testing.T) {
	tests := []struct {
		name       string
		components []config.LockComponent
		want       string
	}{
		{
			name:       "empty name by kind still fails missing node",
			components: []config.LockComponent{selectorComponent("", "node-runtime", "")},
			want:       "typescript provider requires node component",
		},
		{
			name:       "missing node before other required components",
			components: []config.LockComponent{selectorComponent("typescript", "typescript-sdk", "")},
			want:       "typescript provider requires node component",
		},
		{
			name: "missing typescript after node",
			components: []config.LockComponent{
				selectorComponent("node", "node-runtime", ""),
			},
			want: "typescript provider requires typescript component",
		},
		{
			name: "missing server after node and typescript",
			components: []config.LockComponent{
				selectorComponent("node", "node-runtime", ""),
				selectorComponent("typescript", "typescript-sdk", ""),
			},
			want: "typescript provider requires typescript-language-server component",
		},
		{
			name: "missing node engine after required components",
			components: []config.LockComponent{
				selectorComponent("node", "node-runtime", ""),
				selectorComponent("typescript", "typescript-sdk", ""),
				selectorComponent("typescript-language-server", "typescript-language-server", ""),
			},
			want: "typescript-language-server component has no node_engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := selectComponents(tt.components)
			if err == nil {
				t.Fatal("selectComponents succeeded")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func selectorComponent(name, kind, nodeEngine string) config.LockComponent {
	return config.LockComponent{
		Name:    name,
		Kind:    kind,
		Runtime: config.RuntimeConstraint{NodeEngine: nodeEngine},
	}
}
