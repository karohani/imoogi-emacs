package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyAcceptsCompleteManifest(t *testing.T) {
	root := fixture(t)
	if issues := Verify(root, "vendor-manifest.json"); len(issues) != 0 {
		t.Fatalf("Verify() issues = %v", issues)
	}
}

func TestVerifyRejectsUnownedAndModifiedFiles(t *testing.T) {
	root := fixture(t)
	mustWrite(t, filepath.Join(root, "vendor/pkg/new.txt"), "new")
	mustWrite(t, filepath.Join(root, "vendor/pkg/file.txt"), "changed")
	issues := Verify(root, "vendor-manifest.json")
	got := errorsText(issues)
	for _, want := range []string{"unowned file: vendor/pkg/new.txt", "size mismatch", "sha256 mismatch"} {
		if !strings.Contains(got, want) {
			t.Errorf("issues %q do not contain %q", got, want)
		}
	}
}

func TestVerifyRejectsMissingSourceIdentity(t *testing.T) {
	root := fixture(t)
	var d Domain
	if err := loadStrict(filepath.Join(root, "provenance/test.json"), &d); err != nil {
		t.Fatal(err)
	}
	d.Components[0].Source.Commit = "v1"
	writeJSON(t, filepath.Join(root, "provenance/test.json"), d)
	if got := errorsText(Verify(root, "vendor-manifest.json")); !strings.Contains(got, "full 40-hex commit") {
		t.Fatalf("issues = %q", got)
	}
}

func TestVerifyExclusionsMustBeExactAndExplained(t *testing.T) {
	root := fixture(t)
	var idx Index
	if err := loadStrict(filepath.Join(root, "vendor-manifest.json"), &idx); err != nil {
		t.Fatal(err)
	}
	idx.Excludes = []Exclusion{{Path: "vendor/**", Reason: ""}}
	writeJSON(t, filepath.Join(root, "vendor-manifest.json"), idx)
	got := errorsText(Verify(root, "vendor-manifest.json"))
	for _, want := range []string{"has no reason", "must be an exact path"} {
		if !strings.Contains(got, want) {
			t.Errorf("issues %q do not contain %q", got, want)
		}
	}
}

func TestVerifyRejectsStaleExclusion(t *testing.T) {
	root := fixture(t)
	var idx Index
	if err := loadStrict(filepath.Join(root, "vendor-manifest.json"), &idx); err != nil {
		t.Fatal(err)
	}
	idx.Excludes = []Exclusion{{Path: "vendor/pkg/missing.txt", Reason: "test"}}
	writeJSON(t, filepath.Join(root, "vendor-manifest.json"), idx)
	if got := errorsText(Verify(root, "vendor-manifest.json")); !strings.Contains(got, "exclusion is stale") {
		t.Fatalf("issues = %q", got)
	}
}

func TestVerifyRejectsDuplicateOwner(t *testing.T) {
	root := fixture(t)
	var domain Domain
	if err := loadStrict(filepath.Join(root, "provenance/test.json"), &domain); err != nil {
		t.Fatal(err)
	}
	duplicate := domain.Components[0]
	duplicate.ID = "other"
	domain.Components = append(domain.Components, duplicate)
	writeJSON(t, filepath.Join(root, "provenance/test.json"), domain)
	if got := errorsText(Verify(root, "vendor-manifest.json")); !strings.Contains(got, "duplicate ownership") {
		t.Fatalf("issues = %q", got)
	}
}

func TestVerifyRejectsMalformedHashAndPlatformlessBinary(t *testing.T) {
	root := fixture(t)
	var domain Domain
	if err := loadStrict(filepath.Join(root, "provenance/test.json"), &domain); err != nil {
		t.Fatal(err)
	}
	domain.Components[0].Kind = "native-binary"
	domain.Components[0].Files[0].SHA256 = "bad"
	writeJSON(t, filepath.Join(root, "provenance/test.json"), domain)
	got := errorsText(Verify(root, "vendor-manifest.json"))
	for _, want := range []string{"platform os/arch is required", "malformed sha256"} {
		if !strings.Contains(got, want) {
			t.Errorf("issues %q do not contain %q", got, want)
		}
	}
}

func TestVerifyRejectsEscapingPathAndSymlink(t *testing.T) {
	root := fixture(t)
	if err := os.Symlink(filepath.Join(root, "vendor/pkg/file.txt"), filepath.Join(root, "vendor/pkg/link.txt")); err != nil {
		t.Fatal(err)
	}
	var domain Domain
	if err := loadStrict(filepath.Join(root, "provenance/test.json"), &domain); err != nil {
		t.Fatal(err)
	}
	domain.Components[0].Files = append(domain.Components[0].Files,
		File{Path: "../escape", SHA256: strings.Repeat("a", 64)},
		File{Path: "vendor/pkg/link.txt", SHA256: strings.Repeat("a", 64)})
	writeJSON(t, filepath.Join(root, "provenance/test.json"), domain)
	got := errorsText(Verify(root, "vendor-manifest.json"))
	for _, want := range []string{"unsafe repository-relative path", "symlink is not allowed"} {
		if !strings.Contains(got, want) {
			t.Errorf("issues %q do not contain %q", got, want)
		}
	}
}

func TestVerifyDoesNotRewriteManifest(t *testing.T) {
	root := fixture(t)
	path := filepath.Join(root, "provenance/test.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if issues := Verify(root, "vendor-manifest.json"); len(issues) != 0 {
		t.Fatal(issues)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("Verify rewrote a manifest")
	}
}

func TestVerifyRejectsFileOutsideDeclaredRootsWithinBoundary(t *testing.T) {
	root := fixture(t)
	mustWrite(t, filepath.Join(root, "vendor/new-external/file.bin"), "new")
	if got := errorsText(Verify(root, "vendor-manifest.json")); !strings.Contains(got, "file outside declared roots") {
		t.Fatalf("issues = %q", got)
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	path := "vendor/pkg/file.txt"
	mustWrite(t, filepath.Join(root, path), "original")
	hash, err := HashFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(root, "vendor-manifest.json"), Index{
		Schema: IndexSchema, Boundaries: []string{"vendor"}, Roots: []string{"vendor/pkg"}, Manifests: []string{"provenance/test.json"},
	})
	writeJSON(t, filepath.Join(root, "provenance/test.json"), Domain{
		Schema: DomainSchema, Domain: "test", Components: []Component{{
			ID: "pkg", Kind: "source", Version: "1.0.0", Workflow: "test fixture",
			Source: Source{Type: "git", URL: "https://example.invalid/pkg.git", Commit: strings.Repeat("a", 40)},
			Files:  []File{{Path: path, SHA256: hash, Size: 8}},
		}},
	})
	return root
}

func mustWrite(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func errorsText(issues []error) string {
	parts := make([]string, len(issues))
	for i, issue := range issues {
		parts[i] = issue.Error()
	}
	return strings.Join(parts, "\n")
}
