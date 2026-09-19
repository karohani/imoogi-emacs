# Sync Report — Project Documentation Regeneration (2026-09-19)

Scope: SPEC-less project-mode sync. Regenerated `.moai/project/` (product / structure / tech + five codemaps) against HEAD `efa3567` on `main`, and added one clarifying sentence to `README.md`. Working tree only — no commit, no push, no branch.

## 1. Claim

| # | Claim |
|---|---|
| C1 | Eight documentation files under `.moai/project/` were rewritten to describe the repository at HEAD `efa3567`. |
| C2 | `README.md` received exactly one addition: three lines documenting `imoogi-project-notes-command` in the removable-root move section. No other README change. |
| C3 | Every directory-qualified repository path quoted in the regenerated docs exists on disk. |
| C4 | No stale path form (`modules/lsp/`, `modules/anki/`, `modules/flashcards/`, `modules/NN-`, `05-hydra`) survives in the regenerated docs. |
| C5 | The module inventory in the docs (30 boot entries = 29 numbered modules + `development/formatting`; 53 `.el` files under `modules/`) matches what `boot.el` and the filesystem actually contain. |
| C6 | The working tree contains only the files this task was permitted to touch, plus pre-existing untracked entries. |

## 2. Evidence

### Files written

| Path | Lines |
|---|---|
| `.moai/project/structure.md` | 252 |
| `.moai/project/tech.md` | 157 |
| `.moai/project/product.md` | 77 |
| `.moai/project/codemaps/overview.md` | 60 |
| `.moai/project/codemaps/modules.md` | 118 |
| `.moai/project/codemaps/dependencies.md` | 116 |
| `.moai/project/codemaps/data-flow.md` | 151 |
| `.moai/project/codemaps/entry-points.md` | 73 |
| `.moai/reports/sync-report-20260919.md` | this file |
| `README.md` | +4 / −1 lines (one hunk, net +3) |

### V1 — path existence (C3)

Extraction was run in three parts. The single-pattern form the task described over-matched: it also captured bare basenames used inside per-package tables (`00-defaults.el`) and Emacs *package* names (`project.el`, `straight.el`), which are not repository paths. The extraction was therefore split into V1a (directory-qualified paths and known top-level files — the actual repo-path claim), V1b (bare `.el` basenames, checked for resolution somewhere under `modules/`), and V1c (bare provenance manifest names).

```
$ grep -ohE '`[A-Za-z0-9_./-]+`' .moai/project/*.md .moai/project/codemaps/*.md | tr -d '`' \
 | grep -E '^(modules|cmd|internal|tests|docs|vendor|provenance|scripts|assets|templates|bin|tmux)/|^(Makefile|README\.md|ARCHITECTURE\.md|AGENTS\.md|CLAUDE\.md|boot\.el|early-init\.el|packages\.el|packages\.lock|go\.mod|go\.sum|go\.work|toolchains\.json|toolchains\.lock\.json|vendor-manifest\.json)$' \
 | sort -u > paths.txt
$ wc -l < paths.txt
      79
