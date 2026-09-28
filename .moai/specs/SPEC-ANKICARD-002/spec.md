---
id: SPEC-ANKICARD-002
title: "Anki card-option properties and protocol v2: transport and validation"
version: "0.1.1"
status: in-progress
created: 2026-09-20
updated: 2026-09-20
author: jay
priority: P2
phase: "v0.2.0 target"
module: "internal/anki/protocol, internal/anki/planner, modules/org/anki"
lifecycle: spec-anchored
tags: "anki, org-mode, protocol, card-options, validation, inheritance"
tier: M
depends_on: [SPEC-ANKICARD-001]
---

## HISTORY

### v0.1.1 (2026-09-20)

Answers plan audit iteration 1 (`.moai/reports/plan-audit/SPEC-ANKICARD-002-plan-audit.md`,
verdict PASS 0.91). All four blocking-class findings closed; four of the five
optional findings taken. Requirement and criterion **counts are unchanged** at
13 and 14 — every edit below sharpens or bounds an existing entry, or adds a
sub-criterion under one.

- CHANGED: § 8 **drops the run-phase gate** on SPEC-ANKICARD-001's closure
  (audit D1/D2). The gate contradicted `plan.md` § J, which dispositioned the
  same risk as already authorized. The dependency is ordering and process, not
  code availability: every surface this SPEC reads from SPEC-ANKICARD-001 is on
  `main` today, enumerated in § 8. The authorization is the design record's
  ordering note — the operator, told SPEC-ANKICARD-001 was `in-progress`,
  directed cards t11 through t15 be worked in order — repeated in this SPEC's
  own session. `plan.md` § A and § J restated to match.
- CHANGED: REQ-OPT-008 gains sub-clause **5**, bounding the migration-path
  obligation to confirmed candidates (audit D4). The former sub-clause 5
  renumbers to **6**; no obligation changed. Two migration arms return before
  the per-entry pipeline — a dry run, and an entry rejected as a non-candidate
  — so no gate placed inside that pipeline can reach them. Both are inherited
  shapes, both now named, and neither loses coverage.
- CHANGED: § 3's preamble records why four requirements tag `[Where]` as a
  case-split operator over the entry set rather than as a capability gate, and
  why `While` was rejected (audit D9). The tags are unchanged.
- CHANGED: `acceptance.md` gains **AC-OPT-001e**, pinning `PROPERTY+`
  normalization to plain replacement (audit D1) — the one chain rule that
  discriminates reuse of the existing resolver from a reimplementation, and so
  the criterion that makes REQ-OPT-001.1 falsifiable rather than merely stated.
- CHANGED: AC-OPT-002c's assertion method narrowed from a whole-buffer
  comparison to the three card-option values (audit D3). A sync run writes
  `ANKI_NOTE_ID` into the buffer, so the old method failed for a reason the
  criterion is not about.
- CHANGED: AC-OPT-005's verification grep gains the hyphen keyword form and
  drops its `--include` filters, re-filtering by path (audit D5); Definition of
  Done item 4 restated to run it as written.
- CHANGED: AC-OPT-012b's inspection clause replaced by a build-time assertion,
  and its hash-invariance half assigned explicitly to the planner (audit D6) —
  the hash function takes no card-option parameter, so it cannot be handed the
  two variants to compare.
- CHANGED: `acceptance.md` gains a reading note separating `nil`-the-no-value
  from `nil`-the-falsy-spelling, and the two ambiguous sites now read "no
  value" (audit D7).
- CHANGED: `plan.md` DD-5 flags `ANKI_DIRECTION` accepting `nil` as a genuine
  expansion of a confirmed decision, with its collapse cost and plan-gate
  routing (audit D8), and drops the circular half of its justification.
- CHANGED: `plan.md` DD-2 records the auditor's independent concurrence and its
  two added arguments, and corrects the decision's footing — the repository's
  pairing rule mandates pairing, not granularity; the granularity argument
  rests on the precedent in the existing 18 codes.
