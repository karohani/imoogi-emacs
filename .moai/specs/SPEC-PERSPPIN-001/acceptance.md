# Acceptance Criteria — SPEC-PERSPPIN-001

Eighteen criteria, `AC-PIN-001`..`AC-PIN-018`, contiguous. Every requirement in
`spec.md` § 3 is covered by at least one, and each criterion names the ERT test
that decides it. All tests live in `tests/project-perspective-pin-test.el`
unless stated otherwise and run through:

```bash
EMACS=/Applications/Emacs.app/Contents/MacOS/Emacs IMOOGI_ERT_SELECTOR='imoogi-persppin-' ./tests/run.sh
```

## Sandbox

Every test runs inside one macro (plan.md § C.2) that:

- creates a fresh temp directory and roots `user-emacs-directory`,
  `project-list-file`, `imoogi-project-notes-directory`, the notes registry, the
  mounted-roots file, and `org-id-locations-file` under it;
- let-binds `imoogi-project-perspective-alist` to nil, `persp-mode` to t, and
  stubs `persp-current-name` with a test-controlled name (as t34 did);
- let-binds `project-find-functions` to a fresh list built by the module's own
  installer, so a test may prepend or append fakes; the hook's depth record
  (`hook--depth-alist`) is a symbol property and outlives the `let`, which is
  harmless for ordering;
- resets the finder memo (plan.md D-8) and the notes-directory list (D-7);
- stubs `project-prompter`, `read-directory-name`, and
  `imoogi-project-notes--select-entry` to signal `"prompted"`, so reaching any
  prompt fails the test;
- stubs `compile` with an **interactive** lambda that records
  `default-directory`, because `project-compile` uses `call-interactively`;
- kills every buffer visiting a sandbox file and deletes the sandbox.

The test file declares `(defvar eglot-lsp-context)` at top level so its `let`
binding is dynamic under `lexical-binding: t`.

"Projectless buffer" in the scenarios means a `with-temp-buffer` (or a file
buffer) whose `default-directory` is a local sandbox directory that is not under
VC, not inside the mapped root, and not inside any notes directory, and which
has none of the notes buffer-local variables set — so it is not a notes-context
buffer (spec.md § 2).

## AC Matrix

### AC-PIN-001 — Scratch resolves to the Perspective's project (REQ-PIN-001)
**Given** root R (not under VC) mapped to Perspective "src", which is current,
**When** `(project-current nil)` is evaluated in a projectless buffer,
**Then** it returns a project whose `project-root` is R canonicalised, and
`(project-current t)` returns the same without prompting.
Test: `imoogi-persppin-scratch-resolves-perspective-project`.

### AC-PIN-002 — A VC root resolves to its VC project (REQ-PIN-001)
**Given** R initialised with `git init` and mapped and current,
**When** `(project-current nil)` runs in a projectless buffer,
**Then** the result is `equal` to `(project-current nil R)` evaluated with the
finder inhibited, and its car is `vc`.
Test: `imoogi-persppin-vc-root-resolves-vc-project` (`skip-unless (executable-find "git")`).

### AC-PIN-003 — The finder sorts after every depth ≤ 90 member (REQ-PIN-002)
**Given** — unlike the other tests, which take the list the Sandbox builds —
this test re-binds `project-find-functions` to `(project-try-vc)`, then adds a
fake finder F0 with `add-hook … t` (APPEND, depth 90), **then** calls the
installer (depth 95), then adds a fake F1
added with plain `add-hook` (depth 0) and a fake F2 added with `add-hook … t`
(APPEND, depth 90), each fake returning `(transient . OTHER)` only for its own
directory,
**When** the order of `project-find-functions` is read,
**Then** the Perspective finder is the last element (after `project-try-vc`, F0,
F1 and F2); and in a buffer whose directory F2 claims, `(project-current nil)`
returns F2's `(transient . OTHER)`, not R's project.
Test: `imoogi-persppin-finder-is-last-member`.

### AC-PIN-004 — A buffer of another project keeps it (REQ-PIN-002)
**Given** R mapped and current, and a second directory B initialised with
`git init` (unregistered in `project--list`),
**When** `(project-current nil)` and `C-c h p c` (with `compile` stubbed to
record `default-directory`) run in a buffer whose `default-directory` is B,
**Then** the project root is B and the recorded directory is B.
Test: `imoogi-persppin-own-vc-project-beats-perspective` (`skip-unless git`).

