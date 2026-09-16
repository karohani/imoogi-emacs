# Clipboard platform support

`imoogi-clip` is designed for offline use. A platform is supported only after its adapter passes identity, screenshot, file-list, focus, and expected-identity tests in a native GUI session.

## Current implementation gate

| Platform cell | Adapter strategy | Dependency policy | Status |
|---|---|---|---|
| macOS arm64 | AppKit `NSPasteboard` through a repository-built cgo adapter | Apple system frameworks only; no downloaded runtime helper | Implemented; native GUI acceptance incomplete |
| Windows amd64/arm64 | Native Win32 clipboard formats (`CF_HDROP`, DIB/PNG, Unicode text) | Standard library/syscall surface or a vendored, provenance-recorded helper | Design gate only; native evidence required |
| Linux X11 | X11 selection adapter with explicit MIME targets | Any library/tool must be present in the offline build manifest | Design gate only; named desktop evidence required |
| Linux Wayland | Compositor/session-specific data-control or approved system adapter | No generic support claim; each helper needs offline provenance | Design gate only; each compositor/session requires evidence |

Unsupported cells must return a stable `unsupported-capability` result. They must not silently claim asset support because text paste still works.

## macOS spike evidence

Validated on 2026-09-16:

- macOS 26.6.2, Darwin arm64;
- Go 1.26.4 with `CGO_ENABLED=1`;
- Apple `/usr/bin/clang` successfully compiled and linked an ARC Objective-C probe against AppKit;
- the probe read `NSPasteboard.generalPasteboard.changeCount` and advertised types without reading clipboard payload bytes;
- `/usr/bin/osascript`, `pbcopy`, and `pbpaste` exist, but the primary adapter does not depend on those subprocesses.

`NSPasteboard.changeCount` is the macOS clipboard identity used by `inspect`, `checkpoint`, and `expected_clipboard_id` validation. File lists use pasteboard URL/file representations; images use advertised PNG/TIFF data converted to a stable published image format by the adapter. Text remains an Emacs-native yank path.

The native GUI acceptance matrix still has to prove focus behavior, a local kill followed by a screenshot identity change, copied files, multiple-file ordering, and folder rejection. Compilation evidence alone is not a complete platform claim.

## Provenance decision

The first implementation uses only Go's standard library, repository code, and macOS system frameworks. It adds no Go module, runtime download, Homebrew requirement, or boot-path network access. Windows and Linux adapters remain build-tagged unsupported stubs until their native cells and dependency choices are verified.

## Required evidence per platform cell

1. Bounded metadata-only `inspect`.
2. A checkpoint taken after Emacs exports a local kill.
3. Stable identity and rejection when `expected_clipboard_id` changes.
4. Screenshot PNG/JPEG ingestion.
5. Ordered one/multiple regular-file ingestion.
6. Directory rejection before publication.
7. Local drag-and-drop path import.
8. Unsaved staging followed by first-save finalization.
9. Exact OS, architecture, Emacs build, desktop/compositor, adapter, and CLI version in the evidence record.
