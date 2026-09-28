# Research: Coding Agent -> Emacs IPC Bridge MVP (`imoogi-agent`)

This document combines four read-only research lenses: codebase-precedent, external-docs, constraints-risks and prior-SPEC-memory. Each claim names the lens it came from. Line numbers are the ones the lenses reported; small line-range differences between lenses are noted in contradictions item 8.

## Summary

- `imoogi-agent` would be the seventh Go CLI.
  - It would be the first one that calls **into** Emacs. All existing CLIs are started by Emacs as subprocesses (`codemaps/overview.md:14`).
  - It can follow the existing `cmd/` + `internal/` + `bin/` layout. No change is needed to `go.work` or provenance.
  - Several docs that say "six CLIs" would become stale.
- The Emacs server uses defaults only: a local Unix socket, no TCP, no custom name.
- Emacs 30 has `server-eval-args-left`. It lets the CLI pass the event-file path as a raw argument, so the path never has to be escaped into an elisp string.
- `emacsclient` exits 1 both when it cannot connect and when the evaluated elisp raises an error. So the planned exit-code split (1 = connection failure, 3 = protocol error) cannot come from `emacsclient`'s exit status alone.
- Several things have no precedent and must be designed from scratch: Go code that calls emacsclient, elisp escaping for `--eval`, a notification UI in Elisp, and an event-file lifecycle.
- No earlier SPEC or memory note covers this area.

---

## 1. Go CLI conventions

### Layout
- There are six CLIs under `cmd/`: anki, clip, notes, org-preview, provenance, toolchain.
- Each has a thin `main()` that calls an `internal/` package, and builds to `bin/<name>`. `bin/` is ignored by git (`.gitignore:29`, `/bin/`). (codebase-precedent, constraints-risks)

### Argument parsing
None of the CLIs use the `flag` package; arguments are parsed by hand. (codebase-precedent)
- `imoogi-clip` and `imoogi-notes` accept only `--version`. Any other argument exits 2 (`cmd/imoogi-notes/main.go:21`).
- `imoogi-anki` has a testable `func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int` (`cmd/imoogi-anki/main.go:38-76`).
- `imoogi-toolchain` delegates to `internal/cli`.

### Exit codes
- `internal/cli/cli.go:16-24` defines `ExitOK=0, ExitError=1, ExitUsage=2`, plus codes 10–13.
- `imoogi-provenance` uses 1 for general failure (constraints-risks).
- **Exit 2 = usage error** everywhere, which matches the SPEC's "2 = bad request" (codebase-precedent).
- **Exit 1 means a generic error** in current code. The SPEC narrows it to "Emacs connection failure", which leaves local failures (for example, the event file cannot be written) without a clear code (constraints-risks).
- **No existing CLI uses exit 3** (codebase-precedent).
- `imoogi-clip` and `imoogi-notes` report decode and protocol failures as JSON with **exit 0**, not as non-zero exits (codebase-precedent; `codemaps/entry-points.md:24-25` per prior-SPEC-memory).

### Other conventions (codebase-precedent)
- `var version = "dev"` is overridden with `-ldflags -X main.version`. Only `imoogi-clip`'s Makefile target does this.
- stdin is capped with `io.LimitReader(os.Stdin, 1<<20)` (`cmd/imoogi-notes/main.go:24`). This is the only size-limit precedent.
- Output uses `json.NewEncoder` with `SetEscapeHTML(false)`.

### Makefile
- `build-all: build-toolchain build-anki-bin build-org-preview build-clipboard build-notes build-provenance` (`Makefile:109`).
- Each target runs `mkdir -p bin && go build -o bin/<name> ./cmd/<name>`.
- A new `build-agent` target must be added to `build-all`, `.PHONY` (`Makefile:35`) and the help text.
- `ci-local: fmt-check lint test`, where lint is `go vet` and test is `go test ./...` (`Makefile:167-189`).

### go.work
- It contains only `use .` (with `go 1.26`). It exists so Go does not treat the Emacs `vendor/` directory as Go vendoring.
- A new CLI in the same module needs **no go.work change** (codebase-precedent, constraints-risks, external-docs).

