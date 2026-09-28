---
id: SPEC-ANKICARD-001
title: "imoogi-owned Anki note types with themed cards, MathJax, and image media"
version: "0.1.2"
status: in-progress
created: 2026-09-05
updated: 2026-09-05
author: jay
priority: P2
phase: "v0.2.0 target"
module: "internal/anki, cmd/imoogi-anki, modules/anki, modules/24-anki.el"
lifecycle: spec-anchored
tags: "anki, org-mode, note-types, css-theme, mathjax, media, migration"
tier: L
amendment_of: SPEC-ANKI-001
---

## HISTORY

### v0.1.2 (2026-09-05)

- CHANGED: § 3 **re-partitioned** at 24 requirements (count unchanged,
  `REQ-C-001`..`REQ-C-024` still contiguous) so that every requirement carries
  exactly one GEARS trigger. Two defects drove it: former REQ-C-022 merged two
  distinct detected events under one identifier, and former REQ-C-021's
  write-back sub-clause opened a second `When` under a `Where` gate. The split
  was paid for by folding former REQ-C-018 — the first-run mass invalidation,
  which added no obligation beyond what the hashing rule already entails — into
  REQ-C-016 as a case-split on the hash decision. Old → new mapping:

  | v0.1.1 | v0.1.2 | Disposition |
  |---|---|---|
  | REQ-C-001 .. REQ-C-017 | **unchanged identifiers** | REQ-C-005, REQ-C-008, REQ-C-016 rewritten in place |
  | REQ-C-018 | **folded into REQ-C-016.2** | no obligation dropped; the invalidation is the hash rule's consequence |
  | REQ-C-019 | **REQ-C-018** | renumbered; gains the `migrate_candidate` dry-run shape |
  | REQ-C-020 | **REQ-C-019** | renumbered |
  | REQ-C-021 | **REQ-C-020** | renumbered; write-back folded under the single add `When` |
  | REQ-C-022.1 | **REQ-C-021** | split out — note-type mismatch, one event |
  | REQ-C-022.2 | **REQ-C-022** | split out — migration add failure, one event |
  | REQ-C-023, REQ-C-024 | **unchanged identifiers** | REQ-C-024.3 / .4 rewritten in place |

- CHANGED: REQ-C-007.2's aesthetic vocabulary replaced by seven named CSS custom
  properties with literal values and the declarations that consume them, so
  `AC-C-005` can assert exact text. `design.md` § 4.3 keeps the rationale.
- CHANGED: REQ-C-024.3 no longer prohibits **reading** a foreign note type — the
  `modelNames` probe of REQ-C-002 and the `modelFieldNames` read of REQ-C-005.2
  are both required, so the prohibition as written contradicted them. REQ-C-024.4
  narrowed to the stock **types, templates, and CSS**, matching § 4.4's scope, so
  it no longer contradicts REQ-C-005.2's ordinary note-level sync.
- CHANGED: multi-actor requirements (REQ-C-005, REQ-C-008, REQ-C-020) restated
  with a single subject and `realized as:` clauses, so PASS/FAIL is defined for
  the requirement as a unit.
- CHANGED: REQ-C-006.2 gains the empty-token sentinel `unnamed`; REQ-C-015 and
  REQ-C-022 name their diagnostic code literals; REQ-C-012.3 cites `design.md`
  § 9.3 for the stored-name function; the `$…$` heuristic moved to § 2.
- CHANGED: pattern labels corrected — `[Unwanted]` → `[Ubiquitous — negated]`,
  `[While + Unwanted]` → `[While — negated]`. `Unwanted` is the EARS-legacy name
  for the negated form of an existing pattern, not a sixth GEARS pattern; the
  v0.1.1 census below is corrected accordingly.
- CHANGED: § 4.4 now records **three** superseded or inherited parent § 4
  exclusions, the third being the parent's "note-type changes on an
  already-synced entry", which REQ-C-020 supersedes for the confirmed-migration
  case only.
- CHANGED: § 7 declares the install request document and the `migrate_candidate`
  `action` value as `[NEW]` contract surfaces; `acceptance.md` gains an AC for
  the Elisp → Go stylesheet transport and tightens nine criteria; DoD coverage
  floors raised to the measured figures and `make fmt-check` added.
- UNCHANGED: § 4.2's verbatim REQ-016 amendment, § 4.3's AC-016 replacement pair,
  § 5 exclusions (bar the two wording fixes above), § 6 constraints, and every
  `plan.md` design decision and milestone. Acceptance criteria remain 24
  top-level.

### v0.1.1 (2026-09-05)

- CHANGED: § 3 consolidated from **42** requirements to **24**, renumbered
  contiguously `REQ-C-001`..`REQ-C-024`. This is a **renumbering, not a scope
  cut** — the Tier L budget in
  `.claude/rules/moai/workflow/spec-workflow.md` § SPEC Complexity Tier caps
  requirements and
  acceptance criteria at 25 each, independently, and states that exceeding the
  cap is a signal to tier up or split rather than to relax the budget. Every
  obligation the 42 entries carried survives, either inside a merged
  requirement's numbered sub-clauses or — for the four entries of the former
  § 3.8 "non-goals carried as requirements" — as the non-normative § 5 exclusion
  prose that already carried them. Requirements were merged only where they bind
  the same actor under the same GEARS trigger, so each merged requirement still
  matches exactly one GEARS pattern. All five patterns remain represented — the
  census, **corrected at v0.1.2** to stop listing the negated `shall not` form as
  a sixth pattern: Ubiquitous 13 (of which 6 negated), event-driven `When` 4,
  state-driven `While` 2 (of which 1 negated), capability-gate `Where` 2,
  event-detected 3.
- CHANGED: `acceptance.md` unchanged in count at 24 top-level AC (already within
  budget); every `covers REQ-…` pointer re-pointed to the new identifiers, and
  every one of the 24 requirements is covered by at least one AC. No AC was
  added or removed.
- CHANGED: cross-references re-pointed in `plan.md` (§ C rewritten to the 24),
  `design.md`, `research.md`, and `spec-compact.md` (regenerated). The
  historical artifacts `interview.md`, `research-lenses.md`, and
  `review-lenses-1.md` deliberately retain their v0.1.0 identifiers as the
  record of what was approved at Decision Point 1; read them through the
  old→new mapping table in `plan.md` § Revision 3.
