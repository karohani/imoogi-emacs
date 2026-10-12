# progress — SPEC-ANKISRC-001

## §E.1 Plan-phase Audit-Ready Signal

_plan artifacts drafted 2026-10-12 (spec.md, plan.md, acceptance.md); plan-audit iteration 3/3 PASS 0.92 (`.moai/reports/t25/plan-audit.md`)._

## §E.2 Run-phase Evidence

Measured 2026-10-12 on base `fde2122` plus the uncommitted run-phase tree (card t25, branch `WT-anki-open-source`).
cycle_type: tdd. Emacs: `/Applications/Emacs.app/Contents/MacOS/Emacs`.

Where the runs happened: the run-phase agent had read access to the t25 worktree but no write access, because the runtime's worktree-isolation guard blocked it. It ran every check on a byte-for-byte mirror of the t25 tree (`rsync -a --exclude .git`) with the two test changes below applied, and handed back those changes as a patch. Before the patch, the t25 tree itself scored: full ERT `Ran 666 tests, 654 results as expected, 1 unexpected, 11 skipped`, with the one failure being `imoogi-sync-oneway-test-card-options-are-never-written-back`. The patch fixes that failure.

### Verification commands and verbatim tails

| Check | Command | Verbatim tail |
|---|---|---|
| check-parens (tests/run.sh step 1) | `emacs --batch -Q -l t25-checkparens.el` (same three `check-parens` loops as `tests/run.sh`, moved into a file) | `check-parens OK: 122 files under modules/ and tests/` |
| Full ERT | `IMOOGI_TEST_USER_DIR=<scratch> emacs --batch -Q -l tests/run.el` | `Ran 666 tests, 655 results as expected, 0 unexpected, 11 skipped (2026-10-12 22:59:36+0900, 49.278139 sec)` |
| ERT, anki selector | `IMOOGI_ERT_SELECTOR=anki IMOOGI_TEST_USER_DIR=<scratch> emacs --batch -Q -l tests/run.el` | `Ran 51 tests, 51 results as expected, 0 unexpected (2026-10-12 22:59:39+0900, 0.921880 sec)` (baseline 33; 18 new) |
| Go tests | `go test -count=1 ./...` | exit 0; 27 packages `ok`, 0 `FAIL` |
| Go vet | `go vet ./internal/anki/... ./internal/agentipc/... ./cmd/imoogi-anki/... ./cmd/imoogi-agent/...` | exit 0, no output |
| gofmt | `gofmt -l internal/anki/ cmd/imoogi-anki/ internal/agentipc/` | no output |
| Race | `go test -count=1 -race ./internal/anki/opener/ ./cmd/imoogi-anki/` | `ok .../internal/anki/opener 1.479s`, `ok .../cmd/imoogi-anki 1.549s` |
| Coverage | `go test -count=1 -cover ./internal/anki/opener/ ./internal/anki/protocol/ ./internal/anki/planner/ ./cmd/imoogi-anki/` | opener 88.4%, protocol 100.0%, planner 92.8%, cmd/imoogi-anki 82.2% |

**Isolation check.** All 101 paths recorded in the scratch `.org-id-locations` from the patched full ERT run lie under `/private/var/folders` or `/var/folders`. The pre-patch run recorded 102, also all temporary. The user's real `~/.emacs.d/.org-id-locations` still has its earlier mtime (2026-10-09 07:49:40), so no test run wrote to it.

**R-7 measurement (M3 obligation).** Script: `emacs --batch -Q -l t25-r7-bench.el`. Each Org file has 10 cards, spread over 20 directories, and each outcome was timed once.

```
R-7 N=100 files (1000 headings): found-uncached 0.127s, not-found 0.019s, found-cached 0.019s
R-7 N=500 files (5000 headings): found-uncached 0.085s, not-found 0.097s, found-cached 0.082s
R-7 N=1000 files (10000 headings): found-uncached 0.177s, not-found 0.171s, found-cached 0.171s
```

Every timing is well under the 10 s eval budget (REQ-010.3). These numbers come from `-Q`, so the user's mode hooks did not run. The scan only enters `org-mode` (with `delay-mode-hooks`) in files where the text regexp found a hit.

### AC → test matrix

