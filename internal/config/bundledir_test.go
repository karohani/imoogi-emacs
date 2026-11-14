package config

import "testing"

func TestIsBundleDirMatchesTheIdAndItsReplacementsOnly(t *testing.T) {
	const bundle = "2026.11.14.1"
	for name, want := range map[string]bool{
		"2026.11.14.1":              true,
		"2026.11.14.1-0123456789ab": true,
		"2026.11.14.10":             false, // another bundle, not a replacement
		"2026.11.14.1-0123456789aB": false,
		"2026.11.14.1-0123456789":   false,
		"2026.11.14.1-":             false,
		".retired-2026.11.14.1-1":   false,
		"2026.11.14.2":              false,
	} {
		if got := IsBundleDir(name, bundle); got != want {
			t.Errorf("IsBundleDir(%q) = %v, want %v", name, got, want)
		}
	}
	if !IsBundleDir(ReplacementBundleDir(bundle, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"), bundle) {
		t.Fatal("a replacement directory is not recognised as the bundle's")
	}
}

func TestActiveBundleDirReadsTheActivationTarget(t *testing.T) {
	const bundle = "2026.11.14.1"
	for target, want := range map[string]string{
		"toolchains/2026.11.14.1/bin":              "2026.11.14.1",
		"toolchains/2026.11.14.1-0123456789ab/bin": "2026.11.14.1-0123456789ab",
		"toolchains/2026.11.14.10/bin":             "",
		"toolchains/2026.11.14.1/lib":              "",
		"/abs/toolchains/2026.11.14.1/bin":         "",
		"../toolchains/2026.11.14.1/bin":           "",
	} {
		got, ok := ActiveBundleDir(target, bundle)
		if got != want || ok != (want != "") {
			t.Errorf("ActiveBundleDir(%q) = %q, %v; want %q", target, got, ok, want)
		}
	}
}
