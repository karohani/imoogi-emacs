;;; project-notes-copy-test.el --- Project notes copy/refile tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'org)
(require 'org-refile)

(defmacro imoogi-project-notes-copy-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename (make-temp-file "imoogi-project-notes-copy-" t)))
          (user-emacs-directory (expand-file-name "emacs/" sandbox))
          (imoogi-project-notes-directory (expand-file-name "project-notes/" sandbox))
          (imoogi-project-notes-todo-storage 'project)
          (personal (expand-file-name "notes/" sandbox))
          (root-a (file-name-as-directory (expand-file-name "source-a/" sandbox)))
          (root-b (file-name-as-directory (expand-file-name "source-b/" sandbox)))
          (registry-file (expand-file-name ".cache/project-notes.json"
                                           user-emacs-directory))
          (mounted-roots-file
           (expand-file-name ".cache/project-notes-mounted-roots.json"
                             user-emacs-directory))
          (org-id-locations-file (expand-file-name "org-id-locations" sandbox))
          (org-agenda-files nil)
          (org-directory personal))
     (make-directory root-a t)
     (make-directory root-b t)
     (unwind-protect
         (cl-letf (((symbol-function 'imoogi-org--default-directory)
                    (lambda () personal))
                   ((symbol-function 'imoogi-project-notes--registry-file)
                    (lambda () registry-file))
                   ((symbol-function 'imoogi-project-notes--mounted-roots-file)
                    (lambda () mounted-roots-file)))
           (save-window-excursion
             (with-temp-buffer
               (setq default-directory root-a)
               ,@body)))
       (dolist (buffer (buffer-list))
         (when (and (buffer-file-name buffer)
                    (file-in-directory-p (buffer-file-name buffer) sandbox))
           (with-current-buffer buffer
             (set-buffer-modified-p nil))
           (kill-buffer buffer)))
       (delete-directory sandbox t))))

(defun imoogi-project-notes-copy-test--read (file)
  "Return FILE content."
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(defun imoogi-project-notes-copy-test--goto-heading (title)
  "Move point to heading TITLE."
  (goto-char (point-min))
  (re-search-forward (concat "^\\* TODO " (regexp-quote title) "$"))
  (beginning-of-line))

(ert-deftest imoogi-project-notes-link-direct-body-does-not-use-child-artifacts ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks (expand-file-name "tasks.org" dir-a))
           (doc (expand-file-name "references/parent-doc.org" dir-a)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Parent Doc\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Parent\n** TODO Child\n산출물:\n- [[id:CHILD-DOC][Child Doc]]\n")
      (save-buffer)
      (goto-char (point-min))
      (re-search-forward "^\\* TODO Parent$")
      (beginning-of-line)
      (imoogi-project-notes-link-artifact doc)
      (let ((task-text (imoogi-project-notes-copy-test--read tasks)))
        (should (string-match-p "^\\* TODO Parent\\(?:.\\|\n\\)*?산출물:\n- \\[\\[id:[^]]+\\]\\[Parent Doc\\]\\]\n\\*\\* TODO Child"
                                task-text))
        (should (string-match-p "\\[\\[id:CHILD-DOC\\]\\[Child Doc\\]\\]"
                                task-text))))))

(ert-deftest imoogi-project-notes-link-candidates-only-managed-direct-list ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks (expand-file-name "tasks.org" dir-a)))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Parent\nSee [[id:PROSE][Prose]].\n** TODO Child\n산출물:\n- [[id:CHILD][Child]]\n")
      (goto-char (point-min))
      (re-search-forward "^\\* TODO Parent$")
      (beginning-of-line)
      (should-not (member "PROSE" (imoogi-project-notes--task-linked-doc-ids)))
      (should-not (member "CHILD" (imoogi-project-notes--task-linked-doc-ids))))))

