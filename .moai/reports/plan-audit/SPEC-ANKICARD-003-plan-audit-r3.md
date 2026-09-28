# SPEC Review Report: SPEC-ANKICARD-003

Iteration: 3/3
Verdict: **PASS**
Overall Score: **0.923** (harmonic mean; Tier M PASS threshold 0.80)

Reasoning context ignored per M1 Context Isolation. This audit reads
`spec.md`, `plan.md`, `acceptance.md` at v0.1.2 (Tier M artifact set), the
iteration-2 report, and go-org v1.9.1 as resolved by the project's `go.mod`.

**Scope.** This is a delta audit, scoped to the v0.1.2 edits that answer N1-N5
plus a regression check over the closed iteration-1 and iteration-2 findings.
Closures iteration 2 already confirmed are cited, not re-derived.

**Score movement: 0.706 → 0.923. No regression, so no STOP signal.** The
LEAN score-regression clause does not fire.

**The verdict does not rest on the score alone.** All three blocking findings
N1-N3 are closed with criteria I measured as falsifiable. All seven must-pass
criteria clear. Three new findings are recorded below; every one is classed
**optional** under M6, and the classification is defended with a stated
discriminator rather than a hedge.

---

## Must-Pass Results

- **[PASS] MP-1 REQ number consistency** — `grep -c '^#### REQ-ML-' spec.md`
  returns 15; `grep -o 'REQ-ML-[0-9]\{3\}' spec.md | sort -u` returns exactly
  `REQ-ML-001`..`REQ-ML-015` — contiguous, no gaps, no duplicates, uniform
  three-digit padding. Unchanged from iterations 1 and 2, as the v0.1.2
  HISTORY claims (spec.md:L63).

- **[PASS] MP-2 GEARS format compliance** — judged against the **requirement
  layer** (`REQ-XXX` in `spec.md` § 3) only, per M3 § Scope. The
  Given-When-Then entries in `acceptance.md` are the verification layer and
  are graded under Group 4, not here. Each of the fifteen carries exactly one
  GEARS trigger. The four requirements v0.1.2 touched keep their pattern:
  REQ-ML-001 `[Where]` (spec.md:L228), REQ-ML-006 `[Ubiquitous — negated]`
  (spec.md:L333), REQ-ML-009 `[Ubiquitous]` (spec.md:L392). The v0.1.2 edits
  are all sub-clause body text; no lead clause changed, so no label drifted.

- **[PASS] MP-3 YAML frontmatter validity** — all 12 canonical fields present
  with correct types at spec.md:L1-16: `id`, `title`, `version` (quoted
  `"0.1.2"`), `status` (`draft`), `created`/`updated` (both ISO `2026-09-20`),
  `author`, `priority` (`P2`), `phase`, `module`, `lifecycle`
  (`spec-anchored`), `tags` (comma-separated string). No rejected snake_case
  alias (`created_at` / `updated_at` / `labels` / `spec_id`) appears. Two
  extra fields, `tier: M` and `depends_on`, are additive and schema-legal.
  `status: draft` confirmed as the lead asked.

- **[PASS] MP-4 Section 22 language neutrality** — N/A-equivalent, auto-pass.
  The SPEC is scoped to one Go module plus its Emacs Lisp editor commands; it
  names no multi-language tooling surface and claims no cross-language
  coverage. LN-3 applies.

- **[PASS] MP-5 D7 cross-SPEC reconciliation** — no BLOCKING finding.
  `grep -Eo 'SPEC-([A-Z][A-Z0-9]+-)+[0-9]+' spec.md | sort -u` yields
  `SPEC-ANKICARD-001`, `SPEC-ANKICARD-002`, and the SPEC's own id. Both
  referenced SPECs exist at `.moai/specs/<ID>/spec.md` and both read
  `status: in-progress` — neither is `retired`, `superseded`, nor `archived`,
  so D7-4 does not fire and no reconciliation clause is required.

- **[PASS] MP-6 D8 cross-platform discipline** — auto-PASS per D8-4.
  `grep -c 'syscall'` returns 0 for all three artifacts.

- **[PASS] MP-7 clarification gate** — `grep -rn '\[NEEDS CLARIFICATION'
  plan.md spec.md acceptance.md` exits 2 with no match. `research.md` does not
  exist, which is correct for Tier M.

