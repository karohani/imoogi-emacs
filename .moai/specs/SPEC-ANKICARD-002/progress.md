# Progress — SPEC-ANKICARD-002

Phase record for the card-option transport and validation SPEC. Sections §E.2
through §E.4 are placeholders emitted at plan time and owned by the run and
sync phases respectively; they are not populated here.

## §E.1 Plan-phase Audit-Ready Signal

| Item | State |
|---|---|
| SPEC ID regex self-check | Executed as Bash; output `PASS` |
| SPEC ID uniqueness | Confirmed — `.moai/specs/` holds only `SPEC-ANKICARD-001` and `SPEC-TRANSIENT-001` |
| Frontmatter | 12 canonical fields present, plus `tier: M` and `depends_on` |
| Tier | M — justified in `plan.md` § I |
| Artifact set | `spec.md`, `plan.md`, `acceptance.md` (Tier M), plus this file |
| Requirements | 13, contiguous `REQ-OPT-001`..`REQ-OPT-013`, GEARS, one trigger each |
| Acceptance criteria | 14, contiguous `AC-OPT-001`..`AC-OPT-014`, Given-When-Then, each naming its deciding command |
| Exclusions | `spec.md` § 5 — seven `### Out of Scope —` topics |
| Implementation detail in `spec.md` | None — no function, type, or file-internal name; homes are named in `plan.md` § D only |

### Plan audit — iteration 1

| Item | State |
|---|---|
| Verdict | PASS, 0.91 harmonic mean against the Tier M threshold of 0.80 |
| Must-pass | MP-1, MP-2, MP-3, MP-5, MP-6, MP-7 pass; MP-4 not applicable |
| Blocking findings | 4 — D1, D2, D3, D4 — all closed at `spec.md` v0.1.1 |
| Optional findings | 5 — D5, D6, D7, D8 taken; D9 addressed by the alternative fix the audit itself offers |
| Report | `.moai/reports/plan-audit/SPEC-ANKICARD-002-plan-audit.md` |

One audit premise is disputed and recorded rather than silently accepted. D2
stated that the design record's § 7 names t13 nowhere and so cannot authorize
it. That section's ordering note in fact reads that the operator, knowing
SPEC-ANKICARD-001 was `in-progress`, directed cards **t11 through t15** be
worked in order — a range covering this card. The finding's *outcome* was
correct and is applied; its premise was not. The cause was on this side: the
first draft of `plan.md` § A paraphrased that note as covering t11 and t12
only, and the auditor reasoned from the paraphrase.

Open items carried into the plan gate, both flagged reversible with their
collapse cost stated:

1. `plan.md` DD-2 — three diagnostic codes where the backlog card mandated
   one. Upheld independently at audit; collapse remains mechanical.
2. `plan.md` DD-5 — `ANKI_DIRECTION` accepting the falsy spelling `nil`, an
   expansion of a value set the design record enumerates exhaustively.

Status: `draft`. Plan-phase artifacts authored; plan audit run and answered.
Awaiting Implementation Kickoff Approval.

## §E.2 Run-phase Evidence

Base commit for every baseline below: `5d3868f` (the post-t12 tree). Every
figure was measured in this run, against this tree — none is carried over from
the design record or from the delegation prompt.

### M1 — Baseline capture (no production change)

Recorded BEFORE any production edit, which is what keeps AC-OPT-012 from
degenerating into a tautology (`plan.md` §H anti-pattern 3, §E item 1).

| Baseline | Artifact | Recorded by |
|---|---|---|
| Render golden corpus — basic, cloze, cloze+extra, math, empty body | `internal/anki/orgdoc/testdata/render-golden.json` | `UPDATE_ORGDOC_GOLDEN=1 go test ./internal/anki/orgdoc -run TestRenderGoldenCorpus` |
| Option-free request log — a no-op fixture and a mixed add/update fixture | `internal/anki/planner/testdata/request-log-golden.json` | `UPDATE_PLANNER_GOLDEN=1 go test ./internal/anki/planner -run TestOptionFreeRequestLogGolden` |

