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
  `project-try-vc` caches per directory; `project--read-project-buffer` filters
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
- `tests/workspace-bridge-test.el` switch-perspective and context tests.
- `tests/transient-menu-test.el` rows other than `b`.
- `imoogi-project-context-root` semantics (Perspective-first).

### A.4 Explicitly amendable tests

- `tests/transient-menu-test.el:80` row `(project-switch-to-buffer transient--do-exit)`
  becomes the new `b` command with `transient--do-exit`.
- `tests/workspace-bridge-test.el:141` stubs `project-find-file` and expects it
  to be called. If the `f` notes inclusion is implemented by delegating to a
  different `project.el` entry point (D-4 option ii), that stub changes to the
  new entry point; the assertion (`default-directory` follows the Perspective)
  stays.

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

[NEEDS CLARIFICATION: unify transient precedence — should `C-c h p` use one
regime for all entries, and if so which (Perspective-first like `f`, or
own-project-first like `b`)?]

### D-2 — Mechanism: a guarded finder at the end of `project-find-functions`

Chosen over (a) a custom `project-prompter` — rejected because
`imoogi-project-switch-perspective`, `project-switch-project`, and
`project-forget-project` call the prompter directly, so a prompter that answers
with the Perspective root would silently select/forget the current project;
and (b) per-command wrappers binding `project-current-directory-override` —
rejected as the primary mechanism because it covers only the commands wrapped
and leaves `C-x p *`, consult, and the tab name prompting/unpinned. The finder
is installed with `add-hook` at the maximum depth (90 or higher) so later
`add-hook` calls at default depth or with APPEND still sort before it
(AC-PIN-003).

Guards, evaluated cheapest first, each a decline:
1. inhibit variable bound (explicit selection) or re-entry;
2. `eglot-lsp-context` non-nil;
3. `project-current-directory-override` non-nil, or DIR ≠ the current buffer's
   `default-directory` (compared after `expand-file-name` +
   `file-name-as-directory`);
4. `persp-mode` not active;
5. no alist entry for `persp-current-name`;
6. raw mapped root `file-remote-p`;
7. mapped root not `file-directory-p`;
8. notes-context buffer (predicate supplied by `26-project-notes.el` through a
   hook variable owned by `04-projects.el`, so 04 never requires 26). The
   predicate short-circuits on the buffer-local notes variables, then on
   `buffer-file-name` (nil → not a notes file), then compares the file against
   an in-memory list of notes directories. That list is built from
   `--all-entries` off the hot path (first use, and refreshed when the
   registry or mounted state is written, or on a Perspective switch); the
   predicate never calls `--all-entries`, `--read-registry`, or
   `--find-metadata-file` itself (AC-PIN-010, second part).
Then resolve the canonical root with the finder inhibited:
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

Options for the run phase (pick the one that keeps `project-files` untouched):
(i) call `project-find-file-in` with `dirs` = (root, notes-dir) and the
resolved project; (ii) `project-or-external-find-file` with
`project-vc-external-roots-function` let-bound to return the notes directory.
Either must exclude VCS metadata in the notes dir and must not mark notes as
project files. Option (i) is preferred because (ii) is VC-project-only
(transient projects have no external-roots hook).

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

[NEEDS CLARIFICATION: notes-buffer prompt gap — accept that `C-c h p c` and `k`
still prompt when invoked from a project-notes buffer, or extend the finder to
claim notes buffers whose entry key equals the mapped root's identity key
(accepting the worktree source-root redirection)?]

### B.3 Risks

- Stale mapping to a deleted root → declined by guard 7 (AC-PIN-005c).
- Remote root → declined by guard 6 before I/O (AC-PIN-008).
- savehist-restored alist with duplicate names → first match wins (existing).
- Hot path: `project-current` runs on tab-bar redisplay; guards 1-6 are pure
  Lisp, guard 7 is one `file-directory-p`, guard 8 is buffer-local plus an
  in-memory prefix comparison, and the VC lookup on the root is cached by
  `project-try-vc`. `imoogi-project-notes--all-entries` is NOT cached (it
  re-scans metadata, the registry JSON, and mounted roots each call), which is
  why guard 8 must not call it. A stale notes-dir list can misclassify a notes
  directory registered seconds ago until the next refresh — acceptable, since
  the effect is only that the finder claims or declines that one buffer.
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
`project-find-functions` rebuilt by the installer, and prompt stubs that signal.
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
- The four t34 tests and the two named workspace-bridge tests pass unmodified.

## § F Milestones

### M1 — Finder and its guards (highest change likelihood)
AC-PIN-001..009. Installer, guards D-2, inhibit in
`imoogi-project-switch-perspective`.

### M2 — Notes-context exclusion and t34 compatibility
AC-PIN-010, AC-PIN-016. Predicate hook from 26-project-notes.el.

### M3 — `b` command and candidate set
AC-PIN-011 (b part), AC-PIN-012, AC-PIN-017 (b part); transient row change and
the amended `transient-menu-test.el` row.

### M4 — `f` notes files
AC-PIN-013, AC-PIN-017 (f part); D-4.

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
