;;; anki-migrate-test.el --- the setup tail's migration flow -*- lexical-binding: t; -*-

;;; Commentary:

;; SPEC-ANKICARD-001 M6, the Elisp half of design.md SS7.1 steps 1-4 and 11:
;;
;;   dry-run -> N == 0 ? done : y-or-n-p naming N and the scheduling
;;   loss -> declined ? nothing : migrate -> write-back
;;
;; AC-C-018c is the load-bearing one: a declined confirmation must issue
;; no WRITING migrate request at all.  The fake runner records the argv
;; of every invocation, so "no writing request" is asserted over the
;; whole call log rather than inferred from a return value.
;;
;; REQ-C-020.3 is the other: a confirmed migration overwrites BOTH
;; ANKI_NOTE_ID (the new identifier the response reports) and
;; ANKI_NOTE_TYPE (the imoogi- counterpart, derived locally because the
;; response carries no note type).

;;; Code:

(require 'ert)
(require 'imoogi)
(require 'imoogi-setup)
(require 'imoogi-writeback)

(defun imoogi-migrate-test--response (results)
  "Serialize an ok response carrying RESULTS (list of alists)."
  (json-serialize (list (cons 'protocol_version 1)
                        (cons 'ok t)
                        (cons 'results (vconcat results))
                        (cons 'errors (vector)))))

(defun imoogi-migrate-test--candidate (key note-id)
  (list (cons 'key key) (cons 'action "migrate_candidate") (cons 'note_id note-id)))

(defun imoogi-migrate-test--added (key note-id)
  (list (cons 'key key) (cons 'action "added") (cons 'note_id note-id)))

(defmacro imoogi-migrate-test--with-root (var &rest body)
  "Create a temp sync root holding one stock-Basic heading, bind VAR to
it, and run BODY with the imoogi defcustoms pointed at it."
  (declare (indent 1))
  `(let* ((,var (make-temp-file "imoogi-migrate-root" t))
          (imoogi-sync-root ,var)
          (imoogi-default-deck "Default")
          (imoogi-exclude-patterns nil))
     (unwind-protect
         (progn
           (with-temp-file (expand-file-name "cards.org" ,var)
             (insert "* 첫 카드\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ANKI_NOTE_ID: 1001\n:END:\n\n본문\n"))
           ,@body)
       (delete-directory ,var t))))

(defvar imoogi-migrate-test--calls nil
  "Call log of the fake runner: a list of (ARGV . REQUEST-JSON), newest first.

A `defvar' rather than a `let*' binding inside each test: these files are
lexically bound, so a runner closure cannot reach a caller's lexical
variable through `set'/`symbol-value'.  Declaring it special makes the
`let' around each test a dynamic binding the closure genuinely shares.")

(defun imoogi-migrate-test--recording-runner (responses)
  "Return a runner that records each call in `imoogi-migrate-test--calls'
and answers from RESPONSES, an alist keyed by the argv list."
  (lambda (_binary request-json)
    (push (cons imoogi-process-argv request-json) imoogi-migrate-test--calls)
    (cons 0 (or (cdr (assoc imoogi-process-argv responses))
                (imoogi-migrate-test--response nil)))))

;; --- N == 0 -----------------------------------------------------------

(ert-deftest imoogi-migrate-test-zero-candidates-prompts-nothing ()
  "design.md SS7.1 step 2: a dry run reporting no candidates finishes
without a prompt and without a writing migrate request."
  (imoogi-migrate-test--with-root root
    (let* ((imoogi-migrate-test--calls nil)
           (imoogi-process-runner
            (imoogi-migrate-test--recording-runner
             (list (cons '("migrate" "--dry-run") (imoogi-migrate-test--response nil)))))
           (asked nil))
      (cl-letf (((symbol-function 'y-or-n-p) (lambda (&rest _) (setq asked t) t)))
        (let ((report (imoogi-setup--migrate-step "imoogi-anki" root)))
          (should-not asked)
          (should-not (assoc '("migrate") imoogi-migrate-test--calls))
          (should (string-match-p "0\\|없" report)))))))

;; --- N > 0, declined --------------------------------------------------

(ert-deftest imoogi-migrate-test-declined-issues-no-writing-request ()
  "AC-C-018c: the user declines, so no writing `migrate' request is
issued at all -- asserted over the whole call log, not inferred."
  (imoogi-migrate-test--with-root root
    (let* ((imoogi-migrate-test--calls nil)
           (imoogi-process-runner
            (imoogi-migrate-test--recording-runner
             (list (cons '("migrate" "--dry-run")
                                (imoogi-migrate-test--response
                                 (list (imoogi-migrate-test--candidate "cards.org::0" 1001)))))))
           (prompt nil))
      (cl-letf (((symbol-function 'y-or-n-p)
                 (lambda (p) (setq prompt p) nil)))
        (imoogi-setup--migrate-step "imoogi-anki" root))
      (should prompt)
      (should (equal (mapcar #'car imoogi-migrate-test--calls) '(("migrate" "--dry-run"))))
      (should-not (assoc '("migrate") imoogi-migrate-test--calls)))))

(ert-deftest imoogi-migrate-test-prompt-names-the-count-and-the-loss ()
  "REQ-C-019: the confirmation text names both the candidate count and
the loss of review history and scheduling state."
  (imoogi-migrate-test--with-root root
    (let* ((imoogi-migrate-test--calls nil)
           (imoogi-process-runner
            (imoogi-migrate-test--recording-runner
             (list (cons '("migrate" "--dry-run")
                                (imoogi-migrate-test--response
                                 (list (imoogi-migrate-test--candidate "cards.org::0" 1001)
                                       (imoogi-migrate-test--candidate "cards.org::1" 1002)))))))
           (prompt nil))
      (cl-letf (((symbol-function 'y-or-n-p) (lambda (p) (setq prompt p) nil)))
        (imoogi-setup--migrate-step "imoogi-anki" root))
      (should (string-match-p "2" prompt))
      (should (string-match-p "복습\\|이력\\|review" prompt))
      (should (string-match-p "일정\\|스케줄\\|scheduling" prompt)))))

;; --- N > 0, confirmed -------------------------------------------------

(ert-deftest imoogi-migrate-test-confirmed-issues-the-writing-request ()
  "design.md SS7.1 step 4: confirmation issues the same request document
again, this time without --dry-run."
  (imoogi-migrate-test--with-root root
    (let* ((imoogi-migrate-test--calls nil)
           (imoogi-process-runner
            (imoogi-migrate-test--recording-runner
             (list (cons '("migrate" "--dry-run")
                                (imoogi-migrate-test--response
                                 (list (imoogi-migrate-test--candidate "cards.org::0" 1001))))
                          (cons '("migrate")
                                (imoogi-migrate-test--response
                                 (list (imoogi-migrate-test--added "cards.org::0" 2002))))))))
      (cl-letf (((symbol-function 'y-or-n-p) (lambda (&rest _) t)))
        (imoogi-setup--migrate-step "imoogi-anki" root))
      (should (equal (mapcar #'car (reverse imoogi-migrate-test--calls))
                     '(("migrate" "--dry-run") ("migrate"))))
      ;; the two documents are the same sync-shaped request
      (should (equal (cdr (assoc '("migrate") imoogi-migrate-test--calls))
                     (cdr (assoc '("migrate" "--dry-run") imoogi-migrate-test--calls)))))))

(ert-deftest imoogi-migrate-test-confirmed-writes-back-both-properties ()
  "REQ-C-020.3: the heading's ANKI_NOTE_ID takes the new identifier and
its ANKI_NOTE_TYPE takes the imoogi- counterpart of the type it
currently declares."
  (imoogi-migrate-test--with-root root
    (let* ((imoogi-migrate-test--calls nil)
           (imoogi-process-runner
            (imoogi-migrate-test--recording-runner
             (list (cons '("migrate" "--dry-run")
                                (imoogi-migrate-test--response
                                 (list (imoogi-migrate-test--candidate "cards.org::0" 1001))))
                          (cons '("migrate")
                                (imoogi-migrate-test--response
                                 (list (imoogi-migrate-test--added "cards.org::0" 2002))))))))
      (cl-letf (((symbol-function 'y-or-n-p) (lambda (&rest _) t)))
        (imoogi-setup--migrate-step "imoogi-anki" root))
      (with-temp-buffer
        (insert-file-contents (expand-file-name "cards.org" root))
        (delay-mode-hooks (org-mode))
        (goto-char (point-min))
        (org-back-to-heading t)
        (should (equal (org-entry-get (point) "ANKI_NOTE_ID") "2002"))
        (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Basic"))))))

;; --- the counterpart derivation, on its own ---------------------------

(ert-deftest imoogi-migrate-test-counterpart-derivation ()
  "REQ-C-020.3 derives the counterpart locally: a stock name gains the
prefix, an already-imoogi name is left alone (the migration of a
heading the user had already hand-edited is idempotent)."
  (should (equal (imoogi-writeback-note-type-counterpart "Basic") "imoogi-Basic"))
  (should (equal (imoogi-writeback-note-type-counterpart "Cloze") "imoogi-Cloze"))
  (should (equal (imoogi-writeback-note-type-counterpart "imoogi-Basic") "imoogi-Basic"))
  (should (equal (imoogi-writeback-note-type-counterpart "imoogi-Cloze") "imoogi-Cloze")))

(ert-deftest imoogi-migrate-test-writeback-apply-still-writes-id-only ()
  "The ordinary sync write-back is unchanged: it writes ANKI_NOTE_ID
and leaves ANKI_NOTE_TYPE exactly as the heading declares it, so a
hand-written stock heading is not silently re-typed by a sync run."
  (imoogi-migrate-test--with-root root
    (with-temp-file (expand-file-name "cards.org" root)
      (insert "* 첫 카드\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n\n본문\n"))
    (imoogi-writeback-apply root (list (list :key "cards.org::0" :action "added" :note-id 4004)))
    (with-temp-buffer
      (insert-file-contents (expand-file-name "cards.org" root))
      (delay-mode-hooks (org-mode))
      (goto-char (point-min))
      (org-back-to-heading t)
      (should (equal (org-entry-get (point) "ANKI_NOTE_ID") "4004"))
      (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "Basic")))))

(provide 'anki-migrate-test)
;;; anki-migrate-test.el ends here
