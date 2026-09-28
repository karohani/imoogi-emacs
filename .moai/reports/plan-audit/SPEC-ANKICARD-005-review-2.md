# SPEC Review Report: SPEC-ANKICARD-005
Iteration: 2/3
Verdict: PASS
Overall Score: 0.93  (Tier S threshold 0.75; iteration 1 = 0.81 → no score regression, STOP rule not triggered)

Reasoning context ignored per M1 Context Isolation. Iteration 2 is scoped to the iteration-1 defect
delta (D1–D10) plus a regression check, per the Retry Loop Contract. Operator decisions stated as
final ((a) `skipped`; (b) `depends_on` removed) were checked for faithful encoding only.

Artifacts: `spec.md` v0.1.1 (`version: "0.1.1"`, spec.md:4; HISTORY v0.1.1 at spec.md:20–53), `plan.md`,
`progress.md`. Baseline HEAD `c010009`.

Mechanical evidence this run:
- `moai spec lint --strict .moai/specs/SPEC-ANKICARD-005/spec.md` → `✓ No findings — all SPEC documents are valid`, exit 0.
- `grep -c '^#### REQ-'` → 8 (spec.md:132–243); `grep -c '^#### AC-'` → 8 (spec.md:261–350). Tier S ceilings ≤8 / ≤8 hold.
- Scratch-module probes (orgdoc package copied verbatim to the session scratchpad; no repo writes other than this report):
  - `RenderWithOptions(Cloze, "T", "#+BEGIN_EXTRA\nnote\n", Direction=->)` → `*orgdoc.MultilineAnswerMissingError`.
  - `Render(Cloze, …, "Q\n#+BEGIN_SRC org\n,#+BEGIN_EXTRA\n#+END_SRC\n")` → err nil, Text contains `<pre>\n#+BEGIN_EXTRA\n</pre>` (comma stripped); split leaves no residue.
  - `…#+BEGIN_EXAMPLE\n,#+END_EXTRA\n#+END_EXAMPLE\n` → `<pre class="example">\n#+END_EXTRA\n</pre>` (comma stripped).
  - `…#+BEGIN_SRC org\n#+BEGIN_EXTRA\n#+END_SRC\n` → split leaves the BEGIN in `rest` (so the new check will reject it — the R-2 case).
  - Copy of orgdoc with `multiline_test.go` lines 163–211 (the five DD-6 corpus rows) deleted:
    `go test ./internal/anki/orgdoc -count=1 -cover` → `ok … coverage: 100.0% of statements`.

## Must-Pass Results

- [PASS] MP-1 REQ number consistency: REQ-AKX-001..008 at spec.md:132, 150, 164, 175, 192, 213, 220, 243 — sequential, no gaps/duplicates.
- [PASS] MP-2 GEARS compliance (requirement layer only; §4 GWT ACs not graded here): requirement sentences unchanged from iteration 1 apart from sub-clauses; REQ-AKX-005 still "When an entry is rejected … the back end shall report it as skipped …" (spec.md:193–196).
- [PASS] MP-3 frontmatter: spec.md:2–15 — all 12 canonical fields present and correctly typed; `version: "0.1.1"`; `depends_on` removed; `related_specs: [SPEC-ANKICARD-003, SPEC-ANKICARD-004]`; lint clean.
- [N/A] MP-4 language neutrality: project-specific Go + Elisp.
- [PASS] MP-5 D7: referenced SPECs -001 (in-progress), -003 (in-progress), -004 (draft); none retired/superseded/archived.
- [PASS] MP-6 D8: no `syscall` in spec.md/plan.md.
- [PASS] MP-7 clarification gate: no `[NEEDS CLARIFICATION` in plan.md; no research.md (Tier S).

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.90 | 0.75–1.0 | Both operator decisions now stated as decisions (spec.md:26–33; REQ-AKX-005.2 spec.md:202–207; DD-3). Escape advice now accurate (REQ-AKX-007.2 spec.md:228–234). |
| Completeness | 0.95 | 1.0 | SPEC-004 acceptance.md:497/118 reconciliation recorded (DD-8, sync follow-ups); src-block case in R-2; coverage argument in DD-6. |
| Testability | 0.90 | 1.0 | AC-AKX-004 (a)/(b) now discriminating and state their baseline outcomes (spec.md:291–299); AC-AKX-007 asserts the full table message (spec.md:343–347); AC-AKX-005 asserts no `deleteNotes`. |
| Traceability | 0.95 | 1.0 | §7 table maps every REQ to ≥1 AC; REQ-AKX-004's `multiline_answer_missing` half now covered by AC-AKX-004(b). |

