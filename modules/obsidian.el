;;; obsidian.el --- Obsidian.el via straight -*- lexical-binding: t; -*-

(use-package obsidian
  :straight (:host github :repo "licht1stein/obsidian.el" :files ("*.el"))
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
