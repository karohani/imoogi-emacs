# Acceptance Criteria — SPEC-ANKICARD-001

Every criterion below is binary-testable — a Go table test, a fake-client
request-log assertion, an ERT assertion, a direct file inspection, or a named
manual smoke check against a real Anki collection. GEARS requirements are **not**
restated here: `spec.md` § 3 is the requirement layer (`REQ-C-001`..`REQ-C-024`);
this file is the Given-When-Then verification layer. Each heading names the
requirements it verifies.

Twenty-four top-level criteria, some carrying sub-lettered sub-criteria
lettered `a`..`d` (`AC-C-0NNa` .. `AC-C-0NNd`) that group sub-assertions within
one logical criterion.

Several criteria assert against targets that do not exist yet — the
`modelNames` / `createModel` / `updateModelStyling` / `updateModelTemplates` /
`storeMediaFile` connector methods and their fake-client call logs, the
`install-models` and `migrate` subcommands and the install request document, the
`migrate_candidate` action value, and the `internal/anki/media` and
`internal/anki/model` packages. Every one of them is marked `[NEW]` in
`spec.md` § 7; a criterion naming one is a criterion the implementation must
create before it can pass, and is not a claim that the target exists today.
Individual `[NEW]` markers below flag the cases where an **existing** artifact
gains a new assertion, which is the case a reader is most likely to misread.

## AC Matrix

### AC-C-001 — Install step creates absent imoogi-owned types (REQ-C-001, REQ-C-002)

**Given** a collection whose `modelNames` response contains neither
`imoogi-Basic` nor `imoogi-Cloze`,
**When** the install step runs,
**Then** the request log records exactly one `createModel` per imoogi-owned type
and zero `updateModelStyling` / `updateModelTemplates` requests; the
`imoogi-Basic` request carries the field list `["Front", "Back"]`, the
`imoogi-Cloze` request carries `["Text", "Back Extra"]` and `isCloze: true`; and
both carry card templates and a non-empty `css` value.

### AC-C-002 — Install step updates present types and is idempotent (REQ-C-002)

**Given** a collection whose `modelNames` response already contains both
imoogi-owned types,
**When** the install step runs,
**Then** the request log records one `updateModelStyling` and one
`updateModelTemplates` per type and **exactly zero** `createModel` requests;
**And** a second consecutive invocation produces a request log byte-identical to
the first (idempotence).

### AC-C-003 — The AC-016 replacement pair (REQ-C-003, REQ-C-004; replaces parent AC-016)

- **AC-C-003a — sync hot path, blanket in force.**
  **Given** a synchronization run of any shape — add, update, deck move, delete,
  or no-op,
  **When** the run completes,
  **Then** the counts recorded on the three model-write call logs —
  `createModel`, `updateModelStyling`, `updateModelTemplates` — are each
  **exactly 0**. (These three are the endpoints parent AC-016 names that the
  `AnkiConnector` interface actually carries after this SPEC's five additions;
  the interface exposes no scheduling, review-history, or collection-styling
  endpoint at all, so a count assertion over those would assert nothing. Their
  continued absence from the interface is itself asserted by `AC-C-022b`.)
- **AC-C-003b — install path, ownership-scoped.**
  **Given** an invocation of the install step against a collection also holding
  the stock `Basic` and `Cloze` types and at least one user-authored type,
  **When** the step completes,
  **Then** every recorded `createModel` / `updateModelStyling` /
  `updateModelTemplates` request names a model whose name begins with `imoogi-`,
  and the count of such requests naming any other model is **exactly 0**.

### AC-C-004 — Deck-class normalization (REQ-C-006)

**Given** the normalizer and a deck path,
**When** the path is normalized,
**Then** the produced token matches this table exactly:

| `{{Deck}}` value | normalized token | emitted class |
|---|---|---|
| `(PROGRAMMER)::(GO)` | `programmer-go` | `deck-programmer-go` |
| `Geography::Europe` | `geography-europe` | `deck-geography-europe` |
| `A::::B` | `a-b` | `deck-a-b` |
| `2026 Review` | `_2026-review` | `deck-_2026-review` |
| `  Spaced  Name  ` | `spaced-name` | `deck-spaced-name` |
| `Math_Notes` | `math_notes` | `deck-math_notes` |
| `::` | `unnamed` | `deck-unnamed` |
| `---` | `unnamed` | `deck-unnamed` |
| `!!!` | `unnamed` | `deck-unnamed` |

