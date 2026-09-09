;;; org-border-bench.el --- Temporary Org border prototype benchmark -*- lexical-binding: t; -*-

;; Run only in a dedicated GUI Emacs. Does not install a feature/hook.
;; (load-file "tests/benchmarks/org-border-bench.el")
;; (imoogi-border-bench-run "/tmp/imoogi-org-border-results.json")
;; Each variant uses a fresh fixture. Optional args select shapes, line counts,
;; variant order and sample count. Rescan is opt-in: large subtrees can be slow.
;; Timings include synchronous redisplay but not compositor/input event latency.

(require 'org)
(require 'json)
(require 'cl-lib)

(defvar imoogi-border-bench-disable-org-menu nil)
(defvar-local imoogi-border-bench--bounds nil)
(defvar-local imoogi-border-bench--overlays nil)

(defun imoogi-border-bench--clear ()
  (mapc #'delete-overlay imoogi-border-bench--overlays)
  (setq imoogi-border-bench--overlays nil
        imoogi-border-bench--bounds nil))

(defun imoogi-border-bench--org-bounds (rescan)
  "Draw a viewport-clipped outline around the current subtree.
RESCAN deliberately represents recomputing subtree bounds on every command.
Cached variant reuses markers for body edits; real structural invalidation
would still need implementation and tests before shipping this prototype."
  (when (or rescan (null imoogi-border-bench--bounds)
            (< (point) (car imoogi-border-bench--bounds))
            (>= (point) (cdr imoogi-border-bench--bounds)))
    (setq imoogi-border-bench--bounds
          (save-excursion
            (org-back-to-heading t)
            (cons (copy-marker (point))
                  (progn (org-end-of-subtree t t)
                         (copy-marker (point) t)))))))

(defvar-local imoogi-border-bench--edges '(t . t))

(defun imoogi-border-bench--bounded-bounds ()
  "Find local outline edges within 20,000 characters in each direction.
This visual-only prototype does not use the Org element parser. Unknown
edges remain open. It is not a semantic Org parser or a refile helper."
  (save-excursion
    (let* ((origin (line-beginning-position))
           (lo (max (point-min) (- origin 20000)))
           (hi (min (point-max) (+ origin 20000)))
           (heading (progn (goto-char origin)
                           (or (looking-at "^\\*+ ")
                               (re-search-backward "^\\*+ " lo t))))
           (level (and heading (- (match-end 0) (match-beginning 0) 1)))
           (start (if heading (line-beginning-position) lo))
           (end hi) (end-known nil))
      (when heading
        (forward-line 1)
        (when (re-search-forward (format "^\\*\\{1,%d\\} " level) hi t)
          (setq end (line-beginning-position) end-known t)))
      (setq imoogi-border-bench--bounds (cons (copy-marker start) (copy-marker end t))
            imoogi-border-bench--edges (cons heading (or end-known (= end (point-max))))))))

(defun imoogi-border-bench--update (variant)
  "Compare parsing, rendering and bounded local outline lookup separately."
  (setq imoogi-border-bench--edges '(t . t))
  (pcase variant
    ('bounded (imoogi-border-bench--bounded-bounds))
    ('render-only
     ;; Viewport-wide sides isolate drawing; these are not semantic edges.
     (setq imoogi-border-bench--bounds
           (cons (copy-marker (window-start))
                 (copy-marker (window-end nil t) t))
           imoogi-border-bench--edges '(nil . nil)))
    (_ (imoogi-border-bench--org-bounds (eq variant 'rescan))))
  (unless (eq variant 'parser-only)
    (imoogi-border-bench--draw)))

(defun imoogi-border-bench--draw ()
  (let* ((start (max (marker-position (car imoogi-border-bench--bounds))
                     (window-start)))
         (end (min (marker-position (cdr imoogi-border-bench--bounds))
                   (window-end nil t)))
         (pool imoogi-border-bench--overlays)
         (used nil))
    (save-excursion
      (goto-char start)
      (beginning-of-line)
      (while (< (point) end)
        (let ((ov (or (pop pool) (make-overlay (point) (point)))))
          (move-overlay ov (point) (line-end-position))
          (overlay-put ov 'before-string (propertize "│" 'face 'org-level-2))
          (overlay-put ov 'after-string
                       (concat (propertize " " 'display '(space :align-to (- right-fringe 1)))
                               (propertize "│" 'face 'org-level-2)))
          ;; Horizontal edges only at actual subtree boundaries, not viewport cuts.
          (overlay-put ov 'face
                       (cond ((and (car imoogi-border-bench--edges)
                                    (= (point) (car imoogi-border-bench--bounds)))
                              '(:overline "#82b7ff" :extend t))
                             ((and (cdr imoogi-border-bench--edges)
                                    (>= (1+ (line-end-position)) (cdr imoogi-border-bench--bounds)))
                              '(:underline "#82b7ff" :extend t))))
          (push ov used))
        (forward-line 1)))
    (mapc #'delete-overlay pool)
    (setq imoogi-border-bench--overlays (nreverse used))))

(defun imoogi-border-bench--stats (times)
  (let* ((sorted (sort (copy-sequence times) #'<))
         (n (length sorted)))
    `((median_ms . ,(nth (/ n 2) sorted))
      (p95_ms . ,(nth (min (1- n) (1- (ceiling (* n .95)))) sorted))
      (max_ms . ,(car (last sorted))))))

(defun imoogi-border-bench--sample (variant action count)
  (let ((times nil) (operation-times nil) (update-times nil) (display-times nil) (peak 0) (gc-start gcs-done) (gc-time gc-elapsed))
    (dotimes (i (+ count 5))
      (let ((start (float-time)) operation-end update-end)
        (pcase action
          ('edit (if (zerop (% i 2)) (insert "x") (delete-char -1)))
          ('move (forward-line (if (zerop (% i 2)) 1 -1)))
          ('scroll (forward-line 5) (set-window-start nil (line-beginning-position) t))
          ('jump (goto-char (+ (point-min) (floor (* (- (point-max) 100)
                                                    (/ (float (1+ (% i 19))) 20.0)))))
                 (beginning-of-line)
                 (unless (org-at-heading-p) (forward-line 1))
                 (set-window-start nil (line-beginning-position) t)))
        (setq operation-end (float-time))
        (unless (eq variant 'baseline)
          (imoogi-border-bench--update variant))
        (setq update-end (float-time))
        (redisplay t)
        (when (>= i 5)
          (push (* 1000 (- operation-end start)) operation-times)
          (push (* 1000 (- update-end operation-end)) update-times)
          (push (* 1000 (- (float-time) update-end)) display-times)
          (push (* 1000 (- (float-time) start)) times))
        (setq peak (max peak (length imoogi-border-bench--overlays)))))
    (when (and (eq action 'edit) (= (% (+ count 5) 2) 1))
      (delete-char -1))
    `((action . ,action) (samples . ,count) (timing . ,(imoogi-border-bench--stats times))
      (operation . ,(imoogi-border-bench--stats operation-times))
      (update . ,(imoogi-border-bench--stats update-times))
      (redisplay . ,(imoogi-border-bench--stats display-times))
      (peak_overlays . ,peak) (gc_count . ,(- gcs-done gc-start))
      (gc_seconds . ,(- gc-elapsed gc-time)))))

(defun imoogi-border-bench-run (output &optional shapes sizes variants samples)
  "Benchmark generated fixtures in the selected GUI window; write OUTPUT.
Use a dedicated GUI process, not a working Emacs session.
SHAPES, SIZES, VARIANTS and SAMPLES restrict the run.
Restore the window configuration and kill all generated buffers."
  (interactive "FResults JSON: ")
  (unless (display-graphic-p) (user-error "Use GUI Emacs for redisplay timings"))
  (let ((results nil) (buffers nil)
        (metadata `((emacs . ,emacs-version) (system . ,(symbol-name system-type))
                    (columns . ,(window-body-width)) (rows . ,(window-body-height))
                    (gc_threshold . ,gc-cons-threshold)
                    (org_menu_disabled . ,(if imoogi-border-bench-disable-org-menu t :json-false))
                    (note . "Synthetic local operations + redisplay; not end-to-end keyboard/compositor latency. Cached prototype lacks structural-edit invalidation."))))
    (unwind-protect
        (save-window-excursion
          (dolist (shape (or shapes '(many-headings long-subtree)))
            (dolist (lines (or sizes '(10000 100000 1000000)))
              (dolist (variant (or variants '(baseline cached)))
              (let ((buffer (generate-new-buffer " *imoogi-border-benchmark*")))
                (push buffer buffers)
                (switch-to-buffer buffer)
                (let ((inhibit-modification-hooks t))
                  (if (eq shape 'many-headings)
                      (let ((chunk (concat "* Heading\n" (apply #'concat (make-list 9 "Body paragraph with Korean 한글 and plain text.\n")))))
                        (dotimes (_ (/ lines 10)) (insert chunk)))
                    (insert "* One huge heading\n")
                    (let ((chunk (apply #'concat (make-list 100 "Long body paragraph 한글 with ordinary text.\n"))))
                      (dotimes (_ (/ lines 100)) (insert chunk)))))
                (let ((org-startup-folded nil) (org-inhibit-startup t)) (org-mode))
                (when imoogi-border-bench-disable-org-menu
                  (let ((map (copy-keymap (current-local-map))))
                    (define-key map [menu-bar] nil)
                    (use-local-map map)))
                (org-fold-show-all)
                (setq-local truncate-lines t)
                (setq-local buffer-undo-list t)
                (set-buffer-modified-p nil)
                (progn
                  (imoogi-border-bench--clear)
                  (goto-char (point-min)) (forward-line 2)
                  (set-window-start nil (point-min) t)
                  (redisplay t)
                  (garbage-collect)
                  (let ((start (float-time)) entry)
                    (unless (eq variant 'baseline)
                      (imoogi-border-bench--update variant))
                    (redisplay t)
                    (setq entry (* 1000 (- (float-time) start)))
                    (let ((actions nil))
                      (dolist (action '(edit move scroll jump))
                        (imoogi-border-bench--clear)
                        (goto-char (point-min)) (forward-line 2)
                        (set-window-start nil (point-min) t)
                        (redisplay t)
                        (push (imoogi-border-bench--sample variant action (or samples 20)) actions))
                      (push `((shape . ,shape) (lines . ,lines) (characters . ,(buffer-size)) (bytes . ,(1- (position-bytes (point-max))))
                              (variant . ,variant) (entry_ms . ,entry)
                              (actions . ,(vconcat (nreverse actions)))) results))))
                (imoogi-border-bench--clear)
                (kill-buffer buffer)
                (with-temp-file output
                  (insert (json-encode `((metadata . ,metadata) (results . ,(vconcat (reverse results)))))))))))
          (with-temp-file output
            (insert (json-encode `((metadata . ,metadata) (results . ,(vconcat (nreverse results))))))))
      (dolist (buffer buffers)
        (when (buffer-live-p buffer) (kill-buffer buffer)))))
  output)

(provide 'org-border-bench)
;;; org-border-bench.el ends here
