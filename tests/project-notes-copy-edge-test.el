;;; project-notes-copy-edge-test.el --- Project notes copy/refile edge tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'org)
(require 'org-refile)

(defmacro imoogi-project-notes-copy-edge-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename (make-temp-file "imoogi-project-notes-copy-edge-" t)))
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

(defun imoogi-project-notes-copy-edge-test--read (file)
  "Return FILE content."
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(defun imoogi-project-notes-copy-edge-test--goto-heading (title)
  "Move point to TODO heading TITLE."
  (goto-char (point-min))
  (re-search-forward (concat "^\\* TODO " (regexp-quote title) "$"))
  (beginning-of-line))

(defun imoogi-project-notes-copy-edge-test--artifact-files (directory)
  "Return copied Org artifact files below DIRECTORY."
  (let ((artifact-dir (expand-file-name "artifacts/" directory)))
    (and (file-directory-p artifact-dir)
         (directory-files-recursively artifact-dir "\\.org\\'"))))

(defun imoogi-project-notes-copy-edge-test--roam-node (file)
  "Return a real org-roam node object for FILE."
  (unless (fboundp 'org-roam-node-create)
    (require 'org-roam-node))
  (org-roam-node-create :file file :id "edge-node" :title "Edge Node"))

(ert-deftest imoogi-project-notes-refile-keep-variable-copies-with-fresh-ids-without-moving-original ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (rfloc (list "Tasks" tasks-b nil 1))
           source-before)
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Keep copy\n:PROPERTIES:\n:ID:       KEEP-OLD\n:END:\n산출물:\n- [[id:DOC-OLD][Doc]]\nBody.\n")
      (save-buffer)
      (setq source-before (imoogi-project-notes-copy-edge-test--read tasks-a))
      (imoogi-project-notes-copy-edge-test--goto-heading "Keep copy")
      (let ((org-refile-keep t))
        (org-refile nil nil rfloc))
      (should (equal source-before
                     (imoogi-project-notes-copy-edge-test--read tasks-a)))
      (let* ((copies (imoogi-project-notes-copy-edge-test--artifact-files dir-b))
             (text (imoogi-project-notes-copy-edge-test--read (car copies))))
        (should (= 1 (length copies)))
        (should (string-match-p "Keep copy" text))
        (should-not (string-match-p "KEEP-OLD\\|DOC-OLD" text))))))

(ert-deftest imoogi-project-notes-refile-universal-visit-prefix-does-not-copy-or_mutate ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (rfloc (list "Tasks" tasks-b nil 1))
           source-before dest-before)
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Visit only\n:PROPERTIES:\n:ID:       VISIT-OLD\n:END:\n산출물:\n- [[id:DOC-OLD][Doc]]\n")
      (save-buffer)
      (setq source-before (imoogi-project-notes-copy-edge-test--read tasks-a)
            dest-before (imoogi-project-notes-copy-edge-test--read tasks-b))
      (imoogi-project-notes-copy-edge-test--goto-heading "Visit only")
      (org-refile '(4) nil rfloc)
      (should (equal source-before
                     (imoogi-project-notes-copy-edge-test--read tasks-a)))
      (should (equal dest-before
                     (imoogi-project-notes-copy-edge-test--read tasks-b)))
      (should-not (imoogi-project-notes-copy-edge-test--artifact-files dir-b)))))

(ert-deftest imoogi-project-notes-copy-legacy-artifact-root-includes-sibling-link-and_keeps_original ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (doc (expand-file-name "artifacts/legacy.org" dir-a))
           original copied copied-text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Legacy Artifact\n\n"
                "* Legacy Root\n:PROPERTIES:\n:ID:       LEGACY-ROOT\n:TYPE: 설계\n:END:\nRoot body.\n"
                "** Detail\n:PROPERTIES:\n:ID:       DETAIL-OLD\n:END:\n[[id:DETAIL-OLD][self]]\n"
                "* Link\n- [[id:TASK-OLD][Old task]]\nSibling note.\n"))
      (setq original (imoogi-project-notes-copy-edge-test--read doc))
      (find-file doc)
      (goto-char (point-min))
      (re-search-forward "^\\* Legacy Root$")
      (beginning-of-line)
      (setq copied (imoogi-project-notes-copy-to-project entry-b)
            copied-text (imoogi-project-notes-copy-edge-test--read
                         (alist-get 'file copied)))
      (should (equal original (imoogi-project-notes-copy-edge-test--read doc)))
      (should (file-exists-p doc))
      (should (string-match-p "^\\* Legacy Root$" copied-text))
      (should (string-match-p "^\\* Link$" copied-text))
      (should (string-match-p "Sibling note\\." copied-text))
      (should-not (string-match-p "LEGACY-ROOT\\|DETAIL-OLD\\|TASK-OLD"
                                  copied-text)))))

