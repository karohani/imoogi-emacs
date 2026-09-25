;;; org-roam-test.el --- central notes integration tests -*- lexical-binding: t; -*-

;;; Code:
(require 'ert)
(require 'org-roam)
(require 'transient)

(ert-deftest imoogi-org-roam-central-directory-and-cache ()
  (should (equal (file-name-as-directory (expand-file-name org-roam-directory))
                 (file-name-as-directory (expand-file-name "~/notes/permanent/"))))
  (should (fboundp 'org-roam-db-autosync-mode))
  (should (get 'org-roam-node-read--completions :memoize-original-function))
  (let ((before (symbol-function 'org-roam-node-read--completions)))
    (imoogi-org-roam-clear-completions-cache)
    (should-not (eq before (symbol-function 'org-roam-node-read--completions)))
    (should (get 'org-roam-node-read--completions :memoize-original-function))))

(ert-deftest imoogi-org-roam-transient-is-reachable ()
  (should (eq (lookup-key global-map (kbd "C-c n"))
              #'imoogi-org-roam-transient))
  (should (eq (plist-get (cdr (transient-get-suffix
                               'imoogi-org-agenda-transient "r")) :command)
              #'imoogi-org-roam-transient))
  (dolist (binding '(("f" . org-roam-node-find)
                     ("n" . org-roam-capture)
                     ("i" . org-roam-node-insert)
                     ("b" . org-roam-buffer-toggle)
                     ("R" . org-roam-node-random)
                     ("g" . org-roam-graph)
                     ("r" . org-roam-refile)
                     ("a" . org-roam-alias-add)
                     ("A" . org-roam-alias-remove)
                     ("t" . org-roam-tag-add)
                     ("T" . org-roam-tag-remove)
                     ("e" . org-roam-ref-add)
                     ("E" . org-roam-ref-remove)
                     ("d" . org-roam-dailies-goto-today)
                     ("D" . org-roam-dailies-goto-date)
                     ("s" . org-roam-db-sync)
                     ("c" . imoogi-org-roam-clear-completions-cache)
                     ("q" . transient-quit-one)))
    (should (eq (plist-get (cdr (transient-get-suffix
                                 'imoogi-org-roam-transient (car binding)))
                           :command)
                (cdr binding)))))

(ert-deftest imoogi-org-roam-refile-needs-org-heading-or-region ()
  (with-temp-buffer
    (should-not (imoogi-org-roam-refile-context-p))
    (org-mode)
    (insert "* Move me\n")
    (goto-char (point-min))
    (should (imoogi-org-roam-refile-context-p))))

(provide 'imoogi-org-roam-test)
;;; org-roam-test.el ends here
