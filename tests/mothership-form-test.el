;;; mothership-form-test.el --- Metadata form workflows -*- lexical-binding: t; -*-

(require 'ert)
(require 'wid-edit)

(defvar imoogi-mothership--form-widgets)

(defmacro imoogi-test--with-metadata-form (&rest body)
  "Run BODY in a real form with its original note bound as SOURCE."
  (declare (indent 0))
  `(imoogi-test--with-mothership
     (imoogi-mothership-capture "폼 테스트")
     (insert "* 본문\n#+TITLE: 본문 예시\n본문은 유지한다.\n")
     (let ((source (current-buffer)) form)
       (unwind-protect
           (progn
             (imoogi-mothership-edit-metadata)
             (setq form (current-buffer))
             ,@body)
         (when (buffer-live-p form) (kill-buffer form))))))

(ert-deftest imoogi-mothership-form-opens-without-changing-note ()
  (imoogi-test--with-metadata-form
    (should (derived-mode-p 'imoogi-mothership-metadata-mode))
    (should (equal (widget-value (cdr (assoc "TITLE" imoogi-mothership--form-widgets))) "폼 테스트"))
    (dolist (name '("TITLE" "AUTHOR" "DESCRIPTION" "FILETAGS" "ROAM_ALIASES"
                    "TYPE" "STATUS" "CONFIDENCE" "SOURCE" "RELATED" "LAYER"))
      (should (cdr (assoc name imoogi-mothership--form-widgets))))
    (dolist (name '("ID" "CREATED" "MODIFIED"))
      (should-not (assoc name imoogi-mothership--form-widgets)))
    (should (eq (lookup-key (current-local-map) (kbd "C-c C-c")) #'imoogi-mothership-metadata-apply))
    (should (eq (lookup-key (current-local-map) (kbd "C-c C-k")) #'imoogi-mothership-metadata-cancel))
    (with-current-buffer source
      (should (string-match-p "본문은 유지한다" (buffer-string))))))

(ert-deftest imoogi-mothership-form-roundtrip-applies-fields-without-saving ()
  (imoogi-test--with-metadata-form
    (let ((before (with-current-buffer source (buffer-string)))
          (file (buffer-file-name source))
          (disk (with-current-buffer source
                  (with-temp-buffer (insert-file-contents (buffer-file-name source)) (buffer-string))))
          (id (with-current-buffer source (save-excursion (goto-char (point-min)) (org-entry-get nil "ID"))))
          (created (with-current-buffer source (save-excursion (goto-char (point-min)) (org-entry-get nil "CREATED")))))
      (dolist (field '(("TITLE" . "바뀐 제목") ("AUTHOR" . "Jay")
                       ("DESCRIPTION" . "설명을 입력합니다.") ("FILETAGS" . ":concept:history:")
                       ("ROAM_ALIASES" . "\"Memory Extender\" \"미멕스\"")
                       ("TYPE" . "wiki-page") ("STATUS" . "active") ("CONFIDENCE" . "high")
                       ("SOURCE" . "https://example.org/paper")
                       ("RELATED" . "[[id:related][관련 노트]]") ("LAYER" . "entities")))
        (widget-value-set (cdr (assoc (car field) imoogi-mothership--form-widgets)) (cdr field)))
      (should (equal before (with-current-buffer source (buffer-string))))
      (imoogi-mothership-metadata-apply)
      (should (eq (current-buffer) source))
      (goto-char (point-min))
      (should (equal (org-entry-get nil "ID") id))
      (should (equal (org-entry-get nil "CREATED") created))
      (should (equal (org-entry-get nil "CONFIDENCE") "high"))
      (should (equal (org-entry-get nil "ROAM_ALIASES") "\"Memory Extender\" \"미멕스\""))
      (should (equal (org-entry-get nil "SOURCE") "https://example.org/paper"))
      (should (equal (org-entry-get nil "RELATED") "[[id:related][관련 노트]]"))
      (let ((keywords (org-collect-keywords '("TITLE" "AUTHOR" "DESCRIPTION" "FILETAGS"))))
        (should (equal (cadr (assoc "TITLE" keywords)) "바뀐 제목"))
        (should (equal (cadr (assoc "AUTHOR" keywords)) "Jay"))
        (should (equal (cadr (assoc "DESCRIPTION" keywords)) "설명을 입력합니다."))
        (should (equal (cadr (assoc "FILETAGS" keywords)) ":concept:history:")))
      (should (string-match-p "\\* 본문\n#\\+TITLE: 본문 예시\n본문은 유지한다\\." (buffer-string)))
      (should (buffer-modified-p))
      (should (equal disk (with-temp-buffer (insert-file-contents file) (buffer-string)))))))

(ert-deftest imoogi-mothership-form-cancel-preserves-note ()
  (imoogi-test--with-metadata-form
    (let ((before (with-current-buffer source (buffer-string))))
      (widget-value-set (cdr (assoc "AUTHOR" imoogi-mothership--form-widgets)) "취소할 변경")
      (imoogi-mothership-metadata-cancel)
      (should-not (buffer-live-p form))
      (should (equal before (with-current-buffer source (buffer-string)))))))

(ert-deftest imoogi-mothership-form-rejects-multiline-without-partial-changes ()
  (imoogi-test--with-metadata-form
    (let ((before (with-current-buffer source (buffer-string))))
      (widget-value-set (cdr (assoc "TITLE" imoogi-mothership--form-widgets)) "적용하면 안 됨")
      (widget-value-set (cdr (assoc "DESCRIPTION" imoogi-mothership--form-widgets)) "설명\n#+TITLE: 주입")
      (should-error (imoogi-mothership-metadata-apply) :type 'user-error)
      (should (equal before (with-current-buffer source (buffer-string)))))))

(ert-deftest imoogi-mothership-form-detects-concurrent-note-edit ()
  (imoogi-test--with-metadata-form
    (widget-value-set (cdr (assoc "AUTHOR" imoogi-mothership--form-widgets)) "오래된 폼")
    (with-current-buffer source (goto-char (point-max)) (insert "폼을 연 뒤 추가한 내용\n"))
    (let ((updated (with-current-buffer source (buffer-string))))
      (should-error (imoogi-mothership-metadata-apply) :type 'user-error)
      (should (equal updated (with-current-buffer source (buffer-string)))))))

(ert-deftest imoogi-mothership-form-refuses-read-only-or-dead-source ()
  (imoogi-test--with-metadata-form
    (with-current-buffer source (setq buffer-read-only t))
    (should-error (imoogi-mothership-metadata-apply) :type 'buffer-read-only)
    (with-current-buffer source (setq buffer-read-only nil) (set-buffer-modified-p nil))
    (kill-buffer source)
    (should-error (imoogi-mothership-metadata-apply) :type 'user-error)))

(ert-deftest imoogi-mothership-form-rejects-newly-disallowed-enum ()
  (imoogi-test--with-metadata-form
    (let ((before (with-current-buffer source (buffer-string))))
      (widget-value-set (cdr (assoc "STATUS" imoogi-mothership--form-widgets)) "active")
      (let ((imoogi-mothership-property-values '(("STATUS" "draft"))))
        (should-error (imoogi-mothership-metadata-apply) :type 'user-error))
      (should (equal before (with-current-buffer source (buffer-string)))))))

(ert-deftest imoogi-mothership-form-keyboard-edits-and-applies ()
  (imoogi-test--with-metadata-form
    (goto-char (widget-field-start (cdr (assoc "TITLE" imoogi-mothership--form-widgets))))
    (should (eq (key-binding (kbd "C-c C-c")) #'imoogi-mothership-metadata-apply))
    (execute-kbd-macro "new ")
    (execute-kbd-macro (kbd "C-c C-c"))
    (should (eq (current-buffer) source))
    (should (equal (cadr (assoc "TITLE" (org-collect-keywords '("TITLE")))) "new 폼 테스트"))))

(ert-deftest imoogi-mothership-form-does-not-edit-source-block-keywords ()
  (imoogi-test--with-metadata-form
    (imoogi-mothership-metadata-cancel)
    (erase-buffer)
    (insert "#+begin_src org\n#+AUTHOR: sample-author\n#+end_src\n* 본문\n")
    (imoogi-mothership-edit-metadata)
    (setq form (current-buffer))
    (should (equal (widget-value (cdr (assoc "AUTHOR" imoogi-mothership--form-widgets))) ""))
    (widget-value-set (cdr (assoc "AUTHOR" imoogi-mothership--form-widgets)) "Jay")
    (imoogi-mothership-metadata-apply)
    (should (string-match-p "#\\+begin_src org\n#\\+AUTHOR: sample-author\n#\\+end_src" (buffer-string)))
    (should (equal (cadr (assoc "AUTHOR" (org-collect-keywords '("AUTHOR")))) "Jay"))))

(ert-deftest imoogi-mothership-form-preserves-unknown-legacy-enum ()
  (imoogi-test--with-metadata-form
    (imoogi-mothership-metadata-cancel)
    (goto-char (point-min))
    (org-entry-put nil "STATUS" "legacy-value")
    (imoogi-mothership-edit-metadata)
    (setq form (current-buffer))
    (widget-value-set (cdr (assoc "DESCRIPTION" imoogi-mothership--form-widgets)) "설명만 수정")
    (imoogi-mothership-metadata-apply)
    (goto-char (point-min))
    (should (equal (org-entry-get nil "STATUS") "legacy-value"))))

(ert-deftest imoogi-mothership-form-detects-external-file-change ()
  (imoogi-test--with-metadata-form
    (let ((before (with-current-buffer source (buffer-string)))
          (file (buffer-file-name source)))
      (with-temp-file file (insert "다른 프로세스에서 바꾼 내용\n"))
      (set-file-times file (time-add (current-time) 2))
      (should-error (imoogi-mothership-metadata-apply) :type 'user-error)
      (should (equal before (with-current-buffer source (buffer-string)))))))

(ert-deftest imoogi-mothership-form-legacy-heading-properties-remain-local ()
  (imoogi-test--with-metadata-form
    (imoogi-mothership-metadata-cancel)
    (erase-buffer)
    (insert "* 기존 제목\n:PROPERTIES:\n:TYPE: heading-type\n:MODIFIED: heading-date\n:END:\n본문\n")
    (imoogi-mothership-edit-metadata)
    (setq form (current-buffer))
    (widget-value-set (cdr (assoc "TYPE" imoogi-mothership--form-widgets)) "note")
    (imoogi-mothership-metadata-apply)
    (goto-char (point-min))
    (should (org-before-first-heading-p))
    (should (equal (org-entry-get nil "TYPE") "note"))
    (re-search-forward "^\\* 기존 제목")
    (should (equal (org-entry-get nil "TYPE") "heading-type"))
    (should (equal (org-entry-get nil "MODIFIED") "heading-date"))))

(ert-deftest imoogi-mothership-form-buttons-apply-and-cancel ()
  (imoogi-test--with-metadata-form
    (widget-value-set (cdr (assoc "AUTHOR" imoogi-mothership--form-widgets)) "버튼으로 적용")
    (goto-char (point-max))
    (search-backward "적용")
    (widget-button-press (point))
    (with-current-buffer source
      (should (equal (cadr (assoc "AUTHOR" (org-collect-keywords '("AUTHOR")))) "버튼으로 적용"))))
  (imoogi-test--with-metadata-form
    (let ((before (with-current-buffer source (buffer-string))))
      (widget-value-set (cdr (assoc "AUTHOR" imoogi-mothership--form-widgets)) "취소할 값")
      (goto-char (point-max))
      (search-backward "취소")
      (widget-button-press (point))
      (should (equal before (with-current-buffer source (buffer-string)))))))

(ert-deftest imoogi-mothership-form-preserves-affiliated-keyword-lines ()
  (imoogi-test--with-metadata-form
    (imoogi-mothership-metadata-cancel)
    (erase-buffer)
    (insert "#+CAPTION: Keep\n#+TITLE: Old\n* 본문\n")
    (imoogi-mothership-edit-metadata)
    (setq form (current-buffer))
    (widget-value-set (cdr (assoc "TITLE" imoogi-mothership--form-widgets)) "New")
    (imoogi-mothership-metadata-apply)
    (should (string-match-p "#\\+CAPTION: Keep\n#\\+TITLE: New\n\\* 본문" (buffer-string)))
    (should (equal (cdr (assoc "TITLE" (org-collect-keywords '("TITLE")))) '("New")))))

(ert-deftest imoogi-mothership-form-preserves-drawer-ending-at-eof ()
  (imoogi-test--with-metadata-form
    (imoogi-mothership-metadata-cancel)
    (erase-buffer)
    (insert ":PROPERTIES:\n:ID: original\n:END:")
    (imoogi-mothership-edit-metadata)
    (setq form (current-buffer))
    (widget-value-set (cdr (assoc "TITLE" imoogi-mothership--form-widgets)) "New")
    (imoogi-mothership-metadata-apply)
    (goto-char (point-min))
    (should (equal (org-entry-get nil "ID") "original"))
    (should (equal (cadr (assoc "TITLE" (org-collect-keywords '("TITLE")))) "New"))))

;;; mothership-form-test.el ends here
