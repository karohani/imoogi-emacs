;;; anki-notetype-test.el --- note-type defaults, stock compatibility, deck class -*- lexical-binding: t; -*-

;;; Commentary:

;; SPEC-ANKICARD-001 M6:
;;
;;  - AC-C-021b (REQ-C-005.1/.3): every front-end site that hard-codes a
;;    note-type literal names the imoogi-owned types -- the
;;    ANKI_NOTE_TYPE_ALL allowlist, the two marking commands, and the
;;    cloze auto-mark.
;;  - AC-C-022a (REQ-C-005.2): a hand-written heading declaring stock
;;    `Basic' or `Cloze' is still produced as a sync target by the scan,
;;    and the cloze-marker handling still accepts either Cloze form.
;;  - AC-C-004 (REQ-C-006.2): the deck-class mirror table.
;;
;; The AC-C-004 test deliberately re-implements NOTHING.  The normalizer
;; lives in Go (internal/anki/model/deckclass.go) and has no Elisp-side
;; runtime seam -- the wrapper is emitted by the card template, never by
;; this front end -- so an Elisp reimplementation would assert only that
;; two copies of one algorithm agree, which is the drift this SPEC's own
;; contract tests exist to prevent.  What IS asserted is the shared
;; fixture: that the Go table still carries AC-C-004's nine rows
;; verbatim.  Same mechanism, and same rationale, as
;; `imoogi-error-test-codes-match-source-file'.

;;; Code:

(require 'ert)
(require 'org)
(require 'seq)
(require 'imoogi)
(require 'imoogi-scan)

;; --- AC-C-021b: the literal sites ------------------------------------

(ert-deftest imoogi-notetype-test-allowlist-offers-both-imoogi-names ()
  "AC-C-021b: ANKI_NOTE_TYPE_ALL offers the two imoogi-owned names."
  (let ((all (cdr (assoc "ANKI_NOTE_TYPE_ALL" org-global-properties))))
    (should all)
    (should (equal (split-string all " " t) '("imoogi-Basic" "imoogi-Cloze")))))

(defmacro imoogi-notetype-test--with-org (text &rest body)
  (declare (indent 1))
  `(with-temp-buffer
     (org-mode)
     (setq-local transient-mark-mode t)
     (insert ,text)
     ,@body))

