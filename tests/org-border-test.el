;;; org-border-test.el --- Org border display regression tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'org)

(add-to-list 'load-path
             (expand-file-name "../modules/org/"
                               (file-name-directory (or load-file-name
                                                        buffer-file-name))))

(unless (featurep 'imoogi-org-border)
  (load (expand-file-name "../modules/org/imoogi-org-border.el"
                          (file-name-directory (or load-file-name buffer-file-name)))))

(defmacro imoogi-org-border-test--with-buffer (contents &rest body)
  "Create a temporary Org buffer with CONTENTS and run BODY."
  (declare (indent 1))
  `(let ((org-mode-hook nil)
         (org-startup-folded nil)
         (org-inhibit-startup t)
         (imoogi-org-border-enabled nil)
         (imoogi-org-border-idle-delay 30))
     (save-window-excursion
       (with-temp-buffer
         (rename-buffer (generate-new-buffer-name "*imoogi-org-border-test*"))
         (switch-to-buffer (current-buffer))
         (insert ,contents)
         (org-mode)
         (setq-local truncate-lines t)
         ,@body))))

(defmacro imoogi-org-border-test--with-window-end (&rest body)
  "Run BODY with a stable batch-friendly `window-end'."
  (declare (indent 0))
  `(cl-letf (((symbol-function 'window-end)
              (lambda (&optional _window _update) (point-max))))
     ,@body))

(defun imoogi-org-border-test--goto (text)
  "Move point to TEXT in the current buffer."
  (goto-char (point-min))
  (search-forward text)
  (beginning-of-line))

(ert-deftest imoogi-org-border-bounds-use-current-nested-heading ()
  (imoogi-org-border-test--with-buffer "* A\nbody\n** B\nb1\n*** C\nc1\n** D\nd1\n"
    (imoogi-org-border-test--goto "b1")
    (let ((bounds (imoogi-org-border--bounds)))
      (should bounds)
      (should (= (plist-get bounds :level) 2))
      (should (= (plist-get bounds :start)
                 (save-excursion
                   (imoogi-org-border-test--goto "** B")
                   (point))))
      (should (= (plist-get bounds :end)
                 (save-excursion
                   (imoogi-org-border-test--goto "** D")
                   (point))))
      (should (plist-get bounds :top))
      (should (plist-get bounds :bottom)))))

(ert-deftest imoogi-org-border-bounds-keep-distant-bottom-open ()
  (imoogi-org-border-test--with-buffer "* A\nbody-1\nbody-2\nbody-3\nbody-4\nbody-5\n* B\n"
    (let ((imoogi-org-border-scan-limit 18))
      (imoogi-org-border-test--goto "body-1")
      (let ((bounds (imoogi-org-border--bounds)))
        (should bounds)
        (should (= (plist-get bounds :level) 1))
        (should (plist-get bounds :top))
        (should-not (plist-get bounds :bottom))
        (should (< (plist-get bounds :end)
                   (save-excursion
                     (imoogi-org-border-test--goto "* B")
                     (point))))))))

(ert-deftest imoogi-org-border-bounds-recompute-after-heading-insertion ()
  (imoogi-org-border-test--with-buffer "* A\nold\n* C\n"
    (imoogi-org-border-test--goto "old")
    (let ((before (imoogi-org-border--bounds)))
      (should (= (plist-get before :end)
                 (save-excursion
                   (imoogi-org-border-test--goto "* C")
                   (point)))))
    (end-of-line)
    (insert "\n* B\nnew")
    (imoogi-org-border-test--goto "old")
    (let ((after (imoogi-org-border--bounds)))
      (should (= (plist-get after :end)
                 (save-excursion
                   (imoogi-org-border-test--goto "* B")
                   (point)))))))

(ert-deftest imoogi-org-border-mode-disable-cleans-timer-and-overlays ()
  (imoogi-org-border-test--with-buffer "* A\nbody\n"
    (imoogi-org-border-test--with-window-end
      (imoogi-org-border-mode 1)
      (imoogi-org-border--schedule)
      (should (timerp imoogi-org-border--timer))
      (imoogi-org-border--refresh (current-buffer))
      (should imoogi-org-border--overlays)
      (imoogi-org-border-mode -1)
      (should-not (timerp imoogi-org-border--timer))
      (should-not imoogi-org-border--overlays))))

(ert-deftest imoogi-org-border-refresh-skips-folded-body ()
  (imoogi-org-border-test--with-buffer "* A\nbody\n** B\nchild\n"
    (imoogi-org-border-test--with-window-end
      (imoogi-org-border-mode 1)
      (goto-char (point-min))
      (org-fold-hide-subtree)
      (imoogi-org-border--refresh (current-buffer))
      (let ((starts (mapcar #'overlay-start imoogi-org-border--overlays)))
        (should (member (point-min) starts))
        (should-not (member (save-excursion
                              (imoogi-org-border-test--goto "body")
                              (point))
                            starts))))))

(ert-deftest imoogi-org-border-refresh-is-window-specific ()
  (imoogi-org-border-test--with-buffer "* A\na\n* B\nb\n"
    (imoogi-org-border-test--with-window-end
      (imoogi-org-border-mode 1)
      (let ((left (selected-window))
            (right (split-window-right)))
        (set-window-buffer right (current-buffer))
        (set-window-point left (point-min))
        (set-window-point right (save-excursion
                                  (imoogi-org-border-test--goto "* B")
                                  (point)))
        (imoogi-org-border--refresh (current-buffer))
        (let ((left-overlays (cl-remove-if-not
                              (lambda (ov) (eq (overlay-get ov 'window) left))
                              imoogi-org-border--overlays))
              (right-overlays (cl-remove-if-not
                               (lambda (ov) (eq (overlay-get ov 'window) right))
                               imoogi-org-border--overlays)))
          (should left-overlays)
          (should right-overlays)
          (should (= (apply #'min (mapcar #'overlay-start left-overlays))
                     (point-min)))
          (should (= (apply #'min (mapcar #'overlay-start right-overlays))
                     (save-excursion
                       (imoogi-org-border-test--goto "* B")
                       (point)))))))))

(ert-deftest imoogi-org-border-refresh-respects-line-budget ()
  (imoogi-org-border-test--with-buffer "* A\nbody\nbody\nbody\nbody\n"
    (imoogi-org-border-test--with-window-end
      (let ((imoogi-org-border-line-limit 2))
        (imoogi-org-border-mode 1)
        (imoogi-org-border--refresh (current-buffer))
        (should (= (length imoogi-org-border--overlays) 2))))))

(ert-deftest imoogi-org-border-kill-buffer-hook-cleans-state ()
  (let ((buffer (generate-new-buffer " *imoogi-org-border-kill-test*")))
    (unwind-protect
        (with-current-buffer buffer
          (insert "* A\nbody\n")
          (let ((org-mode-hook nil)
                (org-startup-folded nil)
                (org-inhibit-startup t)
                (imoogi-org-border-enabled nil))
            (org-mode))
          (setq-local truncate-lines t)
          (switch-to-buffer buffer)
          (imoogi-org-border-test--with-window-end
            (imoogi-org-border-mode 1)
            (imoogi-org-border--schedule)
            (should (timerp imoogi-org-border--timer))
            (imoogi-org-border--refresh buffer)
            (should imoogi-org-border--overlays)
            (kill-buffer buffer)
            (should-not (buffer-live-p buffer))))
      (when (buffer-live-p buffer)
        (kill-buffer buffer)))))

(ert-deftest imoogi-org-border-does-not-parse-document-for-decoration ()
  (imoogi-org-border-test--with-buffer "* A\nbody\n"
    (imoogi-org-border-mode 1)
    (cl-letf (((symbol-function 'org-element-at-point)
               (lambda (&rest _) (error "Unexpected document parser"))))
      (imoogi-org-border--refresh (current-buffer))
      (should imoogi-org-border--overlays))))

(ert-deftest imoogi-org-border-unknown-heading-and-long-line-are-bounded ()
  (imoogi-org-border-test--with-buffer "* A\n"
    (let ((imoogi-org-border-scan-limit 100))
      (goto-char (point-max))
      (dotimes (_ 100) (insert "body\n"))
      (goto-char 250)
      (let ((bounds (imoogi-org-border--bounds)))
        (should bounds)
        (should-not (plist-get bounds :level))
        (should-not (plist-get bounds :top))
        (should-not (plist-get bounds :bottom)))
      (erase-buffer)
      (insert "* A\n" (make-string 1000 ?x))
      (goto-char 500)
      (should-not (imoogi-org-border--bounds)))))

(provide 'org-border-test)
;;; org-border-test.el ends here
