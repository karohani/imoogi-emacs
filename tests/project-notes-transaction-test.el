;;; project-notes-transaction-test.el --- Project notes link transaction tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'org)

(defmacro imoogi-project-notes-transaction-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename (make-temp-file "imoogi-project-notes-transaction-" t)))
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
          (imoogi-project-notes--pending-link-operation nil)
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

(defun imoogi-project-notes-transaction-test--read (file)
  "Return FILE content."
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(defun imoogi-project-notes-transaction-test--goto-heading (title)
  "Move point to TODO heading TITLE."
  (goto-char (point-min))
  (re-search-forward (concat "^\\* TODO " (regexp-quote title) "$"))
  (beginning-of-line))

(ert-deftest imoogi-project-notes-insert-link-target-save-failure-retry-saves_doc_id_without_source_insert ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (source (expand-file-name "project.org" directory))
           (doc (expand-file-name "references/insert-retry-target.org"
                                  directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency)
           source-before doc-before doc-after doc-id)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Insert Retry Target\n\nBody\n"))
      (find-file source)
      (goto-char (point-max))
      (insert "\nBefore: ")
      (setq source-before (buffer-string)
            doc-before (imoogi-project-notes-transaction-test--read doc))
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
      (should-not (alist-get 'task-file
                             imoogi-project-notes--pending-link-operation))
      (should (equal source-before (buffer-string)))
      (should (equal doc-before
                     (imoogi-project-notes-transaction-test--read doc)))
      (imoogi-project-notes-retry-link-operation)
      (should-not imoogi-project-notes--pending-link-operation)
      (should (equal source-before (buffer-string)))
      (setq doc-after (imoogi-project-notes-transaction-test--read doc))
      (should (string-match-p ":ID:" doc-after))
      (should (not (equal doc-before doc-after)))
      (with-current-buffer (find-file-noselect doc)
        (setq doc-id (car (imoogi-project-notes--document-identity))))
      (goto-char (point-max))
      (imoogi-project-notes-insert-link doc)
      (should (string-match-p (regexp-quote (format "[[id:%s][Insert Retry Target]]" doc-id))
                              (buffer-string))))))

(ert-deftest imoogi-project-notes-link-rejects-dirty-source-before-id-mutation ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/dirty-source.org" directory))
           (original-task-text (imoogi-project-notes-transaction-test--read tasks)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Dirty Source\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Unsaved source\n")
      (imoogi-project-notes-transaction-test--goto-heading "Unsaved source")
      (should-error (imoogi-project-notes-link-artifact doc)
                    :type 'user-error)
      (should-not imoogi-project-notes--pending-link-operation)
      (should (equal original-task-text
                     (imoogi-project-notes-transaction-test--read tasks)))
      (should-not (org-entry-get (point) "ID")))))

(ert-deftest imoogi-project-notes-link-validation-failure-does-not_create_ids_owner_or_dirty_state ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (outside-doc (expand-file-name "outside.org" sandbox))
           (original-task-text))
      (with-temp-file outside-doc
        (insert "#+TITLE: Outside\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Validation task\n")
      (save-buffer)
      (setq original-task-text
            (imoogi-project-notes-transaction-test--read tasks))
      (imoogi-project-notes-transaction-test--goto-heading "Validation task")
      (should-error (imoogi-project-notes-link-artifact outside-doc)
                    :type 'user-error)
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (org-entry-get (point) "ID"))
      (should-not (org-entry-get (point) "IMOOGI_PROJECT_ID"))
      (should-not (buffer-modified-p))
      (should (equal original-task-text
                     (imoogi-project-notes-transaction-test--read tasks))))))

