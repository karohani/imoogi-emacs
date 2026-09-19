;;; git.el --- Magit -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "06-git" 'magit 'diff-hl)

(use-package magit
  :ensure t)

;;; diff-hl — 버퍼 여백에 커밋되지 않은 Git 변경 표시
(use-package diff-hl
  :ensure t
  :hook ((prog-mode . diff-hl-mode)
         (dired-mode . diff-hl-dired-mode)
         (magit-pre-refresh . diff-hl-magit-pre-refresh)
         (magit-post-refresh . diff-hl-magit-post-refresh))
  :init
  (setq diff-hl-flydiff-delay 0.4
        diff-hl-show-staged-changes nil
        diff-hl-update-async t
        diff-hl-global-modes '(not pdf-view-mode image-mode)))

(provide 'imoogi-git)
;;; git.el ends here
