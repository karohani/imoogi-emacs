# Progress — SPEC-ANKICARD-004

Swift arrow card rendering. Tier M, three plan-phase artifacts plus this file.

## §E.1 Plan-phase Audit-Ready Signal

```yaml
plan_status: audit-ready
plan_complete_at: 2026-10-12
spec_id: SPEC-ANKICARD-004
spec_version: "0.1.4"
tier: M
artifacts: [spec.md, plan.md, acceptance.md, progress.md]
requirements: 14
acceptance_criteria: 15
corpus_rows: 42
depends_on: [SPEC-ANKICARD-003, SPEC-ANKICARD-005]
baseline_head: 7d98ec0
next_audit_iteration: 4
next_audit_scope: delta-only (R3-1, R3-2, R3-3, R3-4, O-2)
next_audit_authorization: granted (decision board d-20261012T131704Z-eab1, scope card:t15 — PASS proceeds to run; FAIL stops, no further rounds)
answers_audit: .moai/reports/plan-audit/SPEC-ANKICARD-004-plan-audit-r3.md
```

Iteration 3 (the last of the three the retry loop allows) returned FAIL 0.75.
v0.1.4 is the targeted fix that the escalation's option 2 describes. It closes
the four blocking findings R3-1 to R3-4 and takes the optional O-2. The
per-finding account is in `spec.md` HISTORY v0.1.4.

A fourth, delta-only audit pass goes beyond the iteration cap. It needs the
user's explicit override; it was **granted** (decision board
`d-20261012T131704Z-eab1`) and ran: plan-audit r4 **PASS 0.92**
(`.moai/reports/plan-audit/SPEC-ANKICARD-004-plan-audit-r4.md`). R4-1 applied
to `plan.md` DD-4 before run. The Tier M PASS threshold is 0.80.

The corpus grows from 38 rows to 42. The four new rows (39-42) were measured
against go-org v1.9.1 from the module cache, in a scratch module outside the
checkout that was deleted afterwards. The baseline is unchanged at `7d98ec0`
(`plan.md` § A.2).

## §E.2 Run-phase Evidence

Run phase, `cycle_type: tdd`, card t15. Worked in the L1 worktree
`.claude/worktrees/agent-a8e14baae19af5a96` (branch
`worktree-agent-a8e14baae19af5a96`), fast-forwarded from `7d98ec0` to the
plan commit `ed28d1a` before any code change. Verbatim command logs for every
row below are under `.moai/state/verify/t15-run/` (ignored by git, so they
live with this worktree).

### Commits

| Milestone | SHA | Subject |
|---|---|---|
| M1 | `ea44ff8` | feat(SPEC-ANKICARD-004): M1 the arrow rule and swift composition (t15) |
| M2 | `95b8a36` | feat(SPEC-ANKICARD-004): M2 swift dispatch, pipeline, and swift_arrow_missing (t15) |
| M3 | `945eea8` | test(SPEC-ANKICARD-004): M3 swift golden corpus and non-interference (t15) |
| M4 | `bf076d8` | feat(SPEC-ANKICARD-004): M4 swift-arrow stylesheet rule (t15) |
| M5 | `c72ef69` | feat(SPEC-ANKICARD-004): M5 swift toggle command and completion candidate (t15) |
| M6 | — | README, deferred to the sync phase (see Deviations) |

### AC matrix

Every deciding command was run at `c72ef69` against a clean tree. "Key output"
quotes the command's own lines; the log path holds the full verbose run.

