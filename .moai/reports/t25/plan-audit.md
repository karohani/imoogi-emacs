# SPEC Review Report: SPEC-ANKISRC-001 (card t25)
Iteration: 3/3
Verdict: PASS
Overall Score: 0.92 (Tier M threshold 0.80, from spec-workflow.md § SPEC Complexity Tier)

Auditor: plan-auditor, Claude only (`audit_model` is not set). Reasoning context ignored per M1 Context Isolation. As instructed, this round is limited to confirming round-2 R1–R3 and regression-checking the text changed in v0.1.2, including the O1–O3 rulings. D1–D8 and the earlier lane rulings are taken as fixed.

No blocking findings remain. Two optional notes are below.

## Round 1 (summary)
FAIL, 0.75. All seven must-pass checks passed. The 11 blocking findings were:
- F1: absolute-path requirement vs Org's abbreviated paths
- F2: the charset claim was false for `org-id-include-domain`
- F3: unnamed diagnostic code
- F4: eval-failure path could end silently; Emacs not raised for the no-match / 2+-match outcomes
- F5: ambiguous link target for stock note types
- F6: no failure containment for the ID pass
- F7: no per-call timeout
- F8: AC-012 depended on cache state it didn't fix
- F9: AC-004 didn't assert that entries are still sent
- F10: REQ-001.5 had no AC
- F11: 18 ACs over the Tier M ceiling of 16

## Round 2 (summary)
FAIL, 0.86. F1–F11 were all resolved in v0.1.1. Three new contradictions in the changed text were blocking:
- R1: DD-6 and DD-7 disagreed on which receiver results are normal, so AC-015(d) could not be graded.
- R2: REQ-001.6 stripped `org_id` from every heading in a failing file, contradicting REQ-003.
- R3: DD-9 saved the ID cache before REQ-005's registrations happened.

O1–O3 were optional.

## Must-Pass Results (re-checked on v0.1.2)
- [PASS] MP-1: 13 REQs, REQ-ASRC-001..013 in sequence.
- [PASS] MP-2 (judged on the REQ layer only): every REQ is When/Ubiquitous; the changed REQ-001.6, 005, 010.2 and 011.2 keep their parent's form.
- [PASS] MP-3: 12 fields present; `version: "0.1.2"` (spec.md:L4).
- [N/A] MP-4: not multi-language tooling.
- [PASS] MP-5: no new SPEC references.
- [PASS] MP-6: `syscall` appears 0 times.
- [PASS] MP-7: no `[NEEDS CLARIFICATION]` markers.

## Category Scores
| Dimension | Score | Evidence |
|---|---|---|
| Clarity | 0.75 | One optional imprecision remains, in the R-7 mitigation regex (O4). The requirements themselves read unambiguously. |
| Completeness | 1.0 | Normal receiver results are listed (spec.md:L155). The cache save is now at the end of sync (L115–L116, plan DD-10 L92–L94). Failure-path buffer handling is specified (plan L88–L90). |
| Testability | 1.0 | AC-015(d) now has a concrete non-normal value (`"weird"`), and (f) pins `"not-found"`/`"duplicate"` to 0 notifications and exit 0 (acceptance.md:L85). AC-009(b) covers a cache file outside the targets (L53). |
| Traceability | 1.0 | 16 ACs (ceiling 16) and 13 REQs. REQ-010.2 → AC-015(d)(f); REQ-011.2 (O3) → AC-009(b). |

## Regression Check — round-2 defects
- R1 RESOLVED:
  - REQ-010.2 lists the three normal results `"ok"`/`"not-found"`/`"duplicate"`, each ending with exit 0 and no notification. Anything else, or an eval error, gives a notification and a non-zero exit (spec.md:L155).
  - DD-6 (plan.md:L61) and DD-7 (L70–L71) now agree.
  - DD-6 L62 names a separate classifier and explains why agentipc `classify` is not reused.
  - AC-015 (d) and (f) cover both sides (acceptance.md:L85).
- R2 RESOLVED: REQ-001.6 limits the link-free path to headings that could not get an ID. Headings that already have an `:ID:` on disk still carry it per REQ-003 (spec.md:L96).
- R3 RESOLVED: REQ-005 saves once at the end of `imoogi-sync`, after every sent ID is registered (spec.md:L115–L116). The ID pass only registers (plan DD-9 L90). New DD-10 (L92–L94) registers all sent IDs, pre-existing ones included, before the single save.

## Regression check — other changed text (O1–O3 rulings)
- O1 (plan R-7 L144): the 10 s eval budget is recorded as a risk with a mitigation and a measurement duty in M3. Consistent with REQ-010.3.
- O2 (plan DD-9 L88–L90): on the failure path, the ID pass clears the modified flag on a buffer it opened before closing it, and never closes a buffer the user had open. Consistent with REQ-001.3 and 001.6 and with AC-003. The HISTORY wording "닫는 버퍼는 … 수정되지 않은 버퍼뿐" (spec.md:L29) also holds, since the buffer is unmodified by the time it is closed.
- O3 (spec.md:L161, AC-009(b)): a cached file outside the targets is accepted only if it exists and has exactly one heading-local match. Uniqueness is checked over the targets plus that file. Consistent with REQ-011.3, which falls back to the full scan otherwise.
- The traceability table (spec.md:L210–L211) and the counts (13 REQs, 16 ACs) are unchanged and still correct.

## Defects Found
No blocking defects.

Optional (new, not verdict-bearing):
- O4. plan.md:L144 (R-7 mitigation). The regex `:ID:[ \t]+<ID>$` misses a property line with trailing whitespace. Org's property reader tolerates trailing whitespace, so a duplicate written that way would be silently missed and the ID treated as unique. — Severity: minor — Class: optional — Fix: end the pattern with `[ \t]*$`.
- O5. plan.md:L88–L90. On the failure path, `org-id-get` has already registered the new ID in memory with `org-id-add-location` before the buffer change is discarded. The DD-10 save at the end of sync then persists a cache entry for an ID that is not on disk. The open side handles it correctly: the cache entry fails verification, the scan finds nothing, and the result is `"not-found"`. — Severity: minor — Class: optional — Fix (if wanted): drop the in-memory entry on the failure path with `remhash` on `org-id-locations`.

## Recommendation
PASS. The three round-2 contradictions are resolved, with evidence at spec.md:L96, L115–L116 and L155, plan.md:L61–L62, L70–L71 and L88–L94, and acceptance.md:L53 and L85. The changed text introduces no new contradiction. Every must-pass check passes and the score of 0.92 clears the Tier M threshold of 0.80. O4–O5 are left to the lane; neither blocks run-phase entry.
