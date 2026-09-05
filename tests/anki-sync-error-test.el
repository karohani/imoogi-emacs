;;; imoogi-sync-error-test.el --- Tests for imoogi-sync's error surface -*- lexical-binding: t; -*-

;;; Commentary:

;; AC-004 lives in imoogi-setup-test.el (imoogi-anki-setup's own
;; precondition).  This file covers `imoogi-sync's own error-surface
;; wiring: AC-005, AC-006, AC-007 (acceptance.md SS D.1), and AC-021's
;; unreadable-files-naming clause (SS D.5) via imoogi-scan.el's
;; `:unreadable-files'.
;;
;; `imoogi-scan-root' is stubbed with `cl-letf' rather than a rebindable
;; variable (unlike `imoogi-process-runner', which imoogi-process.el
;; already exposes for exactly this purpose) -- it has no existing
;; test-seam, and adding one is out of this milestone's touched-file
;; list (imoogi-scan.el is read-only reference per the coordinator's
;; B10 scope).

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'imoogi)

(defun imoogi-sync-error-test--scan-with-one-target ()
  (list :entries (list (list :key "a.org::0" :note-id nil :note-type "Basic"
                              :source-path "a.org" :deck nil :tags nil
                              :title "T" :body "B"))
        :census nil
        :scan-complete t
        :unreadable-files nil))

(defmacro imoogi-sync-error-test--with-scan (scan-form &rest body)
  (declare (indent 1))
  `(cl-letf (((symbol-function 'imoogi-scan-root)
              (lambda (&rest _args) ,scan-form)))
     ,@body))

;; --- AC-007: sync root unset -------------------------------------------

(ert-deftest imoogi-sync-error-test-ac007-unset-sync-root-stops-and-never-guesses ()
  (let ((imoogi-binary-path (executable-find "true"))
        (imoogi-sync-root nil)
        (imoogi-process-runner
         (lambda (&rest _args) (error "imoogi-process-run must not be invoked"))))
    (let ((report (imoogi-sync)))
      (should (string-match-p "sync root" report))
      (should (string-match-p "imoogi-anki-setup" report))
      (should-not (string-match-p "wrong-type-argument" report))
      (should-not (string-match-p "Debugger entered" report)))))

;; --- AC-005: nothing listening at AnkiConnect --------------------------

(ert-deftest imoogi-sync-error-test-ac005-stopped-anki-produces-guidance ()
  (imoogi-sync-error-test--with-scan (imoogi-sync-error-test--scan-with-one-target)
    (let* ((imoogi-binary-path (executable-find "true"))
           (imoogi-sync-root "/tmp/root")
           (response-json
            (json-serialize
             (list (cons 'protocol_version 1) (cons 'ok :false)
                   (cons 'results (vector))
                   (cons 'errors
                         (vector (list (cons 'code "anki_unreachable")
                                       (cons 'message "dial tcp 127.0.0.1:8765: connect: connection refused")
                                       (cons 'key :null)))))))
           (imoogi-process-runner
            (lambda (&rest _args) (cons 1 response-json))))
      (let ((report (imoogi-sync)))
        (should (string-match-p "Anki" report))
        (should (string-match-p "running" report))
        (dolist (forbidden '("connection refused" "dial tcp" "goroutine" "panic:" "exit status"))
          (should-not (string-match-p (regexp-quote forbidden) report)))))))

;; --- AC-006: AnkiConnect add-on missing --------------------------------

(ert-deftest imoogi-sync-error-test-ac006-missing-addon-distinct-from-stopped-anki ()
  (imoogi-sync-error-test--with-scan (imoogi-sync-error-test--scan-with-one-target)
    (let* ((imoogi-binary-path (executable-find "true"))
           (imoogi-sync-root "/tmp/root")
           (response-json
            (json-serialize
             (list (cons 'protocol_version 1) (cons 'ok :false)
                   (cons 'results (vector))
                   (cons 'errors
                         (vector (list (cons 'code "ankiconnect_missing")
                                       (cons 'message "handshake returned HTTP 500")
                                       (cons 'key :null)))))))
           (imoogi-process-runner
            (lambda (&rest _args) (cons 1 response-json))))
      (let ((report (imoogi-sync)))
        (should (string-match-p "add-on" report))
        (should (string-match-p "install" report))
        (dolist (forbidden '("HTTP 500" "goroutine" "panic:" "exit status"))
          (should-not (string-match-p (regexp-quote forbidden) report)))
        ;; textually distinct from AC-005's message
        (should-not (equal report (imoogi-error-message "anki_unreachable")))))))

;; --- AC-021's unreadable-files-naming clause ---------------------------

(ert-deftest imoogi-sync-error-test-ac021-names-unreadable-files-in-report ()
  "acceptance.md AC-021: when deletion is suppressed because the scan
was incomplete, the run's report names every unreadable file --
including one inside an excluded subtree (lisp/imoogi-scan.el's
`:unreadable-files' is already exclusion-inclusive per its own
docstring; this test only exercises the report-rendering side)."
  (imoogi-sync-error-test--with-scan
      ;; entries deliberately empty -- this test exercises the
      ;; unreadable-files/delete-suppression report only, not
      ;; imoogi-writeback.el's own buffer-visiting write-back path
      ;; (out of this milestone's B10 scope; read-only reference).
      (list :entries nil
            :census nil
            :scan-complete nil
            :unreadable-files '("c.org" "archive/locked.org"))
    (let* ((imoogi-binary-path (executable-find "true"))
           (imoogi-sync-root "/tmp/root")
           (response-json
            (json-serialize
             (list (cons 'protocol_version 1) (cons 'ok t)
                   (cons 'results (vector))
                   (cons 'errors
                         (vector (list (cons 'code "delete_suppressed")
                                       (cons 'message "scan of the sync root was incomplete; deletion suppressed for this run")
                                       (cons 'key :null)))))))
           (imoogi-process-runner
            (lambda (&rest _args) (cons 0 response-json))))
      (let ((report (imoogi-sync)))
        (should (string-match-p "c\\.org" report))
        (should (string-match-p "archive/locked\\.org" report))
        (should (string-match-p "suppress" report))))))

;; --- Regression: a whole-run advisory on an `ok: true' run must NOT
;;     collapse the results/errors summary --------------------------

(ert-deftest imoogi-sync-error-test-advisory-on-ok-run-does-not-collapse-summary ()
  "A nil-:key diagnostic (delete_suppressed, delete_candidate_unowned)
on an `ok: true' run is additive, not a run-level failure -- unlike
AC-005/AC-006's handshake failure, which genuinely carries no results
at all.  Discarding the \"N results, M errors\" summary here would
silently hide real add/update work that happened in the same run."
  (imoogi-sync-error-test--with-scan (imoogi-sync-error-test--scan-with-one-target)
    (let* ((imoogi-binary-path (executable-find "true"))
           (imoogi-sync-root "/tmp/root")
           (response-json
            (json-serialize
             (list (cons 'protocol_version 1) (cons 'ok t)
                   (cons 'results (vector (list (cons 'key "a.org::0")
                                                 (cons 'action "skipped")
                                                 (cons 'note_id :null))))
                   (cons 'errors
                         (vector (list (cons 'code "delete_candidate_unowned")
                                       (cons 'message "note 1002 exists but its note type or field content does not match the registry's record")
                                       (cons 'key :null)))))))
           (imoogi-process-runner
            (lambda (&rest _args) (cons 0 response-json))))
      (let ((report (imoogi-sync)))
        (should (string-match-p "1 results" report))
        (should (string-match-p "1 errors" report))
        (should (string-match-p "unowned\\|content" report))))))

(ert-deftest imoogi-sync-error-test-keyed-error-names-the-heading-and-the-cause ()
  "A per-entry error (one carrying a :key) must reach the user with the
heading it names and the table's explanation of the code.  Observed before
the fix: the report said only \"1 errors\" -- the count was right, but which
heading failed and why never left the response document, so a Cloze skipped
for a missing {{cN:: marker looked like Cloze silently not working."
  (imoogi-sync-error-test--with-scan (imoogi-sync-error-test--scan-with-one-target)
    (let* ((imoogi-binary-path (executable-find "true"))
           (imoogi-sync-root "/tmp/root")
           (response-json
            (json-serialize
             (list (cons 'protocol_version 1) (cons 'ok t)
                   (cons 'results (vector (list (cons 'key "a.org::0")
                                                 (cons 'action "skipped")
                                                 (cons 'note_id :null))))
                   (cons 'errors
                         (vector (list (cons 'code "cloze_marker_missing")
                                       (cons 'message "cloze note type declared but no {{cN:: marker found in title or body")
                                       (cons 'key "a.org::0")))))))
           (imoogi-process-runner
            (lambda (&rest _args) (cons 0 response-json))))
      (let ((report (imoogi-sync)))
        (should (string-match-p "1 errors" report))
        ;; which heading
        (should (string-match-p "a\\.org::0" report))
        ;; why -- the table's wording, not the raw Go message
        (should (string-match-p "marker" report))
        (should-not (string-match-p "declared but no" report))))))

(provide 'imoogi-sync-error-test)
;;; imoogi-sync-error-test.el ends here