(ert-deftest imoogi-project-notes-link-rejects-clean-stale-document-buffer-before-overwrite ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/stale-doc.org" directory))
           (disk-doc-text "#+TITLE: Stale Doc\n\nChanged on disk\n"))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Stale Doc\n\nOld buffer\n"))
      (find-file-noselect doc)
      (with-temp-file doc
        (insert disk-doc-text))
      (with-current-buffer (find-buffer-visiting doc)
        (set-visited-file-modtime))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Stale doc task\n")
      (save-buffer)
      (imoogi-project-notes-transaction-test--goto-heading "Stale doc task")
      (should-error (imoogi-project-notes-link-artifact doc)
                    :type 'user-error)
      (should-not imoogi-project-notes--pending-link-operation)
      (should (equal disk-doc-text
                     (imoogi-project-notes-transaction-test--read doc)))
      (should-not (org-entry-get (point) "ID")))))

(ert-deftest imoogi-project-notes-link-rejects-clean-stale-task-buffer-before-overwrite ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/stale-task.org" directory))
           (original-doc-text "#+TITLE: Stale Task Doc\n\nBody\n")
           disk-task-text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert original-doc-text))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Stale task\n")
      (save-buffer)
      (setq disk-task-text
            (concat (imoogi-project-notes-transaction-test--read tasks)
                    "\nDisk-side task edit\n"))
      (with-temp-file tasks
        (insert disk-task-text))
      (imoogi-project-notes-transaction-test--goto-heading "Stale task")
      (should-error (imoogi-project-notes-link-artifact doc)
                    :type 'user-error)
      (should-not imoogi-project-notes--pending-link-operation)
      (should (equal disk-task-text
                     (imoogi-project-notes-transaction-test--read tasks)))
      (should (equal original-doc-text
                     (imoogi-project-notes-transaction-test--read doc)))
      (should-not (org-entry-get (point) "ID")))))

(ert-deftest imoogi-project-notes-link-duplicate-link-heading-preparation-failure-rolls_back_ids_owner_and_dirty_state ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/duplicate-link.org" directory))
           (original-doc-text "#+TITLE: Duplicate Link\n\n* Link\n\n* Link\n")
           original-task-text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert original-doc-text))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Duplicate link prep\n")
      (save-buffer)
      (setq original-task-text
            (imoogi-project-notes-transaction-test--read tasks))
      (imoogi-project-notes-transaction-test--goto-heading
       "Duplicate link prep")
      (should-error (imoogi-project-notes-link-artifact doc)
                    :type 'user-error)
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (org-entry-get (point) "ID"))
      (should-not (org-entry-get (point) "IMOOGI_PROJECT_ID"))
      (should-not (buffer-modified-p (find-buffer-visiting tasks)))
      (should-not (buffer-modified-p (find-buffer-visiting doc)))
      (should (equal original-task-text
                     (imoogi-project-notes-transaction-test--read tasks)))
      (should (equal original-doc-text
                     (imoogi-project-notes-transaction-test--read doc))))))

(ert-deftest imoogi-project-notes-link-public-late-document-link-error-rolls_back_generated_ids_and_owner ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/late-link-error.org" directory))
           original-task-text
           original-doc-text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Late Link Error\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Late link rollback\n")
      (save-buffer)
      (setq original-task-text
            (imoogi-project-notes-transaction-test--read tasks)
            original-doc-text
            (imoogi-project-notes-transaction-test--read doc))
      (imoogi-project-notes-transaction-test--goto-heading
       "Late link rollback")
      (cl-letf (((symbol-function 'imoogi-project-notes--ensure-document-task-link)
                 (lambda (&rest _args)
                   (user-error "injected late document link failure"))))
        (should-error (imoogi-project-notes-link-artifact doc)
                      :type 'user-error))
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (org-entry-get (point) "ID"))
      (should-not (org-entry-get (point) "IMOOGI_PROJECT_ID"))
      (should-not (buffer-modified-p (find-buffer-visiting tasks)))
      (should-not (buffer-modified-p (find-buffer-visiting doc)))
      (should (equal original-task-text
                     (imoogi-project-notes-transaction-test--read tasks)))
      (should (equal original-doc-text
                     (imoogi-project-notes-transaction-test--read doc))))))

