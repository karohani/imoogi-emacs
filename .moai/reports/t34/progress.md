# t34 — perspective not pinned to its project

- card: t34 (Class B — defect, reproduction first; leader dispatch, factory `assigned` → lane-4)
- worktree: .moai/worktrees/t34, branch WT-persp-project-pin (from origin/main 7d98ec0)
- scope stop: worktree commit only — no push, PR, or main merge

## Cause (established by reproduction)

Key path: `C-h p m r` = `imoogi-transient-project` "m" → project-notes menu "r" →
`imoogi-project-notes-create-artifact`. Its interactive form calls
`imoogi-project-notes--current-entry` and, on nil, `--select-entry "산출물 프로젝트: "` —
the project picker the card reports.

`C-h n` opens `~/notes/scratch.org` (not a git repo, verified `ls ~/notes/.git` → absent),
so `--current-entry` has no metadata, no buffer-local root, no notes file, and
`project-current` is nil → nil. It never consulted the Perspective→project mapping
(`imoogi-project-perspective-alist`, persisted by savehist,
modules/general/00-defaults.el:124; resolved by `imoogi-project--perspective-root`,
modules/project/04-projects.el:81), even though other Imoogi project commands
(`imoogi-project-context-root`) already prefer it.

Reproduction (RED, before the fix):

    EMACS=/Applications/Emacs.app/Contents/MacOS/Emacs IMOOGI_ERT_SELECTOR='imoogi-project-notes-\(scratch-resolves\|unmapped-perspective\|buffer-project-beats\)' ./tests/run.sh
    exit=1
    (ert-test-failed ((should entry) :form entry :value nil))
    FAILED  imoogi-project-notes-scratch-resolves-perspective-project
    Ran 3 tests, 2 results as expected, 1 unexpected

## Fix

`imoogi-project-notes--current-entry` (modules/project/26-project-notes.el ~:1332): when
the buffer has no project at all (`project-current` nil), fall back to the entry of the
current Perspective's mapped root. A buffer that has its own project — registered or not —
never borrows the Perspective's notes (card-review finding, see below).

Tests: tests/project-notes-perspective-test.el (new file, own sandbox macro; kept out of
tests/project-notes-test.el to avoid conflicting with t23):
- scratch in a pinned Perspective resolves to that project (the reproduction)
- unmapped Perspective still yields no entry (no guessing)
- a buffer of another registered project keeps its own project
- a buffer of an unregistered project does not borrow the Perspective's entry

GREEN (lane, after the review fix):

    EMACS=/Applications/Emacs.app/Contents/MacOS/Emacs IMOOGI_ERT_SELECTOR='\(project-notes\|transient\|imoogi-project-\|perspective\)' ./tests/run.sh
    exit=0
    Ran 271 tests, 264 results as expected, 0 unexpected, 7 skipped (2026-10-12 22:10:06+0900)

(7 skips = the `cli-cache-*` tests, skipped on baseline too.)

## Card-review (advisory, codex_review scope=uncommitted)

| Round | Verdict | Finding | Disposition |
|---|---|---|---|
| 1 | fail | P2: a buffer of unregistered project B in Perspective A silently resolved to A, so `--entry-or-setup` skipped setting up B | fixed: fallback only when `project-current` is nil; test added |
| 2 | pass | — | — |

## Overlap with t23 (26-project-notes.el)

Measured with `git diff -U0 7d98ec0 WT-project-notes-docs` vs this worktree:
- t23 hunks (old-file lines): +47 after 331, 1554, +71 after 2346, 2692–2718.
- t34 hunk: 1332–1335 (inside `--current-entry`).
No overlapping lines; t23's insertion at 331 only shifts t34's hunk by +47 lines on merge.
Test files do not overlap (t34 adds a new test file).

## Gaps / residual risk

- Scope is the reported path (project-notes commands). Generic `project.el` commands run
  from a projectless buffer (`C-h p b` project-switch-to-buffer, `c` project-compile)
  still use `project-current` and may prompt; the card's broader goal ("only the project's
  files and notes visible, VSCode/IntelliJ-like") is not addressed here and would need
  its own card (e.g. a `project-find-functions` finder bound to the Perspective).
- If `~/notes` ever becomes a git repository, scratch would have its own project and the
  fallback would not apply.
- Full suite and Go tests not run lane-locally; interactive transient path verified only
  through `--current-entry`, not by driving the key sequence.
