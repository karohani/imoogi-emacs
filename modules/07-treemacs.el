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

(provide 'imoogi-treemacs)
;;; treemacs.el ends here
