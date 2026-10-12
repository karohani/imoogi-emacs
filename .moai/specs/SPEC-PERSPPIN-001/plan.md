# Implementation Plan — SPEC-PERSPPIN-001

Card t35 (Class C, feature). Tier M: `spec.md`, `plan.md`, `acceptance.md`,
`progress.md`. Development mode: TDD (RED first per AC). Baseline `6feee71`
(contains t34 `4285b31`). Work happens only in `.moai/worktrees/t35`
(branch `WT-persp-full-pin`).

## § A Context

### A.1 What already exists

- `imoogi-project-perspective-alist` (`04-projects.el:9`): canonical root →
  Perspective name, written by `imoogi-project-perspective-name`, persisted by
  savehist (`00-defaults.el:124`).
- `imoogi-project--perspective-root` (`04-projects.el:81`): name → root, with
  `file-directory-p` then `file-truename` — note it does file I/O before any
  remote check, so the finder must not call it unguarded (REQ-PIN-006).
- `imoogi-project-context-root` (`04-projects.el:93`): Perspective-first, then
  `project-current nil`. Used by `f`/`s`/`d` via `imoogi-project-context-call`.
- `imoogi-treemacs--perspective-project-root` (`07-treemacs.el:554`): its own
  `rassoc` fallback; untouched.
- t34 fallback in `imoogi-project-notes--current-entry`
  (`26-project-notes.el` ~1379): only when `project-current` is nil.
- `project.el` 0.11.1 facts used here: `project-find-functions` is a run-until-
  success hook; `project--find-in-directory` absorbs only `permission-denied`;
  `project-current` passes the prompted directory back through the same hook;
  `project-try-vc` caches **positive** results only (a non-VC directory is
  re-searched on every call); `project--read-project-buffer` filters
  by `project-buffers`; `project-forget-project` calls `project-prompter`
  directly; Eglot binds `eglot-lsp-context` to t around its lookup.
- `perspective.el`: `persp-current-name` → `persp-curr` → `persp-mode-guard`,
  which signals when `persp-mode` is off.
- `~/project-notes` is not a git repository (checked 2026-10-13: no `.git`), so
  notes buffers are projectless today and would reach the finder without
  REQ-PIN-008.

### A.2 Other `project-current` callers affected by the finder

| Caller | Effect in a mapped Perspective, projectless buffer | Disposition |
|--------|------------------------------------------------------|-------------|
| `imoogi-tab-name` (`22-tabs.el:54`) | tab shows the project name instead of the buffer name | accepted (consistent with pinning) |
| `imoogi-org-preview--project-root` (`23-org-preview.el:428`) | `~/notes/scratch.org` preview allowed roots widen to include R | accepted, recorded risk B.3 |
| `imoogi-project-notes--project-root` (`26-project-notes.el:527`) | setup from scratch stops prompting for the source folder and uses R | accepted (the Perspective answers it) |
| `imoogi-project-notes--current-entry` | resolves via `project-current` branch instead of the t34 fallback; same entry | REQ-PIN-014 |
| `imoogi-project-notes--current-source-root` | notes buffers: unchanged because of REQ-PIN-008 | guarded |
| `imoogi-treemacs--perspective-project-root` | first branch now answers; same root | no change needed |
| Eglot | declined via `eglot-lsp-context` | REQ-PIN-005 |
| `consult--default-project-function` (`C-x p b`) | stops prompting from scratch | accepted; notes union out of scope |

### A.3 PRESERVE list

- `tests/project-notes-perspective-test.el` (4 tests) — unmodified.
- `tests/workspace-bridge-test.el` switch-perspective tests (unmodified) and the
  two context tests (assertions unmodified; stub moved per A.4).
- `tests/transient-menu-test.el` rows other than `b`.
- `imoogi-project-context-root` semantics (Perspective-first).

### A.4 Explicitly amendable tests

- `tests/transient-menu-test.el:80` row `(project-switch-to-buffer transient--do-exit)`
  becomes the new `b` command with `transient--do-exit`.
