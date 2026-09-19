# Structure

## Overview

imoogi-emacs is a mixed-stack monorepo: an Emacs Lisp configuration (the primary product) plus six companion Go CLIs that the configuration invokes as subprocesses. Both halves are designed to be fully air-gapped once vendored.

As of the 2026-09-19 module reorganization, the Emacs modules live in four packages — `general/`, `project/`, `org/`, `development/` — instead of one flat folder. The numeric prefixes were kept as module identifiers, so command names, feature names, key bindings, and the `imoogi-failed-modules` entries did not change.

Measured at HEAD `efa3567`: 53 Emacs Lisp files under `modules/`, 34 Go packages (`go list ./...`), 50 `.el` files under `tests/` (45 `*-test.el` cases plus the runner, benchmarks and boot helpers), 47 Go `*_test.go` files.

## Directory Tree

```
imoogi-emacs/
├── early-init.el              # First file Emacs loads — startup performance tuning
├── boot.el                    # Entry point: package-user-dir, imoogi-require,
│                              #   the explicit module load list, imoogi-reload
├── packages.el                # SSOT package manifest (imoogi-required-packages)
├── packages.lock              # Frozen version audit — human-readable, not authoritative
├── Makefile                   # Install, build, vendoring, provenance and test entry points
│
├── modules/                   # Emacs Lisp feature modules, grouped into four packages.
│   │                          #   boot.el's explicit list keeps the original load order,
│   │                          #   which is dependency order.
│   ├── general/               # Keys, input method, completion, UI, editing, system
│   │   ├── 00-defaults.el
│   │   ├── 01-keys.el
│   │   ├── 02-completion.el
│   │   ├── 03-which-key.el
│   │   ├── 05-transient.el
│   │   ├── 09-autorevert.el
│   │   ├── 10-theme.el
│   │   ├── 11-editing.el
│   │   ├── 12-navigation.el
│   │   ├── 13-system.el
│   │   └── 21-native-compile.el
│   ├── project/               # project.el, perspective, Treemacs, Git, tabs, notes
│   │   ├── 04-projects.el
│   │   ├── 06-git.el
│   │   ├── 07-treemacs.el
│   │   ├── 22-tabs.el
│   │   └── 26-project-notes.el
│   ├── org/                   # Org/Markdown authoring, preview, clipboard, study cards
│   │   ├── 08-obsidian.el
│   │   ├── 14-org.el
│   │   ├── 15-markdown.el
│   │   ├── 23-org-preview.el
│   │   ├── 24-anki.el
│   │   ├── 25-flashcards.el
│   │   ├── 28-clipboard.el
│   │   ├── anki/              # 10 libraries behind 24-anki.el
│   │   │   ├── imoogi.el
│   │   │   ├── imoogi-config.el
│   │   │   ├── imoogi-error.el
│   │   │   ├── imoogi-process.el
│   │   │   ├── imoogi-props.el
│   │   │   ├── imoogi-scan.el
│   │   │   ├── imoogi-setup.el
│   │   │   ├── imoogi-target-scan.el
│   │   │   ├── imoogi-targets.el
│   │   │   └── imoogi-writeback.el
│   │   └── flashcards/        # 4 libraries behind 25-flashcards.el
│   │       ├── imoogi-flashcards-core.el
│   │       ├── imoogi-flashcards-org.el
│   │       ├── imoogi-flashcards-repository.el
│   │       └── imoogi-flashcards-review.el
│   └── development/           # Formatting, LSP, major modes, folding, terminal, gptel
│       ├── formatting.el      # Unnumbered; loaded right after general/01-keys
│       ├── 16-elisp.el
│       ├── 17-lsp.el          # Auto-discovers and loads development/lang/*.el
│       ├── 18-languages.el
│       ├── 19-folding.el
│       ├── 20-terminal.el
│       ├── 27-gptel.el
│       └── lang/              # 9 per-language LSP configs, auto-discovered
│           ├── bash.el
│           ├── clojure.el
│           ├── go.el
│           ├── java.el
│           ├── javascript.el
│           ├── kotlin.el
│           ├── python.el
│           ├── rust.el
│           └── typescript.el
│
├── cmd/                       # Six Go CLI entry points
│   ├── imoogi-anki/           # Org → Anki sync back end (JSON on stdin/stdout)
│   ├── imoogi-clip/           # Clipboard inspection and asset ingestion
│   ├── imoogi-notes/          # Verified project/study note move
│   ├── imoogi-org-preview/    # Loopback HTTP preview server
│   ├── imoogi-provenance/     # Vendored-artifact provenance generate/verify
│   └── imoogi-toolchain/      # LSP toolchain fetch/setup
│
├── internal/                  # Go business logic (12 top-level packages)
│   ├── activation/            # CLI self-bootstrap cross-compile spec
│   ├── anki/                  # ankiconnect, hashing, media, model, orgdoc,
│   │                          #   planner, protocol, registry
│   ├── artifact/              # Artifact verification and staging primitives
│   ├── cli/                   # imoogi-toolchain subcommand dispatch
│   ├── clipboard/             # Platform clipboard adapters and asset service
│   ├── config/                # Toolchain desired-state config and target constants
│   ├── fetch/                 # Online-only toolchain download
│   ├── lang/                  # Per-language toolchain logic (golang, typescript)
│   ├── notemove/              # Copy + SHA-256 verify + switch note move
│   ├── orgpreview/            # assets, parser, protocol, render, server, session
│   ├── provenance/            # Manifest generation and verification
│   └── setup/                 # Offline-only verify, stage and activate
│
├── vendor/                    # Air-gapped artifacts
│   ├── elpa/                  # Vendored Emacs packages (scripts/vendor.el output)
│   ├── ghostel-module/        # Prebuilt native terminal module
│   ├── tree-sitter/           # tree-sitter grammars (scripts/build-grammars.sh)
│   └── toolchains/            # imoogi-toolchain fetch output
│
├── provenance/                # Per-domain provenance manifests
│   ├── sources.json           # Declared roots, excludes and component sources
│   ├── elpa.json
│   ├── fonts.json
│   ├── ghostel.json
│   ├── metadata.json
│   ├── toolchains.json
│   └── tree-sitter.json
├── vendor-manifest.json       # Manifest index + per-file SHA-256 (generate/verify)
│
├── scripts/
│   ├── vendor.el              # Populates vendor/elpa/ from packages.el (online only)
│   ├── build-grammars.sh      # Builds tree-sitter grammars (online only)
│   ├── install.sh             # Links this repository to ~/.emacs.d
│   ├── install-emacs.sh
│   ├── install-tmux.sh
│   ├── setup-toolchain.sh
│   └── imoogi-editor
│
├── tests/                     # ERT suite, Go/shell tests and the runner
│   ├── run.sh                 # 3-phase orchestrator
│   ├── run.el                 # Loads boot.el in a temp user-emacs-directory
│   ├── boot-health.el         # Boot capture/assert helper used by run.sh
│   ├── benchmarks/
│   └── docker/
│
├── templates/project-notes/   # Org templates for project and study notes
├── assets/fonts/              # Bundled fonts, copied locally on first boot
├── tmux/                      # Optional tmux configuration
├── bin/                       # Built CLI binaries (git-ignored)
│
├── toolchains.json            # Desired-state manifest for LSP toolchain artifacts
├── toolchains.lock.json       # Resolved lockfile, written by `imoogi-toolchain fetch`
├── go.mod / go.sum / go.work  # Go module (github.com/karohani/imoogi-emacs, go 1.26)
│
├── docs/
│   ├── toolchains.md
│   ├── clipboard-platform-support.md
│   └── refactoring/module-packages.md
│
├── README.md                  # Human-facing overview (Korean)
├── ARCHITECTURE.md            # Architecture rationale and module-addition guide
├── AGENTS.md                  # AI-agent operating constraints (air-gap-first)
└── CLAUDE.md                  # MoAI orchestrator directive
```

