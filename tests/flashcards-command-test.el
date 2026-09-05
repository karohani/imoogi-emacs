;;; flashcards-command-test.el --- local flashcards command tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'sqlite)
(require 'org)
(require 'imoogi-flashcards)

(defmacro imoogi-flashcards-command-test--with-root (&rest body)
  "Run BODY with ROOT and DB-FILE bound to temporary paths."
  `(let ((root (make-temp-file "imoogi-flashcards" t))
         (db-file (make-temp-name
                   (expand-file-name "imoogi-flashcards.sqlite"
                                     temporary-file-directory))))
     (unwind-protect
         (let ((imoogi-flashcards-database-file db-file))
           ,@body)
       (when (file-exists-p db-file)
         (delete-file db-file))
       (delete-directory root t))))

(defun imoogi-flashcards-command-test--write (path content)
  "Write CONTENT to PATH."
  (make-directory (file-name-directory path) t)
  (with-temp-file path (insert content)))

(ert-deftest imoogi-flashcards-sync-buffer-rejects-duplicate-ids-before-db-write ()
  (if (not (sqlite-available-p))
      (ert-skip "sqlite support is not available in this Emacs")
    (imoogi-flashcards-command-test--with-root
     (let ((file (expand-file-name "cards.org" root)))
       (imoogi-flashcards-command-test--write
        file
        (concat "* A\n:PROPERTIES:\n:IMOOGI_FLASHCARD_ID: dup\n"
                ":IMOOGI_FLASHCARD_KIND: Basic\n:END:\nA\n"
                "* B\n:PROPERTIES:\n:IMOOGI_FLASHCARD_ID: dup\n"
                ":IMOOGI_FLASHCARD_KIND: Basic\n:END:\nB\n"))
       (let ((buf (find-file-noselect file)))
         (unwind-protect
             (with-current-buffer buf
               (should-error (imoogi-flashcards-sync-buffer)
                             :type 'user-error)
               (should-not (file-exists-p db-file)))
           (kill-buffer buf)))))))

(ert-deftest imoogi-flashcards-sync-root-duplicate-conflict-leaves-db-unchanged ()
  (if (not (sqlite-available-p))
      (ert-skip "sqlite support is not available in this Emacs")
    (imoogi-flashcards-command-test--with-root
     (let ((first (expand-file-name "first.org" root))
           (dup-dir (expand-file-name "dups" root)))
       (imoogi-flashcards-command-test--write
        first
        (concat "* A\n:PROPERTIES:\n:IMOOGI_FLASHCARD_ID: stable\n"
                ":IMOOGI_FLASHCARD_KIND: Basic\n:END:\nA\n"))
       (imoogi-flashcards-sync-root root)
       (imoogi-flashcards-command-test--write
        (expand-file-name "a.org" dup-dir)
        (concat "* Dup A\n:PROPERTIES:\n:IMOOGI_FLASHCARD_ID: dup\n"
                ":IMOOGI_FLASHCARD_KIND: Basic\n:END:\nA\n"))
       (imoogi-flashcards-command-test--write
        (expand-file-name "b.org" dup-dir)
        (concat "* Dup B\n:PROPERTIES:\n:IMOOGI_FLASHCARD_ID: dup\n"
                ":IMOOGI_FLASHCARD_KIND: Basic\n:END:\nB\n"))
       (let ((err (should-error (imoogi-flashcards-sync-root root)
                                :type 'user-error)))
         (should (string-match-p "dup" (error-message-string err))))
       (let ((db (sqlite-open db-file)))
         (unwind-protect
             (progn
               (imoogi-flashcards-db-init db)
               (should (= (imoogi-flashcards-repo-card-count db) 1))
               (should (= (caar (sqlite-select
                                 db
                                 "SELECT count(*) FROM cards WHERE key = 'stable' AND tombstoned_at IS NULL"))
                          1)))
           (sqlite-close db)))))))

(provide 'flashcards-command-test)
;;; flashcards-command-test.el ends here