### Provenance
- Adding a binary under `bin/` **changes no manifest** (all lenses agree). The two lenses describe the relevant field differently; see contradictions item 5.
- constraints-risks adds one exception: a binary shipped through `vendor/toolchains`, the way `imoogi-toolchain` is, would fall inside provenance.

### Third-party dependencies
- `go.mod` has one third-party dependency, `go-org` (tech.md; constraints-risks).
- Useful standard-library pieces (external-docs, from `go doc` on go1.26.4):
  - `os.CreateTemp` creates files with mode 0o600 before umask and random names. The caller must delete the file.
  - `os/exec` never runs a shell. Since Go 1.19, `LookPath` refuses programs found through a current-directory PATH entry.
  - `exec.CommandContext` gives a timeout.
  - `os.Root` / `os.OpenRoot` keep file access inside one directory and reject symlinks that escape it. They do **not** block device files, bind mounts or crossing filesystem boundaries.

### Docs that say "six CLIs"
These must change together:
- `product.md:9`
- `structure.md:5,86,170` (and `:144,178` per prior-SPEC-memory)
- `tech.md:8,55,66,112,155`
- `codemaps/overview.md:8`
- `codemaps/entry-points.md:43`

The overview's "Every Go CLI is invoked by Emacs as a subprocess" (`overview.md:14`) would also stop being true (prior-SPEC-memory).

## 2. Finding and invoking emacsclient

### Existing discovery logic
It exists **only in bash**, in `scripts/imoogi-editor` `find_emacsclient`. It checks, in order:
1. `$EMACSCLIENT`. It must be executable, otherwise the script fails.
2. `command -v emacsclient`.
3. `/Applications/Emacs-${EMACS_VERSION:-31.1}.app/Contents/MacOS/bin/emacsclient`.
4. `/Applications/Emacs.app/...`.
5. The glob `/Applications/Emacs-*.app/...`.

It then runs `exec "${emacsclient}" --create-frame "$@"`, deliberately without `--no-wait` (codebase-precedent, constraints-risks). `install.sh:18-19,88` links only `~/.local/bin/imoogi-editor` and sets VISUAL/EDITOR.

### Precedents and risks
- **No Go code finds or runs emacsclient** (codebase-precedent).
- Rewriting the discovery in Go risks the two copies drifting apart (constraints-risks).
- `tests/setup-toolchain-test.sh:118-130` tests `imoogi-editor` with a **fake emacsclient**. That pattern can be reused to test the new CLI (constraints-risks).

### Machine state (external-docs, local probe)
- `which emacsclient` reported "not found", so it is not on PATH in this shell.
- `/Applications/Emacs.app/Contents/MacOS/bin/emacsclient` exists, along with `bin-arm64-11/`, `bin-x86_64-10_14/` and `bin-x86_64-10_12/` variants.
- Both `Emacs.app` and `Emacs-31.1.app` are installed, so there is more than one candidate.
- `emacsclient --version` reported 30.2.

### emacsclient options relevant here (external-docs)
- `-w/--timeout=N`: the default 0 **waits forever**. This matters for best-effort IPC.
- `-s/--socket-name` or `EMACS_SOCKET_NAME`; `-f` or `EMACS_SERVER_FILE` for TCP.
- `-u/--suppress-output` and `-q/--quiet`.
- `-a/--alternate-editor`: an empty string starts a daemon, which this CLI should avoid.
- `-T` / `EMACSCLIENT_TRAMP` prefixes file arguments for TRAMP. It does not affect `--eval`, but an inherited value in the environment is worth knowing about.

### Exit status (external-docs from `emacsclient.c` on emacs-30, medium confidence; constraints-risks from general knowledge, medium)
- When the server returns `-error`, emacsclient prints `*ERROR*: ...` and exits with `EXIT_FAILURE` (1).
- When it cannot connect, it also exits 1. This was verified locally: `emacsclient -s /nonexistent/sock --eval '(+ 1 1)'` printed "can't find socket; have you started the server?" and exited 1.
- To tell exit 1 from exit 3, the CLI would have to either classify stderr (`*ERROR*:` versus `can't find socket` / `error accessing socket`) or read a status value returned by `imoogi-agent-receive-file`.