(ert-deftest imoogi-project-notes-refile-one-sided-document-link-is-protected ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (doc (expand-file-name "references/one-sided.org" dir-a))
           (rfloc (list "Tasks" tasks-b nil 1))
           original)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       ONE-SIDED-DOC\n:END:\n\n#+TITLE: One Sided\n\n* Link\n- [[id:TASK-ONLY][Task only]]\n"))
      (setq original (imoogi-project-notes-copy-edge-test--read doc))
      (find-file doc)
      (goto-char (point-min))
      (let ((org-refile-keep t))
        (org-refile nil nil rfloc))
      (should (equal original (imoogi-project-notes-copy-edge-test--read doc)))
      (let* ((copies (imoogi-project-notes-copy-edge-test--artifact-files dir-b))
             (text (imoogi-project-notes-copy-edge-test--read (car copies))))
        (should (= 1 (length copies)))
        (should (string-match-p "One Sided" text))
        (should-not (string-match-p "ONE-SIDED-DOC\\|TASK-ONLY" text))))))

(ert-deftest imoogi-project-notes-refile-central-agenda-cross-project-copy-uses-heading_owner ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((imoogi-project-notes-todo-storage 'central)
           (dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-a (imoogi-project-notes--find-entry-by-notes-directory dir-a))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (agenda (imoogi-project-notes--alist-string 'tasks-file entry-a))
           q-heading-position before-agenda)
      (should (equal agenda (imoogi-project-notes--alist-string 'tasks-file entry-b)))
      (find-file agenda)
      (goto-char (point-max))
      (insert "\n* TODO P managed\n:PROPERTIES:\n:ID:       CENTRAL-P\n:CATEGORY: "
              (imoogi-project-notes--entry-name entry-a)
              "\n:END:\n산출물:\n- [[id:DOC-P][Doc P]]\n"
              "\n* TODO Q destination\n:PROPERTIES:\n:CATEGORY: "
              (imoogi-project-notes--entry-name entry-b)
              "\n:END:\n")
      (save-buffer)
      (setq before-agenda (imoogi-project-notes-copy-edge-test--read agenda))
      (imoogi-project-notes-copy-edge-test--goto-heading "Q destination")
      (setq q-heading-position (point))
      (imoogi-project-notes-copy-edge-test--goto-heading "P managed")
      (let ((org-refile-keep t))
        (org-refile nil nil (list "Q destination" agenda nil q-heading-position)))
      (should (equal before-agenda
                     (imoogi-project-notes-copy-edge-test--read agenda)))
      (let* ((copies (imoogi-project-notes-copy-edge-test--artifact-files dir-b))
             (text (and copies
                        (imoogi-project-notes-copy-edge-test--read
                         (car copies)))))
        (should (= 1 (length copies)))
        (should (string-match-p "P managed" text))
        (should-not (string-match-p "CENTRAL-P\\|DOC-P" text))))))

(ert-deftest imoogi-project-notes-refile-same-project-prefix3-copies-with-fresh-ids ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           target-pos before)
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Same prefix3 source\n:PROPERTIES:\n:ID:       SAME-OLD\n:END:\n산출물:\n- [[id:SAME-DOC][Doc]]\nBody.\n"
              "\n* TODO Same destination\n")
      (save-buffer)
      (setq before (imoogi-project-notes-copy-edge-test--read tasks-a))
      (imoogi-project-notes-copy-edge-test--goto-heading "Same destination")
      (setq target-pos (point))
      (imoogi-project-notes-copy-edge-test--goto-heading "Same prefix3 source")
      (org-refile 3 nil (list "Same destination" tasks-a nil target-pos))
      (let ((after (imoogi-project-notes-copy-edge-test--read tasks-a)))
        (should (string-match-p (regexp-quote before) after))
        (should (= 1 (cl-loop with start = 0
                              while (string-match ":ID:[[:space:]]+SAME-OLD"
                                                  after start)
                              count t
                              do (setq start (match-end 0)))))
        (should-not (string-match-p "\\*\\* TODO Same prefix3 source\\(?:.\\|\n\\)*SAME-DOC"
                                    after))))))

