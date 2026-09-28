## Evaluation Report
SPEC: SPEC-ANKICARD-001
Audited HEAD: b799850 (run phase closed at 264bbf6; post-run drift included, see §0)
Auditor: sync-auditor (ankicard-sync-audit-2), 2026-09-08
Overall Verdict: **FAIL** — must-pass firewall on Functionality (F1, blocking). Weighted harmonic score 0.68.

### Dimension Scores
| Dimension | Score | Verdict | Evidence |
|-----------|-------|---------|----------|
| Functionality (40%) | 50/100 | FAIL (must-pass) | 22 AC PASS · AC-C-004/REQ-C-006 PARTIAL (F1) · AC-C-022c UNVERIFIED (AnkiConnect unreachable, `curl` exit 7) |
| Security (25%) | 95/100 | PASS | Stock-type write fence holds by construction + wire test; dry-run zero-write; media confinement (`EvalSymlinks`+`Rel`); no Critical/High |
| Craft (20%) | 90/100 | PASS | Coverage above every plan §G floor; race/vet/gofmt clean; 1 staticcheck SA4000 in a test (F2) |
| Consistency (15%) | 85/100 | PASS | Code conventions followed; doc-layer key drift `ankiconnect_url` vs wire `anki_connect_url` (F3) |

Unweighted harmonic mean 0.75; weighted harmonic mean 0.68. The must-pass firewall forces FAIL on Functionality regardless of the score.

Cross-model second opinion (`mcp__moai__audit_multi`): codex = request-changes/BLOCK, independently naming F1 as HIGH and adding two MEDIUM media-upload items (verified by me, recorded as F4/F5, optional); glm = inconclusive (no API key, fail-open). `disagreement_flag: false`.

---

### §0 Scope drift since run-phase close (264bbf6 → b799850)

```
$ git diff --stat 264bbf6..HEAD -- internal/anki cmd/imoogi-anki modules/24-anki.el modules/anki 'tests/anki-*'
 modules/24-anki.el                 |  25 +++-
 modules/anki/imoogi-target-scan.el | 265 +++++++++++++++++++++++++++++++++++
 modules/anki/imoogi-targets.el     | 274 ++++++++++++++++++++++++++++++++++++
 modules/anki/imoogi.el             | 190 +++++++++++++++----------
 tests/anki-target-scan-test.el     | 278 +++++++++++++++++++++++++++++++++++++
 tests/anki-target-sync-test.el     | 105 ++++++++++++++
 tests/anki-targets-test.el         | 211 ++++++++++++++++++++++++++++
 7 files changed, 1270 insertions(+), 78 deletions(-)
```

User commit 89e94be touched SPEC scope: `imoogi-sync` rewritten for multi-target (host-registered folders/files), the transient gained a fourth column, and the `("a" . imoogi-anki-transient)` binding moved inside `with-eval-after-load 'imoogi-transient`. No Go file changed. The install/migrate tail (`imoogi-setup.el`, `imoogi-process.el`) is untouched. All 94 anki/transient ERT tests pass at HEAD, including `imoogi-transient-anki-is-registered-on-master`. The multi-target sync behaviour itself is user-authored and outside this SPEC's AC; it was exercised, not judged. Post-run files carry no `"Basic"`/`"Cloze"` literal and no model-write call (grep, zero matches).

Working tree in SPEC scope is clean (`git status --short` on SPEC paths: empty). The user's unrelated WIP (org-preview, org-border, README, boot.el) did not break any command I ran; it accounts for the single Elisp failure (environment note, §Craft).

---

### §1 Functionality (40%) — 50/100 — FAIL (must-pass)

**Claim.** 22 of 24 AC verified PASS at b799850; AC-C-004 / REQ-C-006 PARTIAL (normalizer correct and tested, never applied at review time); AC-C-022c UNVERIFIED.

