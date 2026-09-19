;;; imoogi-config.el --- Configuration file read and write -*- lexical-binding: t; -*-

;;; Commentary:

;; plan.md D-11: configuration is owned by Elisp; the Go binary reads no
;; configuration file of its own and consults no environment for its
;; behavior.  This file is the ONLY writer of `imoogi-config-file' --
;; `imoogi-anki-setup' (imoogi-setup.el) is its only caller for writing
;; -- and the reader that populates `imoogi-sync-root' /
;; `imoogi-default-deck' at startup when the file exists.
;;
;; design.md SS2.5 scopes the file's contents to exactly the two values
;; `imoogi-anki-setup' prompts for: the sync root and the default deck.
;; `imoogi-binary-path' and `imoogi-exclude-patterns' are ordinary
;; defcustoms and are never part of this file.
;;
;; `imoogi-sync-root' and `imoogi-default-deck' are defined as defcustoms
;; in imoogi.el, which requires this file -- forward-declared here rather
;; than required, to avoid a circular `require'.

(require 'json)

;;; Code:

;; A value-less `(defvar SYM)' only suppresses byte-compiler warnings
;; within THIS file -- it does not reliably mark SYM special at runtime
;; for other files that `let'-bind it (confirmed by direct repro).  A
;; nil-valued forward declaration does mark it special globally, and is
;; a no-op once imoogi.el's own `defcustom' for the same symbol runs,
;; since neither form overwrites an already-bound value.
(defvar imoogi-sync-root nil)
(defvar imoogi-default-deck nil)
(defvar imoogi-config-file nil)

(defun imoogi-config-write (file sync-root default-deck)
  "Write FILE with exactly SYNC-ROOT and DEFAULT-DECK (design.md SS2.5,
plan.md D-11).  The only caller is `imoogi-anki-setup'."
  (with-temp-file file
    (insert (json-serialize
             (list (cons 'sync_root sync-root)
                   (cons 'default_deck default-deck))))))

(defun imoogi-config-load (&optional file)
  "Read FILE (default `imoogi-config-file') and, when it exists, set
`imoogi-sync-root' and `imoogi-default-deck' from its contents.

A missing file is a normal first-run state -- REQ-017's `imoogi-anki-setup'
has simply never run yet -- and this is a silent no-op, returning nil.
Returns non-nil when the defcustoms were updated."
  (let ((path (or file imoogi-config-file)))
    (when (and path (file-exists-p path))
      (let* ((parsed (with-temp-buffer
                        (insert-file-contents path)
                        (json-parse-string (buffer-string) :object-type 'alist)))
             (root (cdr (assq 'sync_root parsed)))
             (deck (cdr (assq 'default_deck parsed))))
        (when root
          (setq imoogi-sync-root root))
        (when deck
          (setq imoogi-default-deck deck))
        t))))

(provide 'imoogi-config)
;;; imoogi-config.el ends here
