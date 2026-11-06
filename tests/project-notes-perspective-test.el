;;; project-notes-perspective-test.el --- Perspective-pinned project notes tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'org)

(defvar imoogi-project-perspective-alist)

(defmacro imoogi-project-notes-perspective-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename (make-temp-file "imoogi-project-notes-persp-" t)))
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
          (org-directory personal)
          (imoogi-project-perspective-alist nil)
          (current-perspective "main"))
     (make-directory root t)
     (unwind-protect
         (cl-letf (((symbol-function 'imoogi-org--default-directory)
                    (lambda () personal))
                   ((symbol-function 'imoogi-project-notes--registry-file)
                    (lambda () registry-file))
                   ((symbol-function 'imoogi-project-notes--mounted-roots-file)
                    (lambda () mounted-roots-file))
                   ((symbol-function 'persp-current-name)
                    (lambda () current-perspective))
                   ;; Reaching the picker is the reported bug.
                   ((symbol-function 'imoogi-project-notes--select-entry)
                    (lambda (&rest _)
                      (error "Project picker was shown"))))
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

(ert-deftest imoogi-project-notes-scratch-resolves-perspective-project ()
  "A non-project buffer resolves to the project its Perspective is pinned to."
  (imoogi-project-notes-perspective-test--isolated
    (let ((directory (imoogi-project-notes-setup root)))
      (push (cons (imoogi-project--canonical-root root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (imoogi-notes-scratch)
      (should (equal buffer-file-name (expand-file-name "scratch.org" personal)))
      (let ((entry (imoogi-project-notes--current-entry)))
        (should entry)
        (should (file-equal-p (alist-get 'notes-dir entry) directory))))))

(ert-deftest imoogi-project-notes-unmapped-perspective-does-not-guess ()
  "Without a Perspective mapping, a non-project buffer still has no entry."
  (imoogi-project-notes-perspective-test--isolated
    (imoogi-project-notes-setup root)
    (setq current-perspective "main")
    (imoogi-notes-scratch)
    (should-not (imoogi-project-notes--current-entry))))

(ert-deftest imoogi-project-notes-buffer-project-beats-perspective ()
  "A buffer that belongs to a project keeps that project in another Perspective."
  (imoogi-project-notes-perspective-test--isolated
    (let* ((other-root (file-name-as-directory
                        (expand-file-name "other/" sandbox)))
           (directory (imoogi-project-notes-setup root))
           (other-directory (progn (make-directory other-root t)
                                   (imoogi-project-notes-setup other-root))))
      (push (cons (imoogi-project--canonical-root root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (find-file (expand-file-name "project.org" other-directory))
      (should (file-equal-p (alist-get 'notes-dir
                                       (imoogi-project-notes--current-entry))
                            other-directory))
      (should-not (file-equal-p other-directory directory)))))

(ert-deftest imoogi-project-notes-unregistered-project-does-not-borrow-perspective ()
  "A buffer of a project without notes is not resolved to the Perspective's."
  (imoogi-project-notes-perspective-test--isolated
    (let ((other-root (file-name-as-directory
                       (expand-file-name "other/" sandbox))))
      (imoogi-project-notes-setup root)
      (make-directory other-root t)
      (push (cons (imoogi-project--canonical-root root) "src")
            imoogi-project-perspective-alist)
      (setq current-perspective "src")
      (with-temp-buffer
        (setq default-directory other-root)
        (cl-letf (((symbol-function 'project-current)
                   (lambda (&rest _) (list 'transient other-root))))
          (should-not (imoogi-project-notes--current-entry)))))))

;;; project-notes-perspective-test.el ends here
