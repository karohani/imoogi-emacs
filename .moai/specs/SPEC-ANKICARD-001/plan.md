# Plan — SPEC-ANKICARD-001

## Revision 4 — 2026-09-05 — trigger hygiene and measurability

Fourth annotation cycle, after plan-audit iteration 1 returned PASS 0.96 and
three read-only review lenses (requirement testability, acceptance criteria,
scope coherence) were run against revision 3. Every finding folded in here was
non-blocking; the audit verdict is not reopened. **The requirement count is
unchanged at 24 and the acceptance count unchanged at 24 top-level** — this is a
re-partition, not a growth.

**The two defects that forced the re-partition.** Former REQ-C-022 carried a
disjunctive header ("a mismatch **or** an add failure") over two sub-clauses that
open two different moments; former REQ-C-021 carried a second `When` ("when a
migration completes") nested under a `Where` gate. Both broke the SPEC's own
stated rule that a sub-clause is a case-split on its requirement's single
trigger, and both are the same defect class the plan-auditor recorded as D3. The
split is paid for by folding former REQ-C-018 into REQ-C-016: the first-run mass
invalidation adds no obligation the post-transform hash decision does not already
entail, so it is now a case-split on that decision rather than a requirement of
its own.

**Old → new mapping (v0.1.1 → v0.1.2).** `REQ-C-001`..`REQ-C-017` keep their
identifiers; only the migration group moves.

| v0.1.1 | v0.1.2 | Disposition |
|---|---|---|
| REQ-C-001 .. REQ-C-015 | **unchanged** | REQ-C-005, REQ-C-007, REQ-C-008, REQ-C-012, REQ-C-013, REQ-C-015 rewritten in place |
| REQ-C-016 | **REQ-C-016** | gains sub-clause 2 (the folded REQ-C-018) |
| REQ-C-017 | **REQ-C-017** | unchanged |
| REQ-C-018 | **folded into REQ-C-016.2** | no obligation dropped |
| REQ-C-019 | **REQ-C-018** | renumbered; gains the `migrate_candidate` dry-run shape |
| REQ-C-020 | **REQ-C-019** | renumbered; rationale sentence moved to `design.md` § 7.1 |
| REQ-C-021 | **REQ-C-020** | renumbered; write-back folded under the single add `When`; single subject |
| REQ-C-022 sub-clause 1 | **REQ-C-021** | split out — note-type mismatch, one detected event |
| REQ-C-022 sub-clause 2 | **REQ-C-022** | split out — migration add failure, one detected event |
| REQ-C-023, REQ-C-024 | **unchanged** | REQ-C-024.3 and .4 rewritten in place |

Totals: 24 in, 24 out — 17 identifiers untouched, 1 fold, 4 renumberings, 1 split
into 2. **Pattern census after the re-partition**, stated identically in
`spec.md` HISTORY and `spec-compact.md`: Ubiquitous 13 (of which 6 negated),
event-driven `When` 4, state-driven `While` 2 (of which 1 negated),
capability-gate `Where` 2, event-detected 3 — 13 + 4 + 2 + 2 + 3 = 24. The
negated `shall not` form is the EARS-legacy `Unwanted` label and is not a sixth
GEARS pattern. Acceptance criteria stay at 24 top-level; `AC-C-006` gains a fourth
sub-criterion (`AC-C-006d`, the Elisp → Go stylesheet transport) and `AC-C-021`
gains `AC-C-021c`, so the ceiling is not approached.

**What else changed, and why.** The ownership prohibition of REQ-C-024.3 no
longer forbids **reading** a foreign note type — as written it contradicted the
`modelNames` probe REQ-C-002 mandates and the `modelFieldNames("Basic")` read the
planner's field-resolution layer performs on every stock-typed heading.
REQ-C-024.4 narrowed from "any note carried on them" to the stock **types,
templates, and CSS**, matching § B.4's actual scope, so it no longer contradicts
REQ-C-005.2's ordinary sync. REQ-C-007.2's aesthetic vocabulary was replaced by
seven named CSS custom properties with literal values. `migrate --dry-run` now
reports through ordinary `results[]` entries carrying a new `action` value
`migrate_candidate` rather than the invented `{candidates, entries}` envelope
`design.md` § 7.1 previously drew, which contradicted REQ-C-018's no-new-field
clause.

**One correction to a lens finding, made on direct measurement.** The acceptance
lens reported package coverage as `planner` 90.3 and `ankiconnect` 84.3, and
recommended floors of ≥ 90.3 and ≥ 85.0 respectively. Running `go test -count=1 -cover ./internal/anki/...` in this repository at `HEAD f6ee148` (2026-09-05) returns `planner` **90.3**, `ankiconnect` **84.3**, `registry` 87.8, `hashing` 100.0, `orgdoc` 100.0 — the lens figures were correct for this repository (an earlier draft quoted the parent checkout at `413516e`, which lacks this repository's field-resolution and duplicate-id code). Floors: `planner` ≥ 90.3 (its measured figure, so it may not regress) and `ankiconnect` ≥ 85.0 (the run-phase norm, above the measured 84.3 — M1 must lift it).

---

## Revision 3 — 2026-09-05 — budget consolidation

Third annotation cycle, plan-phase pre-audit. **A renumbering, not a scope cut.**
`.claude/rules/moai/workflow/spec-workflow.md` §148-152 caps a Tier L SPEC at 25 requirements
and 25 acceptance criteria independently, and states that exceeding the cap "is a signal to tier
up or to split, not to relax the budget". Revision 2's §C carried **42** requirements; this
revision consolidates them to **24**. No obligation was dropped: every one of the 42 survives
either inside a merged requirement's numbered sub-clauses or — for the four entries of the
former §C.8 "non-goals carried as requirements" — as non-normative exclusion prose that §5 of
`spec.md` already carried. The acceptance set was already within budget at 24 top-level AC and
is unchanged in count; only its `covers REQ-…` pointers were re-pointed.

**What merged, and on what principle.** Requirements were merged only where they bind the same
actor under the same GEARS trigger, so every merged requirement still matches exactly one GEARS
pattern with its sub-clauses sharing that single trigger. The former §3.8 non-goals collapsed
into one Unwanted requirement (REQ-C-024) carrying their four testable prohibitions; the purely
declarative remainder — image-occlusion, highlight-mask, type-in-answer, and card-direction
cards — is carried by `spec.md` §5, where it already lived as an exclusion bullet.

One judgment call worth naming: former REQ-C-023 (Ubiquitous, "issue `storeMediaFile` only for
entries the run actually adds or updates") is recast under former REQ-C-027's `While` run scope
as REQ-C-014. This is meaning-equivalent because 023's own text was already run-scoped — it
speaks of "entries the run actually adds" — so the `While` modifier makes explicit a scope the
original sentence already carried.

**Pattern coverage after consolidation** (all five GEARS patterns retained): Ubiquitous 7,
event-driven `When` 5, state-driven `While` 2, capability-gate `Where` 2, Unwanted 6,
event-detected 2.

**Old → new mapping (42 → 24).** Historical artifacts — `interview.md`, `research-lenses.md`,
`review-lenses-1.md` — deliberately retain their v0.1.0 identifiers as the record of what was
approved at Decision Point 1; read them through this table. `spec.md`, `acceptance.md`,
`design.md`, `research.md`, `spec-compact.md`, and this file carry the new identifiers.

| Old (v0.1.0) | New (v0.1.1) | Disposition |
|---|---|---|
| REQ-C-001, REQ-C-006 | **REQ-C-001** | merged — ownership predicate + field-shape parity |
| REQ-C-002, REQ-C-003, REQ-C-015 | **REQ-C-002** | merged — one install `When`, three sub-clauses |
| REQ-C-004 | **REQ-C-003** | renumbered |
| REQ-C-005 | **REQ-C-004** | renumbered (parent REQ-016 amendment anchor, kept separate) |
| REQ-C-007, REQ-C-008, REQ-C-038 | **REQ-C-005** | merged — note-type naming across all surfaces |
| REQ-C-009, REQ-C-010 | **REQ-C-006** | merged — deck wrapper + normalization |
| REQ-C-011, REQ-C-012 | **REQ-C-007** | merged — stylesheet properties |
| REQ-C-013 | **REQ-C-008** | renumbered |
| REQ-C-014 | **REQ-C-009** | renumbered |
| REQ-C-016, REQ-C-017, REQ-C-018 | **REQ-C-010** | merged — one math `When`, delimiter table + heuristic + `<br>` |
| REQ-C-019 | **REQ-C-011** | renumbered |
| REQ-C-020, REQ-C-021, REQ-C-022 | **REQ-C-012** | merged — resolve / stored-name / two-part rewrite |
| REQ-C-024, REQ-C-026 | **REQ-C-013** | merged — media prohibitions |
| REQ-C-023, REQ-C-027 | **REQ-C-014** | merged — run-scoped gating + dedup |
| REQ-C-025 | **REQ-C-015** | renumbered |
| REQ-C-028 | **REQ-C-016** | renumbered (transform-before-hash, kept sharply separate) |
| REQ-C-029 | **REQ-C-017** | renumbered (no new hash inputs, kept sharply separate) |
| REQ-C-030 | **REQ-C-018** | renumbered |
| REQ-C-031 | **REQ-C-019** | renumbered |
| REQ-C-032 | **REQ-C-020** | renumbered |
| REQ-C-033, REQ-C-035 | **REQ-C-021** | merged — confirmed-migration add + registry/property write-back |
| REQ-C-034, REQ-C-036 | **REQ-C-022** | merged — the two detected migration failure conditions |
| REQ-C-037 | **REQ-C-023** | renumbered (Go + Elisp diagnostic pairing) |
| REQ-C-039, REQ-C-040, REQ-C-041, REQ-C-042 | **REQ-C-024** | merged — one Unwanted requirement; REQ-C-041's advanced-card-type clause carried by `spec.md` §5 prose |

Totals: 42 old → 24 new. **12 merges** absorbing 30 old identifiers, plus **12 straight
renumberings** carrying one old identifier each — 30 + 12 = 42, every old identifier accounted
for exactly once. One of the 30 absorbed identifiers, REQ-C-041, is additionally split: its
note-type prohibition became a sub-clause of REQ-C-024 while its advanced-card-type clause is
carried only by the non-normative §5 prose that already held it. Acceptance criteria: 24,
unchanged in count, all re-pointed; every one of the 24 new requirements is covered by at least
one AC.

**Frontmatter.** `version` → `0.1.1`; `updated` → 2026-09-05. `related_specs` removed — it is not
in the schema's optional-field table and `amendment_of: SPEC-ANKI-001` already carries the link.
`tier`, `amendment_of`, the `### Amendments` sub-section, and the recorded-but-unavailable
`prior_completed_sha` are unchanged.

**Not changed by this revision.** §D design decisions `D-C-1`..`D-C-15`, §E touch surfaces, §F
milestones `M1`..`M6`, §G test strategy, §H risks `R1`..`R15`, and §I tier rationale carry only
re-pointed requirement identifiers; no decision, milestone, or risk disposition was re-litigated.
The Decision Point 1 approval of revision 2 stands.

---

## Revision 2 — 2026-09-05

Second annotation cycle. Three classes of change. **(1) Clarifications closed.** The eight §H rows
(R1-R8) that carried revision 1's ten open-clarification marker strings are all resolved from
`interview.md` 라운드 3; the file now carries zero markers. R1 in particular is no longer an
unknown but a **measurement**: a probe note carrying reordered attributes, `&amp;`/`&lt;`
entities, multiple spaces, a self-closing `<img … />`, and both MathJax delimiter forms was
added to the live collection, read back through `notesInfo`, compared byte-for-byte, and
deleted — Anki returned the stored field HTML unchanged. The revision-1 R1 fallback design is
therefore deleted rather than deferred. **(2) Scope narrowed by user decision.** Per-deck
appearance is now **user-authored only**: imoogi emits the normalized `deck-<class>` wrapper and
ships one base stylesheet identical for every deck, with no automatic palette assignment. The
revision-1 palette-rotation text is deleted from §C and §D and survives only as a REJECTED
alternative (D-C-3). Video and audio move to an explicit non-goal. **(3) Surfaces chosen.**
Installation extends `imoogi-anki-setup` rather than adding a new command; migration travels as
a `imoogi-anki migrate` CLI subcommand with **no new wire field and `protocol.Version`
unchanged**; the default note type for newly marked headings becomes `imoogi-Basic` /
`imoogi-Cloze`, with hand-written stock headings staying backward-compatible but unthemed.
**Numbering.** §C grew from 35 to 42 requirements and §D from 12 to 15 decisions; both were
renumbered contiguously, so `REQ-C-0NN` and `D-C-N` identifiers above the insertion points
have shifted relative to revision 1. Revision-1 identifiers should not be cited.

**Approved at Decision Point 1 on 2026-09-05** — revision 2 accepted as-is; no decision re-litigated at Phase 10.

---

| Field | Value |
|---|---|
| id | `SPEC-ANKICARD-001` |
| title | imoogi-owned Anki note types — themed card presentation, MathJax, and media |
| status | draft |
| created | 2026-09-05 |
| updated | 2026-09-05 (revision 3 — budget consolidation, 42 REQ → 24) |
| author | jay |
| module | `internal/anki/{ankiconnect,orgdoc,planner,protocol,model,media}`, `cmd/imoogi-anki/`, `modules/24-anki.el`, `modules/anki/{imoogi-setup,imoogi-error,imoogi,imoogi-process}.el`, `tests/anki-*.el` |
| amends | `SPEC-ANKI-001` (out-of-repo, `/Users/jay/workspace/imoogi-org-anki/`) — REQ-016, AC-016, §4 |
| tier | **L** (confirmed at Phase 9; see §I) |

> **Why a table, not YAML frontmatter.** `SPEC-TRANSIENT-001/plan.md` — this repo's only
> plan precedent — carries no YAML frontmatter at all; the canonical 12-field block lives
> in `spec.md`. Emitting a partial 5-field YAML block on `plan.md` would invite a
> `FrontmatterInvalid` finding against a file the schema rule does not govern. The mission's
> named fields are therefore carried as a metadata table under the H1, and the full 12-field
> schema is deferred to `spec.md` (Phase 10).

---

## §A Context

`SPEC-ANKI-001` (Tier L, 22 REQ / 25 AC, 6 milestones, plan-audit PASS 0.857 on iteration 5)
shipped one-way Org → Anki sync using the **stock** `Basic` and `Cloze` note types. Cards
therefore inherit whatever styling the user's collection already carries, math reaches Anki
as literal `$…$` that MathJax never renders, and a local `[[file:diagram.png]]` reaches Anki
as an `<img src>` pointing at a path Anki cannot resolve. This SPEC adds imoogi-owned note
types carrying an imoogi-authored light+dark theme with per-deck styling hooks, converts org
math to Anki's MathJax delimiters, and uploads local images into Anki's media collection.

**Repository and parent-SPEC boundary.** The parent SPEC's artifacts live in a *different
checkout* (`/Users/jay/workspace/imoogi-org-anki/.moai/specs/SPEC-ANKI-001/`); its code lives
here. This SPEC **does not edit the parent's files**. The amendments in §B are normative text
carried in *this* SPEC, expressed in `spec.md`'s frontmatter (Phase 10) as
`amendment_of: SPEC-ANKI-001` plus `related_specs`, and enforced by this SPEC's own
acceptance criteria. Recorded, not acted on: the parent's frontmatter still reads
`status: in-progress` (v0.7.1) although its `progress.md` records 25/25 AC pass and six
completed milestones — treat the parent as behaviorally closed and specification-live.

**Methodology.** `tdd` (RED-GREEN-REFACTOR) per `.moai/config/sections/quality.yaml`
`constitution.development_mode`, matching `SPEC-TRANSIENT-001` §A. Route: Hybrid Trunk
`main`-direct unless Tier L selects a PR route at Phase 9.

**Binding inputs.** `interview.md` (14 confirmed decisions across rounds 1-3, plus the SPEC-ID
correction — none reopened here), `research.md` (13 sections + verdict table + evidence index),
`research-lenses.md` (per-lens raw reports). Section references below in the form
`research.md §N` and `C1`-`C6` point at those documents. Round 3 closed every open question
revision 1 carried as an open-clarification marker; §H now records dispositions, not
markers.

---

## §B Parent-SPEC Amendment (read first — this is the blocking item)

`research.md §1` is unambiguous: **the feature this SPEC proposes is forbidden by the parent
SPEC as that SPEC is currently written.** Nothing in §C-§F is implementable until the three
dispositions below are carried in `spec.md`. All four research lenses converged here
independently.

### B.1 The ownership predicate (defined once, used everywhere)

Every clause below turns on one predicate, stated here so the amendment, the install
requirement, and the replacement acceptance criterion cannot drift apart:

> **imoogi-owned note type**: an Anki note type whose model name begins with the literal
> ASCII prefix `imoogi-`. Every other note type in the collection — including the stock
> `Basic` and `Cloze` types, and any user-authored type — is **foreign**.

The predicate is name-based rather than registry-based on purpose: it is decidable from a
single `modelNames` response with no local state, so the run-phase assertion "this request
targeted an imoogi-owned model" is checkable from a request log alone. Reused verbatim in
REQ-C-001, REQ-C-002, REQ-C-003, REQ-C-004, and the AC-016 replacement.

### B.2 REQ-016 amendment — scope the prohibition to foreign note types

Parent REQ-016 (`spec.md:177`), verbatim, prohibits modifying "note-type templates, or
collection styling, on any code path". `createModel` / `updateModelStyling` /
`updateModelTemplates` are literally the actions it names, and the text carries **no
ownership carve-out**.

**Amended text (this SPEC's REQ-C-004 supersedes the quoted clause of parent REQ-016):**

> The imoogi system shall not modify Anki review history, card scheduling state, **the
> templates or styling of any note type it did not itself create — that is, any note type
> whose model name does not begin with `imoogi-`** — or collection-level styling, on any code
> path, including the update and delete paths, and shall not issue any delete request scoped
> by tag, search query, or pattern rather than by explicit individual note identifier.

Everything else in REQ-016 — the scheduling prohibition, the review-history prohibition, the
identifier-scoped-deletion prohibition — carries over **unchanged and unweakened**. Only the
note-type-template and styling clause narrows, and it narrows by ownership, not by
convenience.

### B.3 AC-016 replacement — from a per-run blanket to an ownership-scoped pair

Parent AC-016 (`acceptance.md:102`) ends: "…and no request in the run addresses any
scheduling, review-history, note-type-template, or collection-styling endpoint." That is a
**per-run** assertion. Any model write anywhere in the run breaks it as written. Note the
contrast the research draws with AC-015 (`acceptance.md:95-97`), which counts only
"note-mutating requests and … card-moving requests" — so a read-only `modelNames` probe is
already AC-015-safe while a template write is not AC-016-safe.

The replacement is **two** assertions, not one, because the blanket survives for the hot path
and is only lifted on the install path — which, per D-C-6, is the tail of `imoogi-anki-setup`
and never `imoogi-sync`:

- **AC-C-003a (sync hot path — the blanket survives, verbatim in force).** Given a sync run of
  any shape (add, update, deck move, delete, no-op), when the run executes, then the count of
  requests recorded against `createModel`, `updateModelStyling`, `updateModelTemplates`, and
  every scheduling / review-history / collection-styling endpoint is **exactly 0**.
- **AC-C-003b (install path — ownership-scoped).** Given an invocation of `imoogi-anki-setup`'s
  model-install step, when it executes, then every recorded `createModel` /
  `updateModelStyling` / `updateModelTemplates` request names a model whose name begins with
  `imoogi-`, and the count of such requests naming any other model is **exactly 0**.

Both are mechanically checkable today: `planner/fake_client_test.go:17-50` already logs calls
per action (`addCalls`, `createDeckCalls`, …), so each new connector method arrives with a
matching call log and "no request fired" stays assertable (`research.md §2`, §9).

### B.4 §4 supersession — two exclusions, handled differently

- **"Custom (non-built-in) Anki note types"** (parent `spec.md:213-219`, reinforced by the
  glossary at `:58` and by C-3/REQ-003/REQ-004). **SUPERSEDED by this SPEC.** SPEC-ANKICARD-001
  is the later SPEC that exclusion anticipated. Note the exclusion is superseded *only* for
  imoogi-authored types: user-authored custom note types remain entirely out of scope, and
  REQ-C-004 keeps them untouchable.
- **"Collection ownership guarantees"** (parent `spec.md:258`): "A guarantee that user-authored
  note-type templates and collection CSS are preserved as a specified feature. REQ-016 forbids
  imoogi from touching them; formalizing a persistence and restoration guarantee is separate
  work." This SPEC **inherits that deferred obligation**, because it now does touch models. The
  guarantee it must supply is the narrow one, and only the narrow one: the stock `Basic`
  (measured: 88 notes) and `Cloze` (measured: 27 notes) types, their templates, and their CSS
  are byte-unchanged by every imoogi code path. A general restoration-and-backup guarantee
  remains out of scope.

### B.5 D-12 assumption — this SPEC becomes the "later SPEC"

Parent `plan.md D-12`, verbatim: "**Automating that migration — delete the old note, add a new
one under the new type, update the registry and `ANKI_NOTE_ID` accordingly — is deferred to a
later SPEC.**" Per `interview.md` round 2 decision #6, SPEC-ANKICARD-001 assumes that deferred
work for the narrow, bounded case it creates: the imoogi-created notes sitting on stock types
(measured: 5). §F M5 and REQ-C-018..REQ-C-022 carry it.

Parent REQ-021 (`spec.md:196`) is **not** repealed. It keeps governing every note-type change
the user authors by hand. D-C-13 makes the two disjoint in requirement text rather than in
branch ordering — the parent's own recurring audit defect (`research.md §10`).

---

## §C Requirements Outline (GEARS, `REQ-C-` namespace)

Numbering uses the `C` infix so this SPEC's identifiers never collide with the parent's
`REQ-0NN` / `AC-0NN` / `D-N` namespace, which is cited from live code comments in this repo.
These are outline-grade; `spec.md` carries the final wording. **Consolidated to 24 at
revision 3** — a renumbering, not a scope cut; see the mapping table under the H1. Sub-clauses
share their requirement's single GEARS trigger and are not independent requirements.

### C.1 Note types and installation

- **REQ-C-001** [Ubiquitous] Ownership predicate and field-shape parity: a note type is imoogi's
  own exactly when its model name begins with `imoogi-` (exactly `imoogi-Basic`, `imoogi-Cloze`);
  those types carry field names identical to their stock counterparts (`Front`/`Back`,
  `Text`/`Back Extra`), so the renderer's map shape and the planner's field-resolution layer are
  unchanged.
- **REQ-C-002** [Event-driven] When the user invokes `imoogi-anki-setup`, after its existing
  configuration steps the Go binary queries `modelNames` and then (1) issues one `createModel`
  per absent imoogi-owned type, carrying field list, card templates, and CSS — `imoogi-Cloze`
  with `isCloze: true`; (2) issues `updateModelStyling` + `updateModelTemplates` for each type
  already present, so repeated setup is idempotent; (3) on a styling update, replaces that
  type's entire CSS with the base+user concatenation, discarding hand edits, stating so before
  writing.
- **REQ-C-003** [State-driven + Unwanted] While an ordinary synchronization run is executing, the
  Go binary issues no `createModel` / `updateModelStyling` / `updateModelTemplates` on any code
  path that run reaches.
- **REQ-C-004** [Unwanted] The imoogi system does not modify the templates or styling of any note
  type whose name does not begin with `imoogi-`, on any code path — superseding the narrowed
  clause of parent REQ-016 (§B.2) and leaving every other clause in force.
- **REQ-C-005** [Ubiquitous] Note-type naming across surfaces: (1) the imoogi-owned types are the
  front end's default — marking commands, cloze auto-mark, `ANKI_NOTE_TYPE_ALL`, and the
  renderer's dispatch; (2) a heading declaring stock `Basic`/`Cloze` still synchronizes exactly
  as today, unthemed, with both name sets recognized on both sides; (3) every site hard-coding
  the stock literals names the imoogi-owned types wherever those are in play, while still
  accepting the stock literals.

### C.2 Card appearance

- **REQ-C-006** [Ubiquitous] Deck-derived class hook: (1) each card template wraps its content in
  one element carrying a `{{Deck}}`-derived CSS class, the wrapper existing only in the template
  and never in stored field content; (2) normalization lowercases, maps each `::` to one `-`,
  maps every remaining character outside `[a-z0-9_-]` to `-`, collapses `-` runs, strips leading
  and trailing `-`, and prefixes `_` when the result would begin with a digit.
- **REQ-C-007** [Ubiquitous] Stylesheet properties: (1) both night-mode selector forms
  (`.card.nightMode`, `.card.night_mode`) plus `.card img { max-width: 100%; max-height: none; }`
  and `word-wrap: break-word;`; (2) a restrained editorial base — serif headings over a system
  sans body, bounded measure, generous line height, raised math/code contrast, warm-paper light
  and near-black night — naming **system font stacks only, no network resource of any kind**.
- **REQ-C-008** [Capability gate + Event-driven] Where a user stylesheet exists at the configured
  path (`imoogi-anki.css` beside `imoogi.json` by default), when the install step runs the front
  end passes its contents and the binary appends them verbatim after the base stylesheet; absent,
  the uploaded CSS is the base stylesheet alone.
- **REQ-C-009** [Unwanted] No per-deck colour, font, or other appearance is assigned, rotated, or
  derived from a deck name or hash — the base stylesheet is byte-identical for every deck, and
  every per-deck rule originates in the REQ-C-008 user stylesheet.

### C.3 LaTeX → MathJax

- **REQ-C-010** [Event-driven] When rendered content carries an Org math fragment, the Go binary
  (1) emits `\(…\)` for `$…$` and `\(…\)`, and `\[…\]` for `$$…$$` and `\[…\]`, and no other
  delimiter form; (2) recognizes a single `$` by Org's own heuristic — at most two line breaks,
  attached to both `$` with no intervening whitespace, closing `$` followed by whitespace or by
  punctuation other than a dash; (3) emits `<br>` in place of any line break the converted
  fragment contains.
- **REQ-C-011** [Unwanted] No `$` inside a `<pre>` region, a `<code>` region, an HTML attribute
  value, or a link target is converted, and no reference to a network-hosted MathJax or
  stylesheet resource is emitted.

### C.4 Images and media

- **REQ-C-012** [Event-driven] When rendered content references a local image — relative path,
  `file:` link, or the two-part `[[file:x.png][description]]` form — the Go binary (1) first
  rewrites an anchor whose target carries a go-org image extension (`png`, `gif`, `jpg`, `jpeg`,
  `svg`, `tif`, `tiff`, `webp`, `xbm`, `xpm`, `pbm`, `pgm`, `ppm`, `pnm`, case-insensitively) to
  an `img` whose `alt` carries the description, then applies (2) and (3) as to a one-part
  reference; (2) resolves against the directory of the entry's own Org file, derived from the
  sync root and the entry's recorded source path, confining the result to that root; (3) computes
  the stored name as sanitized basename + content-hash suffix and rewrites `src` to that exact
  name, computed locally and therefore independent of any upload response.
- **REQ-C-013** [Unwanted] The Go binary does not (1) rewrite, upload, or disturb a reference
  whose target carries an `http:` / `https:` scheme; (2) prefix an uploaded media filename with
  `_`; (3) delete, enumerate for deletion, or otherwise reason about media files it did not
  upload during the run in progress.
- **REQ-C-014** [State-driven] While a single synchronization run is in progress, the Go binary
  (1) issues `storeMediaFile` only for entries that run actually adds or updates — an unchanged
  hash issuing no media request at all; (2) uploads each distinct stored media filename at most
  once, however many entries reference it.
- **REQ-C-015** [Event-detected] When a referenced local image cannot be resolved, read, or
  uploaded, the Go binary reports that target as a skipped entry carrying a diagnostic code
  naming the reference, creates or updates no note for it, leaves its registry hash unchanged,
  and continues with the remaining targets.

### C.5 Hashing and transform ordering

- **REQ-C-016** [Ubiquitous] Every transform this SPEC introduces — MathJax conversion,
  two-part-link rewriting, `src` rewriting — is applied **before** the content hash is computed
  for the add/update/no-op decision, and the field content sent to AnkiConnect is byte-identical
  to the content whose hash is recorded in the registry.
- **REQ-C-017** [Unwanted] No input is added to the content hash beyond the four the parent's D-6
  fixes — note type, rendered field values, resolved deck, sorted tag set — image content
  reaching the hash solely through the content-derived stored filename REQ-C-012 writes into the
  field text.
- **REQ-C-018 → folded into REQ-C-016.2 at revision 4.** When the first synchronization run after this rendering change
  executes, the Go binary updates in that one pass each previously synchronized imoogi note whose
  rendered content differs under the new transforms — a note carrying neither math nor a local
  image renders byte-identically and correctly issues no request — that invalidation being the
  intended consequence of the parent's post-render hashing rule, not a defect.

### C.6 Migration of existing imoogi notes

- **REQ-C-018** [Ubiquitous] Migration is reached through the `imoogi-anki migrate` CLI
  subcommand, reading a request of the same shape `sync` reads and writing a response of the same
  schema, adding no field to any existing wire document and leaving the protocol version constant
  unchanged.
- **REQ-C-019** [Event-driven] When `imoogi-anki-setup` completes its install step and the
  registry records one or more entries whose recorded note type is a stock type, the front end
  displays that count and obtains explicit confirmation — stating that migration discards each
  migrated note's review history and scheduling state — before issuing any **writing** `migrate`
  request; a declined confirmation issues none. The dry-run that produces the count is not a
  writing request (it reads the registry and issues no AnkiConnect write), so obtaining the count
  does not presuppose the answer.
- **REQ-C-020** [Capability gate + Event-driven] Where migration is invoked and confirmed, when
  an entry's recorded type is stock `Basic`/`Cloze` **and its declared type is that same stock
  type or its `imoogi-` counterpart**, the Go binary renders under the counterpart, adds the new
  note, reports the new identifier for write-back, and deletes the original only after the add
  succeeds; (2) when a migration completes, it replaces the registry record with one naming the
  new identifier and the imoogi-owned type, and the Elisp front end overwrites that heading's
  `ANKI_NOTE_ID` and `ANKI_NOTE_TYPE`.
- **REQ-C-021 / REQ-C-022** [Event-detected] **Split at revision 4 into two single-trigger requirements** — REQ-C-021 the mismatch, REQ-C-022 the add failure. (1) when, **on any
  path other than a confirmed migration**, an entry's recorded type differs from its declared type
  in any way — the hand-edited stock-to-`imoogi-` case included — the target is skipped carrying
  `note_type_change_unsupported`, no field is updated, and the run continues; on the migration
  path, an entry whose declared type is neither its recorded stock type nor that type's
  counterpart is skipped the same way. Parent REQ-021 is unchanged; the two are partitioned **by
  path in requirement text**, not by branch ordering. (2) When a migration's add step fails, the
  original note, its registry entry, and its heading properties are left untouched, the entry is
  reported skipped with a diagnostic code, and the run continues — no entry ever reaching a state
  where the registry names an identifier the collection no longer holds.

### C.7 Diagnostics, contracts, and non-interference

- **REQ-C-023** [Ubiquitous] Every diagnostic code this SPEC introduces exists as a
  `protocol.Code*` constant in the Go protocol package **and** as a matching entry in the Elisp
  diagnostic table (`modules/anki/imoogi-error.el`), the existing contract test enforcing the
  subset relation.
- **REQ-C-024** [Unwanted] The imoogi system does not (1) detect, report, or delete media files no
  imoogi note references — unused-media cleanup is Anki's own Check Media, which REQ-C-013's
  no-underscore rule keeps able to see imoogi's uploads; (2) upload, rewrite, or diagnose a link
  whose target is video or audio, or whose extension lies outside the REQ-C-012 image set — such
  targets pass through exactly as the renderer produced them; (3) create, modify, or read any note
  type beyond `imoogi-Basic` and `imoogi-Cloze`, the remainder of the parent's "advanced card
  types" exclusion staying in force as the non-normative §5 exclusion it always was; (4) modify
  the stock `Basic`/`Cloze` types, their templates, their CSS, or any note carried on them except
  through a migration confirmed under REQ-C-019 — supplying the narrow collection-ownership
  guarantee §B.4 inherits from the parent's deferred obligation.

---

## §D Design Decisions (highest change-likelihood first)

Renumbered contiguously at revision 2; D-C-3, D-C-4, D-C-5, and D-C-6 are new, and revision 1's
D-C-5 (model writes gated behind an install command) is folded into D-C-6.

### D-C-1 — imoogi-owned note types rather than restyled stock types

**Decision**: create `imoogi-Basic` and `imoogi-Cloze`; never touch stock `Basic` / `Cloze`.

**Rationale**: Anki attaches styling to the **note type**, not the deck, so there is no way to
style imoogi's cards without owning a type. The user's collection carries 88 stock-`Basic` and
27 stock-`Cloze` notes (measured) whose styling is theirs. Owning the type also means owning the
field names, which removes the lowercase-`front`/`back` hazard `planner/fake_client_test.go`
exists to reproduce.

**Rejected — restyle the stock types**: would restyle 115 cards the user never asked to change,
and would need the parent's REQ-016 amended far more broadly than §B.2 does.

**Rejected — one note type per deck**: makes a deck move a note-type change, which parent
REQ-021 forbids outright; `interview.md` decision #2 rejects it on exactly this ground.

### D-C-2 — Ownership is decided by the `imoogi-` name prefix

**Decision**: the predicate of §B.1, name-based, no local state.

**Rationale**: decidable from a single `modelNames` response and from a request log alone, which
is what makes AC-C-003a / AC-C-003b mechanically checkable. A registry-backed ownership record would
make the acceptance criterion depend on local state that a fresh checkout does not have.

**Consequence to accept**: a user who hand-creates a type named `imoogi-Something` hands imoogi
write authority over it. Acceptable — the namespace is imoogi's by convention throughout this
project (`.imoogi-registry.json`, `imoogi-anki`, the `imoogi-` Elisp prefix).

### D-C-3 — Per-deck appearance is user-authored; imoogi ships one base stylesheet

**Decision** (`interview.md` 라운드 3 R2 — the user chose against the recommended option):
imoogi emits the normalized `deck-<class>` wrapper (D-C-7) and ships **one** base stylesheet,
byte-identical for every deck. It assigns no per-deck appearance of any kind. A user stylesheet
at `imoogi-anki.css` beside `imoogi.json` in `user-emacs-directory` is appended verbatim after
the base stylesheet at install time; absent, the base stylesheet ships alone.

**Rationale**: the deck class is the *hook*, and the hook is the whole of imoogi's obligation.
The user asked for an elegant default and per-deck control, not for imoogi to decide what each
deck looks like. Shipping only the hook keeps the base stylesheet reviewable as a single
artifact, keeps deck-name changes free of visual consequence, and means no imoogi release can
alter the appearance of a deck the user has already styled.

**Rejected — automatic palette rotation keyed on the deck class or a deck-name hash.** This was
revision 1's recommended option and the user declined it. Three costs it carries, all real:
a hash-derived palette assigns a colour nobody chose and that nobody can predict before
creating the deck; renaming a deck silently re-colours it, because the hash input changed; and
the palette table becomes imoogi-owned surface that a user wanting a different colour can only
override by fighting a generated rule. A user-authored file has none of these properties and is
strictly more expressive.

**Rejected — a deck→style map in `imoogi.json`.** Would put CSS values into a JSON
configuration file that the Go binary parses, duplicating a language the browser already reads
and forcing every expressible style through a schema. The user stylesheet is CSS, so no schema
constrains it.

**Transport, decided rather than assumed**: the file's location is defined by an Elisp
`defcustom`, so **the front end reads it and passes its contents in the install request**; the
Go binary reads no stylesheet from disk. This preserves `protocol.Config`'s documented property
that "the binary reads no configuration file of its own", keeps the missing-file branch in the
one place that already owns `expand-file-name`, and adds no new diagnostic code.

### D-C-4 — Restrained editorial visual direction, system fonts only

**Decision** (`interview.md` 라운드 3 R3): serif headings over a system sans-serif body, a
bounded content measure, generous line height, raised contrast for math and code; light mode on
a warm paper tone, night mode on a near-black ground with reduced-contrast body text; both
`.card.nightMode` and `.card.night_mode`.

**Rationale**: it answers "우아한" with properties that are checkable in review rather than with
a mood. Every choice is also constraint-driven: a bounded measure is what makes a long cloze
readable on a phone; raised math and code contrast is the one place a reduced-contrast night
body would actively harm comprehension; and the two night-mode selectors are a measured
platform difference, not a stylistic one.

**Rejected — web fonts.** `AGENTS.md §0`'s air-gap rule forbids a network fetch, and Anki
renders cards offline by design. A card that waits on a font CDN degrades to a fallback stack
anyway, so the fallback stack is what should be specified in the first place.

**Rejected — a high-chrome themed look (card borders, drop shadows, accent bars).** Adds
surface that every user stylesheet then has to undo, and reads worse at phone size, where most
review happens.

### D-C-5 — The default note type for newly marked headings becomes the imoogi-owned one

**Decision** (`interview.md` 라운드 3 R6): `C-c a b` writes `imoogi-Basic`, `C-c a c` writes
`imoogi-Cloze`, `ANKI_NOTE_TYPE_ALL` offers both, and the cloze auto-mark's comparison accepts
either `Cloze` form. The Go binary recognizes both name sets. A hand-written stock `Basic` or
`Cloze` heading keeps synchronizing, unthemed (REQ-C-005).

**Rationale**: the theme is the point of the SPEC, and a default that produces unthemed cards
would make every new heading a manual edit away from the feature the user asked for. Keeping
the stock names recognized costs one extra `switch` arm and removes the alternative's entire
failure mode — an existing hand-written heading breaking on upgrade.

**Consequence to accept**: `24-anki.el:187`'s cloze auto-mark writes and compares a literal
today; it grows a two-name comparison. That is the specific site REQ-C-005 names.

**Rejected — keep the stock default and let the user opt in per heading**: makes the feature
invisible by default and turns every new card into a two-step operation.

### D-C-6 — Install rides `imoogi-anki-setup`; migration is a separate `migrate` subcommand

**Decision** (`interview.md` 라운드 3 R5 and R7): the model-install step is the **tail of
`imoogi-anki-setup`** — `modelNames` probe → absent ⇒ `createModel`, present ⇒
`updateModelStyling` + `updateModelTemplates` — and none of it runs on the sync path.
Migration is reached through a distinct `imoogi-anki migrate` CLI subcommand reading a
sync-shaped request on stdin, with **no new wire field and `protocol.Version` unchanged**.
After installing models, setup reports the count of registry entries still on stock types and
asks for explicit confirmation before issuing a `migrate` request.

**Rationale, install side**: this is what lets §B.3 keep the parent's per-run blanket **in force
for the hot path** (AC-C-003a) and lift it only on a command the user explicitly invokes (AC-C-003b) —
the smallest possible amendment to a shipped, audited parent. Extending `imoogi-anki-setup`
rather than adding `imoogi-anki-install-models` also means a user who follows the existing
setup instructions ends up with the models installed, with nothing new to discover.

**Two layers, one of which is not user-facing.** What R5 rejected is a second *Emacs* command
the user would have to know about. The Go binary still needs a verb to dispatch on, so it grows
an `install-models` subcommand — an internal surface invoked only by `imoogi-anki-setup`,
never documented as a user entry point, and named for what it does rather than for who calls it.
The rejection is about the user's command surface; the subcommand is about `main.go`'s `switch`.
Probe-then-act is additionally the safe pattern regardless of an unknown: whether `createModel`
on an existing name errors, no-ops, or overwrites is undocumented (§H R9).

**Rationale, migrate side**: a subcommand is dispatched at `main.go`'s `switch args[0]`, which
already carries a `sync` case, so a sibling case costs nothing structurally. The alternative —
a boolean on `protocol.Config` — would change an existing wire document, and the wire rule is
"No field carries omitempty" with `main.go:79` hard-rejecting a version mismatch and the Elisp
constant pinned at `imoogi-process.el:19`. A subcommand avoids a coordinated version bump for a
one-shot operation entirely.

**Rejected — reconcile models on every sync run**: would break the parent's AC-016 on every run
and force a far wider amendment, for a benefit (self-healing after a manual template edit) that
REQ-C-002 already gives the user on demand.

**Rejected — migrate automatically, without confirmation**: migration destroys scheduling state
(D-C-13), which is exactly the class of loss the parent's REQ-016 exists to prevent.

**Where the prompt's count comes from**: `migrate` carries a dry-run mode that reports the
candidate entries and issues zero AnkiConnect writes; setup calls it, shows the count, asks,
and on confirmation calls `migrate` for real. The alternative — an Elisp reader for
`.imoogi-registry.json` — would duplicate registry-schema knowledge in a second language, which
is the same cross-language drift the diagnostic-code contract test exists to police.

**Consequence to accept**: `migrate` re-renders each candidate, so setup must run the scan
before calling it. The request is sync-shaped precisely because the migrated note must be
re-added with the *new* rendering, not the old (§F M5's dependency on M1-M4).

### D-C-7 — One type per kind, per-deck styling through a template-side `{{Deck}}` wrapper

**Decision**: `<div class="deck-{{Deck}}">…</div>` lives in the **card template** delivered by
`createModel`, never in field content.

**Rationale**: `{{Deck}}` is a template special field Anki expands at review time — it only works
there. Two consequences follow for free, and both matter: the wrapper is invisible to
`hashing.Hash`, and it keeps the wrapper out of stored field content entirely, so no part of the
theme depends on how Anki round-trips a field. Deck changes already invalidate the hash
independently, since the resolved deck is a hash input (parent D-6).

**Rejected — a wrapper injected into field content**: would put a `<div>` into stored HTML,
inside the hash, for no gain — and `{{Deck}}` would not expand there anyway.

### D-C-8 — `{{Deck}}` is the full `::` path, so normalization is mandatory

**Decision**: the rule of REQ-C-006.

**Rationale**: settled from Anki's own source rather than the manual —
`pylib/anki/template.py` sets `fields["Deck"] = self._col.decks.name(self._card.current_deck_id())`,
the full path; `{{Subdeck}}` is the basename. Per MDN's CSS `<ident>` rules a class may not
contain `:` or a space unescaped and may not begin with an unescaped digit, so a raw
`class="deck-{{Deck}}"` emits an unusable class. The user's own measured deck name
`"(PROGRAMMER)::(GO)"` (`modules/24-anki.el:53`) proves parentheses and `::` both occur in
practice, so the rule must handle both.

**Rejected — use `{{Subdeck}}` instead**: collapses `A::Go` and `B::Go` onto one class, losing
the parent-deck distinction the user's own deck layout depends on.

### D-C-9 — Transforms run before the hash; stored bytes equal hashed bytes

**Decision**: media resolution + `src` rewrite + two-part-link rewrite + MathJax conversion all
complete before `hashing.Hash` at `planner.go:243`, and the field map sent to AnkiConnect is the
same map that was hashed.

**Rationale**: this is the SPEC's highest-leverage correctness constraint and three research
lenses reached it independently. The hash has **two** uses — the no-op/update decision at
`planner.go:243`, and the orphan-deletion **ownership predicate** recomputed from a `notesInfo`
response at `planner.go:541`. If a transform runs after hashing, or only on the outbound
request, every imoogi note fails that recomputed predicate **permanently**, orphan deletion
silently stops, and the only symptom is a stream of `delete_candidate_unowned` reports. The
parent's own run-phase report already flags this failure mode as fail-safe-but-silent.

The recomputation's other precondition — that Anki returns stored field HTML unchanged — is now
**measured rather than assumed** (§H R1), so this ordering rule is the only thing standing
between the implementation and a silent deletion stop.

**Rejected — transform on the outbound request only**: exactly the failure above.

**Encoded as a test, not as prose.** `research.md §10` records the parent's recurring audit
defect: "design-narrative ordering mistaken for a behavioral guard". The guard is therefore an
assertion, not a sequence in this document — see §G.

### D-C-10 — Image content reaches the hash through the filename, not through a new hash input

**Decision**: stored media name = sanitized basename + content-hash suffix. Add **no** input to
`hashing.Hash`.

**Rationale**: `hashing.Hash`'s own doc comment establishes that every input must round-trip
through a `notesInfo` response, and raw image bytes cannot. The content-hash suffix solves both
open questions at once: it makes filename collisions between same-named images in different
directories impossible, **and** it makes an image edit change the rendered `<img src>`, hence the
field HTML, hence the hash — so an edited image correctly triggers an update, for free.

**Rejected — hash the image bytes as a fifth input**: breaks the round-trip invariant, so orphan
confirmation could never recompute it.

**Rejected — `_`-prefixed upload names**: Anki treats a leading `_` as "ignore during media
checks", which would exempt imoogi's uploads from the very Check Media cleanup REQ-C-024 relies
on to keep orphan-media out of scope.

### D-C-11 — Media is a separate post-render pass; `Render` stays pure

**Decision**: `orgdoc.Render` keeps its purity contract (parent `plan.md D-1`: "a pure function
of its inputs: no registry, no AnkiConnect client, no side effects"). A new
`internal/anki/media` package takes `(baseDir, renderedFields)` and returns
`(rewrittenFields, []Upload)` — reading local bytes for the content hash, which is read-only
local I/O and no network. The **planner** hashes the rewritten fields, and issues
`StoreMediaFile` for the returned uploads only on the add and update paths.

**Rationale**: `Render`'s signature is `Render(noteType, title, body string)` with `"./"`
hardcoded as the parse base, so it cannot resolve a relative path today — but it does not need
to. Because the media pass operates on rendered HTML and receives `baseDir` itself, **`Render`'s
signature stays unchanged**: go-org's parse base governs `#+INCLUDE` resolution, not the `src`
attributes the media pass rewrites afterwards. That keeps the purity contract literally intact
and the diff smaller than the alternative.

**Two-part links are the same pass's problem** (`interview.md` 라운드 3 R4). Measured in
go-org v1.9.1: `RegularLink.Kind()` (`org/inline.go:402-416`) returns `"image"` only when the
link has no description, or when the description is itself an image URL — so
`[[file:diagram.png][My diagram]]` emits `<a href="diagram.png">My diagram</a>`. Because the
media pass already walks rendered HTML, converting that anchor to `<img src alt="My diagram">`
is one more rule in a pass that exists, not a new mechanism. The description becomes `alt`
rather than being discarded, so the accessibility text the user wrote survives.

**Extension set: exactly go-org's, with nothing added.** The set is
`imageExtensionRegexp` at `org/inline.go:68` — `(?i)^[.](png|gif|jpe?g|svg|tiff?|webp|x[bp]m|p[bgpn]m)$`.
`.avif`, `.bmp`, and `.ico` are **not** added. The reason is consistency rather than caution:
go-org decides the one-part case and imoogi decides the two-part case, so any extension imoogi
recognized and go-org did not would make `[[file:x.avif]]` an anchor and
`[[file:x.avif][d]]` an image — the same file behaving differently for a reason no user could
infer. Adding an extension is a follow-up that changes both decisions together.

**Rejected — grow `Render` a base-path parameter**: churns every call site and every
`orgdoc_test.go` case to relocate a responsibility the post-render pass already owns.

**Upload is deferred to the action phase, not performed during the rewrite.** The stored filename
is `sanitized basename + local content hash` — computed client-side, independent of any upload
response (REQ-C-012) — so the rewrite and the hash both complete without a network call. The
planner therefore rewrites, hashes, compares, and issues `storeMediaFile` **only for the entries
it is about to add or update** (REQ-C-014). Uploading during the rewrite instead would fire a
media request for every image-bearing note on every run, including an otherwise-no-op one, which
is the media analogue of the AC-015 violation this SPEC is careful to avoid elsewhere.

**Rejected — render via Emacs `ox-html` to get math and image handling natively**: reopens parent
`D-1`, which rejected it because "it splits rendering across two languages, requires an Emacs
process in the test loop, and makes the back end untestable in isolation".

**Path-resolution precedent is in-repo**: `internal/orgpreview/assets.go` `AssetResolver.Resolve`
already URL-unescapes, joins against `filepath.Dir(baseFile)`, canonicalizes via `filepath.Abs`
+ `EvalSymlinks` tolerating `os.IsNotExist`, and confines results to allowed roots via `within()`.
Reuse that confinement logic; do not reuse `internal/orgpreview/parser.go`'s `linkRE`, which
belongs to a different parser than the one the anki path uses.

### D-C-12 — imoogi applies Org's `$` heuristic itself; it cannot delegate to go-org

**Decision**: implement REQ-C-010's heuristic in imoogi's own transform.

**Rationale**: this is a genuine behavioral divergence, not a formatting nicety
(`research.md` C3). go-org v1.9.1's `parseLatexFragment` (`org/inline.go:211-224`) requires only
`start+2 < len(input)` and then `strings.Index` for the next `$` anywhere in the remainder — no
word-boundary, whitespace, or digit checks. So `costs $5 and $7` is **already** parsed as a
fragment today. The delimiters currently survive, so the HTML looks unchanged and nobody notices;
the moment imoogi rewrites `$…$` → `\(…\)` that silent misparse becomes visible broken MathJax
on the user's card. imoogi must therefore decide which `$` pairs to convert using Org's rule, not
go-org's.

**Rejected — convert every fragment go-org identified**: ships the `$5` bug to the user's cards.

### D-C-13 — Migration adds before it deletes, and is disjoint from REQ-021 by requirement text

**Decision**: render under the new type → add → write back the new `ANKI_NOTE_ID` and
`ANKI_NOTE_TYPE` → delete the original, only after the add succeeded, and only through the
existing ownership-confirmed delete path.

**Rationale**: the ordering matters for crash safety. Delete-first leaves a window in which the
registry names an identifier Anki no longer holds; a crash there strands the entry. Add-first
leaves the opposite window — a duplicate note, which is visible, recoverable, and never a data
loss. REQ-C-022 makes the add-failure path a no-op on the original. `addNote` will not dedupe
against the stock-type original, because duplicate detection is scoped within a note type and the
model differs (`research.md §8`).

**The migrate command rewrites `ANKI_NOTE_TYPE` itself.** Revision 1 left open whether the user
edits that property first. The R5 flow settles it: setup reports a count and asks once, so the
user cannot be required to hand-edit N headings before answering. REQ-C-020's trigger therefore
reads the *registry-recorded* type, not the declared one, and REQ-C-020.3 writes both properties.

**Candidate set, stated because it is coarser than it looks.** The registry cannot distinguish
an entry that is a legacy artifact from one whose stock type the user chose deliberately: both
are imoogi-created entries recording type `Basic`. The candidate set is therefore *every*
registry entry recording a stock type, migration is all-or-nothing per confirmation, and the
confirmation text names the count and the scheduling loss so the decision is informed.
Declining leaves those entries on stock types and the prompt recurs at the next setup — mildly
annoying, and the honest behavior. Per-heading opt-out is a follow-up candidate (§J).

**Disjointness is partitioned by path, not by type-pair.** REQ-C-020 is gated by a GEARS
`Where` clause naming a *confirmed migration*; REQ-C-021 governs every other path plus the
migration path's own non-counterpart residue, so no (recorded, declared) pair is left ungoverned.
The case that made the type-pair partition wrong is concrete: a user who hand-edits
`ANKI_NOTE_TYPE` to `imoogi-Basic` on a heading whose registry record still says `Basic` must
skip with `note_type_change_unsupported` on the sync path — REQ-021's behavior, unchanged — and
be told by `imoogi-error.el`'s corrective prose to run `imoogi-anki-setup` instead of editing the
property by hand.
An implementer reading `spec.md §3` alone reaches the same behavior as one reading a branch
order — which is precisely the correction parent `D-12` had to make after audit, and
`research.md §10` names as a recurring blocking-defect theme.

**Verified, so not an open question**: the Elisp write-back path uses `org-entry-put`
(`modules/anki/imoogi-writeback.el:72`), which overwrites an existing property rather than only
inserting — so REQ-C-020.3's two-property replacement needs no new write-back mechanism.

**Rejected — an in-place model change**: parent REQ-021 forbids it, note type is a hash input so
a recorded-type mismatch *always* implies a hash mismatch, and `D-12` records that AnkiConnect's
behavior on a field-shape mismatch is unverified.

### D-C-14 — Field names mirror the stock types

**Decision**: `Front` / `Back` and `Text` / `Back Extra`.

**Rationale**: keeps `orgdoc.Render`'s output map shape identical, so `planner.resolveFields`
needs no change. That matters in the reverse direction from the obvious one: `resolveFields`
exists to handle a *customized* stock `Basic` carrying lowercase `front`/`back` — the user's
actual measured situation — so the risk is new code bypassing it and breaking the stock case, not
the layer being needed for imoogi's own models.

### D-C-15 — Template and base CSS ship as embedded assets

**Decision**: `//go:embed` the card templates and the base stylesheet. The *user* stylesheet is
not embedded — it arrives in the install request per D-C-3.

**Rationale**: the base CSS is large enough that a Go string constant would be unreadable and
un-lintable. Flagged explicitly because it is a **first**: `grep -rn "embed" --include='*.go'
internal cmd` returns nothing, and `find . -name "*.css" -not -path "./vendor/*"` returns
nothing. The repo's only existing styling is a ~6-line inline `<style>` string constant at
`internal/orgpreview/server.go:754-764`. Air-gap rule (`AGENTS.md §0`) makes embedding mandatory
rather than merely tidy — no CDN, and Anki's bundled MathJax is the only network-free option,
which `docs.ankiweb.net/math.html` confirms is sufficient.

---

## §E Touch Surfaces

| File | Edit |
|---|---|
| `internal/anki/ankiconnect/client.go` | Add `ModelNames`, `CreateModel`, `UpdateModelStyling`, `UpdateModelTemplates`, `StoreMediaFile` to the 10-method `AnkiConnector` interface (`:45-56`) and to `*Client`. All calls funnel through the single `call(ctx, action, params)` chokepoint; error tiers `TransportError` / `ProtocolError` / `APIError` unchanged. |
| `internal/anki/planner/fake_client_test.go` | Implement the 5 new methods on `fakeClient` with per-action call logs matching the existing `addCalls` / `createDeckCalls` shape (`:17-50`), so AC-C-003a's "exactly 0" is assertable. |
| `internal/anki/orgdoc/orgdoc.go` | Add `imoogi-Basic` / `imoogi-Cloze` to the constants and the note-type `switch`, keeping the stock arms (REQ-C-005); the `default` still returns `unrecognized note type %q`. Add the MathJax post-render transform beside `clozeMarkerPattern` (`:28`), which is the same shape of change with the same documented go-org-passthrough rationale. `Render`'s signature is unchanged (D-C-11). |
| `internal/anki/media/` (new) | Resolve → content-hash → rewrite pass, including the two-part-anchor→`img` rule of REQ-C-012 against go-org's own extension set (`org/inline.go:68`). Reuse `internal/orgpreview/assets.go`'s `Resolve` / `canonical` / `within` confinement logic; content hashing follows `internal/setup/setup.go:611 fileSHA256`. Subpackage-per-concern matches the `internal/anki/*` convention. |
| `internal/anki/model/` (new) | `//go:embed`ed templates + base stylesheet, the deck-class normalizer, the base+user CSS concatenation, and the probe-then-act install logic. Reads no stylesheet from disk — the user CSS arrives in the request (D-C-3). |
| `internal/anki/planner/planner.go` | Transform-before-hash ordering at `:243`; migration branch disjoint from the REQ-021 skip at `:265-275`; ownership recompute at `:541` unchanged. |
| `internal/anki/protocol/protocol.go` | New `Code*` constants (currently 14). **No change to `Request`, `Config`, or `Version`** — the install and migrate documents are new request types carried by new subcommands, and no existing wire document gains a field (D-C-6, closing revision 1's open item here). |
| `cmd/imoogi-anki/main.go` | Two new cases beside `sync` in `run`'s `switch args[0]`: `install-models` (reads the install request: AnkiConnect URL plus the user CSS text) and `migrate` (reads a sync-shaped request; `--dry-run` reports candidates and issues zero writes). Each probes `protocol_version` the way `runSync` does. `main_test.go`'s E2E stub gains both. |
| `modules/anki/imoogi-setup.el` | `imoogi-anki-setup` gains its tail: run the model install, then scan, then call `migrate --dry-run` for the candidate count, then `y-or-n-p` naming the count and the scheduling loss, then `migrate` for real on confirmation. |
| `modules/anki/imoogi.el` | New `defcustom` for the user stylesheet path, defaulting to `(expand-file-name "imoogi-anki.css" user-emacs-directory)` beside the existing `imoogi-config-file` (`:63-66`); read it at install time, absent ⇒ empty. Note-type literals at `:64`, `:93`. |
| `modules/24-anki.el` | `ANKI_NOTE_TYPE_ALL` (`:55`), the note-type-writing commands (`:69`, `:78`), and the cloze auto-mark whose `cond` writes and compares the literal `"Cloze"` (`:187`, `:189-192`) — write the imoogi-owned names, accept both (D-C-5). |
| `modules/anki/imoogi-process.el` | Request construction for the two new subcommands; `protocol.Version` constant at `:19` unchanged. |
| `modules/anki/imoogi-error.el` | One table entry per new `protocol.Code*`; the corrective prose at `:36` and `:44` names `Basic` / `ANKI_NOTE_TYPE` literally. |
| `tests/anki-*.el`, `tests/anki-error-test.el` | Contract test enforces the Go-const ⊆ Elisp-table subset relation. |
| `~/.emacs.d/imoogi-anki.css` (user-owned, not shipped) | The per-deck stylesheet the user may author. imoogi never creates, ships, or version-controls it; its absence is the normal case. |

Build and verify: `make build-anki`, `make test` (`test-elisp test-go test-shell`), `make lint`
(`go vet`). `go.mod`: go 1.26, `go-org v1.9.1`; `go.work` keeps Go out of vendor mode.

---

## §F Milestones (priority-ordered; no time estimates)

**M1 — Priority: High. AnkiConnect client surface and test doubles.**
Add the five methods to `AnkiConnector`, `*Client`, and `fakeClient` with call logs. No behavior
change to any existing path. Acceptance sketch: every existing planner test passes unchanged; a
new test asserts a sync run records zero calls on all five new logs (AC-C-003a's mechanism exists
before anything can violate it). Test strategy: RED — write the zero-model-writes assertion
first; it passes trivially and then keeps passing for the rest of the SPEC as a regression fence.
No dependency; lowest risk; do it first.

**M2 — Priority: High. Embedded template + base stylesheet and the install step.**
`internal/anki/model`: embedded `imoogi-Basic` / `imoogi-Cloze` templates carrying the
`{{Deck}}` wrapper, the full editorial stylesheet of REQ-C-007, the deck-class normalizer, the
base+user concatenation, and the probe-then-act install logic; the `install-models` subcommand
and the `imoogi-anki-setup` tail that calls it. Acceptance sketch: table test over the
normalizer including `"(PROGRAMMER)::(GO)"`, a digit-leading deck, and a deck with consecutive
separators; `fakeClient` asserts absent ⇒ exactly one `createModel` per type, present ⇒
`updateModelStyling` + `updateModelTemplates` and zero `createModel`; a second invocation is
byte-identical to the first (idempotence); a supplied user CSS appears verbatim after the base
CSS and an absent one yields the base alone; the stylesheet text contains no `http` or `@import`
occurrence (REQ-C-007's air-gap clause, mechanically checkable); every model-write request names
a prefix-matching model (AC-C-003b). Test strategy: pure table tests for the normalizer and the
concatenation, `fakeClient` request-log assertions for the wire behavior. **Revision 2 removes
the scope split**: the visual direction is decided (D-C-4), so M2 delivers the complete
stylesheet rather than a skeleton, and nothing in it gates M3 or M4.

**M3 — Priority: High. LaTeX → MathJax transform (pure, table-tested).**
The post-render transform in `orgdoc`. Acceptance sketch: a table test covering all four input
syntaxes; the `costs $5 and $7` non-conversion (D-C-12); a multi-line fragment producing `<br>`;
`$` inside `<pre>`, `<code>`, an attribute, and a link target left alone; `\(\sum_{i=1}^n a_n\)`
in → sub/sup-free out, carried as a **regression test rather than an assertion** because
`research.md` C2 records that neither lens verified go-org's fix version; and
`{{c1::\(x^{2}\)}}` as an explicit cloze-brace-collision edge. Test strategy: pure-function
table tests with rationale-bearing names, following the existing
`TestRender_Cloze_PreservesMarkerByteForByte` convention. Independent of M2.

**M4 — Priority: High. Media resolve / upload / rewrite, and the ordering guard.**
`internal/anki/media` plus the planner wiring. Acceptance sketch: resolution against the entry's
own directory; path confinement rejecting an escape above the sync root; content-hash suffix
distinguishing two same-named images in different directories; a two-part `[[file:x.png][d]]`
becoming `<img … alt="d">` and a two-part `[[file:x.avif][d]]` staying an anchor (REQ-C-012's
extension boundary, both directions); a `.webm` target passing through untouched with no
diagnostic (REQ-C-024); `https://` untouched; missing file producing a skip with a code, no add
request, and an unchanged registry hash; per-run dedupe of a shared image; and — the media
analogue of AC-015 — an unchanged re-sync recording **zero** `storeMediaFile` calls, proving
upload is gated on the add/update paths (REQ-C-014). **The ordering guard is the load-bearing
assertion**: a test asserts that the field map recorded in `fakeClient.addCalls` is byte-equal to
the map whose hash the registry records, and that recomputing `hashing.Hash` from the fake's
stored field values reproduces the registry hash exactly — the same recomputation
`planner.go:541` performs. Prose sequencing is not the guard; this test is. Depends on M3 (both
transforms must be in place before the byte-equality assertion covers the real pipeline).

**M5 — Priority: Medium. Migration of the existing imoogi notes.**
The `migrate` subcommand and its dry-run mode, plus the setup-side confirmation. Acceptance
sketch: dry-run reports the candidate count and records zero AnkiConnect writes of any kind; a
declined confirmation issues no non-dry-run `migrate` request at all; add-before-delete ordering asserted
from the request log; a failed add leaving the original note, registry entry, and heading
properties untouched with the entry reported skipped; registry, `ANKI_NOTE_ID`, and
`ANKI_NOTE_TYPE` all carrying the new values after success; the REQ-C-021 complement still
producing `note_type_change_unsupported` for every non-migration type change. Test strategy:
`fakeClient` sequence assertions on request order, plus a registry round-trip. Depends on M1-M4
(the migrated notes must be re-added with the new rendering, or they migrate straight into a
stale hash).

**M6 — Priority: High (final). Elisp wiring, diagnostics, and contract tests.**
Note-type defaults switched at all named sites with the stock names still accepted (D-C-5), the
user-stylesheet `defcustom`, one `imoogi-error.el` entry per new `protocol.Code*`, corrective
prose updated, and the setup entry point wired end to end. Acceptance sketch: the existing
contract test passes with every new code present on both sides; a hand-written stock `Basic`
heading still syncs (REQ-C-005); `make test` green end to end; a manual smoke test on the real
collection confirming the stock `Basic` 88 and `Cloze` 27 notes are untouched, both night modes
render, and math and images display on desktop and on one mobile client.

---

## §G Test Strategy (cross-cutting)

- **TDD per `quality.yaml`.** RED first at every milestone; M1's zero-model-writes assertion is
  written before any model code exists and stays green for the rest of the SPEC.
- **Pure-function table tests** for the two transforms (`orgdoc` math, `media` rewrite), the
  deck-class normalizer, and the base+user CSS concatenation — no client, no I/O, exhaustive
  edge enumeration.
- **`fakeClient` request-log assertions** for every wire-visible behavior, including the negative
  ones. Negative assertions are the reason each new connector method arrives with its own call
  log rather than sharing one.
- **The transform-before-hash guard is a byte-equality test**, per D-C-9 and `research.md §10`.
- **`cmd/imoogi-anki/main_test.go`** E2E stub gains the `install-models` and `migrate` actions,
  including migrate's dry-run.
- **Elisp**: flat `tests/*.el`; `tests/anki-error-test.el` mechanically enforces the diagnostic
  contract.
- **Coverage bar**: the parent's achieved figures are the floor — `internal/orgdoc` 100.0,
  `internal/hashing` 100.0, `internal/planner` 90.3, `internal/registry` 87.8,
  `internal/ankiconnect` 84.3 (floor 85.0) — each package's floor being its **currently
  measured** figure (`go test -count=1 -cover ./internal/...` at parent-repo
  `HEAD f6ee148`), so no `[MODIFY]` package may regress. The two `[NEW]`
  packages carry no inherited floor and are held to the 85.0 project norm.

---

## §H Risks and Dispositions

Revision 2 carries **zero** open-clarification markers: R1 became a measurement and R2-R8
became decisions (`interview.md` 라운드 3). What remains below is genuine residual risk.

| # | Risk / observation | Disposition |
|---|---|---|
| R1 | **Does Anki canonicalize stored field HTML on save?** If it did, the ownership predicate recomputed at `planner.go:541` would break for every image- or MathJax-bearing field regardless of transform ordering, and orphan deletion would silently stop. All four research lenses flagged this as the top unknown. | **Measured — it does not.** A probe note was added to the live collection carrying reordered attributes, `&amp;` and `&lt;` entities, runs of multiple spaces, a self-closing `<img … />`, and both `\(..\)` and `\[..\]` fragments; `notesInfo` returned every field byte-identical; the probe was deleted. The recomputed predicate is safe and revision 1's fallback design is **deleted, not deferred**. Residual: a future Anki release could introduce normalization — if it is ever observed, the predicate narrows to note-type-only and the probe is re-run. |
| R2 | Per-deck CSS authoring surface. | **Decided (라운드 3 R2, D-C-3):** user-authored file only, no automatic assignment. REQ-C-008, REQ-C-009. |
| R3 | Concrete visual direction for "우아한". | **Decided (라운드 3 R3, D-C-4):** restrained editorial, system fonts only. REQ-C-007. |
| R4 | Two-part image links render as anchors. | **Decided (라운드 3 R4, D-C-11):** rewrite to `img` with the description as `alt`, over go-org's own extension set with nothing added. REQ-C-012. |
| R5 | Install and migrate command surfaces. | **Decided (라운드 3 R5, D-C-6):** install is the tail of `imoogi-anki-setup`; migration is confirmed there and executed by a subcommand. REQ-C-002, REQ-C-018, REQ-C-019. |
| R6 | Default note type for newly marked headings. | **Decided (라운드 3 R6, D-C-5):** imoogi-owned by default, stock names still recognized. REQ-C-005. |
| R7 | `protocol.Version` bump for a migration flag. | **Decided (라운드 3 R7, D-C-6):** no wire field, no bump — a CLI subcommand instead. REQ-C-018. |
| R8 | Video, audio, and extensions outside go-org's image set. | **Decided (라운드 3 R8):** out of scope; such targets pass through untouched with no diagnostic. REQ-C-024, and §J records the follow-up. |
| R9 | **AnkiConnect API behaviors the README does not state**: whether `createModel` on an existing name errors, no-ops, or overwrites; whether `updateModelTemplates` with an absent template name adds or errors; whether `storeMediaFile` overwrites by default. | **Not a user decision — designed around.** D-C-6's probe-then-act never calls `createModel` on a present name, so behavior 1 is unreachable. Behaviors 2 and 3 are M2/M4 run-phase probes against a real AnkiConnect before the code depends on either. |
| R10 | **AnkiConnect documentation provenance (C6).** The canonical `git.sr.ht` source returned 502 and `foosoft.net` returned 404; all request shapes were read from a GitHub fork mirror of medium confidence. | **Verification action.** Re-verify every request shape against the canonical README at M1 before pinning the client signatures. Wording already matches the parent SPEC's own earlier direct fetch, so the expected outcome is confirmation. |
| R11 | **go-org HTML-escapes math interiors** (`org/html_writer.go:326-334`), so `a < b` reaches Anki as `a &lt; b`. | **M3 verification item.** MathJax reads DOM text nodes, where `&lt;` has already been decoded to `<`; unescaping in the source would produce invalid HTML. Keep the escaping, verify rendering on a real card, and record the outcome. |
| R12 | **The first run after this SPEC mass-updates existing imoogi notes.** | Not a defect — the intended consequence of the parent's D-6 post-render hashing rule. **Bounded twice over**: only notes whose rendered bytes actually change under the new transforms are affected (a plain-text note is untouched — the deck wrapper being template-side per D-C-7), and the existing imoogi notes reach fresh hashes through M5 if migrated. REQ-C-016.2 makes it specified behavior. The user-facing announcement is **not** carried as a requirement: no REQ or AC binds it, so the wording here is a release-note recommendation rather than an obligation this SPEC verifies. Stating it in the release notes is advised; nothing fails if it is omitted. |
| R13 | **Five new connector methods break `fakeClient` and every planner test until updated.** | Contained by construction: M1 does exactly this and nothing else, so the breakage window is one milestone wide and no behavior changes inside it. |
| R14 | **The migration candidate set is coarser than the intent.** The registry cannot tell a legacy stock-type entry from one the user chose deliberately, so an all-or-nothing confirmation may migrate a heading the user meant to leave stock. | Accepted, with the cost surfaced rather than hidden: the confirmation names the count and the scheduling loss, declining is free, and the prompt recurs. Measured scale is 5 entries. Per-heading opt-out is a follow-up (§J). |
| R15 | **`imoogi-anki-setup` grows a scan and two subprocess calls**, so a command that was previously configuration-only now does real work and takes correspondingly longer. | Accepted. The alternative — a separate install command — was rejected in D-C-6 because discoverability matters more than setup latency for a once-per-machine operation. Dry-run mode issues zero AnkiConnect writes, so the added cost on the decline path is one scan. |

---

## §I Tier Recommendation — **L** (re-confirmed at revision 2)

Recommended, for Phase 9 confirmation by the user. Revision 2 **strengthens** the L judgment
rather than weakening it: closing the clarifications removed 10 marker strings but added 7 requirements,
3 design decisions, 2 CLI subcommands, and one new user-facing configuration surface.

| Signal | Revision 1 | Revision 2 |
|---|---|---|
| Milestones | 6 (`M1`-`M6`) | 6 (`M1`-`M6`), M5 grown by a dry-run mode and a confirmation flow |
| Non-test deliverables | ≥ 13 | ≥ 15 — 6 Go packages touched or created, 2 embedded assets, **2** CLI subcommands, 5 Elisp modules |
| Languages | 2 (Go + Emacs Lisp) | unchanged, with a mechanically enforced contract between them |
| Requirements | 35 | **42** (`REQ-C-001`..`REQ-C-024`) across 8 groups |
| Design decisions | 12 | **15** |
| Cross-SPEC scope | one REQ, one AC, two §4 exclusions of a shipped audited parent | unchanged |
| Open clarifications | 10 marker strings across 8 rows | **0** |

**Why not M.** File count alone settles it. The parent SPEC is the direct precedent and it was
**re-tiered M→L during audit** on exactly that ground — "File count alone (16 non-test
deliverables) exceeded Tier M's 5-15 band" (`research.md §10`). At ≥ 15 deliverables this SPEC
sits at the same boundary from the same direction, and adds two things the parent did not carry:
a cross-language contract and a formal amendment to a closed SPEC. Choosing M here would repeat
a re-tiering the project has already paid for once, across five plan-audit iterations.

**Why not larger than L.** L is the ceiling; the question is only whether the work should split
into two SPECs. It should not: M2-M4 all write into the same rendering pipeline and share the
transform-before-hash guard, so splitting them would put that single load-bearing assertion on a
seam between two SPECs — the one place it must not sit.

**Tier L budget, as settled at revision 3.** Tier L caps requirements and acceptance criteria at
25 each, independently. This SPEC sits at **24 requirements and 24 acceptance criteria** — one
slot of headroom on each axis. The parent consumed its own AC budget completely at 25/25, so its
late fixes had to extend existing ACs or live in `acceptance.md §D.7`; the single free slot here
exists to keep that from recurring. A late obligation that will not fit should extend an existing
requirement's sub-clauses rather than claim the last identifier.

---

## §J Dependencies and Cross-References

- **Amends** `SPEC-ANKI-001` (out-of-repo): REQ-016 (§B.2), AC-016 (§B.3), §4 "advanced card
  types" and "collection ownership guarantees" (§B.4), `plan.md` D-12 (§B.5).
- **Inherits unchanged** from the parent: D-1 (go-org, `ox-html` rejected), D-6 (hash inputs and
  post-render rationale), D-9 (identifier-based, content-confirmed deletion), D-10 (deck is a card
  property), REQ-021 (restated as REQ-C-021's complement).
- **No dependency** on `SPEC-TRANSIENT-001` (disjoint module set).
- **Source research**: `research.md` (§1 conflict, §3 `{{Deck}}` + normalization, §6 hash
  ordering, §8 migration, §9 touch surfaces, C1-C6 contradictions, open-questions verdict table),
  `research-lenses.md` (per-lens raw reports), `interview.md` (14 binding decisions across three
  rounds, plus the SPEC-ID correction).
- **Deferred to a later SPEC**: video and audio upload and the extensions outside go-org's image
  set (§H R8); per-heading migration opt-out (§H R14); orphan-media cleanup (REQ-C-024 — the
  registry schema is forward-compatible on read, so the door stays open); a general collection
  backup / restoration guarantee; and the remainder of the parent's §4 advanced-card-type
  exclusions.