**Evidence.**
```
$ go test -count=1 ./internal/anki/... ./cmd/imoogi-anki/... -cover
ok  	github.com/karohani/imoogi-emacs/internal/anki/ankiconnect	0.470s	coverage: 86.2% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/hashing	1.154s	coverage: 100.0% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/media	0.845s	coverage: 91.3% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/model	1.533s	coverage: 98.3% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/orgdoc	1.863s	coverage: 100.0% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/planner	3.331s	coverage: 91.5% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/protocol	2.955s	coverage: [no statements]
ok  	github.com/karohani/imoogi-emacs/internal/anki/registry	2.445s	coverage: 87.8% of statements
ok  	github.com/karohani/imoogi-emacs/cmd/imoogi-anki	2.704s	coverage: 86.9% of statements
EXIT=0
```
```
$ make test-elisp   (exit 2)
Ran 247 tests, 246 results as expected, 1 unexpected (2026-09-08 23:39:22+0900, 11.961588 sec)
1 unexpected results:
   FAILED  imoogi-org-preview-browser-navigation-moves-point-and-suppresses-echo
```
anki/install/migrate/notetype/error/writeback/process/setup/target/transient tests: `94 passed`, 0 FAILED, 0 skipped (grep over the log). The one failure is the user's untracked org-preview WIP (`tests/org-preview-test.el:82`), identical to the M1–M6 baseline.

AC-level cross-check (test → AC), all observed PASS in the run above:
- AC-C-001/002/007 install probe-then-act, idempotence, ownership notice: `internal/anki/model/install_test.go` (model 98.3%).
- AC-C-003a: `TestRun_AC_C_003a_SyncRunOfAnyShapeRecordsZeroModelWrites` (6 subtests). AC-C-003b / AC-C-022b: `cmd/imoogi-anki/install_test.go:241 TestInstallModelsWritesNoForeignModelOnTheWire`; interface enumeration `ankiconnect/model_test.go:328`.
- AC-C-004: `model/deckclass_test.go:14` (15 rows incl. the 9 AC rows) + stored-field grep at `:136`; Elisp mirror `imoogi-notetype-test-deck-class-mirror-table`. **See F1 — the assertion covers `DeckClass()`, which production never calls.**
- AC-C-005/006a-c: `model/css_test.go` (custom properties, night selectors, air-gap grep, concat). AC-C-006d: `imoogi-install-test-request-carries-exactly-three-keys`, `…user-css-is-the-file-contents-byte-for-byte`, `…absent-stylesheet-is-the-empty-string`, `…sync-request-gains-no-field`; no `.css` disk read in `model`/`ankiconnect` (grep: only the `//go:embed` line).
- AC-C-008/009/023/024: `orgdoc/math_test.go` (orgdoc 100%).
- AC-C-010–015, 022d: `media/media_test.go` (escape above root `:89`, upstairs-inside-root `:444`, remote untouched `:300`, video/audio pass-through `:272`, no `_` prefix `:356`, dedupe `:184`) + `planner/media_test.go` gating.
- AC-C-016a: `planner/media_test.go:80 TestRun_Media_StoredBytesEqualHashedBytes`; AC-C-016b: `:262 TestHashInputSetIsUnchangedByTheMediaPass`; AC-C-016c: `TestRun_Media_OwnershipPredicateSurvivesAMediaEdit`. (H4 below.)
- AC-C-017: `planner_test.go:216 TestRun_NoOp_HashUnchanged_IssuesNoRequests` + media no-op test.
- AC-C-018/019/020: `planner/migrate_test.go` (16 tests; `:195 …AddsUnderTheCounterpartBeforeDeletingTheOriginal`, `:502 …SkipsBeforeAddingAnything`), `cmd/imoogi-anki/migrate_test.go`; Elisp `imoogi-migrate-test-declined-issues-no-writing-request`, `…confirmed-writes-back-both-properties`, `…prompt-names-the-count-and-the-loss`.
- AC-C-021: error-table pairing verified mechanically (§2); literal sites `imoogi-notetype-test-*`.
- AC-C-022a: `imoogi-notetype-test-stock-headings-still-scan-as-targets`; planner stock-type characterization tests unchanged.
- AC-C-022c: UNVERIFIED —
```
$ curl -s -m 2 -w '\nHTTP=%{http_code}\n' -d '{"action":"version","version":6}' http://127.0.0.1:8765
HTTP=000
CURL_EXIT=7
```

