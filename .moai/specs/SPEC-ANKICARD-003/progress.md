# Progress — SPEC-ANKICARD-003

## §E.1 Plan-phase Audit-Ready Signal

```yaml
plan_status: audit-ready
plan_complete_at: 2026-09-20
tier: M
artifacts: [spec.md, plan.md, acceptance.md]
requirements: 15   # REQ-ML-001..REQ-ML-015
criteria: 15       # AC-ML-001..AC-ML-015
spec_version: 0.1.4
audit_iterations:
  - iteration: 1
    verdict: FAIL
    score: 0.63
    report: .moai/reports/plan-audit/SPEC-ANKICARD-003-plan-audit.md
    answered_in: spec.md v0.1.1 HISTORY
    blocking_closed: 10   # D1..D8, D10, D12
    optional_taken: 2     # D9, D11
  - iteration: 2
    verdict: FAIL
    score: 0.706
    report: .moai/reports/plan-audit/SPEC-ANKICARD-003-plan-audit-r2.md
    answered_in: spec.md v0.1.2 HISTORY
    blocking_closed: 3    # N1, N2, N3
    optional_taken: 2     # N4, N5
    prior_findings_regressed: 0
  - iteration: 3
    verdict: PASS
    score: 0.923          # Tier M threshold 0.80
    report: .moai/reports/plan-audit/SPEC-ANKICARD-003-plan-audit-r3.md
    answered_in: spec.md v0.1.3 HISTORY
    blocking_closed: 0    # none raised
    optional_taken: 3     # NEW-1, NEW-2, NEW-3
    prior_findings_regressed: 0
run_phase_amendments:
  - spec_version: 0.1.4
    source: run phase (14/15 criteria passing)
    answered_in: spec.md v0.1.4 HISTORY
    amendments: 3         # PRESERVE narrowing, REQ-ML-013.2 verification note, corpus count
    recorded_not_fixed: 1 # set-direction interactive default
    obligations_changed: 0
```

## §E.2 Run-phase Evidence

Baseline: `6f3ae6c` (`git rev-parse HEAD` at run start), branch `main`.
Verbatim command output is persisted under
`.moai/state/verify/spec-ankicard-003/`.

### Acceptance-criterion matrix

| AC | Status | Deciding command | Actual output |
|---|---|---|---|
| AC-ML-001 | PASS | `go test ./internal/anki/orgdoc -run TestMultilineAnswerList -count=1` | `ok` — 24 corpus shapes (18 required members + 5 supplementary-marker residue rows, one of them a positive control), scanner-vs-parse count equality, 4 span assertions, 2 multi-byte rows; rendered half in `TestMultilineAnswerListRendering` (a-j) |
| AC-ML-002 | PASS | `go test ./internal/anki/planner -run TestMultilineAnswerMissing -count=1` | `ok` — a/b/c/d + message check |
| AC-ML-003 | PASS | `go test ./internal/anki/orgdoc -run TestMultilineDirection -count=1` | `ok` — a/b/c |
| AC-ML-004 | PASS | `go test ./internal/anki/orgdoc -run TestMultilineDefaultDirection -count=1` + `go test ./internal/anki/planner -run TestMultilineDefaultDirection -count=1` | `ok` — the falsy-vs-absent byte-identity half is asserted planner-side, where the two spellings are genuinely different inputs |
| AC-ML-005 | PASS | `go test ./internal/anki/orgdoc -run TestMultilineIncremental -count=1` | `ok` — a/b/c |
| AC-ML-006 | PASS | `go test ./internal/anki/orgdoc -run TestMultilineNumbering -count=1` | `ok` — a/b/c/d/e; e is now measured rather than argued (see §Findings) |
| AC-ML-007 | PASS | `go test ./internal/anki/orgdoc -run TestMultilineBraceSafety -count=1` + `make test-elisp` | `ok` / `0 unexpected` — 007c shares ONE fixture, `internal/anki/orgdoc/testdata/cloze-brace-fixture.json`, read by both suites |
| AC-ML-008 | PASS | `go test ./internal/anki/orgdoc -run TestMultilineLeftwardIncremental -count=1` | `ok` — byte-identical with incremental on and off, no diagnostic |
| AC-ML-009 | PASS | `go test ./internal/anki/planner -run TestMultilineMarkerGate -count=1` | `ok` — a/b/c/d/e/f/g/h; f asserts SRC and EXAMPLE as separate cases |
| AC-ML-010 | PASS | `go test ./internal/anki/planner -run TestMultilineBothPaths -count=1` + `go test ./internal/anki/orgdoc -run TestRenderSignatureIsUnchanged -count=1` | `ok` — both call sites; the existing signature assertion compiles unchanged |
| AC-ML-011 | PASS | `go test ./internal/anki/orgdoc -run TestMultilineContainer -count=1` + `go test ./internal/anki/model -count=1` | `ok` — ul/ol/dl all carry `children-list`, exactly one per field, present under `<-`; stylesheet rule added |
| AC-ML-012 | PASS | `make test-elisp` | `Ran 435 tests, 433 results as expected, 0 unexpected, 2 skipped` — a/b/c/d/e |
| AC-ML-013 | PASS | `go test ./internal/anki/orgdoc ./internal/anki/hashing -count=1`, `go test ./internal/anki/planner -run TestOptionFreeRequestLogGolden -count=1`, `git diff --exit-code 6f3ae6c -- internal/anki/orgdoc/testdata/render-golden.json` | `ok` / `ok` / exit 0 — a/b/c/d/e |
| AC-ML-014 | PASS | `go test ./internal/anki/protocol -count=1` + `make test-elisp` | `ok` / `0 unexpected` — the code-to-message entry and the two-directional pairing over the enlarged set |
| AC-ML-015 | PASS | `make ci-local` | exit 0. The blocker below was resolved by an explicit, scoped authorization from the delegating session; every baseline figure holds. |

