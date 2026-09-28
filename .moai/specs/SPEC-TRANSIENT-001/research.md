# Research — SPEC-TRANSIENT-001

Phase 6 (Deep Research) artifact. All findings below are attributed as **direct
source reads** performed with `Read`/`Bash`/`Grep` against this checkout —
none are agent-exploration summaries. Sources are cited per-item.

## 1. Exact current source: `modules/05-hydra.el`

Full 94-line file (user-supplied, verified present at this path and byte-for-byte
matching on re-read):

```elisp
;;; hydra.el --- Hydra definitions -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "05-hydra" 'hydra 'ace-window)

(use-package hydra
  :ensure t)

;;; ace-window
(use-package ace-window
  :ensure t
  :bind ("M-o" . ace-window)
  :custom
  (aw-keys '(?a ?s ?d ?f ?g ?h ?j ?k ?l))
  (aw-scope 'frame))

;; 창 관리
(defhydra hydra-window (:hint nil :color amaranth)
  "
  _h_: ←  _l_: →  _j_: ↓  _k_: ↑    _s_: 수평분할  _v_: 수직분할
  _H_: 축소← _L_: 확대→ _J_: 확대↓ _K_: 축소↑    _d_: 삭제  _D_: 나머지삭제
  _b_: 버퍼전환  _f_: 파일열기  _a_: ace-window  _m_: 스왑  _q_: 종료
  "
  ("h" windmove-left)
  ("l" windmove-right)
  ("j" windmove-down)
  ("k" windmove-up)
  ("H" shrink-window-horizontally)
  ("L" enlarge-window-horizontally)
  ("J" enlarge-window)
  ("K" shrink-window)
  ("s" split-window-below)
  ("v" split-window-right)
  ("d" delete-window)
  ("D" delete-other-windows :color blue)
  ("b" imoogi-consult-perspective-buffer)
  ("f" find-file)
  ("a" ace-window)
  ("m" ace-swap-window)
  ("q" nil :color blue))

;; 프로젝트
(defhydra hydra-project (:hint nil :color blue)
  "
  _f_: 파일찾기  _s_: 검색(grep)  _b_: 버퍼  _d_: dired
  _p_: 프로젝트+작업공간 전환  _k_: 버퍼모두닫기  _c_: 컴파일  _q_: 종료
  "
  ("f" project-find-file)
  ("s" project-find-regexp)
  ("b" project-switch-to-buffer)
  ("d" project-dired)
  ("p" imoogi-project-switch-perspective)
  ("k" project-kill-buffers)
  ("c" project-compile)
  ("q" nil))

;; 텍스트 확대/축소
(defhydra hydra-zoom (:hint nil :color amaranth)
  "
  _i_: 확대  _o_: 축소  _0_: 초기화  _q_: 종료
  "
  ("i" text-scale-increase)
  ("o" text-scale-decrease)
  ("0" (text-scale-set 0) :color blue)
  ("q" nil :color blue))

;; Git (Magit)
(defhydra hydra-git (:hint nil :color blue)
  "
  _s_: status  _l_: log  _b_: blame  _d_: diff  _q_: 종료
  "
  ("s" magit-status)
  ("l" magit-log-current)
  ("b" magit-blame)
  ("d" magit-diff-dwim)
  ("q" nil))

;; 마스터 hydra (진입점)
(defhydra hydra-master (:hint nil :color blue)
  "
  _w_: 창관리  _p_: 프로젝트  _g_: Git  _z_: 확대/축소
  _t_: treemacs  _q_: 종료
  "
  ("w" hydra-window/body)
  ("p" hydra-project/body)
  ("g" hydra-git/body)
  ("z" hydra-zoom/body)
  ("t" imoogi-treemacs-toggle-file-tree)
  ("q" nil))

(global-set-key (kbd "C-c h") 'hydra-master/body)

(provide 'imoogi-hydra)
;;; hydra.el ends here
```

**Scope correction confirmed**: there are **5** `defhydra` forms, not 4 —
`hydra-window`, `hydra-project`, `hydra-zoom`, `hydra-git`, and `hydra-master`
(the entry point, which additionally carries its own `t` head calling
`imoogi-treemacs-toggle-file-tree` that is not nested inside any sub-hydra).
All 5 must become `transient-define-prefix` forms.

### Color semantics (verified against hydra head markers above)

- `:color amaranth` (`hydra-window`, `hydra-zoom`): a head with **no** explicit
  `:color` marker stays open after firing. Heads explicitly marked
  `:color blue` (`hydra-window`'s `D`; `hydra-zoom`'s `0` and `q`) — plus the
  bare `q`/`nil` quit head in every hydra — exit. `hydra-window`'s own `q` head
  is also explicitly `:color blue`.
- `:color blue` (whole-hydra, on `hydra-project`, `hydra-git`, `hydra-master`):
  every head exits by default; no per-head marker needed.

## 2. Package/dependency state

Verified via direct reads of `packages.el`, `packages.lock`, and
`vendor/elpa/`:

- `packages.el` §"05-hydra" declares `hydra ace-window` (line 22). `hydra`
  must be removed; `ace-window` is untouched (it is not a hydra — only
  `hydra-window`'s `a`/`m` heads *call* `ace-window`/`ace-swap-window`,
  unrelated to the hydra→transient swap).
- `transient` is **not** currently a direct top-level entry in
  `imoogi-required-packages` — it must be added.
- `vendor/elpa/` already contains `transient-0.13.4` (+ `.signed`), pulled in
  transitively via `magit`'s dependency chain (`magit-4.5.0` depends on
  `magit-section`, `with-editor`, `transient`). Confirmed present:
  ```
  transient-0.13.4
  transient-0.13.4.signed
  ```
  So no `scripts/vendor.el` re-run is required for this SPEC's own
  dev/verification cycle — `packages.el` still needs the SSOT edit so it
  accurately reflects `transient` as a direct (not merely transitive)
  dependency going forward.
- `packages.el`'s header comment (lines 4-5) currently cites `transient` as
  the worked EXAMPLE of an auto-resolved transitive dependency:
  > "여기 적힌 top-level 패키지만 명시하면 전이 의존성(transient, with-editor,
  > dash, markdown-mode 등)은 package.el 이 자동 해결한다."
  Once `transient` becomes a direct entry, this example is no longer
  accurate and must drop `transient` from the illustrative list (leaving
  `with-editor, dash, markdown-mode` — all still genuinely transitive).
- `packages.lock` (a human-readable, non-authoritative audit trail per
  `tech.md`) lists both `hydra 0.15.0` (line 44) and `transient 0.13.4`
  (line 74) today. After migration, the `hydra` row should no longer read as
  an active/required entry; the `transient` row stays (now direct instead of
  transitive).

## 3. Module organization convention

Verified via `.moai/project/structure.md` § Module Organization Convention
and `ARCHITECTURE.md` § "새 모듈 추가 방법" (both read in full):

- Modules live at `modules/NN-name.el`, loaded by `boot.el`'s ascending
  `dolist` (lines 54-75), each load wrapped in `condition-case` for
  per-module failure isolation (`display-warning` on failure; the rest of
  the chain continues).
- Per-module contract: `(imoogi-require "NN-name" 'pkg1 'pkg2 ...)`
  immediately after `;;; Code:`; `use-package` configuration (`:ensure nil`
  for built-ins, `:ensure t` for vendored); end with `(provide 'imoogi-NAME)`;
  registered in `boot.el`'s `dolist` at the correct dependency-order
  position; new packages added to `packages.el`'s
  `imoogi-required-packages` with a `scripts/vendor.el` re-run on an online
  machine.

### Rename-in-place vs. renumber — verified reasoning

`boot.el` (read directly, current state) already carries the **post-rename**
module list — `"14-org" "15-markdown" "16-elisp" "17-lsp" "18-languages"
"19-folding" "20-terminal" "21-native-compile"` — matching the renames
visible in `git status` (`14-org-markdown.el` split into `14-org.el` +
`15-markdown.el`, shifting every later module up by one slot: 15→16, 16→18,
17→19, 18→20, 19→21). Those renames are **insertions** (a new module was
added at slot 15, requiring every later slot to shift).

This SPEC's situation is different in kind: no new module is being inserted
into the load sequence — an *existing* module at slot 05 is having its
backing package swapped (hydra → transient) with no new numbered module
appearing before or after it. **Conclusion: rename in place**
(`modules/05-hydra.el` → `modules/05-transient.el`, keeping slot `05`); no
shift of any other module's number is warranted or required.

### Load-order dependency claim — verified

`structure.md` documents "`05-hydra.el` depends on `02-completion.el`,
`04-projects.el`, and `06-git.el` loading first" — yet `boot.el`'s actual
load order places slot `05` **before** slot `06-git`. This is confirmed to
be **not a functional problem**: `defhydra`/`transient-define-prefix` forms
only store *symbol references* to commands (e.g. `magit-status`,
`imoogi-treemacs-toggle-file-tree`); those symbols are resolved at
**keypress time**, not at module-load time. Existing precedent for this
exact pattern already ships in the current file: `hydra-master`'s `t` head
calls `imoogi-treemacs-toggle-file-tree`, defined in `07-treemacs.el` — two
load positions *after* `05-hydra.el` — and this has worked correctly in
production. The same reasoning applies unchanged to `transient-define-prefix`
suffix specs. **No module renumbering or reordering is required by this
migration.** (The `structure.md` prose describing a "depends on 06 loading
first" relationship is a pre-existing minor documentation imprecision,
unrelated to and out of scope for this SPEC — see Out of Scope in `spec.md`.)

## 4. Additional impact discovered (beyond the user-supplied research)

Two references to the pre-migration symbol names exist **outside**
`modules/05-hydra.el` and were not present in the user-supplied research
brief. Both were found via `grep -rn "hydra" --include="*.el"`.

