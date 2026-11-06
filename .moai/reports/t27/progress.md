# t27 progress — Org keyword lines leak into org-preview (Class B, lane-2)

## Cause-establishing evidence (reproduction)
- Production Org parser is `FallbackParser` (`internal/orgpreview/server.go:96`).
- `FallbackParser.Parse` (`internal/orgpreview/parser.go`) has no rule for `#+KEY: value` lines,
  so they fall through to the paragraph branch, and consecutive keyword lines are joined into one `<p>`.
- RED: `go test ./internal/orgpreview/ -run TestOrgKeywordLines` (before fix) rendered
  `<p ...>#+TITLE: Spec\n#+CATEGORY: imoogi-emacs</p>`. This matches the card's screenshot: the
  keyword lines appear joined as one line, and the heading and property table below render normally.
  The same path also leaks a mid-body `#+caption:` line into the surrounding paragraph.

## Decision record
- Org keyword lines (`^[ \t]*#\+[[:alnum:]_-]+:` followed by space or end of line) are document
  metadata and are skipped. They also end a paragraph. This follows Org export, which does not
  print keyword lines as body text.
- `#+begin_src` / `#+begin_example` (no colon) are unaffected. Promoting `#+TITLE` to a rendered
  document title was not done; that is a feature beyond this defect.

## Verification
- GREEN: `go test ./internal/orgpreview/` -> `ok`. `go vet ./internal/orgpreview/` clean,
  `gofmt -l internal/orgpreview/` empty.
- `go test ./tests/...` -> `ok github.com/karohani/imoogi-emacs/tests` (Go preview contract test).
- `go test ./cmd/...` -> imoogi-agent / imoogi-anki / imoogi-notes ok.
- card-review: see card-review.md (codex, verdict pass, advisory).

## Residual risk / gaps
- Not observed in a browser against a rebuilt preview binary.
- Keyword lines vanish from the preview, so a cursor on one has no preview node to sync to
  (same as blank lines).
- Markdown parser unaffected and not touched.
