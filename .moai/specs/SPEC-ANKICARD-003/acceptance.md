# Acceptance Criteria — SPEC-ANKICARD-003

Every criterion below is binary-testable and names the command that decides it.
GEARS requirements are **not** restated here: `spec.md` § 3 is the requirement
layer (`REQ-ML-001`..`REQ-ML-015`); this file is the Given-When-Then
verification layer. Each heading names the requirements it verifies.

Fifteen top-level criteria, some carrying sub-lettered sub-criteria (`a`..`j`)
that group sub-assertions within one logical criterion. Sub-criteria are
grouped by subject rather than ordered alphabetically, so a later letter may
appear above an earlier one where it belongs with its neighbours.

Several criteria assert against targets that do not exist yet — the composition
step, the option-taking entry point, the `multiline_answer_missing` code, the
`children-list` rule, and the two editor commands. Every one of them is marked
`[CHANGED]` or `[NEW]` in `spec.md` § 7; a criterion naming one is a criterion
the implementation must create before it can pass, and is not a claim that the
target exists today.

**Reading a marker number in this file.** A criterion that writes `{{c1::…}}`
means that exact number, and the number is part of the assertion — `spec.md`
REQ-ML-005 and REQ-ML-006 are precisely about which numbers appear. Where a
criterion does not care, it writes `{{cN::…}}`.

**The entry's options.** Every criterion below describes an entry whose options
have already passed SPEC-ANKICARD-002's validation gate — recognized values, a
cloze-style note type, no swift beside a multiline option. No criterion here
re-verifies those three rules; `.moai/specs/SPEC-ANKICARD-002/acceptance.md`
owns them.

**Command vocabulary.** Go criteria name a targeted invocation of the form
`go test ./internal/anki/<pkg> -run <TestName> -count=1`; the aggregate is
`make test-go`. Elisp criteria name the ERT file under `tests/`; the aggregate
is `make test-elisp`. `make ci-local` (format check, vet, full suite) is the
gate for the Definition of Done below.

## AC Matrix

### AC-ML-001 — The answer list is the first top-level list of the remaining body (REQ-ML-001)

- **AC-ML-001a — a plain list supplies the answers.**
  **Given** a multiline entry with direction `->` whose title is `Capital of
  Japan` and whose body is `- Tokyo\n- Osaka\n`,
  **When** the entry is rendered,
  **Then** the Text field carries both items, each inside a generated marker,
  and the title outside every marker.
- **AC-ML-001b — a nested sub-item is not an answer item.**
  **Given** a multiline entry with direction `->` whose body is
  `- Tokyo\n  - Kanto\n- Osaka\n`,
  **When** the entry is rendered,
  **Then** the generated markers cover `Tokyo` and `Osaka`,
  **And** `Kanto` appears in the rendered output **outside** every generated
  marker.
- **AC-ML-001c — an ordered list counts.**
  **Given** a multiline entry with direction `->` whose body is `1. A\n2. B\n`,
  **When** the entry is rendered,
  **Then** both items are wrapped, exactly as the unordered form in
  AC-ML-001a is.
- **AC-ML-001d — a description item is wrapped on its definition half only.**
  **Given** a multiline entry with direction `->` whose body is
  `- Tokyo :: the capital\n`,
  **When** the entry is rendered,
  **Then** the generated marker covers `the capital` and not `Tokyo`,
  **And** the rendered output carries no marker split across the term and the
  definition — the shape `{{c1::Tokyo` appearing without its closing `}}` in
  the same element is the specific failure this sub-criterion rejects.
- **AC-ML-001f — a checkbox item is wrapped after its checkbox.**
  **Given** a multiline entry with direction `->` whose body is
  `- [ ] Tokyo\n- [X] Osaka\n`,
  **When** the entry is rendered,
  **Then** each generated marker opens **after** the checkbox, so the rendered
  output carries `{{c1::Tokyo}}` and `{{c1::Osaka}}` inside list items that
  still carry their checked and unchecked classes,
  **And** the rendered output contains no occurrence of the string `::[` — the
  signature of a marker whose opening the checkbox parser consumed, which is
  the specific corruption this sub-criterion rejects.
- **AC-ML-001g — a plain sibling in a description-kind list is wrapped
  whole.**
  **Given** a multiline entry with direction `->` whose body is
  `- Tokyo :: the capital\n- Osaka\n` — one `::` item and one plain sibling,
  so the parser classifies the whole list as description-kind,
  **When** the entry is rendered,
  **Then** the first item's marker covers `the capital` and not `Tokyo`,
  **And** the second item's marker covers the whole of `Osaka`,
  **And** neither marker is split across the term and the definition.
