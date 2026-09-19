;;; go.el --- Go gopls configuration -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "lsp/go" 'eglot 'go-mode)

(defun imoogi-go-lsp-setup ()
  "Configure and start gopls for the current Go buffer when available."
  (setq-local eglot-workspace-configuration
              '(:gopls (:usePlaceholders t)))
  (imoogi-eglot-ensure-if-server-available "gopls"))

(use-package go-mode
  :ensure t
  :hook ((go-mode go-ts-mode) . imoogi-go-lsp-setup))

(imoogi-eglot-register-if-available '(go-mode go-ts-mode) '("gopls"))

(provide 'imoogi-lsp-go)
;;; go.el ends here
