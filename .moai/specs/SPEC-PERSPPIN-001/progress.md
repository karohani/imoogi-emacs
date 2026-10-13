# Progress — SPEC-PERSPPIN-001

Pin a project Perspective fully to its project. Tier M, card t35.

## §E.1 Plan-phase Audit-Ready Signal

```yaml
plan_status: audit-ready
plan_complete_at: 2026-10-13
spec_id: SPEC-PERSPPIN-001
spec_version: "0.1.5"  # v0.1.5 = post-run text alignment with the review rework at 2172fe7
tier: M
artifacts: [spec.md, plan.md, acceptance.md, progress.md]
requirements: 15
acceptance_criteria: 18
baseline_head: 6feee71
open_clarifications: 0  # D-1, D-6, R2-D1..D3 decided by lane-3 2026-10-13; D-7 refresh points fixed
answers_audit: .moai/reports/plan-audit/SPEC-PERSPPIN-001-plan-audit-r2.md  # r2 FAIL 0.832 (D4 unresolved); R2-D1..D3 + O-a..O-h fixed in v0.1.3
next_audit_iteration: 3  # last under the 3-iteration cap
next_audit_scope: delta-only (R2-D1..R2-D3, O-a..O-h)
```

## §E.2 Run-phase Evidence

Run 2026-10-13, cycle_type tdd, on top of `641cc35`. Tree: the L1 isolated
worktree `.claude/worktrees/agent-a7c3187e441b483ff` (branch
`worktree-agent-a7c3187e441b483ff`, fast-forwarded to `641cc35`), for
fast-forwarding `WT-persp-full-pin`. Emacs 30.2 at
`/Applications/Emacs.app/Contents/MacOS/Emacs`.

Every run below is
`EMACS=/Applications/Emacs.app/Contents/MacOS/Emacs IMOOGI_ERT_SELECTOR='<selector>' ./tests/run.sh`
(check-parens → offline boot → ert), or `make test-elisp` for the full suite.
`make lint` is `go vet ./...` only and does not cover Emacs Lisp. The elisp
gate is the check-parens and boot-health stages of `run.sh`, plus a
byte-compile probe of the changed files: no warnings in `04-projects.el`,
`05-transient.el` or the new test file; the `26-project-notes.el` warnings
are all at pre-existing lines.

Selectors:
- S-M1 `imoogi-persppin-\(scratch-resolves\|vc-root\|finder-is-last\|own-vc-project\|unmapped-off\|explicit-lookups\|eglot-context\|remote-root\|finder-never\|finder-memo\)`
- S-M2 `imoogi-persppin-\(notes-buffer-declined\|notes-check-reads\|notes-list-refresh\|notes-list-startup\|t34-fallback\)`
- S-M3 `\(imoogi-persppin-\(transient-b-c-k\|switch-buffer-candidates\|no-notes-entry\)\|imoogi-transient-suffix-dispatch-matches-spec-table\)`
- S-M4 `\(imoogi-persppin-\(find-file-candidates\|no-notes-entry\|incidental-prompt\|find-file-remote-context\)\|imoogi-project-context-root-follows-perspective-mapping\|imoogi-project-context-falls-back-to-project-el\)`
- S-M5 `imoogi-persppin-\(notes-not-project-members\|transient-other-entries\|transient-b-c-k\)`
- S-GATE `\(project-notes\|transient\|imoogi-project-\|perspective\|persppin\|workspace\)` (acceptance.md Quality Gate)
- S-AC016 acceptance.md AC-PIN-016 command

