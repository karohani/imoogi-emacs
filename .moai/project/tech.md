# Tech

## Technology Stack Overview

imoogi-emacs is built from two coordinated stacks:

- **Emacs Lisp** (dominant — 53 files under `modules/`, in four packages): the configuration itself, targeting **Emacs 30.x specifically**. For example, `which-key` is deliberately omitted from `packages.el` because it ships built-in as of Emacs 30.
- **Go 1.26** (`github.com/karohani/imoogi-emacs`): six CLIs the configuration invokes as subprocesses. The module has one direct third-party dependency, `github.com/niklasfasching/go-org v1.9.1` (with `golang.org/x/net` as an indirect), and a correspondingly small `go.sum`.

The unifying design principle across both stacks is **air-gap safety**: nothing in the boot path or the offline `setup` path may touch the network. This is the single most important constraint on the project (per `AGENTS.md`, "가장 중요한 제약").

## The Air-Gap Constraint

`boot.el` sets `package-user-dir` to `vendor/elpa/`, calls `package-initialize`, and sets `use-package-always-ensure` to `nil`. `package-refresh-contents` is never called at runtime, so the archive list in `boot.el` matters only during online vendoring. A git clone with `vendor/` populated is, by itself, a complete working installation.

Every module declares its prerequisites with `(imoogi-require "NN-name" 'pkg ...)`. A missing package signals an error that `boot.el`'s `condition-case` catches, so a single unsatisfiable module is skipped and recorded in `imoogi-failed-modules` rather than breaking the boot. `tests/assert-boot.el` and `tests/boot-health.el` use that variable as the pass/fail signal for installation verification.

## Emacs Dependencies

All Emacs packages are declared in `packages.el` (`imoogi-required-packages`), vendored into `vendor/elpa/`, and version-audited in `packages.lock` (a generated audit trail listing 88 packages resolved against Emacs 30.2 — human-readable, not authoritative).

| Category | Packages |
|---|---|
| Completion stack | vertico, orderless, marginalia, embark (+embark-consult), consult, corfu, cape |
| Workspace | perspective (combined with built-in `project.el`) |
| Menus / window | transient, ace-window |
| Git | magit (+magit-section, with-editor) |
| File tree | treemacs (+treemacs-icons-dired, treemacs-magit) |
| Notes | obsidian |
| Theme / UI polish | doom-themes, doom-modeline, nerd-icons |
| Editing | undo-fu, undo-fu-session, yasnippet, yasnippet-snippets, apheleia, dumb-jump, stripspace |
| Navigation | avy, helpful, diff-hl, bufferfile |
| System | exec-path-from-shell, buffer-terminator, persist-text-scale |
| Org / Markdown | org-appear, markdown-mode, markdown-toc, edit-indirect |
| Elisp development | aggressive-indent, highlight-defined, paredit, page-break-lines, elisp-refs |
| Language modes | git-modes, yaml-mode, dockerfile-mode, gnuplot, lua-mode, jinja2-mode, csv-mode, go-mode, rust-mode, crontab-mode, nginx-mode, hcl-mode, nix-mode, fish-mode, vimrc-mode, jenkinsfile-mode, clojure-mode, kotlin-mode, typescript-mode, web-mode |
| Folding | kirigami, outline-indent |
| Terminal | ghostel (native module built on libghostty-vt, vendored as a prebuilt `.dylib`) |
| AI assistant | gptel (+gptel-transient, gptel-anthropic) |
| Compile | compile-angel |

Pop-up command menus are built on `transient`; `general/05-transient.el` replaced the earlier hydra-based menu module. `hydra` is still listed in `packages.lock`, but no module loads it.

`org/25-flashcards.el` additionally requires Emacs built-in `sqlite` support. Where that is unavailable, only that one module is skipped.

### Explicitly rejected / removed dependencies

- `straight.el` — network-dependent bootstrap, incompatible with the air-gap requirement
- `projectile`, `persp-projectile`, `treemacs-projectile` — replaced by built-in `project.el`
- `treemacs-persp`, `treemacs-evil`, `persp-mode`, `evil` — removed
- `auto-package-update` — would attempt a runtime package refresh

## The Go CLIs

Six binaries under `cmd/`, built into `bin/` (git-ignored). Each is a thin `main()` over an `internal/` package; five of the six speak a one-shot JSON request/response protocol over stdin/stdout and perform no user interaction.

