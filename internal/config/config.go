package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DesiredSchema = "imoogi-toolchains-desired/v1"
	LockSchema    = "imoogi-toolchains-lock/v1"
	TargetOS      = "darwin"
	TargetArch    = "arm64"

	maxArtifactSize = 1 << 40
)

var (
	calverRE       = regexp.MustCompile(`^[0-9]{4}\.(0[1-9]|1[0-2])\.(0[1-9]|[12][0-9]|3[01])\.(0|[1-9][0-9]*)$`)
	shaRE          = regexp.MustCompile(`^[0-9a-f]{64}$`)
	versionTokenRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)
)

type Target struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type DesiredManifest struct {
	Schema        string             `json:"schema"`
	CLI           string             `json:"cli_version"`
	Bundle        string             `json:"bundle"`
	Target        Target             `json:"target"`
	Components    []DesiredComponent `json:"components"`
	RuntimePolicy RuntimePolicy      `json:"runtime_policy,omitempty"`
}

type DesiredComponent struct {
	Name            string            `json:"name"`
	Kind            string            `json:"kind"`
	Source          string            `json:"source"`
	UpstreamVersion string            `json:"upstream_version"`
	Revision        string            `json:"revision,omitempty"`
	Runtime         RuntimeConstraint `json:"runtime,omitempty"`
}

type RuntimePolicy struct {
	NodeVersion string `json:"node_version,omitempty"`
}

type ResolvedLock struct {
	Schema      string          `json:"schema"`
	CLI         string          `json:"cli_version"`
	Bundle      string          `json:"bundle"`
	Target      Target          `json:"target"`
	Components  []LockComponent `json:"components"`
	GeneratedBy string          `json:"generated_by,omitempty"`
}

type LockComponent struct {
	Name            string            `json:"name"`
	Kind            string            `json:"kind"`
	Source          string            `json:"source"`
	UpstreamVersion string            `json:"upstream_version"`
	Revision        string            `json:"revision,omitempty"`
	Target          Target            `json:"target"`
	Artifact        Artifact          `json:"artifact"`
	License         License           `json:"license"`
	Provenance      Provenance        `json:"provenance"`
	Install         []InstallEntry    `json:"install"`
	Runtime         RuntimeConstraint `json:"runtime,omitempty"`
	Probe           Probe             `json:"probe"`
}

type Artifact struct {
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
	SourceURL   string `json:"source_url"`
	RetrievedAt string `json:"retrieved_at"`
}

type License struct {
	Path   string `json:"path"`
	Notice string `json:"notice,omitempty"`
}

type Provenance struct {
	Builder   string `json:"builder"`
	Toolchain string `json:"toolchain"`
	Command   string `json:"command"`
}

type InstallEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type RuntimeConstraint struct {
	NodeEngine string `json:"node_engine,omitempty"`
}

type Probe struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

type Plan struct {
	CLI        string
	Bundle     string
	Target     Target
	Components []PlanComponent
}

type PlanComponent struct {
	Name            string
	Kind            string
	UpstreamVersion string
	ArtifactPath    string
	Size            int64
	SHA256          string
}

