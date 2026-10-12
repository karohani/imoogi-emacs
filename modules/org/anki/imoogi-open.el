;;; imoogi-open.el --- Open the Org heading an Anki card came from -*- lexical-binding: t; -*-

;;; Commentary:

;; SPEC-ANKISRC-001's open side.  A card's "Emacs에서 원본 열기" link runs
;; `imoogi-anki open <url>', which validates the ID and evaluates
;;
;;   (imoogi-anki-open-id "<ID>")
;;
;; through emacsclient.  This file defines that receiver.  It is required
;; by modules/org/24-anki.el, so it is loaded with the Anki module rather
;; than by an autoload cookie the module loader never harvests
;; (REQ-ASRC-011.7).
;;
;; Resolution (REQ-ASRC-011, plan.md DD-5):
;;
;;  1. The ID is checked against the shared character set again.
;;  2. Every sync-target file (the legacy root and the registered targets,
;;     exclusions applied) is searched for a heading whose OWN :ID: is the
;;     ID, plus the file `org-id-locations' names when that file lies
;;     outside the targets.  The search is textual first -- one regexp per
;;     file, tolerant of trailing whitespace -- and only a hit is checked
;;     with Org for heading-locality, which keeps a click cheap (plan R-7).
;;     A file the user is visiting is searched in that buffer.
;;  3. Two or more headings: list them and move nowhere.  Any target that
;;     could not be read -- a file or a directory -- leaves uniqueness
;;     unproven, so fewer than two found plus an unread target lists the
;;     unread paths and moves nowhere either ("incomplete").  One heading:
;;     check the file again, go there, then register its location; if it
;;     vanished meanwhile, treat it as none.  None: say so and point at
;;     `imoogi-sync'.  Every outcome raises the Emacs frame.
;;
;; The receiver never creates a file, never saves, reverts, or edits a
;; user buffer, and never writes :ID: or ANKI_NOTE_ID.  The only thing it
;; writes is `org-id-locations-file' (REQ-ASRC-012).

;;; Code:

(require 'cl-lib)
(require 'org)
(require 'org-id)
(require 'imoogi-orgid)
(require 'imoogi-scan)
(require 'imoogi-targets)
(require 'imoogi-target-scan)

(defvar imoogi-sync-root)
(defvar imoogi-exclude-patterns)

(defconst imoogi-anki-open-candidates-buffer "*imoogi-anki-open*"
  "Buffer listing the headings that share one Org ID.")

(defun imoogi-anki-open--focus-frame ()
  "Bring the selected Emacs frame forward and give it input focus.
The click comes from Anki, so without this the answer -- an opened
heading, a not-found message, a candidate list -- would sit behind it
\(REQ-ASRC-011.8)."
  (let ((frame (selected-frame)))
    (when (frame-live-p frame)
      (ignore-errors (make-frame-visible frame))
      (ignore-errors (raise-frame frame))
      (ignore-errors (select-frame-set-input-focus frame)))))

(defun imoogi-anki-open--confirmed-absent-p (path)
  "Non-nil when PATH is known not to exist.
Existence predicates also answer nil when an ancestor denies traversal,
so absence counts only when the nearest reachable ancestor directory can
be listed and shows the missing component.  Anything else is unknown."
  (let* ((path (directory-file-name (expand-file-name path)))
         (parent (file-name-directory path)))
    (cond ((file-exists-p path) nil)
          ((or (null parent) (equal (directory-file-name parent) path)) nil)
          ((file-accessible-directory-p parent) t)
          (t (imoogi-anki-open--confirmed-absent-p parent)))))

(defun imoogi-anki-open--target-files ()
  "Return (FILES . FAILED) for the sync targets.
FILES are the truenames of every sync-target file, without duplicates.
The legacy root and registered directories are walked with the same
discovery the sync uses; registered files are taken as they are; a file
matching `imoogi-exclude-patterns' relative to its root is left out.
FAILED are the directories discovery could not walk and the targets
whose existence could not be determined (see
`imoogi-anki-open--confirmed-absent-p').  A target confirmed not to
exist holds nothing to check and is not a failure."
  (let ((seen (make-hash-table :test #'equal))
        (patterns (and (boundp 'imoogi-exclude-patterns) imoogi-exclude-patterns))
        (targets (imoogi-targets-load))
        roots files failed)
    (when (and (boundp 'imoogi-sync-root) imoogi-sync-root)
      (push imoogi-sync-root roots))
    (dolist (target targets)
      (when (eq (plist-get target :kind) 'directory)
        (push (plist-get target :path) roots)))
    (cl-flet ((add (file root)
                (let ((true (ignore-errors (file-truename file))))
                  (when (and true
                             (not (gethash true seen))
                             (not (imoogi-scan--excluded-p
                                   (file-relative-name file root) patterns)))
                    (puthash true t seen)
                    (push true files)))))
      (dolist (root (nreverse roots))
        (cond
         ((file-directory-p root)
          (let* ((true-root (imoogi-target-scan--true-directory root))
                 (scan (imoogi-target-scan--discover-directory
                        true-root true-root (make-hash-table :test #'equal))))
            (dolist (file (plist-get scan :files))
              (add file true-root))
            (setq failed (append failed (plist-get scan :failures)))))
         ((not (imoogi-anki-open--confirmed-absent-p root))
          (push (directory-file-name (expand-file-name root)) failed))))
      (dolist (target targets)
        (let ((path (plist-get target :path)))
          (when (eq (plist-get target :kind) 'file)
            (cond
             ((file-regular-p path)
              (add path (file-name-directory (expand-file-name path))))
             ((not (imoogi-anki-open--confirmed-absent-p path))
              (push (expand-file-name path) failed)))))))
    (cons (nreverse files) (delete-dups failed))))

(defun imoogi-anki-open--id-line-regexp (id)
  "Return the regexp for a property line naming ID, trailing blanks allowed."
  (concat "^[ \t]*:[Ii][Dd]:[ \t]+" (regexp-quote id) "[ \t]*$"))

(defun imoogi-anki-open--scan-current-buffer (id)
  "Return (POSITION . TITLE) for each heading here whose own :ID: is ID."
  (let ((case-fold-search nil)
        (regexp (imoogi-anki-open--id-line-regexp id))
        hits)
    (save-excursion
      (save-restriction
        (widen)
        (goto-char (point-min))
        (when (re-search-forward regexp nil t)
          (unless (derived-mode-p 'org-mode)
            (delay-mode-hooks (org-mode)))
          (goto-char (point-min))
          (while (re-search-forward regexp nil t)
            (save-excursion
              (when (and (ignore-errors (org-back-to-heading t) t)
                         (equal (imoogi-orgid-local-id) id)
                         (not (assq (point) hits)))
                (push (cons (point) (org-get-heading t t t t)) hits)))))))
    (nreverse hits)))

(defun imoogi-anki-open--scan-file (id file)
  "Return the ID headings in FILE, or the symbol `failed'.
Each heading is (:file FILE :pos POS :title TITLE).  A buffer visiting
FILE in Org mode is searched as it is, unsaved edits included; otherwise
a temporary copy is read, so no buffer is created and nothing the user
sees changes.  A file that vanished since discovery holds nothing; one
that cannot be read is `failed', since it may hold the ID."
  (let* ((visiting (find-buffer-visiting file))
         (hits
          (condition-case nil
              (if (and visiting (with-current-buffer visiting (derived-mode-p 'org-mode)))
                  (with-current-buffer visiting
                    (imoogi-anki-open--scan-current-buffer id))
                (with-temp-buffer
                  (if visiting
                      (insert-buffer-substring visiting)
                    (insert-file-contents file))
                  (imoogi-anki-open--scan-current-buffer id)))
            (file-missing nil)
            (file-error 'failed))))
    (if (eq hits 'failed)
        'failed
      (mapcar (lambda (hit) (list :file file :pos (car hit) :title (cdr hit))) hits))))

(defun imoogi-anki-open--cached-file (id)
  "Return the existing truename `org-id-locations' records for ID, or nil."
  (when org-id-track-globally
    (unless org-id-locations (org-id-locations-load))
    (let ((file (and (hash-table-p org-id-locations) (gethash id org-id-locations))))
      (when (and file (file-regular-p (expand-file-name file)))
        (file-truename (expand-file-name file))))))

(defun imoogi-anki-open--candidates (id)
  "Return (:candidates HEADINGS :failed PATHS) for ID over the targets and cache.
HEADINGS are every heading whose own :ID: is ID.  PATHS are the
directories and files that could not be checked; when there are any,
the uniqueness of HEADINGS is not established."
  (let* ((discovery (imoogi-anki-open--target-files))
         (files (car discovery))
         (failed (cdr discovery))
         (cached (imoogi-anki-open--cached-file id))
         candidates)
    ;; O3: a cached file outside the targets joins the search, so the
    ;; uniqueness check covers the targets plus that one file.
    (when (and cached (not (member cached files)))
      (setq files (append files (list cached))))
    (dolist (file files)
      (let ((hits (imoogi-anki-open--scan-file id file)))
        (if (eq hits 'failed)
            (setq failed (append failed (list file)))
          (setq candidates (append candidates hits)))))
    (list :candidates candidates :failed failed)))

(defun imoogi-anki-open--register (id file)
  "Record FILE as ID's location and save the cache when it changed."
  (when org-id-track-globally
    (let ((before (and (hash-table-p org-id-locations) (gethash id org-id-locations))))
      (unless (and before (equal (ignore-errors (file-truename (expand-file-name before)))
                                 (file-truename file)))
        (imoogi-orgid-register id file)
        (imoogi-orgid-save)))))

(defun imoogi-anki-open--goto (id candidate)
  "Visit CANDIDATE's file and put point on ID's heading there.
Return non-nil when the heading was reached.  The file is checked again
first, because it may have changed since the scan: when it is gone or
no longer holds ID, nothing is shown, no file is created, and a buffer
this function opened for it is killed again."
  (let* ((file (plist-get candidate :file))
         (visiting (find-buffer-visiting file))
         (buffer (or visiting
                     (and (file-regular-p file)
                          (condition-case nil
                              (find-file-noselect file)
                            (file-error nil)))))
         (pos (and (buffer-live-p buffer)
                   (with-current-buffer buffer
                     (car (car (imoogi-anki-open--scan-current-buffer id)))))))
    (if (not pos)
        (progn
          (when (and buffer (not visiting) (buffer-live-p buffer))
            (kill-buffer buffer))
          nil)
      (pop-to-buffer-same-window buffer)
      (when (or (< pos (point-min)) (> pos (point-max)))
        (widen))
      (goto-char pos)
      (org-fold-show-context 'link-search)
      t)))

(defun imoogi-anki-open--not-found (id)
  "Say that no heading carries ID."
  (message "imoogi-anki: Org ID %s의 원본 heading을 찾을 수 없습니다. 파일이 옮겨졌거나 삭제·ID 변경되었을 수 있습니다. 확인 후 M-x imoogi-sync 로 다시 동기화하세요."
           id))

(defun imoogi-anki-open--show-listing (header candidates failed)
  "Show HEADER, CANDIDATES and the unchecked FAILED paths in a listing buffer.
The buffer is `imoogi-anki-open-candidates-buffer'."
  (with-current-buffer (get-buffer-create imoogi-anki-open-candidates-buffer)
    (let ((inhibit-read-only t))
      (erase-buffer)
      (insert header "\n")
      (dolist (candidate candidates)
        (insert "  ")
        (insert-text-button (plist-get candidate :file)
                            'follow-link t
                            'imoogi-file (plist-get candidate :file)
                            'imoogi-pos (plist-get candidate :pos)
                            'action (lambda (button)
                                      (find-file (button-get button 'imoogi-file))
                                      (goto-char (button-get button 'imoogi-pos))))
        (insert (format "  %s\n" (plist-get candidate :title))))
      (when failed
        (insert "\n확인하지 못한 경로:\n")
        (dolist (path failed)
          (insert (format "  %s\n" path)))))
    (special-mode)
    (goto-char (point-min))
    (display-buffer (current-buffer))))

(defun imoogi-anki-open--show-candidates (id candidates failed)
  "List CANDIDATES sharing ID, plus the unchecked FAILED paths."
  (imoogi-anki-open--show-listing
   (concat (format "imoogi-anki: Org ID %s 를 가진 heading이 %d개입니다. 어느 것으로도 이동하지 않았습니다.\n"
                   id (length candidates))
           "하나만 남기고 나머지의 :ID: 줄을 지운 뒤 M-x imoogi-sync 로 다시 동기화하세요.\n")
   candidates failed))

(defun imoogi-anki-open--show-incomplete (id candidates failed)
  "Explain that FAILED could not be checked for ID, listing CANDIDATES found."
  (imoogi-anki-open--show-listing
   (concat (format "imoogi-anki: 읽지 못한 경로가 %d개 있어 Org ID %s 가 하나뿐인지 확인할 수 없습니다. 어느 것으로도 이동하지 않았습니다.\n"
                   (length failed) id)
           "아래 경로의 읽기 권한을 확인한 뒤 링크를 다시 누르세요.\n")
   candidates failed))

;;;###autoload
(defun imoogi-anki-open-id (id)
  "Open the Org heading whose own :ID: is ID; return the outcome as a string.

\"ok\" when exactly one heading was found and opened, \"not-found\" when
none was (or the one found vanished before it could be opened),
\"duplicate\" when several were (they are listed, and point does not
move), and \"incomplete\" when fewer than two were found but some target
could not be read, so a duplicate cannot be ruled out (the unread paths
are listed, and point does not move).  An ID outside the shared
character set returns \"invalid-id\" without touching any file.  The
`imoogi-anki open' handler treats all but \"invalid-id\" as normal."
  (if (not (imoogi-orgid-valid-p id))
      (progn
        (message "imoogi-anki: 올바르지 않은 Org ID 라서 열지 않았습니다: %S" id)
        "invalid-id")
    (let* ((result (imoogi-anki-open--candidates id))
           (candidates (plist-get result :candidates))
           (failed (plist-get result :failed)))
      (prog1
          (cond
           ((cdr candidates)
            (imoogi-anki-open--show-candidates id candidates failed)
            "duplicate")
           (failed
            (imoogi-anki-open--show-incomplete id candidates failed)
            "incomplete")
           ((and candidates (imoogi-anki-open--goto id (car candidates)))
            (imoogi-anki-open--register id (plist-get (car candidates) :file))
            "ok")
           (t
            (imoogi-anki-open--not-found id)
            "not-found"))
        (imoogi-anki-open--focus-frame)))))

(provide 'imoogi-open)
;;; imoogi-open.el ends here
