;;; imoogi-process-test.el --- Tests for imoogi-process.el -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'imoogi-process)

(ert-deftest imoogi-process-test-serialize-request-shape ()
  (let* ((config (list :default-deck "Inbox"
                        :anki-connect-url "http://127.0.0.1:8765"
                        :registry-path "/tmp/reg.json"
                        :sync-root "/tmp/root"
                        :exclude-patterns '("drafts/")
                        :scan-complete t))
         (census (list (list :note-id 1001 :source-path "a.org")))
         (entries (list (list :key "a.org::0" :note-id nil :note-type "Basic"
                               :source-path "a.org" :deck nil :tags nil
                               :title "T" :body "B")))
         (json (imoogi-process-serialize-request config census entries))
         (parsed (json-parse-string json :object-type 'alist :array-type 'list)))
    (should (= (cdr (assq 'protocol_version parsed)) imoogi-protocol-version))
    (let ((cfg (cdr (assq 'config parsed))))
      (should (equal (cdr (assq 'default_deck cfg)) "Inbox"))
      (should (eq (cdr (assq 'scan_complete cfg)) t)))
    (let ((cs (cdr (assq 'census parsed))))
      (should (= (length cs) 1))
      (should (= (cdr (assq 'note_id (car cs))) 1001)))
    (let ((es (cdr (assq 'entries parsed))))
      (should (= (length es) 1))
      ;; note_id null (add trigger) and deck null (absent-value trigger)
      ;; must serialize as JSON null, never be dropped from the object.
      (should (eq (cdr (assq 'note_id (car es))) :null))
      (should (eq (cdr (assq 'deck (car es))) :null))
      ;; an empty tag list serializes as [] -- an array, never an object.
      (should (equal (cdr (assq 'tags (car es))) nil)))))

(ert-deftest imoogi-process-test-empty-tags-serialize-as-array ()
  (let* ((json (imoogi-process-serialize-request
                (list :default-deck "D" :anki-connect-url "u"
                      :registry-path "r" :sync-root "s"
                      :exclude-patterns nil :scan-complete t)
                nil nil)))
    ;; census[], entries[], exclude_patterns[] must all render as "[]",
    ;; never "{}" -- json.el/native json-serialize ambiguity around nil.
    (should (string-match-p "\"census\":\\[\\]" json))
    (should (string-match-p "\"entries\":\\[\\]" json))
    (should (string-match-p "\"exclude_patterns\":\\[\\]" json))))

(ert-deftest imoogi-process-test-parse-response-round-trip ()
  (let* ((response-json
          "{\"protocol_version\":2,\"ok\":true,\"results\":[{\"key\":\"a.org::0\",\"action\":\"added\",\"note_id\":1001},{\"key\":null,\"action\":\"deleted\",\"note_id\":1005}],\"errors\":[{\"code\":\"cloze_marker_missing\",\"message\":\"no marker\",\"key\":\"b.org::0\"}]}")
         (parsed (imoogi-process-parse-response response-json)))
    (should (= (plist-get parsed :protocol-version) 2))
    (should (eq (plist-get parsed :ok) t))
    (should (= (length (plist-get parsed :results)) 2))
    (let ((added (car (plist-get parsed :results)))
          (deleted (cadr (plist-get parsed :results))))
      (should (equal (plist-get added :key) "a.org::0"))
      (should (equal (plist-get added :action) "added"))
      (should (= (plist-get added :note-id) 1001))
      ;; a deleted result's key is null on the wire and must parse to nil,
      ;; not the string "null" or the keyword :null.
      (should (null (plist-get deleted :key))))
    (let ((err (car (plist-get parsed :errors))))
      (should (equal (plist-get err :code) "cloze_marker_missing"))
      (should (equal (plist-get err :key) "b.org::0")))))

(ert-deftest imoogi-process-test-parse-response-malformed-returns-nil ()
  (should (null (imoogi-process-parse-response "not json at all"))))

(ert-deftest imoogi-process-test-run-with-stub ()
  (let ((imoogi-process-runner
         (lambda (_binary _request-json)
           (cons 0 "{\"protocol_version\":2,\"ok\":true,\"results\":[],\"errors\":[]}"))))
    (let ((response (imoogi-process-run
                      "irrelevant-binary"
                      (list :default-deck "D" :anki-connect-url "u"
                            :registry-path "r" :sync-root "s"
                            :exclude-patterns nil :scan-complete t)
                      nil nil)))
      (should (eq (plist-get response :ok) t)))))

(ert-deftest imoogi-process-test-run-nonzero-exit-returns-nil ()
  (let ((imoogi-process-runner
         (lambda (_binary _request-json) (cons 1 "irrelevant stderr-ish text"))))
    (should (null (imoogi-process-run
                   "irrelevant-binary"
                   (list :default-deck "D" :anki-connect-url "u"
                         :registry-path "r" :sync-root "s"
                         :exclude-patterns nil :scan-complete t)
                   nil nil)))))

(ert-deftest imoogi-process-test-call-binary-passes-persistent-log-path ()
  (let ((imoogi-anki-log-file "/tmp/imoogi-anki-test.jsonl")
        observed)
    (cl-letf (((symbol-function 'call-process-region)
               (lambda (&rest _args)
                 (setq observed (getenv "IMOOGI_ANKI_LOG"))
                 (erase-buffer)
                 (insert "{\"protocol_version\":2,\"ok\":true,\"results\":[],\"errors\":[]}")
                 0)))
      (imoogi-process--call-binary "fake-imoogi-anki" "{}"))
    (should (equal observed imoogi-anki-log-file))))

;; --- SPEC-ANKICARD-002: the three card-option keys on the wire.

;; AC-OPT-004a: the keys are ALWAYS emitted. A dropped key and a null value
;; are not interchangeable on this wire, so an unresolved option must arrive
;; as an explicit null rather than as a missing key.
(ert-deftest imoogi-process-test-card-option-keys-are-always-emitted ()
  (let* ((entries (list (list :key "a.org::0" :note-id nil
                               :note-type "imoogi-Cloze" :source-path "a.org"
                               :deck nil :tags nil :title "T" :body "B"
                               :direction nil :incremental nil :swift nil)))
         (json (imoogi-process-serialize-request
                (list :default-deck "D" :anki-connect-url "u"
                      :registry-path "r" :sync-root "s"
                      :exclude-patterns nil :scan-complete t)
                nil entries))
         (parsed (json-parse-string json :object-type 'alist :array-type 'list))
         (entry (car (cdr (assq 'entries parsed)))))
    (should (assq 'direction entry))
    (should (assq 'incremental entry))
    (should (assq 'swift entry))
    (should (eq (cdr (assq 'direction entry)) :null))
    (should (eq (cdr (assq 'incremental entry)) :null))
    (should (eq (cdr (assq 'swift entry)) :null))))

;; AC-OPT-004b: a resolved value is a JSON STRING -- including the falsy
;; spelling, which is exactly the value a boolean-typed wire would collapse
;; into the same thing as "no option at all". The malformed value rides
;; through untouched too: the front end interprets none of them.
(ert-deftest imoogi-process-test-card-options-serialize-as-strings ()
  (let* ((entries (list (list :key "a.org::0" :note-id nil
                               :note-type "imoogi-Cloze" :source-path "a.org"
                               :deck nil :tags nil :title "T" :body "B"
                               :direction "-->" :incremental "T" :swift "nil")))
         (json (imoogi-process-serialize-request
                (list :default-deck "D" :anki-connect-url "u"
                      :registry-path "r" :sync-root "s"
                      :exclude-patterns nil :scan-complete t)
                nil entries))
         (parsed (json-parse-string json :object-type 'alist :array-type 'list))
         (entry (car (cdr (assq 'entries parsed)))))
    (should (equal (cdr (assq 'swift entry)) "nil"))
    (should (stringp (cdr (assq 'swift entry))))
    ;; case intact, and a value the back end will reject carried verbatim
    (should (equal (cdr (assq 'incremental entry)) "T"))
    (should (equal (cdr (assq 'direction entry)) "-->"))))

(provide 'imoogi-process-test)
;;; imoogi-process-test.el ends here
