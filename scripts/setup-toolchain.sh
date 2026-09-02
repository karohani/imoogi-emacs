#!/usr/bin/env bash
# setup-toolchain.sh --- run vendored imoogi-toolchain with auto path detection
#
# Keeps current convention: this repository ships offline artifacts under
# vendor/toolchains/, and bootstrap is selected from toolchains.json
# (cli_version + target).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANIFEST="${REPO_ROOT}/toolchains.json"
CLI_CMD="${1:-setup}"

if [[ ! -f "${MANIFEST}" ]]; then
  echo "toolchains.json not found: ${MANIFEST}" >&2
  exit 1
fi

extract_json_field() {
  local file="$1"
  local key="$2"
  local value
  value="$(sed -n "s/.*\"${key}\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$file" | head -n 1)"
  echo "$value"
}

cli_version="$(extract_json_field "$MANIFEST" "cli_version")"
target_os="$(extract_json_field "$MANIFEST" "os")"
target_arch="$(extract_json_field "$MANIFEST" "arch")"

if [[ -z "$cli_version" || -z "$target_os" || -z "$target_arch" ]]; then
  echo "toolchains.json 에서 cli_version/os/arch 파싱 실패" >&2
  echo "  manifest: $MANIFEST" >&2
  exit 1
fi

toolchain_path="${REPO_ROOT}/vendor/toolchains/cli/${cli_version}/${target_os}-${target_arch}/imoogi-toolchain"
if [[ ! -x "$toolchain_path" ]]; then
  echo "지원되는 bootstrap binary를 찾지 못했습니다." >&2
  echo "  expected: $toolchain_path" >&2
  echo "  manifest target: ${target_os}/${target_arch}, cli=${cli_version}" >&2
  echo "  toolchains.json: $MANIFEST" >&2
  exit 1
fi

run() {
  local command="$1"
  shift || true
  "$toolchain_path" "$command" "$@"
}

case "$CLI_CMD" in
  setup|version|fetch|--help|-h|help)
    ;;
  *)
    echo "알 수 없는 명령입니다: $CLI_CMD" >&2
    echo "사용: $0 [setup|version|fetch|--help]" >&2
    exit 2
    ;;
esac

run "$CLI_CMD"