**And** the wrapper carrying that class appears only in the card template — a
grep of every stored field value produced by the renderer for `class="deck-`
returns zero matches.

### AC-C-005 — Base stylesheet properties (REQ-C-007)

**Given** the embedded base stylesheet,
**When** it is inspected as text,
**Then** it contains both `.card.nightMode` and `.card.night_mode` (the latter
with no space between the class names), a `.card img` rule setting
`max-width: 100%` and `max-height: none`, and `word-wrap: break-word`;
**And** it declares all seven CSS custom properties of REQ-C-007.2 with exactly
the literal values that table fixes — `--imoogi-bg: #f7f3e9` and
`--imoogi-fg: #1c1a17` and `--imoogi-emphasis-fg: #0b0a09` on `.card`, and
`--imoogi-bg: #1a1a1a` and `--imoogi-fg: #c8c4bc` and
`--imoogi-emphasis-fg: #f5f2ec` under **both** night-mode selector forms, with
`--imoogi-measure: 44rem`, `--imoogi-line-height: 1.6`, and the two literal
`--imoogi-serif` / `--imoogi-sans` font stacks declared once each;
**And** each is consumed exactly where REQ-C-007.2 says: `.card` sets
`background: var(--imoogi-bg)`, `color: var(--imoogi-fg)`,
`font-family: var(--imoogi-sans)`, and `line-height: var(--imoogi-line-height)`;
the `h1`-`h6` rule sets `font-family: var(--imoogi-serif)`; the deck-wrapper rule
sets `max-width: var(--imoogi-measure)`; and the math-and-code rule sets
`color: var(--imoogi-emphasis-fg)`. Every one of these is a substring assertion
over the embedded asset;
**And** it contains **zero** occurrences of `http`, `@import`, `url(`, or any
other network-fetch construct (the air-gap clause, mechanically checkable).

### AC-C-006 — User stylesheet concatenation and no per-deck derivation (REQ-C-008, REQ-C-009)

- **AC-C-006a.** **Given** a user stylesheet whose contents are a known marker
  string, **When** the install step runs, **Then** the `css` value in every
  model-write request equals the base stylesheet followed verbatim by that
  marker string, in that order.
- **AC-C-006b.** **Given** no user stylesheet at the configured path, **When**
  the install step runs, **Then** the `css` value equals the base stylesheet
  alone, and no diagnostic is reported.
- **AC-C-006c.** **Given** two runs against collections whose decks differ
  entirely, **When** each install step runs, **Then** the base-stylesheet portion
  of the uploaded CSS is byte-identical between the two runs — no colour, font,
  or other property varies with any deck name or deck-name hash.
- **AC-C-006d — the Elisp → Go transport `[NEW]` assertion.** **Given** a user
  stylesheet at the `defcustom` path whose contents are a known marker string,
  **When** `imoogi-anki-setup` invokes `install-models`, **Then** the JSON
  document the front end writes to that subcommand's stdin is exactly
  `{protocol_version, ankiconnect_url, user_css}`, its `user_css` value equals
  the file's contents byte-for-byte, and the sync `Request` document is
  unchanged and gains no field;
  **And** with no file at that path the same document carries `user_css: ""`;
  **And** a grep of `internal/anki/model` and `internal/anki/ankiconnect` for
  filesystem reads of any `.css` path returns zero matches — the Go binary reads
  no stylesheet from disk.

### AC-C-007 — Styling replacement is wholesale and announced (REQ-C-002)

**Given** an imoogi-owned type whose CSS was hand-edited inside Anki,
**When** the install step updates it,
**Then** the `updateModelStyling` request carries the full base-plus-user
concatenation rather than a merge or a patch, so the hand edit is discarded;
**And** the command emits, before the write, a message stating that imoogi owns
these types' styling outright and that hand edits will be replaced.

### AC-C-008 — MathJax delimiter conversion over all four Org syntaxes (REQ-C-010)

**Given** rendered field content carrying an Org math fragment,
**When** the MathJax transform runs,
**Then** the output is exactly:

