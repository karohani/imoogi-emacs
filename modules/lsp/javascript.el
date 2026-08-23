;;; javascript.el --- JavaScript language server configuration -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "lsp/javascript" 'eglot 'js)

(defun imoogi-javascript-lsp-setup ()
  "Start typescript-language-server for the current JavaScript buffer."
  (imoogi-eglot-ensure-if-server-available "typescript-language-server"))

(use-package js
  :ensure nil
  :hook ((js-mode js-ts-mode) . imoogi-javascript-lsp-setup))

(imoogi-eglot-register-if-available
 '(js-mode js-ts-mode)
 '("typescript-language-server" "--stdio"))

(provide 'imoogi-lsp-javascript)
;;; javascript.el ends here
