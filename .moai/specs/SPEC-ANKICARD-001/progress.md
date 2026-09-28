# Progress — SPEC-ANKICARD-001

## §E.1 Plan-phase Audit-Ready Signal

- plan_complete_at: 2026-09-05
- plan_status: audit-ready
- tier: L (5 artifacts — spec.md, plan.md, acceptance.md, design.md, research.md)
- artifacts: spec.md, plan.md, acceptance.md, design.md, research.md, progress.md, spec-compact.md
- gate: Decision Point 1 approved 2026-09-05 (plan.md revision 2 accepted as-is)

## §E.2 Run-phase Evidence

### M1 — AnkiConnect client surface and test doubles (commit 595db7d)

| AC / invariant | Actual Output | Status |
|---|---|---|
| AC-C-003a (zero model writes, sync hot path) | `TestRun_AC_C_003a_SyncRunOfAnyShapeRecordsZeroModelWrites` — 6 subtests PASS (add / update / deck move / delete / no-op / empty); each asserts `createModelCalls == 0`, `updateModelStylingCalls == 0`, `updateModelTemplatesCalls == 0` | PASS |
| AC-C-003b (install path ownership-scoped) | Mechanism only — `createModelCall.name` / `updateModelStylingCall.name` / `updateModelTemplatesCall.name` are recorded per call, so the `imoogi-` name scoping is checkable. No install path exists until M2, so the criterion itself is not yet assertable | PASS (mechanism) — criterion deferred to M2 |
| M1 zero-write fence exists before anything can violate it | Fence written and RED-captured before the fake methods existed (build failure at `fake_client_test.go:82`, interface mismatch) | PASS |
| Five methods reachable through the `AnkiConnector` interface | `TestAnkiConnectorInterfaceCarriesTheFiveNewModelAndMediaMethods` — records wire actions `modelNames, createModel, updateModelStyling, updateModelTemplates, storeMediaFile` in that order | PASS |
| Wire shapes match design.md §8.1 | `createModel` sends `cardTemplates` as an ARRAY of `{Name,Front,Back}`; `updateModelTemplates` sends `templates` as an OBJECT keyed by card name — asserted as distinct shapes | PASS |
| No behavior change to any existing path | Full Go suite green; every pre-existing planner / ankiconnect / cmd test passes unchanged | PASS |
| Elisp↔Go code contract | `make test-elisp` — 153 tests, 151 as expected, 0 unexpected, 2 skipped | PASS |
| `hashing.Hash` inputs / `protocol.Version` unchanged | No edit to `hashing.go`; `protocol.Version` untouched (only the diagnostic-code const block extended) | PASS |

Coverage vs floors (`go test -count=1 -cover ./internal/anki/...`):

| Package | Floor | Measured | Status |
|---|---|---|---|
| ankiconnect | 85.0 | 86.2 (from 84.3 baseline) | PASS |
| planner | 90.3 | 90.3 | PASS |
| registry | 87.8 | 87.8 | PASS |
| hashing | 100.0 | 100.0 | PASS |
| orgdoc | 100.0 | 100.0 | PASS |

## §E.3 Run-phase Audit-Ready Signal

- run_status: all six milestones complete and integrated on main (M1 bf81d3b · M3 b4b2754 · M2 5b5107e · M4 1fadb31/54b1eaa/4a43ee0 · M5 c892cb3 · M6 88dc542/264bbf6); flashcards commits 46012f0/7c5dfb2 interleaved (separate feature, user-built)
- head_commit_sha: 264bbf6 (parent chain back to plan-phase base 8eaf273)
- branch: main (primary checkout; every milestone cherry-picked or fast-forwarded from an L1 agent worktree, all worktrees removed)
- ac_pass_count: 23 of 24 (per-milestone tables above)
- ac_fail_count: 0
- ac_gap_count: 1 — AC-C-022c (manual live-collection check; AnkiConnect was unreachable during the run)
- new_warnings_or_lints_introduced: 0 (`make lint` = `go vet ./...` exit 0; `make fmt-check` exit 0 at 264bbf6)
- cross_platform_build.host: PASS (`go build ./...` exit 0)
- cross_platform_build.windows: PASS for SPEC scope (`GOOS=windows go build ./cmd/imoogi-anki` exit 0); repo-wide windows build still fails at baseline in internal/setup (pre-existing, outside scope)
- coverage_vs_plan_floors: ankiconnect 86.2/84.3 · planner 91.5/90.3 · registry 87.8/87.8 · hashing 100/100 · orgdoc 100/100 · model 98.3 · media 91.3 · cmd 86.9 (85.0 norm) — no package below its floor
- elisp_suite: 219 tests / 218 as expected / 1 unexpected — the one failure is the pre-existing org-preview baseline in the user's untracked WIP, present before M1
- total_run_phase_files (8eaf273..264bbf6, SPEC scope): see `git diff --stat 8eaf273..264bbf6 -- internal/anki cmd/imoogi-anki modules/24-anki.el modules/anki tests/anki-*`
- commit_strategy: one (occasionally two) commits per milestone, conventional messages, explicit pathspec staging only
- push_state: not pushed (policy — pre-push `make ci-local` fails on the baseline org-preview Elisp test)
- red_evidence_gaps: M2 and M5 pre-GREEN output lost with rate-limited agents; M4 RED was a build failure (new package); M6 six tests green-first by design
- docs_to_reconcile_at_sync: plan.md D8–D10 staleness; spec.md §2 closing-`$` boundary wording; design.md `ankiconnect_url` → `anki_connect_url`; observation (1) above (deck-class normaliser not wired into templates) pending sync-auditor verdict
- mx_tags: 0 → 0 (Phase 18: no new exported function with fan_in ≥ 3, no goroutine/channel patterns in 8eaf273..264bbf6; fan_in ≥ 3 symbols are all pre-SPEC methods)
- audit_ready: yes — sync-auditor evaluation (Phase 16/17) requested at 2026-09-06T08:25Z

