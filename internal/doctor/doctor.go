// Package doctor reports whether the CLI tools in bin/ and the links the
// installer makes are built from, and pointing at, the current repository.
package doctor

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ModulePath is this repository's Go module; binaries built from it carry it.
const ModulePath = "github.com/karohani/imoogi-emacs"

const (
	StatusOK        = "ok"
	StatusMissing   = "missing"   // not built / link absent
	StatusStale     = "stale"     // source changed since the binary was built
	StatusToolchain = "toolchain" // built with a different Go than go.work pins
	StatusDirty     = "dirty"     // built from uncommitted changes; cannot prove it is current (warning)
	StatusObsolete  = "obsolete"  // a leftover from a tool that no longer exists
	StatusWrongLink = "wrong"     // link points somewhere else
	StatusUnknown   = "unknown"   // could not be inspected
)

// Finding is one checked item.
type Finding struct {
	Kind   string `json:"kind"` // tool, bin, link
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Fix    string `json:"fix,omitempty"`
}

// BuildInfo is what a Go binary records about how it was built.
type BuildInfo struct {
	GoVersion string
	Revision  string
	Modified  bool
	Main      string // main package path
}

// Env is everything the checks read; the real one inspects the file system,
// go and git, and tests use a fixed one.
type Env interface {
	Root() string
	Home() string
	Tools() ([]string, error) // cmd/<name> directories that build into bin/<name>
	BinEntries() ([]string, error)
	RequiredGo() (string, error)
	ReadBuildInfo(path string) (BuildInfo, error)
	Deps(tool string) ([]string, error) // repo-relative package directories the tool is built from
	ChangedSince(revision string, paths []string) ([]string, error)
	Readlink(path string) (string, error)
	Exists(path string) bool
}

// Link is a ~/.local/bin entry the installer creates.
type Link struct {
	Name   string
	Target string // relative to ~/.config/imoogi-emacs
}

// ExpectedLinks mirrors scripts/install.sh; a test keeps the two in step.
var ExpectedLinks = []Link{
	{Name: "imoogi-editor", Target: "scripts/imoogi-editor"},
	{Name: "imoogi-emacsctl", Target: "bin/imoogi-emacsctl"},
}

// LegacyLinks are names earlier versions installed into ~/.local/bin.
var LegacyLinks = []Link{{Name: "imoogi-agent", Target: "bin/imoogi-agent"}}

const (
	fixBuild   = "make build-all"
	fixInstall = "make install"
)

// Check inspects the tools and links. It never changes anything.
func Check(env Env) []Finding {
	var out []Finding
	tools, err := env.Tools()
	if err != nil {
		return []Finding{{Kind: "tool", Name: "cmd/", Status: StatusUnknown, Detail: err.Error()}}
	}
	required, reqErr := env.RequiredGo()
	known := map[string]bool{}
	for _, tool := range tools {
		known[tool] = true
		out = append(out, checkTool(env, tool, required, reqErr))
	}
	out = append(out, leftovers(env, known)...)
	out = append(out, checkLinks(env)...)
	return out
}

func checkTool(env Env, tool, required string, reqErr error) Finding {
	f := Finding{Kind: "tool", Name: tool, Status: StatusOK, Fix: fixBuild}
	bin := filepath.Join(env.Root(), "bin", tool)
	if !env.Exists(bin) {
		f.Status, f.Detail = StatusMissing, "bin/"+tool+" is not built"
		return f
	}
	info, err := env.ReadBuildInfo(bin)
	if err != nil {
		f.Status, f.Detail = StatusUnknown, err.Error()
		return f
	}
	if reqErr != nil {
		f.Status, f.Detail = StatusUnknown, "cannot read the pinned Go toolchain: "+reqErr.Error()
		return f
	}
	if info.GoVersion != required {
		f.Status, f.Detail = StatusToolchain, "built with "+info.GoVersion+", go.work pins "+required
		return f
	}
	if info.Revision == "" {
		f.Status, f.Detail = StatusUnknown, "binary records no git revision"
		return f
	}
	deps, err := env.Deps(tool)
	if err != nil {
		f.Status, f.Detail = StatusUnknown, err.Error()
		return f
	}
	changed, err := env.ChangedSince(info.Revision, append(deps, "go.mod", "go.sum", "go.work"))
	if err != nil {
		f.Status, f.Detail = StatusUnknown, err.Error()
		return f
	}
	if len(changed) > 0 {
		f.Status = StatusStale
		f.Detail = fmt.Sprintf("%d file(s) changed since %s: %s", len(changed), short(info.Revision), preview(changed))
		return f
	}
	if info.Modified {
		f.Status, f.Detail = StatusDirty, "built from uncommitted changes on "+short(info.Revision)
		return f
	}
	f.Fix, f.Detail = "", "built from "+short(info.Revision)+" with "+info.GoVersion
	return f
}

