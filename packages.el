;;; packages.el --- imoogi-emacs 패키지 매니페스트(SSOT) -*- lexical-binding: t; -*-

;; 망분리(air-gap) 대응을 위한 vendoring 의 단일 진실 원천.
;; 여기 적힌 top-level 패키지만 명시하면 전이 의존성(transient, with-editor,
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
    projectile perspective persp-projectile
    ;; 05-hydra
    hydra ace-window
    ;; 06-git
    magit
    ;; 07-treemacs (treemacs-evil→evil, treemacs-persp→persp-mode 필요)
    treemacs treemacs-projectile treemacs-icons-dired treemacs-magit
    treemacs-evil treemacs-persp evil persp-mode
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
    ;; 14-org-markdown (markdown-mode 는 obsidian 의존성)
    org-appear markdown-toc
    ;; 15-elisp
    aggressive-indent highlight-defined paredit page-break-lines elisp-refs
    ;; 16-languages
    git-modes yaml-mode dockerfile-mode gnuplot lua-mode jinja2-mode csv-mode
    go-mode rust-mode crontab-mode nginx-mode hcl-mode nix-mode fish-mode
    vimrc-mode jenkinsfile-mode
    ;; 17-folding (treesit-fold 는 tree-sitter 문법 필요로 제외)
    kirigami outline-indent
    ;; 18-terminal (모듈은 타겟 첫 실행 시 빌드)
    vterm
    ;; 19-native-compile
    compile-angel)
  "imoogi-emacs 가 요구하는 top-level 패키지 목록.
전이 의존성은 package.el 이 자동으로 함께 설치한다.")

(provide 'imoogi-packages)
;;; packages.el ends here
