# Sync Audit pass 2 — SPEC-PERSPPIN-001 (card t35)

- Auditor: sync-auditor (independent, skeptical; Claude-only audit, no MCP backend call)
- Tree: `/Users/jay/workspace/imoogi-emacs/.moai/worktrees/t35`, branch `WT-persp-full-pin`, HEAD `45ff2fe`.
  Working tree at audit start: `.moai/reports/t35/progress.md` modified (not by this audit), `card-review.md` and
  `sync-audit.md` untracked. This audit wrote only `sync-audit-2.md`; `with-editor.elc` was never touched.
- Scope (delta-scoped re-audit, per pass-1 defect list): `git diff 3afb732 HEAD -- modules tests README.md`
  (4 files, +257/-21), SPEC v0.1.5 (`git diff 2172fe7 34048c4 -- .moai/specs`), progress.md §E.2 rework + §E.4.
- Inputs: pass 1 `.moai/reports/t35/sync-audit.md` (FAIL 0.846; blocking F1, F2; optional O1–O8),
  card review `.moai/reports/t35/card-review.md` (CR1–CR3).
- Evaluator profile: none configured → built-in default (Functionality 40 / Security 25 / Craft 20 /
  Consistency 15; must-pass Functionality + Security; flat mode).
- Method: every closure was checked by **active probes** in batch Emacs 30.2. The repo was booted as in
  `tests/run.el`, using the test file's `imoogi-persppin-test--isolated` sandboxes, a TRAMP-handler spy, and op
  counters. Each closure also got a **mutation spot-check**: one fresh Emacs per mutation, applied with
  `fset`/`advice-add` after boot, so no repo file was edited. All probe files and sandboxes lived outside the
  repo and were deleted. Two sandboxes orphaned by probe runs that I killed were removed too.

## Evaluation Report
SPEC: SPEC-PERSPPIN-001
Overall Verdict: **PASS** — all five blocking items (CR1, CR2, CR3, F1, F2) are closed, both by probe and by
mutation. No new blocking finding. The must-pass firewall holds: Functionality 92, Security 95.

### Dimension Scores

| Dimension | Score | Verdict | Evidence (verbatim) |
|-----------|-------|---------|---------------------|
| Functionality (40%) | 92/100 | PASS (must-pass met) | `make test-elisp`: `Ran 747 tests, 736 results as expected, 0 unexpected, 11 skipped (2026-10-13 09:57:06+0900, 70.345550 sec)` / `exit=0`. Selector `imoogi-persppin-`: `Ran 30 tests, 30 results as expected, 0 unexpected (2026-10-13 09:57:17+0900, 7.405674 sec)` / `exit=0` (23 → 30 = the 7 new deftests). Mutation baseline: `MUTRESULT none: total=9 expected=9 unexpected=0`. Each mutation turns its target test red (closure table). |
| Security (25%) | 95/100 | PASS | Added-line grep `shell-command\|call-process\|start-process\|make-process\|(eval \|read-from-string\|password\|secret\|token\|api[-_]key` over `git diff 3afb732 HEAD -- modules` → `grep-exit=1` (no match). TRAMP spy, `b` from remote buffer: `(:result prompted :candidates none :perspective-root-calls 0 :tramp-io nil :tramp-pure-ops (file-remote-p expand-file-name))`; remote mapped root: `(:result prompted :candidates none :perspective-root-calls 0 :tramp-io nil)`; non-file predicate on `/ssh:host:/tmp/`: `:remote (nil nil)`. |
| Craft (20%) | 88/100 | PASS (coverage UNVERIFIED — Gap) | Byte-compile into scratch: `=== tests/project-perspective-pin-test.el ===` followed by nothing (no warnings). `04-projects.el` has no warnings of its own. The `26-project-notes.el` warnings are at `:446-474, 2220-2223, 3415, 4693, 5055, 5242, 5357`. These are the pass-1 set shifted by +10/+14 lines, and none falls in the rework hunks (`:1247-1279`, `:2292-2296`). Mutations confirm the new tests are not vacuous (7/7 mutations caught). The 85% coverage criterion is still a **Gap**, because no elisp coverage tool is vendored. |
| Consistency (15%) | 88/100 | PASS | `moai spec lint SPEC-PERSPPIN-001` → `WARNING   StatusTransitionInvalid ... status transition "draft" → "completed" is not a canonical lifecycle edge (commit 3afb732df280e7e99372a935e00577c1d1c638ec)` / `0 error(s), 1 warning(s)` (known, see C-note). Naming of the new symbols: all carry the `imoogi-` prefix (`imoogi-project--explicit-lookup`, `imoogi-project--mark-explicit-lookup`, `imoogi-persppin-test--remote-io-guarded`, `imoogi-persppin-test--boot-advised`). SPEC v0.1.5 ↔ code check below. One README parenthetical is over-broad (N1, optional). |

