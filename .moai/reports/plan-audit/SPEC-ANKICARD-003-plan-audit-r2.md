# SPEC Review Report: SPEC-ANKICARD-003

Iteration: 2/3
Verdict: **FAIL**
Overall Score: **0.706** (harmonic mean; Tier M PASS threshold 0.80)

Reasoning context ignored per M1 Context Isolation. This audit reads
`spec.md`, `plan.md`, `acceptance.md` at v0.1.1 (Tier M artifact set), the
iteration-1 report, and the tree at HEAD `6f3ae6c`.

**The verdict does not turn on the score.** Two new blocking defects — N1 and
N2 below — each ship a broken card and each are caught by no criterion and no
corpus row in the SPEC. Under the finding-consumption rule (M6), a blocking
finding is fixed before the verdict is revisited, independent of where the
aggregate lands. The score is reported honestly and happens to agree; had it
landed above 0.80 the routing would be the same.

**All twelve iteration-1 findings are closed.** None is marked partial. N1 and
N2 are new findings of the same defect class, not unresolved priors — N1 is a
fourth shape the closed D1/D3 fix does not reach, and N2 was *introduced* by
the D10 fix.

---

## Must-Pass Results

- **[PASS] MP-1 REQ number consistency** — `grep -c '^#### REQ-ML-' spec.md`
  returns 15; `grep -o 'REQ-ML-[0-9]\{3\}' spec.md | sort -u` returns exactly
  `REQ-ML-001`..`REQ-ML-015`, contiguous, no gaps, no duplicates, uniform
  three-digit padding. Unchanged from iteration 1, as the HISTORY entry claims.

- **[PASS] MP-2 GEARS format compliance** — judged against the **requirement
  layer** (`REQ-XXX` in `spec.md` § 3) only, per M3 § Scope. The
  Given-When-Then entries in `acceptance.md` are the verification layer and are
  graded under Group 4. Each of the fifteen carries exactly one GEARS trigger.
  The one iteration-1 label inconsistency (D9) is corrected: `REQ-ML-006`
  (spec.md:L266) now reads `[Ubiquitous — negated]`, matching its `shall not`
  lead clause and `REQ-ML-014`'s label at spec.md:L408. Sub-clauses remain
  case-splits on a single trigger's operand; spec.md:L166-168 states this and
  inspection confirms it.

- **[PASS] MP-3 YAML frontmatter validity** — all 12 canonical fields present
  with correct types at spec.md:L2-14. `version: "0.1.1"` (quoted semver,
  correctly bumped), `status: draft`, `created`/`updated` both ISO
  `YYYY-MM-DD`, `priority: P2`, `lifecycle: spec-anchored`, `tags` a
  comma-separated string. No rejected snake_case alias appears. Optional
  `tier: M` and `depends_on` are additive.

- **[N/A] MP-4 Section 22 language neutrality** — N/A: single-project SPEC
  targeting this repository's own Go packages and Elisp modules. No
  multi-language tooling surface. N/A auto-passes.

- **[PASS] MP-5 D7 cross-SPEC reconciliation** — the body references exactly
  `SPEC-ANKICARD-001` and `SPEC-ANKICARD-002`; both directories exist and both
  read `status: in-progress`. Neither is retired, superseded, or archived, so
  no reconciliation clause is required and no BLOCKING finding is emitted.

- **[PASS] MP-6 D8 cross-platform discipline** — `grep -c syscall` returns 0
  for all four artifacts. D8-4 auto-PASS.

- **[PASS] MP-7 clarification gate** — `grep -rn '\[NEEDS CLARIFICATION'` over
  the SPEC directory returns no match (exit 1). `research.md` is absent, which
  is correct for Tier M.

