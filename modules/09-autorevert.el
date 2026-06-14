;;; 09-autorevert.el --- Auto-revert buffers on file change -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "09-autorevert" 'autorevert)

(use-package autorevert
  :ensure nil
  :config
  (setq auto-revert-use-notify t
        auto-revert-avoid-polling t
        auto-revert-check-vc-info nil
        auto-revert-verbose nil
        ;; Dired 등 비파일 버퍼도 자동 갱신
        global-auto-revert-non-file-buffers t)
  (global-auto-revert-mode 1))

;;; 09-autorevert.el ends here
