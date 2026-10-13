# t35 progress — perspective fully pinned to its project

- lane: lane-3 · worktree: .moai/worktrees/t35 · branch: WT-persp-full-pin · base: origin/main 6feee71
- card: t35 (Class C; plan → run → sync in this lane). Predecessor t34 = 4285b31 (in base).
- ref: .moai/reports/t34/progress.md "Gaps / residual risk" (generic project.el commands from a projectless buffer; a project-find-functions finder bound to the Perspective)
- leader stop: commit in worktree only — no push / PR / main merge, no primary-checkout edits
- SPEC: SPEC-PERSPPIN-001 (new)

## Stages
- [x] intake — `moai factory next --card t35` refused twice (exit 4, `refused owned: the factory record assigns the card to lane-2`). Leader ruling d-20261012T222845Z-e4bf: lane-3 carries t35 without a factory lease, skips factory next/stage, and keeps evidence in this file only. Worktree created from 6feee71 on WT-persp-full-pin; core.bare=false.
  decision record: decided_by=leader evidence_refs=moai decision read --scope card:t35#d-20261012T222845Z-e4bf ladder_path=②
- [x] plan — manager-spec commit 1614ba7 (4 SPEC files, +867 lines; 15 REQ / 17 AC, Tier M); `moai spec lint` "No findings" (agent-reported); core.bare=false verified by the lane. Key-binding correction: the menu is `C-c h p` (05-transient.el:644), not `C-h p` as the card says.
- [x] plan revision — commit 2026135 (D-1/D-6 recorded, D-7 notes-list refresh triggers, v0.1.1); lane-verified: `moai spec lint` "No findings", 0 NEEDS CLARIFICATION markers, core.bare=false
- [x] plan-audit r1 — **FAIL 0.792** (Clarity 0.75 / Compl 0.90 / Test 0.70 / Trace 0.85); MP-1..7 pass; 8 blocking defects D1–D8, all text-level (hook depth for APPEND ordering, AC-PIN-013/011 not observable, workspace-bridge stub contradiction, notes buffers inside "projectless", caching claim, glossary term, refresh-trigger coverage); report .moai/reports/plan-audit/SPEC-PERSPPIN-001-plan-audit.md
- [x] plan revision v0.1.2 — commit 6db7b3d (D1–D8 + O1–O8; finder depth 95, one-slot memo per pinned root, f via project-find-file-in, notes buffers outside "projectless"); lane-verified: lint "No findings", diff limited to the 4 SPEC files (+331/-129), core.bare=false
- [x] plan-audit r2 — **FAIL** (harmonic 0.832 ≥ 0.80, but the round fails automatically because D4 is unresolved: Clarity 0.80 / Compl 0.90 / Test 0.75 / Trace 0.90). D1–D3 and D5–D8 closed. New blocking defects:
  - R2-D1: `f` has no project resolution; probe `(error "prompted")`.
  - R2-D2: the global `project-root` override at workspace-bridge-test.el:19 breaks transient projects.
  - R2-D3: the "no longer prompt" claims are too broad.
  Report: .moai/reports/plan-audit/SPEC-PERSPPIN-001-plan-audit-r2.md
  decision record: decided_by=claude+lane-3 evidence_refs=.moai/reports/plan-audit/SPEC-PERSPPIN-001-plan-audit-r2.md#R2-D1,R2-D2 ladder_path=①(auditor-verified probes; route (a) keeps the card's no-prompt goal and is the probe-verified route)
  - R2-D1: route (a), a context-root transient fallback only when the context root is non-nil. Extend it to s/d if they share f's path.
  - R2-D2: allow a one-line test override fix.
