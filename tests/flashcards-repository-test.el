;;; flashcards-repository-test.el --- local flashcards SQLite tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'sqlite)
(require 'imoogi-flashcards-core)
(require 'imoogi-flashcards-repository)

(defmacro imoogi-flashcards-test--with-db (&rest body)
  "Run BODY with DB bound to an in-memory flashcards database."
  `(if (not (sqlite-available-p))
       (ert-skip "sqlite support is not available in this Emacs")
     (let ((db (sqlite-open)))
       (unwind-protect
           (progn
             (imoogi-flashcards-db-init db)
             ,@body)
         (sqlite-close db)))))

(defmacro imoogi-flashcards-test--with-file-db (&rest body)
  "Run BODY with DB and DB-FILE bound to a temporary SQLite database."
  `(if (not (sqlite-available-p))
       (ert-skip "sqlite support is not available in this Emacs")
     (let* ((db-file (make-temp-file "imoogi-flashcards" nil ".sqlite"))
            (db (sqlite-open db-file)))
       (unwind-protect
           (progn
             (imoogi-flashcards-db-init db)
             ,@body)
         (ignore-errors (sqlite-close db))
         (delete-file db-file)))))

(defun imoogi-flashcards-test--card (key &optional heading)
  "Return a simple test card with KEY."
  (make-imoogi-flashcards-card
   :key key
   :note-id key
   :kind "Basic"
   :source-file "cards.org"
   :heading (or heading key)
   :front (or heading key)
   :back "Back"))

(ert-deftest imoogi-flashcards-repo-schema-version-is-idempotent-and-refuses-future ()
  (imoogi-flashcards-test--with-db
   (imoogi-flashcards-db-init db)
   (should (= (caar (sqlite-select
                    db
                    "SELECT version FROM schema_version WHERE component = 'flashcards'"))
              1))
   (sqlite-execute db "UPDATE schema_version SET version = 99 WHERE component = 'flashcards'")
   (should-error (imoogi-flashcards-db-init db))))

(ert-deftest imoogi-flashcards-repo-schema-does-not-persist-org-content ()
  (imoogi-flashcards-test--with-db
   (let ((columns (mapcar #'cadr (sqlite-select db "PRAGMA table_info(cards)"))))
     (should-not (member "front" columns))
     (should-not (member "back" columns))
     (should-not (member "content_hash" columns))
     (should-not (member "heading" columns)))))

(ert-deftest imoogi-flashcards-repo-sync-upserts-state-and-tombstones-missing ()
  (imoogi-flashcards-test--with-db
   (let ((keys (imoogi-flashcards-repo-sync-cards
                db (list (imoogi-flashcards-test--card "a")
                         (imoogi-flashcards-test--card "b"))
                1000)))
     (should (equal keys '("a" "b")))
     (should (= (imoogi-flashcards-repo-card-count db) 2))
     (should (= (caar (sqlite-select db "SELECT count(*) FROM review_state")) 2)))
   (imoogi-flashcards-repo-sync-cards
    db (list (imoogi-flashcards-test--card "a" "A2")) 2000)
   (should (= (imoogi-flashcards-repo-card-count db) 1))
   (should (= (caar (sqlite-select db
                                   "SELECT count(*) FROM cards WHERE key = 'b' AND tombstoned_at = 2000"))
              1))
   (should (equal (caar (sqlite-select db "SELECT source_file FROM cards WHERE key = 'a'"))
                  "cards.org"))))

(ert-deftest imoogi-flashcards-repo-buffer-sync-tombstones-only-that-source ()
  (imoogi-flashcards-test--with-db
   (imoogi-flashcards-repo-sync-cards
    db (list (imoogi-flashcards-test--card "a")
             (let ((card (imoogi-flashcards-test--card "b")))
               (setf (imoogi-flashcards-card-source-file card) "other.org")
               card))
    1000)
   (imoogi-flashcards-repo-sync-cards
    db (list (imoogi-flashcards-test--card "a")) 2000 "cards.org")
   (should (= (caar (sqlite-select db
                                   "SELECT count(*) FROM cards WHERE key = 'b' AND tombstoned_at IS NULL"))
              1))))

(ert-deftest imoogi-flashcards-repo-due-cards-return-source-identifiers-only ()
  (imoogi-flashcards-test--with-db
   (imoogi-flashcards-repo-sync-cards
    db (list (imoogi-flashcards-test--card "a")) 1000)
   (let ((row (car (imoogi-flashcards-repo-due-cards db 1000 1))))
     (should (equal (nth 0 row) "a"))
     (should (equal (nth 1 row) "a"))
     (should (equal (nth 2 row) "Basic"))
     (should (equal (nth 4 row) "cards.org"))
     (should (equal (nth 5 row) "new")))))

(ert-deftest imoogi-flashcards-repo-review-is-atomic-and-logs-history ()
  (imoogi-flashcards-test--with-db
   (imoogi-flashcards-repo-sync-cards
    db (list (imoogi-flashcards-test--card "a")) 1000)
   (let ((next (imoogi-flashcards-repo-review db "a" 'good 2000)))
     (should (= (imoogi-flashcards-state-interval next) 1))
     (should (eq (imoogi-flashcards-state-status next) 'review))
     (should (= (caar (sqlite-select db "SELECT count(*) FROM review_log")) 1))
     (should (= (caar (sqlite-select db "SELECT interval FROM review_state WHERE card_key = 'a'"))
                1)))))

(ert-deftest imoogi-flashcards-repo-review-rolls-back-on-error ()
  (imoogi-flashcards-test--with-db
   (imoogi-flashcards-repo-sync-cards
    db (list (imoogi-flashcards-test--card "a")) 1000)
   (should-error
    (imoogi-flashcards-with-immediate-transaction db
      (sqlite-execute
       db
       "INSERT INTO review_log
          (card_key, rating, reviewed_at, previous_status, next_status,
           previous_interval, next_interval, previous_ease, next_ease,
           previous_due_at, next_due_at)
        VALUES ('a', 'good', 2000, 'new', 'review', 0, 1, 2.5, 2.5, 0, 88400)")
      (error "boom")))
   (should (= (caar (sqlite-select db
                                   "SELECT interval FROM review_state WHERE card_key = 'a'"))
              0))
   (should (= (caar (sqlite-select db "SELECT count(*) FROM review_log"))
              0))))

(ert-deftest imoogi-flashcards-repo-review-busy-has-no-partial-writes ()
  (imoogi-flashcards-test--with-file-db
   (imoogi-flashcards-repo-sync-cards
    db (list (imoogi-flashcards-test--card "a")) 1000)
   (let ((blocked-db (sqlite-open db-file)))
     (unwind-protect
         (progn
           (imoogi-flashcards-db-init blocked-db)
           (sqlite-execute db "BEGIN IMMEDIATE")
           (should-error (imoogi-flashcards-repo-review blocked-db "a" 'good 2000)
                         :type 'user-error)
           (sqlite-execute db "ROLLBACK")
           (should (= (caar (sqlite-select blocked-db
                                           "SELECT interval FROM review_state WHERE card_key = 'a'"))
                      0))
           (should (= (caar (sqlite-select blocked-db "SELECT count(*) FROM review_log"))
                      0)))
       (ignore-errors (sqlite-execute db "ROLLBACK"))
       (sqlite-close blocked-db)))))

(ert-deftest imoogi-flashcards-repo-review-rejects-missing-card ()
  (imoogi-flashcards-test--with-db
   (should-error (imoogi-flashcards-repo-review db "missing" 'good 2000)
                 :type 'user-error)
   (should (= (caar (sqlite-select db "SELECT count(*) FROM review_log")) 0))))

(provide 'flashcards-repository-test)
;;; flashcards-repository-test.el ends here
