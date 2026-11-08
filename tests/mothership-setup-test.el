;;; mothership-setup-test.el --- Mothership setup tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)

(defvar imoogi-mothership-directory)

(ert-deftest imoogi-mothership-setup-creates-layout-and-guide ()
  (let* ((home (make-temp-file "imoogi-mothership-" t))
         (imoogi-mothership-directory (expand-file-name "my brain/" home))
         (org-directory "/previous/")
         (org-roam-directory "/previous/permanent/")
         (org-agenda-files '("/previous/agenda.org")))
    (unwind-protect
        (progn
          (should (equal (imoogi-mothership-setup) imoogi-mothership-directory))
          (should (equal (sort (directory-files imoogi-mothership-directory nil
                                                 "^[^.]") #'string<)
                         '("00. Inbox" "10. Process" "20. Literature Notes"
                           "30. Permanent Notes" "40. Docs" "50. Assets"
                           "60. Collections" "70. Outputs" "80. References"
                           "90. Settings" "README.org")))
          (with-temp-buffer
            (insert-file-contents (expand-file-name "README.org" imoogi-mothership-directory))
            (org-mode)
            (let ((links 0))
              (org-element-map (org-element-parse-buffer) 'link
                (lambda (link)
                  (let ((path (org-element-property :path link)))
                    (when (and (equal (org-element-property :type link) "file")
                               (not (file-name-absolute-p path)))
                      (cl-incf links)
                      (should (file-directory-p
                               (expand-file-name path imoogi-mothership-directory)))))))
              (should (= links 10))))
          (should (equal org-directory "/previous/"))
          (should (equal org-roam-directory "/previous/permanent/"))
          (should (equal org-agenda-files '("/previous/agenda.org"))))
      (delete-directory home t))))

(ert-deftest imoogi-mothership-setup-preserves-guide-and-repairs-missing-folder ()
  (let* ((home (make-temp-file "imoogi-mothership-" t))
         (imoogi-mothership-directory (expand-file-name "brain/" home))
         (guide (expand-file-name "README.org" imoogi-mothership-directory))
         (note (expand-file-name "30. Permanent Notes/custom.org" imoogi-mothership-directory)))
    (unwind-protect
        (progn
          (imoogi-mothership-setup)
          (with-temp-file guide (insert "* 나의 안내\n"))
          (with-temp-file note (insert "* 보존할 지식\n"))
          (delete-directory (expand-file-name "10. Process" imoogi-mothership-directory))
          (imoogi-mothership-setup)
          (should (file-directory-p (expand-file-name "10. Process" imoogi-mothership-directory)))
          (should (equal (with-temp-buffer (insert-file-contents guide) (buffer-string))
                         "* 나의 안내\n"))
          (should (equal (with-temp-buffer (insert-file-contents note) (buffer-string))
                         "* 보존할 지식\n")))
      (delete-directory home t))))

(ert-deftest imoogi-mothership-setup-rejects-file-in-directory-position ()
  (let* ((home (make-temp-file "imoogi-mothership-" t))
         (imoogi-mothership-directory (file-name-as-directory home))
         (collision (expand-file-name "80. References" home)))
    (unwind-protect
        (progn
          (with-temp-file collision (insert "keep"))
          (should-error (imoogi-mothership-setup) :type 'file-error)
          (should (equal (directory-files home nil "^[^.]") '("80. References")))
          (should (equal (with-temp-buffer (insert-file-contents collision) (buffer-string)) "keep")))
      (delete-directory home t))))

(ert-deftest imoogi-mothership-setup-rejects-directory-as-guide ()
  (let* ((home (make-temp-file "imoogi-mothership-" t))
         (imoogi-mothership-directory (file-name-as-directory home)))
    (unwind-protect
        (progn
          (make-directory (expand-file-name "README.org" home))
          (should-error (imoogi-mothership-setup) :type 'file-error)
          (should (equal (directory-files home nil "^[^.]") '("README.org"))))
      (delete-directory home t))))

(ert-deftest imoogi-mothership-setup-missing-template-does-not-create-layout ()
  (let* ((home (make-temp-file "imoogi-mothership-" t))
         (imoogi-mothership-directory (expand-file-name "brain/" home))
         (imoogi-emacs-dir home))
    (unwind-protect
        (progn
          (should-error (imoogi-mothership-setup) :type 'file-error)
          (should-not (file-exists-p imoogi-mothership-directory)))
      (delete-directory home t))))

(ert-deftest imoogi-mothership-setup-interactive-opens-guide ()
  (let* ((home (make-temp-file "imoogi-mothership-" t))
         (imoogi-mothership-directory (expand-file-name "brain/" home))
         (guide (expand-file-name "README.org" imoogi-mothership-directory)))
    (unwind-protect
        (save-window-excursion
          (call-interactively #'imoogi-mothership-setup)
          (should (file-equal-p buffer-file-name guide)))
      (when-let* ((buffer (find-buffer-visiting guide))) (kill-buffer buffer))
      (delete-directory home t))))

;;; mothership-setup-test.el ends here
