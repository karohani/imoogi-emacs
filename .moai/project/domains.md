# Domains

Per-domain implementation summary for imoogi-emacs. Where `product.md` describes what the project is for and `structure.md` describes where files live, this document walks each major domain once: what it does today, how it is built, where its state lives, what tests cover it, and what is known to be missing.

Scope rules for this document:

- Every command, key and path below was confirmed in the source at HEAD `c01da68` (2026-09-27). Line numbers are given as `path:line` where they help.
- "Known gaps" lists only items with a pointer: a backlog card (`moai todo list`, ids `tNN`) or a comment in code. Nothing here is speculative.
- Korean appears only where the code itself uses Korean (menu labels, card text).

## Domain Index

| # | Domain | One line | Primary modules | CLI |
|---|---|---|---|---|
| 1 | Editor core and UX | Defaults, keys, Korean input, completion, transient menus, theme, navigation, editing | `modules/general/*` | - |
| 2 | Project workspace | `project.el` + perspective, tab-bar, Treemacs, Magit | `modules/project/{04,06,07,22}` | - |
| 3 | Project notes | Project and study notes, removable roots, artifacts, verified moves | `modules/project/26-project-notes.el` | `imoogi-notes` |
| 4 | Org knowledge | Org authoring and agenda, Org-roam, Obsidian, Markdown, clipboard assets | `modules/org/{08,14,15,28,29}` | `imoogi-clip` |
| 5 | Org preview | Live browser preview of Org/Markdown over loopback HTTP | `modules/org/23-org-preview.el` | `imoogi-org-preview` |
| 6 | Spaced repetition | One-way Org to Anki sync, plus a SQLite-backed local fallback | `modules/org/{24-anki,25-flashcards}`, `org/anki/`, `org/flashcards/` | `imoogi-anki` |
| 7 | Development tooling | Formatting, elisp, Eglot/LSP, major modes, folding, terminal, gptel | `modules/development/*` | - |
| 8 | Air-gap supply chain | Vendored packages, toolchain fetch/setup, provenance manifests | `packages.el`, `scripts/`, `vendor/`, `provenance/` | `imoogi-toolchain`, `imoogi-provenance` |
| 9 | Agent IPC | Coding agents deliver events into the running Emacs | `modules/development/30-agent.el` | `imoogi-agent` |

All modules are loaded from the explicit list in `boot.el:65-100`; each load is wrapped in `condition-case`, so a module whose prerequisites are missing is skipped and recorded in `imoogi-failed-modules` rather than aborting boot.

---

## 1. Editor core and UX

**Purpose.** Give an IDE-like, keyboard-driven editing baseline that works offline and treats Korean input as a first-class concern.

**Implemented capabilities**

- Defaults and session persistence: `recentf`, `savehist`, `saveplace`, `dired` (`modules/general/00-defaults.el:111-137`).
- Keys (`modules/general/01-keys.el`): `C-g` is rebound to `imoogi-keyboard-quit` (`:16`); `S-SPC` toggles the input method (`:54`); macOS-style `s-c`/`s-v`/`s-x`/`s-w`/`s-z`/`s-a` (`:84-89`); `s-M-l` and `C-M-\` run `imoogi-format-code` (`:94-95`). The input source is switched through `im-select` via `call-process` (`:35`, `:40`).
- Completion (`02-completion.el`): vertico/orderless/marginalia/consult/corfu/cape stack; `C-s` is `consult-line` (`:182`), `C-c e` is `cape-prefix-map` (`:204`), `C-c k`/`C-c i`/`C-c M-x` go to consult commands (`:123-125`); `imoogi-consult-perspective-buffer` scopes buffer switching to the current perspective.
- Which-key hints (`03-which-key.el`).
- Transient menus (`05-transient.el`): the master menu `imoogi-transient-master` on `C-c h` (`:584`, `:601`) fans out to window, project, Git, zoom, code, modes and Treemacs menus, the persistent scratch (`imoogi-notes-scratch`) and `imoogi-reload`. Menu labels are Korean ("창관리", "프로젝트", "설정 재로드").
- Theme (`10-theme.el`): doom-themes, doom-modeline, nerd-icons, `hl-line`; bundled fonts are copied locally and `fc-cache` is run (`:94`); `imoogi-font-report` reports font state.
- Editing (`11-editing.el`): undo-fu with `C-z` / `C-S-z` (`:16-17`) and undo-fu-session, yasnippet, apheleia, dumb-jump, stripspace, elec-pair, visual-wrap, so-long.
- Navigation (`12-navigation.el`): avy on `C-'` (`:14`), helpful, bufferfile.
- System (`13-system.el`): exec-path-from-shell, the Emacs `server` (`:113`, required by domain 9), buffer-terminator, persist-text-scale; `imoogi-system-set-ca-certificate` / `imoogi-system-clear-ca-certificate` configure a private CA, surfaced through `imoogi-system-transient` (`:87`).
- Auto-revert (`09-autorevert.el`) and late native compilation of everything already loaded (`21-native-compile.el`).

