# t26 progress — clipboard image preview in project-notes artifacts (Class B, lane-2)

## Cause-establishing evidence (reproduction)
- Real document: `~/project-notes/2609.01-imoogi-emacs/artifacts/20261012-spec-bug-...org` line 21
  links `[[file:../assets/imoogi-clipboard-1785001473.png]]`; the file exists in `<notes>/assets/`.
  Clipboard paste saves project-notes assets to `<notes-root>/assets`
  (`28-clipboard.el` `imoogi-clipboard--owner-asset-root`), so an artifact under `artifacts/` links up a level.
- `~/.git` and `~/project-notes/.git` do not exist, so `project-current` is nil for notes documents.
- Batch probe on a scratch replica (`notes/artifacts/doc.org` + `notes/assets/x.png`), main code:
  `project=nil` / `roots=(".../notes/artifacts/")`. The asset directory is outside every allowed root.
- The Go resolver (`internal/orgpreview/assets.go:79-84`) returns `target is outside allowed roots`
  for such a path. Alternative cause ruled out: the parser strips the `file:` prefix
  (`internal/orgpreview/parser.go:316-318`) before resolution.
- Why it was not caught: `asset_map` covers only staged `imoogi-asset:` ids, not saved relative
  links, and no test exercised a project-notes artifact layout.

## Decision record
- Fix in Emacs only: `imoogi-org-preview--allowed-roots` also allows the clipboard owner's asset
  root (`<notes-root>/assets` for project notes, `<doc>.assets` for standalone files). The whole
  notes root is not allowed — narrowest root that covers pasted assets.
- Guarded with `fboundp` and `condition-case` so a registry error (overlapping registrations) never
  breaks the preview.

## Verification
- RED (before fix): `imoogi-org-preview-allows-project-notes-assets-from-artifacts` failed
  (`seq-some ... :value nil`), `Ran 1 tests, 0 results as expected, 1 unexpected`.
- GREEN: `IMOOGI_ERT_SELECTOR=imoogi-org-preview ./tests/run.sh` -> exit 0,
  `Ran 16 tests, 16 results as expected, 0 unexpected` (includes check-parens + offline boot).
- `IMOOGI_ERT_SELECTOR=clipboard ./tests/run.sh` -> exit 0, `Ran 20 tests, 20 results as expected`.
- card-review: see card-review.md (codex, verdict pass, advisory).

## Residual risk / gaps
- Not checked end-to-end in a browser against the running preview server. The server rejection is
  read from `assets.go`, not observed in a live request.
- Markdown documents under artifacts/ use the same roots path but were not exercised separately.
- Full ERT suite and Go tests not run lane-locally (Go code unchanged).
- `moai factory next` lease not attempted for t26. The leader recorded the assignment and said to proceed.