- REMOVED: `related_specs` from the frontmatter — not in the schema's
  optional-field table, and `amendment_of: SPEC-ANKI-001` already carries the
  link. `tier`, `amendment_of`, the `### Amendments` sub-section, and the
  recorded-but-unavailable `prior_completed_sha` are unchanged.
- UNCHANGED: § 4 dispositions, § 5 exclusions, § 6 constraints, § 7 brownfield
  delta, and every `plan.md` design decision, milestone, and risk disposition.
  The Decision Point 1 approval of `plan.md` revision 2 stands.

### v0.1.0 (2026-09-05)

- INITIAL: SPEC created via `/moai plan` Phase 10 from `plan.md` revision 2
  (approved at Decision Point 1 on 2026-09-05), `interview.md` (14 binding
  decisions across three rounds plus the SPEC-ID correction), and `research.md`
  (four read-only lenses, synthesized). Tier L, 5 plan-phase artifacts.
- Adds imoogi-owned Anki note types (`imoogi-Basic`, `imoogi-Cloze`) carrying
  an imoogi-authored light-and-night stylesheet with a per-deck styling hook,
  converts Org math to Anki's MathJax delimiters, uploads local images into
  Anki's media collection, and migrates the imoogi-created notes currently
  sitting on stock note types.

### Relationship to SPEC-ANKI-001

`SPEC-ANKI-001` (out-of-repo at `/Users/jay/workspace/imoogi-org-anki/.moai/specs/SPEC-ANKI-001/`;
Tier L, 22 REQ / 25 AC, six milestones, 25/25 AC pass, behaviorally closed) is the
parent SPEC that shipped one-way Org → Anki synchronization on the **stock**
`Basic` and `Cloze` note types. This SPEC does **not** edit the parent's files.
It carries the following dispositions as its own normative text (see § 4):

- **Amends parent REQ-016** — the note-type-template and collection-styling
  prohibition narrows by ownership; every other clause of REQ-016 carries over
  unchanged and unweakened. Exact amended text in § 4.2.
- **Replaces parent AC-016** — the per-run blanket assertion is replaced by an
  ownership-scoped pair (§ 4.3), realized as `AC-C-003a` / `AC-C-003b` in
  `acceptance.md`.
- **Supersedes parent § 4's "custom (non-built-in) Anki note types" exclusion**,
  for imoogi-authored types only. User-authored custom note types stay entirely
  out of scope and untouchable.
- **Supersedes parent § 4's "note-type changes on an already-synced entry"
  exclusion**, for the confirmed-migration case only. Outside a confirmed
  migration the exclusion stands unchanged, and REQ-C-021 is its restatement.
- **Inherits the parent's deferred "collection ownership guarantees" obligation**,
  in the narrow form of REQ-C-024.
- **Assumes the deferred migration of parent `plan.md` D-12** for the bounded
  case this SPEC creates: the imoogi-created notes recorded against stock types.
  Parent REQ-021 is **not** repealed.

### Amendments

- `amendment_of: SPEC-ANKI-001` (successor amendment — the amended SPEC is the
  out-of-repo parent, not this file).
- Prior completed version: parent `v0.7.1`.
- `prior_completed_sha`: **not available** — the parent lives in a different
  checkout, its frontmatter still reads `status: in-progress`, and its
  `progress.md` records 25/25 AC pass with six completed milestones. Treated as
  behaviorally closed and specification-live. Recorded, not acted on.
- Rationale: the parent, as written, forbids exactly the feature this SPEC
  delivers (parent REQ-016 names `createModel` / `updateModelStyling` /
  `updateModelTemplates` with no ownership carve-out). Nothing in § 3 is
  implementable until the § 4 dispositions are in force.
- Scope: parent REQ-016 (one clause), parent AC-016 (replaced), parent § 4
  (three exclusions), parent `plan.md` D-12 (assumed). No other parent
  requirement, criterion, or decision is touched.

## 1. Overview

### 1.1 Purpose

Cards produced by `imoogi-anki` today inherit whatever styling the user's Anki
collection already carries, math reaches Anki as literal `$…$` that MathJax never
renders, and a local `[[file:diagram.png]]` reaches Anki as an `<img src>` naming
a path Anki cannot resolve. This SPEC gives imoogi its own note types so it can
own their presentation, converts Org math into the delimiters Anki's bundled
MathJax actually recognizes, and uploads locally referenced images into Anki's
media collection so they display offline.

### 1.2 Scope

In scope:

- Two imoogi-owned note types, created and kept current by an explicit install
  step that rides `imoogi-anki-setup`.
- One imoogi-authored base stylesheet — light and night mode, system fonts only,
  no network resource of any kind — plus a per-deck CSS **hook** derived from
  Anki's `{{Deck}}` special field, and a user-authored stylesheet appended after
  the base.
- Org-math → MathJax delimiter conversion over all four Org math syntaxes.
- Local image resolution, upload, and `src` rewriting, including the two-part
  `[[file:x.png][description]]` link form.
- A one-time, explicitly confirmed migration of the imoogi-created notes still
  recorded against stock note types.
- The parent-SPEC amendment of § 4, without which none of the above is
  implementable.

Out of scope: § 5.

### 1.3 Reading order

§ 4 is the blocking item and may be read first. § 3 is the requirement layer;
`acceptance.md` is the verification layer; `design.md` carries the pipeline,
the normalization algorithm, the install and migrate sequences, and the wire
surfaces; `plan.md` carries the design decisions `D-C-1`..`D-C-15` and the
milestone sequence `M1`..`M6`; `research.md` carries the evidence.

## 2. Glossary

**imoogi-owned note type** — an Anki note type whose model name begins with the
literal ASCII prefix `imoogi-`. Exactly two exist: `imoogi-Basic` and
`imoogi-Cloze`. Every other note type in the collection — the stock `Basic` and
`Cloze` types, and any user-authored type — is **foreign**. The predicate is
name-based rather than registry-based on purpose: it is decidable from a single
`modelNames` response with no local state, which is what makes the assertion
"this request targeted an imoogi-owned model" checkable from a request log
alone. Used verbatim by REQ-C-001, REQ-C-002, REQ-C-003, REQ-C-004, and the
AC-016 replacement of § 4.3.

**Ordinary synchronization run** — an execution of the `sync` capability, of any
shape (add, update, deck move, delete, no-op). It is the hot path. It issues no
note-type-template, note-type-styling, scheduling, review-history, or
collection-styling request of any kind.

**Install step** — the model-installation work that runs at the tail of
`imoogi-anki-setup`: probe `modelNames`, create the imoogi-owned types that are
absent, update the styling and templates of the ones that are present. It is the
only path on which imoogi issues a note-type write, and it never writes a foreign
model.

