# SPEC Review Report: SPEC-AGENTIPC-002
Iteration: 1/3
Verdict: FAIL
Overall Score: 0.86

Tier S (PASS threshold 0.75, per `spec-workflow.md` § SPEC Complexity Tier). The aggregate score clears the
threshold and all seven must-pass criteria pass. The verdict is FAIL on one blocking internal-consistency defect
(D1): REQ-AIPH-007's normative sentence and plan D-4 prescribe different results for one concrete input. It is a
one-sentence REQ rewording plus one AC sub-case, so the iteration-2 re-audit is scoped to that delta.

Reasoning context ignored per M1 Context Isolation. Operator decisions 1-3 were treated as fixed. They were checked
only for faithful encoding and not scored as design choices.

Artifacts read: `spec.md` (391 lines), `plan.md` (186 lines), `progress.md` (35 lines). For cross-reference:
parent `SPEC-AGENTIPC-001/{spec.md,plan.md,acceptance.md}`, `modules/development/30-agent.el`,
`internal/agentipc/emacsclient.go`, `internal/agentipc/emacsclient_test.go`, `internal/agentipc/agentipc.go`,
`cmd/imoogi-agent/main_test.go`, `tests/agent-test.el`, `tests/run.el`, `scripts/imoogi-editor`,
`modules/general/00-defaults.el`, `.moai/reports/sync-audit/SPEC-AGENTIPC-001-4dim.json`,
`$(go env GOROOT)/src/os/exec/{lp_unix.go,lookpath.go}` (go1.26.4).

## Must-Pass Results

- [PASS] MP-1 REQ number consistency: headings `REQ-AIPH-001`…`REQ-AIPH-008` at spec.md:L133, L141, L150, L160,
  L170, L176, L184, L192. They run in sequence with no gaps or duplicates and use 3-digit padding. There are 8 REQs,
  which is exactly at the Tier S ceiling of 8. ACs `AC-AIPH-001`…`008` at spec.md:L220, L234, L249, L267, L279, L287,
  L297, L309 also number 8, again at the ceiling.
- [PASS] MP-2 EARS/GEARS format compliance. Judged against the requirement layer (`REQ-XXX` in spec.md §3.1–3.2)
  only. The ACs in §3.3 are Given-When-Then, which is correct for the verification layer, and were graded under
  Group 4. Each REQ uses one pattern:
  - REQ-001, 003 and 008 are event-driven, for example spec.md:L135 "When the event file … is detected …, the
    receiver shall reject".
  - REQ-002, 004, 005 and 006 are ubiquitous, for example spec.md:L143 "The receiver shall apply the trust check
    only in …".
  - REQ-007 is event-driven: spec.md:L186 "When `$EMACSCLIENT` is set to a value without a `/`, the CLI shall …".
  - REQ-008 combines "shall fail … and shall not create …" (spec.md:L194).

  No informal "should" or "may" appears in any normative sentence.
- [PASS] MP-3 YAML frontmatter validity. All 12 canonical fields are present with the correct types:
  - `id` (L2), `title` quoted (L3), `version: "0.1.0"` quoted (L4), `status: draft` (L5)
  - `created: 2026-09-27` (L6) and `updated: 2026-09-27` (L7)
  - `author` (L8), `priority: P2` (L9)
  - `phase: "v0.x agent-ipc hardening"` (L10). This is a release-target label, not a prohibited lifecycle token.
  - `module` (L11), `lifecycle: spec-anchored` (L12), `tags` as a comma-separated string (L13)

  No rejected snake_case aliases are used. The optional `tier: S` (L14) and `depends_on` (L15) are well-formed.
  `moai spec lint --strict .moai/specs/SPEC-AGENTIPC-002/spec.md` printed "✓ No findings — all SPEC documents are
  valid" and exited 0.
- [N/A] MP-4 Section 22 language neutrality: the SPEC is scoped to this single project (a Go CLI plus an Emacs Lisp
  module), not to template-bound multi-language tooling. N/A auto-passes.
- [PASS] MP-5 D7 cross-SPEC reconciliation: the only external reference is SPEC-AGENTIPC-001.
  `grep '^status:' .moai/specs/SPEC-AGENTIPC-001/spec.md` returned `status: completed`, which is not
  retired/superseded/archived. spec.md §2.4 (L113-124) explicitly narrows that SPEC's clauses without modifying it.
  No BLOCKING finding.
- [PASS] MP-6 D8 cross-platform discipline: `grep -c syscall` returned 0 for both spec.md and plan.md, so D8
  auto-passes.
