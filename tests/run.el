;;; run.el --- imoogi-emacs test runner -*- lexical-binding: t; -*-

;;; Code:

(setq load-prefer-newer t)

(defvar imoogi-test-root
  (file-name-directory (directory-file-name
                        (file-name-directory (or load-file-name buffer-file-name)))))

(setq user-emacs-directory
      (file-name-as-directory
       (or (getenv "IMOOGI_TEST_USER_DIR")
           (expand-file-name "imoogi-emacs-test-user-emacs.d" temporary-file-directory))))
(make-directory user-emacs-directory t)

(setq package-archives nil)

(defvar compile-angel-excluded-path-regexps nil)
(let ((test-root-regexp (concat "^" (regexp-quote (file-name-as-directory
                                                    (expand-file-name imoogi-test-root))))))
  (add-to-list 'compile-angel-excluded-path-regexps test-root-regexp))

(load (expand-file-name "tests/boot-health.el" imoogi-test-root) nil t)
(declare-function imoogi-boot-health-start-capture "boot-health")
(declare-function imoogi-boot-health-assert "boot-health")
(declare-function compile-angel-exclude-directory "compile-angel")
(imoogi-boot-health-start-capture)
(load (expand-file-name "boot.el" imoogi-test-root))
(imoogi-boot-health-assert t)

(when (fboundp 'compile-angel-exclude-directory)
  (compile-angel-exclude-directory imoogi-test-root))
(when (fboundp 'compile-angel-on-load-mode)
  (compile-angel-on-load-mode -1))

(require 'ert)

(dolist (test-file (directory-files (expand-file-name "tests" imoogi-test-root)
                                    t "-test\\.el\\'"))
  (load test-file nil t))

(ert-run-tests-batch-and-exit)

;;; run.el ends here