- UNCHANGED: every requirement's obligation, the 13/14 counts, the § 5
  exclusions, the § 6 constraints, the tier, and every other design decision.

### v0.1.0 (2026-09-20)

- INITIAL: SPEC created from backlog card **t13** of the design record
  `.moai/reports/anki-card-types-plan-20260920.md` (§ 2 confirmed decisions,
  § 3 work split). Tier M, 3 plan-phase artifacts plus `progress.md`.
- Carries the three Org card-option properties (`ANKI_DIRECTION`,
  `ANKI_INCREMENTAL`, `ANKI_SWIFT`) from their nearest-wins inheritance chain,
  across a version-2 wire contract, into a Go-side validation gate that skips a
  mis-specified entry with a typed diagnostic. It renders nothing new.

## 1. Overview

### 1.1 Purpose

Two later cards — **t14** (Multiline cards) and **t15** (Swift cards) — need
three per-heading options to reach the Go back end before either can render
anything. This SPEC builds that road and nothing that drives on it.

Concretely it delivers one thing: a card-option value the user writes in an Org
PROPERTIES drawer is resolved through the established inheritance chain,
travels the wire as a new protocol version, and is then **validated** — never
acted on, never rendered, never written back.

### 1.2 Scope

In scope: the three Org properties and their resolution; the `protocol.Entry`
fields that carry them; the `protocol.Version` 1 → 2 bump on both sides of the
wire; the Go-side validation gate and its three diagnostic codes; the contract
tests that pin all of it.

Out of scope: every rendering behavior the options eventually select. § 5
enumerates the exclusions.

### 1.3 Reading order

`spec.md` (this file) → `acceptance.md` (the Given-When-Then verification
layer) → `plan.md` (the module-by-module change plan). The design record
`.moai/reports/anki-card-types-plan-20260920.md` § 2 is the source of the
confirmed decisions this SPEC implements and stands in for a Tier L
`research.md`.

## 2. Glossary

**Card-option property** — one of exactly three Org properties this SPEC
introduces: `ANKI_DIRECTION`, `ANKI_INCREMENTAL`, `ANKI_SWIFT`. Named as a set
wherever an obligation binds all three.

**Nearest-wins chain** — the existing three-level property resolution
implemented by `modules/org/anki/imoogi-props.el`: the heading's own PROPERTIES
drawer, then the nearest ancestor heading's own drawer, then the file-level
`#+PROPERTY:` keyword. First hit wins; values never merge across levels; a
present-but-empty value **terminates** the chain rather than falling through.
`ANKI_DECK` and `ANKI_TAGS` already resolve this way.

**Cloze-style note type** — a declared `ANKI_NOTE_TYPE` value that names either
the stock `Cloze` or the imoogi-owned `imoogi-Cloze`. This is the same two-name
set the front end's existing cloze predicate accepts.

**Card kind** — which of the later card shapes an entry's options select.
Inferred from *which* options are present, never declared: `ANKI_DIRECTION` or
`ANKI_INCREMENTAL` implies a multiline card, `ANKI_SWIFT` implies a swift card.
There is no `ANKI_CARD_KIND` property. This SPEC infers no kind and acts on
none; the term exists here only so § 5's exclusions can name what they exclude.

**Option-bearing entry** — a sync target for which at least one card option is
**on**: `ANKI_DIRECTION` naming one of its three arrows, or `ANKI_INCREMENTAL`
or `ANKI_SWIFT` in its truthy spelling. An option in its falsy spelling is off,
and an
entry carrying only falsy options is **not** option-bearing — which is what
lets a heading opt out of an inherited option without thereby claiming to be a
card kind it is not.

Distinct from **carrying a card-option value**, which means only that a field
arrived non-null. Value recognition (§ 3.3 REQ-OPT-009) binds every non-null
value; the conflict and note-type rules bind option-bearing entries only.

