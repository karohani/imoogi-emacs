# SPEC-ANKICARD-001 — Compact

Run-phase load target. Requirements + acceptance criteria + touch surfaces +
exclusions only. Full context: `spec.md`, `design.md`, `plan.md`, `research.md`.

- id: `SPEC-ANKICARD-001` · tier: L · status: draft · phase: `v0.2.0 target` · version: `0.1.2`
- amends: `SPEC-ANKI-001` (out-of-repo) — REQ-016, AC-016, §4, D-12

**Ownership predicate**: an *imoogi-owned note type* is one whose model name
begins with the literal prefix `imoogi-`. Exactly two exist: `imoogi-Basic`,
`imoogi-Cloze`. Everything else is foreign.

## Requirements (GEARS)

24 requirements, seven groups. Each carries exactly one GEARS trigger; numbered
sub-clauses are case-splits on that trigger's operand and are not independent
requirements. `[Ubiquitous — negated]` and `[While — negated]` are the `shall not`
forms of those two patterns, not a sixth pattern. Full wording: `spec.md` § 3.
Old→new mappings: `plan.md` § Revision 3 (42 → 24) and § Revision 4 (the v0.1.2
re-partition).

### Note types and installation
- **REQ-C-001** [Ubiquitous] (1) A note type is imoogi's own exactly when its model name begins with `imoogi-`; exactly `imoogi-Basic` and `imoogi-Cloze` exist — any other `imoogi-`-prefixed model is neither created, updated, nor reported. (2) Field names mirror the stock types — `Front`/`Back`, `Text`/`Back Extra` — so the renderer's map shape and the planner's field-resolution layer are unchanged.
- **REQ-C-002** [When] When the user invokes `imoogi-anki-setup`, after its existing configuration steps query `modelNames`, then: (1) one `createModel` per absent imoogi-owned type carrying field list, card templates, and CSS — `imoogi-Cloze` with `isCloze: true`; (2) `updateModelStyling` + `updateModelTemplates` for each type already present, so repeated setup is idempotent; (3) on a styling update, replace the type's entire CSS with the base+user concatenation, discarding hand edits, stating so before writing.
- **REQ-C-003** [While — negated] While an ordinary synchronization run is executing, issue no `createModel` / `updateModelStyling` / `updateModelTemplates` on any code path that run reaches.
- **REQ-C-004** [Ubiquitous — negated] Do not modify the templates or styling of any note type whose name does not begin with `imoogi-`, on any code path. Supersedes the narrowed clause of parent REQ-016 (§ 4.2); every other clause stays in force.
- **REQ-C-005** [Ubiquitous] The imoogi system names its note types consistently across every surface carrying a note-type literal, realized as: (1) the imoogi-owned types are the front end's default — marking commands, cloze auto-mark, `ANKI_NOTE_TYPE_ALL`, renderer dispatch. (2) a heading declaring stock `Basic`/`Cloze` still synchronizes exactly as the parent's shipped behavior does — characterization baseline: the planner's existing stock-type add/update/no-op tests — unthemed, with both name sets recognized on both sides and the planner's `modelFieldNames` read for a stock type preserved. (3) every site hard-coding a stock literal names the imoogi-owned types where those are in play, while still accepting the stock literals per (2).

### Card appearance
- **REQ-C-006** [Ubiquitous] (1) Each card template wraps its content in one element carrying a `{{Deck}}`-derived CSS class; the wrapper exists only in the template, never in stored field content. (2) Normalization: lowercase → each `::` to one `-` → every remaining char outside `[a-z0-9_-]` to `-` → collapse `-` runs → strip leading/trailing `-` → prefix `_` if the result begins with a digit → if the token is then empty, the literal sentinel `unnamed`, so the class is `deck-unnamed` and never a bare `deck-`.
- **REQ-C-007** [Ubiquitous] (1) CSS carries both `.card.nightMode` and `.card.night_mode` (no space), plus `.card img { max-width: 100%; max-height: none; }` and `word-wrap: break-word;`. (2) Seven CSS custom properties declared with exactly these literal values — `--imoogi-bg` `#f7f3e9` light / `#1a1a1a` night; `--imoogi-fg` `#1c1a17` / `#c8c4bc`; `--imoogi-emphasis-fg` `#0b0a09` / `#f5f2ec`; `--imoogi-measure` `44rem`; `--imoogi-line-height` `1.6`; `--imoogi-serif` and `--imoogi-sans` as literal system stacks — consumed as `.card` background/color/font-family/line-height, `h1`-`h6` font-family, deck-wrapper `max-width`, and the math-and-code rule's `color`. **System font stacks only, no network resource of any kind.** Rationale (not values): `design.md` § 4.3.
- **REQ-C-008** [Where + When] Where a user stylesheet exists at the configured path (`imoogi-anki.css` beside `imoogi.json` by default), when the install step runs the imoogi system uploads base + that file verbatim, realized as: the front end carrying the contents in the install request's `user_css` field, and the Go binary concatenating them after the base stylesheet, reading no stylesheet from disk. Absent ⇒ `user_css: ""` and base alone.
- **REQ-C-009** [Ubiquitous — negated] Assign/rotate/derive no per-deck appearance from a deck name or hash; the base stylesheet is byte-identical for every deck, and every per-deck rule originates in the REQ-C-008 user stylesheet.