| Step | Selector | Exit | Result line |
|------|----------|------|-------------|
| C.1 baseline | plan C.1 selector | 0 | Ran 279 tests, 272 results as expected, 0 unexpected, 7 skipped (2026-10-13 08:37:06+0900) |
| full baseline | `make test-elisp` | 0 | Ran 717 tests, 706 results as expected, 0 unexpected, 11 skipped (2026-10-13 08:38:15+0900) |
| M1 RED | S-M1 | 1 | Ran 10 tests, 3 results as expected, 7 unexpected (08:45:39) |
| M1 GREEN | S-M1 | 0 | Ran 10 tests, 10 results as expected, 0 unexpected (08:46:43) |
| M1 gate | S-GATE | 0 | Ran 289 tests, 282 results as expected, 0 unexpected, 7 skipped (08:48:02) |
| M2 RED | S-M2 | 1 | Ran 5 tests, 1 results as expected, 4 unexpected (08:50:20) |
| M2 GREEN | S-M2 | 0 | Ran 5 tests, 5 results as expected, 0 unexpected (08:51:05) |
| AC-PIN-016 | S-AC016 | 0 | Ran 5 tests, 5 results as expected, 0 unexpected (08:51:15) |
| M2 gate | S-GATE | 0 | Ran 294 tests, 287 results as expected, 0 unexpected, 7 skipped (08:52:06) |
| M3 RED | S-M3 | 1 | Ran 4 tests, 2 results as expected, 2 unexpected (08:53:22) |
| M3 GREEN | S-M3 | 0 | Ran 4 tests, 4 results as expected, 0 unexpected (08:53:51) |
| M3 gate | S-GATE | 0 | Ran 297 tests, 290 results as expected, 0 unexpected, 7 skipped (08:54:49) |
| M4 RED | S-M4 | 1 | Ran 6 tests, 3 results as expected, 3 unexpected (08:56:32) |
| M4 GREEN | S-M4 | 0 | Ran 6 tests, 6 results as expected, 0 unexpected (08:57:18) |
| M4 gate | S-GATE | 0 | Ran 300 tests, 293 results as expected, 0 unexpected, 7 skipped (08:58:28) |
| M5 first run | S-M5 | 0 | Ran 3 tests, 3 results as expected, 0 unexpected (08:58:57) |
| M5 non-vacuity (finder + M3 `b` row disabled) | S-M5 | 1 | Ran 3 tests, 0 results as expected, 3 unexpected (08:59:12) |
| full, before the trigger-7 fix | `make test-elisp` | 2 | Ran 740 tests, 728 results as expected, 1 unexpected, 11 skipped (09:00:30) — `imoogi-treemacs-does-not-auto-start-on-emacs-startup-hook` |
| trigger-7 fix | startup-registered + treemacs startup test | 0 | Ran 2 tests, 2 results as expected, 0 unexpected (09:03:32) |
| **final full** | `make test-elisp` | **0** | **Ran 740 tests, 729 results as expected, 0 unexpected, 11 skipped (2026-10-13 09:05:08+0900)** |
| final gate | S-GATE | 0 | Ran 302 tests, 295 results as expected, 0 unexpected, 7 skipped (09:06:02) |
| final AC run | `imoogi-persppin-` | 0 | Ran 23 tests, 23 results as expected, 0 unexpected (09:06:56) |

The skip sets are unchanged from baseline: the gate skips the 7
`imoogi-project-notes-cli-cache-*` tests; the full suite also skips the 2
GUI org-calendar tests and the 2 toolchain-server tests.

RED notes. Tests that passed at RED pin preserved behaviour, which holds
before the change: AC-PIN-004/005/006 at M1, AC-PIN-016 at M2,
AC-PIN-011/017(b) at M3 (the M1 finder had already removed the prompt),
AC-PIN-017(f), O3-d and workspace-bridge :160 at M4. Non-vacuity was checked
by mutation:
- dropping the inhibit in `imoogi-project-switch-perspective` fails AC-PIN-006;
- dropping the remote-name guards fails AC-PIN-008;
- dropping the O3-d remote gate fails `imoogi-persppin-find-file-remote-context-root-prompts`;
- disabling the finder and the `b` row fails all M5 tests.

