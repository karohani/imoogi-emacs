;;; project-notes-domain-boundary-test.el --- Project notes domain boundary regressions -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'org)
(require 'org-refile)

(defmacro imoogi-project-notes-domain-boundary-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename (make-temp-file "imoogi-project-notes-domain-" t)))
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
     (make-directory personal t)
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

(defun imoogi-project-notes-domain-boundary-test--read (file)
  "Return FILE content."
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(defun imoogi-project-notes-domain-boundary-test--goto-heading (title)
  "Move point to TODO heading TITLE."
  (goto-char (point-min))
  (re-search-forward (concat "^\\* TODO " (regexp-quote title) "$"))
  (beginning-of-line))

(defun imoogi-project-notes-domain-boundary-test--artifact-files (directory)
  "Return copied Org artifact files below DIRECTORY."
  (let ((artifact-dir (expand-file-name "artifacts/" directory)))
    (and (file-directory-p artifact-dir)
         (directory-files-recursively artifact-dir "\\.org\\'"))))

(defun imoogi-project-notes-domain-boundary-test--ids (text)
  "Return Org ID property values in TEXT."
  (let ((start 0)
        ids)
    (while (string-match "^[[:space:]]*:ID:[[:space:]]+\\([^[:space:]\n]+\\)"
                         text start)
      (push (match-string 1 text) ids)
      (setq start (match-end 0)))
    (nreverse ids)))

(ert-deftest imoogi-project-notes-domain-copy-keeps-source-block-id-literals-while-freshening-owned-nodes ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (doc (expand-file-name "references/source-block.org" dir-a))
           original copied-text copied-ids)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       REAL-DOC\n:END:\n\n"
                "#+TITLE: Source Block\n\n"
                "#+begin_src org\n"
                ":PROPERTIES:\n"
                ":ID:       LITERAL-ID\n"
                ":END:\n"
                "[[id:LITERAL-ID][literal link]]\n"
                "#+end_src\n\n"
                "* Real Child\n:PROPERTIES:\n:ID:       REAL-CHILD\n:END:\n"
                "[[id:REAL-CHILD][real child link]]\n"))
      (setq original (imoogi-project-notes-domain-boundary-test--read doc))
      (find-file doc)
      (setq copied-text
            (imoogi-project-notes-domain-boundary-test--read
             (alist-get 'file (imoogi-project-notes-copy-to-project entry-b 'file)))
            copied-ids (imoogi-project-notes-domain-boundary-test--ids copied-text))
      (should (equal original
                     (imoogi-project-notes-domain-boundary-test--read doc)))
      (should (member "LITERAL-ID" copied-ids))
      (should (string-match-p "\\[\\[id:LITERAL-ID\\]\\[literal link\\]\\]"
                              copied-text))
      (should-not (member "REAL-DOC" copied-ids))
      (should-not (member "REAL-CHILD" copied-ids))
      (should-not (string-match-p "\\[\\[id:REAL-CHILD\\]\\[real child link\\]\\]"
                                  copied-text)))))

(ert-deftest imoogi-project-notes-domain-copy-strips-managed-id-link-but-keeps-trailing-prose-and-unrelated-id-link ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (doc (expand-file-name "references/trailing-prose.org" dir-a))
           copied-text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       TRAILING-DOC\n:END:\n\n"
                "#+TITLE: Trailing Prose\n\n"
                "* Link\n"
                "- [[id:TASK-OLD][Old task]] keep this [[id:UNRELATED][citation]] prose\n"))
      (find-file doc)
      (setq copied-text
            (imoogi-project-notes-domain-boundary-test--read
             (alist-get 'file (imoogi-project-notes-copy-to-project entry-b 'file))))
      (should-not (string-match-p "\\[\\[id:TASK-OLD\\]\\[Old task\\]\\]"
                                  copied-text))
      (should (string-match-p "keep this \\[\\[id:UNRELATED\\]\\[citation\\]\\] prose"
                              copied-text)))))

