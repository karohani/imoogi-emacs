;;; java.el --- Java language server configuration -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "lsp/java" 'cc-mode 'eglot)

(defun imoogi-java-lsp-setup ()
  "Start jdtls for the current Java buffer when available."
  (imoogi-eglot-ensure-if-server-available "jdtls"))

(use-package cc-mode
  :ensure nil
  :hook ((java-mode java-ts-mode) . imoogi-java-lsp-setup))

(imoogi-eglot-register-if-available '(java-mode java-ts-mode) '("jdtls"))

(provide 'imoogi-lsp-java)
;;; java.el ends here