---

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.50 | 0.50 — multiple requirements require interpretation | Two requirements diverge from intent under a faithful reading. **REQ-ML-009.2** (spec.md:L322-330) says collapse blank-line runs "in the remaining body" with no qualifier, while `plan.md` DD-1 (L147-152) states the *scanner* skips block interiors — two steps, one skip rule stated for one of them. One engineer collapses everywhere, another skips blocks by analogy; the first mangles a code sample (N2, measured). **REQ-ML-006.1** (spec.md:L270-278) and **REQ-ML-005.2** (spec.md:L261-263) carry no precedence clause, so the all-pre-marked incremental case is undetermined (N3). Improved sharply from iteration 1's five shapes; D1, D2, D3, D4 and D6 are all resolved with measured evidence |
| Completeness | 0.75 | 0.75 — one surface sparse, frontmatter complete | Every required section present and substantive: HISTORY (L18), Overview (L91), Scope (L105), Requirements (L159), Out of Scope (L460, six `### Out of Scope — <topic>` H3 headings each with specific `-` bullets), Constraints (L519), Brownfield Delta (L536), Dependencies (L605), Traceability (L628). The Edge Cases table grew from 23 to 25 rows and now carries the checkbox and plain-sibling rows iteration 1 asked for. Deduction: REQ-ML-001.3 asserts "Three shapes exhaust the cases" (spec.md:L195), which is **false** for go-org v1.9.1 — a fourth parser-consumed prefix exists and is measured below (N1); the M1 corpus (plan.md:L564-582) and the Edge Cases table both omit it |
| Testability | 0.75 | 0.75 — criteria are decidable; coverage has holes | All four iteration-1 broken criteria are fixed and each fix is verified against the tree: AC-ML-006a is now satisfiable, AC-ML-014 names `tests/anki-error-test.el` (the file that actually carries the pairing assertions, confirmed by `grep -n ert-deftest`), AC-ML-013e is load-bearing, AC-ML-007c names a helper REQ-ML-007.3 requires rather than a function that cannot produce the form. Several criteria now name the specific failure signature they reject (`::[` at acceptance.md:L75, a split marker at L66), which is stronger than iteration 1. No weasel words. Deduction: four reachable failures have no criterion — N1's span, N2's block interior, N3's numbering, N4's authored double blank |
| Traceability | 1.00 | 1.0 — complete both directions | `grep '^### AC-ML-'` yields 15 headings naming `REQ-ML-001`..`REQ-ML-015`; `grep -o 'REQ-ML-[0-9]\{3\}' acceptance.md \| sort -u` yields the same 15. No orphaned AC, no uncovered REQ. The deliberate off-by-one from AC-ML-011 onward remains explicit in each heading |

Aggregate: harmonic mean of (0.50, 0.75, 0.75, 1.00) = 4 / (2 + 1.333 + 1.333 + 1)
= **0.706**. Tier M PASS threshold is 0.80 (`spec-workflow.md:L141, L329-330`).
The arithmetic mean, 0.750, is also below threshold; the verdict does not turn
on the choice of mean.

---

## Finding-by-Finding Closure Table

Twelve of twelve closed. "Closed" means the defect is gone, verified against
the artifact and, where the fix named a file or a behavior, against the tree.