- `tests/workspace-bridge-test.el:141`
  (`imoogi-project-context-root-follows-perspective-mapping`) stubs
  `project-find-file` and expects it to be called. D-4 makes `f` call
  `project-find-file-in` instead, so the stub **moves to `project-find-file-in`**
  (a lambda accepting `(suggested dirs project &optional include-all)` that
  records `default-directory`). The assertion stays: the recorded directory
  equals the context root before and after the Perspective switch.
  `imoogi-project-context-falls-back-to-project-el` (`:160`) gets the same stub
  move. No other assertion in either test changes.

## § B Design Decisions

Ordered by change likelihood (highest first).

### D-1 — Precedence asymmetry in the transient is preserved, not unified

`f`/`s`/`d` are Perspective-first (existing behaviour of
`imoogi-project-context-root`, stated in its docstring; the workspace-bridge
tests stub that function, so they pin "follows the context root", not the
precedence inside it); `b`/`c`/`k` follow
`project.el` (own project first, Perspective finder last). In Perspective A, a
buffer of project B gets A's files from `f` and B's buffers from `b`.
Rationale: unifying in either direction regresses something — making `f`
own-project-first breaks the tested "Perspective survives buffer restoration"
behaviour; making `b`/`c`/`k` Perspective-first contradicts the t34
card-review lesson the card itself carries. This card's goal (no prompt from
projectless buffers, pinned candidate lists) is met without choosing.

**Decided (lane-3, 2026-10-13):** keep the asymmetry as written — `f`/`s`/`d`
stay Perspective-first, `b`/`c`/`k` stay own-project-first. Why: unifying in
either direction regresses a tested behaviour or t34's card-review lesson, and
the card's goal (no prompt from projectless buffers, pinned candidate lists) is
met without unifying. Unification is out of scope (spec.md § 5).

### D-2 — Mechanism: a guarded finder at the end of `project-find-functions`

Chosen over (a) a custom `project-prompter` — rejected because
`imoogi-project-switch-perspective`, `project-switch-project`, and
`project-forget-project` call the prompter directly, so a prompter that answers
with the Perspective root would silently select/forget the current project;
and (b) per-command wrappers binding `project-current-directory-override` —
rejected as the primary mechanism because it covers only the commands wrapped
and leaves `C-x p *`, consult, and the tab name prompting/unpinned. The finder
is installed with `add-hook` at **depth 95**, strictly above 90. `add-hook … t`
(APPEND) means depth 90, and at equal positive depth a newer function sorts
*after* an older one (subr.el `add-hook`). So depth 90 would let a later APPEND
land after the finder, while 95 keeps every later default-depth or APPEND
member before it (AC-PIN-003). 95 rather than 100 follows the `add-hook`
docstring's advice against 100 and leaves room above for a deliberately-last
member. Members added later at a depth above 95 are not covered (spec.md
REQ-PIN-002).

Guards, evaluated cheapest first, each a decline:
1. inhibit variable bound (explicit selection) or re-entry;
2. `(bound-and-true-p eglot-lsp-context)` — Eglot is `:commands`-deferred, so
   the variable may be unbound when the finder runs;
3. `project-current-directory-override` non-nil, or DIR ≠ the current buffer's
   `default-directory` (compared after `expand-file-name` +
   `file-name-as-directory`), or DIR is remote (`file-remote-p` on the string —
   no connection is opened);
4. `persp-mode` not active;
5. no alist entry for `persp-current-name`;
6. raw mapped root `file-remote-p`;
7. mapped root not `file-directory-p`;
8. notes-context buffer (predicate supplied by `26-project-notes.el` through a
   hook variable owned by `04-projects.el`, so 04 never requires 26). The
   predicate short-circuits on the buffer-local variables
   `imoogi-project-notes-source-root`, `imoogi-project-notes-directory-local`,
   `imoogi-project-notes-entry-instance-id`, then on `buffer-file-truename`
   (nil → not a notes file). It then compares the expanded truename against
   the in-memory list of truename notes directories (D-7), so a symlinked
   `~/project-notes` still matches. The predicate never calls
   `--all-entries`, `--read-registry`, or `--find-metadata-file` itself
   (AC-PIN-010, second part).