| AC | Test(s) | Status |
|---|---|---|
| AC-ASRC-001 | `imoogi-anki-src-sync-first-sync-gives-heading-local-ids` | PASS |
| AC-ASRC-002 | `imoogi-anki-src-sync-preserves-existing-ids`; `imoogi-writeback-test-org-id-created-and-preserved` (baseline) | PASS |
| AC-ASRC-003 | `imoogi-anki-src-sync-modified-buffer-and-failed-file`; O5: `imoogi-anki-src-sync-save-failure-drops-unsaved-id-location` | PASS |
| AC-ASRC-004 | `imoogi-anki-src-sync-duplicate-ids-are-sent-without-org-id` | PASS |
| AC-ASRC-005 | `imoogi-anki-src-sync-registers-ids-and-saves-once-at-the-end` | PASS |
| AC-ASRC-006 | Go `TestRun_SourceLink_AppendedToBackFieldOnly`; Elisp `imoogi-anki-src-error-table-has-org-id-codes`, `imoogi-error-test-*` table pairing | PASS |
| AC-ASRC-007 | Go `TestRun_SourceLink_UpdatesExistingNoteOnceAndHashStaysConsistent` | PASS |
| AC-ASRC-008 | Go `TestEntryOrgIDIsOptionalOnTheWire`, `TestValidOrgID`, `TestOrgIDInvalidCodeIsDeclared`; Elisp `imoogi-anki-src-process-org-id-is-optional`, `imoogi-anki-src-scan-reads-heading-local-id-only` | PASS |
| AC-ASRC-009 | (a) `imoogi-anki-src-open-cache-hit-opens-heading`; (b) `imoogi-anki-src-open-cached-file-outside-targets-is-accepted` | PASS |
| AC-ASRC-010 | `imoogi-anki-src-open-moved-file-is-rescanned-and-recached` | PASS |
| AC-ASRC-011 | `imoogi-anki-src-open-missing-id-creates-nothing` (both the deleted-file and the never-existed cases) | PASS |
| AC-ASRC-012 | `imoogi-anki-src-open-duplicate-lists-candidates` (cache absent / cache pointing at one copy); O4: `imoogi-anki-src-open-trailing-whitespace-still-counts` | PASS |
| AC-ASRC-013 | `imoogi-anki-src-open-keeps-unsaved-user-buffer-untouched` | PASS |
| AC-ASRC-014 | Go `TestOpenRejectsBadLinksBeforeCallingEmacs`, `TestOpenPassesDecodedIDToTheFixedReceiver`, `TestLispStringQuotesBackslashAndQuote`, `TestOpenSubcommandRejectsMissingOrBadURL`; Elisp `imoogi-anki-src-open-rejects-ids-outside-the-character-set` | PASS |
| AC-ASRC-015 | (a) `TestOpenLaunchesEmacsAndGivesUpAtTheBound`; (b) `TestOpenLaunchesEmacsThenEvaluates`; (c) `TestOpenWithoutEmacsclientOnlyNotifies`; (d)(f) `TestOpenClassifiesTheReceiverResult`; (e) `TestOpenEnforcesPerCallTimeouts`, `TestOpenAttachesPerCallLimits`; additional failure paths: `TestOpenNotifiesWhenEmacsCannotBeLaunched`, `TestOpenNotifiesWhenEmacsclientCannotRun` | PASS |
| AC-ASRC-016 | `TestInstallGeneratesAppWithoutRegistering`, `TestAppleScriptSourceEscapesTheBinaryPath`, `TestInstallURLHandlerSubcommandGeneratesWithoutRegistering` | PASS |
| REQ-011.7 | `imoogi-anki-src-open-receiver-is-required-by-the-module` | PASS |
| DoD — operator click | none (cannot be automated) | GAP (operator) |

### Existing tests changed (SPEC-driven fixture updates)

REQ-ASRC-001 changes what a sync writes: a card heading without its own `:ID:` now receives one. Two existing one-way tests assert that a sync's on-disk delta is `ANKI_NOTE_ID` alone, so their fixtures now start with a heading-local `:ID:` line. Each test also binds `org-id-locations-file`, `org-id-locations`, and `org-id-files` to temporary state. Neither assertion was weakened.

- `imoogi-sync-oneway-test-ac024-synced-entry-leaves-org-untouched` (changed earlier in this run phase)
- `imoogi-sync-oneway-test-card-options-are-never-written-back` (changed in this pass; it was the one failing test)

`tests/anki-error-test.el` registers `org_id_invalid` as a Go-emitted code. It also registers `org_id_duplicated` and `org_id_create_failed` as codes that exist only in Elisp (DD-4).

### Deviations from the SPEC text

