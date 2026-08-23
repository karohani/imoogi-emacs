#!/usr/bin/env bash
# tests/docker/run.sh --- OS/Emacs matrix + degradation contract, in containers
#
# Two tiers, deliberately not multiplied together — they ask different questions
# and so have different pass criteria:
#
#   T2 portability : full toolset installed. Nothing may be skipped.
#   T1 degradation : an optional tool withheld. Nothing may skip or error.
#
# Running the matrix without the second criterion would just multiply false
# passes: a half-broken Emacs still exits 0 (see tests/assert-boot.el).
#
# Not covered here: the macOS-native surface (ghostel .dylib, imoogi-toolchain)
# — darwin-arm64 only, so it lives in tests/run.sh on the host.
set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Base images verified to ship Emacs 30.x (probed 2026-08-22):
#   debian:trixie-slim 30.1 glibc | ubuntu:26.04 30.2 glibc | alpine:edge 30.2 musl
# Rejected: debian:bookworm-slim (28.2), ubuntu:24.04 (29.3) — major-version
# mismatch against the vendored .elc files.
T2_IMAGES=(
  "debian:trixie-slim"
  "ubuntu:26.04"
  "alpine:edge"
)

# The degradation contract: an OPTIONAL tool is withheld and nothing may break.
# ripgrep is genuinely optional -- 11-editing guards it with executable-find, so
# its absence must change nothing at all (measured: zero skips, zero errors).
#
# git is deliberately NOT the withheld tool. It is a hard requirement of this
# config (magit is a git client), and measurement showed the no-git state is not
# a clean degradation: 07-treemacs skips while 06-git half-loads and logs a
# use-package error. Asserting that state as "expected" would have encoded a
# broken configuration as healthy.
T1_IMAGE="debian:trixie-slim"
T1_TOOL_PKGS="git"
T1_EXPECTED_SKIPS=""

PORTABILITY_PKGS="git ripgrep"

declare -a RESULTS=()
overall=0

run_case() {
  local label="$1" base="$2" pkgs="$3" expected="$4" run_ert="${6:-0}"
  local tag="imoogi-install-test:$(echo "$base" | tr ':/' '--')-${5}"

  echo
  echo "=============================================================="
  echo "== ${label}  [${base}]  tools='${pkgs:-none}'"
  echo "=============================================================="

  if ! docker build -q \
      --build-arg "BASE_IMAGE=${base}" \
      --build-arg "TOOL_PKGS=${pkgs}" \
      -f "$ROOT_DIR/tests/docker/Dockerfile" \
      -t "$tag" "$ROOT_DIR" >/dev/null; then
    echo "BUILD FAILED"
    RESULTS+=("FAIL(build)  ${label}  ${base}")
    overall=1
    return
  fi

  if docker run --rm \
      -e "IMOOGI_EXPECTED_SKIPS=${expected}" \
      -e "IMOOGI_RUN_ERT=${run_ert}" "$tag"; then
    RESULTS+=("PASS         ${label}  ${base}")
  else
    RESULTS+=("FAIL         ${label}  ${base}")
    overall=1
  fi
}

# T2 also runs the ERT suite (IMOOGI_RUN_ERT=1) so the tests are repeatable on
# every image, not just the maintainer's host. T1 deliberately does not: it
# measures the boot contract under a missing optional tool, not the test suite.
for image in "${T2_IMAGES[@]}"; do
  run_case "T2 portability+ERT" "$image" "$PORTABILITY_PKGS" "" "t2" 1
done

run_case "T1 degradation" "$T1_IMAGE" "$T1_TOOL_PKGS" "$T1_EXPECTED_SKIPS" "t1" 0

echo
echo "=============================================================="
echo "== summary"
echo "=============================================================="
for line in "${RESULTS[@]}"; do
  echo "  $line"
done

if [[ "$overall" -eq 0 ]]; then
  echo
  echo "ALL PASS"
else
  echo
  echo "SOME CASES FAILED" >&2
fi
exit "$overall"
