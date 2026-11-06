# SPEC Review Report: SPEC-PERSPPIN-001
Iteration: 3/3 (FINAL; plan-phase review stream, delta-scoped over r2 R2-D1..R2-D3 (incl. r1 D4) and O-a..O-h)
Tier: M (spec.md, plan.md, acceptance.md, progress.md read; research.md absent)
Verdict: PASS
Overall Score: 0.90 (harmonic mean of Clarity 0.85 / Completeness 0.95 / Testability 0.90 / Traceability 0.90; Tier M threshold 0.80)

Reasoning context ignored per M1 Context Isolation. The caller's description was used only to scope the audit. Every closure claim below was checked against the files, the code, and batch probes.
Audited at worktree `.moai/worktrees/t35`, branch `WT-persp-full-pin`, HEAD `a1f2273`, SPEC v0.1.3. Delta: `git diff 6db7b3d a1f2273 -- .moai/specs/SPEC-PERSPPIN-001` (+282/-82 over 4 files). Code checked against Emacs 30.2 bundled `project.el` 0.11.1.
Cross-model backends: none (Claude-only, same as r1 and r2).

Score trend: r1 0.792 → r2 0.832 → r3 0.899. No regression, so no STOP signal.

**Probe caveat.** The probes below boot the worktree config the way `tests/run.el` does (`boot.el`, persp-mode on) and then run a **model** of the planned v0.1.3 mechanism: a context-call helper that binds a dynamic variable to `:root`, a depth-95 finder with guards 1-3, the context-call clause, and simplified guards 4-8, plus `f` resolving by route (a). They show that the documented design satisfies the ACs it claims. They do not test real code, because none exists yet. Probe files: scratchpad `probe-r3.el` and `probe-e3.el` (not committed).

## Must-Pass Results

- [PASS] MP-1 REQ number consistency: the headings are still `REQ-PIN-001`..`REQ-PIN-015`, contiguous and three-digit (spec.md:L159, L178, L186, L193, L202, L207, L215, L221, L241, L250, L263, L271, L278, L310, L319). The delta added and removed none. The ACs are `AC-PIN-001`..`AC-PIN-018`, contiguous (acceptance.md:L44-L281). The new AC-PIN-018 is at L281.
- [PASS] MP-2 GEARS. This judgment was made on the requirement layer only; the ACs are Given-When-Then and are graded in Group 4. REQ-PIN-008 (L222-237) is still the Unwanted form ("The Perspective finder shall not claim a lookup … with one exception: …"); an exception clause inside a negated requirement is still a single GEARS clause. REQ-PIN-001 (L160-169) is still compound While+When, and the memo phrase was added to its response. REQ-PIN-013 (L279-302) is still Ubiquitous with case-split sub-clauses.
- [PASS] MP-3 YAML frontmatter: all 12 canonical fields are present at spec.md:L2-13, plus `tier: M` and `related_specs`. `version: "0.1.3"` is quoted (L4), and the dates are ISO. No rejected aliases.
- [N/A] MP-4 language neutrality: this is a single-language (Emacs Lisp) configuration project.
- [PASS] MP-5 D7 cross-SPEC: references = {SPEC-PERSPPIN-001 (self), SPEC-TRANSIENT-001}. `SPEC-TRANSIENT-001/spec.md:5` has `status: draft`. No BLOCKING finding.
- [PASS] MP-6 D8 cross-platform: `grep -rn syscall` over the SPEC directory finds nothing.
- [PASS] MP-7 clarification gate: `grep -rn 'NEEDS CLARIFICATION'` over the SPEC directory finds nothing. research.md is absent. progress.md has `open_clarifications: 0`.

## Closure Table (r2 blocking defects)

