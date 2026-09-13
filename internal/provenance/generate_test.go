package provenance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateIsDeterministicAndExpandsEveryFile(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "vendor/elpa/demo-1.0/demo-pkg.el"),
		`(define-package "demo" "1.0" "Demo" nil :url "https://example.invalid/demo" :commit "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")`)
	mustWrite(t, filepath.Join(root, "vendor/elpa/demo-1.0/demo.el"), "demo")
	mustWrite(t, filepath.Join(root, "vendor/blob.bin"), "blob")
	mustWrite(t, filepath.Join(root, "provenance/sources.json"), `{
  "schema":"imoogi-vendor-sources/v1",
  "roots":["vendor/elpa","vendor/blob.bin"],
  "components":[{"domain":"binary","id":"binary/blob","kind":"binary","version":"1","source":{"type":"download","url":"https://example.invalid/blob-1"},"workflow":"test","paths":["vendor/blob.bin"]}]
}`)
	if err := Generate(root, "provenance/sources.json", "vendor-manifest.json"); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(root, "provenance/elpa.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), "vendor/elpa/demo-1.0/demo.el") {
		t.Fatalf("generated manifest omits package file: %s", first)
	}
	if err := Generate(root, "provenance/sources.json", "vendor-manifest.json"); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(root, "provenance/elpa.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("same inputs produced different manifests")
	}
}

func TestGenerateRejectsGitPackageWithoutImmutableCommit(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "vendor/elpa/demo-1.0/demo-pkg.el"),
		`(define-package "demo" "1.0" "Demo" nil :url "https://example.invalid/demo")`)
	mustWrite(t, filepath.Join(root, "provenance/sources.json"),
		`{"schema":"imoogi-vendor-sources/v1","roots":["vendor/elpa"],"components":[]}`)
	err := Generate(root, "provenance/sources.json", "vendor-manifest.json")
	if err == nil || !strings.Contains(err.Error(), "no full upstream commit") {
		t.Fatalf("error = %v", err)
	}
}

func TestRecordGitSourceCapturesObservedBuildRevision(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "provenance/sources.json")
	mustWrite(t, path, `{"schema":"imoogi-vendor-sources/v1","roots":["vendor"],"components":[{"domain":"tree-sitter","id":"grammar/go","kind":"grammar-binary","version":"v1","source":{"type":"git","url":"https://example.invalid/go","commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"platform":{"os":"darwin","arch":"arm64"},"workflow":"test","paths":["vendor/go.dylib"]}]}`)
	commit := strings.Repeat("b", 40)
	if err := RecordGitSource(root, "provenance/sources.json", "grammar/go", commit, "v1"); err != nil {
		t.Fatal(err)
	}
	var sources Sources
	if err := loadStrict(path, &sources); err != nil {
		t.Fatal(err)
	}
	got := sources.Components[0].Source
	if got.Commit != commit || got.Ref != "v1" {
		t.Fatalf("recorded source = %#v", got)
	}
}
