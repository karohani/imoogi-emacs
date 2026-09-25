;;; 26-project-notes.el --- project-scoped Org notes -*- lexical-binding: t; -*-

;;; Code:

(imoogi-require "26-project-notes" 'cl-lib 'json 'org 'org-agenda 'org-id
                'project 'subr-x)

(require 'cl-lib)
(require 'json)
(require 'org)
(require 'org-agenda)
(require 'org-id)
(require 'project)
(require 'subr-x)

(defgroup imoogi-project-notes nil
  "Project-scoped note files outside source trees."
  :group 'imoogi)

(defcustom imoogi-project-notes-directory (expand-file-name "~/project-notes/")
  "Base directory for project-scoped notes."
  :type 'directory
  :group 'imoogi-project-notes)

(defcustom imoogi-project-notes-todo-storage 'project
  "Where new projects store their TODO source.
`project' creates and registers the project's tasks.org.
`central' keeps TODOs in the default ~/notes/agenda.org and leaves tasks.org as
a navigation file.  Changing this option does not migrate existing projects."
  :type '(choice (const :tag "Project tasks.org" project)
                 (const :tag "Central agenda.org" central))
  :group 'imoogi-project-notes)

(defcustom imoogi-project-notes-command '("imoogi-notes")
  "Command used for verified project-note filesystem operations."
  :type '(repeat string)
  :group 'imoogi-project-notes)

(defvar-local imoogi-project-notes-source-root nil
  "Source root associated with the current project notes buffer.")

(defvar-local imoogi-project-notes-directory-local nil
  "Project notes directory associated with the current buffer.")

(defvar-local imoogi-project-notes-entry-instance-id nil
  "Mounted project-note instance associated with the current buffer.")

(defvar-local imoogi-project-notes-active-p t
  "Whether the current project-note buffer has an available source context.")

(defvar-local imoogi-project-notes-inactive-reason nil
  "Reason the current mounted project-note buffer is inactive.")

(defvar-local imoogi-project-notes-force-edit-session-p nil
  "Whether this inactive buffer was explicitly unlocked for this session.")

(defvar imoogi-project-notes--detached-instance-ids nil
  "Mounted note instances hidden for the current Emacs session.")

(defvar imoogi-project-notes-unmount-preflight-function
  #'imoogi-project-notes--default-unmount-preflight
  "Function that resolves a managed root to a supported unmount descriptor.")

(defvar imoogi-project-notes-unmount-executor-function
  #'imoogi-project-notes--default-unmount-executor
  "Function that performs an already validated unmount descriptor.")

(defvar imoogi-project-notes-move-runner-function
  #'imoogi-project-notes--run-move-command
  "Function that asks the Go helper to move a verified note tree.")

(defconst imoogi-project-notes--metadata-file-name ".imoogi-project.json"
  "File that identifies the kind and layout of a project-notes directory.")

(defconst imoogi-project-notes--folder-name-regexp
  "\\`\\(?:[0-9]\\{4\\}\\.[0-9]+\\|[0-9]\\{2\\}\\.[0-9][0-9]\\|[0-9]\\{6\\}\\.[0-9][0-9]\\|\\(?:[0-9]\\{4\\}\\(?:\\.[0-9]+\\)?\\|[0-9]\\{2\\}\\.[0-9][0-9]\\|[0-9]\\{6\\}\\.[0-9][0-9]\\)-[[:alnum:]_.-]+\\)\\'"
  "Regexp for accepted project-notes folder names.")

(defun imoogi-project-notes--doctor-suggested-name (name)
  "Return a PC-friendly suggestion for invalid folder NAME."
  (if (string-match "\\`\\([0-9]\\{6\\}\\)-\\(.+\\)\\'" name)
      (format "%s-%s" (substring (match-string 1 name) 0 4)
              (match-string 2 name))
    name))

(defconst imoogi-project-notes--documents
  '((domain . ("domain.org" . "도메인 모델"))
    (architecture . ("architecture.org" . "아키텍처"))
    (decisions . ("decisions.org" . "결정 기록"))))

(defconst imoogi-project-notes--artifact-types
  '((research "조사" "research"
              "** 조사 질문\n\n** 확인한 자료\n\n** 비교와 근거\n\n** 결론\n\n** 남은 질문\n")
    (requirements "요구사항" "requirements"
                  "** 해결할 문제\n\n** 사용자 흐름\n\n** 완료 조건\n\n** 범위 밖\n\n** 미결 사항\n")
    (design "설계" "design"
            "** 목적\n\n** 현재 상태\n\n** 검토한 대안\n\n** 선택한 방안\n\n** 구현 범위\n\n** 검증 방법\n")
    (investigation "문제 분석" "investigation"
                   "** 증상\n\n** 재현 방법\n\n** 관찰한 증거\n\n** 원인\n\n** 해결과 검증\n")
    (decision "결정" "decision"
              "** 배경\n\n** 검토한 대안\n\n** 선택\n\n** 이유\n\n** 영향과 남는 제약\n")
    (meeting "회의" "meeting"
             "** 참석자와 목적\n\n** 논의 내용\n\n** 결정 사항\n\n** 후속 작업\n")
    (verification "검증 결과" "verification"
                  "** 검증 대상\n\n** 환경과 방법\n\n** 결과\n\n** 발견한 문제\n\n** 결론\n")
    (runbook "작업 절차" "runbook"
             "** 목적\n\n** 사전 조건\n\n** 절차\n\n** 확인 방법\n\n** 실패 시 복구\n")
    (note "빈 문서" "note" "** 내용\n"))
  "Artifact templates selectable from a project task.")

(defun imoogi-project-notes--registry-file ()
  "Return the project notes registry path."
  (locate-user-emacs-file ".cache/project-notes.json"))

(defun imoogi-project-notes--mounted-roots-file ()
  "Return the host-local mounted roots registry path."
  (locate-user-emacs-file ".cache/project-notes-mounted-roots.json"))

(defun imoogi-project-notes--metadata-file (directory)
  "Return the metadata file below project-notes DIRECTORY."
  (expand-file-name imoogi-project-notes--metadata-file-name directory))

(defun imoogi-project-notes--template-file (name)
  "Return template NAME's absolute path."
  (expand-file-name (format "templates/project-notes/%s.org" name)
                    imoogi-emacs-dir))

(defun imoogi-project-notes--directory-file-name (directory)
  "Return DIRECTORY in canonical file-name form."
  (file-name-as-directory (expand-file-name directory)))

(defun imoogi-project-notes--truename-if-present (path)
  "Return PATH's truename when possible, otherwise expanded PATH."
  (let ((expanded (expand-file-name path)))
    (if (file-exists-p expanded)
        (file-truename expanded)
      expanded)))

(defun imoogi-project-notes--call-git (root &rest args)
  "Run git ARGS in ROOT and return trimmed stdout, or nil on failure."
  (when (executable-find "git")
    (with-temp-buffer
      (let ((default-directory root))
        (when (zerop (apply #'call-process "git" nil t nil args))
          (string-trim (buffer-string)))))))

(defun imoogi-project-notes--git-common-dir (root)
  "Return ROOT's canonical git common directory, or nil outside Git."
  (when-let* ((common (imoogi-project-notes--call-git
                      root "rev-parse" "--git-common-dir")))
    (imoogi-project-notes--truename-if-present
     (if (file-name-absolute-p common)
         common
       (expand-file-name common root)))))

(defun imoogi-project-notes--identity-key (root)
  "Return a stable identity key for source ROOT.
Git worktrees share the same key by using Git's common directory."
  (let ((root (imoogi-project-notes--directory-file-name root)))
    (when (file-remote-p root)
      (user-error "원격 프로젝트는 project-notes 대상으로 등록하지 않습니다: %s" root))
    (concat (if-let* ((git-common (imoogi-project-notes--git-common-dir root)))
                (concat "git:" git-common)
              (concat "dir:" (imoogi-project-notes--truename-if-present root))))))

(defun imoogi-project-notes--project-root ()
  "Return the current source project root, prompting when needed."
  (if-let* ((project (project-current nil)))
      (project-root project)
    (read-directory-name "소스 프로젝트 폴더: " default-directory nil t)))

(defun imoogi-project-notes--slug (name)
  "Convert NAME into a conservative directory slug."
  (let* ((downcased (downcase (or name "project")))
         (ascii (replace-regexp-in-string "[^[:alnum:]._-]+" "-" downcased))
         (trimmed (string-trim ascii "-+" "-+")))
    (if (string-empty-p trimmed) "project" trimmed)))

(defun imoogi-project-notes--alist-string (key alist)
  "Return string value for KEY in ALIST."
  (let ((value (alist-get key alist)))
    (and (stringp value) value)))

(defun imoogi-project-notes--valid-entry-p (entry)
  "Return non-nil when ENTRY is safe plain registry data."
  (and (listp entry)
       (imoogi-project-notes--alist-string 'key entry)
       (imoogi-project-notes--alist-string 'source-root entry)
       (imoogi-project-notes--alist-string 'notes-dir entry)
       (imoogi-project-notes--alist-string 'project-file entry)
       (imoogi-project-notes--alist-string 'tasks-file entry)
       (imoogi-project-notes--alist-string 'journal-file entry)))

(defun imoogi-project-notes--valid-metadata-p (metadata)
  "Return non-nil when METADATA describes a supported notes directory."
  (and (listp metadata)
       (equal (alist-get 'schema_version metadata) 1)
       (member (imoogi-project-notes--alist-string 'type metadata)
               '("project" "study"))
       (imoogi-project-notes--alist-string 'key metadata)
       (imoogi-project-notes--alist-string 'name metadata)
       (imoogi-project-notes--alist-string 'overview metadata)
       (imoogi-project-notes--alist-string 'tasks metadata)
       (imoogi-project-notes--alist-string 'journal metadata)
       (let ((storage (imoogi-project-notes--alist-string
                       'todo_storage metadata)))
         (or (null storage) (member storage '("project" "central"))))
       (if (string= (imoogi-project-notes--alist-string 'type metadata) "study")
           (imoogi-project-notes--alist-string 'study_id metadata)
         t)))

(defun imoogi-project-notes--read-metadata-file (file &optional noerror)
  "Read and validate project-notes metadata FILE.
When NOERROR is non-nil, return nil and warn instead of signaling."
  (condition-case err
      (let ((json-object-type 'alist)
            (json-array-type 'list)
            (json-key-type 'symbol)
            (metadata (json-read-file file)))
        (unless (imoogi-project-notes--valid-metadata-p metadata)
          (error "unsupported or incomplete metadata"))
        metadata)
    (error
     (if noerror
         (progn
           (display-warning
            'imoogi
            (format "프로젝트 노트 메타데이터를 읽지 못해 건너뜀: %s (%s)"
                    file (error-message-string err))
            :warning)
           nil)
       (user-error "프로젝트 노트 메타데이터가 잘못되었습니다: %s (%s)"
                   file (error-message-string err))))))

(defun imoogi-project-notes--find-metadata-file (&optional path)
  "Find project-notes metadata above PATH or the current buffer."
  (let* ((path (or path buffer-file-name default-directory))
         (directory (if (and path (not (file-directory-p path)))
                        (file-name-directory path)
                      path)))
    (when (and directory (not (file-remote-p directory)))
      (when-let* ((root (locate-dominating-file
                         directory imoogi-project-notes--metadata-file-name)))
        (imoogi-project-notes--metadata-file root)))))

(defun imoogi-project-notes--metadata-entry (file metadata)
  "Build a registry-compatible entry from METADATA stored in FILE."
  (let* ((root (file-name-directory file))
         (type (imoogi-project-notes--alist-string 'type metadata))
         (resolve
          (lambda (key)
            (let* ((relative (imoogi-project-notes--alist-string key metadata))
                   (target (expand-file-name relative root)))
              (when (or (file-name-absolute-p relative)
                        (not (file-in-directory-p target root)))
                (user-error "메타데이터의 %s 경로가 노트 폴더 밖을 가리킵니다: %s"
                            key relative))
              target)))
         (source-root (if (string= type "study")
                          root
                        (or (imoogi-project-notes--alist-string
                             'source_root metadata)
                            root))))
    `((key . ,(imoogi-project-notes--alist-string 'key metadata))
      (type . ,type)
      (name . ,(imoogi-project-notes--alist-string 'name metadata))
      ,@(when-let* ((study-id (imoogi-project-notes--alist-string
                               'study_id metadata)))
          `((study-id . ,study-id)))
      (source-root . ,(imoogi-project-notes--directory-file-name source-root))
      (source-declared-p . ,(or (string= type "study")
                                (and (assq 'source_root metadata) t)))
      (notes-dir . ,(imoogi-project-notes--directory-file-name root))
      (project-file . ,(funcall resolve 'overview))
      (tasks-file . ,(funcall resolve 'tasks))
      (journal-file . ,(funcall resolve 'journal))
      (todo-storage . ,(or (imoogi-project-notes--alist-string
                            'todo_storage metadata)
                           "project")))))

(defun imoogi-project-notes--current-metadata-entry ()
  "Return an entry derived from the nearest folder metadata, if any."
  (when-let* ((file (imoogi-project-notes--find-metadata-file))
              (metadata (imoogi-project-notes--read-metadata-file file 'noerror)))
    (imoogi-project-notes--metadata-entry file metadata)))

(defun imoogi-project-notes--read-registry (&optional noerror)
  "Read and validate the project notes registry.
Missing registries are empty.  Invalid registries signal `user-error' unless
NOERROR is non-nil."
  (let ((file (imoogi-project-notes--registry-file)))
    (if (file-exists-p file)
        (condition-case err
            (let* ((json-object-type 'alist)
                   (json-array-type 'list)
                   (json-key-type 'symbol)
                   (data (json-read-file file))
                   (version (alist-get 'version data))
                   (projects (alist-get 'projects data)))
              (unless (equal version 1)
                (error "Registry version must be 1"))
              (unless (assq 'projects data)
                (error "Registry projects key is missing"))
              (unless (listp projects)
                (error "Registry projects must be a list"))
              (let ((valid (cl-remove-if-not
                            #'imoogi-project-notes--valid-entry-p projects)))
                (unless (= (length projects) (length valid))
                  (error "Registry contains malformed project entries"))
                valid))
          (error
           (if noerror
               (progn
                 (display-warning
                  'imoogi
                  (format "프로젝트 노트 레지스트리를 읽지 못해 건너뜀: %s"
                          (error-message-string err))
                  :warning)
                 nil)
             (user-error "프로젝트 노트 레지스트리가 손상되었습니다: %s"
                         (error-message-string err)))))
      nil)))

(defun imoogi-project-notes--write-registry (entries)
  "Write ENTRIES to the project notes registry."
  (let ((file (imoogi-project-notes--registry-file)))
    (make-directory (file-name-directory file) t)
    (with-temp-file file
      (let ((json-encoding-pretty-print t))
        (insert (json-encode `((version . 1)
                               (projects . ,entries))))
        (insert "\n")))))

(defun imoogi-project-notes--empty-mounted-state ()
  "Return a new empty mounted roots registry value."
  '((version . 1) (roots . nil) (source-overrides . nil)))

(defun imoogi-project-notes--valid-mounted-root-p (root)
  "Return non-nil when ROOT is safe mounted-root registry data."
  (let ((path (and (listp root)
                   (imoogi-project-notes--alist-string 'path root))))
    (and path
         (not (file-remote-p path))
         (imoogi-project-notes--alist-string 'id root)
         (imoogi-project-notes--alist-string 'label root)
         (memq (alist-get 'enabled root) '(t :json-false)))))

(defun imoogi-project-notes--valid-source-override-p (override)
  "Return non-nil when OVERRIDE is safe host-local override data."
  (let ((source-root
         (and (listp override)
              (imoogi-project-notes--alist-string 'source-root override))))
    (and source-root
         (not (file-remote-p source-root))
         (imoogi-project-notes--alist-string 'instance-id override))))

(defun imoogi-project-notes--read-mounted-state (&optional noerror)
  "Read and validate the host-local mounted roots registry.
Missing registries return an empty state.  When NOERROR is non-nil, warn and
return an empty state instead of signaling for malformed data."
  (let ((file (imoogi-project-notes--mounted-roots-file)))
    (if (not (file-exists-p file))
        (imoogi-project-notes--empty-mounted-state)
      (condition-case err
          (let* ((json-object-type 'alist)
                 (json-array-type 'list)
                 (json-key-type 'symbol)
                 (json-false :json-false)
                 (data (json-read-file file))
                 (roots (alist-get 'roots data))
                 (overrides (alist-get 'source-overrides data)))
            (unless (equal (alist-get 'version data) 1)
              (error "Mounted roots registry version must be 1"))
            (unless (and (assq 'roots data) (listp roots)
                         (assq 'source-overrides data) (listp overrides))
              (error "Mounted roots registry keys are missing or malformed"))
            (unless (cl-every #'imoogi-project-notes--valid-mounted-root-p roots)
              (error "Mounted roots registry contains malformed roots"))
            (unless (cl-every #'imoogi-project-notes--valid-source-override-p
                              overrides)
              (error "Mounted roots registry contains malformed overrides"))
            data)
        (error
         (if noerror
             (progn
               (display-warning
                'imoogi
                (format "외장 노트 루트 레지스트리를 읽지 못해 건너뜀: %s"
                        (error-message-string err))
                :warning)
               (imoogi-project-notes--empty-mounted-state))
           (user-error "외장 노트 루트 레지스트리가 손상되었습니다: %s"
                       (error-message-string err))))))))

(defun imoogi-project-notes--write-mounted-state (state)
  "Validate and atomically write mounted roots registry STATE."
  (let ((file (imoogi-project-notes--mounted-roots-file))
        (roots (alist-get 'roots state))
        (overrides (alist-get 'source-overrides state)))
    (unless (and (equal (alist-get 'version state) 1)
                 (listp roots)
                 (cl-every #'imoogi-project-notes--valid-mounted-root-p roots)
                 (listp overrides)
                 (cl-every #'imoogi-project-notes--valid-source-override-p
                           overrides))
      (user-error "외장 노트 루트 레지스트리 데이터가 잘못되었습니다"))
    (make-directory (file-name-directory file) t)
    (let ((temporary (make-temp-file
                      (expand-file-name ".project-notes-mounted-roots-"
                                        (file-name-directory file)))))
      (unwind-protect
          (progn
            (with-temp-file temporary
              (let ((json-encoding-pretty-print t))
                (insert (json-encode state) "\n")))
            (rename-file temporary file t))
        (when (file-exists-p temporary)
          (delete-file temporary))))))

(defun imoogi-project-notes--canonical-directory (directory)
  "Return existing local DIRECTORY in canonical directory form."
  (let ((expanded (imoogi-project-notes--directory-file-name directory)))
    (when (file-remote-p expanded)
      (user-error "원격 폴더는 외장 노트 루트로 등록할 수 없습니다: %s"
                  expanded))
    (unless (file-directory-p expanded)
      (user-error "외장 노트 루트 폴더가 없습니다: %s" expanded))
    (imoogi-project-notes--directory-file-name (file-truename expanded))))

(defun imoogi-project-notes--directory-overlap-p (left right)
  "Return non-nil when canonical directories LEFT and RIGHT overlap."
  (or (string= left right)
      (file-in-directory-p left right)
      (file-in-directory-p right left)))

(defun imoogi-project-notes--validate-mounted-root (directory &optional roots)
  "Return canonical DIRECTORY after checking it against registered ROOTS."
  (let* ((canonical (imoogi-project-notes--canonical-directory directory))
         (roots (or roots
                    (alist-get 'roots
                               (imoogi-project-notes--read-mounted-state)))))
    (dolist (root roots)
      (when (and (not (eq (alist-get 'enabled root) :json-false))
                 (imoogi-project-notes--directory-overlap-p
                  canonical
                  (imoogi-project-notes--canonical-directory
                   (imoogi-project-notes--alist-string 'path root))))
        (user-error "이미 등록된 외장 노트 루트와 중복되거나 겹칩니다: %s"
                    (imoogi-project-notes--alist-string 'path root))))
    canonical))

(defun imoogi-project-notes--mounted-root-id (canonical-root)
  "Return stable host-local id for CANONICAL-ROOT."
  (secure-hash 'sha1 canonical-root))

(defun imoogi-project-notes--mounted-instance-id (root-id notes-directory)
  "Return stable instance id below ROOT-ID for NOTES-DIRECTORY."
  (secure-hash
   'sha1
   (concat root-id "\0"
           (imoogi-project-notes--directory-file-name
            (file-truename notes-directory)))))

(defun imoogi-project-notes--decorate-local-entry (entry)
  "Return a copied local registry ENTRY with read-side identity fields."
  (let ((copy (copy-tree entry)))
    (push '(origin . local) copy)
    (push (cons 'logical-key (alist-get 'key copy)) copy)
    (push (cons 'instance-id (concat "local:" (alist-get 'key copy))) copy)
    copy))

(defun imoogi-project-notes--source-override (instance-id state)
  "Return source override for INSTANCE-ID from mounted registry STATE."
  (cl-find instance-id (alist-get 'source-overrides state)
           :key (lambda (override) (alist-get 'instance-id override))
           :test #'string=))

(defun imoogi-project-notes--decorate-mounted-status (entry state)
  "Return mounted ENTRY decorated with source status from STATE."
  (let* ((copy (copy-tree entry))
         (instance-id (alist-get 'instance-id copy))
         (override (imoogi-project-notes--source-override instance-id state))
         (override-root (and override (alist-get 'source-root override)))
         (metadata-root (alist-get 'source-root copy))
         (study-p (string= (alist-get 'type copy) "study"))
         (effective
          (cond
           ((and override-root
                 (not (file-remote-p override-root))
                 (file-directory-p override-root))
            override-root)
           (study-p (alist-get 'notes-dir copy))
           ((and (alist-get 'source-declared-p copy)
                 metadata-root
                 (not (file-remote-p metadata-root))
                 (file-directory-p metadata-root))
            metadata-root)))
         (source-origin
          (cond ((and override-root
                      (not (file-remote-p override-root))
                      (file-directory-p override-root))
                 'local-override)
                (study-p 'notes-directory)
                (effective 'metadata)
                (t 'missing))))
    (push (cons 'effective-source-root
                (and effective
                     (imoogi-project-notes--directory-file-name effective)))
          copy)
    (push (cons 'source-origin source-origin) copy)
    (push (cons 'active-p (and effective t)) copy)
    (unless effective
      (push '(inactive-reason . missing-source) copy))
    copy))

(defun imoogi-project-notes--scan-mounted-root (root)
  "Return valid metadata entries discovered below registered ROOT.
Invalid metadata is warned about and skipped."
  (let* ((path (imoogi-project-notes--alist-string 'path root))
         (root-id (imoogi-project-notes--alist-string 'id root))
         entries)
    (when (and (not (eq (alist-get 'enabled root) :json-false))
               (file-directory-p path))
      (condition-case err
          (dolist (file (directory-files-recursively
                         path
                         (concat "\\`"
                                 (regexp-quote
                                  imoogi-project-notes--metadata-file-name)
                                 "\\'")
                         nil nil))
            (when-let* ((metadata
                         (imoogi-project-notes--read-metadata-file file 'noerror)))
              (condition-case metadata-error
                  (let* ((entry (imoogi-project-notes--metadata-entry file metadata))
                         (notes-dir (alist-get 'notes-dir entry)))
                    (push (cons 'origin 'mounted) entry)
                    (push (cons 'logical-key (alist-get 'key entry)) entry)
                    (push (cons 'mounted-root-id root-id) entry)
                    (push (cons 'mounted-root path) entry)
                    (push (cons 'device-label
                                (imoogi-project-notes--alist-string 'label root))
                          entry)
                    (push (cons 'instance-id
                                (imoogi-project-notes--mounted-instance-id
                                 root-id notes-dir))
                          entry)
                    (push entry entries))
                (error
                 (display-warning
                  'imoogi
                  (format "외장 프로젝트 노트 메타데이터를 건너뜀: %s (%s)"
                          file (error-message-string metadata-error))
                  :warning)))))
        (file-error
         (display-warning
          'imoogi
          (format "외장 노트 루트를 검색하지 못해 건너뜀: %s (%s)"
                  path (error-message-string err))
          :warning))))
    (nreverse entries)))

(defun imoogi-project-notes--all-entries ()
  "Return local and currently available mounted project-note entries.
Local registry entries win when the same canonical notes directory is also
found below a mounted root.  Equal logical keys on different mounted roots
remain distinct through their `instance-id'."
  (let* ((locals (mapcar #'imoogi-project-notes--decorate-local-entry
                         (imoogi-project-notes--read-registry)))
         (local-dirs
          (mapcar (lambda (entry)
                    (imoogi-project-notes--directory-file-name
                     (imoogi-project-notes--truename-if-present
                      (alist-get 'notes-dir entry))))
                  locals))
         (state (imoogi-project-notes--read-mounted-state 'noerror))
         (mounted
          (mapcar (lambda (entry)
                    (imoogi-project-notes--decorate-mounted-status entry state))
                  (apply #'append
                         (mapcar #'imoogi-project-notes--scan-mounted-root
                                 (alist-get 'roots state)))))
         (seen nil)
         result)
    (dolist (entry (append locals mounted))
      (let* ((origin (alist-get 'origin entry))
             (notes-dir
              (imoogi-project-notes--directory-file-name
               (imoogi-project-notes--truename-if-present
                (alist-get 'notes-dir entry))))
             (instance-id (alist-get 'instance-id entry)))
        (unless (or (and (eq origin 'mounted) (member notes-dir local-dirs))
                    (and (eq origin 'mounted)
                         (member instance-id
                                 imoogi-project-notes--detached-instance-ids))
                    (member instance-id seen))
          (push instance-id seen)
          (push entry result))))
    (nreverse result)))

(defun imoogi-project-notes--find-entry-by-key (key &optional entries)
  "Return registry entry matching KEY."
  (cl-find key (or entries (imoogi-project-notes--read-registry))
           :key (lambda (entry) (alist-get 'key entry))
           :test #'string=))

(defun imoogi-project-notes--find-entry-by-notes-file (file &optional entries)
  "Return registry entry whose notes directory contains FILE."
  (let ((expanded (expand-file-name file)))
    (cl-find-if
     (lambda (entry)
       (let ((dir (imoogi-project-notes--alist-string 'notes-dir entry)))
         (and dir (file-in-directory-p expanded dir))))
     (or entries (imoogi-project-notes--read-registry)))))

(defun imoogi-project-notes--find-entry-by-notes-directory
    (directory &optional entries)
  "Return the registry entry whose notes directory is DIRECTORY."
  (let ((directory (file-truename directory)))
    (cl-find-if
     (lambda (entry)
       (let ((notes-dir (imoogi-project-notes--alist-string 'notes-dir entry)))
         (and notes-dir (equal directory (file-truename notes-dir)))))
     (or entries (imoogi-project-notes--read-registry)))))

(defun imoogi-project-notes--current-entry ()
  "Return the registry entry for the current source or notes buffer."
  (let ((entries (imoogi-project-notes--all-entries)))
    (or (and imoogi-project-notes-entry-instance-id
             (cl-find imoogi-project-notes-entry-instance-id entries
                      :key (lambda (entry) (alist-get 'instance-id entry))
                      :test #'string=))
        (and imoogi-project-notes-source-root
             (imoogi-project-notes--find-entry-by-key
              (imoogi-project-notes--identity-key imoogi-project-notes-source-root)
              entries))
        (and buffer-file-name
             (imoogi-project-notes--find-entry-by-notes-file buffer-file-name entries))
        (when-let* ((metadata-entry
                     (imoogi-project-notes--current-metadata-entry)))
          (or (imoogi-project-notes--find-entry-by-key
               (imoogi-project-notes--alist-string 'key metadata-entry) entries)
              metadata-entry))
        (when-let* ((project (project-current nil)))
          (imoogi-project-notes--find-entry-by-key
           (imoogi-project-notes--identity-key (project-root project))
           entries)))))

(defun imoogi-project-notes--current-source-root (&optional entry)
  "Return the source root that should be used for this command.
When invoked from a Git worktree, this keeps journal resume sections distinct
even though the project notes registry key is shared across worktrees."
  (or (when-let* ((project (project-current nil)))
        (project-root project))
      imoogi-project-notes-source-root
      (and entry (imoogi-project-notes--alist-string
                  'effective-source-root entry))
      (and entry (imoogi-project-notes--alist-string 'source-root entry))))

(defun imoogi-project-notes--notes-directory-in-use-p (directory key entries)
  "Return non-nil when DIRECTORY is already used by another registry KEY."
  (let ((directory (imoogi-project-notes--directory-file-name directory)))
    (cl-some
     (lambda (entry)
       (and (not (string= key (alist-get 'key entry)))
            (string= directory
                     (imoogi-project-notes--directory-file-name
                      (imoogi-project-notes--alist-string 'notes-dir entry)))))
     entries)))

(defun imoogi-project-notes--validate-source-root (root)
  "Validate ROOT as a local source project directory."
  (let ((root (imoogi-project-notes--directory-file-name root)))
    (when (file-remote-p root)
      (user-error "원격 프로젝트는 project-notes 대상으로 등록하지 않습니다: %s" root))
    (unless (file-directory-p root)
      (user-error "소스 프로젝트 폴더가 없습니다: %s" root))
    (when (file-in-directory-p
           root
           (imoogi-project-notes--directory-file-name
            imoogi-project-notes-directory))
      (user-error
       "문서 폴더를 작업 프로젝트로 등록할 수 없습니다: %s" root))
    root))

(defun imoogi-project-notes--validate-notes-directory (directory key entries)
  "Validate DIRECTORY as a notes directory for registry KEY."
  (let ((directory (imoogi-project-notes--directory-file-name directory)))
    (when (file-remote-p directory)
      (user-error "원격 폴더는 project-notes 폴더로 등록하지 않습니다: %s" directory))
    (when (imoogi-project-notes--notes-directory-in-use-p directory key entries)
      (user-error "이미 다른 프로젝트가 사용하는 노트 폴더입니다: %s" directory))
    directory))

(defun imoogi-project-notes--start-date ()
  "Return the date prefix used for a newly created project notes directory."
  (format-time-string "%y%m%d"))

(defun imoogi-project-notes--study-year ()
  "Return the two-digit year used in a new study identifier."
  (format-time-string "%y"))

(defun imoogi-project-notes--next-study-id (&optional year additional-root)
  "Return the next YEAR.NN study identifier across available notes roots.
Scan the local notes root, registered mounted roots, and ADDITIONAL-ROOT when
provided so a portable study note cannot reuse an existing identifier."
  (let* ((year (or year (imoogi-project-notes--study-year)))
         (regexp (format "\\`%s\\.\\([0-9][0-9]\\)-" (regexp-quote year)))
         (mounted-state (imoogi-project-notes--read-mounted-state 'noerror))
         (roots (cons imoogi-project-notes-directory
                      (mapcar (lambda (root) (alist-get 'path root))
                              (alist-get 'roots mounted-state))))
         (maximum 0))
    (when additional-root
      (push additional-root roots))
    (dolist (root (delete-dups (delq nil roots)))
      (when (file-directory-p root)
        (dolist (name (directory-files root nil regexp))
          (when (string-match regexp name)
            (setq maximum
                  (max maximum (string-to-number (match-string 1 name))))))))
    (when (>= maximum 99)
      (user-error "%s년 학습 노트가 99개를 초과했습니다" year))
    (format "%s.%02d" year (1+ maximum))))

(defun imoogi-project-notes--default-notes-directory (root key entries)
  "Return the default notes directory for ROOT and KEY."
  (let* ((base (imoogi-project-notes--directory-file-name
                imoogi-project-notes-directory))
         (slug (imoogi-project-notes--slug
                (file-name-nondirectory (directory-file-name root))))
         (dated-slug (format "%s-%s" (imoogi-project-notes--start-date) slug))
         (plain (expand-file-name (file-name-as-directory dated-slug) base)))
    (if (or (file-exists-p plain)
            (imoogi-project-notes--notes-directory-in-use-p plain key entries))
        (expand-file-name
         (file-name-as-directory
          (format "%s-%s" dated-slug (substring (secure-hash 'sha1 key) 0 10)))
         base)
      plain)))

(defun imoogi-project-notes--folder-name-valid-p (name)
  "Return non-nil when NAME follows the project numbering convention."
  (and (stringp name)
       (string-match-p imoogi-project-notes--folder-name-regexp name)))

(defun imoogi-project-notes--folder-id-key (name)
  "Return NAME's scoped numeric ID, or nil when it has no dotted ID.
The date/month prefix is part of the scope, so `26.01' and `260925.01'
remain independent sequences."
  (when (and (stringp name)
             (string-match
              "\\`\\([0-9]+\\.[0-9]+\\)\\(?:-[[:alnum:]_.-]+\\)?\\'"
              name))
    (match-string 1 name)))

(defun imoogi-project-notes--numbered-notes-directory (number root)
  "Return a notes directory named from NUMBER and source ROOT."
  (when (or (string-empty-p (string-trim number))
            (string-match-p "[/\\\\]" number))
    (user-error "프로젝트 번호를 입력하세요 (예: 260925.01 또는 2609)"))
  (let* ((slug (imoogi-project-notes--slug
                (file-name-nondirectory (directory-file-name root))))
         (name (if (string-match-p "-" number)
                   number
                 (format "%s-%s" number slug))))
    (unless (imoogi-project-notes--folder-name-valid-p name)
      (user-error "프로젝트 번호 형식이 올바르지 않습니다: %s" name))
    (expand-file-name (file-name-as-directory name)
                      imoogi-project-notes-directory)))

(defun imoogi-project-notes--replace-template-values (text values)
  "Replace {{KEY}} placeholders in TEXT using VALUES."
  (dolist (pair values text)
    (setq text
          (replace-regexp-in-string
           (regexp-quote (format "{{%s}}" (car pair)))
           (cdr pair)
           text t t))))

(defun imoogi-project-notes--template (name values)
  "Return rendered template NAME using VALUES."
  (let ((file (imoogi-project-notes--template-file name)))
    (unless (file-readable-p file)
      (error "Project notes template is missing: %s" file))
    (imoogi-project-notes--replace-template-values
     (with-temp-buffer
       (insert-file-contents file)
       (buffer-string))
     values)))

(defun imoogi-project-notes--central-tasks-template (values)
  "Return local tasks.org content for central TODO storage."
  (imoogi-project-notes--replace-template-values
   (concat "#+TITLE: {{PROJECT_NAME}} — 할 일 안내\n\n"
           "* 중앙 Agenda 사용\n"
           "이 프로젝트는 TODO 원본을 중앙 agenda 파일에 둡니다.\n"
           "프로젝트별 작업 상태를 별도로 복사하지 말고 아래 파일에 기록합니다.\n\n"
           "현재 프로젝트 Focus Agenda에서 분리하려면 각 TODO에 다음 속성을 둡니다.\n"
           ":CATEGORY: {{PROJECT_NAME}}\n\n"
           "- [[{{TASKS_LINK}}][중앙 할 일 파일]]\n"
           "- [[file:project.org][프로젝트 개요]]\n"
           "- [[file:journal.org][작업 기록]]\n")
   values))

(defun imoogi-project-notes--project-tasks-template (entry values)
  "Return local tasks.org content for ENTRY using VALUES."
  (if (string= (imoogi-project-notes--alist-string 'todo-storage entry) "central")
      (imoogi-project-notes--central-tasks-template values)
    (imoogi-project-notes--template "tasks" values)))

(defun imoogi-project-notes--write-new-file (file content)
  "Create FILE with CONTENT, preserving any existing file or unsaved buffer."
  (when (file-directory-p file)
    (signal 'file-error (list "Project notes file path is a directory" file)))
  (when-let* ((buffer (find-buffer-visiting file)))
    (when (buffer-modified-p buffer)
      (user-error "먼저 저장하거나 버퍼를 닫으세요: %s" file)))
  (unless (file-exists-p file)
    (make-directory (file-name-directory file) t)
    (write-region content nil file nil 'silent nil 'excl)))

(defun imoogi-project-notes--metadata-data (entry created-at)
  "Return folder metadata for ENTRY created at CREATED-AT."
  (let* ((type (imoogi-project-notes--entry-type entry))
         (root (imoogi-project-notes--alist-string 'notes-dir entry))
         (relative (lambda (key)
                     (file-relative-name
                      (imoogi-project-notes--alist-string key entry) root))))
    `((schema_version . 1)
      (type . ,(symbol-name type))
      (key . ,(imoogi-project-notes--alist-string 'key entry))
      (name . ,(imoogi-project-notes--entry-name entry))
      ,@(when (eq type 'study)
          `((study_id . ,(imoogi-project-notes--alist-string 'study-id entry))))
      ,@(when (eq type 'project)
          `((source_root . ,(imoogi-project-notes--alist-string
                             'source-root entry))))
      (created_at . ,created-at)
      (overview . ,(funcall relative 'project-file))
      (tasks . ,(funcall relative 'tasks-file))
      (journal . ,(funcall relative 'journal-file))
      (todo_storage . ,(imoogi-project-notes--alist-string
                         'todo-storage entry)))))

(defun imoogi-project-notes--ensure-metadata (entry created-at)
  "Create ENTRY's folder metadata with CREATED-AT, preserving existing data."
  (let* ((directory (imoogi-project-notes--alist-string 'notes-dir entry))
         (file (imoogi-project-notes--metadata-file directory))
         (expected (imoogi-project-notes--metadata-data entry created-at)))
    (if (file-exists-p file)
        (let ((actual (imoogi-project-notes--read-metadata-file file)))
          (unless (and (equal (alist-get 'key actual) (alist-get 'key expected))
                       (equal (alist-get 'type actual) (alist-get 'type expected)))
            (user-error "노트 폴더의 기존 메타데이터가 다른 항목을 가리킵니다: %s"
                        file)))
      (let ((json-encoding-pretty-print t))
        (imoogi-project-notes--write-new-file
         file (concat (json-encode expected) "\n"))))))

(defun imoogi-project-notes--ensure-directories (directory)
  "Create non-document directories under project notes DIRECTORY."
  (make-directory directory t)
  (make-directory (expand-file-name "assets/" directory) t)
  (make-directory (expand-file-name "references/" directory) t)
  (make-directory (expand-file-name "artifacts/" directory) t))

(defun imoogi-project-notes--safe-ensure-central-agenda ()
  "Ensure and return the central agenda file."
  (unless (fboundp 'imoogi-org-setup)
    (require 'imoogi-org))
  (imoogi-org-setup)
  (imoogi-org--default-agenda-file))

(defun imoogi-project-notes--register-agenda-file-list-only (file)
  "Register FILE in `org-agenda-files' when it is an in-memory list.
This startup path intentionally avoids writing string-backed agenda storage."
  (require 'org-agenda)
  (let ((file (expand-file-name file)))
    (when (and (file-exists-p file) (listp org-agenda-files))
      (setq org-agenda-files
            (delete-dups (append (mapcar (lambda (target)
                                           (expand-file-name target org-directory))
                                         org-agenda-files)
                                 (list file)))))))

(defun imoogi-project-notes--register-agenda-target (file)
  "Register FILE as an agenda target, preserving existing targets."
  (require 'org-agenda)
  (if (and (fboundp 'imoogi-org--agenda-storage-buffer-modified-p)
           (fboundp 'imoogi-org--current-agenda-targets)
           (fboundp 'imoogi-org--write-agenda-storage-file))
      (let ((targets (delete-dups
                      (append (imoogi-org--current-agenda-targets)
                              (list (expand-file-name file))))))
        (when (imoogi-org--agenda-storage-buffer-modified-p)
          (user-error "Save or kill the agenda file-list buffer before registering project notes"))
        (if (stringp org-agenda-files)
            (imoogi-org--write-agenda-storage-file org-agenda-files targets)
          (setq org-agenda-files targets)))
    (imoogi-project-notes--register-agenda-file-list-only file)))

(defun imoogi-project-notes--replace-agenda-target (old-file new-file)
  "Replace OLD-FILE with NEW-FILE in the configured agenda targets."
  (require 'org-agenda)
  (let* ((old-file (expand-file-name old-file))
         (new-file (expand-file-name new-file))
         (targets
          (delete-dups
           (mapcar (lambda (target)
                     (let ((expanded (expand-file-name target org-directory)))
                       (if (string= expanded old-file) new-file expanded)))
                   (if (fboundp 'imoogi-org--current-agenda-targets)
                       (imoogi-org--current-agenda-targets)
                     org-agenda-files)))))
    (when (and (fboundp 'imoogi-org--agenda-storage-buffer-modified-p)
               (imoogi-org--agenda-storage-buffer-modified-p))
      (user-error
       "Save or kill the agenda file-list buffer before moving project notes"))
    (if (stringp org-agenda-files)
        (imoogi-org--write-agenda-storage-file org-agenda-files targets)
      (setq org-agenda-files targets))))

(defun imoogi-project-notes--values (project-name source-root task-file notes-dir
                                                  &optional study-id start-date)
  "Return template values for PROJECT-NAME, SOURCE-ROOT, TASK-FILE and NOTES-DIR."
  `(("PROJECT_NAME" . ,project-name)
    ("PROJECT_ROOT" . ,source-root)
    ("TASKS_LINK" . ,(concat "file:" (file-relative-name task-file notes-dir)))
    ("STUDY_ID" . ,(or study-id ""))
    ("START_DATE" . ,(or start-date (format-time-string "%Y-%m-%d")))))

(defun imoogi-project-notes--entry (key source-root notes-dir todo-storage
                                        &optional type name study-id)
  "Build a registry entry."
  (let* ((study-p (eq type 'study))
         (project-file (expand-file-name (if study-p "study.org" "project.org")
                                         notes-dir))
         (tasks-file (if (eq todo-storage 'central)
                         (imoogi-project-notes--safe-ensure-central-agenda)
                       (expand-file-name "tasks.org" notes-dir)))
         (journal-file (expand-file-name (if study-p "logs/journal.org" "journal.org")
                                         notes-dir)))
    `((key . ,key)
      (type . ,(symbol-name (or type 'project)))
      (name . ,(or name
                   (file-name-nondirectory (directory-file-name source-root))))
      ,@(when study-id `((study-id . ,study-id)))
      (source-root . ,(imoogi-project-notes--directory-file-name source-root))
      (notes-dir . ,(imoogi-project-notes--directory-file-name notes-dir))
      (project-file . ,project-file)
      (tasks-file . ,tasks-file)
      (journal-file . ,journal-file)
      (todo-storage . ,(symbol-name todo-storage)))))

(defun imoogi-project-notes--entry-type (entry)
  "Return ENTRY's persisted type, defaulting legacy entries to project."
  (if (string= (imoogi-project-notes--alist-string 'type entry) "study")
      'study
    'project))

(defun imoogi-project-notes--entry-name (entry)
  "Return the human-readable name stored for ENTRY."
  (or (imoogi-project-notes--alist-string 'name entry)
      (file-name-nondirectory
       (directory-file-name
        (imoogi-project-notes--alist-string 'source-root entry)))))

(defun imoogi-project-notes--entry-todo-storage (entry)
  "Return ENTRY's persisted TODO storage choice."
  (let ((value (imoogi-project-notes--alist-string 'todo-storage entry)))
    (if (member value '("project" "central"))
        (intern value)
      'project)))

(defun imoogi-project-notes--setup-files (entry project-name)
  "Create missing project note files for ENTRY and PROJECT-NAME."
  (let* ((notes-dir (imoogi-project-notes--alist-string 'notes-dir entry))
         (source-root (imoogi-project-notes--alist-string 'source-root entry))
         (project-file (imoogi-project-notes--alist-string 'project-file entry))
         (journal-file (imoogi-project-notes--alist-string 'journal-file entry))
         (task-file (imoogi-project-notes--alist-string 'tasks-file entry))
         (local-tasks-file (expand-file-name "tasks.org" notes-dir))
         (values (imoogi-project-notes--values project-name source-root
                                               task-file notes-dir)))
    (imoogi-project-notes--ensure-directories notes-dir)
    (imoogi-project-notes--write-new-file
     project-file (imoogi-project-notes--template "project" values))
    (imoogi-project-notes--write-new-file
     journal-file (imoogi-project-notes--template "journal" values))
    (imoogi-project-notes--write-new-file
     local-tasks-file (imoogi-project-notes--project-tasks-template
                       entry values))))

(defun imoogi-project-notes--ensure-study-directories (directory)
  "Create the standard learning directories below DIRECTORY."
  (dolist (relative '("materials/books/" "materials/handouts/"
                      "materials/articles/" "materials/slides/"
                      "materials/videos/" "logs/" "concepts/"
                      "assignments/" "assets/"))
    (make-directory (expand-file-name relative directory) t)))

(defun imoogi-project-notes--setup-study-files (entry study-name start-date)
  "Create missing study files for ENTRY named STUDY-NAME."
  (let* ((notes-dir (imoogi-project-notes--alist-string 'notes-dir entry))
         (study-id (imoogi-project-notes--alist-string 'study-id entry))
         (tasks-file (imoogi-project-notes--alist-string 'tasks-file entry))
         (values (imoogi-project-notes--values
                  study-name notes-dir tasks-file notes-dir study-id start-date)))
    (imoogi-project-notes--ensure-study-directories notes-dir)
    (dolist (spec `((,(imoogi-project-notes--alist-string 'project-file entry) . "study")
                    (,(expand-file-name "tasks.org" notes-dir) . "study-tasks")
                    (,(expand-file-name "cards.org" notes-dir) . "cards")
                    (,(expand-file-name "questions.org" notes-dir) . "questions")
                    (,(imoogi-project-notes--alist-string 'journal-file entry) . "study-journal")))
      (imoogi-project-notes--write-new-file
       (car spec) (imoogi-project-notes--template (cdr spec) values)))))

(defun imoogi-project-notes--save-entry (entry)
  "Persist ENTRY in the registry."
  (let* ((entries (imoogi-project-notes--read-registry))
         (key (alist-get 'key entry))
         (others (cl-remove key entries
                            :key (lambda (item) (alist-get 'key item))
                            :test #'string=)))
    (imoogi-project-notes--write-registry (cons entry others))))

(defun imoogi-project-notes--set-buffer-context (entry)
  "Attach ENTRY context to the current buffer."
  (setq-local imoogi-project-notes-source-root
              (or (imoogi-project-notes--alist-string
                   'effective-source-root entry)
                  (imoogi-project-notes--alist-string 'source-root entry)))
  (setq-local imoogi-project-notes-directory-local
              (imoogi-project-notes--alist-string 'notes-dir entry))
  (setq-local imoogi-project-notes-entry-instance-id
              (imoogi-project-notes--alist-string 'instance-id entry))
  (setq-local imoogi-project-notes-active-p
              (if (assq 'active-p entry)
                  (and (alist-get 'active-p entry) t)
                t))
  (setq-local imoogi-project-notes-inactive-reason
              (alist-get 'inactive-reason entry)))

(defun imoogi-project-notes--inactive-mounted-entry-p (entry)
  "Return non-nil when ENTRY is a mounted project with no source."
  (and (eq (alist-get 'origin entry) 'mounted)
       (eq (imoogi-project-notes--entry-type entry) 'project)
       (not (alist-get 'active-p entry))))

(defun imoogi-project-notes--ensure-entry-mutable
    (entry operation &optional current-buffer-only)
  "Ensure OPERATION may mutate files for ENTRY.
An inactive mounted project blocks mutations.  A session force-edit permits
only CURRENT-BUFFER-ONLY operations in the explicitly unlocked buffer."
  (when (and (imoogi-project-notes--inactive-mounted-entry-p entry)
             (not (and current-buffer-only
                       imoogi-project-notes-force-edit-session-p)))
    (user-error
     "비활성 외장 프로젝트에서는 %s 작업을 할 수 없습니다. 먼저 소스를 재연결하세요"
     operation)))

(defun imoogi-project-notes--header-action (label face command help)
  "Return clickable header LABEL with FACE invoking COMMAND and HELP text."
  (let ((map (make-sparse-keymap)))
    (define-key map [header-line mouse-1] command)
    (propertize label
                'face face
                'mouse-face 'highlight
                'help-echo help
                'local-map map)))

(defun imoogi-project-notes--inactive-header-line ()
  "Return the header-line shown in an inactive mounted project buffer."
  (concat
   " imoogi: source missing · read-only  "
   (imoogi-project-notes--header-action
    "[r reconnect]" 'mode-line-emphasis
    #'imoogi-project-notes-reconnect-source
    "소스 폴더 재연결")
   "  "
   (imoogi-project-notes--header-action
    "[e edit this buffer]" 'warning
    #'imoogi-project-notes-force-edit-session
    "현재 버퍼만 이번 세션에 편집")
   "  "
   (imoogi-project-notes--header-action
    "[c clear override]" 'shadow
    #'imoogi-project-notes-clear-source-override
    "이 호스트의 소스 연결 해제")))

(defun imoogi-project-notes--apply-buffer-status (entry)
  "Apply active/read-only status from ENTRY to the current buffer."
  (if (imoogi-project-notes--inactive-mounted-entry-p entry)
      (progn
        (setq buffer-read-only (not imoogi-project-notes-force-edit-session-p))
        (setq-local header-line-format
                    '(:eval (imoogi-project-notes--inactive-header-line))))
    (setq-local imoogi-project-notes-force-edit-session-p nil)
    (setq buffer-read-only nil)
    (setq-local header-line-format nil)))

(defun imoogi-project-notes--find-file (entry file &optional source-root)
  "Open FILE and attach project notes ENTRY context."
  (find-file file)
  (imoogi-project-notes--set-buffer-context entry)
  (imoogi-project-notes--apply-buffer-status entry)
  (when source-root
    (setq-local imoogi-project-notes-source-root
                (imoogi-project-notes--directory-file-name source-root)))
  (current-buffer))

(defun imoogi-project-notes--find-existing-or-create (entry file template-name values)
  "Open FILE for ENTRY, creating it from TEMPLATE-NAME with VALUES if absent."
  (imoogi-project-notes--ensure-entry-mutable entry "문서 생성")
  (if-let* ((buffer (find-buffer-visiting file)))
      (progn
        (switch-to-buffer buffer)
        (imoogi-project-notes--set-buffer-context entry)
        buffer)
    (imoogi-project-notes--write-new-file
     file (imoogi-project-notes--template template-name values))
    (imoogi-project-notes--find-file
     entry file (imoogi-project-notes--current-source-root entry))))

(defun imoogi-project-notes--entry-or-setup ()
  "Return the current project notes entry, creating it when absent."
  (or (imoogi-project-notes--current-entry)
      (let* ((root (imoogi-project-notes--project-root))
             (key (imoogi-project-notes--identity-key root)))
        (imoogi-project-notes-setup root)
        (or (imoogi-project-notes--find-entry-by-key key)
            (error "Project notes setup did not create a registry entry")))))

;;;###autoload
(defun imoogi-project-notes-setup (&optional root directory numbering)
  "Create or register project notes for source ROOT.
With interactive prefix argument, choose DIRECTORY manually.  Otherwise
NUMBERING names a new folder using the project numbering convention.  Existing
files are never overwritten."
  (interactive
   (let* ((root (imoogi-project-notes--project-root))
          (directory (when current-prefix-arg
                       (read-directory-name "프로젝트 노트 폴더: "
                                            imoogi-project-notes-directory nil nil)))
          (existing (and (not directory)
                         (ignore-errors
                           (imoogi-project-notes--find-entry-by-key
                            (imoogi-project-notes--identity-key root)))))
          (numbering (unless (or directory existing)
                       (read-string "프로젝트 번호 (예: 260925.01 또는 2609): "))))
     (list root directory numbering)))
  (let* ((root (imoogi-project-notes--validate-source-root
                (or root (imoogi-project-notes--project-root))))
         (key (imoogi-project-notes--identity-key root))
         (entries (imoogi-project-notes--read-registry))
         (existing (imoogi-project-notes--find-entry-by-key key entries))
         (project-name (file-name-nondirectory (directory-file-name root)))
         (notes-dir (imoogi-project-notes--validate-notes-directory
                     (or directory
                         (imoogi-project-notes--alist-string 'notes-dir existing)
                         (and numbering
                              (imoogi-project-notes--numbered-notes-directory
                               numbering root))
                         (imoogi-project-notes--default-notes-directory root key entries))
                     key entries))
         (storage (if existing
                      (imoogi-project-notes--entry-todo-storage existing)
                    (if (member imoogi-project-notes-todo-storage '(project central))
                        imoogi-project-notes-todo-storage
                      'project)))
         (entry (imoogi-project-notes--entry key root notes-dir storage 'project)))
    (imoogi-project-notes--setup-files entry project-name)
    (imoogi-project-notes--ensure-metadata entry (format-time-string "%Y-%m-%d"))
    (imoogi-project-notes--save-entry entry)
    (imoogi-project-notes--register-agenda-target
     (imoogi-project-notes--alist-string 'tasks-file entry))
    (message "imoogi: 작업 폴더 %s → 문서 폴더 %s" root notes-dir)
    notes-dir))

(defun imoogi-project-notes--replace-treemacs-root (old-root new-root)
  "Replace OLD-ROOT with NEW-ROOT in existing Treemacs workspaces.
Never remove a workspace; only replace the project path in workspaces that
already contain OLD-ROOT."
  (when (and (fboundp 'treemacs-workspaces)
             (fboundp 'treemacs-workspace->projects)
             (fboundp 'treemacs-project->path)
             (fboundp 'treemacs-do-remove-project-from-workspace)
             (fboundp 'treemacs-do-add-project-to-workspace))
    (let ((old-path (directory-file-name (expand-file-name old-root))))
      (dolist (workspace (treemacs-workspaces))
        (dolist (project (cl-remove-if-not
                          (lambda (candidate)
                            (equal old-path
                                   (directory-file-name
                                    (expand-file-name
                                     (treemacs-project->path candidate)))))
                          (treemacs-workspace->projects workspace)))
          ;; Treemacs mutates live buffers and can run user hooks.  A stale
          ;; workspace must never prevent the doctor from finishing.
          (condition-case err
              (with-timeout
                  (1 (user-error "Treemacs 갱신 시간이 초과되었습니다"))
                (let ((treemacs-override-workspace workspace))
                  (treemacs-do-remove-project-from-workspace
                   project :ignore-last-project-restriction)
                  (treemacs-do-add-project-to-workspace
                   new-root
                   (file-name-nondirectory (directory-file-name new-root)))))
            (error
             (display-warning
              'imoogi
              (format "Treemacs 프로젝트 경로를 갱신하지 못했습니다: %s"
                      (error-message-string err))
              :warning))))))))

(defun imoogi-project-notes--rename-entry-directory (entry old-root new-root)
  "Rename ENTRY's notes folder and update local references.
The directory is renamed as one unit; no file inside it is deleted."
  (let ((old-root (file-name-as-directory (expand-file-name old-root)))
        (new-root (file-name-as-directory (expand-file-name new-root)))
        (old-tasks (and entry (imoogi-project-notes--alist-string
                               'tasks-file entry))))
    (when (file-exists-p new-root)
      (user-error "새 프로젝트 폴더가 이미 존재합니다: %s" new-root))
    (dolist (buffer (buffer-list))
      (when-let* ((file (buffer-file-name buffer)))
        (when (and (file-in-directory-p file old-root)
                   (buffer-modified-p buffer))
          (user-error "먼저 저장하거나 닫아야 하는 변경 버퍼가 있습니다: %s" file))))
    (rename-file old-root new-root)
    (dolist (buffer (buffer-list))
      (when-let* ((file (buffer-file-name buffer)))
        (when (file-in-directory-p file old-root)
          (set-visited-file-name
           (expand-file-name (file-relative-name file old-root) new-root)
           t t))))
    (when entry
      (dolist (key '(project-file tasks-file journal-file))
        (let ((file (imoogi-project-notes--alist-string key entry)))
          (when (file-in-directory-p file old-root)
            (setf (alist-get key entry)
                  (expand-file-name (file-relative-name file old-root) new-root)))))
      (setf (alist-get 'notes-dir entry) new-root)
      (when (and old-tasks (file-in-directory-p old-tasks old-root))
        (imoogi-project-notes--replace-agenda-target
         old-tasks (alist-get 'tasks-file entry)))
      (when (boundp 'imoogi-project-perspective-alist)
        (let ((mapping (assoc old-root imoogi-project-perspective-alist)))
          (when mapping
            (setcar mapping new-root)))))
    (imoogi-project-notes--replace-treemacs-root old-root new-root)
    entry))

(defun imoogi-project-notes--doctor-base-directory ()
  "Return the project-notes root used by the doctor.
If a buffer-local or stale customization points at one project directory,
step back to its `project-notes' parent so renames cannot become nested."
  (let* ((configured (file-name-as-directory
                      (expand-file-name imoogi-project-notes-directory)))
         (name (file-name-nondirectory (directory-file-name configured)))
         (parent (file-name-as-directory
                  (file-name-directory (directory-file-name configured)))))
    (if (and (imoogi-project-notes--folder-name-valid-p name)
             (string= (file-name-nondirectory
                       (directory-file-name parent))
                      "project-notes"))
        parent
      configured)))

;;;###autoload
(defun imoogi-project-notes-setup-doctor ()
  "Inspect project-notes folders and ask how to rename each invalid one.
Each answer is handled independently.  Empty input skips that folder.  The
folder is renamed as a whole and its registry, buffers, Perspective mapping,
and Treemacs roots are updated; files are never deleted or overwritten."
  (interactive)
  (let ((base (imoogi-project-notes--doctor-base-directory))
        (changed 0)
        (skipped 0))
    (unless (file-directory-p base)
      (user-error "프로젝트 노트 폴더가 없습니다: %s" base))
    (dolist (directory (directory-files base t "^[^.].*" t))
      (when (file-directory-p directory)
        (let ((name (file-name-nondirectory (directory-file-name directory))))
          (unless (imoogi-project-notes--folder-name-valid-p name)
            (let* ((suggestion (imoogi-project-notes--doctor-suggested-name name))
                   (answer (read-string
                            (format "새 번호/폴더명 [%s] (빈칸은 건너뜀): " name)
                            suggestion)))
              (if (string-empty-p (string-trim answer))
                  (setq skipped (1+ skipped))
                (if (not (imoogi-project-notes--folder-name-valid-p answer))
                    (user-error "프로젝트 폴더명 형식이 올바르지 않습니다: %s" answer)
                  (let* ((old-root (file-name-as-directory directory))
                         (new-root (expand-file-name (file-name-as-directory answer) base))
                         (answer-id (imoogi-project-notes--folder-id-key answer))
                         (duplicate-id
                          (when answer-id
                            (cl-find-if
                             (lambda (candidate)
                               (and (not (equal candidate name))
                                    (equal answer-id
                                           (imoogi-project-notes--folder-id-key candidate))))
                             (mapcar (lambda (path)
                                       (file-name-nondirectory
                                        (directory-file-name path)))
                                     (directory-files base t "^[^.].*" t)))))
                         (entry (or (imoogi-project-notes--find-entry-by-notes-directory old-root)
                                    (let ((metadata-file
                                           (imoogi-project-notes--metadata-file old-root)))
                                      (when (file-readable-p metadata-file)
                                        (imoogi-project-notes--metadata-entry
                                         metadata-file
                                         (imoogi-project-notes--read-metadata-file metadata-file)))))))
                    (if duplicate-id
                        (user-error "ID %s가 이미 사용 중입니다: %s"
                                    answer-id duplicate-id)
                      (imoogi-project-notes--rename-entry-directory
                       entry old-root new-root)
                      (when entry
                        (imoogi-project-notes--save-entry entry))
                      (setq changed (1+ changed)))))))))
    (message "imoogi: project-notes 점검 완료 — 변경 %d개, 건너뜀 %d개 (파일 삭제 없음)"
             changed skipped)
    (list :changed changed :skipped skipped)))))

(defun imoogi-project-notes--open-study-workspace (entry)
  "Open the Perspective and Treemacs workspace represented by study ENTRY."
  (let* ((root (imoogi-project-notes--alist-string 'notes-dir entry))
         (name (if (fboundp 'imoogi-project-perspective-name)
                   (imoogi-project-perspective-name root)
                 (file-name-nondirectory (directory-file-name root)))))
    (when (fboundp 'persp-switch)
      (persp-switch name))
    (imoogi-project-notes--find-file
     entry (imoogi-project-notes--alist-string 'project-file entry) root)
    (when (fboundp 'imoogi-treemacs-open-project-workspace)
      (imoogi-treemacs-open-project-workspace root name))))

;;;###autoload
(defun imoogi-project-notes-setup-study (study-name &optional directory study-id start-date)
  "Create a source-free study note workspace named STUDY-NAME.
The default folder is `YY.NN-STUDY-NAME' below
`imoogi-project-notes-directory'.  DIRECTORY, STUDY-ID and START-DATE are
optional programmatic overrides.  Existing files are never overwritten."
  (interactive
   (let* ((name (read-string "학습 이름: "))
          (mounted-root
           (when current-prefix-arg
             (let ((entry (imoogi-project-notes--select-mounted-root
                           "학습 노트를 만들 외장 루트: ")))
               (when (eq (alist-get 'enabled entry) :json-false)
                 (user-error "비활성 외장 루트에는 학습 노트를 만들 수 없습니다"))
               (let ((path (alist-get 'path entry)))
                 (unless (file-directory-p path)
                   (user-error "외장 노트 루트가 연결되어 있지 않습니다: %s" path))
                 path))))
          (base (or mounted-root imoogi-project-notes-directory))
          (id (imoogi-project-notes--next-study-id nil mounted-root))
          (default (expand-file-name
                    (format "%s-%s/" id (imoogi-project-notes--slug name))
                    base))
          (directory (when mounted-root
                       (read-directory-name "외장 학습 노트 폴더: "
                                            default nil nil))))
     (list name directory id (format-time-string "%Y-%m-%d"))))
  (when (string-empty-p (string-trim study-name))
    (user-error "학습 이름을 입력하세요"))
  (let* ((study-id (or study-id (imoogi-project-notes--next-study-id)))
         (start-date (or start-date (format-time-string "%Y-%m-%d")))
         (key (concat "study:" study-id))
         (entries (imoogi-project-notes--read-registry))
         (existing (imoogi-project-notes--find-entry-by-key key entries))
         (notes-dir
          (imoogi-project-notes--validate-notes-directory
           (or directory
               (imoogi-project-notes--alist-string 'notes-dir existing)
               (expand-file-name
                (format "%s-%s/" study-id
                        (imoogi-project-notes--slug study-name))
                imoogi-project-notes-directory))
           key entries))
         (entry (imoogi-project-notes--entry
                 key notes-dir notes-dir 'project 'study study-name study-id)))
    (imoogi-project-notes--setup-study-files entry study-name start-date)
    (imoogi-project-notes--ensure-metadata entry start-date)
    (imoogi-project-notes--save-entry entry)
    (imoogi-project-notes--register-agenda-target
     (imoogi-project-notes--alist-string 'tasks-file entry))
    (imoogi-project-notes--open-study-workspace entry)
    (message "imoogi: 학습 %s (%s) → %s" study-name study-id notes-dir)
    notes-dir))

;;;###autoload
(defun imoogi-project-notes-open ()
  "Open the current project's overview note."
  (interactive)
  (let* ((entry (imoogi-project-notes--entry-or-setup))
         (source-root (imoogi-project-notes--current-source-root entry)))
    (imoogi-project-notes--find-file
     entry (imoogi-project-notes--alist-string 'project-file entry) source-root)))

;;;###autoload
(defun imoogi-project-notes-tasks ()
  "Open the current project's TODO source."
  (interactive)
  (let* ((entry (imoogi-project-notes--entry-or-setup))
         (source-root (imoogi-project-notes--current-source-root entry)))
    (imoogi-project-notes--find-file
     entry (imoogi-project-notes--alist-string 'tasks-file entry) source-root)))

(defun imoogi-project-notes--worktree-heading (root)
  "Return journal heading text for source worktree ROOT."
  (format "** 작업 공간: %s" (abbreviate-file-name
                             (imoogi-project-notes--directory-file-name root))))

(defun imoogi-project-notes--ensure-worktree-journal-section (entry source-root)
  "Ensure the current worktree has a journal section in ENTRY."
  (let* ((root (or source-root
                   imoogi-project-notes-source-root
                   (imoogi-project-notes--alist-string 'source-root entry)))
         (heading (imoogi-project-notes--worktree-heading root)))
    (goto-char (point-min))
    (unless (re-search-forward
             (concat "^" (regexp-quote heading) "$") nil t)
      (if (re-search-forward "^\\* 진행 기록$" nil t)
          (progn
            (forward-line 1)
            (when (re-search-forward "^\\* " nil t)
              (beginning-of-line)))
        (goto-char (point-max)))
      (unless (bolp) (insert "\n"))
      (insert "\n" heading "\n"
              "마지막 진행 지점:\n"
              "다음 행동:\n"
              "관련 TODO:\n"))
    (beginning-of-line)))

;;;###autoload
(defun imoogi-project-notes-journal ()
  "Open the current project's journal and show this worktree's section."
  (interactive)
  (let* ((entry (imoogi-project-notes--entry-or-setup))
         (source-root (imoogi-project-notes--current-source-root entry)))
    (when (eq (imoogi-project-notes--entry-type entry) 'project)
      (imoogi-project-notes--ensure-entry-mutable entry "작업 기록 변경"))
    (imoogi-project-notes--find-file
     entry (imoogi-project-notes--alist-string 'journal-file entry) source-root)
    (unless (eq (imoogi-project-notes--entry-type entry) 'study)
      (imoogi-project-notes--ensure-worktree-journal-section entry source-root))
    (current-buffer)))

;;;###autoload
(defun imoogi-project-notes-add-document (document)
  "Create and open an on-demand development DOCUMENT."
  (interactive
   (list
    (intern
     (completing-read "개발 문서: "
                      (mapcar (lambda (entry)
                                (symbol-name (car entry)))
                              imoogi-project-notes--documents)
                      nil t))))
  (let* ((spec (alist-get document imoogi-project-notes--documents))
         (entry (imoogi-project-notes--entry-or-setup)))
    (imoogi-project-notes--ensure-entry-mutable entry "개발 문서 생성")
    (unless spec
      (user-error "알 수 없는 프로젝트 문서: %s" document))
    (let* ((notes-dir (imoogi-project-notes--alist-string 'notes-dir entry))
           (source-root (imoogi-project-notes--alist-string 'source-root entry))
           (project-name (imoogi-project-notes--entry-name entry))
           (task-file (imoogi-project-notes--alist-string 'tasks-file entry))
           (file (expand-file-name (car spec)
                                   (expand-file-name "development/" notes-dir)))
           (values (imoogi-project-notes--values project-name source-root
                                                 task-file notes-dir)))
      (imoogi-project-notes--find-existing-or-create
       entry file (symbol-name document) values))))

(defun imoogi-project-notes--entry-label (entry)
  "Return a readable completion label for project notes ENTRY."
  (let* ((root (or (imoogi-project-notes--alist-string
                    'effective-source-root entry)
                   (imoogi-project-notes--alist-string 'source-root entry)))
         (type (imoogi-project-notes--entry-type entry))
         (name (imoogi-project-notes--entry-name entry))
         (mounted (eq (alist-get 'origin entry) 'mounted))
         (inactive (imoogi-project-notes--inactive-mounted-entry-p entry)))
    (format "[%s%s%s] %-20s %s"
            (if (eq type 'study) "학습" "작업")
            (if mounted
                (format "/%s" (or (alist-get 'device-label entry) "외장")) "")
            (if inactive "/비활성" "")
            name (abbreviate-file-name root))))

(defun imoogi-project-notes--select-entry (&optional prompt)
  "Prompt for and return a registered project-notes entry.
Project entries are identified by their source work directory.  Study entries
use their standalone notes directory as the workspace root."
  (let* ((entries (imoogi-project-notes--all-entries))
         (candidates (mapcar (lambda (entry)
                               (cons (imoogi-project-notes--entry-label entry) entry))
                             entries)))
    (unless candidates
      (user-error "등록된 프로젝트 노트가 없습니다"))
    (cdr (assoc (completing-read (or prompt "프로젝트 또는 학습 노트: ")
                                 candidates nil t)
                candidates))))

(defun imoogi-project-notes--replace-source-override
    (state instance-id source-root)
  "Return STATE with INSTANCE-ID mapped to SOURCE-ROOT."
  (let* ((overrides (alist-get 'source-overrides state))
         (others (cl-remove instance-id overrides
                            :key (lambda (item) (alist-get 'instance-id item))
                            :test #'string=))
         (replacement (and source-root
                           `((instance-id . ,instance-id)
                             (source-root . ,source-root)
                             (updated-at . ,(format-time-string "%Y-%m-%d"))))))
    (setf (alist-get 'source-overrides state)
          (if replacement (cons replacement others) others))
    state))

(defun imoogi-project-notes--refresh-current-buffer-entry ()
  "Refresh mounted project-note state in the current buffer."
  (when-let* ((instance-id imoogi-project-notes-entry-instance-id)
              (entry (cl-find instance-id (imoogi-project-notes--all-entries)
                              :key (lambda (item)
                                     (alist-get 'instance-id item))
                              :test #'string=)))
    (imoogi-project-notes--set-buffer-context entry)
    (imoogi-project-notes--apply-buffer-status entry)
    entry))

;;;###autoload
(defun imoogi-project-notes-reconnect-source (&optional entry directory)
  "Reconnect mounted project-note ENTRY to local source DIRECTORY."
  (interactive)
  (let* ((entry (or entry (imoogi-project-notes--current-entry)
                    (imoogi-project-notes--select-entry "재연결할 프로젝트: ")))
         (instance-id (imoogi-project-notes--alist-string 'instance-id entry)))
    (unless (and (eq (alist-get 'origin entry) 'mounted)
                 (eq (imoogi-project-notes--entry-type entry) 'project)
                 instance-id)
      (user-error "외장 project note만 소스 경로를 재연결할 수 있습니다"))
    (let* ((source-root
            (imoogi-project-notes--validate-source-root
             (or directory (read-directory-name "새 소스 프로젝트 폴더: "))))
           (state (imoogi-project-notes--read-mounted-state)))
      (imoogi-project-notes--write-mounted-state
       (imoogi-project-notes--replace-source-override
        state instance-id source-root))
      (imoogi-project-notes--refresh-current-buffer-entry)
      (message "imoogi: 외장 프로젝트 소스를 재연결했습니다: %s" source-root)
      source-root)))

;;;###autoload
(defun imoogi-project-notes-clear-source-override (&optional entry)
  "Clear the host-local source override for mounted project-note ENTRY."
  (interactive)
  (let* ((entry (or entry (imoogi-project-notes--current-entry)
                    (imoogi-project-notes--select-entry "연결을 지울 프로젝트: ")))
         (instance-id (imoogi-project-notes--alist-string 'instance-id entry)))
    (unless (and (eq (alist-get 'origin entry) 'mounted) instance-id)
      (user-error "외장 project note의 연결만 지울 수 있습니다"))
    (imoogi-project-notes--write-mounted-state
     (imoogi-project-notes--replace-source-override
      (imoogi-project-notes--read-mounted-state) instance-id nil))
    (imoogi-project-notes--refresh-current-buffer-entry)
    (message "imoogi: 이 호스트의 소스 연결을 지웠습니다")))

;;;###autoload
(defun imoogi-project-notes-force-edit-session ()
  "Unlock only the current inactive note buffer for this Emacs session."
  (interactive)
  (let ((entry (imoogi-project-notes--current-entry)))
    (unless (and entry (imoogi-project-notes--inactive-mounted-entry-p entry))
      (user-error "현재 버퍼는 비활성 외장 project note가 아닙니다"))
    (setq-local imoogi-project-notes-force-edit-session-p t)
    (setq buffer-read-only nil)
    (setq-local header-line-format
                '(:eval (concat (imoogi-project-notes--inactive-header-line)
                                "  · session edit")))
    (message "imoogi: 현재 버퍼만 이번 세션 동안 편집할 수 있습니다")))

(defun imoogi-project-notes--root-label (root)
  "Return completion label for managed ROOT."
  (format "%-16s %s%s"
          (imoogi-project-notes--alist-string 'label root)
          (abbreviate-file-name
           (imoogi-project-notes--alist-string 'path root))
          (if (eq (alist-get 'enabled root) :json-false) " [disabled]" "")))

(defun imoogi-project-notes--select-mounted-root (&optional prompt)
  "Prompt for a registered mounted root and return it."
  (let* ((roots (alist-get 'roots (imoogi-project-notes--read-mounted-state)))
         (candidates
          (mapcar (lambda (root)
                    (cons (imoogi-project-notes--root-label root) root))
                  roots)))
    (unless candidates
      (user-error "등록된 외장 노트 루트가 없습니다"))
    (cdr (assoc (completing-read (or prompt "외장 노트 루트: ")
                                 candidates nil t)
                candidates))))

;;;###autoload
(defun imoogi-project-notes-mounted-root-add (directory label)
  "Register local DIRECTORY as a managed mounted root named LABEL."
  (interactive
   (let* ((directory (read-directory-name "외장 노트 루트: " nil nil nil))
          (label (read-string "장치/루트 표시 이름: "
                              (file-name-nondirectory
                               (directory-file-name directory)))))
     (list directory label)))
  (when (string-empty-p (string-trim label))
    (user-error "표시 이름을 입력하세요"))
  (unless (file-exists-p directory)
    (if (called-interactively-p 'interactive)
        (when (y-or-n-p (format "폴더를 생성할까요? %s " directory))
          (make-directory directory t))
      (make-directory directory t)))
  (let* ((state (imoogi-project-notes--read-mounted-state))
         (canonical (imoogi-project-notes--validate-mounted-root
                     directory (alist-get 'roots state)))
         (entry `((id . ,(imoogi-project-notes--mounted-root-id canonical))
                  (path . ,canonical)
                  (label . ,label)
                  (enabled . t)
                  (created-at . ,(format-time-string "%Y-%m-%d")))))
    (push entry (alist-get 'roots state))
    (imoogi-project-notes--write-mounted-state state)
    (message "imoogi: 외장 노트 루트를 등록했습니다: %s" canonical)
    entry))

;;;###autoload
(defun imoogi-project-notes-mounted-root-list ()
  "Show registered mounted roots and their currently discovered notes."
  (interactive)
  (let ((state (imoogi-project-notes--read-mounted-state))
        (entries (imoogi-project-notes--all-entries)))
    (with-help-window "*imoogi 외장 노트 루트*"
      (princ "외장 노트 루트\n\n")
      (dolist (root (alist-get 'roots state))
        (princ (format "%s\n" (imoogi-project-notes--root-label root)))
        (dolist (entry entries)
          (when (equal (alist-get 'mounted-root-id entry)
                       (alist-get 'id root))
            (princ (format "  - %s\n"
                           (imoogi-project-notes--entry-label entry)))))))))

;;;###autoload
(defun imoogi-project-notes-mounted-root-edit (&optional root label &rest enabled-args)
  "Edit managed ROOT's LABEL and ENABLED state."
  (interactive)
  (let* ((root (or root (imoogi-project-notes--select-mounted-root)))
         (label (or label
                    (read-string "새 표시 이름: "
                                 (imoogi-project-notes--alist-string
                                  'label root))))
         (enabled (if (called-interactively-p 'interactive)
                      (y-or-n-p "이 루트를 활성화할까요? ")
                    (car enabled-args)))
         (state (imoogi-project-notes--read-mounted-state))
         (id (alist-get 'id root))
         (saved (cl-find id (alist-get 'roots state)
                         :key (lambda (item) (alist-get 'id item))
                         :test #'string=)))
    (unless saved (user-error "등록된 외장 노트 루트가 아닙니다"))
    (setf (alist-get 'label saved) label
          (alist-get 'enabled saved)
          (if (or enabled-args (called-interactively-p 'interactive))
              (if enabled t :json-false)
            (alist-get 'enabled saved)))
    (imoogi-project-notes--write-mounted-state state)
    saved))

;;;###autoload
(defun imoogi-project-notes-mounted-root-remove (&optional root)
  "Remove managed ROOT registration without deleting external files."
  (interactive)
  (let* ((root (or root (imoogi-project-notes--select-mounted-root)))
         (state (imoogi-project-notes--read-mounted-state))
         (id (alist-get 'id root)))
    (setf (alist-get 'roots state)
          (cl-remove id (alist-get 'roots state)
                     :key (lambda (item) (alist-get 'id item))
                     :test #'string=))
    (imoogi-project-notes--write-mounted-state state)
    (message "imoogi: 외장 루트 등록만 제거했습니다. 파일은 변경하지 않았습니다")))

;;;###autoload
(defun imoogi-project-notes-mounted-root-refresh ()
  "Rescan registered mounted roots and report discovered entries."
  (interactive)
  (let* ((state (imoogi-project-notes--read-mounted-state 'noerror))
         (roots (cl-count-if
                 (lambda (root) (not (eq (alist-get 'enabled root) :json-false)))
                 (alist-get 'roots state)))
         (entries (cl-count-if
                   (lambda (entry) (eq (alist-get 'origin entry) 'mounted))
                   (imoogi-project-notes--all-entries))))
    (message "imoogi: 외장 루트 %d개에서 노트 %d개를 찾았습니다" roots entries)
    entries))

(defun imoogi-project-notes--move-command ()
  "Return the executable command for project-note filesystem operations."
  (when-let* ((program (car imoogi-project-notes-command))
              (executable
               (or (executable-find program)
                   (let ((bundled (expand-file-name "bin/imoogi-notes"
                                                    imoogi-emacs-dir)))
                     (and (file-executable-p bundled) bundled)))))
    (cons executable (cdr imoogi-project-notes-command))))

(defun imoogi-project-notes--run-move-command (source destination)
  "Ask the Go helper to move SOURCE to DESTINATION and return its response."
  (let ((command (imoogi-project-notes--move-command)))
    (unless command
      (user-error
       "imoogi-notes 실행 파일이 없습니다. 저장소에서 make build-notes를 실행하세요"))
    (let ((stderr-file (make-temp-file "imoogi-notes-stderr-"))
          response status)
      (unwind-protect
          (with-temp-buffer
            (insert (json-encode `((operation . "move")
                                   (source . ,(expand-file-name source))
                                   (destination . ,(expand-file-name destination)))))
            (setq status
                  (apply #'call-process-region
                         (point-min) (point-max) (car command)
                         t (list t stderr-file) nil (cdr command)))
            (goto-char (point-min))
            (condition-case err
                (let ((json-object-type 'alist)
                      (json-array-type 'list)
                      (json-key-type 'symbol)
                      (json-false :json-false))
                  (setq response (json-read)))
              (error
               (user-error "imoogi-notes 응답을 읽을 수 없습니다: %s"
                           (error-message-string err)))))
            (unless (and (integerp status) (zerop status))
              (user-error "imoogi-notes 실행 실패: %s"
                          (string-trim
                           (with-temp-buffer
                             (insert-file-contents stderr-file)
                             (buffer-string)))))
            response)
        (when (file-exists-p stderr-file)
          (delete-file stderr-file)))))

(defun imoogi-project-notes--retarget-buffers
    (buffers old-directory new-directory entry)
  "Retarget BUFFERS from OLD-DIRECTORY to NEW-DIRECTORY and attach ENTRY."
  (let ((old-directory (file-name-as-directory (expand-file-name old-directory))))
    (dolist (buffer buffers)
      (when-let* ((file (buffer-file-name buffer)))
        (let ((target (expand-file-name
                       (file-relative-name file old-directory)
                       new-directory)))
          (with-current-buffer buffer
            (setq default-directory (file-name-directory target))
            ;; SOURCE no longer exists after the CLI succeeds, so update the
            ;; visited identity directly instead of asking Emacs to inspect
            ;; or rename the old path again.
            (setq buffer-file-name target)
            (set-visited-file-modtime)
            (rename-buffer (file-name-nondirectory target) t)
            (imoogi-project-notes--set-buffer-context entry)
            (imoogi-project-notes--apply-buffer-status entry)))))))

(defun imoogi-project-notes--remove-local-entry (entry)
  "Remove local registry ENTRY after it has moved to a mounted root."
  (let ((key (alist-get 'key entry)))
    (imoogi-project-notes--write-registry
     (cl-remove key (imoogi-project-notes--read-registry)
                :key (lambda (item) (alist-get 'key item))
                :test #'string=))))

;;;###autoload
(defun imoogi-project-notes-move-to-mounted-root (&optional entry root)
  "Move local project-note ENTRY below registered mounted ROOT.
The destination keeps the current note directory name.  Existing destinations
are never overwritten.  Open modified note buffers must be saved before the
copy starts.  The copied tree is hash-verified before local registration and
Agenda paths are changed and the original directory is removed."
  (interactive)
  (let* ((entry (or entry
                    (imoogi-project-notes--current-entry)
                    (imoogi-project-notes--select-entry
                     "외장으로 옮길 로컬 노트: ")))
         (root (or root
                   (imoogi-project-notes--select-mounted-root
                    "옮길 외장 루트: ")))
         (source (imoogi-project-notes--alist-string 'notes-dir entry))
         (root-path (imoogi-project-notes--alist-string 'path root)))
    (unless (eq (alist-get 'origin entry) 'local)
      (user-error "로컬에 등록된 project/study note만 외장으로 옮길 수 있습니다"))
    (when (eq (imoogi-project-notes--entry-todo-storage entry) 'central)
      (user-error
       "중앙 agenda를 사용하는 프로젝트는 외장으로 옮길 수 없습니다. project tasks.org 저장 방식으로 전환하세요"))
    (when (eq (alist-get 'enabled root) :json-false)
      (user-error "비활성 외장 루트로는 노트를 옮길 수 없습니다"))
    (unless (file-directory-p root-path)
      (user-error "외장 노트 루트가 연결되어 있지 않습니다: %s" root-path))
    (unless (file-directory-p source)
      (user-error "옮길 노트 폴더가 없습니다: %s" source))
    (let* ((destination
            (expand-file-name
             (file-name-nondirectory (directory-file-name source)) root-path))
           (tasks-relative
            (file-relative-name
             (imoogi-project-notes--alist-string 'tasks-file entry) source))
           (old-tasks (imoogi-project-notes--alist-string 'tasks-file entry))
           (new-tasks (expand-file-name tasks-relative destination))
           (buffers (imoogi-project-notes--buffers-under-directory source))
           (dirty (cl-remove-if-not #'buffer-modified-p buffers))
           moved-entry)
      (when (file-exists-p destination)
        (user-error "외장 루트에 같은 이름의 폴더가 이미 있습니다: %s"
                    destination))
      (when dirty
        (imoogi-project-notes--show-dirty-buffers dirty)
        (unless (y-or-n-p
                 (format "변경 파일 %d개를 저장하고 외장 루트로 옮길까요? "
                         (length dirty)))
          (user-error "외장 노트 이동을 취소했습니다")))
      (imoogi-project-notes--save-buffers-or-error dirty)
      (when (and (fboundp 'imoogi-org--agenda-storage-buffer-modified-p)
                 (imoogi-org--agenda-storage-buffer-modified-p))
        (user-error
         "Save or kill the agenda file-list buffer before moving project notes"))
      (let ((selected-note-buffer-p (memq (current-buffer) buffers))
            response)
        ;; Do not leave Emacs' process cwd inside SOURCE while the helper
        ;; atomically removes that directory.
        (when selected-note-buffer-p
          (setq default-directory root-path))
        (condition-case err
            (setq response
                  (funcall imoogi-project-notes-move-runner-function
                           source destination))
          (error
           (when (and selected-note-buffer-p (file-directory-p source))
             (setq default-directory (file-name-as-directory source)))
           (signal (car err) (cdr err))))
        (unless (eq (alist-get 'ok response) t)
          (when (and selected-note-buffer-p (file-directory-p source))
            (setq default-directory (file-name-as-directory source)))
          (user-error "외장 노트 이동 실패 [%s]: %s"
                      (or (alist-get 'code response) "unknown")
                      (or (alist-get 'error response) "원인을 확인할 수 없습니다")))
        ;; The CLI has atomically removed SOURCE.  Keep the selected note
        ;; buffer usable while Emacs applies the returned destination paths.
        (when selected-note-buffer-p
          (setq default-directory (file-name-as-directory destination)))
        (unless (and (file-directory-p destination)
                     (not (file-exists-p source)))
          (error "imoogi-notes 성공 응답과 실제 파일 상태가 일치하지 않습니다"))
        (setq moved-entry
              (imoogi-project-notes--metadata-entry
               (imoogi-project-notes--metadata-file destination)
               (imoogi-project-notes--read-metadata-file
                (imoogi-project-notes--metadata-file destination))))
        (push '(origin . mounted) moved-entry)
        (push (cons 'logical-key (alist-get 'key moved-entry)) moved-entry)
        (push (cons 'mounted-root-id
                    (imoogi-project-notes--alist-string 'id root))
              moved-entry)
        (push (cons 'mounted-root root-path) moved-entry)
        (push (cons 'device-label
                    (imoogi-project-notes--alist-string 'label root))
              moved-entry)
        (push (cons 'instance-id
                    (imoogi-project-notes--mounted-instance-id
                     (imoogi-project-notes--alist-string 'id root)
                     destination))
              moved-entry)
        (setq moved-entry
              (imoogi-project-notes--decorate-mounted-status
               moved-entry (imoogi-project-notes--read-mounted-state)))
        (imoogi-project-notes--replace-agenda-target old-tasks new-tasks)
        (imoogi-project-notes--remove-local-entry entry)
        (imoogi-project-notes--retarget-buffers
         buffers source destination moved-entry)
        (when-let* ((warning (imoogi-project-notes--alist-string
                             'warning response)))
          (display-warning 'imoogi warning :warning))
        (message "imoogi: 노트를 외장 루트로 옮겼습니다 (%s개 파일): %s"
                 (or (alist-get 'files response) 0) destination)
        (file-name-as-directory destination)))))

;;;###autoload
(defun imoogi-project-notes-detach (&optional entry)
  "Hide mounted note ENTRY for this Emacs session without unmounting."
  (interactive)
  (let* ((entry (or entry (imoogi-project-notes--select-entry "분리할 외장 노트: ")))
         (instance-id (imoogi-project-notes--alist-string 'instance-id entry)))
    (unless (and (eq (alist-get 'origin entry) 'mounted) instance-id)
      (user-error "외장 노트만 세션에서 분리할 수 있습니다"))
    (cl-pushnew instance-id imoogi-project-notes--detached-instance-ids
                :test #'string=)
    (message "imoogi: 외장 노트를 현재 세션 목록에서 분리했습니다")))

(defun imoogi-project-notes--command-output (&rest command)
  "Return trimmed output when COMMAND exits successfully."
  (with-temp-buffer
    (when (zerop (apply #'process-file (car command) nil t nil (cdr command)))
      (string-trim (buffer-string)))))

(defun imoogi-project-notes--df-descriptor (root)
  "Resolve ROOT through POSIX df and return device/mount data."
  (when-let* ((df (executable-find "df"))
              (output (imoogi-project-notes--command-output df "-P" root)))
    (let* ((lines (split-string output "\n" t))
           (fields (split-string (car (last lines)) "[[:space:]]+" t)))
      (when (>= (length fields) 6)
        (list :device (car fields)
              :mount-point (mapconcat #'identity (nthcdr 5 fields) " "))))))

(defun imoogi-project-notes--default-unmount-preflight (root)
  "Return an unmount descriptor for managed ROOT or signal `user-error'."
  (let* ((path (imoogi-project-notes--alist-string 'path root))
         (resolved (imoogi-project-notes--df-descriptor path))
         (device (plist-get resolved :device))
         (mount-point (plist-get resolved :mount-point)))
    (unless (and device mount-point
                 (let ((canonical-path
                        (imoogi-project-notes--directory-file-name
                         (file-truename path)))
                       (canonical-mount
                        (imoogi-project-notes--directory-file-name
                         (file-truename mount-point))))
                   (or (string= canonical-path canonical-mount)
                       (file-in-directory-p canonical-path canonical-mount))))
      (user-error "외장 루트의 실제 mount point를 확인할 수 없습니다"))
    (when (string= (file-name-as-directory mount-point) "/")
      (user-error "시스템 루트 볼륨은 imoogi에서 unmount할 수 없습니다"))
    (cond
     ((eq system-type 'darwin)
      (unless (executable-find "diskutil")
        (user-error "diskutil을 찾을 수 없어 물리 unmount를 지원하지 않습니다"))
      (list :supported t :platform 'darwin :device device
            :mount-point mount-point
            :command (list "diskutil" "unmount" mount-point)))
     ((eq system-type 'gnu/linux)
      (unless (and (string-prefix-p "/dev/" device)
                   (executable-find "udisksctl"))
        (user-error "이 Linux mount는 안전한 udisksctl unmount를 지원하지 않습니다"))
      (list :supported t :platform 'gnu/linux :device device
            :mount-point mount-point
            :command (list "udisksctl" "unmount" "-b" device)))
     (t
      (user-error "이 운영체제에서는 물리 unmount를 지원하지 않습니다")))))

(defun imoogi-project-notes--default-unmount-executor (descriptor)
  "Execute validated unmount DESCRIPTOR and return non-nil on success."
  (let ((command (plist-get descriptor :command)))
    (with-temp-buffer
      (let ((status (apply #'process-file (car command) nil t nil (cdr command))))
        (unless (zerop status)
          (user-error "장치 unmount 실패: %s" (string-trim (buffer-string))))
        t))))

(defun imoogi-project-notes--buffers-under-directory (directory)
  "Return live file buffers whose files are under DIRECTORY."
  (let ((directory (file-name-as-directory (file-truename directory))))
    (cl-remove-if-not
     (lambda (buffer)
       (when-let* ((file (buffer-file-name buffer)))
         (file-in-directory-p (imoogi-project-notes--truename-if-present file)
                              directory)))
     (buffer-list))))

(defun imoogi-project-notes--show-dirty-buffers (buffers)
  "Display modified file BUFFERS that must be saved before unmount."
  (with-help-window "*imoogi unmount 변경 파일*"
    (princ "Unmount 전에 저장할 변경 파일\n\n")
    (dolist (buffer buffers)
      (princ (format "- %s\n" (buffer-file-name buffer))))))

(defun imoogi-project-notes--save-buffers-or-error (buffers)
  "Save BUFFERS, signaling before any later cleanup on failure."
  (dolist (buffer buffers)
    (with-current-buffer buffer
      (save-buffer))))

(defun imoogi-project-notes--cleanup-unmount-buffers (buffers)
  "Kill in-scope BUFFERS after they have been safely saved."
  (dolist (buffer buffers)
    (when (buffer-live-p buffer)
      (kill-buffer buffer))))

;;;###autoload
(defun imoogi-project-notes-unmount-device (&optional root)
  "Safely save and close buffers, then unmount ROOT's containing device."
  (interactive)
  (let* ((root (or root (imoogi-project-notes--select-mounted-root
                         "Unmount할 외장 루트: ")))
         ;; Preflight must happen before any mutation or save prompt.
         (descriptor (funcall imoogi-project-notes-unmount-preflight-function
                              root))
         (mount-point (plist-get descriptor :mount-point))
         (buffers (imoogi-project-notes--buffers-under-directory mount-point))
         (dirty (cl-remove-if-not #'buffer-modified-p buffers)))
    (unless (plist-get descriptor :supported)
      (user-error "이 장치는 물리 unmount를 지원하지 않습니다"))
    (when dirty
      (imoogi-project-notes--show-dirty-buffers dirty))
    (unless (y-or-n-p
             (format "변경 파일 %d개를 저장하고 장치를 unmount할까요? "
                     (length dirty)))
      (user-error "Unmount를 취소했습니다"))
    (imoogi-project-notes--save-buffers-or-error dirty)
    (imoogi-project-notes--cleanup-unmount-buffers buffers)
    ;; Workspace state is intentionally preserved unless ownership can be
    ;; proven; Treemacs may contain user-added roots outside this device.
    (unless (funcall imoogi-project-notes-unmount-executor-function descriptor)
      (user-error "장치를 unmount하지 못했습니다"))
    (message "imoogi: 변경 파일 %d개를 저장하고 %s를 unmount했습니다"
             (length dirty) mount-point)
    t))

;;;###autoload
(defun imoogi-project-notes-setup-guide ()
  "Explain the distinction between source work and project note folders."
  (interactive)
  (with-help-window "*imoogi 프로젝트 기록 안내*"
    (princ "프로젝트 기록 폴더 안내\n\n")
    (princ "imoogi에서 프로젝트를 선택할 때 보이는 경로는 소스 작업 폴더입니다.\n")
    (princ "예: ~/workspace/imoogi-emacs/\n\n")
    (princ "Org 문서는 소스나 Git worktree 안에 만들지 않습니다. 기본 문서 위치는\n")
    (princ "별도의 번호-프로젝트명 ~/project-notes/ 폴더입니다.\n")
    (princ "예: ~/project-notes/260925.01-imoogi-emacs/project.org\n\n")
    (princ "번호는 월간 2601.1, 연간 26.01, 일간 260925.01처럼 직접 입력합니다.\n")
    (princ "C-c h p m s  현재 소스 작업 폴더에 문서 폴더를 연결·생성\n")
    (princ "C-u C-c h p m s  문서가 저장될 폴더를 직접 지정\n")
    (princ "C-c h p m S  소스 폴더 없이 학습 노트와 작업공간 생성\n")
    (princ "C-c h p m l  등록된 프로젝트 또는 학습 노트를 선택·이동\n\n")
    (princ "학습 노트는 ~/project-notes/YY.NN-학습명/ 아래에 생성되며\n")
    (princ "노트 폴더 자체가 Perspective와 Treemacs의 작업공간이 됩니다.\n\n")
    (princ "같은 Git 저장소의 worktree는 한 문서 폴더를 공유하지만 journal의\n")
    (princ "재개 지점은 실제 작업 폴더별로 나뉩니다. 기존 문서는 덮어쓰지 않습니다.\n")))

(defun imoogi-project-notes--open-entry-file (entry key)
  "Open the file at KEY from project notes ENTRY."
  (imoogi-project-notes--find-file
   entry (imoogi-project-notes--alist-string key entry)
   (or (imoogi-project-notes--alist-string 'effective-source-root entry)
       (imoogi-project-notes--alist-string 'source-root entry))))

;;;###autoload
(defun imoogi-project-notes-list ()
  "Choose a registered project and open its source or principal note."
  (interactive)
  (let* ((entry (imoogi-project-notes--select-entry))
         (study-p (eq (imoogi-project-notes--entry-type entry) 'study))
         (destinations `((,(if study-p "학습 개요" "프로젝트 개요") . project-file)
                         ("할 일" . tasks-file)
                         (,(if study-p "학습 기록" "작업 기록·재개") . journal-file)
                         (,(if study-p "학습 작업공간" "소스 프로젝트") . source)))
         (destination (cdr (assoc
                            (completing-read "열기: " destinations nil t)
                            destinations))))
      (if (eq destination 'source)
        (if (eq (imoogi-project-notes--entry-type entry) 'study)
            (imoogi-project-notes--open-study-workspace entry)
          (progn
            (when (imoogi-project-notes--inactive-mounted-entry-p entry)
              (user-error "소스 프로젝트가 없습니다. 먼저 재연결하세요"))
            (let ((project-prompter
                 (lambda () (imoogi-project-notes--alist-string
                             'effective-source-root entry))))
              (imoogi-project-switch-perspective nil))))
      (imoogi-project-notes--open-entry-file entry destination))))

(defun imoogi-project-notes--agenda-files ()
  "Return existing task files for all registered project notes."
  (delete-dups
   (delq nil
         (mapcar (lambda (entry)
                   (let ((file (imoogi-project-notes--alist-string
                                'tasks-file entry)))
                     (and file (file-exists-p file) file)))
                 (imoogi-project-notes--all-entries)))))

(defun imoogi-project-notes--run-agenda (title files &optional category)
  "Show project execution dashboard TITLE using FILES.
When CATEGORY is non-nil, apply it as a global category filter."
  (unless files
    (user-error "Agenda에 표시할 프로젝트 작업 파일이 없습니다"))
  (let ((org-agenda-files files)
        (org-agenda-category-filter-preset
         (and category (list (concat "+" category)))))
    (org-agenda-run-series
     title
     '(((tags "DEADLINE<>\"\""
              ((org-agenda-overriding-header "기한 지난 작업")
               (org-agenda-skip-function #'imoogi-org-agenda-skip-not-overdue)
               (org-agenda-sorting-strategy
                '(deadline-up priority-down category-keep))))
        (todo "DOING" ((org-agenda-overriding-header "진행 중")))
        (todo "NEXT" ((org-agenda-overriding-header "다음 행동")))
        (agenda "" ((org-agenda-overriding-header "일정")
                    (org-agenda-span 7)
                    (org-agenda-skip-function #'imoogi-org-agenda-skip-overdue)))
        (todo "WAIT" ((org-agenda-overriding-header "대기 중"))))))))

;;;###autoload
(defun imoogi-project-notes-agenda-current ()
  "Show the execution agenda for the current project."
  (interactive)
  (let ((entry (or (imoogi-project-notes--current-entry)
                   (imoogi-project-notes--select-entry "Focus 프로젝트: "))))
    (let ((central (eq (imoogi-project-notes--entry-todo-storage entry)
                       'central)))
      (imoogi-project-notes--run-agenda
       "프로젝트 Focus"
       (list (imoogi-project-notes--alist-string 'tasks-file entry))
       (and central (imoogi-project-notes--entry-name entry))))))

;;;###autoload
(defun imoogi-project-notes-agenda-all ()
  "Show one execution dashboard across registered projects."
  (interactive)
  (imoogi-project-notes--run-agenda
   "전체 프로젝트 Dashboard" (imoogi-project-notes--agenda-files)))

(defun imoogi-project-notes--artifact-spec (kind)
  "Return the artifact template specification for KIND."
  (or (assq kind imoogi-project-notes--artifact-types)
      (user-error "알 수 없는 산출물 종류: %s" kind)))

(defun imoogi-project-notes--unique-artifact-file (directory prefix title)
  "Return a new artifact path below DIRECTORY for PREFIX and TITLE."
  (let* ((base (format "%s-%s-%s" (format-time-string "%Y%m%d") prefix
                       (imoogi-project-notes--slug title)))
         (candidate (expand-file-name (concat base ".org") directory))
         (number 2))
    (while (file-exists-p candidate)
      (setq candidate (expand-file-name
                       (format "%s-%d.org" base number) directory)
            number (1+ number)))
    candidate))

(defun imoogi-project-notes--append-artifact-link (artifact-id title)
  "Append a link to ARTIFACT-ID named TITLE under the current Org heading."
  (let ((link (format "- [[id:%s][%s]]" artifact-id title))
        (subtree-end (save-excursion (org-end-of-subtree t t))))
    (save-excursion
      (forward-line 1)
      (if (re-search-forward "^산출물:[[:space:]]*$" subtree-end t)
          (progn
            (forward-line 1)
            (while (and (< (point) subtree-end) (looking-at "^- "))
              (forward-line 1))
            (insert link "\n"))
        (goto-char subtree-end)
        (unless (bolp) (insert "\n"))
        (insert "\n산출물:\n" link "\n")))))

;;;###autoload
(defun imoogi-project-notes-create-artifact (kind title)
  "Create a KIND artifact named TITLE and link it to the current TODO.
The command assigns stable Org IDs to both sides, saves the task link, and
opens the new file below the project's artifacts directory."
  (interactive
   (progn
     (unless (derived-mode-p 'org-mode)
       (user-error "Org TODO heading에서 실행하세요"))
     (org-back-to-heading t)
     (let* ((labels (mapcar (lambda (spec)
                              (cons (nth 1 spec) (car spec)))
                            imoogi-project-notes--artifact-types))
            (kind (cdr (assoc (completing-read "산출물 종류: " labels nil t)
                              labels)))
            (default-title (org-get-heading t t t t)))
       (list kind (read-string "산출물 제목: " default-title)))))
  (unless (derived-mode-p 'org-mode)
    (user-error "Org TODO heading에서 실행하세요"))
  (org-back-to-heading t)
  (let ((entry (or (imoogi-project-notes--current-entry)
                   (imoogi-project-notes--select-entry "산출물 프로젝트: "))))
    (imoogi-project-notes--ensure-entry-mutable entry "산출물 생성")
    (let* ((spec (imoogi-project-notes--artifact-spec kind))
           (task-title (org-get-heading t t t t))
           (task-id (org-id-get-create))
           (artifact-id (org-id-new))
           (notes-dir (imoogi-project-notes--alist-string 'notes-dir entry))
           (artifact-dir (expand-file-name "artifacts/" notes-dir))
           (file (imoogi-project-notes--unique-artifact-file
                  artifact-dir (nth 2 spec) title))
           (project-name (imoogi-project-notes--entry-name entry))
           (content (imoogi-project-notes--template
                     "artifact"
                     `(("ARTIFACT_TITLE" . ,title)
                       ("ARTIFACT_TYPE" . ,(nth 1 spec))
                       ("ARTIFACT_ID" . ,artifact-id)
                       ("TASK_ID" . ,task-id)
                       ("TASK_TITLE" . ,task-title)
                       ("PROJECT_NAME" . ,project-name)
                       ("ARTIFACT_SECTIONS" . ,(nth 3 spec))))))
      (make-directory artifact-dir t)
      (imoogi-project-notes--write-new-file file content)
      (imoogi-project-notes--append-artifact-link artifact-id title)
      (when buffer-file-name (save-buffer))
      (org-id-add-location artifact-id file)
      (imoogi-project-notes--find-file
       entry file (imoogi-project-notes--current-source-root entry)))))

;;;###autoload
(defun imoogi-notes-scratch ()
  "Open the persistent scratch file under ~/notes without overwriting it."
  (interactive)
  (unless (fboundp 'imoogi-org--default-directory)
    (require 'imoogi-org))
  (let* ((directory (imoogi-org--default-directory))
         (file (expand-file-name "scratch.org" directory)))
    (if-let* ((buffer (find-buffer-visiting file)))
        (switch-to-buffer buffer)
      (make-directory directory t)
      (imoogi-project-notes--write-new-file
       file (imoogi-project-notes--template
             "scratch" `(("PROJECT_NAME" . "Scratch")
                         ("PROJECT_ROOT" . "")
                         ("TASKS_LINK" . "file:agenda.org"))))
      (find-file file))))

(defun imoogi-project-notes--restore-agenda-files ()
  "Restore existing registered project task files to agenda in memory."
  (when (listp org-agenda-files)
    (dolist (entry (condition-case nil
                       (imoogi-project-notes--all-entries)
                     (error (imoogi-project-notes--read-registry 'noerror))))
      (when-let* ((tasks-file (imoogi-project-notes--alist-string
                              'tasks-file entry)))
        (imoogi-project-notes--register-agenda-file-list-only tasks-file)))))

(use-package org
  :ensure nil
  :config
  (imoogi-project-notes--restore-agenda-files))

(provide 'imoogi-project-notes)
;;; 26-project-notes.el ends here