(ert-deftest imoogi-project-notes-link-duplicate-document-id-does-not-mutate-task ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (entry (imoogi-project-notes--find-entry-by-notes-directory dir-a))
           (tasks (expand-file-name "tasks.org" dir-a))
           (doc-a (expand-file-name "references/dup-a.org" dir-a))
           (doc-b (expand-file-name "references/dup-b.org" dir-a))
           before-text)
      (make-directory (file-name-directory doc-a) t)
      (with-temp-file doc-a
        (insert ":PROPERTIES:\n:ID: DUP-DOC\n:END:\n\n#+TITLE: Dup A\n"))
      (with-temp-file doc-b
        (insert ":PROPERTIES:\n:ID: DUP-DOC\n:END:\n\n#+TITLE: Dup B\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Duplicate failure\n")
      (save-buffer)
      (setq before-text (buffer-string))
      (imoogi-project-notes-copy-test--goto-heading "Duplicate failure")
      (should-error (imoogi-project-notes-link-artifact doc-a entry)
                    :type 'user-error)
      (should (equal before-text (buffer-string)))
      (should-not (buffer-modified-p)))))

(ert-deftest imoogi-project-notes-link-dirty-target-document-aborts ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks (expand-file-name "tasks.org" dir-a))
           (doc (expand-file-name "references/dirty.org" dir-a))
           before-doc)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Dirty\n\nOriginal\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Dirty target\n")
      (save-buffer)
      (with-current-buffer (find-file-noselect doc)
        (goto-char (point-max))
        (insert "Unsaved edit\n")
        (setq before-doc (buffer-string)))
      (imoogi-project-notes-copy-test--goto-heading "Dirty target")
      (should-error (imoogi-project-notes-link-artifact doc)
                    :type 'user-error)
      (with-current-buffer (find-file-noselect doc)
        (should (equal before-doc (buffer-string)))
        (should (buffer-modified-p))))))

(ert-deftest imoogi-project-notes-copy-file-freshens-internal-ids-and-strips-managed ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (doc (expand-file-name "development/design.org" dir-a))
           original copied copied-text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID: DOC-OLD\n:END:\n\n#+TITLE: Design\n\n"
                "Ordinary external [[id:EXTERNAL][External]] stays.\n"
                "* Section\n:PROPERTIES:\n:ID: INNER-OLD\n:END:\n"
                "Internal [[id:INNER-OLD][self]] rewrites.\n"
                "* Link\n[[id:TASK-OLD][Old task]]\n"
                "** 관련 작업\n[[id:TASK-LEGACY][Old legacy task]]\n"))
      (setq original (imoogi-project-notes-copy-test--read doc))
      (find-file doc)
      (goto-char (point-min))
      (setq copied (imoogi-project-notes-copy-to-project entry-b 'file)
            copied-text (imoogi-project-notes-copy-test--read
                         (alist-get 'file copied)))
      (should (equal original (imoogi-project-notes-copy-test--read doc)))
      (should-not (string-match-p ":ID:[[:space:]]+DOC-OLD" copied-text))
      (should-not (string-match-p ":ID:[[:space:]]+INNER-OLD" copied-text))
      (should-not (string-match-p "\\[\\[id:INNER-OLD\\]" copied-text))
      (should (string-match-p "\\[\\[id:EXTERNAL\\]\\[External\\]\\] stays"
                              copied-text))
      (should-not (string-match-p "\\[\\[id:TASK-OLD\\]\\[Old task\\]\\]"
                                  copied-text))
      (should-not (string-match-p "\\[\\[id:TASK-LEGACY\\]\\[Old legacy task\\]\\]"
                                  copied-text))
      (should (file-in-directory-p (alist-get 'file copied)
                                   (expand-file-name "artifacts/" dir-b))))))

(ert-deftest imoogi-project-notes-copy-subtree-strips-managed-and-preserves-sibling ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (tasks (expand-file-name "tasks.org" dir-a))
           original copied-text)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Copy me\n:PROPERTIES:\n:ID: TASK-OLD\n:IMOOGI_PROJECT_ID: OLD-PROJECT\n:END:\n산출물:\n- [[id:DOC-OLD][Doc]]\nBody [[id:EXTERNAL][External]]\n* TODO Sibling\nKeep.\n")
      (save-buffer)
      (setq original (imoogi-project-notes-copy-test--read tasks))
      (imoogi-project-notes-copy-test--goto-heading "Copy me")
      (setq copied-text (imoogi-project-notes-copy-test--read
                         (alist-get 'file
                                    (imoogi-project-notes-copy-to-project
                                     entry-b 'subtree))))
      (should (equal original (imoogi-project-notes-copy-test--read tasks)))
      (should (string-match-p "^\\* TODO Copy me" copied-text))
      (should-not (string-match-p "Sibling" copied-text))
      (should-not (string-match-p "DOC-OLD" copied-text))
      (should-not (string-match-p "IMOOGI_PROJECT_ID" copied-text))
      (should (string-match-p "\\[\\[id:EXTERNAL\\]\\[External\\]\\]"
                              copied-text)))))

(ert-deftest imoogi-project-notes-refile-cross-project-cancel-leaves-source-and-dest ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (source-before nil)
           (dest-before (imoogi-project-notes-copy-test--read tasks-b))
           (rfloc (list "Tasks" tasks-b nil 1)))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Protected\n산출물:\n- [[id:DOC][Doc]]\n")
      (save-buffer)
      (setq source-before (imoogi-project-notes-copy-test--read tasks-a))
      (imoogi-project-notes-copy-test--goto-heading "Protected")
      (cl-letf (((symbol-function 'completing-read)
                 (lambda (&rest _) "cancel")))
        (should-error (org-refile nil nil rfloc)
                      :type 'user-error))
      (should (equal source-before
                     (imoogi-project-notes-copy-test--read tasks-a)))
      (should (equal dest-before
                     (imoogi-project-notes-copy-test--read tasks-b))))))

(ert-deftest imoogi-project-notes-refile-cross-project-keep-copies-fresh ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (source-before nil)
           (rfloc (list "Tasks" tasks-b nil 1)))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Protected copy\n:PROPERTIES:\n:ID: TASK-COPY-OLD\n:END:\n산출물:\n- [[id:DOC][Doc]]\n")
      (save-buffer)
      (setq source-before (imoogi-project-notes-copy-test--read tasks-a))
      (imoogi-project-notes-copy-test--goto-heading "Protected copy")
      (org-refile 3 nil rfloc)
      (should (equal source-before
                     (imoogi-project-notes-copy-test--read tasks-a)))
      (let* ((copies (directory-files-recursively
                      (expand-file-name "artifacts/" dir-b) "\\.org\\'"))
             (text (imoogi-project-notes-copy-test--read (car copies))))
        (should (= 1 (length copies)))
        (should (string-match-p "Protected copy" text))
        (should-not (string-match-p "TASK-COPY-OLD\\|DOC" text))))))

