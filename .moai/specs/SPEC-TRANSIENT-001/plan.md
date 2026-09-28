# Plan — SPEC-TRANSIENT-001

## §A Context

Standalone SPEC; no Epic grouping (first SPEC in this project). Development
methodology per `.moai/config/sections/quality.yaml`
`constitution.development_mode`: `tdd` (RED-GREEN-REFACTOR). Route:
Tier M → **Route A, Hybrid Trunk main-direct** (`spec-workflow.md` §
Phase Transitions) — commits land directly on `main`, no per-phase PR/branch.
The Conditional Design Route (`plan → design → run`) does not apply: there is
no frontend-component/page deliverable in the web-UI sense this SPEC targets
(an Emacs Lisp pop-up menu is not a UI-surfaced SPEC under that heuristic).

## §B Key Design Decisions (highest change-likelihood — review these first)

### B.1 Transient prefix naming convention

**Decision**: name the 5 new `transient-define-prefix` commands
`imoogi-transient-window`, `imoogi-transient-project`,
`imoogi-transient-zoom`, `imoogi-transient-git`, `imoogi-transient-master`.

**Rationale**: the project's own convention namespaces every exported
command with `imoogi-` (`imoogi-treemacs-toggle-file-tree`,
`imoogi-consult-perspective-buffer`, `imoogi-project-switch-perspective`,
`imoogi-reload`, ...) — the pre-migration `hydra-*` names were themselves
*not* namespaced, which is atypical for this codebase. Adopting
`imoogi-transient-*` both fixes that inconsistency and keeps a clear,
greppable 1:1 correspondence with the retired `hydra-*` names during review
(`hydra-window` → `imoogi-transient-window`, etc.), which matters because
the retired names are also directly referenced by test code and a comment
(`research.md` § 4). This is the single most reviewable naming choice in the
SPEC — reconsider it here before implementation, not mid-migration.

### B.2 Module rename-in-place, no renumbering

**Decision**: `git mv modules/05-hydra.el modules/05-transient.el` — same
numeric slot `05`, no shift of any other module.

**Rationale**: verified in `research.md` § 3 — the in-flight renames visible
in `git status` (15→16, 16→18, 17→19, 18→20, 19→21) are *insertions*
(a new module added at slot 15, shifting everything after it). This SPEC
swaps an existing module's backing package with no new module inserted
before or after slot 05, so a rename-in-place is the correct, minimal-diff
choice. Also verified: the documented load-order dependency on
`06-git.el` is not a real functional constraint (symbols resolve at
keypress time, not load time — precedent already exists in the current file
via the `t` → `imoogi-treemacs-toggle-file-tree` head, which references a
function defined in `07-treemacs.el`, two slots later). No reordering
needed.

### B.3 Explicit `"q"` suffix in every prefix

**Decision**: every one of the 5 transient prefixes carries its own explicit
`("q" "종료" transient-quit-one)`-shaped suffix, rather than relying solely
on transient's implicit default `q` binding.

