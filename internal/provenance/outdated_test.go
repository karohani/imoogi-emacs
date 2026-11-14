package provenance

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeRemote answers upstream questions from fixed tables so no test needs a network.
type fakeRemote struct {
	tags     map[string][]string
	heads    map[string]string
	npm      map[string]NPMRelease
	node     []NodeRelease
	proxy    map[string]string
	blobs    map[string][]byte
	times    map[string]map[string]time.Time
	archives map[string][2]string // package -> archive, version
	failed   map[string]bool
}

func (f fakeRemote) Tags(_ context.Context, url string) ([]string, error) {
	if f.failed[url] {
		return nil, errors.New("unreachable")
	}
	return f.tags[url], nil
}
func (f fakeRemote) Head(_ context.Context, url string) (string, error) {
	if f.failed[url] {
		return "", errors.New("unreachable")
	}
	return f.heads[url], nil
}
func (f fakeRemote) NPMLatest(_ context.Context, name string) (NPMRelease, error) {
	r, ok := f.npm[name]
	if !ok {
		return NPMRelease{}, errors.New("no package")
	}
	return r, nil
}
func (f fakeRemote) NPMTimes(_ context.Context, name string) (map[string]time.Time, error) {
	return f.times[name], nil
}
func (f fakeRemote) ArchiveVersion(_ context.Context, name string, snapshots bool) (string, string, error) {
	a, ok := f.archives[name]
	if !ok || (a[0] == "melpa" && !snapshots) {
		return "", "", errors.New("not in any archive")
	}
	return a[0], a[1], nil
}
func (f fakeRemote) NodeReleases(context.Context) ([]NodeRelease, error) { return f.node, nil }
func (f fakeRemote) GoProxyLatest(_ context.Context, module string) (string, error) {
	return f.proxy[module], nil
}
func (f fakeRemote) Fetch(_ context.Context, url string) ([]byte, error) { return f.blobs[url], nil }

func TestCompareVersionsAndUpdateLevel(t *testing.T) {
	for _, tc := range []struct {
		current, latest string
		newer           bool
		level           string
	}{
		{"0.10.0", "0.9.0", false, ""},
		{"v0.25.1", "v0.25.2", true, "patch"},
		{"1.39.10", "1.40.0", true, "minor"},
		{"6.0.3", "7.0.0", true, "major"},
		{"2.5", "2.5", false, ""},
		{"2026.07.06-14.34.19", "2026.09.01-10.00.00", true, "release"},
		{"unstable-20250526", "unstable-20260101", true, "release"},
	} {
		newer, level := compareVersions(tc.current, tc.latest)
		if newer != tc.newer || level != tc.level {
			t.Fatalf("compare(%q, %q) = %v %q, want %v %q", tc.current, tc.latest, newer, level, tc.newer, tc.level)
		}
	}
}

func TestLatestTagMatchesTheCurrentTagStyle(t *testing.T) {
	tags := []string{"v0.24.0", "v0.25.1", "v0.26.0-rc1", "v0.25.3", "nightly", "release-2020"}
	if got := latestTag("v0.25.1", tags); got != "v0.25.3" {
		t.Fatalf("v-style: %q (pre-releases and other styles must be ignored)", got)
	}
	// A bare version and a v-prefixed tag are the same style.
	if got := latestTag("1.2.0", []string{"v1.2.0", "v1.3.1", "1.3.0"}); got != "v1.3.1" {
		t.Fatalf("bare vs v: %q", got)
	}
	if got := latestTag("VER2.5", []string{"VER2.0", "VER2.5", "VER2.6"}); got != "VER2.6" {
		t.Fatalf("custom prefix: %q", got)
	}
	if got := latestTag("1.0.0", []string{"foo", "bar"}); got != "" {
		t.Fatalf("no comparable tag: %q", got)
	}
}