- **AC-ML-001h — an ordered item's counter cookie stays outside the marker.**
  **Given** a multiline entry with direction `->` whose body is
  `1. [@5] Tokyo\n2. Osaka\n`,
  **When** the entry is rendered,
  **Then** the first item renders as a list item carrying the value `5` whose
  content is exactly the generated marker around `Tokyo`,
  **And** the rendered output contains no occurrence of the string `:[@` — the
  signature of a marker whose opening the counter parser consumed.

  The signature differs from AC-ML-001f's `::[` by one character, which is why
  the checkbox criterion does not catch this shape. The cookie is consumed on
  ordered lists only, so the same body written with `-` bullets leaves `[@5]`
  in the text as ordinary content.
- **AC-ML-001i — a mid-item bracket is handled by mirroring the parser, not by
  searching for a leading token.**
  **Given** a multiline entry with direction `->` whose body is
  `- Tokyo [ ] is big\n`,
  **When** the entry is rendered,
  **Then** the generated marker's opening survives — the rendered output
  contains `{{c1::` and does not contain `::Tokyo`.

  This is the criterion that fails a composer built by searching for a leading
  token. The parser's status match is unanchored and removes a fixed count
  from the item's start, so it consumes four characters here with no leading
  token present. The item renders lossily either way, because that truncation
  is a pre-existing go-org behavior this SPEC does not fix; what this criterion
  requires is that composition does not additionally destroy the marker.
- **AC-ML-001j — the offset is a byte offset, on multi-byte content.**
  **Given** a multiline entry with direction `->` whose body is
  `- [ ] 서울특별시\n- [X] 부산\n` — Korean answer items carrying a status,
  **When** the entry is rendered,
  **Then** each generated marker opens immediately after its status marker, so
  the rendered output carries `{{c1::서울특별시}}` and `{{c1::부산}}` intact,
  **And** the rendered output is **valid UTF-8**.

  The UTF-8 assertion is the falsifier a rune-rounded composer fails. The
  parser removes a fixed **byte** count and does not round, so a composer that
  rounded would place its opening at a different offset: measured on the
  pre-existing corruption path, `- 서{{c1::울특별시 [ ] 큼}}` loses the
  marker's opening brace, because the four removed bytes are three of `서`
  plus one of `{`. This project's notes are Korean, so an ASCII-only corpus
  would exercise the single case that does not arise here.
- **AC-ML-001e — a lead paragraph and a later list are not answers.**
  **Given** a multiline entry with direction `->` whose body is
  `Think first.\n\n- Tokyo\n\nAlso worth noting.\n\n- Unrelated\n`,
  **When** the entry is rendered,
  **Then** the only generated marker covers `Tokyo`,
  **And** `Think first.`, `Also worth noting.`, and `Unrelated` all appear
  outside every generated marker.

**Decides**: `go test ./internal/anki/orgdoc -run TestMultilineAnswerList -count=1`.
The same test carries the eighteen corpus shapes `plan.md` § F M1 enumerates
across its sixteen rows as required members, including the four that assert a
wrapped span rather than an item count and the multi-byte row that asserts
valid UTF-8.

### AC-ML-002 — An entry with no answer item is skipped with the new code (REQ-ML-002, REQ-ML-011)

- **AC-ML-002a — rightward with no list.**
  **Given** a multiline entry with direction `->` whose remaining body carries
  no list at all,
  **When** the entry is planned,
  **Then** the entry is reported as skipped with the diagnostic code
  `multiline_answer_missing`, and no note is created or updated for it.
- **AC-ML-002b — leftward fails identically.**
  **Given** the same entry with direction `<-` instead,
  **Then** the outcome is identical — the same code, the same skip. The title
  alone does not rescue it.
- **AC-ML-002c — the skip carries the existing identifier.**
  **Given** such an entry that already carries an `ANKI_NOTE_ID`,
  **When** the entry is planned,
  **Then** the reported skip carries that identifier, and the registry entry is
  unchanged.
- **AC-ML-002d — the gate's codes still win.**
  **Given** an entry whose `direction` value is unrecognized **and** whose body
  carries no list,
  **When** the entry is planned,
  **Then** the single reported diagnostic is `card_option_invalid` — the
  validation gate rejects the entry before composition can observe the missing
  answers.

**Decides**: `go test ./internal/anki/planner -run TestMultilineAnswerMissing -count=1`.

### AC-ML-003 — Direction selects what is wrapped (REQ-ML-003)

