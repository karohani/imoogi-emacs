;;; transient-input-method-test.el --- transient input method tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'transient)
(require 'ace-window)

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

(ert-deftest imoogi-raw-key-reader-bypasses-without-disabling-input-method ()
  (with-temp-buffer
    (imoogi-test--with-korean-input-method
     (lambda ()
       (should
        (eq 'selected
            (imoogi-call-with-raw-key-input
             (lambda ()
               (should (equal current-input-method "korean-hangul"))
               (should-not input-method-function)
               'selected))))
       (should (equal current-input-method "korean-hangul"))))))

(ert-deftest imoogi-raw-key-reader-restores-input-function-after-error ()
  (with-temp-buffer
    (imoogi-test--with-korean-input-method
     (lambda ()
       (let ((original input-method-function))
         (should-error
          (imoogi-call-with-raw-key-input
           (lambda ()
             (should-not input-method-function)
             (error "key reader failed"))))
         (should (eq input-method-function original))
         (should (equal current-input-method "korean-hangul")))))))

(ert-deftest imoogi-avy-read-selects-label-with-korean-input-active ()
  (with-temp-buffer
    (imoogi-test--with-korean-input-method
     (lambda ()
       (let* ((unread-command-events (list ?a))
              (saw-raw-reader nil)
             (avy-translate-char-function
              (lambda (char)
                (should (equal current-input-method "korean-hangul"))
                (should-not input-method-function)
                (setq saw-raw-reader t)
                char)))
         (should
          (eq 'selected
              (avy-read (avy-tree '(selected) '(?a ?s))
                        (lambda (&rest _))
                        (lambda ()))))
         (should saw-raw-reader))
       (should (equal current-input-method "korean-hangul"))))))

(ert-deftest imoogi-ace-family-shares-raw-key-reader-boundary ()
  (dolist (command '(ace-window ace-select-window ace-swap-window
                     ace-delete-window ace-delete-other-windows
                     ace-display-buffer))
    (should (fboundp command)))
  (should (advice-member-p #'imoogi-call-with-raw-key-input 'avy-read)))

(ert-deftest imoogi-raw-key-reader-advice-does-not-accumulate ()
  (let (before after)
    (advice-mapc (lambda (advice _props) (push advice before)) 'avy-read)
    (load (expand-file-name "modules/general/05-transient.el" imoogi-test-root) nil t)
    (advice-mapc (lambda (advice _props) (push advice after)) 'avy-read)
    (should (= (length before) (length after)))
    (should (= 1 (cl-count #'imoogi-call-with-raw-key-input after)))))

(provide 'transient-input-method-test)
;;; transient-input-method-test.el ends here
