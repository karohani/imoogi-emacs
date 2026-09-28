# SPEC Review Report: SPEC-AGENTIPC-002
Iteration: 2/3
Verdict: PASS
Overall Score: 0.97

Tier S (PASS threshold 0.75). The score rose from 0.86 in iteration 1 to 0.97, so the score-regression STOP rule does
not fire. Following the Retry Loop Contract, this re-audit covers the iteration-1 defect delta (D1-D9) plus the
must-pass criteria and ceilings. It also measures whether the new AC-AIPH-008 sub-cases can actually be run.

Reasoning context ignored per M1 Context Isolation. The coordinator's summary of changes was treated as a claim and
verified against the artifacts. Artifacts read: `spec.md` v0.1.1 (415 lines), `plan.md` (201 lines), `progress.md`
(43 lines). No SPEC artifact was edited.

## Must-Pass Results

- [PASS] MP-1 REQ number consistency: `grep -c '^#### REQ-AIPH-'` = 8, sequential from REQ-AIPH-001 (spec.md:L150)
  to REQ-AIPH-008 (L213). That is ≤ 8, the Tier S ceiling. `grep -c '^#### AC-AIPH-'` = 8, from AC-AIPH-001 (L241)
  to AC-AIPH-008 (L331), also ≤ 8.
- [PASS] MP-2 EARS/GEARS format compliance, judged on the REQ layer only; the Given-When-Then ACs were not graded
  here. The reworded REQ-AIPH-007 (L203) is event-driven: "When `$EMACSCLIENT` is set to a value without a `/`, the
  CLI shall resolve it only by a `PATH` search in which the first matching entry decides, shall treat … as a
  resolution failure, and on success shall use …". The other seven REQs are unchanged from iteration 1, which
  already passed them.
- [PASS] MP-3 YAML frontmatter validity: all 12 fields are present (L2-L13), with `version: "0.1.1"` quoted.
  `moai spec lint --strict .moai/specs/SPEC-AGENTIPC-002/spec.md` printed "✓ No findings — all SPEC documents are
  valid" and exited 0.
- [N/A] MP-4 language neutrality: single-project SPEC, not template-bound.
- [PASS] MP-5 D7: the only external reference is SPEC-AGENTIPC-001, whose `status` is `completed`. No BLOCKING
  finding.
- [PASS] MP-6 D8: `grep -c syscall` returned 0 for spec.md and plan.md.
- [PASS] MP-7 clarification gate: `grep -rn 'NEEDS CLARIFICATION' .moai/specs/SPEC-AGENTIPC-002/` exited 1 with no
  matches.

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 1.0 | 1.0 | REQ-AIPH-007 and 007.3 (L203, L207-209) now state exec.LookPath first-match semantics: "처음 찾아진 항목이 결과를 정한다 … 뒤의 절대 경로 항목으로 넘어가지 않는다" (the first matching entry decides; a relative first match is a failure and the search does not continue to later absolute entries). They agree with plan D-4 (plan.md:L53-59). The one remaining wording nit is in the plan, not a REQ (R2-O1). |
| Completeness | 1.0 | 1.0 | All sections are present. The new Out of Scope bullet at L384 scopes the absolute-path guard to the bare-name branch; plan R-8 (L183) and R-7 (L182) are added. |
| Testability | 1.0 | 1.0 | All four AC-AIPH-008 sub-cases (L336-339) are binary, and their runnability was measured (see below). The new clause in AC-AIPH-006 (L314) is binary: two consecutive `Deliver` calls record different directory names. |
| Traceability | 1.0 | 1.0 | REQ-AIPH-001.2 is pinned by AC-AIPH-001 L251-252. REQ-AIPH-006.1 is pinned by AC-AIPH-006 L314. REQ-AIPH-007.3 is pinned by AC-AIPH-008(c), and 007.4 by 008(d) (L342). The §7 table maps 8 REQs to 8 ACs (L408-415). |

