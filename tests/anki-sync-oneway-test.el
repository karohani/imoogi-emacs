;;; imoogi-sync-oneway-test.el --- AC-024: nothing flows back but the identifier -*- lexical-binding: t; -*-

;;; Commentary:

;; acceptance.md AC-024 (SS D.5):
;;
;;   Given a previously synchronized entry whose registry hash is
;;   current, and an AnkiConnect stub whose stored note has been altered
;;   on the Anki side (different field text, an extra tag), When a sync
;;   run executes, Then the Org buffer's text is unchanged in every
;;   byte, and no request in the run writes any Anki-sourced field
;;   value, tag, or deck name into an Org file.
;;
;; SCOPE NOTE, stated plainly rather than papered over.  AC-024 has two
;; halves, and only one of them is observable from Elisp:
;;
;;  - "the stub's stored note has been altered on the Anki side" is a
;;    back-end condition.  The Go planner reaches AnkiConnect at all;
;;    the front end never does (spec.md C-6).  Whether an altered stored
;;    note provokes a request is internal/planner's own no-op branch
;;    (REQ-011), covered by the Go suite.
;;
;;  - "the Org buffer's text is unchanged in every byte, and no
;;    Anki-sourced value is written into an Org file" IS a front-end
;;    obligation, and it is what these tests exercise: a full
;;    `imoogi-sync' run over a real on-disk sync root with a real live
;;    buffer, driven by a stubbed process runner.
;;
;; The stubs deliberately return responses carrying Anki-sourced field
;; text, tags, and a deck name as EXTRA keys the response contract has
;; no slot for.  That is the sharp form of the assertion: it is not
;; enough that the parser happens to ignore them today -- the test fails
;; the moment any of those values reaches an Org buffer or file, which
;; is precisely the regression AC-024 exists to forbid.

;;; Code:

(require 'ert)
(require 'org)
(require 'imoogi)

;; Field text, a tag, and a deck name that exist ONLY on the (simulated)
;; Anki side.  None may ever appear in an Org buffer or file.
(defconst imoogi-oneway-test--anki-field "ANKI-SIDE-FIELD-TEXT")
(defconst imoogi-oneway-test--anki-tag "anki-side-tag")
(defconst imoogi-oneway-test--anki-deck "AnkiSideDeck")

(defconst imoogi-oneway-test--poison
  (list imoogi-oneway-test--anki-field
        imoogi-oneway-test--anki-tag
        imoogi-oneway-test--anki-deck)
  "Every Anki-sourced value the stub responses carry.")

(defun imoogi-oneway-test--result (action note-id)
  "A response result for ACTION on NOTE-ID, carrying Anki-sourced values.

The `fields', `tags', and `deck' keys are what an implementation that
had started mirroring Anki's own state back into Org would need; the
response contract has no slot for any of them, and nothing may act on
them."
  (list (cons 'key "note.org::0")
        (cons 'action action)
        (cons 'note_id note-id)
        (cons 'fields (list (cons 'Front imoogi-oneway-test--anki-field)
                            (cons 'Back imoogi-oneway-test--anki-field)))
        (cons 'tags (vector imoogi-oneway-test--anki-tag))
        (cons 'deck imoogi-oneway-test--anki-deck)))

(defun imoogi-oneway-test--response (action note-id)
  "Serialize an `ok: true' response carrying one ACTION result on NOTE-ID."
  (json-serialize
   (list (cons 'protocol_version 1)
         (cons 'ok t)
         (cons 'results (vector (imoogi-oneway-test--result action note-id)))
         (cons 'errors (vector)))))

(defmacro imoogi-oneway-test--with-root (content &rest body)
  "Write CONTENT to note.org under a fresh temp sync root and run BODY.

Binds ROOT, FILE, and BUFFER (a live buffer visiting FILE, opened
before the run, so the \"Org buffer's text\" AC-024 names is a real
buffer rather than a re-read of the file).  Cleans up after."
  (declare (indent 1))
  `(let* ((root (make-temp-file "imoogi-oneway-test" t))
          (file (expand-file-name "note.org" root))
          buffer)
     (unwind-protect
         (progn
           (with-temp-file file (insert ,content))
           (setq buffer (find-file-noselect file))
           ,@body)
       (when (buffer-live-p buffer)
         (with-current-buffer buffer (set-buffer-modified-p nil))
         (kill-buffer buffer))
       (delete-directory root t))))

(defun imoogi-oneway-test--file-contents (file)
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

;; --- AC-024 proper: a synchronized, unchanged entry ---------------------

(ert-deftest imoogi-sync-oneway-test-ac024-synced-entry-leaves-org-untouched ()
  "AC-024: a run over an already-synchronized entry must leave the Org
buffer byte-identical, the file on disk byte-identical, and must write
no Anki-sourced field value, tag, or deck name anywhere."
  (imoogi-oneway-test--with-root
      (concat "* Capital of France\n"
              ":PROPERTIES:\n"
              ":ANKI_NOTE_TYPE: Basic\n"
              ":ANKI_NOTE_ID: 1001\n"
              ":ANKI_DECK: Geography\n"
              ":ANKI_TAGS: geography europe\n"
              ":END:\n"
              "Paris.\n")
    (let ((buffer-before (with-current-buffer buffer (buffer-string)))
          (disk-before (imoogi-oneway-test--file-contents file))
          (captured-request nil))
      (let* ((imoogi-sync-root root)
             (imoogi-exclude-patterns nil)
             (imoogi-binary-path (executable-find "true"))
             (imoogi-process-runner
              (lambda (_binary request-json)
                (setq captured-request request-json)
                ;; The registry hash is current, so the back end reports
                ;; the entry as skipped -- no add, no update, nothing to
                ;; write back (REQ-011).
                (cons 0 (imoogi-oneway-test--response "skipped" 1001)))))
        (imoogi-sync))

      ;; The run genuinely ENGAGED this entry.  Without this the test
      ;; would pass just as well over an empty sync root, which would
      ;; assert nothing at all.
      (should captured-request)
      (let ((request (json-parse-string captured-request
                                        :object-type 'alist
                                        :array-type 'list)))
        (let ((entries (cdr (assq 'entries request)))
              (census (cdr (assq 'census request))))
          (should (= (length entries) 1))
          (should (equal (cdr (assq 'key (car entries))) "note.org::0"))
          (should (= (cdr (assq 'note_id (car entries))) 1001))
          (should (= (length census) 1))
          (should (= (cdr (assq 'note_id (car census))) 1001))))

      ;; AC-024's first clause: unchanged in every byte, buffer and disk.
      (should (equal (with-current-buffer buffer (buffer-string)) buffer-before))
      (should (equal (imoogi-oneway-test--file-contents file) disk-before))
      ;; Not even a modification flag: imoogi neither edited nor saved.
      (should-not (with-current-buffer buffer (buffer-modified-p)))

      ;; AC-024's second clause: no Anki-sourced value reached Org.
      (dolist (value imoogi-oneway-test--poison)
        (should-not (string-match-p (regexp-quote value)
                                    (with-current-buffer buffer (buffer-string))))
        (should-not (string-match-p (regexp-quote value)
                                    (imoogi-oneway-test--file-contents file)))))))

;; --- AC-024 on the one path that DOES write ---------------------------

(ert-deftest imoogi-sync-oneway-test-ac024-added-result-forwards-only-the-identifier ()
  "AC-024's second clause on the write-back path.

Write-back is the only place imoogi modifies an Org file at all, so it
is the only place an Anki-sourced value could leak in.  Given an
`added' result carrying Anki-sourced field text, a tag, and a deck
name alongside the assigned identifier, ONLY the identifier may land:
the note type, deck, and tags in the file stay exactly as the user
wrote them."
  (imoogi-oneway-test--with-root
      (concat "* Capital of France\n"
              ":PROPERTIES:\n"
              ":ANKI_NOTE_TYPE: Basic\n"
              ":ANKI_DECK: Geography\n"
              ":ANKI_TAGS: geography europe\n"
              ":END:\n"
              "Paris.\n")
    (let ((buffer-before (with-current-buffer buffer (buffer-string))))
      (let* ((imoogi-sync-root root)
             (imoogi-exclude-patterns nil)
             (imoogi-binary-path (executable-find "true"))
             (imoogi-process-runner
              (lambda (_binary _request-json)
                (cons 0 (imoogi-oneway-test--response "added" 1001)))))
        (imoogi-sync))

      (let ((after (with-current-buffer buffer (buffer-string)))
            (on-disk (imoogi-oneway-test--file-contents file)))
        ;; The identifier -- and only the identifier -- was written.
        (should (string-match-p ":ANKI_NOTE_ID: +1001" after))
        (should (string-match-p ":ANKI_NOTE_ID: +1001" on-disk))
        ;; No Anki-sourced value came with it.
        (dolist (value imoogi-oneway-test--poison)
          (should-not (string-match-p (regexp-quote value) after))
          (should-not (string-match-p (regexp-quote value) on-disk)))
        ;; The user's own property values are untouched -- the run did
        ;; not overwrite ANKI_DECK or ANKI_TAGS with the response's.
        (should (string-match-p ":ANKI_DECK: +Geography" after))
        (should (string-match-p ":ANKI_TAGS: +geography europe" after))
        ;; Every line of the original buffer survives; the diff is
        ;; additive (the one new property line) and nothing else.
        (dolist (line (split-string buffer-before "\n"))
          (should (member line (split-string after "\n"))))))))

(provide 'imoogi-sync-oneway-test)
;;; imoogi-sync-oneway-test.el ends here
