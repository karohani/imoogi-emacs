;;; org-block-test.el --- Org block insertion and completion tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'org)
(require 'transient)
(require 'yasnippet)
(require 'org-src)

(ert-deftest imoogi-org-json-source-block-uses-json-mode ()
  (should (eq (org-src-get-lang-mode "json") 'js-json-mode)))

(ert-deftest imoogi-org-block-source-template-has-language-field ()
  (should (equal (imoogi-org--block-template 'source)
                 "#+begin_src ${1:language}\n$0\n#+end_src")))

(ert-deftest imoogi-org-block-insertion-uses-yasnippet-fields ()
  (with-temp-buffer
    (org-mode)
    (imoogi-org-insert-source-block)
    (should (equal (buffer-string)
                   "#+begin_src language\n\n#+end_src"))
    (should yas--active-field-overlay)
    (yas-exit-all-snippets))
  (with-temp-buffer
    (org-mode)
    (imoogi-org-insert-example-block)
    (should (equal (buffer-string)
                   "#+begin_example\n\n#+end_example"))
    (yas-exit-all-snippets)))

(ert-deftest imoogi-org-block-shortcut-completion-offers-source-and-example ()
  (with-temp-buffer
    (org-mode)
    (insert "<s")
    (let ((completion (imoogi-org--block-completion-candidates)))
      (should (= (nth 0 completion) 1))
      (should (= (nth 1 completion) 3))
      (should (equal (nth 2 completion) '("<s>" "<e>"))))))

(ert-deftest imoogi-org-block-shortcut-expands-source-snippet ()
  (with-temp-buffer
    (org-mode)
    (insert "<s>")
    (yas-expand)
    (should (equal (buffer-string)
                   "#+begin_src language\n\n#+end_src"))
    (yas-exit-all-snippets)))

(ert-deftest imoogi-org-block-shortcut-inhibits-angle-pairing ()
  (with-temp-buffer
    (org-mode)
    (should (funcall electric-pair-inhibit-predicate ?<))))

(ert-deftest imoogi-org-block-transient-is-reachable-from-org-menu ()
  (should (eq (plist-get (cdr (transient-get-suffix
                               'imoogi-org-agenda-transient "b"))
                         :command)
              #'imoogi-org-block-transient))
  (should (eq (plist-get (cdr (transient-get-suffix
                               'imoogi-org-block-transient "s"))
                         :command)
              #'imoogi-org-insert-source-block))
  (should (eq (plist-get (cdr (transient-get-suffix
                               'imoogi-org-block-transient "e"))
                         :command)
              #'imoogi-org-insert-example-block)))

(provide 'org-block-test)
;;; org-block-test.el ends here
