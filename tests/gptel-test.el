;;; gptel-test.el --- tests for LiteLLM gptel setup -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'json)
(require 'imoogi-gptel)

(ert-deftest imoogi-gptel-setup-round-trips-without-secret ()
  (let* ((directory (make-temp-file "imoogi-gptel-test" t))
         (file (expand-file-name "config.json" directory))
         (imoogi-gptel-config-file file)
         (imoogi-gptel-provider 'litellm)
         (imoogi-gptel-api-protocol 'openai-chat)
         (imoogi-gptel-gateway-url nil)
         (imoogi-gptel-endpoint "/v1/chat/completions")
         (imoogi-gptel-models nil)
         (imoogi-gptel-default-model nil)
         (imoogi-gptel-backend nil))
    (unwind-protect
        (progn
          (imoogi-gptel-setup "http://gateway.internal:4000"
                              '(claude-sonnet gpt-4.1)
                              'claude-sonnet nil file)
          (should (file-exists-p file))
          (should (= (logand (file-modes file) #o777) #o600))
          (with-temp-buffer
            (insert-file-contents file)
            (should-not (string-match-p "api[_-]?key\|secret" (buffer-string))))
          (setq imoogi-gptel-gateway-url nil
                imoogi-gptel-models nil
                imoogi-gptel-default-model nil)
          (should (imoogi-gptel--read-config file))
          (should (equal imoogi-gptel-gateway-url
                         "http://gateway.internal:4000"))
          (should (equal imoogi-gptel-models '(claude-sonnet gpt-4.1)))
          (should (eq imoogi-gptel-provider 'litellm))
          (should (eq imoogi-gptel-api-protocol 'openai-chat))
          (should (eq imoogi-gptel-default-model 'claude-sonnet)))
      (delete-directory directory t))))

(ert-deftest imoogi-gptel-setup-builds-openai-compatible-backend ()
  (let* ((file (make-temp-file "imoogi-gptel-test" nil ".json"))
         (imoogi-gptel-backend nil))
    (unwind-protect
        (progn
          (imoogi-gptel-setup "https://llm.example.test"
                              '(team-model) 'team-model
                              "/v1/chat/completions" file)
          (should (eq gptel-model 'team-model))
          (should (eq gptel-backend imoogi-gptel-backend))
          (should (equal (gptel-backend-host imoogi-gptel-backend)
                         "llm.example.test:443"))
          (should (equal (gptel-backend-protocol imoogi-gptel-backend)
                         "https")))
      (delete-file file))))

(ert-deftest imoogi-gptel-setup-builds-codex-oauth-backend ()
  (let* ((file (make-temp-file "imoogi-gptel-test" nil ".json"))
         (imoogi-gptel-backend nil))
    (unwind-protect
        (progn
          (imoogi-gptel-setup nil '(gpt-5.3-codex) 'gpt-5.3-codex
                              nil file 'codex)
          (should (gptel-openai-oauth-p imoogi-gptel-backend))
          (should (eq gptel-model 'gpt-5.3-codex))
          (should (eq imoogi-gptel-provider 'codex)))
      (delete-file file))))

(ert-deftest imoogi-gptel-setup-builds-claude-backend ()
  (let* ((file (make-temp-file "imoogi-gptel-test" nil ".json"))
         (imoogi-gptel-backend nil))
    (unwind-protect
        (progn
          (imoogi-gptel-setup "https://api.anthropic.com"
                              '(claude-sonnet-4-6) 'claude-sonnet-4-6
                              "/v1/messages" file 'claude)
          (should (gptel-anthropic-p imoogi-gptel-backend))
          (should (eq gptel-model 'claude-sonnet-4-6))
          (should (eq imoogi-gptel-provider 'claude)))
      (delete-file file))))

(ert-deftest imoogi-gptel-setup-builds-litellm-anthropic-backend ()
  (let* ((file (make-temp-file "imoogi-gptel-test" nil ".json"))
         (imoogi-gptel-backend nil))
    (unwind-protect
        (progn
          (imoogi-gptel-setup "https://llm.example.test"
                              '(claude-gateway) 'claude-gateway
                              "/v1/messages" file 'litellm
                              'anthropic-messages)
          (should (gptel-anthropic-p imoogi-gptel-backend))
          (should (eq imoogi-gptel-api-protocol 'anthropic-messages))
          (should (equal (gptel-backend-endpoint imoogi-gptel-backend)
                         "/v1/messages")))
      (delete-file file))))

(ert-deftest imoogi-gptel-fetch-models-reads-openai-model-list ()
  (let ((imoogi-gptel-gateway-url "https://gateway.example.test")
        captured-url
        captured-headers)
    (cl-letf (((symbol-function 'imoogi-gptel--api-key)
               (lambda () "virtual-key"))
              ((symbol-function 'url-retrieve-synchronously)
               (lambda (url &rest _)
                 (setq captured-url url
                       captured-headers url-request-extra-headers)
                 (let ((buffer (generate-new-buffer " *gptel models*")))
                   (with-current-buffer buffer
                     (setq-local url-http-response-status 200)
                     (insert "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n")
                     (insert "{\"data\":[{\"id\":\"claude-sonnet\"},")
                     (insert "{\"id\":\"gpt-5\"},{\"id\":\"claude-sonnet\"}]}"))
                   buffer))))
      (should (equal (imoogi-gptel--fetch-models
                      "https://gateway.example.test")
                     '(claude-sonnet gpt-5)))
      (should (equal captured-url
                     "https://gateway.example.test/v1/models"))
      (should (equal (cdr (assoc "Authorization" captured-headers))
                     "Bearer virtual-key")))))

(ert-deftest imoogi-gptel-fetch-models-rejects-http-error ()
  (let ((imoogi-gptel-gateway-url "https://gateway.example.test"))
    (cl-letf (((symbol-function 'imoogi-gptel--api-key) (lambda () "key"))
              ((symbol-function 'url-retrieve-synchronously)
               (lambda (&rest _)
                 (let ((buffer (generate-new-buffer " *gptel models error*")))
                   (with-current-buffer buffer
                     (setq-local url-http-response-status 401)
                     (insert "HTTP/1.1 401 Unauthorized\r\n\r\n{}"))
                   buffer))))
      (should-error
       (imoogi-gptel--fetch-models "https://gateway.example.test")
       :type 'error))))

(ert-deftest imoogi-gptel-old-config-defaults-to-openai-chat ()
  (let ((file (make-temp-file "imoogi-gptel-old-config" nil ".json"))
        (imoogi-gptel-api-protocol 'anthropic-messages))
    (unwind-protect
        (progn
          (with-temp-file file
            (insert "{\"gateway_url\":\"http://gateway.internal:4000\",")
            (insert "\"provider\":\"litellm\",")
            (insert "\"endpoint\":\"/v1/chat/completions\",")
            (insert "\"models\":[\"model-a\"],\"default_model\":\"model-a\"}"))
          (should (imoogi-gptel--read-config file))
          (should (eq imoogi-gptel-api-protocol 'openai-chat)))
      (delete-file file))))

(ert-deftest imoogi-gptel-setup-rejects-invalid-input ()
  (let ((file (make-temp-file "imoogi-gptel-test" nil ".json")))
    (unwind-protect
        (progn
          (should-error
           (imoogi-gptel-setup "localhost:4000" '(model) 'model nil file)
           :type 'user-error)
          (should-error
           (imoogi-gptel-setup "http://localhost:4000/path"
                               '(model) 'model nil file)
           :type 'user-error)
          (should-error
           (imoogi-gptel-setup "http://localhost:4000"
                               '(model-a) 'model-b nil file)
           :type 'user-error))
      (delete-file file))))

(ert-deftest imoogi-gptel-api-key-uses-gateway-auth-source-host ()
  (let ((imoogi-gptel-gateway-url "http://gateway.internal:4000")
        captured)
    (cl-letf (((symbol-function 'gptel-api-key-from-auth-source)
               (lambda (host user)
                 (setq captured (list host user))
                 "key")))
      (should (equal (imoogi-gptel--api-key) "key"))
      (should (equal captured '("gateway.internal:4000" "apikey"))))))

(provide 'gptel-test)
;;; gptel-test.el ends here