- **AC-ML-003a — `->` wraps the answers.**
  **Given** a multiline entry with direction `->`, title `Capital of Japan`,
  body `- Tokyo\n`,
  **When** the entry is rendered,
  **Then** the Text field carries `{{c1::Tokyo}}` and carries `Capital of
  Japan` with no marker around it.
- **AC-ML-003b — `<-` wraps the title.**
  **Given** the same entry with direction `<-` and incremental off,
  **Then** the Text field carries exactly `{{c2::Capital of Japan}}` and
  carries `Tokyo` with no marker around it. The number is part of the
  assertion: the design record § 2.2 fixes the leftward title at `c2`, and
  numbering is a hash input, so leaving it loose would let it drift silently.
- **AC-ML-003c — `<->` wraps both.**
  **Given** the same entry with direction `<->`,
  **Then** the Text field carries both the title and `Tokyo` inside generated
  markers, and the two markers carry **different** numbers.

**Decides**: `go test ./internal/anki/orgdoc -run TestMultilineDirection -count=1`.

### AC-ML-004 — A multiline entry with no arrow composes as `->` (REQ-ML-004)

- **AC-ML-004a — incremental alone.**
  **Given** an entry whose `incremental` is `t` and whose `direction` resolved
  to no value, title `Capital of Japan`, body `- Tokyo\n- Osaka\n`,
  **When** the entry is rendered,
  **Then** the answers are wrapped and the title is not — the `->` shape.
- **AC-ML-004b — an explicitly falsy direction behaves the same.**
  **Given** the same entry with `direction` carrying the falsy spelling
  `nil` rather than no value,
  **Then** the rendered Text field is byte-identical to AC-ML-004a's.

**Decides**: `go test ./internal/anki/orgdoc -run TestMultilineDefaultDirection -count=1`.

### AC-ML-005 — Incremental selects how many numbers the answers carry (REQ-ML-005)

- **AC-ML-005a — off, the answers share one number.**
  **Given** a multiline entry with direction `->`, incremental off, body
  `- Tokyo\n- Osaka\n`,
  **When** the entry is rendered,
  **Then** the Text field carries `{{c1::Tokyo}}` and `{{c1::Osaka}}` — one
  number for both.
- **AC-ML-005b — on, each answer carries its own number in document order.**
  **Given** the same entry with incremental on,
  **Then** the Text field carries `{{c1::Tokyo}}` and `{{c2::Osaka}}`, in that
  order.
- **AC-ML-005c — `<->` with incremental keeps the title distinct.**
  **Given** the same entry with direction `<->` and incremental on,
  **Then** the two answers carry `c1` and `c2`, and the title carries a number
  equal to neither.

**Decides**: `go test ./internal/anki/orgdoc -run TestMultilineIncremental -count=1`.

### AC-ML-006 — Generated numbers do not collide with a hand-written marker (REQ-ML-006)

- **AC-ML-006a — an item carrying a hand-written marker is not wrapped.**
  **Given** a multiline entry with direction `->`, incremental off, whose body
  is `- Tokyo is {{c4::big}}\n- Osaka\n`,
  **When** the entry is rendered,
  **Then** the first item reaches the rendered output byte-for-byte as written
  — `Tokyo is {{c4::big}}`, with its `}}` intact and no generated marker
  around it,
  **And** the second item carries a generated marker numbered `5`,
  **And** the rendered output contains no marker nested inside another.

  The byte-for-byte half of this assertion is what the v0.1.0 criterion wanted
  and could not have: `spec.md` REQ-ML-007.1 would have separated that `}}`
  into `} }` had the item been wrapped. Not wrapping delivers the survival the
  criterion was reaching for.
- **AC-ML-006d — an entry whose every answer is hand-clozed still renders.**
  **Given** a multiline entry with direction `->` whose body is
  `- {{c1::Tokyo}}\n- {{c2::Osaka}}\n`,
  **When** the entry is planned,
  **Then** nothing is wrapped, both items reach the rendered output unchanged,
  and the entry is **not** skipped — neither with `multiline_answer_missing`
  nor with `cloze_marker_missing`.
- **AC-ML-006e — the not-wrap rule governs incremental.**
  **Given** a multiline entry with direction `->`, incremental **on**, whose
  body is `- {{c1::Tokyo}}\n- {{c1::Osaka}}\n` — every answer pre-marked
  under the **same** number,
  **When** the entry is planned,
  **Then** nothing is wrapped, both markers keep the number `1`, and the entry
  is not skipped.

  The shared number is the point of the fixture. Per-answer numbering binds
  only the spans composition wraps, so with none wrapped the entry yields one
  card, and that is the specified outcome rather than an unmet obligation.
  AC-ML-006d's fixture uses distinct numbers and sets no incremental value, so
  it cannot exercise this; the two criteria are not redundant.
