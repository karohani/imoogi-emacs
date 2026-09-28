# SPEC Review Report: SPEC-ANKICARD-001

Iteration: 1/3
Verdict: PASS
Overall Score: 0.96
Tier: L (PASS threshold 0.85, per `.claude/rules/moai/workflow/spec-workflow.md` § SPEC Complexity Tier)

> **Reasoning context ignored per M1 Context Isolation.** This audit was performed against the
> artifact files alone. Per the Tier L input contract, all five artifacts were read:
> `spec.md` (759 L), `plan.md` (978 L), `acceptance.md` (428 L), `design.md` (598 L),
> `research.md` (366 L).

---

## Must-Pass Results

- **[PASS] MP-1 REQ number consistency.** `grep -oE 'REQ-C-[0-9]{3}' spec.md | sort -u` returns
  exactly 24 identifiers, `REQ-C-001` .. `REQ-C-024`, contiguous, no gap, no duplicate, uniform
  3-digit zero-padding. Heading order is monotonic: spec.md:197 (`REQ-C-001`) through spec.md:451
  (`REQ-C-024`), one `#### REQ-C-0NN` heading per identifier, 24 headings total. Within the
  Tier L requirement ceiling of 25.

- **[PASS] MP-2 GEARS format compliance — judged against the REQUIREMENT LAYER ONLY.** The
  judgment was made against the 24 `REQ-C-XXX` entries in `spec.md` § 3 (spec.md:197-451).
  `acceptance.md`'s Given-When-Then entries are the verification layer and were graded under
  Group 4 (AC-1), never here. Independent pattern census over the 24 requirement headings:

  | GEARS pattern | Count | Requirements |
  |---|---|---|
  | Ubiquitous (`The <subject> shall …`) | 7 | C-001, C-005, C-006, C-007, C-016, C-019, C-023 |
  | Event-driven (`When …, the <subject> shall …`) | 5 | C-002, C-010, C-012, C-018, C-020 |
  | State-driven (`While …, the <subject> shall …`) | 2 | C-003 (While + Unwanted), C-014 |
  | Capability-gate (`Where …`, compound with `When`) | 2 | C-008, C-021 |
  | Unwanted (`The <subject> shall not …`) | 6 | C-004, C-009, C-011, C-013, C-017, C-024 |
  | Event-detected (`When … is detected, … shall …`) | 2 | C-015, C-022 |

  7+5+2+2+6+2 = 24. This census was computed independently from the headings and matches the
  SPEC's own HISTORY claim at spec.md:34-36 exactly. Compound forms `[Where …][While …][When …]`
  (C-003, C-008, C-021) are PASS-equivalent at rubric score 1.0. Zero legacy `If/then` forms
  requiring the `[DEPRECATED — use shall not]` marker. Verified absence of informal modality in
  the requirement layer: `sed -n '182,470p' spec.md | grep -E '\b(should|may|might|could)\b'`
  returns **no match** — every obligation carries `shall` or `shall not`.

  *Judgment stated explicitly:* several requirements (C-009, C-017, C-018, and notably C-020)
  carry explanatory appositives or a trailing rationale sentence alongside the normative clause.
  This is **not** the "mixed informal/formal within a single requirement" failure mode: the
  modality of every obligation in each of those entries remains `shall`/`shall not`, and the
  rationale is separable prose. Recorded as D4 (optional), not an MP-2 failure.