Per-AC matrix:

| AC | Test(s) | Status | Actual output |
|----|---------|--------|---------------|
| AC-PIN-001 | `imoogi-persppin-scratch-resolves-perspective-project` | PASS | root = canonical R; `(project-current t)` equal, no prompt |
| AC-PIN-002 | `imoogi-persppin-vc-root-resolves-vc-project` | PASS | `(vc Git R)` equal to the inhibited `(project-current nil R)` |
| AC-PIN-003 | `imoogi-persppin-finder-is-last-member` | PASS | finder last after try-vc/F0/F1/F2; F2's `(transient . OTHER)` wins |
| AC-PIN-004 | `imoogi-persppin-own-vc-project-beats-perspective` | PASS | root B, `compile` at B |
| AC-PIN-005 | `imoogi-persppin-unmapped-off-or-stale-declines` | PASS | nil + "prompted" for (a)/(b)/(c) |
| AC-PIN-006 | `imoogi-persppin-explicit-lookups-not-claimed` | PASS | (a)/(b) nil; (c) switches to D's Perspective, not "src" |
| AC-PIN-007 | `imoogi-persppin-eglot-context-declines` | PASS | nil under `eglot-lsp-context` t; R's project when nil |
| AC-PIN-008 | `imoogi-persppin-remote-root-no-file-io` | PASS | both nil; no remote I/O recorded |
| AC-PIN-009 | `imoogi-persppin-finder-never-signals` | PASS | nil, tab name a string, stub called at least once |
| AC-PIN-010 | `imoogi-persppin-notes-buffer-declined`, `-notes-check-reads-no-registry`, `-notes-list-refresh-triggers`, `-notes-list-startup-registered` | PASS (deviation: trigger 7 surface) | notes buffer nil, same source root; no registry/metadata read; rows 1-6 rebuild; trigger 7 registered on `window-setup-hook` (see deviation below) |
| AC-PIN-011 | `imoogi-persppin-transient-b-c-k-without-prompt` | PASS | `c` at R; one `yes-or-no-p` naming the project, P1 live; `b` → P1 |
| AC-PIN-012 | `imoogi-persppin-switch-buffer-candidates` | PASS | exactly {P1, Dired R, N1} |
| AC-PIN-013 | `imoogi-persppin-find-file-candidates` | PASS | R files ∪ regular files of N (no `.git/`, no S notes); visits N/project.org; from "s", no N file |
| AC-PIN-014 | `imoogi-persppin-notes-not-project-members` | PASS | no notes path in files/buffers; N1 live after the kill |
| AC-PIN-015 | `imoogi-persppin-transient-other-entries-unchanged`, `-incidental-prompt-removals`, workspace-bridge :141/:160 | PASS | only `b` changed; `F` prompts; d/s/f from notes use R; no context root → "prompted"; setup root = R |
| AC-PIN-016 | t34 four tests (unmodified) + `imoogi-persppin-t34-fallback-retained` | PASS | 5/5 |
| AC-PIN-017 | `imoogi-persppin-no-notes-entry-project-only` | PASS | `b` = (P1), `f` = (R/a.txt), also with the identity key signalling; no message |
| AC-PIN-018 | `imoogi-persppin-finder-memo` | PASS | R-count 1, then 2 after the clear; remap → R2 |
| O3-d (extra) | `imoogi-persppin-find-file-remote-context-root-prompts` | PASS | "prompted", no remote I/O |

Permitted existing-test edits made: exactly E1
(`tests/transient-menu-test.el:80`), E2
(`tests/workspace-bridge-test.el` :141 and :160 stubs moved to
`project-find-file-in`, assertions unchanged) and E3
(`tests/workspace-bridge-test.el:19-20`). The four t34 tests are unmodified.