**Truthy / falsy spelling** — the two recognized values of a boolean
card-option property: `t` and `nil` respectively, compared after trimming
surrounding whitespace and without regard to letter case.

**Falsy value** — the falsy spelling `nil` on any of the three card-option
properties, including `ANKI_DIRECTION`, where it means "no direction" rather
than naming one. It is recognized on all three so that the explicit opt-out
spelling is uniform: a user suppressing an inherited option writes `nil`, and
does not have to remember which properties accept it.

## 3. Requirements (GEARS)

Thirteen requirements across four groups, `REQ-OPT-001`..`REQ-OPT-013`,
contiguous. The `OPT` infix keeps them clear of the parent SPEC's `REQ-0NN`
namespace and of SPEC-ANKICARD-001's `REQ-C-0NN`, both of which are cited from
live code comments in this repository.

Each requirement carries exactly one GEARS trigger. Numbered sub-clauses are
case-splits on that trigger's operand — they narrow *which* operand the
obligation binds, or name *which actor realizes* it, and never introduce a
second `When`, `While`, or `Where`. `[Ubiquitous — negated]` is the `shall not`
form of Ubiquitous, not a separate pattern.

**On the four `[Where]` requirements.** REQ-OPT-003 and REQ-OPT-009 through
REQ-OPT-011 use `Where` as a **case-split operator over the entry set** — "for
those entries having this property, the obligation is this" — rather than in
its narrower capability-gate reading of a feature flag or a static
configuration switch. This is deliberate and is recorded here rather than left
for a reader to wonder about. `While` was considered and rejected: an entry is
not a thing that persists in a state, so "while an entry carries X" describes
nothing continuous. The four requirements each still carry exactly one trigger
and one PASS/FAIL, which is what the pattern tag exists to guarantee.

### 3.1 Property resolution (front end)

#### REQ-OPT-001 [Ubiquitous]
The front end shall resolve each of the three card-option properties through
the same nearest-wins chain that resolves `ANKI_DECK`:

1. Resolution shall reuse the existing chain implementation rather than
   introduce a second one, so that a change to the chain's semantics cannot
   apply to `ANKI_DECK` and not to a card-option property.
2. A value present at any level but empty shall terminate the chain and
   resolve to no value — identical to the `ANKI_DECK` rule. This form is
   therefore already the per-heading override of an inherited value: a heading
   under a file-level `#+PROPERTY: ANKI_SWIFT t` suppresses it by writing
   `:ANKI_SWIFT:` with an empty value in its own drawer.
3. A property absent at all three levels shall resolve to no value. Absent and
   present-but-empty are indistinguishable downstream, by construction.

#### REQ-OPT-002 [Ubiquitous — negated]
The front end shall not interpret, normalize, default, coerce, or write any
card-option property value:

1. It shall carry the resolved text onward exactly as the drawer or keyword
   spells it, including any value it could recognize as malformed. Deciding
   what a value means is the back end's obligation under § 3.3, and a front end
   that coerced first would make that decision unreachable.
2. It shall not write a card-option property into any drawer, and shall not
   infer one from a note type, a heading shape, or a body.
3. `ANKI_NOTE_TYPE` shall continue to be read from the target heading's own
   drawer only, with no inheritance — this SPEC changes nothing about it, and
   it remains the sole property that decides whether a heading is a sync target
   at all.

#### REQ-OPT-003 [Where]
Where a scanned heading is a sync target, the front end shall attach all three
resolved card-option values to that heading's entry, so that an entry's option
set is fixed at scan time and is not re-derived anywhere downstream.

### 3.2 Wire contract

#### REQ-OPT-004 [Ubiquitous]
The request document's entry shape shall carry the three card-option values as
three new nullable string fields, spelled `direction`, `incremental`, and
`swift`:

1. Each field shall carry the resolved property text verbatim, or JSON `null`
   when REQ-OPT-001.2 or REQ-OPT-001.3 resolved it to no value.
2. No field shall be omitted from the serialized document when its value is
   null, consistent with the existing contract's rule that a dropped key and a
   null value are not interchangeable on the wire.
