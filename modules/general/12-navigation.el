;;; 12-navigation.el --- 탐색/도움말 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; avy(점프), helpful(향상된 도움말),
;; bufferfile(파일 이름변경/삭제).

;;; Code:

(imoogi-require "12-navigation" 'avy 'helpful 'bufferfile)

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

;;; bufferfile — 현재 버퍼의 파일을 안전하게 이름변경/복사/삭제
(use-package bufferfile
  :ensure t
  :custom
  (bufferfile-verbose nil)
  (bufferfile-use-vc nil)
  (bufferfile-delete-switch-to 'parent-directory))

(provide 'imoogi-navigation)
;;; 12-navigation.el ends here