$ while read -r p; do [ -e "$p" ] || echo "MISSING: $p"; done < paths.txt
(no output)
```

```
$ # V1b — every bare .el basename quoted in the docs resolves under modules/
$ grep -ohE '`[A-Za-z0-9_.-]+\.el`' .moai/project/*.md .moai/project/codemaps/*.md | tr -d '`' | sort -u \
  | while read -r f; do
      case "$f" in boot.el|early-init.el|packages.el|project.el|straight.el) continue;; esac
      find modules -name "$f" | grep -q . || echo "UNRESOLVED: $f"
    done
(no output)
```

`boot.el`, `early-init.el` and `packages.el` are repository-root files (covered by V1a). `project.el` and `straight.el` are Emacs package names, not files in this repository — `project.el` is an Emacs built-in, `straight.el` appears only in the rejected-dependency list.

```
$ # V1c — bare provenance manifest names
$ for f in elpa.json fonts.json ghostel.json metadata.json tree-sitter.json toolchains.json sources.json; do
    [ -e "provenance/$f" ] || echo "MISSING provenance/$f"; done
(no output)
```

Re-run after this report was written (the report itself quotes repository paths):

```
$ # V1a repeated with .moai/reports/sync-report-20260919.md included — see §2 "Second pass" below
```

### V2 — stale-path check (C4)

```
$ grep -nE 'modules/(lsp|anki|flashcards)/|modules/[0-9]{2}-|05-hydra' \
    .moai/project/structure.md .moai/project/tech.md .moai/project/product.md .moai/project/codemaps/*.md
(no output)
$ echo "exit=$?"
exit=1
```

Exit 1 with no output is `grep`'s no-match result — the required outcome.

### V3 — module count (C5)

The command the task named counts *lines containing the literal string* `modules/`, which the `dolist` entries do not contain (they are package-relative, e.g. `"general/00-defaults"`; `boot.el` prepends `"modules/"` at load time). Its output is therefore 2, not 30:

```
$ grep -c 'modules/' boot.el
2
$ grep -n 'modules/' boot.el
38:;; 스크롤 등 편집 기본값은 modules/general/00-defaults.el 에서 통합 관리한다.
102:      (load (expand-file-name (concat "modules/" module) imoogi-emacs-dir))
```

Line 38 is a comment; line 102 is the `(concat "modules/" module)` call in the loader. Neither is a module entry. The count that actually corresponds to the load list:

```
$ grep -oE '"(general|project|org|development)/[a-z0-9-]+"' boot.el | wc -l
      32
$ grep -oE '"(general|project|org|development)/[a-z0-9-]+"' boot.el | tr '\n' ' '
"project/06-git" "project/06-git" "general/00-defaults" "general/01-keys" "development/formatting" "general/02-completion" "general/03-which-key" "project/04-projects" "general/05-transient" "project/06-git" "project/07-treemacs" "org/08-obsidian" "general/09-autorevert" "general/10-theme" "general/11-editing" "general/12-navigation" "general/13-system" "org/14-org" "org/23-org-preview" "org/15-markdown" "development/16-elisp" "development/17-lsp" "development/18-languages" "development/19-folding" "development/20-terminal" "general/21-native-compile" "project/22-tabs" "org/24-anki" "org/25-flashcards" "project/26-project-notes" "development/27-gptel" "org/28-clipboard"
```

32 matches, of which the first two `"project/06-git"` occurrences are inside the `imoogi-failed-modules` comment (it uses that name as an illustrative example of duplicate accumulation). 32 − 2 = **30 real entries** = 29 numbered modules + the unnumbered `development/formatting`, which is exactly what the regenerated `structure.md` and `codemaps/dependencies.md` tabulate.

```
$ find modules -name '*.el' | wc -l
      53
```

53 = 29 numbered + 1 unnumbered (`development/formatting.el`) + 9 `development/lang/` + 10 `org/anki/` + 4 `org/flashcards/`.

### Supporting measurements cited in the docs

```
$ go list ./... | wc -l
      34
$ ls tests/*.el | wc -l
      47
$ find . -name '*_test.go' -not -path './vendor/*' | wc -l
      47
$ git log --oneline --since=2026-08-22 | wc -l
      92
$ cat go.mod
module github.com/karohani/imoogi-emacs

go 1.26

require github.com/niklasfasching/go-org v1.9.1

require golang.org/x/net v0.38.0 // indirect
$ ls vendor/tree-sitter/*.dylib | wc -l
      17
$ git check-ignore -v bin
.gitignore:29:/bin/	bin
$ ls tests/*.el | wc -l ; ls tests/*-test.el | wc -l
      47
      44
$ grep -n 'executable-find' modules/org/28-clipboard.el modules/org/23-org-preview.el modules/project/26-project-notes.el
modules/org/28-clipboard.el:76:               (or (executable-find program)
modules/org/23-org-preview.el:234:         (program (or (and configured (executable-find configured))
modules/org/23-org-preview.el:237:    (if (and program (executable-find program))
modules/project/26-project-notes.el:1505:                   (or (executable-find program)
```

The three-caller resolution claim ("`PATH` first, then `bin/<name>`") is attributable to those four lines plus `modules/org/28-clipboard.el:78` and `modules/org/23-org-preview.el:233`, which build the `bin/<name>` fallback from `imoogi-emacs-dir`. The `user-error` naming a `make` target was read only at `modules/project/26-project-notes.el:1516`, so the docs attribute it to the note-move path alone.

### V4 — git status (C6)

```
$ git status --short          # captured BEFORE any edit in this task
 M CLAUDE.md
?? .claudeignore
?? .git_hooks/
?? .github/
?? .mcp.json
?? .moai-backups/
?? .moai/
?? .worktreeinclude
?? docs/study-learning-flow.html
?? todo.org

$ git status --short          # after
 M CLAUDE.md
 M README.md
?? .claudeignore
?? .git_hooks/
?? .github/
?? .mcp.json
?? .moai-backups/
?? .moai/
?? .worktreeinclude
?? docs/study-learning-flow.html
?? todo.org
```

The only delta is ` M README.md`. Every `.moai/project/` and `.moai/reports/` file this task wrote sits inside the pre-existing untracked `?? .moai/` entry, so it does not produce a separate status line. `CLAUDE.md` was already modified before this task began and was not touched.

```
$ git diff README.md
@@ -512,7 +512,10 @@
 이미 로컬에 만든 project/study note는 `C-c h p m v`로 외장 루트로 옮긴다.
-먼저 `make build-notes`로 `bin/imoogi-notes`를 빌드한다. 노트와 대상 외장 루트를
+먼저 `make build-notes`로 `bin/imoogi-notes`를 빌드한다. 이때 실행할 명령은
+`imoogi-project-notes-command` defcustom으로 정하며 기본값은 `("imoogi-notes")`다.
+Emacs는 이 이름을 PATH에서 먼저 찾고, 없으면 저장소 안의 `bin/imoogi-notes`를 쓴다.
+노트와 대상 외장 루트를
 선택하면 Emacs는 이 Go 프로그램을 호출한다. Go 프로그램이 같은 폴더 이름으로
```

The README hunk's factual content is read from `modules/project/26-project-notes.el` lines 34-37 (the `defcustom`) and lines 1503-1509 (the `executable-find` → `bin/imoogi-notes` resolution order).

## 3. Baseline-attribution

- **Tree state**: HEAD `efa3567de4e22858d4001f83798290aa6f3e2537`, dated 2026-09-19, branch `main`, primary checkout `/Users/jay/workspace/imoogi-emacs`. Measured with `git log -1 --format='%H %ad' --date=short`.
- **Pre-edit baseline**: `git status --short` and `git diff README.md` were captured before the first write of this task. The README baseline diff was empty (0 lines), so the post-edit diff shown above is attributable entirely to this task.
- **Content baseline**: every factual claim in the regenerated docs was read from the tree at this HEAD — `boot.el`, `ARCHITECTURE.md` § 모듈 패키지 경계, `docs/refactoring/module-packages.md`, `docs/clipboard-platform-support.md`, `Makefile`, `tests/run.sh`, `go.mod`, `toolchains.json`, `provenance/sources.json`, `vendor-manifest.json`, each `cmd/*/main.go`, and each module's header comment and `imoogi-require` line. The previous `.moai/project/` files (written 2026-08-22) were read for wording reuse only, not treated as a source of fact.
- **Verification baseline**: all four verification outputs above were produced in this run, against this tree. V1 was run twice — once over the eight docs, once again after this report was written (below).
- **Superseded facts** from the 2026-08-22 docs, all re-measured and corrected rather than carried forward: "zero third-party Go dependencies / no go.sum" (false — `go-org` is a direct dependency and `go.sum` exists), "no Makefile anywhere in the repository" (false), "22 numbered modules in a flat `modules/`" (false), "`modules/lsp/` exception" (false — now `modules/development/lang/`), "10 Go packages" (false — 34), "4 test files" (false — 47), "a single Go CLI `imoogi-toolchain`" (false — six).

## 4. Gaps (not verified)

1. **`make test` / `make ci-local` were not executed.** This task changed no `.el`, `.go`, or test file, so no test run was performed. The regenerated docs therefore describe the test toolchain from `Makefile` and `tests/run.sh` source, not from an observed passing run. The most recent recorded run is the one in `docs/refactoring/module-packages.md` (2026-09-19: 392 ERT tests, 390 passed, 2 GUI skipped) — cited there, not re-measured here.
2. **`internal/` package purposes are partly inferred.** Only `internal/anki/*` carries `// Package` doc comments. For `activation`, `artifact`, `cli`, `clipboard`, `config`, `fetch`, `lang`, `notemove`, `orgpreview`, `provenance` and `setup`, the one-line purposes in `codemaps/modules.md` were derived from the `cmd/*/main.go` wiring and the file names, not read from a doc comment. They are plausible but unattested.
3. **Feature-history attribution was not walked commit by commit.** `git log --since=2026-08-22 | wc -l` returned 92 and the features named in the docs were confirmed by file existence at HEAD, but no per-commit reading established which commit introduced which feature.
4. **The `IMOOGI_ANKI_LOG` mechanism was confirmed structurally, not behaviourally.** `modules/org/anki/imoogi-process.el:63` sets the variable from `imoogi-anki-log-file` and `cmd/imoogi-anki/log.go:13` reads it; no sync run was executed to observe a log file being written.
5. **The removal of the three ELPA archive-index components (stated in `tech.md`) was taken from the task brief**, corroborated only by their absence from the current `provenance/sources.json` component list. No diff of that file against a previous revision was inspected.
6. **`.github/workflows/label-sync.yml` exists on disk but `.github/` is untracked**, so whether that workflow is active for this repository on the remote was not determined.
7. **Platform scope beyond macOS arm64 was read from documents, not exercised.** `docs/clipboard-platform-support.md` itself states that macOS native GUI acceptance is incomplete and that Windows/Linux cells are design gates only; the docs repeat that framing rather than asserting support.
8. **`bin/` currently holds all six binaries, but it is git-ignored** (`.gitignore:29`), so a fresh clone will not have them. The docs phrase `bin/<name>` as build output and as the fallback resolution path, not as a guaranteed-present file.

## 5. Residual risk

- **Doc drift resumes immediately.** These files are a snapshot of `efa3567`. The module reorganization landed the same day, so any follow-up commit that renames or moves a module invalidates `structure.md`, `codemaps/modules.md` and `codemaps/dependencies.md` together. The stale-path grep in V2 is the cheap regression check to re-run.
- **Package lists are unversioned by choice.** `tech.md` names packages without pinning versions, because `packages.lock` is explicitly a non-authoritative audit trail and `vendor/elpa/` is the real lock. This keeps the doc from going stale on every re-vendor, at the cost of not answering "which version is installed" — that question must go to `vendor/elpa/`.
- **The inferred `internal/` purposes (Gap 2) could be subtly wrong** in a way no check in this run would catch, since nothing compiles or tests against prose. If precision there matters, adding `// Package` doc comments to the eleven packages would make the next regeneration attributable rather than inferred.
- **The README addition is prose in a 1090-line Korean document.** It was placed to match the surrounding register and verified by diff, but no native-reader review was performed, and no rendering of the README was inspected.
- **V1's path extraction is regex-based.** It covers backtick-quoted tokens; a path written without backticks, or inside a fenced directory-tree block (the large tree in `structure.md` is such a block), is not covered by V1a. The tree entries were written from `ls` output of the live tree but are not mechanically re-verified.

