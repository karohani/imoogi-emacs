package jvm

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/config"
)

const launcherJar = "plugins/org.eclipse.equinox.launcher_1.8.0.v20260804-1928.jar"

func TestSupportsComponentByNameOrKind(t *testing.T) {
	for _, c := range []config.LockComponent{
		{Name: "jdtls"},
		{Name: "x", Kind: "java-language-server"},
		{Name: "kotlin-language-server"},
		{Name: "y", Kind: "kotlin-language-server"},
	} {
		if !SupportsComponent(c) {
			t.Fatalf("SupportsComponent(%+v) = false", c)
		}
	}
	for _, c := range []config.LockComponent{{Name: "gopls", Kind: "go-language-server"}, {Name: "clojure-lsp", Kind: "clojure-language-server"}} {
		if SupportsComponent(c) {
			t.Fatalf("SupportsComponent(%+v) = true", c)
		}
	}
}

func TestJdtlsProviderInstallsLauncherAndJavaFreeProbe(t *testing.T) {
	repo, staging, component := jdtlsFixture(t)

	probes, err := newProvider(component, missingJavaHomeTool(t)).Materialize(repo, staging)
	if err != nil {
		t.Fatalf("Materialize failed: %v", err)
	}
	wantProbe := filepath.Join(staging, "bin", "jdtls")
	if len(probes) != 1 || probes[0].Command != wantProbe || strings.Join(probes[0].Args, " ") != "--imoogi-probe" {
		t.Fatalf("probes = %+v", probes)
	}
	assertMode(t, wantProbe, 0o755)
	if _, err := os.Stat(filepath.Join(staging, "lib", "jdtls", filepath.FromSlash(launcherJar))); err != nil {
		t.Fatalf("launcher jar not installed: %v", err)
	}
	// The probe must pass on a target without Java: setup probes every
	// component, and a Java-less machine must still finish setup.
	stdout, stderr, code := runLauncher(t, wantProbe, toolPath(t), nil, "--imoogi-probe")
	if code != 0 {
		t.Fatalf("probe exit = %d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestJdtlsLauncherReportsMissingJavaClearly(t *testing.T) {
	repo, staging, component := jdtlsFixture(t)
	if _, err := newProvider(component, missingJavaHomeTool(t)).Materialize(repo, staging); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(staging, "bin", "jdtls")
	for _, args := range [][]string{{"--imoogi-check-java"}, nil} {
		_, stderr, code := runLauncher(t, launcher, toolPath(t), nil, args...)
		if code != javaUnavailableExit {
			t.Fatalf("args %v exit = %d, want %d (stderr %q)", args, code, javaUnavailableExit, stderr)
		}
		if !strings.Contains(stderr, "jdtls needs Java 21 or newer, but no Java runtime was found") {
			t.Fatalf("stderr = %q", stderr)
		}
	}
}

func TestJdtlsLauncherRejectsTooOldJava(t *testing.T) {
	repo, staging, component := jdtlsFixture(t)
	if _, err := newProvider(component, missingJavaHomeTool(t)).Materialize(repo, staging); err != nil {
		t.Fatal(err)
	}
	for version, major := range map[string]string{`17.0.2" 2022-01-18`: "17", `1.8.0_292"`: "8"} {
		javaHome := fakeJavaHome(t, version, "")
		_, stderr, code := runLauncher(t, filepath.Join(staging, "bin", "jdtls"), toolPath(t), []string{"JAVA_HOME=" + javaHome}, "--imoogi-check-java")
		if code != javaUnavailableExit || !strings.Contains(stderr, "jdtls needs Java 21 or newer, but "+javaHome+"/bin/java is Java "+major) {
			t.Fatalf("version %s: exit = %d stderr = %q", version, code, stderr)
		}
	}
}

func TestJdtlsLauncherKeepsServerStateOutsideBundle(t *testing.T) {
	repo, staging, component := jdtlsFixture(t)
	if _, err := newProvider(component, missingJavaHomeTool(t)).Materialize(repo, staging); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(staging, "bin", "jdtls")
	argsFile := filepath.Join(t.TempDir(), "args")
	javaHome := fakeJavaHome(t, `21.0.9" 2025-10-21 LTS`, argsFile)
	home := t.TempDir()
	env := []string{"JAVA_HOME=" + javaHome, "HOME=" + home}

	if _, stderr, code := runLauncher(t, launcher, toolPath(t), env, "--imoogi-check-java"); code != 0 {
		t.Fatalf("check-java exit = %d stderr=%q", code, stderr)
	}
	if _, stderr, code := runLauncher(t, launcher, toolPath(t), env); code != 0 {
		t.Fatalf("launch exit = %d stderr=%q", code, stderr)
	}
	args := readLines(t, argsFile)
	bundleRoot, _ := filepath.EvalSymlinks(staging)
	cache := filepath.Join(home, "Library", "Caches", "imoogi", "jdtls")
	assertArgPair(t, args, "-jar", filepath.Join(bundleRoot, "lib", "jdtls", filepath.FromSlash(launcherJar)))
	assertArgPair(t, args, "-configuration", filepath.Join(cache, "1.61.0-202609031315", "config"))
	data := argAfter(t, args, "-data")
	if !strings.HasPrefix(data, filepath.Join(cache, "workspace")+"/") {
		t.Fatalf("-data = %q, want under %s", data, cache)
	}
	if !containsArg(args, "-Dosgi.sharedConfiguration.area="+filepath.Join(bundleRoot, "lib", "jdtls", "config_mac_arm")) {
		t.Fatalf("shared configuration not read-only bundle config: %v", args)
	}
	if !containsArg(args, "-Dosgi.sharedConfiguration.area.readOnly=true") {
		t.Fatalf("shared configuration is not read-only: %v", args)
	}

	// A caller that chooses its own workspace keeps it.
	if _, stderr, code := runLauncher(t, launcher, toolPath(t), env, "-data", "/tmp/custom"); code != 0 {
		t.Fatalf("launch with -data exit = %d stderr=%q", code, stderr)
	}
	args = readLines(t, argsFile)
	if got := countArg(args, "-data"); got != 1 {
		t.Fatalf("-data appears %d times: %v", got, args)
	}
	assertArgPair(t, args, "-data", "/tmp/custom")
}

func TestJdtlsProviderRejectsArchiveWithoutEquinoxLauncher(t *testing.T) {
	repo := t.TempDir()
	body := tarGz(t, map[string]string{"config_mac_arm/config.ini": "x"})
	component := lockComponent(t, repo, "jdtls", "java-language-server", "1.61.0-202609031315", "vendor/toolchains/java/jdtls/x/jdtls.tar.gz", body)
	if _, err := newProvider(component, missingJavaHomeTool(t)).Materialize(repo, filepath.Join(repo, ".staging")); err == nil || !strings.Contains(err.Error(), "equinox launcher") {
		t.Fatalf("Materialize error = %v", err)
	}
}

func TestKotlinProviderInstallsLauncherOverUpstreamScript(t *testing.T) {
	repo := t.TempDir()
	upstream := "#!/bin/sh\nprintf 'java=%s JAVA_HOME=%s args=%s\\n' \"$(command -v java)\" \"${JAVA_HOME:-}\" \"$*\"\n"
	body := zipBytes(t, map[string]string{
		"server/bin/kotlin-language-server": upstream,
		"server/lib/server-1.3.13.jar":      "jar",
		"server/licenseReport.html":         "report",
	})
	component := lockComponent(t, repo, "kotlin-language-server", "kotlin-language-server", "1.3.13", "vendor/toolchains/kotlin/kotlin-language-server/1.3.13/darwin-arm64/server.zip", body)
	staging := filepath.Join(repo, ".local", ".staging")

	probes, err := newProvider(component, missingJavaHomeTool(t)).Materialize(repo, staging)
	if err != nil {
		t.Fatalf("Materialize failed: %v", err)
	}
	launcher := filepath.Join(staging, "bin", "kotlin-language-server")
	if len(probes) != 1 || probes[0].Command != launcher || strings.Join(probes[0].Args, " ") != "--imoogi-probe" {
		t.Fatalf("probes = %+v", probes)
	}
	assertMode(t, launcher, 0o755)
	assertMode(t, filepath.Join(staging, "lib", "kotlin-language-server", "bin", "kotlin-language-server"), 0o755)
	if _, err := os.Lstat(filepath.Join(staging, "lib", ".extract-kotlin-language-server")); !os.IsNotExist(err) {
		t.Fatalf("extraction temp left behind: %v", err)
	}
	if _, stderr, code := runLauncher(t, launcher, toolPath(t), nil, "--imoogi-probe"); code != 0 {
		t.Fatalf("probe exit = %d stderr=%q", code, stderr)
	}

	_, stderr, code := runLauncher(t, launcher, toolPath(t), nil, "--imoogi-check-java")
	if code != javaUnavailableExit || !strings.Contains(stderr, "kotlin-language-server needs Java 11 or newer, but no Java runtime was found") {
		t.Fatalf("missing java: exit = %d stderr = %q", code, stderr)
	}

	javaHome := fakeJavaHome(t, `17.0.2" 2022-01-18`, "")
	stdout, stderr, code := runLauncher(t, launcher, toolPath(t), []string{"JAVA_HOME=" + javaHome}, "--stdio")
	if code != 0 {
		t.Fatalf("launch exit = %d stderr=%q", code, stderr)
	}
	// The upstream Gradle script must see the resolved Java on PATH and no
	// JAVA_HOME, so it cannot fall back to a different runtime.
	if want := "java=" + javaHome + "/bin/java JAVA_HOME= args=--stdio"; strings.TrimSpace(stdout) != want {
		t.Fatalf("upstream script saw %q, want %q", stdout, want)
	}
}

func TestProviderIntegrityFailureDoesNotMaterialize(t *testing.T) {
	repo, staging, component := jdtlsFixture(t)
	component.Artifact.SHA256 = strings.Repeat("0", 64)
	if _, err := newProvider(component, missingJavaHomeTool(t)).Materialize(repo, staging); err == nil {
		t.Fatal("Materialize accepted a corrupted artifact")
	}
	if _, err := os.Lstat(filepath.Join(staging, "bin", "jdtls")); !os.IsNotExist(err) {
		t.Fatalf("launcher written after integrity failure: %v", err)
	}
}

func TestProviderRejectsForeignComponent(t *testing.T) {
	component := config.LockComponent{Name: "gopls", Kind: "go-language-server"}
	if _, err := New(component).Materialize(t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("Materialize accepted gopls")
	}
}

// --- fixtures ---

func jdtlsFixture(t *testing.T) (string, string, config.LockComponent) {
	t.Helper()
	repo := t.TempDir()
	body := tarGz(t, map[string]string{
		launcherJar:                 "jar",
		"config_mac_arm/config.ini": "osgi.bundles=x\n",
		"bin/jdtls":                 "#!/usr/bin/env python3\n",
	})
	component := lockComponent(t, repo, "jdtls", "java-language-server", "1.61.0-202609031315",
		"vendor/toolchains/java/jdtls/1.61.0-202609031315/darwin-arm64/jdt-language-server.tar.gz", body)
	return repo, filepath.Join(repo, ".local", ".staging"), component
}

func lockComponent(t *testing.T, repo, name, kind, version, rel string, body []byte) config.LockComponent {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	return config.LockComponent{
		Name:            name,
		Kind:            kind,
		UpstreamVersion: version,
		Artifact:        config.Artifact{Path: rel, Size: int64(len(body)), SHA256: hex.EncodeToString(sum[:])},
		Probe:           config.Probe{Command: "bin/" + name, Args: []string{"--imoogi-probe"}},
	}
}

func missingJavaHomeTool(t *testing.T) string {
	return filepath.Join(t.TempDir(), "no-java-home-tool")
}

// toolPath returns a PATH holding only the utilities the launchers use, so
// tests control whether any `java` is reachable.
func toolPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, tool := range []string{"dirname", "basename", "sed", "head", "cut", "uname", "cksum"} {
		found, err := exec.LookPath(tool)
		if err != nil {
			t.Fatalf("test needs %s: %v", tool, err)
		}
		if err := os.Symlink(found, filepath.Join(dir, tool)); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// fakeJavaHome writes JAVA_HOME/bin/java that reports versionTail on
// -version and records its arguments to argsFile otherwise.
func fakeJavaHome(t *testing.T, versionTail, argsFile string) string {
	t.Helper()
	home := t.TempDir()
	script := "#!/bin/sh\nif [ \"${1:-}\" = -version ]; then echo 'openjdk version \"" + versionTail + "' >&2; exit 0; fi\n"
	if argsFile != "" {
		script += "for a in \"$@\"; do printf '%s\\n' \"$a\"; done > '" + argsFile + "'\n"
	}
	path := filepath.Join(home, "bin", "java")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return home
}

func runLauncher(t *testing.T, launcher, path string, env []string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(launcher, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = append([]string{"PATH=" + path, "HOME=" + t.TempDir()}, env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}
	if err != nil {
		t.Fatalf("run %s: %v", launcher, err)
	}
	return stdout.String(), stderr.String(), 0
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n")
}

func argAfter(t *testing.T, args []string, flag string) string {
	t.Helper()
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	t.Fatalf("%s missing from %v", flag, args)
	return ""
}

func assertArgPair(t *testing.T, args []string, flag, want string) {
	t.Helper()
	if got := argAfter(t, args, flag); got != want {
		t.Fatalf("%s = %q, want %q", flag, got, want)
	}
}

func containsArg(args []string, want string) bool {
	return countArg(args, want) > 0
}

func countArg(args []string, want string) int {
	n := 0
	for _, arg := range args {
		if arg == want {
			n++
		}
	}
	return n
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %o, want %o", path, got, want)
	}
}

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
