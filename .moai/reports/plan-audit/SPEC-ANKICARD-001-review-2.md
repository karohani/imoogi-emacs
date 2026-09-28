# SPEC Review Report: SPEC-ANKICARD-001

Iteration: 2/3
Verdict: PASS
Overall Score: 0.98
Tier: L (PASS threshold 0.85, per `.claude/rules/moai/workflow/spec-workflow.md` § SPEC Complexity Tier, row L)

> **Reasoning context ignored per M1 Context Isolation.** This audit was performed against the
> artifact files alone. Per the Tier L input contract all five artifacts were read at v0.1.2:
> `spec.md` (963 L), `plan.md` (1047 L), `acceptance.md` (511 L), `design.md` (650 L),
> `research.md` (366 L).
>
> **Audit backend**: `grep -rn 'audit_model' .moai/config/` returns no match, so no `audit_model`
> key is configured. The cross-model MCP backends (`audit_multi` / `codex_audit` / `glm_audit`)
> were therefore not called — Claude-only audit, consistent with iteration 1.

## Scope of this iteration

Per the Retry Loop Contract, iteration 2+ is scoped to the enumerated defect delta plus a
regression check. That scope was **widened deliberately** for one reason stated up front: v0.1.2
did not merely patch the iteration-1 defects, it **re-partitioned the requirement layer** (former
`REQ-C-018` folded into `REQ-C-016.2`; `REQ-C-019`..`REQ-C-022` renumbered; former `REQ-C-022`
split into two). A renumbering invalidates iteration 1's mechanical evidence — every line citation
in review-1 for MP-1, MP-2, and Traceability is stale — and it opens a failure class iteration 1
never had to look for: an AC or a plan pointer that carries a re-pointed identifier while still
meaning the old one. So: all seven must-pass verbs were **re-executed** against the new structure
(not carried over), the D1-D7 regression check was run, and a re-partition-specific sweep was added
(§ Re-partition Risk Sweep). Sections HISTORY records as untouched (§ 4.2, § 5, § 6) were
spot-checked rather than re-audited from scratch.

---

## Must-Pass Results

- **[PASS] MP-1 REQ number consistency.** Re-executed against v0.1.2.
  `grep -oE 'REQ-C-[0-9]{3}' spec.md | sort -u` returns exactly 24 identifiers,
  `REQ-C-001`..`REQ-C-024`, contiguous, no gap, no duplicate, uniform 3-digit zero-padding.
  `grep -cE '^#{3,4} REQ-C-[0-9]{3}' spec.md` returns **24**, one heading per identifier, in
  monotonic order from spec.md:292 (`#### REQ-C-001 [Ubiquitous]`) to spec.md:589
  (`#### REQ-C-024 [Ubiquitous — negated]`). Within the Tier L requirement ceiling of 25.
  *Scope stated*: MP-1 binds the requirement layer in `spec.md`. `plan.md`'s § C digest carries a
  duplicate `REQ-C-018` label — reported separately as D8, not as an MP-1 failure.

