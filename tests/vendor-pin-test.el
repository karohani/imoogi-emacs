;;; vendor-pin-test.el --- pinned packages stay at their pinned versions -*- lexical-binding: t; -*-

;; Packages listed in `imoogi-pinned-packages' come from a fixed release, not the
;; rolling MELPA build. These tests fail if vendor/elpa drifts from the pin.

;;; Code:

(require 'ert)

(defvar imoogi-test-root)

(defun imoogi-test--pins ()
  "Load packages.el and return `imoogi-pinned-packages'."
  (load (expand-file-name "packages.el" imoogi-test-root) nil t)
  (symbol-value 'imoogi-pinned-packages))

(ert-deftest imoogi-vendor-pins-agent-shell-release ()
  "agent-shell and the two packages it requires are pinned to MELPA Stable releases."
  (let ((pins (imoogi-test--pins)))
    (should (equal (assq 'agent-shell pins) '(agent-shell "melpa-stable" "0.96.1")))
    (should (equal (assq 'acp pins) '(acp "melpa-stable" "0.18.1")))
    (should (equal (assq 'shell-maker pins) '(shell-maker "melpa-stable" "0.98.2")))
    (should (memq 'agent-shell (symbol-value 'imoogi-required-packages)))))

(ert-deftest imoogi-vendor-holds-exactly-the-pinned-versions ()
  "vendor/elpa has one directory per pinned package, at the pinned version."
  (let ((elpa (expand-file-name "vendor/elpa" imoogi-test-root)))
    (dolist (pin (imoogi-test--pins))
      (let* ((name (symbol-name (car pin)))
             (dirs (directory-files elpa nil (concat "\\`" (regexp-quote name) "-[0-9][0-9.]*\\'"))))
        (should (equal dirs (list (concat name "-" (nth 2 pin)))))))))

;;; vendor-pin-test.el ends here
