package clipboard

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
)

// DefaultOrphanGrace keeps recently written assets out of the orphan list, so
// a paste whose link has not been saved yet is never offered for deletion.
const DefaultOrphanGrace = 7 * 24 * time.Hour

// Orphan is a published asset that no scanned document references.
type Orphan struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mtime"`
}

// OrphanOptions tunes FindOrphans. Protected lists files the caller still
// references from unsaved buffers; they are never reported.
type OrphanOptions struct {
	Grace     time.Duration
	Now       func() time.Time
	Protected []string
}

var orphanDocumentExtensions = map[string]bool{
	".org": true, ".org_archive": true, ".md": true, ".markdown": true,
}

// orphanLinkPattern mirrors imoogi-clipboard--referenced-assets in
// modules/org/28-clipboard.el: an Org file link or a Markdown link target.
var orphanLinkPattern = regexp.MustCompile(`\[\[file:([^\]]+)\]|\]\(([^)]*)\)`)

var markdownLinkTitle = regexp.MustCompile(`\s+("[^"]*"|'[^']*')\s*$`)

// trashAsset is the production trash. Tests replace it; an asset is never
// removed with os.Remove.
var trashAsset = finderTrash

type orphanCandidate struct {
	path       string
	info       os.FileInfo
	folded     string
	needles    []string
	referenced bool
}

// FindOrphans lists the assets directly inside owner's asset root that no
// document references. It only reports a file when it can prove the absence
// of every reference it knows how to look for: any unreadable document,
// manifest, or protected path aborts the scan with an error instead.
//
// Project-notes owners scan every Org and Markdown file under the project
// root; standalone owners scan only their own document. Links are matched by
// file identity, and a file whose name appears anywhere in a document, in any
// of several encodings, also counts as referenced. Files younger than the
// grace window, protected paths, and assets of unfinished transactions in
// store are excluded.
func FindOrphans(owner ResolvedOwner, store Store, opts OrphanOptions) ([]Orphan, error) {
	if owner.Owner.Kind != OwnerProjectNotes && owner.Owner.Kind != OwnerStandalone {
		return nil, fmt.Errorf("orphan scan does not support %q owners", owner.Owner.Kind)
	}
	if owner.AssetRoot == "" || !filepath.IsAbs(owner.AssetRoot) {
		return nil, errors.New("orphan scan requires an absolute asset root")
	}
	if store.Base == "" {
		return nil, errors.New("orphan scan requires the staging store to check unfinished transactions")
	}
	candidates, err := orphanCandidates(owner.AssetRoot, opts)
	if err != nil || len(candidates) == 0 {
		return nil, err
	}
	inFlight, err := inFlightAssetPaths(store)
	if err != nil {
		return nil, err
	}
	for _, path := range append(append([]string(nil), opts.Protected...), inFlight...) {
		if err := protectCandidate(candidates, path); err != nil {
			return nil, err
		}
	}
	documents, err := orphanDocuments(owner)
	if err != nil {
		return nil, err
	}
	for _, document := range documents {
		if err := markReferences(candidates, document); err != nil {
			return nil, err
		}
	}
	var orphans []Orphan
	for _, candidate := range candidates {
		if !candidate.referenced {
			orphans = append(orphans, Orphan{Path: candidate.path, Size: candidate.info.Size(), ModTime: candidate.info.ModTime()})
		}
	}
	sort.Slice(orphans, func(i, j int) bool { return orphans[i].Path < orphans[j].Path })
	return orphans, nil
}

