# Acceptance Criteria — SPEC-ANKICARD-002

Every criterion below is binary-testable and names the command that decides it.
GEARS requirements are **not** restated here: `spec.md` § 3 is the requirement
layer (`REQ-OPT-001`..`REQ-OPT-013`); this file is the Given-When-Then
verification layer. Each heading names the requirements it verifies.

Fourteen top-level criteria, some carrying sub-lettered sub-criteria (`a`..`d`)
that group sub-assertions within one logical criterion.

Several criteria assert against targets that do not exist yet — the three new
protocol fields, the version-2 constant, the three diagnostic codes, the
cloze-style predicate, and the validation gate. Every one of them is marked
`[CHANGED]` or `[NEW]` in `spec.md` § 7; a criterion naming one is a criterion
the implementation must create before it can pass, and is not a claim that the
target exists today.

**Reading `nil` in this file.** The word carries two meanings in this SPEC's
subject matter and they are never interchangeable here. Where a property
resolved to nothing, this file writes **"no value"** (Elisp side) or **`null`**
(wire side) and never `nil`. Where the user wrote the falsy spelling, it
appears as a value — `swift: nil`, a `nil` cell in a value column, or quoted
`"nil"` on the wire — which is a present, non-null string meaning the option is
off. `plan.md` DD-1 carries the same distinction for the protocol shape. The
inheritance opt-out is exactly where confusing the two costs the most.

**Command vocabulary.** Go criteria name a targeted invocation of the form
`go test ./internal/anki/<pkg> -run <TestName> -count=1`; the aggregate is
`make test-go`. Elisp criteria name the ERT file under `tests/`; the aggregate
is `make test-elisp`. `make ci-local` (format check, vet, full suite) is the
gate for the Definition of Done below.

## AC Matrix

### AC-OPT-001 — The three properties resolve through the nearest-wins chain (REQ-OPT-001)

- **AC-OPT-001a — own drawer wins.**
  **Given** an Org buffer whose heading carries `ANKI_DIRECTION: <-` in its own
  drawer while an ancestor carries `ANKI_DIRECTION: ->`,
  **When** the heading's card options are resolved,
  **Then** the resolved direction is `<-`.
- **AC-OPT-001b — nearest ancestor wins over a farther one and over the file.**
  **Given** a three-level heading tree whose grandparent carries
  `ANKI_SWIFT: nil`, whose parent carries `ANKI_SWIFT: t`, and whose file
  carries `#+PROPERTY: ANKI_SWIFT nil`, with the target heading's own drawer
  carrying none,
  **When** the target heading's card options are resolved,
  **Then** the resolved swift value is `t`.
- **AC-OPT-001c — the file-level keyword is the last level.**
  **Given** a heading with no ancestor carrying `ANKI_INCREMENTAL`, in a file
  carrying `#+PROPERTY: ANKI_INCREMENTAL t`,
  **When** the heading's card options are resolved,
  **Then** the resolved incremental value is `t`.
- **AC-OPT-001d — a present-but-empty value terminates the chain.**
  **Given** a file carrying `#+PROPERTY: ANKI_SWIFT t` and a heading whose own
  drawer carries `:ANKI_SWIFT:` with an empty value,
  **When** the heading's card options are resolved,
  **Then** the resolved swift value is **no value** — not `t`, and not the
  empty string treated as a value.
  **And** the same holds for `ANKI_DIRECTION` and `ANKI_INCREMENTAL`.
- **AC-OPT-001e — the accumulating `PROPERTY+` form is normalized to plain
  replacement.**
  **Given** a heading whose own drawer carries `:ANKI_DIRECTION: ->` followed
  by `:ANKI_DIRECTION+: <-`,
  **When** the heading's card options are resolved,
  **Then** the resolved direction is exactly `<-` — the last matching drawer
  line as a plain override — and is **never** the appended form `-> <-`.

  This is the criterion that discriminates reuse from reimplementation, and it
  is why REQ-OPT-001.1 is falsifiable rather than merely stated. Sub-criteria
  a through d describe a nearest-wins chain that any competent
  reimplementation would also satisfy; `PROPERTY+` normalization is the one
  rule the existing chain deliberately does **not** inherit from Org's own
  property accessor, which appends a same-level `PROPERTY+` value even with
  inheritance off. A second implementation built on that accessor fails here
  and passes everything else.
  **And** the same holds for `ANKI_INCREMENTAL+` and `ANKI_SWIFT+`.