**Structure.** Pure Emacs Lisp. Each file calls `(imoogi-require "NN-name" 'pkg ...)` (`boot.el:44`) and configures packages with `use-package`. Transient prefixes for other domains are declared inside `with-eval-after-load 'imoogi-transient` so the menu module can load first.

**Data and state.** Standard Emacs history and place files under the user Emacs directory; nothing in the repository.

**Tests.** `tests/keyboard-quit-test.el`, `tests/intellij-keybindings-test.el`, `tests/transient-menu-test.el`, `tests/transient-input-method-test.el`, `tests/compiled-menu-test.el`, `tests/module-layout-test.el`, `tests/boot-health-test.el`.

**Known gaps.** Card `t7` (picked): after moving to another window while a minibuffer is active, `C-g` does not abort it; the card proposes a DWIM `abort-recursive-edit`. Note that the card still names the pre-reorganization path `modules/01-keys.el`.

---

## 2. Project workspace

**Purpose.** Tie a source project, its buffers and its window layout together, IntelliJ-style.

**Implemented capabilities**

- `project.el` plus `perspective` (`modules/project/04-projects.el:297`, `:304`): `imoogi-project-find-file`, `imoogi-project-find-regexp`, `imoogi-project-dired`, `imoogi-project-switch-perspective`, `imoogi-persp-new`, `imoogi-project-context-call`; `persp-switch-last` on `l` in `perspective-map` (`:311`).
- Git: `magit` and `diff-hl` (`06-git.el:6`, `:10`), with the `imoogi-transient-git` menu (`05-transient.el:397`).
- Treemacs tool windows (`07-treemacs.el`): file tree, bookmarks and structure panels (`imoogi-treemacs-toggle-file-tree`, `-toggle-bookmarks`, `-toggle-structure`, plus refresh and `imoogi-treemacs-check-duplicates`); `M-0` selects the Treemacs window (`:255`); treemacs-magit and treemacs-icons-dired integration.
- Tabs (`22-tabs.el`): `C-c w` tab map (`:47`), `s-[` / `s-]` switch tabs (`:48-49`), `imoogi-transient-tab` (`:66`). The which-key label describes tabs as tmux-window equivalents ("탭(tmux window)", `:60`).

**Structure.** Emacs Lisp only; menus are reachable from `imoogi-transient-project` and `imoogi-transient-treemacs` (`05-transient.el:310`, `:503`).

**Data and state.** Perspective and Treemacs state are held by those packages in the user Emacs directory.

**Tests.** `tests/treemacs-tool-window-test.el`, `tests/workspace-bridge-test.el`.

**Known gaps.** None recorded in the backlog or code comments.

---

## 3. Project notes

**Purpose.** Keep Org notes per source project or per course, beside or entirely apart from the source tree, including on removable devices that move between machines.

**Implemented capabilities** (all in `modules/project/26-project-notes.el`)

- Setup and diagnosis: `imoogi-project-notes-setup`, `-setup-study`, `-setup-doctor`, `-setup-guide`.
- Daily use: `imoogi-project-notes-open`, `-tasks`, `-journal`, `-list`, `-add-document`, `-raiseup`, `-create-artifact`, and agenda views `-agenda-current` / `-agenda-all`.
- Source linkage: `-reconnect-source`, `-clear-source-override`, `-force-edit-session`.
- Removable roots: `-mounted-root-add`, `-list`, `-edit`, `-remove`, `-refresh`, `-detach`, `-unmount-device`, and `-move-to-mounted-root`.
- Persistent scratch buffer: `imoogi-notes-scratch` (also on `n` in the master menu).
- A dedicated menu, `imoogi-project-notes-transient` (`modules/general/05-transient.el:337`).

