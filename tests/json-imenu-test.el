;;; json-imenu-test.el --- JSON 버퍼의 imenu 보정 테스트 -*- lexical-binding: t; -*-

;; js-json-mode 는 js-mode 에서 파생돼 JavaScript 인덱서를 물려받는데, 그건
;; 순수 데이터인 JSON 에서 아무것도 찾지 못한다. 18-languages 가 그 보정을
;; 넣으므로, 실제 JSON 버퍼를 열어 항목이 잡히는지 확인한다.

;;; Code:

(require 'ert)
(require 'imenu)

(defmacro imoogi-json-test--with-buffer (&rest body)
  "임시 .json 파일을 열고 BODY 를 실행한다."
  `(let ((file (make-temp-file "imoogi-json" nil ".json"
                               "{\n  \"name\": \"demo\",\n  \"deps\": {\n    \"inner\": 1\n  }\n}\n")))
     (unwind-protect
         (with-current-buffer (find-file-noselect file) ,@body)
       (delete-file file))))

(ert-deftest imoogi-json-buffer-uses-the-default-imenu-indexer ()
  "js-mode 가 걸어둔 JavaScript 인덱서를 기본 구현으로 되돌린다."
  (imoogi-json-test--with-buffer
   (should (eq major-mode 'js-json-mode))
   (should (eq imenu-create-index-function #'imenu-default-create-index-function))))

(ert-deftest imoogi-json-imenu-indexes-keys ()
  "중첩된 키까지 imenu 항목으로 잡힌다."
  (imoogi-json-test--with-buffer
   (let ((names (mapcar #'car (imenu--make-index-alist))))
     (dolist (key '("name" "deps" "inner"))
       (should (member key names))))))

(provide 'json-imenu-test)
;;; json-imenu-test.el ends here
