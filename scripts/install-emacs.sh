#!/usr/bin/env bash
# install-emacs.sh --- Emacs 본체를 OS 별로 설치한다
#
# 온라인 머신 전용. 폐쇄망에서는 실행하지 않는다 — 이 스크립트는 네트워크에서
# 배포본을 받아온다. 저장소 자체(설정/패키지/문법)는 클론만으로 동작하며,
# 여기서 설치하는 것은 그 설정을 돌릴 Emacs 실행 파일뿐이다.
#
#   ./scripts/install-emacs.sh                    # 기본 버전 설치
#   EMACS_VERSION=30.2 ./scripts/install-emacs.sh # 버전 지정
#   FORCE=1 ./scripts/install-emacs.sh            # 이미 있어도 덮어쓰기
#
# macOS 는 emacsformacosx.com 의 universal .dmg 를 쓴다. 현재 이 머신에 깔린
# 것과 같은 출처라 새 도구를 들일 필요가 없다.
#
# [HARD] 병렬 설치가 기본이다. /Applications/Emacs.app 을 건드리지 않고
# /Applications/Emacs-<버전>.app 으로 따로 넣는다. 새 버전에서 문제가 생겨도
# 기존 Emacs 로 즉시 되돌아갈 수 있어야 하기 때문이다.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EMACS_VERSION="${EMACS_VERSION:-31.1}"
FORCE="${FORCE:-0}"
DIST_DIR="$ROOT_DIR/.cache/dist"   # .gitignore 의 .cache/ 아래라 커밋되지 않는다

# emacsformacosx.com 은 체크섬도 서명도 게시하지 않는다(실측). 그래서 첫 설치는
# 원리상 TOFU(처음 받은 것을 믿음)다. 한 번 받아 해시를 여기 고정해 두면 그
# 다음부터는 재현 가능한 검증이 된다 — 값이 비어 있으면 검증을 건너뛰는 대신
# 계산된 해시를 출력해 고정을 안내한다.
#
# case 를 쓰는 이유: macOS 기본 bash 는 3.2 이고(실측) 연관 배열(declare -A)이
# 없다. 저장소의 다른 스크립트도 쓰지 않는다.
expected_sha256_for() {
  case "$1" in
    # 2026-08-28 에 받아 설치까지 확인한 배포본의 해시.
    31.1) printf '%s' "f0383bcbf0104947d3a46266aae43e3dc9e38fd68eeb35f11ee6b9444277b8ec" ;;
    *) printf '' ;;
  esac
}

log()  { printf '== %s\n' "$*"; }
warn() { printf '!! %s\n' "$*" >&2; }
die()  { printf '오류: %s\n' "$*" >&2; exit 1; }

# 마운트 해제는 EXIT 트랩에서 부른다(마운트 전에는 MOUNT_POINT 가 비어 no-op).
MOUNT_POINT=""
cleanup_mount() {
  [[ -n "$MOUNT_POINT" ]] || return 0
  hdiutil detach "$MOUNT_POINT" -quiet 2>/dev/null || true
  rmdir "$MOUNT_POINT" 2>/dev/null || true
}