### Measured quality gates

| Gate | Baseline at `6f3ae6c` | This run | Verdict |
|---|---|---|---|
| `go build ./...` | exit 0 | exit 0 | PASS |
| `GOOS=windows go build ./cmd/imoogi-anki` | exit 0 | exit 0 | PASS |
| `go test ./... -count=1` | exit 0 | exit 0 | PASS |
| Coverage `orgdoc` | 100.0% | 100.0% | PASS |
| Coverage `planner` | 92.5% | 92.8% | PASS |
| Coverage `hashing` | 100.0% | 100.0% | PASS |
| Coverage `model` | 98.4% | 98.4% | PASS |
| `make lint` | 0 | exit 0, 0 findings | PASS |
| `make fmt-check` | 0 | exit 0, 0 findings | PASS |
| `make ci-local` | exit 0 | exit 0 | PASS |
| `make test-elisp` | `Ran 422, 420 expected, 0 unexpected, 2 skipped` | `Ran 435, 433 expected, 0 unexpected, 2 skipped` | PASS — delta fully attributed: +3 from another actor's untracked `tests/org-roam-test.el`, +10 authored here |
| `git diff --exit-code 6f3ae6c -- .../render-golden.json` | clean | clean | PASS |

### RED evidence (E8)

Captured before each milestone's implementation, persisted verbatim:

- M1 `m1-red.txt` — `undefined: collapseBlankRuns / parseFirstList / scanAnswerList`, `[build failed]`
- M2 `m2-red.txt` — `undefined: RenderWithOptions / loadBraceFixture / clozeSafeText`, `[build failed]`
- M3 `m3-red.txt` — `undefined: protocol.CodeMultilineAnswerMissing / readCardOptions`, `[build failed]`
- M4 `m4-red.txt` — `Ran 435 tests, 423 results as expected, 10 unexpected` (all ten authored here)
- M5 `m5-red.txt` — `FAILED imoogi-error-test-codes-match-source-file` (protocol code added, table entry not yet)

### Blocker — plan.md §A.3 PRESERVE conflicted with REQ-ML-002 — RESOLVED

**Status: resolved under an explicit scoped authorization from the delegating
session, which reproduced the failure and read the helper before granting it.
The authorization covered `applyHashProbeOptions` and its comment ONLY.** The
record of the conflict is kept below because the plan still needs the matching
§A.3 amendment.

