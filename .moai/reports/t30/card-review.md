# t30 / t32 card-review
- tool: codex_review, scope=uncommitted, tree .moai/worktrees/t30 (advisory)
- pass 1: fail — [P2] renderRevision's late rAF scrollTo cancelled the reveal. Fixed: restore the
  viewport synchronously, then reveal; ordering guard added (RED -> GREEN).
- pass 2: fail — [P2] restoreActive always scrolled, which also pulled browser-selected targets back.
  Fixed: activeTarget carries `reveal`; restore reuses it; guard updated.
- pass 3: pass, no findings.
