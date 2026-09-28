# Progress — SPEC-AGENTIPC-001

Coding Agent -> Emacs IPC bridge MVP (`imoogi-agent`). Tier M, three plan-phase artifacts
(spec.md, plan.md, acceptance.md) plus spec-compact.md, research.md, and this file.

## §E.1 Plan-phase Audit-Ready Signal

```yaml
plan_status: audit-ready
plan_complete_at: 2026-09-27T11:44:00+09:00
spec_id: SPEC-AGENTIPC-001
spec_version: "0.1.2"
tier: M
artifacts: [spec.md, plan.md, acceptance.md, spec-compact.md, research.md, progress.md]
requirements: 15
acceptance_criteria: 15
baseline_head: fd7f0b0
plan_audit:
  - iteration: 1
    verdict: PASS
    score: 0.89
    report: .moai/reports/plan-audit/SPEC-AGENTIPC-001-review-1.md
  - iteration: 2
    verdict: PASS
    score: 0.93
    report: .moai/reports/plan-audit/SPEC-AGENTIPC-001-review-2.md
  - iteration: 3
    verdict: PASS
    score: 0.95
    report: .moai/reports/plan-audit/SPEC-AGENTIPC-001-review-3.md
resolved_findings: [D1, D2, D3, D4, O1, O4, O6, N1]
open_optional_findings: [O2, O3, O5, O7, R3-N2]
plan_html: .moai/reports/plan-html/SPEC-AGENTIPC-001-plan.html
```

## §E.2 Run-phase Evidence

Scope actually executed: M1 + M2 (Emacs receiver). Mid-run scope change from the orchestrator narrowed this
agent to M1-M2; M3 was already committed locally (01646c2, NOT pushed) and M4 RED tests were parked uncommitted.
M3-M5 rows below are therefore not evaluated here.

Evidence tree: worktree `.claude/worktrees/agent-a4d9c2a09e8e29e96`, evidence logs under its `.moai/state/verify/`.
Command: `bash .moai/state/verify/ert_sel.sh "^imoogi-agent-ac"` (acceptance.md §0.1 form, Emacs 30.2) at HEAD 01646c2
(elisp files byte-identical to M2 commit a496ded). Output tail: `Ran 12 tests, 12 results as expected, 0 unexpected`.

| AC | Status | Test / command | Actual output |
|----|--------|----------------|---------------|
| AC-AIPC-001 | PASS | `imoogi-agent-ac01-message-notifies-and-logs`, `imoogi-agent-ac01-newlines-become-one-line` | `passed 1/12`, `passed 2/12` |
| AC-AIPC-002 | PASS | `imoogi-agent-ac02-open-file-keeps-focus` | `passed 3/12` |
| AC-AIPC-003 | PASS | `imoogi-agent-ac03-goto-moves-only-target-window` | `passed 4/12` |
| AC-AIPC-004 | PASS | `imoogi-agent-ac04-artifact-created-notice-and-display` | `passed 5/12` |
| AC-AIPC-005 | PASS | `imoogi-agent-ac05-task-finished-success-and-failed` (a,b), `imoogi-agent-ac05-task-finished-artifact-displayed` (c) | `passed 7/12`, `passed 6/12` |
| AC-AIPC-006 | PASS | `imoogi-agent-ac06-syntax-trailing-and-duplicate-keys` | `passed 8/12` |
| AC-AIPC-007 | PASS | `imoogi-agent-ac07-schema-violations` (rows a-o) | `passed 9/12` |
| AC-AIPC-008 | PASS | `imoogi-agent-ac08-size-limit-and-file-preconditions` | `passed 10/12` |
| AC-AIPC-009 | PASS | `imoogi-agent-ac09-payload-path-policy` | `passed 11/12` |
| AC-AIPC-010 | PASS | `imoogi-agent-ac10-never-waits-for-input` + `grep -c -e '26-project-notes' -e 'imoogi-project-notes' modules/development/30-agent.el` | `passed 12/12`; grep prints `0` |
| AC-AIPC-011..014 | NOT EVALUATED | out of this agent's revised scope (Go CLI, parallel agent) | — |
| AC-AIPC-015 | NOT EVALUATED | M5 not started (smoke not run) | — |

Full-suite regression (`make test-elisp`, Emacs 30.2): `Ran 476 tests, 471 results as expected, 1 unexpected, 4 skipped`;
the one unexpected is `imoogi-project-notes-rename-quit-restores-directory`, identical to the pre-change baseline
(`Ran 464 tests, 459 results as expected, 1 unexpected, 4 skipped`) — pre-existing, not introduced here.

