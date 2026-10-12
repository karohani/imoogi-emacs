;;; org-preview-test.el --- tests for live Org preview client -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'org)
(require 'markdown-mode)
(require 'imoogi-org-preview)

(ert-deftest imoogi-org-preview-payload-uses-unsaved-buffer-text ()
  (with-temp-buffer
    (org-mode)
    (insert "* Draft\nUnsaved text")
    (goto-char (point-max))
    (setq-local imoogi-org-preview--session-id "session-a"
                imoogi-org-preview--buffer-id "buffer-a"
                imoogi-org-preview--revision 7)
    (let ((payload (imoogi-org-preview--build-update-payload)))
      (should (equal (alist-get 'text payload) "* Draft\nUnsaved text"))
      (should (= (alist-get 'revision payload) 7))
      (should (equal (alist-get 'version payload) "org-preview/v1"))
      (should (equal (alist-get 'origin payload) "emacs"))
      (should (equal (alist-get 'path payload) ""))
      (should (equal (alist-get 'syntax payload) "org"))
      (should (listp (alist-get 'allowed_roots payload)))
      (should (= (alist-get 'cursor_byte payload)
                 (string-bytes (encode-coding-string "* Draft\nUnsaved text" 'utf-8)))))))

(ert-deftest imoogi-org-preview-markdown-payload-identifies-syntax ()
  (with-temp-buffer
    (markdown-mode)
    (insert "# Draft\n\nUnsaved **Markdown** text")
    (setq-local imoogi-org-preview--session-id "session-md"
                imoogi-org-preview--buffer-id "buffer-md"
                imoogi-org-preview--revision 3)
    (let ((payload (imoogi-org-preview--build-update-payload)))
      (should (equal (alist-get 'syntax payload) "markdown"))
      (should (equal (alist-get 'text payload) (buffer-string))))))

(ert-deftest imoogi-org-preview-enables-in-markdown-mode ()
  (with-temp-buffer
    (markdown-mode)
    (let ((imoogi-org-preview-command '("/no/such/imoogi-org-preview")))
      (imoogi-org-preview-mode 1)
      (should imoogi-org-preview-mode)
      (should (eq imoogi-org-preview--status 'failed)))))

(ert-deftest imoogi-org-preview-payload-uses-utf8-byte-cursor ()
  (with-temp-buffer
    (org-mode)
    (insert "* 제목\n본문")
    (goto-char (point-max))
    (setq-local imoogi-org-preview--session-id "session-a"
                imoogi-org-preview--buffer-id "buffer-a"
                imoogi-org-preview--revision 1)
    (let ((payload (imoogi-org-preview--build-update-payload)))
      (should (= (alist-get 'cursor_byte payload)
                 (string-bytes (encode-coding-string (buffer-string) 'utf-8))))
      (should (> (alist-get 'cursor_byte payload)
                 (length (buffer-string)))))))

(ert-deftest imoogi-org-preview-update-debounce-replaces-prior-timer ()
  (with-temp-buffer
    (org-mode)
    (setq-local imoogi-org-preview-debounce-seconds 30)
    (imoogi-org-preview--schedule-update)
    (let ((first-timer imoogi-org-preview--update-timer))
      (should (timerp first-timer))
      (imoogi-org-preview--schedule-update)
      (should (timerp imoogi-org-preview--update-timer))
      (should-not (eq first-timer imoogi-org-preview--update-timer))
      (cancel-timer imoogi-org-preview--update-timer))))

(ert-deftest imoogi-org-preview-current-element-id-is-source-range-based ()
  (with-temp-buffer
    (org-mode)
    (insert "* Heading\nBody\n")
    (goto-char (point-min))
    (let ((element-id (imoogi-org-preview--current-element-id)))
      (should (string-match-p "\\`org-headline-0-[0-9]+\\'" element-id)))))

(ert-deftest imoogi-org-preview-navigation-payload-includes-cursor-byte ()
  (with-temp-buffer
    (org-mode)
    (insert "* 제목\n본문\n")
    (goto-char (point-max))
    (setq-local imoogi-org-preview-mode t
                imoogi-org-preview--session-id "session-a"
                imoogi-org-preview--buffer-id "buffer-a"
                imoogi-org-preview--revision 2)
    (let ((imoogi-org-preview-navigation-sync t)
          (imoogi-org-preview-port 32123)
          (imoogi-org-preview-token "token-a")
          captured)
      (cl-letf (((symbol-function 'imoogi-org-preview--post-json)
                 (lambda (_path payload _callback)
                   (setq captured payload))))
        (imoogi-org-preview--send-navigation (current-buffer)))
      (should (equal (alist-get 'session_id captured) "session-a"))
      (should (= (alist-get 'cursor_byte captured)
                 (string-bytes (encode-coding-string (buffer-string) 'utf-8))))
      (should (alist-get 'range captured)))))

(ert-deftest imoogi-org-preview-browser-navigation-moves-point-and-suppresses-echo ()
  (with-temp-buffer
    (org-mode)
    (insert "* Heading\nBody\n")
    (goto-char (point-min))
    (setq-local imoogi-org-preview--buffer-id "buffer-a"
                imoogi-org-preview--sent-events nil
                imoogi-org-preview--suppress-navigation nil)
    (cl-letf (((symbol-function 'run-at-time)
               (lambda (&rest _args) 'fake-timer)))
      (imoogi-org-preview--handle-event
       '((protocol_version . 1)
         (buffer_id . "buffer-a")
         (origin . "browser")
         (event_id . "browser-event")
         (range . ((start . 10) (end . 14))))))
    (should (= (point) 11))
    (should imoogi-org-preview--suppress-navigation)))

(ert-deftest imoogi-org-preview-ignores-reflected-event-id ()
  (with-temp-buffer
    (org-mode)
    (insert "* Heading\nBody\n")
    (goto-char (point-min))
    (setq-local imoogi-org-preview--buffer-id "buffer-a"
                imoogi-org-preview--sent-events '("same-event"))
    (imoogi-org-preview--handle-event
     '((protocol_version . 1)
       (buffer_id . "buffer-a")
       (origin . "browser")
       (event_id . "same-event")
       (range . ((start . 10) (end . 14)))))
    (should (= (point) (point-min)))))

(ert-deftest imoogi-org-preview-ignores-wrong-buffer-event ()
  (with-temp-buffer
    (org-mode)
    (insert "* Heading\nBody\n")
    (goto-char (point-min))
    (setq-local imoogi-org-preview--buffer-id "buffer-a")
    (imoogi-org-preview--handle-event
     '((protocol_version . 1)
       (buffer_id . "buffer-b")
       (origin . "browser")
       (event_id . "browser-event")
       (range . ((start . 10) (end . 14)))))
    (should (= (point) (point-min)))))

(ert-deftest imoogi-org-preview-missing-server-command-degrades-preview-only ()
  (with-temp-buffer
    (org-mode)
    (let ((imoogi-org-preview-command '("/no/such/imoogi-org-preview")))
      (insert "* Editable\n")
      (imoogi-org-preview-mode 1)
      (should (eq imoogi-org-preview--status 'failed))
      (insert "still editable")
      (should (string-match-p "still editable" (buffer-string))))))

(ert-deftest imoogi-org-preview-disable-removes-buffer-local-preview-state ()
  (with-temp-buffer
    (org-mode)
    (insert "* Keep me\n")
    (let ((before (buffer-string)))
      (setq-local imoogi-org-preview-mode t
                  imoogi-org-preview--update-timer (run-at-time 60 nil #'ignore)
                  imoogi-org-preview--nav-timer (run-at-time 60 nil #'ignore)
                  imoogi-org-preview--reconnect-timer (run-at-time 60 nil #'ignore)
                  imoogi-org-preview--suppress-navigation t)
      (imoogi-org-preview--disable)
      (should (equal before (buffer-string)))
      (should-not (bound-and-true-p imoogi-org-preview--update-timer))
      (should-not (bound-and-true-p imoogi-org-preview--nav-timer))
      (should-not (bound-and-true-p imoogi-org-preview--reconnect-timer))
      (should-not imoogi-org-preview--suppress-navigation)
      (should (eq imoogi-org-preview--status 'disconnected)))))

(ert-deftest imoogi-org-preview-open-url-includes-browser-session-contract ()
  (with-temp-buffer
    (org-mode)
    (setq-local imoogi-org-preview-mode t
                imoogi-org-preview--session-id "session-a"
		imoogi-org-preview--buffer-id "buffer-a"
		imoogi-org-preview--browser-view "view-a")
    (let ((imoogi-org-preview-port 32123)
          (imoogi-org-preview-token "token-a")
          opened-url)
      (cl-letf (((symbol-function 'browse-url)
                 (lambda (url &rest _args)
                   (setq opened-url url))))
        (imoogi-org-preview-open))
      (should (string-match-p "/preview\\?" opened-url))
      (should (string-match-p "session=session-a" opened-url))
      (should (string-match-p "buffer=buffer-a" opened-url))
	(should (string-match-p "generation=1" opened-url))
	(should (string-match-p "view=view-a" opened-url))
	(should-not (string-match-p "token-a" opened-url)))))

;;; org-preview-test.el ends here

(ert-deftest imoogi-org-preview-coalesces-edits-during-request ()
  (with-temp-buffer
    (org-mode)
    (insert "* First")
    (let ((imoogi-org-preview-port 12345)
          (imoogi-org-preview-token "test")
          (imoogi-org-preview-mode t)
          payloads)
      (cl-letf (((symbol-function 'imoogi-org-preview--post-json)
                 (lambda (_path payload _callback) (push payload payloads))))
        (imoogi-org-preview--send-update (current-buffer))
        (insert " latest")
        (imoogi-org-preview--schedule-update)
        (imoogi-org-preview--send-update (current-buffer))
        (should (= (length payloads) 1))
        (should imoogi-org-preview--dirty)
        (let ((source (current-buffer)))
          (with-temp-buffer
            (insert "HTTP/1.1 200 OK\r\n\r\n")
            (imoogi-org-preview--update-callback nil source)))
        (should (timerp imoogi-org-preview--update-timer))
        (cancel-timer imoogi-org-preview--update-timer)
        (imoogi-org-preview--send-update (current-buffer))
        (should (= (length payloads) 2))
        (should (equal (alist-get 'text (car payloads)) "* First latest"))))))

(ert-deftest imoogi-org-preview-highlight-does-not-parse-org-elements ()
  (with-temp-buffer
    (org-mode)
    (insert "* 제목\n문단")
    (let ((imoogi-org-preview-mode t)
          (imoogi-org-preview-navigation-sync nil)
          (imoogi-org-preview-port 32123)
          (imoogi-org-preview-token "test")
          payload)
      (cl-letf (((symbol-function 'org-element-at-point)
                 (lambda (&rest _) (ert-fail "Cursor updates must not parse Org")))
                ((symbol-function 'imoogi-org-preview--post-json)
                 (lambda (_path data _callback) (setq payload data))))
        (imoogi-org-preview--send-navigation (current-buffer)))
      (should (integerp (alist-get 'cursor_byte payload)))
      (should-not (alist-get 'element_id payload))
      (should-not (alist-get 'range payload)))))

(defun imoogi-org-preview-test--open-event (open)
  "Return a server open EVENT carrying OPEN for the current preview buffer."
  `((version . "org-preview/v1") (buffer_id . "buffer-open") (origin . "browser")
    (event_id . ,(format "open-%s" (random))) (type . "open") (open . ,open)))

(defmacro imoogi-org-preview-test--with-open-buffer (&rest body)
  "Run BODY in an Org preview buffer visiting a file in a temp directory."
  (declare (indent 0))
  `(let* ((dir (file-name-as-directory (make-temp-file "imoogi-open-" t)))
          (doc (expand-file-name "doc.org" dir))
          opened messages)
     (unwind-protect
         (with-current-buffer (find-file-noselect doc)
           (setq-local imoogi-org-preview--buffer-id "buffer-open")
           (cl-letf (((symbol-function 'project-current) (lambda (&rest _) nil))
                     ((symbol-function 'find-file-other-window)
                      (lambda (file &rest _) (push file opened)))
                     ((symbol-function 'message)
                      (lambda (fmt &rest args) (push (apply #'format fmt args) messages))))
             ,@body)
           (set-buffer-modified-p nil)
           (kill-buffer))
       (delete-directory dir t))))

(ert-deftest imoogi-org-preview-open-file-inside-roots ()
  (imoogi-org-preview-test--with-open-buffer
    (let ((other (expand-file-name "other.org" dir)))
      (write-region "* x\n" nil other nil 'silent)
      (let ((preview (current-buffer)))
        (cl-letf (((symbol-function 'find-file-other-window)
                   (lambda (file &rest _)
                     (push file opened)
                     (set-buffer (get-buffer-create " *imoogi-open-other*")))))
          (imoogi-org-preview--handle-event
           (imoogi-org-preview-test--open-event `((kind . "file") (path . ,other)))))
        (should (eq (current-buffer) preview)))
      (should (equal opened (list other))))))

(ert-deftest imoogi-org-preview-open-refuses-missing-or-outside-files ()
  (imoogi-org-preview-test--with-open-buffer
    (let ((outside (make-temp-file "imoogi-outside-" nil ".org")))
      (unwind-protect
          (dolist (path (list (expand-file-name "missing.org" dir) outside))
            (imoogi-org-preview--handle-event
             (imoogi-org-preview-test--open-event `((kind . "file") (path . ,path)))))
        (delete-file outside))
      (should-not opened)
      (should-not (file-exists-p (expand-file-name "missing.org" dir)))
      (should (= (length messages) 2)))))

(ert-deftest imoogi-org-preview-open-id-and-unavailable ()
  (imoogi-org-preview-test--with-open-buffer
    (let ((target (generate-new-buffer "*imoogi-open-target*"))
          (preview (current-buffer))
          popped)
      (unwind-protect
          (progn
            ;; Narrowed away from the target heading: the jump must still land.
            (with-current-buffer target (org-mode) (insert "* a\n* b\n") (narrow-to-region 1 4))
            (cl-letf (((symbol-function 'org-id-find)
                       (lambda (id &optional _)
                         (when (equal id "KNOWN")
                           (with-current-buffer target (copy-marker 5)))))
                      ((symbol-function 'pop-to-buffer)
                       (lambda (buffer &rest _) (setq popped buffer) (set-buffer buffer))))
              (imoogi-org-preview--handle-event
               (imoogi-org-preview-test--open-event '((kind . "id") (id . "KNOWN"))))
              (should (eq popped target))
              (should (= (with-current-buffer target (point)) 5))
              ;; SSE processing continues in the preview buffer.
              (should (eq (current-buffer) preview))
              (setq popped nil)
              (with-current-buffer preview
                (imoogi-org-preview--handle-event
                 (imoogi-org-preview-test--open-event '((kind . "id") (id . "MISSING"))))
                (imoogi-org-preview--handle-event
                 (imoogi-org-preview-test--open-event '((kind . "unavailable") (target . "x.org")))))
              (should-not popped)))
        (kill-buffer target)
        (set-buffer preview))
      (should (= (length messages) 2)))))

(ert-deftest imoogi-org-preview-allows-project-notes-assets-from-artifacts ()
  "Pasted images in a project-notes artifact live in the sibling assets dir."
  (let* ((notes (file-name-as-directory (make-temp-file "imoogi-notes-" t)))
         (assets (expand-file-name "assets/" notes))
         (document (expand-file-name "artifacts/doc.org" notes))
         buffer)
    (unwind-protect
        (progn
          (make-directory assets t)
          (make-directory (file-name-directory document) t)
          (cl-letf (((symbol-function 'project-current) (lambda (&rest _) nil))
                    ((symbol-function 'imoogi-project-notes--read-registry)
                     (lambda () nil)))
            (setq buffer (find-file-noselect document))
            (with-current-buffer buffer
              (setq-local imoogi-project-notes-directory-local notes)
              (should (seq-some (lambda (root) (file-in-directory-p assets root))
                                (imoogi-org-preview--allowed-roots))))))
      (when (buffer-live-p buffer)
        (with-current-buffer buffer (set-buffer-modified-p nil))
        (kill-buffer buffer))
      (delete-directory notes t))))
