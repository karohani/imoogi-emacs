# Sync Audit — SPEC-ANKICARD-004 (Swift arrow card rendering), card t15

- Auditor: sync-auditor (independent, Claude-only; no cross-model backend configured)
- Date: 2026-10-13
- Tree: worktree `.moai/worktrees/t15`, branch `WT-anki-swift-arrow`, HEAD `2f1fd5a`
- Scope: `git diff 7d98ec0..HEAD` (plan `ed28d1a`, run `ea44ff8..5abcc4d`, sync `2f1fd5a`) — 29 files, +4374/−296
- Profile: `default` (harness `default_profile`; SPEC carries no `evaluator_profile`); flat scoring (no `evaluator_mode: hierarchical`)
- Evidence logs: `/private/tmp/claude-501/-Users-jay-workspace-imoogi-emacs/a14120e3-0517-46ef-ab13-336fe6abecc9/scratchpad/sync-audit/ev/` (session scratchpad, outside the repo — ephemeral and will not resolve later; the verbatim lines quoted in this report are the attributable evidence)

## Evaluation Report

SPEC: SPEC-ANKICARD-004
**Overall Verdict: PASS**

### Dimension Scores

| Dimension | Score | Verdict | Evidence (verbatim) |
|---|---|---|---|
| Functionality (40%) | 100/100 | PASS | 26 deciding top-level tests re-run (17 orgdoc + 9 planner), all `--- PASS` (e.g. `--- PASS: TestSwiftArrowLines (0.00s)`, `--- PASS: TestSwiftArrowMissing (0.01s)`, `--- PASS: TestSwiftDispatch (0.00s)`); orgdoc 218 PASS / 0 FAIL, planner 61 PASS / 0 FAIL lines; `Ran 658 tests, 647 results as expected, 0 unexpected, 11 skipped`; README example independently rendered (see § README verification) |
| Security (25%) | 100/100 | PASS | `--- PASS: TestSwiftSnippetBound (0.00s)`; grep shows the only four `@@html:` literals are the fixed table (`swift.go:60`, `:63`, `:64`, `:65`); CSS rule names no URL; `passed  117/658  imoogi-error-test-messages-contain-no-raw-diagnostics (0.000430 sec)`. OWASP applicability: A03 Injection is the only reachable surface (export-snippet content bypasses go-org escaping), bounded mechanically — no line text ever reaches a snippet, only the four fixed constants (`TestSwiftSnippetBound` + grep); side text is wrapped in cloze markers and still escaped by go-org (probe: `:-&gt;` in sides). A05: stylesheet names no network resource. Remaining categories not applicable — no network, auth, session, crypto, or deserialization surface in this diff |
| Craft (20%) | 95/100 | PASS | `orgdoc … coverage: 100.0% of statements`, `planner … coverage: 92.9% of statements`, `hashing … 100.0%`, `model … 98.4%`; `lint exit 0`; `fmt exit 0`; `go test exit 0` (26 packages `ok`) |
| Consistency (15%) | 95/100 | PASS | `make fmt-check` exit 0; `make lint` (go vet) exit 0, no output; refactor reuses the one `#+BEGIN_` pairing and one list walk; one redundant key binding (F2) |

Must-pass firewall: Functionality PASS (all 15 ACs), Security PASS (no Critical/High). Both hold.

Scoring arithmetic:
- Weighted: 0.40·1.00 + 0.25·1.00 + 0.20·0.95 + 0.15·0.95 = **0.9825**
- Harmonic mean: 4 / (1/1.00 + 1/1.00 + 1/0.95 + 1/0.95) = 4 / 4.1053 = **0.974**

Verdict driven by: all ACs PASS on re-execution, no blocking findings, must-pass dimensions pass.

## AC verification (re-executed, not carried over)

All commands run at HEAD `2f1fd5a` in this worktree. Two batched invocations covered every Go AC:

```
go test -count=1 -v -run '^(TestSwiftArrowLines|TestSwiftNumbering|TestSwiftDirectionTable|TestSwiftHandWrittenMarker|TestSwiftBraceSafety|TestSwiftArrowEmphasis|TestSwiftSnippetBound|TestSwiftPipelineOrder|TestRenderSignatureIsUnchanged|TestSwiftDispatch|TestSwiftLineBreak|TestSwiftArrowClass|TestRenderGoldenCorpus|TestMultilineGoldenCorpus|TestSwiftScannerAgreesWithTheParser|TestSwiftMirroredRules|TestSwiftGolden.*)$' ./internal/anki/orgdoc
  -> exit 0; ok  github.com/karohani/imoogi-emacs/internal/anki/orgdoc 0.380s; 218 "--- PASS", 0 "--- FAIL"
go test -count=1 -v -run '^(TestSwiftArrowMissing|TestSwiftUnpairedExtraMarkerIsReported|TestGatePrecedesRendering|TestSwiftWithMultilineOptionIsRejected|TestOptionFreeRequestLogGolden|TestHashCallSiteTakesFourArguments|TestHashIsUnchangedByCardOptions|TestHashChangesThroughRenderedValue|TestCardOptionValueRecognition)$' ./internal/anki/planner
  -> exit 0; ok  github.com/karohani/imoogi-emacs/internal/anki/planner 0.278s; 61 "--- PASS", 0 "--- FAIL"
```

| AC | REQ | Deciding evidence (verbatim) | Status |
|---|---|---|---|
| AC-SW-001 | REQ-SW-001 | `--- PASS: TestSwiftArrowLines (0.00s)`; also `--- PASS: TestSwiftMirroredRules`, `--- PASS: TestSwiftScannerAgreesWithTheParser`. Test asserts byte-identity for zero-arrow rows (`swift_test.go:123: if len(r.sides) == 0 && composed != r.body`) and the 001g broken-output pins (`swift_test.go:189` row 31 `<code>` holds `@@html:`; `:200` row 32 halves in different block elements) | PASS |
| AC-SW-002 | REQ-SW-002, -010 | `--- PASS: TestSwiftArrowMissing (0.01s)`; test asserts no write (`swift_test.go:90 len(client.writeSequence) != 0`), registry unchanged (`:93 reg.Lookup(77) … after != before`), note id carried (`:87`) | PASS |
| AC-SW-003 | REQ-SW-003 | `--- PASS: TestSwiftNumbering (0.00s)` | PASS |
| AC-SW-004 | REQ-SW-004 | `--- PASS: TestSwiftDirectionTable (0.00s)` | PASS |
| AC-SW-005 | REQ-SW-005 | `--- PASS: TestSwiftHandWrittenMarker (0.00s)` | PASS |
| AC-SW-006 | REQ-SW-006 | `--- PASS: TestSwiftBraceSafety (0.00s)`; `passed   27/658  imoogi-anki-cloze-safe-fixture-matches-the-go-composer` | PASS |
| AC-SW-007 | REQ-SW-007 | `--- PASS: TestSwiftArrowEmphasis`, `--- PASS: TestSwiftSnippetBound`; grep below | PASS |
| AC-SW-008 | REQ-SW-008 | `--- PASS: TestSwiftPipelineOrder (0.00s)`, `--- PASS: TestSwiftUnpairedExtraMarkerIsReported (0.00s)` | PASS |
| AC-SW-009 | REQ-SW-009 | `--- PASS: TestRenderSignatureIsUnchanged (0.00s)`, `--- PASS: TestSwiftDispatch (0.00s)` | PASS |
| AC-SW-010 | REQ-SW-010 | `--- PASS: TestGatePrecedesRendering`, `--- PASS: TestSwiftWithMultilineOptionIsRejected`, `--- PASS: TestSwiftUnpairedExtraMarkerIsReported` | PASS |
| AC-SW-011 | REQ-SW-011 | `--- PASS: TestSwiftLineBreak`, `--- PASS: TestSwiftArrowClass`; `#+ATTR_HTML` grep → no output, `rc=1`; `148:.card .swift-arrow {`; `ok … internal/anki/model … coverage: 98.4%` | PASS |
| AC-SW-012 | REQ-SW-012.1-.3 | `--- PASS: TestRenderGoldenCorpus`, `--- PASS: TestMultilineGoldenCorpus`, `--- PASS: TestOptionFreeRequestLogGolden`, `--- PASS: TestHashCallSiteTakesFourArguments`; `git diff --stat 7d98ec0 HEAD -- internal/anki/orgdoc/testdata/` → ` internal/anki/orgdoc/testdata/swift-golden.json \| 58 +++…` / `1 file changed, 58 insertions(+)` | PASS |
| AC-SW-013 | REQ-SW-012.4-.5 | `--- PASS: TestHashIsUnchangedByCardOptions`, `--- PASS: TestHashChangesThroughRenderedValue`, `--- PASS: TestHashCallSiteTakesFourArguments`, `--- PASS: TestCardOptionValueRecognition`, `--- PASS: TestSwiftWithMultilineOptionIsRejected`. `applyHashProbeOptions` docstring read: names the trio, the three evasions, and states "The trio excludes those THREE evasions, not every route" | PASS |
| AC-SW-014 | REQ-SW-013 | `Ran 658 tests, 647 results as expected, 0 unexpected, 11 skipped`; all 12 swift ERT cases `passed` (`imoogi-anki-toggle-swift-*` ×10, `…-swift-property-name-is-a-completion-candidate`, `…-multiline-commands-name-the-card-option-on-a-non-cloze-type`). OFF cases use a prompt trap (`imoogi-anki-test--no-prompt`, `ert-fail` on any `yes-or-no-p`/`y-or-n-p`), not a yes-stub; falsy spelling verified by `imoogi-props-resolve-swift` and an own-drawer regex, not `org-entry-get` | PASS |
| AC-SW-015 | REQ-SW-014 | all nine `imoogi-error-test-*` `passed` incl. `imoogi-error-test-codes-match-source-file`, `imoogi-error-test-swift-arrow-missing-message-states-the-fix`; grep: `protocol.go:126: CodeSwiftArrowMissing = "swift_arrow_missing"`, `imoogi-error.el:71`, `anki-error-test.el:33`, `:157` | PASS |