| # | Iteration-1 finding | Status | Evidence |
|---|---|---|---|
| D1 | Checkbox item corrupted by whole-item wrapping | **CLOSED** | REQ-ML-001.3 second bullet (spec.md:L198-200) makes the checkbox a prefix; AC-ML-001f (acceptance.md:L67-76) asserts the span *and* rejects the `::[` corruption signature; Edge Cases row added; DD-3 (plan.md:L244-254) carries the re-measurement for both `[ ]` and `[X]`; M1 names it a span assertion. Re-measured here: `- [ ] {{c1::A}}` → `<li class="unchecked">{{c1::A}}</li>` clean, `- [-] {{c1::A}}` → `class="indeterminate"` also clean |
| D2 | AC-ML-006a unsatisfiable against REQ-ML-007.1 | **CLOSED** | Resolved by design change, not by softening. REQ-ML-006.1 (spec.md:L270-278) makes a span already carrying a hand-written marker **not wrapped**; AC-ML-006a (acceptance.md:L182-195) now asserts byte-for-byte survival *without* a generated marker, which is satisfiable. DD-5 (plan.md:L372-421) records the reversal and re-measures the nesting passthrough. Residual consequences analyzed below — one is N3 |
| D3 | Plain sibling in a description-kind list unspecified | **CLOSED** | REQ-ML-001.3 third bullet (spec.md:L201-205) names it explicitly, including *why* it is reachable (per-list classification). AC-ML-001g (acceptance.md:L77-86); Edge Cases row; DD-3 measurement. Re-measured: `- term :: {{c1::def}}\n- {{c2::plain}}` → `<dd>{{c1::def}}</dd> … <dt>?</dt><dd>{{c2::plain}}</dd>` clean |
| D4 | DD-1's scanner grammar not block-aware; corpus unnamed | **CLOSED** | DD-1 now carries four named rules with measurements (plan.md:L155-170): block interiors skipped, a list inside a block is not top level, blank-line-run behavior, base indent from the first bullet line. M1 enumerates the corpus as **required members** (plan.md:L564-582) — eleven shapes in ten rows, matching AC-ML-001's "eleven corpus shapes" claim — and flags the two that must be span assertions because count-equality is blind to them |
| D5 | AC-ML-014 named the wrong deciding file | **CLOSED** | acceptance.md:L455-460 now names `tests/anki-error-test.el` and states why. Verified against the tree: that file carries `imoogi-error-test-go-codes-are-subset-of-table` (L51), `imoogi-error-test-codes-match-source-file` (L57), and `imoogi-error-test-table-entries-are-all-paired-with-a-go-constant` (L101) |
| D6 | REQ-ML-006.2 fixed the title at `c2` unconditionally | **CLOSED** | REQ-ML-006.3 (spec.md:L282-287) is qualified to the incremental-**off** case and states what governs when incremental is on. AC-ML-003b (acceptance.md:L134-139) now pins `{{c2::Capital of Japan}}` with its rationale; AC-ML-006b (acceptance.md:L203-211) states the incremental-off condition is load-bearing |
| D7 | AC-ML-013's guard non-load-bearing after REQ-ML-010 | **CLOSED, strengthened** | Adjudicated in full below. AC-ML-013e (acceptance.md:L416-435) is the criterion D7 demanded, verbatim, plus `TestOptionFreeRequestLogGolden` named in the deciding command; REQ-ML-010.1 (spec.md:L344-352) adds structural delegation on top |
| D8 | AC-ML-007c named a function that cannot produce the form | **CLOSED** | REQ-ML-007.3 (spec.md:L299-306) is a *new obligation* requiring the editor to expose separation and pad as one callable helper; AC-ML-007c (acceptance.md:L233-252) names that helper and explains why not `imoogi-anki--cloze-safe-text`. Verified against the tree: the pad is a separate binding in the caller at `modules/org/24-anki.el:248`, so the requirement is warranted. The deciding file `tests/anki-commands-test.el` exists |
| D9 (optional) | Label inconsistency on REQ-ML-006 | **CLOSED** | spec.md:L266 now `[Ubiquitous — negated]` |
| D10 | `#+BEGIN_EXTRA`-between-answers silent loss | **CLOSED** | REQ-ML-009.2 (spec.md:L322-330) adopts the composition-local collapse; DD-4 (plan.md:L290-368) records the reversal, keeps both prior rejections, and explains why the re-hash objection does not reach this fix; AC-ML-009d/AC-ML-009e (acceptance.md:L281-302) are a falsifiable pair. **The fix works and introduces N2** |
| D11 (optional) | REQ-ML-002.1's rationale overreached | **CLOSED** | spec.md:L221-228 narrows the "no visible cue" claim to the empty-body case and gives a separate, correct reason for the prose-but-no-list case |
| D12 (optional) | REQ-ML-012 verified only under `->` | **CLOSED** | AC-ML-011d (acceptance.md:L345-352) builds the `<-` fixture and names the implementation mistake it exists to catch |

---

## D2's Four Consequences, Dispositioned

The brief named four consequences the REQ-ML-006.1 "not wrapped" rule might
not have been followed through on. All four were checked. **This section is
reasoning from the requirement text, not measurement** — nothing below was
probed.

**(a) Numbering when some items are wrapped and others are skipped —
SOUND.** REQ-ML-006.2 (spec.md:L279-281) starts generated numbering above the
highest hand-written number anywhere in the entry, and REQ-ML-005.2 assigns in
document order among the spans that are numbered. For `- Tokyo is
{{c4::big}}\n- Osaka\n`, AC-ML-006a (acceptance.md:L182-195) pins the second
item at `5`, so the composed case is determinate and criterion-covered. One
residual ambiguity — whether an unwrapped item consumes a position in the
document-order sequence — is closed by the same sentence N3's fix requires
("per-answer numbering binds only the wrapped spans").