(ert-deftest imoogi-project-notes-domain-task-artifact-marker-must-be-direct-body ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks (expand-file-name "tasks.org" dir-a))
           (doc (expand-file-name "references/direct-only.org" dir-a))
           text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       DIRECT-DOC\n:END:\n\n#+TITLE: Direct Only\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Direct marker only\n"
              "#+begin_quote\n"
              "산출물:\n"
              "- [[id:FAKE-QUOTE][Quote fake]]\n"
              "#+end_quote\n"
              ":LOGBOOK:\n"
              "산출물:\n"
              "- [[id:FAKE-DRAWER][Drawer fake]]\n"
              ":END:\n")
      (save-buffer)
      (imoogi-project-notes-domain-boundary-test--goto-heading
       "Direct marker only")
      (imoogi-project-notes-link-artifact doc)
      (setq text (imoogi-project-notes-domain-boundary-test--read tasks))
      (should (string-match-p (regexp-quote "#+begin_quote\n산출물:\n- [[id:FAKE-QUOTE][Quote fake]]\n#+end_quote")
                              text))
      (should (string-match-p (regexp-quote ":LOGBOOK:\n산출물:\n- [[id:FAKE-DRAWER][Drawer fake]]\n:END:")
                              text))
      (should (string-match-p (regexp-quote "\n산출물:\n- [[id:DIRECT-DOC][Direct Only]]")
                              text)))))

(ert-deftest imoogi-project-notes-domain-link-rejects-document-id-that-duplicates-owned-task-id-before-mutation ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks (expand-file-name "tasks.org" dir-a))
           (doc (expand-file-name "references/duplicate-task-id.org" dir-a))
           task-before doc-before)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       TASK-AND-DOC\n:END:\n\n#+TITLE: Duplicate Task ID\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Task owns duplicate\n:PROPERTIES:\n:ID:       TASK-AND-DOC\n:END:\n")
      (save-buffer)
      (setq task-before (imoogi-project-notes-domain-boundary-test--read tasks)
            doc-before (imoogi-project-notes-domain-boundary-test--read doc))
      (imoogi-project-notes-domain-boundary-test--goto-heading
       "Task owns duplicate")
      (should-error (imoogi-project-notes-link-artifact doc)
                    :type 'user-error)
      (should (equal task-before
                     (imoogi-project-notes-domain-boundary-test--read tasks)))
      (should (equal doc-before
                     (imoogi-project-notes-domain-boundary-test--read doc)))
      (should-not (buffer-modified-p (find-buffer-visiting tasks)))
      (should-not (buffer-modified-p (find-buffer-visiting doc))))))

