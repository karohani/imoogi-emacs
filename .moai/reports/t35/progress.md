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
- [ ] RESUME HERE: run (manager-develop, tdd) on SPEC-PERSPPIN-001 v0.1.3 at HEAD a1f2273
- [ ] run (manager-develop, tdd)
- [ ] sync (manager-docs) → sync-audit
- [ ] card-review → worktree commit → report to leader

## Decision log
- 2026-10-13 ~07:50: two plan clarifications decided in the lane, before plan-audit:
  - D-1: keep the precedence asymmetry. f/s/d stay Perspective-first and b/c/k stay own-project-first; unifying either way regresses a tested behaviour or t34's card-review lesson.
  - D-6: accept that c/k prompt from a project-notes buffer. The card names projectless buffers only, and claiming notes buffers would redirect the worktree source root.
  decision record: decided_by=claude+lane-3 evidence_refs=.moai/specs/SPEC-PERSPPIN-001/plan.md#D-1,#D-6 ladder_path=①(disk evidence: SPEC rationale + card text)