- **[PASS] MP-3 YAML frontmatter validity.** Verified field-by-field against
  `.claude/rules/moai/development/spec-frontmatter-schema.md` § Canonical 12 Required Fields.
  All 12 present with correct types (spec.md:1-16):

  | Field | Value | Verdict |
  |---|---|---|
  | `id` | `SPEC-ANKICARD-001` | matches `^SPEC-[A-Z][A-Z0-9]+-[0-9]{3}$` |
  | `title` | quoted, non-empty | OK |
  | `version` | `"0.1.1"` quoted semver | OK |
  | `status` | `draft` | in the 8-value status enum (schema:63-68) |
  | `created` | `2026-09-05` | ISO `YYYY-MM-DD` |
  | `updated` | `2026-09-05` | ISO `YYYY-MM-DD` |
  | `author` | `jay` | non-empty |
  | `priority` | `P2` | in `P0`\|`P1`\|`P2`\|`P3` |
  | `phase` | `"v0.2.0 target"` | a release target, **not** a prohibited lifecycle-stage value |
  | `module` | `"internal/anki, cmd/imoogi-anki, modules/anki, modules/24-anki.el"` | path-like, non-empty |
  | `lifecycle` | `spec-anchored` | in enum |
  | `tags` | comma-separated, non-empty | OK |

  **Zero rejected snake_case aliases** — no `created_at`, `updated_at`, `labels`, or `spec_id`.
  Two optional fields are carried and both appear in the schema's Optional Fields table
  (schema:151-152): `tier: L` and `amendment_of: SPEC-ANKI-001`. `amendment_of` is correctly
  paired with the required `### Amendments` sub-section (spec.md:89-104) recording prior
  completed version, `prior_completed_sha`, rationale, and scope. The `related_specs` field
  removed at v0.1.1 was correctly removed — it is not in the schema's optional table.

- **[N/A] MP-4 Section 22 language neutrality.** N/A: single-project SPEC. The scope is one
  brownfield codebase — a Go binary (`internal/anki`, `cmd/imoogi-anki`) plus an Emacs Lisp front
  end (`modules/anki/*.el`), per the `module:` frontmatter and § 7 Brownfield Delta
  (spec.md:646-733). The SPEC targets neither template-bound nor universal multi-language
  content, so the 16-language equal-enumeration criterion does not apply and auto-passes. No
  language-specific tool is named as a cross-language "default"; the toolchain constraint
  (spec.md:642-644, "Go 1.26 with go-org v1.9.1") is scoped to this project's own stack.

- **[PASS] MP-5 D7 cross-SPEC reconciliation.** Verification verb executed. Extracted references:
  `SPEC-ANKI-001`, `SPEC-ANKICARD-001` (self), `SPEC-TRANSIENT-001`.
  - `SPEC-TRANSIENT-001` — exists at `.moai/specs/SPEC-TRANSIENT-001/spec.md`, `status: draft`.
    Not in {retired, superseded, archived}. No reconciliation required. spec.md:743 additionally
    declares "**No dependency** on `SPEC-TRANSIENT-001` — disjoint module set."
  - `SPEC-ANKI-001` — **not present in `.moai/specs/`** → D7-5 SHOULD finding emitted (see D1).
    Not BLOCKING. The SPEC declares it out-of-repo with an absolute path (spec.md:67-68), and
    that path was verified to resolve: `/Users/jay/workspace/imoogi-org-anki/.moai/specs/SPEC-ANKI-001/spec.md`
    exists with `status: in-progress` — **not** retired/superseded/archived, so D7-4 does not fire.
    The SPEC nonetheless supplies full explicit reconciliation in § 4 (spec.md:470-547): a verbatim
    REQ-016 amendment, an AC-016 replacement pair, a two-exclusion supersession, and a D-12
    assumption.

  **No D7 BLOCKING finding.** MP-5 PASS.

- **[PASS] MP-6 D8 cross-platform discipline.** D8 verb executed: `grep -c 'syscall' spec.md`
  returns **0**. Per D8-4, absence of the literal substring `syscall` in the SPEC body is
  auto-PASS — no cross-platform build-tag concern arises. **No D8 BLOCKING finding.**

- **[PASS] MP-7 clarification gate.** Verification verb executed and widened beyond the minimum:
  `grep -rn '\[NEEDS CLARIFICATION' plan.md research.md spec.md acceptance.md design.md` returns
  **zero matches** across all five Tier L artifacts. No unresolved clarification marker exists.
  Both `plan.md` and `research.md` are present, so the N/A branch does not apply.

