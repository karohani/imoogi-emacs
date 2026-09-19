# Codemap — Modules

Every Emacs Lisp file under `modules/`, by package. 53 files: 29 numbered modules, one unnumbered (`development/formatting.el`), 9 per-language LSP configs, 10 Anki libraries and 4 flashcard libraries.

## `modules/general/` — 11 modules

| Module | Purpose |
|---|---|
| `00-defaults.el` | Better defaults plus session persistence (recentf, savehist, saveplace); owns scroll and general editing defaults. |
| `01-keys.el` | Korean input and key translation, including the `S-SPC` hangul toggle. |
| `02-completion.el` | Completion stack: vertico, orderless, marginalia, embark, consult, corfu, cape. |
| `03-which-key.el` | which-key (built in as of Emacs 30). |
| `05-transient.el` | Transient menu definitions; also pulls in ace-window. Replaced the earlier hydra-based menu module. |
| `09-autorevert.el` | Auto-revert buffers on external file change. |
| `10-theme.el` | doom-themes, doom-modeline, nerd-icons. Must load after Treemacs for icon integration. |
| `11-editing.el` | undo-fu(+session), yasnippet, and related editing enhancements. |
| `12-navigation.el` | avy, helpful, bufferfile. |
| `13-system.el` | System integration: exec-path-from-shell, buffer-terminator, persist-text-scale. |
| `21-native-compile.el` | compile-angel; loaded late so it retroactively native-compiles what is already loaded. |

## `modules/project/` — 5 modules

| Module | Purpose |
|---|---|
| `04-projects.el` | `project.el` plus perspective workspaces. |
| `06-git.el` | Magit and diff-hl. |
| `07-treemacs.el` | Treemacs and its companions (icons-dired, magit integration) — the IntelliJ-style tool window. |
| `22-tabs.el` | `tab-bar` as the window layer. |
| `26-project-notes.el` | Project-scoped and study Org notes: creation from templates, registration, agenda wiring, removable-root registration and the verified move through `imoogi-notes`. Defines `imoogi-project-notes-command`. |

## `modules/org/` — 7 modules + 14 libraries

| Module | Purpose |
|---|---|
| `08-obsidian.el` | Obsidian-style note integration. |
| `14-org.el` | Org configuration; the base every other Org-side module depends on. |
| `15-markdown.el` | Markdown configuration (markdown-mode, markdown-toc, edit-indirect). |
| `23-org-preview.el` | Live Org/Markdown HTML preview client; starts and drives the `imoogi-org-preview` server and the browser. |
| `24-anki.el` | One-way Org → Anki sync: adds `modules/org/anki/` to `load-path` and layers this repository's conveniences (completion, key bindings, menu) on top. |
| `25-flashcards.el` | Emacs-native SQLite-backed flashcard fallback for hosts without Anki. Shares no state with `24-anki.el`. |
| `28-clipboard.el` | Clipboard assets for Org and Markdown: inspection, staging leases and asset insertion through `imoogi-clip`. |

### `modules/org/anki/` — 10 libraries

| Library | Purpose |
|---|---|
| `imoogi.el` | Entry point: the sync command, user options including `imoogi-anki-log-file`. |
| `imoogi-config.el` | Configuration file read and write. |
| `imoogi-error.el` | Error code → user-facing message and remediation table. |
| `imoogi-process.el` | Subprocess invocation for the sync-run request/response; sets `IMOOGI_ANKI_LOG`. |
| `imoogi-props.el` | Nearest-wins property resolution across the Org tree. |
| `imoogi-scan.el` | Recursive sync-root traversal. |
| `imoogi-setup.el` | The `imoogi-anki-setup` interactive command. |
| `imoogi-target-scan.el` | Multi-target Org scan grouping. |
| `imoogi-targets.el` | Host-wide Anki sync target registration. |
| `imoogi-writeback.el` | Buffer-mediated `ANKI_NOTE_ID` write-back. |

### `modules/org/flashcards/` — 4 libraries

| Library | Purpose |
|---|---|
| `imoogi-flashcards-core.el` | Core card model and scheduler. |
| `imoogi-flashcards-org.el` | Card extraction from Org. |
| `imoogi-flashcards-repository.el` | SQLite repository. |
| `imoogi-flashcards-review.el` | Review UI. |

