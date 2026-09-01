;;; boot-health-test.el --- tests for shared boot health oracle -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)

(eval-and-compile
  (let* ((current-file (or load-file-name
                           buffer-file-name
                           (and (boundp 'byte-compile-current-file)
                                byte-compile-current-file)))
         (sibling (and current-file
                       (expand-file-name "boot-health.el"
                                         (file-name-directory current-file))))
         (root (locate-dominating-file default-directory "tests/boot-health.el"))
         (fallback (and root (expand-file-name "tests/boot-health.el" root))))
    (load (or (and sibling (file-readable-p sibling) sibling)
              fallback)
          nil t)))

(defvar imoogi-failed-modules)

(defun imoogi-test-boot-health--has-issue-p (regexp issues)
  "Return non-nil when any string in ISSUES matches REGEXP."
  (seq-some (lambda (issue) (string-match-p regexp issue)) issues))

(defun imoogi-test-boot-health--read-required-from-string (source)
  "Read required packages from a temporary packages.el containing SOURCE."
  (let ((file (make-temp-file "imoogi-packages-" nil ".el")))
    (unwind-protect
        (progn
          (with-temp-file file
            (insert source))
          (imoogi-boot-health--read-required-packages file))
      (delete-file file))))

(ert-deftest imoogi-boot-health-required-packages-empty-file-errors ()
  (should-error
   (imoogi-test-boot-health--read-required-from-string "")
   :type 'error))

(ert-deftest imoogi-boot-health-required-packages-unrelated-forms-error ()
  (should-error
   (imoogi-test-boot-health--read-required-from-string
    "(defvar unrelated-packages '(project perspective))")
   :type 'error))

(ert-deftest imoogi-boot-health-required-packages-explicit-empty-defvar-errors ()
  (should-error
   (imoogi-test-boot-health--read-required-from-string
    "(defvar imoogi-required-packages nil)")
   :type 'error))

(ert-deftest imoogi-boot-health-required-packages-non-list-defvar-errors ()
  (should-error
   (imoogi-test-boot-health--read-required-from-string
    "(defvar imoogi-required-packages 'project)")
   :type 'error))

(ert-deftest imoogi-boot-health-required-packages-malformed-syntax-errors ()
  (should-error
   (imoogi-test-boot-health--read-required-from-string
    "(defvar imoogi-required-packages '(project perspective)")
   :type 'end-of-file))

(ert-deftest imoogi-boot-health-required-packages-valid-quoted-list-parses ()
  (should
   (equal
    (imoogi-test-boot-health--read-required-from-string
     "(defvar imoogi-required-packages '(project perspective treemacs))")
    '(project perspective treemacs))))

(ert-deftest imoogi-boot-health-clean-injected-state-has-no-issues ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors nil)
        (imoogi-boot-health-required-packages '(project perspective treemacs))
        (imoogi-boot-health-lock-packages '(perspective treemacs))
        (imoogi-boot-health-vendor-packages '(perspective treemacs))
        (imoogi-boot-health-module-source nil)
        (imoogi-boot-health-required-features nil))
    (should-not (imoogi-boot-health-issues))))

(ert-deftest imoogi-boot-health-nil-fixtures-do-not-read-checkout-files ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors nil)
        (imoogi-boot-health-packages-file "/no/such/packages.el")
        (imoogi-boot-health-lock-file "/no/such/packages.lock")
        (imoogi-boot-health-vendor-dir "/no/such/vendor/elpa")
        (imoogi-boot-health-required-packages nil)
        (imoogi-boot-health-lock-packages nil)
        (imoogi-boot-health-vendor-packages nil)
        (imoogi-boot-health-module-files '("modules/04-projects.el"))
        (imoogi-boot-health-module-source nil)
        (imoogi-boot-health-required-features nil))
    (should-not (imoogi-boot-health-issues))))

(ert-deftest imoogi-boot-health-module-source-nil-entry-is-empty-fixture ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors nil)
        (imoogi-boot-health-required-packages nil)
        (imoogi-boot-health-lock-packages nil)
        (imoogi-boot-health-vendor-packages nil)
        (imoogi-boot-health-module-files '("modules/04-projects.el"
                                           "modules/07-treemacs.el"))
        (imoogi-boot-health-module-source
         '(("modules/04-projects.el" . nil)))
        (imoogi-boot-health-required-features nil))
    (should-not (imoogi-boot-health-issues))))

(ert-deftest imoogi-boot-health-missing-lock-package-is-actionable ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors nil)
        (imoogi-boot-health-required-packages '(perspective treemacs))
        (imoogi-boot-health-lock-packages '(perspective))
        (imoogi-boot-health-vendor-packages '(perspective treemacs))
        (imoogi-boot-health-module-source nil)
        (imoogi-boot-health-required-features nil))
    (should (imoogi-test-boot-health--has-issue-p
             "packages\\.lock 누락: treemacs"
             (imoogi-boot-health-issues)))))

(ert-deftest imoogi-boot-health-missing-vendor-package-is-actionable ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors nil)
        (imoogi-boot-health-required-packages '(perspective treemacs))
        (imoogi-boot-health-lock-packages '(perspective treemacs))
        (imoogi-boot-health-vendor-packages '(perspective))
        (imoogi-boot-health-module-source nil)
        (imoogi-boot-health-required-features nil))
    (should (imoogi-test-boot-health--has-issue-p
             "vendor/elpa 누락: treemacs"
             (imoogi-boot-health-issues)))))

(ert-deftest imoogi-boot-health-unexpected-project-or-treemacs-skip-fails ()
  (let ((imoogi-failed-modules '("04-projects" "07-treemacs"))
        (imoogi-boot-health--errors '((imoogi . "module skip warning")))
        (imoogi-boot-health-required-packages nil)
        (imoogi-boot-health-lock-packages nil)
        (imoogi-boot-health-vendor-packages nil)
        (imoogi-boot-health-module-source nil)
        (imoogi-boot-health-required-features nil))
    (let ((issues (imoogi-boot-health-issues)))
      (should (imoogi-test-boot-health--has-issue-p
               "예상하지 못한 모듈 스킵: 04-projects" issues))
      (should (imoogi-test-boot-health--has-issue-p
               "예상하지 못한 모듈 스킵: 07-treemacs" issues)))))

(ert-deftest imoogi-boot-health-expected-imoogi-warning-is-judged-by-skip-set ()
  (let ((imoogi-failed-modules '("07-treemacs"))
        (imoogi-boot-health--errors '((imoogi . "module skip warning")))
        (imoogi-boot-health-required-packages nil)
        (imoogi-boot-health-lock-packages nil)
        (imoogi-boot-health-vendor-packages nil)
        (imoogi-boot-health-module-source nil)
        (imoogi-boot-health-required-features nil))
    (should-not (imoogi-boot-health-issues '("07-treemacs")))))

(ert-deftest imoogi-boot-health-foreign-error-warning-fails ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors '((use-package . "Cannot load package")))
        (imoogi-boot-health-required-packages nil)
        (imoogi-boot-health-lock-packages nil)
        (imoogi-boot-health-vendor-packages nil)
        (imoogi-boot-health-module-source nil)
        (imoogi-boot-health-required-features nil))
    (should (imoogi-test-boot-health--has-issue-p
             "패키지 수준 오류 (use-package): Cannot load package"
             (imoogi-boot-health-issues)))))

(ert-deftest imoogi-boot-health-projectile-family-is-rejected-in-manifest ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors nil)
        (imoogi-boot-health-required-packages '(projectile perspective))
        (imoogi-boot-health-lock-packages '(projectile perspective))
        (imoogi-boot-health-vendor-packages '(projectile perspective))
        (imoogi-boot-health-module-source nil)
        (imoogi-boot-health-required-features nil))
    (should (imoogi-test-boot-health--has-issue-p
             "제거된 Projectile 계열 패키지가 manifest 에 남아 있음: projectile"
             (imoogi-boot-health-issues)))))

(ert-deftest imoogi-boot-health-projectile-family-is-rejected-in-lock-only ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors nil)
        (imoogi-boot-health-required-packages '(perspective))
        (imoogi-boot-health-lock-packages '(perspective treemacs-projectile))
        (imoogi-boot-health-vendor-packages '(perspective))
        (imoogi-boot-health-module-source nil)
        (imoogi-boot-health-required-features nil))
    (should (imoogi-test-boot-health--has-issue-p
             "제거된 Projectile 계열 패키지가 packages\\.lock 에 남아 있음: treemacs-projectile"
             (imoogi-boot-health-issues)))))

(ert-deftest imoogi-boot-health-projectile-family-is-rejected-in-vendor-only ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors nil)
        (imoogi-boot-health-required-packages '(perspective))
        (imoogi-boot-health-lock-packages '(perspective))
        (imoogi-boot-health-vendor-packages '(perspective persp-projectile))
        (imoogi-boot-health-module-source nil)
        (imoogi-boot-health-required-features nil))
    (should (imoogi-test-boot-health--has-issue-p
             "제거된 Projectile 계열 패키지가 vendor/elpa 에 남아 있음: persp-projectile"
             (imoogi-boot-health-issues)))))

(ert-deftest imoogi-boot-health-projectile-family-is-rejected-in-module-source ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors nil)
        (imoogi-boot-health-required-packages '(perspective))
        (imoogi-boot-health-lock-packages '(perspective))
        (imoogi-boot-health-vendor-packages '(perspective))
        (imoogi-boot-health-module-source
         '(("modules/04-projects.el" . "(imoogi-require \"04-projects\" 'projectile)")
           ("modules/07-treemacs.el" . "(imoogi-require \"07-treemacs\" 'treemacs)")))
        (imoogi-boot-health-required-features nil))
    (should (imoogi-test-boot-health--has-issue-p
             "제거된 Projectile 계열 토큰이 modules/04-projects\\.el 에 남아 있음: projectile"
             (imoogi-boot-health-issues)))))

(ert-deftest imoogi-boot-health-projectile-family-is-rejected-across-mixed-boundaries ()
  (let ((imoogi-failed-modules nil)
        (imoogi-boot-health--errors nil)
        (imoogi-boot-health-required-packages '(persp-projectile perspective))
        (imoogi-boot-health-lock-packages '(projectile perspective))
        (imoogi-boot-health-vendor-packages '(perspective treemacs-projectile))
        (imoogi-boot-health-module-source
         '(("modules/04-projects.el" . "(imoogi-require \"04-projects\" 'project)")
           ("modules/07-treemacs.el" . "(require 'treemacs-projectile)")))
        (imoogi-boot-health-required-features nil))
    (let ((issues (imoogi-boot-health-issues)))
      (should (imoogi-test-boot-health--has-issue-p
               "manifest 에 남아 있음: persp-projectile" issues))
      (should (imoogi-test-boot-health--has-issue-p
               "packages\\.lock 에 남아 있음: projectile" issues))
      (should (imoogi-test-boot-health--has-issue-p
               "vendor/elpa 에 남아 있음: treemacs-projectile" issues))
      (should (imoogi-test-boot-health--has-issue-p
               "modules/07-treemacs\\.el 에 남아 있음: treemacs-projectile"
               issues)))))

(ert-deftest imoogi-boot-health-current-projectile-family-stays-absent ()
  (let ((required (imoogi-boot-health--read-required-packages
                   (expand-file-name "packages.el" imoogi-boot-health-root)))
        (locked (imoogi-boot-health--read-lock-packages
                 (expand-file-name "packages.lock" imoogi-boot-health-root)))
        (vendored (imoogi-boot-health--read-vendor-packages
                   (expand-file-name "vendor/elpa" imoogi-boot-health-root))))
    (dolist (pkg imoogi-boot-health-projectile-family)
      (should-not (memq pkg required))
      (should-not (memq pkg locked))
      (should-not (memq pkg vendored))))
  (dolist (file '("modules/04-projects.el" "modules/07-treemacs.el"))
    (let ((source (imoogi-boot-health--module-source file)))
      (dolist (pkg imoogi-boot-health-projectile-family)
        (should-not (imoogi-boot-health--symbol-token-present-p pkg source))))))

(provide 'imoogi-boot-health-test)
;;; boot-health-test.el ends here