(ert-deftest imoogi-project-notes-refile-same-project-keep-copies-with-fresh-ids ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           target-pos before)
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Same keep source\n:PROPERTIES:\n:ID:       KEEP-SAME-OLD\n:END:\n산출물:\n- [[id:KEEP-SAME-DOC][Doc]]\nBody.\n"
              "\n* TODO Keep destination\n")
      (save-buffer)
      (setq before (imoogi-project-notes-copy-edge-test--read tasks-a))
      (imoogi-project-notes-copy-edge-test--goto-heading "Keep destination")
      (setq target-pos (point))
      (imoogi-project-notes-copy-edge-test--goto-heading "Same keep source")
      (let ((org-refile-keep t))
        (org-refile nil nil (list "Keep destination" tasks-a nil target-pos)))
      (let ((after (imoogi-project-notes-copy-edge-test--read tasks-a)))
        (should (string-match-p (regexp-quote before) after))
        (should (= 1 (cl-loop with start = 0
                              while (string-match ":ID:[[:space:]]+KEEP-SAME-OLD"
                                                  after start)
                              count t
                              do (setq start (match-end 0)))))
        (should-not (string-match-p "\\*\\* TODO Same keep source\\(?:.\\|\n\\)*KEEP-SAME-DOC"
                                    after))))))

(ert-deftest imoogi-project-notes-copy-to-project-partial-region-rejects-without-file ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (tasks-a (expand-file-name "tasks.org" dir-a)))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Direct partial\n:PROPERTIES:\n:ID:       DIRECT-PARTIAL\n:END:\n산출물:\n- [[id:DIRECT-DOC][Doc]]\nBody line\n")
      (save-buffer)
      (re-search-backward "Body line")
      (set-mark (point))
      (forward-char 4)
      (activate-mark)
      (should-error (imoogi-project-notes-copy-to-project entry-b)
                    :type 'user-error)
      (should-not (imoogi-project-notes-copy-edge-test--artifact-files dir-b)))))

(ert-deftest imoogi-project-notes-org-roam-refile-partial-region-never-calls-original ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (target (expand-file-name "references/roam-target.org" dir-b))
           (source-before nil)
           (orig-called nil))
      (make-directory (file-name-directory target) t)
      (with-temp-file target
        (insert "#+TITLE: Roam Target\n"))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Roam partial\n산출물:\n- [[id:ROAM-DOC][Doc]]\nBody line\n")
      (save-buffer)
      (setq source-before (imoogi-project-notes-copy-edge-test--read tasks-a))
      (re-search-backward "Body line")
      (set-mark (point))
      (forward-char 4)
      (activate-mark)
      (should-error
       (imoogi-project-notes--org-roam-refile-around
        (lambda (&rest _args)
          (setq orig-called t)
          (error "original org-roam-refile should not run"))
        (imoogi-project-notes-copy-edge-test--roam-node target))
       :type 'user-error)
      (should-not orig-called)
      (should (equal source-before
                     (imoogi-project-notes-copy-edge-test--read tasks-a)))
      (should-not (imoogi-project-notes-copy-edge-test--artifact-files dir-b)))))


