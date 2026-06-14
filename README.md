# imoogi-emacs

개인 Emacs 설정. 모듈별로 분리하여 관리한다.

## 설치

Emacs 30.x 기준으로 vendoring 되어 있다. 새 머신에서는 저장소를 받은 뒤
`~/.emacs.d/early-init.el` 과 `~/.emacs.d/init.el` 이 이 설정을 로드하게 만든다.

```bash
# 1. 클론
git clone <repo-url> ~/workspace/imoogi-emacs

# 2. ~/.config/imoogi-emacs 로 연결
mkdir -p ~/.config
ln -sfn ~/workspace/imoogi-emacs ~/.config/imoogi-emacs

# 3. Emacs init 디렉터리 준비
mkdir -p ~/.emacs.d
```

`~/.emacs.d/early-init.el`:

```elisp
(load-file (expand-file-name "early-init.el" "~/.config/imoogi-emacs"))
```

`~/.emacs.d/init.el`:

```elisp
(load-file (expand-file-name "boot.el" "~/.config/imoogi-emacs"))
```

기존 Emacs 설정이 있으면 위 두 파일을 덮어쓰기 전에 백업한다. 이후 Emacs 를
재시작하면 첫 부팅 때 동봉 폰트가 사용자 폰트 디렉터리로 복사되고, 다음 재시작부터
폰트가 적용된다.

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

## 상황별 단축키

### 한글 입력
| 키 | 상황 | 동작 |
|----|------|------|
| `S-SPC` | 일반 버퍼 / **ghostel 터미널** | 한/영 전환 (Emacs 내장 korean-hangul 입력기). ghostel 은 `ghostel-ime-mode` 로 터미널 안에서도 S-SPC 한글이 동작한다 |

### 명령·검색·이동 (vertico / consult)
| 키 | 동작 |
|----|------|
| `M-x` | 명령 실행 (vertico 세로 완성) |
| `C-x C-f` | 파일 열기 |
| `C-s` / `M-s l` | consult-line — 현재 버퍼 검색 |
| `C-x b` | consult-buffer — 버퍼 전환 |
| `M-s r` / `M-s g` | consult-ripgrep / grep — 프로젝트·디렉터리 검색 |
| `M-g g` | 줄 이동 · `M-g i` imenu · `M-g f` flymake 진단 |
| `M-y` | consult-yank-pop (kill-ring) |
| `C-.` / `C-;` | embark-act / embark-dwim — 후보·심볼 컨텍스트 액션 |

### 버퍼 내 자동완성 (corfu / cape)
| 키 | 동작 |
|----|------|
| `TAB` | 들여쓰기 또는 완성 (`tab-always-indent`) |
| `C-c e` | cape 접두 맵 (dabbrev/file/elisp 등 보완) |

### 창 관리
| 키 | 동작 |
|----|------|
| `M-o` | ace-window — 창 점프 |
| `C-c h` → `w` | hydra-window (`h/l/j/k` 이동, `s/v` 분할, `d` 삭제, `H/L/J/K` 크기) |

### 프로젝트 (projectile)
| 키 | 동작 |
|----|------|
| `C-c p` | projectile 커맨드 맵 |
| `C-c h` → `p` | hydra-projectile (`f` 파일, `s` ripgrep, `p` 전환, `r` 최근파일) |

### Git
| 키 | 동작 |
|----|------|
| `C-c h` → `g` | hydra-git (`s` status, `l` log, `b` blame, `d` diff) |
| 여백 표시 | diff-hl — 커밋되지 않은 변경을 fringe 에 표시 |

### 파일 탐색기 (treemacs)
| 키 | 동작 |
|----|------|
| `C-x t t` | treemacs 토글 · `M-0` treemacs 창으로 |
| `C-x t 1 / d / B / C-t / M-t` | 단일창 / 디렉터리 / 북마크 / 파일찾기 / 태그찾기 |

### 코드 폴딩 (`C-c z` 접두)
| 키 | 동작 |
|----|------|
| `C-c z a` | 토글 · `C-c z o/O` 열기/재귀 · `C-c z c` 닫기 · `C-c z r/m` 전부 열기/닫기 |

