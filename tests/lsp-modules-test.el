;;; lsp-modules-test.el --- LSP module regression tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'eglot)

(defun imoogi-test-lsp--write-executable (path)
  "Create a minimal executable file at PATH."
  (make-directory (file-name-directory path) t)
  (with-temp-file path
    (insert "#!/bin/sh\nexit 0\n"))
  (set-file-modes path #o755))

(defun imoogi-test-lsp--activated-local-bin (root bundle)
  "Create an active relative .local/bin symlink under ROOT for BUNDLE."
  (let* ((bin (expand-file-name (format ".local/toolchains/%s/bin/" bundle) root))
         (link (expand-file-name ".local/bin" root)))
    (make-directory bin t)
    (make-symbolic-link (format "toolchains/%s/bin" bundle) link)
    bin))

(ert-deftest imoogi-lsp-prepends-active-local-bin-first ()
  (let* ((root (make-temp-file "imoogi-lsp-root-" t))
         (imoogi-emacs-dir (file-name-as-directory root))
         (local-bin (file-name-as-directory
                     (expand-file-name ".local/bin" root)))
         (alternate-local-bin (directory-file-name local-bin))
         (exec-path (list alternate-local-bin "/usr/bin" "/bin"))
         (process-environment
          (list (concat "PATH=" alternate-local-bin ":/usr/bin:/bin"))))
    (imoogi-test-lsp--activated-local-bin root "2026.08.22.1")
    (imoogi-lsp-prepend-local-bin)
    (should (equal (car exec-path) local-bin))
    (should (string-prefix-p local-bin (getenv "PATH")))
    (imoogi-lsp-prepend-local-bin)
    (should (= 1 (cl-count local-bin exec-path :test #'string=)))
    (should-not (member alternate-local-bin exec-path))
    (let ((path-entries (split-string (getenv "PATH") path-separator t)))
      (should (= 1 (cl-count local-bin path-entries :test #'string=)))
      (should-not (member alternate-local-bin path-entries)))))

(ert-deftest imoogi-lsp-ignores-missing-broken-or-absolute-local-bin ()
  (dolist (case '(missing broken absolute))
    (let* ((root (make-temp-file "imoogi-lsp-root-" t))
           (imoogi-emacs-dir (file-name-as-directory root))
           (exec-path '("/usr/bin" "/bin"))
           (process-environment '("PATH=/usr/bin:/bin")))
      (make-directory (expand-file-name ".local" root) t)
      (pcase case
        ('broken
         (make-symbolic-link "toolchains/missing/bin"
                             (expand-file-name ".local/bin" root)))
        ('absolute
         (let ((outside (make-temp-file "imoogi-lsp-outside-" t)))
           (make-directory (expand-file-name "bin" outside) t)
           (make-symbolic-link (expand-file-name "bin" outside)
                               (expand-file-name ".local/bin" root)))))
      (imoogi-lsp-prepend-local-bin)
      (should (equal exec-path '("/usr/bin" "/bin")))
      (should (equal (getenv "PATH") "/usr/bin:/bin")))))

(ert-deftest imoogi-lsp-language-registration-sees-local-toolchain-bin ()
  (let* ((root (make-temp-file "imoogi-lsp-root-" t))
         (imoogi-emacs-dir (file-name-as-directory root))
         (local-bin (imoogi-test-lsp--activated-local-bin root "2026.08.22.1"))
         (exec-path '("/usr/bin" "/bin"))
         (process-environment '("PATH=/usr/bin:/bin"))
         (eglot-server-programs nil))
    (imoogi-test-lsp--write-executable (expand-file-name "gopls" local-bin))
    (imoogi-test-lsp--write-executable
     (expand-file-name "typescript-language-server" local-bin))
    (imoogi-lsp-prepend-local-bin)
    (should (string= (file-truename (expand-file-name "gopls" local-bin))
                     (file-truename (executable-find "gopls"))))
    (should (string= (file-truename
                      (expand-file-name "typescript-language-server" local-bin))
                     (file-truename
                      (executable-find "typescript-language-server"))))
    (load (expand-file-name "modules/lsp/go.el" imoogi-test-root) nil t)
    (load (expand-file-name "modules/lsp/typescript.el" imoogi-test-root) nil t)
    (should (assoc '(go-mode go-ts-mode) eglot-server-programs))
    (should (assoc '(typescript-mode typescript-ts-mode) eglot-server-programs))))

(ert-deftest imoogi-lsp-module-source-stays-side-effect-free ()
  (let ((source (with-temp-buffer
                  (insert-file-contents
                   (expand-file-name "modules/17-lsp.el" imoogi-test-root))
                  (buffer-string))))
    (dolist (forbidden '("call-process"
                         "start-process"
                         "make-process"
                         "shell-command"
                         "async-shell-command"
                         "process-file"
                         "start-file-process"
                         "url-retrieve"
                         "url-retrieve-synchronously"
                         "package-refresh-contents"
                         "package-install"
                         "treesit-install-language-grammar"
                         "imoogi-toolchain"))
      (should-not (string-match-p (regexp-quote forbidden) source)))))

(ert-deftest imoogi-lsp-keeps-consult-xref-and-common-bindings ()
  (should (eq xref-show-definitions-function #'consult-xref))
  (should (eq xref-show-xrefs-function #'consult-xref))
  (dolist (binding '(("M-." . xref-find-definitions)
                     ("M-?" . xref-find-references)
                     ("M-," . xref-go-back)
                     ("C-M-," . xref-go-forward)
                     ("C-c l d" . xref-find-definitions)
                     ("C-c l r" . xref-find-references)
                     ("C-c l b" . xref-go-back)
                     ("C-c l f" . xref-go-forward)
                     ("C-c l R" . eglot-rename)))
    (should (eq (key-binding (kbd (car binding))) (cdr binding)))))

(defun imoogi-test-lsp--mode-starts-eglot-p (mode file)
  "Return non-nil when MODE for FILE starts Eglot with a server available."
  (let (started)
    (cl-letf (((symbol-function 'imoogi-eglot-server-available-p)
               (lambda (_server) t))
              ((symbol-function 'eglot-ensure)
               (lambda () (setq started t))))
      (with-temp-buffer
        (setq buffer-file-name file)
        (funcall mode)))
    started))

(ert-deftest imoogi-lsp-language-hooks-start-when-server-is-available ()
  (dolist (case '((sh-mode . "/tmp/imoogi-lsp-test.sh")
                  (js-mode . "/tmp/imoogi-lsp-test.js")
                  (go-mode . "/tmp/imoogi-lsp-test.go")
                  (python-mode . "/tmp/imoogi-lsp-test.py")
                  (rust-mode . "/tmp/imoogi-lsp-test.rs")
                  (clojure-mode . "/tmp/imoogi-lsp-test.clj")
                  (java-mode . "/tmp/ImoogiLspTest.java")
                  (kotlin-mode . "/tmp/imoogi-lsp-test.kt")
                  (typescript-mode . "/tmp/imoogi-lsp-test.ts")
                  (web-mode . "/tmp/imoogi-lsp-test.tsx")))
    (should (imoogi-test-lsp--mode-starts-eglot-p (car case) (cdr case)))))

(ert-deftest imoogi-go-lsp-keeps-buffer-local-workspace-settings ()
  (cl-letf (((symbol-function 'imoogi-eglot-server-available-p)
             (lambda (_server) nil)))
    (with-temp-buffer
      (setq buffer-file-name "/tmp/imoogi-lsp-test.go")
      (go-mode)
      (should (equal eglot-workspace-configuration
                     '(:gopls (:usePlaceholders t)))))))

(provide 'imoogi-lsp-modules-test)
;;; lsp-modules-test.el ends here
