package notecache

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRunCachesWarmCatalogAndReparsesChangedFileOnly(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeFile(t, filepath.Join(root, "a.org"), ":PROPERTIES:\n:ID: DOC-A\n:END:\n\n#+TITLE: A\n")
	writeFile(t, filepath.Join(root, "b.org"), ":PROPERTIES:\n:ID: DOC-B\n:END:\n\n#+TITLE: B\n")
	req := catalogRequest(root, cacheDir)

	first := Run(context.Background(), req)
	if !first.OK {
		t.Fatalf("first run failed: %#v", first.Errors)
	}
	if first.Stats.Parsed != 2 || first.Stats.CacheHits != 0 {
		t.Fatalf("first stats = %#v, want 2 parsed and 0 hits", first.Stats)
	}

	second := Run(context.Background(), req)
	if !second.OK {
		t.Fatalf("second run failed: %#v", second.Errors)
	}
	if second.Stats.Parsed != 0 || second.Stats.CacheHits != 2 {
		t.Fatalf("second stats = %#v, want 0 parsed and 2 hits", second.Stats)
	}

	writeFile(t, filepath.Join(root, "b.org"), ":PROPERTIES:\n:ID: DOC-B2\n:END:\n\n#+TITLE: B2\n")
	third := Run(context.Background(), req)
	if !third.OK {
		t.Fatalf("third run failed: %#v", third.Errors)
	}
	if third.Stats.Parsed != 1 || third.Stats.CacheHits != 1 {
		t.Fatalf("third stats = %#v, want 1 parsed and 1 hit", third.Stats)
	}
	if _, ok := third.Occurrences["DOC-B2"]; !ok {
		t.Fatalf("changed ID missing from occurrences: %#v", third.Occurrences)
	}
}

