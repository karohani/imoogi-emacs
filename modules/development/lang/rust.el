;;; rust.el --- Rust language server configuration -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "lsp/rust" 'eglot 'rust-mode)

(defun imoogi-rust-lsp-setup ()
  "Start rust-analyzer for the current Rust buffer when available."
  (imoogi-eglot-ensure-if-server-available "rust-analyzer"))

(use-package rust-mode
  :ensure t
  :hook ((rust-mode rust-ts-mode) . imoogi-rust-lsp-setup))

(imoogi-eglot-register-if-available
 '(rust-mode rust-ts-mode)
 '("rust-analyzer"))

(provide 'imoogi-lsp-rust)
;;; rust.el ends here