**Rationale**: behavioral parity with the pre-migration hydras, where every
menu explicitly bound `q` to `nil` (the quit action) as a visible menu item.
Explicit binding also keeps the quit option visible in transient's rendered
popup (matching the pre-migration hint text's `_q_: 종료` line) rather than
being an invisible fallback the user has to already know about.

### B.4 Stay-open vs. exit mapping (the amaranth/blue → `:transient t` translation)

**Decision**: a pre-migration head with **no** explicit `:color` marker
(under a `:color amaranth` hydra) becomes a transient suffix carrying
`:transient t`. A pre-migration head explicitly marked `:color blue` (or
under a whole-hydra `:color blue`) becomes a transient suffix with **no**
`:transient` property — transient's default (exit-after-invoke) already
matches.

**Rationale**: this is the literal semantic mapping the user's task
description specifies. See § C below for the full head-by-head table this
decision produces.

### B.5 Accepted behavioral gap — no illegal-key warning

**Decision**: hydra's `:color amaranth` "illegal key" warning-on-unbound-key
behavior is **not** replicated in transient. Pressing an unbound key inside
any of the 5 new transient popups uses transient's own native
unbound-key handling (transient stays open and reports the key as not
bound — it does not crash or silently exit).

**Rationale**: no directly portable transient equivalent to hydra's amaranth
warning exists (`research.md` § 3, cross-checked against
`vendor/elpa/transient-0.13.4/transient.el`). This is out of scope per
`spec.md` § 4 and is called out here so it is not mistaken for an
oversight during review.

## §C Head-by-Head Mapping Table (hydra → transient)

Every row is drawn directly from the verified pre-migration source in
`research.md` § 1. "Stay-open" = the suffix carries `:transient t`.
"Exit" = default transient behavior (no `:transient` property needed).

### C.1 `hydra-window` → `imoogi-transient-window` (whole-hydra `:color amaranth`)

| Key | Command | Suffix description (원문 힌트) | Stay-open / Exit |
|---|---|---|---|
| `h` | `windmove-left` | ← | Stay-open (`:transient t`) |
| `l` | `windmove-right` | → | Stay-open |
| `j` | `windmove-down` | ↓ | Stay-open |
| `k` | `windmove-up` | ↑ | Stay-open |
| `H` | `shrink-window-horizontally` | 축소← | Stay-open |
| `L` | `enlarge-window-horizontally` | 확대→ | Stay-open |
| `J` | `enlarge-window` | 확대↓ | Stay-open |
| `K` | `shrink-window` | 축소↑ | Stay-open |
| `s` | `split-window-below` | 수평분할 | Stay-open |
| `v` | `split-window-right` | 수직분할 | Stay-open |
| `d` | `delete-window` | 삭제 | Stay-open |
| `D` | `delete-other-windows` | 나머지삭제 | **Exit** (was explicit `:color blue`) |
| `b` | `imoogi-consult-perspective-buffer` | 버퍼전환 | Stay-open |
| `f` | `find-file` | 파일열기 | Stay-open |
| `a` | `ace-window` | ace-window | Stay-open |
| `m` | `ace-swap-window` | 스왑 | Stay-open |
| `q` | (quit) | 종료 | **Exit** (was explicit `:color blue`) |

### C.2 `hydra-project` → `imoogi-transient-project` (whole-hydra `:color blue`)

| Key | Command | Suffix description | Stay-open / Exit |
|---|---|---|---|
| `f` | `project-find-file` | 파일찾기 | Exit (default) |
| `s` | `project-find-regexp` | 검색(grep) | Exit |
| `b` | `project-switch-to-buffer` | 버퍼 | Exit |
| `d` | `project-dired` | dired | Exit |
| `p` | `imoogi-project-switch-perspective` | 프로젝트+작업공간 전환 | Exit |
| `k` | `project-kill-buffers` | 버퍼모두닫기 | Exit |
| `c` | `project-compile` | 컴파일 | Exit |
| `q` | (quit) | 종료 | Exit |

### C.3 `hydra-zoom` → `imoogi-transient-zoom` (whole-hydra `:color amaranth`)

| Key | Command | Suffix description | Stay-open / Exit |
|---|---|---|---|
| `i` | `text-scale-increase` | 확대 | Stay-open |
| `o` | `text-scale-decrease` | 축소 | Stay-open |
| `0` | `(text-scale-set 0)` | 초기화 | **Exit** (was explicit `:color blue`) |
| `q` | (quit) | 종료 | **Exit** (was explicit `:color blue`) |

### C.4 `hydra-git` → `imoogi-transient-git` (whole-hydra `:color blue`)

| Key | Command | Suffix description | Stay-open / Exit |
|---|---|---|---|
| `s` | `magit-status` | status | Exit (default) |
| `l` | `magit-log-current` | log | Exit |
| `b` | `magit-blame` | blame | Exit |
| `d` | `magit-diff-dwim` | diff | Exit |
| `q` | (quit) | 종료 | Exit |

### C.5 `hydra-master` → `imoogi-transient-master` (whole-hydra `:color blue`, entry point)

| Key | Command (post-migration) | Suffix description | Stay-open / Exit |
|---|---|---|---|
| `w` | `imoogi-transient-window` | 창관리 | Exit (default) |
| `p` | `imoogi-transient-project` | 프로젝트 | Exit |
| `g` | `imoogi-transient-git` | Git | Exit |
| `z` | `imoogi-transient-zoom` | 확대/축소 | Exit |
| `t` | `imoogi-treemacs-toggle-file-tree` (unchanged) | treemacs | Exit |
| `q` | (quit) | 종료 | Exit |

`(global-set-key (kbd "C-c h") 'imoogi-transient-master)` replaces the
pre-migration `hydra-master/body` binding.

## §D Milestones (priority-ordered; no time estimates)

**M1 — Priority: High.** Package manifest & vendor bookkeeping.
`packages.el`: add `transient` to `imoogi-required-packages`, remove
`hydra`; edit the header comment (drop `transient` from the
transitive-dependency example list, leaving `with-editor, dash,
markdown-mode`). `packages.lock`: remove the active `hydra 0.15.0` row.
No dependency on later milestones — do this first, lowest risk.

**M2 — Priority: High.** Module migration. `git mv modules/05-hydra.el
modules/05-transient.el`; rewrite its body per § C above: 5
`transient-define-prefix` forms, updated `imoogi-require` call (`'transient
'ace-window` in place of `'hydra 'ace-window`), the `C-c h` binding, and
`(provide 'imoogi-transient)` in place of `(provide 'imoogi-hydra)`. Depends
on M1 (package must be declared before the module that requires it is
exercised, though `imoogi-require` itself only checks `locate-library` —
sequencing M1 first keeps the manifest and the module in sync at every
intermediate commit).

**M3 — Priority: High.** `boot.el` update: change the single dolist string
`"05-hydra"` → `"05-transient"`, same list position. Depends on M2 (the
renamed file must exist before boot.el points at it).

**M4 — Priority: Medium.** Test migration.
`tests/treemacs-tool-window-test.el`: rewrite the test asserting the
treemacs-toggle head of the master menu (currently
`imoogi-treemacs-file-tree-wrapper-is-used-by-master-hydra`, reading
`hydra-master/heads`) to assert via `transient-get-suffix` against
`imoogi-transient-master`'s `"t"` suffix, per the API verified in
`research.md` § 4.1:

```elisp
(ert-deftest imoogi-treemacs-file-tree-wrapper-is-used-by-master-transient ()
  (should (eq (plist-get (cdr (transient-get-suffix 'imoogi-transient-master "t"))
                          :command)
              #'imoogi-treemacs-toggle-file-tree)))
```

Per this project's TDD mode (`quality.yaml` `development_mode: tdd`), this
RED test SHOULD be written/updated *before* M2's implementation lands, then
turned GREEN once M2 completes — the standard RED-GREEN-REFACTOR ordering.
Depends conceptually on M2 (needs the real `imoogi-transient-master` symbol
to exist for the assertion to be meaningful at GREEN time), though authoring
the RED version may precede M2's landing.

**M5 — Priority: Low.** Comment accuracy. `modules/02-completion.el`: update
the one-line comment near the `consult`/`C-c h` conflict note (currently
"...`C-c h'(→consult-history)는 imoogi hydra-master`") to name the migrated
`imoogi-transient-master` entry point instead. No functional change to this
module. Independent of M1-M4; can land any time after M2 (needs the new
symbol name to exist for the reference to be accurate).

**M6 — Priority: High (final).** Verification. Run `tests/run.sh` end to
end (all 3 phases: syntax `check-parens`, offline-boot smoke test, ERT
suite including the M4 test) and manually smoke-test `C-c h` → each of the
4 sub-menus → a representative stay-open head and a representative exit
head per menu, confirming behavior matches § C.

## §E Technical Approach — file-by-file edit summary

| File | Edit |
|---|---|
| `packages.el` | Remove `hydra` from, add `transient` to, `imoogi-required-packages` (§"05-hydra" comment group can be relabeled to reflect the new module name); edit header comment example list. |
| `packages.lock` | Remove the `hydra` row. |
| `modules/05-hydra.el` → `modules/05-transient.el` | `git mv`; full-body rewrite per § C; `imoogi-require` + `provide` updates. |
| `boot.el` | `"05-hydra"` → `"05-transient"` in the module `dolist`. |
| `tests/treemacs-tool-window-test.el` | Rewrite + rename one `ert-deftest` per M4. |
| `modules/02-completion.el` | One-line comment text correction per M5. |

No changes to `cmd/`, `internal/`, `go.mod`, `toolchains.json`,
`toolchains.lock.json`, or any file under `vendor/` (transient is already
vendored — see `research.md` § 2).

## §F Test Strategy

- Existing `tests/run.sh` 3-phase harness is the verification vehicle;
  no new test infrastructure is introduced.
- Phase 1 (`check-parens`) exercises the rewritten `modules/05-transient.el`
  automatically (it globs `modules/*.el`).
- Phase 2 (offline-boot smoke test) exercises the updated `boot.el` dolist
  and confirms the renamed module still loads with `package-archives` nulled
  (air-gap safety — unaffected by this migration, but re-verified as a
  regression check).
- Phase 3 (ERT suite) exercises the M4-updated test.
- No new test file is added; the existing `tests/treemacs-tool-window-test.el`
  suite is sufficient coverage for the one behavior (`t` → treemacs toggle)
  that was previously asserted against hydra internals. Manual interactive
  smoke-testing (§ D, M6) covers the remaining 4 menus' key-binding-and-
  stay-open/exit behavior, since ERT does not currently exercise interactive
  transient/hydra popup behavior for the other menus (pre-migration state
  had no such coverage either — this SPEC does not lower or raise that bar).

## §G Risks & Mitigations

| Risk | Mitigation |
|---|---|
| A suffix's stay-open/exit choice is transcribed incorrectly from § C, silently changing UX. | § C is the single reviewable source of truth for every key; M6's manual smoke test explicitly re-verifies one stay-open and one exit head per menu against § C. |
| `transient-get-suffix` (M4) turns out to behave differently than documented once exercised against a real `transient-define-prefix` (API read from source, not executed at plan-phase). | Plan-phase does not execute code (no implementation in this SPEC); run-phase M4 must confirm the assertion actually passes against the real `imoogi-transient-master` definition before considering M4 done — if the API shape differs, run-phase falls back to inspecting `(get 'imoogi-transient-master 'transient--layout)` directly (also verified present in `research.md` § 4.1) as a documented fallback. |
| `packages.el`'s `imoogi-require` still lists `'hydra` after M2 due to edit-ordering mistake, causing every module-05 load to fail load-time verification even though the file no longer needs hydra. | M2 explicitly includes the `imoogi-require` line edit as part of the single module rewrite, not a separate step — no partial-edit window. |
| Leaving `hydra` in `packages.lock` after removal from `packages.el` reintroduces exactly the audit-drift `packages.lock` exists to prevent. | M1 bundles both edits together. |

## §I MX Tag Plan (Phase 14)

Lightweight scan — this SPEC touches one existing module (behavior-preserving
rewrite) plus small mechanical edits to 4 other files; no new public API
surface, no goroutine-equivalent concurrency risk in Emacs Lisp.

| Target | Tag | Priority | Rationale |
|---|---|---|---|
| `imoogi-transient-master` (in `modules/05-transient.el`) | `@MX:ANCHOR` | P2 | Sole public entry point bound to a global key (`C-c h`); fan_in from `global-set-key` + is the dispatch target of every sub-menu. Matches the ANCHOR criterion (external call surface with meaningful fan-in) despite being new code, because it directly replaces `hydra-master/body`'s prior role. |
| `imoogi-transient-window` / `-project` / `-zoom` / `-git` (in `modules/05-transient.el`) | `@MX:NOTE` | P3 | Each carries the amaranth-vs-blue `:transient t` mapping from `plan.md` §C — worth a short inline note per prefix pointing at that table, since the stay-open/exit choice per suffix is not self-evident from the code alone. |
| `tests/treemacs-tool-window-test.el` rewritten test | `@MX:NOTE` | P3 | Note the `transient-get-suffix` API dependency + the `research.md` §4.1 fallback (`transient--layout` plist) in case the public accessor's shape changes in a future `transient` upgrade. |

No `@MX:WARN` targets identified (no goroutines/concurrency, no complexity >= 15 — each transient prefix is a flat suffix list, no nested control flow). No `@MX:TODO` targets — the SPEC scope is fully covered by AC-001 through AC-016, leaving no known-untested public surface.

## §H Dependencies / Cross-References

- No dependency on any other SPEC (first SPEC in this project).
- Out-of-scope boundary: `spec.md` § 4 (package surface, functional
  behavior, module renumbering, Go toolchain, unrelated docs prose).
- Acceptance criteria: `acceptance.md`.
- Source research: `research.md`.
