# Sync Audit, pass 2 — SPEC-ANKICARD-004 (Swift arrow card rendering), card t15

- Auditor: sync-auditor (independent, Claude-only; no cross-model backend called — see Gaps)
- Date: 2026-10-13
- Tree: worktree `.moai/worktrees/t15`, branch `WT-anki-swift-arrow`, HEAD `e90f269`
- Scope: the post-review rework only — `git diff 2f1fd5a..HEAD` (`53745af`, `f2076e2`, `e90f269`; 8 files, +326/−9). Pass 1 (`sync-audit.md`, HEAD `2f1fd5a`) is not re-scored. This pass checks the four card-review findings (`card-review.md`) and regressions.
- Profile: `default` (harness `default_profile`), flat scoring — same as pass 1.
- Temporary probes: one Go test (`internal/anki/orgdoc/zz_audit_probe_test.go`) and one ERT file (`tests/zz-audit-probe-test.el`). Both were deleted after the run. `git status --short` afterwards printed only `?? .codex/` and `?? .moai/reports/t15/`. Tracked files are unchanged.

## Evaluation Report

SPEC: SPEC-ANKICARD-004
**Overall Verdict: PASS**

### Dimension Scores

| Dimension | Score | Verdict | Evidence (verbatim) |
|---|---|---|---|
| Functionality (40%) | 92/100 | PASS | All four findings are fixed and pinned by tests (§ Findings 1–4). `go test ./internal/anki/orgdoc/ -run 'TestSwiftMirroredRules\|TestSwiftScannerAgreesWithTheParser\|TestMultilineGoldenCorpus\|TestMultiline' -count=1` printed `ok  github.com/karohani/imoogi-emacs/internal/anki/orgdoc 0.260s`. I ran 37 probes the rework did not list against the `renderedInPlace` go-org oracle. None produced a hidden marker. Three admit a line that go-org renders visibly inside a list item (F1). Four withhold a card (F2). ERT: `Ran 659 tests, 648 results as expected, 0 unexpected, 11 skipped`. |
| Security (25%) | 100/100 | PASS | The rework adds no snippet, no URL and no I/O surface. The only new pattern is `keywordLinePattern = regexp.MustCompile(`^(\s*)#\+([^:]+):(\s+(.*)\|$)`)`, which feeds a line-index computation and nothing else. No line text reaches an `@@html:` snippet. OWASP: no reachable category changed since pass 1. |
| Craft (20%) | 95/100 | PASS | `orgdoc … coverage: 100.0% of statements`, `planner … coverage: 92.9% of statements`, `hashing … coverage: 100.0% of statements`. The rework's ten new mirror rows plus one multiline row all `--- PASS`. Each finding was reproduced first (progress §E.2 records the RED output). The finding-3 ERT case is a real reproducer: the old order would delete `ANKI_DIRECTION` before refusing. |
| Consistency (15%) | 92/100 | PASS | `make lint` (= `go vet ./...`) exit 0 with 0 output lines. `make fmt-check` exit 0. `nestedConstructEnds` reuses the package's single pairing rule for each delimiter (`pairedBlocks`, `latexEnvironments`) and does not copy it. `keywordEnd` cites go-org `parseKeyword`. README and code comment now match positional numbering. `spec.md` REQ-SW-003.3 still says the opposite (F3). |

Must-pass firewall: Functionality PASS (the four findings are closed; no hidden-marker disagreement). Security PASS (no Critical or High finding). Both hold.

Scoring arithmetic:
- Weighted: 0.40·0.92 + 0.25·1.00 + 0.20·0.95 + 0.15·0.92 = **0.946**
- Harmonic mean: 4 / (1/0.92 + 1/1.00 + 1/0.95 + 1/0.92) = 4 / 4.2266 = **0.946**

What drives the verdict: all four card-review findings are fixed, regressions are clean, and there are no blocking findings. F1–F4 below are optional new findings. F1 is in tension with REQ-SW-001.5, so it is surfaced explicitly and the orchestrator can decide whether to route it.