- **[PASS] MP-2 GEARS format compliance — judged against the REQUIREMENT LAYER ONLY.** The
  judgment was made against the 24 `REQ-C-XXX` entries in `spec.md` § 3 (spec.md:292-612).
  `acceptance.md`'s Given-When-Then entries are the verification layer and were graded under
  Group 4 (AC-1), never here. The census below was computed **from the requirement bodies, not
  from the bracket labels**, and every label was then checked against its own body — all 24 agree:

  | Pattern | Count | Requirements |
  |---|---|---|
  | Ubiquitous (`The <subject> shall …`) | 7 | C-001, C-005, C-006, C-007, C-016, C-018, C-023 |
  | Ubiquitous, negated response (`… shall not …`) | 6 | C-004, C-009, C-011, C-013, C-017, C-024 |
  | Event-driven (`When <event>, … shall …`) | 4 | C-002, C-010, C-012, C-019 |
  | Event-detected (`When <…detected>, … shall …`) | 3 | C-015, C-021, C-022 |
  | State-driven (`While <state>, … shall …`) | 1 | C-014 |
  | State-driven, negated response | 1 | C-003 |
  | Capability gate (`Where …`, compound with `When`) | 2 | C-008, C-020 |

  7+6+4+3+1+1+2 = 24. Every entry maps onto a pattern under **either** governing enumeration —
  the M3 rubric's five (Ubiquitous / Event-driven / State-driven / Where / Unwanted, where the six
  `Ubiquitous — negated` and the one `While — negated` entries are the `shall not` Unwanted form),
  or this project's `.claude/skills/moai-workflow-spec/SKILL.md`:L53-61 "GEARS Five Patterns
  (current notation)" table (Ubiquitous / Event-driven / State-driven / Capability gate /
  Event-detected, where `Unwanted` sits only in the separate "EARS Five Patterns (legacy)" table at
  L76-84). The SPEC's own label vocabulary follows the project table and is conformant with it;
  the two enumerations differ from each other, not from the SPEC. Recorded as D11 for the record —
  a rubric-vs-SSOT discrepancy, **not** a SPEC defect.

  Modality sweep: `sed -n '292,600p' spec.md | grep -nE '\b(should|may|might|could)\b'` returns
  **no match**; over spec.md:292-613 `grep -c 'shall'` returns 61 and `grep -c 'shall not'`
  returns 8. Every obligation carries `shall` or `shall not`. Zero residual `If/then` forms
  requiring the `[DEPRECATED — use WHEN]` marker.

  Two forms required an explicitly stated judgment rather than a pattern lookup:
  - **REQ-C-008's `Where X` / `Where not X` pair** (spec.md:408-417). The SPEC argues at
    spec.md:276-280 that a complementary exhaustive pair is one gate stated over both arms.
    Accepted: the arms are mutually exclusive and exhaustive, the response clause is the same
    obligation under each, and the compound `Where + When` form is PASS-equivalent at rubric 1.0.
  - **REQ-C-021's `unless` clause** (spec.md:561-571). The `unless that detection occurs inside a
    migration confirmed under REQ-C-019` is an exception carve-out narrowing the single `When`
    trigger's operand set, not a second trigger. One `When`, one PASS/FAIL. Accepted.

- **[PASS] MP-3 YAML frontmatter validity.** Re-verified field-by-field against
  `.claude/rules/moai/development/spec-frontmatter-schema.md` § Canonical 12 Required Fields.
  All 12 present with correct types (spec.md:1-17):

  | Field | Value | Verdict |
  |---|---|---|
  | `id` | `SPEC-ANKICARD-001` | matches the SPEC-ID form |
  | `title` | quoted, non-empty | OK |
  | `version` | `"0.1.2"` quoted semver | OK — bumped from `"0.1.1"` |
  | `status` | `draft` | in the 8-value enum |
  | `created` | `2026-09-05` | ISO `YYYY-MM-DD` |
  | `updated` | `2026-09-05` | ISO `YYYY-MM-DD` |
  | `author` | `jay` | non-empty |
  | `priority` | `P2` | in `P0`\|`P1`\|`P2`\|`P3` |
  | `phase` | `"v0.2.0 target"` | release target, not a lifecycle-stage value |
  | `module` | `"internal/anki, cmd/imoogi-anki, modules/anki, modules/24-anki.el"` | path-like, non-empty |
  | `lifecycle` | `spec-anchored` | in enum |
  | `tags` | comma-separated, non-empty | OK |

  **Zero rejected snake_case aliases** — no `created_at`, `updated_at`, `labels`, `spec_id`. The
  two optional fields (`tier: L`, `amendment_of: SPEC-ANKI-001`) are both in the schema's optional
  table, and `amendment_of` remains paired with the `### Amendments` sub-section.

- **[N/A] MP-4 Section 22 language neutrality.** N/A: single-project SPEC, unchanged at v0.1.2.
  Scope is one brownfield codebase — a Go binary (`internal/anki`, `cmd/imoogi-anki`) plus an Emacs
  Lisp front end (`modules/anki/*.el`), per `module:` and § 7 Brownfield Delta (spec.md:829-936).
  Neither template-bound nor universal multi-language content, so the 16-language
  equal-enumeration criterion does not apply and auto-passes.

- **[PASS] MP-5 D7 cross-SPEC reconciliation.** Verb re-executed.
  `grep -Eo 'SPEC-([A-Z][A-Z0-9]+-)+[0-9]+' spec.md | sort -u` returns three: `SPEC-ANKI-001`,
  `SPEC-ANKICARD-001` (self), `SPEC-TRANSIENT-001`.
  - `SPEC-TRANSIENT-001` — exists at `.moai/specs/SPEC-TRANSIENT-001/spec.md`, `status: draft`.
    Not in {retired, superseded, archived}. No reconciliation required.
  - `SPEC-ANKI-001` — not present in this repo's `.moai/specs/` → D7-5 SHOULD finding (carried as
    D1, unchanged from iteration 1). The parent is declared out-of-repo with an absolute path that
    resolves, its `status:` reads `in-progress` — **not** retired/superseded/archived — so D7-4
    does not fire. § 4 (spec.md:613-704) supplies explicit reconciliation: the verbatim REQ-016
    amendment (§ 4.2), the AC-016 replacement pair (§ 4.3), the three-exclusion supersession
    (§ 4.4), and the D-12 assumption (§ 4.5).

  **No D7 BLOCKING finding.** MP-5 PASS.

