package config

import "strings"

// replacementSuffixLen is how many hex digits of the installed tree's SHA-256
// name a replacement bundle directory.
const replacementSuffixLen = 12

// ReplacementBundleDir names the directory under .local/toolchains that setup
// publishes a completed bundle into when the bundle's own directory is still
// active: "<bundle>-<first 12 hex of the tree SHA-256>". Bundle ids are CalVer
// and contain no "-", so the name never parses as another bundle id.
func ReplacementBundleDir(bundle, treeSHA256 string) string {
	return bundle + "-" + treeSHA256[:replacementSuffixLen]
}

// IsBundleDir reports whether name is a directory of bundle under
// .local/toolchains: the bundle id itself or one of its replacement directories.
func IsBundleDir(name, bundle string) bool {
	if name == bundle {
		return true
	}
	suffix, ok := strings.CutPrefix(name, bundle+"-")
	if !ok || len(suffix) != replacementSuffixLen {
		return false
	}
	for _, r := range suffix {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// ActiveBundleDir returns the bundle directory a relative .local/bin target
// ("toolchains/<dir>/bin") activates, when that directory belongs to bundle.
func ActiveBundleDir(linkTarget, bundle string) (string, bool) {
	parts := strings.Split(linkTarget, "/")
	if len(parts) != 3 || parts[0] != "toolchains" || parts[2] != "bin" || !IsBundleDir(parts[1], bundle) {
		return "", false
	}
	return parts[1], true
}
