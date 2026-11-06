# t30 progress — org-preview follows the Emacs cursor (lane-2)

t32 = t30과 동일 범위 (same symptom: the preview does not follow the editing cursor; one change covers both).

## Base
- Stacked on t29: `git merge --ff-only WT-color-links` -> 92bf381 (includes t27, t28, t29).

## Cause (measured by reading the shipped code path)
- Emacs sends cursor navigation 0.15 s after point moves (`23-org-preview.el` post-command ->
  `/api/emacs/navigation`); the server maps the cursor byte to a node and broadcasts it
  (`server.go` `handleEmacsNavigation` / `mapEmacsNavigation`).
- Browser: `ws.onmessage` -> `markFromMessage` -> `markElement`, which only toggles the
  `active-block` class. Before the fix, `grep -c scrollIntoView internal/orgpreview/server.go` -> 1,
  and that one call is the sidebar `jumpTo`. Nothing scrolled for cursor navigation, and
  `renderRevision` re-pinned the old viewport with `requestAnimationFrame(() => window.scrollTo(x, y))`.

## This was a deliberate earlier decision — now reversed by the cards
- `ae27a58` (2026-09-10, "feat: add colorful Org preview outline and paragraph overview") removed
  auto-scroll on purpose: "Highlight the current editing block without moving browser scroll". It
  added a guard test forbidding `window.onscroll` and `block:'center'`, and the README line
  "브라우저 스크롤은 강제로 이동하지 않으며".
- t30 ("미리보기 화면이 커서를 따라가지 않는다") and t32 ("편집 중인 커서 위치를 따라가야 한다")
  explicitly ask for following, so the operator's cards reverse that decision.
- Decision record (lane judgment): follow while keeping the old decision's concerns. Use
  `block:'nearest'` (a visible block does not move; no centering; the old `block:'center'` guard still
  passes), no `window.onscroll` feedback loop, and scroll only for Emacs-originated navigation
  (`msg.origin === 'emacs'`). Browser clicks and the sidebar keep their own behavior.

## Change
- `server.go` browser script: `markElement(el, reveal)` scrolls with
  `scrollIntoView({block:'nearest',behavior:'smooth'})` when `reveal`. `markFromMessage` passes
  `msg.origin === 'emacs'`. `activeTarget` remembers `reveal`, so `restoreActive` (re-render and
  async Mermaid) keeps only Emacs-followed blocks in view. `renderRevision` restores the old viewport
  synchronously before revealing; the late `requestAnimationFrame` scrollTo, which cancelled the
  reveal, was removed.
- `23-org-preview.el`: `imoogi-org-preview-highlight-current` docstring now says it keeps the
  paragraph in view. `README.md` scroll sentence updated to match.

## Verification
- RED: `go test ./internal/orgpreview/ -run TestBrowserShellKeeps` failed before the fix
  ("markElement must reveal the active block with block:'nearest'"), and again for the ordering
  check before the renderRevision fix.
- GREEN: `go test ./internal/orgpreview/ ./tests/...` -> ok, ok; `go vet` clean; `gofmt -l` empty.
- The extracted inline browser script passes `node --check` (syntax ok).
- `IMOOGI_ERT_SELECTOR=imoogi-org-preview ./tests/run.sh` -> exit 0, `Ran 15 tests, 15 results as expected`.
- card-review: see card-review.md (3 codex passes; 2 rounds of findings fixed; final pass).

## Residual risk / gaps
- The Go test is a structural guard over the embedded script (no JS DOM harness exists for the
  shell). Scrolling behavior was NOT observed in a real browser.
- Smooth scroll on every Emacs cursor message may feel busy on fast cursor movement; the 0.15 s
  Emacs debounce limits the rate.
- A user who scrolls the preview by hand is pulled back when the Emacs cursor next moves, or when
  the next revision re-renders while an Emacs-followed block is off screen. That is what "follow"
  means; there is no opt-out setting yet.