---

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|---|---|---|---|
| Clarity | 0.95 | 1.0 (one deduction) | Requirements are table-precise: REQ-C-010.1 fixes all four delimiter mappings in a 4-row table (spec.md:302-309); REQ-C-006.2 states the deck-class normalization as a 6-step total function (spec.md:263-267); REQ-C-012.1 enumerates the 14 accepted image extensions verbatim (spec.md:331-334). § 2 Glossary (spec.md:146-180) fixes *imoogi-owned note type*, *ordinary synchronization run*, *install step*, *migrate subcommand*, *deck class normalization*, and *foreign model* once, and names the requirements that consume each. No pronoun-reference ambiguity found. **Deduction:** REQ-C-007.2 (spec.md:275-282) uses non-measurable aesthetic vocabulary — "restrained editorial treatment", "generous line height", "raised contrast", "warm paper tone", "near-black ground", "reduced-contrast body text". Resolved only at the artifact-set level by `design.md` § 4.3's 8 ordered blocks (design.md:252-277), not by the requirement text itself. See D6. |
| Completeness | 1.0 | 1.0 | HISTORY spec.md:18; WHY/Purpose § 1.1 spec.md:108; WHAT/Scope § 1.2 spec.md:118; REQUIREMENTS § 3 spec.md:182 (24 entries); ACCEPTANCE CRITERIA `acceptance.md` § AC Matrix (24 top-level); Out of Scope § 5 spec.md:549. **SC-6 satisfied concretely:** eight `### Out of Scope — <topic>` H3 sub-headings (spec.md:560, 570, 579, 585, 594, 603, 609, 614), each carrying at least one specific `-` bullet naming a concrete excluded behavior (e.g. spec.md:571-573 "Uploading, rewriting, or diagnosing a link whose target is a video or audio file…"). Additional sections beyond the minimum: § 2 Glossary, § 4 Parent-SPEC Amendment, § 6 Constraints, § 7 Brownfield Delta, § 8 Dependencies, § 9 Traceability. All 12 frontmatter fields present (MP-3). All 5 Tier L artifacts present and substantive. |
| Testability | 0.90 | 1.0 (one deduction) | **Zero weasel words across the whole of `acceptance.md`**: `grep -nEi '\b(appropriate\|adequate\|reasonable\|proper\|good\|as needed\|if necessary)\b' acceptance.md` returns no match. ACs assert countable or byte-level facts: AC-C-002 "**exactly zero** `createModel` requests" and "byte-identical to the first"; AC-C-003a "each **exactly 0**"; AC-C-016a "**byte-equal** to the field map whose hash the registry records"; AC-C-005 "**zero** occurrences of `http`, `@import`, `url(`". AC-C-004 fixes deck-class normalization as an input→output table. AC-C-022c states its manual smoke check with exact measured counts (88 stock-`Basic`, 27 stock-`Cloze`). **Deduction:** AC-C-005 (acceptance.md:73-83) verifies REQ-C-007.2 only partially — it asserts the presence of a serif heading stack, a system sans body stack, a bounded measure, and distinct light/night colour blocks, but asserts **nothing** for "generous line height", "warm paper tone", "near-black ground", "reduced-contrast body text", or "raised contrast for math and code". Those normative clauses carry no binary test. See D6. |
| Traceability | 1.0 | 1.0 | Bidirectional check executed with `comm` over sorted unique identifier sets. **REQs in `spec.md` with no AC coverage: none** (empty result) — all 24 requirements are named by at least one AC heading. **REQs referenced in `acceptance.md` that do not exist in `spec.md`: none** (empty result) — no orphaned AC. Every AC heading carries an explicit `(REQ-C-0NN, …)` pointer, e.g. `AC-C-016 — Transform-before-hash, the load-bearing guard (REQ-C-016, REQ-C-017)`. Stale-identifier sweep: the pre-consolidation identifiers `REQ-C-025` .. `REQ-C-042` appear **21 times, all confined to `plan.md` lines 23-66** — the § Revision 3 old→new mapping table, which is a legitimate historical record. `spec.md`, `design.md`, `acceptance.md`, and `research.md` are all clean of them (0 occurrences each). § 9 Traceability (spec.md:747-759) additionally routes each artifact class to its owning file. |