---

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.75 | 0.75 — minor ambiguity in one or two requirements | REQ-ML-009.2 (spec.md:L398-405) leaves "block interior" undefined for an unterminated `#+BEGIN_`, and the two readings measurably diverge (NEW-1). REQ-ML-001.3's informative list (spec.md:L257-268) carries two measured inaccuracies that err in the safe direction (NEW-2). Every other requirement reads one way only. |
| Completeness | 1.00 | 1.0 | HISTORY (L18), Overview/Purpose as WHY (L137-150), Scope as WHAT (L151-161), Requirements (L205), Relationship (L514), Out of Scope (L542) with **six** `### Out of Scope — <topic>` H3 sub-headings each carrying specific `-` bullets, Constraints (L608), Brownfield Delta (L625), Dependencies (L694), Traceability (L717). Acceptance criteria live in `acceptance.md` per Tier M. All 12 frontmatter fields present. |
| Testability | 1.00 | 1.0 | Every AC is binary-testable and each section names the command that decides it. `grep -Ei '\b(appropriate\|adequate\|reasonable\|proper\|sufficiently)\b'` over `acceptance.md` and `spec.md` returns no match. I measured the four criteria v0.1.2 adds and confirmed each **can fail** — see the closure table. |
| Traceability | 1.00 | 1.0 | 15 REQs, 15 ACs. `grep -o 'REQ-ML-[0-9]\{3\}' acceptance.md \| sort -u` returns all fifteen, so no REQ is uncovered. Every `### AC-ML-` heading names its REQ in parentheses, and every named REQ exists in `spec.md`. No orphan in either direction. Within the Tier M 16/16 budget. |

Harmonic mean of (0.75, 1.00, 1.00, 1.00) = 4 / (4/3 + 3) = **0.923**.

---

## Closure Table — N1 through N5

| # | Finding | Class at r2 | Disposition | Evidence |
|---|---------|-------------|-------------|----------|
| N1 | Counter cookie is a fourth consumed prefix; "three shapes exhaust the cases" is false | blocking | **CLOSED** — by a stronger fix than the one required | spec.md:L239-268; acceptance.md:L86-98 (AC-ML-001h); plan.md:L660 (corpus row) |
| N2 | Collapse mangles block interiors | blocking | **CLOSED** | spec.md:L398-405; acceptance.md:L343-355 (AC-ML-009f); plan.md:L662 |
| N3 | No precedence between not-wrap and per-answer numbering | blocking | **CLOSED** | spec.md:L347-356; acceptance.md:L230-243 (AC-ML-006e) |
| N4 | Two-list merge uncovered | optional | **CLOSED** — taken | spec.md:L588-594 (§ 5); acceptance.md:L356-368 (AC-ML-009g), L574 (Edge Cases); plan.md:L663 |
| N5 | Unanchored, position-blind consumption not described by the "prefix" model | optional | **CLOSED** — taken, and folded into N1's rule as load-bearing | spec.md:L246-255; acceptance.md:L99-111 (AC-ML-001i), L556; plan.md:L661 |

### N1 — definitional beats enumerative here, and it is not a euphemism

**Adjudication: the author's substitution is upheld, and it is strictly
stronger than the fix iteration 2 required.**

The lead's test is the right one: an implementer must be able to compute the
offset, and if the SPEC does not say how, "definitional" is a euphemism. The
SPEC does say how, on three levels.

1. **The rule names its source.** spec.md:L242-245 — "the composer shall
   derive the offset from the parser's own rule for the vendored parser
   version, and shall **not** determine it by searching the item's text for a
   known token." The parser is vendored and version-pinned (`go.mod`:
   `github.com/niklasfasching/go-org v1.9.1`), so "the parser's own rule" is a
   reachable artifact, not an abstraction.