### AC-PIN-005 — Unmapped, off, or stale: behaviour unchanged (REQ-PIN-003)
**Given** each of: (a) current Perspective "main" with no mapping; (b)
`persp-mode` nil and `persp-current-name` stubbed to signal; (c) a mapping whose
root has been deleted,
**When** `(project-current nil)` runs in a projectless buffer,
**Then** it returns nil without error, and `(project-current t)` reaches the
stubbed prompter (the test asserts the `"prompted"` error).
Test: `imoogi-persppin-unmapped-off-or-stale-declines`.

### AC-PIN-006 — Explicit lookups are not claimed (REQ-PIN-004)
**Given** R mapped and current, a projectless buffer with `default-directory`
D, and a non-VC directory E,
**When** (a) `(project-current nil E)` runs; (b) `(project-current nil)` runs
under `project-current-directory-override` bound to E; (c)
`imoogi-project-switch-perspective` runs with `project-prompter` stubbed to
return D (`persp-switch`, `project-dired`, and Treemacs stubbed),
**Then** (a) and (b) return nil, never R's project; and (c) remembers and
switches to the Perspective of D, not of R.
Test: `imoogi-persppin-explicit-lookups-not-claimed`.

### AC-PIN-007 — Eglot context is declined (REQ-PIN-005)
**Given** R mapped and current,
**When** `(project-current nil)` runs in a projectless buffer with
`eglot-lsp-context` dynamically bound to t (the test file `defvar`s it),
**Then** it returns nil; and with the binding back at nil, the same call
returns R's project (so the decline is attributable to the binding).
Test: `imoogi-persppin-eglot-context-declines`.

### AC-PIN-008 — Remote names cause no file I/O and are not claimed (REQ-PIN-006)
**Given** `file-directory-p`, `file-truename`, and `file-exists-p` stubbed to
signal when called with a remote name, and either (a) a mapping
`"/ssh:host:/srv/app/"` → "remote" with "remote" current, or (b) local R mapped
and current with a buffer whose `default-directory` is `"/ssh:host:/tmp/"`,
**When** `(project-current nil)` runs in (a) a projectless buffer and (b) that
remote-directory buffer, with `project-try-vc` removed from the test's
`project-find-functions` so only the finder sees the remote directory,
**Then** both return nil and no stub signalled.
Test: `imoogi-persppin-remote-root-no-file-io`.

### AC-PIN-009 — The finder never signals (REQ-PIN-007)
**Given** R mapped and current, `persp-mode` t, and `persp-current-name`
stubbed to count its calls and signal an `error` (it is on the finder's path for
every implicit lookup in a projectless buffer, plan.md D-2 guard 5),
**When** `(project-current nil)` and `(imoogi-tab-name)` run in a projectless
buffer,
**Then** both return without error, the project result is nil, and the stub's
call count is at least 1 (the error was actually raised inside the finder, so
the test cannot pass vacuously).
Test: `imoogi-persppin-finder-never-signals`.

### AC-PIN-010 — Notes-context buffers are declined (REQ-PIN-008)
**Given** R mapped and current with a notes entry, and the notes file
`project.org` of that entry visited (its directory is not under VC),
**When** `(project-current nil)` and
`(imoogi-project-notes--current-source-root entry)` run in that buffer,
**Then** `project-current` returns nil and the source root equals the value the
same call returns with the finder removed from `project-find-functions`.

**And** given the notes-context list already built, and
`imoogi-project-notes--all-entries`, `imoogi-project-notes--read-registry`, and
`imoogi-project-notes--find-metadata-file` stubbed to signal,
**When** `(project-current nil)` runs both in that notes buffer and in a
projectless non-notes file buffer,
**Then** no stub signals (the hot path reads no registry or metadata), the notes
buffer still yields nil, and the non-notes buffer still yields R's project.
**And** (refresh triggers, plan.md D-7) a table-driven test wraps the D-7
rebuild function with a call-counting spy (the spy calls the original). It runs
one row per trigger, with that trigger's I/O pointed into the sandbox or
stubbed:

| Row | Trigger invoked | I/O handling |
|-----|-----------------|--------------|
| 1 | `imoogi-project-notes--write-registry` with an entry for new notes dir N2 | sandbox registry file |
| 2 | `imoogi-project-notes--write-mounted-state` with a valid state | sandbox mounted-roots file |
| 3 | `imoogi-project-notes-mounted-root-refresh` | `message` stubbed |
| 4 | `imoogi-project-notes-detach` with a mounted entry argument | none (memory only) |
| 5 | `imoogi-project-notes-unmount-device` on a sandbox root | `imoogi-project-notes-unmount-preflight-function` bound to a stub returning a descriptor with **both** `:supported t` and `:mount-point` set to an existing sandbox directory (the command calls `--buffers-under-directory` → `file-truename` on it, `26-project-notes.el:3929`, :3890-3898); `-executor-function` → t; `y-or-n-p` → t |
| 6 | the named D-7 `persp-switch-hook` function, **called directly** (not via `run-hooks`, so the Treemacs hook members do not run) | none |

**Then** each row increments the spy count by at least 1. After row 1, and
again after row 6 with N2 created on disk under a sandbox mounted root, a buffer
visiting a file in N2 is a notes-context buffer (`project-current` yields nil),
with no registry read during that lookup.

Trigger 7 (startup) cannot run in the batch runner (`tests/run.el` exits inside
`-l`, so `emacs-startup-hook` never fires). It is verified **by registration
only**: its function is a member of `emacs-startup-hook` and sorts after
`imoogi-perspective-state-restore`.
Tests: `imoogi-persppin-notes-buffer-declined`,
`imoogi-persppin-notes-check-reads-no-registry`,
`imoogi-persppin-notes-list-refresh-triggers`,
`imoogi-persppin-notes-list-startup-registered`.

### AC-PIN-011 — `b`, `c`, `k` do not prompt from scratch (REQ-PIN-009)
**Given** R mapped and current; a live buffer P1 visiting `R/a.txt` (so `k`
has a buffer to offer); a projectless buffer from which the commands are
invoked; `compile` stubbed with an interactive lambda recording
`default-directory`; `yes-or-no-p` stubbed to record its prompt and return nil;
and the `b` reader's completion stubbed to return the first candidate,
**When** the commands bound to `b`, `c`, and `k` in `imoogi-transient-project`
are called interactively,
**Then** none reaches the prompter; `c` records R; the `yes-or-no-p` stub was
called exactly once and its prompt contains `(project-name PR)` for R's project
PR (and P1 is still live, since the stub answered nil); and `b` returns a buffer
from AC-PIN-012's set.
Test: `imoogi-persppin-transient-b-c-k-without-prompt`.

### AC-PIN-012 — `b` lists project buffers plus own notes buffers only (REQ-PIN-010)
**Given** R mapped and current with a notes entry N; live buffers: P1 visiting
`R/a.txt`, a Dired buffer on R, X visiting a file in an unrelated directory,
N1 visiting `N/project.org`, and M1 visiting a file of a second project's notes
directory,
**When** the `b` command's candidate collection is captured from a projectless
buffer,
**Then** it contains exactly P1, the Dired buffer, and N1 — not X, not M1, not
`*scratch*`, not the invoking buffer unless it belongs to R.
Test: `imoogi-persppin-switch-buffer-candidates`.

### AC-PIN-013 — `f` lists project files plus own notes files only (REQ-PIN-011)
**Given** R mapped and current with files `a.txt`, `sub/b.txt`; its notes
directory N with `project.org`, `tasks.org`, and a `.git/config`; a second
project S with its own notes directory,
**When** `f` runs with `project-read-file-name-function` let-bound to a stub
that records its ALL-FILES argument and returns the absolute path
`N/project.org`. ALL-FILES is absolute for any number of dirs, because `f` does
not bind `project-files-relative-names` (plan.md D-4; project.el L395-399).
Each recorded element is passed through `expand-file-name`
against R, which leaves an absolute name unchanged, so the comparison is on
absolute names and does not depend on how the reader displays candidates.
**Then** the recorded set is exactly {`R/a.txt`, `R/sub/b.txt`,
`N/project.org`, `N/tasks.org`}: no path under `N/.git/` and no file of S's
notes. The current buffer afterwards visits `N/project.org`. Run again from
Perspective "s" mapped to S, the recorded set contains no file of N.
Test: `imoogi-persppin-find-file-candidates`.

