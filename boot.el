;;; boot.el --- imoogi-emacs entry point -*- lexical-binding: t; -*-

;;; Package repositories
(require 'package)
(setq package-archives '(("melpa" . "https://melpa.org/packages/")
                         ("gnu"   . "https://elpa.gnu.org/packages/")
                         ("nongnu" . "https://elpa.nongnu.org/nongnu/")))
(package-initialize)
(unless package-archive-contents
  (package-refresh-contents))

;;; use-package
(unless (package-installed-p 'use-package)
  (package-install 'use-package))
(require 'use-package)
(setq use-package-always-ensure t)

;;; straight.el bootstrap
(defvar bootstrap-version)
(let ((bootstrap-file
       (expand-file-name "straight/repos/straight.el/bootstrap.el"
                         (or (bound-and-true-p straight-base-dir) user-emacs-directory)))
      (bootstrap-version 7))
  (unless (file-exists-p bootstrap-file)
    (with-current-buffer
        (url-retrieve-synchronously
         "https://raw.githubusercontent.com/radian-software/straight.el/develop/install.el"
         'silent 'inhibit-cookies)
      (goto-char (point-max))
      (eval-print-last-sexp)))
  (load bootstrap-file nil 'nomessage))
(straight-use-package 'use-package)

;;; imoogi-emacs directory
(defvar imoogi-emacs-dir
  (file-name-directory (or load-file-name buffer-file-name))
  "Root directory of imoogi-emacs configuration.")

;;; Load modules
(dolist (module '("completion"
                  "which-key"
                  "projects"
                  "hydra"
                  "git"
                  "keys"
                  "treemacs"
                  "obsidian"))
  (load (expand-file-name (concat "modules/" module) imoogi-emacs-dir)))

(provide 'imoogi-boot)
;;; boot.el ends here
