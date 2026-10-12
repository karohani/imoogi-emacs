;;; lsp-jvm-test.el --- Java 가 없는 머신의 JVM 언어 서버 동작 테스트 -*- lexical-binding: t; -*-

;; jdtls·kotlin-language-server 는 대상 머신의 Java 로 돈다(JVM 은 동봉하지
;; 않는다). Java 가 없으면 서버를 띄우지 않고 이유를 한 번 알려야 하며,
;; 부팅에는 아무 영향이 없어야 한다. 판정은 번들 런처의
;; --imoogi-check-java 에 맡긴다 — Java 를 찾는 규칙이 런처 한 곳에만 있다.

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'eglot)

(defconst imoogi-test-jvm--missing-java-message
  "jdtls needs Java 21 or newer, but no Java runtime was found. Install a JDK or set JAVA_HOME.")

(defun imoogi-test-jvm--write-server (dir command check-exit &optional message)
  "Write COMMAND into DIR whose --imoogi-check-java exits CHECK-EXIT.
MESSAGE, when non-nil, is printed to stderr by the check.  Any other
invocation writes a marker file next to the script, so a test can tell
whether the server itself was started."
  (make-directory dir t)
  (let ((path (expand-file-name command dir)))
    (with-temp-file path
      (insert "#!/bin/sh\n"
              "if [ \"${1:-}\" = --imoogi-check-java ]; then\n"
              (if message (format "  echo '%s' >&2\n" message) "")
              (format "  exit %d\nfi\n" check-exit)
              (format "touch '%s.started'\n" path)))
    (set-file-modes path #o755)
    path))

(defun imoogi-test-jvm--activated-root ()
  "Return a temp repo root whose .local/bin is an active toolchain link."
  (let* ((root (make-temp-file "imoogi-jvm-root-" t))
         (bin (expand-file-name ".local/toolchains/2026.10.12.1/bin/" root)))
    (make-directory bin t)
    (make-symbolic-link "toolchains/2026.10.12.1/bin"
                        (expand-file-name ".local/bin" root))
    root))

(defmacro imoogi-test-jvm--with-env (root extra-path &rest body)
  "Run BODY with ROOT as the repo, its local bin then EXTRA-PATH on exec-path.
Eglot starts and warnings are recorded in `started' and `warnings'."
  (declare (indent 2))
  `(let* ((imoogi-emacs-dir (file-name-as-directory ,root))
          (exec-path (append (when (imoogi-lsp-local-bin-dir)
                               (list (imoogi-lsp-local-bin-dir)))
                             (when ,extra-path (list ,extra-path))
                             '("/usr/bin" "/bin")))
          (imoogi-jvm--preflight-results nil)
          (started 0)
          (warnings nil))
     (cl-letf (((symbol-function 'eglot-ensure)
                (lambda () (setq started (1+ started))))
               ((symbol-function 'display-warning)
                (lambda (_type message &rest _)
                  (push message warnings))))
       ,@body)))

(defun imoogi-test-jvm--open (mode file)
  "Turn on MODE in a temp buffer visiting FILE."
  (with-temp-buffer
    (setq buffer-file-name file)
    (funcall mode)))

(ert-deftest imoogi-jvm-missing-java-skips-eglot-and-warns-once ()
  "번들 런처가 Java 없음을 보고하면 Eglot 을 시작하지 않고 경고는 한 번만 낸다."
  (let* ((root (imoogi-test-jvm--activated-root))
         (bin (expand-file-name ".local/bin/" root))
         (server (imoogi-test-jvm--write-server
                  bin "jdtls" 69 imoogi-test-jvm--missing-java-message)))
    (imoogi-test-jvm--with-env root nil
      (imoogi-test-jvm--open #'java-mode "/tmp/ImoogiJvmTest.java")
      (imoogi-test-jvm--open #'java-mode "/tmp/ImoogiJvmTest2.java")
      (should (= started 0))
      (should-not (file-exists-p (concat server ".started")))
      (should (= (length warnings) 1))
      (should (string-match-p "jdtls" (car warnings)))
      (should (string-match-p (regexp-quote imoogi-test-jvm--missing-java-message)
                              (car warnings))))))

(ert-deftest imoogi-jvm-present-java-starts-eglot ()
  "Java 검사가 통과하면 Java·Kotlin 버퍼 모두 Eglot 을 시작한다."
  (let* ((root (imoogi-test-jvm--activated-root))
         (bin (expand-file-name ".local/bin/" root)))
    (imoogi-test-jvm--write-server bin "jdtls" 0)
    (imoogi-test-jvm--write-server bin "kotlin-language-server" 0)
    (imoogi-test-jvm--with-env root nil
      (imoogi-test-jvm--open #'java-mode "/tmp/ImoogiJvmTest.java")
      (imoogi-test-jvm--open #'kotlin-mode "/tmp/imoogi-jvm-test.kt")
      (should (= started 2))
      (should-not warnings))))

(ert-deftest imoogi-jvm-kotlin-missing-java-skips-eglot ()
  "Kotlin 도 같은 규칙: Java 가 없으면 시작하지 않고 경고한다."
  (let* ((root (imoogi-test-jvm--activated-root))
         (bin (expand-file-name ".local/bin/" root)))
    (imoogi-test-jvm--write-server
     bin "kotlin-language-server" 69
     "kotlin-language-server needs Java 11 or newer, but no Java runtime was found.")
    (imoogi-test-jvm--with-env root nil
      (imoogi-test-jvm--open #'kotlin-mode "/tmp/imoogi-jvm-test.kt")
      (should (= started 0))
      (should (= (length warnings) 1))
      (should (string-match-p "kotlin-language-server needs Java 11" (car warnings))))))

(ert-deftest imoogi-jvm-external-server-is-not-preflighted ()
  "번들 밖(사용자가 직접 설치한) 서버에는 비공개 플래그를 보내지 않는다.
외부 jdtls 는 --imoogi-check-java 를 모르므로, 보내면 서버가 그대로 떠 버린다."
  (let* ((root (make-temp-file "imoogi-jvm-root-" t))
         (external (make-temp-file "imoogi-jvm-external-" t))
         (server (imoogi-test-jvm--write-server
                  external "jdtls" 69 imoogi-test-jvm--missing-java-message)))
    (imoogi-test-jvm--with-env root external
      (should-not (imoogi-lsp-local-bin-dir))
      (imoogi-test-jvm--open #'java-mode "/tmp/ImoogiJvmTest.java")
      (should (= started 1))
      (should-not warnings)
      (should-not (file-exists-p (concat server ".started"))))))

(ert-deftest imoogi-jvm-reset-reruns-preflight ()
  "Java 를 설치한 뒤 캐시를 비우면 재시작 없이 다시 판정한다."
  (let* ((root (imoogi-test-jvm--activated-root))
         (bin (expand-file-name ".local/bin/" root)))
    (imoogi-test-jvm--write-server bin "jdtls" 69 imoogi-test-jvm--missing-java-message)
    (imoogi-test-jvm--with-env root nil
      (imoogi-test-jvm--open #'java-mode "/tmp/ImoogiJvmTest.java")
      (should (= started 0))
      (imoogi-test-jvm--write-server bin "jdtls" 0)
      (imoogi-jvm-reset-preflight)
      (imoogi-test-jvm--open #'java-mode "/tmp/ImoogiJvmTest.java")
      (should (= started 1)))))

(ert-deftest imoogi-jvm-loading-language-files-runs-no-process ()
  "Java/Kotlin 설정을 다시 로드해도 프로세스를 띄우지 않는다(부팅 무영향)."
  (cl-letf (((symbol-function 'call-process)
             (lambda (&rest _) (error "process started at load time")))
            ((symbol-function 'process-file)
             (lambda (&rest _) (error "process started at load time")))
            ((symbol-function 'make-process)
             (lambda (&rest _) (error "process started at load time"))))
    (dolist (file '("jvm" "java" "kotlin"))
      (load (expand-file-name (concat "modules/development/lang/" file)
                              imoogi-test-root)
            nil t))))

(provide 'lsp-jvm-test)
;;; lsp-jvm-test.el ends here
