package provenance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type toolchainLock struct {
	CLIVersion string                   `json:"cli_version"`
	Bundle     string                   `json:"bundle"`
	Target     Platform                 `json:"target"`
	Components []toolchainLockComponent `json:"components"`
}

type toolchainLockComponent struct {
	Name            string `json:"name"`
	UpstreamVersion string `json:"upstream_version"`
	Revision        string `json:"revision"`
	Artifact        struct {
		Path      string `json:"path"`
		Size      int64  `json:"size"`
		SHA256    string `json:"sha256"`
		SourceURL string `json:"source_url"`
	} `json:"artifact"`
}

type toolchainDesired struct {
	CLIVersion string   `json:"cli_version"`
	Bundle     string   `json:"bundle"`
	Target     Platform `json:"target"`
	Components []struct {
		Name            string `json:"name"`
		UpstreamVersion string `json:"upstream_version"`
		Revision        string `json:"revision"`
	} `json:"components"`
}

type cliProvenance struct {
	CLIVersion string `json:"cli_version"`
	Builder    string `json:"builder"`
	Artifact   string `json:"artifact"`
	Size       int64  `json:"size"`
	SHA256     string `json:"sha256"`
}

func verifyCompatibility(repoRoot string, domains map[string]Domain) []error {
	var issues []error
	if domain, ok := domains["elpa"]; ok {
		issues = append(issues, verifyPackagesLock(repoRoot, domain)...)
	}
	if domain, ok := domains["toolchains"]; ok {
		issues = append(issues, verifyToolchainsLock(repoRoot, domain)...)
		issues = append(issues, verifyCLIProvenance(repoRoot, domain)...)
	}
	return issues
}

func verifyPackagesLock(root string, domain Domain) []error {
	data, err := os.ReadFile(filepath.Join(root, "packages.lock"))
	if err != nil {
		return []error{fmt.Errorf("read packages.lock: %w", err)}
	}
	locked := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && !strings.HasPrefix(line, ";;") {
			locked[fields[0]] = fields[1]
		}
	}
	var issues []error
	manifest := map[string]string{}
	for _, component := range domain.Components {
		if !strings.HasPrefix(component.ID, "elpa/") || strings.HasPrefix(component.ID, "elpa/archive-") {
			continue
		}
		name := strings.TrimPrefix(component.ID, "elpa/")
		manifest[name] = component.Version
		if locked[name] != component.Version {
			issues = append(issues, fmt.Errorf("packages.lock mismatch for %s: lock %q manifest %q", name, locked[name], component.Version))
		}
	}
	for name := range locked {
		if _, ok := manifest[name]; !ok {
			issues = append(issues, fmt.Errorf("packages.lock has unmanifested package: %s", name))
		}
	}
	return issues
}

func verifyToolchainsLock(root string, domain Domain) []error {
	data, err := os.ReadFile(filepath.Join(root, "toolchains.lock.json"))
	if err != nil {
		return []error{fmt.Errorf("read toolchains.lock.json: %w", err)}
	}
	var lock toolchainLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return []error{fmt.Errorf("decode toolchains.lock.json: %w", err)}
	}
	issues := verifyToolchainDesired(root, lock)
	byID := map[string]Component{}
	for _, component := range domain.Components {
		byID[component.ID] = component
	}
	seen := map[string]bool{}
	for _, item := range lock.Components {
		id := "toolchain/" + item.Name
		component, ok := byID[id]
		if !ok {
			issues = append(issues, fmt.Errorf("toolchains.lock has unmanifested component: %s", item.Name))
			continue
		}
		seen[id] = true
		if component.Platform == nil || *component.Platform != lock.Target {
			issues = append(issues, fmt.Errorf("toolchains.lock platform mismatch for %s", item.Name))
		}
		if component.Version != item.UpstreamVersion {
			issues = append(issues, fmt.Errorf("toolchains.lock version mismatch for %s", item.Name))
		}
		if item.Revision != "" && component.Source.Commit != item.Revision {
			issues = append(issues, fmt.Errorf("toolchains.lock revision mismatch for %s", item.Name))
		}
		artifactURL := component.Source.ArtifactURL
		if artifactURL == "" {
			artifactURL = component.Source.URL
		}
		if artifactURL != item.Artifact.SourceURL {
			issues = append(issues, fmt.Errorf("toolchains.lock source URL mismatch for %s", item.Name))
		}
		found := false
		for _, file := range component.Files {
			if file.Path == item.Artifact.Path {
				found = true
				if file.SHA256 != item.Artifact.SHA256 || file.Size != item.Artifact.Size {
					issues = append(issues, fmt.Errorf("toolchains.lock artifact mismatch for %s", item.Name))
				}
			}
		}
		if !found {
			issues = append(issues, fmt.Errorf("toolchains.lock artifact is not owned for %s: %s", item.Name, item.Artifact.Path))
		}
	}
	for id := range byID {
		if id == "toolchain/imoogi-cli" {
			if byID[id].Version != lock.CLIVersion {
				issues = append(issues, fmt.Errorf("toolchains.lock cli version mismatch"))
			}
			continue
		}
		if !seen[id] {
			issues = append(issues, fmt.Errorf("canonical toolchain missing from toolchains.lock: %s", id))
		}
	}
	return issues
}

