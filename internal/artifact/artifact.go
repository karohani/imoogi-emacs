package artifact

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	defaultMaxEntries     = 4096
	defaultMaxFileBytes   = 512 << 20
	defaultMaxOutputBytes = 2 << 30
	fileMode              = 0o644
	dirMode               = 0o755
)

type Digest struct {
	Size   int64
	SHA256 string
}

type Limits struct {
	MaxEntries     int
	MaxFileBytes   int64
	MaxOutputBytes int64
}

type ExtractOptions struct {
	AllowRelativeSymlinks bool
}

type ExtractionResult struct {
	Entries int
	Bytes   int64
}

func VerifyFile(file string, expected Digest) error {
	if expected.Size <= 0 {
		return fmt.Errorf("expected size %d is invalid", expected.Size)
	}
	if !isLowerSHA256(expected.SHA256) {
		return fmt.Errorf("expected sha256 %q is invalid", expected.SHA256)
	}

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("open artifact: %w", err)
	}
	defer f.Close()

	hash := sha256.New()
	n, err := io.Copy(hash, f)
	if err != nil {
		return fmt.Errorf("read artifact: %w", err)
	}
	if n != expected.Size {
		return fmt.Errorf("artifact size mismatch: got %d want %d", n, expected.Size)
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	if sum != expected.SHA256 {
		return fmt.Errorf("artifact sha256 mismatch: got %s want %s", sum, expected.SHA256)
	}
	return nil
}

func ExtractTarGzip(archivePath, destination string, expected Digest, limits Limits) (ExtractionResult, error) {
	return ExtractTarGzipWithOptions(archivePath, destination, expected, limits, ExtractOptions{})
}

func ExtractTarGzipWithOptions(archivePath, destination string, expected Digest, limits Limits, options ExtractOptions) (ExtractionResult, error) {
	if err := VerifyFile(archivePath, expected); err != nil {
		return ExtractionResult{}, err
	}
	limits = withDefaultLimits(limits)
	if err := validateDestination(destination); err != nil {
		return ExtractionResult{}, err
	}

	parent := filepath.Dir(destination)
	base := filepath.Base(destination)
	temp, err := os.MkdirTemp(parent, "."+base+".extract-*")
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("create extraction staging: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(temp)
		}
	}()

	result, err := extractVerifiedTarGzip(archivePath, temp, limits, options)
	if err != nil {
		return ExtractionResult{}, err
	}
	if err := commitStaging(temp, destination); err != nil {
		return ExtractionResult{}, err
	}
	committed = true
	return result, nil
}

