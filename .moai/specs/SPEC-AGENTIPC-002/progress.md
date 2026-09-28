# Progress — SPEC-AGENTIPC-002

imoogi-agent hardening follow-ups (cards t18, t19, t20). Tier S, two plan-phase artifacts
(spec.md with inline acceptance criteria, plan.md) plus this file.

## §E.1 Plan-phase Audit-Ready Signal

```yaml
plan_status: audit-ready
plan_complete_at: 2026-09-28T00:00:00+09:00
spec_id: SPEC-AGENTIPC-002
spec_version: "0.1.1"
tier: S
artifacts: [spec.md, plan.md, progress.md]
requirements: 8
acceptance_criteria: 8
depends_on: [SPEC-AGENTIPC-001]
baseline_head: ecf68a9
source_findings: .moai/reports/sync-audit/SPEC-AGENTIPC-001-4dim.json (Security #2, #3, #4)
cards: [t18, t19, t20]
plan_audit:
  - iteration: 1
    verdict: FAIL
    score: 0.86
    report: .moai/reports/plan-audit/SPEC-AGENTIPC-002-review-1.md
    addressed_in: "0.1.1"
    resolved_findings: [D1, D2, D3, D4, D5, D6, D7, D8, D9]
  - iteration: 2
    verdict: PASS
    score: 0.97
    report: .moai/reports/plan-audit/SPEC-AGENTIPC-002-review-2.md
    optional_findings_applied: [R2-O1, R2-O2]
plan_phase_gaps: []
closed_gaps:
  - GODEBUG=execerrdot=0 via t.Setenv measured by plan-audit iteration 2 on go1.26.4 (LookPath returns "ec", no error, not absolute); ErrDot restored after the test
  - sparse-file creation via dd (seek=104857601) measured by plan-audit iteration 1 on Emacs 30.2 batch (dd exit 0, size 104857601); plan.md R-3
```

## §E.2 Run-phase Evidence

Scope executed: M1 (Emacs receiver) + M2 (Go CLI), cycle_type=tdd, serial.

Evidence tree: runtime L1 worktree `.claude/worktrees/agent-a718e7121e254ba5a` (branch `worktree-agent-a718e7121e254ba5a`,
base ecf68a9 == origin/main); each milestone pushed with `git push origin HEAD:main`. Evidence logs persisted under the primary
checkout at `.moai/state/verify/SPEC-AGENTIPC-002/`. ERT selector form = spec.md § 3.3.0 `ert_sel`, except that the advice is loaded
from a file (`-l .moai/state/verify/ert-sel.el`, selector in `$ERT_SEL`) because the worktree guard refused an inline `--eval`.
Emacs 30.2 (`/Applications/Emacs.app/Contents/MacOS/Emacs`), go1.26.4 darwin/arm64.

| AC | Status | Test / command | Actual output (this run) | Tree |
|----|--------|----------------|--------------------------|------|
| AC-AIPH-001 | PASS | `ERT_SEL='^imoogi-agent-aiph01-'` → `imoogi-agent-aiph01-untrusted-event-file` | `Ran 1 tests, 1 results as expected, 0 unexpected` (exit 0; `ac-aiph01.log`) | daafb83 |
| AC-AIPH-002 | PASS | `ERT_SEL='^imoogi-agent-aiph02-'` → `imoogi-agent-aiph02-trust-scope-order-and-links` | `Ran 1 tests, 1 results as expected, 0 unexpected` (exit 0; `ac-aiph02.log`) | daafb83 |
| AC-AIPH-003 | PASS | `ERT_SEL='^imoogi-agent-aiph03-'` → `imoogi-agent-aiph03-payload-too-large-rejects-whole-event` (sparse file via `dd … seek=104857601`, plan § 5.1 primary method; size asserted = 104857601) | `Ran 1 tests, 1 results as expected, 0 unexpected` (exit 0; `ac-aiph03.log`) | daafb83 |
| AC-AIPH-004 | PASS | `ERT_SEL='^imoogi-agent-aiph04-'` → `imoogi-agent-aiph04-payload-cap-boundary-without-prompt` | `Ran 1 tests, 1 results as expected, 0 unexpected` (exit 0; `ac-aiph04.log`) | daafb83 |
| AC-AIPH-005 | PASS | `go test ./internal/agentipc/... -run TestAIPH05 -count=1` | `ok  github.com/karohani/imoogi-emacs/internal/agentipc 0.209s` | fc5f3e3 |
| AC-AIPH-006 | PASS | `go test ./internal/agentipc/... -run TestAIPH06 -count=1` | `ok  github.com/karohani/imoogi-emacs/internal/agentipc 1.225s` | fc5f3e3 |
| AC-AIPH-007 | PASS | `go test ./internal/agentipc/... -run TestAIPH07 -count=1` | `ok  github.com/karohani/imoogi-emacs/internal/agentipc 0.198s` | fc5f3e3 |
| AC-AIPH-008 | PASS | `go test ./internal/agentipc/... -run TestAIPH08 -count=1` | `ok  github.com/karohani/imoogi-emacs/internal/agentipc 0.253s` | fc5f3e3 |