| input fragment | output fragment |
|---|---|
| `$E = mc^2$` | `\(E = mc^2\)` |
| `$$E = mc^2$$` | `\[E = mc^2\]` |
| `\(E = mc^2\)` | `\(E = mc^2\)` (unchanged) |
| `\[E = mc^2\]` | `\[E = mc^2\]` (unchanged) |

**And** no other delimiter form appears anywhere in the output.

### AC-C-009 — MathJax edge cases (REQ-C-010, REQ-C-011)

- **AC-C-009a — the `$5` non-conversion.** **Given** the body text
  `costs $5 and $7 total`, **When** the transform runs, **Then** the output
  contains `$5` and `$7` unchanged and contains **no** `\(` or `\)` — Org's
  inline-math heuristic rejects the candidate on two independent grounds: the
  enclosed text is not attached to the closing `$` (it ends in whitespace), and
  that closing `$` is followed by `7` rather than by whitespace or non-dash
  punctuation. Neither ground is "the character is a digit"; the rule is applied
  as written, regardless of what the underlying parser identified as a fragment.
- **AC-C-009b — newline becomes `<br>`.** **Given** a converted fragment whose
  interior contains a line break, **When** the transform runs, **Then** the
  emitted fragment carries `<br>` at that position and contains no bare newline.
- **AC-C-009c — protected regions and no network reference.** **Given** content
  carrying a `$` inside a `<pre>` region, inside a `<code>` region, inside an
  HTML attribute value, and inside a link target, **When** the transform runs,
  **Then** all four occurrences are byte-unchanged;
  **And** the transform's output contains no reference to any network-hosted
  MathJax or stylesheet resource.

### AC-C-010 — Local image resolution and confinement (REQ-C-012)

- **AC-C-010a.** **Given** an entry whose Org file lives in a subdirectory of the
  sync root and whose body references `[[file:diagram.png]]` present beside that
  Org file, **When** the media pass runs, **Then** the reference resolves against
  the entry's **own** Org-file directory — not the sync root, and not the process
  working directory — and the file is found.
- **AC-C-010b.** **Given** a reference resolving outside the sync root (for
  example `../../etc/passwd`), **When** the media pass runs, **Then** the
  reference is rejected by path confinement, the entry is reported skipped with a
  diagnostic code, and no upload request is issued.

### AC-C-011 — Content-hash-suffixed stored names (REQ-C-012)

- **AC-C-011a.** **Given** a resolvable local image, **When** the media pass
  runs, **Then** the `storeMediaFile` request's `filename` is the sanitized
  basename plus the content-hash suffix **inserted before the extension**
  (`diagram.png` → `diagram-<12 hex>.png`, per `design.md` § 9.3), the rewritten
  `src` attribute equals that exact filename, and the filename was computed
  before any response was received (assertable by running the pass against a
  client that returns an error and observing the same rewritten `src`).
- **AC-C-011b — cross-directory collision.** **Given** two entries in two
  different directories each referencing a file named `diagram.png`, whose
  contents differ, **When** the run completes, **Then** the two stored filenames
  differ, both are uploaded, and neither `src` points at the other's file.
- **AC-C-011c — identical content and identical basename.** **Given** two entries
  referencing two files whose contents are identical **and** whose basenames are
  identical, **When** the run completes, **Then** both resolve to the same stored
  filename and exactly one `storeMediaFile` request is issued for it;
  **And** given two files with identical content but **different** basenames, the
  two stored filenames differ — the stored name being sanitized basename +
  content-hash suffix (REQ-C-012.3), so content alone does not determine it.

### AC-C-012 — Two-part links and the extension boundary (REQ-C-012, REQ-C-024)

- **AC-C-012a.** **Given** `[[file:diagram.png][My diagram]]`, which the renderer
  emits as an anchor, **When** the media pass runs, **Then** the output carries
  an `img` element whose `src` is the stored filename and whose `alt` is
  `My diagram`, and no anchor for that target remains.
- **AC-C-012b — outside the set.** **Given** `[[file:x.avif][d]]`, **When** the
  media pass runs, **Then** the anchor is unchanged, no upload is issued, and no
  diagnostic is reported — `.avif` lies outside go-org's image set.
