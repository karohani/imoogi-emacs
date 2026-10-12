---
id: SPEC-PERSPPIN-001
title: "Pin a project Perspective fully to its project"
version: "0.1.1"
status: draft
created: 2026-10-13
updated: 2026-10-13
author: jay
priority: P2
phase: "v0.2.0 target"
module: "modules/project/04-projects.el, modules/project/26-project-notes.el, modules/general/05-transient.el, tests"
lifecycle: spec-anchored
tags: "emacs, project, perspective, project-find-functions, project-notes, transient"
tier: M
related_specs: [SPEC-TRANSIENT-001]
---

## HISTORY

### v0.1.1 (2026-10-13)
- DECIDED (lane-3): plan.md D-1 keeps the `f`/`s`/`d` vs `b`/`c`/`k`
  precedence asymmetry, and D-6 accepts that `c`/`k` still prompt from a
  project-notes buffer (now a known limitation and out of scope). Both
  plan-phase clarification markers in plan.md are resolved.
- ADDED: plan.md D-7 fixes the refresh trigger points for the notes-directory
  list by function name. REQ-PIN-008 states them, and AC-PIN-010 gains a
  refresh-trigger clause.

### v0.1.0 (2026-10-13)
- INITIAL: SPEC created for card t35 (Class C, feature), the follow-up the t34
  report names under "Gaps / residual risk" (`.moai/reports/t34/progress.md`):
  generic `project.el` commands run from a projectless buffer still prompt for a
  project, and the card's broader goal — the Perspective behaves like a
  VSCode/IntelliJ window pinned to one project — was left unaddressed by t34
  (`4285b31`).

## 1. Overview

### 1.1 Purpose

A Perspective that was opened for a project (`imoogi-project-switch-perspective`)
is recorded in `imoogi-project-perspective-alist` (root → Perspective name,
persisted by savehist). Imoogi's own project commands (`f`, `s`, `d` in the
project transient) and, since t34, the project-notes commands already use that
mapping. The generic `project.el` commands do not: from `*scratch*`,
`~/notes/scratch.org`, `*Help*`, or any other buffer that has no project of its
own, `b` (switch to buffer), `c` (compile) and `k` (kill buffers) ask "which
project?" even though the Perspective already answers that question.

This SPEC makes a mapped Perspective answer it, and limits the project
transient's buffer and file candidate lists to the project's own files plus the
project's own notes files.

### 1.2 Key binding note

The card says `C-h p`. The binding in the code is **`C-c h p`**: `C-c h` is the
master transient (`modules/general/05-transient.el:644`) and its `p` entry opens
`imoogi-transient-project`. This SPEC writes `C-c h p` throughout; the menu is
the same one the card means.

### 1.3 Scope

In scope: the project resolution of a buffer that has no project of its own
while a mapped Perspective is current; the candidate lists of the project
transient's `b` and `f` entries; the non-interference guarantees that keep the
new resolution from leaking into explicit project selection, Eglot, remote
roots, and project-notes buffers.

## 2. Glossary

- **Mapped Perspective** — the current Perspective's name appears as the value
  of an entry in `imoogi-project-perspective-alist`; the entry's key is the
  **mapped root**.
- **Own project** — the project a buffer resolves to through any
  `project-find-functions` member other than the Perspective finder this SPEC
  adds, whether or not that project is registered in `project--list`.
- **Projectless buffer** — a buffer with no own project (`*scratch*`,
  `~/notes/scratch.org` while `~/notes` is not a VCS root, `*Help*`,
  `*Messages*`).
- **Implicit lookup** — a `project-current` call whose directory is the
  current buffer's `default-directory` (the argument was omitted or equals it)
  and that runs with `project-current-directory-override` nil.
- **Explicit lookup** — any other `project-current` call: a directory the user
  picked at a prompt, a directory passed by code, or a call under
  `project-current-directory-override`.
- **Notes entry** — the project-notes registry entry whose identity key
  (`imoogi-project-notes--identity-key`) matches a project root; its
  `notes-dir` lies outside the project root (default `~/project-notes/…`).
- **Notes-context buffer** — a buffer visiting a file inside any registered
  notes directory, or carrying buffer-local project-notes context
  (`imoogi-project-notes-source-root`, `imoogi-project-notes-entry-instance-id`,
  or notes metadata).

## 3. Requirements (GEARS)

Requirements `REQ-PIN-001`..`REQ-PIN-015`, contiguous, in three groups. Each
requirement is one GEARS clause, possibly compound (`[Where][While][When] … shall`);
numbered sub-clauses are case-splits and introduce no further trigger. The "Perspective finder" is the observable role of
a `project-find-functions` member; its name and code shape are left to the run
phase.