RED evidence (captured before GREEN): M1 `.moai/state/verify/m1-red.log` — 6/6 FAILED `(void-function imoogi-agent-receive-file)`;
M2 `.moai/state/verify/m2-red.log` — 6 FAILED (`(equal "error:handler" "ok")` / `(windowp nil)`), 6 M1 tests passed.

## §E.3 Run-phase Audit-Ready Signal

```yaml
run_complete_at: 2026-09-27T16:30:17+09:00
run_commit_sha: a41c41a          # final (M5) on origin/main; M1 b5b7ff2, M2 48a3c7b, M3 fc5fda7, M4 4b72a86
run_status: audit-ready          # agent A's original partial-blocked state resolved — see orchestrator_integration below
ac_pass_count: 10                # AC-AIPC-001..010, exercised by 12 ERT tests (001 and 005 have two each)
ac_fail_count: 0
ac_not_evaluated: [AC-AIPC-011, AC-AIPC-012, AC-AIPC-013, AC-AIPC-014, AC-AIPC-015]
preserve_list_post_run_count: 0  # git diff --quiet fd7f0b0 a496ded over the PRESERVE list → unchanged
l44_pre_commit_fetch: "0 0 at start (fd7f0b0)"
l44_post_push_fetch: "origin/main...a496ded = 2 2 (origin gained c8c5d7f, b9ce802) — push halted"
push_results:
  - commit: f902785
    result: rejected by pre-push hook (make ci-local → pre-existing imoogi-project-notes-rename-quit-restores-directory failure)
  - commit: a496ded
    result: not attempted — the B9 pre-push divergence check fired (origin/main...a496ded = 2 2); STOP per B9, no rebase
new_warnings_or_lints_introduced: 0
cross_platform_build:
  go_build: "go build ./... exit 0"
  go_vet: "go vet ./... exit 0"
total_run_phase_files: 2         # modules/development/30-agent.el, tests/agent-test.el
m1_to_mN_commit_strategy: per-milestone commits; local branch worktree-agent-a4d9c2a09e8e29e96
orchestrator_integration:        # after the blocker above
  replay: "f902785,a496ded cherry-picked onto origin/main b9ce802 in the A worktree (branch WT-agent-emacs-rebased) → b5b7ff2 (M1), 48a3c7b (M2); M3 01646c2 NOT carried"
  test_elisp: "make test-elisp exit 0 — Ran 483 tests, 479 results as expected, 0 unexpected, 4 skipped (baseline failure fixed upstream by c8c5d7f)"
  push: "git push origin HEAD:main exit 0 (pre-push make ci-local passed) — b9ce802..48a3c7b"
agent_b_go_cli:                  # M3+M4, worktree agent-a595ec235cb51fdaf (base b9ce802)
  commits_original: [55d3e8e (M3), facec0a (M4)]
  replay: "cherry-picked onto 48a3c7b in the A worktree → fc5fda7 (M3), 4b72a86 (M4)"
  ac: "AC-AIPC-011..014 PASS — go test -run 'TestAC1[1-4]' ok in cmd/imoogi-agent and internal/agentipc (agent B report, HEAD facec0a)"
  orchestrator_recheck: "go vet ./... ok; go test -count=1 -cover → internal/agentipc 97.2%, cmd/imoogi-agent 100.0% (at 4b72a86)"
  push: "git push origin HEAD:main exit 0 (pre-push make ci-local passed) — 48a3c7b..4b72a86"
  superseded: "agent A local M3 01646c2 discarded (not integrated)"
m5_wiring:                       # agent worktree agent-a7f2a2a18d56ef6e1
  commit: a41c41a                # Makefile, boot.el, scripts/install.sh, tests/module-layout-test.el, tests/setup-toolchain-test.sh
  red: "setup-toolchain-test.sh → FAIL: imoogi-agent CLI was not linked; module-layout ERT → 1 unexpected"
  ac015: "make build-agent exit 0; make -n build-all | grep -c imoogi-agent → 1; module-layout ERT 1/1; setup-toolchain tests: PASS; go.mod/go.sum/go.work/packages.* unchanged"
  smoke_emacs_30_2: "exit=0 | exit=3 bad-path | exit=3 not-loaded | exit=1 can't find socket; leftover event files 0"
  smoke_emacs_31_1: "same four results; server emacs-version \"31.1\"; 30.2 client → 31.1 server exit=0"
  ci_local: "make ci-local exit 0 — Ran 483 tests, 479 expected, 0 unexpected, 4 skipped; pre-push hook passed; 4b72a86..a41c41a"
  daemon_cleanup: "pgrep -fl aipc → none"
final_orchestrator_verification:  # HEAD a41c41a == origin/main, run from worktree agent-a4d9c2a09e8e29e96
  ert_ac001_010: "Ran 12 tests, 12 results as expected, 0 unexpected"
  go_ac011_014: "go test -run TestAC11..14 → ok for cmd/imoogi-agent and internal/agentipc (all 8 lines ok)"
  preserve: "git diff --stat fd7f0b0 a41c41a over CLAUDE.md 13-system.el scripts/imoogi-editor go.* packages.* vendor → empty"
  run_status_final: complete
  ac_pass_count_final: 15
out_of_scope_local_artifacts:
  - "01646c2 (M3, committed locally, unpushed): cmd/imoogi-agent/{main.go,main_test.go}, internal/agentipc/{agentipc.go,args.go,event.go,transport.go,cli_test.go}"
  - "uncommitted M4 RED test parked at .moai/state/verify/parked-m4-transport_test.go.txt"
```