(ert-deftest imoogi-project-notes-link-save-failure-rolls-back-unsaved-buffers-and-retries_exact_postimage ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/rollback.org" directory))
           (original-task-text)
           (original-doc-text)
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Rollback Doc\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Rollback task\n")
      (save-buffer)
      (setq original-task-text
            (imoogi-project-notes-transaction-test--read tasks)
            original-doc-text
            (imoogi-project-notes-transaction-test--read doc))
      (imoogi-project-notes-transaction-test--goto-heading "Rollback task")
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (&rest args)
                   (if fail-once
                       (progn
                         (setq fail-once nil)
                         (signal 'file-error '("prewrite boom")))
                     (apply original-classifier args)))))
        (should-error (imoogi-project-notes-link-artifact doc)
                      :type 'file-error))
      (should imoogi-project-notes--pending-link-operation)
      (should (equal original-task-text
                     (imoogi-project-notes-transaction-test--read tasks)))
      (should (equal original-doc-text
                     (imoogi-project-notes-transaction-test--read doc)))
      (should-not (buffer-modified-p (find-buffer-visiting tasks)))
      (should-not (buffer-modified-p (find-buffer-visiting doc)))
      (imoogi-project-notes-retry-link-operation)
      (should-not imoogi-project-notes--pending-link-operation)
      (should (string-match-p "Rollback Doc"
                              (imoogi-project-notes-transaction-test--read tasks)))
      (should (string-match-p "^\\* Link$"
                              (imoogi-project-notes-transaction-test--read doc))))))

(ert-deftest imoogi-project-notes-link-retry-rejects-concurrent-source-buffer-edit ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/concurrent-buffer.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Concurrent Buffer\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Concurrent buffer task\n")
      (save-buffer)
      (imoogi-project-notes-transaction-test--goto-heading
       "Concurrent buffer task")
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (&rest args)
                   (if fail-once
                       (progn
                         (setq fail-once nil)
                         (signal 'file-error '("prewrite boom")))
                     (apply original-classifier args)))))
        (should-error (imoogi-project-notes-link-artifact doc)
                      :type 'file-error))
      (with-current-buffer (find-buffer-visiting tasks)
        (goto-char (point-max))
        (insert "\nConcurrent user edit\n"))
      (should-error (imoogi-project-notes-retry-link-operation)
                    :type 'user-error))))

(ert-deftest imoogi-project-notes-link-retry-rejects-concurrent-document-disk-edit ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/concurrent-disk.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Concurrent Disk\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Concurrent disk task\n")
      (save-buffer)
      (imoogi-project-notes-transaction-test--goto-heading
       "Concurrent disk task")
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (&rest args)
                   (if fail-once
                       (progn
                         (setq fail-once nil)
                         (signal 'file-error '("prewrite boom")))
                     (apply original-classifier args)))))
        (should-error (imoogi-project-notes-link-artifact doc)
                      :type 'file-error))
      (with-temp-file doc
        (insert "#+TITLE: Concurrent Disk\n\nChanged elsewhere\n"))
      (should-error (imoogi-project-notes-retry-link-operation)
                    :type 'user-error))))

(ert-deftest imoogi-project-notes-link-partial-postwrite-disk-state-retries_without_overwriting_saved_peer ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/partial-post.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (warning-minimum-level :emergency)
           fail-task
           saved-doc-text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Partial Post\n\nBody\n"))
      (with-current-buffer (find-file-noselect doc)
        (add-hook 'after-save-hook
                  (lambda () (error "postwrite persisted"))
                  nil t))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Partial post task\n")
      (save-buffer)
      (imoogi-project-notes-transaction-test--goto-heading
       "Partial post task")
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (buffer before-hash)
                   (if (and fail-task
                            (string= (buffer-file-name buffer) tasks))
                       (progn
                         (setq fail-task nil)
                         (signal 'file-error '("task write blocked")))
                     (prog1 (funcall original-classifier buffer before-hash)
                       (when (string= (buffer-file-name buffer) doc)
                         (setq fail-task t)))))))
        (should-error (imoogi-project-notes-link-artifact doc)
                      :type 'file-error))
      (setq saved-doc-text
            (imoogi-project-notes-transaction-test--read doc))
      (should (string-match-p "^\\* Link$" saved-doc-text))
      (should imoogi-project-notes--pending-link-operation)
      (imoogi-project-notes-retry-link-operation)
      (should-not imoogi-project-notes--pending-link-operation)
      (should (equal saved-doc-text
                     (imoogi-project-notes-transaction-test--read doc)))
      (should (string-match-p "Partial Post"
                              (imoogi-project-notes-transaction-test--read tasks))))))

