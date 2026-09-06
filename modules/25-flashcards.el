;;; 25-flashcards.el --- Emacs-native flashcard fallback -*- lexical-binding: t; -*-

;;; Commentary:

;; Local-only flashcards for air-gapped machines where Anki is unavailable.
;; This deliberately does not share state with modules/24-anki.el.  Org is the
;; source of card content and SQLite holds a local projection, review state, and
;; immutable review log.

;;; Code:

(imoogi-require "25-flashcards" 'org 'sqlite 'seq)

;; Deferred menu registration still needs its macro during byte compilation.
(eval-when-compile (require 'transient))

(require 'sqlite)
(unless (sqlite-available-p)
  (error "[25-flashcards] sqlite module is present but SQLite support is unavailable"))

(defvar imoogi-flashcards-lisp-dir
  (expand-file-name "modules/flashcards/" imoogi-emacs-dir)
  "Directory holding the Emacs-native flashcards implementation.")

(add-to-list 'load-path imoogi-flashcards-lisp-dir)

(require 'imoogi-flashcards-core)
(require 'imoogi-flashcards-org)
(require 'imoogi-flashcards-repository)
(require 'imoogi-flashcards-review)

;;; Completion candidates

(with-eval-after-load 'org
  (dolist (name (list imoogi-flashcards-id-property
                      imoogi-flashcards-kind-property))
    (add-to-list 'org-default-properties name t))
  (add-to-list 'org-global-properties
               (cons (concat imoogi-flashcards-kind-property "_ALL")
                     "Basic Cloze")))

;;; Commands

(defun imoogi-flashcards-sync-root (root)
  "Project flashcards below ROOT into the local SQLite repository."
  (interactive "DFlashcard Org root: ")
  (let* ((scan (imoogi-flashcards-scan-root root))
         (duplicates (plist-get scan :duplicate-note-ids)))
    (when duplicates
      (user-error "imoogi flashcards: 중복 %s 값: %s"
                  imoogi-flashcards-id-property
                  (string-join duplicates ", ")))
    (let ((db (imoogi-flashcards-db-open)))
      (unwind-protect
          (let ((keys (imoogi-flashcards-repo-sync-cards
                       db (plist-get scan :cards) (float-time))))
            (message "imoogi flashcards: %d개 카드를 동기화했습니다" (length keys))
            keys)
        (imoogi-flashcards-db-close db)))))

(defun imoogi-flashcards-sync-buffer ()
  "Project flashcards in the current Org buffer file into SQLite."
  (interactive)
  (unless buffer-file-name
    (user-error "imoogi flashcards: 파일에 저장된 Org 버퍼에서만 동기화할 수 있습니다"))
  (let* ((source-file (expand-file-name buffer-file-name))
         (scan (imoogi-flashcards-scan-file source-file))
         (duplicates (plist-get scan :duplicate-note-ids)))
    (when duplicates
      (user-error "imoogi flashcards: 중복 %s 값: %s"
                  imoogi-flashcards-id-property
                  (string-join duplicates ", ")))
    (let ((db (imoogi-flashcards-db-open)))
      (unwind-protect
          (let ((keys (imoogi-flashcards-repo-sync-cards
                       db (plist-get scan :cards) (float-time) source-file)))
            (message "imoogi flashcards: 현재 파일에서 %d개 카드를 동기화했습니다"
                     (length keys))
            keys)
        (imoogi-flashcards-db-close db)))))

(defalias 'imoogi-flashcards-review #'imoogi-flashcards-review-start)

;;; Keys

(defvar imoogi-flashcards-map
  (let ((map (make-sparse-keymap)))
    (define-key map (kbd "b") #'imoogi-flashcards-mark-basic)
    (define-key map (kbd "c") #'imoogi-flashcards-mark-cloze)
    (define-key map (kbd "x") #'imoogi-flashcards-unmark)
    (define-key map (kbd "z") #'imoogi-flashcards-cloze-region)
    (define-key map (kbd "s") #'imoogi-flashcards-sync-buffer)
    (define-key map (kbd "S") #'imoogi-flashcards-sync-root)
    (define-key map (kbd "r") #'imoogi-flashcards-review)
    map)
  "Local flashcards command prefix map.  Org buffers bind it at C-c f.")

(with-eval-after-load 'org
  (define-key org-mode-map (kbd "C-c f") imoogi-flashcards-map))

;;; Transient menu

(with-eval-after-load 'imoogi-transient
  (transient-define-prefix imoogi-flashcards-transient ()
    "Emacs-native flashcards."
    :column-widths '(20 24 20)
    [["표시 -------------"
      ("b" "Basic 카드로" imoogi-flashcards-mark-basic)
      ("c" "Cloze 카드로" imoogi-flashcards-mark-cloze)
      ("x" "표시 해제" imoogi-flashcards-unmark)]
     ["내용/동기화 ----------"
      ("z" "선택 영역을 빈칸으로" imoogi-flashcards-cloze-region)
      ("s" "현재 파일 동기화" imoogi-flashcards-sync-buffer)
      ("S" "루트 동기화" imoogi-flashcards-sync-root)]
     ["복습 -----------"
      ("r" "due 복습" imoogi-flashcards-review)
      ("q" "종료" transient-quit-one)]])

  (define-key imoogi-flashcards-map (kbd "f") #'imoogi-flashcards-transient)

  (transient-append-suffix 'imoogi-transient-master "a"
    '("f" "Flashcards" imoogi-flashcards-transient)))

(provide 'imoogi-flashcards)
;;; 25-flashcards.el ends here