- **AC-C-012c — video and audio.** **Given** a link whose target is `clip.webm`
  or `sound.mp3`, **When** the run completes, **Then** the rendered output passes
  through exactly as produced, no `storeMediaFile` request names it, and no
  diagnostic is reported.

### AC-C-013 — Remote references untouched (REQ-C-013)

**Given** field content referencing `https://example.com/img.png` and
`http://example.com/img.png`,
**When** the media pass runs,
**Then** both `src` values are byte-unchanged, no `storeMediaFile` request names
either, and no diagnostic is reported.

### AC-C-014 — Media requests are gated and deduplicated (REQ-C-014)

- **AC-C-014a — the AC-015 analogue.** **Given** a collection already in sync and
  an unchanged source tree containing image-bearing entries, **When** a second
  run executes, **Then** the `storeMediaFile` call log records **exactly 0**
  entries, and so do the note-mutating call logs.
- **AC-C-014b — per-run dedupe.** **Given** three entries in one run referencing
  the same image file, **When** the run completes, **Then** the `storeMediaFile`
  call log records exactly one request for that stored filename.

### AC-C-015 — Unresolvable media is a contained skip (REQ-C-015, REQ-C-013)

- **AC-C-015a.** **Given** an entry referencing a local image that does not
  exist, **When** the run executes, **Then** that entry is reported as skipped
  with a diagnostic code naming the reference; no `addNote` or
  `updateNoteFields` request is issued for it; its existing registry hash is
  byte-unchanged; and every other sync target in the run is still processed.
- **AC-C-015b — no underscore prefix.** **Given** any successful upload, **When**
  the request log is inspected, **Then** no `storeMediaFile` `filename` begins
  with `_`;
  **And** no request in any run enumerates, globs, or deletes a media file
  imoogi did not upload during that run.

### AC-C-016 — Transform-before-hash, the load-bearing guard (REQ-C-016, REQ-C-017)

- **AC-C-016a — stored bytes equal hashed bytes.** **Given** an entry carrying
  both math and a local image, **When** the run adds or updates it, **Then** the
  field map recorded in the fake client's add/update call log is **byte-equal**
  to the field map whose hash the registry records;
  **And** recomputing the content hash from the fake client's stored field
  values, the recorded model name, the resolved deck, and the sorted tag set
  reproduces the registry hash **exactly** — the same recomputation the
  orphan-deletion ownership predicate performs. This assertion, not prose
  sequencing, is the guard.
- **AC-C-016b — no new hash inputs.** **Given** the hashing function's exported
  signature, **When** it is exercised across the parameter space, **Then** the
  input set is exactly the four the parent SPEC fixes — note type, rendered field values,
  resolved deck, sorted tag set — with no image bytes, no file path, and no
  media list among them;
  **And** editing a referenced image's content changes the entry's hash solely
  because the content-derived stored filename in the field text changed (an image
  edit produces an update; a byte-identical image produces a no-op).
- **AC-C-016c — the ownership predicate survives a media edit.** **Given** an
  entry synchronized with an image, **When** the image is edited, the entry is
  re-synchronized, and the orphan-deletion ownership predicate is then recomputed
  from a `notesInfo` response, **Then** the predicate matches and the entry is
  recognized as imoogi-owned. (Anki's byte-identical round-trip of stored field
  HTML is a **measurement** — see `research.md` § 14 R1 — not an assumption.)

### AC-C-017 — First run after the rendering change (REQ-C-016)

- **AC-C-017a.** **Given** a collection of previously synchronized imoogi notes,
  **When** the first run after this SPEC's rendering change executes, **Then**
  every note whose rendered field content differs under the new transforms is
  updated in that single pass, and the run reports success — this is specified
  behavior, not an error condition.
- **AC-C-017b.** **Given** a previously synchronized note carrying neither math
  nor a local image, **When** that same run executes, **Then** its rendered
  content is byte-identical, its hash is unchanged, and **no** request is issued
  for it.

### AC-C-018 — The migrate subcommand's shape and its dry-run gate (REQ-C-018, REQ-C-019)

- **AC-C-018a — wire stability.** **Given** the migrate subcommand, **When** its
  request and response documents are compared against the `sync` subcommand's,
  **Then** the request shape and the response schema match, no existing wire
  document has gained a field, and the protocol version constant is unchanged on
  both the Go and the Elisp side.
