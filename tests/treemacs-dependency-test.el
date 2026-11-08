;;; treemacs-dependency-test.el --- Reload dependency recovery -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'treemacs)

(ert-deftest imoogi-treemacs-reload-replaces-missing-python ()
  (let ((treemacs-python-executable "/missing/imoogi-python3"))
    (load (expand-file-name "modules/project/07-treemacs.el" imoogi-emacs-dir) nil t)
    (should (equal treemacs-python-executable (executable-find "python3")))))

(ert-deftest imoogi-treemacs-reload-without-python-disables-python-features ()
  (unwind-protect
      (let ((treemacs-python-executable "/missing/imoogi-python3")
            (find-executable (symbol-function 'executable-find)))
        (cl-letf (((symbol-function 'executable-find)
                   (lambda (program &rest args)
                     (unless (member program '("python3" "/missing/imoogi-python3"))
                       (apply find-executable program args)))))
          (load (expand-file-name "modules/project/07-treemacs.el" imoogi-emacs-dir) nil t)
          (should-not treemacs-python-executable)
          (should (= treemacs-collapse-dirs 0))
          (should-not treemacs-git-commit-diff-mode)
          (should (eq treemacs--git-mode 'simple))))
    (load (expand-file-name "modules/project/07-treemacs.el" imoogi-emacs-dir) nil t)))

(ert-deftest imoogi-treemacs-reload-without-git-disables-existing-git-mode ()
  (unwind-protect
      (let ((treemacs-python-executable "/missing/imoogi-python3")
            (treemacs-git-mode t)
            (treemacs--git-mode 'deferred))
        (cl-letf (((symbol-function 'executable-find) (lambda (&rest _) nil)))
          (load (expand-file-name "modules/project/07-treemacs.el" imoogi-emacs-dir) nil t)
          (should-not treemacs-python-executable)
          (should-not treemacs-git-mode)
          (should-not treemacs--git-mode)))
    (load (expand-file-name "modules/project/07-treemacs.el" imoogi-emacs-dir) nil t)))

;;; treemacs-dependency-test.el ends here
