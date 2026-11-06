;;; imoogi-orgid.el --- Heading-local Org IDs for the open-source link -*- lexical-binding: t; -*-

;;; Commentary:

;; SPEC-ANKISRC-001's sync side.  A card's back carries a link naming the
;; heading's own Org `:ID:' (never a file path), so every sync target needs
;; one before it is sent, and the ID's file location is recorded in Org's
;; own `org-id-locations' cache so the open side can find it fast.
;;
;;  - `imoogi-orgid-ensure' is the pre-send ID pass (REQ-ASRC-001).  It
;;    gives every ID-less `ANKI_NOTE_TYPE' heading in the named files a
;;    heading-local ID, through a live buffer and Org's own API.  Buffer
;;    policy: a buffer the user already visits is reused and is saved only
;;    when it was clean; a modified one is left modified (needs-save).  A
;;    buffer the pass opened itself is closed again.  Each file is its own
;;    failure domain, and the pass never asks the user anything.
;;  - `imoogi-orgid-duplicates' finds IDs claimed by two or more headings
;;    in one sync (REQ-ASRC-004); those headings are sent without one.
;;  - `imoogi-orgid-register' / `imoogi-orgid-save' record the sent IDs and
;;    write `org-id-locations-file' once, at the end of the sync
;;    (REQ-ASRC-005).  The pass itself only registers, never saves.
;;
;; ANKI_NOTE_ID and :ID: are independent (REQ-ASRC-002): nothing here
;; reads or writes ANKI_NOTE_ID.

;;; Code:

(require 'org)
(require 'org-id)
(require 'cl-lib)
(require 'seq)

(defconst imoogi-orgid-regexp "\\`[A-Za-z0-9._:@-]\\{1,128\\}\\'"
  "The Org ID character set shared with the Go side (DD-3).
Matches `protocol.ValidOrgID': the three `org-id-method' formats plus the
@FQDN suffix `org-id-include-domain' adds.")

(defun imoogi-orgid-valid-p (id)
  "Return non-nil when ID is a string inside `imoogi-orgid-regexp'."
  (and (stringp id)
       (let ((case-fold-search nil))
         (string-match-p imoogi-orgid-regexp id))))

(defun imoogi-orgid-local-id ()
  "Return the heading-local :ID: at point, or nil.
Inheritance is off, so a parent's ID is never returned (REQ-ASRC-001.1)."
  (let ((id (org-entry-get (point) "ID" nil)))
    (and id (not (string-empty-p (string-trim id))) (string-trim id))))

(defun imoogi-orgid--target-markers ()
  "Return markers on every ID-less ANKI_NOTE_TYPE heading in the buffer.
Collected first and edited afterwards, so inserting a drawer line never
disturbs the traversal."
  (let (markers)
    (org-map-entries
     (lambda ()
       (when (and (org-entry-get (point) "ANKI_NOTE_TYPE" nil)
                  (not (imoogi-orgid-local-id)))
         (push (point-marker) markers))))
    (nreverse markers)))

(defun imoogi-orgid--forget (ids)
  "Drop IDS from the in-memory location cache.
Used for IDs that exist only in a buffer, so the end-of-sync save never
persists a location for an ID that is not on disk (plan-audit O5)."
  (when (hash-table-p org-id-locations)
    (dolist (id ids) (remhash id org-id-locations))))

