;;; anki-targets-test.el --- Tests for imoogi-targets.el -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'imoogi-targets)

(defmacro imoogi-targets-test--with-home (&rest body)
  "Run BODY with isolated target config and files under a temp dir."
  (declare (indent 0))
  `(let* ((root (make-temp-file "imoogi-targets-test" t))
          (imoogi-targets-file (expand-file-name "config/targets.json" root))
          (imoogi-sync-root nil))
     (unwind-protect
         (progn ,@body)
       (delete-directory root t))))

(defun imoogi-targets-test--write (path content)
  "Write CONTENT to PATH, creating parent directories."
  (make-directory (file-name-directory path) t)
  (with-temp-file path
    (insert content)))

(ert-deftest imoogi-targets-test-roundtrip-restart ()
  (imoogi-targets-test--with-home
    (let ((dir (expand-file-name "deck" root))
          (file (expand-file-name "one.org" root)))
      (make-directory dir)
      (imoogi-targets-test--write file "* Card\n")
      (imoogi-anki-register-directory dir)
      (imoogi-anki-register-file file)
      (let ((loaded (imoogi-targets-load)))
        (should (equal loaded
                       (list (list :kind 'directory :path (file-truename dir))
                             (list :kind 'file :path (file-truename file))))))
      (let ((imoogi-targets-file imoogi-targets-file))
        (should (equal (imoogi-targets-load)
                       (list (list :kind 'directory :path (file-truename dir))
                             (list :kind 'file :path (file-truename file)))))))))

(ert-deftest imoogi-targets-test-duplicate-canonical-symlink-preserves-order ()
  (imoogi-targets-test--with-home
    (let* ((real (expand-file-name "real" root))
           (link (expand-file-name "link" root))
           (file (expand-file-name "card.org" real)))
      (make-directory real)
      (make-symbolic-link real link)
      (imoogi-targets-test--write file "* Card\n")
      (imoogi-anki-register-directory real)
      (imoogi-anki-register-directory link)
      (imoogi-anki-register-file file)
      (should (equal (imoogi-targets-load)
                     (list (list :kind 'directory :path (file-truename real))
                           (list :kind 'file :path (file-truename file))))))))

(ert-deftest imoogi-targets-test-directory-trailing-slash-dedups ()
  (imoogi-targets-test--with-home
    (let ((dir (expand-file-name "deck" root)))
      (make-directory dir)
      (imoogi-anki-register-directory dir)
      (imoogi-anki-register-directory (file-name-as-directory dir))
      (should (equal (imoogi-targets-load)
                     (list (list :kind 'directory
                                 :path (directory-file-name
                                        (file-truename dir)))))))))

(ert-deftest imoogi-targets-test-unregister-removes-registration-only ()
  (imoogi-targets-test--with-home
    (let ((file (expand-file-name "card.org" root)))
      (imoogi-targets-test--write file "* Card\n")
      (imoogi-anki-register-file file)
      (imoogi-anki-unregister-target
       (format "file %s" (file-truename file)))
      (should (file-exists-p file))
      (should (null (imoogi-targets-load))))))

(ert-deftest imoogi-targets-test-malformed-config-stops ()
  (imoogi-targets-test--with-home
    (imoogi-targets-test--write imoogi-targets-file "{")
    (should-error (imoogi-targets-load) :type 'user-error)))

(ert-deftest imoogi-targets-test-missing-or-null-targets-key-stops ()
  (imoogi-targets-test--with-home
    (imoogi-targets-test--write imoogi-targets-file "{}")
    (should-error (imoogi-targets-load) :type 'user-error)
    (imoogi-targets-test--write imoogi-targets-file "{\"targets\":null}")
    (should-error (imoogi-targets-load) :type 'user-error)))

(ert-deftest imoogi-targets-test-invalid-files-are-rejected ()
  (imoogi-targets-test--with-home
    (let ((txt (expand-file-name "card.txt" root))
          (missing (expand-file-name "missing.org" root))
          (dir (expand-file-name "dir" root)))
      (imoogi-targets-test--write txt "x")
      (make-directory dir)
      (should-error (imoogi-anki-register-file txt) :type 'user-error)
      (should-error (imoogi-anki-register-file missing) :type 'user-error)
      (should-error (imoogi-anki-register-file dir) :type 'user-error)
      (should-error (imoogi-targets--add 'bogus txt) :type 'user-error)
      (should-error (imoogi-targets--add 'file "/ssh:host:/tmp/card.org")
                    :type 'user-error))))

(ert-deftest imoogi-targets-test-missing-saved-paths-still-load ()
  (imoogi-targets-test--with-home
    (imoogi-targets-test--write
     imoogi-targets-file
     (json-serialize
      (list (cons 'targets
                  (vector (list (cons 'kind "file")
                                (cons 'path (expand-file-name "gone.org" root))))))))
    (should (equal (imoogi-targets-load)
                   (list (list :kind 'file
                               :path (expand-file-name "gone.org" root)))))))

(ert-deftest imoogi-targets-test-missing-saved-paths-do-not-block-new-registration ()
  (imoogi-targets-test--with-home
    (let ((new (expand-file-name "new.org" root)))
      (imoogi-targets-test--write
       imoogi-targets-file
       (json-serialize
        (list (cons 'targets
                    (vector (list (cons 'kind "file")
                                  (cons 'path (expand-file-name "gone.org" root))))))))
      (imoogi-targets-test--write new "* New\n")
      (imoogi-anki-register-file new)
      (should (equal (imoogi-targets-load)
                     (list (list :kind 'file
                                 :path (expand-file-name "gone.org" root))
                           (list :kind 'file
                                 :path (file-truename new))))))))

(ert-deftest imoogi-targets-test-failed-write-keeps-existing-config ()
  (imoogi-targets-test--with-home
    (let ((first (expand-file-name "first.org" root))
          (second (expand-file-name "second.org" root)))
      (imoogi-targets-test--write first "* First\n")
      (imoogi-targets-test--write second "* Second\n")
      (imoogi-anki-register-file first)
      (let ((before (with-temp-buffer
                      (insert-file-contents imoogi-targets-file)
                      (buffer-string))))
        (cl-letf (((symbol-function 'rename-file)
                   (lambda (&rest _args)
                     (signal 'file-error '("rename failed")))))
          (should-error (imoogi-anki-register-file second)
                        :type 'file-error))
        (should (equal before
                       (with-temp-buffer
                         (insert-file-contents imoogi-targets-file)
                         (buffer-string))))
        (should (equal (imoogi-targets-load)
                       (list (list :kind 'file
                                   :path (file-truename first)))))))))

(ert-deftest imoogi-targets-test-file-list-expands-deduplicates-and-refreshes ()
  (imoogi-targets-test--with-home
    (let* ((dir (expand-file-name "deck" root))
           (file (expand-file-name "nested/plain.org" dir))
           (excluded (expand-file-name "skip.org" dir))
           (external (expand-file-name "single.org" root))
           (imoogi-exclude-patterns '("skip.org")))
      (dolist (path (list file excluded external))
        (imoogi-targets-test--write path "* No card properties\n"))
      (imoogi-anki-register-directory dir)
      (imoogi-anki-register-file file)
      (imoogi-anki-register-file external)
      (cl-letf (((symbol-function 'pop-to-buffer) #'ignore))
        (unwind-protect
            (progn
              (imoogi-anki-list-files)
              (with-current-buffer "*imoogi Anki Files*"
                (should buffer-read-only)
                (should (eq (key-binding (kbd "g")) #'imoogi-anki-list-files))
                (should (string-match-p "총 3개 파일" (buffer-string)))
                (goto-char (point-min))
                (search-forward "[제외] ")
                (should (equal (button-get (button-at (point)) 'file)
                               (file-truename excluded)))
                (goto-char (point-min))
                (search-forward (file-truename external))
                (cl-letf (((symbol-function 'find-file-other-window)
                           (lambda (path) (should (equal path (file-truename external))))))
                  (button-activate (button-at (1- (point))))))
              (imoogi-targets-test--write (expand-file-name "new.org" dir) "* New\n")
              (imoogi-anki-list-files)
              (with-current-buffer "*imoogi Anki Files*"
                (should (string-match-p "총 4개 파일" (buffer-string)))))
          (when (get-buffer "*imoogi Anki Files*")
            (kill-buffer "*imoogi Anki Files*")))))))

(ert-deftest imoogi-targets-test-file-list-includes-legacy-and-missing-path ()
  (imoogi-targets-test--with-home
    (let* ((imoogi-sync-root (expand-file-name "legacy" root))
           (missing (expand-file-name "missing.org" root)))
      (imoogi-targets-test--write (expand-file-name "plain.org" imoogi-sync-root) "* Plain\n")
      (imoogi-targets--write (list (list :kind 'file :path missing)))
      (cl-letf (((symbol-function 'pop-to-buffer) #'ignore))
        (unwind-protect
            (progn
              (imoogi-anki-list-files)
              (with-current-buffer "*imoogi Anki Files*"
                (should (string-match-p "기본 폴더:" (buffer-string)))
                (should (string-match-p "총 1개 파일" (buffer-string)))
                (should (string-match-p "확인 실패" (buffer-string)))
                (should (string-match-p (regexp-quote missing) (buffer-string)))))
          (when (get-buffer "*imoogi Anki Files*")
            (kill-buffer "*imoogi Anki Files*")))))))

(provide 'anki-targets-test)
;;; anki-targets-test.el ends here
