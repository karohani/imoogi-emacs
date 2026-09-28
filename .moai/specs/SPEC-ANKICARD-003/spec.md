---
id: SPEC-ANKICARD-003
title: "Multiline question-and-answer card rendering"
version: "0.1.4"
status: in-progress
created: 2026-09-20
updated: 2026-09-20
author: jay
priority: P2
phase: "v0.2.0 target"
module: "internal/anki/orgdoc, internal/anki/planner, modules/org/anki"
lifecycle: spec-anchored
tags: "anki, org-mode, cloze, multiline, rendering, card-options"
tier: M
depends_on: [SPEC-ANKICARD-002]
---

## HISTORY

### v0.1.4 (2026-09-20)

Answers three findings raised by the **run phase**, which completed with
fourteen of fifteen criteria passing. All three touch plan-phase artifacts and
are therefore amended here rather than recorded in the implementer's notes.
Counts unchanged at 15 and 15. No requirement's obligation changes and no
design decision is revisited; the one requirement edit is a
**verification-method** note, not a new obligation.

- CHANGED: `plan.md` § A.3 **narrows** its PRESERVE entry on
  `internal/anki/planner/baseline_golden_test.go`. The entry preserved the
  file wholesale, but one test inside it went stale **by this SPEC's own
  design**: `TestHashIsUnchangedByCardOptions` builds a probe entry carrying
  `direction` and `incremental` over a body with no list, and expects it to
  reach the add branch — which REQ-ML-002 now correctly rejects with
  `multiline_answer_missing`. Its own helper's docstring names itself "the ONE
  place `TestHashIsUnchangedByCardOptions` needs to change as the wire
  contract grows", so amending it is the change that helper was written for,
  not an erosion of the preservation intent. The entry now names the three
  assertions that must stay byte-identical and states that the probe helper is
  amendable.
- CHANGED: REQ-ML-013.2 gains a **verification-method** note (run-phase
  finding). `org-entry-get` cannot read the falsy spelling back: a drawer
  plainly carrying `:ANKI_DIRECTION: nil` returns Lisp `nil` through it, with
  and without inheritance, indistinguishable from an absent property. The
  production path is unaffected — `imoogi-props--own-value` reads the drawer
  text by regexp and returns the string — so this is a hazard for whoever
  hand-verifies the behavior, not a defect in it. The note names the two
  accessors that do read it back.
- CHANGED: the M1 corpus count corrected in `plan.md` § F and
  `acceptance.md` AC-ML-001 (run-phase finding). Both said "seventeen"; the
  table carries **sixteen rows** and **eighteen shapes**, two rows carrying two
  shapes each. All eighteen are implemented. Cosmetic, but a SPEC should not
  state a number its own artifact contradicts.
- ADDED: `acceptance.md` Edge Cases rows for a **dangling** `#+END_EXTRA`,
  which the run phase measured and which closes the severity question v0.1.3
  left open. An orphan END opens no block interior, so the collapse never
  declines on it; what it does instead is render as visible literal text that
  splits the answer list as an ordinary paragraph would.
- RECORDED, not fixed: `imoogi-anki-set-direction`'s interactive default
  cannot distinguish a cleared value from a never-set one, for the same
  `org-entry-get` reason. No requirement or criterion constrains the prompt's
  default, and changing it is a UX decision rather than a repair, so it is
  stated in REQ-ML-013.2's note as a limitation.

### v0.1.3 (2026-09-20)

Answers plan audit iteration 3
(`.moai/reports/plan-audit/SPEC-ANKICARD-003-plan-audit-r3.md`, verdict **PASS
0.923**). All three findings are optional and none affected the verdict; all
three are folded in anyway, because the M1 corpus is what the implementer
builds against and a shape left out of it is a shape that will not be tested.
Counts unchanged at 15 and 15. No design decision changes.

- CHANGED: REQ-ML-001.3 states that the offset is a **byte** offset, and the
  M1 corpus gains a multi-byte row (audit NEW-3). This is the one finding of
  the three that is not cosmetic in this project, whose own notes are Korean.
  The parser removes a fixed **byte** count, so on multi-byte content the cut
  lands mid-character: measured, `- 서{{c1::울특별시 [ ] 큼}}` loses the
  marker's opening brace, and `- 서울특별시 [ ] 큼` renders **invalid UTF-8**
  outright. An ASCII-only corpus exercises the single case this repository
  does not have.
- CHANGED: two measured inaccuracies corrected in REQ-ML-001.3's informative
  prose (audit NEW-2). The counter cookie is gated on the parser's **list
  kind** rather than on the terminator — measured, `1. [@5] Tokyo :: big`
  leaves `[@5]` in the term, because one `::` makes the list descriptive and
  the cookie is consumed only for an ordered-kind list. And list kind is
  decided from the list's **first item**, not from a `::` appearing anywhere
  in it: `- Plain` followed by `- Term :: Def` stays an ordinary list. Both
  errors were in the direction of over-claiming consumption, so neither
  weakened the obligation; the rule itself is unchanged.
- CHANGED: REQ-ML-009.2 states what the collapse does when a `#+BEGIN_` block
  is never terminated (audit NEW-1). Measured, go-org logs a parse error and
  treats the opening line as plain text, so the blank-line run survives and
  the answer list stays split. The collapse declines to act rather than
  guessing where the block ends; nothing renders worse than it does today, and
  the behavior is now stated rather than left to the implementer.
