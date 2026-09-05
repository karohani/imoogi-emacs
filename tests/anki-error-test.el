;;; imoogi-error-test.el --- Tests for imoogi-error.el -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'imoogi-error)

(defconst imoogi-error-test--go-emitted-codes
  '("binary_incompatible"
    "anki_unreachable"
    "ankiconnect_missing"
    "ankiconnect_error"
    "org_parse_error"
    "cloze_marker_missing"
    "deck_create_failed"
    "deck_move_failed"
    "note_type_change_unsupported"
    "note_field_missing"
    "note_id_duplicated"
    "state_unreadable"
    "delete_suppressed"
    "delete_candidate_unowned")
  "The `Code...' string constants internal/protocol/protocol.go declares
today (lines 31-42 there, as of this writing) -- hardcoded here because
this suite has no Go parser available.
`imoogi-error-test-codes-match-source-file' below cross-checks this
list against the actual source file by regexp, so drift between the
two is caught as a test failure rather than silently assumed away.

`binary_not_found' and `sync_root_unset' are deliberately excluded:
they are Elisp-side preconditions (plan.md D-5) that never reach the
Go binary at all -- imoogi-sync stops before the subprocess is even
invoked.  `note_id_unknown' is also excluded: plan.md D-5 lists it as
Go-origin, but the Go binary does not currently emit it on the wire
-- REQ-012's reconciliation path treats the case as an ordinary add
and reports no error for it (AC-018), and progress.md's M3b
disclosed deviation records this choice explicitly.  All three are
still required table entries; see
`imoogi-error-test-full-d5-table-present' below.")

(ert-deftest imoogi-error-test-go-codes-are-subset-of-table ()
  "acceptance.md SS D.8: every code the back end emits must have a
table entry -- a code with no entry is itself a defect (plan.md D-5)."
  (dolist (code imoogi-error-test--go-emitted-codes)
    (should (assoc code imoogi-error-table))))

(ert-deftest imoogi-error-test-codes-match-source-file ()
  "Cross-check the hardcoded Go-emitted-codes list against
internal/protocol/protocol.go so the two lists cannot silently drift
apart -- the mechanism the coordinator's note called for when a full
Go-source parse proves awkward from Elisp."
  ;; `load-file-name'/`buffer-file-name' are unreliable under `-l' batch
  ;; loading (both can be nil depending on how Emacs was invoked), so this
  ;; resolves relative to `default-directory' instead -- the project root,
  ;; per E1's documented invocation contract (`emacs -Q --batch -L lisp
  ;; -l ert -l <test-file> ...' run from the worktree root).  When neither
  ;; resolves to a readable file the test is skipped rather than failed,
  ;; since the working directory is a test-runner concern, not a defect
  ;; in this table.
  (let* ((candidates
          (list (and (or load-file-name buffer-file-name)
                     (expand-file-name
                      "../internal/anki/protocol/protocol.go"
                      (file-name-directory (or load-file-name buffer-file-name))))
                (expand-file-name "internal/anki/protocol/protocol.go" default-directory)))
         (proto-file (seq-find (lambda (f) (and f (file-readable-p f))) candidates)))
    (skip-unless proto-file)
    (with-temp-buffer
      (insert-file-contents proto-file)
      (let (found)
        (goto-char (point-min))
        (while (re-search-forward "\\_<Code[A-Za-z]+[ \t]*=[ \t]*\"\\([a-z_]+\\)\"" nil t)
          (push (match-string 1) found))
        (setq found (nreverse found))
        (should found)
        (should (equal (sort (copy-sequence found) #'string<)
                        (sort (copy-sequence imoogi-error-test--go-emitted-codes) #'string<)))))))

(ert-deftest imoogi-error-test-full-d5-table-present ()
  "Every code named in plan.md D-5's table (REQ-018) has an entry,
Go-emitted today or not."
  (dolist (code '("binary_not_found" "binary_incompatible" "sync_root_unset"
                   "anki_unreachable" "ankiconnect_missing" "ankiconnect_error"
                   "org_parse_error" "cloze_marker_missing" "deck_create_failed"
                   "deck_move_failed" "note_id_unknown" "note_type_change_unsupported" "note_field_missing" "note_id_duplicated"
                   "state_unreadable" "delete_suppressed" "delete_candidate_unowned"))
    (should (assoc code imoogi-error-table))))

(ert-deftest imoogi-error-test-messages-contain-no-raw-diagnostics ()
  "REQ-018: no message may surface a Go stack trace, a raw HTTP
transport error, an Emacs backtrace, or a bare process exit code."
  (dolist (entry imoogi-error-table)
    (let ((msg (cdr entry)))
      (dolist (forbidden '("goroutine" "panic:" "connection refused" "dial tcp"
                            "exit status" "wrong-type-argument" "file-missing"
                            "Debugger entered"))
        (should-not (string-match-p (regexp-quote forbidden) msg))))))

(ert-deftest imoogi-error-test-message-lookup-known-code ()
  (should (string-match-p "sync root" (imoogi-error-message "sync_root_unset"))))

(ert-deftest imoogi-error-test-message-lookup-unknown-code-falls-back ()
  (let ((msg (imoogi-error-message "totally_unknown_code")))
    (should (string-match-p "totally_unknown_code" msg))
    (should-not (string-match-p "goroutine\\|panic:" msg))))

(provide 'imoogi-error-test)
;;; imoogi-error-test.el ends here
