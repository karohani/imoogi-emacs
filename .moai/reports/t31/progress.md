# t31 progress — clicking a preview link opens it in Emacs (lane-2, Class C)

## Base
- Stacked on t30: `git merge --ff-only WT-preview-follow-cursor` -> 0893589
  (includes t27, t28, t29, t30/t32).

## Before (measured by reading the shipped path)
- A file link rendered with `href` to the server's `/preview?...` endpoint, so a click opened a
  linked-file preview inside the browser. An Org ID link (t28) had no `href` and did nothing.
- Existing channels: browser -> server WebSocket (`browserReadLoop`, `NavigationEvent`) ->
  `broadcastEmacs` -> Emacs SSE (`/api/emacs/events`) -> `imoogi-org-preview--handle-event`,
  which dispatches on `type`.

## Design (trust boundary)
- The browser sends `{type:'open', element_id:<link node id>}` only — no path, href, or ID text.
- Server (`handleBrowserEvent` -> `resolveOpen`): looks up the node in the buffer's committed
  document (`FindNodeByID`); the target comes from the server's own parse.
  - `id:` -> `{kind:"id", id}` when the ID matches `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`.
  - file target -> resolved with the same `AssetResolver` and buffer roots as images
    (allowed roots reused), then `os.Stat` must show a regular file -> `{kind:"file", path}`.
  - missing, outside the roots, or a bad ID -> `{kind:"unavailable", target}`.
  - web links, `color:` links, non-link nodes, and unknown node IDs -> nothing is sent.
  - `NavigationEvent.UnmarshalJSON` deliberately never decodes `open`, so a browser cannot
    supply a target.
- Emacs (`imoogi-org-preview--apply-open`): re-checks the file (absolute, regular file, inside
  `imoogi-org-preview--allowed-roots`) and calls `find-file-other-window`. An ID goes through
  `org-id-find` -> `pop-to-buffer`, widening only if the marker is outside a narrowed region. A
  missing file or ID, or "unavailable", only shows a message; nothing is created. The open runs
  inside `save-current-buffer` so SSE consumption stays in the preview buffer.
- Browser click handler: a click on `a[data-org-kind="link"]` whose `href` is not http(s) calls
  `preventDefault` and `sendOpen`. Web links keep normal browser behavior. Linked-file preview
  mode (no WebSocket) is unchanged.

## Verification
- Go RED: `TestBrowserOpenLinkIsResolvedByServerForEmacs` failed to build (`handleBrowserEvent`
  undefined); `TestBrowserShellSendsLinkClicksToEmacs` failed (no click handler).
- Emacs RED: 3 new `imoogi-org-preview-open-*` tests failed before `apply-open` existed.
- GREEN: `go test ./internal/orgpreview/ ./tests/...` -> ok, ok; `go vet` clean; `gofmt -l` empty;
  the extracted inline browser script passes `node --check`.
- `IMOOGI_ERT_SELECTOR='\(imoogi-org-preview\|imoogi-org-color\)' ./tests/run.sh` -> exit 0,
  `Ran 28 tests, 28 results as expected` (includes check-parens + offline boot).
- card-review: see card-review.md (3 codex passes; 2 findings fixed with tests; final pass).

## Residual risk / gaps
- Not observed end to end in a real browser and Emacs session (structural JS guard plus server
  and Emacs unit tests only).
- This stack does not include t26 (the clipboard asset root in allowed roots). Until both land,
  the Emacs re-check uses the roots on this branch.
- Files opened from the server's extra configured roots (`ServerConfig.AllowedRoots`) are refused
  by the Emacs re-check if Emacs does not list the same root. This is stricter by design.
- `org-id-find` may scan ID locations on a miss (Org's own behavior).