**(b) Whether `ANKI_INCREMENTAL` still produces the design record's card
count — NOT SOUND. This is N3.** See the finding above: with every answer
pre-marked under a shared number, the not-wrap rule makes REQ-ML-005.2's
one-card-per-answer consequence fail, and no clause states which requirement
governs.

**(c) A leftward or bidirectional card whose every answer is pre-marked —
SOUND.** Under `<-` the answers are never wrapped anyway, so the rule changes
nothing; the title wraps at the offset above every hand-written number, and
REQ-ML-005.3's distinctness holds by construction. Under `<->` with every
answer pre-marked, the title still wraps above all of them, so it is still
distinct. If the title is *also* pre-marked, REQ-ML-006.1's "binds an answer
item and the title alike" leaves nothing wrapped — and the card still works,
because every span is already clozed by the author. That is the case
AC-ML-006d pins for `->`; the `<-` variant is uncovered by a criterion but is
not underspecified.

**(d) An author's marker using a number the composer would also generate —
SOUND, and impossible by construction.** REQ-ML-006.2's offset rule makes
collision unreachable: generated numbering begins strictly above the highest
hand-written number. The offset's scan scope is "the title or the remaining
body" (spec.md:L279-281), which is wider than the answer list, so a marker
sitting in a nested non-answer sub-item counts toward it too. AC-ML-006c
covers the title case. The one scope question the SPEC does not answer — a
marker inside a `#+BEGIN_EXTRA` block, which leaves the remaining body before
composition reads it — is recorded under Could Not Verify rather than filed,
because deciding it needs Anki behavior I did not test.

---

## Mechanical Verification Performed

All probes ran in an isolated Go module in the session scratchpad against
go-org v1.9.1, **never inside the checkout**. `git status --short internal/`
returns empty; `git status --short tests/` shows only the three foreign files
the brief named. Nothing in `internal/anki` was created, modified, or removed.

I also read go-org's parser source directly
(`$GOMODCACHE/github.com/niklasfasching/go-org@v1.9.1/org/list.go`), which is
where N1 and N5 come from rather than from guesswork.

### Confirmed by probe (mechanical)

| Claim | Result |
|---|---|
| `1. [@5] A` renders `<li value="5">A</li>` — the cookie is consumed as a prefix | CONFIRMED |
| `1. {{c1::[@5] A}}` renders `<li value="5">:[@5] A}}</li>` — marker opening destroyed | CONFIRMED |
| `1. [@5] {{c1::A}}` renders `<li value="5">{{c1::A}}</li>` — clean | CONFIRMED |
| `1) [@7] …` and `a. [@3] …` behave identically — every ordered terminator | CONFIRMED |
| `- [@5] A` in an **unordered** list is *not* consumed (cookie is ordered-only) | CONFIRMED |
| `1. [@5] [ ] {{c1::A}}` — cookie and status stack; both consumed, wrap clean | CONFIRMED |
| Two and three consecutive `#+BEGIN_EXTRA` blocks between answers: collapse restores one list | CONFIRMED (no defect) |
| Block at the very start / very end of the answer list: `splitExtraBlocks`'s own `TrimSpace` already removes the gap | CONFIRMED (no defect) |
| SRC block with an internal blank-line run: rendered output **differs** with and without the collapse | CONFIRMED (N2) |
| EXAMPLE block likewise | CONFIRMED (N2) |
| `- A\n- B\n\n\n- C\n- D\n` (author-written double blank) merges into one four-item list under the collapse | CONFIRMED (N4) |
| `- Tokyo [ ] is big` loses its first four characters **today**, before this SPEC | CONFIRMED (N5, pre-existing) |
| `- {{c1::Tokyo [ ] is big}}` loses the marker opening | CONFIRMED (N5) |
| `- [ ] term :: def` — checkbox and description prefixes stack cleanly | CONFIRMED (no defect) |
| Every iteration-1 fix that named a file or an Elisp test target resolves in the tree | CONFIRMED |

### Reached by reasoning, not by probe

N3 (the REQ-ML-005.2 / REQ-ML-006.1 precedence conflict) and the D7
adjudication are read from the artifact text and the requirement algebra. They
are not measured, and I say so rather than dressing them as probe results.

---

## Defects Found (structured defect-list)