**Migrate subcommand** — the `imoogi-anki migrate` CLI entry point, reached only
from `imoogi-anki-setup` after an explicit user confirmation, that re-homes an
imoogi-created note from a stock note type onto its imoogi-owned counterpart. It
has a dry-run mode that reports candidates and issues no AnkiConnect write.

**Deck class normalization** — the total function from Anki's `{{Deck}}` value
(the full deck path, `::` separators included) to a CSS-identifier-safe token,
defined by REQ-C-006. The token is consumed only as the tail of the `deck-`
class the card template emits; it never appears in stored field content.

**Foreign model** — any model whose name does not begin with `imoogi-`; the
complement of *imoogi-owned note type*.

**Org inline-math heuristic** — the rule by which a single-`$` fragment is
recognized as math rather than as currency or punctuation: the enclosed text
contains at most two line breaks, is attached to both `$` characters with no
intervening whitespace, and the closing `$` is followed by whitespace or by
punctuation other than a dash. Defined here once and consumed by REQ-C-010.2;
`design.md` § 10.1 records why imoogi applies it itself rather than delegating.

**Base stylesheet** — the single imoogi-authored CSS asset embedded in the Go
binary, fixed by REQ-C-007. **User stylesheet** — the optional user-authored CSS
file of REQ-C-008. **Uploaded CSS** — the concatenation of the two, in that
order, which is the `css` value of every model-write request. These three terms
are used exactly as defined and are not interchangeable; "the stylesheet"
unqualified is not used.

**Install request document** — the `[NEW]` wire document
`{ protocol_version, ankiconnect_url, user_css }` that the front end writes to
the `install-models` subcommand's stdin (§ 7). It is a new document, not a field
added to an existing one, which is what keeps REQ-C-018's wire-stability clause
true.

## 3. Requirements (GEARS)

24 requirements across seven groups. Consolidated at v0.1.1 from the 42-entry
outline of `plan.md` § C revision 2 and re-partitioned at v0.1.2 — a
**renumbering, not a scope cut**: every obligation the 42 entries carried
survives here, either inside a merged requirement's sub-clauses or, for the four
pure non-goals, as non-normative exclusion prose in § 5. The old→new mapping
tables are `plan.md` § Revision 3 and § Revision 4. Identifiers use the `C` infix
so they never collide with the parent SPEC's `REQ-0NN` namespace, which is cited
from live code comments in this repository.

Each requirement carries exactly one GEARS trigger. Numbered sub-clauses inside a
requirement are **case-splits on that trigger's operand**, never additional
moments: a sub-clause narrows *which* operand the obligation applies to, or names
*which actor realizes* it, and never introduces a second `When`, `While`, or
`Where`. A complementary `Where X` / `Where not X` pair — REQ-C-008's
present-and-absent stylesheet arms are the only instance — is **one** gate stated
over both its arms, not two gates: the two arms are exhaustive and mutually
exclusive, so the requirement still has exactly one trigger and one PASS/FAIL.

Where a requirement binds more than one actor, its header names a single
subject and its sub-clauses say how each actor realizes that subject's
obligation, so PASS/FAIL is defined for the requirement as a unit.

Pattern labels name the five GEARS patterns. `[Ubiquitous — negated]` and
`[While — negated]` are the `shall not` forms of Ubiquitous and State-driven
respectively — the EARS-legacy label for that form was `Unwanted`; they are not a
sixth pattern.

### 3.1 Note types and installation

#### REQ-C-001 [Ubiquitous]
The imoogi system shall establish its note-type identity as follows:

1. It shall treat a note type as its own exactly when the model name begins with
   the literal prefix `imoogi-`, and shall recognize exactly two such types:
   `imoogi-Basic` and `imoogi-Cloze`. Any other model name beginning with
   `imoogi-` shall be neither created, updated, nor reported — it passes through
   every code path untouched and raises no diagnostic.
2. The imoogi-owned note types shall carry field names identical to their stock
   counterparts — `Front` / `Back` for `imoogi-Basic`, `Text` / `Back Extra` for
   `imoogi-Cloze` — so that the renderer's output map shape is unchanged and the
   planner's field-resolution layer needs no modification.

#### REQ-C-002 [When]
When the user invokes `imoogi-anki-setup`, the Go binary shall, after that
command's existing configuration steps, query `modelNames`, and then:

1. For each imoogi-owned type absent from the response, issue one `createModel`
   request carrying that type's field list, card templates, and CSS —
   `imoogi-Cloze` carrying `isCloze: true`.
2. For each imoogi-owned type already present in the response, issue
   `updateModelStyling` and `updateModelTemplates` for that type rather than
   `createModel`, so that repeated invocations of `imoogi-anki-setup` are
   idempotent.
3. For each styling update issued under sub-clause 2, replace that type's entire
   CSS with the base-plus-user concatenation, discarding any hand edit made to
   that type inside Anki, and emit into the install step's own report — before
   the first model-write request of that invocation is issued — a message
   stating that imoogi owns these types' styling outright and that hand edits
   will be replaced.

#### REQ-C-003 [While — negated]
While an ordinary synchronization run is executing, the Go binary shall not issue
`createModel`, `updateModelStyling`, or `updateModelTemplates` on any code path
that run reaches.

#### REQ-C-004 [Ubiquitous — negated]
The imoogi system shall not modify the templates or styling of any note type
whose model name does not begin with `imoogi-`, on any code path — superseding
the correspondingly narrowed clause of parent REQ-016 (§ 4.2), and leaving every
other clause of REQ-016 in force.

#### REQ-C-005 [Ubiquitous]
The imoogi system shall name its note types consistently across every surface
that carries a note-type literal, realized as:

1. The imoogi-owned types are the default for a newly marked heading: the front
   end's note-type-marking commands, its cloze auto-mark, and its
   `ANKI_NOTE_TYPE_ALL` completion allowlist shall name `imoogi-Basic` and
   `imoogi-Cloze`, and the Go renderer shall recognize both names in its
   note-type dispatch.
2. A heading whose `ANKI_NOTE_TYPE` names a stock `Basic` or `Cloze` shall
   continue to synchronize exactly as the parent SPEC's shipped behavior
   synchronizes it — the characterization baseline being the planner's existing
   stock-note-type add / update / no-op tests, recorded before modification per
   the § 7 brownfield processing order — carrying no imoogi theme: the Go binary
   shall recognize both the stock and the imoogi-owned name sets, including the
   `modelFieldNames` read the planner's field-resolution layer performs for a
   stock type, and the front end's cloze-marker handling shall accept either
   `Cloze` form.
