;;; org-agenda-overview-test.el --- Agenda overdue grouping -*- lexical-binding: t; -*-
(require 'ert)
(require 'org-agenda)

(ert-deftest imoogi-org-agenda-overdue-entries-appear-once-above-agenda ()
  (let* ((file (make-temp-file "imoogi-overdue-" nil ".org"))
         (org-agenda-files (list file))
         (org-agenda-buffer-name "*imoogi-overdue-test*")
         result-buffer
         (org-agenda-start-day "-3d")
         (org-agenda-span 7)
         (org-agenda-start-on-weekday nil)
         (org-agenda-sticky nil)
         (org-agenda-window-setup 'current-window)
         (today (org-today))
         (date (lambda (day)
                 (format-time-string "%Y-%m-%d" (org-time-from-absolute day)))))
    (unwind-protect
        (save-window-excursion
          (with-temp-file file
            (insert (format "* TODO OverdueBoth\nDEADLINE: <%s> SCHEDULED: <%s>\n** TODO ChildToday\nSCHEDULED: <%s>\n* OverduePlain\nDEADLINE: <%s>\n* DONE FinishedPast\nDEADLINE: <%s>\n* TODO DueToday\nDEADLINE: <%s>\n* TODO DueFuture\nDEADLINE: <%s>\n"
                            (funcall date (- today 2)) (funcall date today)
                            (funcall date today) (funcall date (- today 1))
                            (funcall date (- today 1)) (funcall date today)
                            (funcall date (+ today 2)))))
          (org-agenda nil "a")
          (setq result-buffer (current-buffer))
          (with-current-buffer result-buffer
            (dolist (name '("OverdueBoth" "OverduePlain"))
              (goto-char (point-min))
              (should (= (how-many name (point-min) (point-max)) 1)))
            (goto-char (point-min))
            (should (search-forward "기한 지난 항목" nil t))
            (let ((end (save-excursion (search-forward "===="))))
              (should-not (search-forward "FinishedPast" end t)))
            (should (search-forward "OverdueBoth" nil t))
            (should (search-forward "OverduePlain" nil t))
            (should (search-forward "DueToday" nil t))
            (should (search-forward "DueFuture" nil t))
            (should (string-match-p "ChildToday" (buffer-string)))
            ;; Refresh must retain the two-block layout and avoid duplicates.
            (org-agenda-redo)
            (should (= (how-many "OverdueBoth" (point-min) (point-max)) 1))))
      (when (buffer-live-p result-buffer) (kill-buffer result-buffer))
      (when (find-buffer-visiting file) (kill-buffer (find-buffer-visiting file)))
      (delete-file file))))
