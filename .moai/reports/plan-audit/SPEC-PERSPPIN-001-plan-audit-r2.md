# SPEC Review Report: SPEC-PERSPPIN-001
Iteration: 2/3 (plan-phase review stream, round 2, delta-scoped over r1 D1-D8 plus the optional findings taken)
Tier: M (spec.md, plan.md, acceptance.md, progress.md read; research.md absent)
Verdict: FAIL
Overall Score: 0.83 (harmonic mean of Clarity 0.80 / Completeness 0.90 / Testability 0.75 / Traceability 0.90; Tier M threshold 0.80)

Reasoning context ignored per M1 Context Isolation. The caller's description of the delta was used only to scope the audit. Every closure claim below was checked against the files and the code.
Audited at worktree `.moai/worktrees/t35`, branch `WT-persp-full-pin`, HEAD `6db7b3d`, SPEC v0.1.2. Delta: `git diff 2026135 6db7b3d -- .moai/specs/SPEC-PERSPPIN-001` (+331/-129 over 4 files). Code checked against Emacs 30.2 bundled `project.el` 0.11.1.
Cross-model backends: none (Claude-only, same as r1).

Score trend: r1 0.792 → r2 0.832. No regression, so no STOP signal.

## Must-Pass Results

- [PASS] MP-1 REQ number consistency: the headings are still `REQ-PIN-001`..`REQ-PIN-015`, contiguous and three-digit (spec.md:L135, L152, L160, L167, L176, L181, L189, L195, L211, L220, L233, L241, L248, L275, L284). The delta added and removed none.
- [PASS] MP-2 GEARS (requirement layer only; ACs are Given-When-Then in acceptance.md and are graded in Group 4): the O8 labels are fixed. L189 `REQ-PIN-007 [Event-driven]`, L284 `REQ-PIN-015 [Event-driven]`. REQ-PIN-002 (L153-158) is still Ubiquitous ("The Perspective finder shall be ordered after …"). REQ-PIN-006 (L182-187) is still the canonical Unwanted form ("shall not perform file-system operations on a remote name"). The lane-decision prose was trimmed from REQ-PIN-008 (L206-207) and REQ-PIN-013 (L269-271).
- [PASS] MP-3 YAML frontmatter: all 12 canonical fields are present at spec.md:L2-13. `version: "0.1.2"` is quoted (L4). No rejected aliases.
- [N/A] MP-4 language neutrality: this is a single-language (Emacs Lisp) configuration project.
- [PASS] MP-5 D7 cross-SPEC: references = {SPEC-PERSPPIN-001 (self), SPEC-TRANSIENT-001}. `SPEC-TRANSIENT-001/spec.md` has `status: draft`. No BLOCKING finding.
- [PASS] MP-6 D8 cross-platform: `syscall` count is 0 in spec.md, plan.md, and acceptance.md.
- [PASS] MP-7 clarification gate: `grep -rn 'NEEDS CLARIFICATION'` over the SPEC directory finds nothing. research.md is absent. progress.md has `open_clarifications: 0`.

## D1-D8 Closure Table

