package config

import (
	"fmt"
	"strings"
)

const (
	// KindACPAdapter is an Agent Client Protocol adapter: an npm package whose
	// full dependency closure is installed from a committed package-lock.json
	// and packed into one deterministic tar.gz.
	KindACPAdapter = "acp-adapter"

	// ArtifactPolicyLocalBuild marks an artifact that is never committed. Setup
	// rebuilds it from the committed inputs and verifies it against the lock
	// digest; a mismatch fails, the lock is never rewritten to match.
	ArtifactPolicyLocalBuild = "local-build"

	npmSourcePrefix = "npm:"
)

// NPMPackageName returns the npm package an "npm:<name>" source names, or "".
func NPMPackageName(source string) string {
	name, ok := strings.CutPrefix(source, npmSourcePrefix)
	if !ok {
		return ""
	}
	return name
}

// ACPPackageDir is the repository directory holding an adapter's committed
// package.json and package-lock.json.
func ACPPackageDir(name string) string {
	return "toolchains/acp/" + name
}

func validateArtifactPolicy(name, kind, source, version, revision, policy string) error {
	switch policy {
	case "":
	case ArtifactPolicyLocalBuild:
		if kind != KindACPAdapter {
			return fmt.Errorf("component %q artifact_policy %q needs a builder; only %s has one", name, policy, KindACPAdapter)
		}
	default:
		return fmt.Errorf("component %q artifact_policy %q is unsupported", name, policy)
	}
	if kind != KindACPAdapter {
		return nil
	}
	pkg := NPMPackageName(source)
	if !validNPMName(pkg) {
		return fmt.Errorf("component %q source %q must be npm:<package>", name, source)
	}
	if !validSemVer(version) {
		return fmt.Errorf("component %q upstream_version %q must be SemVer", name, version)
	}
	if !shaRE.MatchString(revision) {
		return fmt.Errorf("component %q revision must be the SHA-256 of its package-lock.json", name)
	}
	return nil
}

func validNPMName(name string) bool {
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `\: `) {
		return false
	}
	parts := strings.Split(name, "/")
	if strings.HasPrefix(name, "@") {
		return len(parts) == 2 && len(parts[0]) > 1 && parts[1] != ""
	}
	return len(parts) == 1
}
