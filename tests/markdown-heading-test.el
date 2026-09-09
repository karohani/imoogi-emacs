;;; markdown-heading-test.el --- Markdown heading colors -*- lexical-binding: t; -*-

(require 'ert)
(require 'markdown-mode)
(require 'hl-line)

(ert-deftest imoogi-markdown-heading-colors-match-org-in-both-modes ()
  (dolist (mode '(markdown-mode gfm-mode))
    (with-temp-buffer
      (dotimes (index 6)
        (insert (make-string (1+ index) ?#) " Heading\n"))
      (funcall mode)
      (font-lock-ensure)
      (goto-char (point-min))
      (dotimes (index 6)
        (let ((face (intern (format "markdown-header-face-%d" (1+ index))))
              (org-face (intern (format "org-level-%d" (1+ index)))))
          (should (eq (get-text-property (line-end-position) 'face) face))
          (dolist (attribute '(:foreground :background))
            (should (equal (face-attribute face attribute nil t)
                           (face-attribute org-face attribute nil t))))
          (should (eq (face-attribute face :extend nil t) t)))
        (forward-line 1)))))

(ert-deftest imoogi-markdown-heading-highlight-respects-body-and-code ()
  (with-temp-buffer
    (insert "# Heading\nBody\n\n```\n# Not a heading\n```\n\nSetext\n======\n")
    (markdown-mode)
    (font-lock-ensure)
    (goto-char (point-min))
    (should-not (funcall hl-line-range-function))
    (forward-line 1)
    (should (funcall hl-line-range-function))
    (search-forward "# Not")
    (should (funcall hl-line-range-function))
    (should-not (memq 'markdown-header-face-1
                     (ensure-list (get-text-property (point) 'face))))
    (search-forward "Setext")
    (should (eq (get-text-property (1- (point)) 'face) 'markdown-header-face-1))
    (should-not (funcall hl-line-range-function))
    ;; Reapplying display settings preserves the buffer contents.
    (let ((before (buffer-string)))
      (imoogi-markdown-heading-setup)
      (font-lock-ensure)
      (should (equal before (buffer-string))))))

;;; markdown-heading-test.el ends here