- **AC-C-018b — dry run writes nothing, and reports through `results[]`.**
  **Given** a registry recording N entries on stock note types, **When**
  `migrate` runs in dry-run mode, **Then** every AnkiConnect **write** call log
  records exactly 0 requests — the Handshake `requestPermission` read the
  existing client performs is permitted and is not a write, so the assertion is
  scoped to writes rather than to all calls;
  **And** the response carries exactly N `results[]` entries whose `action` is
  `migrate_candidate`, so the candidate count is `len(results)` and no response
  field was added;
  **And** `protocol.Version` is unchanged and `Result`'s field set is still
  exactly `{key, action, note_id}`.
- **AC-C-018c — confirmation gate.** **Given** a non-zero candidate count,
  **When** the setup flow presents the count and the user declines, **Then** no
  writing `migrate` request is issued at all, no note is added or deleted, and
  the registry is byte-unchanged;
  **And** the confirmation text names both the candidate count and the loss of
  review history and scheduling state.

### AC-C-019 — Successful migration (REQ-C-020, REQ-C-021)

- **AC-C-019a — add before delete.** **Given** a confirmed migration of an entry
  whose registry-recorded type is stock `Basic`, **When** it executes, **Then**
  the request log shows the `addNote` under `imoogi-Basic` preceding the
  `deleteNotes` of the original, and the delete is issued only after the add
  succeeded;
  **And** the added note's fields are rendered under the counterpart type, not
  copied from the original.
- **AC-C-019b — three-way write-back.** **Given** the same migration, **When** it
  completes, **Then** the registry entry names the new identifier and the
  imoogi-owned note type, the heading's `ANKI_NOTE_ID` property carries the new
  identifier, and the heading's `ANKI_NOTE_TYPE` property carries the
  imoogi-owned name — all three consistent.
- **AC-C-019c — declared-type residue.** **Given** a confirmed migration whose
  entry declares a note type that is neither its recorded stock type nor that
  type's counterpart, **When** migration runs, **Then** that entry is skipped
  with `note_type_change_unsupported` and no note is added or deleted for it.

### AC-C-020 — Migration failure and the parent REQ-021 complement (REQ-C-021, REQ-C-022)

- **AC-C-020a — failed add is a no-op on the original.** **Given** a confirmed
  migration whose `addNote` fails, **When** the run continues, **Then** the
  original note still exists, its registry entry is byte-unchanged, its heading
  properties are unchanged, the entry is reported skipped with a diagnostic code,
  and the remaining entries are still processed;
  **And** no state exists in which the registry names an identifier the
  collection no longer holds.
- **AC-C-020b — the sync-path complement.** **Given** an entry whose registry
  record says `Basic` and whose heading was hand-edited to declare
  `imoogi-Basic`, **When** an ordinary synchronization run executes, **Then** the
  entry is skipped with `note_type_change_unsupported`, no field is written, and
  the run continues — parent REQ-021's behavior, unchanged;
  **And** the Elisp diagnostic table's corrective string for
  `note_type_change_unsupported` contains the literal `imoogi-anki-setup`.

### AC-C-021 — Diagnostic and literal-site contracts (REQ-C-023, REQ-C-005)

- **AC-C-021a — the contract test as it exists today.** **Given** the Go protocol
  package and the Elisp diagnostic table, **When** the existing
  `tests/anki-error-test.el` runs, **Then** the two assertions that test actually
  makes both hold for the four new codes: every `protocol.Code*` constant appears
  as a key in the Elisp table (Go ⊆ Elisp), and the Go constant set equals the
  test's own hardcoded list. A Go-side addition without its Elisp entry
  therefore fails.
- **AC-C-021c — the reverse direction `[NEW]`.** **Given** the same two sides,
  **When** an entry is added to the Elisp table with no matching `protocol.Code*`
  constant, **Then** the contract test fails. This assertion does **not** exist
  today — `tests/anki-error-test.el` checks only Go ⊆ Elisp, so an Elisp-only
  entry currently fails nothing. It is added by this SPEC to make the pairing of
  REQ-C-023 bidirectional.
