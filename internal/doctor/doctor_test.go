package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeEnv struct {
	tools    []string
	bins     []string
	info     map[string]BuildInfo
	deps     map[string][]string
	changed  map[string][]string // revision -> changed files
	required string
	links    map[string]string
	exists   map[string]bool
	files    map[string][]byte // ReadFile contents
	path     map[string]string // LookPath results
	vars     map[string]string // Getenv
	answers  map[string]error  // Handshake results by launcher; absent = no answer
}

const root, home = "/repo", "/home/u"

func (f fakeEnv) Root() string                  { return root }
func (f fakeEnv) Home() string                  { return home }
func (f fakeEnv) Tools() ([]string, error)      { return f.tools, nil }
func (f fakeEnv) BinEntries() ([]string, error) { return f.bins, nil }
func (f fakeEnv) RequiredGo() (string, error)   { return f.required, nil }
func (f fakeEnv) ReadBuildInfo(path string) (BuildInfo, error) {
	info, ok := f.info[filepath.Base(path)]
	if !ok {
		return BuildInfo{}, errors.New("not a Go binary")
	}
	return info, nil
}
func (f fakeEnv) Deps(tool string) ([]string, error) { return f.deps[tool], nil }
func (f fakeEnv) ChangedSince(rev string, _ []string) ([]string, error) {
	return f.changed[rev], nil
}
func (f fakeEnv) Readlink(path string) (string, error) {
	if t, ok := f.links[path]; ok {
		return t, nil
	}
	return "", os.ErrNotExist
}
func (f fakeEnv) Exists(path string) bool { return f.exists[path] }
func (f fakeEnv) ReadFile(path string) ([]byte, error) {
	if data, ok := f.files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}
func (f fakeEnv) LookPath(name string) (string, error) {
	if p, ok := f.path[name]; ok {
		return p, nil
	}
	return "", errors.New("executable file not found in $PATH")
}
func (f fakeEnv) Getenv(key string) string { return f.vars[key] }
func (f fakeEnv) Handshake(launcher string) error {
	if err, ok := f.answers[launcher]; ok {
		return err
	}
	return errors.New("no initialize response within 5s")
}

func healthy() fakeEnv {
	built := BuildInfo{GoVersion: "go1.26.11", Revision: "abc1234def", Main: ModulePath}
	return fakeEnv{
		tools:    []string{"imoogi-anki", "imoogi-emacsctl"},
		bins:     []string{"imoogi-anki", "imoogi-emacsctl"},
		info:     map[string]BuildInfo{"imoogi-anki": built, "imoogi-emacsctl": built},
		deps:     map[string][]string{"imoogi-anki": {"cmd/imoogi-anki", "internal/anki"}, "imoogi-emacsctl": {"cmd/imoogi-emacsctl"}},
		required: "go1.26.11",
		links: map[string]string{
			home + "/.config/imoogi-emacs":       root,
			home + "/.local/bin/imoogi-editor":   home + "/.config/imoogi-emacs/scripts/imoogi-editor",
			home + "/.local/bin/imoogi-emacsctl": home + "/.config/imoogi-emacs/bin/imoogi-emacsctl",
		},
		exists: map[string]bool{
			root + "/bin/imoogi-anki": true, root + "/bin/imoogi-emacsctl": true,
			root + "/scripts/imoogi-editor": true,
		},
	}
}

func statuses(findings []Finding) map[string]string {
	out := map[string]string{}
	for _, f := range findings {
		out[f.Name] = f.Status
	}
	return out
}

func TestHealthySetupIsAllOK(t *testing.T) {
	findings := Check(healthy())
	for _, f := range findings {
		if f.Status != StatusOK {
			t.Fatalf("unexpected finding: %+v", f)
		}
	}
	if !Healthy(findings) || len(findings) != 5 {
		t.Fatalf("want 2 tools + 3 links all ok, got %+v", findings)
	}
}

func TestToolProblemsAreClassified(t *testing.T) {
	env := healthy()
	env.info["imoogi-anki"] = BuildInfo{GoVersion: "go1.26.4", Revision: "abc1234def", Main: ModulePath}
	env.exists[root+"/bin/imoogi-emacsctl"] = false
	got := statuses(Check(env))
	if got["imoogi-anki"] != StatusToolchain || got["imoogi-emacsctl"] != StatusMissing {
		t.Fatalf("toolchain and missing: %v", got)
	}

	env = healthy()
	env.changed = map[string][]string{"abc1234def": {"internal/anki/sync.go"}}
	findings := Check(env)
	got = statuses(findings)
	if got["imoogi-anki"] != StatusStale {
		t.Fatalf("source changed since the build: %v", got)
	}
	for _, f := range findings {
		if f.Name == "imoogi-anki" && !strings.Contains(f.Detail, "internal/anki/sync.go") {
			t.Fatalf("stale detail should name the changed file: %+v", f)
		}
	}

	// Go marks a build modified for any untracked or edited file in the checkout
	// (.moai/ notes, for example). With no change in the tool's own sources since
	// the build commit, the binary matches what a clean build would produce.
	env = healthy()
	env.info["imoogi-anki"] = BuildInfo{GoVersion: "go1.26.11", Revision: "abc1234def", Modified: true, Main: ModulePath}
	findings = Check(env)
	if got := statuses(findings); got["imoogi-anki"] != StatusOK {
		t.Fatalf("a modified checkout with unchanged tool sources is current: %v", got)
	}
	for _, f := range findings {
		if f.Name == "imoogi-anki" && !strings.Contains(f.Detail, "other files") {
			t.Fatalf("detail should say the checkout had other changes: %+v", f)
		}
	}
}

func TestLeftoversAndLinks(t *testing.T) {
	env := healthy()
	env.bins = append(env.bins, "imoogi-agent", "notes.txt", ".imoogi-anki.new")
	env.info["imoogi-agent"] = BuildInfo{GoVersion: "go1.26.4", Revision: "x", Main: ModulePath + "/cmd/imoogi-agent"}
	env.exists[root+"/bin/imoogi-agent"] = true
	env.links[home+"/.config/imoogi-emacs"] = "/tmp/imoogi-toolchain-wrapper.X/install"
	env.links[home+"/.local/bin/imoogi-agent"] = home + "/.config/imoogi-emacs/bin/imoogi-agent"
	delete(env.links, home+"/.local/bin/imoogi-emacsctl")
	got := statuses(Check(env))
	if got["bin/imoogi-agent"] != StatusObsolete {
		t.Fatalf("a binary from this module with no cmd/ is obsolete: %v", got)
	}
	if _, ok := got["bin/notes.txt"]; ok {
		t.Fatalf("files that are not our binaries are left alone: %v", got)
	}
	if got["~/.config/imoogi-emacs"] != StatusWrongLink {
		t.Fatalf("config link pointing elsewhere: %v", got)
	}
	if got["~/.local/bin/imoogi-emacsctl"] != StatusMissing || got["~/.local/bin/imoogi-agent"] != StatusObsolete {
		t.Fatalf("missing and leftover links: %v", got)
	}
	if Healthy(Check(env)) {
		t.Fatal("these problems must fail the doctor")
	}
}

func TestEveryExpectedLinkIsCreatedByTheInstaller(t *testing.T) {
	script, err := os.ReadFile("../../scripts/install.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, link := range ExpectedLinks {
		if !strings.Contains(string(script), link.Name) {
			t.Fatalf("doctor expects ~/.local/bin/%s but scripts/install.sh never creates it", link.Name)
		}
	}
}
