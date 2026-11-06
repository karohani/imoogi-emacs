# t36 card review

Reviewer: codex (`codex_task`, read-only, work_key t36) over the uncommitted diff.

Questions asked: (1) any remaining write-before-refusal path in set-direction /
toggle-incremental or other helper callers; (2) inherited vs own ANKI_NOTE_TYPE mismatch;
(3) whether the new tests fail on the old order; (4) regressions for cloze/untyped headings.

Verdict: **no findings.**
- refusal checks precede writes in both commands and in `imoogi-anki-toggle-swift`
- ANKI_NOTE_TYPE intentionally does not inherit, matching the scanner
- both new tests fail under the old order (confirmed empirically by the RED run in progress.md)
- cloze and untyped headings keep their prior behavior

Lane self-review notes:
- On a Basic heading the type message now replaces the swift prompt — the user is no longer
  asked a question whose "yes" would be thrown away.
- On cloze/untyped headings `imoogi-anki--cloze-type-acceptable-p` runs twice (directly and
  inside ensure); it is pure and silent on success, so harmless.
