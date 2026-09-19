;;; clojure.el --- Clojure language server configuration -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "lsp/clojure" 'clojure-mode 'eglot)

(defun imoogi-clojure-lsp-setup ()
  "Start clojure-lsp for the current Clojure buffer when available."
  (imoogi-eglot-ensure-if-server-available "clojure-lsp"))

(use-package clojure-mode
  :ensure t
  :hook ((clojure-mode clojurec-mode clojurescript-mode)
         . imoogi-clojure-lsp-setup))

(imoogi-eglot-register-if-available
 '(clojure-mode clojurec-mode clojurescript-mode)
 '("clojure-lsp"))

(provide 'imoogi-lsp-clojure)
;;; clojure.el ends here
