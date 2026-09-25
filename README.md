# imoogi-emacs

개인 Emacs 설정. 모듈별로 분리하여 관리한다.

## 설치

Emacs 30.x 기준으로 vendoring 되어 있다. 저장소를 받은 뒤 설치 스크립트를 실행하면
`~/.config/imoogi-emacs` 심볼릭 링크와 `~/.emacs.d` 진입점이 만들어진다.

```bash
git clone <repo-url> ~/workspace/imoogi-emacs
cd ~/workspace/imoogi-emacs
./scripts/install.sh
```

Emacs 자체가 없거나 다른 버전을 나란히 두고 싶으면 `make emacs-install` 을 쓴다.
macOS 는 [emacsformacosx.com](https://emacsformacosx.com/) 의 universal `.dmg`,
Linux 는 배포판 패키지를 쓴다. `/Applications/Emacs.app` 을 덮어쓰지 않고
`/Applications/Emacs-<버전>.app` 으로 따로 설치하므로 기존 버전으로 언제든 돌아갈 수 있다.

```bash
make emacs-install                    # 기본 버전
make emacs-install EMACS_VERSION=30.2 # 버전 지정
```

스크립트는 반복 실행해도 안전하다. 기존 `~/.emacs.d/early-init.el` / `init.el` 이
있으면 덮어쓰기 전에 타임스탬프를 붙여 백업한다(`init.el.bak.20260822185215` 형태).
현재 OS와 아키텍처를 자동 감지해 호환되는 동봉 언어 서버도 함께 설치한다. 해당
플랫폼용 bundle이 없으면 Emacs 설치는 그대로 완료하고 LSP 설치만 건너뛴다.

설치기는 `~/.local/bin/imoogi-editor`도 연결하고 현재 셸의 시작 파일(zsh는
`~/.zshrc`, bash는 `~/.bashrc`)에 `VISUAL`과 `EDITOR`를 설정한다. 새 셸에서
Claude Code나 Codex의 `Ctrl+G`를 누르면 실행 중인 Emacs에 새 프레임이 열리며,
저장 후 `C-x #`로 편집을 끝내면 터미널로 돌아온다. 별도 위치의 `emacsclient`를
쓰려면 `EMACSCLIENT=/path/to/emacsclient`를 설정한다.

직접 하려면 `~/.config/imoogi-emacs` 로 심볼릭 링크를 건 뒤 아래 두 파일을 만든다.

`~/.emacs.d/early-init.el`:

```elisp
(load-file (expand-file-name "early-init.el" "~/.config/imoogi-emacs"))
```

`~/.emacs.d/init.el`:

```elisp
(load-file (expand-file-name "boot.el" "~/.config/imoogi-emacs"))
```

이후 Emacs 를 재시작하면 첫 부팅 때 동봉 폰트가 사용자 폰트 디렉터리로 복사되고,
다음 재시작부터 폰트가 적용된다.

언어 서버 설치를 원하지 않으면 `./scripts/install.sh --without-toolchain`을 사용한다.
나중에 별도로 설치하거나 다시 검증하려면 `make toolchain-setup`만 실행하면 된다.
자세한 무결성·rollback 절차는 [`docs/toolchains.md`](docs/toolchains.md)를 따른다.

모든 패키지가 저장소 안 `vendor/elpa/` 에 동봉돼 있어, **인터넷 없이도 첫 실행부터 그대로 동작한다**(망분리/air-gap 지원).

### (선택) tmux 기본 설정

Emacs 와는 별개로, 터미널을 tmux 로 쓴다면 저장소가 들고 있는 기본 설정을 얹을 수 있다.
**선택 사항이다** — 실행하지 않으면 tmux 는 아무 영향도 받지 않고, Emacs 설정은 그대로 동작한다.

```bash
make tmux-check      # 설정을 검사만 한다 (파일을 하나도 바꾸지 않는다)
make tmux-install    # ~/.tmux.conf 를 저장소 설정을 읽는 한 줄로 바꾼다
```

담고 있는 것은 플러그인 없이 tmux 본체 기능만 쓰는 기본기다.

| 갈래 | 내용 |
|---|---|
| 마우스 | 클릭으로 pane·window 이동, 경계선 끌어 크기 조절, 휠 스크롤(한 칸 2줄), 드래그하면 시스템 클립보드로 복사, 가운데 클릭으로 붙여넣기 |
| 스크롤 | 스크롤백 50,000줄 (기본 2,000줄) |
| 커서 | `cursor-colour` 로 커서 색을 밝게. 모양은 안쪽 프로그램이 정하도록 둔다 |
| 창 구분 | 활성 pane 만 밝은 파랑 굵은 경계선 + 화살표 표시, pane 위에 번호·명령·디렉터리를 보여주는 얇은 제목줄 |
| 상태 바 | Emacs 쪽 doom-one 과 같은 팔레트. prefix 를 눌러 대기 중이면 `PREFIX` 표시가 떠서 키가 먹었는지 헷갈리지 않는다 |
| Emacs 궁합 | `escape-time 10` (ESC 반응), `focus-events on` (창 전환 인지), `mode-keys emacs` (복사 모드 키 배치) |
| 키 | `prefix + r` 설정 다시 읽기, <code>prefix + &#124;</code> / `prefix + -` 로 방향이 보이는 나누기, `prefix + 방향키` pane 이동, `prefix + Shift+방향키` 크기 조절. 새 pane·window 는 현재 디렉터리를 물려받는다 |

prefix 는 기본값 `Ctrl-b` 그대로다. 이 설정을 처음 보는 사람이 아무 키도 못 누르는 상황과
문서·검색 결과와의 어긋남을 피하기 위해서다.

설치기는 **돌고 있는 tmux 서버를 건드리지 않는다.** 검사는 `-f /dev/null` 로 띄운
임시 서버에서 하고, 적용은 사용자가 직접 한다.

```bash
# 돌고 있는 tmux 안에서
prefix + r
# 또는 셸에서
tmux source-file ~/.tmux.conf
```

기존 `~/.tmux.conf` 가 있으면 타임스탬프를 붙여 백업한다(`~/.tmux.conf.bak.20260901201700` 형태).
되돌리기는 그 파일 하나를 제자리로 옮기면 끝이다.

개인 취향 설정은 `~/.tmux.local.conf` 에 둔다. 저장소 기본값을 모두 읽은 **뒤에** 마지막으로
읽히므로, 같은 옵션이면 이쪽이 이긴다. 예를 들어 prefix 를 바꾸고 싶다면:

```tmux
set -g prefix C-a
unbind C-b
bind C-a send-prefix
```

### 첫 부팅이 느린 경우 — 네이티브 컴파일

새 머신이나 **새 Emacs 버전**에서 처음 띄우면 몇 분간 멎은 것처럼 느려질 수 있다.
고장이 아니라 예정된 1회성 비용이다.

`21-native-compile.el` 의 compile-angel 은 로드되는 `.el` 을 전부 컴파일하는데,
바이트 컴파일 결과(`.elc`)는 저장소에 동봉하지만 네이티브 컴파일 결과(`.eln`)는
**머신·버전별 캐시라 동봉하지 않는다**. 그래서 처음 한 번은 직접 만들어야 한다.

- 캐시 위치: `~/.emacs.d/eln-cache/<버전키>/` — 버전마다 디렉터리가 따로 생기므로
  Emacs 를 새 버전으로 올리면 **처음부터 다시 만든다**
- 규모: 이 설정 기준 `.eln` 약 1,970개 / 200MB (Emacs 31.1 실측)
- 네트워크는 필요 없다

`make emacs-install` 은 설치 직후 이 예열을 자동으로 수행한다. 중간에 끊겼거나
이미 깔린 Emacs 를 나중에 예열하려면:

```bash
make emacs-prewarm                                     # 기본 Emacs
make emacs-prewarm EMACS=/Applications/Emacs-31.1.app/Contents/MacOS/Emacs
```

예열은 세 겹을 덮는다. 하나라도 빠지면 "설치는 됐는데 쓰다 보면 가끔 멎는다" 가 된다.

| 겹 | 대상 | 빠뜨리면 |
|---|---|---|
| 1 | `boot.el` 실제 로드 경로 | 부팅 자체가 느림 |
| 2 | `vendor/elpa` 전체 | GUI 전용 코드가 빠짐 — 배치 부팅만으로는 실측 107개에 그침 |
| 3 | Emacs 내장 lisp | **기능을 처음 쓸 때마다 몇 초씩 멎음** |

3번이 특히 눈에 띈다. 배포본에는 내장 `.eln` 이 일부만 들어 있어(31.1 실측 314개)
나머지는 그 기능을 처음 쓰는 순간 컴파일된다. 실제로 파일 열기를 한 번 했을 뿐인데
`url-handlers` `ewoc` `log-view` `kmacro` `thingatpt` 등이 한꺼번에 컴파일되며 멎었다.
`PREWARM_CORE=0` 으로 이 겹만 뺄 수 있다.

빈 캐시 기준 약 260초에 1,494개를 만든다. 이미 예열된 상태에서 재실행하면 15초 안에
끝난다(이미 `.eln` 이 있는 파일은 건너뛴다). 오래 걸릴 때는 `PREWARM_TIMEOUT`(기본
900초)을 늘린다 — 시간을 넘겨도 남은 분량은 다음 실행에서 이어서 만든다.

네이티브 컴파일이 없는 빌드에서는 예열이 필요 없으므로 자동으로 건너뛴다
(`native-comp-available-p` 로 판별). `PREWARM=0` 으로 전체를 끌 수도 있다.

### Emacs 버전을 올렸다면 — `.elc` 를 먼저 지운다

```bash
make clean-elc
```

**버전을 바꿨을 때 가장 먼저 할 일이다.** `modules/**/*.elc` 는 `.gitignore` 대상이라
`git status` 에 보이지 않고, 소스를 고치지 않는 한 재생성되지 않는다. 그래서 낡은
바이트코드가 조용히 남아 새 Emacs 에서만 이상하게 동작할 수 있다.

실제 사례: Emacs 31.1 을 나란히 깔았더니 `17-lsp` 와 `22-tabs` 두 모듈이
`Symbol's value as variable is void: imoogi-transient-lsp` 로 로드에 실패했다.
소스에는 문제가 없었고 — `.elc` 를 지우고 다시 만드니 그대로 사라졌다. 함께
따라오던 ERT 실패 7건(테스트 수가 79 → 88 로 늘어난 것 포함)도 같은 원인이었다.

`imoogi-require` 의 degradation 설계 덕에 부팅이 죽지는 않고 해당 모듈만 조용히
빠지므로, **증상이 "LSP 가 안 켜진다" 처럼 엉뚱하게 나타난다.** 버전을 올린 뒤
무언가 이상하면 `make clean-elc` 를 먼저 해보는 것이 가장 빠르다.

`make clean-elc` 는 지우기 전에 어느 Emacs 가 컴파일했는지 먼저 보여주므로,
섞여 있는지 확인만 하고 싶을 때도 쓸 수 있다.

> 반대 방향이 더 위험하다: 새 Emacs 로 잠깐 부팅해 `.elc` 가 새 버전으로
> 다시 컴파일되면, 평소 쓰는 옛 버전이 그 바이트코드를 읽게 된다.
> `load-prefer-newer` 는 타임스탬프만 보고 컴파일한 버전은 보지 않는다.

`vendor/elpa/**/*.elc` 는 커밋 대상(망분리 반입용)이라 `clean-elc` 가 건드리지
않는다. 그쪽까지 새 버전으로 맞추려면 온라인 머신에서 vendoring 을 다시 돌린다.

## 테스트

```bash
EMACS=/path/to/emacs ./tests/run.sh   # macOS 호스트: check-parens + 오프라인 부팅 + ERT
./tests/docker/run.sh                 # Linux 컨테이너: 설치 + 부팅 매트릭스
```

`tests/docker/run.sh` 는 성격이 다른 두 층을 돌린다. 합격 기준이 서로 반대라 곱하지
않고 나눠 실행한다.

| 층 | 환경 | 합격 기준 |
|---|---|---|
| T2 이식성 | `debian:trixie-slim`(30.1) · `ubuntu:26.04`(30.2) · `alpine:edge`(30.2, musl) — 도구 전부 설치 | 스킵된 모듈 **0개** |
| T1 degradation | `debian:trixie-slim` — `git` 없음 | 선언한 모듈(`06-git`, `07-treemacs`)만 **정확히** 스킵 |

판정은 `tests/assert-boot.el` 이 단독으로 담당하며, Docker 와 호스트가 같은 파일을
쓴다. **Emacs 종료 코드는 판정 근거가 되지 않는다** — `boot.el` 이 모듈 실패를
격리하므로 절반이 깨져도 `exit 0` 이 나온다(측정: `git` 부재 시 모듈 2개 스킵,
Emacs 29.3 + 30.x `.elc` 조합에서 모듈 3개 스킵, 둘 다 `exit 0`). 그래서 실제로
어떤 모듈이 로드됐는지를 `imoogi-failed-modules` 로 확인한다.

`debian:bookworm`(28.2), `ubuntu:24.04`(29.3) 는 vendored `.elc` 와 메이저 버전이
달라 매트릭스에서 제외했다. `vendor/ghostel-module`(Mach-O arm64)과
`vendor/toolchains`(darwin-arm64)는 Linux 에서 실행 자체가 불가능해 컨테이너 범위
밖이며, macOS 호스트에서만 검증한다.

## 망분리(air-gap) 환경

이 설정은 폐쇄망에서 동작하도록 설계됐다. 저장소 하나만 클론해 들고 들어가면 외부 네트워크 없이 작동한다 — 부팅 경로에서 네트워크에 접근하지 않는다.

- 패키지는 `vendor/elpa/` 에 동봉(커밋)되며, 런타임에 `package-refresh-contents` 나 다운로드를 하지 않는다.
- `vendor-manifest.json`과 `provenance/*.json`이 모든 동봉 외부 파일의 upstream,
  고정 버전/commit, 플랫폼, 크기와 SHA-256을 기록한다. `packages.lock`과
  `toolchains.lock.json`도 이 기록과 일치해야 한다.
- 빌드 머신과 타겟의 **Emacs 메이저 버전을 일치**시킬 것(.elc 호환).

### 패키지 추가/업데이트 (온라인 빌드 머신에서만)

```bash
# 1. packages.el 의 imoogi-required-packages 수정 (추가/삭제 시)
# 2. vendoring 재실행
emacs --batch -Q -l scripts/vendor.el            # 누락분만 설치
emacs --batch -Q -l scripts/vendor.el -- upgrade # 전체 최신으로 갱신
# 3. 매니페스트와 전체 테스트 검증
make verify-vendor
make test
# 4. 변경 커밋
git add vendor/ provenance/ vendor-manifest.json packages.lock packages.el
# 5. 폐쇄망으로 반입 (내부 git 미러 pull 또는 저장소 재반입)
```

폐쇄망 안에서는 절대 vendoring 을 돌리지 않는다(네트워크 필요). 업데이트는 항상 온라인 머신 → 반입 순서다.

`scripts/vendor.el`과 `scripts/build-grammars.sh`는 완료 시 provenance manifest를
자동 재생성한다. 그 밖의 외부 바이너리나 글꼴을 갱신했다면 온라인 머신에서
`make provenance-generate`를 실행한다. 새 파일이 manifest에 없거나, 파일이
사라졌거나, 크기/SHA-256/고정 commit/기존 lock이 어긋나면
`make verify-vendor`와 이를 선행하는 `make test`가 실패한다. 예외는
`provenance/sources.json`에 wildcard 없이 정확한 경로와 이유를 기록해야 한다.

### Tree-sitter 문법

Emacs 내장 `treesit` 은 사용하되, 언어별 문법은 런타임에 다운로드하지 않는다
(`treesit-install-language-grammar` 는 네트워크를 쓰므로 폐쇄망 부적합).
온라인 머신에서 빌드해 `vendor/tree-sitter/` 에 커밋하면, `18-languages` 가
문법이 있는 언어만 `*-ts-mode` 로 자동 전환한다. 없으면 기존 전통 major-mode 로
그대로 열린다.

```bash
./scripts/build-grammars.sh          # 전체 (온라인 머신에서만)
./scripts/build-grammars.sh go json  # 일부만
git add vendor/tree-sitter && git commit -m "vendor: update tree-sitter grammars"
```

현재 반입된 문법: `json` `javascript` `typescript` `tsx` `python` `go` `java` `yaml`
`kotlin` `clojure` `regex` `markdown-inline`.

Kotlin 과 Clojure 는 Emacs 에 ts-mode 가 내장돼 있지 않아 패키지(`kotlin-ts-mode`,
`clojure-ts-mode`)와 문법이 **둘 다** 있어야 동작한다. `regex` 와
`markdown-inline` 은 `clojure-ts-mode` 가 함께 요구하는 문법이다 — 없으면 그
모드가 부팅 중 인터넷에서 내려받으려 하므로(기본 `clojure-ts-ensure-grammars` 가
`t`) `18-languages` 가 그 동작을 끄고 문법을 동봉으로 대신한다. 리비전이
어긋나도 같은 다운로드가 일어나므로, 스크립트의 태그는 패키지가 못박은 값과
정확히 일치해야 한다.
결과물은 빌드한 플랫폼 전용이다(macOS `.dylib`, Linux `.so`) — 다른 플랫폼으로
반입하려면 그 플랫폼에서 다시 빌드한다.

문법이 들어오면 딸려오는 이득이 있다. `json-ts-mode` / `java-ts-mode` /
`go-ts-mode` / `typescript-ts-mode` 는 자체 imenu 설정을 갖고 있어 `M-g i` 의
구조 탐색이 정확해지고, `C-c h c` 의 L3(구문 트리)이 켜진다.

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

`M-x` 등 Vertico 완성 목록은 프레임 높이의 약 25%를 사용한다.
`imoogi-completion-height-fraction`으로 비율을 바꾸거나 `nil`로 기본 동작을 사용할 수 있다.
완성 목록의 글자는 실행한 버퍼의 확대 배율을 따르며 최소 한 단계 확대한다.
최소 배율은 `imoogi-completion-minimum-text-scale`로 조절한다.

| 키 | 동작 |
|----|------|
| `M-x` | 명령 실행 (vertico 세로 완성) |
| `C-x C-f` | 파일 열기 |
| `C-s` / `M-s l` | consult-line — 현재 버퍼 검색 |
| `C-x b` | 현재 Perspective 버퍼 전환 · `C-u C-x b` 전체 버퍼 |
| `M-s r` / `M-s g` | consult-ripgrep / grep — 프로젝트·디렉터리 검색 |
| `M-g g` | 줄 이동 · `M-g i` imenu · `M-g f` flymake 진단 |

`C-c h z v`는 현재 버퍼의 `visual-line-mode`를 토글한다. 메뉴에는 문단
줄바꿈의 현재 상태가 `켜짐`/`꺼짐`으로 표시되며, 메뉴를 유지한 채 바꿀 수 있다.
| `M-y` | consult-yank-pop (kill-ring) |
| `C-.` / `C-;` | embark-act / embark-dwim — 후보·심볼 컨텍스트 액션 |

### 버퍼 내 자동완성 (corfu / cape)

프로그래밍·셸 버퍼에서는 입력 후 0.15초가 지나면 Corfu 후보 창이 자동으로 열린다.
Eglot이 연결된 언어는 LSP 후보를 사용하고, Cape가 단어·파일 후보를 보완한다.

| 키 | 동작 |
|----|------|
| `TAB` | 자동 팝업과 별개로 들여쓰기 또는 완성을 수동 실행 (`tab-always-indent`) |
| `C-c e` | cape 접두 맵 (dabbrev/file/elisp 등 보완) |

### LSP / 코드 탐색 (`eglot` + `xref`)
| 키 | 동작 |
|----|------|
| `M-.` / `C-c l d` | 정의로 이동 |
| `M-?` / `C-c l r` | 참조 찾기 |
| `M-,` / `C-c l b` | 이전 xref 위치로 돌아가기 |
| `C-M-,` / `C-c l f` | 다음 xref 위치로 이동 |
| `C-c l R` | 심볼 이름 변경 · `C-c l c/o` 코드 액션/임포트 정리 |
| `C-c l e/q` | Eglot 수동 연결/종료 |

언어별 자동 연결은 `modules/development/lang/` 아래에서 독립적으로 관리한다. 현재 Bash,
JavaScript/TypeScript, Go, Python, Rust, Clojure, Java, Kotlin 설정이 있으며,
새 언어는 같은 폴더에 설정 파일 하나를 추가하면 `17-lsp`가 자동으로 로드한다.
Go/TypeScript 서버는 저장소의 `vendor/toolchains/`에 고정된 artifact를 사용한다.
온라인 머신에서는 동봉 CLI의 `fetch`로
`toolchains.lock.json`과 artifact를 갱신하고, 폐쇄망 타겟에서는 같은 bootstrap의
`setup`만 실행한다. 일반 설치는 OS·아키텍처와 CLI 버전을 자동으로 찾아 실행하며,
수동 재설치는 `make toolchain-setup`을 쓴다. `setup`은 `.local/toolchains/<bundle>/`에 검증 후 설치하고
상대 symlink `.local/bin`을 활성화한다. `17-lsp`는 부팅 중 설치나 다운로드 없이
활성 `.local/bin`이 있을 때만 `exec-path`와 `PATH` 앞에 둔다. 서버가 없어도
메이저 모드와 xref fallback은 그대로 동작한다.

CLI는 SemVer(`1.0.0`), 설치 bundle은 CalVer(`2026.08.22.1`)로 독립 관리한다.
현재 고정 버전, SHA-256, 업데이트와 rollback 절차는
[`docs/toolchains.md`](docs/toolchains.md)를 따른다.

### 코드 이해 (`C-c h` → `c`)

깊이 5단계를 한 메뉴에 모았다. 갖춰지지 않은 단계는 **숨기지 않고 흐리게** 표시해,
지금 이 버퍼에서 어디까지 파고들 수 있는지 한눈에 보이게 한다.

| 단계 | 키 | 필요한 것 |
|------|-----|-----------|
| L1 빠른 탐색 | `i` 이 파일 심볼 · `I` 프로젝트 심볼 | 없음 (imenu) |
| L2 상주 패널 | `s` 구조 패널 · `f` 파일 트리 | treemacs |
| L3 구문 | `t` 구문 트리 | 해당 언어의 tree-sitter 문법 |
| L4 관계 | `d` 정의 · `r` 참조 · `m` 구현 · `y` 타입 정의 | `d`/`r` 없음, `m`/`y` 는 Eglot 연결 |
| L5 전체 | `g` 전체 그래프 | 미구현 (graphviz + analyzer 필요) |

`?` 를 누르면 **무엇이 왜 안 되는지** 이유와 함께 보고한다. 각 단계는 서로
독립적으로 켜진다 — 문법만 반입하면 L3만, Eglot 만 붙이면 L4만 밝아진다.

L1 은 `M-g i`/`M-g I`, L4 의 `d`/`r` 은 `M-.`/`M-?` 로도 쓸 수 있다(Emacs 표준).

### 창 관리
| 키 | 동작 |
|----|------|
| `M-o` | ace-window — 창 점프 |
| `C-c h` → `w` | hydra-window (`h/l/j/k` 이동, `s/v` 분할, `d` 삭제, `H/L/J/K` 크기) |

### 프로젝트와 작업공간 (`project.el` + `perspective.el`)

`C-x p p`로 프로젝트를 열면 프로젝트 폴더와 같은 이름의 Perspective로
전환하고, 왼쪽에는 `Project: <Perspective 이름>` Treemacs workspace를 연다.
처음에는 선택한 프로젝트 폴더 하나만 들어간다. `C-x t a`로 임의 폴더를,
`C-x t p`로 현재 버퍼의 프로젝트를 나중에 추가할 수 있다. 추가한 폴더는 해당
workspace에 저장되어 다음 전환에도 유지된다. `C-u C-x p p`는
현재 Perspective를 유지하는 명시적 다중 프로젝트 동작이므로 Treemacs
workspace도 자동으로 바꾸지 않는다.

같은 기능은 `C-c h t` Treemacs 메뉴에서도 사용할 수 있다. 메뉴에서 `f`는
파일 트리 토글, `a`는 임의 폴더 추가, `p`는 현재 프로젝트 추가, `w`는
Treemacs workspace 전환이다.

`C-c h m`은 현재 버퍼의 mode 안내 메뉴다. `m`으로 활성 minor mode 하나를
선택하면 설명과 그 mode의 전용 키 바인딩을 보고, `l`은 활성 minor mode 목록,
`b`는 현재 버퍼에서 실제로 유효한 전체 키 바인딩을 표시한다. 기본 동작으로
`w` visual-line 토글, `f` outline 폴딩 토글, `y` yasnippet 펼치기,
`d` Flymake 진단, `v` Org/Markdown 브라우저 미리보기를 제공한다. 현재 버퍼에서
해당 minor mode나 major mode가 활성화되지 않은 동작은 메뉴에서 비활성화된다.

| 키 | 동작 |
|----|------|
| `C-x p p` | 프로젝트 선택 → 대응 Perspective 전환/생성 → 프로젝트 Dired |
| `C-u C-x p p` | 현재 Perspective를 유지하고 다른 프로젝트를 함께 열기 |
| `C-x x s / c / l` | Perspective 생성·전환 / 종료 / 직전 작업공간 복귀 |
| `C-x x a / g` | 버퍼를 현재 Perspective에 추가 / 여러 Perspective용 전역 공유 |
| `C-c h` → `p` | 프로젝트·작업공간 메뉴 (아래) |

`C-c h` → `p` 는 세 갈래로 나뉜다.

imoogi Transient 메뉴에서 `?`를 누르면 현재 메뉴의 목적과 기능 목록을 우측
도움말 창에 표시한다. 다시 `?`를 누르면 닫히며, `C-h`는 기존 Transient의
항목별 도움말 동작을 유지한다.

| 갈래 | 키 |
|------|-----|
| 열기 | `p` 프로젝트 전환 · `f` 파일찾기 · `s` 검색(grep) · `d` dired · `b` 버퍼 |
| 작업공간 전환 | `l` 직전 작업공간 · `o` 목록에서 선택 · `n`/`N` 다음/이전(반복 가능) · `#` 번호로 |
| 관리 | `c` 컴파일 · `k` 버퍼모두닫기 · `r` 이름변경 · `K` 작업공간 닫기 · `F` 목록에서 제거 |

같은 명령을 메뉴 없이 바로 쓰려면 `C-x x` 접두를 쓴다(`l` 직전, `s` 선택,
`n`/`p` 다음/이전, `` ` `` 번호로, `r` 이름변경, `c` 닫기).

프로젝트 루트와 기본 Perspective 이름의 연결은 `savehist`로 유지된다.
Perspective의 파일/Dired 버퍼와 창 배치는 정상 종료 시 저장되며, shell·REPL·compile
프로세스는 현재 세션의 작업 컨텍스트에는 포함되지만 재시작 시 다시 생성되지는 않는다.

### 프로젝트 기록 (`project-notes`)

개인 메모는 `~/notes/`, 프로젝트 기록은 소스와 분리된
`~/project-notes/` 아래에 둔다. 프로젝트에서 `C-c h p m s`
(`M-x imoogi-project-notes-setup`)를 실행하면 프로젝트 번호를 입력받아
기본 문서와 자료 폴더를 만든다. 예를 들어 `260925.01`은 2026년 9월 25일
첫 번째 일일 목표이고, PC에서는 `260925.01-프로젝트명`처럼 표시된다.
기존 파일은 덮어쓰지 않는다. `C-u M-x imoogi-project-notes-setup`으로
프로젝트별 저장 폴더를 직접 지정할 수 있다.

메뉴와 프로젝트 목록에서 선택하는 **프로젝트는 소스 작업 폴더**다. 예를 들어
`~/workspace/imoogi-emacs/`를 선택하면 이 작업 폴더를 별도의 번호가 붙은
문서 폴더와 연결한다. 월간 프로젝트는 `2601.1`, 연간 프로젝트는 `26.01`,
일간 목표는 `260925.01`처럼 번호를 입력한다. PC용 폴더명은 여기에 프로젝트명을
붙인 형태를 사용한다.
현재 연결 관계와 worktree 동작은 `C-c h p m h`의 내장 안내에서 확인할 수 있다.

```text
~/notes/
  permanent/                 org-roam 영구 노트
  agenda.org                 개인 할 일 / 중앙 관리 선택 시 프로젝트 할 일
  scratch.org                분류 전 메모 (Scratch 명령에서 생성)
~/project-notes/<번호>-<프로젝트명>/
  project.org                목적·범위·현재 상황·큰 작업 요약
  tasks.org                  실제 TODO와 완료 조건
  journal.org                작업 기록·worktree별 재개 지점
  assets/                    이미지 등 첨부 자료
  references/                참고 자료
  artifacts/                 TODO에서 파생된 조사·설계·검증 산출물
  development/               필요한 문서만 명령으로 생성
    domain.org               용어·개념·관계·업무 규칙
    architecture.org         구성 요소·책임·데이터 흐름
    decisions.org            결정의 배경·대안·이유·영향
```

`C-c h p m` 메뉴:

| 키 | 명령 | 용도 |
|----|------|------|
| `s` | `imoogi-project-notes-setup` | 현재 소스 작업 폴더에 별도 문서 폴더 연결·생성 |
| `S` | `imoogi-project-notes-setup-study` | 소스 폴더 없이 학습 노트와 전용 작업공간 생성 |
| `o` | `imoogi-project-notes-open` | 프로젝트 개요 |
| `t` | `imoogi-project-notes-tasks` | 할 일 원본 열기 |
| `j` | `imoogi-project-notes-journal` | 현재 작업 공간의 재개 지점 |
| `l` | `imoogi-project-notes-list` | 등록된 프로젝트·학습 노트를 골라 작업공간·개요·할 일·기록으로 이동 |
| `a` | `imoogi-project-notes-agenda-current` | 현재 프로젝트 Focus Agenda |
| `A` | `imoogi-project-notes-agenda-all` | 등록된 전체 프로젝트 Dashboard |
| `r` | `imoogi-project-notes-create-artifact` | 현재 TODO의 산출물 파일과 양방향 ID 링크 생성 |
| `d` | `imoogi-project-notes-add-document` | 도메인·구조·결정 문서 추가 |
| `n` | `imoogi-notes-scratch` | `~/notes/scratch.org` 열기 |
| `h` | `imoogi-project-notes-setup-guide` | 작업 폴더와 문서 폴더의 차이 안내 |
| `+` | `imoogi-project-notes-mounted-root-add` | SD 카드·SSD의 imoogi 노트 상위 폴더 등록 |
| `L` | `imoogi-project-notes-mounted-root-list` | 등록한 외장 루트와 발견된 노트 목록 표시 |
| `E` / `R` / `D` | 루트 편집 / 다시 검색 / 등록 제거 | 호스트의 외장 루트 설정 관리 |
| `v` | `imoogi-project-notes-move-to-mounted-root` | 기존 로컬 프로젝트·학습 노트를 등록된 외장 루트로 이동 |
| `x` | `imoogi-project-notes-detach` | 선택한 외장 노트를 현재 세션 목록에서만 분리 |
| `u` | `imoogi-project-notes-unmount-device` | 변경 버퍼를 확인·저장한 뒤 장치 unmount |
| `c` / `C` | 소스 재연결 / 연결 해제 | 외장 project note의 호스트별 소스 경로 관리 |
| `e` | `imoogi-project-notes-force-edit-session` | 비활성 노트의 현재 버퍼만 이번 세션에 편집 |

기존 폴더명이 번호 규약에 맞지 않으면 `M-x imoogi-project-notes-setup-doctor`를
실행한다. Doctor는 잘못된 폴더를 하나씩 보여주고 새 번호 또는 PC용 폴더명을
입력받는다. 입력한 폴더명은 `project-notes` 레지스트리, 열린 버퍼, Perspective,
Treemacs workspace 경로에 함께 반영한다. 빈 입력은 해당 폴더를 건너뛰며, 파일
내용은 삭제하거나 덮어쓰지 않는다.

영속 Scratch는 메인 메뉴에서 **`C-c h n`**으로 바로 열 수도 있다.
Scratch와 작업 기록은 일반 파일 버퍼이므로 `C-x C-s`로 저장한다.
개요에는 진행 요약과 링크를, `tasks.org`에는 실행할 작업을,
`journal.org`에는 멈춘 지점과 다음 행동을 적는다.
작업 상태는 `TODO → NEXT → DOING → DONE`, 대기는 `WAIT`, 취소는
`CANCELLED`로 표현한다. 모든 상태를 순서대로 거칠 필요는 없다.
프로젝트 작업 파일의 `CATEGORY`는 프로젝트 이름으로 초기화하여 Agenda에서 구분한다.
개발 문서는 `d` 명령을 쓸 때 생성하며, 처음에는 기본 Org 문서 세 개만 만든다.

TODO heading에서 `r`을 누르면 조사·요구사항·설계·문제 분석·결정·회의·검증 결과·
작업 절차·빈 문서 중 하나를 선택한다. 새 파일은 `artifacts/`에 만들어지고 TODO의
`산출물:` 목록과 새 문서의 `관련 작업`이 `org-id`로 서로 연결된다. `tasks.org`에는
상태·완료 조건·일정·산출물 링크만 두고 긴 분석과 결과는 산출물 파일에 기록한다.
같은 이름의 파일이 있으면 번호를 붙이며 기존 산출물은 덮어쓰지 않는다.

같은 Git 저장소의 worktree는 공통 Git 디렉터리를 기준으로 같은 기록 폴더를
공유한다. 재개 지점은 각 worktree 경로별로 나뉜다. 별도 clone은 별도 프로젝트로
취급한다. 등록 정보는 Emacs 사용자 디렉터리의 `.cache/project-notes.json`에
보관한다. 부팅만으로 기록 폴더나 템플릿을 생성하지 않는다.

`imoogi-project-notes-directory`로 기본 상위 폴더를 바꿀 수 있다.
`imoogi-project-notes-todo-storage`는 새로 등록하는 프로젝트의 TODO 위치를 정한다:

| 값 | 장점 | 고려할 점 |
|----|------|-----------|
| `project` (기본) | 프로젝트 자료와 작업을 함께 보관 | 파일이 여러 개지만 Agenda에서 합쳐 조회 |
| `central` | `~/notes/agenda.org` 한 곳에서 작업 관리 | 프로젝트 자료와 TODO의 저장 위치가 분리 |

중앙 관리에서는 `tasks.org`가 중앙 Agenda 문서로 안내한다. 프로젝트별 선택은
등록 정보에 저장되며, 설정 변경만으로 기존 TODO를 이동하거나 복제하지 않는다.
프로젝트별 작업 파일은 Org Agenda에 등록하여 개인 일정과 함께 조회한다.

#### SD 카드·SSD의 프로젝트·학습 노트

`C-c h p m +`로 외장 저장장치 안의 노트 상위 폴더를 등록한다. 등록 정보와
호스트별 소스 경로 연결은 Emacs 사용자 디렉터리의
`.cache/project-notes-mounted-roots.json`에만 저장된다. 외장 장치의 각 노트는
자신의 `.imoogi-project.json`으로 발견되며 기존 `.cache/project-notes.json`에
복사하거나 import하지 않는다. 따라서 같은 장치를 다른 PC에 연결해도 노트 폴더
자체의 metadata를 기준으로 목록을 다시 만들 수 있다.

등록한 외장 루트에 새 학습 노트를 만들 때는
`C-u M-x imoogi-project-notes-setup-study`를 실행한다. 학습 이름 다음에 외장
루트를 선택하면 그 루트의 로컬·외장 학습 ID를 모두 확인해 다음 `YY.NN` 폴더를
제안한다. 폴더 선택을 그대로 확정하면 외장 루트 아래에 학습 노트가 만들어진다.

이미 로컬에 만든 project/study note는 `C-c h p m v`로 외장 루트로 옮긴다.
먼저 `make build-notes`로 `bin/imoogi-notes`를 빌드한다. 이때 실행할 명령은
`imoogi-project-notes-command` defcustom으로 정하며 기본값은 `("imoogi-notes")`다.
Emacs는 이 이름을 PATH에서 먼저 찾고, 없으면 저장소 안의 `bin/imoogi-notes`를 쓴다.
노트와 대상 외장 루트를 선택하면 Emacs는 이 Go 프로그램을 호출한다. Go 프로그램이 같은 폴더 이름으로
복사하고 모든 파일의 SHA-256 검증과 원본 전환을 끝낸 뒤 성공 결과를 반환하면,
Emacs가 로컬 등록·Agenda 경로·열린 버퍼 경로를 새 위치로 바꾼다.
대상에 같은 이름이 있으면 덮어쓰지 않고 중단한다.
중앙 `agenda.org`를 사용하는 프로젝트는 노트 폴더 밖에 TODO 원본이 있으므로 이동
대상에서 제외한다. 외장 이동은 프로젝트 자체의 `tasks.org`를 쓰는 항목에 지원된다.

목록에는 장치 표시 이름이 함께 나오며 study note는 노트 폴더 자체를 작업공간으로
열 수 있다. project note의 원래 소스 폴더가 현재 PC에 없으면 `[비활성]`으로
표시되고 문서는 읽기 전용으로 열린다. `c`로 이 PC의 소스 폴더를 재연결할 수 있고,
`C`로 그 연결을 지울 수 있다. 연결은 외장 파일이 아니라 호스트 설정에 저장된다.
`e`는 현재 버퍼만 이번 Emacs 세션 동안 편집하게 하며 다른 문서·산출물 생성은
계속 차단한다.

`x`는 노트 하나를 현재 세션 목록에서 숨기는 논리적 분리다. 장치를 운영체제에서
내리지는 않는다. `u`는 등록 루트가 들어 있는 실제 mount point를 먼저 확인한 뒤,
그 장치 아래의 변경 버퍼 목록을 보여준다. 사용자가 진행을 승인하고 모든 저장이
성공한 경우에만 해당 버퍼를 닫고 unmount 명령을 실행한다. mount 식별 실패,
지원되지 않는 플랫폼, 저장 실패 또는 사용자 취소에서는 버퍼와 등록 상태를
변경하지 않는다. macOS는 `diskutil`, Linux는 block device와 `udisksctl`을 확인한
경우만 물리 unmount를 지원한다. 그 밖의 환경에서는 `x`로 논리적 분리를 사용한다.

외장 루트는 로컬 경로여야 하며 canonical path가 같거나 서로 상위·하위로 겹치는
루트는 중복 등록할 수 없다. 백그라운드 mount 감시는 하지 않으므로 장치를 다시
연결한 뒤에는 `R`로 검색을 갱신한다.

#### 소스 프로젝트가 없는 학습 노트

강의·책·자격시험처럼 소스 코드 폴더가 없는 학습도 project-notes 안에서 관리한다.
`C-c h p m S` 또는 `M-x imoogi-project-notes-setup-study`를 실행하고 학습 이름을
입력하면 `~/project-notes/YY.NN-학습명/`을 만든다. 같은 해의 기존 번호를 확인해
`26.01`, `26.02`처럼 두 자리 순번을 자동으로 증가시킨다.

생성 직후 노트 폴더 자체를 루트로 하는 Perspective와 Treemacs workspace가 열리고
`study.org`가 표시된다. 별도의 소스 프로젝트를 선택하거나 등록할 필요가 없다.
`tasks.org`는 기존 프로젝트 작업과 마찬가지로 Org Agenda에 등록한다. 기존 파일은
덮어쓰지 않는다.

모든 프로젝트 노트 폴더의 최상단에는 `.imoogi-project.json`이 생성된다. 이 파일의
`type`이 `project`인지 `study`인지에 따라 폴더 규약과 사용할 명령을 판별한다.
현재 파일에서 상위 폴더를 탐색하므로 중앙 레지스트리가 없어도 학습 노트의 문맥을
복원할 수 있다. `.cache/project-notes.json`은 전체 목록과 빠른 선택을 위한 인덱스로
계속 사용한다. 기존 메타데이터는 자동으로 덮어쓰지 않는다.

```json
{
  "schema_version": 1,
  "type": "study",
  "key": "study:26.01",
  "name": "Operating Systems",
  "study_id": "26.01",
  "created_at": "2026-01-15",
  "overview": "study.org",
  "tasks": "tasks.org",
  "journal": "logs/journal.org",
  "todo_storage": "project"
}
```

```text
~/project-notes/26.01-operating-systems/
  .imoogi-project.json       폴더 유형·ID·문서 배치 메타데이터
  study.org                 학습 목적·범위·현재 진행 상황
  tasks.org                 읽기·과제·시험 TODO
  cards.org                 인출 문항과 암기카드 후보
  questions.org             사전 회상과 미해결 질문
  materials/
    books/ handouts/ articles/ slides/ videos/
  logs/journal.org          학습 과정 기록
  concepts/                 자기 언어로 정리한 개념
  assignments/              문제 풀이와 적용 결과
  assets/                   캡처·그림·첨부 이미지
```

`study.org`에는 아날로그 노트와 함께 쓸 수 있는 짧은 ID와 정확한 시작일을 기록한다.

```org
#+TITLE: Operating Systems
#+STUDY_ID: 26.01
#+START_DATE: 2026-01-15
```

### Git
| 키 | 동작 |
|----|------|
| `C-c h` → `g` | hydra-git (`s` status, `l` log, `b` blame, `d` diff) |
| 여백 표시 | diff-hl — 커밋되지 않은 변경을 fringe 에 표시 |

### 파일 탐색기 (treemacs)
| 키 | 동작 |
|----|------|
| `s-1` | Project: 편집 버퍼에서는 파일 트리로 이동, Treemacs 안에서는 닫기 |
| `s-2` | Bookmarks: Emacs 북마크 목록 토글 |
| `s-7` | Structure: 현재 버퍼의 Imenu 구조 토글 |
| `C-x t t` | treemacs 토글 · `M-0` treemacs 창으로 |
| `C-x t p` | 현재 `project.el` 프로젝트를 Treemacs에 추가하고 표시 |
| `RET` / `o o` | 선택한 파일을 추가 분할 없이 편집 창에 열기 |
| `o a a` / `o r` | 열 창 직접 선택 / 최근 사용 편집 창에 열기 |
| `o v` / `o h` | 세로 / 가로 분할을 만들고 열기 |
| `o c` | 파일을 열고 Treemacs 닫기 |
| `C-x t 1 / d / B / C-t / M-t` | 단일창 / 디렉터리 / 북마크 / 파일찾기 / 태그찾기 |

`s-1`, `s-2`, `s-7`은 IntelliJ의 도구 창처럼 같은 왼쪽 슬롯을 공유한다.
Treemacs가 보이는 상태에서 편집 버퍼의 `s-1`은 파일 트리로 이동하고,
Treemacs 안의 `s-1`은 파일 트리를 닫는다. `ESC`는 Treemacs를 유지한 채
마지막 편집 창으로 돌아간다. 다른 도구 창 키를 누르면 해당 보기로 교체된다.
Bookmarks와 Structure 안에서는 `g`로 새로고침하고 `q`로 닫는다.
기본 `RET`은 기존 편집 창이 있으면 그 창을 사용하며, Treemacs만 남아 있을
때에만 파일을 표시할 편집 창을 옆에 만든다.

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
| `s-w` | 현재 버퍼 닫기; 수정된 버퍼는 저장 여부 확인 |
| `s-z / s-a` | 되돌리기 / 전체 선택 |

### 진입점 요약
- **`C-c h`** — 마스터 hydra (→ `w` 창, `p` 프로젝트, `g` Git, `z` 줌, `t` treemacs)
- **`C-x p`** — project.el, **`C-x x`** — Perspective, **`C-c l`** — LSP/xref, **`C-c z`** — 폴딩, **`C-c e`** — cape, **`C-c t`** — 터미널

Transient 메뉴에서는 Emacs 내장 입력기를 잠시 꺼 영문 단축키를 사용한다.
하위 메뉴에서도 유지하고, 메뉴 종료 시 기존 입력기 상태를 복원한다.
OS 키보드 입력 언어는 변경하지 않는다.

### Org 기본 폴더 설정

`M-x imoogi-org-setup`을 실행하면 홈 디렉터리에 `~/notes/`와 영구 노트용
`~/notes/permanent/`를 만들고 Org 기본 폴더(`org-directory`)로 설정한다.
기존 폴더와 파일은 보존하며 여러 번 실행해도 된다.
재시작 후에도 기본 Org 폴더는 `~/notes/`다. 폴더 생성은 이 명령을 실행할 때만 수행한다.

기존 설정이 이 구조와 다르면 `M-x imoogi-org-setup-doctor`를 실행한다.
notes 바로 아래의 기존 `.org` 파일을
`permanent/`로 복사하고 `agenda.org`와 `scratch.org`는 그대로 둔다.
원본은 삭제하지 않으며, 같은 이름의 대상 파일도 덮어쓰지 않는다.

Org-roam은 `~/notes/permanent/`를 영구 노트 폴더로 사용한다. `C-c n` 또는
`C-c h o r`로 중앙 노트 Transient를 연다. `f`는 노트 찾기·만들기,
`n`은 캡처, `i`는 링크 삽입, `r`은 현재 제목 또는 선택 영역을 다른
노트로 옮기기(`org-roam-refile`), `b`는 백링크 보기다. 별칭(`a/A`),
태그(`t/T`), 참조(`e/E`)를 관리하고, `d/D`로 일일 노트를 연다.
`R`은 무작위 노트, `g`는 Graphviz `dot`이 있을 때 그래프를 연다.
`s`는 데이터베이스를 동기화한다. 노드 선택 후보는 대량
노트에서도 빠르게 열리도록 10분간 캐시하며, 60초 유휴 상태마다 갱신한다.
노트를 추가하거나 제목을 바꾼 직후 후보가 오래되었다면 메뉴의 `c`로 즉시
초기화할 수 있다. Org-roam 데이터베이스는 자동 동기화된다.

일정·할 일용 `~/notes/agenda.org`도 없을 때만 생성한다. 같은 이름의 기존 파일은
덮어쓰지 않는다. `~/notes/` 바로 아래의 `.org` 파일을 agenda 대상으로 추가하며,
기존에 등록한 다른 agenda 대상도 유지한다. 하위 폴더는 자동으로 포함하지 않는다.
`C-c h o`로 Org Agenda Transient를 연다. `a` 일정, `t` 전체 TODO,
`e` agenda.org 열기, `S` 기본 폴더 설정을 제공한다. Org 제목 위에서는
`s` 일정 지정, `d` 마감일 지정, `T` TODO 상태 변경, `c` 카테고리 설정도 사용할 수 있다.
`c`는 현재 제목에 `CATEGORY` 속성을 설정하며 하위 제목에도 상속된다.
`x` 내보내기는 Org 문서에서는 기본 내보내기 메뉴(`C-c C-e`)를 열고,
Agenda 화면에서는 현재 일정 보기를 파일로 저장한다. 문서는 메뉴에서 HTML·텍스트
등의 형식을 선택하고, Agenda는 저장할 파일의 확장자(`.txt`, `.html`, `.ics`, `.org` 등)로
형식을 선택한다. PDF 출력에는 별도의 로컬 변환 도구가 필요할 수 있다.
`M-x imoogi-org-agenda`로 기본 agenda 메뉴를 열어 `a`로 일정, `t`로 TODO 목록을 볼 수 있다.
일정 보기(`C-c h o a`, `M-x imoogi-org-agenda-overview`)는 오늘보다 마감일이
이른 미완료 항목을 맨 위 **기한 지난 항목**에 마감일 순서로 한 번씩 모은다.
TODO 키워드가 없는 제목도 포함하며, 완료한 항목은 이 모음에서 제외한다.
모은 항목은 아래 일별 일정에서 반복 표시하지 않는다. 오늘·미래 마감일과
마감일 없이 예약한 일정은 기존 일정 보기를 따른다. Agenda의 `g`로 새로고침한다.
사용자가 별도로 지정한 agenda 메뉴의 `a` 명령은 덮어쓰지 않는다.

분류는 Org 기본 **카테고리**와 **태그**를 사용한다. 카테고리는 큰 구분 하나,
태그는 여러 라벨에 적합하다. 별도의 분류 목록을 강제하지 않는다.

```org
#+CATEGORY: 강의

* TODO 강의 업로드 :업로드:온라인:
DEADLINE: <2026-09-10 Thu>
```

카테고리는 기본적으로 파일 이름이며 `#+CATEGORY:`로 파일 전체를 지정하거나
제목의 `CATEGORY` 속성으로 하위 트리별로 지정한다(`C-c C-x p`).
태그는 제목에서 `C-c C-q`로 편집하며 부모 제목의 태그를 기본적으로 상속한다.
Agenda에서 `<`는 현재 항목의 카테고리로 필터, `/`는 태그·카테고리 등의 필터,
`|`는 필터 해제다.

일정·마감일은 Org 항목에서 `C-c C-s` / `C-c C-d`로 입력한다.
날짜 선택 캘린더는 아래쪽 전체 너비로 표시하며, 높이는 창의 약 40% 이내로 맞춘다.
공간이 부족하면 달력 글자만 자동 축소하고, 창을 넓히면 원래 확대 배율로 복원한다.
본문 글자 크기는 유지한다. 아주 작은 창에서는 9pt 이하로 줄이지 않으므로
달력 일부가 잘릴 수 있으며, 이때도 날짜를 직접 입력할 수 있다.

Anki에도 이 폴더를 포함하려면 `M-x imoogi-anki-register-directory`로 `~/notes/`를 등록한다.
`imoogi-org-setup` 자체는 Anki 설정이나 동기화를 실행하지 않는다.

Org와 Markdown 제목은 깊이에 따라 빨강 → 파랑 → 초록 → 노랑 순서로 표시한다.
Markdown의 `#`부터 `######`까지 글자색과 줄 전체 배경색이 적용되며,
5·6단계는 빨강·파랑을 반복한다. `markdown-mode`와 `gfm-mode` 모두 지원한다.
본문 영역에는 테두리를 표시하지 않는다.

### gptel: LiteLLM · Codex · Claude · 기타 API

`gptel`은 `vendor/elpa/`에 동봉되어 있어 Emacs 부팅과 패키지 로딩에는 인터넷이
필요하지 않다. `M-x imoogi-gptel-setup`을 실행하면 먼저 연결 방식을 묻는다.

- **LiteLLM Gateway**: Gateway 주소와 key를 입력하면 `/v1/models`에서 모델 목록 조회
- **Codex / ChatGPT Plus·Pro OAuth**: OpenAI 계정 로그인 사용, API key 불필요
- **Claude / Anthropic API**: Anthropic API와 `auth-source`의 API key 사용
- **기타 OpenAI 호환 API**: base URL, endpoint, 모델 이름을 직접 지정

기타 OpenAI 호환 API는 base URL 또는 완성된 Chat endpoint URL을 받을 수 있다. 예를
들어 `https://gateway.example.com/api/ai_interface/chat/completions`를 입력하면 setup이
base URL `https://gateway.example.com`과 endpoint
`/api/ai_interface/chat/completions`로 자동 분리한다. base URL만 입력하면 endpoint를
별도로 묻는다. `gateway.example.com/api/...`처럼 scheme을 생략하면 `https://`를
자동으로 적용한다.

LiteLLM을 선택했다면 먼저 Gateway의 `config.yaml`에 사용할 모델 별칭을 등록한다.

```yaml
model_list:
  - model_name: claude-sonnet
    litellm_params:
      model: anthropic/claude-sonnet-4-5
  - model_name: coding-small
    litellm_params:
      model: openai/gpt-4.1-mini
```

LiteLLM은 다음 순서로 설정한다.

1. `새 profile 등록`: Gateway를 구분할 새 이름 입력, 예: `company`, `personal`
2. Base URL: API prefix까지 포함한 정확한 주소, 예: `https://gateway.example.com/custom`
3. API 형식: 기본값 `자동 / OpenAI Chat`; 필요할 때 `Anthropic Messages` 선택
4. Chat endpoint: base URL 뒤의 상대 경로, 예: `/v1/chat/completions`
5. Model list endpoint: 모델 목록을 조회할 상대 경로, 예: `/v1/models`
6. API key: Gateway host별로 `auth-source`에 저장
7. Model aliases: 지정한 Model list endpoint에서 자동 조회
8. `C-c h i m`의 요청 메뉴에서 `-m`을 눌러 실제 사용할 모델 선택

`C-c h i N`은 새 profile만 등록하며 이미 사용 중인 이름은 거부한다. `C-c h i E`는
등록된 이름을 자동완성으로 선택하고 Gateway 설정을 수정한다. `C-c h i G` 또는
`M-x imoogi-gptel-switch-litellm-profile`은 자동완성 목록에서 활성 Gateway를 바꾼다.
일반 `M-x imoogi-gptel-setup`에서 LiteLLM을 선택하면 새 profile 등록 흐름으로 들어간다.
경로 prefix가 있는 Gateway는 그 경로까지 base URL에 넣는다. Base URL이
`https://gateway.example.com/custom`이고 Chat endpoint가 `/v1/chat/completions`라면
채팅은 `/custom/v1/chat/completions`, 모델은 `/custom/v1/models`를 사용한다.
두 endpoint는 프로필별로 저장되므로 특수한 Gateway에서는 Chat endpoint와 Model list
endpoint를 서로 다른 경로로 지정할 수 있다.

`C-c h i O`로 `~/.emacs.d/imoogi-gptel.json`을 직접 열 수 있다. API key는 이 파일에
쓰지 않는다. `litellm_profiles` 배열에서 profile의 `gateway_url`, `endpoint`,
`models_endpoint`, `models`, `default_model`을 수정하고 저장한 다음 `C-c h i R`로 다시
읽는다. `active_profile`과 이름이 같은 profile이 현재 설정으로 적용된다.

```json
{
  "gateway_url": "https://gateway.example.com/custom",
  "provider": "litellm",
  "api_protocol": "openai-chat",
  "endpoint": "/v1/chat/completions",
  "models_endpoint": "/special/models",
  "models": ["company-model"],
  "default_model": "company-model",
  "active_profile": "company",
  "litellm_profiles": [
    {
      "name": "company",
      "gateway_url": "https://gateway.example.com/custom",
      "api_protocol": "openai-chat",
      "endpoint": "/v1/chat/completions",
      "models_endpoint": "/special/models",
      "models": ["company-model"],
      "default_model": "company-model"
    }
  ]
}
```

setup은 기존 선택 모델이 조회 목록에 있으면 유지하고, 없으면 첫 모델을 임시
기본값으로 등록한다. setup 중에는 모델을 묻지 않는다. 모델 조회에 실패하면
Gateway 주소와 key를 고칠 수 있도록 오류를 그대로 표시한다. OpenAI Chat은
`/v1/chat/completions`와 `gptel-make-openai`를 사용하고, Anthropic Messages는
`/v1/messages`와 `gptel-make-anthropic`을 사용한다. 이전 설정 파일에 API 형식이
없으면 OpenAI Chat으로 읽어 기존 동작을 유지한다.

설정 함수는 공급자, 주소와 모델만 `~/.emacs.d/imoogi-gptel.json`에 저장한다.
LiteLLM·Claude·기타 API key는 설정 마지막 질문이나 `M-x imoogi-gptel-store-key`로 Emacs
`auth-source`에 별도로 저장한다. 저장소와 JSON 파일에는 API key가 들어가지 않는다.
기본 `~/.authinfo`가 아직 없으면 setup이 빈 파일을 만들고 권한을 `0600`으로
설정한다. 기존 파일은 덮어쓰지 않으며 `.authinfo.gpg`는 EasyPG로 미리 생성해야 한다.
Codex를 선택하면 `gptel-openai-oauth-login`이 OpenAI 로그인을 진행하고 토큰은 gptel의
OAuth 토큰 파일에 저장된다.
암호화 저장을 원하면 `auth-sources`에 `~/.authinfo.gpg`를 우선 등록한 뒤 setup을
실행한다. 전체 안내는 `M-x imoogi-gptel-setup-guide`에서 다시 볼 수 있다.

gptel 설정과 등록 과정은 `~/.emacs.d/.cache/imoogi-gptel.log`에 단계별로 기록된다.
모델 조회가 실패하면 HTTP 상태 코드, Content-Type, 최대 2KB의 응답 본문 요약을
`model-fetch-response`에 기록한다. DNS/TLS 등의 네트워크 예외와 10초 무응답은 각각
`model-fetch-network-error`, `model-fetch-no-response`로 구분한다. API key, token,
secret, password 값은 로그에서 가린다.
`model-fetch-no-response` 뒤에는 DNS, TCP, TLS probe 결과가
`model-network-dns`, `model-network-tcp`, `model-network-tls` 순서로 기록된다.
각 probe는 기본 3초로 제한되며 HTTP 인증 정보는 전송하지 않는다.

폐쇄망 Gateway가 사설 CA를 사용한다면 PEM CA bundle을 지정한다.

```elisp
(setq imoogi-ca-certificate-file "/내부/경로/company-ca.pem")
```

파일은 기존 시스템 trust store에 추가되며 모델 조회와 실제 gptel 요청에 함께
사용된다. 설정 여부와 읽기 가능 여부만 `model-network-ca`에 기록되고 인증서 내용은
로그에 기록하지 않는다.
Emacs 안에서는 `C-c h s P`로 PEM을 선택하면 전역 시스템 설정에 경로가 저장된다.
`C-c h s X`는 추가한 CA 설정을 해제한다.
한 실행에 속한 기록은 같은 `action-id`로 묶이며 공급자 선택, profile 추가·수정·전환,
설정 파일 읽기·쓰기, auth-source 파일 확인과 key 저장·재조회, endpoint와 모델 조회,
backend 생성을 추적할 수 있다. API key, token, secret, password 값은 항상
`<redacted>`로 기록되고 로그 파일 권한은 `0600`으로 맞춘다.

실패 직후 `C-c h i D`를 실행하면 현재 Gateway에서 계산한 auth host, endpoint,
`auth-sources` 파일의 존재·읽기·쓰기 권한, 현재 host의 `apikey` 항목 존재 여부를
확인할 수 있다. `C-c h i L`은 상세 로그를 연다. 이 기능을 설치하기 전에 이미 발생한
오류 메시지는 `C-h e` (`view-echo-area-messages`)에서 확인한다.

| 키 | 동작 |
| --- | --- |
| `C-c h i c` | LiteLLM 채팅 버퍼 열기 |
| `C-c h i s` | 현재 영역 또는 버퍼의 prompt 전송 |
| `C-c h i m` | gptel 모델·옵션·도구 Transient 열기 |
| `C-c h i N` | 새 LiteLLM Gateway profile 등록 |
| `C-c h i E` | 등록된 LiteLLM Gateway profile 수정 |
| `C-c h i G` | 저장된 LiteLLM Gateway profile 전환 |
| `C-c h i O` | gptel 설정 JSON 직접 열기 |
| `C-c h s P` | 사설 CA PEM을 Emacs 전역에 설정하고 저장 |
| `C-c h s X` | Emacs 전역 사설 CA 설정 해제 |
| `C-c h i R` | 수정한 gptel 설정 JSON 다시 읽기 |
| `C-c h i a` | 현재 영역 또는 버퍼를 추가 문맥으로 등록 |
| `C-c h i f` | 파일을 추가 문맥으로 등록 |
| `C-c h i S` | LiteLLM Gateway 설정 |
| `C-c h i k` | virtual key를 auth-source에 등록 |
| `C-c h i D` | 현재 gptel·auth-source 상태 진단 |
| `C-c h i L` | 비밀값이 제거된 단계별 진단 로그 열기 |
| `C-c h i h` | 내장 설정 가이드 보기 |

### Org/Markdown 브라우저 미리보기 (간단 버전)

한 번 `make build-org-preview`로 Go 서버를 빌드한다. Org 파일에서는
**`C-c h o v`**, Org와 Markdown 파일 모두에서는 **`C-c h m v`** 또는
`M-x imoogi-org-preview`를 실행하면 로컬 서버와 브라우저가 열린다. 저장하지
않은 수정도 마지막 변경 후 0.5초 뒤 갱신한다. Org 메뉴의 **`C-c h o V`**나
`M-x imoogi-org-preview-stop`으로 현재 버퍼의 자동 미리보기를 종료한다.

왼쪽 **목차**와 본문 제목은 Org 또는 Markdown 제목 깊이에 따라
빨강·파랑·초록·노랑으로 표시한다.
7단계 이상에서도 같은 순서를 반복하며, 제목 HTML 자체에 글자색과 배경색을 포함한다.
**개요** 탭은 문단·목록·코드 등을 제목 색상을 따른 카드로 보여준다.
항목을 클릭하면 본문의 해당 위치로 이동한다. 작은 화면에서는 목차가 본문 위에 배치된다.

Org와 Markdown 목록은 원문의 `-`, `+`, `*`, `1.`, `1)` 기호와 중첩
들여쓰기를 그대로 구분해 표시한다. Org의 `[ ]`, `[-]`, `[X]` 체크박스도 상태별
색으로 보이지만 미리보기에서는 읽기 전용이다. Markdown의 대괄호 표기는 일반
텍스트로 유지한다. 탭 들여쓰기는 8칸 기준으로 계산한다.

Org의 Mermaid source block은 브라우저에서 다이어그램으로 렌더링한다.

```org
#+begin_src mermaid
flowchart LR
  자료 --> 개념정리 --> 암기카드 --> 복습
#+end_src
```

Mermaid 12.0.0 런타임은 미리보기 실행 파일에 포함되므로 CDN이나 인터넷 연결이
필요하지 않다. 다른 언어의 source block은 기존처럼 코드로 표시한다.

Org property drawer는 `:PROPERTIES:`와 `:END:` 문법을 숨기고 모든 속성의
이름과 값을 작은 정보 표로 표시한다. 로컬 `file:` 링크는 프로젝트 또는 현재 문서의
허용된 폴더 안에서만 동작한다. Org와 Markdown 문서는 저장된 내용을 브라우저 안에서
렌더링하고, 이미 같은 preview session에서 열린 buffer라면 최신 내용을 사용한다.
이미지·텍스트·PDF는 브라우저에서 표시하며 ZIP이나 실행 파일처럼 지원하지 않는
형식은 다운로드하지 않고 안내 화면을 보여준다.

Org 해석과 HTML 생성은 별도 Go 프로세스가 담당한다. Emacs는 비동기 HTTP로
내용을 전달하며, 전송 중 변경은 모아서 다음 요청에 최신 내용을 보낸다.
현재 위치는 0.15초 뒤 작은 위치 메시지로 전달해 브라우저의 문단을 강조한다.
브라우저 스크롤은 강제로 이동하지 않으며 커서 양방향 동기화는 기본으로 끈다.
위치 전송에는 문서 문법 해석을 수행하지 않는다. 전체 버퍼 복사와 JSON 인코딩은
Emacs에서 실행하므로 대용량 문서에서 비용이 완전히 없어지는 것은 아니다.
Org와 Markdown 문법 전체 및 정식 내보내기와의 완전한 일치는 보장하지 않는다.

서버는 로컬 주소에서만 실행하고 화면 자원은 실행 파일에 포함한다.
부팅 시 서버 실행이나 다운로드는 하지 않는다. 폐쇄망에는 대상 OS용으로 빌드한
`bin/imoogi-org-preview` 실행 파일도 함께 반입한다.

### Org/Markdown 클립보드 자산 붙여넣기

먼저 Go CLI를 빌드한다. 저장소 안에서만 쓰려면 `make build-clipboard`,
PATH에 설치하려면 `make install-clipboard`를 실행한다. 빌드 버전은
`make clipboard-version IMOOGI_CLIP_VERSION=버전`으로 확인할 수 있다.
실행 중 다운로드나 네트워크 연결은 없다.

Org 또는 Markdown 버퍼에서 평소처럼 `C-y`나 macOS의 `Command-v`를 누른다.
클립보드가 텍스트이면 기존 Emacs yank가 그대로 동작한다. 파일이나 스크린샷이면
CLI가 먼저 클립보드 세대를 확인하고 자산을 복사한 뒤 문서 형식에 맞는 링크를 넣는다.
파일을 Emacs 창으로 드래그해 놓는 방식도 같은 자산 규칙을 사용한다. 폴더 붙여넣기와
폴더 드래그는 재귀 복사의 부작용을 피하기 위해 거부한다.

| 문서 상태 | 자산 위치 |
| --- | --- |
| project-notes 안의 문서 | 프로젝트 최상위 `assets/` |
| 그 밖의 저장된 문서 | 문서 옆 `<파일명>.assets/` |
| 아직 저장하지 않은 버퍼 | 버퍼·세대별 임시 staging; 첫 저장이 성공하면 최종 위치로 이동 |

저장하지 않은 버퍼를 버리면 staging 자산도 폐기할 수 있다. 저장·이름 변경·Org refile은
Emacs 안에서 수행한 경우 링크와 자산을 함께 조정한다. Finder나 터미널에서 문서만 따로
옮기는 경우는 자동 추적하지 않는 알려진 제한이다. 실패 원인과 최근 요청은
`M-x imoogi-clipboard-show-diagnostics`에서 확인한다.

현재 네이티브 클립보드 어댑터는 macOS AppKit/NSPasteboard를 지원한다. Windows,
X11, Wayland는 플랫폼별 실제 클립보드 형식과 컴파일 검증을 거친 뒤 활성화하며,
현재 빌드에서는 명시적인 `unsupported-capability` 응답을 반환한다. 자세한 지원 범위와
검증 기준은 `docs/clipboard-platform-support.md`에 있다.

Anki 미디어는 기존 동기화 루트 안의 파일만 읽는다. 프로젝트 공용 `assets/`를 카드에서
참조한다면 개별 Org 파일 대신 프로젝트 폴더를 동기화 대상으로 등록해야 한다.
독립 문서의 `<파일명>.assets/`는 그 문서가 속한 등록 폴더 안이므로 그대로 동작한다.

### Anki 동기화 대상 등록

`M-x imoogi-sync`는 `imoogi-anki-setup`에서 지정한 기존 폴더와 이 PC에
추가 등록한 폴더·파일을 함께 동기화한다. 등록 목록은 프로젝트가 아닌
Emacs 사용자 설정 디렉터리의 `imoogi-targets.json`(기본 `~/.emacs.d/imoogi-targets.json`)에
저장되어 재시작 후에도 유지된다. 저장 위치는 `imoogi-targets-file`로 변경할 수 있다.

| 명령 | Org 단축키 | 동작 |
| --- | --- | --- |
| `imoogi-anki-register-directory` | `C-c a D` | 폴더 등록: 하위 폴더의 `.org` 파일도 포함 |
| `imoogi-anki-register-file` | `C-c a F` | `.org` 파일 하나 등록: 같은 폴더의 다른 파일은 제외 |
| `imoogi-anki-list-targets` | `C-c a l` | 등록 목록과 경로 상태 확인 |
| `imoogi-anki-list-files` | `C-c a L` | 기본 폴더·등록 폴더 안의 `.org`와 개별 등록 파일을 중복 없이 펼쳐 보기 |
| `imoogi-anki-open-log` | `C-c a g` | 영속 Anki 동기화 진단 로그 열기 |
| `imoogi-anki-unregister-target` | `C-c a u` | 추가 등록 해제: 원본 파일과 Anki 카드는 유지 |
| `imoogi-sync` | `C-c a s` | 기존 폴더와 추가 등록 대상을 모두 동기화 |

사용 순서:

1. `M-x imoogi-anki-register-directory`로 동기화할 폴더를 선택한다.
2. 다른 위치의 파일도 포함하려면 `M-x imoogi-anki-register-file`로 해당 `.org` 파일을 선택한다.
3. `M-x imoogi-anki-list-targets`로 등록 목록을 확인한다.
4. Org 파일을 저장하고 `M-x imoogi-sync`를 실행한다. 등록 명령 자체는 동기화를 실행하지 않는다.

`M-x` 명령은 어느 버퍼에서든 사용할 수 있고, 위 단축키는 Org 버퍼에서 사용한다.
Anki 메뉴(`C-c a a`)의 **동기화 대상**에서도 같은 명령을 실행할 수 있다.

등록 목록의 **파일 목록 펼치기** 또는 `M-x imoogi-anki-list-files`로 실제 파일을 확인한다.
파일 목록에서 `RET`로 파일을 열고, `g`로 새로고침하며, `q`로 닫는다.
제외 패턴에 걸린 파일은 `[제외]`, 누락되었거나 읽을 수 없는 경로는 `[확인 실패]`로 표시한다.
카드 heading이 없는 `.org` 파일도 목록에는 표시되며, 목록 조회는 Anki 동기화를 실행하지 않는다.

예를 들어 `~/notes/` 폴더와 `~/work/vocabulary.org` 파일을 각각 등록하면
어느 버퍼에서 실행하든 둘 다 처리한다. 기존 폴더 설정 없이 추가 등록만으로도
동기화할 수 있다. Anki 연결과 전용 노트 타입 준비는 기존 setup과 같다.
파일을 먼저 저장한 뒤 동기화한다. 카드로 표시한 제목(`ANKI_NOTE_TYPE`)만 대상이며,
겹치는 폴더·파일과 심볼릭 링크는 같은 파일을 중복 처리하지 않는다.

추가 등록 대상의 동기화 상태는 사용자 설정 디렉터리의 `imoogi-target-state/`에
보관한다. 추가 대상은 카드 추가·갱신만 하며, 제목이나 파일을 없애거나 등록을
해제해도 Anki 카드를 자동 삭제하지 않는다. 기존 setup 폴더의 삭제 규칙은 유지한다.
읽지 못한 대상이 있으면 보고하고 해당 실행의 자동 삭제는 억제한다.
서로 다른 제목이 같은 `ANKI_NOTE_ID`를 사용하면 해당 카드들의 갱신을 막고
충돌을 보고하며, 그 실행에서는 자동 삭제도 억제한다.
이미지는 등록 폴더 안에서 참조하며, 개별 파일은 그 파일이 있는 폴더를 기준으로 한다.

### Anki 카드 옵션 속성

heading에 붙일 수 있는 카드 옵션 속성이 세 개 있다. 값은 `ANKI_DECK`과 똑같은
방식으로 상속된다. 앞의 두 개는 실제로 카드 모양을 바꾼다(아래 "묻고 답하는
카드" 참고). `ANKI_SWIFT`는 아직 값을 검사하는 데까지만 한다.

| 속성 | 받는 값 | 쓰임 |
| --- | --- | --- |
| `ANKI_DIRECTION` | `->`, `<-`, `<->`, `nil` | 어느 쪽을 문제로 낼지 |
| `ANKI_INCREMENTAL` | `t`, `nil` | 항목마다 빈칸을 따로 만들지 |
| `ANKI_SWIFT` | `t`, `nil` | 본문의 줄을 한 줄씩 쪼갠 카드로 만들지 |

`t`가 켬, `nil`이 끔이다. 앞뒤 공백과 대소문자는 가리지 않아서 `T`와 `NIL`도
그대로 받는다. 표에 없는 값을 쓰면 그 heading은 건너뛰고 `card_option_invalid`로
보고한다. 어느 속성의 어떤 값이 걸렸는지 진단 로그에 그대로 남으므로 고칠 줄을
바로 찾을 수 있다.

값은 자기 drawer → 가장 가까운 상위 heading의 drawer → 파일 맨 위의
`#+PROPERTY:` 순서로 찾고, 먼저 찾은 값 하나만 쓴다. 여러 단계의 값을 합치지는
않는다. 상속받은 값을 이 heading에서만 끄려면 자기 drawer에 `nil`을 쓰거나 값을
비워 둔다.

```org
#+PROPERTY: ANKI_SWIFT t

* 이 heading에서만 swift를 끈다
:PROPERTIES:
:ANKI_NOTE_TYPE: imoogi-Cloze
:ANKI_SWIFT:
:ANKI_DIRECTION: ->
:END:
```

카드 옵션을 켠 heading은 `ANKI_NOTE_TYPE`이 `imoogi-Cloze`나 `Cloze`여야 한다.
이 옵션들이 만들 카드가 모두 빈칸(cloze) 카드이기 때문이다. 다른 타입에 옵션을
켜 두면 건너뛰고 `card_option_needs_cloze`로 보고한다. 끈 값(`nil`)만 있는
heading은 옵션을 켠 것이 아니므로 이 제한에 걸리지 않는다. 파일 전체에
`#+PROPERTY:`로 옵션을 걸어 둔 채 Basic heading을 섞어 쓸 때 이 점이 중요하다.

`ANKI_SWIFT`를 켠 채로 `ANKI_DIRECTION`이나 `ANKI_INCREMENTAL`도 함께 켜면 서로
다른 모양의 카드를 동시에 요구하는 셈이라 건너뛰고 `card_option_conflict`로
보고한다. 둘 중 하나를 지우면 된다. 한쪽이 상위 heading이나 파일에서 상속된
값일 수도 있으니, 위의 끄는 방법을 함께 참고한다.

이번 변경으로 front end와 바이너리 사이의 프로토콜 버전이 2로 올라갔다. 예전
바이너리를 그대로 두면 `binary_incompatible`이 나므로 `make build-anki`로 다시
빌드한다.

### Anki 묻고 답하는 카드

`ANKI_DIRECTION`이나 `ANKI_INCREMENTAL`을 켠 heading은 묻고 답하는 카드가 된다.
**heading 제목이 질문이고, 본문의 첫 목록이 답이다.** 빈칸은 imoogi가 알아서
넣으므로 `{{c1::...}}`를 손으로 쓰지 않아도 된다.

```org
* 일본의 수도
:PROPERTIES:
:ANKI_NOTE_TYPE: imoogi-Cloze
:ANKI_DIRECTION: ->
:END:

- 도쿄
- 오사카
```

`C-c a r`로 방향을, `C-c a i`로 항목별 분리를 지정한다(Anki 메뉴에도 있다).
둘 다 노트 타입이 없으면 `imoogi-Cloze`를 함께 달아 주고, 이미 Cloze 계열이면
그대로 둔다. 상속받은 옵션을 이 heading에서만 끄면 속성을 지우는 것이 아니라
`nil`을 써 넣는다 — 지우면 상위 heading이나 파일의 값이 다시 드러나기 때문이다.

**답을 어디서 읽는가.** 본문에서 맨 처음 나오는 목록 하나만 답이 된다. 그 앞의
문단이나 블록은 문제 쪽 설명으로 남고, 뒤에 나오는 두 번째 목록은 그냥 본문이다.
목록은 `-`도 `1.`도 `용어 :: 뜻`도 모두 받는다. 항목 아래에 한 단 들여 쓴
하위 항목은 답이 아니라 보이는 설명으로 남는다. `#+BEGIN_EXTRA` 블록은 답으로
읽지 않는다 — 답 사이에 끼워 넣어도 앞뒤 답이 하나로 이어진다.

본문에 목록이 아예 없으면 그 heading만 건너뛰고 `multiline_answer_missing`으로
보고한다. 방향이 `<-`여도 마찬가지다.

**네 가지 조합.** 답이 `- 도쿄`, `- 오사카` 두 개일 때:

| `ANKI_DIRECTION` | `ANKI_INCREMENTAL` | 나오는 카드 |
| --- | --- | --- |
| `->` (또는 비움) | `nil` | 카드 한 장. 두 답이 한꺼번에 가려지고 제목이 보인다 |
| `->` (또는 비움) | `t` | 카드 두 장. 답 하나씩 번갈아 가려진다 |
| `<-` | 무시됨 | 카드 한 장. 제목이 가려지고 두 답이 보인다 |
| `<->` | `nil` | 카드 두 장. 답이 한꺼번에 가려진 것과 제목이 가려진 것 |
| `<->` | `t` | 카드 세 장. 답 하나씩, 그리고 제목 |

`ANKI_DIRECTION`을 쓰지 않고 `ANKI_INCREMENTAL`만 켜면 `->`로 친다. `<-`는
제목만 가리므로 항목별로 나눌 것이 없고, 그래서 `ANKI_INCREMENTAL`은 조용히
무시된다(진단도 나오지 않는다).

**직접 쓴 빈칸은 건드리지 않는다.** 답 항목이나 제목에 `{{cN::...}}`를 이미
써 두었으면 그 항목은 감싸지 않고 글자 그대로 둔다. Anki의 빈칸은 겹쳐 쓸 수
없어서, 감쌌다면 이미 만들어 둔 카드가 깨지기 때문이다. imoogi가 새로 붙이는
번호는 그 heading 안에서 가장 큰 번호보다 위에서 시작하므로 겹치지 않는다.

### Anki 동기화 진단 로그

`imoogi-sync`, note type 설치, migration 실행은 기본적으로
`~/.emacs.d/.cache/imoogi-anki.log`에 JSON Lines 형식의 진단 기록을 남긴다.
`M-x imoogi-anki-open-log`, Org 버퍼의 `C-c a g`, 또는 Anki transient의
**진단 로그**로 열 수 있다. 저장 위치는 `imoogi-anki-log-file`로 바꾸며 nil이면
로깅을 끈다.

각 실행은 command 시작, 스캔된 카드의 key·source path·note type·note ID,
AnkiConnect handshake, registry load/save, add/update/skip 결과와 오류 코드·원문을
기록한다. 카드 제목과 본문, 렌더링된 필드 내용은 기록하지 않는다. 로그 파일은
권한 `0600`으로 만들고 5 MiB에 도달하면 기존 파일을 `.1`로 한 번 회전한다.
Go 로깅이 포함된 바이너리는 `make build-anki`로 다시 설치한다.

`imoogi-Basic` 항목이 보이지 않을 때에는 로그에서 `event`가 `sync_entry`인 줄의
`note_type`을 먼저 확인한다. 해당 줄이 없다면 저장된 Org 파일이 스캔되지 않은
것이고, `sync_error`가 있다면 같은 줄의 `code`와 `message`에서 AnkiConnect 또는
note type 오류 원문을 확인할 수 있다. `imoogi-sync`는 기존 note type을 사용하지만
새 note type을 설치하지는 않는다. Anki의 **Note Types** 목록에 `imoogi-Basic`과
`imoogi-Cloze`가 없다면 `M-x imoogi-anki-setup`을 실행해 두 타입을 먼저 설치한다.

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
| `14-org` | org · org-appear | org-mode |
| `27-gptel` | gptel · auth-source | LiteLLM Gateway 기반 LLM 채팅·문맥·요청 |
| `28-clipboard` | Org · Markdown · 외부 `imoogi-clip` CLI | 텍스트 yank 보존, 파일·스크린샷 자산 붙여넣기와 저장 수명주기 |
| `15-markdown` | markdown-mode · markdown-toc | Markdown + Org-style 구조 편집 키 |
| `16-elisp` | aggressive-indent · highlight-defined · paredit · page-break-lines · elisp-refs | Elisp 개발 |
| `17-lsp` | Eglot · Flymake · xref (Emacs 30 내장) | 공통 LSP 설정 + `modules/development/lang/*.el` 언어별 자동 로더 |
| `18-languages` | git-modes · yaml · dockerfile · gnuplot · lua · jinja2 · csv · go · rust · crontab · nginx · hcl · nix · fish · vimrc · jenkinsfile · clojure · kotlin · typescript · web/tsx (+내장 sgml/java/treesit) | 21종 파일타입 모드 + 선택적 `*-ts-mode` 전환 |
| `19-folding` | kirigami · outline-indent (+내장 outline/hs-minor) | 코드 폴딩 (`C-c z` 접두) |
| `20-terminal` | ghostel (+ghostel-ime) | libghostty-vt 터미널 (`C-c t`). 모듈은 vendor 동봉, S-SPC 한글 동작 |
| `21-native-compile` | compile-angel | 로드 시 바이트/네이티브 컴파일 |
| `25-flashcards` | SQLite · Org (Emacs 내장) | Anki 미설치 폐쇄망용 로컬 flashcard fallback (`C-c f`) |
| `26-project-notes` | Org · project · JSON (Emacs 내장) | 프로젝트 기록 템플릿·worktree 공유·영속 Scratch (`C-c h p m`) |
| `00-defaults` | (내장) | 상대 줄번호, 줄:열 표시, treesit 레벨4, pixel-scroll, fringe |

### 이미 반영돼 있던 추천 (중복 도입 안 함)

`recentf` · `savehist` · `saveplace` · `auto-revert` (00-defaults/09), `eglot`/`flymake` 기본값 (17-lsp), `which-key`(Emacs 30 내장, 03), `uniquify`, `treemacs`(07), `magit`(06), 폰트·테마(10).

### 의도적으로 미반영 (이유 명시)

| 추천 | 미반영 이유 |
|------|-------------|
| `auto-package-update` | 네트워크로 자동 업데이트 → **망분리 철학과 정면 충돌**. 업데이트는 온라인 머신 vendoring 으로만. |
| `treesit-fold` | 폴딩 패키지 추가와 문법 빌드가 필요해 기본 구성에서는 제외. 문법은 `vendor/tree-sitter/` 로 반입 가능. |
| `inhibit-mouse` | 마우스를 끄는 동작은 과격 — 문서화만. |
| `evil` / `treemacs-evil` | 의도적으로 미사용·제거했으며 vendor 에 없으므로, 쓰려면 `packages.el` 에 명시 추가 후 온라인 머신에서 재-vendoring 해야 한다. |
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

### 설정 모듈 서브패키지

설정 소스는 `modules/general/`, `modules/project/`, `modules/org/`,
`modules/development/`로 구분한다. 언어별 LSP는 `modules/development/lang/`에
있다. 기존 단축키와 사용자 데이터 저장 위치는 동일하며 `M-x imoogi-reload`로
적용할 수 있다. [구조](ARCHITECTURE.md)와
[보존 시나리오·검증 범위](docs/refactoring/module-packages.md)를 참고한다.
