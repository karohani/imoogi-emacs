;;; project-notes-cli-cache-integration-test.el --- Project notes CLI cache integration tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'json)
(require 'org)

(defmacro imoogi-project-notes-cli-cache-test--isolated (&rest body)
  (declare (indent 0) (debug t))
  `(let* ((sandbox (file-truename
                    (make-temp-file "imoogi-project-notes-cli-cache-" t)))
          (user-emacs-directory (expand-file-name "emacs/" sandbox))
          (imoogi-project-notes-directory
           (expand-file-name "project-notes/" sandbox))
          (imoogi-project-notes-todo-storage 'project)
          (personal (expand-file-name "notes/" sandbox))
          (root (file-name-as-directory
                 (expand-file-name "source/" sandbox)))
          (registry-file (expand-file-name ".cache/project-notes.json"
                                           user-emacs-directory))
          (mounted-roots-file
           (expand-file-name ".cache/project-notes-mounted-roots.json"
                             user-emacs-directory))
          (org-id-locations-file (expand-file-name "org-id-locations" sandbox))
          (org-agenda-files nil)
          (org-directory personal))
     (make-directory root t)
     (unwind-protect
         (cl-letf (((symbol-function 'imoogi-org--default-directory)
                    (lambda () personal))
                   ((symbol-function 'imoogi-project-notes--registry-file)
                    (lambda () registry-file))
                   ((symbol-function 'imoogi-project-notes--mounted-roots-file)
                    (lambda () mounted-roots-file)))
           (save-window-excursion
             (with-temp-buffer
               (setq default-directory root)
               ,@body)))
       (dolist (buffer (buffer-list))
         (when (and (buffer-file-name buffer)
                    (file-in-directory-p (buffer-file-name buffer) sandbox))
           (with-current-buffer buffer
             (set-buffer-modified-p nil))
           (kill-buffer buffer)))
       (delete-directory sandbox t))))

(defun imoogi-project-notes-cli-cache-test--binary ()
  "Return the imoogi-notes binary, or skip when no executable exists."
  (let* ((configured (getenv "IMOOGI_NOTES_BIN"))
         (local (expand-file-name "bin/imoogi-notes" imoogi-test-root))
         (binary (or (and configured
                          (> (length configured) 0)
                          (executable-find configured))
                     (and (file-executable-p local) local)
                     (executable-find "imoogi-notes"))))
    (unless binary
      (ert-skip "imoogi-notes binary is not available"))
    binary))

(defun imoogi-project-notes-cli-cache-test--json-get (key alist)
  "Return KEY from ALIST parsed with string JSON keys."
  (alist-get key alist nil nil #'string=))

(defun imoogi-project-notes-cli-cache-test--read (file)
  "Return FILE content."
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(defun imoogi-project-notes-cli-cache-test--write (file text)
  "Write TEXT to FILE after creating its parent directory."
  (make-directory (file-name-directory file) t)
  (with-temp-file file
    (insert text)))

(defun imoogi-project-notes-cli-cache-test--excluded-roots (entry)
  "Return document cache exclusion roots for ENTRY."
  (let ((notes-root (file-truename
                     (imoogi-project-notes--alist-string 'notes-dir entry))))
    (vconcat
     (delq nil
           (mapcar
            (lambda (candidate)
              (let ((root (file-truename
                           (imoogi-project-notes--alist-string
                            'notes-dir candidate))))
                (unless (string= root notes-root)
                  root)))
            (imoogi-project-notes--all-entries))))))

(defun imoogi-project-notes-cli-cache-test--request
    (entry operation &optional extra)
  "Call the notes CLI for ENTRY OPERATION, merging EXTRA request fields."
  (let* ((notes-root (file-truename
                      (imoogi-project-notes--alist-string 'notes-dir entry)))
         (request
          (append
           `((protocol_version . 1)
             (operation . ,operation)
             (correlation_id . "ert-cache-integration")
             (scope . ((project_id . ,(imoogi-project-notes--entry-derived-note-id
                                       entry))
                       (notes_root . ,notes-root)
                       (tasks_file . ,(file-truename
                                       (imoogi-project-notes--alist-string
                                        'tasks-file entry)))
                       (excluded_roots .
                                       ,(imoogi-project-notes-cli-cache-test--excluded-roots
                                         entry))))
             (cache_dir . ,(expand-file-name ".cache/ert-cli-cache/" notes-root)))
           extra))
         (json-encoding-pretty-print nil)
         (payload (json-encode request))
         (stderr-file (make-temp-file "imoogi-notes-cli-cache-stderr-"))
         (binary (imoogi-project-notes-cli-cache-test--binary))
         status response stderr)
    (unwind-protect
        (with-temp-buffer
          (insert payload)
          (setq status
                (call-process-region
                 (point-min) (point-max) binary t (list t stderr-file) nil))
          (setq response (buffer-string))
          (setq stderr
                (when (file-readable-p stderr-file)
                  (imoogi-project-notes-cli-cache-test--read stderr-file)))
          (should (= 0 status))
          (let ((json-object-type 'alist)
                (json-array-type 'list)
                (json-key-type 'string))
            (goto-char (point-min))
            (let ((parsed (json-read)))
              (ert-info
                  ((format "binary=%s response=%s stderr=%s"
                           binary response (or stderr "")))
                (should (equal "ok"
                               (imoogi-project-notes-cli-cache-test--json-get
                                "status" parsed))))
              parsed)))
      (ignore-errors (delete-file stderr-file)))))

(defun imoogi-project-notes-cli-cache-test--documents-by-relative-file
    (response root)
  "Return RESPONSE documents keyed by path relative to ROOT."
  (mapcar
   (lambda (document)
     (cons (file-relative-name
            (imoogi-project-notes-cli-cache-test--json-get "file" document)
            root)
           document))
   (imoogi-project-notes-cli-cache-test--json-get "documents" response)))

(defun imoogi-project-notes-cli-cache-test--doc-field (documents file key)
  "Return KEY from DOCUMENTS entry for relative FILE."
  (imoogi-project-notes-cli-cache-test--json-get
   key (cdr (assoc file documents))))

(defun imoogi-project-notes-cli-cache-test--relative-occurrences
    (response id root)
  "Return RESPONSE occurrences for ID as relative FILE/POSITION pairs."
  (mapcar
   (lambda (occurrence)
     (cons (file-relative-name
            (imoogi-project-notes-cli-cache-test--json-get "file" occurrence)
            root)
           (imoogi-project-notes-cli-cache-test--json-get "position" occurrence)))
   (imoogi-project-notes-cli-cache-test--json-get
    id (imoogi-project-notes-cli-cache-test--json-get "occurrences" response))))

(defun imoogi-project-notes-cli-cache-test--errors-for-relative-file
    (response root file)
  "Return RESPONSE errors attached to relative FILE below ROOT."
  (cl-remove-if-not
   (lambda (error)
     (equal file
            (file-relative-name
             (imoogi-project-notes-cli-cache-test--json-get "file" error)
             root)))
   (imoogi-project-notes-cli-cache-test--json-get "errors" response)))

(ert-deftest imoogi-project-notes-cli-cache-request-uses-cli-default-cache-dir ()
  (imoogi-project-notes-cli-cache-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (imoogi-project-notes-cache-directory nil)
           (request (imoogi-project-notes--cache-request entry "catalog")))
      (should (not (assq 'cache_dir request)))
      (should (vectorp (alist-get 'overlays request)))
      (should (vectorp (alist-get 'excluded_roots
                                  (alist-get 'scope request))))
      (let* ((override (expand-file-name "editor-cache/" directory))
             (imoogi-project-notes-cache-directory override))
        (make-directory override t)
        (let ((request (imoogi-project-notes--cache-request entry "catalog")))
          (should (equal (file-truename override)
                         (alist-get 'cache_dir request))))))))

(ert-deftest imoogi-project-notes-cli-cache-context-requires-selected-buffer ()
  (let ((source (generate-new-buffer " *imoogi-cache-source*"))
        (other (generate-new-buffer " *imoogi-cache-other*")))
    (unwind-protect
        (save-window-excursion
          (switch-to-buffer source)
          (let ((marker (with-current-buffer source
                          (goto-char (point-min))
                          (copy-marker (point))))
                (tick (with-current-buffer source
                        (buffer-modified-tick))))
            (should (imoogi-project-notes--cache-context-current-p
                     source marker tick))
            (switch-to-buffer other)
            (should-not (imoogi-project-notes--cache-context-current-p
                         source marker tick))
            (switch-to-buffer source)
            (with-current-buffer source
              (insert "changed"))
            (should-not (imoogi-project-notes--cache-context-current-p
                         source marker tick))))
      (when (buffer-live-p source)
        (kill-buffer source))
      (when (buffer-live-p other)
        (kill-buffer other)))))

(ert-deftest imoogi-project-notes-cli-cache-superseded-process-does-not-callback ()
  (imoogi-project-notes-cli-cache-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (first-json "{\"version\":1,\"ok\":true,\"status\":\"ok\",\"documents\":[{\"file\":\"first\"}]}")
           (second-json "{\"version\":1,\"ok\":true,\"status\":\"ok\",\"documents\":[{\"file\":\"second\"}]}")
           callbacks
           command)
      (cl-letf (((symbol-function 'imoogi-project-notes--all-entries)
                 (lambda () (list entry)))
                ((symbol-function 'imoogi-project-notes--cache-command-or-error)
                 (lambda () command)))
        (setq command
              (list "sh" "-c"
                    (format "sleep 0.2; printf '%%s' '%s'"
                            first-json)))
        (imoogi-project-notes--cache-call-async
         entry "catalog"
         (lambda (response)
           (push (alist-get 'file (car (alist-get 'documents response)))
                 callbacks)))
        (setq command
              (list "sh" "-c"
                    (format "sleep 0.05; printf '%%s' '%s'"
                            second-json)))
        (imoogi-project-notes--cache-call-async
         entry "catalog"
         (lambda (response)
           (push (alist-get 'file (car (alist-get 'documents response)))
                 callbacks)))
        (let ((deadline (+ (float-time) 2.0)))
          (while (and (< (float-time) deadline)
                      (or imoogi-project-notes--pending-cache-process
                          (< (length callbacks) 1)))
            (accept-process-output nil 0.05)))
        (should (equal '("second") callbacks))))))

(defun imoogi-project-notes-cli-cache-test--emacs-documents (entry)
  "Return Emacs document identities keyed by relative file for ENTRY."
  (let ((root (file-truename
               (imoogi-project-notes--alist-string 'notes-dir entry))))
    (mapcar
     (lambda (doc)
       (cons (file-relative-name (alist-get 'file doc) root)
             doc))
     (imoogi-project-notes--document-catalog entry))))

(defun imoogi-project-notes-cli-cache-test--emacs-occurrences (entry id)
  "Return Emacs project ID occurrences for ID as relative FILE/POSITION pairs."
  (let ((root (file-truename
               (imoogi-project-notes--alist-string 'notes-dir entry))))
    (mapcar
     (lambda (occurrence)
       (cons (file-relative-name (car occurrence) root)
             (cdr occurrence)))
     (imoogi-project-notes--project-id-occurrences entry id))))

(ert-deftest imoogi-project-notes-cli-cache-documents-match-org-parser-identities ()
  (imoogi-project-notes-cli-cache-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (file-doc (expand-file-name "references/file.org" directory))
           (legacy-doc (expand-file-name "development/legacy.org" directory))
           (unicode-doc (expand-file-name "references/한글.org" directory))
           (tricky-doc (expand-file-name "references/tricky.org" directory))
           (indented-doc (expand-file-name "references/indented.org" directory))
           (planning-doc (expand-file-name "references/planning.org" directory))
           (unclosed-doc (expand-file-name "references/unclosed.org" directory)))
      (imoogi-project-notes-cli-cache-test--write
       file-doc
       ":PROPERTIES:\n:ID:       FILE-DOC-ID\n:END:\n#+TITLE: File Document\n\nBody\n")
      (imoogi-project-notes-cli-cache-test--write
       legacy-doc
       "#+TITLE: Legacy Container\n\n* Legacy Artifact\n:PROPERTIES:\n:TYPE:     development\n:ID:       LEGACY-DOC-ID\n:END:\n")
      (imoogi-project-notes-cli-cache-test--write
       unicode-doc
       ":PROPERTIES:\n:ID:       UNICODE-DOC-ID\n:END:\n#+TITLE: 한글 문서\n\n내용\n")
      (imoogi-project-notes-cli-cache-test--write
       tricky-doc
       ":PROPERTIES:\n:ID:       TRICKY-DOC-ID\n:END:\n#+TITLE: Tricky IDs\n\n#+begin_src org\n:ID:       SRC-DECOY-ID\n#+end_src\n\n#+begin_example\n:ID:       EXAMPLE-DECOY-ID\n#+end_example\n\n# :ID:       COMMENT-DECOY-ID\n\n#+begin_quote\n:ID:       QUOTE-DECOY-ID\n#+end_quote\n")
      (imoogi-project-notes-cli-cache-test--write
       indented-doc
       "#+TITLE: Indented Property\n\n* Indented Artifact\n  :PROPERTIES:\n  :TYPE:     development\n  :ID:       INDENTED-DOC-ID\n  :END:\n")
      (imoogi-project-notes-cli-cache-test--write
       planning-doc
       "#+TITLE: Planning Property\n\n* TODO Planning Artifact\nSCHEDULED: <2026-10-03 Sat>\n:PROPERTIES:\n:TYPE:     development\n:ID:       PLANNING-DOC-ID\n:END:\n")
      (imoogi-project-notes-cli-cache-test--write
       unclosed-doc
       "#+TITLE: Unclosed Property\n\n* Unclosed Artifact\n:PROPERTIES:\n:TYPE:     development\n:ID:       UNCLOSED-DOC-ID\nBody without drawer end.\n")
      (let* ((response (imoogi-project-notes-cli-cache-test--request
                        entry "catalog"))
             (cli-docs
              (imoogi-project-notes-cli-cache-test--documents-by-relative-file
               response directory))
             (emacs-docs
              (imoogi-project-notes-cli-cache-test--emacs-documents entry)))
        (dolist (file '("references/file.org"
                        "development/legacy.org"
                        "references/한글.org"
                        "references/tricky.org"
                        "references/indented.org"
                        "references/planning.org"
                        "references/unclosed.org"))
          (ert-info
              ((format "file=%s cli-doc-files=%S" file (mapcar #'car cli-docs)))
            (should (equal (alist-get 'id (cdr (assoc file emacs-docs)))
                           (imoogi-project-notes-cli-cache-test--doc-field
                            cli-docs file "id")))
            (let ((emacs-kind (alist-get 'kind (cdr (assoc file emacs-docs)))))
              (should (equal (and emacs-kind (symbol-name emacs-kind))
                             (imoogi-project-notes-cli-cache-test--doc-field
                              cli-docs file "kind")))
              (when emacs-kind
                (should (equal (alist-get 'title (cdr (assoc file emacs-docs)))
                               (imoogi-project-notes-cli-cache-test--doc-field
                                cli-docs file "title")))))))))))

(ert-deftest imoogi-project-notes-cli-cache-occurrences-match-org-parser ()
  (imoogi-project-notes-cli-cache-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (tasks (expand-file-name "tasks.org" directory))
           (doc (expand-file-name "references/occurrences.org" directory))
           (unclosed-doc (expand-file-name "references/unclosed-occurrence.org"
                                           directory)))
      (imoogi-project-notes-cli-cache-test--write
       doc
       ":PROPERTIES:\n:ID:       OCCUR-DOC-ID\n:END:\n#+TITLE: Occurrences\n\n#+begin_src org\n:ID:       OCCUR-DECOY-ID\n#+end_src\n\n# :ID:       OCCUR-COMMENT-ID\n")
      (imoogi-project-notes-cli-cache-test--write
       unclosed-doc
       "#+TITLE: Unclosed Occurrence\n\n* Unclosed Node\n:PROPERTIES:\n:ID:       OCCUR-UNCLOSED-ID\nBody without drawer end.\n")
      (find-file tasks)
      (goto-char (point-max))
      (insert "\n* TODO Cached task\n:PROPERTIES:\n:ID:       OCCUR-TASK-ID\n:END:\n")
      (save-buffer)
      (let ((response (imoogi-project-notes-cli-cache-test--request
                       entry "index")))
        (dolist (id '("OCCUR-DOC-ID" "OCCUR-TASK-ID"
                      "OCCUR-DECOY-ID" "OCCUR-COMMENT-ID"
                      "OCCUR-UNCLOSED-ID"))
          (should (equal (imoogi-project-notes-cli-cache-test--emacs-occurrences
                          entry id)
                         (imoogi-project-notes-cli-cache-test--relative-occurrences
                          response id directory))))))))

(ert-deftest imoogi-project-notes-cli-cache-scope-matches-project-document-files ()
  (imoogi-project-notes-cli-cache-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (nested-source (expand-file-name "nested-source/" sandbox))
           (nested-notes (expand-file-name "references/nested-project/"
                                           directory))
           (nested-entry (imoogi-project-notes--entry
                          "dir:nested" nested-source nested-notes 'project))
           (outside (expand-file-name "outside.org" sandbox))
           (symlink (expand-file-name "references/outside.org" directory))
           (original-tasks (expand-file-name "tasks.org" directory))
           (sub-tasks (expand-file-name "sub/agenda/tasks.org" directory)))
      (make-directory nested-source t)
      (make-directory nested-notes t)
      (setf (alist-get 'tasks-file entry) sub-tasks)
      (imoogi-project-notes-cli-cache-test--write
       (expand-file-name "references/visible.org" directory)
       "#+TITLE: Visible\n")
      (imoogi-project-notes-cli-cache-test--write
       (expand-file-name ".hidden/hidden.org" directory)
       "#+TITLE: Hidden\n")
      (imoogi-project-notes-cli-cache-test--write
       (expand-file-name ".hidden.org" directory)
       "#+TITLE: Hidden File\n")
      (imoogi-project-notes-cli-cache-test--write
       original-tasks
       "* TODO Root tasks still excluded\n")
      (imoogi-project-notes-cli-cache-test--write
       sub-tasks
       "* TODO Exact configured tasks file\n")
      (imoogi-project-notes-cli-cache-test--write
       (expand-file-name "sub/agenda/other.org" directory)
       "#+TITLE: Sibling Under Task Directory\n")
      (imoogi-project-notes-cli-cache-test--write
       (expand-file-name "vendor/root-vendor.org" directory)
       "#+TITLE: Root Vendor\n")
      (imoogi-project-notes-cli-cache-test--write
       (expand-file-name "development/vendor/dev-vendor.org" directory)
       "#+TITLE: Development Vendor\n")
      (imoogi-project-notes-cli-cache-test--write
       (expand-file-name "inner.org" nested-notes)
       "#+TITLE: Nested\n")
      (imoogi-project-notes-cli-cache-test--write outside "#+TITLE: Outside\n")
      (make-directory (file-name-directory symlink) t)
      (make-symbolic-link outside symlink)
      (imoogi-project-notes--write-registry (list entry nested-entry))
      (let* ((response (imoogi-project-notes-cli-cache-test--request
                        entry "catalog"))
             (cli-files
              (sort
               (mapcar #'car
                       (imoogi-project-notes-cli-cache-test--documents-by-relative-file
                        response directory))
               #'string<))
             (emacs-files
              (sort
               (mapcar (lambda (file)
                         (file-relative-name file directory))
                       (imoogi-project-notes--project-document-files entry))
               #'string<)))
        (should (equal emacs-files cli-files))))))

(ert-deftest imoogi-project-notes-cli-cache-reports-ambiguous-document-identity ()
  (imoogi-project-notes-cli-cache-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (ambiguous (expand-file-name "references/ambiguous.org" directory)))
      (imoogi-project-notes-cli-cache-test--write
       ambiguous
       "#+TITLE: Ambiguous\n\n* First\n:PROPERTIES:\n:TYPE:     development\n:ID:       AMBIGUOUS-A\n:END:\n\n* Second\n:PROPERTIES:\n:TYPE:     development\n:ID:       AMBIGUOUS-B\n:END:\n")
      (let* ((response (imoogi-project-notes-cli-cache-test--request
                        entry "catalog"))
             (docs (imoogi-project-notes-cli-cache-test--documents-by-relative-file
                    response directory))
             (errors
              (imoogi-project-notes-cli-cache-test--errors-for-relative-file
               response directory "references/ambiguous.org")))
        (should (or (equal "multiple document ID candidates"
                           (imoogi-project-notes-cli-cache-test--doc-field
                            docs "references/ambiguous.org" "identity_error"))
                    (cl-some
                     (lambda (error)
                       (string-match-p
                        "multiple\\|ambiguous\\|여러"
                        (or (imoogi-project-notes-cli-cache-test--json-get
                             "message" error)
                            "")))
                     errors)))
        (should-not (imoogi-project-notes-cli-cache-test--doc-field
                     docs "references/ambiguous.org" "id"))))))

(ert-deftest imoogi-project-notes-cli-cache-overlay-detects-duplicate-without-writing ()
  (imoogi-project-notes-cli-cache-test--isolated
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (stable (expand-file-name "references/stable.org" directory))
           (dirty (expand-file-name "references/dirty.org" directory))
           (dirty-disk
            ":PROPERTIES:\n:ID:       DIRTY-DISK-ID\n:END:\n#+TITLE: Dirty\n")
           (dirty-overlay
            ":PROPERTIES:\n:ID:       DUPLICATE-ID\n:END:\n#+TITLE: Dirty overlay\n"))
      (imoogi-project-notes-cli-cache-test--write
       stable
       ":PROPERTIES:\n:ID:       DUPLICATE-ID\n:END:\n#+TITLE: Stable\n")
      (imoogi-project-notes-cli-cache-test--write dirty dirty-disk)
      (let* ((response
              (imoogi-project-notes-cli-cache-test--request
               entry "lookup"
               `((id . "DUPLICATE-ID")
                 (overlays . [((path . ,dirty)
                               (text . ,dirty-overlay)
                               (revision . 42))]))))
             (occurrences
              (imoogi-project-notes-cli-cache-test--relative-occurrences
               response "DUPLICATE-ID" directory)))
        (should (= 2 (length occurrences)))
        (should (member "references/stable.org" (mapcar #'car occurrences)))
        (should (member "references/dirty.org" (mapcar #'car occurrences)))
        (should (equal dirty-disk
                       (imoogi-project-notes-cli-cache-test--read dirty)))))))

(ert-deftest imoogi-project-notes-cli-cache-central-tasks-outside-notes-root-is-exact-scope ()
  (imoogi-project-notes-cli-cache-test--isolated
    (setq imoogi-project-notes-todo-storage 'central)
    (let* ((directory (imoogi-project-notes-setup root))
           (entry (car (imoogi-project-notes--read-registry)))
           (tasks (imoogi-project-notes--alist-string 'tasks-file entry))
           (outside-peer (expand-file-name "outside-peer.org"
                                           (file-name-directory tasks)))
           (doc (expand-file-name "references/central-doc.org" directory))
           (task-disk
            "* TODO Central task\n:PROPERTIES:\n:ID:       CENTRAL-TASK-ID\n:END:\n")
           (task-overlay
            "* TODO Central task\n:PROPERTIES:\n:ID:       CENTRAL-DUPLICATE-ID\n:END:\n"))
      (imoogi-project-notes-cli-cache-test--write
       doc
       ":PROPERTIES:\n:ID:       CENTRAL-DUPLICATE-ID\n:END:\n#+TITLE: Central Doc\n")
      (imoogi-project-notes-cli-cache-test--write tasks task-disk)
      (imoogi-project-notes-cli-cache-test--write
       outside-peer
       "* TODO Outside peer\n:PROPERTIES:\n:ID:       OUTSIDE-PEER-ID\n:END:\n")
      (let* ((response
              (imoogi-project-notes-cli-cache-test--request
               entry "lookup"
               `((id . "CENTRAL-DUPLICATE-ID")
                 (overlays . [((path . ,tasks)
                               (text . ,task-overlay)
                               (revision . 7))]))))
             (duplicate-occurrences
              (imoogi-project-notes-cli-cache-test--relative-occurrences
               response "CENTRAL-DUPLICATE-ID" directory))
             (peer-occurrences
              (imoogi-project-notes-cli-cache-test--relative-occurrences
               response "OUTSIDE-PEER-ID" directory)))
        (should (= 2 (length duplicate-occurrences)))
        (should (member "references/central-doc.org"
                        (mapcar #'car duplicate-occurrences)))
        (should (member (file-relative-name tasks directory)
                        (mapcar #'car duplicate-occurrences)))
        (should-not peer-occurrences)
        (should (equal task-disk
                       (imoogi-project-notes-cli-cache-test--read tasks)))))))

;;; project-notes-cli-cache-integration-test.el ends here
