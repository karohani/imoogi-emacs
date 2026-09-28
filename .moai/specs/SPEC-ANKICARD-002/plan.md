# Plan — SPEC-ANKICARD-002

Implementation plan for the card-option transport and validation layer.
Decisions are ordered by **change-likelihood**, highest first: the data-model
and user-visible-behavior choices lead, the mechanical ones close. Review
attention belongs at the top.

## §A Context

Backlog card **t13** of the design record
`.moai/reports/anki-card-types-plan-20260920.md`. Two later cards — t14
(Multiline) and t15 (Swift) — cannot begin until three per-heading options
reach the Go back end. This SPEC delivers that road and deliberately drives
nothing on it.

The design record § 2 carries the decisions already settled with the user:
three property names, inheritance identical to `ANKI_DECK`, no
`ANKI_CARD_KIND`, note type written at edit time and only validated at sync
time, protocol version 1 → 2. Those are **fixed inputs** to this plan, not open
questions. What follows settles the questions the card left open.

Sequencing: SPEC-ANKICARD-001 is `in-progress`, and its closure is **not** a
precondition of this SPEC's run phase. The design record § 5 risk table places
t12 through t15 after that closure; its own later ordering note supersedes that
line, recording that the operator — told SPEC-ANKICARD-001 was still
`in-progress` — directed that cards **t11 through t15** be worked in order. The
range names this card. The operator repeated the directive in the session that
produced this SPEC, again with the blocker stated first. `spec.md` § 8 carries
the full disposition, including the confirmation that nothing this SPEC reads
from SPEC-ANKICARD-001 is unlanded.

t11 and t12 have **landed as commits on `main`** —
`d053c67` (t11, cloze brace escaping) and `5d3868f` (t12, the supplementary
field). The working tree carries no uncommitted Anki change. This SPEC's base
commit is therefore `5d3868f` or a later descendant of it.

The design record § 7 states the two cards were not yet committed; that
sentence predates the commits above and is stale. Read the git history, not
that line.

## §B Known Issues in the Surrounding Code

Not defects this SPEC fixes — context a reader needs to avoid mis-attributing
a test result.

1. **t12 caused a one-time mass `updated`.** Adding the supplementary field to
   the cloze output changed every existing cloze note's hash, because the hash
   covers every key of the rendered field map. The first sync after t12 reports
   every cloze note as updated with no content change. A byte-identity golden
   recorded before t12 would therefore fail for reasons this SPEC did not
   cause — which is why AC-OPT-012a pins the baseline to the **post-t12** tree.
2. **The orphan-deletion hash comparison is an open hypothesis.** The design
   record § 7 records that the orphan-confirmation predicate may have been
   holding back every cloze deletion before t12. Unverified, biased safe, and
   untouched here.
3. **The front end parses the response's protocol version and never compares
   it.** Skew is caught on the request side, at the binary's probe, in both
   directions. This is sufficient (AC-OPT-006) and is not a gap this SPEC
   closes.
4. **`migrate --dry-run` counts candidates without rendering them, so its
   count can exceed what a confirmed run actually migrates.** The dry-run arm
   emits its candidate result and continues before reaching the per-entry
   render, so an entry that a confirmed run would skip with
   `cloze_marker_missing` is still counted in the number the confirmation
   prompt states. This SPEC **inherits** that gap rather than widening it in
   kind: an entry the new gate rejects is counted the same way, for the same
   reason. Closing it means running the whole per-entry pipeline in dry-run,
   which is a change to the consent gate's own design and belongs to whatever
   SPEC decides the count must be exact.

## §C Pre-flight — facts verified against the tree

Each was read this session, not assumed.