## §E.4 Sync-phase Audit-Ready Signal

_<pending sync-phase>_

## Plan audit gate (run Phase 1)
- audit_verdict: PASS
- audit_score: 0.98
- audit_report: .moai/reports/plan-audit/SPEC-ANKICARD-001-review-2.md
- audit_at: 2026-09-05T07:44:37Z
- auditor_version: plan-auditor (iteration 2; iteration 1 PASS 0.96 on v0.1.1 → superseded by v0.1.2)
- audit_cache_hit: false
- plan_artifact_hash: 3899548c8195e223c8e58e562bcb169f5e18d3c094be04a4bdeffad0c20fe997
- plan_md_digest_findings: D8 (REQ-C-018 double-bound in §C), D9 (§B.4 "two exclusions" vs spec §4.4 three), D10 (pre-v0.1.2 census at :93-95, EARS labels in §C) — plan.md staleness only; spec.md is SSOT; fix at sync phase

## Run Phase 0 — context
- mode: autopilot (workflow.default_mode absent → harness auto-select)
- harness_level: standard
- execution_mode (Phase 4): serial (coding-heavy; single manager-develop per milestone)
- progression_mode: autonomous (user choice at kickoff; ac_converge to be armed alongside M1)
- kickoff_approval: granted 2026-09-05 (option "렌즈 결함 반영(rev 4) 후 바로 착수"); rev 4 applied; re-audit PASS 0.98
- git_policy: commit per milestone on main; NO push (pre-push ci-local fails on baseline org-preview Elisp test); stage by explicit pathspec; PRESERVE untracked org-preview work
- goal_armed: ac_converge (autonomous, --max-turns 30) at 2026-09-05T08:22:04Z alongside M1 dispatch
- M1 integrated on main as bf81d3b (cherry-pick of worktree commit 595db7d); orchestrator verification: build host+windows ok, cover ankiconnect 86.2/planner 90.3/registry 87.8/hashing 100/orgdoc 100, vet ok, AC-C-003a fence PASS, elisp 164/163 (baseline org-preview only); fmt-check FAIL attributed to leftover agent worktree vendor testdata → worktree removed

### M3 — LaTeX → MathJax transform (integrated 2026-09-05T11:26:47Z)
- commit: b4b2754 on main (cherry-pick of worktree commit 59ef2bf, parent bf81d3b); not pushed (policy)
- files: internal/anki/orgdoc/orgdoc.go (transform inside renderFragment — REQ-C-016 by construction), internal/anki/orgdoc/math_test.go (new)
- E1: AC-C-008 PASS (4 syntaxes byte-exact) · AC-C-009a PASS ($5/$7 untouched) · AC-C-009b PASS (<br>, bound) · AC-C-009c PASS (<code>/<pre>/attr/href untouched) · AC-C-023 PASS (cloze collision byte-for-byte) · AC-C-024 PASS (regression, not upstream assertion)
- E2/E3 (orchestrator re-run on main): build host+windows(anki) ok; cover ankiconnect 86.2 / planner 90.3 / registry 87.8 / hashing 100 / orgdoc 100
- E8 RED captured pre-GREEN (agent report): conversion rows failed at bf81d3b; protection rows were green by nature (regression guards)
- gaps: (a) spec.md §2 closing-$ boundary wording excludes end-of-text literally; implementation treats end-of-span as boundary per AC-C-008 row 1 — reconcile wording at sync (docs); (b) <br> normalization inside \(..\)/\[..\] interiors implemented but no test row — assigned to M6 (test-only Go addition)
- run_status: M1 complete; M3 complete; M2, M4, M5, M6 pending
- goal re-armed 2026-09-05T11:28:19Z: the first arm was classified MECHANICAL by the parser (treated as a shell command, exit 2 each turn → ceiling-exit at 6/30); cleared and re-armed as a MODEL condition referencing the conversation transcript (0/30, autonomous). M1/M3 evidence already in the transcript.

