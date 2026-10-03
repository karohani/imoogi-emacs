;;; project-notes-links-test.el --- Project notes document link tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'org)

(defmacro imoogi-project-notes-links-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename (make-temp-file "imoogi-project-notes-links-" t)))
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

(defun imoogi-project-notes-links-test--read (file)
  "Return FILE content."
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(defun imoogi-project-notes-links-test--count (regexp text)
  "Return number of REGEXP matches in TEXT."
  (let ((start 0)
        (count 0))
    (while (string-match regexp text start)
      (setq count (1+ count)
            start (match-end 0)))
    count))

(defun imoogi-project-notes-links-test--goto-heading (title)
  "Move point to heading TITLE."
  (goto-char (point-min))
  (re-search-forward (concat "^\\* TODO " (regexp-quote title) "$"))
  (beginning-of-line))

(ert-deftest imoogi-project-notes-link-existing-document-many-to-many ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "development/shared-design.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Shared Design\n\n* Context\nKeep me.\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO 작업 B\nBody B\n\n* TODO 작업 C\nBody C\n")
      (save-buffer)
      (imoogi-project-notes-links-test--goto-heading "작업 B")
      (imoogi-project-notes-link-artifact doc)
      (imoogi-project-notes-link-artifact doc)
      (imoogi-project-notes-links-test--goto-heading "작업 C")
      (imoogi-project-notes-link-artifact doc)
      (let* ((task-text (imoogi-project-notes-links-test--read tasks))
             (doc-text (imoogi-project-notes-links-test--read doc))
             (doc-id (progn
                       (string-match ":ID:[[:space:]]+\\([^[:space:]\n]+\\)"
                                     doc-text)
                       (match-string 1 doc-text))))
        (should (= 2 (imoogi-project-notes-links-test--count
                      (regexp-quote (format "[[id:%s][Shared Design]]" doc-id))
                      task-text)))
        (should (string-match-p "^\\* Link$" doc-text))
        (should (= 1 (imoogi-project-notes-links-test--count
                      "\\[\\[id:[^]]+\\]\\[작업 B\\]\\]" doc-text)))
        (should (= 1 (imoogi-project-notes-links-test--count
                      "\\[\\[id:[^]]+\\]\\[작업 C\\]\\]" doc-text)))
        (with-current-buffer (find-file-noselect
                              (expand-file-name "project.org" directory))
          (goto-char (point-max))
          (imoogi-project-notes-insert-link doc)
          (save-buffer))
        (setq doc-text (imoogi-project-notes-links-test--read doc))
        (should (= 1 (imoogi-project-notes-links-test--count
                      "\\[\\[id:[^]]+\\]\\[작업 B\\]\\]" doc-text)))
        (find-file tasks)
        (imoogi-project-notes-links-test--goto-heading "작업 B")
        (imoogi-project-notes-unlink-artifact
         `((id . ,doc-id) (file . ,doc) (title . "Shared Design")))
        (setq task-text (imoogi-project-notes-links-test--read tasks)
              doc-text (imoogi-project-notes-links-test--read doc))
        (should (= 1 (imoogi-project-notes-links-test--count
                      (regexp-quote (format "[[id:%s][Shared Design]]" doc-id))
                      task-text)))
        (should-not (string-match-p "\\[\\[id:[^]]+\\]\\[작업 B\\]\\]" doc-text))
        (should (string-match-p "\\[\\[id:[^]]+\\]\\[작업 C\\]\\]" doc-text))
        (should (string-match-p "Keep me\\." doc-text))))))

(ert-deftest imoogi-project-notes-link-catalog-excludes-nested-project-root ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (nested-source (expand-file-name "nested-source/" sandbox))
           (nested-notes (expand-file-name "references/nested-project/"
                                           directory))
           (entry (car (imoogi-project-notes--read-registry)))
           (nested-entry (imoogi-project-notes--entry
                          "dir:nested" nested-source nested-notes 'project)))
      (make-directory nested-source t)
      (make-directory nested-notes t)
      (with-temp-file (expand-file-name "outer.org" directory)
        (insert "#+TITLE: Outer\n"))
      (with-temp-file (expand-file-name "inner.org" nested-notes)
        (insert "#+TITLE: Inner\n"))
      (imoogi-project-notes--write-registry (list entry nested-entry))
      (let ((files (mapcar (lambda (file)
                             (file-relative-name file directory))
                           (imoogi-project-notes--project-document-files
                            entry))))
        (should (member "outer.org" files))
        (should-not (member "references/nested-project/inner.org" files))))))

(ert-deftest imoogi-project-notes-insert-link-creates-id-without-backlink ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (source (expand-file-name "project.org" directory))
           (doc (expand-file-name "references/plain.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Plain Doc\n\nBody\n"))
      (find-file source)
      (goto-char (point-max))
      (insert "\nInserted: ")
      (imoogi-project-notes-insert-link doc)
      (save-buffer)
      (let ((source-text (imoogi-project-notes-links-test--read source))
            (doc-text (imoogi-project-notes-links-test--read doc)))
        (should (string-match-p "\\[\\[id:[^]]+\\]\\[Plain Doc\\]\\]"
                                source-text))
        (should (string-match-p ":ID:[[:space:]]+[^[:space:]\n]+"
                                doc-text))
        (should-not (string-match-p "^\\* Link$" doc-text))))))

(ert-deftest imoogi-project-notes-public-link-commands-select-document-when-noarg ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (imoogi-project-notes--find-entry-by-notes-directory
                   directory))
           (tasks (expand-file-name "tasks.org" directory))
           (source (expand-file-name "project.org" directory))
           (doc (expand-file-name "references/selected.org" directory))
           (selection-count 0))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Selected Doc\n\nBody\n"))
      (cl-letf (((symbol-function 'imoogi-project-notes--select-document)
                 (lambda (actual-entry prompt &optional require-id)
                   (should (eq actual-entry entry))
                   (should (string-match-p "문서" prompt))
                   (should require-id)
                   (setq selection-count (1+ selection-count))
                   `((file . ,doc)))))
        (find-file tasks)
        (goto-char (point-max))
        (insert "\n* TODO Select task\n")
        (save-buffer)
        (imoogi-project-notes-links-test--goto-heading "Select task")
        (imoogi-project-notes-link-artifact nil entry)
        (with-current-buffer (find-file-noselect source)
          (goto-char (point-max))
          (insert "\nSelected: ")
          (imoogi-project-notes-insert-link nil entry)
          (save-buffer)))
      (should (= selection-count 2))
      (should (string-match-p "Selected Doc"
                              (imoogi-project-notes-links-test--read tasks)))
      (should (string-match-p "\\[\\[id:[^]]+\\]\\[Selected Doc\\]\\]"
                              (imoogi-project-notes-links-test--read source))))))

