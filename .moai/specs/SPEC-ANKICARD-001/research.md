# Research — SPEC-ANKICARD-001

| Field | Value |
|---|---|
| SPEC | `SPEC-ANKICARD-001` |
| Date | 2026-09-05 |
| Lenses | codebase-precedent · external-docs · constraints-risks · prior-SPEC-memory |
| Per-lens raw reports | `research-lenses.md` |
| Synthesis body | §1 – §13 below, plus contradictions C1–C6, the open-question verdict table, and the evidence index |
| Post-plan measurements | §14 (appended 2026-09-05, after plan.md revision 2) |

---

## SPEC-ANKICARD-001 — research synthesis

Four read-only lenses (codebase-precedent, external-docs, constraints-risks, prior-SPEC-memory) converged on one picture: **the feature is technically well-supported by AnkiConnect and cleanly implementable in the existing Go architecture, but it is *forbidden* by the parent SPEC as that SPEC is currently written.** The dominant finding is not a technical blocker; it is a specification conflict that SPEC-ANKICARD-001 must explicitly resolve before any code is planned.

---

### 1. The blocking prior-SPEC conflict (all four lenses converge)

`SPEC-ANKI-001` at `/Users/jay/workspace/imoogi-org-anki/.moai/specs/SPEC-ANKI-001/` is recorded as `status: in-progress` in its own frontmatter and is **behaviorally closed at 25/25 AC** (Tier L, 22 REQ / 25 AC, plan-audit PASS 0.857 on iteration 5, 6 milestones, `progress.md` recording six completed milestones). It is treated here as behaviorally closed and specification-live — the same formulation `spec.md` § Amendments uses; neither artifact asserts a `completed` status the parent's frontmatter does not carry. Its identifiers are cited in this repo's code. Two of its clauses prohibit exactly what this SPEC proposes.

**REQ-016 (`spec.md:177`), verbatim:** "The imoogi system shall not modify Anki review history, card scheduling state, **note-type templates, or collection styling**, on any code path, including the update and delete paths, and shall not issue any delete request scoped by tag, search query, or pattern rather than by explicit individual note identifier."

`createModel` / `updateModelStyling` / `updateModelTemplates` — even on imoogi-owned models — are literally the actions REQ-016 names. There is no ownership carve-out in the text.

**AC-016 (`acceptance.md:102`), verbatim tail:** "…and no request in the run addresses any scheduling, review-history, note-type-template, or collection-styling endpoint." This is a *per-run* assertion, not a per-note-type one. A sync run that issues any model write breaks it as written. By contrast **AC-015** (`acceptance.md:95-97`) counts only "note-mutating requests and … card-moving requests" — so a read-only `modelNames` probe is AC-015-safe, while a template/styling *write* is not AC-016-safe.

**§4 out-of-scope, "advanced card types" (`spec.md:213-219`):** "Custom (non-built-in) Anki note types" are excluded; the glossary (`spec.md:58`) fixes the MVP at "only the built-in `Basic` and `Cloze` types," reinforced by C-3/REQ-003/REQ-004.

**§4 out-of-scope, "collection ownership guarantees" (`spec.md:258`):** "A guarantee that user-authored note-type templates and collection CSS are preserved as a specified feature. REQ-016 forbids imoogi from touching them; formalizing a persistence and restoration guarantee is separate work." SPEC-ANKICARD-001 inherits that deferred obligation: because it now *does* touch models, it must supply the preservation guarantee (stock `Basic` 88 notes / `Cloze` 27 notes untouched) that ANKI-001 could omit.

**Required disposition.** SPEC-ANKICARD-001 must (a) amend REQ-016 to scope it to note types imoogi did not create, (b) amend or add an AC replacing AC-016's blanket per-run prohibition with an ownership-scoped one, and (c) declare that it supersedes §4's custom-note-type exclusion — it is the "later SPEC" §4 anticipated. Anything less is self-contradictory with the shipped parent.

**Design consequence flowing from the AC shape:** model create/update should be gated behind an explicit setup/bootstrap command, not run unconditionally on every sync. Probe-then-act via `modelNames` keeps ordinary sync runs free of model-write traffic and preserves the spirit of AC-015/AC-016 for the hot path.

---

### 2. Note-type management: what AnkiConnect gives, what the repo lacks

**API surface (external-docs).** `createModel` takes `modelName`, `inOrderFields`, optional `css`, optional `isCloze` (default false), and `cardTemplates[]` with `Name`/`Front`/`Back`. Cloze models are the same action with `isCloze: true` — no separate endpoint, so `imoogi-Basic` and `imoogi-Cloze` are two `createModel` calls. Idempotency is expressible with exactly the actions named in the topic: `modelNames` → absent ⇒ `createModel`; present ⇒ `updateModelStyling` (`params.model = {name, css}`) and `updateModelTemplates` (`params.model = {name, templates: {"<card name>": {Front, Back}}}`, documented as "Only specified cards and sides will be modified"). `modelStyling` / `modelTemplates` read current values back, enabling a no-op check before writing.

**Repo state (codebase-precedent, constraints-risks — negative findings, both exhaustive).** `grep -rn "storeMediaFile\|createModel\|modelNames\|updateModel" internal/ modules/` → **no matches**. The `AnkiConnector` interface at `internal/anki/ankiconnect/client.go:45-56` is exactly ten methods: `Handshake, DeckNames, CreateDeck, ModelFieldNames, AddNote, UpdateNoteFields, UpdateNoteTags, NotesInfo, ChangeDeck, DeleteNotes`. `*Client` and `planner/fake_client_test.go`'s `fakeClient` both satisfy it, so every added method breaks the double and every planner test until updated. `fake_client_test.go:17-50` logs calls per action (`addCalls`, `createDeckCalls`, …) so "no request fired" is assertable — new model/media actions need matching call logs to make an ownership-scoped AC testable.

