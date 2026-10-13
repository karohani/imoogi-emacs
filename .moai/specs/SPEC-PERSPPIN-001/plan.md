# Implementation Plan — SPEC-PERSPPIN-001

Card t35 (Class C, feature). Tier M: `spec.md`, `plan.md`, `acceptance.md`,
`progress.md`. Development mode: TDD (RED first per AC). Baseline `6feee71`
(contains t34 `4285b31`). Work happens only in `.moai/worktrees/t35`
(branch `WT-persp-full-pin`).

## § A Context

### A.1 What already exists

- `imoogi-project-perspective-alist` (`04-projects.el:9`): canonical root →
  Perspective name, written by `imoogi-project-perspective-name`, persisted by
  savehist (`00-defaults.el:124`).
- `imoogi-project--perspective-root` (`04-projects.el:81`): name → root, with
  `file-directory-p` then `file-truename` — note it does file I/O before any
  remote check, so the finder must not call it unguarded (REQ-PIN-006).
- `imoogi-project-context-root` (`04-projects.el:93`): Perspective-first, then
  `project-current nil`. Used by `f`/`s`/`d` via `imoogi-project-context-call`.
- `imoogi-treemacs--perspective-project-root` (`07-treemacs.el:554`): its own
  `rassoc` fallback; untouched.
- t34 fallback in `imoogi-project-notes--current-entry`
  (`26-project-notes.el` ~1379): only when `project-current` is nil.
- `project.el` 0.11.1 facts used here: `project-find-functions` is a run-until-
  success hook; `project--find-in-directory` absorbs only `permission-denied`;
  `project-current` passes the prompted directory back through the same hook;
  `project-try-vc` caches **positive** results only (a non-VC directory is
  re-searched on every call); `project--read-project-buffer` filters
  by `project-buffers`; `project-forget-project` calls `project-prompter`
  directly; Eglot binds `eglot-lsp-context` to t around its lookup.
- `perspective.el`: `persp-current-name` → `persp-curr` → `persp-mode-guard`,
  which signals when `persp-mode` is off.
- `~/project-notes` is not a git repository (checked 2026-10-13: no `.git`), so
  notes buffers are projectless today and would reach the finder without
  REQ-PIN-008.

### A.2 Other `project-current` callers affected by the finder

| Caller | Effect in a mapped Perspective, projectless buffer | Disposition |
|--------|------------------------------------------------------|-------------|
| `imoogi-tab-name` (`22-tabs.el:54`) | tab shows the project name instead of the buffer name | accepted (consistent with pinning) |
| `imoogi-org-preview--project-root` (`23-org-preview.el:428`) | `~/notes/scratch.org` preview allowed roots widen to include R | accepted, recorded risk B.3 |
| `imoogi-project-notes--project-root` (`26-project-notes.el:527`) | setup from scratch stops prompting for the source folder and uses R | accepted (the Perspective answers it) |
| `imoogi-project-notes--current-entry` | resolves via `project-current` branch instead of the t34 fallback; same entry | REQ-PIN-014 |
| `imoogi-project-notes--current-source-root` | notes buffers: unchanged because of REQ-PIN-008 | guarded |
| `imoogi-treemacs--perspective-project-root` | first branch now answers; same root | no change needed |
| Eglot | declined via `eglot-lsp-context` | REQ-PIN-005 |
| `consult--default-project-function` (`C-x p b`) | stops prompting from scratch | accepted; notes union out of scope |

### A.3 PRESERVE list

- `tests/project-notes-perspective-test.el` (4 tests) — unmodified.
- `tests/workspace-bridge-test.el` switch-perspective tests (unmodified) and the
  two context tests (assertions unmodified; stub moved per A.4).
- `tests/transient-menu-test.el` rows other than `b`.
- `imoogi-project-context-root` semantics (Perspective-first).

### A.4 Explicitly amendable tests

**Decided (lane-3, 2026-10-13): the complete set of permitted edits to existing
tests is E1-E3 below.** Every other place that lists permitted edits (§ E,
spec.md § 7, acceptance.md AC-PIN-015 and DoD) names exactly this set. No
other line of an existing test changes.

- **E1** — `tests/transient-menu-test.el:80` row
  `(project-switch-to-buffer transient--do-exit)` becomes the new `b` command
  with `transient--do-exit`.
