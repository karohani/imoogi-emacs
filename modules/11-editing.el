;;; 11-editing.el --- 편집 강화 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; undo-fu, yasnippet, apheleia, stripspace, dumb-jump, elec-pair 등
;; minimal-emacs.d 가 권장하는 편집 관련 패키지 모음.

;;; Code:

;;; undo-fu — 더 편한 undo/redo (한 번에 redo, 과도한 redo 방지)
(use-package undo-fu
  :ensure t
  :config
  (global-unset-key (kbd "C-z"))
  (global-set-key (kbd "C-z")   'undo-fu-only-undo)
  (global-set-key (kbd "C-S-z") 'undo-fu-only-redo))

;;; undo-fu-session — 세션 간 undo 히스토리 유지
(use-package undo-fu-session
  :ensure t
  :hook (after-init . undo-fu-session-global-mode))

;;; yasnippet — 스니펫 템플릿 확장
(use-package yasnippet
  :ensure t
  :hook (after-init . yas-global-mode)
  :custom
  (yas-also-auto-indent-first-line t)
  (yas-also-indent-empty-lines t)
  (yas-snippet-revival nil)
  (yas-wrap-around-region nil)
  :init
  (setq yas-verbosity 0))

(use-package yasnippet-snippets
  :ensure t
  :after yasnippet)

;;; apheleia — 커서 방해 없는 비동기 코드 포매팅(Black, Prettier, shfmt 등)
(use-package apheleia
  :ensure t
  :hook (prog-mode . apheleia-mode))

;;; dumb-jump — 50+ 언어 'go to definition' (xref 백엔드)
(use-package dumb-jump
  :ensure t
  :init
  (add-hook 'xref-backend-functions #'dumb-jump-xref-activate 90)
  (setq dumb-jump-aggressive nil
        dumb-jump-max-find-time 3
        dumb-jump-selector 'completing-read)
  (when (executable-find "rg")
    (setq dumb-jump-force-searcher 'rg
          dumb-jump-prefer-searcher 'rg)))

;;; stripspace — 저장 시 끝부분 공백/빈 줄 자동 제거(커서 열 보존)
(use-package stripspace
  :ensure t
  :hook ((prog-mode . stripspace-local-mode)
         (text-mode . stripspace-local-mode)
         (conf-mode . stripspace-local-mode))
  :custom
  (stripspace-only-if-initially-clean nil)
  (stripspace-restore-column t))

;;; elec-pair — 괄호/따옴표 자동 짝맞춤(내장)
(use-package elec-pair
  :ensure nil
  :hook (after-init . electric-pair-mode))

;;; 선택 영역 위에 입력하면 대체 (Delete Selection mode)
(delete-selection-mode 1)

(provide 'imoogi-editing)
;;; 11-editing.el ends here
