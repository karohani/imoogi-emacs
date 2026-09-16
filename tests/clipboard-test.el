;;; clipboard-test.el --- clipboard integration tests -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'dired)
(require 'org)
(require 'imoogi-clipboard)

(ert-deftest imoogi-clipboard-native-text-yank ()
  (with-temp-buffer
    (org-mode)
    (let ((kill-ring '("native text")))
      (cl-letf (((symbol-function 'imoogi-clipboard--call-sync)
                 (lambda (&rest _) '((status . "ok") (kind . "text")))))
        (imoogi-clipboard-yank nil)
        (should (equal (buffer-string) "native text"))))))

(ert-deftest imoogi-clipboard-prefix-bypasses-helper ()
  (with-temp-buffer
    (org-mode)
    (let ((kill-ring '("bypass")) (called nil))
      (cl-letf (((symbol-function 'imoogi-clipboard--call-sync)
                 (lambda (&rest _) (setq called t))))
        (imoogi-clipboard-yank '(4))
        (should-not called)
        (should (equal (buffer-string) "bypass"))))))

(ert-deftest imoogi-clipboard-unknown-inspect-falls-back ()
  (with-temp-buffer
    (org-mode)
    (let ((kill-ring '("fallback")))
      (cl-letf (((symbol-function 'imoogi-clipboard--call-sync)
                 (lambda (&rest _) nil)))
        (imoogi-clipboard-yank nil)
        (should (equal (buffer-string) "fallback"))))))

(ert-deftest imoogi-clipboard-external-image-starts-import ()
  (with-temp-buffer
    (org-mode)
    (let (request)
      (cl-letf (((symbol-function 'imoogi-clipboard--call-sync)
                 (lambda (&rest _)
                   '((status . "ok") (kind . "image")
                     (clipboard_id . "macos:2"))))
                ((symbol-function 'imoogi-clipboard--call-async)
                 (lambda (value _callback) (setq request value))))
        (setq imoogi-clipboard--kill-checkpoint
              '((clipboard_id . "macos:1") (kill_generation . 1))
              imoogi-clipboard--kill-generation 1)
        (imoogi-clipboard-yank nil)
        (should (equal (alist-get 'operation request) "paste"))
        (should (equal (alist-get 'expected_clipboard_id request) "macos:2"))))))

(ert-deftest imoogi-clipboard-missing-checkpoint-falls-back-to-native-yank ()
  (with-temp-buffer
    (org-mode)
    (let ((kill-ring '("safe fallback"))
	  (imoogi-clipboard--kill-checkpoint nil))
      (cl-letf (((symbol-function 'imoogi-clipboard--call-sync)
		 (lambda (&rest _)
		   '((status . "ok") (kind . "image")
		     (clipboard_id . "macos:2")))))
	(imoogi-clipboard-yank nil)
	(should (equal (buffer-string) "safe fallback"))))))

(ert-deftest imoogi-clipboard-folder-dnd-is-rejected ()
  (with-temp-buffer
    (org-mode)
    (let ((directory (make-temp-file "imoogi-folder" t)) (called nil))
      (unwind-protect
          (cl-letf (((symbol-function 'imoogi-clipboard--start-import)
                     (lambda (&rest _) (setq called t))))
            (imoogi-clipboard--dnd-handler (concat "file://" directory) 'copy)
            (should-not called))
        (delete-directory directory)))))

(ert-deftest imoogi-clipboard-stale-completion-does-not-insert ()
  (with-temp-buffer
    (org-mode)
    (let (callback)
      (cl-letf (((symbol-function 'imoogi-clipboard--call-async)
                 (lambda (_request value) (setq callback value))))
        (imoogi-clipboard--start-import "import" nil '("/tmp/a"))
	(insert "changed")
        (funcall callback
                 '((status . "ok")
                   (assets . (((id . "x") (path . "/tmp/a"))))))
	(should (equal (buffer-string) "changed"))))))

(ert-deftest imoogi-clipboard-preview-map-is-exact ()
  (with-temp-buffer
    (org-mode)
    (setq imoogi-clipboard--transactions
          '(((staged . t)
             (assets . (((id . "one") (path . "/tmp/one.png"))
                        ((id . "two") (path . "/tmp/two.png")))))))
    (should (equal (imoogi-clipboard-preview-asset-map)
                   '(("imoogi-asset:two" . "/tmp/two.png")
                     ("imoogi-asset:one" . "/tmp/one.png"))))))

(ert-deftest imoogi-clipboard-links-escape-special-targets ()
  (with-temp-buffer
    (org-mode)
    (let ((buffer-file-name "/tmp/note.org"))
      (should
	(equal (imoogi-clipboard--asset-link
		'((id . "a") (path . "/tmp/note.assets/한 글[1](x)%25.png")))
	       "[[file:note.assets/%ED%95%9C%20%EA%B8%80%5B1%5D%28x%29%2525.png][한 글_1__x_%25.png]]"))))
  (with-temp-buffer
    (setq major-mode 'markdown-mode)
    (let ((buffer-file-name "/tmp/note.md"))
      (should (string-match-p
	       "note.assets/.*%20.*%5B1%5D%28x%29%2525.png"
	       (imoogi-clipboard--asset-link
	       '((id . "a") (path . "/tmp/note.assets/한 글[1](x)%25.png"))))))))

(ert-deftest imoogi-clipboard-identity-detects-owner-generation-change ()
  (should (imoogi-clipboard--same-identity-p
           '((session . "s") (buffer . "b") (generation . 1))
           '((session . "s") (buffer . "b") (generation . 1))))
  (should-not (imoogi-clipboard--same-identity-p
               '((session . "s") (buffer . "b") (generation . 1))
               '((session . "s") (buffer . "b") (generation . 2)))))

(ert-deftest imoogi-clipboard-owner-uses-project-notes-registry-fallback ()
  (with-temp-buffer
    (let* ((root (make-temp-file "imoogi-project-owner" t))
           (buffer-file-name (expand-file-name "concepts/topic.org" root)))
      (make-directory (file-name-directory buffer-file-name) t)
      (cl-letf (((symbol-function 'imoogi-project-notes--read-registry)
                 (lambda () `(((key . "project-key") (notes-dir . ,root)))))
                ((symbol-function 'imoogi-project-notes--current-entry)
                 (lambda () `((key . "project-key") (notes-dir . ,root)))))
        (let ((owner (imoogi-clipboard--owner)))
          (should (equal (alist-get 'kind owner) "project_notes"))
          (should (equal (alist-get 'root owner) (expand-file-name root)))
          (should (equal (alist-get 'registry_key owner) "project-key")))))))

(ert-deftest imoogi-clipboard-owner-rejects-overlapping-project-note-roots ()
  (with-temp-buffer
    (let* ((root (make-temp-file "imoogi-project-overlap" t))
           (nested (expand-file-name "nested" root))
           (buffer-file-name (expand-file-name "topic.org" nested)))
      (make-directory nested t)
      (cl-letf (((symbol-function 'imoogi-project-notes--read-registry)
                 (lambda () `(((key . "outer") (notes-dir . ,root))
                              ((key . "inner") (notes-dir . ,nested))))))
        (should-error (imoogi-clipboard--owner) :type 'user-error)))))

(ert-deftest imoogi-clipboard-encoded-file-target-round-trips ()
  (let* ((relative "note.assets/한 글[1].png")
         (encoded (imoogi-clipboard--link-target relative)))
    (should (equal (imoogi-clipboard--decode-link-target encoded) relative))))

(ert-deftest imoogi-clipboard-save-coordinator-rewrites-and-commits ()
  (with-temp-buffer
    (org-mode)
    (let* ((directory (make-temp-file "imoogi-save" t))
           (buffer-file-name (expand-file-name "note.org" directory))
           (asset-path (expand-file-name "note.assets/screen.png" directory))
           (imoogi-clipboard--transactions
            '(((transaction_id . "tx") (transaction_token . "token")
               (identity . ((session . "s") (buffer . "b") (generation . 1)))
               (staged . t) (assets . (((id . "asset")))))))
           operations)
      (insert "[[file:imoogi-asset:asset][screen]]")
      (unwind-protect
          (cl-letf (((symbol-function 'imoogi-clipboard--call-sync)
                     (lambda (request _timeout)
                       (push (alist-get 'operation request) operations)
                       (pcase (alist-get 'operation request)
                         ("finalize"
                          `((status . "ok") (state . "PREPARED")
                            (assets . (((id . "asset") (path . ,asset-path))))))
                         (_ '((status . "ok")))))))
            (imoogi-clipboard--save-around (lambda (&rest _) 'saved))
            (should (string-match-p "note\\.assets/screen\\.png" (buffer-string)))
            (should-not imoogi-clipboard--transactions)
            (should (equal (nreverse operations)
                           '("finalize" "document-saved" "commit"))))
        (delete-directory directory t)))))

(ert-deftest imoogi-clipboard-save-failure-keeps-prepared-transaction-for-retry ()
  (with-temp-buffer
    (org-mode)
    (let* ((directory (make-temp-file "imoogi-save-failure" t))
	   (buffer-file-name (expand-file-name "note.org" directory))
	   (asset-path (expand-file-name "note.assets/screen.png" directory))
	   (imoogi-clipboard--transactions
	    '(((transaction_id . "tx") (transaction_token . "token")
	       (identity . ((session . "s") (buffer . "b") (generation . 1)))
	       (staged . t) (assets . (((id . "asset")))))))
	   operations)
      (insert "[[file:imoogi-asset:asset][screen]]")
      (unwind-protect
	  (cl-letf (((symbol-function 'imoogi-clipboard--call-sync)
		     (lambda (request _timeout)
		       (push (alist-get 'operation request) operations)
		       `((status . "ok") (state . "PREPARED")
			 (assets . (((id . "asset") (path . ,asset-path))))))))
	    (should-error
	     (imoogi-clipboard--save-around (lambda (&rest _) (error "disk full"))))
	    (should imoogi-clipboard--transactions)
	    (should (equal operations '("finalize")))
	    (should (string-match-p "imoogi-asset:asset" (buffer-string))))
	(delete-directory directory t)))))

(ert-deftest imoogi-clipboard-lost-commit-response-converges-on-next-save ()
  (with-temp-buffer
    (org-mode)
    (let* ((directory (make-temp-file "imoogi-lost-commit" t))
           (buffer-file-name (expand-file-name "note.org" directory))
           (asset-path (expand-file-name "note.assets/screen.png" directory))
           (imoogi-clipboard--transactions
            '(((transaction_id . "tx") (transaction_token . "token")
               (identity . ((session . "s") (buffer . "b") (generation . 1)))
               (staged . t) (assets . (((id . "asset")))))))
           operations)
      (insert "[[file:note.assets/screen.png][screen]]")
      (unwind-protect
          (cl-letf (((symbol-function 'imoogi-clipboard--call-sync)
                     (lambda (request _timeout)
                       (push (alist-get 'operation request) operations)
                       (pcase (alist-get 'operation request)
                         ("finalize" `((status . "ok") (state . "COMMITTED")
                                       (assets . (((id . "asset") (path . ,asset-path))))))
                         (_ (error "terminal lifecycle command must not be retried"))))))
            (imoogi-clipboard--save-around (lambda (&rest _) 'saved))
            (should-not imoogi-clipboard--transactions)
            (should (equal operations '("finalize"))))
        (delete-directory directory t)))))

(ert-deftest imoogi-clipboard-visited-rename-copies-and-relinks-assets ()
  (with-temp-buffer
    (org-mode)
    (let* ((directory (make-temp-file "imoogi-rename" t))
           (old-document (expand-file-name "old.org" directory))
           (new-document (expand-file-name "new.org" directory))
           (old-assets (expand-file-name "old.assets" directory))
           (old-asset (expand-file-name "screen.png" old-assets))
           (new-asset (expand-file-name "new.assets/screen.png" directory)))
      (make-directory old-assets)
      (write-region "png" nil old-asset nil 'silent)
      (setq buffer-file-name old-document)
      (insert "[[file:old.assets/screen.png][screen]]")
      (unwind-protect
          (cl-letf (((symbol-function 'imoogi-clipboard--call-sync)
                     (lambda (_request _timeout)
                       `((status . "ok")
                         (assets . (((id . "asset") (path . ,new-asset))))))))
            (imoogi-clipboard--set-visited-around
             (lambda (filename &rest _) (setq buffer-file-name filename))
             new-document)
            (should (string-match-p "new\\.assets/screen\\.png" (buffer-string))))
        (delete-directory directory t)))))

(ert-deftest imoogi-clipboard-project-note-save-as-becomes-standalone ()
  (with-temp-buffer
    (org-mode)
    (let* ((project (make-temp-file "imoogi-project-move" t))
           (outside (make-temp-file "imoogi-project-outside" t))
           (old-document (expand-file-name "concept.org" project))
           (new-document (expand-file-name "moved.org" outside))
           (old-asset (expand-file-name "assets/한 글.png" project))
           (new-asset (expand-file-name "moved.assets/한 글.png" outside))
           (old-owner `((kind . "project_notes") (document . ,old-document)
                        (root . ,project) (registry_key . "project")))
           (buffer-file-name new-document)
           imported-paths)
      (make-directory (file-name-directory old-asset) t)
      (write-region "png" nil old-asset nil 'silent)
      (insert "[[file:assets/%ED%95%9C%20%EA%B8%80.png][image]]")
      (unwind-protect
          (cl-letf (((symbol-function 'imoogi-project-notes--read-registry)
                     (lambda () nil))
                    ((symbol-function 'imoogi-clipboard--call-sync)
                     (lambda (request _timeout)
                       (setq imported-paths (append (alist-get 'paths request) nil))
                       `((status . "ok")
                         (assets . (((id . "new") (path . ,new-asset))))))))
            (imoogi-clipboard--relocate-visited-assets
             old-document new-document old-owner)
            (should (equal imported-paths (list old-asset)))
            (should (string-match-p "moved\.assets/%ED%95%9C%20%EA%B8%80\.png"
                                    (buffer-string))))
        (delete-directory project t)
        (delete-directory outside t)))))

(ert-deftest imoogi-clipboard-dired-rename-preserves-visited-file-metadata ()
  (let* ((directory (make-temp-file "imoogi-dired-rename" t))
	 (old-document (expand-file-name "old.org" directory))
	 (new-document (expand-file-name "new.org" directory))
	 (old-asset (expand-file-name "old.assets/screen.png" directory))
	 (new-asset (expand-file-name "new.assets/screen.png" directory))
	 buffer)
    (make-directory (file-name-directory old-asset) t)
    (write-region "[[file:old.assets/screen.png][screen]]\n" nil
		  old-document nil 'silent)
    (write-region "png" nil old-asset nil 'silent)
    (setq buffer (find-file-noselect old-document))
    (unwind-protect
	(cl-letf (((symbol-function 'imoogi-clipboard--call-sync)
		   (lambda (_request _timeout)
		     `((status . "ok")
		       (assets . (((id . "asset") (path . ,new-asset))))))))
	  (with-current-buffer buffer
	    (org-mode)
	    (imoogi-clipboard-mode 1)
	    (set-buffer-modified-p nil))
	  (dired-rename-file old-document new-document nil)
	  (with-current-buffer buffer
	    (should (file-equal-p buffer-file-name new-document))
	    (should (equal buffer-file-truename (file-truename new-document)))
	    (should (equal (buffer-name) "new.org"))
	    (should (buffer-modified-p))
	    (should (string-match-p "new\\.assets/screen\\.png"
				    (buffer-string)))))
      (when (buffer-live-p buffer) (kill-buffer buffer))
      (delete-directory directory t))))

(ert-deftest imoogi-clipboard-dired-rename-relocates-unvisited-project-note ()
  (let* ((project (make-temp-file "imoogi-dired-project" t))
	 (outside (make-temp-file "imoogi-dired-outside" t))
	 (old-document (expand-file-name "concept.org" project))
	 (new-document (expand-file-name "concept.org" outside))
	 (old-asset (expand-file-name "assets/screen.png" project))
	 (new-asset (expand-file-name "concept.assets/screen.png" outside)))
    (make-directory (file-name-directory old-asset) t)
    (write-region "[[file:assets/screen.png][screen]]\n" nil
		  old-document nil 'silent)
    (write-region "png" nil old-asset nil 'silent)
    (unwind-protect
	(cl-letf (((symbol-function 'imoogi-project-notes--read-registry)
		   (lambda (&optional _)
		     `(((key . "project") (notes-dir . ,project)))))
		  ((symbol-function 'imoogi-project-notes--current-entry)
		   (lambda () nil))
		  ((symbol-function 'imoogi-clipboard--call-sync)
		   (lambda (request _timeout)
		     (if (equal (alist-get 'operation request) "import")
			 (progn
			   (make-directory (file-name-directory new-asset) t)
			   (copy-file old-asset new-asset t)
			   `((status . "ok")
			     (assets . (((id . "asset") (path . ,new-asset))))))
		       '((status . "ok"))))))
	  (dired-rename-file old-document new-document nil)
	  (should (file-exists-p new-document))
	  (should (file-exists-p new-asset))
	  (with-temp-buffer
	    (insert-file-contents new-document)
	    (should (string-match-p "concept\\.assets/screen\\.png"
				    (buffer-string)))))
      (delete-directory project t)
      (delete-directory outside t))))

(provide 'clipboard-test)
;;; clipboard-test.el ends here