**N1. The ordered-list counter cookie `[@N]` is a fourth parser-consumed
prefix, and REQ-ML-001.3 claims three shapes exhaust the cases** —
`spec.md:L193-211` (REQ-ML-001.3) / `plan.md:L564-582` (M1 corpus) —
Severity: **major** — Class: **blocking**.

This is the same corruption class as iteration 1's critical D1, in a shape the
D1/D3 fix does not reach. go-org's `parseListItem` consumes **three** prefixes
in a fixed order, not two: the counter cookie
(`listItemValueRegexp = \[@(\d+)\]\s`, applied when `l.Kind == "ordered"`),
then the status (`listItemStatusRegexp`), then the description term. The SPEC
enumerates the second and third and asserts the enumeration is exhaustive.

Measured against go-org v1.9.1:

```
IN : "1. [@5] A\n2. B\n"              OUT: <ol><li value="5">A</li><li>B</li></ol>
IN : "1. {{c1::[@5] A}}\n"            OUT: <ol><li value="5">:[@5] A}}</li></ol>
IN : "1. [@5] {{c1::A}}\n"            OUT: <ol><li value="5">{{c1::A}}</li></ol>
```

The opening `{{c1` is destroyed exactly as it was for the checkbox. The shape
is reachable: REQ-ML-001.2 (spec.md:L190) explicitly admits the ordered
form, and the cookie is Org's standard way of writing it. It fires on every
ordered terminator — `1.`, `1)`, `a.` all confirmed — and on no unordered list.

Nothing catches it. The count-equality cross-check is blind by construction,
which DD-3 itself states (`plan.md:L282-287`) as the reason the checkbox and
description shapes are span assertions; the cookie is a third span shape and is
in neither the M1 eleven-shape corpus nor the Edge Cases table. AC-ML-001f's
rejection signature is the literal string `::[`, which does not match the
cookie's corruption `:[@`.

The `shall` clause itself is **correct** — "the item's own content less any
structural prefix the renderer's parser reads ahead of that content" covers the
cookie. The defect is the false exhaustiveness claim attached to it, which is
what an implementer will code against.

**Required fix**: replace "Three shapes exhaust the cases" with go-org v1.9.1's
own consumption order — counter cookie (ordered lists only), then status, then
description term — and state that the list is pinned to the vendored parser
version. Add a span-assertion row `1. [@5] A\n2. B\n` to the M1 corpus, an
Edge Cases row, and an AC-ML-001 sub-criterion rejecting the literal `:[@` in
rendered output.

**N2. REQ-ML-009.2's collapse is unqualified and mangles block interiors in
the question context** — `spec.md:L322-330` / `plan.md:L290-368` (DD-4) —
Severity: **major** — Class: **blocking**.

Introduced by the v0.1.1 fix for D10. The requirement says composition "shall
collapse a run of consecutive blank lines in the **remaining body** to a single
blank line" — the whole body, with no exclusion. A `#+BEGIN_SRC` or
`#+BEGIN_EXAMPLE` block sitting in the question context has its internal
blank-line runs collapsed, and the rendered output changes:

```
BEFORE "#+BEGIN_SRC text\nline1\n\n\nline2\n#+END_SRC\n\n- A\n"
  → <div class="src src-text">\nline1\n\n\nline2\n</div>…
AFTER  "#+BEGIN_SRC text\nline1\n\nline2\n#+END_SRC\n\n- A\n"
  → <div class="src src-text">\nline1\n\nline2\n</div>…
identical = false
```

Confirmed for `#+BEGIN_EXAMPLE` too. `#+BEGIN_QUOTE` also loses the run, though
there the rendered paragraphs are unaffected.

The blast radius is bounded — the collapse runs only for a multiline entry, so
REQ-ML-014's no-mass-re-hash guarantee holds — but within that scope it is a
**silent content change**, which is the same failure shape D10 was filed
against. An author's code sample is reformatted without diagnostic, warning, or
report signal. DD-4's stated intent is only to close the list gap; nothing in
it wants this.

The corrective already exists two paragraphs away and was not carried across:
DD-1 requires the *scanner* to skip block interiors (`plan.md:L147-152`), for
the same reason. The collapse is a separate step in M1's deliverable
(`plan.md:L555` lists "the blank-line collapse, the scanner, …" as two
things) and inherits nothing.