(ert-deftest imoogi-project-notes-refile-same-project-keep-copies-safely ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (source-before nil)
           target-pos)
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "
* TODO Same project destination
")
      (imoogi-project-notes-copy-test--goto-heading "Same project destination")
      (setq target-pos (point))
      (goto-char (point-max))
      (insert "
* TODO Same project protected
:PROPERTIES:
:ID: SAME-PROTECTED
:END:
산출물:
- [[id:DOC-SAME][Doc]]
")
      (save-buffer)
      (setq source-before (imoogi-project-notes-copy-test--read tasks-a))
      (imoogi-project-notes-copy-test--goto-heading "Same project protected")
      (let ((org-refile-keep t))
        (org-refile nil nil (list "Same project destination" tasks-a nil target-pos)))
      (should (equal source-before
                     (imoogi-project-notes-copy-test--read tasks-a)))
      (let* ((copies (directory-files-recursively
                      (expand-file-name "artifacts/" dir-a) "\\.org\\'"))
             (text (and copies (imoogi-project-notes-copy-test--read
                                (car copies)))))
        (should (= 1 (length copies)))
        (should (string-match-p "Same project protected" text))
        (should-not (string-match-p "SAME-PROTECTED\\|DOC-SAME" text))))))

(ert-deftest imoogi-project-notes-refile-managed-partial-region-aborts-without-changes ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (source-before nil)
           (dest-before (imoogi-project-notes-copy-test--read tasks-b))
           (rfloc (list "Tasks" tasks-b nil 1)))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "
* TODO Managed partial
산출물:
- [[id:DOC][Doc]]
Body line
")
      (save-buffer)
      (setq source-before (imoogi-project-notes-copy-test--read tasks-a))
      (re-search-backward "Body line")
      (set-mark (point))
      (forward-char 4)
      (activate-mark)
      (let ((org-refile-keep t))
        (should-error (org-refile nil nil rfloc)
                      :type 'user-error))
      (should (equal source-before
                     (imoogi-project-notes-copy-test--read tasks-a)))
      (should (equal dest-before
                     (imoogi-project-notes-copy-test--read tasks-b)))
      (should-not (directory-files-recursively
                   (expand-file-name "artifacts/" dir-b) "\.org\'")))))

