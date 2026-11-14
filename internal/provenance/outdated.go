package provenance

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Status of one bundled component compared with its upstream.
const (
	StatusCurrent = "current" // no newer release upstream
	StatusUpdate  = "update"  // a newer release exists
	StatusMoved   = "moved"   // pinned to a commit or file that upstream has since changed; no release to compare
	StatusWatch   = "watch"   // not bundled; tracked as a candidate
	StatusUnknown = "unknown" // upstream could not be checked
	StatusSkipped = "skipped" // nothing to compare (this repository)
)

// Result reports one component. It is read-only advice: nothing is downloaded or changed.
type Result struct {
	Domain   string `json:"domain"`
	ID       string `json:"id"`
	Current  string `json:"current"`
	Latest   string `json:"latest,omitempty"`
	Status   string `json:"status"`
	Level    string `json:"level,omitempty"` // major, minor, patch or release
	Detail   string `json:"detail,omitempty"`
	Source   string `json:"source,omitempty"`
	Workflow string `json:"workflow,omitempty"` // how this component is refreshed
}

type NPMRelease struct{ Version string }

type NodeRelease struct {
	Version string
	LTS     bool
}

// Remote answers questions about upstreams. The real implementation uses git and HTTPS;
// tests use fixed tables.
type Remote interface {
	Tags(ctx context.Context, url string) ([]string, error)
	Head(ctx context.Context, url string) (string, error)
	NPMLatest(ctx context.Context, name string) (NPMRelease, error)
	NPMTimes(ctx context.Context, name string) (map[string]time.Time, error)
	NodeReleases(ctx context.Context) ([]NodeRelease, error)
	GoProxyLatest(ctx context.Context, module string) (string, error)
	// ArchiveVersion looks a package up in GNU ELPA, then NonGNU ELPA, then (only
	// when snapshots is true) MELPA, whose versions are snapshot dates.
	ArchiveVersion(ctx context.Context, name string, snapshots bool) (archive, version string, err error)
	Fetch(ctx context.Context, url string) ([]byte, error)
}

// ProjectTools inspects dependencies that live outside the provenance manifests.
type ProjectTools interface {
	GoModules(ctx context.Context, root string) ([]Result, error)
	Govulncheck(ctx context.Context, root string) Result
	NPMOutdated(ctx context.Context, dir string) ([]Result, error)
}

var digitsRE = regexp.MustCompile(`\d+`)

