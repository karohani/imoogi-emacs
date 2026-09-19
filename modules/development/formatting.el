;;; formatting.el --- Shared code formatting -*- lexical-binding: t; -*-
;;; Code:
(imoogi-require "formatting")

(defun imoogi-format-code ()
  "Format the active region or current buffer for its major mode.
Prefer an active Eglot server, then a bundled language formatter, and finally
fall back to the current major mode's indentation rules."
  (interactive)
  (cond
   ((and (fboundp 'eglot-managed-p)
         (eglot-managed-p)
         (fboundp 'eglot-format))
    (call-interactively #'eglot-format))
   ((and (derived-mode-p 'go-mode 'go-ts-mode)
         (fboundp 'gofmt)
         (executable-find (if (boundp 'gofmt-command)
                              gofmt-command
                            "gofmt")))
    (gofmt))
   ((and (derived-mode-p 'rust-mode 'rust-ts-mode)
         (fboundp 'rust-format-buffer)
         (executable-find "rustfmt"))
    (rust-format-buffer))
   ((and (derived-mode-p 'json-mode 'js-json-mode 'json-ts-mode)
         (fboundp 'json-pretty-print-buffer))
    (json-pretty-print-buffer))
   (t
    (indent-region (if (use-region-p) (region-beginning) (point-min))
                   (if (use-region-p) (region-end) (point-max))))))

(provide 'imoogi-formatting)
;;; formatting.el ends here
