# t33 — org-preview: recent files in the left sidebar

- card: t33 (Class C, leader dispatch; factory `assigned` → lane-4)
- worktree: .moai/worktrees/t33, branch WT-preview-recent-files; base origin/main 7d98ec0
  + `git merge WT-preview-follow-cursor` (t30, fast-forward to 0893589)
- scope stop: worktree commit only — no push, PR, or main merge

## Decision record (lane judgment, 2026-10-12)

1. **Meaning of "최근 연 파일".** Files previewed in this Emacs session: the server's
   `sessionState.buffers` already holds each previewed buffer's path, browser view and
   generation. Buffers without a file path are skipped. This is **not** Emacs `recentf`;
   if the operator meant recentf, that is a different card (Emacs would have to send the
   list, and entries would only be read-only snapshots).
   No Emacs-side change: `modules/org/23-org-preview.el` is untouched.
2. **Clicking an entry opens that file's live preview.** New endpoint
   `GET /api/recent?session&buffer&generation&view`, authorized by `browserAuthorized`
   (the current page's own credentials). The server composes each entry's live
   `/preview?session=…&buffer=…&generation=…&view=…` URL.
   Trade-off, accepted: a page for buffer A now learns the per-buffer `view` token of the
   other buffers in the **same** session (same Emacs, same user). The capability widens
   from buffer to session; it never crosses sessions. The linked-file snapshot route was
   rejected: it is static (no live updates) and `assetAllowed` refuses files outside A's
   roots, which would hide other projects' files.
3. **Placement.** A third sidebar tab `최근` that reuses the existing `.tab`/`.panel`
   machinery (least new CSS). The current file is marked; the most recently updated
   file comes first; at most 20 entries.
4. **Ordering.** `revision` is per-buffer and cannot be compared across buffers. New
   `bufferState.updatedAt` is set in `commitRevision`.
5. **Refresh.** The list is fetched when the socket opens, after each render, and when
   the tab is clicked. There is no push: `broadcastBrowser` is per-buffer.

## Implementation (lane-direct; no subagent — a subagent tree would start from
origin/main without t30 and copying back would revert t30's script changes)

- `internal/orgpreview/recent.go` (new): `GET /api/recent` → `{files:[{title,path,url,current}]}`,
  gated by `browserAuthorized`. Lists only the current session's buffers that have a path,
  newest first, capped at `recentFilesLimit` = 20.
- `internal/orgpreview/server.go`: `Server.touchSeq` and `bufferState.touched`, set in
  `commitRevision`; one `routes()` line; CSS `.recent-*`; a `최근` tab button and a
  `#recent` panel; `refreshRecent()` called from `ws.onopen` and from the end of
  `renderRevision`; new `refreshRecent`/`fillRecent` and the tab click listener, inserted
  just before `bootEmpty()`. Entries are `<a href>` built with `textContent` (no HTML
  injection) and the full path as tooltip; a ticket counter drops out-of-order responses.
  The tab is empty on linked-file snapshot pages (no live buffer credentials).
- `internal/orgpreview/recent_test.go` (new): auth (missing/wrong view, stale generation),
  session isolation, skips pathless buffers, newest-first ordering, re-ordering after an
  update, current flag, entry URL = the other buffer's own working credentials, cap, and
  shell contract (tab, panel, endpoint, `renderRevision` refresh).

RED (before implementation): `go test ./internal/orgpreview/ -run Recent` → exit 1,
`undefined: recentFilesResponse`, `undefined: recentFilesLimit` (build failed).

GREEN:

    go test -race ./internal/orgpreview/ ./tests/
    exit=0
    ok  	github.com/karohani/imoogi-emacs/internal/orgpreview	1.460s
    ok  	github.com/karohani/imoogi-emacs/tests	1.835s

Also: `gofmt -l internal/orgpreview/` → no output (after formatting recent_test.go);
`go vet ./internal/orgpreview/` → clean; browser script extracted from `browserShell`
→ `node --check` → OK.

Card-review: `codex_review` scope=uncommitted, round 1 → `pass`, no findings
(excluded_untracked empty, so recent.go/recent_test.go were reviewed).

## Overlap with t31 (lane-2, WT-preview-open-links) in internal/orgpreview/server.go

t31 is uncommitted in `.moai/worktrees/t31` (branch head still 0893589), so it was
measured with plain `diff` against its working file, base = 0893589.

| Card | Base lines | Function / area |
|---|---|---|
| t33 | +2 after 56 | `type Server` (touchSeq) |
| t33 | +1 after 75 | `type bufferState` (touched) |
| t33 | +1 after 153 | `routes()` |
| t33 | +2 after 800 | `commitRevision` |
| t33 | +1 after 1091, 1099, 1102 | shell CSS, tab button, `#recent` panel |
| t33 | 1139 (changed) | `connect()` → `ws.onopen` |
| t33 | +1 after 1190 | `renderRevision()` tail |
| t33 | +46 after 1315 | new `refreshRecent`/`fillRecent`, before `bootEmpty()` |
| t31 | 904–918 | `browserReadLoop` → new `handleBrowserEvent`, `resolveOpen`, `orgIDRE` |
| t31 | +7 after 1147 | `connect()` → `root.onclick` (open links in Emacs) |
| t31 | +3 after 1178 | new `sendOpen()` after `sendNavigation()` |

Closest pair: t33 `ws.onopen` (1139) and t31 `root.onclick` (after 1147), both inside
`connect()` but 8 unchanged lines apart. Trial three-way merge on scratch copies
(`git merge-file -p ours base theirs`) → `conflicts=0`. This is a snapshot of t31's
uncommitted state at 2026-10-12 ~22:3x; re-check when t31 commits. Test files do not
overlap (t33 adds `recent_test.go`).

## Gaps / residual risk

- Not exercised in a real browser; JS verified by syntax check and the shell-contract
  test only.
- Elisp side unchanged, so `make test-elisp` was not run; the full suite was not run
  lane-locally.
- The list only knows buffers that have sent a revision in this server's lifetime (not
  Emacs `recentf`, not files from before a server restart).
- Clicking an entry navigates the same tab; the target's credentials can go stale if
  that buffer is re-enabled with a new generation before the list refreshes (the target
  page then shows unauthorized until reopened from Emacs).
