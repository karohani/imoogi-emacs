# Progress — SPEC-ANKICARD-005

짝 없는 `#+BEGIN_EXTRA` / `#+END_EXTRA` 표식 보고 (백로그 카드 t16). Tier S —
`spec.md`(인수 기준 인라인), `plan.md`, 이 파일.

## §E.1 Plan-phase Audit-Ready Signal

```yaml
plan_status: audit-ready
plan_complete_at: 2026-09-28T10:00:00+09:00
plan_audit_final: "iteration 2 PASS 0.93 (.moai/reports/plan-audit/SPEC-ANKICARD-005-review-2.md)"
plan_complete_at: 2026-09-28
spec_id: SPEC-ANKICARD-005
tier: S
artifacts: [spec.md, plan.md, progress.md]
requirements: 8
acceptance_criteria: 8
depends_on: []
related_specs: [SPEC-ANKICARD-003, SPEC-ANKICARD-004]
baseline_head: c010009
```

plan 단계 산출물 작성 완료. `plan-auditor` 감사 대기.

## §E.2 Run-phase Evidence

| AC | Test / command | Actual output | Status |
|---|---|---|---|
| AC-AKX-001 | go test ./internal/anki/orgdoc -run TestRenderRejectsUnpairedExtraMarker | --- PASS: TestRenderRejectsUnpairedExtraMarker (0.00s) | PASS |
| AC-AKX-002 | same (row two_openings_one_closing_leave_it_in_the_extracted_content) | --- PASS | PASS |
| AC-AKX-003 | go test ./internal/anki/orgdoc -run TestRenderAcceptsExtraMarkerLookalikes | --- PASS: TestRenderAcceptsExtraMarkerLookalikes (0.00s) | PASS |
| AC-AKX-004 | go test ./internal/anki/planner -run TestExtraBlockUnbalancedPrecedence | --- PASS: TestExtraBlockUnbalancedPrecedence (0.00s) | PASS |
| AC-AKX-005 | TestExtraBlockUnbalancedSkipsOnlyThatEntry, TestMultilineDanglingExtraMarkerIsReported, TestMultilineUnbalancedOpeningIsReported | 3x --- PASS | PASS |
| AC-AKX-006 | TestMigrate_UnpairedExtraMarker_IsSkippedAndLeavesTheOriginal, TestRenderErrorMapping, TestExtraBlockUnbalancedCodeMatchesWireString | 3x --- PASS | PASS |
| AC-AKX-007 | make test-elisp | Ran 496 tests, 492 results as expected, 0 unexpected, 4 skipped (worktree) | PASS |
| AC-AKX-008 | go test -cover ./internal/anki/... ; git diff --stat c010009 -- golden/testdata/hashing/extra_test.go | orgdoc 100.0%, planner 92.8%; diff empty | PASS |

Evidence: .moai/state/verify/SPEC-ANKICARD-005/ (RED: red-1-compile.log, red-2-behavioral.log, red-m2-elisp.log). Implemented by manager-develop in L1 worktree agent-a36e0dec0346d956e, pushed with `git push origin HEAD:main`.

Orchestrator re-verification (primary checkout after `git merge --ff-only origin/main` to 6678118): `make test-elisp` → Ran 496 tests, 494 results as expected, 0 unexpected, 2 skipped; `go test -cover ./internal/anki/...` → orgdoc 100.0%, planner 92.8%; all 9 new/renamed tests `--- PASS`; `go test ./...` no failures; `go vet ./internal/anki/...` ok; `gofmt -l internal/anki` empty; core.bare false. Log: .moai/state/verify/df00ce0d/ankicard005-elisp.log

## §E.3 Run-phase Audit-Ready Signal

```yaml
run_status: audit-ready
run_complete_at: 2026-09-28
run_commit_sha: 6678118
m1_commit_sha: 7953aa9
ac_pass_count: 8
ac_fail_count: 0
preserve_list_post_run_count: 0
l44_pre_commit_fetch: "0 2 before push (origin/main c010009)"
l44_post_push_fetch: "0 0 (origin/main 6678118)"
new_warnings_or_lints_introduced: 0   # 4 pre-existing staticcheck findings, all in untouched files
cross_platform_build:
  darwin: ok
  linux: ok
  windows_internal_anki: ok
  windows_full: "fails in untouched internal/setup (syscall.Flock) — pre-existing"
total_run_phase_files: 13
m1_to_mN_commit_strategy: "two commits (M1, M2), single push after M2 — pre-push ci-local runs test-elisp whose codes-match-source-file check couples the M1 Go constant to the M2 elisp list (deviation from push-per-milestone, reported to operator)"
core_bare_after_each_commit_and_push: false
spec_status_transition: "draft -> in-progress applied by orchestrator (agent blocked by worktree isolation guard)"
```

## §E.4 Sync-phase Audit-Ready Signal

```yaml
sync_status: complete
sync_complete_at: 2026-09-28
sync_commit_sha: 0d6771f7c2bb5108c1d9261cf1e281874b2ae356
docs_updated:
  - README.md (Anki multiline section: EXTRA no-nesting, extra_block_unbalanced, marker escaping)
  - .moai/project/domains.md (Known gaps t16 line removed)
  - .moai/reports/anki-card-types-plan-20260920.md (correction note, untracked)
verification: "orchestrator re-run at 6678118: make test-elisp Ran 496, 494 expected, 0 unexpected, 2 skipped; go test -cover ./internal/anki/... orgdoc 100.0% planner 92.8%; 9 new/renamed tests PASS; go vet ok; gofmt clean"
audit_note: "Tier S — orchestrator verification batch, no 4-dim audit"
supersedes: "SPEC-ANKICARD-003 progress.md:319 dangling_extra_marker: pinned-not-reported — now reported as extra_block_unbalanced; 003 files not edited"
follow_ups: "SPEC-ANKICARD-004 on resume: REQ-SW-001.7, spec.md:972-980, acceptance.md:497 row, acceptance.md:118 precedent must be revised by manager-spec"
known_flaky: "imoogi-setup-test-ankiconnect-status-invalid-response-on-http-error failed once in the implementer's run, passed in all others"
```

## §F Phase 4 Mode Selection

Input parameters: tier=S; scope≈6 production files + tests (internal/anki/protocol, internal/anki/orgdoc, internal/anki/planner, modules/org/anki/imoogi-error.el, tests); domains=2 (Go anki backend, Emacs front end); concurrency benefit=LOW (coding-heavy, M1→M2 sequential); Agent Teams not requested.

| Mode | Selected | Rationale |
|------|----------|-----------|
| direct | no | multi-file behavior change with new diagnostic code |
| serial | **yes** | coding-heavy; one manager-develop (tdd) runs M1 then M2 |
| fanout | no | not research-heavy |
| sweep | no | not a mechanical bulk transform |

Decision: serial

Justification: small two-milestone change where M2 depends on M1's wire code; a single sequential implementer is simplest.

Kickoff: Implementation Kickoff Approval obtained 2026-09-28 (start; progression=autonomous; push=each milestone commit to origin/main). Plan Audit Gate: skip-eligible (iteration 2 PASS 0.93 ≥ 0.75; plan-artifact hash subjects spec.md/plan.md unchanged since review-2).
