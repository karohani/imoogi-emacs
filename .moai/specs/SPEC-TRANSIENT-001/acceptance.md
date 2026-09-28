# Acceptance Criteria — SPEC-TRANSIENT-001

Every criterion below is binary-testable (ERT assertion, direct file
inspection, or a scripted/manual interactive check against the mapping
table in `plan.md` § C). GEARS requirements are not restated here — see
`spec.md` § 3 for the REQ-XXX requirement layer; this file is the
Given-When-Then verification layer per REQ↔AC traceability below.

## AC Matrix

### AC-001 — Master menu renders the expected suffix set (REQ-001, REQ-002)

**Given** a fresh Emacs session booted with the migrated configuration,
**When** the user presses `C-c h`,
**Then** the `imoogi-transient-master` popup appears, listing exactly the 6
suffixes `w`, `p`, `g`, `z`, `t`, `q` with the descriptions from `plan.md`
§ C.5.

### AC-002 — Sub-menus render the expected suffix set (REQ-001)

**Given** `imoogi-transient-master` is open,
**When** the user presses each of `w`, `p`, `g`, `z` in turn (as separate
sub-scenarios),
**Then**:
- AC-002a: pressing `w` opens `imoogi-transient-window` listing exactly the
  17 suffixes `h l j k H L J K s v d D b f a m q` bound to the commands in
  `plan.md` § C.1.
- AC-002b: pressing `p` opens `imoogi-transient-project` listing exactly the
  8 suffixes `f s b d p k c q` bound to the commands in `plan.md` § C.2.
- AC-002c: pressing `g` opens `imoogi-transient-git` listing exactly the 5
  suffixes `s l b d q` bound to the commands in `plan.md` § C.4.
- AC-002d: pressing `z` opens `imoogi-transient-zoom` listing exactly the 4
  suffixes `i o 0 q` bound to the commands in `plan.md` § C.3.

### AC-003 — Stay-open behavior in `imoogi-transient-window` (REQ-003)

**Given** `imoogi-transient-window` is open,
**When** the user presses `h` (`windmove-left`),
**Then** window focus moves left AND the `imoogi-transient-window` popup
remains open (the next key press is still interpreted as a suffix of the
same prefix, with no need to re-invoke `C-c h w`).

### AC-004 — Exit behavior in `imoogi-transient-window` (REQ-004)

**Given** `imoogi-transient-window` is open,
**When** the user presses `D` (`delete-other-windows`),
**Then** the other windows are deleted AND the transient popup closes.

### AC-005 — Stay-open behavior in `imoogi-transient-zoom` (REQ-003)

**Given** `imoogi-transient-zoom` is open,
**When** the user presses `i` (`text-scale-increase`) twice in succession
without re-invoking the prefix,
**Then** the text scale increases twice AND the popup remains open between
the two presses.

### AC-006 — Exit behavior in `imoogi-transient-zoom` (REQ-004)

**Given** `imoogi-transient-zoom` is open,
**When** the user presses `0` (reset to `text-scale-set 0`),
**Then** the text scale resets to 0 AND the popup closes.

### AC-007 — Whole-menu exit behavior for project/git/master (REQ-005)

**Given** `imoogi-transient-project` is open,
**When** the user presses any bound suffix (e.g. `f` for
`project-find-file`),
**Then** the command executes AND the popup closes — repeated for
`imoogi-transient-git` (e.g. `s` for `magit-status`) and
`imoogi-transient-master` (e.g. `z` for opening `imoogi-transient-zoom`),
confirming no suffix in any of these 3 prefixes carries a stay-open
property.

### AC-008 — Explicit `q` suffix present in every prefix (REQ-006)