func extractVerifiedTarGzip(archivePath, root string, limits Limits, options ExtractOptions) (ExtractionResult, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("open verified artifact: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("open gzip stream: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	seen := map[string]entryKind{}
	var links []pendingSymlink
	var result ExtractionResult
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return ExtractionResult{}, fmt.Errorf("read tar entry: %w", err)
		}
		result.Entries++
		if result.Entries > limits.MaxEntries {
			return ExtractionResult{}, fmt.Errorf("archive has too many entries: %d > %d", result.Entries, limits.MaxEntries)
		}
		if err := validateEntryHeader(hdr, seen, limits, &result, options); err != nil {
			return ExtractionResult{}, err
		}
		entryPath, err := containedPath(root, hdr.Name)
		if err != nil {
			return ExtractionResult{}, err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := mkdirNoSymlinkBelow(root, entryPath); err != nil {
				return ExtractionResult{}, fmt.Errorf("create directory %q: %w", hdr.Name, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := mkdirNoSymlinkBelow(root, filepath.Dir(entryPath)); err != nil {
				return ExtractionResult{}, fmt.Errorf("create parent for %q: %w", hdr.Name, err)
			}
			if err := writeRegularFile(entryPath, tr, hdr.Size); err != nil {
				return ExtractionResult{}, fmt.Errorf("extract file %q: %w", hdr.Name, err)
			}
		case tar.TypeSymlink:
			link, err := validateSymlinkTarget(hdr.Name, hdr.Linkname)
			if err != nil {
				return ExtractionResult{}, err
			}
			links = append(links, link)
		default:
			return ExtractionResult{}, fmt.Errorf("archive entry %q has unsupported type %q", hdr.Name, hdr.Typeflag)
		}
	}
	if err := createVerifiedSymlinks(root, seen, links); err != nil {
		return ExtractionResult{}, err
	}
	return result, nil
}

type entryKind int

const (
	kindDir entryKind = iota + 1
	kindFile
	kindSymlink
)

type pendingSymlink struct {
	Name       string
	Target     string
	TargetName string
}

func validateEntryHeader(hdr *tar.Header, seen map[string]entryKind, limits Limits, result *ExtractionResult, options ExtractOptions) error {
	name, err := cleanArchivePath(hdr.Name)
	if err != nil {
		return fmt.Errorf("archive entry path %q: %w", hdr.Name, err)
	}
	hdr.Name = name

	kind, err := kindForType(hdr.Typeflag, options)
	if err != nil {
		return fmt.Errorf("archive entry %q: %w", name, err)
	}
	if _, ok := seen[name]; ok {
		return fmt.Errorf("archive entry %q is duplicated", name)
	}
	if err := detectPathConflict(name, kind, seen); err != nil {
		return err
	}
	if kind == kindFile {
		if hdr.Size < 0 {
			return fmt.Errorf("archive entry %q has negative size", name)
		}
		if hdr.Size > limits.MaxFileBytes {
			return fmt.Errorf("archive entry %q is too large: %d > %d", name, hdr.Size, limits.MaxFileBytes)
		}
		if result.Bytes > limits.MaxOutputBytes-hdr.Size {
			return fmt.Errorf("archive output is too large: exceeds %d bytes", limits.MaxOutputBytes)
		}
		result.Bytes += hdr.Size
	} else if kind == kindDir && hdr.Size != 0 {
		return fmt.Errorf("archive directory %q has non-zero size", name)
	} else if kind == kindSymlink && hdr.Size != 0 {
		return fmt.Errorf("archive symlink %q has non-zero size", name)
	}
	seen[name] = kind
	return nil
}

func kindForType(typeflag byte, options ExtractOptions) (entryKind, error) {
	switch typeflag {
	case tar.TypeDir:
		return kindDir, nil
	case tar.TypeReg, tar.TypeRegA:
		return kindFile, nil
	case tar.TypeSymlink:
		if !options.AllowRelativeSymlinks {
			return 0, errors.New("symlinks are not supported")
		}
		return kindSymlink, nil
	case tar.TypeLink:
		return 0, errors.New("hardlinks are not supported")
	case tar.TypeChar, tar.TypeBlock:
		return 0, errors.New("device files are not supported")
	case tar.TypeFifo:
		return 0, errors.New("FIFOs are not supported")
	default:
		return 0, fmt.Errorf("type %q is unsupported", typeflag)
	}
}

func detectPathConflict(name string, kind entryKind, seen map[string]entryKind) error {
	parts := strings.Split(name, "/")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[:i], "/")
		if seen[parent] == kindFile || seen[parent] == kindSymlink {
			return fmt.Errorf("archive entry %q conflicts with file parent %q", name, parent)
		}
	}
	if kind == kindDir {
		prefix := name + "/"
		for existing := range seen {
			if strings.HasPrefix(existing, prefix) {
				return fmt.Errorf("archive directory %q appears after child %q", name, existing)
			}
		}
	}
	return nil
}