3. Each field shall carry a **string**, including the two whose recognized
   values are boolean in meaning. A wire-level boolean would require the front
   end to decide which spellings are true, which REQ-OPT-002.1 forbids, and
   would leave the back end unable to name the offending text in a diagnostic.
4. The response document shall gain no field, and no other request field shall
   change shape.

#### REQ-OPT-005 [Ubiquitous]
The wire-contract version shall be **2** on both sides of the boundary — the
back end's compiled-in constant and the front end's declared version — and the
two shall be equal. A change to one without the other is the defect this
requirement exists to make mechanically detectable.

#### REQ-OPT-006 [When]
When the back end receives a request document whose declared protocol version
differs from its own, it shall report that skew through the **existing**
diagnostic gate:

1. The reported code shall be the already-declared `binary_incompatible`.
2. The obligation shall hold in both skew directions — a version-2 front end
   against a version-1 binary, and a version-1 front end against a version-2
   binary — and at every request-reading subcommand.

### 3.3 Validation (back end)

#### REQ-OPT-007 [Ubiquitous]
The back end shall decide whether an entry's declared note type is cloze-style
by deriving it from the existing declared-type-to-renderer mapping rather than
from a second list of note-type names, so that the predicate cannot drift from
the renderer's own dispatch.

1. The predicate shall answer true for exactly the stock `Cloze` and the
   imoogi-owned `imoogi-Cloze`, and false for every other declared value.
2. The front end's existing cloze predicate shall remain the mirror of this
   one, accepting exactly the same two names.

#### REQ-OPT-008 [Ubiquitous]
The back end shall validate an option-bearing entry through a single gate,
placed and ordered as follows:

1. The gate shall run **before** the entry is rendered, so that a mis-specified
   option is reported against the properties the user wrote rather than
   surfacing later as a rendering or field-resolution failure.
2. Exactly one diagnostic shall be emitted per rejected entry, selected in this
   fixed order: `card_option_invalid` (§ 3.3 REQ-OPT-009), then
   `card_option_conflict` (REQ-OPT-010), then `card_option_needs_cloze`
   (REQ-OPT-011). The order is fixed so that an entry offending against more
   than one rule has one defined outcome.
3. A rejected entry shall be reported as **skipped**, carrying its existing
   note identifier where it has one, and shall leave the collection, the
   registry, and the Org heading untouched.
4. The same gate shall govern the ordinary synchronization path and the
   migration path, from one implementation. On the migration path the gate
   shall read the entry's **declared** note type, which names the same
   cloze-style answer as its migration counterpart for every reachable case,
   since a type and its imoogi-owned counterpart are cloze-style together or
   not at all.
5. The migration-path obligation of sub-clause 4 shall bind **confirmed
   migration candidates only**. A migration run reaches its per-entry pipeline
   only for an entry it has already admitted as a candidate; an entry it
   rejects as a non-candidate, and every entry under a dry run, returns before
   that pipeline and shall therefore raise no option diagnostic. Both arms are
   pre-existing shapes this SPEC inherits rather than introduces, and neither
   loses coverage: an entry not migrated is still processed by the ordinary
   path, where the gate does run.
6. An entry for which no card-option property resolved to a value shall reach
   the gate unchanged in behavior — no diagnostic, no skip, no request the
   binary would not otherwise have issued.

#### REQ-OPT-009 [Where]
Where an entry carries a non-null card-option value the back end does not
recognize, it shall reject that entry with the diagnostic code
`card_option_invalid`:

1. The recognized values of `direction` shall be exactly the three arrows
   `->`, `<-`, and `<->`, plus the falsy spelling `nil`; the arrows shall be
   compared after trimming surrounding whitespace.
2. The recognized values of `incremental` and of `swift` shall be exactly the
   truthy spelling `t` and the falsy spelling `nil`.