3. Every site that hard-codes a stock note-type literal — the Go renderer's
   note-type constants and dispatch, the front end's `ANKI_NOTE_TYPE_ALL`
   allowlist and its note-type-writing and cloze-marking commands, and the
   front end's diagnostic table's corrective prose — shall name the imoogi-owned
   types wherever those types are the ones in play, while continuing to accept
   the stock literals per sub-clause 2.

### 3.2 Card appearance

#### REQ-C-006 [Ubiquitous]
The imoogi-owned card templates shall carry a deck-derived class hook:

1. Each imoogi-owned card template shall wrap its rendered content in a single
   element carrying a deck-derived CSS class computed from Anki's `{{Deck}}`
   special field, and that wrapper shall exist only in the card template — never
   in stored field content.
2. The deck-to-class normalization shall lowercase the deck path, replace each
   `::` separator with a single `-`, replace every remaining character outside
   `[a-z0-9_-]` with `-`, collapse consecutive `-` runs to one, strip leading and
   trailing `-`, prefix the guard character `_` when the result would otherwise
   begin with a digit, and — when every preceding step leaves the token empty —
   yield the literal sentinel token `unnamed`, so the emitted class is
   `deck-unnamed` and never the bare `deck-`.

#### REQ-C-007 [Ubiquitous]
The imoogi base stylesheet shall satisfy both of:

1. It shall carry both night-mode selector forms — `.card.nightMode` (Anki
   desktop, AnkiMobile) and `.card.night_mode` (AnkiDroid, with no space between
   the class names) — together with
   `.card img { max-width: 100%; max-height: none; }` and
   `word-wrap: break-word;`.
2. It shall define the following CSS custom properties with exactly these
   literal values, declaring the light values on `.card` and the night values
   under both night-mode selector forms:

   | Custom property | Light value | Night value |
   |---|---|---|
   | `--imoogi-bg` | `#f7f3e9` | `#1a1a1a` |
   | `--imoogi-fg` | `#1c1a17` | `#c8c4bc` |
   | `--imoogi-emphasis-fg` | `#0b0a09` | `#f5f2ec` |
   | `--imoogi-measure` | `44rem` | `44rem` |
   | `--imoogi-line-height` | `1.6` | `1.6` |
   | `--imoogi-serif` | `Iowan Old Style, Palatino Linotype, Palatino, Georgia, serif` | identical |
   | `--imoogi-sans` | `-apple-system, BlinkMacSystemFont, Segoe UI, Roboto, Helvetica Neue, Arial, sans-serif` | identical |

   and it shall consume them in exactly these declarations: `background` and
   `color` on `.card` from `--imoogi-bg` and `--imoogi-fg`; `font-family` on
   `.card` from `--imoogi-sans` and on `h1`-`h6` from `--imoogi-serif`;
   `line-height` on `.card` from `--imoogi-line-height`; `max-width` on the
   REQ-C-006 deck wrapper from `--imoogi-measure`; and `color` on its combined
   math-and-code rule from `--imoogi-emphasis-fg`. It shall name only the system
   font families the two stacks above enumerate, referencing no web font and no
   network-hosted resource of any kind. `design.md` § 4.3 records why these
   values were chosen; this requirement fixes what they are.

#### REQ-C-008 [Where + When]
Where a user stylesheet exists at the configured path — `imoogi-anki.css` beside
`imoogi.json` in the user's Emacs directory by default — when the install step
runs, the imoogi system shall upload, as that step's CSS, the base stylesheet
followed verbatim by that file's contents, realized as: the front end reading the
file and carrying its contents in the `user_css` field of the install request
document (§ 7), and the Go binary concatenating them after the base stylesheet
in the CSS it uploads, reading no stylesheet from disk itself. Where no such file
exists, the front end shall carry the empty string and the uploaded CSS shall be
the base stylesheet alone.

#### REQ-C-009 [Ubiquitous — negated]
The imoogi system shall not assign, rotate, or derive any per-deck colour, font,
or other appearance from a deck name or deck hash — the base stylesheet being
byte-identical for every deck, and every per-deck rule originating in the user
stylesheet of REQ-C-008.

### 3.3 LaTeX to MathJax

#### REQ-C-010 [When]
When rendered field content carries an Org math fragment, the Go binary shall:

1. Emit Anki's MathJax delimiters exactly as the following table fixes, and emit
   no other delimiter form:

   | Org math syntax | Emitted delimiters |
   |---|---|
   | `$…$` | `\(…\)` |
   | `\(…\)` | `\(…\)` |
   | `$$…$$` | `\[…\]` |
   | `\[…\]` | `\[…\]` |

2. Recognize a single-`$` fragment by the Org inline-math heuristic § 2 defines,
   and by no other rule.
3. Emit `<br>` in place of any line break the converted fragment contains.

#### REQ-C-011 [Ubiquitous — negated]
The Go binary shall not convert a `$` occurrence inside a `<pre>` region, a
`<code>` region, an HTML attribute value, or a link target, and shall not emit
any reference to a network-hosted MathJax or stylesheet resource.

### 3.4 Images and media

#### REQ-C-012 [When]
When rendered field content references a local image — by a relative path, by a
`file:` link, or by the two-part `[[file:x.png][description]]` form — the Go
binary shall:

1. For a reference the renderer emitted as an anchor rather than an image, whose
   target carries an extension in go-org's own image set — `png`, `gif`, `jpg`,
   `jpeg`, `svg`, `tif`, `tiff`, `webp`, `xbm`, `xpm`, `pbm`, `pgm`, `ppm`,
   `pnm`, case-insensitively — first rewrite that anchor to an `img` element
   whose `alt` carries the link description, and then apply sub-clauses 2 and 3
   to it exactly as to a one-part reference.
2. Resolve the reference against the directory of the entry's own Org file —
   derived from the configured sync root and the entry's recorded source path —
   and confine the resolved path to that sync root.
3. For a reference that resolves to a readable file, compute that file's stored
   name by the naming function `design.md` § 9.3 fixes — a sanitized basename,
   the separator `-`, and a truncated hex digest of the file's content SHA-256,
   the digest inserted **before** the extension — and rewrite the referencing
   `src` attribute to that exact name; the name is computed locally and
   therefore does not depend on any upload response.

#### REQ-C-013 [Ubiquitous — negated]
The Go binary shall not:

