;;; toolchain-servers-test.el --- 반입된 LSP 서버 테스트 -*- lexical-binding: t; -*-

;; vendor/toolchains 에서 setup 으로 설치된 서버가 Emacs 에 실제로 보이는지
;; 확인한다. 설치는 플랫폼별이고 사용자가 setup 을 돌려야 생기므로, .local/bin
;; 이 활성화되지 않은 환경(컨테이너, 미설치 머신)에서는 건너뛴다.

;;; Code:

(require 'ert)

(ert-deftest imoogi-toolchain-servers-are-on-exec-path ()
  "활성 번들이 있으면 그 안의 서버들이 executable-find 로 잡힌다."
  (skip-unless (imoogi-lsp-local-bin-dir))
  (dolist (command '("gopls" "typescript-language-server" "basedpyright-langserver"))
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

(provide 'toolchain-servers-test)
;;; toolchain-servers-test.el ends here
