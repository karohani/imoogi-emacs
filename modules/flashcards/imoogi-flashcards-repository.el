;;; imoogi-flashcards-repository.el --- SQLite repository -*- lexical-binding: t; -*-

;;; Code:

(require 'cl-lib)
(require 'sqlite)
(require 'subr-x)
(require 'imoogi-flashcards-core)

(defgroup imoogi-flashcards nil
  "Emacs-native local flashcards."
  :group 'imoogi)

(defcustom imoogi-flashcards-database-file
  (expand-file-name ".cache/flashcards.sqlite" user-emacs-directory)
  "SQLite file used by the Emacs-native flashcard fallback."
  :type 'file
  :group 'imoogi-flashcards)

(defconst imoogi-flashcards-schema-component "flashcards"
  "Component key used in schema_version.")

(defconst imoogi-flashcards-schema-version 1
  "Current flashcards schema version.")

(defun imoogi-flashcards-db-open (&optional file)
  "Open flashcards database FILE and ensure the schema exists."
  (let ((path (or file imoogi-flashcards-database-file)))
    (when path
      (make-directory (file-name-directory path) t))
    (let ((db (sqlite-open path)))
      (condition-case err
          (progn
            (imoogi-flashcards-db-init db)
            db)
        (error
         (ignore-errors (sqlite-close db))
         (signal (car err) (cdr err)))))))

(defun imoogi-flashcards-db-close (db)
  "Close DB."
  (sqlite-close db))

(defun imoogi-flashcards-db-init (db)
  "Create or migrate the flashcards schema in DB."
  (sqlite-execute db "PRAGMA foreign_keys = ON")
  (sqlite-execute db "PRAGMA busy_timeout = 250")
  (sqlite-execute db
                  "CREATE TABLE IF NOT EXISTS schema_version (
                     component TEXT PRIMARY KEY,
                     version INTEGER NOT NULL
                   )")
  (let ((version (caar (sqlite-select
                       db
                       "SELECT version FROM schema_version WHERE component = ?"
                       (list imoogi-flashcards-schema-component)))))
    (when (and version (> version imoogi-flashcards-schema-version))
      (error "imoogi flashcards: future SQLite schema version %s is not supported"
             version))
    (unless version
      (sqlite-execute
       db
       "INSERT INTO schema_version(component, version) VALUES (?, ?)"
       (list imoogi-flashcards-schema-component
             imoogi-flashcards-schema-version))))
  (sqlite-execute db
                  "CREATE TABLE IF NOT EXISTS cards (
                     key TEXT PRIMARY KEY,
                     note_id TEXT NOT NULL,
                     kind TEXT NOT NULL,
                     cloze_no INTEGER,
                     source_file TEXT NOT NULL,
                     updated_at INTEGER NOT NULL,
                     tombstoned_at INTEGER
                   )")
  (sqlite-execute db "CREATE INDEX IF NOT EXISTS idx_cards_note_id ON cards(note_id)")
  (sqlite-execute db "CREATE INDEX IF NOT EXISTS idx_cards_due ON cards(tombstoned_at)")
  (sqlite-execute db
                  "CREATE TABLE IF NOT EXISTS review_state (
                     card_key TEXT PRIMARY KEY,
                     status TEXT NOT NULL DEFAULT 'new',
                     reps INTEGER NOT NULL DEFAULT 0,
                     lapses INTEGER NOT NULL DEFAULT 0,
                     interval INTEGER NOT NULL DEFAULT 0,
                     ease REAL NOT NULL DEFAULT 2.5,
                     due_at INTEGER NOT NULL DEFAULT 0,
                     last_reviewed_at INTEGER,
                     FOREIGN KEY(card_key) REFERENCES cards(key) ON DELETE CASCADE
                   )")
  (sqlite-execute db "CREATE INDEX IF NOT EXISTS idx_review_state_due ON review_state(due_at)")
  (sqlite-execute db
                  "CREATE TABLE IF NOT EXISTS review_log (
                     id INTEGER PRIMARY KEY AUTOINCREMENT,
                     card_key TEXT NOT NULL,
                     rating TEXT NOT NULL,
                     reviewed_at INTEGER NOT NULL,
                     previous_status TEXT NOT NULL,
                     next_status TEXT NOT NULL,
                     previous_interval INTEGER NOT NULL,
                     next_interval INTEGER NOT NULL,
                     previous_ease REAL NOT NULL,
                     next_ease REAL NOT NULL,
                     previous_due_at INTEGER NOT NULL,
                     next_due_at INTEGER NOT NULL
                   )"))

(defmacro imoogi-flashcards-with-immediate-transaction (db &rest body)
  "Run BODY in a BEGIN IMMEDIATE transaction against DB."
  (declare (indent 1))
  `(progn
     (condition-case err
         (progn
           (sqlite-execute ,db "BEGIN IMMEDIATE")
           (condition-case body-err
               (let ((result (progn ,@body)))
                 (sqlite-execute ,db "COMMIT")
                 result)
             (error
              (ignore-errors (sqlite-execute ,db "ROLLBACK"))
              (signal (car body-err) (cdr body-err)))))
       (sqlite-error
        (let ((msg (error-message-string err)))
          (if (string-match-p "\\(busy\\|locked\\)" (downcase msg))
              (user-error "imoogi flashcards: SQLite DB가 잠겨 있습니다. 잠시 후 다시 시도하세요")
            (signal (car err) (cdr err))))))))

(defun imoogi-flashcards-repo-upsert-cards (db cards now)
  "Upsert CARDS into DB at NOW and return their keys."
  (let (keys)
    (dolist (card cards)
      (let ((key (imoogi-flashcards-card-key card)))
        (push key keys)
        (sqlite-execute
         db
         "INSERT INTO cards
            (key, note_id, kind, cloze_no, source_file, updated_at, tombstoned_at)
          VALUES (?, ?, ?, ?, ?, ?, NULL)
          ON CONFLICT(key) DO UPDATE SET
            note_id = excluded.note_id,
            kind = excluded.kind,
            cloze_no = excluded.cloze_no,
            source_file = excluded.source_file,
            updated_at = excluded.updated_at,
            tombstoned_at = NULL"
         (list key
               (imoogi-flashcards-card-note-id card)
               (imoogi-flashcards-card-kind card)
               (imoogi-flashcards-card-cloze-no card)
               (imoogi-flashcards-card-source-file card)
               now))
        (sqlite-execute
         db
         "INSERT OR IGNORE INTO review_state
            (card_key, status, reps, lapses, interval, ease, due_at)
          VALUES (?, 'new', 0, 0, 0, 2.5, 0)"
         (list key))))
    (nreverse keys)))

(defun imoogi-flashcards-repo-tombstone-missing (db active-keys now &optional source-file)
  "Mark DB cards missing from ACTIVE-KEYS as tombstoned at NOW.
When SOURCE-FILE is non-nil, only cards from that source file are eligible."
  (let ((active active-keys))
    (dolist (row (if source-file
                     (sqlite-select
                      db
                      "SELECT key FROM cards
                        WHERE tombstoned_at IS NULL AND source_file = ?"
                      (list source-file))
                   (sqlite-select
                    db "SELECT key FROM cards WHERE tombstoned_at IS NULL")))
      (let ((key (car row)))
        (unless (member key active)
          (sqlite-execute db
                          "UPDATE cards SET tombstoned_at = ? WHERE key = ?"
                          (list now key)))))))

(defun imoogi-flashcards-repo-sync-cards (db cards now &optional source-file)
  "Replace the active projection in DB with CARDS at NOW.
When SOURCE-FILE is non-nil, tombstone only stale cards from SOURCE-FILE."
  (imoogi-flashcards-with-immediate-transaction db
    (let ((keys (imoogi-flashcards-repo-upsert-cards db cards now)))
      (imoogi-flashcards-repo-tombstone-missing db keys now source-file)
      keys)))

(defun imoogi-flashcards--state-from-row (row)
  "Convert sqlite ROW to a review state struct."
  (imoogi-flashcards-state-create
   :status (intern (nth 0 row))
   :reps (nth 1 row)
   :lapses (nth 2 row)
   :interval (nth 3 row)
   :ease (nth 4 row)
   :due-at (nth 5 row)
   :last-reviewed-at (nth 6 row)))

(defun imoogi-flashcards-repo-state (db key)
  "Return review state for card KEY in DB."
  (let ((row (car (sqlite-select
                   db
                   "SELECT status, reps, lapses, interval, ease, due_at, last_reviewed_at
                    FROM review_state WHERE card_key = ?"
                   (list key)))))
    (if row
        (imoogi-flashcards--state-from-row row)
      (imoogi-flashcards-state-create))))

(defun imoogi-flashcards-repo-due-cards (db now &optional limit)
  "Return due active cards from DB at NOW."
  (sqlite-select
   db
   "SELECT c.key, c.note_id, c.kind, c.cloze_no, c.source_file,
           s.status, s.reps, s.lapses, s.interval, s.ease, s.due_at, s.last_reviewed_at
      FROM cards c
      JOIN review_state s ON s.card_key = c.key
     WHERE c.tombstoned_at IS NULL AND s.due_at <= ?
     ORDER BY s.due_at ASC, c.updated_at ASC
     LIMIT ?"
   (list now (or limit 50))))

(defun imoogi-flashcards-repo-card-count (db)
  "Return active card count in DB."
  (caar (sqlite-select db "SELECT count(*) FROM cards WHERE tombstoned_at IS NULL")))

(defun imoogi-flashcards-repo-review (db key rating now)
  "Record RATING for card KEY at NOW and update review state atomically."
  (imoogi-flashcards-with-immediate-transaction db
    (unless (car (sqlite-select
                  db
                  "SELECT 1 FROM cards WHERE key = ? AND tombstoned_at IS NULL"
                  (list key)))
      (user-error "imoogi flashcards: 복습할 수 없는 카드입니다: %s" key))
    (let* ((old (imoogi-flashcards-repo-state db key))
           (new (imoogi-flashcards-schedule old rating now)))
      (sqlite-execute
       db
       "UPDATE review_state
           SET status = ?, reps = ?, lapses = ?, interval = ?, ease = ?,
               due_at = ?, last_reviewed_at = ?
         WHERE card_key = ?"
       (list (symbol-name (imoogi-flashcards-state-status new))
             (imoogi-flashcards-state-reps new)
             (imoogi-flashcards-state-lapses new)
             (imoogi-flashcards-state-interval new)
             (imoogi-flashcards-state-ease new)
             (imoogi-flashcards-state-due-at new)
             (imoogi-flashcards-state-last-reviewed-at new)
             key))
      (sqlite-execute
       db
       "INSERT INTO review_log
          (card_key, rating, reviewed_at, previous_status, next_status,
           previous_interval, next_interval,
           previous_ease, next_ease, previous_due_at, next_due_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
       (list key
             (symbol-name rating)
             now
             (symbol-name (imoogi-flashcards-state-status old))
             (symbol-name (imoogi-flashcards-state-status new))
             (imoogi-flashcards-state-interval old)
             (imoogi-flashcards-state-interval new)
             (imoogi-flashcards-state-ease old)
             (imoogi-flashcards-state-ease new)
             (imoogi-flashcards-state-due-at old)
             (imoogi-flashcards-state-due-at new)))
      new)))

(provide 'imoogi-flashcards-repository)
;;; imoogi-flashcards-repository.el ends here
