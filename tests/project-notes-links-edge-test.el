;;; project-notes-links-edge-test.el --- Project notes document link edge tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'org)

(defmacro imoogi-project-notes-links-edge-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename (make-temp-file "imoogi-project-notes-links-edge-" t)))
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
               (setq default-directory root)
               ,@body)))
       (dolist (buffer (buffer-list))
         (when (and (buffer-file-name buffer)
                    (file-in-directory-p (buffer-file-name buffer) sandbox))
           (with-current-buffer buffer
             (set-buffer-modified-p nil))
           (kill-buffer buffer)))
       (delete-directory sandbox t))))

(defun imoogi-project-notes-links-edge-test--read (file)
  "Return FILE content."
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(defun imoogi-project-notes-links-edge-test--hash (text)
  "Return a stable hash for TEXT."
  (secure-hash 'sha1 text))

(defun imoogi-project-notes-links-edge-test--goto-heading (title)
  "Move point to TODO heading TITLE."
  (goto-char (point-min))
  (re-search-forward (concat "^\\* TODO " (regexp-quote title) "$"))
  (beginning-of-line))

(defun imoogi-project-notes-links-edge-test--id-count (text)
  "Return number of Org ID property lines in TEXT."
  (let ((start 0)
        (count 0))
    (while (string-match "^[[:space:]]*:ID:[[:space:]]+" text start)
      (setq count (1+ count)
            start (match-end 0)))
    count))

(ert-deftest imoogi-project-notes-link-artifact-cancelled-selection-does-not-create-ids-or-dirty-buffers ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/candidate.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Candidate\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Cancel me\n")
      (save-buffer)
      (imoogi-project-notes-links-edge-test--goto-heading "Cancel me")
      (let ((task-buffer (current-buffer))
            (quit-seen nil))
        (cl-letf (((symbol-function 'completing-read)
                   (lambda (&rest _args) (signal 'quit nil))))
          (condition-case nil
              (imoogi-project-notes-link-artifact)
            (quit (setq quit-seen t))))
        (should quit-seen)
        (should-not (buffer-modified-p task-buffer))
        (should (= 0 (imoogi-project-notes-links-edge-test--id-count
                      (imoogi-project-notes-links-edge-test--read tasks))))
        (should (= 0 (imoogi-project-notes-links-edge-test--id-count
                      (imoogi-project-notes-links-edge-test--read doc))))
        (should-not (string-match-p "산출물:"
                                    (imoogi-project-notes-links-edge-test--read tasks)))
        (should-not (string-match-p "^\\* Link$"
                                    (imoogi-project-notes-links-edge-test--read doc)))))))

(ert-deftest imoogi-project-notes-link-artifact-does-not-treat-ordinary-todo-id-as-document-id ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/ordinary.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Ordinary\n\n* TODO Ordinary child\n:PROPERTIES:\n:ID:       CHILD-ID\n:END:\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Uses ordinary doc\n")
      (save-buffer)
      (imoogi-project-notes-links-edge-test--goto-heading "Uses ordinary doc")
      (imoogi-project-notes-link-artifact doc)
      (let* ((task-text (imoogi-project-notes-links-edge-test--read tasks))
             (doc-text (imoogi-project-notes-links-edge-test--read doc))
             (file-id (progn
                        (string-match "\\`:PROPERTIES:\n:ID:[[:space:]]+\\([^[:space:]\n]+\\)"
                                      doc-text)
                        (match-string 1 doc-text))))
        (should file-id)
        (should-not (string= file-id "CHILD-ID"))
        (should (string-match-p (regexp-quote (format "[[id:%s][Ordinary]]" file-id))
                                task-text))
        (should-not (string-match-p "\\[\\[id:CHILD-ID\\]\\[Ordinary\\]\\]"
                                    task-text))
        (should (string-match-p "^\\* TODO Ordinary child$" doc-text))))))

(ert-deftest imoogi-project-notes-link-artifact-rejects-duplicate-document-ids ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc-a (expand-file-name "references/a.org" directory))
           (doc-b (expand-file-name "development/b.org" directory)))
      (make-directory (file-name-directory doc-a) t)
      (make-directory (file-name-directory doc-b) t)
      (with-temp-file doc-a
        (insert ":PROPERTIES:\n:ID:       DUPLICATE-DOC\n:END:\n\n#+TITLE: A\n"))
      (with-temp-file doc-b
        (insert ":PROPERTIES:\n:ID:       DUPLICATE-DOC\n:END:\n\n#+TITLE: B\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Duplicate target\n")
      (save-buffer)
      (imoogi-project-notes-links-edge-test--goto-heading "Duplicate target")
      (should-error (imoogi-project-notes-link-artifact doc-a)
                    :type 'user-error)
      (should-not (string-match-p "DUPLICATE-DOC"
                                  (imoogi-project-notes-links-edge-test--read tasks)))
      (should-not (string-match-p "^\\* Link$"
                                  (imoogi-project-notes-links-edge-test--read doc-a))))))

(ert-deftest imoogi-project-notes-link-artifact-inactive-mounted-entry-does-not-mutate-task-or-document ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (inactive-entry (append `((origin . mounted) (active-p . nil))
                                   entry))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/inactive.org" directory)))
      (make-directory (file-name-directory doc) t)
      (with-temp-file doc
        (insert "#+TITLE: Inactive\n\nBody\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Inactive mounted\n")
      (save-buffer)
      (imoogi-project-notes-links-edge-test--goto-heading "Inactive mounted")
      (should-error (imoogi-project-notes-link-artifact doc inactive-entry)
                    :type 'user-error)
      (should (= 0 (imoogi-project-notes-links-edge-test--id-count
                    (imoogi-project-notes-links-edge-test--read tasks))))
      (should (= 0 (imoogi-project-notes-links-edge-test--id-count
                    (imoogi-project-notes-links-edge-test--read doc))))
      (should-not (string-match-p "산출물:"
                                  (imoogi-project-notes-links-edge-test--read tasks)))
      (should-not (string-match-p "^\\* Link$"
                                  (imoogi-project-notes-links-edge-test--read doc))))))

(ert-deftest imoogi-project-notes-unlink-artifact-missing-peer-removes-source-and-warns ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           warning-message)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Missing peer\n산출물:\n- [[id:MISSING-DOC][Missing Doc]]\n")
      (save-buffer)
      (imoogi-project-notes-links-edge-test--goto-heading "Missing peer")
      (cl-letf (((symbol-function 'display-warning)
                 (lambda (_type message &rest _args)
                   (setq warning-message message))))
        (imoogi-project-notes-unlink-artifact
         '((id . "MISSING-DOC") (file . nil) (title . "Missing Doc"))))
      (let ((task-text (imoogi-project-notes-links-edge-test--read tasks)))
        (should warning-message)
        (should (string-match-p "대상 문서를 찾지 못해" warning-message))
        (should-not (string-match-p "MISSING-DOC" task-text))))))

(ert-deftest imoogi-project-notes-document-catalog-excludes-symlink-escape ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (outside (expand-file-name "outside.org" sandbox))
           (link (expand-file-name "references/outside.org" directory)))
      (make-directory (file-name-directory link) t)
      (with-temp-file outside
        (insert "#+TITLE: Outside\n"))
      (make-symbolic-link outside link)
      (let ((files (mapcar (lambda (file)
                             (file-relative-name file directory))
                           (imoogi-project-notes--project-document-files entry))))
        (should-not (member "references/outside.org" files))))))

(ert-deftest imoogi-project-notes-link-artifact-parent-creates-direct-list-without-moving-child-list ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (parent-doc (expand-file-name "references/parent.org" directory))
           (child-doc (expand-file-name "references/child.org" directory)))
      (make-directory (file-name-directory parent-doc) t)
      (with-temp-file parent-doc
        (insert ":PROPERTIES:\n:ID:       PARENT-DOC\n:END:\n\n#+TITLE: Parent Doc\n"))
      (with-temp-file child-doc
        (insert ":PROPERTIES:\n:ID:       CHILD-DOC\n:END:\n\n#+TITLE: Child Doc\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Parent task\nParent body.\n** TODO Child task\n산출물:\n- [[id:CHILD-DOC][Child Doc]]\n")
      (save-buffer)
      (imoogi-project-notes-links-edge-test--goto-heading "Parent task")
      (imoogi-project-notes-link-artifact parent-doc)
      (let* ((task-text (imoogi-project-notes-links-edge-test--read tasks))
             (child-start (string-match "^\\*\\* TODO Child task$" task-text))
             (parent-slice (substring task-text 0 child-start))
             (child-slice (substring task-text child-start)))
        (should (string-match-p "\\[\\[id:PARENT-DOC\\]\\[Parent Doc\\]\\]"
                                parent-slice))
        (should-not (string-match-p "\\[\\[id:PARENT-DOC\\]\\[Parent Doc\\]\\]"
                                    child-slice))
        (should (string-match-p "\\[\\[id:CHILD-DOC\\]\\[Child Doc\\]\\]"
                                child-slice))
        (should-not (string-match-p "\\[\\[id:CHILD-DOC\\]\\[Child Doc\\]\\]"
                                    parent-slice))))))

(ert-deftest imoogi-project-notes-unlink-and-repair-candidates-use-only-current-task-direct-artifacts ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (parent-doc (expand-file-name "references/parent.org" directory))
           (child-doc (expand-file-name "references/child.org" directory))
           unlink-ids repair-ids)
      (make-directory (file-name-directory parent-doc) t)
      (with-temp-file parent-doc
        (insert ":PROPERTIES:\n:ID:       PARENT-DOC\n:END:\n\n#+TITLE: Parent Doc\n"))
      (with-temp-file child-doc
        (insert ":PROPERTIES:\n:ID:       CHILD-DOC\n:END:\n\n#+TITLE: Child Doc\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Parent choices\nSee [[id:PROSE-ID][ordinary prose]].\n산출물:\n- [[id:PARENT-DOC][Parent Doc]]\n** TODO Child choices\n산출물:\n- [[id:CHILD-DOC][Child Doc]]\n")
      (save-buffer)
      (imoogi-project-notes-links-edge-test--goto-heading "Parent choices")
      (cl-letf (((symbol-function 'completing-read)
                 (lambda (prompt collection &rest _args)
                   (cond
                    ((string-match-p "연결 해제할 문서" prompt)
                     (setq unlink-ids
                           (mapcar (lambda (choice)
                                     (alist-get 'id (cdr choice)))
                                   collection)))
                    ((string-match-p "교체할 누락 ID" prompt)
                     (setq repair-ids collection)))
                   (signal 'quit nil))))
        (condition-case nil
            (imoogi-project-notes-unlink-artifact)
          (quit nil))
        (condition-case nil
            (call-interactively #'imoogi-project-notes-repair-link)
          (quit nil)))
      (should (equal unlink-ids '("PARENT-DOC")))
      (should (equal repair-ids '("PARENT-DOC"))))))

(ert-deftest imoogi-project-notes-link-artifact-duplicate-id-failure-leaves-open-buffers-unchanged ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc-a (expand-file-name "references/a.org" directory))
           (doc-b (expand-file-name "development/b.org" directory)))
      (make-directory (file-name-directory doc-a) t)
      (make-directory (file-name-directory doc-b) t)
      (with-temp-file doc-a
        (insert ":PROPERTIES:\n:ID:       DUPLICATE-DOC\n:END:\n\n#+TITLE: A\n"))
      (with-temp-file doc-b
        (insert ":PROPERTIES:\n:ID:       DUPLICATE-DOC\n:END:\n\n#+TITLE: B\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Duplicate must not mutate\n")
      (save-buffer)
      (imoogi-project-notes-links-edge-test--goto-heading "Duplicate must not mutate")
      (let* ((task-buffer (current-buffer))
             (doc-buffer (find-file-noselect doc-a))
             (task-before (buffer-string))
             (doc-before (with-current-buffer doc-buffer (buffer-string)))
             (task-file-before (imoogi-project-notes-links-edge-test--read tasks))
             (doc-file-before (imoogi-project-notes-links-edge-test--read doc-a)))
        (should-error (imoogi-project-notes-link-artifact doc-a)
                      :type 'user-error)
        (should (string= task-before
                         (with-current-buffer task-buffer (buffer-string))))
        (should (string= doc-before
                         (with-current-buffer doc-buffer (buffer-string))))
        (should (string= task-file-before
                         (imoogi-project-notes-links-edge-test--read tasks)))
        (should (string= doc-file-before
                         (imoogi-project-notes-links-edge-test--read doc-a)))
        (should-not (buffer-modified-p task-buffer))
        (should-not (buffer-modified-p doc-buffer))))))

(ert-deftest imoogi-project-notes-public-link-commands-do-not-silently-save-unrelated-dirty-targets ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (source (expand-file-name "project.org" directory))
           (link-doc (expand-file-name "references/link-dirty.org" directory))
           (insert-doc (expand-file-name "references/insert-dirty.org" directory))
           (unlink-doc (expand-file-name "references/unlink-dirty.org" directory)))
      (make-directory (file-name-directory link-doc) t)
      (with-temp-file link-doc
        (insert ":PROPERTIES:\n:ID:       LINK-DIRTY\n:END:\n\n#+TITLE: Link Dirty\n"))
      (with-temp-file insert-doc
        (insert ":PROPERTIES:\n:ID:       INSERT-DIRTY\n:END:\n\n#+TITLE: Insert Dirty\n"))
      (with-temp-file unlink-doc
        (insert ":PROPERTIES:\n:ID:       UNLINK-DIRTY\n:END:\n\n#+TITLE: Unlink Dirty\n* Link\n- [[id:TASK-DIRTY][Dirty Task]]\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Dirty link target\n\n* TODO Dirty unlink target\n:PROPERTIES:\n:ID:       TASK-DIRTY\n:END:\n산출물:\n- [[id:UNLINK-DIRTY][Unlink Dirty]]\n")
      (save-buffer)
      (let* ((link-disk-before (imoogi-project-notes-links-edge-test--read link-doc))
             (insert-disk-before (imoogi-project-notes-links-edge-test--read insert-doc))
             (unlink-disk-before (imoogi-project-notes-links-edge-test--read unlink-doc))
             (link-buffer (find-file-noselect link-doc))
             (insert-buffer (find-file-noselect insert-doc))
             (unlink-buffer (find-file-noselect unlink-doc)))
        (with-current-buffer link-buffer
          (goto-char (point-max))
          (insert "\nUNSAVED LINK TARGET EDIT\n"))
        (with-current-buffer insert-buffer
          (goto-char (point-max))
          (insert "\nUNSAVED INSERT TARGET EDIT\n"))
        (with-current-buffer unlink-buffer
          (goto-char (point-max))
          (insert "\nUNSAVED UNLINK TARGET EDIT\n"))
        (imoogi-project-notes-links-edge-test--goto-heading "Dirty link target")
        (condition-case nil
            (imoogi-project-notes-link-artifact link-doc)
          (error nil))
        (with-current-buffer (find-file-noselect source)
          (goto-char (point-max))
          (insert "\nPlain: ")
          (condition-case nil
              (imoogi-project-notes-insert-link insert-doc)
            (error nil)))
        (find-file tasks)
        (imoogi-project-notes-links-edge-test--goto-heading "Dirty unlink target")
        (condition-case nil
            (imoogi-project-notes-unlink-artifact
             `((id . "UNLINK-DIRTY")
               (file . ,unlink-doc)
               (title . "Unlink Dirty")))
          (error nil))
        (should (string= (imoogi-project-notes-links-edge-test--hash link-disk-before)
                         (imoogi-project-notes-links-edge-test--hash
                          (imoogi-project-notes-links-edge-test--read link-doc))))
        (should (string= (imoogi-project-notes-links-edge-test--hash insert-disk-before)
                         (imoogi-project-notes-links-edge-test--hash
                          (imoogi-project-notes-links-edge-test--read insert-doc))))
        (should (string= (imoogi-project-notes-links-edge-test--hash unlink-disk-before)
                         (imoogi-project-notes-links-edge-test--hash
                          (imoogi-project-notes-links-edge-test--read unlink-doc))))
        (should (buffer-modified-p link-buffer))
        (should (buffer-modified-p insert-buffer))
        (should (buffer-modified-p unlink-buffer))))))

