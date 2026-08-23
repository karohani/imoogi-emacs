;;; treesit-grammar-test.el --- 반입된 tree-sitter 문법 테스트 -*- lexical-binding: t; -*-

;; vendor/tree-sitter/ 에 문법이 있으면 해당 언어가 *-ts-mode 로 열리고 파서가
;; 살아 있어야 한다. 문법은 플랫폼별 바이너리라 반입되지 않은 환경(다른 OS,
;; 컨테이너)에서는 테스트를 건너뛴다 — 없는 것이 곧 실패는 아니다.

;;; Code:

(require 'ert)
(require 'treesit)

(defconst imoogi-treesit-test--expected
  '((json       . json-ts-mode)
    (javascript . js-ts-mode)
    (typescript . typescript-ts-mode)
    (tsx        . tsx-ts-mode)
    (python     . python-ts-mode)
    (go         . go-ts-mode)
    (java       . java-ts-mode)
    (yaml       . yaml-ts-mode))
  "반입 대상 문법과 그 문법이 켜줘야 할 메이저 모드.")

(defconst imoogi-treesit-test--samples
  '((json . ("s.json" . "{\"a\": 1}\n"))
    (javascript . ("s.js" . "function f() { return 1; }\n"))
    (typescript . ("s.ts" . "const x: number = 1;\n"))
    (tsx . ("s.tsx" . "const A = () => 1;\n"))
    (python . ("s.py" . "def f():\n    return 1\n"))
    (go . ("s.go" . "package main\nfunc main() {}\n"))
    (java . ("s.java" . "class A { void m() {} }\n"))
    (yaml . ("s.yaml" . "a: 1\n"))))

(ert-deftest imoogi-treesit-grammar-dir-is-on-the-load-path ()
  "18-languages 가 vendor/tree-sitter/ 를 treesit 경로에 넣는다."
  (should (member imoogi-treesit-grammar-dir treesit-extra-load-path)))

(ert-deftest imoogi-treesit-available-grammars-activate-their-ts-mode ()
  "반입된 문법마다 해당 파일이 ts-mode 로 열리고 파서가 붙는다.
문법이 없는 환경에서는 그 언어를 건너뛴다."
  (let ((checked 0))
    (dolist (entry imoogi-treesit-test--expected)
      (let ((lang (car entry)) (mode (cdr entry)))
        (when (treesit-language-available-p lang)
          (let* ((sample (alist-get lang imoogi-treesit-test--samples))
                 (dir (make-temp-file "imoogi-ts" t))
                 (file (expand-file-name (car sample) dir)))
            (unwind-protect
                (progn
                  (with-temp-file file (insert (cdr sample)))
                  (with-current-buffer (find-file-noselect file)
                    (should (eq major-mode mode))
                    (should (treesit-parser-list))
                    (setq checked (1+ checked))
                    (kill-buffer)))
              (delete-directory dir t))))))
    ;; 하나도 검사하지 못했다면 그 사실을 알린다(조용한 통과 방지).
    (when (zerop checked)
      (message "treesit 문법이 하나도 없어 전부 건너뜀"))))

(provide 'treesit-grammar-test)
;;; treesit-grammar-test.el ends here