- **[PASS] MP-6 D8 cross-platform discipline.** Verb re-executed: `grep -c 'syscall' spec.md`
  returns **0**. Per D8-4, absence of the literal substring is auto-PASS. **No D8 BLOCKING
  finding.**

- **[PASS] MP-7 clarification gate.** Verb re-executed and widened beyond the minimum:
  `grep -rn '\[NEEDS CLARIFICATION' plan.md research.md spec.md acceptance.md design.md | wc -l`
  returns **0** across all five Tier L artifacts. Both `plan.md` and `research.md` exist, so the
  N/A branch does not apply. `acceptance.md`'s DoD independently records that `plan.md` revision 2
  closed all ten prior markers (acceptance.md:478-480).

---

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|---|---|---|---|
| Clarity | 1.0 | 1.0 | **Iteration 1's sole Clarity deduction is fully discharged.** REQ-C-007.2 (spec.md:381-407) no longer carries aesthetic vocabulary; it fixes seven named CSS custom properties with literal values in a 3-column table (`--imoogi-bg: #f7f3e9` / `#1a1a1a`, `--imoogi-measure: 44rem`, `--imoogi-line-height: 1.6`, the two literal font stacks) and names the exact declarations that consume each. `design.md` § 4.3 (design.md:258-264) states the split explicitly — "REQ-C-007.2 fixes what the values are… a value appearing in both places is a drift site" — and restates none of them, so the two artifacts cannot drift. Further clarity gains at v0.1.2: REQ-C-006.2 gains the empty-token sentinel `unnamed` (spec.md:368-375) so the emitted class is never a bare `deck-`; REQ-C-015 and REQ-C-022 name their diagnostic literals (`media_file_not_found`, `media_upload_failed`, `migration_add_failed`); REQ-C-012.3 cites `design.md` § 9.3 for the stored-name function instead of restating it; multi-actor REQ-C-005/008/020 restated with one subject plus `realized as:` clauses so PASS/FAIL is defined for the requirement as a unit. § 3's preamble (spec.md:272-289) now states the sub-clause rule as a checkable proposition and names its one exception. No pronoun-reference ambiguity found. |
| Completeness | 1.0 | 1.0 | HISTORY spec.md:18; WHY/Overview § 1 spec.md:165; WHAT/Scope § 1.2; REQUIREMENTS § 3 spec.md:261 (24 entries); ACCEPTANCE CRITERIA `acceptance.md` (24 top-level, acceptance.md:27-434); Out of Scope § 5 spec.md:705. **SC-6 satisfied concretely and improved:** now **nine** `### Out of Scope — <topic>` H3 sub-headings (spec.md:716, 726, 735, 741, 750, 767, 773, 780, 785), up from eight, each carrying at least one specific `-` bullet. Sections beyond the minimum: § 2 Glossary (spec.md:205), § 4 Parent-SPEC Amendment, § 6 Constraints, § 7 Brownfield Delta, § 8 Dependencies, § 9 Traceability. All 12 frontmatter fields present (MP-3). All 5 Tier L artifacts present and substantive. |
| Testability | 1.0 | 1.0 | **Iteration 1's sole Testability deduction is fully discharged.** AC-C-005 (acceptance.md:91-114) now asserts all seven custom properties **with their exact literal values**, on `.card` for the light set and under **both** night-mode selector forms for the night set, plus each consumption site (`background: var(--imoogi-bg)`, `font-family: var(--imoogi-serif)` on `h1`-`h6`, `max-width: var(--imoogi-measure)` on the deck wrapper, `color: var(--imoogi-emphasis-fg)` on the math-and-code rule) — and states its own mechanism: "Every one of these is a substring assertion over the embedded asset". The air-gap clause remains a zero-occurrence assertion over `http`, `@import`, `url(`. **Zero weasel words across the whole file**: `grep -nEi '\b(appropriate\|adequate\|reasonable\|proper\|as needed\|if necessary)\b' acceptance.md` returns 0. New AC-C-006d (acceptance.md:129-139) binds REQ-C-008's Elisp→Go transport to an exact document shape `{protocol_version, ankiconnect_url, user_css}` plus a zero-match grep for `.css` filesystem reads in the Go packages. AC-C-018b scopes its zero-request assertion precisely to **writes** and states why (the Handshake `requestPermission` read is permitted) — a distinction that would otherwise make the criterion unpassable. The DoD (acceptance.md:435-481) carries measured coverage floors attributed to a named baseline: "`go test -count=1 -cover ./internal/...` in this repository at `HEAD f6ee148` (2026-09-05)", with `ankiconnect` explicitly recorded as measured 84.3 against a floor set at 85.0 and the reason stated. AC-C-024 is declared observe-and-record rather than asserted (acceptance.md:425-434, DoD acceptance.md:442-446) — **not** a testability deduction: its subject is upstream renderer behavior at a pinned version, the classification is explicit rather than silent, and it is still exercised as a regression test. |
| Traceability | 0.92 | 1.0 (three deductions, all in `plan.md`) | Bidirectional check re-executed with `comm` over sorted unique identifier sets against the **new** numbering. **REQs in `spec.md` with no AC coverage: none** (empty result). **REQs referenced in `acceptance.md` that do not exist in `spec.md`: none** (empty result). Stale pre-consolidation identifier sweep (`REQ-C-025`..`REQ-C-099`): `spec.md` 0, `acceptance.md` 0, `design.md` 0, `research.md` 0, `plan.md` 21 — and the 21 are confined to lines 88-131, the § Revision 3 historical mapping table, a legitimate record. Cross-artifact citations the new requirement text introduces were each resolved to a real section that says what the requirement claims: `design.md` § 9.3 (design.md:520-541, the stored-naming table — sanitized basename + `-` + first 12 hex of content SHA-256 before the extension, matching REQ-C-012.3 verbatim), § 2.3 (design.md:106-120, the four-input hash table, matching REQ-C-017), § 7.1 (design.md:371-382, the migrate sequence showing the dry-run issues zero AnkiConnect writes, matching REQ-C-019's rationale pointer), § 4.3 (design.md:258-289). **Deductions:** three `plan.md` sections were not carried forward to v0.1.2 — the § C duplicate `REQ-C-018` (D8), the § B.4 two-vs-three exclusion count (D9), and the § C stale pattern census and labels (D10). All three are in the plan digest; `spec.md`, `acceptance.md`, `design.md`, and `research.md` are clean. |

**Aggregate (harmonic mean, per the Skeptical Evaluation Stance):**
4 / (1/1.0 + 1/1.0 + 1/1.0 + 1/0.92) = 4 / 4.08696 = **0.98**.

0.98 >= 0.85 (Tier L threshold). The verdict is not score-sensitive: dropping Traceability to the
0.75 band still yields a harmonic mean of 0.92, and dropping every dimension to 0.75 yields 0.75 —
below threshold only in a scenario no evidence in this report supports.

**Score regression check (LEAN Workflow):** iteration 1 = 0.96, iteration 2 = 0.98. The score
**improved**; the STOP escalation clause does not fire. The improvement is attributable and not
shaped to avoid the clause: iteration 1's only two scored deductions were the D6 aesthetic-clause
pair (Clarity 0.95, Testability 0.90), and both were discharged completely by the REQ-C-007.2 and
AC-C-005 rewrites. The three new deductions are all minor `plan.md` digest-staleness findings
concentrated in one dimension.

---

## Re-partition Risk Sweep (iteration-2 specific)

The re-partition's characteristic failure is a **re-pointed identifier carrying stale semantics** —
a pointer updated to the new number while still meaning the old one. Checked by reading the
affected bodies, not their headings.

`acceptance.md` — clean:
- **AC-C-017** (acceptance.md:288-299) points at `REQ-C-016` and its body asserts the first-run
  mass invalidation — exactly the obligation folded from former `REQ-C-018` into `REQ-C-016.2`
  (spec.md:507-514). Correct after the fold, including AC-C-017b's complement (a note with neither
  math nor a local image issues **no** request).