`ac_pass_count: 15`, `ac_fail_count: 0` — independently reproduced.

### Grep evidence (AC-SW-007 / -011 / -015)

```
$ grep -n '@@html:' internal/anki/orgdoc/*.go | grep -v '_test.go'
internal/anki/orgdoc/swift.go:60:const swiftLineBreak = "@@html:<br>@@"
internal/anki/orgdoc/swift.go:63:	":->":  `@@html:<b class="swift-arrow">:-&gt;</b>@@`,
internal/anki/orgdoc/swift.go:64:	":<-":  `@@html:<b class="swift-arrow">:&lt;-</b>@@`,
internal/anki/orgdoc/swift.go:65:	":<->": `@@html:<b class="swift-arrow">:&lt;-&gt;</b>@@`,
$ grep -n '#+ATTR_HTML' internal/anki/orgdoc/swift.go
rc=1
$ grep -n 'swift-arrow' internal/anki/model/assets/base.css
148:.card .swift-arrow {
```

### PRESERVE list (plan.md § A.3)

```
$ git diff --stat 7d98ec0 HEAD -- internal/anki/planner/card_options.go internal/anki/hashing modules/org/anki/imoogi-props.el modules/org/anki/imoogi-scan.el modules/org/anki/imoogi-process.el .moai/specs/SPEC-ANKICARD-001 .moai/specs/SPEC-ANKICARD-002 .moai/specs/SPEC-ANKICARD-003 modules/org/29-org-roam.el tests/org-roam-test.el modules/general/05-transient.el tests/module-layout-test.el modules/project/04-projects.el tests/workspace-bridge-test.el CLAUDE.md vendor/
(no output; preserve-rc=0)
```

- `render-golden.json` / `multiline-golden.json`: unmodified (testdata diff shows only the new `swift-golden.json`).
- `TestRenderSignatureIsUnchanged`, `TestHashCallSiteTakesFourArguments`, `TestOptionFreeRequestLogGolden`: no `+`/`-` line touches their bodies; the only diff lines naming them are comments in the reshaped probe docstring.
- Hashing package: `git diff --stat 7d98ec0 HEAD -- internal/anki/hashing/` → no output.
- § A.4 amendable sites: `multilineBody` gained the arrow line (assertions unchanged); `applyHashProbeOptions` reshaped to the all-falsy probe with sibling `TestHashChangesThroughRenderedValue`; fourth site `TestMultilineFalsyOptionsAreByteIdenticalToNoOptions` names `Swift: false` and asserts no `swift-arrow`/`<br>`. No assertion was deleted to go green.

### Quality gates