| Fact | Evidence |
|---|---|
| The renderer has exactly two non-test callers | `internal/anki/planner/planner.go:273` and `internal/anki/planner/migrate.go:188`; a repo-wide grep for `orgdoc.Render` finds no third non-test site |
| Both callers go through the declared-type mapping | `renderType` at `internal/anki/planner/planner.go:372` collapses `imoogi-Basic`→`Basic` and `imoogi-Cloze`→`Cloze`, passing any other name through |
| The hash covers no card option | `internal/anki/hashing/hashing.go` hashes note type, every key of the field map, deck, and sorted tags — the options reach none of them |
| Every diagnostic code is declared in one const block | `internal/anki/protocol/protocol.go`; the front end's table is `modules/org/anki/imoogi-error.el`; the pairing runs both ways in `tests/anki-error-test.el` |
| The pairing test cross-checks a **hardcoded** list against the Go source by regexp | `imoogi-error-test--go-emitted-codes` in `tests/anki-error-test.el`; a new code that is not added there fails `imoogi-error-test-codes-match-source-file` |
| There is exactly one entry-plist producer | `imoogi-scan--file` in `modules/org/anki/imoogi-scan.el`; `imoogi-target-scan.el` groups targets and delegates all parsing to it |
| Version skew is probed once, shared by every subcommand | `probeProtocolVersion` at `cmd/imoogi-anki/main.go:236`, called from `:137` (sync/migrate) and `:278` (install-models) |
| The nearest-wins chain is a reusable generic function | `imoogi-props-resolve` in `modules/org/anki/imoogi-props.el` takes a property name; the deck and tags resolvers are thin wrappers over it |

## §D Design Decisions (highest change-likelihood first)

### DD-1 — The three options travel as nullable **strings**, not as typed values

`protocol.Entry` gains `Direction`, `Incremental`, and `Swift`, each a
`*string` with JSON keys `direction`, `incremental`, `swift`. Null means the
chain resolved to nothing.

Rejected alternative: `Incremental bool` / `Swift bool`. A boolean on the wire
requires someone to decide which spellings are true, and the only actor upstream
of the wire is the front end — which the design record § 2.1 principle 3 and
REQ-OPT-002 both bar from interpreting values. A boolean wire would also make
`card_option_invalid` unable to name the offending text, because the text would
already be gone.

Consequence to keep in view: `"nil"` and `null` are different things on this
wire. `null` is "no value resolved"; `"nil"` is "the user explicitly wrote the
falsy spelling". They behave identically at the validation gate today, and
nothing should come to depend on distinguishing them.

Nullable pointers rather than omitted keys follows the contract's existing rule,
stated in the protocol package doc: a dropped key and a null value are not
interchangeable here.

### DD-2 — Three diagnostic codes, not one — a reversible choice

The card mandated one new code, `card_option_needs_cloze`. This plan adds two
more: `card_option_invalid` and `card_option_conflict`.

The code is what the front end's table turns into a corrective action, and the
three failures have three different fixes — change the note type or drop the
option; fix the value; remove one of two conflicting options. Folding them into
one code produces a message that has to enumerate all three.

**This is the reversible decision in this plan.** Collapsing to one code costs
two const entries, two table entries, two list entries, and a rewrite of
AC-OPT-009 through AC-OPT-011 into one criterion. Nothing structural depends on
the split. If the reviewer prefers the card's literal one-code scope, say so at
the plan gate and the collapse is mechanical.

**Reviewed and upheld at plan audit.** The auditor reached the same conclusion
independently and added two arguments worth keeping beside the decision:

- **The precedent is in the existing codes, and it splits the same way.** The
  18 declared codes split by distinct corrective action rather than by
  subsystem: two for two deck operations, two for two media failures, and —
  the closest analogue — `delete_suppressed` and `delete_candidate_unowned`,
  two codes for two reasons one delete did not happen. Collapsing would make
  the new code the only one in the set covering three corrective actions.
- **The naming argument settles it alone.** `card_option_needs_cloze` is the
  name the backlog card mandated, and it *describes the note-type fix*.
  Emitting it for a malformed arrow would make the code name assert something
  false about the defect — actively misleading, not merely coarse.

