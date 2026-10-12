;;; kotlin.el --- Kotlin language server configuration -*- lexical-binding: t; -*-

;;; Code:
(declare-function imoogi-jvm-ensure-eglot "jvm")
(imoogi-require "lsp/kotlin" 'eglot 'kotlin-mode)

(defun imoogi-kotlin-lsp-setup ()
  "Start kotlin-language-server when it and Java are available.
The Java gate lives in jvm.el; it is called lazily, never at load time."
  (imoogi-jvm-ensure-eglot "kotlin-language-server"))

(use-package kotlin-mode
  :ensure t
  :hook (kotlin-mode . imoogi-kotlin-lsp-setup))

(imoogi-eglot-register-if-available 'kotlin-mode '("kotlin-language-server"))

(provide 'imoogi-lsp-kotlin)
;;; kotlin.el ends here