(ert-deftest imoogi-project-notes-link-and-insert-noarg-selectors-succeed ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (source (expand-file-name "project.org" directory))
           (link-doc (expand-file-name "references/selector-link.org" directory))
           (insert-doc (expand-file-name "references/selector-insert.org" directory)))
      (make-directory (file-name-directory link-doc) t)
      (with-temp-file link-doc
        (insert ":PROPERTIES:\n:ID:       SELECT-LINK\n:END:\n\n#+TITLE: Selector Link\n"))
      (with-temp-file insert-doc
        (insert ":PROPERTIES:\n:ID:       SELECT-INSERT\n:END:\n\n#+TITLE: Selector Insert\n"))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Selector task\n")
      (save-buffer)
      (imoogi-project-notes-links-edge-test--goto-heading "Selector task")
      (cl-letf (((symbol-function 'completing-read)
                 (lambda (prompt collection &rest _args)
                   (cond
                    ((string-match-p "연결할 프로젝트 문서" prompt)
                     (caar (cl-member-if
                           (lambda (choice)
                             (string= (expand-file-name link-doc)
                                      (alist-get 'file (cdr choice))))
                           collection)))
                    (t (error "unexpected prompt: %s" prompt))))))
        (imoogi-project-notes-link-artifact))
      (with-current-buffer (find-file-noselect source)
        (goto-char (point-max))
        (insert "\nPlain selector: ")
        (cl-letf (((symbol-function 'completing-read)
                   (lambda (prompt collection &rest _args)
                     (cond
                      ((string-match-p "삽입할 프로젝트 문서 링크" prompt)
                       (caar (cl-member-if
                             (lambda (choice)
                               (string= (expand-file-name insert-doc)
                                        (alist-get 'file (cdr choice))))
                             collection)))
                      (t (error "unexpected prompt: %s" prompt))))))
          (imoogi-project-notes-insert-link))
        (save-buffer))
      (let ((task-text (imoogi-project-notes-links-edge-test--read tasks))
            (source-text (imoogi-project-notes-links-edge-test--read source))
            (link-text (imoogi-project-notes-links-edge-test--read link-doc))
            (insert-text (imoogi-project-notes-links-edge-test--read insert-doc)))
        (should (string-match-p "\\[\\[id:SELECT-LINK\\]\\[Selector Link\\]\\]"
                                task-text))
        (should (string-match-p "\\[\\[id:SELECT-INSERT\\]\\[Selector Insert\\]\\]"
                                source-text))
        (should (string-match-p "^\\* Link$" link-text))
        (should-not (string-match-p "^\\* Link$" insert-text))))))

(ert-deftest imoogi-project-notes-unlink-recognizes-historical-bare-link-and-legacy-related-lines ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "artifacts/historical.org" directory))
           task-id)
      (make-directory (file-name-directory doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Historical unlink\n:PROPERTIES:\n:ID:       HIST-TASK\n:END:\n산출물:\n- [[id:HIST-DOC][Historical]]\n")
      (save-buffer)
      (setq task-id "HIST-TASK")
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       HIST-DOC\n:END:\n\n#+TITLE: Historical\n\n"
                "* Link\n[[id:" task-id "][Historical unlink]]\n"
                "* Legacy Root\n:PROPERTIES:\n:TYPE: 설계\n:END:\n"
                "** 관련 작업\n[[id:" task-id "][Historical unlink]]\n"
                "** Body\nKeep.\n"))
      (imoogi-project-notes-links-edge-test--goto-heading "Historical unlink")
      (imoogi-project-notes-unlink-artifact
       `((id . "HIST-DOC") (file . ,doc) (title . "Historical")))
      (let ((task-text (imoogi-project-notes-links-edge-test--read tasks))
            (doc-text (imoogi-project-notes-links-edge-test--read doc)))
        (should-not (string-match-p "HIST-DOC" task-text))
        (should-not (string-match-p (regexp-quote (format "[[id:%s][Historical unlink]]" task-id))
                                    doc-text))
        (should (string-match-p "Keep\\." doc-text))))))

(ert-deftest imoogi-project-notes-unlink-preserves-unrelated-link-heading-text ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/link-preserve.org" directory)))
      (make-directory (file-name-directory doc) t)
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Preserve unlink\n:PROPERTIES:\n:ID:       PRESERVE-TASK\n:END:\n산출물:\n- [[id:PRESERVE-DOC][Preserve Doc]]\n")
      (save-buffer)
      (with-temp-file doc
        (insert ":PROPERTIES:\n:ID:       PRESERVE-DOC\n:END:\n\n#+TITLE: Preserve Doc\n\n"
                "* Link\n"
                "Plain prose should stay.\n"
                "- [[id:PRESERVE-TASK][Preserve unlink]] and [[id:KEEP-LINK][Keep Link]] stay related text\n"
                "** Nested\n- [[id:PRESERVE-TASK][Nested unrelated descendant]]\n"))
      (imoogi-project-notes-links-edge-test--goto-heading "Preserve unlink")
      (imoogi-project-notes-unlink-artifact
       `((id . "PRESERVE-DOC") (file . ,doc) (title . "Preserve Doc")))
      (let ((doc-text (imoogi-project-notes-links-edge-test--read doc)))
        (should (string-match-p "Plain prose should stay\\." doc-text))
        (should (string-match-p "\\[\\[id:KEEP-LINK\\]\\[Keep Link\\]\\] stay related text"
                                doc-text))
        (should (string-match-p "Nested unrelated descendant" doc-text))
        (should-not (string-match-p "\\[\\[id:PRESERVE-TASK\\]\\[Preserve unlink\\]\\]"
                                    doc-text))))))


