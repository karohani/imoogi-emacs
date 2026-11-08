// Command org-clipper-host is the Chrome Native Messaging host that saves web
// clips as Org files, plus the CLI that configures and installs it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/karohani/obsidian-clipper/native-host/internal/config"
	"github.com/karohani/obsidian-clipper/native-host/internal/fetch"
	"github.com/karohani/obsidian-clipper/native-host/internal/host"
	"github.com/karohani/obsidian-clipper/native-host/internal/install"
)

// version is overridable at build time: -ldflags "-X main.version=1.2.3".
var version = "0.1.0"

const usage = `org-clipper-host — Chrome Native Messaging host that saves web clips as Org files.

Usage:
  org-clipper-host
      Run as the native messaging host (Chrome starts it; stdin/stdout carry
      the protocol). An argument of the form chrome-extension://<id>/ also
      selects host mode.
  org-clipper-host setup --root <absolute path>
      Set the save root. It must be an existing directory; nothing is created
      or guessed.
  org-clipper-host install --extension-id <32 a-p chars> [--extension-id <id> ...]
      Copy this binary to ~/Library/Application Support/org-clipper/bin/ and
      write the Chrome host manifest com.karohani.org_clipper.json allowing
      exactly these extension IDs.
  org-clipper-host uninstall [--purge-config]
      Remove the Chrome manifest and the installed binary. Clips, logs and the
      config are kept; --purge-config also removes config.json.
  org-clipper-host status
      Show the configured root, manifest presence and allowed extension IDs.
  org-clipper-host --version
      Print the version.

Environment (for testing; leave unset in normal use):
  ORG_CLIPPER_CONFIG_DIR        config directory (default ~/Library/Application Support/org-clipper)
  ORG_CLIPPER_CHROME_HOSTS_DIR  Chrome NativeMessagingHosts directory
  ORG_CLIPPER_LOG_DIR           log directory (default ~/Library/Logs/org-clipper)
`

func main() {
	args := os.Args[1:]
	out := os.Stdout
	if isHostMode(args) {
		// stdout is the protocol stream. Point os.Stdout at stderr so that no
		// stray print anywhere in the process can corrupt the framing.
		os.Stdout = os.Stderr
	}
	os.Exit(run(args, os.Stdin, out, os.Stderr))
}

func isHostMode(args []string) bool {
	return len(args) == 0 || strings.HasPrefix(args[0], "chrome-extension://")
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if isHostMode(args) {
		return runHost(args, stdin, stdout)
	}
	switch args[0] {
	case "--version", "-version", "version":
		fmt.Fprintln(stdout, "org-clipper-host", version)
		return 0
	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	case "setup":
		return runSetup(args[1:], stdout, stderr)
	case "install":
		return runInstall(args[1:], stdout, stderr)
	case "uninstall":
		return runUninstall(args[1:], stdout, stderr)
	case "status":
		return runStatus(stdout)
	}
	fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usage)
	return 2
}

