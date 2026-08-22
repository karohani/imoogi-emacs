;;; assert-boot.el --- assert a real installed boot is healthy -*- lexical-binding: t; -*-

;; Drives the entry points that scripts/install.sh wired into ~/.emacs.d and
;; then judges the result. Exit code is the verdict.
;;
;;   emacs --batch -l tests/assert-boot.el
;;
;; Environment:
;;   IMOOGI_EXPECTED_SKIPS  comma-separated module names allowed to be skipped
;;                          (unset/empty = none allowed). Matching is EXACT in
;;                          both directions: an unexpected skip fails, and an
;;                          expected skip that did not happen also fails.
;;
;; Why not just check Emacs' exit code: boot.el deliberately isolates module
;; failures so one bad module cannot take the session down. That makes a
;; half-broken install exit 0 — observed both with git absent (06-git,
;; 07-treemacs skipped) and on Emacs 29 against 30.x .elc files. The exit code
;; is therefore not a usable signal; the two checks below are.

;;; Code:

(require 'cl-lib)

(defvar imoogi-assert--errors nil
  "Error-level `display-warning' calls captured during boot.")

;; Installed before boot so package-level failures are captured too. use-package
;; reports its own failures through display-warning (`Error (use-package): ...'),
;; and those do NOT mark the module as failed — so imoogi-failed-modules alone
;; would under-report. Both signals are needed.
(defun imoogi-assert--capture (type message &optional level &rest _)
  (when (memq (or level :warning) '(:error :emergency))
    (push (cons type message) imoogi-assert--errors)))
(advice-add 'display-warning :before #'imoogi-assert--capture)

(defun imoogi-assert--load-entry (file)
  (let ((path (expand-file-name file "~/.emacs.d/")))
    (unless (file-readable-p path)
      (message "FAIL: 설치 진입점 없음 — %s (install.sh 를 실행했나요?)" path)
      (kill-emacs 1))
    (load path nil t)))

(imoogi-assert--load-entry "early-init.el")
(imoogi-assert--load-entry "init.el")

(advice-remove 'display-warning #'imoogi-assert--capture)

;; Without the boot.el instrumentation an unbound variable would read as "no
;; modules skipped" and every run would pass — the exact false-pass this file
;; exists to prevent. Fail loudly instead.
(unless (boundp 'imoogi-failed-modules)
  (message "FAIL: boot.el 계측 없음 — imoogi-failed-modules 가 정의되지 않았습니다")
  (message "RESULT: FAIL")
  (kill-emacs 1))

(let* ((raw (or (getenv "IMOOGI_EXPECTED_SKIPS") ""))
       (expected (sort (split-string raw "[, ]+" t) #'string<))
       (actual (sort (copy-sequence (bound-and-true-p imoogi-failed-modules))
                     #'string<))
       (unexpected (cl-set-difference actual expected :test #'string=))
       (missing (cl-set-difference expected actual :test #'string=))
       ;; imoogi-typed errors are the module skips, already judged exactly by
       ;; the set comparison above. Anything else is an unjudged failure.
       (foreign (cl-remove-if (lambda (e) (eq (car e) 'imoogi))
                              imoogi-assert--errors))
       (failed nil))

  (message "--- boot assertion ---")
  (message "emacs           : %s" emacs-version)
  (message "expected skips  : %s" (if expected (string-join expected ", ") "(none)"))
  (message "actual skips    : %s" (if actual (string-join actual ", ") "(none)"))

  (when unexpected
    (setq failed t)
    (message "FAIL: 예상하지 못한 모듈 스킵: %s" (string-join unexpected ", ")))
  (when missing
    (setq failed t)
    (message "FAIL: 스킵될 것으로 예상했으나 정상 로드됨: %s"
             (string-join missing ", ")))
  (dolist (entry (reverse foreign))
    (setq failed t)
    (message "FAIL: 패키지 수준 오류 (%s): %s" (car entry) (cdr entry)))

  (if failed
      (progn (message "RESULT: FAIL") (kill-emacs 1))
    (message "RESULT: PASS")
    (kill-emacs 0)))

;;; assert-boot.el ends here