| AC | Deciding command | Key output (verbatim) | Status |
|---|---|---|---|
| AC-SW-001 | `go test -run '^TestSwiftArrowLines$' ./internal/anki/orgdoc -v` | `--- PASS: TestSwiftArrowLines`; 48 PASS / 0 FAIL (42 corpus rows + sub-criteria a, b, e, f, g); `ok github.com/karohani/imoogi-emacs/internal/anki/orgdoc` (`ac01.txt`) | PASS |
| AC-SW-002 | `go test -run '^TestSwiftArrowMissing$' ./internal/anki/planner -v` | `--- PASS: TestSwiftArrowMissing`; 26 PASS / 0 FAIL (23 zero-arrow rows, message, row 12 under multiline); `ok …/internal/anki/planner` (`ac02.txt`) | PASS |
| AC-SW-003 | `go test -run '^TestSwiftNumbering$' ./internal/anki/orgdoc -v` | `--- PASS: TestSwiftNumbering`; 4 PASS / 0 FAIL (`ac03.txt`) | PASS |
| AC-SW-004 | `go test -run '^TestSwiftDirectionTable$' ./internal/anki/orgdoc -v` | `--- PASS: TestSwiftDirectionTable` (`ac04.txt`) | PASS |
| AC-SW-005 | `go test -run '^TestSwiftHandWrittenMarker$' ./internal/anki/orgdoc -v` | `--- PASS: TestSwiftHandWrittenMarker`; 7 PASS / 0 FAIL (rows 18, 19, 24, 16, the c5→c6/c8 offset, title marker, all-sides-marked) (`ac05.txt`) | PASS |
| AC-SW-006 | `go test -run '^TestSwiftBraceSafety$' ./internal/anki/orgdoc -v`; `make test-elisp 2>&1 \| grep -i 'cloze-safe'` | `--- PASS: TestSwiftBraceSafety` (`ac06.txt`); `passed   27/658  imoogi-anki-cloze-safe-fixture-matches-the-go-composer` (`final-ci-local.txt`) | PASS |
| AC-SW-007 | `go test -run '^(TestSwiftArrowEmphasis\|TestSwiftSnippetBound)$' ./internal/anki/orgdoc -v`; `grep -n '@@html:' internal/anki/orgdoc/*.go \| grep -v '_test.go'` | `--- PASS: TestSwiftArrowEmphasis`, `--- PASS: TestSwiftSnippetBound` (`ac07.txt`); grep: `swift.go:60:const swiftLineBreak = "@@html:<br>@@"` and `swift.go:63-65` (the three arrow entries) — four lines, all in the table | PASS |
| AC-SW-008 | `go test -run '^TestSwiftPipelineOrder$' ./internal/anki/orgdoc -v`; `go test -run '^TestSwiftUnpairedExtraMarkerIsReported$' ./internal/anki/planner -v` | `--- PASS: TestSwiftPipelineOrder` (6 PASS) (`ac08a.txt`); `--- PASS: TestSwiftUnpairedExtraMarkerIsReported` (3 PASS) (`ac08b.txt`) | PASS |
| AC-SW-009 | `go test -run '^(TestRenderSignatureIsUnchanged\|TestSwiftDispatch)$' ./internal/anki/orgdoc -v` | `--- PASS: TestRenderSignatureIsUnchanged`, `--- PASS: TestSwiftDispatch` (8 PASS) (`ac09.txt`) | PASS |
| AC-SW-010 | `go test -run '^(TestGatePrecedesRendering\|TestSwiftWithMultilineOptionIsRejected\|TestSwiftUnpairedExtraMarkerIsReported)$' ./internal/anki/planner -v` | `--- PASS: TestSwiftWithMultilineOptionIsRejected`, `--- PASS: TestGatePrecedesRendering`, `--- PASS: TestSwiftUnpairedExtraMarkerIsReported` (13 PASS) (`ac10.txt`) | PASS |
| AC-SW-011 | `go test -run '^(TestSwiftLineBreak\|TestSwiftArrowClass)$' ./internal/anki/orgdoc -v`; `grep -n '#+ATTR_HTML' internal/anki/orgdoc/swift.go`; `grep -n 'swift-arrow' internal/anki/model/assets/base.css`; `go test ./internal/anki/model/... -count=1` | `--- PASS: TestSwiftLineBreak`, `--- PASS: TestSwiftArrowClass` (`ac11a.txt`); ATTR_HTML grep: no output, `grep exit 1`; `148:.card .swift-arrow {`; `ok github.com/karohani/imoogi-emacs/internal/anki/model 0.436s` | PASS |
| AC-SW-012 | `go test -run '^(TestRenderGoldenCorpus\|TestMultilineGoldenCorpus)$' ./internal/anki/orgdoc -v`; `git diff --stat -- internal/anki/orgdoc/testdata/` | `--- PASS: TestRenderGoldenCorpus`, `--- PASS: TestMultilineGoldenCorpus` (`ac12a.txt`); worktree diff: no output; since the baseline, `git diff --stat 7d98ec0 HEAD -- internal/anki/orgdoc/testdata/` → `internal/anki/orgdoc/testdata/swift-golden.json \| 58 +++…` (`1 file changed, 58 insertions(+)`) — the swift corpus only. Also `--- PASS: TestOptionFreeRequestLogGolden`, `--- PASS: TestHashCallSiteTakesFourArguments` (`ac12b.txt`) | PASS |
| AC-SW-013 | `go test -run '^(TestHashIsUnchangedByCardOptions\|TestHashChangesThroughRenderedValue\|TestHashCallSiteTakesFourArguments)$' ./internal/anki/planner -v`; `go test -run '^(TestCardOptionValueRecognition\|TestSwiftWithMultilineOptionIsRejected)$' ./internal/anki/planner -v` | `--- PASS: TestHashIsUnchangedByCardOptions`, `--- PASS: TestHashChangesThroughRenderedValue`, `--- PASS: TestHashCallSiteTakesFourArguments` (`ac13a.txt`); `--- PASS: TestCardOptionValueRecognition`, `--- PASS: TestSwiftWithMultilineOptionIsRejected` (27 PASS) (`ac13b.txt`). Docstring of `applyHashProbeOptions` names the three legs and the three evasions "rather than every route" (read, `internal/anki/planner/baseline_golden_test.go`) | PASS |
| AC-SW-014 | `make test-elisp 2>&1 \| tail -5` | `Ran 658 tests, 647 results as expected, 0 unexpected, 11 skipped`; the 12 swift cases each `passed` (`imoogi-anki-toggle-swift-*`, `imoogi-anki-swift-property-name-is-a-completion-candidate`, `imoogi-anki-multiline-commands-name-the-card-option-on-a-non-cloze-type`) (`final-ci-local.txt`) | PASS |
| AC-SW-015 | `make test-elisp 2>&1 \| grep -i 'imoogi-error-test'`; `grep -n 'swift_arrow_missing' internal/anki/protocol/protocol.go modules/org/anki/imoogi-error.el tests/anki-error-test.el` | all nine `imoogi-error-test-*` `passed`, including `imoogi-error-test-codes-match-source-file` and `imoogi-error-test-swift-arrow-missing-message-states-the-fix`; grep: `protocol.go:126: CodeSwiftArrowMissing = "swift_arrow_missing"`, `imoogi-error.el:71: ("swift_arrow_missing" .`, `anki-error-test.el:33`, `:157` | PASS |

