;;; keyboard-quit-test.el --- Quit routing regression tests -*- lexical-binding: t; -*-
(require 'ert)
(require 'cl-lib)

(ert-deftest imoogi-quit-cancels-minibuffer-from-another-window ()
  (save-window-excursion
    (delete-other-windows)
    (let* ((origin (selected-window))
           (input (split-window-below))
           called)
      (cl-letf (((symbol-function 'active-minibuffer-window) (lambda () input))
                ((symbol-function 'minibuffer-keyboard-quit)
                 (lambda () (setq called (selected-window))))
                ((symbol-function 'keyboard-quit)
                 (lambda () (interactive) (ert-fail "Used ordinary quit with active minibuffer"))))
        (call-interactively (key-binding (kbd "C-g")))
        (should (eq called input))
        (should (eq (selected-window) origin))))))

(ert-deftest imoogi-quit-preserves-ordinary-quit ()
  (cl-letf (((symbol-function 'active-minibuffer-window) (lambda () nil)))
    (should (eq (condition-case nil
                    (progn (call-interactively (key-binding (kbd "C-g"))) nil)
                  (quit 'quit))
                'quit))))

(ert-deftest imoogi-quit-preserves-minibuffer-local-binding ()
  (should (memq (lookup-key minibuffer-local-map (kbd "C-g"))
                '(minibuffer-keyboard-quit abort-minibuffers))))