- **AC-ML-006b — the ordinary case is the design record's numbering.**
  **Given** a multiline entry with direction `<->`, incremental **off**,
  carrying no hand-written marker anywhere,
  **When** the entry is rendered,
  **Then** the answers carry `c1` and the title carries `c2` — exactly the
  numbers the design record § 2.2 fixes. The incremental-off condition is
  load-bearing: with incremental on the answers consume consecutive numbers
  and AC-ML-005c governs the title instead.
- **AC-ML-006c — a marker in the title counts toward the offset.**
  **Given** a multiline entry with direction `->` whose **title** carries
  `{{c7::…}}` and whose body carries no marker,
  **Then** the generated numbers on the answers begin at `8`,
  **And** the title is unwrapped — both because `->` does not wrap it and
  because it carries a hand-written marker.

**Decides**: `go test ./internal/anki/orgdoc -run TestMultilineNumbering -count=1`.

### AC-ML-007 — A generated marker cannot close early (REQ-ML-007)

- **AC-ML-007a — a brace run inside wrapped content is separated.**
  **Given** a multiline entry with direction `->` whose body is
  `- \\sqrt{a^{2}}\n`,
  **When** the entry is rendered,
  **Then** the Text field contains no `}}` sequence anywhere between the
  generated marker's opening `{{c1::` and its own closing `}}`.
- **AC-ML-007b — content ending in a brace is padded.**
  **Given** a multiline entry with direction `->` whose body is `- f(x}\n`,
  **When** the entry is rendered,
  **Then** the generated marker's closing `}}` is preceded by a space, so the
  boundary does not form a third consecutive brace.
- **AC-ML-007c — the rule matches the editor's.**
  **Given** a fixture file in the repository holding the brace-hazard input
  strings and their required separated forms, read by **both** the Go test and
  the ERT test rather than duplicated as literals in each,
  **When** both suites run,
  **Then** the Elisp helper `spec.md` REQ-ML-007.3 requires — the one applying
  **both** separation and pad — and the Go composer produce the recorded form
  for every input.

  The helper is named rather than `imoogi-anki--cloze-safe-text` because that
  function applies separation only: executed on `f(x}` it returns `f(x}`
  unchanged, since the pad lives in its caller `imoogi-anki-cloze-region`. A
  criterion naming it would assert a form no single existing function
  produces, and the ERT half of the shared fixture would have nothing to call.
  A divergence means an author sees one result when they cloze by hand and
  another when the same text is wrapped by composition; a duplicated literal
  would let the two drift without either suite noticing, which is why the
  fixture is shared rather than copied.

**Decides**: `go test ./internal/anki/orgdoc -run TestMultilineBraceSafety -count=1`
and `tests/anki-commands-test.el` (`make test-elisp`) for the AC-ML-007c pairing.

### AC-ML-008 — Incremental with `<-` is inert (REQ-ML-008)

**Given** a multiline entry with direction `<-` and incremental on, body
`- Tokyo\n- Osaka\n`,
**When** the entry is rendered,
**Then** the rendered Text field is byte-identical to the same entry with
incremental off,
**And** no diagnostic is reported.

**Decides**: `go test ./internal/anki/orgdoc -run TestMultilineLeftwardIncremental -count=1`.

### AC-ML-009 — The pipeline order is split, compose, gate, render (REQ-ML-009)

- **AC-ML-009a — supplementary content never becomes an answer.**
  **Given** a multiline entry with direction `->` whose body is
  `- Tokyo\n\n#+BEGIN_EXTRA\n- not an answer\n#+END_EXTRA\n`,
  **When** the entry is rendered,
  **Then** the generated marker covers `Tokyo` only,
  **And** `not an answer` appears in the `Back Extra` field, carrying no
  generated marker.
- **AC-ML-009b — a generated marker satisfies the marker gate.**
  **Given** a multiline entry with direction `->`, a cloze-style note type, and
  **no** hand-written marker in its title or remaining body,
  **When** the entry is planned,
  **Then** the entry is **not** skipped with `cloze_marker_missing`, and a note
  is created or updated for it.