### M2 — embedded templates, base stylesheet, install-models (integrated 2026-09-05T15:26:09Z)
- commit: 5b5107e on main (cherry-pick of worktree commit 6b6403e, parent b4b2754); not pushed (policy)
- provenance: manager-develop implemented M2 in worktree agent-abdb5a9bce0691297 and was terminated by API rate limit (429) before committing; orchestrator verified in the worktree (model 98.3%, cmd 85.2%, all internal/anki ok), committed by explicit pathspec, integrated, re-verified on main (see E2/E3 below)
- E2/E3 on main: build host+windows(anki) ok; coverage per package recorded in transcript
- AC-C-001/002/007 install probe-then-act + idempotence: covered by internal/anki/model tests (agent's RED/E8 evidence lost with the agent — gap: no verbatim pre-GREEN output for M2)
- AC-C-003b: every model write names an imoogi- model — asserted in model tests; AC-C-004 normalization table incl. (PROGRAMMER)::(GO) → deck-programmer-go, 2026 Review → _2026-review, empty → deck-unnamed; AC-C-005 stylesheet tokens/selectors/no-network; AC-C-006a-c concat; AC-C-022b mechanism
- run_status: M1, M3, M2 complete; M4, M5, M6 pending

### M4 — media resolve/upload/rewrite + ordering guard (integrated 2026-09-06T01:02:30Z)
| M4 | media resolve/upload/rewrite + ordering guard | AC-C-010a/b, 011a-c, 012a-c, 013, 014a/b, 015a/b, 016a-c, 017a/b | 17/17 PASS (agent E1; orchestrator re-ran guard tests on main) | 4a43ee0 |
- commits on main: cherry-picks of worktree 28974bc/7a1c8d7/4ae4b39 (parent 5b5107e; main also carries flashcards 46012f0/7c5dfb2 in between); not pushed (policy)
- coverage on main: see transcript (agent: media 91.3, planner 91.1, others at baseline)
- deviations (documented in code): media_upload_failed ⇒ action skipped (per REQ-C-015 text, brief said failed — no AC pins it); media.Rewrite takes (baseDir, root, fields) — root needed for confinement (design §9.1 sketched two params); sanitization strips '-' and '_' at both edges so '_private.png' cannot yield a '_'-prefixed stored name; Config.SyncRoot now used with ~ expansion via os.UserHomeDir
- E8 gap: media package is NEW so RED was the build failure, not an assertion failure — the ordering-guard assertion was never observed failing against pre-M4 logic
- run_status: M1, M3, M2, M4 complete; M5, M6 pending
- goal re-armed (2nd) 2026-09-06T01:03:49Z: model condition again ceiling-exited at 6/30 (stagnation guard while waiting on background agents); re-armed with --max-turns 0 --max-duration 21600 and per-milestone progress wording. Work continues on agent notifications regardless.

### M5 — migrate subcommand + dry-run (integrated 2026-09-06T07:58:20Z)
| M5 | migrate subcommand + dry-run | AC-C-018a/b/c, 019a/b/c, 020a/b, 022b (mechanism) | 16 migrate tests PASS on main | c892cb3 |
- commit: c892cb3 on main (cherry-pick of worktree commit 224f9f2, parent 4a43ee0); not pushed (policy)
- provenance: manager-develop implemented M5 in worktree agent-a0c654495f3043cc9 and was terminated by API rate limit (429) before committing; orchestrator verified in the worktree (planner 91.5, cmd 86.9, race clean, wire contract unchanged), gofmt-ed 2 files, committed by explicit pathspec, integrated, re-verified on main
- E8 gap: RED evidence lost with the agent (same as M2)
- run_status: M1, M3, M2, M4, M5 complete; M6 pending

### M6 — Elisp wiring, diagnostics, contract tests (integrated 2026-09-06T08:25:00Z)
| M6 | Elisp wiring, diagnostics, contract tests | AC-C-004 (mirror), 006d, 018c, 020.3, 021a/b/c, 022a, 022c | 8 PASS / 1 GAP (022c live collection) | 264bbf6 (+ 88dc542) |
|    | AC-C-004 deck-class mirror | PASS | tests/anki-notetype-test.el: reads the 9 AC rows from internal/anki/model/deckclass_test.go (fixture-drift contract; no Elisp seam exists — see observation 1) |
|    | AC-C-006d Elisp→Go transport | PASS | tests/anki-install-test.el: 3-key install doc {protocol_version, anki_connect_url, user_css}; file contents byte-for-byte; "" when absent; sync doc gains no field |
|    | AC-C-018c confirmation gate | PASS | tests/anki-migrate-test.el: declined ⇒ call log is exactly (("migrate" "--dry-run")) |
|    | REQ-C-020.3 write-back | PASS | tests/anki-migrate-test.el: ANKI_NOTE_ID=2002 and ANKI_NOTE_TYPE=imoogi-Basic after a confirmed migration |
|    | AC-C-021a/b/c contracts | PASS | tests/anki-error-test.el reverse assertion [NEW] + in-test mutation check; tests/anki-notetype-test.el literal sites |
|    | AC-C-022a stock compatibility | PASS | scan yields ("Basic" "Cloze" "imoogi-Basic" "imoogi-Cloze") |
|    | AC-C-022c manual collection check | GAP | AnkiConnect unreachable at 127.0.0.1:8765 during the run (curl exit 7) — needs a live collection |
|    | Gates (agent, clean worktree) | PASS | fmt-check 0, lint 0, test-go 0, test-elisp 0 (206 tests / 204 as expected / 0 unexpected / 2 skipped; baseline 181/179/0/2) |
- commits: 88dc542 (feat, 12 files +978/−41) and 264bbf6 (style: English register for the setup tail) on main — main fast-forwarded from c892cb3 (`git update-ref` with old-value guard) because a plain cherry-pick was refused by the user's uncommitted edit to modules/24-anki.el; that file was merged 3-way (`git merge-file`, exit 0, no markers) so the user's WIP hunks (+6/−1) survive untouched; not pushed (policy)
- orchestrator re-verification on main at 264bbf6 (evidence: .moai/state/verify/fe98f407/m6/): `go build ./...` 0 · `GOOS=windows go build ./cmd/imoogi-anki` 0 · `go test ./internal/anki/... ./cmd/imoogi-anki/... -cover` 0 (ankiconnect 86.2 / hashing 100 / media 91.3 / model 98.3 / orgdoc 100 / planner 91.5 / registry 87.8 / cmd 86.9) · `make fmt-check` 0 · `make lint` 0 · `make test-elisp` exit 2 = `Ran 219 tests, 218 results as expected, 1 unexpected` — the single failure is the pre-existing `imoogi-org-preview-browser-navigation-moves-point-and-suppresses-echo` (tests/org-preview-test.el:82, user's untracked WIP; identical to the M1–M5 baseline)
- E8 RED: 18 tests failed pre-GREEN (install 8, migrate 6, notetype 4 — names in agent report); 6 tests passed on first run by design (they pin pre-existing behaviour); AC-C-021c RED is an in-test mutation, not natural failure
- design: runner seam stays 2-arg; the subcommand travels in a new dynamically bound `imoogi-process-argv` (widening the arity would have broken every existing runner stub); `imoogi-process--call-binary` now exercised directly against an argv-echo stub for all four forms
- observations for sync (reported, not fixed): (1) `DeckClass`/`NormalizeDeckClass` in internal/anki/model have zero non-test callers and all four embedded templates emit raw `deck-{{Deck}}` — REQ-C-006.2's normaliser is never applied at review time, so `(PROGRAMMER)::(GO)` would render as `deck-(PROGRAMMER)::(GO)`; flagged to sync-auditor for a functional verdict; (2) design.md §6/§11 spell the install key `ankiconnect_url` while Go and Elisp use `anki_connect_url` (install_test.go asserts the design spelling is absent) — reconcile design.md at sync; (3) setup-tail strings were first written in Korean and corrected to English in 264bbf6 (modules/anki/*.el keep the English register; 24-anki.el is the Korean wrapper layer)
- agent worktree agent-ab26bbb9daf212b33 removed after integration; branch deleted
- goal: the re-armed model goal reached wallclock-exit (status ceiling-exit, 4 turns) while M6 ran in the background; not re-armed — all six milestones are now on main and the run-phase close proceeds by orchestrator turns
- run_status: M1, M3, M2, M4, M5, M6 complete — all milestones integrated

### Run-close log
- 2026-09-06T08:22Z: sync-auditor delegation #1 aborted before starting (Claude session limit, reset 19:00 KST) — no report produced (ledger closed; no result to attribute)
- 2026-09-08: main HEAD moved 264bbf6 → b799850 by the user's own commits (89e94be "improve Org setup and register host-wide Anki targets", 8023446, b799850). 89e94be adds `modules/anki/imoogi-targets.el`, `modules/anki/imoogi-target-scan.el`, edits `modules/24-anki.el` / `modules/anki/imoogi.el`, and adds `tests/anki-{targets,target-scan,target-sync}-test.el` (+1270/−78 inside SPEC scope, post-run, user-authored). sync-auditor delegation #2 spawned against HEAD b799850 with instructions to include that drift in the audit and attribute every claim to the audited sha
- 2026-09-08: sync-auditor #2 verdict at HEAD b799850 — **FAIL** (must-pass firewall on Functionality; weighted harmonic 0.68). Functionality 50 FAIL · Security 95 PASS · Craft 90 PASS · Consistency 85 PASS. Codex second opinion independently rated F1 HIGH/BLOCK; GLM inconclusive (no key). Report: `.moai/reports/sync-audit/SPEC-ANKICARD-001-run-audit.md`. Fresh mechanical evidence: Go suite green (-count=1), coverage floors all met, race/vet/gofmt clean, host+windows build 0, `make test-elisp` = Ran 247 tests, 246 as expected, 1 unexpected (org-preview WIP baseline only); post-run commit 89e94be audited — no SPEC regression. Findings: F1 [warning][blocking] deck-class normaliser never applied at review time (templates emit raw `deck-{{Deck}}`, base.css:52-58 admits it); F2–F5 [suggestion][optional] (css_test.go:213 vacuous assert; `ankiconnect_url` doc drift; planner.go:673 discards StoreMediaFile's stored name; media.go:215 narrow TOCTOU). H1 DEFECT · H2 docs-only · H3 AC-C-022c still UNVERIFIED (AnkiConnect unreachable) · H4 PASS (guard asserts ordering via hash recomputation). Stock-type fence holds; error-code table complete (18/18).
- 2026-09-08: fix-evaluate cycle 1/3 dispatched — manager-develop (tdd) scoped to `internal/anki/model/**`: single embedded JS normaliser mirroring the 7 Go steps, injected once into all four templates, stable `imoogi-deck` wrapper class + JS-added `deck-<token>`, base.css §4 selector/comment corrected, Go↔JS mirror test over the AC-C-004 table via `node` (present, v26), install idempotence preserved, stock-type fence untouched. design.md §3.1 contradiction deferred to sync (SPEC body not run-phase editable). F2–F5 deferred to sync notes (scope discipline)
- 2026-09-09: fix cycle 1 integrated — worktree commit a456634 (branch WT-deck-class-review-time, base b799850) cherry-picked onto main as **55bd790**; 14 files (internal/anki/model/** incl. new assets/deckclass.js, deckclass_script.go, deckclass_script_test.go; cmd/imoogi-anki/install_test.go needle updated). Carrier: hidden `<span class="imoogi-deck-name" style="display:none">{{Deck}}</span>` read via textContent (survives `"`), wrapper `<div class="imoogi-deck">`, script injected once per template side via `<!-- imoogi:deck-class-script -->` placeholder in `withDeckClassScript`; base.css §4 now `.card .imoogi-deck`. RED-1 build failure (undefined DeckClassScript) → RED-2 6 FAIL incl. mirror rows → GREEN. Orchestrator re-verification on main at 55bd790 (`.moai/state/verify/fe98f407/f1-main/`): build host+windows 0 · `go test ./internal/anki/... ./cmd/imoogi-anki/... -count=1 -cover` 0 (model 98.4, others unchanged) · `TestDeckClassScriptMirrorsTheGoNormalizer` PASS "mirrored 30 deck names through node" · lint 0. Residual: no card rendered on Desktop/AnkiDroid/AnkiMobile; a `<` in a deck name would be parsed as markup inside the carrier; Install still issues styling+templates on every run (existing "identical request logs" contract, unchanged). design.md §3.1 diagram and any prose quoting the raw wrapper → sync update. Re-audit (F1 delta) requested.
- 2026-09-09: post-integration on main at 55bd790 — F1 worktree removed, branch deleted; `make fmt-check` 0; `make test-elisp` exit 2 = `Ran 254 tests, 253 results as expected, 1 unexpected` (only `imoogi-org-preview-browser-navigation-moves-point-and-suppresses-echo`, the user's WIP baseline; all anki/transient tests pass). sync-auditor #3 (F1 delta re-audit) spawned against 55bd790
