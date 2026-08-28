# Makefile --- imoogi-emacs 진입점
#
# 두 갈래를 담는다.
#   1. 설치 — Emacs 본체(OS 별) + 이 저장소를 ~/.emacs.d 로 연결
#   2. 개발 — 테스트/린트/빌드, 그리고 pre-push 훅이 부르는 ci-local
#
# [HARD] ci-local 을 지우거나 이름을 바꾸지 말 것.
# .git/hooks/pre-push 는 Makefile 이 "존재하기만 하면" 무조건
# `make -C <repo> -s ci-local` 을 부르고, 그 실패를 push 실패로 취급한다.
# 이 타깃이 없으면 "No rule to make target 'ci-local'" 로 모든 push 가 막힌다.
# 훅을 건너뛰려면(기록됨): SKIP_MOAI_PREPUSH=1 git push

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

# 설치할 Emacs 버전. macOS 는 emacsformacosx.com 의 이 버전 .dmg 를 받는다.
EMACS_VERSION ?= 31.1

# 테스트에 쓸 Emacs 실행 파일.
# 주의: 이 저장소의 개발 머신에서 `emacs` 는 zsh alias 일 뿐 PATH 상의 바이너리가
# 아니다(실측). Make 는 alias 를 모르므로 PATH 에서 못 찾으면 macOS 표준 위치로
# 물러선다. 다른 빌드로 돌리려면: make test EMACS=/경로/Emacs
EMACS ?= $(shell command -v emacs 2>/dev/null || echo /Applications/Emacs.app/Contents/MacOS/Emacs)

GO ?= go
DIST_DIR := .cache/dist

.DEFAULT_GOAL := help
.PHONY: help emacs-install emacs-prewarm emacs-where install grammars build fmt fmt-check lint test test-elisp test-go ci-local clean

help: ## 이 도움말
	@echo "imoogi-emacs"
	@echo
	@echo "설치"
	@echo "  make emacs-install       Emacs 본체 설치 (OS 자동 감지, 기본 $(EMACS_VERSION))"
	@echo "  make emacs-prewarm       네이티브 컴파일 예열 (설치 시 자동, 끊겼을 때 재실행)"
	@echo "  make emacs-where         이 머신에서 찾은 Emacs 실행 파일 나열"
	@echo "  make install             이 저장소를 ~/.emacs.d 에 연결"
	@echo "  make grammars            tree-sitter 문법 빌드 (온라인 전용)"
	@echo
	@echo "개발"
	@echo "  make test                elisp + go 테스트"
	@echo "  make lint                go vet"
	@echo "  make fmt                 go fmt"
	@echo "  make build               imoogi-toolchain CLI 빌드"
	@echo "  make ci-local            pre-push 훅이 부르는 전체 검사"
	@echo "  make clean               내려받은 배포본 캐시 삭제"
	@echo
	@echo "변수"
	@echo "  EMACS_VERSION=$(EMACS_VERSION)"
	@echo "  EMACS=$(EMACS)"

## ---------------------------------------------------------------- 설치

emacs-install: ## Emacs 본체를 설치한다 (macOS: emacsformacosx.com, Linux: 배포판 패키지)
	@EMACS_VERSION=$(EMACS_VERSION) ./scripts/install-emacs.sh

emacs-prewarm: ## 네이티브 컴파일을 미리 끝내둔다 (emacs-install 이 자동으로 하지만, 끊겼을 때 이어서)
	@EMACS_VERSION=$(EMACS_VERSION) ./scripts/install-emacs.sh --prewarm "$(EMACS)"

emacs-where: ## 이 머신에서 찾은 Emacs 실행 파일을 모두 보여준다
	@echo "PATH:"
	@command -v emacs || echo "  (없음 — zsh alias 는 Make 에서 보이지 않는다)"
	@echo "/Applications:"
	@ls -d /Applications/Emacs*.app 2>/dev/null || echo "  (없음)"
	@echo
	@echo "make 가 쓸 값: $(EMACS)"
	@"$(EMACS)" --version 2>/dev/null | head -1 || echo "  (실행할 수 없음)"

install: ## 이 저장소를 ~/.emacs.d 로 연결한다
	@./scripts/install.sh

grammars: ## tree-sitter 문법을 vendor/tree-sitter/ 로 빌드한다 (온라인 머신 전용)
	@./scripts/build-grammars.sh

## ---------------------------------------------------------------- 개발

build: ## imoogi-toolchain CLI 를 빌드한다
	@$(GO) build ./...

fmt: ## Go 코드를 포맷한다
	@$(GO) fmt ./...

fmt-check: ## 포맷이 어긋난 파일이 있으면 실패한다 (ci-local 용)
	@unformatted="$$(gofmt -l . | grep -v '^vendor/' || true)"; \
	 if [ -n "$$unformatted" ]; then \
	   echo "gofmt 필요:" >&2; echo "$$unformatted" >&2; \
	   echo "고치려면: make fmt" >&2; exit 1; \
	 fi

lint: ## go vet
	@$(GO) vet ./...

test-elisp: ## elisp 테스트 (check-parens + 오프라인 부팅 + ert)
	@EMACS="$(EMACS)" ./tests/run.sh

test-go: ## Go 테스트
	@$(GO) test ./...

test: test-elisp test-go ## 전체 테스트

ci-local: fmt-check lint test ## pre-push 훅 진입점 — 지우지 말 것 (파일 상단 [HARD] 참고)

clean: ## 내려받은 Emacs 배포본 캐시를 지운다
	@rm -rf $(DIST_DIR)
	@echo "== 삭제됨: $(DIST_DIR)"
