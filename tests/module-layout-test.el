;;; module-layout-test.el --- State preservation across module reload -*- lexical-binding: t; -*-
(require 'ert)
(require 'cl-lib)

(ert-deftest imoogi-module-reload-preserves-document-and-user-state ()
  "S2/S3: reload does not migrate user data or replace a working document."
  (let* ((imoogi-config-file "/tmp/user-selected-anki-config.json")
         (imoogi-project-notes-directory "/tmp/user-selected-project-notes/")
         (imoogi-lsp-language-config-dir (make-temp-file "imoogi-custom-lang-" t))
         (language-dir imoogi-lsp-language-config-dir))
    (unwind-protect
        (with-temp-buffer
          (org-mode)
          (setq buffer-file-name "/tmp/imoogi-unsaved-refactor.org")
          (insert "* TODO Preserve this\nUncommitted buffer text\n")
          (setq-local header-line-format "session state")
          (let ((content (buffer-string))
                (position (point)))
            (imoogi-reload)
            (should (equal content (buffer-string)))
            (should (= position (point)))
            (should (buffer-modified-p))
            (should (equal buffer-file-name "/tmp/imoogi-unsaved-refactor.org"))
            (should (equal header-line-format "session state"))
            (should (equal imoogi-config-file "/tmp/user-selected-anki-config.json"))
            (should (equal imoogi-project-notes-directory "/tmp/user-selected-project-notes/"))
            (should (equal imoogi-lsp-language-config-dir language-dir))
            (should-not imoogi-failed-modules)))
      (delete-directory language-dir t))))

(ert-deftest imoogi-module-reload-preserves-development-bindings-and-hooks ()
  "S4/S9: formatting and Git hooks still work without duplicate registration."
  (imoogi-reload)
  (dolist (key '("s-M-l" "C-M-\\"))
    (should (eq (lookup-key global-map (kbd key)) #'imoogi-format-code)))
  (should (fboundp 'imoogi-format-code))
  (dolist (row '((prog-mode-hook . diff-hl-mode)
                 (dired-mode-hook . diff-hl-dired-mode)
                 (magit-pre-refresh-hook . diff-hl-magit-pre-refresh)
                 (magit-post-refresh-hook . diff-hl-magit-post-refresh)
                 (text-mode-hook . visual-line-mode)))
    (should (= 1 (cl-count (cdr row) (symbol-value (car row)))))))


(ert-deftest imoogi-module-reload-preserves-numbered-load-order ()
  "S1: regrouping must preserve the dependency order of existing modules."
  (let ((expected '("00-defaults" "01-keys" "02-completion" "03-which-key"
                    "04-projects" "05-transient" "06-git" "07-treemacs"
                    "08-obsidian" "09-autorevert" "10-theme" "11-editing"
                    "12-navigation" "13-system" "14-org" "23-org-preview"
                    "15-markdown" "16-elisp" "17-lsp" "18-languages"
                    "19-folding" "20-terminal" "21-native-compile" "22-tabs"
                    "24-anki" "25-flashcards" "26-project-notes" "27-gptel"
                    "28-clipboard" "29-org-roam"))
        (original (symbol-function 'load))
        observed)
    (cl-letf (((symbol-function 'load)
               (lambda (file &rest args)
                 (when (and (stringp file)
                            (file-in-directory-p file
                                                 (expand-file-name "modules/" imoogi-test-root))
                            (string-match-p "\\`[0-9][0-9]-" (file-name-nondirectory file)))
                   (push (file-name-base file) observed))
                 (apply original file args))))
      (imoogi-reload))
    (should (equal (nreverse observed) expected))
    (should-not imoogi-failed-modules)))

(ert-deftest imoogi-module-reload-migrates-only-legacy-language-default ()
  "An existing session follows moved bundled languages without losing overrides."
  (let ((imoogi-lsp-language-config-dir
         (expand-file-name "modules/lsp/" imoogi-test-root)))
    (load (expand-file-name "modules/development/17-lsp.el" imoogi-test-root) nil t)
    (should (equal imoogi-lsp-language-config-dir
                   (expand-file-name "modules/development/lang/" imoogi-test-root)))
    (should (file-exists-p (expand-file-name "go.el" imoogi-lsp-language-config-dir)))))

(ert-deftest imoogi-module-reload-migrates-library-defaults-and-keeps-overrides ()
  "Existing Anki/flashcards sessions resolve source libraries at their new paths."
  (let* ((old-anki (expand-file-name "modules/anki/" imoogi-test-root))
         (old-flashcards (expand-file-name "modules/flashcards/" imoogi-test-root))
         (imoogi-anki-lisp-dir old-anki)
         (imoogi-flashcards-lisp-dir old-flashcards)
         (load-path (append (list old-anki old-flashcards) load-path)))
    (imoogi-reload)
    (should-not (member old-anki load-path))
    (should-not (member old-flashcards load-path))
    (should (equal imoogi-anki-lisp-dir
                   (expand-file-name "modules/org/anki/" imoogi-test-root)))
    (should (equal imoogi-flashcards-lisp-dir
                   (expand-file-name "modules/org/flashcards/" imoogi-test-root)))
    (should (string-prefix-p imoogi-anki-lisp-dir (locate-library "imoogi")))
    (let ((imoogi-anki-lisp-dir "/tmp/custom-anki-library/")
          (imoogi-flashcards-lisp-dir "/tmp/custom-flashcards-library/"))
      (imoogi-reload)
      (should (equal imoogi-anki-lisp-dir "/tmp/custom-anki-library/"))
      (should (equal imoogi-flashcards-lisp-dir "/tmp/custom-flashcards-library/")))))

(provide 'module-layout-test)
;;; module-layout-test.el ends here
