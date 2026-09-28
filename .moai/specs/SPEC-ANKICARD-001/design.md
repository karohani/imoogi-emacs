# Design — SPEC-ANKICARD-001

Technical design for the requirement set in `spec.md` § 3. Design *decisions*
(the alternatives considered and rejected) live in `plan.md` § D as
`D-C-1`..`D-C-15` and are referenced here by identifier rather than restated.
Evidence lives in `research.md`.

## 1. Component view

```
                 ┌──────────────────────── Emacs front end ────────────────────────┐
                 │                                                                 │
  user ──C-c a b─┤ 24-anki.el          imoogi.el            imoogi-setup.el        │
                 │  note-type marks     defcustoms            setup + install tail  │
                 │  ANKI_NOTE_TYPE_ALL  user-CSS path         scan → confirm        │
                 │        │                   │                     │               │
                 │        └───────────────────┴──────────┬──────────┘               │
                 │                     imoogi-process.el │ (stdin JSON, stdout JSON)│
                 │                     imoogi-error.el   │  imoogi-writeback.el     │
                 └───────────────────────────────────────┼──────────────────────────┘
                                                         │
                            cmd/imoogi-anki  run(args[0]) switch
                          ┌──────────────┬────────────────┴───────────────┐
                          │              │                                │
                       sync           install-models                   migrate
                          │              │                          (--dry-run)
                          │              │                                │
      ┌───────────────────┴──────┐       │                                │
      │  internal/anki/planner   │◄──────┼────────────────────────────────┘
      │  render → transform →    │       │
      │  hash → decide → dispatch│       │
      └───┬────────┬────────┬────┘       │
          │        │        │            │
   orgdoc │  media │ hashing│ registry   │  model  (NEW)
  (render │ (NEW:  │ (EXIST:│ (EXISTING) │  embedded templates + base CSS
   + math │ resolve│ 4 inputs           │  deck-class normalizer
transform)│ /hash/ │ unchanged)         │  base+user concatenation
          │ rewrite)                    │  probe-then-act install
          └────────┴──────────┬─────────┴──────────┬─────────┘
                              │                    │
                    internal/anki/ankiconnect  internal/anki/protocol
                    (+5 methods, 1 chokepoint)  (+N codes, wire unchanged)
                              │
                          AnkiConnect ──► Anki collection
```

Two rules read off the diagram:

1. **`orgdoc` never touches the filesystem or the network.** The renderer's
   purity contract is inherited unchanged from the parent SPEC's D-1. The math
   transform is a pure post-render string function that lives beside the existing
   cloze-marker guard; the media pass is a separate package that receives the
   entry's base directory as a parameter (`D-C-11`).
2. **Every AnkiConnect request funnels through the client's single `call`
   chokepoint.** The five new methods add no new transport, no new error tier,
   and no new retry policy.

## 2. The pipeline and its ordering invariant

### 2.1 The pipeline

For one sync target, in order:

```
  1. render            orgdoc.Render(noteType, title, body) → fields map[string]string
  2. transform (math)  MathJax delimiter conversion over each field value
  3. transform (media) resolve → confine → content-hash → rewrite src / anchor→img
                        ⇒ (rewrittenFields, []Upload)
  ────────────────────────────────────────────────────────────────────────────────
  4. hash              hash(noteType, rewrittenFields, resolvedDeck, sortedTags)
  5. decide            compare against registry hash → add | update | no-op | skip
  6. dispatch          storeMediaFile(uploads…) then addNote / updateNoteFields
                        — with the SAME rewrittenFields that step 4 hashed
  7. record            registry entry ← {noteID, sourcePath, noteType,
                                          resolvedDeck, hash from step 4}
```

### 2.2 The invariant

> **INVARIANT (transform-before-hash).** Every content transform this SPEC
> introduces completes before step 4, and the field map dispatched at step 6 is
> the *same value* the hash at step 4 was computed over — byte-for-byte, not
> merely equal in intent.

Why it is load-bearing rather than tidy (`D-C-9`, `research.md` § 6): the content
hash has **two** uses. The first is the add/update/no-op decision at step 5. The
second is the **ownership predicate** on the orphan-deletion path, which
*recomputes* the hash from a `notesInfo` response and treats a mismatch as
`delete_candidate_unowned`. If a transform ran after hashing — or only on the
outbound request — every imoogi note would fail that recomputed predicate
permanently, orphan deletion would silently stop, and the only symptom would be a
stream of `delete_candidate_unowned` reports. The parent SPEC's own run-phase
report already classifies this failure mode as fail-safe-but-silent.

