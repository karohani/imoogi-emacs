;;; imoogi-org-border.el --- Lightweight current-subtree border -*- lexical-binding: t; -*-

;;; Code:

(require 'org)
(require 'cl-lib)

(defgroup imoogi-org-border nil
  "Draw a lightweight border around the current Org subtree."
  :group 'org)

(defcustom imoogi-org-border-enabled t
  "When non-nil, enable `imoogi-org-border-mode' in Org buffers."
  :type 'boolean
  :group 'imoogi-org-border)

(defcustom imoogi-org-border-scan-limit 20000
  "Maximum characters to scan before and after point for local subtree bounds."
  :type 'natnum
  :group 'imoogi-org-border)

(defcustom imoogi-org-border-idle-delay 0.1
  "Idle delay, in seconds, before refreshing Org border overlays."
  :type 'number
  :group 'imoogi-org-border)

(defvar-local imoogi-org-border--overlays nil
  "Active Org border overlays in the current buffer.")

(defvar-local imoogi-org-border--timer nil
  "Pending idle timer for refreshing Org border overlays.")

(defvar-local imoogi-org-border-mode nil
  "Non-nil when `imoogi-org-border-mode' is active.")

(defun imoogi-org-border--clear ()
  "Delete active Org border overlays in the current buffer."
  (mapc #'delete-overlay imoogi-org-border--overlays)
  (setq imoogi-org-border--overlays nil))

(defun imoogi-org-border--cancel-timer ()
  "Cancel a pending Org border refresh timer in the current buffer."
  (when (timerp imoogi-org-border--timer)
    (cancel-timer imoogi-org-border--timer))
  (setq imoogi-org-border--timer nil))

(defcustom imoogi-org-border-line-limit 200
  "Maximum physical lines decorated in each window."
  :type 'natnum :group 'imoogi-org-border)

(defface imoogi-org-border-unknown '((t (:inherit shadow)))
  "Border for a heading beyond the local search budget.")

(defun imoogi-org-border--bounds (&optional position)
  "Find local visual edges within the character budget; unknown edges stay open."
  (save-excursion
    (save-match-data
      (goto-char (or position (point)))
      (let* ((pos (point))
             (lo (max (point-min) (- pos imoogi-org-border-scan-limit)))
             (hi (min (point-max) (+ pos imoogi-org-border-scan-limit)))
             (bol (if (= pos (point-min)) pos
                    (if (re-search-backward "\n" lo t) (1+ (point))
                      (and (= lo (point-min)) lo)))))
        (when bol
          (goto-char bol)
          (let* ((found (or (looking-at "^\\*+[ \t]")
                            (re-search-backward "^\\*+[ \t]" lo t)))
                 (level (and found (- (match-end 0) (match-beginning 0) 1)))
                 (start (if found (match-beginning 0) lo))
                 (end hi) (bottom nil))
            (when found
              (goto-char (1+ start))
              (when (re-search-forward (format "^\\*\\{1,%d\\}[ \t]" level) hi t)
                (setq end (match-beginning 0) bottom t)))
            (unless (and (not found) (= lo (point-min)))
              (list :start start :end end :level level :top (and found t)
                    :bottom (or bottom (= end (point-max)))))))))))

(defun imoogi-org-border--level-face (level)
  "Return the Org level face for LEVEL."
  (if level (intern (format "org-level-%d" (1+ (% (1- level) 8))))
    'imoogi-org-border-unknown))

(defun imoogi-org-border--draw-window (window)
  "Decorate WINDOW without forcing redisplay, within character and line budgets."
  (with-selected-window window
    (save-excursion
      (let* ((bounds (imoogi-org-border--bounds (window-point window)))
             (start (and bounds (max (plist-get bounds :start) (window-start window))))
             (end (and bounds (min (plist-get bounds :end)
                                   (or (window-end window) (window-start window)))))
             (face (imoogi-org-border--level-face (plist-get bounds :level)))
             (color (face-foreground face nil t))
             (side (propertize "│" 'face (list :foreground color)))
             (count 0) (steps 0))
        (when (and truncate-lines start end (< start end))
          (goto-char start)
          (unless (bolp) (search-forward "\n" end 'move))
          (while (and (< (point) end) (< count imoogi-org-border-line-limit)
                      (< steps (* 4 imoogi-org-border-line-limit)))
            (cl-incf steps)
            (if (invisible-p (point))
                (goto-char (min end (next-char-property-change (point) end)))
              (let* ((beg (point))
                     (next (save-excursion (search-forward "\n" end 'move) (point)))
                     (eol (if (eq (char-before next) ?\n) (1- next) next)))
                (if (or (not (bolp)) (= beg next))
                    (goto-char end)
                  (let ((ov (make-overlay beg eol nil nil t)) edges)
                    (overlay-put ov 'window window)
                    (overlay-put ov 'before-string side)
                    (overlay-put ov 'after-string
                                 (concat (propertize " " 'display '(space :align-to (- right-fringe 1))) side))
                    (when (and (plist-get bounds :top) (= beg (plist-get bounds :start)))
                      (setq edges (list :overline color :extend t)))
                    (when (and (plist-get bounds :bottom) (>= next (plist-get bounds :end)))
                      (setq edges (append edges (list :underline color :extend t))))
                    (when edges (overlay-put ov 'face edges))
                    (push ov imoogi-org-border--overlays))
                  (cl-incf count)
                  (goto-char next))))))))))

(defun imoogi-org-border--refresh (buffer)
  "Refresh Org border overlays for BUFFER."
  (when (buffer-live-p buffer)
    (with-current-buffer buffer
      (imoogi-org-border--cancel-timer)
      (imoogi-org-border--clear)
      (when (and imoogi-org-border-mode
                 (derived-mode-p 'org-mode))
        (dolist (window (get-buffer-window-list buffer nil t))
          (imoogi-org-border--draw-window window))))))

(defun imoogi-org-border--schedule (&rest _)
  "Schedule an Org border refresh for the current buffer."
  (when (and imoogi-org-border-mode
             (derived-mode-p 'org-mode))
    (imoogi-org-border--cancel-timer)
    (setq imoogi-org-border--timer
          (run-with-idle-timer (max 0.01 imoogi-org-border-idle-delay)
                               nil
                               #'imoogi-org-border--refresh
                               (current-buffer)))))

(defun imoogi-org-border--window-scroll (window _display-start)
  "Schedule a border refresh after WINDOW scrolls."
  (when (eq (window-buffer window) (current-buffer))
    (imoogi-org-border--schedule)))

(defun imoogi-org-border--stop ()
  "Release timer and overlays when the buffer goes away."
  (imoogi-org-border--cancel-timer)
  (imoogi-org-border--clear))

(define-minor-mode imoogi-org-border-mode
  "Draw a lightweight border around the current Org subtree."
  :lighter ""
  (if imoogi-org-border-mode
      (progn
        (unless (derived-mode-p 'org-mode)
          (setq imoogi-org-border-mode nil)
          (user-error "This mode requires Org"))
        (add-hook 'after-change-functions #'imoogi-org-border--schedule nil t)
        (add-hook 'window-configuration-change-hook #'imoogi-org-border--schedule nil t)
        (add-hook 'kill-buffer-hook #'imoogi-org-border--stop nil t)
        (add-hook 'change-major-mode-hook #'imoogi-org-border--stop nil t)
        (add-hook 'post-command-hook #'imoogi-org-border--schedule nil t)
        (add-hook 'window-scroll-functions #'imoogi-org-border--window-scroll nil t)
        (imoogi-org-border--schedule))
    (remove-hook 'after-change-functions #'imoogi-org-border--schedule t)
    (remove-hook 'window-configuration-change-hook #'imoogi-org-border--schedule t)
    (remove-hook 'kill-buffer-hook #'imoogi-org-border--stop t)
    (remove-hook 'change-major-mode-hook #'imoogi-org-border--stop t)
    (remove-hook 'post-command-hook #'imoogi-org-border--schedule t)
    (remove-hook 'window-scroll-functions #'imoogi-org-border--window-scroll t)
    (imoogi-org-border--cancel-timer)
    (imoogi-org-border--clear)))

(provide 'imoogi-org-border)
;;; imoogi-org-border.el ends here
