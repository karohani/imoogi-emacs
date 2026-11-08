;;; perspective-empty-state-test.el --- Empty restore metadata tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)

(ert-deftest imoogi-perspective-restore-empty-metadata-keeps-main-state ()
  "Empty optional files must not turn a valid workspace restore into a failure."
  (let* ((dir (make-temp-file "imoogi-empty-state-" t))
         (imoogi-perspective-state-file (expand-file-name "state.el" dir))
         (imoogi-perspective-active-file (expand-file-name "active.el" dir))
         (imoogi-perspective-treemacs-file (expand-file-name "treemacs.el" dir))
         (imoogi-perspective--treemacs-visible '("stale"))
         loaded warnings)
    (unwind-protect
        (progn
          (with-temp-file imoogi-perspective-state-file (insert "state"))
          (with-temp-file imoogi-perspective-active-file (insert " \n"))
          (with-temp-file imoogi-perspective-treemacs-file)
          (cl-letf (((symbol-function 'persp-state-load)
                     (lambda (file) (setq loaded file)))
                    ((symbol-function 'display-warning)
                     (lambda (&rest warning) (push warning warnings))))
            (imoogi-perspective-state-restore))
          (should (equal loaded imoogi-perspective-state-file))
          (should-not imoogi-perspective--treemacs-visible)
          (should-not warnings))
      (delete-directory dir t))))

(ert-deftest imoogi-perspective-optional-state-formfeed-is-blank ()
  (let ((file (make-temp-file "imoogi-blank-state-")))
    (unwind-protect
        (progn
          (with-temp-file file (insert " \t\r\n\f"))
          (should-not (imoogi-perspective--read-optional-state file)))
      (delete-file file))))

;;; perspective-empty-state-test.el ends here
