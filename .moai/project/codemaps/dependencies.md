# Codemap — Dependencies

## Emacs Module Dependencies

There is no dependency resolver. `boot.el` carries one explicit ordered list, and that order *is* the dependency graph. A module that needs another to have run already is placed after it.

### Load order

```
 1 general/00-defaults          16 org/14-org
 2 general/01-keys              17 org/23-org-preview
 3 development/formatting       18 org/15-markdown
 4 general/02-completion        19 development/16-elisp
 5 general/03-which-key         20 development/17-lsp
 6 project/04-projects          21 development/18-languages
 7 general/05-transient         22 development/19-folding
 8 project/06-git               23 development/20-terminal
 9 project/07-treemacs          24 general/21-native-compile
10 org/08-obsidian              25 project/22-tabs
11 general/09-autorevert        26 org/24-anki
12 general/10-theme             27 org/25-flashcards
13 general/11-editing           28 project/26-project-notes
14 general/12-navigation        29 development/27-gptel
15 general/13-system            30 org/28-clipboard
```

### Load-order constraints that matter

| Constraint | Reason |
|---|---|
| `development/formatting` at position 3 | Formatting ownership is consolidated in one module; the language modules later in the list assume it is already set up. |
| `general/10-theme` after `project/07-treemacs` | Icon integration. |
| `org/23-org-preview`, `org/24-anki`, `org/25-flashcards`, `org/28-clipboard` after `org/14-org` | All operate on Org buffers and the Org element API. |
| `project/26-project-notes` after `org/14-org` and `project/04-projects` | Notes use Org, org-agenda, org-id and the project API. |
| `general/21-native-compile` late | compile-angel retroactively native-compiles what has already been loaded. |

Positions 17 and 26-30 are appended after the existing numbers rather than renumbered: the number is a stable module identifier, and renumbering would change `imoogi-failed-modules` entries and every `imoogi-require` call site.

### Package-level dependencies declared per module

Each module's `imoogi-require` line is the authoritative statement of what it needs.

| Module | Declares |
|---|---|
| `general/00-defaults` | recentf, savehist, saveplace |
| `general/02-completion` | vertico, orderless, marginalia, embark, … |
| `general/03-which-key` | which-key |
| `general/05-transient` | transient, ace-window |
| `general/10-theme` | doom-themes, doom-modeline, nerd-icons |
| `general/11-editing` | undo-fu, undo-fu-session, yasnippet, … |
| `general/12-navigation` | avy, helpful, bufferfile |
| `general/13-system` | exec-path-from-shell, buffer-terminator, … |
| `general/21-native-compile` | compile-angel |
| `project/04-projects` | project, perspective |
| `project/06-git` | magit, diff-hl |
| `project/07-treemacs` | treemacs, treemacs-icons-dired, … |
| `project/22-tabs` | tab-bar |
| `project/26-project-notes` | cl-lib, json, org, org-agenda, org-id, … |
| `org/08-obsidian` | obsidian |
| `org/14-org` | org, org-appear, hl-line, calendar, transient |
| `org/15-markdown` | markdown-mode, markdown-toc, edit-indirect, org, hl-line |
| `org/23-org-preview` | org, org-element, url, url-util, json, browse-url, seq |
| `org/24-anki` | org, json, seq, url, url-http |
| `org/25-flashcards` | org, sqlite, seq |
| `org/28-clipboard` | cl-lib, json, subr-x, url-util |
| `development/formatting` | (no external packages) |
| `development/16-elisp` | aggressive-indent, highlight-defined, paredit, … |
| `development/18-languages` | git-modes, yaml-mode, dockerfile-mode, gnuplot, … |
| `development/19-folding` | kirigami, outline-indent |
| `development/20-terminal` | ghostel |
| `development/27-gptel` | gptel, gptel-transient, gptel-anthropic, … |
| `development/lang/*` | eglot plus that language's major mode |

A declared package that cannot be located makes `imoogi-require` signal, `boot.el` skip that one module, and the name land in `imoogi-failed-modules`. Nothing else in the chain is affected — `org/25-flashcards` skipping on a host without `sqlite` is the designed example.

### Library folders

Two modules load a folder of libraries instead of declaring them in `boot.el`:

- `development/17-lsp.el` → `modules/development/lang/*.el` (auto-discovered, sorted, per-file failure isolation)
- `org/24-anki.el` → `modules/org/anki/*.el` (added to `load-path`, then required)

`org/25-flashcards.el` similarly owns `modules/org/flashcards/`.

## Go Package Graph

### `cmd → internal`

```
cmd/imoogi-anki        → internal/anki/{protocol, planner, orgdoc, model,
                                        media, hashing, registry, ankiconnect}
cmd/imoogi-clip        → internal/clipboard
cmd/imoogi-notes       → internal/notemove
cmd/imoogi-org-preview → internal/orgpreview (+ parser, protocol, render,
                                              server, session, assets)
cmd/imoogi-provenance  → internal/provenance
cmd/imoogi-toolchain   → internal/cli, internal/fetch, internal/setup
```

Each `main()` is thin: argument or stdin decoding, one call into the owning package, one encode of the result. No business logic lives in `cmd/`.

### Internal layering

- `internal/cli` is a dispatch layer only; it receives `fetch` and `setup` as hooks from `main`, so neither is imported by the dispatcher.
- `internal/fetch` (online) and `internal/setup` (offline) both build on `internal/config`, `internal/artifact` and `internal/lang/*`; `internal/activation` supplies the self-bootstrap cross-compile spec to `fetch`.
- `internal/anki` is layered as pure decision (`planner`) over rendering (`orgdoc`, `model`, `media`, `hashing`) with exactly one I/O boundary (`ankiconnect`) and one persistence boundary (`registry`). `protocol` is shared with the Emacs side and imports nothing from the rest.
- `internal/clipboard` isolates platform code behind build-tagged adapter files, so the unsupported path compiles everywhere.
- `internal/provenance` is self-contained: model, generate, verify, plus git-ignore awareness so build output does not fail verification.

### External Go dependencies

One direct: `github.com/niklasfasching/go-org v1.9.1` (Org parsing for the preview server). One indirect: `golang.org/x/net`. Everything else is standard library.

### Elisp → Go

The configuration never imports Go code; it resolves a binary and runs it (see `codemaps/overview.md` § The Elisp ↔ Go Boundary). Resolution in all three callers is "`PATH` first, then `bin/<name>` inside the repository", and the executable is a user option (`imoogi-project-notes-command`, `imoogi-clipboard-command`, `imoogi-org-preview-command`), so the dependency is late-bound and replaceable.