### LaTeX to MathJax
- **REQ-C-010** [When] When rendered content carries an Org math fragment: (1) emit `\(…\)` for `$…$` and `\(…\)`, `\[…\]` for `$$…$$` and `\[…\]`, and no other delimiter form; (2) recognize a single `$` by the Org inline-math heuristic `spec.md` § 2 defines, and by no other rule; (3) emit `<br>` in place of any line break the converted fragment contains.
- **REQ-C-011** [Ubiquitous — negated] Do not convert a `$` inside `<pre>`, `<code>`, an attribute value, or a link target; emit no network-hosted MathJax or stylesheet reference.

### Images and media
- **REQ-C-012** [When] When rendered content references a local image — relative path, `file:` link, or the two-part `[[file:x.png][description]]` form: (1) for a reference the renderer emitted as an anchor whose target carries a go-org image extension (`png gif jpg jpeg svg tif tiff webp xbm xpm pbm pgm ppm pnm`, case-insensitive), first rewrite it to an `img` whose `alt` carries the description, then apply (2) and (3) as to a one-part reference; (2) resolve against the entry's own Org-file directory (sync root + recorded source path) and confine the resolved path to that sync root; (3) for a reference resolving to a readable file, compute the stored name by the function `design.md` § 9.3 fixes — sanitized basename + `-` + 12 hex chars of the content SHA-256, inserted **before** the extension — and rewrite `src` to that exact name; computed locally, independent of any upload response.
- **REQ-C-013** [Ubiquitous — negated] Do not (1) rewrite, upload, or disturb a reference whose target carries an `http:` / `https:` scheme; (2) `_`-prefix an uploaded media filename; (3) delete, or enumerate for deletion, any media file not uploaded during the run in progress.
- **REQ-C-014** [While] While a single run is in progress: (1) issue `storeMediaFile` only for entries that run actually adds or updates — an unchanged hash issues no media request at all; (2) upload each distinct stored media filename at most once, however many entries reference it.
- **REQ-C-015** [When — event-detected] When a local image is detected unresolvable, unreadable, or unuploadable: report the target skipped carrying `media_file_not_found` (resolve/read) or `media_upload_failed` (upload), naming the reference, create/update no note for it, leave its registry hash unchanged, and continue with the remaining targets.

### Hashing and ordering
- **REQ-C-016** [Ubiquitous] (1) Every transform this SPEC introduces (math, two-part-link, `src`) is applied **before** the content hash is computed for the add/update/no-op decision, and the field content sent to AnkiConnect is byte-identical to the content whose hash is recorded in the registry. (2) Add/update/no-op is decided for every previously synchronized imoogi note solely from that post-transform hash, so a note whose rendered content differs updates and a byte-identical one issues no request — including on the first run after this rendering change, whose invalidation is specified behavior, not a defect (`design.md` § 2.3). *(Absorbed former REQ-C-018 at v0.1.2.)*
- **REQ-C-017** [Ubiquitous — negated] Add no input to the content hash beyond the four parent D-6 fixes — note type, rendered field values, resolved deck, sorted tag set; image content reaches the hash solely via the content-derived stored filename REQ-C-012 writes into the field text.

