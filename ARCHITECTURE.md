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
              ├── projects     ← projectile, perspective
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
- **버전 일치** — 빌드 머신과 타겟의 Emacs 메이저 버전을 맞출 것(.elc 호환).

## 모듈 로딩 순서

boot.el의 `dolist`에서 정의된 순서대로 로딩된다. 의존성이 있으므로 순서가 중요하다:

0. **00-defaults** — 내장 기본값/세션 영속(외부 패키지 없음), 가장 먼저
1. **01-keys** — 한글 키매핑 (독립적)
2. **02-completion** — vertico/consult/corfu 스택 (hydra가 consult 함수 참조)
3. **03-which-key** — Emacs 30 내장
4. **04-projects** — projectile, perspective
5. **05-hydra** — ace-window + hydra 정의 (consult, projectile, magit 함수 참조)
6. **06-git** — magit (독립적)
7. **07-treemacs** — treemacs + projectile/magit/perspective/evil 연동
8. **08-obsidian** — obsidian (독립적)
9. **09-autorevert** — global-auto-revert (독립적)
10. **10-theme** — doom-themes/doom-modeline (treemacs 뒤라야 연동 동작)
11. **11-editing** — undo-fu, yasnippet, apheleia, dumb-jump, stripspace, elec-pair
12. **12-navigation** — avy, helpful, diff-hl, bufferfile
13. **13-system** — exec-path-from-shell, server, buffer-terminator, persist-text-scale
14. **14-org-markdown** — org, org-appear, markdown-toc
15. **15-elisp** — aggressive-indent, paredit, highlight-defined 등
16. **16-languages** — 16종 파일타입 메이저 모드
17. **17-folding** — kirigami, outline-indent, 내장 outline/hs-minor
18. **18-terminal** — vterm (모듈은 타겟 첫 실행 시 빌드)
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
2. `use-package` 선언 작성, 끝에 `(provide 'imoogi-이름)` 추가
3. `boot.el`의 모듈 리스트에 `"이름"` 추가 (의존성 순서 고려)
4. 새 패키지를 쓰면 `packages.el` 의 `imoogi-required-packages` 에 추가 후,
   온라인 머신에서 `emacs --batch -Q -l scripts/vendor.el` 재실행 → `vendor/` 커밋
