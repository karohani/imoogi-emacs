---
id: SPEC-PERSPPIN-001
title: "Pin a project Perspective fully to its project"
version: "0.1.3"
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

### v0.1.3 (2026-10-13)
- AUDIT r2 FIX (plan-audit FAIL, harmonic 0.832, D4 unresolved;
  `.moai/reports/plan-audit/SPEC-PERSPPIN-001-plan-audit-r2.md`). Decided
  (lane-3, 2026-10-13):
  - R2-D1 route (a): `f` resolves PROJECT as
    `(or (project-current nil) (and CONTEXT-ROOT (cons 'transient CONTEXT-ROOT)) (project-current t))`,
    and the fallback is gated on a non-nil context root. `s`/`d` share
    `f`'s helper (`imoogi-project-context-call`, `04-projects.el:111-121`), so
    they get the same resolution through the finder's context-call clause.
    `f`/`s`/`d` therefore do not prompt whenever a context root exists
    (REQ-PIN-013.1, plan.md D-2/D-4).
  - R2-D2: a third permitted test edit, E3, makes the `project-root` override
    at `workspace-bridge-test.el:19-20` shape-tolerant. Exactly E1-E3 are
    named in plan.md A.4/§ E, § 7 below, and acceptance.md AC-PIN-015/DoD.
  - R2-D3: the "no longer prompt" claims are scoped to the buffers where they
    hold. The study-note consequence now lists every command that still
    prompts (§ 5), and plan.md D-6 is corrected.
- Optional findings O-a..O-h taken: duplicate plan heading removed (O-a);
  § 7 names both workspace-bridge tests (O-b); `f` must not bind
  `project-files-relative-names` (O-c); unmount row descriptor (O-d); new
  AC-PIN-018 pins the memo (O-e); memo thrash with two frames recorded under
  Residual Risk (O-f); AC-PIN-003 setup sequence (O-g); `file-remote-p`
  checked first in guard 3 (O-h).

### v0.1.2 (2026-10-13)
- AUDIT r1 FIX (plan-audit FAIL 0.792,
  `.moai/reports/plan-audit/SPEC-PERSPPIN-001-plan-audit.md`), all eight
  blocking defects:
  - D1: REQ-PIN-002 now guarantees ordering against members added at depth
    ≤ 90 (default or APPEND). plan.md D-2 installs the finder at depth 95.
  - D2: AC-PIN-013 captures the absolute ALL-FILES argument at
    `project-read-file-name-function`.
  - D3: AC-PIN-011 adds a live file buffer under R and asserts the
    `yes-or-no-p` prompt.
  - D4: option (a) chosen. The `workspace-bridge-test.el:141` stub moves to
    the new entry point and its assertion stays (plan.md A.4, § E,
    AC-PIN-015, DoD agree).
  - D5: "Projectless buffer" excludes notes-context buffers, and the
    study-note Perspective consequence is recorded (§ 5, acceptance.md
    Residual Risk). D-6 is not reopened.
  - D6: § 6 corrects the caching claim. The finder memoizes its answer per
    pinned root (plan.md D-8).
  - D7: the glossary names the real buffer-local variables.
  - D8: the refresh-trigger test covers triggers 1-6, and trigger 7 is
    verified by registration.
- Optional findings taken: truename comparison (O5); a remote DIR declines
  (O6); `defvar` for `eglot-lsp-context` in the test (O4); the `f`/`s`/`d`
  prompt removal and the `m s` change are stated (O1, O2); AC-PIN-009 no
  longer passes vacuously (O3); AC-PIN-006/010 wording (O7); pattern labels
  corrected (O8).

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
- **Projectless buffer** — a buffer that has no own project **and is not a
  notes-context buffer** (`*scratch*`, `~/notes/scratch.org` while `~/notes` is
  not a VCS root, `*Help*`, `*Messages*`). A notes-context buffer with no own
  project is governed by REQ-PIN-008, not by the requirements on projectless
  buffers.
- **Implicit lookup** — a `project-current` call whose directory is the
  current buffer's `default-directory` (the argument was omitted or equals it)
  and that runs with `project-current-directory-override` nil.
- **Explicit lookup** — any other `project-current` call: a directory the user
  picked at a prompt, a directory passed by code, or a call under
  `project-current-directory-override`.
- **Notes entry** — the project-notes registry entry whose identity key
  (`imoogi-project-notes--identity-key`) matches a project root; its
  `notes-dir` lies outside the project root (default `~/project-notes/…`).
- **Notes-context buffer** — a buffer whose visited file's truename
  (`buffer-file-truename`, expanded) lies inside any registered notes directory
  (compared by truename), or in which any of the buffer-local variables
  `imoogi-project-notes-source-root`, `imoogi-project-notes-directory-local`, or
  `imoogi-project-notes-entry-instance-id` is non-nil
  (`26-project-notes.el:48-54`). Folder metadata files are not consulted.

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
project that `project.el` resolves for the mapped root — reusing that resolved
project, without repeating the VC search, for repeated lookups until the next
Perspective switch or a change of the mapping (plan.md D-8):

