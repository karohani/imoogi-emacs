;;; project-notes-public-boundary-test.el --- Public project-notes boundary tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'org)

(defmacro imoogi-project-notes-public-boundary-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename (make-temp-file "imoogi-project-notes-public-" t)))
          (user-emacs-directory (expand-file-name "emacs/" sandbox))
          (imoogi-project-notes-directory (expand-file-name "project-notes/" sandbox))
          (imoogi-project-notes-todo-storage 'project)
          (personal (expand-file-name "notes/" sandbox))
          (root (file-name-as-directory (expand-file-name "source/" sandbox)))
          (registry-file (expand-file-name ".cache/project-notes.json"
                                           user-emacs-directory))
          (mounted-roots-file
           (expand-file-name ".cache/project-notes-mounted-roots.json"
                             user-emacs-directory))
          (org-id-locations-file (expand-file-name "org-id-locations" sandbox))
          (org-agenda-files nil)
          (org-directory personal))
     (make-directory root t)
     (unwind-protect
         (cl-letf (((symbol-function 'imoogi-org--default-directory)
                    (lambda () personal))
                   ((symbol-function 'imoogi-project-notes--registry-file)
                    (lambda () registry-file))
                   ((symbol-function 'imoogi-project-notes--mounted-roots-file)
                    (lambda () mounted-roots-file)))
           (save-window-excursion
             (with-temp-buffer
               (setq default-directory root)
               ,@body)))
       (dolist (buffer (buffer-list))
         (when (and (buffer-file-name buffer)
                    (file-in-directory-p (buffer-file-name buffer) sandbox))
           (with-current-buffer buffer
             (set-buffer-modified-p nil))
           (kill-buffer buffer)))
       (delete-directory sandbox t))))

(defun imoogi-project-notes-public-boundary-test--read (file)
  "Return FILE content."
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(defun imoogi-project-notes-public-boundary-test--goto-heading (title)
  "Move point to TODO heading TITLE."
  (goto-char (point-min))
  (re-search-forward (concat "^\\* TODO " (regexp-quote title) "$"))
  (beginning-of-line))

(defun imoogi-project-notes-public-boundary-test--hash (text)
  "Return stable hash for TEXT."
  (secure-hash 'sha1 text))

(ert-deftest imoogi-project-notes-create-artifact-rejects-dirty-source-before-file-or-id ()
  (imoogi-project-notes-public-boundary-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (original-task-text (imoogi-project-notes-public-boundary-test--read tasks)))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Dirty create\n")
      (imoogi-project-notes-public-boundary-test--goto-heading "Dirty create")
      (should-error (imoogi-project-notes-create-artifact 'design "Dirty Design")
                    :type 'user-error)
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (org-entry-get (point) "ID"))
      (should-not (directory-files (expand-file-name "artifacts/" directory)
                                   nil "\\.org\\'" t))
      (should (equal original-task-text
                     (imoogi-project-notes-public-boundary-test--read tasks))))))

(ert-deftest imoogi-project-notes-create-artifact-rejects-unknown-stable-owner-before-mutation ()
  (imoogi-project-notes-public-boundary-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (artifact-dir (expand-file-name "artifacts/" directory)))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Unknown owner create\n:PROPERTIES:\n:IMOOGI_PROJECT_ID: UNKNOWN-PROJECT\n:END:\n")
      (save-buffer)
      (imoogi-project-notes-public-boundary-test--goto-heading
       "Unknown owner create")
      (let ((source-before (buffer-string))
            (disk-before (imoogi-project-notes-public-boundary-test--read tasks)))
        (should-error (imoogi-project-notes-create-artifact 'design "Unknown Owner Design")
                      :type 'user-error)
        (should-not imoogi-project-notes--pending-link-operation)
        (should (equal source-before (buffer-string)))
        (should (equal disk-before
                       (imoogi-project-notes-public-boundary-test--read tasks)))
        (should-not (org-entry-get (point) "ID"))
        (should-not (directory-files artifact-dir nil "\\.org\\'" t))))))

(ert-deftest imoogi-project-notes-create-artifact-save-failure-keeps-retry-and-artifact ()
  (imoogi-project-notes-public-boundary-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency)
           artifact)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Retry create\n")
      (save-buffer)
      (imoogi-project-notes-public-boundary-test--goto-heading "Retry create")
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (&rest args)
                   (if fail-once
                       (progn
                         (setq fail-once nil)
                         (signal 'file-error '("create prewrite boom")))
                     (apply original-classifier args)))))
        (should-error (imoogi-project-notes-create-artifact 'design "Retry Design")
                      :type 'file-error))
      (should imoogi-project-notes--pending-link-operation)
      (setq artifact (alist-get 'doc-file
                                imoogi-project-notes--pending-link-operation))
      (should (and artifact (file-exists-p artifact)))
      (should-not (string-match-p "Retry Design"
                                  (imoogi-project-notes-public-boundary-test--read
                                   tasks)))
      (imoogi-project-notes-retry-link-operation)
      (should-not imoogi-project-notes--pending-link-operation)
      (should (string-match-p "Retry Design"
                              (imoogi-project-notes-public-boundary-test--read
                               tasks))))))

