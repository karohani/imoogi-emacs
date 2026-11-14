package provenance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NetRemote checks upstreams with `git ls-remote` and plain HTTPS GETs. It sends no
// credentials and identifies itself only as imoogi-provenance.
type NetRemote struct {
	Client   *http.Client
	archives *archiveCache
}

type archiveCache struct {
	once     sync.Once
	versions map[string]map[string]string // archive -> package -> version
}

func NewNetRemote() NetRemote {
	return NetRemote{Client: &http.Client{Timeout: 60 * time.Second}, archives: &archiveCache{}}
}

var archiveOrder = []struct{ name, url string }{
	{"gnu", "https://elpa.gnu.org/packages/archive-contents"},
	{"nongnu", "https://elpa.nongnu.org/nongnu/archive-contents"},
	{"melpa", "https://melpa.org/archive.json"},
}

func (n NetRemote) ArchiveVersion(ctx context.Context, name string, snapshots bool) (string, string, error) {
	n.archives.once.Do(func() {
		n.archives.versions = map[string]map[string]string{}
		for _, a := range archiveOrder {
			body, err := n.Fetch(ctx, a.url)
			if err != nil {
				continue
			}
			if a.name == "melpa" {
				n.archives.versions[a.name] = parseMelpaArchive(body)
			} else {
				n.archives.versions[a.name] = parseArchiveContents(body)
			}
		}
	})
	for _, a := range archiveOrder {
		if a.name == "melpa" && !snapshots {
			continue
		}
		if v, ok := n.archives.versions[a.name][name]; ok {
			return a.name, v, nil
		}
	}
	return "", "", fmt.Errorf("%s is in no reachable package archive", name)
}

func (n NetRemote) git(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=true")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func (n NetRemote) Tags(ctx context.Context, url string) ([]string, error) {
	out, err := n.git(ctx, "ls-remote", "--tags", "--refs", url)
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, line := range strings.Split(out, "\n") {
		if _, ref, ok := strings.Cut(line, "\t"); ok {
			tags = append(tags, strings.TrimPrefix(ref, "refs/tags/"))
		}
	}
	return tags, nil
}

func (n NetRemote) Head(ctx context.Context, url string) (string, error) {
	out, err := n.git(ctx, "ls-remote", url, "HEAD")
	if err != nil {
		return "", err
	}
	commit, _, _ := strings.Cut(strings.TrimSpace(out), "\t")
	if commit == "" {
		return "", errors.New("upstream has no HEAD")
	}
	return commit, nil
}

func (n NetRemote) get(ctx context.Context, url string, out any) error {
	body, err := n.Fetch(ctx, url)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (n NetRemote) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "imoogi-provenance")
	resp, err := n.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 64<<20))
}

func (n NetRemote) NPMLatest(ctx context.Context, name string) (NPMRelease, error) {
	var doc struct{ Version string }
	if err := n.get(ctx, "https://registry.npmjs.org/"+name+"/latest", &doc); err != nil {
		return NPMRelease{}, err
	}
	return NPMRelease{Version: doc.Version}, nil
}

func (n NetRemote) NPMTimes(ctx context.Context, name string) (map[string]time.Time, error) {
	var doc struct {
		Time map[string]time.Time `json:"time"`
	}
	if err := n.get(ctx, "https://registry.npmjs.org/"+name, &doc); err != nil {
		return nil, err
	}
	return doc.Time, nil
}

func (n NetRemote) NodeReleases(ctx context.Context) ([]NodeRelease, error) {
	var index []struct {
		Version string          `json:"version"`
		LTS     json.RawMessage `json:"lts"`
	}
	if err := n.get(ctx, "https://nodejs.org/dist/index.json", &index); err != nil {
		return nil, err
	}
	out := make([]NodeRelease, 0, len(index))
	for _, rel := range index {
		out = append(out, NodeRelease{Version: rel.Version, LTS: string(rel.LTS) != "false" && len(rel.LTS) > 0})
	}
	return out, nil
}

func (n NetRemote) GoProxyLatest(ctx context.Context, module string) (string, error) {
	var doc struct{ Version string }
	if err := n.get(ctx, "https://proxy.golang.org/"+module+"/@latest", &doc); err != nil {
		return "", err
	}
	return doc.Version, nil
}

// LocalTools runs the Go and npm toolchains already on this machine.
type LocalTools struct{}

func run(ctx context.Context, dir, name string, args ...string) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out, exit.ExitCode(), nil
	}
	if err != nil {
		return out, -1, fmt.Errorf("%s: %v %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return out, 0, nil
}

func (LocalTools) GoModules(ctx context.Context, root string) ([]Result, error) {
	out, code, err := run(ctx, root, "go", "list", "-m", "-u", "-json", "all")
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, fmt.Errorf("go list -m -u exited %d", code)
	}
	return parseGoListUpdates(bytes.NewReader(out))
}

func (LocalTools) Govulncheck(ctx context.Context, root string) Result {
	r := Result{Domain: "go", ID: "go/govulncheck", Current: "-", Source: "https://go.dev/security/vuln",
		Workflow: "go install golang.org/x/vuln/cmd/govulncheck@latest"}
	if _, err := exec.LookPath("govulncheck"); err != nil {
		r.Status, r.Detail = StatusUnknown, "govulncheck is not installed; vulnerability check skipped"
		return r
	}
	out, code, err := run(ctx, root, "govulncheck", "./...")
	if err != nil {
		return unknown(r, err)
	}
	switch code {
	case 0:
		r.Status, r.Detail = StatusCurrent, "no known vulnerabilities in called code"
	case 3:
		r.Status, r.Detail = StatusUpdate, firstLines(string(out), 12)
	default:
		r.Status, r.Detail = StatusUnknown, fmt.Sprintf("govulncheck exited %d: %s", code, firstLines(string(out), 6))
	}
	return r
}

func (LocalTools) NPMOutdated(ctx context.Context, dir string) ([]Result, error) {
	if _, err := exec.LookPath("npm"); err != nil {
		return nil, errors.New("npm is not installed")
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules")); err != nil {
		return nil, errors.New("web-clipper/node_modules is missing; run npm ci there first")
	}
	// npm exits 1 when something is outdated; the JSON on stdout is the answer either way.
	out, _, err := run(ctx, dir, "npm", "outdated", "--json")
	if err != nil {
		return nil, err
	}
	return parseNPMOutdated(out)
}

func firstLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = append(lines[:n], "…")
	}
	return strings.Join(lines, "\n")
}

// archive-contents is an Elisp list of (NAME . [(VERSION...) DEPS DESC KIND EXTRA]).
var archiveEntryRE = regexp.MustCompile(`\(([^\s()]+)\s*\.\s*\[\(([0-9 ]+)\)`)

func parseArchiveContents(body []byte) map[string]string {
	out := map[string]string{}
	for _, m := range archiveEntryRE.FindAllSubmatch(body, -1) {
		out[string(m[1])] = strings.Join(strings.Fields(string(m[2])), ".")
	}
	return out
}

func parseMelpaArchive(body []byte) map[string]string {
	var doc map[string]struct {
		Ver []int64 `json:"ver"`
	}
	out := map[string]string{}
	if json.Unmarshal(body, &doc) != nil {
		return out
	}
	for name, pkg := range doc {
		parts := make([]string, len(pkg.Ver))
		for i, n := range pkg.Ver {
			parts[i] = strconv.FormatInt(n, 10)
		}
		out[name] = strings.Join(parts, ".")
	}
	return out
}
