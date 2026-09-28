# SPEC Review Report: SPEC-AGENTIPC-001
Iteration: 1/3
Verdict: PASS
Overall Score: 0.89

Tier: M (from `spec.md:L14` `tier: M`) — PASS threshold 0.80. Aggregate is the harmonic mean of the four rubric dimensions.
Inputs read (Tier M contract): `spec.md` (384 lines), `plan.md` (272), `acceptance.md` (362). Also read for MP-7 and cross-checks: `research.md` (336), `progress.md`, `spec-compact.md` (head).
No author reasoning context was received in the prompt; audit is based on the artifact files only (M1).
Audit backend: Claude-only (no `audit_model` key found under `.moai/config/`, no MCP backend invoked).

## Must-Pass Results

- [PASS] MP-1 REQ number consistency: `grep '^#### REQ-' spec.md` returns exactly REQ-AIPC-001 … REQ-AIPC-015 at spec.md:L154, L162, L170, L183, L192, L202, L209, L217, L228, L239, L251, L259, L270, L276, L290 — sequential, three-digit zero-padding, no gaps, no duplicates. The domain-segmented `REQ-AIPC-NNN` form is consistent throughout (spec.md §7 L370–L384, acceptance.md AC headers).
- [PASS] MP-2 EARS/GEARS format compliance — judged against the **requirement layer only** (spec.md §3 canonical English sentences); the Given-When-Then entries in acceptance.md were graded under Group 4, not here. All 15 canonical sentences match a GEARS pattern:
  - Ubiquitous "The <subject> shall …": REQ-001 L156 ("The Emacs receiver shall expose exactly two entry points…"), REQ-002 L164, REQ-007 L211, REQ-010 L241, REQ-012 L261, REQ-013 L272, REQ-014 L278 (compound "shall map … and shall terminate"), REQ-015 L292.
  - Event-driven "When …, the <subject> shall …": REQ-003 L172, REQ-004 L185, REQ-005 L194, REQ-006 L204, REQ-009 L230, REQ-011 L253.
  - Unwanted (canonical negative) "The receiver shall not …": REQ-008 L219.
  - No `If … then` legacy form, so no deprecation marker is required.
- [PASS] MP-3 YAML frontmatter validity: spec.md:L2–L13 carries all 12 canonical fields with correct types — `id: SPEC-AGENTIPC-001`, `title` quoted, `version: "0.1.0"` quoted semver, `status: draft` (enum), `created: 2026-09-27` / `updated: 2026-09-27` (ISO), `author: jay`, `priority: P1`, `phase: "v0.x agent-ipc MVP"` (release label, not a prohibited stage token), `module` path-like, `lifecycle: spec-anchored`, `tags` comma-separated string. No rejected alias (`created_at`/`updated_at`/`labels`/`spec_id`). Optional `tier: M` at L14 is valid.
- [N/A] MP-4 Section 22 language neutrality: N/A — single-repository Go + Emacs Lisp project, not template-bound multi-language tooling. Auto-pass.
- [PASS] MP-5 D7 cross-SPEC reconciliation: `grep -Eoh 'SPEC-([A-Z][A-Z0-9]+-)+[0-9]+' spec.md plan.md acceptance.md | sort | uniq -c` → only `SPEC-AGENTIPC-001` (self, 3 hits). No external SPEC referenced; no retired/superseded/archived reference; no BLOCKING finding.
- [PASS] MP-6 D8 cross-platform discipline: `grep -n 'syscall' .moai/specs/SPEC-AGENTIPC-001/*.md` → no output. D8 auto-PASS.
- [PASS] MP-7 clarification gate: `grep -rn '\[NEEDS CLARIFICATION' plan.md research.md` → no output. plan.md §8 L263–L265 states "없음". research.md "contradictions" items 1/3/4/7/8 are resolved in plan.md §1 L27–L29, §3.2 L144, D-7 L76–L79, R-3 L238; items 2/5/6 are immaterial to the design or superseded by plan-phase measurement (plan.md §1 table L18–L20).