(defun imoogi-orgid--refuse-prompt (&rest _)
  "Signal instead of asking: the ID pass never prompts (REQ-ASRC-001.3)."
  (signal 'file-error (list "imoogi: the Org ID pass does not ask questions")))

(defun imoogi-orgid--ensure-file (file)
  "Give FILE's ID-less sync targets a heading-local ID.
Return a plist (:status STATUS :created IDS :reason STRING) where STATUS
is `saved', `needs-save', `unchanged' or `failed'."
  (let* ((existing (find-buffer-visiting file))
         (had-unsaved (and existing (buffer-modified-p existing)))
         (buffer existing)
         (opened nil)
         (created nil)
         (outcome
          (condition-case err
              (cl-letf (((symbol-function 'ask-user-about-lock) #'imoogi-orgid--refuse-prompt)
                        ((symbol-function 'ask-user-about-supersession-threat)
                         #'imoogi-orgid--refuse-prompt)
                        ((symbol-function 'yes-or-no-p) #'imoogi-orgid--refuse-prompt)
                        ((symbol-function 'y-or-n-p) #'imoogi-orgid--refuse-prompt))
                (unless (file-writable-p file)
                  (signal 'file-error (list "File is not writable" file)))
                (when (and existing (not had-unsaved)
                           (not (verify-visited-file-modtime existing)))
                  (signal 'file-error
                          (list "Buffer is out of date with the file on disk" file)))
                (unless buffer
                  (let ((enable-local-variables :safe)
                        (large-file-warning-threshold nil))
                    (setq buffer (find-file-noselect file)
                          opened t)))
                (with-current-buffer buffer
                  (when buffer-read-only
                    (signal 'buffer-read-only (list buffer)))
                  (unless (derived-mode-p 'org-mode)
                    (delay-mode-hooks (org-mode)))
                  (save-excursion
                    (save-restriction
                      (widen)
                      (dolist (marker (imoogi-orgid--target-markers))
                        (goto-char marker)
                        (push (org-id-get (point) 'create) created)
                        (set-marker marker nil))))
                  (cond
                   (had-unsaved
                    ;; REQ-ASRC-003.2: an ID made in a modified buffer is
                    ;; not on disk, so it is neither sent nor cached now.
                    (imoogi-orgid--forget created)
                    (list :status 'needs-save :created created))
                   ((null created)
                    (list :status 'unchanged :created nil))
                   (t
                    (save-buffer)
                    (list :status 'saved :created created)))))
            (error
             (imoogi-orgid--forget created)
             (when (buffer-live-p buffer)
               (cond
                (opened
                 (with-current-buffer buffer (set-buffer-modified-p nil)))
                ((and created (not had-unsaved))
                 ;; The user's buffer was clean before this pass touched
                 ;; it, so reverting restores exactly what they had.
                 (ignore-errors
                   (with-current-buffer buffer (revert-buffer t t t))))))
             (list :status 'failed :created nil
                   :reason (error-message-string err))))))
    ;; Only a buffer this pass opened is closed, and only once it is
    ;; unmodified (saved, unchanged, or its changes discarded above).
    (when (and opened (buffer-live-p buffer) (not (buffer-modified-p buffer)))
      (kill-buffer buffer))
    outcome))

(defun imoogi-orgid-ensure (files)
  "Run the ID pass over FILES (absolute paths, each at most once).
Return a plist:
  :saved      files written to disk with new IDs (a rescan is due)
  :needs-save files whose modified buffer received IDs but was not saved
  :failed     list of (FILE . REASON) for files the pass could not handle"
  (let (saved needs-save failed)
    (dolist (file (delete-dups (copy-sequence files)))
      (let ((outcome (imoogi-orgid--ensure-file file)))
        (pcase (plist-get outcome :status)
          ('saved (push file saved))
          ('needs-save (push file needs-save))
          ('failed (push (cons file (plist-get outcome :reason)) failed)))))
    (list :saved (nreverse saved)
          :needs-save (nreverse needs-save)
          :failed (nreverse failed))))

(defun imoogi-orgid-duplicates (entries)
  "Return IDs carried by two or more of ENTRIES, as (ID . FILES).
ENTRIES are (ID . FILE) pairs; FILES keeps first-seen order."
  (let ((by-id (make-hash-table :test #'equal)) order duplicates)
    (dolist (pair entries)
      (unless (gethash (car pair) by-id) (push (car pair) order))
      (puthash (car pair) (append (gethash (car pair) by-id) (list (cdr pair))) by-id))
    (dolist (id (nreverse order))
      (let ((files (gethash id by-id)))
        (when (> (length files) 1)
          (push (cons id (delete-dups (copy-sequence files))) duplicates))))
    (nreverse duplicates)))

(defun imoogi-orgid-register (id file)
  "Record ID at FILE in `org-id-locations', in Org's own path form.
Loads the cache first when it is not in memory yet, so the end-of-sync
save extends the user's cache instead of replacing it."
  (when (and org-id-track-globally (imoogi-orgid-valid-p id))
    (org-id-add-location id (expand-file-name file))
    t))

(defun imoogi-orgid-save ()
  "Write `org-id-locations' to `org-id-locations-file' once."
  (when org-id-track-globally
    (org-id-locations-save)))

(provide 'imoogi-orgid)
;;; imoogi-orgid.el ends here
