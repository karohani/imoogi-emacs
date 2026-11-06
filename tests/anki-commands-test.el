;;; anki-commands-test.el --- modules/org/24-anki.el 의 편집 명령 -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'org)
(require 'cl-lib)
(require 'json)
(require 'seq)

(ert-deftest imoogi-anki-reload-restores-deck-picker-dependencies ()
  "Reload an already provided imoogi implementation after a code update."
  (let ((old-binary-path imoogi-anki-binary-path))
    (unwind-protect
        (progn
          (makunbound 'imoogi-anki-binary-path)
          (fmakunbound 'imoogi-process-list-decks)
          (load (expand-file-name "modules/org/24-anki.el" imoogi-emacs-dir)
                nil t)
          (should (boundp 'imoogi-anki-binary-path))
          (should (fboundp 'imoogi-process-list-decks)))
      (setq imoogi-anki-binary-path old-binary-path))))

(defmacro imoogi-anki-test--with-org (text &rest body)
  "TEXT 로 채운 임시 org 버퍼에서 BODY 를 실행한다.
배치 Emacs 는 `transient-mark-mode' 가 꺼져 있어 `use-region-p' 가 늘 nil
이므로, 명령이 대화형 세션에서 보는 그대로의 조건을 만들기 위해 켠다."
  (declare (indent 1))
  `(with-temp-buffer
     (org-mode)
     (setq-local transient-mark-mode t)
     (insert ,text)
     ,@body))

(defun imoogi-anki-test--select (str)
  "버퍼에서 STR 이 처음 나오는 자리를 활성 리전으로 잡고 (BEG END) 를 돌려준다."
  (goto-char (point-min))
  (search-forward str)
  (let ((beg (match-beginning 0)) (end (match-end 0)))
    (set-mark beg)
    (goto-char end)
    (activate-mark)
    (list beg end)))

(ert-deftest imoogi-anki-cloze-region-sets-cloze-when-type-absent ()
  "카드 표식이 없는 heading 에서 빈칸을 만들면 ANKI_NOTE_TYPE 이
imoogi-Cloze 가 된다.  빈칸을 만든다는 행위 자체가 \"이건 Cloze 카드\"
라는 뜻이므로, 프로퍼티를 따로 달게 시키지 않는다 (카드 t10).

이름이 스톡 \"Cloze\" 에서 \"imoogi-Cloze\" 로 바뀐 것은 REQ-C-005.1 —
새로 표시하는 heading 의 기본값이 imoogi 소유 타입이 되었다."
  (imoogi-anki-test--with-org "* 제목\n\n유럽에서 가장 긴 강은 볼가강이다.\n"
    (let ((r (imoogi-anki-test--select "볼가강")))
      (imoogi-anki-cloze-region (car r) (cadr r)))
    (should (string-match-p "{{c1::볼가강}}" (buffer-string)))
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Cloze"))))

(ert-deftest imoogi-anki-cloze-region-leaves-basic-alone ()
  "이미 Basic 인 heading 은 덮어쓰지 않는다 -- 사용자가 \"없으면\" 이라고
범위를 정했다.  빈칸은 감싸되 프로퍼티는 그대로."
  (imoogi-anki-test--with-org "* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n\n본문 볼가강 끝.\n"
    (let ((r (imoogi-anki-test--select "볼가강")))
      (imoogi-anki-cloze-region (car r) (cadr r)))
    (should (string-match-p "{{c1::볼가강}}" (buffer-string)))
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "Basic"))))

(ert-deftest imoogi-anki-cloze-region-numbers-continue-within-subtree ()
  "같은 heading 안의 두 번째 빈칸은 c2 가 된다."
  (imoogi-anki-test--with-org "* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Cloze\n:END:\n\n{{c1::첫째}} 그리고 둘째.\n"
    (let ((r (imoogi-anki-test--select "둘째")))
      (imoogi-anki-cloze-region (car r) (cadr r)))
    (should (string-match-p "{{c2::둘째}}" (buffer-string)))))

(ert-deftest imoogi-anki-cloze-region-separates-inner-double-brace ()
  "선택 영역 안의 `}}' 는 `} }' 로 띄운다.

Anki 의 cloze 정규식은 비탐욕(non-greedy)이라 첫 `}}' 에서 빈칸을 닫는다.
영역 안에 `}}' 가 그대로 있으면 빈칸이 의도보다 일찍 끝나고 나머지가
글자 그대로 카드에 남는다.  LaTeX 은 중괄호 사이 공백을 무시하므로 수식
의미는 그대로다."
  (imoogi-anki-test--with-org "* 제목\n\n앞 a}}b 뒤\n"
    (let ((r (imoogi-anki-test--select "a}}b")))
      (imoogi-anki-cloze-region (car r) (cadr r)))
    (should (string-match-p (regexp-quote "{{c1::a} }b}}") (buffer-string)))))

(ert-deftest imoogi-anki-cloze-region-separates-trailing-brace-from-marker ()
  "선택이 `}' 로 끝나면 닫는 `}}' 앞에 공백을 넣는다.

`\\sqrt{a^{2}}' 처럼 중괄호로 끝나는 수식은 안쪽 `}}' 를 띄우고 나면
`\\sqrt{a^{2} }' 가 되는데, 그대로 감싸면 `...} }}}' 가 되어 Anki 가 다시
한 글자 일찍 닫는다.  닫기 앞의 공백이 그 경계를 떼어 놓는다."
  (imoogi-anki-test--with-org "* 제목\n\n공식은 $\\sqrt{a^{2}}$ 이다.\n"
    (let ((r (imoogi-anki-test--select "\\sqrt{a^{2}}")))
      (imoogi-anki-cloze-region (car r) (cadr r)))
    (should (string-match-p (regexp-quote "{{c1::\\sqrt{a^{2} } }}") (buffer-string)))))

