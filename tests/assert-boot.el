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

(load (expand-file-name
       "boot-health.el"
       (file-name-directory (or load-file-name buffer-file-name)))
      nil t)

(declare-function imoogi-boot-health-start-capture "boot-health")
(declare-function imoogi-boot-health-assert "boot-health")

(imoogi-boot-health-start-capture)

(defun imoogi-assert--load-entry (file)
  (let ((path (expand-file-name file "~/.emacs.d/")))
    (unless (file-readable-p path)
      (message "FAIL: 설치 진입점 없음 — %s (install.sh 를 실행했나요?)" path)
      (kill-emacs 1))
    (load path nil t)))

(imoogi-assert--load-entry "early-init.el")
(imoogi-assert--load-entry "init.el")

(imoogi-boot-health-assert)

;;; assert-boot.el ends here