## Second pass — V1a including this report

```
$ grep -ohE '`[A-Za-z0-9_./-]+`' .moai/project/*.md .moai/project/codemaps/*.md .moai/reports/sync-report-20260919.md \
  | tr -d '`' | grep -E '^(modules|cmd|internal|tests|docs|vendor|provenance|scripts|assets|templates|bin|tmux)/|^(Makefile|README\.md|ARCHITECTURE\.md|AGENTS\.md|CLAUDE\.md|boot\.el|early-init\.el|packages\.el|packages\.lock|go\.mod|go\.sum|go\.work|toolchains\.json|toolchains\.lock\.json|vendor-manifest\.json)$' \
  | sort -u | while read -r p; do [ -e "$p" ] || echo "MISSING: $p"; done
```

Result recorded below.
```
      87
MISSING: modules/anki/
MISSING: modules/flashcards/
MISSING: modules/lsp/
MISSING: modules/NN-
```

All four "missing" entries are the *stale* path forms quoted verbatim in this report's own C4 claim and V2 command. They are deliberately non-existent — that is what C4 asserts. Excluding them, all 83 remaining paths across the eight docs and this report exist:

```
$ ... | grep -vE '^modules/(anki|flashcards|lsp)/$|^modules/NN-$' \
      | while read -r p; do [ -e "$p" ] || echo "MISSING: $p"; done