**Structure**

- Elisp module creates notes from `templates/project-notes/` (`project.org`, `tasks.org`, `journal.org`, `architecture.org`, `decisions.org`, `domain.org`, `questions.org`, `artifact.org`, `cards.org`, `scratch.org`, and the study set `study.org`, `study-tasks.org`, `study-journal.org`).
- A note folder identifies itself with `.imoogi-project.json` (`:73`), so it can be rediscovered on another machine from the folder alone.
- Moving a note to a mounted root shells out to `imoogi-notes` (`imoogi-project-notes-command`, `:35`) with `call-process-region` (`:2984`): JSON request on stdin (read up to 1 MiB, `cmd/imoogi-notes/main.go:23`), JSON response on stdout. The Go side (`internal/notemove/move.go`) copies, verifies every file with SHA-256 and switches the original before Emacs updates registrations, agenda paths and open buffers.
- Git metadata for source projects is read with `call-process "git"` (`:438`).

**Data and state**

- Default note root `~/project-notes/` (`imoogi-project-notes-directory`, `:21`).
- Project registry `~/.emacs.d/.cache/project-notes.json` (`:312-314`) and host-local mounted-root registry `.cache/project-notes-mounted-roots.json` (`:316-318`).
- Todo placement is configurable through `imoogi-project-notes-todo-storage` (`:26`).

**Tests.** `tests/project-notes-test.el`; `internal/notemove/move_test.go`.

**Known gaps.** Card `t23` (queued): free-form standing documents under `development/<slug>.org`, a per-project document-kind setting in `.imoogi-project.json`, and promoting an artifact into `development/` while keeping its `:ID:`. The card cites the fixed documents constant at `:226`.

---

## 4. Org knowledge

**Purpose.** Org as the main writing surface: agenda, blocks, export, a permanent-note graph, plus Obsidian and Markdown interop and pasting assets from the clipboard.

**Implemented capabilities**

- Org (`modules/org/14-org.el`): `imoogi-org-setup`, `imoogi-org-setup-doctor`, `imoogi-org-agenda`, `imoogi-org-agenda-overview`, `imoogi-org-open-agenda-file`, `imoogi-org-set-category`, `imoogi-org-export`, block insertion (`imoogi-org-insert-block`, `-source-block`, `-example-block`); menus `imoogi-org-block-transient` (`:479`) and `imoogi-org-agenda-transient` (`:488`); Babel languages enabled after Org loads (`:28-29`).
- Org-roam (`29-org-roam.el`): permanent notes, `C-c n` opens `imoogi-org-roam-transient` (`:39`, `:50`); DB autosync turns on only when the notes directory exists; `imoogi-org-roam-clear-completions-cache`.
- Obsidian (`08-obsidian.el`): vault `~/obsidian` (`:13`); in the Obsidian map `C-c C-n` capture, `C-c C-l` insert link, `C-c C-o` follow, `C-c C-p` jump, `C-c C-b` backlinks (`:17-21`).
- Markdown (`15-markdown.el`): markdown-mode and markdown-toc; `imoogi-markdown-insert-task-list-item`, `imoogi-markdown-demote`.
- Clipboard assets (`28-clipboard.el`): `yank` is remapped to `imoogi-clipboard-yank` in the clipboard minor-mode map (`:688`); `imoogi-clipboard-show-diagnostics`. Screenshots and copied files are copied into the note folder under a staging lease, and a lease-renew timer runs every 300 s (`:21`).

**Structure**

- Clipboard work goes to `imoogi-clip` (`imoogi-clipboard-command`, `:13`) through `make-process` (`:121`, `:145`) with a short inspect timeout (`:17`). The protocol (`internal/clipboard/protocol.go:23-33`) defines the operations `inspect`, `checkpoint`, `paste`, `import`, `finalize`, `document-saved`, `commit`, `abort`, `reconcile`, `prune`, `version`.
- Platform adapters: macOS via Objective-C pasteboard (`internal/clipboard/adapter_darwin.go`, `pasteboard_darwin.m`); other platforms return an unsupported-capability result (`adapter_unsupported.go`). Process identity checks exist for darwin and linux.