Then return the memoized project for the pinned root (D-8). On a memo miss,
resolve the canonical root with the finder inhibited:
`project--find-in-directory` on the root, else `(transient . ROOT)`. The whole
body is wrapped so any error declines (REQ-PIN-007).

### D-3 — `b` gets an Imoogi command; `project-buffers` is not advised

A global `project-buffers` advice would make `k` kill notes buffers and leak
notes into every consumer (REQ-PIN-012). The new `b` command resolves the
project (`project-current nil`, else the Perspective root as in t34, else the
normal prompt), builds the set {`project-buffers` minus ignore conditions} ∪
{live buffers visiting files under the root's notes directory}, and reads with
`read-buffer` using that predicate, then `switch-to-buffer`. The t34-style
fallback inside the command makes `b` work from notes-context buffers too.

### D-4 — `f` gains the notes directory as an extra search root

**Decided (audit r1 D4): one code path.** With `default-directory` at the
context root (as today), `f` resolves the project and calls
`project-find-file-in` with `dirs` = (root) or (root, notes-dir) when the root
has a notes entry, always through this single entry point (it no longer calls
`project-find-file`). `project-files` is untouched. Notes files come in only
through the extra `dirs` element, and VCS metadata directories under the notes
dir are excluded because, for a directory outside the root,
`project--dir-ignores` falls back to the default `project-ignores`
(`vc-directory-exclusion-list` + `grep-find-ignored-files`), which
`project--files-in-directory` passes to `find`. The suggested filename keeps `project-find-file`'s list (buffer file
name, `thing-at-point 'filename`). Consequence: the two workspace-bridge stubs
move per A.4.

Rejected: `project-or-external-find-file` with
`project-vc-external-roots-function`, because it only works for VC projects
(transient projects have no external-roots hook). Also rejected: keeping
`project-find-file` for the no-notes case, because two code paths would be
exercised by different tests for no gain.

### D-5 — t34 fallback retained

With the finder installed, the `project-current` branch of `--current-entry`
answers for scratch, so the fallback is normally unreachable. It stays as
defence in depth for inhibited/declined lookups and is pinned by a new test
(AC-PIN-016). Removing it would be a no-gain change to a just-reviewed function.

### D-6 — Notes-context buffers declined

Claiming them would redirect `--current-source-root` (which consults
`project-current` before the buffer-local source root) to the Perspective root,
which differs when the notes buffer was opened from a worktree. Cost: `c`/`k`
from a notes buffer still prompt (out of scope); `b`/`f` work there through D-3
and the Perspective-first context.

**Decided (lane-3, 2026-10-13):** accept the gap — `C-c h p c` and `k` still
prompt from a project-notes buffer, and the finder keeps declining notes
buffers. Why: claiming them would redirect `--current-source-root` for
worktree-opened notes, and the card names projectless buffers (scratch), not
notes buffers. Recorded as a known limitation (acceptance.md § Residual Risk)
and an out-of-scope item (spec.md § 5).

Consequence stated (audit r1 D5, does not reopen D-6): a study-note
Perspective (`imoogi-project-notes--open-study-workspace`,
`26-project-notes.el:2615`) is mapped to the notes directory itself. So every
note file of a study project is a notes-context buffer, and `c`/`k` prompt from
all of them. Only projectless buffers (scratch, help) are pinned in a study
Perspective. Recorded in spec.md § 5 and acceptance.md § Residual Risk.

### D-7 — Notes-directory list: concrete refresh trigger points

**Decided (lane-3, 2026-10-13).** The in-memory list (canonical notes
directories of every entry `imoogi-project-notes--all-entries` returns) is
rebuilt eagerly, never lazily on the finder's path, at exactly these points,
all in `modules/project/26-project-notes.el`:

1. **Registry write** — at the end of `imoogi-project-notes--write-registry`
   (line ~792). Its callers at `6feee71` are `imoogi-project-notes--save-entry`
   (~1845, used by setup/setup-study/move/repair/copy) and
   `imoogi-project-notes--remove-local-entry` (~3701).
