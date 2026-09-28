# SPEC Review Report: SPEC-AGENTIPC-001
Iteration: 2/3
Verdict: PASS
Overall Score: 0.93

Tier: M (`spec.md:L14` `tier: M`) — PASS threshold 0.80. Aggregate is the harmonic mean of the four rubric dimensions.
Scope (Retry Loop Contract, iteration 2): regression check over prior defects D1–D4 plus a check of every region edited for v0.1.1 (HISTORY L19–L37, §2.1 L108–L109, §2.3 L136, REQ-002.4 L192–L194, REQ-004.5 L217–L220, REQ-006.3 L238, REQ-007.1 L244–L245, Out of Scope — 경로 허용 목록 L363–L369, plan.md D-1 L34–L38, §1 L27–L29, §3.1 L118–L139, §5 L235, acceptance.md §0.1 L31–L35, AC-001 L68–L70, AC-007 rows n/o L160–L161, AC-012 L261, AC-013 L271–L273, §4 L350–L351). Must-pass criteria were re-run in full.
Inputs read (Tier M contract): `spec.md` (425 lines), `plan.md` (283), `acceptance.md` (376). Also read: previous report `SPEC-AGENTIPC-001-review-1.md`, `progress.md`, `spec-compact.md` (grep only), `research.md` (grep only, MP-7).
No author reasoning context was received; audit based on artifact files only (M1).
Audit backend: Claude-only (no MCP backend invoked; `grep -rn audit_model .moai/config/` → no match, rc=1, this run).

## Must-Pass Results

- [PASS] MP-1 REQ number consistency: `grep '^#### REQ-' spec.md` → `AIPC-001 … AIPC-015`, sequential, three-digit padding, no gaps or duplicates (spec.md:L177, L185, L196, L209, L222, L232, L240, L249, L260, L271, L283, L291, L302, L308, L322). Still 15 REQ / 15 AC (`grep -c '^### AC-' acceptance.md` → 15), within the Tier M ceiling of 16 each.
- [PASS] MP-2 GEARS format compliance — judged against the **requirement layer** (the bold canonical sentence of each REQ in spec.md §3). The v0.1.1 edits added only numbered Korean sub-items (REQ-002.4 L192, REQ-004.5 L217, REQ-006.3 L238, REQ-007.1 L245); no canonical sentence changed. Ubiquitous: REQ-001 L179, 002 L187, 007 L242, 010 L273, 012 L293, 013 L304, 014 L310, 015 L324. Event-driven: REQ-003 L198, 004 L211, 005 L224, 006 L234, 009 L262, 011 L285. Unwanted (`shall not`): REQ-008 L251. No legacy `If … then` form. The Given-When-Then entries in acceptance.md are the verification layer and were graded under Group 4 only.
- [PASS] MP-3 YAML frontmatter validity: spec.md:L2–L13 carries all 12 canonical fields; `version: "0.1.1"` (L4) is quoted semver and was correctly bumped; `updated: 2026-09-27` (L7) ISO date; `phase: "v0.x agent-ipc MVP"` is a release label, not a prohibited stage token; no rejected alias. Optional `tier: M` (L14) valid.
- [N/A] MP-4 Section 22 language neutrality: single-repository Go + Emacs Lisp project, not template-bound multi-language tooling. Auto-pass.
- [PASS] MP-5 D7 cross-SPEC reconciliation: `grep -Eoh 'SPEC-([A-Z][A-Z0-9]+-)+[0-9]+' spec.md plan.md acceptance.md | sort | uniq -c` → `4 SPEC-AGENTIPC-001` (self only). No external SPEC reference, so no BLOCKING finding.
- [PASS] MP-6 D8 cross-platform discipline: `grep -c syscall` over all six files in the SPEC directory → 0 each. D8 auto-PASS.
- [PASS] MP-7 clarification gate: `grep -rn '\[NEEDS CLARIFICATION' plan.md research.md` → no output (rc=1). plan.md §8 L275 "없음".

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.90 | 0.75–1.0 (one minor residual ambiguity) | The three iteration-1 clarity defects are fixed: status string is now single-valued everywhere (plan.md:L36 "뒤에 아무것도 붙이지 않는다"); optional-field type semantics are explicit (spec.md:L108–L109, L192–L194, L136); newline handling is defined (spec.md:L217–L220, L245). Residual: N1 below (which `*Messages*` line counts as "the notification" when a file display follows the notification). |
| Completeness | 1.00 | 1.0 band | HISTORY now has a v0.1.1 entry (spec.md:L19–L37) with the measurements behind the change. All sections present as in iteration 1; the new `### Out of Scope — 경로 허용 목록` (L363–L369) adds a seventh H3 with `-` bullets. 12/12 frontmatter fields. |
| Testability | 0.88 | 0.75–1.0 band | D2 fixed: AC-012 says "진단 정확히 한 줄(개행으로 끝나는 줄 1개)" (acceptance.md:L261). O4 fixed: the observation surface is named (acceptance.md:L31–L35). O1 fixed: AC-013 requires exactly three arguments (L271–L272). New D3/D4 cases are binary (AC-007 rows n/o L160–L161; AC-001 exact expected string L69–L70). Deductions: N1 (new), O5 (carried). |
| Traceability | 0.95 | 1.0 band (minor sub-clause gaps) | spec.md §7 L408–L424 unchanged and still bidirectional with the AC headers. REQ-002.4 → AC-007 n/o; REQ-004.5/007.1 → AC-001 And clause; AC-012 ↔ REQ-011 now match. Deduction: carried O2/O3 sub-clauses with no scenario. |

