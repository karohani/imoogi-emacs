;;; 14-org-markdown.el --- org / markdown 설정 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; org-mode(내장) 기본 설정 + org-appear, markdown-toc.
;; markdown-mode 자체는 obsidian 의존성으로 이미 vendor 에 포함돼 있다.

;;; Code:

;;; org-mode (내장)
(use-package org
  :ensure nil
  :mode ("\\.org\\'" . org-mode)
  :custom
  (org-hide-leading-stars t)
  (org-startup-indented t)
  (org-adapt-indentation nil)
  (org-edit-src-content-indentation 0)
  (org-startup-truncated t))

;;; org-appear — 강조표시(*굵게* 등) 마크업을 커서가 닿을 때만 표시
(use-package org-appear
  :ensure t
  :hook (org-mode . org-appear-mode))

;;; markdown-toc — 마크다운 목차(TOC) 생성
(use-package markdown-toc
  :ensure t
  :commands (markdown-toc-generate-toc markdown-toc-refresh-toc))

(provide 'imoogi-org-markdown)
;;; 14-org-markdown.el ends here
