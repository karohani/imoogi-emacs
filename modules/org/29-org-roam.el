;;; 29-org-roam.el --- Central notes and cached node completion -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "29-org-roam" 'org-roam 'memoize)

(require 'transient)

(defvar imoogi-org-roam--cache-timer nil
  "Idle timer that refreshes node completion candidates.")

(defun imoogi-org-roam-clear-completions-cache ()
  "Clear the cached node completion list after notes change."
  (interactive)
  (when (get 'org-roam-node-read--completions :memoize-original-function)
    (memoize-restore 'org-roam-node-read--completions)
    (memoize 'org-roam-node-read--completions "10 minute")))

(defun imoogi-org-roam-node-at-point-p ()
  "Return non-nil when point is on an Org-roam node."
  (and (derived-mode-p 'org-mode)
       (org-roam-node-at-point)))

(defun imoogi-org-roam-refile-context-p ()
  "Return non-nil when an Org heading or region can be refiled."
  (and (derived-mode-p 'org-mode)
       (or (org-at-heading-p) (org-region-active-p))))

(defun imoogi-org-roam--enable-db-autosync-if-directory-exists ()
  "Enable Org-roam autosync when the permanent notes directory exists."
  (when (file-directory-p org-roam-directory)
    (org-roam-db-autosync-mode 1)))

(use-package org-roam
  :ensure t
  :after org
  :custom
  (org-roam-directory (imoogi-org--permanent-directory))
  (org-roam-node-default-sort nil)
  :bind (("C-c n" . imoogi-org-roam-transient))
  :config
  (require 'memoize)
  (unless (get 'org-roam-node-read--completions :memoize-original-function)
    (memoize 'org-roam-node-read--completions "10 minute"))
  (when (timerp imoogi-org-roam--cache-timer)
    (cancel-timer imoogi-org-roam--cache-timer))
  (setq imoogi-org-roam--cache-timer
        (run-with-idle-timer 60 t #'imoogi-org-roam-clear-completions-cache))
  (imoogi-org-roam--enable-db-autosync-if-directory-exists))

(transient-define-prefix imoogi-org-roam-transient ()
  "Work with central Org-roam notes."
  :column-widths '(23 23 23)
  [["탐색·연결"
      ("f" "노트 찾기·만들기" org-roam-node-find)
      ("n" "노트 캡처" org-roam-capture)
      ("i" "노트 링크 삽입" org-roam-node-insert :inapt-if-not (lambda () (derived-mode-p 'org-mode)))
      ("b" "백링크 보기" org-roam-buffer-toggle :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("R" "무작위 노트" org-roam-node-random)
      ("g" "노트 그래프" org-roam-graph :inapt-if-not (lambda () (executable-find "dot")))]
     ["현재 노트·제목"
      ("r" "다른 노트로 옮기기" org-roam-refile :inapt-if-not imoogi-org-roam-refile-context-p)
      ("a" "별칭 추가" org-roam-alias-add :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("A" "별칭 제거" org-roam-alias-remove :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("t" "태그 추가" org-roam-tag-add :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("T" "태그 제거" org-roam-tag-remove :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("e" "참조 추가" org-roam-ref-add :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("E" "참조 제거" org-roam-ref-remove :inapt-if-not imoogi-org-roam-node-at-point-p)]
     ["일일 노트·관리"
      ("d" "오늘 일일 노트" org-roam-dailies-goto-today)
      ("D" "날짜로 일일 노트" org-roam-dailies-goto-date)
      ("s" "데이터베이스 동기화" org-roam-db-sync)
      ("c" "노드 후보 캐시 갱신" imoogi-org-roam-clear-completions-cache)
    ("q" "종료" transient-quit-one)]])
  (transient-append-suffix 'imoogi-org-agenda-transient "m"
    '("r" "중앙 노트" imoogi-org-roam-transient))

(provide 'imoogi-org-roam)
;;; 29-org-roam.el ends here