The recomputation's other precondition — that Anki returns stored field HTML
unchanged — is **measured**, not assumed (`research.md` § 14 R1). That measurement
is what leaves the ordering invariant as the only thing standing between the
implementation and a silent deletion stop.

**The invariant is encoded as a test, not as this prose.** `research.md` § 10
records the parent SPEC's recurring audit defect: "design-narrative ordering
mistaken for a behavioral guard." The guard is `AC-C-016a` — a byte-equality
assertion plus a hash recomputation — and this section is documentation of it,
not a substitute for it.

### 2.3 Successor to the parent's design § 2.4 — the hash input table

The hash input set is **unchanged** by this SPEC.

| # | Input | Status under this SPEC |
|---|---|---|
| 1 | Note type (model name) | Unchanged. Now may be `imoogi-Basic` / `imoogi-Cloze`. |
| 2 | Rendered field values, post-render | Unchanged as an *input*; their *content* now carries converted math delimiters and rewritten `src` attributes. |
| 3 | Effective (resolved) deck | Unchanged. |
| 4 | Sorted tag set | Unchanged. |
| — | Source path, heading position, `ANKI_NOTE_ID` | Excluded, as before. |
| — | **Image bytes** | **Excluded — deliberately, and this is the design point.** |

**Why no fifth input** (`D-C-10`): the hashing function's own contract is that
every input must round-trip through a `notesInfo` response, so the recomputed
ownership predicate can reproduce it. Raw image bytes cannot round-trip. Adding
them would break the predicate for exactly the notes this SPEC creates.

**How image content reaches the hash anyway.** The stored media filename is
`sanitized-basename` + a suffix derived from the file's content hash
(REQ-C-012). Editing an image changes its content hash, therefore its stored
filename, therefore the `<img src>` in the rendered field HTML, therefore input
#2, therefore the hash. An image edit correctly triggers an update — for free,
through an input that already round-trips.

The same suffix solves the other open question at the same time: two files named
`diagram.png` in two different directories cannot collide in Anki's flat media
namespace, because their content hashes differ (`AC-C-011b`). Identical content
in two places converges on one upload (`AC-C-011c`).

**Consequence, specified rather than discovered** (REQ-C-016.2, `plan.md` R12): the
first run after this rendering change updates every previously synchronized
imoogi note whose rendered bytes now differ. This is the parent SPEC's D-6 rule
working as designed. It is bounded twice: a note carrying neither math nor a
local image renders byte-identically and issues no request, and the deck wrapper
is template-side so it never enters field content.

## 3. Deck-class normalization

### 3.1 Where the wrapper lives

`{{Deck}}` is a template special field Anki expands **at review time**; it does
not expand inside stored field content. The wrapper therefore lives in the card
template delivered by `createModel` and nowhere else (`D-C-7`, REQ-C-006):

```html
<div class="deck-{{Deck}}">
  <!-- card front / back body -->
</div>
```

Two consequences follow for free: the wrapper is invisible to the hash, and no
part of the theme depends on how Anki round-trips a field.

### 3.2 The value being normalized

Settled from Anki's own source rather than the manual: `{{Deck}}` is the **full
deck path including `::` separators**; `{{Subdeck}}` is the basename only.
`{{Subdeck}}` was rejected (`D-C-8`) because it collapses `A::Go` and `B::Go`
onto one class.

CSS identifier rules make normalization mandatory, not cosmetic: a class may not
contain an unescaped `:` or space, and may not begin with an unescaped digit.

### 3.3 The algorithm (REQ-C-006)

Applied in exactly this order, as a total function:

```
normalize(deck string) string:
  1. s := lowercase(deck)                     // ASCII case fold
  2. s := replaceAll(s, "::", "-")            // each separator → one hyphen
  3. s := map each rune r:                    // rune-wise
           r if r in [a-z0-9_-] else '-'
  4. s := collapse runs of '-' to a single '-'
  5. s := trim leading and trailing '-'
  6. if s begins with a digit: s := "_" + s   // guard character
  7. if s == "": s := "unnamed"                // empty-token sentinel
  8. return s
```

