#!/usr/bin/env bash
# setup-toolchain.sh --- locate and run the vendored toolchain bootstrap
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK_FILE="${REPO_ROOT}/toolchains.lock.json"
CLI_COMMAND="setup"
IF_AVAILABLE=0
PRINT_COMMAND=0

usage() {
  cat <<'EOF'
Usage: scripts/setup-toolchain.sh [options] [command [command-options]]

Detect this machine's OS and architecture, select the matching vendored
imoogi-toolchain bootstrap, and run setup by default.

Commands:
  setup    Install or verify the locked language-server bundle (default)
  version  Report desired, available, and active bundle versions
  fetch    Fetch locked artifacts on an online build machine

Options:
      --if-available  Succeed without setup when no compatible bootstrap is bundled
      --print-command Print the detected platform and command without changing files
  -h, --help          Show help
EOF
}

while (($#)); do
  case "$1" in
    --if-available) IF_AVAILABLE=1 ;;
    --print-command) PRINT_COMMAND=1 ;;
    -h|--help) usage; exit 0 ;;
    *) break ;;
  esac
  shift
done

if (($#)); then
  CLI_COMMAND="$1"
  shift
fi
CLI_ARGS=("$@")
CLI_ARG_COUNT=$#

case "${CLI_COMMAND}" in
  setup|version|fetch) ;;
  help) usage; exit 0 ;;
  *)
    echo "setup-toolchain.sh: unknown command: ${CLI_COMMAND}" >&2
    usage >&2
    exit 2
    ;;
esac

# Overrides keep platform selection testable. Normal installations use uname.
uname_os="${IMOOGI_TOOLCHAIN_UNAME_S:-$(uname -s)}"
uname_arch="${IMOOGI_TOOLCHAIN_UNAME_M:-$(uname -m)}"

case "${uname_os}" in
  Darwin|darwin) target_os="darwin" ;;
  Linux|linux) target_os="linux" ;;
  *) target_os="$(printf '%s' "${uname_os}" | tr '[:upper:]' '[:lower:]')" ;;
esac

case "${uname_arch}" in
  arm64|aarch64) target_arch="arm64" ;;
  x86_64|amd64) target_arch="amd64" ;;
  *) target_arch="${uname_arch}" ;;
esac

platform="${target_os}-${target_arch}"

if [[ ! -f "${LOCK_FILE}" ]]; then
  echo "toolchain setup: missing ${LOCK_FILE}" >&2
  exit 1
fi

cli_version="$(awk -F '"' '$2 == "cli_version" { print $4; exit }' "${LOCK_FILE}")"
if [[ ! "${cli_version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "toolchain setup: invalid or missing cli_version in ${LOCK_FILE}" >&2
  exit 1
fi

cli_rel="vendor/toolchains/cli/${cli_version}/${platform}/imoogi-toolchain"
cli_path="${REPO_ROOT}/${cli_rel}"

echo "== toolchain platform: ${target_os}/${target_arch}"

if [[ ! -f "${cli_path}" || ! -x "${cli_path}" || -L "${cli_path}" ]]; then
  message="toolchain setup: no compatible vendored bootstrap: ${cli_rel}"
  if ((IF_AVAILABLE)); then
    echo "== ${message}; language servers skipped"
    exit 0
  fi
  echo "${message}" >&2
  exit 1
fi

if ((PRINT_COMMAND)); then
  printf '== would run: %s %s' "${cli_rel}" "${CLI_COMMAND}"
  if ((CLI_ARG_COUNT)); then
    printf ' %q' "${CLI_ARGS[@]}"
  fi
  printf '\n'
  exit 0
fi

echo "== running ${CLI_COMMAND} with ${cli_rel}"
(
  cd "${REPO_ROOT}"
  if ((CLI_ARG_COUNT)); then
    exec "${cli_path}" "${CLI_COMMAND}" "${CLI_ARGS[@]}"
  else
    exec "${cli_path}" "${CLI_COMMAND}"
  fi
)
