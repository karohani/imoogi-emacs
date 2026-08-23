package activation

import (
	"strings"
	"testing"
)

func TestBootstrapSpecIsDarwinArm64AndReproducible(t *testing.T) {
	spec := BootstrapSpec{Version: "1.2.3", Output: "out/imoogi-toolchain"}
	if got, want := spec.Env(), []string{"GOOS=darwin", "GOARCH=arm64", "CGO_ENABLED=0"}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("env = %#v, want %#v", got, want)
	}
	args := strings.Join(spec.Args(), " ")
	for _, want := range []string{"build", "-trimpath", "-buildvcs=false", "-ldflags=-buildid= -X main.version=1.2.3", "-o out/imoogi-toolchain", "./cmd/imoogi-toolchain"} {
		if !strings.Contains(args, want) {
			t.Fatalf("args %q missing %q", args, want)
		}
	}
	if !strings.Contains(spec.Provenance(), "GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -buildvcs=false") {
		t.Fatalf("provenance = %q", spec.Provenance())
	}
}
