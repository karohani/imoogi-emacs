# SPEC Review Report: SPEC-ANKICARD-002

Iteration: 1/3
Verdict: **PASS**
Overall Score: **0.91** (harmonic mean; Tier M threshold 0.80)

Reasoning context ignored per M1 Context Isolation. The audit reads
`spec.md`, `plan.md`, `acceptance.md` (Tier M input contract) plus the
repository tree they bind.

Verdict scope note: PASS is the firewall-and-rubric verdict. Two findings
below are classified **blocking** and MUST be fixed before Implementation
Kickoff Approval. PASS does not mean ship as-is.

## Must-Pass Results

- **[PASS] MP-1 REQ number consistency** — `REQ-OPT-001`..`REQ-OPT-013`,
  contiguous, zero gaps, zero duplicates, uniform three-digit padding.
  Verified: `grep -o 'REQ-OPT-[0-9]\{3\}' spec.md | sort -u` yields exactly 13
  ids; `grep -c '^#### REQ-OPT-' spec.md` yields 13. Heading/id counts agree,
  so no id is defined twice or referenced without a heading.

- **[PASS] MP-2 GEARS format compliance** — judged against the **requirement
  layer** (`REQ-XXX` in `spec.md` §3), not the verification layer. All 13
  carry an explicit pattern tag and a conforming `shall` sentence:
  7× `[Ubiquitous]`, 2× `[Ubiquitous — negated]` (the `shall not` form:
  `spec.md:L138` "The front end shall not interpret, normalize, default,
  coerce, or write any card-option property value"), 1× `[When]`
  (`spec.md:L196` "When the back end receives a request document whose
  declared protocol version differs from its own"), 3+1× `[Where]`. No
  informal language, no Given-When-Then presented as a requirement, no mixed
  form. The Given-When-Then entries in `acceptance.md` are `AC-XXX`
  verification-layer criteria and are correctly graded under Group 4, not
  here. See D9 for a pattern-selection quality finding that does not reach
  the MP-2 failure conditions.

- **[PASS] MP-3 YAML frontmatter validity** — all 12 canonical fields present
  with correct types, checked field by field against
  `.claude/rules/moai/development/spec-frontmatter-schema.md`: `id`,
  `title`, `version` ("0.1.0", quoted), `status` (draft), `created`,
  `updated` (both ISO `2026-09-20`), `author`, `priority` (P2), `phase`,
  `module`, `lifecycle` (spec-anchored), `tags` (comma-separated string).
  Zero rejected snake_case aliases — no `created_at`, `updated_at`,
  `labels`, or `spec_id`. Two permitted extras carried: `tier: M`,
  `depends_on`.

- **[N/A] MP-4 Section 22 language neutrality** — N/A: project-scoped Go plus
  Emacs Lisp for one specific repository, not template-bound or universal
  content. The SPEC names no cross-language tooling and enumerates no
  language set, so the 16-language equal-weight obligation has no subject.
  (Stated as project-scoped rather than single-language: it is two
  languages.)

- **[PASS] MP-5 D7 cross-SPEC reconciliation** — the D7 verb executed. Extracted
  SPEC references: `SPEC-ANKICARD-001` and the SPEC's own id. Both resolve
  to an existing `.moai/specs/<ID>/spec.md`. `SPEC-ANKICARD-001` carries
  `status: in-progress` — not in {retired, superseded, archived} — so no
  reconciliation clause is required and no BLOCKING finding is emitted.
  (D2 below is a self-consistency defect in how the dependency is stated,
  not a D7 lifecycle finding.)

- **[PASS] MP-6 D8 cross-platform discipline** — the D8 verb executed:
  `grep -c 'syscall' spec.md plan.md acceptance.md` returns 0 for all three.
  D8-4 auto-PASS. No BLOCKING finding.

- **[PASS] MP-7 clarification gate** — `grep -rn '\[NEEDS CLARIFICATION'` over
  the SPEC directory returns zero matches. No `research.md` exists (Tier M
  does not require one); `plan.md` carries no marker. No open clarification
  topic.

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.90 | 0.75–1.0 | Requirements read unambiguously and the glossary (`spec.md` §2) pre-defines every contested term — "option-bearing entry", "truthy/falsy spelling", "carrying a card-option value". Deductions: D2 (`spec.md:L468` contradicts `plan.md:L402`), D7 (bare `nil` carries two meanings in `acceptance.md`), D9 (4× `[Where]` for a runtime state condition). |
| Completeness | 1.00 | 1.0 | HISTORY §, WHY (§1.1 Purpose), WHAT (§1.2 Scope), HOW (§7 Brownfield Delta, per-surface `[NEW]`/`[CHANGED]`/`[UNCHANGED]`), REQUIREMENTS §3 (13 entries), ACCEPTANCE CRITERIA (`acceptance.md`, 14 entries), Out of Scope §5 with **7** `### Out of Scope — <topic>` H3 sub-headings each carrying specific `-` bullets. Frontmatter complete (MP-3). |
| Testability | 0.85 | 0.75–1.0 | Strong: AC-OPT-009's 17-row value table, AC-OPT-010's 8-row combination table, AC-OPT-011b's two suppressed-code probes (`acceptance.md:L340`ff), AC-OPT-012c's request-log byte-identity with its explicit empty-errors clause. Deductions: D1 (REQ-OPT-001.1 unfalsifiable), D3 (`acceptance.md:L76` method contradicts its own Then), D6 (`acceptance.md:L363` "inspected", not executed). |
| Traceability | 0.90 | 0.75–1.0 | Every AC heading names the requirements it verifies; every AC names the command that decides it; `plan.md` §F maps requirements onto M1–M6 with no requirement unmapped. Deductions: sub-clause REQ-OPT-001.1 has no discriminating criterion (D1); AC-OPT-012's `Decides` line assigns three commands collectively rather than per sub-criterion (D6). |

Harmonic mean = 4 / (1/0.90 + 1/1.00 + 1/0.85 + 1/0.90) = 4 / 4.39869 = **0.9094** → **0.91**.
Tier M threshold 0.80 (`.claude/rules/moai/workflow/spec-workflow.md` § SPEC Complexity Tier). 0.91 ≥ 0.80.

## Defects Found

**D1. REQ-OPT-001.1 has no criterion that can fail** — `spec.md:L125` /
`acceptance.md` AC-OPT-001 — Severity: **major** — Class: **blocking**.
REQ-OPT-001.1 requires reuse of the existing chain "rather than introduce a
second one, so that a change to the chain's semantics cannot apply to
`ANKI_DECK` and not to a card-option property." AC-OPT-001a–d test
own-drawer-wins, nearest-ancestor-wins, file-level-last, and
present-but-empty termination. **A hand-rolled second implementation would
pass all four.** The one chain rule that actually discriminates reuse from
reimplementation is `PROPERTY+` normalization to plain replacement —
implemented at `modules/org/anki/imoogi-props.el` `imoogi-props--own-value`
(regexp `:%s\+?:`, last matching drawer line wins, never an append), named in
`spec.md` §2 and again at `spec.md:L371`, and tested by **no criterion**.
Contrast AC-OPT-007, which carries a real anti-duplication assertion for the
Go predicate. Required fix: add **AC-OPT-001e** — given a heading whose own
drawer carries `:ANKI_DIRECTION: ->` followed by `:ANKI_DIRECTION+: <-`, the
resolved direction is `<-` (plain override, never `-> <-` appended), decided
by `tests/anki-props-test.el`.

**D2. The SPEC-ANKICARD-001 dependency is stated two contradictory ways, and
one of them rests on an unverified premise** — `spec.md:L468` vs
`plan.md:L402` — Severity: **major** — Class: **blocking**.
`spec.md:L468` states 001's "closure precedes this SPEC's run phase, per the
design record § 5 risk table" — supported: that table reads "t12~t15는 001
종료 후 착수" (t12 through t15 begin after 001 closes). `plan.md:L402`
dispositions the same risk as "Recorded, not mitigated — the user directed
the ordering knowingly (design record § 7)". These cannot both hold: one
gates the run phase on 001's closure, the other treats proceeding as already
authorized. The §J citation does not support its claim — design record §7 is
the execution log for **t11 and t12 only**, and `plan.md` §A's own sequencing
paragraph says the user directed "that t11 and t12 proceed regardless",
naming t13 nowhere. Extending that directive to t13 is a
recommendation-premise claim asserted rather than observed
(`verification-claim-integrity.md` §1.1 surface 4). Mechanically the
dependency is ordering-only and nothing is missing: `model.ClozeName`
(`internal/anki/model/model.go:37`), `renderType`
(`internal/anki/planner/planner.go:372`), the diagnostic const block, the
bidirectional pairing test, and `imoogi-anki-cloze-note-type-p`
(`modules/org/24-anki.el:81`) are all on `main` today. The blocker is
process (001's sync-audit is unclosed), not code availability. Required fix:
pick one. Either keep `spec.md:L468`'s gate and delete the `plan.md:L402`
disposition, or drop the gate and route "proceed before 001 closes?" to the
Implementation Kickoff Approval gate as an explicit user decision — and state
in either case that nothing this SPEC reads from 001 is unlanded.

**D3. AC-OPT-002c's assertion method contradicts its own Then clause** —
`acceptance.md:L76` — Severity: **minor** — Class: **blocking**.
The Then clause is correctly scoped ("no card-option property appears in any
drawer that did not already carry it, and no existing card-option value is
modified"), but the method sentence is "Asserted by comparing buffer contents
before and after." A sync run **does** modify the buffer: `imoogi-writeback.el`
calls `org-entry-put (point) "ANKI_NOTE_ID"` at lines 114 and 133. A
whole-buffer comparison over "any Org buffer" after "a scan and a sync run
complete" therefore fails for a reason this criterion is not about. Required
fix: narrow the method to the card-option property lines — compare the three
properties' drawer values before and after, or diff the buffer and assert the
delta touches only `ANKI_NOTE_ID`.

**D4. The migration path has a third gate-bypass arm, neither named nor
pinned** — `spec.md:L219` (REQ-OPT-008.4) / `acceptance.md:L327`
(AC-OPT-011c) — Severity: **minor** — Class: **blocking**.
REQ-OPT-008.4 says "The same gate shall govern the ordinary synchronization
path and the migration path". AC-OPT-011c pins one known bypass — the
`--dry-run` arm returning before the per-entry pipeline. But
`internal/anki/planner/migrate.go` `Migrate` has a **second** early return: an
entry failing `migrationTarget` goes to `nonCandidate` and returns before any
`orgdoc.Render`, so a gate placed before the render call never runs for it. A
non-candidate entry carrying a malformed option raises no diagnostic in a
migrate run. Behaviourally harmless (the next ordinary sync catches it via
`processEntry`), but the SPEC establishes the pattern of pinning known bypass
arms and this one is absent. Required fix: one clause in REQ-OPT-008.4
bounding the migration-path obligation to confirmed candidates, plus one row
in AC-OPT-011c pinning the non-candidate arm as inherited behaviour.

**D5. AC-OPT-005's verification grep is narrower than the claim it decides** —
`acceptance.md:L132` / `acceptance.md:L473` (DoD item 4) — Severity: **minor**
— Class: **optional**. The stated grep filters `--include='*.go'
--include='*.el'` and its alternation omits the hyphen keyword
`:protocol-version`. `tests/anki-process-test.el:54` pins the version as
`(should (= (plist-get parsed :protocol-version) 1))` and is caught only
because that file happens to contain `protocol_version` elsewhere. **No pin
is missed today** — I re-ran the grep with `:protocol-version` added and
without the include filter and found no Anki-subsystem file outside the
14-row table — but DoD item 4 re-runs *this* grep, so a future pin in a
JSON/testdata fixture or in an `.el` file using only the hyphen form would be
invisible to the verification step. Required fix: add `:protocol-version` to
the alternation and drop the `--include` filters, re-filtering by path
instead.

**D6. AC-OPT-012b is half inspection, and its deciding command cannot execute
its other half** — `acceptance.md:L363` — Severity: **minor** — Class:
**optional**. The When clause reads "When its signature and inputs are
inspected" — a code-review assertion, not an executable check. The second
half ("the hash computed with all three options null and the hash computed
with `direction: ->`, `incremental: t` are equal") is genuinely falsifiable,
but cannot be expressed by `go test ./internal/anki/hashing`:
`hashing.Hash(noteType, fields, deck, tags)` takes no card-option parameter,
so the two-entry comparison has to live in the planner test. AC-OPT-012's
`Decides` line names all three packages collectively, so it is covered by
accident rather than by assignment. Required fix: assign the hash-invariance
assertion explicitly to `go test ./internal/anki/planner`, and replace
"inspected" with a compile-time or signature-reflection assertion.

**D7. `nil` carries two distinct meanings in `acceptance.md` without
disambiguation** — `acceptance.md` AC-OPT-001d, AC-OPT-003 vs AC-OPT-008,
AC-OPT-010 — Severity: **minor** — Class: **optional**.
"the resolved swift value is nil (no value)" and "three option values are all
nil" use `nil` for Elisp's no-value; `swift: nil` and the `direction | nil`
table rows use `nil` for the falsy **spelling** — a non-null string. DD-1
carefully disambiguates these on the wire ("`null` is 'no value resolved';
`"nil"` is 'the user explicitly wrote the falsy spelling'") but the
Elisp-side criteria do not carry that discipline. This sits precisely in the
inheritance-opt-out area where the two are most easily confused. Required
fix: write "no value" or "null" for the unresolved case and quoted `"nil"`
for the spelling, throughout `acceptance.md`.

**D8. DD-5 extends a confirmed decision without DD-2's gate routing** —
`plan.md:L153` — Severity: **minor** — Class: **optional**.
The design record §2.1 principle 4 lists `ANKI_DIRECTION`(`->`|`<-`|`<->`),
`ANKI_INCREMENTAL`(`t`), `ANKI_SWIFT`(`t`). For the two booleans it names
only an on-value and specifies no off-spelling, so `t`/`nil` fills a gap
rather than expanding a decision. **`nil` on `ANKI_DIRECTION` is a genuine
extension** — that property's value set is enumerated exhaustively as three
arrows. `plan.md` §A calls §2's decisions "fixed inputs to this plan, not
open questions", yet this one clause is extended without DD-2's reversibility
flag or plan-gate routing. The justification is also partly circular: it
introduces `nil` on two properties, then cites the user's habituation to
`nil` as the reason to add it to the third. The non-circular half stands —
the explicit opt-out spelling should be uniform or it is a trap. Required
fix: give the `ANKI_DIRECTION`-accepts-`nil` clause the DD-2 treatment —
mark it an expansion of a confirmed decision, state the collapse cost, and
surface it at the plan gate.

**D9. Four requirements tag `[Where]` for a per-entry runtime condition** —
`spec.md` REQ-OPT-003, REQ-OPT-009, REQ-OPT-010, REQ-OPT-011 — Severity:
**minor** — Class: **optional**. GEARS reframes `Where` as capability gate /
feature flag / static config. "Where an entry carries a non-null card-option
value the back end does not recognize" is a state condition over a runtime
entity, which is `[While]`, or `[When]` on the encountering event.
`spec.md` §3's preamble defends the usage explicitly ("Numbered sub-clauses
are case-splits on that trigger's operand"), and the sentence form is
syntactically conformant, so this is pattern-selection quality rather than a
GEARS violation — it does not reach MP-2's failure conditions. Required fix
(optional): retag the four as `[While]`, or add one sentence to §3's preamble
recording the deliberate use of `Where` as a case-split operator over an
entity set.

## Claims Verified Mechanically (not defects — recorded as evidence)

These SPEC claims were re-derived from the tree rather than trusted:

1. **The 14-row protocol-version pin table is complete and every row is
   real.** An independent grep widened with `:protocol-version` and stripped
   of the `--include` filters found no Anki-subsystem file outside the table.
   Per-file pin counts, all ≥ 1: `protocol.go` 3, `protocol_test.go` 12,
   `protocol/install_test.go` 6, `main.go` 9, `main_test.go` 14,
   `migrate_test.go` 2, `cmd/.../install_test.go` 4, `log_test.go` 1,
   `imoogi-process.el` 7, `anki-process-test.el` 5, `anki-install-test.el` 7,
   `anki-migrate-test.el` 1, `anki-sync-oneway-test.el` 1,
   `anki-sync-error-test.el` 5. The four out-of-scope subsystems the table
   names (`internal/clipboard/**`, `internal/orgpreview/**`,
   `modules/org/{23,28}-*.el`, `tests/org-preview-*`) are exactly the
   remaining grep hits.

2. **The cloze-style predicate is correct on both sides for all 8 rows of
   AC-OPT-007.** Back end: `renderType(noteType) == orgdoc.NoteTypeCloze`
   where `NoteTypeCloze = "Cloze"` (`orgdoc.go:20`) and `renderType`
   (`planner.go:372`) maps `model.ClozeName`→`Cloze`,
   `model.BasicName`→`Basic`, default pass-through. Front end:
   `(member type (list "Cloze" imoogi-anki-cloze-note-type))`
   (`24-anki.el:81`), case-sensitive. Both answer true for exactly `Cloze`
   and `imoogi-Cloze` and false for `Basic`, `imoogi-Basic`, `imoogi-Other`,
   `MyCloze`, `cloze`, and the empty string.

3. **REQ-OPT-008.4's declared-vs-counterpart claim is true for every reachable
   case.** `migrationTarget` (`migrate.go`) admits a candidate only when the
   registry-recorded type is stock **and** the declared type is that stock
   name or its counterpart. So declared ∈ {`Cloze`,`imoogi-Cloze`} (both
   cloze-style) or {`Basic`,`imoogi-Basic`} (neither). No divergence is
   reachable. AC-OPT-011d pins the case that matters.

4. **Both version-skew directions are mechanically sound, not waved at.**
   `probeProtocolVersion` (`main.go:236`) compares with strict `!=` against
   `protocol.Version` and is reached from `runOverSyncRequest` (`main.go:137`,
   serving `sync` and `migrate`) and `runInstall` (`main.go:278`) — three
   request-reading subcommands, one comparison site, which is exactly what
   AC-OPT-006c asserts is unchanged. The load-bearing detail for AC-OPT-006b:
   the probe unmarshals into an **anonymous struct carrying only
   `protocol_version`**, so an already-built v1 binary ignores the three
   unknown fields of a v2 request and still reaches the comparison. The
   fourth subcommand, `version`, reads no request and is correctly excluded.

5. **The error-taxonomy arithmetic in AC-OPT-014 is exact.** `protocol.go`
   declares 18 codes; `imoogi-error.el` carries 21 table entries;
   `imoogi-error-test--go-emitted-codes` lists 18;
   `imoogi-error-test--elisp-only-codes` lists 3 (`binary_not_found`,
   `sync_root_unset`, `note_id_unknown`). 18 + 3 = 21. Adding three codes
   takes 18→21 and 21→24 with the allowlist unchanged at 3, exactly as
   AC-OPT-014a–c state.

6. **`plan.md` §B item 1 is true.** `orgdoc.go`'s cloze arm emits `Back Extra`
   unconditionally — the source comment reads "Back Extra is emitted even
   when empty" — so t12 did change the field map of **every** cloze note, not
   only those carrying an extra block, and the hash covers every field key.
   The post-t12 baseline that AC-OPT-012a pins is the correct one.

7. **The renderer has exactly two non-test callers**, `planner.go:273` and
   `migrate.go:188`, matching `plan.md` §C. **The hash takes exactly four
   inputs** (`hashing.go` `Hash(noteType, fields, deck, tags)`), none of
   which the card options reach. **`imoogi-props-resolve` is a reusable
   generic** over a property name, with `-deck` and `-tags` as thin wrappers
   performing the empty→nil collapse — so DD-10's "thin wrappers, exactly as
   the deck and tags resolvers are" is accurate. **`5d3868f` is HEAD**, so
   §A's base-commit statement holds.

## Could Not Verify

- **Coverage floors.** `acceptance.md`'s Quality Gates cite
  `internal/anki/planner` "measured at 91.6% after t12" and
  `internal/anki/orgdoc` "100.0% maintained". I did not run `go test -cover`.
  The SPEC itself already treats these as needing re-measurement on the
  run-phase base commit rather than carry-over (Quality Gates closing
  paragraph, §E item 4), which is the correct disposition.
- **Whether the suites currently pass.** `make test-go` / `make test-elisp`
  were not executed; this is a plan-phase audit of specification quality, and
  no criterion's current pass state is claimed here.
- **The LOC and production-file estimates** in `plan.md` §I (400–700 LOC, 8
  production files). Estimates of unwritten code are not mechanically
  checkable at plan time.

## Answer on DD-2 (three diagnostic codes vs the card's one)

**Justified on the project's own error-taxonomy rule. Ship the three.**

The rule, as it exists in the tree rather than as DD-2 characterises it,
mandates **pairing**, not granularity: `protocol.go`'s const-block comment
says "Each has a matching entry in the front end's code-to-message table …
A code with no table entry is itself a defect", and
`tests/anki-error-test.el` enforces that in both directions with a
documented 3-entry allowlist and a negative test proving the reverse
assertion can fail. Nothing there dictates how finely failures are split.

What settles it is the precedent in the existing 18 codes, which split
consistently by **distinct corrective action**, not by subsystem:
`deck_create_failed` / `deck_move_failed` are two codes for two deck
operations; `media_file_not_found` / `media_upload_failed` are two for two
media failures; `delete_suppressed` / `delete_candidate_unowned` are two
codes for two reasons one delete did not happen — the closest analogue to
DD-2's situation, and it split. Collapsing three unrelated fixes (change the
note type / fix the value / remove one of two options) into one code would
make it the only code in the set covering three corrective actions.

The naming argument is decisive on its own: `card_option_needs_cloze` is the
name the card mandated, and it **describes the note-type fix**. Emitting it
for `ANKI_DIRECTION: -->` would make the code name assert something false
about the defect — worse than verbose, actively misleading.

It is a scope expansion in the literal sense — the card said one code — but
it is a six-line expansion inside three surfaces already marked `[CHANGED]`
in `spec.md` §7 (`protocol.go`, `imoogi-error.el`, `anki-error-test.el`),
enforced by an existing bidirectional test, with no new architecture. That
is expansion-within-an-established-pattern, not scope creep. DD-2's own
reversibility disclosure and gate routing are the right handling and should
be kept; I simply would not exercise the collapse.

## Recommendation

**Verdict PASS by must-pass firewall and rubric score (0.91 ≥ Tier M 0.80).
D1 and D2 are blocking-class and D3 and D4 are blocking-class; the
orchestrator routes all four to manager-spec before Implementation Kickoff
Approval.** PASS here is not "ship as-is" — none of the four is a redesign,
and all four are single-clause edits, but each fixes something the SPEC
currently gets wrong rather than merely thin.

Fix before kickoff, in priority order:

1. **D2** — resolve the dependency contradiction between `spec.md:L468` and
   `plan.md:L402`. This one gates whether the run phase may start at all, so
   it is first. State the ordering-only fact (everything read from 001 is on
   `main`) and pick one of the two positions.
2. **D1** — add AC-OPT-001e pinning `PROPERTY+` normalization for at least one
   card-option property, so REQ-OPT-001.1's reuse obligation has a criterion
   that can fail.
3. **D3** — narrow AC-OPT-002c's assertion method to the card-option lines;
   a whole-buffer compare cannot pass across a sync that writes
   `ANKI_NOTE_ID`.
4. **D4** — bound REQ-OPT-008.4 to confirmed candidates and pin the
   `nonCandidate` arm in AC-OPT-011c alongside the existing `--dry-run` row.

D5 through D9 are optional and left to the orchestrator's discretion. D5 and
D7 are the two I would still take: both are one-line edits, D5 repairs the
verification method DoD item 4 depends on, and D7 removes the single most
likely misreading in the SPEC's most subtle area.

Two things this SPEC does notably well, recorded so a revision does not
discard them. The non-interference claim **is** proved mechanically rather
than asserted: AC-OPT-012c compares the full request log byte-for-byte
against an M1 baseline and explicitly identifies the empty-errors slice as
the load-bearing half, because a rejected entry also reports `skipped` and
the action alone cannot distinguish untouched from rejected. And the
record-the-golden-before-editing hazard is guarded four separate times (M1
ordering, DoD item 5, §E item 1, §H anti-pattern 3), which is what keeps the
strongest criterion from degenerating into a tautology.

One calibration note on the contrast in D1: AC-OPT-007's "derived not
duplicated" clause is a regression guard — it tests the predicate against its
own defining expression — not a proof of derivation. It is still the right
shape and still more than the Elisp resolver gets, which is why D1 stands.
