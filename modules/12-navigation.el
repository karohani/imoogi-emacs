;;; 12-navigation.el --- 탐색/도움말/Git 표시 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; avy(점프), helpful(향상된 도움말), diff-hl(여백 Git 변경 표시),
;; bufferfile(파일 이름변경/삭제).

;;; Code:

;;; avy — 화면 내 빠른 점프
(use-package avy
  :ensure t
  :init
  (global-set-key (kbd "C-'") 'avy-goto-char-2))

;;; helpful — 더 풍부한 *help* 버퍼
(use-package helpful
  :ensure t
  :bind
  ([remap describe-command]  . helpful-command)
  ([remap describe-function] . helpful-callable)
  ([remap describe-key]      . helpful-key)
  ([remap describe-symbol]   . helpful-symbol)
  ([remap describe-variable] . helpful-variable)
  :custom
  (helpful-max-buffers 7))

;;; diff-hl — 버퍼 여백에 커밋되지 않은 Git 변경 표시
(use-package diff-hl
  :ensure t
  :hook ((prog-mode . diff-hl-mode)
         (dired-mode . diff-hl-dired-mode)
         (magit-pre-refresh . diff-hl-magit-pre-refresh)
         (magit-post-refresh . diff-hl-magit-post-refresh))
  :init
  (setq diff-hl-flydiff-delay 0.4
        diff-hl-show-staged-changes nil
        diff-hl-update-async t
        diff-hl-global-modes '(not pdf-view-mode image-mode)))

;;; bufferfile — 현재 버퍼의 파일을 안전하게 이름변경/복사/삭제
(use-package bufferfile
  :ensure t
  :custom
  (bufferfile-verbose nil)
  (bufferfile-use-vc nil)
  (bufferfile-delete-switch-to 'parent-directory))

(provide 'imoogi-navigation)
;;; 12-navigation.el ends here