Recording is gated behind an environment variable rather than an "update if
missing" fallback: a golden that regenerates itself the moment it disappears
would let the run that broke it also re-bless it.

Coverage measured at `5d3868f` with `go test ./internal/anki/... ./cmd/imoogi-anki/... -count=1 -cover`:

| Package | Coverage at base |
|---|---|
| `internal/anki/protocol` | `[no statements]` |
| `internal/anki/orgdoc` | 100.0% |
| `internal/anki/planner` | 91.6% |
| `internal/anki/hashing` | 100.0% |
| `internal/anki/model` | 98.4% |
| `internal/anki/media` | 91.3% |
| `internal/anki/ankiconnect` | 86.2% |
| `internal/anki/registry` | 87.8% |
| `cmd/imoogi-anki` | 84.2% |

`protocol` reports `[no statements]`, not a percentage: the package is
declarations only. Its floor is therefore "still no statements" — this SPEC
adds three struct fields and three constants, none of which is a statement.

Elisp baseline at `5d3868f`: `Ran 407 tests, 405 results as expected, 0 unexpected, 2 skipped`.
The two skips are `imoogi-org-calendar-date-prompt-with-persisted-zoom-gui` and
`imoogi-org-calendar-responsive-gui` — both GUI-gated. Neither is a pairing
test, so AC-OPT-014's criteria are decidable rather than skipped.


### M2 — Wire contract

RED (verbatim, before implementation):

```
internal/anki/protocol/card_options_test.go:60:3: unknown field Incremental in struct literal of type protocol.Entry
internal/anki/protocol/card_options_test.go:105:11: entry.Direction undefined (type protocol.Entry has no field or method Direction)
internal/anki/protocol/card_options_test.go:183:13: undefined: protocol.CodeCardOptionInvalid
FAIL	github.com/karohani/imoogi-emacs/internal/anki/protocol [build failed]
```

GREEN: `protocol.Entry` gained `Direction` / `Incremental` / `Swift`, each a
`*string` with the wire keys `direction` / `incremental` / `swift` and no
`omitempty`; `Version` went 1 → 2; the const block gained
`CodeCardOptionInvalid`, `CodeCardOptionConflict`, `CodeCardOptionNeedsCloze`.

`cmd/imoogi-anki/main.go` needed **no edit** — its nine hits are all symbolic
`protocol.Version` references, so the bump reaches the shared skew probe and
all three request-reading subcommands for free. `git diff --stat
cmd/imoogi-anki/main.go` is empty, which is what decides AC-OPT-006c's
"unchanged at one shared probe".

### M3 — Front-end resolution and transport

RED: `(void-function imoogi-props-resolve-direction)`, 5 failing tests.

GREEN: three thin resolvers over the existing `imoogi-props-resolve`, built on
one shared helper that performs the same empty-to-no-value collapse the deck
resolver performs and nothing else — no trimming, no case folding, no
defaulting. `imoogi-scan--file`, the single entry-plist producer, attaches the
three values; `imoogi-process--entry-alist` emits the three keys with an
explicit `:null`; `imoogi-protocol-version` went 1 → 2.

### M4 — Back-end validation gate

RED: `undefined: isClozeStyle`.

GREEN: `internal/anki/planner/card_options.go` carries `isClozeStyle`, derived
from `renderType` rather than from a second name list, and
`validateCardOptions`, the single gate. Both `processEntry` and `migrateOne`
call it BEFORE their render call, from one implementation.

### M5 — Diagnostic pairing

Landed alongside M2's constants rather than after M4, which is what `plan.md`
§F M2 itself argues for: the front end's table and its contract test key on the
constant set, so a code added later than its table entry breaks the pairing
either way. Three table entries added; three codes added to the test-side list;
the front-end-only allowlist is unchanged at its three entries.