## Category Scores (0.0-1.0, rubric-anchored)

| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.80 | 0.75 band (upper) | Contract tables are precise (spec.md §2.1–§2.5 L70–L140; reason enum L104–L116; exit codes L120–L125; violation order L178). Deductions: plan.md D-1 L36–L37 contradicts the status-string contract (D1 below); "one line" promises at L185/L211 undefined for multi-line `text`/`summary` (D4); optional-field type mismatch for `[]`/`{}` unreachable under the chosen parse options (D3). |
| Completeness | 1.00 | 1.0 band | HISTORY spec.md:L17; WHY §1.1 L34; WHAT §1.2 L48; protocol contract §2 L66; REQUIREMENTS §3 L147; Constraints §5 L340; Brownfield delta §6 L349; Traceability §7 L366; ACCEPTANCE in acceptance.md (15 ACs, edge-case table §4 L334, quality gate §5 L348, DoD §6 L356); Out of Scope with six `### Out of Scope — <topic>` H3 headings each with `-` bullets (L306–L338). 12/12 frontmatter fields. REQ 15 / AC 15 within the Tier M ceiling of 16 each. |
| Testability | 0.85 | 0.75–1.0 band | Every AC has a binary verification command (`ert_sel "^imoogi-agent-acNN-"`, `go test … -run 'TestACNN'`, explicit shell block in AC-015) and an explicit pass condition (acceptance.md L22, L37). No weasel words ("appropriate/adequate/reasonable/proper" absent). Deductions: AC-012 L250 looser than REQ-011 (D2); echo-area observation surface unnamed (O4); >1 MiB TEXT case only satisfiable in-process (O5). |
| Traceability | 0.95 | 1.0 band (minor sub-clause gaps) | spec.md §7 L370–L384 maps every REQ to ≥1 AC; every AC header in acceptance.md (L54, L65, L75, L90, L100, L115, L131, L156, L174, L197, L217, L243, L255, L269, L295) cites only existing REQs, and the header lists match §7 bidirectionally. Deduction: a few REQ sub-clauses have no scenario (O2, O3). |

Harmonic mean: 4 / (1/0.80 + 1/1.00 + 1/0.85 + 1/0.95) = 0.89 ≥ 0.80 (Tier M).

## Defects Found (structured defect-list)

D1. AIPC-R1-D1 — plan.md:L36–L37 (D-1) — plan permits a status string `"error:<reason>: <detail>"` and tells the CLI to parse the reason "첫 `:` 뒤부터 다음 `:` 또는 끝까지". This contradicts the normative contract: spec.md:L102 ("`<reason>` 은 다음 중 하나다"), REQ-AIPC-001.2 L159 ("반환값은 항상 `"ok"` 또는 `"error:<reason>"`"), REQ-AIPC-014.2 L281, and the exact-string expectations in AC-006 L125 / AC-007 L136–L150 / AC-013 L259. An implementer following D-1 would fail those ACs. — Severity: minor — Class: blocking (internal consistency) — Required fix: delete the detail-suffix sentence from plan.md D-1 (keep the `condition-case` → `"error:handler"` sentence). If a human-readable detail is actually wanted, amend §2.3, REQ-001.2, REQ-014.2 and the affected ACs together instead.

D2. AIPC-R1-D2 — acceptance.md:L250 — AC-AIPC-012 accepts "stderr 에 진단 한 줄 이상", while REQ-AIPC-011 spec.md:L253 requires "write one diagnostic line to stderr". A two-line diagnostic passes the AC but violates the REQ. — Severity: minor — Class: blocking (AC does not verify its REQ as stated) — Required fix: change AC-012 to "stderr 에 진단 정확히 한 줄", or relax REQ-011 to "at least one diagnostic line".

