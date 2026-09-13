;;; 27-gptel.el --- gptel with a LiteLLM Gateway -*- lexical-binding: t; -*-

;;; Code:

(imoogi-require "27-gptel" 'gptel 'gptel-transient 'gptel-anthropic
                'gptel-openai-oauth 'auth-source 'json 'seq 'url 'url-http
                'url-parse 'subr-x)

;; `transient-define-prefix' is expanded inside `with-eval-after-load' below.
;; Make the macro available while compile-angel byte-compiles this module;
;; otherwise the compiled form evaluates the prefix name as a variable.
(eval-when-compile (require 'transient))

(require 'auth-source)
(require 'json)
(require 'seq)
(require 'subr-x)
(require 'url-parse)
(require 'url)
(require 'url-http)
(require 'gptel)
(require 'gptel-anthropic)
(require 'gptel-openai-oauth)

(defgroup imoogi-gptel nil
  "Guided gptel provider setup for imoogi-emacs."
  :group 'gptel)

(defcustom imoogi-gptel-config-file
  (expand-file-name "imoogi-gptel.json" user-emacs-directory)
  "File containing the non-secret LiteLLM Gateway configuration."
  :type 'file)

(defvar imoogi-gptel-gateway-url nil
  "Configured API base URL, when the provider needs one.")

(defvar imoogi-gptel-provider 'litellm
  "Configured provider: `litellm', `codex', `claude', or `openai-compatible'.")

(defvar imoogi-gptel-endpoint "/v1/chat/completions"
  "Configured LiteLLM chat completion endpoint.")

(defvar imoogi-gptel-api-protocol 'openai-chat
  "Configured HTTP API protocol: `openai-chat' or `anthropic-messages'.")

(defvar imoogi-gptel-models nil
  "Model aliases exposed by the configured LiteLLM Gateway.")

(defvar imoogi-gptel-default-model nil
  "Default LiteLLM model alias used by gptel.")

(defvar imoogi-gptel-backend nil
  "The gptel backend created for the configured LiteLLM Gateway.")

(defun imoogi-gptel--url-components (gateway-url)
  "Validate GATEWAY-URL and return its protocol and host.
GATEWAY-URL is a base URL and therefore may not contain a path."
  (let* ((url (url-generic-parse-url gateway-url))
         (protocol (url-type url))
         (hostname (url-host url))
         (port (url-port url))
         (path (url-filename url)))
    (unless (member protocol '("http" "https"))
      (user-error "LiteLLM 주소는 http:// 또는 https://로 시작해야 합니다"))
    (unless (and hostname (not (string-empty-p hostname)))
      (user-error "LiteLLM 주소에 호스트가 없습니다"))
    (unless (member path '(nil "" "/"))
      (user-error "Gateway 주소에는 경로를 넣지 말고 endpoint를 별도로 지정하세요"))
    (cons protocol
          (if port (format "%s:%d" hostname port) hostname))))

(defun imoogi-gptel--normalize-models (models)
  "Return MODELS as a non-empty list of model symbols."
  (let ((normalized
         (delete-dups
          (delq nil
                (mapcar (lambda (model)
                          (let ((name (string-trim
                                       (if (symbolp model)
                                           (symbol-name model)
                                         model))))
                            (unless (string-empty-p name) (intern name))))
                        models)))))
    (unless normalized
      (user-error "LiteLLM model alias를 하나 이상 입력하세요"))
    normalized))

(defun imoogi-gptel--config-data ()
  "Return the current non-secret configuration as an alist."
  `((gateway_url . ,imoogi-gptel-gateway-url)
    (provider . ,(symbol-name imoogi-gptel-provider))
    (api_protocol . ,(symbol-name imoogi-gptel-api-protocol))
    (endpoint . ,imoogi-gptel-endpoint)
    ;; `json-serialize' uses vectors for JSON arrays.  A Lisp list is
    ;; otherwise interpreted as an object/alist and model strings fail.
    (models . ,(vconcat (mapcar #'symbol-name imoogi-gptel-models)))
    (default_model . ,(symbol-name imoogi-gptel-default-model))))

(defun imoogi-gptel--write-config (file)
  "Atomically write the current non-secret configuration to FILE."
  (make-directory (file-name-directory file) t)
  (let ((temporary (make-temp-file
                    (expand-file-name ".imoogi-gptel-" (file-name-directory file))
                    nil ".json")))
    (unwind-protect
        (progn
          (with-temp-file temporary
            (insert (json-serialize (imoogi-gptel--config-data)
                                    :null-object nil :false-object :json-false))
            (insert "\n"))
          (set-file-modes temporary #o600)
          (rename-file temporary file t))
      (when (file-exists-p temporary)
        (delete-file temporary)))))

(defun imoogi-gptel--read-config (&optional file)
  "Read non-secret gptel settings from FILE and return non-nil on success."
  (let ((target (or file imoogi-gptel-config-file)))
    (when (file-readable-p target)
      (with-temp-buffer
        (insert-file-contents target)
        (let* ((json-object-type 'alist)
               (json-array-type 'list)
               (data (json-read))
               (provider-name (or (alist-get 'provider data) "litellm"))
               (provider (intern provider-name))
               (api-protocol-name (alist-get 'api_protocol data))
               (api-protocol
                (if api-protocol-name
                    (intern api-protocol-name)
                  (if (eq provider 'claude)
                      'anthropic-messages
                    'openai-chat)))
               (gateway-url (alist-get 'gateway_url data))
               (endpoint (alist-get 'endpoint data))
               (models (imoogi-gptel--normalize-models
                        (alist-get 'models data)))
               (default-model (intern (alist-get 'default_model data))))
          (unless (memq provider '(litellm codex claude openai-compatible))
            (error "지원하지 않는 gptel provider입니다: %s" provider))
          (unless (memq api-protocol '(openai-chat anthropic-messages))
            (error "지원하지 않는 API protocol입니다: %s" api-protocol))
          (unless (eq provider 'codex)
            (imoogi-gptel--url-components gateway-url))
          (unless (memq default-model models)
            (error "기본 model %s이 models 목록에 없습니다" default-model))
          (setq imoogi-gptel-provider provider
                imoogi-gptel-gateway-url gateway-url
                imoogi-gptel-api-protocol api-protocol
                imoogi-gptel-endpoint endpoint
                imoogi-gptel-models models
                imoogi-gptel-default-model default-model)
          t)))))

(defun imoogi-gptel--auth-host ()
  "Return the auth-source host for the current Gateway configuration."
  (cdr (imoogi-gptel--url-components imoogi-gptel-gateway-url)))

(defun imoogi-gptel--api-key ()
  "Read the LiteLLM virtual key from auth-source."
  (gptel-api-key-from-auth-source (imoogi-gptel--auth-host) "apikey"))

(defun imoogi-gptel--models-url (gateway-url)
  "Return the OpenAI-compatible models URL for GATEWAY-URL."
  (concat (string-remove-suffix "/" gateway-url) "/v1/models"))

(defun imoogi-gptel--fetch-models (gateway-url)
  "Fetch model identifiers from GATEWAY-URL's `/v1/models' endpoint.
Authenticate with the key stored in `auth-source'.  Return model symbols in
server order, or signal an error that the interactive setup can recover from."
  (imoogi-gptel--url-components gateway-url)
  (let* ((key (imoogi-gptel--api-key))
         (url-request-method "GET")
         (url-request-extra-headers
          `(("Accept" . "application/json")
            ("Authorization" . ,(and (stringp key)
                                      (concat "Bearer " key)))))
         buffer)
    (unless (and (stringp key) (not (string-empty-p key)))
      (error "Gateway API key를 auth-source에서 찾을 수 없습니다"))
    (setq buffer (url-retrieve-synchronously
                  (imoogi-gptel--models-url gateway-url) t t 10))
    (unless buffer
      (error "Gateway의 /v1/models에 연결할 수 없습니다"))
    (unwind-protect
        (with-current-buffer buffer
          (unless (and (boundp 'url-http-response-status)
                       (= url-http-response-status 200))
            (error "/v1/models 응답 실패: HTTP %s"
                   (if (boundp 'url-http-response-status)
                       url-http-response-status "unknown")))
          (goto-char (point-min))
          (unless (re-search-forward "\r?\n\r?\n" nil t)
            (error "/v1/models 응답에 HTTP 본문이 없습니다"))
          (let* ((json-object-type 'alist)
                 (json-array-type 'list)
                 (payload (json-read))
                 (models
                  (mapcar (lambda (item) (alist-get 'id item))
                          (alist-get 'data payload))))
            (imoogi-gptel--normalize-models models)))
      (kill-buffer buffer))))

(defun imoogi-gptel--configure-backend ()
  "Create and select the gptel backend from the loaded configuration."
  (when (and imoogi-gptel-models imoogi-gptel-default-model)
    (setq imoogi-gptel-backend
          (pcase imoogi-gptel-provider
            ('codex
             (gptel-make-openai-oauth "Codex"
               :stream t :models imoogi-gptel-models))
            ((or 'claude
                 (and 'litellm
                      (guard (eq imoogi-gptel-api-protocol
                                 'anthropic-messages))))
             (pcase-let ((`(,protocol . ,host)
                          (imoogi-gptel--url-components imoogi-gptel-gateway-url)))
               (gptel-make-anthropic
                   (if (eq imoogi-gptel-provider 'litellm)
                       "LiteLLM Messages" "Claude")
                 :host host :protocol protocol :endpoint imoogi-gptel-endpoint
                 :stream t :key #'imoogi-gptel--api-key
                 :models imoogi-gptel-models)))
            ((or 'litellm 'openai-compatible)
             (pcase-let ((`(,protocol . ,host)
                          (imoogi-gptel--url-components imoogi-gptel-gateway-url)))
               (gptel-make-openai
                   (if (eq imoogi-gptel-provider 'litellm)
                       "LiteLLM" "OpenAI-compatible")
                 :host host :protocol protocol :endpoint imoogi-gptel-endpoint
                 :stream t :key #'imoogi-gptel--api-key
                 :models imoogi-gptel-models))))
          gptel-backend imoogi-gptel-backend
          gptel-model imoogi-gptel-default-model)))

(defun imoogi-gptel-store-key ()
  "Create the LiteLLM virtual-key entry through Emacs auth-source.
The secret is requested and saved by the selected auth-source backend."
  (interactive)
  (when (eq imoogi-gptel-provider 'codex)
    (user-error "Codex는 API key 대신 M-x gptel-openai-oauth-login으로 인증합니다"))
  (unless imoogi-gptel-gateway-url
    (user-error "먼저 M-x imoogi-gptel-setup을 실행하세요"))
  (let* ((host (imoogi-gptel--auth-host))
         (existing (car (auth-source-search
                         :host host :user "apikey" :max 1
                         :require '(:secret)))))
    (if existing
        (message "API key가 auth-source에 이미 등록되어 있습니다: %s" host)
      (let* ((entry (car (auth-source-search
                          :host host :user "apikey" :max 1
                          :create '(:secret))))
             (save (and entry (plist-get entry :save-function))))
        (unless entry
          (user-error "사용 가능한 auth-source 저장소가 없습니다"))
        (when save (funcall save))
        (auth-source-forget-all-cached)
        (message "API key를 auth-source에 등록했습니다: %s" host)))))

(defun imoogi-gptel--model-symbols (models)
  "Return just the model symbols from gptel MODELS specifications."
  (mapcar (lambda (model) (if (consp model) (car model) model)) models))

(defun imoogi-gptel--read-models (initial)
  "Prompt for comma-separated models using INITIAL as the default."
  (imoogi-gptel--normalize-models
   (split-string (read-string "Model names (쉼표 구분): "
                              (mapconcat #'symbol-name initial ","))
                 "," t "[[:space:]]*")))

(defun imoogi-gptel--read-api-protocol ()
  "Prompt for the HTTP request protocol and return its symbol."
  (let ((choices '(("자동 / OpenAI Chat (권장)" . openai-chat)
                   ("Anthropic Messages" . anthropic-messages))))
    (alist-get (completing-read "API 형식: " choices nil t nil nil
                                "자동 / OpenAI Chat (권장)")
               choices nil nil #'string=)))

(defun imoogi-gptel--discover-litellm-models (gateway)
  "Offer key storage, then discover models from LiteLLM GATEWAY.
Fall back to manual model input when discovery is unavailable."
  (setq imoogi-gptel-provider 'litellm
        imoogi-gptel-gateway-url gateway)
  (when (y-or-n-p "API key를 auth-source에 등록하거나 확인할까요? ")
    (imoogi-gptel-store-key))
  (condition-case err
      (let ((models (imoogi-gptel--fetch-models gateway)))
        (message "LiteLLM에서 model %d개를 불러왔습니다" (length models))
        models)
    (error
     (message "모델 자동 조회 실패, 수동 입력으로 전환합니다: %s"
              (error-message-string err))
     (imoogi-gptel--read-models imoogi-gptel-models))))

(defun imoogi-gptel--read-setup-arguments ()
  "Read provider-specific arguments for `imoogi-gptel-setup'."
  (let* ((choices '(("LiteLLM Gateway" . litellm)
                    ("Codex / ChatGPT Plus·Pro OAuth" . codex)
                    ("Claude / Anthropic API" . claude)
                    ("기타 OpenAI 호환 API" . openai-compatible)))
         (provider (alist-get
                    (completing-read "사용할 LLM 연결 방식: " choices nil t)
                    choices nil nil #'string=))
         (gateway
          (pcase provider
            ('codex nil)
            ('claude "https://api.anthropic.com")
            ('litellm (read-string "LiteLLM Gateway URL: "
                                   (or imoogi-gptel-gateway-url
                                       "http://localhost:4000")))
            ('openai-compatible
             (read-string "OpenAI 호환 API base URL: "
                          (or imoogi-gptel-gateway-url "http://localhost:8000")))))
         (_ (unless (eq provider 'codex)
              (imoogi-gptel--url-components gateway)))
         (defaults
          (pcase provider
            ('codex '(gpt-5.3-codex gpt-5.3-codex-spark gpt-5.4-mini
                      gpt-5.4 gpt-5.5 gpt-5.6-sol gpt-5.6-terra
                      gpt-5.6-luna))
            ('claude (seq-take (imoogi-gptel--model-symbols
                                gptel--anthropic-models) 4))
            (_ imoogi-gptel-models)))
         (models (if (eq provider 'litellm)
                     (imoogi-gptel--discover-litellm-models gateway)
                   (imoogi-gptel--read-models defaults)))
         (default (intern
                   (completing-read "Default model: " models nil t nil nil
                                    (symbol-name (car models)))))
         (api-protocol
          (pcase provider
            ('claude 'anthropic-messages)
            ('codex 'openai-chat)
            (_ (imoogi-gptel--read-api-protocol))))
         (endpoint
          (pcase provider
            ('codex "/backend-api/codex/responses")
            ('claude "/v1/messages")
            (_ (if (eq api-protocol 'anthropic-messages)
                   "/v1/messages"
                 "/v1/chat/completions")))))
    (list gateway models default endpoint nil provider api-protocol)))

;;;###autoload
(defun imoogi-gptel-setup (gateway-url models default-model
                           &optional endpoint config-file provider api-protocol)
  "Configure gptel for PROVIDER with DEFAULT-MODEL from MODELS.
PROVIDER is one of `litellm', `codex', `claude', or `openai-compatible'.
GATEWAY-URL and ENDPOINT apply to HTTP API providers.  API-PROTOCOL selects
`openai-chat' or `anthropic-messages'.  Existing Lisp callers that omit
PROVIDER and API-PROTOCOL retain the original LiteLLM OpenAI Chat behavior.
CONFIG-FILE contains only non-secret settings; API keys are stored separately
with `auth-source'."
  (interactive
   (imoogi-gptel--read-setup-arguments))
  (let* ((provider-value (or provider 'litellm))
         (protocol-value
          (or api-protocol
              (if (eq provider-value 'claude)
                  'anthropic-messages
                'openai-chat)))
         (normalized (imoogi-gptel--normalize-models models))
         (default (if (symbolp default-model)
                      default-model
                    (intern default-model)))
         (endpoint-value
          (or endpoint
              (pcase provider-value
                ('codex "/backend-api/codex/responses")
                ('claude "/v1/messages")
                (_ "/v1/chat/completions"))))
         (target (or config-file imoogi-gptel-config-file)))
    (unless (memq provider-value '(litellm codex claude openai-compatible))
      (user-error "지원하지 않는 provider입니다: %s" provider-value))
    (unless (memq protocol-value '(openai-chat anthropic-messages))
      (user-error "지원하지 않는 API protocol입니다: %s" protocol-value))
    (unless (eq provider-value 'codex)
      (imoogi-gptel--url-components gateway-url))
    (unless (memq default normalized)
      (user-error "기본 model은 model aliases 목록에 있어야 합니다"))
    (unless (string-prefix-p "/" endpoint-value)
      (user-error "endpoint는 /로 시작해야 합니다"))
    (setq imoogi-gptel-provider provider-value
          imoogi-gptel-gateway-url gateway-url
          imoogi-gptel-api-protocol protocol-value
          imoogi-gptel-endpoint endpoint-value
          imoogi-gptel-models normalized
          imoogi-gptel-default-model default)
    (imoogi-gptel--write-config target)
    (imoogi-gptel--configure-backend)
    (when (and (called-interactively-p 'interactive)
               (not (memq provider-value '(codex litellm)))
               (y-or-n-p "API key를 auth-source에 등록할까요? "))
      (imoogi-gptel-store-key))
    (when (and (called-interactively-p 'interactive)
               (eq provider-value 'codex)
               (y-or-n-p "지금 OpenAI 계정으로 Codex OAuth 로그인을 할까요? "))
      (gptel-openai-oauth-login imoogi-gptel-backend))
    (message "gptel 설정 완료: %s / %s" provider-value default)
    target))

(defun imoogi-gptel--ensure-configured ()
  "Signal a helpful error unless the LiteLLM backend is configured."
  (unless imoogi-gptel-backend
    (user-error "M-x imoogi-gptel-setup으로 LLM 공급자를 먼저 설정하세요")))

(defun imoogi-gptel-chat ()
  "Open a gptel chat using the configured LiteLLM backend."
  (interactive)
  (imoogi-gptel--ensure-configured)
  (call-interactively #'gptel))

(defun imoogi-gptel-send ()
  "Send the current region or buffer prompt through LiteLLM."
  (interactive)
  (imoogi-gptel--ensure-configured)
  (call-interactively #'gptel-send))

(defun imoogi-gptel-menu ()
  "Open gptel's request menu after checking LiteLLM setup."
  (interactive)
  (imoogi-gptel--ensure-configured)
  (call-interactively #'gptel-menu))

(defun imoogi-gptel-setup-guide ()
  "Show the built-in LiteLLM and gptel setup guide."
  (interactive)
  (with-help-window "*imoogi gptel 설정*"
    (princ "gptel 공급자 설정\n\n")
    (princ "1. M-x imoogi-gptel-setup을 실행하고 연결 방식을 선택합니다.\n")
    (princ "2. LiteLLM은 Gateway URL과 key로 /v1/models를 조회합니다.\n")
    (princ "   조회된 모델 중 기본 모델을 고르고 API 형식을 선택합니다.\n")
    (princ "   조회 실패 시 model_name을 직접 입력할 수 있습니다.\n")
    (princ "3. Codex는 ChatGPT Plus/Pro OAuth를 사용하며 API key가 필요 없습니다.\n")
    (princ "4. Claude는 Anthropic 모델을 고르고 API key를 auth-source에 저장합니다.\n")
    (princ "5. 기타는 OpenAI 호환 base URL, endpoint, 모델 이름을 입력합니다.\n")
    (princ "6. C-c h i c로 채팅을 열거나 C-c h i s로 현재 내용을 전송합니다.\n\n")
    (princ (format "설정 파일: %s\n" imoogi-gptel-config-file))
    (princ (format "현재 provider: %s\n" imoogi-gptel-provider))
    (when imoogi-gptel-gateway-url
      (princ (format "현재 API 주소: %s\n" imoogi-gptel-gateway-url)))
    (when imoogi-gptel-models
      (princ (format "현재 models: %s\n"
                     (mapconcat #'symbol-name imoogi-gptel-models ", "))))))

(condition-case err
    (when (imoogi-gptel--read-config)
      (imoogi-gptel--configure-backend))
  (error
   (display-warning 'imoogi-gptel
                    (format "gptel 설정을 읽지 못했습니다: %s"
                            (error-message-string err))
                    :warning)))

(with-eval-after-load 'imoogi-transient
  (add-to-list 'imoogi-transient--purposes
               '(imoogi-gptel-transient . "gptel 공급자·채팅·전송·문맥을 관리합니다."))
  (transient-define-prefix imoogi-gptel-transient ()
    "gptel and LiteLLM Gateway commands."
    :column-widths '(20 22)
    [["대화 ----------------"
      ("c" "채팅 열기" imoogi-gptel-chat)
      ("s" "현재 내용 전송" imoogi-gptel-send)
      ("m" "gptel 요청 메뉴" imoogi-gptel-menu)]
     ["문맥·설정 ------------"
      ("a" "영역·버퍼 문맥" gptel-add)
      ("f" "파일 문맥" gptel-add-file)
      ("S" "공급자 설정" imoogi-gptel-setup)
      ("k" "API key 등록" imoogi-gptel-store-key)
      ("h" "설정 가이드" imoogi-gptel-setup-guide)
      ("q" "종료" transient-quit-one)]])
  (transient-append-suffix 'imoogi-transient-master "g"
    '("i" "AI / gptel" imoogi-gptel-transient)))

(provide 'imoogi-gptel)
;;; 27-gptel.el ends here
