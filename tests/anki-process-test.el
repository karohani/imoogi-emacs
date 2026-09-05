;;; imoogi-process-test.el --- Tests for imoogi-process.el -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
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
    (should (= (cdr (assq 'protocol_version parsed)) 1))
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
          "{\"protocol_version\":1,\"ok\":true,\"results\":[{\"key\":\"a.org::0\",\"action\":\"added\",\"note_id\":1001},{\"key\":null,\"action\":\"deleted\",\"note_id\":1005}],\"errors\":[{\"code\":\"cloze_marker_missing\",\"message\":\"no marker\",\"key\":\"b.org::0\"}]}")
         (parsed (imoogi-process-parse-response response-json)))
    (should (= (plist-get parsed :protocol-version) 1))
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
           (cons 0 "{\"protocol_version\":1,\"ok\":true,\"results\":[],\"errors\":[]}"))))
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

(provide 'imoogi-process-test)
;;; imoogi-process-test.el ends here