| Defect | Status | Evidence |
|--------|--------|----------|
| D4 / R2-D1 PIN-WSB-RESOLVE | **RESOLVED** | Route (a) is now written into plan.md D-4 L195-207: `(or (project-current nil) (and CONTEXT-ROOT (cons 'transient CONTEXT-ROOT)) (project-current t))`, with the fallback gated on a non-nil CONTEXT-ROOT. The permitted-edit list E1-E3 reads the same in plan A.4 L60-93, § E L428-430, spec § 7 L449-457, AC-PIN-015 L229-234, and DoD L320-331. **Probe:** `probe-wsb-141` is a copy of `workspace-bridge-test.el:141` with E2 only (the stub moves to `project-find-file-in`, the assertions are unchanged, and no `project-current` stub is added) under E3 and the modelled route (a). It PASSES. `probe-wsb-160` (E2 only; the existing `project-current` stub is kept) PASSES. I also confirmed that `imoogi-project-context-call` (04-projects.el:111-121) feeds the stubbed `imoogi-project-context-root` through `imoogi-project-context`'s `:root`, so the stub used by test :141 reaches CONTEXT-ROOT. |
| R2-D2 PIN-TRANSIENT-OVERRIDE | **RESOLVED** | E3 is listed at every edit-list site: plan A.4 L79-93, § E, spec § 7 L456-457, and DoD L329-331. Plan § F M1 L436 sequences E3 before AC-PIN-001. **Probe `probe-e3.el`** (E3 alone; the new finder is NOT installed, so this shows "E3 breaks nothing", not "finder + E3 is green" — that remains the run phase's § E check): the real `tests/workspace-bridge-test.el` was loaded and the E3 method applied: **34/34 tests pass, 0 unexpected.** E3 therefore keeps every existing list-form assertion valid. Non-vacuity: `(cadr (cons 'transient "/tmp/"))` still signals, so the E3 shape-tolerance is needed. |
| R2-D3 PIN-NOTES-FSD | **RESOLVED** | The "no longer prompt" claims are now scoped by context-root existence rather than buffer kind. Each site is consistent with the route (a) + clause design: REQ-PIN-013.1 (spec.md:L286-293), the § 4 rows f/s/d (L329-331), § 5 "Reverting" (L377-379), the study-note bullet (L359-370), plan D-6 L254-288 (L276-283 lists still-prompting, non-prompting, and prompt-by-design commands), B.3 L365-370, and acceptance Residual Risk L342-346. The false D-6 claim from r2 is corrected (D-6 now credits `f`/`s`/`d` to D-4 route (a) and the D-2 clause). **Probes**, with R non-VC, mapped and current, run from a notes-context buffer (`default-directory` = N, a notes buffer-local set): `probe-d-from-notes` (`dired` gets R), `probe-s-from-notes` (`xref-show-xrefs` called; the project passed to `project-files` is rooted at R), and `probe-f-from-notes` (R/a.txt ∈ ALL-FILES) all PASS. Non-vacuity: `probe-d-from-notes-without-clause-prompts` shows that with the clause disabled, `d` from the same buffer reaches the prompter. |

### Requested probe items

1. **Route (a) `f` and tests :141/:160 with exactly E1-E3:** PASS (see the table). E1 does not touch these tests. E2 and E3 are both needed, and nothing else is.
2. **Context-call clause for `s`/`d`:** verified against project.el 0.11.1. `project-dired` (L1324-1327) is `(dired (project-root (project-current t)))`. `project-find-regexp` (L992-1017) binds `pr (project-current t)` before `project-files` and `xref-show-xrefs`, and its interactive spec calls `project--read-regexp` (L1058-1061) by symbol, so the AC's stub is effective. Both calls go through `project--find-in-directory` (L248-254) on the let-bound `default-directory`. `project-try-vc` declines for a non-VC R, the depth-95 finder runs, guard 3 passes (DIR = the buffer's let-bound `default-directory`), and the clause returns `(transient . CONTEXT-ROOT)` before guard 8. Confirmed by the s/d probes.
3. **REQ-PIN-008 single exception and its safety argument:** holds for the current code. `imoogi-project-notes--current-source-root` (26-project-notes.el:1392) is called only from `--find-existing-or-create` (:1959), `--restore-workspace` (:1977), `-open` (:2693), `-tasks` (:2703), `-journal` (:2812), and :6431. All of these are project-notes commands, and none is on a hook. 26-project-notes.el has no `add-hook`. The `org-mode-hook` members in modules/ (14-org.el:523, 28-clipboard.el:709) do no project lookup. So nothing reachable from `f`/`s`/`d`, including the `find-file` that `project-find-file-in` performs inside the clause's dynamic extent, calls `--current-source-root`. One scope wording nit is listed as O3-c below.
4. **No-context-root path still prompts (REQ-PIN-003):** `probe-no-context-root-prompts` runs with Perspective "main" unmapped, in a projectless sandbox buffer. Both `imoogi-project-dired` and `imoogi-project-find-file` signal `"prompted"`. PASS. The clause variable is bound to `:root` (nil), not to the fallback `default-directory` (plan.md L154-156), and route (a)'s middle branch is gated on CONTEXT-ROOT.

## Optional-finding closure (O-a..O-h)

| r2 optional | Status | Evidence |
|-------------|--------|----------|
| O-a duplicate `### B.3 Risks` | RESOLVED | Only one heading remains, at plan.md:L360. |
| O-b § 7 names both tests | RESOLVED | spec.md:L452-455 names `:141` and `:160` (E2) and E3. |
| O-c `f` must not bind `project-files-relative-names` | RESOLVED | plan.md:L228-232. AC-PIN-013 acceptance.md:L204-205 now gives the correct reason. |
| O-d unmount row descriptor | RESOLVED | acceptance.md:L156, row 5: `:supported t` **and** an existing sandbox `:mount-point`. |
| O-e memo AC | RESOLVED | New AC-PIN-018 (acceptance.md:L281-296) is traced in § 9 (spec.md:L484) and in plan § F M1 (L435). It is binary: R-count 1 after two lookups, 2 after the clear, and R2 after a remap. The count uses `file-equal-p`, and the Sandbox's "projectless buffer" excludes the mapped root (acceptance.md L36-40, the Sandbox "projectless buffer" paragraph), so the buffer's own-dir lookups do not pollute the count. |
| O-f memo thrash | RESOLVED as disclosure | plan.md:L351-355 and acceptance.md:L351-355 record it as accepted. Introduces O3-a below. |
| O-g AC-PIN-003 setup | RESOLVED | acceptance.md:L59-62: re-bind to `(project-try-vc)`, then F0, then the installer. |
| O-h guard 3 order | RESOLVED | plan.md:L139-143: `file-remote-p` on the raw string first, before any `expand-file-name`. |

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.85 | 0.75-1.0 | D-4's resolution is now explicit (plan.md L195-207), the clause is defined with its gating (L145-156), and the prompt claims are scoped (spec.md L286-293). Residual ambiguity is wording only: REQ-PIN-001's "without repeating the VC search … until the next Perspective switch" against the accepted multi-frame thrash (O3-a); REQ-PIN-003 carries no cross-reference to the clause, which skips guards 4-7 (O3-b); and the route (a) middle branch is called "deliberately redundant" but is not gated by guards 1-3 (O3-d). |
| Completeness | 0.95 | 1.0 band, minus one sparse spot | All sections are present, and the frontmatter is complete (L2-15). The study-note consequence is now exhaustive (spec.md L359-370; plan L276-283). The memo has an AC, and the thrash and clause scope are disclosed (plan B.3 L365-370). The remote-mapping behaviour of `f` under route (a) is not disclosed (O3-d). |
| Testability | 0.90 | 1.0 band, minus minor | r2's 0.75 rested entirely on R2-D1 and R2-D2, and both now pass by probe. The new AC-PIN-015 rows (L237-256) and AC-PIN-018 are binary and executable. One minor point: the `f`-from-notes row says "use R" without naming the check (O3-e). |
| Traceability | 0.90 | 0.75-1.0 | § 9 (spec.md L465-484) is 1:1 and now includes AC-PIN-018 → REQ-PIN-001. The REQ-PIN-008 exception is verified by AC-PIN-015's notes-context rows, but § 9 maps AC-PIN-015 to REQ-PIN-013 only (O3-f). |