The aggregate of 0.97 is reported rather than 1.0 to reflect the two optional nits below.

## Auditor Measurement (this run, this tree)

A throwaway module was written to `/Users/jay/workspace/imoogi-emacs/.audit-tmp-lookpath/` (a `go.mod` with
`go 1.26` and `lp_test.go`). It was run with `GOWORK=off GOFLAGS=-mod=mod go test -v -count=1 ./...` on go1.26.4
darwin/arm64 and then deleted: `rm -rf .audit-tmp-lookpath`, after which `ls` reported "No such file or directory"
and `git status --short | grep -c audit-tmp` printed 0. Verbatim output:

```
lp_test.go:27: default PATH=.:Q -> "ec" err=exec: "ec": cannot run executable found relative to current directory errDot=true
lp_test.go:35: GODEBUG=execerrdot=0 via t.Setenv, PATH=. -> "ec" err=<nil> abs=false
lp_test.go:42: default (after restore) PATH=. -> "ec" err=exec: "ec": cannot run executable found relative to current directory
ok  	lp	0.453s
```

What the output shows:
- AC-AIPH-008(c) is runnable and matches REQ-007.3. With `PATH=".:Q"` and `ec` in both the cwd and Q, LookPath
  returns ErrDot on the `.` match and does not fall through to `Q/ec`.
- AC-AIPH-008(d) is runnable. `t.Setenv("GODEBUG", "execerrdot=0")` changes LookPath behavior at runtime within the
  same test process: it returns `"ec"` with a nil error and a non-absolute path, so only the REQ-007.4
  `filepath.IsAbs` guard can turn it into a failure. The case therefore discriminates between an implementation
  with the guard and one without it.
- `t.Setenv` cleanup restores the default. The next test got ErrDot again, so (d) cannot leak into other tests.
- The source citation in HISTORY and plan §5.3 checks out: `go1.26.4 runtime/runtime.go` `syscall_runtimeSetenv`
  is at lines 202-209, and on `GODEBUG` it calls `godebugEnv.Store(p)` and then `godebugNotify(true)`.

## Defects Found (structured defect-list)

R2-O1. PLAN-WORDING (optional) — plan.md:L167. The plan says that under `execerrdot=0`, "`LookPath` 는 `./ec` 를 오류
없이 돌려주고" (LookPath returns `./ec` without an error). The measured return value is `"ec"`, because
`filepath.Join(".", "ec")` drops the `./`. The guard outcome is the same either way, since the path is non-absolute.
Severity: minor. Class: optional.
Required fix: write "`ec`(상대 경로)" ("`ec`, a relative path") instead of `./ec`, so a run-phase test asserting the
intermediate value does not copy the wrong literal.

R2-O2. STALE-GAP-RECORD (optional) — progress.md:L27-28 and plan.md:L166. The gap "GODEBUG=execerrdot=0 taking effect
via t.Setenv … not executed" is now closed by the measurement above. Severity: minor. Class: optional.
Required fix: optionally move the entry to `closed_gaps`, citing this report.

No blocking defects.

## Regression Check (Iteration 2)

Defects from iteration 1:
- D1 (blocking) — RESOLVED.
  - REQ-AIPH-007 (L203) now says "the first matching entry decides … a first match in an empty or relative `PATH`
    entry, or any non-absolute result, as a resolution failure".
  - REQ-007.3 (L207-209) states that the search never continues to a later absolute entry and cites lp_unix.go.
  - AC-AIPH-008(c) (L338) pins `PATH` = `.` followed by Q. Q holds only `ec`, and the expected result is exit 1
    without running Q's `ec`.
  - Plan D-4 (L55-56) is consistent with this. Measured above.
- D2 — RESOLVED. The D-1 rationale is narrowed to "다른 사용자가 직접 쓸 수 있는 파일" (a file another user can write
  directly) at plan.md:L28-29, with the limitation spelled out at L30-32. D-5 carries the same caveat (L71-72). The
  new R-7 (L182) names the residual risk and the per-user-TMPDIR exception, and points to the Out of Scope entry.