**Decides**: `tests/anki-props-test.el` (`make test-elisp`).

### AC-OPT-002 — The front end interprets nothing and writes nothing (REQ-OPT-002)

- **AC-OPT-002a — a malformed value is carried verbatim.**
  **Given** a heading whose drawer carries `ANKI_DIRECTION: -->` and
  `ANKI_INCREMENTAL: yes` — neither a recognized value,
  **When** the heading is resolved and serialized,
  **Then** the serialized document carries the strings `-->` and `yes`
  unchanged; the front end raises no error, drops no entry, and substitutes no
  default.
- **AC-OPT-002b — no case-folding and no value mapping.**
  **Given** headings carrying `ANKI_INCREMENTAL: T`, `ANKI_SWIFT: NIL`, and
  `ANKI_DIRECTION: <->`,
  **When** the values are serialized,
  **Then** the wire carries `T`, `NIL`, and `<->` with their case intact; the
  front end maps no spelling onto another and substitutes no canonical form.
  The case-insensitivity that decides recognition is the back end's, per
  REQ-OPT-009.3.
- **AC-OPT-002c — no card-option property is written back.**
  **Given** an Org buffer whose headings carry a mix of card-option values and
  none,
  **When** a scan and a sync run complete,
  **Then** no card-option property appears in any drawer that did not already
  carry it, and no existing card-option value is modified.

  Asserted by reading the three properties' drawer values before and after and
  comparing those, **not** by comparing whole buffer contents. A sync run does
  modify the buffer: its write-back writes `ANKI_NOTE_ID` into each
  newly-added heading's drawer. A whole-buffer comparison would therefore fail
  for a reason this criterion is not about. Equivalently, diff the buffer and
  assert the delta touches only `ANKI_NOTE_ID`.

  The migration write-back writes `ANKI_NOTE_TYPE` alongside `ANKI_NOTE_ID`
  and is a different path; this criterion pins the ordinary sync path, whose
  write-back touches `ANKI_NOTE_ID` alone.
- **AC-OPT-002d — `ANKI_NOTE_TYPE` still does not inherit.**
  **Given** a parent heading carrying `ANKI_NOTE_TYPE: imoogi-Cloze` and a
  child carrying card options but no `ANKI_NOTE_TYPE` of its own,
  **When** the file is scanned,
  **Then** the child produces **no** sync-target entry.

**Decides**: `tests/anki-props-test.el`, `tests/anki-scan-test.el` (`make test-elisp`).

### AC-OPT-003 — The scan attaches the resolved options to the entry (REQ-OPT-003)

**Given** a file whose heading carries `ANKI_NOTE_TYPE: imoogi-Cloze` and
`ANKI_DIRECTION: <->`, under a file-level `#+PROPERTY: ANKI_SWIFT t`,
**When** the sync root is scanned,
**Then** the produced entry carries a direction of `<->`, a swift of `t`, and
**no value** for incremental,
**And** a heading carrying no card-option property at any level produces an
entry whose three option values are all **no value**.

**Decides**: `tests/anki-scan-test.el` (`make test-elisp`).

### AC-OPT-004 — The wire carries three new keys, always present (REQ-OPT-004)

- **AC-OPT-004a — the keys are always emitted.**
  **Given** an entry whose three card-option values are all unresolved,
  **When** the request document is serialized,
  **Then** the entry object contains the keys `direction`, `incremental`, and
  `swift`, each with the JSON value `null`. A missing key fails.
- **AC-OPT-004b — a resolved value is a JSON string.**
  **Given** an entry resolving `incremental` to `t` and `swift` to `nil`,
  **When** the request document is serialized,
  **Then** both keys carry JSON **strings** — `"t"` and `"nil"` — not JSON
  booleans and not JSON null. The `"nil"` case is the one a boolean-typed wire
  would silently collapse.
- **AC-OPT-004c — the back end round-trips the shape.**
  **Given** a request document carrying all three keys, one null and two
  populated,
  **When** it is decoded into the entry shape and re-encoded,
  **Then** the three fields survive with their null-ness intact.