2. **The rule states the parser's mechanism, correctly.** spec.md:L246-255
   says two of the three consumptions are position-blind: the expressions are
   unanchored and the parser then removes **a fixed byte count from the start
   of the item** regardless of where the match occurred; only the description
   term is position-aware. I read the vendored source rather than trusting the
   claim:

   ```
   ~/go/pkg/mod/github.com/niklasfasching/go-org@v1.9.1/org/list.go
   L32  var listItemValueRegexp  = regexp.MustCompile(`\[@(\d+)\]\s`)
   L33  var listItemStatusRegexp = regexp.MustCompile(`\[( |X|-)\]\s`)
   L88  if m := listItemValueRegexp.FindStringSubmatch(content); m != nil && l.Kind == "ordered" {
   L89      value, content = m[1], content[len("[@] ")+len(m[1]):]
   L91  if m := listItemStatusRegexp.FindStringSubmatch(content); m != nil {
   L92      status, content = m[1], content[len("[ ] "):]
   L95      if m := descriptiveListItemRegexp.FindStringIndex(content); m != nil {
   L96          dterm, content = content[:m[0]], content[m[1]:]
   ```

   Both matches use `FindStringSubmatch` — unanchored. Both slices index from
   zero by a constant. The description term slices at its own match index. The
   SPEC's description is byte-accurate, including the consumption **order**,
   which matters because the status regexp runs against the already-sliced
   post-cookie content.

3. **Checkability moved to the criteria, where it belongs.** The requirement
   is definitional; the falsifiers are the four span-assertion corpus rows
   (plan.md:L658-661) and AC-ML-001f/g/h/i. I ran each of the two new ones
   against go-org v1.9.1 and confirmed the correct composition passes and the
   wrong one fails:

   ```
   AC-ML-001h  correct  "1. [@5] {{c1::Tokyo}}\n2. {{c1::Osaka}}\n"
                     => <ol><li value="5">{{c1::Tokyo}}</li><li>{{c1::Osaka}}</li></ol>   no ":[@"  PASS
               wrong    "1. {{c1::[@5] Tokyo}}\n2. {{c1::Osaka}}\n"
                     => <ol><li value="5">:[@5] Tokyo}}</li>…                              has ":[@"  FAIL
   ```

   So the criterion is not vacuous and the required rejection signature
   discriminates.

What the author gave up is an enumeration that a later go-org silently
invalidates. What the author kept is the same enumeration, relabelled
informative and pinned to v1.9.1 (spec.md:L257). An implementer therefore has
both the concrete list and a rule that outlives it. The trade is not a
checkable claim for an unimplementable one; it is a checkable claim plus a
rule, with the checking relocated to the layer that can actually run.

### N2 — the qualification holds, and its boundaries are measured

AC-ML-009f (acceptance.md:L343-355) asserts the block's rendered interior is
byte-identical with and without the collapse. **It can fail.** Measured
against the M1 corpus row `#+BEGIN_SRC text\nx\n\n\ny\n#+END_SRC\n\n- A\n`
with a naive collapse that does not skip blocks:

```
A (no collapse) : <div class="src src-text">…<pre>\nx\n\n\ny\n</pre>…
B (naive collapse): <div class="src src-text">…<pre>\nx\n\ny\n</pre>…
identical = false
```

The three boundary shapes the lead named:

| Shape | Result |
|---|---|
| Blank run straddling the closing line (`x\n\n#+END_SRC\n\n\n- A`) | Rendering identical with and without the skip. A boundary off-by-one is harmless here. |
| Blank run ending at `#+END_` / starting at `#+BEGIN_` | Both identical. Harmless. |
| Nested block (`#+BEGIN_QUOTE` wrapping `#+BEGIN_SRC` with an internal run) | Collapse **does** change the rendered interior, so nesting must be handled. A depth counter handles it; a boolean toggle would exit early at `#+END_SRC`. Measured separately: a blank-line run inside `#+BEGIN_QUOTE` but outside any inner block renders identically either way, so the early-exit variant has no observable consequence. |
| **Unterminated block** | Diverges. This is NEW-1 below. |

### N3 — the sentence determines the case rather than describing it

REQ-ML-006.1's added paragraph (spec.md:L347-356) reads "This rule
**governs** where it meets REQ-ML-005.2: per-answer numbering binds only the
spans composition actually wraps." That is a precedence assignment with a
scope restriction, not a restatement: it names which requirement yields, and
it converts REQ-ML-005.2's one-card-per-answer clause from an unconditional
obligation into a consequence scoped to the wrapped set. The all-pre-marked
case therefore has exactly one specified outcome.