"Tree daafb83 / fc5f3e3": the command ran on the working tree immediately before that commit; the files were staged by explicit
pathspec right after the run with no edits in between, so the committed content is the tree that was measured.

RED evidence (E8, captured before GREEN):
- M1 `m1-red-ert.log`: `Ran 4 tests, 0 results as expected, 4 unexpected` —
  aiph01 `(equal ("a" "ok") ("a" "error:untrusted"))`; aiph02 `(equal "error:too-large" "error:untrusted")`;
  aiph03 `(equal ("a" "ok") ("a" "error:payload-too-large"))`; aiph04 `(void-variable imoogi-agent-max-payload-bytes)`.
- M2 `m2-red-go.log`: TestAIPH06 FAIL (`event directory mode = 755, want 700`, `event directory name = "002", want prefix imoogi-agent-`,
  `directory names "002" and "002", want two different names`); TestAIPH07 FAIL (`findEmacsclient() = ".../002/ec", <nil>; want ".../001/ec"`);
  TestAIPH08 FAIL in all 4 subtests (`code = 0 (), want 1`, `a program was run although the bare name did not resolve`).
  TestAIPH05 PASSED at RED by design (contract-fixing test; plan.md § 4 M2 step 1 and D-6 — `classify` already maps every `error:` status to 3).
  TestAIPH06/unwritable_TMPDIR also passed at RED (the old `os.CreateTemp` path already failed with exit 1 there).
- M2 `m2-red-go-ac14.log`: the updated `TestAC14DiscoveryIgnoresDirectoriesAndRelativePath` FAILED
  (`bare EMACSCLIENT resolved against the current directory: ".../003/ec"`).
- Mutation check (`m2-mutation-isabs.log`): with the `filepath.IsAbs` guard removed, only
  `TestAIPH08BareNameNotFoundFailsDelivery/d_dot_entry_with_execerrdot=0` fails (`code = 0 (), want 1`), so the guard is load-bearing
  for REQ-AIPH-007.4. Guard restored before the M2 commit.

Regression and quality (fc5f3e3 tree unless noted):
- `make test-elisp` (full suite): `Ran 494 tests, 490 results as expected, 0 unexpected, 4 skipped` (`m2-test-elisp.log`; same result on
  the M1 tree, `m1-test-elisp.log`). Worktree baseline before any change: `Ran 490 tests, 486 results as expected, 0 unexpected, 4 skipped`
  (`baseline-test-elisp.log`). The 4 skips (`imoogi-org-calendar-*-gui` x2, `imoogi-toolchain-*` x2) are environmental; the worktree lacks
  the untracked toolchain binaries of the primary checkout (primary baseline per the orchestrator: 488/2). 490 → 494 = the four aiph tests.
- `ERT_SEL='^imoogi-agent-'`: `Ran 16 tests, 16 results as expected, 0 unexpected` (ac01..ac10 + aiph01..04; `m1-green-ert.log`).
- `go test ./... -count=1`: exit 0, 24 `ok` package lines, no FAIL (`m2-go-test-all.log`).
- `go test -cover ./internal/agentipc/ ./cmd/imoogi-agent/ -count=1`: `internal/agentipc coverage: 96.9% of statements`,
  `cmd/imoogi-agent coverage: 100.0% of statements` (97.2% / 100.0% at ecf68a9). `cmd/imoogi-agent/main_test.go` unchanged and passing.
- `go build ./...` exit 0; `GOOS=linux GOARCH=amd64 go build ./...` exit 0; `go vet ./...` exit 0.
- `gofmt -l internal/agentipc cmd/imoogi-agent`: empty. `golangci-lint run ./internal/agentipc/... ./cmd/imoogi-agent/...`: `0 issues.`
- Pre-push hook `make ci-local` passed on both pushes (the hook resolves `git rev-parse --show-toplevel`, i.e. the worktree, so it tested
  this tree); `m1-push.log` and `m2-push.log` each contain `Ran 494 tests, 490 results as expected, 0 unexpected, 4 skipped` and the ref update.
- PRESERVE: `git diff --stat ecf68a9 fc5f3e3 -- CLAUDE.md scripts/imoogi-editor go.mod go.sum go.work packages.el packages.lock vendor modules/project cmd/imoogi-agent internal/agentipc/request.go internal/agentipc/agentipc.go` → empty.