### M6 — Non-interference and documentation

Both goldens were re-run against the finished tree on the COMPARE path, and
their files were verified byte-unchanged afterwards (md5 before/after) — so the
pass is a comparison against the M1 recording, not a silent re-blessing.

`README.md`'s Anki section gained `### Anki 카드 옵션 속성`: the three property
names, their recognized values, the nearest-wins inheritance and how to switch
an inherited option off, the cloze-style note-type requirement, the conflict
rule, and the `make build-anki` rebuild the version bump requires. Written in
that section's existing language, which this SPEC does not change.

## §E.3 Run-phase Audit-Ready Signal

```yaml
run_complete_at: 2026-09-20
run_commit_sha: pending-backfill-run-phase
run_status: complete
ac_pass_count: 14
ac_fail_count: 0
ac_gap_count: 0
preserve_list_post_run_count: 3
new_warnings_or_lints_introduced: 0
cross_platform_build:
  darwin_amd64: pass
  windows_amd64: pass
total_run_phase_files: 22
m1_to_mN_commit_strategy: single-commit (the orchestrator commits; this agent ran no writing git command)
```

### Verification batch — commands and observed output

| Claim | Command | Observed |
|---|---|---|
| Builds on both platforms | `go build ./...` ; `GOOS=windows go build ./cmd/imoogi-anki` | exit 0 ; exit 0 |
| Go suite green | `go test ./... -count=1` | exit 0, 22 `ok` packages, zero `FAIL` lines |
| Elisp suite green | `make test-elisp` | exit 0 — `Ran 422 tests, 420 results as expected, 0 unexpected, 2 skipped` |
| Definition of Done item 2 | `make ci-local` | exit 0 (format check, vet, verify-vendor, elisp, Go, shell) |
| Lint clean | `make lint` | exit 0, empty output |
| Format clean | `make fmt-check` | exit 0 |

Elisp test count moved 407 → 422, exactly the 15 tests added (5 property, 5
scan, 2 serialization, 1 write-back, 1 version-skew, 1 cloze predicate), with
`0 unexpected` on both sides of the move and the same 2 GUI-gated skips.

`make ci-local` logs two module-load errors during its boot phase —
`25-flashcards` (no SQLite support in this Emacs) and `06-git` (magit not
vendored). Both are pre-existing environment conditions unrelated to this SPEC,
both are handled by the boot layer as skips, and the target still exits 0.

AC-OPT-006 was initially thinner than the criterion and was closed rather than
reported as a gap. Its "at every request-reading subcommand" clause had tests
for `sync` and `install-models` but only a shared-code argument for `migrate`,
and its older-binary arm had no test driving the front end at all. Two tests
were added: a migrate-subcommand skew test, and an Elisp test feeding a
version-1 binary's `binary_incompatible` response and asserting the front end
renders its OWN table's message (naming `make build-anki`) with none of the
binary's machine-oriented detail.

### AC-OPT-005 pin audit (Definition of Done item 4)

The grep was re-run AS WRITTEN — hyphen form in the alternation, no
`--include` filter — and every hit read. All fourteen table rows read `2`. The
three out-of-scope groups the criterion names (clipboard, org-preview,
`.moai/**` prose) account for every remaining hit except one.

**The table is now fifteen rows, not fourteen.** The extra pin is
`internal/anki/protocol/card_options_test.go`, a file this SPEC created; it
references `protocol.Version` symbolically and so cannot drift. It is a new pin
rather than one the table missed — the table enumerated the pre-SPEC tree
correctly. `acceptance.md` is a plan-phase artifact this agent does not own, so
the row is reported rather than added.

One literal `protocol_version 1` remains in the Anki subsystem, at
`tests/anki-sync-error-test.el:117`. It is deliberate: it is the AC-OPT-006b
fixture, in which the stub answers exactly as an older binary would. A fixture
that declared version 2 could not exercise the older-binary arm at all.