**Note-type identity is hardcoded in at least six places** (constraints-risks' enumeration; codebase-precedent counted three and is superseded here):
- `internal/anki/orgdoc/orgdoc.go`: `NoteTypeBasic = "Basic"`, `NoteTypeCloze = "Cloze"`, plus a `switch` whose `default` returns `orgdoc: unrecognized note type %q` — an `imoogi-Basic` entry fails there before ever reaching AnkiConnect.
- `modules/24-anki.el:55` `("ANKI_NOTE_TYPE_ALL" . "Basic Cloze")` (property completion allowlist); `:69` and `:78` write the literals; `:187` auto-marks `Cloze`; `:189-192` compares `(string= type "Cloze")`.
- `modules/anki/imoogi-error.el:36` prose: "…change its ANKI_NOTE_TYPE to Basic."
- Prior SPEC: REQ-003/REQ-004 name them literally; `design.md:58` and `:90` type the schema as `string (Basic | Cloze)`.

**No CSS or embedded-asset precedent exists.** `find . -name "*.css" -not -path "./vendor/*"` → zero hits. The only styling in the tree is a ~6-line inline `<style>` in `const browserShell` at `internal/orgpreview/server.go:754-764` (light-only, hard-coded hex, `system-ui`). `grep -rniE "night_mode|nightMode|prefers-color-scheme|data-theme"` → **no output**. No `//go:embed` anywhere in `internal/` or `cmd/` — a template/CSS asset would be the first embedded asset; the existing precedent is Go string constants.

**Air-gap constraint (AGENTS.md §0).** "부팅 경로에… 네트워크 의존을 절대 추가하지 말 것." No CDN-loaded MathJax or CSS in generated templates. Anki's bundled MathJax is the only network-free option — which the external-docs lens confirms is sufficient.

**Naming.** `imoogi-Basic`/`imoogi-Cloze` is consistent with existing artifact naming (`.imoogi-registry.json`, `imoogi.json`, `imoogi-anki`, `imoogi-` Elisp prefix).

**Diagnostic codes are a closed set.** `internal/anki/protocol/protocol.go` declares 14 codes under the rule "A code with no [front-end] table entry is itself a defect… so cmd/imoogi and internal/planner never invent one ad hoc." New failure modes (model creation failed, media upload failed, image not found) each need a `protocol.Code*` const *and* a `modules/anki/imoogi-error.el` table entry.

---

### 3. Per-deck styling, `{{Deck}}`, and CSS class normalization

**`{{Deck}}`'s exact value is settled from Anki's own source, not the manual.** The manual (`docs.ankiweb.net/templates/fields.html`) lists `{{Deck}}` / `{{Subdeck}}` and notes special field names are case sensitive, but does not state the nested format. `pylib/anki/template.py` does:

```python
fields["Deck"] = self._col.decks.name(self._card.current_deck_id())
fields["Subdeck"] = DeckManager.basename(fields["Deck"])
```

So **`{{Deck}}` is the full deck path including `::` separators** (`Geography::Europe`), and `{{Subdeck}}` is the last component only.

**Normalization is mandatory, not optional.** Per MDN's CSS `<ident>` rules, a class may contain `A-Z a-z 0-9 - _` and non-ASCII ≥ U+00A0 (or escapes); it must not begin with an unescaped digit, or an unescaped hyphen followed by a digit; identifiers are case-sensitive. `:` and space are not valid unescaped. So `class="deck-{{Deck}}"` emits an unusable class unless imoogi defines a normalization rule.

**The user's real deck names make this concrete.** `modules/24-anki.el:53` records the *measured* deck name `"(PROGRAMMER)::(GO)"` and the measured failure that `org-global-properties`' `_ALL` mechanism breaks it into `("???" "::" "???")`. `imoogi-anki-set-deck`'s docstring states the `::` convention ("예: \"Geography::Europe\""). The normalization rule must therefore handle `::`, whitespace, **and parentheses**, and guard a leading digit. A reasonable rule to specify: lowercase → replace `::` with a single separator → replace every character outside `[a-z0-9_-]` with `-` → collapse runs → prefix-guard.

**The wrapper belongs in the card template, not in field content — and that removes a phantom risk.** Because `{{Deck}}` is a template special field expanded by Anki at review time, `<div class="deck-{{Deck}}">` must live in the `createModel` card template. Consequently it is (a) invisible to `hashing.Hash`, and (b) outside the "Anki may canonicalize stored field HTML" hazard entirely. The prior-SPEC-memory lens raised both points but applied the canonicalization worry to the wrapper; that worry does not attach if the wrapper is template-side, which is the only place `{{Deck}}` works at all.

**Deck changes already invalidate the hash** independently: the resolved deck is both hashed and recorded verbatim in the registry (`design.md:110`, D-6). And per C-8/REQ-010/D-10, deck is a *card* property moved via `changeDeck` — no note-type change is needed to restyle by deck, which is precisely the rationale recorded in `interview.md` ("카드 덱 이동 시 타입 변경 불필요(REQ-021 회피)").

**Night mode class names differ per client — a light+dark theme must carry both:**
- Desktop manual and AnkiMobile manual: `.card.nightMode { background-color: #555; }`, `.nightMode .myclass { color: yellow; }`.
- AnkiDroid wiki: `.card.night_mode { color: white; background-color: #303030; }`, with an explicit warning: "contrary to the other examples on this page there should *not* be a space between the `.card` and `.night_mode` classes."
- AnkiDroid also recommends `.card img {max-width: 100%; max-height: none;}` and `word-wrap: break-word;` (Android stopped auto-breaking long words since 4.4).
- Anki default `.card` styling for reference: `font-family: arial; font-size: 20px; text-align: center; color: black; background-color: white;`. Styling is shared across all cards of a note type.

---

### 4. LaTeX / MathJax

**Anki's side is unambiguous.** `docs.ankiweb.net/math.html`: "MathJax is supported out of the box on Anki 2.1+, AnkiMobile, and AnkiDroid 2.9+." Delimiters are `\(`…`\)` inline and `\[`…`\]` display; "$...$ or $$...$$ does not apply in Anki." Content must be TeX. Newlines inside must be `<br>`-style — a plain newline breaks rendering. And: "Anki has special logic for cloze deletions that might not work if you change the standard delimiters for MathJax equations" — i.e. emit exactly `\(`/`\[`, never a custom delimiter. AnkiDroid 2.15+ hides card content during MathJax render and supports `onUpdateHook`.

**Org-mode's `$` heuristic, verbatim from the manual** (`orgmode.org/manual/LaTeX-fragments.html`): "single '$' characters are only recognized as math delimiters if the enclosed text contains at most two line breaks, is directly attached to the '$' characters with no whitespace in between, and if the closing '$' is followed by whitespace or punctuation (but not a dash)." `$$…$$`, `\(…\)`, `\[…\]` and `\begin{...}` environments (the `\begin` on its own line, preceded only by whitespace) are also recognized. The manual itself recommends `\(...\)` over `$...$`.

**go-org does not simply "pass math through" — the measured picture is sharper (constraints-risks, read from the pinned v1.9.1 source in the module cache).**
- `org/inline.go:83-88` — `latexFragmentPairs = { "\\(": "\\)", "\\[": "\\]", "$$": "$$", "$": "$" }`. Math *is* parsed into a `LatexFragment` node.
- `org/html_writer.go:346-350` — `WriteLatexFragment` re-emits `OpeningPair + content + ClosingPair`, so the **delimiter bytes** survive. The interior comes from `parseRawInline`, so `_`/`^`/`*` inside math are protected from emphasis/sub/superscript.
- **But** `org/html_writer.go:326-334` — `WriteText` HTML-escapes raw text (`html.EscapeString(t.Content)` when `t.IsRaw`), so `a < b` inside math reaches Anki as `a &lt; b`. This is a MathJax-visible transform imoogi does not control and must decide about (unescape inside fragments, or document as a limitation).
- **And critically** `org/inline.go:211-224` — `parseLatexFragment` for `$` requires only `start+2 < len(input)` and then `strings.Index` for the next `$` anywhere in the remainder. No word-boundary, whitespace, or digit checks. So `costs $5 and $7` is already parsed as a `LatexFragment` with content `5 and $7`. Today the delimiters are preserved so the HTML *looks* unchanged; the moment imoogi rewrites `$..$` → `\(..\)` it becomes real MathJax. **The org-heuristic requirement is therefore a genuine behavioral divergence from the parser imoogi already uses, not a formatting nicety** — imoogi cannot delegate the heuristic to go-org and must apply org's rule itself when deciding which `$` pairs to convert.

**In-repo precedent for the transform's shape exists and is exactly one function away.** `internal/anki/orgdoc/orgdoc.go` `renderFragment` is a bare `org.New().Parse(strings.NewReader(s), "./").Write(org.NewHTMLWriter())`, but the same file already carries a post-render regex guard with a documented go-org-passthrough argument: `clozeMarkerPattern = regexp.MustCompile(`\{\{c\d+::`)` ("go-org passes X through verbatim, research.md §2"). The math and image transforms are the same shape of change and belong in the same place. There is **no** LaTeX/MathJax handling anywhere in the repo today.

**Prior SPEC treats go-org's gaps as accepted limitation, not defect.** `plan.md D-1` consequence: "`go-org` coverage of Org constructs is narrower than `ox-html`. Constructs that render poorly are a documented limitation for this SPEC, not a defect." D-1 also *rejected* rendering via Emacs `ox-html` (which would have supplied native math handling for free) because "it splits rendering across two languages, requires an Emacs process in the test loop, and makes the back end untestable in isolation." A CARD SPEC proposing to move math/image transforms into Elisp would be re-opening a settled decision.

---

### 5. Images and media

**AnkiConnect supports the whole flow.** `storeMediaFile` takes `filename` plus exactly one of `data` (base64), `path` (local absolute path), or `url` — so imoogi can upload a local org-relative image by path without base64-encoding it. Companions relevant to the orphan question: `getMediaFilesNames` (glob), `retrieveMediaFile` (returns base64, or `false` if nonexistent — usable as an "already uploaded?" probe), `deleteMediaFile`. Anki media lives in `collection.media` and is referenced from fields as plain `<img src="filename.jpg">`; the manual advises avoiding spaces and special characters — which supports a sanitized `basename + content-hash` naming scheme.

**`orgdoc.Render` cannot resolve relative paths today, and cannot host the upload.** Its signature is `Render(noteType, title, body string)` — no path parameter — and `renderFragment` hardcodes `"./"` as the parse base. `protocol.Entry.SourcePath` exists on the wire (written by Elisp as `(file-relative-name file root)` at `modules/anki/imoogi-scan.el:117`, serialized at `modules/anki/imoogi-process.el:45,52`) and `protocol.Config.SyncRoot` carries the root, so the org file's directory is reconstructible as `filepath.Dir(filepath.Join(req.Config.SyncRoot, entry.SourcePath))` — but it is never passed to `orgdoc`. Fixing this changes `Render`'s signature, and `plan.md D-1` documents `Render` as "a pure function of its inputs: no registry, no AnkiConnect client, no side effects." **Upload therefore cannot live inside `Render`.** The shape that satisfies both is two-phase: render → collect referenced local media → upload via `storeMediaFile` → rewrite `<img src>` in the rendered field text → *then* hash.

**Path-resolution precedent is strong and in-repo, but in the orgpreview module.** `internal/orgpreview/assets.go` `AssetResolver.Resolve(baseFile, target)` URL-unescapes, rejects non-empty and `file:` schemes, joins relative targets against `filepath.Dir(baseFile)`, canonicalizes via `filepath.Abs` + `EvalSymlinks` (tolerating `os.IsNotExist`), and confines results to allowed roots via `within()`. That confinement logic is directly reusable.

**Two different parsers are in play — do not conflate them.** `internal/orgpreview/parser.go:33` has a hand-rolled `linkRE = regexp.MustCompile(`\[\[file:([^\]\n]+)\](?:\[([^\]\n]*)\])?\]`)` plus `isImageTarget` at `:275` — that is orgpreview's own parser, matching only `file:`-prefixed links. The **anki** path uses go-org, whose behavior differs:
- `org/inline.go:68` — the image extension set is fixed: `(?i)^[.](png|gif|jpe?g|svg|tiff?|webp|x[bp]m|p[bgpn]m)$`. No `.avif`, `.bmp`, `.ico`. `.webm`/`.mp4` become `<video>`, needing a separate decision.
- `org/inline.go:402-416` `Kind()` — a link is `"image"` only when it has no description, **or** when the description itself is a `file:`/`http(s):` URL with an image extension. So **`[[file:diagram.png][My diagram]]` renders as `<a href="diagram.png">My diagram</a>`, not an `<img>`.** In the two-part image case the `src` is taken from the *description*, not the URL.
- `org/html_writer.go:391-421` `WriteRegularLink` — `url := html.EscapeString(l.URL)`, so a post-render regex rewrite of `<img src>` must match escaped forms (`&amp;` in a filename).