**Required fix**: qualify REQ-ML-009.2 so the collapse skips block interiors —
the same skip DD-1 already requires of the scanner — and add one AC-ML-009
sub-criterion asserting that a body carrying a `#+BEGIN_SRC` block with an
internal blank-line run renders identically with and without the collapse.

**N3. REQ-ML-006.1 and REQ-ML-005.2 carry no precedence, so the
all-pre-marked incremental case is undetermined** — `spec.md:L270-278` /
`spec.md:L261-263` / `acceptance.md:L196-202` (AC-ML-006d) — Severity:
**minor** — Class: **blocking**.

REQ-ML-005.2 requires that with incremental on "each answer item shall carry
**its own** number, assigned in document order", and gives the consequence as
its rationale: "so the entry yields one card per answer". REQ-ML-006.1 requires
that a span already carrying a hand-written marker "shall **not** be wrapped".

For body `- {{c1::A}}\n- {{c1::B}}` with incremental on, the two items carry
the *same* number. REQ-ML-006.1 forbids composition from changing that, so
REQ-ML-005.2's obligation is unmet and the entry yields one card rather than
two. No clause states which requirement governs.

AC-ML-006d pins the all-pre-marked case but builds it with `{{c1::…}}` and
`{{c2::…}}` — distinct numbers — and sets no incremental value, so it passes
without ever exercising the conflict. The reader of the Edge Cases row "Every
answer item hand-clozed" is told the entry renders, not what incremental does
to it.

This is reasoning, not measurement: it is a requirement-algebra conflict, and
I did not run anything to establish it.

**Required fix**: one precedence sentence in REQ-ML-006.1 stating that the
not-wrap rule governs and that REQ-ML-005.2's per-answer numbering binds only
the spans that are wrapped, so the one-card-per-answer consequence is scoped to
them.

**N4. The collapse silently merges a deliberately-authored two-list body, and
no criterion or Edge Cases row covers it** — `spec.md:L322-330` /
`spec.md:L495-504` (§ 5) — Severity: **minor** — Class: **optional**.

Measured: `- A\n- B\n\n\n- C\n- D\n` renders today as two lists and, under the
collapse, as one four-item list — so `C` and `D` become answers. A blank-line
run is the only way Org lets an author write two adjacent lists of the same
bullet without intervening prose, so this is the shape the change actually
takes away.

The behavior **is disclosed**: § 5's narrowed exclusion (spec.md:L500-504) and
DD-4's "What this does to DD-4's own scope" (plan.md:L360-366) both say a
blank-line run no longer separates one authored list into two. Classed optional
for that reason. What is missing is coverage: the Edge Cases table exists
expressly "so a reader can check the list is complete" (acceptance.md:L480-481)
and carries the block-residue and paragraph-separated cases but not this third
one, and AC-ML-009's falsifiable pair does not include it.

**Required fix**: one Edge Cases row, and optionally one AC-ML-009
sub-criterion pinning the merge so it stays deliberate.

**N5. go-org's status and counter regexps are unanchored and slice
positionally, which the SPEC's "prefix" model does not describe** —
`spec.md:L193-196` — Severity: **minor** — Class: **optional**.

`listItemStatusRegexp` and `listItemValueRegexp` are matched with
`FindStringSubmatch` (unanchored), after which the parser slices the content by
a **fixed length from position zero**. A `[ ] ` anywhere in an item therefore
destroys the item's first four characters:

```
IN : "- Tokyo [ ] is big\n"           OUT: <li class="unchecked">o [ ] is big</li>
IN : "- {{c1::Tokyo [ ] is big}}\n"   OUT: <li class="unchecked">::Tokyo [ ] is big}}</li>
IN : "1. Tokyo [@5] big\n"            OUT: <li value="5">[@5] big</li>
```

Classed optional because the first line shows the corruption is **pre-existing**
— the item already renders broken today, without this SPEC. The relevance is
that REQ-ML-001.3's model ("the structural prefix the parser reads ahead of
that content") does not describe this parser: there is no leading prefix here,
and an implementation that searches for a *leading* `[ ] ` finds none, wraps
whole, and produces the second line above. AC-ML-001f's `::[` signature does
not catch it either.

