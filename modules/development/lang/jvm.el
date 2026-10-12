;;; jvm.el --- Java runtime gate for JVM-hosted language servers -*- lexical-binding: t; -*-

;; jdtls and kotlin-language-server run on the Java the machine already has —
;; no JVM is vendored.  When Java is missing (or too old) the server must not
;; start, and the user gets one clear warning instead of a crashed server.
;;
;; The decision belongs to the bundled launcher: `<server> --imoogi-check-java'
;; exits 0, or exits non-zero with the reason on stderr.  Java lookup rules
;; therefore live in one place (the launcher) rather than being re-implemented
;; here.  The check runs lazily — the first time a buffer of that language
;; opens — so startup never spawns a process.
;;
;; Only servers inside the active toolchain bundle are checked.  A server the
;; user installed elsewhere does not know the private flag and would simply
;; start, so it is trusted as-is.

;;; Code:

(require 'seq)

(declare-function imoogi-lsp-local-bin-dir "17-lsp")
(declare-function imoogi-eglot-server-available-p "17-lsp")
(declare-function eglot-ensure "eglot")

(defvar imoogi-jvm--preflight-results nil
  "Alist of (COMMAND . (OK . MESSAGE)) for servers already checked.")

(defun imoogi-jvm-reset-preflight ()
  "Forget earlier Java checks, e.g. after installing a JDK."
  (interactive)
  (setq imoogi-jvm--preflight-results nil))

(defun imoogi-jvm--bundled-server (command)
  "Return the path of COMMAND when it resolves inside the toolchain bundle."
  (let ((path (executable-find command))
        (local-bin (imoogi-lsp-local-bin-dir)))
    (when (and path local-bin
               (file-in-directory-p (file-truename path)
                                    (file-truename local-bin)))
      path)))

(defun imoogi-jvm--preflight (path)
  "Run PATH --imoogi-check-java; return (OK . MESSAGE)."
  (with-temp-buffer
    (let* ((default-directory temporary-file-directory)
           (status (condition-case err
                       (call-process path nil t nil "--imoogi-check-java")
                     (error (error-message-string err)))))
      (cons (eq status 0)
            (string-trim (if (stringp status)
                             status
                           (buffer-string)))))))

(defun imoogi-jvm-server-ready-p (command)
  "Return non-nil when COMMAND may start; warn once when it may not."
  (let ((cached (assoc command imoogi-jvm--preflight-results)))
    (if cached
        (cadr cached)
      (let* ((path (imoogi-jvm--bundled-server command))
             (result (if path (imoogi-jvm--preflight path) (cons t ""))))
        (push (cons command result) imoogi-jvm--preflight-results)
        (unless (car result)
          (display-warning
           'imoogi-lsp
           (format "%s 를 시작하지 않았습니다 — %s\n(JDK 설치 후 M-x imoogi-jvm-reset-preflight 로 다시 확인)"
                   command
                   (if (string-empty-p (cdr result))
                       "Java 실행 환경을 확인하지 못했습니다."
                     (cdr result)))
           :warning))
        (car result)))))

(defun imoogi-jvm-ensure-eglot (command)
  "Start Eglot for COMMAND when it exists and its Java check passes."
  (when (and (imoogi-eglot-server-available-p command)
             (imoogi-jvm-server-ready-p command)
             (require 'eglot nil t))
    (eglot-ensure)))

(provide 'imoogi-lsp-jvm)
;;; jvm.el ends here
