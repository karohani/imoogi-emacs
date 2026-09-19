package notemove

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Request struct {
	Operation   string `json:"operation"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type Response struct {
	OK          bool   `json:"ok"`
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination,omitempty"`
	Files       int    `json:"files,omitempty"`
	Bytes       int64  `json:"bytes,omitempty"`
	Code        string `json:"code,omitempty"`
	Error       string `json:"error,omitempty"`
	Warning     string `json:"warning,omitempty"`
}

type manifestEntry struct {
	Path   string
	Kind   string
	Digest string
	Size   int64
}

func Move(request Request) Response {
	if request.Operation != "move" {
		return failure("invalid_request", "operation must be move")
	}
	source, destination, err := validate(request.Source, request.Destination)
	if err != nil {
		return failure("invalid_path", err.Error())
	}

	before, files, bytes, err := manifest(source)
	if err != nil {
		return failure("source_read_failed", err.Error())
	}
	parent := filepath.Dir(destination)
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(destination)+".imoogi-moving-")
	if err != nil {
		return failure("staging_failed", err.Error())
	}
	keepStaging := false
	defer func() {
		if !keepStaging {
			_ = os.RemoveAll(staging)
		}
	}()

	if err := copyTree(source, staging); err != nil {
		return failure("copy_failed", err.Error())
	}
	after, _, _, err := manifest(source)
	if err != nil || !equalManifest(before, after) {
		return failure("source_changed", "source changed while it was being copied")
	}
	copied, _, _, err := manifest(staging)
	if err != nil || !equalManifest(before, copied) {
		return failure("verification_failed", "copied tree does not match source")
	}
	if err := os.Rename(staging, destination); err != nil {
		return failure("destination_commit_failed", err.Error())
	}
	keepStaging = true

	tombstone := filepath.Join(filepath.Dir(source), "."+filepath.Base(source)+".imoogi-moved")
	if _, err := os.Lstat(tombstone); err == nil {
		_ = os.RemoveAll(destination)
		return failure("source_commit_failed", "source-side move marker already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = os.RemoveAll(destination)
		return failure("source_commit_failed", err.Error())
	}
	if err := os.Rename(source, tombstone); err != nil {
		_ = os.RemoveAll(destination)
		return failure("source_commit_failed", err.Error())
	}
	response := Response{OK: true, Source: source, Destination: destination, Files: files, Bytes: bytes}
	if err := os.RemoveAll(tombstone); err != nil {
		response.Warning = fmt.Sprintf("moved successfully; cleanup pending at %s: %v", tombstone, err)
	}
	return response
}

func validate(source, destination string) (string, string, error) {
	if !filepath.IsAbs(source) || !filepath.IsAbs(destination) {
		return "", "", errors.New("source and destination must be absolute paths")
	}
	source = filepath.Clean(source)
	destination = filepath.Clean(destination)
	linkInfo, err := os.Lstat(source)
	if err != nil {
		return "", "", fmt.Errorf("source directory is unavailable: %s", source)
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		return "", "", errors.New("source directory must not be a symbolic link")
	}
	info, err := os.Stat(source)
	if err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("source directory is unavailable: %s", source)
	}
	if _, err := os.Lstat(destination); err == nil {
		return "", "", fmt.Errorf("destination already exists: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(destination))
	if err != nil {
		return "", "", fmt.Errorf("destination parent is unavailable: %w", err)
	}
	realSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		return "", "", err
	}
	destination = filepath.Join(parent, filepath.Base(destination))
	if inside(realSource, destination) || inside(destination, realSource) {
		return "", "", errors.New("source and destination must not contain each other")
	}
	return realSource, destination, nil
}

func inside(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func copyTree(source, destination string) error {
	rootInfo, err := os.Stat(source)
	if err != nil {
		return err
	}
	if err := os.Chmod(destination, rootInfo.Mode().Perm()); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil || relative == "." {
			return err
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case entry.Type()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case entry.IsDir():
			return os.Mkdir(target, info.Mode().Perm())
		case entry.Type().IsRegular():
			return copyFile(path, target, info)
		default:
			return fmt.Errorf("unsupported file type: %s", path)
		}
	})
}

func copyFile(source, destination string, info fs.FileInfo) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	syncErr := out.Sync()
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chtimes(destination, info.ModTime(), info.ModTime())
}

func manifest(root string) ([]manifestEntry, int, int64, error) {
	var result []manifestEntry
	files := 0
	var bytes int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == "." {
			return err
		}
		switch {
		case entry.Type()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			result = append(result, manifestEntry{Path: relative, Kind: "symlink", Digest: link})
		case entry.IsDir():
			result = append(result, manifestEntry{Path: relative, Kind: "directory"})
		case entry.Type().IsRegular():
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			hash := sha256.New()
			size, copyErr := io.Copy(hash, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			files++
			bytes += size
			result = append(result, manifestEntry{Path: relative, Kind: "file", Digest: hex.EncodeToString(hash.Sum(nil)), Size: size})
		default:
			return fmt.Errorf("unsupported file type: %s", path)
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, files, bytes, err
}

func equalManifest(left, right []manifestEntry) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func failure(code, message string) Response {
	return Response{OK: false, Code: code, Error: message}
}