(ert-deftest imoogi-project-notes-refile-unrelated-partial-region-passes-through ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (rfloc (list "Tasks" tasks-b nil 1)))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "
* TODO Plain partial
Body line
")
      (save-buffer)
      (imoogi-project-notes-copy-test--goto-heading "Plain partial")
      (set-mark (point))
      (forward-char 6)
      (activate-mark)
      (cl-letf (((symbol-function
                  'imoogi-project-notes--copy-current-source-to-entry)
                 (lambda (&rest _)
                   (error "project-note copy guard should not run"))))
        (org-refile nil nil rfloc))
      (should-not (directory-files-recursively
                   (expand-file-name "artifacts/" dir-b) "\\.org\\'")))))

(ert-deftest imoogi-project-notes-refile-target-heading-project-id-wins-over-category ()
  (let* ((entry-a '((name . "Project A") (note-id . "project-a")
                    (notes-dir . "/tmp/imoogi-project-a/")))
         (entry-b '((name . "Project B") (note-id . "project-b")
                    (notes-dir . "/tmp/imoogi-project-b/")))
         target-pos)
    (with-temp-buffer
      (org-mode)
      (insert "
* TODO Target
:PROPERTIES:
:IMOOGI_PROJECT_ID: project-b
:CATEGORY: Project A
:END:
")
      (goto-char (point-min))
      (re-search-forward "^\\* TODO Target$")
      (beginning-of-line)
      (setq target-pos (point))
      (should (eq entry-b
                  (imoogi-project-notes--entry-at-refile-target
                   (list "Target" nil nil target-pos)
                   (list entry-a entry-b)))))))

