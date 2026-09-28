---
id: SPEC-TRANSIENT-001
title: "Migrate hydra pop-up keybinding menus to transient"
version: "0.1.0"
status: draft
created: 2026-08-22
updated: 2026-08-22
author: jay
priority: P2
phase: "v0.1.0 target"
module: "modules/05-transient.el (+ boot.el, packages.el, packages.lock, tests/treemacs-tool-window-test.el)"
lifecycle: spec-anchored
tags: "emacs, transient, hydra, keybindings, ui, migration"
tier: M
---

## HISTORY

### v0.1.0 (2026-08-22)
- INITIAL: SPEC created via `/moai plan` (Phase 6 research + Phase 8 planning
  + Phase 9-10 document creation, combined per explicit small-scope
  instruction). Migrates `modules/05-hydra.el`'s 5 `defhydra` menus to
  `transient`, retiring `hydra` as a dependency.

## 1. Overview

`imoogi-emacs` currently uses the `hydra` package to implement 5 pop-up
keybinding menus (window management, project navigation, text zoom, Git
status, and a master entry-point menu bound to `C-c h`). `transient` — the
menu library that powers Magit and is already vendored in this checkout as a
transitive dependency of `magit` — is the modern, actively-maintained
replacement for this class of pop-up command menu in the Emacs ecosystem.
This SPEC migrates all 5 menus from `hydra` to `transient`, removes `hydra`
as a project dependency, and updates the SSOT package manifest, the module
loader, and the one existing test that inspects a hydra-internal data
structure.

This is a **behavior-preserving** migration: every key binding, every
invoked command, the global entry point (`C-c h`), and the "does the menu
stay open or close after this action" semantics of the existing 5 menus are
carried over unchanged. No new menu items, no removed menu items, and no
functional changes to any of the underlying toggle/window/project/git/zoom
commands are in scope.

## 2. Background

