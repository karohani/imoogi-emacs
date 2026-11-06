;;; imoogi-writeback-test.el --- Tests for imoogi-writeback.el -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'org)
(require 'org-id)
(require 'imoogi-writeback)

(defmacro imoogi-test--with-file (relative-path content &rest body)
  "Write CONTENT to RELATIVE-PATH under a fresh temp ROOT, run BODY with
ROOT and FILE bound, clean up after."
  (declare (indent 2))
  `(let* ((root (make-temp-file "imoogi-wb-test" t))
          (file (expand-file-name ,relative-path root))
          (org-id-locations-file (expand-file-name "org-id-locations" root))
          (org-id-locations (make-hash-table :test 'equal))
          (org-id-files nil))
     (unwind-protect
         (progn
           (make-directory (file-name-directory file) t)
           (with-temp-file file (insert ,content))
           ,@body)
       (let ((buf (find-buffer-visiting file)))
         (when buf (kill-buffer buf)))
       (delete-directory root t))))

;; --- The modified-buffer split. One of the four highest-risk behaviors
;; named in the delegation prompt: a write-back onto a file with
;; pre-existing unsaved edits must write into the buffer and leave it
;; modified+unsaved, without touching the file on disk, and must leave
;; the rest of the buffer byte-identical outside the identifier properties.

(ert-deftest imoogi-writeback-test-modified-buffer-stays-unsaved ()
  (imoogi-test--with-file
   "note.org"
   "* Capital of France\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nParis.\n"
   (let* ((buffer (find-file-noselect file))
          (on-disk-before (with-temp-buffer
                             (insert-file-contents file)
                             (buffer-string))))
     (with-current-buffer buffer
       ;; the user has unsaved edits in progress before the sync run.
       (goto-char (point-max))
       (insert "A user note, unrelated to imoogi.\n"))
     (should (buffer-modified-p buffer))
     (let ((rest-before-writeback
            (with-current-buffer buffer (buffer-string))))
       (should (equal (imoogi-writeback-apply
                       root (list (list :key "note.org::0"
                                        :action "added"
                                        :note-id 1001)))
                      '("note.org")))
       (with-current-buffer buffer
         ;; the identifier is now present in the buffer text.
         (should (string-match-p ":ANKI_NOTE_ID: +1001" (buffer-string)))
         (goto-char (point-min))
         (should (org-entry-get (point) "ID" nil))
         ;; the buffer is still modified and unsaved by imoogi.
         (should (buffer-modified-p buffer))
         ;; the user's own prior edit is still present, unsaved.
         (should (string-match-p "A user note, unrelated to imoogi\\." (buffer-string)))
         ;; the rest of the buffer, outside the identifier properties, is
         ;; byte-identical to what it was before write-back.
         (let ((before-lines (split-string rest-before-writeback "\n"))
               (after-lines (split-string (buffer-string) "\n")))
           (dolist (line before-lines)
             (unless (string-match-p ":PROPERTIES:\\|:END:\\|ANKI_NOTE_TYPE" line)
               (should (member line after-lines))))))
       ;; the file on disk is unchanged -- imoogi did not save it.
       (should (equal (with-temp-buffer
                         (insert-file-contents file)
                         (buffer-string))
                       on-disk-before))))))

(ert-deftest imoogi-writeback-test-clean-buffer-saves ()
  (imoogi-test--with-file
   "note.org"
   "* Capital of France\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nParis.\n"
   (let ((needs-save
          (imoogi-writeback-apply root (list (list :key "note.org::0"
                                                     :action "added"
                                                     :note-id 1001)))))
     ;; nothing needed a save -- the buffer had no prior unsaved mods.
     (should (null needs-save))
     ;; the identifier landed on disk.
     (let ((on-disk (with-temp-buffer
                       (insert-file-contents file)
                       (buffer-string))))
       (should (string-match-p ":ANKI_NOTE_ID: +1001" on-disk))))))

(ert-deftest imoogi-writeback-test-no-existing-buffer-opens-and-saves ()
  (imoogi-test--with-file
   "note.org"
   "* Capital of France\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nParis.\n"
   ;; no buffer visits FILE yet at all.
   (should (null (find-buffer-visiting file)))
   (let ((needs-save
          (imoogi-writeback-apply root (list (list :key "note.org::0"
                                                     :action "added"
                                                     :note-id 2002)))))
     (should (null needs-save))
     (let ((on-disk (with-temp-buffer
                       (insert-file-contents file)
                       (buffer-string))))
       (should (string-match-p ":ANKI_NOTE_ID: +2002" on-disk))))))

(ert-deftest imoogi-writeback-test-skips-non-added-results ()
  (imoogi-test--with-file
   "note.org"
   "* Capital of France\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ANKI_NOTE_ID: 1001\n:END:\nParis.\n"
   (let ((before (with-temp-buffer (insert-file-contents file) (buffer-string))))
     (imoogi-writeback-apply root (list (list :key "note.org::0"
                                                :action "updated"
                                                :note-id 1001)))
     (should (equal (with-temp-buffer (insert-file-contents file) (buffer-string))
                     before)))))

(ert-deftest imoogi-writeback-test-org-id-created-and-preserved ()
  "Normal and migration writeback preserve heading identity on retries."
  (dolist (apply-fn '(imoogi-writeback-apply imoogi-writeback-apply-migration))
    (dolist (existing-id '(nil "existing-org-id"))
      (imoogi-test--with-file
       "note.org"
       (concat "* Card\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n"
               (when existing-id (concat ":ID: " existing-id "\n"))
               ":END:\nBody.\n")
       (funcall apply-fn root '((:key "note.org::0" :action "added" :note-id 1001)))
       (with-current-buffer (find-buffer-visiting file)
         (goto-char (point-min))
         (let ((id (org-entry-get (point) "ID" nil)))
           (should (and id (not (string-empty-p id))))
           (should (equal (org-entry-get (point) "ANKI_NOTE_ID" nil) "1001"))
           (when existing-id (should (equal id existing-id)))
           (funcall apply-fn root '((:key "note.org::0" :action "added" :note-id 1002)))
           (should (equal (org-entry-get (point) "ID" nil) id))
           (should (equal (org-entry-get (point) "ANKI_NOTE_ID" nil) "1002"))
           (should (equal (with-temp-buffer
                            (insert-file-contents file)
                            (org-mode)
                            (goto-char (point-min))
                            (org-entry-get (point) "ID" nil))
                          id))))))))

(ert-deftest imoogi-writeback-test-child-cards-have-distinct-local-ids ()
  "Parent identity is not reused, and multiple cards receive their own IDs."
  (imoogi-test--with-file
   "note.org"
   "* Parent\n:PROPERTIES:\n:ID: parent-id\n:END:\n** First\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nOne.\n** Second\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nTwo.\n"
   (imoogi-writeback-apply root '((:key "note.org::0" :action "added" :note-id 1001)
                                (:key "note.org::1" :action "added" :note-id 1002)))
   (with-current-buffer (find-buffer-visiting file)
     (goto-char (point-min))
     (outline-next-heading)
     (let ((first-id (org-entry-get (point) "ID" nil)))
       (should first-id)
       (should-not (equal first-id "parent-id"))
       (should (equal (org-entry-get (point) "ANKI_NOTE_ID" nil) "1001"))
       (outline-next-heading)
       (let ((second-id (org-entry-get (point) "ID" nil)))
         (should second-id)
         (should-not (member second-id (list "parent-id" first-id)))
         (should (equal (org-entry-get (point) "ANKI_NOTE_ID" nil) "1002")))))))

(ert-deftest imoogi-writeback-test-missing-target-does-not-create-id ()
  (imoogi-test--with-file
   "note.org" "* Ordinary heading\nBody.\n"
   (imoogi-writeback-apply root '((:key "note.org::0" :action "added" :note-id 1001)))
   (with-current-buffer (find-buffer-visiting file)
     (goto-char (point-min))
     (should-not (org-entry-get (point) "ID" nil))
     (should-not (org-entry-get (point) "ANKI_NOTE_ID" nil)))))

(provide 'imoogi-writeback-test)
;;; imoogi-writeback-test.el ends here