(ert-deftest imoogi-project-notes-task-ordinary-id-link-is-not-managed-relation ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/cited.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID: CITED-DOC\n:END:\n\n#+TITLE: Cited Doc\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Citation task\nSee [[id:CITED-DOC][Cited Doc]] in prose.\n")
      (save-buffer)
      (imoogi-project-notes-links-test--goto-heading "Citation task")
      (imoogi-project-notes-link-artifact doc)
      (let ((task-text (imoogi-project-notes-links-test--read tasks)))
        (should (= 2 (imoogi-project-notes-links-test--count
                      "\\[\\[id:CITED-DOC\\]\\[Cited Doc\\]\\]"
                      task-text))))
      (imoogi-project-notes-unlink-artifact
       `((id . "CITED-DOC") (file . ,doc) (title . "Cited Doc")))
      (let ((task-text (imoogi-project-notes-links-test--read tasks))
            (doc-text (imoogi-project-notes-links-test--read doc)))
        (should (= 1 (imoogi-project-notes-links-test--count
                      "\\[\\[id:CITED-DOC\\]\\[Cited Doc\\]\\]"
                      task-text)))
        (should (string-match-p "See \\[\\[id:CITED-DOC\\]\\[Cited Doc\\]\\] in prose"
                                task-text))
        (should-not (string-match-p "\\[\\[id:[^]]+\\]\\[Citation task\\]\\]"
                                    doc-text))))))

(ert-deftest imoogi-project-notes-legacy-related-task-dedups-and-unlinks ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "artifacts/legacy.org" directory))
           task-id)
      (make-directory (file-name-directory doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO 기존 작업\n")
      (imoogi-project-notes-links-test--goto-heading "기존 작업")
      (setq task-id (org-id-get-create))
      (save-buffer)
      (with-temp-file doc
        (insert "#+TITLE: Legacy\n\n* Legacy\n:PROPERTIES:\n:ID: LEGACY-DOC\n:TYPE: 설계\n:END:\n\n** 관련 작업\n- [[id:"
                task-id "][기존 작업]]\n\n** Body\nKeep legacy.\n"))
      (imoogi-project-notes-link-artifact doc)
      (let ((doc-text (imoogi-project-notes-links-test--read doc)))
        (should-not (string-match-p "^\\* Link$" doc-text))
        (should (= 1 (imoogi-project-notes-links-test--count
                      (regexp-quote (format "[[id:%s][기존 작업]]" task-id))
                      doc-text))))
      (imoogi-project-notes-unlink-artifact
       `((id . "LEGACY-DOC") (file . ,doc) (title . "Legacy")))
      (let ((task-text (imoogi-project-notes-links-test--read tasks))
            (doc-text (imoogi-project-notes-links-test--read doc)))
        (should-not (string-match-p "\\[\\[id:LEGACY-DOC\\]\\[Legacy\\]\\]"
                                    task-text))
        (should-not (string-match-p (regexp-quote
                                     (format "[[id:%s][기존 작업]]" task-id))
                                    doc-text))
        (should (string-match-p "Keep legacy\\." doc-text))))))

