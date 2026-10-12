// Package jvm installs language servers that run on a Java runtime the
// target machine already provides: jdtls (Java) and kotlin-language-server.
//
// No JVM is vendored. Each server gets a POSIX sh launcher in bin/ that finds
// Java the same way on every call, refuses to start with a one-line reason
// when Java is missing or too old, and keeps every byte the server writes
// outside the installed bundle (setup re-hashes the bundle tree on reuse, so
// a server writing into it would break the next setup).
//
// Launchers understand two private flags:
//
//	--imoogi-probe       check the installed layout only; never touches Java.
//	                     setup probes every component, and a Java-less target
//	                     must still finish setup.
//	--imoogi-check-java  run the Java checks and exit 0, or exit 69 with the
//	                     reason on stderr. Emacs calls this before starting
//	                     Eglot so a missing Java becomes a message, not a
//	                     crashed server.
package jvm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karohani/imoogi-emacs/internal/artifact"
	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang"
)

// javaUnavailableExit is EX_UNAVAILABLE from sysexits.h.
const javaUnavailableExit = 69

const defaultJavaHomeTool = "/usr/libexec/java_home"

type role int

const (
	roleUnknown role = iota
	roleJdtls
	roleKotlin
)

type server struct {
	command string // bin/<command> and lib/<command>
	minJava int
}

var servers = map[role]server{
	roleJdtls:  {command: "jdtls", minJava: 21},
	roleKotlin: {command: "kotlin-language-server", minJava: 11},
}

func classify(component config.LockComponent) role {
	switch {
	case component.Name == "jdtls" || component.Kind == "java-language-server":
		return roleJdtls
	case component.Name == "kotlin-language-server" || component.Kind == "kotlin-language-server":
		return roleKotlin
	default:
		return roleUnknown
	}
}

// SupportsComponent reports whether this provider installs component.
func SupportsComponent(component config.LockComponent) bool {
	return classify(component) != roleUnknown
}

type Provider struct {
	Component    config.LockComponent
	javaHomeTool string
}

func New(component config.LockComponent) Provider {
	return newProvider(component, defaultJavaHomeTool)
}

func newProvider(component config.LockComponent, javaHomeTool string) Provider {
	return Provider{Component: component, javaHomeTool: javaHomeTool}
}

func (p Provider) Materialize(repoRoot, stagingRoot string) ([]lang.Probe, error) {
	component := p.Component
	var script string
	var err error
	switch classify(component) {
	case roleJdtls:
		script, err = p.installJdtls(repoRoot, stagingRoot)
	case roleKotlin:
		script, err = p.installKotlin(repoRoot, stagingRoot)
	default:
		return nil, fmt.Errorf("jvm provider cannot materialize component %q kind %q", component.Name, component.Kind)
	}
	if err != nil {
		return nil, err
	}
	srv := servers[classify(component)]
	launcher, err := lang.ContainedStagePath(stagingRoot, "bin/"+srv.command)
	if err != nil {
		return nil, err
	}
	if err := lang.WriteExecutable(launcher, []byte(script)); err != nil {
		return nil, fmt.Errorf("write %s launcher: %w", srv.command, err)
	}
	probe, err := lang.AbsoluteProbe(stagingRoot, component.Probe)
	if err != nil {
		return nil, err
	}
	return []lang.Probe{probe}, nil
}

