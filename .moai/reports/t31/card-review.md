# t31 card-review
- tool: codex_review, scope=uncommitted, tree .moai/worktrees/t31 (advisory)
- pass 1: fail — [P2] ID jump into a narrowed buffer landed at the narrowing boundary. Fixed: widen
  only when the marker is outside the restriction; test narrows the target buffer (RED -> GREEN).
- pass 2: fail — [P2] opening switched the current buffer during SSE processing. Fixed:
  save-current-buffer around the open; tests assert the preview buffer stays current (RED -> GREEN).
- pass 3: pass, no findings.
