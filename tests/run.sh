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
IMOOGI_TEST_ROOT="$ROOT_DIR" \
"$EMACS_BIN" --batch -Q \
  --eval "(setq user-emacs-directory (file-name-as-directory \"$TEST_TMP/boot-user-emacs.d\"))" \
  --eval "(setq load-prefer-newer t)" \
  --eval "(setq package-archives nil)" \
  --eval "(defvar compile-angel-excluded-path-regexps nil)" \
  --eval "(add-to-list 'compile-angel-excluded-path-regexps (concat \"^\" (regexp-quote (file-name-as-directory (expand-file-name (getenv \"IMOOGI_TEST_ROOT\"))))))" \
  -l tests/boot-health.el \
  --eval "(imoogi-boot-health-start-capture)" \
  -l boot.el \
  --eval "(imoogi-boot-health-assert)" \
  --eval "(when (fboundp 'compile-angel-exclude-directory) (compile-angel-exclude-directory (getenv \"IMOOGI_TEST_ROOT\")))" \
  --eval "(when (fboundp 'compile-angel-on-load-mode) (compile-angel-on-load-mode -1))"

echo "== ert =="
IMOOGI_TEST_USER_DIR="$TEST_TMP/ert-user-emacs.d" \
  "$EMACS_BIN" --batch -Q -l tests/run.el
