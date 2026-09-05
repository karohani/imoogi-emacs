;;; flashcards-org-test.el --- local flashcards Org scan tests -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'org)
(require 'imoogi-flashcards-org)

(defmacro imoogi-flashcards-test--with-root (&rest body)
  "Run BODY with ROOT bound to a temporary directory."
  `(let ((root (make-temp-file "imoogi-flashcards" t)))
     (unwind-protect
         (progn ,@body)
       (delete-directory root t))))

(defun imoogi-flashcards-test--write (path content)
  "Write CONTENT to PATH."
  (make-directory (file-name-directory path) t)
  (with-temp-file path (insert content)))

(ert-deftest imoogi-flashcards-scan-basic-front-back ()
  (imoogi-flashcards-test--with-root
   (imoogi-flashcards-test--write
    (expand-file-name "cards.org" root)
    (concat "* Capital\n"
            ":PROPERTIES:\n"
            ":IMOOGI_FLASHCARD_ID: n1\n"
            ":IMOOGI_FLASHCARD_KIND: Basic\n"
            ":END:\n"
            "Seoul\n"))
   (let* ((scan (imoogi-flashcards-scan-root root))
          (card (car (plist-get scan :cards))))
     (should (null (plist-get scan :duplicate-note-ids)))
     (should (equal (imoogi-flashcards-card-key card) "n1"))
     (should (equal (imoogi-flashcards-card-front card) "Capital"))
     (should (equal (imoogi-flashcards-card-back card) "Seoul")))))

(ert-deftest imoogi-flashcards-scan-cloze-generates-one-card-per-number ()
  (imoogi-flashcards-test--with-root
   (imoogi-flashcards-test--write
    (expand-file-name "cards.org" root)
    (concat "* Cloze\n"
            ":PROPERTIES:\n"
            ":IMOOGI_FLASHCARD_ID: n2\n"
            ":IMOOGI_FLASHCARD_KIND: Cloze\n"
            ":END:\n"
            "{{c1::Seoul}} is in {{c2::Korea}}.\n"))
   (let* ((scan (imoogi-flashcards-scan-root root))
          (cards (plist-get scan :cards))
          (keys (mapcar #'imoogi-flashcards-card-key cards)))
     (should (equal keys '("n2::c1" "n2::c2")))
     (should (string-match-p "\\[\\.\\.\\.\\]"
                             (imoogi-flashcards-card-front (car cards))))
     (should (string-match-p "Seoul is in Korea"
                             (imoogi-flashcards-card-back (car cards)))))))

(ert-deftest imoogi-flashcards-scan-duplicate-note-id-is-conflict ()
  (imoogi-flashcards-test--with-root
   (imoogi-flashcards-test--write
    (expand-file-name "cards.org" root)
    (concat "* A\n:PROPERTIES:\n:IMOOGI_FLASHCARD_ID: dup\n"
            ":IMOOGI_FLASHCARD_KIND: Basic\n:END:\nA\n"
            "* B\n:PROPERTIES:\n:IMOOGI_FLASHCARD_ID: dup\n"
            ":IMOOGI_FLASHCARD_KIND: Basic\n:END:\nB\n"))
   (let ((scan (imoogi-flashcards-scan-root root)))
     (should (equal (plist-get scan :duplicate-note-ids) '("dup")))
     (should (null (plist-get scan :cards))))))

(ert-deftest imoogi-flashcards-scan-file-duplicate-note-id-is-conflict ()
  (imoogi-flashcards-test--with-root
   (let ((file (expand-file-name "cards.org" root)))
     (imoogi-flashcards-test--write
      file
      (concat "* A\n:PROPERTIES:\n:IMOOGI_FLASHCARD_ID: dup\n"
              ":IMOOGI_FLASHCARD_KIND: Basic\n:END:\nA\n"
              "* B\n:PROPERTIES:\n:IMOOGI_FLASHCARD_ID: dup\n"
              ":IMOOGI_FLASHCARD_KIND: Basic\n:END:\nB\n"))
     (let ((scan (imoogi-flashcards-scan-file file)))
       (should (equal (plist-get scan :duplicate-note-ids) '("dup")))
       (should (null (plist-get scan :cards)))))))

(ert-deftest imoogi-flashcards-resolve-card-reads-current-org-content ()
  (imoogi-flashcards-test--with-root
   (let ((file (expand-file-name "cards.org" root)))
     (imoogi-flashcards-test--write
      file
      (concat "* Capital\n"
              ":PROPERTIES:\n"
              ":IMOOGI_FLASHCARD_ID: n1\n"
              ":IMOOGI_FLASHCARD_KIND: Basic\n"
              ":END:\n"
              "Seoul\n"))
     (should (equal (imoogi-flashcards-card-back
                     (imoogi-flashcards-resolve-card file "n1"))
                    "Seoul"))
     (imoogi-flashcards-test--write
      file
      (concat "* Capital\n"
              ":PROPERTIES:\n"
              ":IMOOGI_FLASHCARD_ID: n1\n"
              ":IMOOGI_FLASHCARD_KIND: Basic\n"
              ":END:\n"
              "Busan\n"))
     (should (equal (imoogi-flashcards-card-back
                     (imoogi-flashcards-resolve-card file "n1"))
                    "Busan")))))

(ert-deftest imoogi-flashcards-mark-basic-adds-id-and-kind ()
  (with-temp-buffer
    (insert "* Card\nBack\n")
    (org-mode)
    (goto-char (point-min))
    (imoogi-flashcards-mark-basic)
    (should (org-entry-get (point) imoogi-flashcards-id-property nil))
    (should (equal (org-entry-get (point) imoogi-flashcards-kind-property nil)
                   "Basic"))))

(provide 'flashcards-org-test)
;;; flashcards-org-test.el ends here
