;;; org-archive-test.el --- Done-time logging and old-entry archiving -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'org)
(require 'org-archive)

(defun imoogi-org-archive-test--stamp (days-ago)
  "Return an inactive Org timestamp DAYS-AGO days before now."
  (format-time-string "[%Y-%m-%d %a %H:%M]"
                      (time-subtract (current-time) (days-to-time days-ago))))

(defun imoogi-org-archive-test--headings (file)
  "Return the heading titles of FILE in order."
  ;; The archive file carries no #+TODO line, so declare CANCELLED here too.
  (let ((org-todo-keywords '((sequence "TODO" "|" "DONE" "CANCELLED"))))
    (with-temp-buffer
      (insert-file-contents file)
      (org-mode)
      (org-map-entries (lambda () (org-get-heading t t t t))))))

(defmacro imoogi-org-archive-test--with-file (var &rest body)
  "Bind VAR to a fresh agenda file with mixed entries, then run BODY."
  (declare (indent 1))
  `(let* ((dir (make-temp-file "imoogi-org-archive-" t))
          (,var (expand-file-name "agenda.org" dir))
          (org-archive-location "%s_archive::")
          (org-archive-save-context-info nil))
     (unwind-protect
         (progn
           (with-temp-file ,var
             (insert "#+TODO: TODO | DONE CANCELLED\n"
                     "* DONE old done\n"
                     "CLOSED: " (imoogi-org-archive-test--stamp 400) "\n"
                     "* CANCELLED old cancelled\n"
                     "CLOSED: " (imoogi-org-archive-test--stamp 500) "\n"
                     "* DONE recent done\n"
                     "CLOSED: " (imoogi-org-archive-test--stamp 10) "\n"
                     "* DONE done without closed\n"
                     "* TODO still open\n"
                     "** DONE old child\n"
                     "CLOSED: " (imoogi-org-archive-test--stamp 800) "\n"))
           ,@body)
       (let ((buffer (find-buffer-visiting ,var)))
         (when buffer
           (with-current-buffer buffer (set-buffer-modified-p nil))
           (kill-buffer buffer)))
       (let ((archive (find-buffer-visiting (concat ,var "_archive"))))
         (when archive
           (with-current-buffer archive (set-buffer-modified-p nil))
           (kill-buffer archive)))
       (delete-directory dir t))))

(ert-deftest imoogi-org-log-done-records-completion-time ()
  "Finishing a TODO records a CLOSED timestamp (t6 prerequisite)."
  (should (eq org-log-done 'time)))

(ert-deftest imoogi-org-archive-old-done-moves-only-old-finished-entries ()
  "Only done entries closed before the cutoff move to the archive file."
  (imoogi-org-archive-test--with-file file
    (with-current-buffer (find-file-noselect file)
      (cl-letf (((symbol-function 'y-or-n-p) (lambda (&rest _) t)))
        (should (= 3 (imoogi-org-archive-old-done 365))))
      (should-not (buffer-modified-p)))
    (should (equal (imoogi-org-archive-test--headings file)
                   '("recent done" "done without closed" "still open")))
    (should (equal (sort (imoogi-org-archive-test--headings
                          (concat file "_archive"))
                         #'string<)
                   '("old cancelled" "old child" "old done")))))

(ert-deftest imoogi-org-archive-old-done-declined-changes-nothing ()
  "Declining the confirmation leaves the file untouched."
  (imoogi-org-archive-test--with-file file
    (let ((before (with-temp-buffer (insert-file-contents file) (buffer-string))))
      (with-current-buffer (find-file-noselect file)
        (cl-letf (((symbol-function 'y-or-n-p) (lambda (&rest _) nil)))
          (should (= 0 (imoogi-org-archive-old-done 365)))))
      (should (equal (with-temp-buffer (insert-file-contents file) (buffer-string))
                     before))
      (should-not (file-exists-p (concat file "_archive"))))))

(ert-deftest imoogi-org-archive-old-done-nothing-to-move-does-not-ask ()
  "With no entry past the cutoff, no question is asked and nothing moves."
  (imoogi-org-archive-test--with-file file
    (with-current-buffer (find-file-noselect file)
      (cl-letf (((symbol-function 'y-or-n-p)
                 (lambda (&rest _) (error "Should not ask"))))
        (should (= 0 (imoogi-org-archive-old-done 1000)))))))

(provide 'org-archive-test)
;;; org-archive-test.el ends here
