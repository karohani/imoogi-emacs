# Architecture

## 로딩 흐름

```
~/.emacs.d/early-init.el       ← (로더) → early-init.el
  └── early-init.el            ← 스타트업 성능 최적화(GC, file-name-handler, UI 억제)

~/.emacs.d/init.el
  └── boot.el                  ← package.el → vendor/elpa/, use-package (오프라인)
        └── modules/ (순서대로 load)
              ├── defaults     ← 더 나은 기본값 + recentf/savehist/saveplace
              ├── completion   ← ivy, counsel, swiper
              ├── which-key    ← Emacs 30 내장
              ├── projects     ← project.el, perspective
              ├── hydra        ← ace-window, hydra 정의, 글로벌 키바인딩
              ├── git          ← magit
              ├── keys         ← 한영전환, 한글 key-translation-map
              ├── treemacs     ← treemacs + 관련 패키지
              ├── obsidian     ← obsidian (vendored)
              ├── autorevert
              └── theme        ← doom-themes, doom-modeline, nerd-icons
```

## 디렉토리 레이아웃

```
~/workspace/imoogi-emacs/          ← git 저장소 (실제 파일)
  ├── early-init.el                ← 스타트업 성능 최적화
  ├── boot.el                      ← 로더
  ├── packages.el                  ← 패키지 매니페스트(SSOT)
  ├── packages.lock                ← 동결 버전 기록(감사용)
  ├── scripts/vendor.el            ← 온라인 vendoring 스크립트
  ├── vendor/elpa/                 ← 동봉된 패키지 (커밋됨, 망분리용)
  ├── vendor/tree-sitter/          ← 선택적 tree-sitter 문법 라이브러리
  ├── assets/fonts/                ← Nerd Font (동봉, 선택)
  └── modules/                     ← 기능 모듈
~/.config/imoogi-emacs/            ← 심볼릭 링크 → ~/workspace/imoogi-emacs
~/.emacs.d/                        ← Emacs 런타임 (init.el, early-init.el 로더, .cache/)
  ├── early-init.el                ← repo의 early-init.el 로더
  └── init.el                      ← boot.el 로더 + custom-set-* 블록
```

설정 코드와 패키지 모두 `~/workspace/imoogi-emacs`에 존재한다(self-contained).
`~/.emacs.d/`는 로더와 캐시(.cache/, eln-cache 등)만 포함한다.

## 망분리(air-gap) 설계

저장소 하나를 클론해 폐쇄망에 들고 들어가면 인터넷 없이 동작한다.

- **부팅 경로에 네트워크 의존 없음** — `boot.el` 은 `package-refresh-contents`
  를 호출하지 않고, `package-user-dir` 을 저장소 안 `vendor/elpa/` 로 지정한다.
- **vendoring** — 온라인 머신에서 `scripts/vendor.el` 이 `packages.el` 목록 +
  전이 의존성을 `vendor/elpa/` 로 설치하고 바이트컴파일한다.
- **lock** — 커밋된 `vendor/` 자체가 동결 상태(git 커밋 = 버전 고정).
  MELPA 는 rolling 아카이브라 원격 재설치로는 버전 재현이 불가능하므로,
  진실의 원천은 원격이 아니라 git 이다. `packages.lock` 은 사람이 읽는 감사 기록.
- **업데이트** = 온라인 머신에서 vendor 재실행 → `vendor/` 커밋 → 폐쇄망 반입.
  폐쇄망 내부에서는 업데이트하지 않는다(네트워크 필요).
- **Tree-sitter 문법** — 온라인 머신에서 빌드한 grammar 라이브러리를
  `vendor/tree-sitter/` 에 커밋해 반입한다. 부팅 중 다운로드/빌드는 하지 않는다.
- **버전 일치** — 빌드 머신과 타겟의 Emacs 메이저 버전을 맞출 것(.elc 호환).

## 모듈 로딩 순서

boot.el의 `dolist`에서 정의된 순서대로 로딩된다. 의존성이 있으므로 순서가 중요하다:

0. **00-defaults** — 내장 기본값/세션 영속(외부 패키지 없음), 가장 먼저
1. **01-keys** — 한글 키매핑 (독립적)
2. **02-completion** — vertico/consult/corfu 스택 (hydra가 consult 함수 참조)
3. **03-which-key** — Emacs 30 내장
4. **04-projects** — project.el + perspective 작업공간 연결/영속화
5. **05-hydra** — ace-window + hydra 정의 (consult, project.el, magit 함수 참조)
6. **06-git** — magit (독립적)
7. **07-treemacs** — 전역 treemacs + project.el/magit/icons-dired 연동
8. **08-obsidian** — obsidian (독립적)
9. **09-autorevert** — global-auto-revert (독립적)
10. **10-theme** — doom-themes/doom-modeline (treemacs 뒤라야 연동 동작)
11. **11-editing** — undo-fu, yasnippet, apheleia, dumb-jump, stripspace, elec-pair
12. **12-navigation** — avy, helpful, diff-hl, bufferfile
13. **13-system** — exec-path-from-shell, server, buffer-terminator, persist-text-scale
14. **14-org-markdown** — org, org-appear, markdown-toc
15. **15-elisp** — aggressive-indent, paredit, highlight-defined 등
16. **16-languages** — 21종 파일타입 메이저 모드
17. **17-folding** — kirigami, outline-indent, 내장 outline/hs-minor
18. **18-terminal** — ghostel + ghostel-ime (모듈은 vendor/ghostel-module/ 동봉)
19. **19-native-compile** — compile-angel (소급 컴파일하므로 마지막)

## 패키지 관리

| 메커니즘 | 용도 | 저장 위치 |
|----------|------|-----------|
| package.el + use-package | 모든 패키지 | `vendor/elpa/` (저장소에 커밋) |
| `packages.el` | 필요 패키지 매니페스트(SSOT) | 저장소 루트 |
| `scripts/vendor.el` | 온라인 vendoring·갱신 | 저장소 루트 |
| `packages.lock` | 동결 버전 기록(감사용) | 저장소 루트 |

straight.el 은 망분리 대응을 위해 제거됐다(부트스트랩이 네트워크 의존).

## 새 모듈 추가 방법

1. `modules/이름.el` 파일 생성
2. `;;; Code:` 바로 아래에 **사전조건 점검**을 맨 위에 둔다:
   `(imoogi-require "이름" 'pkg1 'pkg2 ...)` — 필요한 라이브러리가 vendor 에
   동봉됐는지(또는 내장인지) `locate-library` 로 확인하고, 누락 시 그 모듈만
   건너뛴다(boot.el 이 `condition-case` 로 감싸 나머지 모듈은 계속 로딩).
3. `use-package` 선언 작성, 끝에 `(provide 'imoogi-이름)` 추가
4. `boot.el`의 모듈 리스트에 `"이름"` 추가 (의존성 순서 고려)
5. 새 패키지를 쓰면 `packages.el` 의 `imoogi-required-packages` 에 추가 후,
   온라인 머신에서 `emacs --batch -Q -l scripts/vendor.el` 재실행 → `vendor/` 커밋

## 테스트 전략

세 가지 층으로 나뉘며, 각 층이 답하는 질문이 다르다. 합격 기준이 서로 달라서
곱하지 않고 나눠 실행한다.

| 층 | 질문 | 실행 | 합격 기준 |
|---|---|---|---|
| 부팅 | 설치 후 모듈이 다 올라오나 | `tests/assert-boot.el` | 스킵된 모듈이 기대 집합과 정확히 일치 |
| 단위/동작 | 각 기능이 정의대로 동작하나 | `tests/*-test.el` (ERT) | 전부 통과 |
| 이식성 | 다른 OS/Emacs 에서도 같은가 | `tests/docker/run.sh` | 위 둘을 각 이미지에서 재현 |

```sh
EMACS=/path/to/emacs ./tests/run.sh   # 호스트: check-parens + 오프라인 부팅 + ERT
./tests/docker/run.sh                 # 컨테이너 매트릭스 (위 전체를 각 이미지에서 재실행)
```

### 종료 코드는 판정 근거가 아니다

`boot.el` 은 모듈 실패를 격리하므로(한 모듈이 죽어도 나머지는 로딩) **설정이
절반 깨져도 Emacs 는 `exit 0` 으로 끝난다.** 실측된 두 사례:

- `git` 부재 → `06-git`, `07-treemacs` 스킵, `exit 0`
- Emacs 29.3 + 30.x `.elc` → 모듈 3개 스킵 + use-package 오류 다수, `exit 0`

그래서 판정은 **무엇이 로드됐는지**로 한다. `boot.el` 이 건너뛴 모듈을
`imoogi-failed-modules` 에 기록하고, `tests/assert-boot.el` 이 그 집합을 기대값과
양방향으로 대조하며(예상 못 한 스킵 + 예상했으나 일어나지 않은 스킵 둘 다 실패),
`imoogi` 외 타입의 error 급 경고(use-package 등)도 실패로 잡는다. 계측 변수가
없으면 "스킵 0개"로 읽혀 거짓 통과가 나므로, 변수 미정의 자체를 실패 처리한다.

