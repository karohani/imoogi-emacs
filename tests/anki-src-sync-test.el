;;; anki-src-sync-test.el --- SPEC-ANKISRC-001 sync side: Org IDs on the wire -*- lexical-binding: t; -*-

;;; Commentary:

;; AC-ASRC-001..005 and 008 (the Elisp half): the pre-scan ID pass, the
;; heading-local :ID: in the scan, the optional `org_id' on the wire,
;; duplicate IDs, and the single end-of-sync `org-id-locations' save.
;;
;; Every test isolates `org-id-locations-file', `org-id-locations' and
;; `org-id-files' to temporary state; nothing reads or writes the user's
;; real ID cache.  Test names carry `anki' so the `anki' ERT selector
;; runs them.

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'org)
(require 'org-id)
(require 'imoogi)

(defvar imoogi-anki-src-test--requests nil
  "Request JSON strings the stub runner received, newest first.")

(defvar imoogi-anki-src-test--action "updated"
  "Action the stub runner reports for every entry.")

(defun imoogi-anki-src-test--respond (request-json)
  "Answer REQUEST-JSON with one result per entry."
  (let* ((parsed (json-parse-string request-json :object-type 'alist :array-type 'list
                                    :null-object nil))
         (next 4000))
    (json-serialize
     (list (cons 'protocol_version 2)
           (cons 'ok t)
           (cons 'results
                 (vconcat
                  (mapcar (lambda (entry)
                            (list (cons 'key (cdr (assq 'key entry)))
                                  (cons 'action imoogi-anki-src-test--action)
                                  (cons 'note_id (or (cdr (assq 'note_id entry))
                                                     (cl-incf next)))))
                          (cdr (assq 'entries parsed)))))
           (cons 'errors (vector))))))

(defmacro imoogi-anki-src-test--with-sync (&rest body)
  "Run BODY with a fresh sync ROOT, isolated Org ID state and a stub binary.
Binds BASE, ROOT and LOCATIONS (the temporary `org-id-locations-file')."
  (declare (indent 0))
  `(let* ((base (file-name-as-directory (make-temp-file "imoogi-src-sync" t)))
          (root (file-name-as-directory (expand-file-name "notes" base)))
          (locations (expand-file-name "state/org-id-locations" base))
          (org-id-locations-file locations)
          (org-id-locations (make-hash-table :test 'equal))
          (org-id-files nil)
          (org-id-track-globally t)
          (imoogi-targets-file (expand-file-name "state/targets.json" base))
          (imoogi-sync-root root)
          (imoogi-exclude-patterns nil)
          (imoogi-anki-binary-path (executable-find "true"))
          (imoogi-anki-src-test--requests nil)
          (imoogi-anki-src-test--action "updated")
          (imoogi-process-runner
           (lambda (_binary request-json)
             (push request-json imoogi-anki-src-test--requests)
             (cons 0 (imoogi-anki-src-test--respond request-json)))))
     (make-directory root t)
     (make-directory (file-name-directory locations) t)
     (unwind-protect (progn ,@body)
       (dolist (buffer (buffer-list))
         (let ((file (buffer-file-name buffer)))
           (when (and file (file-in-directory-p file base))
             (with-current-buffer buffer (set-buffer-modified-p nil))
             (kill-buffer buffer))))
       (dolist (file (directory-files-recursively base "" nil))
         (ignore-errors (set-file-modes file #o644)))
       (delete-directory base t))))

(defun imoogi-anki-src-test--write (root relative content)
  "Write CONTENT to RELATIVE under ROOT and return the absolute path."
  (let ((file (expand-file-name relative root)))
    (make-directory (file-name-directory file) t)
    (with-temp-file file (insert content))
    file))

(defun imoogi-anki-src-test--read (file)
  "Return FILE's contents from disk."
  (with-temp-buffer (insert-file-contents file) (buffer-string)))

(defun imoogi-anki-src-test--disk-ids (file)
  "Return the heading-local IDs of FILE's headings on disk, in order.
A heading without its own ID contributes nil."
  (with-temp-buffer
    (insert-file-contents file)
    (delay-mode-hooks (org-mode))
    (org-map-entries (lambda () (org-entry-get (point) "ID" nil)))))

(defun imoogi-anki-src-test--entries ()
  "Return every request entry the stub received, as alists, in send order."
  (apply #'append
         (mapcar (lambda (json)
                   (cdr (assq 'entries
                              (json-parse-string json :object-type 'alist
                                                 :array-type 'list :null-object nil))))
                 (reverse imoogi-anki-src-test--requests))))

(defun imoogi-anki-src-test--raw-entries ()
  "Return every sent entry parsed losslessly enough to re-serialize it."
  (apply #'append
         (mapcar (lambda (json)
                   (append (cdr (assq 'entries
                                      (json-parse-string json :object-type 'alist
                                                         :array-type 'array)))
                           nil))
                 (reverse imoogi-anki-src-test--requests))))

(defun imoogi-anki-src-test--entry (key)
  "Return the sent entry whose key is KEY."
  (cl-find key (imoogi-anki-src-test--entries)
           :key (lambda (e) (cdr (assq 'key e))) :test #'equal))

(defun imoogi-anki-src-test--sent-org-id (key)
  "Return the org_id the request carried for KEY, or nil when absent."
  (cdr (assq 'org_id (imoogi-anki-src-test--entry key))))

(defun imoogi-anki-src-test--cached-files ()
  "Return every file `org-id-locations' maps an ID to, expanded."
  (let (files)
    (when (hash-table-p org-id-locations)
      (maphash (lambda (_id file) (push (expand-file-name file) files)) org-id-locations))
    files))

;; --- AC-ASRC-001: a new heading's first sync; an excluded file is untouched.

(ert-deftest imoogi-anki-src-sync-first-sync-gives-heading-local-ids ()
  (imoogi-anki-src-test--with-sync
    (let* ((file (imoogi-anki-src-test--write
                  root "a.org"
                  (concat "* Card A\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nA\n"
                          "* Parent\n:PROPERTIES:\n:ID: parent-id\n:END:\n"
                          "** Child\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nC\n")))
           (excluded (imoogi-anki-src-test--write
                      root "drafts/x.org"
                      "* Draft\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nD\n"))
           (excluded-before (imoogi-anki-src-test--read excluded))
           (excluded-mtime (file-attribute-modification-time (file-attributes excluded)))
           (imoogi-exclude-patterns '("drafts/")))
      (imoogi-sync)
      (let* ((ids (imoogi-anki-src-test--disk-ids file))
             (card-id (nth 0 ids))
             (child-id (nth 2 ids)))
        (should card-id)
        (should child-id)
        (should-not (equal card-id child-id))
        (should-not (member "parent-id" (list card-id child-id)))
        (should (equal (nth 1 ids) "parent-id"))
        (should (equal (imoogi-anki-src-test--sent-org-id "a.org::0") card-id))
        (should (equal (imoogi-anki-src-test--sent-org-id "a.org::1") child-id)))
      (should (equal (imoogi-anki-src-test--read excluded) excluded-before))
      (should (equal (file-attribute-modification-time (file-attributes excluded))
                     excluded-mtime))
      (should-not (find-buffer-visiting file))
      (should-not (find-buffer-visiting excluded)))))

;; --- AC-ASRC-002: an existing ID and an existing ANKI_NOTE_ID are preserved.

(ert-deftest imoogi-anki-src-sync-preserves-existing-ids ()
  (imoogi-anki-src-test--with-sync
    (let ((file (imoogi-anki-src-test--write
                 root "a.org"
                 (concat "* Has ID\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ID: X-id-1\n:END:\nA\n"
                         "* Synced\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ANKI_NOTE_ID: 777\n:END:\nB\n")))
          (imoogi-anki-src-test--action "added"))
      (imoogi-sync)
      (let ((disk (imoogi-anki-src-test--read file))
            (ids (imoogi-anki-src-test--disk-ids file)))
        (should (string-match-p "^:ID: X-id-1$" disk))
        (should (equal (nth 0 ids) "X-id-1"))
        (should (nth 1 ids))
        (should-not (equal (nth 1 ids) "X-id-1"))
        ;; the stub answers `added' with the entry's own note_id (777) for
        ;; the synced heading, so the write-back rewrites the same value.
        (should (string-match-p ":ANKI_NOTE_ID: 777" disk))
        (should (equal (imoogi-anki-src-test--sent-org-id "a.org::0") "X-id-1"))
        (should (equal (imoogi-anki-src-test--sent-org-id "a.org::1") (nth 1 ids)))
        (should (equal (cdr (assq 'note_id (imoogi-anki-src-test--entry "a.org::1"))) 777))))))

;; --- AC-ASRC-003: modified buffer, read-only file, healthy file.

(ert-deftest imoogi-anki-src-sync-modified-buffer-and-failed-file ()
  (imoogi-anki-src-test--with-sync
    (let* ((a (imoogi-anki-src-test--write
               root "a.org" "* A\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nA\n"))
           (b (imoogi-anki-src-test--write
               root "b.org"
               (concat "* B0\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nB\n"
                       "* B1\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ID: b1-disk\n:END:\nB1\n")))
           (c (imoogi-anki-src-test--write
               root "c.org" "* C\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nC\n"))
           (a-disk (imoogi-anki-src-test--read a))
           (b-disk (imoogi-anki-src-test--read b))
           (a-buffer (find-file-noselect a))
           (prompts 0)
           report)
      (with-current-buffer a-buffer
        (goto-char (point-max))
        (insert "user edit\n"))
      (set-file-modes b #o444)
      (cl-letf (((symbol-function 'yes-or-no-p) (lambda (&rest _) (cl-incf prompts) t))
                ((symbol-function 'y-or-n-p) (lambda (&rest _) (cl-incf prompts) t)))
        (setq report (imoogi-sync)))
      (should (= prompts 0))
      ;; (a) unsaved user edits survive, nothing saved, needs-save reported.
      (should (buffer-live-p a-buffer))
      (should (buffer-modified-p a-buffer))
      (should (string-match-p "user edit" (with-current-buffer a-buffer (buffer-string))))
      (should (equal (imoogi-anki-src-test--read a) a-disk))
      (should-not (imoogi-anki-src-test--sent-org-id "a.org::0"))
      (should (string-match-p "a\\.org" report))
      ;; an ID not on disk is not persisted to the location cache (O5).
      (should-not (member (file-truename a) (mapcar #'file-truename
                                                    (imoogi-anki-src-test--cached-files))))
      ;; (b) failure confined to the file; the disk :ID: still travels.
      (should (equal (imoogi-anki-src-test--read b) b-disk))
      (should-not (imoogi-anki-src-test--sent-org-id "b.org::0"))
      (should (equal (imoogi-anki-src-test--sent-org-id "b.org::1") "b1-disk"))
      (should (string-match-p "b\\.org" report))
      (should (string-match-p (regexp-quote (imoogi-error-message "org_id_create_failed")) report))
      ;; (c) the healthy file got its ID and sent it.
      (should (equal (imoogi-anki-src-test--sent-org-id "c.org::0")
                     (car (imoogi-anki-src-test--disk-ids c))))
      ;; buffers the pass opened are gone; the user's buffer remains.
      (should-not (find-buffer-visiting b))
      (should-not (find-buffer-visiting c))
      (should (eq (find-buffer-visiting a) a-buffer)))))

;; --- O5: a save failure discards the in-memory location of the ID it made.

(ert-deftest imoogi-anki-src-sync-save-failure-drops-unsaved-id-location ()
  (imoogi-anki-src-test--with-sync
    (let* ((file (imoogi-anki-src-test--write
                  root "a.org" "* A\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nA\n"))
           (disk (imoogi-anki-src-test--read file))
           (real-save (symbol-function 'save-buffer))
           report)
      (cl-letf (((symbol-function 'save-buffer)
                 (lambda (&rest args)
                   ;; The configuration may visit files by their truename.
                   (if (equal (file-truename (buffer-file-name)) (file-truename file))
                       (signal 'file-error (list "Simulated save failure" file))
                     (apply real-save args)))))
        (setq report (imoogi-sync)))
      (should (equal (imoogi-anki-src-test--read file) disk))
      (should-not (imoogi-anki-src-test--sent-org-id "a.org::0"))
      (should-not (find-buffer-visiting file))
      (should (string-match-p "a\\.org" report))
      (should (= 0 (hash-table-count org-id-locations))))))

;; --- AC-ASRC-004: duplicate IDs are reported, sent without org_id.

(ert-deftest imoogi-anki-src-sync-duplicate-ids-are-sent-without-org-id ()
  (imoogi-anki-src-test--with-sync
    (let* ((same (imoogi-anki-src-test--write
                  root "same.org"
                  (concat "* S0\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ID: dup-1\n:END:\nS0\n"
                          "* S1\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ID: dup-1\n:END:\nS1\n")))
           (x (imoogi-anki-src-test--write
               root "x.org" "* X\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ID: dup-2\n:END:\nX\n"))
           (y (imoogi-anki-src-test--write
               root "y.org" "* Y\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ID: dup-2\n:END:\nY\n"))
           (before (mapcar #'imoogi-anki-src-test--read (list same x y)))
           (baseline (mapcar (lambda (entry)
                               (json-serialize
                                (imoogi-process--entry-alist
                                 (plist-put (copy-sequence entry) :org-id nil))))
                             (plist-get (imoogi-scan-root root nil) :entries)))
           (report (imoogi-sync))
           (sent (imoogi-anki-src-test--entries)))
      (should (= (length sent) 4))
      (dolist (entry sent)
        (should-not (assq 'org_id entry)))
      (should (equal (sort (mapcar #'json-serialize (imoogi-anki-src-test--raw-entries)) #'string<)
                     (sort baseline #'string<)))
      (should (string-match-p "4 results" report))
      (should-not (string-match-p "failed" report))
      (dolist (needle '("dup-1" "dup-2" "same\\.org" "x\\.org" "y\\.org"))
        (should (string-match-p needle report)))
      (should (equal (mapcar #'imoogi-anki-src-test--read (list same x y)) before)))))

;; --- AC-ASRC-005: every sent ID is registered, then saved once.

(ert-deftest imoogi-anki-src-sync-registers-ids-and-saves-once-at-the-end ()
  (imoogi-anki-src-test--with-sync
    (let* ((one (imoogi-anki-src-test--write
                 base "one/n.org" "* New\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nN\n"))
           (two (imoogi-anki-src-test--write
                 base "two/e.org" "* Existing\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ID: existing-id\n:END:\nE\n"))
           (imoogi-sync-root nil)
           (real-save (symbol-function 'org-id-locations-save))
           (saves nil))
      (imoogi-targets--write (list (list :kind 'directory :path (expand-file-name "one" base))
                                   (list :kind 'directory :path (expand-file-name "two" base))))
      (cl-letf (((symbol-function 'org-id-locations-save)
                 (lambda ()
                   (push (copy-hash-table org-id-locations) saves)
                   (funcall real-save))))
        (imoogi-sync))
      (let ((new-id (car (imoogi-anki-src-test--disk-ids one))))
        (should new-id)
        (should (= (length saves) 1))
        ;; at the moment of the one save, both IDs were already registered.
        (should (gethash new-id (car saves)))
        (should (gethash "existing-id" (car saves)))
        (let ((on-disk (with-temp-buffer
                         (insert-file-contents locations)
                         (read (current-buffer))))
              found)
          (dolist (item on-disk)
            (dolist (id (cdr item))
              (push (cons id (file-truename (expand-file-name (car item)))) found)))
          (should (equal (cdr (assoc new-id found)) (file-truename one)))
          (should (equal (cdr (assoc "existing-id" found)) (file-truename two))))))))

;; --- AC-ASRC-008 (Elisp half): org_id is emitted only when present.

(ert-deftest imoogi-anki-src-process-org-id-is-optional ()
  (let ((with (json-serialize (imoogi-process--entry-alist
                               (list :key "a.org::0" :note-type "Basic" :source-path "a.org"
                                     :title "T" :body "B" :org-id "abc"))))
        (without (json-serialize (imoogi-process--entry-alist
                                  (list :key "a.org::0" :note-type "Basic" :source-path "a.org"
                                        :title "T" :body "B")))))
    (should (string-match-p "\"org_id\":\"abc\"" with))
    (should-not (string-match-p "org_id" without))))

(ert-deftest imoogi-anki-src-scan-reads-heading-local-id-only ()
  (let ((root (make-temp-file "imoogi-src-scan" t)))
    (unwind-protect
        (progn
          (with-temp-file (expand-file-name "a.org" root)
            (insert "* P\n:PROPERTIES:\n:ID: parent\n:END:\n"
                    "** Child\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nC\n"
                    "** Own\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ID: own-id\n:END:\nO\n"))
          (let ((entries (plist-get (imoogi-scan-root root nil) :entries)))
            (should (null (plist-get (nth 0 entries) :org-id)))
            (should (equal (plist-get (nth 1 entries) :org-id) "own-id"))))
      (delete-directory root t))))

;; --- AC-ASRC-006 (Elisp half): the front-end table carries the new codes.

(ert-deftest imoogi-anki-src-error-table-has-org-id-codes ()
  (dolist (code '("org_id_invalid" "org_id_duplicated" "org_id_create_failed"))
    (should (assoc code imoogi-error-table))))

(provide 'anki-src-sync-test)
;;; anki-src-sync-test.el ends here