Harmonic mean (4 dims, unweighted): 4 / (1/92 + 1/95 + 1/88 + 1/88) = **90.7** (0.907).
Weighted arithmetic (reference only): 0.40·92 + 0.25·95 + 0.20·88 + 0.15·88 = 91.4.
Pass 1 → pass 2: 0.846 → 0.907.

### Closure table

| Item | Probe (active, batch Emacs) | Mutation spot-check | Status |
|------|-----------------------------|---------------------|--------|
| **CR1** explicit `(project-current nil default-directory)` | From scratch in mapped "src": `(:explicit nil :explicit-nonslash nil :explicit-abbr nil :implicit "source/" :explicit-after-memo nil :explicit-t prompted :f ("source/" ("source/")) :s "source/" :d "source/")`. Explicit stays unclaimed before and after the memo fills, and with `t` it prompts. Implicit is still claimed. f/s/d context-call is intact. Nested implicit inside explicit: `(:outer nil :marker-during-explicit t :nested-implicit (transient . "source/"))`. Idempotent after 3 more installs: `(:advices 1 :finders 1 ...)`. Prompt path (marker nil after the prompt): unmapped + prompter returns `default-directory` → `"elsewhere/"`; notes buffer + same → `"project-notes/261013-source/"` (never R). | Advice made a plain pass-through → `FAILED imoogi-persppin-explicit-default-directory-not-claimed` / `MUTRESULT cr1: total=1 expected=0 unexpected=1` | **CLOSED** |
| **CR2** `b` remote buffer / remote mapped root | See Security row: both reach the prompt, `:perspective-root-calls 0`, `:tramp-io nil`. The only handler ops are `file-remote-p` and `expand-file-name`, both pure parsing in the finder's existing guard path. | `imoogi-project--buffer-project` reverted to the pre-rework body → `FAILED ...remote-buffer-prompts`, `FAILED ...remote-root-prompts` / `MUTRESULT cr2: total=2 expected=0 unexpected=2` | **CLOSED** |
| **CR3** rename rollback rebuilds the list | Covered by the new test (stubbed Treemacs failure after the commit); probe not repeated. | `:around` advice restoring the pre-rollback list (simulates the removed rebuild call) → `FAILED imoogi-persppin-rename-rollback-rebuilds-notes-list` / `MUTRESULT cr3: total=1 expected=0 unexpected=1` | **CLOSED** |
| **F1** Dired on another project's notes | Pass-1 P17/P18 reproduction rerun: `:journal (:journal "project-notes/261013-xproj/journal.org" :buffer-source-root "xproj/" :headings ("xproj/"))`. Pass 1 had `:buffer-source-root "source/"` and a `source/` heading. P14, Dired on R's own notes: `(:context-buffer-p t :project nil :b no-prompt :b-cands ("a.txt" "n1.org") ... :c prompted :k prompted)`. `b` still lists R's buffers plus notes; `c`/`k` prompt, as README limitation 1 now states. | Predicate forced to nil for non-file buffers → `FAILED ...dired-on-other-notes-keeps-source-root`, `FAILED ...non-file-notes-check-no-io`, `FAILED ...notes-check-reads-no-registry` / `MUTRESULT f1: total=3 expected=0 unexpected=3` | **CLOSED** |
| **F2** README vs D-1 | README.md:462-464 now reads "다른 프로젝트에 속한 버퍼에서는 `b`/`c`/`k`와 `C-x p` 명령이 그 버퍼의 프로젝트를 쓰고, `f`/`s`/`d`는 Perspective의 프로젝트를 따른다." The P1 rerun matches it: `(:project-current "other/" :c "other/" :b ("o.txt") :f "source/" :d "source/")`. | n/a (doc) | **CLOSED** |