1. Rewrite, upload, or otherwise disturb an image reference whose target carries
   an `http:` or `https:` scheme.
2. Prefix an uploaded media filename with `_`.
3. Delete, or enumerate for deletion, any media file it did not upload during
   the run in progress.

#### REQ-C-014 [While]
While a single synchronization run is in progress, the Go binary shall:

1. Issue `storeMediaFile` only for entries that run actually adds or updates, an
   entry whose hash is unchanged issuing no media request at all.
2. Upload each distinct stored media filename at most once, however many entries
   reference it.

#### REQ-C-015 [When — event-detected]
When a referenced local image is detected to be unresolvable, unreadable, or
unuploadable, the Go binary shall report that sync target as a skipped entry
carrying `media_file_not_found` for a resolution or read failure and
`media_upload_failed` for an upload failure, each naming the reference, shall
create or update no note for it, shall leave that entry's existing registry hash
unchanged, and shall continue processing the remaining sync targets.

### 3.5 Hashing and transform ordering

#### REQ-C-016 [Ubiquitous]
The Go binary shall order transform and hash as follows:

1. It shall apply every content transform this SPEC introduces — MathJax
   delimiter conversion, two-part-link rewriting, and `src` rewriting — to
   rendered field content **before** the content hash is computed for the
   add/update/no-op decision, and the field content it sends to AnkiConnect
   shall be byte-identical to the content whose hash it records in the registry.
2. It shall decide add, update, or no-op for every previously synchronized
   imoogi note solely from that post-transform hash, so that a note whose
   rendered content differs under the sub-clause 1 transforms is updated and a
   note whose rendered content is byte-identical issues no request at all —
   including on the first run executed after this SPEC's rendering change, whose
   resulting invalidation is specified behavior rather than an error condition.
   (`design.md` § 2.3 records why.)

#### REQ-C-017 [Ubiquitous — negated]
The Go binary shall not add any input to the content hash beyond the four the
parent SPEC's D-6 fixes — note type, rendered field values, resolved deck, sorted
tag set — image content reaching the hash solely through the content-derived
stored filename that REQ-C-012 writes into the field text.

### 3.6 Migration of existing imoogi notes

#### REQ-C-018 [Ubiquitous]
The migration capability shall be reached through the `imoogi-anki migrate` CLI
subcommand, which shall read a request document of the same shape the `sync`
subcommand reads and write a response of the same schema, adding no field to any
existing wire document and leaving the protocol version constant unchanged. Its
dry-run mode shall report each candidate as one ordinary `results[]` entry
carrying the new `action` value `migrate_candidate` and that entry's existing
note identifier, so the candidate count is the length of `results[]` and no
response field is added; the front end shall accept that value as a
count-only outcome that triggers no write-back (§ 7).

#### REQ-C-019 [When]
When `imoogi-anki-setup` completes its install step and the registry records one
or more entries whose recorded note type is a stock type, the front end shall
display that count and shall obtain the user's explicit confirmation, stating
that migration discards each migrated note's review history and scheduling state,
before issuing any **writing** `migrate` request; a declined confirmation shall
issue none. (`design.md` § 7.1 records why obtaining the count does not
presuppose the answer.)

#### REQ-C-020 [Where + When]
Where migration has been invoked and confirmed under REQ-C-019, when an entry's
registry-recorded note type is a stock `Basic` or `Cloze` **and** its declared
note type is either that same stock type or that type's `imoogi-` counterpart,
the imoogi system shall re-home that entry onto the counterpart type, realized
as:

1. The Go binary shall render the entry under the counterpart type, add a new
   note under that type, and delete the original note only after the add has
   succeeded.
2. The Go binary shall then replace that entry's registry record with one naming
   the new identifier and the imoogi-owned note type, and shall report the new
   identifier in that entry's `results[]` record.
3. The Elisp front end shall overwrite that heading's `ANKI_NOTE_ID` property
   with the reported identifier and its `ANKI_NOTE_TYPE` property with the
   `imoogi-` counterpart of the type the heading currently declares, deriving
   that counterpart locally so no response field is added.

#### REQ-C-021 [When — event-detected]
When an entry's registry-recorded note type is detected to differ from its
declared note type — the hand-edited stock-to-`imoogi-` case included — the Go
binary shall report that sync target as a skipped entry carrying
`note_type_change_unsupported`, shall update no field on that note, and shall
continue processing the remaining targets, **unless** that detection occurs
inside a migration confirmed under REQ-C-019 and the declared type is the
recorded stock type's `imoogi-` counterpart, which is the REQ-C-020 case. Parent
REQ-021 is unchanged; the two requirements are partitioned **by requirement
text**, not by branch ordering.

#### REQ-C-022 [When — event-detected]
When a confirmed migration's add step is detected to have failed for an entry,
the Go binary shall leave the original note, its registry entry, and its heading
properties untouched, shall report the entry as skipped carrying
`migration_add_failed`, and shall continue processing the remaining entries, so
that no entry ever reaches a state where the registry names an identifier the
collection no longer holds.

### 3.7 Diagnostics, contracts, and non-interference

#### REQ-C-023 [Ubiquitous]
Every diagnostic code this SPEC introduces — `model_install_failed`,
`media_file_not_found`, `media_upload_failed`, and `migration_add_failed` —
shall exist as a `protocol.Code*` constant in the Go protocol package **and** as
a matching entry in the Elisp diagnostic table
(`modules/anki/imoogi-error.el`), the two sides paired one-for-one.

#### REQ-C-024 [Ubiquitous — negated]
The imoogi system shall not:

1. Detect, report, or delete media files in Anki's collection that no imoogi note
   references — unused-media cleanup being Anki's own Check Media function, which
   REQ-C-013's no-underscore rule deliberately keeps able to see imoogi's
   uploads.
2. Upload, rewrite, or report a diagnostic for a link whose target is a video or
   audio file, or whose extension lies outside the image set REQ-C-012
   enumerates — such a target passing through to Anki exactly as the renderer
   produced it.
3. Create or modify any note type beyond `imoogi-Basic` and `imoogi-Cloze` — the
   remainder of the parent SPEC's "advanced card types" exclusion staying in
   force as the non-normative § 5 exclusion it always was. Reading a foreign
   model is not prohibited: the `modelNames` probe of REQ-C-002 and the
   `modelFieldNames` read of REQ-C-005.2 are both required.