func LoadDesired(path string) (*DesiredManifest, error) {
	var manifest DesiredManifest
	if err := loadStrict(path, &manifest); err != nil {
		return nil, err
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func LoadLock(path string) (*ResolvedLock, error) {
	var lock ResolvedLock
	if err := loadStrict(path, &lock); err != nil {
		return nil, err
	}
	if err := lock.Validate(); err != nil {
		return nil, err
	}
	return &lock, nil
}

func loadStrict(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("%s: trailing JSON values", filepath.Base(path))
	}
	return nil
}

func (m DesiredManifest) Validate() error {
	if m.Schema != DesiredSchema {
		return fmt.Errorf("desired schema %q is unsupported", m.Schema)
	}
	if !validSemVer(m.CLI) {
		return fmt.Errorf("cli_version %q is not SemVer", m.CLI)
	}
	if !calverRE.MatchString(m.Bundle) {
		return fmt.Errorf("bundle %q is not CalVer YYYY.MM.DD.N", m.Bundle)
	}
	if err := validateTarget("desired target", m.Target); err != nil {
		return err
	}
	if m.RuntimePolicy.NodeVersion != "" {
		if _, err := parseVersion(strings.TrimPrefix(m.RuntimePolicy.NodeVersion, "v")); err != nil {
			return fmt.Errorf("runtime_policy node_version: %w", err)
		}
	}
	if len(m.Components) == 0 {
		return errors.New("desired components are empty")
	}
	seen := map[string]struct{}{}
	for _, component := range m.Components {
		if err := component.validate(); err != nil {
			return err
		}
		key := component.Name
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate desired component %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (c DesiredComponent) validate() error {
	if c.Name == "" {
		return errors.New("desired component name is empty")
	}
	if c.Kind == "" {
		return fmt.Errorf("desired component %q kind is empty", c.Name)
	}
	if c.Source == "" {
		return fmt.Errorf("desired component %q source is empty", c.Name)
	}
	if c.UpstreamVersion == "" {
		return fmt.Errorf("desired component %q upstream_version is empty", c.Name)
	}
	if err := validateUpstreamVersion(c.Name, c.UpstreamVersion); err != nil {
		return fmt.Errorf("desired component %q upstream_version: %w", c.Name, err)
	}
	if err := c.Runtime.validate(c.Name); err != nil {
		return err
	}
	return nil
}

func (l ResolvedLock) Validate() error {
	if l.Schema != LockSchema {
		return fmt.Errorf("lock schema %q is unsupported", l.Schema)
	}
	if !validSemVer(l.CLI) {
		return fmt.Errorf("lock cli_version %q is not SemVer", l.CLI)
	}
	if !calverRE.MatchString(l.Bundle) {
		return fmt.Errorf("lock bundle %q is not CalVer YYYY.MM.DD.N", l.Bundle)
	}
	if err := validateTarget("lock target", l.Target); err != nil {
		return err
	}
	if len(l.Components) == 0 {
		return errors.New("lock components are empty")
	}
	seen := map[string]struct{}{}
	for _, component := range l.Components {
		if err := component.validate(); err != nil {
			return err
		}
		if _, ok := seen[component.Name]; ok {
			return fmt.Errorf("duplicate lock component %q", component.Name)
		}
		seen[component.Name] = struct{}{}
	}
	if err := l.validateRuntimeCompatibility(); err != nil {
		return err
	}
	return nil
}

func (c LockComponent) validate() error {
	if c.Name == "" {
		return errors.New("lock component name is empty")
	}
	if c.Kind == "" {
		return fmt.Errorf("lock component %q kind is empty", c.Name)
	}
	if c.Source == "" {
		return fmt.Errorf("lock component %q source is empty", c.Name)
	}
	if c.UpstreamVersion == "" {
		return fmt.Errorf("lock component %q upstream_version is empty", c.Name)
	}
	if err := validateUpstreamVersion(c.Name, c.UpstreamVersion); err != nil {
		return fmt.Errorf("lock component %q upstream_version: %w", c.Name, err)
	}
	if err := validateTarget("lock component "+c.Name+" target", c.Target); err != nil {
		return err
	}
	if err := c.Artifact.validate(c.Name); err != nil {
		return err
	}
	if c.License.Path == "" {
		return fmt.Errorf("lock component %q license path is empty", c.Name)
	}
	if err := ValidateRepoPath(c.License.Path); err != nil {
		return fmt.Errorf("lock component %q license path: %w", c.Name, err)
	}
	if c.License.Notice != "" {
		if err := ValidateRepoPath(c.License.Notice); err != nil {
			return fmt.Errorf("lock component %q notice path: %w", c.Name, err)
		}
	}
	if c.Provenance.Builder == "" || c.Provenance.Toolchain == "" || c.Provenance.Command == "" {
		return fmt.Errorf("lock component %q provenance is incomplete", c.Name)
	}
	if len(c.Install) == 0 {
		return fmt.Errorf("lock component %q install surface is empty", c.Name)
	}
	installSeen := map[string]struct{}{}
	for _, entry := range c.Install {
		if err := entry.validate(c.Name); err != nil {
			return err
		}
		if _, ok := installSeen[entry.Path]; ok {
			return fmt.Errorf("lock component %q duplicate install path %q", c.Name, entry.Path)
		}
		installSeen[entry.Path] = struct{}{}
	}
	if err := c.Runtime.validate(c.Name); err != nil {
		return err
	}
	if c.Probe.Command == "" {
		return fmt.Errorf("lock component %q probe command is empty", c.Name)
	}
	if err := ValidateRepoPath(c.Probe.Command); err != nil {
		return fmt.Errorf("lock component %q probe command: %w", c.Name, err)
	}
	return nil
}

func (a Artifact) validate(component string) error {
	if err := ValidateRepoPath(a.Path); err != nil {
		return fmt.Errorf("lock component %q artifact path: %w", component, err)
	}
	if a.Size <= 0 || a.Size > maxArtifactSize {
		return fmt.Errorf("lock component %q artifact size %d is invalid", component, a.Size)
	}
	if !shaRE.MatchString(a.SHA256) {
		return fmt.Errorf("lock component %q artifact sha256 %q is not lowercase SHA-256", component, a.SHA256)
	}
	if a.SourceURL == "" {
		return fmt.Errorf("lock component %q source_url is empty", component)
	}
	if a.RetrievedAt == "" {
		return fmt.Errorf("lock component %q retrieved_at is empty", component)
	}
	if _, err := time.Parse(time.RFC3339, a.RetrievedAt); err != nil {
		return fmt.Errorf("lock component %q retrieved_at is not RFC3339: %w", component, err)
	}
	return nil
}

func (e InstallEntry) validate(component string) error {
	if e.Path == "" {
		return fmt.Errorf("lock component %q install path is empty", component)
	}
	if err := ValidateRepoPath(e.Path); err != nil {
		return fmt.Errorf("lock component %q install path: %w", component, err)
	}
	if e.Mode != "file" && e.Mode != "executable" && e.Mode != "directory" {
		return fmt.Errorf("lock component %q install mode %q is unsupported", component, e.Mode)
	}
	return nil
}

func (r RuntimeConstraint) validate(component string) error {
	if r.NodeEngine == "" {
		return nil
	}
	if _, err := parseEngineRange(r.NodeEngine); err != nil {
		return fmt.Errorf("component %q node_engine: %w", component, err)
	}
	return nil
}

func (l ResolvedLock) validateRuntimeCompatibility() error {
	nodeVersion := ""
	for _, component := range l.Components {
		if component.Name == "node" || component.Kind == "node-runtime" {
			nodeVersion = strings.TrimPrefix(component.UpstreamVersion, "v")
			break
		}
	}
	for _, component := range l.Components {
		if component.Runtime.NodeEngine == "" {
			continue
		}
		if nodeVersion == "" {
			return fmt.Errorf("component %q requires node_engine %q but lock has no node runtime", component.Name, component.Runtime.NodeEngine)
		}
		ok, err := CheckNodeEngine(nodeVersion, component.Runtime.NodeEngine)
		if err != nil {
			return fmt.Errorf("component %q node_engine: %w", component.Name, err)
		}
		if !ok {
			return fmt.Errorf("component %q requires node_engine %q but node is %s", component.Name, component.Runtime.NodeEngine, nodeVersion)
		}
	}
	return nil
}

func validateTarget(label string, target Target) error {
	if target.OS != TargetOS || target.Arch != TargetArch {
		return fmt.Errorf("%s %s/%s is unsupported; want %s/%s", label, target.OS, target.Arch, TargetOS, TargetArch)
	}
	return nil
}

func validateUpstreamVersion(component, version string) error {
	if !versionTokenRE.MatchString(version) || strings.Contains(version, "..") {
		return fmt.Errorf("%q is not a path-safe version token", version)
	}
	switch component {
	case "gopls", "node":
		if !strings.HasPrefix(version, "v") || !validSemVer(strings.TrimPrefix(version, "v")) {
			return fmt.Errorf("%q must be v-prefixed SemVer", version)
		}
	case "typescript", "typescript-language-server":
		if !validSemVer(version) {
			return fmt.Errorf("%q must be SemVer", version)
		}
	}
	return nil
}

// ValidateRepoPath requires a clean slash-separated path below the repository root.
func ValidateRepoPath(path string) error {
	if path == "" {
		return errors.New("empty path")
	}
	if filepath.IsAbs(path) || strings.HasPrefix(path, "/") {
		return fmt.Errorf("%q is absolute", path)
	}
	if strings.Contains(path, `\`) {
		return fmt.Errorf("%q contains backslash", path)
	}
	if strings.Contains(path, ":") {
		return fmt.Errorf("%q contains drive or scheme separator", path)
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean != path {
		return fmt.Errorf("%q is not clean repository-relative path", path)
	}
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("%q escapes repository", path)
	}
	return nil
}

func (m DesiredManifest) ValidateLock(lock ResolvedLock) error {
	if m.CLI != lock.CLI {
		return fmt.Errorf("cli_version mismatch: desired %s lock %s", m.CLI, lock.CLI)
	}
	if m.Bundle != lock.Bundle {
		return fmt.Errorf("bundle mismatch: desired %s lock %s", m.Bundle, lock.Bundle)
	}
	if m.Target != lock.Target {
		return fmt.Errorf("target mismatch: desired %s/%s lock %s/%s", m.Target.OS, m.Target.Arch, lock.Target.OS, lock.Target.Arch)
	}
	if m.RuntimePolicy.NodeVersion != "" {
		nodeVersion := ""
		for _, component := range lock.Components {
			if component.Name == "node" || component.Kind == "node-runtime" {
				nodeVersion = strings.TrimPrefix(component.UpstreamVersion, "v")
				break
			}
		}
		if nodeVersion == "" {
			return fmt.Errorf("runtime_policy node_version %s has no locked node component", m.RuntimePolicy.NodeVersion)
		}
		if strings.TrimPrefix(m.RuntimePolicy.NodeVersion, "v") != nodeVersion {
			return fmt.Errorf("runtime_policy node_version %s differs from locked node %s", m.RuntimePolicy.NodeVersion, nodeVersion)
		}
	}
	lockByName := map[string]LockComponent{}
	for _, component := range lock.Components {
		lockByName[component.Name] = component
	}
	for _, desired := range m.Components {
		locked, ok := lockByName[desired.Name]
		if !ok {
			return fmt.Errorf("desired component %q is missing from lock", desired.Name)
		}
		if desired.Kind != locked.Kind || desired.Source != locked.Source || desired.UpstreamVersion != locked.UpstreamVersion || desired.Revision != locked.Revision {
			return fmt.Errorf("component %q identity differs between desired manifest and lock", desired.Name)
		}
		if desired.Runtime != locked.Runtime {
			return fmt.Errorf("component %q runtime constraints differ between desired manifest and lock", desired.Name)
		}
		delete(lockByName, desired.Name)
	}
	if len(lockByName) > 0 {
		names := make([]string, 0, len(lockByName))
		for name := range lockByName {
			names = append(names, name)
		}
		sort.Strings(names)
		return fmt.Errorf("lock has components not present in desired manifest: %s", strings.Join(names, ", "))
	}
	return nil
}

func DetectDrift(previous, next ResolvedLock) error {
	previousByIdentity := map[string]LockComponent{}
	for _, component := range previous.Components {
		previousByIdentity[identityKey(component)] = component
	}
	for _, component := range next.Components {
		if old, ok := previousByIdentity[identityKey(component)]; ok {
			if old.Artifact.SHA256 != component.Artifact.SHA256 || old.Artifact.Size != component.Artifact.Size {
				return fmt.Errorf("component %q has same identity but different artifact digest or size", component.Name)
			}
		}
	}
	return nil
}

func identityKey(c LockComponent) string {
	return strings.Join([]string{c.Name, c.Kind, c.Source, c.UpstreamVersion, c.Revision, c.Target.OS, c.Target.Arch}, "\x00")
}

func BuildPlan(manifest DesiredManifest, lock ResolvedLock) (Plan, error) {
	if err := manifest.Validate(); err != nil {
		return Plan{}, err
	}
	if err := lock.Validate(); err != nil {
		return Plan{}, err
	}
	if err := manifest.ValidateLock(lock); err != nil {
		return Plan{}, err
	}
	components := make([]PlanComponent, 0, len(lock.Components))
	for _, component := range lock.Components {
		components = append(components, PlanComponent{
			Name:            component.Name,
			Kind:            component.Kind,
			UpstreamVersion: component.UpstreamVersion,
			ArtifactPath:    component.Artifact.Path,
			Size:            component.Artifact.Size,
			SHA256:          component.Artifact.SHA256,
		})
	}
	sort.Slice(components, func(i, j int) bool {
		return components[i].Name < components[j].Name
	})
	return Plan{
		CLI:        lock.CLI,
		Bundle:     lock.Bundle,
		Target:     lock.Target,
		Components: components,
	}, nil
}

func WriteLock(w io.Writer, lock ResolvedLock) error {
	if err := lock.Validate(); err != nil {
		return err
	}
	lock.Components = append([]LockComponent(nil), lock.Components...)
	sort.Slice(lock.Components, func(i, j int) bool {
		return lock.Components[i].Name < lock.Components[j].Name
	})
	for i := range lock.Components {
		lock.Components[i].Install = append([]InstallEntry(nil), lock.Components[i].Install...)
		sort.Slice(lock.Components[i].Install, func(a, b int) bool {
			return lock.Components[i].Install[a].Path < lock.Components[i].Install[b].Path
		})
	}
	return writeDeterministic(w, lock)
}

func WriteDesired(w io.Writer, manifest DesiredManifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	manifest.Components = append([]DesiredComponent(nil), manifest.Components...)
	sort.Slice(manifest.Components, func(i, j int) bool {
		return manifest.Components[i].Name < manifest.Components[j].Name
	})
	return writeDeterministic(w, manifest)
}

func writeDeterministic(w io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

func validSemVer(version string) bool {
	mainAndBuild := strings.Split(version, "+")
	if len(mainAndBuild) > 2 {
		return false
	}
	if len(mainAndBuild) == 2 && !validIdentifierList(mainAndBuild[1], false) {
		return false
	}

	mainAndPre := strings.Split(mainAndBuild[0], "-")
	if len(mainAndPre) > 2 {
		return false
	}
	if len(mainAndPre) == 2 && !validIdentifierList(mainAndPre[1], true) {
		return false
	}

	core := strings.Split(mainAndPre[0], ".")
	if len(core) != 3 {
		return false
	}
	for _, part := range core {
		if !validNumericIdentifier(part, true) {
			return false
		}
	}
	return true
}

func validIdentifierList(list string, rejectNumericLeadingZero bool) bool {
	if list == "" {
		return false
	}
	identifiers := strings.Split(list, ".")
	for _, identifier := range identifiers {
		if identifier == "" {
			return false
		}
		allDigits := true
		for _, r := range identifier {
			isDigit := r >= '0' && r <= '9'
			isAlpha := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
			if !isDigit && !isAlpha && r != '-' {
				return false
			}
			if !isDigit {
				allDigits = false
			}
		}
		if rejectNumericLeadingZero && allDigits && !validNumericIdentifier(identifier, true) {
			return false
		}
	}
	return true
}

func validNumericIdentifier(identifier string, rejectLeadingZero bool) bool {
	if identifier == "" {
		return false
	}
	for _, r := range identifier {
		if r < '0' || r > '9' {
			return false
		}
	}
	return !rejectLeadingZero || len(identifier) == 1 || identifier[0] != '0'
}

func CheckNodeEngine(version, engine string) (bool, error) {
	version = strings.TrimPrefix(version, "v")
	v, err := parseVersion(version)
	if err != nil {
		return false, err
	}
	rangeChecks, err := parseEngineRange(engine)
	if err != nil {
		return false, err
	}
	for _, check := range rangeChecks {
		if !check.matches(v) {
			return false, nil
		}
	}
	return true, nil
}

type version3 struct {
	major int
	minor int
	patch int
}

type engineCheck struct {
	op      string
	version version3
}

func parseEngineRange(engine string) ([]engineCheck, error) {
	parts := strings.Fields(engine)
	if len(parts) == 0 {
		return nil, errors.New("empty range")
	}
	checks := make([]engineCheck, 0, len(parts))
	for _, part := range parts {
		op := ""
		for _, candidate := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(part, candidate) {
				op = candidate
				part = strings.TrimPrefix(part, candidate)
				break
			}
		}
		if op == "" {
			return nil, fmt.Errorf("unsupported comparator %q", part)
		}
		v, err := parseVersion(strings.TrimPrefix(part, "v"))
		if err != nil {
			return nil, err
		}
		checks = append(checks, engineCheck{op: op, version: v})
	}
	return checks, nil
}

func parseVersion(version string) (version3, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return version3{}, fmt.Errorf("version %q must have major.minor.patch", version)
	}
	nums := [3]int{}
	for i := 0; i < 3; i++ {
		part := parts[i]
		if idx := strings.IndexAny(part, "-+"); idx >= 0 {
			part = part[:idx]
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return version3{}, fmt.Errorf("version %q is invalid", version)
		}
		nums[i] = n
	}
	return version3{major: nums[0], minor: nums[1], patch: nums[2]}, nil
}

func (c engineCheck) matches(v version3) bool {
	cmp := compareVersion(v, c.version)
	switch c.op {
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case "=":
		return cmp == 0
	}
	return false
}

func compareVersion(a, b version3) int {
	if a.major != b.major {
		return a.major - b.major
	}
	if a.minor != b.minor {
		return a.minor - b.minor
	}
	return a.patch - b.patch
}

func rejectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	return scanValue(dec)
}

func scanValue(dec *json.Decoder) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		keys := map[string]struct{}{}
		for dec.More() {
			token, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := token.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if _, ok := keys[key]; ok {
				return fmt.Errorf("duplicate key %q", key)
			}
			keys[key] = struct{}{}
			if err := scanValue(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token()
		return err
	case '[':
		for dec.More() {
			if err := scanValue(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token()
		return err
	}
	return nil
}
