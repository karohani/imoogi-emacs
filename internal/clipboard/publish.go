package clipboard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

const assetIndexFilename = ".imoogi-asset-index.json"

type assetCacheEntry struct {
	Size    int64  `json:"size"`
	ModTime int64  `json:"mtime_ns"`
	SHA256  string `json:"sha256"`
}

type assetCache struct {
	Version int                        `json:"version"`
	Files   map[string]assetCacheEntry `json:"files"`
}

type Publisher struct {
	Limits          Limits
	ContainmentRoot string
}

type PublishResult struct {
	Assets       []Asset
	CreatedFiles []string
	Orphans      []string
}

func (p Publisher) Publish(paths []string, root string) (PublishResult, error) {
	limits := p.Limits
	if limits.MaxFiles == 0 {
		limits = DefaultLimits()
	}
	sources, err := preflightSources(paths, limits)
	if err != nil {
		return PublishResult{}, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return PublishResult{}, fmt.Errorf("create asset root: %w", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return PublishResult{}, fmt.Errorf("canonical asset root: %w", err)
	}
	root = canonicalRoot
	if p.ContainmentRoot != "" {
		containment, err := canonicalForContainment(p.ContainmentRoot)
		if err != nil || (!pathWithin(containment, root) && containment != root) {
			return PublishResult{}, fmt.Errorf("%s: asset root is outside its owner", CodePathEscape)
		}
	}
	cache, err := loadAssetCache(root)
	if err != nil {
		return PublishResult{}, fmt.Errorf("load asset cache: %w", err)
	}
	var result PublishResult
	for _, source := range sources {
		asset, created, err := publishOne(source, root, &cache)
		if err != nil {
			result.Orphans = compensate(result.CreatedFiles)
			if len(result.Orphans) != 0 {
				return result, fmt.Errorf("publish %q: %w: %s", source.path, err, CodeNeedsReconciliation)
			}
			return PublishResult{}, fmt.Errorf("publish %q: %w", source.path, err)
		}
		result.Assets = append(result.Assets, asset)
		if created != "" {
			result.CreatedFiles = append(result.CreatedFiles, created)
		}
	}
	if err := saveAssetCache(root, cache); err != nil {
		result.Orphans = compensate(result.CreatedFiles)
		if len(result.Orphans) != 0 {
			return result, fmt.Errorf("save asset cache: %w: %s", err, CodeNeedsReconciliation)
		}
		return PublishResult{}, fmt.Errorf("save asset cache: %w", err)
	}
	return result, nil
}

type sourceFile struct {
	path string
	size int64
}

func preflightSources(paths []string, limits Limits) ([]sourceFile, error) {
	if len(paths) == 0 {
		return nil, errors.New("no source paths")
	}
	if len(paths) > limits.MaxFiles {
		return nil, fmt.Errorf("%s: %d files exceeds %d", CodeLimitExceeded, len(paths), limits.MaxFiles)
	}
	var total int64
	sources := make([]sourceFile, 0, len(paths))
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", path, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%s: %q", CodeDirectoryRejected, path)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s: %q", CodeNonRegularRejected, path)
		}
		if info.Size() > limits.MaxFileBytes || total > limits.MaxBatchBytes-info.Size() {
			return nil, fmt.Errorf("%s: %q", CodeLimitExceeded, path)
		}
		total += info.Size()
		sources = append(sources, sourceFile{path: path, size: info.Size()})
	}
	return sources, nil
}