**Remote `https:` images stay as-is** (topic constraint; no lens found any reason to disturb this — Anki renders remote `<img src>` where network is available, and the air-gap rule concerns imoogi's own boot path, not user-authored card content).

---

### 6. Hashing, ownership, and the ordering trap

This is the highest-leverage correctness finding, and three lenses reached it independently.

**The hash has two uses.** `hashing.Hash(noteType, fields, deck, tags)` (`internal/anki/hashing/hashing.go`) is computed at `planner.go:243` as `hashing.Hash(entry.NoteType, fields, resolvedDeck, entry.Tags)` for the no-op/update decision — and recomputed at `planner.go:541` as `hashing.Hash(info.ModelName, fieldValuesToStrings(info.Fields), e.ResolvedDeck, info.Tags)` from a `notesInfo` response as the **ownership predicate** for orphan deletion, where a mismatch yields `delete_candidate_unowned`. Prior SPEC D-6/D-9 and `design.md:172` document this dual use.

**Ordering is a correctness requirement, not a preference.** The media upload + `<img src>` rewrite and the LaTeX delimiter transform **must** run before the hash at `planner.go:243`, and the field text imoogi *stores* must be byte-identical to what it *hashed*. If any transform runs after hashing, or only on the outbound request, every imoogi note fails the ownership predicate permanently and orphan deletion silently stops working — the fail-safe-but-silent mode the prior SPEC already flagged.

**Media→hash: answered, and answered for free.** D-6's stated invariant is that the hash covers "the material that determines the synchronized state of a note," is computed *post-render*, and that "hashing after rendering rather than before means a change in rendering behavior correctly invalidates every note (REQ-011 must not skip a note whose output would now differ)." Once `<img src>` carries a content-hash suffix, an image edit changes the rendered field HTML and therefore changes the hash — **no new hash input is needed**, and none should be added, because `hashing.Hash`'s own doc comment establishes that every input must be recoverable from a `notesInfo` response for the round-trip to work, and raw image bytes are not. `design.md §2.4` is the exact table where this would be documented (inputs: note type / rendered field values post-go-org / effective deck / sorted tag set; excluded: source path, heading position, `ANKI_NOTE_ID`).

**Corollary: the collision-avoidance mechanism and the "does an image edit trigger an update" answer are the same decision.** Without the content-hash suffix, image edits are invisible to REQ-010/REQ-011 and steady-state stays a no-op. With it, they are visible for free.

**One-time mass invalidation is unavoidable.** Any rendering change (MathJax delimiters, rewritten `<img src>`) invalidates every stored hash at once and produces a single collection-wide update of all existing imoogi notes. This is correct behavior per D-6 but must be stated in the SPEC as an expected first-run effect.

**Content-hash helper precedent exists** outside the anki tree: `internal/setup/setup.go:611 fileSHA256` (`sha256.Sum256` + `hex.EncodeToString`), `internal/artifact/artifact.go:45 VerifyFile` (streaming `sha256.New()` + `io.Copy`), `internal/setup/setup.go:532 bundleTreeSHA256`. `internal/anki/hashing` already imports `crypto/sha256` and `encoding/hex`.

---

### 7. Orphan media and registry tracking — out of scope, with grounds

Three independent grounds converge on **out of scope**, and the SPEC should say so explicitly rather than leave it open:

1. **Anki already owns it.** `docs.ankiweb.net/media.html` documents Tools ▸ Check Media as the user-facing tool that finds and trashes unused media. Important corollary: "Files beginning with an underscore (e.g., `_dog.jpg`) signal Anki to ignore them during media checks" — so imoogi must **not** `_`-prefix uploaded card images, or it would exempt them from the cleanup it is relying on.
2. **No deletion precedent exists.** D-9 makes deletion "identifier-based, content-confirmed, and never inferred," and `confirmAndDelete` is note-scoped with a three-part confirmation predicate that has **no media analogue**. There is no delete path in this codebase that is not ID-driven; inventing an inference-based one for media contradicts the project's stated deletion posture. `ankiconnect/client.go:353-356` even notes the absence of "a pattern-scoped delete action for this method to accidentally reach for (spec.md REQ-016)."
3. **The registry has no media concept.** `registry.Entry` is exactly `{note_id, source_path, note_type, resolved_deck, content_hash}` (`design.md:89-92`), described as holding only "everything recomputable," written wholesale by atomic temp-file + `Sync` + `os.Rename` once per run. Adding a media list is a schema change; it *would* be backward-compatible on read (JSON array of objects, missing fields decode as zero values, and `Load` treats only unmarshal *failure* as `*CorruptError` → `state_unreadable` → abort), so the door stays open for a later SPEC — but nothing today needs it.

---

### 8. Migration of the 5 existing test notes

**REQ-021 blocks in-place conversion by construction.** `planner.go:265-275`: `if regEntry.NoteType != entry.NoteType` → `ActionSkipped` + `CodeNoteTypeChangeUnsupported`, no field written, no card moved, registry untouched. And because note type is a hash input, `design.md:158` states a recorded-type mismatch "*always* implies a hash mismatch" — REQ-021 fires before REQ-010 for every `Basic` → `imoogi-Basic` edit. There is no in-repo escape hatch.

**The prior SPEC already decided the route, and it is manual delete + re-add.** `plan.md D-12`: "a user who changes an already-synced entry's note type must delete and re-add it manually for this MVP (spec.md §4). **Automating that migration — delete the old note, add a new one under the new type, update the registry and `ANKI_NOTE_ID` accordingly — is deferred to a later SPEC.**" `spec.md:223-224` repeats it. D-12's rejected alternative also records *why* an in-place field write is unsafe: "a `Basic` note's `Front`/`Back` shape does not match a freshly-rendered `Cloze` entry's single `Text` field… AnkiConnect's actual behavior on a field-shape mismatch is unverified."

**Mechanics of the manual route.** `spec.md:246` gives exactly two clean removal routes: remove the heading, or remove **both** `ANKI_NOTE_ID` and `ANKI_NOTE_TYPE` together. Both go through orphan-delete + re-add, which discards scheduling for those notes. AC-016's "leaves scheduling alone" governs the *update* path, so this is user cost, not an AC violation — but the SPEC must say it. With only 5 imoogi-created notes, the cost is trivial; the 88 stock `Basic` and 27 stock `Cloze` notes are untouched because they carry no imoogi identifiers.

**Helpful detail:** `addNote` normally rejects duplicates (first-field match within the same note type) unless `options.allowDuplicate` is set. Since `imoogi-Basic`/`imoogi-Cloze` are new types, re-added notes will **not** collide with the stock-type originals.

**Alternatively**, SPEC-ANKICARD-001 could become the "later SPEC" D-12 anticipates and automate delete+re-add. That is a scope decision the prior SPEC has already framed; either way, an in-place model change would require a new REQ superseding REQ-021 *and* an AnkiConnect action not present in the current client (whose existence no lens verified).

---

### 9. Touch surfaces and contract costs

- **`ankiconnect.AnkiConnector`** — adding `ModelNames`, `CreateModel`, `UpdateModelStyling`, `UpdateModelTemplates`, `StoreMediaFile` breaks `fakeClient` and every planner test until updated; each needs a call-log slice to keep negative assertions ("no model write during sync") testable. All calls funnel through the single `func (c *Client) call(ctx, action string, params any)` chokepoint; error tiers are `TransportError`/`ProtocolError`/`APIError`; `const Version = 6`.
- **`protocol.Version = 1`** — `cmd/imoogi-anki/main.go:79` hard-rejects any mismatch with `binary_incompatible`; `modules/anki/imoogi-process.el:19` pins the Elisp constant. Any new `Request`/`Config` field (per-deck styling toggle, media policy flag) requires a coordinated bump on both sides. The wire rule is documented: "No field carries omitempty."
- **`orgdoc.Render` signature** must grow a base-path parameter, and the note-type `switch`'s `default` must learn the imoogi-prefixed names.
- **`resolveFields`** (`planner.go:130-183`) lowercases both sides against a cached `ModelFieldNames` and errors with `note_field_missing` when unmatched. Its doc comment records the measured hazard — "`updateNoteFields` accepts an unknown field name, answers error:null, and changes nothing" — and `fake_client_test.go`'s `modelFields` comment names the user's exact situation: "a customized 'Basic' carrying lowercase front/back is the observed case this exists to reproduce." For imoogi-owned models this layer is dead weight but harmless; the risk is the reverse — new code must not bypass it and thereby break the stock-`Basic` case.
- **Error table** — `modules/anki/imoogi-error.el` needs an entry for each new `protocol.Code*`.
- **Elisp note-type literals** — the five sites in `modules/24-anki.el` plus the `imoogi-error.el:36` prose.
- **Build/test** — `make build-anki`, `make test` (`test-elisp test-go test-shell`), `make lint` = `go vet`. Go tests: pure-function tests in `internal/anki/orgdoc/orgdoc_test.go` with rationale-bearing names (e.g. `TestRender_Cloze_PreservesMarkerByteForByte`); Elisp tests flat in `tests/*.el`; contract test at `tests/org-preview-contract_test.go`. `go.mod`: `go 1.26`, `github.com/niklasfasching/go-org v1.9.1`; `go.work` exists to keep Go out of vendor mode.
- **Layout convention** — `internal/anki/*` uses subpackage-per-concern, so a new `internal/anki/<something>` (e.g. `model`, `media`) matches. (`internal/orgpreview` has a confusing dual layout — flat files holding real logic plus thin delegating subpackages — that is not the anki-side convention.)

---

### 10. Process memory worth carrying forward

SPEC-ANKI-001 was re-tiered M→L during audit ("File count alone (16 non-test deliverables) exceeded Tier M's 5-15 band") and needed **five** plan-audit iterations (0.70 → 0.82 → 0.75 → 0.84 → PASS 0.857). Recurring blocking-defect themes: requirement pairs not disjoint *in requirement text* (D18/D26/D-12), registry state written once and never refreshed (D27), and design-narrative ordering mistaken for a behavioral guard — the last is directly relevant, since this SPEC's central ordering claim (transform-before-hash) must be expressed as a *testable behavioral guard*, not as prose sequencing in the design narrative. Also note the Tier L acceptance-criterion budget was fully consumed at 25/25, so late fixes had to extend existing ACs or live in `acceptance.md §D.7`.

---

### contradictions

Every cross-lens conflict found, with each side attributed. None are silently reconciled.

**C1 — AC identifier for the "no note-type-template/styling endpoint" criterion. RESOLVED by direct read; both lenses are recorded here so the correction is auditable.**
- *constraints-risks* calls it **AC-016**, citing `acceptance.md:102`.
- *prior-SPEC-memory* calls it **AC-018**, citing the same `acceptance.md:102`.
- **Resolution:** `sed -n '99,102p' acceptance.md` shows the heading "**AC-016 — a changed body updates in place and leaves scheduling alone**" immediately above line 102. **constraints-risks is correct; prior-SPEC-memory's AC-018 label is wrong.** The criterion is AC-016. (constraints-risks' AC-015 label for the "second run, zero mutations" criterion at lines 95-97 is also correct.)