- **AC-OPT-004d — the response gains nothing.**
  **Given** the response document shape,
  **When** it is encoded for a run of any outcome,
  **Then** its key set is identical to the version-1 key set.

**Decides**: `go test ./internal/anki/protocol -count=1`;
`tests/anki-process-test.el` (`make test-elisp`).

### AC-OPT-005 — The declared version is 2 on both sides and every pin agrees (REQ-OPT-005)

**Given** the wire-contract version constant in `internal/anki/protocol` and
the declared version in `modules/org/anki/imoogi-process.el`,
**When** both are read,
**Then** both are `2`,
**And** every file that pins a protocol version for the Anki subsystem asserts
`2` rather than `1`. The pinned set is enumerated by this grep, filtered to
the Anki subsystem by path:

```
grep -rln 'protocol_version\|:protocol-version\|ProtocolVersion\|imoogi-protocol-version\|protocol\.Version' .
```

The alternation carries the hyphen form `:protocol-version` and the command
applies **no** `--include` filter, deliberately. A pin written only in the
hyphen keyword form, or living in a JSON or testdata fixture rather than a
`.go` or `.el` file, would be invisible to a narrower search — and this grep
is what Definition of Done item 4 re-runs, so a search narrower than the claim
it decides would let a future pin hide from the verification step. No pin is
missed today; the breadth guards the next one.

The set is exactly:

| File | Pin |
|---|---|
| `internal/anki/protocol/protocol.go` | the constant itself |
| `internal/anki/protocol/protocol_test.go` | constant assertion |
| `internal/anki/protocol/install_test.go` | install-document assertion |
| `cmd/imoogi-anki/main.go` | response emission + skew probe |
| `cmd/imoogi-anki/main_test.go` | sync fixtures |
| `cmd/imoogi-anki/migrate_test.go` | migrate fixtures |
| `cmd/imoogi-anki/install_test.go` | install fixtures |
| `cmd/imoogi-anki/log_test.go` | log fixtures |
| `modules/org/anki/imoogi-process.el` | the declared version |
| `tests/anki-process-test.el` | asserts a parsed response's version |
| `tests/anki-install-test.el` | install-document assertion |
| `tests/anki-migrate-test.el` | migrate fixtures |
| `tests/anki-sync-oneway-test.el` | sync fixtures |
| `tests/anki-sync-error-test.el` | error-path fixtures |

The unfiltered grep also matches three groups that are all **out of scope**,
listed so a reader of its output can dismiss them without re-deciding:

| Group | Files | Why out of scope |
|---|---|---|
| Clipboard subsystem | `internal/clipboard/**`, `modules/org/28-clipboard.el` | Its own unrelated protocol version. Includes two `testdata/*.json` fixtures — the exact shape a `--include='*.go'` filter would have hidden, which is why the filter is gone. |
| Org-preview subsystem | `internal/orgpreview/**`, `modules/org/23-org-preview.el`, `tests/org-preview-*` | Its own unrelated protocol version. One hit is a vendored minified JavaScript bundle. |
| Documentation and state | `.moai/**` — this SPEC, SPEC-ANKICARD-001, the audit reports, the backlog | Prose about the version, not a pin of it. |

A hit outside the fourteen-row table and these three groups is a pin the table
missed, and is resolved before the Definition of Done is claimed.

**Decides**: `make test-go` and `make test-elisp`, both of which fail on any
un-updated pin.

### AC-OPT-006 — Version skew is reported in both directions at every subcommand (REQ-OPT-006)

- **AC-OPT-006a — an older front end against this binary.**
  **Given** a request document declaring `protocol_version: 1`,
  **When** it is fed to each of the `sync`, `migrate`, and `install-models`
  subcommands,
  **Then** each writes a well-formed response whose `ok` is false and whose
  errors carry exactly the code `binary_incompatible`, and each exits non-zero.
- **AC-OPT-006b — this front end against an older binary.**
  **Given** a stub binary whose declared version is 1 and a front end declaring
  2,
  **When** a sync run is attempted,
  **Then** the front end renders the `binary_incompatible` message from its own
  table, and no AnkiConnect request is issued.