Pass-1 optionals: O1 fixed (README now names `C-x p p`/`C-x p F` and TRAMP; see N1 for the TRAMP part). O2 fixed
(README:470-472). O3 fixed (REQ-PIN-013.1 "for a local CONTEXT-ROOT"; § 5). O5 fixed (acceptance gate skip
list = 7 + 2 + 2, which matches today's 11 skips). O6 is documented in spec § 5, though the exclude-functions docstring is
unchanged (N4). O7 is still open (Info, startup window).

**O4** is real: the case (b) stub now answers `"src"` and counts calls. Mutation with the `persp-mode` clause
removed → `MUTRESULT o4: total=1 expected=0 unexpected=1`. Mutation with the clause moved after
`(persp-current-name)` → `MUTRESULT o4b: total=1 expected=0 unexpected=1`.
**O8** is real. The pre-test-load capture at real boot gives `(:finder t :advice t)`. A mutation that removes the hook
member and the advice after boot, before the test file loads → `FAILED imoogi-persppin-finder-installed-at-boot` /
`MUTRESULT o8: total=1 expected=0 unexpected=1`.

### Regression probes (rework side effects)

| # | Probe | Result |
|---|-------|--------|
| R1 | `project-switch-project` (named switch command) from scratch in pinned "src" | `(:switch-to-d (transient . "dproj/") :switch-to-r (transient . "source/") :override-after nil)`: D resolves on its own merits and the override is cleaned up |
| R2 | `project-find-file` from a VC buffer of another repo | `(vc "other/" ("other/"))` |
| R3 | Eglot-style calls | `(:lsp-context-implicit nil :lsp-context-explicit-vc vc :eglot--current-project (transient . "elsewhere/") :implicit-without-lsp-context "source/")`. The real `eglot--current-project` is not redirected to R |
| R4 | `project-uniquify-dirname-transform` (not set in `modules/`) | `(:r-from-scratch-at-r "source" :elsewhere "elsewhere" :vc-sub "other/other/sub" :real-uniquify ("other/x.txt" "source/x.txt"))`. Explicit, so R is declined from its own directory (the SPEC-stated consequence). VC transform and real uniquify work, with no error |
| R5 | Tab name / project mode line | `(:scratch "source" :other-vc "other" :mode-line " source" :unmapped "*scratch*")` |
| N | Non-file false positives of the new predicate branch | `:home (nil "source/")`, `:home-tilde nil`, `:messages (".../t35/" nil t)`, `:compilation-in-root (nil "source/")`, `:notes-top-dir nil`, `:sibling-prefix nil` (`<notes>-old/` not matched), `:remote (nil nil)`, `:dd-nil nil`. No ordinary non-file buffer is marked |
| H | Hot path, 20 direct finder calls per buffer kind (memo warm) | `((scratch :claimed t :ops ((file-directory-p . 20))) (dired-notes :claimed nil :ops ((file-directory-p . 20))) (compilation :claimed t :ops ((file-directory-p . 20))))`. Exactly 1 `file-directory-p` per call (the deleted-root guard). There is no `file-truename`, `file-exists-p`, `file-attributes`, `file-readable-p`, `insert-file-contents`, `directory-files`, `--all-entries` or `--read-registry` call on the finder path |

Explicit `project-current` callers in `modules/`: exactly `04-projects.el:145` (`resolve-root`) and `:458`
(`switch-perspective`), and both bind the inhibit, as the SPEC says. Vendored explicit callers whose behaviour follows
REQ-PIN-004.3 (they resolve on their own merits): `gptel-context.el:288,298`, `ghostel.el:6682`. Upstream
`project--find-default-from` calls it explicitly only under the override, where the finder already declined.

### SPEC v0.1.5 ↔ code (changed items)

All consistent:
- § 2: the implicit/explicit definitions match `(and directory t)`.
- § 2: the notes-context non-file clause and "remote never compared" match the `cond` branch with the raw-string
  `file-remote-p`.
- REQ-PIN-004.3: the mechanism paragraph matches the advice and the idempotent installer. "Only explicit callers in
  `modules/`" matches the grep above.
- REQ-PIN-006: the `b` fallback guard text and plan D-3 order (`persp-mode` → remote `default-directory` → raw root →
  remote raw root → `--perspective-root`) match `04-projects.el:316-330`.
- REQ-PIN-008: the input list and the eight refresh points match. Plan D-7 trigger 8 (a direct call at the end of
  `--rollback-rename`) matches `26-project-notes.el:2292-2296`.
- § 5: the three new limitations match the code (`expand-file-name`, not a truename, for non-file buffers).
- AC-PIN-005(b), AC-PIN-006(d), the AC-PIN-010 rows, the AC-PIN-017 rows, and the DoD boot test each match their
  named test.
- Quality Gate: the 11-skip wording matches today's run.
- §E.4 `follow_up_sync`: base `34048c4`, and the follow-up commit is `45ff2fe`, whose parent is `34048c4`.

**C-note (not blocking): `StatusTransitionInvalid draft→completed (3afb732)`.** I agree it should not block. It is a
lifecycle-record artifact: the lane brief limited the run phase to §E.2, so `in-progress` was never written. It has no
effect on code, ACs, or traceability. Clearing it needs a history rewrite on a branch already used as the review base.
§E.4 records it as accepted (`lint_note`). It is scored as a small Consistency deduction.

### Findings (structured defect-list)

No blocking findings.

- **N1 [Low, confidence High] [optional]** `README.md:460` — the parenthetical "프로젝트가 없는 TRAMP 원격 버퍼는
  예외로 계속 묻는다" is attached to the sentence listing `b`/`c`/`k`/`f`/`s`/`d`. But `f`/`s`/`d` from a projectless
  TRAMP buffer in a locally mapped Perspective do **not** prompt. They resolve the local context root, per
  REQ-PIN-013.1 ("from any buffer"), because `imoogi-project-context-call` rebinds `default-directory` to the root.
  **Reproduction**: in mapped "src", from a temp buffer with `default-directory` `/ssh:host:/tmp/`:
  `(:f-result no-prompt :f-root "source/" :c-result prompted :tramp-io nil)`. This is the same class as pass-1 O1:
  an over-extended exception, not an inverted main rule like F2. Lineage: the clause was introduced by the O1 fix
  in `45ff2fe`. No remote I/O happens, and no wrong behaviour versus
  the SPEC. **Fix** (one clause): "(명시적으로 고르는 `C-x p p`/`C-x p F`, 그리고 프로젝트가 없는 TRAMP 원격 버퍼의
  `b`/`c`/`k`와 `C-x p` 명령은 예외로 계속 묻는다)".