(ert-deftest imoogi-project-notes-refile-central-agenda-unknown-project-id-does-not-fallback-to-category ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((imoogi-project-notes-todo-storage 'central)
           (dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-a (imoogi-project-notes--find-entry-by-notes-directory dir-a))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (agenda (imoogi-project-notes--alist-string 'tasks-file entry-a))
           target-pos before-agenda)
      (should (equal agenda (imoogi-project-notes--alist-string 'tasks-file entry-b)))
      (find-file agenda)
      (goto-char (point-max))
      (insert "\n* TODO Unknown source\n:PROPERTIES:\n:ID:       UNKNOWN-SOURCE\n:CATEGORY: "
              (imoogi-project-notes--entry-name entry-a)
              "\n:END:\n산출물:\n- [[id:UNKNOWN-DOC][Doc]]\n"
              "\n* TODO Unknown target\n:PROPERTIES:\n:IMOOGI_PROJECT_ID: DOES-NOT-EXIST\n:CATEGORY: "
              (imoogi-project-notes--entry-name entry-b)
              "\n:END:\n")
      (save-buffer)
      (setq before-agenda (imoogi-project-notes-copy-edge-test--read agenda))
      (imoogi-project-notes-copy-edge-test--goto-heading "Unknown target")
      (setq target-pos (point))
      (imoogi-project-notes-copy-edge-test--goto-heading "Unknown source")
      (let ((org-refile-keep t))
        (should-error (org-refile nil nil (list "Unknown target" agenda nil target-pos))
                      :type 'user-error))
      (should (equal before-agenda
                     (imoogi-project-notes-copy-edge-test--read agenda)))
      (should-not (imoogi-project-notes-copy-edge-test--artifact-files dir-b)))))

(ert-deftest imoogi-project-notes-copy-whole-file-region-at-end-captures-entire-file ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (doc (expand-file-name "artifacts/whole.org" dir-a))
           before copied-text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       WHOLE-FILE-ID\n:END:\n\n#+TITLE: Whole File\n\n"
                "Intro body survives.\n"
                "* Body Heading\n:PROPERTIES:\n:ID:       WHOLE-HEADING-ID\n:END:\n[[id:WHOLE-HEADING-ID][self]]\n"
                "* Link\n- [[id:WHOLE-TASK][Whole task]]\n"))
      (setq before (imoogi-project-notes-copy-edge-test--read doc))
      (find-file doc)
      (goto-char (point-max))
      (push-mark (point-min) t t)
      (activate-mark)
      (let ((copied (imoogi-project-notes-copy-to-project entry-b)))
        (setq copied-text (imoogi-project-notes-copy-edge-test--read
                           (alist-get 'file copied))))
      (should (equal before (imoogi-project-notes-copy-edge-test--read doc)))
      (should (string-match-p "#\\+TITLE: Whole File" copied-text))
      (should (string-match-p "Intro body survives\." copied-text))
      (should (string-match-p "^\\* Body Heading$" copied-text))
      (should (string-match-p "^\\* Link$" copied-text))
      (should-not (string-match-p "WHOLE-FILE-ID\\|WHOLE-HEADING-ID\\|WHOLE-TASK"
                                  copied-text)))))

(ert-deftest imoogi-project-notes-org-roam-same-project-legacy-root-with-sibling-link-is-blocked ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (doc (expand-file-name "artifacts/legacy-split.org" dir-a))
           (target (expand-file-name "references/roam-same-project.org" dir-a))
           (before nil)
           (orig-called nil))
      (make-directory (file-name-directory doc) t)
      (make-directory (file-name-directory target) t)
      (with-temp-file doc
        (insert "#+TITLE: Legacy Split\n\n"
                "* Legacy Root\n:PROPERTIES:\n:ID:       LEGACY-SPLIT\n:TYPE: 설계\n:END:\nRoot body.\n"
                "* Link\n- [[id:SAME-ROAM-TASK][Same roam task]]\n"))
      (with-temp-file target
        (insert "#+TITLE: Same Project Target\n"))
      (setq before (imoogi-project-notes-copy-edge-test--read doc))
      (find-file doc)
      (goto-char (point-min))
      (re-search-forward "^\\* Legacy Root$")
      (beginning-of-line)
      (should-error
       (imoogi-project-notes--org-roam-refile-around
        (lambda (&rest _args)
          (setq orig-called t)
          (error "original org-roam-refile should not run"))
        (imoogi-project-notes-copy-edge-test--roam-node target))
       :type 'user-error)
      (should-not orig-called)
      (should (equal before (imoogi-project-notes-copy-edge-test--read doc))))))


(ert-deftest imoogi-project-notes-refile-task-outbound-link-without-document-backlink-is-protected ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (doc (expand-file-name "references/no-backlink.org" dir-a))
           (rfloc (list "Tasks" tasks-b nil 1))
           before)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       NO-BACKLINK-DOC\n:END:\n\n#+TITLE: No Backlink\n\nBody only.\n"))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Outbound only\n:PROPERTIES:\n:ID:       OUTBOUND-TASK\n:END:\n산출물:\n- [[id:NO-BACKLINK-DOC][No Backlink]]\n")
      (save-buffer)
      (setq before (imoogi-project-notes-copy-edge-test--read tasks-a))
      (imoogi-project-notes-copy-edge-test--goto-heading "Outbound only")
      (let ((org-refile-keep t))
        (org-refile nil nil rfloc))
      (should (equal before (imoogi-project-notes-copy-edge-test--read tasks-a)))
      (let* ((copies (imoogi-project-notes-copy-edge-test--artifact-files dir-b))
             (text (and copies (imoogi-project-notes-copy-edge-test--read (car copies)))))
        (should (= 1 (length copies)))
        (should (string-match-p "Outbound only" text))
        (should-not (string-match-p "OUTBOUND-TASK\\|NO-BACKLINK-DOC" text))))))

(ert-deftest imoogi-project-notes-refile-task-inbound-document-link-without-task-list-is-protected ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (doc (expand-file-name "references/inbound-only.org" dir-a))
           (rfloc (list "Tasks" tasks-b nil 1))
           before-source before-dest)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       INBOUND-DOC\n:END:\n\n#+TITLE: Inbound Only\n\n* Link\n- [[id:INBOUND-TASK][Inbound only]]\n"))
      (find-file tasks-a)
      (goto-char (point-max))
      (insert "\n* TODO Inbound only\n:PROPERTIES:\n:ID:       INBOUND-TASK\n:END:\nTask has no local 산출물 list.\n")
      (save-buffer)
      (setq before-source (imoogi-project-notes-copy-edge-test--read tasks-a)
            before-dest (imoogi-project-notes-copy-edge-test--read tasks-b))
      (imoogi-project-notes-copy-edge-test--goto-heading "Inbound only")
      (let ((org-refile-keep t))
        (org-refile nil nil rfloc))
      (should (equal before-source (imoogi-project-notes-copy-edge-test--read tasks-a)))
      (should (equal before-dest (imoogi-project-notes-copy-edge-test--read tasks-b)))
      (let* ((copies (imoogi-project-notes-copy-edge-test--artifact-files dir-b))
             (text (and copies (imoogi-project-notes-copy-edge-test--read (car copies)))))
        (should (= 1 (length copies)))
        (should (string-match-p "Inbound only" text))
        (should-not (string-match-p "INBOUND-TASK" text))))))

