#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT_SOURCE="${PROJECT_ROOT}/scripts/setup-toolchain.sh"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/imoogi-toolchain-wrapper.XXXXXX")"
trap 'rm -rf "${TEST_ROOT}"' EXIT

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

new_fixture() {
  local name="$1" version="${2:-1.2.3}"
  local root="${TEST_ROOT}/${name}"
  mkdir -p "${root}/scripts"
  cp "${SCRIPT_SOURCE}" "${root}/scripts/setup-toolchain.sh"
  cat >"${root}/toolchains.lock.json" <<EOF
{
  "schema": "imoogi-toolchains-lock/v1",
  "cli_version": "${version}"
}
EOF
  printf '%s\n' "${root}"
}

install_fake_cli() {
  local root="$1" platform="$2" version="${3:-1.2.3}"
  local cli="${root}/vendor/toolchains/cli/${version}/${platform}/imoogi-toolchain"
  mkdir -p "$(dirname "${cli}")"
  cat >"${cli}" <<'EOF'
#!/usr/bin/env bash
printf '%s\n%s\n' "$PWD" "$*" >"${IMOOGI_FAKE_TOOLCHAIN_LOG}"
EOF
  chmod +x "${cli}"
}

darwin_root="$(new_fixture darwin)"
install_fake_cli "${darwin_root}" darwin-arm64
darwin_output="$({
  IMOOGI_TOOLCHAIN_UNAME_S=Darwin \
  IMOOGI_TOOLCHAIN_UNAME_M=arm64 \
  bash "${darwin_root}/scripts/setup-toolchain.sh" --print-command
})"
[[ "${darwin_output}" == *"toolchain platform: darwin/arm64"* ]] || fail "Darwin platform was not detected"
[[ "${darwin_output}" == *"vendor/toolchains/cli/1.2.3/darwin-arm64/imoogi-toolchain setup"* ]] || fail "locked CLI was not selected"

log="${TEST_ROOT}/fake-cli.log"
IMOOGI_TOOLCHAIN_UNAME_S=Darwin \
IMOOGI_TOOLCHAIN_UNAME_M=aarch64 \
IMOOGI_FAKE_TOOLCHAIN_LOG="${log}" \
bash "${darwin_root}/scripts/setup-toolchain.sh" >/dev/null
invocation_cwd="$(sed -n '1p' "${log}")"
invocation_args="$(sed -n '2p' "${log}")"
darwin_root_real="$(cd "${darwin_root}" && pwd -P)"
invocation_cwd_real="$(cd "${invocation_cwd}" && pwd -P)"
[[ "${invocation_cwd_real}" == "${darwin_root_real}" ]] || fail "CLI did not run from repository root"
[[ "${invocation_args}" == "setup" ]] || fail "CLI did not receive setup command"

IMOOGI_TOOLCHAIN_UNAME_S=Darwin \
IMOOGI_TOOLCHAIN_UNAME_M=arm64 \
IMOOGI_FAKE_TOOLCHAIN_LOG="${log}" \
bash "${darwin_root}/scripts/setup-toolchain.sh" fetch --dry-run >/dev/null
[[ "$(sed -n '2p' "${log}")" == "fetch --dry-run" ]] || fail "CLI options were not forwarded"

linux_root="$(new_fixture linux)"
linux_output="$({
  IMOOGI_TOOLCHAIN_UNAME_S=Linux \
  IMOOGI_TOOLCHAIN_UNAME_M=x86_64 \
  bash "${linux_root}/scripts/setup-toolchain.sh" --if-available
})"
[[ "${linux_output}" == *"toolchain platform: linux/amd64"* ]] || fail "Linux platform was not normalized"
[[ "${linux_output}" == *"language servers skipped"* ]] || fail "missing optional bootstrap was not skipped"

invalid_root="$(new_fixture invalid not-a-version)"
if IMOOGI_TOOLCHAIN_UNAME_S=Darwin \
   IMOOGI_TOOLCHAIN_UNAME_M=arm64 \
   bash "${invalid_root}/scripts/setup-toolchain.sh" --print-command >/dev/null 2>&1; then
  fail "invalid cli_version was accepted"
