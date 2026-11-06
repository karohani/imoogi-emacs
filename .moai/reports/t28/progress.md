# t28 progress — Org ID links render as raw source in org-preview (Class B, lane-2)

## Cause-establishing evidence (reproduction)
- `linkRE` in `internal/orgpreview/parser.go` matched only `file:` and `http(s)://` bracket links,
  so `[[id:UUID][desc]]` was never parsed as a link and stayed as plain text.
- RED: `go test ./internal/orgpreview/ -run TestOrgIDLink` (before fix) rendered
  `<span class="list-content">[[id:E17712CF-ECC8-436F-9E49-C920A209BA52][bug &lt;preview&gt; note]]</span>`,
  which matches the card's example.

## Decision record
- Parser: `id:` added to the `linkRE` bracket-link prefixes; the node is an ordinary `link` with
  `Attrs.target = "id:<UUID>"`.
- Renderer: an `id:` target renders as an anchor carrying the ID and no `href`. The ID cannot be
  resolved to a file inside the Go server; resolving it belongs to Emacs (t31).
- No Emacs-side change was needed for this card. `23-org-preview.el` does not touch link rendering.

## Rendered link form (handoff to t31)
```html
<a data-org-id="link-…" data-org-preview-id="link-…" data-org-kind="link"
   data-org-range-start="N" data-org-range-end="M"
   data-org-link-id="E17712CF-ECC8-436F-9E49-C920A209BA52">description</a>
```
- `data-org-link-id`: the raw Org ID (text after `id:`), HTML-escaped. There is no `href`.
- The description is HTML-escaped. With no description (`[[id:X]]`), the label is `id:X`.
- `data-org-range-start/end` are the source byte range of the whole `[[...]]`.
- t31 can hook clicks on `a[data-org-link-id]` and send the ID to Emacs (`org-id-goto`/`org-id-find`).

## Verification
- GREEN: `go test ./internal/orgpreview/ ./tests/...` -> both `ok`. `go vet` clean, `gofmt -l` empty.
- card-review: see card-review.md (codex, verdict pass, advisory).

## Residual risk / gaps
- Not observed in a browser against a rebuilt preview binary.
- Markdown parser unchanged (Org ID links are Org-only).
- The link is visible but not yet clickable until t31.