| Gate | Threshold | Command | Observed (verbatim) | Status |
|---|---|---|---|---|
| Go tests | exit 0 | `go test ./... -count=1` | `go test exit 0`; 26 packages `ok` | PASS |
| orgdoc coverage | ≥ 100.0% | `go test ./internal/anki/... -count=1 -cover` | `ok … internal/anki/orgdoc 2.157s coverage: 100.0% of statements` | PASS |
| planner coverage | ≥ 92.8% | same | `ok … internal/anki/planner 2.970s coverage: 92.9% of statements` | PASS |
| hashing / model | unchanged | same | `coverage: 100.0%` / `coverage: 98.4%` | PASS |
| Lint | 0 new | `make lint` (`go vet ./...`) | `lint exit 0`, no output | PASS |
| Format | exit 0 | `make fmt-check` | `fmt exit 0` (stderr carries two pre-existing gofmt parse notes from `vendor/elpa/go-mode-1.6.0/test/testdata/...`, filtered by the target) | PASS |
| Elisp | 0 unexpected, ≥ 645 | `make test-elisp` | `elisp exit 0`; `Ran 658 tests, 647 results as expected, 0 unexpected, 11 skipped` | PASS |
| Native build | exit 0 | `go build ./...` | `build exit 0` | PASS |
| Windows build (anki) | exit 0 | `GOOS=windows GOARCH=amd64 go build ./internal/anki/...` | `win anki exit 0` | PASS |
| Windows build (module) | exit 0 | `GOOS=windows GOARCH=amd64 go build ./...` | fails `internal/setup/setup.go:378:15: undefined: syscall.Flock` | pre-existing: `git show 7d98ec0:internal/setup/setup.go \| grep -c 'syscall.Flock'` → `2`; `git diff --stat 7d98ec0 HEAD -- internal/setup/` → no output. Not introduced. |
| Full local CI | exit 0 | `make ci-local` | not re-run (union of the rows above) | GAP |

## Deviations review (progress.md § Deviations, Route, O-1)

1. **Context-free route `"x\n"+line` instead of parsing the line alone** (`isPlainTextLine`, `swift.go:183-186`). Within latitude: REQ-SW-001.6 states "How the composer obtains it is the implementer's choice"; DD-4 records the single-line parse as *one* route measured, not a mandate. The chosen route asks go-org itself (no shape list), satisfying the definitional obligation. Its failure direction is over-exclusion (a lone opener the parser later rejects is withheld), never a hidden marker. Independent probe confirms it: `a) 도쿄 :-> 일본` → `swift card option is on but the entry's body carries no arrow line` (go-org reads `a)` as an ordered list); `-----⏎도쿄 :-> 일본` → `<hr>\n<p>도쿄 <b class="swift-arrow">:-&gt;</b> {{c1::일본}}</p>`. **Accepted.**
2. **O-1 implemented** (affiliated keyword after `#+RESULTS:`, unclosed `:RESULTS:` drawer, nested source block as result, empty block). Required anyway by REQ-SW-001.5's "the one construct it reads next"; each case is in `swiftMirrorCases` and checked against go-org by `--- PASS: TestSwiftScannerAgreesWithTheParser`. **Accepted.**
3. **`multiline.go` refactor — `pairedBlocks` / `walkList`.** Read line-by-line against the base: `blockRegions` now derives the identical interior from `pairedBlocks` spans; `walkList`'s four roles reproduce the old branch structure exactly (blank → close span; sibling bullet → new item; nested bullet → close span; continuation → extend only while open; `blankRun ≥ 2` and indent-below-`minIndent` both terminate). Behavior preservation evidenced by `--- PASS: TestMultilineGoldenCorpus (0.00s)`, `multiline-golden.json` unmodified, orgdoc `coverage: 100.0%`. Satisfies REQ-SW-001.6 "reused rather than copied". **Accepted.**
4. Milestone-boundary moves (A.4 sites and the ERT code list landing in M2) keep every commit green; no SPEC impact. **Accepted.**

## Elisp toggle semantics (REQ-SW-013)