- UNCHANGED: every requirement's obligation, the 15/15 counts, the tier, the
  § 6 constraints, and DD-1 through DD-7.

### v0.1.2 (2026-09-20)

Answers plan audit iteration 2
(`.moai/reports/plan-audit/SPEC-ANKICARD-003-plan-audit-r2.md`, verdict FAIL
0.706). All three blocking findings closed; **both** optional findings taken.
Counts unchanged at 15 and 15. The audit confirmed all twelve iteration-1
findings closed and upheld the D7 dissent; nothing here reverses a design
decision.

- CHANGED: REQ-ML-001.3 drops its **exhaustiveness claim** and restates the
  rule as **parser-derived** rather than as an enumeration of shapes (audit
  N1, and N5 folded in). The audit found a fourth consumed prefix — the
  ordered-list counter cookie `[@N]` — and the v0.1.1 text asserted three
  shapes exhausted the cases, which is what an implementer codes against. The
  required fix was to enumerate four; this SPEC instead makes the enumeration
  **informative and version-pinned** and the obligation definitional, so a
  fifth prefix in a later go-org needs no fifth edit.
- CHANGED: the same sub-clause records that consumption is **position-blind**
  for the cookie and the status (audit N5, optional, taken and treated as
  load-bearing rather than decorative). Read directly from the vendored
  parser: both regexps are unanchored, and the parser then slices a **fixed
  byte count from position zero** regardless of where the match occurred. An
  implementation that searches for a *leading* token therefore finds none in
  `- Tokyo [ ] is big` and wraps whole, producing a corrupted card. The
  "structural prefix" wording invited exactly that reading, so leaving it
  unqualified would have shipped a hazard the SPEC's own model created.
- CHANGED: REQ-ML-006.1 gains a **precedence** sentence (audit N3). The
  not-wrap rule governs, and REQ-ML-005.2's per-answer numbering binds only
  the spans composition actually wraps — so the one-card-per-answer
  consequence is scoped to them, and an entry whose answers are all pre-marked
  under one shared number yields one card rather than silently failing an
  unmet obligation.
- CHANGED: REQ-ML-009.2's collapse is **qualified to skip block interiors**
  (audit N2, introduced by the v0.1.1 fix for D10). Unqualified, it stripped
  blank lines from a `#+BEGIN_SRC` or `#+BEGIN_EXAMPLE` block in the question
  context — a silent content change, which is the same failure shape D10 was
  filed against. DD-1 already required the scanner to skip block interiors;
  the collapse is a separate step and inherited nothing.
- CHANGED: § 5's later-lists exclusion notes the deliberately-authored
  two-list body the collapse merges (audit N4, optional, taken), and
  `acceptance.md` gains the Edge Cases row and sub-criterion that make it
  checkable rather than merely disclosed.
- UNCHANGED: every requirement's obligation apart from those named above, the
  15/15 counts, the tier, the § 6 constraints, and every design decision —
  DD-1 through DD-7 all stand as written in v0.1.1.

### v0.1.1 (2026-09-20)

Answers plan audit iteration 1
(`.moai/reports/plan-audit/SPEC-ANKICARD-003-plan-audit.md`, verdict FAIL
0.63 against the Tier M threshold of 0.80). All ten blocking findings closed;
both optional findings taken. Requirement and criterion **counts are
unchanged** at 15 and 15 — every edit below sharpens, corrects, or bounds an
existing entry, or adds a sub-clause or sub-criterion under one. The spine
decision, DD-1's pre-render source-level composition, is unchanged; the audit
confirmed all twelve of its go-org claims byte-for-byte and asked for none of
it back.

- CHANGED: REQ-ML-001.3 is **generalized** from the description-list rule into
  one **answer-content** rule covering all three item shapes (audit D1, D3).
  The audit measured that wrapping a checkbox item whole destroys the marker's
  opening — go-org reads the status before the content — and that a plain
  sibling inside a description-kind list has no definition half to wrap. Both
  reproduce in this SPEC's own re-probe. One rule replaces what would
  otherwise be three ad-hoc ones: the wrapped span is the item's content less
  whatever structural prefix the parser reads ahead of it.
- CHANGED: REQ-ML-006 gains sub-clause **1**, stating that a span already
  carrying a hand-written marker is **not wrapped** (audit D2). This resolves
  a genuine contradiction rather than softening a criterion: the former
  AC-ML-006a wanted a hand-written marker to survive byte-for-byte inside a
  generated marker, while REQ-ML-007.1 separates exactly those braces, and
  exempting them does not rescue it because Anki's non-greedy close then ends
  the outer marker at the inner one. Nesting is impossible in Anki, so the
  only coherent rule is not to wrap. The former sub-clauses renumber to **2**
  and **3**; the offset rule is unchanged for the spans that are wrapped.
- CHANGED: REQ-ML-006.3 (formerly .2) is qualified to the incremental-**off**
  case (audit D6). Unqualified, it fixed the title at `c2` while REQ-ML-005.3
  forces `c3` under incremental `<->` — a contradiction that was unpinned in
  either direction.
- CHANGED: REQ-ML-006's pattern label corrected to `[Ubiquitous — negated]`
  (audit D9, optional, taken). Its lead clause is a `shall not`, the form
  § 3's preamble distinguishes.
