package notecache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Run(ctx context.Context, request Request) Response {
	if request.Version != 0 && request.Version != ProtocolVersion {
		return failure("invalid_request", "unsupported protocol version")
	}
	switch request.Operation {
	case "catalog", "index", "lookup", "backlinks":
	default:
		return failure("invalid_request", "operation must be catalog, index, lookup, or backlinks")
	}
	scope, err := normalizeScope(request)
	if err != nil {
		return failure("invalid_scope", err.Error())
	}
	response := Response{
		Version:     ProtocolVersion,
		OK:          true,
		Status:      "ok",
		ScopeKey:    scope.key,
		Occurrences: map[string][]Occurrence{},
	}
	store, err := openStore(ctx, scope.cacheDir, scope.key)
	if err != nil {
		return failure("cache_open_failed", err.Error())
	}
	defer store.close()
	parserMatches, err := store.parserVersionMatches(ctx)
	if err != nil {
		return failure("cache_read_failed", err.Error())
	}
	files, scanErrors := scanFiles(scope)
	response.Errors = append(response.Errors, scanErrors...)
	response.Stats.Scanned = len(files)
	if len(scanErrors) > 0 {
		response.OK = false
		response.Status = "error"
		response.Errors = append([]Error{{Code: "incomplete_scan", Message: "catalog scan did not complete; cache was not updated"}}, response.Errors...)
		return response
	}
	scanned := make(map[string]struct{}, len(files))
	changedRows := make([]fileRow, 0)
	rows := make([]fileRow, 0, len(files))
	overlayByPath := make(map[string]Overlay)
	for _, overlay := range scope.overlays {
		overlayByPath[overlay.Path] = overlay
	}
	invalidatedRows := make([]string, 0)
	for _, file := range files {
		select {
		case <-ctx.Done():
			return failure("cancelled", ctx.Err().Error())
		default:
		}
		scanned[file.path] = struct{}{}
		fp := FileFingerprint{Path: displayPath(file.path), SHA256: file.hash, Size: file.size, ModTimeNS: file.modTimeNS}
		if overlay, ok := overlayByPath[file.path]; ok {
			fp.SHA256 = contentHash(overlay.Text)
			fp.Size = int64(len(overlay.Text))
			fp.ModTimeNS = 0
			fp.Overlay = true
			if !parserMatches {
				invalidatedRows = append(invalidatedRows, file.path)
			}
			row := rowFromParsed(file.path, fp.SHA256, fp.Size, fp.ModTimeNS, parseOrg(file.path, overlay.Text, file.documentCandidate, file.tasksFile))
			rows = append(rows, row)
			response.Stats.Parsed++
			response.Files = append(response.Files, fp)
			continue
		}
		if parserMatches {
			cached, ok, err := store.load(ctx, file.path, file.hash)
			if err != nil {
				return failure("cache_read_failed", err.Error())
			}
			if ok {
				rows = append(rows, *cached)
				response.Stats.CacheHits++
				response.Files = append(response.Files, fp)
				continue
			}
		}
		parsed := parseOrg(file.path, file.text, file.documentCandidate, file.tasksFile)
		row := rowFromParsed(file.path, file.hash, file.size, file.modTimeNS, parsed)
		rows = append(rows, row)
		changedRows = append(changedRows, row)
		response.Stats.Parsed++
		response.Files = append(response.Files, fp)
	}
	_, generation, err := store.save(ctx, scanned, changedRows, invalidatedRows)
	if err != nil {
		return failure("cache_write_failed", err.Error())
	}
	response.Generation = generation
	if response.Generation == 0 {
		response.Generation, _ = store.generation(ctx)
	}
	fillResponse(&response, request, rows)
	return response
}

type scope struct {
	projectID     string
	notesRoot     string
	tasksFile     string
	excludedRoots map[string]struct{}
	cacheDir      string
	overlays      []Overlay
	key           string
}

type sourceFile struct {
	path              string
	text              string
	hash              string
	size              int64
	modTimeNS         int64
	documentCandidate bool
	tasksFile         bool
}