- [x] plan revision v0.1.3 — commit a1f2273 (route (a) for f; context-call clause in the finder so f/s/d (shared helper imoogi-project-context-call, 04-projects.el:111-136) do not prompt when a context root exists; E3 override fix; claims scoped; AC-PIN-018 memo); lane-verified: lint "No findings", 4 SPEC files only (+282/-82), core.bare=false
- [x] plan-audit r3 (final) — **PASS 0.899** (Clarity 0.85 / Compl 0.95 / Test 0.90 / Trace 0.90). D4/R2-D1..D3 closed and O-a..O-h closed, with no blocking defects. Optional O3-a..g are carried into run as guidance; O3-d (f's transient branch must not apply to a remote root) is taken as an implementation constraint consistent with REQ-PIN-006. Report: .moai/reports/plan-audit/SPEC-PERSPPIN-001-plan-audit-r3.md
  decision record: decided_by=claude+lane-3 evidence_refs=.moai/reports/plan-audit/SPEC-PERSPPIN-001-plan-audit-r3.md#verdict-PASS-0.899 ladder_path=①(disk evidence PASS → plan→run gate open; leader dispatch covers plan→run→sync in lane)
- [x] run — manager-develop (tdd) worked in an auto-isolated tree (worktree-agent-a7c3187e441b483ff), 11 commits 67c4e3c…565e15b (RED/GREEN pairs M1–M4, M5 tests, evidence); fast-forwarded WT-persp-full-pin 641cc35 → 565e15b. The agent reports 18/18 AC PASS plus the O3-d test, and existing-test edits exactly E1–E3.
  Lane re-verification at 565e15b (logs .moai/state/verify/t35/):
  - `make test-elisp` exit 0: "Ran 740 tests, 729 results as expected, 0 unexpected, 11 skipped" (baseline 717/0/11).
  - selector `imoogi-persppin-` plus the 4 t34 tests: exit 0, "Ran 27 tests, 27 results as expected, 0 unexpected".
  - tests/project-notes-perspective-test.el diff vs 641cc35 is empty; `git status` is clean; core.bare=false.
  - Lane read the 04-projects.el/05-transient.el diff: the finder guards are ordered as designed and route (a) carries the O3-d remote gate.
  - Noted for review: it uses the private project.el helpers `project--find-default-from` and `project--buffer-check`.
  Deviation: trigger 7 (startup notes-list rebuild) is registered on `window-setup-hook`, not `emacs-startup-hook`. The existing test `imoogi-treemacs-does-not-auto-start-on-emacs-startup-hook` (workspace-bridge-test.el:523) pins that hook to contain only `imoogi-perspective-state-restore`, and editing it would be a 4th existing-test edit outside E1–E3.
  decision record: decided_by=claude+lane-3 evidence_refs=tests/workspace-bridge-test.el:523,commit 09e62b0 ladder_path=①(accept the deviation: window-setup-hook runs right after emacs-startup-hook, so the rebuild still happens once at startup after the restore; manager-spec updates the SPEC text)
- [x] SPEC text alignment v0.1.4 — commit 89a0af0 (trigger 7 on window-setup-hook, D-8 per-Perspective memo, O3-a/b/c/e/f, gate = make test-elisp); lane-verified: 4 SPEC files only (+144/-46), core.bare=false
- [x] sync — commit 3afb732 (README "### 프로젝트와 작업공간" +13 lines: pinned Perspective paragraph + 알려진 제한; spec.md status draft → completed; SPEC progress §E.3/§E.4; no CHANGELOG.md in repo). Lane-verified: README diff is the new paragraph only; core.bare=false. Agent-reported: spec lint "No findings".
- [x] card-review (codex_task job-20261013T001423Z-6f489bc0, read-only): 3 medium findings CR1–CR3. The lane confirmed all three by code read → .moai/reports/t35/card-review.md
- [x] sync-audit pass 1 — **FAIL**, harmonic 0.846 (F 78 / S 95 / C 85 / Cons 82); all 18 ACs pass, and all 12 probe scenarios behave as required. Blocking: F1 (a Dired buffer on a notes folder is not notes context, so the journal source root is redirected; reproduced by the auditor) and F2 (the README misstates D-1 for f/s/d). Optional O1–O8 → .moai/reports/t35/sync-audit.md
  decision record: decided_by=claude+lane-3 evidence_refs=.moai/reports/t35/card-review.md#CR1-CR3,.moai/reports/t35/sync-audit.md#F1-F2 ladder_path=①(confirmed defects → rework; the card does not close with them open)
  - CR1: fix with an `:around` advice on `project-current` that marks lookups with an explicit DIRECTORY, so the finder declines them. Measured: the only explicit-directory callers in modules/ are 04-projects.el:138 (resolve-root, inhibited) and :431 (switch-perspective, inhibited), so nothing internal relies on the finder for explicit lookups. Rejected: narrowing REQ-PIN-004.
  - CR2: add the remote guards to `imoogi-project--buffer-project` before `--perspective-root`.
  - CR3: rebuild the notes-directory list at the end of `--rollback-rename`.
  - F1: option (a). For non-file buffers, also check `default-directory` against the in-memory notes list (no file I/O). Amend §2 / REQ-PIN-008 / AC-PIN-010.
  - F2 + O1 + O2: README wording (manager-docs).
  - O3 + O5: SPEC text (manager-spec).
  - O4: strengthen the AC-PIN-005 case (b) test.
  - O8: add a boot-installed test if feasible.
  - O6 and O7: recorded as known limitations.
- [x] rework — manager-develop, auto-isolated (worktree-agent-a3a4684d5ca75bf55): 7 commits 02c10ad…2172fe7. Fast-forwarded WT-persp-full-pin 3afb732 → 2172fe7.
  - Agent-reported RED before each fix (CR1, CR2 ×2, CR3, F1 ×3).
  - O4 mutation proof: the old test passed with the persp-mode check removed; the new one fails.
  - O8 boot-install test `imoogi-persppin-finder-installed-at-boot`, with mutation proof.
  Lane re-verification at 2172fe7 (log .moai/state/verify/t35/test-elisp-rework.txt):
  - `make test-elisp` exit 0: "Ran 747 tests, 736 results as expected, 0 unexpected, 11 skipped".
  - Existing-test edits vs 6feee71 are still only transient-menu-test.el (2 lines) and workspace-bridge-test.el (12 lines) = E1–E3.
  - The t34 test file is unchanged; core.bare=false.
  - Lane read the module diff:
    - the explicit-lookup marker is checked in the finder guard, and the advice is installed with the hook;
    - the `b` fallback checks remote on the raw strings before `--perspective-root`;
    - the rollback ends with a rebuild;
    - the non-file `default-directory` branch is remote-checked first and has no truename.
- [x] SPEC text v0.1.5 — commit 34048c4. Covers the CR1 explicit-lookup definition and mechanism, the F1 non-file notes context, remote `b` in REQ-PIN-006 / AC-PIN-017, the rollback trigger, the AC-PIN-005(b)/boot test, and O3-d/O5; AC count stays 18. Lint: 1 warning `StatusTransitionInvalid` draft→completed at 3afb732, which belongs to manager-docs.
- [x] docs follow-up — commit 45ff2fe (README F2/O1/O2 + Dired·compile·shell in the notes limitation; §E.4 sync_commit_sha 3afb732 + follow_up_sync). Lane-verified: diff is README + SPEC progress.md only, the README text matches D-1, core.bare=false.
  Lint warning `StatusTransitionInvalid draft→completed (3afb732)` left in place: only a history rewrite could clear it.
  Cause: the lane's run brief said "only progress.md §E.2 is yours", so the run agent skipped its draft→in-progress transition at M1. This was a lane brief error, disclosed here.
  decision record: decided_by=claude+lane-3 evidence_refs=moai spec lint SPEC-PERSPPIN-001 (1 warning),.moai/specs/SPEC-PERSPPIN-001/progress.md#E.4-lint_note ladder_path=①(accept the warning; no amend/rebase of committed history)
- [x] sync-audit pass 2 — **PASS**, harmonic 0.907 (F 92 / S 95 / C 88 / Cons 88), 0 blocking → .moai/reports/t35/sync-audit-2.md.
  - CR1, CR2, CR3, F1 and F2 were closed by active probes, and each mutation of a fix turned its test red.
  - The O4 and O8 tests are real.
  - Auditor runs: `make test-elisp` "Ran 747 tests, 736 results as expected, 0 unexpected, 11 skipped"; selector `imoogi-persppin-` 30/30.
  Optional findings and their disposition (lane, ①):
  - N1 (README: the TRAMP exception is over-broad, since f/s/d from a projectless TRAMP buffer resolve to the local context root): fixed in the final commit with the auditor's exact wording.
  - N2 (advice forwards exactly 2 args; fine for project.el 0.11.1) and N4 (exclude-functions docstring should also forbid calling `project-current`): follow-ups, not fixed here.
  - N3 (pre-first-rebuild window for non-file buffers) and N5 (Dired-on-notes tab name falls back to the default): no action, consistent with D-6/D-7.
- [x] card-review CR1–CR3 closed (see rework and sync-audit pass 2)
- [ ] final worktree commit (README N1 + t35 reports) → report to leader. No push / PR / merge. in parallel (read-only codex_task job-20261013T001423Z-6f489bc0 on `git diff 641cc35 565e15b -- modules tests`; codex_review scope=card is not used because the worktree has no git-strategy.yaml, same as t15) → sync-audit
- [ ] run (manager-develop, tdd)
- [ ] sync (manager-docs) → sync-audit
- [ ] card-review → worktree commit → report to leader

## Decision log
- 2026-10-13 ~07:50: two plan clarifications decided in the lane, before plan-audit:
  - D-1: keep the precedence asymmetry. f/s/d stay Perspective-first and b/c/k stay own-project-first; unifying either way regresses a tested behaviour or t34's card-review lesson.
  - D-6: accept that c/k prompt from a project-notes buffer. The card names projectless buffers only, and claiming notes buffers would redirect the worktree source root.
  decision record: decided_by=claude+lane-3 evidence_refs=.moai/specs/SPEC-PERSPPIN-001/plan.md#D-1,#D-6 ladder_path=①(disk evidence: SPEC rationale + card text)
