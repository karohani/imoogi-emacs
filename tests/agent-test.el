;;; agent-test.el --- tests for the coding-agent IPC receiver -*- lexical-binding: t; -*-

;; SPEC-AGENTIPC-001. Test names start with `imoogi-agent-acNN-' so that the
;; acceptance.md §0.1 selector `^imoogi-agent-acNN-' runs exactly one AC.

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'json)
(require 'subr-x)

;; The module is loaded directly (not only through boot.el) so the tests do
;; not depend on its position in the boot list. The module defines only
;; defvar/defconst/defun forms, so loading it twice is harmless. NOERROR keeps
;; the rest of the suite runnable when the file is absent.
(load (expand-file-name "modules/development/30-agent.el" imoogi-test-root) t t)

(defconst imoogi-agent-test--timestamp "2026-09-27T10:47:00+09:00")

(defun imoogi-agent-test--base (type payload &rest extra)
  "Return a valid envelope alist of TYPE with PAYLOAD and EXTRA cells."
  (append `((version . "1")
            (type . ,type)
            (timestamp . ,imoogi-agent-test--timestamp))
          extra
          `((payload . ,payload))))

(defun imoogi-agent-test--set (alist key value)
  "Return ALIST with KEY replaced by VALUE (appended when absent)."
  (append (assq-delete-all key (copy-alist alist)) (list (cons key value))))

(defun imoogi-agent-test--drop (alist key)
  "Return ALIST without KEY."
  (assq-delete-all key (copy-alist alist)))

(defun imoogi-agent-test--json (alist)
  "Serialize ALIST as a JSON string."
  (json-serialize alist))

(defun imoogi-agent-test--write (content &optional suffix)
  "Write CONTENT (a string) to a fresh temporary file and return its path."
  (let ((file (make-temp-file "imoogi-aipc-test-" nil (or suffix ".json"))))
    (let ((coding-system-for-write 'utf-8-unix))
      (write-region content nil file nil 'silent))
    file))

(defun imoogi-agent-test--log-lines ()
  "Return the lines of the agent log buffer (nil when it does not exist)."
  (when-let* ((buffer (get-buffer "*imoogi-agent*")))
    (with-current-buffer buffer
      (split-string (buffer-string) "\n" t))))

(defun imoogi-agent-test--clear-messages ()
  "Empty *Messages* so that lines added by the next call can be observed."
  (with-current-buffer (messages-buffer)
    (let ((inhibit-read-only t))
      (erase-buffer))))

(defun imoogi-agent-test--messages ()
  "Return the lines currently in *Messages*."
  (with-current-buffer (messages-buffer)
    (split-string (buffer-string) "\n" t)))

(defun imoogi-agent-test--receive-json (json)
  "Receive JSON through the string entry point; return (STATUS MESSAGES NEW-LOG)."
  (imoogi-agent-test--clear-messages)
  (let* ((before (length (imoogi-agent-test--log-lines)))
         (status (imoogi-agent-receive-json json)))
    (list status
          (imoogi-agent-test--messages)
          (nthcdr before (imoogi-agent-test--log-lines)))))

(defun imoogi-agent-test--receive-file (file)
  "Receive FILE through the file entry point; return (STATUS MESSAGES NEW-LOG)."
  (imoogi-agent-test--clear-messages)
  (let* ((before (length (imoogi-agent-test--log-lines)))
         (status (imoogi-agent-receive-file file)))
    (list status
          (imoogi-agent-test--messages)
          (nthcdr before (imoogi-agent-test--log-lines)))))

(defmacro imoogi-agent-test--with-temp-files (vars &rest body)
  "Bind VARS to nil, run BODY, then delete every file path bound in VARS."
  (declare (indent 1))
  `(let ,(mapcar (lambda (v) (list v nil)) vars)
     (unwind-protect
         (progn ,@body)
       (dolist (file (list ,@vars))
         (cond ((not (stringp file)))
               ((file-directory-p file) (delete-directory file t))
               ((or (file-exists-p file) (file-symlink-p file))
                (delete-file file)))))))

(defun imoogi-agent-test--both (json)
  "Receive JSON through both entry points; return the two result triples."
  (let ((file (imoogi-agent-test--write json)))
    (unwind-protect
        (list (imoogi-agent-test--receive-json json)
              (imoogi-agent-test--receive-file file))
      (delete-file file))))

(defun imoogi-agent-test--window-state ()
  "Return an observable snapshot of the window layout."
  (list (selected-window) (selected-frame)
        (length (window-list nil 'never))
        (mapcar #'window-buffer (window-list nil 'never))))

;;; AC-AIPC-001 --------------------------------------------------------------

(ert-deftest imoogi-agent-ac01-message-notifies-and-logs ()
  "AC-AIPC-001: message event → literal echo text, one log line, read-only log."
  (let* ((text "50% 완료 %s")
         (json (imoogi-agent-test--json
                (imoogi-agent-test--base "message" `((text . ,text))
                                         '(project . "imoogi-emacs")
                                         '(session . "s-42")))))
    (imoogi-agent-test--with-temp-files (file)
      (setq file (imoogi-agent-test--write json))
      (let ((content (with-temp-buffer
                       (insert-file-contents-literally file)
                       (buffer-string)))
            (mtime (file-attribute-modification-time (file-attributes file))))
        (pcase-let ((`(,status ,messages ,log) (imoogi-agent-test--receive-file file)))
          (should (equal status "ok"))
          (should (= 1 (cl-count text messages :test #'equal)))
          (should (= 1 (length log)))
          (should (string-match-p "message" (car log)))
          (should (string-match-p "imoogi-emacs" (car log)))
          (should (string-match-p "s-42" (car log)))
          (should (string-match-p "\\bok\\b" (car log))))
        (should (with-current-buffer "*imoogi-agent*" buffer-read-only))
        (pcase-let ((`(,status ,messages ,log) (imoogi-agent-test--receive-json json)))
          (should (equal status "ok"))
          (should (= 1 (cl-count text messages :test #'equal)))
          (should (= 1 (length log)))
          (should (string-match-p "message.*imoogi-emacs.*s-42.*\\bok\\b" (car log))))
        (should (equal content (with-temp-buffer
                                 (insert-file-contents-literally file)
                                 (buffer-string))))
        (should (equal mtime (file-attribute-modification-time
                              (file-attributes file))))))))

(ert-deftest imoogi-agent-ac01-newlines-become-one-line ()
  "AC-AIPC-001 And: LF, CRLF, and CR each become one ⏎ in notice and log."
  (let ((json (concat "{\"version\":\"1\",\"type\":\"message\","
                      "\"timestamp\":\"" imoogi-agent-test--timestamp "\","
                      "\"project\":\"p\\nq\","
                      "\"payload\":{\"text\":\"첫 줄\\n둘째 줄\\r\\n셋째 줄\\r넷째 줄\"}}")))
    (pcase-let ((`(,status ,messages ,log) (imoogi-agent-test--receive-json json)))
      (should (equal status "ok"))
      (should (= 1 (cl-count "첫 줄⏎둘째 줄⏎셋째 줄⏎넷째 줄" messages :test #'equal)))
      (should (= 1 (length log)))
      (should (string-match-p "p⏎q" (car log))))))

;;; AC-AIPC-005 (a)(b) -------------------------------------------------------

(ert-deftest imoogi-agent-ac05-task-finished-success-and-failed ()
  "AC-AIPC-005 (a)(b): success and failed notices carry the summary."
  (pcase-let ((`(,status ,messages ,_)
               (imoogi-agent-test--receive-json
                (imoogi-agent-test--json
                 (imoogi-agent-test--base "task-finished"
                                          '((status . "success")
                                            (summary . "테스트 42개 통과")))))))
    (should (equal status "ok"))
    (let ((lines (cl-remove-if-not
                  (lambda (l) (string-prefix-p "✓ Agent task finished" l)) messages)))
      (should (= 1 (length lines)))
      (should (string-match-p "테스트 42개 통과" (car lines)))))
  (pcase-let ((`(,status ,messages ,_)
               (imoogi-agent-test--receive-json
                (imoogi-agent-test--json
                 (imoogi-agent-test--base "task-finished"
                                          '((status . "failed")
                                            (summary . "빌드 실패")))))))
    (should (equal status "ok"))
    (let ((lines (cl-remove-if-not
                  (lambda (l) (string-prefix-p "✗ Agent task failed" l)) messages)))
      (should (= 1 (length lines)))
      (should (string-match-p "빌드 실패" (car lines))))))

;;; AC-AIPC-006 ---------------------------------------------------------------

(ert-deftest imoogi-agent-ac06-syntax-trailing-and-duplicate-keys ()
  "AC-AIPC-006: parse, trailing-content, and duplicate-key rejections."
  (let* ((valid (imoogi-agent-test--json
                 (imoogi-agent-test--base "message" '((text . "hi")))))
         (ts imoogi-agent-test--timestamp)
         (cases
          `(("{\"version\":\"1\"," . "error:parse")
            (,(make-string 10001 ?\[) . "error:parse")
            (,(concat valid "{}") . "error:trailing-content")
            (,(concat "{\"version\":\"1\",\"type\":\"message\",\"type\":\"open-file\","
                      "\"timestamp\":\"" ts "\",\"payload\":{\"text\":\"hi\"}}")
             . "error:duplicate-key")
            (,(concat "{\"version\":\"1\",\"type\":\"message\","
                      "\"timestamp\":\"" ts "\","
                      "\"payload\":{\"text\":\"a\",\"text\":\"b\"}}")
             . "error:duplicate-key"))))
    (delete-other-windows)
    (dolist (case cases)
      (let ((windows (imoogi-agent-test--window-state)))
        (dolist (result (imoogi-agent-test--both (car case)))
          (pcase-let ((`(,status ,messages ,log) result))
            (should (equal status (cdr case)))
            (should-not messages)
            (should (= 1 (length log)))
            (should (string-match-p (regexp-quote (substring (cdr case) 6)) (car log)))))
        (should (equal windows (imoogi-agent-test--window-state)))))))

;;; AC-AIPC-007 ---------------------------------------------------------------

(ert-deftest imoogi-agent-ac07-schema-violations ()
  "AC-AIPC-007: schema violations map to the specified reasons."
  (let* ((msg (imoogi-agent-test--base "message" '((text . "hi"))))
         (goto (imoogi-agent-test--base "goto-location"
                                        '((path . "/tmp/x.txt") (line . 3))))
         (finish (imoogi-agent-test--base "task-finished" '((status . "success"))))
         (artifact (imoogi-agent-test--base "artifact-created"
                                            '((path . "/tmp/x.md"))))
         (payload-set (lambda (event key value)
                        (imoogi-agent-test--set
                         event 'payload
                         (imoogi-agent-test--set (alist-get 'payload event) key value))))
         (payload-drop (lambda (event key)
                         (imoogi-agent-test--set
                          event 'payload
                          (imoogi-agent-test--drop (alist-get 'payload event) key))))
         (cases
          `(("a" ,(imoogi-agent-test--set msg 'version "2") "error:bad-version")
            ("b" ,(imoogi-agent-test--set msg 'version 1) "error:bad-version")
            ("c" ,(imoogi-agent-test--set msg 'type "progress") "error:unknown-type")
            ("d" ,(imoogi-agent-test--drop msg 'type) "error:unknown-type")
            ("e" ,(funcall payload-drop msg 'text) "error:bad-field")
            ("f" ,(funcall payload-set msg 'text "") "error:bad-field")
            ("g" ,(funcall payload-set msg 'text :null) "error:bad-field")
            ("h0" ,(funcall payload-set goto 'line 0) "error:bad-field")
            ("h1" ,(funcall payload-set goto 'line "3") "error:bad-field")
            ("h2" ,(funcall payload-set goto 'line 2.5) "error:bad-field")
            ("i0" ,(funcall payload-set finish 'status "done") "error:bad-field")
            ("i1" ,(funcall payload-set finish 'status :false) "error:bad-field")
            ("j0" ,(imoogi-agent-test--drop msg 'timestamp) "error:bad-field")
            ("j1" ,(imoogi-agent-test--set msg 'timestamp "") "error:bad-field")
            ("k" ,(imoogi-agent-test--set msg 'payload ["x"]) "error:bad-field")
            ("l" ,(funcall payload-set
                           (imoogi-agent-test--set
                            (imoogi-agent-test--set msg 'project :null) 'extra 1)
                           'extra 1)
             "ok")
            ("m" ,(funcall payload-drop
                           (imoogi-agent-test--set msg 'version "2") 'text)
             "error:bad-version")
            ("n0" ,(imoogi-agent-test--set msg 'project []) "error:bad-field")
            ("n1" ,(imoogi-agent-test--set msg 'session nil) "error:bad-field")
            ("n2" ,(imoogi-agent-test--set msg 'project :false) "error:bad-field")
            ("n3" ,(imoogi-agent-test--set msg 'session 3) "error:bad-field")
            ("o0" ,(funcall payload-set finish 'summary []) "error:bad-field")
            ("o1" ,(funcall payload-set artifact 'title nil) "error:bad-field")
            ("o2" ,(funcall payload-set goto 'column "2") "error:bad-field"))))
    (delete-other-windows)
    (dolist (case cases)
      (let ((windows (imoogi-agent-test--window-state)))
        (pcase-let ((`(,status ,messages ,log)
                     (imoogi-agent-test--receive-json
                      (imoogi-agent-test--json (nth 1 case)))))
          (should (equal (list (car case) status) (list (car case) (nth 2 case))))
          (should (= 1 (length log)))
          (unless (equal (car case) "l")
            (should-not messages)
            (should (equal windows (imoogi-agent-test--window-state)))))))))

;;; AC-AIPC-008 ---------------------------------------------------------------

(defun imoogi-agent-test--sized-message (bytes)
  "Return a valid message event JSON string of exactly BYTES UTF-8 bytes."
  (let* ((skeleton (imoogi-agent-test--json
                    (imoogi-agent-test--base "message" '((text . "hi"))
                                             '(pad . ""))))
         (missing (- bytes (string-bytes skeleton))))
    (imoogi-agent-test--json
     (imoogi-agent-test--base "message" '((text . "hi"))
                              `(pad . ,(make-string missing ?x))))))

(ert-deftest imoogi-agent-ac08-size-limit-and-file-preconditions ()
  "AC-AIPC-008: size limit before reading; event-file preconditions."
  (require 'tramp)
  (require 'tramp-sh)
  (let ((inserts 0)
        (connections 0)
        (limit 1048576))
    (imoogi-agent-test--with-temp-files (big exact dir)
      (setq big (imoogi-agent-test--write (make-string (1+ limit) ?x)))
      (setq exact (imoogi-agent-test--write
                   (imoogi-agent-test--sized-message limit)))
      (setq dir (make-temp-file "imoogi-aipc-test-dir-" t))
      (should (= (1+ limit) (file-attribute-size (file-attributes big))))
      (should (= limit (file-attribute-size (file-attributes exact))))
      (cl-letf* ((orig-insert (symbol-function 'insert-file-contents))
                 (orig-literal (symbol-function 'insert-file-contents-literally))
                 ((symbol-function 'insert-file-contents)
                  (lambda (&rest args) (cl-incf inserts) (apply orig-insert args)))
                 ((symbol-function 'insert-file-contents-literally)
                  (lambda (&rest args) (cl-incf inserts) (apply orig-literal args)))
                 ((symbol-function 'tramp-maybe-open-connection)
                  (lambda (&rest _) (cl-incf connections)
                    (error "Remote connection attempted"))))
        ;; (a) oversized file: rejected without reading its content.
        (should (equal (car (imoogi-agent-test--receive-file big)) "error:too-large"))
        (should (= 0 inserts))
        ;; (b) oversized JSON string.
        (should (equal (car (imoogi-agent-test--receive-json
                             (make-string (1+ limit) ?x)))
                       "error:too-large"))
        ;; (c) exactly at the limit is accepted.
        (should (equal (car (imoogi-agent-test--receive-file exact)) "ok"))
        ;; (d)-(g) event-file preconditions.
        (should (equal (car (imoogi-agent-test--receive-file
                             (concat big ".does-not-exist")))
                       "error:unreadable"))
        (should (equal (car (imoogi-agent-test--receive-file "event.json"))
                       "error:unreadable"))
        (should (equal (car (imoogi-agent-test--receive-file dir))
                       "error:unreadable"))
        (should (equal (car (imoogi-agent-test--receive-file "/ssh:host:/tmp/e.json"))
                       "error:unreadable"))
        (should (= 0 connections))))))

;;; Window-display fixtures (AC-AIPC-002 ~ 005, 009, 010) ---------------------

(defun imoogi-agent-test--user-buffer ()
  "Show a fresh user buffer A with point 10 in the only window; return it."
  (delete-other-windows)
  (let ((buffer (get-buffer-create "*imoogi-agent-test-user*")))
    (with-current-buffer buffer
      (erase-buffer)
      (insert (make-string 40 ?a)))
    (set-window-buffer (selected-window) buffer)
    (set-buffer buffer)
    (goto-char 10)
    (set-window-point (selected-window) 10)
    buffer))

(defun imoogi-agent-test--assert-user-untouched (buffer window frame)
  "Assert the user still sits in BUFFER at point 10 in WINDOW of FRAME."
  (should (eq (selected-window) window))
  (should (eq (selected-frame) frame))
  (should (eq (window-buffer window) buffer))
  (should (= 10 (window-point window)))
  (should (= 10 (with-current-buffer buffer (point)))))

(defun imoogi-agent-test--file-window (file)
  "Return the window displaying the buffer that visits FILE, or nil."
  (when-let* ((buffer (find-buffer-visiting file)))
    (get-buffer-window buffer)))

(defun imoogi-agent-test--kill-visiting (&rest files)
  "Kill buffers visiting FILES without asking."
  (dolist (file files)
    (when-let* ((buffer (and (stringp file) (find-buffer-visiting file))))
      (with-current-buffer buffer (set-buffer-modified-p nil))
      (kill-buffer buffer))))

(defun imoogi-agent-test--event-json (type payload)
  "Return a JSON event of TYPE with PAYLOAD."
  (imoogi-agent-test--json (imoogi-agent-test--base type payload)))

;;; AC-AIPC-002 ---------------------------------------------------------------

(ert-deftest imoogi-agent-ac02-open-file-keeps-focus ()
  "AC-AIPC-002: open-file shows F in another window and never selects it."
  (imoogi-agent-test--with-temp-files (file)
    (setq file (imoogi-agent-test--write "hello\n" ".txt"))
    (unwind-protect
        (let* ((user (imoogi-agent-test--user-buffer))
               (window (selected-window))
               (frame (selected-frame))
               (json (imoogi-agent-test--event-json "open-file" `((path . ,file)))))
          (should (equal (car (imoogi-agent-test--receive-json json)) "ok"))
          (let ((shown (imoogi-agent-test--file-window file)))
            (should (windowp shown))
            (should-not (eq shown window))
            (imoogi-agent-test--assert-user-untouched user window frame)
            (let ((count (length (window-list nil 'never))))
              (should (equal (car (imoogi-agent-test--receive-json json)) "ok"))
              (should (= count (length (window-list nil 'never))))
              (should (eq shown (imoogi-agent-test--file-window file)))
              (imoogi-agent-test--assert-user-untouched user window frame))))
      (imoogi-agent-test--kill-visiting file)
      (delete-other-windows))))

;;; AC-AIPC-003 ---------------------------------------------------------------

(ert-deftest imoogi-agent-ac03-goto-moves-only-target-window ()
  "AC-AIPC-003: goto-location clamps line/column and moves only its window."
  (imoogi-agent-test--with-temp-files (file)
    (setq file (imoogi-agent-test--write
                "line one\nline two\nline three\nline four\nline five\n" ".txt"))
    (unwind-protect
        (let* ((user (imoogi-agent-test--user-buffer))
               (window (selected-window))
               (frame (selected-frame))
               (target (find-file-noselect file t))
               (expect
                (lambda (line column)
                  (with-current-buffer target
                    (save-excursion
                      (goto-char (point-min))
                      (forward-line (1- line))
                      (if (eq column 'eol) (end-of-line) (move-to-column column))
                      (point))))))
          (dolist (case `((((line . 3) (column . 4)) ,(funcall expect 3 3))
                          (((line . 3)) ,(funcall expect 3 0))
                          (((line . 99)) ,(with-current-buffer target
                                            (save-excursion
                                              (goto-char (point-max))
                                              (line-beginning-position))))
                          (((line . 2) (column . 999)) ,(funcall expect 2 'eol))))
            (should (equal (car (imoogi-agent-test--receive-json
                                 (imoogi-agent-test--event-json
                                  "goto-location" `((path . ,file) ,@(car case)))))
                           "ok"))
            (let ((shown (imoogi-agent-test--file-window file)))
              (should (windowp shown))
              (should-not (eq shown window))
              (should (= (cadr case) (window-point shown))))
            (imoogi-agent-test--assert-user-untouched user window frame)))
      (imoogi-agent-test--kill-visiting file)
      (delete-other-windows))))

;;; AC-AIPC-004 ---------------------------------------------------------------

(ert-deftest imoogi-agent-ac04-artifact-created-notice-and-display ()
  "AC-AIPC-004: artifact notice uses the title or the file name."
  (let* ((dir (make-temp-file "imoogi-aipc-test-dir-" t))
         ;; The file name must not contain the artifact type, or the log
         ;; assertion below would pass without artifactType being logged.
         (file (expand-file-name "260927-104700-weekly.md" dir)))
    (unwind-protect
        (progn
          (with-temp-file file (insert "# weekly\n"))
          (let* ((user (imoogi-agent-test--user-buffer))
                 (window (selected-window))
                 (frame (selected-frame)))
            (pcase-let ((`(,status ,messages ,log)
                         (imoogi-agent-test--receive-json
                          (imoogi-agent-test--event-json
                           "artifact-created" `((path . ,file)
                                                (artifactType . "report")
                                                (title . "주간 보고"))))))
              (should (equal status "ok"))
              (should (= 1 (cl-count "Agent artifact created: 주간 보고" messages
                                     :test #'equal)))
              (should (string-match-p " ok report 주간 보고 " (car log))))
            (let ((shown (imoogi-agent-test--file-window file)))
              (should (windowp shown))
              (should-not (eq shown window)))
            (imoogi-agent-test--assert-user-untouched user window frame)
            (pcase-let ((`(,status ,messages ,_)
                         (imoogi-agent-test--receive-json
                          (imoogi-agent-test--event-json
                           "artifact-created" `((path . ,file))))))
              (should (equal status "ok"))
              (should (= 1 (cl-count "Agent artifact created: 260927-104700-weekly.md"
                                     messages :test #'equal))))
            (imoogi-agent-test--assert-user-untouched user window frame)))
      (imoogi-agent-test--kill-visiting file)
      (delete-other-windows)
      (delete-directory dir t))))

;;; AC-AIPC-005 (c) -----------------------------------------------------------

(ert-deftest imoogi-agent-ac05-task-finished-artifact-displayed ()
  "AC-AIPC-005 (c): success with an artifact notifies and shows the file."
  (imoogi-agent-test--with-temp-files (file)
    (setq file (imoogi-agent-test--write "result\n" ".txt"))
    (unwind-protect
        (let* ((user (imoogi-agent-test--user-buffer))
               (window (selected-window))
               (frame (selected-frame)))
          (pcase-let ((`(,status ,messages ,_)
                       (imoogi-agent-test--receive-json
                        (imoogi-agent-test--event-json
                         "task-finished" `((status . "success") (artifact . ,file))))))
            (should (equal status "ok"))
            (should (= 1 (cl-count-if (lambda (l) (string-prefix-p "✓ Agent task finished" l))
                                      messages))))
          (let ((shown (imoogi-agent-test--file-window file)))
            (should (windowp shown))
            (should-not (eq shown window)))
          (imoogi-agent-test--assert-user-untouched user window frame))
      (imoogi-agent-test--kill-visiting file)
      (delete-other-windows))))

;;; AC-AIPC-009 ---------------------------------------------------------------

(ert-deftest imoogi-agent-ac09-payload-path-policy ()
  "AC-AIPC-009: payload paths must be absolute, local, existing regular files."
  (require 'tramp)
  (require 'tramp-sh)
  (let* ((dir (make-temp-file "imoogi-aipc-test-dir-" t))
         (f (expand-file-name "f.txt" dir))
         (d (expand-file-name "d" dir))
         (lf (expand-file-name "lf" dir))
         (ld (expand-file-name "ld" dir))
         (lx (expand-file-name "lx" dir))
         (missing (expand-file-name "missing.txt" dir))
         (connections 0))
    (unwind-protect
        (progn
          (with-temp-file f (insert "file\n"))
          (make-directory d)
          (make-symbolic-link f lf)
          (make-symbolic-link d ld)
          (make-symbolic-link (expand-file-name "gone" dir) lx)
          (imoogi-agent-test--user-buffer)
          (cl-letf (((symbol-function 'tramp-maybe-open-connection)
                     (lambda (&rest _) (cl-incf connections)
                       (error "Remote connection attempted"))))
            ;; (a) and (b) are accepted; (b) opens the resolved file.
            (should (equal (car (imoogi-agent-test--receive-json
                                 (imoogi-agent-test--event-json
                                  "open-file" `((path . ,f)))))
                           "ok"))
            (should (windowp (imoogi-agent-test--file-window (file-truename f))))
            (imoogi-agent-test--kill-visiting (file-truename f))
            (imoogi-agent-test--user-buffer)
            (should (equal (car (imoogi-agent-test--receive-json
                                 (imoogi-agent-test--event-json
                                  "open-file" `((path . ,lf)))))
                           "ok"))
            (let ((buffer (window-buffer (imoogi-agent-test--file-window (file-truename f)))))
              (should (equal (buffer-file-name buffer) (file-truename f))))
            (imoogi-agent-test--kill-visiting (file-truename f))
            ;; (c)-(i) are rejected without any visible change.
            (imoogi-agent-test--user-buffer)
            (let ((layout (imoogi-agent-test--window-state))
                  (buffers (buffer-list)))
              (dolist (path (list (file-name-nondirectory f) missing d ld lx
                                  "/dev/null" "/ssh:host:/etc/hosts"))
                (should (equal (list path (car (imoogi-agent-test--receive-json
                                                (imoogi-agent-test--event-json
                                                 "open-file" `((path . ,path))))))
                               (list path "error:bad-path"))))
              ;; The same policy covers every payload path field.
              (should (equal (car (imoogi-agent-test--receive-json
                                   (imoogi-agent-test--event-json
                                    "goto-location" `((path . ,missing) (line . 1)))))
                             "error:bad-path"))
              (should (equal (car (imoogi-agent-test--receive-json
                                   (imoogi-agent-test--event-json
                                    "artifact-created" `((path . ,missing)))))
                             "error:bad-path"))
              (pcase-let ((`(,status ,messages ,_)
                           (imoogi-agent-test--receive-json
                            (imoogi-agent-test--event-json
                             "task-finished" `((status . "success")
                                               (artifact . ,missing))))))
                (should (equal status "error:bad-path"))
                (should-not messages))
              (should (equal layout (imoogi-agent-test--window-state)))
              (should (equal (sort (mapcar #'buffer-name buffers) #'string<)
                             (sort (mapcar #'buffer-name (buffer-list)) #'string<)))))
          (should (= 0 connections)))
      (imoogi-agent-test--kill-visiting (file-truename f))
      (delete-other-windows)
      (delete-directory dir t))))

;;; AC-AIPC-010 ---------------------------------------------------------------

(defvar imoogi-agent-test-unsafe nil
  "A variable with no safe-local-variable property (unsafe as a local).")

(ert-deftest imoogi-agent-ac10-never-waits-for-input ()
  "AC-AIPC-010: indirect prompts are suppressed; project notes are untouched."
  (let* ((dir (make-temp-file "imoogi-aipc-test-dir-" t))
         (large (expand-file-name "large.txt" dir))
         (locals (expand-file-name "locals.txt" dir))
         (changed (expand-file-name "changed.txt" dir))
         (inputs 0)
         (notes-calls 0)
         (large-file-warning-threshold 100)
         (notes-counter (lambda (&rest _) (cl-incf notes-calls)))
         (notes-functions nil))
    (unwind-protect
        (progn
          (with-temp-file large (insert (make-string 1000 ?l)))
          (with-temp-file locals
            (insert "-*- imoogi-agent-test-unsafe: 1; eval: (message \"x\") -*-\nbody\n"))
          (with-temp-file changed (insert "old\n"))
          (let ((buffer (find-file-noselect (file-truename changed) t)))
            (with-current-buffer buffer (set-buffer-modified-p nil)))
          (with-temp-file changed (insert "new content on disk\n"))
          (set-file-times changed (time-add (current-time) 10))
          (mapatoms (lambda (symbol)
                      (when (and (fboundp symbol)
                                 (string-prefix-p "imoogi-project-notes-" (symbol-name symbol))
                                 (functionp (symbol-function symbol)))
                        (push symbol notes-functions))))
          (dolist (symbol notes-functions)
            (advice-add symbol :before notes-counter))
          (unwind-protect
              (cl-letf (((symbol-function 'read-string)
                         (lambda (&rest _) (cl-incf inputs) (error "Input requested")))
                        ((symbol-function 'read-from-minibuffer)
                         (lambda (&rest _) (cl-incf inputs) (error "Input requested")))
                        ((symbol-function 'y-or-n-p)
                         (lambda (&rest _) (cl-incf inputs) (error "Input requested")))
                        ((symbol-function 'yes-or-no-p)
                         (lambda (&rest _) (cl-incf inputs) (error "Input requested")))
                        ((symbol-function 'read-key)
                         (lambda (&rest _) (cl-incf inputs) (error "Input requested")))
                        ((symbol-function 'read-char)
                         (lambda (&rest _) (cl-incf inputs) (error "Input requested")))
                        ((symbol-function 'read-event)
                         (lambda (&rest _) (cl-incf inputs) (error "Input requested"))))
                (dolist (file (list large locals changed))
                  (imoogi-agent-test--user-buffer)
                  (pcase-let ((`(,status ,messages ,_)
                               (imoogi-agent-test--receive-json
                                (imoogi-agent-test--event-json
                                 "open-file" `((path . ,file))))))
                    (should (equal (list file status) (list file "ok")))
                    (should-not (member "x" messages)))
                  (should (equal (list file (car (imoogi-agent-test--receive-json
                                                  (imoogi-agent-test--event-json
                                                   "goto-location"
                                                   `((path . ,file) (line . 1))))))
                                 (list file "ok"))))
                ;; Remaining event types, for the project-notes call count.
                (dolist (event `(("message" ((text . "hi")))
                                 ("artifact-created" ((path . ,large)))
                                 ("task-finished" ((status . "failed")))))
                  (should (equal (car (imoogi-agent-test--receive-json
                                       (imoogi-agent-test--event-json
                                        (car event) (cadr event))))
                                 "ok"))))
            (dolist (symbol notes-functions)
              (advice-remove symbol notes-counter)))
          (should (= 0 inputs))
          (should (= 0 notes-calls))
          (should-not (with-current-buffer (find-buffer-visiting (file-truename locals))
                        (local-variable-p 'imoogi-agent-test-unsafe)))
          (with-temp-buffer
            (insert-file-contents
             (expand-file-name "modules/development/30-agent.el" imoogi-test-root))
            (should-not (search-forward "26-project-notes" nil t))
            (goto-char (point-min))
            (should-not (search-forward "imoogi-project-notes" nil t))))
      (imoogi-agent-test--kill-visiting (file-truename large) (file-truename locals)
                                        (file-truename changed))
      (delete-other-windows)
      (delete-directory dir t))))

;;; SPEC-AGENTIPC-002 ---------------------------------------------------------
;;
;; Test names start with `imoogi-agent-aiphNN-' so that the spec.md § 3.3.0
;; selector `^imoogi-agent-aiphNN-' runs exactly one AC.

;; Special in the module; declared here so the `let' in aiph04 binds it
;; dynamically even when the module failed to load.
(defvar imoogi-agent-max-payload-bytes)

(defun imoogi-agent-test--receive-file-counting (file)
  "Receive FILE; return (STATUS MESSAGES NEW-LOG INSERTS).
INSERTS counts calls of the `insert-file-contents' family during the call."
  (let ((inserts 0))
    (cl-letf* ((orig-insert (symbol-function 'insert-file-contents))
               (orig-literal (symbol-function 'insert-file-contents-literally))
               ((symbol-function 'insert-file-contents)
                (lambda (&rest args) (cl-incf inserts) (apply orig-insert args)))
               ((symbol-function 'insert-file-contents-literally)
                (lambda (&rest args) (cl-incf inserts) (apply orig-literal args))))
      (append (imoogi-agent-test--receive-file file) (list inserts)))))

(defun imoogi-agent-test--sparse-file (file bytes)
  "Create FILE as a sparse file of BYTES bytes with dd; return FILE."
  (should (= 0 (call-process "dd" nil nil nil "if=/dev/zero" (concat "of=" file)
                             "bs=1" "count=0" (format "seek=%d" bytes))))
  (should (= bytes (file-attribute-size (file-attributes file))))
  file)

;;; AC-AIPH-001 ---------------------------------------------------------------

(ert-deftest imoogi-agent-aiph01-untrusted-event-file ()
  "AC-AIPH-001: foreign-owned or group/other-writable event files are refused."
  (let ((json (imoogi-agent-test--json
               (imoogi-agent-test--base "message" '((text . "aiph")))))
        (real-uid (user-uid)))
    (imoogi-agent-test--with-temp-files (file)
      (setq file (imoogi-agent-test--write json))
      (should (= #o600 (file-modes file)))
      (let ((untrusted
             (lambda (label)
               (pcase-let ((`(,status ,messages ,log ,inserts)
                            (imoogi-agent-test--receive-file-counting file)))
                 (should (equal (list label status) (list label "error:untrusted")))
                 (should (equal (list label inserts) (list label 0)))
                 (should-not (member "aiph" messages))
                 (should (= 1 (length log)))
                 (should (string-match-p "\\] - project=- session=- untrusted$"
                                         (car log)))))))
        ;; (a) group-writable, (b) other-writable.
        (set-file-modes file #o620)
        (funcall untrusted "a")
        (set-file-modes file #o602)
        (funcall untrusted "b")
        ;; (c) mode 0600 but owned by someone else.
        (set-file-modes file #o600)
        (cl-letf (((symbol-function 'user-uid) (lambda () (1+ real-uid))))
          (funcall untrusted "c"))
        ;; (d) control: mode 0600, owned by the user.
        (pcase-let ((`(,status ,messages ,_ ,_)
                     (imoogi-agent-test--receive-file-counting file)))
          (should (equal status "ok"))
          (should (member "aiph" messages)))))))

;;; AC-AIPH-002 ---------------------------------------------------------------

(ert-deftest imoogi-agent-aiph02-trust-scope-order-and-links ()
  "AC-AIPH-002: trust check scope, its order, and symbolic-link resolution."
  (let* ((json (imoogi-agent-test--json
                (imoogi-agent-test--base "message" '((text . "aiph")))))
         (dir (make-temp-file "imoogi-aipc-test-dir-" t))
         (big (expand-file-name "big.json" dir))
         (sub (expand-file-name "sub" dir))
         (bad (expand-file-name "bad.json" dir))
         (good (expand-file-name "good.json" dir))
         (bad-link (expand-file-name "bad-link.json" dir))
         (good-link (expand-file-name "good-link.json" dir)))
    (unwind-protect
        (progn
          ;; (a) the string entry point has no file, so no trust check.
          (should (equal (car (imoogi-agent-test--receive-json json)) "ok"))
          ;; (b) untrusted comes before too-large.
          (let ((coding-system-for-write 'utf-8-unix))
            (write-region (make-string 1048577 ?x) nil big nil 'silent))
          (should (= 1048577 (file-attribute-size (file-attributes big))))
          (set-file-modes big #o620)
          (should (equal (car (imoogi-agent-test--receive-file big)) "error:untrusted"))
          ;; (c) file preconditions come before the trust check.
          (make-directory sub)
          (set-file-modes sub #o620)
          (should (equal (car (imoogi-agent-test--receive-file sub)) "error:unreadable"))
          (set-file-modes sub #o700)
          ;; (d) a link to an untrusted file is judged by its target.
          (let ((coding-system-for-write 'utf-8-unix))
            (write-region json nil bad nil 'silent)
            (write-region json nil good nil 'silent))
          (set-file-modes bad #o620)
          (set-file-modes good #o600)
          (make-symbolic-link bad bad-link)
          (make-symbolic-link good good-link)
          (should (equal (car (imoogi-agent-test--receive-file bad-link))
                         "error:untrusted"))
          ;; (e) a link to a trusted file is accepted.
          (should (equal (car (imoogi-agent-test--receive-file good-link)) "ok")))
      (when (file-directory-p sub)
        (set-file-modes sub #o700))
      (delete-directory dir t))))

;;; AC-AIPH-003 ---------------------------------------------------------------

(ert-deftest imoogi-agent-aiph03-payload-too-large-rejects-whole-event ()
  "AC-AIPH-003: a payload file over the cap rejects the whole event."
  (let* ((dir (make-temp-file "imoogi-aipc-test-dir-" t))
         (p (expand-file-name "big.bin" dir))
         (lp (expand-file-name "big-link.bin" dir)))
    (unwind-protect
        (progn
          (imoogi-agent-test--sparse-file p 104857601)
          (make-symbolic-link p lp)
          (dolist (case `(("a" "open-file" ((path . ,p)))
                          ("b" "goto-location" ((path . ,p) (line . 1)))
                          ("c" "artifact-created" ((path . ,p) (title . "big")))
                          ("d" "task-finished" ((status . "success") (artifact . ,p)))
                          ("e" "open-file" ((path . ,lp)))))
            (imoogi-agent-test--user-buffer)
            (let ((layout (imoogi-agent-test--window-state))
                  (point (window-point (selected-window))))
              (pcase-let ((`(,status ,messages ,log)
                           (imoogi-agent-test--receive-json
                            (imoogi-agent-test--event-json (nth 1 case) (nth 2 case)))))
                (should (equal (list (car case) status)
                               (list (car case) "error:payload-too-large")))
                (should-not (find-buffer-visiting p))
                (should (equal layout (imoogi-agent-test--window-state)))
                (should (= point (window-point (selected-window))))
                (should-not (cl-some (lambda (l)
                                       (or (string-prefix-p "Agent artifact created" l)
                                           (string-prefix-p "✓ Agent task finished" l)))
                                     messages))
                (should (= 1 (length log)))
                (should (string-match-p "payload-too-large" (car log))))))
          ;; (f) the path policy comes first.
          (let ((default-directory dir))
            (should (equal (car (imoogi-agent-test--receive-json
                                 (imoogi-agent-test--event-json
                                  "open-file" `((path . ,(file-relative-name p dir))))))
                           "error:bad-path"))))
      (imoogi-agent-test--kill-visiting p (file-truename p))
      (delete-other-windows)
      (delete-directory dir t))))

;;; AC-AIPH-004 ---------------------------------------------------------------

(ert-deftest imoogi-agent-aiph04-payload-cap-boundary-without-prompt ()
  "AC-AIPH-004: the cap is inclusive, fixed, and opens files without asking."
  (should (= imoogi-agent-max-payload-bytes 104857600))
  (let ((original large-file-warning-threshold))
    (imoogi-agent-test--with-temp-files (a b)
      (setq a (imoogi-agent-test--write "0123456789abcdef" ".txt"))
      (setq b (imoogi-agent-test--write "0123456789abcdefg" ".txt"))
      (should (= 16 (file-attribute-size (file-attributes a))))
      (should (= 17 (file-attribute-size (file-attributes b))))
      (unwind-protect
          (let ((large-file-warning-threshold 1)
                (imoogi-agent-max-payload-bytes 16))
            ;; (a) exactly at the cap: shown in another window, no question.
            (let* ((user (imoogi-agent-test--user-buffer))
                   (window (selected-window))
                   (frame (selected-frame)))
              (should (equal (car (imoogi-agent-test--receive-json
                                   (imoogi-agent-test--event-json
                                    "open-file" `((path . ,a)))))
                             "ok"))
              (let ((shown (imoogi-agent-test--file-window a)))
                (should (windowp shown))
                (should-not (eq shown window)))
              (imoogi-agent-test--assert-user-untouched user window frame))
            ;; (b) one byte over the cap.
            (should (equal (car (imoogi-agent-test--receive-json
                                 (imoogi-agent-test--event-json
                                  "open-file" `((path . ,b)))))
                           "error:payload-too-large")))
        (imoogi-agent-test--kill-visiting a b)
        (delete-other-windows)))
    (should (equal large-file-warning-threshold original))))

(provide 'imoogi-agent-test)
;;; agent-test.el ends here
