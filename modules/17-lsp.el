;;; 17-lsp.el --- Eglot LSP 및 xref 공통 설정 -*- lexical-binding: t; -*-

;; 공통 LSP 수명주기와 xref 진입점만 관리한다. 언어별 서버 연결과
;; workspace 설정은 modules/lsp/*.el 에서 독립적으로 로드한다.

;;; Code:

(imoogi-require "17-lsp" 'eglot 'flymake 'xref 'seq)

(defvar eglot-server-programs)

(defvar imoogi-lsp-language-config-dir
  (expand-file-name "modules/lsp/" imoogi-emacs-dir)
  "Directory containing language-specific LSP configurations.")

(defun imoogi-lsp-local-bin-dir ()
  "Return active repository-local toolchain bin directory, or nil.

The activation path must be the relative .local/bin symlink created by the
toolchain setup command.  Missing, broken, absolute, or non-toolchain links are
ignored so Emacs startup stays side-effect free."
  (let* ((local-root (expand-file-name ".local/" imoogi-emacs-dir))
         (link (expand-file-name "bin" local-root))
         (target (file-symlink-p link)))
    (when (and target
               (not (file-name-absolute-p target))
               (string-prefix-p "toolchains/" target)
               (file-directory-p link))
      (let ((toolchains-root (file-truename
                              (expand-file-name "toolchains/" local-root)))
            (bin-root (file-truename link)))
        (when (file-in-directory-p bin-root toolchains-root)
          (file-name-as-directory link))))))

(defun imoogi-lsp-prepend-local-bin ()
  "Prepend active repository-local toolchain bin to process lookup paths."
  (when-let ((local-bin (imoogi-lsp-local-bin-dir)))
    (setq exec-path
          (cons local-bin
                (seq-remove (lambda (entry)
                              (imoogi-lsp-same-directory-p entry local-bin))
                            exec-path)))
    (let* ((current-path (getenv "PATH"))
           (entries (if current-path
                        (split-string current-path path-separator t)
                      nil)))
      (setenv "PATH"
              (mapconcat #'identity
                         (cons local-bin
                               (seq-remove
                                (lambda (entry)
                                  (imoogi-lsp-same-directory-p entry local-bin))
                                entries))
                         path-separator)))))

(defun imoogi-lsp-same-directory-p (a b)
  "Return non-nil when A and B name the same directory string."
  (and (stringp a)
       (stringp b)
       (string= (directory-file-name a) (directory-file-name b))))

(defun imoogi-eglot-server-available-p (server)
  "Return non-nil when SERVER command is available locally."
  (let ((command (if (listp server) (car server) server)))
    (and (stringp command) (executable-find command))))

(defun imoogi-eglot-register-if-available (modes server)
  "Register SERVER for MODES when its executable exists."
  (when (imoogi-eglot-server-available-p server)
    (with-eval-after-load 'eglot
      (add-to-list 'eglot-server-programs (cons modes server)))))

(defun imoogi-eglot-ensure-if-server-available (server)
  "Start Eglot only when SERVER executable exists locally."
  (when (and (imoogi-eglot-server-available-p server)
             (require 'eglot nil t))
    (eglot-ensure)))

(defun imoogi-eglot-ensure-if-any-server-available (&rest servers)
  "Start Eglot when any command in SERVERS is available locally."
  (when (and (seq-some #'imoogi-eglot-server-available-p servers)
             (require 'eglot nil t))
    (eglot-ensure)))

(use-package eglot
  :ensure nil
  :commands (eglot eglot-ensure eglot-rename eglot-code-actions
                   eglot-code-action-organize-imports eglot-shutdown)
  :custom
  (eglot-autoshutdown t)
  (eglot-sync-connect 0)
  (eglot-extend-to-xref t)
  (eglot-events-buffer-config '(:size 0 :format short)))

(use-package flymake
  :ensure nil
  :custom
  (flymake-show-diagnostics-at-end-of-line nil)
  (flymake-wrap-around nil))

(defvar-keymap imoogi-lsp-map
  :doc "LSP와 xref 코드 탐색 명령의 공통 접두 맵."
  "d" #'xref-find-definitions
  "r" #'xref-find-references
  "a" #'xref-find-apropos
  "b" #'xref-go-back
  "f" #'xref-go-forward
  "e" #'eglot
  "R" #'eglot-rename
  "c" #'eglot-code-actions
  "o" #'eglot-code-action-organize-imports
  "q" #'eglot-shutdown)

(use-package xref
  :ensure nil
  :config
  (global-set-key (kbd "C-c l") imoogi-lsp-map))

(with-eval-after-load 'which-key
  (which-key-add-key-based-replacements "C-c l" "LSP/xref"))

(imoogi-lsp-prepend-local-bin)

(defun imoogi-lsp-load-language-configurations ()
  "Load every language-specific LSP configuration with failure isolation."
  (when (file-directory-p imoogi-lsp-language-config-dir)
    (dolist (file (directory-files imoogi-lsp-language-config-dir t
                                   "\\`[[:alnum:]-]+\\.el\\'"))
      (condition-case err
          (load (file-name-sans-extension file))
        (error
         (display-warning
          'imoogi-lsp
          (format "LSP 언어 설정 %s 로딩 실패(건너뜀): %s"
                  (file-name-base file) (error-message-string err))
          :error))))))

(imoogi-lsp-load-language-configurations)

(provide 'imoogi-lsp)
;;; 17-lsp.el ends here
