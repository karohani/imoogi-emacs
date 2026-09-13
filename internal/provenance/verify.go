package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	shaRE    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	commitRE = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

func Verify(repoRoot, indexPath string) []error {
	var idx Index
	if err := loadStrict(filepath.Join(repoRoot, indexPath), &idx); err != nil {
		return []error{err}
	}
	issues := validateIndex(idx)
	owned := map[string]string{}
	domains := map[string]Domain{}
	excluded := map[string]struct{}{}
	exclusionsSeen := map[string]bool{}
	insideRoots := map[string]bool{}
	for _, item := range idx.Excludes {
		excluded[filepath.ToSlash(filepath.Clean(item.Path))] = struct{}{}
	}
	for _, manifestPath := range idx.Manifests {
		var domain Domain
		if err := loadStrict(filepath.Join(repoRoot, manifestPath), &domain); err != nil {
			issues = append(issues, err)
			continue
		}
		if _, exists := domains[domain.Domain]; exists {
			issues = append(issues, fmt.Errorf("duplicate domain manifest: %s", domain.Domain))
		}
		domains[domain.Domain] = domain
		issues = append(issues, validateDomain(repoRoot, domain, owned)...)
	}
	issues = append(issues, verifyCompatibility(repoRoot, domains)...)
	for _, root := range idx.Roots {
		absRoot := filepath.Join(repoRoot, filepath.FromSlash(root))
		err := filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				issues = append(issues, fmt.Errorf("walk %s: %w", root, walkErr))
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				issues = append(issues, err)
				return nil
			}
			rel = filepath.ToSlash(rel)
			insideRoots[rel] = true
			if _, ok := excluded[rel]; ok {
				exclusionsSeen[rel] = true
				return nil
			}
			if _, ok := owned[rel]; !ok {
				issues = append(issues, fmt.Errorf("unowned file: %s", rel))
			}
			return nil
		})
		if err != nil {
			issues = append(issues, fmt.Errorf("walk root %s: %w", root, err))
		}
	}
	for _, boundary := range idx.Boundaries {
		err := filepath.WalkDir(filepath.Join(repoRoot, filepath.FromSlash(boundary)), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				issues = append(issues, fmt.Errorf("walk boundary %s: %w", boundary, walkErr))
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if _, ok := excluded[rel]; ok {
				exclusionsSeen[rel] = true
				return nil
			}
			if !insideRoots[rel] {
				issues = append(issues, fmt.Errorf("file outside declared roots: %s", rel))
			}
			return nil
		})
		if err != nil {
			issues = append(issues, fmt.Errorf("walk boundary %s: %w", boundary, err))
		}
	}
	for path := range excluded {
		if !exclusionsSeen[path] {
			issues = append(issues, fmt.Errorf("exclusion is stale or outside coverage roots: %s", path))
		}
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].Error() < issues[j].Error() })
	return issues
}

func validateIndex(idx Index) []error {
	var issues []error
	if idx.Schema != IndexSchema {
		issues = append(issues, fmt.Errorf("unsupported index schema %q", idx.Schema))
	}
	if len(idx.Boundaries) == 0 || len(idx.Roots) == 0 || len(idx.Manifests) == 0 {
		issues = append(issues, fmt.Errorf("index boundaries, roots, and manifests must not be empty"))
	}
	seenPaths := map[string]string{}
	for _, p := range idx.Boundaries {
		if err := safeRelativePath(p); err != nil {
			issues = append(issues, err)
		}
		if old, ok := seenPaths[p]; ok {
			issues = append(issues, fmt.Errorf("duplicate index path %s (%s and boundary)", p, old))
		}
		seenPaths[p] = "boundary"
	}
	for _, p := range idx.Roots {
		if err := safeRelativePath(p); err != nil {
			issues = append(issues, err)
		}
		if old, ok := seenPaths[p]; ok {
			issues = append(issues, fmt.Errorf("duplicate index path %s (%s and root)", p, old))
		}
		seenPaths[p] = "root"
	}
	for _, p := range idx.Manifests {
		if err := safeRelativePath(p); err != nil {
			issues = append(issues, err)
		}
		if old, ok := seenPaths[p]; ok {
			issues = append(issues, fmt.Errorf("duplicate index path %s (%s and manifest)", p, old))
		}
		seenPaths[p] = "manifest"
	}
	seenExclusions := map[string]string{}
	for _, ex := range idx.Excludes {
		if err := safeRelativePath(ex.Path); err != nil {
			issues = append(issues, err)
		}
		if ex.Reason == "" {
			issues = append(issues, fmt.Errorf("exclusion %s has no reason", ex.Path))
		}
		if strings.ContainsAny(ex.Path, "*?[") {
			issues = append(issues, fmt.Errorf("exclusion %s must be an exact path", ex.Path))
		}
		if old, ok := seenExclusions[ex.Path]; ok {
			issues = append(issues, fmt.Errorf("duplicate exclusion %s (%s)", ex.Path, old))
		}
		seenExclusions[ex.Path] = ex.Reason
	}
	return issues
}