func normalizeScope(request Request) (scope, error) {
	projectID := request.ProjectID
	notesRoot := request.NotesRoot
	tasksFile := request.TasksFile
	excludedRoots := request.ExcludedRoots
	if request.Scope != nil {
		projectID = request.Scope.ProjectID
		notesRoot = request.Scope.NotesRoot
		tasksFile = request.Scope.TasksFile
		excludedRoots = request.Scope.ExcludedRoots
	}
	if projectID == "" {
		return scope{}, errors.New("project_id is required")
	}
	root, err := validateRoot(notesRoot)
	if err != nil {
		return scope{}, fmt.Errorf("notes_root: %w", err)
	}
	result := scope{
		projectID:     projectID,
		notesRoot:     root,
		excludedRoots: map[string]struct{}{},
		cacheDir:      request.CacheDir,
	}
	for _, excluded := range excludedRoots {
		path, err := validatePathInside(root, excluded, true)
		if err != nil {
			return scope{}, fmt.Errorf("excluded_roots: %w", err)
		}
		result.excludedRoots[path] = struct{}{}
	}
	if tasksFile != "" {
		path, err := validateLocalFile(tasksFile)
		if err != nil {
			return scope{}, fmt.Errorf("tasks_file: %w", err)
		}
		result.tasksFile = path
	}
	for _, overlay := range request.Overlays {
		path, err := validateOverlayPath(root, result.tasksFile, overlay.Path)
		if err != nil {
			return scope{}, fmt.Errorf("overlay: %w", err)
		}
		overlay.Path = path
		result.overlays = append(result.overlays, overlay)
	}
	result.key = scopeKey(result)
	if result.cacheDir != "" {
		if !filepath.IsAbs(result.cacheDir) || remotePath(result.cacheDir) {
			return scope{}, errors.New("cache_dir must be an absolute local path")
		}
		result.cacheDir = filepath.Clean(result.cacheDir)
	}
	return result, nil
}

func validateRoot(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) || remotePath(path) {
		return "", errors.New("must be an absolute local path")
	}
	clean := filepath.Clean(path)
	info, err := os.Lstat(clean)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", errors.New("must be a real directory")
	}
	real, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", err
	}
	return real, nil
}

func validatePathInside(root, path string, directory bool) (string, error) {
	if path == "" || !filepath.IsAbs(path) || remotePath(path) {
		return "", errors.New("must be an absolute local path")
	}
	clean := filepath.Clean(path)
	info, err := os.Lstat(clean)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("symbolic links are not accepted")
	}
	real := clean
	if err == nil {
		real, err = filepath.EvalSymlinks(clean)
		if err != nil {
			return "", err
		}
		if directory && !info.IsDir() {
			return "", errors.New("must be a directory")
		}
	}
	if !insidePath(root, real) {
		return "", fmt.Errorf("path escapes notes_root: %s", path)
	}
	return real, nil
}

func validateLocalFile(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) || remotePath(path) {
		return "", errors.New("must be an absolute local path")
	}
	clean := filepath.Clean(path)
	info, err := os.Lstat(clean)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
		return "", errors.New("must be a real file")
	}
	real, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", err
	}
	return real, nil
}

func validateOverlayPath(root, tasksFile, path string) (string, error) {
	if tasksFile != "" {
		clean, err := validateLocalFile(path)
		if err == nil && clean == tasksFile {
			return clean, nil
		}
	}
	return validatePathInside(root, path, false)
}

func remotePath(path string) bool {
	return strings.HasPrefix(path, "/ssh:") || strings.HasPrefix(path, "/scp:") || strings.HasPrefix(path, "/sudo:")
}