## Check 1 — Findings 1 and 2 (scanner vs go-org)

### Rework tests

```
go test ./internal/anki/orgdoc/ -run 'TestSwiftMirroredRules|TestSwiftScannerAgreesWithTheParser|TestMultilineGoldenCorpus' -count=1 -v
    --- PASS: TestSwiftMirroredRules/a_caption_keyword_across_a_blank_joins_the_result (0.00s)
    --- PASS: TestSwiftMirroredRules/a_name_then_a_caption_join_the_result (0.00s)
    --- PASS: TestSwiftMirroredRules/an_attr_latex_keyword_is_the_whole_result (0.00s)
    --- PASS: TestSwiftMirroredRules/a_caption_then_a_name_is_the_whole_result (0.00s)
    --- PASS: TestSwiftMirroredRules/a_name_keyword_as_the_last_line_attaches_nothing (0.00s)
    --- PASS: TestSwiftMirroredRules/a_caption_run_as_the_last_lines_attaches_nothing (0.00s)
    --- PASS: TestSwiftMirroredRules/a_nested_block_in_a_result_list_is_consumed_whole (0.00s)
    --- PASS: TestSwiftMirroredRules/a_nested_block_in_a_list_is_consumed_whole (0.00s)
    --- PASS: TestSwiftMirroredRules/a_nested_environment_in_a_list_is_consumed_whole (0.00s)
    --- PASS: TestSwiftMirroredRules/an_unclosed_nested_block_ends_the_list_at_the_shallow_line (0.00s)
    --- PASS: TestSwiftMirroredRules/a_sibling_bullet_inside_a_nested_block_opens_no_item (0.00s)
    --- PASS: TestSwiftMirroredRules/a_shallow_block_opener_ends_the_list (0.00s)
    --- PASS: TestSwiftScannerAgreesWithTheParser/a_name_keyword_across_a_blank_joins_the_result (0.00s)
    --- PASS: TestSwiftScannerAgreesWithTheParser/a_caption_keyword_across_a_blank_joins_the_result (0.00s)
    --- PASS: TestSwiftScannerAgreesWithTheParser/a_nested_block_in_a_result_list_is_consumed_whole (0.00s)
    --- PASS: TestSwiftScannerAgreesWithTheParser/a_nested_block_in_a_list_is_consumed_whole (0.00s)
    --- PASS: TestSwiftScannerAgreesWithTheParser/an_unclosed_nested_block_ends_the_list_at_the_shallow_line (0.00s)
    --- PASS: TestSwiftScannerAgreesWithTheParser/a_sibling_bullet_inside_a_nested_block_opens_no_item (0.00s)
    --- PASS: TestSwiftScannerAgreesWithTheParser/a_shallow_block_opener_ends_the_list (0.00s)
PASS
ok  	github.com/karohani/imoogi-emacs/internal/anki/orgdoc	0.262s
```

Both card-review reproduction bodies are now pinned at count 0 and agree with go-org:
- `a_name_keyword_across_a_blank_joins_the_result` is finding 1.
- `a_nested_block_in_a_result_list_is_consumed_whole` is finding 2.

I also read go-org v1.9.1 directly (`parseBlock`, `parseLatexBlock`, `parseKeyword`, `parseAffiliated`, `parseNodeWithName`, `parseListItem`, `parseDrawer`):
- The block and environment stop functions ignore `parentStop`.
- `parseDrawer` honours it.

The fifth walk rule is therefore correct to cover blocks and environments only.

### Independent probes (temporary `_test.go`, deleted)

Each probe compares `len(scanArrowLines(swiftTitle, body))` with `renderedInPlace(body, arrowLine)`. Here `r` = `#+BEGIN_SRC sh :exports code⏎echo⏎#+END_SRC⏎#+RESULTS:⏎`.

Command: `go test ./internal/anki/orgdoc/ -run TestZZAuditProbe -count=1 -v`. Output (verbatim; go-org fallback log lines omitted):

