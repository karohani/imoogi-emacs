;;; 29-org-roam.el --- Central notes and cached node completion -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "29-org-roam" 'org-roam 'memoize)

(require 'transient)
(require 'org-id)

(defconst imoogi-mothership-note-folders
  '("00. Inbox" "10. Process" "20. Literature Notes" "30. Permanent Notes"
    "40. Docs" "60. Collections" "70. Outputs")
  "Top-level note categories whose existing descendants can receive notes.")

(defcustom imoogi-mothership-property-values
  '(("TYPE" "note" "wiki-page" "literature-note" "permanent-note"
     "documentation" "collection" "output")
    ("STATUS" "draft" "active" "review" "archived")
    ("CONFIDENCE" "unreviewed" "low" "medium" "high"))
  "Allowed choices for Mothership metadata properties.
Add a property such as LAYER here to make it available in the picker."
  :type '(alist :key-type string :value-type (repeat string))
  :group 'imoogi)

(defun imoogi-mothership--metadata-context-p ()
  "Return non-nil in an Org file within the configured Mothership."
  (and (derived-mode-p 'org-mode) buffer-file-name
       (file-directory-p (expand-file-name imoogi-mothership-directory))
       (file-in-directory-p buffer-file-name
                            (expand-file-name imoogi-mothership-directory))))

(defun imoogi-mothership-set-property (&optional property value)
  "Select an enumerated file-level PROPERTY and its VALUE in Mothership.
Choices come from `imoogi-mothership-property-values'.  Do not save the buffer."
  (interactive)
  (unless (imoogi-mothership--metadata-context-p)
    (user-error "모선 안의 Org 노트에서 사용하세요"))
  (barf-if-buffer-read-only)
  (setq property (or property
                     (completing-read "메타데이터 항목: "
                                      (mapcar #'car imoogi-mothership-property-values)
                                      nil t)))
  (let ((choices (cdr (assoc property imoogi-mothership-property-values))))
    (unless (and choices (let ((case-fold-search nil))
                           (string-match-p "\\`[A-Z][A-Z0-9_]*\\'" property))
                 (not (member property '("ID" "CREATED" "MODIFIED"))))
      (user-error "목록으로 관리할 수 없는 메타데이터 항목입니다: %s" property))
    (setq value (or value (completing-read (format "%s 값: " property) choices nil t)))
    (unless (and (stringp value) (not (string-empty-p value))
                 (not (string-match-p "[\r\n]" value)) (member value choices))
      (user-error "%s의 허용된 값 중 하나를 선택하세요" property))
    (save-excursion
      (save-restriction
        (widen)
        (goto-char (point-min))
        (when (org-at-heading-p)
          (insert "\n")
          (goto-char (point-min)))
        (org-entry-put nil property value)))))

(defun imoogi-mothership--update-modified ()
  "Refresh an existing file-level MODIFIED property on Mothership saves."
  (when (imoogi-mothership--metadata-context-p)
    (save-excursion
      (save-restriction
        (widen)
        (goto-char (point-min))
        (when (and (org-before-first-heading-p) (org-entry-get nil "MODIFIED"))
          (org-entry-put nil "MODIFIED" (format-time-string "[%Y-%m-%d %a %H:%M]")))))))

(add-hook 'before-save-hook #'imoogi-mothership--update-modified)

(defun imoogi-mothership--root ()
  "Return the existing Mothership root, or explain how to initialize it."
  (let ((root (file-name-as-directory (expand-file-name imoogi-mothership-directory))))
    (unless (file-directory-p root)
      (user-error "먼저 M-x imoogi-mothership-setup을 실행하세요"))
    root))

(defun imoogi-mothership-open ()
  "Open the Mothership operating guide."
  (interactive)
  (let* ((root (imoogi-mothership--root))
         (file (expand-file-name "README.org" root)))
    (unless (file-regular-p file)
      (user-error "README.org가 없습니다. M-x imoogi-mothership-setup을 실행하세요"))
    (unless (file-in-directory-p file root)
      (user-error "README.org가 모선 밖의 파일을 가리킵니다"))
    (find-file file)))

(defun imoogi-mothership-browse ()
  "Browse Mothership in Dired."
  (interactive)
  (dired (imoogi-mothership--root)))

(defun imoogi-mothership--folders (&optional notes-only)
  "List existing relative folders, optionally under the note categories only.
Hidden and symbolic-link folders are excluded to avoid aliases and cycles."
  (let ((root (imoogi-mothership--root)) folders)
    (cl-labels
        ((walk (parent)
           (dolist (path (directory-files parent t "\\`[^.]" t))
             (when (and (file-directory-p path) (not (file-symlink-p path)))
               (push (file-relative-name path root) folders)
               (walk path)))))
      (if notes-only
          (dolist (name imoogi-mothership-note-folders)
            (let ((path (expand-file-name name root)))
              (when (and (file-directory-p path) (not (file-symlink-p path)))
                (push name folders)
                (walk path))))
        (walk root)))
    (sort folders #'string<)))

(defun imoogi-mothership-create-folder (parent name &optional open)
  "Create a numbered child of relative PARENT named NAME.
Numbers are local to PARENT and never include its prefix.  Existing numeric
prefixes, including files and links, are reserved.  With OPEN, browse the result."
  (interactive
   (list (completing-read "부모 폴더: " (imoogi-mothership--folders) nil t)
         (read-string "새 폴더 이름 (번호 제외): ") t))
  (let* ((root (imoogi-mothership--root))
         (parent-dir (expand-file-name parent root))
         (name (string-trim name)))
    (unless (and (member parent (imoogi-mothership--folders))
                 (file-in-directory-p parent-dir root))
      (user-error "모선 안의 기존 분류 폴더를 선택하세요"))
    (when (or (string-empty-p name) (member name '("." ".."))
              (string-match-p "[/\\\\[:cntrl:]]" name)
              (string-match-p "\\`[0-9]+\\.[ \t]" name))
      (user-error "번호·경로 문자·줄바꿈 없이 폴더 이름만 입력하세요"))
    (let ((lock (expand-file-name ".imoogi-folder-number.lock" parent-dir))
          (number 0) directory owned-lock)
      ;; Cooperating Emacs sessions must claim the parent before scanning.
      (unwind-protect
          (progn
            (let ((inhibit-quit t))
              (condition-case nil
                  (progn (make-directory lock) (setq owned-lock t))
                (file-already-exists
                 (user-error "이 폴더에 다른 생성 작업 또는 남아 있는 번호 잠금이 있습니다: %s" lock))))
            (dolist (entry (directory-files parent-dir nil "\\`[0-9]+\\."))
              (when (string-match "\\`\\([0-9]+\\)\\." entry)
                (setq number (max number (string-to-number (match-string 1 entry))))))
            (setq directory (expand-file-name (format "%02d. %s" (1+ number) name) parent-dir))
            (make-directory directory))
        (when owned-lock
          (let ((inhibit-quit t)) (delete-directory lock))))
      (when open (dired directory))
      (message "모선 폴더 생성: %s" (file-relative-name directory root))
      directory)))

(defun imoogi-mothership-create-note (folder title)
  "Create and visit a new Org note with TITLE in Mothership FOLDER.
Existing files and visiting buffers are preserved by adding a numeric suffix."
  (interactive
   (progn
     (imoogi-mothership--root)
     (list (completing-read "노트 폴더: " (imoogi-mothership--folders t) nil t)
           (read-string "노트 제목: "))))
  (let* ((root (imoogi-mothership--root))
         (directory (expand-file-name folder root))
         (title (string-trim title))
         (slug (replace-regexp-in-string "[^[:alnum:]_.-]+" "-" title)))
    (unless (and (member folder (imoogi-mothership--folders t))
                 (file-directory-p directory)
                 (file-in-directory-p directory root))
      (user-error "모선 안의 노트 폴더를 선택하세요. 빠진 폴더는 초기화 명령으로 복구하세요"))
    (when (or (string-empty-p title) (string-match-p "[/\\\n\r]" title)
              (member slug '("" "-" "." "..")))
      (user-error "경로 문자나 줄바꿈이 없는 노트 제목을 입력하세요"))
    (let ((file (expand-file-name (concat slug ".org") directory))
          (suffix 2)
          (id (org-id-new)))
      (while (or (file-exists-p file) (file-symlink-p file) (find-buffer-visiting file))
        (setq file (expand-file-name (format "%s-%d.org" slug suffix) directory)
              suffix (1+ suffix)))
      (let ((coding-system-for-write 'utf-8-unix)
            (timestamp (format-time-string "[%Y-%m-%d %a %H:%M]")))
        (write-region (format (concat ":PROPERTIES:\n:ID: %s\n:TYPE: note\n:ROAM_ALIASES:\n"
                                      ":CREATED: %s\n:MODIFIED: %s\n:SOURCE:\n:RELATED:\n"
                                      ":CONFIDENCE:\n:LAYER:\n:STATUS:\n:END:\n"
                                      "#+TITLE: %s\n#+AUTHOR: %s\n#+DESCRIPTION:\n#+FILETAGS:\n#+DATE: %s\n\n")
                              id timestamp timestamp title
                              (replace-regexp-in-string "[\r\n]+" " " user-full-name)
                              timestamp)
                      nil file nil 'silent nil 'excl))
      (org-id-add-location id file)
      (find-file file)
      (goto-char (point-max)))))

(defun imoogi-mothership-capture (title)
  "Create a quick Inbox note with TITLE."
  (interactive "s메모 제목: ")
  (imoogi-mothership-create-note "00. Inbox" title))

(defun imoogi-mothership--read-file ()
  "Select an existing Org file within Mothership, returning its full path."
  (let* ((root (imoogi-mothership--root))
         (files (cl-remove-if-not
                 (lambda (file) (file-in-directory-p file root))
                 (directory-files-recursively root "\\.org\\'")))
         (names (mapcar (lambda (file) (file-relative-name file root)) files)))
    (unless names (user-error "모선에 Org 파일이 없습니다"))
    (let* ((name (completing-read "모선 노트: " names nil t))
           (file (expand-file-name name root)))
      (unless (and (member name names) (file-regular-p file)
                   (file-in-directory-p file root))
        (user-error "모선 안의 기존 노트를 선택하세요"))
      file)))

(defun imoogi-mothership-find-file ()
  "Find an Org file by its relative path across Mothership."
  (interactive)
  (find-file (imoogi-mothership--read-file)))

(defun imoogi-mothership-search ()
  "Search Mothership contents with Consult and ripgrep."
  (interactive)
  (let ((root (imoogi-mothership--root)))
    (unless (executable-find "rg")
      (user-error "본문 검색에는 ripgrep(rg)이 필요합니다"))
    (require 'consult)
    (consult-ripgrep root)))

(defun imoogi-mothership-insert-link ()
  "Insert an Org ID link to a selected Mothership note.
For a saved note without an ID, add and save one before inserting the link."
  (interactive)
  (unless (derived-mode-p 'org-mode) (user-error "Org 버퍼에서 사용하세요"))
  (barf-if-buffer-read-only)
  (let* ((file (imoogi-mothership--read-file))
         (target (find-file-noselect file))
         id title)
    (with-current-buffer target
      (save-excursion
        (save-restriction
          (widen)
          (goto-char (point-min))
          (setq id (org-id-get nil)
                title (or (cadr (assoc "TITLE" (org-collect-keywords '("TITLE"))))
                          (file-name-base file)))
          (unless id
            (when (buffer-modified-p)
              (user-error "ID를 추가하기 전에 대상 노트의 변경 사항을 저장하세요"))
            (setq id (org-id-get-create))
            (save-buffer)))))
    (org-id-add-location id file)
    (insert (org-link-make-string (concat "id:" id) title))))

(defvar imoogi-org-roam--cache-timer nil
  "Idle timer that refreshes node completion candidates.")

(defun imoogi-org-roam-clear-completions-cache ()
  "Clear the cached node completion list after notes change."
  (interactive)
  (when (get 'org-roam-node-read--completions :memoize-original-function)
    (memoize-restore 'org-roam-node-read--completions)
    (memoize 'org-roam-node-read--completions "10 minute")))

(defun imoogi-org-roam-node-at-point-p ()
  "Return non-nil when point is on an Org-roam node."
  (and (derived-mode-p 'org-mode)
       (org-roam-node-at-point)))

(defun imoogi-org-roam-refile-context-p ()
  "Return non-nil when an Org heading or region can be refiled."
  (and (derived-mode-p 'org-mode)
       (or (org-at-heading-p) (org-region-active-p))))

(defun imoogi-org-roam--enable-db-autosync-if-directory-exists ()
  "Enable Org-roam autosync when the permanent notes directory exists."
  (when (file-directory-p org-roam-directory)
    (org-roam-db-autosync-mode 1)))

(use-package org-roam
  :ensure t
  :after org
  :custom
  (org-roam-directory (imoogi-org--permanent-directory))
  (org-roam-node-default-sort nil)
  :bind (("C-c n" . imoogi-mothership-transient))
  :config
  (require 'memoize)
  (unless (get 'org-roam-node-read--completions :memoize-original-function)
    (memoize 'org-roam-node-read--completions "10 minute"))
  (when (timerp imoogi-org-roam--cache-timer)
    (cancel-timer imoogi-org-roam--cache-timer))
  (setq imoogi-org-roam--cache-timer
        (run-with-idle-timer 60 t #'imoogi-org-roam-clear-completions-cache))
  (imoogi-org-roam--enable-db-autosync-if-directory-exists))

(transient-define-prefix imoogi-org-roam-transient ()
  "Work with central Org-roam notes."
  :column-widths '(23 23 23)
  [["탐색·연결"
      ("f" "노트 찾기·만들기" org-roam-node-find)
      ("n" "노트 캡처" org-roam-capture)
      ("i" "노트 링크 삽입" org-roam-node-insert :inapt-if-not (lambda () (derived-mode-p 'org-mode)))
      ("b" "백링크 보기" org-roam-buffer-toggle :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("R" "무작위 노트" org-roam-node-random)
      ("g" "노트 그래프" org-roam-graph :inapt-if-not (lambda () (executable-find "dot")))]
     ["현재 노트·제목"
      ("r" "다른 노트로 옮기기" org-roam-refile :inapt-if-not imoogi-org-roam-refile-context-p)
      ("a" "별칭 추가" org-roam-alias-add :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("A" "별칭 제거" org-roam-alias-remove :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("t" "태그 추가" org-roam-tag-add :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("T" "태그 제거" org-roam-tag-remove :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("e" "참조 추가" org-roam-ref-add :inapt-if-not imoogi-org-roam-node-at-point-p)
      ("E" "참조 제거" org-roam-ref-remove :inapt-if-not imoogi-org-roam-node-at-point-p)]
     ["일일 노트·관리"
      ("d" "오늘 일일 노트" org-roam-dailies-goto-today)
      ("D" "날짜로 일일 노트" org-roam-dailies-goto-date)
      ("s" "데이터베이스 동기화" org-roam-db-sync)
      ("c" "노드 후보 캐시 갱신" imoogi-org-roam-clear-completions-cache)
    ("q" "종료" transient-quit-one)]])

(transient-define-prefix imoogi-mothership-transient ()
  "Create, find, search, and connect Mothership notes."
  :column-widths '(23 23 23)
  [["시작·작성"
    ("o" "README 열기" imoogi-mothership-open)
    ("n" "새 노트" imoogi-mothership-create-note)
    ("c" "Inbox 빠른 메모" imoogi-mothership-capture)
    ("e" "메타데이터 폼" imoogi-mothership-edit-metadata
     :inapt-if-not (lambda () (and (imoogi-mothership--metadata-context-p)
                                  (not buffer-read-only))))
    ("E" "목록형 속성 선택" imoogi-mothership-set-property
     :inapt-if-not (lambda () (and (imoogi-mothership--metadata-context-p)
                                  (not buffer-read-only))))]
   ["탐색·연결"
    ("f" "파일명으로 찾기" imoogi-mothership-find-file)
    ("s" "본문 검색" imoogi-mothership-search)
    ("i" "노트 링크 삽입" imoogi-mothership-insert-link
     :inapt-if-not (lambda () (and (derived-mode-p 'org-mode) (not buffer-read-only))))]
   ["폴더·관리"
    ("+" "하위 폴더 만들기" imoogi-mothership-create-folder)
    ("d" "모선 폴더 열기" imoogi-mothership-browse)
    ("r" "Org-roam 메뉴" imoogi-org-roam-transient)
    ("S" "초기화·폴더 복구" imoogi-mothership-setup)
    ("q" "종료" transient-quit-one)]])

(transient-append-suffix 'imoogi-org-agenda-transient "m"
  '("r" "모선" imoogi-mothership-transient))

(provide 'imoogi-org-roam)
;;; 29-org-roam.el ends here