3. The falsy spelling shall be recognized on all three properties, compared
   after trimming surrounding whitespace and without regard to letter case, and
   shall mean the option is off — it therefore does not make its entry
   option-bearing, and so triggers neither REQ-OPT-010's conflict rule nor
   REQ-OPT-011's note-type rule. Accepting it on all three is what makes the
   explicit opt-out spelling uniform, so that a user suppressing an inherited
   option need not remember which properties accept which word.
4. The diagnostic's machine-oriented detail shall name the offending property
   and its actual value, so that the user can find the drawer line that carries
   it.

#### REQ-OPT-010 [Where]
Where an entry resolves `swift` to its truthy spelling **and** also carries a
multiline option that is on — `direction` naming one of its three arrows, or
`incremental` in its truthy spelling — the back end shall reject that entry
with the diagnostic code `card_option_conflict`.

The two option groups select mutually exclusive card kinds, and no precedence
between them is defined anywhere. Silently choosing one would make the
resulting card differ from what the user wrote with no signal, which is the
outcome this rejection exists to prevent. Because all three properties inherit,
the inherited arm of such a conflict is suppressed per REQ-OPT-001.2.

#### REQ-OPT-011 [Where]
Where an option-bearing entry's declared `ANKI_NOTE_TYPE` is not a cloze-style
note type, the back end shall reject that entry with the diagnostic code
`card_option_needs_cloze`.

Every card kind the options select is built on a cloze-style note type, so an
option on any other type describes a card that cannot be produced. Reporting it
here — rather than letting the entry synchronize as though the options were
absent — is what keeps a user's stated intent from being silently discarded.

### 3.4 Non-interference and diagnostics

#### REQ-OPT-012 [Ubiquitous — negated]
The back end shall not change what any entry renders to:

1. The renderer's entry point shall gain no parameter, and its output field map
   shall be byte-identical for any given note type, title, and body regardless
   of what the three new fields carry.
2. The content hash's input set shall gain no member, and the hash of an
   unchanged entry shall be unchanged, so that this SPEC causes no entry to be
   reported as updated on account of content that did not change.
3. An entry declaring a cloze-style note type, carrying valid card options, and
   carrying no cloze marker in its title or remaining body shall continue to be
   skipped with the existing `cloze_marker_missing` diagnostic. This SPEC
   introduces no marker the options generate.

#### REQ-OPT-013 [Ubiquitous]
Each diagnostic code this SPEC introduces shall be paired with a user-facing
message in the front end's code-to-message table, which remains the only
component that renders prose for a code:

1. Each message shall name the problem and state a corrective action, and shall
   contain no stack trace, raw transport error, backtrace, or bare exit code.
2. The existing two-directional pairing assertion shall continue to hold over
   the enlarged code set — every back-end code has a table entry, and every
   table entry that is not on the documented front-end-only allowlist has a
   back-end constant.

## 4. Relationship to SPEC-ANKICARD-001

SPEC-ANKICARD-001 is a **dependency, not a parent**: this SPEC amends none of
its requirements, supersedes none of its exclusions, and changes no behavior it
introduced. The dependency is ordering only — SPEC-ANKICARD-001 owns the
imoogi-owned note types (`imoogi-Basic`, `imoogi-Cloze`) whose names
REQ-OPT-007 reads, and all of those names are already on `main`. Its closure
is **not** a precondition of this SPEC's run phase; § 8 carries that
disposition and the authorization behind it.

Two of its surfaces this SPEC extends without altering:

- The declared-type-to-renderer mapping it introduced is the derivation source
  for REQ-OPT-007's predicate. The mapping itself is unchanged.
- Its diagnostic-code const block and the front end's code-to-message table are
  the surfaces REQ-OPT-013 enlarges, through the pairing contract it already
  established.

## 5. Out of Scope

This SPEC deliberately renders nothing new. The exclusions below are what makes
that claim checkable: each names work that an implementation could plausibly
fold in, and states where it actually belongs.

### Out of Scope — Multiline card rendering