(ert-deftest imoogi-project-notes-link-retry-rejects_pending_filename_dirty_buffer_exemption ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/pending-dirty.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Pending Dirty\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Pending dirty task\n")
      (save-buffer)
      (imoogi-project-notes-transaction-test--goto-heading
       "Pending dirty task")
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (&rest args)
                   (if fail-once
                       (progn
                         (setq fail-once nil)
                         (signal 'file-error '("prewrite boom")))
                     (apply original-classifier args)))))
        (should-error (imoogi-project-notes-link-artifact doc)
                      :type 'file-error))
      (with-current-buffer (find-buffer-visiting doc)
        (goto-char (point-max))
        (insert "\nUnsaved same-file edit\n"))
      (should-error (imoogi-project-notes-retry-link-operation)
                    :type 'user-error))))

(ert-deftest imoogi-project-notes-link-rejects-readonly-peer-without_changes ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/readonly-peer.org" directory))
           (original-doc-text "#+TITLE: Readonly Peer\n\nBody\n")
           original-task-text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert original-doc-text))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Readonly peer task\n")
      (save-buffer)
      (setq original-task-text
            (imoogi-project-notes-transaction-test--read tasks))
      (with-current-buffer (find-file-noselect doc)
        (setq buffer-read-only t))
      (imoogi-project-notes-transaction-test--goto-heading
       "Readonly peer task")
      (should-error (imoogi-project-notes-link-artifact doc)
                    :type 'user-error)
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (org-entry-get (point) "ID"))
      (should (equal original-task-text
                     (imoogi-project-notes-transaction-test--read tasks)))
      (should (equal original-doc-text
                     (imoogi-project-notes-transaction-test--read doc))))))

