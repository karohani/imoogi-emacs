;;; imoogi-scan.el --- Recursive sync-root traversal -*- lexical-binding: t; -*-

;;; Commentary:

;; Walks a sync root recursively for .org files, to unlimited depth, in
;; deterministic (sorted) order, and reads every file under it -- an
;; excluded file included.  The traversal itself is never filtered;
;; `imoogi-exclude-patterns' bounds only what it is allowed to produce as
;; a sync target (design.md SS3 step 2, plan.md D-13).
;;
;; Two outputs, from the same read:
;;
;;  - entries[] -- sync targets, from non-excluded files only.  A
;;    heading whose own PROPERTIES drawer carries ANKI_NOTE_TYPE.
;;  - census[]  -- every ANKI_NOTE_ID occurrence found in any heading's
;;    own drawer, excluded files included, in scan order.  This is a
;;    property-value scan, never a render: a census occurrence never
;;    contributes field content, tags, a deck, or a hash.
;;
;; scan-complete is true only when every .org file under the root was
;; read successfully -- an unreadable file, excluded or not, sets it
;; false for the whole run (REQ-015).

;;; Code:

(require 'org)
(require 'cl-lib)
(require 'imoogi-props)

(defun imoogi-scan--org-files (root)
  "Return the .org files under ROOT, sorted deterministically.

Depth-first: at each directory level, entries (both files and
subdirectories) are sorted alphabetically before recursion, so scan
order does not depend on filesystem enumeration order.  Hidden entries
(names starting with a dot) are skipped."
  (let (files)
    (dolist (name (sort (directory-files root nil "\\`[^.]") #'string<))
      (let ((path (expand-file-name name root)))
        (cond
         ((file-directory-p path)
          (setq files (nconc files (imoogi-scan--org-files path))))
         ((string-suffix-p ".org" path)
          (setq files (nconc files (list path)))))))
    files))

(defun imoogi-scan--excluded-p (relative-path patterns)
  "Return non-nil if RELATIVE-PATH matches any of PATTERNS.

A pattern matches when it appears as a substring of RELATIVE-PATH --
sufficient for the directory-prefix patterns imoogi-exclude-patterns is
documented to carry (e.g. \"drafts/\", \"archive/\")."
  (cl-some (lambda (pattern) (string-match-p (regexp-quote pattern) relative-path))
           patterns))

(defun imoogi-scan--entry-body ()
  "Return the raw Org body text of the entry at point: everything after
the heading's planning/property metadata, up to the next headline at
any level, trimmed of leading/trailing whitespace."
  (save-excursion
    (org-end-of-meta-data t)
    (let ((start (point))
          (end (save-excursion
                 (or (outline-next-heading) (point))
                 (point))))
      (string-trim (buffer-substring-no-properties start end)))))

(defun imoogi-scan--file (file relative-path excluded)
  "Scan FILE (already known to exist and be readable) for sync targets
and census entries.  RELATIVE-PATH is FILE's path relative to the sync
root.  Sync targets are collected only when EXCLUDED is nil; census
entries are always collected, from every heading's own drawer.

Returns a plist (:entries LIST :census LIST)."
  (let (entries census (index 0))
    (with-temp-buffer
      (insert-file-contents file)
      (delay-mode-hooks (org-mode))
      (goto-char (point-min))
      (org-map-entries
       (lambda ()
         (let ((note-id (org-entry-get (point) "ANKI_NOTE_ID" nil))
               (note-type (org-entry-get (point) "ANKI_NOTE_TYPE" nil)))
           (when note-id
             (push (list :note-id (string-to-number note-id)
                          :source-path relative-path)
                   census))
           (when (and note-type (not excluded))
             (let* ((key (format "%s::%d" relative-path index))
                    (title (org-get-heading t t t t))
                    (body (imoogi-scan--entry-body))
                    (deck (imoogi-props-resolve-deck))
                    (tags (imoogi-props-resolve-tags)))
               (push (list :key key
                            :note-id (and note-id (string-to-number note-id))
                            :note-type note-type
                            :source-path relative-path
                            :deck deck
                            :tags tags
                            :title title
                            :body body)
                     entries)
               (setq index (1+ index))))))))
    (list :entries (nreverse entries) :census (nreverse census))))

(defun imoogi-scan-root (root patterns)
  "Recursively scan ROOT for .org files and produce the request-side
scan output.  PATTERNS is `imoogi-exclude-patterns'.

Returns a plist:
  :entries          sync-target plists, non-excluded files only
  :census           census plists, every file including excluded ones
  :scan-complete    t only if every .org file under ROOT was read
  :unreadable-files paths (relative to ROOT) that could not be read"
  (let (all-entries all-census unreadable)
    (dolist (file (imoogi-scan--org-files root))
      (let* ((relative (file-relative-name file root))
             (excluded (imoogi-scan--excluded-p relative patterns)))
        (condition-case nil
            (let ((result (imoogi-scan--file file relative excluded)))
              (setq all-entries (nconc all-entries (plist-get result :entries)))
              (setq all-census (nconc all-census (plist-get result :census))))
          (file-error (push relative unreadable))
          (file-missing (push relative unreadable)))))
    (list :entries all-entries
          :census all-census
          :scan-complete (null unreadable)
          :unreadable-files (nreverse unreadable))))

(provide 'imoogi-scan)
;;; imoogi-scan.el ends here
