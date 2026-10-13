;;; project-perspective-pin-test.el --- Perspective-pinned project resolution tests -*- lexical-binding: t; -*-

;; SPEC-PERSPPIN-001: a projectless buffer in a mapped Perspective resolves to
;; the Perspective's project, and the project transient's `b'/`f' candidate
;; lists add the project's own notes.  Every test runs in a sandbox so nothing
;; touches ~/.emacs.d/projects, ~/project-notes, or the savehist file.

(require 'ert)
(require 'cl-lib)
(require 'org)
(require 'project)

(defvar imoogi-project-perspective-alist)
;; Eglot binds this around its lookups; declared here so the `let' below is
;; dynamic under lexical binding even when Eglot is not loaded.
(defvar eglot-lsp-context)
(defvar persp-switch-hook)
(defvar imoogi-project--perspective-finder-memo)
(defvar imoogi-project--perspective-finder-inhibit)
(defvar imoogi-project-notes--context-directories)

(defmacro imoogi-persppin-test--isolated (&rest body)
  "Run BODY in a sandbox with the Perspective finder installed.
BODY runs in a projectless buffer whose `default-directory' is `elsewhere'.
Prompting signals \"prompted\", and `compile' records its directory in
`compiled'."
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename (make-temp-file "imoogi-persppin-" t)))
          (user-emacs-directory (expand-file-name "emacs/" sandbox))
          (project-list-file (expand-file-name "projects" user-emacs-directory))
          (project--list nil)
          (imoogi-project-notes-directory (expand-file-name "project-notes/" sandbox))
          (imoogi-project-notes-todo-storage 'project)
          (personal (expand-file-name "notes/" sandbox))
          (root (file-name-as-directory (expand-file-name "source/" sandbox)))
          (elsewhere (file-name-as-directory (expand-file-name "elsewhere/" sandbox)))
          (registry-file (expand-file-name ".cache/project-notes.json"
                                           user-emacs-directory))
          (mounted-roots-file
           (expand-file-name ".cache/project-notes-mounted-roots.json"
                             user-emacs-directory))
          (org-id-locations-file (expand-file-name "org-id-locations" sandbox))
          (org-agenda-files nil)
          (org-directory personal)
          (imoogi-project-perspective-alist nil)
          (persp-mode t)
          (current-perspective "main")
          (compiled nil)
          (project-find-functions (list #'project-try-vc))
          (imoogi-project--perspective-finder-memo nil)
          (imoogi-project--perspective-finder-inhibit nil)
          (imoogi-project-notes--context-directories 'unbuilt)
          (project-prompter (lambda (&rest _) (error "prompted"))))
     (ignore personal registry-file mounted-roots-file compiled)
     (when (fboundp 'imoogi-project-install-perspective-finder)
       (imoogi-project-install-perspective-finder))
     (dolist (directory (list user-emacs-directory personal root elsewhere))
       (make-directory directory t))
     (unwind-protect
         (cl-letf (((symbol-function 'imoogi-org--default-directory)
                    (lambda () personal))
                   ((symbol-function 'imoogi-project-notes--registry-file)
                    (lambda () registry-file))
                   ((symbol-function 'imoogi-project-notes--mounted-roots-file)
                    (lambda () mounted-roots-file))
                   ((symbol-function 'persp-current-name)
                    (lambda () current-perspective))
                   ((symbol-function 'read-directory-name)
                    (lambda (&rest _) (error "prompted")))
                   ((symbol-function 'imoogi-project-notes--select-entry)
                    (lambda (&rest _) (error "prompted")))
                   ;; `project-compile' uses `call-interactively'.
                   ((symbol-function 'compile)
                    (lambda (&rest _)
                      (interactive)
                      (setq compiled default-directory))))
           (save-window-excursion
             (with-temp-buffer
               (setq default-directory elsewhere)
               ,@body)))
       (dolist (buffer (buffer-list))
         (when (or (and (buffer-file-name buffer)
                        (file-in-directory-p (buffer-file-name buffer) sandbox))
                   (and (with-current-buffer buffer (derived-mode-p 'dired-mode))
                        (file-in-directory-p
                         (buffer-local-value 'default-directory buffer)
                         sandbox)))
           (with-current-buffer buffer
             (set-buffer-modified-p nil))
           (kill-buffer buffer)))
       (delete-directory sandbox t))))

(defun imoogi-persppin-test--canonical (directory)
  "Return DIRECTORY the way the Perspective alist stores it."
  (file-name-as-directory (file-truename (expand-file-name directory))))

(defun imoogi-persppin-test--git-init (directory)
  "Create DIRECTORY and make it a Git repository."
  (make-directory directory t)
  (let ((default-directory (file-name-as-directory directory)))
    (should (zerop (call-process "git" nil nil nil "init" "-q")))))

(defun imoogi-persppin-test--suffix (key)
  "Return the command bound to KEY in `imoogi-transient-project'."
  (plist-get (cdr (transient-get-suffix 'imoogi-transient-project key))
             :command))

(defmacro imoogi-persppin-test--prompted (&rest body)
  "Assert that BODY reaches a stubbed prompt."
  (declare (indent 0) (debug t))
  `(should (equal (cadr (should-error (progn ,@body))) "prompted")))

;;; M1 — the finder and its guards

(ert-deftest imoogi-persppin-scratch-resolves-perspective-project ()
  "AC-PIN-001: a projectless buffer resolves to the mapped non-VC root."
  (imoogi-persppin-test--isolated
    (push (cons (imoogi-persppin-test--canonical root) "src")
          imoogi-project-perspective-alist)
    (setq current-perspective "src")
    (let ((project (project-current nil)))
      (should project)
      (should (equal (project-root project)
                     (imoogi-persppin-test--canonical root)))
      (should (equal (project-current t) project)))))

(ert-deftest imoogi-persppin-vc-root-resolves-vc-project ()
  "AC-PIN-002: a VC root resolves to the same VC project a file there gets."
  (skip-unless (executable-find "git"))
  (imoogi-persppin-test--isolated
    (imoogi-persppin-test--git-init root)
    (push (cons (imoogi-persppin-test--canonical root) "src")
          imoogi-project-perspective-alist)
    (setq current-perspective "src")
    (let ((project (project-current nil))
          (expected (let ((imoogi-project--perspective-finder-inhibit t))
                      (project-current nil (imoogi-persppin-test--canonical root)))))
      (should (eq (car-safe expected) 'vc))
      (should (eq (car-safe project) 'vc))
      (should (equal project expected)))))

(ert-deftest imoogi-persppin-finder-is-last-member ()
  "AC-PIN-003: the finder sorts after every member added at depth 90 or less."
  (imoogi-persppin-test--isolated
    (let* ((other (file-name-as-directory (expand-file-name "other/" sandbox)))
           (claims (lambda (mine)
                     (lambda (dir)
                       (when (file-equal-p dir mine)
                         (cons 'transient mine)))))
           (f0 (funcall claims (expand-file-name "f0/" sandbox)))
           (f1 (funcall claims (expand-file-name "f1/" sandbox)))
           (f2 (funcall claims other))
           (project-find-functions (list #'project-try-vc)))
      (make-directory other t)
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (add-hook 'project-find-functions f0 t)
      (imoogi-project-install-perspective-finder)
      (add-hook 'project-find-functions f1)
      (add-hook 'project-find-functions f2 t)
      (should (eq (car (last project-find-functions))
                  #'imoogi-project-perspective-find))
      (dolist (member (list #'project-try-vc f0 f1 f2))
        (should (memq member (butlast project-find-functions))))
      (with-temp-buffer
        (setq default-directory other)
        (should (equal (project-current nil) (cons 'transient other)))))))

(ert-deftest imoogi-persppin-own-vc-project-beats-perspective ()
  "AC-PIN-004: a buffer of another, unregistered VC project keeps it."
  (skip-unless (executable-find "git"))
  (imoogi-persppin-test--isolated
    (let ((other (file-name-as-directory (expand-file-name "other/" sandbox))))
      (imoogi-persppin-test--git-init other)
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (with-temp-buffer
        (setq default-directory other)
        (should (file-equal-p (project-root (project-current nil)) other))
        (call-interactively (imoogi-persppin-test--suffix "c"))
        (should (file-equal-p compiled other))))))

(ert-deftest imoogi-persppin-unmapped-off-or-stale-declines ()
  "AC-PIN-005: unmapped, `persp-mode' off, or a deleted root behaves as before."
  (imoogi-persppin-test--isolated
    (push (cons (imoogi-persppin-test--canonical root) "src")
          imoogi-project-perspective-alist)
    ;; (a) the current Perspective has no mapping.
    (setq current-perspective "main")
    (should-not (project-current nil))
    (imoogi-persppin-test--prompted (project-current t))
    ;; (b) `persp-mode' is off; `persp-current-name' would signal.
    (setq current-perspective "src")
    (let ((persp-mode nil))
      (cl-letf (((symbol-function 'persp-current-name)
                 (lambda () (error "persp-mode-guard"))))
        (should-not (project-current nil))
        (imoogi-persppin-test--prompted (project-current t))))
    ;; (c) the mapped root was deleted.
    (let ((gone (file-name-as-directory (expand-file-name "gone/" sandbox))))
      (make-directory gone t)
      (push (cons (imoogi-persppin-test--canonical gone) "gone")
            imoogi-project-perspective-alist)
      (delete-directory gone t)
      (setq current-perspective "gone")
      (should-not (project-current nil))
      (imoogi-persppin-test--prompted (project-current t)))))

(ert-deftest imoogi-persppin-explicit-lookups-not-claimed ()
  "AC-PIN-006: explicit directories and the project switch resolve on their own."
  (imoogi-persppin-test--isolated
    (let ((explicit (file-name-as-directory (expand-file-name "explicit/" sandbox)))
          (switched nil)
          (remembered nil))
      (make-directory explicit t)
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      ;; (a) a directory passed by code.
      (should-not (project-current nil explicit))
      ;; (b) `project-current-directory-override'.
      (let ((project-current-directory-override explicit))
        (should-not (project-current nil)))
      ;; (c) the switch command selects the buffer's own directory.
      (let ((project-prompter (lambda () elsewhere)))
        (cl-letf (((symbol-function 'persp-switch)
                   (lambda (name) (setq switched name)))
                  ((symbol-function 'project-remember-project)
                   (lambda (project &rest _) (setq remembered project)))
                  ((symbol-function 'project-dired) #'ignore)
                  ((symbol-function 'imoogi-treemacs-open-project-workspace)
                   #'ignore))
          (imoogi-project-switch-perspective nil)))
      (should (file-equal-p (project-root remembered) elsewhere))
      (should (equal switched
                     (cdr (assoc (imoogi-persppin-test--canonical elsewhere)
                                 imoogi-project-perspective-alist))))
      (should-not (equal switched "src")))))

(ert-deftest imoogi-persppin-eglot-context-declines ()
  "AC-PIN-007: lookups made for Eglot are never claimed."
  (imoogi-persppin-test--isolated
    (push (cons (imoogi-persppin-test--canonical root) "src")
          imoogi-project-perspective-alist)
    (setq current-perspective "src")
    (let ((eglot-lsp-context t))
      (should-not (project-current nil)))
    (let ((eglot-lsp-context nil))
      (should (equal (project-root (project-current nil))
                     (imoogi-persppin-test--canonical root))))))

(ert-deftest imoogi-persppin-remote-root-no-file-io ()
  "AC-PIN-008: remote roots and remote lookup directories cause no file I/O."
  (imoogi-persppin-test--isolated
    (let* ((io nil)
           (guard (lambda (original)
                    (lambda (file &rest args)
                      (if (and (stringp file) (file-remote-p file))
                          (progn (push file io) (error "Remote file I/O: %s" file))
                        (apply original file args)))))
           (project-find-functions nil))
      (imoogi-project-install-perspective-finder)
      (cl-letf (((symbol-function 'file-directory-p)
                 (funcall guard (symbol-function 'file-directory-p)))
                ((symbol-function 'file-truename)
                 (funcall guard (symbol-function 'file-truename)))
                ((symbol-function 'file-exists-p)
                 (funcall guard (symbol-function 'file-exists-p))))
        ;; (a) the Perspective is mapped to a TRAMP root.
        (let ((imoogi-project-perspective-alist
               '(("/ssh:host:/srv/app/" . "remote"))))
          (setq current-perspective "remote")
          (should-not (project-current nil)))
        ;; (b) a TRAMP buffer inside a local Perspective.
        (push (cons (imoogi-persppin-test--canonical root) "src")
              imoogi-project-perspective-alist)
        (setq current-perspective "src")
        (with-temp-buffer
          (setq default-directory "/ssh:host:/tmp/")
          (should-not (project-current nil))))
      (should-not io))))

(ert-deftest imoogi-persppin-finder-never-signals ()
  "AC-PIN-009: an error inside the finder declines instead of escaping."
  (imoogi-persppin-test--isolated
    (let ((calls 0))
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (cl-letf (((symbol-function 'persp-current-name)
                 (lambda ()
                   (setq calls (1+ calls))
                   (error "Perspective lookup failed"))))
        (should-not (project-current nil))
        (should (stringp (imoogi-tab-name))))
      (should (>= calls 1)))))

(ert-deftest imoogi-persppin-finder-memo ()
  "AC-PIN-018: the resolved project is memoized per pinned root."
  (imoogi-persppin-test--isolated
    (let* ((searches 0)
           (second (file-name-as-directory (expand-file-name "second/" sandbox)))
           (original (symbol-function 'project-try-vc)))
      (make-directory second t)
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (cl-letf (((symbol-function 'project-try-vc)
                 (lambda (dir)
                   (when (file-equal-p dir root)
                     (setq searches (1+ searches)))
                   (funcall original dir))))
        (let ((first-lookup (project-current nil))
              (second-lookup (project-current nil)))
          (should first-lookup)
          (should (equal first-lookup second-lookup))
          (should (file-equal-p (project-root first-lookup) root))
          (should (= searches 1)))
        (should (memq #'imoogi-project--perspective-finder-memo-clear
                      (default-value 'persp-switch-hook)))
        (imoogi-project--perspective-finder-memo-clear)
        (should (file-equal-p (project-root (project-current nil)) root))
        (should (= searches 2))
        (setq imoogi-project-perspective-alist
              (list (cons (imoogi-persppin-test--canonical second) "src")))
        (should (file-equal-p (project-root (project-current nil)) second))))))

;;; M2 — notes-context exclusion and t34 compatibility

(defun imoogi-persppin-test--visit (file &optional contents)
  "Create FILE with CONTENTS when missing and return a buffer visiting it."
  (make-directory (file-name-directory file) t)
  (unless (file-exists-p file)
    (with-temp-file file (insert (or contents ""))))
  (find-file-noselect file))

(defun imoogi-persppin-test--write-metadata (directory key study-id)
  "Write minimal study metadata for KEY and STUDY-ID in DIRECTORY."
  (make-directory directory t)
  (with-temp-file (expand-file-name ".imoogi-project.json" directory)
    (insert (json-encode
             `((schema_version . 1) (type . "study") (key . ,key) (name . ,key)
               (study_id . ,study-id)
               (overview . "study.org") (tasks . "tasks.org")
               (journal . "journal.org") (todo_storage . "project")))
            "\n")))

(defmacro imoogi-persppin-test--recording (functions log &rest body)
  "Run BODY with each of FUNCTIONS stubbed to push its name on LOG and signal."
  (declare (indent 2) (debug t))
  `(cl-letf ,(mapcar (lambda (function)
                       `((symbol-function ',function)
                         (lambda (&rest _)
                           (push ',function ,log)
                           (error "Hot path called %s" ',function))))
                     functions)
     ,@body))

(ert-deftest imoogi-persppin-notes-buffer-declined ()
  "AC-PIN-010: a notes buffer keeps the resolution it had before the finder."
  (imoogi-persppin-test--isolated
    (let ((directory (imoogi-project-notes-setup root)))
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (with-current-buffer
          (imoogi-persppin-test--visit (expand-file-name "project.org" directory))
        (let ((entry (imoogi-project-notes--current-entry)))
          (should entry)
          (should-not (project-current nil))
          (should (equal (imoogi-project-notes--current-source-root entry)
                         (let ((project-find-functions
                                (remq #'imoogi-project-perspective-find
                                      project-find-functions)))
                           (imoogi-project-notes--current-source-root entry)))))))))

(ert-deftest imoogi-persppin-notes-check-reads-no-registry ()
  "AC-PIN-010: the notes-context check reads no registry or metadata."
  (imoogi-persppin-test--isolated
    (let ((directory (imoogi-project-notes-setup root))
          (log nil))
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (imoogi-project-notes--rebuild-context-directories)
      (let ((notes (imoogi-persppin-test--visit
                    (expand-file-name "project.org" directory)))
            (plain (imoogi-persppin-test--visit
                    (expand-file-name "plain.txt" elsewhere))))
        (imoogi-persppin-test--recording
            (imoogi-project-notes--all-entries
             imoogi-project-notes--read-registry
             imoogi-project-notes--find-metadata-file)
            log
          (with-current-buffer notes
            (should-not (project-current nil)))
          (with-current-buffer plain
            (should (equal (project-root (project-current nil))
                           (imoogi-persppin-test--canonical root))))))
      (should-not log))))

(ert-deftest imoogi-persppin-notes-list-refresh-triggers ()
  "AC-PIN-010: each plan D-7 trigger rebuilds the notes-directory list."
  (imoogi-persppin-test--isolated
    (let* ((rebuilds 0)
           (log nil)
           (original (symbol-function
                      'imoogi-project-notes--rebuild-context-directories))
           (second-root (file-name-as-directory
                         (expand-file-name "second-source/" sandbox)))
           ;; Outside `imoogi-project-notes-directory', so only the rebuilt
           ;; list (not the pre-build prefix fallback) can recognise it.
           (second-notes (file-name-as-directory
                          (expand-file-name "loose-notes/second/" sandbox)))
           (drive (file-name-as-directory (expand-file-name "drive/" sandbox)))
           (mounted-notes (expand-file-name "study/" drive))
           (imoogi-project-notes--detached-instance-ids nil)
           (claimed-p (lambda (buffer)
                        (with-current-buffer buffer
                          (project-current nil))))
           (rebuilt-p (lambda (trigger)
                        (let ((before rebuilds))
                          (funcall trigger)
                          (> rebuilds before)))))
      (make-directory second-root t)
      (make-directory drive t)
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (imoogi-project-notes-setup second-root second-notes)
      (cl-letf (((symbol-function
                  'imoogi-project-notes--rebuild-context-directories)
                 (lambda (&rest args)
                   (setq rebuilds (1+ rebuilds))
                   (apply original args)))
                ((symbol-function 'message) #'ignore))
        ;; Row 1: a registry write.  Forget N2 first so the row is not vacuous.
        (setq imoogi-project-notes--context-directories nil)
        (let ((probe (imoogi-persppin-test--visit
                      (expand-file-name "probe.txt" second-notes))))
          (should (funcall claimed-p probe))
          (should (funcall rebuilt-p
                           (lambda ()
                             (imoogi-project-notes--write-registry
                              (imoogi-project-notes--read-registry)))))
          (imoogi-persppin-test--recording (imoogi-project-notes--read-registry) log
            (should-not (funcall claimed-p probe))))
        ;; Row 2: a mounted-state write.
        (let ((canonical (imoogi-project-notes--canonical-directory drive)))
          (should (funcall rebuilt-p
                           (lambda ()
                             (imoogi-project-notes--write-mounted-state
                              `((version . 1)
                                (roots . (((id . ,(imoogi-project-notes--mounted-root-id
                                                   canonical))
                                           (path . ,canonical) (label . "Drive")
                                           (enabled . t))))
                                (source-overrides . nil)))))))
        ;; Row 3: an explicit rescan of mounted roots.
        (should (funcall rebuilt-p #'imoogi-project-notes-mounted-root-refresh))
        ;; Row 4: a session detach (memory only).
        (should (funcall rebuilt-p
                         (lambda ()
                           (imoogi-project-notes-detach
                            '((origin . mounted)
                              (instance-id . "persppin-not-a-real-instance"))))))
        ;; Row 5: an unmount.
        (let ((imoogi-project-notes-unmount-preflight-function
               (lambda (_root) (list :supported t :mount-point drive)))
              (imoogi-project-notes-unmount-executor-function (lambda (_) t)))
          (cl-letf (((symbol-function 'y-or-n-p) (lambda (&rest _) t)))
            (should (funcall rebuilt-p
                             (lambda ()
                               (imoogi-project-notes-unmount-device
                                `((path . ,drive) (id . "drive")
                                  (label . "Drive"))))))))
        ;; Row 6: a Perspective switch picks up notes created outside Emacs.
        (imoogi-persppin-test--write-metadata mounted-notes "study:26.99" "26.99")
        (let ((probe (imoogi-persppin-test--visit
                      (expand-file-name "probe.txt" mounted-notes))))
          (should (funcall claimed-p probe))
          (should (funcall rebuilt-p
                           #'imoogi-project-notes--refresh-context-directories))
          (imoogi-persppin-test--recording (imoogi-project-notes--read-registry) log
            (should-not (funcall claimed-p probe)))))
      (should (memq #'imoogi-project-notes--refresh-context-directories
                    (default-value 'persp-switch-hook)))
      (should-not log))))

(ert-deftest imoogi-persppin-notes-list-startup-registered ()
  "AC-PIN-010: the startup rebuild runs after the Perspective state restore.
It is registered on `window-setup-hook', which Emacs runs after
`emacs-startup-hook' (where the restore lives).  `emacs-startup-hook' must stay
exactly the restore function: workspace-bridge-test.el pins that, and the
batch runner never runs either hook, so registration is what is verified."
  (should (memq #'imoogi-perspective-state-restore
                (default-value 'emacs-startup-hook)))
  (should-not (memq #'imoogi-project-notes--rebuild-context-directories
                    (default-value 'emacs-startup-hook)))
  (should (memq #'imoogi-project-notes--rebuild-context-directories
                (default-value 'window-setup-hook))))

(ert-deftest imoogi-persppin-t34-fallback-retained ()
  "AC-PIN-016: with the finder inhibited, the t34 fallback still answers."
  (imoogi-persppin-test--isolated
    (let ((directory (imoogi-project-notes-setup root)))
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (imoogi-notes-scratch)
      (let ((imoogi-project--perspective-finder-inhibit t))
        (should-not (project-current nil))
        (let ((entry (imoogi-project-notes--current-entry)))
          (should entry)
          (should (file-equal-p (alist-get 'notes-dir entry) directory)))))))

;;; M3 — the `b' command and its candidate set

(defmacro imoogi-persppin-test--capturing-buffers (candidates &rest body)
  "Run BODY with `read-buffer' recording its offered buffers in CANDIDATES.
The stub answers with the first offered buffer."
  (declare (indent 1) (debug t))
  `(cl-letf (((symbol-function 'read-buffer)
              (lambda (_prompt &optional _default _require-match predicate)
                (setq ,candidates
                      (cl-remove-if-not
                       (lambda (buffer)
                         (funcall predicate (cons (buffer-name buffer) buffer)))
                       (buffer-list)))
                (buffer-name (car ,candidates)))))
     ,@body))

(defun imoogi-persppin-test--kill-sandbox-buffers (sandbox)
  "Kill every buffer visiting a file below SANDBOX."
  (dolist (buffer (buffer-list))
    (when (and (buffer-file-name buffer)
               (file-in-directory-p (buffer-file-name buffer) sandbox))
      (with-current-buffer buffer (set-buffer-modified-p nil))
      (kill-buffer buffer))))

(defun imoogi-persppin-test--same-set-p (left right)
  "Return non-nil when lists LEFT and RIGHT hold the same elements."
  (null (cl-set-exclusive-or left right)))

(ert-deftest imoogi-persppin-transient-b-c-k-without-prompt ()
  "AC-PIN-011: `b', `c' and `k' act on the Perspective's project from scratch."
  (imoogi-persppin-test--isolated
    (let ((p1 (imoogi-persppin-test--visit (expand-file-name "a.txt" root)))
          (prompts nil)
          (candidates nil))
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (call-interactively (imoogi-persppin-test--suffix "c"))
      (should (equal compiled (imoogi-persppin-test--canonical root)))
      (cl-letf (((symbol-function 'yes-or-no-p)
                 (lambda (prompt) (push prompt prompts) nil)))
        (call-interactively (imoogi-persppin-test--suffix "k")))
      (should (= (length prompts) 1))
      (should (string-match-p (regexp-quote (project-name (project-current nil)))
                              (car prompts)))
      (should (buffer-live-p p1))
      (imoogi-persppin-test--capturing-buffers candidates
        (should (eq (call-interactively (imoogi-persppin-test--suffix "b")) p1)))
      (should (equal candidates (list p1))))))

(ert-deftest imoogi-persppin-switch-buffer-candidates ()
  "AC-PIN-012: `b' offers the project's buffers plus its own notes buffers."
  (imoogi-persppin-test--isolated
    (let* ((other-root (file-name-as-directory
                        (expand-file-name "other-source/" sandbox)))
           (directory (imoogi-project-notes-setup root))
           (other-directory (progn (make-directory other-root t)
                                   (imoogi-project-notes-setup other-root)))
           (candidates nil))
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (imoogi-persppin-test--kill-sandbox-buffers sandbox)
      (let ((p1 (imoogi-persppin-test--visit (expand-file-name "a.txt" root)))
            (tree (dired-noselect root))
            (x (imoogi-persppin-test--visit
                (expand-file-name "unrelated/x.txt" sandbox)))
            (n1 (imoogi-persppin-test--visit
                 (expand-file-name "project.org" directory)))
            (m1 (imoogi-persppin-test--visit
                 (expand-file-name "project.org" other-directory))))
        (imoogi-persppin-test--capturing-buffers candidates
          (call-interactively (imoogi-persppin-test--suffix "b")))
        (should (imoogi-persppin-test--same-set-p candidates (list p1 tree n1)))
        (dolist (excluded (list x m1 (get-buffer "*scratch*")))
          (should-not (memq excluded candidates)))))))

(ert-deftest imoogi-persppin-no-notes-entry-project-only ()
  "AC-PIN-017: without a notes entry, or for a remote root, lists are project-only."
  (imoogi-persppin-test--isolated
    (let ((p1 (imoogi-persppin-test--visit (expand-file-name "a.txt" root)))
          (messages nil)
          (candidates nil))
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (cl-letf (((symbol-function 'message)
                 (lambda (format-string &rest args)
                   (when format-string
                     (push (apply #'format-message format-string args) messages))
                   nil)))
        ;; `b' with no notes entry.
        (imoogi-persppin-test--capturing-buffers candidates
          (call-interactively (imoogi-persppin-test--suffix "b")))
        (should (equal candidates (list p1)))
        ;; `f' with no notes entry.
        (with-temp-buffer
          (setq default-directory elsewhere)
          (should (equal (imoogi-persppin-test--find-file-candidates
                          root (expand-file-name "a.txt" root))
                         (list (expand-file-name "a.txt" root)))))
        ;; `b' and `f' when the identity key refuses the root (as for a
        ;; remote root).
        (cl-letf (((symbol-function 'imoogi-project-notes--identity-key)
                   (lambda (&rest _) (user-error "Remote project"))))
          (setq candidates nil)
          (imoogi-persppin-test--capturing-buffers candidates
            (call-interactively (imoogi-persppin-test--suffix "b")))
          (should (equal candidates (list p1)))
          (with-temp-buffer
            (setq default-directory elsewhere)
            (should (equal (imoogi-persppin-test--find-file-candidates
                            root (expand-file-name "a.txt" root))
                           (list (expand-file-name "a.txt" root)))))))
      (should-not messages))))

;;; M4 — `f' adds the notes files; route (a) and the context-call clause

(defun imoogi-persppin-test--find-file-candidates (root answer)
  "Run the project transient's `f' and return its candidates.
Candidates are expanded against ROOT, sorted; the reader answers ANSWER."
  (let ((recorded nil))
    (let ((project-read-file-name-function
           (lambda (_prompt all-files &rest _)
             (setq recorded all-files)
             answer)))
      (call-interactively (imoogi-persppin-test--suffix "f")))
    (sort (mapcar (lambda (file) (expand-file-name file root)) recorded)
          #'string<)))

(defun imoogi-persppin-test--regular-files (directory)
  "Return the regular files below DIRECTORY, VCS metadata excluded."
  (cl-remove-if (lambda (file) (string-match-p "/\\.git/" file))
                (directory-files-recursively directory "" nil)))

(ert-deftest imoogi-persppin-find-file-candidates ()
  "AC-PIN-013: `f' offers the project's files plus its own notes files."
  (imoogi-persppin-test--isolated
    (let* ((other-root (file-name-as-directory
                        (expand-file-name "other-source/" sandbox)))
           (directory (file-name-as-directory (imoogi-project-notes-setup root)))
           (other-directory (progn (make-directory other-root t)
                                   (imoogi-project-notes-setup other-root)))
           (answer (expand-file-name "project.org" directory)))
      (dolist (file '("a.txt" "sub/b.txt"))
        (imoogi-persppin-test--visit (expand-file-name file root)))
      (imoogi-persppin-test--visit (expand-file-name "tasks.org" directory))
      (imoogi-persppin-test--visit (expand-file-name ".git/config" directory))
      (imoogi-persppin-test--kill-sandbox-buffers sandbox)
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (push (cons (imoogi-persppin-test--canonical other-root) "s")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (let ((files (imoogi-persppin-test--find-file-candidates root answer)))
        (dolist (file (list (expand-file-name "a.txt" root)
                            (expand-file-name "sub/b.txt" root)
                            answer
                            (expand-file-name "tasks.org" directory)))
          (should (member file files)))
        (should (equal files
                       (sort (append
                              (list (expand-file-name "a.txt" root)
                                    (expand-file-name "sub/b.txt" root))
                              (imoogi-persppin-test--regular-files directory))
                             #'string<)))
        (should-not (cl-some (lambda (file) (string-match-p "/\\.git/" file))
                             files))
        (should-not (cl-some (lambda (file) (file-in-directory-p file other-directory))
                             files)))
      (should (file-equal-p buffer-file-name answer))
      (with-temp-buffer
        (setq default-directory elsewhere)
        (setq current-perspective "s")
        (let ((files (imoogi-persppin-test--find-file-candidates
                      other-root
                      (expand-file-name "project.org" other-directory))))
          (should files)
          (should-not (cl-some (lambda (file) (file-in-directory-p file directory))
                               files)))))))

(ert-deftest imoogi-persppin-incidental-prompt-removals ()
  "AC-PIN-015: `f'/`s'/`d' stop prompting whenever a context root exists."
  (imoogi-persppin-test--isolated
    (let ((directory (imoogi-project-notes-setup root))
          (dired-at nil)
          (searched nil)
          (searched-project nil))
      (imoogi-persppin-test--visit (expand-file-name "a.txt" root))
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (cl-letf (((symbol-function 'dired)
                 (lambda (dir &rest _) (setq dired-at dir))))
        ;; `d' from a projectless buffer.
        (imoogi-project-dired)
        (should (file-equal-p dired-at root))
        ;; `d', `s' and `f' from a notes-context buffer of R's notes.
        (setq dired-at nil)
        (with-current-buffer
            (imoogi-persppin-test--visit (expand-file-name "project.org" directory))
          (should (imoogi-project-notes--context-buffer-p))
          (should-not (project-current nil))
          (imoogi-project-dired)
          (should (file-equal-p dired-at root))
          (cl-letf (((symbol-function 'project--read-regexp) (lambda () "needle"))
                    ((symbol-function 'project-files)
                     (lambda (project &rest _)
                       (setq searched-project project)
                       (list (expand-file-name "a.txt" root))))
                    ((symbol-function 'xref-show-xrefs)
                     (lambda (&rest _) (setq searched t))))
            (imoogi-project-find-regexp))
          (should searched)
          (should (file-equal-p (project-root searched-project) root))
          (should (member (expand-file-name "a.txt" root)
                          (imoogi-persppin-test--find-file-candidates
                           root (expand-file-name "a.txt" root)))))
        ;; No context root: the unmapped Perspective still prompts.
        (with-temp-buffer
          (setq default-directory elsewhere)
          (setq current-perspective "main")
          (imoogi-persppin-test--prompted (imoogi-project-find-file))
          (imoogi-persppin-test--prompted (imoogi-project-dired))))
      ;; Project-notes setup from scratch uses the mapped root.
      (with-temp-buffer
        (setq default-directory elsewhere)
        (setq current-perspective "src")
        (should (file-equal-p (imoogi-project-notes--project-root) root))))))

(ert-deftest imoogi-persppin-find-file-remote-context-root-prompts ()
  "O3-d: `f' builds no transient project for a remote context root.
Like `s' and `d', it falls through to the prompt, with no file I/O on the
remote name (REQ-PIN-006)."
  (imoogi-persppin-test--isolated
    (let* ((io nil)
           (guard (lambda (original)
                    (lambda (file &rest args)
                      (if (and (stringp file) (file-remote-p file))
                          (progn (push file io) (error "Remote file I/O: %s" file))
                        (apply original file args)))))
           (project-find-functions nil))
      (imoogi-project-install-perspective-finder)
      (cl-letf (((symbol-function 'imoogi-project-context-root)
                 (lambda () "/ssh:host:/srv/app/"))
                ((symbol-function 'project-find-file-in)
                 (lambda (&rest _) (error "Remote project was searched")))
                ((symbol-function 'file-directory-p)
                 (funcall guard (symbol-function 'file-directory-p)))
                ((symbol-function 'file-truename)
                 (funcall guard (symbol-function 'file-truename)))
                ((symbol-function 'file-exists-p)
                 (funcall guard (symbol-function 'file-exists-p))))
        (imoogi-persppin-test--prompted (imoogi-project-find-file)))
      (should-not io))))

;;; M5 — non-membership and unchanged entries

(ert-deftest imoogi-persppin-notes-not-project-members ()
  "AC-PIN-014: notes stay out of the project outside the `b'/`f' lists."
  (imoogi-persppin-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (p1 (imoogi-persppin-test--visit (expand-file-name "a.txt" root)))
           (n1 (imoogi-persppin-test--visit
                (expand-file-name "project.org" directory))))
      (push (cons (imoogi-persppin-test--canonical root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (let ((project (project-current nil)))
        (should (file-equal-p (project-root project) root))
        (should-not (cl-some (lambda (file)
                               (file-in-directory-p (expand-file-name file root)
                                                    directory))
                             (project-files project)))
        (should (memq p1 (project-buffers project)))
        (should-not (memq n1 (project-buffers project)))
        (project-kill-buffers t)
        (should-not (buffer-live-p p1))
        (should (buffer-live-p n1))))))

(ert-deftest imoogi-persppin-transient-other-entries-unchanged ()
  "AC-PIN-015: only `b' changes among the project transient's entries."
  (dolist (row '(("p" . imoogi-project-switch-perspective)
                 ("f" . imoogi-project-find-file)
                 ("s" . imoogi-project-find-regexp)
                 ("d" . imoogi-project-dired)
                 ("b" . imoogi-project-switch-to-buffer)
                 ("m" . imoogi-project-notes-transient)
                 ("a" . imoogi-persp-new)
                 ("l" . persp-switch-last)
                 ("o" . persp-switch)
                 ("n" . persp-next)
                 ("N" . persp-prev)
                 ("#" . persp-switch-by-number)
                 ("r" . persp-rename)
                 ("K" . persp-kill)
                 ("c" . project-compile)
                 ("k" . project-kill-buffers)
                 ("F" . project-forget-project)
                 ("q" . transient-quit-one)))
    (should (eq (imoogi-persppin-test--suffix (car row)) (cdr row))))
  (imoogi-persppin-test--isolated
    (push (cons (imoogi-persppin-test--canonical root) "src")
          imoogi-project-perspective-alist)
    (setq current-perspective "src")
    (imoogi-persppin-test--prompted
      (call-interactively (imoogi-persppin-test--suffix "F")))))

;;; project-perspective-pin-test.el ends here