func versionNumbers(v string) []int64 {
	var out []int64
	for _, part := range digitsRE.FindAllString(v, -1) {
		n, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}

// compareVersions reports whether latest is newer than current and by how much.
func compareVersions(current, latest string) (bool, string) {
	a, b := versionNumbers(current), versionNumbers(latest)
	for len(a) < len(b) {
		a = append(a, 0)
	}
	for len(b) < len(a) {
		b = append(b, 0)
	}
	for i := range a {
		if a[i] == b[i] {
			continue
		}
		if b[i] < a[i] {
			return false, ""
		}
		switch {
		case len(a) > 0 && a[0] >= 1900: // date-stamped releases have no semantic levels
			return true, "release"
		case i == 0:
			return true, "major"
		case i == 1:
			return true, "minor"
		default:
			return true, "patch"
		}
	}
	return false, ""
}

func leadingNonDigits(s string) string {
	for i, r := range s {
		if r >= '0' && r <= '9' {
			return s[:i]
		}
	}
	return s
}

// A bare version and a v-prefixed tag are the same style.
func prefixKey(p string) string { return strings.TrimSuffix(strings.ToLower(p), "v") }

var prereleaseRE = regexp.MustCompile(`(?i)(alpha|beta|rc\d*|pre|dev|nightly|snapshot|canary)`)

func isPrerelease(v string) bool {
	return prereleaseRE.MatchString(strings.TrimPrefix(v, leadingNonDigits(v)))
}

// latestTag returns the newest release tag written in the same style as current.
func latestTag(current string, tags []string) string {
	want, pre := prefixKey(leadingNonDigits(current)), isPrerelease(current)
	best := ""
	for _, tag := range tags {
		prefix := leadingNonDigits(tag)
		if prefix == tag || prefixKey(prefix) != want || (!pre && isPrerelease(tag)) {
			continue
		}
		if best == "" {
			best = tag
		} else if newer, _ := compareVersions(best, tag); newer {
			best = tag
		}
	}
	return best
}

var snapshotRE = regexp.MustCompile(`^\d{8}\.\d+$`)

// A bare date (lua-mode's 20221027) is also a snapshot for archive lookups.
var dateVersionRE = regexp.MustCompile(`^\d{8}(\.\d+)?$`)

// releaseOf maps a GitHub release download or raw file URL to its repository and tag.
func releaseOf(url string) (repo, ref string) {
	if rest, ok := strings.CutPrefix(url, "https://github.com/"); ok {
		parts := strings.Split(rest, "/")
		if len(parts) >= 5 && parts[2] == "releases" && parts[3] == "download" {
			return "https://github.com/" + parts[0] + "/" + parts[1], parts[4]
		}
	}
	if rest, ok := strings.CutPrefix(url, "https://raw.githubusercontent.com/"); ok {
		if parts := strings.Split(rest, "/"); len(parts) >= 2 {
			return "https://github.com/" + parts[0] + "/" + parts[1], ""
		}
	}
	return "", ""
}

func npmPackageOf(url string) string {
	rest, ok := strings.CutPrefix(url, "https://registry.npmjs.org/")
	if !ok {
		return ""
	}
	name, _, ok := strings.Cut(rest, "/-/")
	if !ok {
		return ""
	}
	return name
}

func goModuleOf(url string) string {
	rest, ok := strings.CutPrefix(url, "https://proxy.golang.org/")
	if !ok {
		return ""
	}
	module, _, ok := strings.Cut(rest, "/@v/")
	if !ok {
		return ""
	}
	return module
}

func sha256Hex(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

func short(commit string) string {
	if len(commit) > 10 {
		return commit[:10]
	}
	return commit
}

func versioned(r Result, latest string) Result {
	r.Latest = latest
	if newer, level := compareVersions(r.Current, latest); newer {
		r.Status, r.Level = StatusUpdate, level
	} else {
		r.Status = StatusCurrent
	}
	return r
}

func unknown(r Result, err error) Result {
	r.Status, r.Detail = StatusUnknown, err.Error()
	return r
}

// checkComponent compares one provenance component with its upstream.
func checkComponent(ctx context.Context, remote Remote, domain string, c Component) Result {
	r := Result{Domain: domain, ID: c.ID, Current: c.Version, Source: c.Source.URL, Workflow: c.Workflow}
	src := c.Source
	if strings.Contains(src.URL, "github.com/karohani/imoogi-emacs") || c.Kind == "repository-metadata" {
		r.Status, r.Detail = StatusSkipped, "built from this repository"
		return r
	}
	if module := goModuleOf(src.ArtifactURL); module != "" {
		latest, err := remote.GoProxyLatest(ctx, module)
		if err != nil {
			return unknown(r, err)
		}
		return versioned(r, latest)
	}
	for _, url := range []string{src.URL, src.ArtifactURL} {
		if name := npmPackageOf(url); name != "" {
			rel, err := remote.NPMLatest(ctx, name)
			if err != nil {
				return unknown(r, err)
			}
			return versioned(r, rel.Version)
		}
	}
	if strings.HasPrefix(src.URL, "https://nodejs.org/dist/") {
		return checkNode(ctx, remote, r)
	}
	if src.Type == "download" && strings.HasPrefix(c.Version, "sha256:") {
		body, err := remote.Fetch(ctx, src.URL)
		if err != nil {
			return unknown(r, err)
		}
		if sha256Hex(body) == strings.TrimPrefix(c.Version, "sha256:") {
			r.Status = StatusCurrent
		} else {
			r.Status, r.Detail = StatusMoved, "upstream file changed (sha256 "+short(sha256Hex(body))+")"
		}
		return r
	}
	url, ref := src.URL, src.Ref
	if src.Type != "git" {
		if repo, tag := releaseOf(src.URL); repo != "" {
			url, ref = repo, tag
		} else {
			r.Status, r.Detail = StatusUnknown, "no upstream rule for "+src.Type+" source"
			return r
		}
	}
	compareAs := ref
	if compareAs == "" {
		compareAs = c.Version
	}
	if !snapshotRE.MatchString(c.Version) {
		tags, err := remote.Tags(ctx, url)
		if err != nil {
			return gitFallback(ctx, remote, domain, r, err)
		}
		if tag := latestTag(compareAs, tags); tag != "" {
			if newer, level := compareVersions(compareAs, tag); newer {
				r.Latest, r.Status, r.Level = tag, StatusUpdate, level
				return r
			}
			r.Latest, r.Status = tag, StatusCurrent
			return r
		}
	}
	// A snapshot or an untagged project: compare the pinned commit with upstream HEAD.
	if src.Commit == "" {
		r.Status, r.Detail = StatusUnknown, "no comparable tag and no pinned commit"
		return r
	}
	head, err := remote.Head(ctx, url)
	if err != nil {
		return gitFallback(ctx, remote, domain, r, err)
	}
	if head == src.Commit {
		r.Status = StatusCurrent
		return r
	}
	r.Status, r.Detail = StatusMoved, "upstream HEAD "+short(head)+" differs from pinned "+short(src.Commit)
	return r
}

// gitFallback handles an Emacs package whose recorded URL is a homepage or an
// ELPA page rather than a git repository: ask the package archives instead.
func gitFallback(ctx context.Context, remote Remote, domain string, r Result, gitErr error) Result {
	if domain != "elpa" {
		return unknown(r, gitErr)
	}
	name := strings.TrimPrefix(r.ID, "elpa/")
	archive, version, err := remote.ArchiveVersion(ctx, name, dateVersionRE.MatchString(r.Current))
	if err != nil {
		return unknown(r, fmt.Errorf("%v; %v", gitErr, err))
	}
	r = versioned(r, version)
	r.Detail = "from the " + archive + " package archive"
	return r
}

// Node stays on its major line; a newer line is mentioned, not proposed.
func checkNode(ctx context.Context, remote Remote, r Result) Result {
	releases, err := remote.NodeReleases(ctx)
	if err != nil {
		return unknown(r, err)
	}
	major := versionNumbers(r.Current)
	sameLine, newestLTS, newest := "", "", ""
	for _, rel := range releases {
		nums := versionNumbers(rel.Version)
		if len(nums) == 0 || isPrerelease(rel.Version) {
			continue
		}
		if newest == "" || first(compareVersions(newest, rel.Version)) {
			newest = rel.Version
		}
		if rel.LTS && (newestLTS == "" || first(compareVersions(newestLTS, rel.Version))) {
			newestLTS = rel.Version
		}
		if len(major) > 0 && nums[0] == major[0] && (sameLine == "" || first(compareVersions(sameLine, rel.Version))) {
			sameLine = rel.Version
		}
	}
	if sameLine == "" {
		sameLine = r.Current
	}
	r = versioned(r, sameLine)
	r.Detail = "newest LTS " + newestLTS + "; newest release " + newest
	return r
}

func first(b bool, _ string) bool { return b }

// watchNPM reports a candidate that is not bundled, with its release pace.
func watchNPM(ctx context.Context, remote Remote, now time.Time, id, name, note string) Result {
	r := Result{Domain: "watch", ID: id, Current: "-", Status: StatusWatch, Source: "https://www.npmjs.com/package/" + name, Detail: note}
	rel, err := remote.NPMLatest(ctx, name)
	if err != nil {
		return unknown(r, err)
	}
	r.Latest = rel.Version
	times, err := remote.NPMTimes(ctx, name)
	if err != nil {
		return r
	}
	recent := 0
	for version, at := range times {
		if version != "created" && version != "modified" && now.Sub(at) <= 30*24*time.Hour {
			recent++
		}
	}
	published := ""
	if at, ok := times[rel.Version]; ok {
		published = at.UTC().Format("2006-01-02")
	}
	r.Detail = fmt.Sprintf("%s published %s; %d releases in 30 days; %s", rel.Version, published, recent, note)
	return r
}

var mermaidVersionRE = regexp.MustCompile(`version:"([0-9]+\.[0-9]+\.[0-9]+[^"\s]*)"\},[A-Za-z0-9_$]+=i\(\(\)=>[A-Za-z0-9_$]+\.version,"getVersion"\)`)

// mermaidBundleVersion reads the version the minified Mermaid bundle reports.
func mermaidBundleVersion(bundle []byte) string {
	if m := mermaidVersionRE.FindSubmatch(bundle); m != nil {
		return string(m[1])
	}
	return ""
}

const mermaidBundle = "internal/orgpreview/web/mermaid.min.js"

func checkEmbeddedMermaid(ctx context.Context, remote Remote, root string) Result {
	r := Result{Domain: "embedded", ID: "orgpreview/mermaid", Source: "https://www.npmjs.com/package/mermaid",
		Workflow: "replace " + mermaidBundle + " with dist/mermaid.min.js from npm and its LICENSE; go test ./internal/orgpreview/"}
	bundle, err := os.ReadFile(filepath.Join(root, mermaidBundle))
	if err != nil {
		return unknown(r, err)
	}
	if r.Current = mermaidBundleVersion(bundle); r.Current == "" {
		r.Status, r.Detail = StatusUnknown, "version not found in "+mermaidBundle
		return r
	}
	rel, err := remote.NPMLatest(ctx, "mermaid")
	if err != nil {
		return unknown(r, err)
	}
	return versioned(r, rel.Version)
}

// Outdated checks every provenance component plus the bundled assets and
// dependencies outside it. It never modifies the repository.
func Outdated(ctx context.Context, root, indexPath string, remote Remote, tools ProjectTools, now time.Time) ([]Result, error) {
	var index Index
	if err := loadStrict(filepath.Join(root, indexPath), &index); err != nil {
		return nil, err
	}
	type job struct {
		domain string
		c      Component
	}
	var jobs []job
	for _, manifest := range index.Manifests {
		var d Domain
		if err := loadStrict(filepath.Join(root, manifest), &d); err != nil {
			return nil, err
		}
		for _, c := range d.Components {
			jobs = append(jobs, job{d.Domain, c})
		}
	}
	results := make([]Result, len(jobs))
	var wg sync.WaitGroup
	slots := make(chan struct{}, 8)
	for i, j := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			results[i] = checkComponent(ctx, remote, j.domain, j.c)
		}()
	}
	wg.Wait()
	results = append(results, checkEmbeddedMermaid(ctx, remote, root))
	results = append(results, watchNPM(ctx, remote, now, "watch/pele", "pele",
		"Obsidian's Mermaid renderer; mermaid.js stays until Pele settles"))
	if tools != nil {
		if mods, err := tools.GoModules(ctx, root); err != nil {
			results = append(results, Result{Domain: "go", ID: "go/modules", Status: StatusUnknown, Detail: err.Error()})
		} else {
			results = append(results, mods...)
		}
		results = append(results, tools.Govulncheck(ctx, root))
		if deps, err := tools.NPMOutdated(ctx, filepath.Join(root, "web-clipper")); err != nil {
			results = append(results, Result{Domain: "web-clipper", ID: "npm/web-clipper", Status: StatusUnknown, Detail: err.Error()})
		} else {
			results = append(results, deps...)
		}
	}
	return results, nil
}