4. Modify the stock `Basic` and `Cloze` note types themselves — their model
   definitions, their card templates, or their CSS — on any code path, the
   migration path included. Note-level work on a note carried on a stock type is
   outside this prohibition and is governed by REQ-C-005.2 for an ordinary
   synchronization run and by REQ-C-020 for a confirmed migration. This clause
   supplies the narrow collection-ownership guarantee § 4.4 inherits from the
   parent SPEC's deferred obligation.

## 4. Parent-SPEC Amendment (blocking item)

`research.md` § 1 is unambiguous: the feature this SPEC proposes is **forbidden by
the parent SPEC as that SPEC is currently written**. All four research lenses
converged here independently. Nothing in § 3 is implementable until the four
dispositions below are in force.

### 4.1 The ownership predicate

Stated once in § 2 (*imoogi-owned note type*) so the amendment, the install
requirement, and the replacement acceptance criterion cannot drift apart.

### 4.2 REQ-016 amendment — scope the prohibition to foreign note types

Parent REQ-016 prohibits modifying "note-type templates, or collection styling,
on any code path". `createModel` / `updateModelStyling` / `updateModelTemplates`
are literally the actions it names, and the text carries no ownership carve-out.

**Amended text, verbatim** (this SPEC's REQ-C-004 supersedes the quoted clause of
parent REQ-016):

> The imoogi system shall not modify Anki review history, card scheduling state,
> **the templates or styling of any note type it did not itself create — that is,
> any note type whose model name does not begin with `imoogi-`** — or
> collection styling, on any code path, including the update and delete
> paths, and shall not issue any delete request scoped by tag, search query, or
> pattern rather than by explicit individual note identifier.

The three unamended words of the quoted clause — "or collection styling" — are
reproduced exactly as the parent writes them, since this SPEC asserts the rest of
REQ-016 is unchanged and unweakened. Everything else in REQ-016 — the scheduling
prohibition, the review-history prohibition, the identifier-scoped-deletion
prohibition — carries over
**unchanged and unweakened**. Only the note-type-template and styling clause
narrows, and it narrows by ownership, not by convenience.

### 4.3 AC-016 replacement — an ownership-scoped pair

Parent AC-016 ends: "…and no request in the run addresses any scheduling,
review-history, note-type-template, or collection-styling endpoint." That is a
**per-run** assertion; any model write anywhere in the run breaks it as written.
The replacement is **two** assertions, because the blanket survives for the hot
path and is lifted only on the install path:

- **The sync hot path — the blanket survives, verbatim in force.** Given a sync
  run of any shape, the count of requests recorded against `createModel`,
  `updateModelStyling`, `updateModelTemplates`, and every scheduling /
  review-history / collection-styling endpoint is **exactly 0**. Realized as
  `AC-C-003a`.
- **The install path — ownership-scoped.** Given an invocation of the install
  step, every recorded `createModel` / `updateModelStyling` /
  `updateModelTemplates` request names a model whose name begins with `imoogi-`,
  and the count of such requests naming any other model is **exactly 0**.
  Realized as `AC-C-003b`.

Both are mechanically checkable: the planner's fake client already logs calls per
action, so each new connector method arrives with a matching call log and "no
request fired" stays assertable.

### 4.4 Parent § 4 supersession — three exclusions, handled differently

- **"Custom (non-built-in) Anki note types"** — **SUPERSEDED**, for
  imoogi-authored types only. SPEC-ANKICARD-001 is the later SPEC that exclusion
  anticipated. User-authored custom note types remain entirely out of scope, and
  REQ-C-004 keeps them untouchable.
- **"Note-type changes on an already-synced entry"** — **SUPERSEDED for the
  confirmed-migration case only.** The parent excluded "any automatic migration
  of an existing note's fields, template, or model to match a newly-declared note
  type", deferring the delete-and-re-add to a later SPEC via `plan.md` D-12. This
  is that SPEC, and REQ-C-020 performs exactly that delete-and-re-add — but only
  inside a migration the user confirmed under REQ-C-019. Outside a confirmed
  migration the exclusion stands unchanged and REQ-C-021 restates it.
- **"Collection ownership guarantees"** — **INHERITED as a narrow obligation**.
  Because this SPEC now does touch models, it must supply the guarantee the
  parent could omit, and only the narrow one: the stock `Basic` (measured: 88
  notes) and `Cloze` (measured: 27 notes) types, their templates, and their CSS
  are byte-unchanged by every imoogi code path. Note-level work on notes carried
  on those types — an ordinary run's add, update, or delete, and a confirmed
  migration's re-home — is outside the guarantee and is governed by REQ-C-005.2
  and REQ-C-020. REQ-C-024 carries it. A general
  backup-and-restoration guarantee stays out of scope (§ 5).

### 4.5 D-12 assumption

Parent `plan.md` D-12 deferred "delete the old note, add a new one under the new
type, update the registry and `ANKI_NOTE_ID` accordingly" to a later SPEC. This
SPEC assumes that deferred work for the narrow, bounded case it creates: the
imoogi-created notes sitting on stock types (measured: 5). REQ-C-018..REQ-C-022
carry it. Parent REQ-021 is **not** repealed — it keeps governing every note-type
change the user authors by hand, and REQ-C-021 makes the two disjoint in
requirement text rather than in branch ordering.

## 5. Out of Scope

The following are deliberately **not** built by this SPEC. These are
**non-normative exclusions**, not requirements: they carry no `REQ-C-` identifier
and no acceptance criterion of their own. Where an exclusion has a testable
counterpart — a prohibition an implementation could violate — that counterpart is
stated in § 3 and cited here in parentheses; the rest is declarative scope
fencing. At v0.1.1 the former § 3.8 "non-goals carried as requirements" was
folded into this section, its four testable prohibitions surviving as the
sub-clauses of REQ-C-024.

### Out of Scope — Orphan media cleanup

- Detecting, reporting, or deleting media files in Anki's collection that no
  imoogi note references. Anki's own Tools ▸ Check Media owns unused-media
  cleanup; REQ-C-013's no-underscore rule exists precisely to keep imoogi's
  uploads visible to it.
- Any registry schema field tracking uploaded media. The registry holds only
  recomputable state; the schema stays forward-compatible on read, so a later
  SPEC may add one.

### Out of Scope — Video and audio media

- Uploading, rewriting, or diagnosing a link whose target is a video or audio
  file, or whose extension lies outside the image set REQ-C-012 enumerates. Such
  targets pass through untouched, with no diagnostic (REQ-C-024).
