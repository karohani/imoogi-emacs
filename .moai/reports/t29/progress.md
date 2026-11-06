# t29 progress — color links in org-preview and org-export (lane-2, Class C)

## History
- Blocked first: the `color:` link type was not on origin/main. It existed only as uncommitted
  operator WIP in the primary checkout (14-org.el +83/-2, untracked tests/org-color-test.el).
- Leader relayed the operator's decision B: the lane carries the WIP as Class C, stacked on t28 + t27.

## Integration (stacked base)
- `git merge WT-preview-id-links` (t28, fast-forward to a26ff62), then `git merge WT-preview-keywords`
  (t27) -> conflict in `internal/orgpreview/orgpreview_test.go` (both cards appended a test at the
  same spot), resolved by keeping both tests -> merge commit adbfd49.
- Overlapping lines: t28 changed `parser.go:34` (`linkRE`, added `id:`); t29 changes the same line
  (adds `color:`). t27 added `keywordRE` in the same `var` block. t28 added an `id:` branch at the
  top of `renderLink`; t29 adds a `color:` branch above it.

## WIP import (commit 965942c)
- Copied only the `imoogi-org-color*` parts of primary `modules/org/14-org.el` (face, link
  registration, link-at-point, apply/remove, color transient, agenda-menu "C" entry) and
  `tests/org-color-test.el`. Primary files were only read.
- `diff` worktree vs primary 14-org.el after import: the only remaining differences are the
  yasnippet hunks (line 8 require list, 458-460, 517-519), which were left out on purpose.
- Imported tests: `IMOOGI_ERT_SELECTOR=imoogi-org-color ./tests/run.sh` -> 5/5 passed.

## Cause (measured)
- Export: the WIP registered `:face` only (`grep -c ':export'` -> 0). Baseline HTML export of
  `[[color:blue1][파란 글]]` was `<a href="color:blue1">파란 글</a>`, a broken link with no color.
- Preview: `linkRE` had no `color:` prefix, so the link rendered as raw text.
- Name resolution: `blue1` is an Emacs/X11 name, not CSS. In batch, `color-name-to-rgb "#3498db"`
  returned `(0.0 1.0 1.0)` (terminal approximation). `tty-color-standard-values` returns true
  values independent of the display, but only for lowercase, spaceless names
  (`LightBlue`/`light blue`/`Red` -> nil, `lightblue` -> values).

## Design (implemented)
- One resolver, `imoogi-org-color-hex` (14-org.el): normalizes the name (lowercase, spaces
  removed), then `tty-color-standard-values` -> `#rrggbb`, or nil.
- Export: `:export #'imoogi-org-color--export`. HTML-derived backends get
  `<span style="color:#rrggbb">desc</span>`; an invalid color gives plain text; other backends get
  the text. An undescribed link's raw color text is HTML-escaped.
- Preview: `imoogi-org-color-preview-map` -> payload `color_map` {name: #rrggbb}, following the
  `asset_map` pattern (23-org-preview.el). Go: `RevisionRequest.ColorMap` -> `Renderer.ColorMap`.
  A color link renders `<span data-org-kind="link" ... style="color:#rrggbb">label</span>`. The
  style is emitted only for values matching `^#(hex3|hex6)$` (no CSS injection). An undescribed
  link shows the bare color name.

## Verification
- RED before the fix: 4 new ERT tests failed (hex resolver, HTML export, invalid export, payload
  map); the Go test failed to build (`unknown field ColorMap`); the escaping test exported a raw
  `<img src=x onerror=alert(1)>`; the case-normalization check and undescribed-label test failed.
- GREEN: `IMOOGI_ERT_SELECTOR='\(imoogi-org-color\|imoogi-org-preview\)' ./tests/run.sh` -> exit 0,
  `Ran 25 tests, 25 results as expected, 0 unexpected` (includes check-parens + offline boot).
- `go test ./internal/orgpreview/ ./tests/...` -> ok, ok. `go vet` clean, `gofmt -l` empty.
- JSON shape: `json-encode` of the payload gives `{"color_map":{"blue1":"#0000ff"}}`, or `null` when
  there are no color links (Go unmarshals that to a nil map).
- card-review: see card-review.md (3 codex passes; 2 rounds of findings fixed; 1 accepted limitation).

## Residual risk / gaps
- Accepted limitation (codex round 3): a linked-file preview (`handleFilePreview`, a `.org` file
  opened from a link without an Emacs buffer) has no `color_map`. Hex colors still render; named
  colors (`blue1`) show as uncolored text. Fixing it needs a Go-side X11 color table. Candidate follow-up card.
- LaTeX and other non-HTML backends export the plain text without color (no xcolor dependency added).
- Not observed in a browser against a rebuilt preview binary.
- Full ERT and Go suites not run lane-locally (card-scoped).