- **AC-C-018** (acceptance.md:300-324) points at `REQ-C-018, REQ-C-019` and asserts the migrate
  subcommand's wire shape, the `migrate_candidate` dry-run reporting through `results[]`, and the
  confirmation gate — matching the **new** REQ-C-018 (spec.md:523-533) and REQ-C-019
  (spec.md:534-542), not the old numbering's meaning.
- **AC-C-019** (acceptance.md:325-343) points at `REQ-C-020, REQ-C-021` and asserts
  add-before-delete, the three-way write-back, and the declared-type residue skip — matching new
  REQ-C-020 (spec.md:543-560).
- **AC-C-020** (acceptance.md:344-360) points at `REQ-C-021, REQ-C-022` and asserts the failed-add
  no-op **and** the sync-path complement — matching the split into new REQ-C-021 (mismatch,
  spec.md:561-571) and REQ-C-022 (add failure, spec.md:572-581).

`design.md`, `research.md` — clean; no stale-semantics case found.

`plan.md` — **three staleness findings**, D8/D9/D10 below. `plan.md` § F, § G, § H and § J were
swept separately (`grep -nE 'REQ-C-01[6-9]|REQ-C-02[0-2]' plan.md` filtered to lines > 460): the
pointers at :477, :783, :789-790, :800-801, :814, :928, :980, :982, :987 and :1037 all carry the
**new** semantics — :987 (risk R12) reads "REQ-C-016.2 makes it specified behavior", which is the
post-fold identifier, and :800-801 correctly separates REQ-C-020's `Where`-gated confirmed
migration from REQ-C-021's every-other-path rule. Sections § D and § E carry no requirement-number
pointers in the affected range.