**C2 — Does go-org mangle `_` / `^` inside LaTeX fragments? UNRESOLVED, but primary source favors "no."**
- *external-docs* flags go-org issue #12 ("Subscript and superscript in LaTeX fragments"): input `\(\sum_{i=1}^n a_n\)` had `_`/`^` rendered as `<sub>`/`<sup>`, "making it impossible for MathJax to process." Issue closed; fix commit `76b157b8` came from a search snippet and **was not verified on the commit page**. The lens itself recommends checking the vendored version.
- *constraints-risks* read the pinned **v1.9.1** source directly: `parseLatexFragment` parses the interior with `parseRawInline`, producing raw `Text` with no emphasis/sub/superscript, and `WriteLatexFragment` re-emits it between the original pairs. On that reading `_`/`^`/`*` inside math are protected in the version this repo actually uses.
- **Neither lens verified the fix version.** Primary-source reading of the pinned version outweighs an unresolved issue tracker, but the SPEC should carry a regression test (`\(\sum_{i=1}^n a_n\)` in → sub/sup-free out) rather than assert the behavior.

**C3 — "go-org passes all four math syntaxes through verbatim." The topic premise and codebase-precedent state it; constraints-risks materially qualifies it.**
- *Topic premise + codebase-precedent:* the measured fact is that go-org's HTML writer passes all four syntaxes through verbatim, so the delimiter transform is imoogi's job.
- *constraints-risks (from v1.9.1 source):* go-org does **not** pass math through untouched. It *parses* math into a `LatexFragment` node and re-emits the delimiter bytes — so the delimiters survive, but (a) `WriteText` HTML-escapes the interior, so `a < b` becomes `a &lt; b` in the output, and (b) `parseLatexFragment` for `$` is just `strings.Index` for the next `$`, with none of org-mode's whitespace/punctuation/line-break checks, so `costs $5 and $7` is *already* being parsed as a fragment with content `5 and $7`.
- **Both are true at different levels.** "Verbatim" holds for the delimiter bytes and is why the current output looks unchanged; it does **not** hold for the interior text (entity escaping) or for parse semantics. The practical consequence is the sharper one: imoogi cannot delegate org's `$` heuristic to go-org, because go-org has already made the opposite decision on `$5`, and imoogi's rewrite would turn that silent misparse into visible broken MathJax.