Step 2 runs before step 3 so a `::` becomes **one** hyphen rather than two; step
4 then keeps every other run to one as well. Step 6's guard character is `_` —
inside the permitted class, and distinct from the `-` separator so it cannot be
confused with a stripped edge.

The guard at step 6 is defensive: the emitted class is `deck-` + the token, which
already begins with a letter, so a digit-leading token could never actually
produce an invalid class in this SPEC's own usage. It is specified anyway so the
normalizer is a correct standalone function and stays correct if a later SPEC
uses the token without the prefix.

Step 5 can produce the empty string (a deck named `::`, `---`, or `!!!` — one
made entirely of separators or punctuation). Step 7 maps that to the literal
sentinel `unnamed`, so the emitted class is `deck-unnamed` rather than the bare
`deck-` (REQ-C-006.2). The bare form is a valid CSS identifier, but it is also
the exact prefix every other deck class begins with, so a user rule written as
`.deck-` would be indistinguishable from a typo for a real deck. The sentinel
makes the empty case nameable and selectable. No error, no diagnostic.

### 3.4 Worked example — the user's real deck

The user's own measured deck name is `(PROGRAMMER)::(GO)`, which proves both
parentheses and `::` occur in practice.

| Step | Value |
|---|---|
| input | `(PROGRAMMER)::(GO)` |
| 1 lowercase | `(programmer)::(go)` |
| 2 `::` → `-` | `(programmer)-(go)` |
| 3 rune map | `-programmer---go-` |
| 4 collapse | `-programmer-go-` |
| 5 trim | `programmer-go` |
| 6 digit guard | `programmer-go` (not applicable) |
| 7 empty sentinel | `programmer-go` (not applicable) |
| **emitted class** | **`deck-programmer-go`** |

A user styling that deck writes `.deck-programmer-go { … }` in their own
stylesheet.

## 4. Embedded assets — templates and the base stylesheet

### 4.1 Packaging

Both card templates and the base stylesheet are `//go:embed`ed into
`internal/anki/model` (`D-C-15`). This is a **first** for the repository — no
`//go:embed` and no `.css` file exists in the tree today, and the only existing
styling is a short inline `<style>` string constant in the org-preview server.
Embedding is mandatory rather than tidy: the air-gap rule forbids a network
fetch, and a stylesheet of this size as a Go string constant would be neither
readable nor lintable.

The **user** stylesheet is not embedded. It arrives in the install request as
text (§ 6).

### 4.2 Template structure

Two types, four templates:

| Model | Field list | Card | Front | Back |
|---|---|---|---|---|
| `imoogi-Basic` | `Front`, `Back` | `Card 1` | `deck-` wrapper around `{{Front}}` | wrapper around `{{FrontSide}}`, a rule, `{{Back}}` |
| `imoogi-Cloze` | `Text`, `Back Extra` | `Cloze` | wrapper around `{{cloze:Text}}` | wrapper around `{{cloze:Text}}` and `{{Back Extra}}` |

`imoogi-Cloze` is created with `isCloze: true` — the same `createModel` action, no
separate endpoint. Field names mirror the stock types exactly (`D-C-14`,
REQ-C-001), so the renderer's output map shape is unchanged and the planner's
field-resolution layer needs no modification. That layer exists to handle a
*customized* stock `Basic` carrying lowercase field names — the user's actual
measured situation — so the risk it guards is new code bypassing it and breaking
the stock case, not imoogi's own models needing it.

### 4.3 Base stylesheet structure (REQ-C-007; `D-C-4`)

**This section records why the visual direction is what it is; REQ-C-007.2 fixes
what the values are.** The seven `--imoogi-*` custom properties, their literal
light and night values, and the declarations that consume them are normative
there and are not restated here — a value appearing in both places is a drift
site, which is the defect this split removes.

Ordered blocks:

1. **Reset and box model** — `.card` base, `word-wrap: break-word`.
2. **Media** — `.card img { max-width: 100%; max-height: none; }`.
3. **Typography tokens** — a serif stack for headings and a system sans-serif
   stack for body, both naming system families only. No `@font-face`, no
   `@import`, no `url(`, no `http`.
4. **Measure and rhythm** — a bounded content measure and a generous line height,
   because a bounded measure is what makes a long cloze readable at phone size.