fi

install_root="${TEST_ROOT}/install"
install_home="${TEST_ROOT}/home"
mkdir -p "${install_root}/scripts" "${install_home}"
cp "${PROJECT_ROOT}/scripts/install.sh" "${install_root}/scripts/install.sh"
cp "${PROJECT_ROOT}/scripts/imoogi-editor" "${install_root}/scripts/imoogi-editor"
printf '%s\n' 'export KEEP_ME=yes' >"${install_home}/.zshrc"
printf '%s\n' ';;; fixture' >"${install_root}/early-init.el"
printf '%s\n' ';;; fixture' >"${install_root}/boot.el"
cat >"${install_root}/scripts/setup-toolchain.sh" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >"${IMOOGI_FAKE_INSTALL_LOG}"
EOF
chmod +x "${install_root}/scripts/setup-toolchain.sh"

install_log="${TEST_ROOT}/install-toolchain.log"
HOME="${install_home}" SHELL=/bin/zsh IMOOGI_FAKE_INSTALL_LOG="${install_log}" \
bash "${install_root}/scripts/install.sh" >/dev/null
[[ "$(cat "${install_log}")" == "--if-available" ]] || fail "install.sh did not auto-run optional setup"
[[ -L "${install_home}/.local/bin/imoogi-editor" ]] || fail "external-editor wrapper was not linked"
[[ "$(readlink "${install_home}/.local/bin/imoogi-editor")" == "${install_home}/.config/imoogi-emacs/scripts/imoogi-editor" ]] || fail "external-editor link target is wrong"
profile="${install_home}/.zshrc"
[[ "$(grep -Fxc '# >>> imoogi-emacs external editor >>>' "${profile}")" == "1" ]] || fail "editor block start marker is missing"
[[ "$(grep -Fxc "export VISUAL=\"${install_home}/.local/bin/imoogi-editor\"" "${profile}")" == "1" ]] || fail "VISUAL was not configured"
[[ "$(grep -Fxc 'export EDITOR="${VISUAL}"' "${profile}")" == "1" ]] || fail "EDITOR was not configured"
[[ "$(grep -Fxc 'export KEEP_ME=yes' "${profile}")" == "1" ]] || fail "existing shell configuration was not preserved"
expected_profile="${TEST_ROOT}/expected-zshrc"
cp "${profile}" "${expected_profile}"

rm -f "${install_log}"
HOME="${install_home}" SHELL=/bin/zsh IMOOGI_FAKE_INSTALL_LOG="${install_log}" \
bash "${install_root}/scripts/install.sh" --without-toolchain >/dev/null
[[ ! -e "${install_log}" ]] || fail "--without-toolchain still ran setup"
[[ "$(grep -Fxc '# >>> imoogi-emacs external editor >>>' "${profile}")" == "1" ]] || fail "rerun duplicated the editor block"
cmp -s "${profile}" "${expected_profile}" || fail "rerun changed an up-to-date editor block"

fake_emacsclient="${TEST_ROOT}/fake-emacsclient"
editor_log="${TEST_ROOT}/editor.log"
cat >"${fake_emacsclient}" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$@" >"${IMOOGI_FAKE_EDITOR_LOG}"
EOF
chmod +x "${fake_emacsclient}"
EMACSCLIENT="${fake_emacsclient}" IMOOGI_FAKE_EDITOR_LOG="${editor_log}" \
bash "${PROJECT_ROOT}/scripts/imoogi-editor" "+12:4" "/tmp/file with spaces.txt"
[[ "$(sed -n '1p' "${editor_log}")" == "--create-frame" ]] || fail "editor did not request a graphical frame"
[[ "$(sed -n '2p' "${editor_log}")" == "+12:4" ]] || fail "editor did not preserve the line/column argument"
[[ "$(sed -n '3p' "${editor_log}")" == "/tmp/file with spaces.txt" ]] || fail "editor did not preserve a spaced filename"
[[ "$(wc -l <"${editor_log}" | tr -d ' ')" == "3" ]] || fail "editor added unexpected arguments"

echo "setup-toolchain tests: PASS"
