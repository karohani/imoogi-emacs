;;; completion.el --- Ivy, Counsel, Swiper -*- lexical-binding: t; -*-

(use-package ivy
  :ensure t
  :config
  (ivy-mode 1)
  (setq ivy-use-virtual-buffers t
        ivy-count-format "(%d/%d) "
        ivy-wrap t))

(use-package counsel
  :ensure t
  :after ivy
  :config
  (counsel-mode 1)
  :bind
  (("M-x"     . counsel-M-x)
   ("C-x C-f" . counsel-find-file)
   ("C-x b"   . counsel-switch-buffer)))

(use-package swiper
  :ensure t
  :after ivy
  :bind
  (("C-s" . swiper)))

(provide 'imoogi-completion)
;;; completion.el ends here
