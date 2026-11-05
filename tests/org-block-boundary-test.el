;;; org-block-boundary-test.el --- Org snippet initialization boundaries -*- lexical-binding: t; -*-

(require 'ert)
(require 'org)
(require 'org-src)
(require 'yasnippet)

(ert-deftest imoogi-org-boundary-hook-initializes-local-snippets ()
  "Org snippets also work before the global after-init hook has run."
  (let ((yas-global-mode nil))
    (with-temp-buffer
      (org-mode)
      (should yas-minor-mode)
      (should (memq #'yas--post-command-handler post-command-hook))
      (insert "<s>")
      (should (yas-expand))
      (should yas--active-field-overlay)
      (yas-exit-all-snippets))))

(ert-deftest imoogi-org-boundary-command-initializes-disabled-snippets ()
  (with-temp-buffer
    (org-mode)
    (yas-minor-mode -1)
    (imoogi-org-insert-source-block)
    (should yas-minor-mode)
    (should (memq #'yas--post-command-handler post-command-hook))
    (should yas--active-field-overlay)
    (should (equal (buffer-string) "#+begin_src language\n\n#+end_src"))
    (yas-exit-all-snippets)))

(ert-deftest imoogi-org-boundary-completion-initializes-disabled-snippets ()
  (with-temp-buffer
    (org-mode)
    (yas-minor-mode -1)
    (insert "<e>")
    (let* ((completion (imoogi-org--block-completion-candidates))
           (exit (plist-get (nthcdr 3 completion) :exit-function)))
      (should (functionp exit))
      (funcall exit "<e>" 'finished))
    (should yas-minor-mode)
    (should (equal (buffer-string) "#+begin_example\n\n#+end_example"))
    (yas-exit-all-snippets)))

(ert-deftest imoogi-org-boundary-non-org-command-does-not-mutate ()
  (with-temp-buffer
    (insert "untouched")
    (should-error (imoogi-org-insert-source-block) :type 'user-error)
    (should (equal (buffer-string) "untouched"))
    (should-not (bound-and-true-p yas-minor-mode))))

(ert-deftest imoogi-org-boundary-json-respects-disabled-remapping ()
  (let ((major-mode-remap-alist nil)
        (major-mode-remap-defaults nil))
    (should (eq (org-src-get-lang-mode "json") 'js-json-mode))))

(ert-deftest imoogi-org-boundary-json-respects-enabled-remapping ()
  (let ((major-mode-remap-alist '((js-json-mode . json-ts-mode)))
        (major-mode-remap-defaults nil))
    (should (eq (org-src-get-lang-mode "json") 'json-ts-mode))))

(provide 'org-block-boundary-test)