Harmonic mean = 4 / (1/0.85 + 1/0.95 + 1/0.90 + 1/0.90) = 4 / 4.4513 = **0.899** (≥ 0.80 Tier M threshold; ≥ r2 0.832, so no STOP).

## Defects Found (structured defect-list)

No blocking defect. Each finding below was checked against the strict test: would a correct implementation fail an AC, or is a requirement unimplementable or contradictory in scope? None meets it. All findings are optional and can be taken in the run phase or ignored.

O3-a. PIN-MEMO-REQ-WORDING — spec.md:L163-165 (REQ-PIN-001) vs plan.md:L351-355 and acceptance.md:L351-355 — REQ-PIN-001 says the resolved project is reused "without repeating the VC search, for repeated lookups until the next Perspective switch or a change of the mapping". The accepted one-slot memo repeats the search on every alternation between two frames that show different Perspectives, with no `persp-switch-hook` run. So the plan's design does not literally meet the REQ in the two-frame case. No AC fails (AC-PIN-018 is single-frame), and a small alist memo would meet the REQ. This is wording, and the requirement remains implementable. — Severity: minor — Class: optional — Suggested fix: change the phrase to "for repeated lookups under the same (Perspective name, mapped root) memo key, until the memo is cleared on a Perspective switch (plan.md D-8)". Alternatively, take the alist memo in the run phase.