- **N2 [Info, confidence Medium] [optional]** `04-projects.el:207` — `imoogi-project--mark-explicit-lookup (function
  &optional maybe-prompt directory)` forwards exactly two arguments. If a future `project.el` gives `project-current`
  a third parameter, the advice would silently drop it. Bundled project.el 0.11.1 (Emacs 30.2) has two, so nothing
  can go wrong today. Optional: `(function &rest args)` and test `(nth 1 args)`.
- **N3 [Info, confidence High] [optional]** pre-first-rebuild window for the non-file branch. Before the first rebuild,
  the fallback list is `imoogi-project-notes-directory` itself, so a Dired/compile/shell buffer anywhere under it,
  including the top-level notes directory, counts as notes context. After the rebuild, only registered entry
  directories count. Probe: `:unbuilt-notes-top-dir t` vs `:notes-top-dir nil`. Likewise, Emacs started with its
  initial `default-directory` inside a notes directory makes `*scratch*`/`*Messages*` notes context, by the stated rule.
  This is the same class as the documented D-7 startup window and O7. No action needed.
- **N4 [Info, confidence High] [optional]** `04-projects.el:127-132` — the docstring of
  `imoogi-project-perspective-finder-exclude-functions` still forbids only file I/O. The re-entry/recursion
  limitation (pass-1 O6) lives in spec § 5 only. Optional: add "or call `project-current`" to the docstring.
- **N5 [Info, confidence Medium] [optional]** Derived consequence of the F1 fix, consistent with D-6 and not a defect:
  `imoogi-tab-name` in a Dired buffer on notes now falls back to the tab-bar default name, because
  `(project-current nil)` is nil there (P14 `:project nil`). Notes file buffers have behaved this way since the run
  phase. This was not measured directly: the journal probe switched the window buffer.

### Recommendations

1. The card can close on this audit. Fold N1 into any later README touch. It is optional and does not reopen the
   sync deliverable.
2. N2/N4 are one-line hardening/docstring items for a future pass. N3/N5 need no action.
3. Keep the `StatusTransitionInvalid` note as recorded in §E.4. Do not rewrite history for it.