Read `imoogi-anki-toggle-swift` (`modules/org/24-anki.el`):
- Direction is decided from `imoogi-props-resolve-swift` (inherited value), truthy iff trimmed `t` case-insensitive — matches Go `readBoolean` (`strings.EqualFold(trimmed, optionTruthy)`). REQ-SW-013.1 ✓
- OFF branch: one `org-set-property "ANKI_SWIFT" imoogi-anki-option-falsy`; no `imoogi-anki--ensure-cloze-type`, no `imoogi-anki--multiline-options-on`, no prompt. REQ-SW-013.2/.3/.4 OFF halves ✓ (pinned by `…-off-keeps-the-multiline-option-without-a-prompt`, `…-off-on-a-basic-heading-writes-the-falsy-spelling`, `…-off-creates-no-note-type`, `…-turns-an-inherited-option-off-without-a-prompt`, all under the prompt trap).
- ON branch: multiline conflict (`yes-or-no-p` naming every on option, clearing each to `nil` on confirm) → note-type contract → write `t`. Not routed through `imoogi-anki--write-card-option` (DD-10) ✓. Note-type message now says "카드 옵션" ✓. `ANKI_SWIFT` in `imoogi-anki-property-names` ✓. Key `w` in map and transient ✓.

## README verification (`### Anki 화살표 카드`)

The README's Org example was rendered through the real composer with a temporary in-package test (`internal/anki/orgdoc/zz_syncaudit_tmp_test.go`, deleted immediately after the run; `git status --short` afterwards showed only `?? .codex/` and `?? .moai/reports/t15/`):

```
RenderWithOptions(NoteTypeCloze, "나라와 수도", "도쿄 :-> 일본\n파리 :<- 프랑스\n베를린 :<-> 독일\n", CardOptions{Swift: true})
Text="<p>나라와 수도</p>\n<p>\n도쿄 <b class=\"swift-arrow\">:-&gt;</b> {{c1::일본}}<br>\n{{c4::파리}} <b class=\"swift-arrow\">:&lt;-</b> 프랑스<br>\n{{c6::베를린}} <b class=\"swift-arrow\">:&lt;-&gt;</b> {{c5::독일}}</p>\n" err=<nil>
```

| README claim | Observed | Match |
|---|---|---|
| `일본` is c1 | `{{c1::일본}}` | ✓ |
| `파리` is c4 | `{{c4::파리}}` | ✓ |
| `독일` and `베를린` are c5 and c6 | `{{c5::독일}}`, `{{c6::베를린}}` | ✓ |
| `:->` hides right, one card | right wrapped, left bare | ✓ |
| `:<-` hides left, one card | left wrapped, right bare | ✓ |
| `:<->` hides both, two cards | both wrapped, two distinct numbers | ✓ |
| `C-c a w` / menu **화살표 카드** | `(define-key map (kbd "w") #'imoogi-anki-toggle-swift)`; transient `("w" "화살표 카드" imoogi-anki-toggle-swift)`; ERT `…-is-reachable-from-the-key-map-and-the-menu` passed | ✓ |
| ON adds `imoogi-Cloze`, asks before clearing direction/incremental | code + ERT `…-turns-an-absent-option-on`, `…-on-with-both-multiline-options-asks-first` | ✓ |
| OFF writes `nil`, does not delete | code + ERT `…-turns-an-inherited-option-off-without-a-prompt` | ✓ |
| No arrow line → skipped as `swift_arrow_missing`, existing note kept | `TestSwiftArrowMissing` (no write, registry unchanged, note id carried) | ✓ |
| Limitations: inline constructs unsupported; inline footnote refused by Anki; rare shapes withheld rather than mis-blanked | matches spec § 5, AC-SW-001g pins, progress.md known limitation 1 | ✓ |

## Commit hygiene

```
$ git log --format='=== %h%n%B%n--end--' 7d98ec0..HEAD   (9 commits)
every subject ends "(t15)"; every message ends:
🗿 MoAI
Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
grep -c 'Co-Authored-By' → 9
```

All nine commits name t15 in the subject and end with the required trailer lines. Conventional-commit types (`docs`/`feat`/`test`) with scope `SPEC-ANKICARD-004`. Author `jay <a@gmail.com>` matches `~/.gitconfig` and prior history on `origin/main`. Shared-config check: `core.bare=false`; repository-local `[user]` equals the global value; the git-config-corruption fix `43f88d1` is an ancestor of HEAD (`43f88d1 is ancestor`).

## Findings (structured defect list)

No blocking findings.