AC-ML-006e (acceptance.md:L230-243) builds the fixture the conflict needs —
`- {{c1::Tokyo}}\n- {{c1::Osaka}}\n` with incremental **on**, two answers
sharing `c1` — and asserts nothing is wrapped, both markers keep `1`, and the
entry is not skipped. A composer that resolved the conflict the other way
renumbers to `c1`/`c2` and fails the second clause. The criterion discriminates.

This closure is **reasoning, not measurement**: it is requirement algebra, and
nothing I ran bears on it. That matches how iteration 2 filed it.

### N5 — the reclassification is upheld, and the trap is now exercised

**Adjudication: the author is right on the facts, iteration 2 was right to
class the truncation pre-existing, and folding N5 into the definitional rule
does remove the trap the v0.1.1 wording created.**

I re-measured all three truncation cases the lead cited, plus the
composition variants that matter:

```
- Tokyo [ ] is big                 => <li class="unchecked">o [ ] is big</li>
- Tokyo :: a [ ] b                 => <dt class="unchecked">o</dt><dd>a [ ] b</dd>
1. [@5] [ ] {{c1::A}}              => <li value="5" class="unchecked">{{c1::A}}</li>
```

The first line is the status quo: the item already renders lossily today,
without this SPEC. That is what makes the truncation pre-existing and the
optional classification correct.

The trap is the second half of the author's argument, and it is real. The
v0.1.1 phrase "structural prefix the parser reads ahead of that content"
describes a leading token. There is none here. An implementer searching for
one finds nothing and wraps whole:

```
wrong (leading-token search)  "- {{c1::Tokyo [ ] is big}}\n"
                           => <li class="unchecked">::Tokyo [ ] is big}}</li>
correct (parser mirror, offset 4) "- Toky{{c1::o [ ] is big}}\n"
                           => <li class="unchecked">{{c1::o [ ] is big}}</li>
```

