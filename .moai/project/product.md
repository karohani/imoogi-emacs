# Product

## Project Name

imoogi-emacs

## Description

imoogi-emacs is a personal Emacs configuration (개인 Emacs 설정) built to boot and operate fully inside a network-isolated (air-gapped / 망분리) environment. It pairs a modular Emacs Lisp configuration with six companion Go CLIs that handle the work Emacs is poor at — rendering and network protocol for Anki sync, clipboard access, verified file moves, HTML preview, provenance hashing, and language-server toolchain management. Both halves share the same design principle: everything the editor needs at boot or at runtime is vendored ahead of time, so cloning the repository (or carrying it into an isolated network) is sufficient to get a fully working, modern editing environment with zero network access required.

The project is actively evolving — packages and modules are added, swapped and modernized over time — but the air-gap boot guarantee is the one constraint that must never regress.

## Target Audience

- Single maintainer: jay (solo project)
- Personal use only — not published or distributed as a public/shared package
- Anyone who needs to work inside a network-isolated development environment (the primary motivating use case) while still having a modern, IDE-like Emacs experience

## Feature Areas

### Editing basics and Korean input

Modern defaults with session persistence (recentf, savehist, saveplace), a vertico/orderless/marginalia/embark/consult/corfu/cape completion stack, undo-fu with session persistence, yasnippet, code folding, avy/helpful navigation, and doom-themes/doom-modeline/nerd-icons for visual identity. Korean input is a first-class concern: `S-SPC` hangul toggle works throughout, including inside the vendored ghostel terminal. Pop-up command menus are built on `transient`.

### Project workspaces

Built-in `project.el` combined with `perspective` for workspaces, `tab-bar` as the window layer, Magit with diff-hl for Git, and Treemacs for an IntelliJ-style tool-window layout.

### Org writing and browser preview

Org and Markdown authoring conventions, Obsidian-style note integration, an Org agenda overview, and a live HTML preview: `imoogi-org-preview` serves the current Org or Markdown buffer over loopback HTTP with a session token, and Emacs opens and drives it in a browser.

### Project notes and study notes

Project-scoped Org notes that live beside — or entirely apart from — the source project. A note folder identifies itself with its own `.imoogi-project.json`, so a removable device carrying notes can be plugged into a different machine and rediscovered from the folder metadata alone; the host-specific source-path link stays in the host's own registry. New study notes can be created directly under a registered removable root, and an existing local note can be moved to one through the `imoogi-notes` helper, which copies, verifies every file with SHA-256 and switches the original before Emacs updates registrations, agenda paths and open buffers. Removable roots can be refreshed, logically detached, or physically unmounted where the platform supports it.

### Anki sync and local flashcards

One-way Org → Anki flashcard sync. The only thing that travels back into Org is the `ANKI_NOTE_ID` that imoogi itself writes. The Emacs layer scans sync roots, resolves per-heading properties nearest-wins, and hands a JSON request to the `imoogi-anki` back end, which renders fields, owns the two imoogi note types, plans the note writes, handles media, and talks to AnkiConnect. Sync runs append to a persistent diagnostic log so a failed run can be inspected afterwards. Where Anki is unavailable — the closed-network case — `org/25-flashcards.el` provides an Emacs-native SQLite-backed flashcard fallback with its own scheduler and review UI, sharing no state with the Anki path.

### Clipboard assets

Pasting a screenshot or a copied file into an Org or Markdown buffer goes through `imoogi-clip`, which inspects the clipboard, identifies its kind, and copies the asset into the note folder under a staging lease, with clipboard-identity checks so the asset written is the asset that was seen.

### LSP-backed languages

Eglot/Flymake/xref integration with nine per-language configurations (Bash, Clojure, Go, Java, JavaScript, Kotlin, Python, Rust, TypeScript/TSX), plus major modes for roughly twenty more file types. Language-server runtimes (Node, TypeScript, typescript-language-server, gopls) are vendored and activated by `imoogi-toolchain` rather than fetched at Emacs boot. Formatting is owned by one shared module loaded early in the boot sequence.

### Terminal

`ghostel`, a native terminal module built on libghostty-vt, vendored as a prebuilt platform module — with Korean input support carried into it.

### AI assistant

`gptel` configured against a LiteLLM gateway, including support for a globally configured private CA so the gateway can be reached inside a managed network.

## Use Cases

1. **Working inside a network-isolated (air-gapped) development environment** — the primary use case. The maintainer carries the repository, with `vendor/` already populated, onto an offline machine and gets a complete modern Emacs setup with no further network-dependent steps.
2. **Adding or modernizing a package or feature module** — on an online machine, edit `packages.el`, run `scripts/vendor.el`, refresh the provenance manifests, commit, and carry the update into the air-gapped environment.
3. **Adding LSP support for a new language** — drop a new file into `modules/development/lang/` (auto-discovered), and, if a new runtime is needed, extend `toolchains.json` and re-run `imoogi-toolchain fetch` online before `setup` offline.
4. **General-purpose software development across many languages** — with LSP-backed completion, diagnostics and navigation, Git through Magit, and workspace management through Treemacs and perspective.
5. **Studying from Org notes** — writing notes per project or per course, previewing them in a browser, attaching screenshots from the clipboard, and turning headings into flashcards through Anki or the local fallback.
6. **Keeping notes on a removable device** — creating or moving project and study notes onto an SD card or SSD, and picking them up again on another machine.

## Key Constraints

- **Offline boot is absolute.** No network access occurs during `early-init.el` → `boot.el` → module loading, nor during offline `imoogi-toolchain setup`. Every dependency is vendored, and vendored artifacts are hash-verified against a provenance manifest.
- **Graceful degradation over hard failure.** A module whose prerequisites are missing is skipped and recorded, not fatal; an unsupported clipboard platform returns a stable `unsupported-capability` result rather than pretending to work.
- **User data never lives in the repository.** Notes, registries, caches, study IDs and logs stay in the user's own directories, and structural refactors of the repository do not move or convert them.
- **Single developer.** Conventions favour a reviewable, self-explanatory layout over tooling that assumes a team.

## Explicit Non-Goals

- Not intended for distribution as a public/shared Emacs distribution or package.
- Not claiming full platform support beyond the current `darwin/arm64` build target for vendored binary artifacts.
- No performance or security constraints beyond the air-gap requirement and the provenance guarantee that follows from it.
