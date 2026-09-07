;;; 14-org.el --- org 설정 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; org-mode(내장) 기본 설정 + org-appear.

;;; Code:

(imoogi-require "14-org" 'org 'org-appear 'hl-line)

(defvar imoogi-org-lisp-dir
  (expand-file-name "org/" (file-name-directory (or load-file-name buffer-file-name)))
  "Directory holding extra Org display implementation files.")

(add-to-list 'load-path imoogi-org-lisp-dir)

(require 'imoogi-org-border)

;;;###autoload
(defun imoogi-org-setup ()
  "Create ~/notes and use it as the default Org directory.
Existing notes are preserved.  This command does not run Anki setup or sync."
  (interactive)
  (require 'org)
  (let ((directory (expand-file-name "~/notes/")))
    (make-directory directory t)
    (setq org-directory directory)
    (message "imoogi: 기본 Org 폴더: %s" directory)
    directory))

(defun imoogi-org-hl-line-range ()
  "제목 줄의 색을 가리지 않도록 본문에서만 현재 줄을 강조한다."
  (unless (org-at-heading-p)
    (cons (line-beginning-position)
          (min (point-max) (1+ (line-end-position))))))

(defun imoogi-org-heading-setup ()
  "제목 배경을 줄 끝까지 적용하고 기존 버퍼의 표시도 갱신한다."
  (setq-local org-fontify-whole-heading-line t)
  (setq-local hl-line-range-function #'imoogi-org-hl-line-range)
  (setq-local org-ellipsis " ▼")
  (setq-local buffer-display-table
              (copy-sequence (or buffer-display-table (make-display-table))))
  (set-display-table-slot
   buffer-display-table 4
   (vconcat (mapcar (lambda (char) (make-glyph-code char 'org-ellipsis))
                   org-ellipsis)))
  (dolist (alias '(outline block drawer))
    (org-fold-core-set-folding-spec-property
     (org-fold-core-get-folding-spec-from-alias alias) :ellipsis org-ellipsis))
  (when (fboundp 'hl-line-unhighlight) (hl-line-unhighlight))
  (when (fboundp 'global-hl-line-unhighlight) (global-hl-line-unhighlight))
  (when (and imoogi-org-border-enabled
             (not (local-variable-p 'imoogi-org-border-mode)))
    (imoogi-org-border-mode 1))
  (font-lock-flush))

;;; org-mode (내장)
(use-package org
  :ensure nil
  :mode ("\\.org\\'" . org-mode)
  :hook (org-mode . imoogi-org-heading-setup)
  :custom
  (org-directory (expand-file-name "~/notes/"))
  (org-hide-leading-stars t)
  (org-startup-indented t)
  (org-adapt-indentation nil)
  (org-edit-src-content-indentation 0)
  (org-startup-truncated t)
  (org-fontify-whole-heading-line t)
  (org-cycle-level-faces t)
  (org-ellipsis " ▼")
  :config
  ;; 빨강 → 파랑 → 초록 → 노랑. :extend 로 제목 뒤 빈 공간까지 칠한다.
  ;; user 테마에 등록해 테마 재적용과 새 프레임에서도 유지한다.
  (let ((palette '(("#ff8c92" "#3b2930" "#a12635" "#fbe9ec")
                   ("#82b7ff" "#253449" "#205da8" "#e8f0fc")
                   ("#a5d67d" "#2c392b" "#386b23" "#edf5e7")
                   ("#f2d479" "#3d3726" "#806000" "#faf3d9"))))
    (dotimes (index 8)
      (let ((colors (nth (% index 4) palette)))
        (custom-theme-set-faces
         'user
         `(,(intern (format "org-level-%d" (1+ index)))
           ((((background light))
             (:foreground ,(nth 2 colors) :background ,(nth 3 colors)
              :weight bold :extend t))
            (t (:foreground ,(nth 0 colors) :background ,(nth 1 colors)
                :weight bold :extend t))))))))
  ;; 설정 재로드 시 이미 열려 있던 Org 버퍼도 줄 전체를 다시 칠한다.
  (dolist (buffer (buffer-list))
    (with-current-buffer buffer
      (when (derived-mode-p 'org-mode)
        (imoogi-org-heading-setup)))))

;;; org-appear — 강조표시(*굵게* 등) 마크업을 커서가 닿을 때만 표시
(use-package org-appear
  :ensure t
  :hook (org-mode . org-appear-mode))

(provide 'imoogi-org)
;;; 14-org.el ends here
