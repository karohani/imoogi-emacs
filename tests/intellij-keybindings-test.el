;;; intellij-keybindings-test.el --- IntelliJ-style global key tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)

(defconst imoogi-test-korean-key-pairs
  '(("ㅁ" . "a") ("ㅠ" . "b") ("ㅊ" . "c") ("ㅇ" . "d")
    ("ㄷ" . "e") ("ㄹ" . "f") ("ㅎ" . "g") ("ㅗ" . "h")
    ("ㅑ" . "i") ("ㅓ" . "j") ("ㅏ" . "k") ("ㅣ" . "l")
    ("ㅡ" . "m") ("ㅜ" . "n") ("ㅐ" . "o") ("ㅔ" . "p")
    ("ㅂ" . "q") ("ㄱ" . "r") ("ㄴ" . "s") ("ㅅ" . "t")
    ("ㅕ" . "u") ("ㅍ" . "v") ("ㅈ" . "w") ("ㅌ" . "x")
    ("ㅛ" . "y") ("ㅋ" . "z")))

(ert-deftest imoogi-close-current-buffer-is-bound-at-s-w ()
  (should (eq (lookup-key global-map (kbd "s-w"))
              #'kill-current-buffer)))

(ert-deftest imoogi-modified-commands-support-korean-input ()
  (dolist (modifier '("C-" "M-" "s-"))
    (dolist (pair imoogi-test-korean-key-pairs)
      (should (equal
               (lookup-key key-translation-map
                           (kbd (concat modifier (car pair))))
               (kbd (concat modifier (cdr pair))))))))

(ert-deftest imoogi-save-buffer-supports-korean-input ()
  (should (equal (lookup-key key-translation-map (kbd "s-ㄴ"))
                 (kbd "s-s")))
  (should (eq (key-binding (kbd "s-s")) #'save-buffer)))

;;; intellij-keybindings-test.el ends here
