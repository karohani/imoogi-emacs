;;; imoogi-targets.el --- Host-wide Anki sync target registration -*- lexical-binding: t; -*-

;;; Commentary:

;; Stores host-local sync targets outside project roots.  Directories are
;; recursive scan roots for the sync layer; files are explicit .org files.

;;; Code:

(require 'cl-lib)
(require 'json)
(require 'button)

(defgroup imoogi nil
  "Sync Org-mode headings to Anki flashcards via AnkiConnect."
  :group 'org
  :prefix "imoogi-")

(defcustom imoogi-targets-file
  (expand-file-name "imoogi-targets.json" user-emacs-directory)
  "Path to the host-wide imoogi target registration file."
  :type 'file
  :group 'imoogi)

(defvar imoogi-sync-root nil)
(defvar imoogi-exclude-patterns)

(declare-function imoogi-target-scan "imoogi-target-scan" (root targets patterns))
(declare-function imoogi-scan--excluded-p "imoogi-scan" (relative-path patterns))

(defun imoogi-targets--local-absolute-path-p (path)
  "Return non-nil when PATH is an absolute local filename string."
  (and (stringp path)
       (file-name-absolute-p path)
       (not (file-remote-p path))))

(defun imoogi-targets--normalize-kind (kind)
  "Return KIND as a target symbol, or signal `user-error'."
  (let ((symbol (cond
                 ((memq kind '(directory file)) kind)
                 ((string= kind "directory") 'directory)
                 ((string= kind "file") 'file)
                 (t nil))))
    (or symbol
        (user-error "imoogi: invalid target kind: %S" kind))))

(defun imoogi-targets--parse-record (record)
  "Parse one JSON RECORD into a target plist."
  (unless (listp record)
    (user-error "imoogi: malformed target record: %S" record))
  (let ((kind (imoogi-targets--normalize-kind (cdr (assq 'kind record))))
        (path (cdr (assq 'path record))))
    (unless (imoogi-targets--local-absolute-path-p path)
      (user-error "imoogi: invalid target path: %S" path))
    (list :kind kind :path path)))

(defun imoogi-targets-load (&optional file)
  "Return targets from FILE as plists, reading the file each time.

The result is a list of plists with `:kind' (`directory' or `file') and
`:path'.  Missing files return nil.  Malformed files signal
`user-error'.  Saved paths may no longer exist; load only validates the
stored schema and that paths are absolute local filenames."
  (let ((path (or file imoogi-targets-file)))
    (if (not (file-exists-p path))
        nil
      (condition-case nil
          (let* ((parsed (with-temp-buffer
                           (insert-file-contents path)
                          (json-parse-string (buffer-string)
                                             :object-type 'alist
                                             :array-type 'array)))
                 (entry (assq 'targets parsed))
                 (records (cdr entry)))
            (unless (and (listp parsed) entry (vectorp records))
              (user-error "imoogi: malformed targets file: %s" path))
            (mapcar #'imoogi-targets--parse-record (append records nil)))
        ((json-parse-error json-end-of-file)
         (user-error "imoogi: malformed targets file: %s" path))
        (file-error
         (user-error "imoogi: cannot read targets file: %s" path))
        (wrong-type-argument
         (user-error "imoogi: malformed targets file: %s" path))))))

(defun imoogi-targets--json-object (targets)
  "Return the JSON object form for TARGETS."
  (list (cons 'targets
              (vconcat
               (mapcar (lambda (target)
                         (list (cons 'kind (symbol-name (plist-get target :kind)))
                               (cons 'path (plist-get target :path))))
                       targets)))))

(defun imoogi-targets--write (targets &optional file)
  "Atomically write TARGETS to FILE."
  (let* ((path (or file imoogi-targets-file))
         (dir (file-name-directory path))
         (tmp nil))
    (make-directory dir t)
    (unwind-protect
        (progn
          (setq tmp (make-temp-file (expand-file-name ".imoogi-targets-" dir)
                                    nil ".json"))
          (with-temp-file tmp
            (let ((coding-system-for-write 'utf-8))
              (insert (json-serialize (imoogi-targets--json-object targets)))))
          (rename-file tmp path t)
          nil)
      (when (and tmp (file-exists-p tmp))
        (ignore-errors (delete-file tmp))))))

(defun imoogi-targets--validate-register-path (kind path)
  "Return canonical absolute PATH for KIND registration."
  (unless (imoogi-targets--local-absolute-path-p path)
    (user-error "imoogi: target must be a local absolute path: %s" path))
  (unless (file-exists-p path)
    (user-error "imoogi: target does not exist: %s" path))
  (unless (file-readable-p path)
    (user-error "imoogi: target is not readable: %s" path))
  (pcase kind
    ('directory
     (unless (file-directory-p path)
       (user-error "imoogi: target is not a directory: %s" path)))
    ('file
     (unless (file-regular-p path)
       (user-error "imoogi: target is not a regular file: %s" path))
     (unless (string-suffix-p ".org" path)
       (user-error "imoogi: target file must end in .org: %s" path)))
    (_ (user-error "imoogi: invalid target kind: %S" kind)))
  (if (eq kind 'directory)
      (directory-file-name (file-truename path))
    (file-truename path)))

(defun imoogi-targets--stored-canonical-path (kind path)
  "Return PATH's canonical form for KIND when possible."
  (condition-case nil
      (if (eq kind 'directory)
          (directory-file-name (file-truename path))
        (file-truename path))
    (file-error path)))

(defun imoogi-targets--add (kind path)
  "Register KIND/PATH and return the canonical path."
  (let* ((kind (imoogi-targets--normalize-kind kind))
         (canonical (imoogi-targets--validate-register-path kind
                                                            (expand-file-name path)))
         (targets (imoogi-targets-load))
         (already (cl-find-if
                   (lambda (target)
                     (and (eq (plist-get target :kind) kind)
                          (string= (imoogi-targets--stored-canonical-path
                                    kind
                                    (plist-get target :path))
                                   canonical)))
                   targets)))
    (unless already
      (imoogi-targets--write
       (append targets (list (list :kind kind :path canonical)))))
    canonical))

;;;###autoload
(defun imoogi-anki-register-directory (directory)
  "Register DIRECTORY as a recursive Anki sync target."
  (interactive "DRegister Anki directory: ")
  (message "imoogi: registered directory target %s"
           (imoogi-targets--add 'directory directory)))

;;;###autoload
(defun imoogi-anki-register-file (file)
  "Register FILE as an explicit .org Anki sync target."
  (interactive
   (list (read-file-name "Register Anki org file: "
                         nil nil t
                         (and buffer-file-name
                              (file-name-nondirectory buffer-file-name)))))
  (message "imoogi: registered file target %s"
           (imoogi-targets--add 'file file)))

(defun imoogi-targets--display-name (target)
  "Return a stable completion/display name for TARGET."
  (format "%s %s"
          (symbol-name (plist-get target :kind))
          (plist-get target :path)))

;;;###autoload
(defun imoogi-anki-unregister-target (target-name)
  "Remove TARGET-NAME from registration only.

This never deletes files, directories, Anki cards, or Org properties."
  (interactive
   (let* ((targets (imoogi-targets-load))
          (choices (mapcar #'imoogi-targets--display-name targets)))
     (list (completing-read "Unregister Anki target: " choices nil t))))
  (let* ((targets (imoogi-targets-load))
         (remaining (cl-remove-if
                     (lambda (target)
                       (string= (imoogi-targets--display-name target) target-name))
                     targets))
         (removed (- (length targets) (length remaining))))
    (when (zerop removed)
      (user-error "imoogi: no registered target named %s" target-name))
    (imoogi-targets--write remaining)
    (message "imoogi: unregistered %s" target-name)))

;;;###autoload
(defun imoogi-anki-list-targets ()
  "Show registered Anki sync targets in a read-only buffer."
  (interactive)
  (let ((targets (imoogi-targets-load)))
    (with-current-buffer (get-buffer-create "*imoogi Anki Targets*")
      (let ((inhibit-read-only t))
        (erase-buffer)
        (insert "imoogi Anki targets\n\n")
        (insert-text-button "파일 목록 펼치기"
                            'action (lambda (_) (imoogi-anki-list-files))
                            'follow-link t)
        (insert "\n\n")
        (if targets
            (dolist (target targets)
              (let* ((path (plist-get target :path))
                     (status (cond
                              ((not (file-exists-p path)) "missing")
                              ((not (file-readable-p path)) "unreadable")
                              (t "ok"))))
                (insert (format "%-9s %-10s %s\n"
                                (plist-get target :kind) status path))))
          (insert "No host-wide targets registered.\n"))
        (when imoogi-sync-root
          (insert (format "\nLegacy sync root: %s\n" imoogi-sync-root)))
        (setq buffer-read-only t)
        (goto-char (point-min)))
      (pop-to-buffer (current-buffer)))))

(define-derived-mode imoogi-anki-files-mode special-mode "Anki Files"
  "Browse resolved Anki files; RET opens a file and g refreshes the list.")
(define-key imoogi-anki-files-mode-map (kbd "g") #'imoogi-anki-list-files)

;;;###autoload
(defun imoogi-anki-list-files ()
  "List Org files in the legacy root and registered directories and files.
Resolve duplicates as sync does, including files without card headings.
Mark excluded files and paths that could not be scanned.  Never sync to Anki."
  (interactive)
  (require 'imoogi-target-scan)
  (let* ((patterns (and (boundp 'imoogi-exclude-patterns) imoogi-exclude-patterns))
         (groups (imoogi-target-scan imoogi-sync-root (imoogi-targets-load) patterns))
         (count 0))
    (with-current-buffer (get-buffer-create "*imoogi Anki Files*")
      (imoogi-anki-files-mode)
      (let ((inhibit-read-only t))
        (erase-buffer)
        (insert "imoogi Anki 파일 목록\nRET: 파일 열기   g: 새로고침   q: 닫기\n\n")
        (dolist (group groups)
          (let ((root (plist-get group :root))
                (scan (plist-get group :scan)))
            (insert (format "%s: %s\n"
                            (if (plist-get group :legacy) "기본 폴더" "등록 대상") root))
            (dolist (file (plist-get scan :files))
              (cl-incf count)
              (insert (if (imoogi-scan--excluded-p (file-relative-name file root) patterns)
                          "  [제외] " "  [대상] "))
              (insert-text-button file 'follow-link t 'file file
                                  'action (lambda (button)
                                            (find-file-other-window (button-get button 'file))))
              (insert "\n"))
            (dolist (path (plist-get scan :unreadable-files))
              (insert (format "  [확인 실패: 누락·읽기 불가·범위 밖 경로] %s\n" path)))
            (insert "\n")))
        (insert (format "총 %d개 파일 (제외 파일 포함). 카드 heading이 있는 항목만 동기화됩니다.\n" count))
        (goto-char (point-min)))
      (pop-to-buffer (current-buffer)))))

(provide 'imoogi-targets)
;;; imoogi-targets.el ends here