// runHost serves the protocol. Chrome passes the caller origin as the first
// argument; when the config carries an allow-list, an origin outside it is
// refused before a single message is read (Chrome's allowed_origins in the
// manifest is the primary gate; this is defence in depth).
func runHost(args []string, stdin io.Reader, stdout io.Writer) int {
	logDir, _ := config.LogDir()
	logger, closer := host.OpenLog(logDir)
	defer closer.Close()
	if len(args) > 0 {
		cfg, _ := config.Load()
		if cfg != nil && len(cfg.AllowedExtensionIDs) > 0 && !originAllowed(args[0], cfg.AllowedExtensionIDs) {
			logger.Log(host.Entry{Type: "origin", Status: "refused"})
			return 1
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	h := &host.Handler{
		LoadConfig: config.Load,
		Fetcher:    fetch.New(fetch.DefaultOptions()),
		Log:        logger,
		Version:    version,
	}
	if err := h.Serve(ctx, stdin, stdout); err != nil {
		return 1
	}
	return 0
}

func originAllowed(origin string, ids []string) bool {
	id, ok := strings.CutPrefix(origin, "chrome-extension://")
	if !ok {
		return false
	}
	id, _ = strings.CutSuffix(id, "/")
	return config.ExtensionIDRe.MatchString(id) && slices.Contains(ids, id)
}

func newFlags(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	return fs
}

func runSetup(args []string, stdout, stderr io.Writer) int {
	fs := newFlags("setup", stderr)
	root := fs.String("root", "", "absolute path of the save root")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || *root == "" {
		fmt.Fprintln(stderr, "usage: org-clipper-host setup --root <absolute path>")
		return 2
	}
	clean, err := config.ValidateRoot(*root)
	if err != nil {
		fmt.Fprintln(stderr, "setup:", err)
		return 1
	}
	c, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotConfigured) {
		fmt.Fprintln(stderr, "setup: existing config unreadable:", err)
		return 1
	}
	c.Root = clean
	if err := config.Save(c); err != nil {
		fmt.Fprintln(stderr, "setup:", err)
		return 1
	}
	p, _ := config.Path()
	fmt.Fprintf(stdout, "save root set to %s\nconfig written to %s\n", clean, p)
	return 0
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func runInstall(args []string, stdout, stderr io.Writer) int {
	fs := newFlags("install", stderr)
	var ids multiFlag
	fs.Var(&ids, "extension-id", "Chrome extension ID allowed to connect (repeatable)")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || len(ids) == 0 {
		fmt.Fprintln(stderr, "usage: org-clipper-host install --extension-id <id> [--extension-id <id> ...]")
		return 2
	}
	self, err := os.Executable()
	if err == nil {
		self, err = filepath.EvalSymlinks(self)
	}
	if err != nil {
		fmt.Fprintln(stderr, "install: cannot locate the running binary:", err)
		return 1
	}
	res, err := install.Install(self, ids)
	if err != nil {
		fmt.Fprintln(stderr, "install:", err)
		return 1
	}
	fmt.Fprintf(stdout, "installed binary: %s\nwrote manifest:   %s\nallowed origins:\n", res.Binary, res.Manifest)
	for _, id := range res.IDs {
		fmt.Fprintf(stdout, "  chrome-extension://%s/\n", id)
	}
	if c, err := config.Load(); err != nil || c.Root == "" {
		fmt.Fprintln(stdout, "note: no save root configured yet; run: org-clipper-host setup --root <absolute path>")
	}
	return 0
}

func runUninstall(args []string, stdout, stderr io.Writer) int {
	fs := newFlags("uninstall", stderr)
	purge := fs.Bool("purge-config", false, "also remove config.json")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: org-clipper-host uninstall [--purge-config]")
		return 2
	}
	removed, err := install.Uninstall(*purge)
	for _, p := range removed {
		fmt.Fprintln(stdout, "removed", p)
	}
	if len(removed) == 0 {
		fmt.Fprintln(stdout, "nothing to remove")
	}
	if err != nil {
		fmt.Fprintln(stderr, "uninstall:", err)
		return 1
	}
	fmt.Fprintln(stdout, "clips and logs were not touched")
	return 0
}

func runStatus(stdout io.Writer) int {
	c, err := config.Load()
	switch {
	case err == nil:
		fmt.Fprintln(stdout, "save root:          ", c.Root)
	case errors.Is(err, config.ErrNotConfigured):
		fmt.Fprintln(stdout, "save root:           (not configured)")
	default:
		fmt.Fprintln(stdout, "save root:           (config error:", err.Error()+")")
	}
	m, path, err := install.ReadManifest()
	switch {
	case err == nil:
		fmt.Fprintf(stdout, "chrome manifest:     present (%s)\nhost binary:         %s\n", path, m.Path)
	case path != "":
		fmt.Fprintf(stdout, "chrome manifest:     absent (%s)\n", path)
	}
	ids := "(none)"
	if c != nil && len(c.AllowedExtensionIDs) > 0 {
		ids = strings.Join(c.AllowedExtensionIDs, ", ")
	}
	fmt.Fprintln(stdout, "allowed extensions: ", ids)
	return 0
}