**CN-1 re-check, harder than iteration 1.** v0.1.2's HISTORY records that v0.1.1's REQ-C-024.3
prohibited *reading* a foreign note type while REQ-C-002's `modelNames` probe and REQ-C-005.2's
`modelFieldNames` read both require it — a genuine contradiction that **iteration 1 did not catch**.
Stated plainly for the record: review-1 scored the SPEC 0.96 with CN-1 implicitly clean, and the
contradiction was found by the author, not by this auditor. The v0.1.2 text is now checked
directly: REQ-C-024.3 (spec.md:599-604) carries the explicit carve-out "Reading a foreign model is
not prohibited: the `modelNames` probe of REQ-C-002 and the `modelFieldNames` read of REQ-C-005.2
are both required", and REQ-C-024.4 (spec.md:605-612) is narrowed to the stock types' "model
definitions, their card templates, or their CSS", routing note-level work to REQ-C-005.2 and
REQ-C-020. The contradiction is resolved. The other rewritten requirements were swept against
their neighbours: REQ-C-003 (no model writes during an ordinary sync) vs REQ-C-002 (install step)
is disjoint by the § 2 glossary's separation of *ordinary synchronization run* from *install step*;
REQ-C-004 vs REQ-C-002's `updateModelStyling` is disjoint by the `imoogi-` ownership predicate;
REQ-C-021 vs REQ-C-020 is disjoint by REQ-C-021's explicit `unless` carve-out, stated as
partitioned "**by requirement text**, not by branch ordering". **CN-1 PASS on `spec.md`.**