func TestReleaseRefFromURLs(t *testing.T) {
	repo, ref := releaseOf("https://github.com/naver/nanumfont/releases/download/VER2.5/NanumGothicCoding-2.5.zip")
	if repo != "https://github.com/naver/nanumfont" || ref != "VER2.5" {
		t.Fatalf("release download: %q %q", repo, ref)
	}
	if got := npmPackageOf("https://registry.npmjs.org/@scope/pkg/-/pkg-1.0.0.tgz"); got != "@scope/pkg" {
		t.Fatalf("scoped npm: %q", got)
	}
	if got := npmPackageOf("https://registry.npmjs.org/typescript/-/typescript-6.0.3.tgz"); got != "typescript" {
		t.Fatalf("npm: %q", got)
	}
	if got := goModuleOf("https://proxy.golang.org/golang.org/x/tools/gopls/@v/v0.23.0.zip"); got != "golang.org/x/tools/gopls" {
		t.Fatalf("go proxy: %q", got)
	}
}

func TestCheckComponentPicksAStrategyPerSource(t *testing.T) {
	ctx := context.Background()
	remote := fakeRemote{
		tags: map[string][]string{
			"https://github.com/tree-sitter/tree-sitter-bash": {"v0.25.0", "v0.25.1", "v0.26.0"},
			"https://github.com/abo-abo/ace-window":           {"0.7.1", "0.9.0"},
			"https://github.com/naver/nanumfont":              {"VER2.0", "VER2.5"},
		},
		heads: map[string]string{
			"https://github.com/radian-software/apheleia": "newcommit",
			"https://github.com/abo-abo/ace-window":       "same",
		},
		npm:    map[string]NPMRelease{"basedpyright": {Version: "1.40.1"}},
		node:   []NodeRelease{{Version: "v25.1.0"}, {Version: "v24.20.0", LTS: true}, {Version: "v24.19.0", LTS: true}},
		proxy:  map[string]string{"golang.org/x/tools/gopls": "v0.24.0"},
		blobs:  map[string][]byte{"https://raw.githubusercontent.com/o/r/main/NFM.ttf": []byte("font-v2")},
		failed: map[string]bool{"https://github.com/gone/away": true},
	}
	check := func(c Component) Result { return checkComponent(ctx, remote, "domain", c) }

	if r := check(Component{ID: "grammar/bash", Version: "v0.25.1", Source: Source{Type: "git", URL: "https://github.com/tree-sitter/tree-sitter-bash", Ref: "v0.25.1"}}); r.Status != StatusUpdate || r.Latest != "v0.26.0" || r.Level != "minor" {
		t.Fatalf("tagged grammar: %+v", r)
	}
	// Newer than every tag and still at upstream HEAD: up to date.
	if r := check(Component{ID: "elpa/ace-window", Version: "0.10.0", Source: Source{Type: "git", URL: "https://github.com/abo-abo/ace-window", Commit: "same"}}); r.Status != StatusCurrent {
		t.Fatalf("untagged newer release: %+v", r)
	}
	// A MELPA snapshot has no release to compare; report new upstream commits instead.
	if r := check(Component{ID: "elpa/apheleia", Version: "20260613.2149", Source: Source{Type: "git", URL: "https://github.com/radian-software/apheleia", Commit: "oldcommit"}}); r.Status != StatusMoved {
		t.Fatalf("snapshot: %+v", r)
	}
	if r := check(Component{ID: "toolchain/basedpyright", Version: "1.39.10", Source: Source{Type: "archive", URL: "https://registry.npmjs.org/basedpyright/-/basedpyright-1.39.10.tgz"}}); r.Status != StatusUpdate || r.Latest != "1.40.1" {
		t.Fatalf("npm archive: %+v", r)
	}
	r := check(Component{ID: "toolchain/node", Version: "v24.19.0", Source: Source{Type: "archive", URL: "https://nodejs.org/dist/v24.19.0/node-v24.19.0-darwin-arm64.tar.gz"}})
	if r.Status != StatusUpdate || r.Latest != "v24.20.0" || !strings.Contains(r.Detail, "v25.1.0") {
		t.Fatalf("node stays on its major and mentions the newest line: %+v", r)
	}
	if r := check(Component{ID: "toolchain/gopls", Version: "v0.23.0", Source: Source{Type: "git", URL: "https://go.googlesource.com/tools", ArtifactURL: "https://proxy.golang.org/golang.org/x/tools/gopls/@v/v0.23.0.zip"}}); r.Status != StatusUpdate || r.Latest != "v0.24.0" {
		t.Fatalf("go proxy: %+v", r)
	}
	if r := check(Component{ID: "font/nanum", Version: "2.5", Source: Source{Type: "archive", URL: "https://github.com/naver/nanumfont/releases/download/VER2.5/NanumGothicCoding-2.5.zip"}}); r.Status != StatusCurrent {
		t.Fatalf("release archive: %+v", r)
	}
	if r := check(Component{ID: "font/nerd", Version: "sha256:" + sha256Hex([]byte("font-v1")), Source: Source{Type: "download", URL: "https://raw.githubusercontent.com/o/r/main/NFM.ttf"}}); r.Status != StatusMoved {
		t.Fatalf("content-addressed download changed upstream: %+v", r)
	}
	if r := check(Component{ID: "toolchain/imoogi-cli", Version: "1.0.0", Source: Source{Type: "git", URL: "https://github.com/karohani/imoogi-emacs"}}); r.Status != StatusSkipped {
		t.Fatalf("own repository: %+v", r)
	}
	if r := check(Component{ID: "elpa/gone", Version: "1.0", Source: Source{Type: "git", URL: "https://github.com/gone/away", Commit: "x"}}); r.Status != StatusUnknown || r.Detail == "" {
		t.Fatalf("unreachable upstream is reported, not hidden: %+v", r)
	}
}

