;;; clipboard-trace-test.el --- Durable diagnostic tracing -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(require 'imoogi-clipboard)

(defmacro imoogi-trace-test--isolated (&rest body)
  (declare (indent 0))
  `(let* ((directory (make-temp-file "imoogi-trace-test" t))
          (user-emacs-directory (file-name-as-directory directory))
          (imoogi-clipboard-trace-file (expand-file-name "trace.log" directory))
          (imoogi-clipboard-trace-max-bytes 8192))
     (unwind-protect (progn ,@body)
       (when (fboundp 'imoogi-clipboard-trace-mode)
         (imoogi-clipboard-trace-mode -1))
       (delete-directory directory t))))

(defun imoogi-trace-test--text ()
  (with-temp-buffer
    (insert-file-contents imoogi-clipboard-trace-file)
    (buffer-string)))

(ert-deftest imoogi-clipboard-trace-opt-in-and-disable ()
  (imoogi-trace-test--isolated
    (should (boundp 'imoogi-clipboard-trace-mode))
    (should-not imoogi-clipboard-trace-mode)
    (imoogi-clipboard--owner)
    (should-not (file-exists-p imoogi-clipboard-trace-file))
    (imoogi-clipboard-trace-mode 1)
    (imoogi-clipboard-trace-mode 1)
    (imoogi-clipboard--owner)
    (let ((text (imoogi-trace-test--text)))
      (should (= 1 (imoogi-trace-test--begin-count text)))
      (imoogi-clipboard-trace-mode -1)
      (let ((after (imoogi-trace-test--text)))
        (imoogi-clipboard--owner)
        (should (equal after (imoogi-trace-test--text)))))))

(defun imoogi-trace-test--begin-count (text)
  (with-temp-buffer
    (insert text)
    (how-many "phase=begin" (point-min) (point-max))))

(ert-deftest imoogi-clipboard-trace-real-nesting-and-privacy ()
  (imoogi-trace-test--isolated
    (with-temp-buffer
      (setq buffer-file-name (expand-file-name "PRIVATE-note.org" directory))
      (insert "PRIVATE-BODY")
      (imoogi-clipboard-trace-mode 1)
      (let ((value (imoogi-clipboard--owner)))
        (should value))
      (let ((text (imoogi-trace-test--text)))
        (should (string-match-p "phase=begin.*depth=1.*fn=imoogi-clipboard--owner" text))
        (should (string-match-p "phase=begin.*depth=2.*fn=imoogi-project-notes--read-registry" text))
        (should (string-match-p "phase=end.*duration=.*outcome=ok" text))
        (should (string-match-p "callers=.*imoogi-clipboard--owner" text))
        (should-not (string-match-p "PRIVATE\\|imoogi-trace-test[[:alnum:]]*/" text))))))

(ert-deftest imoogi-clipboard-trace-preserves-control-and-write-failures ()
  (imoogi-trace-test--isolated
    (imoogi-clipboard-trace-mode 1)
    (let ((value (list 'private-value)))
      (should (eq value (imoogi-clipboard--trace-call 'test (lambda () value))))
      (should (equal '(error "PRIVATE-error")
                     (condition-case err
                         (imoogi-clipboard--trace-call 'test (lambda () (error "PRIVATE-error")))
                       (error err))))
      (should (eq 'quit
                  (condition-case nil
                      (imoogi-clipboard--trace-call 'test (lambda () (signal 'quit nil)))
                    (quit 'quit))))
      (should (string-match-p "outcome=nonlocal-exit" (imoogi-trace-test--text)))
      (should-not (string-match-p "PRIVATE" (imoogi-trace-test--text)))
      (cl-letf (((symbol-function 'write-region) (lambda (&rest _) (error "disk full"))))
        (should (eq value (imoogi-clipboard--trace-call 'test (lambda () value))))
        (should-error (imoogi-clipboard--trace-call 'test (lambda () (error "original")))
                      :type 'error)))))

(ert-deftest imoogi-clipboard-trace-hooks-and-bounds ()
  (imoogi-trace-test--isolated
    (let ((before-change-functions (list (lambda (&rest _) (ert-fail "before hook"))))
          (after-change-functions (list (lambda (&rest _) (ert-fail "after hook")))))
      (imoogi-clipboard-trace-mode 1)
      (let ((imoogi-clipboard-trace-max-bytes 2048))
        (dotimes (_ 30) (imoogi-clipboard--owner))
        (should (<= (file-attribute-size (file-attributes imoogi-clipboard-trace-file)) 2048))
        (should (<= (file-attribute-size (file-attributes (concat imoogi-clipboard-trace-file ".1"))) 2048))
        (should (= 2 (length (directory-files directory nil "trace.log"))))))))

(ert-deftest imoogi-clipboard-trace-enabling-failure-is-visible ()
  (imoogi-trace-test--isolated
    (setq imoogi-clipboard-trace-file directory)
    (should-error (imoogi-clipboard-trace-mode 1))
    (should-not imoogi-clipboard-trace-mode)))

(ert-deftest imoogi-clipboard-trace-reenable-bounds-existing-backup ()
  (imoogi-trace-test--isolated
    (let ((backup (concat imoogi-clipboard-trace-file ".1")))
      (write-region (make-string 4096 ?x) nil backup nil 'silent)
      (imoogi-clipboard-trace-mode 1)
      (should (= 4096 (file-attribute-size (file-attributes backup))))
      (imoogi-clipboard-trace-mode -1)
      (setq imoogi-clipboard-trace-max-bytes 2048)
      (imoogi-clipboard-trace-mode 1)
      (let ((current-size (file-attribute-size (file-attributes imoogi-clipboard-trace-file)))
            (backup-size (file-attribute-size (file-attributes backup))))
        (message "trace cap reduction: max=2048 current=%s previous=%s"
                 current-size backup-size)
        (should (<= current-size 2048))
        (should (<= backup-size 2048))))))

(ert-deftest imoogi-clipboard-trace-preserves-editor-state-and-skips-annotations ()
  (imoogi-trace-test--isolated
    (with-temp-buffer
      (insert "original buffer")
      (string-match "ori" "original")
      (let ((matches (match-data)) (position (point)) (buffer (current-buffer))
            (write-region-annotate-functions
             (list (lambda (&rest _) (ert-fail "annotation hook"))))
            (write-region-post-annotation-function
             (lambda () (ert-fail "post annotation hook"))))
        (imoogi-clipboard-trace-mode 1)
        (should (eq 'result (imoogi-clipboard--trace-call 'test (lambda ()
                               (should (equal matches (match-data)))
                               'result))))
        (should (equal matches (match-data)))
        (should (= position (point)))
        (should (eq buffer (current-buffer)))
        (should (equal "original buffer" (buffer-string)))))))

(ert-deftest imoogi-clipboard-trace-targets-timer-and-paired-ids ()
  (imoogi-trace-test--isolated
    (imoogi-clipboard-trace-mode 1)
    (imoogi-org-preview--send-update nil)
    (imoogi-clipboard--renew-active-leases)
    (let ((timer (run-at-time 0 nil #'imoogi-clipboard--owner)))
      (unwind-protect (sleep-for 0.02) (cancel-timer timer)))
    (let ((text (imoogi-trace-test--text)))
      (should (string-match-p "fn=imoogi-org-preview--send-update" text))
      (should (string-match-p "fn=imoogi-clipboard--renew-active-leases" text))
      (should (string-match-p "callers=.*timer-event-handler" text))
      (let ((start 0) ids)
        (while (string-match "phase=begin id=\\([0-9]+\\)" text start)
          (push (match-string 1 text) ids)
          (setq start (match-end 0)))
        (should (= 3 (length ids)))
        (dolist (id ids)
          (should (string-match-p (concat "phase=end id=" id " ") text)))))))

;;; clipboard-trace-test.el ends here
