;;; packages.el --- imoogi-emacs 패키지 매니페스트(SSOT) -*- lexical-binding: t; -*-

;; 망분리(air-gap) 대응을 위한 vendoring 의 단일 진실 원천.
;; 여기 적힌 top-level 패키지만 명시하면 전이 의존성(with-editor,
;; dash, markdown-mode 등)은 package.el 이 자동 해결한다.
;;
;; 사용처:
;;   - scripts/vendor.el  : 온라인 머신에서 이 목록을 vendor/elpa/ 로 설치
;;   - (런타임 boot.el 은 이 파일을 읽지 않는다 — 설치된 vendor/ 만 사용)
;;
;; 패키지 추가/삭제 = 이 리스트 수정 후 온라인 머신에서 `make vendor` 재실행.
;; which-key 는 Emacs 30 에 내장되어 있어 목록에서 제외한다.

;;; Code:

(defvar imoogi-required-packages
  '(;; 02-completion (vertico 스택 — minimal-emacs.d 추천)
    vertico orderless marginalia embark embark-consult consult corfu cape
    ;; 04-projects
    perspective
    ;; 05-transient
    transient ace-window
    ;; 06-git
    magit
    ;; 07-treemacs (treemacs-persp/treemacs-evil 은 imoogi 스택과 안 맞아 제외)
    treemacs treemacs-icons-dired treemacs-magit
    ;; 08-obsidian (straight → package.el 로 이관, markdown-mode 동반)
    obsidian
    ;; 10-theme
    doom-themes doom-modeline nerd-icons
    ;; 11-editing
    undo-fu undo-fu-session yasnippet yasnippet-snippets apheleia
    dumb-jump stripspace
    ;; 12-navigation
    avy helpful diff-hl bufferfile
    ;; 13-system
    exec-path-from-shell buffer-terminator persist-text-scale
    ;; 14-org
    org-appear
    ;; 15-markdown (markdown-mode 는 obsidian 의존성)
    markdown-toc edit-indirect
    ;; 16-elisp
    aggressive-indent highlight-defined paredit page-break-lines elisp-refs
    ;; 18-languages
    git-modes yaml-mode dockerfile-mode gnuplot lua-mode jinja2-mode csv-mode
    go-mode rust-mode crontab-mode nginx-mode hcl-mode nix-mode fish-mode
    vimrc-mode jenkinsfile-mode clojure-mode kotlin-mode typescript-mode web-mode
    ;; tree-sitter 메이저 모드 — Emacs 30 에 내장되지 않은 언어만 패키지로 보충.
    ;; (json/js/ts/tsx/python/go/java/yaml 은 내장이라 문법만 있으면 된다)
    kotlin-ts-mode clojure-ts-mode
    ;; 19-folding (treesit-fold 는 tree-sitter 문법 필요로 제외)
    kirigami outline-indent
    ;; 20-terminal (ghostel; 네이티브 모듈은 vendor/ghostel-module/ 에 동봉)
    ghostel
    ;; 21-native-compile
    compile-angel)
  "imoogi-emacs 가 요구하는 top-level 패키지 목록.
전이 의존성은 package.el 이 자동으로 함께 설치한다.")

(provide 'imoogi-packages)
;;; packages.el ends here