// jdtls ships as a tarball whose root is the server home (bin/, plugins/,
// config_*/). Its own bin/jdtls is a Python script; a bare macOS target has
// no python3 (the /usr/bin stub prompts for a Command Line Tools install), so
// the logic is ported to sh below instead of called.
func (p Provider) installJdtls(repoRoot, stagingRoot string) (string, error) {
	component := p.Component
	home, err := lang.ContainedStagePath(stagingRoot, "lib/jdtls")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(home), 0o755); err != nil {
		return "", err
	}
	if _, err := artifact.ExtractTarGzip(lang.ArtifactPath(repoRoot, component), home, digest(component), artifact.Limits{
		MaxEntries:     20_000,
		MaxFileBytes:   512 << 20,
		MaxOutputBytes: 1 << 30,
	}); err != nil {
		return "", fmt.Errorf("extract jdtls artifact: %w", err)
	}
	launchers, err := filepath.Glob(filepath.Join(home, "plugins", "org.eclipse.equinox.launcher_*.jar"))
	if err != nil {
		return "", err
	}
	if len(launchers) != 1 {
		return "", fmt.Errorf("jdtls archive must contain exactly one equinox launcher jar, found %d", len(launchers))
	}
	launcherRel := "lib/jdtls/plugins/" + filepath.Base(launchers[0])
	// The target is darwin/arm64 (config.TargetOS/TargetArch).
	configRel := "lib/jdtls/config_mac_arm"
	if info, err := os.Stat(filepath.Join(stagingRoot, filepath.FromSlash(configRel), "config.ini")); err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("jdtls archive is missing %s/config.ini", strings.TrimPrefix(configRel, "lib/jdtls/"))
	}

	srv := servers[roleJdtls]
	var b strings.Builder
	b.WriteString(launcherHeader())
	fmt.Fprintf(&b, "launcher_jar=\"$bundle_dir\"/%s\nshared_config=\"$bundle_dir\"/%s\n", shellQuote(launcherRel), shellQuote(configRel))
	b.WriteString(probeBlock(`[ -f "$launcher_jar" ] && [ -f "$shared_config/config.ini" ]`))
	b.WriteString(p.javaBlock(srv))
	fmt.Fprintf(&b, `# State lives in the user's cache, never in the bundle: setup re-hashes the
# bundle tree, so a server writing into it would fail the next setup.
cache_root="${XDG_CACHE_HOME:-$HOME/Library/Caches}/imoogi/jdtls"
config_dir="$cache_root"/%s/config
has_data=0
for arg in "$@"; do
	case "$arg" in
	-data) has_data=1 ;;
	esac
done
if [ "$has_data" = 0 ]; then
	workspace_key=$(printf '%%s' "$PWD" | cksum | cut -d ' ' -f 1)
	set -- -data "$cache_root/workspace/$(basename -- "$PWD")-$workspace_key" "$@"
fi
xml_opts=
if [ "$java_major" -ge 24 ]; then
	xml_opts="-Djdk.xml.maxGeneralEntitySizeLimit=0 -Djdk.xml.totalEntitySizeLimit=0"
fi
# $xml_opts is unquoted on purpose: zero or two fixed words.
exec "$java_bin" $xml_opts \
	-Declipse.application=org.eclipse.jdt.ls.core.id1 \
	-Dosgi.bundles.defaultStartLevel=4 \
	-Declipse.product=org.eclipse.jdt.ls.core.product \
	-Dosgi.checkConfiguration=true \
	"-Dosgi.sharedConfiguration.area=$shared_config" \
	-Dosgi.sharedConfiguration.area.readOnly=true \
	-Dosgi.configuration.cascaded=true \
	-Xms1G \
	--add-modules=ALL-SYSTEM \
	--add-opens java.base/java.util=ALL-UNNAMED \
	--add-opens java.base/java.lang=ALL-UNNAMED \
	-jar "$launcher_jar" \
	-configuration "$config_dir" \
	"$@"
`, shellQuote(component.UpstreamVersion))
	return b.String(), nil
}

