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
                              'openai-chat "company")
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
          (should (equal imoogi-gptel-models '(model-a))))
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
           (imoogi-gptel-setup "http://localhost:4000/path"
                               '(model) 'model nil file)
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
  (let ((imoogi-gptel-provider 'litellm)
        (imoogi-gptel-gateway-url "http://gateway.internal:4000")
        (auth-sources nil)
        (search-count 0))
    (cl-letf (((symbol-function 'auth-source-search)
               (lambda (&rest args)
                 (if (plist-get args :create)
                     (list (list :secret (lambda () "new-key")
                                 :save-function #'ignore))
                   (setq search-count (1+ search-count))
                   nil)))
              ((symbol-function 'auth-source-forget-all-cached) #'ignore))
      (should-error (imoogi-gptel-store-key) :type 'user-error))))

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