(ert-deftest imoogi-project-notes-insert-link-rejects-unknown-stable-owner-before-mutation ()
  (imoogi-project-notes-public-boundary-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/unknown-owner.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Unknown Owner Doc\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Unknown owner insert\n:PROPERTIES:\n:IMOOGI_PROJECT_ID: UNKNOWN-PROJECT\n:END:\nBody before \n")
      (save-buffer)
      (imoogi-project-notes-public-boundary-test--goto-heading
       "Unknown owner insert")
      (let ((source-before (buffer-string))
            (doc-before (imoogi-project-notes-public-boundary-test--read doc)))
        (should-error (imoogi-project-notes-insert-link doc)
                      :type 'user-error)
        (should-not imoogi-project-notes--pending-link-operation)
        (should (equal source-before (buffer-string)))
        (should (equal doc-before
                       (imoogi-project-notes-public-boundary-test--read doc)))
        (should-not (string-match-p ":ID:" doc-before))))))

(ert-deftest imoogi-project-notes-insert-link-rejects-dirty-target-before-source-insert ()
  (imoogi-project-notes-public-boundary-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (source (expand-file-name "project.org" directory))
           (doc (expand-file-name "references/dirty-target.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Dirty Target\n\nBody\n"))
      (find-file-noselect doc)
      (with-current-buffer (find-buffer-visiting doc)
        (goto-char (point-max))
        (insert "\nUnsaved target edit\n"))
      (find-file source)
      (goto-char (point-max))
      (insert "\nBefore: ")
      (let ((source-before (buffer-string))
            (doc-before (imoogi-project-notes-public-boundary-test--read doc)))
        (should-error (imoogi-project-notes-insert-link doc)
                      :type 'user-error)
        (should-not imoogi-project-notes--pending-link-operation)
        (should (equal source-before (buffer-string)))
        (should (equal doc-before
                       (imoogi-project-notes-public-boundary-test--read doc)))))))

(ert-deftest imoogi-project-notes-insert-link-rejects-stale-target-before-id ()
  (imoogi-project-notes-public-boundary-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (source (expand-file-name "project.org" directory))
           (doc (expand-file-name "references/stale-target.org" directory))
           (disk-text "#+TITLE: Stale Target\n\nDisk edit\n"))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Stale Target\n\nOld buffer\n"))
      (find-file-noselect doc)
      (with-temp-file doc
        (insert disk-text))
      (find-file source)
      (goto-char (point-max))
      (insert "\nBefore: ")
      (let ((source-before (buffer-string)))
        (should-error (imoogi-project-notes-insert-link doc)
                      :type 'user-error)
        (should-not imoogi-project-notes--pending-link-operation)
        (should (equal source-before (buffer-string)))
        (should (equal disk-text
                       (imoogi-project-notes-public-boundary-test--read doc)))))))

(ert-deftest imoogi-project-notes-insert-link-rejects-unowned-source ()
  (imoogi-project-notes-public-boundary-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (outside (expand-file-name "outside.org" sandbox))
           (doc (expand-file-name "references/owned.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Owned\n"))
      (find-file outside)
      (insert "#+TITLE: Outside\n\n")
      (should-error (imoogi-project-notes-insert-link doc)
                    :type 'user-error)
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (string-match-p "Owned" (buffer-string))))))