Aggregate 0.93 (arithmetic mean 0.925; harmonic 0.924). Iter 1 → iter 2: 0.81 → 0.93, an increase — the
LEAN score-regression STOP rule does not fire.

## Defects Found (structured defect-list)

No blocking defects. No new defects introduced by v0.1.1.

Residual (optional, carried as accepted debt — not a defect):
- R-D9 — spec.md:48–51 — Tier S file-count guidance (< 5 files) is exceeded (6 production files + tests); HISTORY v0.1.1 now records this explicitly and keeps Tier S on LOC grounds. Guidance only. — Severity: minor — Class: optional — Required fix: none.

## Regression Check (Iteration 2+ only)

Defects from iteration 1:
- D1 (AC-AKX-004(b) non-discriminating) — RESOLVED: body is now `#+BEGIN_EXTRA\nnote\n` (spec.md:293) with the baseline outcome stated (spec.md:293–296) and the sentence "(a)와 (b)는 기준선에서 경쟁 코드를 내는 본문이므로, 판정이 해당 진단보다 먼저 돌 때만 통과한다" (spec.md:298–299). Probe confirms `*orgdoc.MultilineAnswerMissingError` at baseline for this body.
- D2 (depends_on gate unstated) — RESOLVED by operator decision: `depends_on` removed from frontmatter (spec.md:2–15 has none; `grep depends_on` finds only HISTORY prose at spec.md:26,29 and `progress.md:16 depends_on: []`); -003 moved to `related_specs`; HISTORY spec.md:26–29 explains that only -003's code (on main at `08f7504`) is needed, so the run-phase pre-flight will not block.
- D3 (DD-3 `failed` cost misstated) — RESOLVED: the erroneous cost paragraph was removed; DD-3 now records `skipped` as the operator decision and cites migrate.go:217 (`code, _ := renderError(err)`) as the reason both paths report the same value (plan.md DD-3; REQ-AKX-005.2 spec.md:202–207). Verified against migrate.go:214–218.
- D4 (SPEC-004 acceptance.md:497 unreconciled) — RESOLVED: DD-8 quotes acceptance.md:497 verbatim and cites acceptance.md:118; the sync follow-up list names both rows for -004's resumption. SPEC-004 files remain on the PRESERVE list (not edited).
- D5 (comma-escape advice misleading) — RESOLVED: REQ-AKX-007.2 (spec.md:228–234) and the DD-7 draft now recommend verbatim `=#+BEGIN_EXTRA=` or mid-line in prose, and comma escape only inside `#+BEGIN_SRC` / `#+BEGIN_EXAMPLE`. Probe confirms go-org strips the comma inside both block types (`<pre>\n#+BEGIN_EXTRA\n</pre>`, `<pre class="example">\n#+END_EXTRA\n</pre>`) and keeps it in prose.
- D6 (src-block false positive unnamed) — RESOLVED: R-2 names the case with the exact body, states it was previously rendered literally and will now be rejected, and gives the comma escape as the way out. Probe confirms both halves (unescaped → residue in `rest`; escaped → no residue, clean render).
- D7 (weak ERT assertion) — RESOLVED: AC-AKX-007 requires the report to contain the full return value of `(imoogi-error-message "extra_block_unbalanced")` and explains why the `BEGIN_EXTRA` substring is insufficient (spec.md:343–347).
- D8 (no-delete assertion) — RESOLVED: AC-AKX-005 Then adds "`writeSequence`에 `deleteNotes`도 없으며" (spec.md:313–314).
- D9 (Tier S file count) — ACKNOWLEDGED in HISTORY (spec.md:48–51); guidance-only, accepted.
- D10 (extra_alias_test.go path; corpus-row coverage) — RESOLVED: DD-6's last row gives full paths including `internal/anki/planner/extra_alias_test.go`; DD-6 adds a measured coverage argument, independently reproduced here (100.0% with rows 163–211 removed), plus a fallback if new branches drop coverage.

Regression scan over the v0.1.1 delta: REQ normative sentences unchanged; REQ/AC counts unchanged (8/8);
traceability table unchanged and still complete; no new cross-SPEC references beyond -001/-003/-004; no new
file-path citations that fail to resolve; lint clean. No regressions found.

## Recommendation

PASS. All seven must-pass criteria pass (evidence above); the aggregate 0.93 exceeds the Tier S threshold
0.75 and improves on iteration 1 (0.81). Every iteration-1 must-fix (D1, D2) and should-fix (D3–D6) is
resolved with evidence, and the optional D7, D8, D10 were also applied. D9 stays as accepted, documented
guidance debt. The SPEC is ready for the Implementation Kickoff Approval gate. That gate is still required
and is not bypassed by this verdict. The run-phase depends_on pre-flight trivially passes now that
`depends_on` is absent.
