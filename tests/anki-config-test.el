;;; imoogi-config-test.el --- Tests for imoogi-config.el -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'imoogi-config)

(defmacro imoogi-config-test--with-temp-file (var &rest body)
  "Bind VAR to a fresh nonexistent temp file path for the duration of
BODY, deleting it afterward whether or not BODY created it."
  (declare (indent 1))
  `(let ((,var (make-temp-file "imoogi-config-test" nil ".json")))
     (delete-file ,var)
     (unwind-protect
         (progn ,@body)
       (when (file-exists-p ,var)
         (delete-file ,var)))))

(ert-deftest imoogi-config-test-write-creates-file-with-both-values ()
  (imoogi-config-test--with-temp-file file
    (imoogi-config-write file "/tmp/some-root" "TestDeck")
    (should (file-exists-p file))
    (let* ((parsed (with-temp-buffer
                      (insert-file-contents file)
                      (json-parse-string (buffer-string) :object-type 'alist))))
      (should (equal (cdr (assq 'sync_root parsed)) "/tmp/some-root"))
      (should (equal (cdr (assq 'default_deck parsed)) "TestDeck")))))

(ert-deftest imoogi-config-test-write-only-touches-the-two-values ()
  "design.md SS2.5 / plan.md D-11: `imoogi-anki-setup' writes exactly
the sync root and the default deck -- nothing else belongs in this
file."
  (imoogi-config-test--with-temp-file file
    (imoogi-config-write file "/tmp/root" "Deck")
    (let* ((parsed (with-temp-buffer
                      (insert-file-contents file)
                      (json-parse-string (buffer-string) :object-type 'alist))))
      (should (= (length parsed) 2)))))

(ert-deftest imoogi-config-test-load-populates-both-defcustoms ()
  (imoogi-config-test--with-temp-file file
    (imoogi-config-write file "/tmp/loaded-root" "LoadedDeck")
    (let ((imoogi-sync-root nil)
          (imoogi-default-deck "Default"))
      (imoogi-config-load file)
      (should (equal imoogi-sync-root "/tmp/loaded-root"))
      (should (equal imoogi-default-deck "LoadedDeck")))))

(ert-deftest imoogi-config-test-load-absent-file-is-a-no-op ()
  "First run, before `imoogi-anki-setup' has ever written anything --
must not error, must not touch the defcustoms."
  (imoogi-config-test--with-temp-file file
    ;; file was deleted by the fixture macro and never recreated
    (let ((imoogi-sync-root "unchanged")
          (imoogi-default-deck "unchanged"))
      (should (null (imoogi-config-load file)))
      (should (equal imoogi-sync-root "unchanged"))
      (should (equal imoogi-default-deck "unchanged")))))

(ert-deftest imoogi-config-test-round-trip ()
  (imoogi-config-test--with-temp-file file
    (imoogi-config-write file "/root/one" "Deck1")
    (let ((imoogi-sync-root nil)
          (imoogi-default-deck nil))
      (imoogi-config-load file)
      (should (equal imoogi-sync-root "/root/one"))
      (should (equal imoogi-default-deck "Deck1")))))

(provide 'imoogi-config-test)
;;; imoogi-config-test.el ends here