**C4 — Is the `deck-{{Deck}}` wrapper exposed to Anki's field-HTML canonicalization? Intra-lens tension in prior-SPEC-memory, resolved by where the wrapper must live.**
- *prior-SPEC-memory* states both that "`{{Deck}}` is rendered by *Anki at review time*, not by imoogi, so it is invisible to `hashing.Hash`" **and** that "a `<div>` wrapper in the field content is precisely the kind of markup Anki may canonicalize."
- These cannot both apply to the same wrapper. `{{Deck}}` only expands inside a **card template**, so `<div class="deck-{{Deck}}">` must be template-side (delivered via `createModel`), which places it outside both the hash and the field-canonicalization risk. The canonicalization worry remains real for *other* markup imoogi puts in field content (rewritten `<img>`, MathJax delimiters) — see C5.

**C5 — Does Anki normalize stored field HTML? NONE of the four lenses could determine this, and all flagged it as the top unknown.**
- *constraints-risks:* "the single highest-risk unknown"; if Anki rewrites whitespace, attribute order, or entity form on save, the recomputed-hash ownership predicate in `confirmAndDelete` breaks for image/MathJax-bearing fields **regardless of transform ordering**. Needs an empirical `addNote` → `notesInfo` → byte-compare round-trip.
- *prior-SPEC-memory:* the prior SPEC's own run-phase report already records this as residual risk: "a real Anki may normalize stored field content (HTML rewriting, entity or whitespace canonicalization) in ways no stub reproduces, and the failure mode remains fail-safe-but-silent — deletion simply stops happening, reported only as `delete_candidate_unowned`."
- *external-docs:* no official documentation addresses it.
- This is not a contradiction between lenses; it is a unanimous gap, recorded here because it is the one unknown that could invalidate the design.

