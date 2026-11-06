---
id: SPEC-ANKICARD-004
title: "Swift arrow card rendering"
version: "0.1.4"
status: completed
created: 2026-09-21
updated: 2026-10-13
author: jay
priority: P2
phase: "v0.2.0 target"
module: "internal/anki/orgdoc, internal/anki/planner, modules/org/anki"
lifecycle: spec-anchored
tags: "anki, org-mode, cloze, swift, arrow, rendering, card-options"
tier: M
depends_on: [SPEC-ANKICARD-003, SPEC-ANKICARD-005]
---

## HISTORY

### v0.1.4 (2026-10-12)

Closes plan audit iteration 3
(`.moai/reports/plan-audit/SPEC-ANKICARD-004-plan-audit-r3.md`, **FAIL 0.75**,
iteration cap reached). This revision is the targeted fix that the escalation's
option 2 describes. The delta-only re-audit it names is still pending
authorization. All four blocking findings are closed, and the optional one is
taken. Counts are unchanged at 14 requirements and 15 criteria. The corpus
grows from 38 rows to 42. Every new row and every rule restated below was
measured against go-org v1.9.1 from the module cache, in a scratch module
outside the checkout.

- **R3-1 — closed.** REQ-SW-001.6 no longer says the context-dependent
  constructs at v1.9.1 "are the three". The set is stated as derived by rule,
  with an informative list. `plan.md` DD-4's "exactly" and M1's "the three"
  are struck. The fourth construct the audit measured is now mirrored: the
  **result attached to a source block**. go-org attaches the construct that
  follows a `#+RESULTS:` line to a preceding `#+BEGIN_SRC` block, with only
  blank lines allowed between them. Its writer then suppresses that construct
  under `:exports code` or `:exports none`. The rule is written as the parser's
  and the writer's own, including two edges measured here. A blank line
  directly after `#+RESULTS:` does not protect the arrow line (the attached
  paragraph starts on the blank line), while two blank lines do. A `:RESULTS:`
  drawer is suppressed whole. The derivation claim is softened: the list was
  read from the parser's dispatch, a case can govern lines beyond its own
  delimiters, and that is how the earlier reading missed this one. Rows 39
  (`:exports code` → 0) and 40 (default exports → 1).
- **R3-2 — closed.** Three mirrored rules are restated as go-org's own rules.
  (a) The **footnote definition**'s blank-pair end is tested only outside a
  nested `#+BEGIN_` block or LaTeX environment, because those run to their own
  close. The opening is ASCII-labelled, at column zero, and followed by
  whitespace or the end of the line. One deliberate deviation from the audit's
  wording is measured, not missed: the audit listed *lists* among the
  constructs that carry the definition past a blank pair. Measured, a list does
  not, and neither does an ordinary drawer: with a blank pair inside either,
  the arrow line below renders in place. The code agrees, because `parseList`
  and `parseDrawer` honour the definition's stop while `parseBlock` and
  `parseLatexBlock` do not. Lists are therefore not named. (b) The **property
  drawer** suppresses only when every interior line is a property line and a
  closing `:END:` is reached with no blank line before it. Otherwise its lines
  render normally. (c) The LaTeX environment's `\begin` and `\end` are
  case-insensitive, as well as its name. § 5's "wholly nested shapes agree" is
  narrowed to the nestings actually measured. Rows 41 (footnote, then a source
  block holding a blank pair → 0) and 42 (a `:PROPERTIES:` drawer with a
  non-property line → 1).
- **R3-3 — closed.** REQ-SW-013.4's mirror conflict check binds the **on**
  direction only, as sub-clause 3's note-type contract already did. Turning
  swift off writes the falsy `ANKI_SWIFT` and touches no multiline option.
  Mirrored in `plan.md` DD-10 and M5. AC-SW-014 gains the OFF case, unstubbed,
  and its both-on case now states the ON direction.
- **R3-4 — closed.** AC-SW-008's first clause no longer makes the
  counterfactual claim about a field. It now observes the checked split
  directly: its `extra` output carries the arrow line, and its `rest` output
  carries none.
- **O-2 — taken.** REQ-SW-001.6 now fixes how the mirrored rules compose. Block
  and environment regions are computed first, over the whole body. The footnote
  and source-block-result rules then run over those regions. The excluded set
  is the union. This reuses `blockRegions` unchanged. § 5's interleaving entry
  is restated for that composition: the two interleavings it previously
  described as agreeing or not diverging now over-exclude, which withholds a
  card and never composes a marker that is hidden. The quote-first interleaving
  it previously disclosed as diverging now agrees.
- UNCHANGED: the 14/15 counts, the tier, and every requirement's obligation
  other than those named above.

### v0.1.3 (2026-10-12)

Completes the closure of plan audit iteration 2
(`.moai/reports/plan-audit/SPEC-ANKICARD-004-plan-audit-r2.md`, **FAIL 0.75**)
for audit iteration 3. v0.1.2 began that closure and was never audited; read
against the six findings and against the tree at `7d98ec0`, it left a residue on
four of them, contradicted itself across the four files on one rule, and
predated SPEC-ANKICARD-005. This entry names the closure of each finding and what
v0.1.3 changed to reach it. Counts unchanged at 14 requirements and 15 criteria;
the corpus grows from 32 rows to 38.

- **N1 — closed.** v0.1.2 struck the closed four-shape list and added the
  fixed-width line and the footnote definition to REQ-SW-001.5's informative
  list; both stand. Two residues are fixed here. REQ-SW-001.6 had kept a
  `shall` on the per-line parse, which is one measured route rather than the
  obligation; it now requires the classification to come from the parser's
  **classification rules**, and the per-line parse is recorded in `plan.md`
  DD-4 as one route. And the context-dependent half was itself a closed list of
  three — the same defect one layer down. Measured at go-org v1.9.1, two more
  constructs make a later line something other than in-place paragraph text: a
  **footnote definition continues** across the lines below it until a second
  consecutive blank line (`[fn:1] 메모`⏎`도쿄 :-> 일본` renders to nothing), and
  a **`:PROPERTIES:` drawer written mid-body** suppresses its interior. Both are
  now in the informative list, the context-dependent half is stated as a rule
  (mirror the parser's opening-and-continuation rule for every such construct)
  rather than as a set, and REQ-SW-001.5's head names the property the classes
  share — a paragraph rendered **in place**.
- **N1 — the LaTeX environment rule is corrected.** v0.1.2 said the
  environment's interior "shall be identified from the parser's own whole-body
  classification", which needs the source positions go-org's AST does not carry
  (the route the audit said the SPEC must not imply), while its own HISTORY
  entry and `plan.md` DD-4 said the pairing was widened. Measured, go-org's
  rule is line-anchored (`\begin{NAME}` alone on a line, closed by the next
  `\end{NAME}` with the same name compared case-insensitively, an unmatched
  opening opening nothing) — the same shape as the `#+BEGIN_` rule the existing
  scanner already mirrors. The scanner now mirrors it, and all four files say so.
  One measured divergence is disclosed rather than claimed away: delimiters of
  the two forms that **interleave** rather than nest.
- **N2 — closed.** v0.1.2 added § 5's inline-construct exclusion and qualified
  Constraint 7. Three wordings still promised more than the predicate delivers
  and are corrected: REQ-SW-011.3's "Inline constructs carry no such risk",
  REQ-SW-001.7's unqualified "element", and REQ-SW-005's head clause, which now
  says the hand-written cloze marker is the one inline construct it protects. The
  glossary's "Arrow line" carried the pre-N1 definition and is restated.
- **N3 — closed.** v0.1.2 corrected all five stale container sites; § 8's
  sentence was left broken by that edit and is repaired.
- **N4 — closed.** v0.1.2 narrowed the claim to three evasions. The helper
  docstring obligation in REQ-SW-012.4 now carries the narrowing as well as the
  three legs, matching `plan.md` DD-5 and AC-SW-013.
- **N5 — closed in `spec.md` by v0.1.2; closed in `acceptance.md` here.**
  REQ-SW-013.3 binds the note-type contract to the ON direction, and the current
  tree supports the choice: `card_option_needs_cloze`'s message in
  `modules/org/anki/imoogi-error.el` tells the user to switch an inherited option
  off by giving it `nil` in the heading's own drawer, which is exactly what the
  OFF path writes. AC-SW-014 lacked the case the finding required — an
  `imoogi-Basic` heading with an inherited swift-on — and gains it.
- **N6 — closed** by v0.1.2's rows 27-29; unchanged.
- CHANGED: **SPEC-ANKICARD-005 propagated.** It landed after this SPEC was
  written: every cloze-style render now runs split, **check**, compose, gate,
  render, and an unpaired `#+BEGIN_EXTRA` marker skips the entry with
  `extra_block_unbalanced` before composition. REQ-SW-008, REQ-SW-010, § 4, § 5's
  supplementary-marker exclusion, § 8, and the edge-case table all described
  the pre-005 behavior and are corrected. It joins `depends_on`; it is
  `completed`.
- CHANGED: the baseline moves from `08f7504` to `7d98ec0`, re-measured rather
  than carried (`plan.md` § A.2): Go coverage is unchanged, and the Elisp suite
  is now 645 tests, so the `≥ 435` threshold was stale.
- ADDED: corpus rows 33-38 — footnote continuation, a footnote ended by two
  blank lines, an unclosed `\begin`, a mid-body property drawer, the one-line
  `\begin{…} … \end{…}`, and an ordinary drawer. The last two were edge-case
  assertions with no corpus body, the shape iteration 2's N6 named.
- CHANGED: every `go test -run` pattern in `acceptance.md` is anchored
  (`^Name$`, `^(A|B)$`). `moai spec lint` reported all fifteen as
  `VacuousTestAssertion`, since an unanchored pattern also selects longer test
  names; the eight pre-existing names were confirmed to exist exactly first.
- UNCHANGED: the 14/15 counts, the tier, DD-3, every requirement's obligation
  other than those named above.

### v0.1.2 (2026-09-21)

Answers plan audit iteration 2
(`.moai/reports/plan-audit/SPEC-ANKICARD-004-plan-audit-r2.md`, verdict **FAIL
0.75**, unchanged from iteration 1, so no STOP signal). All thirteen
iteration-1 findings were confirmed closed and none reopened. All five new
blocking findings closed here; the one optional finding taken. Counts unchanged
at 14 and 15; the corpus grows from 24 rows to 32.

**One fix introduced by the previous iteration, and one failure direction it
opened:**

- CHANGED: REQ-SW-001.6 **strikes the closed enumeration of four shapes** the
  v0.1.1 fix introduced (audit N1). REQ-SW-001.5 forbids "matching a list of
  known line shapes" in normative language, and .6 then prescribed exactly such
  a list — the fix reinstating the failure its own rule exists to prevent.
  Measured, the four were incomplete: a **fixed-width line** (`: A :-> B`)
  renders inside `<pre class="example">`, reaching the block interior's hazard
  without a block, and a **footnote definition** (`[fn:1] A :-> B`) renders to
  nothing at all, reaching the comment line's silent hazard. Both are in
  REQ-SW-001.5's informative list now, and .6 requires the context-free
  classification to derive from the **parser's own node classification**, which
  measured answers correctly for every shape including the two. No replacement
  list is offered.
- CHANGED: REQ-SW-001.5 moves the **LaTeX environment** into the
  context-dependent group — a finding neither the brief nor the audit placed
  correctly. Measured, a multi-line `\begin{align}` environment is a distinct
  node whose interior go-org passes through **raw**, while the interior line
  read on its own is an ordinary paragraph: the per-line classification the
  audit proposed misclassifies it, and the existing `#+BEGIN_`/`#+END_` pairing
  does not see the delimiter. The pairing rule is widened to both forms. The
  one-line `\begin{align} … \end{align}` is genuinely a paragraph and stays
  admitted.
- ADDED: § 5 gains **Out of Scope — Arrows inside an inline construct** (audit
  N2). The definitional rule classifies a **line**; the corruption it must
  prevent is **inline**, so a link, code span, verbatim span, emphasis, an
  author's own export snippet, and an inline footnote are all admitted and all
  split by composition. Six measured outcomes are tabulated. The worst is the
  inline footnote, whose content relocates to the document's end so the closing
  `}}` lands **before** its `{{c1::` and Anki refuses the note. Disclosure is
  the proportionate move: six constructs across five delimiter syntaxes is a
  second grammar, and REQ-SW-001.6 has just struck one enumeration for being
  incomplete. A render-twice round-trip check is recorded as considered and
  rejected.
- CHANGED: **Constraint 7's rationale** is qualified from "no line's enclosing
  element can change" to "no line's **block-level** element can change" (audit
  N2). The unqualified form is literally true and beside the point, and reading
  it as the general claim is what kept the inline class invisible.