```
P01_name_two_blanks                              scanner=1 goorg_in_place=true  AGREE
P02_caption_then_table                           scanner=0 goorg_in_place=true  DISAGREE-WITHHELD(optional-class)
P03_caption_attr_then_table                      scanner=0 goorg_in_place=true  DISAGREE-WITHHELD(optional-class)
P04_caption_table_blank                          scanner=1 goorg_in_place=true  AGREE
P05_deeper_subitem_block                         scanner=0 goorg_in_place=false AGREE
P06_deeper_subitem_block_arrow_item_level        scanner=0 goorg_in_place=false AGREE
P07_deeper_subitem_in_result_list                scanner=0 goorg_in_place=false AGREE
P08_unclosed_in_item_indented_arrow              scanner=0 goorg_in_place=false AGREE
P09_unclosed_in_item_col0_arrow                  scanner=1 goorg_in_place=true  AGREE
P10_unclosed_in_result_list                      scanner=0 goorg_in_place=false AGREE
P11_block_on_bullet_line                         scanner=1 goorg_in_place=false DISAGREE-INVISIBLE(blocking-class)
P12_env_on_bullet_line                           scanner=1 goorg_in_place=false DISAGREE-INVISIBLE(blocking-class)
P13_nested_close_after_blank_pair_col0_arrow     scanner=1 goorg_in_place=true  AGREE
P14_nested_close_col0_indented_arrow             scanner=0 goorg_in_place=false AGREE
P15_drawer_in_item_shallow                       scanner=1 goorg_in_place=true  AGREE
P16_name_no_results_blank                        scanner=1 goorg_in_place=true  AGREE
P17_name_no_results_direct                       scanner=1 goorg_in_place=true  AGREE
P18_name_name                                    scanner=0 goorg_in_place=false AGREE
P19_name_list_col0_arrow                         scanner=1 goorg_in_place=true  AGREE
P20_attr_html_blank                              scanner=0 goorg_in_place=false AGREE
P21_caption_block                                scanner=1 goorg_in_place=true  AGREE
P22_lowercase_name_blank                         scanner=0 goorg_in_place=false AGREE
P23_empty_name_blank                             scanner=0 goorg_in_place=false AGREE
P24_indented_name_blank                          scanner=0 goorg_in_place=false AGREE
P25_quote_in_item_bullet_inside                  scanner=0 goorg_in_place=false AGREE
P26_ordered_nested_block                         scanner=0 goorg_in_place=false AGREE
P27_caption_two_blanks                           scanner=1 goorg_in_place=true  AGREE
P28_name_attr_latex                              scanner=1 goorg_in_place=true  AGREE
P29_name_blank_text_blank_arrow                  scanner=1 goorg_in_place=true  AGREE
P30_name_src_with_result                         scanner=0 goorg_in_place=true  DISAGREE-WITHHELD(optional-class)
P31_nested_block_then_shallow_text_then_arrow    scanner=1 goorg_in_place=true  AGREE
P32_nested_src_result_shallow_example            scanner=0 goorg_in_place=false AGREE
P33_caption_name_blank                           scanner=1 goorg_in_place=true  AGREE
P34_name_fixed_width                             scanner=0 goorg_in_place=true  DISAGREE-WITHHELD(optional-class)
P35_two_sub_blocks_second_unclosed               scanner=1 goorg_in_place=true  AGREE
P36_result_list_nested_env                       scanner=0 goorg_in_place=false AGREE
P37_result_list_block_on_bullet_line             scanner=1 goorg_in_place=false DISAGREE-INVISIBLE(blocking-class)
--- PASS: TestZZAuditProbe (0.00s)
```

The four variants named in the brief all agree with go-org:
- P01: `#+NAME` then two blanks.
- P05–P07: a block in a deeper sub-item.
- P08–P10: an unclosed `#+BEGIN_` in an item followed by an arrow line.
- P02/P03 (a `#+CAPTION` run followed by a table) is the exception: it disagrees, but in the withheld direction.

