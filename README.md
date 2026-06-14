# imoogi-emacs

개인 Emacs 설정. 모듈별로 분리하여 관리한다.

## 설치

```bash
# 1. 클론
git clone <repo-url> ~/workspace/imoogi-emacs

# 2. 심볼릭 링크
ln -s ~/workspace/imoogi-emacs ~/.config/imoogi-emacs

# 3. init.el 설정 (~/.emacs.d/init.el)
```

```elisp
(load-file (expand-file-name "boot.el" "~/.config/imoogi-emacs"))
```

모든 패키지가 저장소 안 `vendor/elpa/` 에 동봉돼 있어, **인터넷 없이도 첫 실행부터 그대로 동작한다**(망분리/air-gap 지원).

## 망분리(air-gap) 환경

이 설정은 폐쇄망에서 동작하도록 설계됐다. 저장소 하나만 클론해 들고 들어가면 외부 네트워크 없이 작동한다 — 부팅 경로에서 네트워크에 접근하지 않는다.

- 패키지는 `vendor/elpa/` 에 동봉(커밋)되며, 런타임에 `package-refresh-contents` 나 다운로드를 하지 않는다.
- 커밋된 `vendor/` 디렉터리 자체가 lock 역할(git 커밋 = 버전 동결). `packages.lock` 은 사람이 읽는 버전 감사용.
- 빌드 머신과 타겟의 **Emacs 메이저 버전을 일치**시킬 것(.elc 호환).

### 패키지 추가/업데이트 (온라인 빌드 머신에서만)

```bash
# 1. packages.el 의 imoogi-required-packages 수정 (추가/삭제 시)
# 2. vendoring 재실행
emacs --batch -Q -l scripts/vendor.el            # 누락분만 설치
emacs --batch -Q -l scripts/vendor.el -- upgrade # 전체 최신으로 갱신
# 3. 변경 커밋
git add vendor/ packages.lock packages.el && git commit -m "vendor: update packages"
# 4. 폐쇄망으로 반입 (내부 git 미러 pull 또는 저장소 재반입)
```

폐쇄망 안에서는 절대 vendoring 을 돌리지 않는다(네트워크 필요). 업데이트는 항상 온라인 머신 → 반입 순서다.

### 폰트

폰트는 `assets/fonts/` 에 동봉되며, 첫 부팅 시 OS 폰트 디렉터리(macOS: `~/Library/Fonts/`, Linux: `~/.local/share/fonts/`)로 **자동 복사**된다(로컬 복사, 네트워크 불필요). 복사 후 **Emacs 재시작**하면 적용된다.

- **NanumGothicCoding** (나눔고딕코딩) — 기본 코딩 폰트(한글/영문 고정폭). `imoogi-font-family` / `imoogi-font-size` 로 조정.
- **NFM.ttf** (Symbols Nerd Font Mono) — doom-modeline 아이콘용. 없으면 아이콘만 □ 로 보이고 기능은 정상.

## 주요 키바인딩

| 키 | 기능 |
|----|------|
| `C-c h` | 마스터 hydra (창, 프로젝트, Git, 줌, treemacs) |
| `C-c p` | projectile 커맨드 맵 |
| `C-c e` | cape (completion-at-point 접두 맵) |
| `M-o` | ace-window (창 점프) |
| `S-SPC` | 한영 전환 |
| `M-x` | vertico 세로 완성 |
| `C-s` / `M-s l` | consult-line (현재 버퍼 검색) |
| `C-x b` | consult-buffer (버퍼 전환) |
| `M-s r` | consult-ripgrep (프로젝트 검색) |
| `M-g g` | consult-goto-line · `M-g i` consult-imenu |
| `C-.` / `C-;` | embark-act / embark-dwim (컨텍스트 액션) |
| `C-'` | avy 점프 |
| `C-z` / `C-S-z` | undo-fu undo / redo |

## 패키지 관리

- **package.el + use-package** — 모든 패키지를 단일 메커니즘으로 관리
- **vendoring** — `vendor/elpa/` 에 동봉(망분리 지원). straight.el 은 제거됨
- **`packages.el`** — 필요 패키지 단일 목록(SSOT)
- **`scripts/vendor.el`** — 온라인 머신에서 vendor/ 채우기·갱신
- **`packages.lock`** — 동결된 패키지 버전 기록(감사용)

## minimal-emacs.d 추천 셋업 반영