**Given** each of the 5 transient prefixes in turn,
**When** its suffix list is inspected (via `transient-get-suffix PREFIX
"q"`) or the popup is opened interactively,
**Then** an explicit `"q"` suffix is defined (not solely relying on
transient's implicit default binding), and pressing `q` closes the popup
without invoking any other command.

### AC-009 — `packages.el` declares `transient`, not `hydra` (REQ-007)

**Given** `packages.el` after migration,
**When** `imoogi-required-packages` is inspected,
**Then** it contains the symbol `transient` and does NOT contain the symbol
`hydra`.

### AC-010 — `boot.el` loads the renamed module (REQ-008)

**Given** `boot.el` after migration,
**When** the module `dolist` is inspected,
**Then** it contains the string `"05-transient"` at the exact list index
formerly occupied by `"05-hydra"`, and does NOT contain `"05-hydra"`.

### AC-011 — Module contract (`imoogi-require` + `provide`) (REQ-009, REQ-010)

**Given** `modules/05-transient.el` after migration,
**When** its leading `imoogi-require` call and its trailing `provide` form
are inspected,
**Then** `imoogi-require` lists `'transient` and `'ace-window` (not
`'hydra`), and the file ends with `(provide 'imoogi-transient)`.

### AC-012 — `packages.lock` no longer lists `hydra` as active (REQ-011)

**Given** `packages.lock` after migration,
**When** searched for `hydra`,
**Then** no row presents `hydra` as a currently-required package (removal
of the row, or an equivalent explicit "no longer required" marking, both
satisfy this criterion — the requirement is that `hydra` no longer reads as
active).

### AC-013 — Test migration passes (REQ-012)

**Given** `tests/treemacs-tool-window-test.el` is run via `tests/run.sh`,
**When** the ERT suite executes,
**Then** the test previously named
`imoogi-treemacs-file-tree-wrapper-is-used-by-master-hydra` has been renamed
to a name containing no "hydra" substring, asserts via
`transient-get-suffix` that `imoogi-transient-master`'s `"t"` suffix's
`:command` is `imoogi-treemacs-toggle-file-tree`, and the assertion passes.

### AC-014 — Comment accuracy in `modules/02-completion.el` (REQ-013)

**Given** `modules/02-completion.el` after migration,
**When** the comment documenting the `C-c h` binding-conflict rationale is
read,
**Then** it no longer contains the substring `hydra-master` and instead
names the migrated master transient entry point (`imoogi-transient-master`
or an equivalent accurate reference).

## Edge Cases

### AC-015 — Unbound key inside a sub-menu (documented, non-blocking)

**Given** `imoogi-transient-window` (or any of the 4 other prefixes) is
open,
**When** the user presses a key not bound to any suffix (e.g. `x`),
**Then** transient's native unbound-key handling applies (the popup remains
open and reports the key as not bound) — this is an *accepted* behavioral
difference from hydra's stricter amaranth "illegal key" warning (see
`spec.md` § 4, `plan.md` § B.5) and MUST NOT be treated as a regression or a
failing criterion.

### AC-016 — Graceful degradation on missing package (regression check)

**Given** the `transient` package is hypothetically absent from
`vendor/elpa/` (out of scope to actually remove it, but exercised via
`boot.el`'s existing `condition-case` isolation reasoning),
**When** `modules/05-transient.el` is loaded,
**Then** `imoogi-require` signals an error naming the missing package,
`boot.el`'s `condition-case` catches it and emits a `display-warning`, and
every other module continues to load — matching the pre-migration
`05-hydra.el` behavior for a missing `hydra`/`ace-window` package
unchanged.

## Definition of Done

- [ ] All of AC-001 through AC-014 pass (functional + structural
      requirements).
- [ ] AC-015 and AC-016 (edge cases) are explicitly exercised and their
      accepted/expected behavior confirmed — not merely assumed.
- [ ] `tests/run.sh` (all 3 phases: `check-parens` syntax validation,
      offline-boot smoke test, ERT suite) passes cleanly with no new
      failures or warnings introduced.
- [ ] `go build ./...` / `go test ./...` are unaffected (this SPEC touches
      no Go files) — re-run as a no-op confirmation only.
- [ ] No stray reference to `hydra` remains in `packages.el`,
      `packages.lock`, `boot.el`, `modules/05-transient.el`, or
      `modules/02-completion.el` (verified via `grep -rn hydra` over the
      affected files, excluding `vendor/` and any incidental prose in
      `research.md`/`plan.md`/`spec.md` themselves, which intentionally
      retain historical references for traceability).
- [ ] `modules/05-transient.el` follows the project's module contract
      (`imoogi-require` first, `use-package`/`transient-define-prefix`
      body, trailing `provide`) per `structure.md` § Module Organization
      Convention.
- [ ] Manual interactive smoke test performed: `C-c h` → each of the 4
      sub-menus → at least one stay-open head and one exit head per menu
      (where applicable) confirmed against `plan.md` § C.

## Quality Gate Criteria (TRUST 5, adapted for this Emacs Lisp SPEC)

- **Tested**: verified via `tests/run.sh`'s existing 3-phase harness
  (syntax, offline-boot smoke test, ERT), including the AC-013 test
  rewrite. This project's ERT suite does not carry numeric coverage
  instrumentation (no `pytest --cov`/`go test -cover` equivalent exists for
  this Elisp harness); test adequacy for this SPEC is judged by AC-001
  through AC-016 collectively covering every changed file and every
  behavioral semantic (stay-open vs. exit) rather than by a percentage
  threshold. This is a documented deviation from the numeric
  `test_coverage_target: 85` in `quality.yaml`, which targets the project's
  Go side; no equivalent instrumented target exists on the Elisp side.
- **Readable**: `imoogi-transient-*` naming follows the project's existing
  `imoogi-` namespacing convention (plan.md § B.1); suffix descriptions
  reuse the original Korean hint text verbatim for continuity.
- **Unified**: `modules/05-transient.el` follows the same `use-package` /
  `imoogi-require` / `provide` structure as every other `modules/NN-*.el`
  file; no new stylistic pattern introduced.
- **Secured**: not applicable — no network access, no credential handling,
  no user-input parsing beyond Emacs's own keymap dispatch is introduced by
  this migration.
- **Trackable**: implementation commits follow Conventional Commits scoped
  to `SPEC-TRANSIENT-001` per the project's git conventions (`plan.md` §A
  route: Tier M, Route A Hybrid Trunk main-direct — direct commits to
  `main`, no per-phase PR).