(ert-deftest imoogi-project-notes-domain-split-legacy-root-rejects-unlink-then-repair-retargets ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks (expand-file-name "tasks.org" dir-a))
           (orphan (expand-file-name "artifacts/orphan-link.org" dir-a))
           (moved-root (expand-file-name "references/moved-root.org" dir-a))
           (replacement (expand-file-name "references/replacement.org" dir-a))
           task-before orphan-before moved-before task-id)
      (make-directory (file-name-directory orphan) t)
      (make-directory (file-name-directory moved-root) t)
      (make-directory (file-name-directory replacement) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Split legacy relation\n:PROPERTIES:\n:ID:       SPLIT-TASK\n:END:\n산출물:\n- [[id:LEGACY-MOVED][Moved Legacy]]\n")
      (save-buffer)
      (setq task-id "SPLIT-TASK")
      (with-temp-file orphan
        (insert "#+TITLE: Old Artifact Shell\n\n"
                "* Link\n- [[id:" task-id "][Split legacy relation]]\n"))
      (with-temp-file moved-root
        (insert "#+TITLE: Moved Root\n\n"
                "* Moved Legacy\n:PROPERTIES:\n:ID:       LEGACY-MOVED\n:TYPE: 설계\n:END:\nBody.\n"))
      (with-temp-file replacement
        (insert "#+TITLE: Replacement\n\nBody\n"))
      (setq task-before (imoogi-project-notes-domain-boundary-test--read tasks)
            orphan-before (imoogi-project-notes-domain-boundary-test--read orphan)
            moved-before (imoogi-project-notes-domain-boundary-test--read moved-root))
      (imoogi-project-notes-domain-boundary-test--goto-heading
       "Split legacy relation")
      (should-error
       (imoogi-project-notes-unlink-artifact
        `((id . "LEGACY-MOVED")
          (file . ,moved-root)
          (title . "Moved Legacy")))
       :type 'user-error)
      (should (equal task-before
                     (imoogi-project-notes-domain-boundary-test--read tasks)))
      (should (equal orphan-before
                     (imoogi-project-notes-domain-boundary-test--read orphan)))
      (should (equal moved-before
                     (imoogi-project-notes-domain-boundary-test--read moved-root)))
      (imoogi-project-notes-repair-link "LEGACY-MOVED" replacement)
      (let ((task-text (imoogi-project-notes-domain-boundary-test--read tasks))
            (replacement-text
             (imoogi-project-notes-domain-boundary-test--read replacement)))
        (should (equal orphan-before
                       (imoogi-project-notes-domain-boundary-test--read orphan)))
        (should-not (string-match-p "LEGACY-MOVED" task-text))
        (should (string-match-p "Replacement" task-text))
        (should (string-match-p "Split legacy relation" replacement-text))))))

(ert-deftest imoogi-project-notes-domain-copy-rejects-symlink-escape-destination-before-writing-or-publishing-ids ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (doc (expand-file-name "references/symlink-source.org" dir-a))
           (link (expand-file-name "artifacts/escape.org" dir-b))
           (outside-target (expand-file-name "outside-created.org" sandbox))
           published)
      (make-directory (file-name-directory doc) t)
      (make-directory (file-name-directory link) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       ESCAPE-SOURCE\n:END:\n\n#+TITLE: Symlink Source\n"))
      (make-symbolic-link outside-target link)
      (find-file doc)
      (cl-letf (((symbol-function 'imoogi-project-notes--copy-destination-file)
                 (lambda (&rest _args) link))
                ((symbol-function 'imoogi-project-notes--publish-copy-ids)
                 (lambda (&rest _args) (setq published t))))
        (should-error (imoogi-project-notes-copy-to-project entry-b 'file)
                      :type 'user-error))
      (should-not published)
      (should-not (file-exists-p outside-target)))))

(ert-deftest imoogi-project-notes-domain-create-artifact-unlink-copy-preserves-final-link-contract ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-a (imoogi-project-notes--find-entry-by-notes-directory dir-a))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (tasks (expand-file-name "tasks.org" dir-a))
           artifact-file artifact-id copied-text)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Create unlink copy\n")
      (save-buffer)
      (imoogi-project-notes-domain-boundary-test--goto-heading
       "Create unlink copy")
      (imoogi-project-notes-create-artifact 'design "Created Design" entry-a)
      (setq artifact-file (buffer-file-name)
            artifact-id
            (progn
              (goto-char (point-min))
              (re-search-forward "^[[:space:]]*:ID:[[:space:]]+\\([^[:space:]\n]+\\)")
              (match-string 1)))
      (find-file tasks)
      (imoogi-project-notes-domain-boundary-test--goto-heading
       "Create unlink copy")
      (imoogi-project-notes-unlink-artifact
       `((id . ,artifact-id)
         (file . ,artifact-file)
         (title . "Created Design")))
      (should-not (string-match-p artifact-id
                                  (imoogi-project-notes-domain-boundary-test--read tasks)))
      (should-not (string-match-p "Create unlink copy"
                                  (imoogi-project-notes-domain-boundary-test--read
                                   artifact-file)))
      (find-file artifact-file)
      (setq copied-text
            (imoogi-project-notes-domain-boundary-test--read
             (alist-get 'file (imoogi-project-notes-copy-to-project entry-b 'file))))
      (should (string-match-p "^\\* Link$" copied-text))
      (should-not (string-match-p artifact-id copied-text))
      (should-not (string-match-p "Create unlink copy" copied-text)))))