// TrashOrphans moves each requested path to the trash when a fresh scan still
// reports it as an orphan. Paths that are referenced again, outside the asset
// root, symlinks, or missing are skipped. It returns the paths it moved.
func TrashOrphans(owner ResolvedOwner, store Store, opts OrphanOptions, paths []string, trash func(string) error) ([]string, error) {
	if trash == nil {
		trash = trashAsset
	}
	orphans, err := FindOrphans(owner, store, opts)
	if err != nil {
		return nil, err
	}
	type scanned struct {
		path string
		info os.FileInfo
	}
	fresh := make([]scanned, 0, len(orphans))
	for _, orphan := range orphans {
		if info, err := os.Lstat(orphan.Path); err == nil && info.Mode().IsRegular() {
			fresh = append(fresh, scanned{orphan.Path, info})
		}
	}
	var trashed []string
	moved := make(map[string]bool)
	for _, requested := range paths {
		// Lstat before canonicalizing: resolving a symlink would let it match
		// the orphan it points at.
		info, err := os.Lstat(requested)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		canonical, err := canonicalForContainment(requested)
		if err != nil || filepath.Dir(canonical) != owner.AssetRoot || !pathWithin(owner.AssetRoot, canonical) {
			continue
		}
		for _, orphan := range fresh {
			if moved[orphan.path] || !os.SameFile(orphan.info, info) {
				continue
			}
			if err := trash(orphan.path); err != nil {
				return trashed, fmt.Errorf("trash %q: %w", orphan.path, err)
			}
			if _, err := os.Lstat(orphan.path); err == nil {
				return trashed, fmt.Errorf("trash %q: file is still present", orphan.path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return trashed, err
			}
			moved[orphan.path] = true
			trashed = append(trashed, orphan.path)
			break
		}
	}
	return trashed, nil
}

func orphanCandidates(root string, opts OrphanOptions) ([]*orphanCandidate, error) {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	grace := opts.Grace
	if grace <= 0 {
		grace = DefaultOrphanGrace
	}
	cutoff := now().Add(-grace)
	var candidates []*orphanCandidate
	for _, entry := range entries {
		// Dotfiles cover the asset index and every temporary file.
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		candidates = append(candidates, &orphanCandidate{
			path:    path,
			info:    info,
			folded:  foldName(entry.Name()),
			needles: basenameNeedles(entry.Name()),
		})
	}
	return candidates, nil
}

// inFlightAssetPaths returns every asset path recorded by a transaction that
// is neither committed nor aborted. A dedup hit reuses an existing asset
// without creating it, so Assets is read as well as CreatedFiles.
func inFlightAssetPaths(store Store) ([]string, error) {
	var paths []string
	sessions, err := subdirectories(filepath.Join(store.Base, "sessions"))
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		buffers, err := subdirectories(session)
		if err != nil {
			return nil, err
		}
		for _, buffer := range buffers {
			generations, err := subdirectories(buffer)
			if err != nil {
				return nil, err
			}
			for _, generation := range generations {
				transactions, err := subdirectories(filepath.Join(generation, "transactions"))
				if err != nil {
					return nil, err
				}
				for _, transaction := range transactions {
					manifest, err := store.Load(transaction)
					if err != nil {
						return nil, fmt.Errorf("transaction manifest %q: %w", transaction, err)
					}
					if manifest.State == StateCommitted || manifest.State == StateAborted {
						continue
					}
					for _, asset := range manifest.Assets {
						paths = append(paths, asset.Path)
					}
					paths = append(paths, manifest.CreatedFiles...)
					paths = append(paths, manifest.StagingFiles...)
				}
			}
		}
	}
	return paths, nil
}

func subdirectories(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var directories []string
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, filepath.Join(directory, entry.Name()))
		}
	}
	return directories, nil
}

// protectCandidate marks the candidate that is path. A path that does not
// exist still protects any candidate with the same folded name, because a
// differently normalized spelling of an existing file can fail to resolve.
func protectCandidate(candidates []*orphanCandidate, path string) error {
	if path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		folded := foldName(filepath.Base(path))
		for _, candidate := range candidates {
			if candidate.folded == folded {
				candidate.referenced = true
			}
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("protected path %q: %w", path, err)
	}
	markSameFile(candidates, info)
	return nil
}

func markSameFile(candidates []*orphanCandidate, info os.FileInfo) {
	if !info.Mode().IsRegular() {
		return
	}
	for _, candidate := range candidates {
		if !candidate.referenced && candidate.info.Size() == info.Size() && os.SameFile(candidate.info, info) {
			candidate.referenced = true
		}
	}
}

