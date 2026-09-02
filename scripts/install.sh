#!/usr/bin/env bash
# install.sh --- wire this repository into ~/.emacs.d
#
# Automates the manual steps documented in README.md's "설치" section:
#   1. symlink this repo to ~/.config/imoogi-emacs
#   2. write ~/.emacs.d/early-init.el and ~/.emacs.d/init.el as one-line
#      loaders that point at that symlink
#
# Safe to re-run: an existing, differing early-init.el/init.el is backed up
# (with a timestamp suffix) before being overwritten, never dropped silently.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_LINK="${HOME}/.config/imoogi-emacs"
EMACS_DIR="${HOME}/.emacs.d"

mkdir -p "$(dirname "$CONFIG_LINK")"
ln -sfn "$REPO_ROOT" "$CONFIG_LINK"
echo "== linked $CONFIG_LINK -> $REPO_ROOT"

mkdir -p "$EMACS_DIR"

write_loader() {
  local target="$1" source_file="$2"
  local expected
  expected="(load-file (expand-file-name \"${source_file}\" \"${CONFIG_LINK}\"))"

  if [[ -e "$target" || -L "$target" ]]; then
    if [[ -f "$target" && "$(cat "$target")" == "$expected" ]]; then
      echo "== $target already up to date"
      return
    fi
    local backup="${target}.bak.$(date +%Y%m%d%H%M%S)"
    mv "$target" "$backup"
    echo "== existing $target backed up to $backup"
  fi

  printf '%s\n' "$expected" >"$target"
  echo "== wrote $target"
}

write_loader "${EMACS_DIR}/early-init.el" "early-init.el"
write_loader "${EMACS_DIR}/init.el" "boot.el"

cat <<'EOF'

설치 완료. Emacs를 재시작하세요.

(선택) 언어 서버(LSP)까지 쓰려면 아래를 실행:
  ./scripts/setup-toolchain.sh setup
자세한 내용은 docs/toolchains.md 를 참고하세요.
EOF
