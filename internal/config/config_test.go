package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFixtureManifestAndLock(t *testing.T) {
	manifest := loadFixtureManifest(t)
	lock := loadFixtureLock(t)

	if err := manifest.ValidateLock(*lock); err != nil {
		t.Fatalf("ValidateLock failed: %v", err)
	}
	plan, err := BuildPlan(*manifest, *lock)
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	if plan.CLI != "1.2.3" {
		t.Fatalf("plan CLI = %q", plan.CLI)
	}
	if plan.Bundle != "2026.08.15.1" {
		t.Fatalf("plan Bundle = %q", plan.Bundle)
	}
	if got, want := len(plan.Components), 4; got != want {
		t.Fatalf("plan components = %d, want %d", got, want)
	}
	if got := plan.Components[0].Name; got != "gopls" {
		t.Fatalf("components not sorted by name; first = %q", got)
	}
}

func TestStrictParsingRejectsUnknownAndDuplicateFields(t *testing.T) {
	t.Run("unknown", func(t *testing.T) {
		path := writeTemp(t, `{
		  "schema": "imoogi-toolchains-desired/v1",
		  "cli_version": "1.2.3",
		  "bundle": "2026.08.15.1",
		  "target": {"os": "darwin", "arch": "arm64"},
		  "components": [{"name":"gopls","kind":"go-language-server","source":"golang.org/x/tools/gopls","upstream_version":"v0.20.0"}],
		  "extra": true
		}`)
		if _, err := LoadDesired(path); err == nil {
			t.Fatal("LoadDesired succeeded, want unknown field error")
		}
	})
	t.Run("duplicate", func(t *testing.T) {
		if _, err := LoadDesired(filepath.Join("testdata", "duplicate-key.json")); err == nil {
			t.Fatal("LoadDesired succeeded, want duplicate key error")
		}
	})
}

func TestManifestValidationRejectsInvalidVersionsPlatformAndDuplicates(t *testing.T) {
	tests := map[string]func(DesiredManifest) DesiredManifest{
		"cli semver": func(m DesiredManifest) DesiredManifest {
			m.CLI = "v.260101.1"
			return m
		},
		"bundle calver": func(m DesiredManifest) DesiredManifest {
			m.Bundle = "1.2.3"
			return m
		},
		"platform": func(m DesiredManifest) DesiredManifest {
			m.Target.Arch = "amd64"
			return m
		},
		"duplicate": func(m DesiredManifest) DesiredManifest {
			m.Components = append(m.Components, m.Components[0])
			return m
		},
		"runtime policy node version": func(m DesiredManifest) DesiredManifest {
			m.RuntimePolicy.NodeVersion = "22.14.0.1"
			return m
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			manifest := mutate(*loadFixtureManifest(t))
			if err := manifest.Validate(); err == nil {
				t.Fatal("Validate succeeded, want error")
			}
		})
	}
}

func TestComponentVersionValidationRejectsPathLikeAndWrongFormats(t *testing.T) {
	tests := []struct {
		name      string
		component string
		version   string
	}{
		{name: "node traversal", component: "node", version: "../../outside"},
		{name: "node missing prefix", component: "node", version: "24.19.0"},
		{name: "gopls absolute fragment", component: "gopls", version: "/tmp/gopls"},
		{name: "gopls invalid semver", component: "gopls", version: "v0.23"},
		{name: "typescript prefixed semver", component: "typescript", version: "v6.0.3"},
		{name: "typescript language server scheme", component: "typescript-language-server", version: "https:6.0.0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manifest := *loadFixtureManifest(t)
			for i := range manifest.Components {
				if manifest.Components[i].Name == tc.component {
					manifest.Components[i].UpstreamVersion = tc.version
				}
			}
			if err := manifest.Validate(); err == nil {
				t.Fatalf("Validate accepted %s version %q", tc.component, tc.version)
			}
		})
	}
}