1. **Receiver returns a fourth value.** `imoogi-anki-open-id` returns `"invalid-id"` for an ID outside the character set and opens no file (REQ-011.1). The handler treats every value other than the three normal ones as a failure and shows a notification, which REQ-010.2 already requires. The handler validates the ID first, so in practice this value only appears when the receiver is called directly.
2. **The cache save after a found heading is conditional.** `imoogi-anki-open--register` registers and saves only when the cache does not already map the ID to that file (compared by truename). REQ-011.5 reads as a save on every hit, but the end state of `org-id-locations-file` is the same either way.
3. **Receiver resolution order.** The receiver does not look up the cache first and then scan. It adds the cached file to the target-file set when that file lies outside the targets, then scans every file once. Since F8 already requires a full uniqueness scan even on a cache hit, the outcome matches REQ-011.2/011.3 for every case in AC-009..012.
4. **How the ID pass finds its files.** DD-9 describes detecting ID-less headings in a temporary buffer. The implementation instead runs the normal scan, collects the files whose entries lack `:org-id`, runs the ID pass on those files only, and rescans when any file was saved. The result is the same: only those files are visited, and the request reflects what is on disk.
5. **Extra guard in the ID pass.** Beyond REQ-001.3, a clean user buffer whose file changed on disk (`verify-visited-file-modtime` fails) is refused as a file failure, rather than saved over the newer file. `ask-user-about-lock`, `ask-user-about-supersession-threat`, `yes-or-no-p`, and `y-or-n-p` are rebound to signal instead of prompting, so the pass can never ask the user anything (O2).
6. **Installer additions beyond DD-8.** After `Info.plist` is rewritten, the installer re-signs the app with `codesign --force --sign -`, because the rewrite breaks osacompile's ad-hoc signature. It sets `LSUIElement`, so the app never shows a Dock icon. It refuses to replace an existing `ImoogiAnki.app` whose `Info.plist` does not carry its own bundle ID. The shim shows an AppleScript alert only for failures other than the binary's own exit codes 1 and 2, which the binary has already reported with a notification.
7. **Exit codes.** `imoogi-anki open` exits 2 when a link is rejected before Emacs is contacted, and 1 for every other failure. The SPEC only requires a non-zero code.
8. **`internal/agentipc`.** Two exported wrappers were added: `FindEmacsclient` and `ParseLispString`. Neither changes behavior. `imoogi-agent` still uses its own `classify`, and its tests (`go test ./internal/agentipc/ ./cmd/imoogi-agent/`) pass unchanged.
9. **Optional audit notes folded in.** For O4, the uniqueness regexp ends in `[ \t]*$`. For O5, IDs made in a modified buffer or on a failed pass are `remhash`-ed from the in-memory cache.

### Gaps (not observed)

- **Operator click (DoD).** Nobody has installed the handler on macOS, synced, and clicked "Emacs에서 원본 열기" on a real card, either with Emacs running or with it closed.
- **R-4: does Anki pass custom-scheme links to LaunchServices?** Only the operator can verify this. If Anki does not hand the link over, M4 and M5 deliver nothing, so record this result first.
- **The real macOS tools never ran.** `osacompile`, `codesign`, `lsregister`, `osascript`, `open -a`, and a real emacsclient eval were all fake runners in the tests. Nobody has checked that the generated `.app` compiles, launches, and receives `open location`.
- **Daemon without a GUI frame.** The receiver focuses `selected-frame`. With a daemon and no GUI frame the heading opens but nothing is visible. This case is documented in the README and was not tested.
- **No Elisp coverage figure.** The project has no Elisp coverage tool, so Elisp coverage is evidenced only by the AC matrix above.
- **Frontmatter status.** All four artifacts still read `status: draft`. The `draft → in-progress` change and the `updated:` refresh belong on the lane's M1 commit, which this pass did not make.

### Residual risk

- The R-7 timings were measured under `-Q`. In a full user configuration, `org-mode` hooks run only on files where the text search found a hit, so the effect on timing should stay small.
- If eval times out (R-7), Emacs may still finish opening the heading after the failure notification has appeared, as the `@MX:WARN` on `ExecRunner` notes.

## §E.3 Run-phase Audit-Ready Signal

```yaml
run_complete_at: 2026-10-12
run_commit_sha: "<pending — lane commits>"
run_status: complete-pending-operator-verification
ac_pass_count: 16
ac_fail_count: 0
operator_gaps: [dod-operator-click, r4-anki-custom-scheme-handoff]
preserve_list_post_run_count: 5   # plan.md §A.1 items, all intact
new_warnings_or_lints_introduced: 0   # go vet, gofmt, check-parens
cross_platform_build:
  darwin: "go build ./... ok"
total_run_phase_files: 24   # 12 modified + 12 new (opener/ counts 5), excluding SPEC artifacts
m1_to_mN_commit_strategy: "lane decides (uncommitted tree handed back)"
```

## §E.4 Sync-phase Audit-Ready Signal

_<pending sync-phase>_