O3-b. PIN-CLAUSE-REQ003-XREF — spec.md:L186-191 (REQ-PIN-003) vs plan.md:L145-156 — the clause skips guards 4-8, so inside `imoogi-project-context-call` the finder claims even when `persp-mode` is off, the Perspective is unmapped, or the root does not exist. The clause also fires in test :141 (alist nil, nonexistent root), although route (a)'s middle branch would give :141 its project without it. In real code, CONTEXT-ROOT is either an existing mapped root or an own-project root that an earlier member already answers. REQ-PIN-003's "shall decline … every command behaves exactly as before" is therefore observably violated only when a member above depth 95 owns the context root, which REQ-PIN-002 already disclaims. — Severity: minor — Class: optional — Suggested fix: add "except the context-call clause (REQ-PIN-008 exception, REQ-PIN-013.1)" to REQ-PIN-003, or keep guard 5 inside the clause.

O3-c. PIN-CLAUSE-EXTENT — spec.md:L223-226 (REQ-PIN-008) — the exception is described as "scoped to those commands". Its real scope is the dynamic extent of `imoogi-project-context-call`. That extent includes minibuffer reads and the `find-file` (with its hooks) that `project-find-file-in` performs. The safety argument holds today (see probe item 3), but it rests on "nothing in that extent calls `--current-source-root`", not on what the commands themselves call. B.3 (plan.md L367-370) mentions the minibuffer reads but not the `find-file` hooks. — Severity: minor — Class: optional — Suggested fix: reword to "scoped to the dynamic extent of `imoogi-project-context-call`, during which nothing calls `--current-source-root`", and add the find-file hooks to B.3.

