;;; org-heading-test.el --- Org heading display regression tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'org)
(require 'hl-line)

(ert-deftest imoogi-org-heading-background-survives-current-line-highlight ()
  (save-window-excursion
    (with-temp-buffer
      (rename-buffer (generate-new-buffer-name "*org-heading-test*"))
      (switch-to-buffer (current-buffer))
      (insert "* 제목\n본문\n")
      (org-mode)
      (font-lock-ensure)
      (goto-char (point-min))
      (let ((global-hl-line-mode t))
	(global-hl-line-highlight)
	(unwind-protect
            (progn
              (should (= (overlay-start global-hl-line-overlay)
			 (overlay-end global-hl-line-overlay)))
              (end-of-line)
              (should (memq 'org-level-1
                            (ensure-list (get-text-property (point) 'face))))
              (should (eq t (face-attribute 'org-level-1 :extend)))
              (forward-line 1)
              (global-hl-line-highlight)
              (should (< (overlay-start global-hl-line-overlay)
			 (overlay-end global-hl-line-overlay))))
          (global-hl-line-unhighlight))))))

(ert-deftest imoogi-org-heading-reload-updates-fold-marker-without-unfolding ()
  (dolist (style '(text-properties overlays))
    (let ((org-fold-core-style style)
          (org-ellipsis "...")
          (org-mode-hook nil))
      (with-temp-buffer
        (insert "* 제목\n본문\n")
        (org-mode)
        (goto-char (point-min))
        (org-fold-hide-subtree)
        (imoogi-org-heading-setup)
        (should (equal org-ellipsis " ▼"))
        (should (equal (org-fold-core-get-folding-spec-property 'outline :ellipsis)
                       " ▼"))
        (should (equal (mapcar #'glyph-char
                               (display-table-slot buffer-display-table 4))
                       (string-to-list " ▼")))
        (forward-line 1)
        (should (org-invisible-p))))))

;;; org-heading-test.el ends here