- [PASS] MP-7 clarification gate: `grep -rn 'NEEDS CLARIFICATION' .moai/specs/SPEC-AGENTIPC-002/` exited 1 with no
  matches. research.md does not exist, which is correct for Tier S.

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.75 | 0.75 | REQ-AIPH-007 (spec.md:L186, L190) reads two ways for a PATH in which a relative entry precedes an absolute one (D1). Every other REQ has one reading, and the order §2.3 (spec.md:L103-107) matches REQ-002.2 (L146) and the code layout (30-agent.el:L53-63, L146-157). |
| Completeness | 1.0 | 1.0 | HISTORY (L18), purpose/WHY (§1.1 L39), scope/WHAT (§1.2 L54), protocol delta (§2), REQUIREMENTS (§3.1-3.2), ACs (§3.3), Out of Scope with five `### Out of Scope — <topic>` H3s each with `-` bullets (L333-359), constraints (§5), brownfield delta (§6), traceability (§7). Frontmatter complete. |
| Testability | 1.0 | 1.0 | Every AC names a runnable selector: `ert_sel "^imoogi-agent-aiphNN-"` or `go test … -run 'TestAIPHNN'`. Results are binary: status strings, `file-attribute-size`, mode strings `600`/`700`, TMPDIR emptiness, record-file absence. No weasel words. Methods measured feasible by the auditor (see Measurements). |
| Traceability | 0.75 | 0.75 | Every REQ has at least one AC and every AC cites a real REQ (§7, L381-390). Two sub-clauses are not pinned: REQ-AIPH-007's relative-before-absolute case (D1) and REQ-AIPH-001.2's `-` log fields (D4). |

Aggregate: the arithmetic mean is 0.875 and the harmonic mean is 0.857. Reported as 0.86, which is ≥ the Tier S
threshold of 0.75.

## Auditor Measurements (this run, this tree, HEAD ecf68a9)

These are observed facts, not defects. They close several of the plan's own open questions.

1. Sparse file (resolves plan.md R-3 and progress.md `plan_phase_gaps`). Command:
   `emacs --batch -Q --eval '(… (call-process "dd" nil nil nil "if=/dev/zero" (concat "of=" p) "bs=1" "count=0" "seek=104857601") …)'`.
   Output: `dd exit 0` and `size 104857601`. The plan §5.1 first-choice method works on Emacs 30.2 on this host. The
   §5.1 fallback, which would conflict with AC-AIPH-003's Given (a 104,857,601-byte P), is not needed.
2. `defconst` dynamic binding (AC-AIPH-004 method). The same batch command, with a byte-compiled caller reading a
   `defconst` inside `(let ((aud-cap 16)) …)`, printed `let-bound defconst seen by compiled fn: t`.
3. Native compilation. The same batch printed `30.2 native=nil`. The HISTORY `user-uid` `cl-letf` measurement
   against a byte-compiled caller therefore transfers to the test run, and no subr-trampoline question arises.
4. `exec.LookPath` semantics, from `$(go env GOROOT)/src/os/exec/lp_unix.go` go1.26.4, lines ~60-76:
   - The loop returns on the first PATH entry that holds the file.
   - If that entry is relative (`""` becomes `"."`), it returns `path, &Error{file, ErrDot}` and does not continue to
     later entries.
   - The ErrDot branch is skipped when `execerrdot.Value() == "0"` (the runtime GODEBUG), in which case it returns
     the relative path with a nil error.

   This is the basis of D1 and D3.
5. Line citations verified:
   - 30-agent.el: `imoogi-agent-max-bytes` at L19, `imoogi-agent--read-file` at L47-67,
     `imoogi-agent--resolve-path` at L146-157, `imoogi-agent--visit` at L242-250 (binds `large-file-warning-threshold`
     to nil), header comment at L3-11.
   - emacsclient.go: `Deliver` at L49-86, `writeEventFile` at L91-115, `classify` at L158-180 (L163 maps every
     `"error:"` status to `ExitRejected` = 3, agentipc.go:L19). `findEmacsclient` starts at L120 and ends at L149;
     see D6.
   - emacsclient_test.go L355-362 is the bare-`ec` cwd assertion.
   - main_test.go: L91 and L364 set only absolute EMACSCLIENT values. L159-160 and L267-268 check that TMPDIR is empty.
   - 00-defaults.el L40 is the `(* 100 1024 1024)` threshold.
   - ERT helpers `--write`, `--receive-file`, `--receive-json`, `--window-state`, `--kill-visiting`,
     `--with-temp-files` exist (agent-test.el:L41, L74, L65, L103, L357, L83). `ac10` is the last test (L574).
   - SPEC-001 clauses cited in §2.4 all exist: REQ-AIPC-001.1, 003.5, 007, 008.1, 009(.3), 012(.1/.4), 013, 014.2.
   - 4dim finding indices #0, #1, #2-#5 match the global-indexed `findings` array.