func insidePath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func scopeKey(s scope) string {
	excluded := make([]string, 0, len(s.excludedRoots))
	for root := range s.excludedRoots {
		excluded = append(excluded, root)
	}
	sort.Strings(excluded)
	input := strings.Join(append([]string{s.projectID, s.notesRoot, s.tasksFile}, excluded...), "\x00")
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

func scanFiles(s scope) ([]sourceFile, []Error) {
	var files []sourceFile
	var errs []Error
	err := filepath.WalkDir(s.notesRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			errs = append(errs, Error{Code: "scan_failed", Message: walkErr.Error(), File: path})
			return nil
		}
		if path == s.notesRoot {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if shouldSkipDir(s, path, entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			return nil
		}
		if filepath.Ext(path) != ".org" {
			return nil
		}
		real, err := validatePathInside(s.notesRoot, path, false)
		if err != nil {
			errs = append(errs, Error{Code: "invalid_path", Message: err.Error(), File: path})
			return nil
		}
		localTasks := filepath.Join(s.notesRoot, "tasks.org")
		if real == localTasks && real != s.tasksFile {
			return nil
		}
		source, err := readSourceFile(real, real != s.tasksFile, real == s.tasksFile)
		if err != nil {
			errs = append(errs, Error{Code: "read_failed", Message: err.Error(), File: real})
			return nil
		}
		files = append(files, source)
		return nil
	})
	if err != nil {
		errs = append(errs, Error{Code: "scan_failed", Message: err.Error()})
	}
	if s.tasksFile != "" && !containsFile(files, s.tasksFile) {
		if source, err := readSourceFile(s.tasksFile, false, true); err == nil {
			files = append(files, source)
		} else {
			errs = append(errs, Error{Code: "read_failed", Message: err.Error(), File: s.tasksFile})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, errs
}

func shouldSkipDir(s scope, path, name string) bool {
	if _, ok := s.excludedRoots[path]; ok {
		return true
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	if filepath.Dir(path) == s.notesRoot && (name == "vendor" || name == "assets") {
		return true
	}
	if path != s.notesRoot {
		if _, err := os.Stat(filepath.Join(path, ".imoogi-project.json")); err == nil {
			return true
		}
	}
	return false
}

func readSourceFile(path string, documentCandidate bool, tasksFile bool) (sourceFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return sourceFile{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return sourceFile{}, err
	}
	sum := sha256.Sum256(data)
	return sourceFile{
		path:              path,
		text:              string(data),
		hash:              hex.EncodeToString(sum[:]),
		size:              info.Size(),
		modTimeNS:         info.ModTime().UnixNano(),
		documentCandidate: documentCandidate,
		tasksFile:         tasksFile,
	}, nil
}

func containsFile(files []sourceFile, path string) bool {
	for _, file := range files {
		if file.path == path {
			return true
		}
	}
	return false
}

func rowFromParsed(path, hash string, size, modTimeNS int64, parsed parsedFile) fileRow {
	sortParsed(&parsed)
	return fileRow{
		path:        path,
		hash:        hash,
		size:        size,
		modTimeNS:   modTimeNS,
		document:    parsed.document,
		occurrences: parsed.occurrences,
		links:       parsed.links,
	}
}

func fillResponse(response *Response, request Request, rows []fileRow) {
	for _, row := range rows {
		if row.document != nil {
			document := *row.document
			document.File = displayPath(document.File)
			if includeDocument(request, document) {
				response.Documents = append(response.Documents, document)
			}
		}
		for _, occurrence := range row.occurrences {
			if request.Operation == "lookup" && request.ID != "" && occurrence.ID != request.ID {
				continue
			}
			response.Occurrences[occurrence.ID] = append(response.Occurrences[occurrence.ID], Occurrence{File: displayPath(occurrence.File), Position: occurrence.Position})
		}
		for _, link := range row.links {
			if includeLink(request, link) {
				link.File = displayPath(link.File)
				response.Links = append(response.Links, link)
			}
		}
	}
	sort.Slice(response.Documents, func(i, j int) bool { return response.Documents[i].File < response.Documents[j].File })
	sort.Slice(response.Links, func(i, j int) bool {
		if response.Links[i].File == response.Links[j].File {
			return response.Links[i].Position < response.Links[j].Position
		}
		return response.Links[i].File < response.Links[j].File
	})
}

func includeDocument(request Request, document Document) bool {
	if request.Operation != "lookup" || request.ID == "" {
		return true
	}
	return document.ID == request.ID
}

func includeLink(request Request, link Link) bool {
	switch request.Operation {
	case "backlinks":
		return request.ID == "" || link.TargetID == request.ID
	case "lookup":
		return request.ID == "" || link.SourceID == request.ID || link.TargetID == request.ID
	default:
		return true
	}
}
