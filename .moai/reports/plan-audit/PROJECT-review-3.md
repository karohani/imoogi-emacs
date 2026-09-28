# SPEC Review Report: PROJECT (.moai/project/{product,structure,tech}.md)
Iteration: 3/3
Verdict: FAIL
Overall Score: 0.80

## Scope Note — Document-Type Adaptation

Reasoning context ignored per M1 Context Isolation (none was supplied for this invocation).

Same adaptation as iterations 1-2: `product.md`, `structure.md`, `tech.md` are project-level descriptive documents produced by the `moai-workflow-project` skill, not SPEC artifacts (no YAML frontmatter, no `REQ-XXX`/`AC-XXX`, no `plan.md`/`research.md`). Re-verified:

```
grep -c "SPEC-" .moai/project/{product,structure,tech}.md   → 0, 0, 0
head -5 on all three files                                  → plain "# Title" headers, no frontmatter
```

MP-1, MP-2, MP-3, and MP-7 remain N/A by document-type mismatch. Per the note attached to this invocation, iteration 2's FAIL was driven by fast-moving repo state (a local `imoogi-toolchain fetch` had run mid-session). This iteration re-verifies **every** factual claim fresh against the live repository right now — treating neither iteration 2's findings nor iteration 3's presumed fixes as still current until independently checked.

## Must-Pass Results
- [N/A] MP-1 REQ number consistency: no `REQ-XXX` entities exist in this document type
- [N/A] MP-2 EARS/GEARS format compliance: no `REQ-XXX` entities exist to check
- [N/A] MP-3 YAML frontmatter validity: none of the three files carry YAML frontmatter (re-verified, `head -5` shows bare `# Title`)
- [N/A] MP-4 Section 22 language neutrality: tech.md is scoped to a single project's stack (Emacs Lisp + Go), not a multi-language tooling template
- [N/A] MP-5 D7 cross-SPEC reconciliation: `grep -c "SPEC-"` returns 0 for all three files (re-verified)
- [PASS] MP-6 D8 cross-platform discipline: `grep -in "syscall" .moai/project/*.md` returns no matches (re-verified) — D8 auto-PASS per D8-4
- [N/A] MP-7 clarification gate: neither `plan.md` nor `research.md` exists for this document type

**No must-pass criterion forces FAIL.** As in iterations 1-2, the verdict is driven by the category-score findings (Group 6 consistency + factual-accuracy verification), documented next.

## Regression Check (Iteration 3)

Every defect carried into this iteration (D1-D8 across iterations 1-2) was independently re-verified against the live repository, not assumed resolved:

- **D1-D5** (iteration 1: build-status, test-file, package-caveat/count) — **REMAIN RESOLVED**. `go build ./...` exits 0; `go list ./...` enumerates exactly 10 packages; `internal/fetch/fetch_test.go` exists; no "Known Issues / currently fails" language present in tech.md.
- **D6** (`stale-toolchain-artifact-nonexistence-claim`) — **REMAINS RESOLVED, RE-VERIFIED FRESH**. Live check this iteration:
  ```
  $ find vendor -maxdepth 2 -type d | grep toolchains
  vendor/toolchains
  vendor/toolchains/cli  vendor/toolchains/go  vendor/toolchains/licenses
  vendor/toolchains/node vendor/toolchains/typescript
  $ ls toolchains.lock.json                    → exists
  $ git status --short vendor/toolchains toolchains.lock.json
  ?? toolchains.lock.json
  ?? vendor/toolchains/
  ```
  structure.md (Directory Tree comment, Directory Purposes table, Key File Locations) and tech.md § Go toolchain vendoring all currently state the accurate three-part fact: `imoogi-toolchain fetch` HAS been run in this checkout, `vendor/toolchains/{go,licenses,typescript,cli,node}/` and `toolchains.lock.json` exist and are populated, and both are **untracked/uncommitted** in git. This matches the live `git status --short` output exactly.
- **D7** (`unverifiable-node-version-claim`) — **REMAINS RESOLVED, RE-VERIFIED FRESH**. tech.md:L37 states "Node v24.19.0". Live `toolchains.lock.json` (`"upstream_version": "v24.19.0"` for the `node` component), `toolchains.json` (`"node_version": "v24.19.0"`), and `vendor/toolchains/node/v24.19.0/` (the only version directory present) all agree. No `v22.23.2` reference remains in tech.md.
- **D8** (`fabricated-typescript-version-claim`) — **REMAINS RESOLVED, RE-VERIFIED FRESH**. tech.md:L38 states "TypeScript 6.0.3" (no longer paired with a fabricated "7.0.2"). Live `toolchains.lock.json` (`"upstream_version": "6.0.3"` for the `typescript` component) and `vendor/toolchains/typescript/typescript/6.0.3/` both confirm 6.0.3 is the only version present. `typescript-language-server 6.0.0` (tech.md:L39) also matches the lockfile's `typescript-language-server` component exactly. `gopls v0.23.0` (tech.md:L40) matches the lockfile's `gopls` component exactly.