## Directory Purposes

| Directory | Purpose |
|---|---|
| `modules/general/` | Key bindings, Korean input, completion, UI, general editing and common system settings. |
| `modules/project/` | `project.el`, perspective, Treemacs, Git, tab-bar, and project/study notes. |
| `modules/org/` | Org and Markdown authoring conventions, browser preview, clipboard assets, Anki sync and local flashcards. The implementation libraries live in `org/anki/` and `org/flashcards/`. |
| `modules/development/` | Formatting, common LSP wiring, major modes, folding, terminal and gptel. |
| `modules/development/lang/` | Per-language LSP configuration, auto-discovered by `17-lsp.el`. |
| `cmd/` | Six Go CLI entry points, each a thin `main()` over an `internal/` package. |
| `internal/` | Go business logic, following standard Go internal-package conventions. |
| `vendor/` | Air-gapped artifacts. `elpa/`, `ghostel-module/` and `tree-sitter/` are committed — the committed directory itself is the version lock. `toolchains/` holds the `imoogi-toolchain fetch` output. |
| `provenance/` | Declared sources and per-domain manifests for every vendored external file. |
| `scripts/` | One-directional (online → offline) vendoring and install tooling. |
| `templates/project-notes/` | Org templates used when a project or study note is created. |
| `assets/fonts/` | Bundled fonts copied locally on first boot; no download required. |
| `tests/` | Syntax validation, an offline-boot smoke test, the ERT suite, Go tests and shell tests. |
| `bin/` | Build output of the `make build-*` targets. Git-ignored (`.gitignore` line 29). |
| `docs/` | Human-facing documentation for the toolchain workflow, clipboard platform support and the module reorganization. |

## Module Load Order

`boot.el` carries one explicit list of relative module paths. The order crosses package boundaries deliberately: it is the original dependency order, preserved through the reorganization.