**The three smaller findings:**

- CHANGED: five stale sites describing the abandoned container are corrected
  (audit N3) — § 1.2's scope line, § 4, § 8, AC-SW-011's heading, and Residual
  Risk 4. § 4's entry moves from "Consumed unchanged" to a **named non-reuse**
  with its reason, which is the one a reader could have acted on wrongly.
- CHANGED: REQ-SW-012.4 narrows "every way an option could reach the hash" to
  the **three evasions the legs actually exclude** (audit N4). Folding the
  resolved value into one of the four existing arguments passes all three legs;
  the route is low-reachability and is recorded rather than guarded.
- CHANGED: REQ-SW-013.3 binds the note-type contract to the **on** direction
  only (audit N5). Read unqualified, a heading carrying `imoogi-Basic` with an
  inherited swift-on could never opt out through the command; and writing
  `imoogi-Cloze` as the consequence of *declining* an option creates a marker
  the user did not ask for. The shape is inherited from the incremental toggle
  and is resolved here for this command alone.

**The optional finding taken:** three corpus rows for the heading, keyword, and
comment shapes (N6), which AC-SW-001e2 previously asserted in prose only.

**Two candidate findings the audit raised and retracted after measuring** are
recorded as confirmed-sound rather than acted on: the numbering base counts an
unclosed opening, so REQ-SW-005.3 is correct; and the existing toggle calls the
heading guard itself, so DD-10's replacement loses nothing.

- UNCHANGED: the 14/15 counts, the tier, DD-3 (accepted outright by the audit),
  DD-4's rule, DD-5's reshaping, DD-9, and every other requirement's obligation.

### v0.1.1 (2026-09-21)

Answers plan audit iteration 1
(`.moai/reports/plan-audit/SPEC-ANKICARD-004-plan-audit.md`, verdict **FAIL
0.75** against the Tier M threshold of 0.80). All seven blocking findings
closed; all six optional findings taken. Requirement and criterion counts are
unchanged at 14 and 15 — every edit below sharpens, corrects, or bounds an
existing entry, or adds a sub-clause. Every go-org measurement the audit
re-ran reproduced exactly, so no evidence is revised; the defects were in rules
layered on top of the measurements, where a predicate's scope was narrower than
the obligation above it.

**Two design decisions change, not merely their wording:**

- CHANGED: REQ-SW-011 **abandons the container attribute line** for an inline
  line break plus a class on the emphasis element, and gains sub-clause 3
  forbidding composition from inserting any line of its own (audit D12, D3).
  The container was a column-zero insertion, and measured it **terminates a
  list** when the arrow sits on an indented continuation line — a structural
  change to authored content, worse than the silence § 5 discloses. It also had
  no stated end, so a prose line below an arrow line was drawn inside the
  styled container (D3). The inline form has neither property: measured, the
  emphasis and the break leave a list item intact. The break is emitted only
  where the next line is itself an arrow line, which keeps REQ-SW-001.7 true
  unqualified for every non-arrow line. `plan.md` DD-3 is rewritten; the
  stylesheet delta moves from a container class to `swift-arrow`.
- CHANGED: REQ-SW-001.5 **replaces two exclusions with one positive
  definitional rule** — an arrow line must be a line the parser reads as
  ordinary paragraph text (audit D12, and a class the audit did not name). The
  former block-interior and list-item exclusions missed list **continuation**
  lines, which carry no bullet; and this SPEC's own re-probe found three more
  shapes that match the expression and are not paragraph text: a table row
  (measured, `| A :-> B |` is a table whose cell delimiter the right side would
  swallow into a marker), a heading, and a keyword line — plus a comment line,
  which renders to **nothing at all**, so an arrow composed there would produce
  a note whose only markers are invisible. An enumeration would have missed
  them; the definitional form, which SPEC-ANKICARD-003 REQ-ML-001.3 established
  for the same reason, covers them and whatever a later parser adds.

**The remaining blocking findings:**

- CHANGED: REQ-SW-005 gains sub-clause **1**, making a line whose first arrow
  token lies **inside a hand-written marker's span** not an arrow line (audit
  D1, critical). The head clause promised protection the per-side rule could
  not deliver: a marker spanning the arrow leaves each flank holding only a
  fragment, so neither guard fires and one side is wrapped anyway. Measured,
  `{{c1::도쿄 :-> 일본}}` composes to a field whose Anki rendering is one card
  showing `{{c2::일본} }` as literal text — the author's deletion destroyed.
  The clause fixes the span as Anki's own non-greedy extent and extends an
  **unclosed** opening to the end of the body, because a generated marker below
  one supplies the `}}` it lacks and brings it to life. It also states that the
  existing presence predicate cannot decide this and an extent scan is needed.
  The former sub-clauses renumber to **2**, **3**, and **4**.
- CHANGED: REQ-SW-001.3 is restated so the expression and the "first token"
  rule agree (audit D2). Non-greediness minimises the left side but does not
  pin the first token: measured, `:-> B :-> C` puts the first token **in the
  left side** and makes the second one the arrow. Settled toward rejection,
  which matches sub-clause 4's existing rejection of `:-> B`.
- CHANGED: REQ-SW-013.3 stops routing the new command through the existing
  card-option write helper (audit D5). That helper runs the swift-conflict
  check before the note-type check, so every swift **off** would first be asked
  whether to turn swift off, and a declining user could never turn it off at
  all. The note-type helper's failure message also stops naming multiline
  options.
- CHANGED: REQ-SW-013.4's mirror check clears **every** on multiline option
  rather than "it" (audit D9). Both can be on at once.
- CHANGED: REQ-SW-012.4 names the four-argument compile-time assertion as the
  **third leg** of the hash contract and restates the sufficiency claim (audit
  D4). The pair alone admits an implementation that hashes the resolved boolean
  while ignoring the wire field; the trio does not. The audit's judgment that
  the reshaping is right and the two halves individually necessary is accepted
  as written.

**All six optional findings taken:** the corpus count (D6), `plan.md` § A.2's
cross-reference (D7), Constraint 6's overstated scope (D8), the trim's Unicode
definition (D10, measured against a non-breaking space), AC-SW-007's mechanicity
claim (D11), and § 5's "drops … with no signal" wording.

**Accepted with a stated refusal.** The audit accepted the list-item
disposition as a trade and named a third option without requiring it — admit
bullet lines whose computed content offset is zero. § 5 now refuses it on three
stated reasons rather than leaving it unanswered.

- UNCHANGED: the 14/15 counts, the tier, every other requirement's obligation,
  and DD-1, DD-2, DD-4's block reasoning, DD-5's reshaping, DD-6, DD-7, DD-8.

### v0.1.0 (2026-09-21)

- INITIAL: SPEC created from backlog card **t15** of the design record
  `.moai/reports/anki-card-types-plan-20260920.md` (§ 2.2 generation rules,
  § 2.3 decision C, § 3 work split). Tier M, 3 plan-phase artifacts plus
  `progress.md`.
- Closes the four-card series. It renders the second and last card kind the
  options of SPEC-ANKICARD-002 select: a heading marked `ANKI_SWIFT` whose body
  carries arrow lines, composed into one `imoogi-Cloze` note carrying one card
  per arrow side.
- Consumes SPEC-ANKICARD-003's composition machinery rather than duplicating
  it: the option-taking entry point, the block-interior scan, the bullet
  grammar, the brace-safety rule, and the hand-written-marker convention are
  all reused as that SPEC left them at `08f7504`. § 4 records the relationship.
- Discharges the two deferrals SPEC-ANKICARD-003 § 5 assigned to this card: the
  editor command that writes `ANKI_SWIFT`, and that property's entry in the
  property-name completion candidates.

## 1. Overview

### 1.1 Purpose

`ANKI_SWIFT` has been carried from an Org drawer to a Go-side validation gate
since SPEC-ANKICARD-002, and read by nothing since. This SPEC is the renderer
that consumes it, and it is the last one: once it lands, every card-option
property has a rendering meaning.

