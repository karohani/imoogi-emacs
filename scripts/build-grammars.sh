#!/usr/bin/env bash
# build-grammars.sh --- tree-sitter 문법을 vendor/tree-sitter/ 로 빌드해 넣는다
#
# 온라인 머신 전용. 폐쇄망에서는 실행하지 않는다 — 결과물(.dylib/.so)이 저장소에
# 커밋돼 있으므로 클론만 하면 그대로 동작한다.
#
#   ./scripts/build-grammars.sh          # 전체
#   ./scripts/build-grammars.sh go json  # 일부만
#
# 왜 imoogi-toolchain 파이프라인을 쓰지 않는가: 문법은 Emacs 가
# `treesit-extra-load-path' 로 직접 읽는 단일 공유 라이브러리라, 툴체인이 제공하는
# staging/probe/활성화 심볼릭 링크가 필요 없다. 18-languages.el 이 이미
# vendor/tree-sitter/ 를 그 경로에 넣고 있어 추가 배선도 없다.
#
# 플랫폼: 결과물은 빌드한 OS/아키텍처 전용이다(macOS -> .dylib, Linux -> .so).
# 다른 플랫폼으로 반입하려면 그 플랫폼에서 다시 실행해야 한다.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$ROOT_DIR/vendor/tree-sitter"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

case "$(uname -s)" in
  Darwin) EXT="dylib" ;;
  Linux)  EXT="so" ;;
  *) echo "지원하지 않는 OS: $(uname -s)" >&2; exit 1 ;;
esac

# language|repo|tag|subdir  — tag 를 고정해 재현 가능하게 유지한다.
GRAMMARS=(
  "json|tree-sitter/tree-sitter-json|v0.24.8|"
  "javascript|tree-sitter/tree-sitter-javascript|v0.23.1|"
  "typescript|tree-sitter/tree-sitter-typescript|v0.23.2|typescript/"
  "tsx|tree-sitter/tree-sitter-typescript|v0.23.2|tsx/"
  "python|tree-sitter/tree-sitter-python|v0.23.6|"
  "go|tree-sitter/tree-sitter-go|v0.23.4|"
  "java|tree-sitter/tree-sitter-java|v0.23.5|"
  "yaml|ikatyang/tree-sitter-yaml|v0.5.0|"
)

mkdir -p "$OUT_DIR"
wanted=("$@")
built=0

for entry in "${GRAMMARS[@]}"; do
  IFS='|' read -r lang repo tag sub <<<"$entry"
  if [[ ${#wanted[@]} -gt 0 ]]; then
    printf '%s\n' "${wanted[@]}" | grep -qx "$lang" || continue
  fi

  echo "== $lang ($repo $tag)"
  clone="$WORK_DIR/$lang"
  git clone -q --depth 1 --branch "$tag" "https://github.com/$repo.git" "$clone"
  src="$clone/${sub}src"
  [[ -f "$src/parser.c" ]] || { echo "  parser.c 없음 — 건너뜀" >&2; continue; }

  # scanner 는 C 또는 C++ 이며, C++ 이면 c++ 로 링크해야 한다(yaml 이 그 경우).
  sources=("$src/parser.c")
  compiler=cc
  if [[ -f "$src/scanner.c" ]]; then
    sources+=("$src/scanner.c")
  elif [[ -f "$src/scanner.cc" ]]; then
    sources+=("$src/scanner.cc")
    compiler=c++
  fi

  "$compiler" -shared -fPIC -O2 -I "$src" \
    -o "$OUT_DIR/libtree-sitter-$lang.$EXT" "${sources[@]}" 2>/dev/null
  built=$((built + 1))
done

echo
echo "$built 개 문법을 $OUT_DIR 에 설치했습니다."
echo "확인: emacs --batch -l boot.el --eval '(princ (treesit-language-available-p (quote go)))'"
echo "결과물을 커밋해야 폐쇄망 타겟에서 동작합니다."