| r1 defect | Status | Evidence |
|-----------|--------|----------|
| D1 PIN-DEPTH | RESOLVED | plan.md:L105-112 installs the finder at depth 95 and says why. spec.md REQ-PIN-002 L153-158 scopes the guarantee to members at depth ≤ 90 and states the caveat for higher depths. AC-PIN-003 L58-68 now adds F0 (APPEND) before the installer and F1 (depth 0) and F2 (APPEND) after it. Batch probe with `project` loaded and `project-find-functions` let-bound: F0@t, finder@95, F1@0, F2@t, G3@96 → `(G1 project-try-vc G0 G2 finder G3)`. The finder sorts after every member at depth ≤ 90 and before the 96 member, as REQ-PIN-002's caveat says. The global value is untouched after the `let`. |
| D2 PIN-AC013-CAPTURE | RESOLVED | AC-PIN-013 L200-209 records ALL-FILES at `project-read-file-name-function`. project.el L1255-1282: `project-find-file-in` → `(project-files project dirs)`, and `project--read-file-name` (L1232-1253) passes ALL-FILES through unchanged. `project-files-relative-names` is nil unless the caller binds it (only `project-find-file` L1095 and `project-find-regexp` L1006 do), so the names are absolute (L395-399, VC path L639-658). The stated reason "absolute when several dirs" is imprecise but harmless (see O-c). |
| D3 PIN-AC011-K | RESOLVED | AC-PIN-011 L171-183 adds live buffer P1 visiting `R/a.txt`, a `yes-or-no-p` stub that records its prompt, and an interactive `compile` stub. project.el L1726-1745: `project-kill-buffers` prompts `"Kill %d buffers in %s? "` with `(project-name pr)` only when the buffer list is non-empty. P1 makes it non-empty. |
| D4 PIN-WSB-PRESERVE | **UNRESOLVED** (the documents now agree with each other, but the claim they agree on is false; see R2-D1) | plan.md A.4 L62-71, § E L330-332, AC-PIN-015 L225-230, and the DoD L288-291 now all say "only the stub move, assertions unchanged". The workspace-bridge test `imoogi-project-context-root-follows-perspective-mapping` (tests/workspace-bridge-test.el:141-157) does not stub `project-current`. Under D-4, `f` must get a PROJECT before it calls the stubbed `project-find-file-in`. Probe (copy of the test with only the A.4 stub move, `f` resolving through `(project-current t)`): it reaches `project-prompter` and fails with `(error "prompted")`. Environment: `persp-mode=t`, `alist=nil`, and the context root does not exist (`imoogi-test--root` never creates the subdirectory). This is partly an r1 audit gap: r1 fix option (a) did not anticipate the project-resolution step. |
| D5 PIN-NOTES-SCOPE | RESOLVED as asked (with a new consequence gap, R2-D3) | The glossary at spec.md:L104-108 excludes notes-context buffers from "Projectless buffer". REQ-PIN-001 L145-146 restates the exclusion. The study-note consequence is recorded at spec.md § 5 L324-330, plan D-6 L194-199, B.3 L271-272, and acceptance Residual Risk L303-306. D-6 is not reopened. |
| D6 PIN-VC-CACHE | RESOLVED | spec.md § 6 L362-369 and plan A.1 L27-28 / B.3 L274-278 now say that `project-try-vc` caches only positive results. Plan D-8 L244-262 defines a one-slot memo keyed by (Perspective name, raw root). Guards 1-8, including guard 7's `file-directory-p`, run before a memo hit. Invalidation is implicit when the key changes and explicit on `persp-switch-hook`. The accepted staleness (`git init`) is recorded in acceptance Residual Risk L307-308. |
| D7 PIN-GLOSS-METADATA | RESOLVED | spec.md:L118-123 names `imoogi-project-notes-source-root`, `-directory-local`, and `-entry-instance-id`, and says "Folder metadata files are not consulted". Verified as `defvar-local` at 26-project-notes.el:48, :51, :54. Plan guard 8 L128-135 matches. |
| D8 PIN-TRIGGER-COVERAGE | RESOLVED | AC-PIN-010 L143-161 is a table-driven spy test with rows 1-6. Trigger 7 is verified by registration only (acceptance L162-165), and plan D-7 L238-242 agrees. The trigger functions exist: `--write-registry` :792, `--write-mounted-state` :863, `-mounted-root-refresh` :3252 (`message` only), `-detach` :3819 (requires `origin 'mounted` plus `instance-id`; the row passes "a mounted entry argument"), `-unmount-device` :3920 (`y-or-n-p`, `-unmount-preflight-function` :69, `-executor-function` :73). One test-writing trap is listed as O-d. |