- CHANGED: REQ-ML-007 gains sub-clause **3**, requiring the editor to expose
  separation-and-pad as one callable helper (audit D8). The pad lives in
  `imoogi-anki-cloze-region`, not in `imoogi-anki--cloze-safe-text`, so the
  shared-fixture criterion previously named a function that cannot produce the
  form it asserts.
- CHANGED: REQ-ML-009.1 now **collapses** the blank-line gap a removed
  supplementary block leaves, rather than documenting the answer loss it
  caused (audit D10). `plan.md` DD-4 records the reversal and why the
  objection that sank the other two fixes does not reach this one.
- CHANGED: REQ-ML-010.1 requires the new entry point to **delegate** to the
  existing one for a non-multiline entry (audit D7, strengthened). The audit
  asked for a criterion; delegation makes the byte-identity structural, and
  the criterion then guards the delegation.
- CHANGED: REQ-ML-002.1's rationale narrowed to the empty-body case (audit
  D11, optional, taken). A lead paragraph with no list is a visible cue, so
  the former sentence overreached on the shape it most often describes.
- CHANGED: § 5's later-lists exclusion restated, since REQ-ML-009.1's collapse
  means a blank-line run no longer separates one authored list into two.
- UNCHANGED: every requirement's obligation apart from those named above, the
  15/15 counts, the tier, the § 6 constraints, and DD-1, DD-2, DD-3's
  definition-half decision, DD-6, and DD-7.

### v0.1.0 (2026-09-20)

- INITIAL: SPEC created from backlog card **t14** of the design record
  `.moai/reports/anki-card-types-plan-20260920.md` (§ 2.2 generation rules,
  § 2.3 decisions B and F, § 3 work split). Tier M, 3 plan-phase artifacts
  plus `progress.md`.
- Consumes the transport and validation SPEC-ANKICARD-002 landed at `6f3ae6c`
  and renders the first card kind those options select: a heading title as the
  question, its top-level body list as the answers, composed into one
  `imoogi-Cloze` note whose Text field carries both.
- Backlog card **t15** (swift arrow cards) is a sibling, not a part of this
  SPEC. § 5 names the exclusion.

## 1. Overview

### 1.1 Purpose

SPEC-ANKICARD-002 carried three card-option properties from an Org drawer to a
Go-side validation gate and deliberately rendered nothing. This SPEC is the
first renderer that consumes them.

Concretely it delivers one thing: a heading carrying a multiline option becomes
a question-and-answer card. The heading title is the question, the top-level
list in its body holds the answers, and the two are composed into a single
`imoogi-Cloze` note whose Text field carries cloze markers placed according to
the resolved direction.

### 1.2 Scope

In scope: identifying the answer list in a body; composing title and answers
into one cloze-marked fragment per direction and per incremental setting; the
`multiline_answer_missing` diagnostic; the answer-list container class and its
stylesheet rule; passing options into the renderer on both call sites; the
editor commands that write `ANKI_DIRECTION` and `ANKI_INCREMENTAL`.

Out of scope: swift arrow cards, every option SPEC-ANKICARD-002 already
validates, and the three rejections § 5 records. § 5 enumerates the exclusions.

### 1.3 Reading order

`spec.md` (this file) → `acceptance.md` (the Given-When-Then verification
layer) → `plan.md` (the module-by-module change plan, carrying the design
decisions and the go-org probe evidence behind them). The design record
`.moai/reports/anki-card-types-plan-20260920.md` § 2.2 is the source of the
confirmed generation rules and stands in for a Tier L `research.md`.

## 2. Glossary

**Multiline option** — `ANKI_DIRECTION` naming one of its three arrows, or
`ANKI_INCREMENTAL` in its truthy spelling. Exactly the two properties
SPEC-ANKICARD-002 groups as the multiline pair; `ANKI_SWIFT` is the other
group and selects a different card kind.

**Multiline entry** — a sync target for which at least one multiline option is
on. This is SPEC-ANKICARD-002's **option-bearing entry** narrowed to the
multiline pair: an entry bearing only `ANKI_SWIFT` is option-bearing but is not
a multiline entry, and an entry whose multiline options are all in their falsy
spelling is neither.

**Remaining body** — what is left of a heading's body after the supplementary
`#+BEGIN_EXTRA` blocks have been removed. This is the existing split
(`internal/anki/orgdoc/extra.go`), unchanged here; every rule in this SPEC that
reads "the body" reads the remaining body.

**Answer list** — the first list the renderer's own parser recognizes at the
top level of the remaining body. § 3.1 fixes what "top level" and "first" mean.

**Answer item** — one item of the answer list, at the list's own base
indentation. An item nested under another item is not an answer item.

**Generated marker** — a cloze marker this SPEC's composition step writes into
the fragment. Distinct from a **hand-written marker**, which the user typed in
the Org buffer (usually via `imoogi-anki-cloze-region`). Both are ordinary
`{{cN::…}}` markers once composed; the distinction matters only for numbering
(§ 3.2 REQ-ML-006).

**Composition** — the step that reads a multiline entry's title and remaining
body and produces the single Org source fragment the renderer then renders.
Composition happens before rendering, on Org source; § 3.3 fixes its place in
the pipeline.

## 3. Requirements (GEARS)

Fifteen requirements across four groups, `REQ-ML-001`..`REQ-ML-015`,
contiguous. The `ML` infix keeps them clear of SPEC-ANKICARD-002's `REQ-OPT-`
namespace, SPEC-ANKICARD-001's `REQ-C-`, and the parent SPEC's `REQ-0NN` — all
three of which are cited from live code comments in this repository.