(ert-deftest imoogi-project-notes-insert-link-central-source-rejects-wrong-entry ()
  (imoogi-project-notes-public-boundary-test--isolated
    (setq imoogi-project-notes-todo-storage 'central)
    (let* ((dir-a (imoogi-project-notes-setup root))
           (other-root (file-name-as-directory
                        (expand-file-name "other-source/" sandbox)))
           (dir-b (progn
                    (make-directory other-root t)
                    (imoogi-project-notes-setup other-root)))
           (entry-a (imoogi-project-notes--find-entry-by-notes-directory dir-a))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (agenda (expand-file-name "agenda.org" personal))
           (doc-a (expand-file-name "references/a.org" dir-a)))
      (make-directory (file-name-directory doc-a) t)
      (with-temp-file doc-a
        (insert "#+TITLE: A Doc\n"))
      (find-file agenda)
      (goto-char (point-max))
      (insert "\n* TODO Central B\n:PROPERTIES:\n:IMOOGI_PROJECT_ID: "
              (imoogi-project-notes--entry-derived-note-id entry-b)
              "\n:END:\n")
      (save-buffer)
      (imoogi-project-notes-public-boundary-test--goto-heading "Central B")
      (should-error (imoogi-project-notes-insert-link doc-a entry-a)
                    :type 'user-error)
      (should-not (string-match-p "A Doc" (buffer-string))))))

(ert-deftest imoogi-project-notes-insert-link-central-body-cursor-inserts-at-point ()
  (imoogi-project-notes-public-boundary-test--isolated
    (setq imoogi-project-notes-todo-storage 'central)
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (imoogi-project-notes--find-entry-by-notes-directory directory))
           (agenda (expand-file-name "agenda.org" personal))
           (doc (expand-file-name "references/body-cursor.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Body Cursor Doc\n\nBody\n"))
      (find-file agenda)
      (goto-char (point-max))
      (insert "\n* TODO Central Body\n:PROPERTIES:\n:IMOOGI_PROJECT_ID: "
              (imoogi-project-notes--entry-derived-note-id entry)
              "\n:END:\nBody before CURSOR after\n")
      (search-backward "CURSOR")
      (delete-region (point) (+ (point) (length "CURSOR")))
      (set-mark (line-beginning-position))
      (setq mark-active t
            transient-mark-mode t)
      (imoogi-project-notes-insert-link doc)
      (let* ((text (buffer-string))
             (heading-pos (string-match-p
                           (rx line-start "* TODO Central Body" line-end)
                           text))
             (link-pos (string-match-p
                        (rx "[[id:" (+ (not (any "]")))
                            "][Body Cursor Doc]]")
                        text)))
        (should heading-pos)
        (should link-pos)
        (should (> link-pos heading-pos))
        (should (string-match-p
                 (rx "Body before [[id:" (+ (not (any "]")))
                     "][Body Cursor Doc]] after")
                 text))))))


(ert-deftest imoogi-project-notes-insert-link-rejects-duplicate-target-id ()
  (imoogi-project-notes-public-boundary-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (source (expand-file-name "project.org" directory))
           (doc-a (expand-file-name "references/a.org" directory))
           (doc-b (expand-file-name "references/b.org" directory)))
      (make-directory (file-name-directory doc-a) t)
      (with-temp-file doc-a
        (insert ":PROPERTIES:\n:ID:       DUPLICATE-DOC\n:END:\n\n#+TITLE: A\n"))
      (with-temp-file doc-b
        (insert ":PROPERTIES:\n:ID:       DUPLICATE-DOC\n:END:\n\n#+TITLE: B\n"))
      (find-file source)
      (goto-char (point-max))
      (insert "\nBefore: ")
      (let ((source-before (buffer-string)))
        (should-error (imoogi-project-notes-insert-link doc-a)
                      :type 'user-error)
        (should-not imoogi-project-notes--pending-link-operation)
        (should (equal source-before (buffer-string)))))))

(ert-deftest imoogi-project-notes-insert-link-target-save-failure-keeps-retry-without-source_insert ()
  (imoogi-project-notes-public-boundary-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (source (expand-file-name "project.org" directory))
           (doc (expand-file-name "references/retry-target.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Retry Target\n\nBody\n"))
      (find-file source)
      (goto-char (point-max))
      (insert "\nBefore: ")
      (let ((source-before (buffer-string))
            (doc-before (imoogi-project-notes-public-boundary-test--read doc)))
        (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                   (lambda (&rest args)
                     (if fail-once
                         (progn
                           (setq fail-once nil)
                           (signal 'file-error '("insert prewrite boom")))
                       (apply original-classifier args)))))
          (should-error (imoogi-project-notes-insert-link doc)
                        :type 'file-error))
        (should imoogi-project-notes--pending-link-operation)
        (should (equal source-before (buffer-string)))
        (should (equal doc-before
                       (imoogi-project-notes-public-boundary-test--read doc)))
        (should (equal 'insert-link
                       (alist-get 'operation
                                  imoogi-project-notes--pending-link-operation)))
        (should (equal doc
                       (alist-get 'doc-file
                                  imoogi-project-notes--pending-link-operation)))))))

(ert-deftest imoogi-project-notes-insert-link-target-after-save-error-is-classified_saved ()
  (imoogi-project-notes-public-boundary-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (source (expand-file-name "project.org" directory))
           (doc (expand-file-name "references/hook-target.org" directory))
           (warning-minimum-level :emergency))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Hook Target\n\nBody\n"))
      (with-current-buffer (find-file-noselect doc)
        (add-hook 'after-save-hook
                  (lambda () (error "insert postwrite boom"))
                  nil t))
      (find-file source)
      (goto-char (point-max))
      (insert "\nBefore: ")
      (imoogi-project-notes-insert-link doc)
      (should-not imoogi-project-notes--pending-link-operation)
      (should (string-match-p "\\[\\[id:[^]]+\\]\\[Hook Target\\]\\]"
                              (buffer-string)))
      (should (string-match-p ":ID:"
                              (imoogi-project-notes-public-boundary-test--read
                               doc))))))

;;; project-notes-public-boundary-test.el ends here
