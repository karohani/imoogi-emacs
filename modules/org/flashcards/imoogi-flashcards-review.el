;;; imoogi-flashcards-review.el --- Review UI -*- lexical-binding: t; -*-

;;; Code:

(require 'cl-lib)
(require 'subr-x)
(require 'imoogi-flashcards-core)
(require 'imoogi-flashcards-org)
(require 'imoogi-flashcards-repository)

(defvar-local imoogi-flashcards-review--db nil)
(defvar-local imoogi-flashcards-review--queue nil)
(defvar-local imoogi-flashcards-review--current nil)
(defvar-local imoogi-flashcards-review--current-card nil)
(defvar-local imoogi-flashcards-review--showing-answer nil)

(defvar imoogi-flashcards-review-mode-map
  (let ((map (make-sparse-keymap)))
    (define-key map (kbd "SPC") #'imoogi-flashcards-review-show-answer)
    (define-key map (kbd "RET") #'imoogi-flashcards-review-show-answer)
    (define-key map (kbd "1") #'imoogi-flashcards-review-again)
    (define-key map (kbd "2") #'imoogi-flashcards-review-hard)
    (define-key map (kbd "3") #'imoogi-flashcards-review-good)
    (define-key map (kbd "4") #'imoogi-flashcards-review-easy)
    (define-key map (kbd "q") #'imoogi-flashcards-review-quit)
    map)
  "Keymap for flashcard review buffers.")

(define-derived-mode imoogi-flashcards-review-mode special-mode "Flashcards"
  "Major mode for local flashcard review."
  (add-hook 'kill-buffer-hook #'imoogi-flashcards-review--close-db nil t))

(defun imoogi-flashcards-review--close-db ()
  "Close the buffer-local SQLite connection, if one is open."
  (when imoogi-flashcards-review--db
    (ignore-errors
      (imoogi-flashcards-db-close imoogi-flashcards-review--db))
    (setq imoogi-flashcards-review--db nil)))

(defun imoogi-flashcards-review-quit ()
  "Close the review DB connection and quit the review window."
  (interactive)
  (imoogi-flashcards-review--close-db)
  (quit-window t))

(defun imoogi-flashcards-review--insert-current ()
  "Render the current flashcard review state."
  (let ((inhibit-read-only t)
        (row imoogi-flashcards-review--current))
    (erase-buffer)
    (if (null row)
        (insert "No due flashcards.\n")
      (let ((key (nth 0 row))
            (kind (nth 2 row))
            (source (nth 4 row)))
        (setq imoogi-flashcards-review--current-card
              (imoogi-flashcards-resolve-card source key))
        (insert (format "%s  %s\n%s\n\n" kind key (make-string 60 ?-)))
        (insert (format "%s\n\n"
                        (imoogi-flashcards-card-front
                         imoogi-flashcards-review--current-card)))
        (when imoogi-flashcards-review--showing-answer
          (insert (format "%s\n\n%s\n\n"
                          (make-string 60 ?-)
                          (imoogi-flashcards-card-back
                           imoogi-flashcards-review--current-card)))
          (insert "[1] Again   [2] Hard   [3] Good   [4] Easy\n"))
        (unless imoogi-flashcards-review--showing-answer
          (insert "SPC/RET: show answer\n"))
        (insert (format "\nSource: %s :: %s\n"
                        source
                        (imoogi-flashcards-card-heading
                         imoogi-flashcards-review--current-card))))))
  (goto-char (point-min)))

(defun imoogi-flashcards-review--advance ()
  "Advance the review queue."
  (setq imoogi-flashcards-review--current (pop imoogi-flashcards-review--queue)
        imoogi-flashcards-review--showing-answer nil)
  (imoogi-flashcards-review--insert-current))

(defun imoogi-flashcards-review-show-answer ()
  "Reveal the answer for the current card."
  (interactive)
  (unless imoogi-flashcards-review--current
    (user-error "imoogi flashcards: due 카드가 없습니다"))
  (setq imoogi-flashcards-review--showing-answer t)
  (imoogi-flashcards-review--insert-current))

(defun imoogi-flashcards-review-rate (rating)
  "Record RATING for the current card."
  (interactive
   (list (intern (completing-read "Rating: " '("again" "hard" "good" "easy")
                                  nil t))))
  (unless imoogi-flashcards-review--current
    (user-error "imoogi flashcards: due 카드가 없습니다"))
  (unless imoogi-flashcards-review--showing-answer
    (user-error "imoogi flashcards: 먼저 답을 확인하세요"))
  (let ((key (nth 0 imoogi-flashcards-review--current)))
    (imoogi-flashcards-repo-review imoogi-flashcards-review--db
                                   key rating (float-time))
    (message "imoogi flashcards: %s" (imoogi-flashcards-rating-name rating))
    (imoogi-flashcards-review--advance)))

(defun imoogi-flashcards-review-again () (interactive) (imoogi-flashcards-review-rate 'again))
(defun imoogi-flashcards-review-hard () (interactive) (imoogi-flashcards-review-rate 'hard))
(defun imoogi-flashcards-review-good () (interactive) (imoogi-flashcards-review-rate 'good))
(defun imoogi-flashcards-review-easy () (interactive) (imoogi-flashcards-review-rate 'easy))

(defun imoogi-flashcards-review-start (&optional limit)
  "Open a review buffer for due flashcards, bounded by LIMIT."
  (interactive "P")
  (let ((db (imoogi-flashcards-db-open)))
    (condition-case err
        (let ((rows (imoogi-flashcards-repo-due-cards
                     db (float-time)
                     (or (and limit (prefix-numeric-value limit)) 50)))
              (buf (get-buffer-create "*imoogi flashcards*")))
          (with-current-buffer buf
            (imoogi-flashcards-review--close-db)
            (imoogi-flashcards-review-mode)
            (setq imoogi-flashcards-review--db db
                  imoogi-flashcards-review--queue rows)
            (imoogi-flashcards-review--advance))
          (pop-to-buffer buf))
      (error
       (when (and (buffer-live-p (get-buffer "*imoogi flashcards*"))
                  (eq (buffer-local-value
                       'imoogi-flashcards-review--db
                       (get-buffer "*imoogi flashcards*"))
                      db))
         (with-current-buffer "*imoogi flashcards*"
           (setq imoogi-flashcards-review--db nil)))
       (ignore-errors (imoogi-flashcards-db-close db))
       (signal (car err) (cdr err))))))

(provide 'imoogi-flashcards-review)
;;; imoogi-flashcards-review.el ends here
