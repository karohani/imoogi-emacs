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

언어 서버(LSP)까지 쓰려면 `docs/toolchains.md` 의 `imoogi-toolchain setup` 을
추가로 실행한다. 실행하지 않아도 에디터는 정상 동작하며 LSP 기능만 꺼진 채로 남는다.

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

**버전을 바꿨을 때 가장 먼저 할 일이다.** `modules/*.elc` 는 `.gitignore` 대상이라
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
| 키 | 동작 |
|----|------|
| `M-x` | 명령 실행 (vertico 세로 완성) |
| `C-x C-f` | 파일 열기 |
| `C-s` / `M-s l` | consult-line — 현재 버퍼 검색 |
| `C-x b` | 현재 Perspective 버퍼 전환 · `C-u C-x b` 전체 버퍼 |
| `M-s r` / `M-s g` | consult-ripgrep / grep — 프로젝트·디렉터리 검색 |
| `M-g g` | 줄 이동 · `M-g i` imenu · `M-g f` flymake 진단 |
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

언어별 자동 연결은 `modules/lsp/` 아래에서 독립적으로 관리한다. 현재 Bash,
JavaScript/TypeScript, Go, Python, Rust, Clojure, Java, Kotlin 설정이 있으며,
새 언어는 같은 폴더에 설정 파일 하나를 추가하면 `17-lsp`가 자동으로 로드한다.
Go/TypeScript 서버는 저장소의 `vendor/toolchains/`에 고정된 artifact를 사용한다.
온라인 머신에서는 `vendor/toolchains/cli/1.0.0/darwin-arm64/imoogi-toolchain fetch`로
`toolchains.lock.json`과 artifact를 갱신하고, 폐쇄망 타겟에서는 같은 bootstrap의
`setup`만 실행한다. `setup`은 `.local/toolchains/<bundle>/`에 검증 후 설치하고
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
| 키 | 동작 |
|----|------|
| `C-x p p` | 프로젝트 선택 → 대응 Perspective 전환/생성 → 프로젝트 Dired |
| `C-u C-x p p` | 현재 Perspective를 유지하고 다른 프로젝트를 함께 열기 |
| `C-x x s / c / l` | Perspective 생성·전환 / 종료 / 직전 작업공간 복귀 |
| `C-x x a / g` | 버퍼를 현재 Perspective에 추가 / 여러 Perspective용 전역 공유 |
| `C-c h` → `p` | 프로젝트·작업공간 메뉴 (아래) |

`C-c h` → `p` 는 세 갈래로 나뉜다.

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
| `15-markdown` | markdown-mode · markdown-toc | Markdown + Org-style 구조 편집 키 |
| `16-elisp` | aggressive-indent · highlight-defined · paredit · page-break-lines · elisp-refs | Elisp 개발 |
| `17-lsp` | Eglot · Flymake · xref (Emacs 30 내장) | 공통 LSP 설정 + `modules/lsp/*.el` 언어별 자동 로더 |
| `18-languages` | git-modes · yaml · dockerfile · gnuplot · lua · jinja2 · csv · go · rust · crontab · nginx · hcl · nix · fish · vimrc · jenkinsfile · clojure · kotlin · typescript · web/tsx (+내장 sgml/java/treesit) | 21종 파일타입 모드 + 선택적 `*-ts-mode` 전환 |
| `19-folding` | kirigami · outline-indent (+내장 outline/hs-minor) | 코드 폴딩 (`C-c z` 접두) |
| `20-terminal` | ghostel (+ghostel-ime) | libghostty-vt 터미널 (`C-c t`). 모듈은 vendor 동봉, S-SPC 한글 동작 |
| `21-native-compile` | compile-angel | 로드 시 바이트/네이티브 컴파일 |
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
