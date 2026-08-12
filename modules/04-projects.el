;;; projects.el --- project.el and Perspective workspaces -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "04-projects" 'project 'perspective)

(require 'cl-lib)
(require 'subr-x)

(defvar imoogi-project-perspective-alist nil
  "Canonical project roots mapped to their stable Perspective names.")

(defvar imoogi-perspective-state-file
  (locate-user-emacs-file ".cache/perspective-state.el")
  "File used to persist Perspective buffers and window layouts.")

(defun imoogi-project--canonical-root (root)
  "Return ROOT as a canonical directory name."
  (file-name-as-directory (file-truename (expand-file-name root))))

(defun imoogi-project--path-components (root)
  "Return the non-empty path components in ROOT."
  (split-string (directory-file-name root) "/" t))

(defun imoogi-project--path-suffix (root depth)
  "Return the last DEPTH components of ROOT joined by a slash."
  (let* ((parts (imoogi-project--path-components root))
         (start (max 0 (- (length parts) depth))))
    (string-join (nthcdr start parts) "/")))

(defun imoogi-project--known-roots (root)
  "Return canonical known roots, including ROOT and persisted mappings."
  (delete-dups
   (delq nil
         (mapcar
          (lambda (candidate)
            (when (file-directory-p candidate)
              (imoogi-project--canonical-root candidate)))
          (append (list root)
                  (project-known-project-roots)
                  (mapcar #'car imoogi-project-perspective-alist))))))

(defun imoogi-project--unique-suffix (root roots)
  "Return ROOT's shortest suffix that is unique among ROOTS."
  (let ((others (delete root (copy-sequence roots))))
    (cl-loop for depth from 1 to (length (imoogi-project--path-components root))
             for suffix = (imoogi-project--path-suffix root depth)
             unless (cl-some
                     (lambda (other)
                       (string= suffix (imoogi-project--path-suffix other depth)))
                     others)
             return suffix)))

(defun imoogi-project--name-used-by-other-root-p (name root)
  "Return non-nil when NAME is already assigned to a root other than ROOT."
  (cl-some (lambda (entry)
             (and (not (equal (car entry) root))
                  (equal (cdr entry) name)))
           imoogi-project-perspective-alist))

(defun imoogi-project-perspective-name (root)
  "Return the stable Perspective name assigned to project ROOT.
The first computed mapping is persisted through `savehist' and is never
renamed merely because the known project registry changes."
  (let* ((canonical-root (imoogi-project--canonical-root root))
         (stored (assoc canonical-root imoogi-project-perspective-alist)))
    (or (cdr stored)
        (let* ((roots (imoogi-project--known-roots canonical-root))
               (basename (file-name-nondirectory
                          (directory-file-name canonical-root)))
               (same-basename
                (cl-remove-if-not
                 (lambda (candidate)
                   (equal basename
                          (file-name-nondirectory
                           (directory-file-name candidate))))
                 roots))
               (candidate
                (if (> (length same-basename) 1)
                    (or (imoogi-project--unique-suffix canonical-root roots)
                        basename)
                  basename))
               (name
                (if (imoogi-project--name-used-by-other-root-p
                     candidate canonical-root)
                    (format "%s--%s" candidate
                            (substring (secure-hash 'sha1 canonical-root) 0 8))
                  candidate)))
          (push (cons canonical-root name) imoogi-project-perspective-alist)
          name))))

(defun imoogi-project-switch-perspective (keep-perspective)
  "Select a project and open it in a matching Perspective workspace.
With KEEP-PERSPECTIVE (interactively, a prefix argument), remember and open
the selected project without leaving the current Perspective."
  (interactive "P")
  (let* ((selected-root (funcall project-prompter))
         (project
          (let ((project-prompter (lambda () selected-root)))
            (project-current t selected-root)))
         (root (project-root project)))
    (project-remember-project project)
    (unless keep-perspective
      (persp-switch (imoogi-project-perspective-name root)))
    (let ((default-directory root))
      (project-dired))))

(defun imoogi-perspective-state-restore ()
  "Restore saved Perspective state without making startup fragile."
  (when (file-readable-p imoogi-perspective-state-file)
    (condition-case err
        (persp-state-load imoogi-perspective-state-file)
      (error
       (display-warning
        'imoogi-projects
        (format "Perspective state restore failed: %s"
                (error-message-string err))
        :warning)))
    nil))

(defun imoogi-perspective-state-save ()
  "Save Perspective state on normal Emacs exit."
  (when (bound-and-true-p persp-mode)
    (condition-case err
        (progn
          (make-directory (file-name-directory imoogi-perspective-state-file) t)
          (persp-state-save imoogi-perspective-state-file))
      (error
       (display-warning
        'imoogi-projects
        (format "Perspective state save failed: %s"
                (error-message-string err))
        :warning)))
    nil))

(use-package project
  :ensure nil
  :demand t
  :bind (:map project-prefix-map
              ("p" . imoogi-project-switch-perspective)))

;;; Perspective
(use-package perspective
  :ensure t
  :custom
  (persp-mode-prefix-key (kbd "C-x x"))
  (persp-state-default-file imoogi-perspective-state-file)
  :config
  (persp-mode 1)
  (define-key perspective-map (kbd "l") #'persp-switch-last)
  (add-hook 'emacs-startup-hook #'imoogi-perspective-state-restore -50)
  (add-hook 'kill-emacs-hook #'imoogi-perspective-state-save))

(provide 'imoogi-projects)
;;; projects.el ends here