Each requirement carries exactly one GEARS trigger. Numbered sub-clauses are
case-splits on that trigger's operand and never introduce a second `When`,
`While`, or `Where`. `[Ubiquitous — negated]` is the `shall not` form of
Ubiquitous, not a separate pattern.

**What this SPEC's renderer may assume.** SPEC-ANKICARD-002's validation gate
runs **before** the render on both the ordinary path and the migration path,
and already rejects a malformed option value, a swift-with-multiline conflict,
and an option on a non-cloze note type. Every entry reaching this SPEC's
composition step has therefore passed all three rules: its option values are
recognized, its note type is cloze-style, and `ANKI_SWIFT` is not on beside a
multiline option. Those three rejections are **not** re-specified here, and no
requirement below restates them.

### 3.1 Identifying the answers

#### REQ-ML-001 [Where]
Where an entry is a multiline entry, the back end shall take its answers from
the **answer list** — the first list its own parser recognizes at the top level
of the remaining body:

1. An **answer item** shall be an item at that list's own base indentation. An
   item nested under another item shall not be an answer item, and its content
   shall render as it does today.
2. Both the unordered and the ordered list forms shall count, since both are
   ordinary Org lists and an author numbering their answers has not thereby
   declined to be asked about them.
3. The span an answer item contributes shall be its **answer content** — the
   text beginning at the offset where the renderer's own parser begins that
   item's content, after everything that parser consumes for the item's
   structure. The obligation is **definitional, not an enumeration**: the
   composer shall derive the offset from the parser's own rule for the
   vendored parser version, and shall **not** determine it by searching the
   item's text for a known token.

   That prohibition is the load-bearing half. Read directly from the vendored
   parser, two of the three consumptions are **position-blind**: the
   expressions matching the counter cookie and the status are unanchored, and
   the parser then removes a **fixed byte count from the start of the item**
   regardless of where the match occurred. An item reading `- Tokyo [ ] is
   big` therefore loses its first four bytes today, with no leading token
   present anywhere. A composer looking for a leading token finds none, wraps
   the whole item, and ships a corrupted card. Only the description term is
   position-aware, splitting at its own match.

   The offset is a **byte** offset, and composition shall treat it as one.
   Rounding it to a rune boundary is not a safe simplification: the parser
   does not round, so a composer that did would place its marker at a
   different offset than the parser starts content at — the exact failure this
   sub-clause exists to prevent. On multi-byte content the difference is not
   subtle. Measured, `- 서{{c1::울특별시 [ ] 큼}}` loses the marker's opening
   brace, because the four bytes removed are three bytes of `서` plus one of
   `{`; and `- 서울특별시 [ ] 큼` renders **invalid UTF-8**, because the cut
   lands inside a character. The corpus shall therefore carry multi-byte
   content, this project's notes being Korean.

   As of go-org v1.9.1 the parser consumes three things, in this order, and
   this list is **informative** — a later version may consume a fourth, which
   the definitional rule above already covers:
   - The **counter cookie** (`[@5]`), consumed only when the parser's own list
     **kind** is ordered. The gate is the kind, not the terminator: measured,
     `1. [@5] Tokyo :: big` leaves `[@5]` in the term, because one `::` makes
     the list descriptive. On an ordered-kind list it is consumed for every
     terminator (`1.`, `1)`, `a.`), and never on an unordered one.
   - The **status** (`- [ ] answer`, `- [X] answer`, `- [-] answer`), on any
     list kind, including a description item's term.
   - The **description term** and its separator, leaving the **definition
     half** as the answer content. The form already states which half is the
     answer.

   An item from which the parser consumes none of these — including a plain
   item sitting in a description-kind list — contributes its whole content. A
   plain sibling has no definition half to take, and the parser classifies
   kind per **list** rather than per item, deciding it from the list's
   **first** item: measured, `- Term :: Def` followed by `- Plain` is a
   description list carrying a plain sibling, while `- Plain` followed by
   `- Term :: Def` is an ordinary list in which the `::` is just text. The
   sibling shape is reachable whenever the first item carries a `::`.

   One rule covers all of them because every failure is the same failure: a
   generated marker whose opening lands before the offset the parser starts
   content at is destroyed by that parser. `plan.md` DD-3 carries the measured
   evidence per shape.
4. Only the **first** such list shall supply answers. A later list in the same
   body shall render as ordinary body content.
5. Content preceding the answer list — a lead paragraph, a block — shall render
   as question context, outside every generated marker.

#### REQ-ML-002 [Where]
Where a multiline entry's remaining body yields no answer item, the back end
shall reject that entry with the diagnostic code `multiline_answer_missing`:

1. The obligation shall hold for **every** direction, including `<-`. On an
   entry whose remaining body is empty, a leftward card hides the only content
   it has and leaves a prompt with no visible cue — a worse card than the
   rightward case, not an exempt one. Where the body carries prose but no
   list the card would have a cue, and the entry is still rejected: a card
   asking a question it states no answer to is not a card, and admitting one
   direction here would make the diagnostic's meaning depend on which arrow
   was written.
2. The rejected entry shall be reported as skipped, carrying its existing note
   identifier where it has one, and shall leave the collection, the registry,
   and the Org heading untouched — the same treatment every other skip
   diagnostic already receives.

