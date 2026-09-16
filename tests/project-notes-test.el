;;; project-notes-test.el --- Project notes integration tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)

(defmacro imoogi-project-notes-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename (make-temp-file "imoogi-project-notes-" t)))
          (user-emacs-directory (expand-file-name "emacs/" sandbox))
          (imoogi-project-notes-directory (expand-file-name "project-notes/" sandbox))
          (imoogi-project-notes-todo-storage 'project)
          (personal (expand-file-name "notes/" sandbox))
          (root (file-name-as-directory (expand-file-name "source/" sandbox)))
          (org-id-locations-file (expand-file-name "org-id-locations" sandbox))
          (org-agenda-files nil)
          (org-directory personal))
     (make-directory root t)
     (unwind-protect
         (cl-letf (((symbol-function 'imoogi-org--default-directory)
                    (lambda () personal)))
           (save-window-excursion
             (with-temp-buffer
               (setq default-directory root)
               ,@body)))
       (dolist (buffer (buffer-list))
         (when (and (buffer-file-name buffer)
                    (file-in-directory-p (buffer-file-name buffer) sandbox))
           (with-current-buffer buffer (set-buffer-modified-p nil))
           (kill-buffer buffer)))
       (delete-directory sandbox t))))

(defun imoogi-project-notes-test--read (file)
  (with-temp-buffer (insert-file-contents file) (buffer-string)))