Optional findings taken in v0.1.2 (regression check):
- O1/O2 are stated in REQ-PIN-013.1/.3 (L256-265), § 4 L294-296, and the new § 5 "Reverting incidental behaviour changes" (L334-344), with AC coverage in AC-PIN-015 "And" (L232-237). The AC-PIN-015 coverage is hit by R2-D2, and the statement is over-broad (R2-D3).
- O3 resolved: AC-PIN-009 L117-127 now counts calls to a `persp-current-name` that signals.
- O4 resolved: Sandbox L33-34 and C.2 L310-311 add `(defvar eglot-lsp-context)`; guard 2 uses `bound-and-true-p`; AC-PIN-007 L98-104 adds the nil-binding control.
- O5 resolved: truename comparison in the glossary L118-120 and guard 8 L130-133.
- O6 resolved: REQ-PIN-006 L182-187, guard 3 L118-121, and AC-PIN-008(b) L106-115.
- O7 resolved: the AC-PIN-006 parenthetical was dropped; AC-PIN-010 row 6 calls the trigger directly.
- O8 resolved: see MP-2.

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.80 | 0.75-1.0 | D1/D5/D7 ambiguities are gone (L104-108, L118-123, L153-158). Remaining: D-4 does not say how `f` obtains PROJECT (plan.md L153-157; R2-D1). REQ-PIN-013.1 L256-258, § 4 L294-296, and § 5 L337-338 say "no longer prompt" without the notes-context exception, and plan D-6 L184-185 says `f` works from notes buffers (R2-D3). |
| Completeness | 0.90 | 0.75-1.0 | The study-note consequence and memo staleness are disclosed (spec § 5 L324-330, acceptance L303-308), and a new § 5 H3 L334-344 was added. Gap: the study-note disclosure lists `c`/`k` only and leaves out `s`/`d` (and `f` under the obvious resolution) (R2-D3). |
| Testability | 0.75 | 0.75 | D2, D3, D8, O3, and O4 are fixed, so those ACs are now executable and non-vacuous (L117-127, L140-165, L171-183, L200-209). Two ACs cannot pass for a correct implementation in the runner as written: AC-PIN-015's "only the stub move" (R2-D1, shown by probe), and every AC whose R is not under VC, because a test file globally overrides transient `project-root` (R2-D2, shown by probe). |
| Traceability | 0.90 | 0.75-1.0 | § 9 L414-430 is unchanged and still 1:1. REQ-PIN-008's seven triggers are now each covered (AC-PIN-010 rows 1-6 plus registration for 7). REQ-PIN-013.1/.3's new consequences are traced to AC-PIN-015 "And". The D-8 memo (a § 6 constraint) has no direct AC (O-e). |

Harmonic mean = 4 / (1/0.80 + 1/0.90 + 1/0.75 + 1/0.90) = 4 / 4.806 = **0.832** (≥ 0.80).
The verdict is still FAIL. Per the Retry Loop Contract, a prior-iteration defect that remains unresolved (D4) is an automatic FAIL. Per M6, the two new blocking defects (R2-D2, R2-D3) must also be fixed first. The aggregate score alone would clear the Tier M threshold.

## Defects Found (structured defect-list)