(ert-deftest imoogi-project-notes-unlink-retry-saves_recorded_postimage ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/unlink-retry.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency)
           doc-id)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Unlink Retry\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Unlink retry task\n")
      (save-buffer)
      (imoogi-project-notes-transaction-test--goto-heading
       "Unlink retry task")
      (imoogi-project-notes-link-artifact doc)
      (setq doc-id
            (let ((doc-text (imoogi-project-notes-transaction-test--read doc)))
              (string-match ":ID:[[:space:]]+\\([^[:space:]\n]+\\)"
                            doc-text)
              (match-string 1 doc-text)))
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (&rest args)
                   (if fail-once
                       (progn
                         (setq fail-once nil)
                         (signal 'file-error '("unlink boom")))
                     (apply original-classifier args)))))
        (should-error
         (imoogi-project-notes-unlink-artifact
          `((id . ,doc-id) (file . ,doc) (title . "Unlink Retry")))
         :type 'file-error))
      (imoogi-project-notes-retry-link-operation)
      (should-not (string-match-p
                   (regexp-quote
                    (format "[[id:%s][Unlink Retry]]" doc-id))
                   (imoogi-project-notes-transaction-test--read tasks)))
      (should-not (string-match-p "Unlink retry task"
                                  (imoogi-project-notes-transaction-test--read doc))))))

(ert-deftest imoogi-project-notes-unlink-missing-peer-does-not_create_task_id ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (warning-minimum-level :emergency))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Missing peer no id\n산출물:\n- [[id:MISSING-DOC][Missing Doc]]\n")
      (save-buffer)
      (imoogi-project-notes-transaction-test--goto-heading
       "Missing peer no id")
      (imoogi-project-notes-unlink-artifact
       '((id . "MISSING-DOC") (file . nil) (title . "Missing Doc")))
      (should-not (org-entry-get (point) "ID"))
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (string-match-p
                   "\\[\\[id:MISSING-DOC\\]\\[Missing Doc\\]\\]"
                   (imoogi-project-notes-transaction-test--read tasks))))))

(ert-deftest imoogi-project-notes-unlink-missing-peer-save-failure-rolls_back_and_retries_source_only ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (original-task-text)
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Missing peer retry\n산출물:\n- [[id:MISSING-RETRY][Missing Retry]]\n")
      (save-buffer)
      (setq original-task-text
            (imoogi-project-notes-transaction-test--read tasks))
      (imoogi-project-notes-transaction-test--goto-heading
       "Missing peer retry")
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (&rest args)
                   (if fail-once
                       (progn
                         (setq fail-once nil)
                         (signal 'file-error '("missing peer task blocked")))
                     (apply original-classifier args)))))
        (should-error
         (imoogi-project-notes-unlink-artifact
          '((id . "MISSING-RETRY") (file . nil) (title . "Missing Retry")))
         :type 'file-error))
      (should imoogi-project-notes--pending-link-operation)
      (should (equal original-task-text
                     (imoogi-project-notes-transaction-test--read tasks)))
      (should-not (buffer-modified-p (find-buffer-visiting tasks)))
      (should-not (org-entry-get (point) "ID"))
      (imoogi-project-notes-retry-link-operation)
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (org-entry-get (point) "ID"))
      (should-not (string-match-p
                   "\\[\\[id:MISSING-RETRY\\]\\[Missing Retry\\]\\]"
                   (imoogi-project-notes-transaction-test--read tasks))))))

(ert-deftest imoogi-project-notes-unlink-missing-peer-postwrite-hook-error-is_classified_source_only ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (warning-minimum-level :emergency))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Missing peer hook\n산출물:\n- [[id:MISSING-HOOK][Missing Hook]]\n")
      (save-buffer)
      (add-hook 'after-save-hook
                (lambda () (error "missing peer postwrite"))
                nil t)
      (imoogi-project-notes-transaction-test--goto-heading
       "Missing peer hook")
      (imoogi-project-notes-unlink-artifact
       '((id . "MISSING-HOOK") (file . nil) (title . "Missing Hook")))
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (org-entry-get (point) "ID"))
      (should-not (string-match-p
                   "\\[\\[id:MISSING-HOOK\\]\\[Missing Hook\\]\\]"
                   (imoogi-project-notes-transaction-test--read tasks))))))

(ert-deftest imoogi-project-notes-unlink-duplicate-link-heading-preparation-failure-keeps_links_unchanged ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/unlink-duplicate.org" directory))
           task-id
           original-task-text
           original-doc-text)
      (make-directory (file-name-directory doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Unlink duplicate\n산출물:\n- [[id:UNLINK-DUP][Unlink Duplicate]]\n")
      (imoogi-project-notes-transaction-test--goto-heading
       "Unlink duplicate")
      (setq task-id (org-id-get-create))
      (save-buffer)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID: UNLINK-DUP\n:END:\n\n#+TITLE: Unlink Duplicate\n\n* Link\n- [[id:"
                task-id "][Unlink duplicate]]\n\n* Link\n"))
      (setq original-task-text
            (imoogi-project-notes-transaction-test--read tasks)
            original-doc-text
            (imoogi-project-notes-transaction-test--read doc))
      (should-error
       (imoogi-project-notes-unlink-artifact
        `((id . "UNLINK-DUP") (file . ,doc) (title . "Unlink Duplicate")))
       :type 'user-error)
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (buffer-modified-p (find-buffer-visiting tasks)))
      (should-not (buffer-modified-p (find-buffer-visiting doc)))
      (should (equal original-task-text
                     (imoogi-project-notes-transaction-test--read tasks)))
      (should (equal original-doc-text
                     (imoogi-project-notes-transaction-test--read doc))))))