(ert-deftest imoogi-project-notes-copy-final-file-collision-fails-without-id-publication ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (doc (expand-file-name "references/collision-source.org" dir-a))
           (collision (expand-file-name "artifacts/final-collision.org" dir-b))
           published)
      (make-directory (file-name-directory doc) t)
      (make-directory (file-name-directory collision) t)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       COLLISION-SOURCE-ID\n:END:\n\n#+TITLE: Collision Source\n"))
      (with-temp-file collision
        (insert "#+TITLE: Existing Collision\n"))
      (find-file doc)
      (cl-letf (((symbol-function 'imoogi-project-notes--copy-destination-file)
                 (lambda (&rest _args) collision))
                ((symbol-function 'imoogi-project-notes--publish-copy-ids)
                 (lambda (&rest _args) (setq published t))))
        (should-error (imoogi-project-notes-copy-to-project entry-b 'file)
                      :type 'user-error))
      (should-not published)
      (should (string= "#+TITLE: Existing Collision\n"
                       (imoogi-project-notes-copy-edge-test--read collision))))))

(ert-deftest imoogi-project-notes-refile-prefix2-uses-running-clock-destination-for-protected-task ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-a (expand-file-name "tasks.org" dir-a))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           before-source before-dest)
      (unwind-protect
          (progn
            (find-file tasks-b)
            (goto-char (point-max))
            (insert "\n* TODO Clock destination\n")
            (save-buffer)
            (imoogi-project-notes-copy-edge-test--goto-heading "Clock destination")
            (org-clock-in)
            (find-file tasks-a)
            (goto-char (point-max))
            (insert "\n* TODO Clocked protected\n:PROPERTIES:\n:ID:       CLOCKED-PROTECTED\n:END:\n산출물:\n- [[id:CLOCKED-DOC][Doc]]\n")
            (save-buffer)
            (setq before-source (imoogi-project-notes-copy-edge-test--read tasks-a)
                  before-dest (imoogi-project-notes-copy-edge-test--read tasks-b))
            (imoogi-project-notes-copy-edge-test--goto-heading "Clocked protected")
            (cl-letf (((symbol-function 'org-refile-get-location)
                       (lambda (&rest _args)
                         (error "prefix 2 should use the running clock destination")))
                      ((symbol-function 'completing-read)
                       (lambda (prompt collection &rest _args)
                         (cond
                          ((string-match-p "동작" prompt) "copy")
                          (t (error "unexpected prompt: %s" prompt))))))
              (org-refile 2))
            (should (equal before-source
                           (imoogi-project-notes-copy-edge-test--read tasks-a)))
            (should (equal before-dest
                           (imoogi-project-notes-copy-edge-test--read tasks-b)))
            (let* ((copies (imoogi-project-notes-copy-edge-test--artifact-files dir-b))
                   (text (and copies (imoogi-project-notes-copy-edge-test--read (car copies)))))
              (should (= 1 (length copies)))
              (should (string-match-p "Clocked protected" text))
              (should-not (string-match-p "CLOCKED-PROTECTED\\|CLOCKED-DOC" text))))
        (when (and (boundp 'org-clock-current-task) org-clock-current-task)
          (ignore-errors (org-clock-out)))))))


(ert-deftest imoogi-project-notes-org-roam-document-containing-task-artifact-without-backlink-is-protected ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (doc (expand-file-name "artifacts/document-task-only.org" dir-a))
           (target (expand-file-name "references/roam-target.org" dir-b))
           (before nil)
           (orig-called nil))
      (make-directory (file-name-directory doc) t)
      (make-directory (file-name-directory target) t)
      (with-temp-file doc
        (insert "#+TITLE: Document Task Only\n\n"
                "* TODO Embedded managed task\n:PROPERTIES:\n:ID:       DOC-TASK-ONLY\n:END:\n산출물:\n- [[id:DOC-ONLY-TARGET][Doc only target]]\n"))
      (with-temp-file target
        (insert "#+TITLE: Roam Target\n"))
      (setq before (imoogi-project-notes-copy-edge-test--read doc))
      (find-file doc)
      (goto-char (point-min))
      (should-error
       (imoogi-project-notes--org-roam-refile-around
        (lambda (&rest _args)
          (setq orig-called t)
          (error "original org-roam-refile should not run"))
        (imoogi-project-notes-copy-edge-test--roam-node target))
       :type 'user-error)
      (should-not orig-called)
      (should (equal before (imoogi-project-notes-copy-edge-test--read doc)))
      (should-not (imoogi-project-notes-copy-edge-test--artifact-files dir-b)))))