**C6 — AnkiConnect documentation provenance. Flagged by external-docs itself, no counter-lens.**
- The canonical `git.sr.ht/~foosoft/anki-connect` returned HTTP 502 and `foosoft.net/projects/anki-connect/` returned 404 during the session. All AnkiConnect JSON samples were read from a **GitHub fork mirror** (`iamqiz/anki-connect-fork`). Wording matches SPEC-ANKI-001's own earlier direct fetch of the canonical source, but the fork could lag upstream. **Medium confidence** — re-verify against the canonical README before pinning request shapes in the SPEC.

**No other cross-lens contradictions were found.** The lenses agree on: media→hash via content-hash-suffixed filenames; orphan-media cleanup out of scope; transform-before-hash ordering; `Render` needing a base path and a two-phase upload; REQ-021 blocking in-place migration with D-12's delete+re-add as the settled route; and the REQ-016/§4 supersession requirement.

---

### open questions — verdicts

| Question | Verdict |
|---|---|
| Does image content affect `hashing.Hash`? | **Yes, and for free.** A content-hash suffix in the stored media filename changes the rendered `<img src>`, hence the field HTML, hence the hash. Add **no** new hash input — every input must round-trip through `notesInfo`, and image bytes cannot. |
| Must the registry track uploaded media (orphan cleanup)? | **No — out of scope, say so explicitly.** Anki's Check Media owns unused-media cleanup; D-9's deletion posture has no media analogue; the registry schema is deliberately "everything recomputable." Do **not** `_`-prefix uploads (that would exempt them from Check Media). Schema is forward-compatible if a later SPEC wants it. |
| Idempotent model create/update | **`modelNames` → absent ⇒ `createModel`; present ⇒ `updateModelStyling` + `updateModelTemplates`.** `modelStyling`/`modelTemplates` allow a no-op read-back check. **Gate behind an explicit setup command, not per-run**, so ordinary sync runs stay clean of model-write traffic. **Unknown:** whether `createModel` on an existing name errors, no-ops, or overwrites — the README does not say; probe-then-act is the safe pattern regardless. Also unknown: whether `updateModelTemplates` with an absent template name adds or errors; whether `storeMediaFile` overwrites by default (a `deleteExisting` param exists in some versions but was not in the fetched README). |
| Migration of the 5 existing test notes | **Delete + re-add**, per REQ-021 and `plan.md D-12`. Routes: remove the heading, or remove **both** `ANKI_NOTE_ID` and `ANKI_NOTE_TYPE`. Scheduling for those 5 notes is lost — user cost, not an AC violation, but must be stated. `addNote` will not dedupe against the stock-type originals since the model differs. Automating this would make SPEC-ANKICARD-001 the "later SPEC" D-12 anticipates. |
| Night-mode CSS hooks | **Ship both.** `.card.nightMode` (desktop + AnkiMobile) **and** `.card.night_mode` (AnkiDroid, explicitly **no space** between the classes). Add `.card img {max-width:100%; max-height:none;}` and `word-wrap: break-word;` per AnkiDroid guidance. |
| MathJax on AnkiDroid / iOS | **Available out of the box** — "Anki 2.1+, AnkiMobile, and AnkiDroid 2.9+." Emit exactly `\(`/`\)` and `\[`/`\]`; custom delimiters break Anki's cloze logic. Newlines inside math must be `<br>`. No CDN needed, satisfying the air-gap rule. *Unknown:* bundled MathJax version per client, and offline macro/`\require` availability beyond "supported." |
| `{{Deck}}` value format for `::`-nested decks | **Full deck path including `::`** (Anki source: `fields["Deck"] = self._col.decks.name(...)`; `{{Subdeck}}` is the basename). CSS ident rules make normalization **mandatory**; the user's real deck `"(PROGRAMMER)::(GO)"` proves parentheses and `::` both occur. The wrapper lives in the **card template**, not field content. |

