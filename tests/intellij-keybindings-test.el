;;; intellij-keybindings-test.el --- IntelliJ-style global key tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)

(ert-deftest imoogi-close-current-buffer-is-bound-at-s-w ()
  (should (eq (lookup-key global-map (kbd "s-w"))
              #'kill-current-buffer)))

(ert-deftest imoogi-close-current-buffer-supports-korean-input ()
  (should (equal (lookup-key key-translation-map (kbd "s-ㅈ"))
                 (kbd "s-w"))))

;;; intellij-keybindings-test.el ends here