5. **Light theme** — a warm paper ground with a near-black ink.
6. **Night theme** — duplicated under **both** selector forms:
   `.card.nightMode` (Anki desktop, AnkiMobile) and `.card.night_mode`
   (AnkiDroid — no space between the class names). A near-black ground with
   reduced-contrast body text.
7. **Math and code** — raised contrast in both themes. This is the one place a
   reduced-contrast night body would actively harm comprehension, so it is
   exempted deliberately rather than inherited.
8. **Deck hook comment** — a comment naming the `deck-<normalized>` convention
   and pointing at the user stylesheet, and **no** `deck-*` rule of imoogi's own
   (REQ-C-009).

The air-gap property of block 3 is mechanically checkable as a text assertion
over the embedded asset (`AC-C-005`), not a review judgment.

## 5. Diagnostic codes

New `protocol.Code*` constants, each paired with an entry in the Elisp diagnostic
table; the existing contract test enforces the subset relation (REQ-C-023).

| Constant | Wire value | Raised when | Requirement |
|---|---|---|---|
| `CodeModelInstallFailed` | `model_install_failed` | a `createModel` / `updateModelStyling` / `updateModelTemplates` request fails during the install step | REQ-C-002 |
| `CodeMediaFileNotFound` | `media_file_not_found` | a referenced local image cannot be resolved, is outside the sync root, or is unreadable | REQ-C-012, REQ-C-015 |
| `CodeMediaUploadFailed` | `media_upload_failed` | `storeMediaFile` fails for a resolved, readable file | REQ-C-015 |
| `CodeMigrationAddFailed` | `migration_add_failed` | a confirmed migration's `addNote` fails; the original is left untouched | REQ-C-022 |

### 5.1 The one new `action` enum value

`protocol.Result.Action` carries five values today — `added`, `updated`,
`skipped`, `deleted`, `failed`. This SPEC adds a sixth, `migrate_candidate`,
emitted only by `migrate --dry-run` (REQ-C-018).

| Value | Emitted by | `note_id` | Front-end handling |
|---|---|---|---|
| `migrate_candidate` | `migrate --dry-run`, one per candidate entry | the entry's existing note identifier | counted only; triggers no write-back and no property edit |

It is a new **value**, not a new field: `Result` stays `{key, action, note_id}`
and `protocol.Version` stays constant, so REQ-C-018's wire-stability clause
holds. It is nonetheless a contract change — an Elisp front end that switched
exhaustively on the five known values would not recognize it — so it is listed
here, in `spec.md` § 7 under both Protocol and the Elisp delta, and in § 11
below.

The existing `note_type_change_unsupported` is **reused unchanged** for
REQ-C-021's complement — no new code, because the behavior is the parent SPEC's
REQ-021 behavior and the corrective prose is what changes (it now names
`imoogi-anki-setup` as the remedy).

> Derived, not carried from `plan.md`: the constant names and wire values above
> are this design's proposal. `plan.md` requires that new codes exist on both
> sides (REQ-C-023) but does not enumerate them.

## 6. Install-step sequence (`D-C-6`)

```
 imoogi-anki-setup (Elisp)
   … existing configuration steps …
   1. read the user stylesheet at the defcustom path
      → present ? its contents : ""                       (REQ-C-008)
   2. build the install request: { protocol_version, ankiconnect_url, user_css }
   3. call: imoogi-anki install-models   (stdin JSON → stdout JSON)

 install-models (Go)
   4. probe protocol_version, as runSync does; mismatch ⇒ binary_incompatible
   5. css := base_stylesheet + user_css                    (REQ-C-008)
   6. names := ModelNames()
   7. for each of { imoogi-Basic, imoogi-Cloze }:
        absent  ⇒ CreateModel{name, fields, templates, css, isCloze}   (REQ-C-002)
        present ⇒ UpdateModelStyling{name, css}                        (REQ-C-002)
                  UpdateModelTemplates{name, templates}
   8. report per-type outcome; any failure ⇒ model_install_failed
```

**Probe-then-act, not create-and-catch.** Whether `createModel` on an existing
name errors, no-ops, or overwrites is undocumented (`plan.md` R9); probe-then-act
never reaches that branch, so the unknown is unreachable rather than merely
unlikely.

