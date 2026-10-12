;;; java.el --- Java language server configuration -*- lexical-binding: t; -*-

;;; Code:
(declare-function imoogi-jvm-ensure-eglot "jvm")
(imoogi-require "lsp/java" 'cc-mode 'eglot)

(defun imoogi-java-lsp-setup ()
  "Start jdtls for the current Java buffer when it and Java are available.
The Java gate lives in jvm.el; it is called lazily, never at load time."
  (imoogi-jvm-ensure-eglot "jdtls"))

(use-package cc-mode
  :ensure nil
  :hook ((java-mode java-ts-mode) . imoogi-java-lsp-setup))

(imoogi-eglot-register-if-available '(java-mode java-ts-mode) '("jdtls"))

(provide 'imoogi-lsp-java)
;;; java.el ends here
