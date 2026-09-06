;;; anki-commands-test.el --- modules/24-anki.el 의 편집 명령 -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'org)

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

(provide 'anki-commands-test)
;;; anki-commands-test.el ends here