(ert-deftest imoogi-anki-cloze-region-leaves-brace-free-text-untouched ()
  "중괄호가 없는 영역은 공백 하나 더 붙지 않는다 -- 이스케이프가 일반
경로를 건드리지 않는다는 회귀 방지."
  (imoogi-anki-test--with-org "* 제목\n\n유럽에서 가장 긴 강은 볼가강이다.\n"
    (let ((r (imoogi-anki-test--select "볼가강")))
      (imoogi-anki-cloze-region (car r) (cadr r)))
    (should (string-match-p (regexp-quote "{{c1::볼가강}}") (buffer-string)))))

(ert-deftest imoogi-anki-cloze-region-falls-back-to-word-at-point ()
  "영역을 잡지 않고 불러도 커서가 놓인 낱말이 빈칸이 된다.

명령의 interactive 사양이 직접 경계를 고르므로, 대화형 호출과 같은
경로를 거치도록 `call-interactively' 로 부른다."
  (imoogi-anki-test--with-org "* 제목\n\n유럽에서 가장 긴 강은 Volga 이다.\n"
    (goto-char (point-min))
    (search-forward "Volga")
    (goto-char (match-beginning 0))
    (deactivate-mark)
    (call-interactively #'imoogi-anki-cloze-region)
    (should (string-match-p (regexp-quote "{{c1::Volga}}") (buffer-string)))))

(ert-deftest imoogi-anki-set-deck-writes-file-level-when-file-has-none ()
  "파일에 #+PROPERTY: ANKI_DECK 이 아직 없으면 heading 이 아니라 파일 맨 위에
쓴다 -- 파일 하나가 보통 덱 하나라, 첫 카드의 덱이 파일 전체의 기본 덱이
된다.  heading 에는 아무것도 남기지 않아 상속으로만 해석된다."
  (imoogi-anki-test--with-org "* 첫 카드\n\n본문\n"
    (goto-char (point-max))
    (imoogi-anki-set-deck "(PROGRAMMER)::(GO)")
    (goto-char (point-min))
    (should (looking-at "#\\+PROPERTY: ANKI_DECK (PROGRAMMER)::(GO)$"))
    (goto-char (point-max)) (org-back-to-heading t)
    (should-not (org-entry-get (point) "ANKI_DECK"))
    ;; 상속으로는 보인다 -- 동기화가 실제로 쓰는 경로
    (should (equal (org-entry-get (point) "ANKI_DECK" t) "(PROGRAMMER)::(GO)"))))

(ert-deftest imoogi-anki-set-deck-before-first-heading-updates-file-default ()
  "At the file header, replace the file default instead of editing a heading."
  (imoogi-anki-test--with-org
      "#+TITLE: 단어\n#+PROPERTY: ANKI_DECK 기존 덱\n\n* 첫 카드\n"
    (goto-char (point-min))
    (let (initial)
      (cl-letf (((symbol-function 'imoogi-anki--known-decks)
                 (lambda () '("기존 덱" "새 덱")))
                ((symbol-function 'completing-read)
                 (lambda (_prompt _collection _predicate _require-match input
                          &rest _)
                   (setq initial input)
                   "새 덱")))
        (call-interactively #'imoogi-anki-set-deck))
      (should (equal initial "기존 덱")))
    (goto-char (point-min))
    (should (= (count-matches "^#\\+PROPERTY: ANKI_DECK") 1))
    (should (re-search-forward "^#\\+PROPERTY: ANKI_DECK 새 덱$" nil t))
    (should-not (re-search-forward "^:ANKI_DECK:" nil t))))

(ert-deftest imoogi-anki-set-deck-before-first-heading-creates-file-default ()
  (imoogi-anki-test--with-org "#+TITLE: 단어\n\n* 첫 카드\n"
    (goto-char (point-min))
    (cl-letf (((symbol-function 'imoogi-anki--known-decks) (lambda () nil))
              ((symbol-function 'completing-read)
               (lambda (&rest _) "새 덱")))
      (call-interactively #'imoogi-anki-set-deck))
    (goto-char (point-min))
    (should (looking-at "#\\+PROPERTY: ANKI_DECK 새 덱$"))))

(ert-deftest imoogi-anki-known-decks-merges-live-and-local-names ()
  (let ((imoogi-default-deck "Default")
        (imoogi-sync-root nil))
    (cl-letf (((symbol-function 'executable-find) (lambda (_name) "/bin/imoogi-anki"))
              ((symbol-function 'imoogi-process-list-decks)
               (lambda (_binary _url) '("현재 덱" "Default"))))
      (should (equal (imoogi-anki--known-decks) '("현재 덱" "Default"))))))

(ert-deftest imoogi-anki-set-deck-writes-heading-level-when-file-already-has-one ()
  "파일 레벨 덱이 이미 있으면 heading 에 쓴다 -- 그 heading 만 다른 덱."
  (imoogi-anki-test--with-org "#+PROPERTY: ANKI_DECK 기본덱\n\n* 예외 카드\n\n본문\n"
    (org-set-regexps-and-options)
    (goto-char (point-max))
    (imoogi-anki-set-deck "다른덱")
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_DECK") "다른덱"))
    (goto-char (point-min))
    (should (= 1 (count-matches "^#\\+PROPERTY: ANKI_DECK")))))

;; --- SPEC-ANKICARD-002 AC-OPT-007: 앞단 cloze 판별의 나머지 절반

(ert-deftest imoogi-anki-test-cloze-note-type-p-matches-the-back-end ()
  "AC-OPT-007: 앞단 판별기가 back end 판별기와 똑같은 여덟 값에 똑같이
답한다.

같은 표를 양쪽에서 돌리는 것이 요점이다.  두 판별기가 각자 맞기만 해서는
부족하고, 서로 어긋나지 않아야 한다 -- 어긋나면 사용자가 Org 쪽에서 보는
빈칸 자동 표시와 back end 가 내리는 판정이 갈라진다."
  (dolist (row '(("Cloze" . t)
                 ("imoogi-Cloze" . t)
                 ("Basic" . nil)
                 ("imoogi-Basic" . nil)
                 ("imoogi-Other" . nil)
                 ("MyCloze" . nil)
                 ("cloze" . nil)
                 ("" . nil)))
    (should (eq (imoogi-anki-cloze-note-type-p (car row))
                (cdr row)))))

(provide 'anki-commands-test)
;;; anki-commands-test.el ends here

;;; --- SPEC-ANKICARD-003: 멀티라인 카드 옵션 명령 (REQ-ML-013) ---

(ert-deftest imoogi-anki-set-direction-writes-the-chosen-arrow ()
  "AC-ML-012a: 카드 옵션이 없는 heading 에 방향을 지정하면 그 heading 자신의
드로어에 ANKI_DIRECTION 이 쓰인다."
  (imoogi-anki-test--with-org "* 제목\n\n- 답 하나\n"
    (goto-char (point-max))
    (imoogi-anki-set-direction "<->")
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_DIRECTION") "<->"))))

;; 거짓 철자를 확인할 때는 `org-entry-get' 이 아니라 `imoogi-props-resolve-*'
;; 로 읽는다.  실측: `:ANKI_DIRECTION: nil' 이 드로어에 분명히 쓰여 있어도
;; `org-entry-get' 은 Lisp nil 을 돌려준다 -- 값이 있는 프로퍼티와 없는
;; 프로퍼티를 구별해 주지 못한다.  back end 가 실제로 읽는 경로는
;; `imoogi-props--own-value' 로, 드로어 글을 정규식으로 직접 읽으므로
;; 문자열 "nil" 을 그대로 돌려준다.  그래서 이쪽이 맞는 접근자이자 더 강한
;; 단언이다 -- 동기화가 보게 될 바로 그 값을 확인한다.

(ert-deftest imoogi-anki-set-direction-clear-writes-the-falsy-spelling ()
  "AC-ML-012b 의 방향 쪽 절반: 끄는 것은 지우는 것이 아니다.

세 프로퍼티 모두 상속되므로, 프로퍼티를 지우면 조상(또는 파일 레벨)의 값이
다시 드러난다.  heading 자신의 드로어에 거짓 철자를 쓰는 것이 문서화된
해제 방법이고, back end 가 세 프로퍼티 모두에서 그 철자를 인식하는 이유도
바로 이 균일성 때문이다."
  (imoogi-anki-test--with-org
      "#+PROPERTY: ANKI_DIRECTION ->\n\n* 제목\n\n- 답 하나\n"
    (org-set-regexps-and-options)
    (goto-char (point-max))
    (imoogi-anki-set-direction nil)
    (org-back-to-heading t)
    ;; 동기화가 실제로 읽는 경로로 확인한다.
    (should (equal (imoogi-props-resolve-direction) "nil"))
    ;; 그리고 지워진 것이 아니라 쓰인 것이다 -- 드로어에 줄이 남아 있어야
    ;; 파일 레벨 `#+PROPERTY:' 의 상속이 끊긴다.
    (should (string-match-p "^[ \t]*:ANKI_DIRECTION:[ \t]*nil[ \t]*$"
                            (buffer-string)))))

