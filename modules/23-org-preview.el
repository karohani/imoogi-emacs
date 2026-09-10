;;; 23-org-preview.el --- live Org HTML preview client -*- lexical-binding: t; -*-

;;; Code:

(imoogi-require "23-org-preview" 'org 'org-element 'url 'url-util 'json 'browse-url 'seq)

(require 'cl-lib)
(require 'seq)
(require 'subr-x)
(require 'url-util)

(declare-function org-element-at-point "org-element")
(declare-function org-element-property "org-element")
(declare-function org-element-type "org-element")
(declare-function project-current "project")
(declare-function project-root "project")

(defgroup imoogi-org-preview nil
  "Live Org HTML preview backed by the local imoogi-org-preview server."
  :group 'org)

(defcustom imoogi-org-preview-command
  '("imoogi-org-preview" "--host" "127.0.0.1" "--port" "0" "--print-bootstrap-json")
  "Command used to start the preview server.
The server is expected to print one JSON line containing `port' and `token'."
  :type '(repeat string))

(defcustom imoogi-org-preview-host "127.0.0.1"
  "Loopback host used by the Org preview client."
  :type 'string)

(defcustom imoogi-org-preview-port nil
  "Port of an already-running preview server.
When nil, `imoogi-org-preview-mode' starts `imoogi-org-preview-command'."
  :type '(choice (const :tag "Start local server" nil) integer))

(defcustom imoogi-org-preview-token nil
  "Token of an already-running preview server."
  :type '(choice (const :tag "Use server bootstrap token" nil) string))

(defcustom imoogi-org-preview-debounce-seconds 0.5
  "Trailing debounce before sending an edited Org buffer revision."
  :type 'number)

(defcustom imoogi-org-preview-reconnect-seconds 1.0
  "Delay before reconnecting the Emacs event stream."
  :type 'number)

(defcustom imoogi-org-preview-navigation-sync nil
  "When non-nil, enable optional bidirectional cursor synchronization."
  :type 'boolean)

(defconst imoogi-org-preview-protocol-version "org-preview/v1")

(defvar imoogi-org-preview--server-process nil)
(defvar imoogi-org-preview--server-output "")
(defvar imoogi-org-preview--server-port nil)
(defvar imoogi-org-preview--server-token nil)
(defvar imoogi-org-preview--buffers nil)

(defvar-local imoogi-org-preview--open-pending nil)
(defvar-local imoogi-org-preview--sending nil)
(defvar-local imoogi-org-preview--dirty nil)

(defvar-local imoogi-org-preview--status 'disconnected)
(defvar-local imoogi-org-preview--session-id nil)
(defvar-local imoogi-org-preview--buffer-id nil)
(defvar-local imoogi-org-preview--revision 0)
(defvar-local imoogi-org-preview--update-timer nil)
(defvar-local imoogi-org-preview--nav-timer nil)
(defvar-local imoogi-org-preview--sse-process nil)
(defvar-local imoogi-org-preview--sse-buffer "")
(defvar-local imoogi-org-preview--sse-in-headers t)
(defvar-local imoogi-org-preview--reconnect-timer nil)
(defvar-local imoogi-org-preview--suppress-navigation nil)
(defvar-local imoogi-org-preview--sent-events nil)

(defun imoogi-org-preview--status-text ()
  "Return the modeline status for the current Org preview buffer."
  (format " OrgPreview[%s]" imoogi-org-preview--status))

;;;###autoload
(define-minor-mode imoogi-org-preview-mode
  "Send unsaved Org buffer revisions to the local HTML preview server."
  :init-value nil
  :lighter (:eval (imoogi-org-preview--status-text))
  (if imoogi-org-preview-mode
      (imoogi-org-preview--enable)
    (imoogi-org-preview--disable)))

;;;###autoload
(defun imoogi-org-preview ()
  "Enable live Org HTML preview for the current buffer and open the browser."
  (interactive)
  (unless (derived-mode-p 'org-mode)
    (user-error "Org preview is only available in org-mode buffers"))
  (imoogi-org-preview-mode 1)
  (imoogi-org-preview-open))

;;;###autoload
(defun imoogi-org-preview-open ()
  "Open the current buffer's preview page in the default browser."
  (interactive)
  (unless imoogi-org-preview-mode
    (imoogi-org-preview-mode 1))
  (if-let* ((port (imoogi-org-preview--port))
            (token (imoogi-org-preview--token))
            (session imoogi-org-preview--session-id))
      (progn
        (setq imoogi-org-preview--open-pending nil)
        (browse-url
       (format "http://%s:%d/preview?session=%s&buffer=%s&token=%s"
               imoogi-org-preview-host port
               (url-hexify-string session)
               (url-hexify-string imoogi-org-preview--buffer-id)
               (url-hexify-string token))))
    (setq imoogi-org-preview--open-pending t)
    (message "Org preview is starting; the browser will open when ready")))

;;;###autoload
(defun imoogi-org-preview-stop ()
  "Stop live Org preview for the current buffer."
  (interactive)
  (imoogi-org-preview-mode -1))

(defun imoogi-org-preview--enable ()
  "Enable preview state and hooks for the current buffer."
  (unless (derived-mode-p 'org-mode)
    (setq imoogi-org-preview-mode nil)
    (user-error "Org preview is only available in org-mode buffers"))
  (setq imoogi-org-preview--session-id (or imoogi-org-preview--session-id
                                           (imoogi-org-preview--make-id "session"))
        imoogi-org-preview--buffer-id (or imoogi-org-preview--buffer-id
                                          (imoogi-org-preview--make-buffer-id)))
  (cl-pushnew (current-buffer) imoogi-org-preview--buffers)
  (add-hook 'after-change-functions #'imoogi-org-preview--after-change nil t)
  (add-hook 'kill-buffer-hook #'imoogi-org-preview--disable nil t)
  (when imoogi-org-preview-navigation-sync
    (add-hook 'post-command-hook #'imoogi-org-preview--post-command nil t))
  (imoogi-org-preview--set-status 'connecting)
  (imoogi-org-preview--ensure-server)
  (imoogi-org-preview--schedule-update))

(defun imoogi-org-preview--disable ()
  "Disable preview hooks and transient resources for the current buffer."
  (remove-hook 'kill-buffer-hook #'imoogi-org-preview--disable t)
  (remove-hook 'after-change-functions #'imoogi-org-preview--after-change t)
  (remove-hook 'post-command-hook #'imoogi-org-preview--post-command t)
  (mapc #'imoogi-org-preview--cancel-timer
        (list imoogi-org-preview--update-timer
              imoogi-org-preview--nav-timer
              imoogi-org-preview--reconnect-timer))
  (when (process-live-p imoogi-org-preview--sse-process)
    (delete-process imoogi-org-preview--sse-process))
  (setq imoogi-org-preview--open-pending nil
        imoogi-org-preview--sending nil
        imoogi-org-preview--dirty nil
        imoogi-org-preview--update-timer nil
        imoogi-org-preview--nav-timer nil
        imoogi-org-preview--reconnect-timer nil
        imoogi-org-preview--sse-process nil
        imoogi-org-preview--sse-buffer ""
        imoogi-org-preview--sse-in-headers t
        imoogi-org-preview--status 'disconnected
        imoogi-org-preview--suppress-navigation nil)
  (setq imoogi-org-preview--buffers
        (delq (current-buffer) imoogi-org-preview--buffers))
  (when (and (null imoogi-org-preview--buffers)
             (process-live-p imoogi-org-preview--server-process))
    (delete-process imoogi-org-preview--server-process)))

(defun imoogi-org-preview--cancel-timer (timer)
  "Cancel TIMER when it is active."
  (when (timerp timer)
    (cancel-timer timer)))

(defun imoogi-org-preview--set-status (status)
  "Set current preview STATUS and refresh the mode line."
  (setq imoogi-org-preview--status status)
  (force-mode-line-update))

(defun imoogi-org-preview--make-id (prefix)
  "Return an unguessable-ish local id with PREFIX."
  (format "%s-%s-%06x"
          prefix
          (format-time-string "%Y%m%d%H%M%S%N")
          (random #x1000000)))

(defun imoogi-org-preview--make-buffer-id ()
  "Return a stable buffer id for this preview session."
  (secure-hash 'sha1
               (format "%s\0%s\0%s"
                       (or buffer-file-name "")
                       (buffer-name)
                       (imoogi-org-preview--make-id "buffer"))))

(defun imoogi-org-preview--port ()
  "Return the active preview server port."
  (or imoogi-org-preview-port imoogi-org-preview--server-port))

(defun imoogi-org-preview--token ()
  "Return the active preview server token."
  (or imoogi-org-preview-token imoogi-org-preview--server-token))

(defun imoogi-org-preview--ensure-server ()
  "Ensure the preview server is available, then connect the current buffer."
  (cond
   ((and (imoogi-org-preview--port) (imoogi-org-preview--token))
    (imoogi-org-preview--connect-current-buffer))
   ((process-live-p imoogi-org-preview--server-process)
    nil)
   (t
    (imoogi-org-preview--start-server))))

(defun imoogi-org-preview--start-server ()
  "Start the configured preview server command."
  (let* ((configured (car imoogi-org-preview-command))
         (local (expand-file-name "bin/imoogi-org-preview" imoogi-emacs-dir))
         (program (or (and configured (executable-find configured))
                      (and (equal configured "imoogi-org-preview")
                           (file-executable-p local) local))))
    (if (and program (executable-find program))
        (setq imoogi-org-preview--server-output ""
              imoogi-org-preview--server-process
              (make-process
               :name "imoogi-org-preview"
               :buffer "*imoogi-org-preview-server*"
               :command (cons program (cdr imoogi-org-preview-command))
               :noquery t
               :filter #'imoogi-org-preview--server-filter
               :sentinel #'imoogi-org-preview--server-sentinel))
      (imoogi-org-preview--mark-buffers-failed
       (format "Org preview server command not found: %s" program)))))

(defun imoogi-org-preview--server-filter (_process chunk)
  "Read server bootstrap information from CHUNK."
  (setq imoogi-org-preview--server-output
        (concat imoogi-org-preview--server-output chunk))
  (while (string-match "\\([^\n\r]+\\)[\n\r]+" imoogi-org-preview--server-output)
    (let ((line (match-string 1 imoogi-org-preview--server-output)))
      (setq imoogi-org-preview--server-output
            (substring imoogi-org-preview--server-output (match-end 0)))
      (imoogi-org-preview--maybe-read-bootstrap line))))

(defun imoogi-org-preview--maybe-read-bootstrap (line)
  "Parse a server bootstrap LINE when it contains JSON."
  (when (string-prefix-p "{" (string-trim-left line))
    (condition-case nil
        (let* ((json-object-type 'alist)
               (data (json-read-from-string line))
               (port (alist-get 'port data))
               (token (alist-get 'token data)))
          (when (and (integerp port) (stringp token))
            (setq imoogi-org-preview--server-port port
                  imoogi-org-preview--server-token token)
            (imoogi-org-preview--connect-buffers)))
      (error nil))))

(defun imoogi-org-preview--server-sentinel (_process event)
  "Handle server process EVENT."
  (unless (string-match-p "\\`\\(?:open\\|run\\)" event)
    (setq imoogi-org-preview--server-process nil
          imoogi-org-preview--server-port nil
          imoogi-org-preview--server-token nil)
    (imoogi-org-preview--mark-buffers-failed
     (format "Org preview server stopped: %s" (string-trim event)))))

(defun imoogi-org-preview--connect-buffers ()
  "Connect all active Org preview buffers to the server."
  (dolist (buffer (copy-sequence imoogi-org-preview--buffers))
    (when (buffer-live-p buffer)
      (with-current-buffer buffer
        (when imoogi-org-preview-mode
          (imoogi-org-preview--connect-current-buffer))))))

(defun imoogi-org-preview--connect-current-buffer ()
  "Connect the current buffer's event stream and send its latest revision."
  (when (and imoogi-org-preview-mode
             (imoogi-org-preview--port)
             (imoogi-org-preview--token))
    (when imoogi-org-preview-navigation-sync
      (imoogi-org-preview--connect-sse))
    (when imoogi-org-preview--open-pending
      (imoogi-org-preview-open))
    (imoogi-org-preview--schedule-update)))

(defun imoogi-org-preview--mark-buffers-failed (message)
  "Mark active buffers failed and display MESSAGE."
  (dolist (buffer (copy-sequence imoogi-org-preview--buffers))
    (when (buffer-live-p buffer)
      (with-current-buffer buffer
        (when imoogi-org-preview-mode
          (imoogi-org-preview--set-status 'failed)))))
  (message "%s" message))

(defun imoogi-org-preview--after-change (&rest _args)
  "Schedule a preview update after an Org buffer edit."
  (imoogi-org-preview--schedule-update))

(defun imoogi-org-preview--post-command ()
  "Schedule point synchronization unless this command came from preview."
  (unless imoogi-org-preview--suppress-navigation
    (imoogi-org-preview--schedule-navigation)))

(defun imoogi-org-preview--schedule-update ()
  "Send one trailing update for the current buffer."
  (setq imoogi-org-preview--dirty t)
  (imoogi-org-preview--cancel-timer imoogi-org-preview--update-timer)
  (setq imoogi-org-preview--update-timer
        (run-at-time imoogi-org-preview-debounce-seconds nil
                     #'imoogi-org-preview--send-update (current-buffer))))

(defun imoogi-org-preview--schedule-navigation ()
  "Send one trailing point navigation event for the current buffer."
  (imoogi-org-preview--cancel-timer imoogi-org-preview--nav-timer)
  (setq imoogi-org-preview--nav-timer
        (run-at-time 0.05 nil
                     #'imoogi-org-preview--send-navigation (current-buffer))))

(defun imoogi-org-preview--send-update (buffer)
  "Send BUFFER's current unsaved text to the preview server."
  (when (buffer-live-p buffer)
    (with-current-buffer buffer
      (setq imoogi-org-preview--update-timer nil)
      (when (and imoogi-org-preview-mode
                 (not imoogi-org-preview--sending)
                 (imoogi-org-preview--port)
                 (imoogi-org-preview--token))
        (setq imoogi-org-preview--sending t
              imoogi-org-preview--dirty nil)
        (cl-incf imoogi-org-preview--revision)
        (condition-case err
            (save-restriction
              (widen)
              (imoogi-org-preview--post-json
               "/api/emacs/revisions"
               (imoogi-org-preview--build-update-payload)
               #'imoogi-org-preview--update-callback))
          (error
           (setq imoogi-org-preview--sending nil)
           (imoogi-org-preview--set-status 'failed)
           (message "Org preview: %s" (error-message-string err))))))))

(defun imoogi-org-preview--send-navigation (buffer)
  "Send BUFFER's current point location to the preview server."
  (when (buffer-live-p buffer)
    (with-current-buffer buffer
      (setq imoogi-org-preview--nav-timer nil)
      (when (and imoogi-org-preview-mode
                 (not imoogi-org-preview--suppress-navigation)
                 (imoogi-org-preview--port)
                 (imoogi-org-preview--token))
        (let ((event-id (imoogi-org-preview--make-id "event")))
          (push event-id imoogi-org-preview--sent-events)
          (setq imoogi-org-preview--sent-events
                (seq-take imoogi-org-preview--sent-events 64))
          (imoogi-org-preview--post-json
           "/api/emacs/navigation"
           (imoogi-org-preview--base-payload
            `((event_id . ,event-id)
              (cursor_byte . ,(imoogi-org-preview--cursor-byte))
              (element_id . ,(imoogi-org-preview--current-element-id))
              (range . ,(imoogi-org-preview--current-element-range))))
           #'imoogi-org-preview--navigation-callback))))))

(defun imoogi-org-preview--base-payload (extra)
  "Return the base protocol envelope merged with EXTRA."
  (append
   `((version . ,imoogi-org-preview-protocol-version)
     (protocol_version . 1)
     (session_id . ,imoogi-org-preview--session-id)
     (buffer_id . ,imoogi-org-preview--buffer-id)
     (revision . ,imoogi-org-preview--revision)
     (origin . "emacs"))
   extra))

(defun imoogi-org-preview--build-update-payload ()
  "Return a JSON-ready payload for the current buffer revision."
  (imoogi-org-preview--base-payload
   `((event_id . ,(imoogi-org-preview--make-id "event"))
     (path . ,(if buffer-file-name (expand-file-name buffer-file-name) ""))
     (buffer_name . ,(buffer-name))
     (allowed_roots . ,(imoogi-org-preview--allowed-roots))
     (text . ,(buffer-substring-no-properties (point-min) (point-max)))
     (cursor_byte . ,(imoogi-org-preview--cursor-byte)))))

(defun imoogi-org-preview--project-root ()
  "Return the current project root or containing directory."
  (or (when (fboundp 'project-current)
        (when-let* ((project (project-current nil)))
          (expand-file-name (project-root project))))
      (when buffer-file-name
        (file-name-directory (expand-file-name buffer-file-name)))
      default-directory))

(defun imoogi-org-preview--allowed-roots ()
  "Return directories the preview server may use to resolve file links."
  (seq-uniq
   (delq nil
         (list (when-let* ((root (imoogi-org-preview--project-root)))
                 (expand-file-name root))
               (when buffer-file-name
                 (file-name-directory (expand-file-name buffer-file-name)))))
   #'string=))

(defun imoogi-org-preview--cursor-byte ()
  "Return zero-based UTF-8 byte offset for current point."
  (string-bytes
   (encode-coding-string
    (buffer-substring-no-properties (point-min) (point))
    'utf-8)))

(defun imoogi-org-preview--current-element-id ()
  "Return a deterministic source-range id for the Org element at point."
  (when (derived-mode-p 'org-mode)
    (condition-case nil
        (let* ((element (org-element-at-point))
               (type (org-element-type element))
               (begin (org-element-property :begin element))
               (end (org-element-property :end element)))
          (when (and type begin end)
            (format "org-%s-%d-%d" type (1- begin) (1- end))))
      (error nil))))

(defun imoogi-org-preview--current-element-range ()
  "Return JSON-ready source byte range for the Org element at point."
  (when (derived-mode-p 'org-mode)
    (condition-case nil
        (let* ((element (org-element-at-point))
               (begin (org-element-property :begin element))
               (end (org-element-property :end element)))
          (when (and begin end)
            `((start . ,(imoogi-org-preview--byte-offset-at begin))
              (end . ,(imoogi-org-preview--byte-offset-at end)))))
      (error nil))))

(defun imoogi-org-preview--byte-offset-at (position)
  "Return zero-based UTF-8 byte offset for buffer POSITION."
  (save-excursion
    (goto-char (max (point-min) (min (point-max) position)))
    (imoogi-org-preview--cursor-byte)))

(defun imoogi-org-preview--post-json (path payload callback)
  "POST PAYLOAD to PATH and run CALLBACK with the URL status."
  (let* ((url-request-method "POST")
         (url-request-extra-headers
          `(("Content-Type" . "application/json; charset=utf-8")
            ("X-Org-Preview-Token" . ,(imoogi-org-preview--token))))
         (url-request-data
          (encode-coding-string (json-encode payload) 'utf-8))
         (url (format "http://%s:%d%s"
                      imoogi-org-preview-host
                      (imoogi-org-preview--port)
                      path))
         (source-buffer (current-buffer)))
    (url-retrieve url callback (list source-buffer) t t)))

(defun imoogi-org-preview--http-ok-p ()
  "Return non-nil when the current URL response buffer has a 2xx status."
  (save-excursion
    (goto-char (point-min))
    (looking-at "HTTP/[0-9.]+ 2[0-9][0-9]\\_>")))

(defun imoogi-org-preview--update-callback (status source-buffer)
  "Handle an update response with URL STATUS for SOURCE-BUFFER."
  (when (buffer-live-p source-buffer)
    (with-current-buffer source-buffer
      (setq imoogi-org-preview--sending nil)
      (when (and imoogi-org-preview-mode imoogi-org-preview--dirty)
        (imoogi-org-preview--schedule-update))))
  (imoogi-org-preview--response-callback status source-buffer))

(defun imoogi-org-preview--navigation-callback (status source-buffer)
  "Handle a navigation response with URL STATUS for SOURCE-BUFFER."
  (imoogi-org-preview--response-callback status source-buffer))

(defun imoogi-org-preview--response-callback (status source-buffer)
  "Handle a URL response STATUS for SOURCE-BUFFER."
  (let ((response-buffer (current-buffer)))
    (unwind-protect
      (when (and (buffer-live-p source-buffer)
                 (buffer-local-value 'imoogi-org-preview-mode source-buffer))
        (with-current-buffer source-buffer
          (if (and (not (plist-get status :error))
                   (with-current-buffer response-buffer
                     (imoogi-org-preview--http-ok-p)))
              (imoogi-org-preview--set-status 'connected)
            (imoogi-org-preview--set-status 'reconnecting)
            (imoogi-org-preview--schedule-reconnect))))
      (when (buffer-live-p response-buffer)
        (kill-buffer response-buffer)))))

(defun imoogi-org-preview--connect-sse ()
  "Connect or reconnect the current buffer's server-sent event stream."
  (when (process-live-p imoogi-org-preview--sse-process)
    (delete-process imoogi-org-preview--sse-process))
  (setq imoogi-org-preview--sse-buffer ""
        imoogi-org-preview--sse-in-headers t
        imoogi-org-preview--sse-process
        (make-network-process
         :name (format "imoogi-org-preview-sse-%s" imoogi-org-preview--session-id)
         :host imoogi-org-preview-host
         :service (imoogi-org-preview--port)
         :coding 'utf-8
         :noquery t
         :filter #'imoogi-org-preview--sse-filter
         :sentinel #'imoogi-org-preview--sse-sentinel))
  (process-put imoogi-org-preview--sse-process 'buffer (current-buffer))
  (process-send-string
   imoogi-org-preview--sse-process
   (format (concat "GET /api/emacs/events?session=%s&buffer=%s HTTP/1.1\r\n"
                   "Host: %s:%d\r\n"
                   "Accept: text/event-stream\r\n"
                   "Cache-Control: no-cache\r\n"
                   "X-Org-Preview-Token: %s\r\n"
                   "Connection: keep-alive\r\n\r\n")
           (url-hexify-string imoogi-org-preview--session-id)
           (url-hexify-string imoogi-org-preview--buffer-id)
           imoogi-org-preview-host
           (imoogi-org-preview--port)
           (imoogi-org-preview--token))))

(defun imoogi-org-preview--sse-filter (process chunk)
  "Process SSE CHUNK for PROCESS."
  (when-let* ((buffer (process-get process 'buffer)))
    (when (buffer-live-p buffer)
      (with-current-buffer buffer
        (setq imoogi-org-preview--sse-buffer
              (concat imoogi-org-preview--sse-buffer chunk))
        (when imoogi-org-preview--sse-in-headers
          (when (string-match "\r?\n\r?\n" imoogi-org-preview--sse-buffer)
            (setq imoogi-org-preview--sse-buffer
                  (substring imoogi-org-preview--sse-buffer (match-end 0))
                  imoogi-org-preview--sse-in-headers nil)
            (imoogi-org-preview--set-status 'connected)))
        (unless imoogi-org-preview--sse-in-headers
          (imoogi-org-preview--consume-sse-buffer))))))

(defun imoogi-org-preview--consume-sse-buffer ()
  "Consume complete events from the buffer-local SSE buffer."
  (while (string-match "\r?\n\r?\n" imoogi-org-preview--sse-buffer)
    (let ((raw (substring imoogi-org-preview--sse-buffer 0 (match-beginning 0))))
      (setq imoogi-org-preview--sse-buffer
            (substring imoogi-org-preview--sse-buffer (match-end 0)))
      (imoogi-org-preview--handle-sse-event raw))))

(defun imoogi-org-preview--handle-sse-event (raw)
  "Handle one raw SSE event block."
  (let ((data-lines nil))
    (dolist (line (split-string raw "\r?\n"))
      (when (string-prefix-p "data:" line)
        (push (string-trim-left (substring line 5)) data-lines)))
    (when data-lines
      (condition-case nil
          (let* ((json-object-type 'alist)
                 (json-array-type 'list)
                 (data (json-read-from-string
                        (string-join (nreverse data-lines) "\n"))))
            (imoogi-org-preview--handle-event data))
        (error nil)))))

(defun imoogi-org-preview--handle-event (event)
  "Apply one server EVENT to the current buffer."
  (let ((version (or (alist-get 'version event)
                     (alist-get 'protocol_version event)))
        (buffer-id (alist-get 'buffer_id event))
        (origin (alist-get 'origin event))
        (event-id (alist-get 'event_id event))
        (type (alist-get 'type event)))
    (when (and (or (equal version imoogi-org-preview-protocol-version)
                   (equal version 1))
               (equal buffer-id imoogi-org-preview--buffer-id)
               (not (member event-id imoogi-org-preview--sent-events)))
      (pcase (or type (when (or (alist-get 'element_id event) (alist-get 'range event)) "navigate"))
        ("navigate"
         (unless (equal origin "emacs")
           (imoogi-org-preview--apply-navigation event)))
        ("status"
         (when-let* ((status (alist-get 'status event)))
           (imoogi-org-preview--set-status (intern status))))))))

(defun imoogi-org-preview--apply-navigation (event)
  "Move point according to a browser-originated navigation EVENT."
  (let* ((point-data (alist-get 'point event))
         (range-data (alist-get 'range event))
         (offset (or (alist-get 'offset point-data)
                     (alist-get 'offset event)
                     (alist-get 'start range-data))))
    (when (integerp offset)
      (setq imoogi-org-preview--suppress-navigation t)
      (goto-char (imoogi-org-preview--byte-offset-to-position offset))
      (when-let* ((window (get-buffer-window (current-buffer) t)))
        (with-selected-window window
          (recenter)))
      (run-at-time 0.1 nil
                   (lambda (buffer)
                     (when (buffer-live-p buffer)
                       (with-current-buffer buffer
                         (setq imoogi-org-preview--suppress-navigation nil))))
                   (current-buffer)))))

(defun imoogi-org-preview--byte-offset-to-position (offset)
  "Return buffer position for zero-based UTF-8 byte OFFSET."
  (let ((target (max 0 offset))
        (pos (point-min))
        (bytes 0))
    (save-excursion
      (goto-char (point-min))
      (while (and (< bytes target) (< (point) (point-max)))
        (let* ((char (char-after))
               (width (string-bytes (encode-coding-string (string char) 'utf-8))))
          (setq bytes (+ bytes width)
                pos (1+ (point)))
          (forward-char 1))))
    (max (point-min) (min (point-max) pos))))

(defun imoogi-org-preview--sse-sentinel (process event)
  "Handle SSE PROCESS EVENT."
  (when-let* ((buffer (process-get process 'buffer)))
    (when (buffer-live-p buffer)
      (with-current-buffer buffer
        (unless (or (not imoogi-org-preview-mode)
                    (string-match-p "\\`\\(?:open\\|run\\)" event))
          (imoogi-org-preview--set-status 'reconnecting)
          (imoogi-org-preview--schedule-reconnect))))))

(defun imoogi-org-preview--schedule-reconnect ()
  "Schedule an SSE reconnect for the current buffer."
  (imoogi-org-preview--cancel-timer imoogi-org-preview--reconnect-timer)
  (setq imoogi-org-preview--reconnect-timer
        (run-at-time imoogi-org-preview-reconnect-seconds nil
                     #'imoogi-org-preview--reconnect (current-buffer))))

(defun imoogi-org-preview--reconnect (buffer)
  "Reconnect BUFFER to the current preview server."
  (when (buffer-live-p buffer)
    (with-current-buffer buffer
      (setq imoogi-org-preview--reconnect-timer nil)
      (when imoogi-org-preview-mode
        (imoogi-org-preview--ensure-server)))))

(provide 'imoogi-org-preview)
;;; 23-org-preview.el ends here
