;;; 15-elisp.el --- Elisp 개발 경험 강화 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; aggressive-indent, highlight-defined, paredit, page-break-lines, elisp-refs.
;; 대부분 emacs-lisp-mode 등 lisp 계열에만 적용한다.

;;; Code:

;;; aggressive-indent — 코드 작성 중 자동으로 들여쓰기 유지(lisp 계열)
(use-package aggressive-indent
  :ensure t
  :hook ((emacs-lisp-mode . aggressive-indent-mode)
         (lisp-mode . aggressive-indent-mode)))

;;; highlight-defined — 정의된 심볼(함수/변수/페이스)에 색 입히기
(use-package highlight-defined
  :ensure t
  :hook (emacs-lisp-mode . highlight-defined-mode))

;;; paredit — S-식(괄호) 구조를 유지하며 편집
(use-package paredit
  :ensure t
  :hook ((emacs-lisp-mode . paredit-mode)
         (lisp-mode . paredit-mode)
         (lisp-interaction-mode . paredit-mode)
         (scheme-mode . paredit-mode)))

;;; page-break-lines — ^L(폼 피드)을 가로줄로 표시
(use-package page-break-lines
  :ensure t
  :hook (after-init . global-page-break-lines-mode))

;;; elisp-refs — elisp 심볼 참조 검색(helpful 도 활용)
(use-package elisp-refs
  :ensure t
  :commands (elisp-refs-function elisp-refs-variable elisp-refs-symbol))

(provide 'imoogi-elisp)
;;; 15-elisp.el ends here
