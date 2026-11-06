package artifact

import (
	"archive/tar"
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

// ExtractZip verifies archivePath against expected, then extracts it into
// destination with the same discipline as ExtractTarGzip: clean contained
// paths, no symlinks or special files, explicit 0644/0755 modes, entry and
// size caps, and an all-or-nothing commit through a sibling staging dir.
//
// Zip entries are mapped onto tar headers so both formats share one entry
// validator — a second, zip-only rule set would drift from the tar one.
func ExtractZip(archivePath, destination string, expected Digest, limits Limits) (ExtractionResult, error) {
	if err := VerifyFile(archivePath, expected); err != nil {
		return ExtractionResult{}, err
	}
	limits = withDefaultLimits(limits)
	if err := validateDestination(destination); err != nil {
		return ExtractionResult{}, err
	}

	temp, err := os.MkdirTemp(filepath.Dir(destination), "."+filepath.Base(destination)+".extract-*")
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("create extraction staging: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(temp)
		}
	}()

	result, err := extractVerifiedZip(archivePath, temp, limits)
	if err != nil {
		return ExtractionResult{}, err
	}
	if err := commitStaging(temp, destination); err != nil {
		return ExtractionResult{}, err
	}
	committed = true
	return result, nil
}

func extractVerifiedZip(archivePath, root string, limits Limits) (ExtractionResult, error) {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("open zip archive: %w", err)
	}
	defer zr.Close()

	seen := map[string]entryKind{}
	var result ExtractionResult
	for _, file := range zr.File {
		result.Entries++
		if result.Entries > limits.MaxEntries {
			return ExtractionResult{}, fmt.Errorf("archive has too many entries: %d > %d", result.Entries, limits.MaxEntries)
		}
		hdr, err := zipEntryHeader(file)
		if err != nil {
			return ExtractionResult{}, err
		}
		if err := validateEntryHeader(hdr, seen, limits, &result, ExtractOptions{}); err != nil {
			return ExtractionResult{}, err
		}
		entryPath, err := containedPath(root, hdr.Name)
		if err != nil {
			return ExtractionResult{}, err
		}
		if hdr.Typeflag == tar.TypeDir {
			if err := mkdirNoSymlinkBelow(root, entryPath); err != nil {
				return ExtractionResult{}, fmt.Errorf("create directory %q: %w", hdr.Name, err)
			}
			continue
		}
		if err := mkdirNoSymlinkBelow(root, filepath.Dir(entryPath)); err != nil {
			return ExtractionResult{}, fmt.Errorf("create parent for %q: %w", hdr.Name, err)
		}
		if err := writeZipFile(file, entryPath, hdr.Size); err != nil {
			return ExtractionResult{}, fmt.Errorf("extract file %q: %w", hdr.Name, err)
		}
	}
	return result, nil
}

func zipEntryHeader(file *zip.File) (*tar.Header, error) {
	mode := file.Mode()
	hdr := &tar.Header{Name: file.Name}
	switch {
	case mode.IsDir():
		hdr.Typeflag = tar.TypeDir
	case mode&os.ModeSymlink != 0:
		hdr.Typeflag = tar.TypeSymlink
	case mode.IsRegular():
		if file.UncompressedSize64 > math.MaxInt64 {
			return nil, fmt.Errorf("archive entry %q size overflows", file.Name)
		}
		hdr.Typeflag = tar.TypeReg
		hdr.Size = int64(file.UncompressedSize64)
	default:
		// Named pipes, devices, sockets: kindForType rejects these.
		hdr.Typeflag = tar.TypeFifo
	}
	return hdr, nil
}

// writeZipFile copies exactly size bytes, then reads once more so a stream
// longer than its header, or one failing archive/zip's CRC check (which
// only fires at EOF), is rejected rather than silently truncated.
func writeZipFile(file *zip.File, destination string, size int64) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	if err := writeRegularFile(destination, rc, size); err != nil {
		return err
	}
	var extra [1]byte
	n, err := rc.Read(extra[:])
	if n > 0 {
		return errors.New("entry is larger than its declared size")
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