D3. AIPC-R1-D3 — plan.md:L117–L119 vs spec.md:L82–L83, L113, L166 — the chosen parse options `:object-type 'alist :array-type 'list :null-object nil :false-object nil` map `[]` and `{}` to `nil`. Probe (this audit, Emacs 30.2 `--batch -Q`): `(json-parse-string "{\"project\":[],\"session\":{},\"payload\":{\"text\":\"x\",\"summary\":[]}}" …)` → `((project) (session) (payload (text . "x") (summary)))`. So `"project":[]`, `"session":{}`, `"summary":[]`, `"title":{}` become indistinguishable from "absent" and would be accepted as `"ok"`, while §2.1 types `project`/`session` as strings and §2.3 L113 lists "형식 불일치" under `bad-field`. — Severity: minor — Class: blocking (stated criterion unreachable by the planned mechanism) — Required fix: either (a) state explicitly in spec.md §2.1/§2.2 (and REQ-002) that an empty array/object in an optional field is treated as absent, or (b) change plan.md §3.1 to parse options that preserve the distinction (e.g. `:array-type 'array` plus a distinct `:null-object`/object representation) and add an AC-007 row such as `project = []` → `"error:bad-field"`.

D4. AIPC-R1-D4 — spec.md:L185, L211, L92, L96 — REQ-AIPC-004 promises "one notification line in the echo area" and REQ-AIPC-007 "one line appended" to the log, but `text` (L92) is constrained only to be non-empty and `summary`/`title` only to be strings; an embedded newline produces a multi-line echo and a multi-line log entry, breaking both REQs and making the log unparseable line-by-line. — Severity: minor — Class: blocking (violates a criterion the document states) — Required fix: define newline handling in REQ-004/REQ-007 (e.g. "newlines in displayed/logged values are replaced by a single space" or escaped) and add one AC-001 sub-case with `text` containing `\n`; or relax the REQs to "one entry".

Optional findings (surface to orchestrator; not required for this verdict):

O1. acceptance.md:L260–L261 — AC-013 tolerates an extra `--timeout=N` argument ("그 하나만 추가로 허용"), but plan.md D-9 L89 says emacsclient receives only `--eval EXPR PATH`, and REQ-014.4 L284 says the limit does not rely on emacsclient's own option. Dead allowance; tighten to exactly three arguments. — Severity: minor — Class: optional.

