---
id: workflow-imoogi-vendor-update
name: update-vendor
description: Update imoogi-emacs vendored ELPA packages while preserving the air-gap policy
source: manual
triggers:
  - imoogi-vendor
  - imoogi vendor
  - imoogi vendor update
  - imoogi vendor upgrade
  - imoogi 패키지 업데이트
  - imoogi 벤더 업데이트
  - imoogi packages.el
  - imoogi packages.lock
quality: high
argument-hint: "[missing|upgrade|package-name]"
---

# Update Vendor Skill

## Purpose

Update this repo's `vendor/elpa/` package store from `packages.el` without weakening the air-gap guarantee. Use this when packages are added, removed, upgraded, or `packages.lock` needs to be regenerated.

## When to Activate

Activate for project-specific requests like:

- "imoogi-vendor"
- "imoogi vendor update"
- "imoogi vendor upgrade"
- "imoogi 패키지 업데이트"
- "imoogi 벤더 업데이트"

## Required Context

- Repo root: `/Users/jay/workspace/imoogi-emacs`
- Manifest: `packages.el`, variable `imoogi-required-packages`
- Vendoring script: `scripts/vendor.el`
- Vendor store: `vendor/elpa/`
- Lock/audit file: `packages.lock`
- Preferred Emacs binary on this machine: `/Applications/Emacs.app/Contents/MacOS/Emacs`

## Workflow

1. Inspect `AGENTS.md`, `packages.el`, `scripts/vendor.el`, and relevant module files before editing.
2. If adding a package, add only the top-level package to `imoogi-required-packages`; let package.el resolve transitive dependencies.
3. Never add runtime package installation to `early-init.el`, `boot.el`, or `modules/`. Boot must remain network-free.
4. On an online machine, run one of:

```bash
/Applications/Emacs.app/Contents/MacOS/Emacs --batch -Q -l scripts/vendor.el
/Applications/Emacs.app/Contents/MacOS/Emacs --batch -Q -l scripts/vendor.el -- upgrade
```

5. Verify `vendor/elpa/` contains the added or updated package directories and `.signed` files when available.
6. Verify `packages.lock` reflects the package count and versions.
7. Run the offline boot check:

```bash
/Applications/Emacs.app/Contents/MacOS/Emacs --batch -Q --eval '(progn (setq load-prefer-newer t user-emacs-directory "/tmp/imoogi-emacs-test/") (load-file "boot.el") (kill-emacs 0))'
```

8. For package changes tied to a module, run module-specific recognition or behavior tests. For language modes, create temporary buffers and assert `major-mode`.
9. Run final hygiene checks:

```bash
git diff --check
rg -in "akia|begin (rsa|openssh|private)|secret|token|password|api[_-]?key" packages.el packages.lock vendor/elpa
```

10. Commit atomically. Keep vendored artifacts, `packages.el`, and `packages.lock` together. Put module wiring and docs in separate commits when possible.

## Gotchas

- Do not run vendoring inside the closed/air-gapped target network.
- Do not call `package-refresh-contents` anywhere in runtime boot code.
- Do not vendor native `.eln` artifacts; they are machine-specific caches.
- Match build-machine and target Emacs major versions because `.elc` compatibility matters.
- `scripts/vendor.el` also refreshes `assets/fonts/NFM.ttf`; check whether that file changed unexpectedly.
- If batch Emacs prompts during startup hooks, wrap the check in `(kill-emacs 0)` after loading.

## Examples

### Add one missing package

1. Edit `packages.el`.
2. Run `Emacs --batch -Q -l scripts/vendor.el`.
3. Verify boot and targeted behavior.
4. Commit manifest, lockfile, and vendor additions together.

### Full upgrade

1. Run `Emacs --batch -Q -l scripts/vendor.el -- upgrade`.
2. Review every package directory and `packages.lock` change carefully.
3. Run offline boot and module-specific checks.
4. Split commits by concern if config/docs changed too.