### Migration
- **REQ-C-018** [Ubiquitous] Migration is reached through `imoogi-anki migrate`, reading a sync-shaped request and writing the same response schema; no wire document gains a field and the protocol version constant is unchanged. Dry-run reports each candidate as one ordinary `results[]` entry carrying the **new `action` value `migrate_candidate`** and the entry's existing note id, so the count is `len(results)`; the front end treats that value as count-only, triggering no write-back.
- **REQ-C-019** [When] When setup completes the install step and the registry records stock-type entries, display the count and obtain explicit confirmation — stating the loss of review history and scheduling state — before any **writing** `migrate` request; a decline issues none. (`design.md` § 7.1 records why the count does not presuppose the answer.)
- **REQ-C-020** [Where + When] Where migration has been invoked and confirmed under REQ-C-019, when an entry's recorded type is stock `Basic`/`Cloze` **and** its declared type is that stock type or its `imoogi-` counterpart, the imoogi system re-homes it onto the counterpart, realized as: (1) the Go binary renders under the counterpart, adds, and deletes the original only after the add succeeded; (2) the Go binary then replaces the registry record with the new identifier and imoogi-owned type and reports the new identifier in that entry's `results[]` record; (3) the Elisp front end overwrites `ANKI_NOTE_ID` from that report and `ANKI_NOTE_TYPE` with the locally derived `imoogi-` counterpart, since `protocol.Result` carries no note type.
- **REQ-C-021** [When — event-detected] When an entry's registry-recorded note type is detected to differ from its declared type — the hand-edited stock-to-`imoogi-` case included — skip with `note_type_change_unsupported`, write no field, continue; **unless** the detection occurs inside a migration confirmed under REQ-C-019 and the declared type is the recorded stock type's `imoogi-` counterpart, which is the REQ-C-020 case. Parent REQ-021 unchanged; the two are partitioned **by requirement text**, not by branch ordering.
- **REQ-C-022** [When — event-detected] When a confirmed migration's add step is detected to have failed, leave the original note, its registry entry, and its heading properties untouched; report the entry skipped carrying `migration_add_failed`; continue — so no entry ever reaches a state where the registry names an identifier the collection no longer holds.

### Contracts and non-interference
- **REQ-C-023** [Ubiquitous] Every diagnostic code this SPEC introduces — `model_install_failed`, `media_file_not_found`, `media_upload_failed`, `migration_add_failed` — exists as a `protocol.Code*` constant **and** as an entry in the Elisp diagnostic table (`modules/anki/imoogi-error.el`), the two sides paired one-for-one.
- **REQ-C-024** [Ubiquitous — negated] Do not (1) detect, report, or delete unreferenced collection media — Anki's Check Media owns it, which REQ-C-013's no-underscore rule keeps possible; (2) upload, rewrite, or diagnose a video/audio target or any extension outside the REQ-C-012 image set — such targets pass through exactly as rendered; (3) create or modify any note type beyond the two — the remainder of the parent's "advanced card types" exclusion staying in force as non-normative § 5 prose. **Reading** a foreign model is not prohibited: the `modelNames` probe (REQ-C-002) and the `modelFieldNames` read (REQ-C-005.2) are both required; (4) modify the stock `Basic`/`Cloze` types themselves — their model definitions, card templates, or CSS — on any code path including migration. Note-level work on a note carried on a stock type is outside this clause and is governed by REQ-C-005.2 (ordinary sync) and REQ-C-020 (confirmed migration). This is the narrow collection-ownership guarantee § 4.4 inherits from the parent.

## Acceptance Criteria (Given / When / Then)

Full text with sub-criteria: `acceptance.md`. Condensed:

- **AC-C-001** Given no imoogi-owned type in `modelNames`, When install runs, Then exactly one `createModel` per type with the correct field list (+ `isCloze` for Cloze) and zero update requests.
- **AC-C-002** Given both types present, When install runs, Then one `updateModelStyling` + one `updateModelTemplates` per type, zero `createModel`; a second run is byte-identical.
- **AC-C-003a** Given a sync run of any shape, When it completes, Then the three model-**write** call logs (`createModel`, `updateModelStyling`, `updateModelTemplates`) each record exactly 0. The interface exposes no scheduling / review-history / collection-styling endpoint at all — asserted by enumerating its method set in AC-C-022b.
- **AC-C-003b** Given install against a collection also holding stock and user-authored types, When it completes, Then every model write names an `imoogi-`-prefixed model and the count naming any other is exactly 0.
- **AC-C-004** Given a deck path, When normalized, Then `(PROGRAMMER)::(GO)` → `deck-programmer-go`; `A::::B` → `deck-a-b`; `2026 Review` → `deck-_2026-review`; `::` / `---` / `!!!` → `deck-unnamed`; and no stored field value contains `class="deck-`.
- **AC-C-005** Given the base stylesheet, When inspected, Then both night-mode selectors, the `.card img` rule, and `word-wrap: break-word` are present; all seven `--imoogi-*` custom properties are declared with the exact literal values REQ-C-007.2 fixes (light on `.card`, night under both night selectors) and consumed in the exact declarations it names; and `http` / `@import` / `url(` occurrences are zero.
- **AC-C-006** Given a user stylesheet, When install runs, Then uploaded CSS = base + that file verbatim; absent ⇒ base alone; the base portion is byte-identical across collections with different decks; and (`AC-C-006d`) the install request document written to `install-models` stdin is exactly `{protocol_version, ankiconnect_url, user_css}` with `user_css` byte-equal to the file, the sync `Request` gaining no field and the Go side reading no `.css` from disk.
- **AC-C-007** Given a hand-edited imoogi-owned type, When install updates it, Then the full base+user CSS replaces it wholesale and the command announces this beforehand.
- **AC-C-008** Given each of the four Org math syntaxes, When transformed, Then `$…$`/`\(…\)` → `\(…\)` and `$$…$$`/`\[…\]` → `\[…\]`, with no other delimiter form present.
- **AC-C-009** Given `costs $5 and $7 total` / a multi-line fragment / a `$` inside `<pre>`, `<code>`, an attribute, and a link target, When transformed, Then no conversion for `$5`/`$7`, `<br>` for the line break, and all four protected occurrences byte-unchanged.
- **AC-C-010** Given an image beside the entry's own Org file, When the media pass runs, Then it resolves against that directory; and a reference escaping the sync root is rejected, skipped with a code, and issues no upload.
- **AC-C-011** Given a resolvable image, When the pass runs, Then `filename` = sanitized basename + content-hash suffix inserted before the extension, and `src` equals it; two same-named files with different content produce different names; identical content **and** identical basename converge on one name and one upload; identical content with different basenames does not.
- **AC-C-012** Given `[[file:diagram.png][My diagram]]`, When the pass runs, Then an `img` with `alt="My diagram"`; `[[file:x.avif][d]]` stays an anchor with no diagnostic; `.webm`/`.mp3` targets pass through untouched.
- **AC-C-013** Given `http(s)://` image references, When the pass runs, Then both are byte-unchanged, unuploaded, and undiagnosed.
- **AC-C-014** Given an unchanged re-sync of image-bearing entries, When it runs, Then `storeMediaFile` count is exactly 0; and three entries sharing one image produce exactly one upload.
- **AC-C-015** Given a missing local image, When the run executes, Then the entry is skipped with a code, no add/update is issued for it, its registry hash is byte-unchanged, other targets still process; and no uploaded filename begins with `_`.
- **AC-C-016** Given an entry with math and an image, When added or updated, Then the dispatched field map is byte-equal to the hashed field map and recomputing the hash from the stored values reproduces the registry hash exactly; the hash input set is still exactly four; and the ownership predicate still matches after an image edit.
- **AC-C-017** *(covers REQ-C-016.2)* Given previously synchronized notes, When the first post-change run executes, Then differing notes update in that one pass and report success; a note with neither math nor image issues no request.
- **AC-C-018** Given the migrate subcommand, When compared with `sync`, Then request shape and response schema match with no new wire field and an unchanged version constant (`Result` still `{key, action, note_id}`); dry-run records exactly 0 on every AnkiConnect **write** log (the Handshake read is permitted) and returns N `results[]` entries with `action: migrate_candidate`; a decline issues no writing request and leaves the registry byte-unchanged; the prompt names the count and the scheduling loss.
- **AC-C-019** Given a confirmed migration, When it executes, Then `addNote` precedes `deleteNotes` and the delete follows a successful add; registry, `ANKI_NOTE_ID`, and `ANKI_NOTE_TYPE` all carry the new values; a non-counterpart declared type is skipped with `note_type_change_unsupported`.
- **AC-C-020** Given a failing migration add, When the run continues, Then the original note, registry entry, and heading properties are untouched and the entry is skipped with a code; and a hand-edited stock-to-imoogi heading still skips with `note_type_change_unsupported` on the sync path, its corrective prose naming `imoogi-anki-setup`.
- **AC-C-021** Given the Go protocol package and the Elisp table, When `tests/anki-error-test.el` runs, Then the two assertions it makes today hold for the four new codes (Go ⊆ Elisp; Go set == its hardcoded list), so a Go-side addition without its Elisp entry fails; **`AC-C-021c` adds the reverse assertion, which does not exist today** — an Elisp-only entry currently fails nothing. And every stock-literal site names the imoogi-owned types while still accepting both.
- **AC-C-022** Given hand-written stock `Basic`/`Cloze` headings, When a run executes, Then each synchronizes as before, unthemed; across install + sync + migrate no request on the three model-**write** endpoints names a non-`imoogi-` model (read endpoints excluded — `modelNames` returns foreign names by construction and `modelFieldNames("Basic")` is required); the real collection's stock types, templates, and CSS stay byte-unchanged and the note counts are exactly `88 − N` and `27 − M` for the dry-run-recorded candidate counts N and M; and no request enumerates or deletes collection media.
- **AC-C-023** (edge) Given `{{c1::$x^{2}$}}`, When the pipeline runs, Then the cloze marker survives byte-for-byte, the math becomes `\(x^{2}\)`, and braces stay balanced.
- **AC-C-024** (edge, regression) Given `\(\sum_{i=1}^n a_n\)`, When the pipeline runs, Then no `<sub>`/`<sup>` appears inside the fragment.

