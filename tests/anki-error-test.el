;;; imoogi-error-test.el --- Tests for imoogi-error.el -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'seq)
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
    "delete_candidate_unowned"
    "model_install_failed"
    "media_file_not_found"
    "media_upload_failed"
    "migration_add_failed"
    "card_option_invalid"
    "card_option_conflict"
    "card_option_needs_cloze"
    "multiline_answer_missing"
    "extra_block_unbalanced"
    "swift_arrow_missing"
    "org_id_invalid")
  "The `Code...' string constants internal/anki/protocol/protocol.go
declares today -- hardcoded here because this suite has no Go parser
available.  Each later SPEC appends its own codes here in the order
protocol.go declares them.
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

(defconst imoogi-error-test--elisp-only-codes
  '("binary_not_found" "sync_root_unset" "note_id_unknown"
    "org_id_duplicated" "org_id_create_failed")
  "Table entries that legitimately have no `protocol.Code*' constant.

The first two are Elisp-side preconditions (plan.md D-5): `imoogi-sync'
stops before the subprocess is invoked, so the Go binary never emits
them and never should.  `note_id_unknown' is D-5's Go-origin code that
the binary does not currently emit (REQ-012 reconciles the case as an
ordinary add) -- it stays in the table because D-5 names it, and it
stays here because the Go side declaring it would be a change, not a
fix.  `org_id_duplicated' and `org_id_create_failed' are SPEC-ANKISRC-001
DD-4's front-end-only reports: the ID pass and the duplicate check run in
Elisp before the request is built, so no Go code exists for either.
Everything NOT on this list must be paired with a Go constant.")

(ert-deftest imoogi-error-test-table-entries-are-all-paired-with-a-go-constant ()
  "AC-C-021c (REQ-C-023), the reverse direction: an entry added to the
Elisp table with no matching `protocol.Code*' constant fails.

Until this SPEC the contract ran one way only -- every Go code needed a
table entry, but an Elisp-only entry failed nothing -- so the pairing
REQ-C-023 requires was enforceable from one side alone.  The
`imoogi-error-test--elisp-only-codes' allowlist is what keeps the
assertion honest rather than merely strict: the three codes on it are
paired with a documented reason instead of a constant, and adding a
fourth is a deliberate edit to this file."
  (dolist (entry imoogi-error-table)
    (let ((code (car entry)))
      (should (or (member code imoogi-error-test--go-emitted-codes)
                  (member code imoogi-error-test--elisp-only-codes))))))

(ert-deftest imoogi-error-test-reverse-assertion-rejects-an-unpaired-entry ()
  "The reverse assertion above is only worth having if it can fail.
Run its predicate over a table carrying one unpaired entry and confirm
it rejects -- so a future refactor that quietly widens the allowlist to
everything is caught here rather than by nothing."
  (let ((imoogi-error-table (cons '("invented_code" . "no Go constant declares this")
                                   imoogi-error-table)))
    (should-not (seq-every-p
                 (lambda (entry)
                   (or (member (car entry) imoogi-error-test--go-emitted-codes)
                       (member (car entry) imoogi-error-test--elisp-only-codes)))
                 imoogi-error-table))))

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

(ert-deftest imoogi-error-test-swift-arrow-missing-message-states-the-fix ()
  "SPEC-ANKICARD-004 AC-SW-015 (REQ-SW-014.1): the message names the
problem and a corrective action -- including what does NOT count as an
arrow line, since a body that looks to its author as though it carries
arrows is exactly the case that produces this code."
  (let ((msg (cdr (assoc "swift_arrow_missing" imoogi-error-table))))
    (should msg)
    ;; the problem: the option, and the arrow line it found none of
    (should (string-match-p "ANKI_SWIFT" msg))
    (should (string-match-p "arrow line" msg))
    ;; the three tokens
    (should (string-match-p (regexp-quote ":->") msg))
    (should (string-match-p (regexp-quote ":<-") msg))
    (should (string-match-p (regexp-quote ":<->") msg))
    ;; what does not count -- the shapes an author most often writes
    (should (string-match-p "list" msg))
    (should (string-match-p (regexp-quote "#+BEGIN_") msg))
    (should (string-match-p "table" msg))
    (should (string-match-p "footnote" msg))
    ;; and how to opt out, inheritance included
    (should (string-match-p (regexp-quote "`nil'") msg))
    (should (string-match-p "inherited" msg))))

(ert-deftest imoogi-error-test-message-lookup-known-code ()
  (should (string-match-p "sync root" (imoogi-error-message "sync_root_unset"))))

(ert-deftest imoogi-error-test-message-lookup-unknown-code-falls-back ()
  (let ((msg (imoogi-error-message "totally_unknown_code")))
    (should (string-match-p "totally_unknown_code" msg))
    (should-not (string-match-p "goroutine\\|panic:" msg))))

(provide 'imoogi-error-test)
;;; imoogi-error-test.el ends here