| Binary | Purpose | Interface |
|---|---|---|
| `imoogi-anki` | Renders Org sync targets into Anki fields and drives AnkiConnect for a one-way Org → Anki sync. Writes a diagnostic log to the path in `IMOOGI_ANKI_LOG`. | One JSON request on stdin, one JSON response on stdout |
| `imoogi-clip` | Inspects the OS clipboard (text / file list / image), and copies the selected asset into the note folder. | JSON on stdin/stdout, `--version` flag |
| `imoogi-notes` | Moves a project or study note folder to another root: copies, verifies every file with SHA-256, then switches the original. Refuses to overwrite an existing destination name. | JSON on stdin/stdout, `--version` flag |
| `imoogi-org-preview` | Loopback HTTP server rendering Org/Markdown to HTML for a browser preview. Binds `127.0.0.1` on an ephemeral port with a session token, and can print one bootstrap JSON line for the Emacs client. | HTTP on loopback + `--addr/--host/--port/--token/--print-bootstrap-json` |
| `imoogi-provenance` | `generate` writes the provenance manifests and `vendor-manifest.json`; `verify` checks that every vendored external file matches; `record-git-source ID COMMIT REF` pins a git-sourced component. | Subcommand argv |
| `imoogi-toolchain` | `fetch` (online) downloads and vendors LSP toolchain artifacts and writes `toolchains.lock.json`; `setup` (offline) verifies, stages under `.local/`, and atomically activates `.local/bin`. | Subcommand argv |

Go packages: 34 in total (`go list ./...`) — the six `cmd/` packages, 27 under `internal/` (including eight `internal/anki/*` and six `internal/orgpreview/*` sub-packages), and one under `tests/`.

### LSP toolchain runtimes (managed by `imoogi-toolchain`, not `packages.el`)

From `toolchains.json` (bundle `2026.08.23.1`, target `darwin/arm64`): Node v24.19.0, TypeScript 6.0.3, typescript-language-server 6.0.0, gopls v0.23.0.

## Provenance Verification

Every vendored external file is accounted for by a provenance manifest.

- `provenance/sources.json` (schema `imoogi-vendor-sources/v1`) declares the boundaries (`assets`, `vendor`), the roots covered, explicit excludes with reasons, ELPA source overrides, and one component record per vendored group — each naming the upstream source and the workflow that produces it.
- `imoogi-provenance generate` (`make provenance-generate`) writes the per-domain manifests (`provenance/elpa.json`, `fonts.json`, `ghostel.json`, `metadata.json`, `toolchains.json`, `tree-sitter.json`) and the index `vendor-manifest.json`, which records a SHA-256 for each covered file.
- `imoogi-provenance verify` (`make provenance-verify`, aliased `make verify-vendor`) fails if any covered file deviates. Verification skips files that git ignores.

The three ELPA archive-index components were removed from the manifest set on 2026-09-19; the archive indices are vendoring-time inputs, not shipped artifacts.

## Vendoring Workflows

Both vendoring workflows are strictly **one-directional (online → offline)** and must never be run inside the air-gapped network.

### Emacs package vendoring

1. Edit `imoogi-required-packages` in `packages.el`.
2. On an **online** machine: `emacs --batch -Q -l scripts/vendor.el` (install missing only) or `... -- upgrade` (refresh all).
3. Run `make provenance-generate`, then commit `vendor/`, `packages.el`, `packages.lock`, `provenance/` and `vendor-manifest.json`.
4. Carry the repository into the air-gapped target environment.

### tree-sitter grammars

`make grammars` (`scripts/build-grammars.sh`, online only) builds the grammars into `vendor/tree-sitter/` as platform `.dylib` files. 17 grammars are currently vendored (bash, clojure, dockerfile, go, html, java, javascript, jsdoc, json, kotlin, markdown-inline, python, regex, rust, tsx, typescript, yaml).

### Go toolchain vendoring

`make toolchain-setup` wraps `scripts/setup-toolchain.sh`. The underlying split is `imoogi-toolchain fetch` (online: reads `toolchains.json`, downloads Node/npm tarballs, builds gopls, writes `vendor/toolchains/` and `toolchains.lock.json`) and `imoogi-toolchain setup` (offline: verifies the locked artifacts, stages them under `.local/`, probes them, and atomically activates `.local/bin` as a relative symlink consumed by `modules/development/17-lsp.el`). No install or download occurs at Emacs boot.

