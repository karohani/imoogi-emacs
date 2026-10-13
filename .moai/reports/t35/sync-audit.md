# Sync Audit — SPEC-PERSPPIN-001 (card t35)

- Auditor: sync-auditor (independent, skeptical; Claude-only audit, no MCP backend call)
- Tree: `/Users/jay/workspace/imoogi-emacs/.moai/worktrees/t35`, branch `WT-persp-full-pin`, HEAD `3afb732`
  (base `6feee71`, plan checkpoint `641cc35`). Working tree at audit time: `.moai/reports/t35/progress.md`
  modified (not by this audit), `.moai/reports/t35/card-review.md` untracked (appeared mid-audit, not by this audit).
- Scope: `git diff 641cc35 HEAD -- modules tests README.md` (7 files, +1170/-27), SPEC v0.1.4 (15 REQ / 18 AC).
- Evaluator profile: none configured (no `harness.yaml` default, no `evaluator_profile` in the SPEC) → built-in
  default (Functionality 40 / Security 25 / Craft 20 / Consistency 15; must-pass Functionality + Security; flat mode).

## Evaluation Report
SPEC: SPEC-PERSPPIN-001
Overall Verdict: **FAIL** — the must-pass firewall holds (no AC fails, no Critical/High security finding), but there are
two **blocking** findings (F1 correctness regression, F2 README claim false). Both have small, delta-scoped fixes.

### Dimension Scores

| Dimension | Score | Verdict | Evidence (verbatim) |
|-----------|-------|---------|---------------------|
| Functionality (40%) | 78/100 | PASS (must-pass met; F1 blocking) | `make test-elisp`: `Ran 740 tests, 729 results as expected, 0 unexpected, 11 skipped (2026-10-13 09:17:02+0900, 68.175469 sec)` / `exit=0`. Selector `imoogi-persppin-`: `Ran 23 tests, 23 results as expected, 0 unexpected (2026-10-13 09:18:17+0900, 6.185132 sec)`. t34 + fallback + workspace-bridge selector: `Ran 7 tests, 7 results as expected, 0 unexpected (2026-10-13 09:25:26+0900, 1.340810 sec)`. Real boot (no sandbox): `BOOT project-find-functions: (project-try-vc imoogi-project-perspective-find)` / `BOOT exclude fns: (imoogi-project-notes--context-buffer-p)` / global-hook lookup from a projectless buffer → `(transient . ".../sa-boot-uzkA63/r/")`. Probe P18 (F1): `(:journal "project-notes/261013-xproj/journal.org" :buffer-source-root "source/" :headings ("** 작업 공간: /private/var/folders/.../imoogi-persppin-RGJrYN/source/"))` |
| Security (25%) | 95/100 | PASS | Added-line grep for `shell-command|call-process|start-process|make-process|eval|read-from-string|password|secret|token|api[-_]key` → only `324:+(with-eval-after-load 'perspective` (not an eval surface). Probe P6 (remote `default-directory`, TRAMP handler op spy): `(:finder nil :tramp-ops-from-finder nil)`. AC-PIN-008 test passes with signalling I/O stubs. |
| Craft (20%) | 85/100 | PASS (coverage UNVERIFIED — gap) | Byte-compile of changed modules into scratch: `=== modules/project/04-projects.el ===` followed by no warnings; all `26-project-notes.el` warnings are at lines 446-474, 2210-2213, 3401, 4679, 5041, 5228, 5343 — none in the diff hunks. No elisp coverage tool vendored (`no coverage/lint package vendored`) → the 85% coverage criterion is a **Gap**, not a PASS. |
| Consistency (15%) | 82/100 | PASS (F2 blocking) | Naming grep of added `defun/defvar/defmacro/defcustom` lacking `imoogi-` prefix → `none`. Existing-test edits = exactly E1–E3 (diff below). t34 test file: `git diff 6feee71 HEAD --stat -- tests/project-notes-perspective-test.el` → empty. F2/O-findings below are statement inconsistencies. |