- **AC-OPT-006c — no second mechanism.**
  **Given** the diff of this SPEC,
  **When** the version-skew code path is inspected,
  **Then** the number of distinct version-comparison sites in
  `cmd/imoogi-anki` is unchanged at one shared probe, and no new
  version-negotiation field appears on either document.

**Decides**: `go test ./cmd/imoogi-anki -count=1`;
`tests/anki-sync-error-test.el` (`make test-elisp`).

### AC-OPT-007 — The cloze-style predicate answers for exactly two names (REQ-OPT-007)

**Given** the back end's cloze-style predicate,
**When** it is applied to each declared note-type value below,
**Then** it answers exactly:

| Declared `ANKI_NOTE_TYPE` | Cloze-style |
|---|---|
| `Cloze` | true |
| `imoogi-Cloze` | true |
| `Basic` | false |
| `imoogi-Basic` | false |
| `imoogi-Other` | false |
| `MyCloze` | false |
| `cloze` | false |
| `` (empty) | false |

**And** the predicate is derived from the existing declared-type-to-renderer
mapping rather than from a second literal list — asserted by a test that fails
if the predicate answers true for a name the mapping does not collapse onto the
cloze renderer.
**And** the front end's existing cloze predicate answers identically for the
same eight values.

**Decides**: `go test ./internal/anki/planner -count=1`;
`tests/anki-commands-test.el` (`make test-elisp`).

### AC-OPT-008 — A card option on a non-cloze note type is rejected (REQ-OPT-011)

**Given** an entry declaring `ANKI_NOTE_TYPE: imoogi-Basic` and carrying
`ANKI_SWIFT: t`,
**When** the run processes it,
**Then** the result action for that entry is `skipped`, carrying the entry's
existing note identifier where it has one,
**And** exactly one error is reported for it, with code
`card_option_needs_cloze`,
**And** the request log records no `addNote`, `updateNoteFields`, `deleteNotes`,
or deck request for that entry,
**And** the registry record for that entry, if any, is unchanged.
**And** the same holds for `ANKI_DIRECTION` and for `ANKI_INCREMENTAL` on a
non-cloze type, and for a declared type of `Basic`.

**And** an explicitly falsy option on a non-cloze type is **accepted**: an
entry declaring `imoogi-Basic` and carrying `swift: nil` — and nothing else —
synchronizes exactly as it would with all three fields null, with no error.
This is the case that makes the inheritance opt-out usable: a basic heading
under a file-level `#+PROPERTY: ANKI_SWIFT t` can write `ANKI_SWIFT: nil` in
its own drawer without thereby being told to change its note type.

**Decides**: `go test ./internal/anki/planner -count=1`.

### AC-OPT-009 — An unrecognized card-option value is rejected (REQ-OPT-009)

**Given** an entry declaring `ANKI_NOTE_TYPE: imoogi-Cloze` with a cloze marker
in its body, carrying the single card-option value in each row below,
**When** the run processes it,
**Then** the outcome is exactly:

| Property | Value on the wire | Outcome |
|---|---|---|
| `direction` | `->` | accepted |
| `direction` | `<-` | accepted |
| `direction` | `<->` | accepted |
| `direction` | `  ->  ` | accepted (trimmed) |
| `direction` | `nil` | accepted, option off |
| `direction` | `NIL` | accepted, option off (case-insensitive) |
| `direction` | `-->` | `card_option_invalid` |
| `direction` | `<=>` | `card_option_invalid` |
| `direction` | `` (empty string on the wire) | `card_option_invalid` |
| `incremental` | `t` | accepted, option on |
| `incremental` | `T` | accepted, option on (case-insensitive) |
| `incremental` | `nil` | accepted, option off |
| `incremental` | `yes` | `card_option_invalid` |
| `incremental` | `1` | `card_option_invalid` |
| `swift` | `t` | accepted, option on |
| `swift` | ` nil ` | accepted, option off (trimmed) |
| `swift` | `true` | `card_option_invalid` |

**And** a rejected entry is reported as `skipped` with exactly one error,
**And** that error's machine-oriented detail contains both the property name and
the offending value.

Note the empty-string row: the front end resolves a present-but-empty property
to **null** (AC-OPT-001d), so an empty string can only reach the back end from a
hand-built or third-party request. Rejecting it keeps the back end's recognized
set closed rather than trusting the front end to have normalized.

