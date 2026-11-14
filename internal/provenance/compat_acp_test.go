package provenance

import (
	"path/filepath"
	"strings"
	"testing"
)

// acpFixture writes a desired manifest and a lock holding one acp-adapter whose
// revision is the digest of its committed package-lock.json.
func acpFixture(t *testing.T, policy string) (string, Component) {
	t.Helper()
	root := t.TempDir()
	lockfile := "toolchains/acp/codex-acp/package-lock.json"
	mustWrite(t, filepath.Join(root, lockfile), `{"lockfileVersion":3}`)
	lockSum, err := HashFile(filepath.Join(root, lockfile))
	if err != nil {
		t.Fatal(err)
	}
	policyField := ""
	if policy != "" {
		policyField = `,"artifact_policy":"` + policy + `"`
	}
	bundle := "vendor/toolchains/acp/codex-acp/2.5.0/darwin-arm64/bundle.tgz"
	url := "https://registry.npmjs.org/@agentclientprotocol/codex-acp/-/codex-acp-2.5.0.tgz"
	mustWrite(t, filepath.Join(root, "toolchains.json"), `{"cli_version":"1","bundle":"b","target":{"os":"darwin","arch":"arm64"},"components":[{"name":"codex-acp","kind":"acp-adapter","upstream_version":"2.5.0","revision":"`+lockSum+`"`+policyField+`}]}`)
	mustWrite(t, filepath.Join(root, "toolchains.lock.json"), `{"cli_version":"1","bundle":"b","target":{"os":"darwin","arch":"arm64"},"components":[{"name":"codex-acp","kind":"acp-adapter","upstream_version":"2.5.0","revision":"`+lockSum+`"`+policyField+`,"artifact":{"path":"`+bundle+`","size":1,"sha256":"`+strings.Repeat("b", 64)+`","source_url":"`+url+`"}}]}`)
	files := []File{{Path: lockfile, SHA256: lockSum, Size: 21}}
	if policy == "" {
		files = append(files, File{Path: bundle, Size: 1, SHA256: strings.Repeat("b", 64)})
	}
	return root, Component{ID: "toolchain/codex-acp", Version: "2.5.0", Source: Source{Type: "archive", URL: url},
		Platform: &Platform{OS: "darwin", Arch: "arm64"}, Files: files}
}

func domainWith(c Component) Domain {
	return Domain{Components: []Component{c, {ID: "toolchain/imoogi-cli", Version: "1"}}}
}

func TestVerifyToolchainsLockAcceptsACPAdapterRevisionAsLockfileDigest(t *testing.T) {
	root, component := acpFixture(t, "")
	if got := errorsText(verifyToolchainsLock(root, domainWith(component))); got != "" {
		t.Fatalf("issues = %q", got)
	}
}

func TestVerifyToolchainsLockRejectsACPRevisionWithoutMatchingOwnedLockfile(t *testing.T) {
	root, component := acpFixture(t, "")
	component.Files[0].SHA256 = strings.Repeat("c", 64)
	if got := errorsText(verifyToolchainsLock(root, domainWith(component))); !strings.Contains(got, "package-lock.json") {
		t.Fatalf("issues = %q", got)
	}
}

func TestVerifyToolchainsLockExemptsLocalBuildArtifactFromOwnership(t *testing.T) {
	root, component := acpFixture(t, "local-build")
	if got := errorsText(verifyToolchainsLock(root, domainWith(component))); got != "" {
		t.Fatalf("issues = %q", got)
	}
	// A committed artifact is still required to be owned.
	root, component = acpFixture(t, "")
	component.Files = component.Files[:1]
	if got := errorsText(verifyToolchainsLock(root, domainWith(component))); !strings.Contains(got, "not owned") {
		t.Fatalf("issues = %q", got)
	}
}

func TestVerifyToolchainsLockRejectsAnOwnedLocalBuildArtifact(t *testing.T) {
	root, component := acpFixture(t, "local-build")
	component.Files = append(component.Files, File{Path: "vendor/toolchains/acp/codex-acp/2.5.0/darwin-arm64/bundle.tgz", Size: 1, SHA256: strings.Repeat("b", 64)})
	if got := errorsText(verifyToolchainsLock(root, domainWith(component))); !strings.Contains(got, "local-build") {
		t.Fatalf("issues = %q", got)
	}
}