- **F1** [Low] [optional] [confidence: high] `README.md` § "Anki 화살표 카드", "화살표 줄이 되는 줄" paragraph — the README's not-an-arrow-line list (list items, blocks/environments, tables, footnote definitions, EXTRA) is a subset of the `swift_arrow_missing` message's list (also headings, keyword/comment/fixed-width lines, PROPERTIES drawers, suppressed source-block results, a hand-written-blank-spanning arrow). The leading sentence "heading 바로 밑의 평범한 문장 줄이어야 하고" covers them definitionally, so nothing is false. Required fix (optional): add "제목·키워드·주석·고정폭 줄" or point to the diagnostic message.
- **F2** [Low] [optional] [confidence: high] `modules/org/24-anki.el:527` and `:546` — `w` is bound to `imoogi-anki-toggle-swift` both in the `defvar imoogi-anki-map` body and in the reload `dolist`. Keys added after the map's creation (`D`, `F`, `l`, `L`, `g`, `u`) live only in the `dolist`; the duplicate is harmless. Required fix (optional): drop one of the two (keeping the `dolist` entry preserves reload behavior).
- **F3** [Low] [optional] [confidence: high] `internal/anki/orgdoc/swift.go:507-510` — with a CRLF body the `\r` stays as line content before the inline break (probe: `{{c1::B}}\r@@html:<br>@@`). Renders harmlessly in HTML; outside SPEC scope (bodies arrive LF from Emacs). No fix required.
- **F4** [Info] [optional] [confidence: high] `modules/org/24-anki.el` toggle ON path — on an `imoogi-Basic` heading with a multiline option on, confirming the conflict clears the multiline option(s) and then the note-type check refuses swift (two non-atomic writes). Disclosed as progress.md known limitation 5; the order is forced by AC-SW-014's "decline writes nothing" (a type-first order would write `imoogi-Cloze` before the prompt); the heading is one the gate already rejects. Untested combination. No fix required.
- **F5** [Low] [optional] [confidence: medium] `tests/anki-commands-test.el:424` — the decline case asserts `(should-not (org-entry-get (point) "ANKI_SWIFT"))`, which also passes if the falsy spelling `nil` were written (the SPEC records `org-entry-get` returns Lisp nil for it). The code writes nothing on decline, so behavior is correct; the assertion is just weaker than the own-drawer regex used elsewhere. Required fix (optional): assert `(should-not (string-match-p "ANKI_SWIFT" (buffer-string)))`.
- **F6** [Info] [optional] [confidence: high] `.moai/specs/SPEC-ANKICARD-004/acceptance.md` § Definition of Done — all nine boxes remain `[ ]` while `spec.md` is `status: completed`. Each item is evidenced in this report (AC 15/15, gates, goldens, A.4 sites, PRESERVE, README, corpus byte-identity rows, §E.2/§E.3); "no git write against the shared checkout" is the one item not verifiable here (Gap). Required fix (optional): tick the boxes in the sync record.

## Recommendations

- Optional: address F1, F2, F5 in a follow-up docs/test touch-up; none gates this card.
- Before relying on the cards in production, exercise acceptance.md Residual Risks 1-4 against a live Anki collection (unchanged from the run record).

## Gaps (not observed in this audit)

- `make ci-local` not re-run as a single target (its constituents fmt-check, lint, go test, elisp were each run).
- "No git command that writes ran against the shared checkout" (DoD item) cannot be verified from this worktree.
- Residual risks 1-4 (number gaps incl. a body with no `c1`, stylesheet in Anki's webview, `<b>` between cloze markers, early-adopter `ANKI_SWIFT` headings changing on next sync) — need a running Anki; inherited unverified.
- Cross-model second opinion (codex/GLM) not run: no `audit_model` configured, and the brief excluded agents/background work.
- The two intermittent ERT failures the run record observed did not reproduce in this audit's single elisp run (`0 unexpected`); a single run cannot rule them out.
- `go test` coverage was measured for `internal/anki/...` only; no project-wide coverage figure.

## Residual risk

- The context-dependent mirror of go-org v1.9.1 is pinned by `TestSwiftScannerAgreesWithTheParser`; a go-org upgrade could change classifications. The agreement test would catch drift for corpus and mirror cases only, not for shapes outside them.
- Disclosed over-exclusions (environment-first / source-first interleavings, `:PROPERTIES:` inside another drawer, unclosed block header carrying an arrow) withhold cards silently beyond the `swift_arrow_missing` skip only when no other arrow line exists.
- Inline-construct arrows (rows 31/32) are pinned broken, not repaired, per § 5.