- Composing a heading title and its top-level body list into a cloze-wrapped
  question-and-answer card. Backlog card **t14**.
- The cloze-numbering rules a direction selects — wrapping answer items for
  `->`, the title for `<-`, both for `<->`, and per-item numbering when
  `incremental` is on.
- The `multiline_answer_missing` diagnostic for a multiline entry with no
  answer items, and the `.children-list` card styling. Both are t14's.
- Whether `ANKI_INCREMENTAL` on its own, with no `ANKI_DIRECTION` beside it,
  implies a default direction. It is valid transport here and its rendering
  meaning is t14's to define; this SPEC states no default.

### Out of Scope — Swift card rendering

- Splitting an entry body's arrow lines into per-line cloze-numbered cards, and
  the `swift_arrow_missing` diagnostic for a swift entry carrying no arrow
  line. Backlog card **t15**.
- Recognizing the in-body arrow forms `:->`, `:<-`, and `:<->` at all. This
  SPEC reads arrows only as `ANKI_DIRECTION` property values, never in a body.

### Out of Scope — Passing options into the renderer

- Widening the renderer's entry point to accept the card options, on either the
  synchronization or the migration call site. Both call sites keep their
  current arguments here. The first card that renders an option owns that
  widening, and owns re-examining whether the migration path must pass options
  too.

### Out of Scope — Editor commands that write the options

- Any interactive command that sets `ANKI_DIRECTION`, `ANKI_INCREMENTAL`, or
  `ANKI_SWIFT` in a drawer, and any keybinding or transient entry for one. The
  design record § 2.1 principle 3 places note-type and option authoring at edit
  time, in commands t14 and t15 introduce.
- Registering the three names as property-name completion candidates in the
  editor convenience layer. That surface belongs with the commands that write
  them.

### Out of Scope — A declared card-kind property

- An `ANKI_CARD_KIND` property, or any other explicit kind declaration. The
  design record § 2.3 decision D settled kind as inferred from which options
  are present, and this SPEC infers no kind at all.

### Out of Scope — Inheritance-mechanism changes

- Altering the nearest-wins chain, its present-but-empty termination rule, or
  its `PROPERTY+` normalization. REQ-OPT-001.1 requires reuse precisely so that
  this SPEC cannot change the chain's behavior for `ANKI_DECK` or `ANKI_TAGS`.
- Making `ANKI_NOTE_TYPE` inherit. It continues not to.

### Out of Scope — Replacement-cloze and image-occlusion card kinds

- The regular-expression-driven replacement cloze and the image-occlusion card
  kind the design record § 2.2 records as deferred and excluded respectively.
  Neither has a card-option property here.

## 6. Constraints

1. **No new note type.** Every card kind these options eventually select is
   built on the existing cloze-style note types. Nothing in this SPEC creates,
   updates, or names a new Anki model.
2. **The back end validates; it never mutates.** The declared note type is
   written at edit time by the front end. No code path here changes an entry's
   `ANKI_NOTE_TYPE`, writes a property, or edits an Org file.
3. **Version skew rides the existing gate.** REQ-OPT-006 forbids a second
   mechanism; the version bump is a constant change plus its pins, not a
   protocol negotiation.
4. **One diagnostic per rejected entry.** REQ-OPT-008.2's fixed order is what
   makes a multiply-offending entry testable.
5. **Byte-identical rendering.** REQ-OPT-012 is the constraint the whole SPEC
   is shaped around: an entry that changes no content must produce the same
   hash it produced before, so that no user sees a mass of spurious updates.

## 7. Brownfield Delta

Per-surface disposition. `[NEW]` marks an artifact this SPEC creates;
`[CHANGED]` an existing one it modifies; `[UNCHANGED]` one it deliberately
leaves alone despite sitting on the path.

### [CHANGED] Protocol (`internal/anki/protocol`)

The entry shape gains the three fields of REQ-OPT-004. The version constant
goes to 2 (REQ-OPT-005). The diagnostic-code const block gains three codes
(REQ-OPT-009, REQ-OPT-010, REQ-OPT-011). The response shape gains nothing.

