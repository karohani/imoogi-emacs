;;; project-notes-cli-cache-test.el --- Project notes CLI cache adapter tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'org)

(defmacro imoogi-project-notes-cli-cache-unit-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename
                    (make-temp-file "imoogi-project-notes-cli-cache-unit-" t)))
          (source-root (file-name-as-directory
                        (expand-file-name "source/" sandbox)))
          (notes-root (file-name-as-directory
                       (expand-file-name "project-notes/261003-source/"
                                         sandbox)))
          (tasks-file (expand-file-name "tasks.org" notes-root))
          (entry (imoogi-project-notes--entry
                  "dir:source" source-root notes-root 'project)))
     (make-directory source-root t)
     (make-directory notes-root t)
     (with-temp-file tasks-file
       (insert "* Tasks\n"))
     (setf (alist-get 'tasks-file entry) tasks-file)
     (clrhash imoogi-project-notes--snapshot-live-record-cache)
     (unwind-protect
         (cl-letf (((symbol-function 'imoogi-project-notes--all-entries)
                    (lambda () (list entry))))
           ,@body)
       (clrhash imoogi-project-notes--snapshot-live-record-cache)
       (dolist (buffer (buffer-list))
         (when (and (buffer-file-name buffer)
                    (file-in-directory-p (buffer-file-name buffer) sandbox))
           (with-current-buffer buffer
             (set-buffer-modified-p nil))
           (kill-buffer buffer)))
       (delete-directory sandbox t))))


(defun imoogi-project-notes-cli-cache-unit-test--binary ()
  "Return the local imoogi-notes binary, or skip."
  (let* ((configured (getenv "IMOOGI_NOTES_BIN"))
         (root (or (and (boundp 'imoogi-test-root) imoogi-test-root)
                   default-directory))
         (local (expand-file-name "bin/imoogi-notes" root))
         (binary (or (and configured
                          (> (length configured) 0)
                          (executable-find configured))
                     (and (file-executable-p local) local)
                     (executable-find "imoogi-notes"))))
    (unless binary
      (ert-skip "imoogi-notes binary is not available"))
    binary))

(defun imoogi-project-notes-cli-cache-unit-test--read (file)
  "Return FILE content."
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(ert-deftest imoogi-project-notes-cli-cache-live-document-id-overlays-snapshot ()
  (imoogi-project-notes-cli-cache-unit-test--isolated
    (let* ((doc-file (expand-file-name "references/live.org" notes-root))
           (snapshot `((documents . [((file . ,doc-file))])
                       (occurrences . nil))))
      (make-directory (file-name-directory doc-file) t)
      (with-temp-file doc-file
        (insert "#+TITLE: Live Doc\n"))
      (with-current-buffer (find-file-noselect doc-file)
        (org-mode)
        (goto-char (point-min))
        (insert ":PROPERTIES:\n:ID:       LIVE-DOC-ID\n:END:\n\n"))
      (let ((imoogi-project-notes--document-cache-snapshot snapshot))
        (should (equal doc-file
                       (alist-get
                        'file
                        (imoogi-project-notes--document-by-id
                         entry "LIVE-DOC-ID"))))
        (let ((occurrences (imoogi-project-notes--project-id-occurrences
                            entry "LIVE-DOC-ID")))
          (should (= 1 (length occurrences)))
          (should (equal doc-file (caar occurrences)))
          (should (integerp (cdar occurrences))))))))


(ert-deftest imoogi-project-notes-cli-cache-live-hidden-and-nested-ids-ignored ()
  (imoogi-project-notes-cli-cache-unit-test--isolated
    (let* ((doc-file (expand-file-name "references/live.org" notes-root))
           (hidden-file (expand-file-name "assets/hidden.org" notes-root))
           (nested-root (file-name-as-directory
                         (expand-file-name "references/nested/" notes-root)))
           (nested-file (expand-file-name "nested.org" nested-root))
           (snapshot `((documents . [((file . ,doc-file)
                                      (id . "VISIBLE-ID")
                                      (title . "Visible")
                                      (kind . "file"))])
                       (occurrences .
                                    ((VISIBLE-ID .
                                                 [((file . ,doc-file)
                                                   (position . 24))]))))))
      (make-directory (file-name-directory doc-file) t)
      (make-directory (file-name-directory hidden-file) t)
      (make-directory nested-root t)
      (with-temp-file doc-file
        (insert ":PROPERTIES:\n:ID:       VISIBLE-ID\n:END:\n#+TITLE: Visible\n"))
      (with-temp-file hidden-file
        (insert "* Hidden\n:PROPERTIES:\n:ID:       VISIBLE-ID\n:END:\n"))
      (with-temp-file (expand-file-name imoogi-project-notes--metadata-file-name
                                        nested-root)
        (insert "{}\n"))
      (with-temp-file nested-file
        (insert "* Nested\n:PROPERTIES:\n:ID:       VISIBLE-ID\n:END:\n"))
      (dolist (file (list hidden-file nested-file))
        (with-current-buffer (find-file-noselect file)
          (org-mode)))
      (let ((imoogi-project-notes--document-cache-snapshot snapshot))
        (should (equal doc-file
                       (alist-get
                        'file
                        (imoogi-project-notes--document-by-id
                         entry "VISIBLE-ID"))))))))

(ert-deftest imoogi-project-notes-cli-cache-validates-target-against-current-disk ()
  (imoogi-project-notes-cli-cache-unit-test--isolated
    (let* ((outside-root (file-name-as-directory
                          (expand-file-name "outside/" sandbox)))
           (outside-file (expand-file-name "escaped.org" outside-root))
           (link-file (expand-file-name "references/escaped.org" notes-root))
           (snapshot `((documents . [((file . ,link-file)
                                      (id . "ESCAPED-ID")
                                      (title . "Escaped")
                                      (kind . "file"))])
                       (occurrences . nil))))
      (make-directory outside-root t)
      (make-directory (file-name-directory link-file) t)
      (with-temp-file outside-file
        (insert "#+TITLE: Outside\n"))
      (make-symbolic-link outside-file link-file)
      (with-current-buffer (find-file-noselect tasks-file)
        (org-mode)
        (let ((imoogi-project-notes--document-cache-snapshot snapshot))
          (should-error
           (imoogi-project-notes--validate-document-target-file
            entry link-file "산출물 연결")
           :type 'user-error))))))


(ert-deftest imoogi-project-notes-cli-cache-async-link-writes-without-fullscan ()
  (imoogi-project-notes-cli-cache-unit-test--isolated
    (let* ((binary (imoogi-project-notes-cli-cache-unit-test--binary))
           (doc-file (expand-file-name "references/async-write.org" notes-root))
           (org-id-locations-file (expand-file-name "org-id-locations" sandbox))
           (imoogi-project-notes-command (list binary))
           (imoogi-project-notes-cache-directory
            (expand-file-name "cli-cache/" sandbox)))
      (make-directory (file-name-directory doc-file) t)
      (make-directory imoogi-project-notes-cache-directory t)
      (with-temp-file doc-file
        (insert "#+TITLE: Async Doc\n\nBody.\n"))
      (with-temp-file tasks-file
        (insert "* TODO Async Task\n"))
      (cl-letf (((symbol-function 'imoogi-project-notes--all-entries)
                 (lambda () (list entry)))
                ((symbol-function 'imoogi-project-notes--project-document-files)
                 (lambda (&rest _)
                   (error "project document fullscan fallback was used")))
                ((symbol-function 'completing-read)
                 (lambda (_prompt choices &rest _)
                   (caar choices))))
        (let ((task-buffer (find-file-noselect tasks-file)))
          (switch-to-buffer task-buffer)
          (org-mode)
          (goto-char (point-min))
          (imoogi-project-notes--link-artifact-async entry)
          (let ((deadline (+ (float-time) 5.0)))
            (while (and (< (float-time) deadline)
                        imoogi-project-notes--pending-cache-process)
              (accept-process-output nil 0.05)))
          (should-not imoogi-project-notes--pending-cache-process)))
      (let* ((task-text
              (imoogi-project-notes-cli-cache-unit-test--read tasks-file))
             (doc-text
              (imoogi-project-notes-cli-cache-unit-test--read doc-file))
             (doc-id (progn
                       (should (string-match
                                ":ID:[[:space:]]+\\([^[:space:]\n]+\\)"
                                doc-text))
                       (match-string 1 doc-text)))
             (task-id (progn
                        (should (string-match
                                 ":ID:[[:space:]]+\\([^[:space:]\n]+\\)"
                                 task-text))
                        (match-string 1 task-text))))
        (should (string-match-p
                 (regexp-quote (format "[[id:%s][Async Doc]]" doc-id))
                 task-text))
        (should (string-match-p "^\\* Link$" doc-text))
        (should (string-match-p
                 (regexp-quote (format "[[id:%s][Async Task]]" task-id))
                 doc-text))))))

(ert-deftest imoogi-project-notes-cli-cache-live-task-id-overlays-snapshot ()
  (imoogi-project-notes-cli-cache-unit-test--isolated
    (let* ((doc-file (expand-file-name "references/live.org" notes-root))
           (snapshot `((documents . [((file . ,doc-file)
                                      (id . "DUPLICATE-LIVE-ID")
                                      (title . "Live Doc")
                                      (kind . "file"))])
                       (occurrences .
                                    ((DUPLICATE-LIVE-ID .
                                                        [((file . ,doc-file)
                                                          (position . 24))]))))))
      (make-directory (file-name-directory doc-file) t)
      (with-temp-file doc-file
        (insert ":PROPERTIES:\n:ID:       DUPLICATE-LIVE-ID\n:END:\n#+TITLE: Live Doc\n"))
      (with-current-buffer (find-file-noselect tasks-file)
        (org-mode)
        (goto-char (point-max))
        (insert "\n* TODO Live task\n:PROPERTIES:\n:ID:       DUPLICATE-LIVE-ID\n:END:\n"))
      (let ((imoogi-project-notes--document-cache-snapshot snapshot))
        (should-error
         (imoogi-project-notes--document-by-id entry "DUPLICATE-LIVE-ID")
         :type 'user-error)))))

;;; project-notes-cli-cache-test.el ends here