**Required fix**: none required for this SPEC. If taken, one sentence in
REQ-ML-001.3 noting that the parser's prefix consumption is position-blind, so
the scanner must mirror the parser's own match rather than look for a leading
token.

---

## Adjudication of the Author's D7 Dissent

**The dissent is upheld. The fix is genuinely stronger, and it does not trade
one unfalsifiable claim for another.**

The author's position is that iteration 1 asked for a criterion and the author
instead required REQ-ML-010.1 to make the new entry point **delegate** to
`orgdoc.Render` for the non-multiline path, with AC-ML-013e guarding the
delegation.

Read against the artifact, the premise is not quite right — and the outcome is
better than the dissent claims. **The criterion iteration 1 asked for is
present, verbatim.** D7's required fix read: "AC-ML-013 gains a sub-criterion
asserting that the new entry point with no options set produces output
byte-identical to `Render` over every case in the existing corpus, and names
`TestOptionFreeRequestLogGolden` in its deciding command." AC-ML-013e
(acceptance.md:L416-435) asserts exactly that, over exactly that corpus, and
names exactly that test. The deciding command at acceptance.md:L437-440 carries
`go test ./internal/anki/planner -run TestOptionFreeRequestLogGolden -count=1`.

So the delegation is **additive**, not a substitute — which is the answer to
"stronger, or a different unfalsifiable claim?". The two cover different
things and neither is redundant:

- AC-ML-013e is executable and falsifiable. It drives the production entry
  point directly and fails on divergence. It covers only cases **in** the
  existing corpus.
- REQ-ML-010.1's delegation covers every no-option shape the corpus does
  *not* contain, which no finite criterion can reach. It is a structural
  guarantee, and its observable consequence is precisely what AC-ML-013e
  measures.

The chain that closes the falsy case holds too: AC-ML-013c establishes
falsy ≡ no-value, AC-ML-013e establishes no-option ≡ `Render`, and the
Glossary's definition of "multiline entry" (spec.md:L131-137) puts an
all-falsy entry on the delegating branch. Nothing in that chain is assumed.

**One observation, not a defect.** The delegation itself has no direct
assertion. An implementer who reimplements rather than delegates, and happens
to match on every corpus case, violates REQ-ML-010.1 undetectably. That is a
violated requirement with no observable consequence, since what users depend on
— byte-identity over the corpus plus the real-run golden — is what AC-ML-013e
and `TestOptionFreeRequestLogGolden` actually measure. Worth one sentence in
the run phase's review, not a finding here.

This section is reasoning, not measurement. I did not execute the Go suite.

---

## Regression Check (iteration 2)

Every iteration-1 defect was checked for resolution. None is unresolved; the
per-defect evidence is in the closure table above.

- D1 — RESOLVED (REQ-ML-001.3 bullet 2, AC-ML-001f, Edge Cases row, M1 span assertion)
- D2 — RESOLVED (REQ-ML-006.1 design change, AC-ML-006a rewritten, DD-5 replaced)
- D3 — RESOLVED (REQ-ML-001.3 bullet 3, AC-ML-001g, Edge Cases row)
- D4 — RESOLVED (DD-1's four measured rules, M1's eleven required corpus shapes)
- D5 — RESOLVED (`tests/anki-error-test.el`, verified present with the pairing assertions)
- D6 — RESOLVED (REQ-ML-006.3 qualified, AC-ML-003b pins `c2`, AC-ML-006b scoped)
- D7 — RESOLVED and strengthened (AC-ML-013e + REQ-ML-010.1 delegation)
- D8 — RESOLVED (REQ-ML-007.3 new obligation, AC-ML-007c names the required helper)
- D9 — RESOLVED (label corrected)
- D10 — RESOLVED as to the answer loss (REQ-ML-009.2 collapse, AC-ML-009d/e pair) — **and the fix introduced N2**
- D11 — RESOLVED (rationale narrowed)
- D12 — RESOLVED (AC-ML-011d)

**No stagnation.** No defect appears unchanged across both iterations.

**No score regression, so no STOP signal.** 0.63 → 0.706 is upward on every
dimension that moved (Clarity 0.50 → 0.50 with a much smaller residual,
Testability 0.50 → 0.75; Completeness and Traceability held). The LEAN
score-regression clause does not fire, and iteration 3 is the ordinary path
with a three-item blocking delta.

