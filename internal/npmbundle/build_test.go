package npmbundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karohani/imoogi-emacs/internal/artifact"
)

const (
	testPkg     = "@scope/adapter"
	testVersion = "1.2.3"
)

// fakeNode writes a node distribution archive holding the two files Build runs.
func fakeNode(t *testing.T) (string, artifact.Digest) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{
		"node-v24.19.0-darwin-arm64/bin/node":                            "node\n",
		"node-v24.19.0-darwin-arm64/lib/node_modules/npm/bin/npm-cli.js": "npm\n",
	} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "node.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return path, artifact.Digest{Size: int64(buf.Len()), SHA256: hex.EncodeToString(sum[:])}
}

const testLockfile = `{
  "name": "wrapper",
  "lockfileVersion": 3,
  "packages": {
    "": {"dependencies": {"@scope/adapter": "1.2.3"}},
    "node_modules/@scope/adapter": {"version": "1.2.3"}
  }
}
`

func packageDir(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"package.json":      `{"name":"wrapper","private":true,"dependencies":{"@scope/adapter":"1.2.3"}}`,
		"package-lock.json": testLockfile,
	})
	sum := sha256.Sum256([]byte(testLockfile))
	return dir, hex.EncodeToString(sum[:])
}

// installer is a fake npm: it records argv and lays out what npm ci would.
func installer(calls *[][]string, version string) Runner {
	return func(_ context.Context, dir string, argv []string) error {
		*calls = append(*calls, append([]string{dir}, argv...))
		// The extractor writes 0644 files; node must be made executable to run.
		if info, err := os.Stat(argv[0]); err != nil || info.Mode().Perm()&0o100 == 0 {
			return errors.New("node is not executable: " + argv[0])
		}
		for _, required := range []string{"package.json", "package-lock.json"} {
			if _, err := os.Stat(filepath.Join(dir, required)); err != nil {
				return err
			}
		}
		files := map[string]string{
			"node_modules/@scope/adapter/package.json":  `{"name":"@scope/adapter","version":"` + version + `","bin":{"adapter":"dist/index.js"}}`,
			"node_modules/@scope/adapter/dist/index.js": "main\n",
			"node_modules/@scope/adapter/LICENSE":       "Apache license\n",
			"node_modules/.package-lock.json":           "{}",
		}
		for rel, body := range files {
			path := filepath.Join(dir, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				return err
			}
		}
		return nil
	}
}

func buildOptions(t *testing.T, run Runner) BuildOptions {
	t.Helper()
	dir, lockSum := packageDir(t)
	nodeArchive, nodeDigest := fakeNode(t)
	out := t.TempDir()
	return BuildOptions{
		PackageDir:    dir,
		LockSHA256:    lockSum,
		Package:       testPkg,
		Version:       testVersion,
		NodeArchive:   nodeArchive,
		NodeDigest:    nodeDigest,
		Output:        filepath.Join(out, "bundle.tgz"),
		LicenseOutput: filepath.Join(out, "LICENSE"),
		Run:           run,
	}
}

func TestBuildRunsNpmCiWithTheLockedNodeAndPacks(t *testing.T) {
	var calls [][]string
	opts := buildOptions(t, installer(&calls, testVersion))
	digest, err := Build(context.Background(), opts)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("runner calls = %d", len(calls))
	}
	argv := calls[0][1:]
	if !strings.HasSuffix(argv[0], "node-v24.19.0-darwin-arm64/bin/node") ||
		!strings.HasSuffix(argv[1], "node-v24.19.0-darwin-arm64/lib/node_modules/npm/bin/npm-cli.js") {
		t.Fatalf("npm did not run from the locked node: %q", argv[:2])
	}
	joined := strings.Join(argv[2:], " ")
	for _, flag := range CIArgs {
		if !strings.Contains(joined, flag) {
			t.Fatalf("npm args %q miss %q", joined, flag)
		}
	}
	if !strings.Contains(joined, "--cache ") {
		t.Fatalf("npm args %q do not isolate the cache", joined)
	}
	got, err := DigestFile(opts.Output)
	if err != nil || got != digest {
		t.Fatalf("returned digest %+v, file digest %+v (%v)", digest, got, err)
	}
	license, err := os.ReadFile(opts.LicenseOutput)
	if err != nil || string(license) != "Apache license\n" {
		t.Fatalf("license = %q, %v", license, err)
	}
	if _, err := os.Stat(calls[0][0]); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("build directory %s was left behind (%v)", calls[0][0], err)
	}
}

func TestBuildTwiceIsByteIdentical(t *testing.T) {
	var calls [][]string
	first := buildOptions(t, installer(&calls, testVersion))
	a, err := Build(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	second := first
	second.Output = filepath.Join(t.TempDir(), "bundle.tgz")
	second.LicenseOutput = ""
	b, err := Build(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("two builds differ: %+v vs %+v", a, b)
	}
}

func TestBuildRejectsInputsBeforeRunningNpm(t *testing.T) {
	tests := map[string]struct {
		edit func(*BuildOptions)
		want string
	}{
		"lockfile digest": {func(o *BuildOptions) { o.LockSHA256 = strings.Repeat("0", 64) }, "package-lock.json"},
		"pinned version":  {func(o *BuildOptions) { o.Version = "1.2.4" }, "1.2.4"},
		"node digest":     {func(o *BuildOptions) { o.NodeDigest.SHA256 = strings.Repeat("0", 64) }, "node"},
		"existing output": {func(o *BuildOptions) { _ = os.WriteFile(o.Output, []byte("x"), 0o644) }, "exists"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var calls [][]string
			opts := buildOptions(t, installer(&calls, testVersion))
			tc.edit(&opts)
			_, err := Build(context.Background(), opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Build() error = %v, want %q", err, tc.want)
			}
			if len(calls) != 0 {
				t.Fatal("npm ran despite invalid inputs")
			}
		})
	}
}

func TestBuildRejectsAnInstallThatDoesNotMatchThePin(t *testing.T) {
	var calls [][]string
	opts := buildOptions(t, installer(&calls, "9.9.9"))
	if _, err := Build(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "9.9.9") {
		t.Fatalf("Build() error = %v, want installed version mismatch", err)
	}
	if _, err := os.Stat(opts.Output); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a rejected build left an artifact behind")
	}
}

func TestBuildSurfacesNpmFailure(t *testing.T) {
	opts := buildOptions(t, func(context.Context, string, []string) error { return errors.New("ENOTFOUND registry.npmjs.org") })
	if _, err := Build(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "ENOTFOUND") {
		t.Fatalf("Build() error = %v", err)
	}
}

func TestBinEntryReadsTheSingleBin(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"package.json":  `{"name":"@scope/adapter","version":"1.2.3","bin":{"adapter":"./dist/index.js"}}`,
		"dist/index.js": "main\n",
	})
	got, err := BinEntry(dir, testPkg, testVersion)
	if err != nil || got != "dist/index.js" {
		t.Fatalf("BinEntry() = %q, %v", got, err)
	}
	writeTree(t, dir, map[string]string{"package.json": `{"name":"@scope/adapter","version":"1.2.3","bin":{"a":"../x.js"}}`})
	if _, err := BinEntry(dir, testPkg, testVersion); err == nil {
		t.Fatal("BinEntry accepted an escaping bin path")
	}
}