O2. spec.md:L197 — REQ-AIPC-005.2 second sentence ("선택된 창에만 보이면 다른 창에 한 번 더 표시한다") and the related case where the target file is the buffer shown in the selected window (user's point must not move) have no AC scenario; AC-002/003 always start from a different buffer A. — Severity: minor — Class: optional — Suggested fix: add AC-002(b) with F already shown only in the selected window, asserting window count +1, selected window unchanged, its point unchanged.

O3. spec.md:L219, L215 — REQ-AIPC-008 "write to … any payload file" and REQ-AIPC-007.3 (log buffer recreated after being killed) have no AC check; AC-001 L61 checks only the event file's content/mtime. — Severity: minor — Class: optional.

O4. acceptance.md:L58, L94, L108 — "에코 영역에 … 나타나며" does not name the observation surface. Probe (this audit): in Emacs 30.2 `--batch`, `(current-message)` returns `nil` after `(message "hello")`, while `*Messages*` does record the line. The ACs are testable, but a tester using `current-message` gets a false FAIL. — Severity: minor — Class: optional — Suggested fix: state "마지막 `*Messages*` 줄이 …" (or "`message` 호출 인자를 기록해 …").

O5. acceptance.md:L249 — "1 MiB 를 넘는 TEXT" is listed as an invocation "실행하면". A single >1 MiB argv element is refused by the OS at exec time on both macOS (ARG_MAX 1 MiB total) and Linux (per-argument limit), so this case is only satisfiable by calling `run(...)` in-process (plan.md §3.2 L136 already designs for this), and REQ-011's "완성된 이벤트가 1 MiB 초과" branch is effectively unreachable from a real shell. (Platform limit stated from general knowledge; not executed in this audit.) — Severity: minor — Class: optional — Suggested fix: note in AC-012 that this case runs through in-process `run(...)`.

O6. spec.md:L230–L235 / plan.md:L123 — no explicit decision is recorded on containment/allowlisting of payload paths (research.md L215 flags "no allowlist or containment check"); any absolute local regular file may be opened. This is a legitimate MVP choice for a same-user tool, but it should appear as an explicit Out of Scope bullet so the decision is visible. — Severity: minor — Class: optional.

O7. spec.md:L240–L246, L272, L261 — some REQs carry implementation-level identifiers (`server-eval-args-left`, `point-max`, exact app-bundle paths, Makefile target names). These are justified as the user-confirmed external contract (§2 L68) and safety-bearing, so no change is required; noted for RQ-4 completeness. — Severity: minor — Class: optional.

## Regression Check (Iteration 2+ only)

N/A — iteration 1.

## Verification performed in this audit

Verified by reading/running (this run, this tree, HEAD `fd7f0b0` per gitStatus):
- REQ-013 order equals `scripts/imoogi-editor` `find_emacsclient` L5–L37 (`$EMACSCLIENT` must be executable → PATH → `Emacs-${EMACS_VERSION:-31.1}.app` → `Emacs.app` → `Emacs-*.app` glob). Match.
- `ls cmd` → 6 CLIs (`imoogi-anki imoogi-clip imoogi-notes imoogi-org-preview imoogi-provenance imoogi-toolchain`), so "일곱 번째 CLI" (spec.md:L45) holds.
- boot.el module `dolist` at L65 ending with `"org/29-org-roam"` at L101; `imoogi-require` at L44 (matches plan.md §5 L222–L223).
- tests/module-layout-test.el expected list ends with `"29-org-roam"` (plan.md L164).
- scripts/install.sh `CONFIG_LINK="${HOME}/.config/imoogi-emacs"` L16, `EDITOR_LINK` L19, `ln -sfn` L88–L89; tests/setup-toolchain-test.sh runs install.sh under a fake `HOME` and checks `-L`/`readlink` at L101–L102 — AC-015's link assertion is implementable the same way.
- Makefile: `.PHONY` L35, `build-all` L109, `build-notes` L132, `ci-local` L189 exist.
- modules/org/28-clipboard.el L97–L98 uses the cited parse options.
- go.work, packages.el, packages.lock, vendor/ exist (AC-015 `git diff --quiet` pathspecs resolve).
- Two Emacs 30.2 batch probes (D3, O4).

Not verified (author-reported, relied on as stated): the HISTORY emacsclient measurements (spec.md:L24–L30, plan.md §1 table), the `ert_sel` advice selector result (acceptance.md L10–L12), the D-5 batch window-split measurement (acceptance.md L26–L29), and `make ci-local` baseline status.

## Recommendation

PASS. Rationale per must-pass criterion: MP-1 sequential REQ-AIPC-001..015 (spec.md:L154–L290); MP-2 all 15 canonical sentences in GEARS form (requirement layer); MP-3 12/12 canonical fields (spec.md:L2–L13); MP-4 N/A; MP-5 no external SPEC references; MP-6 no `syscall`; MP-7 no clarification markers. Aggregate 0.89 ≥ Tier M threshold 0.80.

The four blocking-minor defects are one-line edits and should be applied by manager-spec before Implementation Kickoff Approval:
1. D1 — remove the `"error:<reason>: <detail>"` allowance from plan.md D-1 (L36–L37).
2. D2 — align AC-012 L250 with REQ-011 L253 ("exactly one line" on both sides, or "at least one" on both).
3. D3 — decide `[]`/`{}` semantics for optional fields: declare "treated as absent" in spec.md §2.1/§2.2, or change plan.md §3.1 parse options and add an AC-007 row.
4. D4 — define newline handling for echoed/logged values in REQ-004/REQ-007 and add an AC-001 sub-case.

Optional findings O1–O7 are left to orchestrator discretion.
