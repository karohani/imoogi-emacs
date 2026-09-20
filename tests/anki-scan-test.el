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

;; --- SPEC-ANKICARD-002: the scan attaches the three resolved card-option
;; values to the entry, so an entry's option set is fixed at scan time and is
;; never re-derived downstream.

;; AC-OPT-003: a sync target carries its three resolved values, drawn from
;; whichever level of the chain supplied each.
(ert-deftest imoogi-scan-test-entry-carries-resolved-card-options ()
  (imoogi-test--with-root
   (imoogi-test--write (expand-file-name "opts.org" root)
                        (concat "#+PROPERTY: ANKI_SWIFT t\n"
                                "* Optioned\n:PROPERTIES:\n"
                                ":ANKI_NOTE_TYPE: imoogi-Cloze\n"
                                ":ANKI_DIRECTION: <->\n:END:\n"
                                "Body {{c1::x}}\n"))
   (let* ((scan (imoogi-scan-root root nil))
          (entry (car (plist-get scan :entries))))
     (should (equal (plist-get entry :direction) "<->"))
     ;; inherited from the file-level keyword
     (should (equal (plist-get entry :swift) "t"))
     ;; absent at every level -- no value, which the wire spells null
     (should (null (plist-get entry :incremental))))))

;; AC-OPT-003's second half: a heading carrying no card-option property at any
;; level produces an entry whose three values are all no value.
(ert-deftest imoogi-scan-test-entry-without-card-options-carries-none ()
  (imoogi-test--with-root
   (imoogi-test--write (expand-file-name "bare.org" root)
                        "* Bare\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Basic\n:END:\nBody\n")
   (let* ((scan (imoogi-scan-root root nil))
          (entry (car (plist-get scan :entries))))
     (should (null (plist-get entry :direction)))
     (should (null (plist-get entry :incremental)))
     (should (null (plist-get entry :swift))))))

;; AC-OPT-002a through the scan: a value the back end will reject reaches the
;; entry unchanged. The front end drops no entry and substitutes no default.
(ert-deftest imoogi-scan-test-malformed-card-option-reaches-the-entry-verbatim ()
  (imoogi-test--with-root
   (imoogi-test--write (expand-file-name "bad.org" root)
                        (concat "* Bad\n:PROPERTIES:\n"
                                ":ANKI_NOTE_TYPE: imoogi-Cloze\n"
                                ":ANKI_DIRECTION: -->\n:ANKI_INCREMENTAL: yes\n:END:\n"
                                "Body {{c1::x}}\n"))
   (let* ((scan (imoogi-scan-root root nil))
          (entries (plist-get scan :entries))
          (entry (car entries)))
     (should (= (length entries) 1))
     (should (equal (plist-get entry :direction) "-->"))
     (should (equal (plist-get entry :incremental) "yes")))))

;; AC-OPT-002d: ANKI_NOTE_TYPE still does not inherit. A child carrying card
;; options but no note type of its own is not a sync target at all -- this SPEC
;; changes nothing about the property that decides target-hood.
(ert-deftest imoogi-scan-test-note-type-still-does-not-inherit ()
  (imoogi-test--with-root
   (imoogi-test--write (expand-file-name "inherit.org" root)
                        (concat "* Parent\n:PROPERTIES:\n"
                                ":ANKI_NOTE_TYPE: imoogi-Cloze\n:END:\n"
                                "Parent body {{c1::x}}\n"
                                "** Child\n:PROPERTIES:\n:ANKI_SWIFT: t\n:END:\n"
                                "Child body\n"))
   (let* ((scan (imoogi-scan-root root nil))
          (entries (plist-get scan :entries)))
     (should (= (length entries) 1))
     (should (equal (plist-get (car entries) :title) "Parent")))))

;; AC-OPT-010's suppression case: an inherited option that would conflict is
;; switched off by an empty value in the heading's own drawer, so the entry
;; reaches the back end with a null swift and no conflict to report.
(ert-deftest imoogi-scan-test-inherited-option-is-suppressible-by-empty-value ()
  (imoogi-test--with-root
   (imoogi-test--write (expand-file-name "suppress.org" root)
                        (concat "#+PROPERTY: ANKI_SWIFT t\n"
                                "* Suppressed\n:PROPERTIES:\n"
                                ":ANKI_NOTE_TYPE: imoogi-Cloze\n"
                                ":ANKI_DIRECTION: ->\n:ANKI_SWIFT:\n:END:\n"
                                "Body {{c1::x}}\n"))
   (let* ((scan (imoogi-scan-root root nil))
          (entry (car (plist-get scan :entries))))
     (should (equal (plist-get entry :direction) "->"))
     (should (null (plist-get entry :swift))))))

(provide 'imoogi-scan-test)
;;; imoogi-scan-test.el ends here
