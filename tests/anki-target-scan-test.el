;;; anki-target-scan-test.el --- Tests for imoogi-target-scan.el -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'imoogi-target-scan)

(defun imoogi-target-scan-test--write (path content)
  "Write CONTENT to PATH, creating parent directories as needed."
  (make-directory (file-name-directory path) t)
  (with-temp-file path (insert content)))

(defmacro imoogi-target-scan-test--with-root (&rest body)
  "Run BODY with ROOT bound to a fresh temp directory."
  (declare (indent 0))
  `(let ((root (make-temp-file "imoogi-target-scan-test" t)))
     (unwind-protect
         (progn ,@body)
       (delete-directory root t))))

(defun imoogi-target-scan-test--scan-for-root (groups root)
  "Return the scan plist for ROOT in GROUPS."
  (plist-get
   (cl-find (file-name-as-directory (file-truename root)) groups
            :key (lambda (group) (plist-get group :root))
            :test #'string=)
   :scan))

(defun imoogi-target-scan-test--entry-paths (scan)
  "Return source paths from SCAN entries."
  (mapcar (lambda (entry) (plist-get entry :source-path))
          (plist-get scan :entries)))

(defun imoogi-target-scan-test--census-paths (scan)
  "Return source paths from SCAN census."
  (mapcar (lambda (entry) (plist-get entry :source-path))
          (plist-get scan :census)))

(ert-deftest imoogi-target-scan-test-legacy-dirs-and-files ()
  (imoogi-target-scan-test--with-root
    (let ((legacy (expand-file-name "legacy" root))
          (dir-a (expand-file-name "dir-a" root))
          (dir-b (expand-file-name "dir-b" root))
          (single (expand-file-name "single.org" root)))
      (imoogi-target-scan-test--write
       (expand-file-name "l.org" legacy)
       "* L\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (imoogi-target-scan-test--write
       (expand-file-name "nested/a.org" dir-a)
       "* A\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (imoogi-target-scan-test--write
       (expand-file-name "b.org" dir-b)
       "* B\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (imoogi-target-scan-test--write
       single
       "* S\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (let* ((groups (imoogi-target-scan
                      legacy
                      (list (list :kind 'directory :path dir-a)
                            (list :kind 'directory :path dir-b)
                            (list :kind 'file :path single))
                      nil))
             (legacy-scan (imoogi-target-scan-test--scan-for-root groups legacy))
             (dir-a-scan (imoogi-target-scan-test--scan-for-root groups dir-a))
             (dir-b-scan (imoogi-target-scan-test--scan-for-root groups dir-b))
             (single-scan (imoogi-target-scan-test--scan-for-root
                           groups (file-name-directory single))))
        (should (= (length groups) 4))
        (should (plist-get (car groups) :legacy))
        (should (equal (imoogi-target-scan-test--entry-paths legacy-scan)
                       '("l.org")))
        (should (equal (imoogi-target-scan-test--entry-paths dir-a-scan)
                       '("nested/a.org")))
        (should (equal (imoogi-target-scan-test--entry-paths dir-b-scan)
                       '("b.org")))
        (should (equal (imoogi-target-scan-test--entry-paths single-scan)
                       '("single.org")))
        (should (cl-every (lambda (group)
                            (plist-get (plist-get group :scan) :scan-complete))
                          groups))))))

(ert-deftest imoogi-target-scan-test-dedups-overlaps-and-aliases ()
  (imoogi-target-scan-test--with-root
    (let ((dir (expand-file-name "cards" root))
          (alias (expand-file-name "alias.org" root)))
      (imoogi-target-scan-test--write
       (expand-file-name "same.org" dir)
       "* Same\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ANKI_NOTE_ID: 77\n:END:\n")
      (make-symbolic-link (expand-file-name "same.org" dir) alias)
      (let* ((groups (imoogi-target-scan
                      dir
                      (list (list :kind 'file :path alias)
                            (list :kind 'file :path (expand-file-name "same.org" dir)))
                      nil))
             (entry-count (apply #'+
                                 (mapcar (lambda (group)
                                           (length (plist-get
                                                    (plist-get group :scan)
                                                    :entries)))
                                         groups)))
             (legacy-scan (plist-get (car groups) :scan)))
        (should (= entry-count 1))
        (should (equal (imoogi-target-scan-test--entry-paths legacy-scan)
                       '("same.org")))
        (should (= (length (plist-get legacy-scan :census)) 1))))))

(ert-deftest imoogi-target-scan-test-same-basename-in-different-groups ()
  (imoogi-target-scan-test--with-root
    (let ((a (expand-file-name "a" root))
          (b (expand-file-name "b" root)))
      (imoogi-target-scan-test--write
       (expand-file-name "note.org" a)
       "* A\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (imoogi-target-scan-test--write
       (expand-file-name "note.org" b)
       "* B\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (let* ((groups (imoogi-target-scan
                      nil
                      (list (list :kind 'directory :path a)
                            (list :kind 'directory :path b))
                      nil))
             (a-scan (imoogi-target-scan-test--scan-for-root groups a))
             (b-scan (imoogi-target-scan-test--scan-for-root groups b)))
        (should (equal (imoogi-target-scan-test--entry-paths a-scan)
                       '("note.org")))
        (should (equal (imoogi-target-scan-test--entry-paths b-scan)
                       '("note.org")))
        (should (equal (mapcar (lambda (entry) (plist-get entry :title))
                               (plist-get a-scan :entries))
                       '("A")))
        (should (equal (mapcar (lambda (entry) (plist-get entry :title))
                               (plist-get b-scan :entries))
                       '("B")))))))

(ert-deftest imoogi-target-scan-test-missing-targets-mark-global-incomplete ()
  (imoogi-target-scan-test--with-root
    (let ((ok (expand-file-name "ok" root))
          (missing-dir (expand-file-name "missing" root))
          (missing-file (expand-file-name "gone.org" root)))
      (imoogi-target-scan-test--write
       (expand-file-name "ok.org" ok)
       "* OK\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (let ((groups (imoogi-target-scan
                     nil
                     (list (list :kind 'directory :path ok)
                           (list :kind 'directory :path missing-dir)
                           (list :kind 'file :path missing-file))
                     nil)))
        (should (= (length groups) 3))
        (should-not (cl-some (lambda (group)
                               (plist-get (plist-get group :scan) :scan-complete))
                             groups))
        (should (= (apply #'+
                          (mapcar (lambda (group)
                                    (length (plist-get
                                             (plist-get group :scan)
                                             :unreadable-files)))
                                  groups))
                   2))
        (should (equal (imoogi-target-scan-test--entry-paths
                        (imoogi-target-scan-test--scan-for-root groups ok))
                       '("ok.org")))))))

(ert-deftest imoogi-target-scan-test-directory-symlink-cycle-is-not-revisited ()
  (imoogi-target-scan-test--with-root
    (let ((dir (expand-file-name "cards" root)))
      (imoogi-target-scan-test--write
       (expand-file-name "a.org" dir)
       "* A\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (make-symbolic-link dir (expand-file-name "loop" dir))
      (let* ((groups (imoogi-target-scan
                      nil
                      (list (list :kind 'directory :path dir))
                      nil))
             (scan (imoogi-target-scan-test--scan-for-root groups dir)))
        (should (plist-get scan :scan-complete))
        (should (equal (imoogi-target-scan-test--entry-paths scan)
                       '("a.org")))))))

(ert-deftest imoogi-target-scan-test-explicit-hidden-file-in-covered-directory ()
  (imoogi-target-scan-test--with-root
    (let ((dir (expand-file-name "cards" root))
          (hidden (expand-file-name ".hidden.org" root)))
      (imoogi-target-scan-test--write
       (expand-file-name "visible.org" dir)
       "* V\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (imoogi-target-scan-test--write
       hidden
       "* H\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (let* ((groups (imoogi-target-scan
                      root
                      (list (list :kind 'file :path hidden))
                      nil))
             (scan (plist-get (car groups) :scan)))
        (should (= (length groups) 1))
        (should (equal (imoogi-target-scan-test--entry-paths scan)
                       '("cards/visible.org" ".hidden.org")))))))

(ert-deftest imoogi-target-scan-test-file-symlink-outside-root-is-incomplete ()
  (imoogi-target-scan-test--with-root
    (let ((dir (expand-file-name "cards" root))
          (outside (expand-file-name "outside.org" root))
          (link (expand-file-name "cards/link.org" root)))
      (imoogi-target-scan-test--write
       outside
       "* O\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\n")
      (make-directory dir t)
      (make-symbolic-link outside link)
      (let* ((groups (imoogi-target-scan
                      nil
                      (list (list :kind 'directory :path dir))
                      nil))
             (scan (plist-get (car groups) :scan)))
        (should-not (plist-get scan :scan-complete))
        (should (equal (plist-get scan :entries) nil))
        (should (member link (plist-get scan :unreadable-files)))))))

(ert-deftest imoogi-target-scan-test-excluded-files-remain-in-global-census ()
  (imoogi-target-scan-test--with-root
    (let ((dir (expand-file-name "cards" root)))
      (imoogi-target-scan-test--write
       (expand-file-name "live.org" dir)
       "* Live\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ANKI_NOTE_ID: 1\n:END:\n")
      (imoogi-target-scan-test--write
       (expand-file-name "archive/old.org" dir)
       "* Old\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ANKI_NOTE_ID: 2\n:END:\n")
      (let* ((groups (imoogi-target-scan
                      nil
                      (list (list :kind 'directory :path dir))
                      '("archive/")))
             (scan (imoogi-target-scan-test--scan-for-root groups dir)))
        (should (equal (imoogi-target-scan-test--entry-paths scan)
                       '("live.org")))
        (should (equal (sort (mapcar (lambda (entry) (plist-get entry :note-id))
                                     (plist-get scan :census))
                             #'<)
                       '(1 2)))))))

(ert-deftest imoogi-target-scan-test-crossgroup-census-is-shared ()
  (imoogi-target-scan-test--with-root
    (let ((a (expand-file-name "a" root))
          (b (expand-file-name "b" root)))
      (imoogi-target-scan-test--write
       (expand-file-name "a.org" a)
       "* A\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ANKI_NOTE_ID: 42\n:END:\n")
      (imoogi-target-scan-test--write
       (expand-file-name "b.org" b)
       "* B\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ANKI_NOTE_ID: 42\n:END:\n")
      (let* ((groups (imoogi-target-scan
                      nil
                      (list (list :kind 'directory :path a)
                            (list :kind 'directory :path b))
                      nil))
             (a-scan (imoogi-target-scan-test--scan-for-root groups a))
             (b-scan (imoogi-target-scan-test--scan-for-root groups b)))
        (should (= (length (plist-get a-scan :census)) 2))
        (should (= (length (plist-get b-scan :census)) 2))
        (should (member "a.org" (imoogi-target-scan-test--census-paths a-scan)))
        (should (member "../b/b.org" (imoogi-target-scan-test--census-paths a-scan)))
        (should (member "../a/a.org" (imoogi-target-scan-test--census-paths b-scan)))
        (should (member "b.org" (imoogi-target-scan-test--census-paths b-scan)))))))

(ert-deftest imoogi-target-scan-test-directory-symlink-outside-root-is-incomplete ()
  (imoogi-target-scan-test--with-root
    (let ((dir (expand-file-name "cards" root))
          (outside (expand-file-name "outside" root)))
      (make-directory dir t)
      (imoogi-target-scan-test--write
       (expand-file-name "old.org" outside)
       "* Old\n:PROPERTIES:\n:ANKI_NOTE_ID: 42\n:END:\n")
      (make-symbolic-link outside (expand-file-name "linked" dir))
      (let ((scan (plist-get (car (imoogi-target-scan dir nil nil)) :scan)))
        (should-not (plist-get scan :scan-complete))
        (should (plist-get scan :unreadable-files))))))

(provide 'anki-target-scan-test)
;;; anki-target-scan-test.el ends here
