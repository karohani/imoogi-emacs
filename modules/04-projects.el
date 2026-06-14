;;; projects.el --- Projectile, Perspective -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "04-projects" 'projectile 'perspective 'persp-projectile)

(use-package projectile
  :ensure t
  :config
  (projectile-mode 1)
  (setq projectile-project-search-path '("~/workspace/")
        projectile-switch-project-action #'projectile-dired
        ;; ivy 제거 후 기본 completing-read(vertico) 사용
        projectile-completion-system 'default)
  :bind-keymap
  ("C-c p" . projectile-command-map))

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