- **AC-C-021b.** **Given** the sites that hard-code stock note-type literals,
  **When** each is inspected, **Then** the `ANKI_NOTE_TYPE_ALL` allowlist offers
  both imoogi-owned names, the note-type-writing commands write them, the cloze
  auto-mark writes `imoogi-Cloze` and its comparison accepts either `Cloze` form,
  and the corrective prose names the imoogi-owned type.

### AC-C-022 — Backward compatibility and the collection guarantee (REQ-C-005, REQ-C-024)

- **AC-C-022a — hand-written stock headings still sync.** **Given** a heading
  whose `ANKI_NOTE_TYPE` names stock `Basic` (and one naming stock `Cloze`),
  **When** an ordinary run executes, **Then** each synchronizes exactly as before
  this SPEC — added, updated, or skipped identically — and neither carries the
  imoogi theme.
- **AC-C-022b — no foreign model is ever written.** **Given** a full run of every
  code path this SPEC touches — install, sync, migrate dry-run, and migrate —
  **When** the complete request log is inspected, **Then** **no** request on the
  three model-**write** endpoints (`createModel`, `updateModelStyling`,
  `updateModelTemplates`) names a model whose name does not begin with
  `imoogi-`; in particular none names `Basic` or `Cloze`. The scope is the write
  endpoints, mirroring `AC-C-003b`: the read endpoints are deliberately excluded
  because `modelNames` returns every foreign name by construction and
  `modelFieldNames("Basic")` is a **required** call on the stock-sync path
  (`AC-C-022a`, REQ-C-005.2);
  **And** the `AnkiConnector` interface exposes no scheduling, review-history, or
  collection-styling method at all — asserted by enumerating its method set.
- **AC-C-022c — manual collection check.** **Given** the user's real collection
  holding 88 stock-`Basic` and 27 stock-`Cloze` notes, and a confirmed migration
  whose dry-run reported N stock-`Basic` candidates and M stock-`Cloze`
  candidates (N and M are read from that dry-run's `results[]` and recorded
  before the writing run), **When** install, sync, and that confirmed migration
  have all run, **Then** the stock `Basic` and `Cloze` note types, their card
  templates, and their CSS are byte-unchanged;
  **And** the stock-`Basic` note count is exactly `88 − N` and the stock-`Cloze`
  note count is exactly `27 − M`.
- **AC-C-022d — no media cleanup.** **Given** any run, **When** the request log
  is inspected, **Then** no request enumerates or deletes collection media, and
  no report names an unreferenced media file.

## Edge Cases

### AC-C-023 — Cloze brace and MathJax delimiter collision (REQ-C-010)

**Given** an `imoogi-Cloze` entry whose body carries `{{c1::$x^{2}$}}`,
**When** the render-plus-transform pipeline runs,
**Then** the cloze marker `{{c1::` survives byte-for-byte, the math becomes
`\(x^{2}\)`, the closing braces are balanced, and Anki's cloze logic is not
disturbed by a non-standard delimiter (no delimiter other than `\(`/`\)` and
`\[`/`\]` appears).

### AC-C-024 — Sub/superscript inside a math fragment (REQ-C-010; carried as a regression test)

**Given** the fragment `\(\sum_{i=1}^n a_n\)`,
**When** the render-plus-transform pipeline runs,
**Then** the output contains no `<sub>` or `<sup>` element inside the fragment
and the TeX interior reaches Anki intact. Carried as a **regression test rather
than an assertion of upstream behavior**: `research.md` C2 records that no lens
verified the upstream fix version, and the pinned renderer version's source
reading is the only evidence.

## Definition of Done

- [ ] `AC-C-001` through `AC-C-023`, including every sub-lettered sub-criterion,
      pass. `AC-C-023` (the cloze-brace / MathJax-delimiter collision) is in the
      pass set rather than the observe-and-record set: it asserts imoogi's own
      transform output, which imoogi controls, so it is a normal assertion.
- [ ] `AC-C-024` alone is exercised and its observed behavior recorded rather
      than asserted — it is the one criterion whose subject is **upstream**
      renderer behavior at a pinned version (`research.md` C2: no lens verified
      the upstream fix version). It is carried as a regression test, so a
      pinned-version upgrade that changes the behavior turns it red by design.