2. **Mounted-state write** — at the end of
   `imoogi-project-notes--write-mounted-state` (line ~863). Its callers are
   `imoogi-project-notes-raiseup` (~2775), `-reconnect-source` (~3068),
   `-clear-source-override` (~3084), `-mounted-root-add` (~3191),
   `-mounted-root-edit` (~3234), and `-mounted-root-remove` (~3248).
3. **Explicit rescan** — at the end of
   `imoogi-project-notes-mounted-root-refresh` (line ~3252), which rescans
   mounted roots without writing state.
4. **Session detach** — at the end of `imoogi-project-notes-detach` (line
   ~3818), which changes `imoogi-project-notes--detached-instance-ids` without
   writing a file.
5. **Unmount** — after the executor succeeds in
   `imoogi-project-notes-unmount-device` (line ~3920), which does not write
   state either.
6. **Perspective switch** — a function on `persp-switch-hook` (alongside the
   members added at `04-projects.el:339`; registered from 26 under
   `with-eval-after-load 'perspective`), which also picks up notes directories
   created or mounted outside Emacs.
7. **Startup** — once on `emacs-startup-hook` at a depth after
   `imoogi-perspective-state-restore` (which runs at −50).

Each rebuild calls `--all-entries` inside `condition-case`; on any error
(corrupt registry, unreadable mount) the previous list is kept and nothing is
displayed. Before the first rebuild (list never built), the predicate treats a
file as a notes file when it is inside `imoogi-project-notes-directory` — a
pure string-prefix check with no I/O. Rebuilds happen at the call sites (a
direct call at the end of each function in 1-5), not via advice. Trigger 6 is a
named function, so the test can call it directly without running the other
`persp-switch-hook` members. Trigger 7 cannot run in the batch runner
(`tests/run.el` exits inside `-l`, so `emacs-startup-hook` never fires) and is
verified by registration only (AC-PIN-010, third part).

### D-8 — Memoize the finder's answer per pinned root

**Decided (audit r1 D6).** `project-try-vc` caches only positive results, so a
non-VC mapped root (the study-note case) would repeat a `locate-dominating-file`
walk and a `hack-dir-local-variables` pass on every tab-bar redisplay. The
finder therefore keeps a one-slot memo `(KEY . PROJECT)`, where KEY is
(Perspective name, raw mapped root string from the alist):

- **Hit**: KEY equals the current (name, raw root). Guards 1-8 still run first
  on every call, including the `file-directory-p` of guard 7, so a deleted root
  still declines.
- **Miss**: resolve as in D-2 and store.
- **Invalidation**: KEY no longer matching (a remap or alist change) invalidates
  it implicitly, and a named function on `persp-switch-hook` clears it
  explicitly (the same hook as D-7 trigger 6).

Accepted staleness: running `git init` in a pinned non-VC root keeps the
transient project until the next Perspective switch. Chosen over option (b)
(accepting the uncached search) because the finder runs on redisplay.

### B.3 Risks

### B.3 Risks

- Stale mapping to a deleted root → declined by guard 7 (AC-PIN-005c).
- Remote root → declined by guard 6 before I/O; remote lookup directory (a
  TRAMP buffer in a local Perspective) → declined by guard 3 (AC-PIN-008).
- Study-note Perspectives: `c`/`k` prompt from every note file (D-6
  consequence, accepted).
- savehist-restored alist with duplicate names → first match wins (existing).
- Hot path: `project-current` runs on tab-bar redisplay. Guards 1-6 are pure
  Lisp, guard 7 is one `file-directory-p`, guard 8 is buffer-local plus an
  in-memory prefix comparison, and the resolved project comes from the D-8
  memo. Without the memo a non-VC root would be re-searched on every call,
  because `project-try-vc` caches only positive results.
  `imoogi-project-notes--all-entries` is NOT cached (it
  re-scans metadata, the registry JSON, and mounted roots each call), which is
  why guard 8 must not call it. A notes directory created outside Emacs (or a
  mount appearing without `--write-mounted-state`) is unknown until the next
  Perspective switch (D-7 trigger 6); until then the finder may claim a buffer
  visiting it — acceptable, since the effect is limited to that one buffer.
