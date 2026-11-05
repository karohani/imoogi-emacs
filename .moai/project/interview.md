# Project Interview

## Stage A Round 1: Ownership, Purpose, and Goal
Question: Where is imoogi-emacs right now — actively evolving, stable/maintenance, or mid-overhaul?
Answer: Actively evolving — packages/modules keep being added and swapped over time (this hydra→transient work is an example).
Domain: developer-tooling / editor-configuration (auto-populated from codebase analysis: Emacs Lisp config + a companion Go CLI that manages LSP toolchain artifacts)
Goal: Provide a modern, self-contained Emacs configuration that boots and works fully in a network-isolated (air-gapped) environment — modernizing package choices over time while never breaking offline bootability.

## Stage A Round 2: Constraints and Non-Goals
Question: What hard constraints should the docs record?
Answer: Air-gap only.
Constraints: The boot path (early-init.el → boot.el → modules/) must never touch the network — no package-refresh-contents, url-retrieve, or runtime installs during Emacs startup. All Emacs packages are vendored into vendor/elpa/ (committed to git); the Go toolchain manager (imoogi-toolchain) follows the same fetch (online) / setup (offline, no network) split. No other performance or security constraints are recorded.

## Stage A Round 3: Scope, Boundaries, and Documentation Priority
Question: Which aspect should structure.md/tech.md document most carefully?
Answer: Module boundaries and how to add a new module — the numbered-module convention (modules/NN-name.el), the imoogi-require contract, and boot.el's load order/failure-isolation behavior.

Question: What's explicitly in scope vs. out of scope for this project?
Answer: In: Emacs config + Go toolchain CLI, both air-gapped for personal use.
Scope: In-scope — the Emacs configuration itself (modules/, vendor/elpa/) and the imoogi-toolchain Go CLI (cmd/, internal/) that manages LSP toolchain artifacts for it, both designed to run fully offline once vendored. Out-of-scope — publishing this as a distributable/public package, and platform targets beyond the current darwin/arm64 build.

## Stage B Round 4: Verification, Surfaces, and Sharing
Verification: Emacs side — tests/run.sh (3 phases: check-parens syntax validation, an offline-boot smoke test with package-archives set to nil, then the ERT suite via tests/run.el). Go side — standard `go test ./...` per package. (auto-populated from codebase analysis)
UI surface: has-ui — Emacs itself is the UI, and this config invests heavily in UI customization (doom-themes/doom-modeline/nerd-icons, Treemacs IntelliJ-style tool windows, hydra/transient pop-up menus). (auto-populated from codebase analysis)
External systems: none at runtime. The only network contact is fetch-time-only, on an online machine, during `imoogi-toolchain fetch` (npm registry, Go module proxy) and `scripts/vendor.el` (MELPA) — never during Emacs boot or offline `setup`. (auto-populated from codebase analysis)
Team sharing: solo — "개인 Emacs 설정" (a personal Emacs configuration); single maintainer (jay) per .moai/config/sections/user.yaml. (auto-populated from codebase analysis)
