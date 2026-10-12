# t23 — project-notes standing documents

- card: t23 (Class C, leader dispatch; lane-4)
- worktree: .moai/worktrees/t23, branch WT-project-notes-docs (from origin/main 7d98ec0)
- scope stop: worktree commit only — no push, PR, or main merge (leader instruction)
- factory record: `moai factory next --card t23` refused `serial-slot` (t15 holds it);
  leader then recorded `moai factory assign t23 --to lane-4` and told the lane to proceed.
  Observed: `t23 run=tmsgdu state=assigned owner=lane-4 lease=none mode=serial`.

## Decision record (lane judgment, 2026-10-12)

Context: `imoogi-project-notes--documents` (26-project-notes.el) is a fixed constant of
three documents (domain / architecture / decisions); per-project artifact kinds already
live in `.imoogi-project.json` as `artifact_preset` (precedent to mirror).

1. **Standing documents in metadata.** New metadata key `documents` — JSON array of
   `{kind, file, label}` — mirroring `artifact_preset`: to-json / from-json / entry
   accessor with fallback to the built-in constant for legacy metadata, written by
   `--metadata-data`, read by `--metadata-entry`, persisted by a save helper cloned from
   `--save-artifact-preset-metadata`. `file` must be one safe component ending `.org`.
   The Go `imoogi-notes` CLI only `os.Stat`s `.imoogi-project.json`
   (internal/notecache/service.go:355), so a new key cannot break it.
2. **add-document accepts new names.** `completing-read` without require-match; an
   unknown name becomes kind `<slug>`, file `development/<slug>.org`, created from a new
   generic `templates/project-notes/document.org`, and is recorded in `documents`.
   Built-ins keep their own templates.
3. **Promote an artifact into development/.** New command
   `imoogi-project-notes-promote-artifact`, run from an artifact buffer under
   `artifacts/`: refuses unsaved buffers and name collisions, moves the file with its
   `:ID:` intact, updates `org-id-locations` (`org-id-add-location`), records the doc in
   `documents`. Existing `[[id:]]` links keep resolving because they key on the ID.
4. **Optional item (project.org shortcut):** implemented minimally — when a standing
   document is created or promoted, a `- [[file:development/<file>][<label>]]` line is
   appended under `* 바로가기` in project.org if absent; skipped (with a message) when
   project.org has unsaved changes. Nothing more (no regeneration of the list).

## Run-phase evidence

Implementation: manager-develop subagent (TDD). The runtime isolated it in a separate
worktree on the same base 7d98ec0; its five changed files were copied verbatim into t23
(`diff -rq` showed only those files differ), then verified here.

Deviations from the decision record (accepted by the lane):
- `documents` is read from the metadata file only (`--entry-documents`), not stored on
  registry entries: the runtime form has dotted pairs `json-encode` rejects, and
  `--save-entry` encodes whole entries into the registry.
- Promote resets the buffer's modified flag after `rename-file` (Emacs 30 marks it
  modified); safe because unsaved buffers are refused first and contents are unchanged.
- Shortcut line goes after the last list item under `* 바로가기`, keeping the template's
  trailing guide sentence below the links. Added for every development document the
  command newly creates (built-in or custom), never on reopen.
- Transient key `p` "산출물→개발 문서" → `imoogi-project-notes-promote-artifact`.

Card-review follow-ups (lane-authored): every parsed `:ID:` node property is relocated;
promotion is refused when a relative file link other than `../` would change destination.
See `.moai/reports/t23/card-review.md`.

RED (subagent, before implementation): `Ran 40 tests, 35 results as expected, 5 unexpected`.

GREEN (lane, in t23, after the review fixes):

    EMACS=/Applications/Emacs.app/Contents/MacOS/Emacs IMOOGI_ERT_SELECTOR='\(project-notes\|transient\)' ./tests/run.sh
    exit=0
    Ran 243 tests, 236 results as expected, 0 unexpected, 7 skipped (2026-10-12 22:03:46+0900, 34.871586 sec)

The 7 skips are the `cli-cache-*` tests, also skipped on the unchanged baseline
(subagent baseline: `Ran 239 tests, 232 results as expected, 0 unexpected, 7 skipped`).

## Gaps / residual risk

- Full suite (`make test`, Go tests) not run lane-locally; CI runs it after push (no push
  in this card's scope).
- Interactive paths (completing-read without require-match, transient key) are covered
  only through the non-interactive calls and the transient spec-table test.
- The Go `imoogi-notes` cache indexes by path on its own rescan; promotion does not ask
  it to refresh (the in-Emacs snapshot is per-operation only).