- [ ] `AC-C-016a` (the transform-before-hash byte-equality guard) exists as a
      test asserting byte equality and hash reproduction, not as prose ordering
      in any document. This is the SPEC's single load-bearing correctness
      assertion.
- [ ] `AC-C-003a`'s zero-model-writes assertion was written **first** (milestone
      M1, before any model code exists) and stayed green for the remainder of the
      SPEC.
- [ ] `make build-anki` succeeds; `make test` (`test-elisp`, `test-go`,
      `test-shell`) is green end to end; `make lint` (`go vet`) is clean; and
      `make fmt-check` is clean — `ci-local` gates on it, so a green `make test`
      alone is not sufficient evidence.
- [ ] Package coverage meets or exceeds each package's **currently measured**
      figure, so no `[MODIFY]` package regresses. Measured by
      `go test -count=1 -cover ./internal/...` in this repository at `HEAD f6ee148` (2026-09-05):
      `orgdoc` ≥ 100.0, `hashing` ≥ 100.0, `planner` ≥ 90.3, `registry` ≥ 87.8,
      `ankiconnect` ≥ 85.0 (measured 84.3 in this repository at HEAD f6ee148 — the floor is set at the run-phase norm, so M1's five new client methods must lift it). The two `[NEW]` packages,
      `internal/anki/media` and `internal/anki/model`, carry no inherited floor
      and are held to the 85.0 project norm, with table-test coverage of the
      normalizer, the CSS concatenation, the math transform, and the media
      rewrite.
- [ ] The Go-const-to-Elisp-table contract test passes with every new diagnostic
      code present on both sides.
- [ ] Manual smoke test on the real collection: both night modes render, math
      displays, images display, the deck class appears on the wrapper, and the
      stock `Basic` 88 / `Cloze` 27 notes are untouched — confirmed on desktop
      and on one mobile client.
- [ ] Every `[MODIFY]` and `[EXISTING]` module in `spec.md` § 7 carries
      characterization coverage before modification, per the brownfield delta
      processing order.
- [ ] No document in this SPEC directory carries an unresolved clarification
      marker (`plan.md` revision 2 closed all ten; `plan.md` §H records
      dispositions, not markers).

## Quality Gate Criteria (TRUST 5)

- **Tested** — Go side: table tests for both pure transforms, the deck-class
  normalizer, and the CSS concatenation; fake-client request-log assertions for
  every wire-visible behavior including the negative ones (each new connector
  method arrives with its own call log so a "zero requests" assertion is
  possible); the byte-equality ordering guard; the CLI E2E stub extended with
  `install-models` and `migrate`. Elisp side: flat `tests/*.el` ERT plus the
  cross-language diagnostic contract test. The parent SPEC's achieved coverage
  figures are the floor.
- **Readable** — the `imoogi-` prefix is the ownership predicate and the
  namespace convention already used throughout the project. Test names carry
  their rationale, following the existing convention. The base stylesheet ships
  as an embedded asset rather than a Go string constant precisely so it stays
  readable and lintable.
- **Unified** — `internal/anki/media` and `internal/anki/model` follow the
  `internal/anki/*` subpackage-per-concern convention. New Elisp lives in the
  existing `modules/anki/*.el` modules with their established contract. New CLI
  cases sit beside `sync` in the existing dispatch.
- **Secured** — no network access is added on any path: the stylesheet and
  templates are embedded, MathJax is Anki's bundled copy, and the air-gap clause
  of REQ-C-007 / REQ-C-011 is mechanically checked by `AC-C-005`. Media
  resolution is confined to the sync root by an existing, reused confinement
  routine (`AC-C-010b`). No credential handling and no new user-input parsing
  beyond the existing wire schema is introduced. The destructive surface —
  deletion during migration — is gated behind an explicit confirmation
  (`AC-C-018c`) and ordered add-before-delete so no window loses data
  (`AC-C-019a`).
- **Trackable** — commits follow Conventional Commits scoped to
  `SPEC-ANKICARD-001`. This SPEC amends a closed parent SPEC; the amendment is
  recorded in `spec.md` HISTORY (`amendment_of: SPEC-ANKI-001`) and in § 4, and
  every parent identifier it touches (REQ-016, AC-016, § 4, D-12, REQ-021) is
  named explicitly rather than implied.