(ert-deftest imoogi-project-notes-bare-link-heading-unlinks-historical-relation ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "artifacts/bare-link.org" directory))
           task-id)
      (make-directory (file-name-directory doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Bare link task\n산출물:\n- [[id:BARE-DOC][Bare Doc]]\n")
      (imoogi-project-notes-links-test--goto-heading "Bare link task")
      (setq task-id (org-id-get-create))
      (save-buffer)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID: BARE-DOC\n:END:\n\n#+TITLE: Bare Doc\n\n* Link\n[[id:"
                task-id "][Bare link task]]\n\nBody keeps [[id:OTHER][Other]].\n"))
      (imoogi-project-notes-unlink-artifact
       `((id . "BARE-DOC") (file . ,doc) (title . "Bare Doc")))
      (let ((doc-text (imoogi-project-notes-links-test--read doc)))
        (should-not (string-match-p
                     (regexp-quote (format "[[id:%s][Bare link task]]" task-id))
                     doc-text))
        (should (string-match-p "Body keeps \\[\\[id:OTHER\\]\\[Other\\]\\]"
                                doc-text))))))

(ert-deftest imoogi-project-notes-unlink-keeps-comment-and-source-block-id-links ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "artifacts/syntax-aware.org" directory)))
      (make-directory (file-name-directory doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Syntax-aware task\n:PROPERTIES:\n:ID:       SYNTAX-TASK\n:END:\n산출물:\n- [[id:SYNTAX-DOC][Syntax Doc]]\n")
      (save-buffer)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       SYNTAX-DOC\n:END:\n\n#+TITLE: Syntax Doc\n\n* Link\n# - [[id:SYNTAX-TASK][Syntax-aware task]]\n#+begin_src org\n- [[id:SYNTAX-TASK][Syntax-aware task]]\n#+end_src\n- [[id:SYNTAX-TASK][Syntax-aware task]]\n"))
      (imoogi-project-notes-links-test--goto-heading "Syntax-aware task")
      (imoogi-project-notes-unlink-artifact
       `((id . "SYNTAX-DOC") (file . ,doc) (title . "Syntax Doc")))
      (let ((doc-text (imoogi-project-notes-links-test--read doc)))
        (should (string-match-p
                 "# - \\[\\[id:SYNTAX-TASK\\]\\[Syntax-aware task\\]\\]"
                 doc-text))
        (should (string-match-p
                 "#\\+begin_src org\n- \\[\\[id:SYNTAX-TASK\\]\\[Syntax-aware task\\]\\]\n#\\+end_src"
                 doc-text))
        (should (= 2 (imoogi-project-notes-links-test--count
                      "\\[\\[id:SYNTAX-TASK\\]\\[Syntax-aware task\\]\\]"
                      doc-text)))))))