### 대화형 팝업(transient) 검증

transient 메뉴는 "키를 눌렀을 때 팝업이 유지되는가/닫히는가"가 핵심 동작인데,
이는 배치 모드에서 **실제로 검증 가능하다**(`tests/transient-menu-test.el`).
두 층으로 나눠 잡는다.

1. **구조** — `transient--make-predicate-map` 이 만든 keymap 을 명령 심볼 벡터로
   조회해, 각 suffix 가 실제 디스패치될 pre-command 를 본다. 선언된 `:transient`
   속성을 읽는 것보다 강하다: transient 자신의 기본값 해석까지 거친 결과다.

   | pre-command | 의미 |
   |---|---|
   | `transient--do-call` | 실행 후 팝업 유지 |
   | `transient--do-exit` | 실행 후 팝업 닫힘 |
   | `transient--do-stack` | 하위 메뉴로 진입(스택) |
   | `transient--do-quit-one` | 종료 |

2. **동작** — `transient-setup` 으로 메뉴를 띄우고 `execute-kbd-macro` 로 실제
   키를 눌러 `transient--prefix` 로 유지/종료를 관찰한다. 배치 모드에서 창 분할과
   `text-scale` 변화까지 그대로 재현된다.

프롬프트를 띄우는 명령(`project-find-file`, `magit-status` 등)은 배치에서 멈추므로
동작 층에서 제외하고 구조 층으로만 검증한다.

이 테스트들은 transient 내부 API(`transient--*`)에 의존한다. transient 업그레이드로
깨지면 공개 API인 `transient-get-suffix` 의 `:transient` 속성을 읽는 방식으로 낮춰
잡는다(더 약한 검증).

#### hydra 대비 유일한 동작 차이

suffix 로 지정한 명령이 **그 자체로 transient prefix 인 경우**(현재
`magit-blame` 하나), transient 는 이를 항상 `do-stack` 으로 디스패치한다 —
`do-exit` 으로 만들 수 없다(`transient.el` `transient--make-predicate-map`,
`prefix/nil → do-stack`). 그 결과 hydra `:color blue` 시절에는 메뉴가 닫히고
`magit-blame` 이 떴지만, 지금은 Git 메뉴가 스택에 쌓인 채 `magit-blame` 메뉴가
열리고 거기서 `q` 를 누르면 Git 메뉴로 돌아온다. 라이브러리 고유 동작이라
설정 쪽에서 되돌릴 수 없다.

## 모듈이 메뉴에 항목을 등록하는 방식

`05-transient.el` 은 하위 메뉴를 하드코딩하지 않는다. 각 모듈이 자기 메뉴를
정의하고 `transient-append-suffix` 로 마스터에 **스스로 등록**한다
(`modules/17-lsp.el` 의 LSP 메뉴가 이 방식의 참조 구현).

```elisp
(with-eval-after-load 'imoogi-transient
  (transient-define-prefix imoogi-transient-foo () ...)
  (transient-append-suffix 'imoogi-transient-master "t"
    '("F" "Foo" imoogi-transient-foo)))
```

이 방향으로 두는 이유는 degradation 모델과 맞추기 위해서다. 05 가 17 을 알면
17 이 로드 실패했을 때 마스터에 죽은 항목이 남지만, 반대로 두면 모듈이 사라질 때
항목도 함께 사라진다 — `imoogi-require` 가 이미 하는 일과 같은 결이다.

- **`with-eval-after-load 'imoogi-transient` 로 감싼다.** 05 보다 먼저 로드되는
  모듈(01~04)은 마스터가 아직 없어 바로 붙일 수 없다. 06 이후라도 감싸 두면 모듈
  번호가 바뀌어도 깨지지 않는다.
- **transient 를 `imoogi-require` 사전조건에 넣지 않는다.** 등록 블록 전체가
  지연되므로, 05 가 건너뛰어져도 해당 모듈의 접두 맵(예: `C-c l`)은 그대로 산다.

### 키 충돌은 조용히 일어난다

[HARD] `transient-append-suffix` 가 **이미 있는 키**를 만나면 키 집합은 그대로 둔 채
**명령만 교체한다**(실측). 그래서 "키 목록이 기대와 같다"는 단언으로는 덮어쓰기를
잡지 못한다 — 실제로 그런 테스트를 먼저 쓴 뒤 일부러 충돌시켜 통과하는 것을 확인했다.
`tests/transient-menu-test.el` 은 **(키 . 명령) 쌍**을 통째로 단언한다. 새 모듈이
항목을 등록하면 이 기대값도 함께 갱신해야 하고, 그 강제가 곧 충돌 감지다.