### Passing the path to `--eval`
- **Option A (external-docs, from Emacs 30.1 NEWS:74-81 and the bundled manual):** use `server-eval-args-left`. Run `emacsclient --eval '(imoogi-agent-receive-file (pop server-eval-args-left))' <path>`. The expression **must** `pop` the argument even if it fails, otherwise Emacs evaluates the path as Lisp. This removes the escaping problem entirely.
- **Option B (constraints-risks):** keep the literal `"<path>"` form, but restrict the event-file path to a name the CLI generates from a known-safe character set.
  - Go's `strconv.Quote` / `%q` is not elisp read syntax. `\x` escapes are greedy in elisp, and the `\u` / `\U` forms differ.
  - Only `\` and `"` need escaping in an elisp string literal.
  - Shell quoting is not an issue, because `exec` is used without a shell.
- Not determined (external-docs): whether `server-eval-args-left` behaves differently under a daemon with a custom `server-name`, and the maximum argument or message length emacsclient accepts. The second matters for `imoogi-agent-receive-json`.

## 3. Emacs server setup

- `modules/general/13-system.el` ~112-122 contains `use-package server :ensure nil :if (not (daemonp)) :hook (after-init . imoogi--server-start)`, which calls `(unless (server-running-p) (server-start))`.
- Nothing in modules, boot.el, init.el or early-init.el sets `server-name`, `server-socket-dir`, `server-use-tcp` or `server-auth-dir`. So the server uses the default local socket, with no TCP and no auth key (codebase-precedent, constraints-risks). codebase-precedent did not check user-level custom files outside the repo.
- Probed values from a batch Emacs 30.2 session (external-docs):
  - `server-socket-dir = "/var/folders/pf/.../T/emacs501"`
  - `server-use-tcp = nil`
  - `server-auth-dir = "~/.emacs.d/server/"`
  - `server-name = "server"`
- Socket security (external-docs):
  - emacsclient refuses a socket directory the user does not own, or one that is group- or world-writable (`st.st_uid != uid || (st.st_mode & (S_IWGRP|S_IWOTH))`).
  - `server-auth-dir` and the auth key apply only to TCP.
  - The actual permissions of the socket directory on this machine were not checked (constraints-risks gap).
- `exec-path-from-shell` copies `TMPDIR` into GUI Emacs (`13-system.el:98-110`). If the agent's `TMPDIR` differs from Emacs's, emacsclient will not find the socket and the CLI reports a connection failure, exit 1 (constraints-risks).
- If the server is running (or restarted) under `daemonp`, the hook is skipped. Not investigated further.

## 4. Module registration and boot health

- Modules load from a hard-coded `dolist` in `boot.el` (65-97 per codebase-precedent, 63-103 per constraints-risks). Each is wrapped in `condition-case`.
  - A module that fails is added to `imoogi-failed-modules` and triggers `display-warning :error`.
  - `imoogi-require` (`boot.el:44-52`) raises an error when a package is missing.
- Module template: start with `(imoogi-require "NN-name" 'pkg ...)`, end with `(provide 'imoogi-NAME)`, and add `"<package>/NN-name"` to the `dolist` (AGENTS.md §2, SPEC-TRANSIENT-001 research.md:171-174; prior-SPEC-memory).
- The highest number used is 29 (`org/29-org-roam`), so the **next free number is 30**, for example `development/30-agent` (codebase-precedent, constraints-risks).
- **Test coupling:** `tests/module-layout-test.el` `imoogi-module-reload-preserves-numbered-load-order` (~48-67) hard-codes the full `expected` order of `NN-` modules and asserts `(should-not imoogi-failed-modules)`. A new module that is not added there breaks `make test-elisp`, and with it `ci-local` and the pre-push hook (constraints-risks).
- `tests/boot-health.el` fails on any `display-warning` at `:error` or `:emergency` during boot. Its `imoogi-boot-health-required-features` lists only `imoogi-projects` and `imoogi-treemacs`, so it probably does not need extending (constraints-risks, not confirmed).
- The new module may require only built-ins or packages already vendored (constraints-risks; tech.md:10-14; AGENTS.md §0).

