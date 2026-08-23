# tests/docker/Dockerfile --- install + boot verification on a clean Linux box
#
# Parameterized over the base image so one file serves the whole matrix:
#   docker build --build-arg BASE_IMAGE=alpine:edge --build-arg TOOL_PKGS="git ripgrep" ...
#
# BASE_IMAGE must ship Emacs 30.x — vendored .elc files under vendor/elpa/ are
# byte-compiled for Emacs 30 and silently degrade on 28/29 (measured: Emacs 29.3
# skips 03-which-key, 06-git, 12-navigation yet still exits 0). tests/docker/run.sh
# holds the verified list of usable images.
#
# TOOL_PKGS is the external-tool axis. Installing them exercises portability
# (nothing may be skipped); leaving them out exercises the degradation contract
# (exactly the declared modules may be skipped).
#
# Out of scope by construction: vendor/ghostel-module (Mach-O arm64) and
# vendor/toolchains (darwin-arm64) cannot run on Linux at all — those stay on
# the macOS host suite.
ARG BASE_IMAGE=debian:trixie-slim
FROM ${BASE_IMAGE}

ARG TOOL_PKGS=""

# bash is required by scripts/install.sh and is not present on Alpine.
RUN set -eu; \
    if command -v apt-get >/dev/null 2>&1; then \
      apt-get update -qq; \
      DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        emacs-nox bash ca-certificates ${TOOL_PKGS}; \
      rm -rf /var/lib/apt/lists/*; \
    elif command -v apk >/dev/null 2>&1; then \
      apk add --no-cache emacs-nox bash ca-certificates ${TOOL_PKGS}; \
    else \
      echo "unsupported package manager in ${BASE_IMAGE}" >&2; exit 1; \
    fi

WORKDIR /opt/imoogi-emacs
COPY . .

RUN ./scripts/install.sh

# assert-boot.el drives ~/.emacs.d (what install.sh actually wired up) and owns
# the verdict, so the exit code here is meaningful. IMOOGI_EXPECTED_SKIPS is
# supplied per run by tests/docker/run.sh.
#
# IMOOGI_RUN_ERT=1 additionally runs the full host suite (check-parens, offline
# boot, ERT) in the container, so the ERT tests are repeatable on every image in
# the matrix rather than only on the maintainer's macOS box. Only meaningful in
# the T2 tier, where the external tools those tests need are installed.
CMD ["sh", "-c", "emacs --batch -l /opt/imoogi-emacs/tests/assert-boot.el && \
  { [ \"${IMOOGI_RUN_ERT:-0}\" != 1 ] || bash /opt/imoogi-emacs/tests/run.sh; }"]