## §E.4 Sync-phase Audit-Ready Signal

```yaml
sync_status: complete
sync_complete_at: "2026-09-27T21:10:00+09:00"
sync_commit_sha: 69ee8c1aca9d520834beb3017ed3fed96465e2dd
docs_updated:
  - .moai/project/product.md
  - .moai/project/structure.md
  - .moai/project/tech.md
  - .moai/project/codemaps/overview.md
  - .moai/project/codemaps/entry-points.md
  - .moai/project/codemaps/data-flow.md
  - README.md
changelog: "not emitted (user decision, HUMAN GATE 2)"
gate_sync_1: "make ci-local exit 0 — Ran 490 tests, 488 expected, 0 unexpected, 2 skipped; all Go ok; setup-toolchain PASS (log .moai/state/verify/df00ce0d/ci-local.log)"
coverage: "cmd/imoogi-agent 100.0%, internal/agentipc 97.2%"
lint: "go vet ok; golangci-lint 0 issues"
mx: "no P1/P2 violations"
four_dim_audit: "PASS harmonic 0.873 (F 0.93 / S 0.87 / C 0.82 / Cons 0.88), threshold 0.80, binding (no critical/contested) — .moai/reports/sync-audit/SPEC-AGENTIPC-001-4dim.json"
findings: "15 minor; 4 carried to backlog t18-t21 (JSON holds 16 entries: 15 code findings + #16 history-metadata note on 4b72a86 author)"
evidence_gap: "AC-015 daemon smoke not re-run in sync (read-only review); relies on run-phase record"
history_note: "commit 4b72a86 author recorded as t <t@example.invalid> (git-config corruption fixed in 43f88d1); left as-is, no history rewrite"
backup: ".moai/backups/sync-20260927-210300"
```

## §F Phase 4 Mode Selection

Input parameters: tier=M; scope≈11 files (6 new, 5 one-line edits); domains=2 (Emacs Lisp module, Go CLI) + build wiring;
language mix=elisp+Go+Makefile+shell; concurrency benefit=LOW (coding-heavy, M1→M5 sequential dependency); Agent Teams not requested.

| Mode | Selected | Rationale |
|------|----------|-----------|
| direct | no | multi-file new feature, not trivial |
| serial | **yes** | coding-heavy, milestones depend on each other; plan.md §9 recommends serial |
| fanout | no | not research-heavy; <3 domains |
| sweep | no | not mechanical bulk transform |

Decision: serial

Justification: A single manager-develop (cycle_type=tdd) runs M1→M5 in order. Coding tasks have few truly parallel parts
(Anthropic coding-task parallelism caveat), and M3/M4 depend on the M1/M2 receiver contract.

Kickoff: Implementation Kickoff Approval obtained 2026-09-27 (start; progression=autonomous; push=each milestone commit to origin/main).
Re-selection (operator request, same day): split into two concurrent write agents with isolated trees —
agent A (runtime-assigned worktree agent-a4d9c2a09e8e29e96): M1-M2 Emacs; agent B (Agent isolation=worktree, own branch, no push): M3-M4 Go.
Disjoint file sets; M5 wiring runs after both, following integration of both branches by the orchestrator.

Plan Audit Gate: skip-eligible — review-3 PASS 0.95 ≥ Tier M 0.80; all plan artifacts mtime (≤11:47:59) precede review-3 (11:52:26).