**Decides**: `go test ./internal/anki/planner -count=1`.

### AC-OPT-010 — Swift together with a multiline option is rejected (REQ-OPT-010)

**Given** an entry declaring a cloze-style note type with a cloze marker in its
body, carrying the option combination in each row below,
**When** the run processes it,
**Then** the outcome is exactly:

| `swift` | `direction` | `incremental` | Outcome |
|---|---|---|---|
| `t` | `->` | null | `card_option_conflict` |
| `t` | null | `t` | `card_option_conflict` |
| `t` | `<->` | `t` | `card_option_conflict` |
| `t` | null | `nil` | accepted (falsy is not option-bearing) |
| `t` | `nil` | null | accepted (falsy direction is not option-bearing) |
| `nil` | `->` | `t` | accepted |
| `t` | null | null | accepted |
| null | `->` | `t` | accepted |

**And** a rejected entry is reported as `skipped` with exactly one error.
**And** an inherited conflict is suppressible: given a file-level
`#+PROPERTY: ANKI_SWIFT t` and a heading carrying `ANKI_DIRECTION: ->` plus
`:ANKI_SWIFT:` with an empty value, the entry reaches the back end with a null
swift and is accepted.

**Decides**: `go test ./internal/anki/planner -count=1` for the table;
`tests/anki-scan-test.el` for the suppression case.

### AC-OPT-011 — One diagnostic per entry, in the fixed order, before rendering (REQ-OPT-008)

- **AC-OPT-011a — the order is fixed.**
  **Given** an entry declaring `ANKI_NOTE_TYPE: imoogi-Basic`, carrying
  `swift: t` and `direction: bogus` — offending against all three rules at once,
  **When** the run processes it,
  **Then** exactly **one** error is reported for it, with code
  `card_option_invalid`.
  **And** with the direction corrected to `->` (leaving the conflict and the
  non-cloze type), exactly one error is reported, with code
  `card_option_conflict`.
  **And** with the swift option removed (leaving only the non-cloze type),
  exactly one error is reported, with code `card_option_needs_cloze`.
- **AC-OPT-011b — the gate precedes rendering.**
  The renderer fails on exactly two conditions, and each gives one usable
  probe. Both must hold.
  **Given** an entry declaring an unrecognized note type such as `Widget` and
  carrying `swift: t` — a type the renderer rejects outright,
  **When** the run processes it,
  **Then** the reported code is `card_option_needs_cloze`, not
  `org_parse_error`.
  **And given** an entry declaring `imoogi-Cloze`, carrying
  `direction: bogus`, whose title and body contain no cloze marker,
  **When** the run processes it,
  **Then** the reported code is `card_option_invalid`, not
  `cloze_marker_missing`.
  Each pairing is evidence the gate ran before the render, because the
  suppressed code is one only the renderer can raise.
- **AC-OPT-011c — the migration path shares the gate.**
  **Given** a migration run over a candidate entry carrying `swift: t` and
  `direction: ->`,
  **When** the migration processes it,
  **Then** it is reported as `skipped` with code `card_option_conflict`; the
  original note, its registry record, and the Org heading are all unchanged,
  and no `addNote` or `deleteNotes` request is recorded for it.
  **And** the two arms that return before the per-entry pipeline raise no
  option diagnostic, each pinned as inherited behaviour rather than left
  unnoticed:

  | Arm | Given | Then |
  |---|---|---|
  | Dry run | the same conflicting entry, under `migrate --dry-run` | reported as `migrate_candidate`; no option diagnostic. The pre-existing count gap of `plan.md` § B item 4. |
  | Non-candidate | an entry the run rejects as a non-candidate — its registry-recorded type is not stock, or its declared type is neither that recorded type nor its counterpart — carrying `swift: t` and `direction: ->` | whatever the existing non-candidate handling already reports for it, unchanged; no option diagnostic. |

  Neither arm loses coverage: an entry not migrated is still processed by the
  ordinary synchronization path, where the gate does run and the conflict is
  reported. These rows pin that the gate was not silently expected to cover
  code it never reaches, per REQ-OPT-008.5.

  The bypass is structural, not a placement choice: both arms return before
  the per-entry pipeline, so no position for the gate inside that pipeline can
  reach them.