func TestRunHashesContentWhenModTimeAndSizeArePreserved(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	path := filepath.Join(root, "same.org")
	writeFile(t, path, ":PROPERTIES:\n:ID: FIRST\n:END:\n")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	req := catalogRequest(root, cacheDir)
	if response := Run(context.Background(), req); !response.OK {
		t.Fatalf("first run failed: %#v", response.Errors)
	}
	replacement := ":PROPERTIES:\n:ID: OTHER\n:END:\n"
	if len(replacement) != int(info.Size()) {
		t.Fatalf("test fixture size changed: %d vs %d", len(replacement), info.Size())
	}
	if err := os.WriteFile(path, []byte(replacement), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	response := Run(context.Background(), req)
	if !response.OK {
		t.Fatalf("second run failed: %#v", response.Errors)
	}
	if response.Stats.Parsed != 1 {
		t.Fatalf("stats = %#v, want content hash to force one parse", response.Stats)
	}
	if _, ok := response.Occurrences["OTHER"]; !ok {
		t.Fatalf("new same-size ID missing: %#v", response.Occurrences)
	}
}

func TestRunOverlaysReplaceRequestResultButAreNotPersisted(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	path := filepath.Join(root, "draft.org")
	writeFile(t, path, ":PROPERTIES:\n:ID: DISK-ID\n:END:\n\n#+TITLE: Disk\n")
	req := catalogRequest(root, cacheDir)
	if response := Run(context.Background(), req); !response.OK {
		t.Fatalf("first run failed: %#v", response.Errors)
	}
	req.Overlays = []Overlay{{Path: path, Text: ":PROPERTIES:\n:ID: OVERLAY-ID\n:END:\n\n#+TITLE: Overlay\n", Revision: "r1"}}
	overlay := Run(context.Background(), req)
	if !overlay.OK {
		t.Fatalf("overlay run failed: %#v", overlay.Errors)
	}
	if _, ok := overlay.Occurrences["OVERLAY-ID"]; !ok {
		t.Fatalf("overlay ID missing: %#v", overlay.Occurrences)
	}
	req.Overlays = nil
	disk := Run(context.Background(), req)
	if !disk.OK {
		t.Fatalf("disk run failed: %#v", disk.Errors)
	}
	if _, ok := disk.Occurrences["OVERLAY-ID"]; ok {
		t.Fatalf("overlay ID persisted into disk cache: %#v", disk.Occurrences)
	}
	if _, ok := disk.Occurrences["DISK-ID"]; !ok {
		t.Fatalf("disk ID missing after overlay run: %#v", disk.Occurrences)
	}
}

func TestRunKeepsIDlessDocumentCandidateWithoutIdentityKind(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeFile(t, filepath.Join(root, "untitled.org"), "#+TITLE: Metadata Only\n\nBody\n")
	response := Run(context.Background(), catalogRequest(root, cacheDir))
	if !response.OK {
		t.Fatalf("run failed: %#v", response.Errors)
	}
	if len(response.Documents) != 1 {
		t.Fatalf("documents = %#v, want one candidate", response.Documents)
	}
	doc := response.Documents[0]
	if doc.ID != "" || doc.Kind != "" || doc.Title != "" {
		t.Fatalf("IDless document identity fields = %#v, want empty id/title/kind", doc)
	}
}

func TestRunKeepsDuplicateOccurrencesIgnoresLiteralsAndReportsUTF8Positions(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeFile(t, filepath.Join(root, "dup.org"), strings.Join([]string{
		":PROPERTIES:",
		":ID: DUP",
		":END:",
		"",
		"#+BEGIN_SRC org",
		":PROPERTIES:",
		":ID: FAKE",
		":END:",
		"#+END_SRC",
		"",
		"* 제목",
		":PROPERTIES:",
		":ID: DUP",
		":END:",
		"",
	}, "\n"))
	response := Run(context.Background(), catalogRequest(root, cacheDir))
	if !response.OK {
		t.Fatalf("run failed: %#v", response.Errors)
	}
	if got := len(response.Occurrences["DUP"]); got != 2 {
		t.Fatalf("DUP occurrences = %d, want 2: %#v", got, response.Occurrences)
	}
	if _, ok := response.Occurrences["FAKE"]; ok {
		t.Fatalf("literal source block ID was indexed: %#v", response.Occurrences["FAKE"])
	}
	positions := response.Occurrences["DUP"]
	if positions[1].Position <= positions[0].Position {
		t.Fatalf("positions are not increasing: %#v", positions)
	}
}

func TestRunMatchesOrgPropertyDrawerParityEdges(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeFile(t, filepath.Join(root, "indented.org"), "* TODO A\n  :PROPERTIES:\n  :ID: INDENTED\n  :END:\n")
	writeFile(t, filepath.Join(root, "planning.org"), "* TODO A\nSCHEDULED: <2026-10-03 Sat>\n:PROPERTIES:\n:ID: PLANNED\n:END:\n")
	writeFile(t, filepath.Join(root, "unclosed.org"), ":PROPERTIES:\n:ID: UNCLOSED\n")
	response := Run(context.Background(), catalogRequest(root, cacheDir))
	if !response.OK {
		t.Fatalf("run failed: %#v", response.Errors)
	}
	for _, id := range []string{"INDENTED", "PLANNED"} {
		if _, ok := response.Occurrences[id]; !ok {
			t.Fatalf("%s missing from occurrences: %#v", id, response.Occurrences)
		}
	}
	if _, ok := response.Occurrences["UNCLOSED"]; ok {
		t.Fatalf("unclosed property drawer was indexed: %#v", response.Occurrences["UNCLOSED"])
	}
}

func TestRunParserVersionMismatchInvalidatesHashMatchedCache(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeFile(t, filepath.Join(root, "doc.org"), ":PROPERTIES:\n:ID: DOC\n:END:\n")
	req := catalogRequest(root, cacheDir)
	first := Run(context.Background(), req)
	if !first.OK {
		t.Fatalf("first run failed: %#v", first.Errors)
	}
	scope, err := normalizeScope(req)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(cacheDir, scope.key+".sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE meta SET value='-1' WHERE key='parser_version'`); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	second := Run(context.Background(), req)
	if !second.OK {
		t.Fatalf("second run failed: %#v", second.Errors)
	}
	if second.Stats.Parsed != 1 || second.Stats.CacheHits != 0 {
		t.Fatalf("stats = %#v, want parser version mismatch to reparse", second.Stats)
	}
	third := Run(context.Background(), req)
	if !third.OK {
		t.Fatalf("third run failed: %#v", third.Errors)
	}
	if third.Stats.Parsed != 0 || third.Stats.CacheHits != 1 {
		t.Fatalf("stats = %#v, want parser version reset to restore cache hits", third.Stats)
	}
}

func TestRunParserVersionMismatchWithOverlayInvalidatesPersistedRow(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	path := filepath.Join(root, "doc.org")
	writeFile(t, path, ":PROPERTIES:\n:ID: DOC\n:END:\n")
	req := catalogRequest(root, cacheDir)
	first := Run(context.Background(), req)
	if !first.OK {
		t.Fatalf("first run failed: %#v", first.Errors)
	}
	scope, err := normalizeScope(req)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(cacheDir, scope.key+".sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE meta SET value='-1' WHERE key='parser_version'`); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE ids SET id='STALE' WHERE file=? AND id='DOC'`, path); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	req.Overlays = []Overlay{{Path: path, Text: ":PROPERTIES:\n:ID: OVERLAY\n:END:\n"}}
	overlay := Run(context.Background(), req)
	if !overlay.OK {
		t.Fatalf("overlay run failed: %#v", overlay.Errors)
	}
	if _, ok := overlay.Occurrences["OVERLAY"]; !ok {
		t.Fatalf("overlay ID missing from response: %#v", overlay.Occurrences)
	}

	req.Overlays = nil
	disk := Run(context.Background(), req)
	if !disk.OK {
		t.Fatalf("disk run failed: %#v", disk.Errors)
	}
	if _, ok := disk.Occurrences["STALE"]; ok {
		t.Fatalf("stale persisted row was reused after overlay version mismatch: %#v", disk.Occurrences)
	}
	if _, ok := disk.Occurrences["DOC"]; !ok {
		t.Fatalf("disk ID missing after invalidation: %#v", disk.Occurrences)
	}
	if disk.Stats.Parsed != 1 || disk.Stats.CacheHits != 0 {
		t.Fatalf("stats = %#v, want stale row deleted and disk reparsed", disk.Stats)
	}
}

func TestRunBoundariesSymlinkDeleteAndNestedProject(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeFile(t, filepath.Join(root, "keep.org"), ":PROPERTIES:\n:ID: KEEP\n:END:\n")
	writeFile(t, filepath.Join(root, "vendor", "skip.org"), ":PROPERTIES:\n:ID: VENDOR\n:END:\n")
	writeFile(t, filepath.Join(root, "nested", ".imoogi-project.json"), "{}")
	writeFile(t, filepath.Join(root, "nested", "skip.org"), ":PROPERTIES:\n:ID: NESTED\n:END:\n")
	if err := os.Symlink(filepath.Join(root, "keep.org"), filepath.Join(root, "linked.org")); err != nil {
		t.Fatal(err)
	}
	req := catalogRequest(root, cacheDir)
	first := Run(context.Background(), req)
	if !first.OK {
		t.Fatalf("first run failed: %#v", first.Errors)
	}
	if _, ok := first.Occurrences["KEEP"]; !ok {
		t.Fatalf("KEEP missing: %#v", first.Occurrences)
	}
	for _, id := range []string{"VENDOR", "NESTED"} {
		if _, ok := first.Occurrences[id]; ok {
			t.Fatalf("%s should be outside boundary: %#v", id, first.Occurrences)
		}
	}
	if err := os.Remove(filepath.Join(root, "keep.org")); err != nil {
		t.Fatal(err)
	}
	second := Run(context.Background(), req)
	if !second.OK {
		t.Fatalf("second run failed: %#v", second.Errors)
	}
	if _, ok := second.Occurrences["KEEP"]; ok {
		t.Fatalf("deleted file remained in cache: %#v", second.Occurrences)
	}
}

func TestRunScopeBoundariesForHiddenRootVendorNestedVendorAndTasksAncestor(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	tasks := filepath.Join(root, "sub", "agenda", "tasks.org")
	writeFile(t, filepath.Join(root, ".hidden.org"), ":PROPERTIES:\n:ID: HIDDEN\n:END:\n")
	writeFile(t, filepath.Join(root, "vendor", "skip.org"), ":PROPERTIES:\n:ID: ROOT-VENDOR\n:END:\n")
	writeFile(t, filepath.Join(root, "assets", "skip.org"), ":PROPERTIES:\n:ID: ROOT-ASSETS\n:END:\n")
	writeFile(t, filepath.Join(root, "dev", "vendor", "keep.org"), ":PROPERTIES:\n:ID: NESTED-VENDOR\n:END:\n")
	writeFile(t, filepath.Join(root, "sub", "docs", "keep.org"), ":PROPERTIES:\n:ID: TASK-ANCESTOR-DOC\n:END:\n")
	writeFile(t, tasks, "* TODO Indexed\n:PROPERTIES:\n:ID: TASK-IN-SUBDIR\n:END:\n")
	req := catalogRequest(root, cacheDir)
	req.Scope.TasksFile = tasks
	response := Run(context.Background(), req)
	if !response.OK {
		t.Fatalf("run failed: %#v", response.Errors)
	}
	for _, id := range []string{"NESTED-VENDOR", "TASK-ANCESTOR-DOC", "TASK-IN-SUBDIR"} {
		if _, ok := response.Occurrences[id]; !ok {
			t.Fatalf("%s missing from occurrences: %#v", id, response.Occurrences)
		}
	}
	for _, id := range []string{"HIDDEN", "ROOT-VENDOR", "ROOT-ASSETS"} {
		if _, ok := response.Occurrences[id]; ok {
			t.Fatalf("%s should be outside boundary: %#v", id, response.Occurrences)
		}
	}
}

func TestRunIncompleteScanFailsWithoutPruningExistingCache(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	a := filepath.Join(root, "a.org")
	b := filepath.Join(root, "b.org")
	writeFile(t, a, ":PROPERTIES:\n:ID: A\n:END:\n")
	writeFile(t, b, ":PROPERTIES:\n:ID: B\n:END:\n")
	req := catalogRequest(root, cacheDir)
	first := Run(context.Background(), req)
	if !first.OK {
		t.Fatalf("first run failed: %#v", first.Errors)
	}
	if err := os.Chmod(b, 0); err != nil {
		t.Fatal(err)
	}
	failed := Run(context.Background(), req)
	if err := os.Chmod(b, 0o600); err != nil {
		t.Fatal(err)
	}
	if failed.OK {
		t.Skip("chmod did not make the fixture unreadable on this filesystem")
	}
	if len(failed.Errors) == 0 || failed.Errors[0].Code != "incomplete_scan" {
		t.Fatalf("unexpected incomplete scan response: %#v", failed.Errors)
	}
	restored := Run(context.Background(), req)
	if !restored.OK {
		t.Fatalf("restored run failed: %#v", restored.Errors)
	}
	if restored.Stats.Parsed != 0 || restored.Stats.CacheHits != 2 {
		t.Fatalf("restored stats = %#v, want old cache preserved with two hits", restored.Stats)
	}
}

func TestRunIndexesManagedTaskAndDocumentBacklinks(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	tasks := filepath.Join(root, "tasks.org")
	writeFile(t, tasks, "\n* TODO Build\n:PROPERTIES:\n:ID: TASK-ID\n:END:\n산출물:\n- [[id:DOC-ID][Doc]]\nBody [[id:ORDINARY][Ordinary]]\n")
	writeFile(t, filepath.Join(root, "doc.org"), ":PROPERTIES:\n:ID: DOC-ID\n:END:\n\n#+TITLE: Doc\n\n* Link\n- [[id:TASK-ID][Build]]\n")
	req := catalogRequest(root, cacheDir)
	req.Scope.TasksFile = tasks
	response := Run(context.Background(), req)
	if !response.OK {
		t.Fatalf("catalog failed: %#v", response.Errors)
	}
	if len(response.Documents) != 1 {
		t.Fatalf("documents = %#v, want only project document, not tasks file", response.Documents)
	}
	if got := managedTargets(response.Links); !equalStrings(got, []string{"DOC-ID", "TASK-ID"}) {
		t.Fatalf("managed targets = %#v", got)
	}
	backlinks := req
	backlinks.Operation = "backlinks"
	backlinks.ID = "DOC-ID"
	backlinkResponse := Run(context.Background(), backlinks)
	if len(backlinkResponse.Links) != 1 || backlinkResponse.Links[0].SourceID != "TASK-ID" {
		t.Fatalf("backlinks response = %#v", backlinkResponse.Links)
	}
}

func TestRunAllowsCentralTasksFileOutsideNotesRootAndOverlay(t *testing.T) {
	project := t.TempDir()
	central := t.TempDir()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	tasks := filepath.Join(central, "tasks.org")
	writeFile(t, filepath.Join(project, "doc.org"), ":PROPERTIES:\n:ID: DOC-ID\n:END:\n\n#+TITLE: Doc\n")
	writeFile(t, tasks, "\n* TODO Central\n:PROPERTIES:\n:ID: TASK-DISK\n:END:\n산출물:\n- [[id:DOC-ID][Doc]]\n")
	req := catalogRequest(project, cacheDir)
	req.Scope.TasksFile = tasks
	response := Run(context.Background(), req)
	if !response.OK {
		t.Fatalf("central tasks run failed: %#v", response.Errors)
	}
	if _, ok := response.Occurrences["TASK-DISK"]; !ok {
		t.Fatalf("central task ID missing: %#v", response.Occurrences)
	}
	if len(response.Documents) != 1 || response.Documents[0].ID != "DOC-ID" {
		t.Fatalf("central tasks file became a document candidate: %#v", response.Documents)
	}

	req.Overlays = []Overlay{{Path: tasks, Text: "\n* TODO Central dirty\n:PROPERTIES:\n:ID: TASK-OVERLAY\n:END:\n산출물:\n- [[id:DOC-ID][Doc]]\n"}}
	overlay := Run(context.Background(), req)
	if !overlay.OK {
		t.Fatalf("central overlay run failed: %#v", overlay.Errors)
	}
	if _, ok := overlay.Occurrences["TASK-OVERLAY"]; !ok {
		t.Fatalf("central task overlay ID missing: %#v", overlay.Occurrences)
	}
	req.Overlays = nil
	disk := Run(context.Background(), req)
	if _, ok := disk.Occurrences["TASK-OVERLAY"]; ok {
		t.Fatalf("central task overlay persisted: %#v", disk.Occurrences)
	}
}

func catalogRequest(root, cacheDir string) Request {
	return Request{
		Version:   ProtocolVersion,
		Operation: "catalog",
		Scope: &Scope{
			ProjectID: "p",
			NotesRoot: root,
		},
		CacheDir: cacheDir,
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func managedTargets(links []Link) []string {
	var targets []string
	for _, link := range links {
		targets = append(targets, link.TargetID)
	}
	sort.Strings(targets)
	return targets
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