(ert-deftest imoogi-project-notes-retarget-retry-saves_recorded_postimage ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (old-doc (expand-file-name "references/retarget-old.org" directory))
           (new-doc (expand-file-name "references/retarget-new.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency)
           task-id)
      (make-directory (file-name-directory old-doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Retarget retry task\n산출물:\n- [[id:RETARGET-OLD][Retarget Old]]\n")
      (imoogi-project-notes-transaction-test--goto-heading
       "Retarget retry task")
      (setq task-id (org-id-get-create))
      (save-buffer)
      (with-temp-file old-doc
        (insert ":PROPERTIES:\n:ID: RETARGET-OLD\n:END:\n\n#+TITLE: Retarget Old\n\n* Link\n- [[id:"
                task-id "][Retarget retry task]]\n"))
      (with-temp-file new-doc
        (insert "#+TITLE: Retarget New\n\nBody\n"))
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (&rest args)
                   (if fail-once
                       (progn
                         (setq fail-once nil)
                         (signal 'file-error '("retarget boom")))
                     (apply original-classifier args)))))
        (should-error (imoogi-project-notes-repair-link
                       "RETARGET-OLD" new-doc)
                      :type 'file-error))
      (imoogi-project-notes-retry-link-operation)
      (should-not (string-match-p "RETARGET-OLD"
                                  (imoogi-project-notes-transaction-test--read tasks)))
      (should (string-match-p "Retarget New"
                              (imoogi-project-notes-transaction-test--read tasks)))
      (should-not (string-match-p "Retarget retry task"
                                  (imoogi-project-notes-transaction-test--read old-doc)))
      (should (string-match-p "Retarget retry task"
                              (imoogi-project-notes-transaction-test--read new-doc))))))

(ert-deftest imoogi-project-notes-retarget-new-doc-preparation-failure-keeps_old_link_unchanged ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (old-doc (expand-file-name "references/retarget-keep-old.org" directory))
           (new-doc (expand-file-name "references/retarget-duplicate.org" directory))
           task-id
           original-task-text
           original-old-text
           original-new-text)
      (make-directory (file-name-directory old-doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Retarget keep old\n산출물:\n- [[id:RETARGET-KEEP-OLD][Retarget Keep Old]]\n")
      (imoogi-project-notes-transaction-test--goto-heading
       "Retarget keep old")
      (setq task-id (org-id-get-create))
      (save-buffer)
      (with-temp-file old-doc
        (insert ":PROPERTIES:\n:ID: RETARGET-KEEP-OLD\n:END:\n\n#+TITLE: Retarget Keep Old\n\n* Link\n- [[id:"
                task-id "][Retarget keep old]]\n"))
      (with-temp-file new-doc
        (insert "#+TITLE: Retarget Duplicate\n\n* Link\n\n* Link\n"))
      (setq original-task-text
            (imoogi-project-notes-transaction-test--read tasks)
            original-old-text
            (imoogi-project-notes-transaction-test--read old-doc)
            original-new-text
            (imoogi-project-notes-transaction-test--read new-doc))
      (should-error (imoogi-project-notes-repair-link
                     "RETARGET-KEEP-OLD" new-doc)
                    :type 'user-error)
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (buffer-modified-p (find-buffer-visiting tasks)))
      (should-not (buffer-modified-p (find-buffer-visiting old-doc)))
      (should-not (buffer-modified-p (find-buffer-visiting new-doc)))
      (should (equal original-task-text
                     (imoogi-project-notes-transaction-test--read tasks)))
      (should (equal original-old-text
                     (imoogi-project-notes-transaction-test--read old-doc)))
      (should (equal original-new-text
                     (imoogi-project-notes-transaction-test--read new-doc))))))

