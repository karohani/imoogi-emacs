# t35 card-review — SPEC-PERSPPIN-001

- Tree: `.moai/worktrees/t35`, branch `WT-persp-full-pin`; reviewed diff `git diff 641cc35 565e15b -- modules tests` (run phase)
- Tool: `codex_review scope=card` is not used, because the worktree has no `.moai/config/sections/git-strategy.yaml` (the same condition as t15). The fallback is a read-only `codex_task`, job `job-20261013T001423Z-6f489bc0`. It is advisory and non-binding.
- Codex ran read-only probes against bundled project.el 0.11.1 (Emacs 30.2) and did not run the test suite. The lane checked each finding against the code before accepting it.

## Findings and lane verification

| # | Sev | Finding | Lane verification | Status |
|---|---|---|---|---|
| 1 | medium | `imoogi-project--perspective-find` (04-projects.el ~:155) claims an explicit `(project-current nil DIR)` when DIR equals the buffer's `default-directory`: the directory-equality guard cannot tell it from an implicit lookup (REQ-PIN-004 "shall not claim an explicit lookup"); AC-PIN-006 only tests a DIR different from `default-directory` | Code read: the only explicit-lookup guards are `project-current-directory-override` and `(not (imoogi-project--same-directory-p dir default-directory))` | CONFIRMED (code read) |
| 2 | medium | `imoogi-project--buffer-project` (04-projects.el ~:300, transient `b`) falls back to `imoogi-project--perspective-root` without the finder's remote guards: a projectless remote buffer gets the local R's buffers instead of the prompt, and a remote mapped root reaches `file-directory-p`/canonicalization (REQ-PIN-006) | Code read: the `when-let*` checks only `persp-mode` and then calls `--perspective-root`, with no `file-remote-p` check | CONFIRMED (code read) |
| 3 | medium | A rename rollback leaves the notes-directory list describing the failed rename. The registry write in the commit callback rebuilds the list (26-project-notes.el:802), but `imoogi-project-notes--rollback-rename` (~:2229) restores the registry and mounted state with `with-temp-file` and never rebuilds it, so a fresh buffer under the restored N is not treated as notes context | Code read: the rollback ends with direct `with-temp-file` writes to `registry-file`/`mounted-state-file`, and it has no `--rebuild-context-directories` call | CONFIRMED (code read) |

Other codex observations, none of them defects:
- On a memo hit the finder still runs `file-directory-p`, which the SPEC allows (deleted-root guard).
- The private helpers match project.el 0.11.1.
- The `read-buffer` predicate and default follow project.el's pattern.
- There is no recursion and no escaping error.
- The remote/notes I/O tests are not vacuous.

## Closure

| # | Fix commit | Test | Sync-audit pass 2 |
|---|---|---|---|
| CR1 | 02c10ad (`:around` advice `imoogi-project--mark-explicit-lookup` + marker `imoogi-project--explicit-lookup`) | `imoogi-persppin-explicit-default-directory-not-claimed` | CLOSED (probe + mutation) |
| CR2 | e4b735b (remote guards in `imoogi-project--buffer-project`) | `-switch-buffer-remote-buffer-prompts`, `-switch-buffer-remote-root-prompts` | CLOSED (probe + mutation, no TRAMP I/O) |
| CR3 | e3c225e (rebuild at the end of `--rollback-rename`) | `-rename-rollback-rebuilds-notes-list` | CLOSED (mutation) |

## Disposition

CR1–CR3 are merged with the sync-audit pass-1 blocking findings F1 and F2 (.moai/reports/t35/sync-audit.md) into one run-phase rework, with reproduction tests first, followed by sync-audit pass 2. The card does not close with them open.

The lane's decisions, recorded in progress.md:
- CR1: fix it with an `:around` advice on `project-current` that marks explicit-DIRECTORY lookups. The only explicit callers in modules/ already inhibit the finder.
- CR2: add the remote guards before the `b` Perspective fallback.
- CR3: rebuild the list at the end of the rollback.
- F1: treat a non-file buffer whose `default-directory` is inside a notes directory as notes context.
