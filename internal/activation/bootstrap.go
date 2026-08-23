package activation

import "fmt"

const (
	BootstrapGOOS   = "darwin"
	BootstrapGOARCH = "arm64"
	BootstrapCGO    = "0"
)

type BootstrapSpec struct {
	Version string
	Output  string
}

func (s BootstrapSpec) Env() []string {
	return []string{
		"GOOS=" + BootstrapGOOS,
		"GOARCH=" + BootstrapGOARCH,
		"CGO_ENABLED=" + BootstrapCGO,
	}
}

func (s BootstrapSpec) Args() []string {
	version := s.Version
	if version == "" {
		version = "dev"
	}
	output := s.Output
	if output == "" {
		output = "imoogi-toolchain"
	}
	return []string{
		"build",
		"-trimpath",
		"-buildvcs=false",
		"-ldflags=-buildid= -X main.version=" + version,
		"-o",
		output,
		"./cmd/imoogi-toolchain",
	}
}

func (s BootstrapSpec) Provenance() string {
	return fmt.Sprintf("GOOS=%s GOARCH=%s CGO_ENABLED=%s go %s", BootstrapGOOS, BootstrapGOARCH, BootstrapCGO, joinArgs(s.Args()))
}

func joinArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	out := args[0]
	for _, arg := range args[1:] {
		out += " " + arg
	}
	return out
}