func TestValidateRepoPath(t *testing.T) {
	for _, path := range []string{"../outside", "vendor/../outside", "/tmp/outside", `vendor\outside`, "https:outside", "vendor//outside"} {
		t.Run(path, func(t *testing.T) {
			if err := ValidateRepoPath(path); err == nil {
				t.Fatalf("ValidateRepoPath(%q) succeeded", path)
			}
		})
	}
	if err := ValidateRepoPath("vendor/toolchains/node/v24.19.0/node.tgz"); err != nil {
		t.Fatalf("ValidateRepoPath rejected clean path: %v", err)
	}
}

func TestSemVerRejectsInvalidPrereleaseIdentifiers(t *testing.T) {
	tests := []string{
		"1.2.3-..",
		"1.2.3-alpha..1",
		"1.2.3-01",
		"1.2.3-alpha.01",
		"1.2.3+",
		"1.2.3+build..1",
	}
	for _, version := range tests {
		t.Run(version, func(t *testing.T) {
			manifest := *loadFixtureManifest(t)
			manifest.CLI = version
			if err := manifest.Validate(); err == nil {
				t.Fatal("Validate succeeded, want SemVer error")
			}
		})
	}

	manifest := *loadFixtureManifest(t)
	manifest.CLI = "1.2.3-alpha.1+build.01"
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate rejected valid SemVer: %v", err)
	}
}

func TestLockValidationRejectsInvalidArtifactAndRuntime(t *testing.T) {
	tests := map[string]func(ResolvedLock) ResolvedLock{
		"absolute path": func(l ResolvedLock) ResolvedLock {
			l.Components[0].Artifact.Path = "/tmp/gopls.tar.gz"
			return l
		},
		"traversal path": func(l ResolvedLock) ResolvedLock {
			l.Components[0].License.Path = "../LICENSE"
			return l
		},
		"dirty path": func(l ResolvedLock) ResolvedLock {
			l.Components[0].Install[0].Path = "bin/./gopls"
			return l
		},
		"bad sha": func(l ResolvedLock) ResolvedLock {
			l.Components[0].Artifact.SHA256 = strings.Repeat("A", 64)
			return l
		},
		"bad size": func(l ResolvedLock) ResolvedLock {
			l.Components[0].Artifact.Size = 0
			return l
		},
		"bad retrieved_at": func(l ResolvedLock) ResolvedLock {
			l.Components[0].Artifact.RetrievedAt = "2026-08-15"
			return l
		},
		"path-like upstream version": func(l ResolvedLock) ResolvedLock {
			l.Components[0].UpstreamVersion = "v0.23.0/../../outside"
			return l
		},
		"duplicate component": func(l ResolvedLock) ResolvedLock {
			l.Components = append(l.Components, l.Components[0])
			return l
		},
		"node engine below range": func(l ResolvedLock) ResolvedLock {
			for i := range l.Components {
				if l.Components[i].Name == "node" {
					l.Components[i].UpstreamVersion = "v18.19.0"
				}
			}
			return l
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			lock := mutate(*loadFixtureLock(t))
			if err := lock.Validate(); err == nil {
				t.Fatal("Validate succeeded, want error")
			}
		})
	}
}

func TestManifestLockConsistency(t *testing.T) {
	tests := map[string]func(DesiredManifest, ResolvedLock) (DesiredManifest, ResolvedLock){
		"bundle mismatch": func(m DesiredManifest, l ResolvedLock) (DesiredManifest, ResolvedLock) {
			l.Bundle = "2026.08.16.1"
			return m, l
		},
		"missing component": func(m DesiredManifest, l ResolvedLock) (DesiredManifest, ResolvedLock) {
			l.Components = l.Components[:len(l.Components)-1]
			return m, l
		},
		"extra component": func(m DesiredManifest, l ResolvedLock) (DesiredManifest, ResolvedLock) {
			m.Components = m.Components[:len(m.Components)-1]
			return m, l
		},
		"identity mismatch": func(m DesiredManifest, l ResolvedLock) (DesiredManifest, ResolvedLock) {
			l.Components[0].UpstreamVersion = "v0.21.0"
			return m, l
		},
		"runtime mismatch": func(m DesiredManifest, l ResolvedLock) (DesiredManifest, ResolvedLock) {
			m.Components[len(m.Components)-1].Runtime.NodeEngine = ">=22.0.0 <23.0.0"
			return m, l
		},
		"runtime policy node version mismatch": func(m DesiredManifest, l ResolvedLock) (DesiredManifest, ResolvedLock) {
			m.RuntimePolicy.NodeVersion = "20.0.0"
			return m, l
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			manifest, lock := mutate(*loadFixtureManifest(t), *loadFixtureLock(t))
			if err := manifest.ValidateLock(lock); err == nil {
				t.Fatal("ValidateLock succeeded, want error")
			}
		})
	}
}

