package doctor

import (
	"bufio"
	"bytes"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// SystemEnv inspects a real checkout with go and git.
type SystemEnv struct {
	RootDir, HomeDir string
}

// NewSystemEnv locates the repository that contains dir.
func NewSystemEnv(dir string) (SystemEnv, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return SystemEnv{}, fmt.Errorf("not inside the imoogi-emacs repository: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return SystemEnv{}, err
	}
	return SystemEnv{RootDir: strings.TrimSpace(string(out)), HomeDir: home}, nil
}

func (e SystemEnv) Root() string { return e.RootDir }
func (e SystemEnv) Home() string { return e.HomeDir }

// The doctor itself runs with `go run`, so it is never a bin/ tool.
const self = "imoogi-doctor"

func (e SystemEnv) Tools() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(e.RootDir, "cmd"))
	if err != nil {
		return nil, err
	}
	var tools []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != self {
			tools = append(tools, entry.Name())
		}
	}
	return tools, nil
}

func (e SystemEnv) BinEntries() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(e.RootDir, "bin"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

var toolchainRE = regexp.MustCompile(`(?m)^toolchain\s+(go\S+)`)
var goLineRE = regexp.MustCompile(`(?m)^go\s+(\S+)`)

// RequiredGo is the toolchain go.work pins (workspace mode reads go.work), then go.mod.
func (e SystemEnv) RequiredGo() (string, error) {
	for _, file := range []string{"go.work", "go.mod"} {
		data, err := os.ReadFile(filepath.Join(e.RootDir, file))
		if err != nil {
			continue
		}
		if m := toolchainRE.FindSubmatch(data); m != nil {
			return string(m[1]), nil
		}
		if m := goLineRE.FindSubmatch(data); m != nil {
			return "go" + string(m[1]), nil
		}
	}
	return "", fmt.Errorf("no go or toolchain line in go.work or go.mod")
}

func (e SystemEnv) ReadBuildInfo(path string) (BuildInfo, error) {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return BuildInfo{}, err
	}
	out := BuildInfo{GoVersion: info.GoVersion, Main: info.Path}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			out.Revision = s.Value
		case "vcs.modified":
			out.Modified = s.Value == "true"
		}
	}
	return out, nil
}

func (e SystemEnv) Deps(tool string) ([]string, error) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{if .Module}}{{if .Module.Main}}{{.Dir}}{{end}}{{end}}", "./cmd/"+tool)
	cmd.Dir = e.RootDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list %s: %v %s", tool, err, strings.TrimSpace(stderr.String()))
	}
	var dirs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if rel, err := filepath.Rel(e.RootDir, line); err == nil && line != "" {
			dirs = append(dirs, rel)
		}
	}
	return dirs, nil
}

// ChangedSince lists tracked files that differ from revision in the working tree,
// plus new untracked files, under paths.
func (e SystemEnv) ChangedSince(revision string, paths []string) ([]string, error) {
	diff := exec.Command("git", append([]string{"-C", e.RootDir, "diff", "--name-only", revision, "--"}, paths...)...)
	var stderr bytes.Buffer
	diff.Stderr = &stderr
	out, err := diff.Output()
	if err != nil {
		return nil, fmt.Errorf("compare with %s: %s", short(revision), strings.TrimSpace(stderr.String()))
	}
	untracked, err := exec.Command("git", append([]string{"-C", e.RootDir, "ls-files", "--others", "--exclude-standard", "--"}, paths...)...).Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(string(out)+string(untracked), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func (e SystemEnv) Readlink(path string) (string, error) { return os.Readlink(path) }

func (e SystemEnv) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (e SystemEnv) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (e SystemEnv) LookPath(name string) (string, error) { return exec.LookPath(name) }
func (e SystemEnv) Getenv(key string) string             { return os.Getenv(key) }

// handshakeTimeout bounds the ACP initialize exchange.
var handshakeTimeout = 5 * time.Second

const initializeRequest = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{}}}` + "\n"

// Handshake starts launcher, sends an ACP initialize request on stdin and waits
// for the response with id 1 on stdout. The adapter runs in its own process
// group, killed as a whole afterwards, so helpers it spawned (codex-acp starts
// codex app-server) do not outlive the check.
//
// @MX:WARN: [AUTO] spawns the adapter and reads its stdout in a goroutine
// @MX:REASON: the reader goroutine ends when the process group is killed and its pipe closes; timeout + group kill are what bound it
func (e SystemEnv) Handshake(launcher string) error {
	cmd := exec.Command(launcher)
	cmd.Dir = os.TempDir()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	// An owned pipe rather than StdoutPipe: Wait must not close the read end
	// while the reader is still draining the response.
	stdout, stdoutW, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdout.Close()
	cmd.Stdout = stdoutW
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.WaitDelay = time.Second
	startErr := cmd.Start()
	stdoutW.Close()
	if startErr != nil {
		return startErr
	}
	waited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(waited) }()
	defer func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-waited
		_ = stdout.Close() // unblocks the reader if a helper still holds the pipe
	}()
	if _, err := io.WriteString(stdin, initializeRequest); err != nil {
		return fmt.Errorf("send initialize: %w", err)
	}
	result := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64<<10), 8<<20)
		for scanner.Scan() {
			var msg struct {
				ID     *int            `json:"id"`
				Result json.RawMessage `json:"result"`
				Error  *struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if json.Unmarshal(scanner.Bytes(), &msg) != nil || msg.ID == nil || *msg.ID != 1 {
				continue
			}
			switch {
			case msg.Error != nil:
				result <- fmt.Errorf("adapter error: %s", msg.Error.Message)
			case len(msg.Result) == 0:
				result <- errors.New("initialize response has no result")
			default:
				result <- nil
			}
			return
		}
		result <- errExited
	}()
	select {
	case err := <-result:
		if errors.Is(err, errExited) {
			// stderr is safe to read only once Wait has returned.
			select {
			case <-waited:
				return fmt.Errorf("adapter exited without answering: %s", lastLines(stderr.String()))
			case <-time.After(2 * time.Second):
				return errors.New("adapter closed stdout without answering")
			}
		}
		return err
	case <-time.After(handshakeTimeout):
		return fmt.Errorf("no initialize response within %s", handshakeTimeout)
	}
}

var errExited = errors.New("exited")

func lastLines(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}
	return strings.Join(lines, " | ")
}
