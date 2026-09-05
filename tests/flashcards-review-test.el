;;; flashcards-review-test.el --- local flashcards review UI tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'imoogi-flashcards-review)

(ert-deftest imoogi-flashcards-review-kill-buffer-closes-db ()
  (let ((closed 0)
        (buf (generate-new-buffer " *imoogi-flashcards-review-test*")))
    (cl-letf (((symbol-function 'imoogi-flashcards-db-close)
               (lambda (_db) (cl-incf closed))))
      (unwind-protect
          (with-current-buffer buf
            (imoogi-flashcards-review-mode)
            (setq imoogi-flashcards-review--db :fake-db)
            (kill-buffer buf)
            (should (= closed 1)))
        (when (buffer-live-p buf)
          (kill-buffer buf))))))

(ert-deftest imoogi-flashcards-review-quit-closes-db ()
  (let ((closed 0)
        (buf (generate-new-buffer " *imoogi-flashcards-review-test*")))
    (cl-letf (((symbol-function 'imoogi-flashcards-db-close)
               (lambda (_db) (cl-incf closed)))
              ((symbol-function 'quit-window)
               (lambda (&optional _kill _window) nil)))
      (unwind-protect
          (with-current-buffer buf
            (imoogi-flashcards-review-mode)
            (setq imoogi-flashcards-review--db :fake-db)
            (imoogi-flashcards-review-quit)
            (should (= closed 1))
            (should-not imoogi-flashcards-review--db))
        (when (buffer-live-p buf)
          (kill-buffer buf))))))

(ert-deftest imoogi-flashcards-review-start-closes-db-when-rendering-fails ()
  (let ((closed 0)
        (buf (get-buffer-create "*imoogi flashcards*")))
    (cl-letf (((symbol-function 'imoogi-flashcards-db-open)
               (lambda (&optional _file) :fake-db))
              ((symbol-function 'imoogi-flashcards-repo-due-cards)
               (lambda (&rest _args) '(("missing" nil nil nil "missing.org"))))
              ((symbol-function 'imoogi-flashcards-resolve-card)
               (lambda (&rest _args) (error "missing source")))
              ((symbol-function 'imoogi-flashcards-db-close)
               (lambda (_db) (cl-incf closed))))
      (unwind-protect
          (progn
            (should-error (imoogi-flashcards-review-start))
            (should (= closed 1)))
        (when (buffer-live-p buf)
          (kill-buffer buf))))))

(provide 'flashcards-review-test)
;;; flashcards-review-test.el ends here