### 3.1 Resolution through the Perspective

#### REQ-PIN-001 [Event-driven + State-driven]
While `persp-mode` is active and the current Perspective is mapped to a root
that is an existing local directory, when an implicit lookup is made for a
projectless buffer, the Perspective finder shall resolve the lookup to the
project that `project.el` resolves for the mapped root:

1. a VC project rooted at the mapped root when the root is under VC;
2. a `transient` project rooted at the mapped root when it is not (study-note
   roots without a VCS).

Rationale: the result is then identical to what a file buffer inside the root
yields, so `project-buffers`, `project-name`, and every `project.el` consumer
see one project, not two shapes of it.

#### REQ-PIN-002 [Ubiquitous]
The Perspective finder shall be the last member of `project-find-functions`,
ordered after `project-try-vc` and after any member added later by another
module or package, so that a buffer with an own project — registered or not —
always keeps it, and the Perspective never overrides it (the t34 card-review
lesson).

#### REQ-PIN-003 [State-driven]
While `persp-mode` is inactive, or the current Perspective is not mapped, or
the mapped root is not an existing directory (deleted, unmounted), the
Perspective finder shall decline, so every command behaves exactly as before
this SPEC — including prompting — and no project is guessed from the
Perspective name, the buffer list, or `default-directory`.

#### REQ-PIN-004 [Ubiquitous — negated]
The Perspective finder shall not claim an explicit lookup. In particular:

1. a directory chosen at a `project-prompter` prompt or passed by
   `project-switch-project` shall resolve on its own merits;
2. the root selected in `imoogi-project-switch-perspective` (`C-c h p p`,
   `C-x p p`) shall resolve to the selected root even when it is not under VC
   and even when it equals the current buffer's `default-directory`.

#### REQ-PIN-005 [Ubiquitous — negated]
The Perspective finder shall not claim a lookup made while `eglot-lsp-context`
is non-nil, so Eglot never roots a language server at the Perspective's project
for a file that lies outside it.

#### REQ-PIN-006 [Ubiquitous — negated]
The Perspective finder shall not perform file-system operations on a mapped
root that is remote (`file-remote-p`); it shall decline for such a root before
any existence or truename check, because `project-current` runs on every
tab-bar redisplay (`imoogi-tab-name`).

#### REQ-PIN-007 [Event-detected]
When any error is signalled while the Perspective finder evaluates a lookup,
the finder shall decline (return no project) instead of propagating the error,
because `project--find-in-directory` only absorbs `permission-denied` and an
escaping error would break every `project-current` caller.

#### REQ-PIN-008 [Ubiquitous — negated]
The Perspective finder shall not claim a lookup for a notes-context buffer;
such buffers keep the resolution they have before this SPEC (and the
project-notes commands keep resolving them through their own entry), so
`imoogi-project-notes--current-source-root` and journal resume sections are not
redirected to the Perspective's root. The notes-context determination shall not
read the notes registry, folder metadata files, or mounted roots on each lookup,
because the lookup runs on every tab-bar redisplay; it shall rely on buffer-local
state and an in-memory list of notes directories that is refreshed when the
registry or mounted state is written, when mounted roots are rescanned, when a
mounted note is detached, after an unmount, on a Perspective switch, and at
startup (plan.md D-7). Decided (lane-3, 2026-10-13):
the resulting gap — `c`/`k` still prompt from a notes buffer — is a known
limitation (§ 5).

### 3.2 The project transient (`C-c h p`)

#### REQ-PIN-009 [State-driven + Event-driven]
While a mapped Perspective is current, when the user invokes `b` (buffer),
`c` (compile), or `k` (kill buffers) from `C-c h p` in a projectless buffer,
the command shall act on the mapped root's project without a project prompt:

1. `c` shall run `compile` with `default-directory` at the mapped root;
2. `k` shall offer to kill that project's buffers;
3. `b` shall offer the candidate list defined by REQ-PIN-010.

#### REQ-PIN-010 [Event-driven]
When the user invokes `b` from `C-c h p`, the buffer candidate list shall
consist of exactly:

1. the live buffers that belong to the resolved project under `project.el`'s
   belonging relation (`project-buffers`, minus
   `project-ignore-buffer-conditions`); and
2. the live buffers visiting a file inside the resolved project's notes
   directory, when the project has a notes entry;

and shall contain no other buffer. The resolved project is the buffer's own
project when it has one (REQ-PIN-002), otherwise the Perspective's project.