1. a VC project rooted at the mapped root when the root is under VC;
2. a `transient` project rooted at the mapped root when it is not (study-note
   roots without a VCS).

A notes-context buffer is excluded by the definition of "projectless buffer"
(REQ-PIN-008 governs it).

Rationale: the result is then identical to what a file buffer inside the root
yields, so `project-buffers`, `project-name`, and every `project.el` consumer
see one project, not two shapes of it.

#### REQ-PIN-002 [Ubiquitous]
The Perspective finder shall be ordered after `project-try-vc` and after every
member of `project-find-functions` added — before or after it — at a depth of
90 or less (plain `add-hook`, or `add-hook` with APPEND), so that a buffer whose
own project is found by such a member — registered or not — always keeps it,
and the Perspective never overrides it (the t34 card-review lesson). Members
added at a positive depth above the finder's are not covered by this guarantee.

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
The Perspective finder shall not perform file-system operations on a remote
name: it shall decline, before any existence or truename check, when the mapped
root is remote (`file-remote-p`) or when the lookup directory itself is remote
(a TRAMP buffer in a local Perspective is never claimed, so `c` never compiles
it locally in R). `project-current` runs on every tab-bar redisplay
(`imoogi-tab-name`).

#### REQ-PIN-007 [Event-driven]
When any error is signalled while the Perspective finder evaluates a lookup,
the finder shall decline (return no project) instead of propagating the error,
because `project--find-in-directory` only absorbs `permission-denied` and an
escaping error would break every `project-current` caller.

#### REQ-PIN-008 [Ubiquitous — negated]
The Perspective finder shall not claim a lookup for a notes-context buffer,
with one exception: a lookup of the context root made while
`imoogi-project-context-call` runs `f`/`s`/`d` (REQ-PIN-013.1; plan.md D-2
context-call clause). That exception is scoped to those commands, which never
call `imoogi-project-notes--current-source-root`. Outside it, such buffers keep
the resolution they have before this SPEC (and the
project-notes commands keep resolving them through their own entry), so
`imoogi-project-notes--current-source-root` and journal resume sections are not
redirected to the Perspective's root. The notes-context determination shall not
read the notes registry, folder metadata files, or mounted roots on each lookup,
because the lookup runs on every tab-bar redisplay; it shall rely on buffer-local
state and an in-memory list of notes directories that is refreshed when the
registry or mounted state is written, when mounted roots are rescanned, when a
mounted note is detached, after an unmount, on a Perspective switch, and at
startup (plan.md D-7). The resulting prompt from `c`/`k` in notes-context
buffers is a known limitation (§ 5, plan.md D-6).

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

1. `f`, `s`, `d` keep choosing their root through
   `imoogi-project-context-root`, which prefers the Perspective mapping over
   the buffer's own project (implemented and documented in its docstring,
   `04-projects.el:93`; the workspace-bridge tests pin that `f` follows the
   context root across a Perspective switch and falls back to `project.el`
   when unmapped). One consequence changes. Today they prompt when the context
   root is not under VC. After this SPEC, **whenever a context root exists**,
   they do not prompt, from any buffer — projectless, notes-context, or a
   buffer of another project — because they resolve the context root, falling
   back to `(transient . CONTEXT-ROOT)` (decided lane-3, 2026-10-13; plan.md
   D-4 route (a) for `f`, D-2's context-call clause for `s`/`d`, which share
   `f`'s helper). When no context root exists (unmapped Perspective and no own
   project), they prompt exactly as before (REQ-PIN-003);
2. `p` (switch project) and `F` (forget project) keep prompting, because both
   are explicit selections and `F` is destructive;
3. `m` (project notes) keeps the t34 entry resolution. One consequence
   changes: `C-c h p m s p` (`imoogi-project-notes-setup`) run from a
   projectless buffer in a mapped Perspective uses the mapped root instead of
   prompting for the source folder, because `imoogi-project-notes--project-root`
   calls `project-current`;
4. `a`, `l`, `o`, `n`, `N`, `#`, `r`, `K` are Perspective commands with no
   project resolution and are untouched.

This preserves an existing asymmetry: in Perspective A, a buffer of project B
gets A's files from `f` but B's buffers from `b` (plan.md D-1; unification is out
of scope, § 5).

### 3.3 Compatibility with t34

#### REQ-PIN-014 [Ubiquitous]
`imoogi-project-notes--current-entry` shall return the same entry as at
`4285b31` for the four t34 scenarios (`tests/project-notes-perspective-test.el`):
scratch in a pinned Perspective resolves to its project; an unmapped
Perspective yields no entry; a buffer of another registered project keeps its
project; a buffer of an unregistered project does not borrow the Perspective's
entry. The t34 Perspective fallback branch shall be retained as defence in
depth for lookups where the finder declines.

#### REQ-PIN-015 [Event-driven]
When the resolved project has no notes entry, or its root is remote (the notes
identity key refuses remote roots), the `b` and `f` candidate lists shall equal
the project-only lists and no error shall be shown.