func validateSymlinkTarget(name, target string) (pendingSymlink, error) {
	if target == "" {
		return pendingSymlink{}, fmt.Errorf("archive symlink %q has empty target", name)
	}
	if strings.Contains(target, `\`) {
		return pendingSymlink{}, fmt.Errorf("archive symlink %q target contains backslash", name)
	}
	if strings.Contains(target, ":") {
		return pendingSymlink{}, fmt.Errorf("archive symlink %q target contains drive or scheme separator", name)
	}
	if path.IsAbs(target) {
		return pendingSymlink{}, fmt.Errorf("archive symlink %q target is absolute", name)
	}
	cleanTarget := path.Clean(target)
	if cleanTarget == "." {
		return pendingSymlink{}, fmt.Errorf("archive symlink %q target is empty", name)
	}
	if cleanTarget != target {
		return pendingSymlink{}, fmt.Errorf("archive symlink %q target is not clean", name)
	}
	resolved := path.Clean(path.Join(path.Dir(name), cleanTarget))
	if resolved == "." || strings.HasPrefix(resolved, "../") || resolved == ".." || path.IsAbs(resolved) {
		return pendingSymlink{}, fmt.Errorf("archive symlink %q target escapes archive root", name)
	}
	return pendingSymlink{Name: name, Target: cleanTarget, TargetName: resolved}, nil
}

func createVerifiedSymlinks(root string, seen map[string]entryKind, links []pendingSymlink) error {
	for _, link := range links {
		targetKind, ok := seen[link.TargetName]
		if !ok {
			return fmt.Errorf("archive symlink %q target %q is not present", link.Name, link.TargetName)
		}
		if targetKind == kindSymlink {
			return fmt.Errorf("archive symlink %q target %q is another symlink", link.Name, link.TargetName)
		}
		linkPath, err := containedPath(root, link.Name)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(filepath.Dir(linkPath), filepath.FromSlash(link.Target))
		if err := ensurePathBelow(root, targetPath); err != nil {
			return fmt.Errorf("archive symlink %q target escapes destination: %w", link.Name, err)
		}
		info, err := os.Lstat(targetPath)
		if err != nil {
			return fmt.Errorf("archive symlink %q target %q is not materialized: %w", link.Name, link.TargetName, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive symlink %q target %q is a symlink", link.Name, link.TargetName)
		}
		if err := mkdirNoSymlinkBelow(root, filepath.Dir(linkPath)); err != nil {
			return fmt.Errorf("create parent for symlink %q: %w", link.Name, err)
		}
		if _, err := os.Lstat(linkPath); err == nil {
			return fmt.Errorf("archive symlink %q destination already exists", link.Name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect symlink destination %q: %w", link.Name, err)
		}
		if err := os.Symlink(link.Target, linkPath); err != nil {
			return fmt.Errorf("create symlink %q: %w", link.Name, err)
		}
	}
	return nil
}

func cleanArchivePath(name string) (string, error) {
	if name == "" {
		return "", errors.New("empty path")
	}
	name = strings.TrimRight(name, "/")
	if name == "" {
		return "", errors.New("empty path")
	}
	if strings.Contains(name, `\`) {
		return "", errors.New("backslash is not allowed")
	}
	if strings.Contains(name, ":") {
		return "", errors.New("drive or scheme separator is not allowed")
	}
	if path.IsAbs(name) {
		return "", errors.New("absolute path is not allowed")
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", errors.New("path traversal is not allowed")
	}
	if clean != name {
		return "", errors.New("path is not clean")
	}
	for _, part := range strings.Split(clean, "/") {
		if part == "." || part == ".." || part == "" {
			return "", errors.New("path contains invalid segment")
		}
	}
	return clean, nil
}

func containedPath(root, archiveName string) (string, error) {
	clean, err := cleanArchivePath(archiveName)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	target := filepath.Join(rootAbs, filepath.FromSlash(clean))
	rel, err := filepath.Rel(rootAbs, target)
	if err != nil {
		return "", fmt.Errorf("check containment: %w", err)
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("entry %q escapes destination", archiveName)
	}
	return target, nil
}

func ensurePathBelow(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve target: %w", err)
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return fmt.Errorf("check containment: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%s escapes %s", targetAbs, rootAbs)
	}
	return nil
}

func writeRegularFile(file string, r io.Reader, size int64) error {
	if _, err := os.Lstat(file); err == nil {
		return errors.New("destination file already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.CopyN(f, r, size)
	if err != nil {
		return err
	}
	if n != size {
		return fmt.Errorf("copied %d bytes, want %d", n, size)
	}
	return f.Chmod(fileMode)
}

func mkdirNoSymlinkBelow(root, dir string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	rootInfo, err := os.Lstat(rootAbs)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", rootAbs)
	}
	if !rootInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", rootAbs)
	}

	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, dirAbs)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%s escapes extraction root %s", dirAbs, rootAbs)
	}

	current := rootAbs
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, dirMode); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", current)
		}
	}
	return nil
}

func validateDestination(destination string) error {
	if destination == "" {
		return errors.New("destination is empty")
	}
	info, err := os.Lstat(destination)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect destination: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("destination is a symlink")
	}
	if !info.IsDir() {
		return errors.New("destination is not a directory")
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		return fmt.Errorf("read destination: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("destination is not empty")
	}
	return nil
}

func commitStaging(staging, destination string) error {
	info, err := os.Lstat(destination)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := os.Rename(staging, destination); err != nil {
			return fmt.Errorf("commit extraction: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("inspect destination: %w", err)
	case info.Mode()&os.ModeSymlink != 0:
		return errors.New("destination is a symlink")
	case !info.IsDir():
		return errors.New("destination is not a directory")
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		return fmt.Errorf("read destination: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("destination is not empty")
	}
	if err := os.Remove(destination); err != nil {
		return fmt.Errorf("remove empty destination: %w", err)
	}
	if err := os.Rename(staging, destination); err != nil {
		_ = os.Mkdir(destination, dirMode)
		return fmt.Errorf("commit extraction: %w", err)
	}
	return nil
}

func withDefaultLimits(l Limits) Limits {
	if l.MaxEntries <= 0 {
		l.MaxEntries = defaultMaxEntries
	}
	if l.MaxFileBytes <= 0 {
		l.MaxFileBytes = defaultMaxFileBytes
	}
	if l.MaxOutputBytes <= 0 {
		l.MaxOutputBytes = defaultMaxOutputBytes
	}
	return l
}

func isLowerSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}
