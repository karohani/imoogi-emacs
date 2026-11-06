# t23 card-review (advisory)

Tool: `codex_review` (backend codex), scope `uncommitted`, tree `.moai/worktrees/t23`.
`scope=card` reviews committed diff from the merge base; the review ran before the commit,
so the uncommitted scope covered the full card change.

| Round | Verdict | Findings | Disposition |
|---|---|---|---|
| 1 | fail | P2 promote updated only the first `:ID:`; P2 relative `file:` links change destination after move | fixed: register every ID; refuse promotion when a relative link (not `../`, not absolute) would break |
| 2 | fail | P2 plain `file:` links bypassed the regexp guard; P2 `:ID:` text inside blocks was collected | fixed: both helpers now use `org-element-parse-buffer` (node-property / link elements) |
| 3 | pass | none | — |

Tests added for each finding (tests/project-notes-test.el, promote test): copied-section ID
relocated, example-block ID not registered, three link forms refused.
Re-review ceiling: 3 rounds used.
