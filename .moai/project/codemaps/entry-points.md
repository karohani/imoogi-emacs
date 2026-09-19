# Codemap — Entry Points

## Emacs

| File | Role |
|---|---|
| `early-init.el` | First file Emacs loads. Startup performance tuning, before any package machinery exists. |
| `boot.el` | The real entry point. Sets `package-user-dir` to `vendor/elpa/`, defines `imoogi-require` and `imoogi-failed-modules`, loads the 30 modules in dependency order, and defines `imoogi-reload`. |
| `packages.el` | The package manifest (`imoogi-required-packages`). Read by `scripts/vendor.el`, not by the boot path. |
| `modules/<package>/<name>.el` | One feature each. Entered only through `boot.el`'s list. |
| `modules/development/lang/<language>.el` | Entered by `17-lsp.el`'s auto-discovery, not by `boot.el`. |
| `modules/org/anki/imoogi.el` | Entered by `24-anki.el`; exposes `imoogi-sync` and `imoogi-anki-setup`. |
| `modules/org/flashcards/imoogi-flashcards-core.el` | Entered by `25-flashcards.el`. |

`imoogi-reload` re-runs `boot.el` in a live session and reports the modules it skipped. It overwrites definitions but cannot undo removals — a deleted key binding or a detached hook still needs a restart.

## Go Binaries

Each binary is `cmd/<name>/main.go`, built into `bin/<name>`.

| Entry point | Invocation | Contract |
|---|---|---|
| `cmd/imoogi-anki/main.go` | Spawned by `modules/org/anki/imoogi-process.el` | One JSON request on stdin → one JSON response on stdout, once per sync run. No user interaction. Log path from `IMOOGI_ANKI_LOG`. |
| `cmd/imoogi-clip/main.go` | Spawned by `modules/org/28-clipboard.el` | `imoogi-clip [--version]`. Reads up to 1 MiB of JSON from stdin, writes one JSON response. A decode failure is itself returned as a structured failure, not an exit code. |
| `cmd/imoogi-notes/main.go` | Spawned by `modules/project/26-project-notes.el` | `imoogi-notes [--version]`. Same 1 MiB stdin JSON → stdout JSON shape; `{ok, code, error}`. |
| `cmd/imoogi-org-preview/main.go` | Spawned by `modules/org/23-org-preview.el` | `--addr` / `--host` / `--port` / `--token` / `--print-bootstrap-json`. Prints one bootstrap JSON line, then serves loopback HTTP until interrupted. |
| `cmd/imoogi-provenance/main.go` | `make provenance-generate` / `provenance-verify` | `imoogi-provenance <verify\|generate\|record-git-source ID COMMIT REF>`. Runs against the current working directory; non-zero exit on any deviation. |
| `cmd/imoogi-toolchain/main.go` | `make toolchain-setup`, `scripts/setup-toolchain.sh` | `fetch` (online) / `setup` (offline), dispatched by `internal/cli` with the two hooks supplied by `main`. |

Binary resolution from Emacs follows the same shape in all three callers: the configured command name on `PATH` first, then `bin/<name>` inside the repository. The note-move caller additionally raises a `user-error` naming `make build-notes` when neither resolves. The command is a user option in each case — `imoogi-project-notes-command`, `imoogi-clipboard-command`, `imoogi-org-preview-command`.

## Make Targets

`make help` is the index. The entry points that matter:

| Target | Entry into |
|---|---|
| `install` | Links the repository to `~/.emacs.d` (`scripts/install.sh`) |
| `emacs-install`, `emacs-prewarm`, `emacs-where` | Editor installation and native-compile pre-warming |
| `toolchain-setup` | `scripts/setup-toolchain.sh` → `imoogi-toolchain` |
| `grammars` | `scripts/build-grammars.sh` → `vendor/tree-sitter/` (online only) |
| `build` | Compile check of every Go package |
| `build-all` | All six CLIs into `bin/` |
| `build-toolchain`, `build-anki-bin`, `build-org-preview`, `build-clipboard`, `build-notes`, `build-provenance` | One CLI each |
| `build-anki`, `install-clipboard` | Install a CLI into `ANKI_PREFIX` / `CLIPBOARD_PREFIX` |
| `provenance-generate`, `provenance-verify`, `verify-vendor` | `imoogi-provenance` |
| `fmt`, `fmt-check`, `lint` | `gofmt` and `go vet` |
| `test-elisp`, `test-go`, `test-shell`, `test` | The three suites and the aggregate |
| `ci-local` | `fmt-check` + `lint` + `test`; the pre-push hook entry point |
| `clean-elc`, `clean` | Byte-compiled module output; downloaded Emacs distribution cache |
| `tmux-install`, `tmux-check` | Optional tmux configuration |

## Test Entry Points

| Entry | Runs |
|---|---|
| `tests/run.sh` | Three phases: recursive `check-parens` over `early-init.el`, `boot.el`, `packages.el` and every `.el` under `modules/` and `tests/`; an offline boot with `package-archives` nil, bracketed by `tests/boot-health.el` capture/assert; then the ERT suite. |
| `tests/run.el` | Loads `boot.el` in a temporary `user-emacs-directory`, loads every `tests/*-test.el`, ends with `(ert-run-tests-batch-and-exit)`. |
| `go test ./...` | Every Go package. |
| `make test` | `verify-vendor` + `test-elisp` + `test-go` + `test-shell`. |
| `make ci-local` | The pre-push gate. |

`tests/benchmarks/` and `tests/docker/` hold the performance and cross-distribution portability harnesses, which are driven separately rather than by `make test`.

## Vendoring Entry Points (online machine only)

| Entry | Produces |
|---|---|
| `emacs --batch -Q -l scripts/vendor.el` | `vendor/elpa/`, `packages.lock` |
| `emacs --batch -Q -l scripts/vendor.el -- upgrade` | The same, refreshing all vendored packages |
| `make grammars` | `vendor/tree-sitter/*.dylib` |
| `imoogi-toolchain fetch` | `vendor/toolchains/`, `toolchains.lock.json` |
| `make provenance-generate` | `provenance/*.json`, `vendor-manifest.json` |