| # | Module | # | Module |
|---|---|---|---|
| 1 | `general/00-defaults` | 16 | `org/14-org` |
| 2 | `general/01-keys` | 17 | `org/23-org-preview` |
| 3 | `development/formatting` | 18 | `org/15-markdown` |
| 4 | `general/02-completion` | 19 | `development/16-elisp` |
| 5 | `general/03-which-key` | 20 | `development/17-lsp` |
| 6 | `project/04-projects` | 21 | `development/18-languages` |
| 7 | `general/05-transient` | 22 | `development/19-folding` |
| 8 | `project/06-git` | 23 | `development/20-terminal` |
| 9 | `project/07-treemacs` | 24 | `general/21-native-compile` |
| 10 | `org/08-obsidian` | 25 | `project/22-tabs` |
| 11 | `general/09-autorevert` | 26 | `org/24-anki` |
| 12 | `general/10-theme` | 27 | `org/25-flashcards` |
| 13 | `general/11-editing` | 28 | `project/26-project-notes` |
| 14 | `general/12-navigation` | 29 | `development/27-gptel` |
| 15 | `general/13-system` | 30 | `org/28-clipboard` |

30 entries: 29 numbered modules plus the unnumbered `development/formatting`, which is loaded third so that every later module sees the shared formatting setup.

Load-order dependency examples:

- `development/formatting` is loaded immediately after `general/01-keys` because formatting ownership is shared by the language modules that come later.
- `general/10-theme` must load after `project/07-treemacs` for icon integration.
- `org/23-org-preview` and `org/24-anki` depend on `org/14-org`; both were appended after the existing numbers rather than renumbering the sequence.
- `general/21-native-compile` is deliberately late — it retroactively native-compiles everything already loaded.

## Package Boundary Rules

- The numeric prefix is a module identifier, not a directory-ordering key. `imoogi-failed-modules` entries, feature names and command names are unchanged by the reorganization.
- `boot.el`'s explicit list is the only load order. Folders are never loaded wholesale in directory order, and no duplicate configuration is left behind at the old paths.
- Configuration files, registries, notes, study IDs and user data are not moved by the reorganization. Only the previous built-in LSP default path was switched to `modules/development/lang/`; a user-specified LSP folder is kept as-is.
- An existing session picks up the new layout with `imoogi-reload`.
- External packages and the Go CLIs were out of scope for the reorganization.

## Module Organization Convention

Each `modules/<package>/<name>.el` file follows a fixed per-module contract (see `AGENTS.md` §2 and `ARCHITECTURE.md` for the authoritative rationale):

1. Immediately after `;;; Code:`, call `(imoogi-require "NN-name" 'pkg1 'pkg2 ...)` — a `boot.el` helper that uses `locate-library` to confirm every needed package is present in `vendor/` or built in, and signals an error if not. The string argument keeps the original module identifier even though the file moved.
2. `boot.el`'s loader wraps each module load in `condition-case`, so a failed `imoogi-require` (or any load error) only skips that one module — the rest of the load chain continues, with a `display-warning`, and the name is appended to `imoogi-failed-modules`.
3. Configuration is done via `use-package`: built-ins use `:ensure nil`, vendored packages use `:ensure t` (safe — no network fetch, since the package is already installed locally in `vendor/elpa/`).
4. The file ends with `(provide 'imoogi-NAME)`.
5. The new `"<package>/NN-name"` string is registered in `boot.el`'s `dolist`, positioned according to dependency order.
6. If new packages are needed, they are added to `imoogi-required-packages` in `packages.el`, and `scripts/vendor.el` is re-run on an online machine.

### The `development/lang/` auto-discovery pattern

Language-specific LSP configs live in `modules/development/lang/` rather than in the numbered sequence. `17-lsp.el` auto-discovers, sorts and loads each file there, with per-file failure isolation. Adding LSP support for a new language requires only dropping a new file into that folder — no `boot.el` edit and no module-number change. `24-anki.el` uses the same pattern for `modules/org/anki/`.

## Where User Data Lives

No user data is stored inside this repository. The repository holds configuration, vendored artifacts and templates only.

- Emacs session state, caches and registries live under the Emacs user directory (`~/.emacs.d/.cache/`), including backups, auto-saves and the mounted-note-root registry `.cache/project-notes-mounted-roots.json`.
- Project and study notes live in the user's own note folders, which may be on a removable device. Each note folder identifies itself with its own `.imoogi-project.json`, so the same device plugged into another machine can be rediscovered from the folder metadata.
- The Anki diagnostic log path is user-configurable (`imoogi-anki-log-file`) and is passed to the Go back end through the `IMOOGI_ANKI_LOG` environment variable.
- Toolchain artifacts are staged under `.local/` by `imoogi-toolchain setup`.

## Key File Locations

- **Boot sequence**: `early-init.el` → `boot.el` (the module `dolist`) → `modules/<package>/*.el`
- **Package manifest (SSOT)**: `packages.el` (`imoogi-required-packages`)
- **Package version audit**: `packages.lock` (not authoritative — `vendor/elpa/` is)
- **Provenance**: `provenance/sources.json` (declared sources) + `vendor-manifest.json` (index and per-file hashes)
- **Toolchain desired state / lockfile**: `toolchains.json` / `toolchains.lock.json`
- **Go module root**: `go.mod` (`github.com/karohani/imoogi-emacs`, Go 1.26)
- **Test entry points**: `tests/run.sh`, `go test ./...`, `make test`, `make ci-local`