(ert-deftest imoogi-anki-toggle-incremental-turns-an-inherited-option-off ()
  "AC-ML-012b: 상속된 옵션을 끄면 heading 자신의 드로어에 거짓 철자가 쓰이고,
프로퍼티가 단순히 지워지지는 않는다."
  (imoogi-anki-test--with-org
      "#+PROPERTY: ANKI_INCREMENTAL t\n\n* 제목\n\n- 답 하나\n"
    (org-set-regexps-and-options)
    (goto-char (point-max))
    (imoogi-anki-toggle-incremental)
    (org-back-to-heading t)
    (should (equal (imoogi-props-resolve-incremental) "nil"))
    (should (string-match-p "^[ \t]*:ANKI_INCREMENTAL:[ \t]*nil[ \t]*$"
                            (buffer-string)))))

(ert-deftest imoogi-anki-toggle-incremental-turns-an-absent-option-on ()
  "옵션이 어디에도 없으면 토글은 켠다."
  (imoogi-anki-test--with-org "* 제목\n\n- 답 하나\n"
    (goto-char (point-max))
    (imoogi-anki-toggle-incremental)
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_INCREMENTAL") "t"))))

(ert-deftest imoogi-anki-set-direction-follows-the-note-type-contract ()
  "AC-ML-012c: `imoogi-anki-cloze-region' 과 같은 계약.
타입이 없으면 imoogi-Cloze 를 달아 주고, 이미 Cloze 계열이면 그대로 두고,
다른 타입이면 알리기만 하고 아무것도 바꾸지 않는다."
  ;; 타입 없음 -> imoogi-Cloze 를 달아 준다
  (imoogi-anki-test--with-org "* 제목\n\n- 답 하나\n"
    (goto-char (point-max))
    (imoogi-anki-set-direction "->")
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Cloze")))

  ;; 이미 Cloze 계열 -> 손대지 않는다 (손으로 쓴 스톡 이름을 조용히 바꾸지 않는다)
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Cloze\n:END:\n\n- 답 하나\n"
    (goto-char (point-max))
    (imoogi-anki-set-direction "->")
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "Cloze"))
    (should (equal (org-entry-get (point) "ANKI_DIRECTION") "->")))

  ;; 다른 타입 -> 알리기만 하고 아무것도 쓰지 않는다
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Basic\n:END:\n\n- 답 하나\n"
    (goto-char (point-max))
    (imoogi-anki-set-direction "->")
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Basic"))
    (should-not (org-entry-get (point) "ANKI_DIRECTION"))))

