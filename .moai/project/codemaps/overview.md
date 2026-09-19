# Codemap — Overview

## System Summary

imoogi-emacs is one repository holding two cooperating systems:

1. **The Emacs configuration** — 53 Emacs Lisp files under `modules/`, grouped into four packages (`general/`, `project/`, `org/`, `development/`) and loaded in an explicit dependency order by `boot.el`. This is the product.
2. **Six Go CLIs** — `cmd/imoogi-anki`, `cmd/imoogi-clip`, `cmd/imoogi-notes`, `cmd/imoogi-org-preview`, `cmd/imoogi-provenance`, `cmd/imoogi-toolchain`, backed by 27 packages under `internal/`. These exist to do what Emacs Lisp does poorly: HTML rendering, HTTP serving, platform clipboard access, cryptographic verification, archive handling and cross-platform binary management.

Everything both halves need at runtime is vendored under `vendor/` and hash-verified against `vendor-manifest.json`, so the whole system runs with no network access.

## The Elisp ↔ Go Boundary

**Every Go CLI is invoked by Emacs as a subprocess. Nothing is linked, loaded as a dynamic module, or shared in-process.** The only native module in the configuration is the vendored `ghostel` terminal, which is not part of the Go side at all.

The boundary has one dominant shape and one exception:

- **One-shot JSON request/response over stdin/stdout** — `imoogi-anki`, `imoogi-clip` and `imoogi-notes`. Emacs builds a request document, writes it to the process's stdin, reads one response document from stdout, and decodes it. Standard error is captured separately for diagnostics. The Go side performs no user interaction and asks no questions; it returns a result with an `ok` flag and a machine-readable error code, and the Emacs layer maps that code to a user-facing message and remediation.
- **Loopback HTTP with a session token** — `imoogi-org-preview`. Emacs starts the server with `--port 0 --print-bootstrap-json`, reads one bootstrap JSON line carrying the chosen port and token, and from then on talks HTTP to `127.0.0.1` while the browser renders the page.

`imoogi-provenance` and `imoogi-toolchain` are not invoked from Emacs at all: they are maintenance tools driven from `make`, and `imoogi-toolchain setup` produces the `.local/bin` symlink tree that `modules/development/17-lsp.el` later reads.

Consequences of the subprocess boundary worth keeping in mind:

- Each binary is independently buildable and testable (`make build-<name>`, `go test ./...`), and its protocol is a package (`internal/anki/protocol`, `internal/orgpreview/protocol`) rather than an implicit convention.
- A missing binary is a recoverable, reportable condition — each Emacs caller resolves the executable on `PATH` first and then at `bin/<name>` inside the repository. The note-move path raises a `user-error` naming the `make build-notes` target when neither is found.
- No Go failure can corrupt Emacs state: writes back into buffers are always performed by the Emacs side after a successful response.

## Layering

```
early-init.el
    └── boot.el ──────────────── package-user-dir → vendor/elpa/, imoogi-require
            └── modules/<package>/*.el   (explicit load order)
                    ├── development/17-lsp.el ──► modules/development/lang/*.el
                    ├── org/24-anki.el ────────► modules/org/anki/*.el
                    ├── org/25-flashcards.el ──► modules/org/flashcards/*.el
                    │
                    └── subprocess boundary
                            ├── bin/imoogi-anki       (JSON stdio)
                            ├── bin/imoogi-clip       (JSON stdio)
                            ├── bin/imoogi-notes      (JSON stdio)
                            └── bin/imoogi-org-preview (loopback HTTP)

make ──► bin/imoogi-toolchain  fetch(online) / setup(offline) ──► .local/bin
     ──► bin/imoogi-provenance generate / verify ──► vendor-manifest.json
```

## Where To Start Reading

| Question | File |
|---|---|
| What loads, and in what order? | `boot.el` |
| What does a module look like? | `modules/general/00-defaults.el`, then any other module |
| Why is the layout this way? | `ARCHITECTURE.md`, `docs/refactoring/module-packages.md` |
| What may an agent not do here? | `AGENTS.md` |
| How does a Go CLI get called? | `modules/org/anki/imoogi-process.el` |
| What is the JSON contract? | `internal/anki/protocol/protocol.go` |
| How is offline safety proven? | `tests/run.sh` phase 2, `tests/boot-health.el` |
| How is a vendored file trusted? | `provenance/sources.json`, `internal/provenance/verify.go` |