### 문맥 의존 항목

마이너 모드를 새로 만들 필요는 대개 없다. transient 항목에 조건을 걸면 된다.

| 키워드 | 효과 |
|---|---|
| `:if` `:if-not` `:if-non-nil` `:if-nil` | 조건 불충족 시 **숨김** |
| `:if-mode` `:if-not-mode` `:if-derived` `:if-not-derived` | major-mode 기준 숨김 |
| `:inapt-if-*` (같은 접미) | 숨기지 않고 **흐리게** — 기능의 존재는 알린다 |

발견성 면에서는 `inapt` 쪽이 대체로 낫다. LSP 메뉴는 Eglot 이 붙지 않은 버퍼에서
서버 의존 명령(`R c o K`)을 흐리게만 처리한다. 술어가 참조하는 함수가 아직 로드되지
않았을 수 있으므로 `fboundp` 로 먼저 막는다(`imoogi-lsp-eglot-active-p` 참조).

마이너 모드는 키를 **켜고 끌 수 있어야** 하거나 **특정 버퍼에서만** 살아야 할 때만
만든다. 그 경우 `define-minor-mode` 의 `:keymap` 에 진입점을 두면 마이너 모드 맵이
전역보다 우선하므로 버퍼 안에서만 다른 메뉴를 띄울 수 있다.

### 팝업 크기와 열 정렬

[HARD] **transient 는 열 폭을 `length`(문자 수)로 계산한다** — `transient.el` 전체에
`string-width` 호출이 하나도 없다. 한글은 표시 폭이 2라 문자 수로 세면 열이 밀린다.
실측(`imoogi-transient-window`, 지정 전/후 열 시작 위치):

```
지정 전: (0 6 14) (0 7) (0 7 25) (25)      ← 행마다 제각각
지정 후: (0 15 32 53) × 데이터 행 전부      ← 정렬됨
```

그래서 한글 설명이 들어가는 prefix 에는 `:column-widths` 로 **표시 폭 기준** 값을
직접 준다. 그룹 제목 뒤의 `─` 는 열 경계를 눈으로 잡아주는 구분선이며, 지정한 폭을
넘지 않도록 길이를 맞춘다(넘으면 그 행이 다음 열을 밀어낸다).

[HARD] **그룹 제목의 구분선은 ASCII 로 쓴다.** 박스 드로잉 문자(`─` 등)는 East
Asian Ambiguous 라 폰트에 따라 2칸으로 그려지는데, Emacs 의 `char-width` 는 1로
계산한다. 그 차이만큼 그 줄이 오른쪽으로 밀리고, 데이터 행은 밀리지 않아 헤더만
어긋난다. NanumGothicCoding 실측:

```
"열기 ───────────"  실제 27칸 / 계산 16칸   ← 11칸 어긋남
"열기 -----------"  실제 16칸 / 계산 16칸   ← 일치
" p 프로젝트 전환"  실제 16칸 / 계산 16칸
```

`tests/transient-menu-test.el` 이 제목에 그런 문자가 다시 들어오는지 지킨다.

새 메뉴를 추가할 때 필요한 폭은 이렇게 구한다.

```elisp
(+ 2 (max (string-width "그룹 제목")
          (string-width "k 가장 긴 설명")))   ; 키 + 공백 포함
```

팝업 높이는 `imoogi-transient-min-height-fraction`(기본 `0.33`)으로 프레임 높이의
최소 비율을 정한다. transient 는 `fit-window-to-buffer` 를 최소 높이 1로 호출해
내용에 딱 맞게 줄이므로, `transient--fit-window-to-buffer` 에 `:after` advice 를 걸어
하한만 끌어올린다(내용이 더 길면 그만큼 커진다). `nil` 로 두면 transient 기본 동작.

> **주의 — `.elc` 캐시**: 모듈을 고친 뒤 `emacs --batch -l boot.el` 로 확인할 때는
> `load-prefer-newer` 를 켜지 않으면 **옛 `.elc` 를 읽어** 변경이 반영되지 않은 것처럼
> 보인다. `tests/run.el` 은 이미 켜 두었다. 모듈 이름을 바꿨다면 옛 이름의 `.elc`
> (예: `05-hydra.elc`)도 지운다 — `.gitignore` 대상이라 커밋에는 안 잡히지만 로컬에서
> 계속 로드될 수 있다.
