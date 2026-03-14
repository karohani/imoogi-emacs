# Architecture

## 로딩 흐름

```
~/.emacs.d/init.el
  └── boot.el                  ← 패키지 저장소, use-package, straight.el 부트스트랩
        └── modules/ (순서대로 load)
              ├── completion   ← ivy, counsel, swiper
              ├── which-key
              ├── projects     ← projectile, perspective
              ├── hydra        ← ace-window, hydra 정의, 글로벌 키바인딩
              ├── git          ← magit
              ├── keys         ← 한영전환, 한글 key-translation-map
              ├── treemacs     ← treemacs + 관련 패키지
              └── obsidian     ← obsidian.el (straight.el)
```

## 디렉토리 레이아웃

```
~/workspace/imoogi-emacs/          ← git 저장소 (실제 파일)
~/.config/imoogi-emacs/            ← 심볼릭 링크 → ~/workspace/imoogi-emacs
~/.emacs.d/                        ← Emacs 런타임 (elpa/, straight/, .cache/ 등)
  └── init.el                      ← boot.el 로더 + custom-set-* 블록
```

설정 코드는 `~/workspace/imoogi-emacs`에만 존재한다.
`~/.emacs.d/`는 Emacs가 자동 생성하는 파일(elpa, straight, cache)만 포함한다.

## 모듈 로딩 순서

boot.el의 `dolist`에서 정의된 순서대로 로딩된다. 의존성이 있으므로 순서가 중요하다:

1. **completion** — ivy/counsel이 먼저 로드되어야 다른 모듈에서 counsel 함수 참조 가능
2. **which-key** — 독립적, 일찍 로드하여 이후 키바인딩에 설명 표시
3. **projects** — projectile, perspective (counsel에 의존)
4. **hydra** — ace-window + hydra 정의 (counsel, projectile, magit 함수 참조)
5. **git** — magit (독립적)
6. **keys** — 한글 키매핑 (독립적)
7. **treemacs** — treemacs + projectile/magit/perspective 연동
8. **obsidian** — straight.el로 설치 (독립적)

## 패키지 관리자

| 관리자 | 용도 | 저장 위치 |
|--------|------|-----------|
| package.el (MELPA/GNU/NonGNU) | 대부분의 패키지 | `~/.emacs.d/elpa/` |
| straight.el | GitHub 직접 설치 | `~/.emacs.d/straight/` |

## 새 모듈 추가 방법

1. `modules/이름.el` 파일 생성
2. `use-package` 선언 작성, 끝에 `(provide 'imoogi-이름)` 추가
3. `boot.el`의 모듈 리스트에 `"이름"` 추가 (의존성 순서 고려)
