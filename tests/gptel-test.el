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
         (imoogi-gptel-litellm-profiles nil)
         (imoogi-gptel-active-profile nil)
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

(ert-deftest imoogi-gptel-persists-and-switches-multiple-litellm-profiles ()
  (let* ((directory (make-temp-file "imoogi-gptel-profiles" t))
         (file (expand-file-name "config.json" directory))
         (imoogi-gptel-config-file file)
         (imoogi-gptel-litellm-profiles nil)
         (imoogi-gptel-active-profile nil)
         (imoogi-gptel-backend nil))
    (unwind-protect
        (progn
          (imoogi-gptel-setup "https://gateway-a.example.test"
                              '(model-a) 'model-a nil file 'litellm
                              'openai-chat "company" "/catalog/models")
          (imoogi-gptel-setup "https://gateway-b.example.test"
                              '(model-b) 'model-b nil file 'litellm
                              'openai-chat "personal")
          (should (= (length imoogi-gptel-litellm-profiles) 2))
          (should (equal imoogi-gptel-active-profile "personal"))
          (setq imoogi-gptel-litellm-profiles nil
                imoogi-gptel-active-profile nil)
          (should (imoogi-gptel--read-config file))
          (should (= (length imoogi-gptel-litellm-profiles) 2))
          (imoogi-gptel-switch-litellm-profile "company")
          (should (equal imoogi-gptel-active-profile "company"))
          (should (equal imoogi-gptel-gateway-url
                         "https://gateway-a.example.test"))
          (should (equal imoogi-gptel-models '(model-a)))
          (should (equal imoogi-gptel-models-endpoint "/catalog/models")))
      (delete-directory directory t))))

(ert-deftest imoogi-gptel-updating-profile-does-not-duplicate-it ()
  (let ((imoogi-gptel-litellm-profiles nil))
    (imoogi-gptel--upsert-litellm-profile
     (imoogi-gptel--profile-record "company" "https://old.example.test"
                                   '(old) 'old 'openai-chat
                                   "/v1/chat/completions"))
    (imoogi-gptel--upsert-litellm-profile
     (imoogi-gptel--profile-record "company" "https://new.example.test"
                                   '(new) 'new 'openai-chat
                                   "/v1/chat/completions"))
    (should (= (length imoogi-gptel-litellm-profiles) 1))
    (should (equal
             (alist-get 'gateway_url (car imoogi-gptel-litellm-profiles))
             "https://new.example.test"))))

(ert-deftest imoogi-gptel-active-profile-overrides-stale-top-level-fields ()
  (let* ((directory (make-temp-file "imoogi-gptel-manual-config" t))
         (file (expand-file-name "config.json" directory))
         (imoogi-gptel-litellm-profiles nil)
         (imoogi-gptel-active-profile nil))
    (unwind-protect
        (progn
          (with-temp-file file
            (insert "{\"gateway_url\":\"https://stale.example.test\",")
            (insert "\"provider\":\"litellm\",\"api_protocol\":\"openai-chat\",")
            (insert "\"endpoint\":\"/v1/chat/completions\",")
            (insert "\"models_endpoint\":\"/v1/models\",")
            (insert "\"models\":[\"stale\"],\"default_model\":\"stale\",")
            (insert "\"active_profile\":\"custom\",\"litellm_profiles\":[{")
            (insert "\"name\":\"custom\",\"gateway_url\":\"https://gateway.example.test/prefix\",")
            (insert "\"api_protocol\":\"openai-chat\",")
            (insert "\"endpoint\":\"/chat\",\"models_endpoint\":\"/catalog\",")
            (insert "\"models\":[\"custom-model\"],")
            (insert "\"default_model\":\"custom-model\"}]}"))
          (should (imoogi-gptel--read-config file))
          (should (equal imoogi-gptel-gateway-url
                         "https://gateway.example.test/prefix"))
          (should (equal imoogi-gptel-endpoint "/chat"))
          (should (equal imoogi-gptel-models-endpoint "/catalog"))
          (should (eq imoogi-gptel-default-model 'custom-model)))
      (delete-directory directory t))))

(ert-deftest imoogi-gptel-open-config-requires-an-existing-file ()
  (let ((imoogi-gptel-config-file
         (expand-file-name "missing.json" (make-temp-file "imoogi-gptel" t))))
    (should-error (imoogi-gptel-open-config) :type 'user-error)))

(ert-deftest imoogi-gptel-new-profile-rejects-an-existing-name ()
  (let ((imoogi-gptel-litellm-profiles
         (list (imoogi-gptel--profile-record
                "company" "https://gateway.example.test" '(model) 'model
                'openai-chat "/v1/chat/completions"))))
    (cl-letf (((symbol-function 'read-string) (lambda (&rest _) "company")))
      (should-error (imoogi-gptel--read-new-profile-name)
                    :type 'user-error))))

(ert-deftest imoogi-gptel-existing-profile-selection-uses-completion ()
  (let ((imoogi-gptel-active-profile "personal")
        (imoogi-gptel-litellm-profiles
         (list (imoogi-gptel--profile-record
                "company" "https://a.example.test" '(a) 'a
                'openai-chat "/v1/chat/completions")
               (imoogi-gptel--profile-record
                "personal" "https://b.example.test" '(b) 'b
                'openai-chat "/v1/chat/completions")))
        captured)
    (cl-letf (((symbol-function 'completing-read)
               (lambda (prompt collection &rest args)
                 (setq captured (list prompt collection args))
                 "company")))
      (should (equal (imoogi-gptel--read-existing-profile-name) "company"))
      (should (equal (nth 1 captured) '("company" "personal")))
      (should (eq (nth 1 (nth 2 captured)) t))
      (should (equal (nth 4 (nth 2 captured)) "personal")))))

(ert-deftest imoogi-gptel-litellm-reads-exact-base-and-chat-endpoint ()
  (let ((imoogi-gptel-litellm-profiles nil)
        (answers '("company"
                   "https://llm-gateway.example.test/custom"
                   "/v1/chat/completions"
                   "/internal/models")))
    (cl-letf (((symbol-function 'read-string)
               (lambda (&rest _) (pop answers)))
              ((symbol-function 'y-or-n-p) (lambda (&rest _) nil))
              ((symbol-function 'imoogi-gptel--fetch-models)
               (lambda (gateway &optional endpoint models-endpoint)
                 (should (equal gateway
                                "https://llm-gateway.example.test/custom"))
                 (should (equal endpoint "/v1/chat/completions"))
                 (should (equal models-endpoint "/internal/models"))
                 '(gateway-model)))
              ((symbol-function 'imoogi-gptel--read-api-protocol)
               (lambda (&optional _) 'openai-chat)))
      (should
       (equal
        (imoogi-gptel--read-litellm-profile-arguments)
        '("https://llm-gateway.example.test/custom"
          (gateway-model) gateway-model
          "/v1/chat/completions"
          nil litellm openai-chat "company" "/internal/models"))))))

(ert-deftest imoogi-gptel-litellm-base-path-is-added-to-backend-endpoint ()
  (let* ((file (make-temp-file "imoogi-gptel-test" nil ".json"))
         (imoogi-gptel-backend nil))
    (unwind-protect
        (progn
          (imoogi-gptel-setup "https://gateway.example.test/custom"
                              '(model) 'model "/v1/chat/completions" file
                              'litellm 'openai-chat "custom")
          (should (equal (gptel-backend-host imoogi-gptel-backend)
                         "gateway.example.test:443"))
          (should (equal (gptel-backend-endpoint imoogi-gptel-backend)
                         "/custom/v1/chat/completions")))
      (delete-file file))))

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

(ert-deftest imoogi-gptel-models-url-preserves-custom-chat-prefix ()
  (should
   (equal
    (imoogi-gptel--models-url
     "https://gateway.example.test/custom" nil "/catalog/models")
    "https://gateway.example.test/custom/catalog/models"))
  (should
   (equal
    (imoogi-gptel--models-url
     "https://gateway.example.test"
     "/custom/v1/chat/completions")
    "https://gateway.example.test/custom/v1/models"))
  (should
   (equal
    (imoogi-gptel--models-url
     "https://gateway.example.test/api/ai_interface"
     "/v1/chat/completions")
    "https://gateway.example.test/api/ai_interface/v1/models"))
  (should
   (equal
    (imoogi-gptel--models-url
     "https://gateway.example.test/api/ai_interface/v1"
     "/chat/completions")
    "https://gateway.example.test/api/ai_interface/v1/models"))
  (should
   (equal (imoogi-gptel--models-url "https://gateway.example.test")
          "https://gateway.example.test/v1/models")))

(ert-deftest imoogi-gptel-fetch-models-rejects-http-error ()
  (let* ((directory (make-temp-file "imoogi-gptel-http-error" t))
         (imoogi-gptel-log-file (expand-file-name "gptel.log" directory))
         (imoogi-gptel-gateway-url "https://gateway.example.test"))
    (unwind-protect
        (cl-letf (((symbol-function 'imoogi-gptel--api-key) (lambda () "key"))
                  ((symbol-function 'url-retrieve-synchronously)
                   (lambda (&rest _)
                     (let ((buffer (generate-new-buffer " *gptel models error*")))
                       (with-current-buffer buffer
                         (setq-local url-http-response-status 401)
                         (insert "HTTP/1.1 401 Unauthorized\r\n")
                         (insert "Content-Type: application/json\r\n\r\n")
                         (insert "{\"error\":\"invalid virtual key\",")
                         (insert "\"api_key\":\"must-not-leak\"}"))
                       buffer))))
          (should-error
           (imoogi-gptel--fetch-models "https://gateway.example.test")
           :type 'error)
          (with-temp-buffer
            (insert-file-contents imoogi-gptel-log-file)
            (should (search-forward "action=model-fetch-response" nil t))
            (should (search-forward ":status=401" nil t))
            (should (search-forward ":content-type=\"application/json\"" nil t))
            (should (search-forward "invalid virtual key" nil t))
            (should (search-forward "<redacted>" nil t))
            (should-not (search-forward "must-not-leak" nil t))))
      (delete-directory directory t))))

(ert-deftest imoogi-gptel-fetch-models-logs-network-failure-details ()
  (let* ((directory (make-temp-file "imoogi-gptel-network-error" t))
         (imoogi-gptel-log-file (expand-file-name "gptel.log" directory)))
    (unwind-protect
        (cl-letf (((symbol-function 'imoogi-gptel--api-key) (lambda () "key"))
                  ((symbol-function 'url-retrieve-synchronously)
                   (lambda (&rest _) (signal 'file-error '("TLS handshake failed")))))
          (should-error
           (imoogi-gptel--fetch-models "https://gateway.example.test")
           :type 'file-error)
          (with-temp-buffer
            (insert-file-contents imoogi-gptel-log-file)
            (should (search-forward "action=model-fetch-network-error" nil t))
            (should (search-forward ":error-type=file-error" nil t))
            (should (search-forward "TLS handshake failed" nil t))))
      (delete-directory directory t))))

(ert-deftest imoogi-gptel-fetch-models-logs-timeout-without-response ()
  (let* ((directory (make-temp-file "imoogi-gptel-no-response" t))
         (imoogi-gptel-log-file (expand-file-name "gptel.log" directory)))
    (unwind-protect
        (cl-letf (((symbol-function 'imoogi-gptel--api-key) (lambda () "key"))
                  ((symbol-function 'url-retrieve-synchronously) (lambda (&rest _) nil)))
          (should-error
           (imoogi-gptel--fetch-models "https://gateway.example.test")
           :type 'error)
          (with-temp-buffer
            (insert-file-contents imoogi-gptel-log-file)
            (should (search-forward "action=model-fetch-no-response" nil t))
            (should (search-forward ":timeout-seconds=10" nil t))))
      (delete-directory directory t))))

(ert-deftest imoogi-gptel-litellm-setup-defers-model-choice-to-gptel-menu ()
  (let ((imoogi-gptel-default-model 'model-b))
    (cl-letf (((symbol-function 'completing-read)
               (lambda (&rest _)
                 (ert-fail "LiteLLM setup must not prompt for a model"))))
      (should (eq (imoogi-gptel--setup-default-model
                   'litellm '(model-a model-b))
                  'model-b))
      (should (eq (imoogi-gptel--setup-default-model
                   'litellm '(model-c model-d))
                  'model-c)))))

(ert-deftest imoogi-gptel-litellm-discovery-failure-does-not-prompt-manually ()
  (cl-letf (((symbol-function 'y-or-n-p) (lambda (&rest _) nil))
            ((symbol-function 'imoogi-gptel--fetch-models)
             (lambda (&rest _) (error "HTTP 401")))
            ((symbol-function 'imoogi-gptel--read-models)
             (lambda (&rest _)
               (ert-fail "Discovery failure must not open manual model input"))))
    (should-error
     (imoogi-gptel--discover-litellm-models "https://gateway.example.test")
     :type 'user-error)))

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
           (imoogi-gptel-setup "http://localhost:4000"
                               '(model-a) 'model-b nil file)
           :type 'user-error))
      (delete-file file))))

(ert-deftest imoogi-gptel-splits-complete-openai-compatible-endpoint ()
  (should
   (equal
    (imoogi-gptel--split-api-url
     "https://sandbox.example.com/api/ai_interface/chat/completions")
    '("https://sandbox.example.com" .
      "/api/ai_interface/chat/completions")))
  (should
   (equal
    (imoogi-gptel--split-api-url
     "sandbox.example.com/api/ai_interface/chat/completions")
    '("https://sandbox.example.com" .
      "/api/ai_interface/chat/completions"))))

(ert-deftest imoogi-gptel-splits-base-only-openai-compatible-url ()
  (should (equal (imoogi-gptel--split-api-url "http://localhost:8000")
                 '("http://localhost:8000")))
  (should-error
   (imoogi-gptel--split-api-url "https://gateway.example.test/path?token=x")
   :type 'user-error))

(ert-deftest imoogi-gptel-complete-endpoint-skips-separate-endpoint-prompt ()
  (let ((answers '("sandbox.example.com/api/ai_interface/chat/completions"))
        (prompt-count 0))
    (cl-letf (((symbol-function 'read-string)
               (lambda (&rest _)
                 (setq prompt-count (1+ prompt-count))
                 (pop answers))))
      (should
       (equal (imoogi-gptel--read-openai-compatible-location)
              '("https://sandbox.example.com" .
                "/api/ai_interface/chat/completions")))
      (should (= prompt-count 1)))))

(ert-deftest imoogi-gptel-base-only-url-prompts-for-endpoint ()
  (let ((answers '("https://gateway.example.test"
                   "/custom/chat/completions"))
        (prompt-count 0))
    (cl-letf (((symbol-function 'read-string)
               (lambda (&rest _)
                 (setq prompt-count (1+ prompt-count))
                 (pop answers))))
      (should
       (equal (imoogi-gptel--read-openai-compatible-location)
              '("https://gateway.example.test" .
                "/custom/chat/completions")))
      (should (= prompt-count 2)))))

(ert-deftest imoogi-gptel-api-key-uses-gateway-auth-source-host ()
  (let ((imoogi-gptel-gateway-url "http://gateway.internal:4000")
        captured)
    (cl-letf (((symbol-function 'gptel-api-key-from-auth-source)
               (lambda (host user)
                 (setq captured (list host user))
                 "key")))
      (should (equal (imoogi-gptel--api-key) "key"))
      (should (equal captured '("gateway.internal:4000" "apikey"))))))

(ert-deftest imoogi-gptel-store-key-persists-without-second-confirmation ()
  (let ((imoogi-gptel-provider 'litellm)
        (imoogi-gptel-gateway-url "http://gateway.internal:4000")
        (auth-sources nil)
        (search-count 0)
        save-behavior
        cache-cleared)
    (cl-letf (((symbol-function 'auth-source-search)
               (lambda (&rest args)
                 (if (plist-get args :create)
                     (list
                      (list :secret (lambda () "new-key")
                            :save-function
                            (lambda ()
                              (setq save-behavior auth-source-save-behavior))))
                   (setq search-count (1+ search-count))
                   (when (= search-count 2)
                     (list (list :secret (lambda () "new-key")))))))
              ((symbol-function 'auth-source-forget-all-cached)
               (lambda () (setq cache-cleared t))))
      (should (imoogi-gptel-store-key))
      (should (eq save-behavior t))
      (should cache-cleared)
      (should (= search-count 2)))))

(ert-deftest imoogi-gptel-store-key-rejects-unpersisted-entry ()
  (let* ((directory (make-temp-file "imoogi-gptel-log" t))
         (imoogi-gptel-log-file (expand-file-name "gptel.log" directory))
         (imoogi-gptel-provider 'litellm)
        (imoogi-gptel-gateway-url "http://gateway.internal:4000")
        (auth-sources nil)
        (search-count 0))
    (unwind-protect
        (cl-letf (((symbol-function 'auth-source-search)
                   (lambda (&rest args)
                     (if (plist-get args :create)
                         (list (list :secret (lambda () "new-key")
                                     :save-function #'ignore))
                       (setq search-count (1+ search-count))
                       nil)))
                  ((symbol-function 'auth-source-forget-all-cached) #'ignore))
          (should-error (imoogi-gptel-store-key) :type 'user-error)
          (with-temp-buffer
            (insert-file-contents imoogi-gptel-log-file)
            (should (search-forward "action=auth-store-failed" nil t))
            (should (search-forward "stage=verify" nil t))))
      (delete-directory directory t))))

(ert-deftest imoogi-gptel-log-redacts-secrets-and-is-private ()
  (let* ((directory (make-temp-file "imoogi-gptel-log" t))
         (imoogi-gptel-log-file (expand-file-name "gptel.log" directory))
         (imoogi-gptel--action-id "test-action"))
    (unwind-protect
        (progn
          (imoogi-gptel--log 'test :host "gateway.example.test"
                             :api-key "never-write-this"
                             :access-token "nor-this")
          (with-temp-buffer
            (insert-file-contents imoogi-gptel-log-file)
            (should (search-forward "action-id=test-action" nil t))
            (should (search-forward "host=\"gateway.example.test\"" nil t))
            (should-not (search-forward "never-write-this" nil t))
            (should-not (search-forward "nor-this" nil t))
            (goto-char (point-min))
            (should (search-forward "<redacted>" nil t)))
          (should (= (logand (file-modes imoogi-gptel-log-file) #o777)
                     #o600)))
      (delete-directory directory t))))

(ert-deftest imoogi-gptel-store-key-supports-backend-without-save-function ()
  (let ((imoogi-gptel-provider 'litellm)
        (imoogi-gptel-gateway-url "http://gateway.internal:4000")
        (auth-sources nil)
        (search-count 0))
    (cl-letf (((symbol-function 'auth-source-search)
               (lambda (&rest args)
                 (if (plist-get args :create)
                     (list (list :secret (lambda () "new-key")))
                   (setq search-count (1+ search-count))
                   (when (= search-count 2)
                     (list (list :secret (lambda () "new-key")))))))
              ((symbol-function 'auth-source-forget-all-cached) #'ignore))
      (should (imoogi-gptel-store-key))
      (should (= search-count 2)))))

(ert-deftest imoogi-gptel-creates-missing-plain-auth-source-file ()
  (let* ((directory (make-temp-file "imoogi-gptel-auth-source" t))
         (file (expand-file-name ".authinfo" directory))
         (auth-sources (list file)))
    (unwind-protect
        (progn
          (should (equal (imoogi-gptel--ensure-auth-source-file) file))
          (should (file-exists-p file))
          (should (= (logand (file-modes file) #o777) #o600)))
      (delete-directory directory t))))

(ert-deftest imoogi-gptel-never-overwrites-existing-auth-source-file ()
  (let* ((directory (make-temp-file "imoogi-gptel-auth-source" t))
         (file (expand-file-name ".authinfo" directory))
         (auth-sources (list file)))
    (unwind-protect
        (progn
          (with-temp-file file (insert "existing credentials\n"))
          (should (equal (imoogi-gptel--ensure-auth-source-file) file))
          (with-temp-buffer
            (insert-file-contents file)
            (should (equal (buffer-string) "existing credentials\n"))))
      (delete-directory directory t))))

(ert-deftest imoogi-gptel-does-not-create-plaintext-gpg-file ()
  (let* ((directory (make-temp-file "imoogi-gptel-auth-source" t))
         (file (expand-file-name ".authinfo.gpg" directory))
         (auth-sources (list file)))
    (unwind-protect
        (progn
          (should-error (imoogi-gptel--ensure-auth-source-file)
                        :type 'user-error)
          (should-not (file-exists-p file)))
      (delete-directory directory t))))

(provide 'gptel-test)
;;; gptel-test.el ends here