func TestEmbeddedMermaidVersionIsReadFromTheBundle(t *testing.T) {
	bundle := []byte(`x={version:"12.0.0"},Ab=i(()=>Cd.version,"getVersion")`)
	if got := mermaidBundleVersion(bundle); got != "12.0.0" {
		t.Fatalf("mermaid version: %q", got)
	}
}

func TestWatchReportsReleasePace(t *testing.T) {
	now := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	remote := fakeRemote{
		npm: map[string]NPMRelease{"pele": {Version: "1.13.0"}},
		times: map[string]map[string]time.Time{"pele": {
			"created": now.AddDate(0, -2, 0), "modified": now,
			"1.0.0": now.AddDate(0, 0, -40), "1.12.0": now.AddDate(0, 0, -19), "1.13.0": now.AddDate(0, 0, -14),
		}},
	}
	r := watchNPM(context.Background(), remote, now, "watch/pele", "pele", "mermaid.js stays until Pele settles")
	if r.Status != StatusWatch || r.Latest != "1.13.0" || !strings.Contains(r.Detail, "2 releases in 30 days") || !strings.Contains(r.Detail, "2026-10-31") {
		t.Fatalf("watch: %+v", r)
	}
}

func TestParseGoListUpdatesKeepsDirectRequirementsOnly(t *testing.T) {
	stream := `{"Path":"github.com/karohani/imoogi-emacs","Main":true}
{"Path":"gopkg.in/yaml.v3","Version":"v3.0.1"}
{"Path":"modernc.org/sqlite","Version":"v1.56.0","Update":{"Path":"modernc.org/sqlite","Version":"v1.57.2"}}
{"Path":"golang.org/x/sys","Version":"v0.47.0","Indirect":true,"Update":{"Version":"v0.48.0"}}`
	got, err := parseGoListUpdates(strings.NewReader(stream))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "go/gopkg.in/yaml.v3" || got[0].Status != StatusCurrent ||
		got[1].Status != StatusUpdate || got[1].Latest != "v1.57.2" || got[1].Level != "minor" {
		t.Fatalf("go list: %+v", got)
	}
}