## `modules/development/` — 7 modules + 9 language configs

| Module | Purpose |
|---|---|
| `formatting.el` | Shared code formatting. Unnumbered and loaded third, so every later module sees one formatting owner. |
| `16-elisp.el` | Elisp development: aggressive-indent, highlight-defined, paredit and companions. |
| `17-lsp.el` | Eglot and xref common configuration; auto-discovers, sorts and loads `modules/development/lang/*.el` with per-file failure isolation. Consumes the `.local/bin` tree produced by `imoogi-toolchain setup`. |
| `18-languages.el` | Major modes for roughly twenty file types (git-modes, yaml, dockerfile, gnuplot, lua, jinja2, csv, go, rust, crontab, nginx, hcl, nix, fish, vimrc, jenkinsfile, clojure, kotlin, typescript, web). |
| `19-folding.el` | Code folding (kirigami, outline-indent). |
| `20-terminal.el` | ghostel terminal (libghostty-vt native module), with Korean input support. |
| `27-gptel.el` | gptel against a LiteLLM gateway, including private-CA configuration. |

### `modules/development/lang/` — 9 per-language LSP configs

`bash.el`, `clojure.el`, `go.el`, `java.el`, `javascript.el`, `kotlin.el`, `python.el`, `rust.el`, `typescript.el`. Each calls `imoogi-require` with its original `"lsp/<language>"` identifier — preserved through the reorganization — and wires Eglot for that language's major modes. `typescript.el` covers both TypeScript and TSX.

## Go Packages

### `cmd/` — 6 entry points

| Package | Purpose |
|---|---|
| `cmd/imoogi-anki` | Reads one JSON request from stdin, writes one JSON response to stdout, once per sync run. No user interaction. |
| `cmd/imoogi-clip` | Decodes a clipboard request, runs `clipboard.Service`, encodes the response. |
| `cmd/imoogi-notes` | Decodes a move request, runs `notemove.Move`, encodes the response. |
| `cmd/imoogi-org-preview` | Parses the listen flags, constructs `orgpreview.Server`, listens on loopback and serves until interrupted. |
| `cmd/imoogi-provenance` | Dispatches `verify`, `generate` and `record-git-source`. |
| `cmd/imoogi-toolchain` | Wires `cli.Run` with the `fetch` and `setup` hooks. |

### `internal/` — 27 packages

| Package | Purpose |
|---|---|
| `activation` | CLI self-bootstrap cross-compile spec used by `fetch`. |
| `anki/ankiconnect` | The only component permitted to talk to AnkiConnect; client, errors and model. |
| `anki/hashing` | Content hash used to decide whether a note changed. |
| `anki/media` | Media pass over rendered fields. |
| `anki/model` | Owns imoogi's two note types: names, fields, CSS, deck-class scripts and installation. |
| `anki/orgdoc` | Renders a sync target's title and body into Anki field values, including math. |
| `anki/planner` | The pure decision layer: creates, updates, deletes, duplicate handling, field names, migration. |
| `anki/protocol` | The JSON documents exchanged with the Emacs Lisp layer. |
| `anki/registry` | Persists the derived per-sync-root state. |
| `artifact` | Artifact verification and staging primitives. |
| `cli` | `imoogi-toolchain` subcommand dispatch. |
| `clipboard` | Clipboard adapters (macOS AppKit via cgo, an unsupported fallback), asset ownership and the request/response service. |
| `config` | Toolchain desired-state config and the `TargetOS`/`TargetArch` constants. |
| `fetch` | Online-only toolchain download and lockfile write. |
| `lang`, `lang/golang`, `lang/typescript` | Per-language toolchain logic (gopls; tsc and typescript-language-server). |
| `notemove` | Copy, SHA-256 verify and switch for the note move. |
| `orgpreview` + `assets`, `parser`, `protocol`, `render`, `server`, `session` | Org/Markdown intermediate representation, parsing, rendering, static assets, the loopback HTTP server and its session/token handling. |
| `provenance` | Manifest model, generation, verification and git-ignore awareness. |
| `setup` | Offline-only verify, stage, probe and activate. |
