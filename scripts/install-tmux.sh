#!/usr/bin/env bash
# install-tmux.sh --- 저장소의 tmux 기본 설정을 이 머신에 연결한다
#
# 선택 사항이다. Emacs 설정은 tmux 없이도 완전히 동작하며, 이 스크립트를
# 돌리지 않으면 tmux 는 아무 영향도 받지 않는다.
#
#   ./scripts/install-tmux.sh            # 설치
#   ./scripts/install-tmux.sh --check    # 설정 문법·옵션만 검사하고 끝 (아무것도 안 바꿈)
#
# 하는 일은 install.sh 와 같은 모양이다.
#   1. 이 저장소를 ~/.config/imoogi-emacs 로 심볼릭 링크
#   2. ~/.tmux.conf 를 그 링크의 tmux/tmux.conf 를 읽는 한 줄로 교체
#      (기존 파일은 타임스탬프를 붙여 백업한다 — 조용히 버리지 않는다)
#
# [HARD] 돌고 있는 tmux 서버는 절대 건드리지 않는다. kill-server 도, 기존
# 세션에 대한 source-file 도 하지 않는다. 이 머신에서는 tmux 안에서 장시간
# 작업이 돌고 있을 수 있고, 설정 설치가 그걸 끊어먹어서는 안 된다.
# 적용은 사용자가 직접 `tmux source-file ~/.tmux.conf` 를 부르거나 새 세션을
# 열 때 일어난다.
#
# [HARD] 검사는 반드시 `-f /dev/null` 로 띄운 임시 서버에서 한다. 그러지
# 않으면 임시 서버가 사용자의 기존 ~/.tmux.conf 까지 읽어버려서, 남의 설정
# 값을 이 파일의 검사 결과로 착각하게 된다(실측으로 한 번 걸렸다).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_LINK="${HOME}/.config/imoogi-emacs"
SOURCE_CONF="${ROOT_DIR}/tmux/tmux.conf"
TARGET_CONF="${HOME}/.tmux.conf"
LOADER="source-file ${CONFIG_LINK}/tmux/tmux.conf"

CHECK_ONLY=0
[[ "${1:-}" == "--check" ]] && CHECK_ONLY=1

die() { echo "!! $*" >&2; exit 1; }

## ---------------------------------------------------------------- 임시 서버

# 소켓 이름에 PID 를 넣어 이 스크립트 전용으로 만든다. 사용자의 기본 소켓과
# 절대 겹치지 않는다.
PROBE_SOCKET="imoogi-tmux-check-$$"
PROBE_UP=0

cleanup_probe() {
  [[ "$PROBE_UP" -eq 1 ]] || return 0
  tmux -L "$PROBE_SOCKET" kill-server 2>/dev/null || true
  # kill-server 는 소켓 파일까지 지우지는 않는다(실측). 임시 이름이라 남겨두면
  # /tmp 에 쓰레기가 쌓이므로 직접 지운다.
  rm -f "${TMUX_TMPDIR:-/tmp}/tmux-$(id -u)/${PROBE_SOCKET}" 2>/dev/null || true
  PROBE_UP=0
}
# RETURN 이 아니라 EXIT 인 이유: die 는 exit 를 부르는데 RETURN 트랩은
# exit 에서 발동하지 않아, 검사 실패 시 임시 서버가 남는다.
trap cleanup_probe EXIT

start_probe() {
  tmux -f /dev/null -L "$PROBE_SOCKET" new-session -d 'sleep 120' \
    || die "임시 tmux 서버를 띄우지 못했습니다."
  PROBE_UP=1
}

# 파일 하나를 임시 서버에서 실제로 읽혀 본다. 문법 오류·모르는 명령이면
# tmux 가 줄 번호와 함께 오류를 내고 0 이 아닌 값으로 끝난다.
probe_source() {
  tmux -L "$PROBE_SOCKET" source-file "$1"
}

# 값이 정말 들어갔는지 되읽어 확인한다. tmux 는 옵션 종류(session/window/pane)를
# 틀리게 지정해도 오류를 내지 않고 조용히 무시하기 때문에(실측), 문법 검사만
# 으로는 부족하다.
expect_option() {
  local desc="$1" scope="$2" name="$3" want="$4" got
  got="$(tmux -L "$PROBE_SOCKET" show-options "$scope" "$name" 2>/dev/null || true)"
  if [[ "$got" != "$want" ]]; then
    die "설정 검사 실패 — ${desc}: ${name} 이 '${want}' 여야 하는데 '${got}' 입니다."
  fi
  printf '   %-26s %s\n' "$name" "$got"
}