---

## Could Not Verify

- **Anki's own rendering.** Whether Anki accepts `<sup>`, `<br>`, or a marker
  spanning `</p><ul><li>` inside a cloze deletion. I ran no Anki instance. DD-2
  and DD-5 are both explicit that this is unverified and treat it as a reason
  to be conservative, which remains the right handling. N1, N2, N4, and N5 are
  Org-level and go-org-level and do not depend on it.
- **Whether the M1 property test will be written to the enumerated corpus.**
  D4's fix makes the corpus a named obligation, which is the most a plan-phase
  audit can establish; whether the implementer honors it is a run-phase
  question.
- **The Go and Elisp suites.** I did not run `go test ./...`, `make lint`, or
  `make test-elisp`. The brief supplied the baseline at `6f3ae6c` and I did not
  re-derive it. AC-ML-015 pins those figures; nothing in this audit depends on
  re-measuring them.
- **The Elisp criteria AC-ML-012a-e.** Judged by reading. I confirmed the
  `imoogi-anki-cloze-region` note-type contract exists and that the pad is a
  separate binding at `modules/org/24-anki.el:248` (which is what makes
  REQ-ML-007.3 warranted), but I did not execute the transient or the
  property-writing path.
- **Whether a hand-written marker inside a `#+BEGIN_EXTRA` block should count
  toward REQ-ML-006.2's offset.** Anki numbers cloze deletions per note across
  fields, and the extra block leaves the remaining body before composition
  reads it. The SPEC scopes the offset to "the title or the remaining body"
  (spec.md:L279), which excludes it. Whether that is right depends on Anki
  behavior I did not test, so no defect is filed.

---

## Recommendation

**FAIL at iteration 2/3**, on three blocking findings, routed under M6 rather
than on the aggregate score.

This is a substantially better SPEC than v0.1.0. All twelve prior findings
closed, two of them by reversing a design decision rather than patching a
sentence, and the author's dissent on D7 is upheld. The requirement and
criterion counts held at 15 and 15, as the HISTORY entry claims, and no fix
expanded scope.

Fix in this order. All three are bounded edits to existing entries; no
requirement needs to be added and no design decision reversed.

1. **N1 (major)** — replace REQ-ML-001.3's "Three shapes exhaust the cases"
   with go-org v1.9.1's actual consumption order (counter cookie for ordered
   lists, then status, then description term), pinned to the vendored version.
   Add the `1. [@5] A` span-assertion row to the M1 corpus, an Edge Cases row,
   and an AC-ML-001 sub-criterion rejecting `:[@` in rendered output.
   `spec.md:L193-211`, `plan.md:L564-582`, `acceptance.md:L67-86, L483-507`.
2. **N2 (major)** — qualify REQ-ML-009.2 so the collapse skips block interiors,
   matching the skip DD-1 already requires of the scanner, and add an AC-ML-009
   sub-criterion asserting that a `#+BEGIN_SRC` block with an internal
   blank-line run renders identically with and without the collapse.
   `spec.md:L322-330`, `plan.md:L147-152`, `acceptance.md:L281-302`.
3. **N3 (minor)** — add one precedence sentence to REQ-ML-006.1: the not-wrap
   rule governs, and REQ-ML-005.2's per-answer numbering binds only the wrapped
   spans, so the one-card-per-answer consequence is scoped to them.
   `spec.md:L270-278`.

**N4 and N5 are optional** — surfaced for the orchestrator's discretion, not
routed into the revision. N4 is already disclosed in § 5 and DD-4 and needs
only an Edge Cases row; N5 is pre-existing go-org behavior this SPEC does not
make worse.

Do not let the count manufacture scope expansion. Three blocking findings, each
a sentence-level edit plus one test row. The spine — DD-1's pre-render,
source-level composition — remains correct and confirmed, and the D2 and D10
reversals are both improvements this audit endorses.

Re-audit at iteration 3 will be scoped to this three-item defect delta plus a
regression check over N1, N2, N3 and the twelve prior findings.

---

*Working-tree note: every probe ran in an isolated Go module under the session
scratchpad. `git status --short internal/` is empty; `git status --short
tests/` shows only the three foreign files the brief named. No git command that
writes was run.*