- **AC-OPT-011d — the gate reads the declared type on both paths.**
  **Given** a migration candidate whose declared type is `Cloze` and whose
  counterpart is `imoogi-Cloze`, carrying `direction: ->`,
  **When** the migration processes it,
  **Then** it is **not** rejected — the declared and counterpart types agree on
  cloze-style-ness, which is the property REQ-OPT-008.4 relies on.

**Decides**: `go test ./internal/anki/planner -count=1`.

### AC-OPT-012 — The renderer and the hash are byte-identical (REQ-OPT-012.1, REQ-OPT-012.2)

- **AC-OPT-012a — the renderer's signature and output are unchanged.**
  **Given** the renderer entry point after this SPEC,
  **When** its signature is compared against the tree at the SPEC's base commit,
  **Then** it takes the same parameters and returns the same types — it gained
  no card-option parameter,
  **And** for a golden corpus of note type, title, and body triples covering
  basic, cloze, cloze-with-extra-block, math, and empty-body cases, the
  produced field map is byte-identical to the golden recorded at the base
  commit. The base commit is the post-t12 tree, not an older one: t12 already
  added the supplementary field to the cloze output, and that change is part of
  the baseline rather than a regression this criterion should catch.
- **AC-OPT-012b — the hash input set is unchanged.**
  **Given** the content-hash function after this SPEC,
  **When** a compiled call site invokes it with exactly the four existing
  arguments — note type, rendered fields, resolved deck, sorted tags,
  **Then** the package builds. A parameter added for a card option makes this
  call fail to compile, which is the assertion: it is a build-time check, not
  a reviewer reading a signature.
  **And** for one fixed entry processed twice through the planner — once with
  all three option fields null, once with `direction: ->` and
  `incremental: t` — the recorded content hash is **equal** both times.

  The second half must live in a planner test, not a hashing test: the hash
  function takes no card-option parameter, so it cannot be handed the two
  variants to compare. Only the planner sees both an entry's options and the
  hash computed for it.

**Decides (AC-OPT-012b specifically)**: `go test ./internal/anki/planner -count=1`.
- **AC-OPT-012c — an option-free entry issues no new request.**
  **Given** a synchronization run over entries none of which carries a card
  option, against a registry recording each entry's current hash,
  **When** the run completes,
  **Then** every result action is `skipped` carrying the entry's existing note
  identifier — which is what the unchanged-hash branch has always returned,
  there being no distinct `no-op` action value — the errors slice is **empty**,
  and the request log is byte-identical to the log the same fixture produced
  before this SPEC.
  The empty errors slice is the load-bearing half: a rejected entry is also
  reported as `skipped`, so the action alone cannot distinguish an untouched
  entry from a rejected one.

**Decides**: `go test ./internal/anki/orgdoc -count=1`,
`go test ./internal/anki/hashing -count=1`,
`go test ./internal/anki/planner -count=1`.

### AC-OPT-013 — The marker-missing baseline is pinned for t14 (REQ-OPT-012.3)

**Given** an entry declaring `ANKI_NOTE_TYPE: imoogi-Cloze`, carrying
`ANKI_DIRECTION: ->` — a valid option on a valid type — whose title and body
contain no `{{cN::` marker,
**When** the run processes it,
**Then** it passes the option gate and is then reported as `skipped` with the
existing code `cloze_marker_missing`,
**And** no note is created for it.

This is the behavior backlog card **t14** will change: a multiline card
generates its own markers from the direction, so this entry will become valid
there. Pinning it here is what makes t14's diff reviewable — the test is
expected to be **updated** by t14, not to keep passing forever.

**Decides**: `go test ./internal/anki/planner -count=1`.

### AC-OPT-014 — Every new code is paired with a user-facing message (REQ-OPT-013)

- **AC-OPT-014a — the hardcoded list is updated.**
  **Given** the test-side list of back-end-emitted codes in
  `tests/anki-error-test.el`,
  **When** `imoogi-error-test-codes-match-source-file` runs,
  **Then** it passes — meaning the list gained exactly
  `card_option_needs_cloze`, `card_option_invalid`, and `card_option_conflict`,
  and matches the constants the back end declares.
