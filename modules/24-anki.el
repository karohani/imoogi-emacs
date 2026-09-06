;;; 24-anki.el --- one-way Org to Anki flashcard sync -*- lexical-binding: t; -*-

;;; Commentary:

;; Org 노트를 Anki 카드로 단방향 동기화한다. Anki 에서 Org 로 돌아오는 것은
;; imoogi 자신이 써 넣는 노트 식별자(ANKI_NOTE_ID) 뿐이다.
;;
;; 구현은 modules/anki/*.el 에, 렌더링과 AnkiConnect 통신을 맡는 Go 백엔드는
;; cmd/imoogi-anki/ 와 internal/anki/ 에 있다. 17-lsp.el 이 modules/lsp/ 를
;; 다루는 방식과 같게, 이 파일은 로드 경로를 붙이고 이 설정 저장소에만
;; 해당하는 편의(자동완성·단축키·메뉴)를 얹는 층이다.  modules/anki/*.el 은
;; 상류 패키지 그대로 두고 여기서만 감싼다.
;;
;; 백엔드 바이너리는 `make build-anki' 로 빌드한다.

;;; Code:

(imoogi-require "24-anki" 'org 'json 'seq 'url 'url-http)

;; compile-angel compiles this file in a fresh Emacs process.  Expand the
;; menu macro there too, before the deferred runtime registration executes.
(eval-when-compile (require 'transient))

(defvar imoogi-anki-lisp-dir
  (expand-file-name "modules/anki/" imoogi-emacs-dir)
  "Directory holding the Org-to-Anki implementation files.")

(add-to-list 'load-path imoogi-anki-lisp-dir)

;; `imoogi-sync' 와 `imoogi-anki-setup' 은 autoload 로 선언돼 있지만, 모듈
;; 로더는 autoload 쿠키를 수확하지 않으므로 여기서 직접 require 한다.
;;
;; imoogi.el 은 imoogi-setup.el 을 일부러 require 하지 않는다(imoogi-setup.el
;; 이 imoogi 를 require 하므로 순환이 된다). 원래 패키지는 autoload 로
;; 그 구멍을 메우지만 여기서는 통하지 않으므로, imoogi 를 먼저 올린 뒤
;; imoogi-setup 을 따로 require 한다.  이게 없으면 M-x imoogi-anki-setup 이
;; 아예 없는 명령이 되고, 최초 설정 자체를 할 수 없다.
(require 'imoogi)
(require 'imoogi-setup)

;;; 프로퍼티 자동완성

(defconst imoogi-anki-property-names
  '("ANKI_NOTE_TYPE" "ANKI_DECK" "ANKI_TAGS" "ANKI_NOTE_ID")
  "imoogi 가 읽는 Org 프로퍼티 이름.
`C-c C-x p'(`org-set-property')의 이름 후보로 등록된다 — 등록하지 않으면
버퍼에 이미 쓰인 프로퍼티만 후보로 나와서, 첫 카드를 만들 때 이름을
통째로 타이핑해야 한다.")

(with-eval-after-load 'org
  (dolist (name imoogi-anki-property-names)
    (add-to-list 'org-default-properties name t))

  ;; 값 후보: `PROP_ALL' 은 org 가 값을 Lisp 로 읽으므로 공백·괄호가 없는
  ;; 값에만 쓸 수 있다.  ANKI_NOTE_TYPE 은 imoogi-Basic/imoogi-Cloze 둘
  ;; 뿐이라 적합하다 (REQ-C-005.1 — imoogi 가 소유한 타입이 새 heading 의
  ;; 기본값이다).  손으로 쓴 스톡 Basic/Cloze heading 도 그대로 동기화되므로
  ;; (REQ-C-005.2) 후보에서 빠지는 것이 곧 금지는 아니다 — 후보는 편의일
  ;; 뿐이고, 값은 직접 입력해도 된다.
  ;; ANKI_DECK 은 적합하지 않다 — "(PROGRAMMER)::(GO)" 같은 실제 덱 이름이
  ;; ("???" "::" "???") 로 깨진다(실측).  덱은 아래 `imoogi-anki-set-deck'
  ;; 이 completing-read 로 처리한다.
  (add-to-list 'org-global-properties
               '("ANKI_NOTE_TYPE_ALL" . "imoogi-Basic imoogi-Cloze")))

;;; 노트 타입 이름 (REQ-C-005)

(defconst imoogi-anki-basic-note-type "imoogi-Basic"
  "새로 표시하는 Basic heading 이 받는 노트 타입 이름 (REQ-C-005.1).")

(defconst imoogi-anki-cloze-note-type "imoogi-Cloze"
  "새로 표시하는 Cloze heading 이 받는 노트 타입 이름 (REQ-C-005.1).")

(defun imoogi-anki-cloze-note-type-p (type)
  "TYPE 이 Cloze 계열이면 non-nil.

스톡 \"Cloze\" 와 imoogi 소유 \"imoogi-Cloze\" 를 모두 받아들인다
(REQ-C-005.2).  손으로 스톡 Cloze 를 쓴 heading 이 지금도 그대로 동기화
되므로, 빈칸 자동 표시가 그 heading 을 \"다른 타입\" 이라고 잘못 알리면
안 된다."
  (and type (member type (list "Cloze" imoogi-anki-cloze-note-type)) t))

;;; 명령

(defun imoogi-anki--at-heading ()
  "현재 heading 으로 이동한다.  org 버퍼가 아니면 알기 쉬운 오류를 낸다."
  (unless (derived-mode-p 'org-mode)
    (user-error "imoogi: Org 버퍼에서만 쓸 수 있습니다"))
  (org-back-to-heading t))

(defun imoogi-anki-mark-basic ()
  "이 heading 을 imoogi-Basic 카드로 표시한다(제목이 앞면, 본문이 뒷면).

imoogi 가 소유한 타입을 쓴다(REQ-C-005.1) — 스톡 Basic 은 Anki 가 소유한
타입이라 imoogi 가 그 모양(템플릿·CSS)에 손대지 않는다.  이미 스톡으로
쓰인 heading 은 그대로 동기화되고(REQ-C-005.2), `imoogi-anki-setup' 이
옮길지 물어본다."
  (interactive)
  (imoogi-anki--at-heading)
  (org-set-property "ANKI_NOTE_TYPE" imoogi-anki-basic-note-type)
  (message "imoogi: %s 카드로 표시했습니다" imoogi-anki-basic-note-type))

(defun imoogi-anki-mark-cloze ()
  "이 heading 을 imoogi-Cloze 카드로 표시한다(REQ-C-005.1).
제목과 본문이 한 필드로 들어가며, `{{cN::...}}' 표식이 하나도 없으면
동기화에서 이 heading 만 건너뛴다."
  (interactive)
  (imoogi-anki--at-heading)
  (org-set-property "ANKI_NOTE_TYPE" imoogi-anki-cloze-note-type)
  (message "imoogi: %s 카드로 표시했습니다" imoogi-anki-cloze-note-type))

(defun imoogi-anki-unmark ()
  "이 heading 의 카드 표시를 해제한다(ANKI_NOTE_TYPE 만 지운다).

ANKI_NOTE_ID 는 일부러 남긴다.  식별자를 지우는 것은 \"이 노트를 더는
소유하지 않는다\"는 선언이라 다음 동기화에서 Anki 쪽 노트가 삭제 후보가
되므로, 되돌리기 어려운 그 결정은 손으로 하도록 남겨 둔다."
  (interactive)
  (imoogi-anki--at-heading)
  (org-delete-property "ANKI_NOTE_TYPE")
  (message "imoogi: 카드 표시를 해제했습니다%s"
           (if (org-entry-get (point) "ANKI_NOTE_ID")
               " (ANKI_NOTE_ID 는 남겨 두었습니다)"
             "")))

(defun imoogi-anki--known-decks ()
  "`imoogi-sync-root' 아래 Org 파일에서 이미 쓰인 덱 이름을 모은다.

Anki 에 직접 묻지 않는다 — AnkiConnect 대화는 Go 백엔드가 전담한다는
이 패키지의 경계를 지키기 위해서다.  후보는 편의일 뿐이고, 목록에 없는
덱 이름도 그대로 입력하면 된다(없으면 동기화가 만들어 준다)."
  (let ((decks (and imoogi-default-deck (list imoogi-default-deck))))
    (when (and imoogi-sync-root (file-directory-p imoogi-sync-root))
      (dolist (file (directory-files-recursively imoogi-sync-root "\\.org\\'"))
        (with-temp-buffer
          (insert-file-contents file)
          (goto-char (point-min))
          (while (re-search-forward
                  "^[ \t]*\\(?::ANKI_DECK:\\|#\\+PROPERTY:[ \t]+ANKI_DECK\\)[ \t]+\\(.+?\\)[ \t]*$"
                  nil t)
            (push (match-string-no-properties 1) decks)))))
    (delete-dups (nreverse decks))))

(defun imoogi-anki--file-level-deck-p ()
  "이 버퍼에 파일 레벨 `#+PROPERTY: ANKI_DECK' 줄이 있으면 non-nil."
  (save-excursion
    (goto-char (point-min))
    (let ((case-fold-search t))
      (re-search-forward "^#\\+PROPERTY:[ \t]+ANKI_DECK\\b" nil t))))

(defun imoogi-anki-set-deck (deck)
  "ANKI_DECK 를 DECK 으로 지정한다.  `::' 로 중첩한다 — 예: \"Geography::Europe\".

어디에 쓰는지는 파일 상태로 정한다.  파일에 `#+PROPERTY: ANKI_DECK' 이
아직 없으면 heading 이 아니라 파일 맨 위에 그 줄을 쓴다 — 파일 하나가
보통 덱 하나라, 첫 카드의 덱이 파일 전체의 기본 덱이 되는 게 자연스럽고,
heading 마다 같은 덱을 되풀이해 달 필요가 없어진다.  파일 레벨 덱이
이미 있으면 heading 에 쓴다 — 그 heading 만 다른 덱으로 보내는 예외."
  (interactive
   (list (completing-read "덱: " (imoogi-anki--known-decks) nil nil
                          (org-entry-get (point) "ANKI_DECK" t))))
  (if (imoogi-anki--file-level-deck-p)
      (progn
        (imoogi-anki--at-heading)
        (org-set-property "ANKI_DECK" deck)
        (message "imoogi: 이 heading 만 덱 %s 으로 지정했습니다" deck))
    (save-excursion
      (goto-char (point-min))
      (insert (format "#+PROPERTY: ANKI_DECK %s\n" deck)))
    ;; #+PROPERTY 는 org 가 모드 진입 때 읽어 캐시하므로, 지금 쓴 줄이
    ;; 상속(`org-entry-get' 의 INHERIT)에 바로 보이도록 다시 읽게 한다.
    (org-set-regexps-and-options)
    (message "imoogi: 파일 기본 덱을 %s 으로 지정했습니다 (파일 맨 위 #+PROPERTY)" deck)))

(defun imoogi-anki-set-tags (tags)
  "이 heading 의 ANKI_TAGS 를 TAGS 로 지정한다(공백으로 구분).
빈 문자열을 주면 값이 빈 프로퍼티가 되고, 이는 상속을 끊는다는 뜻이다."
  (interactive
   (list (read-string "태그(공백 구분): "
                      (or (org-entry-get (point) "ANKI_TAGS" t) ""))))
  (imoogi-anki--at-heading)
  (org-set-property "ANKI_TAGS" tags))

(defun imoogi-anki--next-cloze-number ()
  "이 subtree 에서 아직 쓰지 않은 가장 작은 cloze 번호."
  (save-excursion
    (imoogi-anki--at-heading)
    (let ((end (save-excursion (org-end-of-subtree t t)))
          (highest 0))
      (while (re-search-forward "{{c\\([0-9]+\\)::" end t)
        (setq highest (max highest (string-to-number (match-string 1)))))
      (1+ highest))))

(defun imoogi-anki-cloze-region (beg end &optional number)
  "선택 영역 BEG..END 를 `{{cN::...}}' 로 감싼다.

N 은 이 heading 안에서 아직 안 쓴 다음 번호다.  접두 인자로 번호를 직접
주면(`C-u 1') 그 번호를 쓴다 — 여러 빈칸을 한 카드에서 동시에 보이게
할 때 쓴다.

빈칸을 만든다는 행위 자체가 \"이건 Cloze 카드\" 라는 뜻이므로, heading 에
ANKI_NOTE_TYPE 이 아직 없으면 Cloze 로 지정해 준다 (프로퍼티를 따로 달게
시키지 않는다).  이미 다른 타입(Basic)으로 표시돼 있으면 덮어쓰지 않고
알려만 준다 — 그 경우 동기화하면 {{cN::}} 가 Basic 카드에 글자 그대로
들어가므로, 의도한 게 아니면 `imoogi-anki-mark-cloze' 로 바꾸면 된다."
  (interactive "r\nP")
  (unless (use-region-p)
    (user-error "imoogi: 빈칸으로 만들 영역을 먼저 선택하세요"))
  (let* ((n (if number (prefix-numeric-value number) (imoogi-anki--next-cloze-number)))
         (text (buffer-substring-no-properties beg end)))
    (delete-region beg end)
    (insert (format "{{c%d::%s}}" n text))
    (save-excursion
      (imoogi-anki--at-heading)
      (let ((type (org-entry-get (point) "ANKI_NOTE_TYPE")))
        (cond
         ((null type)
          (org-set-property "ANKI_NOTE_TYPE" imoogi-anki-cloze-note-type)
          (message "imoogi: c%d 빈칸을 만들고 이 heading 을 %s 카드로 표시했습니다"
                   n imoogi-anki-cloze-note-type))
         ;; 스톡 "Cloze" 든 "imoogi-Cloze" 든 이미 Cloze 계열이면 건드리지
         ;; 않는다 (REQ-C-005.2) — 손으로 쓴 스톡 heading 을 조용히 다시
         ;; 타이핑하는 것은 사용자가 정하지 않은 변경이다.
         ((imoogi-anki-cloze-note-type-p type)
          (message "imoogi: c%d 빈칸을 만들었습니다" n))
         (t
          (message "imoogi: c%d 빈칸을 만들었지만 이 heading 은 %s 카드입니다 — 이대로 동기화하면 표식이 글자 그대로 들어갑니다. Cloze 로 바꾸려면 C-c a c"
                   n type)))))))

;;; 키 — C-c a 접두 맵 (17-lsp.el 의 C-c l 맵과 같은 구조)

(defvar imoogi-anki-map
  (let ((map (make-sparse-keymap)))
    (define-key map (kbd "b") #'imoogi-anki-mark-basic)
    (define-key map (kbd "c") #'imoogi-anki-mark-cloze)
    (define-key map (kbd "x") #'imoogi-anki-unmark)
    (define-key map (kbd "d") #'imoogi-anki-set-deck)
    (define-key map (kbd "t") #'imoogi-anki-set-tags)
    (define-key map (kbd "z") #'imoogi-anki-cloze-region)
    (define-key map (kbd "s") #'imoogi-sync)
    map)
  "Anki 명령 접두 맵.  Org 버퍼에서 `C-c a' 에 붙는다.

`C-c a' 는 org 세계에서 관례적으로 `org-agenda' 자리다.  이 설정은 agenda
를 바인딩하지 않아 지금은 비어 있고(실측), 나중에 agenda 를 쓰게 되면
여기 접두를 옮기면 된다.")

(with-eval-after-load 'org
  (define-key org-mode-map (kbd "C-c a") imoogi-anki-map))

;; Also update the existing prefix map when this module is reloaded.
(dolist (binding '(("D" . imoogi-anki-register-directory)
                   ("F" . imoogi-anki-register-file)
                   ("l" . imoogi-anki-list-targets)
                   ("L" . imoogi-anki-list-files)
                   ("u" . imoogi-anki-unregister-target)))
  (define-key imoogi-anki-map (kbd (car binding)) (cdr binding)))

;;; transient 메뉴

;; 17-lsp.el 과 같은 구조: 05-transient 가 로드된 경우에만 메뉴를 만들고,
;; 마스터 메뉴에는 스스로 등록한다.  05 가 건너뛰어졌으면 메뉴만 없고
;; 위 `C-c a' 접두 맵은 그대로 동작한다.  등록 키가 다른 모듈과 겹치면
;; 나중 등록이 조용히 이기므로, tests/transient-menu-test.el 이 마스터의
;; 키·명령 쌍을 단언해 충돌을 잡는다.
(with-eval-after-load 'imoogi-transient
  (transient-define-prefix imoogi-anki-transient ()
    "Org → Anki 카드."
    :column-widths '(20 22 16 22)
    [["표시 -------------"
      ("b" "imoogi-Basic 카드로" imoogi-anki-mark-basic)
      ("c" "imoogi-Cloze 카드로" imoogi-anki-mark-cloze)
      ("x" "표시 해제" imoogi-anki-unmark)]
     ["내용 ---------------"
      ("d" "덱 지정" imoogi-anki-set-deck)
      ("t" "태그 지정" imoogi-anki-set-tags)
      ("z" "선택 영역을 빈칸으로" imoogi-anki-cloze-region)]
     ["실행 -----------"
      ("s" "동기화" imoogi-sync)
      ("S" "최초 설정" imoogi-anki-setup)
      ("q" "종료" transient-quit-one)]
     ["동기화 대상 ---------"
      ("D" "폴더 등록" imoogi-anki-register-directory)
      ("F" "파일 등록" imoogi-anki-register-file)
      ("l" "등록 목록" imoogi-anki-list-targets)
      ("L" "파일 목록" imoogi-anki-list-files)
      ("u" "등록 해제" imoogi-anki-unregister-target)]])

  (define-key imoogi-anki-map (kbd "a") #'imoogi-anki-transient)

  (transient-append-suffix 'imoogi-transient-master "t"
    '("a" "Anki" imoogi-anki-transient)))

(provide 'imoogi-anki)
;;; 24-anki.el ends here
