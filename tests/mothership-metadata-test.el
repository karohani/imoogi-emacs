;;; mothership-metadata-test.el --- Metadata editing -*- lexical-binding: t; -*-

(require 'ert)
(require 'cl-lib)
(defvar imoogi-mothership-property-values)

(ert-deftest imoogi-mothership-metadata-is-present-in-new-notes ()
  (imoogi-test--with-mothership
    (imoogi-mothership-capture "메타데이터")
    (goto-char (point-min))
    (should (equal (org-entry-get nil "TYPE") "note"))
    (let ((properties (org-entry-properties)))
      (dolist (field '("ROAM_ALIASES" "CREATED" "MODIFIED" "SOURCE" "RELATED"
                       "CONFIDENCE" "LAYER" "STATUS"))
        (should (assoc field properties))))
    (should (equal (org-entry-get nil "CREATED") (org-entry-get nil "MODIFIED")))
    (let ((keywords (org-collect-keywords '("AUTHOR" "DESCRIPTION" "FILETAGS"))))
      (should (assoc "AUTHOR" keywords))
      (should (assoc "DESCRIPTION" keywords))
      (should (assoc "FILETAGS" keywords)))))

(ert-deftest imoogi-mothership-metadata-picker-updates-file-property ()
  (imoogi-test--with-mothership
    (imoogi-mothership-capture "선택")
    (insert "* 본문 제목\n")
    (forward-line -1)
    (let ((answers '("CONFIDENCE" "high")))
      (cl-letf (((symbol-function 'completing-read)
                 (lambda (_prompt collection _predicate require-match &rest _)
                   (should require-match)
                   (let ((answer (pop answers)))
                     (should (member answer collection))
                     answer))))
        (call-interactively #'imoogi-mothership-set-property)))
    (should (org-at-heading-p))
    (should-not (org-entry-get nil "CONFIDENCE"))
    (save-excursion
      (goto-char (point-min))
      (should (equal (org-entry-get nil "CONFIDENCE") "high")))))

(ert-deftest imoogi-mothership-metadata-supports-configured-enums ()
  (imoogi-test--with-mothership
    (imoogi-mothership-capture "사용자 목록")
    (let ((imoogi-mothership-property-values '(("LAYER" "entities" "sources"))))
      (imoogi-mothership-set-property "LAYER" "entities"))
    (goto-char (point-min))
    (should (equal (org-entry-get nil "LAYER") "entities"))))

(ert-deftest imoogi-mothership-metadata-rejects-invalid-values-and-identity-edits ()
  (imoogi-test--with-mothership
    (imoogi-mothership-capture "검증")
    (let ((original (buffer-string)))
      (should-error (imoogi-mothership-set-property "CONFIDENCE" "certain") :type 'user-error)
      (should-error (imoogi-mothership-set-property "UNDEFINED" "x") :type 'user-error)
      (let ((imoogi-mothership-property-values '(("ID" "replacement"))))
        (should-error (imoogi-mothership-set-property "ID" "replacement") :type 'user-error))
      (should (equal (buffer-string) original)))))

(ert-deftest imoogi-mothership-metadata-rejects-outside-and-read-only-buffers ()
  (with-temp-buffer
    (org-mode)
    (should-error (imoogi-mothership-set-property "TYPE" "note") :type 'user-error))
  (imoogi-test--with-mothership
    (imoogi-mothership-capture "읽기 전용")
    (let ((buffer-read-only t))
      (should-error (imoogi-mothership-set-property "TYPE" "note") :type 'buffer-read-only))))

(ert-deftest imoogi-mothership-metadata-save-updates-modified-only ()
  (imoogi-test--with-mothership
    (imoogi-mothership-capture "저장 시각")
    (goto-char (point-min))
    (let ((created (org-entry-get nil "CREATED"))
          (file buffer-file-name))
      (org-entry-put nil "MODIFIED" "[2000-01-01 Sat 00:00]")
      (save-buffer)
      (should (equal (org-entry-get nil "CREATED") created))
      (should-not (equal (org-entry-get nil "MODIFIED") "[2000-01-01 Sat 00:00]"))
      (with-temp-buffer
        (insert-file-contents file)
        (org-mode)
        (should (equal (org-entry-get nil "CREATED") created))
        (should-not (equal (org-entry-get nil "MODIFIED") "[2000-01-01 Sat 00:00]"))))))

(ert-deftest imoogi-mothership-metadata-menu-exposes-picker ()
  (should (eq (plist-get (cdr (transient-get-suffix 'imoogi-mothership-transient "e")) :command)
              #'imoogi-mothership-edit-metadata))
  (should (eq (plist-get (cdr (transient-get-suffix 'imoogi-mothership-transient "E")) :command)
              #'imoogi-mothership-set-property)))

(ert-deftest imoogi-mothership-metadata-save-preserves-legacy-note ()
  (imoogi-test--with-mothership
    (let ((file (expand-file-name "00. Inbox/legacy.org" imoogi-mothership-directory)))
      (with-temp-file file (insert "#+TITLE: 기존 노트\n"))
      (with-current-buffer (find-file-noselect file)
        (goto-char (point-max))
        (insert "본문\n")
        (save-buffer)
        (goto-char (point-min))
        (should-not (org-entry-get nil "MODIFIED"))
        (should-not (org-entry-get nil "CREATED"))))))

(ert-deftest imoogi-mothership-metadata-rejects-lowercase-reserved-names ()
  (imoogi-test--with-mothership
    (imoogi-mothership-capture "보호할 속성")
    (let ((original (buffer-string)))
      (dolist (field '("id" "Id" "created" "modified" "Modified"))
        (let ((imoogi-mothership-property-values (list (list field "replacement"))))
          (should-error (imoogi-mothership-set-property field "replacement") :type 'user-error)))
      (should (equal original (buffer-string))))))

(ert-deftest imoogi-mothership-metadata-picker-creates-preamble-before-first-heading ()
  (imoogi-test--with-mothership
    (let ((file (expand-file-name "00. Inbox/heading.org" imoogi-mothership-directory)))
      (with-temp-file file (insert "* 기존 제목\n본문\n"))
      (with-current-buffer (find-file-noselect file)
        (goto-char (point-min))
        (imoogi-mothership-set-property "TYPE" "note")
        (goto-char (point-min))
        (should (org-before-first-heading-p))
        (should (equal (org-entry-get nil "TYPE") "note"))
        (re-search-forward "^\\* 기존 제목")
        (should-not (org-entry-get nil "TYPE"))))))

(ert-deftest imoogi-mothership-metadata-save-preserves-heading-modified ()
  (imoogi-test--with-mothership
    (let* ((file (expand-file-name "00. Inbox/heading-time.org" imoogi-mothership-directory))
           (text "* 기존 제목\n:PROPERTIES:\n:MODIFIED: original-heading-date\n:END:\n"))
      (with-temp-file file (insert text))
      (with-current-buffer (find-file-noselect file)
        (set-buffer-modified-p t)
        (save-buffer)
        (should (equal (buffer-string) text))))))

;;; mothership-metadata-test.el ends here