## 5. Project notes, artifacts and resolving `project`

- `imoogi-project-notes--ensure-directories` (`26-project-notes.el:1428-1433`) creates `assets/`, `references/` and `artifacts/`, but **not** `artifacts/agent/`.
- The notes root is `imoogi-project-notes-directory` (default `~/project-notes/`).
- `imoogi-project-notes-create-artifact` (3222-3270) uses `(expand-file-name "artifacts/" notes-dir)` and `make-directory ... t`. It **cannot be reused** for agent artifacts (constraints-risks):
  - It requires an Org TODO heading and raises `user-error` otherwise.
  - It prompts interactively.
  - It names files `%Y%m%d-<prefix>-<slug>.org` with a `-N` suffix on collision (`--unique-artifact-file`, 3193-3203).

  This naming differs from the SPEC's `YYMMDD-HHMMSS-{type}.{ext}` (codebase-precedent).
- `--write-new-file` (1327-1336) uses `'excl` and refuses to overwrite. With timestamp naming, two events of the same type in the same second would collide (constraints-risks).
- Resolving a project without a buffer:
  - `(imoogi-project-notes--find-entry-by-key (imoogi-project-notes--identity-key ROOT))` (403-411, 1103-1126).
  - `--identity-key` raises `user-error` for `file-remote-p` roots and keys git worktrees by the git common dir.
  - Mounted entries carry an `instance-id`, so lookup should go through `--all-entries` (938).
  - Inactive mounted entries must be blocked with `--ensure-entry-mutable` (1624-1640), which raises `user-error`.
- `--current-entry` (1128-1166) depends on buffer-local state. Under `emacsclient --eval` the current buffer is arbitrary, so it **must not** be used (constraints-risks).
- Not determined: whether a `project` given as a plain name (not a root path) can be resolved. Only lookups by key, root and notes directory were found (constraints-risks gap).
- `--find-file` (1682-1690) is a plain `find-file` with no containment check (constraints-risks).

## 6. Calling Go CLIs from Emacs, and UI patterns

- Finding binaries (`28-clipboard.el:75-83`; `23-org-preview.el:233`; `26-project-notes.el:2723`):
  - The code tries `executable-find` first, then falls back to `(expand-file-name (concat "bin/" program) imoogi-emacs-dir)`.
  - The note-move caller raises a `user-error` naming `make build-notes` when neither is found (prior-SPEC-memory).
- For an agent calling `imoogi-agent` from a shell, the CLI needs to be on PATH through a new install link, or called by absolute path. `install.sh` links only `imoogi-editor` (constraints-risks).
- The clipboard request envelope carries `protocol_version`, `operation` and `correlation_id` (`28-clipboard.el:85-100`). It already refuses remote buffers (69-73).
- Diagnostics buffers: `*imoogi-clipboard*` has a timestamped `--log` and is shown with `pop-to-buffer` (102-113). Other `*imoogi ...*` buffers exist: `*imoogi flashcards*`, `*imoogi Anki Targets*`, `*imoogi 메뉴 도움말*` (codebase-precedent).
- **The codebase uses neither `notifications-notify` nor `alert`** (codebase-precedent).
- `buffer-terminator` (`13-system.el:125-132`) kills buffers that are inactive, not visible and not modified after 30 minutes. A hidden agent-notification buffer may be killed (constraints-risks).
- Handlers that run under `--eval` must never prompt (`read-string`, `y-or-n-p`). A prompt blocks the server call and, because the default `-w` is 0, the CLI waits with it (constraints-risks + external-docs).

## 7. JSON handling

### In the Elisp codebase
- Native `json-parse-string` / `json-parse-buffer` is used in `13-system.el:74`, `28-clipboard.el:97` and anki `imoogi-process.el:146` / `imoogi-config` / `imoogi-targets` / `imoogi-setup`. `23-org-preview.el:524` is also native, but its classification is disputed; see contradictions item 2.
- The legacy `json-read` / `json-read-file` / `json-encode` is used in `26-project-notes.el:308,540,637,709,2751` and `27-gptel.el:254,562`.
- `28-clipboard` uses `:object-type 'alist :array-type 'list :null-object nil :false-object nil` (codebase-precedent). Whether other modules pass `:null-object` / `:false-object` is disputed; see contradictions item 1.