**The probe's "INVISIBLE" label on P11/P12/P37 is wrong, and I corrected it by reading the HTML.** `renderedInPlace` tests only for a *top-level* `<p>`. In these three bodies go-org does render the arrow line, visibly, inside a list item. The composed cloze is intact (`TestZZAuditProbeHTML`, verbatim):

```
BODY "- #+BEGIN_EXAMPLE\nx\n  #+END_EXAMPLE\n  도쿄 :-> 일본"
 scanner=1 in_place=false
 HTML(composed)=<ul>
<li>
<pre class="example">
x
</pre>
<p>도쿄 <b class="swift-arrow">:-&gt;</b> {{c1::일본}}</p>
</li>
</ul>
BODY "- \\begin{align}\nx\n  \\end{align}\n  도쿄 :-> 일본"
 scanner=1 in_place=false
 HTML(composed)=<ul> <li> \begin{align} x \end{align} <p>도쿄 <b class="swift-arrow">:-&gt;</b> {{c1::일본}}</p> </li> </ul>
```

No hidden marker therefore exists. It is a list-placement admission, reported as F1.

The withheld direction (P02/P03/P30/P34, plus the unaffiliated baselines `r + ": out⏎도쿄 :-> 일본"` and `r + "| a |⏎도쿄 :-> 일본"`, both `scanner=0 in_place=true`) contradicts no corpus row. A grep of `swift_test.go` for `RESULTS:` finds no row whose result construct is a table or a fixed-width line. These are reported as F2.

## Check 2 — walkList change preserved multiline behaviour

```
go test ./internal/anki/orgdoc -run '^TestMultilineGoldenCorpus$' -count=1 -v 2>&1 | grep -E '^(--- (PASS|FAIL)|ok|FAIL)'
--- PASS: TestMultilineGoldenCorpus (0.00s)
ok  	github.com/karohani/imoogi-emacs/internal/anki/orgdoc	0.542s

go test ./internal/anki/orgdoc/ -run 'TestSwiftMirroredRules|TestSwiftScannerAgreesWithTheParser|TestMultilineGoldenCorpus|TestMultiline' -count=1
ok  	github.com/karohani/imoogi-emacs/internal/anki/orgdoc	0.260s

git diff --stat 7d98ec0..HEAD -- internal/anki/orgdoc/testdata/multiline-golden.json internal/anki/orgdoc/testdata/render-golden.json internal/anki/hashing
(no output)  golden_diff_exit=0

git diff --stat 2f1fd5a..HEAD -- internal/anki/orgdoc/multiline_golden_test.go internal/anki/planner internal/anki/orgdoc/testdata
(no output)  rework_golden_exit=0
```

- `multiline-golden.json`, `render-golden.json`, and the `hashing` package are unchanged since `7d98ec0`.
- The rework touches no golden file, no golden test, and nothing under `planner`.
- The 11-line `multiline_golden_test.go` change visible in `7d98ec0..HEAD` predates the rework, so it falls in pass 1's scope.
- The rework adds one multiline corpus row, `a_nested_block_with_a_shallow_line_stays_in_its_item` (`found: true, items: 2`), which passes.

## Check 3 — Finding 3 (Elisp note-type order)

Code read of `modules/org/24-anki.el`:
- `imoogi-anki--ensure-cloze-type` is now `(when (imoogi-anki--cloze-type-acceptable-p) (unless (org-entry-get (point) "ANKI_NOTE_TYPE") (org-set-property …)) t)`.
- `imoogi-anki--cloze-type-acceptable-p` returns t for a nil or cloze-style type. For any other type it messages and returns nil, and it writes nothing.
- The old `cond` gave the same three outcomes (nil type → write Cloze, t; cloze → t; other → message, nil), so the refactor preserves behaviour for every caller.

Callers:
- `imoogi-anki--write-card-option` (line 395) is used by `imoogi-anki-set-direction` and `imoogi-anki-toggle-incremental`.
- The swift ON path (lines 518–520) now runs `acceptable-p`, then `resolve-multiline-conflict`, then `ensure-cloze-type`.
- On a refused type, the swift path messages exactly once, because `acceptable-p` short-circuits the `and`.