---

### evidence index

**Prior SPEC (out of repo), `/Users/jay/workspace/imoogi-org-anki/.moai/specs/SPEC-ANKI-001/`**
`spec.md:58` (glossary: built-in Basic/Cloze only) · `:177` REQ-016 · `:196` REQ-021 · `:213-219` §4 custom note types out of scope · `:223-224` note-type-change exclusion + "manual step for this MVP (plan.md D-12)" · `:246` two clean removal routes · `:258` §4 collection-ownership guarantees.
`plan.md:23-31` D-1 (go-org; `ox-html` rejected; narrower coverage is a documented limitation) · `:140-150` D-6 (hash inputs; post-render rationale; second use as ownership predicate) · `:224-234` D-12 (manual migration; field-shape mismatch unverified; automation deferred).
`design.md:58`,`:90` (`string (Basic | Cloze)`) · `:89-92` registry field table · `:96-110` §2.4 hash inputs · `:158` recorded-type mismatch ⇒ hash mismatch · `:172` step 11 orphan confirmation.
`acceptance.md:95-97` AC-015 · `:99-102` AC-016 (verified by direct read).
`research.md` §2 (go-org 80/20; construct list contains no math/image/media entry), §3.2 (`notesInfo` returns noteId/modelName/tags/fields/mod/cards, **no deck**), §3.4 (`addNote` `allowDuplicate`).
`progress.md` — audit 0.70→0.82→0.75→0.84→PASS 0.857; 6 milestones, 25/25 AC; coverage internal_planner 89.2 / internal_orgdoc 100.0 / internal_hashing 100.0; residual risk on Anki field normalization.
Negative: `grep -rn -i "latex|mathjax|math|image|media|styling|css|createModel|template|night"` over all of `/Users/jay/workspace/imoogi-org-anki/.moai/reports/` → **zero substantive hits**.