R2-D1. PIN-WSB-RESOLVE (D4 unresolved) — plan.md:L153-164 (D-4), L62-71 (A.4), L330-332 (§ E); acceptance.md:L225-230 (AC-PIN-015), L288-291 (DoD); spec.md:L403-406 (§ 7) — D-4 makes `f` call `project-find-file-in` directly. That function takes PROJECT as an argument, so `f` must now resolve the project itself, and D-4 does not say how ("`f` resolves the project"). Test `imoogi-project-context-root-follows-perspective-mapping` (workspace-bridge-test.el:141-157) stubs `imoogi-project-context-root` but not `project-current`. Its context root is a nonexistent temp subdirectory, and the global alist is nil. With only the A.4 stub move, a `(project-current t)` resolution reaches the prompter. Probe: `(error "prompted")`, test FAILED. So "pass with only the A.4 stub move and unchanged assertions" (AC-PIN-015, § E, DoD) is false under the plan's own mechanism. The sibling test at :160 passes only because it already stubs `project-current`. — Severity: major — Class: blocking — Required fix: choose one and state it in D-4, A.4, § E, AC-PIN-015, the DoD, and spec § 7 alike. (a) D-4 specifies the resolution: `(or (project-current nil) (and CONTEXT-ROOT (cons 'transient CONTEXT-ROOT)) (project-current t))`, where CONTEXT-ROOT is the `:root` already computed by `imoogi-project-context-call`. The transient fallback must be gated on a non-nil context root. Ungated, `f` in an unmapped projectless buffer would stop prompting, which REQ-PIN-003 forbids and nothing discloses. Probe: this passes test 1 with only the stub move, but **only together with the R2-D2 fix**. Or (b) keep `(project-current t)` and list a `project-current` stub in test 1 (returning `(imoogi-test--project default-directory)`, as test 2 already does) as an additional permitted edit. Then drop "only the stub move" from AC-PIN-015, § E, and the DoD.

R2-D2. PIN-TRANSIENT-OVERRIDE (new; missed in r1) — tests/workspace-bridge-test.el:16-20; plan.md:L136-138 (D-2 returns `(transient . ROOT)`), A.4 L58-71; acceptance.md:L44-49 (AC-PIN-001, R not under VC), L232-237 (AC-PIN-015 "And", R not under VC), L288-291 (DoD) — workspace-bridge-test.el defines `(cl-defmethod project-root ((project (head transient))) (cadr project))` at top level. That replaces project.el's `(cdr project)` method for the whole ERT session. `tests/run.el` L42-44 loads every `*-test.el` before running any selector, so this applies to the new test file too. Every project of shape `(transient . ROOT)`, which is both what the finder returns (D-2) and what `project-current t` builds, then signals `wrong-type-argument listp` on `project-root`. Probe in the loaded test environment: `(wrong-type-argument listp ".../first-context/")`. Reproduced in `emacs -Q`: before override `"/tmp/"`, after override `(wrong-type-argument listp "/tmp/")`. AC-PIN-001 ("`project-root` is R canonicalised"), AC-PIN-015's `imoogi-project-dired` / `--project-root` checks, and any `c`/`k`/`f` AC run on a non-VC R therefore fail for a correct implementation. Meanwhile the DoD allows only two edits to existing tests. — Severity: major — Class: blocking — Required fix: add a third permitted edit to A.4, the DoD, and § E: make the override shape-tolerant (`(if (consp (cdr project)) (cadr project) (cdr project))`), or delete it and have `imoogi-test--project` return `(cons 'transient (file-truename root))`. Verified: the shape-tolerant override plus R2-D1 route (a) makes the test-1 probe pass. **Recommended: the shape-tolerant override.** It is a one-line change that keeps every existing workspace-bridge assertion valid, and it is the variant the probe verified. The delete-and-cons route would require re-checking every other test in that file that builds list-form transient projects. Alternatively, state that the new test file must not depend on `project-root` of transient projects. That is not recommended, because REQ-PIN-001.2 is exactly the transient case.

R2-D3. PIN-NOTES-FSD (new contradiction from the O1 and D5 edits) — spec.md:L256-258 (REQ-PIN-013.1), L294-296 (§ 4), L337-338 (§ 5 "Reverting"), L324-330 (§ 5 study-note bullet); plan.md:L184-185 (D-6), L197-198, L271-272 (B.3); acceptance.md:L303-306 — REQ-PIN-013.1 says that for a non-VC mapped root, `f`/`s`/`d` "no longer prompt, because the finder answers the `default-directory` that `imoogi-project-context-call` let-binds". That let-binding happens in the current buffer. From a notes-context buffer, guard 8 (REQ-PIN-008) therefore declines, `project-try-vc` finds nothing for a non-VC root, and `project-find-regexp` / `project-dired` / a `(project-current t)`-resolving `f` prompt as before. No code in modules/ runs `git init` on study notes directories (grep for `"init"`, `git init`, and `vc-create-repo` finds nothing). So in the study-note Perspective, the case REQ-PIN-001.2 names, `s` and `d` still prompt from **every** note file. The § 5 study-note bullet, D-6 L197-198, B.3 L271-272, and acceptance L303-306 list only `c`/`k`. D-6 L184-185 also claims "`b`/`f` work there through D-3 and the Perspective-first context", which is false for `f` unless R2-D1 route (a) is taken. This is the same class of defect as r1 D5 (CN-1). — Severity: minor — Class: blocking — Required fix: qualify REQ-PIN-013.1, § 4 (f/s/d rows), and the § 5 "Reverting" bullet with "from a projectless buffer". Extend the study-note consequence in § 5 L324-330, D-6 L194-199, B.3, and Residual Risk to `s`/`d` (and to `f` unless route (a) is taken), and correct D-6 L184-185. If `s`/`d` prompting from every study note is not acceptable, raise it as a new lane decision rather than editing REQs silently. Possible remedy: give `s`/`d` the same context-root transient fallback as route (a).

## Optional findings (not verdict-bearing)

O-a. plan.md:L264 and L266 — duplicated `### B.3 Risks` heading, an artifact of inserting D-8. Delete one. — Class: optional.
O-b. spec.md:L403-406 (§ 7 Tests) names only `tests/workspace-bridge-test.el:141`, while A.4 L69-71 moves the stub in both :141 and :160. Name both (and the R2-D2 edit). — Class: optional.
O-c. acceptance.md:L202-203 says "ALL-FILES is absolute when `project-find-file-in` receives several dirs". In fact it is absolute because `f` no longer binds `project-files-relative-names` (project.el L395-399), whatever the dir count. State in D-4 that `f` must not bind it. If `f` did bind it, N's files would come back relative to N (`project--files-in-directory` L395-396) and would expand against R to wrong names. — Class: optional.
O-d. AC-PIN-010 row 5 (acceptance.md L154): `imoogi-project-notes-unmount-device` calls `--buffers-under-directory` → `(file-truename mount-point)` (26-project-notes.el:3929, :3890-3898). The preflight stub's descriptor must therefore carry `:supported t` **and** a sandbox `:mount-point`, or the row errors with `wrong-type-argument`. Say so in the row. — Class: optional.
O-e. The D-8 memo (spec § 6 L366-369) has no direct AC. A small test would pin it: count `project-try-vc` (or `locate-dominating-file`) calls across two lookups for a non-VC root, assert one, and assert a miss after the trigger-6/memo-clear function runs. — Class: optional.
O-f. The one-slot memo thrashes when two frames show different Perspectives, because each tab-bar redisplay alternates keys. This costs performance only; a small alist keyed the same way would avoid it. — Class: optional.
O-g. AC-PIN-003 L59-62 adds F0 "before the installer runs", but the Sandbox (L21-25) has the macro build `project-find-functions` through the installer. Say that this test re-binds the list to `(project-try-vc)`, adds F0, and then calls the installer. — Class: optional.
O-h. Guard 3 (plan.md L118-121) lists the `expand-file-name` comparison before the `file-remote-p` check. `expand-file-name` on a TRAMP name enters the TRAMP handler. That opens no connection for an absolute name, but checking `file-remote-p` first is cheaper and keeps REQ-PIN-006's "before any … check" obvious. — Class: optional.

## Regression Check (Iteration 2)

Defects from iteration 1:
- D1: RESOLVED — depth 95 plus scoped REQ-PIN-002; batch probe confirms the ordering.
- D2: RESOLVED — capture at `project-read-file-name-function`; absolute names confirmed in project.el 0.11.1.
- D3: RESOLVED — live P1 plus prompt assertion; `project-kill-buffers` code confirms it is observable.
- D4: UNRESOLVED — the documents agree, but the agreed claim fails by probe (R2-D1). Part of the cause is r1's own incomplete fix option (a).
- D5: RESOLVED as specified — the consequence statement is incomplete for `s`/`d`/`f` (R2-D3, new).
- D6: RESOLVED — caching claim corrected and D-8 memo designed.
- D7: RESOLVED — real `defvar-local` variables named.
- D8: RESOLVED — table-driven rows 1-6, trigger 7 by registration.

No defect has appeared unchanged in every iteration (no stagnation flag).

## Recommendation

FAIL (iteration 2/3). All must-pass criteria pass, and the aggregate score is 0.832 ≥ 0.80. The verdict fails on one unresolved prior defect (D4) and two new blocking defects. All three are text edits to the plan and the ACs (plus the decision of which D-4 resolution route to take), confined to D-4, A.4, § E, AC-PIN-015, the DoD, spec § 7, REQ-PIN-013.1, § 4, § 5, D-6, B.3, and Residual Risk. Fix route for manager-spec:

1. R2-D1: choose route (a) (context-root-gated transient fallback in `f`) or (b) (an extra `project-current` stub in workspace-bridge test 1). Write it into D-4, then make A.4, § E, AC-PIN-015, the DoD, and spec § 7 say exactly which edits the existing tests get.
2. R2-D2: add the `workspace-bridge-test.el:16-20` `project-root` override fix (shape-tolerant, or removed with a cons-shaped `imoogi-test--project`) as a permitted edit everywhere the edit list appears. Route (a) of R2-D1 depends on it.
3. R2-D3: scope the "no longer prompt" claims to projectless buffers. Extend the study-note consequence to `s`/`d` (and `f` unless route (a)), and correct D-6 L184-185. Escalate to a lane decision if `s`/`d` prompting from study notes is unacceptable.

Iteration 3 should be a delta re-audit over R2-D1..R2-D3. It is the last iteration under the 3-iteration cap.
