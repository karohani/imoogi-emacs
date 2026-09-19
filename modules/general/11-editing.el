;;; 11-editing.el --- 편집 강화 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; undo-fu, yasnippet, apheleia, stripspace, dumb-jump, elec-pair 등
;; minimal-emacs.d 가 권장하는 편집 관련 패키지 모음.

;;; Code:

(imoogi-require "11-editing" 'undo-fu 'undo-fu-session 'yasnippet
                'yasnippet-snippets 'apheleia 'dumb-jump 'stripspace)

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

;;; 소프트 랩 — 긴 줄을 화면에서만 접어 보여주기(내장)
;; 코드(prog-mode)는 00-defaults.el 의 truncate-lines t 를 그대로 두고,
;; 산문 계열(text-mode 파생 — org, markdown 등)에서만 랩을 켠다. 파일에는
;; 개행이 들어가지 않으므로(하드 랩이 아님) diff 가 지저분해지지 않는다.
;; 주의: visual-line-mode 는 C-a/C-e/C-k 를 '논리 줄'이 아닌 '화면 줄' 기준으로
;; 바꾼다. 논리 줄 기준을 유지하려면 M-m / C-M-e 등을 쓰거나 이 훅을 뺀다.
(add-hook 'text-mode-hook #'visual-line-mode)

;;; visual-wrap — 접힌 줄에 원래 들여쓰기를 이어 붙임(Emacs 30 내장)
;; 외부 adaptive-wrap 패키지가 내장으로 흡수된 것. 소프트 랩된 목록/인용문이
;; 왼쪽 끝으로 붙어 버리는 문제를 막는다.
(use-package visual-wrap
  :ensure nil
  :hook (text-mode . visual-wrap-prefix-mode))

;;; so-long — 한 줄이 지나치게 긴 파일에서 무거운 기능 자동 해제(내장)
;; 미니파이 JS, 대용량 로그 등 so-long-threshold(기본 10000자)를 넘는 줄이 있는
;; 파일에서 폰트 락 등을 끄고 Emacs 가 멎는 것을 막는다. 기본 동작은 메이저 모드를
;; so-long-mode 로 교체하는 것이며, 원래 모드를 유지하고 싶으면
;; (setq so-long-action 'so-long-minor-mode) 로 바꾼다.
(use-package so-long
  :ensure nil
  :hook (after-init . global-so-long-mode))

(provide 'imoogi-editing)
;;; 11-editing.el ends here