**Two layers, one of which is not user-facing.** The user's command surface gains
nothing — installation rides `imoogi-anki-setup`, so a user following the
existing setup instructions ends up with the models installed. The Go binary
still needs a verb to dispatch on, so `install-models` exists as an internal
subcommand named for what it does, never documented as a user entry point.

**Wholesale replacement is announced** (REQ-C-002): step 7's update path replaces
the type's entire CSS rather than merging, so a hand edit made inside Anki is
discarded. The command says so before writing.

**Hot-path isolation.** None of steps 4-8 is reachable from `sync`. That
isolation is what lets the parent's per-run blanket survive verbatim on the hot
path (`AC-C-003a`) and be lifted only on a command the user explicitly invokes
(`AC-C-003b`) — the smallest possible amendment to a shipped, audited parent.

## 7. Migrate-subcommand sequence and failure modes (`D-C-6`, `D-C-13`)

### 7.1 Sequence

```
 imoogi-anki-setup, after the install step
   1. call: imoogi-anki migrate --dry-run   (sync-shaped request)
        → ordinary Response; results[] carries one entry per candidate,
          each { key, action: "migrate_candidate", note_id };
          N := len(results); zero AnkiConnect writes            (REQ-C-018, C-019)
   2. N == 0 ⇒ done, no prompt
   3. N  > 0 ⇒ y-or-n-p naming N and the loss of review history
               and scheduling state
        declined ⇒ no writing migrate request is issued at all        (REQ-C-019)
   4. confirmed ⇒ call: imoogi-anki migrate   (same request, no dry-run)

 migrate (Go), per candidate entry
   5. gate: recorded type is stock AND declared type is that stock type
            or its imoogi- counterpart                                (REQ-C-020)
            otherwise ⇒ skip note_type_change_unsupported             (REQ-C-021)
   6. render the entry under the counterpart type (full pipeline of § 2)
   7. AddNote(counterpart, rewrittenFields, deck, tags)
        failure ⇒ migration_add_failed; original, registry entry, and
                  heading properties untouched; continue              (REQ-C-022)
   8. success ⇒ report the new identifier for write-back
   9. DeleteNotes([original]) — only now, through the existing
        ownership-confirmed delete path                               (REQ-C-020)
  10. registry entry ← { new noteID, imoogi- type, new hash }         (REQ-C-020)

 write-back (Elisp)
  11. org-entry-put ANKI_NOTE_ID   ← new identifier                   (REQ-C-020)
      org-entry-put ANKI_NOTE_TYPE ← imoogi- counterpart
```

**Why obtaining the count does not presuppose the answer** (REQ-C-019). Step 1
runs before the prompt of step 3, which looks like acting before consent. It is
not: the dry-run reads the registry and issues **zero** AnkiConnect writes
(`AC-C-018b` asserts exactly that over every write call log), so the only thing
it produces is the number the user needs in order to answer. Asking "migrate
some unknown quantity of notes?" would be the worse consent, not the better one.

### 7.2 Why add precedes delete

Delete-first leaves a window in which the registry names an identifier Anki no
longer holds; a crash there strands the entry with no automated recovery.
Add-first leaves the opposite window — a duplicate note, which is visible,
recoverable, and never a data loss. `addNote` will not dedupe against the
stock-type original, because Anki scopes duplicate detection within a note type
and the model differs.

### 7.3 Failure modes

| Failure | Behavior | Requirement |
|---|---|---|
| Dry-run cannot read the registry | abort with the existing `state_unreadable` path; no prompt | (existing) |
| User declines | no writing `migrate` request at all; entries stay on stock types; prompt recurs at the next setup | REQ-C-019 |
| Render fails for a candidate | skip that entry with its render diagnostic; original untouched; continue | REQ-C-022 |
| `AddNote` fails | `migration_add_failed`; original note, registry entry, and heading properties all untouched; continue | REQ-C-022 |
| `DeleteNotes` fails after a successful add | the new note and the updated registry entry stand; the original survives as a visible duplicate, reported — never a lost note | REQ-C-020 (add-before-delete ordering) |
| Declared type is neither the recorded stock type nor its counterpart | skip `note_type_change_unsupported` | REQ-C-021 |

### 7.4 Disjointness from parent REQ-021

REQ-C-020 is gated by a `Where` clause naming a **confirmed migration**;
REQ-C-021 governs every other path plus the migration path's own non-counterpart
residue. No (recorded, declared) pair is left ungoverned, and the partition is in
requirement *text* rather than in branch ordering — which is precisely the
correction the parent SPEC's D-12 had to make after audit, and which
`research.md` § 10 names as a recurring blocking-defect theme.