### 3.2 Composing the card

#### REQ-ML-003 [Ubiquitous]
The resolved direction shall select what composition wraps in a generated
marker:

1. `->` shall wrap each answer item and shall leave the title unwrapped.
2. `<-` shall wrap the title and shall leave the answer items unwrapped.
3. `<->` shall wrap both.

#### REQ-ML-004 [Ubiquitous]
A multiline entry whose direction did not resolve to an arrow shall compose as
though `->` had been written:

1. The obligation shall bind both the absent case and the explicitly falsy
   case, since SPEC-ANKICARD-002 recognizes the falsy spelling on
   `ANKI_DIRECTION` as "no direction" rather than as an error.
2. This is what makes `ANKI_INCREMENTAL` usable on its own — it is the reading
   SPEC-ANKICARD-002 § 5 deferred to this card, and it is settled here as the
   design record's own default rather than as a new one.

#### REQ-ML-005 [Ubiquitous]
The incremental setting shall select how many distinct numbers the answers
carry:

1. Off, every answer item shall share **one** number, so the entry yields one
   card that hides all the answers together.
2. On, each answer item shall carry **its own** number, assigned in document
   order, so the entry yields one card per answer.
3. Where the direction is `<->`, the title's number shall be distinct from
   every answer's.

#### REQ-ML-006 [Ubiquitous — negated]
Composition shall not disturb a hand-written marker already present in the
title or the remaining body:

1. A span that already carries a hand-written marker shall **not be wrapped**.
   The rule binds an answer item and the title alike, and the unwrapped span
   shall reach the renderer byte-for-byte as the author wrote it. Anki's cloze
   pattern closes at the first `}}` after an opening marker, so a generated
   marker placed around a hand-written one is ended by it — the two cannot
   nest, and wrapping anyway would destroy the author's existing card. An
   entry whose every answer item carries a hand-written marker is therefore
   composed with nothing wrapped; it is **not** rejected, because its answers
   are present and already clozed.

   This rule **governs** where it meets REQ-ML-005.2: per-answer numbering
   binds only the spans composition actually wraps. An entry whose answers are
   all pre-marked under one shared number therefore yields one card, and that
   is the specified outcome rather than an unmet obligation — composition
   cannot renumber an author's marker without destroying the card that marker
   already makes, which is the whole reason this rule exists. The
   one-card-per-answer consequence REQ-ML-005.2 states is scoped to the
   wrapped spans accordingly.
2. Generated numbering, for the spans that **are** wrapped, shall begin above
   the highest number any hand-written marker carries, so a generated marker
   can never collide with one the author wrote elsewhere in the entry.
3. Where no hand-written marker is present — the ordinary case — and the
   incremental setting is **off**, the generated numbers shall be exactly
   those the design record § 2.2 fixes: `c1` for the answers and `c2` for the
   title. With incremental on, the answers consume consecutive numbers and
   REQ-ML-005.3 places the title above them, so no fixed pair applies.

#### REQ-ML-007 [Ubiquitous]
Composition shall keep a generated marker from closing early:

1. A run of two or more consecutive `}` characters inside wrapped content shall
   be separated by a single space each, and wrapped content ending in `}` shall
   carry one space before the closing `}}`.
2. The rule shall be the one the editor command already applies
   (`imoogi-anki--cloze-safe-text` and its pad), for the same reason: Anki's
   cloze pattern is non-greedy and closes at the first `}}` after the opening
   marker. Without it, an answer item carrying LaTeX produces a card that ends
   its blank early and shows the remainder literally.
3. The editor shall expose separation and pad together as one callable helper,
   so that the two implementations of this rule can be compared against one
   shared fixture. Today the separation lives in
   `imoogi-anki--cloze-safe-text` and the pad in its caller, so no single
   existing function produces the form this requirement describes, and a test
   asserting the rule would have to reimplement half of it — which is the
   drift the shared fixture exists to prevent.

#### REQ-ML-008 [Where]
Where a multiline entry's direction is `<-` and its incremental setting is on,
the incremental setting shall have no effect and shall raise no diagnostic.
`<-` wraps only the title, so there is nothing for per-item numbering to
number. Reporting it would cost a diagnostic code to describe a combination
that produces a correct card.

### 3.3 Pipeline placement

#### REQ-ML-009 [Ubiquitous]
Composition shall sit between the supplementary-content split and the
cloze-marker gate, in this fixed order — split, compose, gate, render:

1. Composition shall read the **remaining body**, so supplementary content can
   never become an answer item.
2. Composition shall collapse a run of consecutive blank lines in the
   remaining body to a single blank line before identifying the answer list,
   **skipping block interiors** — the same exclusion REQ-ML-001's scanner
   observes, and for the same reason. A blank-line run inside a `#+BEGIN_SRC`
   or `#+BEGIN_EXAMPLE` block is content the author wrote, and collapsing it
   reformats a code sample with no diagnostic and no report signal. That is a
   silent content change of exactly the kind this sub-clause exists to
   prevent, so the collapse must not reach inside a block.

   Where a `#+BEGIN_` line is never terminated, the extent of the block is
   undefined and the collapse shall **decline to act** on the run rather than
   guess where the block ends. The consequence is that the answer list stays
   split, which is the behavior of the tree today: measured, go-org logs a
   parse error, treats the opening line as plain text, and renders the two
   lists separately. Declining costs a benefit on malformed input; guessing
   would risk the silent content change this sub-clause forbids, on input the
   parser itself could not read.
   Removing a supplementary block leaves a gap wide enough to separate one
   authored list into two, and without the collapse every answer after the
   block is silently dropped from the card. The collapse binds **composition
   only** — it runs for a multiline entry and for no other, so it changes
   nothing about what any other entry renders to. `plan.md` DD-4 carries the
   measurement and records that this reverses that decision's earlier
   disposition.
