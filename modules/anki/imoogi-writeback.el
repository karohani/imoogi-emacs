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

(defun imoogi-writeback-apply (sync-root results)
  "For each RESULTS plist with :action \"added\", locate the originating
heading via a live buffer visiting its file under SYNC-ROOT (opening
one if none exists), write its ANKI_NOTE_ID property via Org's own
property API, and either save the buffer (no prior unsaved
modifications) or leave it modified and unsaved (prior unsaved
modifications present).

Returns the list of relative file paths left needing a save."
  (let (needs-save)
    (dolist (result results)
      (when (equal (plist-get result :action) "added")
        (let* ((key (plist-get result :key))
               (note-id (plist-get result :note-id))
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
                (org-entry-put (point) "ANKI_NOTE_ID" (number-to-string note-id))))
            (if had-unsaved
                (push relative-path needs-save)
              (save-buffer))))))
    (nreverse needs-save)))

(provide 'imoogi-writeback)
;;; imoogi-writeback.el ends here