## Files to modify

| Path | Delta |
|---|---|
| `internal/anki/ankiconnect/client.go` | [MODIFY] +5 interface and client methods: `ModelNames`, `CreateModel`, `UpdateModelStyling`, `UpdateModelTemplates`, `StoreMediaFile` |
| `internal/anki/orgdoc/orgdoc.go` | [MODIFY] imoogi-owned constants + dispatch arms (stock arms kept); MathJax post-render transform; `Render` signature unchanged |
| `internal/anki/planner/planner.go` | [MODIFY] transform-before-hash ordering; migration branch disjoint from the note-type-change skip; media dispatch on add/update only |
| `internal/anki/media/` | [NEW] resolve → confine → content-hash → rewrite; two-part-anchor→`img` |
| `internal/anki/model/` | [NEW] embedded templates + base stylesheet; deck-class normalizer; base+user CSS concatenation; probe-then-act install |
| `internal/anki/protocol/protocol.go` | [NEW] `CodeModelInstallFailed`, `CodeMediaFileNotFound`, `CodeMediaUploadFailed`, `CodeMigrationAddFailed`; [NEW] one `action` **enum value** `migrate_candidate` (a value, not a field); [NEW] the install request document `{protocol_version, ankiconnect_url, user_css}`. [EXISTING] `Request`, `Config`, `Response`, `Result`, `Version` all unchanged |
| `internal/anki/hashing/` | [EXISTING] unchanged — no new hash input |
| `internal/anki/registry/` | [EXISTING] schema unchanged; migration rewrites values only |
| `cmd/imoogi-anki/main.go` | [MODIFY] two new dispatch cases: `install-models`, `migrate` (with `--dry-run`); E2E stub extended |
| `internal/anki/planner/fake_client_test.go` | [MODIFY] five new methods with per-action call logs |
| `modules/anki/imoogi-setup.el` | [MODIFY] install → scan → dry-run count → `y-or-n-p` → migrate tail |
| `modules/anki/imoogi.el` | [MODIFY] user-stylesheet `defcustom`; note-type literals |
| `modules/anki/imoogi-process.el` | [MODIFY] request construction for the two subcommands (install doc carries `user_css`); handles `migrate_candidate` as a count-only action with no write-back; version constant unchanged |
| `modules/anki/imoogi-error.el` | [MODIFY] one entry per new code; corrective prose updated |
| `modules/anki/imoogi-writeback.el` | [EXISTING] property overwrite already sufficient |
| `modules/24-anki.el` | [MODIFY] `ANKI_NOTE_TYPE_ALL`, note-type-writing commands, cloze auto-mark |
| `tests/anki-error-test.el`, `tests/*.el` | [MODIFY] contract test covers the new codes |

## Exclusions (what NOT to build)

- **Orphan media cleanup** — no detection, reporting, or deletion of unreferenced collection media; no registry media field. Anki's Check Media owns it, which is why uploads are never `_`-prefixed.
- **Video and audio media** — no upload, rewrite, or diagnostic; and no extension added beyond go-org's own image set (`.avif`, `.bmp`, `.ico` excluded deliberately).
- **Remote image download** — `http:`/`https:` references are left exactly as authored; offline non-display is the accepted cost.
- **Automatic per-deck palette assignment** — no colour/font derived from a deck name or hash; no deck→style map in `imoogi.json`. imoogi ships the hook, the user ships the rules.
- **Foreign note types and advanced card types** — no creating or modifying of stock or user-authored types (**reading** them is required and not excluded); no image-occlusion, highlight-mask, type-in-answer, or card-direction features.
- **A third `imoogi-`-prefixed model** — exactly two imoogi-owned types exist; any other `imoogi-`-prefixed model in the collection passes through untouched with no diagnostic.
- **Note-type changes outside a confirmed migration** — the parent SPEC's exclusion stands on the ordinary sync path (REQ-C-021); only the confirmed-migration case is superseded.
- **Collection backup / restoration guarantees** — only the narrow byte-unchanged guarantee of REQ-C-024.
- **Per-heading migration opt-out** — migration is all-or-nothing per confirmation.
- **Scheduling preservation across migration** — structurally impossible under delete-and-re-add; surfaced in the confirmation text as a stated cost.
