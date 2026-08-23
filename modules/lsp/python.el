;;; python.el --- Python language server configuration -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "lsp/python" 'eglot 'python)

(defconst imoogi-python-lsp-server-candidates
  '("basedpyright-langserver"
    "pyright-langserver"
    "pyrefly"
    "pylsp"
    "jedi-language-server"
    ("ruff" "server"))
  "Python language server commands supported by built-in Eglot.")

(defun imoogi-python-lsp-setup ()
  "Start Eglot when a supported Python language server is available."
  (apply #'imoogi-eglot-ensure-if-any-server-available
         imoogi-python-lsp-server-candidates))

(use-package python
  :ensure nil
  :hook ((python-mode python-ts-mode) . imoogi-python-lsp-setup))

(provide 'imoogi-lsp-python)
;;; python.el ends here