- **AC-OPT-014b — every back-end code has a message.**
  **Given** the enlarged code set,
  **When** `imoogi-error-test-go-codes-are-subset-of-table` runs,
  **Then** it passes.
- **AC-OPT-014c — the reverse pairing still holds.**
  **Given** the enlarged message table,
  **When** `imoogi-error-test-table-entries-are-all-paired-with-a-go-constant`
  runs,
  **Then** it passes, and the front-end-only allowlist is **unchanged** at its
  three existing entries — none of the three new codes is added to it.
- **AC-OPT-014d — the messages stay clean and actionable.**
  **Given** the three new messages,
  **When** `imoogi-error-test-messages-contain-no-raw-diagnostics` runs,
  **Then** it passes,
  **And** each message names the property class at issue and states a
  corrective action: change the note type or remove the option; fix the value
  and what the recognized values are; remove one of the two conflicting
  options.

**Decides**: `tests/anki-error-test.el` (`make test-elisp`).

## Edge Cases

| Case | Expected |
|---|---|
| All three options null | Entry behaves exactly as before this SPEC (AC-OPT-012c). |
| `incremental: nil` alone on a cloze entry | Accepted, option off, not option-bearing (AC-OPT-010). |
| `swift: nil` alone on a **basic** entry | Accepted. A falsy option is off, so the entry is not option-bearing and the note-type rule does not fire (AC-OPT-008). |
| `direction` present with `incremental: nil` | Accepted; the direction is option-bearing on its own. |
| `direction: nil` on a cloze entry | Accepted, option off. The falsy spelling is recognized on all three properties, so the explicit opt-out is uniform. |
| Option-bearing entry, cloze type, no cloze marker | `cloze_marker_missing` (AC-OPT-013). |
| Option-bearing entry on a note type absent from Anki | Reaches the existing field-resolution path unchanged; this SPEC adds no earlier failure for it. |
| Entry with a duplicated note identifier plus a bad option | The existing duplicate-identifier handling is unchanged; this SPEC adds no interaction with it. |
| Empty string arriving on the wire for any option | `card_option_invalid` (AC-OPT-009) — unreachable from this front end, closed against others. |
| A file-level option over a file containing no cloze entries | Every sync-target entry in the file is rejected with `card_option_needs_cloze`, which is the intended signal that a file-level option was written too broadly. |

## Quality Gates

| Gate | Threshold |
|---|---|
| `make fmt-check` | exit 0 |
| `make lint` | exit 0 (`go vet`) |
| `make test-go` | exit 0 |
| `make test-elisp` | exit 0, zero unexpected results |
| `go build ./...` | exit 0 |
| `GOOS=windows go build ./cmd/imoogi-anki` | exit 0 |
| `internal/anki/protocol` coverage | not below its pre-SPEC figure |
| `internal/anki/planner` coverage | not below its pre-SPEC figure (measured at 91.6% after t12) |
| `internal/anki/orgdoc` coverage | 100.0% maintained |

Coverage floors are stated as "not below the measured pre-SPEC figure" rather
than as absolute numbers, because the pre-SPEC figure must be measured on the
run-phase base commit — a number copied from the design record's t12 entry is a
carry-over, not a baseline.

## Definition of Done

1. All fourteen acceptance criteria pass, each by the command it names.
2. `make ci-local` exits 0.
3. Every new diagnostic code is declared in the back-end const block, paired in
   the front-end message table, and present in the test-side code list —
   verified by `tests/anki-error-test.el` rather than by inspection.
4. Every protocol-version pin in the AC-OPT-005 table reads `2`, verified by
   re-running that criterion's grep **as written there** — hyphen form in the
   alternation, no `--include` filter — and reading each hit. A hit outside the
   table is either a new pin to update or a fifth unrelated subsystem to name;
   it is never ignored.
5. The golden render corpus of AC-OPT-012a is recorded **before** any
   production change, so byte-identity is measured against the real baseline.
6. `README.md`'s existing Anki section documents the three properties, their
   recognized values, their inheritance, and the cloze-style note-type
   requirement — written in that section's current language, which this SPEC
   does not change.
7. No file outside `spec.md` § 7's surface list is modified.
