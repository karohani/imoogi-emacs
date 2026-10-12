# t15 card-review — SPEC-ANKICARD-004

- Tree: `.moai/worktrees/t15`, branch `WT-anki-swift-arrow`, HEAD `2f1fd5a`
- Tool: `codex_review scope=card` → **inconclusive** ("no integration target configured": the worktree has no `.moai/config/sections/git-strategy.yaml`; that file is untracked in the primary checkout). Fallback: read-only `codex_task` review of `git diff 7d98ec0..HEAD -- . ':(exclude).moai'` (job `job-20261012T165400Z-40082e8f`). Advisory, non-binding.
- Codex's result was static (go-org v1.9.1 source comparison, no tests run). The lane reproduced each finding before accepting it.

## Findings and lane reproduction

| # | Sev | Finding | Lane verification | Status |
|---|---|---|---|---|
| 1 | high | `swift.go` `constructEnd` stops at the blank after a `#+NAME:` attached to a suppressed `#+RESULTS:`; go-org suppresses the following paragraph, so the generated cloze is invisible | Temporary test calling `scanArrowLines` + the package's `renderedInPlace` oracle on body `#+BEGIN_SRC sh :exports code⏎echo⏎#+END_SRC⏎#+RESULTS:⏎#+NAME: r⏎⏎도쿄 :-> 일본` → `scanner count=1, go-org renders arrow line in place=false` (DISAGREE) | CONFIRMED |
| 2 | high | `walkList` (`multiline.go`) ends a list at the unindented line inside a nested `#+BEGIN_EXAMPLE`; the arrow after the block is admitted though go-org keeps it inside the suppressed result list | Same harness, body `…#+RESULTS:⏎- item⏎  #+BEGIN_EXAMPLE⏎x⏎  #+END_EXAMPLE⏎  도쿄 :-> 일본` → `scanner count=1, … in place=false` (DISAGREE) | CONFIRMED |
| 3 | medium | `imoogi-anki-toggle-swift` ON path runs `imoogi-anki--resolve-multiline-conflict` (clears multiline options) before `imoogi-anki--ensure-cloze-type` refuses a non-cloze type | Code read (`modules/org/24-anki.el`, the `(when (and (imoogi-anki--resolve-multiline-conflict …) (imoogi-anki--ensure-cloze-type)) …)` form) against REQ-SW-013.3 "any other type is reported and nothing is changed" | CONFIRMED (spec violation; previously disclosed by run as a known limitation, sync-audit F4) |
| 4 | low | README claims inserting an arrow line does not shift other lines' numbers | README "번호" paragraph vs REQ-SW-003 positional numbering (line i → c(2i-1)/c(2i)): inserting a line before line k shifts k's numbers | CONFIRMED |

The temporary test file was deleted after the run; the tree is clean.

## Disposition

All four are fixed in a run-phase rework (reproduction tests first), then the sync audit is re-run. Findings 1 and 2 are the SPEC's own worst failure mode (a composed marker the card never shows), so the card does not close with them open.
