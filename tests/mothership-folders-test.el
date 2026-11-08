;;; mothership-folders-test.el --- Sibling folder numbering -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)

(ert-deftest imoogi-mothership-folder-numbers-are-local-to-parent ()
  (imoogi-test--with-mothership
    (let* ((first (imoogi-mothership-create-folder "30. Permanent Notes" "인공지능"))
           (relative (file-relative-name first imoogi-mothership-directory))
           (child (imoogi-mothership-create-folder relative "언어모델"))
           (second (imoogi-mothership-create-folder "30. Permanent Notes" "통계")))
      (should (equal relative "30. Permanent Notes/01. 인공지능"))
      (should (equal (file-name-nondirectory child) "01. 언어모델"))
      (should (equal (file-name-nondirectory second) "02. 통계"))
      (should (file-directory-p child))
      (should (file-directory-p second)))))

(ert-deftest imoogi-mothership-folder-uses-next-number-without-renaming ()
  (imoogi-test--with-mothership
    (let ((parent (expand-file-name "40. Docs" imoogi-mothership-directory)))
      (make-directory (expand-file-name "01. 기존" parent))
      (make-directory (expand-file-name "03. 기존" parent))
      (with-temp-file (expand-file-name "05. 예약" parent) (insert "keep"))
      (should (equal (file-name-nondirectory (imoogi-mothership-create-folder "40. Docs" "새 폴더"))
                     "06. 새 폴더"))
      (should (file-directory-p (expand-file-name "01. 기존" parent)))
      (should (file-directory-p (expand-file-name "03. 기존" parent))))))

(ert-deftest imoogi-mothership-folder-descendants-are-note-destinations ()
  (imoogi-test--with-mothership
    (let* ((folder (imoogi-mothership-create-folder "20. Literature Notes" "논문"))
           (relative (file-relative-name folder imoogi-mothership-directory)))
      (should (member relative (imoogi-mothership--folders t)))
      (imoogi-mothership-create-note relative "논문 요약")
      (should (file-in-directory-p buffer-file-name folder))
      (let ((asset-folder (imoogi-mothership-create-folder "50. Assets" "이미지")))
        (should (member (file-relative-name asset-folder imoogi-mothership-directory)
                        (imoogi-mothership--folders)))
        (should-not (member (file-relative-name asset-folder imoogi-mothership-directory)
                            (imoogi-mothership--folders t)))))))

(ert-deftest imoogi-mothership-folder-discovers-manual-folders-and-renames ()
  (imoogi-test--with-mothership
    (let* ((parent (expand-file-name "30. Permanent Notes/01. 이전 이름" imoogi-mothership-directory))
           (renamed (expand-file-name "30. Permanent Notes/07. 바뀐 이름" imoogi-mothership-directory)))
      (make-directory (expand-file-name "01. 하위" parent) t)
      (rename-file parent renamed)
      (let ((relative "30. Permanent Notes/07. 바뀐 이름/01. 하위"))
        (should (member relative (imoogi-mothership--folders t)))
        (imoogi-mothership-create-note relative "새 노트")
        (should (file-in-directory-p buffer-file-name (expand-file-name "01. 하위" renamed)))))))