**Aggregate (harmonic mean, per the Skeptical Evaluation Stance):**
4 / (1/0.95 + 1/1.0 + 1/0.90 + 1/1.0) = 4 / 4.1637 = **0.96**.

0.96 ≥ 0.85 (Tier L threshold). The verdict is not score-sensitive: even dropping Testability to
the 0.75 band yields a harmonic mean of 0.91, still a PASS.

---

## Defects Found (structured defect-list)

D1. **D7-5-ANKI-001** — `spec.md`:L67-68 — Referenced SPEC `SPEC-ANKI-001` is not present in
`.moai/specs/`, so the D7 verb emits its SHOULD-severity "referenced SPEC not found" finding.
The finding is surfaced rather than absorbed, per M2. It is **already resolved in place**: the
SPEC declares the parent out-of-repo with its absolute path, that path was verified to resolve to
a real `spec.md`, its `status:` was read as `in-progress` (not retired/superseded/archived), and
§ 4 supplies explicit reconciliation for every clause the amendment touches. No BLOCKING
condition; MP-5 stands PASS. — Severity: minor — Class: optional — Required fix: none required.
If the parent is ever vendored into this repo's `.moai/specs/`, drop the out-of-repo path prose.

D2. **PARENT-STATUS-INCONSISTENCY** — `research.md`:L24 vs `spec.md`:L93-96 — The same fact about
the same external artifact is stated two different ways inside one Tier L artifact set.
`research.md`:L24 asserts the parent "is complete and CLOSED"; `spec.md`:L93-96 correctly and
carefully states that "its frontmatter still reads `status: in-progress`… Treated as behaviorally
closed and specification-live. Recorded, not acted on." I verified the parent's frontmatter
directly: `status: in-progress`. `spec.md` is the authoritative and accurate statement; the
`research.md` phrasing overstates. — Severity: minor — Class: optional — Required fix: soften
`research.md`:L24 to match spec.md's formulation ("recorded as in-progress, behaviorally closed
at 25/25 AC"), so the two artifacts do not disagree about a verifiable fact.

D3. **SUBCLAUSE-TRIGGER-CLAIM** — `spec.md`:L191-193 vs L410-422 — spec.md:191-193 states as a
general rule that "Sub-clauses inside a requirement are numbered. They share that requirement's
single GEARS trigger and pattern." That meta-claim does not hold for REQ-C-021: sub-clause 1
triggers on "**When** an entry's registry-recorded note type is a stock `Basic` or `Cloze`…" and
sub-clause 2 on "**When** a migration completes for an entry…" — two distinct `When` triggers
nested under one `Where`. The requirement itself remains valid GEARS (compound `Where + When` is
PASS-equivalent), so MP-2 is unaffected; only the SPEC's self-description is inaccurate for this
one entry. — Severity: minor — Class: optional — Required fix: either soften L191-193 to "share
that requirement's governing `Where`/`While` scope", or split REQ-C-021 into two requirements
(budget permits: 24 of 25 used).

D4. **RATIONALE-IN-REQUIREMENT-TEXT** — `spec.md`:L400-409 (REQ-C-020), also L290-294 (C-009),
L378-382 (C-017), L384-390 (C-018) — Normative requirement bodies carry embedded explanatory
rationale. REQ-C-020 is the clearest case: its second sentence, "The dry-run request that produces
the count is not a writing request — it reads the registry and issues no AnkiConnect write of any
kind — so obtaining the count does not presuppose the answer", is a standalone non-`shall`
sentence inside a requirement. This does **not** flip MP-2 (every obligation in these entries
still carries `shall`/`shall not`; the rationale is separable), and the reasoning is genuinely
load-bearing for understanding the confirmation gate. Reported for the record rather than as a
correctness problem. — Severity: minor — Class: optional — Required fix: none required; if
tightened, relocate the appositives to `design.md` § 7 and leave the `shall` clause alone.

