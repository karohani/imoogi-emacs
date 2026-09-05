;;; flashcards-boot-test.el --- local flashcards boot degradation tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'sqlite)

(ert-deftest imoogi-flashcards-module-errors-when-sqlite-runtime-is-absent ()
  "boot.el catches this error and skips only 25-flashcards."
  (let* ((root (if (boundp 'imoogi-test-root)
                   imoogi-test-root
                 (or (and (or load-file-name buffer-file-name)
                          (file-name-directory
                           (directory-file-name
                            (file-name-directory
                             (or load-file-name buffer-file-name)))))
                     default-directory)))
         (file (expand-file-name "modules/25-flashcards.el" root)))
    (cl-letf (((symbol-function 'sqlite-available-p) (lambda () nil)))
      (should-error (load file nil t)))))

(ert-deftest imoogi-flashcards-boot-skips-only-flashcards-when-sqlite-runtime-is-absent ()
  "A missing SQLite runtime must not disable the existing Anki module."
  (let* ((root (if (boundp 'imoogi-test-root)
                   imoogi-test-root
                 (or (and (or load-file-name buffer-file-name)
                          (file-name-directory
                           (directory-file-name
                            (file-name-directory
                             (or load-file-name buffer-file-name)))))
                     default-directory)))
         (boot (expand-file-name "boot.el" root))
         (tmp (make-temp-file "imoogi-flashcards-boot" t)))
    (unwind-protect
        (let ((user-emacs-directory (file-name-as-directory tmp)))
          (cl-letf (((symbol-function 'sqlite-available-p) (lambda () nil)))
            (load boot nil t))
          (should (member "25-flashcards" imoogi-failed-modules))
          (should (featurep 'imoogi-anki)))
      (delete-directory tmp t))))

(provide 'flashcards-boot-test)
;;; flashcards-boot-test.el ends here