6. Operator decisions are encoded faithfully:
   - (1) spec.md:L92, L152, L154, L158 and §1.3 L66-69, with `too-large` unchanged.
   - (2) §1.3 L65, REQ-007/008 and the §4 imoogi-editor carve-out. The wording defect is D1.
   - (3) spec.md:L91, L135-147, L178.

Regression reasoning against SPEC-AGENTIPC-001 ACs is based on reading the code; no tests were run.
- TestAC13DeliveryArgumentsPermissionsAndCleanup checks four things: an absolute path, perm 600, content, and that the
  path is gone after Deliver. All four still hold with a file inside a directory removed by `RemoveAll`.
- TestAC14UnwritableTempDirIsNotDelivered still holds: `os.MkdirTemp` fails under a 0500 TMPDIR, so the fake is not
  run and the code is 1.
- The main_test.go TMPDIR-empty checks still hold.
- Only emacsclient_test.go L355-362 changes, which the DoD declares (spec.md:L326).
- ERT fixtures use `make-temp-file`, which the HISTORY measured as mode 600, so they stay trusted under umask 022. R-1
  covers umask 002.

## Defects Found (structured defect-list)

D1. REQ-EARS-CONSISTENCY (must-fix) — spec.md:L186, L190; plan.md:L49; spec.md:L315 — REQ-AIPH-007's normative
sentence says the bare name is resolved "only by searching the directories in `PATH`, ignoring any `PATH` entry that
is relative to the current directory, and shall use the absolute path that search returns". Take `PATH=".:P"` with
an executable `ec` both in the cwd D and in P. Read literally ("ignoring" means skip), the REQ resolves to `P/ec`.
Plan D-4 (plan.md:L49) and operator decision 2 use `exec.LookPath`, which returns `ErrDot` on the first, relative
match and never reaches P (lp_unix.go, measurement 4). D-4 then fails with exit 1. Two reasonable implementers would
diverge, and the plan contradicts its own REQ. No AC pins the case: AC-AIPH-008(b) (L315) puts only `emacsclient`,
not `ec`, in the absolute directory. REQ-007.3 (L190, "only found in") does not settle it either. Severity: major.
Class: blocking.
Required fix: reword REQ-AIPH-007 to match exec.LookPath first-match semantics. For example: "…the CLI shall resolve
it only by a `PATH` search in which the first matching entry decides, and a first match in an empty or relative
`PATH` entry is a resolution failure; on success it shall use the absolute path that search returns". Align
REQ-007.3 with this. Add sub-case (c) to AC-AIPH-008: `EMACSCLIENT=ec`, `PATH` = `.` followed by an absolute
directory that also holds an executable `ec`. Then: exit 1, one diagnostic line naming `EMACSCLIENT` and `ec`, no
program run, no TMPDIR entry. Do not change plan D-4, which already matches operator decision 2.

D2. PLAN-THREAT-MODEL-ACCURACY (should-fix) — plan.md:L27-28, L60-62; plan.md §6 R-table (L160-167) — The D-1
rationale says that if the real file is user-owned and not group- or other-writable, "내용은 다른 사용자가 정할 수
없다" (another user cannot determine its content). That does not hold under the SPEC's own t19 threat model
(spec.md:L47-49): an untrusted repository checkout or a downloaded file is user-owned, mode 0644, and has
attacker-chosen content. The scenario, which is reasoning from code and was not exploited:
- After a CLI timeout the 0700 directory is removed (D-5).
- A local user recreates the same-named directory in a shared `/tmp`.
- They plant `event.json` as a symlink to such a file.
- The late server read passes `file-regular-p`/`file-readable-p`, which follow links (30-agent.el:L56-57), and the
  truename-based trust check (plan.md:L30), so the event is dispatched.

Impact is bounded to display, notices and log lines, because no event value is evaluated. The mitigations
(directory-owner check, link refusal) are already excluded in spec.md §4 (L351). Severity: minor. Class: optional.
Required fix: correct the D-1/D-5 sentence to "a file whose content another user can write directly" and add an R-7
row naming the residual risk (a symlink to a user-owned file with attacker-chosen content after the CLI directory is
removed), pointing to the Out of Scope entry.

D3. PLAN-GODEBUG-DEPENDENCE (should-fix) — plan.md:L18, L49; emacsclient.go:L128-129 — The guarantee in REQ-AIPH-007.3
depends on the ErrDot default. At runtime `GODEBUG=execerrdot=0` makes `exec.LookPath` return the relative path with
a nil error (lp_unix.go `execerrdot.Value() != "0"` branch), and that holds regardless of `go.mod`. The same applies
to the existing `PATH` fallback at L129. Severity: minor. Class: optional.
Required fix: in plan D-4, require `filepath.IsAbs(found)` after a successful `exec.LookPath` for the bare-name
branch, treating a non-absolute result as not found. Optionally apply the same guard to the `emacsclient` PATH
fallback, or state the GODEBUG dependence as an accepted residual in §6.