Deviation (needs a scope-doc update, routed to manager-spec). AC-PIN-010
verifies trigger 7 as "a member of `emacs-startup-hook` [that] sorts after
`imoogi-perspective-state-restore`" (acceptance.md L164-167; plan.md D-7
item 7). The existing test
`imoogi-treemacs-does-not-auto-start-on-emacs-startup-hook`
(`tests/workspace-bridge-test.el:523`) asserts that `emacs-startup-hook` is
exactly `(imoogi-perspective-state-restore)`, and changing it would be a
fourth edit outside E1-E3. The rebuild is therefore registered on
`window-setup-hook`. `startup.el` runs that hook after `emacs-startup-hook`,
under the same `inhibit-startup-hooks` guard, so it still runs once at
startup, after the restore. Commit `09e62b0`.

Run-phase choices within the plan's options:
- D-8 memo is a per-key alist (the run-phase option named in plan.md D-8), not
  one slot. Two frames with different Perspectives no longer thrash it, and
  REQ-PIN-001's memo wording (O3-a) then holds literally.
- O3-d taken: route (a)'s middle branch skips a remote context root.

### Rework after review

Run 2026-10-13, cycle_type tdd, reproduction-first, on top of `3afb732`.
Inputs: `.moai/reports/t35/card-review.md` (CR1-CR3) and
`.moai/reports/t35/sync-audit.md` (F1, O4, O8). Tree: the L1 isolated
worktree `.claude/worktrees/agent-a3a4684d5ca75bf55` (branch
`worktree-agent-a3a4684d5ca75bf55`, fast-forwarded to `3afb732`), for
fast-forwarding `WT-persp-full-pin`.

Selector runs below are
`IMOOGI_ERT_SELECTOR='<selector>' IMOOGI_TEST_USER_DIR=<fresh tmp dir> /Applications/Emacs.app/Contents/MacOS/Emacs --batch -Q -l tests/run.el`
(the ert stage of `run.sh`, with the same boot). The full gate is
`EMACS=/Applications/Emacs.app/Contents/MacOS/Emacs make test-elisp`.
S-REL is
`imoogi-persppin-\|imoogi-project-notes-scratch\|imoogi-project-notes-unmapped\|imoogi-project-notes-buffer-project\|imoogi-project-notes-unregistered\|imoogi-project-switch\|imoogi-project-context\|imoogi-project-perspective\|imoogi-perspective-\|imoogi-treemacs-\|imoogi-transient-suffix`
(the persppin tests, the four t34 tests, workspace-bridge, and the transient
dispatch table).