(ert-deftest imoogi-project-notes-copy-complete-subtree-region-at-end-excludes-next-sibling ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (entry-b (imoogi-project-notes--find-entry-by-notes-directory dir-b))
           (doc (expand-file-name "artifacts/subtree-region.org" dir-a))
           copied-text)
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Subtree Region\n\n"
                "* TODO First complete\n:PROPERTIES:\n:ID:       FIRST-COMPLETE\n:END:\n산출물:\n- [[id:FIRST-DOC][First doc]]\nBody.\n"
                "* TODO Next sibling\n:PROPERTIES:\n:ID:       NEXT-SIBLING\n:END:\nShould not copy.\n"))
      (find-file doc)
      (goto-char (point-min))
      (re-search-forward "^\\* TODO First complete$")
      (beginning-of-line)
      (let ((beg (point))
            (end (save-excursion (org-end-of-subtree t t))))
        (goto-char end)
        (push-mark beg t t)
        (activate-mark))
      (let ((copied (imoogi-project-notes-copy-to-project entry-b)))
        (setq copied-text (imoogi-project-notes-copy-edge-test--read
                           (alist-get 'file copied))))
      (should (string-match-p "First complete" copied-text))
      (should (string-match-p "Body\." copied-text))
      (should-not (string-match-p "FIRST-COMPLETE\\|FIRST-DOC" copied-text))
      (should-not (string-match-p "Next sibling\\|NEXT-SIBLING\\|Should not copy"
                                  copied-text)))))


(ert-deftest imoogi-project-notes-org-refile-complete-legacy-root-region-at-end-protects-sibling-link ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (tasks-b (expand-file-name "tasks.org" dir-b))
           (doc (expand-file-name "artifacts/legacy-region-refile.org" dir-a))
           (before nil)
           (orig-called nil))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Legacy Region Refile\n\n"
                "* Legacy Root\n:PROPERTIES:\n:ID:       LEGACY-REGION-REFILE\n:TYPE: 설계\n:END:\nRoot body.\n"
                "* Link\n- [[id:LEGACY-REGION-TASK][Legacy region task]]\n"))
      (setq before (imoogi-project-notes-copy-edge-test--read doc))
      (find-file doc)
      (goto-char (point-min))
      (re-search-forward "^\\* Legacy Root$")
      (beginning-of-line)
      (let ((beg (point))
            (end (save-excursion (org-end-of-subtree t t))))
        (goto-char end)
        (push-mark beg t t)
        (activate-mark))
      (let ((org-refile-keep t))
        (imoogi-project-notes--org-refile-around
         (lambda (&rest _args)
           (setq orig-called t)
           (error "original org-refile should not run"))
         nil nil (list "Tasks" tasks-b nil 1) nil))
      (should-not orig-called)
      (should (equal before (imoogi-project-notes-copy-edge-test--read doc)))
      (should (string-match-p "^\\* Link$"
                              (imoogi-project-notes-copy-edge-test--read doc)))
      (let* ((copies (imoogi-project-notes-copy-edge-test--artifact-files dir-b))
             (text (and copies (imoogi-project-notes-copy-edge-test--read (car copies)))))
        (should (= 1 (length copies)))
        (should (string-match-p "Legacy Root" text))
        (should-not (string-match-p "LEGACY-REGION-REFILE\\|LEGACY-REGION-TASK" text))))))

