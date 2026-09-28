# SPEC Review Report: SPEC-ANKICARD-005
Iteration: 1/3
Verdict: FAIL
Overall Score: 0.81  (Tier S threshold 0.75 — the FAIL is defect-routed, not score-routed)

Reasoning context ignored per M1 Context Isolation. Operator decisions stated as FINAL in the
audit brief were checked for faithful encoding only, not scored as design choices.

Baseline: HEAD `c010009`. Artifacts read: `spec.md` (376 lines), `plan.md` (308 lines),
`progress.md` (34 lines). Tier S (spec.md:14) — ACs inline in spec.md §4.

Mechanical evidence gathered this run:
- `moai spec lint --strict .moai/specs/SPEC-ANKICARD-005/spec.md` → `✓ No findings — all SPEC documents are valid`, exit 0.
- Probe: `extra.go` + orgdoc package copied verbatim into a scratch module (scratchpad, no repo writes),
  `splitExtraBlocks` exported via a shim, 20 body shapes run through it, plus `Render` /
  `RenderWithOptions` on the AC bodies. Results are quoted per finding below.

## Must-Pass Results

- [PASS] MP-1 REQ number consistency: `#### REQ-AKX-001` (spec.md:99) through `#### REQ-AKX-008` (spec.md:206), 8 entries, sequential, no gaps or duplicates, consistent 3-digit padding. ACs `AC-AKX-001`–`008` (spec.md:224–304) likewise.
- [PASS] MP-2 GEARS compliance (judged on the REQ layer only; §4 Given–When–Then ACs are the verification layer and not penalized): 001 "When … the back end shall reject" (spec.md:100), 002 "When … shall reject" (117), 003 "The supplementary split shall not change …" (131), 004 "The unpaired-marker check shall run …" (142), 005 "When an entry is rejected … shall report …" (159), 006 "The protocol package shall declare …" (179), 007 "The front end's code-to-message table shall carry …" (186), 008 "The back end shall not change …" (207). All five-pattern conformant.
- [PASS] MP-3 frontmatter: spec.md:2–13 carries all 12 canonical fields with correct types — `id`, `title` (quoted), `version: "0.1.0"` (quoted semver), `status: draft`, `created: 2026-09-28`, `updated: 2026-09-28`, `author: jay`, `priority: P2`, `phase: "v0.2.0 target"` (release label, not a lifecycle token), `module` (path-like string), `lifecycle: spec-anchored`, `tags` (comma string). No snake_case aliases. Optional `tier: S`, `depends_on`, `related_specs`. Lint clean.
- [N/A] MP-4 language neutrality: project-specific Go + Elisp SPEC, not template-bound or multi-language tooling.
- [PASS] MP-5 D7: references extracted — SPEC-ANKICARD-001 `in-progress`, -003 `in-progress`, -004 `draft`. None retired/superseded/archived; all exist. No BLOCKING. (Non-D7 cross-SPEC tension with -004 recorded as D4 below.)
- [PASS] MP-6 D8: `grep -c syscall` → spec.md 0, plan.md 0. Auto-pass.
- [PASS] MP-7 clarification gate: `grep -n 'NEEDS CLARIFICATION' plan.md` → no match (exit 1); no research.md at Tier S.

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.85 | 0.75–1.0 | Terms defined precisely (spec.md:77–91); marker grammar pinned to the split's own (REQ-AKX-001.1, spec.md:104–106; DD-5). Deductions: pending question (b) unstated (D2); DD-3's cost of the `failed` alternative misstated (D3). |
| Completeness | 0.85 | 0.75–1.0 | HISTORY (19), WHY/overview (54–75), REQs (93), ACs (219), Out of Scope with 5 `### Out of Scope — …` H3s each carrying `-` bullets (315–351), Constraints (353), Traceability (361). Deductions: depends_on consequence absent (D2); SPEC-004 acceptance.md:497 not reconciled (D4). |
| Testability | 0.80 | 0.75 | Every AC carries a runnable command and binary Then. AC-AKX-004(b) is binary but non-discriminating (D1). No weasel words found. |
| Traceability | 0.75 | 0.75 | §7 table (spec.md:363–372) maps every REQ to ≥1 AC and every AC names valid REQs. But REQ-AKX-004's `multiline_answer_missing` half has no discriminating AC (D1) — effectively one REQ partially uncovered. |

Aggregate 0.81 (arithmetic mean 0.8125, harmonic 0.810; ≥ 0.75 Tier S threshold). Verdict FAIL is driven by the blocking defect D1
(M6: blocking findings are fixed before the verdict is revisited), plus D2 which the audit brief
explicitly asked to be stated in the SPEC.