### 7.5 The candidate set is coarser than the intent

The registry cannot distinguish a legacy stock-type entry from one whose stock
type the user chose deliberately: both are imoogi-created entries recording type
`Basic`. The candidate set is therefore every registry entry recording a stock
type, migration is all-or-nothing per confirmation, and the confirmation text
names the count and the scheduling loss so the decision is informed. Measured
scale: 5 entries. Per-heading opt-out is out of scope (`spec.md` § 5).

## 8. AnkiConnect surface additions

### 8.1 New methods

Added to the `AnkiConnector` interface (ten methods today) and to the concrete
client. All five funnel through the existing single `call(ctx, action, params)`
chokepoint; the `TransportError` / `ProtocolError` / `APIError` tiering is
unchanged.

| Method | Action | Params | Result | Used by |
|---|---|---|---|---|
| `ModelNames` | `modelNames` | none | `[]string` | install probe (REQ-C-002) |
| `CreateModel` | `createModel` | `modelName`, `inOrderFields`, `css`, `isCloze`, `cardTemplates[]{Name,Front,Back}` | model | install, absent branch |
| `UpdateModelStyling` | `updateModelStyling` | `model{name, css}` | none | install, present branch |
| `UpdateModelTemplates` | `updateModelTemplates` | `model{name, templates{cardName:{Front,Back}}}` | none | install, present branch |
| `StoreMediaFile` | `storeMediaFile` | `filename`, `path` (local absolute) | stored filename | media dispatch (REQ-C-012) |

`storeMediaFile` accepts `data` (base64), `path`, or `url`; `path` is used, so no
base64 encoding is needed for a local file.

> **Verification action before signatures are pinned** (`plan.md` R10): the
> canonical AnkiConnect documentation source was unreachable during research and
> all request shapes were read from a fork mirror of medium confidence. Re-verify
> every shape above against the canonical README at milestone M1. The wording
> already matches the parent SPEC's own earlier direct fetch, so the expected
> outcome is confirmation.

### 8.2 Fake-client counterparts

`fakeClient` in the planner's test package implements all five with per-action
call logs matching the existing `addCalls` / `createDeckCalls` shape:

| Log | Records | Makes checkable |
|---|---|---|
| `modelNamesCalls` | invocation count | that install probes exactly once |
| `createModelCalls` | full params per call | `AC-C-001`, `AC-C-003a/b` |
| `updateModelStylingCalls` | model name + css | `AC-C-002`, `AC-C-006`, `AC-C-007` |
| `updateModelTemplatesCalls` | model name + templates | `AC-C-002`, `AC-C-003b` |
| `storeMediaFileCalls` | filename + path | `AC-C-011`, `AC-C-014`, `AC-C-015b` |

Each new method arrives **with** its own log rather than sharing one, because
every negative assertion in `acceptance.md` is a count on a specific log
(`plan.md` § G). This is also why milestone M1 exists as a standalone step: it
adds the five methods and the five logs and changes no behavior, so the
zero-model-writes regression fence exists before anything could violate it, and
the window in which the test double is out of date is one milestone wide
(`plan.md` R13).

## 9. Media pass

### 9.1 Shape

`internal/anki/media` exposes a single entry point taking `(baseDir,
renderedFields)` and returning `(rewrittenFields, []Upload)`. It performs
read-only local I/O (reading bytes to compute a content hash) and no network
call — the stored filename is computed client-side, so the rewrite and the hash
both complete before any request is issued.

`orgdoc.Render`'s signature is therefore **unchanged** (`D-C-11`): go-org's parse
base governs `#+INCLUDE` resolution, not the `src` attributes the media pass
rewrites afterwards, and the pass receives `baseDir` itself.

### 9.2 Resolution and confinement

Base directory: the directory of the entry's own Org file, derived from the
configured sync root and the entry's recorded source path. Resolution reuses the
org-preview asset resolver's existing logic — URL-unescape, join against the base
directory, canonicalize with absolute-path plus symlink evaluation tolerating a
non-existent tail, and confine to the allowed root. The org-preview **link
regex** is deliberately *not* reused: it belongs to a different parser than the
one the Anki path uses.

