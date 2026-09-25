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
           (with-current-buffer buffer (set-buffer-modified-p nil))
           (kill-buffer buffer)))
       (delete-directory sandbox t))))

(defun imoogi-project-notes-test--read (file)
  (with-temp-buffer (insert-file-contents file) (buffer-string)))

(defun imoogi-project-notes-test--write-metadata
    (directory key type &optional source-root study-id)
  "Write minimal project-note metadata in DIRECTORY for tests."
  (make-directory directory t)
  (with-temp-file (expand-file-name ".imoogi-project.json" directory)
    (let ((json-encoding-pretty-print t))
      (insert
       (json-encode
        `((schema_version . 1)
          (type . ,type)
          (key . ,key)
          (name . ,key)
          ,@(when study-id `((study_id . ,study-id)))
          ,@(when source-root `((source_root . ,source-root)))
          (overview . ,(if (string= type "study") "study.org" "project.org"))
          (tasks . "tasks.org")
          (journal . "journal.org")
          (todo_storage . "project"))))
      (insert "\n"))))

(ert-deftest imoogi-project-notes-mounted-state-round-trip ()
  (imoogi-project-notes-test--isolated
    (let* ((external (file-name-as-directory
                      (expand-file-name "external/" sandbox)))
           (_ (make-directory external t))
           (canonical (imoogi-project-notes--canonical-directory external))
           (root-entry `((id . ,(imoogi-project-notes--mounted-root-id canonical))
                         (path . ,canonical)
                         (label . "Portable")
                         (enabled . t)))
           (state `((version . 1)
                    (roots . (,root-entry))
                    (source-overrides . nil))))
      (imoogi-project-notes--write-mounted-state state)
      (should (equal state (imoogi-project-notes--read-mounted-state)))
      (should-not
       (equal (imoogi-project-notes--mounted-roots-file)
              (imoogi-project-notes--registry-file))))))

(ert-deftest imoogi-project-notes-mounted-state-rejects-remote-paths ()
  (imoogi-project-notes-test--isolated
    (dolist (state
             '(((version . 1)
                (roots . (((id . "remote") (path . "/ssh:host:/notes/")
                           (label . "Remote") (enabled . t))))
                (source-overrides . nil))
               ((version . 1)
                (roots . nil)
                (source-overrides
                 . (((instance-id . "mounted:one")
                     (source-root . "/ssh:host:/source/")))))))
      (should-error (imoogi-project-notes--write-mounted-state state)
                    :type 'user-error))))

(ert-deftest imoogi-project-notes-mounted-root-rejects-overlap-and-symlink ()
  (imoogi-project-notes-test--isolated
    (let* ((external (file-name-as-directory
                      (expand-file-name "external/" sandbox)))
           (child (file-name-as-directory (expand-file-name "child/" external)))
           (alias (expand-file-name "external-link" sandbox)))
      (make-directory child t)
      (make-symbolic-link external alias)
      (let* ((canonical (imoogi-project-notes--canonical-directory external))
             (roots `(((id . ,(imoogi-project-notes--mounted-root-id canonical))
                       (path . ,canonical)
                       (label . "Portable")
                       (enabled . t)))))
        (should-error (imoogi-project-notes--validate-mounted-root external roots)
                      :type 'user-error)
        (should-error (imoogi-project-notes--validate-mounted-root child roots)
                      :type 'user-error)
        (should-error (imoogi-project-notes--validate-mounted-root alias roots)
                      :type 'user-error)))))

(ert-deftest imoogi-project-notes-mounted-discovery-preserves-colliding-keys ()
  (imoogi-project-notes-test--isolated
    (let* ((external-a (file-name-as-directory
                        (expand-file-name "drive-a/" sandbox)))
           (external-b (file-name-as-directory
                        (expand-file-name "drive-b/" sandbox)))
           (study-a (expand-file-name "26.01-os/" external-a))
           (study-b (expand-file-name "26.01-os-copy/" external-b)))
      (imoogi-project-notes-test--write-metadata
       study-a "study:26.01" "study" nil "26.01")
      (imoogi-project-notes-test--write-metadata
       study-b "study:26.01" "study" nil "26.01")
      (let* ((canonical-a (imoogi-project-notes--canonical-directory external-a))
             (canonical-b (imoogi-project-notes--canonical-directory external-b))
             (state
              `((version . 1)
                (roots . (((id . ,(imoogi-project-notes--mounted-root-id canonical-a))
                           (path . ,canonical-a) (label . "A") (enabled . t))
                          ((id . ,(imoogi-project-notes--mounted-root-id canonical-b))
                           (path . ,canonical-b) (label . "B") (enabled . t))))
                (source-overrides . nil))))
        (imoogi-project-notes--write-mounted-state state)
        (let ((entries (imoogi-project-notes--all-entries)))
          (should (= (length entries) 2))
          (should (equal (mapcar (lambda (entry) (alist-get 'logical-key entry))
                                 entries)
                         '("study:26.01" "study:26.01")))
          (should (= (length (delete-dups
                              (mapcar (lambda (entry)
                                        (alist-get 'instance-id entry))
                                      entries)))
                     2)))))))

(ert-deftest imoogi-project-notes-mounted-discovery-skips-invalid-metadata ()
  (imoogi-project-notes-test--isolated
    (let* ((external (file-name-as-directory
                      (expand-file-name "external/" sandbox)))
           (valid (expand-file-name "valid/" external))
           (invalid (expand-file-name "invalid/" external)))
      (imoogi-project-notes-test--write-metadata
       valid "study:26.02" "study" nil "26.02")
      (make-directory invalid t)
      (with-temp-file (expand-file-name ".imoogi-project.json" invalid)
        (insert "{broken"))
      (let* ((canonical (imoogi-project-notes--canonical-directory external))
             (state `((version . 1)
                      (roots . (((id . ,(imoogi-project-notes--mounted-root-id
                                         canonical))
                                 (path . ,canonical)
                                 (label . "Portable")
                                 (enabled . t))))
                      (source-overrides . nil))))
        (imoogi-project-notes--write-mounted-state state)
        (let ((warning-minimum-level :emergency))
          (should (= (length (imoogi-project-notes--all-entries)) 1)))))))

(ert-deftest imoogi-project-notes-mounted-discovery-skips-unreadable-root ()
  (imoogi-project-notes-test--isolated
    (let ((root '((id . "root") (path . "/unreadable/")
                  (label . "USB") (enabled . t))))
      (cl-letf (((symbol-function 'file-directory-p) (lambda (_path) t))
                ((symbol-function 'directory-files-recursively)
                 (lambda (&rest _args)
                   (signal 'file-error '("Permission denied")))))
        (let ((warning-minimum-level :emergency))
          (should-not (imoogi-project-notes--scan-mounted-root root)))))))

(ert-deftest imoogi-project-notes-mounted-inactive-buffer-is-read-only ()
  (imoogi-project-notes-test--isolated
    (let* ((external (file-name-as-directory
                      (expand-file-name "external/" sandbox)))
           (notes (expand-file-name "project-note/" external))
           (missing (expand-file-name "missing-source/" sandbox)))
      (imoogi-project-notes-test--write-metadata
       notes "dir:portable" "project" missing)
      (dolist (name '("project.org" "tasks.org" "journal.org"))
        (with-temp-file (expand-file-name name notes) (insert "* Note\n")))
      (let* ((canonical (imoogi-project-notes--canonical-directory external))
             (state `((version . 1)
                      (roots . (((id . ,(imoogi-project-notes--mounted-root-id
                                         canonical))
                                 (path . ,canonical) (label . "USB") (enabled . t))))
                      (source-overrides . nil))))
        (imoogi-project-notes--write-mounted-state state)
        (let* ((entry (car (imoogi-project-notes--all-entries)))
               (buffer (imoogi-project-notes--find-file
                        entry (alist-get 'project-file entry))))
          (should (imoogi-project-notes--inactive-mounted-entry-p entry))
          (should buffer-read-only)
          (should header-line-format)
          (let* ((header (imoogi-project-notes--inactive-header-line))
                 (reconnect (string-match "reconnect" header)))
            (should reconnect)
            (should (eq (lookup-key (get-text-property reconnect 'local-map header)
                                    [header-line mouse-1])
                        'imoogi-project-notes-reconnect-source)))
          (imoogi-project-notes-force-edit-session)
          (should-not buffer-read-only)
          (should imoogi-project-notes-force-edit-session-p)
          (should-error
           (imoogi-project-notes--find-existing-or-create
            entry (expand-file-name "development/domain.org" notes)
            "project" nil)
           :type 'user-error)
          (kill-buffer buffer))))))

(ert-deftest imoogi-project-notes-remote-metadata-source-stays-inactive ()
  (imoogi-project-notes-test--isolated
    (let ((entry '((type . "project")
                   (origin . mounted)
                   (instance-id . "mounted:remote")
                   (source-declared-p . t)
                   (source-root . "/ssh:host:/source/")
                   (notes-dir . "/local/notes/"))))
      (cl-letf (((symbol-function 'file-directory-p)
                 (lambda (path)
                   (when (file-remote-p path)
                     (ert-fail "remote source was probed"))
                   nil)))
        (let ((decorated
               (imoogi-project-notes--decorate-mounted-status
                entry (imoogi-project-notes--empty-mounted-state))))
          (should-not (alist-get 'active-p decorated))
          (should (eq (alist-get 'source-origin decorated) 'missing)))))))

(ert-deftest imoogi-project-notes-mounted-reconnect-is-instance-scoped ()
  (imoogi-project-notes-test--isolated
    (let* ((external-a (file-name-as-directory
                        (expand-file-name "drive-a/" sandbox)))
           (external-b (file-name-as-directory
                        (expand-file-name "drive-b/" sandbox)))
           (notes-a (expand-file-name "same/" external-a))
           (notes-b (expand-file-name "same/" external-b))
           (missing (expand-file-name "missing/" sandbox))
           (new-source (expand-file-name "reconnected/" sandbox)))
      (make-directory new-source t)
      (imoogi-project-notes-test--write-metadata
       notes-a "dir:same" "project" missing)
      (imoogi-project-notes-test--write-metadata
       notes-b "dir:same" "project" missing)
      (let* ((canonical-a (imoogi-project-notes--canonical-directory external-a))
             (canonical-b (imoogi-project-notes--canonical-directory external-b))
             (state
              `((version . 1)
                (roots . (((id . ,(imoogi-project-notes--mounted-root-id canonical-a))
                           (path . ,canonical-a) (label . "A") (enabled . t))
                          ((id . ,(imoogi-project-notes--mounted-root-id canonical-b))
                           (path . ,canonical-b) (label . "B") (enabled . t))))
                (source-overrides . nil))))
        (imoogi-project-notes--write-mounted-state state)
        (let* ((entries (imoogi-project-notes--all-entries))
               (first (car entries))
               (second (cadr entries)))
          (imoogi-project-notes-reconnect-source first new-source)
          (setq entries (imoogi-project-notes--all-entries)
                first (cl-find (alist-get 'instance-id first) entries
                               :key (lambda (entry) (alist-get 'instance-id entry))
                               :test #'string=)
                second (cl-find (alist-get 'instance-id second) entries
                                :key (lambda (entry) (alist-get 'instance-id entry))
                                :test #'string=))
          (should (alist-get 'active-p first))
          (should (equal (file-truename new-source)
                         (file-truename (alist-get 'effective-source-root first))))
          (should-not (alist-get 'active-p second))
          (imoogi-project-notes-clear-source-override first)
          (should-not
           (alist-get 'active-p
                      (cl-find (alist-get 'instance-id first)
                               (imoogi-project-notes--all-entries)
                               :key (lambda (entry) (alist-get 'instance-id entry))
                               :test #'string=))))))))

(ert-deftest imoogi-project-notes-agenda-includes-mounted-tasks-once ()
  (imoogi-project-notes-test--isolated
    (let* ((external (file-name-as-directory
                      (expand-file-name "external/" sandbox)))
           (notes (expand-file-name "study/" external)))
      (imoogi-project-notes-test--write-metadata
       notes "study:26.03" "study" nil "26.03")
      (with-temp-file (expand-file-name "tasks.org" notes) (insert "* TODO Read\n"))
      (let* ((canonical (imoogi-project-notes--canonical-directory external))
             (state `((version . 1)
                      (roots . (((id . ,(imoogi-project-notes--mounted-root-id
                                         canonical))
                                 (path . ,canonical) (label . "USB") (enabled . t))))
                      (source-overrides . nil))))
        (imoogi-project-notes--write-mounted-state state)
        (should (equal (imoogi-project-notes--agenda-files)
                       (list (expand-file-name "tasks.org" notes))))))))

(ert-deftest imoogi-project-notes-mounted-root-lifecycle-preserves-files ()
  (imoogi-project-notes-test--isolated
    (let* ((external (expand-file-name "external/" sandbox))
           (marker (expand-file-name "keep.txt" external)))
      (make-directory external t)
      (with-temp-file marker (insert "keep"))
      (let ((root-entry
             (imoogi-project-notes-mounted-root-add external "Portable")))
        (should-error
         (imoogi-project-notes-mounted-root-add external "Duplicate")
         :type 'user-error)
        (imoogi-project-notes-mounted-root-edit root-entry "Renamed")
        (should (eq (alist-get 'enabled
                               (car (alist-get
                                     'roots
                                     (imoogi-project-notes--read-mounted-state))))
                    t))
        (imoogi-project-notes-mounted-root-edit root-entry "Renamed" nil)
        (should (eq (alist-get 'enabled
                               (car (alist-get
                                     'roots
                                     (imoogi-project-notes--read-mounted-state))))
                    :json-false))
        (imoogi-project-notes-mounted-root-remove root-entry)
        (should-not (alist-get 'roots
                               (imoogi-project-notes--read-mounted-state)))
        (should (file-exists-p marker))))))

(ert-deftest imoogi-project-notes-detach-is-session-only ()
  (imoogi-project-notes-test--isolated
    (let* ((external (file-name-as-directory
                      (expand-file-name "external/" sandbox)))
           (notes (expand-file-name "study/" external)))
      (imoogi-project-notes-test--write-metadata
       notes "study:26.04" "study" nil "26.04")
      (let* ((canonical (imoogi-project-notes--canonical-directory external))
             (state `((version . 1)
                      (roots . (((id . ,(imoogi-project-notes--mounted-root-id
                                         canonical))
                                 (path . ,canonical) (label . "USB") (enabled . t))))
                      (source-overrides . nil))))
        (imoogi-project-notes--write-mounted-state state)
        (let* ((imoogi-project-notes--detached-instance-ids nil)
               (entry (car (imoogi-project-notes--all-entries))))
          (imoogi-project-notes-detach entry)
          (should-not (imoogi-project-notes--all-entries))
          (should (alist-get 'roots
                             (imoogi-project-notes--read-mounted-state))))))))

(ert-deftest imoogi-project-notes-unmount-preflight-failure-has-zero-mutation ()
  (imoogi-project-notes-test--isolated
    (let* ((external (file-name-as-directory
                      (expand-file-name "external/" sandbox)))
           (file (expand-file-name "note.org" external))
           (root-entry '((id . "root") (path . "unused")
                         (label . "USB") (enabled . t)))
           (saved nil) (cleaned nil) (executed nil))
      (make-directory external t)
      (with-temp-file file (insert "original"))
      (let ((buffer (find-file-noselect file)))
        (with-current-buffer buffer
          (goto-char (point-max))
          (insert " changed"))
        (cl-letf (((symbol-function 'imoogi-project-notes--save-buffers-or-error)
                   (lambda (_buffers) (setq saved t)))
                  ((symbol-function 'imoogi-project-notes--cleanup-unmount-buffers)
                   (lambda (_buffers) (setq cleaned t)))
                  (imoogi-project-notes-unmount-preflight-function
                   (lambda (_root) (user-error "unsupported")))
                  (imoogi-project-notes-unmount-executor-function
                   (lambda (_descriptor) (setq executed t))))
          (should-error (imoogi-project-notes-unmount-device root-entry)
                        :type 'user-error)
          (should-not saved)
          (should-not cleaned)
          (should-not executed)
          (should (buffer-live-p buffer)))))))

(ert-deftest imoogi-project-notes-unmount-saves-kills-then-executes ()
  (imoogi-project-notes-test--isolated
    (let* ((external (file-name-as-directory
                      (expand-file-name "external/" sandbox)))
           (inside-file (expand-file-name "note.org" external))
           (outside-file (expand-file-name "outside.org" sandbox))
           (root-entry `((id . "root") (path . ,external)
                         (label . "USB") (enabled . t)))
           events)
      (make-directory external t)
      (with-temp-file inside-file (insert "inside"))
      (with-temp-file outside-file (insert "outside"))
      (let ((inside (find-file-noselect inside-file))
            (outside (find-file-noselect outside-file)))
        (with-current-buffer inside
          (goto-char (point-max))
          (insert " changed"))
        (cl-letf ((imoogi-project-notes-unmount-preflight-function
                   (lambda (_root)
                     (push 'preflight events)
                     (list :supported t :mount-point external
                           :device "/dev/test" :command '("false"))))
                  (imoogi-project-notes-unmount-executor-function
                   (lambda (_descriptor)
                     (push 'execute events)
                     t))
                  ((symbol-function 'y-or-n-p) (lambda (_prompt) t))
                  ((symbol-function 'imoogi-project-notes--show-dirty-buffers)
                   (lambda (_buffers) (push 'shown events)))
                  ((symbol-function 'imoogi-project-notes--save-buffers-or-error)
                   (lambda (buffers)
                     (push 'save events)
                     (dolist (buffer buffers)
                       (with-current-buffer buffer
                         (set-buffer-modified-p nil)))))
                  ((symbol-function 'imoogi-project-notes--cleanup-unmount-buffers)
                   (lambda (buffers)
                     (push 'cleanup events)
                     (dolist (buffer buffers) (kill-buffer buffer)))))
          (should (imoogi-project-notes-unmount-device root-entry))
          (should (equal (nreverse events)
                         '(preflight shown save cleanup execute)))
          (should-not (buffer-live-p inside))
          (should (buffer-live-p outside)))))))

(ert-deftest imoogi-project-notes-unmount-save-failure-stops-cleanup ()
  (imoogi-project-notes-test--isolated
    (let ((root-entry '((id . "root") (path . "/tmp/")
                        (label . "USB") (enabled . t)))
          cleaned executed)
      (cl-letf ((imoogi-project-notes-unmount-preflight-function
                 (lambda (_root)
                   (list :supported t :mount-point sandbox
                         :device "/dev/test" :command '("false"))))
                ((symbol-function 'imoogi-project-notes--buffers-under-directory)
                 (lambda (_directory) (list (current-buffer))))
                ((symbol-function 'buffer-modified-p) (lambda (&optional _buffer) t))
                ((symbol-function 'y-or-n-p) (lambda (_prompt) t))
                ((symbol-function 'imoogi-project-notes--show-dirty-buffers)
                 #'ignore)
                ((symbol-function 'imoogi-project-notes--save-buffers-or-error)
                 (lambda (_buffers) (error "save failed")))
                ((symbol-function 'imoogi-project-notes--cleanup-unmount-buffers)
                 (lambda (_buffers) (setq cleaned t)))
                (imoogi-project-notes-unmount-executor-function
                 (lambda (_descriptor) (setq executed t))))
        (should-error (imoogi-project-notes-unmount-device root-entry))
        (should-not cleaned)
        (should-not executed)))))

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

(ert-deftest imoogi-project-notes-setup-uses-explicit-numbering ()
  (imoogi-project-notes-test--isolated
    (let ((directory (imoogi-project-notes-setup root nil "260925.01")))
      (should (string-suffix-p "/260925.01-source/" directory))
      (should (file-exists-p (expand-file-name "project.org" directory)))))

(ert-deftest imoogi-project-notes-doctor-asks-and-renames-invalid-folder ()
  (imoogi-project-notes-test--isolated
    (let* ((old (expand-file-name "260925-source/"
                                 imoogi-project-notes-directory))
           (new (expand-file-name "260925.01-source/"
                                 imoogi-project-notes-directory)))
      (imoogi-project-notes-test--write-metadata
       old "dir:source" "project" root)
      (dolist (file '("project.org" "tasks.org" "journal.org"))
        (with-temp-file (expand-file-name file old)
          (insert (format "* %s\n" file))))
      (cl-letf (((symbol-function 'read-string)
                 (lambda (&rest _prompt) "260925.01-source"))
                ((symbol-function 'imoogi-project-notes--replace-treemacs-root)
                 #'ignore))
        (let ((result (imoogi-project-notes-setup-doctor)))
          (should (= (plist-get result :changed) 1))
          (should (= (plist-get result :skipped) 0))))
      (should-not (file-exists-p old))
      (should (file-exists-p new))
      (should (file-exists-p (expand-file-name "project.org" new)))))))

(ert-deftest imoogi-project-notes-doctor-rejects-duplicate-scoped-id ()
  (imoogi-project-notes-test--isolated
    (let* ((old (expand-file-name "260925-source/"
                                 imoogi-project-notes-directory))
           (duplicate (expand-file-name "260925.01-existing/"
                                        imoogi-project-notes-directory)))
      (make-directory duplicate t)
      (imoogi-project-notes-test--write-metadata
       old "dir:source" "project" root)
      (cl-letf (((symbol-function 'read-string)
                 (lambda (&rest _prompt) "260925.01-source")))
        (should-error (imoogi-project-notes-setup-doctor)
                      :type 'user-error))
      (should (file-directory-p old))
      (should (file-directory-p duplicate)))))

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

(ert-deftest imoogi-project-notes-prefix-creates-study-under-mounted-root ()
  (imoogi-project-notes-test--isolated
    (let* ((external (file-name-as-directory
                      (expand-file-name "external/" sandbox)))
           (canonical (progn
                        (make-directory external t)
                        (imoogi-project-notes--canonical-directory external)))
           (mounted `((id . ,(imoogi-project-notes--mounted-root-id canonical))
                      (path . ,canonical)
                      (label . "Portable")
                      (enabled . t))))
      (make-directory (expand-file-name "26.03-existing/" external) t)
      (make-directory (expand-file-name "26.07-local/"
                                        imoogi-project-notes-directory) t)
      (imoogi-project-notes--write-mounted-state
       `((version . 1) (roots . (,mounted)) (source-overrides . nil)))
      (let ((current-prefix-arg '(4)) selected-default)
        (cl-letf (((symbol-function 'read-string)
                   (lambda (&rest _) "Cognitive Science"))
                  ((symbol-function 'imoogi-project-notes--select-mounted-root)
                   (lambda (&rest _) mounted))
                  ((symbol-function 'read-directory-name)
                   (lambda (_prompt default &rest _)
                     (setq selected-default default)
                     default))
                  ((symbol-function 'imoogi-project-notes--open-study-workspace)
                   #'ignore))
          (call-interactively #'imoogi-project-notes-setup-study)
          (should (equal selected-default
                         (expand-file-name "26.08-cognitive-science/" external)))
          (should (file-exists-p
                   (expand-file-name "26.08-cognitive-science/study.org"
                                     external)))
          (should (file-exists-p
                   (expand-file-name
                    "26.08-cognitive-science/.imoogi-project.json"
                    external))))))))

(ert-deftest imoogi-project-notes-moves-local-study-to-mounted-root ()
  (imoogi-project-notes-test--isolated
    (let* ((external (expand-file-name "external/" sandbox))
           (_ (make-directory external t))
           (canonical (imoogi-project-notes--canonical-directory external))
           (root-entry
            `((id . ,(imoogi-project-notes--mounted-root-id canonical))
              (path . ,canonical) (label . "Portable") (enabled . t)))
           (directory (imoogi-project-notes-setup-study
                       "Operating Systems" nil "26.01" "2026-01-15"))
           (study-buffer (current-buffer))
           (local-entry (car (imoogi-project-notes--all-entries)))
           (destination
            (expand-file-name
             (file-name-nondirectory (directory-file-name directory))
             external))
           (old-tasks (expand-file-name "tasks.org" directory))
           (new-tasks (expand-file-name "tasks.org" destination)))
      (imoogi-project-notes--write-mounted-state
       `((version . 1) (roots . (,root-entry)) (source-overrides . nil)))
      (with-temp-file (expand-file-name "concepts/C01-process.org" directory)
        (insert "#+TITLE: Process\n\n* Context switch\n"))
      (setq org-agenda-files (list old-tasks))
      (let ((imoogi-project-notes-move-runner-function
             (lambda (source target)
               (unless (equal (file-truename default-directory)
                              (file-truename external))
                 (ert-fail (format "move runner cwd was %s" default-directory)))
               (rename-file source target)
               '((ok . t) (files . 7)))))
        (should (equal (file-name-as-directory destination)
                       (imoogi-project-notes-move-to-mounted-root
                        local-entry root-entry))))
      (should-not (file-exists-p directory))
      (should (file-exists-p
               (expand-file-name "concepts/C01-process.org" destination)))
      (should-not (imoogi-project-notes--read-registry))
      (should (equal org-agenda-files (list new-tasks)))
      (with-current-buffer study-buffer
        (should (equal buffer-file-name
                       (expand-file-name "study.org" destination))))
      (let ((entries (imoogi-project-notes--all-entries)))
        (should (= (length entries) 1))
        (should (eq (alist-get 'origin (car entries)) 'mounted))
        (should (equal (alist-get 'notes-dir (car entries))
                       (file-name-as-directory destination)))))))

(ert-deftest imoogi-project-notes-move-never-overwrites-mounted-directory ()
  (imoogi-project-notes-test--isolated
    (let* ((external (expand-file-name "external/" sandbox))
           (_ (make-directory external t))
           (canonical (imoogi-project-notes--canonical-directory external))
           (root-entry
            `((id . ,(imoogi-project-notes--mounted-root-id canonical))
              (path . ,canonical) (label . "Portable") (enabled . t)))
           (directory (imoogi-project-notes-setup-study
                       "Operating Systems" nil "26.01" "2026-01-15"))
           (local-entry (car (imoogi-project-notes--all-entries)))
           (destination
            (expand-file-name
             (file-name-nondirectory (directory-file-name directory))
             external)))
      (make-directory destination t)
      (with-temp-file (expand-file-name "keep.txt" destination)
        (insert "existing"))
      (should-error
       (imoogi-project-notes-move-to-mounted-root local-entry root-entry)
       :type 'user-error)
      (should (file-directory-p directory))
      (should (equal "existing"
                     (imoogi-project-notes-test--read
                      (expand-file-name "keep.txt" destination))))
      (should (= (length (imoogi-project-notes--read-registry)) 1)))))

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
  (dolist (key '("s" "S" "o" "t" "j" "l" "a" "A" "r" "d" "n" "h"
                 "+" "L" "E" "R" "D" "x" "u" "c" "C" "e"))
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