### [CHANGED] Planner (`internal/anki/planner`)

Gains the cloze-style predicate of REQ-OPT-007, derived from the existing
declared-type mapping, and the single validation gate of REQ-OPT-008 reached
from both the ordinary and the migration entry paths. The field-resolution
layer, the deck fallback, and the add/update/no-op partition are untouched.

### [UNCHANGED] Renderer (`internal/anki/orgdoc`)

Deliberately untouched. REQ-OPT-012.1 makes its signature and output stability
an obligation rather than an accident, and § 5 names the widening as a later
card's work.

### [UNCHANGED] Content hashing (`internal/anki/hashing`)

Deliberately untouched. Its input set is note type, rendered field values,
resolved deck, and sorted tags; the card options reach none of the four, which
is what REQ-OPT-012.2 asserts.

### [UNCHANGED] CLI (`cmd/imoogi-anki`)

The version-skew probe is already shared by every request-reading subcommand
and already compares against the protocol constant, so the bump reaches all of
them with no edit to the command layer. Its version-pinning tests change.

### [CHANGED] Property resolution (`modules/org/anki/imoogi-props.el`)

Gains resolver entry points for the three card-option properties, built on the
existing chain function (REQ-OPT-001.1). The chain itself is unchanged.

### [CHANGED] Scan (`modules/org/anki/imoogi-scan.el`)

The single entry-plist producer gains the three resolved values (REQ-OPT-003).
The multi-target grouping layer above it delegates parsing here and needs no
change of its own.

### [CHANGED] Process (`modules/org/anki/imoogi-process.el`)

The declared protocol version goes to 2 (REQ-OPT-005); the entry serializer
emits the three new keys with a null for an unresolved value (REQ-OPT-004).

### [CHANGED] Diagnostics (`modules/org/anki/imoogi-error.el`)

Gains three code-to-message entries (REQ-OPT-013).

### [CHANGED] Tests

The version-pinning tests on both sides of the wire, the code-pairing tests
including the hardcoded back-end code list they cross-check, the property and
scan tests, and new planner tests for the validation gate.

### [CHANGED] Documentation (`README.md`)

The Anki section gains the three properties, their recognized values, their
inheritance, and the note-type requirement.

## 8. Dependencies

- **SPEC-ANKICARD-001** — an ordering dependency only (§ 4). Its closure is
  **not** a precondition of this SPEC's run phase.

  Nothing this SPEC reads from it is unlanded. Every surface named in § 4 and
  § 7 is on `main` today: the imoogi-owned type names, the
  declared-type-to-renderer mapping REQ-OPT-007 derives its predicate from,
  the diagnostic-code const block REQ-OPT-013 enlarges, the bidirectional
  pairing test that enforces it, and the front end's mirror predicate. The
  dependency is process — SPEC-ANKICARD-001's own sync audit is unclosed — not
  code availability.

  Proceeding ahead of that closure is authorized. The design record's ordering
  note records that the operator, **told that SPEC-ANKICARD-001 was still
  `in-progress`**, directed that cards t11 through t15 be worked in order —
  a range that names this card. The operator repeated that directive in the
  session that produced this SPEC, again with the blocker stated first. The
  design record § 5 risk table predates both and is superseded on this point;
  it is not read here as a gate.

  The consequence the ordering note itself draws still holds and is recorded
  rather than mitigated: SPEC-ANKICARD-001's sync audit will run against a
  HEAD this SPEC has moved.
- **Backlog card t12** — already landed on `main`. Its supplementary-field work
  is what makes REQ-OPT-012.2's byte-identity baseline the post-t12 tree rather
  than an older one.
- No new third-party dependency.

## 9. Traceability

Every requirement is covered by at least one criterion in `acceptance.md`;
every criterion names the requirements it verifies and the command that decides
it. `plan.md` § F maps requirements onto milestones.