One correction to how this decision was first argued above. The rule in
`protocol.go` and `tests/anki-error-test.el` mandates **pairing**, not
granularity: it requires that every code have a message and every message a
code, and is silent on how finely failures are split. The granularity argument
rests on the precedent, not on that rule. The conclusion is unchanged; its
footing is.

### DD-3 — An unrecognized value rejects the entry; it is never ignored

Rejected alternative: treat an unrecognized value as absent.

The entry is a flashcard in a spaced-repetition collection. Ignoring `-->` when
the user meant `->` produces a card that lacks the reversal they asked for, with
no signal, and the user discovers it weeks later during review. A skip is loud,
costs one sync run, and names the line to fix.

The asymmetry is the argument: a wrong rejection is corrected in seconds; a
wrong acceptance is a silently defective card.

### DD-4 — Swift and multiline on one heading is a conflict, not a precedence

`ANKI_SWIFT` truthy alongside an arrow-valued `ANKI_DIRECTION` or a truthy
`ANKI_INCREMENTAL` rejects the entry. A falsy value on either side is off and
does not conflict (DD-5).

The card kind is inferred from which options are present (design record § 2.3
decision D). Both groups present therefore names two kinds, and no precedence
between them is defined anywhere. Inventing one here would bind t14 and t15 to
a rule neither has asked for, and would produce a card the user cannot predict
from what they wrote.

The inheritance interaction is real and handled, not sidestepped: because all
three options inherit, a file-level `#+PROPERTY: ANKI_SWIFT t` plus a
heading-level `ANKI_DIRECTION` conflicts on that heading. The escape hatch is
the one the chain already provides — an empty value in the heading's own drawer
terminates the chain (`:ANKI_SWIFT:` with nothing after it). This is the same
suppression `ANKI_DECK` has always had, not a new mechanism, and the conflict
message points at it.

### DD-5 — The spellings are `t` and `nil`, trimmed, case-insensitive — and `nil` is recognized on all three

`t` is on; `nil` is off; both are compared after trimming surrounding
whitespace and without regard to case; everything else is
`card_option_invalid`.

For the two boolean properties this fills a gap rather than expanding a
decision: the design record § 2.1 names an on-value for each and specifies no
off-spelling at all, so some falsy spelling had to be chosen.

**`ANKI_DIRECTION` accepting `nil` is a genuine expansion of a confirmed
decision, and is flagged here as DD-2 is.** That property's value set is
enumerated exhaustively in the design record as three arrows; adding a fourth
recognized value goes beyond filling a gap. Surfaced at the plan gate for the
same reason DD-2 is: the operator settled the enumeration and should be the one
to widen it.

The justification, stated without the circularity it is easy to fall into: the
reason is **not** that users are habituated to `nil` by the other two
properties, since those spellings are themselves introduced by this SPEC. The
reason is that an explicit opt-out spelling that works on two of three
inheriting properties and errors on the third is a trap — the failure is
invisible on a drawer line and the user has no way to predict which property
behaves which way.

Collapse cost if declined: one clause in REQ-OPT-009.1, two rows in
AC-OPT-009's table, one row each in AC-OPT-010's table and the edge-case table.
Nothing structural depends on it. Declining leaves the empty-value form
(`:ANKI_DIRECTION:` with nothing after it) as the only direction opt-out, which
still works — it is the chain's own termination rule — so the feature is not
lost, only its second spelling.

`t` / `nil` rather than `true` / `false` or `yes` / `no` because the values are
written in Org by an Emacs user, and these are the spellings that surroundings
use. Case-insensitivity costs one comparison and removes a whole class of skips
whose cause (`T` instead of `t`) is invisible on a drawer line. Trimming
matches what the chain already tolerates.

An explicitly falsy value is **off, not option-bearing** — and that holds for
*every* rule that reads option-bearing-ness, not only the conflict rule.
`swift: nil` beside `direction: ->` is not a conflict, because the user wrote
"not swift". `swift: nil` alone on a basic heading is likewise not a
note-type violation, for the same reason.

