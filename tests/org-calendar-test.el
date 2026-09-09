;;; org-calendar-test.el --- Calendar popup layout -*- lexical-binding: t; -*-
(require 'ert)
(require 'calendar)
(require 'face-remap)

(ert-deftest imoogi-org-calendar-fits-after-late-zoom ()
  (save-window-excursion
    (let ((buffer (generate-new-buffer " *calendar-layout-test*")))
      (unwind-protect
          (progn
            (delete-other-windows)
            (switch-to-buffer (get-buffer-create " *calendar-parent*"))
            (let ((window (split-window-below)))
              (set-window-buffer window buffer)
              (with-selected-window window
                (calendar-mode)
                (let ((inhibit-read-only t)) (calendar-generate 9 2026))
                (goto-char (point-min))
                (text-scale-set 4)
                (set-window-start window (line-beginning-position 3))
                (imoogi-org-calendar-fit-window window)
                (should (= (window-start window) (point-min)))
                (should (= (window-vscroll window) 0))
                (should (memq #'imoogi-org-calendar-fit-window
                              text-scale-mode-hook))
                (should (memq #'imoogi-org-calendar-fit-window
                              window-buffer-change-functions))
                (should (= text-scale-mode-amount 4)))))
        (kill-buffer buffer)
        (kill-buffer " *calendar-parent*")))))

(ert-deftest imoogi-org-calendar-responsive-gui ()
  (skip-unless (display-graphic-p))
  (let* ((frame (selected-frame))
         (old-size (cons (frame-pixel-width) (frame-pixel-height)))
         (old-height (face-attribute 'default :height frame))
         (calendar-buffer " *calendar-responsive-test*")
         (display-buffer-alist
          (cons `(,(regexp-quote calendar-buffer)
                  (display-buffer-in-side-window) (side . bottom))
                display-buffer-alist)))
    (unwind-protect
        (save-window-excursion
          (set-face-attribute 'default frame :height 160)
          (calendar)
          (with-current-buffer calendar-buffer (text-scale-set 4))
          (dolist (size '((1800 1000) (850 600) (1800 1000)))
            (set-frame-size frame (car size) (cadr size) t)
            ;; Redisplay runs the actual window change hooks.
            (redisplay t)
            (with-current-buffer calendar-buffer
              (let* ((window (get-buffer-window calendar-buffer))
                     (pixels (window-text-pixel-size window nil t 10000 10000)))
                (should (eq (window-parameter window 'window-side) 'bottom))
                (should (<= (window-pixel-height window)
                            (* 0.4 (window-pixel-height (frame-root-window frame)))))
                (should (<= (car pixels) (window-body-width window t)))
                (should (<= (cdr pixels) (window-body-height window t)))
                (should (= (window-start window) (point-min)))
                (should (= text-scale-mode-amount 4))
                (if (= (car size) 850)
                    (should imoogi-org-calendar--fit-cookie)
                  (should-not imoogi-org-calendar--fit-cookie))))))
      (when (get-buffer calendar-buffer) (kill-buffer calendar-buffer))
      (set-face-attribute 'default frame :height old-height)
      (set-frame-size frame (car old-size) (cdr old-size) t))))

;;; org-calendar-test.el ends here