## 4. Command inventory (`C-c h p`, `imoogi-transient-project`)

| Key | Command | Resolution today | Needs beyond the finder |
|-----|---------|------------------|-------------------------|
| `p` | `imoogi-project-switch-perspective` | prompts (explicit) | must stay a prompt; selection must bypass the finder (REQ-PIN-004) |
| `f` | `imoogi-project-find-file` | Perspective-first context; prompts when the context root is not under VC | notes files in the candidate list (REQ-PIN-011); route (a) resolution — no prompt whenever a context root exists, from any buffer |
| `s` | `imoogi-project-find-regexp` | Perspective-first context; prompts when the context root is not under VC | notes are not searched (REQ-PIN-012); context-call clause — no prompt whenever a context root exists, from any buffer |
| `d` | `imoogi-project-dired` | Perspective-first context; prompts when the context root is not under VC | context-call clause — no prompt whenever a context root exists, from any buffer |
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
- Study-note Perspectives: `imoogi-project-notes--open-study-workspace`
  (`26-project-notes.el:2615`) maps a study Perspective to the notes directory
  itself. Every file of a study project is therefore a notes-context buffer,
  and the finder declines for all of them. From **every** note file of that
  project, these still prompt: `C-c h p c`, `C-c h p k`, and every generic
  `project.el` command outside `C-c h p` that resolves with
  `(project-current t)` (for example `C-x p c`, `C-x p k`, `C-x p f`,
  `C-x p g`, `C-x p d`, `C-x p b`). From the same files, `C-c h p b`, `f`, `s`
  and `d` do not prompt (plan.md D-3, D-4). `p` and `F` prompt everywhere by
  design. Only projectless buffers such as scratch are pinned there for every
  command. This consequence of D-6 is accepted (decided lane-3, 2026-10-13).
  Changing it would need a new lane decision.
- Making a notes directory a project, or adding notes to `project-files` /
  `project-buffers` / `project-external-roots`.

### Out of Scope — Reverting incidental behaviour changes
These changes follow from the finder and are accepted. Undoing them is out of
scope:
- `f`/`s`/`d` stop prompting when the context root is not under VC — from any
  buffer, as long as a context root exists. With no context root they still
  prompt (REQ-PIN-013.1).
- `C-c h p m s p` (project-notes setup) from a projectless buffer in a mapped
  Perspective uses the mapped root instead of prompting for the source folder
  (REQ-PIN-013.3).
- The tab name of a projectless buffer shows the project name, and the org
  preview's allowed roots for `~/notes/scratch.org` include the mapped root
  (plan.md A.2).

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
- `project-try-vc` caches only **positive** results per directory
  (`vc-file-setprop`, project.el L545-551). For a mapped root that is not under
  VC, every uncached call repeats the `locate-dominating-file` walk plus
  `project--value-in-dir` (`hack-dir-local-variables` in a temp buffer).
  Therefore the finder memoizes its resolved project per pinned root, keyed by
  (Perspective name, raw mapped root) and cleared on a Perspective switch
  (plan.md D-8). Per call, the finder costs one alist scan, one
  `file-directory-p`, an in-memory notes-context check, and a memo hit.
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
A `project-find-functions` member installed at depth 95, with the guards of
REQ-PIN-003..008, a per-pinned-root memo (plan.md D-8), and an inhibit used by
explicit selection.

### [CHANGED] Explicit selection (`modules/project/04-projects.el`)
`imoogi-project-switch-perspective` resolves the selected root with the finder
inhibited.

### [CHANGED] Context call (`modules/project/04-projects.el:111-136`)
`imoogi-project-context-call` exposes its context root to the finder for the
duration of the call (context-call clause). `imoogi-project-find-file`
resolves PROJECT by route (a) and calls `project-find-file-in` (plan.md D-4).

### [CHANGED] Project transient (`modules/general/05-transient.el`)
`b` invokes an Imoogi buffer command (REQ-PIN-010); `f` keeps
`imoogi-project-find-file`, whose candidate list gains the notes files.

### [CHANGED] Project notes (`modules/project/26-project-notes.el`)
Supplies the notes-context predicate and the notes file/buffer lists for a
root. `--current-entry` is unchanged in behaviour (REQ-PIN-014).

### [UNCHANGED] `project-files`, `project-buffers`, `project-kill-buffers`, Treemacs
No method, advice, or hook on these.

### [NEW / CHANGED] Tests
New `tests/project-perspective-pin-test.el`. Existing tests receive exactly
three edits (plan.md § A.4), and no others:
- **E1** — `tests/transient-menu-test.el:80`: the `b` row.
- **E2** — `tests/workspace-bridge-test.el:141` and `:160`: the
  `project-find-file` stub moves to `project-find-file-in`; assertions
  unchanged.
- **E3** — `tests/workspace-bridge-test.el:19-20`: the transient `project-root`
  override becomes `(if (consp (cdr project)) (cadr project) (cdr project))`.

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
| AC-PIN-018 | REQ-PIN-001 |
