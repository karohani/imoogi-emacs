;;; kotlin.el --- Kotlin language server configuration -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "lsp/kotlin" 'eglot 'kotlin-mode)

(defun imoogi-kotlin-lsp-setup ()
  "Start kotlin-language-server for the current Kotlin buffer when available."
  (imoogi-eglot-ensure-if-server-available "kotlin-language-server"))

(use-package kotlin-mode
  :ensure t
  :hook (kotlin-mode . imoogi-kotlin-lsp-setup))

(imoogi-eglot-register-if-available 'kotlin-mode '("kotlin-language-server"))

(provide 'imoogi-lsp-kotlin)
;;; kotlin.el ends here
