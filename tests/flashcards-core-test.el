;;; flashcards-core-test.el --- local flashcards scheduler tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'imoogi-flashcards-core)

(ert-deftest imoogi-flashcards-scheduler-new-card-table ()
  (let* ((now 1000)
         (state (imoogi-flashcards-state-create))
         (again (imoogi-flashcards-schedule state 'again now))
         (hard (imoogi-flashcards-schedule state 'hard now))
         (good (imoogi-flashcards-schedule state 'good now))
         (easy (imoogi-flashcards-schedule state 'easy now)))
    (should (= (imoogi-flashcards-state-reps again) 0))
    (should (eq (imoogi-flashcards-state-status again) 'learning))
    (should (= (imoogi-flashcards-state-lapses again) 0))
    (should (= (imoogi-flashcards-state-interval again) 0))
    (should (= (imoogi-flashcards-state-due-at again) (+ now 600)))
    (should (eq (imoogi-flashcards-state-status hard) 'learning))
    (should (= (imoogi-flashcards-state-interval hard) 1))
    (should (eq (imoogi-flashcards-state-status good) 'review))
    (should (= (imoogi-flashcards-state-interval good) 1))
    (should (eq (imoogi-flashcards-state-status easy) 'review))
    (should (= (imoogi-flashcards-state-interval easy) 4))
    (should (= (imoogi-flashcards-state-due-at good) (+ now 86400)))
    (should (< (imoogi-flashcards-state-ease hard) 2.5))
    (should (> (imoogi-flashcards-state-ease easy) 2.5))))

(ert-deftest imoogi-flashcards-scheduler-mature-card-table ()
  (let* ((now 1000)
         (state (imoogi-flashcards-state-create
                 :status 'review :reps 2 :lapses 1 :interval 10 :ease 2.0
                 :due-at now))
         (again (imoogi-flashcards-schedule state 'again now))
         (hard (imoogi-flashcards-schedule state 'hard now))
         (good (imoogi-flashcards-schedule state 'good now))
         (easy (imoogi-flashcards-schedule state 'easy now)))
    (should (eq (imoogi-flashcards-state-status again) 'learning))
    (should (= (imoogi-flashcards-state-lapses again) 2))
    (should (= (imoogi-flashcards-state-interval again) 1))
    (should (= (imoogi-flashcards-state-interval hard) 12))
    (should (= (imoogi-flashcards-state-interval good) 20))
    (should (= (imoogi-flashcards-state-interval easy) 26))
    (should (= (imoogi-flashcards-state-lapses good) 1))))

(ert-deftest imoogi-flashcards-scheduler-rounds-half-up-and-clamps-ease ()
  (let* ((now 1000)
         (state (imoogi-flashcards-state-create
                 :status 'review :reps 2 :interval 3 :ease 1.5))
         (good (imoogi-flashcards-schedule state 'good now))
         (easy (imoogi-flashcards-schedule
                (imoogi-flashcards-state-create
                 :status 'review :reps 2 :interval 10 :ease 2.95)
                'easy now)))
    (should (= (imoogi-flashcards-state-interval good) 5))
    (should (= (imoogi-flashcards-state-ease easy) 3.0))))

(provide 'flashcards-core-test)
;;; flashcards-core-test.el ends here
