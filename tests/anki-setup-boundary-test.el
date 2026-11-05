;;; anki-setup-boundary-test.el --- Handshake rejection boundaries -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'imoogi-setup)

(defun imoogi-setup-boundary--with-response (body callback)
  "Call CALLBACK with a local HTTP response BODY, without network access."
  (let ((response (generate-new-buffer " *anki-handshake-boundary*")))
    (unwind-protect
        (progn
          (with-current-buffer response
            (setq-local url-http-response-status 200)
            (insert "HTTP/1.1 200 OK\nContent-Type: application/json\n\n" body))
          (cl-letf (((symbol-function 'url-retrieve-synchronously)
                     (lambda (_url _silent _cookies _timeout)
                       (should (equal url-request-method "POST"))
                       (should (equal (cdr (assoc "Content-Type" url-request-extra-headers))
                                      "application/json"))
                       (let ((request (json-parse-string url-request-data :object-type 'alist)))
                         (should (equal (alist-get 'action request) "requestPermission"))
                         (should (equal (alist-get 'version request) 6)))
                       response)))
            (funcall callback))
          (should-not (buffer-live-p response)))
      (when (buffer-live-p response) (kill-buffer response)))))

(ert-deftest imoogi-setup-boundary-accepts-granted-handshake ()
  (imoogi-setup-boundary--with-response
   "{\"result\":{\"permission\":\"granted\",\"version\":6},\"error\":null}"
   (lambda () (should (eq (imoogi-setup--ankiconnect-status "http://test.invalid") 'ok)))))

(ert-deftest imoogi-setup-boundary-rejects-denied-errors-and-malformed-results ()
  (dolist (body '("{\"result\":{\"permission\":\"denied\"},\"error\":null}"
                  "{\"result\":null,\"error\":\"denied\"}"
                  "{\"result\":{\"permission\":\"granted\"},\"error\":\"failure\"}"
                  "{\"result\":{\"permission\":\"granted\"},\"error\":false}"
                  "{\"result\":false,\"error\":null}"
                  "{\"result\":[],\"error\":null}"
                  "{\"result\":{},\"error\":null}"
                  "{\"result\":\"granted\",\"error\":null}"
                  "{\"result\":{\"permission\":true},\"error\":null}"
                  "{\"error\":null}" "[]" "42" "null" "not json"))
    (ert-info ((format "Response: %s" body))
      (imoogi-setup-boundary--with-response
       body
       (lambda ()
         (should (eq (imoogi-setup--ankiconnect-status "http://test.invalid")
                     'invalid-response)))))))

(ert-deftest imoogi-setup-boundary-rejected-handshake-preserves-config ()
  (dolist (body '("{\"result\":{\"permission\":\"denied\"},\"error\":null}"
                  "{\"result\":{\"permission\":\"granted\"},\"error\":\"failure\"}"))
    (let ((config (make-temp-file "imoogi-handshake-config-"))
          (imoogi-anki-binary-path (executable-find "true"))
          (imoogi-sync-root "original-root")
          (imoogi-default-deck "original-deck"))
      (unwind-protect
          (progn
            (with-temp-file config (insert "original-config"))
            (imoogi-setup-boundary--with-response
             body
             (lambda ()
               (cl-letf (((symbol-function 'imoogi-setup--install-step)
                          (lambda (&rest _) (ert-fail "Install ran after rejected handshake")))
                         ((symbol-function 'imoogi-setup--migrate-step)
                          (lambda (&rest _) (ert-fail "Migration ran after rejected handshake"))))
                 (should (equal (imoogi-anki-setup "/tmp/new-root" "new-deck" config)
                                (imoogi-error-message "ankiconnect_missing"))))))
            (should (equal imoogi-sync-root "original-root"))
            (should (equal imoogi-default-deck "original-deck"))
            (should (equal (with-temp-buffer (insert-file-contents config) (buffer-string))
                           "original-config")))
        (delete-file config)))))

(provide 'anki-setup-boundary-test)
