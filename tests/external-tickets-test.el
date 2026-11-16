;;; external-tickets-test.el --- external task metadata -*- lexical-binding: t; -*-
(require 'ert)
(require 'org)
(require 'json)
(require 'cl-lib)

(defun imoogi-external-test--receive (payload)
  (imoogi-emacsctl-receive-json
   (json-serialize `((version . "1") (type . "task-create")
                     (timestamp . "2026-11-16T00:00:00Z") (payload . ,payload)))))

(defmacro imoogi-external-test--file (&rest body)
  (declare (indent 0))
  `(let* ((directory (make-temp-file "imoogi-external-" t))
          (file (expand-file-name "tasks.org" directory)))
     (unwind-protect
         (progn (with-temp-file file (insert "* Inbox\n** TODO Existing\n:PROPERTIES:\n:ID: original\n:END:\n* Other\n")) ,@body)
       (when-let* ((buffer (find-buffer-visiting file)))
         (with-current-buffer buffer (set-buffer-modified-p nil)) (kill-buffer buffer))
       (delete-directory directory t))))

(ert-deftest imoogi-external-properties-use-native-completion ()
  (with-temp-buffer
    (org-mode) (insert "* Task\n") (goto-char (point-min))
    (dolist (key '("EXTERNAL_SYSTEM" "EXTERNAL_ID" "EXTERNAL_URL"))
      (should (member key (org-buffer-property-keys nil t t))))
    (let ((imoogi-org-external-systems '("jira" "internal")))
      (should (equal (org-property-get-allowed-values nil "EXTERNAL_SYSTEM") '("jira" "internal")))
      (cl-letf (((symbol-function 'org-completing-read)
                 (lambda (_prompt collection _predicate require-match &rest _)
                   (should (assoc "internal" collection))
                   (should-not require-match)
                   "custom")))
        (org-set-property "EXTERNAL_SYSTEM" (org-read-property-value "EXTERNAL_SYSTEM")))
      (should (equal (org-entry-get nil "EXTERNAL_SYSTEM") "custom")))))

(ert-deftest imoogi-external-task-persists-properties-and-unique-id ()
  (imoogi-external-test--file
    (should (equal "ok" (imoogi-external-test--receive
                         `((path . ,file) (title . "Fix login") (externalSystem . "youtrack")
                           (externalId . "PROJ-123") (externalUrl . "https://example.com/PROJ-123")))))
    (with-temp-buffer
      (insert-file-contents file) (org-mode)
      (goto-char (point-min)) (search-forward "** TODO Fix login")
      (should (equal (org-entry-get nil "EXTERNAL_SYSTEM") "youtrack"))
      (should (equal (org-entry-get nil "EXTERNAL_ID") "PROJ-123"))
      (should (equal (org-entry-get nil "EXTERNAL_URL") "https://example.com/PROJ-123"))
      (should (org-entry-get nil "ID"))
      (should-not (equal (org-entry-get nil "ID") "original"))
      (org-up-heading-safe) (should (equal (org-get-heading t t t t) "Inbox")))))

(ert-deftest imoogi-external-task-default-agenda-and-missing-inbox ()
  (imoogi-external-test--file
    (with-temp-file file (insert "* Other\n"))
    (rename-file file (expand-file-name "agenda.org" directory))
    (setq file (expand-file-name "agenda.org" directory))
    (let ((org-directory directory))
      (should (equal "ok" (imoogi-external-test--receive '((title . "Read notes"))))))
    (with-temp-buffer (insert-file-contents file)
      (should (search-forward "* Inbox\n** TODO Read notes\n" nil t)))))

(ert-deftest imoogi-external-task-rejects-invalid-fields-without-write ()
  (imoogi-external-test--file
    (let ((before (with-temp-buffer (insert-file-contents file) (buffer-string))))
      (dolist (fields '(((title . " ")) ((title . "a\n* injected")) ((title . "a\x00b"))
                        ((title . "a") (externalSystem . "jira"))
                        ((title . "a") (externalId . "A-1"))
                        ((title . "a") (externalUrl . "https://example.com"))
                        ((title . "a") (externalSystem . "jira") (externalId . "A-1\n:ID: bad"))
                        ((title . "a") (externalSystem . "jira") (externalId . "A-1") (externalUrl . "file:///tmp/a"))
                        ((title . "a") (externalSystem . "jira") (externalId . "A-1") (externalUrl . "https:///a"))))
        (should (equal "error:bad-field" (imoogi-external-test--receive (cons `(path . ,file) fields)))))
      (should (equal before (with-temp-buffer (insert-file-contents file) (buffer-string)))))))

(ert-deftest imoogi-external-task-preserves-dirty-buffer-and-disk ()
  (imoogi-external-test--file
    (let ((before (with-temp-buffer (insert-file-contents file) (buffer-string))))
      (with-current-buffer (find-file-noselect file) (goto-char (point-max)) (insert "Unsaved\n")
        (let ((dirty (buffer-string)))
          (should (equal "error:dirty-buffer" (imoogi-external-test--receive `((path . ,file) (title . "New")))))
          (should (equal dirty (buffer-string)))
          (should (buffer-modified-p))))
      (should (equal before (with-temp-buffer (insert-file-contents file) (buffer-string)))))))

(ert-deftest imoogi-external-task-rejects-stale-buffer ()
  (imoogi-external-test--file
    (find-file-noselect file)
    (with-temp-file file (insert "* Outside update\n"))
    (set-file-times file (time-add (current-time) 5))
    (should (equal "error:stale-buffer" (imoogi-external-test--receive `((path . ,file) (title . "New")))))
    (with-temp-buffer (insert-file-contents file) (should (equal (buffer-string) "* Outside update\n")))))

(ert-deftest imoogi-external-task-rejects-read-only-and-locked ()
  (imoogi-external-test--file
    (with-current-buffer (find-file-noselect file)
      (setq buffer-read-only t)
      (should (equal "error:read-only" (imoogi-external-test--receive `((path . ,file) (title . "New"))))))
    (with-current-buffer (find-buffer-visiting file) (setq buffer-read-only nil))
    (cl-letf (((symbol-function 'file-locked-p) (lambda (_) "someone")))
      (should (equal "error:locked" (imoogi-external-test--receive `((path . ,file) (title . "New"))))))))

(ert-deftest imoogi-external-task-rejects-missing-remote-and-non-org-paths ()
  (imoogi-external-test--file
    (dolist (path (list (expand-file-name "absent.org" directory) "/ssh:host:/tmp/a.org"))
      (should (equal "error:bad-path" (imoogi-external-test--receive `((path . ,path) (title . "New"))))))
    (let ((text (expand-file-name "notes.txt" directory)))
      (with-temp-file text (insert "hello"))
      (should (equal "error:bad-path" (imoogi-external-test--receive `((path . ,text) (title . "New"))))))))

(ert-deftest imoogi-external-task-creates-default-agenda-in-existing-directory ()
  (imoogi-external-test--file
    (setq file (expand-file-name "agenda.org" directory))
    (let ((org-directory directory))
      (should-not (file-exists-p file))
      (should (equal "ok" (imoogi-external-test--receive '((title . "First task"))))))
    (with-temp-buffer (insert-file-contents file)
      (should (search-forward "* Inbox\n** TODO First task\n" nil t)))))

(ert-deftest imoogi-external-task-save-failure-rolls-back-existing-buffer ()
  (imoogi-external-test--file
    (with-current-buffer (find-file-noselect file)
      (let ((before (buffer-string)))
        (cl-letf (((symbol-function 'save-buffer) (lambda (&rest _) (error "Disk full"))))
          (should (equal "error:handler" (imoogi-external-test--receive `((path . ,file) (title . "New"))))))
        (should (equal (buffer-string) before))
        (should-not (buffer-modified-p))
        (should (equal before (with-temp-buffer (insert-file-contents file) (buffer-string))))))))

(defun imoogi-external-test--rejected-before-dispatch (file payload)
  (let ((before (with-temp-buffer (insert-file-contents file) (buffer-string)))
        dispatched)
    (cl-letf (((symbol-function 'imoogi-emacsctl-dispatch)
               (lambda (_) (setq dispatched t))))
      (should (equal "error:bad-field" (imoogi-external-test--receive payload))))
    (should-not dispatched)
    (should (equal before (with-temp-buffer (insert-file-contents file) (buffer-string))))))

(ert-deftest imoogi-external-review-f1-rejects-unicode-controls-before-dispatch ()
  (imoogi-external-test--file
    (dolist (key '(title externalSystem externalId externalUrl))
      (dolist (char '(127 128 133 159 8232 8233))
        (let ((payload `((path . ,file) (title . "Task") (externalSystem . "jira")
                         (externalId . "A-1") (externalUrl . "https://example.com/a"))))
          (setcdr (assq key payload) (concat (cdr (assq key payload)) (string char)))
          (imoogi-external-test--rejected-before-dispatch file payload))))
    (dolist (char '(160 8199 8239 12288))
      (imoogi-external-test--rejected-before-dispatch
       file `((path . ,file) (title . ,(string char)))))))

(ert-deftest imoogi-external-review-f2-rejects-malformed-urls-before-dispatch ()
  (imoogi-external-test--file
    (dolist (url '("https://example.com:bad" "https://[bad" "https://example.com/%zz"
                   "https://example.com/%" "https://example.com/a#%zz" "https://example.com/?q=%zz"))
      (imoogi-external-test--rejected-before-dispatch
       file `((path . ,file) (title . "Task") (externalSystem . "jira")
              (externalId . "A-1") (externalUrl . ,url))))))

(ert-deftest imoogi-external-review-f2-keeps-valid-urls ()
  (imoogi-external-test--file
    (dolist (url '("https://example.com:443/a%20b?q=x%2Fy%25%20#part" "http://[::1]:8080/a?q=1"))
      (should (equal "ok" (imoogi-external-test--receive
                           `((path . ,file) (title . "Task") (externalSystem . "jira")
                             (externalId . "A-1") (externalUrl . ,url))))))))

(ert-deftest imoogi-external-review-f3-uses-target-unfinished-keyword ()
  (imoogi-external-test--file
    (with-temp-file file (insert "#+TODO: OPEN | DONE\n* Inbox\n"))
    (should (equal "ok" (imoogi-external-test--receive `((path . ,file) (title . "Task")))))
    (with-temp-buffer
      (insert-file-contents file) (org-mode) (goto-char (point-min))
      (search-forward "Task")
      (should (equal (org-get-todo-state) "OPEN")))))

(ert-deftest imoogi-external-review-f3-rejects-no-unfinished-keyword ()
  (imoogi-external-test--file
    (with-temp-file file (insert "#+TODO: | DONE\n* Inbox\n"))
    (let ((before (with-temp-buffer (insert-file-contents file) (buffer-string))))
      (should (equal "error:no-todo-keyword"
                     (imoogi-external-test--receive `((path . ,file) (title . "Task")))))
      (should (equal before (with-temp-buffer (insert-file-contents file) (buffer-string)))))))

(ert-deftest imoogi-external-review-url-scheme-is-independent-of-buffer-case-fold ()
  (imoogi-external-test--file
    (let ((case-fold-search nil))
      (should (equal "ok" (imoogi-external-test--receive
                           `((path . ,file) (title . "Task") (externalSystem . "jira")
                             (externalId . "A-1") (externalUrl . "HTTPS://example.com/a"))))))))

(ert-deftest imoogi-external-claude-save-then-error-keeps-persisted-task ()
  (imoogi-external-test--file
    (let ((save (symbol-function 'save-buffer)))
      (with-current-buffer (find-file-noselect file)
        (cl-letf (((symbol-function 'save-buffer)
                   (lambda (&rest args) (apply save args) (error "Post-save error"))))
          (should (equal "ok" (imoogi-external-test--receive `((path . ,file) (title . "Committed"))))))
        (should (equal (buffer-string)
                       (with-temp-buffer (insert-file-contents file) (buffer-string))))
        (should-not (buffer-modified-p))))))

(ert-deftest imoogi-external-claude-notification-error-does-not-reverse-success ()
  (imoogi-external-test--file
    (cl-letf (((symbol-function 'imoogi-emacsctl--notify) (lambda (_) (error "Display unavailable"))))
      (should (equal "ok" (imoogi-external-test--receive `((path . ,file) (title . "Committed"))))))
    (with-temp-buffer (insert-file-contents file)
      (should (search-forward "Committed" nil t)))))

(ert-deftest imoogi-external-claude-refreshes-saved-todo-declaration ()
  (imoogi-external-test--file
    (with-current-buffer (find-file-noselect file)
      (goto-char (point-min)) (insert "#+TODO: OPEN | DONE\n") (save-buffer)
      (should (equal (car org-not-done-keywords) "TODO"))
      (should (equal "ok" (imoogi-external-test--receive `((path . ,file) (title . "Fresh keyword")))))
      (goto-char (point-min)) (search-forward "Fresh keyword")
      (should (equal "OPEN" (org-get-todo-state))))
    (with-temp-buffer (insert-file-contents file) (org-mode) (goto-char (point-min))
      (search-forward "Fresh keyword")
      (should (equal "OPEN" (org-get-todo-state))))))

(ert-deftest imoogi-external-claude-ignores-local-keywords-open-or-closed ()
  (imoogi-external-test--file
    (let ((org-todo-keywords '((sequence "TODO" "|" "DONE"))))
      (with-temp-file file
        (insert "# -*- org-todo-keywords: ((sequence \"LOCAL\" \"|\" \"DONE\")); -*-\n* Inbox\n"))
      (should (equal "ok" (imoogi-external-test--receive `((path . ,file) (title . "Closed case")))))
      (with-current-buffer (find-file-noselect file)
        (setq-local org-todo-keywords '((sequence "LOCAL" "|" "DONE")))
        (org-set-regexps-and-options)
        (should (equal (car org-not-done-keywords) "TODO"))
        (should (equal "ok" (imoogi-external-test--receive `((path . ,file) (title . "Open case"))))))
      (with-temp-buffer (insert-file-contents file)
        (should (search-forward "** TODO Closed case" nil t))
        (should (search-forward "** TODO Open case" nil t))))))

(ert-deftest imoogi-external-claude-registers-id-outside-agenda-after-save ()
  (require 'org-id)
  (imoogi-external-test--file
    (let ((org-id-locations (make-hash-table :test 'equal))
          (org-id-files nil) (org-id-extra-files nil) (org-agenda-files nil)
          (org-id-track-globally t)
          (org-id-locations-file (expand-file-name "ids" directory)) id)
      (should (equal "ok" (imoogi-external-test--receive `((path . ,file) (title . "Indexed task")))))
      (with-temp-buffer (insert-file-contents file) (org-mode) (goto-char (point-min))
        (search-forward "Indexed task") (setq id (org-entry-get nil "ID")))
      (with-temp-buffer
        (let ((found (org-id-find id)))
          (should found)
          (should (equal (file-truename (car found)) (file-truename file))))))))

(ert-deftest imoogi-external-claude-dirty-buffer-keeps-own-lock ()
  (imoogi-external-test--file
    (let ((create-lockfiles t))
      (with-current-buffer (find-file-noselect file)
        (goto-char (point-max)) (insert "Unsaved")
        (should (eq (file-locked-p file) t))
        (should (equal "error:dirty-buffer" (imoogi-external-test--receive `((path . ,file) (title . "New")))))
        (should (eq (file-locked-p file) t))))))

(ert-deftest imoogi-external-claude-save-error-releases-only-new-edit-lock ()
  (imoogi-external-test--file
    (let ((create-lockfiles t))
      (with-current-buffer (find-file-noselect file)
        (should-not (file-locked-p file))
        (cl-letf (((symbol-function 'save-buffer) (lambda (&rest _) (error "Disk unavailable"))))
          (should (equal "error:handler" (imoogi-external-test--receive `((path . ,file) (title . "New"))))))
        (should-not (buffer-modified-p))
        (should-not (file-locked-p file))))))

(ert-deftest imoogi-external-claude-interaction-is-already-inhibited ()
  (imoogi-external-test--file
    (cl-letf (((symbol-function 'save-buffer)
               (lambda (&rest _) (read-from-minibuffer "Must not prompt: "))))
      (should (equal "error:handler" (imoogi-external-test--receive `((path . ,file) (title . "New"))))))))

(ert-deftest imoogi-external-claude-index-error-does-not-reverse-success ()
  (imoogi-external-test--file
    (cl-letf (((symbol-function 'org-id-add-location) (lambda (&rest _) (error "Index unavailable"))))
      (should (equal "ok" (imoogi-external-test--receive
                           `((path . ,file) (title . "Committed without index"))))))
    (with-temp-buffer (insert-file-contents file)
      (should (search-forward "Committed without index" nil t)))))

(ert-deftest imoogi-external-claude-keyword-calculation-uses-explicit-input ()
  (should (equal "NEXT" (imoogi-emacsctl--task-keyword
                         "* Inbox\n" '((sequence "NEXT" "|" "DONE"))))))

(ert-deftest imoogi-external-claude-calculation-does-not-run-mode-hooks ()
  (let* ((called nil)
         (hook (lambda () (setq called t)))
         (change-major-mode-hook (list hook))
         (after-change-major-mode-hook (list hook))
         (org-mode-hook (list hook)))
    (should (equal "NEXT" (imoogi-emacsctl--task-keyword
                           "* Inbox\n" '((sequence "NEXT" "|" "DONE")))))
    (should-not called)))

(ert-deftest imoogi-external-final-legacy-writer-cannot-bypass-save ()
  (imoogi-external-test--file
    (with-current-buffer (find-file-noselect file)
      (let ((called nil))
        (setq-local local-write-file-hooks (list (lambda () (setq called t) t)))
        (should (equal "ok" (imoogi-external-test--receive
                             `((path . ,file) (title . "Persist despite legacy hook")))))
        (should-not called)
        (should-not (buffer-modified-p))
        (should (equal (buffer-string)
                       (with-temp-buffer (insert-file-contents file) (buffer-string))))))))

(ert-deftest imoogi-external-final-keyword-honors-setupfile-dependency ()
  (let (reads)
    (cl-letf (((symbol-function 'org-file-contents)
               (lambda (file &rest _) (push file reads) "#+TODO: REMOTE | DONE\n")))
      (should (equal "REMOTE" (imoogi-emacsctl--task-keyword
                               "#+SETUPFILE: https://example.invalid/setup.org\n"
                               '((sequence "TODO" "|" "DONE"))))))
    (should (member "https://example.invalid/setup.org" reads))))
