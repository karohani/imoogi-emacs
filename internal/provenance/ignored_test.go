package provenance

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// gitFixture turns the plain fixture into a git repository whose .gitignore
// hides .DS_Store, then drops one such file inside the covered root.
func gitFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := fixture(t)
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@example.invalid"}, {"config", "user.name", "t"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	mustWrite(t, filepath.Join(root, ".gitignore"), ".DS_Store\n")
	mustWrite(t, filepath.Join(root, "vendor/pkg/.DS_Store"), "finder junk")
	return root
}

func TestVerifySkipsGitIgnoredFiles(t *testing.T) {
	root := gitFixture(t)
	if issues := Verify(root, "vendor-manifest.json"); len(issues) != 0 {
		t.Fatalf("Verify() should ignore git-ignored files, got %v", issues)
	}
}

func TestExpandFilesSkipsGitIgnoredFiles(t *testing.T) {
	root := gitFixture(t)
	files, err := expandFiles(root, []string{"vendor/pkg"}, nil, gitIgnored(root))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if filepath.Base(f.Path) == ".DS_Store" {
			t.Fatalf("expandFiles() included git-ignored file %s", f.Path)
		}
	}
	if len(files) != 1 {
		t.Fatalf("expandFiles() = %d files, want 1", len(files))
	}
}
