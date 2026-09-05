;;; imoogi-setup-test.el --- Tests for imoogi-setup.el -*- lexical-binding: t; -*-

;;; Commentary:

;; AC-003, AC-004 (acceptance.md SS D.1), plus the two Anki-unreachable
;; and AnkiConnect-invalid-response branches REQ-017 also names but no
;; numbered AC covers directly (same shape as REQ-021's uncovered edge
;; case, per plan.md SS I's precedent).
;;
;; A stub AnkiConnect handshake server is stood up with
;; `make-network-process' per the coordinator's documented interpretation
;; (progress.md SS F) -- no real Anki install exists in this sandbox.
;; AC-005's "nothing listening" and AC-006's "answers with an HTTP error"
;; conditions are both directly testable without a stub (an unreachable
;; port, and a stub that deliberately answers wrong); only the AC-003
;; success handshake needs a stub that speaks AnkiConnect correctly.

;;; Code:

(require 'ert)
(require 'imoogi-setup)
(require 'imoogi-config)

(defun imoogi-setup-test--free-port ()
  "Return a TCP port on 127.0.0.1 that is very likely free right now:
bind a server, read the OS-assigned port, then release it immediately."
  (let* ((probe (make-network-process
                  :name "imoogi-setup-test-probe" :server t
                  :host "127.0.0.1" :service t :family 'ipv4))
         (port (process-contact probe :service)))
    (delete-process probe)
    port))

(defun imoogi-setup-test--start-stub (respond-fn)
  "Start a one-shot local HTTP server on 127.0.0.1.  RESPOND-FN is
called with the raw request string and must return the full HTTP
response string (status line, headers, blank line, body) to send back.
Returns (PROCESS . PORT); caller deletes PROCESS when done."
  (let* ((proc (make-network-process
                :name "imoogi-setup-test-stub" :server t
                :host "127.0.0.1" :service t :family 'ipv4
                :filter (lambda (conn request)
                          (process-send-string conn (funcall respond-fn request))
                          (delete-process conn))))
         (port (process-contact proc :service)))
    (cons proc port)))

(defun imoogi-setup-test--ankiconnect-granted-response (_request)
  "An AnkiConnect handshake response granting permission -- the shape
`requestPermission' returns on success (internal/ankiconnect/client_test.go's
own fixture, mirrored here for the Elisp side's stub)."
  (let* ((body (json-serialize
                (list (cons 'result
                             (list (cons 'permission "granted")
                                   (cons 'requireApiKey :false)
                                   (cons 'version 6)))
                      (cons 'error :null)))))
    (concat "HTTP/1.1 200 OK\r\n"
            "Content-Type: application/json\r\n"
            (format "Content-Length: %d\r\n" (string-bytes body))
            "Connection: close\r\n\r\n"
            body)))

(defmacro imoogi-setup-test--with-granted-stub (url-var &rest body)
  "Bind URL-VAR to \"http://127.0.0.1:<port>\" of a running AnkiConnect
handshake stub for the duration of BODY."
  (declare (indent 1))
  `(let* ((started (imoogi-setup-test--start-stub
                     #'imoogi-setup-test--ankiconnect-granted-response))
          (,url-var (format "http://127.0.0.1:%d" (cdr started))))
     (unwind-protect
         (progn ,@body)
       (when (process-live-p (car started))
         (delete-process (car started))))))

(ert-deftest imoogi-setup-test-ankiconnect-status-ok-on-valid-handshake ()
  (imoogi-setup-test--with-granted-stub url
    (should (eq (imoogi-setup--ankiconnect-status url) 'ok))))

(ert-deftest imoogi-setup-test-ankiconnect-status-unreachable-when-nothing-listens ()
  "AC-005's Given, exercised directly against a closed port -- no stub
needed, the absence of a listener IS the test condition."
  (let ((url (format "http://127.0.0.1:%d" (imoogi-setup-test--free-port))))
    (should (eq (imoogi-setup--ankiconnect-status url) 'unreachable))))

(ert-deftest imoogi-setup-test-ankiconnect-status-invalid-response-on-http-error ()
  "AC-006's Given, exercised directly against a stub answering with a
plain HTTP error -- no AnkiConnect-shaped body, no stub interpretation
needed."
  (let* ((started (imoogi-setup-test--start-stub
                    (lambda (_req) "HTTP/1.1 500 Internal Server Error\r\n\r\n")))
         (url (format "http://127.0.0.1:%d" (cdr started))))
    (unwind-protect
        (should (eq (imoogi-setup--ankiconnect-status url) 'invalid-response))
      (when (process-live-p (car started))
        (delete-process (car started))))))

;; --- AC-003 -----------------------------------------------------------

(ert-deftest imoogi-setup-test-ac003-setup-checks-prompts-and-writes ()
  "AC-003: given a built binary and a running Anki with AnkiConnect
responding, invoking `imoogi-anki-setup' with the sync root and
default deck reports both checks successful, and the config file
records both values -- the user having edited no file by hand."
  (imoogi-setup-test--with-granted-stub anki-url
    (let* ((config-file (make-temp-file "imoogi-setup-test" nil ".json"))
           (imoogi-anki-connect-url anki-url)
           (imoogi-binary-path (executable-find "true"))
           (imoogi-sync-root nil)
           (imoogi-default-deck "Default"))
      (delete-file config-file)
      (unwind-protect
          (progn
            (should imoogi-binary-path)
            (let ((report (imoogi-anki-setup "/tmp/some-root" "TestDeck" config-file)))
              (should (string-match-p "binary" report))
              (should (string-match-p "AnkiConnect" report))
              (should (string-match-p "success\\|ok\\|complete" report)))
            (should (file-exists-p config-file))
            (let* ((parsed (with-temp-buffer
                              (insert-file-contents config-file)
                              (json-parse-string (buffer-string) :object-type 'alist))))
              (should (equal (cdr (assq 'sync_root parsed)) "/tmp/some-root"))
              (should (equal (cdr (assq 'default_deck parsed)) "TestDeck")))
            (should (equal imoogi-sync-root "/tmp/some-root"))
            (should (equal imoogi-default-deck "TestDeck")))
        (when (file-exists-p config-file) (delete-file config-file))))))

;; --- AC-004 -----------------------------------------------------------

(ert-deftest imoogi-setup-test-ac004-missing-binary-is-named-not-signalled ()
  "AC-004: given `imoogi-binary-path' pointing at a nonexistent path,
invoking `imoogi-anki-setup' reports that the binary was not found and
references the README build step, with no Emacs backtrace and no raw
`file-missing' signal text."
  (let* ((config-file (make-temp-file "imoogi-setup-test" nil ".json"))
         (imoogi-binary-path "/definitely/does/not/exist/imoogi")
         (imoogi-sync-root nil)
         (imoogi-default-deck "Default"))
    (delete-file config-file)
    (unwind-protect
        (let ((report (imoogi-anki-setup "/tmp/root" "Deck" config-file)))
          (should (string-match-p "not found\\|binary" report))
          (should (string-match-p "README\\|build" report))
          (should-not (string-match-p "file-missing" report))
          (should-not (string-match-p "Debugger entered" report))
          ;; and no config file was written on a failed precondition
          (should-not (file-exists-p config-file)))
      (when (file-exists-p config-file) (delete-file config-file)))))

(provide 'imoogi-setup-test)
;;; imoogi-setup-test.el ends here
