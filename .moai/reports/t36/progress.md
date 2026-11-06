# t36 — card option written before the note-type refusal

- card: t36 (Class B — defect, reproduction first; leader dispatch, factory `assigned` → lane-5)
- worktree: .moai/worktrees/t36, branch WT-card-option-refuse (from origin/main 6feee71)
- scope stop: worktree commit only — no push, PR, main merge, or primary-checkout edit
- plan stage skipped per dispatch (Class B run → sync)

## Cause (established by reproduction)

`imoogi-anki--write-card-option` (modules/org/24-anki.el), shared by
`imoogi-anki-set-direction` and `imoogi-anki-toggle-incremental`, ran
`imoogi-anki--resolve-swift-conflict` first. On a "yes" that writes `ANKI_SWIFT: nil`.
Only afterwards did `imoogi-anki--ensure-cloze-type` refuse a non-cloze ANKI_NOTE_TYPE
(e.g. imoogi-Basic), so a refused heading still had its swift switched off. Same shape as
t15 finding 3, which `imoogi-anki-toggle-swift` already fixed by judging the type first.

Reproduction (RED, before the fix):

    IMOOGI_ERT_SELECTOR='refuses-a-non-cloze-type-before-clearing-swift' Emacs --batch -Q -l tests/run.el
    FAILED imoogi-anki-set-direction-refuses-a-non-cloze-type-before-clearing-swift
      (should (equal (buffer-string) before)) failed
    FAILED imoogi-anki-toggle-incremental-refuses-a-non-cloze-type-before-clearing-swift
    Ran 2 tests, 0 results as expected, 2 unexpected

The message assertion (names imoogi-Basic) passed in both RED runs; only the whole-buffer
equality failed. That rules out "the refusal never happened" and isolates the write order.

## Fix

`imoogi-anki--write-card-option`: order is now type judgment
(`imoogi-anki--cloze-type-acceptable-p`, writes nothing) → swift-conflict resolution →
`imoogi-anki--ensure-cloze-type` (adds imoogi-Cloze to an untyped heading). The
"declining swift leaves no note type" property (AC-ML-012d) is kept because note-type
writing still follows the swift check.

Tests (tests/anki-commands-test.el): macro `imoogi-anki-test--refused-unchanged` + two
ERTs — imoogi-Basic heading with ANKI_SWIFT t, prompt answered "yes", assert the message
names the type, the buffer is byte-identical, and ANKI_SWIFT stays `t`.

## Verification (after the fix)

- full ERT: `IMOOGI_TEST_USER_DIR=<scratch> Emacs --batch -Q -l tests/run.el` → exit 0,
  "Ran 719 tests, 708 results as expected, 0 unexpected, 11 skipped" (includes boot-health)
- check-parens over early-init/boot/packages + modules/ + tests/ → ok
- `git config --get core.bare` → false
- No .go changes, so go test not re-run.

## Sync

No SPEC artifact changes (defect fix inside SPEC-ANKICARD-003's existing contract:
refused commands leave no trace). No CHANGELOG in repo; README does not describe the
ordering. Nothing to sync beyond this record.

## Gap

None operator-held: the behavior is fully covered by batch ERT (prompt and message stubbed).