- Org preview allowed roots widen for `~/notes/scratch.org` in a mapped
  Perspective (the preview server may resolve links under R).
- Prompted directory equal to the buffer's `default-directory`: inside
  `project-current`, the prompt is reached only after the first lookup on that
  same directory failed, i.e. the finder already declined in the same state, so
  it declines again. The one caller that resolves a selected root outside
  `project-current`'s prompt branch, `imoogi-project-switch-perspective`
  (`(project-current t selected-root)`), binds the inhibit (AC-PIN-006c).

## § C Pre-flight

### C.1 Baseline

```bash
EMACS=/Applications/Emacs.app/Contents/MacOS/Emacs IMOOGI_ERT_SELECTOR='\(project-notes\|transient\|imoogi-project-\|perspective\|workspace\)' ./tests/run.sh
```
Record the count and the skip set before any change.

### C.2 Sandbox macro

`imoogi-persppin-test--isolated`: extends the t34 macro
(`tests/project-notes-perspective-test.el`) with `project-list-file`,
`project--list` let-bound to nil, `persp-mode` let-bound to t,
`project-find-functions` rebuilt by the installer, the D-8 memo and the D-7
notes-directory list reset, and prompt stubs that signal. The test file
declares `(defvar eglot-lsp-context)` so its `let` binding is dynamic under
`lexical-binding: t`. Stubs for `compile` are interactive lambdas, because
`project-compile` uses `call-interactively`.
Git-backed tests use `(skip-unless (executable-find "git"))` and
`call-process "git" nil nil nil "init" "-q"` inside the sandbox.

## § D Constraints

- No `project-files` / `project-buffers` method or advice.
- Nothing on the finder's path reads the notes registry, folder metadata, or
  mounted roots (REQ-PIN-008).
- No new package dependency.
- 04-projects.el does not `require` 26-project-notes.el.
- No edits to `.moai/reports/**`, CHANGELOG, or agent files in this SPEC's run.

## § E Self-Verification

- Every AC test fails before its milestone's change (RED evidence in
  progress.md §E.2) and passes after.
- The four t34 tests pass unmodified.
- The two named workspace-bridge tests pass with only the A.4 stub move
  (`project-find-file` → `project-find-file-in`). Their assertions are
  unchanged.

## § F Milestones

### M1 — Finder and its guards (highest change likelihood)
AC-PIN-001..009. Installer at depth 95, guards D-2, D-8 memo, inhibit in
`imoogi-project-switch-perspective`.

### M2 — Notes-context exclusion and t34 compatibility
AC-PIN-010, AC-PIN-016. Predicate hook from 26-project-notes.el; the D-7
notes-directory list and its seven refresh points.

### M3 — `b` command and candidate set
AC-PIN-011 (b part), AC-PIN-012, AC-PIN-017 (b part); transient row change and
the amended `transient-menu-test.el` row.

### M4 — `f` notes files
AC-PIN-013, AC-PIN-017 (f part); D-4; the A.4 workspace-bridge stub move.

### M5 — Non-membership and unchanged entries (most mechanical)
AC-PIN-011 (c/k part), AC-PIN-014, AC-PIN-015; full selector run (Quality Gate).

## § G Anti-Patterns

- Advising `project-current` or `project-buffers` globally.
- Replacing `project-prompter`.
- Calling `imoogi-project--perspective-root` from the finder before the remote
  check.
- Letting the finder claim a lookup whose directory is not the buffer's own.
- Writing to the real `~/.emacs.d/projects` from tests.

## § H Cross-References

- `.moai/reports/t34/progress.md` § Gaps / residual risk.
- `modules/project/04-projects.el:81-136`, `26-project-notes.el` ~1344-1400,
  `05-transient.el:314`, `22-tabs.el:54`, `23-org-preview.el:428`,
  `07-treemacs.el:554`.
- `project.el` 0.11.1 (`project-current`, `project--find-in-directory`,
  `project--read-project-buffer`, `project-try-vc`).
- SPEC-TRANSIENT-001 (transient layout).
