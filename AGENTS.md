# AGENTS.md — imoogi-emacs

AI 코딩 에이전트가 이 저장소에서 작업할 때 반드시 지켜야 할 규칙과 맥락. 사람용 개요는 `README.md`, 구조는 `ARCHITECTURE.md` 참고.

## 0. 가장 중요한 제약 — 망분리(air-gap)

이 설정은 **인터넷이 차단된 폐쇄망에서 동작해야 한다.** 저장소를 클론해 들고 들어가면 외부 네트워크 없이 그대로 작동하는 것이 목표(self-contained).

- **부팅 경로(early-init.el → boot.el → modules/)에 네트워크 의존을 절대 추가하지 말 것.**
  - 금지: `package-refresh-contents`, 런타임 패키지 다운로드, `url-retrieve`, `nerd-icons-install-fonts`, 부팅 중 `brew install` 등.
- 모든 패키지는 저장소 안 `vendor/elpa/` 에 동봉(커밋)되어 있다. `boot.el` 은 `package-user-dir` 을 거기로 지정하고 `use-package-always-ensure nil` 로 둔다.
- 커밋된 `vendor/` 자체가 lock(버전 동결). MELPA 는 rolling 이라 원격 재현이 불가능하므로 진실의 원천은 git 이다. `packages.lock` 은 사람이 읽는 감사 기록.

## 1. 디렉터리 구조

```
early-init.el            스타트업 성능 최적화(GC, file-name-handler, UI 억제)
boot.el                  로더: package-user-dir→vendor, imoogi-require 정의, 모듈 로딩
packages.el              패키지 매니페스트(SSOT) — imoogi-required-packages
packages.lock            동결 버전 기록(감사용, vendor 스크립트가 생성)
scripts/vendor.el        온라인 vendoring 스크립트
vendor/elpa/             동봉 패키지(커밋됨)
assets/fonts/            동봉 글꼴(NanumGothicCoding, NFM.ttf) + OFL.txt
modules/NN-name.el       기능 모듈(번호 순 로딩)
```

런타임 로더: `~/.emacs.d/{early-init,init}.el` 은 `~/.config/imoogi-emacs`(저장소 심볼릭 링크)의 파일을 load 만 한다.

## 2. 모듈 작성 규칙

각 `modules/NN-name.el` 은:

1. `;;; Code:` **바로 아래 최상단에 사전조건 점검**을 둔다:
   ```elisp
   (imoogi-require "NN-name" 'pkg1 'pkg2 ...)
   ```
   `imoogi-require`(boot.el 정의)는 `locate-library` 로 필요 라이브러리가 vendor 에 있는지(또는 내장인지) 확인하고, 누락 시 error 를 시그널한다. boot.el 이 각 모듈 로딩을 `condition-case` 로 감싸므로 **그 모듈만 건너뛰고 나머지는 계속 로딩**된다.
2. `use-package` 로 설정. 내장 패키지는 `:ensure nil`, vendor 패키지는 `:ensure t`(이미 설치돼 있어 네트워크 안 탐).
3. 파일 끝에 `(provide 'imoogi-NAME)`.
4. `boot.el` 의 `dolist` 모듈 리스트에 `"NN-name"` 추가(로딩 순서 = 의존성 순서).

## 3. 패키지 추가/업데이트 (반드시 온라인 머신에서)

폐쇄망 안에서는 절대 vendoring 하지 않는다(네트워크 필요). 항상 온라인 빌드 머신 → 반입 순서.

```bash
# 1) packages.el 의 imoogi-required-packages 수정
# 2) vendoring (전이 의존성 자동 해결 + 바이트컴파일 + packages.lock 갱신)
emacs --batch -Q -l scripts/vendor.el            # 누락분만 설치
emacs --batch -Q -l scripts/vendor.el -- upgrade # 전체 최신 갱신
# 3) 커밋
git add packages.el packages.lock vendor/ && git commit
# 4) 폐쇄망으로 반입(내부 git 미러 pull 또는 저장소 재반입)
```

- 빌드/타겟 Emacs **메이저 버전 일치** 필수(.elc 호환). 현재 30.x / macOS.
- 네이티브 `.eln` 은 머신별 캐시라 vendor 에 넣지 않는다(타겟에서 JIT/compile-angel 로 생성).

## 4. 검증 (변경 후 항상)

```bash
# 문법
emacs --batch -Q --eval '(with-temp-buffer (insert-file-contents "FILE") (check-parens))'
# 오프라인 부팅(네트워크 아카이브 0개로 vendor 만으로 로드되는지)
emacs --batch -Q --eval '(setq user-emacs-directory "/tmp/t/")' -l boot.el
```

`package-archives` 를 nil 로 둔 채 전 모듈이 로드되면 air-gap 안전이 입증된 것.

## 5. 핵심 결정사항(되돌리지 말 것)

- **완성 스택은 vertico/consult/corfu** (ivy/counsel/swiper 에서 이관됨). hydra 는 consult/projectile 함수를 참조한다.
- **straight.el 제거됨** — 모든 패키지는 package.el + vendoring.
- **which-key 는 Emacs 30 내장**(`:ensure nil`).
- **한글 입력**: macOS 는 포커스 시 OS 입력을 영문 강제(im-select), 한글은 Emacs 입력기(S-SPC). 단 **vterm 버퍼에서는 영문 강제를 건너뛰어 OS 한글 입력기를 쓴다**(vterm 은 Emacs 입력기 미지원).
- **vterm 네이티브 모듈**은 동봉하지 않고 **타겟 첫 실행 시 빌드**(호스트별로 다름). 빌드 산출물(`build/`, `*.so`, `*.dylib`)은 `.gitignore` 처리됨.
- **의도적 미반영**: auto-package-update(망분리 위반), treesit-fold(문법 빌드 필요), evil(미사용). README "미반영" 표 참고.

## 6. 커밋 관례

- 손으로 쓴 설정과 동봉 바이너리(vendor/, assets/)는 **별도 커밋**으로 분리.
- `.claude/`, `.omc/`, `.devcontainer/` 등 작업과 무관한 디렉터리는 커밋하지 말 것.
- 커밋 메시지 끝에 Co-Authored-By 트레일러 유지.
- 푸시/커밋은 사용자가 요청할 때만.