### 4.1 `tests/treemacs-tool-window-test.el` (real test breakage)

```elisp
(ert-deftest imoogi-treemacs-file-tree-wrapper-is-used-by-master-hydra ()
  (should (eq (cadr (assoc "t" hydra-master/heads))
              #'imoogi-treemacs-toggle-file-tree)))
```

This test reads `hydra-master`'s internal `/heads` alist — a data structure
that `defhydra` generates at macro-expansion time and that has **no
transient equivalent by that name**. Once `hydra-master` is retired, this
test will error (void-variable `hydra-master/heads`) rather than merely
fail. It **must** be rewritten against the transient prefix.

**Verified replacement API** (read directly from
`vendor/elpa/transient-0.13.4/transient.el`, lines 1844-1852):

```elisp
(defun transient-get-suffix (prefix loc)
  "Return the suffix or group at LOC in PREFIX. ..."
  (or (car (transient--locate-child prefix loc))
      (error "%s not found in %s" loc prefix)))
```

This is a **public**, documented API (`transient-get-suffix`, referenced
from transient's own `(transient)Modifying Existing Transients` info node,
also used internally by `transient-suffix-put`). Called with `PREFIX` a
prefix command symbol and `LOC` a key-description string (e.g. `"t"`), it
returns the suffix spec — a cons/list whose `cdr` is a plist carrying
`:command` (and `:key`, `:description`, etc.), matched by
`transient--match-child`'s `(plist-get (transient--suffix-props child)
:command)` comparison (same file, line 1887). The replacement assertion
shape is therefore:

```elisp
(let ((suffix (transient-get-suffix 'imoogi-transient-master "t")))
  (should (eq (plist-get (cdr suffix) :command)
              #'imoogi-treemacs-toggle-file-tree)))
```

(`transient--suffix-props` is `(defalias 'transient--suffix-props #'cdr)` —
a private alias; using the public `transient-get-suffix` + a plain `(cdr
...)` avoids depending on that private alias name.)

The test's own name (`...-is-used-by-master-hydra`) must also be renamed to
drop the now-inaccurate "hydra" reference.

### 4.2 `modules/02-completion.el` comment (stale reference)

```elisp
;;; Consult — 검색/탐색/미리보기
;; 주의: minimal 기본 바인딩 중 `C-c h'(→consult-history)는 imoogi hydra-master
;; 와 충돌하므로 제외했다.
```

This comment documents *why* `consult`'s default `C-c h` binding was
excluded (conflict with the master menu's entry point). It names the symbol
`hydra-master`, which this SPEC retires. Left as-is, it becomes a stale/
inaccurate reference to a symbol that no longer exists — the inaccuracy is
introduced *by* this migration, not pre-existing. A single-line text
correction (no functional change, no other edits to `02-completion.el`) is
in scope; see `spec.md` REQ-013.

### 4.3 False positive ruled out — `tests/workspace-bridge-test.el`

`grep -l transient` also matches this file, but on inspection every hit is
unrelated: `(list 'transient (file-truename root))` uses Emacs's built-in
`project.el` "transient project" concept (an ephemeral, unregistered
project), and a buffer literally named `"*imoogi-transient*"` used for
unrelated perspective-state serialization tests. **No changes needed here.**

## 5. Scope reconfirmation (per user instruction — not re-litigated, restated for traceability)

1. Add `transient` as an explicit top-level package in `packages.el`; remove
   `hydra`.
2. Rewrite `modules/05-hydra.el` → `modules/05-transient.el`, reimplementing
   all 5 `defhydra` forms as `transient-define-prefix` forms with identical
   keybindings, identical invoked commands, identical `C-c h` entry point,
   and identical amaranth-vs-blue "stay open" semantics.
3. Update `packages.el`'s stale-after-migration comment.
4. No other modules/features affected, beyond the two narrowly-scoped
   accuracy fixes discovered in §4.1/§4.2 above (both are direct,
   unavoidable consequences of retiring the `hydra-*` symbol names — see
   `spec.md` Out of Scope for the explicit boundary).

## 6. Sources

- Direct `Read` of `modules/05-hydra.el` (user-supplied content, re-verified
  present and identical in this checkout).
- Direct `Read` of `.moai/project/structure.md`, `.moai/project/tech.md`,
  `boot.el`, `packages.el`, `ARCHITECTURE.md` (module-addition guide).
- Direct `Bash grep`/`ls` of `packages.lock`, `vendor/elpa/`, `modules/`,
  `tests/` for `hydra`/`transient` references.
- Direct `Read` of `vendor/elpa/transient-0.13.4/transient.el` (lines
  1062, 1630-1852, 1960, 2254-2622, 2908, 3205, 4779) confirming the
  `transient-get-suffix` public accessor and the `transient--layout` /
  suffix-plist internal shape.
- Direct `Read` of `tests/treemacs-tool-window-test.el` and
  `modules/02-completion.el` (impact discovery beyond the pre-gathered
  brief).