3. A generated marker shall **satisfy** the cloze-marker gate, so a multiline
   entry carrying no hand-written marker shall render rather than be skipped
   with `cloze_marker_missing`.
4. Sub-clause 3 does not amend SPEC-ANKICARD-002. Its REQ-OPT-012.3 binds an
   entry "carrying no cloze marker in its title or remaining body" and states
   in the same breath that that SPEC "introduces no marker the options
   generate". This SPEC introduces exactly such a marker, which is the case
   that wording carved out — § 4 records the relationship.

#### REQ-ML-010 [Ubiquitous]
The card options shall reach the renderer on **both** call sites, through an
entry point that takes them:

1. The existing entry point shall keep its current parameters and its current
   output for every note type, so the byte-identity assertion
   SPEC-ANKICARD-002 established continues to compile and to pass unchanged.
   The new entry point shall **delegate** to it for an entry that is not a
   multiline entry, rather than reimplement its behavior. Delegation makes
   REQ-ML-014.1 structural instead of merely tested: a second rendering path
   for the no-option case would be free to drift from the first, and the
   corpus that would catch the drift exercises only the path no production
   caller takes once both call sites move.
2. The migration path shall pass the options too. A stock-`Cloze` multiline
   entry is valid, and a migration that rendered it without its options would
   produce a Text field the ordinary path then immediately rewrites — a
   spurious update on the very next synchronization. This is the re-examination
   SPEC-ANKICARD-002 § 5 assigned to the first rendering card.

#### REQ-ML-011 [Ubiquitous]
`multiline_answer_missing` shall be reported only for an entry that already
passed SPEC-ANKICARD-002's validation gate, so it shall never compete with that
gate's three codes for the one-diagnostic-per-entry slot. It is fourth by
construction rather than by a fourth entry in the gate's fixed order: the gate
runs before the render, and this condition is detectable only during
composition.

### 3.4 Presentation, front end, and non-interference

#### REQ-ML-012 [Ubiquitous]
The answer list shall render inside a container the card stylesheet can target,
and the stylesheet shall carry a rule for it:

1. The container shall be identified by the class name `children-list`, on
   whichever element the answer list renders as — the unordered, the ordered,
   and the description form alike. A container carried by only some of the
   three would leave REQ-ML-001.2 and REQ-ML-001.3 rendering unstyled cards.
2. The rule shall be added to the imoogi-owned base stylesheet, which the
   install step owns outright, and shall name no network-hosted resource and
   derive nothing from a deck name — the properties that stylesheet already
   asserts mechanically.

#### REQ-ML-013 [Ubiquitous]
The editor shall gain commands that write the two multiline properties, reached
from the existing Anki transient:

1. Setting a direction shall offer the three arrows and an explicit clear, and
   setting the incremental option shall be a toggle. Direction is a three-value
   choice and incremental is a boolean; one prompt carrying both would ask
   about incremental on every direction edit, which is the more common lone
   change.
2. Turning an inherited option **off** shall write the falsy spelling into the
   heading's own drawer rather than delete the property, because all three
   card-option properties inherit and deletion would let an ancestor's value
   apply again.

   **Verification method.** `org-entry-get` shall not be used to check that
   this happened. Measured under `emacs --batch -Q`: a drawer plainly carrying
   `:ANKI_DIRECTION: nil` returns Lisp `nil` through `org-entry-get`, with and
   without inheritance, which is indistinguishable from the property being
   absent — a hand-verifier reading it would record a false negative on a
   correct write. Two accessors do read the value back: this project's own
   `imoogi-props-resolve-*`, whose `imoogi-props--own-value` reads the drawer
   text by regexp, and Org's `org-entry-properties`, which returns the pair
   `("ANKI_DIRECTION" . "nil")`. The production path is unaffected, so this
   binds verification only.

   **Stated limitation, not a defect.** For the same reason, the direction
   command's interactive default cannot distinguish a value the user cleared
   from one never set. Nothing in this SPEC constrains the prompt's default,
   and changing it would be a user-experience decision rather than a repair,
   so it is recorded here rather than treated as work.
3. Where the heading carries no `ANKI_NOTE_TYPE`, the command shall write
   `imoogi-Cloze`; where it carries a cloze-style type, the command shall leave
   it; where it carries any other type, the command shall report it and change
   nothing. This is the contract `imoogi-anki-cloze-region` already follows.
4. Where the heading resolves `ANKI_SWIFT` to its truthy spelling, the command
   shall report the conflict and shall write nothing unless the user confirms
   clearing swift. Writing a direction beside an active swift would create
   exactly the entry SPEC-ANKICARD-002's conflict rule rejects, at the moment
   the user believed they had configured a card.
5. The two property names shall join the editor's property-name completion
   candidates. `ANKI_SWIFT` shall not — it belongs with the command that writes
   it, which is t15's.

#### REQ-ML-014 [Ubiquitous — negated]
The back end shall not change what a **non**-multiline entry renders to:

