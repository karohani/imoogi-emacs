;;; 02-completion.el --- Vertico/Consult/Corfu 완성 스택 -*- lexical-binding: t; -*-

;; minimal-emacs.d 추천 completion 스택. (기존 ivy/counsel/swiper 에서 이관)
;;   - vertico    : 미니버퍼 세로 완성 UI (M-x, C-x C-f 등)
;;   - orderless  : 공백 구분 다중 패턴 매칭
;;   - marginalia : 완성 후보에 풍부한 주석
;;   - embark     : 후보에 대한 컨텍스트 액션(우클릭 메뉴 같은)
;;   - consult    : 검색/미리보기/버퍼·파일 탐색 명령 모음
;;   - corfu/cape : 버퍼 내(in-buffer) 자동완성

;;; Code:

;;; Vertico — 미니버퍼 세로 완성 UI
(use-package vertico
  :ensure t
  :config
  (vertico-mode))

;;; Orderless — 유연한 매칭(공백으로 여러 패턴, 순서 무관)
(use-package orderless
  :ensure t
  :custom
  (completion-styles '(orderless basic))
  (completion-category-defaults nil)
  (completion-category-overrides '((file (styles partial-completion)))))

;;; Marginalia — 후보 주석
(use-package marginalia
  :ensure t
  :hook (after-init . marginalia-mode))

;;; Embark — 컨텍스트 액션
(use-package embark
  :ensure t
  :bind (("C-." . embark-act)
         ("C-;" . embark-dwim)
         ("C-h B" . embark-bindings))
  :init
  (setq prefix-help-command #'embark-prefix-help-command)
  :config
  (add-to-list 'display-buffer-alist
               '("\\`\\*Embark Collect \\(Live\\|Completions\\)\\*"
                 nil
                 (window-parameters (mode-line-format . none)))))

(use-package embark-consult
  :ensure t
  :hook (embark-collect-mode . consult-preview-at-point-mode))

;;; Consult — 검색/탐색/미리보기
;; 주의: minimal 기본 바인딩 중 `C-c h'(→consult-history)는 imoogi hydra-master
;; 와 충돌하므로 제외했다.
(use-package consult
  :ensure t
  :bind (("C-c M-x" . consult-mode-command)
         ("C-c k"   . consult-kmacro)
         ("C-c i"   . consult-info)
         ;; ctl-x-map
         ("C-x M-:" . consult-complex-command)
         ("C-x b"   . consult-buffer)            ; 기존 counsel-switch-buffer 대체
         ("C-x 4 b" . consult-buffer-other-window)
         ("C-x r b" . consult-bookmark)
         ("C-x p b" . consult-project-buffer)
         ;; 레지스터
         ("M-#"   . consult-register-load)
         ("M-'"   . consult-register-store)
         ("C-M-#" . consult-register)
         ("M-y"   . consult-yank-pop)
         ;; goto-map (M-g)
         ("M-g e"   . consult-compile-error)
         ("M-g f"   . consult-flymake)
         ("M-g g"   . consult-goto-line)
         ("M-g M-g" . consult-goto-line)
         ("M-g o"   . consult-outline)
         ("M-g m"   . consult-mark)
         ("M-g k"   . consult-global-mark)
         ("M-g i"   . consult-imenu)
         ("M-g I"   . consult-imenu-multi)
         ;; search-map (M-s)
         ("M-s d" . consult-find)
         ("M-s g" . consult-grep)
         ("M-s G" . consult-git-grep)
         ("M-s r" . consult-ripgrep)
         ("M-s l" . consult-line)               ; 기존 swiper 대체
         ("M-s L" . consult-line-multi)
         ("M-s k" . consult-keep-lines)
         ("M-s u" . consult-focus-lines)
         ("M-s e" . consult-isearch-history)
         :map isearch-mode-map
         ("M-e"   . consult-isearch-history)
         ("M-s e" . consult-isearch-history)
         ("M-s l" . consult-line)
         :map minibuffer-local-map
         ("M-s" . consult-history)
         ("M-r" . consult-history))
  :hook (completion-list-mode . consult-preview-at-point-mode)
  :init
  (setq register-preview-delay 0.5
        register-preview-function #'consult-register-format)
  (advice-add #'register-preview :override #'consult-register-window)
  (setq xref-show-xrefs-function #'consult-xref
        xref-show-definitions-function #'consult-xref)
  :config
  (consult-customize
   consult-theme :preview-key '(:debounce 0.2 any)
   consult-ripgrep consult-git-grep consult-grep
   consult-bookmark consult-recent-file consult-xref
   consult-source-bookmark consult-source-file-register
   consult-source-recent-file consult-source-project-recent-file
   :preview-key '(:debounce 0.4 any))
  (setq consult-narrow-key "<"))

;;; C-s 를 consult-line 으로 (기존 swiper 자리)
(global-set-key (kbd "C-s") #'consult-line)

;;; Corfu — 버퍼 내 자동완성 팝업
(use-package corfu
  :ensure t
  :hook ((prog-mode  . corfu-mode)
         (shell-mode . corfu-mode)
         (eshell-mode . corfu-mode))
  :custom
  (read-extended-command-predicate #'command-completion-default-include-p)
  (text-mode-ispell-word-completion nil)
  (tab-always-indent 'complete)
  :config
  (global-corfu-mode))

;;; Cape — completion-at-point 백엔드 확장
;; 주의: minimal 기본 `C-c p'(cape-prefix-map)는 imoogi projectile 과 충돌하므로
;; `C-c e' 로 변경했다.
(use-package cape
  :ensure t
  :bind ("C-c e" . cape-prefix-map)
  :init
  (add-hook 'completion-at-point-functions #'cape-dabbrev)
  (add-hook 'completion-at-point-functions #'cape-file)
  (add-hook 'completion-at-point-functions #'cape-elisp-block))

(provide 'imoogi-completion)
;;; 02-completion.el ends here
