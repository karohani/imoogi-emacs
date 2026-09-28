# imoogi-emacs Security Review — 2026-08-23

Baseline: HEAD `511777b` (main), uncommitted `.el` edits in tree. Scope: boot path, Go toolchain CLI (`cmd/`, `internal/`), scripts, git hooks, vendored artifacts, devcontainer (untracked, local-only).

## Verdict

No remotely exploitable vulnerability found. The project's real attack surface is the **supply chain / air-gap integrity** path, and that is where the findings are. One finding is a live, confirmed break of the documented install path.

## Findings (severity-ordered)

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| 1 | **High** (confirmed) | Committed air-gap CLI binary is stale — `setup` fails on target | `vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain`, `internal/fetch/fetch.go` `buildBootstrap` |
| 2 | Medium (local-only) | Devcontainer sandbox can disable its own firewall; egress porous; unpinned installs | `.devcontainer/Dockerfile`, `.devcontainer/init-firewall.sh` |
| 3 | Medium | Opaque committed binaries with no re-verifiable provenance | `vendor/tree-sitter/*.dylib`, `vendor/ghostel-module/`, 313 `.elc`, `packages.lock` |
| 4 | Low–Med | Online `fetch` hardening gaps (TOFU npm, same-origin node checksums, inherited Go env, unbounded HTTP) | `internal/fetch/fetch.go` |
| 5 | Low | `scripts/vendor.el` runs with `-Q` — early-init TLS hardening not in effect; MELPA unsigned | `scripts/vendor.el:6-8,34-46`, `early-init.el:96-99` |
| 6 | Low | `.local/bin` prepended to `PATH`/`exec-path` for all Emacs subprocesses, unverified | `modules/17-lsp.el:36-55` |

### F1 — Stale committed CLI (High, confirmed)

Evidence:
- `go build` at HEAD → `491492881ff4…`; committed binary → `a5ea1ffa79f3…` (mismatch).
- `go build` from `git archive 5808620` → `a5ea1ffa79f3…` (exact match ⇒ build is reproducible; binary was built one commit before `511777b` added basedpyright).
- `strings` basedpyright count: committed 0, HEAD build 8.
- Air-gap replica run: `imoogi-toolchain setup` → `no provider for component "basedpyright" kind "python-language-server"`, `exit=10`.
- Developer machine `.local/toolchains/2026.08.23.1/install.json` lists basedpyright ⇒ built from a fresh compile, not the committed binary, so the break is invisible locally.

Control gap: `buildBootstrap` skips rebuild when binary + provenance already exist; `validateBootstrapProvenance` binds binary↔provenance only — nothing binds either to the source tree. A stale (or tampered) binary passes `fetch`'s own check.

Fix: always rebuild to staging and compare to the committed hash (fail on mismatch), and/or a test/CI gate (`go build … && shasum`), refreshing binary + `provenance.json` together. Record the source commit in provenance.

### F2 — Devcontainer (Medium, `.devcontainer/` is gitignored — local only)

- `sudoers`: `node ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh`; script accepts `--off` ⇒ the permissionless agent (`alias cc="claude --dangerously-skip-permissions"`) can run `sudo init-firewall.sh --off` and remove its own containment.
- Egress: `tcp/22` to any host, `udp+tcp/53` to any host (tunnel/exfil channels), allowlist resolved to IPs once at start (rot).
- Unpinned: lazygit `curl | tar` from latest release (no checksum), `npm install -g …@latest`, `uv:latest`.

Fix: drop `--off` from the sudo-allowed script (separate root-only script), restrict 22/53 to needed hosts, pin versions + checksums.

### F3 — Opaque binaries without re-verifiable provenance (Medium)

- 12 tree-sitter `.dylib` built by `scripts/build-grammars.sh` from **git tags** (mutable), no commit SHA, no output hashes, compiler stderr suppressed.
- `vendor/ghostel-module/ghostel-module.dylib` (1.2 MB): only `ghostel-module.version` = `0.34.0`; no hash / release URL.
- 313 committed `.elc`: no mechanism to confirm they match their `.el`.
- `packages.lock`: names + versions only, archive column `?`, no hashes.

Design anchors integrity to the git commit (documented), but nothing supports *later* re-verification. Fix: record commit SHA + SHA-256 per grammar, hash + upstream URL for ghostel module, add a rebuild-and-compare check (grammars; `.elc`) on the online machine.

### F4 — Online fetch hardening (Low–Medium)

- npm tarballs (`typescript`, `typescript-language-server`, `basedpyright`): no independent hash on first fetch (`validate` nil or presence-only) → trust-on-first-use into the lock. (Registry `dist.integrity` is same-channel; marginal gain.)
- node: `SHASUMS256.txt` from the same origin; `SHASUMS256.txt.sig` (GPG) not verified.
- `buildGopls` / `buildBootstrap`: `go` inherits `os.Environ()` — `GOFLAGS`, `GONOSUMDB`, `GOINSECURE`, `GOPROXY` can silently bypass sumdb.
- `http.DefaultClient` (no timeout); `download` has no body-size cap.

### F5 — vendor.el TLS posture (Low)

Runs `emacs --batch -Q`, so `gnutls-verify-error t` from `early-init.el` is not loaded; `package-check-signature` left at default; MELPA has no signatures. Actual batch-mode NSM behavior **not executed/verified**.

### F6 — `.local/bin` PATH prepend (Low)

`imoogi-lsp-prepend-local-bin` puts a git-ignored, user-writable directory first in `PATH` for every subprocess, validating only the symlink shape — not the bundle content. Same-user write access required; low.

## Positive controls (keep as-is)

- `internal/artifact/artifact.go`: staging + rename, symlink refusal/validation, `O_EXCL`, entry/size limits, path-clean checks.
- `internal/setup/setup.go`: `flock`, probes run with `PATH=/usr/bin:/bin HOME=<tmp>`, bundle-tree SHA-256, lock artifact paths validated (`config.go:325`).
- All 5 lock artifacts: size + SHA-256 **OK** against `toolchains.lock.json`.
- Boot path has no network calls; `ffap-machine-p-known 'reject`; `clojure-ts-ensure-grammars nil`; `ghostel-module-auto-install nil`.
- No secrets in tracked files (pattern scan).

## Gaps (not verified)

- Not read in full: `internal/cli/cli.go`, `internal/config/config.go`, the 87 vendored elpa packages, `tests/docker/`.
- Secrets scan covered `git ls-files` only — untracked `.moai/`, `.claude/`, `.omc/`, `.omx/` not scanned.
- Batch-mode TLS/NSM behavior of `vendor.el` not executed.
- No dynamic testing of Emacs modules beyond static review.

## Action items

1. Rebuild + recommit CLI binary and provenance; add rebuild-and-compare gate to `fetch` (F1).
2. Remove `--off` from the sudo-allowed firewall script; tighten 22/53; pin devcontainer installs (F2).
3. Record commit SHA + SHA-256 for grammars and ghostel module; add `.elc` rebuild check (F3).
4. Scrub Go env in `fetch` (`GOFLAGS=`, `GONOSUMDB=`, `GOINSECURE=`), add HTTP timeout + size cap, optionally verify node GPG signature (F4).
5. Set `gnutls-verify-error t` and `network-security-level 'high` inside `vendor.el` (F5).