// parseGoListUpdates reads `go list -m -u -json all` and keeps direct requirements.
func parseGoListUpdates(r io.Reader) ([]Result, error) {
	dec := json.NewDecoder(bufio.NewReader(r))
	var out []Result
	for {
		var m struct {
			Path, Version  string
			Main, Indirect bool
			Update         *struct{ Version string }
		}
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if m.Main || m.Indirect {
			continue
		}
		res := Result{Domain: "go", ID: "go/" + m.Path, Current: m.Version, Status: StatusCurrent,
			Source: "https://pkg.go.dev/" + m.Path, Workflow: "go get " + m.Path + "@latest && go mod tidy"}
		if m.Update != nil && m.Update.Version != "" {
			res = versioned(res, m.Update.Version)
		}
		out = append(out, res)
	}
	return out, nil
}

// parseNPMOutdated reads `npm outdated --json`, which lists only outdated direct dependencies.
func parseNPMOutdated(b []byte) ([]Result, error) {
	var deps map[string]struct{ Current, Wanted, Latest string }
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(b, &deps); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(deps))
	for name := range deps {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []Result
	for _, name := range names {
		d := deps[name]
		res := versioned(Result{Domain: "web-clipper", ID: "npm/" + name, Current: d.Current,
			Source: "https://www.npmjs.com/package/" + name, Workflow: "cd web-clipper && npm install " + name + "@latest"}, d.Latest)
		res.Detail = "within the semver range: " + d.Wanted
		out = append(out, res)
	}
	return out, nil
}