## Build and Install Targets

`make help` lists every target. The load-bearing ones:

| Target | Effect |
|---|---|
| `install` | Links this repository to `~/.emacs.d` |
| `emacs-install` / `emacs-prewarm` / `emacs-where` | Installs Emacs (macOS via emacsformacosx.com, Linux via the distribution package), pre-runs native compilation, lists found binaries |
| `toolchain-setup` | Detects OS/arch and installs the compatible bundled language servers |
| `grammars` | Builds tree-sitter grammars (online machine only) |
| `build` | Compile check of every Go package |
| `build-all` | Builds all six CLIs into `bin/` |
| `build-toolchain`, `build-anki-bin`, `build-org-preview`, `build-clipboard`, `build-notes`, `build-provenance` | Build one CLI into `bin/` |
| `build-anki`, `install-clipboard` | Install a CLI into `ANKI_PREFIX` / `CLIPBOARD_PREFIX` |
| `provenance-generate` / `provenance-verify` / `verify-vendor` | Refresh and check vendored provenance |
| `fmt` / `fmt-check` / `lint` | `gofmt`, a formatting gate for CI, and `go vet` |
| `test-elisp` / `test-go` / `test-shell` / `test` | The three suites and the aggregate |
| `ci-local` | `fmt-check` + `lint` + `test` — the pre-push hook entry point |
| `clean-elc` / `clean` | Removes byte-compiled module output; removes the downloaded Emacs distribution cache |
| `tmux-install` / `tmux-check` | Optional tmux configuration |

## Testing

### Emacs side

`tests/run.sh` is a bash orchestrator with three phases:

1. **Syntax validation** — `check-parens` over `early-init.el`, `boot.el`, `packages.el`, and recursively over every `.el` file under `modules/` and `tests/`. The recursion is what covers the module sub-packages.
2. **Offline-boot smoke test** — loads `boot.el` in a temporary `user-emacs-directory` with `package-archives` set to `nil`, wrapped in `tests/boot-health.el` capture/assert calls, proving air-gap safety and an empty `imoogi-failed-modules`.
3. **ERT suite** — `tests/run.el` loads `boot.el` in a temporary user directory, loads every `tests/*-test.el`, and finishes with `(ert-run-tests-batch-and-exit)`.

50 `.el` files live under `tests/` — 45 `*-test.el` cases plus `run.el`, `boot-health.el`, `assert-boot.el` and the `tests/benchmarks/` scripts. They cover Anki sync (scan, props, targets, write-back, process, errors, note types, migration), flashcards (core, org extraction, repository, review, boot), clipboard, project notes, org and markdown headings, the org agenda overview, gptel, the LSP modules, the IntelliJ key bindings, the compiled menus, module layout, and boot health. `tests/benchmarks/` and `tests/docker/` hold the performance and portability harnesses.

### Go side

`go test ./...` over 47 `*_test.go` files alongside their packages.

### Shell side

`make test-shell` exercises the install and platform-detection scripts.

### CI

`.github/workflows/label-sync.yml` is the only workflow file present, and `.github/` is currently untracked in git. The enforced gate is local: `make ci-local` is the pre-push hook entry point.

## Dev Environment Requirements

- **Emacs 30.x** — vendoring is done against Emacs 30.2; the portability harness also exercises 30.1.
- **Go 1.26** for building or modifying the CLIs.
- **macOS on arm64** is the primary and fully-supported target: `internal/config` pins `TargetOS = "darwin"` / `TargetArch = "arm64"`, and the vendored ghostel module and tree-sitter grammars are macOS arm64 artifacts. The clipboard adapter has an implemented macOS AppKit path and a stable `unsupported-capability` path elsewhere; Windows and Linux clipboard cells are design gates only (`docs/clipboard-platform-support.md`). `make emacs-install` and the portability harness do cover Linux for the editor itself.
- **Network access is required only on a separate online machine** used for vendoring (`scripts/vendor.el`, `make grammars`, `imoogi-toolchain fetch`). The air-gapped target machine needs no network access at any point.

## Scope Boundaries

In scope: the Emacs configuration (`modules/`, `vendor/elpa/`) and the six Go CLIs (`cmd/`, `internal/`), all designed to run fully offline once vendored, for personal (single-maintainer) use.

Out of scope: publishing this as a distributable public package; full platform support beyond `darwin/arm64` for the vendored binary artifacts.
