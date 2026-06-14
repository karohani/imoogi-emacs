;;; projects.el --- Projectile, Perspective -*- lexical-binding: t; -*-

(use-package projectile
  :ensure t
  :config
  (projectile-mode 1)
  (setq projectile-project-search-path '("~/workspace/")
        projectile-switch-project-action #'projectile-dired)
  :bind-keymap
  ("C-c p" . projectile-command-map))

(use-package counsel-projectile
  :ensure t
  :after (counsel projectile)
  :config
  (counsel-projectile-mode 1))

;;; Perspective
(use-package perspective
  :ensure t
  :custom
  (persp-mode-prefix-key (kbd "C-x x"))
  :config
  (persp-mode 1))

(use-package persp-projectile
  :ensure t
  :after (perspective projectile))

(provide 'imoogi-projects)
;;; projects.el ends here
