;;; imoogi-flashcards-core.el --- Core model and scheduler -*- lexical-binding: t; -*-

;;; Code:

(require 'cl-lib)

(defconst imoogi-flashcards-ratings '(again hard good easy)
  "Supported review ratings, ordered from worst to best.")

(defconst imoogi-flashcards-default-ease 2.5
  "Initial ease factor for new cards.")

(defconst imoogi-flashcards-min-ease 1.3
  "Lower bound for ease factor.")

(defconst imoogi-flashcards-max-ease 3.0
  "Upper bound for ease factor.")

(defconst imoogi-flashcards-statuses '(new learning review)
  "Supported scheduling statuses.")

(cl-defstruct imoogi-flashcards-card
  key note-id kind cloze-no source-file heading front back hash)

(cl-defstruct (imoogi-flashcards-state
               (:constructor imoogi-flashcards-state-create))
  (status 'new)
  (reps 0)
  (lapses 0)
  (interval 0)
  (ease imoogi-flashcards-default-ease)
  (due-at 0)
  last-reviewed-at)

(defun imoogi-flashcards--round-days (days)
  "Return DAYS rounded half-up to at least one whole day."
  (max 1 (floor (+ days 0.5))))

(defun imoogi-flashcards--clamp-ease (ease)
  "Clamp EASE into the supported SM-2-family bounds."
  (min imoogi-flashcards-max-ease
       (max imoogi-flashcards-min-ease ease)))

(defun imoogi-flashcards-schedule (state rating now)
  "Return next review STATE after RATING at NOW.

This is an intentionally small SM-2-family table with explicit states:

new/learning Again: stay learning, due in 10 minutes, no lapse.
new/learning Hard:  stay learning, due in 1 day.
new/learning Good:  graduate to review, due in 1 day.
new/learning Easy:  graduate to review, due in 4 days.
review Again:       enter learning, due in 1 day, lapse +1.
review Hard:        stay review, previous interval * 1.2.
review Good:        stay review, previous interval * ease.
review Easy:        stay review, previous interval * ease * 1.3.

Day intervals use round-half-up, and ease is clamped to 1.3..3.0."
  (unless (memq rating imoogi-flashcards-ratings)
    (error "Unknown flashcard rating: %S" rating))
  (let* ((old-reps (or (imoogi-flashcards-state-reps state) 0))
         (old-interval (or (imoogi-flashcards-state-interval state) 0))
         (old-ease (or (imoogi-flashcards-state-ease state)
                       imoogi-flashcards-default-ease))
         (old-lapses (or (imoogi-flashcards-state-lapses state) 0))
         (old-status (or (imoogi-flashcards-state-status state) 'new))
         (learning (memq old-status '(new learning)))
         status reps interval ease lapses due-delay)
    (unless (memq old-status imoogi-flashcards-statuses)
      (error "Unknown flashcard status: %S" old-status))
    (if learning
        (pcase rating
          ('again
           (setq status 'learning
                 reps 0
                 interval 0
                 ease old-ease
                 lapses old-lapses
                 due-delay (* 10 60)))
          ('hard
           (setq status 'learning
                 reps (1+ old-reps)
                 interval 1
                 ease (imoogi-flashcards--clamp-ease (- old-ease 0.15))
                 lapses old-lapses
                 due-delay 86400))
          ('good
           (setq status 'review
                 reps (1+ old-reps)
                 interval 1
                 ease old-ease
                 lapses old-lapses
                 due-delay 86400))
          ('easy
           (setq status 'review
                 reps (1+ old-reps)
                 interval 4
                 ease (imoogi-flashcards--clamp-ease (+ old-ease 0.15))
                 lapses old-lapses
                 due-delay (* 4 86400))))
      (pcase rating
        ('again
         (setq status 'learning
               reps 0
               interval 1
               ease (imoogi-flashcards--clamp-ease (- old-ease 0.20))
               lapses (1+ old-lapses)
               due-delay 86400))
        ('hard
         (setq status 'review
               reps (1+ old-reps)
               interval (imoogi-flashcards--round-days
                         (* (max 1 old-interval) 1.2))
               ease (imoogi-flashcards--clamp-ease (- old-ease 0.15))
               lapses old-lapses
               due-delay (* interval 86400)))
        ('good
         (setq status 'review
               reps (1+ old-reps)
               interval (imoogi-flashcards--round-days
                         (* (max 1 old-interval) old-ease))
               ease old-ease
               lapses old-lapses
               due-delay (* interval 86400)))
        ('easy
         (setq status 'review
               reps (1+ old-reps)
               interval (imoogi-flashcards--round-days
                         (* (max 1 old-interval) old-ease 1.3))
               ease (imoogi-flashcards--clamp-ease (+ old-ease 0.15))
               lapses old-lapses
               due-delay (* interval 86400)))))
    (imoogi-flashcards-state-create
     :status status
     :reps reps
     :lapses lapses
     :interval interval
     :ease ease
     :last-reviewed-at now
     :due-at (+ now due-delay))))

(defun imoogi-flashcards-rating-name (rating)
  "Return a user-facing name for RATING."
  (pcase rating
    ('again "Again")
    ('hard "Hard")
    ('good "Good")
    ('easy "Easy")
    (_ (symbol-name rating))))

(provide 'imoogi-flashcards-core)
;;; imoogi-flashcards-core.el ends here
