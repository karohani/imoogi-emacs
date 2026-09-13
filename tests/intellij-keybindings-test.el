;;; intellij-keybindings-test.el --- IntelliJ-style global key tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'cl-lib)

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

(ert-deftest imoogi-format-code-has-intellij-and-emacs-bindings ()
  (should (eq (lookup-key global-map (kbd "s-M-l"))
              #'imoogi-format-code))
  (should (eq (lookup-key global-map (kbd "C-M-\\"))
              #'imoogi-format-code)))

(ert-deftest imoogi-format-code-prefers-eglot-formatter ()
  (let (called)
    (cl-letf (((symbol-function 'eglot-managed-p) (lambda () t))
              ((symbol-function 'eglot-format)
               (lambda (&rest _) (interactive) (setq called 'eglot)))
              ((symbol-function 'indent-region)
               (lambda (&rest _) (ert-fail "indent fallback was used"))))
      (call-interactively #'imoogi-format-code)
      (should (eq called 'eglot)))))

(ert-deftest imoogi-format-code-uses-language-formatter-before-indentation ()
  (let (called)
    (cl-letf (((symbol-function 'eglot-managed-p) (lambda () nil))
              ((symbol-function 'derived-mode-p)
               (lambda (&rest modes) (memq 'go-mode modes)))
              ((symbol-function 'executable-find) (lambda (_) "/tmp/gofmt"))
              ((symbol-function 'gofmt) (lambda () (setq called 'gofmt)))
              ((symbol-function 'indent-region)
               (lambda (&rest _) (ert-fail "indent fallback was used"))))
      (imoogi-format-code)
      (should (eq called 'gofmt)))))

(ert-deftest imoogi-format-code-falls-back-to-major-mode-indentation ()
  (with-temp-buffer
    (insert "first\nsecond\n")
    (let (bounds)
      (cl-letf (((symbol-function 'eglot-managed-p) (lambda () nil))
                ((symbol-function 'indent-region)
                 (lambda (beg end &rest _) (setq bounds (cons beg end)))))
        (imoogi-format-code)
        (should (equal bounds (cons (point-min) (point-max))))))))

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