**Count-preserving addition check.** HISTORY claims `acceptance.md` "gains an AC for the Elisp → Go
stylesheet transport" while top-level count stays 24. Verified: `grep -cE '^#{2,4} AC-C-'
acceptance.md` returns **24**, and the new assertion landed as sub-criterion **AC-C-006d**
(acceptance.md:129-139) under the existing AC-C-006, covering REQ-C-008's `user_css` field. The
claim holds.

---

## Regression Check (iteration 1 defects D1-D7)

- **D1 — D7-5-ANKI-001** (`SPEC-ANKI-001` not in this repo's `.moai/specs/`) — **UNRESOLVED BY
  DESIGN, and correctly so.** Re-executed: the D7 verb still emits the SHOULD finding. Iteration 1
  required no fix; the parent remains out-of-repo, its `status:` is still `in-progress` (not
  retired/superseded/archived), and § 4's reconciliation is intact. No action. Carried below as D1.
- **D2 — PARENT-STATUS-INCONSISTENCY** — **RESOLVED.** `research.md`:L22 now reads: "`SPEC-ANKI-001`
  … **is recorded as `status: in-progress` in its own frontmatter** and is behaviorally closed at
  25/25 AC … It is treated here as behaviorally closed and specification-live — the same
  formulation `spec.md` § Amendments uses; **neither artifact asserts a `completed` status the
  parent's frontmatter does not carry**." The overstated "complete and CLOSED" is gone and the two
  artifacts now agree on a verifiable fact.
- **D3 — SUBCLAUSE-TRIGGER-CLAIM** — **RESOLVED**, and resolved structurally rather than by
  softening the prose. The re-partition split former REQ-C-021's second `When` into its own
  requirement. New REQ-C-020 (spec.md:543-560) carries one `Where + When` trigger with its
  write-back folded under the single add `When` as sub-clauses 1-3. The preamble at spec.md:272-280
  was strengthened at the same time — sub-clauses are now defined as "case-splits on that trigger's
  operand, never additional moments" — and I verified that claim holds for all 24 requirements,
  with REQ-C-008's complementary `Where`/`Where not` pair explicitly named as the one exception
  and argued as a single gate (accepted, see MP-2).
- **D4 — RATIONALE-IN-REQUIREMENT-TEXT** — **RESOLVED (beyond what was required).** Iteration 1
  required no fix. The clearest case, former REQ-C-020's standalone non-`shall` rationale sentence,
  was relocated: new REQ-C-019 (spec.md:534-542) now ends with the pointer "(`design.md` § 7.1
  records why obtaining the count does not presuppose the answer.)" and design.md:371-382 carries
  the sequence. REQ-C-016.2 does the same with "(`design.md` § 2.3 records why.)".
- **D5(a) — AC-C-a / AC-C-b in plan.md** — **RESOLVED BY RE-POINTING, not by deletion.**
  `grep -nE 'AC-C-[ab]\b' plan.md | wc -l` returns **0** (was 7). Verified the references survive
  under the correct identifiers rather than having been dropped: `grep -n 'AC-C-003[ab]' plan.md`
  returns seven hits — :282 and :286 define the pair in § B.3, and :508, :598, :850, :875, :891
  consume it in § D and § F, including "so AC-C-003a's 'exactly 0' is assertable" on the
  `fake_client_test.go` touch-surface row. The pointer chain is intact end to end.
- **D5(b) — acceptance.md lettering self-description** — **RESOLVED.** acceptance.md:10-12 now
  reads "some carrying sub-lettered sub-criteria lettered `a`..`d` (`AC-C-0NNa` .. `AC-C-0NNd`)",
  which covers the `c` and `d` sub-criteria the file actually carries.
- **D6 — AESTHETIC-CLAUSE-UNVERIFIED** — **RESOLVED.** Both halves. The requirement half:
  REQ-C-007.2's six unbounded aesthetic terms replaced by seven named custom properties with
  literal values plus their consumption declarations. The verification half: AC-C-005 asserts every
  one of those values and every consumption site as a substring assertion. This was iteration 1's
  only scored Testability deduction and its only Clarity deduction; discharging it is what moved
  both dimensions to 1.0.
- **D7 — AC-CEILING-INTERPRETATION** (recorded, not a defect) — **UNCHANGED.** 24 top-level
  criteria, within the Tier L ceiling of 25; the sub-lettered expansion is still declared at
  acceptance.md:10-12 as sub-assertions within one logical criterion. Iteration 1 accepted that
  interpretation; iteration 2 accepts it on the same reasoning and records the acceptance so a
  later auditor need not re-derive it. Carried below as D7.

**Stagnation check:** no defect appears unchanged across both iterations. Of the six actionable
iteration-1 findings, six are resolved (D1 required no fix and is unchanged by design; D7 is a
recorded interpretation, not a defect). manager-spec made substantive, verifiable progress.

---

## Defects Found (structured defect-list)

D8. **PLAN-DIGEST-DUPLICATE-REQ-ID** — `plan.md`:L426 and `plan.md`:L434 — `plan.md` § C, the
requirement digest, carries the identifier `REQ-C-018` **twice with different content**: § C.5
(hashing) has "- **REQ-C-018 → folded into REQ-C-016.2 at revision 4.** When the first
synchronization run after this rendering change executes…", and § C.6 (migration) has
"- **REQ-C-018** [Ubiquitous] Migration is reached through the `imoogi-anki migrate` CLI
subcommand…". A reader scanning § C's bullet list meets the same identifier bound to two
obligations. **This is not an MP-1 failure**: MP-1 binds the requirement layer in `spec.md`, where
`REQ-C-018` appears exactly once (spec.md:523) and the 24 identifiers are unique. It is also
partially self-mitigating — the § C.5 entry leads with its own fold annotation, so the ambiguity is
resolvable in place. — Severity: minor — Class: optional — Required fix: strike the identifier from
the § C.5 entry, e.g. relabel it "*(former REQ-C-018 — folded into REQ-C-016.2 at revision 4;
retained as the record of the obligation's origin)*", so the live `REQ-C-018` is the only bearer of
that label in § C.

D9. **PLAN-B4-EXCLUSION-COUNT-STALE** — `plan.md`:L295-310 vs `spec.md`:L613+ § 4.4 — v0.1.2's
HISTORY records "§ 4.4 now records **three** superseded or inherited parent § 4 exclusions, the
third being the parent's 'note-type changes on an already-synced entry', which REQ-C-020
supersedes for the confirmed-migration case only." `spec.md` § 4.4 is titled "three exclusions,
handled differently" and lists all three. `plan.md` § B.4 is still titled "**two** exclusions,
handled differently" and lists only two — the custom-note-types supersession and the
collection-ownership inheritance. The middle disposition, which is the one authorizing REQ-C-020's
delete-and-re-add against the parent's exclusion, is absent from the plan's amendment section.
`spec.md` is the authoritative surface and is correct; the plan is stale. The disposition is not
lost from the artifact set — `plan.md` § B.5 and D-C-13 still carry the D-12 assumption and the
REQ-021 disjointness — but a reader working from § B.4 alone would count two dispositions where
three are in force. — Severity: minor — Class: optional — Required fix: retitle `plan.md` § B.4 to
"three exclusions" and add the "note-type changes on an already-synced entry" bullet, mirroring
`spec.md` § 4.4's middle entry.

D10. **PLAN-DIGEST-CENSUS-AND-LABEL-STALE** — `plan.md`:L93-95, and `plan.md`:L350, L376, L388,
L403, L422, L469 — `plan.md` § Revision 3 carries the **pre-v0.1.2** pattern census: "(all five
GEARS patterns retained): Ubiquitous 7, event-driven `When` 5, state-driven `While` 2,
capability-gate `Where` 2, Unwanted 6, event-detected 2". `spec.md`:L52-53 carries the v0.1.2
census: "Ubiquitous 13 (of which 6 negated), event-driven `When` 4, state-driven `While` 2 (of
which 1 negated), capability-gate `Where` 2, event-detected 3". Both total 24, but the breakdowns
disagree, and `plan.md`'s reflects the v0.1.1 partition (before former REQ-C-022 was split, which
is what moved event-detected from 2 to 3). Relatedly, `plan.md` § C still labels six requirements
`[Unwanted]` and uses `[Event-driven]` / `[Capability gate + Event-driven]` where `spec.md` v0.1.2
uses `[Ubiquitous — negated]`, `[When]`, and `[Where + When]`. **Direction of the fix stated
explicitly, because it is the opposite of what it first appears:** `spec.md`'s labels are the ones
conformant with this project's canonical GEARS table
(`.claude/skills/moai-workflow-spec/SKILL.md`:L53-61, "GEARS Five Patterns (current notation)",
whose fifth pattern is Event-detected and which places `Unwanted` only in the separate "EARS Five
Patterns (legacy)" table at L76-84). `plan.md` carries the EARS-legacy vocabulary. Bring `plan.md`
up to `spec.md`, not the reverse. — Severity: minor — Class: optional — Required fix: recompute
`plan.md`:L93-95's census against the v0.1.2 partition (the body-derived table in this report's
MP-2 section is available as the reference), and relabel `plan.md` § C's six `[Unwanted]` entries
plus the `[Event-driven]` / `[Capability gate + Event-driven]` entries to match `spec.md` § 3.

D11. **RUBRIC-VS-PROJECT-SSOT-DISCREPANCY** (recorded against the harness, **not** against this
SPEC) — this auditor's M3 rubric vs `.claude/skills/moai-workflow-spec/SKILL.md`:L53-61 — The two
governing enumerations of "the five GEARS patterns" differ in their fifth member. The M3 rubric
lists **Unwanted** ("The <subject> shall not [action] — GEARS canonical negative form") and does
not list Event-detected as a separate pattern. The project's SPEC skill lists **Event-detected**
("replaces IF/THEN") as the fifth and places Unwanted only in its "EARS Five Patterns (legacy —
6-month backward-compatibility window)" table. SPEC-ANKICARD-001 v0.1.2 relabelled its negated
requirements on the stated ground that "`Unwanted` is the EARS-legacy name for the negated form of
an existing pattern, not a sixth GEARS pattern" (spec.md:285-289, HISTORY spec.md:56-59) — a claim
that is **correct against the project skill** and is the reason no MP-2 or CN-1 finding is raised
for it. Recorded so a later auditor reading MP-2 against the M3 rubric alone does not mistake the
SPEC's conformant labelling for a defect. — Severity: minor — Class: optional — Required fix: none
against this SPEC. If the discrepancy is to be closed, it belongs to the harness (reconcile the
plan-auditor M3 rubric with `moai-workflow-spec/SKILL.md`), not to SPEC-ANKICARD-001.

D1. **D7-5-ANKI-001** — `spec.md`:L67-68, § 4 — Carried forward unchanged from iteration 1. The D7
verb still emits its SHOULD-severity "referenced SPEC not found" finding for `SPEC-ANKI-001`,
surfaced rather than absorbed per M2. Already resolved in place: the parent is declared out-of-repo
with an absolute path that resolves, its status is `in-progress`, and § 4 reconciles every clause
the amendment touches. No BLOCKING condition; MP-5 stands PASS. — Severity: minor — Class: optional
— Required fix: none required. If the parent is ever vendored into this repo's `.moai/specs/`, drop
the out-of-repo path prose.

D7. **AC-CEILING-INTERPRETATION** (recorded, not a defect against the SPEC) —
`acceptance.md`:L10-12 — 24 top-level acceptance criteria, within the Tier L ceiling of 25,
expanding to sub-lettered sub-assertions declared as "sub-assertions within one logical criterion".
Iteration 2 accepts this interpretation on the same reasoning iteration 1 gave — the ceiling's
stated purpose is auditor-holdability, and 24 logical criteria grouping related assertions about
one requirement cluster is holdable in a way a flat expansion would not be. Recorded so it need not
be re-derived at iteration 3. — Severity: minor — Class: optional — Required fix: none.

**No blocking-class findings. No critical- or major-severity findings. No unresolved must-pass
failures.** All six open findings are minor and optional; three (D8, D9, D10) are `plan.md` digest
staleness from the v0.1.2 edit, and three (D11, D1, D7) are records rather than fixes.

---

## Recommendation

**PASS at 0.98, above the Tier L threshold of 0.85.** Every must-pass verdict was re-executed
against v0.1.2 rather than carried over from iteration 1, because the re-partition invalidated
iteration 1's citations:

1. **MP-1** — 24 unique contiguous identifiers `REQ-C-001`..`REQ-C-024` with 24 matching headings
   in monotonic order, spec.md:292-589. No gap, no duplicate, uniform padding.
2. **MP-2** — judged against the requirement layer only. A census computed from the requirement
   *bodies* (7 Ubiquitous / 6 negated-Ubiquitous / 4 Event-driven / 3 Event-detected / 1
   State-driven / 1 negated-State-driven / 2 Capability-gate = 24) confirms every requirement maps
   onto a pattern under either governing enumeration, every bracket label agrees with its own body,
   and a modality sweep over spec.md:292-600 returns zero informal `should`/`may`/`might`/`could`.
   Given-When-Then entries in `acceptance.md` are the verification layer and were graded under
   Group 4.
3. **MP-3** — all 12 canonical fields present and correctly typed; `version` bumped to `"0.1.2"`;
   zero rejected snake_case aliases; both optional fields in the schema's optional table.
4. **MP-4** — N/A, single-project Go + Elisp scope, auto-passes.
5. **MP-5** — D7 verb re-executed over all three extracted references; one SHOULD finding (D1); no
   referenced SPEC carries a retired/superseded/archived status; no BLOCKING condition.
6. **MP-6** — D8 verb re-executed; `syscall` count 0; auto-PASS.
7. **MP-7** — clarification-gate grep returns zero matches across all five artifacts.

**The regression check is the strongest evidence in this report.** All six actionable iteration-1
findings are resolved, and two of them were resolved *structurally* rather than cosmetically. D3
was not fixed by softening the offending sentence — the requirement layer was re-partitioned so the
sentence became true, and the preamble was strengthened to a stronger, checkable claim (sub-clauses
are case-splits on the trigger's operand) which I verified holds for all 24 entries, with its one
exception named in the SPEC itself. D6 was not fixed by adding a vague assertion — the aesthetic
vocabulary was replaced with seven literal-valued custom properties and AC-C-005 rewritten to
assert each value and each consumption site, which is what moved Clarity and Testability to 1.0.
D5(a) was verified as resolved by *re-pointing* rather than by deletion: the AC-016 replacement
pair still has seven live references in `plan.md`, now under the correct `AC-C-003a` / `AC-C-003b`
identifiers. The author additionally found and fixed a CN-1 contradiction (REQ-C-024.3 prohibiting
reads that REQ-C-002 and REQ-C-005.2 require) that **iteration 1 missed** — stated plainly because
it bounds how much confidence review-1's 0.96 deserved on that dimension.

**One correction this auditor made mid-audit, recorded so it is not repeated.** An earlier pass of
this report raised the v0.1.2 pattern-label change as a blocking internal-consistency defect, on
the premise that `Unwanted` is one of the five GEARS patterns and that the SPEC had dropped it.
That premise was checked against `.claude/skills/moai-workflow-spec/SKILL.md` and found to be
wrong: the project's "GEARS Five Patterns (current notation)" table names Event-detected as the
fifth and confines Unwanted to a separately-headed EARS-legacy table, so the SPEC's relabelling is
conformant and its § 3 preamble does enumerate five distinct bases. The finding was withdrawn and
replaced by D11, which records the rubric-vs-SSOT discrepancy as a harness matter. Reachability of
a term in one table is not evidence of its status in another.

**The three remaining actionable findings are all `plan.md` digest staleness and none is blocking.**
The v0.1.2 edit updated `spec.md`, `acceptance.md`, `design.md`, and `research.md` and left three
`plan.md` sections behind: § C's duplicate `REQ-C-018` label (D8), § B.4's two-vs-three exclusion
count (D9), and § Revision 3's pattern census plus § C's EARS-legacy labels (D10). None affects an
obligation, an acceptance criterion, or an implementation decision — `spec.md` is authoritative on
all three points and is correct on all three. D9 is the highest-value single fix, because § B.4 is
the plan's amendment section and it currently under-counts the parent-SPEC dispositions that
authorize REQ-C-020.

**No STOP escalation.** The score moved 0.96 → 0.98; the LEAN score-regression clause does not
fire, and the improvement is attributable to the two discharged iteration-1 deductions rather than
to a re-weighting.

**This SPEC is approved to proceed to the Implementation Kickoff Approval human gate.** That gate
remains mandatory and is not affected by this verdict. Per M6, D8/D9/D10 are surfaced for the
orchestrator's discretion and are **not** a fix-then-re-audit route: routing them into a revision
would produce digest churn in an artifact whose authoritative counterparts are already correct. If
any subset is addressed, the confirming re-audit is scoped to that enumerated delta in `plan.md`
and does not require re-running the requirement-layer audit.