### Emacs 30.2 native parser behaviour (external-docs, local probes, high confidence)
- Native JSON is always available; libjansson is no longer used.
- `:object-type 'alist` gives **symbol keys**.
- Duplicate keys are **kept**, e.g. `((a . 1) (a . 2))`, and `assq` / `alist-get` return the **first** one.
- The defaults are `:null` and `:false`. Both are non-nil, so they count as true in elisp: a payload `"artifact": null` would look present unless `:null-object nil` is passed or the value is compared explicitly (constraints-risks + external-docs).
- Error types:
  - `json-trailing-content`, `json-end-of-file` and `json-parse-error` all inherit from `json-parse-error`.
  - **`json-object-too-deep` inherits only from `json-error`.**
  - `json-utf8-decode-error` exists for bad UTF-8.
  - Catch `json-error` to cover all of them.
- The nesting limit is 10000.
- `json-parse-buffer` reads only the first value. It does not reject trailing content unless the caller checks point afterwards.

### Differences between Go and Emacs (external-docs)
- Go `encoding/json`: a later duplicate key replaces an earlier one, and struct fields match case-insensitively.
- Emacs alists: the first duplicate wins, and matching is case-sensitive.
- So `{"type":"message","type":"open-file"}` would be read differently if the CLI validates it and Emacs parses it again.
- `DisallowUnknownFields` exists for strict decoding.

### Size limits
- Go side: a 1 MiB stdin precedent.
- Emacs side: **no size cap anywhere** (constraints-risks).

## 8. Security

- **Paths in payloads are untrusted.** `open-file`, `goto-location` and `artifact-created` accept paths from JSON.
  - **Existing building blocks:**
    - `file-remote-p` rejects TRAMP paths (`26-project-notes.el:407,683,694,763`).
    - `file-truename` resolves symlinks (383,768,802).
    - `file-in-directory-p` is used in `--find-entry-by-notes-file`.
  - **Missing:** no allowlist or containment check for opening arbitrary absolute paths (constraints-risks).
  - External-docs notes:
    - `file-remote-p` never opens a new connection.
    - `file-truename` resolves symlinks at every level.
    - `file-in-directory-p` returns nil if the directory does not exist.
- **Symlink trust has come up before.** The memory note on `internal/notemove` hardening lists open items to "reject escaping/absolute symlinks (they dangle after a move but report OK)" and "retarget Emacs buffers first in the post-success path". That is the nearest earlier record of path-trust concerns (prior-SPEC-memory).
- **`--eval` injection:** see §2. Both mitigations close it: `server-eval-args-left`, or a CLI-generated path with a restricted character set.
- **Event-file lifecycle:** there is no existing IPC event file. Precedents for permissions and cleanup are disputed; see contradictions item 3.
  - `os.CreateTemp` gives 0600 on the Go side (external-docs).
  - Deleting the file in Emacs after parsing conflicts with keeping it for debugging (constraints-risks).
- **Socket permissions:** see §3. The default local socket is protected by the ownership and permission check on its directory.
- **Blocking and best-effort:** the default `emacsclient -w 0` waits forever. Use a timeout via `-w` and/or `exec.CommandContext` (external-docs).

## 9. Test conventions

- `tests/run.sh` runs three phases (codebase-precedent, constraints-risks):
  1. check-parens over all `modules/**.el` and `tests/**.el`.
  2. An offline boot (`package-archives nil`) with boot-health.
  3. ERT through `tests/run.el`, which loads every `tests/*-test.el` after a full `boot.el` boot and calls `ert-run-tests-batch-and-exit`.
- ERT style: `lexical-binding`, `(require 'ert)`, `ert-deftest`, `make-temp-file`, and `cl-letf` stubs.
- Go tests:
  - Most live next to the code, e.g. `cmd/imoogi-anki/main_test.go:25,105,121`. They call `run(...)` with `bytes.Buffer` and `t.TempDir()`.
  - One Go test lives under `tests/` (`org-preview-contract_test.go`).