install_macos() {
  local url="https://emacsformacosx.com/emacs-builds/Emacs-${EMACS_VERSION}-universal.dmg"
  local dmg="$DIST_DIR/Emacs-${EMACS_VERSION}-universal.dmg"
  local app="/Applications/Emacs-${EMACS_VERSION}.app"

  if [[ -e "$app" && "$FORCE" != "1" ]]; then
    log "이미 설치됨: $app"
    log "덮어쓰려면: FORCE=1 $0"
    return 0
  fi

  mkdir -p "$DIST_DIR"

  if [[ -f "$dmg" ]]; then
    log "캐시된 배포본 사용: $dmg"
  else
    log "내려받는 중: $url"
    # .part 로 받고 성공했을 때만 옮긴다 — 중단된 다운로드가 캐시로 남아
    # 다음 실행에서 "캐시 사용"으로 잘못 집히는 것을 막는다.
    curl -fL --progress-bar -o "${dmg}.part" "$url" \
      || die "다운로드 실패. 버전 번호($EMACS_VERSION)가 맞는지 확인하세요: https://emacsformacosx.com/builds"
    mv "${dmg}.part" "$dmg"
  fi

  local got expected
  got="$(shasum -a 256 "$dmg" | cut -d' ' -f1)"
  expected="$(expected_sha256_for "$EMACS_VERSION")"

  if [[ -n "$expected" ]]; then
    if [[ "$got" != "$expected" ]]; then
      rm -f "$dmg"
      die "체크섬 불일치 — 받은 파일을 삭제했습니다.
  기대: $expected
  실제: $got"
    fi
    log "체크섬 확인됨 ($EMACS_VERSION)"
  else
    warn "이 버전($EMACS_VERSION)에 고정된 체크섬이 없어 검증을 건너뜁니다."
    warn "아래 값을 이 스크립트의 EMACS_MACOS_SHA256 에 넣어두면 다음부터 검증됩니다:"
    printf '    ["%s"]="%s"\n' "$EMACS_VERSION" "$got" >&2
  fi

  # -mountpoint 로 마운트 위치를 직접 지정한다. hdiutil 출력에서 경로를 긁으면
  # 볼륨 이름이 바뀔 때 깨지고, /Volumes 아래 동명 볼륨과도 충돌한다.
  local mnt
  mnt="$(mktemp -d)"
  # 실패·중단 어느 경로로 빠져나가도 마운트가 남지 않도록 트랩으로 건다.
  # RETURN 이 아니라 EXIT 인 이유: `die` 는 exit 로 끝나고, exit 은 RETURN 트랩을
  # 발동시키지 않는다 — 마운트한 뒤 실패하면 디스크 이미지가 붙은 채로 남는다.
  MOUNT_POINT="$mnt"
  trap 'cleanup_mount' EXIT

  log "디스크 이미지 마운트"
  hdiutil attach -nobrowse -readonly -mountpoint "$mnt" "$dmg" >/dev/null

  local src
  src="$(find "$mnt" -maxdepth 1 -name '*.app' -print -quit)"
  [[ -n "$src" ]] || die "디스크 이미지 안에서 .app 을 찾지 못했습니다: $mnt"

  log "설치: $src -> $app"
  if [[ -w /Applications ]]; then
    rm -rf "$app"
    cp -R "$src" "$app"
  else
    warn "/Applications 에 쓰기 권한이 없어 sudo 로 진행합니다."
    sudo rm -rf "$app"
    sudo cp -R "$src" "$app"
  fi

  # 설치했다고 말하지 않고 실제로 실행해 확인한다.
  local reported
  reported="$("$app/Contents/MacOS/Emacs" --version | head -1)"
  log "설치 확인: $reported"

  cat <<EOF

설치 완료: $app

실행 방법
  GUI       open -a "Emacs-${EMACS_VERSION}"
  터미널    $app/Contents/MacOS/Emacs -nw
  테스트    make test EMACS=$app/Contents/MacOS/Emacs

기존 /Applications/Emacs.app 은 건드리지 않았습니다. 새 버전으로 충분히
확인한 뒤에 기본으로 삼으세요.
EOF
}

install_linux() {
  local pm=""
  for candidate in apt-get dnf pacman zypper; do
    if command -v "$candidate" >/dev/null 2>&1; then pm="$candidate"; break; fi
  done
  [[ -n "$pm" ]] || die "지원하는 패키지 관리자를 찾지 못했습니다(apt-get/dnf/pacman/zypper)."

  # 배포판 패키지는 상류 릴리스보다 한참 뒤처진다. EMACS_VERSION 을 맞춰주지
  # 못하므로, 무엇이 깔릴지 먼저 알리고 진행한다.
  warn "배포판 패키지는 버전을 고를 수 없습니다 — EMACS_VERSION=$EMACS_VERSION 은 무시됩니다."
  warn "특정 버전이 필요하면 소스 빌드를 쓰세요: https://ftp.gnu.org/gnu/emacs/"

  log "패키지 관리자: $pm"
  case "$pm" in
    apt-get) sudo apt-get update && sudo apt-get install -y emacs ;;
    dnf)     sudo dnf install -y emacs ;;
    pacman)  sudo pacman -S --needed emacs ;;
    zypper)  sudo zypper install -y emacs ;;
  esac

  log "설치 확인: $(emacs --version | head -1)"
}

case "$(uname -s)" in
  Darwin) install_macos ;;
  Linux)  install_linux ;;
  *) die "지원하지 않는 OS: $(uname -s)" ;;
esac