func verifyToolchainDesired(root string, lock toolchainLock) []error {
	data, err := os.ReadFile(filepath.Join(root, "toolchains.json"))
	if err != nil {
		return []error{fmt.Errorf("read toolchains.json: %w", err)}
	}
	var desired toolchainDesired
	if err := json.Unmarshal(data, &desired); err != nil {
		return []error{fmt.Errorf("decode toolchains.json: %w", err)}
	}
	var issues []error
	if desired.CLIVersion != lock.CLIVersion || desired.Bundle != lock.Bundle || desired.Target != lock.Target {
		issues = append(issues, fmt.Errorf("toolchains.json header does not match toolchains.lock.json"))
	}
	locked := map[string]struct{ version, revision string }{}
	for _, component := range lock.Components {
		locked[component.Name] = struct{ version, revision string }{component.UpstreamVersion, component.Revision}
	}
	for _, component := range desired.Components {
		value, ok := locked[component.Name]
		if !ok || value.version != component.UpstreamVersion || value.revision != component.Revision {
			issues = append(issues, fmt.Errorf("toolchains.json component does not match lock: %s", component.Name))
		}
		delete(locked, component.Name)
	}
	for name := range locked {
		issues = append(issues, fmt.Errorf("toolchains.lock.json component missing from desired input: %s", name))
	}
	return issues
}

func verifyCLIProvenance(root string, domain Domain) []error {
	var component *Component
	for i := range domain.Components {
		if domain.Components[i].ID == "toolchain/imoogi-cli" {
			component = &domain.Components[i]
			break
		}
	}
	if component == nil {
		return []error{fmt.Errorf("canonical toolchain is missing toolchain/imoogi-cli")}
	}
	var provenancePath string
	files := map[string]File{}
	for _, file := range component.Files {
		files[file.Path] = file
		if strings.HasSuffix(file.Path, ".provenance.json") {
			provenancePath = file.Path
		}
	}
	if provenancePath == "" {
		return []error{fmt.Errorf("toolchain/imoogi-cli has no provenance view")}
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(provenancePath)))
	if err != nil {
		return []error{fmt.Errorf("read %s: %w", provenancePath, err)}
	}
	var view cliProvenance
	if err := json.Unmarshal(data, &view); err != nil {
		return []error{fmt.Errorf("decode %s: %w", provenancePath, err)}
	}
	var issues []error
	if view.CLIVersion != component.Version {
		issues = append(issues, fmt.Errorf("cli provenance version mismatch"))
	}
	if component.Platform == nil || view.Builder != component.Platform.OS+"/"+component.Platform.Arch {
		issues = append(issues, fmt.Errorf("cli provenance builder mismatch"))
	}
	artifact, ok := files[view.Artifact]
	if !ok {
		issues = append(issues, fmt.Errorf("cli provenance artifact is not owned: %s", view.Artifact))
	} else if artifact.Size != view.Size || artifact.SHA256 != view.SHA256 {
		issues = append(issues, fmt.Errorf("cli provenance artifact mismatch"))
	}
	return issues
}
