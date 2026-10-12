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

;;; project-perspective-pin-test.el ends here
