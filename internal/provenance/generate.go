package provenance

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const SourcesSchema = "imoogi-vendor-sources/v1"

type Sources struct {
	Schema        string            `json:"schema"`
	Boundaries    []string          `json:"boundaries"`
	Roots         []string          `json:"roots"`
	Excludes      []Exclusion       `json:"excludes,omitempty"`
	ELPAOverrides map[string]Source `json:"elpa_overrides,omitempty"`
	Components    []SourceComponent `json:"components"`
}

type SourceComponent struct {
	Domain   string    `json:"domain"`
	ID       string    `json:"id"`
	Kind     string    `json:"kind"`
	Version  string    `json:"version"`
	Source   Source    `json:"source"`
	Platform *Platform `json:"platform,omitempty"`
	Workflow string    `json:"workflow"`
	Paths    []string  `json:"paths"`
}

var (
	packageRE       = regexp.MustCompile(`\(define-package\s+"([^"]+)"\s+"([^"]+)"`)
	packageURLRE    = regexp.MustCompile(`:url\s+"([^"]+)"`)
	packageCommitRE = regexp.MustCompile(`:commit\s+"([0-9a-f]{40})"`)
)

// Generate expands pinned source declarations into deterministic per-file manifests.
func Generate(repoRoot, sourcePath, indexPath string) error {
	var sources Sources
	if err := loadStrict(filepath.Join(repoRoot, sourcePath), &sources); err != nil {
		return err
	}
	if sources.Schema != SourcesSchema {
		return fmt.Errorf("unsupported sources schema %q", sources.Schema)
	}
	components := append([]SourceComponent{}, sources.Components...)
	elpa, err := discoverELPA(repoRoot, sources.ELPAOverrides)
	if err != nil {
		return err
	}
	components = append(components, elpa...)
	ignored := gitIgnored(repoRoot)
	domains := map[string][]Component{}
	for _, declaration := range components {
		files, err := expandFiles(repoRoot, declaration.Paths, sources.Excludes, ignored)
		if err != nil {
			return fmt.Errorf("component %s: %w", declaration.ID, err)
		}
		domains[declaration.Domain] = append(domains[declaration.Domain], Component{
			ID: declaration.ID, Kind: declaration.Kind, Version: declaration.Version,
			Source: declaration.Source, Platform: declaration.Platform,
			Workflow: declaration.Workflow, Files: files,
		})
	}
	domainNames := make([]string, 0, len(domains))
	for name := range domains {
		domainNames = append(domainNames, name)
	}
	sort.Strings(domainNames)
	manifests := make([]string, 0, len(domainNames))
	for _, name := range domainNames {
		path := filepath.ToSlash(filepath.Join("provenance", name+".json"))
		manifests = append(manifests, path)
		items := domains[name]
		sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
		if err := writeCanonical(filepath.Join(repoRoot, path), Domain{Schema: DomainSchema, Domain: name, Components: items}); err != nil {
			return err
		}
	}
	idx := Index{Schema: IndexSchema, Boundaries: sources.Boundaries, Roots: sources.Roots, Manifests: manifests, Excludes: sources.Excludes}
	return writeCanonical(filepath.Join(repoRoot, indexPath), idx)
}

// RecordGitSource replaces a declared component's revision with the commit
// observed by its build workflow.
func RecordGitSource(repoRoot, sourcePath, id, commit, ref string) error {
	if !commitRE.MatchString(commit) {
		return fmt.Errorf("git source requires a full 40-hex commit")
	}
	path := filepath.Join(repoRoot, sourcePath)
	var sources Sources
	if err := loadStrict(path, &sources); err != nil {
		return err
	}
	for i := range sources.Components {
		component := &sources.Components[i]
		if component.ID != id {
			continue
		}
		if component.Source.Type != "git" {
			return fmt.Errorf("component %s is not a git source", id)
		}
		component.Source.Commit = commit
		component.Source.Ref = ref
		return writeCanonical(path, sources)
	}
	return fmt.Errorf("component not found: %s", id)
}

func discoverELPA(root string, overrides map[string]Source) ([]SourceComponent, error) {
	dirs, err := os.ReadDir(filepath.Join(root, "vendor/elpa"))
	if err != nil {
		return nil, err
	}
	var out []SourceComponent
	for _, dir := range dirs {
		if !dir.IsDir() || dir.Name() == "archives" {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(root, "vendor/elpa", dir.Name(), "*-pkg.el"))
		if err != nil || len(matches) != 1 {
			return nil, fmt.Errorf("%s: expected exactly one *-pkg.el", dir.Name())
		}
		data, err := os.ReadFile(matches[0])
		if err != nil {
			return nil, err
		}
		m := packageRE.FindSubmatch(data)
		url := packageURLRE.FindSubmatch(data)
		commit := packageCommitRE.FindSubmatch(data)
		if len(m) == 0 || len(url) == 0 {
			return nil, fmt.Errorf("%s: package source metadata not found", dir.Name())
		}
		commitValue := ""
		if len(commit) > 0 {
			commitValue = string(commit[1])
		}
		source := Source{Type: "git", URL: string(url[1]), Commit: commitValue}
		if override, ok := overrides[string(m[1])]; ok {
			source = override
		}
		if source.Type != "git" || len(source.Commit) != 40 {
			return nil, fmt.Errorf("%s: package has no full upstream commit", dir.Name())
		}
		paths, err := elpaPaths(root, dir.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, SourceComponent{
			Domain: "elpa", ID: "elpa/" + string(m[1]), Kind: "elisp-source", Version: string(m[2]),
			Source:   source,
			Workflow: "emacs --batch -Q -l scripts/vendor.el", Paths: paths,
		})
	}
	return out, nil
}

func elpaPaths(root, directory string) ([]string, error) {
	paths := []string{"vendor/elpa/" + directory}
	signed := "vendor/elpa/" + directory + ".signed"
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(signed)))
	if err == nil {
		paths = append(paths, signed)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect package signature %s: %w", signed, err)
	}
	return paths, nil
}

func expandFiles(root string, paths []string, excludes []Exclusion, ignored map[string]struct{}) ([]File, error) {
	excluded := map[string]bool{}
	for _, ex := range excludes {
		excluded[ex.Path] = true
	}
	seen := map[string]bool{}
	var out []File
	for _, declared := range paths {
		if err := safeRelativePath(declared); err != nil {
			return nil, err
		}
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(declared)), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if excluded[rel] {
				return nil
			}
			if _, ok := ignored[rel]; ok {
				return nil
			}
			if seen[rel] {
				return fmt.Errorf("path included twice: %s", rel)
			}
			seen[rel] = true
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("unsupported file type: %s", rel)
			}
			hash, err := HashFile(path)
			if err != nil {
				return err
			}
			out = append(out, File{Path: rel, SHA256: hash, Size: info.Size()})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	if len(out) == 0 {
		return nil, fmt.Errorf("no files matched %s", strings.Join(paths, ", "))
	}
	return out, nil
}

func writeCanonical(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
