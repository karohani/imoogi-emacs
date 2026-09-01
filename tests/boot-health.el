;;; boot-health.el --- shared boot health assertions -*- lexical-binding: t; -*-

;;; Code:

(require 'cl-lib)
(require 'package)
(require 'seq)
(require 'subr-x)

(defvar imoogi-boot-health-root
  (file-name-directory
   (directory-file-name
    (file-name-directory (or load-file-name buffer-file-name default-directory))))
  "Root directory of the imoogi-emacs checkout.")

(defvar imoogi-boot-health-packages-file nil
  "Override for packages.el used by tests.")

(defvar imoogi-boot-health-lock-file nil
  "Override for packages.lock used by tests.")

(defvar imoogi-boot-health-vendor-dir nil
  "Override for vendor/elpa directory used by tests.")

(defconst imoogi-boot-health--unset :unset
  "Sentinel meaning no test fixture was injected.")

(defvar imoogi-boot-health-required-packages imoogi-boot-health--unset
  "Injected top-level package list used by tests.")

(defvar imoogi-boot-health-lock-packages imoogi-boot-health--unset
  "Injected locked package names used by tests.")

(defvar imoogi-boot-health-vendor-packages imoogi-boot-health--unset
  "Injected vendored package names used by tests.")

(defvar imoogi-boot-health-module-files
  '("modules/04-projects.el" "modules/07-treemacs.el")
  "Module files scanned for removed Projectile-family package tokens.")

(defvar imoogi-boot-health-module-source imoogi-boot-health--unset
  "Injected module source map used by tests.
Each entry is (FILE . SOURCE).")

(defvar imoogi-boot-health-required-features '(imoogi-projects imoogi-treemacs)
  "Features that must be provided after a healthy full boot.")

(defvar imoogi-boot-health--errors nil
  "Error-level `display-warning' calls captured during boot.")

(defvar imoogi-boot-health--capturing nil
  "Non-nil when warning capture advice is installed.")

(defconst imoogi-boot-health-projectile-family
  '(projectile persp-projectile treemacs-projectile)
  "Projectile-family packages intentionally removed from this configuration.")

(defun imoogi-boot-health--capture-warning (type message &optional level &rest _)
  "Capture error-level warnings of TYPE and MESSAGE at LEVEL."
  (when (memq (or level :warning) '(:error :emergency))
    (push (cons type message) imoogi-boot-health--errors)))

(defun imoogi-boot-health-start-capture ()
  "Start collecting boot-time error warnings."
  (setq imoogi-boot-health--errors nil)
  (unless imoogi-boot-health--capturing
    (advice-add 'display-warning :before #'imoogi-boot-health--capture-warning)
    (setq imoogi-boot-health--capturing t)))

(defun imoogi-boot-health-stop-capture ()
  "Stop collecting boot-time error warnings."
  (when imoogi-boot-health--capturing
    (advice-remove 'display-warning #'imoogi-boot-health--capture-warning)
    (setq imoogi-boot-health--capturing nil)))

(defun imoogi-boot-health--packages-file ()
  "Return the package manifest path."
  (or imoogi-boot-health-packages-file
      (expand-file-name "packages.el" imoogi-boot-health-root)))

(defun imoogi-boot-health--lock-file ()
  "Return the lock file path."
  (or imoogi-boot-health-lock-file
      (expand-file-name "packages.lock" imoogi-boot-health-root)))

(defun imoogi-boot-health--vendor-dir ()
  "Return the vendored ELPA directory."
  (or imoogi-boot-health-vendor-dir
      (expand-file-name "vendor/elpa" imoogi-boot-health-root)))

(defun imoogi-boot-health--read-required-packages (file)
  "Read `imoogi-required-packages' from FILE without evaluating it."
  (with-temp-buffer
    (insert-file-contents file)
    (goto-char (point-min))
    (let ((found nil)
          (result nil)
          done)
      (while (not done)
        (let ((read-start (point)))
          (condition-case err
              (let ((form (read (current-buffer))))
                (when (and (consp form)
                           (eq (car form) 'defvar)
                           (eq (cadr form) 'imoogi-required-packages))
                  (setq found t)
                  (let ((value (caddr form)))
                    (setq result
                          (if (and (consp value) (eq (car value) 'quote))
                              (cadr value)
                            value)))))
            (end-of-file
             (goto-char read-start)
             (with-syntax-table emacs-lisp-mode-syntax-table
               (forward-comment (point-max)))
             (if (= (point) (point-max))
                 (setq done t)
               (signal (car err) (cdr err)))))))
      (unless found
        (error "packages.el 에 imoogi-required-packages defvar 가 없습니다: packages.el 을 확인하세요"))
      (unless (listp result)
        (error "packages.el 의 imoogi-required-packages 값은 패키지 심볼 목록이어야 합니다"))
      (unless result
        (error "packages.el 의 imoogi-required-packages 목록이 비어 있습니다: 망분리 배포용 manifest 를 복구하세요"))
      result)))

(defun imoogi-boot-health-required-packages ()
  "Return the top-level package manifest."
  (if (eq imoogi-boot-health-required-packages imoogi-boot-health--unset)
      (imoogi-boot-health--read-required-packages
       (imoogi-boot-health--packages-file))
    imoogi-boot-health-required-packages))

(defun imoogi-boot-health--read-lock-packages (file)
  "Read package names from packages.lock FILE."
  (with-temp-buffer
    (insert-file-contents file)
    (let (packages)
      (dolist (line (split-string (buffer-string) "\n" t))
        (unless (or (string-match-p "\\`[[:space:]]*\\(?:;;\\|#\\|\\'\\)" line)
                    (string-match-p "\\`[[:space:]]*NAME[[:space:]]+" line))
          (let ((fields (split-string line "[[:space:]]+" t)))
            (when fields
              (push (intern (car fields)) packages)))))
      (nreverse packages))))

(defun imoogi-boot-health-lock-packages ()
  "Return package names recorded in packages.lock."
  (if (eq imoogi-boot-health-lock-packages imoogi-boot-health--unset)
      (imoogi-boot-health--read-lock-packages
       (imoogi-boot-health--lock-file))
    imoogi-boot-health-lock-packages))

(defun imoogi-boot-health--entry-package-name (entry)
  "Return package name represented by vendor ENTRY, or nil."
  (when (string-match
         "\\`\\(.+\\)-[0-9][[:alnum:]._-]*\\(?:\\.signed\\)?\\'" entry)
    (intern (match-string 1 entry))))

(defun imoogi-boot-health--read-vendor-packages (dir)
  "Read package names present under vendored ELPA DIR."
  (delete-dups
   (delq nil
         (mapcar #'imoogi-boot-health--entry-package-name
                 (directory-files dir nil "\\`[^.]")))))

(defun imoogi-boot-health-vendor-packages ()
  "Return package names available in vendor/elpa."
  (if (eq imoogi-boot-health-vendor-packages imoogi-boot-health--unset)
      (imoogi-boot-health--read-vendor-packages
       (imoogi-boot-health--vendor-dir))
    imoogi-boot-health-vendor-packages))

(defun imoogi-boot-health--module-source (file)
  "Return source text for FILE."
  (if (eq imoogi-boot-health-module-source imoogi-boot-health--unset)
      (with-temp-buffer
        (insert-file-contents (expand-file-name file imoogi-boot-health-root))
        (buffer-string))
    (let ((entry (assoc file imoogi-boot-health-module-source)))
      (if entry
          (or (cdr entry) "")
        ""))))

(defun imoogi-boot-health--symbol-token-present-p (symbol source)
  "Return non-nil when SYMBOL appears as a Lisp symbol token in SOURCE."
  (let ((name (regexp-quote (symbol-name symbol)))
        (source (or source "")))
    (or (string-match-p (format "\\_<%s\\_>" name) source)
        (string-match-p (format "'%s\\_>" name) source))))

(defun imoogi-boot-health--built-in-package-p (pkg)
  "Return non-nil when PKG is supplied by Emacs itself."
  (and (fboundp 'package-built-in-p)
       (package-built-in-p pkg)))

(defun imoogi-boot-health--expected-skips ()
  "Return exact expected module skip list from the environment."
  (sort (split-string (or (getenv "IMOOGI_EXPECTED_SKIPS") "") "[, ]+" t)
        #'string<))

(defun imoogi-boot-health-issues (&optional expected-skips)
  "Return boot and package-integrity issues.
EXPECTED-SKIPS, when non-nil, overrides `IMOOGI_EXPECTED_SKIPS'."
  (let* ((expected (sort (copy-sequence (or expected-skips
                                           (imoogi-boot-health--expected-skips)))
                         #'string<))
         (actual (if (boundp 'imoogi-failed-modules)
                     (sort (copy-sequence (bound-and-true-p imoogi-failed-modules))
                           #'string<)
                   :missing-instrumentation))
         (required (imoogi-boot-health-required-packages))
         (external-required
          (cl-remove-if #'imoogi-boot-health--built-in-package-p required))
         (locked (imoogi-boot-health-lock-packages))
         (vendored (imoogi-boot-health-vendor-packages))
         (issues nil))
    (if (eq actual :missing-instrumentation)
        (push "boot.el 계측 없음: imoogi-failed-modules 가 정의되지 않았습니다" issues)
      (dolist (module (cl-set-difference actual expected :test #'string=))
        (push (format "예상하지 못한 모듈 스킵: %s" module) issues))
      (dolist (module (cl-set-difference expected actual :test #'string=))
        (push (format "스킵될 것으로 예상했으나 정상 로드됨: %s" module) issues)))
    (dolist (entry (reverse imoogi-boot-health--errors))
      (unless (eq (car entry) 'imoogi)
        (push (format "패키지 수준 오류 (%s): %s" (car entry) (cdr entry))
              issues)))
    (dolist (pkg external-required)
      (unless (memq pkg locked)
        (push (format "packages.lock 누락: %s" pkg) issues))
      (unless (memq pkg vendored)
        (push (format "vendor/elpa 누락: %s" pkg) issues)))
    (dolist (pkg imoogi-boot-health-projectile-family)
      (when (memq pkg required)
        (push (format "제거된 Projectile 계열 패키지가 manifest 에 남아 있음: %s" pkg)
              issues)))
    (dolist (pkg imoogi-boot-health-projectile-family)
      (when (memq pkg locked)
        (push (format "제거된 Projectile 계열 패키지가 packages.lock 에 남아 있음: %s" pkg)
              issues))
      (when (memq pkg vendored)
        (push (format "제거된 Projectile 계열 패키지가 vendor/elpa 에 남아 있음: %s" pkg)
              issues))
      (dolist (file imoogi-boot-health-module-files)
        (when (imoogi-boot-health--symbol-token-present-p
               pkg (imoogi-boot-health--module-source file))
          (push (format "제거된 Projectile 계열 토큰이 %s 에 남아 있음: %s"
                        file pkg)
                issues))))
    (dolist (feature imoogi-boot-health-required-features)
      (unless (featurep feature)
        (push (format "필수 feature 미제공: %s" feature) issues)))
    (nreverse issues)))

(defun imoogi-boot-health-assert (&optional no-exit)
  "Assert that the just-loaded boot is healthy.
When NO-EXIT is non-nil, signal an error instead of exiting Emacs."
  (imoogi-boot-health-stop-capture)
  (let ((issues (imoogi-boot-health-issues)))
    (message "--- boot assertion ---")
    (message "emacs           : %s" emacs-version)
    (message "expected skips  : %s"
             (if-let ((expected (imoogi-boot-health--expected-skips)))
                 (string-join expected ", ")
               "(none)"))
    (message "actual skips    : %s"
             (cond
              ((not (boundp 'imoogi-failed-modules)) "(unknown)")
              ((bound-and-true-p imoogi-failed-modules)
               (string-join (sort (copy-sequence imoogi-failed-modules) #'string<)
                            ", "))
              (t "(none)")))
    (dolist (issue issues)
      (message "FAIL: %s" issue))
    (if issues
        (if no-exit
            (error "boot health failed: %s" (string-join issues "; "))
          (message "RESULT: FAIL")
          (kill-emacs 1))
      (message "RESULT: PASS")
      (unless no-exit
        (kill-emacs 0)))))

(provide 'imoogi-boot-health)
;;; boot-health.el ends here