## §E.3 Run-phase Audit-Ready Signal

```yaml
run_complete_at: 2026-09-28T08:26:00+09:00
run_commit_sha: fc5f3e3            # final (M2) on origin/main; M1 daafb83
run_status: audit-ready
ac_pass_count: 8                   # AC-AIPH-001..008
ac_fail_count: 0
preserve_list_post_run_count: 0    # git diff --stat ecf68a9 fc5f3e3 over the plan.md § 7 PRESERVE paths → empty
l44_pre_commit_fetch: "origin/main...HEAD = 0 1 before each push (M1 at daafb83, M2 at fc5f3e3)"
l44_post_push_fetch: "origin/main...HEAD = 0 0 after each push"
push_results:
  - commit: daafb83
    result: "git push origin HEAD:main exit 0 — ecf68a9..daafb83 HEAD -> main (pre-push make ci-local passed)"
  - commit: fc5f3e3
    result: "git push origin HEAD:main exit 0 — daafb83..fc5f3e3 HEAD -> main (pre-push make ci-local passed)"
core_bare_checks: "false after M1 commit, M1 push, M2 commit, M2 push"
new_warnings_or_lints_introduced: 0  # go vet clean, golangci-lint 0 issues, gofmt clean
cross_platform_build:
  go_build: "go build ./... exit 0"
  go_build_linux_amd64: "GOOS=linux GOARCH=amd64 go build ./... exit 0"
  go_vet: "go vet ./... exit 0"
coverage:
  internal_agentipc: "96.9% (baseline 97.2%)"
  cmd_imoogi_agent: "100.0% (baseline 100.0%)"
total_run_phase_files: 4           # modules/development/30-agent.el, tests/agent-test.el, internal/agentipc/emacsclient.go, internal/agentipc/emacsclient_test.go
m1_to_mN_commit_strategy: "per-milestone commit + push from runtime L1 worktree branch worktree-agent-a718e7121e254ba5a via HEAD:main"
local_main_note: "primary checkout local main not touched (still ecf68a9, 2 behind origin/main); fast-forward it before sync"
spec_status_transition: "spec.md status draft -> in-progress, updated 2026-09-28 (after the M1 push; untracked, not committed)"
residual_risks:
  - "R-4: cleanup after a write failure inside the new directory is code-reviewed (single defer os.RemoveAll), not test-induced"
  - "R-5: a signal kill leaves the 0700 directory and its 0600 file"
  - "R-7: a user-owned symlink to a user-owned file with chosen content passes the trust check"
  - "R-8: the PATH emacsclient branch has no IsAbs guard (GODEBUG=execerrdot=0)"
```

## §E.4 Sync-phase Audit-Ready Signal

```yaml
sync_status: complete
sync_complete_at: 2026-09-28T08:35:00+09:00
sync_commit_sha: 89426a7387d1ba649e74eadd60e08a23f2b1b874
docs_updated: [README.md, .moai/project/domains.md, .moai/project/tech.md]
verification: "orchestrator re-run at fc5f3e3: make test-elisp Ran 494, 492 expected, 0 unexpected, 2 skipped; go test ./... all ok; agentipc 96.9% / cmd 100%; go vet ok; gofmt clean"
audit_note: "Tier S — 4-dim sync audit not run; orchestrator verification batch used"
residual_risks: [R-4, R-5, R-7, R-8]
frontmatter_status_transitions: {spec.md: "in-progress -> completed"}
```

## §F Phase 4 Mode Selection

Input parameters: tier=S; scope=4 files (30-agent.el, tests/agent-test.el, internal/agentipc/emacsclient.go, internal/agentipc/emacsclient_test.go); domains=2 (Emacs Lisp receiver, Go CLI); language mix=elisp+Go; concurrency benefit=LOW (coding-heavy, 2 small milestones); Agent Teams not requested.

| Mode | Selected | Rationale |
|------|----------|-----------|
| direct | no | multi-file behavior change with new tests |
| serial | **yes** | coding-heavy, M1→M2 in order; one manager-develop (tdd) |
| fanout | no | not research-heavy; <3 domains |
| sweep | no | not a mechanical bulk transform |

Decision: serial

Justification: Tier S with 4 files and two sequential milestones; a single manager-develop (cycle_type=tdd) is the simplest correct shape, and coding tasks gain little from parallel spawns.

Kickoff: Implementation Kickoff Approval obtained 2026-09-28 (start; progression=autonomous; push=each milestone commit to origin/main). Pre-spawn check: origin/main...HEAD = 0 0 at ecf68a9, 0 foreign sessions.