| Defect | Step | Selector | Exit | Result line |
|--------|------|----------|------|-------------|
| baseline | — | `imoogi-persppin-` | 0 | Ran 23 tests, 23 results as expected, 0 unexpected (09:34:10) |
| CR1 | RED | `imoogi-persppin-explicit-default-directory` | 1 | Ran 1 tests, 0 results as expected, 1 unexpected (09:39:15) — `(project-current nil ".../elsewhere/")` `:value (transient . ".../source/")` |
| CR1 | GREEN | S-REL | 0 | Ran 80 tests, 80 results as expected, 0 unexpected (09:40:12) |
| CR2 | RED | `imoogi-persppin-switch-buffer-remote` | 1 | Ran 2 tests, 0 results as expected, 2 unexpected (09:40:33) — remote buffer: `b` "did not signal an error" (it switched to R's buffer); remote root: `(equal "Remote file I/O: /ssh:host:/srv/app/" "prompted")` |
| CR2 | GREEN | same, then S-REL | 0 / 0 | Ran 2 tests, 2 results as expected, 0 unexpected (09:40:46); Ran 82 tests, 82 results as expected, 0 unexpected (09:40:55) |
| O4 | old test, `persp-mode` check deleted | `imoogi-persppin-unmapped-off-or-stale-declines` | 0 | Ran 1 tests, 1 results as expected, 0 unexpected (09:41:12) — vacuous, as the audit said |
| O4 | new test, `persp-mode` check deleted | same | 1 | Ran 1 tests, 0 results as expected, 1 unexpected (09:41:22) — `(should-not (project-current nil))` |
| O4 | new test, check restored | `imoogi-persppin-` | 0 | Ran 26 tests, 26 results as expected, 0 unexpected (09:41:37) |
| CR3 | RED | `imoogi-persppin-rename-rollback` | 1 | Ran 1 tests, 0 results as expected, 1 unexpected (09:42:00) — `(imoogi-project-notes--context-buffer-p)` `:value nil` for a fresh file under the restored N |
| CR3 | GREEN | same, then `rename\|rollback\|imoogi-persppin-` | 0 / 0 | Ran 1 tests, 1 results as expected, 0 unexpected (09:42:10); Ran 33 tests, 33 results as expected, 0 unexpected (09:42:19) |
| F1 | RED | the 3 F1 tests | 1 | Ran 3 tests, 0 results as expected, 3 unexpected (09:43:01) |
| F1 | RED (journal reproduction) | `imoogi-persppin-dired-on-other-notes` | 1 | Ran 1 tests, 0 results as expected, 1 unexpected (09:43:18) — X's journal text contains `** 작업 공간: .../source/` |
| F1 | GREEN | the 3 F1 tests, then S-REL | 0 / 0 | Ran 3 tests, 3 results as expected, 0 unexpected (09:43:35); Ran 85 tests, 85 results as expected, 0 unexpected (09:43:46) |
| O8 | first run | `imoogi-persppin-finder-installed-at-boot` | 0 | Ran 1 tests, 1 results as expected, 0 unexpected (09:44:07) |
| O8 | install call replaced by `(ignore)` | `imoogi-persppin-` | 1 | Ran 30 tests, 29 results as expected, 1 unexpected (09:44:21) — only the boot test fails |
| O8 | hook added without the advice | `imoogi-persppin-` | 1 | Ran 30 tests, 29 results as expected, 1 unexpected (09:44:37) — `(should imoogi-persppin-test--boot-advised)` |
| O8 | restored | `imoogi-persppin-` | 0 | Ran 30 tests, 30 results as expected, 0 unexpected (09:44:53) |
| **full** | gate | `make test-elisp` | **0** | **Ran 747 tests, 736 results as expected, 0 unexpected, 11 skipped (2026-10-13 09:46:19+0900, 72.680222 sec)** |

Every mutation was reverted before the next step; `git diff --exit-code -- modules/project/04-projects.el`
returned 0 each time. The skip set is the same 11 tests as before (7
`cli-cache-*`, 2 GUI org-calendar, 2 toolchain-server). `git status --short`
was empty after the full run (`with-editor.elc` untouched). A byte-compile
probe of the two changed modules shows no warnings in `04-projects.el`; every
`26-project-notes.el` warning sits at the same pre-existing code, shifted by
the added lines (460/467, 2220-2223, 3415, 4693, 5055, 5242, 5357, 446-474).

Cause or symptom, checked once per defect:
- CR1, cause. `project-current` replaces an omitted DIRECTORY with
  `default-directory` before it runs the finders, so no check inside a finder
  can tell the two calls apart. The marker is set where the information
  still exists.
- CR2, cause. `b`'s fallback was a second path to the mapped root, built on
  `--perspective-root` (written for `f`/`s`/`d`), without the finder's
  guards. The guards now sit at that path.
- CR3, cause. The notes-directory list is a cache that the registry and
  mounted-state writers rebuild. The rollback restores both files with direct
  writes, so it was the one writer that skipped the rebuild.
- F1, cause. The journal heading is the symptom. The notes-context predicate
  covered visited files only, so the finder claimed non-file buffers inside
  notes directories, and `--current-source-root` (which predates this SPEC)
  took the Perspective's root.

| Defect | Fix | Test(s) | Status |
|--------|-----|---------|--------|
| CR1 | `04-projects.el`: `imoogi-project--explicit-lookup` + `imoogi-project--mark-explicit-lookup` (`:around` on `project-current`), checked in `imoogi-project--perspective-find`, installed by `imoogi-project-install-perspective-finder` (idempotent). Commit `02c10ad` | `imoogi-persppin-explicit-default-directory-not-claimed` | PASS: explicit DIR == `default-directory` → nil (and prompts with MAYBE-PROMPT); the implicit lookup → R; after reinstalling, 1 advice and 1 hook member |
| CR2 | `04-projects.el` `imoogi-project--buffer-project`: decline for a remote `default-directory` or a remote raw mapped root, before `--perspective-root`. Commit `e4b735b` | `imoogi-persppin-switch-buffer-remote-buffer-prompts`, `-remote-root-prompts` | PASS: "prompted", no `read-buffer`, no remote file check |
| O4 | test only. Commit `816b791` | `imoogi-persppin-unmapped-off-or-stale-declines` (b) | PASS: nil, "prompted", 0 `persp-current-name` calls; fails when the check is deleted |
| CR3 | `26-project-notes.el` `imoogi-project-notes--rollback-rename`: rebuild the list as the last step. Commit `e3c225e` | `imoogi-persppin-rename-rollback-rebuilds-notes-list` | PASS: N restored, N2 gone, a fresh file under N is notes context and declined |
| F1 | `26-project-notes.el` `imoogi-project-notes--context-buffer-p`: for a buffer visiting no file, a local `default-directory`, expanded only, compared with the same in-memory list. Commit `e3bd1ea` | `imoogi-persppin-dired-on-other-notes-keeps-source-root`, `-non-file-notes-check-no-io`, plus a Dired row in `-notes-check-reads-no-registry` | PASS: X's journal keeps X's root, no heading for R; the predicate reads no registry, metadata, truename, existence or attributes |
| O8 | test only. Commit `3ba878c` | `imoogi-persppin-finder-installed-at-boot` | PASS: finder in the global `project-find-functions`; advice present at test-file load time (after `boot.el`) |

Constraints held: existing-test edits are still exactly E1-E3 (all new tests
and the O4 change are in `tests/project-perspective-pin-test.el`, which this
SPEC added); the four t34 tests are unmodified; spec.md, plan.md,
acceptance.md and README.md are untouched. The SPEC wording these fixes need
(§ 2 notes-context, REQ-PIN-004 mechanism, REQ-PIN-008, AC-PIN-006/010/017)
is routed to manager-spec.

## §E.3 Run-phase Audit-Ready Signal

Filled at sync from the §E.2 evidence above (the run phase left it empty).

```yaml
run_status: audit-ready
run_complete_at: 2026-10-13
cycle_type: tdd
run_head: 89a0af0
final_full_suite: "Ran 740 tests, 729 results as expected, 0 unexpected, 11 skipped"  # re-verified by the lane before sync
acceptance_criteria_pass: 18/18
deviations: [AC-PIN-010 trigger 7 on window-setup-hook (09e62b0), aligned in spec text 89a0af0]
```

## §E.4 Sync-phase Audit-Ready Signal

```yaml
sync_complete_at: 2026-10-13
sync_commit_sha: pending-backfill  # a commit cannot name its own SHA; parent 89a0af0
sync_status: complete
docs_updated: [README.md "프로젝트와 작업공간 (`project.el` + `perspective.el`)"]
changelog_entry_position: none  # repo has no CHANGELOG.md
b12_self_test_a: n/a (no CHANGELOG.md)
b12_self_test_b: acceptance.md distinct AC ids = 18
b12_self_test_c: n/a (no CHANGELOG.md)
frontmatter_status_transitions:
  spec.md: draft -> completed  # in-progress/implemented never recorded; merged close on this commit
  plan.md / acceptance.md / progress.md: no frontmatter
```