O3-d. PIN-ROUTE-A-REMOTE — plan.md:L195-207 and L222-226 ("deliberately redundant"); spec.md:L286-293, L394-395 — route (a)'s middle branch is not gated by guards 1-3, but the clause is. With a remote, non-VC mapped root, `imoogi-project--perspective-root` (04-projects.el:81-91) returns the TRAMP root after `file-directory-p` succeeds. Guard 3 then declines the clause, so `s`/`d` still prompt, while `f` builds `(transient . REMOTE)` and runs `project-find-file-in` (a `find` over TRAMP) with no prompt. So the "redundant" claim is inexact, f diverges from s/d, and the change for `f` is undisclosed. Remote pinning is out of scope (§ 5 L394-395), and no AC exercises it, so this is not blocking. — Severity: minor — Class: optional — Suggested fix: gate the middle branch on `(not (file-remote-p CONTEXT-ROOT))`, or scope REQ-PIN-013.1's "whenever a context root exists" to a local context root and disclose the remote `f` behaviour.

O3-e. PIN-AC015-F-CHECK — acceptance.md:L240-250 — the notes-context `imoogi-project-find-file` row says the command "uses R" but names no observable. Suggested fix: "the recorded ALL-FILES contains `R/a.txt`" (as the probe asserted), or "the PROJECT reaching `project-read-file-name-function` is rooted at R". — Severity: minor — Class: optional.

O3-f. PIN-TRACE-008-EXC — spec.md:L476, L481 — the REQ-PIN-008 exception is verified by AC-PIN-015's notes-context rows, but § 9 maps AC-PIN-015 only to REQ-PIN-013. — Severity: minor — Class: optional — Suggested fix: add REQ-PIN-008 to the AC-PIN-015 row in § 9, or note the exception in the AC-PIN-010 title.

O3-g. PIN-REQ001-HOW — spec.md:L163-165 — the memo phrase puts a caching mechanism in a requirement (RQ-3/RQ-4: HOW in a REQ). It is acceptable because it states an observable constraint, the VC-search count that AC-PIN-018 measures. — Severity: minor — Class: optional — No fix required.

## Regression Check (Iteration 3)

Defects from previous iterations:
- D1-D3, D5-D8 (r1): RESOLVED in r2. The delta does not reopen them: AC-PIN-003's ordering text was only re-sequenced (O-g), D-6 was only re-scoped, and the D-8 memo gained an AC.
- D4 (r1) / R2-D1: RESOLVED. Route (a) is specified, and test :141/:160 pass with E2+E3 only (probe).
- R2-D2: RESOLVED. E3 is everywhere, and 34/34 existing workspace-bridge tests pass with it (probe).
- R2-D3: RESOLVED. The claims are scoped by context-root existence, the study-note consequence is complete, and D-6 is corrected. Notes-context s/d/f probes pass.

New contradictions introduced by the delta: none blocking. O3-a, O3-b, and O3-d are wording-level tensions in edge cases (two frames, members above depth 95, remote roots), each outside every AC and each with a one-line fix.
Stagnation: none. No defect has appeared unchanged across all three iterations.

## Recommendation

PASS (iteration 3/3, final). Must-pass rationale:
- MP-1: REQ-PIN-001..015 are contiguous (spec.md L159-L319).
- MP-2: every REQ is a GEARS clause. REQ-PIN-008's exception sits inside the Unwanted form (L222).
- MP-3: there are 12 canonical fields, with a quoted version (L2-13).
- MP-4: N/A (single language).
- MP-5: SPEC-TRANSIENT-001 is `draft`.
- MP-6: `syscall` does not appear.
- MP-7: no clarification markers.

All r2 blocking defects (R2-D1/D4, R2-D2, R2-D3) are closed, with probe evidence. The harmonic mean is 0.899, against the Tier M threshold of 0.80.

Suggested (non-gating) run-phase actions, cheapest first:
1. O3-d: gate route (a)'s middle branch on a local CONTEXT-ROOT. This is a one-line change, and it makes the "redundant" claim exact.
2. O3-e: name the observable in the `f`-from-notes row of AC-PIN-015.
3. O3-a, O3-b, O3-c: one-phrase REQ wording fixes (memo key; REQ-PIN-003 cross-reference; dynamic-extent scope).
4. O3-f: add REQ-PIN-008 to AC-PIN-015's § 9 row.