(no output)
```

V2 is deliberately not run over this report: the report must quote the forbidden patterns in order to document the check.

## Final state

```
$ git status --short
 M CLAUDE.md
 M README.md
?? .claudeignore
?? .git_hooks/
?? .github/
?? .mcp.json
?? .moai-backups/
?? .moai/
?? .worktreeinclude
?? docs/study-learning-flow.html
?? todo.org
```

No commit, no push, no branch created. `CLAUDE.md` and the untracked entries are pre-existing and untouched by this task.

## Post-review corrections (same run)

Three evidence-accuracy corrections were applied after an independent review of this report:

1. The `README.md` row in §2 read `+3 lines / -1 line`; the diff adds 4 lines and removes 1 (net +3). Corrected to `+4 / −1`.
2. `codemaps/overview.md`, `codemaps/dependencies.md` and `codemaps/entry-points.md` generalized "PATH first, then `bin/<name>`, then a `user-error` naming the make target" to all Emacs→Go callers. The resolution half was then verified for all three callers (grep output added to §2); the `user-error` half was verified only for the note-move path, so the three files now attribute it there alone.
3. `structure.md` and `tech.md` said "47 test files under `tests/`". The orchestrator re-measured after this report was drafted: `find tests -name "*.el"` = 50 and `*-test.el` = 45, all tracked by git (the drafter's 47/44 undercounted `tests/benchmarks/` and one test file). Both files now say 50 `.el` files of which 45 are `*-test.el` cases. `docs/refactoring/module-packages.md` records 49 test sources as of the module reorganization — a different measurement (it counts the pre-move source set), not reconciled here.
4. `tech.md` § Provenance offered `bin/` as the example of what the git-ignore skip protects. `bin/` is not under any `roots` entry in `provenance/sources.json`, so the skip is not what protects it. The example was dropped; the sentence now states the skip without an unverified illustration.

A fifth item was noted and deliberately not acted on: the README hunk leaves `노트와 대상 외장 루트를` as a short line before `선택하면…`. It renders as one soft-wrapped paragraph. Rewrapping would touch line 517, which is outside this task's one-addition scope — flagged for the orchestrator instead.
