;;; keyboard-quit-test.el --- Quit routing regression tests -*- lexical-binding: t; -*-
(require 'ert)
(require 'cl-lib)

(defconst imoogi-keyboard-quit-test--root
  (file-name-directory (directory-file-name
                        (file-name-directory (or load-file-name buffer-file-name)))))

(ert-deftest imoogi-quit-cancels-minibuffer-from-another-window ()
  (save-window-excursion
    (delete-other-windows)
    (let* ((origin (selected-window))
           (input (split-window-below))
           called)
      (cl-letf (((symbol-function 'active-minibuffer-window) (lambda () input))
                ((symbol-function 'abort-minibuffers)
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

(ert-deftest imoogi-quit-works-without-delete-selection-mode ()
  "C-g must not depend on another module having loaded `delsel'."
  (let* ((keys (expand-file-name "modules/general/01-keys.el"
                                 imoogi-keyboard-quit-test--root))
         (form `(progn
                  (defun imoogi-require (&rest _) nil)
                  (load ,keys nil t)
                  (princ (if (featurep 'delsel) "DELSEL-LOADED" "NO-DELSEL"))
                  (princ " ")
                  (cl-letf (((symbol-function 'active-minibuffer-window)
                             (lambda () (selected-window))))
                    (princ (condition-case err
                               (progn (imoogi-keyboard-quit) "RETURNED")
                             (void-function (format "VOID:%S" (cadr err)))
                             (quit "QUIT")
                             (error "ERROR"))))))
         (output (with-temp-buffer
                   (call-process (expand-file-name invocation-name invocation-directory)
                                 nil t nil "-Q" "--batch" "-l" "cl-lib"
                                 "--eval" (prin1-to-string form))
                   (buffer-string))))
    (should (string-match-p "NO-DELSEL" output))
    (should-not (string-match-p "VOID:" output))))