`make test-elisp` exit 0. Verbatim excerpt:

```
   passed   38/659  imoogi-anki-set-direction-clear-writes-the-falsy-spelling (0.002709 sec)
   passed   39/659  imoogi-anki-set-direction-confirming-the-swift-conflict-clears-swift (0.003223 sec)
   passed   40/659  imoogi-anki-set-direction-declining-the-swift-conflict-writes-nothing (0.000790 sec)
   passed   41/659  imoogi-anki-set-direction-follows-the-note-type-contract (0.004910 sec)
   passed   42/659  imoogi-anki-set-direction-writes-the-chosen-arrow (0.002584 sec)
   passed   45/659  imoogi-anki-toggle-incremental-also-honours-the-swift-conflict (0.000803 sec)
   passed   46/659  imoogi-anki-toggle-incremental-turns-an-absent-option-on (0.002595 sec)
   passed   47/659  imoogi-anki-toggle-incremental-turns-an-inherited-option-off (0.002767 sec)
   passed   54/659  imoogi-anki-toggle-swift-on-refuses-a-non-cloze-type (0.000847 sec)
   passed   55/659  imoogi-anki-toggle-swift-on-refuses-a-non-cloze-type-before-clearing-options (0.000832 sec)
Ran 659 tests, 648 results as expected, 0 unexpected, 11 skipped (2026-10-13 02:26:02+0900, 50.454981 sec)
```

The new case stubs `yes-or-no-p` to t. It asserts byte-identical buffer text, the imoogi-Basic refusal message, and that `ANKI_DIRECTION` is kept, so the ON path writes nothing on a non-cloze type.

## Check 4 — Finding 4 (numbering wording)

- README now reads `이미 있는 줄의 화살표 방향을 바꿔도 어느 줄의 번호도 바뀌지 않는다. 반면 화살표 줄을 중간에 끼우거나 지우면 그 아래 화살표 줄의 번호가 모두 밀린다.`
- The `composeArrowLines` comment now reads `changing a line's arrow renumbers no line … Inserting or deleting an arrow line does renumber every arrow line below it`.

Both match positional numbering (line i → c(base+2i−1)/c(base+2i)). `spec.md` REQ-SW-003.3 (lines 759–764) still implies the opposite (F3). progress §E.2 already flags it for the SPEC owner.

## Check 5 — Regression

```
go test ./internal/anki/... -count=1 -cover
ok  	github.com/karohani/imoogi-emacs/internal/anki/ankiconnect	0.563s	coverage: 86.2% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/hashing	0.855s	coverage: 100.0% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/media	1.224s	coverage: 91.3% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/model	1.952s	coverage: 98.4% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/orgdoc	1.566s	coverage: 100.0% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/planner	2.648s	coverage: 92.9% of statements
ok  	github.com/karohani/imoogi-emacs/internal/anki/protocol	2.237s	coverage: [no statements]
ok  	github.com/karohani/imoogi-emacs/internal/anki/registry	3.055s	coverage: 87.8% of statements

make lint            -> lint_exit=0, 0 output lines   (Makefile:172-173 — lint is `go vet ./...` only)
make fmt-check       -> fmt_exit=0; stderr tail (vendor noise, filtered by the target's grep):
vendor/elpa/go-mode-1.6.0/test/testdata/indentation_tests/composite_literal_key.go:1:1: expected 'package', found 'func'
vendor/elpa/go-mode-1.6.0/test/testdata/indentation_tests/multiline_struct.go:1:1: expected 'package', found 'func'
make test-elisp      -> exit=0; Ran 659 tests, 648 results as expected, 0 unexpected, 11 skipped
```

All thresholds are met: orgdoc 100.0%, planner 92.9% (≥ 92.9%). Lint, fmt-check, and test-elisp are all clean.

## Check 6 — Commit hygiene (`git log --format='%H%n%an <%ae>%n%B' 2f1fd5a..HEAD`)