(ert-deftest imoogi-project-notes-retarget-public-late-new-backlink-error-preserves_old_relation ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (old-doc (expand-file-name "references/retarget-late-old.org" directory))
           (new-doc (expand-file-name "references/retarget-late-new.org" directory))
           task-id
           original-task-text
           original-old-text
           original-new-text)
      (make-directory (file-name-directory old-doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Retarget late rollback\n산출물:\n- [[id:RETARGET-LATE-OLD][Retarget Late Old]]\n")
      (imoogi-project-notes-transaction-test--goto-heading
       "Retarget late rollback")
      (setq task-id (org-id-get-create))
      (save-buffer)
      (with-temp-file old-doc
        (insert ":PROPERTIES:\n:ID: RETARGET-LATE-OLD\n:END:\n\n#+TITLE: Retarget Late Old\n\n* Link\n- [[id:"
                task-id "][Retarget late rollback]]\n"))
      (with-temp-file new-doc
        (insert "#+TITLE: Retarget Late New\n\nBody\n"))
      (setq original-task-text
            (imoogi-project-notes-transaction-test--read tasks)
            original-old-text
            (imoogi-project-notes-transaction-test--read old-doc)
            original-new-text
            (imoogi-project-notes-transaction-test--read new-doc))
      (cl-letf (((symbol-function 'imoogi-project-notes--ensure-document-task-link)
                 (lambda (&rest _args)
                   (user-error "injected late retarget backlink failure"))))
        (should-error (imoogi-project-notes-repair-link
                       "RETARGET-LATE-OLD" new-doc)
                      :type 'user-error))
      (should-not imoogi-project-notes--pending-link-operation)
      (should-not (buffer-modified-p (find-buffer-visiting tasks)))
      (should-not (buffer-modified-p (find-buffer-visiting old-doc)))
      (should-not (buffer-modified-p (find-buffer-visiting new-doc)))
      (should (equal original-task-text
                     (imoogi-project-notes-transaction-test--read tasks)))
      (should (equal original-old-text
                     (imoogi-project-notes-transaction-test--read old-doc)))
      (should (equal original-new-text
                     (imoogi-project-notes-transaction-test--read new-doc))))))

(ert-deftest imoogi-project-notes-link-retry-rejects_nested_project_document_endpoint ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--all-entries)))
           (tasks (expand-file-name "tasks.org" directory))
           (nested-dir (expand-file-name "nested-project/" directory))
           (nested-doc (expand-file-name "references/nested-owned.org" nested-dir))
           (nested-source (expand-file-name "nested-source/" root))
           (text "#+TITLE: Nested Owned\n\nBody\n")
           hash
           registry)
      (make-directory (file-name-directory nested-doc) t)
      (with-temp-file nested-doc
        (insert text))
      (with-temp-file (expand-file-name "tasks.org" nested-dir)
        (insert "#+TITLE: Nested Tasks\n"))
      (with-temp-file (expand-file-name "overview.org" nested-dir)
        (insert "#+TITLE: Nested Overview\n"))
      (with-temp-file (expand-file-name "journal.org" nested-dir)
        (insert "#+TITLE: Nested Journal\n"))
      (make-directory nested-source t)
      (setq registry (imoogi-project-notes--read-registry))
      (push `((key . ,(concat "dir:" nested-source))
              (type . "project")
              (name . "nested-source")
              (source-root . ,(directory-file-name nested-source))
              (notes-dir . ,(directory-file-name nested-dir))
              (project-file . ,(expand-file-name "overview.org" nested-dir))
              (tasks-file . ,(expand-file-name "tasks.org" nested-dir))
              (journal-file . ,(expand-file-name "journal.org" nested-dir)))
            registry)
      (imoogi-project-notes--write-registry registry)
      (setq hash (secure-hash 'sha1 text))
      (setq imoogi-project-notes--pending-link-operation
            `((operation . link)
              (entry-note-id . ,(imoogi-project-notes--entry-derived-note-id
                                 entry))
              (entry-instance-id . ,(alist-get 'instance-id entry))
              (task-file . ,tasks)
              (task-id . "TASK-ID")
              (task-title . "Task")
              (doc-file . ,nested-doc)
              (doc-id . "NESTED-DOC")
              (doc-title . "Nested Owned")
              (endpoints . (((role . doc)
                             (file . ,nested-doc)
                             (pre-file-hash . ,hash)
                             (pre-buffer-hash . ,hash)
                             (pre-text . ,text)
                             (post-buffer-hash . ,hash)
                             (post-text . ,text))))))
      (should-error (imoogi-project-notes-retry-link-operation)
                    :type 'user-error)
      (should imoogi-project-notes--pending-link-operation)
      (should (equal text
                     (imoogi-project-notes-transaction-test--read
                      nested-doc))))))