### 9.3 Stored naming

`sanitized-basename` + `-` + a short prefix of the file's content SHA-256,
inserted **before** the extension: `diagram.png` → `diagram-3f9a1c2b8e40.png`.

Fixed here so REQ-C-012.3 can cite one definition rather than restate it:

| Element | Rule |
|---|---|
| basename | the reference's final path element with its extension removed |
| sanitization | lowercase; every character outside `[a-z0-9_-]` replaced by `-`; `-` runs collapsed to one; leading and trailing `-` stripped; empty result replaced by `img` |
| separator | a single `-` between sanitized basename and digest |
| digest | the first **12** lowercase hex characters of the file content's SHA-256 |
| extension | the original extension, lowercased, re-appended after the digest |

Content hashing follows the existing in-repo `fileSHA256` helper shape. Twelve
hex characters is 48 bits, which makes an accidental collision across a personal
collection's media set negligible while keeping the stored name readable in
Anki's media browser. No `_` prefix (REQ-C-013) — Anki treats a leading `_` as "ignore during
media checks", which would exempt imoogi's uploads from the very Check Media
cleanup that keeps orphan-media out of this SPEC's scope.

### 9.4 The two-part link rule

The renderer emits `[[file:diagram.png][My diagram]]` as an anchor, not an
image, because it classifies a link as an image only when it has no description
(or when the description is itself an image URL). Because the media pass already
walks rendered HTML, converting that anchor to
`<img src="…" alt="My diagram">` is one more rule in a pass that exists, not a
new mechanism, and the description survives as accessibility text rather than
being discarded.

**Extension set: exactly go-org's, with nothing added.** `.avif`, `.bmp`, and
`.ico` are excluded for consistency rather than caution — go-org decides the
one-part case and imoogi decides the two-part case, so any extension imoogi
recognized and go-org did not would make `[[file:x.avif]]` an anchor and
`[[file:x.avif][d]]` an image, the same file behaving differently for a reason no
user could infer. Adding an extension is a follow-up that changes both decisions
together.

### 9.5 Dispatch gating

The planner rewrites, hashes, compares, and then issues `storeMediaFile` **only
for the entries it is about to add or update** (REQ-C-014), deduplicated by
stored filename within the run (REQ-C-014). Uploading during the rewrite instead
would fire a media request for every image-bearing note on every run, including
an otherwise-no-op one — the media analogue of the AC-015 violation this SPEC is
careful to avoid elsewhere.

## 10. Math transform

Lives in `orgdoc` beside the existing cloze-marker post-render guard, which is
the same shape of change with the same documented rationale.

### 10.1 Why imoogi applies Org's `$` heuristic itself (`D-C-12`)

go-org's fragment parser, for `$`, requires only that a closing `$` exist
somewhere in the remainder — no word-boundary, whitespace, or digit checks. So
`costs $5 and $7` is **already** parsed as a fragment today. The delimiters
currently survive verbatim, so the HTML looks unchanged and nobody notices; the
moment imoogi rewrites `$…$` → `\(…\)`, that silent misparse becomes visible
broken MathJax on the user's card. imoogi must therefore decide which `$` pairs
to convert using **Org's** rule (REQ-C-010), not the parser's:

- at most two line breaks in the enclosed text,
- attached to both `$` characters with no intervening whitespace,
- the closing `$` followed by whitespace or by punctuation other than a dash.

### 10.2 Emission rules

- `$…$` and `\(…\)` → `\(…\)`; `$$…$$` and `\[…\]` → `\[…\]`. No other delimiter
  form is emitted — Anki's cloze logic has special handling that a custom
  delimiter breaks.
- A line break inside a converted fragment becomes `<br>`; a bare newline is
  unrenderable by Anki's MathJax.
- `$` inside a `<pre>` region, a `<code>` region, an attribute value, or a link
  target is never converted.
- No network-hosted MathJax reference is emitted; Anki's bundled copy is used.

### 10.3 Two behaviors carried as regression tests rather than assertions

- **Entity escaping inside math** (`plan.md` R11): the renderer HTML-escapes raw
  text, so `a < b` reaches Anki as `a &lt; b`. MathJax reads DOM text nodes,
  where `&lt;` has already been decoded to `<`, and unescaping in the source
  would produce invalid HTML. Keep the escaping; verify rendering on a real card;
  record the outcome.
