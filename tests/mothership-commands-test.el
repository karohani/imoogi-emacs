;;; mothership-commands-test.el --- Mothership workflows -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'org-id)

(defmacro imoogi-test--with-mothership (&rest body)
  "Run BODY with a real temporary Mothership and clean up visiting buffers."
  (declare (indent 0))
  `(let* ((home (file-truename (make-temp-file "imoogi-mothership-commands-" t)))
          (imoogi-mothership-directory (expand-file-name "brain/" home))
          (org-id-locations (make-hash-table :test #'equal))
          (org-id-files nil))
     (unwind-protect
         (save-window-excursion
           (imoogi-mothership-setup)
           ,@body)
       (dolist (buffer (buffer-list))
         (when-let* ((file (buffer-file-name buffer)))
           (when (file-in-directory-p file home)
             (with-current-buffer buffer (set-buffer-modified-p nil))
             (kill-buffer buffer))))
       (delete-directory home t))))

(ert-deftest imoogi-mothership-create-note-preserves-duplicates-and-registers-id ()
  (imoogi-test--with-mothership
    (imoogi-mothership-create-note "30. Permanent Notes" "기억과 학습")
    (let ((first buffer-file-name)
          (id (org-id-get nil))
          (text (buffer-string)))
      (should (file-in-directory-p first (expand-file-name "30. Permanent Notes" imoogi-mothership-directory)))
      (should (file-exists-p first))
      (should (stringp id))
      (should (file-equal-p (org-id-find-id-file id) first))
      (imoogi-mothership-create-note "30. Permanent Notes" "기억과 학습")
      (should-not (equal first buffer-file-name))
      (should-not (equal id (org-id-get nil)))
      (should (equal text (with-temp-buffer (insert-file-contents first) (buffer-string)))))))

(ert-deftest imoogi-mothership-create-note-rejects-unsafe-input ()
  (imoogi-test--with-mothership
    (dolist (title '("" "   " ".." "제목\n#+TITLE: 다른 제목" "../escape"))
      (should-error (imoogi-mothership-create-note "00. Inbox" title) :type 'user-error))
    (should-error (imoogi-mothership-create-note "../outside" "탈출") :type 'user-error)
    (should-not (directory-files (expand-file-name "00. Inbox" imoogi-mothership-directory) nil "^[^.]"))))

(ert-deftest imoogi-mothership-create-note-preserves-unsaved-visiting-buffer ()
  (imoogi-test--with-mothership
    (let* ((file (expand-file-name "00. Inbox/draft.org" imoogi-mothership-directory))
           (draft (find-file-noselect file)))
      (with-current-buffer draft (insert "unsaved draft"))
      (imoogi-mothership-create-note "00. Inbox" "draft")
      (should-not (equal buffer-file-name file))
      (should-not (file-exists-p file))
      (should (equal (with-current-buffer draft (buffer-string)) "unsaved draft")))))

(ert-deftest imoogi-mothership-create-note-rejects-outside-symlink ()
  (imoogi-test--with-mothership
    (let ((folder (expand-file-name "00. Inbox" imoogi-mothership-directory)))
      (delete-directory folder)
      (make-symbolic-link home folder)
      (should-error (imoogi-mothership-create-note "00. Inbox" "outside") :type 'user-error)
      (should-not (file-exists-p (expand-file-name "outside.org" home))))))

(ert-deftest imoogi-mothership-create-note-preserves-unsaved-alias-buffer ()
  (imoogi-test--with-mothership
    (let* ((alias (expand-file-name "alias" home))
           (file (expand-file-name "00. Inbox/draft.org" imoogi-mothership-directory))
           (find-file-visit-truename nil))
      (make-symbolic-link imoogi-mothership-directory alias)
      (let ((draft (find-file-noselect (expand-file-name "00. Inbox/draft.org" alias))))
        (with-current-buffer draft (insert "unsaved alias draft"))
        (imoogi-mothership-create-note "00. Inbox" "draft")
        (should-not (file-exists-p file))
        (should (equal (with-current-buffer draft (buffer-string)) "unsaved alias draft"))))))

(ert-deftest imoogi-mothership-open-rejects-outside-readme-symlink ()
  (imoogi-test--with-mothership
    (let ((outside (expand-file-name "outside.org" home))
          (readme (expand-file-name "README.org" imoogi-mothership-directory)))
      (with-temp-file outside (insert "outside private note"))
      (delete-file readme)
      (make-symbolic-link outside readme)
      (should-error (imoogi-mothership-open) :type 'user-error)
      (should-not (find-buffer-visiting outside)))))

(ert-deftest imoogi-mothership-capture-goes-to-inbox ()
  (imoogi-test--with-mothership
    (imoogi-mothership-capture "떠오른 생각")
    (should (file-in-directory-p buffer-file-name
                                (expand-file-name "00. Inbox" imoogi-mothership-directory)))))

(ert-deftest imoogi-mothership-open-and-browse-use-configured-root ()
  (imoogi-test--with-mothership
    (imoogi-mothership-open)
    (should (file-equal-p buffer-file-name (expand-file-name "README.org" imoogi-mothership-directory)))
    (imoogi-mothership-browse)
    (should (derived-mode-p 'dired-mode))
    (should (file-equal-p default-directory imoogi-mothership-directory))))

(ert-deftest imoogi-mothership-find-and-link-resolve-real-note ()
  (imoogi-test--with-mothership
    (imoogi-mothership-create-note "40. Docs" "사용 설명")
    (let ((file buffer-file-name)
          (id (org-id-get nil)))
      (cl-letf (((symbol-function 'completing-read)
                 (lambda (_prompt choices &rest _)
                   (should (member (file-relative-name file imoogi-mothership-directory) choices))
                   (file-relative-name file imoogi-mothership-directory))))
        (with-temp-buffer
          (imoogi-mothership-find-file)
          (should (file-equal-p buffer-file-name file)))
        (with-temp-buffer
          (org-mode)
          (imoogi-mothership-insert-link)
          (goto-char (point-min))
          (let ((link (org-element-context)))
            (should (equal (org-element-property :type link) "id"))
            (should (equal (org-element-property :path link) id))
            (should (file-equal-p (org-id-find-id-file id) file))))))))

(ert-deftest imoogi-mothership-link-adds-durable-id-to-existing-note ()
  (imoogi-test--with-mothership
    (let ((file (expand-file-name "40. Docs/legacy.org" imoogi-mothership-directory)))
      (with-temp-file file (insert "#+TITLE: 기존 노트\n\n내용\n"))
      (cl-letf (((symbol-function 'completing-read) (lambda (&rest _) "40. Docs/legacy.org")))
        (with-temp-buffer
          (org-mode)
          (imoogi-mothership-insert-link)
          (goto-char (point-min))
          (let ((id (org-element-property :path (org-element-context))))
            (should (stringp id))
            (with-temp-buffer
              (insert-file-contents file)
              (org-mode)
              (should (equal (org-id-get nil) id)))))))))

(ert-deftest imoogi-mothership-link-does-not-save-unsaved-target-changes ()
  (imoogi-test--with-mothership
    (let ((file (expand-file-name "40. Docs/dirty.org" imoogi-mothership-directory)))
      (with-temp-file file (insert "#+TITLE: 원본\n"))
      (with-current-buffer (find-file-noselect file)
        (goto-char (point-max))
        (insert "저장하지 않은 내용"))
      (cl-letf (((symbol-function 'completing-read) (lambda (&rest _) "40. Docs/dirty.org")))
        (with-temp-buffer
          (org-mode)
          (should-error (imoogi-mothership-insert-link) :type 'user-error)
          (should (string-empty-p (buffer-string)))))
      (should (equal (with-temp-buffer (insert-file-contents file) (buffer-string)) "#+TITLE: 원본\n")))))

(ert-deftest imoogi-mothership-find-excludes-outside-symlink ()
  (imoogi-test--with-mothership
    (let ((outside (expand-file-name "outside.org" home)))
      (with-temp-file outside (insert "외부 노트"))
      (make-symbolic-link outside (expand-file-name "40. Docs/outside.org" imoogi-mothership-directory))
      (cl-letf (((symbol-function 'completing-read)
                 (lambda (_prompt choices &rest _)
                   (should-not (member "40. Docs/outside.org" choices))
                   "README.org")))
        (imoogi-mothership-find-file)
        (should (file-equal-p buffer-file-name (expand-file-name "README.org" imoogi-mothership-directory)))))))

(ert-deftest imoogi-mothership-search-is-confined-to-root ()
  (skip-unless (executable-find "rg"))
  (imoogi-test--with-mothership
    (let ((inside (expand-file-name "40. Docs/inside.org" imoogi-mothership-directory)))
      (with-temp-file inside (insert "mothership-needle"))
      (with-temp-file (expand-file-name "outside.org" home) (insert "mothership-needle"))
      (cl-letf (((symbol-function 'consult-ripgrep)
                 (lambda (root &rest _)
                   (should (file-equal-p root imoogi-mothership-directory))
                   (let ((matches (process-lines "rg" "-l" "mothership-needle" root)))
                     (should (= (length matches) 1))
                     (should (file-equal-p (car matches) inside))))))
        (imoogi-mothership-search)))))

(ert-deftest imoogi-mothership-commands-require-initialized-root ()
  (let ((imoogi-mothership-directory "/missing/imoogi-mothership-root/"))
    (should-error (imoogi-mothership-open) :type 'user-error)
    (should-error (imoogi-mothership-create-note "00. Inbox" "메모") :type 'user-error)))

(ert-deftest imoogi-mothership-menu-is-main-and-org-roam-is-nested ()
  (should (eq (key-binding (kbd "C-c n")) #'imoogi-mothership-transient))
  (dolist (binding '(("o" . imoogi-mothership-open)
                     ("n" . imoogi-mothership-create-note)
                     ("c" . imoogi-mothership-capture)
                     ("f" . imoogi-mothership-find-file)
                     ("s" . imoogi-mothership-search)
                     ("i" . imoogi-mothership-insert-link)
                     ("d" . imoogi-mothership-browse)
                     ("S" . imoogi-mothership-setup)
                     ("r" . imoogi-org-roam-transient)))
    (should (eq (plist-get (cdr (transient-get-suffix 'imoogi-mothership-transient (car binding))) :command)
                (cdr binding)))))

(ert-deftest imoogi-mothership-menu-keys-enter-org-roam-submenu ()
  (unwind-protect
      (progn
        (execute-kbd-macro (kbd "C-c n"))
        (should (eq (oref transient--prefix command) 'imoogi-mothership-transient))
        (execute-kbd-macro (kbd "r"))
        (should (eq (oref transient--prefix command) 'imoogi-org-roam-transient)))
    (transient--stack-zap)))

;;; mothership-commands-test.el ends here