#### REQ-PIN-011 [Event-driven]
When the user invokes `f` from `C-c h p`, the file candidate list shall consist
of the files `project-find-file` offers for the resolved root plus the regular
files inside that root's notes directory (VCS metadata directories excluded),
when the root has a notes entry. Notes files of any other project shall never
appear. The resolved root for `f` is `imoogi-project-context-root`, unchanged
(see REQ-PIN-013).

#### REQ-PIN-012 [Ubiquitous — negated]
Notes files and notes buffers shall not become members of the project outside
the `b` and `f` candidate lists: `project-files`, `project-buffers`,
`project-kill-buffers` (`k`), `project-find-regexp`, `project-compile`'s
`default-directory`, VC operations, and `project-find-file` invoked outside the
transient (`C-x p f`) shall see the same set as before this SPEC.

#### REQ-PIN-013 [Ubiquitous]
The remaining `C-c h p` entries shall keep their current resolution:

1. `f`, `s`, `d` keep resolving through `imoogi-project-context-root`, which
   prefers the Perspective mapping over the buffer's own project (implemented
   and documented in its docstring, `04-projects.el:93`; the workspace-bridge
   tests pin that `f` follows the context root across a Perspective switch and
   falls back to `project.el` when unmapped);
2. `p` (switch project) and `F` (forget project) keep prompting, because both
   are explicit selections and `F` is destructive;
3. `m` (project notes) keeps the t34 resolution;
4. `a`, `l`, `o`, `n`, `N`, `#`, `r`, `K` are Perspective commands with no
   project resolution and are untouched.

Decision recorded: this preserves an existing asymmetry — in Perspective A, a
buffer of project B gets A's files from `f` but B's buffers from `b`.
**Decided (lane-3, 2026-10-13):** the asymmetry is kept as written; unifying in
either direction regresses a tested behaviour or t34's card-review lesson, and
the card's goal is met without it (plan.md D-1). Unification is out of scope
(§ 5).

### 3.3 Compatibility with t34

#### REQ-PIN-014 [Ubiquitous]
`imoogi-project-notes--current-entry` shall return the same entry as at
`4285b31` for the four t34 scenarios (`tests/project-notes-perspective-test.el`):
scratch in a pinned Perspective resolves to its project; an unmapped
Perspective yields no entry; a buffer of another registered project keeps its
project; a buffer of an unregistered project does not borrow the Perspective's
entry. The t34 Perspective fallback branch shall be retained as defence in
depth for lookups where the finder declines.

#### REQ-PIN-015 [Event-detected]
When the resolved project has no notes entry, or its root is remote (the notes
identity key refuses remote roots), the `b` and `f` candidate lists shall equal
the project-only lists and no error shall be shown.

## 4. Command inventory (`C-c h p`, `imoogi-transient-project`)

| Key | Command | Resolution today | Needs beyond the finder |
|-----|---------|------------------|-------------------------|
| `p` | `imoogi-project-switch-perspective` | prompts (explicit) | must stay a prompt; selection must bypass the finder (REQ-PIN-004) |
| `f` | `imoogi-project-find-file` | Perspective-first context | notes files in the candidate list (REQ-PIN-011) |
| `s` | `imoogi-project-find-regexp` | Perspective-first context | none — notes are not searched (REQ-PIN-012) |
| `d` | `imoogi-project-dired` | Perspective-first context | none |
| `b` | `project-switch-to-buffer` | `project-current t` → prompts when projectless | finder removes the prompt; notes buffers need a union the generic command cannot express (REQ-PIN-010) |
| `m` | `imoogi-project-notes-transient` | t34 fallback | none (REQ-PIN-014) |
| `c` | `project-compile` | `project-current t` | finder alone suffices |
| `k` | `project-kill-buffers` | `project-current t` | finder alone suffices; notes buffers must not be killed (REQ-PIN-012) |
| `F` | `project-forget-project` | `project-prompter` directly | none — keeps prompting by design |
| `a l o n N # r K` | Perspective commands | n/a | none |

## 5. Out of Scope

### Out of Scope — Unifying the transient's two precedence regimes
- Changing `f`/`s`/`d` to prefer the buffer's own project, or `b`/`c`/`k` to
  prefer the Perspective. Decided (lane-3, 2026-10-13): the asymmetry is kept
  as written (plan.md D-1).

### Out of Scope — Commands outside `C-c h p`
- Notes inclusion for `C-x p b` (`consult-project-buffer`), `C-x b`
  (`imoogi-consult-perspective-buffer`), `C-x p f`, and consult sources. They
  inherit the finder's prompt removal automatically but not the notes union.
- Restricting `C-x b` or any other buffer switcher to the project.

