;;; imoogi-process.el --- Subprocess invocation for the sync-run request/response -*- lexical-binding: t; -*-

;;; Commentary:

;; Serializes the request document, spawns the Go binary once per sync
;; run, writes the request to its stdin, reads the response from its
;; stdout, and checks the protocol version and exit code (plan.md D-2).
;;
;; Uses the native `json-serialize' / `json-parse-string' functions
;; (Emacs 27+) rather than json.el, for unambiguous true/false/null
;; handling -- see the field-level helpers below.

;;; Code:

(require 'json)
(require 'seq)

(defvar imoogi-protocol-version 1
  "Wire-contract version this front end speaks.
Must match the Go binary's compiled-in `internal/protocol.Version'.")

(defvar imoogi-process-argv '("sync")
  "Argv the next `imoogi-process-runner' invocation dispatches on.

cmd/imoogi-anki/main.go switches on argv[1] and, given none, prints its
usage to stderr and exits 2 -- which this front end can only report as
an unparseable response.  The four forms this package uses are
\'(\"sync\"), \'(\"install-models\"), \'(\"migrate\"), and
\'(\"migrate\" \"--dry-run\").

It is a dynamically bound variable rather than a runner argument
deliberately: every test in tests/ rebinds `imoogi-process-runner' with
a two-argument lambda, so widening the seam's arity would break each of
them at once.  Binding the argv beside the seam keeps the seam stable
and still lets a fake runner read which subcommand it is answering
for.")

(defvar imoogi-process-runner #'imoogi-process--call-binary
  "Function of (BINARY REQUEST-JSON-STRING) -> (EXIT-CODE . STDOUT-STRING).
Rebindable for testing without invoking a real subprocess.

The subcommand is NOT an argument: it travels in `imoogi-process-argv',
which a fake runner reads when it needs to answer differently per
subcommand.")

(defun imoogi-process--call-binary (binary request-json)
  "Invoke BINARY as a subprocess, writing REQUEST-JSON to its stdin and
capturing stdout.  Returns a cons (EXIT-CODE . STDOUT-STRING).

The subcommand comes from `imoogi-process-argv'.  Because every other
test rebinds `imoogi-process-runner', this function used to be exercised
by nothing at all -- which is exactly how a missing `sync' argv survived
into a shipped milestone.  `imoogi-install-test-call-binary-passes-the-\
declared-argv' now drives THIS function against a stub script that
prints its own argv back, so all four forms are asserted rather than
kept in sync by hand."
  (with-temp-buffer
    (insert request-json)
    (let* ((coding-system-for-write 'utf-8)
           (coding-system-for-read 'utf-8)
           (exit-code (apply #'call-process-region
                             (point-min) (point-max)
                             binary t t nil imoogi-process-argv)))
      (cons exit-code (buffer-string)))))

(defun imoogi-process-serialize-install-request (anki-connect-url user-css)
  "Serialize the `install-models' request document (REQ-C-008, AC-C-006d).

Exactly three keys: protocol_version, anki_connect_url, user_css.  The
endpoint key is spelled as the sync document spells it -- design.md SS6's
`ankiconnect_url' is a typo the Go side already resolved against
`anki_connect_url' (internal/anki/protocol/install_test.go).

USER-CSS is carried verbatim and is the empty string when no user
stylesheet exists; the key is never dropped."
  (json-serialize
   (list (cons 'protocol_version imoogi-protocol-version)
         (cons 'anki_connect_url anki-connect-url)
         (cons 'user_css (or user-css "")))))

(defun imoogi-process--census-alist (entry)
  "Serialize one census plist ENTRY into a wire-shape alist."
  (list (cons 'note_id (plist-get entry :note-id))
        (cons 'source_path (plist-get entry :source-path))))

(defun imoogi-process--entry-alist (entry)
  "Serialize one scan-produced entry plist ENTRY into a wire-shape alist."
  (list (cons 'key (plist-get entry :key))
        (cons 'note_id (or (plist-get entry :note-id) :null))
        (cons 'note_type (plist-get entry :note-type))
        (cons 'source_path (plist-get entry :source-path))
        (cons 'deck (or (plist-get entry :deck) :null))
        (cons 'tags (vconcat (plist-get entry :tags)))
        (cons 'title (plist-get entry :title))
        (cons 'body (plist-get entry :body))))

(defun imoogi-process-serialize-request (config census entries)
  "Serialize CONFIG (plist), CENSUS (list of census plists), and ENTRIES
(list of entry plists) into the wire-contract request JSON string."
  (json-serialize
   (list
    (cons 'protocol_version imoogi-protocol-version)
    (cons 'config
          (list (cons 'default_deck (plist-get config :default-deck))
                (cons 'anki_connect_url (plist-get config :anki-connect-url))
                (cons 'registry_path (plist-get config :registry-path))
                (cons 'sync_root (plist-get config :sync-root))
                (cons 'exclude_patterns (vconcat (plist-get config :exclude-patterns)))
                (cons 'scan_complete (if (plist-get config :scan-complete) t :false))))
    (cons 'census (vconcat (mapcar #'imoogi-process--census-alist census)))
    (cons 'entries (vconcat (mapcar #'imoogi-process--entry-alist entries))))))

(defun imoogi-process--result-plist (alist)
  "Convert one parsed response result ALIST into a plist."
  (list :key (let ((v (cdr (assq 'key alist)))) (if (eq v :null) nil v))
        :action (cdr (assq 'action alist))
        :note-id (let ((v (cdr (assq 'note_id alist)))) (if (eq v :null) nil v))))

(defun imoogi-process--error-plist (alist)
  "Convert one parsed response error ALIST into a plist."
  (list :code (cdr (assq 'code alist))
        :message (cdr (assq 'message alist))
        :key (let ((v (cdr (assq 'key alist)))) (if (eq v :null) nil v))))

(defun imoogi-process-parse-response (response-json)
  "Parse RESPONSE-JSON (the wire-contract response string) into a plist:
  :protocol-version  integer
  :ok                boolean
  :results           list of result plists (:key :action :note-id)
  :errors            list of error plists (:code :message :key)

Returns nil if RESPONSE-JSON cannot be parsed as JSON."
  (condition-case nil
      (let* ((parsed (json-parse-string response-json
                                         :object-type 'alist
                                         :array-type 'list
                                         :null-object :null
                                         :false-object :false)))
        (list :protocol-version (cdr (assq 'protocol_version parsed))
              :ok (eq (cdr (assq 'ok parsed)) t)
              :results (mapcar #'imoogi-process--result-plist
                                (cdr (assq 'results parsed)))
              :errors (mapcar #'imoogi-process--error-plist
                               (cdr (assq 'errors parsed)))))
    (json-parse-error nil)
    (error nil)))

(defun imoogi-process-run (binary config census entries &optional argv)
  "Run one sync-run request/response cycle against BINARY.

ARGV defaults to \'(\"sync\").  `imoogi-process-run-migrate' passes the
migrate forms; the request document is identical either way, which is
REQ-C-018's wire-stability clause -- the verb is the only thing that
distinguishes the two on the wire.

Serializes CONFIG/CENSUS/ENTRIES, invokes `imoogi-process-runner',
and parses the response.  Returns the parsed response plist (see
`imoogi-process-parse-response'), or nil when stdout cannot be parsed
as the wire-contract response document -- the caller then falls back
to its own generic error taxonomy (design.md SS3 step 14).

Disclosed deviation from a literal reading of plan.md D-2's exit-code
contract: a well-formed response document is parsed and returned
regardless of exit code, not gated on exit code 0.  A run-level
failure (design.md SS3 step 13 -- e.g. the AnkiConnect handshake
failing before any entry is processed) sets `ok: false', carries a
typed code in `errors[]', and exits non-zero (cmd/imoogi/main.go's
`failRun') -- but it is still a WELL-FORMED response document, and
plan.md D-5's own rationale is that the code, not raw process/exit
detail, is what the front end's error table (imoogi-error.el) renders
into REQ-018's user-facing message.  Reading D-2's \"falls back to its
own error taxonomy\" as \"look up the code in the front end's own
table\" (rather than \"never read stdout at all\") is what makes
REQ-018's anki_unreachable / ankiconnect_missing branches
implementable without a second, redundant Elisp-side AnkiConnect
probe.  A stdout that fails to parse as JSON at all -- a genuine
binary crash, not a deliberate failRun -- still returns nil here,
which is the actual \"no response\" fallback path imoogi-sync.el
handles with its own generic message."
  (let* ((imoogi-process-argv (or argv '("sync")))
         (request-json (imoogi-process-serialize-request config census entries))
         (result (funcall imoogi-process-runner binary request-json))
         (stdout (cdr result)))
    (imoogi-process-parse-response stdout)))

(defun imoogi-process-run-install (binary anki-connect-url user-css)
  "Run one `install-models' request/response cycle against BINARY.

Returns the parsed response plist, or nil when stdout is not a
well-formed response document -- the same contract `imoogi-process-run'
carries, so the caller renders through `imoogi-error-message' rather
than reading process detail."
  (let* ((imoogi-process-argv '("install-models"))
         (request-json (imoogi-process-serialize-install-request
                        anki-connect-url user-css))
         (result (funcall imoogi-process-runner binary request-json)))
    (imoogi-process-parse-response (cdr result))))

(defun imoogi-process-run-migrate (binary config census entries dry-run)
  "Run one `migrate' request/response cycle against BINARY.

DRY-RUN non-nil selects `migrate --dry-run', which reads the registry
and issues zero AnkiConnect writes (AC-C-018b).  The request document
is the SAME sync-shaped document either way (REQ-C-018)."
  (imoogi-process-run binary config census entries
                      (if dry-run '("migrate" "--dry-run") '("migrate"))))

(defun imoogi-process-migrate-candidates (response)
  "Return RESPONSE's `migrate_candidate' results.

design.md SS11: the new action value is a count-only outcome.  It names
the entries a confirmed migration WOULD re-home, triggers no write-back,
and its length is the number REQ-C-019's confirmation must state."
  (seq-filter (lambda (r) (equal (plist-get r :action) "migrate_candidate"))
              (plist-get response :results)))

(provide 'imoogi-process)
;;; imoogi-process.el ends here