(ert-deftest imoogi-project-notes-link-postwrite-hook-error-is-classified ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/hook-doc.org" directory))
           (warning-minimum-level :emergency))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Hook Doc\n\nBody\n"))
      (with-current-buffer (find-file-noselect doc)
        (add-hook 'after-save-hook
                  (lambda () (error "postwrite boom"))
                  nil t))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Hook task\n")
      (save-buffer)
      (imoogi-project-notes-links-test--goto-heading "Hook task")
      (imoogi-project-notes-link-artifact doc)
      (should-not imoogi-project-notes--pending-link-operation)
      (should (string-match-p "^\\* Link$"
                              (imoogi-project-notes-links-test--read doc)))
      (should (string-match-p "Hook Doc"
                              (imoogi-project-notes-links-test--read tasks))))))

(ert-deftest imoogi-project-notes-link-save-failure-keeps-retry-operation ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/retry-doc.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (fail-once t)
           (warning-minimum-level :emergency))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Retry Doc\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Retry task\n")
      (save-buffer)
      (imoogi-project-notes-links-test--goto-heading "Retry task")
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
      (imoogi-project-notes-retry-link-operation)
      (should-not imoogi-project-notes--pending-link-operation)
      (should (string-match-p "Retry Doc"
                              (imoogi-project-notes-links-test--read tasks)))
      (should (string-match-p "^\\* Link$"
                              (imoogi-project-notes-links-test--read doc))))))