| Commit | Subject | Card id `t15` | `🗿 MoAI` | `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` |
|---|---|---|---|---|
| `53745af` | `fix(anki): close the t15 card-review findings for swift arrow cards` | subject + body (`Card t15, SPEC-ANKICARD-004 rework…`) | yes | yes |
| `f2076e2` | `docs(SPEC-ANKICARD-004): record the t15 card-review rework evidence` | subject + body | yes | yes |
| `e90f269` | `test(anki): pin the list walk's check order for nested blocks` | body (`Card t15, SPEC-ANKICARD-004 rework follow-up`) | yes | yes |

All three are Conventional Commits, name the card id, and carry the same trailer set as `2f1fd5a`. The author is `jay <a@gmail.com>`, the same identity as the earlier t15 commits.

## Findings (structured defect-list)

The four card-review findings are **closed**:

| Card-review # | Status | Deciding evidence |
|---|---|---|
| 1 (NAME + blank after suppressed result) | FIXED | `--- PASS: TestSwiftScannerAgreesWithTheParser/a_name_keyword_across_a_blank_joins_the_result`; probes P01, P16–P29, P33 agree |
| 2 (shallow line inside nested block in a list) | FIXED | `--- PASS: …/a_nested_block_in_a_result_list_is_consumed_whole`; probes P05–P10, P13, P14, P25, P26, P31, P32, P35, P36 agree |
| 3 (swift ON clears multiline options before refusing) | FIXED | `passed 55/659 imoogi-anki-toggle-swift-on-refuses-a-non-cloze-type-before-clearing-options` |
| 4 (README numbering claim) | FIXED | README + `composeArrowLines` comment text quoted in Check 4 |

New findings (none blocking):

