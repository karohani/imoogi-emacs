# SPEC Review Report: PROJECT (.moai/project/{product,structure,tech}.md)
Iteration: 2/3
Verdict: FAIL
Overall Score: 0.65

## Scope Note — Document-Type Adaptation

Reasoning context ignored per M1 Context Isolation (none was supplied for this invocation).

Same adaptation as iteration 1: `product.md`, `structure.md`, `tech.md` are project-level descriptive documents produced by the `moai-workflow-project` skill, not SPEC artifacts (no YAML frontmatter, no `REQ-XXX`/`AC-XXX`, no `plan.md`/`research.md`). Re-verified:

```
grep -c "SPEC-" .moai/project/{product,structure,tech}.md   → 0, 0, 0
head -3 on all three files                                  → plain "# Title" headers, no frontmatter
```

MP-1, MP-2, MP-3, and MP-7 remain N/A by document-type mismatch. Per the Retry Loop Contract, this iteration's re-audit is scoped to the enumerated defect delta from `PROJECT-review-1.md` (D1-D5), plus a fresh factual-verification sweep against the live repository, since the previous FAIL was driven entirely by factual-accuracy (Group 6 / repurposed-Testability) defects rather than structural gaps.

## Must-Pass Results
- [N/A] MP-1 REQ number consistency: no `REQ-XXX` entities exist in this document type
- [N/A] MP-2 EARS/GEARS format compliance: no `REQ-XXX` entities exist to check
- [N/A] MP-3 YAML frontmatter validity: none of the three files carry YAML frontmatter (re-verified, `head -3` shows bare `# Title`)
- [N/A] MP-4 Section 22 language neutrality: tech.md is scoped to a single project's stack (Emacs Lisp + Go), not a multi-language tooling template
- [N/A] MP-5 D7 cross-SPEC reconciliation: `grep -c "SPEC-"` returns 0 for all three files (re-verified)
- [PASS] MP-6 D8 cross-platform discipline: `grep -in "syscall" .moai/project/*.md` returns no matches (re-verified, exit 1 / no hits) — D8 auto-PASS per D8-4
- [N/A] MP-7 clarification gate: neither `plan.md` nor `research.md` exists for this document type

**No must-pass criterion forces FAIL.** As in iteration 1, the verdict is driven by the category-score findings (Group 6 consistency + factual-accuracy verification), documented next.

## Regression Check (Iteration 2)

Verified each of iteration 1's five defects against the live repository:

- **D1** (`stale-build-status`, tech.md — "go build ./... currently fails") — **RESOLVED**. tech.md no longer contains a "Known Issues / Current State" section asserting a build failure. Re-verified: `go build ./...` exits 0.
- **D2** (`stale-test-file-claim`, tech.md L101 — "no `internal/fetch/fetch_test.go`") — **RESOLVED**. tech.md's current § Testing → Go side (L101) now enumerates `internal/fetch/fetch_test.go` alongside the other seven `*_test.go` files. Re-verified: `internal/fetch/fetch_test.go` exists (17,590 bytes).
- **D3** (`stale-package-caveat`, structure.md L158 — "8 Go packages resolve fine", false mid-refactor caveat) — **RESOLVED**. structure.md's current § Go Package Organization (L160) states: *"`go build ./...` and `go list ./...` currently succeed cleanly across all 10 packages (`cmd/imoogi-toolchain` + 9 `internal/*` packages)."* Re-verified: `go list ./...` enumerates exactly 10 packages (1 `cmd/imoogi-toolchain` + 9 `internal/*`), matching the doc exactly.
- **D4** (`nonexistent-vendored-toolchain-directory`, structure.md L55-56 / tech.md L80 — claimed `vendor/toolchains/` and `toolchains.lock.json` already exist) — **RE-OPENED (regressed with inverted polarity)**. The iteration-1 fix correctly rewrote both documents to state that `vendor/toolchains/` and `toolchains.lock.json` "do not yet exist in this checkout" / "have not been run in this checkout" — accurate at fix-time. Since then, the repository's actual state has changed: `vendor/toolchains/` (with `go/`, `licenses/`, `typescript/`, `cli/`, `node/` subdirectories) and `toolchains.lock.json` now **exist on disk** in this checkout (verified: `find vendor -maxdepth 2 -type d` lists `vendor/toolchains`; `ls toolchains.lock.json` succeeds; `git status --short` shows both as untracked `??`, i.e. present-but-uncommitted, not absent). The documents' current "does not yet exist" / "has not been run" claims are therefore false again, in the opposite direction from iteration 1. See D6 below for full citation and fix.
- **D5** (`stale-package-count-in-known-issues`, tech.md L109 — "8 packages" undercounted) — **RESOLVED** (subsumed by D3's fix; the corrected "10 packages" statement in structure.md L160 is internally consistent with `go list ./...`).

**3 of 5 iteration-1 defects are cleanly resolved (D1, D2, D3/D5). D4 is not resolved — the underlying pattern (documents lagging the live toolchain-vendoring state) recurred with the claim's truth value flipped**, which the Retry Loop Contract's stagnation-detection clause treats as continued lack of convergence on the toolchain-vendoring topic, not as a fresh unrelated defect — see § Recommendation.

## Category Scores (0.0-1.0, adapted rubric)
| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.75 | Minor ambiguity in one or two areas a reasonable engineer would resolve consistently | Prose remains unambiguous throughout; no new clarity issues introduced by the iteration-1 fixes |
| Completeness | 0.75 | One non-critical section missing or sparse; all required product/structure/tech sections otherwise present | All required sections present and well-organized (unchanged from iteration 1); docked because the toolchain-vendoring state description (structure.md Directory Tree/Purposes/Key Files, tech.md § Go toolchain vendoring) is materially wrong again, per D6 |
| Testability (repurposed: Factual Verifiability) | 0.25 | Most claims are subjective or, when checked, several core factual claims are directly contradicted by verification | 3 of 5 prior factual defects fixed, but 3 new/re-opened factual defects found this iteration (D6, D7, D8) — see below. The dimension stays in the 0.25 band because the remaining defects are concentrated in the same core "current toolchain-vendoring state" claims a reader would rely on most, even though total defect count dropped |
| Traceability | N/A | — | No `REQ-XXX`/`AC-XXX` entities exist; substituted by Group 6 Consistency below |

## Group 6 — Consistency (substituting for Traceability)
- CN-1 (no two requirements contradict): **FAIL** — see D6-D8; structure.md and tech.md both assert `vendor/toolchains/`/`toolchains.lock.json` non-existence, contradicted by the actual checkout; tech.md's LSP-runtime version list is internally self-contradictory against its own repository's toolchain manifests
- CN-2 (exclusions don't conflict with included requirements): PASS — Non-Goals (product.md:L45-47) and Scope Boundaries (tech.md:L117-121) remain mutually consistent (darwin/arm64-only, personal-use-only, stated identically in both docs)
- CN-3 (priority/labels consistent with stated scope): N/A — no priority/label fields exist in this document type

## Defects Found (structured defect-list)

D6. `stale-toolchain-artifact-nonexistence-claim` (continuation/re-open of iteration-1 D4) — `structure.md:L56-58`, `structure.md:L90-91`, `structure.md:L113`, `structure.md:L126-127`, `tech.md:L80` — Four locations across two files assert that `vendor/toolchains/` and/or `toolchains.lock.json` "do not yet exist in this checkout" or that `imoogi-toolchain fetch` "has not been run in this checkout":
   - structure.md:L56-58: *"NOTE: vendor/toolchains/ does not yet exist in this checkout ... not yet executed here."*
   - structure.md:L90-91: *"toolchains.lock.json (resolved lockfile, written by `fetch`) does not yet exist in this checkout."*
   - structure.md:L113: *"...that directory does not yet exist in this checkout."*
   - structure.md:L126-127: *"Toolchain resolved lockfile: `toolchains.lock.json` (not yet present — written by `imoogi-toolchain fetch`, which has not been run in this checkout)"*
   - tech.md:L80: *"...but `vendor/toolchains/` and `toolchains.lock.json` ... do not yet exist — `vendor/` currently contains only `elpa/`, `ghostel-module/`, and `tree-sitter/`."*

   Verified directly against the repository:
   ```
   $ find vendor -maxdepth 2 -type d
   vendor
   vendor/tree-sitter
   vendor/ghostel-module
   vendor/elpa
   vendor/toolchains
   vendor/toolchains/go
   vendor/toolchains/licenses
   vendor/toolchains/typescript
   vendor/toolchains/cli
   vendor/toolchains/node
   $ ls toolchains.lock.json
   toolchains.lock.json
   $ git status --short vendor/toolchains toolchains.lock.json
   ?? toolchains.lock.json
   ?? vendor/toolchains/
   ```
   `vendor/toolchains/` and `toolchains.lock.json` both exist on disk in this checkout — the documents' "does not yet exist in this checkout" / "has not been run in this checkout" claims are about filesystem presence, not git-tracked status, and are false as written. (They are, correctly, untracked/uncommitted — a materially different and worth-stating fact the docs do not currently capture.) — Severity: **critical** — Class: **blocking** — Required fix: re-verify current `vendor/` and root-directory contents at rewrite time, and state the accurate three-part current state: (a) `imoogi-toolchain fetch` HAS been run in this checkout, (b) `vendor/toolchains/{go,licenses,typescript,cli,node}/` and `toolchains.lock.json` exist and are populated, (c) they are **not yet committed to git** (untracked). Update all five citations above accordingly.

D7. `unverifiable-node-version-claim` — `tech.md:L37` (§ LSP toolchain runtimes) — The document states: *"Node v22.23.2 / v24.19.0"*. Verified against every toolchain-state source in the repository:
   ```
   $ grep '"upstream_version"' toolchains.lock.json | grep -A1 '"name": "node"'   → "v24.19.0" (only)
   $ grep node_version toolchains.json                                            → "node_version": "v24.19.0" (only)
   $ find vendor/toolchains/node -maxdepth 1                                       → vendor/toolchains/node/v24.19.0 (only)
   ```
   No source in the repository's toolchain configuration, lockfile, or vendored artifact tree references Node v22.23.2 — only v24.19.0 exists everywhere. The `v22.23.2` figure traces to a stale "Current Bundle" section in `docs/toolchains.md:L11`, which is itself inconsistent with its own repository's `toolchains.lock.json` and vendored `vendor/toolchains/node/` directory (both v24.19.0) — tech.md propagated that stale reference without independent verification against the live lockfile/vendor tree. — Severity: **critical** — Class: **blocking** — Required fix: change tech.md:L37 to `Node v24.19.0` (the only version present in `toolchains.lock.json`, `toolchains.json`, and `vendor/toolchains/node/`), and flag `docs/toolchains.md:L11` (out of this audit's scope but the apparent source of the stale figure) for correction in a separate pass.

D8. `fabricated-typescript-version-claim` — `tech.md:L38` (§ LSP toolchain runtimes) — The document states: *"TypeScript 6.0.3 / 7.0.2"*. Verified:
   ```
   $ grep -rn "7\.0\.2" . --include="*" | grep -v "\.git/\|vendor/elpa\|node_modules"
   .moai/project/tech.md:38:- TypeScript 6.0.3 / 7.0.2      ← the only hit
   $ grep '"upstream_version"' toolchains.lock.json | grep -A1 typescript   → "6.0.3" (only)
   $ find vendor/toolchains/typescript -maxdepth 2                          → typescript/6.0.3 (only)
   ```
   `7.0.2` does not appear anywhere in `toolchains.json`, `toolchains.lock.json`, `docs/toolchains.md`, or the vendored `vendor/toolchains/typescript/` tree — it appears nowhere in the repository except this one line of tech.md. Only TypeScript 6.0.3 is real and vendored. — Severity: **critical** — Class: **blocking** — Required fix: change tech.md:L38 to `TypeScript 6.0.3` only (drop the unsourced `/ 7.0.2`), matching `toolchains.lock.json` and the vendored artifact path.

No other new defects found this iteration. Structural completeness (Group 2 adapted), Non-Goals/Scope-Boundaries consistency (CN-2), the numbered-modules count (22, re-verified), the packages.lock package count (85, re-verified against `vendor/elpa/` directory count), the Go package count and enumeration, the test-file enumeration, the CI-workflow claim (`label-sync.yml` only), and cross-platform discipline (D8/MP-6) all pass with cited evidence.

## Recommendation

FAIL, but materially improved from iteration 1 (0.55 → 0.65). Three of five prior defects are genuinely and correctly resolved (D1/D2/D3/D5). The remaining failure is a narrower, single coherent cluster: **the toolchain-vendoring artifact state (D6) and the specific version numbers within it (D7, D8) are stale/inaccurate again**, because the live repository's toolchain-vendoring state changed after the iteration-1 fix was written (someone ran `imoogi-toolchain fetch` since then, per the lockfile's `retrieved_at: 2026-08-22T02:51:57Z` timestamp), and the two version numbers were apparently copied from an already-stale `docs/toolchains.md` source without independent verification against the live lockfile.

This is a **stagnation-adjacent pattern, not a fresh unrelated defect**: the same subsystem (Go toolchain vendoring state) has now produced a factual-accuracy defect in both iteration 1 (claimed-exists-when-absent) and iteration 2 (claimed-absent-when-exists, plus two fabricated/stale version numbers). If iteration 3 produces a third defect in this same subsystem, it should be flagged per the Retry Loop Contract's stagnation-detection clause as a misunderstanding rather than a missed fix — the author appears to be writing this section from memory or from a stale companion doc rather than re-verifying live state at each rewrite.

Numbered fix instructions for the document author (likely `manager-docs` / `moai project` regeneration):

1. Before rewriting any toolchain-vendoring-state language, run this exact verification sequence and write the docs from its live output, not from memory or from `docs/toolchains.md`:
   ```
   find vendor -maxdepth 2 -type d
   ls toolchains.lock.json toolchains.json
   git status --short vendor/toolchains toolchains.lock.json
   grep '"upstream_version"' toolchains.lock.json
   ```
2. Fix structure.md:L56-58, L90-91, L113, L126-127 and tech.md:L80 (D6) to state: `vendor/toolchains/` and `toolchains.lock.json` exist and are populated (fetch has run), but are currently **untracked/uncommitted** in git.
3. Fix tech.md:L37 (D7) to `Node v24.19.0` only.
4. Fix tech.md:L38 (D8) to `TypeScript 6.0.3` only (remove the unsourced `7.0.2`).
5. As a secondary, out-of-scope note: `docs/toolchains.md:L11` ("Node: `v22.23.2`") is itself stale against its own repository's `toolchains.lock.json`/`vendor/toolchains/node/` (both v24.19.0) — worth a follow-up correction pass so future project-doc regenerations don't re-propagate it into tech.md a third time.
6. Re-run the full verification sequence in step 1 one more time immediately before resubmitting for iteration 3, and diff the doc's claims against that output line-by-line before writing.

Verdict: FAIL
