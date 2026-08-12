#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EMACS_BIN="${EMACS:-emacs}"
TEST_TMP="$(mktemp -d "${TMPDIR:-/tmp}/imoogi-emacs-tests.XXXXXX")"

cleanup() {
  rm -rf "$TEST_TMP"
}
trap cleanup EXIT

cd "$ROOT_DIR"

echo "== check-parens =="
"$EMACS_BIN" --batch -Q \
  --eval "(dolist (file '(\"early-init.el\" \"boot.el\" \"packages.el\")) (with-temp-buffer (insert-file-contents file) (emacs-lisp-mode) (check-parens)))" \
  --eval "(dolist (file (directory-files \"modules\" t \"\\\\.el\\\\'\")) (with-temp-buffer (insert-file-contents file) (emacs-lisp-mode) (check-parens)))" \
  --eval "(dolist (file (directory-files \"tests\" t \"\\\\.el\\\\'\")) (with-temp-buffer (insert-file-contents file) (emacs-lisp-mode) (check-parens)))"

echo "== offline boot =="
"$EMACS_BIN" --batch -Q \
  --eval "(setq user-emacs-directory (file-name-as-directory \"$TEST_TMP/boot-user-emacs.d\"))" \
  --eval "(setq load-prefer-newer t)" \
  --eval "(setq package-archives nil)" \
  -l boot.el

echo "== ert =="
IMOOGI_TEST_USER_DIR="$TEST_TMP/ert-user-emacs.d" \
  "$EMACS_BIN" --batch -Q -l tests/run.el