## Verified-correct claims (evidence for what holds)

- **DD-2 residue table (plan.md:86–97) — all 10 rows match the actual regex.** Probe output against the
  copied `extraBlockPattern` (`(?ims)^[ \t]*#\+begin_extra[^\n]*\n(.*?)^[ \t]*#\+end_extra[^\n]*\n?`):
  stray END → `rest="- A\n#+END_EXTRA\n- B"`; balanced nested → `rest="- A\ntail\n#+END_EXTRA\n- B" extra="outer\n#+BEGIN_EXTRA\ninner"`;
  open2/close1 → `rest="- A\n- B" extra="first\n#+BEGIN_EXTRA\nsecond"`; sequential → `extra="one\n\ntwo"`;
  last-line BEGIN without newline stays in rest; `,#+BEGIN_EXTRA` / `=#+END_EXTRA=` / mid-line / indented-lowercase
  produce no residue. REQ-AKX-002.1's "no line-anchored END can exist in the extracted interior" holds by the
  non-greedy construction.
- **Precedence vs code order.** `Render` Cloze arm: split (orgdoc.go:73) → marker gate (74–76).
  `RenderWithOptions`: split (multiline.go:602) → compose (603–606, `MultilineAnswerMissingError`) → gate (607–609).
  Option gate precedes render (planner.go:286 before 295). REQ-AKX-004 and DD-4 describe this correctly.
- **Byte-identity (REQ-AKX-008).** The check only adds an error return after an unchanged `splitExtraBlocks`;
  bodies without residue flow through the same code. Golden corpus outputs contain zero markers
  (`grep -c _EXTRA` → render-golden.json 1 hit = the key name `cloze_with_extra_block`; others 0), and every
  EXTRA-bearing test input outside the two multiline test files is balanced (extra_test.go:45,67,78,98,110;
  golden_baseline_test.go:54; multiline_golden_test.go:71; planner/extra_alias_test.go:27) — so AC-AKX-008's
  "these files unchanged and passing" is achievable.
- **Skip contract.** `renderError` doc (planner/multiline.go:42–54) defines skip as a user-correctable content
  condition; both existing render-time codes map to skip (55–63). Registry-minus-census deletion
  (planner.go:111, 578–598) excludes census-reported IDs, so REQ-AKX-005.3 holds.
- **Emacs side.** `imoogi--keyed-error-lines` renders `key (title): <table message>` (imoogi.el:157–181);
  writeback reacts only to `"added"` (imoogi-writeback.el:75); ERT names cited in AC-AKX-007 exist
  (anki-error-test.el:52, 58, 102, 141; anki-sync-error-test.el:196); the cross-check regex matches a
  `CodeExtraBlockUnbalanced = "extra_block_unbalanced"` constant; `imoogi-error-test-full-d5-table-present`
  (131–138) checks membership only, so a new entry breaks nothing.
- **Cited tests/lines** exist: TestRenderErrorMapping (planner/multiline_test.go:391), TestMultilineDanglingExtraMarker (446),
  TestMultilineUnbalancedOpeningLeavesTheQuestionClean (530), TestMultilineSequentialExtraBlocksKeepEveryAnswer (566);
  orgdoc corpus rows 163–223 and collapse row 324–332; helpers `runOne`/`expectSkippedWithCode`/`expectAccepted`
  (card_options_test.go:37,67,83); `writeSequence` at fake_client_test.go:238; `protocol.Version = 2` (protocol.go:21).
  Commits `08f7504`, `5d3868f` exist.
- **Operator decisions faithfully encoded:** (1) no nesting — REQ-AKX-003 + Out of Scope; (2) reject that note only,
  others proceed — REQ-AKX-001/005; (3) Emacs table + bidirectional pairing test — REQ-AKX-007, AC-AKX-007;
  (4) t14 pinned tests replaced — DD-6. REQ-AKX-002 extends decision 2 to the extracted side; this is disclosed
  (spec.md:47–48) with DD-2 rationale — faithful-with-disclosed-extension.
- Tier S ceilings: 8 REQs / 8 ACs (≤ 8 / ≤ 8).

## Defects Found (structured defect-list)