D4. AC-COVERAGE-REQ-001.2 (optional) — spec.md:L138 vs L229-230 — REQ-AIPH-001.2 requires the rejection log line's
`type`, `project` and `session` to be `-` because the content was never read. AC-AIPH-001 only asserts "exactly one
line containing `untrusted`". An implementation that parsed the file before rejecting would still fail the
`insert-file-contents` = 0 check, so the risk is low. Severity: minor. Class: optional.
Required fix: add to AC-AIPH-001 Then: "the added log line contains ` - project=- session=- untrusted`" (or the
equivalent pattern for the REQ-AIPC-007 format).

D5. AC-COVERAGE-REQ-006.1 (optional) — spec.md:L180 vs L287-293 — "예측 가능한 고정 이름을 쓰지 않는다" (no
predictable fixed name) is not observed by AC-AIPH-006. Only the `imoogi-agent-` prefix is checked. Severity: minor.
Class: optional.
Required fix: either add "two consecutive `Deliver` calls record different parent directory names" to AC-AIPH-006,
or demote the clause to a design note that `os.MkdirTemp` satisfies (plan D-5).

D6. CITATION-RANGE (optional) — spec.md:L374, plan.md:L96 — `findEmacsclient` is cited as lines 120-127. The
function spans 120-149 (emacsclient.go); 120-127 is only its `$EMACSCLIENT` branch. Severity: minor. Class:
optional.
Required fix: write "`findEmacsclient`(120-149행, `$EMACSCLIENT` 분기 120-127행)" (lines 120-149, `$EMACSCLIENT`
branch at lines 120-127).

D7. STALE-GAP-RECORD (optional) — progress.md:L20-21, plan.md:L19, L164; spec.md:L35 — The sparse-file gap is closed
by auditor measurement 1 (`dd` exit 0, size 104857601 on Emacs 30.2). Severity: minor. Class: optional.
Required fix: optionally record that the plan-audit measured the method, so run-phase M1 does not need to repeat
R-3's first task. The §5.1 fallback can stay as a contingency.

D8. DANGLING-PATTERN-NOTE (optional) — spec.md:L128-129 — The note "`[Ubiquitous — negated]` 는 `shall not` 형식이다"
(that label means the `shall not` form) explains a label that no REQ in this SPEC carries. Severity: minor. Class:
optional.
Required fix: remove the note, or keep it only if a REQ is relabeled.

D9. PARENT-CLAUSE-TABLE (optional) — spec.md:L117-124 — The §2.4 table lists the SPEC-AGENTIPC-001 clauses this SPEC
narrows. It omits REQ-AIPC-009, specifically 009.4 "통과한 경로는 열어서 보여 주기만 한다" (a path that passes is
only opened and shown). REQ-AIPH-003 now rejects some policy-passing paths as `payload-too-large`. Severity: minor.
Class: optional.
Required fix: add a row "REQ-AIPC-009(.4) | a path that passes the path policy is still rejected with
`payload-too-large` above the cap (REQ-AIPH-003)".

## Regression Check (Iteration 2+ only)

N/A — iteration 1.

## Recommendation

FAIL, scoped to one blocking item. Instructions for manager-spec:

1. (D1, blocking) Reword REQ-AIPH-007 (spec.md:L186) and REQ-AIPH-007.3 (L190) to exec.LookPath first-match
   semantics: a first match in an empty or relative `PATH` entry is a resolution failure, and there is no skipping to
   later entries. Add AC-AIPH-008 sub-case (c): `PATH` = `.` followed by an absolute directory that also holds `ec`,
   expecting exit 1 with nothing run. Plan D-4 needs no change. If the operator instead wants "skip relative entries
   and keep searching", that is a change to decision 2 and must go back to the operator; do not decide it in the SPEC.
2. (Optional, recommended in the same pass because they are cheap) D2 (correct the D-1 rationale and add an R-7
   residual row), D3 (IsAbs guard in D-4), and D4 (assert the `-` log fields in AC-AIPH-001).
3. D5-D9 are at the orchestrator's discretion.

Iteration-2 re-audit scope: the D1 delta, plus a regression check that the REQ/AC counts stay ≤ 8 each and that
`moai spec lint --strict` stays clean.

Must-pass rationale: MP-1 has 8 sequential REQs (L133-L192). MP-2 finds all eight REQ sentences in GEARS form. MP-3
has 12 fields and a clean lint. MP-4 is N/A. MP-5: the parent is `completed`. MP-6: syscall count is 0. MP-7: no
markers.
