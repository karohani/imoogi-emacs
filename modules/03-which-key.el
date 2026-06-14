;;; which-key.el --- Which Key -*- lexical-binding: t; -*-

;; which-key 는 Emacs 30 부터 내장(별도 설치 불필요).
(use-package which-key
  :ensure nil
  :config
  (which-key-mode 1)
  (setq which-key-idle-delay 0.5))

(provide 'imoogi-which-key)
;;; which-key.el ends here