**Data and state.** Default Org directory `~/notes/` (`14-org.el:31-33`), default agenda file `~/notes/agenda.org` (`:40-42`), permanent notes `~/notes/permanent/` (`:36-38`). Assets land in the note's own folder.

**Tests.** `tests/org-setup-test.el`, `org-agenda-overview-test.el`, `org-block-test.el`, `org-calendar-test.el`, `org-heading-test.el`, `org-roam-test.el`, `markdown-heading-test.el`, `json-imenu-test.el`, `clipboard-test.el`; Go `internal/clipboard/{core,protocol,service}_test.go`.

**Known gaps**

- Card `t4` (picked): Denote is not vendored; the card asks whether it should sit beside or replace Obsidian.
- Card `t6` (queued): archive long-done agenda items; blocked on `org-log-done` not being set.
- Card `t5` (queued) states that `14-org.el` has no `org-agenda-files` / `org-directory` settings. That description is out of date: `14-org.el` now defines a default Org directory and agenda file (`:31-42`, `:54-66`). The card should be re-checked before it is worked.

---

## 5. Org preview

**Purpose.** Render the current Org or Markdown buffer as HTML in a browser, live, without any network beyond loopback.

**Implemented capabilities**

- `imoogi-org-preview` starts or reuses the server and opens the page; `imoogi-org-preview-open` and `imoogi-org-preview-stop` (`modules/org/23-org-preview.el`).
- Debounced updates (`imoogi-org-preview-debounce-seconds`, default 0.5 s, `:41`), reconnect interval (`:45`), current-heading highlight (`:49`), optional Emacs-browser navigation sync (`:53`, off by default).
- Server-side rendering of Org and Markdown, assets, file preview, and bundled Mermaid (`/static/mermaid.min.js`).

**Structure**

- Emacs launches `imoogi-org-preview` with `make-process` (`:240`); the server prints one bootstrap JSON line (`--print-bootstrap-json`) with host, port and token (`cmd/imoogi-org-preview/main.go:19-53`). It binds only to loopback (`--host` default `127.0.0.1`) and generates a session token when none is given.
- Emacs posts buffer revisions with `url-retrieve` and a JSON body (`23-org-preview.el:484-490`).
- HTTP routes (`internal/orgpreview/server.go:134-143`): `/`, `/health`, `/api/emacs/revisions`, `/api/emacs/events`, `/api/emacs/navigation`, `/api/sessions/`, `/api/file-preview`, `/ws/browser` (WebSocket to the browser), `/asset`, `/static/mermaid.min.js`.
- Go packages: `internal/orgpreview/{parser,render,protocol,session,server,assets,web}`.

**Data and state.** In-memory sessions only; host, port and token are configurable (`:28-37`).

**Tests.** `tests/org-preview-test.el`, `tests/org-preview-contract_test.go` (Emacs/Go protocol contract), `internal/orgpreview/orgpreview_test.go`.

**Known gaps.** None recorded.

---

## 6. Spaced repetition (Anki sync and local flashcards)

**Purpose.** Turn Org headings into flashcards. The main path syncs one way to Anki; where Anki is unavailable, an Emacs-native SQLite deck is the fallback. The two share no state.

### 6a. Anki sync

**Implemented capabilities**

- Marking cards in Org: `imoogi-anki-mark-basic`, `-mark-cloze`, `-unmark`, `-set-tags` (`modules/org/24-anki.el`), on the Org-mode prefix `C-c a` (`:423`) and in `imoogi-anki-transient` (`:442`).
- Sync targets: `imoogi-anki-register-directory`, `-register-file`, `-unregister-target`, `-list-targets`, `-list-files` (`modules/org/anki/imoogi-targets.el`).
- `imoogi-sync` runs a sync; `imoogi-anki-open-log` opens the diagnostic log (`modules/org/anki/imoogi.el`).
- Two imoogi-owned note types (`imoogi-Basic`, `imoogi-Cloze`) installed by the back end's `install-models` command, and a `migrate` command for existing notes (`cmd/imoogi-anki/main.go:63-70`).
- The only value written back into Org is `ANKI_NOTE_ID` (`modules/org/anki/imoogi-writeback.el`).