- **AC-ML-009d — a supplementary block between two answers loses neither.**
  **Given** a multiline entry with direction `->` whose body is
  `- Tokyo\n\n#+BEGIN_EXTRA\nnote\n#+END_EXTRA\n\n- Osaka\n`,
  **When** the entry is rendered,
  **Then** **both** `Tokyo` and `Osaka` carry generated markers — removing the
  block leaves a two-blank-line gap that would otherwise split the authored
  list in two, and the collapse of `spec.md` REQ-ML-009.2 closes it,
  **And** `note` appears in the `Back Extra` field, carrying no marker,
  **And** the rendered output carries exactly one element bearing the class
  `children-list`, containing both answers.
- **AC-ML-009e — the collapse does not merge across genuine content.**
  **Given** a multiline entry with direction `->` whose body is
  `- Tokyo\n\nNote.\n\n- Osaka\n` — a paragraph, not a removed block,
  between the two,
  **When** the entry is rendered,
  **Then** the generated marker covers `Tokyo` only, and `Osaka` renders
  outside every marker, because a paragraph still separates the lists and only
  the first supplies answers.

  The pair is what makes the collapse falsifiable in both directions: a
  collapse that did nothing fails AC-ML-009d, and one that swallowed
  intervening prose fails AC-ML-009e.
- **AC-ML-009f — the collapse does not reach inside a block.**
  **Given** a multiline entry with direction `->` whose body carries a
  `#+BEGIN_SRC` block holding an internal run of two blank lines, followed by
  an answer list,
  **When** the entry is rendered,
  **Then** the block's rendered interior is byte-identical to the same block
  rendered from a body that never passed through the collapse,
  **And** the same holds for `#+BEGIN_EXAMPLE`.

  Measured, an unqualified collapse changes this output, so the criterion is
  not vacuous. An author's code sample reformatted with no diagnostic is a
  silent content change — the failure shape the collapse itself was introduced
  to remove, which is why this sub-criterion sits beside the pair above.
  **And** the `#+BEGIN_EXAMPLE` half is asserted by its own case rather than
  assumed from the `#+BEGIN_SRC` one: the two take different render paths, so
  one passing is not evidence for the other.
- **AC-ML-009h — an unterminated block makes the collapse decline.**
  **Given** a multiline entry whose body opens with a `#+BEGIN_SRC` line that
  is never terminated, followed by two answer items separated by a blank-line
  run,
  **When** the entry is rendered,
  **Then** the two lists remain separate and only the first supplies answers,
  **And** the document still renders — go-org treats the unterminated opening
  as plain text rather than failing.

  The extent of an unterminated block is undefined, so the collapse declines
  rather than guessing where it ends. This criterion pins the decline as the
  specified outcome; without it, an implementer could reasonably read the
  block exclusion as applying to a block that never closes and collapse the
  whole remainder of the body.
- **AC-ML-009g — a deliberately-authored two-list body merges, and that is
  pinned.**
  **Given** a multiline entry with direction `->` whose body is
  `- Tokyo\n- Osaka\n\n\n- Kyoto\n- Nara\n` — two adjacent lists separated
  only by a blank-line run, which is the only way Org lets an author write
  them with no prose between,
  **When** the entry is rendered,
  **Then** all four items carry generated markers and all four sit inside the
  single `children-list` container.

  This is the cost the collapse takes, asserted rather than merely disclosed.
  It is the deliberate trade `spec.md` § 5 records: the merged outcome is
  visible on the card, where the dropped-answer outcome it replaces was
  silent.
- **AC-ML-009c — the gate still fires for everyone else.**
  **Given** an entry declaring a cloze-style note type, carrying **no**
  multiline option on, and carrying no marker,
  **When** the entry is planned,
  **Then** it is skipped with `cloze_marker_missing`, exactly as it is today.

**Decides**: `go test ./internal/anki/planner -run TestMultilineMarkerGate -count=1`.

### AC-ML-010 — Both call sites pass the options, and the old entry point is untouched (REQ-ML-010)

- **AC-ML-010a — the existing entry point's signature is unchanged.**
  **Given** SPEC-ANKICARD-002's `TestRenderSignatureIsUnchanged`,
  **When** the package is compiled,
  **Then** the test compiles and passes without modification. A parameter added
  to the existing entry point makes this fail to compile, which is the
  assertion.
- **AC-ML-010b — the migration path renders the same Text as the sync path.**
  **Given** a stock-`Cloze` multiline entry migrated to its imoogi-owned
  counterpart,
  **When** the migration renders it and the ordinary path then plans the same
  entry,
  **Then** the two Text values are byte-identical, and the ordinary path
  reports the entry as a no-op rather than as an update.