## ---------------------------------------------------------------- 사전 확인

command -v tmux >/dev/null 2>&1 || {
  echo "!! tmux 가 없습니다. 먼저 설치하세요." >&2
  case "$(uname -s)" in
    Darwin) echo "     brew install tmux" >&2 ;;
    Linux)  echo "     sudo apt install tmux   (또는 dnf/pacman)" >&2 ;;
  esac
  exit 1
}
[[ -f "$SOURCE_CONF" ]] || die "설정 파일이 없습니다: $SOURCE_CONF"

echo "== tmux $(tmux -V | awk '{print $2}') · 설정 원본 $SOURCE_CONF"

## ---------------------------------------------------------------- 1. 설정 검사

echo "== 설정을 임시 서버에서 검사합니다 (기존 세션은 건드리지 않습니다)"
start_probe
probe_source "$SOURCE_CONF" || die "설정에 오류가 있습니다. 위 메시지의 줄 번호를 확인하세요."

expect_option "마우스"       -gv  mouse            on
expect_option "스크롤백"     -gv  history-limit    50000
expect_option "포커스 전달"  -gv  focus-events     on
expect_option "클립보드"     -gv  set-clipboard    on
expect_option "복사 모드 키" -gwv mode-keys        emacs
expect_option "pane 경계선"  -gwv pane-border-lines heavy
expect_option "커서 색"      -gpv cursor-colour    "#51afef"
cleanup_probe
echo "== 설정 검사 통과"

if [[ "$CHECK_ONLY" -eq 1 ]]; then
  echo "== --check 였으므로 아무것도 바꾸지 않고 끝냅니다."
  exit 0
fi

## ---------------------------------------------------------------- 2. 링크

mkdir -p "$(dirname "$CONFIG_LINK")"
ln -sfn "$ROOT_DIR" "$CONFIG_LINK"
echo "== 연결됨 $CONFIG_LINK -> $ROOT_DIR"

## ---------------------------------------------------------------- 3. 로더 쓰기

BACKUP=""
if [[ -e "$TARGET_CONF" || -L "$TARGET_CONF" ]]; then
  if [[ -f "$TARGET_CONF" && "$(cat "$TARGET_CONF")" == "$LOADER" ]]; then
    echo "== $TARGET_CONF 는 이미 최신입니다"
  else
    BACKUP="${TARGET_CONF}.bak.$(date +%Y%m%d%H%M%S)"
    mv "$TARGET_CONF" "$BACKUP"
    echo "== 기존 $TARGET_CONF 를 $BACKUP 로 백업했습니다"
  fi
fi

if [[ ! -f "$TARGET_CONF" ]]; then
  printf '%s\n' "$LOADER" >"$TARGET_CONF"
  echo "== $TARGET_CONF 를 썼습니다"
fi

## ---------------------------------------------------------------- 4. 사후 검사

# 방금 쓴 파일이 실제로 읽히는지까지 확인한다. 여기서 실패하면 백업을 되돌린다.
echo "== 방금 쓴 $TARGET_CONF 를 다시 검사합니다"
start_probe
if ! probe_source "$TARGET_CONF"; then
  cleanup_probe
  if [[ -n "$BACKUP" ]]; then
    mv "$BACKUP" "$TARGET_CONF"
    die "설치한 설정이 읽히지 않아 백업을 되돌렸습니다: $TARGET_CONF"
  fi
  die "설치한 설정이 읽히지 않습니다: $TARGET_CONF"
fi
expect_option "로더 경유 마우스" -gv mouse on
cleanup_probe

## ---------------------------------------------------------------- 안내

cat <<EOF

설치 완료.

지금 돌고 있는 tmux 세션은 그대로 둡니다. 적용하려면 둘 중 하나:
  · 돌고 있는 tmux 안에서   prefix + r        (기본 prefix 는 Ctrl-b)
  · 또는 셸에서             tmux source-file ~/.tmux.conf

일부 항목(focus-events 등)은 클라이언트를 다시 붙여야 완전히 반영됩니다:
  tmux detach  후 tmux attach

개인 취향 설정은 여기에 두세요 — 기본 설정 뒤에 마지막으로 읽힙니다:
  ~/.tmux.local.conf
EOF

if [[ -n "$BACKUP" ]]; then
  cat <<EOF

되돌리려면 백업 한 개만 복원하면 끝입니다:
  mv "$BACKUP" "$TARGET_CONF"
EOF
fi
