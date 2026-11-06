;;; toolchain-servers-test.el --- 반입된 LSP 서버 테스트 -*- lexical-binding: t; -*-

;; vendor/toolchains 에서 setup 으로 설치된 서버가 Emacs 에 실제로 보이는지
;; 확인한다. 설치는 플랫폼별이고 사용자가 setup 을 돌려야 생기므로, .local/bin
;; 이 활성화되지 않은 환경(컨테이너, 미설치 머신)에서는 건너뛴다.

;;; Code:

(require 'ert)

(ert-deftest imoogi-toolchain-servers-are-on-exec-path ()
  "활성 번들이 있으면 그 안의 서버들이 executable-find 로 잡힌다."
  (skip-unless (imoogi-lsp-local-bin-dir))
  (dolist (command '("gopls" "typescript-language-server" "basedpyright-langserver"
                     "jdtls" "kotlin-language-server" "clojure-lsp"))
    (let ((found (executable-find command)))
      (should found)
      ;; 시스템 어딘가가 아니라 저장소 번들에서 잡혀야 한다.
      (should (string-prefix-p (expand-file-name (imoogi-lsp-local-bin-dir))
                               (expand-file-name found))))))

(ert-deftest imoogi-toolchain-python-server-registers-for-eglot ()
  "basedpyright 가 있으면 eglot 의 python 후보 목록에 그 이름이 있다."
  (skip-unless (executable-find "basedpyright-langserver"))
  (require 'eglot)
  (should (seq-find (lambda (entry)
                      (string-match-p "basedpyright-langserver" (format "%S" entry)))
                    eglot-server-programs)))

;; clojure-lsp 은 네이티브 빌드라 Java 없이도 돈다 — 번들이 있으면 eglot 에
;; 등록돼 있어야 한다. jdtls·kotlin 은 Java 검사(lsp-jvm-test.el)를 거친다.
(ert-deftest imoogi-toolchain-clojure-server-registers-for-eglot ()
  "clojure-lsp 가 있으면 eglot 의 clojure 후보 목록에 그 이름이 있다."
  (skip-unless (executable-find "clojure-lsp"))
  (require 'eglot)
  (should (seq-find (lambda (entry)
                      (string-match-p "clojure-lsp" (format "%S" entry)))
                    eglot-server-programs)))

(provide 'toolchain-servers-test)
;;; toolchain-servers-test.el ends here