### AC-PIN-014 — Notes stay out of the project elsewhere (REQ-PIN-012)
**Given** R mapped and current with N1 visiting a notes file of R,
**When** `(project-files pr)`, `(project-buffers pr)`, and
`project-kill-buffers` with `no-confirm` run for R's project,
**Then** no notes path is in the files, N1 is not in the buffers, and N1 is
still live after the kill.
Test: `imoogi-persppin-notes-not-project-members`.

### AC-PIN-015 — The other transient entries keep their resolution (REQ-PIN-013)
**Given** the transient definition,
**When** the suffix commands of `imoogi-transient-project` are read,
**Then** `p`, `f`, `s`, `d`, `m`, `c`, `k`, `F` and the Perspective entries are
bound to the same commands as in `6feee71` (only `b` changes).
`tests/workspace-bridge-test.el`
`imoogi-project-context-root-follows-perspective-mapping` (:141) and
`imoogi-project-context-falls-back-to-project-el` (:160) pass after exactly the
plan.md A.4 edits E2 (stub move `project-find-file` → `project-find-file-in`
in both tests) and E3 (shape-tolerant `project-root` override at :19-20), with
unchanged assertions. Existing tests receive no edit beyond E1-E3.
`project-forget-project` invoked interactively reaches the prompter.

**And** given R not under VC, mapped and current, the incidental changes hold
exactly as scoped (REQ-PIN-013.1, .3):
- `imoogi-project-dired` from a projectless buffer reaches no prompter; with
  `dired` stubbed to record its directory, the recorded directory is R.
- From a **notes-context** buffer of R's notes directory, all of the
  following reach no prompter and use R (the context-call clause and route
  (a)):
  - `imoogi-project-dired`, with `dired` stubbed;
  - `imoogi-project-find-regexp`, with `project--read-regexp` stubbed to
    return a fixed regexp (its interactive spec would otherwise call
    `read-regexp`) and `xref-show-xrefs` stubbed to record its call, with
    `project-files` stubbed to return `(R/a.txt)` so no grep or find process
    runs; R is checked from the project passed to `project-files`;
  - `imoogi-project-find-file`, with the AC-PIN-013 stub on
    `project-read-file-name-function`.
- With **no** context root (current Perspective "main" unmapped, projectless
  buffer), both `imoogi-project-find-file` and `imoogi-project-dired` reach
  the prompter (the `"prompted"` error), so the fallback is gated for the
  route (a) path and the context-call path alike (REQ-PIN-003).
- `(imoogi-project-notes--project-root)` from a projectless buffer returns R
  without calling the `read-directory-name` stub.
Tests: `imoogi-persppin-transient-other-entries-unchanged`,
`imoogi-persppin-incidental-prompt-removals`, plus the two named
workspace-bridge tests.

### AC-PIN-016 — t34 outcomes unchanged (REQ-PIN-014)
**Given** the t34 test file `tests/project-notes-perspective-test.el`
unmodified,
**When** it runs with the finder installed,
**Then** all four tests pass; and a new test shows that with the finder
inhibited, scratch in a pinned Perspective still resolves through the retained
fallback.
Command:
```bash
EMACS=/Applications/Emacs.app/Contents/MacOS/Emacs IMOOGI_ERT_SELECTOR='\(imoogi-project-notes-\(scratch-resolves\|unmapped-perspective\|buffer-project-beats\|unregistered-project\)\|imoogi-persppin-t34-fallback-retained\)' ./tests/run.sh
```

### AC-PIN-017 — No notes entry, or a remote root: project-only lists (REQ-PIN-015)
**Given** R mapped and current with no notes entry,
**When** the `b` and `f` candidate collections are captured,
**Then** they equal the project-only sets of AC-PIN-012 / AC-PIN-013 and no
message or error is emitted; and with the notes identity-key function stubbed
to signal `user-error` (the remote refusal), the same holds.
Test: `imoogi-persppin-no-notes-entry-project-only`.

