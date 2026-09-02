#!/usr/bin/env bash
# install.sh --- wire this repository into ~/.emacs.d
#
# Automates the manual steps documented in README.md's "설치" section:
#   1. symlink this repo to ~/.config/imoogi-emacs
#   2. write ~/.emacs.d/early-init.el and ~/.emacs.d/init.el as one-line
#      loaders that point at that symlink
#   3. configure VISUAL/EDITOR for Claude Code and Codex external editing
#   4. detect and set up a compatible vendored language-server bundle
#
# Safe to re-run: an existing, differing early-init.el/init.el is backed up
# (with a timestamp suffix) before being overwritten, never dropped silently.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_LINK="${HOME}/.config/imoogi-emacs"
EMACS_DIR="${HOME}/.emacs.d"
EDITOR_BIN_DIR="${HOME}/.local/bin"
EDITOR_LINK="${EDITOR_BIN_DIR}/imoogi-editor"
INSTALL_TOOLCHAIN=1

usage() {
  cat <<'EOF'
Usage: scripts/install.sh [options]

Options:
      --without-toolchain  Install Emacs configuration without optional language servers
  -h, --help               Show help
EOF
}

while (($#)); do
  case "$1" in
    --without-toolchain) INSTALL_TOOLCHAIN=0 ;;
    -h|--help) usage; exit 0 ;;
    *)
      echo "install.sh: unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

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

shell_profile() {
  case "$(basename "${SHELL:-}")" in
    zsh) printf '%s\n' "${HOME}/.zshrc" ;;
    bash) printf '%s\n' "${HOME}/.bashrc" ;;
    *) printf '%s\n' "${HOME}/.profile" ;;
  esac
}

configure_external_editor() {
  local profile start_marker end_marker start_count end_count tmp
  profile="$(shell_profile)"
  start_marker="# >>> imoogi-emacs external editor >>>"
  end_marker="# <<< imoogi-emacs external editor <<<"

  mkdir -p "${EDITOR_BIN_DIR}"
  ln -sfn "${CONFIG_LINK}/scripts/imoogi-editor" "${EDITOR_LINK}"
  echo "== linked ${EDITOR_LINK} -> ${CONFIG_LINK}/scripts/imoogi-editor"

  mkdir -p "$(dirname "${profile}")"
  touch "${profile}"
  start_count="$(grep -Fxc "${start_marker}" "${profile}" || true)"
  end_count="$(grep -Fxc "${end_marker}" "${profile}" || true)"
  if [[ "${start_count}" != "${end_count}" || "${start_count}" -gt 1 ]]; then
    echo "install.sh: malformed imoogi-emacs editor block in ${profile}" >&2
    return 1
  fi

  tmp="$(mktemp "${TMPDIR:-/tmp}/imoogi-editor-profile.XXXXXX")"
  if ! awk -v start="${start_marker}" -v end="${end_marker}" '
    $0 == start {
      if (skip || seen_end) exit 2
      skip = 1
      next
    }
    $0 == end {
      if (!skip) exit 2
      skip = 0
      seen_end = 1
      next
    }
    !skip { print }
    END { if (skip) exit 2 }
  ' "${profile}" >"${tmp}"; then
    rm -f "${tmp}"
    echo "install.sh: malformed imoogi-emacs editor block in ${profile}" >&2
    return 1
  fi

  # The block is intentionally last so this explicit installer choice wins
  # over older unmanaged EDITOR/VISUAL assignments without deleting them.
  {
    printf '%s\n' "${start_marker}"
    printf 'export VISUAL="%s"\n' "${EDITOR_LINK}"
    printf 'export EDITOR="${VISUAL}"\n'
    printf '%s\n' "${end_marker}"
  } >>"${tmp}"

  if cmp -s "${tmp}" "${profile}"; then
    rm -f "${tmp}"
    echo "== ${profile} editor variables already up to date"
  else
    cp "${tmp}" "${profile}"
    rm -f "${tmp}"
    echo "== configured VISUAL and EDITOR in ${profile}"
  fi
}

configure_external_editor

if ((INSTALL_TOOLCHAIN)); then
  "${REPO_ROOT}/scripts/setup-toolchain.sh" --if-available
else
  echo "== optional language servers skipped (--without-toolchain)"
fi

cat <<'EOF'

설치 완료. Emacs를 재시작하세요.
호환되는 동봉 언어 서버가 있으면 함께 설치되었습니다.
새 셸부터 Claude Code/Codex의 Ctrl+G 외부 편집기로 Emacs를 사용합니다.
자세한 내용은 docs/toolchains.md 를 참고하세요.
EOF