The second case is the one worth stating explicitly, because getting it wrong
breaks DD-4's escape hatch in its explicit form: a basic heading under a
file-level `#+PROPERTY: ANKI_SWIFT t` that writes `ANKI_SWIFT: nil` to opt out
would otherwise be rejected with `card_option_needs_cloze` — punished for
declining the option. The empty-value form of the escape hatch resolves to
null and never had this problem; the explicit form only works if falsy means
off everywhere. `spec.md` § 2 carries the definition that makes it so.

### DD-6 — The cloze-style predicate lives in the planner, derived from `renderType`

The predicate is `renderType(noteType) == orgdoc.NoteTypeCloze`. It introduces
no second list of note-type names, so it cannot drift from the renderer's own
dispatch — which is the failure mode a literal `[]string{"Cloze",
"imoogi-Cloze"}` would invite the first time a third name appears.

Rejected home: `internal/anki/model`. It is the package that owns note-type
identity (`OwnedPrefix`, `BasicName`, `ClozeName`, `IsOwned`), which makes it
the obvious candidate — but its package doc states that **nothing in it is
reachable from an ordinary synchronization run**, and says that isolation is
the point: it is what keeps SPEC-ANKICARD-001's per-run "no note-type write"
blanket verifiable. A predicate the sync hot path calls on every option-bearing
entry would break that stated invariant. This is the load-bearing reason, and
it outranks the aesthetic pull toward `model`.

Rejected home: `internal/anki/orgdoc`. It knows only the two stock names; it
would have to learn the imoogi-owned ones, duplicating exactly what `renderType`
exists to do.

The front end's mirror, `imoogi-anki-cloze-note-type-p`, stays where it is.
AC-OPT-007 asserts the two agree over a shared eight-value table rather than
trusting them to.

### DD-7 — One diagnostic per rejected entry, in a fixed order

`card_option_invalid` → `card_option_conflict` → `card_option_needs_cloze`.

Without a stated order, an entry offending against all three has an outcome
that depends on evaluation order in the implementation, which is untestable and
changes under refactoring.

The order runs cheapest-and-most-local first: a malformed value is a defect in
one drawer line and must be reported even when the other two rules would also
fire, since the conflict rule cannot correctly classify a value it does not
recognize. The note-type rule runs last because it is the one the user is most
likely to have intended differently, and reporting it while a value is still
malformed would send them to the wrong line.

### DD-8 — Migration validates on the declared type; the renderer call is untouched

The gate is one function reached from both the ordinary path and the migration
path. On the migration path it reads the entry's **declared** note type rather
than the migration counterpart.

The two always agree on cloze-style-ness: a type and its imoogi-owned
counterpart are both cloze-style or both not, by the counterpart relation
itself. Reading the declared type keeps one signature and one call shape for
both paths.

The renderer call in the migration path keeps its current arguments. This SPEC
renders nothing, so it passes nothing. Widening that call is named in
`spec.md` § 5 as the later card's work, together with the obligation to
re-examine whether migration must pass options at all once rendering depends on
them.

### DD-9 — Version skew rides the existing probe; nothing is negotiated

The bump is a constant change plus its pins. The binary's shared probe already
compares the request's declared version against its own and answers
`binary_incompatible` on any difference, in either direction, for every
request-reading subcommand. Nothing needs adding.

Rejected alternative: a compatibility window in which a version-2 binary
accepts a version-1 request by treating the three fields as null. It looks
kind and is not: the front end and the binary ship from the same checkout and
are rebuilt by one command, so skew means a stale build, and the useful
response to a stale build is to say so. A silent acceptance would let a user
write card options that the binary quietly ignores — the exact failure DD-3
rejects elsewhere.

### DD-10 — Resolution reuses the existing chain function, not a second one

`imoogi-props-resolve` already takes a property name and applies the full
nearest-wins chain, including the present-but-empty termination and the
`PROPERTY+` normalization. The three new resolvers are thin wrappers, exactly as
the deck and tags resolvers are.

