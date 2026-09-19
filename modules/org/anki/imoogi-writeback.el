;;; imoogi-writeback.el --- Buffer-mediated ANKI_NOTE_ID write-back -*- lexical-binding: t; -*-

;;; Commentary:

;; Write-back for newly `added' results is performed through a live
;; Emacs buffer visiting the originating file, using Org's own property
;; API -- never direct file I/O (plan.md D-4, REQ-007).
;;
;;  - No prior unsaved modifications: write the property, save the
;;    buffer.  The user sees a saved file with the identifier in it.
;;  - Prior unsaved modifications: write the property into the buffer
;;    and leave it modified and unsaved.  The user's own next save
;;    carries both their edits and the identifier.

;;; Code:

(require 'org)

(defun imoogi-writeback--find-target (n)
  "Move point in the current Org buffer to the Nth (0-based) sync
target in document order -- a heading whose own PROPERTIES drawer
carries ANKI_NOTE_TYPE -- and return t.  Return nil if fewer than N+1
sync targets exist.

Mirrors the `key' derivation `imoogi-scan.el' used to number targets,
so a result's key resolves back to the same heading the scan found."
  (goto-char (point-min))
  (let ((index 0) (found nil))
    (org-map-entries
     (lambda ()
       (when (and (not found) (org-entry-get (point) "ANKI_NOTE_TYPE" nil))
         (if (= index n)
             (setq found (point))
           (setq index (1+ index))))))
    (when found
      (goto-char found)
      t)))

(defun imoogi-writeback--key-parts (key)
  "Split KEY (\"<source_path>::<n>\") into (RELATIVE-PATH . INDEX)."
  (let* ((pos (string-match "::\\([0-9]+\\)\\'" key))
         (relative-path (substring key 0 pos))
         (index (string-to-number (match-string 1 key))))
    (cons relative-path index)))

(defconst imoogi-writeback-note-type-prefix "imoogi-"
  "Prefix marking a note type imoogi owns (REQ-C-005).")

(defun imoogi-writeback-note-type-counterpart (note-type)
  "Return NOTE-TYPE's imoogi-owned counterpart (REQ-C-020.3).

A stock name gains the prefix; a name that already carries it is
returned unchanged, so migrating a heading the user had already
hand-edited to the imoogi- form is idempotent rather than producing
`imoogi-imoogi-Basic'.

Derived locally on purpose: `protocol.Result' carries no note type, and
design.md SS11 chose local derivation over adding a response field for
one value the front end can compute."
  (if (string-prefix-p imoogi-writeback-note-type-prefix note-type)
      note-type
    (concat imoogi-writeback-note-type-prefix note-type)))

(defun imoogi-writeback--apply-added (sync-root results write-fn)
  "For each RESULTS plist with :action \"added\", locate the originating
heading under SYNC-ROOT and call WRITE-FN with point on it and the
result plist as its argument.

Returns the list of relative file paths left needing a save: a buffer
that already had unsaved modifications is left modified rather than
saved, so the user's own next save carries both their edits and
whatever WRITE-FN wrote (plan.md D-4, REQ-007)."
  (let (needs-save)
    (dolist (result results)
      (when (equal (plist-get result :action) "added")
        (let* ((key (plist-get result :key))
               (parts (imoogi-writeback--key-parts key))
               (relative-path (car parts))
               (index (cdr parts))
               (file (expand-file-name relative-path sync-root))
               (existing-buffer (find-buffer-visiting file))
               (had-unsaved (and existing-buffer (buffer-modified-p existing-buffer)))
               (buffer (or existing-buffer (find-file-noselect file))))
          (with-current-buffer buffer
            (unless (derived-mode-p 'org-mode)
              (delay-mode-hooks (org-mode)))
            (save-excursion
              (when (imoogi-writeback--find-target index)
                (funcall write-fn result)))
            (if had-unsaved
                (push relative-path needs-save)
              (save-buffer))))))
    (nreverse needs-save)))

(defun imoogi-writeback-apply-migration (sync-root results)
  "Write back a confirmed migration's results (REQ-C-020.3).

For each added result -- a successful re-home reports `added' carrying
the NEW identifier -- overwrite the heading's ANKI_NOTE_ID with that
identifier AND its ANKI_NOTE_TYPE with the imoogi- counterpart of the
type the heading currently declares.

Both writes go through `org-entry-put', which overwrites an existing
property rather than only inserting a missing one -- verified, not
assumed (design.md SS11).  The two properties move together because a
heading left declaring the stock type after its note has been re-homed
is exactly the recorded-vs-declared mismatch the next ordinary sync
would report as `note_type_change_unsupported'.

Returns the list of relative file paths left needing a save."
  (imoogi-writeback--apply-added
   sync-root results
   (lambda (result)
     (org-entry-put (point) "ANKI_NOTE_ID"
                    (number-to-string (plist-get result :note-id)))
     (let ((declared (org-entry-get (point) "ANKI_NOTE_TYPE" nil)))
       (when declared
         (org-entry-put (point) "ANKI_NOTE_TYPE"
                        (imoogi-writeback-note-type-counterpart declared)))))))

(defun imoogi-writeback-apply (sync-root results)
  "For each RESULTS plist with :action \"added\", locate the originating
heading via a live buffer visiting its file under SYNC-ROOT (opening
one if none exists), write its ANKI_NOTE_ID property via Org's own
property API, and either save the buffer (no prior unsaved
modifications) or leave it modified and unsaved (prior unsaved
modifications present).

Returns the list of relative file paths left needing a save."
  (imoogi-writeback--apply-added
   sync-root results
   (lambda (result)
     (org-entry-put (point) "ANKI_NOTE_ID"
                    (number-to-string (plist-get result :note-id))))))

(provide 'imoogi-writeback)
;;; imoogi-writeback.el ends here
