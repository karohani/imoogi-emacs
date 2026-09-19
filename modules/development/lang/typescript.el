;;; typescript.el --- TypeScript and TSX language server configuration -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "lsp/typescript" 'eglot 'typescript-mode 'web-mode)

(defun imoogi-typescript-lsp-setup ()
  "Start typescript-language-server for the current TypeScript buffer."
  (imoogi-eglot-ensure-if-server-available "typescript-language-server"))

(defun imoogi-web-lsp-setup ()
  "Start TypeScript LSP for TSX or JSX files opened with web-mode."
  (when (and buffer-file-name
             (string-match-p "\\.[tj]sx\\'" buffer-file-name))
    (imoogi-typescript-lsp-setup)))

(use-package typescript-mode
  :ensure t
  :hook ((typescript-mode typescript-ts-mode tsx-ts-mode)
         . imoogi-typescript-lsp-setup))

(use-package web-mode
  :ensure t
  :hook (web-mode . imoogi-web-lsp-setup))

(imoogi-eglot-register-if-available
 '(typescript-mode typescript-ts-mode)
 '("typescript-language-server" "--stdio"))
(imoogi-eglot-register-if-available
 '(web-mode tsx-ts-mode)
 '("typescript-language-server" "--stdio"))

(provide 'imoogi-lsp-typescript)
;;; typescript.el ends here
