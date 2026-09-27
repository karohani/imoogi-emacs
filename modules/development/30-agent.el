;;; 30-agent.el --- coding agent -> Emacs IPC receiver -*- lexical-binding: t; -*-

;; SPEC-AGENTIPC-001. Coding agents (Claude Code, Codex, ...) call the
;; `imoogi-agent' CLI, which writes one JSON event to a temporary file and asks
;; the running Emacs server to read it through `imoogi-agent-receive-file'.
;;
;; The receiver is a closed protocol: exactly two entry points share one
;; validation and dispatch path, every call returns a status string ("ok" or
;; "error:<reason>"), and no event value is ever evaluated as code. Only
;; defvar/defconst/defun forms live here so that loading the file twice (the
;; boot list plus a direct test load) is harmless.

;;; Code:

(imoogi-require "30-agent")

(require 'subr-x)

(defconst imoogi-agent-max-bytes 1048576
  "Largest accepted event, in bytes (event file size or JSON string bytes).")

(defconst imoogi-agent-log-buffer-name "*imoogi-agent*"
  "Name of the append-only, session-scoped event log buffer.")

(defconst imoogi-agent--types
  '("message" "open-file" "goto-location" "artifact-created" "task-finished")
  "Event types accepted by protocol version 1.")

(defconst imoogi-agent--json-options
  '(:object-type alist :array-type array :null-object :null :false-object :false)
  "Strict `json-parse-buffer' options.
Every JSON value stays distinguishable: null → :null, false → :false,
true → t, [] → a vector, {} → nil, object → alist.")

;;; Rejection plumbing

(defun imoogi-agent--reject (reason)
  "Abort the current event with REASON (a string without the error: prefix)."
  (throw 'imoogi-agent--reject reason))

(defun imoogi-agent--one-line (value)
  "Return VALUE with every CRLF, LF, and CR replaced by one ⏎."
  (replace-regexp-in-string "\r\n\\|[\n\r]" "⏎" value t t))

;;; Reading and parsing

(defun imoogi-agent--read-file (file)
  "Return the content of event FILE after the precondition and size checks.
FILE must be an absolute, local, readable regular file of at most
`imoogi-agent-max-bytes' bytes. The remote test comes before anything that
touches the file, because asking a remote file for its size already opens a
connection. The size is checked before the content is read."
  (unless (and (stringp file)
               (file-name-absolute-p file)
               (not (file-remote-p file))
               (file-regular-p file)
               (file-readable-p file))
    (imoogi-agent--reject "unreadable"))
  (let ((size (file-attribute-size (file-attributes file))))
    (unless size
      (imoogi-agent--reject "unreadable"))
    (when (> size imoogi-agent-max-bytes)
      (imoogi-agent--reject "too-large")))
  (with-temp-buffer
    (let ((coding-system-for-read 'utf-8))
      (insert-file-contents file))
    (buffer-string)))

(defun imoogi-agent--check-string (json)
  "Return JSON after the type and size checks of the string entry point."
  (unless (stringp json)
    (imoogi-agent--reject "parse"))
  (when (> (string-bytes json) imoogi-agent-max-bytes)
    (imoogi-agent--reject "too-large"))
  ;; A unibyte string holds raw UTF-8 bytes (what `json-serialize' and
  ;; process output produce); decode it so the parser sees characters.
  (if (multibyte-string-p json)
      json
    (decode-coding-string json 'utf-8)))

(defun imoogi-agent--parse (json)
  "Parse JSON (a string) into the strict representation.
Reject syntax errors (including nesting depth and encoding errors) with
parse, and anything but whitespace after the first value with
trailing-content."
  (with-temp-buffer
    (insert json)
    (goto-char (point-min))
    (let ((value (condition-case nil
                     (apply #'json-parse-buffer imoogi-agent--json-options)
                   (json-error (imoogi-agent--reject "parse")))))
      (skip-chars-forward " \t\n\r")
      (unless (eobp)
        (imoogi-agent--reject "trailing-content"))
      value)))

(defun imoogi-agent--duplicate-keys-p (object)
  "Return non-nil when the alist OBJECT holds the same key twice."
  (let ((keys (mapcar #'car object)))
    (/= (length keys) (length (delete-dups (copy-sequence keys))))))

;;; Validation

(defun imoogi-agent--field (object key kind &optional optional)
  "Return the value of KEY in OBJECT after checking it against KIND.
KIND is `string', `text' (non-empty string), or `positive' (integer ≥ 1).
When OPTIONAL is non-nil, a missing key or a JSON null means absent and
yields nil. Any other mismatch — a missing required key, a required null,
[], {}, false, true, or a value of the wrong kind — rejects the event with
bad-field."
  (let ((cell (assq key object)))
    (if (and optional (or (null cell) (eq (cdr cell) :null)))
        nil
      (let ((value (cdr cell)))
        (unless (and cell
                     (pcase kind
                       ('string (stringp value))
                       ('text (and (stringp value) (not (string-empty-p value))))
                       ('positive (and (integerp value) (> value 0)))))
          (imoogi-agent--reject "bad-field"))
        value))))

(defun imoogi-agent--validate (event)
  "Validate EVENT (parsed JSON) and return it as a normalized plist.
Checks run in the order duplicate keys, version, type, fields, paths, so
that the first violation in that order is the one reported."
  (let ((payload (and (listp event) (cdr (assq 'payload event)))))
    (when (or (and (listp event) (imoogi-agent--duplicate-keys-p event))
              (and (consp payload) (imoogi-agent--duplicate-keys-p payload)))
      (imoogi-agent--reject "duplicate-key"))
    (unless (and (listp event) (equal (cdr (assq 'version event)) "1"))
      (imoogi-agent--reject "bad-version"))
    (let ((type (cdr (assq 'type event))))
      (unless (member type imoogi-agent--types)
        (imoogi-agent--reject "unknown-type"))
      (imoogi-agent--field event 'timestamp 'text)
      (let ((project (imoogi-agent--field event 'project 'string t))
            (session (imoogi-agent--field event 'session 'string t)))
        ;; `payload' must be a JSON object: an alist, or nil for {}.
        (unless (and (assq 'payload event) (listp payload))
          (imoogi-agent--reject "bad-field"))
        (append (list :type type :project project :session session)
                (imoogi-agent--validate-payload type payload))))))

(defun imoogi-agent--validate-payload (type payload)
  "Validate PAYLOAD of TYPE and return its fields as a plist."
  (pcase type
    ("message"
     (list :text (imoogi-agent--field payload 'text 'text)))
    ("open-file"
     (list :path (imoogi-agent--field payload 'path 'string)))
    ("goto-location"
     (list :path (imoogi-agent--field payload 'path 'string)
           :line (imoogi-agent--field payload 'line 'positive)
           :column (or (imoogi-agent--field payload 'column 'positive t) 1)))
    ("artifact-created"
     (list :path (imoogi-agent--field payload 'path 'string)
           :artifact-type (imoogi-agent--field payload 'artifactType 'string t)
           :title (imoogi-agent--field payload 'title 'string t)))
    ("task-finished"
     (let ((status (imoogi-agent--field payload 'status 'string)))
       (unless (member status '("success" "failed"))
         (imoogi-agent--reject "bad-field"))
       (list :status status
             :summary (imoogi-agent--field payload 'summary 'string t)
             :artifact (imoogi-agent--field payload 'artifact 'string t))))))

;;; Notification and log

(defun imoogi-agent--notify (text)
  "Show TEXT, made one line, in the echo area without format interpretation."
  (message "%s" (imoogi-agent--one-line text)))

(defun imoogi-agent--log-buffer ()
  "Return the log buffer, creating it (read-only) when it does not exist."
  (or (get-buffer imoogi-agent-log-buffer-name)
      (with-current-buffer (get-buffer-create imoogi-agent-log-buffer-name)
        (special-mode)
        (current-buffer))))

(defun imoogi-agent--log-value (value)
  "Return VALUE as a one-line log token, or - when it is not a string."
  (if (stringp value) (imoogi-agent--one-line value) "-"))

(defun imoogi-agent--log (event result detail)
  "Append one line for EVENT with RESULT and optional DETAIL to the log.
EVENT is the parsed JSON (possibly nil or not an object); RESULT is ok or
a rejection reason."
  (let* ((object (and (listp event) event))
         (line (concat
                (format-time-string "[%Y-%m-%d %H:%M:%S] ")
                (imoogi-agent--log-value (cdr (assq 'type object)))
                " project=" (imoogi-agent--log-value (cdr (assq 'project object)))
                " session=" (imoogi-agent--log-value (cdr (assq 'session object)))
                " " result
                (if (and (stringp detail) (not (string-empty-p detail)))
                    (concat " " (imoogi-agent--one-line detail))
                  ""))))
    (with-current-buffer (imoogi-agent--log-buffer)
      (let ((inhibit-read-only t))
        (save-excursion
          (goto-char (point-max))
          (insert line "\n"))))))

;;; Dispatch

(defun imoogi-agent-dispatch (event)
  "Carry out the validated, normalized EVENT plist.
Return a detail string for the log line."
  (pcase (plist-get event :type)
    ("message"
     (imoogi-agent--notify (plist-get event :text))
     (plist-get event :text))
    ("task-finished"
     (let* ((summary (plist-get event :summary))
            (notice (concat (if (equal (plist-get event :status) "success")
                                "✓ Agent task finished"
                              "✗ Agent task failed")
                            (if (and summary (not (string-empty-p summary)))
                                (concat ": " summary)
                              ""))))
       (imoogi-agent--notify notice)
       (concat (plist-get event :status)
               (if summary (concat " " summary) ""))))
    (type (error "Event type %s is not handled yet" type))))

;;; Entry points

(defun imoogi-agent--receive (reader)
  "Run one event through READER, validation, dispatch, and the log.
READER returns the JSON string. Return \"ok\" or \"error:<reason>\"; no
elisp error escapes. User interaction is inhibited throughout, so a stray
prompt becomes an error:handler rejection instead of hanging the server."
  (let (event detail)
    (let ((reason
           (condition-case err
               (catch 'imoogi-agent--reject
                 (let ((inhibit-interaction t))
                   (setq event (imoogi-agent--parse (funcall reader)))
                   (setq detail (imoogi-agent-dispatch
                                 (imoogi-agent--validate event))))
                 nil)
             (error
              (setq detail (error-message-string err))
              "handler"))))
      (ignore-errors
        (imoogi-agent--log event (or reason "ok") detail))
      (if reason (concat "error:" reason) "ok"))))

;; @MX:ANCHOR: [AUTO] File entry point called by the fixed emacsclient
;; expression of the imoogi-agent CLI (internal/agentipc).
;; @MX:REASON: External contract — the CLI's evaluation expression names this
;; function and parses its "ok"/"error:<reason>" return value.
(defun imoogi-agent-receive-file (file)
  "Receive the agent event stored in FILE and return a status string.
FILE is read, never modified or deleted. The result is \"ok\" or
\"error:<reason>\"."
  (imoogi-agent--receive (lambda () (imoogi-agent--read-file file))))

;; @MX:ANCHOR: [AUTO] String entry point sharing the validation and dispatch
;; path of `imoogi-agent-receive-file'.
;; @MX:REASON: Protocol contract — both entry points must return the same
;; status and cause the same side effects for the same event.
(defun imoogi-agent-receive-json (json)
  "Receive the agent event in the JSON string and return a status string."
  (imoogi-agent--receive (lambda () (imoogi-agent--check-string json))))

(provide 'imoogi-agent)
;;; 30-agent.el ends here