### AC-PIN-018 — The finder memoizes per pinned root (REQ-PIN-001, plan.md D-8)
**Given** R not under VC, mapped to "src", "src" current, and `project-try-vc`
wrapped with a spy that calls the original and counts calls whose argument
satisfies `(file-equal-p ARG R)`. The finder passes a canonical root (truename
with a trailing slash), so the spy must not compare strings. Lookups of the
projectless buffer's own directory are not counted.
**When** `(project-current nil)` runs twice in a projectless buffer,
**Then** both return the same project rooted at R and the R-count is 1 (the
second lookup is a memo hit and does not search again).
**And when** the D-8 clear function on `persp-switch-hook` is called directly
and `(project-current nil)` runs again, **Then** the R-count is 2 (miss after
clear).
**And when** the alist entry for "src" is changed to a second non-VC root R2
(no Perspective switch) and `(project-current nil)` runs, **Then** the result
is rooted at R2 (implicit invalidation by key).
Test: `imoogi-persppin-finder-memo`.

## Edge Cases

- `~/notes` becomes a git repository: scratch then has an own project and the
  finder never runs for it (REQ-PIN-002). Covered by AC-PIN-004's mechanism.
- Two alist entries carrying the same Perspective name after a savehist
  restore: the first match wins, as `imoogi-project--perspective-root` already
  does. Not asserted; recorded in plan.md § B risks.
- The invoking buffer is itself a project buffer: the `b` default candidate is
  `other-buffer`, per `project.el`; no change asserted.
- A worktree of R opened in its own Perspective: identity keys are shared, so
  its notes entry is R's — `f`/`b` from the worktree Perspective list the same
  notes. Accepted.

## Quality Gate Criteria

- `./tests/run.sh` with `IMOOGI_ERT_SELECTOR='\(project-notes\|transient\|imoogi-project-\|perspective\|persppin\|workspace\)'`
  exits 0, unexpected = 0 (skips limited to the pre-existing `cli-cache-*`).
- Full `./tests/run.sh` offline boot stage passes (no boot-health warnings).
- `check-parens` stage passes.
- No test writes outside its sandbox (verified by the sandbox macro's cleanup
  and by `project-list-file` binding).

## Definition of Done

- All eighteen ACs pass with their named tests.
- The four t34 tests pass unmodified.
- Edits to existing tests are exactly these three, per plan.md § A.4, and no
  others:
  - E1: the `b` row in `tests/transient-menu-test.el:80`;
  - E2: the `project-find-file` → `project-find-file-in` stub move in
    `tests/workspace-bridge-test.el:141` and `:160`, assertions unchanged;
  - E3: the shape-tolerant transient `project-root` override at
    `tests/workspace-bridge-test.el:19-20`.
- `progress.md` §E.2/§E.3 populated by the run phase with the commands and
  exit codes above.

## Residual Risk

- Known limitation — Decided (lane-3, 2026-10-13): `C-c h p c` and `k` still
  prompt when invoked from a project-notes buffer, because the finder declines
  notes buffers (REQ-PIN-008, plan.md D-6). Not an AC failure; out of scope.
- Decided (lane-3, 2026-10-13): the `f`/`s`/`d` vs `b`/`c`/`k` precedence
  asymmetry is kept as written (plan.md D-1); AC-PIN-004 and AC-PIN-015 assert
  each side as it stands.
- Study-note Perspectives (consequence of D-6, accepted; spec.md § 5): the
  Perspective is mapped to the notes directory itself, so every note file of a
  study project is a notes-context buffer. From every one of them,
  `C-c h p c`, `C-c h p k`, and the generic `project.el` commands outside
  `C-c h p` that resolve with `(project-current t)` (e.g. `C-x p c/k/f/g/d/b`)
  still prompt. `C-c h p b`, `f`, `s`, `d` do not. Only projectless buffers
  (scratch, help) are pinned there for every command.
- Finder memo (plan.md D-8): `git init` inside a pinned non-VC root keeps the
  transient project until the next Perspective switch.
- Finder memo thrash (O-f, accepted): with two frames showing different pinned
  Perspectives, alternating tab-bar redisplays alternate the memo key, so the
  one-slot memo misses on every alternation and repeats the VC search for a
  non-VC root. This costs time only, never correctness. A small alist keyed the
  same way is a run-phase option.
- Interactive driving of the real `C-c h p` key sequence is not automated; the
  ACs call the bound commands directly.
- `imoogi-tab-name` and `imoogi-org-preview--allowed-roots` change for
  projectless buffers in a mapped Perspective (plan.md § B.3); asserted only for
  "does not signal".