D5. **AC-IDENTIFIER-DRIFT** — `plan.md`:L217, L221, L443, L533, L785, L810, L826 and
`acceptance.md`:L10-12 — Two identifier-scheme inconsistencies in the plan-phase artifact set,
neither reaching `spec.md` or `acceptance.md`'s normative content:
(a) `plan.md` uses the outline-grade labels `AC-C-a` / `AC-C-b` in seven places for the
    AC-016 replacement pair. No such identifier exists in `acceptance.md`, which defines them as
    `AC-C-003a` / `AC-C-003b`. `spec.md` § 4.3 (L509, L516) carries the correct mapping
    ("Realized as `AC-C-003a`" / "`AC-C-003b`"), so the pointer chain is recoverable, but a reader
    working from `plan.md` § F milestones M1/M2 alone cannot resolve the label.
(b) `acceptance.md`:L10-12 self-describes its sub-criteria as "(`AC-C-0NNa` / `AC-C-0NNb`)", but
    the file also carries `AC-C-006c`, `AC-C-016c`, `AC-C-022c`, and `AC-C-022d`.
— Severity: minor — Class: optional — Required fix: (a) re-point the seven `AC-C-a` / `AC-C-b`
occurrences in `plan.md` to `AC-C-003a` / `AC-C-003b`; (b) amend `acceptance.md`:L10-12 to say
sub-criteria are lettered `a`..`d`.

