;;; compiled-menu-test.el --- compiled transient menu reload regressions -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)

(defconst imoogi-compiled-menu-test--root
  (file-name-directory (directory-file-name
                        (file-name-directory (or load-file-name buffer-file-name)))))

(defconst imoogi-compiled-menu-test--emacs
  (or (getenv "EMACS")
      (expand-file-name invocation-name invocation-directory))
  "Emacs binary used for clean subprocess compile/load checks.")

(defun imoogi-compiled-menu-test--run-script (script)
  "Run SCRIPT in a clean Emacs subprocess and return (STATUS . OUTPUT)."
  (let ((file (make-temp-file "imoogi-compiled-menu-script-" nil ".el"))
        (output (generate-new-buffer " *imoogi compiled menu test*")))
    (unwind-protect
        (progn
          (with-temp-file file (insert script))
          (cons (call-process imoogi-compiled-menu-test--emacs
                              nil output nil "--batch" "-Q" "-l" file)
                (with-current-buffer output (buffer-string))))
      (ignore-errors (delete-file file))
      (kill-buffer output))))

(defun imoogi-compiled-menu-test--common-setup-form (tmpdir)
  "Return setup form shared by clean subprocesses using TMPDIR."
  `(progn
     (setq debug-on-error t
           package-enable-at-startup nil
           package-archives nil
           user-emacs-directory ,(expand-file-name "user-emacs.d/" tmpdir))
     (require 'package)
     (setq package-user-dir
           ,(expand-file-name "vendor/elpa/" imoogi-compiled-menu-test--root))
     (package-initialize)
     (require 'seq)
     (require 'use-package)
     (setq use-package-always-ensure nil)
     (defvar imoogi-emacs-dir
       ,(file-name-as-directory (expand-file-name imoogi-compiled-menu-test--root)))
     (add-to-list 'load-path (expand-file-name "modules/anki/" imoogi-emacs-dir))
     (add-to-list 'load-path (expand-file-name "modules/flashcards/" imoogi-emacs-dir))
     (defun imoogi-require (module &rest packages)
       (let ((missing (seq-remove
                       (lambda (p) (locate-library (symbol-name p)))
                       packages)))
         (when missing (error "[%s] missing packages %S" module missing))))))

(ert-deftest imoogi-compiled-anki-and-flashcards-menus-survive-clean-elc-load ()
  "Compiled Org, Anki and Flashcards modules register their deferred menus."
  (let ((tmpdir (make-temp-file "imoogi-compiled-menu-" t)))
    (unwind-protect
        (let* ((compile-script
                (format "%S"
                        `(progn
                           ,(imoogi-compiled-menu-test--common-setup-form tmpdir)
                           (dolist (module '("14-org" "24-anki" "25-flashcards"))
                             (let* ((src (expand-file-name
                                          (concat "modules/" module ".el")
                                          imoogi-emacs-dir))
                                    (dst (expand-file-name
                                          (concat module ".el") ,tmpdir))
                                    (elc (byte-compile-dest-file dst)))
                               (copy-file src dst t)
                               (unless (byte-compile-file dst)
                                 (error "byte compile failed for %s" module))
                               (unless (file-exists-p elc)
                                 (error "missing %s" elc)))))))
               (load-script
                (format "%S"
                        `(progn
                           ,(imoogi-compiled-menu-test--common-setup-form tmpdir)
                           (load ,(expand-file-name "modules/05-transient.el"
                                                    imoogi-compiled-menu-test--root)
                                 nil t)
                           (load ,(expand-file-name "14-org.elc" tmpdir) nil nil t)
                           (load ,(expand-file-name "24-anki.elc" tmpdir) nil nil t)
                           (load ,(expand-file-name "25-flashcards.elc" tmpdir) nil nil t)
                           (dolist (row '((imoogi-org-agenda-transient "o")
                                          (imoogi-anki-transient "a")
                                          (imoogi-flashcards-transient "f")))
                             (unless (fboundp (car row))
                               (error "%S is not defined" (car row)))
                             (unless (eq (plist-get
                                          (cdr (transient-get-suffix
                                                'imoogi-transient-master
                                                (cadr row)))
                                          :command)
                                         (car row))
                               (error "%S is not registered" (car row)))))))
               (compiled (imoogi-compiled-menu-test--run-script compile-script))
               (loaded (and (equal (car compiled) 0)
                            (imoogi-compiled-menu-test--run-script load-script))))
          (unless (equal (car compiled) 0)
            (ert-fail (format "compile subprocess failed:\n%s" (cdr compiled))))
          (unless (equal (car loaded) 0)
            (ert-fail (format "load subprocess failed:\n%s" (cdr loaded)))))
      (delete-directory tmpdir t))))

(provide 'compiled-menu-test)
;;; compiled-menu-test.el ends here