- **E2** — `tests/workspace-bridge-test.el:141`
  (`imoogi-project-context-root-follows-perspective-mapping`) and `:160`
  (`imoogi-project-context-falls-back-to-project-el`) stub `project-find-file`.
  D-4 makes `f` call `project-find-file-in`, so in both tests the stub **moves
  to `project-find-file-in`** (a lambda accepting
  `(suggested dirs project &optional include-all)` that records
  `default-directory`). Assertions unchanged: the recorded directory equals the
  context root (test :141, before and after the Perspective switch) and the
  canonical root (test :160). Test :141 needs no `project-current` stub because
  D-4's resolution falls back to `(transient . CONTEXT-ROOT)` when the context
  root is non-nil. Test :160 keeps its existing `project-current` stub.
- **E3** — `tests/workspace-bridge-test.el:19-20`, the top-level
  `(cl-defmethod project-root ((project (head transient))) (cadr project))`,
  becomes shape-tolerant:
  `(cl-defmethod project-root ((project (head transient))) (if (consp (cdr project)) (cadr project) (cdr project)))`.
  Why it is needed: `tests/run.el:42-44` loads every `*-test.el` before any
  selector runs, so this override replaces `project.el`'s `(cdr project)`
  method for the whole ERT session. Every `(transient . ROOT)` project — the
  shape the finder returns for a non-VC root (D-2), the shape D-4's fallback
  builds, and the shape `project-current t` builds — then signals
  `(wrong-type-argument listp ROOT)` in `project-root`. That would fail
  AC-PIN-001, AC-PIN-015's "And" clause, the non-VC `c`/`k`/`f` ACs, and test
  :141 under E2. The shape-tolerant form keeps the file's list-form
  `imoogi-test--project` working, because its `cdr` is a cons.

## § B Design Decisions

Ordered by change likelihood (highest first).

### D-1 — Precedence asymmetry in the transient is preserved, not unified

`f`/`s`/`d` are Perspective-first (existing behaviour of
`imoogi-project-context-root`, stated in its docstring; the workspace-bridge
tests stub that function, so they pin "follows the context root", not the
precedence inside it); `b`/`c`/`k` follow
`project.el` (own project first, Perspective finder last). In Perspective A, a
buffer of project B gets A's files from `f` and B's buffers from `b`.
Rationale: unifying in either direction regresses something — making `f`
own-project-first breaks the tested "Perspective survives buffer restoration"
behaviour; making `b`/`c`/`k` Perspective-first contradicts the t34
card-review lesson the card itself carries. This card's goal (no prompt from
projectless buffers, pinned candidate lists) is met without choosing.

**Decided (lane-3, 2026-10-13):** keep the asymmetry as written — `f`/`s`/`d`
stay Perspective-first, `b`/`c`/`k` stay own-project-first. Why: unifying in
either direction regresses a tested behaviour or t34's card-review lesson, and
the card's goal (no prompt from projectless buffers, pinned candidate lists) is
met without unifying. Unification is out of scope (spec.md § 5).

### D-2 — Mechanism: a guarded finder at the end of `project-find-functions`

Chosen over (a) a custom `project-prompter` — rejected because
`imoogi-project-switch-perspective`, `project-switch-project`, and
`project-forget-project` call the prompter directly, so a prompter that answers
with the Perspective root would silently select/forget the current project;
and (b) per-command wrappers binding `project-current-directory-override` —
rejected as the primary mechanism because it covers only the commands wrapped
and leaves `C-x p *`, consult, and the tab name prompting/unpinned. The finder
is installed with `add-hook` at **depth 95**, strictly above 90. `add-hook … t`
(APPEND) means depth 90, and at equal positive depth a newer function sorts
*after* an older one (subr.el `add-hook`). So depth 90 would let a later APPEND
land after the finder, while 95 keeps every later default-depth or APPEND
member before it (AC-PIN-003). 95 rather than 100 follows the `add-hook`
docstring's advice against 100 and leaves room above for a deliberately-last
member. Members added later at a depth above 95 are not covered (spec.md
REQ-PIN-002).

Guards, evaluated cheapest first, each a decline:
1. inhibit variable bound (explicit selection) or re-entry; or the
   explicit-lookup marker `imoogi-project--explicit-lookup` set. Decided
   (lane-3, 2026-10-13, review rework CR1): `project-current` replaces an
   omitted DIRECTORY with `default-directory` before it runs the finders. So an
   `:around` advice on `project-current`, `imoogi-project--mark-explicit-lookup`,
   binds the marker to `(and directory t)`; a nested implicit call binds it
   back to nil. `imoogi-project-install-perspective-finder` installs the
   advice together with the finder, and calling the installer again changes
   nothing. Without the marker, `(project-current nil default-directory)`
   would look implicit (spec.md REQ-PIN-004.3);
