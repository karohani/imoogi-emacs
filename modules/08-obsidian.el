;;; 08-obsidian.el --- Obsidian (vendored via package.el) -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "08-obsidian" 'obsidian)

;; 과거 straight.el 로 GitHub 에서 설치했으나, 망분리 vendoring 통일을 위해
;; MELPA 패키지(obsidian)로 이관. vendor/elpa/ 에 동봉된다.
(use-package obsidian
  :ensure t
  :config
  (global-obsidian-mode t)
  :custom
  (obsidian-directory "~/obsidian")
  (obsidian-inbox-directory "Inbox")
  (markdown-enable-wiki-links t)
  :bind (:map obsidian-mode-map
              ("C-c C-n" . obsidian-capture)
              ("C-c C-l" . obsidian-insert-link)
              ("C-c C-o" . obsidian-follow-link-at-point)
              ("C-c C-p" . obsidian-jump)
              ("C-c C-b" . obsidian-backlink-jump)))

(provide 'imoogi-obsidian)
;;; obsidian.el ends here