- **AC-ML-010c — a multiline entry on the ordinary path is wrapped.**
  **Given** a multiline entry planned through the ordinary synchronization
  path,
  **Then** its Text field carries the generated markers — the options reached
  the renderer rather than being dropped at the call site.

**Decides**: `go test ./internal/anki/planner -run TestMultilineBothPaths -count=1`
and `go test ./internal/anki/orgdoc -run TestRenderSignatureIsUnchanged -count=1`.

### AC-ML-011 — The answer list renders inside the styled container (REQ-ML-012)

- **AC-ML-011a — the container class is emitted, on every list form.**
  **Given** three multiline entries with direction `->` whose answer lists are
  respectively unordered, ordered, and description-style,
  **When** each is rendered,
  **Then** each Text field carries a single element bearing the class
  `children-list` that contains that entry's answer items — the `<ul>`, the
  `<ol>`, and the `<dl>` alike.
  **And** no entry's Text field carries a second element bearing that class.
- **AC-ML-011d — the container is present under `<-` too.**
  **Given** a multiline entry with direction `<-` and two answer items, under
  which no answer is wrapped,
  **When** the entry is rendered,
  **Then** the Text field still carries the element bearing `children-list`
  containing both items. An implementation attaching the container as part of
  the answer-wrapping step would omit it here, which is the failure this
  sub-criterion exists to catch.
- **AC-ML-011b — the stylesheet carries a rule for it.**
  **Given** `internal/anki/model/assets/base.css`,
  **When** the file is read,
  **Then** it contains a rule whose selector names `children-list`.
- **AC-ML-011c — the stylesheet's existing assertions still hold.**
  **Given** the same file,
  **Then** the existing air-gap and deck-neutrality assertions pass unchanged —
  the new rule names no network-hosted resource and no deck.

**Decides**: `go test ./internal/anki/orgdoc -run TestMultilineContainer -count=1`
and `go test ./internal/anki/model -count=1`.

### AC-ML-012 — The editor commands write the properties correctly (REQ-ML-013)

- **AC-ML-012a — setting a direction writes it.**
  **Given** a heading with no card-option property,
  **When** the direction command is invoked and `<->` is chosen,
  **Then** the heading's own drawer carries `ANKI_DIRECTION: <->`.
- **AC-ML-012b — turning an inherited option off writes the falsy spelling.**
  **Given** a file carrying `#+PROPERTY: ANKI_INCREMENTAL t` and a heading
  carrying no `ANKI_INCREMENTAL` of its own,
  **When** the incremental toggle is invoked to turn it off,
  **Then** the heading's own drawer carries `ANKI_INCREMENTAL` with the falsy
  spelling, and the property is **not** merely deleted.
- **AC-ML-012c — the note-type contract matches the cloze command's.**
  **Given** three headings — one with no `ANKI_NOTE_TYPE`, one with
  `imoogi-Cloze`, one with `imoogi-Basic`,
  **When** the direction command is invoked on each,
  **Then** the first gains `ANKI_NOTE_TYPE: imoogi-Cloze`, the second is
  unchanged, and the third is unchanged and reported to the user.
- **AC-ML-012d — an active swift blocks the write unless confirmed.**
  **Given** a heading resolving `ANKI_SWIFT` to its truthy spelling,
  **When** the direction command is invoked and the user declines to clear
  swift,
  **Then** no property is written at all — neither the direction nor the note
  type.
  **And** when the user confirms, the direction is written **and**
  `ANKI_SWIFT` carries the falsy spelling in the heading's own drawer.
- **AC-ML-012e — the two names join property completion.**
  **Given** the editor's property-name completion candidates,
  **Then** they include `ANKI_DIRECTION` and `ANKI_INCREMENTAL`,
  **And** they do **not** include `ANKI_SWIFT`, which belongs with t15.

**Decides**: `tests/anki-commands-test.el` (`make test-elisp`).

### AC-ML-013 — A non-multiline entry renders byte-identically (REQ-ML-014)

- **AC-ML-013a — the existing corpus is unmodified and passes.**
  **Given** `internal/anki/orgdoc/testdata/render-golden.json` at this SPEC's
  base commit `6f3ae6c`,
  **When** the suite runs at any commit of this SPEC,
  **Then** the file is byte-identical to its base-commit content and its test
  passes — verified by `git diff --exit-code 6f3ae6c -- <path>` returning 0.
- **AC-ML-013b — the multiline corpus is a separate file.**
  **Given** the repository tree,
  **Then** the multiline byte-identity cases live in a golden file distinct
  from `render-golden.json`, with its own recording environment variable.