(ert-deftest imoogi-project-notes-domain-unlink-missing-peer-cancel-keeps-source-bytes-and-dirty-state ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks (expand-file-name "tasks.org" dir-a))
           before)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Cancel missing peer\n산출물:\n- [[id:MISSING-CANCEL][Missing Cancel]]\n")
      (save-buffer)
      (setq before (imoogi-project-notes-domain-boundary-test--read tasks))
      (imoogi-project-notes-domain-boundary-test--goto-heading
       "Cancel missing peer")
      (cl-letf (((symbol-function 'completing-read)
                 (lambda (&rest _args) "cancel")))
        (should-error
         (imoogi-project-notes-unlink-artifact
          '((id . "MISSING-CANCEL")
            (file . nil)
            (title . "Missing Cancel")))
         :type 'user-error))
      (should (equal before
                     (imoogi-project-notes-domain-boundary-test--read tasks)))
      (should-not (buffer-modified-p (find-buffer-visiting tasks))))))

(ert-deftest imoogi-project-notes-domain-explicit-other-entry-link-unlink-repair-rejects-without-change ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (doc-a (expand-file-name "references/a-doc.org" dir-a))
           (new-doc-a (expand-file-name "references/new-doc.org" dir-a))
           before-task before-doc before-new)
      (make-directory (file-name-directory doc-a) t)
      (with-temp-file doc-a
        (insert ":PROPERTIES:\n:ID:       A-DOC\n:END:\n\n#+TITLE: A Doc\n"
                "* Link\n- [[id:A-TASK][A task]]\n"))
      (with-temp-file new-doc-a
        (insert ":PROPERTIES:\n:ID:       NEW-A-DOC\n:END:\n\n#+TITLE: New A Doc\n"))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO A task\n:PROPERTIES:\n:ID:       A-TASK\n:END:\n산출물:\n- [[id:A-DOC][A Doc]]\n")
      (save-buffer)
      (setq before-task (imoogi-project-notes-domain-boundary-test--read tasks-a)
            before-doc (imoogi-project-notes-domain-boundary-test--read doc-a)
            before-new (imoogi-project-notes-domain-boundary-test--read new-doc-a))
      (imoogi-project-notes-domain-boundary-test--goto-heading "A task")
      (should-error (imoogi-project-notes-link-artifact doc-a entry-b)
                    :type 'user-error)
      (should-error
       (imoogi-project-notes-unlink-artifact
        `((id . "A-DOC") (file . ,doc-a) (title . "A Doc"))
        entry-b)
       :type 'user-error)
      (should-error (imoogi-project-notes-repair-link "A-DOC" new-doc-a entry-b)
                    :type 'user-error)
      (should (equal before-task
                     (imoogi-project-notes-domain-boundary-test--read tasks-a)))
      (should (equal before-doc
                     (imoogi-project-notes-domain-boundary-test--read doc-a)))
      (should (equal before-new
                     (imoogi-project-notes-domain-boundary-test--read new-doc-a))))))

(ert-deftest imoogi-project-notes-domain-unknown-project-id-rejects-even-with-one-tasks-owner ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (doc-a (expand-file-name "references/known.org" dir-a))
           before-task before-doc)
      (make-directory (file-name-directory doc-a) t)
      (with-temp-file doc-a
        (insert ":PROPERTIES:\n:ID:       KNOWN-DOC\n:END:\n\n#+TITLE: Known\n"))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Unknown owner\n:PROPERTIES:\n:IMOOGI_PROJECT_ID: NO-SUCH-PROJECT\n:END:\n")
      (save-buffer)
      (setq before-task (imoogi-project-notes-domain-boundary-test--read tasks-a)
            before-doc (imoogi-project-notes-domain-boundary-test--read doc-a))
      (imoogi-project-notes-domain-boundary-test--goto-heading "Unknown owner")
      (should-error (imoogi-project-notes-link-artifact doc-a)
                    :type 'user-error)
      (should (equal before-task
                     (imoogi-project-notes-domain-boundary-test--read tasks-a)))
      (should (equal before-doc
                     (imoogi-project-notes-domain-boundary-test--read doc-a))))))

(ert-deftest imoogi-project-notes-domain-live-dirty-heading-id-collision-rejects-link-before-mutation ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (target (expand-file-name "references/target.org" dir-a))
           (collider (expand-file-name "references/collider.org" dir-a))
           before-task before-target collider-buffer)
      (make-directory (file-name-directory target) t)
      (with-temp-file target
        (insert ":PROPERTIES:\n:ID:       LIVE-COLLIDE\n:END:\n\n#+TITLE: Target\n"))
      (with-temp-file collider
        (insert "#+TITLE: Collider\n\n* Placeholder\n"))
      (setq collider-buffer (find-file-noselect collider))
      (with-current-buffer collider-buffer
        (goto-char (point-max))
        (insert "\n* Dirty Collision\n:PROPERTIES:\n:ID:       LIVE-COLLIDE\n:END:\n"))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Live collision\n")
      (save-buffer)
      (setq before-task (imoogi-project-notes-domain-boundary-test--read tasks-a)
            before-target (imoogi-project-notes-domain-boundary-test--read target))
      (imoogi-project-notes-domain-boundary-test--goto-heading "Live collision")
      (should-error (imoogi-project-notes-link-artifact target)
                    :type 'user-error)
      (should (equal before-task
                     (imoogi-project-notes-domain-boundary-test--read tasks-a)))
      (should (equal before-target
                     (imoogi-project-notes-domain-boundary-test--read target)))
      (should (buffer-modified-p collider-buffer)))))

(ert-deftest imoogi-project-notes-domain-live-unsaved-task-outbound-relation-protects-document-refile ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (doc-a (expand-file-name "references/live-outbound.org" dir-a))
           (rfloc (list "Tasks" tasks-b nil 1))
           orig-called doc-before)
      (make-directory (file-name-directory doc-a) t)
      (with-temp-file doc-a
        (insert ":PROPERTIES:\n:ID:       LIVE-OUTBOUND-DOC\n:END:\n\n#+TITLE: Live Outbound\n"))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Live outbound task\n:PROPERTIES:\n:ID:       LIVE-OUTBOUND-TASK\n:END:\n산출물:\n- [[id:LIVE-OUTBOUND-DOC][Live Outbound]]\n")
      (find-file doc-a)
      (setq doc-before (imoogi-project-notes-domain-boundary-test--read doc-a))
      (cl-letf (((symbol-function 'completing-read)
                 (lambda (&rest _args) "cancel")))
        (should-error
         (imoogi-project-notes--org-refile-around
          (lambda (&rest _args)
            (setq orig-called t)
            (error "original org-refile must not run"))
          nil nil rfloc nil)
         :type 'user-error))
      (should-not orig-called)
      (should (equal doc-before
                     (imoogi-project-notes-domain-boundary-test--read doc-a)))
      (should-not (imoogi-project-notes-domain-boundary-test--artifact-files dir-b)))))

(ert-deftest imoogi-project-notes-domain-central-copied-task-uses-destination-category-not-retained-source-category ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((imoogi-project-notes-todo-storage 'central)
           (dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-a (imoogi-project-notes--find-entry-by-notes-directory dir-a))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (agenda (imoogi-project-notes--alist-string 'tasks-file entry-a))
           copied-file copied-category)
      (find-file agenda)
      (goto-char (point-max))
      (insert "\n* TODO Central copy category\n:PROPERTIES:\n:ID:       CENTRAL-CAT-TASK\n:IMOOGI_PROJECT_ID: "
              (imoogi-project-notes--entry-derived-note-id entry-a)
              "\n:CATEGORY: "
              (imoogi-project-notes--entry-name entry-a)
              "\n:END:\n산출물:\n- [[id:CENTRAL-CAT-DOC][Doc]]\n")
      (save-buffer)
      (imoogi-project-notes-domain-boundary-test--goto-heading
       "Central copy category")
      (setq copied-file
            (alist-get 'file (imoogi-project-notes-copy-to-project
                              entry-b 'subtree)))
      (with-current-buffer (find-file-noselect copied-file)
        (goto-char (point-min))
        (re-search-forward "^\\* TODO Central copy category$")
        (setq copied-category (org-get-category)))
      (should (equal copied-category
                     (imoogi-project-notes--entry-name entry-b)))
      (should-not (string-match-p
                   (concat ":CATEGORY:[[:space:]]+"
                           (regexp-quote
                            (imoogi-project-notes--entry-name entry-a)))
                   (imoogi-project-notes-domain-boundary-test--read copied-file))))))

(ert-deftest imoogi-project-notes-domain-central-ambiguous-target-prompts-candidates-and-does-not-mutate-agenda ()
  (imoogi-project-notes-domain-boundary-test--isolated
    (let* ((imoogi-project-notes-todo-storage 'central)
           (dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-a (imoogi-project-notes--find-entry-by-notes-directory dir-a))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (agenda (imoogi-project-notes--alist-string 'tasks-file entry-a))
           target-pos before-agenda seen-candidates)
      (should (equal agenda (imoogi-project-notes--alist-string 'tasks-file entry-b)))
      (find-file agenda)
      (goto-char (point-max))
      (insert "\n* TODO Ambiguous source\n:PROPERTIES:\n:ID:       AMBIG-SOURCE\n:IMOOGI_PROJECT_ID: "
              (imoogi-project-notes--entry-derived-note-id entry-a)
              "\n:CATEGORY: "
              (imoogi-project-notes--entry-name entry-a)
              "\n:END:\n산출물:\n- [[id:AMBIG-DOC][Doc]]\n"
              "\n* TODO Ambiguous target\n")
      (save-buffer)
      (setq before-agenda (imoogi-project-notes-domain-boundary-test--read agenda))
      (imoogi-project-notes-domain-boundary-test--goto-heading "Ambiguous target")
      (setq target-pos (point))
      (imoogi-project-notes-domain-boundary-test--goto-heading "Ambiguous source")
      (cl-letf (((symbol-function 'completing-read)
                 (lambda (prompt collection &rest _args)
                   (cond
                    ((string-match-p "프로젝트" prompt)
                     (setq seen-candidates
                           (mapcar (lambda (choice)
                                     (imoogi-project-notes--entry-name
                                      (cdr choice)))
                                   collection))
                     (caar (cl-member-if
                            (lambda (choice)
                              (string= (imoogi-project-notes--entry-name entry-b)
                                       (imoogi-project-notes--entry-name
                                        (cdr choice))))
                            collection)))
                    ((string-match-p "동작" prompt) "copy")
                    (t (error "unexpected prompt: %s" prompt))))))
        (let ((org-refile-keep t))
          (org-refile nil nil (list "Ambiguous target" agenda nil target-pos))))
      (should (equal (sort seen-candidates #'string<)
                     (sort (list (imoogi-project-notes--entry-name entry-a)
                                 (imoogi-project-notes--entry-name entry-b))
                           #'string<)))
      (should (equal before-agenda
                     (imoogi-project-notes-domain-boundary-test--read agenda)))
      (should (= 1 (length (imoogi-project-notes-domain-boundary-test--artifact-files
                            dir-b)))))))

;;; project-notes-domain-boundary-test.el ends here
