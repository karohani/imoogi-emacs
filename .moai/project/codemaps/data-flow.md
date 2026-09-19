# Codemap — Data Flow

Six flows carry essentially all of the system's runtime behaviour.

## 1. Boot

```
emacs
  └─ early-init.el                      startup performance tuning
       └─ boot.el
            ├─ package-user-dir → vendor/elpa/ ; package-initialize
            ├─ use-package-always-ensure = nil          (never fetch)
            ├─ package-archives set, but never refreshed (no network)
            ├─ backup / auto-save / lock dirs → ~/.emacs.d/.cache/
            ├─ defun imoogi-require                     (locate-library gate)
            ├─ imoogi-failed-modules = nil              (reset on every load)
            └─ dolist over 30 module paths
                   └─ condition-case
                        ├─ ok    → module configures itself via use-package
                        └─ error → append name to imoogi-failed-modules
                                   + display-warning, continue the chain
```

The reset of `imoogi-failed-modules` at the top matters: `boot.el` is re-loadable through `imoogi-reload`, and without the reset a re-load would accumulate duplicate entries and mislead anything reading the variable as a verdict — which `tests/boot-health.el` and `tests/assert-boot.el` do.

No step in this flow performs network I/O. `tests/run.sh` phase 2 proves it by booting with `package-archives` set to `nil` and asserting an empty failure list.

## 2. Anki sync — request / response

```
user: imoogi-sync
  │
  ├─ modules/org/anki/imoogi-targets.el   resolve the registered sync roots
  ├─ modules/org/anki/imoogi-scan.el      walk the root, collect Org targets
  ├─ modules/org/anki/imoogi-target-scan.el  group targets per sync target
  ├─ modules/org/anki/imoogi-props.el     resolve properties nearest-wins
  ├─ modules/org/anki/imoogi-config.el    read the stored configuration
  │
  └─ modules/org/anki/imoogi-process.el
       ├─ setenv IMOOGI_ANKI_LOG ← imoogi-anki-log-file   (persistent diagnostics)
       ├─ spawn bin/imoogi-anki, stderr captured to a temp file
       ├─ write ONE JSON request document to stdin
       │      └─ internal/anki/protocol decodes it
       │           ├─ orgdoc   render title/body into Anki fields (incl. math)
       │           ├─ hashing  content hash → did this note change?
       │           ├─ registry load the derived per-sync-root state
       │           ├─ model    ensure imoogi's two note types exist
       │           ├─ planner  pure decision: create / update / delete / migrate
       │           ├─ media    media pass over rendered fields
       │           └─ ankiconnect  the only component allowed to do network I/O
       └─ read ONE JSON response document from stdout
            ├─ ok=false → imoogi-error.el maps the code to a message + remedy
            └─ ok=true  → imoogi-writeback.el writes ANKI_NOTE_ID back into
                          the Org buffers (buffer-mediated, never file-level)
```

The sync is one-way. The only value that travels Anki → Org is the note identifier imoogi itself wrote. The Go process performs no user interaction; every prompt and every message belongs to the Emacs layer.

## 3. Note move — request / response

```
user: C-c h p m v  (move a project/study note to a removable root)
  │
  ├─ 26-project-notes.el  choose the note and the target registered root
  ├─ resolve the helper: imoogi-project-notes-command
  │     └─ executable-find, else <repo>/bin/imoogi-notes,
  │        else user-error naming `make build-notes`
  │
  └─ spawn bin/imoogi-notes, one JSON request on stdin
       └─ internal/notemove.Move
            ├─ refuse if the destination name already exists (no overwrite)
            ├─ copy the folder under the same name
            ├─ SHA-256 verify every copied file
            └─ switch the original
       └─ one JSON response on stdout  {ok, code, error}
  │
  └─ on ok: Emacs rewrites the local registration, the agenda paths and the
            paths of any open buffers to the new location
```

Registration data stays host-side (`~/.emacs.d/.cache/project-notes-mounted-roots.json`); the note folder carries its own `.imoogi-project.json`, which is what lets the same device be rediscovered on another machine. A project whose TODO source lives outside the note folder (central `agenda.org`) is excluded from the move.

## 4. Clipboard ingest

```
user: paste an asset into an Org/Markdown buffer
  │
  ├─ 28-clipboard.el  inspect call (metadata only, ~0.12s budget)
  │     └─ bin/imoogi-clip → internal/clipboard adapter
  │          macOS: NSPasteboard changeCount + advertised types
  │          elsewhere: stable `unsupported-capability` result
  ├─ classify: text / file list / image
  │     text stays on the Emacs-native yank path — no helper involved
  ├─ checkpoint the clipboard identity, take a staging lease
  │     (renewed on an interval while the operation is open)
  └─ ingest call: copy the asset into the note folder, converting images to a
     stable published format; the response is rejected if the clipboard
     identity no longer matches the checkpoint
  │
  └─ Emacs inserts the link to the stored asset
```

The identity checkpoint is the correctness mechanism: it guarantees the asset written is the asset the user saw, even though inspection and ingestion are two separate subprocess calls.

## 5. Org browser preview

```
user: open preview
  │
  ├─ 23-org-preview.el starts imoogi-org-preview-command:
  │     imoogi-org-preview --host 127.0.0.1 --port 0 --print-bootstrap-json
  │
  ├─ Go: orgpreview.NewServer{token: generated when empty}
  │        Listen on loopback, ephemeral port
  │        print ONE bootstrap JSON line: {port, token}
  │
  ├─ Emacs reads that line and now knows where and how to talk
  ├─ Emacs pushes buffer content / cursor position over loopback HTTP
  │     └─ internal/orgpreview: parser → ir → render → web assets
  └─ browse-url opens the page; the browser renders and follows the position
```

Everything is loopback-only and token-gated; the server is a child process of the Emacs session and is stopped with it.

## 6. Toolchain fetch and setup

```
ONLINE machine
  make toolchain-setup / imoogi-toolchain fetch
    ├─ internal/config     read toolchains.json (desired state, target darwin/arm64)
    ├─ internal/fetch      download Node/npm tarballs, build gopls
    ├─ internal/activation cross-compile spec for the CLI's own bootstrap
    ├─ internal/artifact   verify and place artifacts under vendor/toolchains/
    └─ write toolchains.lock.json deterministically
  make provenance-generate
    └─ internal/provenance  hash every covered file → provenance/*.json
                            + vendor-manifest.json
  git commit ; carry the repository across the air gap
       │
OFFLINE machine
  imoogi-toolchain setup
    ├─ verify the locked artifacts against the lockfile
    ├─ stage them under .local/
    ├─ probe them in a controlled environment
    └─ atomically activate .local/bin as a relative symlink
         └─ modules/development/17-lsp.el finds the language servers there
  make provenance-verify
    └─ internal/provenance  re-hash every covered file; git-ignored files skipped
```

The split is the same online → offline discipline used for Emacs packages, applied to binaries: the online half may use the network and writes a lockfile; the offline half may only verify what is already present.