Harmonic mean: 4 / (1/0.90 + 1/1.00 + 1/0.88 + 1/0.95) = 4 / 4.300 = 0.93 ≥ 0.80 (Tier M). Score rose from 0.89 (iteration 1), so no STOP signal.

## Defects Found (structured defect-list)

No blocking defects remain. The four blocking defects from iteration 1 are resolved (see Regression Check). One new optional finding and four carried optional findings follow.

N1. AIPC-R2-N1 — acceptance.md:L32–L34 together with spec.md:L216, L234 and acceptance.md:L103–L104, L120 — The v0.1.1 O4 fix defines "에코 영역 알림이 X 이다" as "the **last** line of `*Messages*` right after the receive call is X". AC-004 and AC-005(c) combine a notification with a file display, and neither the SPEC nor plan.md fixes the order of the two steps (REQ-006's sentence order puts the notification first and the display last). If the display comes after `message`, anything that writes to `*Messages*` while the file is opened (a major-mode or `find-file-hook` function in the fully booted config that `ert_sel` loads) becomes the last line, and a conforming implementation fails AC-004/AC-005(c). This is a hypothesis: whether the booted config writes such a message when it opens a `.md` file was not executed in this audit. — Severity: minor — Class: optional — Suggested fix: pick one and write it down, either (a) in plan.md §3.1, "the notification is emitted after any file display", or (b) in acceptance.md §0.1, define the observation as "a line appended to `*Messages*` during the receive call is X" instead of "the last line".

Carried optional findings from iteration 1 (not addressed in v0.1.1; still optional, still not required for the verdict):
- O2 — spec.md:L227 — REQ-005.2 second sentence (file shown only in the selected window → shown once more elsewhere; user point unchanged) has no AC scenario.
- O3 — spec.md:L251, L247 — REQ-008 "write to … any payload file" and REQ-007.3 (log buffer recreated after being killed) have no AC check.
- O5 — acceptance.md:L260 — "1 MiB 를 넘는 TEXT" is only reachable through in-process `run(...)`, because the OS refuses a single argument that size at exec time. Not marked as such.
- O7 — implementation-level identifiers in some REQs (`server-eval-args-left`, `point-max`, app-bundle paths); justified as the external contract, no change needed.

## Regression Check (Iteration 2+ only)

Defects from previous iteration:

- D1 (plan.md D-1 allowed `"error:<reason>: <detail>"`, contradicting §2.3 / REQ-001.2 / REQ-014.2 / AC exact strings) — **RESOLVED**. plan.md:L36 now reads "정확히 `"ok"` 또는 `"error:<reason>"` … 뒤에 아무것도 붙이지 않는다", and L37 moves human-readable detail to the log line only. `grep -n 'error:<reason>:\|<detail>'` across the SPEC directory matches only the HISTORY note that records the deletion (spec.md:L23). plan.md §3.2 L165 ("reason 은 `error:` 뒤 전체") is now consistent with it.
- D2 (AC-012 accepted "한 줄 이상" while REQ-011 requires one line) — **RESOLVED**. acceptance.md:L261 "stderr 에 진단 정확히 한 줄(개행으로 끝나는 줄 1개)" matches REQ-011 spec.md:L285 "write one diagnostic line to stderr".
- D3 (`[]`/`{}` in optional fields collapsed to "absent" under the planned parse options) — **RESOLVED**. The SPEC chose option (b) plus an explicit statement: spec.md:L108–L109 and REQ-002.4 L192–L194 make `[]`, `{}`, numbers and `true`/`false` in optional fields `bad-field`; §2.3 L136 lists them; plan.md:L118–L127 switches to `:array-type 'array :null-object :null :false-object :false` and uses `assq` for key presence; AC-007 rows n (L160) and o (L161) test envelope and payload optional fields. Verified in this audit with Emacs 30.2 `--batch -Q`: `(json-parse-string "{\"project\":[],\"session\":{},\"a\":null,\"b\":false,\"c\":true,\"payload\":{},\"n\":3,\"f\":2.5}" :object-type 'alist :array-type 'array :null-object :null :false-object :false)` → `((project . []) (session) (a . :null) (b . :false) (c . t) (payload) (n . 3) (f . 2.5))`. So `[]` (vector), `{}` (`nil`), `null` (`:null`), `false` (`:false`) and an absent key (no `assq` hit) are all distinguishable, and `(stringp nil)` → `nil`, `(natnump 2.5)` → `nil` confirm the planned predicates reject them. Duplicate keys are preserved (`{"k":1,"k":2}` → `((k . 1) (k . 2))`), so D-11 still works under the new options.
- D4 (multi-line `text`/`summary` broke the "one line" promises of REQ-004/REQ-007) — **RESOLVED**. REQ-004.5 (spec.md:L217–L220) replaces LF, CR and CRLF with a single `⏎` in every event-derived value in notification and log lines; REQ-006.3 (L238) and REQ-007.1 (L245) point to it; AC-001 And clause (acceptance.md:L68–L70) gives the exact expected string and checks log line count +1 with `p⏎q`. Verified: the regex in plan.md:L139 applied to the AC-001 input, `(replace-regexp-in-string "\r\n\\|[\n\r]" "⏎" "첫 줄\n둘째 줄\r\n셋째 줄\r넷째 줄")`, → `"첫 줄⏎둘째 줄⏎셋째 줄⏎넷째 줄"`, which matches AC-001 L69 exactly.

Optional findings from iteration 1: O1 resolved (acceptance.md:L271–L272), O4 resolved (acceptance.md:L31–L35; it introduced N1 above), O6 resolved (spec.md:L363–L369). O2, O3, O5, O7 carried unchanged (optional).

No stagnation: every blocking defect changed between iterations.

## Verification performed in this audit

Commands run from the repository root, this tree, HEAD `fd7f0b0`:
- `grep '^#### REQ-'` / `grep -c '^### AC-'` → 15 / 15.
- D7 SPEC-ID extraction → self only. D8 `grep -c syscall` → 0 in all files. MP-7 grep → no output.
- Residual greps for `error:<reason>:`, `<detail>`, `timeout=N` → only HISTORY notes and research.md background (research.md:L100 describes the emacsclient flag, not an AC allowance).
- Weasel-word grep (`appropriate|adequate|reasonable|적절|충분히`) over spec.md and acceptance.md → no match.
- Emacs 30.2 batch probes for D3 (parse options, duplicate keys, `natnump`/`stringp`/`listp`/`vectorp`) and D4 (newline regex), outputs quoted above. An extra probe confirmed the trailing-content check in plan.md:L129: after `json-parse-buffer` on `{"a":1} {}` and skipping whitespace, `(eobp)` → `nil`.
- `spec-compact.md` shows `version: 0.1.1` and the new parse options (L6, L10–L11), so it is in step with spec.md.

Not verified: whether the booted configuration writes to `*Messages*` when it opens a file (N1 hypothesis); the HISTORY emacsclient measurements, the `ert_sel` selector result and the D-5 window-split measurement (author-reported, relied on as stated, same as iteration 1).

## Recommendation

PASS. Must-pass evidence: MP-1 REQ-AIPC-001..015 sequential (spec.md:L177–L322); MP-2 all 15 canonical sentences in GEARS form, unchanged by v0.1.1; MP-3 12/12 canonical fields with version bumped to `"0.1.1"` (spec.md:L2–L13); MP-4 N/A; MP-5 self-reference only; MP-6 no `syscall`; MP-7 no clarification markers. Aggregate 0.93 ≥ Tier M threshold 0.80, up from 0.89.

All four iteration-1 blocking defects (D1–D4) are resolved, and the D3/D4 fixes were checked against Emacs 30.2 behavior. Nothing blocks Implementation Kickoff Approval. N1 is a cheap wording fix and worth applying before run: otherwise the M2 implementer may find AC-004/AC-005(c) failing for a reason unrelated to the receiver. O2, O3 and O5 remain at the orchestrator's discretion.