- D3 — RESOLVED. REQ-AIPH-007.4 (spec.md:L210-211), plan D-4 (L57-59) and AC-AIPH-008(d) (L339) are in place. The
  existing `PATH` `emacsclient` fallback is explicitly out of scope (spec.md:L384) and recorded as R-8 (plan.md:L183).
  That is an acceptable scoping choice.
- D4 — RESOLVED. AC-AIPH-001 Then (L251-252) asserts that the log line contains ` - project=- session=- untrusted`.
  This matches the log format in 30-agent.el:L213-222: the timestamp prefix ends in a space, then `-` for the type,
  and no detail follows a rejection by throw.
- D5 — RESOLVED. AC-AIPH-006 (L314) adds the check that two consecutive `Deliver` calls record different directory
  names. Plan D-5 (L73) adds the rationale.
- D6 — RESOLVED. spec.md:L399 now cites "`findEmacsclient`(120-149행, `$EMACSCLIENT` 분기 120-127행)".
- D7 — RESOLVED. HISTORY L51-52, plan L19 and R-3 (L178), and progress.md `closed_gaps` (L29-30) record the
  measurement.
- D8 — RESOLVED. The `[Ubiquitous — negated]` explanation is gone from §3. The only remaining occurrence is the
  HISTORY line that records its removal (L33).
- D9 — RESOLVED. The §2.4 table has a new REQ-AIPC-009 · 009.4 row (L140).

No defect has stagnated. The REQ and AC counts stayed at 8 each.

## Recommendation

PASS. Must-pass rationale:
- MP-1: 8 sequential REQs (L150-L213).
- MP-2: all eight REQ sentences use GEARS patterns, including the reworded REQ-007 (L203).
- MP-3: 12 fields are present and `lint --strict` exits 0.
- MP-4: N/A.
- MP-5: the parent SPEC is `completed`.
- MP-6: syscall count 0.
- MP-7: no markers.

The single blocking defect from iteration 1 (D1) is resolved and the new sub-cases were measured runnable on
go1.26.4. R2-O1 and R2-O2 are optional and can be folded in at the orchestrator's discretion; neither requires
another audit iteration.

## Run-gate recheck (2026-09-28)

Verdict: PASS (unchanged). Score: 0.97 (unchanged).

The SPEC files are untracked, so there was no git diff to compare against. The check was a comparison with the
review-2 citations above:
- spec.md: mtime `Sep 27 22:14:20`, which is before review-2. It is still 415 lines, the cited lines L203, L210,
  L338-339 and L399 read exactly as quoted above, and there are still 8 REQs and 8 ACs.
- plan.md: still 201 lines. The spot-checked cited lines L19, L28, L55, L57, L73, L178, L182 and L183 are unchanged.
  The only edited text is §5.3 L166-167. It now reads "plan-audit 2회차가 go1.26.4 에서 실행해 확인" (verified by
  running on go1.26.4 in plan-audit iteration 2) and "`LookPath` 는 `ec`(`./` 없이)" (LookPath returns `ec`, without
  `./`). This resolves R2-O1.
- progress.md: grew from 43 to 49 lines. It now has `plan_status: audit-ready`, `plan_complete_at`, an iteration-2
  PASS 0.97 entry with `optional_findings_applied: [R2-O1, R2-O2]`, and `plan_phase_gaps: []`. The GODEBUG gap moved
  to `closed_gaps` (L35). This resolves R2-O2.
- `moai spec lint --strict .moai/specs/SPEC-AGENTIPC-002/spec.md` printed "✓ No findings" and exited 0.

The only changes are R2-O1 and R2-O2. No REQ or AC text changed, and no new defects were found.

Gap: plan.md had no pre-edit snapshot, so the claim that nothing else in it changed rests on the unchanged line count
and on spot-checking the lines review-2 cited, not on a full byte diff.