- **AC-ML-013c — an entry with only falsy options is unchanged.**
  **Given** an entry whose `direction`, `incremental`, and `swift` all carry
  the falsy spelling,
  **When** the entry is rendered,
  **Then** the field map is byte-identical to the same entry rendered with all
  three resolved to no value.
- **AC-ML-013e — the new entry point with no options equals the old one.**
  **Given** every case in the existing byte-identity corpus,
  **When** each is rendered through the **new** entry point with no card
  option set, and through the existing entry point,
  **Then** the two field maps are byte-identical for every case.

  This is the criterion that actually guards production. Once REQ-ML-010 moves
  both call sites, the existing entry point has no production caller, so a
  corpus exercising it alone proves nothing about what users get. The
  delegation REQ-ML-010.1 requires is what makes this hold structurally; this
  criterion checks the delegation is there.
  **And** `TestOptionFreeRequestLogGolden` passes — it drives the real run
  while seeding its fixture hash through the existing entry point, so a
  divergence between the two paths flips every no-op entry to `updated` and
  trips it.
- **AC-ML-013d — the hash input set gained no member.**
  **Given** the content-hash function,
  **When** the package is compiled,
  **Then** it takes no card-option parameter — a build-time assertion in the
  hashing package, not a reviewer reading a signature.

**Decides**: `go test ./internal/anki/orgdoc ./internal/anki/hashing -count=1`,
`go test ./internal/anki/planner -run TestOptionFreeRequestLogGolden -count=1`,
plus the `git diff --exit-code` invocation named in AC-ML-013a.

### AC-ML-014 — The new code is paired with a message (REQ-ML-015)

- **AC-ML-014a — the table carries an entry.**
  **Given** `modules/org/anki/imoogi-error.el`,
  **Then** it carries a message for `multiline_answer_missing` that names the
  problem and states a corrective action, and that contains no stack trace,
  raw transport error, backtrace, or bare exit code.
- **AC-ML-014b — the two-directional pairing still holds.**
  **Given** the existing pairing assertion and its hard-coded back-end code
  list,
  **When** the suite runs,
  **Then** every back-end code has a table entry and every table entry not on
  the documented front-end-only allowlist has a back-end constant — with the
  enlarged code set.

**Decides**: `tests/anki-error-test.el` (`make test-elisp`) and
`go test ./internal/anki/protocol -count=1`. That file, not
`tests/anki-sync-error-test.el`, is where the two-directional pairing
assertion and the hard-coded back-end code list live; SPEC-ANKICARD-002 names
it correctly for the same criterion.

### AC-ML-015 — The measured baseline does not regress (all requirements)

**Given** the baseline measured at `6f3ae6c` — `go test ./... -count=1` exit 0;
coverage `orgdoc` 100.0%, `planner` 92.5%, `hashing` 100.0%, `model` 98.4%;
`make lint` 0 findings; `make fmt-check` 0 findings; `make test-elisp` exit 0
reporting `Ran 422 tests, 420 results as expected, 0 unexpected, 2 skipped`,
**When** the same commands run at this SPEC's final commit,
**Then** `go test ./... -count=1` exits 0; `make lint` and `make fmt-check`
report 0 findings; `make test-elisp` exits 0 with 0 unexpected results;
**And** per-package coverage for `orgdoc`, `planner`, `hashing`, and `model` is
no lower than the figure above for that package.

**Decides**: `make ci-local`, plus `go test -cover ./internal/anki/...` for the
per-package figures.

## Edge Cases

Each row states the input shape and the required outcome. All are covered by
the criteria above; the table exists so a reader can check the list is complete
rather than infer it.