### 터미널 / 편집 / 도움말
| 키 | 동작 |
|----|------|
| `C-c t` | ghostel 터미널 (한글은 S-SPC 로 입력) |
| `C-z` / `C-S-z` | undo-fu undo / redo |
| `C-'` | avy — 화면 내 빠른 점프 |
| `C-h f/v/k` | helpful — 향상된 도움말 (describe-* 대체) |
| 저장 시 자동 | stripspace(끝공백 제거) · apheleia(포매팅) |

### macOS Cmd 키
| 키 | 동작 |
|----|------|
| `s-c / s-v / s-x` | 복사 / 붙여넣기 / 잘라내기 |
| `s-z / s-a` | 되돌리기 / 전체 선택 |

### 진입점 요약
- **`C-c h`** — 마스터 hydra (→ `w` 창, `p` 프로젝트, `g` Git, `z` 줌, `t` treemacs)
- **`C-c p`** — projectile, **`C-c z`** — 폴딩, **`C-c e`** — cape, **`C-c t`** — 터미널

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
| `16-languages` | git-modes · yaml · dockerfile · gnuplot · lua · jinja2 · csv · go · rust · crontab · nginx · hcl · nix · fish · vimrc · jenkinsfile · clojure · kotlin · typescript · web/tsx (+내장 sgml/java) | 21종 파일타입 모드 |
| `17-folding` | kirigami · outline-indent (+내장 outline/hs-minor) | 코드 폴딩 (`C-c z` 접두) |
| `18-terminal` | ghostel (+ghostel-ime) | libghostty-vt 터미널 (`C-c t`). 모듈은 vendor 동봉, S-SPC 한글 동작 |
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

### 터미널: ghostel (네이티브 모듈)

터미널은 [ghostel](https://github.com/dakra/ghostel)(libghostty-vt 기반)을 쓴다. vterm 보다 기능이 우수하고, 결정적으로 **`ghostel-ime-mode` 로 Emacs 한글 입력기(S-SPC)가 터미널 안에서도 동작**한다(vterm 은 불가).

**air-gap 동작**: ghostel 의 elisp 는 vendor 에, **네이티브 모듈은 사전빌드 바이너리를 `vendor/ghostel-module/` 에 동봉**(커밋)했다(aarch64-macos). 따라서 동일 arch(Apple Silicon macOS) 타겟은 **클론만 하면 빌드 없이 바로 동작**한다. `ghostel-module-auto-install` 은 `nil` 이라 부팅·사용 중 다운로드를 시도하지 않는다.

**모듈 갱신 / 다른 arch 대응** (온라인 머신에서):

```
M-x ghostel-download-module        # 현재 플랫폼 사전빌드 바이너리 다운로드
C-u M-x ghostel-download-module    # 특정 릴리스 태그 선택
M-x ghostel-module-compile         # Zig 로 소스 빌드(zig 0.15.2 필요)
```

받은 모듈은 `vendor/ghostel-module/` 에 저장되며, 그걸 커밋해 폐쇄망으로 반입한다. 타겟 arch 가 다르면(예: x86_64-linux) 해당 arch 바이너리를 같은 위치에 동봉하면 된다.

## 라이선스 / 글꼴 출처

이 저장소에 동봉된 글꼴(`assets/fonts/`)은 각 오픈소스 라이선스에 따라 재배포된다.

| 글꼴 | 출처 | 라이선스 |
|------|------|----------|
| 나눔고딕코딩 (NanumGothicCoding) | [naver/nanumfont](https://github.com/naver/nanumfont) © NHN Corporation | SIL Open Font License 1.1 |
| Symbols Nerd Font Mono (NFM.ttf) | [nerd-icons.el](https://github.com/rainstormstudio/nerd-icons.el) | MIT / OFL (각 심볼 세트별) |

- 나눔고딕코딩은 **SIL Open Font License (OFL) 1.1** 하에 배포되며, 라이선스를 명시하면 상용 소프트웨어 포함 재배포가 허용된다. 전문은 [`assets/fonts/OFL.txt`](assets/fonts/OFL.txt) 참조.
- OFL 조건에 따라 글꼴 원본과 라이선스 전문을 함께 동봉한다.