(ert-deftest imoogi-anki-set-direction-declining-the-swift-conflict-writes-nothing ()
  "AC-ML-012d 앞 절반: swift 가 켜진 heading 에서 사용자가 swift 해제를
거절하면 아무 프로퍼티도 쓰이지 않는다 -- 방향도, 노트 타입도.

거절한 명령이 흔적을 남기지 않는 것이 요점이다.  방향을 그대로 썼다면
다음 동기화에서 SPEC-ANKICARD-002 의 충돌 규칙이 거부할 heading 이
만들어지고, 사용자는 카드를 설정했다고 믿은 시점에는 그것을 알 수 없다."
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_SWIFT: t\n:END:\n\n- 답 하나\n"
    (goto-char (point-max))
    (cl-letf (((symbol-function 'yes-or-no-p) (lambda (&rest _) nil)))
      (imoogi-anki-set-direction "->"))
    (org-back-to-heading t)
    (should-not (org-entry-get (point) "ANKI_DIRECTION"))
    (should-not (org-entry-get (point) "ANKI_NOTE_TYPE"))
    (should (equal (org-entry-get (point) "ANKI_SWIFT") "t"))))

(ert-deftest imoogi-anki-set-direction-confirming-the-swift-conflict-clears-swift ()
  "AC-ML-012d 뒤 절반: 확인하면 방향이 쓰이고 ANKI_SWIFT 는 heading 자신의
드로어에서 거짓 철자가 된다 (지워지지 않는다 -- 상속 때문)."
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_SWIFT: t\n:END:\n\n- 답 하나\n"
    (goto-char (point-max))
    (cl-letf (((symbol-function 'yes-or-no-p) (lambda (&rest _) t)))
      (imoogi-anki-set-direction "->"))
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_DIRECTION") "->"))
    (should (equal (imoogi-props-resolve-swift) "nil"))))

(ert-deftest imoogi-anki-toggle-incremental-also-honours-the-swift-conflict ()
  "충돌 규칙은 두 멀티라인 옵션 모두에 걸린다 -- 둘 다 swift 와
다른 카드 종류를 고르기 때문이다."
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_SWIFT: t\n:END:\n\n- 답 하나\n"
    (goto-char (point-max))
    (cl-letf (((symbol-function 'yes-or-no-p) (lambda (&rest _) nil)))
      (imoogi-anki-toggle-incremental))
    (org-back-to-heading t)
    (should-not (org-entry-get (point) "ANKI_INCREMENTAL"))))

(defmacro imoogi-anki-test--refused-unchanged (command)
  "imoogi-Basic + swift heading 에서 COMMAND 를 실행하고, 거부되어 버퍼가 그대로인지 본다.

swift 해제 확인에는 \"예\" 로 답하게 둔다.  노트 타입 확인이 swift 충돌
해소보다 늦으면, 거부될 heading 의 ANKI_SWIFT 가 nil 로 바뀐 뒤에야
거부된다 (카드 t36 -- t15 발견 3 과 같은 유형)."
  `(imoogi-anki-test--with-org
       "* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Basic\n:ANKI_SWIFT: t\n:END:\n\n- 답 하나\n"
     (goto-char (point-max))
     (let ((before (buffer-string))
           said)
       (cl-letf (((symbol-function 'yes-or-no-p) (lambda (&rest _) t))
                 ((symbol-function 'message)
                  (lambda (fmt &rest args) (push (apply #'format fmt args) said))))
         ,command)
       (should (seq-some (lambda (m) (string-match-p "imoogi-Basic" m)) said))
       (should (equal (buffer-string) before)))
     (org-back-to-heading t)
     (should (equal (org-entry-get (point) "ANKI_SWIFT") "t"))))

(ert-deftest imoogi-anki-set-direction-refuses-a-non-cloze-type-before-clearing-swift ()
  "카드 t36: 다른 타입이면 swift 해제에 \"예\" 라고 답할 상황이어도 아무것도 바꾸지 않는다."
  (imoogi-anki-test--refused-unchanged (imoogi-anki-set-direction "->")))

(ert-deftest imoogi-anki-toggle-incremental-refuses-a-non-cloze-type-before-clearing-swift ()
  "카드 t36: `imoogi-anki-set-direction' 과 같은 순서를 지킨다."
  (imoogi-anki-test--refused-unchanged (imoogi-anki-toggle-incremental)))

(ert-deftest imoogi-anki-multiline-property-names-are-completion-candidates ()
  "AC-ML-012e: 두 이름은 프로퍼티 이름 후보에 들어간다.

이 테스트는 원래 ANKI_SWIFT 가 들어가지 않는다는 것도 단언했다 -- 그
이름은 그 프로퍼티를 쓰는 명령(t15)과 함께 간다는 것이 SPEC-ANKICARD-003
REQ-ML-013.5 의 판단이었다.  SPEC-ANKICARD-004 가 그 명령을 들여오면서 이
유보를 거둬들이므로(REQ-SW-013.5), 그 단언은 아래
`imoogi-anki-swift-property-name-is-a-completion-candidate' 가 뒤집어
이어받는다."
  (should (member "ANKI_DIRECTION" imoogi-anki-property-names))
  (should (member "ANKI_INCREMENTAL" imoogi-anki-property-names)))

;;; --- SPEC-ANKICARD-004: swift 카드 옵션 명령 (REQ-SW-013) ---

;; 아래 OFF 방향 테스트는 확인 프롬프트를 `항상 예' 로 바꿔 두지 않는다
;; (AC-SW-014, plan.md DD-10).  그렇게 바꿔 두면 프롬프트가 떠도 테스트가
;; 통과해, 고치려는 결함이 그대로 숨는다.  대신 프롬프트가 뜨는 순간 테스트가
;; 실패하도록 덫을 놓는다 -- 답을 주는 것이 아니라 뜨는 것 자체를 잡는다.
;; 배치 Emacs 에서 맨 프롬프트는 표준 입력을 읽으므로, 덫이 없으면 실행
;; 환경에 따라 실패하는 대신 멈춰 버릴 수 있다.

(defmacro imoogi-anki-test--no-prompt (&rest body)
  "BODY 를 실행하되, 확인 프롬프트가 뜨면 그 자리에서 테스트를 실패시킨다."
  (declare (indent 0))
  `(cl-letf (((symbol-function 'yes-or-no-p)
              (lambda (prompt &rest _) (ert-fail (list "프롬프트가 떴다" prompt))))
             ((symbol-function 'y-or-n-p)
              (lambda (prompt &rest _) (ert-fail (list "프롬프트가 떴다" prompt)))))
     ,@body))

(defun imoogi-anki-test--own-drawer-line-p (property value)
  "heading 자신의 드로어에 PROPERTY 가 VALUE 로 쓰여 있으면 non-nil."
  (string-match-p (format "^[ \t]*:%s:[ \t]*%s[ \t]*$"
                          (regexp-quote property) (regexp-quote value))
                  (buffer-string)))

(ert-deftest imoogi-anki-toggle-swift-turns-an-absent-option-on ()
  "AC-SW-014: ANKI_SWIFT 가 없는 heading 에서 토글하면 켜고, 노트 타입이
없으면 imoogi-Cloze 를 달아 준다."
  (imoogi-anki-test--with-org "* 제목\n\n도쿄 :-> 일본\n"
    (goto-char (point-max))
    (imoogi-anki-test--no-prompt
      (imoogi-anki-toggle-swift))
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_SWIFT") "t"))
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Cloze"))))

(ert-deftest imoogi-anki-toggle-swift-leaves-a-cloze-style-type-alone ()
  "AC-SW-014: 이미 Cloze 계열이면 노트 타입은 그대로 둔다 -- 손으로 쓴 스톡
이름을 조용히 바꾸지 않는다."
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Cloze\n:END:\n\n도쿄 :-> 일본\n"
    (goto-char (point-max))
    (imoogi-anki-test--no-prompt
      (imoogi-anki-toggle-swift))
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_SWIFT") "t"))
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "Cloze"))))

(ert-deftest imoogi-anki-toggle-swift-turns-an-inherited-option-off-without-a-prompt ()
  "AC-SW-014: 상속으로 켜진 swift 를 끄면 heading 자신의 드로어에 거짓 철자가
쓰이고 프로퍼티가 지워지지 않으며, swift 충돌 프롬프트는 뜨지 않는다.

`imoogi-anki--write-card-option' 을 거쳤다면 여기서 \"swift 를 끄고
계속할까요?\" 가 떴을 것이다 -- 끄려는 바로 그 경로에서.  검증은
`org-entry-get' 이 아니라 `imoogi-props-resolve-swift' 로 한다
(REQ-ML-013.2 의 실측)."
  (imoogi-anki-test--with-org
      "#+PROPERTY: ANKI_SWIFT t\n\n* 제목\n\n도쿄 :-> 일본\n"
    (org-set-regexps-and-options)
    (goto-char (point-max))
    (imoogi-anki-test--no-prompt
      (imoogi-anki-toggle-swift))
    (org-back-to-heading t)
    (should (equal (imoogi-props-resolve-swift) "nil"))
    (should (imoogi-anki-test--own-drawer-line-p "ANKI_SWIFT" "nil"))))

(ert-deftest imoogi-anki-toggle-swift-on-with-both-multiline-options-asks-first ()
  "AC-SW-014: 켜는 방향에서 두 멀티라인 옵션이 모두 켜져 있으면 묻는다.
거절하면 아무것도 쓰지 않는다 -- ANKI_SWIFT 도, ANKI_NOTE_TYPE 도."
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_DIRECTION: ->\n:ANKI_INCREMENTAL: t\n:END:\n\n도쿄 :-> 일본\n"
    (goto-char (point-max))
    (let ((asked 0))
      (cl-letf (((symbol-function 'yes-or-no-p)
                 (lambda (&rest _) (setq asked (1+ asked)) nil)))
        (imoogi-anki-toggle-swift))
      (should (= asked 1)))
    (org-back-to-heading t)
    (should-not (org-entry-get (point) "ANKI_SWIFT"))
    (should-not (org-entry-get (point) "ANKI_NOTE_TYPE"))
    (should (equal (imoogi-props-resolve-direction) "->"))
    (should (equal (imoogi-props-resolve-incremental) "t"))))

(ert-deftest imoogi-anki-toggle-swift-on-confirmed-clears-every-multiline-option ()
  "AC-SW-014: 확인하면 켜진 멀티라인 옵션을 하나도 남김없이 거짓 철자로 끄고
ANKI_SWIFT 를 쓴다.  하나만 끄면 충돌 규칙이 거부하는 바로 그 상태가 남는다.
알림은 끈 옵션을 모두 이름으로 부른다."
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_DIRECTION: ->\n:ANKI_INCREMENTAL: t\n:END:\n\n도쿄 :-> 일본\n"
    (goto-char (point-max))
    (let (prompt-text)
      (cl-letf (((symbol-function 'yes-or-no-p)
                 (lambda (prompt &rest _) (setq prompt-text prompt) t)))
        (imoogi-anki-toggle-swift))
      (should (string-match-p "ANKI_DIRECTION" prompt-text))
      (should (string-match-p "ANKI_INCREMENTAL" prompt-text)))
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_SWIFT") "t"))
    (should (equal (imoogi-props-resolve-direction) "nil"))
    (should (equal (imoogi-props-resolve-incremental) "nil"))
    (should (imoogi-anki-test--own-drawer-line-p "ANKI_DIRECTION" "nil"))
    (should (imoogi-anki-test--own-drawer-line-p "ANKI_INCREMENTAL" "nil"))))

(ert-deftest imoogi-anki-toggle-swift-off-keeps-the-multiline-option-without-a-prompt ()
  "AC-SW-014 (REQ-SW-013.4): 끄는 방향에서는 멀티라인 충돌 확인을 하지 않는다.

heading 자신의 드로어에 ANKI_DIRECTION 이 있고 swift 는 상속으로만 켜진
경우 -- 동기화가 card_option_conflict 로 거부하는 바로 그 heading 이다.
swift 를 끄는 것이 그 진단이 안내하는 해결이므로, 여기서 방향을 지울지
묻는다면 사용자가 지키려던 옵션을 지우게 된다."
  (imoogi-anki-test--with-org
      "#+PROPERTY: ANKI_SWIFT t\n\n* 제목\n:PROPERTIES:\n:ANKI_DIRECTION: ->\n:END:\n\n- 답 하나\n"
    (org-set-regexps-and-options)
    (goto-char (point-max))
    (imoogi-anki-test--no-prompt
      (imoogi-anki-toggle-swift))
    (org-back-to-heading t)
    (should (equal (imoogi-props-resolve-swift) "nil"))
    (should (imoogi-anki-test--own-drawer-line-p "ANKI_SWIFT" "nil"))
    (should (equal (imoogi-props-resolve-direction) "->"))
    (should-not (string-match-p "ANKI_INCREMENTAL" (buffer-string)))))

(ert-deftest imoogi-anki-toggle-swift-on-refuses-a-non-cloze-type ()
  "AC-SW-014: 켜는 방향에서 노트 타입이 imoogi-Basic 이면 알리기만 하고
아무것도 쓰지 않는다 -- ANKI_SWIFT 도, ANKI_NOTE_TYPE 도."
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Basic\n:END:\n\n도쿄 :-> 일본\n"
    (goto-char (point-max))
    (let (said)
      (cl-letf (((symbol-function 'message)
                 (lambda (fmt &rest args) (push (apply #'format fmt args) said))))
        (imoogi-anki-test--no-prompt
          (imoogi-anki-toggle-swift)))
      ;; 실패 메시지는 멀티라인 쌍이 아니라 카드 옵션을 부른다 -- 이제
      ;; swift heading 에서도 이 메시지가 나오기 때문이다.
      (should (seq-some (lambda (m) (string-match-p "imoogi-Basic" m)) said))
      (should (seq-some (lambda (m) (string-match-p "카드 옵션" m)) said))
      (should-not (seq-some (lambda (m) (string-match-p "멀티라인" m)) said)))
    (org-back-to-heading t)
    (should-not (org-entry-get (point) "ANKI_SWIFT"))
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Basic"))))

(ert-deftest imoogi-anki-toggle-swift-on-refuses-a-non-cloze-type-before-clearing-options ()
  "REQ-SW-013.3: 다른 타입이면 알리기만 하고 아무것도 바꾸지 않는다 -- 멀티라인
충돌 확인에 \"예\" 라고 답할 수 있는 상황이어도.

노트 타입 확인이 충돌 해소보다 늦으면, imoogi-Basic heading 자신의
ANKI_DIRECTION 이 지워진 뒤에야 swift 가 거부된다 (카드 리뷰 t15, 발견 3).
그래서 프롬프트는 \"예\" 로 답하게 두고, 프로퍼티가 하나도 바뀌지 않았는지
본다."
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Basic\n:ANKI_DIRECTION: ->\n:END:\n\n도쿄 :-> 일본\n"
    (goto-char (point-max))
    (let ((before (buffer-string))
          said)
      (cl-letf (((symbol-function 'yes-or-no-p) (lambda (&rest _) t))
                ((symbol-function 'message)
                 (lambda (fmt &rest args) (push (apply #'format fmt args) said))))
        (imoogi-anki-toggle-swift))
      (should (seq-some (lambda (m) (string-match-p "imoogi-Basic" m)) said))
      (should (equal (buffer-string) before)))
    (org-back-to-heading t)
    (should-not (org-entry-get (point) "ANKI_SWIFT"))
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Basic"))
    (should (equal (org-entry-get (point) "ANKI_DIRECTION") "->"))))

(ert-deftest imoogi-anki-toggle-swift-off-on-a-basic-heading-writes-the-falsy-spelling ()
  "AC-SW-014 (REQ-SW-013.3): 끄는 방향에서는 노트 타입 계약이 걸리지 않는다.

heading 자신의 드로어에 imoogi-Basic 이 있고 swift 는 상속으로만 켜진
경우 -- 동기화가 card_option_needs_cloze 로 거부하는 heading 이다.  그
진단이 안내하는 해결이 바로 자신의 드로어에 `nil' 을 쓰는 것이므로, 여기서
거부하면 명령이 진단의 처방을 거부하는 셈이 된다."
  (imoogi-anki-test--with-org
      "#+PROPERTY: ANKI_SWIFT t\n\n* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Basic\n:END:\n\n본문\n"
    (org-set-regexps-and-options)
    (goto-char (point-max))
    (let (said)
      (cl-letf (((symbol-function 'message)
                 (lambda (fmt &rest args) (push (apply #'format fmt args) said))))
        (imoogi-anki-test--no-prompt
          (imoogi-anki-toggle-swift)))
      (should-not (seq-some (lambda (m) (string-match-p "카드 옵션을 달 수 없습니다" m)) said)))
    (org-back-to-heading t)
    (should (equal (imoogi-props-resolve-swift) "nil"))
    (should (imoogi-anki-test--own-drawer-line-p "ANKI_SWIFT" "nil"))
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Basic"))))

(ert-deftest imoogi-anki-toggle-swift-off-creates-no-note-type ()
  "AC-SW-014: 노트 타입이 없는 heading 에서 상속된 swift 를 끄면 거짓 철자만
쓰이고 ANKI_NOTE_TYPE 은 생기지 않는다 -- 카드 옵션을 거절하는 행위가 카드
표식을 만들지 않는다."
  (imoogi-anki-test--with-org
      "#+PROPERTY: ANKI_SWIFT t\n\n* 제목\n\n본문\n"
    (org-set-regexps-and-options)
    (goto-char (point-max))
    (imoogi-anki-test--no-prompt
      (imoogi-anki-toggle-swift))
    (org-back-to-heading t)
    (should (equal (imoogi-props-resolve-swift) "nil"))
    (should-not (org-entry-get (point) "ANKI_NOTE_TYPE"))))

(ert-deftest imoogi-anki-multiline-commands-name-the-card-option-on-a-non-cloze-type ()
  "REQ-SW-013.3: 노트 타입 헬퍼의 실패 메시지는 이제 멀티라인 쌍이 아니라
카드 옵션을 부른다.  기존 명령에서도 같은 헬퍼가 같은 메시지를 낸다."
  (imoogi-anki-test--with-org
      "* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Basic\n:END:\n\n- 답 하나\n"
    (goto-char (point-max))
    (let (said)
      (cl-letf (((symbol-function 'message)
                 (lambda (fmt &rest args) (push (apply #'format fmt args) said))))
        (imoogi-anki-set-direction "->"))
      (should (seq-some (lambda (m) (string-match-p "카드 옵션" m)) said))
      (should-not (seq-some (lambda (m) (string-match-p "멀티라인" m)) said)))))

(ert-deftest imoogi-anki-swift-property-name-is-a-completion-candidate ()
  "AC-SW-014 (REQ-SW-013.5): ANKI_SWIFT 는 이제 프로퍼티 이름 후보다 --
그 프로퍼티를 쓰는 명령이 이 카드와 함께 들어왔다."
  (should (member "ANKI_SWIFT" imoogi-anki-property-names)))

(ert-deftest imoogi-anki-toggle-swift-is-reachable-from-the-key-map-and-the-menu ()
  "REQ-SW-013: swift 토글은 `C-c a' 접두 맵과 Anki transient 에서 닿는다."
  (should (eq (lookup-key imoogi-anki-map (kbd "w")) #'imoogi-anki-toggle-swift))
  (should (fboundp 'imoogi-anki-transient))
  (should (eq (plist-get (cdr (transient-get-suffix 'imoogi-anki-transient "w")) :command)
              #'imoogi-anki-toggle-swift)))

;;; --- AC-ML-007c: Go 작성기와 편집기 헬퍼가 같은 하나의 픽스처를 읽는다 ---

(ert-deftest imoogi-anki-cloze-safe-fixture-matches-the-go-composer ()
  "AC-ML-007c: 빈칸 안전 변환을 분리와 패딩까지 한 번에 적용하는 헬퍼가,
Go 작성기와 똑같은 하나의 픽스처의 모든 줄에서 같은 결과를 낸다.

같은 파일을 양쪽에서 읽는 것이 요점이다.  리터럴을 양쪽에 베껴 두면 둘이
말없이 갈라질 수 있고, 그러면 사용자가 손으로 빈칸을 만들 때 보는 결과와
composition 이 같은 글을 감쌌을 때의 결과가 달라진다.

`imoogi-anki--cloze-safe-text' 가 아니라 이 헬퍼를 부르는 이유: 그 함수는
분리만 하고, 패딩은 호출자인 `imoogi-anki-cloze-region' 에 있다.  그래서
어떤 기존 함수 하나도 이 요구가 말하는 형태를 만들지 못한다 (REQ-ML-007.3)."
  (let* ((path (expand-file-name
                "internal/anki/orgdoc/testdata/cloze-brace-fixture.json"
                imoogi-emacs-dir))
         (rows (with-temp-buffer
                 (insert-file-contents path)
                 (json-parse-buffer :object-type 'alist :array-type 'list))))
    (should (> (length rows) 0))
    (dolist (row rows)
      (let ((in (alist-get 'in row))
            (want (alist-get 'want row))
            (why (alist-get 'why row)))
        (should (equal (cons why (imoogi-anki-cloze-safe-span in))
                       (cons why want)))))))