**Baseline-attribution.** All commands run against HEAD `b799850`, clean SPEC-scope tree, 2026-09-08. Coverage figures match the orchestrator's 264bbf6 logs at `.moai/state/verify/fe98f407/m6/go-test.log` byte-for-byte (no Go change since), and were re-measured with `-count=1`, not consumed from the snapshot.

**Gaps.** AC-C-022c (live collection: stock `Basic`/`Cloze` byte-unchanged, `88−N` / `27−M` counts) not observed. No review-time rendering of a card was observed (no Anki instance) — F1 is established from the templates and the callers, not from a rendered card.

**Residual-risk.** Anki's actual `{{Deck}}` expansion inside a `class` attribute (HTML-escaping of `(`/`)`/`:`) was not measured; it does not change F1's conclusion in either direction. First-run mass invalidation (REQ-C-016.2) is specified behaviour and will re-upload every math/image note once.

#### Hypotheses
- **H1 — DEFECT (blocking, severity warning).** `NormalizeDeckClass`/`DeckClass` (`internal/anki/model/deckclass.go:39,99`) have zero non-test callers (grep). All four templates emit `<div class="deck-{{Deck}}">` (`assets/basic-front.html:1`, `basic-back.html:1`, `cloze-front.html:1`, `cloze-back.html:1`). There is no template-side script. `base.css:52-58` acknowledges it: "the runtime class carries whatever punctuation the deck name carries" and switches the base rule to `[class^="deck-"]`. So the base theme renders, but the user contract in `design.md` §3.4 (`.deck-programmer-go { … }`) matches nothing: the review-time class for `(PROGRAMMER)::(GO)` is `deck-(PROGRAMMER)::(GO)`, and `2026 Review` splits into two tokens `deck-2026` and `Review`. REQ-C-006.1/.2, the §2 glossary ("consumed only as the tail of the `deck-` class the card template emits"), and REQ-C-009's premise ("every per-deck rule originating in the user stylesheet") are not honoured at the only surface where they matter. Root cause is a design-level gap faithfully implemented: `design.md` §3.1 draws the raw wrapper while §3.2 says normalization is "mandatory, not cosmetic". AC-C-004 verifies the wrong surface (the Go function), which is why every test is green.
- **H2 — CONFIRMED, docs only.** Wire key is `anki_connect_url` on both sides (`protocol.go:135`, `imoogi-process.el:78`). `protocol/install_test.go:71` asserts `ankiconnect_url` is absent; `tests/anki-install-test.el:55` asserts the three-key set. No test pins the wrong spelling. The wrong spelling survives only in `spec.md` §2/§7, `design.md:335,618`, `spec-compact.md`, and `acceptance.md:132` (AC-C-006d's own text). → F3.
- **H3 — GAP stands.** AnkiConnect unreachable (above). No read-only action was attempted.
- **H4 — CONFIRMED PASS.** `planner.go:270-278`: `media.Rewrite` runs before `hashing.Hash`; the MathJax transform lives inside `orgdoc.renderFragment` (`orgdoc.go:110`), so it precedes the hash by construction. `TestRun_Media_StoredBytesEqualHashedBytes` first asserts the transforms reached the dispatched text (`src="<stored>"` present, `src="diagram.png"` absent, `\(` present) — guarding against a vacuous pass — then recomputes `hashing.Hash` from the fake client's field map and compares to the registry. That asserts ordering, not presence. The migrate path (`migrate.go:188-206`) uses the same render → media → hash order.

---

### §2 Security (25%) — 95/100 — PASS

**Claim.** No Critical/High finding. The stock-type fence (REQ/AC-C-003, C-004, C-024.3/.4) holds on every code path; inputs at the trust boundary are validated.

**Evidence.**
```
$ grep -rn 'CreateModel\|UpdateModelStyling\|UpdateModelTemplates' --include='*.go' internal/anki cmd/imoogi-anki | grep -v _test.go
  … interface + client method definitions (client.go:62-64, 411, 441, 470) …
internal/anki/model/install.go:100:		if err := client.CreateModel(ctx, spec.Name, spec.InOrderFields, css, spec.IsCloze, spec.Templates); err != nil {
internal/anki/model/install.go:105:	if err := client.UpdateModelStyling(ctx, spec.Name, css); err != nil {
internal/anki/model/install.go:108:	if err := client.UpdateModelTemplates(ctx, spec.Name, spec.Templates); err != nil {
```
`spec.Name` ranges over `model.Owned()` = `{imoogi-Basic, imoogi-Cloze}` only (`model.go:82-103`). `migrate.go` issues `AddNote`/`deleteNotes` only, never a model write (grep above). Non-test Go `"Basic"`/`"Cloze"` literals: `orgdoc.go:19-20` (renderer dispatch, required by REQ-C-005.2), `model.go:100` (a card-template *name*, not a model), comments only elsewhere. Elisp modules: `24-anki.el:81,221` (cloze-family membership, required by REQ-C-005.2). Wire-level: `TestInstallModelsWritesNoForeignModelOnTheWire` PASS.

Other boundary checks (read, tests observed PASS):
- Media confinement `media.go:284-305`: `Abs` → `EvalSymlinks` → `Clean` → `Rel` against the canonical root; `TestRewriteRejectsEscapeAboveSyncRoot` (`../../secret.png` → `NotFoundError`). Description escaped into `alt` (`TestRewriteEscapesTheDescriptionIntoTheAltAttribute`).
- Base stylesheet air-gap: substring test for `http`, `@import`, `url(`; asset inspected, none present.
- `migrate`: `--dry-run` is the only accepted flag, unknown flag → exit 2 (`main.go:90-96`), so a typo cannot fall through to the writing path. Dry run issues no request and the registry is not persisted (`main.go:103`). Elisp gate `y-or-n-p` names the count and the scheduling loss; declined ⇒ argv log exactly `(("migrate" "--dry-run"))` (test PASS). Add precedes delete; original deleted only via the ownership-confirmed path (`migrate.go:237-248`).
- Elisp → Go install document: exactly `{protocol_version, anki_connect_url, user_css}` (`imoogi-process.el:76-79`) matching `InstallRequest` tags (`protocol.go:133-140`); version mismatch → `binary_incompatible` (`cmd/imoogi-anki/install_test.go:320`).
- Error-code table pairing (REQ-C-023), mechanical:
```
Go codes (protocol.go):        18
Elisp table (imoogi-error.el): 21
Go codes missing from Elisp:   (none)
Elisp-only codes:              binary_not_found, note_id_unknown, sync_root_unset   ← front-end-raised codes, expected
```

**Baseline-attribution.** HEAD `b799850`; greps and tests run 2026-09-08 against that tree.

**Gaps.** Live-collection guarantee (AC-C-022c / REQ-C-024.4) not observed. `note_id_unknown` is an Elisp-table entry with no emitter on either side (grep) — harmless, pre-SPEC.

**Residual-risk.** F5 (TOCTOU between hashing image bytes and Anki reading the path) is a narrow single-run race, not a security boundary. AnkiConnect URL is user configuration, not untrusted input.

---

### §3 Craft (20%) — 90/100 — PASS

**Claim.** Coverage meets every plan §G floor; race, vet, gofmt clean; one vacuous test assertion flagged by staticcheck.

**Evidence.**
```
$ go test -race ./internal/anki/planner/... ./internal/anki/media/...
ok  	github.com/karohani/imoogi-emacs/internal/anki/planner	2.055s
ok  	github.com/karohani/imoogi-emacs/internal/anki/media	1.673s
$ go vet ./internal/anki/... ./cmd/imoogi-anki/...          → exit 0, no output
$ gofmt -l internal/anki cmd/imoogi-anki                     → (empty)
$ golangci-lint run ./internal/anki/... ./cmd/imoogi-anki/...
internal/anki/model/css_test.go:213:5: SA4000: identical expressions on the left and right side of the '!=' operator (staticcheck)
	if model.BaseCSS() != model.BaseCSS() {
1 issues:
* staticcheck: 1
$ GOOS=windows go build ./cmd/imoogi-anki   → WINDOWS_BUILD_EXIT=0
$ go build ./internal/anki/... ./cmd/imoogi-anki/...   → HOST_BUILD_EXIT=0
```
Coverage vs plan §G floors (plan.md:64, 960-965): ankiconnect 86.2 ≥ 85.0 · planner 91.5 ≥ 90.3 · registry 87.8 ≥ 87.8 · hashing 100 ≥ 100 · orgdoc 100 ≥ 100 · model 98.3, media 91.3, cmd 86.9 ≥ 85.0 norm. All above floor.

**Baseline-attribution.** HEAD `b799850`, `-count=1`, 2026-09-08. The project gate `make lint` is `go vet ./...` only (Makefile:118), so progress.md's "0 new lints" claim is accurate against the project gate; the SA4000 finding comes from the additional `golangci-lint` run this audit performed.

**Gaps.** No golangci-lint baseline at 8eaf273 was measured, so "introduced by this SPEC" for F2 rests on the file being new in M2 (`git diff --stat 8eaf273..264bbf6` lists `css_test.go` as added — inferred from the model package being new, not re-diffed).

**Residual-risk.** RED evidence for M2/M5 was lost with rate-limited agents (progress.md); tests are green now but pre-GREEN failure was never observed for those milestones. Environment: `make test-elisp` exits 2 on the user's org-preview WIP, so the pre-push `ci-local` gate cannot pass until that is resolved (out of SPEC scope).

---

### §4 Consistency (15%) — 85/100 — PASS

**Claim.** Code follows the parent SPEC's patterns (typed codes, fake-client call logs, `//go:embed`, English register in `modules/anki/*.el` with the Korean wrapper in `24-anki.el`). The wire-key spelling drifted at the documentation layer only.

**Evidence.** Wire key grep (§1 H2). Elisp `argv` seam kept 2-arg with the subcommand in a dynamically bound `imoogi-process-argv` (`imoogi-process.el:22`), exercised by `imoogi-install-test-call-binary-passes-the-declared-argv`. Migrate reuses the sync request/response shape (`main.go:98-106`, `imoogi-process.el:199-206`). Commits 8eaf273..264bbf6 use conventional messages with the SPEC id.

**Baseline-attribution.** HEAD `b799850`, 2026-09-08.

**Gaps.** Byte-compile warnings for the post-run Elisp were not collected. `design.md` §3.1 vs §3.2/§3.4 self-contradiction (raw wrapper vs mandatory normalization) is the documentary root of F1.

**Residual-risk.** plan.md D8–D10 staleness and spec.md §2 closing-`$` wording (already listed for sync in progress.md) remain open.

---

### Findings (structured defect-list)
- **F1 [warning] [blocking]** `internal/anki/model/assets/basic-front.html:1` (also `basic-back.html:1`, `cloze-front.html:1`, `cloze-back.html:1`; dead normalizer at `internal/anki/model/deckclass.go:39,99`) — REQ-C-006.2's deck-class normalization is never applied at review time; the emitted class is the raw `{{Deck}}` path, so the documented per-deck user selector (`design.md` §3.4 `.deck-programmer-go`) matches nothing and REQ-C-009's user-stylesheet channel is inoperative. Confidence: high (code + `base.css:52-58` comment + codex concurrence). — Required fix: in each of the four templates, emit the raw deck into a data attribute or hidden element and add a template-side `<script>` that applies the seven `NormalizeDeckClass` steps and sets `class="deck-<token>"` on the wrapper (keep the `deck-` prefix so `base.css`'s `[class^="deck-"]` rule still matches); add a JS↔Go mirror test over the AC-C-004 table in the style of `imoogi-notetype-test-deck-class-mirror-table`, and make the Go template test assert the script is present in all four assets; reconcile `design.md` §3.1 with §3.2/§3.4. Alternative above the auditor's authority: amend REQ-C-006/AC-C-004 to accept the raw class — a user decision; the verdict against the SPEC as written stands. Blocks Phase 19: **yes**.
- **F2 [suggestion] [optional]** `internal/anki/model/css_test.go:213` — `model.BaseCSS() != model.BaseCSS()` is always false (staticcheck SA4000); the "stable across calls" assertion is vacuous (the loop below it is meaningful). Confidence: high. — Required fix: drop the line, or compare two separately captured values. Blocks Phase 19: no.
- **F3 [suggestion] [optional]** `.moai/specs/SPEC-ANKICARD-001/acceptance.md:132`, `design.md:335,618`, `spec.md:256,876`, `spec-compact.md:70,99` — install key spelled `ankiconnect_url`; the wire, Go tags, Elisp, and both tests use `anki_connect_url`, so AC-C-006d's text contradicts the test that verifies it. Confidence: high. — Required fix: reconcile all six sites to `anki_connect_url` at sync. Blocks Phase 19: no.
- **F4 [suggestion] [optional]** `internal/anki/planner/planner.go:673` (`uploadMedia`) — the stored filename `StoreMediaFile` returns is discarded (`_`), so an Anki-side rename (the case `ankiconnect/model_test.go:286` exercises) would leave the field `src` pointing at a name the collection does not hold while the registry records success. REQ-C-012.3 deliberately makes the name local, and AnkiConnect overwrites by default, so this is defensive hardening rather than a SPEC gap. Confidence: medium (raised by codex, verified by code read). — Suggested fix: compare the returned name with `u.Filename` and report `media_upload_failed` on mismatch. Blocks Phase 19: no.
- **F5 [suggestion] [optional]** `internal/anki/media/media.go:215` + `planner.go:673` — bytes are hashed from `os.ReadFile`, then Anki re-reads the path at upload; a concurrent replacement of the image inside one run uploads different bytes under the old content-derived name. Narrow race, no reproduction. Confidence: medium (codex, code read). — Suggested fix: carry the captured bytes in `media.Upload` and upload via `storeMediaFile`'s `data` field. Blocks Phase 19: no.

### Not defects, recorded for the record
- AC-C-022c remains UNVERIFIED (needs a live collection); a Phase 19 precondition for the user, not an F-numbered item.
- Post-run drift (§0) audited, not skipped; no SPEC regression observed.

### Recommendations
1. Fix F1 (template-side normalization + mirror test) and re-audit scoped to the F1 delta: re-run `go test -count=1 ./internal/anki/model/...`, `make test-elisp` (anki-notetype tests), and inspect the four assets.
2. At sync: F3 doc reconciliation, `design.md` §3.1 correction, progress.md observations (1)/(2) closure, plan.md D8–D10.
3. Optional hardening F2/F4/F5 at the orchestrator's discretion; none blocks Phase 19.
4. Before Phase 19 closure the user should run install → sync → confirmed migration against the live collection and record N/M and the 88−N / 27−M counts (AC-C-022c).
