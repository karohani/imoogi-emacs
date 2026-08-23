;;; 14-org.el --- org 설정 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; org-mode(내장) 기본 설정 + org-appear.

;;; Code:

(imoogi-require "14-org" 'org 'org-appear)

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

(provide 'imoogi-org)
;;; 14-org.el ends here