// leftovers reports binaries this module built for tools that no longer exist.
func leftovers(env Env, known map[string]bool) []Finding {
	entries, err := env.BinEntries()
	if err != nil {
		return nil
	}
	sort.Strings(entries)
	var out []Finding
	for _, name := range entries {
		if known[name] || strings.HasPrefix(name, ".") {
			continue
		}
		info, err := env.ReadBuildInfo(filepath.Join(env.Root(), "bin", name))
		if err != nil || !strings.HasPrefix(info.Main, ModulePath) {
			continue // not ours; leave it alone
		}
		out = append(out, Finding{Kind: "bin", Name: "bin/" + name, Status: StatusObsolete,
			Detail: "built from " + info.Main + ", which no longer exists", Fix: "rm bin/" + name})
	}
	return out
}

func checkLinks(env Env) []Finding {
	config := filepath.Join(env.Home(), ".config", "imoogi-emacs")
	var out []Finding
	f := Finding{Kind: "link", Name: "~/.config/imoogi-emacs", Status: StatusOK, Detail: "→ " + env.Root()}
	if target, err := env.Readlink(config); err != nil {
		f.Status, f.Detail, f.Fix = StatusMissing, "not linked", fixInstall
	} else if filepath.Clean(target) != filepath.Clean(env.Root()) {
		f.Status, f.Detail, f.Fix = StatusWrongLink, "points to "+target+", not "+env.Root(), fixInstall
	}
	out = append(out, f)
	for _, link := range ExpectedLinks {
		path := filepath.Join(env.Home(), ".local", "bin", link.Name)
		want := filepath.Join(config, link.Target)
		f := Finding{Kind: "link", Name: "~/.local/bin/" + link.Name, Status: StatusOK, Detail: "→ " + want}
		if target, err := env.Readlink(path); err != nil {
			f.Status, f.Detail, f.Fix = StatusMissing, "not linked", fixInstall
		} else if filepath.Clean(target) != want {
			f.Status, f.Detail, f.Fix = StatusWrongLink, "points to "+target+", not "+want, fixInstall
		}
		out = append(out, f)
	}
	for _, link := range LegacyLinks {
		path := filepath.Join(env.Home(), ".local", "bin", link.Name)
		if target, err := env.Readlink(path); err == nil && filepath.Clean(target) == filepath.Join(config, link.Target) {
			out = append(out, Finding{Kind: "link", Name: "~/.local/bin/" + link.Name, Status: StatusObsolete,
				Detail: "left by an older install", Fix: fixInstall})
		}
	}
	return out
}

// Healthy reports whether nothing needs fixing; a dirty build is only a warning.
func Healthy(findings []Finding) bool {
	for _, f := range findings {
		if f.Status != StatusOK && f.Status != StatusDirty {
			return false
		}
	}
	return true
}

func short(rev string) string {
	if len(rev) > 7 {
		return rev[:7]
	}
	return rev
}

func preview(files []string) string {
	if len(files) > 3 {
		return strings.Join(files[:3], ", ") + fmt.Sprintf(" and %d more", len(files)-3)
	}
	return strings.Join(files, ", ")
}