(ert-deftest imoogi-notetype-test-mark-basic-writes-the-imoogi-name ()
  "AC-C-021b: `C-c a b' marks the heading imoogi-Basic (REQ-C-005.1)."
  (imoogi-notetype-test--with-org "* 제목\n\n본문\n"
    (goto-char (point-max))
    (imoogi-anki-mark-basic)
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Basic"))))

(ert-deftest imoogi-notetype-test-mark-cloze-writes-the-imoogi-name ()
  "AC-C-021b: `C-c a c' marks the heading imoogi-Cloze (REQ-C-005.1)."
  (imoogi-notetype-test--with-org "* 제목\n\n본문\n"
    (goto-char (point-max))
    (imoogi-anki-mark-cloze)
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Cloze"))))

(ert-deftest imoogi-notetype-test-cloze-automark-writes-the-imoogi-name ()
  "AC-C-021b: the cloze auto-mark writes imoogi-Cloze."
  (imoogi-notetype-test--with-org "* 제목\n\n유럽에서 가장 긴 강은 볼가강이다.\n"
    (goto-char (point-min))
    (search-forward "볼가강")
    (let ((beg (match-beginning 0)) (end (match-end 0)))
      (set-mark beg) (goto-char end) (activate-mark)
      (imoogi-anki-cloze-region beg end))
    (org-back-to-heading t)
    (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") "imoogi-Cloze"))))

(ert-deftest imoogi-notetype-test-cloze-automark-accepts-either-cloze-form ()
  "AC-C-021b / REQ-C-005.2: the auto-mark's comparison accepts either
Cloze form, so a hand-written stock `Cloze' heading is left alone
rather than reported as a type mismatch."
  (dolist (declared '("Cloze" "imoogi-Cloze"))
    (imoogi-notetype-test--with-org
        (format "* 제목\n:PROPERTIES:\n:ANKI_NOTE_TYPE: %s\n:END:\n\n본문 볼가강 끝.\n" declared)
      (goto-char (point-min))
      (search-forward "볼가강")
      (let ((beg (match-beginning 0)) (end (match-end 0)))
        (set-mark beg) (goto-char end) (activate-mark)
        (imoogi-anki-cloze-region beg end))
      (org-back-to-heading t)
      ;; unchanged: the declared type wins, whichever Cloze form it is
      (should (equal (org-entry-get (point) "ANKI_NOTE_TYPE") declared)))))

;; --- AC-C-022a: stock headings are still sync targets -----------------

(ert-deftest imoogi-notetype-test-stock-headings-still-scan-as-targets ()
  "AC-C-022a (REQ-C-005.2): a hand-written stock `Basic' or `Cloze'
heading is still produced as a sync target, carrying its declared type
verbatim -- the scan filters on the property's presence, never on its
value, so no name set is privileged."
  (let ((root (make-temp-file "imoogi-notetype-root" t)))
    (unwind-protect
        (progn
          (with-temp-file (expand-file-name "stock.org" root)
            (insert "* 스톡 Basic\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n\n앞\n\n"
                    "* 스톡 Cloze\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Cloze\n:END:\n\n{{c1::빈칸}}\n\n"
                    "* imoogi Basic\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Basic\n:END:\n\n앞\n\n"
                    "* imoogi Cloze\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Cloze\n:END:\n\n{{c1::빈칸}}\n"))
          (let* ((scan (imoogi-scan-root root nil))
                 (types (mapcar (lambda (e) (plist-get e :note-type))
                                (plist-get scan :entries))))
            (should (equal types '("Basic" "Cloze" "imoogi-Basic" "imoogi-Cloze")))))
      (delete-directory root t))))

;; --- AC-C-004: the deck-class mirror table ---------------------------

(defconst imoogi-notetype-test--deck-class-table
  '(("(PROGRAMMER)::(GO)" "programmer-go"  "deck-programmer-go")
    ("Geography::Europe"  "geography-europe" "deck-geography-europe")
    ("A::::B"             "a-b"            "deck-a-b")
    ("2026 Review"        "_2026-review"   "deck-_2026-review")
    ("  Spaced  Name  "   "spaced-name"    "deck-spaced-name")
    ("Math_Notes"         "math_notes"     "deck-math_notes")
    ("::"                 "unnamed"        "deck-unnamed")
    ("---"                "unnamed"        "deck-unnamed")
    ("!!!"                "unnamed"        "deck-unnamed"))
  "AC-C-004's nine rows, verbatim.  This is an EXPECTATION table, not an
implementation: nothing here normalizes anything.")

(defun imoogi-notetype-test--go-deck-class-rows (file)
  "Read the (DECK TOKEN CLASS) rows out of the Go table in FILE."
  (with-temp-buffer
    (insert-file-contents file)
    (goto-char (point-min))
    (let (rows)
      (while (re-search-forward
              "{\"[^\"]*\",[ \t]*\"\\([^\"]*\\)\",[ \t]*\"\\([^\"]*\\)\",[ \t]*\"\\([^\"]*\\)\"}"
              nil t)
        (push (list (match-string 1) (match-string 2) (match-string 3)) rows))
      (nreverse rows))))

(ert-deftest imoogi-notetype-test-deck-class-mirror-table ()
  "AC-C-004: every row of the acceptance table is present verbatim in
the Go normalizer's own table, deck -> token -> emitted class.

Read rather than recomputed: the normalizer is Go-side and has no Elisp
seam (the wrapper is emitted by the card template), so the assertion
this side can honestly make is that the shared fixture still carries
the acceptance rows -- the Go test is what asserts the algorithm
produces them."
  (let* ((candidates
          (list (and (or load-file-name buffer-file-name)
                     (expand-file-name "../internal/anki/model/deckclass_test.go"
                                       (file-name-directory (or load-file-name buffer-file-name))))
                (expand-file-name "internal/anki/model/deckclass_test.go" default-directory)))
         (go-file (seq-find (lambda (f) (and f (file-readable-p f))) candidates)))
    (skip-unless go-file)
    (let ((rows (imoogi-notetype-test--go-deck-class-rows go-file)))
      (should rows)
      (dolist (expected imoogi-notetype-test--deck-class-table)
        (should (member expected rows)))
      ;; REQ-C-006.2's closing clause: no row emits the bare prefix.
      (dolist (row rows)
        (should-not (equal (nth 2 row) "deck-"))))))

(provide 'anki-notetype-test)
;;; anki-notetype-test.el ends here