D6. **AESTHETIC-CLAUSE-UNVERIFIED** — `spec.md`:L275-282 (REQ-C-007.2) and `acceptance.md`:L73-83
(AC-C-005) — REQ-C-007.2 is normative ("the imoogi base stylesheet **shall** present…") but its
operative terms are aesthetic and unbounded: "restrained editorial treatment", "generous line
height", "raised contrast for math and code", "warm paper tone", "near-black ground",
"reduced-contrast body text". AC-C-005 verifies only the mechanically checkable subset — the two
night-mode selector forms, the `.card img` rule, `word-wrap: break-word`, presence of a serif
heading stack and a system sans body stack, presence of distinct light/night colour blocks, and
zero network-fetch constructs. The tone, contrast, and line-height clauses have **no** binary test
in `acceptance.md`. This is the sole Testability deduction. **Classified optional deliberately**:
`design.md` § 4.3 (L252-277) fixes the visual direction as 8 ordered blocks and `plan.md` D-C-4
records the decision, so implementation is not left undetermined; and demanding numeric contrast
ratios or a line-height figure for a stylesheet SPEC would be the speculative over-specification
M6 and the Enforce Simplicity behavior forbid. — Severity: minor — Class: optional — Required fix:
if tightened, add one AC-C-005 assertion binding the two most consequential claims to a checkable
form (e.g. "the night block and the light block each define `--fg`/`--bg` custom properties, and
the math/code rule's colour differs from the body colour in both blocks"), rather than adding
numeric aesthetic thresholds.

D7. **AC-CEILING-INTERPRETATION** (recorded, not a defect against the SPEC) —
`acceptance.md`:L10-12 — The SPEC carries 24 **top-level** acceptance criteria, within the Tier L
ceiling of 25, but those expand to roughly 54 sub-lettered assertions. `acceptance.md`:L10-12
declares sub-criteria to be "sub-assertions within one logical criterion" rather than independent
criteria. **This audit accepted that interpretation**, and states so explicitly so a later
auditor reading the 25-ceiling literally does not have to re-derive the judgment. Rationale: the
ceiling's own stated purpose in `spec-workflow.md`:L152 is auditor-holdability — "it lands
hardest on the plan-auditor, which must hold every requirement and criterion in view at once" —
and 24 logical criteria, each grouping tightly related assertions about one requirement cluster,
is holdable in a way 54 flat criteria would not be. No fix required. — Severity: minor — Class:
optional — Required fix: none.

**No blocking-class findings. No critical-severity findings. No unresolved must-pass failures.**

---

## Regression Check

Not applicable — iteration 1. No prior report exists at
`.moai/reports/plan-audit/SPEC-ANKICARD-001-review-*.md`.

---

## Recommendation

**PASS.** The must-pass firewall is clear on every one of the seven criteria, and each verdict is
grounded in a command that was actually run against the artifact files:

1. **MP-1** — the identifier set is contiguous `REQ-C-001` .. `REQ-C-024` with 24 matching
   `#### REQ-C-0NN` headings in monotonic order (spec.md:197-451), no gap, no duplicate.
2. **MP-2** — judged against the requirement layer only. An independently computed pattern census
   (7 Ubiquitous / 5 When / 2 While / 2 Where / 6 Unwanted / 2 event-detected = 24) confirms every
   requirement matches a GEARS pattern, and a modality sweep over spec.md:182-470 finds zero
   informal `should`/`may`/`might`/`could`. Given-When-Then entries in `acceptance.md` are the
   correct verification-layer format and were graded under Group 4, never penalized here.
3. **MP-3** — all 12 canonical fields present and correctly typed against the schema SSOT; zero
   rejected snake_case aliases; the two optional fields carried (`tier`, `amendment_of`) are both
   in the schema's optional table, and `amendment_of` is correctly paired with an `### Amendments`
   sub-section.
4. **MP-4** — N/A, single-project Go + Elisp scope, auto-passes.
5. **MP-5** — D7 verb executed over all three extracted SPEC references; one SHOULD finding
   emitted and surfaced as D1; no referenced SPEC carries a retired/superseded/archived status;
   no BLOCKING condition.
6. **MP-6** — D8 verb executed; `syscall` count is 0; auto-PASS per D8-4.
7. **MP-7** — the clarification-gate grep returns zero matches across all five artifacts.

Three properties of this SPEC deserve to be named as the reason the score sits where it does, each
with its own citation. **The blocking conflict is confronted rather than elided**: § 4
(spec.md:470-547) states the parent-SPEC prohibition that forbids this feature, quotes the amended
REQ-016 text verbatim, replaces the parent's per-run blanket AC-016 with an ownership-scoped pair,
and bounds the supersession to imoogi-authored types only — a SPEC that had quietly proceeded past
that conflict would have failed CN-1. **The load-bearing invariant is bound to a mechanical
assertion rather than to prose**: REQ-C-016's transform-before-hash ordering is verified by
AC-C-016a's byte-equality and hash-recomputation assertion (acceptance.md:216-227), and both
`acceptance.md` and `plan.md` § F M4 state explicitly that "prose sequencing is not the guard;
this test is." **Traceability is complete in both directions** — the `comm` check returns an empty
set on each side, and the 21 pre-consolidation identifiers that survive anywhere are confined to
`plan.md`'s § Revision 3 mapping table, where they belong.

The seven findings above are all minor-severity and all classified optional. Per M6, they are
surfaced for the orchestrator's discretion and are **not** a fix-then-re-audit route: routing them
into a revision would produce wording churn, and in D6's case would risk adding acceptance criteria
for aesthetic properties the SPEC never claimed to bound numerically. If any subset is addressed,
D5(a) — the seven `AC-C-a` / `AC-C-b` occurrences in `plan.md` — is the highest-value single fix,
because it is the one finding where a reader working from a single artifact cannot resolve an
identifier without consulting another.

**This SPEC is approved to proceed to the Implementation Kickoff Approval human gate.** That gate
remains mandatory and is not affected by this verdict.
