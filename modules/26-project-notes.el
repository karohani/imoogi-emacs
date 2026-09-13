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

(defvar-local imoogi-project-notes-source-root nil
  "Source root associated with the current project notes buffer.")

(defvar-local imoogi-project-notes-directory-local nil
  "Project notes directory associated with the current buffer.")

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

(defun imoogi-project-notes--current-entry ()
  "Return the registry entry for the current source or notes buffer."
  (let ((entries (imoogi-project-notes--read-registry)))
    (or (and imoogi-project-notes-source-root
             (imoogi-project-notes--find-entry-by-key
              (imoogi-project-notes--identity-key imoogi-project-notes-source-root)
              entries))
        (and buffer-file-name
             (imoogi-project-notes--find-entry-by-notes-file buffer-file-name entries))
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

(defun imoogi-project-notes--values (project-name source-root task-file notes-dir)
  "Return template values for PROJECT-NAME, SOURCE-ROOT, TASK-FILE and NOTES-DIR."
  `(("PROJECT_NAME" . ,project-name)
    ("PROJECT_ROOT" . ,source-root)
    ("TASKS_LINK" . ,(concat "file:" (file-relative-name task-file notes-dir)))))

(defun imoogi-project-notes--entry (key source-root notes-dir todo-storage)
  "Build a registry entry."
  (let* ((project-file (expand-file-name "project.org" notes-dir))
         (tasks-file (if (eq todo-storage 'central)
                         (imoogi-project-notes--safe-ensure-central-agenda)
                       (expand-file-name "tasks.org" notes-dir)))
         (journal-file (expand-file-name "journal.org" notes-dir)))
    `((key . ,key)
      (source-root . ,(imoogi-project-notes--directory-file-name source-root))
      (notes-dir . ,(imoogi-project-notes--directory-file-name notes-dir))
      (project-file . ,project-file)
      (tasks-file . ,tasks-file)
      (journal-file . ,journal-file)
      (todo-storage . ,(symbol-name todo-storage)))))

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
              (imoogi-project-notes--alist-string 'source-root entry))
  (setq-local imoogi-project-notes-directory-local
              (imoogi-project-notes--alist-string 'notes-dir entry)))

(defun imoogi-project-notes--find-file (entry file &optional source-root)
  "Open FILE and attach project notes ENTRY context."
  (find-file file)
  (imoogi-project-notes--set-buffer-context entry)
  (when source-root
    (setq-local imoogi-project-notes-source-root
                (imoogi-project-notes--directory-file-name source-root)))
  (current-buffer))

(defun imoogi-project-notes--find-existing-or-create (entry file template-name values)
  "Open FILE for ENTRY, creating it from TEMPLATE-NAME with VALUES if absent."
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
(defun imoogi-project-notes-setup (&optional root directory)
  "Create or register project notes for source ROOT.
With interactive prefix argument, choose DIRECTORY manually.  Existing files are
never overwritten."
  (interactive
   (let* ((root (imoogi-project-notes--project-root))
          (directory (when current-prefix-arg
                       (read-directory-name "프로젝트 노트 폴더: "
                                            imoogi-project-notes-directory nil nil))))
     (list root directory)))
  (let* ((root (imoogi-project-notes--validate-source-root
                (or root (imoogi-project-notes--project-root))))
         (key (imoogi-project-notes--identity-key root))
         (entries (imoogi-project-notes--read-registry))
         (existing (imoogi-project-notes--find-entry-by-key key entries))
         (notes-dir (imoogi-project-notes--validate-notes-directory
                     (or directory
                         (imoogi-project-notes--alist-string 'notes-dir existing)
                         (imoogi-project-notes--default-notes-directory
                          root key entries))
                     key entries))
         (storage (if existing
                      (imoogi-project-notes--entry-todo-storage existing)
                    (if (member imoogi-project-notes-todo-storage '(project central))
                        imoogi-project-notes-todo-storage
                      'project)))
         (entry (imoogi-project-notes--entry key root notes-dir storage))
         (project-name (file-name-nondirectory (directory-file-name root))))
    (imoogi-project-notes--setup-files entry project-name)
    (imoogi-project-notes--save-entry entry)
    (imoogi-project-notes--register-agenda-target
     (imoogi-project-notes--alist-string 'tasks-file entry))
    (message "imoogi: 작업 폴더 %s → 문서 폴더 %s" root notes-dir)
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
    (imoogi-project-notes--find-file
     entry (imoogi-project-notes--alist-string 'journal-file entry) source-root)
    (imoogi-project-notes--ensure-worktree-journal-section entry source-root)
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
    (unless spec
      (user-error "알 수 없는 프로젝트 문서: %s" document))
    (let* ((notes-dir (imoogi-project-notes--alist-string 'notes-dir entry))
           (source-root (imoogi-project-notes--alist-string 'source-root entry))
           (project-name (file-name-nondirectory
                          (directory-file-name source-root)))
           (task-file (imoogi-project-notes--alist-string 'tasks-file entry))
           (file (expand-file-name (car spec)
                                   (expand-file-name "development/" notes-dir)))
           (values (imoogi-project-notes--values project-name source-root
                                                 task-file notes-dir)))
      (imoogi-project-notes--find-existing-or-create
       entry file (symbol-name document) values))))

(defun imoogi-project-notes--entry-label (entry)
  "Return a readable completion label for project notes ENTRY."
  (let* ((root (imoogi-project-notes--alist-string 'source-root entry))
         (name (file-name-nondirectory (directory-file-name root))))
    (format "%-24s %s" name (abbreviate-file-name root))))

(defun imoogi-project-notes--select-entry (&optional prompt)
  "Prompt for and return a registered project-notes entry.
Entries are identified by their source work directory; Org files live in the
separate notes directory recorded by each entry."
  (let* ((entries (imoogi-project-notes--read-registry))
         (candidates (mapcar (lambda (entry)
                               (cons (imoogi-project-notes--entry-label entry) entry))
                             entries)))
    (unless candidates
      (user-error "등록된 프로젝트 노트가 없습니다"))
    (cdr (assoc (completing-read (or prompt "소스 작업 폴더의 프로젝트: ")
                                 candidates nil t)
                candidates))))

;;;###autoload
(defun imoogi-project-notes-setup-guide ()
  "Explain the distinction between source work and project note folders."
  (interactive)
  (with-help-window "*imoogi 프로젝트 기록 안내*"
    (princ "프로젝트 기록 폴더 안내\n\n")
    (princ "imoogi에서 프로젝트를 선택할 때 보이는 경로는 소스 작업 폴더입니다.\n")
    (princ "예: ~/workspace/imoogi-emacs/\n\n")
    (princ "Org 문서는 소스나 Git worktree 안에 만들지 않습니다. 기본 문서 위치는\n")
    (princ "별도의 ~/project-notes/<시작일>-<프로젝트>/ 폴더입니다.\n")
    (princ "예: ~/project-notes/260918-imoogi-emacs/project.org\n\n")
    (princ "C-c h p m s  현재 소스 작업 폴더에 문서 폴더를 연결·생성\n")
    (princ "C-u C-c h p m s  문서가 저장될 폴더를 직접 지정\n")
    (princ "C-c h p m l  소스 작업 폴더 기준으로 등록 프로젝트를 선택·이동\n\n")
    (princ "같은 Git 저장소의 worktree는 한 문서 폴더를 공유하지만 journal의\n")
    (princ "재개 지점은 실제 작업 폴더별로 나뉩니다. 기존 문서는 덮어쓰지 않습니다.\n")))

(defun imoogi-project-notes--open-entry-file (entry key)
  "Open the file at KEY from project notes ENTRY."
  (imoogi-project-notes--find-file
   entry (imoogi-project-notes--alist-string key entry)
   (imoogi-project-notes--alist-string 'source-root entry)))

;;;###autoload
(defun imoogi-project-notes-list ()
  "Choose a registered project and open its source or principal note."
  (interactive)
  (let* ((entry (imoogi-project-notes--select-entry))
         (destinations '(("프로젝트 개요" . project-file)
                         ("할 일" . tasks-file)
                         ("작업 기록·재개" . journal-file)
                         ("소스 프로젝트" . source)))
         (destination (cdr (assoc
                            (completing-read "열기: " destinations nil t)
                            destinations))))
    (if (eq destination 'source)
        (let ((project-prompter
               (lambda () (imoogi-project-notes--alist-string
                           'source-root entry))))
          (imoogi-project-switch-perspective nil))
      (imoogi-project-notes--open-entry-file entry destination))))

(defun imoogi-project-notes--agenda-files ()
  "Return existing task files for all registered project notes."
  (delete-dups
   (delq nil
         (mapcar (lambda (entry)
                   (let ((file (imoogi-project-notes--alist-string
                                'tasks-file entry)))
                     (and file (file-exists-p file) file)))
                 (imoogi-project-notes--read-registry)))))

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
       (and central
            (file-name-nondirectory
             (directory-file-name
              (imoogi-project-notes--alist-string 'source-root entry))))))))

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
  (let* ((entry (or (imoogi-project-notes--current-entry)
                    (imoogi-project-notes--select-entry "산출물 프로젝트: ")))
         (spec (imoogi-project-notes--artifact-spec kind))
         (task-title (org-get-heading t t t t))
         (task-id (org-id-get-create))
         (artifact-id (org-id-new))
         (notes-dir (imoogi-project-notes--alist-string 'notes-dir entry))
         (artifact-dir (expand-file-name "artifacts/" notes-dir))
         (file (imoogi-project-notes--unique-artifact-file
                artifact-dir (nth 2 spec) title))
         (project-name (file-name-nondirectory
                        (directory-file-name
                         (imoogi-project-notes--alist-string 'source-root entry))))
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
     entry file (imoogi-project-notes--current-source-root entry))))

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
    (dolist (entry (imoogi-project-notes--read-registry 'noerror))
      (when-let* ((tasks-file (imoogi-project-notes--alist-string
                              'tasks-file entry)))
        (imoogi-project-notes--register-agenda-file-list-only tasks-file)))))

(use-package org
  :ensure nil
  :config
  (imoogi-project-notes--restore-agenda-files))

(provide 'imoogi-project-notes)
;;; 26-project-notes.el ends here