Concretely it delivers one thing. A heading marked `ANKI_SWIFT: t` has its body
read line by line for an arrow. Each arrow line becomes a pair of cloze-numbered
sides, so one heading yields one `imoogi-Cloze` note carrying as many cards as
its arrow lines earn. The heading title stays unwrapped above them as context.

### 1.2 Scope

In scope: recognizing the three in-body arrow forms `:->`, `:<-`, and `:<->`;
the classification that decides which lines can carry them (§ 3.1);
the per-line numbering rule; the arrow's emphasis; the `swift_arrow_missing`
diagnostic; the inline line break between consecutive arrow lines; the arrow's
own class and its stylesheet rule; extending the option-taking entry point's
dispatch; the editor command that writes
`ANKI_SWIFT` and that property's completion candidacy.

Out of scope: multiline card rendering, every option SPEC-ANKICARD-002 already
validates, and the seven exclusions § 5 records.

### 1.3 Reading order

`spec.md` (this file) → `acceptance.md` (the Given-When-Then verification
layer) → `plan.md` (the module-by-module change plan, carrying the design
decisions and the go-org probe evidence behind them). The design record
`.moai/reports/anki-card-types-plan-20260920.md` § 2.2 is the source of the
confirmed generation rules and stands in for a Tier L `research.md`.

## 2. Glossary

**Swift option** — `ANKI_SWIFT` in its truthy spelling. It is the other of
SPEC-ANKICARD-002's two option groups; `ANKI_DIRECTION` and `ANKI_INCREMENTAL`
are the multiline pair and select a different card kind.

**Swift entry** — a sync target for which the swift option is on. This is
SPEC-ANKICARD-002's **option-bearing entry** narrowed to the swift option: an
entry bearing only a multiline option is option-bearing but is not a swift
entry, and an entry whose swift option is in its falsy spelling is neither.

**Remaining body** — what is left of a heading's body after the supplementary
`#+BEGIN_EXTRA` blocks have been removed by the **checked** split
(`internal/anki/orgdoc/extra.go`, unchanged here), which SPEC-ANKICARD-005 made
reject a split that leaves a marker unpaired. Every rule below that reads "the
body" reads the remaining body, exactly as SPEC-ANKICARD-003 REQ-ML-009.1 fixed
it.

**Arrow token** — one of the three literal strings `:->`, `:<-`, `:<->`.

**Arrow line** — a line of the remaining body that § 3.1 admits: a line the
renderer's parser reads as ordinary paragraph text rendered in place
(REQ-SW-001.5), whose first arrow token lies outside every hand-written marker's
span (REQ-SW-005.1), and which matches the arrow expression with a non-empty
side on each flank (REQ-SW-001.1-.4).

**Left side / right side** — the two spans an arrow line contributes, being the
text before and after the arrow token, each with surrounding whitespace
trimmed.

**Generated marker** / **hand-written marker** — as SPEC-ANKICARD-003 § 2
defines them: a marker this SPEC's composition writes, versus one the author
typed in the Org buffer. Both are ordinary `{{cN::…}}` markers once composed;
the distinction matters only for numbering and for what composition may wrap.

**Composition** — the step that reads a swift entry's title and remaining body
and produces the single Org source fragment the renderer then renders.
Composition happens before rendering, on Org source; § 3.3 fixes its place in
the pipeline.

## 3. Requirements (GEARS)

Fourteen requirements across four groups, `REQ-SW-001`..`REQ-SW-014`,
contiguous. The `SW` infix keeps them clear of SPEC-ANKICARD-003's `REQ-ML-`,
SPEC-ANKICARD-002's `REQ-OPT-`, SPEC-ANKICARD-001's `REQ-C-`, and the parent
SPEC's `REQ-0NN` — all of which are cited from live code comments in this
repository.

Each requirement carries exactly one GEARS trigger. Numbered sub-clauses are
case-splits on that trigger's operand and never introduce a second `When`,
`While`, or `Where`. `[Ubiquitous — negated]` is the `shall not` form of
Ubiquitous, not a separate pattern.

**What this SPEC's renderer may assume.** SPEC-ANKICARD-002's validation gate
runs **before** the render on both the ordinary path and the migration path,
and already rejects a malformed option value, a swift-with-multiline conflict,
and an option on a non-cloze note type. Every entry reaching this SPEC's
composition step has therefore passed all three rules: its option values are
recognized, its note type is cloze-style, and no multiline option is on beside
swift. Those three rejections are **not** re-specified here, and no requirement
below restates them. The second of them is load-bearing twice over: because the
gate rejects the combination, a swift entry is never also a multiline entry, so
the two renderers never contend for one entry and their two diagnostics can
never both apply.

### 3.1 Identifying the arrow lines

#### REQ-SW-001 [Where]
Where an entry is a swift entry, the back end shall take its cards from the
**arrow lines** of the remaining body:

1. A line shall be an arrow line where its text matches the expression
   `(.+?)\s*(:<->|:->|:<-)\s*(.+)`, subject to sub-clauses 3 through 6 and to
   REQ-SW-005.1. The alternation shall be ordered longest-first, so `:<->` is
   never read as `:<-` followed by a stray `>`; the ordering is the rule, not
   an incidental property of the expression as written.
2. Detection shall be a **pre-render text pass**, on Org source, before go-org
   runs. This is definitional rather than a performance preference. Measured
   against the vendored go-org v1.9.1, the renderer escapes both angle
   brackets, so after rendering the three tokens read `:-&gt;`, `:&lt;-`, and
   `:&lt;-&gt;` — a post-render scanner would have to match the escaped forms,
   and would be matching text that no longer stands in a known relation to the
   source the marker must be written into. `plan.md` DD-1 carries the
   measurement.
3. The arrow token the expression selects shall be the line's **first** arrow
   token, and a line for which it is not shall **not** be an arrow line.
   Non-greediness minimises the left side; it does not pin the first token,
   because the left group still requires one character. Measured,
   `:-> B :-> C` yields the left side `:-> B` and the **second** token as the
   arrow, so an implementation reading the expression literally and one reading
   "the first token" produce different cards from one line. The rule is settled
   toward rejection: a line whose left side still contains an arrow token is
   not an arrow line, which is also what makes this consistent with
   sub-clause 4's rejection of `:-> B`, the same shape with the tail removed.
   A token appearing **after** the selected one is unaffected: it is right-side
   text and shall render as literal escaped text, carrying no emphasis and
   earning no second pair of numbers, so `A :-> B :-> C` yields the pair `A` /
   `B :-> C` and one line's worth of cards.
4. Each side shall be **trimmed** of Unicode whitespace, and a line whose left
   or right side is empty after trimming shall **not** be an arrow line. The
   expression's two `.+` groups already reject a wholly absent side; the trim
   is what additionally rejects a side that is only whitespace, which the
   greedy `\s*` can otherwise leave behind. The trim shall be Unicode-aware
   rather than ASCII-only, because the expression's own `\s` is ASCII-only and
   the two disagree on characters a Korean input method produces: measured, the
   right side of `A :->` followed by a non-breaking space survives the
   expression and is emptied by a Unicode trim but not by an ASCII one. Without
   this clause an entry could compose a marker around a non-breaking space — a
   blank with no answer behind it.