### Gaps and deviations — stated rather than left to be noticed

1. **RED evidence is uneven across milestones.** M2 and M3 have verbatim
   behavioral RED (a compile failure on the new fields, then
   `(void-function imoogi-props-resolve-direction)`). M4's first RED was only
   structural — a compile failure on `isClozeStyle` — so a behavioral RED was
   captured afterwards by temporarily removing BOTH `validateCardOptions`
   calls and re-running the gate tests. That run reported, among others:

   ```
   --- FAIL: TestGatePrecedesRendering
       card_options_test.go:320: code = "org_parse_error", want "card_option_needs_cloze"
       card_options_test.go:329: code = "cloze_marker_missing", want "card_option_invalid"
   --- FAIL: TestMigrationPathSharesTheGate
       card_options_test.go:386: migration issued addNote 1 / deleteNotes 0 for a rejected entry, want none
   ```

   Both files were restored from a byte copy taken before the edit and the
   package re-run green. **M5 has no captured RED** — its two edits landed
   alongside M2's constants and the pairing tests were run only after. The
   pairing contract is bidirectional and mechanical, so the tests would have
   failed had either edit been wrong, but that is an argument rather than an
   observation.

2. **Some criteria are pins, not RED-GREEN cycles.** AC-OPT-002c, AC-OPT-002d,
   and the two "absent resolves to no value" scan assertions passed on the
   pre-SPEC tree by construction: they pin behavior this SPEC must NOT change.
   That is what they are for, and it is why they cannot show a RED.

3. **AC-OPT-002c lives in `tests/anki-sync-oneway-test.el`**, not in the two
   files its Decides line names. Its Then clause requires a completed sync run,
   and the stub-runner harness for that lives in that file; duplicating the
   harness to satisfy a filename would be the worse trade. `make test-elisp`,
   the aggregate the criterion also names, still decides it.

4. **The hash-invariance probe only became discriminating at M6.** At M1 the
   option fields did not exist, so `applyHashProbeOptions` was a no-op and the
   test compared an entry against itself. It was given a real body (a valid
   direction plus incremental) once the fields existed. The M1 entry for it
   records a fixture put in place, not a comparison already meaningful.

5. **`spec.md` frontmatter now reads `status: in-progress`.** This is the
   `draft → in-progress` transition this phase owns, and it rides the M1
   commit. It was not named in the delegation's scope fence, so it is called
   out here: it is one line, and reverting it is one line.

6. **The README table's third column describes intent, not behavior.** It says
   what each option will eventually select. The paragraph above it states
   plainly that nothing is rendered yet. Naming the eventual meaning is
   deliberate — a property whose only documented effect is "the binary checks
   it" tells a user nothing about why they would write it.

### Coverage — measured after implementation, against the M1 floors

| Package | M1 floor | After | Verdict |
|---|---|---|---|
| `internal/anki/protocol` | `[no statements]` | `[no statements]` | held |
| `internal/anki/orgdoc` | 100.0% | 100.0% | held |
| `internal/anki/planner` | 91.6% | 92.5% | rose |
| `internal/anki/hashing` | 100.0% | 100.0% | held |
| `cmd/imoogi-anki` | 84.2% | 84.2% | held |

### Scope fence

`internal/anki/orgdoc`, `internal/anki/hashing`, and `cmd/imoogi-anki/main.go`
carry **zero** production diffs (`git diff --stat` empty for all three). The
only orgdoc addition is one new test file, which is the artifact AC-OPT-012a's
own deciding command requires.

The operator's three in-progress files — `CLAUDE.md`,
`modules/project/04-projects.el`, `tests/workspace-bridge-test.el` — were not
touched; their mtimes predate this session.


## §E.4 Sync-phase Audit-Ready Signal

_<pending sync-phase>_
