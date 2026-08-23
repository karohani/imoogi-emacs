;;; 15-markdown.el --- markdown 설정 -*- lexical-binding: t; -*-

;; markdown-mode + markdown-toc. 기본 조작감은 org-mode 구조 편집 키에 맞춘다.
;; markdown-mode 자체는 obsidian 의존성으로 이미 vendor 에 포함돼 있다.

;;; Code:

(imoogi-require "15-markdown" 'markdown-mode 'markdown-toc 'edit-indirect)

;;; markdown-mode — org-mode 기본 구조 편집 키와 맞추기
(declare-function markdown-cur-list-item-bounds "markdown-mode")
(declare-function markdown-demote "markdown-mode")
(declare-function markdown-insert-list-item "markdown-mode" (&optional arg))
(declare-function markdown-insert-gfm-checkbox "markdown-mode")

(defun imoogi-markdown-list-has-previous-sibling-p (bounds)
  "Return non-nil when list item BOUNDS has a previous item at the same level."
  (let ((indent (nth 2 bounds))
        (item-start (nth 0 bounds))
        found
        done)
    (save-excursion
      (goto-char item-start)
      (forward-line -1)
      (while (and (not found) (not done))
        (if (looking-at-p "^[[:blank:]]*$")
            (setq done t)
          (let ((prev (markdown-cur-list-item-bounds)))
            (when (and prev
                       (< (nth 0 prev) item-start)
                       (= (nth 2 prev) indent))
              (setq found t)))
          (if (bobp)
              (setq done t)
            (forward-line -1)))))
    found))

(defun imoogi-markdown-insert-task-list-item (&optional arg)
  "Insert a Markdown task list item, like `org-insert-todo-heading'."
  (interactive "p")
  (markdown-insert-list-item arg)
  (markdown-insert-gfm-checkbox))

(defun imoogi-markdown-demote ()
  "Demote Markdown structure without turning first list items into code blocks."
  (interactive)
  (let ((bounds (markdown-cur-list-item-bounds)))
    (if (and bounds
             (not (imoogi-markdown-list-has-previous-sibling-p bounds)))
        (user-error "Cannot demote the first item at this list level")
      (call-interactively #'markdown-demote))))

(use-package markdown-mode
  :ensure t
  :commands (markdown-mode gfm-mode)
  :bind (:map markdown-mode-map
              ("M-<left>" . markdown-promote)
              ("M-<right>" . imoogi-markdown-demote)
              ("M-<up>" . markdown-move-up)
              ("M-<down>" . markdown-move-down)
              ("M-S-<return>" . imoogi-markdown-insert-task-list-item)))

;;; markdown-toc — 마크다운 목차(TOC) 생성
(use-package markdown-toc
  :ensure t
  :commands (markdown-toc-generate-toc markdown-toc-refresh-toc))

(provide 'imoogi-markdown)
;;; 15-markdown.el ends here