A second implementation would let the chain's semantics diverge between
`ANKI_DECK` and a card option — a divergence nothing would catch, since no test
compares the two chains.

## §E Self-Verification Before Handoff

Before the run phase declares a milestone complete:

1. The golden corpus of AC-OPT-012a exists and was recorded **before** the
   first production edit. A golden recorded afterward proves nothing.
2. Every protocol-version pin in the AC-OPT-005 table was read, not assumed.
   Re-run the grep; read each hit.
3. `tests/anki-error-test.el` passes without the front-end-only allowlist
   growing. A new code added to that allowlist instead of to the table is the
   pairing contract being routed around.
4. Coverage floors were **measured on the base commit**, not carried over from
   the design record's t12 figures.

## §F Milestones (priority-ordered; no time estimates)

### M1 — Baseline capture (no production change)

Record the render golden corpus, the hash-invariance fixtures, and the
option-free request log that AC-OPT-012 compares against. Measure and record
package coverage for `protocol`, `planner`, and `orgdoc` on the base commit.

This is first because every later milestone's non-interference claim is
measured against it, and a baseline taken after a change is not a baseline.

**Covers**: the measurement side of REQ-OPT-012.

### M2 — Wire contract

Add the three fields to the entry shape; bump the version constant to 2; add
the three diagnostic-code constants. Update every pin in the AC-OPT-005 table
on both sides of the wire.

The code constants land here rather than with the gate that raises them,
matching the existing const block's own stated convention: the front end's
table and its contract test key on the constant set, so a code added later than
its table entry breaks the pairing either way.

**Covers**: REQ-OPT-004, REQ-OPT-005, REQ-OPT-006.

### M3 — Front-end resolution and transport

Add the three resolvers over the existing chain; attach the resolved values in
the single entry-plist producer; emit the three keys in the entry serializer.

**Covers**: REQ-OPT-001, REQ-OPT-002, REQ-OPT-003.

### M4 — Back-end validation gate

Add the cloze-style predicate; add the single gate, placed before the render
call and reached from both the ordinary and the migration paths; implement the
three rules in DD-7's fixed order.

**Covers**: REQ-OPT-007, REQ-OPT-008, REQ-OPT-009, REQ-OPT-010, REQ-OPT-011.

### M5 — Diagnostic pairing

Add the three message-table entries; add the three codes to the test-side code
list; confirm all four pairing assertions pass and the front-end-only allowlist
is unchanged.

**Covers**: REQ-OPT-013.

### M6 — Non-interference verification and documentation

Re-run M1's goldens against the finished tree. Document the three properties in
the README's Anki section: names, recognized values, inheritance, and the
cloze-style note-type requirement.

**Covers**: the assertion side of REQ-OPT-012.

## §G Test Strategy

- **Go, table-driven, at the planner.** The validation gate's three rules and
  their ordering are a value table; AC-OPT-009 through AC-OPT-011 are written
  as tables so the test is the table.
- **Go, golden, at the renderer and the hash.** Non-interference is a
  byte-comparison against M1's recording, not a judgement.
- **Go, request-log, at the planner.** "Issued no request" is asserted from the
  fake client's call log, which is how the existing suite asserts the same
  shape.
- **Elisp, ERT, at the chain and the scan.** Inheritance and suppression are
  buffer-level behaviors; they are asserted in an Org buffer, not by reasoning.
- **Elisp, contract, at the error table.** The pairing is already mechanical in
  both directions; this SPEC only enlarges its input.
- **Not tested here**: anything a card option renders. There is nothing to
  render.

## §H Anti-Patterns to Avoid

1. **Interpreting a value in Elisp** "because it is easy there". It makes
   `card_option_invalid` unable to name the offending text and puts the
   recognized-value set in two places.
2. **Adding a second note-type name list** for the cloze predicate. DD-6 exists
   because that list will drift.
3. **Recording the golden after editing.** It converts the strongest criterion
   in this SPEC into a tautology.