func TestBuildPlanValidatesDirectStructInputs(t *testing.T) {
	manifest := *loadFixtureManifest(t)
	lock := *loadFixtureLock(t)
	lock.Components[0].Artifact.SHA256 = "not-a-sha"

	if _, err := BuildPlan(manifest, lock); err == nil {
		t.Fatal("BuildPlan succeeded with invalid lock, want error")
	}
}

func TestWritersValidateDirectStructInputs(t *testing.T) {
	t.Run("desired", func(t *testing.T) {
		manifest := *loadFixtureManifest(t)
		manifest.CLI = "1.2.3-01"
		var out bytes.Buffer
		if err := WriteDesired(&out, manifest); err == nil {
			t.Fatal("WriteDesired succeeded with invalid manifest, want error")
		}
		if out.Len() != 0 {
			t.Fatalf("WriteDesired wrote %q for invalid input", out.String())
		}
	})
	t.Run("lock", func(t *testing.T) {
		lock := *loadFixtureLock(t)
		lock.Components[0].Artifact.RetrievedAt = "not-rfc3339"
		var out bytes.Buffer
		if err := WriteLock(&out, lock); err == nil {
			t.Fatal("WriteLock succeeded with invalid lock, want error")
		}
		if out.Len() != 0 {
			t.Fatalf("WriteLock wrote %q for invalid input", out.String())
		}
	})
}

func TestSameIdentityDifferentDigestReportsDrift(t *testing.T) {
	previous := *loadFixtureLock(t)
	next := *loadFixtureLock(t)
	next.Components[0].Artifact.SHA256 = strings.Repeat("e", 64)

	if err := DetectDrift(previous, next); err == nil {
		t.Fatal("DetectDrift succeeded, want drift error")
	}
}

func TestNodeEngineConstraint(t *testing.T) {
	ok, err := CheckNodeEngine("22.14.0", ">=20.0.0 <23.0.0")
	if err != nil {
		t.Fatalf("CheckNodeEngine returned error: %v", err)
	}
	if !ok {
		t.Fatal("22.14.0 should satisfy range")
	}
	ok, err = CheckNodeEngine("18.19.0", ">=20.0.0 <23.0.0")
	if err != nil {
		t.Fatalf("CheckNodeEngine returned error: %v", err)
	}
	if ok {
		t.Fatal("18.19.0 should not satisfy range")
	}
}

func TestDeterministicWriters(t *testing.T) {
	lock := *loadFixtureLock(t)
	var first, second bytes.Buffer
	if err := WriteLock(&first, lock); err != nil {
		t.Fatalf("WriteLock failed: %v", err)
	}
	lock.Components[0], lock.Components[3] = lock.Components[3], lock.Components[0]
	if err := WriteLock(&second, lock); err != nil {
		t.Fatalf("WriteLock failed: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("WriteLock output differs between runs")
	}
	if !bytes.HasSuffix(first.Bytes(), []byte("\n")) {
		t.Fatal("WriteLock output lacks trailing newline")
	}
}

func loadFixtureManifest(t *testing.T) *DesiredManifest {
	t.Helper()
	manifest, err := LoadDesired(filepath.Join("testdata", "toolchains.json"))
	if err != nil {
		t.Fatalf("LoadDesired fixture failed: %v", err)
	}
	return manifest
}

func loadFixtureLock(t *testing.T) *ResolvedLock {
	t.Helper()
	lock, err := LoadLock(filepath.Join("testdata", "toolchains.lock.json"))
	if err != nil {
		t.Fatalf("LoadLock fixture failed: %v", err)
	}
	return lock
}

func writeTemp(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
