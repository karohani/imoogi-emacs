;;; org-border-bounds-test.el --- Local outline prototype checks -*- lexical-binding: t; -*-

(require 'ert)
(load (expand-file-name "org-border-bench.el"
                        (file-name-directory (or load-file-name buffer-file-name))))

(ert-deftest imoogi-border-local-nested-and-sibling ()
  (with-temp-buffer
    (insert "* Parent\nbody\n** Child\nchild body\n** Sibling\nsibling body\n* Next\n")
    (goto-char (point-min))
    (search-forward "child body")
    (imoogi-border-bench--bounded-bounds)
    (should (equal (buffer-substring-no-properties
                    (car imoogi-border-bench--bounds)
                    (cdr imoogi-border-bench--bounds))
                   "** Child\nchild body\n"))
    (search-forward "sibling body")
    (imoogi-border-bench--bounded-bounds)
    (should (equal (buffer-substring-no-properties
                    (car imoogi-border-bench--bounds)
                    (cdr imoogi-border-bench--bounds))
                   "** Sibling\nsibling body\n"))))

(ert-deftest imoogi-border-local-unknown-edges-stay-open ()
  (with-temp-buffer
    ;; Place point beyond both lookup limits.
    (insert "* Far away\n")
    (dotimes (_ 20000) (insert "body\n"))
    (goto-char 50000)
    (let ((origin (line-beginning-position)))
      (imoogi-border-bench--bounded-bounds)
      (should-not (car imoogi-border-bench--edges))
      (should-not (cdr imoogi-border-bench--edges))
      (should (= (car imoogi-border-bench--bounds) (- origin 20000)))
      (should (= (cdr imoogi-border-bench--bounds) (+ origin 20000))))))

(ert-deftest imoogi-border-local-heading-edit-recomputes ()
  (with-temp-buffer
    (insert "* Parent\nbody\n** Child\nchild body\n* Next\n")
    (goto-char (point-min))
    (search-forward "child body")
    (beginning-of-line)
    (insert "*** New\n")
    (imoogi-border-bench--bounded-bounds)
    (should (string-prefix-p
             "*** New\n"
             (buffer-substring-no-properties
              (car imoogi-border-bench--bounds)
              (cdr imoogi-border-bench--bounds))))))

;;; org-border-bounds-test.el ends here
