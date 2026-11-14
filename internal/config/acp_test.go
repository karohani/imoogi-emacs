package config

import (
	"strings"
	"testing"
)

const lockfileDigest = "1111111111111111111111111111111111111111111111111111111111111111"

func acpDesired() DesiredComponent {
	return DesiredComponent{
		Name:            "claude-agent-acp",
		Kind:            KindACPAdapter,
		Source:          "npm:@agentclientprotocol/claude-agent-acp",
		UpstreamVersion: "0.99.1",
		Revision:        lockfileDigest,
		ArtifactPolicy:  ArtifactPolicyLocalBuild,
	}
}

func TestACPAdapterDesiredComponentValidates(t *testing.T) {
	if err := acpDesired().validate(); err != nil {
		t.Fatalf("validate() = %v", err)
	}
	committed := acpDesired()
	committed.ArtifactPolicy = ""
	if err := committed.validate(); err != nil {
		t.Fatalf("committed adapter validate() = %v", err)
	}
}

func TestACPAdapterDesiredComponentRejectsBadFields(t *testing.T) {
	tests := map[string]struct {
		edit func(*DesiredComponent)
		want string
	}{
		"unknown policy":         {func(c *DesiredComponent) { c.ArtifactPolicy = "download-later" }, "artifact_policy"},
		"local-build other kind": {func(c *DesiredComponent) { c.Kind = "typescript-sdk" }, "artifact_policy"},
		"revision not digest":    {func(c *DesiredComponent) { c.Revision = "abc" }, "package-lock.json"},
		"revision missing":       {func(c *DesiredComponent) { c.Revision = "" }, "package-lock.json"},
		"source not npm":         {func(c *DesiredComponent) { c.Source = "github.com/x/y" }, "npm:"},
		"version not semver":     {func(c *DesiredComponent) { c.UpstreamVersion = "latest" }, "SemVer"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := acpDesired()
			tc.edit(&c)
			err := c.validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validate() = %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestACPPackageNameAndDirectory(t *testing.T) {
	c := acpDesired()
	if got := NPMPackageName(c.Source); got != "@agentclientprotocol/claude-agent-acp" {
		t.Fatalf("NPMPackageName = %q", got)
	}
	if got := ACPPackageDir(c.Name); got != "toolchains/acp/claude-agent-acp" {
		t.Fatalf("ACPPackageDir = %q", got)
	}
}

func TestLockComponentRejectsLocalBuildOnOtherKinds(t *testing.T) {
	lock := loadFixtureLock(t)
	lock.Components[0].ArtifactPolicy = ArtifactPolicyLocalBuild
	if err := lock.Validate(); err == nil || !strings.Contains(err.Error(), "artifact_policy") {
		t.Fatalf("Validate() = %v, want artifact_policy rejection", err)
	}
}

func TestValidateLockRequiresMatchingArtifactPolicy(t *testing.T) {
	manifest := loadFixtureManifest(t)
	lock := loadFixtureLock(t)
	desired := acpDesired()
	manifest.Components = append(manifest.Components, desired)
	locked := lock.Components[0]
	locked.Name, locked.Kind, locked.Source = desired.Name, desired.Kind, desired.Source
	locked.UpstreamVersion, locked.Revision, locked.Runtime = desired.UpstreamVersion, desired.Revision, desired.Runtime
	locked.ArtifactPolicy = desired.ArtifactPolicy
	lock.Components = append(lock.Components, locked)
	if err := manifest.ValidateLock(*lock); err != nil {
		t.Fatalf("ValidateLock() = %v", err)
	}
	lock.Components[len(lock.Components)-1].ArtifactPolicy = ""
	if err := manifest.ValidateLock(*lock); err == nil || !strings.Contains(err.Error(), "artifact_policy") {
		t.Fatalf("ValidateLock() = %v, want artifact_policy mismatch", err)
	}
}