D1. AC-004B-NONDISCRIMINATING — spec.md:255–256 (AC-AKX-004 item (b)) — The body `#+BEGIN_EXTRA\nnote\n- A\n` does NOT produce `multiline_answer_missing` at baseline: probe `RenderWithOptions(Cloze, "T", "#+BEGIN_EXTRA\nnote\n- A\n", Direction=->)` → `err=<nil>` (the `- A` list is found). So the AC passes whether the new check runs before or after `composeMultiline`, and REQ-AKX-004's ordering claim against `multiline_answer_missing` (REQ-AKX-004.1, spec.md:146–150) is left unverified. — Severity: major — Class: blocking — Required fix: replace item (b)'s body with one that yields `MultilineAnswerMissingError` at baseline and carries residue, e.g. `#+BEGIN_EXTRA\nnote\n` (probe: `*orgdoc.MultilineAnswerMissingError`; the unterminated BEGIN stays in `rest` and would be flagged). Optionally state in the AC that the baseline outcome of each row is the competing code, so the discriminating property is explicit.

D2. DEPENDS-ON-GATE-UNSTATED — spec.md:15, plan.md § C (210–221) — `depends_on: [SPEC-ANKICARD-003]` while -003 is `status: in-progress` (SPEC-ANKICARD-003/spec.md:5); the run-phase Depends_on Pre-flight treats anything but `completed` as unfulfilled and blocks. `grep -i "depends|pre-flight|의존|선행|ignore-deps"` over spec/plan/progress finds only the two frontmatter lines — the pending question is stated nowhere. — Severity: major — Class: blocking (brief asked that pending questions be stated clearly) — Required fix: add one HISTORY bullet (or plan § C line) naming the block and both resolutions: (i) drop `depends_on` (the code SPEC-005 builds on is on main at `c010009`; -003's status, not its code, is what blocks), or (ii) run with `--ignore-deps` and the rationale logged to `.moai/logs/depends-on-override.log`. Both are one-line changes; do not resolve it in the SPEC.

D3. DD3-FAILED-COST-UNDERSTATED — plan.md:131–133 (DD-3), spec.md:168–172 (REQ-AKX-005.2) — DD-3 says choosing `failed` changes "매핑 한 줄과 AC-AKX-005/006의 기대값만". Incorrect: the migration path ignores the skip flag and always returns `skipped` (migrate.go:217 `code, _ := renderError(err)` → `return skipped, …`). Flipping to `failed` would also require a migrate.go change (or accepting that the two paths diverge, contradicting REQ-AKX-005.1's "same mapping"), a rewrite of REQ-AKX-005's normative text ("report it as skipped", spec.md:160), and AC-AKX-006(a). Still cheap, but not as cheap as stated. — Severity: minor — Class: optional (should-fix; both answers remain cheap — only the cost sentence is inaccurate) — Required fix: correct DD-3's cost sentence to list migrate.go:214–218, REQ-AKX-005 text, AC-AKX-005, AC-AKX-006(a), and the mapping row.

D4. SPEC004-ACCEPTANCE-ROW-UNRECONCILED — plan.md:201–208 (DD-8), spec.md:343–351, plan.md:305–306 — DD-8 and the sync follow-ups cite only SPEC-ANKICARD-004 spec.md:972–980 and REQ-SW-001.7, but SPEC-004 acceptance.md:497 explicitly pins the opposite outcome: "Dangling `#+END_EXTRA` between two arrow lines | Both lines are still arrow lines; the marker renders as visible literal text on its own line". Under REQ-AKX-001.4 (all split paths) that row becomes false if swift routes through the same split. Not D7-BLOCKING (004 is `draft`), and PRESERVE correctly forbids editing 004. — Severity: minor — Class: optional — Required fix: name `SPEC-ANKICARD-004/acceptance.md:497` (and acceptance.md:118's reference to "the precedent SPEC-ANKICARD-003 set") in DD-8 and the sync follow-up list, so 004's resumption revises the acceptance row, not only the prose.

D5. COMMA-ESCAPE-ADVICE-MISLEADING — plan.md:190–191 (DD-7 draft message), spec.md:195–196 (REQ-AKX-007.2) — The message says "To show the marker text itself on a card … escape it with a leading comma (`,#+BEGIN_EXTRA')". Probe: `,#+BEGIN_EXTRA` inside a valid block renders `<p>,#+BEGIN_EXTRA</p>` — the comma stays visible; Org only strips the comma inside src/example blocks. — Severity: minor — Class: optional — Required fix: in REQ-AKX-007.2 and DD-7, recommend verbatim `=#+BEGIN_EXTRA=` (renders `<code class="verbatim">#+END_EXTRA</code>` — clean) or mid-line for prose, and comma-escape only inside src/example blocks.

D6. SRC-BLOCK-FALSE-POSITIVE-UNNAMED — plan.md:276–277 (R-2), spec.md:329–330 — A lone line-head marker inside `#+BEGIN_SRC` (e.g. a card teaching Org syntax) is now rejected: probe `"Q {{c1::x}}\n#+BEGIN_SRC org\n#+BEGIN_EXTRA\n#+END_SRC\n"` → `rest` keeps the BEGIN → flagged. Out of Scope preserves the split's pre-existing treatment, but R-2 does not name this new rejection. — Severity: minor — Class: optional — Required fix: add the src-block case to R-2 with its escape (`,#+BEGIN_EXTRA` inside the src block).

D7. AC007-WEAK-ASSERTION — spec.md:299–301 — The new ERT asserts the report contains `BEGIN_EXTRA`; the existing `multiline_answer_missing` table message also contains `#+BEGIN_EXTRA` (imoogi-error.el:68), so a wrong-code mapping could still satisfy the substring. — Severity: minor — Class: optional — Required fix: assert the report contains `(imoogi-error-message "extra_block_unbalanced")` verbatim (or a phrase unique to it).

D8. AC005-NO-DELETE-ASSERTION — spec.md:268–274 — REQ-AKX-005.3 (not a delete candidate) is argued from code but AC-AKX-005 asserts only "no updateNoteFields for the first note". — Severity: minor — Class: optional — Required fix: add "and no `deleteNotes` in `writeSequence`" to the Then.

D9. TIER-S-FILE-COUNT — spec.md:14, plan.md § F — Touch set is ~6 production files (protocol.go, extra.go, orgdoc.go, orgdoc/multiline.go, planner/multiline.go, imoogi-error.el) plus ~8 test files, beyond Tier S's "< 5 files" guidance (LOC likely within). Guidance only. — Severity: minor — Class: optional — Required fix: none required; optionally note the file-count exception in HISTORY.

D10. MISC-CITATION — plan.md:174 (DD-6 last row) — "`extra_alias_test.go`" is listed without its package; it lives at `internal/anki/planner/extra_alias_test.go` (no `orgdoc/extra_alias_test.go` exists). Also DD-6 removes 5 scanner corpus rows without measuring the effect on the orgdoc 100.0% floor AC-AKX-008 demands. — Severity: minor — Class: optional — Required fix: give the full path; optionally note that the removed rows' branches are covered elsewhere (or keep them as scanner-level facts).

Note on AC placement: ACs live in spec.md §4 while the tier table's wording says "AC inline in spec.md §3". Lint is clean and the Group A AC grep finds them; the §-number in the tier table is illustrative. Acceptable, no defect.

## Regression Check (Iteration 2+ only)

N/A — iteration 1.

## Recommendation

FAIL on defects, not on score: all seven must-pass criteria pass and the aggregate (0.81) clears the Tier S
threshold (0.75). Iteration 2 should be scoped to this defect delta.

Routing (must-fix / should-fix / optional):

Must-fix (blocking, before re-audit):
1. D1 — spec.md:255–256: change AC-AKX-004(b)'s body to `#+BEGIN_EXTRA\nnote\n` (baseline: `multiline_answer_missing`), so the AC distinguishes check-before-compose from check-after-compose.
2. D2 — add a HISTORY bullet (or plan § C line) stating that `depends_on: [SPEC-ANKICARD-003]` (in-progress) will block the run-phase pre-flight, and list both resolutions (drop the field; or `--ignore-deps` with logged rationale). Leave the choice to the operator.

Should-fix:
3. D3 — correct DD-3's cost sentence: the `failed` alternative also needs migrate.go:214–218 (it hardcodes `skipped`), REQ-AKX-005 text, and AC-AKX-006(a), besides the mapping row and AC-AKX-005.
4. D4 — cite SPEC-ANKICARD-004 acceptance.md:497 (pins the opposite outcome) in DD-8 and the sync follow-ups.
5. D5 — REQ-AKX-007.2 / DD-7: comma escape leaves the comma visible in prose; recommend verbatim `=…=` or mid-line, comma only inside src/example blocks.
6. D6 — name the src-block false positive in R-2.

Optional: D7 (tighter ERT assertion), D8 (no-delete assertion), D9 (Tier S file count), D10 (extra_alias_test.go path; corpus-row removal vs 100% floor).

Pending operator questions — status as requested:
- (a) skipped vs failed: stated clearly (spec.md:47–49, REQ-AKX-005.2, DD-3, marked as an assumption). Both answers are cheap, but the SPEC's cost statement for `failed` is inaccurate (D3).
- (b) depends_on vs in-progress SPEC-003: NOT stated anywhere (D2). Both resolutions are one-line.