- Extending the recognized image-extension set beyond go-org's own — `.avif`,
  `.bmp`, and `.ico` are deliberately excluded, because the one-part and two-part
  link forms must agree and go-org decides the one-part case.

### Out of Scope — Remote image download

- Fetching, caching, or uploading an image whose target carries an `http:` or
  `https:` scheme. Such references are left exactly as authored (REQ-C-013), and
  the accepted cost is that they do not display when Anki is offline.

### Out of Scope — Automatic per-deck palette assignment

- Assigning, rotating, or deriving any per-deck colour, font, or other appearance
  from a deck name or a deck-name hash (REQ-C-009). imoogi emits the
  `deck-<class>` hook and one base stylesheet identical for every deck; every
  per-deck rule originates in the user's own stylesheet.
- A deck-to-style map expressed in `imoogi.json` or any other schema-constrained
  configuration surface.

### Out of Scope — Foreign note types and advanced card types

- Creating or modifying any note type beyond `imoogi-Basic` and `imoogi-Cloze`;
  restyling, re-templating, or otherwise touching a stock or user-authored note
  type (REQ-C-004, REQ-C-024). **Reading** a foreign model is not excluded and
  not prohibited — the `modelNames` probe and the `modelFieldNames` read the
  planner's field-resolution layer performs for a stock type are both required
  (REQ-C-002, REQ-C-005).
- Recognizing, installing, or reporting a third model whose name begins with
  `imoogi-`. Exactly two imoogi-owned types exist; any other `imoogi-`-prefixed
  model in the user's collection passes through every code path untouched and
  raises no diagnostic (REQ-C-001.1). Nothing is added to the diagnostic table
  for it.
- Image-occlusion cards, highlight-mask cards, type-in-answer cards, and
  card-direction control — the remainder of the parent SPEC's advanced-card-type
  exclusion.

### Out of Scope — Collection backup and restoration guarantees

- A general guarantee that user-authored note-type templates and collection CSS
  are persisted and restorable. Only the narrow byte-unchanged guarantee of
  REQ-C-024 is supplied here.

### Out of Scope — Note-type changes outside a confirmed migration

- Converting an already-synced entry from one note type to another by editing its
  `ANKI_NOTE_TYPE` property on the ordinary sync path. The parent SPEC's
  exclusion stands unchanged there and REQ-C-021 restates it; only the
  confirmed-migration case of REQ-C-020 is superseded (§ 4.4).

### Out of Scope — Per-heading migration opt-out

- Selecting which stock-type entries migrate. Migration is all-or-nothing per
  confirmation; declining is free and the prompt recurs at the next setup.

### Out of Scope — Scheduling preservation across migration

- Preserving review history or scheduling state for a migrated note. Anki
  attaches both to the note, and migration is a delete-and-re-add, so the loss is
  structural. It is surfaced in the confirmation text as a stated user cost, not
  hidden.

## 6. Constraints

- **Air-gap.** No card template, stylesheet, or generated field content may
  reference a network-hosted resource. Anki's bundled MathJax and system font
  stacks are the only permitted sources (REQ-C-007, REQ-C-011).
- **Wire stability.** No existing wire document gains a field, and the protocol
  version constant is unchanged; the install and migrate capabilities travel as
  new CLI subcommands (REQ-C-018).
- **Cross-language diagnostic contract.** Every new diagnostic code exists on
  both the Go and the Elisp side, enforced by the existing contract test
  (REQ-C-023).
- **Renderer purity.** The Org renderer stays a pure function of its inputs — no
  registry, no AnkiConnect client, no side effects — inherited unchanged from the
  parent SPEC's D-1. Media resolution is a separate post-render pass.
- **CSS identifier rules.** A class may not contain an unescaped `:` or space and
  may not begin with an unescaped digit, which is what makes REQ-C-006's
  normalization mandatory rather than cosmetic.
- **Methodology.** TDD (RED-GREEN-REFACTOR) per the project's configured
  development mode.