**Structure**

- Elisp libraries in `modules/org/anki/`: `imoogi.el` (entry point and defcustoms), `imoogi-config`, `imoogi-scan` / `imoogi-target-scan` (collect headings, nearest-wins property resolution in `imoogi-props`), `imoogi-process` (runs the binary with `call-process-region`, `:66`), `imoogi-writeback`, `imoogi-error` (maps back-end error codes to messages), `imoogi-setup`.
- `imoogi-anki` reads a JSON request on stdin and writes JSON on stdout; subcommands `sync`, `install-models`, `migrate`, `--version`.
- Go packages `internal/anki/{orgdoc,planner,model,media,hashing,registry,protocol,ankiconnect}`: Org rendering to card fields, note-write planning, media handling, content hashing, and the AnkiConnect HTTP client (default `http://127.0.0.1:8765`, `imoogi.el:58`).

**Data and state.** Config `~/.emacs.d/imoogi.json` (`imoogi-config-file`, `imoogi.el:63`); JSONL diagnostic log `~/.emacs.d/.cache/imoogi-anki.log`, which never includes card titles or bodies (`:69-75`); optional user stylesheet (`:77`); sync root and exclude patterns (`:34`, `:42`).

**Tests.** `tests/anki-*-test.el` (commands, config, error, install, migrate, notetype, process, props, scan, setup, sync-error, sync-oneway, target-scan, target-sync, targets, writeback); Go `cmd/imoogi-anki/*_test.go` and `internal/anki/*/*_test.go` (planner 12 files, orgdoc 7, model 6).

**Known gaps**

- Card `t15` (picked): "Swift" arrow cards (`:->`, `:<-`, `:<->`), planned in `internal/anki/orgdoc/swift.go`.

### 6b. Local flashcards (SQLite)

**Implemented capabilities**

- Marking: `imoogi-flashcards-mark-basic`, `-mark-cloze`, `-unmark`, `-cloze-region` (`modules/org/flashcards/imoogi-flashcards-org.el`).
- Syncing Org into the database: `imoogi-flashcards-sync-root`, `imoogi-flashcards-sync-buffer` (`modules/org/25-flashcards.el`).
- Review session: `imoogi-flashcards-review-start`, show answer, rate `again` / `hard` / `good` / `easy`, quit (`imoogi-flashcards-review.el`).
- Org-mode prefix `C-c f` (`25-flashcards.el:106`) and `imoogi-flashcards-transient` (`:111`).

**Structure.** Four libraries: `-core` (scheduler), `-org` (heading parsing), `-repository` (SQLite access), `-review` (UI). The module is skipped when Emacs has no SQLite support (`boot.el` comment on the `org/25-flashcards` entry).

**Data and state.** `~/.emacs.d/.cache/flashcards.sqlite` (`imoogi-flashcards-database-file`, `imoogi-flashcards-repository.el:14`).

**Tests.** `tests/flashcards-{boot,command,core,org,repository,review}-test.el`.

**Known gaps.** None recorded.

---

## 7. Development tooling

**Purpose.** LSP-backed editing across many languages, plus terminal, folding, formatting and an LLM client, all working offline.

**Implemented capabilities**