- **F1 [medium, confidence high] [optional]** `internal/anki/orgdoc/multiline.go` `walkList`/`nestedConstructEnds`.
  - **Classification.** Optional under this pass's blocking criterion: go-org renders the line, and the HTML cited in Check 1 shows no hidden marker. It is still a REQ-SW-001.5 list-content miss, so it must be fixed or disclosed. It must not be left silent.
  - **What happens.** A `#+BEGIN_` block or LaTeX environment opened *on the bullet line itself* (`- #+BEGIN_EXAMPLE`, `- \begin{align}`) is not mirrored. go-org `parseListItem` re-tokenizes the item content (`d.tokens[i] = tokenize(strings.Repeat(" ", minIndent) + content)`), so the block opens there and swallows its shallow interior. The scanner's patterns are anchored at line start (`beginBlockLinePattern = (?i)^\s*#\+BEGIN_(\w+)`, `beginLatexLinePattern = (?i)^(\s*)\\begin{([^}]+)}(\s*)$`), so they never match a bullet line. The walk then ends the list at the shallow line, and the indented arrow line below is admitted (P11, P12, P37).
  - **Effect.** The cloze is composed and visible, but it sits inside `<li>`, a placement that REQ-SW-001.5/§ 5 excludes. It is not a hidden marker.
  - **Origin.** Pre-existing: the rework did not touch this shape, and the pre-rework walk also ended at the shallow line. This was inferred from the code and not re-executed at `2f1fd5a`.
  - **Required fix:** in `nestedConstructEnds`, also map a bullet line whose content after the bullet matches `beginBlockLinePattern`/`beginLatexLinePattern` to its matching close (go-org's retokenize), and add P11/P12 as mirror rows. Alternatively, pin the shape in `swiftDisclosedDivergences` and add it to progress Known limitations.
- **F2 [low, confidence high] [optional]** `internal/anki/orgdoc/swift.go` `constructEnd`.
  - **What happens.** When a suppressed result's construct is a fixed-width line (`: out`) or a table (`| a |`), it is read through the paragraph fallback. The fallback runs on through the arrow line below, so the card is withheld while go-org renders the arrow in place. This covers P02, P03, P30, P34, and the plain `#+RESULTS:⏎: out⏎도쿄 :-> 일본` baseline.
  - **Why it matters anyway.** `: out` is the usual org-babel result form, so a user can hit this withholding. It never hides a marker, and no corpus row contradicts it.
  - **Required fix:** add fixed-width and table runs to `constructEnd`, or list the case as a known limitation.
- **F3 [low, confidence high] [optional]** `.moai/specs/SPEC-ANKICARD-004/spec.md:759-764` REQ-SW-003.3 still says positional numbering prevents renumbering when a `:<-` line is inserted above another. That contradicts the corrected README and comment.
  - **Required fix:** reword it to "changing a line's arrow renumbers nothing; inserting or deleting an arrow line renumbers the lines below". progress §E.2 already flags this for the SPEC owner.
- **F4 [low, confidence high, verified] [optional — out of t15 scope]** `modules/org/24-anki.el:388-397` `imoogi-anki--write-card-option` runs `resolve-swift-conflict` (which writes `ANKI_SWIFT: nil` on "yes") before `ensure-cloze-type` refuses a non-cloze type. This is the mirror image of card-review finding 3, on `set-direction`/`toggle-incremental`.
  - **Verification.** A temporary ERT probe (deleted) failed twice with `Ran 2 tests, 0 results as expected, 2 unexpected`:
    - Own swift case: the buffer went from `:ANKI_SWIFT: t` to `:ANKI_SWIFT: nil` while `:ANKI_NOTE_TYPE: imoogi-Basic` was refused.
    - Inherited-swift case: `:ANKI_SWIFT: nil` was added to the child drawer.
  - **Scope.** `git log -S resolve-swift-conflict` shows that the helper dates from `08f7504` (SPEC-ANKICARD-003). REQ-SW-013.3 states that this SPEC "changes nothing about the existing commands".
  - **Required fix (new card, not t15):** in `--write-card-option`, call `imoogi-anki--cloze-type-acceptable-p` before `resolve-swift-conflict`, the same order the swift toggle now uses.

## Recommendations

- F1 is the one finding that touches a SPEC-stated rule. It needs either a one-function fix with two mirror rows, or an explicit disclosure. Either is acceptable. Leaving it silent is not.
- File F4 as a follow-up card against SPEC-ANKICARD-003 behaviour.
- Fold F2 and F3 into progress Known limitations and the SPEC owner's follow-up.

## Gaps (not observed)

- **Oracle limit.** `renderedInPlace` checks for a top-level `<p>` only. It cannot tell "rendered inside a list item" apart from "suppressed". I read the HTML by hand only for P11, P12, and the `: out` / table baselines.
- **Pre-rework baseline for F1.** Behaviour at `2f1fd5a` was inferred from the code, which has no bullet-line opener handling in either version. It was not re-executed, because writing git commands are out of scope.
- **Probe coverage.** I ran 37 hand-chosen shapes. This is not an exhaustive or generative search.
- **Lint scope.** `golangci-lint` was not run. `make lint` is `go vet` only.
- **Cross-model second opinion.** No `audit_model` key exists in `.moai/config/sections/*.yaml` (grep returned nothing), and pass 1 ran Claude-only. No `audit_multi`/`codex_audit`/`glm_audit` call was made.
- **Live Anki and Emacs session.** No live round-trip was run. The evidence is unit, ERT, and go-org rendering only.
- **Evidence logs.** The logs live in the session scratchpad (`…/scratchpad/elisp.log`, `fmt.log`, `lint.log`, `elisp-probe.log`), which is outside the repo and ephemeral. The verbatim lines quoted here are the attributable evidence.

## Residual risk

- F1 also affects the multiline answer span for the same bullet-line shape, because `walkList` is shared. I did not measure this.
- The keyword mirror covers go-org v1.9.1's `parseKeyword` switch. `SETUPFILE`/`INCLUDE` load or include content. Probes do not cover them, though they are unlikely inside a suppressed result.
- A go-org version bump would invalidate every mirrored rule. The parser-agreement test is the tripwire.