2. `(bound-and-true-p eglot-lsp-context)` — Eglot is `:commands`-deferred, so
   the variable may be unbound when the finder runs;
3. DIR is remote — `file-remote-p` on the raw string, checked **first** within
   this guard and before any `expand-file-name` (which would enter the TRAMP
   handler); then `project-current-directory-override` non-nil; then
   DIR ≠ the current buffer's `default-directory` (compared after
   `expand-file-name` + `file-name-as-directory`);

**Context-call clause (R2-D1, between guards 3 and 4).** While
`imoogi-project-context-call` is running (`04-projects.el:111-121`), it binds a
dynamic variable to its CONTEXT-ROOT (the `:root` of `imoogi-project-context`).
When that variable is non-nil and the expanded DIR equals it, the finder
returns `(transient . CONTEXT-ROOT)` and skips guards 4-8 and the memo.
Skipping guard 8 is the one stated exception in REQ-PIN-008. It is safe
because `f`/`s`/`d` never call `imoogi-project-notes--current-source-root`,
which is the redirection D-6 guards against. The
finder is reached only after every other member declined, so the root is not
under VC and transient is the right shape. Guards 1-3 still apply. When the
context root is nil (unmapped Perspective and no own project), the variable is
nil, the clause does nothing, and the command prompts as before (REQ-PIN-003).

4. `persp-mode` not active;
5. no alist entry for `persp-current-name`;
6. raw mapped root `file-remote-p`;
7. mapped root not `file-directory-p`;
8. notes-context buffer (predicate supplied by `26-project-notes.el` through a
   hook variable owned by `04-projects.el`, so 04 never requires 26). The
   predicate short-circuits on the buffer-local variables
   `imoogi-project-notes-source-root`, `imoogi-project-notes-directory-local`,
   `imoogi-project-notes-entry-instance-id`. For a file buffer it then
   compares the expanded `buffer-file-truename` against the in-memory list of
   truename notes directories (D-7), so a symlinked `~/project-notes` still
   matches. For a buffer visiting no file (Dired, compilation, shell) it
   compares the local `default-directory`, expanded but not resolved to a
   truename (no I/O), and skips a remote `default-directory` on the raw string
   (review rework, `26-project-notes.el` `--context-buffer-p`). A non-file
   buffer that reaches a notes directory through a symlinked path is therefore
   missed (spec.md § 5 known limitation). The predicate never calls
   `--all-entries`, `--read-registry`, or `--find-metadata-file` itself
   (AC-PIN-010, second part).
Then return the memoized project for the pinned root (D-8). On a memo miss,
resolve the canonical root with the finder inhibited:
`project--find-in-directory` on the root, else `(transient . ROOT)`. The whole
body is wrapped so any error declines (REQ-PIN-007).

### D-3 — `b` gets an Imoogi command; `project-buffers` is not advised