func TestParseNPMOutdated(t *testing.T) {
	got, err := parseNPMOutdated([]byte(`{"webpack":{"current":"5.90.0","wanted":"5.90.3","latest":"6.0.0"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "npm/webpack" || got[0].Latest != "6.0.0" || got[0].Level != "major" || !strings.Contains(got[0].Detail, "5.90.3") {
		t.Fatalf("npm outdated: %+v", got)
	}
}

func TestElpaPackagesWithoutAGitURLUseTheArchives(t *testing.T) {
	ctx := context.Background()
	remote := fakeRemote{
		failed:   map[string]bool{"https://elpa.gnu.org/packages/csv-mode.html": true, "https://web-mode.org/": true, "https://example.org/lua": true},
		archives: map[string][2]string{"csv-mode": {"gnu", "1.28"}, "web-mode": {"nongnu", "17.3.23"}, "lua-mode": {"melpa", "20250301.1200"}},
	}
	check := func(id, version, url string) Result {
		return checkComponent(ctx, remote, "elpa", Component{ID: id, Version: version, Source: Source{Type: "git", URL: url, Commit: "c"}})
	}
	if r := check("elpa/csv-mode", "1.27", "https://elpa.gnu.org/packages/csv-mode.html"); r.Status != StatusUpdate || r.Latest != "1.28" || !strings.Contains(r.Detail, "gnu") {
		t.Fatalf("gnu archive: %+v", r)
	}
	if r := check("elpa/web-mode", "17.3.23", "https://web-mode.org/"); r.Status != StatusCurrent {
		t.Fatalf("nongnu archive: %+v", r)
	}
	// MELPA versions are snapshot dates; only a snapshot may be compared with them.
	if r := check("elpa/lua-mode", "20221027", "https://example.org/lua"); r.Status != StatusUpdate || r.Latest != "20250301.1200" {
		t.Fatalf("melpa snapshot: %+v", r)
	}
	if r := check("elpa/markdown-mode", "2.8", "https://example.org/lua"); r.Status != StatusUnknown {
		t.Fatalf("a release must not be compared with a MELPA date: %+v", r)
	}
}

func TestRepositoryMetadataAndNodePrereleases(t *testing.T) {
	ctx := context.Background()
	if r := checkComponent(ctx, fakeRemote{}, "metadata", Component{ID: "metadata/package-lock", Kind: "repository-metadata", Version: "1", Source: Source{Type: "repository", URL: "scripts/vendor.el"}}); r.Status != StatusSkipped {
		t.Fatalf("repository metadata: %+v", r)
	}
	remote := fakeRemote{node: []NodeRelease{{Version: "v27.0.0-alpha.1"}, {Version: "v26.1.0", LTS: true}, {Version: "v24.19.0", LTS: true}}}
	r := checkComponent(ctx, remote, "toolchains", Component{ID: "toolchain/node", Version: "v24.19.0", Source: Source{Type: "archive", URL: "https://nodejs.org/dist/v24.19.0/node.tar.gz"}})
	if strings.Contains(r.Detail, "alpha") || !strings.Contains(r.Detail, "v26.1.0") {
		t.Fatalf("node must ignore pre-releases: %+v", r)
	}
}

func TestParsePackageArchives(t *testing.T) {
	gnu := parseArchiveContents([]byte(`(1 (csv-mode . [(1 28) ((emacs (27 1))) "Major mode" tar ((:url . "x"))])
 (lv . [(0 15 0) nil "Other" single nil]))`))
	if gnu["csv-mode"] != "1.28" || gnu["lv"] != "0.15.0" {
		t.Fatalf("archive-contents: %v", gnu)
	}
	melpa := parseMelpaArchive([]byte(`{"lua-mode":{"ver":[20250301,1200],"deps":null}}`))
	if melpa["lua-mode"] != "20250301.1200" {
		t.Fatalf("melpa: %v", melpa)
	}
}