(ert-deftest imoogi-mothership-folder-rejects-invalid-input-and-outside-links ()
  (imoogi-test--with-mothership
    (make-symbolic-link home (expand-file-name "40. Docs/alias" imoogi-mothership-directory))
    (dolist (parent '("." "../" "40. Docs/alias"))
      (should-error (imoogi-mothership-create-folder parent "거부") :type 'user-error))
    (dolist (name '("" "  " ".." "../밖" "a/b" "a\\b" "a\nb" "01. 번호 입력"))
      (should-error (imoogi-mothership-create-folder "40. Docs" name) :type 'user-error))
    (should (equal (directory-files (expand-file-name "40. Docs" imoogi-mothership-directory)
                                   nil "^[^.]") '("alias")))))

(ert-deftest imoogi-mothership-folder-respects-existing-number-lock ()
  (imoogi-test--with-mothership
    (let ((lock (expand-file-name "40. Docs/.imoogi-folder-number.lock" imoogi-mothership-directory)))
      (make-directory lock)
      (should-error (imoogi-mothership-create-folder "40. Docs" "충돌") :type 'user-error)
      (should (file-directory-p lock))
      (should-not (file-exists-p (expand-file-name "40. Docs/01. 충돌" imoogi-mothership-directory))))))

(ert-deftest imoogi-mothership-folder-cleans-lock-after-creation-failure ()
  (imoogi-test--with-mothership
    (let ((mkdir (symbol-function 'make-directory))
          (lock (expand-file-name "40. Docs/.imoogi-folder-number.lock" imoogi-mothership-directory)))
      (cl-letf (((symbol-function 'make-directory)
                 (lambda (path &rest args)
                   (if (string-suffix-p "01. 실패" path)
                       (signal 'file-error '("injected failure"))
                     (apply mkdir path args)))))
        (should-error (imoogi-mothership-create-folder "40. Docs" "실패") :type 'file-error))
      (should-not (file-exists-p lock)))))

(ert-deftest imoogi-mothership-folder-command-is-reachable-and-opens-directory ()
  (imoogi-test--with-mothership
    (should (eq (plist-get (cdr (transient-get-suffix 'imoogi-mothership-transient "+")) :command)
                #'imoogi-mothership-create-folder))
    (cl-letf (((symbol-function 'completing-read) (lambda (&rest _) "40. Docs"))
              ((symbol-function 'read-string) (lambda (&rest _) "설명서")))
      (call-interactively #'imoogi-mothership-create-folder)
      (unwind-protect
          (progn
            (should (derived-mode-p 'dired-mode))
            (should (file-equal-p default-directory
                                 (expand-file-name "40. Docs/01. 설명서" imoogi-mothership-directory))))
        (kill-buffer (current-buffer))))))

(ert-deftest imoogi-mothership-folder-allows-ascii-and-numbers-over-99 ()
  (imoogi-test--with-mothership
    (make-directory (expand-file-name "40. Docs/99. old" imoogi-mothership-directory))
    (should (equal (file-name-nondirectory (imoogi-mothership-create-folder "40. Docs" "notes"))
                   "100. notes"))))

(ert-deftest imoogi-mothership-folder-cleans-lock-on-quit ()
  (imoogi-test--with-mothership
    (let ((mkdir (symbol-function 'make-directory))
          (lock (expand-file-name "40. Docs/.imoogi-folder-number.lock" imoogi-mothership-directory))
          interrupted)
      (cl-letf (((symbol-function 'make-directory)
                 (lambda (path &rest args)
                   (if (string-suffix-p "01. quit" path)
                       (signal 'quit nil)
                     (apply mkdir path args)))))
        (condition-case nil
            (imoogi-mothership-create-folder "40. Docs" "quit")
          (quit (setq interrupted t))))
      (should interrupted)
      (should-not (file-exists-p lock)))))

(ert-deftest imoogi-mothership-folder-cleans-lock-on-acquisition-quit ()
  (imoogi-test--with-mothership
    (let ((mkdir (symbol-function 'make-directory))
          (lock (expand-file-name "40. Docs/.imoogi-folder-number.lock" imoogi-mothership-directory))
          (quit-flag nil)
          interrupted)
      (cl-letf (((symbol-function 'make-directory)
                 (lambda (path &rest args)
                   (prog1 (apply mkdir path args)
                     (when (equal path lock) (setq quit-flag t))))))
        (condition-case nil
            (imoogi-mothership-create-folder "40. Docs" "cancelled")
          (quit (setq interrupted t))))
      (should interrupted)
      (should-not (file-exists-p lock)))))

;;; mothership-folders-test.el ends here