- Formatting: `imoogi-format-code` (`modules/development/formatting.el`), loaded third in boot so every language module sees it; bound to `s-M-l` / `C-M-\`.
- Elisp: aggressive-indent, highlight-defined, paredit, page-break-lines, elisp-refs (`16-elisp.el:12-36`).
- LSP (`17-lsp.el`): Eglot with Flymake and xref; `C-c l` is `imoogi-lsp-map` (`:134`) and `imoogi-transient-lsp` (`:153`). It auto-discovers `modules/development/lang/*.el` (bash, clojure, go, java, javascript, kotlin, python, rust, typescript) with per-file failure isolation.
- Major modes for about twenty more file types (`18-languages.el:133-267`: git-modes, yaml, dockerfile, lua, jinja2, csv, go, rust, nginx, hcl, nix, fish, kotlin, typescript, web-mode and others); tree-sitter grammars from `vendor/tree-sitter/`.
- Folding with kirigami under `C-c z` (`19-folding.el:14-19`).
- Terminal: `ghostel` on `C-c t` (`20-terminal.el:20`), using the prebuilt module in `vendor/ghostel-module/`.
- gptel against a LiteLLM gateway (`27-gptel.el`): `imoogi-gptel-chat`, `-send`, `-menu` (`imoogi-gptel-transient`, `:1130`), profile management (`-add-litellm-profile`, `-edit-`, `-switch-`), `-store-key` (auth-source), `-auth-diagnose`, `-open-log`, `-open-config`, `-reload-config`, `-setup-guide`.
- Code understanding menu `imoogi-transient-code` (`05-transient.el:475`) and `imoogi-code-capability-report`.

**Structure.** Language servers are not fetched at boot; they come from domain 8 (`imoogi-toolchain`). gptel reaches the gateway over HTTP with an optional private CA from domain 1.

**Data and state.** gptel config `~/.emacs.d/imoogi-gptel.json` (non-secret, `27-gptel.el:30`) and private log `~/.emacs.d/.cache/imoogi-gptel.log` (`:35`); keys go to auth-source.

**Tests.** `tests/lsp-modules-test.el`, `toolchain-servers-test.el`, `treesit-grammar-test.el`, `gptel-test.el`.

**Known gaps.** Card `t3` (picked): jdtls, basedpyright, kotlin-language-server and clojure-lsp are not vendored, so the Java, Python, Kotlin and Clojure configs are silently inactive; the JVM bundling decision is open.

---

## 8. Air-gap supply chain

**Purpose.** Make the offline boot guarantee hold: every dependency is vendored ahead of time and hash-verified.

**Implemented capabilities**

- Emacs packages: `packages.el` (`imoogi-required-packages`) is the manifest; `scripts/vendor.el` fills `vendor/elpa/` online; `packages.lock` is an audit record only.
- Tree-sitter grammars built by `scripts/build-grammars.sh`; fonts in `assets/fonts/`.
- `imoogi-toolchain` subcommands `fetch` (online download into `vendor/toolchains/`, writing `toolchains.lock.json`), `setup` (offline verify, stage and activate into `.local/`) and `version` (`cmd/imoogi-toolchain/main.go:103-107`). Desired state lives in `toolchains.json`; languages implemented in `internal/lang/{golang,typescript}`.
- `imoogi-provenance` subcommands `verify`, `generate`, `record-git-source ID COMMIT REF` (`cmd/imoogi-provenance/main.go:14`), over `provenance/*.json` and `vendor-manifest.json`.
- Makefile entry points: `build-all` and per-CLI `build-*` targets, `verify-vendor`, `test` (`verify-vendor test-elisp test-go test-shell`), and `ci-local` (`fmt-check lint test`), which is the pre-push hook (`Makefile:110-194`).

**Structure.** Go packages `internal/{cli,config,fetch,setup,artifact,activation,lang,provenance}`. `fetch` is the only networked step and is online-only; `setup` never touches the network.

**Data and state.** `vendor/`, `provenance/`, `vendor-manifest.json`, `toolchains.json`, `toolchains.lock.json` in the repository; staged toolchains in `.local/`; built binaries in `bin/` (git-ignored).

**Tests.** `internal/{cli,config,fetch,setup,artifact,activation}/*_test.go`, `internal/lang/{golang,typescript}`, `internal/provenance/*_test.go` (4 files); `tests/setup-toolchain-test.sh`; `tests/run.sh` includes an offline-boot smoke test (`tests/run.el`, `boot-health.el`).

**Known gaps.** Card `t3` (see domain 7) is also a supply-chain item: the missing servers need a vendoring decision.

---

## 9. Agent IPC

**Purpose.** Let external coding agents (Claude Code, Codex and others) push events into the running Emacs: notifications, opening files, jumping to a location, and announcing artifacts or finished tasks.

**Implemented capabilities**

- `imoogi-agent` subcommands `message`, `open-file`, `goto`, `artifact` (`--type`, `--title`) and `finish` (`--artifact`), with common options `--project`, `--session`, `--timeout` (`internal/agentipc/request.go:81-91`).
- Receiver in Emacs (`modules/development/30-agent.el`): protocol v1 event types `message`, `open-file`, `goto-location`, `artifact-created`, `task-finished` (`:32-33`); two entry points, `imoogi-agent-receive-file` (`:384`) and `imoogi-agent-receive-json` (`:394`), share one validation and dispatch path and return `"ok"` or `"error:<reason>"`.
- Safety rules in the receiver: events larger than 1 MiB are rejected (`:19`), duplicate JSON keys are rejected (`:124`, `:155-157`), and no event value is ever evaluated as code (header comment, `:7-9`).
- File-trust check (SPEC-AGENTIPC-002): `imoogi-agent-receive-file` rejects an event file whose owner is not the current user or which is group/world-writable with `error:untrusted`, before reading it (`:75-85`); `imoogi-agent-receive-json` is unaffected.
- Payload size cap: a file to open that is larger than 100 MiB (`imoogi-agent-max-payload-bytes`, `:22`) rejects the whole event with `error:payload-too-large` and nothing is displayed (`:180-192`). Both new reasons map to CLI exit 3.
- CLI side (`internal/agentipc/emacsclient.go`): the 0600 event file lives in a per-call 0700 private directory (`os.MkdirTemp`, `:89-95`) removed on every path; a bare `EMACSCLIENT` (no `/`) is resolved only via PATH and fails with exit 1 if the match is in `.`, a relative or empty PATH entry, or is not absolute (`findEmacsclient`, `:126-134`).
- Every event is appended to the session log buffer `*imoogi-agent*` (`:29`, `:245`).

**Structure.** The CLI writes one JSON event to a temporary file and runs `emacsclient` to call `imoogi-agent-receive-file` (`internal/agentipc/emacsclient.go`). It depends on the Emacs server started in `13-system.el`. `make build-agent` builds the binary.

**Data and state.** Transient event files only; the log is an in-session buffer.

**Tests.** `tests/agent-test.el`; `internal/agentipc/{emacsclient,request}_test.go`; `cmd/imoogi-agent/main_test.go`.

**Known gaps** (residual risks from SPEC-AGENTIPC-002 plan.md)

- R-5: if the CLI is killed by a signal, its private temp directory is left behind (a 0700 directory holding a 0600 file; a late read still faces the trust check).
- R-7: a user-owned symlink to a user-owned file with attacker-chosen content still passes the trust check on a shared `/tmp` (display/notice/log impact only).
- R-8: the plain PATH lookup of `emacsclient` (`emacsclient.go:145`) has no absolute-path guard, so it may accept a relative match under `GODEBUG=execerrdot=0`.

---

## Cross-domain seams

| Direction | Mechanism | Used by |
|---|---|---|
| Emacs calls a Go CLI, one request per call | JSON on stdin, JSON on stdout (`call-process-region`) | `imoogi-anki` (domain 6), `imoogi-notes` (domain 3) |
| Emacs runs a long-lived Go process | `make-process` (asynchronous subprocess) | `imoogi-clip` (domain 4) |
| Emacs runs a Go server and talks HTTP | Bootstrap JSON line, then loopback HTTP and a browser WebSocket with a session token | `imoogi-org-preview` (domain 5) |
| Go talks to an external app | AnkiConnect HTTP on `127.0.0.1:8765` | `imoogi-anki` (domain 6) |
| An external agent calls into Emacs | `imoogi-agent` writes a temp JSON file and calls `emacsclient` | `imoogi-agent` (domain 9) |
| Build and test time only | Makefile and shell, not called by Emacs at runtime | `imoogi-toolchain`, `imoogi-provenance` (domain 8) |
| Emacs-only storage | SQLite file | local flashcards (domain 6b) |

Shared conventions across these seams: CLI paths are defcustoms (`imoogi-anki-binary-path`, `imoogi-project-notes-command`, `imoogi-clipboard-command`, `imoogi-org-preview-command`); and a missing binary or unsupported platform degrades to a stable error result rather than breaking boot.

## See also

- [product.md](product.md) — purpose, audience, feature areas, constraints
- [structure.md](structure.md) — directory tree, module load order, package boundary rules
- [tech.md](tech.md) — languages, dependencies, build and test toolchain
- [codemaps/overview.md](codemaps/overview.md), [codemaps/modules.md](codemaps/modules.md), [codemaps/entry-points.md](codemaps/entry-points.md), [codemaps/data-flow.md](codemaps/data-flow.md), [codemaps/dependencies.md](codemaps/dependencies.md)