`ac_pass_count: 15`, `ac_fail_count: 0`.

### Quality gates

| Gate | Threshold | Command | Observed | Status |
|---|---|---|---|---|
| Go tests | exit 0 | `go test ./... -count=1` (after M4) and `make ci-local` (after M5) | exit 0, 26 packages `ok` (`m4-go-all.txt`); `ci-local exit 0` | PASS |
| `orgdoc` coverage | ≥ 100.0% | `go test ./internal/anki/... -count=1 -cover` | `coverage: 100.0% of statements` | PASS |
| `planner` coverage | ≥ 92.8% | same | `coverage: 92.9% of statements` | PASS |
| `hashing` / `model` | unchanged | same | `100.0%` / `98.4%` | PASS |
| Lint | 0 new | `make lint` | exit 0, no output | PASS |
| Format | exit 0 | `make fmt-check` | exit 0 | PASS |
| Elisp | 0 unexpected, ≥ 645 | `make test-elisp` (inside `make ci-local`) | `Ran 658 tests, 647 results as expected, 0 unexpected, 11 skipped` | PASS |
| Full local CI | exit 0 | `make ci-local` | `ci-local exit 0`, `setup-toolchain tests: PASS` | PASS |
| Native build | exit 0 | `go build ./...` | exit 0 | PASS |
| Windows build | exit 0 | `GOOS=windows GOARCH=amd64 go build ./internal/anki/...` | exit 0 | PASS (anki packages) |
| Windows build, whole module | — | `GOOS=windows GOARCH=amd64 go build ./...` | exit 1 in `internal/setup/setup.go:358-378` (`undefined: syscall.Flock`, `syscall.LOCK_EX`, `syscall.O_NOFOLLOW`) | pre-existing at `7d98ec0` (`preflight-build.txt`), not introduced; no file of that package is touched here |

Coverage before → after (baseline `7d98ec0`, re-measured in this worktree,
`preflight-anki-cover.txt`): `orgdoc` 100.0 → 100.0, `planner` 92.8 → 92.9,
`hashing` 100.0 → 100.0, `model` 98.4 → 98.4. Elisp tests 645 → 658 (13 new).

### RED evidence (captured before each GREEN)

