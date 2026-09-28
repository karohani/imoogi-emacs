# SPEC Review Report: PROJECT (.moai/project/{product,structure,tech}.md)
Iteration: 1/3
Verdict: FAIL
Overall Score: 0.55

## Scope Note — Document-Type Adaptation

Reasoning context ignored per M1 Context Isolation (none was supplied for this invocation).

The plan-auditor checklist (M3/M5/Groups 1-8) is designed for SPEC documents (`spec.md` + `REQ-XXX`/`AC-XXX`/12-field YAML frontmatter/GEARS notation). The three audited documents (`product.md`, `structure.md`, `tech.md`) are **project-level descriptive documents** produced by the `moai-workflow-project` skill, not SPEC artifacts — they carry no YAML frontmatter, no `REQ-XXX`, no `AC-XXX`, and no `plan.md`/`research.md` companions. Verified directly:

```
grep -c "SPEC-" .moai/project/{product,structure,tech}.md   → 0, 0, 0
grep -in "^### REQ-\|^REQ-" .moai/project/*.md              → (no matches)
grep -in "^AC-\|^### AC-" .moai/project/*.md                → (no matches)
head -5 on all three files                                  → plain "# Title" headers, no frontmatter
```

Consequently MP-1, MP-2, MP-3, and MP-7 are **N/A by document-type mismatch** (the entities they bind do not exist in this document class) — not silently passed, and not treated as failures. The audit instead applies the checks that generalize to this document type: internal consistency (Group 6), structural completeness against the project-doc template (Group 2, adapted), and — because the core purpose of `product.md`/`structure.md`/`tech.md` is to accurately describe the *current* state of the repository — **factual accuracy verified against the actual repository** (the direct analogue of GEARS/AC "testability": a claim a reader cannot independently verify, or that verification contradicts, is the project-doc equivalent of an untestable or false acceptance criterion). This substitution is disclosed explicitly per dimension below.

## Must-Pass Results
- [N/A] MP-1 REQ number consistency: no `REQ-XXX` entities exist in this document type (verified via grep, 0 matches in all three files)
- [N/A] MP-2 EARS/GEARS format compliance: no `REQ-XXX` entities exist to check against GEARS patterns
- [N/A] MP-3 YAML frontmatter validity: none of the three files carry YAML frontmatter (`head -5` on each shows a bare `# Title` heading, not a `---` block)
- [N/A] MP-4 Section 22 language neutrality: tech.md is scoped to a specific single project's stack (Emacs Lisp + Go), not a multi-language tooling template — the 16-language enumeration rule does not apply
- [N/A] MP-5 D7 cross-SPEC reconciliation: `grep -c "SPEC-"` returns 0 for all three files — no cross-SPEC references exist to reconcile
- [PASS] MP-6 D8 cross-platform discipline: `grep -in "syscall" .moai/project/*.md` returns no matches — D8 auto-PASS per D8-4 (no `syscall` mention, no cross-platform-discipline concern)
- [N/A] MP-7 clarification gate: neither `plan.md` nor `research.md` exists for this document type (these are project docs, not SPEC artifacts)

**No must-pass criterion forces FAIL.** The verdict below is driven entirely by the category-score findings (Group 6 consistency + factual-accuracy verification), documented next.

## Category Scores (0.0-1.0, adapted rubric)
| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.75 | Minor ambiguity in one or two areas a reasonable engineer would resolve consistently | Prose is unambiguous throughout; the one soft spot is tech.md:L109's self-contradictory count ("the other 8 Go packages resolve fine" while `go list ./...` shows 9 non-`fetch` packages even under the doc's own now-false premise — see D2) |
| Completeness | 0.75 | One non-critical section missing or sparse; all required product/structure/tech sections otherwise present | All three docs cover their expected sections fully (product.md: Name/Description/Audience/Features/Use-Cases/Non-Goals; structure.md: Overview/Tree/Purposes/Key-Files/Module-Convention/Go-Organization; tech.md: Stack/Dependencies/Rationale/Dev-Env/Build-Workflow/Testing/Known-Issues/Verification/Scope) — docked only because the "Known Issues" section (a required, load-bearing section for tech.md) is materially wrong, which is a completeness defect in the sense that it fails to completely and correctly capture current state |
| Testability (repurposed: Factual Verifiability) | 0.25 | Most claims are subjective or, when checked, several core factual claims are directly contradicted by verification | Multiple concrete, independently-checkable claims fail verification — see Defects D1-D4 below. This dimension is repurposed from AC binary-testability (N/A for prose project docs) to the project-doc analogue: can a reader verify each factual claim, and does verification confirm it? |
| Traceability | N/A | — | No `REQ-XXX`/`AC-XXX` entities exist in this document type; traceability as defined by the rubric does not apply. Substituted by Group 6 Consistency (below), which is fully applicable and is where the found defects concentrate |