[minimal-emacs.d](https://github.com/jamescherti/minimal-emacs.d) README 가 권장하는 패키지/설정을 imoogi 구조에 맞게 도입했다. (완성 스택은 사용자 선택에 따라 ivy/counsel → vertico 로 이관)

### 도입한 패키지 (모듈별)

| 모듈 | 패키지 | 용도 |
|------|--------|------|
| `02-completion` | vertico · orderless · marginalia · embark · embark-consult · consult · corfu · cape | 미니버퍼/버퍼 내 완성 스택 |
| `11-editing` | undo-fu(+session) · yasnippet(+snippets) · apheleia · dumb-jump · stripspace · elec-pair | undo, 스니펫, 비동기 포매팅, go-to-def, 공백정리, 괄호짝 |
| `12-navigation` | avy · helpful · diff-hl · bufferfile | 점프, 향상된 도움말, 여백 Git 표시, 파일 조작 |
| `13-system` | exec-path-from-shell · server · buffer-terminator · persist-text-scale | 환경변수 동기화, 서버, 버퍼 정리, 텍스트 배율 유지 |
| `14-org-markdown` | org · org-appear · markdown-toc | org/markdown |
| `15-elisp` | aggressive-indent · highlight-defined · paredit · page-break-lines · elisp-refs | Elisp 개발 |
| `16-languages` | git-modes · yaml · dockerfile · gnuplot · lua · jinja2 · csv · go · rust · crontab · nginx · hcl · nix · fish · vimrc · jenkinsfile (+내장 sgml) | 16종 파일타입 모드 |
| `17-folding` | kirigami · outline-indent (+내장 outline/hs-minor) | 코드 폴딩 (`C-c z` 접두) |
| `18-terminal` | vterm | libvterm 기반 터미널 (`C-c t`, 모듈은 타겟 첫 실행 시 빌드) |
| `19-native-compile` | compile-angel | 로드 시 바이트/네이티브 컴파일 |
| `00-defaults` | (내장) | 상대 줄번호, 줄:열 표시, treesit 레벨4, pixel-scroll, fringe |

### 이미 반영돼 있던 추천 (중복 도입 안 함)

`recentf` · `savehist` · `saveplace` · `auto-revert` (00-defaults/09), `eglot`/`flymake` 기본값 (00-defaults), `which-key`(Emacs 30 내장, 03), `uniquify`, `treemacs`(07), `magit`(06), 폰트·테마(10).

### 의도적으로 미반영 (이유 명시)

| 추천 | 미반영 이유 |
|------|-------------|
| `auto-package-update` | 네트워크로 자동 업데이트 → **망분리 철학과 정면 충돌**. 업데이트는 온라인 머신 vendoring 으로만. |
| `treesit-fold` | 언어별 tree-sitter 문법(별도 설치/빌드)이 필요 → air-gap 부적합. 문법 갖춘 환경이면 추가 가능. |
| `inhibit-mouse` | 마우스를 끄는 동작은 과격 — 문서화만. |
| `evil` (vim 키) | 사용자 선택으로 미사용 (treemacs-evil 의존성으로 vendor 에는 존재). |
| `easysession` · `quick-sdcv` · `eat` | 선택사항. 필요하면 `packages.el` 에 추가 후 재-vendoring. |

위 미반영 패키지를 쓰려면 `packages.el` 의 `imoogi-required-packages` 에 추가하고 온라인 머신에서 `scripts/vendor.el` 을 재실행하면 된다.

### vterm (터미널) 빌드 — 타겟 첫 실행 시

vterm 의 **elisp 는 vendor 에 동봉**되지만, 고성능을 내는 네이티브 모듈(`vterm-module`)은 C 라이브러리(libvterm) 기반이라 **타겟 머신에서 직접 빌드**해야 한다. 컴파일된 모듈은 OS·아키텍처에 묶이므로 동봉하지 않고, 타겟에서 최초 1회 빌드하는 방식을 택했다.

**1) 타겟에 빌드 도구 준비** (폐쇄망이면 내부 미러/사전설치 이미지로):

```bash
# macOS
brew install cmake libtool libvterm

# Debian/Ubuntu
sudo apt install cmake libtool-bin libvterm-dev
```

> 시스템 `libvterm` 을 반드시 설치할 것. 없으면 cmake 가 libvterm 소스를 **인터넷에서 받으려 시도**하므로 폐쇄망에서 빌드가 실패한다.

**2) 최초 실행 시 빌드**

- Emacs 에서 `M-x vterm` (또는 `C-c t`) 실행 → 모듈이 없으면 `Compile vterm-module?` 확인이 뜬다. `y` 입력하면 빌드된다.
- 또는 수동으로 `M-x vterm-module-compile`.
- 빌드 결과 `vterm-module.so`(또는 `.dylib`)는 `vendor/elpa/vterm-*/` 에 생성된다. 한 번 빌드하면 이후 실행은 바로 동작한다.

**참고**: 빌드 도구를 둘 수 없는 타겟이라면, 동일 OS/아키텍처의 빌드 머신에서 빌드한 `vterm-module.*` 를 `vendor/elpa/vterm-*/` 에 복사해 동봉해도 된다. 순수 elisp 터미널을 원하면 `eat` 패키지가 대안이다(빌드 불필요).

## 라이선스 / 글꼴 출처

이 저장소에 동봉된 글꼴(`assets/fonts/`)은 각 오픈소스 라이선스에 따라 재배포된다.

| 글꼴 | 출처 | 라이선스 |
|------|------|----------|
| 나눔고딕코딩 (NanumGothicCoding) | [naver/nanumfont](https://github.com/naver/nanumfont) © NHN Corporation | SIL Open Font License 1.1 |
| Symbols Nerd Font Mono (NFM.ttf) | [nerd-icons.el](https://github.com/rainstormstudio/nerd-icons.el) | MIT / OFL (각 심볼 세트별) |

- 나눔고딕코딩은 **SIL Open Font License (OFL) 1.1** 하에 배포되며, 라이선스를 명시하면 상용 소프트웨어 포함 재배포가 허용된다. 전문은 [`assets/fonts/OFL.txt`](assets/fonts/OFL.txt) 참조.
- OFL 조건에 따라 글꼴 원본과 라이선스 전문을 함께 동봉한다.
