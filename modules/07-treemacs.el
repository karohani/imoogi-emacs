;;; treemacs.el --- Treemacs & related packages -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "07-treemacs" 'treemacs 'treemacs-icons-dired
                'treemacs-magit 'bookmark 'button 'cl-lib 'imenu 'subr-x)

(require 'bookmark)
(require 'button)
(require 'cl-lib)
(require 'imenu)
(require 'subr-x)

(defconst imoogi-treemacs-bookmarks-buffer-name "*Imoogi Bookmarks*"
  "Buffer used for the IntelliJ-style Bookmarks tool window.")

(defconst imoogi-treemacs-structure-buffer-name "*Imoogi Structure*"
  "Buffer used for the IntelliJ-style Structure tool window.")

(defvar-local imoogi-treemacs--source-buffer nil
  "Editor buffer represented by the current tool window.")

(define-derived-mode imoogi-treemacs-tool-window-mode special-mode "Tool-Window"
  "Base mode for lightweight IntelliJ-style tool windows."
  (setq-local truncate-lines t)
  (setq-local cursor-type nil)
  (local-set-key (kbd "q") #'imoogi-treemacs-hide-tool-window))

(defun imoogi-treemacs--custom-tool-window (kind)
  "Return the visible custom tool window of KIND, if any."
  (let ((expected-buffer
         (get-buffer
          (pcase kind
            ('bookmarks imoogi-treemacs-bookmarks-buffer-name)
            ('structure imoogi-treemacs-structure-buffer-name)
            (_ "")))))
    (and expected-buffer
         (cl-find-if
          (lambda (window)
            (and (eq (window-parameter window 'imoogi-treemacs-tool-window)
                     kind)
                 (eq (window-buffer window) expected-buffer)))
          (window-list (selected-frame) 'no-minibuffer)))))

(defun imoogi-treemacs--editor-window (&optional preferred)
  "Return a non-side editor window, preferring PREFERRED when suitable."
  (let ((selected (selected-window)))
    (cond
     ((and (window-live-p preferred)
           (not (window-parameter preferred 'window-side)))
      preferred)
     ((and (not (window-parameter selected 'window-side))
           (not (window-dedicated-p selected)))
      selected)
     ((get-mru-window (selected-frame) nil t t))
     (t selected))))

(defun imoogi-treemacs--editor-buffer ()
  "Return the editor buffer associated with the current frame."
  (window-buffer (imoogi-treemacs--editor-window)))

(defun imoogi-treemacs-hide-tool-window ()
  "Hide Treemacs and the lightweight Bookmarks/Structure tool windows."
  (interactive)
  (when-let ((window (and (fboundp 'treemacs-get-local-window)
                          (treemacs-get-local-window))))
    (when (window-live-p window)
      (delete-window window)))
  (dolist (window (window-list (selected-frame) 'no-minibuffer))
    (when (window-parameter window 'imoogi-treemacs-tool-window)
      (delete-window window))))

(defun imoogi-treemacs--display-tool-buffer (buffer kind)
  "Display BUFFER as tool window KIND in Treemacs' left-side slot."
  (let ((window
         (display-buffer-in-side-window
          buffer
          `((side . left)
            (slot . -1)
            (window-width . ,(if (boundp 'treemacs-width)
                                 treemacs-width
                               35))
            (window-parameters
             . ((imoogi-treemacs-tool-window . ,kind)
                (no-delete-other-windows . t)))))))
    (set-window-dedicated-p window t)
    (select-window window)
    window))

(defun imoogi-treemacs-toggle-file-tree ()
  "Focus or close the Treemacs file tree like IntelliJ's Command-1 window.
When Treemacs is visible but not selected, focus it.  When it is already
selected, close it.  Otherwise replace any custom tool window and show it."
  (interactive)
  (require 'treemacs)
  (let ((window (treemacs-get-local-window)))
    (cond
     ((and (window-live-p window)
           (eq (selected-window) window))
      (treemacs-quit))
     ((window-live-p window)
      (treemacs-select-window))
     (t
      (imoogi-treemacs-hide-tool-window)
      (treemacs-select-window)))))

(defun imoogi-treemacs--bookmark-button-action (button)
  "Visit the bookmark represented by BUTTON in an editor window."
  (let ((bookmark-name (button-get button 'imoogi-bookmark-name))
        (target-window
         (imoogi-treemacs--editor-window
          (button-get button 'imoogi-editor-window))))
    (bookmark-jump
     bookmark-name
     (lambda (buffer &rest _)
       (set-window-buffer target-window buffer)
       (select-window target-window)
       target-window))))

(defun imoogi-treemacs--render-bookmarks (buffer _source-buffer)
  "Render Emacs bookmarks into BUFFER."
  (bookmark-maybe-load-default-file)
  (let ((editor-window (imoogi-treemacs--editor-window)))
    (with-current-buffer buffer
      (let ((inhibit-read-only t))
        (erase-buffer)
        (imoogi-treemacs-tool-window-mode)
        (setq-local header-line-format " Bookmarks")
        (local-set-key (kbd "g") #'imoogi-treemacs-refresh-bookmarks)
        (if bookmark-alist
            (dolist (bookmark bookmark-alist)
              (let* ((name (car bookmark))
                     (location
                      (condition-case nil
                          (format "%s" (or (bookmark-location bookmark) ""))
                        (error ""))))
                (insert-text-button
                 name
                 'follow-link t
                 'help-echo location
                 'imoogi-bookmark-name name
                 'imoogi-editor-window editor-window
                 'action #'imoogi-treemacs--bookmark-button-action)
                (unless (string-empty-p location)
                  (insert (propertize (format "\n  %s" location)
                                      'face 'shadow)))
                (insert "\n")))
          (insert (propertize "저장된 북마크가 없습니다.\n"
                              'face 'shadow)))
        (goto-char (point-min))))))

(defun imoogi-treemacs-refresh-bookmarks ()
  "Refresh the currently displayed Bookmarks tool window."
  (interactive)
  (imoogi-treemacs--render-bookmarks (current-buffer) nil))

(defun imoogi-treemacs-toggle-bookmarks ()
  "Toggle the bookmark list like IntelliJ's Command-2 tool window."
  (interactive)
  (let ((buffer (get-buffer-create imoogi-treemacs-bookmarks-buffer-name)))
    (if-let ((window (imoogi-treemacs--custom-tool-window 'bookmarks)))
        (delete-window window)
      (let ((source-buffer (imoogi-treemacs--editor-buffer)))
        (imoogi-treemacs-hide-tool-window)
        (imoogi-treemacs--render-bookmarks buffer source-buffer)
        (imoogi-treemacs--display-tool-buffer buffer 'bookmarks)))))

(defun imoogi-treemacs--structure-button-action (button)
  "Jump to the Imenu item represented by BUTTON."
  (let ((source-buffer (button-get button 'imoogi-source-buffer))
        (index-item (button-get button 'imoogi-imenu-item))
        (target-window
         (imoogi-treemacs--editor-window
          (button-get button 'imoogi-editor-window))))
    (unless (buffer-live-p source-buffer)
      (user-error "원본 버퍼가 더 이상 존재하지 않습니다"))
    (select-window target-window)
    (switch-to-buffer source-buffer)
    (imenu index-item)
    (recenter)))

(defun imoogi-treemacs--insert-structure-items
    (items depth source-buffer editor-window)
  "Insert Imenu ITEMS at DEPTH for SOURCE-BUFFER and EDITOR-WINDOW."
  (dolist (item items)
    (when (and (consp item)
               (not (equal (car item) "*Rescan*")))
      (if (imenu--subalist-p item)
          (progn
            (insert (make-string (* depth 2) ?\s)
                    (propertize (car item) 'face 'font-lock-keyword-face)
                    "\n")
            (imoogi-treemacs--insert-structure-items
             (cdr item) (1+ depth) source-buffer editor-window))
        (insert (make-string (* depth 2) ?\s))
        (insert-text-button
         (format "%s" (car item))
         'follow-link t
         'imoogi-source-buffer source-buffer
         'imoogi-editor-window editor-window
         'imoogi-imenu-item item
         'action #'imoogi-treemacs--structure-button-action)
        (insert "\n")))))

(defun imoogi-treemacs--render-structure (buffer source-buffer)
  "Render SOURCE-BUFFER's Imenu tree into BUFFER."
  (let ((editor-window (imoogi-treemacs--editor-window))
        (index
         (when (buffer-live-p source-buffer)
           (with-current-buffer source-buffer
             (cl-remove-if-not
              (lambda (item)
                (and (consp item)
                     (not (equal (car item) "*Rescan*"))))
              (imenu--make-index-alist t))))))
    (with-current-buffer buffer
      (let ((inhibit-read-only t))
        (erase-buffer)
        (imoogi-treemacs-tool-window-mode)
        (setq-local imoogi-treemacs--source-buffer source-buffer)
        (setq-local header-line-format
                    (format " Structure — %s" (buffer-name source-buffer)))
        (local-set-key (kbd "g") #'imoogi-treemacs-refresh-structure)
        (if index
            (imoogi-treemacs--insert-structure-items
             index 0 source-buffer editor-window)
          (insert (propertize "이 버퍼에는 Structure 정보가 없습니다.\n"
                              'face 'shadow)))
        (goto-char (point-min))))))

(defun imoogi-treemacs-refresh-structure ()
  "Refresh the currently displayed Structure tool window."
  (interactive)
  (unless (buffer-live-p imoogi-treemacs--source-buffer)
    (user-error "원본 버퍼가 더 이상 존재하지 않습니다"))
  (imoogi-treemacs--render-structure
   (current-buffer) imoogi-treemacs--source-buffer))

(defun imoogi-treemacs-toggle-structure ()
  "Toggle the current buffer structure like IntelliJ's Command-7 window."
  (interactive)
  (if-let ((window (imoogi-treemacs--custom-tool-window 'structure)))
      (delete-window window)
    (let ((source-buffer (imoogi-treemacs--editor-buffer))
          (buffer (get-buffer-create imoogi-treemacs-structure-buffer-name)))
      (imoogi-treemacs-hide-tool-window)
      (imoogi-treemacs--render-structure buffer source-buffer)
      (imoogi-treemacs--display-tool-buffer buffer 'structure))))

(use-package treemacs
  :ensure t
  :defer t
  :init
  (with-eval-after-load 'winum
    (define-key winum-keymap (kbd "M-0") #'treemacs-select-window))
  :config
  (progn
    (setq treemacs-buffer-name-function            #'treemacs-default-buffer-name
          treemacs-buffer-name-prefix              " *Treemacs-Buffer-"
          treemacs-collapse-dirs                   (if treemacs-python-executable 3 0)
          treemacs-deferred-git-apply-delay        0.5
          treemacs-directory-name-transformer      #'identity
          treemacs-display-in-side-window          t
          treemacs-eldoc-display                   'simple
          treemacs-file-event-delay                2000
          treemacs-file-extension-regex            treemacs-last-period-regex-value
          treemacs-file-follow-delay               0.2
          treemacs-file-name-transformer           #'identity
          treemacs-follow-after-init               t
          treemacs-expand-after-init               t
          treemacs-find-workspace-method           'find-for-file-or-pick-first
          treemacs-git-command-pipe                ""
          treemacs-goto-tag-strategy               'refetch-index
          treemacs-header-scroll-indicators        '(nil . "^^^^^^")
          treemacs-hide-dot-git-directory          t
          treemacs-hide-dot-jj-directory           t
          treemacs-indentation                     2
          treemacs-indentation-string              " "
          treemacs-is-never-other-window           nil
          treemacs-max-git-entries                 5000
          treemacs-missing-project-action          'ask
          treemacs-move-files-by-mouse-dragging    t
          treemacs-move-forward-on-expand          nil
          treemacs-no-png-images                   nil
          treemacs-no-delete-other-windows         t
          treemacs-project-follow-cleanup          nil
          treemacs-persist-file                    (expand-file-name ".cache/treemacs-persist" user-emacs-directory)
          treemacs-position                        'left
          treemacs-read-string-input               'from-child-frame
          treemacs-recenter-distance               0.1
          treemacs-recenter-after-file-follow      nil
          treemacs-recenter-after-tag-follow       nil
          treemacs-recenter-after-project-jump     'always
          treemacs-recenter-after-project-expand   'on-distance
          treemacs-litter-directories              '("/node_modules" "/.venv" "/.cask")
          treemacs-project-follow-into-home        nil
          treemacs-show-cursor                     nil
          treemacs-show-hidden-files               t
          treemacs-silent-filewatch                nil
          treemacs-silent-refresh                  nil
          treemacs-sorting                         'alphabetic-asc
          treemacs-select-when-already-in-treemacs 'move-back
          treemacs-space-between-root-nodes        t
          treemacs-tag-follow-cleanup              t
          treemacs-tag-follow-delay                1.5
          treemacs-text-scale                      nil
          treemacs-user-mode-line-format           nil
          treemacs-user-header-line-format         nil
          treemacs-wide-toggle-width               70
          treemacs-width                           35
          treemacs-width-increment                 1
          treemacs-width-is-initially-locked       t
          treemacs-workspace-switch-cleanup        nil)

    ;; The default width and height of the icons is 22 pixels. If you are
    ;; using a Hi-DPI display, uncomment this to double the icon size.
    ;;(treemacs-resize-icons 44)

    (treemacs-follow-mode t)
    (treemacs-filewatch-mode t)
    (treemacs-fringe-indicator-mode 'always)
    (when treemacs-python-executable
      (treemacs-git-commit-diff-mode t))

    (pcase (cons (not (null (executable-find "git")))
                 (not (null treemacs-python-executable)))
      (`(t . t)
       (treemacs-git-mode 'deferred))
      (`(t . _)
       (treemacs-git-mode 'simple)))

    (treemacs-hide-gitignored-files-mode nil))
  :bind
  (:map global-map
        ("s-1"       . imoogi-treemacs-toggle-file-tree)
        ("s-2"       . imoogi-treemacs-toggle-bookmarks)
        ("s-7"       . imoogi-treemacs-toggle-structure)
        ("M-0"       . treemacs-select-window)
        ("C-x t 1"   . treemacs-delete-other-windows)
        ("C-x t t"   . imoogi-treemacs-toggle-file-tree)
        ("C-x t p"   . treemacs-add-and-display-current-project)
        ("C-x t d"   . treemacs-select-directory)
        ("C-x t B"   . treemacs-bookmark)
        ("C-x t C-t" . treemacs-find-file)
        ("C-x t M-t" . treemacs-find-tag)
   :map treemacs-mode-map
        ("<escape>"  . treemacs-select-window)
        ("s-1"       . imoogi-treemacs-toggle-file-tree)))

(use-package treemacs-icons-dired
  :hook (dired-mode . treemacs-icons-dired-enable-once)
  :ensure t)

(use-package treemacs-magit
  :after (treemacs magit)
  :ensure t)

;; (treemacs-persp / treemacs-evil 은 persp-mode / evil 용이라 imoogi 스택
;;  (perspective + 키바인딩)과 맞지 않아 제거했다. treemacs 는 전역으로 동작.)

;; Treemacs 는 더 이상 시작 시 자동으로 열리지 않는다. 필요할 때
;; M-0 / <s-1> / C-x t t 로 직접 연다 (imoogi-treemacs-toggle-file-tree).

;;; ------------------------------------------------------------------
;;; 프로젝트 유일성 — 한 폴더는 한 workspace 에만
;;;
;;; treemacs 는 원래 중복을 허용한다: workspace 는 서로 독립적인 "뷰"라는 게
;;; 원설계라, 같은 저장소를 여러 workspace 에 두는 것이 버그가 아니다. 그래서
;;; 이 유일성은 treemacs 가 주는 보장이 아니라 **이 설정이 얹는 규칙**이고,
;;; 우리가 지켜야 한다.
;;;
;;; 규칙이 필요한 이유는 아래 perspective 연동이다. 한 프로젝트가 두 workspace 에
;;; 걸쳐 있으면 "이 프로젝트의 workspace" 라는 질문에 답이 둘이라, 어디로 전환할지
;;; 정할 수 없다(실측: moai-adk 가 StockTrader 와 MOAI-ADK Research 양쪽에 있었다).

(defvar imoogi-treemacs-unique-projects t
  "non-nil 이면 한 경로가 두 workspace 에 동시에 들어가는 것을 막는다.")

(defun imoogi-treemacs--canonical (path)
  "PATH 를 비교 가능한 표준형으로 바꾼다. nil 이면 nil."
  (when path
    (directory-file-name (file-truename (expand-file-name path)))))

(defun imoogi-treemacs--workspace-has-p (workspace path)
  "WORKSPACE 가 PATH 를 프로젝트로 담고 있으면 non-nil."
  (let ((target (imoogi-treemacs--canonical path)))
    (seq-some (lambda (project)
                (equal target (imoogi-treemacs--canonical
                               (treemacs-project->path project))))
              (treemacs-workspace->projects workspace))))

(defun imoogi-treemacs--workspaces-with (path &optional exclude)
  "PATH 를 담은 workspace 목록. EXCLUDE 로 준 workspace 는 제외한다."
  (seq-filter (lambda (workspace)
                (and (not (eq workspace exclude))
                     (imoogi-treemacs--workspace-has-p workspace path)))
              (treemacs-workspaces)))

(defun imoogi-treemacs-check-duplicates ()
  "두 개 이상의 workspace 에 걸친 프로젝트를 찾아 보고한다.
advice 가 새 중복을 막아도 이미 들어 있던 것은 남으므로, 점검 수단이 따로 필요하다."
  (interactive)
  (let ((table (make-hash-table :test 'equal))
        (duplicates nil))
    (dolist (workspace (treemacs-workspaces))
      (dolist (project (treemacs-workspace->projects workspace))
        (let ((key (imoogi-treemacs--canonical (treemacs-project->path project))))
          (push (treemacs-workspace->name workspace) (gethash key table)))))
    (maphash (lambda (path names)
               (when (> (length names) 1)
                 (push (cons path (nreverse names)) duplicates)))
             table)
    (if (null duplicates)
        (message "treemacs: 중복 없음 — 모든 프로젝트가 workspace 하나에만 속합니다.")
      (with-current-buffer (get-buffer-create "*treemacs 프로젝트 중복*")
        (erase-buffer)
        (insert "여러 workspace 에 걸친 프로젝트\n\n")
        (dolist (entry duplicates)
          (insert (format "  %s\n      %s\n\n"
                          (abbreviate-file-name (car entry))
                          (string-join (cdr entry) " / "))))
        (insert "정리 방법: 남길 workspace 가 아닌 쪽으로 이동한 뒤,\n"
                "트리에서 그 프로젝트에 커서를 두고\n"
                "  M-x treemacs-remove-project-from-workspace\n")
        (goto-char (point-min))
        (display-buffer (current-buffer))))))

(defun imoogi-treemacs--reject-cross-workspace (path)
  "PATH 가 이미 다른 workspace 에 있으면 거부 사유를 돌려준다(없으면 nil).

반환 형태는 `treemacs-do-add-project-to-workspace' 의 규약을 따른다.
`duplicate-project' 대신 `invalid-path' 를 쓰는 이유가 있다 — 호출자
(treemacs-interface.el)의 duplicate-project 가지는
`(goto-char (treemacs-project->position duplicate))' 를 실행하는데, 다른
workspace 의 프로젝트는 현재 버퍼에 위치가 없어 그 자체가 오류가 된다.
invalid-path 가지는 문자열만 쓰므로 안전하다."
  (when (and imoogi-treemacs-unique-projects path)
    (when-let* ((others (imoogi-treemacs--workspaces-with
                         path (ignore-errors (treemacs-current-workspace)))))
      (list 'invalid-path
            (format "이미 '%s' workspace 에 있습니다 — 한 폴더는 한 workspace 규칙"
                    (string-join (mapcar #'treemacs-workspace->name others) " / "))))))

;; 길목이 하나라 여기만 막으면 된다: 대화식 추가, project-follow,
;; add-and-display-current-project 등 7개 호출 지점이 전부 이 함수를 지난다(실측).
(define-advice treemacs-do-add-project-to-workspace
    (:around (original path name) imoogi-unique-projects)
  "다른 workspace 에 이미 있는 경로면 추가를 거부한다."
  (or (imoogi-treemacs--reject-cross-workspace path)
      (funcall original path name)))

;;; ------------------------------------------------------------------
;;; perspective 작업공간 → treemacs workspace 연동
;;;
;;; 이름이 아니라 **경로**로 잇는다. treemacs 가 자기 workspace 안에 프로젝트
;;; 경로를 이미 갖고 있으므로, 설정 파일에 머신별 매핑 표를 둘 필요가 없다 —
;;; 저장소를 다른 머신에 클론해도 그 머신의 treemacs-persist 를 읽을 뿐이다.
;;;
;;; (treemacs-persp 패키지는 persp-mode 용이라 이 스택(perspective)과 맞지 않아
;;;  위에서 제외했다. 그래서 이 연결을 직접 만든다.)

(defvar imoogi-treemacs-follow-perspective t
  "non-nil 이면 작업공간을 바꿀 때 treemacs workspace 도 따라 전환한다.")

(defun imoogi-treemacs--perspective-project-root ()
  "지금 작업공간이 가리키는 프로젝트 루트. 못 찾으면 nil."
  (or (when-let* ((project (and (fboundp 'project-current) (project-current nil))))
        (project-root project))
      ;; 작업공간을 막 만들어 *scratch* 만 있는 경우엔 위가 nil 이다. 그럴 때는
      ;; 04-projects.el 이 유지하는 이름↔루트 표를 거꾸로 본다.
      (when (and (fboundp 'persp-current-name)
                 (boundp 'imoogi-project-perspective-alist))
        (car (rassoc (persp-current-name) imoogi-project-perspective-alist)))))

(defun imoogi-treemacs-follow-perspective-maybe ()
  "현재 작업공간의 프로젝트를 담은 treemacs workspace 로 전환한다.

세 경우에 아무것도 하지 않는다 — 모르면 가만히 두는 쪽이 기존 동작을 지킨다.
  · 작업공간이 어느 프로젝트인지 알 수 없을 때
  · 지금 workspace 가 이미 그 프로젝트를 담고 있을 때
  · 그 프로젝트를 담은 workspace 가 하나도 없을 때"
  (when (and imoogi-treemacs-follow-perspective
             (featurep 'treemacs))
    (when-let* ((root (imoogi-treemacs--perspective-project-root))
                (current (ignore-errors (treemacs-current-workspace))))
      (unless (imoogi-treemacs--workspace-has-p current root)
        (when-let* ((target (car (imoogi-treemacs--workspaces-with root current))))
          (treemacs-do-switch-workspace target))))))

(with-eval-after-load 'perspective
  (add-hook 'persp-switch-hook #'imoogi-treemacs-follow-perspective-maybe))

(provide 'imoogi-treemacs)
;;; treemacs.el ends here