(ert-deftest imoogi-project-notes-refile-central-agenda-target-heading-owner-is-used ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((imoogi-project-notes-todo-storage 'central)
           (dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-a (imoogi-project-notes--find-entry-by-key
                     (imoogi-project-notes--identity-key root-a)
                     (imoogi-project-notes--all-entries)))
           (entry-b (imoogi-project-notes--find-entry-by-key
                     (imoogi-project-notes--identity-key root-b)
                     (imoogi-project-notes--all-entries)))
           (agenda (imoogi-project-notes--alist-string 'tasks-file entry-a))
           target-pos source-before)
      (find-file agenda)
      (goto-char (point-max))
      (insert "
* TODO P managed
:PROPERTIES:
:ID: CENTRAL-P
:CATEGORY: "
              (imoogi-project-notes--entry-name entry-a)
              "
:END:
산출물:
- [[id:DOC-P][Doc P]]
"
              "
* TODO Q destination
:PROPERTIES:
:CATEGORY: "
              (imoogi-project-notes--entry-name entry-b)
              "
:END:
")
      (save-buffer)
      (setq source-before (imoogi-project-notes-copy-test--read agenda))
      (imoogi-project-notes-copy-test--goto-heading "Q destination")
      (setq target-pos (point))
      (imoogi-project-notes-copy-test--goto-heading "P managed")
      (let ((org-refile-keep t))
        (org-refile nil nil (list "Q destination" agenda nil target-pos)))
      (should (equal source-before
                     (imoogi-project-notes-copy-test--read agenda)))
      (let* ((copies (directory-files-recursively
                      (expand-file-name "artifacts/" dir-b) "\\.org\\'"))
             (text (and copies (imoogi-project-notes-copy-test--read
                                (car copies)))))
        (should (= 1 (length copies)))
        (should (string-match-p "P managed" text))
        (should-not (string-match-p "CENTRAL-P\\|DOC-P" text))))))

(ert-deftest imoogi-project-notes-refile-central-agenda-unowned-target-does-not-guess-other-heading ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((imoogi-project-notes-todo-storage 'central)
           (dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-a (imoogi-project-notes--find-entry-by-key
                     (imoogi-project-notes--identity-key root-a)
                     (imoogi-project-notes--all-entries)))
           (entry-b (imoogi-project-notes--find-entry-by-key
                     (imoogi-project-notes--identity-key root-b)
                     (imoogi-project-notes--all-entries)))
           (agenda (imoogi-project-notes--alist-string 'tasks-file entry-a))
           target-pos before)
      (find-file agenda)
      (goto-char (point-max))
      (insert "
* TODO P managed
:PROPERTIES:
:ID: CENTRAL-P
:CATEGORY: "
              (imoogi-project-notes--entry-name entry-a)
              "
:END:
산출물:
- [[id:DOC-P][Doc P]]
"
              "
* TODO Other owned
:PROPERTIES:
:CATEGORY: "
              (imoogi-project-notes--entry-name entry-b)
              "
:END:
"
              "
* TODO Unowned target
")
      (save-buffer)
      (setq before (imoogi-project-notes-copy-test--read agenda))
      (imoogi-project-notes-copy-test--goto-heading "Unowned target")
      (setq target-pos (point))
      (imoogi-project-notes-copy-test--goto-heading "P managed")
      (let ((org-refile-keep t))
        (should-error
         (org-refile nil nil (list "Unowned target" agenda nil target-pos))
         :type 'user-error))
      (should (equal before (imoogi-project-notes-copy-test--read agenda)))
      (should-not (directory-files-recursively
                   (expand-file-name "artifacts/" dir-b) "\\.org\\'")))))

(ert-deftest imoogi-project-notes-refile-central-agenda-invalid-project-id-rejects ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((imoogi-project-notes-todo-storage 'central)
           (dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-a (imoogi-project-notes--find-entry-by-key
                     (imoogi-project-notes--identity-key root-a)
                     (imoogi-project-notes--all-entries)))
           (entry-b (imoogi-project-notes--find-entry-by-key
                     (imoogi-project-notes--identity-key root-b)
                     (imoogi-project-notes--all-entries)))
           (agenda (imoogi-project-notes--alist-string 'tasks-file entry-a))
           target-pos before)
      (find-file agenda)
      (goto-char (point-max))
      (insert "
* TODO P managed
:PROPERTIES:
:CATEGORY: "
              (imoogi-project-notes--entry-name entry-a)
              "
:END:
산출물:
- [[id:DOC-P][Doc P]]

* TODO Invalid target
:PROPERTIES:
:IMOOGI_PROJECT_ID: missing-project
:CATEGORY: "
              (imoogi-project-notes--entry-name entry-b)
              "
:END:
")
      (save-buffer)
      (setq before (imoogi-project-notes-copy-test--read agenda))
      (imoogi-project-notes-copy-test--goto-heading "Invalid target")
      (setq target-pos (point))
      (imoogi-project-notes-copy-test--goto-heading "P managed")
      (let ((org-refile-keep t))
        (should-error
         (org-refile nil nil (list "Invalid target" agenda nil target-pos))
         :type 'user-error))
      (should (equal before (imoogi-project-notes-copy-test--read agenda)))
      (should-not (directory-files-recursively
                   (expand-file-name "artifacts/" dir-b) "\\.org\\'")))))

(ert-deftest imoogi-project-notes-copy-existing-destination-fails-before-id-publish ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (doc (expand-file-name "references/source.org" dir-a))
           (collision (expand-file-name "artifacts/collision.org" dir-b))
           published)
      (make-directory (file-name-directory doc) t)
      (make-directory (file-name-directory collision) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID: SOURCE-ID\n:END:\n\n#+TITLE: Source\n"))
      (with-temp-file collision
        (insert "#+TITLE: Existing\n"))
      (find-file doc)
      (cl-letf (((symbol-function 'imoogi-project-notes--copy-destination-file)
                 (lambda (&rest _) collision))
                ((symbol-function 'imoogi-project-notes--publish-copy-ids)
                 (lambda (&rest _)
                   (setq published t))))
        (should-error (imoogi-project-notes-copy-to-project entry-b 'file)
                      :type 'user-error))
      (should-not published)
      (should (string-match-p "#\\+TITLE: Existing"
                              (imoogi-project-notes-copy-test--read
                               collision))))))

(ert-deftest imoogi-project-notes-org-roam-same-project-managed-file-root-is-blocked ()
  (imoogi-project-notes-copy-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (doc (expand-file-name "references/managed.org" dir-a))
           (orig-called nil))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Managed\n\n* Link\n- [[id:TASK][Task]]\n"))
      (find-file doc)
      (goto-char (point-min))
      (unless (fboundp 'org-roam-node-create)
        (require 'org-roam-node))
      (let ((node (org-roam-node-create :file doc
                                        :id "primary-node"
                                        :title "Managed")))
        (should-error
         (imoogi-project-notes--org-roam-refile-around
          (lambda (&rest _)
            (setq orig-called t))
          node)
         :type 'user-error))
      (should-not orig-called))))

;;; project-notes-copy-test.el ends here