(ert-deftest imoogi-project-notes-org-roam-complete-legacy-root-region-at-beginning-protects-sibling-link ()
  (imoogi-project-notes-copy-edge-test--isolated
    (let* ((dir-a (imoogi-project-notes-setup root-a))
           (dir-b (imoogi-project-notes-setup root-b))
           (doc (expand-file-name "artifacts/legacy-region-roam.org" dir-a))
           (target (expand-file-name "references/roam-region-target.org" dir-b))
           (before nil)
           (orig-called nil))
      (make-directory (file-name-directory doc) t)
      (make-directory (file-name-directory target) t)
      (with-temp-file doc
        (insert "#+TITLE: Legacy Region Roam\n\n"
                "* Legacy Root\n:PROPERTIES:\n:ID:       LEGACY-REGION-ROAM\n:TYPE: 설계\n:END:\nRoot body.\n"
                "* Link\n- [[id:LEGACY-REGION-ROAM-TASK][Legacy region roam task]]\n"))
      (with-temp-file target
        (insert "#+TITLE: Roam Region Target\n"))
      (setq before (imoogi-project-notes-copy-edge-test--read doc))
      (find-file doc)
      (goto-char (point-min))
      (re-search-forward "^\\* Legacy Root$")
      (beginning-of-line)
      (let ((beg (point))
            (end (save-excursion (org-end-of-subtree t t))))
        (set-mark end)
        (goto-char beg)
        (activate-mark))
      (cl-letf (((symbol-function 'completing-read)
                 (lambda (prompt collection &rest _args)
                   (cond
                    ((string-match-p "동작" prompt) "copy")
                    (t (error "unexpected prompt: %s" prompt))))))
        (imoogi-project-notes--org-roam-refile-around
         (lambda (&rest _args)
           (setq orig-called t)
           (error "original org-roam-refile should not run"))
         (imoogi-project-notes-copy-edge-test--roam-node target)))
      (should-not orig-called)
      (should (equal before (imoogi-project-notes-copy-edge-test--read doc)))
      (should (string-match-p "^\\* Link$"
                              (imoogi-project-notes-copy-edge-test--read doc)))
      (let* ((copies (imoogi-project-notes-copy-edge-test--artifact-files dir-b))
             (text (and copies (imoogi-project-notes-copy-edge-test--read (car copies)))))
        (should (= 1 (length copies)))
        (should (string-match-p "Legacy Root" text))
        (should-not (string-match-p "LEGACY-REGION-ROAM\\|LEGACY-REGION-ROAM-TASK" text))))))

;;; project-notes-copy-edge-test.el ends here