// kotlin-language-server ships as server.zip holding server/bin (a Gradle
// start script) and server/lib. The start script owns the classpath order, so
// the launcher hands it the resolved Java through PATH rather than rebuilding
// the java command line.
func (p Provider) installKotlin(repoRoot, stagingRoot string) (string, error) {
	component := p.Component
	extract, err := lang.ContainedStagePath(stagingRoot, "lib/.extract-kotlin-language-server")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(extract), 0o755); err != nil {
		return "", err
	}
	if _, err := artifact.ExtractZip(lang.ArtifactPath(repoRoot, component), extract, digest(component), artifact.Limits{
		MaxEntries:     20_000,
		MaxFileBytes:   512 << 20,
		MaxOutputBytes: 1 << 30,
	}); err != nil {
		return "", fmt.Errorf("extract kotlin-language-server artifact: %w", err)
	}
	defer os.RemoveAll(extract)
	home, err := lang.ContainedStagePath(stagingRoot, "lib/kotlin-language-server")
	if err != nil {
		return "", err
	}
	if err := os.Rename(filepath.Join(extract, "server"), home); err != nil {
		return "", fmt.Errorf("install kotlin-language-server: %w", err)
	}
	upstream := filepath.Join(home, "bin", "kotlin-language-server")
	if err := lang.EnsureExecutable(upstream); err != nil {
		return "", fmt.Errorf("kotlin-language-server start script: %w", err)
	}

	srv := servers[roleKotlin]
	var b strings.Builder
	b.WriteString(launcherHeader())
	b.WriteString("server_script=\"$bundle_dir\"/lib/kotlin-language-server/bin/kotlin-language-server\n")
	b.WriteString(probeBlock(`[ -x "$server_script" ] && [ -d "$bundle_dir/lib/kotlin-language-server/lib" ]`))
	b.WriteString(p.javaBlock(srv))
	b.WriteString(`# The Gradle script prefers JAVA_HOME and dies when it is invalid; clear it
# and put the Java this launcher already checked first on PATH instead.
unset JAVA_HOME
PATH="$(dirname -- "$java_bin"):$PATH"
export PATH
exec "$server_script" "$@"
`)
	return b.String(), nil
}

func launcherHeader() string {
	// bin/ is reached through the .local/bin symlink; cd -P resolves to the
	// bundle so lib/ is found (same shape as the node-based launchers).
	return `#!/bin/sh
set -eu
bin_dir=$(CDPATH= cd -P -- "$(dirname -- "$0")" && pwd)
bundle_dir=$(CDPATH= cd -P -- "$bin_dir/.." && pwd)
`
}

func probeBlock(condition string) string {
	return fmt.Sprintf(`if [ "${1:-}" = --imoogi-probe ]; then
	if %s; then
		exit 0
	fi
	echo "$0: installed server files are missing" >&2
	exit 1
fi
`, condition)
}

// javaBlock resolves Java in a fixed order: JAVA_HOME, then the macOS
// java_home registry, then a java on PATH. /usr/bin/java on macOS is only a
// stub that opens an "install Java" dialog, so it never counts as a runtime.
func (p Provider) javaBlock(srv server) string {
	need := fmt.Sprintf("%s needs Java %d or newer", srv.command, srv.minJava)
	return fmt.Sprintf(`imoogi_java_fail() {
	printf '%%s\n' "$1" >&2
	exit %[1]d
}
java_bin=
if [ -n "${JAVA_HOME:-}" ] && [ -x "$JAVA_HOME/bin/java" ]; then
	java_bin=$JAVA_HOME/bin/java
elif [ -x %[2]s ] && java_home_dir=$(%[2]s 2>/dev/null) && [ -x "$java_home_dir/bin/java" ]; then
	java_bin=$java_home_dir/bin/java
elif java_on_path=$(command -v java 2>/dev/null); then
	if [ "$java_on_path" != /usr/bin/java ] || [ "$(uname -s)" != Darwin ]; then
		java_bin=$java_on_path
	fi
fi
if [ -z "$java_bin" ]; then
	imoogi_java_fail "%[3]s, but no Java runtime was found. Install a JDK or set JAVA_HOME."
fi
java_version=$("$java_bin" -version 2>&1 | sed -n 's/.*version "\([0-9][0-9._]*\).*/\1/p' | head -n 1)
java_major=${java_version%%%%.*}
if [ "$java_major" = 1 ]; then
	java_major=$(printf '%%s' "$java_version" | cut -d . -f 2)
fi
case "$java_major" in
'' | *[!0-9]*) imoogi_java_fail "%[3]s, but the version of $java_bin could not be read." ;;
esac
if [ "$java_major" -lt %[4]d ]; then
	imoogi_java_fail "%[3]s, but $java_bin is Java $java_major."
fi
if [ "${1:-}" = --imoogi-check-java ]; then
	exit 0
fi
`, javaUnavailableExit, shellQuote(p.javaHomeTool), need, srv.minJava)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func digest(component config.LockComponent) artifact.Digest {
	return artifact.Digest{Size: component.Artifact.Size, SHA256: component.Artifact.SHA256}
}