AC-ML-001i (acceptance.md:L99-111) requires the rendered output to contain
`{{c1::` and not contain `::Tokyo`. The wrong composition fails both halves;
the correct one passes both. So the criterion exercises exactly the trap, and
the explicit prohibition at spec.md:L244-245 ("shall **not** determine it by
searching the item's text for a known token") closes the wording that invited
it. Folding N5 into N1's rule rather than adding a fourth sub-clause is the
right shape: the two findings are one defect seen from two sides.

### Corpus coverage

The M1 corpus (plan.md:L645-663) is **14 table rows carrying 15 shapes** —
the `- A\n\n- B\n` / `- A\n\n\n- B\n` row carries two. That matches the
"fifteen corpus shapes" AC-ML-001's Decides line claims (acceptance.md:L121)
and the "fifteen are required members" claim at plan.md:L646. Span assertions
went from two to four (plan.md:L665-666).

**The two new span rows are the ones that matter.** They are precisely the
shapes the count-equality cross-check cannot catch and the existing `::[`
signature does not reach: the counter cookie (whose corruption signature
`:[@` differs by one character) and the position-blind row (which has no
leading token at all and is the row that fails a token-searching
implementation). The four span rows now cover every shape any iteration has
found: checkbox, mixed-description, cookie, position-blind.

I verified every corpus row's stated discriminator against go-org v1.9.1. All
twelve I re-ran render as the plan claims, including the three block rows, the
`1) `/`a.` ordered terminators, the `- A\n+ B\n` merge, and the indented-first-
bullet row.

### N4's row and criterion

Edge Cases row at acceptance.md:L574 and AC-ML-009g at L356-368 pin the
four-item merge of two adjacent author-written lists. Measured:

```
"- A\n- B\n\n\n- C\n- D\n"      => <ul>A B</ul><ul>C D</ul>   (two lists)
collapsed                        => <ul>A B C D</ul>            (one list)
```

The merge is real and the criterion asserts the post-collapse shape, so a
collapse that did not reach this case fails it. § 5's exclusion (spec.md:L588-594)
names the cost in prose and points at the criterion. Both halves present.

### The delegation observation belongs in `plan.md` § E

**Confirmed, and the placement is correct.** plan.md:L616-625 records it as a
run-phase review note with the reason a criterion was not added: REQ-ML-010.1
requires the new entry point to call the existing one, and asserting an
internal call would test the implementation rather than the behavior. The
observable contract is byte-identity, which AC-ML-013e already holds. A
criterion here would be the over-engineering M6 exists to brake. A review note
is the right instrument for a property that is real but not observable at the
behavior boundary.

### § 8 and traceability

§ 8 (spec.md:L694-712) records the SPEC-ANKICARD-002 dependency as **process,
not code availability**, states every consumed surface is present at `6f3ae6c`,
and proceeds on the operator's recorded ordering directive. It introduces **no
run-phase gate** on SPEC-ANKICARD-002. Confirmed as the lead asked.
Traceability is complete in both directions (see the Category Scores table).
`status: draft`, counts 15/15.

---

## Defects Found (structured defect-list)

Three new findings. **All three are classed optional under M6**, and none
affects the verdict. Each is measured unless marked otherwise.

**NEW-1. REQ-ML-009.2's block skip leaves "block interior" undefined for an
unterminated `#+BEGIN_`, and the natural implementation silently declines to
collapse the rest of the body** — `spec.md:L398-405` / `plan.md:L147-152` —
Severity: **minor** — Class: **optional**.

This **is** the iteration-1-D10 pattern the lead asked me to hunt: the N2 fix
reintroduces the shape N2's own ancestor was closing. I found it and measured
it. The discriminator that keeps it optional is stated below.

go-org backtracks on an unterminated block and treats the `#+BEGIN_` line as
plain text, so the bullets that follow **are** parsed as a list:

```
IN : "#+BEGIN_SRC text\nx\n\n\ny\n\n- A\n- B\n"
OUT: <p>#+BEGIN_SRC text\nx</p><p>\ny</p><ul><li>A</li><li>B</li></ul>
```

A line-scanning skip with a depth counter — the natural reading of "skipping
block interiors" — sees `#+BEGIN_SRC`, never sees a matching `#+END_`, and
treats everything after it as interior. The collapse then never runs:

```
body "#+BEGIN_SRC text\nx\n\n- A\n\n\n- B\n"
  raw                     first list = 1 item   (B is dropped)
  collapse, skip blocks   first list = 1 item   (B is still dropped)
  collapse, no skip       first list = 2 items  (B survives)

body "#+BEGIN_EXAMPLE\nx\n\n- A\n\n\n- B\n"    same three outcomes
```

The corresponding terminated body collapses correctly under the same skip
(`#+BEGIN_SRC text\nx\n#+END_SRC\n\n- A\n\n\n- B\n` → 2 items), so the skip
itself is sound; only the unterminated case diverges.

**Why optional, stated as a discriminator rather than a hedge.** The test that
separated iteration 2's blocking N2 from this one is: *does the v0.1.2 edit
make any input render worse than it renders today?* N2 did — a code sample was
reformatted with no diagnostic. NEW-1 does not: the raw and skip-blocks rows
above are identical at 1 item, so nothing regresses. This is a failure to
deliver the collapse's benefit on malformed input, not a corruption of valid
input. Under M6 that is optional.

Two things the run-phase reviewer should carry:

- The SPEC's own N1 move is the fix. "Block interior" wants the same treatment
  the answer-span offset got: derive it from the parser's behavior — an
  unterminated `#+BEGIN_` opens no block, because the parser backtracks —
  rather than from a line-scan token count.
- **The DD-1 cross-check is not blind to this, but only if the corpus carries
  the row.** On an unterminated-block body a depth-counter scanner finds 0
  top-level items where the parse finds 1, so the count-equality property
  fails and forces the scanner fix; REQ-ML-009.2's "the same exclusion
  REQ-ML-001's scanner observes" then pulls the collapse along. The corpus's
  three block rows are all terminated, so the row does not exist today. One
  row added at run-phase M1 closes it without a plan-phase edit.

**Required fix**: none at plan phase. If taken: one clause in REQ-ML-009.2
stating that a `#+BEGIN_` line with no matching `#+END_` opens no block
interior, plus one M1 corpus row.

**NEW-2. Two informative claims in REQ-ML-001.3 are measurably inaccurate**
— `spec.md:L260-261` and `spec.md:L269-273` — Severity: **minor** —
Class: **optional**.

Both are in prose the SPEC itself labels informative, and both err in the
direction that cannot corrupt a card.

*Claim A* (spec.md:L260-261): the cookie "is consumed on every ordered
terminator (`1.`, `1)`, `a.`)". The parser gates consumption on
`l.Kind == "ordered"` (list.go:L88), and `listKind` returns `"descriptive"`
whenever the list's **first** item matches the description regexp — even with
an ordered terminator:

```
IN : "1. [@5] Tokyo :: big\n"    OUT: <dl><dt>[@5] Tokyo</dt><dd>big</dd></dl>   cookie NOT consumed
IN : "1. [@5] Tokyo\n2. Osaka :: big\n"
                                 OUT: <ol><li value="5">Tokyo</li><li>Osaka :: big</li></ol>  consumed
```

*Claim B* (spec.md:L271-273): "the parser classifies list kind per list rather
than per item, so that shape is reachable from a single `::` written anywhere
in the list." The first half is right; "anywhere" is not. `parseList`
(list.go:L65) computes `list.Kind` from the first token only — the second
sample above shows a `::` in item 2 leaving the list ordered.

**Why optional.** Both errors are safe-direction. Claim A over-predicts
consumption, so an implementer following it skips *more* bytes than the parser
will, landing the marker opening after the parser's content start rather than
before it — measured intact:

```
per informative list  "1. [@5] A :: {{c1::B}}\n2. [@9] {{c1::C}}\n"
                   => <dl><dt>[@5] A</dt><dd>{{c1::B}}</dd><dt>?</dt><dd>[@9] {{c1::C}}</dd></dl>
per parser           "1. [@5] A :: {{c1::B}}\n2. {{c1::[@9] C}}\n"
                   => <dl>…<dd>{{c1::[@9] C}}</dd></dl>
```

Both keep the marker; they differ only in whether `[@9]` sits inside the blank.
And the mechanism that consumes Claim B is immune: DD-1 (plan.md:L164-166)
reads description-kind from the AST's `DescriptiveListItem` type, not by
searching for `::`. The conclusion Claim B supports — that a plain sibling in a
description-kind list is reachable — is true; only "anywhere" overstates it.
The corpus row `- term :: def\n- plain\n` puts the `::` in the first item, so
the corpus is correct as written.

**Required fix**: none required. If taken: change "on every ordered
terminator" to "on a list the parser classifies as ordered", and "anywhere in
the list" to "in the list's first item".

**NEW-3. No corpus row or criterion exercises the position-blind slice on
multi-byte content, and the natural rune-safe instinct corrupts the marker** —
`plan.md:L661` / `acceptance.md:L99-111` — Severity: **minor** —
Class: **optional**.

The parser's slice is a **byte** slice, which spec.md:L250-252 states
correctly. On CJK content it cuts mid-rune, and the composition offset
inherits that. Measured on `- 서울특별시 [ ] 큼` (23 bytes, 11 runes):

| Composer's offset | Rendered output | Marker opening survives |
|---|---|---|
| raw, no composition | `<li class="unchecked">\xef\xbf\xbd\xef\xbf\xbd특별시 [ ] 큼</li>` | n/a — invalid UTF-8 today |
| 4 bytes (correct parser mirror) | `<li class="unchecked">{{c1::…특별시 [ ] 큼}}</li>` | **yes** |
| 6 bytes (rounded **up** to a rune boundary) | `<li class="unchecked">…{{c1::특별시 [ ] 큼}}</li>` | **yes** |
| 3 bytes (rounded **down** to a rune boundary) | `<li class="unchecked">{c1::울특별시 [ ] 큼}}</li>` | **no — `{` consumed** |
| 0 (leading-token search) | `<li class="unchecked">::서울특별시 [ ] 큼}}</li>` | **no** |

The invalid UTF-8 in the raw row is pre-existing go-org behavior, the same
class as N5, and not this SPEC's to fix. What is new is the third row: a Go
implementer who declines to split a rune and rounds the offset **down** — a
natural instinct, and the only rune-rounding direction that is wrong — ships a
destroyed marker. AC-ML-001i's fixture is ASCII (`- Tokyo [ ] is big`), so it
passes while this fails. For a project whose Org content is Korean, this is
the more likely instance of the shape.

**Required fix**: none required. If taken: one M1 corpus row with a multi-byte
first rune and a mid-item `[ ] `, and one clause in AC-ML-001i requiring the
offset to be computed in bytes rather than runes.

---

## Regression Check (iteration 3)

Checked for **reopening only**, per the delta scope. Iteration 2 confirmed all
twelve iteration-1 findings closed; that confirmation is cited, not re-derived.

Findings whose text v0.1.2 touched:

| Prior finding | Touched by | Still closed? | Evidence |
|---|---|---|---|
| D1, D3 — checkbox and plain-sibling span corruption | REQ-ML-001.3 rewritten | **YES** | The definitional rule (spec.md:L239-245) covers both by construction, and the informative list still names the status and the description term (spec.md:L262-267). AC-ML-001f (acceptance.md:L68) and AC-ML-001g (L80) are present and unchanged, with their `::[` and split-marker signatures intact. Re-measured: `- [ ] {{c1::Tokyo}}\n- [X] {{c1::Osaka}}\n` renders with both markers intact and both classes preserved. |
| D2 — hand-written marker must not be wrapped | REQ-ML-006.1 gained the precedence paragraph | **YES** | The not-wrap lead clause is intact at spec.md:L336-345; the precedence text is appended at L347-356 and restricts REQ-ML-005.2, never REQ-ML-006.1. AC-ML-006a and AC-ML-006d unchanged. |
| D10 — removed block's gap silently drops answers | REQ-ML-009.2 gained the block-skip qualification | **YES for valid input** | The collapse obligation is intact (spec.md:L398); the skip is a qualification, not a removal. AC-ML-009d (the block-between-answers case) and AC-ML-009e (the paragraph case) are both present and still form the falsifiable pair. The unterminated-block case is NEW-1 above, classed optional because it does not regress past the status quo. |
| D6 — REQ-ML-006.3 qualified to incremental-off | untouched by v0.1.2 | **YES** | Present at spec.md:L358-363, qualification intact. |
| D8 — editor helper exposed as one callable | untouched | **YES** | REQ-ML-007.3 present at spec.md:L376-382. |
| D9 — REQ-ML-006 pattern label | untouched | **YES** | `[Ubiquitous — negated]` at spec.md:L333. |
| D4, D5, D7, D11, D12 | untouched by v0.1.2 | **YES** | No v0.1.2 edit reaches their text; iteration 2's closure stands. The D7 dissent the r2 report upheld is unreversed — HISTORY at spec.md:L26-27 states no design decision is reversed, and DD-1 through DD-7 read as at v0.1.1. |

**No stagnation.** No finding appears unchanged across all three iterations.

**No new instance of the D10 pattern that ships a worse card.** NEW-1 is the
pattern recurring in a strictly weaker form, and the discriminator is recorded
with it rather than assumed.

---

## Could Not Verify

- **Whether the `#+BEGIN_EXTRA` split can leave a dangling `#+BEGIN_` or
  `#+END_` in the remaining body.** That split belongs to SPEC-ANKICARD-002
  and backlog card t12. If it can, NEW-1's trigger is reachable from
  well-formed author input rather than only from malformed input, which would
  raise its severity. I did not read the split's implementation, so this is an
  open question, not a claim.
- **`#+BEGIN_EXAMPLE` for AC-ML-009f.** I measured the `#+BEGIN_SRC` half of
  the criterion directly. The `#+BEGIN_EXAMPLE` half I take from iteration 2's
  report, which states it confirmed for `EXAMPLE` too; I did not re-run it.
- **N3's closure is reasoning, not measurement.** Requirement algebra decides
  it. Nothing I ran bears on whether the precedence sentence resolves the
  conflict — only on whether AC-ML-006e's fixture can discriminate, which is
  itself a reading of the criterion rather than a run.
- **Whether v0.1.1 carried NEW-2's Claim B verbatim.** `.moai/specs/` is
  untracked, so there is no prior revision to diff against. I cannot say
  whether the "anywhere in the list" phrasing is new in v0.1.2 or inherited.
  It is inaccurate either way.
- **Run-phase quality gates.** I ran no build, test, lint, or format command.
  The baseline figures in the lead's brief are taken as given and were not
  re-measured; § A.2 and the Quality Gate table are not verified by this audit.
- **The Emacs Lisp surface.** REQ-ML-007.3 and REQ-ML-013 name editor
  functions I did not read. Their criteria are judged as written.

---

## Mechanical vs Reasoning Ledger

**Confirmed by probe** (go-org v1.9.1, resolved from the project's `go.mod`,
run in an isolated scratch module outside the checkout; no git command was run
and no file in the working tree was read, modified, or staged beyond the SPEC
artifacts and the rule files):

- The vendored parser's three consumptions, their order, their unanchored
  regexps, and the fixed-byte slicing — read from `org/list.go` L32-33, L88-99,
  and L63-79, not inferred.
- All four v0.1.2 criteria are falsifiable: AC-ML-001h, AC-ML-001i,
  AC-ML-009f, AC-ML-009g. For each I produced the correct composition and the
  wrong one and showed the rendered outputs differ on the asserted signature.
- Every DD-3 rendering claim (plan.md:L272-306), including
  `1. [@5] [ ] {{c1::A}}` and `- Tokyo :: a [ ] b`.
- Twelve of the fifteen M1 corpus shapes, each rendering as its stated
  discriminator claims.
- NEW-1's three-way collapse comparison on both `#+BEGIN_SRC` and
  `#+BEGIN_EXAMPLE`, and the terminated control.
- NEW-2's two counterexamples and the safe-direction check.
- NEW-3's five-offset table.
- The N2 boundary shapes: unterminated, nested, straddling, and both
  block-edge variants.

**Reached by reasoning, not by probe:**

- N3's closure (requirement algebra).
- The classification of NEW-1, NEW-2, and NEW-3 as optional. The measurements
  are facts; the severity judgement is mine.
- That the delegation note belongs in § E rather than as a criterion.
- That the DD-1 cross-check would catch a depth-counter scanner given an
  unterminated-block corpus row — the counts follow from the parser behavior I
  measured, but I did not build the scanner to observe the disagreement.
- The frontmatter, section-presence, traceability, and count checks were done
  with `grep`, which is mechanical; reading them as satisfying the schema is
  judgement.

---

## Recommendation

**PASS at 0.923, above the Tier M threshold of 0.80. The SPEC is ready for the
Implementation Kickoff Approval gate.**

Rationale per must-pass, each with evidence above: MP-1 fifteen contiguous
REQs; MP-2 every requirement carries one GEARS trigger and the four touched
requirements keep their labels; MP-3 all twelve frontmatter fields with
correct types and no rejected alias; MP-4 single-language scope; MP-5 both
referenced SPECs exist and neither is retired; MP-6 no `syscall`; MP-7 no
clarification marker and no `research.md` required at Tier M.

The three blocking findings are closed with criteria I measured as falsifiable
rather than merely present. N1's definitional substitution is the stronger fix,
not a euphemism — the SPEC names the parser, states its mechanism byte-
accurately, prohibits the reading that caused the defect, and relocates the
checking to four span assertions that discriminate. N5's reclassification is
upheld on the author's facts, which I re-measured.

**For the run-phase reviewer**, two residual risks worth carrying, neither
blocking:

1. **NEW-1** — decide what "block interior" means for an unterminated
   `#+BEGIN_` before writing the collapse. Mirroring the parser (which
   backtracks to plain text) is the reading that preserves REQ-ML-009.2's
   benefit. One M1 corpus row makes the cross-check catch a wrong choice.
2. **NEW-3** — compute the answer offset in **bytes**, never in runes. Rounding
   down to a rune boundary destroys the marker on Korean content, and no
   current criterion exercises it. Given this project's Org content, add a
   multi-byte corpus row at M1.

NEW-2's two prose corrections can ride along with any future edit; nothing
depends on them.

---

Reported by: plan-auditor (iteration 3 of 3, delta scope)
Artifacts audited: `.moai/specs/SPEC-ANKICARD-003/{spec,plan,acceptance}.md` at v0.1.2
Parser evidence: `github.com/niklasfasching/go-org v1.9.1` per `go.mod`
