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
                ;; Org can resize an already fitted calendar while prompting.
                ;; The frame, content and zoom have not changed in this case.
                (window-resize window -60 nil window t)
                (redisplay t)
                (should (<= (cdr pixels) (window-body-height window t)))
                (if (= (car size) 850)
                    (should imoogi-org-calendar--fit-cookie)
                  (should-not imoogi-org-calendar--fit-cookie))))))
      (when (get-buffer calendar-buffer) (kill-buffer calendar-buffer))
      (set-face-attribute 'default frame :height old-height)
      (set-frame-size frame (car old-size) (cdr old-size) t))))

(ert-deftest imoogi-org-calendar-date-prompt-with-persisted-zoom-gui ()
  (skip-unless (display-graphic-p))
  (require 'persist-text-scale)
  (let ((was-enabled persist-text-scale-mode)
        (persist-text-scale--data nil)
        (persist-text-scale-default-text-scale-amount 4)
        (persist-text-scale-fallback-to-previous-scale nil)
        (persist-text-scale-autosave-interval nil)
        (persist-text-scale-file (make-temp-file "calendar-scale-"))
        observed timer)
    (unwind-protect
        (save-window-excursion
          (persist-text-scale-mode 1)
          (with-temp-buffer
            (switch-to-buffer (current-buffer))
            (insert "* TODO Test\n")
            (org-mode)
            (goto-char (point-min))
            (let ((minibuffer-setup-hook
                   (cons
                    (lambda ()
                      (setq timer
                            (run-at-time
                             0.2 nil
                             (lambda ()
                               (redisplay t)
                               (let ((w (get-buffer-window calendar-buffer)))
                                 (setq observed
                                       (and w
                                            (with-current-buffer calendar-buffer
                                              (list (window-body-height w t)
                                                    (cdr (window-text-pixel-size
                                                          w nil t 10000 10000))
                                                    text-scale-mode-amount)))))
                               (when (active-minibuffer-window)
                                 (with-current-buffer
                                     (window-buffer (active-minibuffer-window))
                                   (delete-minibuffer-contents)
                                   (insert "2026-09-09")
                                   (exit-minibuffer)))))))
                    minibuffer-setup-hook)))
              ;; Run the menu's actual command with a real recursive minibuffer.
              ;; Keyboard macros suppress the redisplay hooks being tested.
              (call-interactively
               (plist-get (cdr (transient-get-suffix
                                'imoogi-org-agenda-transient "s")) :command)))
            (should observed)
            (should (= (nth 2 observed) 4))
            (should (>= (car observed) (cadr observed)))
            (should (string-match-p "SCHEDULED: <2026-09-09" (buffer-string)))))
      (when timer (cancel-timer timer))
      (unless was-enabled (persist-text-scale-mode -1))
      (delete-file persist-text-scale-file))))

;;; org-calendar-test.el ends here
