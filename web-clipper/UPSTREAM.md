# Upstream

This directory is a file-only copy (no git history) of the Obsidian Web Clipper fork.

| Field | Value |
|-------|-------|
| Fork | https://github.com/karohani/obsidian-clipper |
| Upstream | https://github.com/obsidianmd/obsidian-clipper |
| Imported commit | `6d56d618b00bd970aa738d6a7a61edee27783e81` ("Lazy content script (#974)") |
| Imported on | 2026-11-08 |
| Local changes at import | `README.md` (Korean "Chrome에서 개발하며 사용하기" section, uncommitted in the fork) |

Additions made here (not in upstream): `native-host/` (Go native messaging helper) and the
Org save target in the extension. To pull upstream changes, diff a fresh checkout of the
fork against this directory and port the changes by hand.