See `research.md` for the full source read, the exact pre-migration
`modules/05-hydra.el` content, the verified `transient` package state
(already vendored via `magit`'s transitive dependency chain), the module
rename-in-place rationale, the verified late-binding load-order reasoning,
and the two additional impact points discovered beyond the pre-supplied
research brief (`tests/treemacs-tool-window-test.el`,
`modules/02-completion.el`).

## 3. Requirements (GEARS)

### REQ-001 [Ubiquitous]
The migrated module (`modules/05-transient.el`) shall reimplement all 5
pre-migration hydra menus — window, project, zoom, git, and master — as
`transient-define-prefix` commands (`imoogi-transient-window`,
`imoogi-transient-project`, `imoogi-transient-zoom`, `imoogi-transient-git`,
`imoogi-transient-master` respectively), each exposing the exact same set of
key bindings and invoked commands as its pre-migration `hydra-*`
counterpart.

### REQ-002 [Ubiquitous]
The migrated module shall bind the global key sequence `C-c h` to
`imoogi-transient-master`, replacing the pre-migration binding to
`hydra-master/body`.

### REQ-003 [When]
When a user presses a suffix key within `imoogi-transient-window` or
`imoogi-transient-zoom` that corresponds to a pre-migration hydra head NOT
marked `:color blue`, the transient prefix shall remain open (via an
explicit stay-open suffix property) after the suffix command executes.

### REQ-004 [When]
When a user presses a suffix key within `imoogi-transient-window` or
`imoogi-transient-zoom` that corresponds to a pre-migration hydra head
marked `:color blue` (`delete-other-windows` in `hydra-window`; the
zoom-reset action and the quit action in `hydra-zoom`), the transient prefix
shall close after the suffix command executes.

### REQ-005 [Ubiquitous]
Every suffix in `imoogi-transient-project`, `imoogi-transient-git`, and
`imoogi-transient-master` shall close its transient prefix after execution,
matching the whole-menu `:color blue` behavior of the corresponding
pre-migration `hydra-project`, `hydra-git`, and `hydra-master`.

### REQ-006 [Ubiquitous]
Each of the 5 transient prefixes shall define an explicit `"q"` suffix bound
to a quitting action, in addition to (not in place of) transient's built-in
`q` binding.

### REQ-007 [Ubiquitous]
`packages.el`'s `imoogi-required-packages` shall declare `transient` as a
direct top-level entry and shall no longer declare `hydra`.

### REQ-008 [Ubiquitous]
`boot.el`'s module-loading `dolist` shall load `"05-transient"` at the exact
list position currently occupied by `"05-hydra"`.

### REQ-009 [When]
When `modules/05-hydra.el` is renamed to `modules/05-transient.el`, the
module's leading `imoogi-require` precondition call shall check for the
`transient` package instead of the `hydra` package, and shall continue to
check for `ace-window` unchanged.

### REQ-010 [Ubiquitous]
The migrated module shall end with `(provide 'imoogi-transient)` in place of
`(provide 'imoogi-hydra)`.

### REQ-011 [Ubiquitous]
`packages.lock` shall no longer carry an active entry for `hydra`.

### REQ-012 [When]
When `tests/treemacs-tool-window-test.el`'s existing test asserting the
treemacs-toggle head of the master menu is run, it shall verify the binding
via the transient suffix definition of `imoogi-transient-master` (using
transient's public `transient-get-suffix` accessor) rather than via the
retired `hydra-master/heads` alist, and its test name shall no longer
contain the substring "hydra".

### REQ-013 [Ubiquitous]
The one-line Korean comment in `modules/02-completion.el` documenting the
`C-c h` binding-conflict rationale shall name the migrated master transient
entry point rather than the retired `hydra-master` symbol.

## 4. Out of Scope

### Out of Scope — Package surface beyond hydra/transient
- `ace-window` and its configuration/keybindings (`M-o`, `aw-keys`,
  `aw-scope`) are unaffected — `ace-window` is not a hydra and needs no
  migration.
- No other entry in `packages.el`'s `imoogi-required-packages` is added,
  removed, or reordered.

### Out of Scope — Functional or behavioral changes
- No new menu heads, no removed menu heads, and no reassigned keybindings
  within any of the 5 menus.
- No change to the global `C-c h` target key sequence, or to any of the
  underlying toggle/window/project/git/zoom functions themselves
  (`imoogi-treemacs-toggle-file-tree`, `imoogi-consult-perspective-buffer`,
  `imoogi-project-switch-perspective`, `windmove-*`, `magit-*`, etc.).
- Replicating hydra's `:color amaranth` "illegal key" warning behavior for
  an unbound key press is out of scope; transient's native unbound-key
  handling is accepted as-is (see `research.md` — no directly portable
  transient equivalent exists).

### Out of Scope — Module renumbering
- No change to the numeric load-order position of the migrated module
  (stays at slot `05`); no other `modules/NN-*.el` file is renumbered as
  part of this SPEC.
- No change to `boot.el`'s module load order beyond the single
  `"05-hydra"` → `"05-transient"` string substitution.

### Out of Scope — Go toolchain / vendor tooling
- No changes to `cmd/`, `internal/`, or the `imoogi-toolchain` Go CLI.
- No re-run of `scripts/vendor.el` as part of this SPEC's run-phase
  deliverable — `transient-0.13.4` is already present in `vendor/elpa/` via
  `magit`'s transitive dependency chain (see `research.md` § 2). A
  maintainer MAY re-run vendoring later to formalize `transient` as a
  directly-installed (rather than transitively-pulled) entry; that
  execution is not required by, or blocking on, this SPEC.

### Out of Scope — Pre-existing documentation inaccuracies unrelated to this migration
- `structure.md`'s note that `05-hydra.el` "depends on ... 06-git.el
  loading first" is a pre-existing minor documentation imprecision
  (load order already places slot 05 before slot 06; see `research.md` §
  3 for the verified late-binding reasoning). Correcting that unrelated
  prose is not part of this SPEC.

## 5. Dependencies

- Depends on: none (first SPEC in this project; no prior SPEC to build on).
- Depended on by: none currently.

## 6. Traceability

- Acceptance criteria: `acceptance.md` (AC-001 through AC-016).
- Implementation plan, head-by-head mapping table, and milestone sequence:
  `plan.md`.
- Source research and verified API citations: `research.md`.
