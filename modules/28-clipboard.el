;;; 28-clipboard.el --- clipboard assets for Org and Markdown -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "28-clipboard" 'cl-lib 'json 'subr-x 'url-util)

(require 'cl-lib)
(require 'json)
(require 'subr-x)
(require 'url-util)

(defgroup imoogi-clipboard nil "Clipboard asset integration." :group 'imoogi)

(defcustom imoogi-clipboard-command '("imoogi-clip")
  "Command used to run the offline clipboard helper."
  :type '(repeat string) :group 'imoogi-clipboard)

(defcustom imoogi-clipboard-inspect-timeout 0.12
  "Maximum seconds for metadata-only inspect and checkpoint calls."
  :type 'number :group 'imoogi-clipboard)

(defcustom imoogi-clipboard-lease-renew-interval 300
  "Seconds between staging lease renewals."
  :type 'integer :group 'imoogi-clipboard)

(defvar imoogi-clipboard--session-id nil)
(defvar imoogi-clipboard--process-start-id
  (format "%s:%s" (emacs-pid) (float-time before-init-time)))
(defvar imoogi-clipboard--kill-generation 0)
(defvar imoogi-clipboard--kill-checkpoint nil)
(defvar imoogi-clipboard--diagnostics-buffer "*imoogi-clipboard*")
(defvar imoogi-clipboard--lease-timer nil)
(defvar imoogi-clipboard--managed-buffers nil)
(defvar-local imoogi-clipboard--buffer-id nil)
(defvar-local imoogi-clipboard--generation 1)
(defvar-local imoogi-clipboard--pending nil)
(defvar-local imoogi-clipboard--transactions nil)
(defvar-local imoogi-clipboard--pending-renames nil)
(defvar imoogi-clipboard--refile-source-directory nil)

(defun imoogi-clipboard--id (prefix)
  "Return a locally unique identifier beginning with PREFIX."
  (format "%s-%s-%x" prefix (format-time-string "%s%N")
          (random most-positive-fixnum)))

(defun imoogi-clipboard--session-id ()
  "Return the process-wide clipboard session identifier."
  (or imoogi-clipboard--session-id
      (setq imoogi-clipboard--session-id (imoogi-clipboard--id "session"))))

(defun imoogi-clipboard--ensure-identity ()
  "Initialize and return the current buffer identity."
  (unless imoogi-clipboard--buffer-id
    (setq imoogi-clipboard--buffer-id (imoogi-clipboard--id "buffer")
          imoogi-clipboard--generation 1))
  `((session . ,(imoogi-clipboard--session-id))
    (buffer . ,imoogi-clipboard--buffer-id)
    (generation . ,imoogi-clipboard--generation)))

(defun imoogi-clipboard--same-identity-p (left right)
  "Return non-nil when LEFT and RIGHT name the same buffer generation."
  (and (equal (alist-get 'session left) (alist-get 'session right))
       (equal (alist-get 'buffer left) (alist-get 'buffer right))
       (= (or (alist-get 'generation left) 0)
          (or (alist-get 'generation right) 0))))

(defun imoogi-clipboard--supported-buffer-p ()
  "Return non-nil when asset insertion is allowed in this buffer."
  (and (derived-mode-p 'org-mode 'markdown-mode 'gfm-mode)
       (not buffer-read-only)
       (not (and buffer-file-name (file-remote-p buffer-file-name)))))

(defun imoogi-clipboard--command ()
  "Return the helper command or nil when unavailable."
  (when-let* ((program (car imoogi-clipboard-command))
              (executable
               (or (executable-find program)
                   (let ((candidate (expand-file-name
                                     (concat "bin/" program) imoogi-emacs-dir)))
                     (and (file-executable-p candidate) candidate)))))
    (cons executable (cdr imoogi-clipboard-command))))

(defun imoogi-clipboard--request (operation &optional fields)
  "Build a protocol request for OPERATION with FIELDS."
  (append `((protocol_version . 1) (operation . ,operation)
            (correlation_id . ,(imoogi-clipboard--id "request")))
          fields))

(defun imoogi-clipboard--encode (value)
  "Encode VALUE as UTF-8 JSON."
  (encode-coding-string
   (json-serialize value :null-object nil :false-object :false) 'utf-8))

(defun imoogi-clipboard--decode (buffer)
  "Decode a response envelope from BUFFER."
  (with-current-buffer buffer
    (goto-char (point-min))
    (json-parse-buffer :object-type 'alist :array-type 'list
                       :null-object nil :false-object nil)))

(defun imoogi-clipboard--log (format-string &rest args)
  "Append a sanitized diagnostic using FORMAT-STRING and ARGS."
  (with-current-buffer (get-buffer-create imoogi-clipboard--diagnostics-buffer)
    (goto-char (point-max))
    (insert (format-time-string "%FT%T%z ")
            (apply #'format format-string args) "\n")))

(defun imoogi-clipboard-show-diagnostics ()
  "Display clipboard helper diagnostics."
  (interactive)
  (pop-to-buffer (get-buffer-create imoogi-clipboard--diagnostics-buffer)))

(defun imoogi-clipboard--call-sync (request timeout)
  "Send REQUEST and wait no more than TIMEOUT seconds."
  (when-let* ((command (imoogi-clipboard--command)))
    (let ((buffer (generate-new-buffer " *imoogi-clipboard-sync*"))
          process response)
      (unwind-protect
          (condition-case err
              (progn
                (setq process
                      (make-process :name "imoogi-clipboard-sync" :buffer buffer
                                    :command command :connection-type 'pipe
                                    :coding '(utf-8-unix . utf-8-unix) :noquery t))
                (process-send-string process (imoogi-clipboard--encode request))
                (process-send-eof process)
                (let ((deadline (+ (float-time) timeout)))
                  (while (and (process-live-p process) (< (float-time) deadline))
                    (accept-process-output process 0.01)))
                (when (process-live-p process)
                  (delete-process process)
                  (error "clipboard helper timed out"))
                (when (zerop (process-exit-status process))
                  (setq response (imoogi-clipboard--decode buffer))))
            (error (imoogi-clipboard--log "sync failure: %s"
                                          (error-message-string err))))
        (when (process-live-p process) (delete-process process))
        (kill-buffer buffer))
      response)))

(defun imoogi-clipboard--call-async (request callback)
  "Send REQUEST and invoke CALLBACK with a response or nil."
  (if-let* ((command (imoogi-clipboard--command)))
      (let* ((buffer (generate-new-buffer " *imoogi-clipboard-async*"))
             (process
              (make-process
               :name "imoogi-clipboard-async" :buffer buffer :command command
               :connection-type 'pipe :coding '(utf-8-unix . utf-8-unix)
               :noquery t
               :sentinel
               (lambda (proc _event)
                 (when (memq (process-status proc) '(exit signal))
                   (unwind-protect
                       (funcall callback
                                (and (zerop (process-exit-status proc))
                                     (condition-case err
                                         (imoogi-clipboard--decode buffer)
                                       (error
                                        (imoogi-clipboard--log
                                         "async decode failure: %s"
                                         (error-message-string err))
                                        nil))))
                     (kill-buffer buffer)))))))
        (process-send-string process (imoogi-clipboard--encode request))
        (process-send-eof process)
        process)
    (funcall callback nil)
    nil))

(defun imoogi-clipboard--owner ()
  "Return the explicit owner descriptor for the current buffer."
  (let* ((registry-matches
          (and buffer-file-name
               (fboundp 'imoogi-project-notes--read-registry)
               (seq-filter
                (lambda (entry)
                  (when-let* ((root (alist-get 'notes-dir entry)))
                    (file-in-directory-p (expand-file-name buffer-file-name)
                                         (expand-file-name root))))
                (imoogi-project-notes--read-registry))))
         (_ (when (> (length registry-matches) 1)
              (user-error "imoogi clipboard: overlapping project-notes registrations")))
         (registry-entry
          (or (car registry-matches)
              (and buffer-file-name
                   (fboundp 'imoogi-project-notes--current-entry)
                   (imoogi-project-notes--current-entry))))
         (notes-root
          (or (and (boundp 'imoogi-project-notes-directory-local)
                   imoogi-project-notes-directory-local)
              (alist-get 'notes-dir registry-entry))))
   (cond
   ((and notes-root buffer-file-name
         (file-in-directory-p (expand-file-name buffer-file-name)
                              (expand-file-name notes-root)))
    `((kind . "project_notes")
      (document . ,(expand-file-name buffer-file-name))
      (root . ,(expand-file-name notes-root))
      (registry_key . ,(or (alist-get 'key registry-entry)
                           (and (boundp 'imoogi-project-notes-source-root)
                                imoogi-project-notes-source-root)
                           "buffer-local"))))
   (buffer-file-name
    (let ((document (expand-file-name buffer-file-name)))
      `((kind . "standalone") (document . ,document)
        (root . ,(concat (file-name-sans-extension document) ".assets")))))
   (t
    (let ((identity (imoogi-clipboard--ensure-identity)))
      `((kind . "staging")
        (session . ,(alist-get 'session identity))
        (buffer . ,(alist-get 'buffer identity))
        (generation . ,(alist-get 'generation identity))))))))

(defun imoogi-clipboard--native-yank ()
  "Invoke the original interactive `yank'."
  (let ((imoogi-clipboard-mode nil)) (call-interactively #'yank)))

(defun imoogi-clipboard--external-offer-p (clipboard-id)
  "Return non-nil when CLIPBOARD-ID differs from the local-kill checkpoint."
  (and (stringp clipboard-id)
	   imoogi-clipboard--kill-checkpoint
	   (not (and (equal clipboard-id
				    (alist-get 'clipboard_id imoogi-clipboard--kill-checkpoint))
			 (= imoogi-clipboard--kill-generation
			    (or (alist-get 'kill_generation
					      imoogi-clipboard--kill-checkpoint) -1))))))

(defun imoogi-clipboard-yank (arg)
  "Paste text natively or import an external clipboard asset.
With prefix ARG, always invoke native `yank'."
  (interactive "P")
  (if (or arg (not (imoogi-clipboard--supported-buffer-p)))
      (imoogi-clipboard--native-yank)
    (let* ((inspection (imoogi-clipboard--call-sync
                        (imoogi-clipboard--request "inspect")
                        imoogi-clipboard-inspect-timeout))
           (kind (alist-get 'kind inspection))
           (clipboard-id (alist-get 'clipboard_id inspection)))
      (if (not (and (equal (alist-get 'status inspection) "ok")
                    (member kind '("files" "image")) clipboard-id
                    (imoogi-clipboard--external-offer-p clipboard-id)))
          (imoogi-clipboard--native-yank)
        (imoogi-clipboard--start-import "paste" clipboard-id nil)))))

(defun imoogi-clipboard--start-import (operation clipboard-id paths)
  "Start async OPERATION for CLIPBOARD-ID or explicit PATHS."
  (imoogi-clipboard--renew-lease)
  (let* ((buffer (current-buffer))
         (identity (imoogi-clipboard--ensure-identity))
	 (modification-tick (buffer-chars-modified-tick))
         (marker (copy-marker (point) t))
         (fields `((identity . ,identity) (owner . ,(imoogi-clipboard--owner))))
         (fields (if clipboard-id
                     (append fields `((expected_clipboard_id . ,clipboard-id))) fields))
         (fields (if paths (append fields `((paths . ,(vconcat paths)))) fields)))
    (push marker imoogi-clipboard--pending)
    (imoogi-clipboard--call-async
     (imoogi-clipboard--request operation fields)
     (lambda (response)
       (if (and (buffer-live-p buffer) (marker-buffer marker)
                (= modification-tick
		   (with-current-buffer buffer (buffer-chars-modified-tick)))
		(with-current-buffer buffer
		  (imoogi-clipboard--same-identity-p
		   identity (imoogi-clipboard--ensure-identity))))
           (with-current-buffer buffer
             (unwind-protect (imoogi-clipboard--finish-import marker response identity)
               (setq imoogi-clipboard--pending
                     (delq marker imoogi-clipboard--pending))
               (set-marker marker nil)))
         (imoogi-clipboard--log "discarded stale import completion")
         (set-marker marker nil))))))

(defun imoogi-clipboard--finish-import (marker response identity)
  "Insert links from RESPONSE at MARKER and retain IDENTITY."
  (if (not (equal (alist-get 'status response) "ok"))
      (imoogi-clipboard--log "import failed: %s %s"
                             (alist-get 'code response)
                             (alist-get 'message response))
    (atomic-change-group
      (goto-char marker)
      (dolist (asset (alist-get 'assets response))
        (insert (imoogi-clipboard--asset-link asset) "\n")))
	(when-let* ((transaction-id (alist-get 'transaction_id response)))
	  (push `((transaction_id . ,transaction-id)
		  (transaction_token . ,(alist-get 'transaction_token response))
		  (identity . ,identity)
		  (staged . t)
		  (assets . ,(alist-get 'assets response)))
		imoogi-clipboard--transactions))))

(defun imoogi-clipboard--link-target (target)
  "Encode TARGET for an Org or Markdown link, preserving path separators."
  (if (string-prefix-p "imoogi-asset:" target)
      target
    (mapconcat #'url-hexify-string (split-string target "/" nil) "/")))

(defun imoogi-clipboard--decode-link-target (target)
  "Decode the UTF-8 percent escapes in link TARGET."
  (decode-coding-string (url-unhex-string target) 'utf-8))

(defun imoogi-clipboard--link-label (label)
  "Escape LABEL for Org and Markdown link descriptions."
  (replace-regexp-in-string "[][()\\\\\n\r]" "_" label))

(defun imoogi-clipboard--asset-link (asset)
  "Return an Org or Markdown link for ASSET."
  (let* ((absolute (alist-get 'path asset))
         (target (if buffer-file-name
                     (file-relative-name absolute (file-name-directory buffer-file-name))
                   (concat "imoogi-asset:" (alist-get 'id asset))))
	 (target (imoogi-clipboard--link-target target))
	 (label (imoogi-clipboard--link-label
		 (file-name-nondirectory (or absolute target)))))
    (if (derived-mode-p 'org-mode)
        (format "[[file:%s][%s]]" target label)
      (format "[%s](%s)" label target))))

(defun imoogi-clipboard-preview-asset-map ()
  "Return exact opaque asset mappings for the current preview buffer."
  (let (mapping)
    (dolist (transaction imoogi-clipboard--transactions)
      (when (alist-get 'staged transaction)
        (dolist (asset (alist-get 'assets transaction))
          (push (cons (concat "imoogi-asset:" (alist-get 'id asset))
                      (alist-get 'path asset))
                mapping))))
    mapping))

(defun imoogi-clipboard--lifecycle-request (operation transaction &optional owner)
  "Build lifecycle OPERATION request for TRANSACTION and optional OWNER."
  (imoogi-clipboard--request
   operation
   (append `((identity . ,(alist-get 'identity transaction))
             (transaction_id . ,(alist-get 'transaction_id transaction))
             (transaction_token . ,(alist-get 'transaction_token transaction)))
           (when owner `((owner . ,owner))))))

(defun imoogi-clipboard--prepare-transaction (transaction)
  "Finalize staged TRANSACTION for the current saved document."
  (let ((response
         (imoogi-clipboard--call-sync
          (imoogi-clipboard--lifecycle-request
           "finalize" transaction (imoogi-clipboard--owner)) 10.0)))
    (unless (equal (alist-get 'status response) "ok")
      (error "imoogi clipboard finalize failed: %s"
             (or (alist-get 'message response) "no response")))
    (when (equal (alist-get 'state response) "NEEDS_RECONCILIATION")
      (setq response
            (imoogi-clipboard--call-sync
             (imoogi-clipboard--lifecycle-request "reconcile" transaction) 10.0))
      (unless (equal (alist-get 'status response) "ok")
        (error "imoogi clipboard reconcile failed: %s"
               (or (alist-get 'message response) "no response"))))
    (dolist (asset (alist-get 'assets response))
      (let* ((opaque (concat "imoogi-asset:" (alist-get 'id asset)))
             (relative (file-relative-name
                        (alist-get 'path asset)
                        (file-name-directory buffer-file-name))))
        (save-excursion
          (goto-char (point-min))
          (while (search-forward opaque nil t)
		    (replace-match (imoogi-clipboard--link-target relative) t t)))))
    (append transaction
            `((prepared_assets . ,(alist-get 'assets response))
              (lifecycle_state . ,(alist-get 'state response))))))

(defun imoogi-clipboard--notify-lifecycle (operation transaction)
  "Synchronously send lifecycle OPERATION for TRANSACTION."
  (imoogi-clipboard--call-sync
   (imoogi-clipboard--lifecycle-request operation transaction) 5.0))

(defun imoogi-clipboard--save-around (original &rest args)
  "Coordinate staged assets around ORIGINAL `basic-save-buffer'."
  (let ((staged (seq-filter (lambda (transaction)
                              (alist-get 'staged transaction))
                            imoogi-clipboard--transactions)))
    (if (or (not imoogi-clipboard-mode) (null staged) (null buffer-file-name))
        (apply original args)
      (let ((change-group (prepare-change-group)) prepared saved-result)
        (activate-change-group change-group)
        (condition-case err
            (progn
              (setq prepared (mapcar #'imoogi-clipboard--prepare-transaction staged))
              (setq saved-result (apply original args))
              (accept-change-group change-group)
              (dolist (transaction prepared)
                (if (equal (alist-get 'lifecycle_state transaction) "COMMITTED")
                    (setq imoogi-clipboard--transactions
                          (seq-remove
                           (lambda (candidate)
                             (equal (alist-get 'transaction_id candidate)
                                    (alist-get 'transaction_id transaction)))
                           imoogi-clipboard--transactions))
                  (let ((saved (imoogi-clipboard--notify-lifecycle
                                "document-saved" transaction)))
                  (if (not (equal (alist-get 'status saved) "ok"))
                      (imoogi-clipboard--log
                       "document-saved acknowledgement needs reconciliation: %s"
                       (alist-get 'message saved))
                    (let ((committed (imoogi-clipboard--notify-lifecycle
                                      "commit" transaction)))
                      (if (equal (alist-get 'status committed) "ok")
                          (setq imoogi-clipboard--transactions
                                (seq-remove
                                 (lambda (candidate)
                                   (equal (alist-get 'transaction_id candidate)
                                          (alist-get 'transaction_id transaction)))
                                 imoogi-clipboard--transactions))
                        (imoogi-clipboard--log
                         "commit needs reconciliation: %s"
                         (alist-get 'message committed))))))))
              saved-result)
	  (error
	   (cancel-change-group change-group)
	   (imoogi-clipboard--log
	    "save failed; prepared assets retained for retry: %s"
	    (error-message-string err))
	   (signal (car err) (cdr err))))))))

(defun imoogi-clipboard--asset-paths-for-document (document)
  "Return regular files from standalone asset directory of DOCUMENT."
  (let ((directory (concat (file-name-sans-extension document) ".assets")))
    (when (file-directory-p directory)
      (seq-filter #'file-regular-p
                  (directory-files directory t directory-files-no-dot-files-regexp)))))

(defun imoogi-clipboard--owner-asset-root (owner document)
  "Return OWNER's asset root for DOCUMENT."
  (pcase (alist-get 'kind owner)
    ("project_notes" (expand-file-name "assets" (alist-get 'root owner)))
    ("standalone" (or (alist-get 'root owner)
                      (concat (file-name-sans-extension document) ".assets")))))

(defun imoogi-clipboard--referenced-assets (document owner)
  "Return regular asset files referenced by the current buffer under OWNER."
  (let ((base (file-name-directory document))
        (asset-root (imoogi-clipboard--owner-asset-root owner document))
        paths)
    (when asset-root
      (save-excursion
        (goto-char (point-min))
        (while (re-search-forward
                "\\(?:\\[\\[file:\\([^]]+\\)\\]\\|](\\([^)]*\\))\\)" nil t)
          (let* ((raw (or (match-string-no-properties 1)
                          (match-string-no-properties 2)))
                 (path (and raw (expand-file-name
                                 (imoogi-clipboard--decode-link-target raw) base))))
            (when (and path (file-regular-p path)
                       (file-in-directory-p path asset-root))
              (push path paths)))))
      (delete-dups (nreverse paths)))))

(defun imoogi-clipboard--same-project-owner-p (left right)
  "Return non-nil when LEFT and RIGHT share one project-notes root."
  (and (equal (alist-get 'kind left) "project_notes")
       (equal (alist-get 'kind right) "project_notes")
       (equal (file-truename (alist-get 'root left))
              (file-truename (alist-get 'root right)))))

(defun imoogi-clipboard--relocate-visited-assets (old-document new-document &optional old-owner)
  "Copy managed standalone assets from OLD-DOCUMENT for NEW-DOCUMENT."
  (let* ((old-owner (or old-owner
                        `((kind . "standalone")
                          (document . ,old-document)
                          (root . ,(concat (file-name-sans-extension old-document) ".assets")))))
         (new-owner (imoogi-clipboard--owner)))
  (when (and old-document new-document
             (not (equal (expand-file-name old-document)
                         (expand-file-name new-document)))
             (not (imoogi-clipboard--same-project-owner-p old-owner new-owner)))
    (let ((paths (or (imoogi-clipboard--referenced-assets old-document old-owner)
                     (and (equal (alist-get 'kind old-owner) "standalone")
                          (imoogi-clipboard--asset-paths-for-document old-document)))))
      (when paths
        (let* ((identity (imoogi-clipboard--ensure-identity))
               (response
                (imoogi-clipboard--call-sync
                 (imoogi-clipboard--request
                  "import"
                  `((identity . ,identity) (owner . ,(imoogi-clipboard--owner))
                    (paths . ,(vconcat paths))))
                 15.0)))
          (unless (equal (alist-get 'status response) "ok")
            (error "imoogi clipboard relocation failed: %s"
                   (or (alist-get 'message response) "no response")))
          (cl-mapc
           (lambda (old-path asset)
             (let ((old-relative
                    (imoogi-clipboard--link-target
                     (file-relative-name old-path (file-name-directory old-document))))
                   (new-relative
                    (imoogi-clipboard--link-target
                     (file-relative-name (alist-get 'path asset)
                                         (file-name-directory new-document)))))
               (save-excursion
                 (goto-char (point-min))
                 (while (search-forward old-relative nil t)
                   (replace-match new-relative t t)))))
           paths (alist-get 'assets response))))))))

(defun imoogi-clipboard--set-visited-around (original filename &rest args)
  "Run ORIGINAL `set-visited-file-name' and relocate managed assets."
  (let* ((new (expand-file-name filename))
	 (pending (seq-find
		   (lambda (entry)
		     (equal new (alist-get 'new entry)))
		   imoogi-clipboard--pending-renames))
	 (old (or (alist-get 'old pending) buffer-file-name))
	 (old-owner (or (alist-get 'owner pending)
			(and buffer-file-name (imoogi-clipboard--owner)))))
    (prog1 (apply original filename args)
      (when (and imoogi-clipboard-mode old buffer-file-name)
	(setq imoogi-clipboard--pending-renames
	      (delq pending imoogi-clipboard--pending-renames))
	(cl-incf imoogi-clipboard--generation)
	(imoogi-clipboard--relocate-visited-assets old buffer-file-name old-owner)))))

(defun imoogi-clipboard--same-existing-file-p (left right)
  "Return non-nil when existing paths LEFT and RIGHT name the same file."
  (and left right
       (or (equal (expand-file-name left) (expand-file-name right))
	   (ignore-errors (file-equal-p left right)))))

(defun imoogi-clipboard--owner-for-document (document)
  "Resolve the clipboard owner for existing DOCUMENT without visiting it."
  (with-temp-buffer
    (setq buffer-file-name (expand-file-name document))
    (imoogi-clipboard--owner)))

(defun imoogi-clipboard--relocate-unvisited-document (old new old-owner)
  "Relocate managed links in renamed unvisited document NEW from OLD-OWNER."
  (let ((buffer (find-file-noselect new))
	(completed nil))
    (unwind-protect
	(with-current-buffer buffer
	  (imoogi-clipboard--relocate-visited-assets old new old-owner)
	  (when (buffer-modified-p)
	    (save-buffer))
	  (setq completed t)
	  (imoogi-clipboard--log "managed rename completed: %s -> %s" old new))
      (when (buffer-live-p buffer)
	(unless completed
	  (imoogi-clipboard--log "managed rename requires attention: %s -> %s"
				 old new))
	(kill-buffer buffer)))))

(defun imoogi-clipboard--rename-file-around (original source destination &rest args)
  "Run ORIGINAL and remember a managed rename for editor post-processing.

Dired and Treemacs update visited buffers after `rename-file' returns.  This
advice deliberately leaves that metadata to them and only carries the old
asset owner into the later `set-visited-file-name' call."
  (let* ((old (expand-file-name source))
	 (new (expand-file-name destination))
	 (registered-owner
	  (and (not (get-file-buffer old))
	       (member (downcase (or (file-name-extension old) "")) '("org" "md"))
	       (imoogi-clipboard--owner-for-document old)))
	 (managed
	  (seq-filter
	   (lambda (buffer)
	     (and (buffer-live-p buffer)
		  (buffer-local-value 'imoogi-clipboard-mode buffer)
		  (imoogi-clipboard--same-existing-file-p
		   old (buffer-local-value 'buffer-file-name buffer))))
	   imoogi-clipboard--managed-buffers)))
    (prog1 (apply original source destination args)
      (dolist (buffer managed)
	(with-current-buffer buffer
	  (push `((old . ,old) (new . ,new)
		  (owner . ,(imoogi-clipboard--owner)))
		imoogi-clipboard--pending-renames)
	  ;; Use Emacs' public visited-file API so buffer name, truename and
	  ;; modification state remain coherent even when the caller later looks
	  ;; up the old path (as Dired and Treemacs do).
	  (condition-case err
	      (set-visited-file-name new t)
	    (error
	     (imoogi-clipboard--log "managed rename partially completed: %s"
				    (error-message-string err)))))))
      (when (and (null managed)
		 registered-owner
		 (equal (alist-get 'kind registered-owner) "project_notes"))
	(condition-case err
	    (imoogi-clipboard--relocate-unvisited-document
	     old new registered-owner)
	  (error
	   (imoogi-clipboard--log "managed rename partially completed: %s -> %s: %s"
				  old new (error-message-string err)))))))

(defun imoogi-clipboard--org-refile-around (original &rest args)
  "Run ORIGINAL `org-refile' while retaining the source directory."
  (let ((imoogi-clipboard--refile-source-directory
         (and buffer-file-name (file-name-directory buffer-file-name))))
    (apply original args)))

(defun imoogi-clipboard--after-refile-insert ()
  "Copy regular file links in a refiled subtree into the destination owner."
  (when (and imoogi-clipboard-mode imoogi-clipboard--refile-source-directory
             buffer-file-name)
    (save-restriction
      (org-narrow-to-subtree)
      (goto-char (point-min))
      (while (re-search-forward "\\[\\[file:\\([^]]+\\)\\]" nil t)
        (let* ((raw (match-string-no-properties 1))
               (source (expand-file-name (imoogi-clipboard--decode-link-target raw)
                                         imoogi-clipboard--refile-source-directory)))
          (when (file-regular-p source)
            (let ((response
                   (imoogi-clipboard--call-sync
                    (imoogi-clipboard--request
                     "import"
                     `((identity . ,(imoogi-clipboard--ensure-identity))
                       (owner . ,(imoogi-clipboard--owner))
                       (paths . ,(vector source))))
                    15.0)))
              (when (equal (alist-get 'status response) "ok")
                (let ((relative
                       (imoogi-clipboard--link-target
                        (file-relative-name
                         (alist-get 'path (car (alist-get 'assets response)))
                         (file-name-directory buffer-file-name)))))
                  (replace-match relative t t nil 1))))))))))

(defun imoogi-clipboard--after-kill (&rest _)
  "Capture OS clipboard identity after a local kill export."
  (cl-incf imoogi-clipboard--kill-generation)
  (let ((response (imoogi-clipboard--call-sync
                   (imoogi-clipboard--request
                    "checkpoint"
                    `((kill_generation . ,imoogi-clipboard--kill-generation)
                      (identity . ,(imoogi-clipboard--ensure-identity))
                      (lease . ((pid . ,(emacs-pid))
                                (process_start . ,imoogi-clipboard--process-start-id)))))
                   imoogi-clipboard-inspect-timeout)))
    (setq imoogi-clipboard--kill-checkpoint
          (and (equal (alist-get 'status response) "ok")
               `((clipboard_id . ,(alist-get 'clipboard_id response))
                 (kill_generation . ,imoogi-clipboard--kill-generation)
                 (captured_at . ,(float-time)))))))

(defun imoogi-clipboard--renew-lease ()
  "Renew the current buffer staging lease when integration is active."
  (when imoogi-clipboard-mode
    (imoogi-clipboard--call-sync
     (imoogi-clipboard--request
      "checkpoint"
      `((identity . ,(imoogi-clipboard--ensure-identity))
	(lease . ((pid . ,(emacs-pid))
		  (process_start . ,imoogi-clipboard--process-start-id)))))
     imoogi-clipboard-inspect-timeout)))

(defun imoogi-clipboard--renew-active-leases ()
  "Renew leases for every live clipboard-enabled buffer."
  (dolist (buffer (buffer-list))
    (with-current-buffer buffer
      (when imoogi-clipboard-mode
	(imoogi-clipboard--renew-lease)))))

(defun imoogi-clipboard--ensure-lease-timer ()
  "Start the process-wide staging lease timer once."
  (unless (timerp imoogi-clipboard--lease-timer)
    (setq imoogi-clipboard--lease-timer
	  (run-at-time imoogi-clipboard-lease-renew-interval
		       imoogi-clipboard-lease-renew-interval
		       #'imoogi-clipboard--renew-active-leases))))

(defun imoogi-clipboard--dnd-handler (url action)
  "Import a local file URL from drag-and-drop and return ACTION."
  (when (imoogi-clipboard--supported-buffer-p)
    (let ((path (url-unhex-string (string-remove-prefix "file://" url))))
      (if (file-directory-p path)
          (message "imoogi: 폴더 붙여넣기는 지원하지 않습니다")
        (imoogi-clipboard--start-import "import" nil (list path))))
    action))

(defun imoogi-clipboard--dispose-buffer ()
  "Abort disposable staging transactions owned by the current buffer."
  (dolist (transaction imoogi-clipboard--transactions)
    (when (alist-get 'staged transaction)
      (imoogi-clipboard--notify-lifecycle "abort" transaction)))
  (setq imoogi-clipboard--transactions nil
	imoogi-clipboard--managed-buffers
	(delq (current-buffer) imoogi-clipboard--managed-buffers)))

(defvar imoogi-clipboard-mode-map
  (let ((map (make-sparse-keymap)))
    (define-key map [remap yank] #'imoogi-clipboard-yank) map))

(define-minor-mode imoogi-clipboard-mode
  "Make normal paste asset-aware in Org and Markdown buffers."
  :lighter " Clip" :keymap imoogi-clipboard-mode-map
  (if imoogi-clipboard-mode
      (progn (imoogi-clipboard--ensure-identity)
	     (cl-pushnew (current-buffer) imoogi-clipboard--managed-buffers)
	     (imoogi-clipboard--ensure-lease-timer)
	     (imoogi-clipboard--renew-lease)
	     (add-hook 'kill-buffer-hook #'imoogi-clipboard--dispose-buffer nil t))
    (remove-hook 'kill-buffer-hook #'imoogi-clipboard--dispose-buffer t)
	(imoogi-clipboard--dispose-buffer)
    (mapc (lambda (marker) (set-marker marker nil)) imoogi-clipboard--pending)
    (setq imoogi-clipboard--pending nil)))

(defun imoogi-clipboard--maybe-enable ()
  "Enable clipboard integration in supported major modes."
  (when (derived-mode-p 'org-mode 'markdown-mode 'gfm-mode)
    (imoogi-clipboard-mode 1)))

(add-hook 'org-mode-hook #'imoogi-clipboard--maybe-enable)
(add-hook 'markdown-mode-hook #'imoogi-clipboard--maybe-enable)
(advice-add 'kill-new :after #'imoogi-clipboard--after-kill)
(advice-add 'kill-append :after #'imoogi-clipboard--after-kill)
(advice-add 'basic-save-buffer :around #'imoogi-clipboard--save-around)
(advice-add 'set-visited-file-name :around #'imoogi-clipboard--set-visited-around)
(advice-add 'rename-file :around #'imoogi-clipboard--rename-file-around)
(with-eval-after-load 'org
  (advice-add 'org-refile :around #'imoogi-clipboard--org-refile-around)
  (add-hook 'org-after-refile-insert-hook #'imoogi-clipboard--after-refile-insert))
(add-to-list 'dnd-protocol-alist '("^file:" . imoogi-clipboard--dnd-handler))

(provide 'imoogi-clipboard)
;;; 28-clipboard.el ends here
