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

Unset (nil) by default.  A sync run stops rather than guessing a root
when this is unset -- see `imoogi-sync'."
  :type '(choice (const :tag "Unset" nil) directory)
  :group 'imoogi)

(defcustom imoogi-exclude-patterns nil
  "List of path-pattern strings, relative to `imoogi-sync-root'.

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

;;;###autoload
(defun imoogi-sync ()
  "Run one imoogi sync: scan, resolve, dispatch, write back, report.

Stops before dispatch (REQ-018 preconditions) when `imoogi-binary-path'
does not resolve to an executable, or when `imoogi-sync-root' is
unset -- both rendered through `imoogi-error-message' (imoogi-error.el)
rather than as a raw format string, per REQ-018.

Returns the human-readable report string (also shown via `message')."
  (interactive)
  (let* ((binary (executable-find imoogi-binary-path))
         (report
          (cond
           ((null binary)
            (imoogi-error-message "binary_not_found"))
           ((null imoogi-sync-root)
            (imoogi-error-message "sync_root_unset"))
           (t
            (let* ((scan (imoogi-scan-root imoogi-sync-root imoogi-exclude-patterns))
                   (entries (plist-get scan :entries))
                   (unreadable (plist-get scan :unreadable-files))
                   (config (list :default-deck imoogi-default-deck
                                  :anki-connect-url imoogi-anki-connect-url
                                  :registry-path (imoogi--registry-path imoogi-sync-root)
                                  :sync-root imoogi-sync-root
                                  :exclude-patterns imoogi-exclude-patterns
                                  :scan-complete (plist-get scan :scan-complete)))
                   (response (imoogi-process-run binary config
                                                  (plist-get scan :census)
                                                  entries)))
              (cond
               ((null response)
                "imoogi: sync run failed -- no response from binary")
               (t
                (let* ((needs-save
                        (imoogi-writeback-apply imoogi-sync-root
                                                 (plist-get response :results)))
                       (nil-key-errors (imoogi--nil-key-errors response))
                       (unreadable-note
                        (when unreadable
                          (format " Unreadable and skipped: %s."
                                  (mapconcat #'identity unreadable ", "))))
                       (base
                        (if (not (eq (plist-get response :ok) t))
                            ;; REQ-018's Go-side triggers: a whole-run
                            ;; failure (anki_unreachable /
                            ;; ankiconnect_missing) rendered through the
                            ;; same table the Elisp-side preconditions
                            ;; above use -- never the code's own raw
                            ;; :message text.  design.md SS3 step 13: a
                            ;; run this far along (`ok: false') never
                            ;; carries results, so the summary line
                            ;; would report nothing anyway.
                            (imoogi-error-message
                             (plist-get (car nil-key-errors) :code))
                          ;; An `ok: true' run may still carry whole-run
                          ;; ADVISORY diagnostics that name no single
                          ;; entry (delete_suppressed,
                          ;; delete_candidate_unowned) -- these are
                          ;; additive information, not a reason to
                          ;; discard the results/errors summary the way
                          ;; a genuine run-level failure is.
                          (concat
                           (format "imoogi: sync complete (%d results, %d errors)"
                                   (length (plist-get response :results))
                                   (length (plist-get response :errors)))
                           (mapconcat
                            (lambda (err)
                              (concat " " (imoogi-error-message (plist-get err :code))))
                            nil-key-errors
                            "")))))
                  (concat base
                          (imoogi--keyed-error-lines response entries)
                          (or unreadable-note "")
                          (if needs-save
                              (format "; needs save: %s"
                                      (mapconcat #'identity needs-save ", "))
                            ""))))))))))
    (message "%s" report)
    report))

(provide 'imoogi)
;;; imoogi.el ends here