- The fake-emacsclient pattern in `tests/setup-toolchain-test.sh:118-130` can be reused.
- `tests/project-notes-test.el` fixtures could probably help build a notes-directory registry in tests. They were not read.

## 10. Prior SPECs, memory and project constraints

- **No earlier SPEC covers this area.** `.moai/specs/` contains only SPEC-ANKICARD-001..004 and SPEC-TRANSIENT-001. A grep across specs, project docs, codemaps, memory, AGENTS.md, CLAUDE.md, `.moai/reports` and `docs` found only `structure.md:132` (the `imoogi-editor` line in the directory tree) (prior-SPEC-memory).
- `.moai/reports/security-review-20260823.md` covers toolchain fetching and TOFU only (prior-SPEC-memory).
- The memory index has 4 notes. None covers emacsclient or agents. Parts that matter here:
  - The user wants replies in Korean.
  - The checkout contains unrelated uncommitted work from others: stage by explicit pathspec, never sweep-stage, and implementing agents run no git command that writes.
  - The workflow is SPEC → plan audit → kickoff approval → implement → gate re-run → pathspec commit.
  - Tier M work lands directly on main (Hybrid Trunk).
  - `main` has unpushed commits, and `.moai/` is mostly untracked.
- **Air-gap rule:** tech.md:10 calls air-gap safety "the single most important constraint". product.md:68 says "Offline boot is absolute". `interview.md:25` says "External systems: none at runtime." New Emacs packages must go packages.el → online `scripts/vendor.el` → packages.lock → provenance (AGENTS.md §0, §2).
- AGENTS.md §5 lists decisions not to reverse; none concern IPC. §6 says to commit hand-written config and bundled binaries separately, and only when the user asks.

## 11. Repository state risks

- There are uncommitted edits in the same files this SPEC would likely touch:
  - `modules/project/26-project-notes.el` (106 lines)
  - `tests/project-notes-test.el` (65 lines)
  - `CLAUDE.md`

  Whether these touch the entry or artifact helpers cited above was not determined (constraints-risks).
- Emacs version mismatch:
  - The task says Emacs 30.
  - tech.md says packages were resolved against Emacs 30.2.
  - `Makefile` and `scripts/install-emacs.sh:21` default to `EMACS_VERSION=31.1`.
  - The `imoogi-editor` lookup order tries `Emacs-31.1.app` **before** `Emacs.app` (30.2), and both are installed.

  Which Emacs actually runs the server is unknown. `server-eval-args-left` exists in both (Emacs 30.1+), but Emacs 31.1 behaviour was not checked (codebase-precedent, constraints-risks, external-docs).

---

### contradictions

1. **Do any modules pass `:null-object` / `:false-object`?**
   - codebase-precedent: `28-clipboard.el:85-100` decodes with `json-parse-buffer :object-type 'alist :array-type 'list :null-object nil :false-object nil`.
   - constraints-risks: "No module passes `:null-object` or `:false-object`, so the defaults `:null` and `:false` are both non-nil and truthy."
   - These cannot both be true. Unresolved; re-read `28-clipboard.el:97`.

2. **Is `23-org-preview.el` a native or legacy JSON user?**
   - codebase-precedent lists `23-org-preview:524` as native `json-parse-string` / `json-parse-buffer` (with other hits at 265 and 612).
   - constraints-risks' findings list 23-org-preview among the **legacy** `json-read` / `json-read-file` users, while its own evidence cites `23-org-preview.el:524` as `json-parse-buffer :object-type 'alist`.
   - The file may use both. Unresolved.

3. **Is there a precedent for restricted-permission temp files?**
   - codebase-precedent: yes. `13-system.el` `imoogi-system--write-config` (~24-40) creates a file with `make-temp-file` in the target directory, applies `set-file-modes #o600`, renames it, and deletes the temp file in `unwind-protect`. It calls this the precedent for event-file permissions and cleanup.
   - constraints-risks: "The only temp-file patterns are `mktemp` under `${TMPDIR:-/tmp}` in `install.sh` and `run.sh`. Permissions (0600) and deleting the event file ... have to be designed from scratch."
   - Both agree that no IPC event file exists today. They disagree on whether an Elisp 0600 / atomic-write precedent exists.

