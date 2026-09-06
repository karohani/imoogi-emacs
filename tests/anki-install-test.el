;;; anki-install-test.el --- install-models transport and argv -*- lexical-binding: t; -*-

;;; Commentary:

;; SPEC-ANKICARD-001 M6.  Two things this file pins down:
;;
;;  - AC-C-006d (REQ-C-008): the document the front end writes to
;;    `install-models' stdin carries exactly {protocol_version,
;;    anki_connect_url, user_css}, its user_css is the stylesheet file's
;;    contents byte-for-byte when the file exists and the empty string
;;    when it does not, and the sync request document gains no field.
;;
;;  - The argv every subcommand is actually invoked with.  Every other
;;    test in tests/ rebinds `imoogi-process-runner', so no test used to
;;    exercise the real `imoogi-process--call-binary' at all -- which is
;;    exactly how the M1-era missing-`sync'-argv defect survived.  The
;;    last test here calls the real function against a stub script that
;;    prints its own argv back, so the four argv forms are asserted
;;    rather than assumed.

;;; Code:

(require 'ert)
(require 'imoogi)
(require 'imoogi-process)

(defmacro imoogi-install-test--with-stylesheet (contents var &rest body)
  "Bind `imoogi-user-stylesheet-file' to a temp file holding CONTENTS
\(or to a path that does not exist, when CONTENTS is nil), bind VAR to
that path, and run BODY."
  (declare (indent 2))
  `(let* ((,var (make-temp-file "imoogi-anki-css" nil ".css"))
          (imoogi-user-stylesheet-file ,var))
     (unwind-protect
         (progn
           (if ,contents
               ;; utf-8 is pinned on the write: an unpinned write of
               ;; non-ASCII content prompts for a coding system, and a
               ;; prompt in batch reads stdin and dies at end-of-file.
               (let ((coding-system-for-write 'utf-8))
                 (with-temp-file ,var (insert ,contents)))
             (delete-file ,var))
           ,@body)
       (when (file-exists-p ,var) (delete-file ,var)))))

(ert-deftest imoogi-install-test-request-carries-exactly-three-keys ()
  "AC-C-006d: the install request document is exactly
{protocol_version, anki_connect_url, user_css} -- no more, no fewer,
and spelled with the sync document's `anki_connect_url', not
design.md SS6's `ankiconnect_url'."
  (let* ((json (imoogi-process-serialize-install-request
                "http://127.0.0.1:8765" ".card { letter-spacing: 0.01em; }"))
         (parsed (json-parse-string json :object-type 'alist :array-type 'list)))
    (should (equal (sort (mapcar (lambda (kv) (symbol-name (car kv))) parsed) #'string<)
                   '("anki_connect_url" "protocol_version" "user_css")))
    (should (= (cdr (assq 'protocol_version parsed)) imoogi-protocol-version))
    (should (equal (cdr (assq 'anki_connect_url parsed)) "http://127.0.0.1:8765"))
    (should (equal (cdr (assq 'user_css parsed)) ".card { letter-spacing: 0.01em; }"))))

(ert-deftest imoogi-install-test-user-css-is-the-file-contents-byte-for-byte ()
  "AC-C-006d: with a stylesheet at the defcustom path, `user_css'
equals that file's contents byte-for-byte -- a marker string carrying
a newline, a non-ASCII rune, and a quote, so any re-encoding shows."
  (let ((marker ".deck-한국어 { content: \"x\"; }\n/* end */\n"))
    (imoogi-install-test--with-stylesheet marker _path
      (should (equal (imoogi-user-stylesheet-contents) marker))
      (let* ((json (imoogi-process-serialize-install-request "u" (imoogi-user-stylesheet-contents)))
             (parsed (json-parse-string json :object-type 'alist)))
        (should (equal (cdr (assq 'user_css parsed)) marker))))))

(ert-deftest imoogi-install-test-absent-stylesheet-is-the-empty-string ()
  "AC-C-006d / REQ-C-008: no file at the configured path carries the
empty string -- never nil, never a JSON null, and never a diagnostic."
  (imoogi-install-test--with-stylesheet nil path
    (should-not (file-exists-p path))
    (should (equal (imoogi-user-stylesheet-contents) ""))
    (let* ((json (imoogi-process-serialize-install-request "u" (imoogi-user-stylesheet-contents)))
           (parsed (json-parse-string json :object-type 'alist)))
      (should (equal (cdr (assq 'user_css parsed)) "")))))

(ert-deftest imoogi-install-test-sync-request-gains-no-field ()
  "AC-C-006d's closing clause: the sync request document is unchanged
-- it carries neither `user_css' nor any other new top-level key."
  (let* ((json (imoogi-process-serialize-request
                (list :default-deck "D" :anki-connect-url "u" :registry-path "r"
                      :sync-root "s" :exclude-patterns nil :scan-complete t)
                nil nil))
         (parsed (json-parse-string json :object-type 'alist :array-type 'list)))
    (should (equal (sort (mapcar (lambda (kv) (symbol-name (car kv))) parsed) #'string<)
                   '("census" "config" "entries" "protocol_version")))))

(ert-deftest imoogi-install-test-run-install-uses-the-install-argv ()
  "The install call is dispatched on the `install-models' verb, and
carries the install document rather than the sync one."
  (let* (seen
         (imoogi-process-runner
          (lambda (_binary request-json)
            (push (cons imoogi-process-argv request-json) seen)
            (cons 0 (json-serialize
                     (list (cons 'protocol_version 1) (cons 'ok t)
                           (cons 'results (vector (list (cons 'key "imoogi-Basic")
                                                        (cons 'action "added")
                                                        (cons 'note_id :null))))
                           (cons 'errors (vector))))))))
    (let ((response (imoogi-process-run-install "imoogi-anki" "u" "/* css */")))
      (should (equal (car (car seen)) '("install-models")))
      (should (string-match-p "user_css" (cdr (car seen))))
      (should (eq (plist-get response :ok) t))
      (should (equal (plist-get (car (plist-get response :results)) :key) "imoogi-Basic")))))

(ert-deftest imoogi-install-test-report-names-each-type-and-the-wholesale-replacement ()
  "REQ-C-002: the setup tail reports the per-type outcome and states
that the type's CSS is replaced wholesale, so a hand edit made inside
Anki is known to be discarded rather than silently lost."
  (let ((report (imoogi-setup--install-report
                 (list :ok t
                       :results (list (list :key "imoogi-Basic" :action "added" :note-id nil)
                                      (list :key "imoogi-Cloze" :action "updated" :note-id nil))
                       :errors nil))))
    (should (string-match-p "imoogi-Basic" report))
    (should (string-match-p "added" report))
    (should (string-match-p "imoogi-Cloze" report))
    (should (string-match-p "updated" report))
    (should (string-match-p "wholesale\\|전체\\|통째" report))))

(ert-deftest imoogi-install-test-report-names-a-failed-type ()
  "A `model_install_failed' error is rendered through the code table,
never as a raw Go error string."
  (let ((report (imoogi-setup--install-report
                 (list :ok t
                       :results (list (list :key "imoogi-Basic" :action "added" :note-id nil))
                       :errors (list (list :code "model_install_failed"
                                           :message "imoogi-Cloze: dial tcp: connection refused"
                                           :key "imoogi-Cloze"))))))
    (should (string-match-p "imoogi-Cloze" report))
    (should (string-match-p (regexp-quote (imoogi-error-message "model_install_failed")) report))
    (should-not (string-match-p "dial tcp" report))))

;; --- the real argv, through the real call-binary ----------------------

(defun imoogi-install-test--argv-stub ()
  "Write a stub executable that prints its own argv, one per line, on
stdout and exits 0.  Returns its path."
  (let ((path (make-temp-file "imoogi-argv-stub")))
    (with-temp-file path
      (insert "#!/bin/sh\n"
              "cat >/dev/null\n"
              "for a in \"$@\"; do echo \"$a\"; done\n"))
    (set-file-modes path #o755)
    path))

(ert-deftest imoogi-install-test-call-binary-passes-the-declared-argv ()
  "The real `imoogi-process--call-binary' -- the one every other test
rebinds away -- passes exactly the argv the caller declared.  This is
the assertion whose absence let the M1-era missing-`sync' defect ship."
  (let ((stub (imoogi-install-test--argv-stub)))
    (unwind-protect
        (dolist (argv '(("sync") ("install-models") ("migrate") ("migrate" "--dry-run")))
          (let* ((imoogi-process-argv argv)
                 (result (imoogi-process--call-binary stub "{}")))
            (should (= (car result) 0))
            (should (equal (split-string (string-trim (cdr result)) "\n" t) argv))))
      (delete-file stub))))

(ert-deftest imoogi-install-test-run-entry-points-declare-their-argv ()
  "The three entry points bind the argv their subcommand needs, and
the plain sync run still binds `sync'."
  (let* (seen
         (imoogi-process-runner
          (lambda (_binary _json)
            (push imoogi-process-argv seen)
            (cons 0 (json-serialize (list (cons 'protocol_version 1) (cons 'ok t)
                                          (cons 'results (vector)) (cons 'errors (vector)))))))
         (config (list :default-deck "D" :anki-connect-url "u" :registry-path "r"
                       :sync-root "s" :exclude-patterns nil :scan-complete t)))
    (imoogi-process-run "b" config nil nil)
    (imoogi-process-run-install "b" "u" "")
    (imoogi-process-run-migrate "b" config nil nil t)
    (imoogi-process-run-migrate "b" config nil nil nil)
    (should (equal (nreverse seen)
                   '(("sync") ("install-models") ("migrate" "--dry-run") ("migrate"))))))

(provide 'anki-install-test)
;;; anki-install-test.el ends here
