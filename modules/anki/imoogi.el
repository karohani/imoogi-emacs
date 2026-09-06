;;; imoogi.el --- Sync Org-mode headings to Anki flashcards -*- lexical-binding: t; -*-

;; Author: imoogi
;; Keywords: outlines, org, anki

;;; Commentary:

;; imoogi is a one-way Org-mode to Anki flashcard sync.  This file is the
;; package entry point: the defcustoms every other module reads, and the
;; `imoogi-sync' interactive command that drives one full sync run:
;; scan -> resolve -> assemble request -> dispatch -> parse response ->
;; write back -> report.
;;
;; See design.md SS3 (SPEC-ANKI-001) for the full sync-run sequence this
;; command implements.

;;; Code:

(require 'org)
(require 'json)
(require 'seq)

(defgroup imoogi nil
  "Sync Org-mode headings to Anki flashcards via AnkiConnect."
  :group 'org
  :prefix "imoogi-")

(defcustom imoogi-binary-path "imoogi-anki"
  "Path to the imoogi Go binary.
A relative value (the default) is resolved on `exec-path'."
  :type 'string
  :group 'imoogi)

(defcustom imoogi-sync-root nil
  "Absolute path to the directory imoogi recursively scans for sync targets.

Unset (nil) by default.  Host-registered folders/files can be used
without this legacy root -- see `imoogi-sync'."
  :type '(choice (const :tag "Unset" nil) directory)
  :group 'imoogi)

(defcustom imoogi-exclude-patterns nil
  "List of path-pattern strings, relative to each target's sync root.

A file whose relative path matches any pattern here is excluded from
sync-target eligibility.  It is still scanned for the identifier
census -- see `imoogi-scan-root'."
  :type '(repeat string)
  :group 'imoogi)

(defcustom imoogi-default-deck "Default"
  "Anki deck used by the back end when a sync target's ANKI_DECK
resolves to no value (absent everywhere in the inheritance chain, or
present but empty at some level)."
  :type 'string
  :group 'imoogi)

(defcustom imoogi-anki-connect-url "http://127.0.0.1:8765"
  "URL at which the imoogi binary reaches AnkiConnect."
  :type 'string
  :group 'imoogi)

(defcustom imoogi-config-file
  (expand-file-name "imoogi.json" user-emacs-directory)
  "Path to the imoogi configuration file written by `imoogi-anki-setup'."
  :type 'file
  :group 'imoogi)

(defcustom imoogi-user-stylesheet-file
  (expand-file-name "imoogi-anki.css"
                    (file-name-directory imoogi-config-file))
  "Path to the user's own Anki stylesheet, appended after imoogi's base
stylesheet on every install run (REQ-C-008).

Beside `imoogi-config-file' by default.  The file is optional: when it
does not exist the install step uploads the base stylesheet alone and
reports nothing -- an absent file is a configuration choice, not an
error.

Read by the FRONT END, never by the Go binary: its contents travel in
the install request's `user_css' field.  That split is deliberate
(design.md SS11) -- path expansion already lives here, the wire
contract's \"the binary reads no configuration file of its own\"
property survives, and no new diagnostic code is needed for a missing
file."
  :type 'file
  :group 'imoogi)

(defun imoogi-user-stylesheet-contents ()
  "Return the contents of `imoogi-user-stylesheet-file', or \"\" when
no readable file sits at that path (REQ-C-008).

Never nil: the install request carries `user_css' as a JSON string on
both branches, so the absent case is the empty string rather than a
dropped key."
  (if (and imoogi-user-stylesheet-file
           (file-readable-p imoogi-user-stylesheet-file))
      (with-temp-buffer
        (let ((coding-system-for-read 'utf-8))
          (insert-file-contents imoogi-user-stylesheet-file))
        (buffer-string))
    ""))

(require 'imoogi-scan)
(require 'imoogi-targets)
(require 'imoogi-target-scan)
(require 'imoogi-props)
(require 'imoogi-writeback)
(require 'imoogi-process)
(require 'imoogi-config)
(require 'imoogi-error)

;; `imoogi-setup' is deliberately NOT required here: `imoogi-anki-setup'
;; is autoloaded, and a first-time user's very first invocation reaches
;; it via that autoload alone, without ever loading imoogi.el (design.md
;; SS2.5's setup command exists precisely to be usable before any other
;; part of the package has run).  `imoogi-setup.el' itself `require's
;; `imoogi' for the defcustoms it needs (`imoogi-binary-path',
;; `imoogi-anki-connect-url', `imoogi-sync-root', `imoogi-default-deck',
;; `imoogi-config-file') -- requiring it back here would be circular.

;; First-run population: if `imoogi-anki-setup' has already written a
;; configuration file, load it now so `imoogi-sync-root' and
;; `imoogi-default-deck' are populated at package startup rather than
;; requiring the user to set them by hand (REQ-017, plan.md D-11).
(imoogi-config-load)

(defun imoogi--registry-path (sync-root)
  "Return the registry file path for SYNC-ROOT: one registry per root."
  (expand-file-name ".imoogi-registry.json" sync-root))

(defun imoogi--keyed-error-lines (response entries)
  "Render every error plist in RESPONSE's :errors that carries a :key
as one line naming the heading and explaining the code.

A keyed error is a per-entry diagnostic (cloze_marker_missing,
note_id_duplicated, note_field_missing, ...): it says which heading
failed and why.  Before this existed the report counted these errors
but never showed them -- \"1 errors\" was the whole story, so a Cloze
skipped for a missing {{cN:: marker looked like Cloze silently not
working.  ENTRIES (the scan's :entries) supplies each key's title so the
line reads as the heading the user wrote, not only as file::N.  The
explanation is the REQ-018 table's wording (`imoogi-error-message'),
never the code's raw :message."
  (let ((title-by-key (mapcar (lambda (e) (cons (plist-get e :key) (plist-get e :title)))
                              entries)))
    (mapconcat
     (lambda (err)
       (let* ((key (plist-get err :key))
              (title (cdr (assoc key title-by-key))))
         (format "\n  %s%s: %s"
                 key
                 (if (and title (not (string-empty-p title))) (format " (%s)" title) "")
                 (imoogi-error-message (plist-get err :code)))))
     (seq-filter (lambda (err) (plist-get err :key)) (plist-get response :errors))
     "")))

(defun imoogi--nil-key-errors (response)
  "Return every error plist in RESPONSE's :errors carrying a nil :key.

A nil :key marks a whole-run-scoped diagnostic (design.md SS3 step 13's
handshake failure; also delete_suppressed and delete_candidate_unowned,
neither of which name one specific sync-target entry) rather than a
per-entry error such as cloze_marker_missing, which always carries the
failing entry's :key."
  (seq-filter (lambda (err) (null (plist-get err :key)))
              (plist-get response :errors)))

(defun imoogi--sync-scan (binary root registry-path scan &optional registered duplicate-ids)
  "Sync SCAN under ROOT with BINARY and REGISTRY-PATH.
REGISTERED protects host-registered cards from orphan deletion.
DUPLICATE-IDS names identifiers claimed across multiple target groups."
  (let* ((entries (plist-get scan :entries))
         (duplicates (seq-filter
                      (lambda (entry) (memq (plist-get entry :note-id) duplicate-ids))
                      entries))
         (config (list :default-deck imoogi-default-deck
                       :anki-connect-url imoogi-anki-connect-url
                       :registry-path registry-path
                       :sync-root root
                       ;; ../ protects legacy cards moved to an external target.
                       :exclude-patterns (cons "../" imoogi-exclude-patterns)
                       ;; Registrations are selections, not an exhaustive root:
                       ;; removing a selection never authorizes orphan deletion.
                       :scan-complete (and (not registered)
                                           (null duplicate-ids)
                                           (plist-get scan :scan-complete))))
         (response (imoogi-process-run
                    binary config
                    ;; Keep conflicted IDs out of reconciliation too.  The
                    ;; incomplete-scan gate above protects them from deletion.
                    (seq-remove (lambda (entry)
                                  (memq (plist-get entry :note-id) duplicate-ids))
                                (plist-get scan :census))
                    (seq-difference entries duplicates #'equal))))
    (if (null response)
        "imoogi: sync run failed -- no response from binary"
      (when (eq (plist-get response :ok) t)
        ;; A complete registered selection intentionally disables deletion.
        ;; Report that policy below rather than counting it as a scan failure.
        (when (and registered (plist-get scan :scan-complete))
          (setf (plist-get response :errors)
                (seq-remove (lambda (err)
                              (and (null (plist-get err :key))
                                   (equal (plist-get err :code) "delete_suppressed")))
                            (plist-get response :errors))))
        (dolist (entry duplicates)
          (push (list :key (plist-get entry :key) :code "note_id_duplicated")
                (plist-get response :errors))
          (push (list :key (plist-get entry :key) :action "failed"
                      :note-id (plist-get entry :note-id))
                (plist-get response :results))))
      (let* ((needs-save (imoogi-writeback-apply root (plist-get response :results)))
             (nil-key-errors (imoogi--nil-key-errors response))
             (unreadable (plist-get scan :unreadable-files))
             (base (if (not (eq (plist-get response :ok) t))
                       (imoogi-error-message (plist-get (car nil-key-errors) :code))
                     (concat
                      (format "imoogi: sync complete (%d results, %d errors)"
                              (length (plist-get response :results))
                              (length (plist-get response :errors)))
                      (mapconcat
                       (lambda (err)
                         (concat " " (imoogi-error-message (plist-get err :code))))
                       nil-key-errors "")))))
        (concat base
                (when registered " (등록 대상: 추가·갱신만)")
                (imoogi--keyed-error-lines response entries)
                (when unreadable
                  (format " Unreadable and skipped: %s."
                          (mapconcat #'identity unreadable ", ")))
                (when needs-save
                  (format "; needs save: %s" (mapconcat #'identity needs-save ", "))))))))

(defun imoogi--target-registry-path (root)
  "Return the host-local state file for registered ROOT."
  (let ((directory (expand-file-name "imoogi-target-state/"
                                     (file-name-directory imoogi-targets-file))))
    (make-directory directory t)
    (expand-file-name (concat (secure-hash 'sha256 root) ".json") directory)))

(defun imoogi--target-duplicate-ids (groups)
  "Return identifiers claimed by more than one heading in GROUPS."
  (let ((counts (make-hash-table :test #'eql)) duplicates)
    (dolist (group groups)
      (dolist (entry (plist-get (plist-get group :scan) :entries))
        (when-let* ((id (plist-get entry :note-id)))
          (puthash id (1+ (gethash id counts 0)) counts))))
    (maphash (lambda (id count) (when (> count 1) (push id duplicates))) counts)
    duplicates))

;;;###autoload
(defun imoogi-sync ()
  "Sync the legacy root and every host-registered folder or Org file.
Register additional targets with `imoogi-anki-register-directory' or
`imoogi-anki-register-file'.  Host registrations do not delete orphan
Anki cards.  With no registrations, retain the legacy root workflow."
  (interactive)
  (let* ((binary (executable-find imoogi-binary-path))
         (targets (imoogi-targets-load))
         (report
          (cond
           ((null binary) (imoogi-error-message "binary_not_found"))
           ((and (null imoogi-sync-root) (null targets))
            (concat (imoogi-error-message "sync_root_unset")
                    " Or register a folder/file with imoogi-anki-register-directory/file."))
           ((null targets)
            (imoogi--sync-scan binary imoogi-sync-root
                               (imoogi--registry-path imoogi-sync-root)
                               (imoogi-scan-root imoogi-sync-root imoogi-exclude-patterns)))
           (t
            (let* ((groups (imoogi-target-scan imoogi-sync-root targets
                                               imoogi-exclude-patterns))
                   (duplicates (imoogi--target-duplicate-ids groups)))
              (mapconcat
               (lambda (group)
                 (let* ((root (plist-get group :root))
                        (legacy (plist-get group :legacy))
                        (registry (if legacy (imoogi--registry-path root)
                                    (imoogi--target-registry-path root))))
                   (concat root "\n"
                           (imoogi--sync-scan binary root registry
                                              (plist-get group :scan)
                                              (not legacy) duplicates))))
               groups "\n"))))))
    (message "%s" report)
    report))

(provide 'imoogi)
;;; imoogi.el ends here