**This repo, `/Users/jay/workspace/imoogi-emacs/`**
`internal/anki/ankiconnect/client.go:45-56` (10-method interface), `:353-356` (no pattern-scoped delete, REQ-016).
`internal/anki/orgdoc/orgdoc.go` — `Render(noteType, title, body string)`, `org.New().Parse(strings.NewReader(s), "./")`, `clozeMarkerPattern`, `NoteTypeBasic`/`NoteTypeCloze`, `unrecognized note type %q`.
`internal/anki/planner/planner.go:130-183` `resolveFields` · `:243` hash for update decision · `:265-275` REQ-021 skip branch · `:541` recomputed ownership hash.
`internal/anki/hashing/hashing.go` (dual-use doc comment; format-stability warning; `\x00`/`\x01` framing + `sha256.Sum256`).
`internal/anki/registry/registry.go` (5-field `Entry`; `*CorruptError` on unmarshal failure; atomic temp+`Sync`+`Rename`).
`internal/anki/protocol/protocol.go` (`Version = 1`; 14 diagnostic codes; "No field carries omitempty"; `Entry.SourcePath`; `Config.SyncRoot`).
`internal/anki/planner/fake_client_test.go:17-50` (per-action call logs; lowercase-front/back comment).
`cmd/imoogi-anki/main.go:79` (`binary_incompatible`).
`internal/orgpreview/assets.go` (`Resolve`/`canonical`/`within`) · `internal/orgpreview/parser.go:33` `linkRE`, `:275` `isImageTarget` · `internal/orgpreview/server.go:754-764` (`browserShell`, the repo's only `<style>`).
`modules/24-anki.el:52-55` (measured `"(PROGRAMMER)::(GO)"` breakage; `ANKI_NOTE_TYPE_ALL`), `:69`,`:78`,`:128`,`:187`,`:189-192` · `modules/anki/imoogi-error.el:36`,`:43` · `modules/anki/imoogi-process.el:19`,`:45`,`:52` · `modules/anki/imoogi-scan.el:117` · `modules/anki/imoogi.el:64`,`:93`.
`Makefile:99-105`,`:118`,`:130` · `AGENTS.md` §0 air-gap · `go.mod` (go 1.26, go-org v1.9.1) · `go.work`.
`.moai/specs/SPEC-ANKICARD-001/interview.md` (23 lines; 4 confirmed decisions + 6 open questions — the only artifact for this SPEC; no spec.md/plan.md/research.md yet). `.moai/decisions/` holds only `adr-template.md` and `lsp-client-choice.md`; `grep -rn -i "anki" CLAUDE.md .moai/project/ .moai/docs/` → **no matches**.
Negatives: `find . -name "*.css" -not -path "./vendor/*"` → **none**. `grep -rniE "night_mode|nightMode|prefers-color-scheme|data-theme"` → **none**. `grep -rn "embed" --include='*.go' internal cmd` (non-test) → **none**. `grep -rn "storeMediaFile\|createModel\|modelNames\|updateModel" internal/ modules/` → **none**.

**go-org v1.9.1** (module cache): `org/inline.go:68` image ext regex, `:69` video · `:83-88` `latexFragmentPairs` · `:211-224` `parseLatexFragment` · `:402-416` `Kind()` · `org/html_writer.go:326-334` `WriteText` escaping · `:346-350` `WriteLatexFragment` · `:391-421` `WriteRegularLink`.

**External**: AnkiConnect README (fork mirror `iamqiz/anki-connect-fork` — see C6) for `createModel`/`updateModelTemplates`/`updateModelStyling`/`storeMediaFile`/`retrieveMediaFile`/`getMediaFilesNames`/`deleteMediaFile`/`modelNames`/`modelFieldNames`/`modelStyling`/`modelTemplates` · `docs.ankiweb.net/templates/fields.html` · `github.com/ankitects/anki` `pylib/anki/template.py` · `docs.ankiweb.net/math.html` · `docs.ankiweb.net/templates/styling.html` · `docs.ankimobile.net/night-mode.html` · AnkiDroid wiki `Advanced-formatting` · `docs.ankiweb.net/media.html` · `orgmode.org/manual/LaTeX-fragments.html` · `github.com/niklasfasching/go-org/issues/12` · MDN `Web/CSS/ident`.
---

### 14. Post-research measurements

Two measurements taken **after** the four lenses reported, during the revision-2
annotation cycle. Both closed items the synthesis above had left as unknowns;
the sections above are preserved unedited so the original uncertainty stays
auditable.

#### 14.1 R1 — does Anki canonicalize stored field HTML on save?

`C5` above records this as the one unanimous gap: all four lenses flagged it, and
if Anki rewrote whitespace, attribute order, or entity form on save, the
recomputed-hash ownership predicate on the orphan-deletion path would break for
every image- or MathJax-bearing field **regardless of transform ordering**.

**Method.** A probe note was added to the live user collection through the same
AnkiConnect surface imoogi uses, carrying, in one field, every canonicalization
trigger the lenses had named:

- attributes in a non-canonical order,
- `&amp;` and `&lt;` character entities,
- runs of multiple consecutive spaces,
- a self-closing `<img … />` element,
- an inline MathJax fragment `\(..\)`,
- a display MathJax fragment `\[..\]`.

The note's fields were then read back through `notesInfo` and compared with the
submitted text byte-for-byte. The probe note was deleted afterwards.

**Result: byte-identical.** Anki returned the stored field HTML unchanged on
every one of the six triggers — attribute order preserved, entities preserved in
their submitted form, multiple spaces preserved, the self-closing form preserved,
and both MathJax delimiter forms preserved.

**Consequence.** The recomputed ownership predicate is safe, and revision 1's
R1 fallback design was **deleted rather than deferred**. What remains is the
ordering invariant of `design.md` § 2.2, which is now the only thing standing
between the implementation and a silent deletion stop — which is why it is
carried as a byte-equality assertion (`acceptance.md` AC-C-016a) rather than as
design prose.

**Residual.** A future Anki release could introduce normalization. If it is ever
observed, the predicate narrows to note-type-only and this probe is re-run.
Recorded as `plan.md` § H R1.

#### 14.2 go-org passthrough measurement (pinned v1.9.1)

Read from the pinned module source rather than inferred, closing the `C3`
qualification.

**Math — all four syntaxes, delimiter bytes preserved.** The HTML writer's
LaTeX-fragment path re-emits `OpeningPair + content + ClosingPair`, so each of
the four Org syntaxes reaches the writer's output with its delimiters verbatim:

| Org input | go-org HTML output |
|---|---|
| `$E = mc^2$` | `$E = mc^2$` |
| `$$E = mc^2$$` | `$$E = mc^2$$` |
| `\(E = mc^2\)` | `\(E = mc^2\)` |
| `\[E = mc^2\]` | `\[E = mc^2\]` |

This is what makes the delimiter conversion imoogi's own job (REQ-C-010): nothing
upstream will do it.

**Two qualifications on "verbatim", both material.**

1. *Interior escaping.* The writer HTML-escapes raw text, so `a < b` inside a
   fragment reaches Anki as `a &lt; b`. MathJax reads DOM text nodes where
   `&lt;` has already been decoded, and unescaping at the source would produce
   invalid HTML — so the escaping is kept and verified on a real card
   (`plan.md` § H R11).
2. *Parse semantics.* The `$` fragment parser requires only that a further `$`
   exist somewhere in the remainder — no word-boundary, whitespace, or digit
   check — so `costs $5 and $7` is **already** parsed as a fragment today. The
   delimiters currently survive, so nothing looks wrong; converting `$…$` to
   `\(…\)` would make that silent misparse visible as broken MathJax. imoogi
   therefore applies Org's own inline-math heuristic itself (REQ-C-010,
   `D-C-12`).

**Images — `src` emitted as authored.** A one-part link whose target carries an
extension in go-org's image set is emitted as `<img src="<path as authored>">`,
with the URL HTML-escaped. No path resolution, no rewriting, no upload — which is
what leaves the media pass with a well-defined post-render surface to operate on
(REQ-C-012).

**Two-part links — emitted as anchors.** A link is classified as an image only
when it has no description, or when the description is itself an image URL.
Therefore:

| Org input | go-org HTML output |
|---|---|
| `[[file:diagram.png]]` | `<img src="diagram.png">` |
| `[[file:diagram.png][My diagram]]` | `<a href="diagram.png">My diagram</a>` |

This is the measurement REQ-C-012 exists to answer: imoogi rewrites the second
form to `<img src="…" alt="My diagram">` in the media pass, preserving the
description as accessibility text.

**Image extension set, verbatim from the pinned source:** `png`, `gif`, `jpg`,
`jpeg`, `svg`, `tif`, `tiff`, `webp`, `xbm`, `xpm`, `pbm`, `pgm`, `ppm`, `pnm`
(case-insensitive). `.avif`, `.bmp`, and `.ico` are absent — which is why
REQ-C-012 adopts this set unchanged rather than extending it (`D-C-11`).