func validateDomain(root string, domain Domain, owned map[string]string) []error {
	var issues []error
	if domain.Schema != DomainSchema {
		issues = append(issues, fmt.Errorf("domain %s: unsupported schema %q", domain.Domain, domain.Schema))
	}
	if domain.Domain == "" || len(domain.Components) == 0 {
		issues = append(issues, fmt.Errorf("domain name and components must not be empty"))
	}
	ids := map[string]struct{}{}
	for _, component := range domain.Components {
		prefix := "component " + component.ID
		if component.ID == "" || component.Kind == "" || component.Version == "" || component.Workflow == "" {
			issues = append(issues, fmt.Errorf("%s: id, kind, version, and workflow are required", prefix))
		}
		if _, ok := ids[component.ID]; ok {
			issues = append(issues, fmt.Errorf("duplicate component id: %s", component.ID))
		}
		ids[component.ID] = struct{}{}
		if component.Source.URL == "" {
			issues = append(issues, fmt.Errorf("%s: source URL is required", prefix))
		}
		switch component.Source.Type {
		case "git":
			if !commitRE.MatchString(component.Source.Commit) {
				issues = append(issues, fmt.Errorf("%s: git source requires a full 40-hex commit", prefix))
			}
		case "archive", "download", "repository":
		default:
			issues = append(issues, fmt.Errorf("%s: unsupported source type %q", prefix, component.Source.Type))
		}
		if requiresPlatform(component.Kind) && (component.Platform == nil || component.Platform.OS == "" || component.Platform.Arch == "") {
			issues = append(issues, fmt.Errorf("%s: platform os/arch is required", prefix))
		}
		if len(component.Files) == 0 {
			issues = append(issues, fmt.Errorf("%s: files must not be empty", prefix))
		}
		for _, file := range component.Files {
			if err := safeRelativePath(file.Path); err != nil {
				issues = append(issues, fmt.Errorf("%s: %w", prefix, err))
				continue
			}
			if owner, ok := owned[file.Path]; ok {
				issues = append(issues, fmt.Errorf("duplicate ownership: %s by %s and %s", file.Path, owner, component.ID))
			} else {
				owned[file.Path] = component.ID
			}
			if !shaRE.MatchString(file.SHA256) {
				issues = append(issues, fmt.Errorf("%s: malformed sha256 for %s", prefix, file.Path))
				continue
			}
			info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(file.Path)))
			if err != nil {
				issues = append(issues, fmt.Errorf("%s: missing file %s", prefix, file.Path))
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				issues = append(issues, fmt.Errorf("%s: symlink is not allowed: %s", prefix, file.Path))
				continue
			}
			if !info.Mode().IsRegular() {
				issues = append(issues, fmt.Errorf("%s: not a regular file: %s", prefix, file.Path))
				continue
			}
			if info.Size() != file.Size {
				issues = append(issues, fmt.Errorf("%s: size mismatch for %s: expected %d actual %d", prefix, file.Path, file.Size, info.Size()))
			}
			actual, err := HashFile(filepath.Join(root, filepath.FromSlash(file.Path)))
			if err != nil {
				issues = append(issues, fmt.Errorf("%s: hash %s: %w", prefix, file.Path, err))
			} else if actual != file.SHA256 {
				issues = append(issues, fmt.Errorf("%s: sha256 mismatch for %s: expected %s actual %s", prefix, file.Path, file.SHA256, actual))
			}
		}
	}
	return issues
}

func safeRelativePath(path string) error {
	if path == "" || filepath.IsAbs(path) || filepath.Clean(path) == "." || strings.Contains(path, "\\") {
		return fmt.Errorf("unsafe repository-relative path: %q", path)
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean != path || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("unsafe repository-relative path: %q", path)
	}
	return nil
}

func requiresPlatform(kind string) bool {
	return strings.Contains(kind, "binary") || strings.Contains(kind, "native") || strings.Contains(kind, "grammar")
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