func orphanDocuments(owner ResolvedOwner) ([]string, error) {
	if owner.Owner.Kind == OwnerStandalone {
		info, err := os.Stat(owner.Owner.Document)
		if err != nil {
			return nil, fmt.Errorf("standalone document: %w", err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("standalone document %q is not a regular file", owner.Owner.Document)
		}
		return []string{owner.Owner.Document}, nil
	}
	root := owner.ContainmentRoot
	var documents []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if path != root && strings.HasPrefix(name, ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			// Emacs lock files are dangling symlinks named .#<file>.
			if strings.HasPrefix(name, ".#") {
				return nil
			}
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 && !strings.HasPrefix(name, ".") {
			info, err := os.Stat(path)
			if err != nil {
				if orphanDocumentExtensions[strings.ToLower(filepath.Ext(name))] {
					return fmt.Errorf("document %q: %w", path, err)
				}
				return nil
			}
			// WalkDir does not descend into symlinked directories, so the
			// documents inside one cannot be proven absent.
			if info.IsDir() {
				return fmt.Errorf("project contains symlinked directory %q; orphan scan cannot cover it", path)
			}
		}
		if !orphanDocumentExtensions[strings.ToLower(filepath.Ext(name))] {
			return nil
		}
		documents = append(documents, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return documents, nil
}

func markReferences(candidates []*orphanCandidate, document string) error {
	info, err := os.Stat(document)
	if err != nil {
		return fmt.Errorf("document %q: %w", document, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("document %q is not a regular file", document)
	}
	data, err := os.ReadFile(document)
	if err != nil {
		return fmt.Errorf("document %q: %w", document, err)
	}
	text := string(data)
	directory := filepath.Dir(document)
	resolved := make(map[string]bool)
	for _, match := range orphanLinkPattern.FindAllStringSubmatch(text, -1) {
		raw := match[1]
		if raw == "" {
			raw = match[2]
		}
		for _, target := range linkTargetVariants(raw) {
			path := expandLinkTarget(target, directory)
			if path == "" || resolved[path] {
				continue
			}
			resolved[path] = true
			if info, err := os.Stat(path); err == nil {
				markSameFile(candidates, info)
			}
		}
	}
	haystacks := textVariants(text)
	for _, candidate := range candidates {
		if candidate.referenced {
			continue
		}
	search:
		for _, needle := range candidate.needles {
			for _, haystack := range haystacks {
				if strings.Contains(haystack, needle) {
					candidate.referenced = true
					break search
				}
			}
		}
	}
	return nil
}

// linkTargetVariants returns raw plus every form reachable by stripping a
// title, an Org search suffix, a file: prefix, angle brackets, surrounding
// space, or percent escapes. Each step shortens the string, so the set is
// small and finite.
func linkTargetVariants(raw string) []string {
	seen := make(map[string]bool)
	var forms []string
	var add func(string)
	add = func(form string) {
		if form == "" || seen[form] {
			return
		}
		seen[form] = true
		forms = append(forms, form)
		add(strings.TrimSpace(form))
		add(strings.TrimPrefix(form, "file:"))
		if strings.HasPrefix(form, "<") && strings.HasSuffix(form, ">") {
			add(form[1 : len(form)-1])
		}
		if before, _, found := strings.Cut(form, "::"); found {
			add(before)
		}
		if title := markdownLinkTitle.FindStringIndex(form); title != nil {
			add(form[:title[0]])
		}
		add(decodePercentLenient(form))
	}
	add(raw)
	return forms
}

func expandLinkTarget(target, directory string) string {
	if strings.HasPrefix(target, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		target = filepath.Join(home, target[2:])
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(directory, target)
	}
	return filepath.Clean(target)
}

// textVariants returns the lowercased document text, raw and percent-decoded,
// each in its original, composed, and decomposed Hangul form.
func textVariants(text string) []string {
	return uniqueStrings(
		normalizedForms(strings.ToLower(text)),
		normalizedForms(strings.ToLower(decodePercentLenient(text))),
	)
}

// basenameNeedles returns the lowercased name in each normalization, plus its
// URL path escape and the Org link escape (which encodes only []%\).
func basenameNeedles(name string) []string {
	var needles []string
	for _, form := range normalizedForms(name) {
		needles = append(needles, strings.ToLower(form),
			strings.ToLower(url.PathEscape(form)), strings.ToLower(orgLinkEscape(form)))
	}
	return uniqueStrings(needles)
}

func normalizedForms(value string) []string {
	return uniqueStrings([]string{value, hangulNFC(value), hangulNFD(value)})
}

func uniqueStrings(groups ...[]string) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, group := range groups {
		for _, value := range group {
			if value != "" && !seen[value] {
				seen[value] = true
				unique = append(unique, value)
			}
		}
	}
	return unique
}

func foldName(name string) string {
	return strings.ToLower(hangulNFC(name))
}

func orgLinkEscape(value string) string {
	var builder strings.Builder
	for index := 0; index < len(value); index++ {
		switch character := value[index]; character {
		case '[', ']', '%', '\\':
			fmt.Fprintf(&builder, "%%%02X", character)
		default:
			builder.WriteByte(character)
		}
	}
	return builder.String()
}

// decodePercentLenient decodes valid %XX escapes and leaves everything else,
// including '+' and malformed escapes, unchanged, like url-unhex-string.
func decodePercentLenient(value string) string {
	if !strings.Contains(value, "%") {
		return value
	}
	decoded := make([]byte, 0, len(value))
	for index := 0; index < len(value); index++ {
		if value[index] == '%' && index+2 < len(value) && isHex(value[index+1]) && isHex(value[index+2]) {
			decoded = append(decoded, unhex(value[index+1])<<4|unhex(value[index+2]))
			index += 2
			continue
		}
		decoded = append(decoded, value[index])
	}
	return string(decoded)
}

func isHex(character byte) bool {
	return '0' <= character && character <= '9' || 'a' <= character && character <= 'f' || 'A' <= character && character <= 'F'
}

func unhex(character byte) byte {
	switch {
	case '0' <= character && character <= '9':
		return character - '0'
	case 'a' <= character && character <= 'f':
		return character - 'a' + 10
	default:
		return character - 'A' + 10
	}
}

// Hangul syllables compose and decompose algorithmically (Unicode 3.12), which
// covers the NFC/NFD split between Emacs and macOS file names without a
// normalization table dependency. Other scripts are left unchanged.
const (
	hangulSBase  = 0xAC00
	hangulLBase  = 0x1100
	hangulVBase  = 0x1161
	hangulTBase  = 0x11A7
	hangulLCount = 19
	hangulVCount = 21
	hangulTCount = 28
	hangulNCount = hangulVCount * hangulTCount
	hangulSCount = hangulLCount * hangulNCount
)

func hangulNFD(value string) string {
	var builder strings.Builder
	for _, character := range value {
		index := character - hangulSBase
		if index < 0 || index >= hangulSCount {
			builder.WriteRune(character)
			continue
		}
		builder.WriteRune(hangulLBase + index/hangulNCount)
		builder.WriteRune(hangulVBase + index%hangulNCount/hangulTCount)
		if trailing := index % hangulTCount; trailing != 0 {
			builder.WriteRune(hangulTBase + trailing)
		}
	}
	return builder.String()
}

func hangulNFC(value string) string {
	characters := []rune(value)
	var builder strings.Builder
	for index := 0; index < len(characters); index++ {
		leading := characters[index] - hangulLBase
		if leading < 0 || leading >= hangulLCount || index+1 >= len(characters) {
			builder.WriteRune(characters[index])
			continue
		}
		vowel := characters[index+1] - hangulVBase
		if vowel < 0 || vowel >= hangulVCount {
			builder.WriteRune(characters[index])
			continue
		}
		syllable := hangulSBase + (leading*hangulVCount+vowel)*hangulTCount
		index++
		if index+1 < len(characters) {
			if trailing := characters[index+1] - hangulTBase; trailing > 0 && trailing < hangulTCount {
				syllable += trailing
				index++
			}
		}
		builder.WriteRune(syllable)
	}
	return builder.String()
}

// trashScript moves argv[0] to the Trash with NSFileManager. Unlike a Finder
// "delete" Apple event it needs no Automation permission, and Finder
// rejected POSIX file references in practice (error -1728).
const trashScript = `ObjC.import("Foundation");
function run(argv) {
  var error = $();
  if (!$.NSFileManager.defaultManager.trashItemAtURLResultingItemURLError(
      $.NSURL.fileURLWithPath(argv[0]), null, error)) {
    throw new Error(ObjC.unwrap(error.localizedDescription));
  }
  return "ok";
}`

// finderTrash moves path to the macOS Trash, so the user can restore it.
// The path travels as an argument, never as script text.
func finderTrash(path string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%s: moving assets to the trash is only supported on macOS", CodeUnsupportedCapability)
	}
	output, err := exec.Command("osascript", "-l", "JavaScript", "-e", trashScript, path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("trash: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