1. An entry for which no multiline option is on shall produce a byte-identical
   field map, so this SPEC causes no existing note to be reported as updated on
   account of content that did not change.
2. The content hash's input set shall gain no member.
3. The byte-identity corpus SPEC-ANKICARD-002 recorded shall be **extended, not
   replaced**: its recorded values and its signature assertion shall stand
   unmodified, and this SPEC's multiline cases shall be measured by a separate
   corpus. Re-recording the existing corpus to admit new cases would destroy
   the very evidence it exists to carry.

#### REQ-ML-015 [Ubiquitous]
The diagnostic code this SPEC introduces shall be paired with a user-facing
message in the front end's code-to-message table, which remains the only
component that renders prose for a code:

1. The message shall name the problem and state a corrective action, and shall
   contain no stack trace, raw transport error, backtrace, or bare exit code.
2. The existing two-directional pairing assertion shall continue to hold over
   the enlarged code set, including the hard-coded back-end code list it
   cross-checks against.

## 4. Relationship to SPEC-ANKICARD-002

SPEC-ANKICARD-002 is a **dependency, not a parent**. This SPEC amends none of
its requirements and changes no behavior it introduced. Three of its surfaces
are consumed here exactly as it left them, and two of its deferrals are
discharged:

- **Consumed unchanged.** The three resolved option values on the entry, the
  validation gate that runs before the render on both paths, and the recognized
  value sets (`->`/`<-`/`<->` for direction, the truthy and falsy spellings for
  the booleans, all compared after trimming). This SPEC defines no second
  parser for any of them.
- **Discharged deferral — the renderer widening.** Its § 5 named "widening the
  renderer's entry point to accept the card options, on either the
  synchronization or the migration call site" as the first rendering card's
  work, and assigned that card the question of whether the migration path must
  pass options too. REQ-ML-010 answers both: a new entry point takes the
  options, the existing one is untouched, and the migration path passes them.
- **Discharged deferral — incremental alone.** Its § 5 stated no default
  direction for `ANKI_INCREMENTAL` written without `ANKI_DIRECTION`, leaving
  the rendering meaning to this card. REQ-ML-004 settles it as `->`.
- **Carve-out filled, not amended.** Its REQ-OPT-012.3 preserved the
  `cloze_marker_missing` skip for an option-bearing entry with no marker, and
  qualified itself with "This SPEC introduces no marker the options generate".
  This SPEC introduces such markers, so for a multiline entry the gate is
  satisfied by composition (REQ-ML-009.3). Every other entry reaches that gate
  exactly as before.

## 5. Out of Scope

This SPEC renders one card kind and no other. The exclusions below are what
makes that claim checkable: each names work an implementation could plausibly
fold in, and states where it actually belongs.

### Out of Scope — Swift card rendering

- Splitting an entry body's arrow lines into per-line cloze-numbered cards, the
  `swift_arrow_missing` diagnostic, and the swift numbering rule the design
  record § 2.2 records. Backlog card **t15**.
- Recognizing the in-body arrow forms `:->`, `:<-`, and `:<->` at all. This
  SPEC reads arrows only as `ANKI_DIRECTION` property values, never in a body.
- An editor command that writes `ANKI_SWIFT`, and that property's entry in the
  property-name completion candidates. Both belong with t15.

### Out of Scope — Re-specifying the validated rules

- The three rejections SPEC-ANKICARD-002's gate already performs — a malformed
  option value, swift beside a multiline option, an option on a non-cloze note
  type. Composition receives only entries that passed them, and adds no fourth
  rule to that gate.
- Changing the gate's fixed diagnostic order, or its one-diagnostic-per-entry
  rule. REQ-ML-011 places the new code outside that contest rather than inside
  the order.

### Out of Scope — Wrapping an answer item's nested children

- Extending a generated marker over the sub-items nested under an answer item.
  Measured, the naive source-level form leaves the marker spanning the
  intervening markup, which `plan.md` DD-2 rejects with its evidence; the
  design record § 5 independently flags multi-line HTML inside a cloze as
  needing real-Anki verification this SPEC does not perform. An answer item's
  own content is wrapped and its nested children stay visible (REQ-ML-001.1).

### Out of Scope — Treating later lists as answers

- Taking answers from the second and subsequent lists of a body. REQ-ML-001.4
  fixes the answer list as the first one; a later list is ordinary content. A
  card wanting two answer groups is two headings.
- Note that REQ-ML-009.2's blank-line collapse means a run of blank lines no
  longer separates one authored list into two, so this exclusion now bites
  only where genuine content — a paragraph, a block — sits between the lists.
  That is the case where treating the second list as answers would wrap text
  the author wrote as commentary.
- The collapse takes something with it, and the cost is named rather than
  left to be discovered: a blank-line run is the only way Org lets an author
  write two adjacent lists of the same bullet with no prose between them, so
  an author who did that deliberately now gets one merged answer list. The
  trade is deliberate — the merged case is visible on the card, where the
  dropped-answer case it replaces was silent — and `acceptance.md` AC-ML-009g
  pins it so it cannot change by accident.

### Out of Scope — Replacement-cloze and image-occlusion card kinds

- The regular-expression-driven replacement cloze and the image-occlusion card
  kind the design record § 2.2 records as deferred and excluded respectively.
  Neither has a card-option property, here or in SPEC-ANKICARD-002.

