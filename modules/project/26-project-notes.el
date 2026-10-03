;;; 26-project-notes.el --- project-scoped Org notes -*- lexical-binding: t; -*-

;;; Code:

(imoogi-require "26-project-notes" 'cl-lib 'json 'org 'org-agenda 'org-id
                'project 'seq 'subr-x)

(require 'cl-lib)
(require 'json)
(require 'org)
(require 'org-agenda)
(require 'org-id)
(require 'project)
(require 'seq)
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

(defcustom imoogi-project-notes-cache-directory nil
  "Optional directory passed to the imoogi-notes document cache CLI.
When nil, Emacs omits cache_dir and lets the CLI choose its own user cache
location.  This keeps project folders free of editor-local cache databases."
  :type '(choice (const :tag "Use CLI default user cache" nil)
                 directory)
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

(defvar imoogi-project-notes--pending-link-operation nil
  "Recoverable in-memory project document relation operation.")

(defvar imoogi-project-notes--document-cache-snapshot nil
  "Dynamic CLI catalog snapshot used to avoid project-wide Emacs scans.")

(defvar imoogi-project-notes--pending-cache-process nil
  "Current asynchronous imoogi-notes catalog process.")

(defvar imoogi-project-notes--snapshot-live-record-cache (make-hash-table :test #'equal)
  "Live buffer records layered over a CLI document cache snapshot.")

(defconst imoogi-project-notes--metadata-file-name ".imoogi-project.json"
  "File that identifies the kind and layout of a project-notes directory.")

(defconst imoogi-project-notes--mounted-root-marker-file-name
  ".imoogi-mounted-root.json"
  "File written at a mounted root that proves the device is really attached.")

(defconst imoogi-project-notes--number-id-regexp
  "\\(?:[0-9]\\{6\\}\\.[0-9][0-9]\\|[0-9]\\{4\\}\\.[0-9][0-9]\\|[0-9]\\{2\\}\\.[0-9][0-9]\\|L[0-9][0-9][0-9]\\)"
  "Regexp matching a project-notes numbering id without folder slug.")

(defconst imoogi-project-notes--folder-name-regexp
  (concat "\\`" imoogi-project-notes--number-id-regexp
          "\\(?:-[[:alnum:]_.-]+\\)?\\'")
  "Regexp for accepted project-notes folder names.")

(defconst imoogi-project-notes--number-level-order
  '(daily monthly yearly lifetime)
  "Supported project-notes numbering levels from narrow to broad.")

(defun imoogi-project-notes--number-id-from-name (name)
  "Return the visible numbering id prefix from folder NAME, or nil."
  (when (and (stringp name)
             (string-match
              (concat "\\`\\(" imoogi-project-notes--number-id-regexp
                      "\\)\\(?:-[[:alnum:]_.-]+\\)?\\'")
              name))
    (match-string 1 name)))

(defun imoogi-project-notes--parse-number-id (number-id)
  "Parse NUMBER-ID and return its structured alist, or nil.
Recognized forms are daily YYMMDD.NN, monthly YYMM.NN, yearly YY.NN, and
lifetime LNNN."
  (when (stringp number-id)
    (cond
     ((string-match "\\`\\([0-9]\\{6\\}\\)\\.\\([0-9][0-9]\\)\\'" number-id)
      `((number-id . ,number-id)
        (number-level . daily)
        (scope . ,(match-string 1 number-id))
        (sequence . ,(string-to-number (match-string 2 number-id)))))
     ((string-match "\\`\\([0-9]\\{4\\}\\)\\.\\([0-9][0-9]\\)\\'" number-id)
      `((number-id . ,number-id)
        (number-level . monthly)
        (scope . ,(match-string 1 number-id))
        (sequence . ,(string-to-number (match-string 2 number-id)))))
     ((string-match "\\`\\([0-9]\\{2\\}\\)\\.\\([0-9][0-9]\\)\\'" number-id)
      `((number-id . ,number-id)
        (number-level . yearly)
        (scope . ,(match-string 1 number-id))
        (sequence . ,(string-to-number (match-string 2 number-id)))))
     ((string-match "\\`L\\([0-9][0-9][0-9]\\)\\'" number-id)
      `((number-id . ,number-id)
        (number-level . lifetime)
        (scope . "L")
        (sequence . ,(string-to-number (match-string 1 number-id))))))))

(defun imoogi-project-notes--parse-folder-number (name)
  "Parse the visible project-notes number from folder NAME."
  (when-let* ((number-id (imoogi-project-notes--number-id-from-name name)))
    (imoogi-project-notes--parse-number-id number-id)))

(defun imoogi-project-notes--format-number-id (level scope sequence)
  "Return a NUMBER-ID for LEVEL, SCOPE, and SEQUENCE."
  (unless (and (integerp sequence) (<= 1 sequence))
    (user-error "번호 순서는 1 이상의 정수여야 합니다"))
  (pcase level
    ('daily
     (unless (and (stringp scope)
                  (string-match-p "\\`[0-9]\\{6\\}\\'" scope))
       (user-error "일간 프로젝트 범위는 YYMMDD 형식이어야 합니다: %s" scope))
     (when (> sequence 99)
       (user-error "일간 프로젝트 순서는 99를 초과할 수 없습니다"))
     (format "%s.%02d" scope sequence))
    ('monthly
     (unless (and (stringp scope)
                  (string-match-p "\\`[0-9]\\{4\\}\\'" scope))
       (user-error "월간 프로젝트 범위는 YYMM 형식이어야 합니다: %s" scope))
     (when (> sequence 99)
       (user-error "월간 프로젝트 순서는 99를 초과할 수 없습니다"))
     (format "%s.%02d" scope sequence))
    ('yearly
     (unless (and (stringp scope)
                  (string-match-p "\\`[0-9]\\{2\\}\\'" scope))
       (user-error "연간 프로젝트 범위는 YY 형식이어야 합니다: %s" scope))
     (when (> sequence 99)
       (user-error "연간 프로젝트 순서는 99를 초과할 수 없습니다"))
     (format "%s.%02d" scope sequence))
    ('lifetime
     (when (> sequence 999)
       (user-error "평생 프로젝트 순서는 999를 초과할 수 없습니다"))
     (format "L%03d" sequence))
    (_ (user-error "알 수 없는 프로젝트 번호 수준입니다: %s" level))))

(defun imoogi-project-notes--doctor-name-scope (name)
  "Return (LEVEL SCOPE SLUG) for a folder NAME lacking its `.NN' suffix."
  (when (string-match
         "\\`\\([0-9]\\{6\\}\\|[0-9]\\{4\\}\\|[0-9]\\{2\\}\\)-\\(.+\\)\\'" name)
    (let ((scope (match-string 1 name)))
      (list (pcase (length scope) (6 'daily) (4 'monthly) (_ 'yearly))
            scope
            (match-string 2 name)))))

(defun imoogi-project-notes--doctor-folder-parts (name)
  "Return (LEVEL SCOPE SLUG) for folder NAME, numbered or not.
SLUG is nil when NAME carries no suffix after its numbering id."
  (if-let* ((info (imoogi-project-notes--parse-folder-number name)))
      (let ((id (alist-get 'number-id info)))
        (list (alist-get 'number-level info)
              (alist-get 'scope info)
              (and (> (length name) (length id))
                   (substring name (1+ (length id))))))
    (imoogi-project-notes--doctor-name-scope name)))

(defun imoogi-project-notes--doctor-resolve-answer (name answer)
  "Return the folder name requested by ANSWER for folder NAME, or nil.
Nil means skip: ANSWER is blank or equals NAME.  `+' selects the next free
number in NAME's scope and a bare number selects that sequence; any other
ANSWER is taken as the complete new folder name."
  (let ((answer (string-trim answer)))
    (cond
     ((or (string-empty-p answer) (string= answer name)) nil)
     ((or (string= answer "+") (string-match-p "\\`[0-9]+\\'" answer))
      (pcase (imoogi-project-notes--doctor-folder-parts name)
        (`(,level ,scope ,slug)
         (let ((id (if (string= answer "+")
                       (imoogi-project-notes--next-number-id level scope)
                     (imoogi-project-notes--format-number-id
                      level scope (string-to-number answer)))))
           (if slug (format "%s-%s" id slug) id)))
        (_ (user-error "번호 범위를 알 수 없는 폴더라 '%s'를 쓸 수 없습니다: %s"
                       answer name))))
     (t answer))))

(defconst imoogi-project-notes--doctor-log-buffer "*imoogi-project-notes-doctor*"
  "Buffer recording what the project-notes doctor did and why it failed.")

(defun imoogi-project-notes--doctor-log (format-string &rest args)
  "Append a timestamped line built from FORMAT-STRING and ARGS to the doctor log."
  (with-current-buffer (get-buffer-create imoogi-project-notes--doctor-log-buffer)
    (let ((inhibit-read-only t))
      (goto-char (point-max))
      (insert (format-time-string "[%F %T] ")
              (apply #'format format-string args)
              "\n"))))

(defun imoogi-project-notes--doctor-suggested-name (name)
  "Return a valid numbered suggestion for invalid folder NAME.
The sequence is the next free number after the largest one in NAME's scope."
  (pcase (imoogi-project-notes--doctor-name-scope name)
    (`(,level ,scope ,slug)
     (format "%s-%s" (imoogi-project-notes--next-number-id level scope) slug))
    (_ name)))

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

(defconst imoogi-project-notes--default-artifact-presets
  '((development
     (spec "스펙" "spec"
           "** 문제와 목표\n\n** 요구사항\n\n** 구현 방안\n\n** 완료 조건\n\n** 검증 방법\n")
     (investigation "문제 분석" "investigation"
                    "** 증상\n\n** 재현 방법\n\n** 관찰한 증거\n\n** 원인\n\n** 해결과 검증\n")
     (decision "결정" "decision"
               "** 배경\n\n** 검토한 대안\n\n** 선택\n\n** 이유\n\n** 영향과 남는 제약\n")
     (meeting "회의" "meeting"
              "** 참석자와 목적\n\n** 논의 내용\n\n** 결정 사항\n\n** 후속 작업\n")
     (runbook "운영 절차" "runbook"
              "** 목적\n\n** 사전 조건\n\n** 절차\n\n** 확인 방법\n\n** 실패 시 복구\n"))
    (study
     (concept "개념 노트" "concept"
              "** 핵심 개념\n\n** 예시\n\n** 연결된 개념\n\n** 헷갈리는 점\n")
     (flashcard "암기 카드" "flashcard"
                "** 앞면\n\n** 뒷면\n\n** 기억 단서\n")
     (question "문제 카드" "question"
               "** 문제\n\n** 풀이\n\n** 오답 원인\n\n** 다시 볼 점\n")))
  "Built-in artifact presets copied into each project's metadata.")

(defun imoogi-project-notes--default-artifact-preset (kind)
  "Return a mutable default artifact preset for KIND."
  (copy-tree (or (alist-get kind imoogi-project-notes--default-artifact-presets)
                 (alist-get 'development imoogi-project-notes--default-artifact-presets))))

(defun imoogi-project-notes--artifact-preset-to-json (preset)
  "Convert runtime PRESET specs to JSON-safe alists."
  (mapcar (lambda (spec)
            `((kind . ,(symbol-name (nth 0 spec)))
              (label . ,(nth 1 spec))
              (prefix . ,(nth 2 spec))
              (sections . ,(nth 3 spec))))
          preset))

(defun imoogi-project-notes--valid-artifact-prefix-p (prefix)
  "Return non-nil when PREFIX is one safe filename component."
  (and (stringp prefix)
       (string-match-p "\\`[[:alnum:]_+=.@-]+\\'" prefix)))

(defun imoogi-project-notes--artifact-preset-from-json (value)
  "Convert JSON-safe PRESET VALUE to runtime specs."
  (when (or (listp value) (vectorp value))
    (let ((value (if (vectorp value) (append value nil) value)))
      (cl-loop for spec in value
             for kind = (imoogi-project-notes--alist-string 'kind spec)
             for label = (imoogi-project-notes--alist-string 'label spec)
             for prefix = (imoogi-project-notes--alist-string 'prefix spec)
             for sections = (imoogi-project-notes--alist-string 'sections spec)
             when (and kind label prefix sections
                        (imoogi-project-notes--valid-artifact-prefix-p prefix))
             collect (list (intern kind) label prefix sections)))))

(defun imoogi-project-notes--entry-artifact-preset (entry)
  "Return ENTRY's artifact preset, falling back for legacy metadata."
  (or (alist-get 'artifact-preset entry)
      (imoogi-project-notes--default-artifact-preset
       (if (eq (imoogi-project-notes--entry-type entry) 'study)
           'study
         'development))))

(defun imoogi-project-notes--registry-file ()
  "Return the project notes registry path."
  (locate-user-emacs-file ".cache/project-notes.json"))

(defun imoogi-project-notes--mounted-roots-file ()
  "Return the host-local mounted roots registry path."
  (locate-user-emacs-file ".cache/project-notes-mounted-roots.json"))

(defun imoogi-project-notes--repair-marker-file ()
  "Return the path of the pending session-repair marker."
  (locate-user-emacs-file "imoogi-project-notes/repair-pending.json"))

(defun imoogi-project-notes--write-repair-marker
    (entry old-root new-root failures)
  "Persist a recoverable session projection marker for ENTRY.
FAILURES describes projections that could not be updated.  Never overwrite an
existing marker: the doctor must repair the older projection first."
  (let ((file (imoogi-project-notes--repair-marker-file)))
    (when (file-exists-p file)
      (user-error "보류 중인 project-notes 복구가 있습니다: %s" file))
    (make-directory (file-name-directory file) t)
    (let ((json-encoding-pretty-print t))
      (with-temp-file file
        (insert (json-encode
                `((note_id . ,(imoogi-project-notes--entry-note-id entry))
                  (old_path . ,old-root)
                  (new_path . ,new-root)
                  (failed_projections . ,failures)
                  (expected_workspaces .
                   ,(vconcat
                     (delete-dups
                      (delq nil (mapcar #'car failures)))))
                  (created_at . ,(format-time-string "%Y-%m-%dT%H:%M:%S%z"))))
                "\n")))))

(defun imoogi-project-notes--consume-repair-marker ()
  "Repair the projection recorded in the pending marker, if any.
Return non-nil when a marker was consumed successfully."
  (let ((file (imoogi-project-notes--repair-marker-file)))
    (when (file-readable-p file)
      (let* ((json-object-type 'alist)
             (json-key-type 'symbol)
             (json-array-type 'list)
             (marker (json-read-file file))
             (old-root (alist-get 'old_path marker))
             (new-root (alist-get 'new_path marker))
             (expected-workspaces
              (delete-dups
               (cl-remove-if-not #'stringp
                                 (append (alist-get 'expected_workspaces marker)
                                         nil)))))
        (unless expected-workspaces
          (user-error "project-notes 복구 marker에 workspace 정보가 없습니다"))
        (unless (and old-root new-root (file-directory-p new-root))
          (user-error "project-notes 복구 marker의 경로가 유효하지 않습니다: %s" file))
        (unless (and (fboundp 'treemacs-workspaces)
                     (fboundp 'treemacs-workspace->projects)
                     (fboundp 'treemacs-project->path)
                     (fboundp 'treemacs-do-remove-project-from-workspace)
                     (fboundp 'treemacs-do-add-project-to-workspace))
          (require 'treemacs-workspaces nil t))
        (unless (and (fboundp 'treemacs-workspaces)
                     (fboundp 'treemacs-workspace->projects)
                     (fboundp 'treemacs-project->path)
                     (fboundp 'treemacs-do-remove-project-from-workspace)
                     (fboundp 'treemacs-do-add-project-to-workspace))
          (user-error "Treemacs가 로드되지 않아 project-notes 복구를 수행할 수 없습니다"))
        (let ((workspaces (treemacs-workspaces)))
          (unless (cl-every (lambda (name)
                              (cl-some (lambda (workspace)
                                         (equal name
                                                (treemacs-workspace->name workspace)))
                                       workspaces))
                            expected-workspaces)
            (user-error "복구 대상 Treemacs workspace가 없어 marker를 보존합니다")))
        (let ((failures
               (imoogi-project-notes--replace-treemacs-root old-root new-root)))
          (if failures
              (user-error "Treemacs 복구가 아직 끝나지 않았습니다: %S" failures)
            (let ((workspaces (treemacs-workspaces)))
              (unless
                  (cl-every
                   (lambda (name)
                     (let ((workspace
                            (cl-find name workspaces
                                     :key #'treemacs-workspace->name
                                     :test #'equal)))
                       (and workspace
                            (cl-some
                             (lambda (project)
                               (equal (directory-file-name
                                       (expand-file-name
                                        (treemacs-project->path project)))
                                      (directory-file-name
                                       (expand-file-name new-root))))
                             (treemacs-workspace->projects workspace))))
                   expected-workspaces)
                (user-error "Treemacs 복구 결과를 확인하지 못해 marker를 보존합니다")))
            (delete-file file)
            t)))))))

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

(defvar imoogi-project-notes--note-id-counter 0
  "Process-local salt for portable note IDs.")

(defun imoogi-project-notes--new-note-id (&optional seed)
  "Return a new portable note id, salted by optional SEED."
  (setq imoogi-project-notes--note-id-counter
        (1+ imoogi-project-notes--note-id-counter))
  (upcase
   (if (fboundp 'uuidgen-4)
       (uuidgen-4)
     (secure-hash
      'sha1
      (format "%s\0%s\0%s\0%s\0%s\0%s"
              seed
              imoogi-project-notes--note-id-counter
              (float-time) (random) (emacs-pid) (user-uid))))))

(defun imoogi-project-notes--entry-note-id (entry &optional create)
  "Return ENTRY's portable note id.
When CREATE is non-nil, create and store a new id in ENTRY when absent."
  (or (imoogi-project-notes--alist-string 'note-id entry)
      (and create
           (let ((note-id (imoogi-project-notes--new-note-id
                           (format "%S" entry))))
             (setf (alist-get 'note-id entry) note-id)
             note-id))))

(defun imoogi-project-notes--entry-derived-note-id (entry)
  "Return ENTRY's persisted note id or a stable read-side legacy id."
  (or (imoogi-project-notes--entry-note-id entry)
      (concat "legacy:"
              (secure-hash
               'sha1
               (concat (or (imoogi-project-notes--alist-string 'key entry) "")
                       "\0"
                       (or (imoogi-project-notes--alist-string
                            'notes-dir entry)
                           ""))))))

(defun imoogi-project-notes--number-info-for-directory (directory)
  "Return parsed numbering info for DIRECTORY's folder name."
  (imoogi-project-notes--parse-folder-number
   (file-name-nondirectory (directory-file-name directory))))

(defun imoogi-project-notes--entry-number-info (entry)
  "Return parsed numbering info for ENTRY from metadata or folder name."
  (or (when-let* ((number-id
                   (imoogi-project-notes--alist-string 'number-id entry)))
        (imoogi-project-notes--parse-number-id number-id))
      (when-let* ((directory
                   (imoogi-project-notes--alist-string 'notes-dir entry)))
        (imoogi-project-notes--number-info-for-directory directory))))

(defun imoogi-project-notes--entry-ensure-derived-fields (entry &optional create-note-id)
  "Populate optional identity and numbering fields on ENTRY."
  (imoogi-project-notes--entry-note-id entry create-note-id)
  (when-let* ((info (imoogi-project-notes--entry-number-info entry)))
    (setf (alist-get 'number-id entry)
          (alist-get 'number-id info))
    (setf (alist-get 'number-level entry)
          (symbol-name (alist-get 'number-level info))))
  (unless (imoogi-project-notes--alist-string 'source-root-kind entry)
    (setf (alist-get 'source-root-kind entry)
          (if (eq (imoogi-project-notes--entry-type entry) 'study)
              "notes"
            "source")))
  entry)

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
  (let* ((number-id (imoogi-project-notes--alist-string 'number_id metadata))
         (number-level (imoogi-project-notes--alist-string 'number_level metadata))
         (number-info (and number-id
                           (imoogi-project-notes--parse-number-id number-id)))
         (source-kind (imoogi-project-notes--alist-string
                       'source_root_kind metadata))
         (workspace-role (imoogi-project-notes--alist-string
                          'workspace_role metadata)))
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
         (let ((note-id (imoogi-project-notes--alist-string
                         'note_id metadata)))
           (or (null note-id) (not (string-empty-p note-id))))
         (or (null number-id) number-info)
         (or (null number-level)
             (and number-info
                  (string= number-level
                           (symbol-name
                            (alist-get 'number-level number-info)))))
         (or (null source-kind) (member source-kind '("source" "notes")))
         (or (null workspace-role)
             (member workspace-role '("project" "study")))
         (if (string= (imoogi-project-notes--alist-string 'type metadata) "study")
             (imoogi-project-notes--alist-string 'study_id metadata)
           t))))

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
         (number-id (or (imoogi-project-notes--alist-string
                         'number_id metadata)
                        (imoogi-project-notes--number-id-from-name
                         (file-name-nondirectory
                          (directory-file-name root)))))
         (number-info (and number-id
                           (imoogi-project-notes--parse-number-id number-id)))
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
      (note-id . ,(imoogi-project-notes--alist-string 'note_id metadata))
      (number-id . ,(and number-info (alist-get 'number-id number-info)))
      (number-level . ,(and number-info
                            (symbol-name (alist-get 'number-level number-info))))
      (source-root-kind . ,(imoogi-project-notes--alist-string
                            'source_root_kind metadata))
      (workspace-role . ,(imoogi-project-notes--alist-string
                          'workspace_role metadata))
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
                           "project"))
      (artifact-preset . ,(or (imoogi-project-notes--artifact-preset-from-json
                               (alist-get 'artifact_preset metadata))
                              (imoogi-project-notes--default-artifact-preset
                               (if (string= type "study") 'study 'development)))))))

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
                  ;; An unplugged root cannot be canonicalized; compare its
                  ;; recorded path instead of failing the whole registration.
                  (let ((path (imoogi-project-notes--alist-string 'path root)))
                    (if (file-directory-p path)
                        (imoogi-project-notes--canonical-directory path)
                      (imoogi-project-notes--directory-file-name path)))))
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

(defun imoogi-project-notes--mounted-root-marker-file (root)
  "Return the marker file path for mounted ROOT."
  (expand-file-name imoogi-project-notes--mounted-root-marker-file-name
                    (imoogi-project-notes--alist-string 'path root)))

(defun imoogi-project-notes--write-mounted-root-marker (root)
  "Write the device marker identifying mounted ROOT."
  (with-temp-file (imoogi-project-notes--mounted-root-marker-file root)
    (let ((json-encoding-pretty-print t))
      (insert (json-encode
               `((version . 1)
                 (id . ,(imoogi-project-notes--alist-string 'id root))
                 (label . ,(imoogi-project-notes--alist-string 'label root))))
              "\n"))))

;; @MX:NOTE A bare directory check cannot tell an attached device from an
;; empty stand-in folder left at the same path after unmount; only the marker
;; written at registration can.  Reads may still scan `unverified' roots.
(defun imoogi-project-notes--mounted-root-status (root)
  "Return the live connection status of mounted ROOT.
`disconnected' when its path is absent, `connected' when the marker id
matches ROOT, `unverified' when the folder has no marker, and `foreign' when
the marker belongs to another root."
  (let ((path (imoogi-project-notes--alist-string 'path root)))
    (if (not (and path (file-directory-p path)))
        'disconnected
      (let* ((file (imoogi-project-notes--mounted-root-marker-file root))
             (marker (and (file-readable-p file)
                          (ignore-errors
                            (let ((json-object-type 'alist)
                                  (json-key-type 'symbol))
                              (json-read-file file))))))
        (cond ((not (file-exists-p file)) 'unverified)
              ((equal (alist-get 'id marker)
                      (imoogi-project-notes--alist-string 'id root))
               'connected)
              (t 'foreign))))))

(defun imoogi-project-notes--mount-point-note (path)
  "Describe where PATH is mounted, warning when it is the system disk."
  (let ((mount (plist-get (imoogi-project-notes--df-descriptor path)
                          :mount-point)))
    (cond ((null mount) "")
          ((member mount '("/" "/System/Volumes/Data"))
           (format " [경고: 내장 디스크(%s)로 보입니다]" mount))
          (t (format " [마운트 위치: %s]" mount)))))

;; @MX:ANCHOR Single write gate for mounted roots (setup, setup-study, move,
;; location chooser); keep every mounted write behind it.
(defun imoogi-project-notes--ensure-mounted-root-writable (root)
  "Signal unless mounted ROOT is enabled and verifiably attached.
An unverified root is marked after confirmation; this is also how roots
registered before markers existed are migrated."
  (let ((path (imoogi-project-notes--alist-string 'path root))
        (label (imoogi-project-notes--alist-string 'label root)))
    (when (eq (alist-get 'enabled root) :json-false)
      (user-error "비활성 외장 루트에는 쓸 수 없습니다: %s" label))
    (pcase (imoogi-project-notes--mounted-root-status root)
      ('connected root)
      ('disconnected
       (user-error "외장 노트 루트가 연결되어 있지 않습니다: %s" path))
      ('foreign
       (user-error "다른 외장 루트의 표식이 있는 폴더입니다: %s" path))
      ('unverified
       (unless (y-or-n-p
                (format "%s 에 장치 표식이 없습니다%s. 이 폴더가 외장 루트 %s 가 맞나요? "
                        path (imoogi-project-notes--mount-point-note path) label))
         (user-error "외장 루트를 확인하지 못해 중단했습니다: %s" path))
       (imoogi-project-notes--write-mounted-root-marker root)
       root))))

(defun imoogi-project-notes--mounted-root-for-directory (directory)
  "Return the enabled registered mounted root containing DIRECTORY, or nil."
  (when directory
    (let ((directory (file-truename (expand-file-name directory))))
      (cl-find-if
       (lambda (root)
         (let ((path (imoogi-project-notes--alist-string 'path root)))
           (and (not (eq (alist-get 'enabled root) :json-false))
                path
                (file-directory-p path)
                (file-in-directory-p directory (file-truename path)))))
       (alist-get 'roots (imoogi-project-notes--read-mounted-state 'noerror))))))

(defun imoogi-project-notes--decorate-local-entry (entry)
  "Return a copied local registry ENTRY with read-side identity fields."
  (let ((copy (copy-tree entry)))
    (imoogi-project-notes--entry-ensure-derived-fields copy)
    (push '(origin . local) copy)
    (push (cons 'logical-key (alist-get 'key copy)) copy)
    (push (cons 'logical-note-id
                (imoogi-project-notes--entry-derived-note-id copy))
          copy)
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
    (imoogi-project-notes--entry-ensure-derived-fields copy)
    (push (cons 'logical-note-id
                (imoogi-project-notes--entry-derived-note-id copy))
          copy)
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

(defun imoogi-project-notes--scan-local-metadata ()
  "Return local metadata entries, including folders missing from the registry."
  (let ((root (expand-file-name imoogi-project-notes-directory))
        entries)
    (when (file-directory-p root)
      (dolist (file (directory-files-recursively
                     root
                     (concat "\\`"
                             (regexp-quote
                              imoogi-project-notes--metadata-file-name)
                             "\\'")
                     nil nil))
        (when-let* ((metadata
                     (imoogi-project-notes--read-metadata-file file 'noerror))
                    (entry (ignore-errors
                             (imoogi-project-notes--metadata-entry
                              file metadata))))
          (push (cons 'origin 'local) entry)
          (push (cons 'instance-id
                      (concat "local:"
                              (or (alist-get 'key entry)
                                  (imoogi-project-notes--entry-derived-note-id
                                   entry))))
                entry)
          (push entry entries))))
    (nreverse entries)))

(defun imoogi-project-notes--all-entries ()
  "Return local and currently available mounted project-note entries.
Local registry entries win when the same canonical notes directory is also
found below a mounted root.  Equal logical keys on different mounted roots
remain distinct through their `instance-id'."
  (let* ((locals (append (imoogi-project-notes--scan-local-metadata)
                         (mapcar #'imoogi-project-notes--decorate-local-entry
                                 (imoogi-project-notes--read-registry))))
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

(defun imoogi-project-notes--root-number-id-entries (root origin &optional label)
  "Return numbering id entries discovered directly below ROOT."
  (let ((root (and root (file-name-as-directory (expand-file-name root))))
        result)
    (when (and root (file-directory-p root))
      (dolist (directory (directory-files root t "^[^.].*" t))
        (when (file-directory-p directory)
          (let* ((name (file-name-nondirectory
                        (directory-file-name directory)))
                 (metadata-file (imoogi-project-notes--metadata-file directory))
                 (metadata (and (file-readable-p metadata-file)
                                (imoogi-project-notes--read-metadata-file
                                 metadata-file 'noerror)))
                 (metadata-entry
                  (and metadata
                       (ignore-errors
                         (imoogi-project-notes--metadata-entry
                          metadata-file metadata))))
                 (number-id
                  (or (and metadata-entry
                           (imoogi-project-notes--alist-string
                            'number-id metadata-entry))
                      (imoogi-project-notes--number-id-from-name name)))
                 (number-info (and number-id
                                   (imoogi-project-notes--parse-number-id
                                    number-id))))
            (when number-info
              (push `((number-id . ,(alist-get 'number-id number-info))
                      (number-level . ,(symbol-name
                                        (alist-get 'number-level number-info)))
                      (directory . ,(imoogi-project-notes--directory-file-name
                                     directory))
                      (folder-name . ,name)
                      (origin . ,origin)
                      ,@(when label `((label . ,label)))
                      ,@(when metadata-entry
                          `((key . ,(alist-get 'key metadata-entry))
                            (logical-note-id
                             . ,(imoogi-project-notes--entry-derived-note-id
                                 metadata-entry)))))
                    result))))))
    (nreverse result)))

(defun imoogi-project-notes--all-number-id-entries (&optional additional-root)
  "Return visible numbering ids from local, mounted, and ADDITIONAL-ROOT roots."
  (let* ((mounted-state (imoogi-project-notes--read-mounted-state 'noerror))
         (roots `((local ,imoogi-project-notes-directory nil)
                  ,@(mapcar
                     (lambda (root)
                       `(mounted ,(alist-get 'path root)
                                 ,(imoogi-project-notes--alist-string
                                   'label root)))
                     (cl-remove-if
                      (lambda (root)
                        (eq (alist-get 'enabled root) :json-false))
                      (alist-get 'roots mounted-state)))
                  ,@(when additional-root
                      `((additional ,additional-root nil)))))
         result)
    (dolist (spec roots)
      (setq result
            (append result
                    (imoogi-project-notes--root-number-id-entries
                     (nth 1 spec) (car spec) (nth 2 spec)))))
    (cl-remove-duplicates
     result
     :test (lambda (left right)
             (and (equal (alist-get 'number-id left)
                         (alist-get 'number-id right))
                  (equal (alist-get 'directory left)
                         (alist-get 'directory right)))))))

(defun imoogi-project-notes--number-id-conflict (number-id &optional except-directory)
  "Return an existing numbering entry for NUMBER-ID, ignoring EXCEPT-DIRECTORY."
  (let ((except-directory (and except-directory
                               (imoogi-project-notes--directory-file-name
                                except-directory))))
    (cl-find-if
     (lambda (entry)
       (and (string= number-id (alist-get 'number-id entry))
            (not (and except-directory
                      (string=
                       except-directory
                       (imoogi-project-notes--directory-file-name
                        (alist-get 'directory entry)))))))
     (imoogi-project-notes--all-number-id-entries))))

(defun imoogi-project-notes--scope-number-ids (level scope)
  "Return sorted visible numbering ids that share LEVEL and SCOPE."
  (sort (delete-dups
         (delq nil
               (mapcar
                (lambda (entry)
                  (let ((info (imoogi-project-notes--parse-number-id
                               (alist-get 'number-id entry))))
                    (and (eq level (alist-get 'number-level info))
                         (equal scope (alist-get 'scope info))
                         (alist-get 'number-id info))))
                (imoogi-project-notes--all-number-id-entries))))
        #'string<))

(defun imoogi-project-notes--next-sequence (level scope)
  "Return one more than the largest sequence used in LEVEL and SCOPE."
  (1+ (apply #'max 0
             (mapcar (lambda (id)
                       (alist-get 'sequence
                                  (imoogi-project-notes--parse-number-id id)))
                     (imoogi-project-notes--scope-number-ids level scope)))))

(defun imoogi-project-notes--next-number-id (level scope)
  "Return the next free numbering id in LEVEL and SCOPE."
  (imoogi-project-notes--format-number-id
   level scope (imoogi-project-notes--next-sequence level scope)))

(defun imoogi-project-notes--scope-number-ids-label (level scope)
  "Describe the numbering ids already used in LEVEL and SCOPE."
  (let ((ids (imoogi-project-notes--scope-number-ids level scope)))
    (format "기존 번호: %s" (if ids (mapconcat #'identity ids ", ") "없음"))))

(defun imoogi-project-notes--read-sequence (level scope)
  "Read a sequence for LEVEL and SCOPE, defaulting to the next free number."
  (read-number (format "새 순서 번호 (%s): "
                       (imoogi-project-notes--scope-number-ids-label
                        level scope))
               (imoogi-project-notes--next-sequence level scope)))

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
    (or (when-let* ((metadata-entry
                     (imoogi-project-notes--current-metadata-entry)))
          (let* ((note-id (imoogi-project-notes--entry-note-id metadata-entry))
                 (metadata-dir
                  (imoogi-project-notes--directory-file-name
                   (alist-get 'notes-dir metadata-entry)))
                 (candidates
                  (if note-id
                      (cl-remove-if-not
                       (lambda (candidate)
                         (string= note-id
                                  (imoogi-project-notes--entry-note-id
                                   candidate)))
                       entries)
                    entries)))
            (or (cl-find metadata-dir candidates
                         :key (lambda (candidate)
                                (imoogi-project-notes--directory-file-name
                                 (alist-get 'notes-dir candidate)))
                         :test #'string=)
                (and note-id (= (length candidates) 1) (car candidates))
                metadata-entry)))
        (and imoogi-project-notes-entry-instance-id
             (cl-find imoogi-project-notes-entry-instance-id entries
                      :key (lambda (entry) (alist-get 'instance-id entry))
                      :test #'string=))
        (and imoogi-project-notes-source-root
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
  (imoogi-project-notes--number-id-from-name name))

(defun imoogi-project-notes--numbered-notes-directory (number root &optional base)
  "Return a notes directory named from NUMBER and source ROOT.
The folder lives below BASE, or `imoogi-project-notes-directory' when nil."
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
                      (or base imoogi-project-notes-directory))))

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
  "Create FILE with CONTENT when absent, preserving any existing file."
  (when (file-directory-p file)
    (signal 'file-error (list "Project notes file path is a directory" file)))
  (when-let* ((buffer (find-buffer-visiting file)))
    (when (buffer-modified-p buffer)
      (user-error "먼저 저장하거나 버퍼를 닫으세요: %s" file)))
  (unless (file-exists-p file)
    (make-directory (file-name-directory file) t)
    (write-region content nil file nil 'silent)))

(defun imoogi-project-notes--exclusive-create-file (file content)
  "Create new FILE with CONTENT, failing when FILE already exists."
  (when (file-directory-p file)
    (signal 'file-error (list "Project notes file path is a directory" file)))
  (when-let* ((buffer (find-buffer-visiting file)))
    (when (buffer-modified-p buffer)
      (user-error "먼저 저장하거나 버퍼를 닫으세요: %s" file)))
  (when (file-exists-p file)
    (signal 'file-already-exists
            (list "Project notes file already exists" file)))
  (make-directory (file-name-directory file) t)
  (write-region content nil file nil 'silent nil 'excl))

(defun imoogi-project-notes--metadata-data (entry created-at)
  "Return folder metadata for ENTRY created at CREATED-AT."
  (imoogi-project-notes--entry-ensure-derived-fields entry 'create-note-id)
  (let* ((type (imoogi-project-notes--entry-type entry))
         (root (imoogi-project-notes--alist-string 'notes-dir entry))
         (relative (lambda (key)
                     (file-relative-name
                      (imoogi-project-notes--alist-string key entry) root))))
    `((schema_version . 1)
      (type . ,(symbol-name type))
      (key . ,(imoogi-project-notes--alist-string 'key entry))
      (name . ,(imoogi-project-notes--entry-name entry))
      (note_id . ,(imoogi-project-notes--entry-note-id entry 'create-note-id))
      ,@(when-let* ((number-id (imoogi-project-notes--alist-string
                                'number-id entry)))
          `((number_id . ,number-id)
            (number_level . ,(imoogi-project-notes--alist-string
                              'number-level entry))))
      (source_root_kind . ,(imoogi-project-notes--alist-string
                            'source-root-kind entry))
      (workspace_role . ,(symbol-name type))
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
                         'todo-storage entry))
      (artifact_preset . ,(imoogi-project-notes--artifact-preset-to-json
                           (imoogi-project-notes--entry-artifact-preset entry))))))

(defun imoogi-project-notes--metadata-merge-derived-fields (metadata expected)
  "Return METADATA with missing optional derived keys copied from EXPECTED."
  (let ((copy (copy-tree metadata))
        (changed nil))
    (dolist (key '(note_id number_id number_level source_root_kind workspace_role))
      (when (assq key expected)
        (let ((value (alist-get key expected)))
          (if (or (and (eq key 'note_id) (assq key copy))
                  (equal value (alist-get key copy)))
              nil
            (setf (alist-get key copy) value)
            (setq changed t)))))
    (cons copy changed)))

(defun imoogi-project-notes--rewrite-metadata-derived-fields (entry created-at)
  "Rewrite ENTRY's derived metadata fields while preserving unknown fields."
  (let* ((directory (imoogi-project-notes--alist-string 'notes-dir entry))
         (file (imoogi-project-notes--metadata-file directory))
         (expected (imoogi-project-notes--metadata-data entry created-at))
         (metadata (if (file-readable-p file)
                       (imoogi-project-notes--read-metadata-file file)
                     expected))
         (copy (copy-tree metadata)))
    (dolist (key '(number_id number_level source_root_kind workspace_role))
      (setq copy (assq-delete-all key copy))
      (when (assq key expected)
        (push (cons key (alist-get key expected)) copy)))
    (unless (assq 'note_id copy)
      (push (cons 'note_id (alist-get 'note_id expected)) copy))
    (let ((json-encoding-pretty-print t))
      (with-temp-file file
        (insert (json-encode copy) "\n")))))

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
                        file))
          (pcase-let ((`(,merged . ,changed)
                       (imoogi-project-notes--metadata-merge-derived-fields
                        actual expected)))
            (when changed
              (let ((json-encoding-pretty-print t))
                (with-temp-file file
                  (insert (json-encode merged) "\n"))))))
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
                                        &optional type name study-id artifact-preset)
  "Build a registry entry."
  (let* ((study-p (eq type 'study))
         (project-file (expand-file-name (if study-p "study.org" "project.org")
                                         notes-dir))
         (tasks-file (if (eq todo-storage 'central)
                         (imoogi-project-notes--safe-ensure-central-agenda)
                       (expand-file-name "tasks.org" notes-dir)))
         (journal-file (expand-file-name (if study-p "logs/journal.org" "journal.org")
                                         notes-dir))
         (entry
          `((key . ,key)
            (type . ,(symbol-name (or type 'project)))
            (name . ,(or name
                         (file-name-nondirectory
                          (directory-file-name source-root))))
            (note-id . nil)
            (number-id . nil)
            (number-level . nil)
            (source-root-kind . nil)
            (workspace-role . nil)
            ,@(when study-id `((study-id . ,study-id)))
            (source-root . ,(imoogi-project-notes--directory-file-name source-root))
            (notes-dir . ,(imoogi-project-notes--directory-file-name notes-dir))
            (project-file . ,project-file)
            (tasks-file . ,tasks-file)
            (journal-file . ,journal-file)
            (todo-storage . ,(symbol-name todo-storage))
            (artifact-preset . ,(or artifact-preset
                                     (imoogi-project-notes--default-artifact-preset
                                      (if study-p 'study 'development)))))))
    (imoogi-project-notes--entry-ensure-derived-fields entry 'create-note-id)))

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
  (imoogi-project-notes--entry-ensure-derived-fields entry 'create-note-id)
  (let* ((entries (imoogi-project-notes--read-registry))
         (key (alist-get 'key entry))
         (others (cl-remove key entries
                            :key (lambda (item) (alist-get 'key item))
                            :test #'string=)))
    (imoogi-project-notes--write-registry (cons entry others))))

;; @MX:NOTE Notes below a mounted root are discovered by scanning their
;; metadata, never through the local registry: `--all-entries' lets a local row
;; win over the same directory, which would hide the mounted origin from
;; detach/unmount.
(defun imoogi-project-notes--commit-new-entry (entry)
  "Persist newly set-up ENTRY and return the entry callers should use.
Local notes go to the registry.  Notes below a mounted root drop any stale
local row and are returned as rediscovered mounted entries."
  (let ((notes-dir (imoogi-project-notes--alist-string 'notes-dir entry)))
    (if (not (imoogi-project-notes--mounted-root-for-directory notes-dir))
        (progn (imoogi-project-notes--save-entry entry) entry)
      (imoogi-project-notes--remove-local-entry entry)
      (or (imoogi-project-notes--find-entry-by-notes-directory
           notes-dir (imoogi-project-notes--all-entries))
          entry))))

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
        (or (imoogi-project-notes--find-entry-by-key
             key (imoogi-project-notes--all-entries))
            (error "Project notes setup did not create a registry entry")))))

(defun imoogi-project-notes--restore-workspace (entry)
  "Restore Treemacs roots associated with ENTRY when available."
  (when (fboundp 'imoogi-treemacs-open-workspace-roots)
    (let* ((notes-root (imoogi-project-notes--alist-string 'notes-dir entry))
           (source-root (and (not (equal (alist-get 'source-root-kind entry)
                                         "notes"))
                             (imoogi-project-notes--current-source-root entry)))
           (roots (delq nil (list source-root notes-root)))
           (name (or (and (fboundp 'imoogi-project-perspective-name)
                          (imoogi-project-perspective-name
                           (or source-root notes-root)))
                     (imoogi-project-notes--alist-string 'name entry)
                     "project-notes")))
      (when roots
        (imoogi-treemacs-open-workspace-roots roots name)))))

;;;###autoload
(defun imoogi-project-notes-setup (&optional root directory numbering preset-kind)
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
                            (imoogi-project-notes--identity-key root)
                            (imoogi-project-notes--all-entries)))))
          (location (unless (or directory existing)
                      (imoogi-project-notes--read-notes-location
                       "프로젝트 노트 저장 위치: ")))
          (numbering (unless (or directory existing)
                       (read-string "프로젝트 번호 (예: 260925.01 또는 2609): ")))
          (directory (or directory
                         (and location
                              (imoogi-project-notes--numbered-notes-directory
                               numbering root
                               (imoogi-project-notes--alist-string
                                'path location)))))
          (preset-kind (unless existing
                         (intern (completing-read
                                  "프로젝트 산출물 preset: "
                                  '("development" "study") nil t nil nil
                                  "development")))))
     (list root directory numbering preset-kind)))
  (let* ((root (imoogi-project-notes--validate-source-root
                (or root (imoogi-project-notes--project-root))))
         (key (imoogi-project-notes--identity-key root))
         (entries (imoogi-project-notes--read-registry))
         (existing (or (imoogi-project-notes--find-entry-by-key key entries)
                       (imoogi-project-notes--find-entry-by-key
                        key (imoogi-project-notes--all-entries))))
         (project-name (file-name-nondirectory (directory-file-name root)))
         (notes-dir (imoogi-project-notes--validate-notes-directory
                     (or directory
                         (imoogi-project-notes--alist-string 'notes-dir existing)
                         (and numbering
                              (imoogi-project-notes--numbered-notes-directory
                               numbering root))
                         (imoogi-project-notes--default-notes-directory root key entries))
                     key entries))
         (metadata-note-id
          (when-let* ((metadata-file
                       (imoogi-project-notes--metadata-file notes-dir))
                      (metadata (and (file-readable-p metadata-file)
                                     (imoogi-project-notes--read-metadata-file
                                      metadata-file 'noerror))))
            (imoogi-project-notes--alist-string 'note_id metadata)))
         (mounted-root (imoogi-project-notes--mounted-root-for-directory
                        notes-dir))
         (storage (cond
                   (existing (imoogi-project-notes--entry-todo-storage existing))
                   ;; Mounted notes cannot use the central agenda file.
                   (mounted-root 'project)
                   ((member imoogi-project-notes-todo-storage '(project central))
                    imoogi-project-notes-todo-storage)
                   (t 'project)))
         (entry (imoogi-project-notes--entry
                 key root notes-dir storage 'project nil nil
                 (or (alist-get 'artifact-preset existing)
                     (imoogi-project-notes--default-artifact-preset
                      (or preset-kind 'development))))))
    (when-let ((note-id (or metadata-note-id
                            (imoogi-project-notes--entry-note-id existing))))
      (setf (alist-get 'note-id entry) note-id))
    (when (and mounted-root (not existing))
      (imoogi-project-notes--ensure-mounted-root-writable mounted-root))
    (imoogi-project-notes--setup-files entry project-name)
    (imoogi-project-notes--ensure-metadata entry (format-time-string "%Y-%m-%d"))
    (setq entry (imoogi-project-notes--commit-new-entry entry))
    (imoogi-project-notes--register-agenda-target
     (imoogi-project-notes--alist-string 'tasks-file entry))
    (imoogi-project-notes--restore-workspace entry)
    (message "imoogi: 작업 폴더 %s → 문서 폴더 %s" root notes-dir)
    notes-dir))

;; Treemacs struct slots are written through `cl-struct-slot-value'.  A plain
;; `(setf (treemacs-project->path ...))' only expands correctly when Treemacs
;; is loaded at compile or load time; otherwise it compiles into a call to the
;; nonexistent `(setf treemacs-project->path)' function.
;; Treemacs is not required at compile time, so declare its dynamic variable:
;; without this, the `let' below would be a lexical binding Treemacs never sees.
(defvar treemacs-override-workspace)

(defun imoogi-project-notes--set-treemacs-struct-slot
    (type slot object value)
  "Set Treemacs struct TYPE SLOT on OBJECT to VALUE at runtime."
  (let ((offset (cl-struct-slot-offset type slot)))
    (if (eq (cl-struct-sequence-type type) 'list)
        (setcar (nthcdr offset object) value)
      (aset object offset value))))

(defun imoogi-project-notes--replace-treemacs-root (old-root new-root)
  "Replace OLD-ROOT with NEW-ROOT in existing Treemacs workspaces.
Never remove a workspace or a project: every project whose path is OLD-ROOT
keeps its object, label and position, and only its path changes.  Return a
list of (WORKSPACE-NAME MESSAGE) failures, or nil on success.

Treemacs's own remove/add commands are deliberately not used.  They act on
the workspace shown in each open Treemacs window, not on the workspace being
repaired, so for any other workspace they either removed nothing (no window)
or failed with \"arrayp, nil\" (a window showing another workspace)."
  (when (and (fboundp 'treemacs-workspaces)
             (fboundp 'treemacs-workspace->projects)
             (fboundp 'treemacs-project->path))
    (let* ((old-path (directory-file-name (expand-file-name old-root)))
           (new-path (if (fboundp 'treemacs-canonical-path)
                         (treemacs-canonical-path new-root)
                       (directory-file-name (expand-file-name new-root))))
           (changed nil)
           (failures nil))
      (dolist (workspace (treemacs-workspaces))
        (dolist (project (treemacs-workspace->projects workspace))
            (when (equal old-path
                         (directory-file-name
                          (expand-file-name (treemacs-project->path project))))
            (imoogi-project-notes--set-treemacs-struct-slot
             'treemacs-project 'path project new-path)
            (cl-pushnew workspace changed :test #'eq))))
      (when changed
        ;; Open Treemacs windows still index projects by the old path, so a
        ;; window showing a changed workspace is rebuilt from the workspace.
        ;; Display trouble is reported but never undoes the saved path.
        (condition-case err
            (progn
              (when (fboundp 'treemacs--persist)
                (treemacs--persist))
              (when (and (fboundp 'treemacs--scope-store)
                         (fboundp 'treemacs-scope-shelf->buffer)
                         (fboundp 'treemacs-scope-shelf->workspace))
                (pcase-dolist (`(,_ . ,shelf) (treemacs--scope-store))
                  (let ((buffer (treemacs-scope-shelf->buffer shelf)))
                    (when (and (buffer-live-p buffer)
                               (memq (treemacs-scope-shelf->workspace shelf)
                                     changed))
                      ;; Same rebuild as `treemacs--consolidate-projects':
                      ;; reset the path index, then redraw every project.
                      ;; Expanded projects come back collapsed.
                      (with-current-buffer buffer
                        (let ((treemacs-override-workspace
                               (treemacs-scope-shelf->workspace shelf))
                              (inhibit-read-only t))
                          (treemacs--invalidate-buffer-project-cache)
                          (treemacs--reset-dom)
                          (erase-buffer)
                          (treemacs--render-projects
                           (treemacs-workspace->projects
                            treemacs-override-workspace))
                          (goto-char (point-min)))))))))
          (error
           (dolist (workspace changed)
             (push (list (treemacs-workspace->name workspace)
                         (error-message-string err))
                   failures))
           (display-warning
            'imoogi
            (format "Treemacs 프로젝트 경로를 갱신하지 못했습니다: %s"
                    (error-message-string err))
            :warning))))
      (nreverse failures))))

(defun imoogi-project-notes--rollback-rename
    (entry entry-before old-root new-root renamed buffer-files
           agenda-before agenda-storage-file-before perspective-before
           treemacs-before marker-file marker-created metadata-before
           metadata-before-file registry-file registry-before
           mounted-state-file mounted-state-before)
  "Restore all state captured before a project-notes directory rename."
  (when renamed
    (when (file-directory-p new-root)
      (rename-file new-root (directory-file-name old-root))))
  (dolist (buffer-file buffer-files)
    (when (buffer-live-p (car buffer-file))
      (with-current-buffer (car buffer-file)
        (when (buffer-file-name)
          (set-visited-file-name (cdr buffer-file) t t)))))
  (when entry
    (dolist (cell entry-before)
      (setf (alist-get (car cell) entry) (cdr cell))))
  (when (boundp 'org-agenda-files)
    (setq org-agenda-files agenda-before))
  (when (and (stringp org-agenda-files)
             agenda-storage-file-before)
    (with-temp-file org-agenda-files
      (insert agenda-storage-file-before)))
  (when (boundp 'imoogi-project-perspective-alist)
    (setq imoogi-project-perspective-alist perspective-before))
  (dolist (workspace-state treemacs-before)
    (let ((workspace (car workspace-state))
          (projects (cdr workspace-state)))
      (dolist (project-state projects)
        (imoogi-project-notes--set-treemacs-struct-slot
         'treemacs-project 'path (car project-state) (cdr project-state)))
      (imoogi-project-notes--set-treemacs-struct-slot
       'treemacs-workspace 'projects workspace (mapcar #'car projects))))
  (when (and treemacs-before (fboundp 'treemacs--persist))
    (treemacs--persist))
  (dolist (buffer-file buffer-files)
    (when (and (buffer-live-p (car buffer-file)) entry)
      (with-current-buffer (car buffer-file)
        (imoogi-project-notes--set-buffer-context entry))))
  (when (and marker-created (file-exists-p marker-file))
    (delete-file marker-file))
  (when (and metadata-before
             (file-directory-p old-root))
    (with-temp-file metadata-before-file
      (insert metadata-before)))
  (if registry-before
      (with-temp-file registry-file (insert registry-before))
    (when (file-exists-p registry-file)
      (delete-file registry-file)))
  (if mounted-state-before
      (with-temp-file mounted-state-file (insert mounted-state-before))
    (when (file-exists-p mounted-state-file)
      (delete-file mounted-state-file))))

(defun imoogi-project-notes--rename-entry-directory
    (entry old-root new-root &optional commit allow-pending-repair)
  "Rename ENTRY's notes folder and update local references.
The optional COMMIT function runs after the directory and in-memory paths are
updated.  Filesystem and metadata failures roll back; a Treemacs projection
failure is recorded for the doctor to replay after the rename."
  (let* ((old-root (file-name-as-directory (expand-file-name old-root)))
         (new-root (file-name-as-directory (expand-file-name new-root)))
         (old-tasks (and entry (imoogi-project-notes--alist-string
                                'tasks-file entry)))
         (entry-before (copy-tree entry))
         (agenda-before (and (boundp 'org-agenda-files)
                             (copy-tree org-agenda-files)))
         (agenda-storage-file-before
          (and (boundp 'org-agenda-files)
               (stringp org-agenda-files)
               (file-readable-p org-agenda-files)
               (with-temp-buffer
                 (insert-file-contents org-agenda-files)
                 (buffer-string))))
         (perspective-before (and (boundp 'imoogi-project-perspective-alist)
                                  (copy-tree imoogi-project-perspective-alist)))
         (treemacs-before
          (when (and (fboundp 'treemacs-workspaces)
                     (fboundp 'treemacs-workspace->projects)
                     (fboundp 'treemacs-project->path))
            (mapcar
             (lambda (workspace)
               (cons workspace
                     (mapcar (lambda (project)
                               (cons project (treemacs-project->path project)))
                             (treemacs-workspace->projects workspace))))
             (treemacs-workspaces))))
         (metadata-before-file (imoogi-project-notes--metadata-file old-root))
         (metadata-before
          (and (file-readable-p metadata-before-file)
               (with-temp-buffer
                 (insert-file-contents metadata-before-file)
                 (buffer-string))))
         (registry-file (imoogi-project-notes--registry-file))
         (registry-before
          (and (file-readable-p registry-file)
               (with-temp-buffer
                 (insert-file-contents registry-file)
                 (buffer-string))))
         (mounted-state-file (imoogi-project-notes--mounted-roots-file))
         (mounted-state-before
          (and (file-readable-p mounted-state-file)
               (with-temp-buffer
                 (insert-file-contents mounted-state-file)
                 (buffer-string))))
         (buffer-files nil)
         (renamed nil)
         (marker-file (imoogi-project-notes--repair-marker-file))
         (marker-created nil))
    (when (file-exists-p new-root)
      (user-error "새 프로젝트 폴더가 이미 존재합니다: %s" new-root))
    (when (and (not allow-pending-repair)
               (file-exists-p (imoogi-project-notes--repair-marker-file)))
      (user-error "보류 중인 project-notes 복구를 먼저 doctor로 처리하세요"))
    (when (and (fboundp 'imoogi-org--agenda-storage-buffer-modified-p)
               (imoogi-org--agenda-storage-buffer-modified-p))
      (user-error "먼저 agenda 파일 목록 버퍼를 저장하거나 닫으세요"))
    (dolist (buffer (buffer-list))
      (when-let* ((file (buffer-file-name buffer)))
        (when (and (file-in-directory-p file old-root)
                   (buffer-modified-p buffer))
          (user-error "먼저 저장하거나 닫아야 하는 변경 버퍼가 있습니다: %s" file))))
    (dolist (buffer (buffer-list))
      (when-let* ((file (buffer-file-name buffer)))
        (when (file-in-directory-p file old-root)
          (push (cons buffer file) buffer-files))))
    ;; A trailing slash makes `rename-file' treat the destination as an
    ;; existing directory and nest OLD-ROOT inside it.  Pass the directory
    ;; name without that marker, while retaining NEW-ROOT normalized below.
    (condition-case err
        (progn
          (rename-file old-root (directory-file-name new-root))
          (setq renamed t)
          (when entry
            (dolist (key '(project-file tasks-file journal-file source-root
                           effective-source-root))
              (let ((file (imoogi-project-notes--alist-string key entry)))
                (when (and file (file-in-directory-p file old-root))
                  (setf (alist-get key entry)
                        (expand-file-name
                         (file-relative-name file old-root) new-root)))))
            (setf (alist-get 'notes-dir entry) new-root))
          (when commit
            (funcall commit entry))
          ;; Session projections are updated after the durable commit.  The
          ;; Treemacs projection is the only one with a replayable doctor
          ;; repair path; other projection errors remain hard failures.
          (dolist (buffer-file buffer-files)
            (with-current-buffer (car buffer-file)
              (set-visited-file-name
               (expand-file-name
                (file-relative-name (cdr buffer-file) old-root) new-root)
               t t)
              (when entry
                (imoogi-project-notes--set-buffer-context entry))))
          (when (and old-tasks (file-in-directory-p old-tasks old-root))
            (imoogi-project-notes--replace-agenda-target
             old-tasks (alist-get 'tasks-file entry)))
          (when (boundp 'imoogi-project-perspective-alist)
            (let ((mapping (assoc old-root imoogi-project-perspective-alist)))
              (when mapping
                (setcar mapping new-root))))
          (let ((projection-failures
                 (imoogi-project-notes--replace-treemacs-root
                  old-root new-root)))
            (when projection-failures
              (imoogi-project-notes--write-repair-marker
               entry old-root new-root projection-failures)
              (setq marker-created t)))
          entry)
      (quit
       (imoogi-project-notes--rollback-rename
        entry entry-before old-root new-root renamed buffer-files
        agenda-before agenda-storage-file-before perspective-before
        treemacs-before marker-file marker-created metadata-before
        metadata-before-file registry-file registry-before
        mounted-state-file mounted-state-before)
       (signal (car err) (cdr err)))
      (error
       (imoogi-project-notes--rollback-rename
        entry entry-before old-root new-root renamed buffer-files
        agenda-before agenda-storage-file-before perspective-before
        treemacs-before marker-file marker-created metadata-before
        metadata-before-file registry-file registry-before
        mounted-state-file mounted-state-before)
       (signal (car err) (cdr err))))))

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

(defun imoogi-project-notes--edit-artifact-preset (entry)
  "Interactively edit and return ENTRY's copied artifact preset."
  (let ((preset (copy-tree (imoogi-project-notes--entry-artifact-preset entry))))
    (setq preset
          (mapcar
           (lambda (spec)
             (let ((prefix (read-string (format "%s 파일명 prefix: " (nth 0 spec))
                                        (nth 2 spec))))
               (unless (imoogi-project-notes--valid-artifact-prefix-p prefix)
                 (user-error "파일명 prefix는 파일명 문자만 사용할 수 있습니다: %s" prefix))
               (list (nth 0 spec)
                     (read-string (format "%s 표시 이름: " (nth 0 spec)) (nth 1 spec))
                     prefix
                     (read-string (format "%s 섹션 템플릿: " (nth 0 spec)) (nth 3 spec)))))
           preset))
    (while (y-or-n-p "새 산출물 종류를 추가할까요? ")
      (let ((kind (intern (read-string "종류 식별자: "))))
        (let ((prefix (read-string "파일명 prefix: ")))
          (unless (imoogi-project-notes--valid-artifact-prefix-p prefix)
            (user-error "파일명 prefix는 파일명 문자만 사용할 수 있습니다: %s" prefix))
          (setq preset
                (append preset
                        (list (list kind
                                    (read-string "표시 이름: ")
                                    prefix
                                    (read-string "섹션 템플릿: "))))))))
    (setf (alist-get 'artifact-preset entry) preset)
    entry))

(defun imoogi-project-notes--save-artifact-preset-metadata (entry)
  "Persist ENTRY's artifact preset without changing existing metadata."
  (let* ((directory (imoogi-project-notes--alist-string 'notes-dir entry))
         (file (imoogi-project-notes--metadata-file directory))
         (metadata (if (file-readable-p file)
                       (imoogi-project-notes--read-metadata-file file)
                     (imoogi-project-notes--metadata-data
                      entry (format-time-string "%Y-%m-%d")))))
    (setf (alist-get 'artifact_preset metadata)
          (imoogi-project-notes--artifact-preset-to-json
           (imoogi-project-notes--entry-artifact-preset entry)))
    (let ((json-encoding-pretty-print t))
      (with-temp-file file
        (insert (json-encode metadata) "\n")))))

(defun imoogi-project-notes--metadata-has-artifact-preset-p (entry)
  "Return non-nil when ENTRY metadata explicitly stores a preset."
  (let ((file (imoogi-project-notes--metadata-file
               (imoogi-project-notes--alist-string 'notes-dir entry))))
    (and (file-readable-p file)
         (alist-get 'artifact_preset
                    (imoogi-project-notes--read-metadata-file file)))))

;;;###autoload
(defun imoogi-project-notes-setup-doctor (&optional rename-valid)
  "Inspect project-notes folders and ask how to rename each invalid one.
Each answer is handled independently.  Empty input skips that folder.  The
folder is renamed as a whole and its registry, buffers, Perspective mapping,
and Treemacs roots are updated; files are never deleted or overwritten.
With a prefix argument, also offer repairs for folders that already match the
numbering grammar."
  (interactive "P")
  (let ((base (imoogi-project-notes--doctor-base-directory))
        (changed 0)
        (skipped 0)
        (current nil))
    (unless (file-directory-p base)
      (user-error "프로젝트 노트 폴더가 없습니다: %s" base))
    (imoogi-project-notes--doctor-log "doctor 시작: %s" base)
    ;; Any failure stops the doctor as before, but first records which folder
    ;; and answer caused it so the reason survives the echo area.
    (condition-case err
        (progn
          (imoogi-project-notes--consume-repair-marker)
          (dolist (directory (directory-files base t "^[^.].*" t))
            (when (file-directory-p directory)
              (let* ((name (file-name-nondirectory (directory-file-name directory)))
                     (valid (imoogi-project-notes--folder-name-valid-p name)))
                (when (or rename-valid (not valid))
                  (setq current (list name nil))
                  (let* ((parts (imoogi-project-notes--doctor-folder-parts name))
                         (raw (read-string
                               (format "새 번호/폴더명 [%s] (%s%s빈칸은 건너뜀): "
                                       name
                                       (if parts
                                           (concat
                                            (imoogi-project-notes--scope-number-ids-label
                                             (nth 0 parts) (nth 1 parts))
                                            ", '+'=다음 번호, 숫자=해당 번호, ")
                                         "")
                                       "")
                               ;; Numbered folders start empty so RET keeps
                               ;; them; unnumbered ones get a ready suggestion.
                               (unless valid
                                 (imoogi-project-notes--doctor-suggested-name name))))
                         (answer (progn
                                   (setq current (list name raw))
                                   (imoogi-project-notes--doctor-resolve-answer
                                    name raw))))
                    (if (null answer)
                        (progn
                          (setq skipped (1+ skipped))
                          (imoogi-project-notes--doctor-log "건너뜀: %s" name))
                      (setq current (list name raw answer))
                      (if (not (imoogi-project-notes--folder-name-valid-p answer))
                          (user-error "프로젝트 폴더명 형식이 올바르지 않습니다: %s" answer)
                        (let* ((old-root (file-name-as-directory directory))
                               (new-root (file-name-as-directory
                                          (expand-file-name answer base)))
                               (answer-id (imoogi-project-notes--folder-id-key answer))
                               (duplicate-id
                                (and answer-id
                                     (imoogi-project-notes--number-id-conflict
                                      answer-id old-root)))
                               (entry (or (imoogi-project-notes--find-entry-by-notes-directory old-root)
                                          (let ((metadata-file
                                                 (imoogi-project-notes--metadata-file old-root)))
                                            (when (file-readable-p metadata-file)
                                              (imoogi-project-notes--metadata-entry
                                               metadata-file
                                               (imoogi-project-notes--read-metadata-file metadata-file)))))))
                          (if duplicate-id
                              (user-error "ID %s가 이미 사용 중입니다: %s"
                                          answer-id
                                          (alist-get 'directory duplicate-id))
                            (when entry
                              ;; The doctor's answer is authoritative for repaired
                              ;; folder numbering; stale registry/metadata must not
                              ;; be written back over it.
                              (setf (alist-get 'number-id entry) answer-id
                                    (alist-get 'number-level entry)
                                    (symbol-name
                                     (alist-get 'number-level
                                                (imoogi-project-notes--parse-number-id
                                                 answer-id)))))
                            (imoogi-project-notes--rename-entry-directory
                             entry old-root new-root
                             (lambda (committed-entry)
                               (when committed-entry
                                 (imoogi-project-notes--rewrite-metadata-derived-fields
                                  committed-entry (format-time-string "%Y-%m-%d"))
                                 (imoogi-project-notes--save-entry committed-entry)))
                             t)
                            (setq changed (1+ changed))
                            (imoogi-project-notes--doctor-log
                             "변경: %s → %s" name answer))))))
                  (setq current nil))))))
      ((error quit)
       (imoogi-project-notes--doctor-log
        "실패: %s\n  폴더: %s\n  입력: %s\n  결과 이름: %s\n  에러: %S"
        (error-message-string err)
        (or (nth 0 current) "(폴더 처리 전)")
        (or (nth 1 current) "-")
        (or (nth 2 current) "-")
        err)
       (imoogi-project-notes--doctor-log
        "중단 — 그때까지 변경 %d개, 건너뜀 %d개" changed skipped)
       (unless noninteractive
         (display-buffer imoogi-project-notes--doctor-log-buffer))
       (signal (car err) (cdr err))))
    (imoogi-project-notes--doctor-log
     "완료 — 변경 %d개, 건너뜀 %d개" changed skipped)
    (message "imoogi: project-notes 점검 완료 — 변경 %d개, 건너뜀 %d개 (파일 삭제 없음, 기록: %s)"
             changed skipped imoogi-project-notes--doctor-log-buffer)
    (when (called-interactively-p 'interactive)
      (dolist (entry (imoogi-project-notes--read-registry))
        (unless (imoogi-project-notes--metadata-has-artifact-preset-p entry)
          (let ((kind (intern (completing-read
                               "기존 프로젝트 preset: "
                               '("development" "study") nil t nil nil
                               (if (eq (imoogi-project-notes--entry-type entry) 'study)
                                   "study"
                                 "development")))))
            (setf (alist-get 'artifact-preset entry)
                  (imoogi-project-notes--default-artifact-preset kind))
            (imoogi-project-notes--save-artifact-preset-metadata entry)
            (imoogi-project-notes--save-entry entry)))
        (when (y-or-n-p
               (format "프로젝트 %s의 산출물 preset을 수정할까요? "
                       (imoogi-project-notes--entry-name entry)))
          (imoogi-project-notes--edit-artifact-preset entry)
          (imoogi-project-notes--save-artifact-preset-metadata entry)
          (imoogi-project-notes--save-entry entry))))
    (list :changed changed :skipped skipped)))

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
           (imoogi-project-notes--alist-string
            'path (imoogi-project-notes--read-notes-location
                   "학습 노트 저장 위치: ")))
          (id (imoogi-project-notes--next-study-id nil mounted-root))
          (directory (when mounted-root
                       (expand-file-name
                        (format "%s-%s/" id (imoogi-project-notes--slug name))
                        mounted-root))))
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
         (metadata-note-id
          (when-let* ((metadata-file
                       (imoogi-project-notes--metadata-file notes-dir))
                      (metadata (and (file-readable-p metadata-file)
                                     (imoogi-project-notes--read-metadata-file
                                      metadata-file 'noerror))))
            (imoogi-project-notes--alist-string 'note_id metadata)))
         (entry (imoogi-project-notes--entry
                 key notes-dir notes-dir 'project 'study study-name study-id
                 (imoogi-project-notes--default-artifact-preset 'study))))
    (when-let ((note-id (or metadata-note-id
                            (imoogi-project-notes--entry-note-id existing))))
      (setf (alist-get 'note-id entry) note-id))
    (when-let* (((not existing))
                (mounted-root (imoogi-project-notes--mounted-root-for-directory
                               notes-dir)))
      (imoogi-project-notes--ensure-mounted-root-writable mounted-root))
    (imoogi-project-notes--setup-study-files entry study-name start-date)
    (imoogi-project-notes--ensure-metadata entry start-date)
    (setq entry (imoogi-project-notes--commit-new-entry entry))
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
     entry (imoogi-project-notes--alist-string 'project-file entry) source-root)
    (imoogi-project-notes--restore-workspace entry)))

;;;###autoload
(defun imoogi-project-notes-tasks ()
  "Open the current project's TODO source."
  (interactive)
  (let* ((entry (imoogi-project-notes--entry-or-setup))
         (source-root (imoogi-project-notes--current-source-root entry)))
    (imoogi-project-notes--find-file
     entry (imoogi-project-notes--alist-string 'tasks-file entry) source-root)
    (imoogi-project-notes--restore-workspace entry)))

;;;###autoload
(defun imoogi-project-notes-raiseup ()
  "Promote the current note directory to a broader numbering level.
The directory is renamed in place after duplicate and path preflight checks."
  (interactive)
  (let* ((entry (imoogi-project-notes--entry-or-setup))
         (old-root (imoogi-project-notes--alist-string 'notes-dir entry))
         (info (or (imoogi-project-notes--entry-number-info entry)
                   (user-error "현재 노트 폴더의 번호를 해석할 수 없습니다")))
         (level (alist-get 'number-level info))
         (levels (cdr (memq level imoogi-project-notes--number-level-order)))
         (target-level
          (progn
            (when (null levels)
              (user-error "이미 최상위 lifetime 수준입니다"))
            (intern (completing-read
                     "상위 번호 수준: "
                     (mapcar #'symbol-name levels) nil t))))
         (scope (pcase target-level
                  ('monthly (substring (alist-get 'scope info) 0 4))
                  ('yearly (substring (alist-get 'scope info) 0 2))
                  ('lifetime "L")
                  (_ (user-error "지원하지 않는 승격 수준입니다"))))
         (sequence (imoogi-project-notes--read-sequence target-level scope))
         (target-id (imoogi-project-notes--format-number-id
                     target-level scope sequence))
         (conflict (imoogi-project-notes--number-id-conflict
                    target-id old-root))
         (old-name (file-name-nondirectory (directory-file-name old-root)))
         (old-id (alist-get 'number-id info))
         (mounted-p (eq (alist-get 'origin entry) 'mounted))
         (mounted-root-id (alist-get 'mounted-root-id entry))
         (old-instance-id (alist-get 'instance-id entry))
         (slug (when (and old-id
                          (string-prefix-p old-id old-name))
                 (substring old-name (length old-id))))
         (new-name (concat target-id (or slug "")))
         (new-root (expand-file-name new-name
                                     (file-name-directory
                                      (directory-file-name old-root)))))
    (imoogi-project-notes--ensure-entry-mutable entry "상위 수준 승격")
    (when conflict
      (user-error "중복 번호 ID입니다: %s" target-id))
    (when (file-exists-p new-root)
      (user-error "새 프로젝트 폴더가 이미 존재합니다: %s" new-root))
    (setf (alist-get 'number-id entry) target-id
          (alist-get 'number-level entry) (symbol-name target-level))
    (imoogi-project-notes--rename-entry-directory
     entry old-root new-root
     (lambda (committed-entry)
       (imoogi-project-notes--ensure-metadata
        committed-entry (format-time-string "%Y-%m-%d"))
       (if mounted-p
           (let ((new-instance-id
                  (imoogi-project-notes--mounted-instance-id
                   mounted-root-id
                   (alist-get 'notes-dir committed-entry)))
                 (state (imoogi-project-notes--read-mounted-state)))
             (setf (alist-get 'instance-id committed-entry) new-instance-id)
             (let ((overrides (copy-tree (alist-get 'source-overrides state)))
                   (changed nil))
               (dolist (override overrides)
                 (when (equal (alist-get 'instance-id override) old-instance-id)
                   (setf (alist-get 'instance-id override) new-instance-id)
                   (setq changed t)))
               (when changed
                 (setf (alist-get 'source-overrides state) overrides)
                 (imoogi-project-notes--write-mounted-state state))))
         (imoogi-project-notes--save-entry committed-entry))))
    (message "프로젝트 노트를 %s 수준으로 승격했습니다: %s" target-level new-name)
    new-root))

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

(defun imoogi-project-notes--read-notes-location (prompt)
  "Ask where a new note should live with PROMPT; return a mounted root or nil.
Nil means the local `imoogi-project-notes-directory'.  Nothing is asked when
no enabled mounted root is currently attached."
  (let* ((roots (cl-remove-if
                 (lambda (root)
                   (or (eq (alist-get 'enabled root) :json-false)
                       (memq (imoogi-project-notes--mounted-root-status root)
                             '(disconnected foreign))))
                 (alist-get 'roots
                            (imoogi-project-notes--read-mounted-state 'noerror))))
         (local (format "로컬  %s" (abbreviate-file-name
                                   imoogi-project-notes-directory)))
         (candidates
          (cons (cons local nil)
                (mapcar
                 (lambda (root)
                   (cons (concat (imoogi-project-notes--root-label root)
                                 (if (eq (imoogi-project-notes--mounted-root-status
                                          root)
                                         'unverified)
                                     " [표식 없음]"
                                   ""))
                         root))
                 roots))))
    (when roots
      (when-let* ((root (cdr (assoc (completing-read prompt candidates nil t
                                                     nil nil local)
                                    candidates))))
        (imoogi-project-notes--ensure-mounted-root-writable root)))))

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
        (when (y-or-n-p
               (format "폴더를 생성할까요? %s%s " directory
                       (imoogi-project-notes--mount-point-note
                        (locate-dominating-file directory #'file-directory-p))))
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
    (when (and (eq (imoogi-project-notes--mounted-root-status entry) 'foreign)
               (not (and (called-interactively-p 'interactive)
                         (y-or-n-p
                          (format "다른 외장 루트 표식이 있습니다. 덮어쓸까요? %s "
                                  canonical)))))
      (user-error "다른 외장 루트의 표식이 있는 폴더입니다: %s" canonical))
    (imoogi-project-notes--write-mounted-root-marker entry)
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

(defun imoogi-project-notes--cache-command ()
  "Return the executable command for project-note cache operations."
  (imoogi-project-notes--move-command))

(defun imoogi-project-notes--cache-command-or-error ()
  "Return the project note CLI command or signal an actionable error."
  (or (imoogi-project-notes--cache-command)
      (user-error
       "imoogi-notes 실행 파일이 없습니다. 저장소에서 make build-notes를 실행하세요")))

(defun imoogi-project-notes--cache-available-p ()
  "Return non-nil when the project note CLI is available."
  (not (null (imoogi-project-notes--cache-command))))

(defun imoogi-project-notes--cache-dir (entry)
  "Return the cache directory for ENTRY."
  (when (and (stringp imoogi-project-notes-cache-directory)
             (> (length imoogi-project-notes-cache-directory) 0))
    (file-truename imoogi-project-notes-cache-directory)))

(defun imoogi-project-notes--cache-scope (entry)
  "Return the CLI cache scope for ENTRY."
  (let* ((notes-dir (file-truename
                     (imoogi-project-notes--alist-string 'notes-dir entry)))
         (tasks-file (file-truename
                      (imoogi-project-notes--alist-string 'tasks-file entry)))
         (project-id (imoogi-project-notes--entry-derived-note-id entry))
         (excluded-roots
          (cl-remove-if
           (lambda (dir)
             (or (string= dir notes-dir)
                 (not (file-in-directory-p dir notes-dir))))
           (mapcar (lambda (candidate)
                     (file-truename
                      (imoogi-project-notes--alist-string 'notes-dir candidate)))
                   (imoogi-project-notes--all-entries)))))
    `((project_id . ,project-id)
      (notes_root . ,notes-dir)
      (tasks_file . ,tasks-file)
      (excluded_roots . ,(vconcat excluded-roots)))))

(defun imoogi-project-notes--cache-buffer-overlay-p (entry buffer)
  "Return non-nil when BUFFER belongs to ENTRY's note scope."
  (when-let* ((file (buffer-file-name buffer)))
    (let ((notes-dir (imoogi-project-notes--alist-string 'notes-dir entry))
          (tasks-file (imoogi-project-notes--alist-string 'tasks-file entry)))
      (or (file-in-directory-p file notes-dir)
          (string= (file-truename file) (file-truename tasks-file))))))

(defun imoogi-project-notes--cache-overlays (entry)
  "Return live visiting buffer overlays for ENTRY."
  (let (overlays)
    (dolist (buffer (buffer-list) (nreverse overlays))
      (when (imoogi-project-notes--cache-buffer-overlay-p entry buffer)
        (with-current-buffer buffer
          (when buffer-file-name
            (push `((path . ,(file-truename buffer-file-name))
                    (text . ,(imoogi-project-notes--buffer-string buffer))
                    (revision . ,(number-to-string (buffer-modified-tick))))
                  overlays)))))))

(defun imoogi-project-notes--cache-request (entry operation &optional id)
  "Return an imoogi-notes cache request for ENTRY OPERATION."
  `((version . 1)
    (operation . ,operation)
    (scope . ,(imoogi-project-notes--cache-scope entry))
    (overlays . ,(vconcat (imoogi-project-notes--cache-overlays entry)))
    ,@(when-let* ((cache-dir (imoogi-project-notes--cache-dir entry)))
        `((cache_dir . ,cache-dir)))
    ,@(when id `((id . ,id)))))

(defun imoogi-project-notes--read-json-response-current-buffer ()
  "Read a JSON object from the current buffer as an alist."
  (goto-char (point-min))
  (let ((json-object-type 'alist)
        (json-array-type 'list)
        (json-key-type 'symbol)
        (json-false :json-false))
    (json-read)))

(defun imoogi-project-notes--cache-response-ok-p (response)
  "Return non-nil when RESPONSE is a successful cache response."
  (or (eq (alist-get 'ok response) t)
      (string= (alist-get 'status response) "ok")))

(defun imoogi-project-notes--cache-response-message (response)
  "Return a human-readable message from cache RESPONSE."
  (or (alist-get 'message response)
      (alist-get 'code response)
      (when-let* ((error (car (alist-get 'errors response))))
        (or (alist-get 'message error)
            (alist-get 'code error)))
      "unknown"))

(defun imoogi-project-notes--cache-call-sync (entry operation &optional id)
  "Synchronously call the note cache CLI for ENTRY OPERATION."
  (let* ((command (imoogi-project-notes--cache-command-or-error))
         (stderr-file (make-temp-file "imoogi-notes-cache-stderr-"))
         response status)
    (unwind-protect
        (with-temp-buffer
          (insert (json-encode
                   (imoogi-project-notes--cache-request entry operation id)))
          (setq status
                (apply #'call-process-region
                       (point-min) (point-max) (car command)
                       t (list t stderr-file) nil (cdr command)))
          (setq response (imoogi-project-notes--read-json-response-current-buffer))
          (unless (and (integerp status) (zerop status))
            (user-error "imoogi-notes cache 실행 실패: %s"
                        (string-trim
                         (with-temp-buffer
                           (insert-file-contents stderr-file)
                           (buffer-string)))))
          (unless (imoogi-project-notes--cache-response-ok-p response)
            (user-error "imoogi-notes cache 오류: %s"
                        (imoogi-project-notes--cache-response-message response)))
          response)
      (when (file-exists-p stderr-file)
        (delete-file stderr-file)))))

(defun imoogi-project-notes--kill-cache-process ()
  "Kill the pending note cache process, if any."
  (let ((process imoogi-project-notes--pending-cache-process))
    (setq imoogi-project-notes--pending-cache-process nil)
    (when (process-live-p process)
      (delete-process process))))

;;;###autoload
(defun imoogi-project-notes-cancel-cache-operation ()
  "Cancel the current asynchronous project note cache operation."
  (interactive)
  (imoogi-project-notes--kill-cache-process)
  (message "프로젝트 문서 캐시 요청을 취소했습니다"))

(defun imoogi-project-notes--cache-call-async (entry operation callback &optional id)
  "Call note cache CLI asynchronously for ENTRY OPERATION, then CALLBACK."
  (let* ((command (imoogi-project-notes--cache-command-or-error))
         (stdout-buffer (generate-new-buffer " *imoogi-notes-cache*"))
         (stderr-buffer (generate-new-buffer " *imoogi-notes-cache-stderr*"))
         (request (json-encode
                   (imoogi-project-notes--cache-request entry operation id)))
         process
         sentinel)
    (imoogi-project-notes--kill-cache-process)
    (setq sentinel
          (lambda (proc _event)
            (unless (process-live-p proc)
              (let ((current
                     (eq proc imoogi-project-notes--pending-cache-process)))
                (when current
                  (setq imoogi-project-notes--pending-cache-process nil))
                (unwind-protect
                    (when current
                      (if (and (eq (process-status proc) 'exit)
                               (zerop (process-exit-status proc)))
                          (with-current-buffer stdout-buffer
                            (let ((response
                                   (imoogi-project-notes--read-json-response-current-buffer)))
                              (if (imoogi-project-notes--cache-response-ok-p response)
                                  (funcall callback response)
                                (display-warning
                                 'imoogi
                                 (format "프로젝트 문서 캐시 오류: %s"
                                         (imoogi-project-notes--cache-response-message
                                          response))
                                 :warning))))
                        (display-warning
                         'imoogi
                         (format "프로젝트 문서 캐시 프로세스가 실패했습니다: %s"
                                 (process-exit-status proc))
                         :warning)))
                  (when (buffer-live-p stdout-buffer)
                    (kill-buffer stdout-buffer))
                  (when (buffer-live-p stderr-buffer)
                    (kill-buffer stderr-buffer)))))))
    (setq process
          (make-process
           :name "imoogi-notes-cache"
           :buffer stdout-buffer
           :command command
           :connection-type 'pipe
           :stderr stderr-buffer
           :noquery t))
    (setq imoogi-project-notes--pending-cache-process process)
    (set-process-sentinel process sentinel)
    (if (process-live-p process)
        (progn
          (process-send-string process request)
          (process-send-eof process))
      (funcall sentinel process "finished\n"))
    process))

(defun imoogi-project-notes--snapshot-file (file)
  "Return canonical FILE for snapshot comparison."
  (and (stringp file) (file-truename file)))

(defun imoogi-project-notes--document-file-eligible-p (entry file)
  "Return non-nil when FILE is an eligible project document for ENTRY."
  (let* ((truename (imoogi-project-notes--snapshot-file file))
         (root (file-truename
                (imoogi-project-notes--alist-string 'notes-dir entry)))
         (tasks-file (file-truename
                      (imoogi-project-notes--alist-string 'tasks-file entry)))
         (local-tasks (file-truename (expand-file-name "tasks.org" root)))
         (metadata-root
          (when truename
            (locate-dominating-file
             (file-name-directory truename)
             imoogi-project-notes--metadata-file-name)))
         (nested-roots
          (cl-remove-if
           (lambda (dir) (string= dir root))
           (mapcar (lambda (candidate)
                     (file-truename
                      (imoogi-project-notes--alist-string 'notes-dir candidate)))
                   (imoogi-project-notes--all-entries)))))
    (and truename
         (imoogi-project-notes--org-file-p truename)
         (file-in-directory-p truename root)
         (not (member truename (list tasks-file local-tasks)))
         (or (null metadata-root)
             (string= (file-truename metadata-root) root))
         (not (cl-some (lambda (nested-root)
                         (file-in-directory-p truename nested-root))
                       nested-roots))
         (not (imoogi-project-notes--hidden-path-p truename root))
         (not (string-match-p
               "\\`\\(?:assets\\|\\.git\\|\\.cache\\|vendor\\)/"
               (file-relative-name truename root))))))

(defun imoogi-project-notes--snapshot-buffer-document-p (entry file snapshot-files)
  "Return non-nil when FILE should be treated as a document in ENTRY."
  (let ((truename (imoogi-project-notes--snapshot-file file)))
    (and (imoogi-project-notes--document-file-eligible-p entry truename)
         (or (member truename snapshot-files)
             (find-buffer-visiting truename)))))

(defun imoogi-project-notes--buffer-id-occurrence-map ()
  "Return current buffer Org ID properties grouped by ID."
  (let (occurrences)
    (save-excursion
      (save-restriction
        (widen)
        (goto-char (point-min))
        (while (re-search-forward "^[[:space:]]*:ID:[[:space:]]+\\([^[:space:]\n]+\\)"
                                  nil t)
          (let ((id (match-string 1))
                (id-beg (match-beginning 1)))
            (when (imoogi-project-notes--node-property-at-position-p
                   "ID" id-beg)
              (push id-beg (alist-get id occurrences nil nil #'string=)))))))
    (mapcar (lambda (entry)
              (cons (car entry) (nreverse (cdr entry))))
            (nreverse occurrences))))

(defun imoogi-project-notes--snapshot-live-buffer-record
    (entry buffer snapshot-files)
  "Return tick-cached live BUFFER record for ENTRY."
  (when-let* ((file (buffer-file-name buffer))
              (truename (imoogi-project-notes--snapshot-file file)))
    (let* ((key (list (imoogi-project-notes--entry-derived-note-id entry)
                      (imoogi-project-notes--snapshot-file
                       (imoogi-project-notes--alist-string 'notes-dir entry))
                      (imoogi-project-notes--snapshot-file
                       (imoogi-project-notes--alist-string 'tasks-file entry))
                      truename
                      buffer))
           (tick (buffer-modified-tick buffer))
           (cached (gethash key imoogi-project-notes--snapshot-live-record-cache)))
      (if (and cached (= tick (alist-get 'tick cached)))
          (alist-get 'record cached)
        (let (record)
          (with-current-buffer buffer
            (when (derived-mode-p 'org-mode)
              (let* ((tasks-file
                      (file-truename
                       (imoogi-project-notes--alist-string 'tasks-file entry)))
                     (document-p
                      (imoogi-project-notes--snapshot-buffer-document-p
                       entry truename snapshot-files))
                     (tasks-p (string= truename tasks-file))
                     (doc
                      (and document-p
                           (condition-case err
                               (pcase-let ((`(,id ,title ,kind)
                                            (imoogi-project-notes--document-identity)))
                                 `((file . ,truename)
                                   ,@(when id `((id . ,id)))
                                   ,@(when title `((title . ,title)))
                                   ,@(when kind `((kind . ,kind)))))
                             (user-error
                              `((file . ,truename)
                                (kind . legacy-root)
                                (identity_error . ,(error-message-string err))))))))
                (when (or document-p tasks-p)
                  (setq record
                        `((file . ,truename)
                          (document . ,(or doc
                                           (and document-p `((file . ,truename)))))
                          (occurrences . ,(imoogi-project-notes--buffer-id-occurrence-map))))))))
          (puthash key `((tick . ,tick) (record . ,record))
                   imoogi-project-notes--snapshot-live-record-cache)
          record)))))

(defun imoogi-project-notes--snapshot-live-records (entry)
  "Return current live buffer records layered over the active snapshot."
  (let ((snapshot-files
         (mapcar (lambda (doc)
                   (imoogi-project-notes--snapshot-file (alist-get 'file doc)))
                 (append (alist-get 'documents
                                    imoogi-project-notes--document-cache-snapshot)
                         nil)))
        records)
    (dolist (buffer (buffer-list) (nreverse records))
      (when (imoogi-project-notes--cache-buffer-overlay-p entry buffer)
        (when-let* ((record
                     (imoogi-project-notes--snapshot-live-buffer-record
                      entry buffer snapshot-files)))
          (push record records))))))

(defun imoogi-project-notes--snapshot-documents (entry)
  "Return document entries from the active CLI snapshot plus live buffers."
  (let ((docs (copy-sequence
               (append (alist-get 'documents
                                  imoogi-project-notes--document-cache-snapshot)
                       nil)))
        (records (imoogi-project-notes--snapshot-live-records entry)))
    (dolist (record records)
      (when-let* ((doc (alist-get 'document record)))
        (let ((file (alist-get 'file record)))
          (setq docs
                (cl-remove file docs
                           :key (lambda (item)
                                  (imoogi-project-notes--snapshot-file
                                   (alist-get 'file item)))
                           :test #'string=))
          (push doc docs))))
    (nreverse docs)))

(defun imoogi-project-notes--snapshot-occurrences (entry id)
  "Return occurrence conses for ID from snapshot plus live buffers."
  (let* ((records (imoogi-project-notes--snapshot-live-records entry))
         (live-files (mapcar (lambda (record) (alist-get 'file record))
                             records))
         (occurrences
          (mapcar
           (lambda (occurrence)
             (cons (alist-get 'file occurrence)
                   (alist-get 'position occurrence)))
           (append (alist-get (intern id)
                              (alist-get 'occurrences
                                         imoogi-project-notes--document-cache-snapshot))
                   nil))))
    (setq occurrences
          (cl-remove-if
           (lambda (occurrence)
             (member (imoogi-project-notes--snapshot-file (car occurrence))
                     live-files))
           occurrences))
    (dolist (record records)
      (dolist (pos (alist-get id (alist-get 'occurrences record)
                              nil nil #'string=))
        (push (cons (alist-get 'file record) pos) occurrences)))
    (nreverse occurrences)))

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
    (imoogi-project-notes--ensure-mounted-root-writable root)
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

(defun imoogi-project-notes--artifact-spec (kind &optional entry)
  "Return the artifact template specification for KIND and ENTRY."
  (let ((spec (or (assq kind (imoogi-project-notes--entry-artifact-preset entry))
                  (assq kind imoogi-project-notes--artifact-types))))
    (if (and spec (imoogi-project-notes--valid-artifact-prefix-p (nth 2 spec)))
        spec
      (user-error "알 수 없는 산출물 종류 또는 잘못된 파일명 prefix: %s" kind))))

(defun imoogi-project-notes--artifact-kind-from-decision-tree (&optional entry)
  "Recommend an artifact kind by asking about the current work."
  (let ((preset (imoogi-project-notes--entry-artifact-preset entry)))
    (cond
     ((and (assq 'concept preset)
           (yes-or-no-p "개념을 설명하고 연결 관계를 정리하나요? "))
      'concept)
     ((and (assq 'flashcard preset)
           (yes-or-no-p "반복 암기할 앞면과 뒷면을 만드나요? "))
      'flashcard)
     ((and (assq 'question preset)
           (yes-or-no-p "문제와 풀이를 기록하나요? "))
      'question)
     ((and (assq 'runbook preset)
           (yes-or-no-p "반복해서 실행할 절차나 운영 작업인가요? "))
      'runbook)
     ((and (assq 'investigation preset)
           (yes-or-no-p "특정 사건의 원인이나 증거를 분석하나요? "))
      'investigation)
     ((and (assq 'spec preset)
           (yes-or-no-p "요구사항을 구현 가능한 계획과 완료 조건으로 만들까요? "))
      'spec)
     ((and (assq 'decision preset)
           (yes-or-no-p "대안 중 중요한 선택과 근거를 남기나요? "))
      'decision)
     ((and (assq 'meeting preset)
           (yes-or-no-p "회의 내용과 후속 작업을 기록하나요? "))
      'meeting)
     (t
      (let* ((choice (completing-read
                      "산출물 종류를 직접 선택: "
                      (mapcar (lambda (spec)
                                (cons (format "%s (%s)" (nth 1 spec) (nth 0 spec))
                                      (nth 0 spec)))
                              preset)
                      nil t))
             (selected (assoc choice
                              (mapcar (lambda (spec)
                                        (cons (format "%s (%s)" (nth 1 spec) (nth 0 spec))
                                              (nth 0 spec)))
                                      preset))))
        (or (cdr selected) (caar preset)))))))

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

(defun imoogi-project-notes--org-file-p (file)
  "Return non-nil when FILE is an Org file."
  (and (stringp file) (string-match-p "\\.org\\'" file)))

(defun imoogi-project-notes--hidden-path-p (file root)
  "Return non-nil when FILE below ROOT has a hidden path component."
  (cl-some (lambda (part)
             (and (> (length part) 0)
                  (eq (aref part 0) ?.)))
           (split-string (file-relative-name file root) "/" t)))

(defun imoogi-project-notes--project-document-files (entry)
  "Return project document Org files eligible for document linking."
  (if imoogi-project-notes--document-cache-snapshot
      (mapcar (lambda (doc) (alist-get 'file doc))
              (imoogi-project-notes--snapshot-documents entry))
    (let ((root (file-truename
                 (imoogi-project-notes--alist-string 'notes-dir entry))))
      (cl-remove-if-not
       (lambda (file)
         (imoogi-project-notes--document-file-eligible-p entry file))
       (directory-files-recursively root "\\.org\\'")))))

(defun imoogi-project-notes--buffer-string-hash (&optional buffer)
  "Return a stable hash for BUFFER's text."
  (with-current-buffer (or buffer (current-buffer))
    (secure-hash 'sha1 (buffer-substring-no-properties (point-min) (point-max)))))

(defun imoogi-project-notes--file-string-hash (file)
  "Return a stable hash for FILE's content."
  (with-temp-buffer
    (insert-file-contents file)
    (secure-hash 'sha1 (buffer-string))))

(defun imoogi-project-notes--file-string (file)
  "Return FILE's content as a string."
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(defun imoogi-project-notes--buffer-string (&optional buffer)
  "Return BUFFER's text without properties."
  (with-current-buffer (or buffer (current-buffer))
    (buffer-substring-no-properties (point-min) (point-max))))

(defun imoogi-project-notes--file-buffer (file)
  "Return an Org buffer visiting FILE."
  (let ((buffer (find-file-noselect file)))
    (with-current-buffer buffer
      (unless (derived-mode-p 'org-mode)
        (org-mode)))
    buffer))

(defun imoogi-project-notes--org-title ()
  "Return the current Org buffer title or first useful heading."
  (or (save-excursion
        (goto-char (point-min))
        (when (re-search-forward "^#\\+TITLE:[[:space:]]*\\(.+\\)$" nil t)
          (string-trim (match-string 1))))
      (save-excursion
        (goto-char (point-min))
        (when (re-search-forward org-heading-regexp nil t)
          (org-get-heading t t t t)))
      (file-name-base (or buffer-file-name (buffer-name)))))

(defun imoogi-project-notes--preamble-id-position ()
  "Return point at a file-level ID value in the Org preamble, or nil."
  (save-excursion
    (goto-char (point-min))
    (let ((limit (or (save-excursion
                       (re-search-forward org-heading-regexp nil t))
                     (point-max))))
      (catch 'found
        (while (re-search-forward "^[[:space:]]*:ID:[[:space:]]+\\([^[:space:]\n]+\\)"
                                  limit t)
          (let ((pos (match-beginning 1)))
            (when (imoogi-project-notes--node-property-at-position-p
                   "ID" pos)
              (throw 'found pos))))
        nil))))

(defun imoogi-project-notes--node-property-at-position-p (key pos)
  "Return non-nil when POS is inside an Org node property named KEY."
  (save-match-data
    (save-excursion
      (goto-char pos)
      (let ((context (org-element-context)))
        (and (eq (org-element-type context) 'node-property)
             (string= (org-element-property :key context) key))))))

(defun imoogi-project-notes--preamble-id ()
  "Return the file-level ID in the Org preamble, or nil."
  (when-let* ((pos (imoogi-project-notes--preamble-id-position)))
    (save-excursion
      (goto-char pos)
      (string-trim
       (buffer-substring-no-properties
        pos (line-end-position))))))

(defun imoogi-project-notes--top-level-id-candidates ()
  "Return recognized top-level legacy artifact ID candidates."
  (let (candidates)
    (org-with-wide-buffer
     (org-map-entries
      (lambda ()
        (when (and (= (org-outline-level) 1)
                   (not (string= (org-get-heading t t t t) "Link"))
                   (org-entry-get (point) "TYPE"))
          (when-let* ((id (org-entry-get (point) "ID")))
            (push (list id (org-get-heading t t t t) (point)) candidates))))
      nil 'file))
    (nreverse candidates)))

(defun imoogi-project-notes--document-identity (&optional create)
  "Return (ID TITLE KIND) for the current document.
When CREATE is non-nil, add a file-level ID if no unambiguous identity exists."
  (or (when-let* ((id (imoogi-project-notes--preamble-id)))
        (list id (imoogi-project-notes--org-title) 'file))
      (let ((candidates (imoogi-project-notes--top-level-id-candidates)))
        (cond
         ((= (length candidates) 1)
          (list (caar candidates) (cadar candidates) 'legacy-root))
         ((> (length candidates) 1)
          (user-error "문서 ID 후보가 여러 개라 선택할 수 없습니다: %s"
                      (or buffer-file-name (buffer-name))))
         (create
          (let ((id (imoogi-project-notes--new-note-id)))
            (save-excursion
              (goto-char (point-min))
              (insert ":PROPERTIES:\n:ID:       " id "\n:END:\n\n"))
            (list id (imoogi-project-notes--org-title) 'file)))))))

(defun imoogi-project-notes--document-info (file &optional create)
  "Return document info for FILE, creating a document ID when CREATE is non-nil."
  (with-current-buffer (imoogi-project-notes--file-buffer file)
    (pcase-let ((`(,id ,title ,kind)
                 (imoogi-project-notes--document-identity create)))
      `((file . ,(expand-file-name file))
        (id . ,id)
        (title . ,title)
        (kind . ,kind)))))

(defun imoogi-project-notes--document-catalog (entry &optional create)
  "Return current project document catalog for ENTRY.
When CREATE is non-nil, cataloging may assign IDs to opened candidates."
  (if (and imoogi-project-notes--document-cache-snapshot (not create))
      (imoogi-project-notes--snapshot-documents entry)
    (mapcar (lambda (file)
              (imoogi-project-notes--document-info file create))
            (imoogi-project-notes--project-document-files entry))))

(defun imoogi-project-notes--validate-document-target-file (entry file operation)
  "Return canonical FILE when it is a valid document target for ENTRY."
  (unless file
    (user-error "%s 대상 문서가 없습니다" operation))
  (let* ((expanded (expand-file-name file))
         (truename (file-truename expanded))
         (task-file (and buffer-file-name (file-truename buffer-file-name)))
         (snapshot imoogi-project-notes--document-cache-snapshot)
         (eligible (and (not snapshot)
                        (mapcar #'file-truename
                                (imoogi-project-notes--project-document-files
                                 entry)))))
    (when (and task-file (string= truename task-file))
      (user-error "%s 대상이 현재 작업 파일과 같습니다: %s" operation file))
    (unless (if snapshot
                (and (file-exists-p expanded)
                     (imoogi-project-notes--document-file-eligible-p
                      entry expanded))
              (member truename eligible))
      (user-error "%s 대상은 현재 프로젝트 문서여야 합니다: %s" operation file))
    truename))

(defun imoogi-project-notes--buffer-id-occurrences (id)
  "Return positions in the current Org buffer with Org ID property ID."
  (let (positions)
    (save-excursion
      (save-restriction
        (widen)
        (goto-char (point-min))
        (while (re-search-forward "^[[:space:]]*:ID:[[:space:]]+\\([^[:space:]\n]+\\)"
                                  nil t)
          (let ((id-beg (match-beginning 1)))
            (when (and (string= (match-string 1) id)
                       (imoogi-project-notes--node-property-at-position-p
                        "ID" id-beg))
              (push id-beg positions))))))
    (nreverse positions)))

(defun imoogi-project-notes--file-id-occurrences (file id)
  "Return positions where FILE contains Org ID property ID.
Use a live visiting buffer when available so unsaved project buffers are not
classified from stale disk content."
  (when (and file (file-exists-p file))
    (if-let* ((buffer (find-buffer-visiting file)))
        (with-current-buffer buffer
          (imoogi-project-notes--buffer-id-occurrences id))
      (with-temp-buffer
        (insert-file-contents file)
        (org-mode)
        (imoogi-project-notes--buffer-id-occurrences id)))))

(defun imoogi-project-notes--project-id-occurrences (entry id)
  "Return project-owned FILE/POS occurrences for ID."
  (if imoogi-project-notes--document-cache-snapshot
      (imoogi-project-notes--snapshot-occurrences entry id)
    (let ((files (delete-dups
                  (delq nil
                        (cons (imoogi-project-notes--alist-string 'tasks-file entry)
                              (imoogi-project-notes--project-document-files entry)))))
          occurrences)
      (dolist (file files)
        (dolist (pos (imoogi-project-notes--file-id-occurrences file id))
          (push (cons file pos) occurrences)))
      (nreverse occurrences))))

(defun imoogi-project-notes--document-by-id (entry id)
  "Find exactly one current-project document with ID."
  (let* ((occurrences (imoogi-project-notes--project-id-occurrences entry id))
         (matches
          (cl-remove-if-not
           (lambda (doc) (string= id (alist-get 'id doc)))
           (imoogi-project-notes--document-catalog entry))))
    (cond
     ((> (length occurrences) 1)
      (user-error "프로젝트 안에서 ID가 중복됩니다: %s" id))
     ((= (length matches) 1) (car matches))
     ((> (length matches) 1)
      (user-error "프로젝트 문서 ID가 중복됩니다: %s" id))
     ((= (length occurrences) 1)
      (user-error "ID가 문서가 아닌 프로젝트 항목을 가리킵니다: %s" id))
     (t nil))))

(defun imoogi-project-notes--select-document (entry prompt &optional _require-id)
  "Read a project document from ENTRY using PROMPT."
  (let* ((root (imoogi-project-notes--alist-string 'notes-dir entry))
         (docs (imoogi-project-notes--document-catalog entry))
         (choices
          (mapcar (lambda (doc)
                    (cons (format "%s — %s"
                                  (or (alist-get 'title doc)
                                      (file-name-base (alist-get 'file doc)))
                                  (file-relative-name (alist-get 'file doc) root))
                          doc))
                  docs))
         (choice (completing-read prompt choices nil t)))
    (cdr (assoc choice choices))))

(defun imoogi-project-notes--org-id-link-regexp (id)
  "Return a regexp matching an Org ID link to ID."
  (concat "\\[\\[id:" (regexp-quote id) "\\]\\[[^]\n]+\\]\\]"))

(defun imoogi-project-notes--current-heading-direct-end ()
  "Return end of the current heading's direct body before child headings."
  (save-excursion
    (org-back-to-heading t)
    (or (save-excursion
          (forward-line 1)
          (when (re-search-forward org-heading-regexp nil t)
            (match-beginning 0)))
        (point-max))))

(defun imoogi-project-notes--task-artifact-heading-line-p ()
  "Return non-nil when point is a direct task artifact section marker."
  (save-excursion
    (beginning-of-line)
    (and (looking-at-p "^산출물:[[:space:]]*$")
         (let* ((heading (save-excursion
                           (org-back-to-heading t)
                           (point)))
                (datum (org-element-context))
                (accepted nil)
                (blocked nil))
           (while datum
             (let ((type (org-element-type datum)))
               (cond
                ((memq type '(src-block example-block comment comment-block
                                        keyword quote-block special-block
                                        center-block verse-block drawer
                                        property-drawer plain-list item table))
                 (setq blocked t
                       datum nil))
                ((eq type 'section)
                 (let ((parent (org-element-property :parent datum)))
                   (setq accepted
                         (and (eq (org-element-type parent) 'headline)
                              (= (org-element-property :begin parent) heading))
                         datum nil)))
                (t
                 (setq datum (org-element-property :parent datum))))))
           (and accepted (not blocked))))))

(defun imoogi-project-notes--task-artifact-section-bounds (subtree-end)
  "Return (BEG END) for the current task's artifact list, or nil."
  (save-excursion
    (org-back-to-heading t)
    (forward-line 1)
    (catch 'bounds
      (while (re-search-forward "^산출물:[[:space:]]*$" subtree-end t)
        (beginning-of-line)
        (when (imoogi-project-notes--task-artifact-heading-line-p)
          (forward-line 1)
          (let ((beg (point)))
            (while (and (< (point) subtree-end)
                        (looking-at "[[:space:]]*- "))
              (forward-line 1))
            (throw 'bounds (list beg (point)))))
        (forward-line 1))
      nil)))

(defun imoogi-project-notes--task-artifact-link-present-p (doc-id subtree-end)
  "Return non-nil when current task subtree already links DOC-ID."
  (when-let* ((bounds (imoogi-project-notes--task-artifact-section-bounds
                       subtree-end)))
    (pcase-let ((`(,beg ,end) bounds))
      (save-excursion
        (goto-char beg)
        (re-search-forward (imoogi-project-notes--org-id-link-regexp doc-id)
                           end t)))))

(defun imoogi-project-notes--ensure-task-artifact-link (doc-id title)
  "Ensure current task has one artifact link to DOC-ID named TITLE."
  (let ((link (format "- [[id:%s][%s]]" doc-id title))
        (subtree-end (imoogi-project-notes--current-heading-direct-end)))
    (unless (imoogi-project-notes--task-artifact-link-present-p
             doc-id subtree-end)
      (save-excursion
        (if-let* ((bounds (imoogi-project-notes--task-artifact-section-bounds
                           subtree-end)))
            (goto-char (cadr bounds))
          (goto-char subtree-end)
          (unless (bolp) (insert "
"))
          (insert "
산출물:
"))
        (insert link "
")))))

(defun imoogi-project-notes--remove-id-link-lines-in-region (id beg end)
  "Remove relation lines linking to ID between BEG and END.
Relation sections historically allowed either list items or a bare ID link line."
  (save-excursion
    (goto-char beg)
    (while (re-search-forward (imoogi-project-notes--org-id-link-regexp id)
                              end t)
      (let* ((match-beg (match-beginning 0))
             (match-end (match-end 0))
             (line-beg (line-beginning-position))
             (line-end (line-end-position))
             (line-limit (min (1+ line-end) (point-max)))
             (relation-line
              (save-excursion
                (goto-char line-beg)
                (imoogi-project-notes--managed-id-link-line-p)))
             (remaining
              (string-trim
               (concat
                (buffer-substring-no-properties line-beg match-beg)
                (buffer-substring-no-properties match-end line-end)))))
        (when relation-line
          (if (or (string-empty-p remaining)
                  (string= remaining "-"))
              (progn
                (delete-region line-beg line-limit)
                (setq end (- end (- line-limit line-beg))))
            (delete-region match-beg match-end)
            (setq end (- end (- match-end match-beg)))))))))

(defun imoogi-project-notes--remove-task-artifact-link (doc-id)
  "Remove current task artifact link to DOC-ID."
  (let ((subtree-end (imoogi-project-notes--current-heading-direct-end)))
    (when-let* ((bounds (imoogi-project-notes--task-artifact-section-bounds
                         subtree-end)))
      (pcase-let ((`(,beg ,end) bounds))
        (imoogi-project-notes--remove-id-link-lines-in-region doc-id beg end)))))

(defun imoogi-project-notes--link-heading-bounds (&optional create)
  "Return (BEG END) for the top-level Link heading in current buffer.
When CREATE is non-nil, append the heading when absent."
  (let (bounds)
    (org-with-wide-buffer
     (goto-char (point-min))
     (while (re-search-forward "^\\* Link[[:space:]]*$" nil t)
       (push (copy-marker (match-beginning 0)) bounds)))
    (cond
     ((> (length bounds) 1)
      (user-error "Link heading이 여러 개라 자동 수정할 수 없습니다: %s"
                  (or buffer-file-name (buffer-name))))
     ((= (length bounds) 1)
      (save-excursion
        (goto-char (car bounds))
        (list (point) (imoogi-project-notes--current-heading-direct-end))))
     (create
      (goto-char (point-max))
      (unless (bolp) (insert "\n"))
      (insert "\n* Link\n")
      (forward-line -1)
      (list (point) (point-max))))))

(defun imoogi-project-notes--legacy-related-bounds ()
  "Return bounds for all legacy related-task headings in current document."
  (let (bounds)
    (org-with-wide-buffer
     (goto-char (point-min))
     (while (re-search-forward "^\\*\\{2,\\} 관련 작업[[:space:]]*$" nil t)
       (push (save-excursion
               (goto-char (match-beginning 0))
               (list (point) (imoogi-project-notes--current-heading-direct-end)))
             bounds)))
    bounds))

(defun imoogi-project-notes--managed-id-link-present-in-region-p (id beg end)
  "Return non-nil when managed relation lines in BEG END link to ID."
  (save-excursion
    (goto-char beg)
    (catch 'found
      (while (< (point) end)
        (let ((line-end (line-end-position)))
          (when (and (imoogi-project-notes--managed-id-link-line-p)
                     (re-search-forward
                      (imoogi-project-notes--org-id-link-regexp id)
                      line-end t))
            (throw 'found t)))
        (forward-line 1))
      nil)))

(defun imoogi-project-notes--document-task-link-present-p (task-id)
  "Return non-nil when current document already links TASK-ID as a relation."
  (or (pcase-let ((`(,beg ,end) (imoogi-project-notes--link-heading-bounds)))
        (and beg
             (imoogi-project-notes--managed-id-link-present-in-region-p
              task-id beg end)))
      (cl-some
       (lambda (bounds)
         (pcase-let ((`(,beg ,end) bounds))
           (imoogi-project-notes--managed-id-link-present-in-region-p
            task-id beg end)))
       (imoogi-project-notes--legacy-related-bounds))))

(defun imoogi-project-notes--preflight-document-relation-target ()
  "Validate current document relation sections before any mutation."
  (imoogi-project-notes--link-heading-bounds))

(defun imoogi-project-notes--ensure-existing-task-id-valid (entry)
  "Reject an existing current task ID that is not unique in ENTRY scope."
  (when-let* ((task-id (org-entry-get (point) "ID")))
    (let ((occurrences
           (imoogi-project-notes--project-id-occurrences entry task-id)))
      (when (> (length occurrences) 1)
        (user-error "프로젝트 안에서 작업 ID가 중복됩니다: %s" task-id)))))

(defun imoogi-project-notes--validate-document-id-file (entry doc-id doc-file)
  "Return canonical DOC-FILE after checking it is DOC-ID's project document.
Missing files are allowed so unlink can offer a source-only removal.  Existing
files must be eligible documents in ENTRY and must carry DOC-ID as their
document identity."
  (unless (and (stringp doc-id) (> (length doc-id) 0))
    (user-error "연결 해제할 문서 ID가 없습니다"))
  (when doc-file
    (setq doc-file (expand-file-name doc-file))
    (when (file-exists-p doc-file)
      (setq doc-file
            (imoogi-project-notes--validate-document-target-file
             entry doc-file "산출물 연결 해제"))
      (let* ((canonical (file-truename doc-file))
             (known (imoogi-project-notes--document-by-id entry doc-id)))
        (when (and known
                   (not (string= canonical
                                 (file-truename (alist-get 'file known)))))
          (user-error "문서 ID와 파일이 서로 다른 문서를 가리킵니다: %s" doc-id)))
      (with-current-buffer (imoogi-project-notes--file-buffer doc-file)
        (pcase-let ((`(,actual-id ,_title ,_kind)
                     (or (imoogi-project-notes--document-identity)
                         (list nil nil nil))))
          (unless actual-id
            (user-error "문서 파일에 ID가 없어 연결 해제 대상으로 사용할 수 없습니다: %s"
                        doc-file))
          (unless (string= actual-id doc-id)
            (user-error "문서 파일의 ID가 요청한 ID와 다릅니다: %s" doc-id))))
    doc-file)))

(defun imoogi-project-notes--ensure-document-task-link (task-id task-title)
  "Ensure current document has a top-level Link relation to TASK-ID."
  (unless (imoogi-project-notes--document-task-link-present-p task-id)
    (pcase-let ((`(,beg ,end) (imoogi-project-notes--link-heading-bounds 'create)))
      (goto-char end)
      (unless (bolp) (insert "\n"))
      (insert (format "- [[id:%s][%s]]\n" task-id task-title)))))

(defun imoogi-project-notes--remove-document-task-link (task-id)
  "Remove relation links to TASK-ID from current document Link/legacy sections."
  (when-let* ((bounds (imoogi-project-notes--link-heading-bounds)))
    (pcase-let ((`(,beg ,end) bounds))
      (imoogi-project-notes--remove-id-link-lines-in-region task-id beg end)))
  (dolist (bounds (imoogi-project-notes--legacy-related-bounds))
    (pcase-let ((`(,beg ,end) bounds))
      (imoogi-project-notes--remove-id-link-lines-in-region task-id beg end))))

(defun imoogi-project-notes--confirm-source-only-unlink (doc-id)
  "Confirm unlinking only the task-side relation for missing DOC-ID."
  (let ((choice
         (if (and noninteractive
                  (subrp (symbol-function 'completing-read)))
             "remove"
           (completing-read
            (format "대상 문서 %s를 찾지 못했습니다. 작업 쪽 링크만 제거할까요? "
                    doc-id)
            '("remove" "cancel") nil t nil nil "cancel"))))
    (unless (string= choice "remove")
      (user-error "대상 문서를 찾지 못해 링크 해제를 취소했습니다: %s" doc-id))))

(defun imoogi-project-notes--classify-save-buffer (buffer before-hash)
  "Save BUFFER and return a status symbol after checking disk content."
  (with-current-buffer buffer
    (let ((file buffer-file-name)
          (post-hash (imoogi-project-notes--buffer-string-hash buffer)))
      (condition-case err
          (progn
            (save-buffer)
            (if (and file (file-exists-p file)
                     (string= (imoogi-project-notes--file-string-hash file)
                              post-hash))
                'saved
              (signal 'file-error (list "Save returned but disk differed" file))))
        (error
         (cond
          ((and file (file-exists-p file)
                (string= (imoogi-project-notes--file-string-hash file)
                         post-hash))
           (display-warning 'imoogi
                            (format "저장 후 hook 오류가 있었지만 파일은 기록됨: %s (%s)"
                                    file (error-message-string err))
                            :warning)
           'saved-after-hook-error)
          ((and file (file-exists-p file)
                (string= (imoogi-project-notes--file-string-hash file)
                         before-hash))
           (signal (car err) (cdr err)))
          (t
           (signal (car err) (cdr err)))))))))

(defun imoogi-project-notes--ensure-peer-buffer-clean (buffer operation)
  "Abort OPERATION when BUFFER has unrelated unsaved edits."
  (when (buffer-modified-p buffer)
    (let* ((file (buffer-file-name buffer))
           (pending-files
            (and imoogi-project-notes--pending-link-operation
                 (delq nil
                       (list (alist-get 'doc-file
                                        imoogi-project-notes--pending-link-operation)
                             (alist-get 'old-doc-file
                                        imoogi-project-notes--pending-link-operation)
                             (alist-get 'new-doc-file
                                        imoogi-project-notes--pending-link-operation))))))
      (unless (and file (member file pending-files))
        (user-error "%s 전에 먼저 저장하거나 버퍼를 정리하세요: %s"
                    operation (or file (buffer-name)))))))

(defun imoogi-project-notes--endpoint-preimage (role buffer operation)
  "Return clean preimage metadata for ROLE BUFFER before OPERATION mutates it."
  (when (buffer-modified-p buffer)
    (user-error "%s 전에 먼저 저장하거나 버퍼를 정리하세요: %s"
                operation (or (buffer-file-name buffer) (buffer-name buffer))))
  (let ((file (buffer-file-name buffer)))
    (unless (and file (file-exists-p file))
      (user-error "%s 대상 파일을 찾지 못했습니다: %s"
                  operation (or file (buffer-name buffer))))
    (let ((file-hash (imoogi-project-notes--file-string-hash file))
          (buffer-hash (imoogi-project-notes--buffer-string-hash buffer)))
      (unless (string= file-hash buffer-hash)
        (user-error "%s 전에 파일을 다시 방문하거나 되돌리세요: %s"
                    operation file))
      (imoogi-project-notes--ensure-endpoint-writable
       `((role . ,role) (file . ,file)) operation)
      `((role . ,role)
        (file . ,file)
        (pre-file-hash . ,file-hash)
        (pre-buffer-hash . ,buffer-hash)
        (pre-text . ,(imoogi-project-notes--buffer-string buffer))))))

(defun imoogi-project-notes--endpoint-buffer (endpoint)
  "Return a live buffer for ENDPOINT's file."
  (imoogi-project-notes--file-buffer (alist-get 'file endpoint)))

(defun imoogi-project-notes--ensure-endpoint-writable (endpoint operation)
  "Signal unless ENDPOINT can be mutated for OPERATION."
  (let* ((file (alist-get 'file endpoint))
         (buffer (and file (find-buffer-visiting file))))
    (unless (and file (file-exists-p file))
      (user-error "%s 대상 파일을 찾지 못했습니다: %s" operation file))
    (unless (file-writable-p file)
      (user-error "%s 대상 파일을 쓸 수 없습니다: %s" operation file))
    (when (and buffer (buffer-local-value 'buffer-read-only buffer))
      (user-error "%s 대상 버퍼가 읽기 전용입니다: %s" operation file))))

(defun imoogi-project-notes--restore-endpoint-preimage-for-preparation
    (endpoint)
  "Restore ENDPOINT after a preparation failure when disk is unchanged."
  (when (and (eq (imoogi-project-notes--endpoint-disk-state endpoint) 'pre)
             (memq (imoogi-project-notes--endpoint-buffer-state endpoint)
                   '(post changed)))
    (let ((buffer (imoogi-project-notes--endpoint-buffer endpoint)))
      (imoogi-project-notes--ensure-endpoint-writable
       endpoint "산출물 링크 준비 복구")
      (with-current-buffer buffer
        (erase-buffer)
        (insert (alist-get 'pre-text endpoint))
        (set-buffer-modified-p nil)))))

(defmacro imoogi-project-notes--with-link-operation-preparation
    (endpoints &rest body)
  "Run BODY, restoring ENDPOINTS when preparation fails before commit.
ENDPOINTS must be preimage alists captured before any mutation.  This wrapper
does not create pending retry metadata; pending state is owned by
`imoogi-project-notes--run-link-operation' after BODY completes."
  (declare (indent 1) (debug t))
  `(let ((imoogi-project-notes--preparation-endpoints ,endpoints))
     (condition-case err
         (progn ,@body)
       (error
        (dolist (endpoint imoogi-project-notes--preparation-endpoints)
          (imoogi-project-notes--restore-endpoint-preimage-for-preparation
           endpoint))
        (setq imoogi-project-notes--pending-link-operation nil)
        (signal (car err) (cdr err))))))

(defun imoogi-project-notes--endpoint-disk-state (endpoint)
  "Return ENDPOINT disk state compared with pre/post hashes."
  (let ((file (alist-get 'file endpoint)))
    (cond
     ((not (and file (file-exists-p file))) 'missing)
     ((string= (imoogi-project-notes--file-string-hash file)
               (alist-get 'post-buffer-hash endpoint))
      'post)
     ((string= (imoogi-project-notes--file-string-hash file)
               (alist-get 'pre-file-hash endpoint))
      'pre)
     (t 'changed))))

(defun imoogi-project-notes--endpoint-buffer-state (endpoint)
  "Return ENDPOINT buffer state compared with pre/post hashes."
  (let* ((file (alist-get 'file endpoint))
         (buffer (and file (find-buffer-visiting file)))
         (hash (and buffer
                    (imoogi-project-notes--buffer-string-hash buffer))))
    (cond
     ((not buffer) 'absent)
     ((string= hash (alist-get 'post-buffer-hash endpoint)) 'post)
     ((string= hash (alist-get 'pre-buffer-hash endpoint)) 'pre)
     (t 'changed))))

(defun imoogi-project-notes--endpoint-with-postimage (endpoint)
  "Return ENDPOINT extended with its planned post-edit buffer state."
  (let* ((buffer (imoogi-project-notes--endpoint-buffer endpoint))
         (post-text (imoogi-project-notes--buffer-string buffer))
         (post-hash (secure-hash 'sha1 post-text)))
    (append `((post-buffer-hash . ,post-hash)
              (post-text . ,post-text))
            endpoint)))

(defun imoogi-project-notes--restore-endpoint-preimage-if-unsaved (endpoint)
  "Restore ENDPOINT buffer when its planned edit did not reach disk."
  (when (and (eq (imoogi-project-notes--endpoint-disk-state endpoint) 'pre)
             (eq (imoogi-project-notes--endpoint-buffer-state endpoint) 'post))
    (let ((buffer (imoogi-project-notes--endpoint-buffer endpoint))
          (pre-text (alist-get 'pre-text endpoint)))
      (imoogi-project-notes--ensure-endpoint-writable
       endpoint "산출물 링크 저장 복구")
      (with-current-buffer buffer
        (erase-buffer)
        (insert pre-text)
        (set-buffer-modified-p nil)))))

(defun imoogi-project-notes--endpoint-retry-ready-p (endpoint)
  "Return non-nil when ENDPOINT is in an accepted retry state."
  (let ((disk-state (imoogi-project-notes--endpoint-disk-state endpoint))
        (buffer-state (imoogi-project-notes--endpoint-buffer-state endpoint)))
    (pcase disk-state
      ('pre (memq buffer-state '(absent pre post)))
      ('post (memq buffer-state '(absent post)))
      (_ nil))))

(defun imoogi-project-notes--prepare-endpoint-for-retry (endpoint)
  "Restore ENDPOINT's exact planned postimage when retry still needs saving."
  (unless (imoogi-project-notes--endpoint-retry-ready-p endpoint)
    (user-error "링크 재시도 대상이 변경되어 자동 재시도할 수 없습니다: %s"
                (alist-get 'file endpoint)))
  (when (eq (imoogi-project-notes--endpoint-disk-state endpoint) 'pre)
    (let ((buffer (imoogi-project-notes--endpoint-buffer endpoint)))
      (imoogi-project-notes--ensure-endpoint-writable
       endpoint "산출물 링크 재시도")
      (with-current-buffer buffer
        (erase-buffer)
        (insert (alist-get 'post-text endpoint))
        (set-buffer-modified-p t)))))

(defun imoogi-project-notes--retry-operation-entry (operation)
  "Return the current registry entry for retry OPERATION."
  (let* ((entry-instance-id (alist-get 'entry-instance-id operation))
         (entry-note-id (alist-get 'entry-note-id operation))
         (entries (imoogi-project-notes--all-entries))
         (entry
          (cond
           (entry-instance-id
            (cl-find-if
             (lambda (candidate)
               (string= entry-instance-id
                        (alist-get 'instance-id candidate)))
             entries))
           (entry-note-id
            (cl-find-if
             (lambda (candidate)
               (string= entry-note-id
                        (imoogi-project-notes--entry-derived-note-id
                         candidate)))
             entries)))))
    (unless entry
      (user-error "재시도할 프로젝트 항목을 찾지 못했습니다"))
    entry))

(defun imoogi-project-notes--retry-task-file-entry-count (file)
  "Return distinct logical project count using task FILE."
  (let ((truename (file-truename file))
        note-ids)
    (dolist (candidate (imoogi-project-notes--all-entries))
      (when (string= truename
                     (file-truename
                      (imoogi-project-notes--alist-string
                       'tasks-file candidate)))
        (cl-pushnew (imoogi-project-notes--entry-derived-note-id candidate)
                    note-ids
                    :test #'string=)))
    (length note-ids)))

(defun imoogi-project-notes--retry-task-endpoint-owned-p
    (entry endpoint operation)
  "Return non-nil when task ENDPOINT still belongs to ENTRY/OPERATION."
  (let* ((file (alist-get 'file endpoint))
         (expected-file (imoogi-project-notes--alist-string 'tasks-file entry))
         (task-id (alist-get 'task-id operation))
         (expected-owner (imoogi-project-notes--entry-derived-note-id entry))
         (entry-count (and file
                           (imoogi-project-notes--retry-task-file-entry-count
                            file)))
         (pre-state (and (eq (imoogi-project-notes--endpoint-disk-state endpoint)
                             'pre)
                         (memq (imoogi-project-notes--endpoint-buffer-state
                                endpoint)
                               '(absent pre)))))
    (and file
         expected-file
         (string= (file-truename file) (file-truename expected-file))
         (or (not task-id)
             (and pre-state (= entry-count 1))
             (condition-case nil
                 (with-current-buffer (imoogi-project-notes--file-buffer file)
                   (save-excursion
                     (imoogi-project-notes--goto-heading-id task-id)
                     (let ((owner (org-entry-get (point)
                                                 "IMOOGI_PROJECT_ID")))
                       (and (org-entry-is-todo-p)
                            (or (string= owner expected-owner)
                                (and (not owner) (= entry-count 1)))))))
               (error nil))))))

(defun imoogi-project-notes--retry-document-endpoint-owned-p (entry endpoint)
  "Return non-nil when document ENDPOINT is still eligible for ENTRY."
  (let ((file (alist-get 'file endpoint)))
    (and file
         (ignore-errors
           (string= (file-truename file)
                    (imoogi-project-notes--validate-document-target-file
                     entry file "산출물 링크 재시도"))))))

(defun imoogi-project-notes--retry-endpoint-owned-p
    (entry endpoint operation)
  "Return non-nil when ENDPOINT still belongs to ENTRY/OPERATION."
  (pcase (alist-get 'role endpoint)
    ('task
     (imoogi-project-notes--retry-task-endpoint-owned-p
      entry endpoint operation))
    ((or 'doc 'old-doc 'new-doc)
     (imoogi-project-notes--retry-document-endpoint-owned-p entry endpoint))
    (_ nil)))

(defun imoogi-project-notes--validate-retry-operation (operation endpoints)
  "Validate OPERATION and ENDPOINTS against current project state."
  (let ((entry (imoogi-project-notes--retry-operation-entry operation)))
    (imoogi-project-notes--ensure-entry-mutable entry "산출물 링크 재시도")
    (dolist (endpoint endpoints)
      (unless (imoogi-project-notes--retry-endpoint-owned-p
               entry endpoint operation)
        (user-error "링크 재시도 대상이 현재 프로젝트에 속하지 않습니다: %s"
                    (alist-get 'file endpoint)))
      (imoogi-project-notes--ensure-endpoint-writable
       endpoint "산출물 링크 재시도"))
    entry))

(defun imoogi-project-notes--retry-link-operation-save (operation)
  "Retry OPERATION by saving its recorded endpoint postimages."
  (let ((endpoints (alist-get 'endpoints operation)))
    (unless endpoints
      (user-error "이전 형식의 링크 재시도 작업은 자동 재시도할 수 없습니다"))
    (imoogi-project-notes--validate-retry-operation operation endpoints)
    (dolist (endpoint endpoints)
      (imoogi-project-notes--prepare-endpoint-for-retry endpoint))
    (condition-case err
        (progn
          (dolist (endpoint endpoints)
            (when (eq (imoogi-project-notes--endpoint-disk-state endpoint) 'pre)
              (imoogi-project-notes--classify-save-buffer
               (imoogi-project-notes--endpoint-buffer endpoint)
               (alist-get 'pre-file-hash endpoint))))
          (when-let* ((doc-id (alist-get 'doc-id operation))
                      (doc-file (alist-get 'doc-file operation)))
            (org-id-add-location doc-id doc-file))
          (when-let* ((new-doc-id (alist-get 'new-doc-id operation))
                      (new-doc-file (alist-get 'new-doc-file operation)))
            (org-id-add-location new-doc-id new-doc-file))
          (when-let* ((task-id (alist-get 'task-id operation))
                      (task-file (alist-get 'task-file operation)))
            (org-id-add-location task-id task-file))
          (setq imoogi-project-notes--pending-link-operation nil)
          t)
      (error
       (dolist (endpoint endpoints)
         (imoogi-project-notes--restore-endpoint-preimage-if-unsaved endpoint))
       (display-warning
        'imoogi
        (format "프로젝트 문서 관계 재시도가 일부 실패했습니다: %s"
                (error-message-string err))
        :warning)
       (signal (car err) (cdr err))))))

(defun imoogi-project-notes--link-operation-base
    (operation entry task-file task-id task-title &rest pairs)
  "Return common retry metadata for OPERATION."
  (append
   `((operation . ,operation)
     (entry-note-id . ,(imoogi-project-notes--entry-derived-note-id entry))
     (entry-instance-id . ,(alist-get 'instance-id entry))
     (task-file . ,task-file)
     (task-id . ,task-id)
     (task-title . ,task-title))
   pairs))

(defun imoogi-project-notes--run-link-operation (operation metadata endpoints)
  "Save OPERATION METADATA ENDPOINTS transactionally."
  (let ((prepared (mapcar #'imoogi-project-notes--endpoint-with-postimage
                          endpoints)))
    (setq imoogi-project-notes--pending-link-operation
          (append metadata `((endpoints . ,prepared))))
    (condition-case err
        (progn
          (dolist (endpoint prepared)
            (unless (eq (imoogi-project-notes--endpoint-disk-state endpoint)
                        'pre)
              (user-error "링크 저장 전 대상 파일이 변경되었습니다: %s"
                          (alist-get 'file endpoint)))
            (imoogi-project-notes--classify-save-buffer
             (imoogi-project-notes--endpoint-buffer endpoint)
             (alist-get 'pre-file-hash endpoint)))
          (setq imoogi-project-notes--pending-link-operation nil)
          t)
      (error
       (dolist (endpoint prepared)
         (imoogi-project-notes--restore-endpoint-preimage-if-unsaved endpoint))
       (setq imoogi-project-notes--pending-link-operation
             (append metadata `((endpoints . ,prepared))))
       (display-warning
        'imoogi
        (format "프로젝트 문서 관계 저장이 일부 실패했습니다. `imoogi-project-notes-retry-link-operation'로 재시도하세요: %s"
                (error-message-string err))
        :warning)
       (signal (car err) (cdr err))))))

(defun imoogi-project-notes--endpoint-changed-p (endpoint)
  "Return non-nil when ENDPOINT's buffer differs from its disk file."
  (let ((file (alist-get 'file endpoint))
        (buffer (imoogi-project-notes--endpoint-buffer endpoint)))
    (not (and file
              (file-exists-p file)
              (string= (imoogi-project-notes--file-string-hash file)
                       (imoogi-project-notes--buffer-string-hash buffer))))))

(defun imoogi-project-notes--changed-endpoints (&rest endpoints)
  "Return ENDPOINTS whose buffers need saving."
  (cl-remove-if-not #'imoogi-project-notes--endpoint-changed-p
                    (delq nil endpoints)))

(defun imoogi-project-notes--save-prepared-relation
    (operation entry task-buffer doc-buffer task-id doc-id task-title doc-title
               task-preimage doc-preimage)
  "Save prepared task/document relation buffers with retry metadata."
  (imoogi-project-notes--run-link-operation
   operation
   (imoogi-project-notes--link-operation-base
    operation entry (buffer-file-name task-buffer) task-id task-title
    `(doc-file . ,(buffer-file-name doc-buffer))
    `(doc-id . ,doc-id)
    `(doc-title . ,doc-title))
   (imoogi-project-notes--changed-endpoints doc-preimage task-preimage))
  (when doc-id
    (org-id-add-location doc-id (buffer-file-name doc-buffer)))
  (when task-id
    (org-id-add-location task-id (buffer-file-name task-buffer)))
  t)

(defun imoogi-project-notes--save-prepared-retarget
    (entry task-buffer old-doc-buffer new-doc-buffer
           task-id old-doc-id new-doc-id task-title new-doc-title
           task-preimage old-doc-preimage new-doc-preimage)
  "Save prepared retarget buffers with concrete retry metadata."
  (let ((task-file (buffer-file-name task-buffer))
        (old-doc-file (and old-doc-buffer (buffer-file-name old-doc-buffer)))
        (new-doc-file (buffer-file-name new-doc-buffer)))
    (imoogi-project-notes--run-link-operation
     'retarget
     (imoogi-project-notes--link-operation-base
      'retarget entry task-file task-id task-title
      `(old-doc-file . ,old-doc-file)
      `(old-doc-id . ,old-doc-id)
      `(new-doc-file . ,new-doc-file)
      `(new-doc-id . ,new-doc-id)
      `(new-doc-title . ,new-doc-title))
     (imoogi-project-notes--changed-endpoints
      old-doc-preimage new-doc-preimage task-preimage))
    (org-id-add-location new-doc-id new-doc-file)
    (org-id-add-location task-id task-file)
    t))

(defun imoogi-project-notes--same-entry-p (a b)
  "Return non-nil when registry entries A and B identify the same project."
  (and a b
       (or (eq a b)
           (string= (imoogi-project-notes--entry-derived-note-id a)
                    (imoogi-project-notes--entry-derived-note-id b)))))

(defun imoogi-project-notes--select-task-entry-candidate (candidates prompt)
  "Read a project entry from CANDIDATES using PROMPT."
  (when (and noninteractive
             (subrp (symbol-function 'completing-read)))
    (user-error "작업 프로젝트를 자동으로 판별할 수 없습니다"))
  (let* ((choices (mapcar (lambda (entry)
                            (cons (imoogi-project-notes--entry-label entry)
                                  entry))
                          candidates))
         (choice (completing-read prompt choices nil t)))
    (cdr (assoc choice choices))))

(defun imoogi-project-notes--task-entry (&optional entry)
  "Return the current task's owning project entry.
When ENTRY is provided, validate it against the actual task file and stable
heading ownership metadata instead of trusting the caller's ambient choice."
  (unless buffer-file-name
    (user-error "작업 파일을 찾을 수 없습니다"))
  (let* ((candidates (imoogi-project-notes--entries-for-tasks-file
                      buffer-file-name))
         (project-id (and (derived-mode-p 'org-mode)
                          (org-entry-get (point) "IMOOGI_PROJECT_ID")))
         (category (and (derived-mode-p 'org-mode) (org-get-category)))
         (by-project-id
          (and project-id
               (list (imoogi-project-notes--entry-from-project-id
                      project-id candidates "작업"))))
         (by-category
          (and (not project-id)
               (imoogi-project-notes--unique-entries
                (cl-remove-if-not
                 (lambda (candidate)
                   (string= category
                            (imoogi-project-notes--entry-name candidate)))
                 candidates))))
         selected)
    (unless candidates
      (user-error "현재 파일은 등록된 프로젝트 tasks.org가 아닙니다: %s"
                  buffer-file-name))
    (setq selected
          (cond
           (entry
            (unless (cl-some (lambda (candidate)
                               (imoogi-project-notes--same-entry-p
                                candidate entry))
                             candidates)
              (user-error "선택한 프로젝트가 현재 작업 파일을 소유하지 않습니다: %s"
                          buffer-file-name))
            entry)
           ((= (length by-project-id) 1) (car by-project-id))
           ((= (length candidates) 1) (car candidates))
           ((= (length by-category) 1) (car by-category))
           ((> (length by-category) 1)
            (user-error "작업 CATEGORY가 중복 프로젝트를 가리킵니다: %s"
                        category))
           (t
            (imoogi-project-notes--select-task-entry-candidate
             candidates "작업 프로젝트: "))))
    (when project-id
      (let ((expected (imoogi-project-notes--entry-derived-note-id selected)))
        (unless (string= project-id expected)
          (user-error "작업의 IMOOGI_PROJECT_ID가 선택한 프로젝트와 다릅니다: %s"
                      project-id))))
    selected))

(defun imoogi-project-notes--ensure-task-project-owner (entry)
  "Record central-task ownership for ENTRY on the current task when needed."
  (let* ((task-file (and buffer-file-name (file-truename buffer-file-name)))
         (matches
          (and task-file
               (cl-remove-if-not
                (lambda (candidate)
                  (string= task-file
                           (file-truename
                            (imoogi-project-notes--alist-string
                             'tasks-file candidate))))
                (imoogi-project-notes--all-entries)))))
    (when (> (length matches) 1)
      (let ((expected (imoogi-project-notes--entry-derived-note-id entry))
            (actual (org-entry-get (point) "IMOOGI_PROJECT_ID")))
        (when (and actual (not (string= actual expected)))
          (user-error "작업의 IMOOGI_PROJECT_ID가 선택한 프로젝트와 다릅니다: %s"
                      actual))
        (unless actual
          (org-entry-put (point) "IMOOGI_PROJECT_ID" expected))))))

(defun imoogi-project-notes--link-current-task-to-document (doc-file &optional entry)
  "Link current task to DOC-FILE and return its document info."
  (unless (derived-mode-p 'org-mode)
    (user-error "Org TODO heading에서 실행하세요"))
  (org-back-to-heading t)
  (unless (org-entry-is-todo-p)
    (user-error "Org TODO heading에서 실행하세요"))
  (let* ((entry (imoogi-project-notes--task-entry entry))
         (task-buffer (current-buffer))
         (notes-dir (imoogi-project-notes--alist-string 'notes-dir entry))
         (doc-file (and doc-file (expand-file-name doc-file)))
         task-id task-title
         doc-buffer
         doc-id doc-title
         task-preimage doc-preimage)
    (imoogi-project-notes--ensure-entry-mutable entry "산출물 연결")
    (setq doc-file
          (imoogi-project-notes--validate-document-target-file
           entry doc-file "산출물 연결"))
    (imoogi-project-notes--ensure-existing-task-id-valid entry)
    (setq task-title (org-get-heading t t t t)
          doc-buffer (imoogi-project-notes--file-buffer doc-file))
    (with-current-buffer doc-buffer
      (imoogi-project-notes--preflight-document-relation-target)
      (when-let* ((existing (imoogi-project-notes--document-identity)))
        (imoogi-project-notes--document-by-id entry (car existing))))
    (setq task-preimage
          (imoogi-project-notes--endpoint-preimage
           'task task-buffer "산출물 연결")
          doc-preimage
          (imoogi-project-notes--endpoint-preimage
           'doc doc-buffer "산출물 연결"))
    (imoogi-project-notes--with-link-operation-preparation
        (list task-preimage doc-preimage)
      (with-current-buffer doc-buffer
        (pcase-let ((`(,id ,title ,_kind)
                     (imoogi-project-notes--document-identity 'create)))
          (setq doc-id id
                doc-title title)
          (imoogi-project-notes--document-by-id entry doc-id)))
      (setq task-id (org-id-get-create))
      (imoogi-project-notes--ensure-task-project-owner entry)
      (with-current-buffer doc-buffer
        (imoogi-project-notes--ensure-document-task-link task-id task-title))
      (with-current-buffer task-buffer
        (imoogi-project-notes--ensure-task-artifact-link doc-id doc-title)))
    (imoogi-project-notes--save-prepared-relation
     'link entry task-buffer doc-buffer task-id doc-id task-title doc-title
     task-preimage doc-preimage)
    `((file . ,(buffer-file-name doc-buffer))
      (id . ,doc-id)
      (title . ,doc-title))))

(defun imoogi-project-notes--unlink-current-task-from-document
    (doc-id doc-file &optional entry)
  "Unlink current task from DOC-ID and DOC-FILE where available."
  (unless (derived-mode-p 'org-mode)
    (user-error "Org TODO heading에서 실행하세요"))
  (org-back-to-heading t)
  (let* ((entry (imoogi-project-notes--task-entry entry))
         (task-buffer (current-buffer))
         task-id
         (task-title (org-get-heading t t t t))
         doc-buffer
         (doc-title doc-id)
         task-preimage doc-preimage)
    (imoogi-project-notes--ensure-entry-mutable entry "산출물 연결 해제")
    (imoogi-project-notes--ensure-existing-task-id-valid entry)
    (setq doc-file
          (imoogi-project-notes--validate-document-id-file entry doc-id doc-file)
          doc-buffer (and doc-file
                          (file-exists-p doc-file)
                          (imoogi-project-notes--file-buffer doc-file)))
    (setq task-preimage
          (imoogi-project-notes--endpoint-preimage
           'task task-buffer "산출물 연결 해제"))
    (when doc-buffer
      (setq doc-preimage
            (imoogi-project-notes--endpoint-preimage
             'doc doc-buffer "산출물 연결 해제")))
    (with-current-buffer task-buffer
      (setq task-id (org-entry-get (point) "ID")))
    (if (and doc-buffer task-id)
        (with-current-buffer doc-buffer
          (unless (imoogi-project-notes--document-task-link-present-p task-id)
            (user-error "대상 문서에 작업 링크가 없어 repair-link로 재연결해야 합니다: %s"
                        doc-id)))
      (imoogi-project-notes--confirm-source-only-unlink doc-id))
    (imoogi-project-notes--with-link-operation-preparation
        (delq nil (list task-preimage doc-preimage))
      (with-current-buffer task-buffer
        (imoogi-project-notes--remove-task-artifact-link doc-id))
      (when (and doc-buffer task-id)
        (with-current-buffer doc-buffer
          (setq doc-title (imoogi-project-notes--org-title))
          (imoogi-project-notes--remove-document-task-link task-id))))
    (if (and doc-buffer task-id)
        (imoogi-project-notes--save-prepared-relation
         'unlink entry task-buffer doc-buffer task-id doc-id task-title doc-title
         task-preimage doc-preimage)
      (progn
        (imoogi-project-notes--run-link-operation
         'unlink
         (imoogi-project-notes--link-operation-base
          'unlink entry (buffer-file-name task-buffer) task-id task-title
          `(doc-file . ,(and doc-buffer (buffer-file-name doc-buffer)))
          `(doc-id . ,doc-id)
          `(doc-title . ,doc-title))
         (imoogi-project-notes--changed-endpoints task-preimage))
        (display-warning
         'imoogi
         (format "대상 문서를 찾지 못해 작업 쪽 링크만 제거했습니다: %s" doc-id)
         :warning)))))

(defun imoogi-project-notes--retarget-current-task-document
    (old-doc-id new-doc-file &optional entry)
  "Retarget the current task relation from OLD-DOC-ID to NEW-DOC-FILE."
  (unless (derived-mode-p 'org-mode)
    (user-error "Org TODO heading에서 실행하세요"))
  (org-back-to-heading t)
  (unless (org-entry-is-todo-p)
    (user-error "Org TODO heading에서 실행하세요"))
  (let* ((entry (imoogi-project-notes--task-entry entry))
         (task-buffer (current-buffer))
         (notes-dir (imoogi-project-notes--alist-string 'notes-dir entry))
         (new-doc-file (and new-doc-file (expand-file-name new-doc-file)))
         task-id task-title
         old-doc
         old-doc-buffer
         new-doc-buffer
         new-doc-id new-doc-title
         task-preimage old-doc-preimage new-doc-preimage)
    (imoogi-project-notes--ensure-entry-mutable entry "산출물 링크 교체")
    (setq new-doc-file
          (imoogi-project-notes--validate-document-target-file
           entry new-doc-file "산출물 링크 교체"))
    (imoogi-project-notes--ensure-existing-task-id-valid entry)
    (setq task-title (org-get-heading t t t t))
    (setq old-doc (imoogi-project-notes--document-by-id entry old-doc-id))
    (when-let* ((file (alist-get 'file old-doc)))
      (setq old-doc-buffer (and (file-exists-p file)
                                (imoogi-project-notes--file-buffer file)))
      (when old-doc-buffer
        (setq old-doc-preimage
              (imoogi-project-notes--endpoint-preimage
               'old-doc old-doc-buffer "산출물 링크 교체"))))
    (unless old-doc-buffer
      (display-warning
       'imoogi
       (format "기존 문서를 찾지 못해 작업 쪽 링크만 교체합니다: %s" old-doc-id)
       :warning))
    (setq new-doc-buffer (imoogi-project-notes--file-buffer new-doc-file))
    (with-current-buffer new-doc-buffer
      (imoogi-project-notes--preflight-document-relation-target)
      (when-let* ((existing (imoogi-project-notes--document-identity)))
        (imoogi-project-notes--document-by-id entry (car existing))))
    (setq task-preimage
          (imoogi-project-notes--endpoint-preimage
           'task task-buffer "산출물 링크 교체")
          new-doc-preimage
          (imoogi-project-notes--endpoint-preimage
           'new-doc new-doc-buffer "산출물 링크 교체"))
    (imoogi-project-notes--with-link-operation-preparation
        (delq nil (list task-preimage old-doc-preimage new-doc-preimage))
      (setq task-id (org-id-get-create))
      (imoogi-project-notes--ensure-task-project-owner entry)
      (when old-doc-buffer
        (with-current-buffer old-doc-buffer
          (imoogi-project-notes--remove-document-task-link task-id)))
      (with-current-buffer new-doc-buffer
        (pcase-let ((`(,id ,title ,_kind)
                     (imoogi-project-notes--document-identity 'create)))
          (setq new-doc-id id
                new-doc-title title)
          (imoogi-project-notes--document-by-id entry new-doc-id)
          (imoogi-project-notes--ensure-document-task-link task-id task-title)))
      (with-current-buffer task-buffer
        (imoogi-project-notes--remove-task-artifact-link old-doc-id)
        (imoogi-project-notes--ensure-task-artifact-link
         new-doc-id new-doc-title)))
    (imoogi-project-notes--save-prepared-retarget
     entry task-buffer old-doc-buffer new-doc-buffer
     task-id old-doc-id new-doc-id task-title new-doc-title
     task-preimage old-doc-preimage new-doc-preimage)
    `((old-id . ,old-doc-id)
      (file . ,new-doc-file)
      (id . ,new-doc-id)
      (title . ,new-doc-title))))

(defun imoogi-project-notes--task-linked-doc-ids ()
  "Return document IDs from the current task's managed artifact list."
  (let (ids)
    (save-excursion
      (org-back-to-heading t)
      (when-let* ((bounds (imoogi-project-notes--task-artifact-section-bounds
                           (imoogi-project-notes--current-heading-direct-end))))
        (pcase-let ((`(,beg ,end) bounds))
          (goto-char beg)
          (while (re-search-forward "\\[\\[id:\\([^]]+\\)\\]\\[[^]\n]+\\]\\]"
                                    end t)
            (push (match-string 1) ids)))))
    (delete-dups (nreverse ids))))

(defun imoogi-project-notes--select-linked-document (entry)
  "Select an already linked document for current task."
  (let* ((ids (imoogi-project-notes--task-linked-doc-ids))
         (docs (mapcar (lambda (id)
                         (or (imoogi-project-notes--document-by-id entry id)
                             `((id . ,id) (title . ,id) (file . nil))))
                       ids))
         (choices (mapcar (lambda (doc)
                            (cons (format "%s — %s"
                                          (alist-get 'title doc)
                                          (or (alist-get 'file doc) "missing"))
                                  doc))
                          docs))
         (choice (completing-read "연결 해제할 문서: " choices nil t)))
    (cdr (assoc choice choices))))

(defun imoogi-project-notes--goto-heading-id (id)
  "Move point to the heading whose ID property is ID."
  (let (found)
    (org-with-wide-buffer
     (goto-char (point-min))
     (org-map-entries
      (lambda ()
        (when (string= (org-entry-get (point) "ID") id)
          (setq found (point))))
      nil 'file))
    (unless found
      (user-error "ID에 해당하는 heading을 찾지 못했습니다: %s" id))
    (goto-char found)))

(defun imoogi-project-notes--entry-by-note-id (note-id &optional entries)
  "Return registered entry with NOTE-ID."
  (cl-find-if
   (lambda (entry)
     (string= note-id (imoogi-project-notes--entry-derived-note-id entry)))
   (or entries (imoogi-project-notes--all-entries))))

(defun imoogi-project-notes--entry-equal-p (a b)
  "Return non-nil when entries A and B refer to the same project note."
  (and a b
       (string= (file-truename (imoogi-project-notes--alist-string
                                'notes-dir a))
                (file-truename (imoogi-project-notes--alist-string
                                'notes-dir b)))))

(defun imoogi-project-notes--entry-same-scope-p (a b)
  "Return non-nil when entries A and B have the same cache-relevant scope."
  (and a b
       (string= (imoogi-project-notes--entry-derived-note-id a)
                (imoogi-project-notes--entry-derived-note-id b))
       (string= (file-truename (imoogi-project-notes--alist-string
                                'notes-dir a))
                (file-truename (imoogi-project-notes--alist-string
                                'notes-dir b)))
       (string= (file-truename (imoogi-project-notes--alist-string
                                'tasks-file a))
                (file-truename (imoogi-project-notes--alist-string
                                'tasks-file b)))))

(defun imoogi-project-notes--unique-entries (entries)
  "Return ENTRIES without duplicate project-note roots."
  (let (unique)
    (dolist (entry entries (nreverse unique))
      (unless (cl-some (lambda (seen)
                         (imoogi-project-notes--entry-equal-p
                          seen entry))
                       unique)
        (push entry unique)))))

(defun imoogi-project-notes--org-id-link-at-point-p ()
  "Return non-nil when point is on an Org id link element."
  (save-match-data
    (let ((context (org-element-context)))
      (and (eq (org-element-type context) 'link)
           (string= (org-element-property :type context) "id")))))

(defun imoogi-project-notes--managed-id-link-line-p ()
  "Return non-nil when the current line is a managed relation ID link."
  (save-excursion
    (beginning-of-line)
    (unless (or (org-in-src-block-p t)
                (looking-at-p "[[:space:]]*#"))
      (let ((line-end (line-end-position))
            (bare-line
             (save-excursion
               (looking-at
                "[[:space:]]*\\[\\[id:[^]\n]+\\]\\[[^]\n]+\\]\\][[:space:]]*$"))))
        (when (or bare-line (looking-at "[[:space:]]*- "))
          (re-search-forward "\\[\\[id:[^]\n]+\\]\\[[^]\n]+\\]\\]"
                             line-end t)
          (and (match-beginning 0)
               (goto-char (match-beginning 0))
               (imoogi-project-notes--org-id-link-at-point-p)))))))

(defun imoogi-project-notes--section-has-managed-id-links-p (bounds)
  "Return non-nil when BOUNDS contain managed ID list links."
  (pcase-let ((`(,beg ,end) bounds))
    (save-excursion
      (goto-char beg)
      (catch 'found
        (while (< (point) end)
          (when (imoogi-project-notes--managed-id-link-line-p)
            (throw 'found t))
          (forward-line 1))
        nil))))

(defun imoogi-project-notes--current-task-managed-p ()
  "Return non-nil when the current task has managed artifact relations."
  (and (derived-mode-p 'org-mode)
       (save-excursion
         (org-back-to-heading t)
         (when-let* ((bounds (imoogi-project-notes--task-artifact-section-bounds
                              (imoogi-project-notes--current-heading-direct-end))))
           (imoogi-project-notes--section-has-managed-id-links-p bounds)))))

(defun imoogi-project-notes--file-has-document-task-link-p (file task-id)
  "Return non-nil when FILE has a managed document relation to TASK-ID."
  (if-let* ((buffer (find-buffer-visiting file)))
      (with-current-buffer buffer
        (save-excursion
          (imoogi-project-notes--document-task-link-present-p task-id)))
    (with-temp-buffer
      (insert-file-contents file)
      (org-mode)
      (imoogi-project-notes--document-task-link-present-p task-id))))

(defun imoogi-project-notes--current-task-inbound-managed-p ()
  "Return non-nil when a project document links to the current task."
  (and (derived-mode-p 'org-mode)
       (save-excursion
         (org-back-to-heading t)
         (when-let* ((task-id (org-entry-get (point) "ID"))
                     (entry (condition-case nil
                                (imoogi-project-notes--task-entry)
                              (user-error nil))))
           (cl-some
            (lambda (file)
              (imoogi-project-notes--file-has-document-task-link-p file task-id))
            (imoogi-project-notes--project-document-files entry))))))

(defun imoogi-project-notes--buffer-tasks-file-links-document-p (doc-id)
  "Return non-nil when the current Org buffer links a task to DOC-ID."
  (let (found)
    (save-excursion
      (save-restriction
        (widen)
        (org-map-entries
         (lambda ()
           (when-let* ((bounds (imoogi-project-notes--task-artifact-section-bounds
                                (imoogi-project-notes--current-heading-direct-end))))
             (pcase-let ((`(,beg ,end) bounds))
               (when (imoogi-project-notes--managed-id-link-present-in-region-p
                      doc-id beg end)
                 (setq found t)))))
         nil 'file)))
    found))

(defun imoogi-project-notes--tasks-file-links-document-p (tasks-file doc-id)
  "Return non-nil when TASKS-FILE has a managed artifact link to DOC-ID."
  (when (and tasks-file (file-exists-p tasks-file))
    (if-let* ((buffer (find-buffer-visiting tasks-file)))
        (with-current-buffer buffer
          (imoogi-project-notes--buffer-tasks-file-links-document-p doc-id))
      (with-temp-buffer
        (insert-file-contents tasks-file)
        (org-mode)
        (imoogi-project-notes--buffer-tasks-file-links-document-p doc-id)))))

(defun imoogi-project-notes--current-document-inbound-managed-p ()
  "Return non-nil when a project task links to the current document."
  (when-let* ((identity (imoogi-project-notes--document-identity))
              (doc-id (car identity))
              (file buffer-file-name)
              (entry (imoogi-project-notes--find-entry-by-notes-file file)))
    (imoogi-project-notes--tasks-file-links-document-p
     (imoogi-project-notes--alist-string 'tasks-file entry) doc-id)))

(defun imoogi-project-notes--current-file-has-managed-task-relations-p ()
  "Return non-nil when any heading in current file owns managed artifacts."
  (when (derived-mode-p 'org-mode)
    (save-excursion
      (save-restriction
        (widen)
        (let (found)
          (org-map-entries
           (lambda ()
             (when (imoogi-project-notes--current-task-managed-p)
               (setq found t)))
           nil 'file)
          found)))))

(defun imoogi-project-notes--document-managed-p ()
  "Return non-nil when the current document has managed relation links."
  (or (when-let* ((bounds (imoogi-project-notes--link-heading-bounds)))
        (imoogi-project-notes--section-has-managed-id-links-p bounds))
      (cl-some #'imoogi-project-notes--section-has-managed-id-links-p
               (imoogi-project-notes--legacy-related-bounds))))

(defun imoogi-project-notes--copy-source-kind ()
  "Return the safe copy source kind at point.
The result is one of `file' or `subtree'.  Partial active regions are rejected."
  (unless (derived-mode-p 'org-mode)
    (user-error "Org 버퍼에서 실행하세요"))
  (cond
   ((org-region-active-p)
    (or (imoogi-project-notes--complete-active-region-kind)
        (user-error "부분 region은 프로젝트 문서 copy 대상으로 사용할 수 없습니다")))
   ((org-before-first-heading-p) 'file)
   ((and (save-excursion
           (org-back-to-heading t)
           (= (org-outline-level) 1))
         (save-excursion
           (org-back-to-heading t)
           (or (org-entry-get (point) "TYPE")
               (imoogi-project-notes--document-managed-p))))
    'file)
   ((org-at-heading-p) 'subtree)
   (t (user-error "복사할 Org heading 또는 파일 위치에서 실행하세요"))))

(defun imoogi-project-notes--copy-source-protected-p (&optional kind)
  "Return non-nil when current source KIND has managed project relations."
  (let ((kind (or kind (imoogi-project-notes--copy-source-kind))))
    (pcase kind
      ('file (or (imoogi-project-notes--document-managed-p)
                 (imoogi-project-notes--current-document-inbound-managed-p)
                 (imoogi-project-notes--current-file-has-managed-task-relations-p)))
      ('subtree (save-excursion
                  (goto-char (imoogi-project-notes--copy-source-point kind))
                  (org-back-to-heading t)
                  (or (imoogi-project-notes--current-task-managed-p)
                      (imoogi-project-notes--current-task-inbound-managed-p)
                      (and (= (org-outline-level) 1)
                           (org-entry-get (point) "TYPE")
                           (imoogi-project-notes--document-managed-p)))))
      (_ nil))))

(defun imoogi-project-notes--managed-refile-source-context-p ()
  "Return non-nil when current Org buffer/region is a managed source."
  (and (derived-mode-p 'org-mode)
       (or (imoogi-project-notes--document-managed-p)
           (imoogi-project-notes--current-document-inbound-managed-p)
           (imoogi-project-notes--current-file-has-managed-task-relations-p)
           (save-excursion
             (when (org-region-active-p)
               (goto-char (region-beginning)))
             (condition-case nil
                 (progn
                   (org-back-to-heading t)
                   (or (imoogi-project-notes--current-task-managed-p)
                       (imoogi-project-notes--current-task-inbound-managed-p)))
               (user-error nil)
               (error nil))))))

(defun imoogi-project-notes--complete-active-region-kind ()
  "Return `file' or `subtree' when the active region is a complete copy unit."
  (when (org-region-active-p)
    (let ((beg (region-beginning))
          (end (region-end)))
      (cond
       ((and (= beg (point-min)) (= end (point-max))) 'file)
       ((save-excursion
          (goto-char beg)
          (and (org-at-heading-p)
               (= end (save-excursion (org-end-of-subtree t t)))))
        (if (imoogi-project-notes--active-region-legacy-root-with-links-p
             beg end)
            'file
          'subtree))))))

(defun imoogi-project-notes--active-region-legacy-root-with-links-p (beg end)
  "Return non-nil when BEG..END selects a legacy root but omits its Link sibling."
  (save-excursion
    (goto-char beg)
    (and (org-at-heading-p)
         (= (org-outline-level) 1)
         (= end (save-excursion (org-end-of-subtree t t)))
         (org-entry-get (point) "TYPE")
         (or (imoogi-project-notes--link-heading-bounds)
             (imoogi-project-notes--legacy-related-bounds)))))

(defun imoogi-project-notes--refile-source-kind ()
  "Return copy source kind for managed refile guards, or nil for unrelated sources.
Reject ambiguous partial regions when the surrounding source is managed."
  (when (derived-mode-p 'org-mode)
    (if (org-region-active-p)
        (or (imoogi-project-notes--complete-active-region-kind)
            (progn
              (when (imoogi-project-notes--managed-refile-source-context-p)
                (user-error "부분 region은 프로젝트 문서 copy 대상으로 사용할 수 없습니다"))
              nil))
      (condition-case err
          (imoogi-project-notes--copy-source-kind)
        (user-error
         (when (imoogi-project-notes--managed-refile-source-context-p)
           (user-error "%s" (error-message-string err)))
         nil)))))

(defun imoogi-project-notes--select-copy-destination-entry (&optional source-entry)
  "Read a writable project entry as copy destination."
  (let* ((entries (cl-remove-if
                   #'imoogi-project-notes--inactive-mounted-entry-p
                   (imoogi-project-notes--all-entries)))
         (choices
          (mapcar (lambda (entry)
                    (cons (format "%s — %s"
                                  (imoogi-project-notes--entry-name entry)
                                  (imoogi-project-notes--alist-string
                                   'notes-dir entry))
                          entry))
                  entries))
         (choice (completing-read
                  (if source-entry "복사할 프로젝트: " "프로젝트: ")
                  choices nil t)))
    (cdr (assoc choice choices))))

(defun imoogi-project-notes--copy-source-point (kind)
  "Return the point anchoring current copy source KIND."
  (if (and (eq kind 'subtree) (org-region-active-p))
      (region-beginning)
    (point)))

(defun imoogi-project-notes--copy-source-title (kind)
  "Return a title for current copy source KIND."
  (pcase kind
    ('file (imoogi-project-notes--org-title))
    ('subtree (save-excursion
                (goto-char (imoogi-project-notes--copy-source-point kind))
                (org-back-to-heading t)
                (org-get-heading t t t t)))))

(defun imoogi-project-notes--copy-source-text (kind destination-entry)
  "Return Org text for current copy source KIND for DESTINATION-ENTRY."
  (pcase kind
    ('file
     (buffer-substring-no-properties (point-min) (point-max)))
    ('subtree
     (save-excursion
       (goto-char (imoogi-project-notes--copy-source-point kind))
       (org-back-to-heading t)
       (let ((title (org-get-heading t t t t))
             (beg (point))
             (end (save-excursion (org-end-of-subtree t t))))
         (concat "#+TITLE: " title "\n"
                 "#+CATEGORY: "
                 (imoogi-project-notes--entry-name destination-entry)
                 "\n\n"
                 (buffer-substring-no-properties beg end)))))))

(defun imoogi-project-notes--copy-destination-file
    (destination-entry source-file title)
  "Return a unique destination file for SOURCE-FILE named TITLE."
  (let* ((notes-dir (imoogi-project-notes--alist-string
                     'notes-dir destination-entry))
         (artifact-dir (expand-file-name "artifacts/" notes-dir))
         (base-title (or title (file-name-base source-file))))
    (imoogi-project-notes--unique-artifact-file
     artifact-dir "copy" base-title)))

(defun imoogi-project-notes--strip-managed-link-lines-in-region (beg end)
  "Strip managed ID links between BEG and END, preserving surrounding prose."
  (save-excursion
    (goto-char beg)
    (while (< (point) end)
      (let* ((line-beg (line-beginning-position))
             (line-end (line-end-position))
             (line-limit (min (1+ line-end) (point-max))))
        (if (not (imoogi-project-notes--managed-id-link-line-p))
            (forward-line 1)
          (goto-char line-beg)
          (when (re-search-forward "\\[\\[id:[^]\n]+\\]\\[[^]\n]+\\]\\]"
                                   line-end t)
            (let ((match-beg (match-beginning 0))
                  (match-end (match-end 0)))
              (when (save-excursion
                      (goto-char match-beg)
                      (imoogi-project-notes--org-id-link-at-point-p))
                (let ((remaining (string-trim
                                  (concat
                                   (buffer-substring-no-properties
                                    line-beg match-beg)
                                   (buffer-substring-no-properties
                                    match-end line-end)))))
                  (if (or (string-empty-p remaining) (string= remaining "-"))
                      (progn
                        (delete-region line-beg line-limit)
                        (setq end (- end (- line-limit line-beg)))
                        (goto-char line-beg))
                    (delete-region match-beg match-end)
                    (setq end (- end (- match-end match-beg)))
                    (forward-line 1)))))))))))

(defun imoogi-project-notes--remove-copied-source-owner-properties ()
  "Remove source-only ownership properties from a copied Org buffer."
  (goto-char (point-min))
  (while (re-search-forward
          "^[[:space:]]*:\\(IMOOGI_PROJECT_ID\\|CATEGORY\\):[[:space:]]*\\([^\n]*\\)\n"
          nil t)
    (let ((beg (match-beginning 0))
          (name (match-string 1))
          (value-beg (match-beginning 2)))
      (when (imoogi-project-notes--node-property-at-position-p name value-beg)
        (delete-region beg (match-end 0))
        (goto-char beg)))))

(defun imoogi-project-notes--strip-current-task-artifact-relations ()
  "Strip managed artifact relation links from the current task only."
  (when-let* ((bounds (imoogi-project-notes--task-artifact-section-bounds
                       (imoogi-project-notes--current-heading-direct-end))))
    (pcase-let ((`(,beg ,end) bounds))
      (imoogi-project-notes--strip-managed-link-lines-in-region beg end))))

(defun imoogi-project-notes--strip-managed-copy-relations ()
  "Remove managed relationship entries from the current copied Org buffer."
  (org-with-wide-buffer
   (when-let* ((bounds (imoogi-project-notes--link-heading-bounds)))
     (pcase-let ((`(,beg ,end) bounds))
       (imoogi-project-notes--strip-managed-link-lines-in-region beg end)))
   (dolist (bounds (imoogi-project-notes--legacy-related-bounds))
     (pcase-let ((`(,beg ,end) bounds))
       (imoogi-project-notes--strip-managed-link-lines-in-region beg end)))
   (goto-char (point-min))
   (while (re-search-forward org-heading-regexp nil t)
     (goto-char (match-beginning 0))
     (imoogi-project-notes--strip-current-task-artifact-relations)
     (forward-line 1))
   (imoogi-project-notes--remove-copied-source-owner-properties)))

(defun imoogi-project-notes--freshen-copy-ids ()
  "Assign fresh IDs in current buffer and rewrite copied internal ID links."
  (let ((old-to-new nil)
        (seen nil))
    (org-with-wide-buffer
     (goto-char (point-min))
     (while (re-search-forward "^\\([[:space:]]*:ID:[[:space:]]+\\)\\([^[:space:]\n]+\\)"
                               nil t)
       (let ((prefix (match-string 1))
             (old (match-string 2))
             (beg (match-beginning 0))
             (end (match-end 0)))
         (when (eq (org-element-type
                    (save-excursion
                      (goto-char beg)
                      (org-element-context)))
                   'node-property)
           (when (member old seen)
             (user-error "복사 원본에 중복 ID가 있어 안전하게 복사할 수 없습니다: %s"
                         old))
           (push old seen)
           (let ((new (imoogi-project-notes--new-note-id)))
             (push (cons old new) old-to-new)
             (delete-region beg end)
             (insert prefix new)))))
     (dolist (mapping old-to-new)
       (goto-char (point-min))
       (while (re-search-forward
               (concat "\\[\\[id:" (regexp-quote (car mapping)) "\\]")
               nil t)
         (when (save-excursion
                 (goto-char (match-beginning 0))
                 (imoogi-project-notes--org-id-link-at-point-p))
           (replace-match (concat "[[id:" (cdr mapping) "]") t t)))))
    (nreverse old-to-new)))

(defun imoogi-project-notes--copy-transform (text)
  "Return (TEXT ID-MAP) for copied Org TEXT."
  (with-temp-buffer
    (insert text)
    (org-mode)
    (imoogi-project-notes--strip-managed-copy-relations)
    (let ((id-map (imoogi-project-notes--freshen-copy-ids)))
      (list (buffer-string) id-map))))

(defun imoogi-project-notes--publish-copy-ids (id-map file)
  "Publish copied ID-MAP locations to FILE."
  (dolist (mapping id-map)
    (org-id-add-location (cdr mapping) file)))

(defun imoogi-project-notes--warn-external-id-citations (content id-map)
  "Warn when copied CONTENT still contains ID links outside copied ID-MAP."
  (let ((fresh-ids (mapcar #'cdr id-map))
        external)
    (with-temp-buffer
      (insert content)
      (org-mode)
      (org-with-wide-buffer
       (goto-char (point-min))
       (while (re-search-forward "\\[\\[id:\\([^]\n]+\\)\\]" nil t)
         (let ((id (match-string 1)))
           (when (and (save-excursion
                        (goto-char (match-beginning 0))
                        (imoogi-project-notes--org-id-link-at-point-p))
                      (not (member id fresh-ids)))
             (push id external))))))
    (when external
      (display-warning
       'imoogi
       (format "복사본에 외부 ID 링크가 남아 있습니다: %s"
               (string-join (delete-dups (nreverse external)) ", "))
       :warning))))

(defun imoogi-project-notes--copy-current-source-to-entry
    (destination-entry &optional kind)
  "Copy current Org file/subtree to DESTINATION-ENTRY with fresh IDs."
  (let* ((kind (or kind (imoogi-project-notes--copy-source-kind)))
         (source-file buffer-file-name)
         (title (imoogi-project-notes--copy-source-title kind))
         (destination-file
          (imoogi-project-notes--copy-destination-file
           destination-entry source-file title)))
    (imoogi-project-notes--ensure-entry-mutable
     destination-entry "프로젝트 문서 복사")
    (unless (file-in-directory-p
             (file-truename destination-file)
             (file-truename
              (imoogi-project-notes--alist-string 'notes-dir destination-entry)))
      (user-error "복사 대상이 프로젝트 문서 폴더 밖입니다: %s" destination-file))
    (when (file-exists-p destination-file)
      (user-error "복사 대상 파일이 이미 있어 덮어쓰지 않습니다: %s"
                  destination-file))
    (pcase-let ((`(,content ,id-map)
                 (imoogi-project-notes--copy-transform
                  (imoogi-project-notes--copy-source-text
                   kind destination-entry))))
      (imoogi-project-notes--warn-external-id-citations content id-map)
      (imoogi-project-notes--exclusive-create-file destination-file content)
      (imoogi-project-notes--publish-copy-ids id-map destination-file)
      (message "프로젝트 문서를 복사했습니다: %s" destination-file)
      `((file . ,destination-file)
        (id-map . ,id-map)
        (kind . ,kind)
        (source-file . ,source-file)))))

;;;###autoload
(defun imoogi-project-notes-copy-to-project (&optional destination-entry kind)
  "Copy current Org file or complete subtree to another project note.
The copy keeps the original unchanged, assigns fresh IDs to copied ID-bearing
nodes, rewrites internal copied ID links, strips managed project relationship
entries, and writes a unique destination file without overwriting."
  (interactive)
  (let ((entry (or destination-entry
                   (imoogi-project-notes--select-copy-destination-entry
                    (imoogi-project-notes--current-entry)))))
    (imoogi-project-notes--copy-current-source-to-entry entry kind)))

(defun imoogi-project-notes--entries-for-tasks-file (file &optional entries)
  "Return registered entries whose tasks file is FILE."
  (let ((truename (file-truename file)))
    (imoogi-project-notes--unique-entries
     (cl-remove-if-not
      (lambda (entry)
        (when-let* ((tasks-file (imoogi-project-notes--alist-string
                                 'tasks-file entry)))
          (string= truename (file-truename tasks-file))))
      (or entries (imoogi-project-notes--all-entries))))))

(defun imoogi-project-notes--entry-from-project-id
    (project-id candidates context)
  "Return the unique CANDIDATES entry for PROJECT-ID.
Signal a user error naming CONTEXT when PROJECT-ID is ambiguous."
  (when (and project-id (not (string-empty-p project-id)))
    (let ((matches (imoogi-project-notes--unique-entries
                    (cl-remove-if-not
                     (lambda (candidate)
                       (string= project-id
                                (imoogi-project-notes--entry-derived-note-id
                                 candidate)))
                     candidates))))
      (cond
       ((= (length matches) 1) (car matches))
       ((= (length matches) 0)
        (user-error "%s의 IMOOGI_PROJECT_ID가 등록된 프로젝트와 일치하지 않습니다: %s"
                    context project-id))
       ((> (length matches) 1)
        (user-error "%s의 IMOOGI_PROJECT_ID가 중복 프로젝트를 가리킵니다: %s"
                    context project-id))))))

(defun imoogi-project-notes--entry-from-category (category candidates)
  "Return the unique CANDIDATES entry named CATEGORY."
  (when (and category (not (string-empty-p category)))
    (let ((matches (imoogi-project-notes--unique-entries
                    (cl-remove-if-not
                     (lambda (candidate)
                       (string= category
                                (imoogi-project-notes--entry-name candidate)))
                     candidates))))
      (cond
       ((= (length matches) 1) (car matches))
       ((> (length matches) 1)
        (user-error "작업 CATEGORY가 중복 프로젝트를 가리킵니다: %s"
                    category))))))

(defun imoogi-project-notes--entry-at-refile-target (rfloc candidates)
  "Return destination entry from RFLOC heading metadata among CANDIDATES."
  (let ((pos (nth 3 rfloc)))
    (when pos
      (goto-char (cond
                  ((markerp pos) (marker-position pos))
                  ((integerp pos) pos)
                  (t (point-min))))
      (unless (org-at-heading-p)
        (condition-case nil
            (org-back-to-heading t)
          (error nil)))
      (when (org-at-heading-p)
        (or (imoogi-project-notes--entry-from-project-id
             (org-entry-get (point) "IMOOGI_PROJECT_ID")
             candidates "대상 작업")
            (imoogi-project-notes--entry-from-category
             (org-get-category) candidates))))))

(defun imoogi-project-notes--refile-destination-entry (rfloc)
  "Return the project entry owning RFLOC's destination."
  (when-let* ((file (nth 1 rfloc)))
    (or (imoogi-project-notes--find-entry-by-notes-file file)
        (let ((candidates (imoogi-project-notes--entries-for-tasks-file file)))
          (cond
           ((= (length candidates) 1) (car candidates))
           ((> (length candidates) 1)
            (or (with-current-buffer (find-file-noselect file)
                  (save-excursion
                    (imoogi-project-notes--entry-at-refile-target
                     rfloc candidates)))
                (imoogi-project-notes--select-task-entry-candidate
                 candidates "대상 작업 프로젝트: "))))))))

(defun imoogi-project-notes--copy-action-for-refile (copy-requested)
  "Return refile guard action for COPY-REQUESTED."
  (if copy-requested
      'copy
    (if (and noninteractive
             (subrp (symbol-function 'completing-read)))
        'cancel
    (let* ((choices '(("copy" . copy) ("cancel" . cancel)))
           (choice (completing-read
                    "연결된 프로젝트 항목은 이동하지 않습니다. 동작: "
                    choices nil t nil nil "cancel")))
      (cdr (assoc choice choices))))))

(defun imoogi-project-notes--clock-refile-location ()
  "Return an RFLOC-like destination for the running clock."
  (unless (and (boundp 'org-clock-marker)
               (markerp org-clock-marker)
               (marker-buffer org-clock-marker))
    (user-error "실행 중인 clock 대상이 없습니다"))
  (let* ((marker (copy-marker org-clock-marker))
         (buffer (marker-buffer marker))
         (file (buffer-file-name buffer)))
    (unless file
      (user-error "실행 중인 clock 대상 파일을 찾지 못했습니다"))
    (with-current-buffer buffer
      (save-excursion
        (goto-char marker)
        (org-back-to-heading t)
        (list (org-get-heading t t t t) file nil marker)))))

(defun imoogi-project-notes--refile-target-for-arg (arg default-buffer rfloc)
  "Return RFLOC, using ARG-specific Org destinations without prompting needlessly."
  (cond
   (rfloc rfloc)
   ((equal arg 2) (imoogi-project-notes--clock-refile-location))
   (t (org-refile-get-location "연결된 항목 대상" default-buffer nil))))

(defun imoogi-project-notes--org-refile-around
    (orig &optional arg default-buffer rfloc msg)
  "Guard `org-refile' for cross-project managed project-note entities."
  (if (member arg '(0 (4) (16) (64)))
      (funcall orig arg default-buffer rfloc msg)
    (let* ((kind (imoogi-project-notes--refile-source-kind))
           (protected (and kind
                           (imoogi-project-notes--copy-source-protected-p kind)))
           (source-entry (or (and buffer-file-name
                                   (imoogi-project-notes--find-entry-by-notes-file
                                    buffer-file-name))
                             (and (derived-mode-p 'org-mode)
                                  (condition-case nil
                                      (imoogi-project-notes--task-entry)
                                    (user-error nil)))
                             (imoogi-project-notes--current-entry)))
           (copy-requested (or (equal arg 3) org-refile-keep)))
      (if (not protected)
          (funcall orig arg default-buffer rfloc msg)
        (let* ((target (imoogi-project-notes--refile-target-for-arg
                        arg default-buffer rfloc))
               (destination-entry
                (imoogi-project-notes--refile-destination-entry target)))
          (cond
           ((not destination-entry)
            (user-error "프로젝트 밖 대상에는 연결된 항목을 이동하지 않습니다"))
           ((imoogi-project-notes--entry-equal-p source-entry destination-entry)
            (cond
             (copy-requested
              (imoogi-project-notes--copy-current-source-to-entry
               destination-entry kind))
             ((eq kind 'file)
              (user-error "문서 관계를 나누는 same-project refile은 지원하지 않습니다"))
             (t
              (funcall orig arg default-buffer target msg))))
           ((eq (imoogi-project-notes--copy-action-for-refile copy-requested)
                'copy)
            (imoogi-project-notes--copy-current-source-to-entry
             destination-entry kind))
           (t (user-error "프로젝트 항목 이동을 취소했습니다"))))))))

(defun imoogi-project-notes--org-roam-refile-around (orig node)
  "Guard `org-roam-refile' for cross-project managed project-note entities."
  (let* ((kind (imoogi-project-notes--refile-source-kind))
         (protected (and kind (imoogi-project-notes--copy-source-protected-p kind)))
         (source-entry (or (and buffer-file-name
                                 (imoogi-project-notes--find-entry-by-notes-file
                                  buffer-file-name))
                           (imoogi-project-notes--current-entry)))
         (target-file (and (fboundp 'org-roam-node-file)
                           (org-roam-node-file node)))
         (destination-entry
          (and target-file
               (imoogi-project-notes--find-entry-by-notes-file target-file))))
    (cond
     ((not protected)
      (funcall orig node))
     ((not destination-entry)
      (user-error "프로젝트 밖 org-roam 대상에는 연결된 항목을 이동하지 않습니다"))
     ((imoogi-project-notes--entry-equal-p source-entry destination-entry)
      (if (eq kind 'file)
          (user-error "문서 관계를 나누는 same-project org-roam refile은 지원하지 않습니다")
        (funcall orig node)))
     ((eq (imoogi-project-notes--copy-action-for-refile nil) 'copy)
      (imoogi-project-notes--copy-current-source-to-entry destination-entry kind))
     (t (user-error "프로젝트 항목 이동을 취소했습니다")))))

(defun imoogi-project-notes--install-refile-guards ()
  "Install narrow project-notes refile guards idempotently."
  (with-eval-after-load 'org-refile
    (unless (advice-member-p #'imoogi-project-notes--org-refile-around
                             'org-refile)
      (advice-add 'org-refile :around
                  #'imoogi-project-notes--org-refile-around)))
  (with-eval-after-load 'org-roam-node
    (unless (advice-member-p #'imoogi-project-notes--org-roam-refile-around
                             'org-roam-refile)
      (advice-add 'org-roam-refile :around
                  #'imoogi-project-notes--org-roam-refile-around))))

(defun imoogi-project-notes--append-artifact-link (artifact-id title)
  "Append a link to ARTIFACT-ID named TITLE under the current Org heading."
  (imoogi-project-notes--ensure-task-artifact-link artifact-id title))

(defun imoogi-project-notes--cache-context-current-p
    (buffer marker tick &optional entry)
  "Return non-nil when BUFFER/MARKER/TICK still name the original request."
  (and (buffer-live-p buffer)
       (markerp marker)
       (marker-buffer marker)
       (eq (window-buffer (selected-window)) buffer)
       (= tick (buffer-modified-tick buffer))
       (or (not entry)
           (when-let* ((registered
                        (imoogi-project-notes--entry-by-note-id
                         (imoogi-project-notes--entry-derived-note-id entry))))
             (and (imoogi-project-notes--entry-same-scope-p
                   entry registered)
                  (with-current-buffer buffer
                    (save-excursion
                      (goto-char marker)
                      (condition-case nil
                          (imoogi-project-notes--entry-same-scope-p
                           registered
                           (imoogi-project-notes--task-entry registered))
                        (user-error nil)))))))))

(defun imoogi-project-notes--cache-document-choices (entry response)
  "Return completing-read choices for RESPONSE documents in ENTRY."
  (let ((root (imoogi-project-notes--alist-string 'notes-dir entry)))
    (mapcar (lambda (doc)
              (cons (format "%s — %s"
                            (or (alist-get 'title doc)
                                (file-name-base (alist-get 'file doc)))
                            (file-relative-name (alist-get 'file doc) root))
                    doc))
            (alist-get 'documents response))))

(defun imoogi-project-notes--cache-read-document (entry response prompt)
  "Read a document from RESPONSE using PROMPT."
  (when (active-minibuffer-window)
    (user-error "minibuffer가 사용 중이라 프로젝트 문서 선택을 시작하지 않았습니다"))
  (let* ((choices (imoogi-project-notes--cache-document-choices entry response))
         (choice (completing-read prompt choices nil t)))
    (cdr (assoc choice choices))))

(defun imoogi-project-notes--cache-fresh-document (response selected)
  "Return SELECTED document's fresh entry from RESPONSE."
  (let ((file (and selected (alist-get 'file selected))))
    (and file
         (cl-find file (alist-get 'documents response)
                  :key (lambda (doc) (alist-get 'file doc))
                  :test #'string=))))

(defun imoogi-project-notes--cache-run-selected-document-operation
    (entry marker tick selected operation)
  "Refresh catalog, then run OPERATION for SELECTED when context is current."
  (let ((buffer (marker-buffer marker)))
    (imoogi-project-notes--cache-call-async
     entry "catalog"
     (lambda (fresh)
       (if (not (imoogi-project-notes--cache-context-current-p
                 buffer marker tick entry))
           (message "프로젝트 문서 캐시 결과를 버렸습니다: 원본 버퍼가 바뀌었습니다")
         (let ((doc (imoogi-project-notes--cache-fresh-document fresh selected))
               (imoogi-project-notes--document-cache-snapshot fresh)
               (imoogi-project-notes--snapshot-live-record-cache
                (make-hash-table :test #'equal)))
           (with-current-buffer buffer
             (save-excursion
               (goto-char marker)
               (funcall operation doc)))))))))

(defun imoogi-project-notes--link-artifact-async (entry)
  "Start asynchronous document selection and link it to the current task."
  (let ((buffer (current-buffer))
        (marker (copy-marker (point)))
        (tick (buffer-modified-tick)))
    (imoogi-project-notes--cache-call-async
     entry "catalog"
     (lambda (response)
       (if (not (imoogi-project-notes--cache-context-current-p
                 buffer marker tick entry))
           (message "프로젝트 문서 캐시 결과를 버렸습니다: 원본 버퍼가 바뀌었습니다")
         (let ((doc (imoogi-project-notes--cache-read-document
                     entry response "연결할 프로젝트 문서: ")))
           (imoogi-project-notes--cache-run-selected-document-operation
            entry marker tick doc
            (lambda (fresh-doc)
              (imoogi-project-notes--link-current-task-to-document
               (alist-get 'file fresh-doc) entry)))))))))

;;;###autoload
(defun imoogi-project-notes-link-artifact (&optional document entry)
  "Link the current TODO to an existing project DOCUMENT."
  (interactive)
  (let* ((entry (imoogi-project-notes--task-entry entry))
         (doc (cond
               ((stringp document)
                `((file . ,(expand-file-name document))))
               ((and document (listp document)) document)
               ((called-interactively-p 'interactive)
                (if (imoogi-project-notes--cache-available-p)
                    (progn
                      (imoogi-project-notes--link-artifact-async entry)
                      nil)
                  (user-error
                   "imoogi-notes 실행 파일이 없습니다. 저장소에서 make build-notes를 실행하세요")))
               (t (imoogi-project-notes--select-document
                   entry "연결할 프로젝트 문서: " t)))))
    (when doc
      (unless (alist-get 'file doc)
        (signal 'quit nil))
      (imoogi-project-notes--link-current-task-to-document
       (alist-get 'file doc) entry))))

;;;###autoload
(defun imoogi-project-notes-unlink-artifact (&optional document entry)
  "Remove the relation between the current TODO and DOCUMENT."
  (interactive)
  (let* ((entry (imoogi-project-notes--task-entry entry))
         (doc (or document
                  (imoogi-project-notes--select-linked-document entry))))
    (imoogi-project-notes--unlink-current-task-from-document
     (alist-get 'id doc) (alist-get 'file doc) entry)))

;;;###autoload
(defun imoogi-project-notes-insert-link (&optional document entry)
  "Insert an ID link to a project DOCUMENT at point without creating a relation."
  (interactive)
  (setq-local case-fold-search nil)
  (cl-labels
      ((source-entry-strict
        (expected)
        (let* ((notes-entry
                (and buffer-file-name
                     (imoogi-project-notes--find-entry-by-notes-file
                      buffer-file-name)))
               (task-entries
                (and buffer-file-name
                     (imoogi-project-notes--entries-for-tasks-file
                      buffer-file-name)))
               (actual
                (cond
                 (task-entries
                  (imoogi-project-notes--task-entry expected))
                 (notes-entry notes-entry)
                 (expected
                  (user-error "문서 링크 삽입 위치가 선택한 프로젝트 문서 안이 아닙니다: %s"
                              (or buffer-file-name (buffer-name)))))))
          (when (and expected actual
                     (not (imoogi-project-notes--entry-equal-p
                           expected actual)))
            (user-error "문서 링크 삽입 위치의 실제 프로젝트와 선택한 프로젝트가 다릅니다"))
          (or expected actual
              (user-error "문서 링크 삽입 위치의 프로젝트 소유자를 확인할 수 없습니다: %s"
                          (or buffer-file-name (buffer-name))))))
       (save-prepared-document-id
        (actual-entry doc-buffer doc-id doc-title doc-preimage)
        (imoogi-project-notes--run-link-operation
         'insert-link
         `((operation . insert-link)
           (entry-note-id . ,(imoogi-project-notes--entry-derived-note-id
                              actual-entry))
           (entry-instance-id . ,(alist-get 'instance-id actual-entry))
           (doc-file . ,(buffer-file-name doc-buffer))
           (doc-id . ,doc-id)
           (doc-title . ,doc-title))
         (imoogi-project-notes--changed-endpoints doc-preimage))
        (when doc-id
          (org-id-add-location doc-id (buffer-file-name doc-buffer)))))
    (let* ((entry (save-mark-and-excursion
                    (source-entry-strict entry)))
           (doc (cond
                 ((stringp document)
                  `((file . ,(expand-file-name document))))
                 ((and document (listp document)) document)
                 (t (imoogi-project-notes--select-document
                     entry "삽입할 프로젝트 문서 링크: " t)))))
      (unless (alist-get 'file doc)
        (signal 'quit nil))
      (setf (alist-get 'file doc)
            (imoogi-project-notes--validate-document-target-file
             entry (alist-get 'file doc) "문서 링크 삽입"))
      (imoogi-project-notes--ensure-entry-mutable entry "문서 링크 삽입")
      (when-let* ((visited (find-buffer-visiting (alist-get 'file doc))))
        (unless (verify-visited-file-modtime visited)
          (user-error "문서 링크 삽입 전에 파일을 다시 방문하거나 되돌리세요: %s"
                      (alist-get 'file doc))))
      (let* ((doc-buffer (imoogi-project-notes--file-buffer
                          (alist-get 'file doc)))
               doc-id doc-title doc-preimage)
          (with-current-buffer doc-buffer
            (when-let* ((existing (imoogi-project-notes--document-identity)))
              (imoogi-project-notes--document-by-id entry (car existing)))
            (setq doc-preimage
                  (imoogi-project-notes--endpoint-preimage
                   'doc doc-buffer "문서 링크 삽입"))
            (pcase-let ((`(,id ,title ,_kind)
                         (imoogi-project-notes--document-identity 'create)))
              (setq doc-id id
                    doc-title title)
              (imoogi-project-notes--document-by-id entry doc-id)))
          (save-prepared-document-id
           entry doc-buffer doc-id doc-title doc-preimage)
          (insert (format "[[id:%s][%s]]" doc-id doc-title))))))

;;;###autoload
(defun imoogi-project-notes-repair-link (old-id new-document &optional entry)
  "Retarget current task relation from OLD-ID to NEW-DOCUMENT."
  (interactive
   (let* ((entry (imoogi-project-notes--task-entry))
          (old-id (completing-read "교체할 누락 ID: "
                                   (imoogi-project-notes--task-linked-doc-ids)
                                   nil t))
          (new-doc (imoogi-project-notes--select-document
                    entry "새 문서 대상: " t)))
     (list old-id new-doc entry)))
  (let* ((entry (imoogi-project-notes--task-entry entry))
         (new-doc (if (stringp new-document)
                      `((file . ,(expand-file-name new-document)))
                    new-document)))
    (unless (alist-get 'file new-doc)
      (signal 'quit nil))
    (imoogi-project-notes--retarget-current-task-document
     old-id (alist-get 'file new-doc) entry)))

;;;###autoload
(defun imoogi-project-notes-retry-link-operation ()
  "Retry the last pending project document relation operation."
  (interactive)
  (unless imoogi-project-notes--pending-link-operation
    (user-error "재시도할 project-notes 링크 작업이 없습니다"))
  (let* ((operation imoogi-project-notes--pending-link-operation)
         (endpoints (alist-get 'endpoints operation))
         (task-file (alist-get 'task-file operation))
         (doc-file (alist-get 'doc-file operation))
         (doc-id (alist-get 'doc-id operation))
         (entry-note-id (alist-get 'entry-note-id operation))
         (entries (imoogi-project-notes--all-entries))
         (entry (or (cl-find-if
                     (lambda (candidate)
                       (and entry-note-id
                            (string= entry-note-id
                                     (imoogi-project-notes--entry-derived-note-id
                                      candidate))))
                     entries)
                    (and task-file
                         (imoogi-project-notes--find-entry-by-notes-file
                          task-file entries))
                    (and doc-file
                         (imoogi-project-notes--find-entry-by-notes-file
                          doc-file entries))
                    (when-let* ((new-doc-file
                                 (alist-get 'new-doc-file operation)))
                      (imoogi-project-notes--find-entry-by-notes-file
                       new-doc-file entries)))))
    (unless entry
      (user-error "재시도할 프로젝트 항목을 찾지 못했습니다"))
    (if endpoints
        (progn
          (imoogi-project-notes--retry-link-operation-save operation)
          (when (eq (alist-get 'operation operation) 'insert-link)
            (message "문서 ID 저장을 완료했습니다. 원래 위치에서 `imoogi-project-notes-insert-link'를 다시 실행해 링크를 삽입하세요.")))
      (unless task-file
        (user-error "이전 형식의 링크 재시도 작업은 자동 재시도할 수 없습니다"))
      (with-current-buffer (imoogi-project-notes--file-buffer task-file)
        (goto-char (point-min))
        (imoogi-project-notes--goto-heading-id (alist-get 'task-id operation))
        (pcase (alist-get 'operation operation)
          ('link
           (imoogi-project-notes--link-current-task-to-document doc-file entry))
          ('unlink
           (imoogi-project-notes--unlink-current-task-from-document
            doc-id doc-file entry))
          ('retarget
           (imoogi-project-notes--retarget-current-task-document
            (alist-get 'old-doc-id operation)
            (alist-get 'new-doc-file operation)
            entry))
          (_ (user-error "알 수 없는 링크 재시도 작업입니다: %s"
                         (alist-get 'operation operation))))))))

;;;###autoload
(defun imoogi-project-notes-create-artifact (kind title &optional entry)
  "Create a KIND artifact named TITLE and link it to the current TODO.
The command assigns stable Org IDs to both sides, saves the task link, and
opens the new file below the project's artifacts directory."
  (interactive
   (progn
     (unless (derived-mode-p 'org-mode)
       (user-error "Org TODO heading에서 실행하세요"))
     (org-back-to-heading t)
    (let* ((entry (or (imoogi-project-notes--current-entry)
                      (imoogi-project-notes--select-entry "산출물 프로젝트: ")))
           (kind (imoogi-project-notes--artifact-kind-from-decision-tree entry))
            (default-title (org-get-heading t t t t)))
       (list kind (read-string "산출물 제목: " default-title) entry))))
  (unless (derived-mode-p 'org-mode)
    (user-error "Org TODO heading에서 실행하세요"))
  (org-back-to-heading t)
  (let ((entry (imoogi-project-notes--task-entry entry)))
      (imoogi-project-notes--ensure-entry-mutable entry "산출물 생성")
      (let* ((spec (imoogi-project-notes--artifact-spec kind entry))
             (task-title (org-get-heading t t t t))
             (artifact-id (org-id-new))
             (notes-dir (imoogi-project-notes--alist-string 'notes-dir entry))
             (artifact-dir (expand-file-name "artifacts/" notes-dir))
             (file (imoogi-project-notes--unique-artifact-file
                    artifact-dir (nth 2 spec) title))
             (project-name (imoogi-project-notes--entry-name entry))
             (task-buffer (current-buffer))
             task-id content doc-buffer task-preimage doc-preimage created)
        (imoogi-project-notes--ensure-existing-task-id-valid entry)
        (setq task-preimage
              (imoogi-project-notes--endpoint-preimage
               'task task-buffer "산출물 생성"))
        (setq task-id (org-id-get-create))
        (imoogi-project-notes--ensure-task-project-owner entry)
        (setq content
              (imoogi-project-notes--template
               "artifact"
               `(("ARTIFACT_TITLE" . ,title)
                 ("ARTIFACT_TYPE" . ,(nth 1 spec))
                 ("ARTIFACT_ID" . ,artifact-id)
                 ("TASK_ID" . ,task-id)
                 ("TASK_TITLE" . ,task-title)
                 ("PROJECT_NAME" . ,project-name)
                 ("ARTIFACT_SECTIONS" . ,(nth 3 spec)))))
        (condition-case err
            (progn
              (imoogi-project-notes--exclusive-create-file file content)
              (setq created t
                    doc-buffer (imoogi-project-notes--file-buffer file))
              (with-current-buffer doc-buffer
                (setq doc-preimage
                      (imoogi-project-notes--endpoint-preimage
                       'doc doc-buffer "산출물 생성")))
              (with-current-buffer task-buffer
                (imoogi-project-notes--append-artifact-link artifact-id title))
              (imoogi-project-notes--save-prepared-relation
               'link entry task-buffer doc-buffer task-id artifact-id
               task-title title task-preimage doc-preimage)
              (imoogi-project-notes--find-file
               entry file (imoogi-project-notes--current-source-root entry)))
          (error
           (when (and created
                      (file-exists-p file)
                      (string= (imoogi-project-notes--file-string-hash file)
                               (secure-hash 'sha1 content))
                      (not imoogi-project-notes--pending-link-operation))
             (delete-file file))
           (when (and (not imoogi-project-notes--pending-link-operation)
                      (eq (imoogi-project-notes--endpoint-disk-state
                           task-preimage)
                          'pre))
             (imoogi-project-notes--restore-endpoint-preimage-for-preparation
              task-preimage))
           (signal (car err) (cdr err)))))))

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

(imoogi-project-notes--install-refile-guards)

(provide 'imoogi-project-notes)
;;; 26-project-notes.el ends here