Harmonic mean (4 dims, unweighted): 4 / (1/78 + 1/95 + 1/85 + 1/82) = **84.6** (0.846).
Weighted arithmetic (reference only): 0.40·78 + 0.25·95 + 0.20·85 + 0.15·82 = 84.3.

### Findings (structured defect-list)

- **F1 [Medium, confidence High] [blocking]** `modules/project/26-project-notes.el:1247-1264`
  (`imoogi-project-notes--context-buffer-p`) + `modules/project/04-projects.el:165-181` — A **Dired buffer on a
  notes directory** is not a notes-context buffer (the predicate keys only on `buffer-file-truename` and the three
  buffer-locals), so the finder claims it for the current Perspective. `imoogi-project-notes--current-source-root`
  (`26-project-notes.el:1465-1474`) consults `(project-current nil)` first, so from that Dired buffer the notes
  commands (`imoogi-project-notes-journal`, `-open`, `-tasks`, `--restore-workspace`, and the document helpers at
  :2032, :2050, :6508) take the **Perspective's** root as the source root of **another project's** entry. Before this
  SPEC they took the entry's own root. This is a regression of exactly what REQ-PIN-008's rationale forbids ("so
  `imoogi-project-notes--current-source-root` and journal resume sections are not redirected"). It is literally
  SPEC-conformant, because spec.md § 2 defines notes-context by visited file only, so this is a SPEC gap that the
  implementation inherits. The journal command inserts a wrong worktree heading into the other project's journal
  buffer and sets that buffer's source root to the Perspective root. `--ensure-worktree-journal-section`
  (:2858-2880) does not save, so the heading persists on the user's next save of the journal.
  **Reproduction** (batch, repo config booted as in `tests/run.el`, inside `imoogi-persppin-test--isolated`):
  1. `(imoogi-project-notes-setup XROOT)` for an unrelated non-VC dir XROOT; map ROOT → "src"; current Perspective "src";
     `(imoogi-project-notes--rebuild-context-directories)`.
  2. `(with-current-buffer (dired-noselect XNOTES) (imoogi-project-notes-journal))` with
     `imoogi-treemacs-open-workspace-roots` stubbed to `ignore`.
  3. Observed: X's `journal.org` gets `** 작업 공간: <ROOT>/` and buffer-local `imoogi-project-notes-source-root` = ROOT
     (P18). Same Dired buffer: `--current-source-root` = `source/` with the finder, `xproj/` with it inhibited (P17).
     P14: `(:context-buffer-p nil :project ".../source/")` for Dired on R's own notes.
  **Required fix** (needs a lane-3 decision because it amends spec.md § 2): either
  (a) extend the notes-context predicate so that a buffer with no `buffer-file-truename` counts as notes-context when
  its expanded `default-directory` has a prefix in the same in-memory `imoogi-project-notes--context-directories`
  list. This is still no file I/O. Amend spec.md § 2 + REQ-PIN-008 and add a Dired row to AC-PIN-010. Or
  (b) record it as an accepted known limitation in spec.md § 5 / acceptance.md Residual Risk and the README.
  Option (a) is the cheaper and safer one. Side effect of (a) to accept explicitly: Dired, compile, and shell buffers
  inside a study Perspective's notes root also become notes-context. `C-c h p c`/`k` from a Dired of the study notes
  would then prompt, where today they do not. That is consistent with D-6's accepted consequence for study note
  files, but it is a behaviour change.

- **F2 [Low, confidence High] [blocking — doc deliverable accuracy]** `README.md:461-462` — The sentence
  "자기 프로젝트가 있는 버퍼는 그 프로젝트를 그대로 쓰고" follows a list that includes `f`/`s`/`d`, so it claims those
  commands use the buffer's own project. They do not. By the accepted D-1 asymmetry (REQ-PIN-013, plan.md D-1),
  `f`/`s`/`d` follow the Perspective's root even from a buffer of another project.
  **Reproduction** (P1): in Perspective "src" mapped to ROOT, from a file buffer of a separate git repo OTHER:
  `:project-current OTHER/`, `:c OTHER/`, `:b ("o.txt")`, but `:d ROOT/` and `:f (ROOT/r.txt)`.
  **Required fix**: qualify the clause, e.g. "자기 프로젝트가 있는 버퍼에서 `b`/`c`/`k`와 `C-x p` 명령은 그 프로젝트를
  그대로 쓰고, `f`/`s`/`d`는 Perspective의 프로젝트를 따른다."

- **O1 [Low, confidence High] [optional]** `README.md:459` — "일반 `C-x p` 명령이 프로젝트를 묻지 않는다" is broader
  than the code: `C-x p p` (`imoogi-project-switch-perspective`) and `C-x p F` (`project-forget-project`) still prompt
  by design (spec § 4), and so does a TRAMP buffer (REQ-PIN-006). Fix: "(`p`/`F` 제외)" or similar.

- **O2 [Low, confidence High] [optional]** `README.md:467-468` — the `git init` limitation holds only on the finder path
  (projectless buffers: `b`/`c`/`k`, `C-x p`). `f`/`s`/`d` pick up the new VC project immediately. P19:
  `(:scratch-before transient :scratch-after transient :f-context vc)`. Fix: scope the sentence to buffers with no
  project of their own.

- **O3 [Low, confidence High] [optional]** `.moai/specs/SPEC-PERSPPIN-001/spec.md` REQ-PIN-013.1 / plan.md D-4 — the
  route (a) formula omits the O3-d remote gate the code carries (`04-projects.el:249-254`,
  `(not (file-remote-p context-root))`), which is tested by `imoogi-persppin-find-file-remote-context-root-prompts`.
  progress.md records O3-d as an implementation constraint, but the v0.1.4 "post-run text alignment" did not add it to
  the SPEC. Fix: add "for a local CONTEXT-ROOT" to the formula.

- **O4 [Low, confidence High] [optional]** `tests/project-perspective-pin-test.el:194-200` — AC-PIN-005(b) is vacuous
  for the `persp-mode` guard specifically. With `persp-current-name` stubbed to signal, the top-level `condition-case`
  would decline even if the guard were missing. Probe P4 shows the guard is real: with `persp-mode` nil and the name stub
  returning the mapped "src", the result is `(:result nil :persp-current-name-calls 0)`. Fix (optional): stub the name to
  return "src" in case (b) and assert zero calls.

- **O5 [Low, confidence Medium] [optional]** `.moai/specs/SPEC-PERSPPIN-001/acceptance.md` Quality Gate — "skips limited
  to the pre-existing `cli-cache-*`" is not what the gate run shows: 11 skips = 7 `cli-cache-*` + 2
  `org-calendar-*-gui` + 2 `toolchain-*`. All of these are environment skips unrelated to this SPEC. Fix: reword to "skips
  limited to pre-existing environment-dependent tests".

- **O6 [Low, confidence High] [optional]** `04-projects.el:173-174` (exclude hook) — an exclude predicate that itself
  calls `project-current` recurses until `excessive-lisp-nesting`. The `condition-case` contains it (no error escapes),
  but it takes ~5.6 s (P7: `:max-depth 104 :secs 5.59`). The docstring already forbids I/O in predicates, and the only
  shipped predicate does none, so this state is not reachable today. Optional: also mention `project-current` in the
  docstring's "must not" sentence.

- **O7 [Info, confidence Medium] [optional]** `26-project-notes.el:1257-1262` — before the first rebuild, the fallback
  compares `(expand-file-name imoogi-project-notes-directory)`, which is not a truename, against `buffer-file-truename`.
  If `~/project-notes` is a symlink, notes buffers are not recognised until `window-setup-hook` runs. This is a startup
  window only.

- **O8 [Low, confidence High] [optional]** `tests/project-perspective-pin-test.el` — no test asserts that the finder is
  installed at boot. Every test re-installs it inside the sandbox, so a broken `use-package project :config` clause
  (`04-projects.el:511-513`) would pass the suite. Verified manually at real boot (Functionality evidence). Optional: a
  registration test like `notes-list-startup-registered` asserting
  `(memq #'imoogi-project-perspective-find (default-value 'project-find-functions))`.

### Probe results (beyond the AC tests)

Each probe ran in batch Emacs with the repo booted as in `tests/run.el`, using `imoogi-persppin-test--isolated`
sandboxes. All temporary files lived in the session scratchpad, and they were deleted.

| # | Probe | Result |
|---|-------|--------|
| P1 | Buffer of another VC project inside pinned Perspective keeps its project | PASS: `project-current`/`c`/`b` → OTHER; `d`/`f` → ROOT (D-1 as decided; see F2) |
| P2 | Unregistered project found by another finder member keeps its own | PASS: `(:own (transient . ".../unreg/") :registered nil)` |
| P3 | `*scratch*` in an unmapped Perspective still prompts | PASS: `(:nil nil :prompt "prompted")` |
| P4 | `persp-mode` off (name stub returns a mapped name) | PASS: `(:result nil :persp-current-name-calls 0)` |
| P5 | Mapped root deleted after a memo hit | PASS: `:before ".../gone/" :after nil :prompt "prompted"` (`file-directory-p` runs before the memo) |
| P6 | Remote `default-directory`, TRAMP-handler op spy | PASS: `(:finder nil :tramp-ops-from-finder nil)` |
| P7 | Recursion: exclude predicate calling `project-current` | Terminates, no error escapes; slow (O6) |
| P7b | Recursion: `project-current` inside the finder's own resolution | PASS: inhibit stops re-entry; `:root ".../source/"` |
| P8 | Memo after remap and remap back | PASS: `:a source/ :b r2/ :c source/ :memo-size 2` |
| P9 | `imoogi-project-switch-perspective` choosing the current non-VC dir | PASS: `:switched "elsewhere"`, alist entry for `elsewhere/`, remembered |
| P10 | `b` from `*scratch*`: notes in, other projects out | PASS: `(:p1 t :n1 t :o1 nil :m1 nil)` |
| P11 | `k` (confirm yes) never kills notes buffers | PASS: `(:p1-live nil :n1-live t :n2-live t)` |
| P12 | Rebuild error during `--write-registry` | PASS: `(:ret nil :file-exists t :contains t :list ("/sentinel/"))` (write done, previous list kept) |
| P12b | Rebuild error during `--write-mounted-state` | PASS: `(:file-exists t)` |
| P13 | Hot path after memo fill (projectless + notes buffer, 20 finder calls) | PASS: `((file-directory-p . 20) (finder-calls . 20))`, with no `file-truename`, registry, `all-entries`, `insert-file-contents`, or `directory-files` on the finder path. The 180 `directory-files` / 460 `file-directory-p` calls in `project-current`+`imoogi-tab-name` come from upstream `project-try-vc` on a non-VC dir, which predates this SPEC |
| P14 | Dired on notes dir | Claimed by finder → F1 |
| P15 | `f` with a **VC** root + notes (untested by AC-PIN-013) | PASS: `:project-car vc`, notes files included, `:notes-git nil` |
| P16 | Context-call binding seen by another buffer at the root during `f` | Benign: answers `(transient . ROOT)`, the same as the mapping would |
| P17/P18 | Dired on another project's notes in Perspective A → notes commands | FAIL → F1 |
| P19 | `git init` inside pinned non-VC root | Scratch keeps `transient`; `f` sees `vc` (O2) |

Re-entrancy check: every `--write-registry` / `--write-mounted-state` caller is command-level (`--save-entry`,
`reconnect-source`, `clear-source-override`, `mounted-root-add/edit/remove`, `--remove-local-entry`, promote). None is
reachable from `--all-entries` (`--scan-local-metadata`, `--read-registry`, `--read-mounted-state`,
`--scan-mounted-root`), so the rebuild added to the writers cannot recurse.

### AC vacuity review

Each AC test was checked for whether it fails when the guarded behaviour is removed:
- AC-PIN-001/002/007/018: positive and negative halves; the memo spy counts `project-try-vc` on R via `file-equal-p`. Non-vacuous.
- AC-PIN-003: F2 claims OTHER; if the finder sorted before F2, OTHER would resolve to R. Non-vacuous.
- AC-PIN-004/011/012/014/017: exact-set or liveness assertions; `b` capture filters `buffer-list` through the real predicate. Non-vacuous.
- AC-PIN-005: (a) and (c) non-vacuous; (b) partially vacuous (O4), covered by probe P4.
- AC-PIN-006(c): the prompter returns `elsewhere` = `default-directory` (non-VC). Without the inhibit, the finder would answer R, so the inhibit is load-bearing.
- AC-PIN-008: the guard pushes onto `io` before signalling, so a swallowed error still fails `(should-not io)`. Non-vacuous.
- AC-PIN-009: `calls >= 1`. Non-vacuous.
- AC-PIN-010: rows 1 and 6 first establish "claimed" and then "declined after rebuild". Row 5's rebuild is the only trigger in `unmount-device`, which writes no mounted state (`26-project-notes.el:4019-4021`). Non-vacuous.
- AC-PIN-013/015: absolute ALL-FILES capture; context-call checks also come from a notes-context buffer (asserted by `--context-buffer-p`). Non-vacuous.
- AC-PIN-016: the t34 file is unmodified since base; 4/4 pass with the finder installed.

Existing-test edits are exactly E1 (`transient-menu-test.el:80`), E2 (`workspace-bridge-test.el:152-154`, `:171-173`,
stub `project-find-file` → `project-find-file-in`, assertions unchanged), and E3 (`workspace-bridge-test.el:20`,
shape-tolerant `project-root`). No other existing test file changed (`git diff 6feee71 HEAD --stat -- tests/`).

### Lane-decision consistency

- D-1 precedence asymmetry: consistent across spec § 3.2/§ 5, plan D-1, acceptance Residual Risk, and the code (P1). **README misstates it (F2).**
- D-6 (`c`/`k` prompt from notes buffers): consistent in spec § 5, acceptance, README limitation 1, and the code. Gap for non-file notes buffers (F1).
- Route (a) for `f` + context-call clause for `f`/`s`/`d`: consistent in spec, plan D-2/D-4, and the code (`04-projects.el:141-163`, `:225-261`). The O3-d remote gate is missing from the SPEC text (O3).
- Trigger 7 on `window-setup-hook`: consistent in spec HISTORY/REQ-PIN-008, plan D-7, acceptance, the code comment (`26-project-notes.el:6559-6569`), and the test.
- Memo per Perspective: consistent in spec REQ-PIN-001/§ 6, plan D-8, acceptance O-f, and the code docstring (`04-projects.el:105-112`).

### Hot path (REQ-PIN-006/007/008, spec § 6)

The finder costs, per call, guard checks on raw strings (`file-remote-p` before `expand-file-name`), one `rassoc`, one
`file-directory-p`, the in-memory notes predicate, and an `assoc` memo hit (P13: exactly 1 `file-directory-p` per call,
nothing else). `condition-case` wraps the whole body (`04-projects.el:185-193`). The notes-list rebuild is wrapped and
returns nil (P12/P12b). No registry, metadata, or mounted-root read is on the lookup path (AC-PIN-010 test + P13).

### Recommendations

1. F1: take a lane-3 decision. The preferred fix is option (a), the `default-directory` prefix check for non-file
   buffers against the in-memory list, with SPEC § 2 / REQ-PIN-008 / AC-PIN-010 amended. The re-audit can be scoped to
   that predicate, a new Dired row, and P17/P18.
2. F2: one-clause README fix; fold O1/O2 into the same edit.
3. Optional: O3 (SPEC text), O4 (test hardening), O5 (gate wording).