4. **What should exit 1 mean?**
   - The SPEC: "1 = Emacs connection failure."
   - constraints-risks: in-repo `internal/cli` `ExitError = 1` means a generic error, and `imoogi-provenance` uses 1 for general failure.
   - external-docs and constraints-risks both report that emacsclient itself returns 1 for both connection failures and elisp errors.
   - codebase-precedent calls the SPEC's codes compatible with conventions, citing only exit 2.
   - This is a conflict between the SPEC and the lens findings, not between lenses, and needs a design decision.

5. **Which provenance field was described?**
   - codebase-precedent quotes the `sources.json` **roots**: `assets/fonts`, `packages.el`, `packages.lock`, `provenance/sources.json`, `toolchains.json`, `toolchains.lock.json`, `vendor/elpa`, `vendor/ghostel-module`, `vendor/toolchains`, `vendor/tree-sitter`.
   - constraints-risks quotes `sources.json:3` `"boundaries": ["assets", "vendor"]`.
   - Both reach the same conclusion (a new `bin/` binary has no provenance impact). Only constraints-risks adds the `vendor/toolchains` exception.
   - Recorded so the reader can check which field actually governs.

6. **How strongly is the emacsclient exit-1 overlap supported?**
   - external-docs: medium confidence, based on emacs-30 `emacsclient.c` read through a summariser. The connection-failure case was verified locally; the `*ERROR*` case was not.
   - constraints-risks: medium confidence, from general Emacs knowledge only.
   - The two lenses agree on the claim. Only the connection-failure half has been observed.

7. **Emacs version.** The task says Emacs 30; tech.md says 30.2; the Makefile and `install-emacs.sh` default to 31.1. Both Emacs.app (30.2) and Emacs-31.1.app are installed, and `imoogi-editor` prefers 31.1. This is a conflict among repo sources and the task, surfaced by three lenses.

8. **Line-range differences (minor).**
   - boot.el module list: 65-97 (codebase-precedent) vs 63-103 (constraints-risks).
   - `imoogi-require`: 44-52 vs 44-51.
   - server block: ~112-122 vs 113-122.
   - Not substantive, but re-check before citing in the SPEC.

### NONE found (confirmed absences)

- No Go code that finds or runs emacsclient (codebase-precedent).
- No precedent in the repo for quoting strings into `emacsclient --eval` (codebase-precedent, constraints-risks).
- No `notifications-notify` or `alert` notification UI in Elisp (codebase-precedent).
- No IPC event-file writer (codebase-precedent, constraints-risks).
- No Emacs-side JSON size limit (constraints-risks).
- No allowlist or containment check for opening arbitrary payload paths (constraints-risks).
- No earlier SPEC, memory note, report or doc covering emacsclient, imoogi-editor, agent IPC or `imoogi-agent` (prior-SPEC-memory; confirmed by codebase-precedent and constraints-risks).
- No existing CLI that uses exit code 3 (codebase-precedent).
- No `server-name`, `server-socket-dir`, `server-use-tcp` or `server-auth-dir` customisation in the repo (codebase-precedent, constraints-risks).

### Open gaps (not investigated by any lens)

- The maximum argument or message length emacsclient accepts (matters for `imoogi-agent-receive-json`).
- How `server-eval-args-left` behaves under a daemon or custom server name, and under Emacs 31.1.
- The actual permissions of the socket directory, and user-level custom files outside the repo.
- Whether `internal/clipboard` has a reusable protocol-version or error-code type.
- The `README.md` EDITOR-flow text around lines 31-35.
- The git history and rationale for `scripts/imoogi-editor`.
- Whether the uncommitted `26-project-notes.el` changes touch the helpers cited in §5.
- Resolving `project` given as a plain name.
- Official Claude Code / Codex documentation on calling external CLIs.