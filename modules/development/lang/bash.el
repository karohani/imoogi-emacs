;;; bash.el --- Bash language server configuration -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "lsp/bash" 'eglot 'sh-script)

(defun imoogi-bash-lsp-setup ()
  "Start bash-language-server for the current shell buffer when available."
  (imoogi-eglot-ensure-if-server-available "bash-language-server"))

(use-package sh-script
  :ensure nil
  :hook ((sh-mode bash-ts-mode) . imoogi-bash-lsp-setup))

(imoogi-eglot-register-if-available
 '(sh-mode bash-ts-mode)
 '("bash-language-server" "start"))

(provide 'imoogi-lsp-bash)
;;; bash.el ends here
