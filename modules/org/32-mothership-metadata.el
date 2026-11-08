;;; 32-mothership-metadata.el --- Mothership metadata form -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "32-mothership-metadata" 'wid-edit 'org 'org-roam)
(require 'wid-edit)

(defconst imoogi-mothership--form-fields
  '(("TITLE" keyword "제목")
    ("TYPE" property "종류")
    ("ROAM_ALIASES" property "별칭 (여러 단어는 큰따옴표로 묶기)")
    ("DESCRIPTION" keyword "설명")
    ("AUTHOR" keyword "작성자")
    ("FILETAGS" keyword "태그 (:concept:history: 형식)")
    ("SOURCE" property "출처 (URL 또는 Org 링크)")
    ("RELATED" property "관련 노트 (Org 링크)")
    ("CONFIDENCE" property "신뢰도")
    ("LAYER" property "계층")
    ("STATUS" property "상태")
    ("ID" property "ID" read-only)
    ("CREATED" property "생성일" read-only)
    ("MODIFIED" property "수정일" read-only))
  "Metadata names, storage kinds, labels, and optional read-only markers.")

(defvar-local imoogi-mothership--form-source nil)
(defvar-local imoogi-mothership--form-source-file nil)
(defvar-local imoogi-mothership--form-source-tick nil)
(defvar-local imoogi-mothership--form-original nil)
(defvar-local imoogi-mothership--form-widgets nil)

