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

(provide 'imoogi-agent-test)
;;; agent-test.el ends here
