# t29 card-review
- tool: codex_review, scope=uncommitted, tree .moai/worktrees/t29 (advisory)
- pass 1: fail — [P2] undescribed color link exported its raw link text into HTML (script injection).
  Fixed: escaped with org-html-encode-plain-text; regression test added (was RED, now GREEN).
- pass 2: fail — [P2] mixed-case/spaced names (LightBlue, light blue) did not resolve; [P2] preview
  label for an undescribed link showed "color:blue1". Both fixed with tests (RED -> GREEN).
- pass 3: fail — [P2] linked-file preview has no color_map, so named colors there are uncolored.
  Accepted as a documented limitation (needs a Go-side color table; out of card scope).
  Re-review ceiling reached.