(ert-deftest imoogi-project-notes-setup-preserves-and-registers ()
  (imoogi-project-notes-test--isolated
    (cl-letf (((symbol-function 'imoogi-project-notes--start-date)
               (lambda () "260918")))
      (setq org-agenda-files (list (expand-file-name "existing.org" sandbox)))
      (let* ((directory (imoogi-project-notes-setup root))
             (tasks (expand-file-name "tasks.org" directory))
             (overview (expand-file-name "project.org" directory))
             (metadata-file (expand-file-name ".imoogi-project.json" directory)))
        (should (equal (file-name-nondirectory (directory-file-name directory))
                       "260918-source"))
        (should (equal (sort (directory-files directory nil "\\.org\\'") #'string<)
                       '("journal.org" "project.org" "tasks.org")))
        (should (file-directory-p (expand-file-name "assets" directory)))
        (should (file-directory-p (expand-file-name "references" directory)))
        (let ((metadata (imoogi-project-notes--read-metadata-file metadata-file)))
          (should (equal (alist-get 'type metadata) "project"))
          (should (equal (alist-get 'source_root metadata) root))
          (should (equal (alist-get 'overview metadata) "project.org")))
        (should-not (file-exists-p (expand-file-name "development" directory)))
        (should-not (string-match-p "{{" (imoogi-project-notes-test--read overview)))
        (should (member tasks org-agenda-files))
        (should (member (expand-file-name "existing.org" sandbox) org-agenda-files))
        (with-temp-file overview (insert "* My existing overview\n"))
        (should (equal directory (imoogi-project-notes-setup root)))
        (should (equal "* My existing overview\n"
                       (imoogi-project-notes-test--read overview)))
        (setq org-agenda-files nil)
        (imoogi-project-notes--restore-agenda-files)
        (should (equal org-agenda-files (list tasks)))))))

(ert-deftest imoogi-project-notes-default-directory-collision-keeps-date-prefix ()
  (imoogi-project-notes-test--isolated
    (cl-letf (((symbol-function 'imoogi-project-notes--start-date)
               (lambda () "260918")))
      (let ((first (imoogi-project-notes-setup root))
            (other (expand-file-name "elsewhere/source/" sandbox)))
        (make-directory other t)
        (let ((second (imoogi-project-notes-setup other)))
          (should (string-match-p "/260918-source/\\'" first))
          (should (string-match-p "/260918-source-[0-9a-f]\\{10\\}/\\'" second)))))))

(ert-deftest imoogi-project-notes-study-setup-needs-no-source-project ()
  (imoogi-project-notes-test--isolated
    (let (perspective-call treemacs-call)
      (cl-letf (((symbol-function 'imoogi-project-notes--study-year)
                 (lambda () "26"))
                ((symbol-function 'persp-switch)
                 (lambda (name) (setq perspective-call name)))
                ((symbol-function 'imoogi-project-perspective-name)
                 (lambda (directory)
                   (concat "study:" (file-name-nondirectory
                                     (directory-file-name directory)))))
                ((symbol-function 'imoogi-treemacs-open-project-workspace)
                 (lambda (directory name)
                   (setq treemacs-call (list directory name)))))
        (let* ((directory (imoogi-project-notes-setup-study
                           "Operating Systems" nil "26.01" "2026-01-15"))
               (entry (car (imoogi-project-notes--read-registry)))
               (study (expand-file-name "study.org" directory))
               (metadata-file (expand-file-name ".imoogi-project.json" directory)))
          (should (string-match-p "/26\\.01-operating-systems/\\'" directory))
          (should (equal (alist-get 'type entry) "study"))
          (should (equal (alist-get 'study-id entry) "26.01"))
          (should (equal (alist-get 'source-root entry) directory))
          (let ((metadata (imoogi-project-notes--read-metadata-file metadata-file)))
            (should (equal (alist-get 'type metadata) "study"))
            (should (equal (alist-get 'study_id metadata) "26.01"))
            (should (equal (alist-get 'overview metadata) "study.org"))
            (should-not (assq 'source_root metadata)))
          (should (equal buffer-file-name study))
          (should (equal perspective-call "study:26.01-operating-systems"))
          (should (equal treemacs-call
                         (list directory "study:26.01-operating-systems")))
          (dolist (file '("study.org" "tasks.org" "cards.org" "questions.org"
                          "logs/journal.org"))
            (should (file-exists-p (expand-file-name file directory))))
          (dolist (subdir '("materials/books" "materials/handouts"
                            "materials/articles" "materials/slides"
                            "materials/videos" "logs" "concepts"
                            "assignments" "assets"))
            (should (file-directory-p (expand-file-name subdir directory))))
          (let ((contents (imoogi-project-notes-test--read study)))
            (should (string-match-p "^#\\+STUDY_ID: 26\\.01$" contents))
            (should (string-match-p "^#\\+START_DATE: 2026-01-15$" contents)))
          (should (member (expand-file-name "tasks.org" directory)
                          org-agenda-files))
          (with-current-buffer (find-buffer-visiting study)
            (erase-buffer)
            (insert "* 기존 학습 기록\n")
            (save-buffer))
          (should (equal directory
                         (imoogi-project-notes-setup-study
                          "Operating Systems" directory "26.01" "2026-01-16")))
          (should (equal "* 기존 학습 기록\n"
                         (imoogi-project-notes-test--read study))))))))

(ert-deftest imoogi-project-notes-folder-metadata-recovers-study-context ()
  (imoogi-project-notes-test--isolated
    (let* ((directory (imoogi-project-notes-setup-study
                       "Operating Systems" nil "26.01" "2026-01-15"))
           (concept (expand-file-name "concepts/C01-process.org" directory))
           (registry (imoogi-project-notes--registry-file)))
      (with-temp-file concept (insert "#+TITLE: Process\n"))
      (delete-file registry)
      (with-temp-buffer
        (setq buffer-file-name concept)
        (let ((entry (imoogi-project-notes--current-entry)))
          (should (equal (alist-get 'type entry) "study"))
          (should (equal (alist-get 'study-id entry) "26.01"))
          (should (equal (alist-get 'project-file entry)
                         (expand-file-name "study.org" directory))))))))

(ert-deftest imoogi-project-notes-folder-metadata-keeps-document-paths-inside-root ()
  (imoogi-project-notes-test--isolated
    (let* ((directory (imoogi-project-notes-setup-study
                       "Operating Systems" nil "26.01" "2026-01-15"))
           (file (expand-file-name ".imoogi-project.json" directory))
           (metadata (imoogi-project-notes--read-metadata-file file)))
      (setf (alist-get 'tasks metadata) "../outside.org")
      (should-error (imoogi-project-notes--metadata-entry file metadata)
                    :type 'user-error))))

(ert-deftest imoogi-project-notes-study-id-increments-by-existing-folders ()
  (imoogi-project-notes-test--isolated
    (make-directory (expand-file-name "26.01-first/" imoogi-project-notes-directory) t)
    (make-directory (expand-file-name "26.07-seventh/" imoogi-project-notes-directory) t)
    (make-directory (expand-file-name "25.99-old/" imoogi-project-notes-directory) t)
    (should (equal (imoogi-project-notes--next-study-id "26") "26.08"))))

(ert-deftest imoogi-project-notes-rejects-a-notes-directory-as-source-root ()
  (imoogi-project-notes-test--isolated
    (let ((notes-root (expand-file-name "existing-notes/" imoogi-project-notes-directory)))
      (make-directory notes-root t)
      (should-error (imoogi-project-notes-setup notes-root) :type 'user-error))))

(ert-deftest imoogi-project-notes-first-open-and-optional-documents ()
  (imoogi-project-notes-test--isolated
    (cl-letf (((symbol-function 'imoogi-project-notes--project-root) (lambda () root)))
      (imoogi-project-notes-open))
    (should (equal (file-name-nondirectory buffer-file-name) "project.org"))
    (let ((directory (file-name-directory buffer-file-name)))
      (imoogi-project-notes-add-document 'domain)
      (should (equal buffer-file-name (expand-file-name "development/domain.org" directory)))
      (goto-char (point-max))
      (insert "\nUnsaved personal domain detail\n")
      (imoogi-project-notes-add-document 'domain)
      (should (string-match-p "Unsaved personal domain detail" (buffer-string)))
      (should (buffer-modified-p)))))

(ert-deftest imoogi-project-notes-central-storage-stays-central ()
  (imoogi-project-notes-test--isolated
    (setq imoogi-project-notes-todo-storage 'central)
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (agenda (expand-file-name "agenda.org" personal))
           (local-tasks (expand-file-name "tasks.org" directory)))
      (should (equal (alist-get 'tasks-file entry) agenda))
      (should (file-exists-p agenda))
      (should (string-match-p "agenda\\.org" (imoogi-project-notes-test--read local-tasks)))
      (should-not (string-match-p "^#\\+TODO:" (imoogi-project-notes-test--read local-tasks)))
      (setq imoogi-project-notes-todo-storage 'project)
      (imoogi-project-notes-setup root)
      (should (equal (alist-get 'tasks-file (car (imoogi-project-notes--read-registry))) agenda)))))

(ert-deftest imoogi-project-notes-distinct-roots-and-custom-directory ()
  (imoogi-project-notes-test--isolated
    (let ((other (expand-file-name "elsewhere/source/" sandbox))
          (custom (file-name-as-directory (expand-file-name "custom-notes" sandbox))))
      (make-directory other t)
      (let ((first (imoogi-project-notes-setup root)))
        (should-not (equal first (imoogi-project-notes-setup other)))
        (should-error (imoogi-project-notes-setup other first) :type 'user-error))
      (should (equal custom (imoogi-project-notes-setup root custom)))
      (should (equal custom (imoogi-project-notes-setup root))))))

(ert-deftest imoogi-project-notes-corrupt-registry-is-preserved ()
  (imoogi-project-notes-test--isolated
    (let ((registry (imoogi-project-notes--registry-file)))
      (make-directory (file-name-directory registry) t)
      (with-temp-file registry (insert "broken registry; preserve me"))
      (should-error (imoogi-project-notes-setup root))
      (should (equal (imoogi-project-notes-test--read registry)
                     "broken registry; preserve me")))))

(ert-deftest imoogi-project-notes-string-agenda-and-read-only-restore ()
  (imoogi-project-notes-test--isolated
    (let ((store (expand-file-name "agenda-list" sandbox)))
      (with-temp-file store (insert (expand-file-name "existing.org" sandbox) "\n"))
      (setq org-agenda-files store)
      (let* ((directory (imoogi-project-notes-setup root))
             (contents (imoogi-project-notes-test--read store)))
        (should (equal org-agenda-files store))
        (should (string-match-p (regexp-quote (expand-file-name "tasks.org" directory)) contents))
        (imoogi-project-notes--restore-agenda-files)
        (should (equal contents (imoogi-project-notes-test--read store)))))))

(ert-deftest imoogi-project-notes-scratch-and-startup-do-not-overwrite ()
  (imoogi-project-notes-test--isolated
    (imoogi-project-notes--restore-agenda-files)
    (should-not (file-exists-p imoogi-project-notes-directory))
    (should-not (file-exists-p (imoogi-project-notes--registry-file)))
    (imoogi-notes-scratch)
    (should (equal buffer-file-name (expand-file-name "scratch.org" personal)))
    (insert "My unsaved thought\n")
    (imoogi-notes-scratch)
    (should (string-match-p "My unsaved thought" (buffer-string)))
    (should (buffer-modified-p))))

(ert-deftest imoogi-project-notes-worktrees-share-tasks-not-resume-section ()
  (skip-unless (executable-find "git"))
  (imoogi-project-notes-test--isolated
    (let ((default-directory root)
          (worktree (expand-file-name "second-worktree/" sandbox)))
      (should (zerop (call-process "git" nil nil nil "init" "-q")))
      (should (zerop (call-process "git" nil nil nil "-c" "user.name=Test"
                                  "-c" "user.email=test@example.invalid" "commit"
                                  "--allow-empty" "--no-gpg-sign" "-qm" "fixture")))
      (should (zerop (call-process "git" nil nil nil "worktree" "add" "--detach" worktree)))
      (let ((directory (imoogi-project-notes-setup root)))
        (should (equal directory (imoogi-project-notes-setup worktree)))
        (should (= 1 (length (imoogi-project-notes--read-registry))))
        (let ((default-directory root)) (imoogi-project-notes-journal))
        (let ((default-directory worktree)) (imoogi-project-notes-journal))
        (let ((default-directory worktree)) (imoogi-project-notes-journal))
        (should (= 2 (how-many "^\\*+ 작업 공간:" (point-min) (point-max))))
        (goto-char (point-min))
        (re-search-forward "^\\*\\* 작업 공간:")
        (org-up-heading-safe)
        (should (equal (org-get-heading t t t t) "진행 기록"))
        (should (string-match-p (regexp-quote (directory-file-name root)) (buffer-string)))
        (should (string-match-p (regexp-quote (directory-file-name worktree)) (buffer-string)))
        (should (buffer-modified-p))))))

(ert-deftest imoogi-project-notes-transient-commands-available ()
  (should (eq (plist-get (cdr (transient-get-suffix 'imoogi-transient-project "m")) :command)
              'imoogi-project-notes-transient))
  (dolist (key '("s" "S" "o" "t" "j" "l" "a" "A" "r" "d" "n" "h"))
    (should (commandp (plist-get (cdr (transient-get-suffix 'imoogi-project-notes-transient key))
                                 :command)))))

(ert-deftest imoogi-project-notes-artifact-links-task-both-ways ()
  (imoogi-project-notes-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (tasks (expand-file-name "tasks.org" directory)))
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO 모델 조회 구현\n완료 조건:\n- 목록을 선택한다.\n")
      (forward-line -3)
      (imoogi-project-notes-create-artifact 'design "모델 조회 설계")
      (let* ((artifact buffer-file-name)
             (task-text (imoogi-project-notes-test--read tasks))
             (artifact-text (imoogi-project-notes-test--read artifact)))
        (should (file-in-directory-p artifact
                                     (expand-file-name "artifacts/" directory)))
        (should (string-match-p "산출물:" task-text))
        (should (string-match-p
                 "\\[\\[id:[^]]+\\]\\[모델 조회 설계\\]\\]" task-text))
        (should (string-match-p "\\* 모델 조회 설계" artifact-text))
        (should (string-match-p "\\*\\* 관련 작업" artifact-text))
        (should (string-match-p
                 "\\[\\[id:[^]]+\\]\\[모델 조회 구현\\]\\]" artifact-text))
        (should (string-match-p "\\*\\* 검토한 대안" artifact-text))))))

(ert-deftest imoogi-project-notes-agenda-files-cover-each-project-once ()
  (imoogi-project-notes-test--isolated
    (let ((other (file-name-as-directory (expand-file-name "other/" sandbox))))
      (make-directory other t)
      (imoogi-project-notes-setup root)
      (imoogi-project-notes-setup other)
      (let ((files (imoogi-project-notes--agenda-files)))
        (should (= (length files) 2))
        (should (cl-every #'file-exists-p files))))))

;;; project-notes-test.el ends here