- **Coverage floor.** Each package's floor is its **currently measured** figure,
  so no `[MODIFY]` package may regress. Measured by
  `go test -count=1 -cover ./internal/...` in this repository at `HEAD f6ee148` (2026-09-05):
  `orgdoc` 100.0, `hashing` 100.0, `planner` 90.3, `registry` 87.8,
  `ankiconnect` 84.3 (`protocol` reports no statements); floors are `planner` ≥ 90.3 and `ankiconnect` ≥ 85.0 (the run-phase norm — M1's five new client methods must lift it). The two new packages,
  `internal/anki/media` and `internal/anki/model`, carry no inherited floor and
  are held to the same 85.0 project norm. `acceptance.md`'s Definition of Done
  carries the same figures.
- **Toolchain.** Go 1.26 with go-org v1.9.1; verification via `make build-anki`,
  `make test` (Elisp + Go + shell), `make lint`, and `make fmt-check`.
- **Renderer-version contingency** (`plan.md` R1). The two behaviors of
  `design.md` § 10.3 — go-org's `$`-fragment classification and its handling of
  sub/superscript inside a math fragment — are carried as regression tests
  against the pinned go-org version rather than as assertions about upstream. If
  a pinned-version upgrade turns either test red, the disposition is to hold the
  pin and open a follow-up SPEC, not to relax the test: the tests exist to detect
  exactly that change.

## 7. Brownfield Delta

The repository is brownfield: every module below already exists except where
marked `[NEW]`.

### [DELTA] AnkiConnect client (`internal/anki/ankiconnect`)
- [EXISTING] The ten-method `AnkiConnector` interface, the single `call`
  chokepoint, and the `TransportError` / `ProtocolError` / `APIError` tiering —
  unchanged context; characterization tests only.
- [MODIFY] Extend the interface and the concrete client with `ModelNames`,
  `CreateModel`, `UpdateModelStyling`, `UpdateModelTemplates`, and
  `StoreMediaFile`. Every existing method keeps its behavior.

### [DELTA] Org renderer (`internal/anki/orgdoc`)
- [EXISTING] `Render`'s signature and its purity contract; the cloze-marker
  post-render guard — the shape the new transform follows.
- [MODIFY] Add `imoogi-Basic` / `imoogi-Cloze` to the note-type constants and
  dispatch, keeping the stock arms (REQ-C-005); add the MathJax post-render
  transform beside the existing cloze guard.

### [DELTA] Planner (`internal/anki/planner`)
- [EXISTING] The field-resolution layer (it exists to handle a customized stock
  `Basic` carrying lowercase field names — the user's measured situation, and the
  reason new code must not bypass it); the recomputed ownership predicate on the
  orphan-deletion path.
- [MODIFY] Apply the new transforms before the hash is computed (REQ-C-016); add
  the migration branch, disjoint from the existing note-type-change skip by
  requirement text (REQ-C-021); issue `StoreMediaFile` on the add and update
  paths only (REQ-C-014).

### [DELTA] Protocol (`internal/anki/protocol`)
- [EXISTING] The `Request` / `Config` / `Response` / `Result` documents and the
  version constant — **no field is added to any of them, and the version
  constant is unchanged** (REQ-C-018). `Result` stays `{key, action, note_id}`,
  which is why REQ-C-020.3 has the front end derive the `imoogi-` counterpart
  locally rather than read a note type off the wire.
- [EXISTING] The fourteen diagnostic codes, `note_type_change_unsupported`
  among them — reused unchanged by REQ-C-021.
- [NEW] Four `Code*` constants, one per new failure mode:
  `model_install_failed`, `media_file_not_found`, `media_upload_failed`,
  `migration_add_failed` (REQ-C-023).
- [NEW] One `action` **enum value**, `migrate_candidate`, joining the five the
  constant block already carries (`added`, `updated`, `skipped`, `deleted`,
  `failed`). This is a value, not a field: `Result`'s shape is untouched and
  `Version` stays constant, so REQ-C-018's wire-stability clause holds. It is
  nonetheless a **contract change the Elisp side must handle**, and is listed
  here and in the Elisp delta below for that reason.
- [NEW] The install request document `{ protocol_version, ankiconnect_url,
  user_css }`, read by the `install-models` subcommand from stdin. A new
  document rather than a field added to `Request`, which is what carries
  REQ-C-008's stylesheet transport without touching the sync wire contract.

### [DELTA] Content hashing (`internal/anki/hashing`) — EXISTING, unchanged
- [EXISTING] The four hash inputs and the round-trip invariant that every input
  must be recoverable from a `notesInfo` response. **No new hash input is added**
  (REQ-C-017). Characterization tests only.

### [DELTA] Registry (`internal/anki/registry`) — EXISTING, unchanged
- [EXISTING] The five-field entry and the atomic write. Migration rewrites the
  *values* of an entry's identifier and note type (REQ-C-020); the **schema** is
  untouched.

### [DELTA] Media pass (`internal/anki/media`)
- [NEW] Resolve → confine → content-hash → rewrite, including the
  two-part-anchor-to-`img` rule, over rendered HTML with the entry's own base
  directory supplied. Reuses the existing path-confinement logic from the
  org-preview asset resolver; the org-preview link regex is **not** reused (it
  belongs to a different parser).

### [DELTA] Note-type model package (`internal/anki/model`)
- [NEW] Embedded card templates and base stylesheet, the deck-class normalizer,
  the base-plus-user CSS concatenation, and the probe-then-act install logic.
  Reads no stylesheet from disk.

### [DELTA] CLI (`cmd/imoogi-anki`)
- [EXISTING] The `run` dispatch and its `sync` case; the protocol-version probe.
- [MODIFY] Two new sibling cases: `install-models` (internal surface, invoked
  only by `imoogi-anki-setup`) and `migrate` (with a dry-run mode). The E2E stub
  gains both.

### [DELTA] Elisp front end (`modules/anki/*.el`)
- [EXISTING] The write-back path's property overwrite, the process-layer request
  construction, and the pinned protocol-version constant.
- [MODIFY] `imoogi-setup.el` gains the install-then-scan-then-confirm tail;
  `imoogi.el` gains the user-stylesheet `defcustom` and switched note-type
  literals; `imoogi-process.el` constructs the two new subcommand requests — the
  install request document above, carrying the user stylesheet's contents in
  `user_css` (REQ-C-008) — and **handles the new `migrate_candidate` action
  value** as a count-only outcome that triggers no write-back, the candidate
  count being `(length results)`; `imoogi-error.el` gains one table entry per new
  code and updated corrective prose.

### [DELTA] Module loader (`modules/24-anki.el`)
- [MODIFY] `ANKI_NOTE_TYPE_ALL`, the two note-type-writing commands, and the
  cloze auto-mark — write the imoogi-owned names, accept both name sets.

### [DELTA] Tests (`tests/`, `internal/**/*_test.go`)
- [EXISTING] The Go-const-to-Elisp-table contract test; the planner's fake client
  and its per-action call logs.
- [MODIFY] The fake client implements the five new methods with matching call
  logs, so every negative assertion in `acceptance.md` stays checkable.
- [NEW] Table tests for the normalizer, the math transform, the media rewrite,
  and the CSS concatenation; the transform-before-hash byte-equality guard.

### [DELTA] User-owned stylesheet (`imoogi-anki.css`) — NOT shipped
- [EXISTING] Absent by default. imoogi never creates, ships, or version-controls
  it; its absence is the normal case.

## 8. Dependencies

- **Amends** `SPEC-ANKI-001` (out-of-repo): REQ-016 (§ 4.2), AC-016 (§ 4.3), § 4
  "advanced card types" and "collection ownership guarantees" (§ 4.4), and
  `plan.md` D-12 (§ 4.5).
- **Inherits unchanged** from the parent: D-1 (go-org; Emacs `ox-html`
  rejected), D-6 (hash inputs and the post-render rationale), D-9
  (identifier-based, content-confirmed deletion), D-10 (deck is a card
  property), and REQ-021 (restated as REQ-C-021's complement).
- **No dependency** on `SPEC-TRANSIENT-001` — disjoint module set.
- **Depended on by**: none currently.

## 9. Traceability

- Acceptance criteria: `acceptance.md` (`AC-C-001` .. `AC-C-024`, with
  sub-lettered sub-criteria).
- Technical design — pipeline, ordering invariant, normalization algorithm,
  install and migrate sequences, wire surfaces, Elisp touch points: `design.md`.
- Design decisions `D-C-1`..`D-C-15`, touch-surface table, milestones `M1`..`M6`,
  risk register `R1`..`R15`, Tier rationale, and the two requirement-renumbering
  mapping tables (§ Revision 3 for 42 → 24, § Revision 4 for the v0.1.2
  re-partition): `plan.md`.
- Evidence, contradictions `C1`..`C6`, open-question verdicts, and the post-plan
  measurements: `research.md` (see § 14 for the R1 probe and the go-org
  passthrough measurement).
- Clarification record — 14 binding decisions across three rounds plus the
  SPEC-ID correction and the Decision Point 1 gate: `interview.md`.