## Group 6 — Consistency (substituting for Traceability)
- CN-1 (no two requirements contradict): **FAIL** — see D1-D4; tech.md and structure.md each assert a build-failure/missing-file state that is directly contradicted by the repository's actual current state
- CN-2 (exclusions don't conflict with included requirements): PASS — Non-Goals (product.md:L45-47) and Scope Boundaries (tech.md:L117-121) are mutually consistent (darwin/arm64-only, personal-use-only, both stated identically in both docs)
- CN-3 (priority/labels consistent with stated scope): N/A — no priority/label fields exist in this document type

## Defects Found (structured defect-list)

D1. `stale-build-status` — `tech.md:L107-109` (§ Known Issues / Current State) — The document asserts: *"`go build ./...` currently fails. `internal/fetch/fetch.go` references two undefined helpers (`buildBootstrap`, `copyFile`) and carries an unused `encoding/json` import."* Verified directly against the repository:
   ```
   $ go build ./...
   (exit 0, no output)
   $ grep -n "func buildBootstrap\|func copyFile" internal/fetch/fetch.go
   601:func buildBootstrap(ctx context.Context, opts Options, repoRoot, staging, cliVersion string) ([]stagedArtifact, error) {
   778:func copyFile(source, destination string, mode os.FileMode) error {
   ```
   `go build ./...` succeeds with exit code 0, and both `buildBootstrap` and `copyFile` are defined in `internal/fetch/fetch.go` (not undefined as claimed). — Severity: **critical** — Class: **blocking** — Required fix: remove or rewrite the "Known Issues / Current State" section in tech.md to reflect the current, passing build state (or re-verify and re-word if some other build failure exists that this audit did not reproduce).

D2. `stale-test-file-claim` — `tech.md:L101` (§ Testing → Go side) — The document states: *"There is no `internal/fetch/fetch_test.go` — consistent with that package's current mid-refactor state (see below)."* Verified: `internal/fetch/fetch_test.go` exists (17,574 bytes, confirmed via `ls -la internal/fetch/` and `find internal cmd -name "*_test.go"`). — Severity: **major** — Class: **blocking** — Required fix: update the Go-side test-file enumeration in tech.md to include `internal/fetch/fetch_test.go` alongside the other seven listed `*_test.go` files.

D3. `stale-package-caveat` — `structure.md:L158` (§ Go Package Organization) — The document states: *"**Current caveat**: `internal/fetch` is mid-refactor and does not currently compile (`go build ./...` fails there — see `tech.md` § Known Issues). All other 8 Go packages resolve cleanly."* This is the same false premise as D1 (verified `go build ./...` succeeds, exit 0), and the package count is independently off even under the doc's own (false) premise: `go list ./...` enumerates 10 packages total (`cmd/imoogi-toolchain` + 9 `internal/*` packages including `fetch`); excluding `fetch` leaves 9 "other" packages, not 8 as claimed. — Severity: **critical** — Class: **blocking** — Required fix: remove this caveat entirely (the underlying premise is false) or, if some other build issue is meant, re-verify against the actual `go build ./...` / `go list ./...` output and correct the package count.

D4. `nonexistent-vendored-toolchain-directory` — `structure.md:L55-56` (Directory Tree) and `tech.md:L80` (§ Go toolchain vendoring) — structure.md's directory tree lists `vendor/toolchains/` as an existing subdirectory ("Node/TypeScript/gopls/CLI binaries, incl. the imoogi-toolchain binary itself (`vendor/toolchains/cli/1.0.0/darwin-arm64/`)"), and tech.md states the CLI binary is *"treated as a vendored toolchain artifact (committed at `vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain`, with its SHA-256 recorded in `docs/toolchains.md`)"*. Verified: `find vendor -maxdepth 2 -type d` and `find vendor -iname "*toolchain*"` show `vendor/` contains only `elpa/`, `ghostel-module/`, and `tree-sitter/` — **no `toolchains/` subdirectory exists anywhere under `vendor/`**. Additionally, `toolchains.lock.json` — listed in structure.md's Key File Locations (L125: "Toolchain resolved lockfile: `toolchains.lock.json`") as a current key file — does not exist (`ls toolchains.lock.json` → No such file or directory); only `toolchains.json` (the desired-state manifest) exists. — Severity: **critical** — Class: **blocking** — Required fix: either (a) actually run the vendoring workflow and commit `vendor/toolchains/` + `toolchains.lock.json` before claiming they exist, or (b) rephrase both documents to describe the toolchain-vendoring artifacts as a documented *workflow/design*, not as already-committed current state — e.g. move the specific committed-path claims out of the "current state" directory tree and Key File Locations sections into the workflow-description sections where they already partially live (tech.md § Build and Vendoring Workflow correctly describes this as a *process*; the inconsistency is that other sections of the same two documents assert the *result* of that process as already-existing fact).

D5. `stale-package-count-in-known-issues` — `tech.md:L109` — Even taken at face value (ignoring D1/D3's falsified premise), the phrase "the other 8 Go packages resolve fine via `go list ./...`" undercounts: `go list ./...` enumerates 9 non-`fetch` packages (`cmd/imoogi-toolchain`, `internal/activation`, `internal/artifact`, `internal/cli`, `internal/config`, `internal/lang`, `internal/lang/golang`, `internal/lang/typescript`, `internal/setup`). — Severity: **minor** — Class: **optional** (subsumed by D1/D3's required fix — once the Known Issues section is corrected or removed, this miscount is moot) — Required fix: none needed independently; resolved as a side effect of fixing D1/D3.

No other defects found. Structural completeness (Group 2 adapted), Non-Goals/Scope-Boundaries consistency (CN-2), and cross-platform discipline (D8/MP-6) all pass with cited evidence above.

## Regression Check (Iteration 2+ only)
N/A — this is iteration 1.

## Recommendation

FAIL. The failure is not structural (all three documents are well-organized, clearly written, and internally consistent in scope/audience/non-goals) — it is **factual accuracy**, which is the core value proposition of `product.md`/`structure.md`/`tech.md` as onboarding/context documents. Four of five confirmed defects (D1, D3, D4, and D2) form a single coherent cluster: the documents describe the Go toolchain side of the project (`internal/fetch`, the vendored `imoogi-toolchain` binary, `toolchains.lock.json`) as if a "mid-refactor, currently broken, not-yet-vendored" state were still current, when the repository's actual current state shows the build passing, the missing helper functions present, the test file present, and only the *toolchain-artifact vendoring step itself* (an online-machine, git-committed step) not yet executed.

Numbered fix instructions for the document author (likely `manager-docs` / `moai project` regeneration):

1. Re-run `go build ./...` and `go list ./...` against the current tree before writing any "Known Issues" or "current caveat" language, and either remove tech.md's § Known Issues / Current State section (L107-109) and structure.md's Go Package Organization caveat (L158) entirely, or replace them with whatever build issue is *actually* reproducible today.
2. Correct tech.md's Go-side testing enumeration (L99-101) to include `internal/fetch/fetch_test.go`.
3. Either commit `vendor/toolchains/` (with the CLI binary and toolchain artifacts) and `toolchains.lock.json` to match what structure.md/tech.md currently assert as fact, or rewrite structure.md's directory tree (L55-56) and Key File Locations (L125) plus tech.md's § Go toolchain vendoring (L80) to describe the toolchain-vendoring artifacts as a **not-yet-executed workflow** rather than already-committed state — matching the actual `vendor/` contents (`elpa/`, `ghostel-module/`, `tree-sitter/` only) and the actual root-level files (`toolchains.json` present, `toolchains.lock.json` absent).
4. Re-verify all "current state" factual assertions in tech.md and structure.md against the live repository one more time before resubmitting for iteration 2 — the defects found here are the kind a single `go build ./... && go list ./... && find vendor -maxdepth 2 -type d` pass would have caught.

Verdict: FAIL
