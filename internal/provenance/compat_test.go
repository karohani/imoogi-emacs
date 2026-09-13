package provenance

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyCLIProvenanceRejectsContradictoryArtifact(t *testing.T) {
	root := t.TempDir()
	binaryPath := "vendor/toolchains/cli/1/darwin-arm64/imoogi-toolchain"
	viewPath := binaryPath + ".provenance.json"
	mustWrite(t, filepath.Join(root, binaryPath), "binary")
	hash, err := HashFile(filepath.Join(root, binaryPath))
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, viewPath), `{"cli_version":"1","builder":"darwin/arm64","artifact":"`+binaryPath+`","size":6,"sha256":"`+strings.Repeat("0", 64)+`"}`)
	domain := Domain{Components: []Component{{
		ID: "toolchain/imoogi-cli", Version: "1", Platform: &Platform{OS: "darwin", Arch: "arm64"},
		Files: []File{{Path: binaryPath, Size: 6, SHA256: hash}, {Path: viewPath}},
	}}}
	got := errorsText(verifyCLIProvenance(root, domain))
	if !strings.Contains(got, "artifact mismatch") {
		t.Fatalf("issues = %q", got)
	}
}

func TestVerifyToolchainDesiredRejectsLockConflict(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "toolchains.json"), `{"cli_version":"1","bundle":"b","target":{"os":"darwin","arch":"arm64"},"components":[{"name":"gopls","upstream_version":"v2","revision":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]}`)
	lock := toolchainLock{CLIVersion: "1", Bundle: "b", Target: Platform{OS: "darwin", Arch: "arm64"}}
	lock.Components = append(lock.Components, toolchainLockComponent{
		Name: "gopls", UpstreamVersion: "v1", Revision: strings.Repeat("a", 40),
	})
	if got := errorsText(verifyToolchainDesired(root, lock)); !strings.Contains(got, "does not match lock") {
		t.Fatalf("issues = %q", got)
	}
}

func TestVerifyToolchainsLockChecksArtifactURLForRevisionedSource(t *testing.T) {
	root := t.TempDir()
	commit := strings.Repeat("a", 40)
	mustWrite(t, filepath.Join(root, "toolchains.json"), `{"cli_version":"1","bundle":"b","target":{"os":"darwin","arch":"arm64"},"components":[{"name":"gopls","upstream_version":"v1","revision":"`+commit+`"}]}`)
	mustWrite(t, filepath.Join(root, "toolchains.lock.json"), `{"cli_version":"1","bundle":"b","target":{"os":"darwin","arch":"arm64"},"components":[{"name":"gopls","upstream_version":"v1","revision":"`+commit+`","artifact":{"path":"vendor/gopls","size":1,"sha256":"`+strings.Repeat("b", 64)+`","source_url":"https://proxy.invalid/wrong.zip"}}]}`)
	domain := Domain{Components: []Component{
		{ID: "toolchain/gopls", Version: "v1", Source: Source{Type: "git", URL: "https://example.invalid/tools", ArtifactURL: "https://proxy.invalid/right.zip", Commit: commit}, Files: []File{{Path: "vendor/gopls", Size: 1, SHA256: strings.Repeat("b", 64)}}},
		{ID: "toolchain/imoogi-cli", Version: "1"},
	}}
	if got := errorsText(verifyToolchainsLock(root, domain)); !strings.Contains(got, "source URL mismatch") {
		t.Fatalf("issues = %q", got)
	}
}

func TestVerifyToolchainsLockRejectsPlatformConflict(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "toolchains.json"), `{"cli_version":"1","bundle":"b","target":{"os":"darwin","arch":"arm64"},"components":[{"name":"demo","upstream_version":"v1"}]}`)
	mustWrite(t, filepath.Join(root, "toolchains.lock.json"), `{"cli_version":"1","bundle":"b","target":{"os":"darwin","arch":"arm64"},"components":[{"name":"demo","upstream_version":"v1","artifact":{"path":"vendor/demo","size":1,"sha256":"`+strings.Repeat("b", 64)+`","source_url":"https://example.invalid/demo"}}]}`)
	domain := Domain{Components: []Component{
		{ID: "toolchain/demo", Version: "v1", Source: Source{URL: "https://example.invalid/demo"}, Platform: &Platform{OS: "linux", Arch: "amd64"}, Files: []File{{Path: "vendor/demo", Size: 1, SHA256: strings.Repeat("b", 64)}}},
		{ID: "toolchain/imoogi-cli", Version: "1"},
	}}
	if got := errorsText(verifyToolchainsLock(root, domain)); !strings.Contains(got, "platform mismatch") {
		t.Fatalf("issues = %q", got)
	}
}