(defvar imoogi-mothership-metadata-mode-map
  (let ((map (make-sparse-keymap)))
    (set-keymap-parent map widget-keymap)
    (define-key map (kbd "C-c C-c") #'imoogi-mothership-metadata-apply)
    (define-key map (kbd "C-c C-k") #'imoogi-mothership-metadata-cancel)
    map))

(defvar imoogi-mothership--form-field-map
  (let ((map (copy-keymap widget-field-keymap)))
    (define-key map (kbd "C-c C-c") #'imoogi-mothership-metadata-apply)
    (define-key map (kbd "C-c C-k") #'imoogi-mothership-metadata-cancel)
    map))

(define-derived-mode imoogi-mothership-metadata-mode fundamental-mode "Mothership Metadata"
  "Edit metadata widgets; apply with C-c C-c or cancel with C-c C-k."
  (setq-local truncate-lines nil
              word-wrap t))

(defun imoogi-mothership--keyword-nodes (tree name)
  "Return NAME keyword nodes in TREE's file preamble, ignoring body examples."
  (let ((limit (or (org-element-map tree 'headline
                    (lambda (node) (org-element-property :begin node)) nil t)
                   (point-max))))
    (org-element-map tree 'keyword
      (lambda (node)
        (when (and (< (org-element-property :begin node) limit)
                   (equal (org-element-property :key node) name))
          node)))))

(defun imoogi-mothership--form-read-values ()
  "Read metadata from the current Org file without changing it."
  (save-excursion
    (save-restriction
      (widen)
      (goto-char (point-min))
      (let ((tree (org-element-parse-buffer)))
        (mapcar
         (lambda (field)
           (let* ((name (car field))
                  (value (if (eq (cadr field) 'property)
                             (and (org-before-first-heading-p) (org-entry-get nil name))
                           (mapconcat
                            (lambda (node) (org-element-property :value node))
                            (imoogi-mothership--keyword-nodes tree name) " "))))
             (cons name (or value ""))))
         imoogi-mothership--form-fields)))))

(defun imoogi-mothership--form-label (text)
  "Insert protected TEXT outside editable widgets."
  (let ((start (point)))
    (widget-insert text)
    (add-text-properties start (point)
                         '(read-only t rear-nonsticky (read-only)))))

(defun imoogi-mothership-edit-metadata ()
  "Open a metadata form for the current Mothership note.
No changes reach the note until Apply; the note is not saved automatically."
  (interactive)
  (unless (imoogi-mothership--metadata-context-p)
    (user-error "모선 안의 Org 노트에서 사용하세요"))
  (barf-if-buffer-read-only)
  (let ((source (current-buffer))
        (file buffer-file-name)
        (tick (buffer-chars-modified-tick))
        (values (imoogi-mothership--form-read-values)))
    (pop-to-buffer (generate-new-buffer (format "*Mothership Metadata: %s*" (buffer-name source))))
    (imoogi-mothership-metadata-mode)
    (setq imoogi-mothership--form-source source
          imoogi-mothership--form-source-file file
          imoogi-mothership--form-source-tick tick
          imoogi-mothership--form-original values)
    (imoogi-mothership--form-label
     "모선 노트 메타데이터\nTAB/Shift-TAB으로 이동합니다. 텍스트는 한 줄로 입력하세요.\n적용: C-c C-c / 취소: C-c C-k / 노트 저장: 적용 후 C-x C-s\n\n")
    (dolist (field imoogi-mothership--form-fields)
      (let* ((name (car field))
             (value (cdr (assoc name values)))
             (choices (and (eq (cadr field) 'property)
                           (cdr (assoc name imoogi-mothership-property-values)))))
        (imoogi-mothership--form-label (format "%s (%s)\n" (nth 2 field) name))
        (if (nth 3 field)
            (imoogi-mothership--form-label (format "  %s  [읽기 전용]\n\n" value))
          (push
           (cons name
                 (if choices
                     (apply #'widget-create 'menu-choice :tag "선택" :value value
                            :keymap imoogi-mothership-metadata-mode-map
                            :format "%[%t%]: %v\n\n"
                            (mapcar (lambda (choice)
                                      (list 'const :tag (if (string-empty-p choice) "미지정" choice) choice))
                                    (delete-dups (append (list "" value) (copy-sequence choices)))))
                   (widget-create 'editable-field :size 60 :value value :format "%v\n\n"
                                  :keymap imoogi-mothership--form-field-map)))
           imoogi-mothership--form-widgets))))
    (widget-create 'push-button :keymap imoogi-mothership-metadata-mode-map
                   :notify (lambda (&rest _) (imoogi-mothership-metadata-apply)) "적용")
    (widget-insert "  ")
    (widget-create 'push-button :keymap imoogi-mothership-metadata-mode-map
                   :notify (lambda (&rest _) (imoogi-mothership-metadata-cancel)) "취소")
    (widget-insert "\n")
    (widget-setup)
    (set-buffer-modified-p nil)
    (goto-char (point-min))
    (widget-forward 1)))

(defun imoogi-mothership--form-put-keyword (name value)
  "Set preamble keyword NAME to VALUE, leaving body examples untouched."
  (let ((nodes (imoogi-mothership--keyword-nodes (org-element-parse-buffer) name)))
    (dolist (node (reverse (cdr nodes)))
      (goto-char (or (org-element-property :post-affiliated node)
                     (org-element-property :begin node)))
      (delete-region (line-beginning-position) (min (point-max) (1+ (line-end-position)))))
    (if nodes
        (progn
          (goto-char (or (org-element-property :post-affiliated (car nodes))
                         (org-element-property :begin (car nodes))))
          (delete-region (line-beginning-position) (line-end-position)))
      (goto-char (point-min))
      (org-roam-end-of-meta-data 'drawers)
      (unless (bolp) (end-of-line) (insert "\n")))
    (insert (format "#+%s: %s" name value))
    (unless nodes (insert "\n"))))

(defun imoogi-mothership-metadata-apply ()
  "Apply validated form values to the original note without saving it."
  (interactive)
  (unless (derived-mode-p 'imoogi-mothership-metadata-mode)
    (user-error "메타데이터 폼에서 사용하세요"))
  (let ((source imoogi-mothership--form-source)
        (file imoogi-mothership--form-source-file)
        (tick imoogi-mothership--form-source-tick)
        (original imoogi-mothership--form-original)
        (form (current-buffer))
        values)
    (unless (buffer-live-p source) (user-error "원본 노트가 닫혔습니다"))
    (dolist (entry imoogi-mothership--form-widgets)
      (let* ((name (car entry))
             (value (widget-value (cdr entry)))
             (choices (cdr (assoc name imoogi-mothership-property-values))))
        (unless (and (stringp value) (not (string-match-p "[\n\r\0]" value)))
          (user-error "%s에는 줄바꿈 없이 텍스트를 입력하세요" name))
        (when (and (equal name "TITLE") (string-empty-p (string-trim value))
                   (not (equal value (cdr (assoc name original)))))
          (user-error "제목을 입력하세요"))
        (when (and choices (not (string-empty-p value))
                   (not (member value choices)) (not (equal value (cdr (assoc name original)))))
          (user-error "%s의 허용된 값 중 하나를 선택하세요" name))
        (push (cons name value) values)))
    (with-current-buffer source
      (unless (and (imoogi-mothership--metadata-context-p) (equal file buffer-file-name)
                   (= tick (buffer-chars-modified-tick)) (verify-visited-file-modtime source))
        (user-error "원본 노트가 변경됐습니다. 폼을 닫고 다시 열어주세요"))
      (barf-if-buffer-read-only)
      (atomic-change-group
        (save-excursion
          (save-restriction
            (widen)
            (dolist (field imoogi-mothership--form-fields)
              (let* ((name (car field)) (value (cdr (assoc name values))))
                (when (and (not (nth 3 field)) (not (equal value (cdr (assoc name original)))))
                  (goto-char (point-min))
                  (when (org-at-heading-p) (insert "\n") (goto-char (point-min)))
                  (if (eq (cadr field) 'keyword)
                      (imoogi-mothership--form-put-keyword name value)
                    (org-entry-put nil name value)))))))
        (org-set-regexps-and-options)))
    (kill-buffer form)
    (pop-to-buffer source)
    (message "메타데이터를 적용했습니다. C-x C-s로 노트를 저장하세요")))

(defun imoogi-mothership-metadata-cancel ()
  "Close the form without changing the note."
  (interactive)
  (unless (derived-mode-p 'imoogi-mothership-metadata-mode)
    (user-error "메타데이터 폼에서 사용하세요"))
  (let ((source imoogi-mothership--form-source))
    (kill-buffer (current-buffer))
    (when (buffer-live-p source) (pop-to-buffer source))))

(provide 'imoogi-mothership-metadata)
;;; 32-mothership-metadata.el ends here
