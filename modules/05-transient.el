;;; transient.el --- Transient menu definitions -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "05-transient" 'transient 'ace-window)

(use-package transient
  :ensure t)

;;; 팝업 표시 — 크기와 정렬
;;
;; 정렬: transient 는 열 폭을 `length'(문자 수)로 계산한다(transient.el 의
;; `transient--columns-width' 계열, string-width 를 쓰지 않음). 한글은 표시 폭이
;; 2라 문자 수로 세면 열이 밀린다. 그래서 각 prefix 에 `:column-widths' 로 실제
;; 표시 폭 기준 값을 직접 준다. 그룹 제목 뒤의 ─ 는 열 경계를 눈으로 잡아주는
;; 구분선이며, 폭 안에 들어가도록 길이를 맞춘다.

(defcustom imoogi-transient-min-height-fraction 0.33
  "transient 팝업이 차지할 프레임 높이의 최소 비율.
내용이 더 길면 그만큼 커진다(이 값은 하한이지 고정 높이가 아니다).
nil 이면 transient 기본 동작(내용에 딱 맞춤)을 쓴다."
  :type '(choice (const :tag "transient 기본" nil) number)
  :group 'imoogi)

(defun imoogi-transient--enlarge-window (window)
  "WINDOW 를 프레임 높이의 최소 비율까지 넓힌다.
transient 는 `fit-window-to-buffer' 를 최소 높이 1로 호출해 팝업을 내용에 딱
맞게 줄인다. 짧은 메뉴가 너무 납작해 보이므로 하한만 끌어올린다."
  (when (and imoogi-transient-min-height-fraction
             (window-live-p window)
             ;; 프레임 전체를 팝업으로 쓰는 경우(부모 없음)는 건드리지 않는다.
             (window-parent window))
    (let* ((target (round (* imoogi-transient-min-height-fraction
                             (frame-height (window-frame window)))))
           (delta (- target (window-height window))))
      (when (> delta 0)
        (ignore-errors (window-resize window delta nil t))
        ;; transient 가 방금 기록한 preserved-size 를 지워, 다음 렌더에서 다시
        ;; 줄어들지 않게 한다.
        (set-window-parameter window 'window-preserved-size nil)))))

(advice-add 'transient--fit-window-to-buffer
            :after #'imoogi-transient--enlarge-window)

;;; ace-window
(use-package ace-window
  :ensure t
  :bind ("M-o" . ace-window)
  :custom
  (aw-keys '(?a ?s ?d ?f ?g ?h ?j ?k ?l))
  (aw-scope 'frame))

;; 창 관리
;; @MX:NOTE 각 suffix 의 :transient t 유무는 hydra 시절 head 색을 그대로 옮긴
;; 것이다(SPEC-TRANSIENT-001 plan.md §C.1). 이 메뉴는 amaranth 였으므로 기본이
;; 열린 채 유지이고, 예외는 D(나머지삭제)와 q(종료) 둘뿐이다.
(transient-define-prefix imoogi-transient-window ()
  "창 이동·분할·크기 조절 메뉴."
  :column-widths '(12 15 19 19)
  [["이동 ──────"
    ("h" "←" windmove-left :transient t)
    ("l" "→" windmove-right :transient t)
    ("j" "↓" windmove-down :transient t)
    ("k" "↑" windmove-up :transient t)]
   ["크기 ─────────"
    ("H" "축소←" shrink-window-horizontally :transient t)
    ("L" "확대→" enlarge-window-horizontally :transient t)
    ("J" "확대↓" enlarge-window :transient t)
    ("K" "축소↑" shrink-window :transient t)]
   ["분할·삭제 ───────"
    ("s" "수평분할" split-window-below :transient t)
    ("v" "수직분할" split-window-right :transient t)
    ("d" "삭제" delete-window :transient t)
    ("D" "나머지삭제" delete-other-windows)]
   ["기타 ────────────"
    ("b" "버퍼전환" imoogi-consult-perspective-buffer :transient t)
    ("f" "파일열기" find-file :transient t)
    ("a" "ace-window" ace-window :transient t)
    ("m" "스왑" ace-swap-window :transient t)
    ("q" "종료" transient-quit-one)]])

;; 프로젝트
;; @MX:NOTE hydra 시절 whole-hydra :color blue — 모든 suffix 가 실행 후 닫힌다
;; (plan.md §C.2). 그래서 :transient t 가 하나도 없다.
(transient-define-prefix imoogi-transient-project ()
  "프로젝트·작업공간 메뉴."
  :column-widths '(17 17 21)
  [["열기 ───────────"
    ("p" "프로젝트 전환" imoogi-project-switch-perspective)
    ("f" "파일찾기" project-find-file)
    ("s" "검색(grep)" project-find-regexp)
    ("d" "dired" project-dired)
    ("b" "버퍼" project-switch-to-buffer)]
   ;; 이미 열려 있는 작업공간 사이를 오가는 쪽. n/N 은 훑어보는 동작이라
   ;; 메뉴를 열어 둔 채 반복할 수 있게 :transient t 를 준다.
   ["작업공간 전환 ──"
    ("l" "직전 작업공간" persp-switch-last)
    ("o" "목록에서 선택" persp-switch)
    ("n" "다음" persp-next :transient t)
    ("N" "이전" persp-prev :transient t)
    ("#" "번호로" persp-switch-by-number)]
   ["관리 ──────────────"
    ("c" "컴파일" project-compile)
    ("k" "버퍼모두닫기" project-kill-buffers)
    ("r" "작업공간 이름변경" persp-rename)
    ("K" "작업공간 닫기" persp-kill)
    ("F" "목록에서 제거" project-forget-project)
    ("q" "종료" transient-quit-one)]])

;; 텍스트 확대/축소
(defun imoogi-transient-zoom-reset ()
  "텍스트 배율을 기본값으로 되돌린다."
  (interactive)
  (text-scale-set 0))

;; @MX:NOTE amaranth 였으므로 i/o 는 열린 채 유지, 0(초기화)과 q 만 닫힌다
;; (plan.md §C.3).
(transient-define-prefix imoogi-transient-zoom ()
  "텍스트 확대/축소 메뉴."
  :column-widths '(14)
  [["확대/축소 ───"
    ("i" "확대" text-scale-increase :transient t)
    ("o" "축소" text-scale-decrease :transient t)
    ("0" "초기화" imoogi-transient-zoom-reset)
    ("q" "종료" transient-quit-one)]])

;; Git (Magit)
;; @MX:NOTE hydra 시절 whole-hydra :color blue — 전부 실행 후 닫힌다
;; (plan.md §C.4).
(transient-define-prefix imoogi-transient-git ()
  "Magit 진입 메뉴."
  :column-widths '(14)
  [["Git ─────────"
    ("s" "status" magit-status)
    ("l" "log" magit-log-current)
    ("b" "blame" magit-blame)
    ("d" "diff" magit-diff-dwim)
    ("q" "종료" transient-quit-one)]])

;;; 코드 이해 — 깊이 5단계
;;
;; L1 빠른 질의 → L2 상주 패널 → L3 구문 → L4 관계 → L5 전체 그래프.
;; 각 단계는 서로 독립적으로 가용해진다(문법을 반입하면 L3만, graphviz 를 깔면
;; L5만 켜진다). 갖춰지지 않은 단계는 숨기지 않고 흐리게 둬서(:inapt-*),
;; 이 메뉴가 "지금 이 버퍼에서 어디까지 파고들 수 있는지"를 보여주는 지도가 되게 한다.
;;
;; @MX:NOTE 이 메뉴는 단일 모듈 소관이 아니다 — L1(consult/02) L2(treemacs/07)
;; L3(treesit/18) L4(eglot/17) 이 각각 다른 모듈에 흩어져 있어, 모듈이 스스로
;; 등록하는 방식(17-lsp 참조)을 쓸 수 없다. 마스터 메뉴를 소유한 여기에 둔다.
;; 참조하는 명령들은 이 파일보다 뒤에 로드되지만, 심볼은 키를 누를 때 해석되므로
;; 문제가 되지 않는다(마스터의 t → treemacs 항목이 이미 같은 형태).

(defun imoogi-code--panel-available-p ()
  "L2 상주 패널(treemacs)이 로드됐으면 non-nil."
  (fboundp 'imoogi-treemacs-toggle-structure))

(defun imoogi-code--syntax-available-p ()
  "현재 버퍼에 tree-sitter 파서가 살아 있으면 non-nil.
major-mode 로 언어를 역추적하지 않고 버퍼에 직접 묻는다 — `treesit-explore-mode'
가 실제로 요구하는 조건이 파서의 존재이기 때문이다."
  (and (fboundp 'treesit-parser-list)
       (treesit-parser-list)
       t))

(defun imoogi-code--relations-available-p ()
  "현재 버퍼가 Eglot 관리 하에 있으면 non-nil."
  (and (fboundp 'eglot-managed-p) (eglot-managed-p)))

(defun imoogi-code--graph-available-p ()
  "L5 전체 그래프 생성이 준비됐으면 non-nil.

아직 생성기가 없어 항상 nil 이다. graphviz(dot)와 언어별 analyzer 를 갖춘 뒤
이 함수의 조건만 바꾸면 메뉴 항목이 저절로 켜진다 — 메뉴 정의는 손대지 않는다."
  nil)

(defun imoogi-code-capability-report ()
  "코드 이해 5단계 중 지금 무엇이 되고 무엇이 왜 안 되는지 보고한다."
  (interactive)
  (let ((buf (current-buffer)))
    (with-output-to-temp-buffer "*imoogi 코드 이해 가용성*"
      (princ (format "버퍼: %s (%s)\n\n" (buffer-name buf)
                     (buffer-local-value 'major-mode buf)))
      (dolist (row
               (list
                (list "L1 빠른 탐색 (imenu)" t "—")
                (list "L2 상주 패널 (treemacs)"
                      (imoogi-code--panel-available-p)
                      "07-treemacs 모듈이 로드되지 않았다. git 설치 여부를 확인한다.")
                (list "L3 구문 트리 (tree-sitter)"
                      (imoogi-code--syntax-available-p)
                      (concat "이 버퍼에 파서가 없다. 문법 라이브러리를 "
                              "vendor/tree-sitter/ 에 반입하면 켜진다."))
                (list "L4 관계 — xref" t "—")
                (list "L4 관계 — Eglot 심화"
                      (imoogi-code--relations-available-p)
                      "이 버퍼에 Eglot 이 붙지 않았다. C-c l e 로 연결한다.")
                (list "L4 관계 — 호출/타입 계층" nil
                      (concat "이 Emacs 의 내장 eglot 에 기능 자체가 없다"
                              " (eglot.el 에 hierarchy 없음). 상위 버전 필요."))
                (list "L5 전체 그래프"
                      (imoogi-code--graph-available-p)
                      "생성기 미구현. graphviz(dot) + 언어별 analyzer 가 필요하다.")))
        (princ (format "%-2s %-28s %s\n"
                       (if (nth 1 row) "✓" "·")
                       (nth 0 row)
                       (if (nth 1 row) "" (nth 2 row))))))))

(transient-define-prefix imoogi-transient-code ()
  "코드 이해 — 깊이 5단계."
  :column-widths '(17 14 13 13 15)
  [["L1 빠른 탐색 ──"
    ("i" "이 파일 심볼" consult-imenu)
    ("I" "프로젝트 심볼" consult-imenu-multi)]
   ["L2 상주 패널 ─"
    ("s" "구조 패널" imoogi-treemacs-toggle-structure
     :inapt-if-not imoogi-code--panel-available-p)
    ("f" "파일 트리" imoogi-treemacs-toggle-file-tree
     :inapt-if-not imoogi-code--panel-available-p)]
   ["L3 구문 ────"
    ("t" "구문 트리" treesit-explore-mode
     :inapt-if-not imoogi-code--syntax-available-p)]
   ["L4 관계 ────"
    ("d" "정의로" xref-find-definitions)
    ("r" "참조 찾기" xref-find-references)
    ("m" "구현" eglot-find-implementation
     :inapt-if-not imoogi-code--relations-available-p)
    ("y" "타입 정의" eglot-find-typeDefinition
     :inapt-if-not imoogi-code--relations-available-p)]
   ["L5 전체 ─────"
    ("g" "전체 그래프" imoogi-code-capability-report
     :inapt-if-not imoogi-code--graph-available-p)
    ("?" "가용성 보고" imoogi-code-capability-report)
    ("q" "종료" transient-quit-one)]])

;; 마스터 메뉴 (진입점)
;; @MX:ANCHOR C-c h 로 전역 바인딩된 유일한 공개 진입점. 하위 4개 메뉴의
;; 디스패치 지점이며, hydra-master/body 의 역할을 그대로 대체한다.
(transient-define-prefix imoogi-transient-master ()
  "imoogi 마스터 메뉴."
  :column-widths '(15 15 17)
  [["메뉴 ─────────"
    ("w" "창관리" imoogi-transient-window)
    ("p" "프로젝트" imoogi-transient-project)
    ("g" "Git" imoogi-transient-git)
    ("z" "확대/축소" imoogi-transient-zoom)]
   ["도구 ─────────"
    ("c" "코드 이해" imoogi-transient-code)
    ("t" "treemacs" imoogi-treemacs-toggle-file-tree)]
   ["설정 ───────────"
    ("R" "설정 재로드" imoogi-reload)
    ("q" "종료" transient-quit-one)]])

(global-set-key (kbd "C-c h") 'imoogi-transient-master)

(provide 'imoogi-transient)
;;; transient.el ends here
