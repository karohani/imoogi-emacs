;;; json-imenu-test.el --- JSON 버퍼의 imenu 테스트 -*- lexical-binding: t; -*-

;; JSON 은 두 경로 중 하나로 처리된다.
;;
;;   문법 있음 : json-ts-mode 가 자체 imenu 설정으로 키를 잡는다
;;   문법 없음 : js-json-mode 가 js-mode 의 JavaScript 인덱서를 물려받아
;;               아무것도 못 찾으므로, 18-languages 가 기본 인덱서 + 키 패턴으로
;;               보정한다
;;
;; 그래서 "어느 모드인가"가 아니라 "키가 잡히는가"를 단언한다 — 모드는 문법
;; 반입 여부에 따라 바뀌는 구현 세부사항이다.

;;; Code:

(require 'ert)
(require 'imenu)

(defmacro imoogi-json-test--with-buffer (&rest body)
  "임시 .json 파일을 열고 BODY 를 실행한다."
  `(let* ((dir (make-temp-file "imoogi-json" t))
          (file (expand-file-name "sample.json" dir)))
     (unwind-protect
         (progn
           (with-temp-file file
             (insert "{\n  \"name\": \"demo\",\n  \"deps\": {\n    \"inner\": 1\n  }\n}\n"))
           (with-current-buffer (find-file-noselect file)
             (unwind-protect (progn ,@body) (kill-buffer))))
       (delete-directory dir t))))

(ert-deftest imoogi-json-imenu-indexes-keys ()
  "경로와 무관하게 JSON 키가 imenu 항목으로 잡힌다."
  (imoogi-json-test--with-buffer
   (let ((names (mapcar #'car (imenu--make-index-alist))))
     (dolist (key '("name" "deps"))
       (should (member key names))))))

(ert-deftest imoogi-json-uses-tree-sitter-when-the-grammar-is-present ()
  "문법이 반입돼 있으면 json-ts-mode 가 인계받는다(보정 불필요)."
  (skip-unless (treesit-language-available-p 'json))
  (imoogi-json-test--with-buffer
   (should (eq major-mode 'json-ts-mode))
   (should (treesit-parser-list))))

(ert-deftest imoogi-json-fallback-restores-the-default-indexer ()
  "문법이 없을 때 쓰는 보정: js-mode 의 JavaScript 인덱서를 되돌린다.

문법 반입 여부와 무관하게 보정 함수 자체를 검사한다 — 문법이 있는 머신에서도
이 안전망이 살아 있는지 확인하기 위해서다."
  (with-temp-buffer
    (setq-local imenu-create-index-function #'js--imenu-create-index)
    (imoogi-json-setup-imenu)
    (should (eq imenu-create-index-function #'imenu-default-create-index-function))
    (should imenu-generic-expression)
    ;; 패턴이 실제로 키를 잡는지도 함께 본다.
    (insert "{\n  \"alpha\": 1\n}\n")
    (should (member "alpha" (mapcar #'car (imenu--make-index-alist))))))

(provide 'json-imenu-test)
;;; json-imenu-test.el ends here