(ert-deftest imoogi-project-notes-link-retry-rejects_ambiguous_task_endpoint_without_heading_owner ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--all-entries)))
           (tasks (expand-file-name "tasks.org" directory))
           task-id
           task-text
           hash
           registry
           duplicate-source)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Ambiguous retry task\n")
      (imoogi-project-notes-transaction-test--goto-heading
       "Ambiguous retry task")
      (setq task-id (org-id-get-create))
      (save-buffer)
      (setq task-text (imoogi-project-notes-transaction-test--read tasks)
            hash (secure-hash 'sha1 task-text)
            registry (imoogi-project-notes--read-registry)
            duplicate-source (expand-file-name "duplicate-source/" root))
      (make-directory duplicate-source t)
      (push `((key . ,(concat "dir:" duplicate-source))
              (type . "project")
              (name . "duplicate-source")
              (source-root . ,(directory-file-name duplicate-source))
              (notes-dir . ,(imoogi-project-notes--alist-string
                             'notes-dir entry))
              (project-file . ,(imoogi-project-notes--alist-string
                                'project-file entry))
              (tasks-file . ,tasks)
              (journal-file . ,(imoogi-project-notes--alist-string
                                'journal-file entry)))
            registry)
      (imoogi-project-notes--write-registry registry)
      (setq imoogi-project-notes--pending-link-operation
            `((operation . unlink)
              (entry-note-id . ,(imoogi-project-notes--entry-derived-note-id
                                 entry))
              (entry-instance-id . ,(alist-get 'instance-id entry))
              (task-file . ,tasks)
              (task-id . ,task-id)
              (task-title . "Ambiguous retry task")
              (doc-id . "DOC")
              (doc-title . "Doc")
              (endpoints . (((role . task)
                             (file . ,tasks)
                             (pre-file-hash . ,hash)
                             (pre-buffer-hash . ,hash)
                             (pre-text . ,task-text)
                             (post-buffer-hash . ,hash)
                             (post-text . ,task-text))))))
      (should-error (imoogi-project-notes-retry-link-operation)
                    :type 'user-error)
      (should imoogi-project-notes--pending-link-operation)
      (should (equal task-text
                     (imoogi-project-notes-transaction-test--read
                      tasks))))))

(ert-deftest imoogi-project-notes-link-retry-rejects-inactive-mounted-transition ()
  (imoogi-project-notes-transaction-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/retry-inactive.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency)
           entries entry)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Retry Inactive\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Retry inactive task\n")
      (save-buffer)
      (imoogi-project-notes-transaction-test--goto-heading
       "Retry inactive task")
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (&rest args)
                   (if fail-once
                       (progn
                         (setq fail-once nil)
                         (signal 'file-error '("prewrite boom")))
                     (apply original-classifier args)))))
        (should-error (imoogi-project-notes-link-artifact doc)
                      :type 'file-error))
      (setq entries (imoogi-project-notes--all-entries)
            entry (copy-tree (car entries)))
      (setf (alist-get 'origin entry) 'mounted)
      (setf (alist-get 'instance-id entry)
            (alist-get 'entry-instance-id
                       imoogi-project-notes--pending-link-operation))
      (setf (alist-get 'active-p entry) nil)
      (setf (alist-get 'inactive-reason entry) "test inactive")
      (cl-letf (((symbol-function 'imoogi-project-notes--all-entries)
                 (lambda () (list entry))))
        (should-error (imoogi-project-notes-retry-link-operation)
                      :type 'user-error))
      (should imoogi-project-notes--pending-link-operation))))

;;; project-notes-transaction-test.el ends here