- **Sub/superscript inside a fragment** (`research.md` C2): the pinned renderer
  version parses fragment interiors as raw text, so `_` and `^` are protected —
  but no lens verified the upstream fix version. `AC-C-024` carries this as a
  regression test, not as an assertion of upstream behavior.

## 11. Elisp touch points

| Module | Change | Requirement |
|---|---|---|
| `modules/24-anki.el` | `ANKI_NOTE_TYPE_ALL` offers both imoogi-owned names; the two note-type-writing commands write them; the cloze auto-mark writes `imoogi-Cloze` and its comparison accepts either `Cloze` form | REQ-C-005 |
| `modules/anki/imoogi.el` | New `defcustom` for the user-stylesheet path, defaulting to `imoogi-anki.css` beside the existing config file in the user's Emacs directory; read at install time, absent ⇒ empty string. Note-type literals switched | REQ-C-008, REQ-C-005 |
| `modules/anki/imoogi-setup.el` | `imoogi-anki-setup` gains its tail: read the user CSS → `install-models` → `migrate --dry-run` for the count → `y-or-n-p` naming the count and the scheduling loss → `migrate` on confirmation | REQ-C-002, REQ-C-019 |
| `modules/anki/imoogi-process.el` | Request construction for the two new subcommands — the install request document `{protocol_version, ankiconnect_url, user_css}` and the migrate request. Handles the new `migrate_candidate` action value as a count-only outcome (`(length results)`), triggering no write-back. The pinned protocol-version constant is **unchanged** | REQ-C-008, REQ-C-018 |
| `modules/anki/imoogi-error.el` | One table entry per new `protocol.Code*`; corrective prose updated to name the imoogi-owned types and to direct a hand-edited note-type change to `imoogi-anki-setup` rather than to a manual property edit | REQ-C-023, REQ-C-005, REQ-C-021 |
| `modules/anki/imoogi-writeback.el` | **No new mechanism.** The existing property write already overwrites rather than only inserting, so REQ-C-020.3's two-property replacement needs nothing new — verified, not assumed. The `ANKI_NOTE_TYPE` value is derived locally as the `imoogi-` counterpart of the declared type, since `protocol.Result` carries no note type | REQ-C-020 |
| `tests/anki-error-test.el` | The existing contract test now covers the new codes; it fails if either side is added without the other | REQ-C-023 |

**Transport of the user stylesheet, decided rather than assumed** (`D-C-3`): the
file's location is an Elisp `defcustom`, so the **front end** reads it and passes
its contents in the install request; the Go binary reads no stylesheet from disk.
This preserves the wire contract's documented property that the binary reads no
configuration file of its own, keeps the missing-file branch in the one place
that already owns path expansion, and adds no new diagnostic code.

## 12. Decision index

Design decisions referenced above, by `plan.md` § D identifier:

| Id | Decision | Sections here |
|---|---|---|
| `D-C-1` | imoogi-owned types rather than restyled stock types | § 4.2, § 8 |
| `D-C-2` | Ownership decided by the `imoogi-` name prefix | § 6, § 8.2 |
| `D-C-3` | Per-deck appearance user-authored; one base stylesheet | § 4.3, § 11 |
| `D-C-4` | Restrained editorial direction, system fonts only | § 4.3 |
| `D-C-5` | imoogi-owned types become the front end's default | § 11 |
| `D-C-6` | Install rides setup; migrate is a separate subcommand | § 6, § 7 |
| `D-C-7` | Per-deck styling via a template-side `{{Deck}}` wrapper | § 3.1 |
| `D-C-8` | `{{Deck}}` is the full `::` path; normalization mandatory | § 3.2, § 3.3 |
| `D-C-9` | Transforms run before the hash; stored bytes equal hashed bytes | § 2.2 |
| `D-C-10` | Image content reaches the hash through the filename | § 2.3, § 9.3 |
| `D-C-11` | Media is a separate post-render pass; `Render` stays pure | § 9 |
| `D-C-12` | imoogi applies Org's `$` heuristic itself | § 10.1 |
| `D-C-13` | Migration adds before it deletes; disjoint by requirement text | § 7.2, § 7.4 |
| `D-C-14` | Field names mirror the stock types | § 4.2 |
| `D-C-15` | Templates and base CSS ship as embedded assets | § 4.1 |