(ert-deftest imoogi-project-notes-link-retarget-failure-retry-preserves-replacement ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (old-doc (expand-file-name "references/old-doc.org" directory))
           (new-doc (expand-file-name "references/new-doc.org" directory))
           (original-classifier
            (symbol-function 'imoogi-project-notes--classify-save-buffer))
           (save-count 0)
           (warning-minimum-level :emergency)
           task-id new-id)
      (make-directory (file-name-directory old-doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Retarget task\n산출물:\n- [[id:OLD-DOC][Old Doc]]\n")
      (imoogi-project-notes-links-test--goto-heading "Retarget task")
      (setq task-id (org-id-get-create))
      (save-buffer)
      (with-temp-file old-doc
        (insert ":PROPERTIES:\n:ID: OLD-DOC\n:END:\n\n#+TITLE: Old Doc\n\n* Link\n- [[id:"
                task-id "][Retarget task]]\n"))
      (with-temp-file new-doc
        (insert "#+TITLE: New Doc\n\nBody\n"))
      (cl-letf (((symbol-function 'imoogi-project-notes--classify-save-buffer)
                 (lambda (&rest args)
                   (setq save-count (1+ save-count))
                   (if (= save-count 2)
                       (signal 'file-error '("between phases boom"))
                     (apply original-classifier args)))))
        (should-error (imoogi-project-notes-repair-link "OLD-DOC" new-doc)
                      :type 'file-error))
      (should imoogi-project-notes--pending-link-operation)
      (should (eq 'retarget
                  (alist-get 'operation
                             imoogi-project-notes--pending-link-operation)))
      (imoogi-project-notes-retry-link-operation)
      (should-not imoogi-project-notes--pending-link-operation)
      (let ((new-text (imoogi-project-notes-links-test--read new-doc)))
        (should (string-match ":ID:[[:space:]]+\\([^[:space:]\n]+\\)"
                              new-text))
        (setq new-id (match-string 1 new-text))
        (should (= 1 (imoogi-project-notes-links-test--count
                      (regexp-quote
                       (format "[[id:%s][Retarget task]]" task-id))
                      new-text))))
      (let ((task-text (imoogi-project-notes-links-test--read tasks))
            (old-text (imoogi-project-notes-links-test--read old-doc)))
        (should-not (string-match-p "\\[\\[id:OLD-DOC\\]\\[Old Doc\\]\\]"
                                    task-text))
        (should (= 1 (imoogi-project-notes-links-test--count
                      (regexp-quote (format "[[id:%s][New Doc]]" new-id))
                      task-text)))
        (should-not (string-match-p
                     (regexp-quote (format "[[id:%s][Retarget task]]" task-id))
                     old-text))))))

(ert-deftest imoogi-project-notes-link-retarget-postwrite-hook-error-is-classified ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (old-doc (expand-file-name "references/hook-old.org" directory))
           (new-doc (expand-file-name "references/hook-new.org" directory))
           (warning-minimum-level :emergency)
           task-id)
      (make-directory (file-name-directory old-doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Retarget hook task\n산출물:\n- [[id:HOOK-OLD][Hook Old]]\n")
      (imoogi-project-notes-links-test--goto-heading "Retarget hook task")
      (setq task-id (org-id-get-create))
      (save-buffer)
      (with-temp-file old-doc
        (insert ":PROPERTIES:\n:ID: HOOK-OLD\n:END:\n\n#+TITLE: Hook Old\n\n* Link\n- [[id:"
                task-id "][Retarget hook task]]\n"))
      (with-temp-file new-doc
        (insert "#+TITLE: Hook New\n\nBody\n"))
      (with-current-buffer (find-file-noselect new-doc)
        (add-hook 'after-save-hook
                  (lambda () (error "retarget postwrite boom"))
                  nil t))
      (imoogi-project-notes-repair-link "HOOK-OLD" new-doc)
      (should-not imoogi-project-notes--pending-link-operation)
      (let ((task-text (imoogi-project-notes-links-test--read tasks))
            (new-text (imoogi-project-notes-links-test--read new-doc)))
        (should-not (string-match-p "HOOK-OLD" task-text))
        (should (string-match-p "Hook New" task-text))
        (should (= 1 (imoogi-project-notes-links-test--count
                      (regexp-quote
                       (format "[[id:%s][Retarget hook task]]" task-id))
                      new-text)))))))

(ert-deftest imoogi-project-notes-link-preparation-failure-rolls-back-buffers ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/prep-fail.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Prep Fail Doc\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Prep fail task\n")
      (save-buffer)
      (let ((task-original (imoogi-project-notes-links-test--read tasks))
            (doc-original (imoogi-project-notes-links-test--read doc)))
        (imoogi-project-notes-links-test--goto-heading "Prep fail task")
        (cl-letf (((symbol-function 'imoogi-project-notes--ensure-task-artifact-link)
                   (lambda (&rest _)
                     (user-error "synthetic preparation failure"))))
          (should-error (imoogi-project-notes-link-artifact doc)
                        :type 'user-error))
        (should-not imoogi-project-notes--pending-link-operation)
        (should (string= task-original
                         (imoogi-project-notes-links-test--read tasks)))
        (should (string= doc-original
                         (imoogi-project-notes-links-test--read doc)))
        (should (string= task-original
                         (with-current-buffer (find-file-noselect tasks)
                           (buffer-string))))
        (should (string= doc-original
                         (with-current-buffer (find-file-noselect doc)
                           (buffer-string))))))))

(ert-deftest imoogi-project-notes-retarget-preparation-failure-rolls-back-buffers ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (old-doc (expand-file-name "references/rollback-old.org" directory))
           (new-doc (expand-file-name "references/rollback-new.org" directory))
           task-id)
      (make-directory (file-name-directory old-doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Retarget rollback task\n산출물:\n- [[id:ROLLBACK-OLD][Rollback Old]]\n")
      (imoogi-project-notes-links-test--goto-heading "Retarget rollback task")
      (setq task-id (org-id-get-create))
      (save-buffer)
      (with-temp-file old-doc
        (insert ":PROPERTIES:\n:ID: ROLLBACK-OLD\n:END:\n\n#+TITLE: Rollback Old\n\n* Link\n- [[id:"
                task-id "][Retarget rollback task]]\n"))
      (with-temp-file new-doc
        (insert "#+TITLE: Rollback New\n\nBody\n"))
      (let ((task-original (imoogi-project-notes-links-test--read tasks))
            (old-original (imoogi-project-notes-links-test--read old-doc))
            (new-original (imoogi-project-notes-links-test--read new-doc)))
        (cl-letf (((symbol-function 'imoogi-project-notes--ensure-task-artifact-link)
                   (lambda (&rest _)
                     (user-error "synthetic retarget preparation failure"))))
          (should-error (imoogi-project-notes-repair-link "ROLLBACK-OLD" new-doc)
                        :type 'user-error))
        (should-not imoogi-project-notes--pending-link-operation)
        (should (string= task-original
                         (imoogi-project-notes-links-test--read tasks)))
        (should (string= old-original
                         (imoogi-project-notes-links-test--read old-doc)))
        (should (string= new-original
                         (imoogi-project-notes-links-test--read new-doc)))
        (should (string= task-original
                         (with-current-buffer (find-file-noselect tasks)
                           (buffer-string))))
        (should (string= old-original
                         (with-current-buffer (find-file-noselect old-doc)
                           (buffer-string))))
        (should (string= new-original
                         (with-current-buffer (find-file-noselect new-doc)
                           (buffer-string))))))))

(ert-deftest imoogi-project-notes-unlink-rejects-existing-peer-without-document-id ()
  (imoogi-project-notes-links-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/no-id-peer.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: No ID Peer\n\n* Link\n- [[id:TASK-ID][No ID task]]\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO No ID task\n:PROPERTIES:\n:ID: TASK-ID\n:END:\n산출물:\n- [[id:NO-ID-PEER][No ID Peer]]\n")
      (save-buffer)
      (let ((task-original (imoogi-project-notes-links-test--read tasks))
            (doc-original (imoogi-project-notes-links-test--read doc)))
        (imoogi-project-notes-links-test--goto-heading "No ID task")
        (should-error
         (imoogi-project-notes-unlink-artifact
          `((id . "NO-ID-PEER") (file . ,doc) (title . "No ID Peer")))
         :type 'user-error)
        (should (string= task-original
                         (imoogi-project-notes-links-test--read tasks)))
        (should (string= doc-original
                         (imoogi-project-notes-links-test--read doc)))))))

(ert-deftest imoogi-project-notes-link-central-task-owner-property-is-recorded ()
  (imoogi-project-notes-links-test--isolated
    (setq imoogi-project-notes-todo-storage 'central)
    (let* ((first-directory (imoogi-project-notes-setup root))
           (other-root (file-name-as-directory
                        (expand-file-name "other-source/" sandbox)))
           (second-directory (progn
                               (make-directory other-root t)
                               (imoogi-project-notes-setup other-root)))
           (entry (imoogi-project-notes--find-entry-by-notes-directory
                   first-directory))
           (other-entry (imoogi-project-notes--find-entry-by-notes-directory
                         second-directory))
           (agenda (expand-file-name "agenda.org" personal))
           (doc (expand-file-name "references/central-doc.org" first-directory)))
      (should (equal (alist-get 'tasks-file entry)
                     (alist-get 'tasks-file other-entry)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Central Doc\n\nBody\n"))
      (find-file agenda)
      (goto-char (point-max))
      (insert "\n* TODO Shared central task\n")
      (save-buffer)
      (imoogi-project-notes-links-test--goto-heading "Shared central task")
      (imoogi-project-notes-link-artifact doc entry)
      (save-buffer)
      (should (string-match-p
               (concat ":IMOOGI_PROJECT_ID:[[:space:]]+"
                       (regexp-quote
                        (imoogi-project-notes--entry-derived-note-id entry)))
               (imoogi-project-notes-links-test--read agenda)))
      (should (file-directory-p second-directory)))))

(ert-deftest imoogi-project-notes-link-central-task-invalid-project-id-rejects ()
  (imoogi-project-notes-links-test--isolated
    (setq imoogi-project-notes-todo-storage 'central)
    (let* ((first-directory (imoogi-project-notes-setup root))
           (other-root (file-name-as-directory
                        (expand-file-name "other-source/" sandbox)))
           (_second-directory (progn
                                (make-directory other-root t)
                                (imoogi-project-notes-setup other-root)))
           (entry (imoogi-project-notes--find-entry-by-notes-directory
                   first-directory))
           (agenda (expand-file-name "agenda.org" personal))
           (doc (expand-file-name "references/central-doc.org" first-directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Central Doc\n\nBody\n"))
      (find-file agenda)
      (goto-char (point-max))
      (insert "\n* TODO Invalid central task\n:PROPERTIES:\n:IMOOGI_PROJECT_ID: missing-project\n:CATEGORY: "
              (imoogi-project-notes--entry-name entry)
              "\n:END:\n")
      (imoogi-project-notes-links-test--goto-heading "Invalid central task")
      (should-error (imoogi-project-notes-link-artifact doc)
                    :type 'user-error))))

;;; project-notes-links-test.el ends here