func publishOne(source sourceFile, root string, cache *assetCache) (Asset, string, error) {
	input, err := os.Open(source.path)
	if err != nil {
		return Asset{}, "", err
	}
	defer input.Close()

	// Hash the source before allocating a destination so an identical asset
	// already in the asset root can be linked without creating a duplicate.
	sourceHash := sha256.New()
	if _, err := io.Copy(sourceHash, input); err != nil {
		return Asset{}, "", err
	}
	if _, err := input.Seek(0, io.SeekStart); err != nil {
		return Asset{}, "", err
	}
	hashHex := hex.EncodeToString(sourceHash.Sum(nil))
	if existing, err := findExistingAsset(root, source.size, hashHex, cache); err != nil {
		return Asset{}, "", err
	} else if existing != "" {
		id, err := randomHex(16)
		if err != nil {
			return Asset{}, "", err
		}
		return Asset{
			ID:           id,
			Source:       source.path,
			Path:         existing,
			RelativePath: filepath.Base(existing),
			MIME:         mime.TypeByExtension(strings.ToLower(filepath.Ext(existing))),
			SHA256:       hashHex,
			Size:         source.size,
		}, "", nil
	}

	base := sanitizedFilename(filepath.Base(source.path))
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for sequence := 0; sequence < 10000; sequence++ {
		name := base
		if sequence > 0 {
			name = fmt.Sprintf("%s-%d%s", stem, sequence, ext)
		}
		destination := filepath.Join(root, name)
		output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			if existingHash, hashErr := fileSHA256(destination); hashErr == nil && existingHash == hashHex {
				id, idErr := randomHex(16)
				if idErr != nil {
					return Asset{}, "", idErr
				}
				cache.record(destination, source.size, hashHex)
				return Asset{ID: id, Source: source.path, Path: destination,
					RelativePath: filepath.Base(destination),
					MIME:         mime.TypeByExtension(strings.ToLower(ext)),
					SHA256:       hashHex, Size: source.size}, "", nil
			}
			continue
		}
		if err != nil {
			return Asset{}, "", err
		}
		written, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil || closeErr != nil || written != source.size {
			_ = os.Remove(destination)
			if copyErr != nil {
				return Asset{}, "", copyErr
			}
			if closeErr != nil {
				return Asset{}, "", closeErr
			}
			return Asset{}, "", fmt.Errorf("short copy: %d != %d", written, source.size)
		}
		id, err := randomHex(16)
		if err != nil {
			_ = os.Remove(destination)
			return Asset{}, "", err
		}
		cache.record(destination, written, hashHex)
		return Asset{
			ID:           id,
			Source:       source.path,
			Path:         destination,
			RelativePath: name,
			MIME:         mime.TypeByExtension(strings.ToLower(ext)),
			SHA256:       hashHex,
			Size:         written,
		}, destination, nil
	}
	return Asset{}, "", errors.New("cannot allocate collision-free destination")
}

func findExistingAsset(root string, size int64, wantHash string, cache *assetCache) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.Name() == assetIndexFilename {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() || info.Size() != size {
			continue
		}
		path := filepath.Join(root, entry.Name())
		cached, ok := cache.Files[entry.Name()]
		var existingHash string
		if ok && cached.Size == info.Size() && cached.ModTime == info.ModTime().UnixNano() {
			existingHash = cached.SHA256
		} else if existingHash, err = fileSHA256(path); err != nil {
			return "", err
		} else {
			cache.Files[entry.Name()] = assetCacheEntry{Size: info.Size(), ModTime: info.ModTime().UnixNano(), SHA256: existingHash}
		}
		if existingHash == wantHash {
			return path, nil
		}
	}
	return "", nil
}

func (c *assetCache) record(path string, size int64, hash string) {
	if c.Files == nil {
		c.Files = make(map[string]assetCacheEntry)
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	c.Files[filepath.Base(path)] = assetCacheEntry{Size: size, ModTime: info.ModTime().UnixNano(), SHA256: hash}
}

func loadAssetCache(root string) (assetCache, error) {
	cache := assetCache{Version: 1, Files: make(map[string]assetCacheEntry)}
	data, err := os.ReadFile(filepath.Join(root, assetIndexFilename))
	if errors.Is(err, os.ErrNotExist) {
		return cache, nil
	}
	if err != nil {
		return cache, err
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return assetCache{Version: 1, Files: make(map[string]assetCacheEntry)}, nil
	}
	if cache.Files == nil {
		cache.Files = make(map[string]assetCacheEntry)
	}
	return cache, nil
}

func saveAssetCache(root string, cache assetCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(root, assetIndexFilename+".tmp-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temporaryName, 0o644); err != nil {
		return err
	}
	return os.Rename(temporaryName, filepath.Join(root, assetIndexFilename))
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func compensate(paths []string) []string {
	var orphans []string
	for index := len(paths) - 1; index >= 0; index-- {
		if err := os.Remove(paths[index]); err != nil && !errors.Is(err, os.ErrNotExist) {
			orphans = append(orphans, paths[index])
		}
	}
	return orphans
}

func sanitizedFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 || r < 0x20 {
			return '-'
		}
		return r
	}, name)
	if name == "" || name == "." || name == ".." {
		return "asset"
	}
	return name
}
