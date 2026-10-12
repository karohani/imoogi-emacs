# Acceptance Criteria — SPEC-PERSPPIN-001

Seventeen criteria, `AC-PIN-001`..`AC-PIN-017`, contiguous. Every requirement in
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
  installer, so a test may prepend or append fakes without touching the global;
- stubs `project-prompter`, `read-directory-name`, and
  `imoogi-project-notes--select-entry` to signal `"prompted"`, so reaching any
  prompt fails the test;
- kills every buffer visiting a sandbox file and deletes the sandbox.

"Projectless buffer" in the scenarios means a `with-temp-buffer` (or a file
buffer) whose `default-directory` is a sandbox directory that is not under VC
and not inside the mapped root.

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

### AC-PIN-003 — The finder is last, even after later additions (REQ-PIN-002)
**Given** the installer has run, and a fake finder returning
`(transient . OTHER)` is then added with plain `add-hook` (depth 0) and another
with `add-hook … t` (append),
**When** the order of `project-find-functions` is read,
**Then** the Perspective finder is the last element; and in a buffer whose
directory the fakes claim, `(project-current nil)` returns `(transient . OTHER)`.
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
**Then** (a) and (b) return nil (or a transient project for E when the caller
prompts), never R; and (c) remembers and switches to the Perspective of D, not
of R.
Test: `imoogi-persppin-explicit-lookups-not-claimed`.

### AC-PIN-007 — Eglot context is declined (REQ-PIN-005)
**Given** R mapped and current,
**When** `(project-current nil)` runs in a projectless buffer with
`eglot-lsp-context` bound to t,
**Then** it returns nil.
Test: `imoogi-persppin-eglot-context-declines`.

### AC-PIN-008 — Remote roots cause no file I/O (REQ-PIN-006)
**Given** a mapping `"/ssh:host:/srv/app/"` → "remote" and "remote" current,
with `file-directory-p`, `file-truename`, and `file-exists-p` stubbed to signal
when called with a remote name,
**When** `(project-current nil)` runs in a projectless buffer,
**Then** it returns nil and no stub signalled.
Test: `imoogi-persppin-remote-root-no-file-io`.

### AC-PIN-009 — The finder never signals (REQ-PIN-007)
**Given** R mapped and current and `imoogi-project--canonical-root` stubbed to
signal an `error`,
**When** `(project-current nil)` and `(imoogi-tab-name)` run in a projectless
buffer,
**Then** both return without error and the project result is nil.
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
**And** given the list built with only N, when a second notes directory N2 is
registered through `imoogi-project-notes--write-registry` (and, separately,
when `persp-switch-hook` runs after N2 appears on disk under a mounted root),
**Then** a buffer visiting a file in N2 is immediately treated as a
notes-context buffer (`project-current` yields nil), with no registry read on
the lookup itself.
Tests: `imoogi-persppin-notes-buffer-declined`,
`imoogi-persppin-notes-check-reads-no-registry`,
`imoogi-persppin-notes-list-refresh-triggers`.

### AC-PIN-011 — `b`, `c`, `k` do not prompt from scratch (REQ-PIN-009)
**Given** R mapped and current, a projectless buffer, `compile` stubbed to
record `default-directory`, `yes-or-no-p` stubbed to return nil, and the `b`
reader's completion stubbed to return the first candidate,
**When** the commands bound to `b`, `c`, and `k` in `imoogi-transient-project`
are called interactively,
**Then** none reaches the prompter, `c` records R, `k` asks about R's project
name, and `b` returns a buffer from AC-PIN-012's set.
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
**When** the `f` command's completion candidates are captured and each is
expanded to an absolute file name against R (the comparison is on that absolute
set, so it does not depend on how a candidate outside R is displayed),
**Then** the set is exactly {`R/a.txt`, `R/sub/b.txt`, `N/project.org`,
`N/tasks.org`}: no path under `N/.git/` and no file of S's notes; selecting the
candidate that expands to `N/project.org` visits that file; and run from
Perspective "s" mapped to S, the set contains no file of N.
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
bound to the same commands as in `6feee71` (only `b` changes), and
`tests/workspace-bridge-test.el` `imoogi-project-context-root-follows-perspective-mapping`
and `imoogi-project-context-falls-back-to-project-el` pass; and
`project-forget-project` invoked interactively reaches the prompter.
Tests: `imoogi-persppin-transient-other-entries-unchanged`, plus the two named
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

- All seventeen ACs pass with their named tests.
- The four t34 tests pass unmodified.
- The amended `b` row in `tests/transient-menu-test.el` is the only edit to an
  existing assertion, plus any `f` stub change named in plan.md § A.4.
- `progress.md` §E.2/§E.3 populated by the run phase with the commands and
  exit codes above.

## Residual Risk

- Known limitation — Decided (lane-3, 2026-10-13): `C-c h p c` and `k` still
  prompt when invoked from a project-notes buffer, because the finder declines
  notes buffers (REQ-PIN-008, plan.md D-6). Not an AC failure; out of scope.
- Decided (lane-3, 2026-10-13): the `f`/`s`/`d` vs `b`/`c`/`k` precedence
  asymmetry is kept as written (plan.md D-1); AC-PIN-004 and AC-PIN-015 assert
  each side as it stands.

- Interactive driving of the real `C-c h p` key sequence is not automated; the
  ACs call the bound commands directly.
- `imoogi-tab-name` and `imoogi-org-preview--allowed-roots` change for
  projectless buffers in a mapped Perspective (plan.md § B.3); asserted only for
  "does not signal".
