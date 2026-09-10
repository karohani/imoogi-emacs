;;; transient-input-method-test.el --- transient input method tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'transient)

(defun imoogi-test--with-korean-input-method (fn)
  "Run FN with the built-in Korean input method active."
  (let ((original current-input-method))
    (unwind-protect
        (progn
          (activate-input-method "korean-hangul")
          (should (equal current-input-method "korean-hangul"))
          (funcall fn))
      (if original
          (activate-input-method original)
        (deactivate-input-method)))))

(ert-deftest imoogi-transient-disables-and-restores-built-in-input-method ()
  (with-temp-buffer
    (switch-to-buffer (current-buffer))
    (imoogi-test--with-korean-input-method
     (lambda ()
       (unwind-protect
           (progn
             (transient-setup 'imoogi-transient-zoom)
             (should-not current-input-method)
             (execute-kbd-macro (kbd "q"))
             (should (equal current-input-method "korean-hangul")))
         (transient--stack-zap)
         (setq imoogi-transient--input-method-state nil))))))

(ert-deftest imoogi-transient-keeps-input-method-disabled-between-submenus ()
  (with-temp-buffer
    (switch-to-buffer (current-buffer))
    (imoogi-test--with-korean-input-method
     (lambda ()
       (unwind-protect
           (progn
             (transient-setup 'imoogi-transient-master)
             (should-not current-input-method)
             (execute-kbd-macro (kbd "w"))
             (should-not current-input-method)
             (execute-kbd-macro (kbd "q"))
             (should-not current-input-method)
             (execute-kbd-macro (kbd "q"))
             (should (equal current-input-method "korean-hangul")))
         (transient--stack-zap)
         (setq imoogi-transient--input-method-state nil))))))

(ert-deftest imoogi-transient-restores-input-method-on-suspend ()
  (with-temp-buffer
    (switch-to-buffer (current-buffer))
    (imoogi-test--with-korean-input-method
     (lambda ()
       (unwind-protect
           (progn
             (transient-setup 'imoogi-transient-zoom)
             (should-not current-input-method)
             (execute-kbd-macro (kbd "C-z"))
             (should (equal current-input-method "korean-hangul")))
         (transient--stack-zap)
         (setq imoogi-transient--input-method-state nil))))))

(ert-deftest imoogi-transient-suspends-input-method-after-buffer-switch ()
  (let ((first (generate-new-buffer " *imoogi transient first*"))
        (second (generate-new-buffer " *imoogi transient second*")))
    (unwind-protect
        (with-current-buffer first
          (switch-to-buffer first)
          (activate-input-method "korean-hangul")
          (transient-setup 'imoogi-transient-zoom)
          (should-not current-input-method)
          (with-current-buffer second
            (activate-input-method "korean-hangul"))
          (switch-to-buffer second)
          (imoogi-transient--suspend-input-method-after-buffer-change)
          (should-not current-input-method)
          (execute-kbd-macro (kbd "q"))
          (should (equal (buffer-local-value 'current-input-method first)
                         "korean-hangul"))
          (should (equal (buffer-local-value 'current-input-method second)
                         "korean-hangul")))
      (transient--stack-zap)
      (setq imoogi-transient--input-method-state nil)
      (mapc (lambda (buffer)
              (when (buffer-live-p buffer)
                (kill-buffer buffer)))
            (list first second)))))

(ert-deftest imoogi-transient-restores-input-method-when-setup-errors ()
  (with-temp-buffer
    (switch-to-buffer (current-buffer))
    (activate-input-method "korean-hangul")
    (cl-letf (((symbol-function 'transient--init-transient)
               (lambda (&rest _)
                 (error "setup failed"))))
      (should-error (transient-setup 'imoogi-transient-zoom)))
    (should (equal current-input-method "korean-hangul"))))

(ert-deftest imoogi-transient-restores-input-method-when-setup-quits ()
  (with-temp-buffer
    (switch-to-buffer (current-buffer))
    (activate-input-method "korean-hangul")
    (cl-letf (((symbol-function 'transient--init-transient)
               (lambda (&rest _)
                 (signal 'quit nil))))
      (let ((caught nil))
        (condition-case nil
            (transient-setup 'imoogi-transient-zoom)
          (quit (setq caught t)))
        (should caught)))
    (should (equal current-input-method "korean-hangul"))))

(provide 'transient-input-method-test)
;;; transient-input-method-test.el ends here
