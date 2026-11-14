;;; 30-emacsctl.el --- coding agent -> Emacs IPC receiver -*- lexical-binding: t; -*-

;; SPEC-AGENTIPC-001. Coding agents (Claude Code, Codex, ...) call the
;; `imoogi-emacsctl' CLI, which writes one JSON event to a temporary file and asks
;; the running Emacs server to read it through `imoogi-emacsctl-receive-file'.
;;
;; The receiver is a closed protocol: exactly two entry points share one
;; validation and dispatch path, every call returns a status string ("ok" or
;; "error:<reason>"), and no event value is ever evaluated as code. Only
;; defvar/defconst/defun forms live here so that loading the file twice (the
;; boot list plus a direct test load) is harmless.

;;; Code:

(imoogi-require "30-emacsctl")

(require 'subr-x)

(defconst imoogi-emacsctl-max-bytes 1048576
  "Largest accepted event, in bytes (event file size or JSON string bytes).")

(defconst imoogi-emacsctl-max-payload-bytes 104857600
  "Largest payload file the receiver opens, in bytes (100 MiB).
A payload path naming a larger file rejects the whole event with
payload-too-large. The value equals the `large-file-warning-threshold' set
in modules/general/00-defaults.el but never reads that variable: the
receiver binds it to nil when visiting, so this cap is the only bound.")

(defconst imoogi-emacsctl-log-buffer-name "*imoogi-emacsctl*"
  "Name of the append-only, session-scoped event log buffer.")

(defconst imoogi-emacsctl--types
  '("message" "open-file" "goto-location" "artifact-created" "task-finished")
  "Event types accepted by protocol version 1.")

(defconst imoogi-emacsctl--json-options
  '(:object-type alist :array-type array :null-object :null :false-object :false)
  "Strict `json-parse-buffer' options.
Every JSON value stays distinguishable: null → :null, false → :false,
true → t, [] → a vector, {} → nil, object → alist.")

;;; Rejection plumbing

(defun imoogi-emacsctl--reject (reason)
  "Abort the current event with REASON (a string without the error: prefix)."
  (throw 'imoogi-emacsctl--reject reason))

(defun imoogi-emacsctl--one-line (value)
  "Return VALUE with every CRLF, LF, and CR replaced by one ⏎."
  (replace-regexp-in-string "\r\n\\|[\n\r]" "⏎" value t t))

;;; Reading and parsing

(defun imoogi-emacsctl--read-file (file)
  "Return the content of event FILE after the precondition, trust, and size checks.
FILE must be an absolute, local, readable regular file of at most
`imoogi-emacsctl-max-bytes' bytes. The checks run in the order file
conditions (unreadable), trust (untrusted), size (too-large), and only then
is the content read. The remote test comes before anything that touches the
file, because asking a remote file for its size already opens a connection.
The trust and size checks look at the file that would be read, that is FILE
with symbolic links resolved: it must be owned by `user-uid' and writable
by neither its group nor others."
  (unless (and (stringp file)
               (file-name-absolute-p file)
               (not (file-remote-p file))
               (file-regular-p file)
               (file-readable-p file))
    (imoogi-emacsctl--reject "unreadable"))
  ;; @MX:NOTE: [AUTO] Trust check (SPEC-AGENTIPC-002 REQ-AIPH-001/002). Threat
  ;; model: in a shared, world-writable TMPDIR (/tmp on Linux without TMPDIR)
  ;; another local user can plant a file under the name the CLI used after the
  ;; CLI timed out and removed its directory, and the server reads it late
  ;; (SPEC-AGENTIPC-001 plan.md D-10). A file owned by someone else or writable
  ;; by group/others is refused before any judgement, including its size. A
  ;; user-owned link to a user-owned file with chosen content still passes
  ;; (SPEC-AGENTIPC-002 plan.md R-7).
  (let* ((true (file-truename file))
         (attributes (file-attributes true 'integer))
         (modes (file-modes true)))
    (unless (and attributes
                 (eql (file-attribute-user-id attributes) (user-uid))
                 modes
                 (zerop (logand modes #o022)))
      (imoogi-emacsctl--reject "untrusted"))
    (let ((size (file-attribute-size attributes)))
      (unless size
        (imoogi-emacsctl--reject "unreadable"))
      (when (> size imoogi-emacsctl-max-bytes)
        (imoogi-emacsctl--reject "too-large"))))
  (with-temp-buffer
    (let ((coding-system-for-read 'utf-8))
      (insert-file-contents file))
    (buffer-string)))

(defun imoogi-emacsctl--check-string (json)
  "Return JSON after the type and size checks of the string entry point."
  (unless (stringp json)
    (imoogi-emacsctl--reject "parse"))
  (when (> (string-bytes json) imoogi-emacsctl-max-bytes)
    (imoogi-emacsctl--reject "too-large"))
  ;; A unibyte string holds raw UTF-8 bytes (what `json-serialize' and
  ;; process output produce); decode it so the parser sees characters.
  (if (multibyte-string-p json)
      json
    (decode-coding-string json 'utf-8)))

(defun imoogi-emacsctl--parse (json)
  "Parse JSON (a string) into the strict representation.
Reject syntax errors (including nesting depth and encoding errors) with
parse, and anything but whitespace after the first value with
trailing-content."
  (with-temp-buffer
    (insert json)
    (goto-char (point-min))
    (let ((value (condition-case nil
                     (apply #'json-parse-buffer imoogi-emacsctl--json-options)
                   (json-error (imoogi-emacsctl--reject "parse")))))
      (skip-chars-forward " \t\n\r")
      (unless (eobp)
        (imoogi-emacsctl--reject "trailing-content"))
      value)))

(defun imoogi-emacsctl--duplicate-keys-p (object)
  "Return non-nil when the alist OBJECT holds the same key twice."
  (let ((keys (mapcar #'car object)))
    (/= (length keys) (length (delete-dups (copy-sequence keys))))))

;;; Validation

(defun imoogi-emacsctl--field (object key kind &optional optional)
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
          (imoogi-emacsctl--reject "bad-field"))
        value))))

(defun imoogi-emacsctl--validate (event)
  "Validate EVENT (parsed JSON) and return it as a normalized plist.
Checks run in the order duplicate keys, version, type, fields, paths, so
that the first violation in that order is the one reported."
  (let ((payload (and (listp event) (cdr (assq 'payload event)))))
    (when (or (and (listp event) (imoogi-emacsctl--duplicate-keys-p event))
              (and (consp payload) (imoogi-emacsctl--duplicate-keys-p payload)))
      (imoogi-emacsctl--reject "duplicate-key"))
    (unless (and (listp event) (equal (cdr (assq 'version event)) "1"))
      (imoogi-emacsctl--reject "bad-version"))
    (let ((type (cdr (assq 'type event))))
      (unless (member type imoogi-emacsctl--types)
        (imoogi-emacsctl--reject "unknown-type"))
      (imoogi-emacsctl--field event 'timestamp 'text)
      (let ((project (imoogi-emacsctl--field event 'project 'string t))
            (session (imoogi-emacsctl--field event 'session 'string t)))
        ;; `payload' must be a JSON object: an alist, or nil for {}.
        (unless (and (assq 'payload event) (listp payload))
          (imoogi-emacsctl--reject "bad-field"))
        (imoogi-emacsctl--resolve-paths
         (append (list :type type :project project :session session)
                 (imoogi-emacsctl--validate-payload type payload)))))))

(defun imoogi-emacsctl--resolve-path (path)
  "Return the real path of the payload PATH or reject the event.
PATH must be absolute and local, and after resolving symbolic links it must
name an existing regular file (not a directory, device, FIFO, socket, or
dangling link); otherwise the reason is bad-path. The remote test never
opens a connection, and it runs again on the resolved name so a link cannot
lead onto a remote file. A resolved file larger than
`imoogi-emacsctl-max-payload-bytes' then rejects the whole event with
payload-too-large; this runs during validation, before dispatch, so a
rejected task-finished shows no notice either."
  (unless (and (file-name-absolute-p path) (not (file-remote-p path)))
    (imoogi-emacsctl--reject "bad-path"))
  (let ((true (file-truename path)))
    (unless (and (not (file-remote-p true)) (file-regular-p true))
      (imoogi-emacsctl--reject "bad-path"))
    (let ((size (file-attribute-size (file-attributes true))))
      (unless size
        (imoogi-emacsctl--reject "bad-path"))
      (when (> size imoogi-emacsctl-max-payload-bytes)
        (imoogi-emacsctl--reject "payload-too-large")))
    true))

(defun imoogi-emacsctl--resolve-paths (event)
  "Replace every payload path in the plist EVENT by its checked real path.
The original name is kept under :name for notices."
  (dolist (key '(:path :artifact))
    (when-let* ((path (plist-get event key)))
      (setq event (plist-put event key (imoogi-emacsctl--resolve-path path)))
      (unless (plist-get event :name)
        (setq event (plist-put event :name (file-name-nondirectory path))))))
  event)

(defun imoogi-emacsctl--validate-payload (type payload)
  "Validate PAYLOAD of TYPE and return its fields as a plist."
  (pcase type
    ("message"
     (list :text (imoogi-emacsctl--field payload 'text 'text)))
    ("open-file"
     (list :path (imoogi-emacsctl--field payload 'path 'string)))
    ("goto-location"
     (list :path (imoogi-emacsctl--field payload 'path 'string)
           :line (imoogi-emacsctl--field payload 'line 'positive)
           :column (or (imoogi-emacsctl--field payload 'column 'positive t) 1)))
    ("artifact-created"
     (list :path (imoogi-emacsctl--field payload 'path 'string)
           :artifact-type (imoogi-emacsctl--field payload 'artifactType 'string t)
           :title (imoogi-emacsctl--field payload 'title 'string t)))
    ("task-finished"
     (let ((status (imoogi-emacsctl--field payload 'status 'string)))
       (unless (member status '("success" "failed"))
         (imoogi-emacsctl--reject "bad-field"))
       (list :status status
             :summary (imoogi-emacsctl--field payload 'summary 'string t)
             :artifact (imoogi-emacsctl--field payload 'artifact 'string t))))))

;;; Notification and log

(defun imoogi-emacsctl--notify (text)
  "Show TEXT, made one line, in the echo area without format interpretation."
  (message "%s" (imoogi-emacsctl--one-line text)))

(defun imoogi-emacsctl--log-buffer ()
  "Return the log buffer, creating it (read-only) when it does not exist."
  (or (get-buffer imoogi-emacsctl-log-buffer-name)
      (with-current-buffer (get-buffer-create imoogi-emacsctl-log-buffer-name)
        (special-mode)
        (current-buffer))))

(defun imoogi-emacsctl--log-value (value)
  "Return VALUE as a one-line log token, or - when it is not a string."
  (if (stringp value) (imoogi-emacsctl--one-line value) "-"))

(defun imoogi-emacsctl--log (event result detail)
  "Append one line for EVENT with RESULT and optional DETAIL to the log.
EVENT is the parsed JSON (possibly nil or not an object); RESULT is ok or
a rejection reason."
  (let* ((object (and (listp event) event))
         (line (concat
                (format-time-string "[%Y-%m-%d %H:%M:%S] ")
                (imoogi-emacsctl--log-value (cdr (assq 'type object)))
                " project=" (imoogi-emacsctl--log-value (cdr (assq 'project object)))
                " session=" (imoogi-emacsctl--log-value (cdr (assq 'session object)))
                " " result
                (if (and (stringp detail) (not (string-empty-p detail)))
                    (concat " " (imoogi-emacsctl--one-line detail))
                  ""))))
    (with-current-buffer (imoogi-emacsctl--log-buffer)
      (let ((inhibit-read-only t))
        (save-excursion
          (goto-char (point-max))
          (insert line "\n"))))))

;;; Display

(defconst imoogi-emacsctl--display-action
  '((display-buffer-reuse-window
     display-buffer-use-some-window
     display-buffer-pop-up-window)
    (inhibit-same-window . t)
    (reusable-frames . visible))
  "`display-buffer' action for agent files (plan.md D-5).
The selected window is never a candidate, so the user's window, frame, and
point stay where they are; a file shown only in the selected window is
shown once more elsewhere.")

(defun imoogi-emacsctl--visit (file)
  "Return a buffer visiting FILE without asking the user anything.
The causes of indirect prompts are removed: no large-file warning, no
changed-on-disk question (NOWARN), only safe file-local variables, and no
`eval' local variable."
  (let ((large-file-warning-threshold nil)
        (enable-local-variables :safe)
        (enable-local-eval nil))
    (find-file-noselect file t)))

(defun imoogi-emacsctl--display (file &optional line column)
  "Show FILE in a window other than the selected one without selecting it.
When LINE is non-nil, move that window's point to LINE and COLUMN (both
1-based); a line past the end clamps to the line holding `point-max' and
a column past the end of the line clamps to the line end."
  (let* ((buffer (imoogi-emacsctl--visit file))
         (window (display-buffer buffer imoogi-emacsctl--display-action)))
    (when (and line (window-live-p window))
      (set-window-point
       window
       (with-current-buffer buffer
         (save-excursion
           (save-restriction
             (widen)
             (goto-char (point-min))
             (if (>= line (line-number-at-pos (point-max)))
                 (goto-char (point-max))
               (forward-line (1- line)))
             (move-to-column (1- (min column most-positive-fixnum)))
             (point))))))
    window))

;;; Dispatch

(defun imoogi-emacsctl--notice-for-finish (event)
  "Return the echo-area notice for the task-finished EVENT."
  (let ((summary (plist-get event :summary)))
    (concat (if (equal (plist-get event :status) "success")
                "✓ Agent task finished"
              "✗ Agent task failed")
            (if (and summary (not (string-empty-p summary)))
                (concat ": " summary)
              ""))))

(defun imoogi-emacsctl-dispatch (event)
  "Carry out the validated, normalized EVENT plist.
Handlers that show a file display it first and notify last, so the notice
is the final message of the call. Return a detail string for the log."
  (pcase (plist-get event :type)
    ("message"
     (imoogi-emacsctl--notify (plist-get event :text))
     (plist-get event :text))
    ("open-file"
     (imoogi-emacsctl--display (plist-get event :path))
     (plist-get event :path))
    ("goto-location"
     (imoogi-emacsctl--display (plist-get event :path)
                            (plist-get event :line)
                            (plist-get event :column))
     (format "%s:%d:%d" (plist-get event :path)
             (plist-get event :line) (plist-get event :column)))
    ("artifact-created"
     (imoogi-emacsctl--display (plist-get event :path))
     (imoogi-emacsctl--notify (concat "Agent artifact created: "
                                   (or (plist-get event :title)
                                       (plist-get event :name))))
     (string-join (delq nil (list (plist-get event :artifact-type)
                                  (plist-get event :title)
                                  (plist-get event :path)))
                  " "))
    ("task-finished"
     (when-let* ((artifact (plist-get event :artifact)))
       (imoogi-emacsctl--display artifact))
     (imoogi-emacsctl--notify (imoogi-emacsctl--notice-for-finish event))
     (string-join (delq nil (list (plist-get event :status)
                                  (plist-get event :summary)
                                  (plist-get event :artifact)))
                  " "))))

;;; Entry points

(defun imoogi-emacsctl--receive (reader)
  "Run one event through READER, validation, dispatch, and the log.
READER returns the JSON string. Return \"ok\" or \"error:<reason>\"; no
elisp error escapes. User interaction is inhibited throughout, so a stray
prompt becomes an error:handler rejection instead of hanging the server."
  (let (event detail)
    (let ((reason
           (condition-case err
               (catch 'imoogi-emacsctl--reject
                 (let ((inhibit-interaction t))
                   (setq event (imoogi-emacsctl--parse (funcall reader)))
                   (setq detail (imoogi-emacsctl-dispatch
                                 (imoogi-emacsctl--validate event))))
                 nil)
             (error
              (setq detail (error-message-string err))
              "handler"))))
      (ignore-errors
        (imoogi-emacsctl--log event (or reason "ok") detail))
      (if reason (concat "error:" reason) "ok"))))

;; @MX:ANCHOR: [AUTO] File entry point called by the fixed emacsclient
;; expression of the imoogi-emacsctl CLI (internal/emacsctl).
;; @MX:REASON: External contract — the CLI's evaluation expression names this
;; function and parses its "ok"/"error:<reason>" return value.
(defun imoogi-emacsctl-receive-file (file)
  "Receive the agent event stored in FILE and return a status string.
FILE is read, never modified or deleted. The result is \"ok\" or
\"error:<reason>\"."
  (imoogi-emacsctl--receive (lambda () (imoogi-emacsctl--read-file file))))

;; @MX:ANCHOR: [AUTO] String entry point sharing the validation and dispatch
;; path of `imoogi-emacsctl-receive-file'.
;; @MX:REASON: Protocol contract — both entry points must return the same
;; status and cause the same side effects for the same event.
(defun imoogi-emacsctl-receive-json (json)
  "Receive the agent event in the JSON string and return a status string."
  (imoogi-emacsctl--receive (lambda () (imoogi-emacsctl--check-string json))))

(provide 'imoogi-emacsctl)
;;; 30-emacsctl.el ends here