| Milestone | Log | Verbatim first failure |
|---|---|---|
| M1 | `red-M1.txt` | `--- FAIL: TestSwiftArrowLines/row_01` … `swift_test.go:121: arrow lines = [], want [["도쿄" "일본"]]` — 63 failing (sub)tests against a stub `swift.go` |
| M1 (agreement test) | `red-M1-agreement.txt` | 37 failing subtests of `TestSwiftScannerAgreesWithTheParser` against the stub |
| M2 Go | `red-M2-go.txt` | `swift_pipeline_test.go:19: err = cloze note type declared but no {{cN:: marker found in title or body, want *SwiftArrowMissingError — the split ran first`; `multiline_test.go:429: renderError = ("org_parse_error", false), want ("swift_arrow_missing", true)`; `swift_test.go:86: code = "cloze_marker_missing", want "swift_arrow_missing"` |
| M2 hash sibling | `red-M2-hash-sibling.txt` | `baseline_golden_test.go:332: the swift option left the Text field unchanged: "<p>A {{c1::cloze}} heading</p>\n<p>\n도쿄 :-&gt; 일본</p>\n"` |
| M2 Elisp | `red-M2-elisp.txt` | `FAILED  imoogi-error-test-go-codes-are-subset-of-table`, `FAILED  imoogi-error-test-swift-arrow-missing-message-states-the-fix` |
| M3 | `red-M3.txt` | `swift_golden_test.go:70: reading golden testdata/swift-golden.json: open testdata/swift-golden.json: no such file or directory (record it with UPDATE_SWIFT_GOLDEN=1)` |
| M4 | `red-M4.txt` | ``css_test.go:301: base stylesheet carries no rule naming `.card .swift-arrow` `` |
| M5 | `red-M5-elisp.txt` | 12 `FAILED  imoogi-anki-toggle-swift-*` / `…-swift-property-name-…` / `…-name-the-card-option-…` |

### Route chosen for the context-free classification

REQ-SW-001.6 leaves the route to the implementer; `plan.md` DD-4 records
parsing the line alone as "one route measured". That route was measured here
and is **not** used, because it admits three shapes that render the arrow to
nothing: read alone, `#+RESULTS: a :-> b`, `#+NAME: A :-> B`, and
`#+CAPTION: A :-> B` each parse as `org.Paragraph` (the opener fails and falls
back to text), while `#+RESULTS: a :-> b`⏎`foo` and `#+NAME: A :-> B`⏎`foo`
render `<p>foo</p>` and `#+CAPTION: A :-> B`⏎`foo` moves the arrow into a
`<figcaption>`. The route used (`isPlainTextLine`) parses `x`⏎line and admits
the line only when go-org continues the paragraph through it — that is, when
go-org lexes it as plain text. A lone opener the parser later rejects (an
unclosed `#+BEGIN_SRC go :-> x`) is withheld too: over-exclusion, never a hidden
marker. Pinned by `TestSwiftMirroredRules` and `swiftDisclosedDivergences`.

### Deviations from the plan, with reasons

1. **Branch.** The brief named branch `WT-anki-swift-arrow` at `ed28d1a`; this
   worktree was on `worktree-agent-a8e14baae19af5a96` at `7d98ec0`, and
   `WT-anki-swift-arrow` is checked out in `.moai/worktrees/t15`. The branch was
   fast-forwarded to `ed28d1a` (one docs commit) and the run committed here; it
   is not renamed, since the target name is taken. Integration into
   `WT-anki-swift-arrow` is the lead's.
2. **M6 (README) not done.** The brief puts README in the sync phase, and this
   agent's ownership excludes README. The Definition of Done's README checkbox
   stays open for manager-docs.
3. **Milestone boundaries.** The A.4 amendable sites (the hash-probe reshape
   with its sibling and docstring, and the `multilineBody` arrow line) break the
   moment swift renders, so they landed in M2 rather than M3, keeping every
   commit green. The `imoogi-error.el` message and the ERT code list (REQ-SW-014,
   planned for M5) also landed in M2, because adding the Go constant without
   them fails `imoogi-error-test-codes-match-source-file`.
4. **One pairing rule, one list walk — refactored, not copied.** `multiline.go`'s
   `#+BEGIN_` pairing now returns spans (`pairedBlocks`), with `blockRegions`
   deriving its interior from them, and `scanAnswerList`'s continuation walk is
   extracted as `walkList`. The swift scanner needs a block's closing line
   (the source-block result, and an empty block, which has no interior line) and
   every list rather than the first. Behaviour is unchanged:
   `TestMultilineGoldenCorpus`, `TestMultilineAnswerList`, and the rest of the
   multiline tests pass, and `multiline-golden.json` is unmodified.