A global `project-buffers` advice would make `k` kill notes buffers and leak
notes into every consumer (REQ-PIN-012). The new `b` command resolves the
project (`project-current nil`, else the Perspective root as in t34, else the
normal prompt), builds the set {`project-buffers` minus ignore conditions} ∪
{live buffers visiting files under the root's notes directory}, and reads with
`read-buffer` using that predicate, then `switch-to-buffer`. The t34-style
fallback inside the command makes `b` work from notes-context buffers too.
The fallback (`imoogi-project--buffer-project`) carries the finder's remote
guards (review rework): it requires `persp-mode`, then declines when the
buffer's `default-directory` or the raw mapped root is remote. Both are
checked on the raw strings before any file operation. In that case `b` falls
through to `(project-current t)` and prompts (spec.md REQ-PIN-006,
acceptance.md AC-PIN-017).

### D-4 — `f` gains the notes directory as an extra search root

**Decided (audit r1 D4): one code path.** With `default-directory` at the
context root (as today, via `imoogi-project-context-call`), `f` calls
`project-find-file-in` with `dirs` = (root) or (root, notes-dir) when the root
has a notes entry, always through this single entry point (it no longer calls
`project-find-file`). `project-files` is untouched.

**Decided (lane-3, 2026-10-13), R2-D1 route (a): how `f` obtains PROJECT.**
`f` resolves

    (or (project-current nil)
        (and CONTEXT-ROOT (cons 'transient CONTEXT-ROOT))
        (project-current t))

where CONTEXT-ROOT is the `:root` already computed by
`imoogi-project-context-call`. The transient fallback applies **only** when
CONTEXT-ROOT is non-nil. With no context root (unmapped Perspective, no own
project), `f` reaches `(project-current t)` and prompts exactly as before
(REQ-PIN-003). This is what lets workspace-bridge test :141 pass with only E2
plus E3 (A.4).

**Which case holds for `s`/`d`: the same code path.** All three wrappers —
`imoogi-project-find-file`, `imoogi-project-find-regexp`, and
`imoogi-project-dired` (`04-projects.el:123-136`) — go through the single
helper `imoogi-project-context-call` (`04-projects.el:111-121`), which
let-binds `default-directory` to the context root and calls the command.
`project-find-regexp` (project.el L1004) and `project-dired` (L1327) resolve
with `(project-current t)` internally, so `s`/`d` cannot take route (a)'s `or`
form literally. They get the same resolution through D-2's context-call
clause: inside the helper, a lookup of CONTEXT-ROOT that no other finder
answers returns `(transient . CONTEXT-ROOT)`. That is the same result as
route (a)'s second branch, gated the same way. So `f`, `s` and `d` do not
prompt whenever a context root exists, from any buffer, including
notes-context buffers.

For `f`, route (a)'s middle branch and the context-call clause produce the same
project. `f` runs inside the helper, so `(project-current nil)` already reaches
the clause. The explicit branch is kept as the stated contract and is
deliberately redundant.

`f` must **not** bind `project-files-relative-names` (O-c). With it bound,
`project--files-in-directory` would return the notes files relative to the
notes directory, and they would then expand against R to wrong names. Unbound,
ALL-FILES is absolute for every dir count (project.el L395-399), which
AC-PIN-013 relies on. Notes files come in only
through the extra `dirs` element, and VCS metadata directories under the notes
dir are excluded because, for a directory outside the root,
`project--dir-ignores` falls back to the default `project-ignores`
(`vc-directory-exclusion-list` + `grep-find-ignored-files`), which
`project--files-in-directory` passes to `find`. The suggested filename keeps `project-find-file`'s list (buffer file
name, `thing-at-point 'filename`). Consequence: the existing tests get exactly
edits E2 and E3 (A.4).

Rejected: `project-or-external-find-file` with
`project-vc-external-roots-function`, because it only works for VC projects
(transient projects have no external-roots hook). Also rejected: keeping
`project-find-file` for the no-notes case, because two code paths would be
exercised by different tests for no gain.

### D-5 — t34 fallback retained

With the finder installed, the `project-current` branch of `--current-entry`
answers for scratch, so the fallback is normally unreachable. It stays as
defence in depth for inhibited/declined lookups and is pinned by a new test
(AC-PIN-016). Removing it would be a no-gain change to a just-reviewed function.

### D-6 — Notes-context buffers declined

Claiming them would redirect `--current-source-root` (which consults
`project-current` before the buffer-local source root) to the Perspective root,
which differs when the notes buffer was opened from a worktree. Cost: `c`/`k`
from a notes-context buffer still prompt (out of scope). `b` works there through
D-3's t34-style fallback. `f`, `s` and `d` work there whenever a context root
exists, through D-4 route (a) and D-2's context-call clause.

**Decided (lane-3, 2026-10-13):** accept the gap — `C-c h p c` and `k` still
prompt from a project-notes buffer, and the finder keeps declining notes
buffers. Why: claiming them would redirect `--current-source-root` for
worktree-opened notes, and the card names projectless buffers (scratch), not
notes buffers. Recorded as a known limitation (acceptance.md § Residual Risk)
and an out-of-scope item (spec.md § 5).

Consequence stated (audit r1 D5, does not reopen D-6): a study-note
Perspective (`imoogi-project-notes--open-study-workspace`,
`26-project-notes.el:2615`) is mapped to the notes directory itself. So every
note file of a study project is a notes-context buffer, and the finder declines
for all of them. Which commands still prompt there, given R2-D1 route (a)
(R2-D3, Decided (lane-3, 2026-10-13)):
- **Still prompt from every note file:** `C-c h p c` and `k`, and every generic
  `project.el` command outside `C-c h p` that resolves with
  `(project-current t)` (for example `C-x p c`, `C-x p k`, `C-x p f`, `C-x p g`,
  `C-x p d`, `C-x p b`/`consult-project-buffer`).
- **Do not prompt there:** `b` (D-3 fallback to the Perspective root) and
  `f`/`s`/`d` (the context root is the study root; D-4 route (a) and D-2's
  context-call clause).
- **Keep prompting by design everywhere:** `p` and `F`.

Only projectless buffers (scratch, help) are pinned in a study Perspective for
every command. Recorded in spec.md § 5 and acceptance.md § Residual Risk.

### D-7 — Notes-directory list: concrete refresh trigger points

**Decided (lane-3, 2026-10-13).** The in-memory list (canonical notes
directories of every entry `imoogi-project-notes--all-entries` returns) is
rebuilt eagerly, never lazily on the finder's path, at exactly these points,
all in `modules/project/26-project-notes.el`:

1. **Registry write** — at the end of `imoogi-project-notes--write-registry`
   (line ~792). Its callers at `6feee71` are `imoogi-project-notes--save-entry`
   (~1845, used by setup/setup-study/move/repair/copy) and
   `imoogi-project-notes--remove-local-entry` (~3701).
2. **Mounted-state write** — at the end of
   `imoogi-project-notes--write-mounted-state` (line ~863). Its callers are
   `imoogi-project-notes-raiseup` (~2775), `-reconnect-source` (~3068),
   `-clear-source-override` (~3084), `-mounted-root-add` (~3191),
   `-mounted-root-edit` (~3234), and `-mounted-root-remove` (~3248).
3. **Explicit rescan** — at the end of
   `imoogi-project-notes-mounted-root-refresh` (line ~3252), which rescans
   mounted roots without writing state.
4. **Session detach** — at the end of `imoogi-project-notes-detach` (line
   ~3818), which changes `imoogi-project-notes--detached-instance-ids` without
   writing a file.
5. **Unmount** — after the executor succeeds in
   `imoogi-project-notes-unmount-device` (line ~3920), which does not write
   state either.
6. **Perspective switch** — a function on `persp-switch-hook` (alongside the
   members added at `04-projects.el:339`; registered from 26 under
   `with-eval-after-load 'perspective`), which also picks up notes directories
   created or mounted outside Emacs.
7. **Startup** — once at startup, after `imoogi-perspective-state-restore`.
   **Decided (lane-3, 2026-10-13, run deviation):** the plan placed this on
   `emacs-startup-hook`. The implementation (commit `09e62b0`) registers
   `imoogi-project-notes--rebuild-context-directories` on **`window-setup-hook`**
   instead, with a plain `add-hook` at the end of `26-project-notes.el`. The
   reason is the existing test
   `imoogi-treemacs-does-not-auto-start-on-emacs-startup-hook`
   (`tests/workspace-bridge-test.el:523`), which pins `emacs-startup-hook` to
   contain only `imoogi-perspective-state-restore` (added at −50,
   `04-projects.el:548`). Emacs runs `window-setup-hook` right after
   `emacs-startup-hook`, so the rebuild still happens once at startup, after
   the restore.
8. **Rename rollback** — at the end of `imoogi-project-notes--rollback-rename`
   (review rework, commit `e3c225e`). A failed notes-directory rename restores
   the old directory, and without a rebuild the list would still name the
   renamed path.

Each rebuild calls `--all-entries` inside `condition-case`; on any error
(corrupt registry, unreadable mount) the previous list is kept and nothing is
displayed. Before the first rebuild (list never built), the predicate treats a
file as a notes file when it is inside `imoogi-project-notes-directory` — a
pure string-prefix check with no I/O. Rebuilds happen at the call sites (a
direct call at the end of each function in 1-5 and 8), not via advice. Trigger 6 is a
named function, so the test can call it directly without running the other
`persp-switch-hook` members (in the implementation it is
`imoogi-project-notes--refresh-context-directories`). Trigger 7 cannot run in
the batch runner (`tests/run.el` exits inside `-l`, so neither
`emacs-startup-hook` nor `window-setup-hook` fires). It is verified by
registration only (AC-PIN-010, third part): the rebuild is on
`window-setup-hook`, it is not on `emacs-startup-hook`, and
`imoogi-perspective-state-restore` is on `emacs-startup-hook`.

### D-8 — Memoize the finder's answer per pinned root

**Decided (audit r1 D6).** `project-try-vc` caches only positive results, so a
non-VC mapped root (the study-note case) would repeat a `locate-dominating-file`
walk and a `hack-dir-local-variables` pass on every tab-bar redisplay. The
finder therefore keeps a memo keyed per Perspective.

**Decided (lane-3, 2026-10-13, run deviation):** the plan said one slot. The
implementation keeps an alist, `imoogi-project--perspective-finder-memo`
(`04-projects.el:105`), of `(KEY . PROJECT)` entries, where KEY is
`(PERSPECTIVE-NAME . RAW-ROOT)` and RAW-ROOT is the string stored in
`imoogi-project-perspective-alist`:

- **Hit**: an entry for the current KEY exists. The guards still run first on
  every call, including the `file-directory-p` of guard 7, so a deleted root
  still declines.
- **Miss**: resolve as in D-2 (`imoogi-project-resolve-root`) and push a new
  entry.
- **Invalidation**: a remap or alist change produces a new KEY, so the old
  entry is simply never hit (implicit). `imoogi-project--perspective-finder-memo-clear`
  on `persp-switch-hook` (`04-projects.el:552`) empties the whole alist on
  every Perspective switch (explicit).

Accepted staleness: running `git init` in a pinned non-VC root keeps the
transient project until the next Perspective switch. Chosen over option (b)
(accepting the uncached search) because the finder runs on redisplay.

O-f resolved by the run deviation: with two frames showing different pinned
Perspectives, each frame's KEY has its own entry, so alternating tab-bar
redisplays hit the memo and no longer evict each other. Entries accumulate
only until the next Perspective switch clears the alist.

Pinned by AC-PIN-018 (O-e): for a non-VC root, two lookups run the VC search
once, and the D-8 clear function forces the next lookup to search again.

### B.3 Risks

- Stale mapping to a deleted root → declined by guard 7 (AC-PIN-005c).
- Remote root → declined by guard 6 before I/O; remote lookup directory (a
  TRAMP buffer in a local Perspective) → declined by guard 3 (AC-PIN-008).
- Study-note Perspectives: `C-c h p c`/`k` and generic `C-x p` commands prompt
  from every note file; `b`/`f`/`s`/`d` do not (D-6 consequence, accepted).
- Context-call clause scope (O3-c): `imoogi-project--context-call-root` is bound
  for the whole time `imoogi-project-context-call` runs. That covers more than
  the minibuffer reads of `f`/`s`/`d`. It also covers everything the command
  triggers before it returns, in particular the hooks run when `f` visits the
  chosen file (`find-file-hook`, the major-mode hooks such as `org-mode-hook`,
  `hack-local-variables-hook`) and `d`'s Dired setup hooks.

  During that time, an implicit lookup from any buffer whose
  `default-directory` equals the context root, and that no other finder
  answers, resolves to `(transient . CONTEXT-ROOT)`. Two kinds of buffer
  qualify:
  - the buffer the command was invoked from, whose `default-directory` is
    let-bound to the root;
  - a newly visited file that sits directly in the context root.

  Because the clause skips guard 8, this also holds for a notes-context buffer
  when the context root is a study notes directory. Eglot is still declined
  (guard 2).

  Today no hook in `26-project-notes.el` calls
  `imoogi-project-notes--current-source-root` or `project-current` from these
  hooks (no `find-file-hook` or mode hook is added there), so the D-6
  redirection does not occur. A future hook that does so would see the context
  root while the command runs. The effect ends when the command returns. At
  worst a redisplay meanwhile shows the root's project name.
- savehist-restored alist with duplicate names → first match wins (existing).
- Hot path: `project-current` runs on tab-bar redisplay. Guards 1-6 are pure
  Lisp, guard 7 is one `file-directory-p`, guard 8 is buffer-local plus an
  in-memory prefix comparison, and the resolved project comes from the D-8
  memo. Without the memo a non-VC root would be re-searched on every call,
  because `project-try-vc` caches only positive results.
  `imoogi-project-notes--all-entries` is NOT cached (it
  re-scans metadata, the registry JSON, and mounted roots each call), which is
  why guard 8 must not call it. A notes directory created outside Emacs (or a
  mount appearing without `--write-mounted-state`) is unknown until the next
  Perspective switch (D-7 trigger 6); until then the finder may claim a buffer
  visiting it — acceptable, since the effect is limited to that one buffer.
- Org preview allowed roots widen for `~/notes/scratch.org` in a mapped
  Perspective (the preview server may resolve links under R).
- Prompted directory equal to the buffer's `default-directory`: inside
  `project-current`, the prompt is reached only after the first lookup on that
  same directory failed, i.e. the finder already declined in the same state, so
  it declines again. The one caller that resolves a selected root outside
  `project-current`'s prompt branch, `imoogi-project-switch-perspective`
  (`(project-current t selected-root)`), binds the inhibit (AC-PIN-006c).

## § C Pre-flight

### C.1 Baseline

```bash
EMACS=/Applications/Emacs.app/Contents/MacOS/Emacs IMOOGI_ERT_SELECTOR='\(project-notes\|transient\|imoogi-project-\|perspective\|workspace\)' ./tests/run.sh
```
Record the count and the skip set before any change.

### C.2 Sandbox macro

`imoogi-persppin-test--isolated`: extends the t34 macro
(`tests/project-notes-perspective-test.el`) with `project-list-file`,
`project--list` let-bound to nil, `persp-mode` let-bound to t,
`project-find-functions` rebuilt by the installer, the D-8 memo and the D-7
notes-directory list reset, and prompt stubs that signal. The test file
declares `(defvar eglot-lsp-context)` so its `let` binding is dynamic under
`lexical-binding: t`. Stubs for `compile` are interactive lambdas, because
`project-compile` uses `call-interactively`.
Git-backed tests use `(skip-unless (executable-find "git"))` and
`call-process "git" nil nil nil "init" "-q"` inside the sandbox.

## § D Constraints

- No `project-files` / `project-buffers` method or advice.
- Nothing on the finder's path reads the notes registry, folder metadata, or
  mounted roots (REQ-PIN-008).
- No new package dependency.
- 04-projects.el does not `require` 26-project-notes.el.
- No edits to `.moai/reports/**`, CHANGELOG, or agent files in this SPEC's run.

## § E Self-Verification

- Every AC test fails before its milestone's change (RED evidence in
  progress.md §E.2) and passes after.
- The four t34 tests pass unmodified.
- Existing tests receive exactly the A.4 edits E1, E2 and E3, and no others. The
  two workspace-bridge context tests (:141, :160) pass with E2 plus E3, with
  their assertions unchanged.

## § F Milestones

### M1 — Finder and its guards (highest change likelihood)
AC-PIN-001..009, AC-PIN-018. Installer at depth 95, guards D-2, D-8 memo,
inhibit in `imoogi-project-switch-perspective`; edit E3 (A.4) lands here first,
because AC-PIN-001's non-VC root needs it.

### M2 — Notes-context exclusion and t34 compatibility
AC-PIN-010, AC-PIN-016. Predicate hook from 26-project-notes.el; the D-7
notes-directory list and its eight refresh points (trigger 8, rename rollback, added by the review rework).

### M3 — `b` command and candidate set
AC-PIN-011 (b part), AC-PIN-012, AC-PIN-017 (b part); the transient row change
and edit E1 (A.4).

### M4 — `f` notes files
AC-PIN-013, AC-PIN-017 (f part); D-4 including route (a) and D-2's
context-call clause for `s`/`d`; edit E2 (A.4).

### M5 — Non-membership and unchanged entries (most mechanical)
AC-PIN-011 (c/k part), AC-PIN-014, AC-PIN-015; full selector run (Quality Gate).

## § G Anti-Patterns

- Advising `project-current` or `project-buffers` globally.
- Replacing `project-prompter`.
- Calling `imoogi-project--perspective-root` from the finder before the remote
  check.
- Letting the finder claim a lookup whose directory is not the buffer's own.
- Writing to the real `~/.emacs.d/projects` from tests.

## § H Cross-References

- `.moai/reports/t34/progress.md` § Gaps / residual risk.
- `modules/project/04-projects.el:81-136`, `26-project-notes.el` ~1344-1400,
  `05-transient.el:314`, `22-tabs.el:54`, `23-org-preview.el:428`,
  `07-treemacs.el:554`.
- `project.el` 0.11.1 (`project-current`, `project--find-in-directory`,
  `project--read-project-buffer`, `project-try-vc`).
- SPEC-TRANSIENT-001 (transient layout).