**All 8 defects from iterations 1-2 are cleanly resolved on fresh, independent re-verification.** The toolchain-vendoring subsystem — the multi-iteration stagnation-risk cluster flagged at the end of the iteration-2 report — is fully accurate as of this audit and shows no sign of the "claimed-exists-when-absent / claimed-absent-when-exists" oscillation pattern recurring a third time. This breaks the stagnation pattern.

However, broadening verification beyond the previously-flagged toolchain-vendoring cluster (per this iteration's instruction to check fresh rather than assume anything is settled) surfaced **one new defect** not examined in either prior iteration — see D9 below.

## Category Scores (0.0-1.0, adapted rubric)
| Dimension | Score | Rubric Band | Evidence |
|-----------|-------|-------------|----------|
| Clarity | 0.75 | Minor ambiguity in one or two areas a reasonable engineer would resolve consistently | Prose remains unambiguous throughout; unchanged from prior iterations |
| Completeness | 0.75 | One non-critical section missing or sparse; all required product/structure/tech sections otherwise present | All required sections present and well-organized; unchanged from prior iterations |
| Testability (repurposed: Factual Verifiability) | 0.75 | One area of inaccuracy against otherwise fully-verified claims | Every toolchain-vendoring claim, every version number, every package/module count, every test-file enumeration, every CI-workflow claim, and every Go-package existence/dispatch claim independently re-verified accurate this iteration (see verification log below) — docked only for D9, a single isolated mischaracterization of one Go package's actual responsibility |
| Traceability | N/A | — | No `REQ-XXX`/`AC-XXX` entities exist; substituted by Group 6 Consistency below |

### Full verification log this iteration (beyond the D1-D8 regression check)

All PASS unless noted:
- `go build ./...` → exit 0; `go list ./...` → 10 packages, matches structure.md:L161
- `packages.lock` header states "총 85 패키지"; `ls -d vendor/elpa/*/ | grep -v archives | wc -l` → 85, matches product/tech.md's "85 resolved package/version pairs"
- `modules/*.el` (numbered) → exactly 22 files, matches product.md/structure.md's "22 numbered feature modules"
- `modules/lsp/*.el` → bash, clojure, go, java, javascript, kotlin, python, rust, typescript (9 files), matches structure.md's Directory Tree listing exactly
- `go.mod` → `module github.com/karohani/imoogi-emacs`, `go 1.26`, no `go.sum` (zero third-party deps), matches structure.md/tech.md
- `.github/workflows/` → `label-sync.yml` only, matches structure.md/tech.md's "no CI test workflow" claim
- Go test files (`find internal cmd -name "*_test.go"`) → 8 files, matches tech.md § Testing → Go side enumeration exactly
- Emacs test files (`tests/*.el`) → matches structure.md's Directory Tree and tech.md § Testing → Emacs side enumeration exactly
- `tests/run.sh` (30 lines) → confirmed 3-phase structure (check-parens syntax validation → offline-boot smoke test with `package-archives nil` → ERT via `tests/run.el`), matches tech.md § Testing → Emacs side exactly
- Spot-checked ~18 package version numbers in tech.md's dependency table (vertico, orderless, marginalia, embark, consult, corfu, cape, perspective, hydra, ace-window, magit, treemacs, obsidian, doom-themes, doom-modeline, nerd-icons, ghostel, compile-angel) against `packages.lock` → all match exactly
- `internal/config` (`TargetOS`/`TargetArch` constants), `internal/artifact` (verification/extraction functions), `internal/cli` (fetch/setup dispatch), `internal/lang/golang` + `internal/lang/typescript` (Materialize functions for gopls/tsc/typescript-language-server) → all match their structure.md descriptions
- No `Makefile` anywhere in the repository root, matches tech.md's explicit claim
- `AGENTS.md:L5` "가장 중요한 제약" quote, referenced by tech.md, confirmed to exist verbatim
- `which-key` confirmed absent from `packages.el` with the exact rationale comment ("Emacs 30 built-in"), matches product.md:L21/tech.md:L7

## Group 6 — Consistency (substituting for Traceability)
- CN-1 (no two requirements contradict): **FAIL** — see D9; structure.md's description of `internal/activation`'s responsibility contradicts the package's actual source code
- CN-2 (exclusions don't conflict with included requirements): PASS — Non-Goals (product.md:L45-47) and Scope Boundaries (tech.md:L117-121) remain mutually consistent (darwin/arm64-only, personal-use-only, stated identically in both docs)
- CN-3 (priority/labels consistent with stated scope): N/A — no priority/label fields exist in this document type

## Defects Found (structured defect-list)

D9. `mischaracterized-package-responsibility` — `structure.md:L86` (Directory Tree) and `structure.md:L159` (§ Go Package Organization, listed as one of the "focused, single-responsibility sub-packages") — structure.md's Directory Tree comments `internal/activation/` as: *"Atomic activation of .local/bin (relative symlink)"*. Verified directly against the source code:

   ```
   $ cat internal/activation/bootstrap.go
   package activation
   ...
   type BootstrapSpec struct { Version string; Output string }
   func (s BootstrapSpec) Env() []string { ... GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 ... }
   func (s BootstrapSpec) Args() []string { ... "build" "-trimpath" ... "./cmd/imoogi-toolchain" ... }
   func (s BootstrapSpec) Provenance() string { ... }

   $ grep -rln '"github.com/karohani/imoogi-emacs/internal/activation"' internal/*/*.go cmd/*/*.go
   internal/fetch/fetch.go     ← the ONLY importer

   $ grep -n "activation\." internal/fetch/fetch.go
   654:  spec := activation.BootstrapSpec{Version: cliVersion, Output: stagePath}
   656:  command := activation.BootstrapSpec{Version: cliVersion, Output: filepath.ToSlash(rel)}.Provenance()
   ```

   `internal/activation` contains only a `BootstrapSpec` type (build environment, `go build` args, and a provenance string for cross-compiling the `imoogi-toolchain` CLI binary itself, darwin/arm64, for vendoring under `vendor/toolchains/cli/...`), and it is imported exclusively by `internal/fetch/fetch.go` — the online-only artifact-vendoring path. It contains no symlink logic whatsoever.

   The actual "atomic activation of `.local/bin` via a relative symlink" logic — which structure.md's own comment describes correctly as a real mechanism in the codebase — lives in `internal/setup/setup.go`'s `activate()` function:

   ```
   $ sed -n '424,434p' internal/setup/setup.go
   func activate(localRoot, bundle string) error {
       linkTarget := filepath.ToSlash(filepath.Join("toolchains", bundle, "bin"))
       tempLink := filepath.Join(localRoot, ".bin-"+bundle+".tmp")
       _ = os.Remove(tempLink)
       if err := os.Symlink(linkTarget, tempLink); err != nil { ... }
       if err := os.Rename(tempLink, filepath.Join(localRoot, "bin")); err != nil { ... }
       return nil
   }
   ```

   This confirms `internal/setup` — already separately and correctly described in structure.md:L84 as "Offline-only: verifies + stages + activates vendored artifacts" — is where the symlink-activation behavior actually lives. `internal/activation` is a differently-scoped package (CLI-self-bootstrap-build spec) whose name coincidentally overlaps with the word "activation" used correctly elsewhere in the same document for a different package. A reader following structure.md's Directory Tree to locate the `.local/bin` symlink-activation code would be pointed at the wrong package.

   — Severity: **major** — Class: **blocking** — Required fix: correct structure.md:L86 to describe `internal/activation`'s actual responsibility (e.g. "Cross-compile bootstrap spec (GOOS/GOARCH/build args/provenance) used by `internal/fetch` to build and vendor the `imoogi-toolchain` CLI binary itself") and, if a comment is desired at the tree location, cross-reference `internal/setup`'s `activate()` function as the actual home of the `.local/bin` relative-symlink activation logic already correctly described at structure.md:L84.

No other new defects found this iteration. Structural completeness (Group 2 adapted), Non-Goals/Scope-Boundaries consistency (CN-2), every toolchain-vendoring fact (D6-D8 regression), every count/version/file-enumeration claim spot-checked this iteration, and cross-platform discipline (D8/MP-6) all pass with cited evidence above.

## Recommendation

FAIL, materially improved again (0.65 → 0.80). This is genuinely close to a clean PASS: **all 8 defects carried from iterations 1-2 are confirmed resolved on fresh, independent re-verification** (not assumed — each was re-checked against the live repository state as of this audit, including the toolchain-vendoring cluster that had oscillated between iterations 1 and 2 due to concurrent `imoogi-toolchain fetch` activity). That stagnation risk did not recur a third time.

The single remaining defect (D9) is narrow and isolated: a one-line mischaracterization of what `internal/activation` does, discovered only because this iteration's instruction was to verify every factual claim fresh rather than assume settled areas are still accurate. It does not touch the toolchain-vendoring cluster, build status, test enumeration, version numbers, or package/module counts — all of which remain accurate.

Because this is iteration 3 of 3 (the Retry Loop Contract's hard ceiling), per the LEAN Workflow Additions the orchestrator should escalate to the user with the three standard options (PASS-with-debt / scope-reduction / explicit override to iterate further) rather than auto-triggering a fourth pass. Given the small, single-line scope of the one remaining defect, PASS-with-debt (documenting D9 and fixing it in a follow-up edit) is a reasonable option to present alongside a one-line direct fix.

Numbered fix instruction for the document author (likely `manager-docs` / `moai project` regeneration), if a fourth pass is authorized or a direct fix is made instead:

1. Fix structure.md:L86 (D9) to correctly describe `internal/activation` as the CLI's own cross-compile bootstrap spec (consumed only by `internal/fetch`), not as the `.local/bin` symlink-activation logic — which structure.md already correctly attributes to `internal/setup` at L84.
2. Optionally add a one-line cross-reference at L84's `internal/setup` bullet pointing to the `activate()` function name, so the two "activation"-adjacent packages are unambiguously distinguished for a reader.

Verdict: FAIL