5. **The prompt in the OFF-path ERT cases is trapped, not stubbed.** AC-SW-014
   forbids stubbing the confirmation prompt to always-yes. The OFF cases replace
   `yes-or-no-p`/`y-or-n-p` with a function that fails the test if called, so a
   prompt that fires fails the test rather than being answered. A bare prompt
   under `--batch` would read standard input, which fails or blocks depending on
   how the suite is launched.
6. **`TestSwiftScannerAgreesWithTheParser` added.** Beyond the ACs: for every
   corpus row and mirror case decided by placement, the scanner's count is
   compared with whether go-org itself renders the line inside a top-level
   paragraph. It is the falsifiability check the multiline scanner has in
   `TestMultilineAnswerList`.

### O-1 (plan-audit r4, optional) — disposition

Implemented, because the general rule in `spec.md` REQ-SW-001.5 ("the **one**
construct it reads next") already requires it and each case fell out of rules
the scanner mirrors anyway: an affiliated keyword after `#+RESULTS:` (the
unrecognized first construct is read as a paragraph run, which go-org's
fall-back also does), an unclosed `:RESULTS:` drawer (runs to the body's end),
a nested source block as the result (carries its own result with it), and an
empty source block (seen through the paired span). Each is a case in
`swiftMirrorCases`, and the agreement test checks it against go-org.

### Known limitations

1. **Over-exclusions, disclosed and pinned** (`swiftDisclosedDivergences`): the
   environment-first and source-first interleavings of § 5; a `:PROPERTIES:`
   line inside an ordinary drawer (go-org renders it as text; the property rule
   is computed on its own per REQ-SW-001.6); a line under a column-zero `*`
   (the list walk reads it as a bullet — unreachable, a body holds no
   headline); an unclosed block header carrying the arrow. Each withholds a card
   and never composes a hidden marker.
2. **Constructs the title opens are not modelled.** The renderer parses the
   title, a blank line, and the body together, and the scanner reads the body
   alone (the marker-extent scan is the exception: it reads the title too). A
   heading title that itself opens a construct — `[fn:1] …`, `- …`,
   `#+BEGIN_SRC` — can change how the first body lines render. A title of that
   shape is unusual for a card heading.
3. **A footnote definition whose own content opens a block on the label line**
   (`[fn:1] #+BEGIN_SRC` …) is not seen as a region, so a blank pair inside that
   block ends the definition early in the scanner. Not reachable from an
   ordinary note.
4. **Whitespace-only lines** are blank to the scanner by Unicode whitespace;
   go-org counts only ASCII whitespace. A line of only non-breaking spaces is
   blank here and text to go-org.
5. **The ON path's two writes are not atomic**, the same shape as the existing
   multiline commands: confirming the multiline conflict clears those options
   before the note-type check runs, so on an `imoogi-Basic` heading with a
   multiline option on, a confirmed toggle clears the multiline option and then
   refuses swift. The validation gate already rejects that heading.
6. **Residual risks 1-4 of `acceptance.md`** are unchanged: none was observed
   against a running Anki collection.

### Observations outside scope

- Two ERT tests failed intermittently in two of five full Elisp runs
  (`red-M2-elisp.txt`, `red-M5-elisp.txt`) and passed in the baseline, both
  GREEN runs, and `make ci-local`:
  `imoogi-project-notes-cli-cache-superseded-process-does-not-callback`
  (`tests/project-notes-cli-cache-integration-test.el:226`) and
  `imoogi-setup-test-ankiconnect-status-invalid-response-on-http-error`
  (`tests/anki-setup-test.el:86`). Neither file nor its subject is touched here;
  both look timing-dependent. Not investigated further.
- `git config --get core.bare` printed `false` after every commit, and the
  local `[user]` / `core.bare` values were identical in every snapshot from
  just after M1 (23:05) to just before the evidence commit (23:22)
  (`gitconfig-after-M1.txt`, `gitconfig-before-M*.txt`,
  `gitconfig-before-evidence.txt`). The shared `.git/config` carries a local
  `[user]` block (`user.name=jay`, `user.email=a@gmail.com`). Its origin is not
  attributable from this run: the first snapshot was taken after M1. The
  pre-commit hook printed `heavy gate not opted in — skipping the project-wide
  steps` on every commit, so the pre-commit `go test` the hazard names did not
  run here. `stat` shows the file rewritten at 23:21:41, inside the
  `make ci-local` window or by another session on the shared repository, and
  that rewrite left the `[user]` and `core.bare` values unchanged. The file was
  not edited by this run.

### Rework after card review

Source: `.moai/reports/t15/card-review.md` (four confirmed findings). Cycle:
TDD, reproduction first. Worked in the isolated agent worktree
`.claude/worktrees/agent-a496d6cfbd37ce242` (branch
`worktree-agent-a496d6cfbd37ce242`, fast-forwarded to `2f1fd5a`) under the
coordinator's explicit approval, for a fast-forward onto `WT-anki-swift-arrow`.
Evidence files sit under `.moai/state/verify/t15-rework/` in that tree, which
is disposed with the agent worktree; the verbatim quotes below are the durable
copy.

Commit: `53745af` — `fix(anki): close the t15 card-review findings for swift
arrow cards` (all four findings; one rework commit). This record was first
committed in `f2076e2`; the two extra list rows and this record's corrections
follow in one later commit.

**Finding 1 (high) — keyword attached to a suppressed result.** `constructEnd`
now mirrors go-org v1.9.1 `parseKeyword` (`org/keyword.go`): `#+NAME:` attaches
the next construct; a run of `#+CAPTION:`/`#+ATTR_HTML:` attaches the construct
after the run, and any other keyword in the run leaves the first one standing
alone; every other keyword (`#+ATTR_LATEX:` included) is one line. A blank line
after the keyword opens a paragraph that runs on through the text below it.
Seven mirror cases added to `swiftMirrorCases` (five-row keyword table plus two
end-of-body converses). RED (`red-mirror.txt`, `red-agree.txt`), verbatim:

```
--- FAIL: TestSwiftMirroredRules/a_name_keyword_across_a_blank_joins_the_result (0.00s)
    swift_test.go:322: arrow lines = [["도쿄" "일본"]], want 0
--- FAIL: TestSwiftMirroredRules/a_caption_keyword_across_a_blank_joins_the_result (0.00s)
    swift_test.go:322: arrow lines = [["도쿄" "일본"]], want 0
--- FAIL: TestSwiftMirroredRules/a_name_then_a_caption_join_the_result (0.00s)
    swift_test.go:322: arrow lines = [["도쿄" "일본"]], want 0
--- FAIL: TestSwiftMirroredRules/an_attr_latex_keyword_is_the_whole_result (0.00s)
    swift_test.go:322: arrow lines = [], want 1
--- FAIL: TestSwiftScannerAgreesWithTheParser/a_name_keyword_across_a_blank_joins_the_result (0.00s)
    swift_test.go:404: go-org renders "도쿄 :-> 일본" in place = false, scanner count = 1
--- FAIL: TestSwiftScannerAgreesWithTheParser/an_attr_latex_keyword_is_the_whole_result (0.00s)
    swift_test.go:404: go-org renders "도쿄 :-> 일본" in place = true, scanner count = 0
```

The `swift_test.go` line numbers in both RED blocks are from the first RED run,
before the two end-of-body converse rows were inserted above the test bodies;
the same assertions sit at 324 and 406 in `53745af`, and at 326 and 408 after
the two list rows added in the follow-up commit.

The `#+ATTR_LATEX:` row was a pre-existing over-exclusion (the paragraph
fall-back swallowed the line after it); the oracle renders it in place, and it
now agrees. `#+CAPTION:` then `#+NAME:` renders in place (count 1), as read from
the source. The oracle agreed with the source reading on every row; no
disagreement to record.

**Finding 2 (high) — block nested in a list item.** `walkList` (shared with the
multiline scan) now consumes a paired `#+BEGIN_` block or LaTeX environment
opened on an item line whole, as one continuation, as go-org's `parseBlock` and
`parseLatexBlock` do (their stop ignores the item's). Four mirror cases (the
result-list body from the review, the same nest outside a result, a LaTeX
environment in a list, and the unclosed-block converse) plus one multiline
corpus row (`a_nested_block_with_a_shallow_line_stays_in_its_item`, item count
only). RED, verbatim:

```
--- FAIL: TestSwiftMirroredRules/a_nested_block_in_a_result_list_is_consumed_whole (0.00s)
    swift_test.go:322: arrow lines = [["도쿄" "일본"]], want 0
--- FAIL: TestSwiftMirroredRules/a_nested_block_in_a_list_is_consumed_whole (0.00s)
    swift_test.go:322: arrow lines = [["도쿄" "일본"]], want 0
--- FAIL: TestSwiftMirroredRules/a_nested_environment_in_a_list_is_consumed_whole (0.00s)
    swift_test.go:322: arrow lines = [["도쿄" "일본"]], want 0
--- FAIL: TestSwiftScannerAgreesWithTheParser/a_nested_block_in_a_result_list_is_consumed_whole (0.00s)
    swift_test.go:404: go-org renders "  도쿄 :-> 일본" in place = false, scanner count = 1
--- FAIL: TestMultilineAnswerList/corpus/a_nested_block_with_a_shallow_line_stays_in_its_item (0.00s)
    multiline_test.go:206: scanner counted 1 items, parse counted 2 (collapsed "- A\n  #+BEGIN_EXAMPLE\nx\n  #+END_EXAMPLE\n- B\n")
```

The multiline RED was captured against the `2f1fd5a` `multiline.go`, restored
temporarily and then put back (`red-multiline.txt`). `TestMultilineGoldenCorpus`
passes and `multiline-golden.json` is byte-identical (golden check below).

Two further list rows were added after the rework commit, at review, to pin the
order of the walk's checks; both agreed with go-org on the first run (no RED —
they are converses, not reproductions): a sibling-shaped bullet inside the
nested block opens no item (count 0), and a block opener shallower than the item
ends the list before the nested-block rule is consulted (count 1).

```
--- PASS: TestSwiftScannerAgreesWithTheParser/a_sibling_bullet_inside_a_nested_block_opens_no_item (0.00s)
--- PASS: TestSwiftScannerAgreesWithTheParser/a_shallow_block_opener_ends_the_list (0.00s)
$ go test ./internal/anki/orgdoc -count=1 -cover
ok  	github.com/karohani/imoogi-emacs/internal/anki/orgdoc	0.275s	coverage: 100.0% of statements
```

**Finding 3 (medium) — swift ON path cleared options before refusing.** New
non-writing predicate `imoogi-anki--cloze-type-acceptable-p` runs first on the ON
path; `imoogi-anki--ensure-cloze-type` keeps its contract for every other caller
(it now delegates its judgment to the predicate, then writes `imoogi-Cloze` only
when the type is absent). New ERT case
`imoogi-anki-toggle-swift-on-refuses-a-non-cloze-type-before-clearing-options`
(imoogi-Basic + own `ANKI_DIRECTION: ->`, `yes-or-no-p` answering t). RED
(`red-elisp.txt`), verbatim:

```
(ert-test-failed
 ((should (equal (buffer-string) before)) :form
  (equal
   #("* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Basic\n:ANKI_DIRECTION: nil\n:END:\n\n도쿄 :-> 일본\n"
```

Known limitation 5 above is closed by this change.

**Finding 4 (low) — numbering claim.** README "번호" paragraph and the
`composeArrowLines` comment corrected: changing an arrow keeps every number;
inserting or deleting an arrow line renumbers the arrow lines below it, and
Anki's history follows the number. Prose only; no test.

**GREEN, verbatim key output:**

```
$ go test ./internal/anki/... -count=1 -cover
ok  	.../internal/anki/ankiconnect	2.666s	coverage: 86.2% of statements
ok  	.../internal/anki/hashing	0.902s	coverage: 100.0% of statements
ok  	.../internal/anki/media	1.606s	coverage: 91.3% of statements
ok  	.../internal/anki/model	0.601s	coverage: 98.4% of statements
ok  	.../internal/anki/orgdoc	1.270s	coverage: 100.0% of statements
ok  	.../internal/anki/planner	2.004s	coverage: 92.9% of statements
ok  	.../internal/anki/protocol	3.000s	coverage: [no statements]
ok  	.../internal/anki/registry	2.336s	coverage: 87.8% of statements
$ go test ./internal/anki/orgdoc -count=1 -v -run TestMultilineGoldenCorpus
--- PASS: TestMultilineGoldenCorpus (0.00s)
$ go test ./internal/anki/orgdoc -count=1 -v -run TestSwiftScannerAgreesWithTheParser
--- PASS: TestSwiftScannerAgreesWithTheParser (0.00s)
$ make test-elisp
Ran 659 tests, 648 results as expected, 0 unexpected, 11 skipped
$ make lint        -> exit 0, no output
$ make fmt-check   -> exit 0; its only output is two parse messages from
                      vendored go-mode test fixtures, not project code:
vendor/elpa/go-mode-1.6.0/test/testdata/indentation_tests/composite_literal_key.go:1:1: expected 'package', found 'func'
vendor/elpa/go-mode-1.6.0/test/testdata/indentation_tests/multiline_struct.go:1:1: expected 'package', found 'func'
$ git diff --stat 2f1fd5a -- internal/anki/orgdoc/testdata internal/anki/hashing
(empty)
```

The RED Elisp run also showed the two timing-dependent tests already recorded
under "Observations outside scope"
(`imoogi-project-notes-cli-cache-superseded-process-does-not-callback`,
`imoogi-setup-test-ankiconnect-status-invalid-response-on-http-error`); both
passed in the GREEN run. No swift golden rows were added: the new bodies are
parser-agreement cases, which `swiftMirrorCases` holds; `swift-golden.json` is
unchanged.

**Known limitations added by the rework.**

7. **A source block's result inside a list item across blank lines.** go-org's
   `parseSrcBlockResult` skips any number of blank lines before `#+RESULTS:`,
   while the item loop itself ends at a blank pair; the list walk ends at the
   blank pair. A `#+RESULTS:` so attached inside an item is not mirrored. Out of
   this rework's scope.
8. **The multiline answer span over a nested block** follows the walk's existing
   continuation rule: the span runs over the whole nested block, as it already
   did for a fully indented one. Only the item count is pinned for the
   shallow-interior shape.

**Follow-up for the SPEC owner.** `spec.md` REQ-SW-003.3 reads as though
positional numbering keeps numbers stable when a `:<-` line is inserted above
another; positional numbering keeps them stable when a line's arrow changes,
not when a line is inserted. Not edited here (spec.md body is out of this
agent's scope).

## §E.3 Run-phase Audit-Ready Signal

```yaml
run_complete_at: 2026-10-12
run_commit_sha: c72ef69  # last implementation commit (M5); this evidence commit follows it
run_status: complete
cycle_type: tdd
branch: worktree-agent-a8e14baae19af5a96
base: ed28d1a
milestone_commits: [ea44ff8, 95b8a36, 945eea8, bf076d8, c72ef69]
ac_pass_count: 15
ac_fail_count: 0
preserve_list_post_run_count: 0  # PRESERVE-list files modified
golden_files_modified: []  # render-golden.json and multiline-golden.json unchanged
golden_files_added: [internal/anki/orgdoc/testdata/swift-golden.json]
coverage_after:
  orgdoc: 100.0
  planner: 92.9
  hashing: 100.0
  model: 98.4
elisp_tests: {ran: 658, unexpected: 0, skipped: 11, baseline: 645}
l44_pre_commit_fetch: not-run  # no push; lane-local run
l44_post_push_fetch: not-applicable  # not pushed
new_warnings_or_lints_introduced: 0
cross_platform_build:
  darwin: pass
  windows_anki_packages: pass
  windows_whole_module: fail-preexisting  # internal/setup syscall, at 7d98ec0 too
total_run_phase_files: 23  # 22 in M1-M5 (git diff --name-only ed28d1a c72ef69) + this progress.md
m1_to_mN_commit_strategy: one commit per milestone, M1-M5; M6 (README) deferred to sync
deferred: [M6 README — manager-docs]
```

## §E.4 Sync-phase Audit-Ready Signal

Sync phase, card t15, worktree `.moai/worktrees/t15` (branch
`WT-anki-swift-arrow`, base `5abcc4d`). Docs-only: README § "Anki 화살표 카드"
(milestone M6, deferred from run) plus the intro line of § "Anki 카드 옵션
속성", and the t15 row of `.moai/reports/anki-card-types-plan-20260920.md`.
Logs under `.moai/state/verify/t15-sync/` (ignored by git).

```yaml
sync_complete_at: 2026-10-13
sync_commit_sha: see `git log -1` on WT-anki-swift-arrow  # this commit; a file cannot name its own SHA
sync_status: complete
m6_readme: done
changelog: not-applicable  # repository has no CHANGELOG.md; none created
b12_self_test_a: not-applicable  # no CHANGELOG to grep
b12_self_test_b: {acceptance_distinct_ac_ids: 15}  # grep -oE 'AC-([A-Z0-9]+-)*[0-9]+' acceptance.md | sort -u | wc -l
b12_self_test_c: not-applicable  # no CHANGELOG entry
changelog_entry_position: not-applicable
frontmatter_status_transitions:
  spec.md: "in-progress -> implemented -> completed (updated: 2026-10-13)"
  plan.md: no-frontmatter
  acceptance.md: no-frontmatter
  progress.md: no-frontmatter
verification:
  go_test_anki: "go test ./internal/anki/... -count=1 -> exit 0, 8 packages ok"
  elisp: "make test-elisp -> Ran 658 tests, 647 results as expected, 0 unexpected, 11 skipped"
  spec_lint: "moai spec lint spec.md -> 0 error(s), 0 warning(s) (1 INFO OwnershipTransitionUnmeasured)"
pushed: false
```