(ert-deftest imoogi-project-notes-link-artifact-explicit-document-arguments-stay-inside-current-project ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory))
           (other-root (file-name-as-directory (expand-file-name "other-source/" sandbox)))
           (other-directory nil)
           (other-doc nil)
           (nested-entry nil)
           (nested-doc (expand-file-name "nested-root/doc.org" directory)))
      (make-directory other-root t)
      (setq other-directory (imoogi-project-notes-setup other-root)
            other-doc (expand-file-name "references/other.org" other-directory))
      (make-directory (file-name-directory other-doc) t)
      (with-temp-file other-doc
        (insert "#+TITLE: Other Project\n"))
      (make-directory (file-name-directory nested-doc) t)
      (with-temp-file nested-doc
        (insert "#+TITLE: Nested Other Root\n"))
      (setq nested-entry
            `((key . "nested-test")
              (type . "project")
              (name . "nested-root")
              (note-id . "NESTED-NOTE-ID")
              (source-root . ,(expand-file-name "nested-source/" sandbox))
              (notes-dir . ,(file-name-as-directory
                              (expand-file-name "nested-root/" directory)))
              (tasks-file . ,(expand-file-name "tasks.org"
                                           (expand-file-name "nested-root/" directory)))
              (todo-storage . "project")))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Explicit domain\n")
      (save-buffer)
      (imoogi-project-notes-links-edge-test--goto-heading "Explicit domain")
      (let ((original-read-registry (symbol-function 'imoogi-project-notes--read-registry)))
        (cl-letf (((symbol-function 'imoogi-project-notes--read-registry)
                   (lambda ()
                     (cons nested-entry
                           (cl-remove-if
                            (lambda (entry)
                              (string= "nested-test" (alist-get 'key entry)))
                            (funcall original-read-registry))))))
          (should-error (imoogi-project-notes-link-artifact tasks)
                        :type 'user-error)
          (should-error (imoogi-project-notes-link-artifact other-doc)
                        :type 'user-error)
          (should-error (imoogi-project-notes-link-artifact nested-doc)
                        :type 'user-error)))
      (should-not (string-match-p "산출물:"
                                  (imoogi-project-notes-links-edge-test--read tasks))))))

(ert-deftest imoogi-project-notes-insert-link-explicit-document-rejects-other-project ()
  (imoogi-project-notes-links-edge-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (source (expand-file-name "project.org" directory))
           (other-root (file-name-as-directory (expand-file-name "other-source/" sandbox)))
           (other-directory nil)
           (other-doc nil))
      (make-directory other-root t)
      (setq other-directory (imoogi-project-notes-setup other-root)
            other-doc (expand-file-name "references/plain-other.org" other-directory))
      (make-directory (file-name-directory other-doc) t)
      (with-temp-file other-doc
        (insert "#+TITLE: Plain Other\n"))
      (find-file source)
      (goto-char (point-max))
      (insert "\nPlain other: ")
      (should-error (imoogi-project-notes-insert-link other-doc)
                    :type 'user-error)
      (should-not (string-match-p "Plain Other"
                                  (buffer-string))))))

;;; project-notes-links-edge-test.el ends here
