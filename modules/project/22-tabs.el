;;; 22-tabs.el --- tab-bar: tmux 의 window 층 -*- lexical-binding: t; no-native-compile: t; -*-

;; [HARD] no-native-compile: 17-lsp.el 과 같은 이유다 — Emacs 31.1 네이티브
;; 컴파일러가 `with-eval-after-load 'imoogi-transient' + transient-define-prefix'
;; 조합을 깨뜨려 모듈이 통째로 로드에 실패한다. 자세한 경위는 17-lsp.el 머리말 참고.

;; tmux 를 쓰던 손이 Emacs 에서 비는 자리는 "window" 하나다.
;;
;;   tmux session  →  perspective (04-projects)   이미 있음
;;   tmux window   →  tab-bar                     이 모듈
;;   tmux pane     →  Emacs 창 (05-transient)     이미 있음
;;
;; tab-bar 는 Emacs 내장이라 벤더링이 필요 없다 — 망분리 저장소에서 같은 층을
;; 외부 패키지로 채우는 것보다 압도적으로 싸다.
;;
;; 접두 자리: Emacs 표준은 `C-x t' 지만 이 저장소는 그 자리를 treemacs 가
;; 먼저 쓰고 있다(07-treemacs). 그래서 비어 있는 `C-c w' 를 쓴다(w = window,
;; tmux 용어). 자주 쓰는 앞/뒤 이동만 `s-[' / `s-]' 로 따로 뺀다.

;;; Code:

(imoogi-require "22-tabs" 'tab-bar)

(defvar-keymap imoogi-tab-map
  :doc "tab-bar(=tmux window) 명령 접두 맵."
  "c" #'tab-bar-new-tab
  "k" #'tab-bar-close-tab
  "K" #'tab-bar-close-other-tabs
  "r" #'tab-bar-rename-tab
  "n" #'tab-bar-switch-to-next-tab
  "p" #'tab-bar-switch-to-prev-tab
  "s" #'tab-bar-switch-to-tab
  "l" #'tab-bar-switch-to-recent-tab
  "u" #'tab-bar-undo-close-tab
  "t" #'tab-bar-mode)

(use-package tab-bar
  :ensure nil
  :custom
  ;; 탭이 하나뿐이면 막대를 숨긴다 — 쓰기 전에는 화면을 차지하지 않는다.
  (tab-bar-show 1)
  (tab-bar-new-tab-choice "*scratch*")
  (tab-bar-close-button-show nil)
  ;; 새 탭 이름은 그 탭에서 처음 연 버퍼가 아니라 현재 프로젝트를 따르게 한다.
  (tab-bar-tab-name-function #'imoogi-tab-name)
  :config
  (global-set-key (kbd "C-c w") imoogi-tab-map)
  (global-set-key (kbd "s-[") #'tab-bar-switch-to-prev-tab)
  (global-set-key (kbd "s-]") #'tab-bar-switch-to-next-tab))

(defun imoogi-tab-name ()
  "탭 이름 — 현재 프로젝트 이름, 없으면 현재 버퍼 이름.
tmux 에서 window 이름이 하는 역할과 맞춘다."
  (if-let* ((project (and (fboundp 'project-current) (project-current nil))))
      (file-name-nondirectory
       (directory-file-name (project-root project)))
    (tab-bar-tab-name-current)))

(with-eval-after-load 'which-key
  (which-key-add-key-based-replacements "C-c w" "탭(tmux window)"))

;; 마스터 메뉴에 자기 항목을 스스로 등록한다(ARCHITECTURE.md 참조).
;; 05-transient 가 이 모듈을 알 필요가 없고, 이 모듈이 로드 실패하면 항목도
;; 함께 사라진다.
(with-eval-after-load 'imoogi-transient
  (transient-define-prefix imoogi-transient-tab ()
    "탭 — tmux 의 window 층."
    :column-widths '(17 19)
    [["이동 -----------"
      ("n" "다음" tab-bar-switch-to-next-tab :transient t)
      ("p" "이전" tab-bar-switch-to-prev-tab :transient t)
      ("l" "직전 탭" tab-bar-switch-to-recent-tab)
      ("s" "목록에서 선택" tab-bar-switch-to-tab)]
     ["관리 -------------"
      ("c" "새 탭" tab-bar-new-tab)
      ("r" "이름 변경" tab-bar-rename-tab)
      ("k" "닫기" tab-bar-close-tab)
      ("u" "닫은 탭 되살리기" tab-bar-undo-close-tab)
      ("t" "탭 막대 토글" tab-bar-mode)
      ("q" "종료" transient-quit-one)]])

  (transient-append-suffix 'imoogi-transient-master "t"
    '("T" "탭" imoogi-transient-tab)))

(provide 'imoogi-tabs)
;;; 22-tabs.el ends here
