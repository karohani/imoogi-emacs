;;; audit.el --- Org-level audit of a rendered clip -*- lexical-binding: t; -*-
;;
;; Usage: emacs --batch -Q -l audit.el FILE EXPECTED-HEADLINES
;;
;; Loads FILE the way a user would (find-file, so the mode line and file-local
;; variables are processed with every safety prompt pre-answered "yes"), then
;; parses, opens every "[[" position, runs every babel block and exports the
;; buffer.  Any execution path is recorded in `org-clipper-pwned'.  Prints
;; AUDIT-OK on success, otherwise the list of problems, and exits non-zero.

(require 'org)
(require 'org-element)
(require 'ob)
(require 'ox)
(require 'ox-ascii)

(defvar org-clipper-pwned nil)
(defun org-clipper--pwn (what &rest _)
  (push what org-clipper-pwned)
  nil)

;; Answer every confirmation with "no" but record that it was asked: being
;; asked at all means an executable construct survived rendering.
(fset 'yes-or-no-p (lambda (&rest _) (org-clipper--pwn 'confirm-prompt) nil))
(fset 'y-or-n-p (lambda (&rest _) (org-clipper--pwn 'confirm-prompt) nil))
(setq org-link-shell-confirm-function (lambda (&rest _) (org-clipper--pwn 'shell-link) nil)
      org-link-elisp-confirm-function (lambda (&rest _) (org-clipper--pwn 'elisp-link) nil)
      org-confirm-babel-evaluate (lambda (&rest _) (org-clipper--pwn 'babel) t)
      org-export-use-babel t
      browse-url-browser-function (lambda (&rest _) nil)
      enable-local-variables :all
      enable-local-eval t
      safe-local-variable-values nil)
(org-babel-do-load-languages 'org-babel-load-languages '((emacs-lisp . t) (shell . t)))

(let* ((file (nth 0 command-line-args-left))
       (want-headlines (string-to-number (nth 1 command-line-args-left)))
       (problems nil)
       (forbidden '(babel-call inline-babel-call inline-src-block src-block macro
                    drawer property-drawer fixed-width diary-sexp latex-fragment
                    latex-environment export-snippet export-block target
                    radio-target footnote-reference footnote-definition planning
                    clock comment comment-block dynamic-block timestamp
                    citation inlinetask node-property)))
  (setq command-line-args-left nil)
  (find-file file)
  (unless (eq major-mode 'org-mode)
    (push (format "major mode is %s" major-mode) problems))
  (when file-local-variables-alist
    (push (format "file-local variables applied: %S" file-local-variables-alist) problems))
  (when (hack-local-variables-prop-line)
    (push "mode-line variables present" problems))
  (let ((tree (org-element-parse-buffer)))
    (org-element-map tree forbidden
      (lambda (el) (push (format "forbidden element %s at %d" (org-element-type el)
                                 (org-element-property :begin el))
                         problems)))
    (org-element-map tree 'keyword
      (lambda (k)
        (unless (member (org-element-property :key k) '("TITLE" "DATE" "SOURCE"))
          (push (format "keyword %s" (org-element-property :key k)) problems))))
    (org-element-map tree 'link
      (lambda (l)
        (let ((type (org-element-property :type l))
              (path (org-element-property :path l)))
          (unless (or (member type '("http" "https"))
                      (and (equal type "file") (string-match-p "\\`assets/[0-9a-f]\\{32\\}\\.\\(png\\|jpg\\|webp\\)\\'" path)))
            (push (format "link %s:%s" type path) problems)))))
    (let ((n (length (org-element-map tree 'headline #'identity))))
      (unless (= n want-headlines)
        (push (format "headlines %d, want %d" n want-headlines) problems))))
  ;; Try to follow every bracket-looking position that is not an allowed link.
  (goto-char (point-min))
  (while (search-forward "[[" nil t)
    (save-excursion
      (goto-char (match-beginning 0))
      (let ((ctx (org-element-context)))
        (unless (and (eq (org-element-type ctx) 'link)
                     (member (org-element-property :type ctx) '("http" "https" "file")))
          (ignore-errors (org-open-at-point))))))
  (ignore-errors (org-babel-execute-buffer))
  (let ((exported (ignore-errors (org-export-as 'ascii))))
    (unless exported
      (push "export failed" problems))
    (when (and exported (string-match-p "INCLUDED-SECRET" exported))
      (push "INCLUDE was processed" problems)))
  (when org-clipper-pwned
    (push (format "executed: %S" org-clipper-pwned) problems))
  (if problems
      (progn (princ (format "AUDIT-FAIL %s\n" file))
             (dolist (p (nreverse problems)) (princ (format "  %s\n" p)))
             (kill-emacs 1))
    (princ "AUDIT-OK\n")
    (kill-emacs 0)))
