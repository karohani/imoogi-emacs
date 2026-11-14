// Package npmbundle installs an npm package's full dependency closure from a
// committed package-lock.json and packs it into a deterministic tar.gz, so the
// same inputs always produce the same bytes and the lock can pin a SHA-256.
package npmbundle

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/karohani/imoogi-emacs/internal/artifact"
)

const packRoot = "node_modules"

// Pack writes root/node_modules as a tar.gz whose bytes depend only on the
// tree's paths, file contents and executable bits: entries are sorted, every
// header carries mtime 0, uid/gid 0 and no owner names, modes are 0755 or
// 0644, and the gzip header has no name or mtime. npm's hidden lockfile
// (node_modules/.package-lock.json) and .bin link directories are left out;
// any symlink or special file is an error.
//
// @MX:ANCHOR: [AUTO] the artifact digest the lock pins is the output of this function
// @MX:REASON: fetch writes the digest and setup rebuilds and compares against it; any change to entry order, headers or compression changes every recorded sha256
func Pack(root string, w io.Writer) error {
	type entry struct {
		name string
		path string
		dir  bool
		exec bool
		size int64
	}
	var entries []entry
	base := filepath.Join(root, packRoot)
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() && d.Name() == ".bin" {
			return filepath.SkipDir
		}
		if rel == packRoot+"/.package-lock.json" {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		switch {
		case info.IsDir():
			entries = append(entries, entry{name: rel + "/", path: path, dir: true})
		case info.Mode().IsRegular():
			entries = append(entries, entry{name: rel, path: path, exec: info.Mode().Perm()&0o111 != 0, size: info.Size()})
		default:
			return fmt.Errorf("unsupported entry in npm install: %s (%s)", rel, info.Mode().Type())
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return err
	}
	tw := tar.NewWriter(gz)
	epoch := time.Unix(0, 0)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, ModTime: epoch, Format: tar.FormatPAX}
		if e.dir {
			hdr.Typeflag, hdr.Mode = tar.TypeDir, 0o755
		} else {
			hdr.Typeflag, hdr.Mode, hdr.Size = tar.TypeReg, 0o644, e.size
			if e.exec {
				hdr.Mode = 0o755
			}
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("pack %s: %w", e.name, err)
		}
		if e.dir {
			continue
		}
		if err := copyFileTo(tw, e.path, e.size); err != nil {
			return fmt.Errorf("pack %s: %w", e.name, err)
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func copyFileTo(w io.Writer, path string, size int64) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(w, io.LimitReader(f, size+1))
	if err != nil {
		return err
	}
	if n != size {
		return fmt.Errorf("file changed while packing (%d bytes, expected %d)", n, size)
	}
	return nil
}

// DigestFile returns the size and SHA-256 of path.
func DigestFile(path string) (artifact.Digest, error) {
	f, err := os.Open(path)
	if err != nil {
		return artifact.Digest{}, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return artifact.Digest{}, err
	}
	return artifact.Digest{Size: n, SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}

func cleanRelative(path string) (string, bool) {
	path = strings.TrimPrefix(path, "./")
	clean := filepath.ToSlash(filepath.Clean(path))
	if path == "" || filepath.IsAbs(path) || clean != path || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(path, `\`) {
		return "", false
	}
	return clean, true
}