4. **Widening the renderer's signature** "while we are here, for t14". The
   moment it takes an option, REQ-OPT-012.1 is false and every
   non-interference claim needs re-argument.
5. **Adding a new code to the front-end-only allowlist** instead of to the
   message table. It makes the pairing test pass while defeating its purpose.
6. **Normalizing the body or trimming a field** anywhere on the path. t12
   already demonstrated what a stray normalization costs: every existing note
   reported as updated.

## §I Tier Recommendation — **M**

| Axis | Measurement | Tier |
|---|---|---|
| Scope | Roughly 400-700 LOC including tests; no new package, no new dependency | M (300-1000) |
| Production files | 8: the protocol, the planner, the migration path, and four front-end files, plus the README | M (5-15) |
| Requirements | 13 against the M ceiling of 16 | M, with headroom |
| Acceptance criteria | 14 against the M ceiling of 16 | M, with headroom |

The file count crosses 15 only when the contract-test updates are counted.
Those are a dozen single-value edits to an already-pinned constant, mechanical
and individually trivial — they add test-suite breadth, not design surface, and
counting them as design surface is what would tier this up on a technicality.

The Tier L artifact set is declined deliberately. `research.md` would carry
nothing the design record
`.moai/reports/anki-card-types-plan-20260920.md` does not already carry, that
record having been produced from the upstream project's wiki and source plus a
measured go-org render probe. `design.md` would carry § D above and nothing
more; the decisions here are ten, not a system architecture.

If the requirement or criterion count grows past 16 during the run phase, that
is the signal to tier up or split — not to relax the budget.

## §J Risks and Dispositions

| Risk | Degree | Disposition |
|---|---|---|
| A stray normalization on the render path causes a second mass `updated` | Medium | AC-OPT-012a's golden, recorded at M1 before any edit, is the guard. Anti-pattern 6 names the temptation. |
| SPEC-ANKICARD-001 is still `in-progress`, so its audit runs against a HEAD this SPEC has moved | Medium | Recorded, not mitigated. The operator authorized the ordering with the blocker stated — the design record's ordering note names t11 through t15, and the directive was repeated in this SPEC's own session. `spec.md` § 8 carries the disposition and the confirmation that nothing read from SPEC-ANKICARD-001 is unlanded; there is no run-phase gate anywhere in this SPEC. This SPEC touches none of SPEC-ANKICARD-001's requirements, so the overlap is on the tree, not on the specification. |
| A protocol-version pin is missed and a test passes for the wrong reason | Medium | The AC-OPT-005 table enumerates all fourteen; the verification step re-runs the grep rather than trusting the table. Both suites fail loudly on a missed pin, so the realistic failure is a **silently updated** pin in an unrelated subsystem — which is why the table names the four out-of-scope subsystems explicitly. |
| Users write a file-level option over a file whose entries are mostly basic | Medium | Every such entry is rejected with `card_option_needs_cloze`. Loud by design (DD-3), and the acceptance edge-case table records it as the intended signal rather than a surprise. |
| The three-code split (DD-2) is more than the card asked for | Low | Reversible; the collapse is mechanical and the plan gate is where to ask for it. |
| `t14` finds the transport shape wrong once it renders | Low | The options travel as verbatim text, which is the shape that constrains a consumer least. A consumer needing a parsed form can parse it; a consumer needing the original text could not recover it from a parsed one. |

## §K Cross-References

- `.moai/reports/anki-card-types-plan-20260920.md` — the design record; § 2 the
  confirmed decisions, § 3 the card split, § 7 the execution log for t11 and
  t12 and the pre-flight findings for t13 through t15.
- `.moai/specs/SPEC-ANKICARD-001/` — the ordering dependency; its `spec.md` § 3
  for the note-type identity this SPEC's predicate reads.
- `spec.md` § 5 — the exclusions that make "renders nothing new" checkable.
- `acceptance.md` — the fourteen criteria and the command that decides each.