| Shape | Required outcome | Criterion |
|---|---|---|
| Nested sub-item under an answer item | Visible, outside every generated marker | AC-ML-001b |
| Checkbox item (`- [ ] A`, `- [X] B`) | Wrapped after the checkbox; no `::[` in the output | AC-ML-001f |
| Plain sibling inside a description-kind list | Wrapped whole; it has no definition half | AC-ML-001g |
| Ordered item with a counter cookie (`1. [@5] A`) | Wrapped after the cookie; no `:[@` in the output | AC-ML-001h |
| Mid-item bracket (`- Tokyo [ ] is big`) | Marker opening survives; the parser's own truncation is pre-existing | AC-ML-001i |
| Multi-byte answer items with a status | Marker intact, output valid UTF-8; the offset is in bytes | AC-ML-001j |
| Ordered item whose list is description-kind (`1. [@5] T :: D`) | Cookie not consumed; the gate is list kind, not terminator | AC-ML-001h (corpus, `plan.md` DD-3) |
| Ordered list as the answer list | Wrapped like an unordered one | AC-ML-001c |
| Description list (`term :: definition`) | Definition wrapped, term visible, marker not split | AC-ML-001d |
| Lead paragraph before the list | Question context, unwrapped | AC-ML-001e |
| Second list later in the body | Ordinary content, unwrapped | AC-ML-001e |
| No list at all | `multiline_answer_missing`, every direction | AC-ML-002a, AC-ML-002b |
| Malformed option value **and** no list | `card_option_invalid` alone | AC-ML-002d |
| Hand-written marker inside an answer item | That item is not wrapped; it survives byte-for-byte | AC-ML-006a |
| Every answer item hand-clozed | Nothing wrapped, entry still renders, no diagnostic | AC-ML-006d |
| Every answer hand-clozed under one number, incremental on | Not wrapped; one card. The not-wrap rule governs | AC-ML-006e |
| Hand-written marker in the title | Counts toward the offset; title unwrapped | AC-ML-006c |
| Answer item carrying LaTeX braces | No early close inside the generated marker | AC-ML-007a |
| Answer item ending in `}` | Space before the closing `}}` | AC-ML-007b |
| `<-` with incremental on | Inert, no diagnostic | AC-ML-008 |
| `#+BEGIN_EXTRA` list in the body | Extra field, never an answer | AC-ML-009a |
| `#+BEGIN_EXTRA` block *between* two answers | Both answers survive; the gap is collapsed | AC-ML-009d |
| Paragraph between two lists | Still two lists; only the first supplies answers | AC-ML-009e |
| Block with an internal blank-line run in the question context | Interior unchanged; the collapse skips blocks | AC-ML-009f |
| Unterminated `#+BEGIN_` block | Collapse declines; lists stay split; document still renders | AC-ML-009h |
| Stray `#+END_EXTRA` with no opener, or left by a terminated pair | Survives the split, renders as visible literal text, and splits the answer list as a paragraph would. An orphan END opens no block interior, so the collapse never declines on it | AC-ML-009e (same mechanism — genuine content between lists) |
| Two adjacent author-written lists, blank-line separated | Merged into one answer list; pinned as a deliberate trade | AC-ML-009g |
| Bullet line inside `#+BEGIN_SRC` / `#+BEGIN_EXAMPLE` | Not an answer item; never wrapped | AC-ML-001a (corpus, `plan.md` § F M1) |
| `<-` direction with answers | Container still emitted though nothing is wrapped | AC-ML-011d |
| Description-style answer list | Container still carries `children-list` | AC-ML-011a |
| Multiline entry with no hand-written marker | Renders; not skipped | AC-ML-009b |
| Non-multiline entry with no marker | Skipped as today | AC-ML-009c |
| All three options falsy | Byte-identical to no options | AC-ML-013c |
| Stock-`Cloze` multiline entry on the migration path | Same Text as the sync path; no spurious update | AC-ML-010b |

## Quality Gate Criteria

| Gate | Threshold | Command |
|---|---|---|
| Go suite | exit 0 | `go test ./... -count=1` |
| Coverage, `internal/anki/orgdoc` | ≥ 100.0% | `go test -cover ./internal/anki/orgdoc` |
| Coverage, `internal/anki/planner` | ≥ 92.5% | `go test -cover ./internal/anki/planner` |
| Coverage, `internal/anki/hashing` | ≥ 100.0% | `go test -cover ./internal/anki/hashing` |
| Coverage, `internal/anki/model` | ≥ 98.4% | `go test -cover ./internal/anki/model` |
| Lint | 0 findings | `make lint` |
| Format | 0 findings | `make fmt-check` |
| Elisp suite | exit 0, 0 unexpected | `make test-elisp` |
| Existing byte-identity corpus | unmodified | `git diff --exit-code 6f3ae6c -- internal/anki/orgdoc/testdata/render-golden.json` |

## Definition of Done

1. Every criterion `AC-ML-001`..`AC-ML-015` passes, each by the command its
   section names.
2. `make ci-local` exits 0.
3. The existing byte-identity corpus and its signature assertion are unmodified
   — the `git diff --exit-code` invocation in AC-ML-013a returns 0.
4. Every requirement `REQ-ML-001`..`REQ-ML-015` is named by at least one
   passing criterion, verified by reading the criterion headings rather than
   asserted.
5. `README.md`'s Anki section documents the two properties, how the answers are
   read from the body, and what the four direction-and-incremental combinations
   produce.
6. No file outside `spec.md` § 7's delta table is modified.