5. An arrow line shall be a line the renderer's own parser reads as **ordinary
   paragraph text rendered in place** — part of a paragraph the renderer emits
   at the line's own position in the document, not one an enclosing construct
   relocates, suppresses, or passes through raw. The obligation is
   **definitional, not an enumeration**: the composer shall derive the
   classification from the parser's own rules for the vendored parser version,
   and shall **not** determine it by matching a list of known line shapes.

   "Rendered in place" is what makes the definition decide the cases a bare
   "paragraph" does not. go-org builds a paragraph node inside a footnote
   definition and inside a list item too, so "the line is in a paragraph node"
   would admit both; neither is emitted at the line's position. Conversely a
   paragraph inside an ordinary drawer, or one carrying a `#+NAME:`,
   `#+CAPTION:`, or `#+ATTR_HTML:` line above it, is emitted in place as a
   `<p>` — measured — and is admitted. So is one under a **standalone**
   `#+RESULTS:` line, and one under a `#+RESULTS:` attached to a source block
   whose export setting lets the result render. Under `:exports code` or
   `:exports none`, the same paragraph renders to nothing (the source-block
   result bullet below).

   That framing is load-bearing, because the failure it prevents is not a
   rendering blemish but a structural change to the author's document, and the
   shapes that cause it do not resemble one another. As of go-org v1.9.1 the
   classification excludes, and this list is **informative** — a later version
   may read a further shape as something other than paragraph text, which the
   definitional rule above already covers:
   - A **block interior**, and the interior of a **LaTeX environment**.
     Measured, `#+BEGIN_SRC` and `#+BEGIN_EXAMPLE` interiors render inside
     `<pre>`, where a marker's braces show on the card as literal text.
     `#+BEGIN_QUOTE` renders an ordinary paragraph and would in fact carry a
     marker correctly, so the single `#+BEGIN_`/`#+END_` pairing rule
     SPEC-ANKICARD-003's scanner already implements costs the quoted case.
     Reusing that one rule rather than writing a narrower one is deliberate: a
     second pairing rule for the **same** `#+BEGIN_` delimiter in one package
     would diverge invisibly. (One rule per delimiter, as the LaTeX
     environment below receives, is not that hazard: no delimiter is then
     paired by two rules that could disagree about it.)

     A `\begin{…}` / `\end{…}` environment is the **same hazard under a
     different delimiter**, and neither existing mechanism sees it. Measured, a
     multi-line `\begin{align}` environment is a distinct parser node whose
     interior go-org passes through **raw**, so a snippet composed there shows
     as literal characters exactly as in a `#+BEGIN_SRC` interior; and the
     interior line read **on its own** is ordinary paragraph text, so nothing
     line-local catches it either.

     The parser's rule for the environment is line-anchored and has the same
     shape as its `#+BEGIN_` rule: an environment opens on a line holding only
     `\begin{NAME}` (surrounding whitespace allowed), closes at the next line
     holding only `\end{NAME}` with the same name, and an opening with no
     matching close opens nothing — the line renders as plain text. The whole
     line is matched case-insensitively, the `\begin` and `\end` keywords as
     well as the name: measured, `\BEGIN{align}` … `\END{align}` is a raw
     environment. A starred name (`align*`) is a name like any other. The
     interior shall be identified by **mirroring that rule** in the scanner, as
     a second delimiter form beside the `#+BEGIN_` one, exactly as the existing
     scanner mirrors the `#+BEGIN_` rule. That is not a second copy of the
     `#+BEGIN_` rule — the delimiters and the name capture differ — and it needs
     no source position from the parser's tree, which carries none.

     The one-line form `\begin{align} … \end{align}` is genuinely a paragraph
     and is **not** excluded: the anchored opening does not match a line that
     carries anything after the brace.
   - **List content** — a bullet line, and every line the list-continuation
     rule attaches to one. A bullet line is excluded because its content begins
     at an offset the parser computes, which SPEC-ANKICARD-003 REQ-ML-001.3
     makes definitional and entangles with three consumptions; one of those,
     the description term's `::`, competes with the arrow for the same line
     (measured, `- Tokyo :: Japan :-> x` is a description item whose definition
     half already carries the arrow). A **continuation** line is excluded for a
     different reason: measured, `- 수도` followed by an indented
     `도쿄 :-> 일본` is one list item holding two lines, so a card composed on
     the second line would sit inside a list item — exactly the placement § 5
     discloses as excluded, reached by a line that carries no bullet for a
     line-local test to catch. The list-continuation rule is the only thing
     that identifies it.

     Historically this clause also carried a sharper hazard: composition used
     to write a container line at column zero, which measured **terminates the
     list**. REQ-SW-011.3 removed that mechanism, so the hazard no longer
     applies and is recorded here as history rather than as the reason.
   - The **continuation of a footnote definition**. A footnote definition does
     not end at its own line. It opens on a line that starts at column zero
     with `[fn:LABEL]`, where LABEL is a run of ASCII letters, digits, `_`, or
     `-`, followed by whitespace or by the end of the line. Measured, `[fn:메모]
     노트` (a Korean label), `[fn:1]노트` (no space), and ` [fn:1] 노트`
     (indented) are **not** definitions, and the line below each renders in
     place. go-org continues a definition across the lines below it until the
     second of two consecutive blank lines, a headline, or another
     footnote-definition line. That end condition is tested only **between**
     the constructs the definition holds, never inside a `#+BEGIN_` block or a
     LaTeX environment nested in it, because each of those runs to its own
     close whatever blank lines it contains. A list or an ordinary drawer
     nested in it does not carry it past a blank pair.

     Measured: `[fn:1] 메모` followed by `도쿄 :-> 일본` is one footnote
     definition rendering to **nothing** (and, when something references the
     footnote, relocating to the footnotes section at the document's end), even
     with one blank line between the two; with two blank lines between them the
     arrow line is an ordinary paragraph again. A `#+BEGIN_SRC`, a
     `#+BEGIN_QUOTE`, or a `\begin{align}` holding two blank lines, nested
     between the opening and the arrow line, still renders the arrow line to
     nothing. A list or an ordinary drawer holding two blank lines in the same
     position leaves it in place.
   - The interior of a **property drawer**. A line holding only `:PROPERTIES:`
     (name compared case-insensitively, surrounding whitespace allowed) opens a
     property drawer **only when** every line after it, up to the next line
     holding only `:END:` (also case-insensitive), is a property line — `:KEY:`
     or `:KEY: value`, the key a run of non-whitespace characters — with no
     blank line among them, and that `:END:` is reached. Such a drawer renders
     nothing at all. Otherwise the `:PROPERTIES:` line is plain text, nothing
     is suppressed, and every line below it renders by its own classification.
     Measured: a non-property line inside, a blank line inside, or a missing
     `:END:` each leaves `:X: 도쿄 :-> 일본` or `도쿄 :-> 일본` rendering as an
     in-place paragraph. A `:PROPERTIES:` line inside another drawer opens
     nothing either (measured: it renders as text, and so does the property
     line below it). A drawer of any other name renders its interior as
     in-place paragraphs and is admitted. The scan strips the heading's own
     drawers before the body is read, so the property drawer is reachable only
     when written mid-body — reachable all the same, and its interior line read
     on its own is a paragraph.
   - The **result attached to a source block**. When a `#+BEGIN_SRC` block
     closes (the block rule above), go-org looks past any blank lines for a
     line beginning `#+RESULTS:` (case-insensitive, leading whitespace
     allowed). If the first non-blank line is one, the parser attaches it to
     the block as the block's result, together with the **one** construct it
     reads next, starting at the line immediately after `#+RESULTS:`. go-org's
     HTML writer then emits that result only when the block's header does
     **not** carry an `:exports` value of `code` or `none`. Under either value,
     the attached construct renders to nothing, wherever it would otherwise
     have been an in-place paragraph.

     The rule is the parser's and the writer's, and its edges are measured
     rather than assumed:
     - A paragraph that begins on the blank line directly after `#+RESULTS:`
       runs on through the text lines below it. So one blank line between
       `#+RESULTS:` and the arrow line does not protect the arrow line. Two
       blank lines do.
     - A `:RESULTS:` drawer is the one construct, and it is suppressed whole.
     - A non-blank line between `#+END_SRC` and `#+RESULTS:` detaches the
       result, and so does any block kind other than `SRC`.
     - The export value is read as go-org parses the header. `:exports CODE`,
       `:EXPORTS code`, and a header with no language token
       (`#+BEGIN_SRC :exports code`) each leave the result rendering in place.
     - `:exports results`, `:exports both`, and an absent `:exports` render it
       in place too.
   - Every **context-free** shape the parser classifies as something other
     than a paragraph. Measured at v1.9.1 these include a table row
     (`| A :-> B |`, whose cell delimiters the right side would swallow into a
     marker), a heading (`* A :-> B`), a keyword line (`#+TITLE: A :-> B`), a
     comment line (`# A :-> B`), a **fixed-width line** (`: A :-> B`, which
     renders inside `<pre class="example">` — the block interior's hazard
     reached without a block), and a **footnote definition**
     (`[fn:1] A :-> B`, which renders to **nothing at all**).

     The comment line, the footnote definition, the property drawer, and the
     suppressed source-block result share the worst property: they render
     nothing, so an arrow composed there produces a marker nobody sees — and
     where it is the entry's only marker, a note Anki refuses as a cloze note
     with no deletion, with nothing announcing it. Four shapes behave this way,
     and each was found only after an earlier list had been written. That is
     the argument for the definitional rule rather than for a longer list.
6. The classification of sub-clause 5 shall be derived from the parser's own
   classification rules for the vendored version, never from a list of named
   line shapes. This sub-clause is the implementation half of sub-clause 5's
   obligation and shall not narrow it: the two halves below are split by
   whether the answer depends on the lines around it, and neither is a closed
   set.

   - **Context-dependent classes** — a line whose classification depends on a
     construct opened on an earlier line. These cannot be decided from one
     line: measured, an indented arrow line reads as a paragraph on its own and
     as list content after a bullet, a line inside a source block reads as a
     paragraph on its own, and so does a line inside a LaTeX environment, a
     footnote definition's continuation, a property drawer, or a source
     block's suppressed result. The scanner shall identify them by
     **mirroring the parser's own opening-and-continuation rule — and, where
     the renderer's writer decides whether a construct is emitted, the
     writer's rule — for every construct that can make a later line something
     other than in-place paragraph text**. SPEC-ANKICARD-003's scanner already
     mirrors three such rules — the `#+BEGIN_` pairing, the bullet grammar,
     and the list-continuation rule — and they are reused rather than copied.
     The set is defined by that rule, not by a list. As read for this
     revision, the constructs it reaches at go-org v1.9.1 beyond the reused
     three **include** the four sub-clause 5 measures: the LaTeX environment,
     the footnote definition, the property drawer, and the result attached to
     a source block. That list is informative, and a construct the rule
     reaches that it omits is still bound by the rule.

     The mirrored rules shall be composed in one fixed order. First, the
     block regions (the reused `#+BEGIN_` pairing, unchanged) and the LaTeX
     environment regions are each computed over the whole remaining body.
     Next, the footnote-definition rule and the source-block-result rule run
     over the body, reading those regions rather than the raw lines. The
     property-drawer and list rules are computed on their own. A line is
     excluded when **any** rule excludes it. § 5's interleaving entry states
     its outcomes for this composition.
   - **Context-free classes** — a line whose classification depends on that
     line alone. The classification shall come from the parser's own
     classification of the line, not from testing it against named shapes.
     How the composer obtains it is the implementer's choice; `plan.md` DD-4
     records one route measured to answer correctly for every context-free
     shape sub-clause 5 lists, including the two no earlier list named. No
     route that maps a node of the parser's tree back to a source line is
     available, because no node carries a position.

   Where the informative list comes from. A construct that governs **later**
   lines is opened by a case of the parser's construct dispatch, which at
   v1.9.1 is a switch of thirteen cases, and the list in sub-clause 5 was
   **read** from that switch case by case rather than collected by probing.
   The reading is not a proof that the list is complete. A case can govern
   lines beyond its own delimiters, and whether those lines are emitted can
   be decided by the writer rather than the parser. The block case does both:
   it attaches a following `#+RESULTS:` to a source block, and the writer
   suppresses it under `:exports code|none`. An earlier reading of the same
   switch missed that. The obligation is therefore the rule above. A later
   parser version, or a case read wrongly, is covered by the rule and not by
   the list.

   Three drafts of this sub-clause narrowed the obligation and are recorded so
   the shape is not repeated. One enumerated four context-free shapes and said
   "only" those were new; the fixed-width line and the footnote definition were
   outside it. The next listed three context-dependent classes as the set and
   located the LaTeX environment through the parser's tree, which carries no
   positions; the footnote definition's continuation and the property drawer
   were outside it. The third said the remaining context-dependent constructs
   at v1.9.1 "are the three" and called its reading of the switch a
   derivation; the source block's attached result was outside it.
7. Content that is not an arrow line — a lead paragraph, a list, a block, a
   table, an ordinary sentence between two arrow lines — shall render as it
   does today, in document order, outside every generated marker. It is
   context, and dropping or restructuring it would be a content change the user
   did not ask for. The rule binds the content's **text, its document order,
   and the block-level element it renders inside**: composition shall write
   nothing that changes which paragraph, list, block, or other block-level
   element a non-arrow line belongs to.

   The rule binds **non-arrow** lines, and it is stated at block level on
   purpose. An arrow line's own inline constructs — a link, a code or verbatim
   span, emphasis, an author's export snippet, an inline footnote — are split
   by composition when the arrow sits inside one; that is § 5's
   inline-construct exclusion, measured there, and no clause of this SPEC
   promises otherwise.

#### REQ-SW-002 [Where]
Where a swift entry's remaining body yields no arrow line, the back end shall
reject that entry with the diagnostic code `swift_arrow_missing`:

1. The rejected entry shall be reported as skipped, carrying its existing note
   identifier where it has one, and shall leave the collection, the registry,
   and the Org heading untouched — the same treatment every other skip
   diagnostic already receives.
2. The obligation shall bind regardless of what the body does contain. A body
   whose every arrow sits where REQ-SW-001.5 excludes it — in list content,
   inside a block or a LaTeX environment, in a table, in a footnote
   definition — or on a line with one empty side, or inside a hand-written
   marker's span, yields no arrow line and is rejected: a heading that asks for a swift card and
   produces none is the case this diagnostic exists to report, and admitting it
   silently would leave the user with a note that reviews nothing.

### 3.2 Composing the cards

#### REQ-SW-003 [Ubiquitous]
Arrow lines shall be numbered by position, counting arrow lines only:

1. The **i-th** arrow line of an entry, counted from one over the arrow lines
   in document order and ignoring every other line, shall number its **right**
   side `c(2i−1)` and its **left** side `c(2i)`.
2. The count shall restart at one for **every entry**, since a heading is one
   note (design record § 2.3 decision C) and Anki numbers cloze deletions
   within a note.
3. The numbering shall be **positional**, so a side the direction leaves
   unwrapped still consumes its number and that number simply goes unused. The
   alternative — packing numbers densely over the wrapped sides only — would
   make a line's numbers depend on the tokens of the lines above it, so
   inserting one `:<-` line above another would silently renumber every card
   below it and Anki would lose their review histories.

#### REQ-SW-004 [Ubiquitous]
The arrow token shall select which of a line's two sides composition wraps in a
generated marker:

1. `:->` shall wrap the right side and shall leave the left side unwrapped.
2. `:<-` shall wrap the left side and shall leave the right side unwrapped.
3. `:<->` shall wrap both, so a bidirectional line yields two cards.

#### REQ-SW-005 [Ubiquitous — negated]
Composition shall not disturb a hand-written marker already present in the
title or the remaining body.

The head clause binds the hand-written **cloze marker** and no other authored
construct. The cloze marker is the one inline construct this SPEC protects,
because it is the one composition already reads in its own vocabulary — the
numbering base and the wrap decision both scan for it — so protecting it needs
no second grammar. The other inline constructs an arrow can sit inside are § 5's
exclusion, with their measured cost; reading this requirement as protecting them
would promise what no sub-clause below delivers.

1. A line whose first arrow token lies **inside a hand-written marker's span**
   shall **not be an arrow line**. This clause, not sub-clause 2, is what makes
   the head clause above true: a marker that spans the arrow leaves each side
   holding only a fragment of it, so a per-side guard sees no whole marker on
   either flank and wraps one of them anyway. Measured, composing
   `{{c1::도쿄 :-> 일본}}` under the remaining sub-clauses yields
   `{{c1::도쿄 @@html:…@@ {{c2::일본} } }}`, and Anki's non-greedy close then
   produces **one** card whose visible text is `{{c2::일본} }` — the author's
   deletion destroyed and a generated marker shown as literal characters.

   A **span** shall be taken as opening at `{{cN::` and closing at the first
   `}}` after it, which is Anki's own non-greedy rule rather than a second one
   invented here. An **unclosed** opening shall be treated as spanning to the
   end of the remaining body: on its own it produces no Anki card at all, but a
   generated marker written below it supplies the closing `}}` it lacks and so
   brings it to life around content the author never marked.

   Deciding this needs a marker-**extent** scan and shall not be attempted with
   the existing presence predicate, which matches only a marker's opening
   prefix and therefore answers "is there a marker somewhere" rather than "does
   this offset sit inside one".

   Reachability is what makes this a regression rather than an authoring edge
   case: the existing cloze-region command writes exactly this shape when the
   user selects a whole line, and `ANKI_SWIFT` inherits, so a file-level
   property reaches headings marked before the option existed.
2. A side that already carries a hand-written marker shall **not be wrapped**,
   and shall reach the renderer byte-for-byte as the author wrote it. Anki's
   cloze pattern closes at the first `}}` after an opening marker, so the two
   cannot nest and wrapping anyway would destroy the card the author's marker
   already makes. This is SPEC-ANKICARD-003 REQ-ML-006.1 applied to a side
   rather than to an answer item; the rule is the same rule. Sub-clause 1
   handles the marker that spans the arrow; this one handles the marker that
   sits wholly on one side of it, and both are needed.
3. Generated numbering shall begin above the highest number any hand-written
   marker in the entry carries. The offset shall apply to the whole formula —
   the i-th arrow line numbers its right side `c(base + 2i − 1)` and its left
   side `c(base + 2i)`, where `base` is that highest number, or zero where
   there is none. Offsetting the formula rather than the first number is what
   preserves REQ-SW-003's positional property in the presence of an author's
   marker; and where no hand-written marker is present — the ordinary case —
   `base` is zero and the numbers are exactly those the design record § 2.2
   fixes.
4. Sub-clause 2 **governs** where it meets REQ-SW-004: the direction table
   binds only the sides composition actually wraps. A line whose selected side
   is already marked therefore contributes no generated marker and is the
   specified outcome rather than an unmet obligation.

#### REQ-SW-006 [Ubiquitous]
Composition shall keep a generated marker from closing early, by the rule
SPEC-ANKICARD-003 REQ-ML-007 already fixes and from that rule's single
implementation: a run of two or more consecutive `}` characters inside wrapped
content separated by a single space each, and wrapped content ending in `}`
carrying one space before the closing `}}`. This SPEC adds no second copy of
the rule and changes nothing about it; it is restated here only because a
reader checking swift's composition against this document would otherwise find
the obligation absent.

#### REQ-SW-007 [Ubiquitous]
The arrow token shall be emphasized in the rendered card, and the emphasis
shall be written as an Org **export snippet** rather than as raw HTML:

1. Composition shall replace the arrow token with an export snippet carrying a
   `<b>` element around the token's HTML-escaped text, and that element shall
   carry the class REQ-SW-011.2 names. Measured, raw `<b>` written into Org
   source renders as `&lt;b&gt;` — the tag shows on the card as literal
   characters — while an export snippet's content passes through go-org
   verbatim. Org's own bold markup is not an alternative: it emits `<strong>`,
   not `<b>`. `plan.md` DD-2 carries both measurements.
2. The token inside the snippet shall be HTML-escaped (`:-&gt;`, `:&lt;-`,
   `:&lt;-&gt;`). Measured, both the escaped and the unescaped forms render as
   the intended characters; the escaped form is specified because the snippet's
   content is emitted as raw HTML, where a bare `<` or `>` is tolerated rather
   than well-formed.
3. **Composition shall place inside an export snippet only a value drawn from
   a fixed table of constants** — the three arrow forms and the line break
   REQ-SW-011.1 requires — and shall never assemble a snippet's content from a
   line's own text. The snippet bypasses go-org's escaping entirely, so text
   placed there reaches the card as markup; and measured, an unrecognized
   backend name renders the snippet to **nothing at all**
   (`@@latex:<b>x</b>@@` yields an empty span), so a constructed snippet can
   also delete content silently. The constraint is narrow and absolute because
   neither failure is visible in the Org buffer.

   The bound is on **composition**, not on the card. An export snippet an
   author writes in a body already reaches the card as raw HTML and has done
   since before this SPEC; that is pre-existing go-org behavior and is not
   changed here.

### 3.3 Pipeline placement

#### REQ-SW-008 [Ubiquitous]
Composition shall sit between the checked supplementary-content split and the
cloze-marker gate, in the fixed order SPEC-ANKICARD-003 REQ-ML-009 established
and SPEC-ANKICARD-005 REQ-AKX-004 extended — split, check, compose, gate,
render:

1. Composition shall read the **remaining body** of the checked split, so
   supplementary content can never become an arrow line. An arrow inside a
   `#+BEGIN_EXTRA` block is supplementary content and is not a card. A split
   that leaves an `#+BEGIN_EXTRA` or `#+END_EXTRA` line unpaired is rejected by
   the check with `extra_block_unbalanced` before composition runs, so no
   stray marker line ever reaches the arrow scan. The swift path shall take
   that check from its existing single implementation, as the other two
   cloze-style paths do, rather than calling the unchecked split.
2. Composition shall **not** collapse blank-line runs. The collapse
   SPEC-ANKICARD-003 REQ-ML-009.2 requires exists to keep one authored answer
   **list** from being split in two by the gap a removed supplementary block
   leaves; swift reads independent lines and has no list to keep together, so a
   wider gap merely yields more paragraphs and loses nothing. Declining to
   reuse the collapse is therefore a smaller surface, not a gap: the failure it
   guards against is unreachable here.
3. A generated marker shall **satisfy** the cloze-marker gate, so a swift entry
   carrying no hand-written marker shall render rather than be skipped with
   `cloze_marker_missing`. The gate is in fact unreachable for a composed swift
   entry: a side is unwrappable only by being empty after trimming, which
   REQ-SW-001.4 excludes from being an arrow line at all, or by already
   carrying a hand-written marker, which satisfies the gate by itself.

#### REQ-SW-009 [Ubiquitous]
The swift option shall reach the renderer through the **existing**
option-taking entry point, whose dispatch this SPEC extends rather than
duplicates:

1. The renderer's card-option carrier shall gain the swift option, and the
   existing option-taking entry point shall dispatch on card kind. A third
   entry point shall **not** be added: SPEC-ANKICARD-003 REQ-ML-010.1 made the
   no-option path structural by having the option-taking entry point delegate
   to the unwidened one, and a third door would reopen exactly the drift that
   delegation closed.
2. The unwidened entry point shall keep its current parameters and its current
   output for every note type, so the signature assertion SPEC-ANKICARD-002
   established continues to compile and to pass unchanged.
3. The dispatch shall be **total**: where both the swift option and a multiline
   option are somehow on, the entry shall be treated as **swift**, and where
   the note type is not cloze-style the entry shall delegate to the unwidened
   entry point. Neither branch is reachable from production — the validation
   gate rejects both combinations before the render — and both are specified so
   that the dispatch has a defined answer rather than an accidental one. Swift
   is chosen over multiline for the unreachable first branch only because a
   tie must be broken somewhere and the swift check is the narrower predicate.

#### REQ-SW-010 [Ubiquitous]
`swift_arrow_missing` shall be reported only for an entry that already passed
SPEC-ANKICARD-002's validation gate, so it shall never compete with that gate's
three codes for the one-diagnostic-per-entry slot. It is fifth by construction
rather than by a fifth entry in the gate's fixed order: the gate runs before the
render, and this condition is detectable only during composition. It can never
co-occur with `multiline_answer_missing`, because the gate rejects an entry
carrying swift beside a multiline option before either renderer runs. And it can
never co-occur with `extra_block_unbalanced`, the render-time code
SPEC-ANKICARD-005 added: that check runs on the split, before composition
(REQ-SW-008), so an entry it rejects never reaches the arrow scan, and the
reported code is `extra_block_unbalanced` even where the body also carries no
arrow line.

### 3.4 Presentation, front end, and non-interference

#### REQ-SW-011 [Ubiquitous]
Two consecutive arrow lines shall render on separate lines, and the arrow shall
carry a class the card stylesheet can target:

1. Composition shall append an **inline line break** to an arrow line **if and
   only if** the line immediately following it is also an arrow line. Without
   it the card is wrong rather than merely plain: measured, consecutive arrow
   lines form **one** paragraph carrying a literal newline, and HTML collapses
   that newline to a space, so two cards' worth of text run together on one
   line. The break shall ride the same export-snippet mechanism as the
   emphasis, drawn from the same fixed table (REQ-SW-007.3).

   The condition is narrow on purpose. An inline break inserted wherever an
   arrow line is followed by any non-blank line would move a following
   **prose** line onto its own line, which REQ-SW-001.7 forbids; restricting it
   to an arrow-line successor leaves every non-arrow line rendering exactly as
   it does today, and fixes only the shape this requirement exists to fix.

2. The emphasis element of REQ-SW-007.1 shall carry the class `swift-arrow`,
   and the imoogi-owned base stylesheet shall carry a rule for it. The rule
   shall name no network-hosted resource and derive nothing from a deck name —
   the properties that stylesheet already asserts mechanically.

3. Composition shall **not** insert a line of its own into the body. Neither
   sub-clause above may be implemented by writing a line at column zero, by
   inserting a blank line, or by otherwise changing which element any line
   renders inside. Measured, a container attribute line — the mechanism
   SPEC-ANKICARD-003 REQ-ML-012 uses for the answer list, and the obvious way
   to reach for a class here — **terminates a list** when the arrow sits on an
   indented continuation line: `- 수도` followed by `  도쿄 :-> 일본` is one
   list item holding two lines, and an attribute line between them splits it
   into a truncated item plus a sibling paragraph. REQ-SW-001.5 now excludes
   list content, so that particular shape can no longer arise; this sub-clause
   is the second, independent guard, because the hazard belongs to the
   insertion mechanism rather than to the list, and the next multi-line
   construct nobody thought of would meet it again. The emphasis and the break
   carry no such risk **to a line's block-level element**: measured, they leave
   a list item intact. They can still split an inline construct the arrow sits
   inside, which is § 5's inline-construct exclusion and is not this
   sub-clause's claim.

#### REQ-SW-012 [Ubiquitous — negated]
The back end shall not change what a **non**-swift entry renders to:

1. An entry for which the swift option is not on shall produce a byte-identical
   field map, so this SPEC causes no existing note to be reported as updated on
   account of content that did not change. The obligation covers the
   no-option entry and the multiline entry alike.
2. The content hash's input set shall gain no member.
3. The two byte-identity corpora the earlier SPECs recorded shall be
   **extended, not replaced**: their recorded values and their signature
   assertion shall stand unmodified, and this SPEC's swift cases shall be
   measured by a separate corpus of their own, mirroring the split
   SPEC-ANKICARD-003 REQ-ML-014.3 established.
4. The hash probe SPEC-ANKICARD-002 recorded shall be **reshaped, not
   deleted**, and its narrowed claim shall be stated where the probe lives.
   The probe asserts that a card-option field carried on the entry does not
   reach the hash as an input of its own, and it has until now demonstrated
   that by setting an option with no rendering meaning. This SPEC removes the
   last such option, so the probe shall instead carry every option field in its
   **falsy** spelling — present on the wire, resolved off, rendering
   identically — and shall be paired with a sibling assertion that an option
   which is **on** does change the hash, through the rendered field value and
   only through it.

   Those two halves are individually necessary and **not jointly sufficient**,
   and the SPEC shall say so rather than imply otherwise. An implementation
   that fed the *resolved boolean* to the hash while ignoring the wire field
   would pass both: the falsy-versus-absent pair resolves off on both sides and
   hashes alike, and the on-versus-absent pair is supposed to differ. What
   excludes it is the **compile-time assertion that the hash call site takes
   exactly its four existing arguments**, which already exists and which
   `plan.md` § A.3 preserves. The contract is therefore a **trio**, and the
   three together exclude the three evasions each leg is aimed at: the first
   excludes hashing the wire field, the second excludes ignoring the option
   entirely, and the third excludes introducing a new parameter for it. The
   trio, not the pair, shall be what REQ-SW-012.2 is verified by. The helper
   docstring this sub-clause requires shall name all three legs **and** the
   three evasions they exclude, stated as three rather than as every route, so
   the narrowing is read where the probe is.

   The trio does **not** exclude every conceivable route, and the claim is
   pitched at that strength deliberately, because the looser form is the same
   overclaim this sub-clause exists to repair. An implementation that folded the
   resolved value into one of the four **existing** arguments — appending it to
   the sorted tags, say — passes all three legs while REQ-SW-012.2 forbids it.
   That route is low-reachability, the hashing package being on the PRESERVE
   list and marked unchanged in § 7, so it is recorded here rather than guarded
   against.
5. The shared option fixture the gate's own tables run against shall be
   extended to satisfy this card kind's precondition too, so those tables go on
   testing the gate rather than becoming tests of the renderers downstream of
   it. This mirrors what SPEC-ANKICARD-003 did to the same fixture when it gave
   the multiline options a rendering meaning, and it is the last such extension
   the series needs.

#### REQ-SW-013 [Ubiquitous]
The editor shall gain the command that writes `ANKI_SWIFT`, reached from the
existing Anki transient, and that property shall join the property-name
completion candidates:

1. Setting the swift option shall be a **toggle**, mirroring the incremental
   toggle, and shall read the **inherited** value to decide which way to flip —
   the state on screen is the state the user means to reverse.
2. Turning the option **off** shall write the falsy spelling into the heading's
   own drawer rather than delete the property, because all three card-option
   properties inherit and deletion would let an ancestor's value apply again.
   Verification of this shall not use `org-entry-get`, which SPEC-ANKICARD-003
   REQ-ML-013.2 measured to return Lisp `nil` for a drawer plainly carrying the
   falsy spelling; the accessors that do read it back are named there.
3. The note-type contract shall bind the **on** direction only. Where the
   command is turning the option on: a heading carrying no `ANKI_NOTE_TYPE`
   receives `imoogi-Cloze`; a cloze-style type is left alone; any other type is
   reported and nothing is changed.

   Where the command is turning the option **off**, the contract shall not
   apply: the falsy spelling shall be written whatever the note type is, and no
   note type shall be created. Both halves of the unqualified form are wrong on
   that path. Refusing to write would leave a heading carrying `imoogi-Basic`
   with an inherited swift-on unable to opt out through the command at all —
   the exact heading SPEC-ANKICARD-002's gate already rejects, and whose
   diagnostic tells the user to remove the option. And writing `imoogi-Cloze`
   onto a heading that carried no note type, as the consequence of *declining* a
   card option, would create a card marker the user did not ask for.

   The current tree argues for this resolution rather than against it. The
   message the front end shows for `card_option_needs_cloze`
   (`modules/org/anki/imoogi-error.el`) tells the user, for an option inherited
   from an ancestor or a file-level line, to switch it off for the heading alone
   by giving it the value `nil` in its own drawer. That is exactly the write the
   OFF path performs; binding the contract to the OFF path too would make the
   command refuse the very fix the diagnostic prescribes.

   The ambiguity is inherited rather than introduced: the incremental toggle
   reaches the same helper on its own off path and has the same shape. This SPEC
   resolves it for the command it owns and changes nothing about the existing
   commands, which is why the resolution is a clause here rather than an
   amendment to an earlier SPEC.

   The command shall reach that contract through the existing **note-type**
   helper, and shall **not** be routed through the existing card-option write
   helper. That helper runs the swift-conflict check unconditionally before the
   note-type check, and for this command the check is backwards on the exact
   path sub-clause 2 specifies: turning swift **off** happens precisely when
   the resolved value is on, so every off would first be asked whether to turn
   swift off, and a user who declined could never turn it off at all. The
   note-type helper shall therefore be usable on its own — either by factoring
   it out of the write helper, or by giving the write helper a parameter
   naming which conflict check to run — and this SPEC leaves the choice to the
   implementer while requiring that the off path fire no swift-conflict prompt.

   The note-type helper's own failure message shall stop naming multiline
   options, since it now reports for a swift heading too.
4. The multiline conflict check shall bind the **on** direction only, as
   sub-clause 3's note-type contract does. Where the command is turning the
   option on and the heading resolves **any** multiline option to an on value,
   the command shall report the conflict and shall write nothing unless the
   user confirms. On confirmation it shall clear **every** multiline option
   that is on — writing the falsy spelling into the heading's own drawer for
   each, never deleting the property — and the report shall name each of
   them. Both `ANKI_DIRECTION` and `ANKI_INCREMENTAL` can be on at once, and
   the existing swift-conflict check clears one property only because it has
   one to clear; the mirror has two, so "clear it" would leave the heading in
   the very state the check exists to prevent.

   This is the **mirror** of the swift-conflict check the multiline commands
   already perform, and it exists for the same reason: writing swift beside an
   active direction would create exactly the entry SPEC-ANKICARD-002's conflict
   rule rejects, at the moment the user believed they had configured a card.

   Where the command is turning the option **off**, the check shall not run.
   The command shall write the falsy `ANKI_SWIFT` into the heading's own
   drawer, shall fire no multiline-conflict prompt, and shall read, clear, or
   write no multiline option. The heading the OFF path reaches with a
   multiline option on is the one SPEC-ANKICARD-002's gate already rejects
   with `card_option_conflict`. That code's message
   (`modules/org/anki/imoogi-error.el`) tells the user to remove one of the
   two, and to switch an inherited option off by giving it `nil` in the
   heading's own drawer. Turning swift off **is** that resolution. Running the
   check here would ask a user who wants to keep the multiline card whether to
   clear it. Declining would write nothing, so the user could not opt out of
   swift through the command, and confirming would clear the very option they
   meant to keep. That is sub-clause 3's argument, applied to the second gate
   on the same path. The existing `imoogi-anki--resolve-swift-conflict` runs
   unconditionally, and mirroring its shape without this clause would
   reintroduce the defect.
5. `ANKI_SWIFT` shall join the property-name completion candidates, discharging
   the exclusion SPEC-ANKICARD-003 REQ-ML-013.5 recorded — that a property's
   candidacy travels with the command that writes it, and that this card owns
   both.

#### REQ-SW-014 [Ubiquitous]
The diagnostic code this SPEC introduces shall be paired with a user-facing
message in the front end's code-to-message table, which remains the only
component that renders prose for a code:

1. The message shall name the problem and state a corrective action — including
   what does **not** count as an arrow line, since a body that looks to its
   author as though it carries arrows is the case that will produce this
   diagnostic — and shall contain no stack trace, raw transport error,
   backtrace, or bare exit code.
2. The existing two-directional pairing assertion shall continue to hold over
   the enlarged code set, including the hard-coded back-end code list it
   cross-checks against.

## 4. Relationship to SPEC-ANKICARD-003

SPEC-ANKICARD-003 is a **dependency, not a parent**. This SPEC amends none of
its requirements and changes no behavior it introduced. Its surfaces are
consumed exactly as it left them at `08f7504`, and two of its deferrals are
discharged:

- **Consumed unchanged.** The option-taking entry point and its delegation to
  the unwidened one (REQ-ML-010.1); the fixed pipeline order (REQ-ML-009), as
  SPEC-ANKICARD-005 since extended it with the unpaired-marker check; the
  supplementary split it composes after, taken in that checked form; the
  block-interior pairing rule, the bullet grammar, and the list-continuation
  rule its scanner carries; the brace-safety rule and its single
  implementation (REQ-ML-007); the hand-written-marker conventions — not
  wrapping a marked span, and numbering above the highest marker (REQ-ML-006.1
  and REQ-ML-006.2). This SPEC defines no second copy of any of them.
- **Deliberately NOT reused — the container-attribute mechanism.** REQ-ML-012
  attaches a class by writing `#+ATTR_HTML:` above the answer list, and that is
  the natural way to reach a class here too. It is refused: the attribute line
  is written at column zero, and measured it **terminates a list** when the
  arrow sits on an indented continuation. REQ-SW-011.3 forbids the mechanism
  outright, and the class rides the emphasis element instead. This is the one
  surface of SPEC-ANKICARD-003 this SPEC declines; declining it changes nothing
  about the multiline card, which goes on using it.
- **Discharged deferral — the swift editor command.** Its § 5 named "an editor
  command that writes `ANKI_SWIFT`, and that property's entry in the
  property-name completion candidates" as belonging to this card. REQ-SW-013
  delivers both.
- **Discharged deferral — the in-body arrow forms.** Its § 5 stated that it
  reads arrows only as `ANKI_DIRECTION` property values and never in a body,
  and assigned the in-body forms here. REQ-SW-001 recognizes them.
- **A note on the two arrow vocabularies.** `ANKI_DIRECTION` takes `->`, `<-`,
  `<->`; an arrow line carries `:->`, `:<-`, `:<->`. The leading colon is what
  separates them, and it is the reason a direction value can never be mistaken
  for an arrow token or the reverse. The two are read by different code on
  different inputs and this SPEC keeps them apart rather than unifying them: a
  shared parser would have to be told which vocabulary it was reading, which is
  the whole of what distinguishes them.

## 5. Out of Scope

This SPEC renders one card kind and no other. The exclusions below are what
makes that claim checkable: each names work an implementation could plausibly
fold in, and states where it actually belongs.

### Out of Scope — Multiline card rendering

- Every rule SPEC-ANKICARD-003 owns: the answer list, the direction and
  incremental semantics, the `multiline_answer_missing` diagnostic, the
  `children-list` container, and the direction and incremental editor commands.
  This SPEC reads none of those options and composes none of those cards.

### Out of Scope — Arrows in list content

- Reading an arrow that sits in an Org list — on a bullet line or on a line the
  list-continuation rule attaches to one. REQ-SW-001.5 excludes it, and the
  cost is named here rather than left to be found: a body whose arrows all sit
  in list content yields no arrow line and is reported with
  `swift_arrow_missing`; a body **mixing** plain arrow lines with list arrows
  carries the list arrow as ordinary content rather than as a card, with no
  diagnostic.
- Nothing is dropped, and the wording matters because a future reader should
  not set out to repair a loss that is not occurring. REQ-SW-001.7 keeps the
  list line on the card, in document order, rendered as the list item it is;
  what is withheld is the cloze pair. That is materially unlike the failure
  class the predecessor SPEC overturned, where answers left the card entirely
  and the reviewer had no way to see it.
- The trade is deliberate. Reporting it would need a diagnostic code describing
  "some arrows were not read", which is a new requirement and a new code for a
  shape whose correct rendering is itself undecided — the description term's
  `::` and the checkbox status both compete for the same line, and neither has
  an obvious answer.
- **A third option was offered and is refused, on a stated reason.** The plan
  audit noted that the ambiguity comes entirely from the three consumptions
  SPEC-ANKICARD-003's offset function performs, and that swift could admit a
  bullet line whose computed offset is zero while excluding the rest — reusing
  two functions already reused here and adding no second grammar. It is
  refused for three reasons, none of them effort. It would make which lines
  become cards depend on a value computed by a **different SPEC's** rule, so a
  later change there would silently change what this card renders. It would
  split one visible class in two on a distinction invisible in the buffer:
  `- 도쿄 :-> 일본` would become a card while `- [ ] 도쿄 :-> 일본` would not,
  with no signal — trading a disclosed uniform limitation for an undisclosed
  irregular one. And it was proposed largely to remove a silence that, on the
  corrected reading above, is a withheld card rather than lost content.
  `acceptance.md` pins the excluded behavior so it cannot change by accident,
  and a later card may revisit it with the diagnostic it would need.

### Out of Scope — Arrows in block interiors

- Reading an arrow inside a `#+BEGIN_…` block. REQ-SW-001.5 excludes it by
  reusing SPEC-ANKICARD-003's one pairing rule, which costs the
  `#+BEGIN_QUOTE` case that would in fact have rendered correctly. Narrowing
  the exclusion to the verbatim block types would mean a second pairing rule in
  one package, whose divergence from the first would be invisible until a card
  rendered wrong.
- Matching go-org where a `#+BEGIN_` block and a `\begin{…}` environment
  **interleave** rather than nest — one construct's closing line falling
  inside the other. The outcomes below are stated for the composition
  REQ-SW-001.6 fixes: block and environment regions computed independently
  over the whole body, with the excluded set taken as their union. The parser
  side of each is measured. The scanner side is reasoned from that composition
  and from `blockRegions`' code, because no scanner exists yet to measure.

  Two nestings were measured and agree: an environment wholly inside a
  `#+BEGIN_SRC`, and a `#+BEGIN_SRC` wholly inside an environment. go-org
  renders each as one raw interior, and the union excludes each interior. That
  claim is no wider than those two shapes. Where a footnote definition holds a
  nested block or environment, agreement rests on REQ-SW-001.5's footnote rule
  skipping those regions, which is what corpus row 41 pins.

  Three interleavings, each of which ends with the arrow line `A :-> B`:
  - **Environment first** (`\begin{align}`⏎`#+BEGIN_QUOTE`⏎`\end{align}`⏎`A :-> B`⏎`#+END_QUOTE`).
    go-org renders the environment, then `A :-> B` as an in-place paragraph,
    then the orphan `#+END_QUOTE` as text. The union pairs the quote across
    the environment's close, so it **over-excludes** the arrow line.
  - **Source block first** (`#+BEGIN_SRC go`⏎`\begin{align}`⏎`#+END_SRC`⏎`A :-> B`⏎`\end{align}`).
    go-org copies the `\begin{align}` line into the source interior, renders
    `A :-> B` in place, and renders the orphan `\end{align}` as text. The
    independent environment scan opens on the line inside the source
    interior, so the union **over-excludes** the arrow line here too.
  - **Quote first** (`#+BEGIN_QUOTE`⏎`\begin{align}`⏎`#+END_QUOTE`⏎`A :-> B`⏎`\end{align}`).
    go-org's environment scan runs past the quote's close, the quote fails and
    renders as text, and the arrow line is raw environment interior. The union
    excludes the arrow line through the environment region, so it **agrees**.

  In each of these shapes, the composition's divergence from the parser is
  over-exclusion: a withheld card, or `swift_arrow_missing` where the line was
  the body's only arrow line. It never composes a marker the card hides.
  Matching the parser exactly would mean modelling its failed-block
  backtracking, a second parser in one package, for shapes no author writes on
  purpose. The over-exclusion is accepted instead.

### Out of Scope — Arrows inside an inline construct

- Composing an arrow line whose arrow sits inside an Org **inline** construct —
  a link, a code span, a verbatim span, emphasis, an author's own export
  snippet, or an inline footnote. REQ-SW-001.5 classifies a **line**; these
  live inside a line the parser reads as ordinary paragraph text, so the rule
  admits them and composition splits them. REQ-SW-005.1 closes exactly one
  member of the class — the hand-written cloze marker — and the rest stay open.
- The cost is measured, not estimated, and is stated per member so that a
  reader can judge it rather than take a summary:

  | Construct | Source | What composition produces |
  |---|---|---|
  | Link | `[[url][도쿄 :-> 일본]]` | the marker straddles `</a>`: `{{c1::일본</a>}}` |
  | Code span | `~도쿄 :-> 일본~` | the raw `@@html:` text shows on the card as literal characters |
  | Verbatim | `=도쿄 :-> 일본=` | the `=` inside the snippet's `class="…"` ends the span early |
  | Emphasis | `*도쿄 :-> 일본* 메모` | the marker straddles `</strong>` |
  | Author's snippet | `@@html:<i>…:-> …</i>@@` | the nested `@@` closes early and text is lost |
  | Inline footnote | `[fn::도쿄 :-> 일본]` | the content relocates to the document's end, so the closing `}}` lands **before** its `{{c1::` and Anki finds no deletion |

  The last row is the worst and is why the class is disclosed rather than left
  unmentioned: Anki refuses a cloze note with no deletion, so the heading
  produces nothing and the failure surfaces at Anki rather than in the Org
  buffer. The code-span row is the sharpest in kind, being byte-for-byte the
  failure REQ-SW-001.5's block-interior bullet exists to prevent, reached
  through a line that same sub-clause admits.
- **Why disclosure rather than repair.** Six constructs with five different
  delimiter syntaxes is a second grammar, and REQ-SW-001.6 has just struck one
  enumeration for being incomplete; adding another at the inline level would
  repeat the mistake one layer down. A general inline-extent scan is the
  honest fix and it is a larger piece of work than this card.
- **One alternative was considered and rejected.** Composition could render the
  line twice — before and after — and decline where the two differ beyond the
  inserted marker and snippet. It would cover the whole class at once with no
  new grammar. It is rejected because it makes every arrow line pay a second
  render, and because "differ beyond the insertion" is itself a comparison rule
  this SPEC would have to define and test — a new mechanism of exactly the kind
  the last two iterations were spent removing.
- `acceptance.md` pins the measured behavior so it cannot change silently,
  following the precedent SPEC-ANKICARD-003 set for the dangling supplementary
  marker: the shape is recorded as specified-and-not-repaired rather than
  reported, and a later card may take it up with the scan it needs.

### Out of Scope — A second arrow on one line

- Reading the second and subsequent arrow tokens of a line as further pairs.
  REQ-SW-001.3 fixes the first token as the line's arrow; the rest is right-side
  text. A line wanting two pairs is two lines.

### Out of Scope — A per-arrow-line note

- Producing one Anki note per arrow line. The design record § 2.3 decision C
  fixed the heading-to-note-identifier relation at one to one and named
  per-line notes as the rejected alternative; a swift heading is one note
  carrying several cards, which is what Anki's cloze model already provides.

### Out of Scope — Unpaired supplementary markers

- Detecting, reporting, or repairing a nested or asymmetric `#+BEGIN_EXTRA`.
  SPEC-ANKICARD-005 (backlog card t16, `completed`) owns that: its check runs
  on the split and skips the entry with `extra_block_unbalanced` before
  composition. This SPEC adds no handling of its own and re-detects nothing; it
  takes the checked split as REQ-SW-008.1 requires, so a stray marker line never
  reaches the arrow scan. Earlier drafts of this entry, written before that SPEC
  landed, said a stray marker would be read as ordinary non-arrow content; on
  the current tree the entry is skipped instead.

## 6. Constraints

1. **No new note type.** The card is an `imoogi-Cloze` note. Nothing here
   creates, updates, or names a new Anki model.
2. **Composition is source-level and pre-render.** The wrapping and the
   emphasis both happen on Org source before the renderer runs, never on
   rendered HTML. `plan.md` DD-1 carries the measured justification.
3. **The back end validates and renders; it never mutates.** No code path here
   writes a property or edits an Org file. Properties are written at edit time
   by the command of REQ-SW-013.
4. **One diagnostic per skipped entry.** REQ-SW-010 keeps the new code outside
   SPEC-ANKICARD-002's fixed order rather than adding a fifth position to it.
5. **Byte-identical rendering for everyone else.** REQ-SW-012 is the constraint
   this SPEC is shaped around: an entry with no swift option must produce the
   hash it produced before, so no user sees a mass of spurious updates.
6. **Composition places raw HTML in exactly one construct.** REQ-SW-007.3 bounds
   the export snippet to a fixed table of constants. No user text, and no text
   derived from user text, is ever placed inside one. The bound is on
   composition and not on the card: an export snippet an author writes in a body
   already reaches the card as raw HTML, and did so before this SPEC.
7. **Composition inserts no line of its own.** REQ-SW-011.3. Every construct
   this SPEC writes is inline, so no line's **block-level** element can change.

   The qualifier is load-bearing and replaces a stronger claim the measurements
   contradict. An inline insertion cannot change which paragraph, list, or
   block a line belongs to — that is what REQ-SW-011.3 buys. It can still split
   an **inline** element and straddle its boundary, which § 5's inline-construct
   entry measures and excludes from this SPEC's scope. Reading this constraint
   as "nothing composition writes can change any element" is the reading that
   made the inline class invisible for two iterations.

## 7. Brownfield Delta

Per-surface disposition. `[NEW]` marks an artifact this SPEC creates;
`[CHANGED]` an existing one it modifies; `[UNCHANGED]` one it deliberately
leaves alone despite sitting on the path.

### [NEW] Swift composition (`internal/anki/orgdoc`)

The arrow-line identification of § 3.1 and the composition rules of § 3.2, in a
new file beside the existing multiline composition. Carries the typed
arrow-missing error REQ-SW-002 names. Reuses the line splitter, the block-region
pairing, the bullet grammar, the list-continuation rule, the brace-safety
helper, the marker predicate, and the highest-number scan from the multiline
file, and the checked split from `extra.go`, rather than copying any of them.
New here: the marker-extent scan, the mirrored rules for the context-dependent
constructs REQ-SW-001.6 reaches (at v1.9.1 these include the LaTeX environment,
the footnote definition, the property drawer, and the result attached to a
source block), their fixed composition order, and the context-free line
classification.

### [CHANGED] Renderer card options and entry point (`internal/anki/orgdoc`)

The card-option carrier gains the swift option and the option-taking entry
point's dispatch gains its branch (REQ-SW-009). The unwidened entry point keeps
its parameters and its behavior (REQ-SW-012.1) and remains the path every
option-free entry takes.

### [CHANGED] Protocol (`internal/anki/protocol`)

The diagnostic-code const block gains `swift_arrow_missing`. No field changes
shape; the wire contract's version is unchanged, since this SPEC adds no field
in either direction.

### [CHANGED] Planner (`internal/anki/planner`)

The option reader maps the resolved swift value onto the renderer's carrier, and
the render-error mapping gains the new code beside the three it already carries
(`cloze_marker_missing`, `multiline_answer_missing`, and SPEC-ANKICARD-005's
`extra_block_unbalanced`).
The validation gate, the field-resolution layer, the deck fallback, and the
add / update / no-op partition are untouched.

### [UNCHANGED] Content hashing (`internal/anki/hashing`)

Deliberately untouched. Its input set is note type, rendered field values,
resolved deck, and sorted tags; a swift entry changes the second of those
through the rendered value alone, which is what REQ-SW-012.2 asserts.

### [CHANGED] Card stylesheet (`internal/anki/model/assets/base.css`)

Gains the `swift-arrow` rule of REQ-SW-011.2. No container class is added: the
class rides the emphasis element, because REQ-SW-011.3 forbids composition from
inserting the attribute line a container would need.

### [CHANGED] Editor commands (`modules/org/24-anki.el`)

Gains the swift toggle of REQ-SW-013, its key binding, its transient entry, the
mirror conflict check, and `ANKI_SWIFT` in the property-name completion
candidates.

### [CHANGED] Diagnostics (`modules/org/anki/imoogi-error.el`)

Gains one code-to-message entry (REQ-SW-014).

### [UNCHANGED] Property resolution and scan (`modules/org/anki`)

Deliberately untouched. SPEC-ANKICARD-002 already resolves all three properties
through the nearest-wins chain and attaches them to the entry; this SPEC reads
what that work produces and adds no resolution of its own.

### [CHANGED] Tests

A new swift corpus separate from the two existing byte-identity corpora
(REQ-SW-012.3); composition and arrow-identification tests; the planner's
diagnostic-mapping tests; the reshaped hash probe and its new sibling
(REQ-SW-012.4); the extended shared option fixture (REQ-SW-012.5); the
code-pairing tests including the hard-coded back-end code list; and ERT coverage
for the new command.

### [CHANGED] Documentation (`README.md`)

The Anki section gains the swift card: what the property does, what an arrow
line is and what is not one, the three tokens, and how many cards a heading
yields.

## 8. Dependencies

- **SPEC-ANKICARD-003** — landed on `main` at commit `08f7504`. Every surface
  this SPEC consumes from it — the option-taking entry point, the pipeline
  order, the scanner's block, bullet, and list-continuation rules, the
  brace-safety helper, and the hand-written-marker conventions — is in that
  commit. At `7d98ec0` all of them are unchanged except the pipeline order,
  which SPEC-ANKICARD-005 extended with the unpaired-marker check
  (`git diff 08f7504 7d98ec0 -- internal/anki/orgdoc/multiline.go` touches only
  `composeMultiline`'s comment and `RenderWithOptions`). The
  container-attribute mechanism is present there too and is deliberately
  **not** consumed (§ 4).

  Its frontmatter reads `in-progress`, so the run-phase dependency pre-flight
  will find it unfulfilled. Proceeding is authorized on the same footing each
  earlier card in this series recorded: the design record's ordering note has
  the operator, told the predecessor was still in flight, directing that cards
  t11 through t15 be worked in order — a range naming this card — and the
  directive was repeated in the session that produced this SPEC.

  The consequence is recorded rather than mitigated: SPEC-ANKICARD-003's own
  closure will run against a HEAD this SPEC has moved.
- **SPEC-ANKICARD-002** — landed at `6f3ae6c`. The validation gate this SPEC's
  § 3 preamble relies on, the recognized value sets, and the wire fields are all
  in that commit.
- **SPEC-ANKICARD-005** (backlog card t16) — landed on `main`, frontmatter
  `completed`, present at `7d98ec0`. It supplies the checked split and the
  `extra_block_unbalanced` code this SPEC's pipeline runs through (REQ-SW-008,
  REQ-SW-010). This SPEC consumes the check unchanged and repairs nothing it
  owns (§ 5).
- No new third-party dependency. The renderer keeps go-org at its current
  version; `plan.md` DD-1, DD-2, and DD-3 record the probes run against it.

## 9. Traceability

Every requirement is covered by at least one criterion in `acceptance.md`;
every criterion names the requirements it verifies and the command that decides
it. `plan.md` § F maps requirements onto milestones.