### Out of Scope — Notes-context buffers
- Prompt removal for `c`/`k` invoked from a notes-context buffer (REQ-PIN-008
  keeps those buffers on their current resolution). Decided (lane-3,
  2026-10-13): accepted as a known limitation — `C-c h p c` and `k` still
  prompt from a project-notes buffer, because claiming notes buffers would
  redirect `--current-source-root` for worktree-opened notes and the card names
  projectless buffers (scratch), not notes buffers (plan.md D-6).
- Making a notes directory a project, or adding notes to `project-files` /
  `project-buffers` / `project-external-roots`.

### Out of Scope — Perspective bookkeeping
- Repairing, pruning, or migrating `imoogi-project-perspective-alist` entries
  (stale roots, duplicate names restored by savehist).
- Mapping a Perspective that was created by hand (`C-c h p a`) to a project.
- Treemacs changes (`imoogi-treemacs--perspective-project-root` keeps its own
  fallback).

### Out of Scope — Remote projects
- Pinning a Perspective to a TRAMP root. Such a mapping declines (REQ-PIN-006).

## 6. Constraints

- Emacs 30.2 with bundled `project.el` 0.11.1; `perspective.el`
  20260211; completion is vertico + consult.
- `persp-current-name` signals when `persp-mode` is off (`persp-mode-guard`);
  every Perspective query in the finder sits behind a `persp-mode` check.
- `project-try-vc` caches per directory (`vc-file-setprop`); the finder's cost
  per call is one alist scan, one existence check, an in-memory notes-context
  check, and the cached VC lookup on the mapped root.
- `imoogi-project-notes--all-entries` is not cached: it scans local metadata,
  reads the JSON registry, and scans mounted roots on every call. It is
  therefore unusable on the finder's path (REQ-PIN-008).
- Module layering: `04-projects.el` must not require `26-project-notes.el`.
  The notes-context check (REQ-PIN-008) and the notes candidates (REQ-PIN-010,
  REQ-PIN-011) are supplied from the notes module or resolved through
  `fboundp`-guarded calls, as t34 did in the opposite direction.
- Tests run in batch through `./tests/run.sh` with
  `EMACS=/Applications/Emacs.app/Contents/MacOS/Emacs`; no test may write
  `~/.emacs.d/projects`, `~/project-notes`, or the real savehist file.

## 7. Brownfield Delta

### [NEW] Perspective finder (`modules/project/04-projects.el`)
A `project-find-functions` member appended at the end, with the guards of
REQ-PIN-003..008, and an inhibit used by explicit selection.

### [CHANGED] Explicit selection (`modules/project/04-projects.el`)
`imoogi-project-switch-perspective` resolves the selected root with the finder
inhibited.

### [CHANGED] Project transient (`modules/general/05-transient.el`)
`b` invokes an Imoogi buffer command (REQ-PIN-010); `f` keeps
`imoogi-project-find-file`, whose candidate list gains the notes files.

### [CHANGED] Project notes (`modules/project/26-project-notes.el`)
Supplies the notes-context predicate and the notes file/buffer lists for a
root. `--current-entry` is unchanged in behaviour (REQ-PIN-014).

### [UNCHANGED] `project-files`, `project-buffers`, `project-kill-buffers`, Treemacs
No method, advice, or hook on these.

### [NEW / CHANGED] Tests
New `tests/project-perspective-pin-test.el`; amended rows in
`tests/transient-menu-test.el` (`b`) and possibly
`tests/workspace-bridge-test.el` (`f`), see plan.md § A.4.

## 8. Dependencies

- t34 `4285b31` (in base `6feee71`): the `--current-entry` Perspective fallback.
- SPEC-TRANSIENT-001: the transient layout and the `:transient` semantics of
  `C-c h p` (all entries exit except `n`/`N`).

## 9. Traceability

| AC | Verifies |
|----|----------|
| AC-PIN-001, AC-PIN-002 | REQ-PIN-001 |
| AC-PIN-003, AC-PIN-004 | REQ-PIN-002 |
| AC-PIN-005 | REQ-PIN-003 |
| AC-PIN-006 | REQ-PIN-004 |
| AC-PIN-007 | REQ-PIN-005 |
| AC-PIN-008 | REQ-PIN-006 |
| AC-PIN-009 | REQ-PIN-007 |
| AC-PIN-010 | REQ-PIN-008 |
| AC-PIN-011 | REQ-PIN-009 |
| AC-PIN-012 | REQ-PIN-010 |
| AC-PIN-013 | REQ-PIN-011 |
| AC-PIN-014 | REQ-PIN-012 |
| AC-PIN-015 | REQ-PIN-013 |
| AC-PIN-016 | REQ-PIN-014 |
| AC-PIN-017 | REQ-PIN-015 |