### Out of Scope — A per-answer note

- Producing one Anki note per answer item. Incremental produces one note with
  several numbered blanks, which Anki renders as several cards; the design
  record § 2.3 decision C fixed the heading-to-note-identifier relation at one
  to one, and a per-answer note would break it.

## 6. Constraints

1. **No new note type.** The card is an `imoogi-Cloze` note. Nothing here
   creates, updates, or names a new Anki model.
2. **Composition is source-level and pre-render.** The wrapping happens on Org
   source before the renderer runs, never on rendered HTML. `plan.md` DD-1
   carries the measured justification; the constraint is stated here because
   several requirements read as satisfiable either way and only one way works.
3. **The back end validates and renders; it never mutates.** No code path here
   writes a property or edits an Org file. Properties are written at edit time
   by the commands of REQ-ML-013.
4. **One diagnostic per skipped entry.** REQ-ML-011 keeps the new code outside
   SPEC-ANKICARD-002's fixed order rather than adding a fourth position to it.
5. **Byte-identical rendering for everyone else.** REQ-ML-014 is the constraint
   this SPEC is shaped around: an entry with no multiline option must produce
   the hash it produced before, so no user sees a mass of spurious updates.

## 7. Brownfield Delta

Per-surface disposition. `[NEW]` marks an artifact this SPEC creates;
`[CHANGED]` an existing one it modifies; `[UNCHANGED]` one it deliberately
leaves alone despite sitting on the path.

### [NEW] Multiline composition (`internal/anki/orgdoc`)

The answer-list identification of § 3.1 and the composition rules of § 3.2, in
a new file beside the existing supplementary-content split. Carries the typed
answer-missing error REQ-ML-002 names.

### [CHANGED] Renderer entry point (`internal/anki/orgdoc`)

Gains the option-taking entry point of REQ-ML-010 and the pipeline order of
REQ-ML-009. The existing entry point keeps its parameters and its behavior
(REQ-ML-014.1) and remains the path every non-multiline entry takes.

### [CHANGED] Protocol (`internal/anki/protocol`)

The diagnostic-code const block gains `multiline_answer_missing`. No field
changes shape; the wire contract's version is unchanged, since this SPEC adds
no field in either direction.

### [CHANGED] Planner (`internal/anki/planner`)

Both call sites pass the options to the new entry point (REQ-ML-010.2) and map
the typed answer-missing error onto the new code, mirroring the existing
cloze-marker mapping. The validation gate, the field-resolution layer, the deck
fallback, and the add/update/no-op partition are untouched.

### [UNCHANGED] Content hashing (`internal/anki/hashing`)

Deliberately untouched. Its input set is note type, rendered field values,
resolved deck, and sorted tags; a multiline entry changes the first of those
through the rendered value alone, which is what REQ-ML-014.2 asserts.

### [CHANGED] Card stylesheet (`internal/anki/model/assets/base.css`)

Gains the `children-list` rule of REQ-ML-012.

### [CHANGED] Editor commands (`modules/org/24-anki.el`)

Gains the direction and incremental commands of REQ-ML-013, their key bindings,
their transient entries, and the two new property-name completion candidates.

### [CHANGED] Diagnostics (`modules/org/anki/imoogi-error.el`)

Gains one code-to-message entry (REQ-ML-015).

### [UNCHANGED] Property resolution and scan (`modules/org/anki`)

Deliberately untouched. SPEC-ANKICARD-002 already resolves all three properties
through the nearest-wins chain and attaches them to the entry; this SPEC reads
what that work produces and adds no resolution of its own.

### [CHANGED] Tests

A new multiline corpus separate from the existing byte-identity corpus
(REQ-ML-014.3), composition and answer-identification tests, the planner's
diagnostic-mapping tests, the code-pairing tests including the hard-coded
back-end code list, and ERT coverage for the new commands.

### [CHANGED] Documentation (`README.md`)

The Anki section gains the multiline card: what the two properties do, how the
answers are read from the body, and what the four direction-and-incremental
combinations produce.

## 8. Dependencies

- **SPEC-ANKICARD-002** — landed on `main` at commit `6f3ae6c`. Every surface
  this SPEC consumes — the three entry fields, the validation gate on both
  paths, the recognized value sets, the front-end resolvers, and the
  code-pairing contract — is in that commit.

  Its frontmatter reads `in-progress`, so the run-phase dependency pre-flight
  will find it unfulfilled. Proceeding is authorized on the same footing
  SPEC-ANKICARD-002 itself recorded for its own dependency: the design record's
  ordering note has the operator, told the predecessor was still in flight,
  directing that cards t11 through t15 be worked in order — a range naming this
  card — and the directive was repeated in the session that produced this SPEC.
  The dependency is process, not code availability.

  The consequence is recorded rather than mitigated: SPEC-ANKICARD-002's own
  closure will run against a HEAD this SPEC has moved.
- **Backlog card t12** — already landed. Its supplementary-field split is the
  step REQ-ML-009.1 composes after, and its byte-identity corpus is the one
  REQ-ML-014.3 extends rather than replaces.
- No new third-party dependency. The renderer keeps go-org at its current
  version; `plan.md` DD-1 and DD-3 record the probes run against it.

## 9. Traceability

Every requirement is covered by at least one criterion in `acceptance.md`;
every criterion names the requirements it verifies and the command that decides
it. `plan.md` § F maps requirements onto milestones.
