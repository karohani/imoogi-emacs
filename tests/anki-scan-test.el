;;; imoogi-scan-test.el --- Tests for imoogi-scan.el -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'imoogi-scan)

(defun imoogi-test--write (path content)
  "Write CONTENT to PATH, creating parent directories as needed."
  (make-directory (file-name-directory path) t)
  (with-temp-file path (insert content)))

(defmacro imoogi-test--with-root (&rest body)
  "Run BODY with ROOT bound to a fresh temp directory, cleaned up after."
  `(let ((root (make-temp-file "imoogi-scan-test" t)))
     (unwind-protect
         (progn ,@body)
       (delete-directory root t))))

;; --- AC-008 shape: recursion into nested/deep, exclusion filters targets
;; in both directions (never traversal). This is one of the four
;; highest-risk behaviors named in the delegation prompt Section D.

(ert-deftest imoogi-scan-test-recursion-and-exclusion ()
  (imoogi-test--with-root
   (imoogi-test--write (expand-file-name "top.org" root)
                        "* T\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody T\n")
   (imoogi-test--write (expand-file-name "nested/deep/inner.org" root)
                        "* I\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody I\n")
   (imoogi-test--write (expand-file-name "drafts/wip.org" root)
                        "* W\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody W\n")
   (let* ((scan (imoogi-scan-root root '("drafts/")))
          (entries (plist-get scan :entries))
          (paths (mapcar (lambda (e) (plist-get e :source-path)) entries)))
     (should (= (length entries) 2))
     (should (member "top.org" paths))
     (should (member (file-relative-name (expand-file-name "nested/deep/inner.org" root) root) paths))
     (should-not (member (file-relative-name (expand-file-name "drafts/wip.org" root) root) paths))
     (should (plist-get scan :scan-complete)))))

(ert-deftest imoogi-scan-test-census-includes-excluded-files ()
  (imoogi-test--with-root
   (imoogi-test--write (expand-file-name "a.org" root)
                        "* A\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ANKI_NOTE_ID: 1001\n:END:\nBody\n")
   (imoogi-test--write (expand-file-name "archive/old.org" root)
                        "* O\n:PROPERTIES:\n:ANKI_NOTE_ID: 1002\n:END:\nBody\n")
   (let* ((scan (imoogi-scan-root root '("archive/")))
          (census (plist-get scan :census))
          (ids (mapcar (lambda (c) (plist-get c :note-id)) census)))
     ;; the excluded file's identifier is still censused, even though it
     ;; contributes no sync target (no ANKI_NOTE_TYPE at all here).
     (should (member 1001 ids))
     (should (member 1002 ids))
     (should (= (length (plist-get scan :entries)) 1)))))

(ert-deftest imoogi-scan-test-duplicate-identifier-first-occurrence-wins ()
  (imoogi-test--with-root
   (imoogi-test--write (expand-file-name "a.org" root)
                        "* A\n:PROPERTIES:\n:ANKI_NOTE_ID: 9001\n:END:\nBody\n")
   (imoogi-test--write (expand-file-name "z.org" root)
                        "* Z\n:PROPERTIES:\n:ANKI_NOTE_ID: 9001\n:END:\nBody\n")
   (let* ((scan (imoogi-scan-root root nil))
          (census (plist-get scan :census))
          (occurrences (cl-remove-if-not
                        (lambda (c) (= (plist-get c :note-id) 9001)) census)))
     ;; both occurrences are recorded (census is a presence-and-location
     ;; record, not deduplicated); scan order is deterministic (a.org
     ;; sorts before z.org), so the first occurrence is a.org.
     (should (= (length occurrences) 2))
     (should (equal (plist-get (car occurrences) :source-path) "a.org")))))

(ert-deftest imoogi-scan-test-key-derivation-counts-targets-only ()
  (imoogi-test--with-root
   (imoogi-test--write
    (expand-file-name "f.org" root)
    (concat "* Untyped\nno marker here\n"
            "* T1\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody 1\n"
            "* Untyped2\nno marker\n"
            "* T2\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody 2\n"))
   (let* ((scan (imoogi-scan-root root nil))
          (entries (plist-get scan :entries))
          (keys (mapcar (lambda (e) (plist-get e :key)) entries)))
     (should (equal keys '("f.org::0" "f.org::1"))))))

(ert-deftest imoogi-scan-test-scan-complete-false-on-unreadable-file ()
  (imoogi-test--with-root
   (let ((bad (expand-file-name "locked.org" root)))
     (imoogi-test--write bad "* L\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody\n")
     (imoogi-test--write (expand-file-name "ok.org" root)
                          "* K\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody\n")
     (set-file-modes bad 0)
     (unwind-protect
         (if (file-readable-p bad)
             ;; running as root or on a filesystem that ignores 0 perms --
             ;; skip rather than false-fail.
             (ert-skip "cannot simulate an unreadable file in this environment")
           (let ((scan (imoogi-scan-root root nil)))
             (should-not (plist-get scan :scan-complete))
             (should (member "locked.org" (plist-get scan :unreadable-files)))
             ;; the readable file's target is still collected.
             (should (= (length (plist-get scan :entries)) 1))))
       (set-file-modes bad #o644)))))

(ert-deftest imoogi-scan-test-note-type-own-drawer-only-no-inherit ()
  (imoogi-test--with-root
   (imoogi-test--write
    (expand-file-name "f.org" root)
    (concat "* Parent\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n"
            "** Child\nno marker of its own\n"))
   (let* ((scan (imoogi-scan-root root nil))
          (entries (plist-get scan :entries)))
     (should (= (length entries) 1))
     (should (equal (plist-get (car entries) :title) "Parent")))))

(provide 'imoogi-scan-test)
;;; imoogi-scan-test.el ends here