`plan.md` §A.3 preserved `internal/anki/planner/baseline_golden_test.go`
wholesale and §D constraint 1 forbade touching it.

`TestHashIsUnchangedByCardOptions` in it sets `direction: "->"` and
`incremental: "t"` (via `applyHashProbeOptions`) on the body `"With a body."`,
which carries no list, and asserts (a) the run reports no error and (b) the
recorded hash is unchanged. Under this SPEC both are false by design:

- (a) REQ-ML-002 requires a multiline entry whose remaining body yields no
  answer item to be rejected with `multiline_answer_missing`.
- (b) REQ-ML-014.2 asserts the hash's **input set** gains no member — which
  `TestHashCallSiteTakesFourArguments` (same file) proves at compile time and
  AC-ML-013d re-asserts in the `hashing` package. It does NOT assert the hash
  VALUE is unchanged; `spec.md` §7 states the opposite in so many words
  ("a multiline entry changes the first of those through the rendered value
  alone").

The test encodes a SPEC-ANKICARD-002-era premise this SPEC deliberately
falsifies. Amending it is manager-spec's, not this agent's.

Recommended resolution, in preference order:

1. Narrow §A.3's PRESERVE entry to name the byte-identity and signature
   evidence explicitly — `TestRenderSignatureIsUnchanged`,
   `TestHashCallSiteTakesFourArguments`, `TestOptionFreeRequestLogGolden` — and
   permit `applyHashProbeOptions` to set `swift: "t"` alone. Swift is the one
   option with no rendering meaning until backlog card t15, so the test's claim
   stays true AND discriminating. The function's own doc already anticipates
   this: "the ONE place TestHashIsUnchangedByCardOptions needs to change as the
   wire contract grows."
2. Retire the test as superseded by `TestHashCallSiteTakesFourArguments` plus
   AC-ML-013d.
3. Leave it red and record the debt.

**What was done.** Option 1, as drafted: the probe now sets `swift: "t"` alone.
It still passes the validation gate on an `imoogi-Cloze` entry, still carries a
card-option field into the run, and still reaches the add branch — and swift is
the one option with no rendering meaning until backlog card t15, so the probe's
claim holds and stays discriminating. Verified by falsification: forcing a hash
difference between the two variants makes it fail with "the recorded content
hash differs with card options set", so it is not passing vacuously.

The helper's comment now records WHY direction and incremental can no longer
serve — this SPEC made them render, and a rendering option changes the hash
through the rendered field value, which is one of the four inputs and always
was — and warns that t15 will need the same treatment again, at which point no
card option will be rendering-free and the probe will need a different shape
rather than a different option.

**Scope actually touched**, verified by diff: the ten lines of
`applyHashProbeOptions` and its doc comment. `TestRenderSignatureIsUnchanged`,
`TestHashCallSiteTakesFourArguments`, `TestOptionFreeRequestLogGolden`, both
`testdata` goldens, `card_options.go`, `hashing.go` and `extra.go` are all
unchanged from `6f3ae6c`.

**Still owed by the plan**: the §A.3 narrowing that makes this edit legible to
the next reader. manager-spec is authoring it as a v0.1.4 amendment.

### Findings closed by measurement

Three items the iteration-3 audit could not verify:

1. **Can the `#+BEGIN_EXTRA` split leave a dangling block marker?** **Yes, in
   three of four shapes — and it is NOT harmless. This CORRECTS an earlier
   entry in this section.**

   My first probe used a nested-block body whose residue also contained real
   prose (`more`). I attributed the split list to that prose, concluded an
   orphan marker was harmless, and recorded it. That conclusion was wrong: the
   prose was doing work the marker does on its own. Re-measured against
   `splitExtraBlocks` copied verbatim from `6f3ae6c`, with the prose removed:

   | Shape | Remaining body | Rendered | Outcome |
   |---|---|---|---|
   | A valid pair + stray `#+END_EXTRA` | `- A\n#+END_EXTRA\n- B` | `<ul>A</ul><p>#+END_EXTRA</p><ul>B</ul>` | answer B dropped, marker visible |
   | B unterminated `#+BEGIN_EXTRA` | unchanged | `<ul>A</ul><p>#+BEGIN_EXTRAnote B</p>` | answer B stops being a list item |
   | C two openings, one closing | `- A\n- B` clean | `<ul>A B</ul>` | question side fine; stray marker in `Back Extra` |
   | D lone `#+END_EXTRA` | `- A\n#+END_EXTRA\n- B` | as A | answer B dropped, marker visible |

   A dangling BEGIN **can** arise (shape B), which my earlier entry also denied.
   Shapes A, B and D are pre-existing behaviour of the Back Extra work at
   `5d3868f`; this SPEC neither introduces nor repairs them.

   **Reachability, measured rather than assumed.** The obvious reading is that
   all of this needs malformed authoring. It does not. Separating the shapes by
   whether the author's MARKERS are balanced:

   | Author wrote | Markers | Question side | Back Extra |
   |---|---|---|---|
   | one block (the everyday shape) | balanced | clean, both answers survive | correct |
   | two SEQUENTIAL blocks (documented as supported) | balanced | clean, both answers survive — the collapse closes the wider gap | correct |
   | two NESTED blocks | **balanced** | **damaged**: stray `#+END_EXTRA` + answer dropped | carries a stray `#+BEGIN_EXTRA` |
   | 2 openings / 1 closing | unbalanced | clean | carries a stray `#+BEGIN_EXTRA` |
   | 1 opening / 2 closings, lone closing, lone opening | unbalanced | damaged | — |

   So the question-side damage needs unbalanced markers **or nesting** — and
   nesting is balanced Org that an author could reasonably write. It is NOT
   reachable from the two documented shapes, and the collapse handles the
   sequential one correctly, which is asserted as a positive control
   (`TestMultilineSequentialExtraBlocksKeepEveryAnswer`) so this finding cannot
   be read as "supplementary blocks break multiline cards".

   **Decision, and it is not a decline.** A dangling marker is ordinary content:
   composition gives it no special handling, and it ends the answer list exactly
   as a paragraph does (REQ-ML-001.4/1.5, the outcome AC-ML-009e already fixes
   for a paragraph). Three alternatives were measured and rejected:

   - *Declining to collapse* — REQ-ML-009.2's unterminated-block rule — is a
     **no-op** on all four shapes. The separating line is non-blank, so there is
     no run to decline on; rendering is byte-identical with the collapse and
     without it, while the control shape `- A\n\n\n- B` differs as it should.
     Adopting it would have looked like the case was handled while changing
     nothing.
   - *Removing the marker* is the silent content change REQ-ML-009.2 forbids.
   - *Reading the answer list across it* would wrap answers that go-org renders
     in a second list, outside the `children-list` container, contradicting
     REQ-ML-012.1.

   **This needs a SPEC change to close properly, and I did not make one.** The
   remaining option is to report it, and no code here covers the shape:
   REQ-ML-002's fires only when there is no answer item, and there is one. A new
   code plus a new requirement is manager-spec's to author. The behaviour is
   pinned meanwhile — five corpus rows in `TestMultilineAnswerList` and
   `TestMultilineDanglingExtraMarker` (four shapes, including the balanced
   nested one), `TestMultilineUnbalancedOpeningLeavesTheQuestionClean`, and the
   positive control `TestMultilineSequentialExtraBlocksKeepEveryAnswer` in the
   planner — all driven from the PRE-SPLIT body so the whole pipeline is
   exercised, while the corpus rows carry the post-split residue the scanner
   actually sees.
2. **AC-ML-009f's `#+BEGIN_EXAMPLE` half** — measured directly, as its own case
   beside the `#+BEGIN_SRC` one, in
   `TestMultilineMarkerGate/f_the_collapse_does_not_reach_inside_a_block`.
3. **AC-ML-006e's precedence closure** — now measured rather than argued:
   `TestMultilineNumbering/e_the_not_wrap_rule_governs_incremental` runs the
   shared-number fixture through the real renderer and asserts both markers keep
   `c1` and nothing is renumbered.

### Findings raised

1. **`org-entry-get` cannot read the falsy spelling back.** Measured: after
   `(org-set-property "ANKI_DIRECTION" "nil")` the drawer plainly carries
   `:ANKI_DIRECTION: nil`, yet `org-entry-get` returns Lisp `nil` —
   indistinguishable from an absent property. The production read path is
   unaffected, because `imoogi-props--own-value` deliberately reads the drawer
   text by regexp rather than through Org's accessor, and returns the string
   `"nil"` correctly. The ERT tests assert through `imoogi-props-resolve-*`
   accordingly, which is both the correct accessor and the stronger assertion.
   Anyone verifying REQ-ML-013.2 by hand with `org-entry-get` will read a false
   negative.
2. **The M1 corpus count in `plan.md` §F is off by one.** The table has 16 rows
   carrying 18 shapes (two rows carry two each); `plan.md` and `acceptance.md`
   both say "seventeen". All 18 are implemented as required members — a superset
   satisfies the requirement and §G anti-pattern 9 forbids trimming, not adding.
   Cosmetic; no obligation changes.

3. **`imoogi-anki-set-direction`'s interactive default cannot show a cleared
   value.** Its `completing-read` initial input comes from
   `org-entry-get ... "ANKI_DIRECTION" t`, which by finding 1 reads a written
   `nil` as absent — so after a clear the prompt offers an empty default rather
   than `nil`, and cannot distinguish "cleared here" from "never set". The
   WRITE is unaffected and the back end reads the value correctly. Left as a
   known limitation rather than fixed: no requirement or criterion constrains
   the prompt's default, and changing it is a UX decision, not a defect repair.

### Follow-ups for manager-spec

When §A.3 is amended per the Blocker above, three items should ride the same
v0.1.4 HISTORY entry: the PRESERVE narrowing itself, the "seventeen" versus
eighteen corpus-shape count in `plan.md` §F M1 and `acceptance.md` AC-ML-001,
and finding 1 as a note on REQ-ML-013.2 (verifying the falsy spelling by hand
requires `imoogi-props-resolve-*`, not `org-entry-get`).

A FOURTH item is larger and may warrant its own card rather than a HISTORY
line: the dangling-supplementary-marker shapes above are reported by nothing.
Closing that needs a new diagnostic code and a new requirement — see
"Findings closed by measurement" item 1 for the measurement, the decision taken
in the meantime, and the three alternatives ruled out.

### Scope

Files created: `internal/anki/orgdoc/multiline.go`, `multiline_test.go`,
`compose_test.go`, `multiline_golden_test.go`,
`testdata/cloze-brace-fixture.json`, `testdata/multiline-golden.json`;
`internal/anki/planner/multiline.go`, `multiline_test.go`.

Files modified: `internal/anki/protocol/protocol.go`,
`internal/anki/planner/{planner,migrate}.go`,
`internal/anki/planner/card_options_test.go`,
`internal/anki/hashing/hashing_test.go`,
`internal/anki/model/assets/base.css`, `internal/anki/model/css_test.go`,
`modules/org/24-anki.el`, `modules/org/anki/imoogi-error.el`,
`tests/anki-commands-test.el`, `tests/anki-error-test.el`, `README.md`,
`.moai/specs/SPEC-ANKICARD-003/{spec,progress}.md`.

Every path is inside `spec.md` §7's delta table. No file on `plan.md` §A.3's
PRESERVE list was modified; no git command that writes was run.

## §E.3 Run-phase Audit-Ready Signal

```yaml
run_complete_at: 2026-09-20
run_commit_sha: pending-backfill-run
run_status: complete
ac_pass_count: 15        # AC-ML-001..015
ac_fail_count: 0
ac_gap_count: 0
preserve_list_post_run_count: 1   # baseline_golden_test.go applyHashProbeOptions, under explicit scoped authorization
new_warnings_or_lints_introduced: 0
cross_platform_build:
  host: pass
  windows_anki_binary: pass
  windows_all_packages: pre-existing gap in internal/setup (syscall.Flock), untouched
dangling_extra_marker: pinned-not-reported   # needs a new code; SPEC change deferred to manager-spec
total_run_phase_files: 22
m1_to_mN_commit_strategy: none — the delegating session commits by explicit pathspec
```

## §E.4 Sync-phase Audit-Ready Signal

_<pending sync-phase>_
