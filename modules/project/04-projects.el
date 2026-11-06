;;; projects.el --- project.el and Perspective workspaces -*- lexical-binding: t; -*-

;;; Code:
(imoogi-require "04-projects" 'project 'perspective)

(require 'cl-lib)
(require 'subr-x)

(defvar imoogi-project-perspective-alist nil
  "Canonical project roots mapped to their stable Perspective names.")

(defvar imoogi-perspective-state-file
  (locate-user-emacs-file ".cache/perspective-state.el")
  "File used to persist Perspective buffers and window layouts.")

(defvar imoogi-perspective-active-file
  (locate-user-emacs-file ".cache/perspective-active.el")
  "File used to persist the Perspective active at the previous exit.")

(defvar imoogi-perspective-treemacs-file
  (locate-user-emacs-file ".cache/perspective-treemacs.el")
  "File recording which Perspectives had a visible Treemacs window.")

(defvar imoogi-perspective--treemacs-visible nil
  "Perspective names whose Treemacs window should be restored after startup.")

(defun imoogi-perspective--treemacs-visible-p ()
  "Return non-nil if a Treemacs buffer is visible on the selected frame."
  (cl-some (lambda (window)
             (with-current-buffer (window-buffer window)
               (derived-mode-p 'treemacs-mode)))
           (window-list)))

(defun imoogi-perspective--remember-treemacs-visibility ()
  "Remember whether the Perspective being left displays Treemacs."
  (when-let ((name (persp-current-name)))
    (setq imoogi-perspective--treemacs-visible
          (delete name imoogi-perspective--treemacs-visible))
    (when (imoogi-perspective--treemacs-visible-p)
      (push name imoogi-perspective--treemacs-visible))))

(defun imoogi-perspective--treemacs-visible-names ()
  "Return names of Perspectives with a visible Treemacs window."
  (cl-loop for name in (persp-names)
           when (with-perspective name
                  (imoogi-perspective--treemacs-visible-p))
           collect name))

(defun imoogi-perspective--restore-treemacs ()
  "Replace a saved Treemacs placeholder in the current Perspective."
  (when (and (member (persp-current-name)
                     imoogi-perspective--treemacs-visible)
             (not (imoogi-perspective--treemacs-visible-p)))
    (let ((placeholder
           (cl-find-if
            (lambda (window)
              (and (eq (window-parameter window 'window-side) 'left)
                   (window-dedicated-p window)
                   (string-prefix-p "*scratch*"
                                    (buffer-name (window-buffer window)))))
            (window-list)))
          (editor-window (selected-window)))
      (unwind-protect
          (condition-case err
              (progn
                (when (window-live-p placeholder)
                  (let ((ignore-window-parameters t))
                    (delete-window placeholder)))
                (when (fboundp 'imoogi-treemacs-follow-perspective-maybe)
                  (imoogi-treemacs-follow-perspective-maybe))
                (treemacs-select-window))
            (error
             (display-warning
              'imoogi-projects
              (format "Treemacs restore failed: %s"
                      (error-message-string err))
              :warning)))
        (when (window-live-p editor-window)
          (select-window editor-window))))))

(defun imoogi-project--perspective-root ()
  "Return the project root mapped to the current Perspective, if any."
  (when (and (fboundp 'persp-current-name)
             (boundp 'imoogi-project-perspective-alist))
    (when-let ((root
                (cl-loop for (candidate . name)
                         in (symbol-value 'imoogi-project-perspective-alist)
                         when (equal name (persp-current-name))
                         return candidate)))
      (when (file-directory-p root)
        (imoogi-project--canonical-root root)))))

;;; Perspective finder
;;
;; A buffer with no project of its own (scratch, help) follows the project its
;; Perspective was opened for, so generic `project.el' commands stop asking
;; "which project?".  The finder is the last member of `project-find-functions':
;; a buffer that any other member resolves keeps its own project.

(defvar imoogi-project--perspective-finder-inhibit nil
  "Non-nil while the Perspective finder must decline every lookup.
Explicit project selection binds it, and so does the finder's own resolution
of the mapped root, which therefore cannot re-enter the finder.")

(defvar imoogi-project--explicit-lookup nil
  "Non-nil while `project-current' runs for an explicit DIRECTORY argument.
`imoogi-project--mark-explicit-lookup' binds it.  `project-current' replaces
an omitted DIRECTORY with `default-directory' before it runs the finders, so
without this marker a DIRECTORY passed by code that equals
`default-directory' would look implicit to the Perspective finder.")

(defvar imoogi-project--perspective-finder-memo nil
  "Projects the Perspective finder resolved, as an alist (KEY . PROJECT).
KEY is (PERSPECTIVE-NAME . RAW-ROOT), RAW-ROOT being the string stored in
`imoogi-project-perspective-alist', so a remap misses implicitly.
`project-try-vc' caches only positive results; without this memo a non-VC
root would be searched again on every tab-bar redisplay.  Keyed per
Perspective rather than one slot, so two frames showing different
Perspectives do not evict each other.  Cleared on every Perspective switch.")

(defvar imoogi-project--context-call-root nil
  "Context root of the running `imoogi-project-context-call', or nil.
While it is non-nil, the Perspective finder answers a lookup of exactly this
root that no other member answered with a transient project, from any buffer,
so `f', `s' and `d' never prompt when a context root exists.")

(defvar imoogi-project-perspective-finder-exclude-functions nil
  "Predicates that keep the current buffer away from the Perspective finder.
Each function is called with no arguments in the buffer being looked up; the
finder declines when any of them returns non-nil.  The finder runs on every
tab-bar redisplay, so these functions must not do file I/O.
`26-project-notes.el' adds its notes-context predicate here, which keeps this
module from requiring it.")

(defun imoogi-project--same-directory-p (left right)
  "Return non-nil when directory names LEFT and RIGHT are the same string."
  (string= (file-name-as-directory (expand-file-name left))
           (file-name-as-directory (expand-file-name right))))

(defun imoogi-project-resolve-root (root)
  "Return the project `project.el' resolves for ROOT itself.
The Perspective finder is inhibited, so this is ROOT's VC project when ROOT
is under VC, and otherwise a transient project rooted at ROOT."
  (let ((imoogi-project--perspective-finder-inhibit t))
    (or (project-current nil root)
        (cons 'transient root))))

(defun imoogi-project--perspective-find (dir)
  "Return the mapped Perspective's project for an implicit lookup of DIR.
Guards run cheapest first and each one declines: explicit lookups, Eglot,
remote names (checked before any file operation), an inactive or unmapped
Perspective, a missing root, and excluded buffers.  Inside
`imoogi-project-context-call', a lookup of its context root is answered
before the Perspective guards (the context-call clause)."
  (unless (or imoogi-project--perspective-finder-inhibit
              imoogi-project--explicit-lookup
              (bound-and-true-p eglot-lsp-context)
              ;; On the raw strings first: `expand-file-name' would already
              ;; enter the TRAMP handler.
              (file-remote-p dir)
              (file-remote-p default-directory)
              project-current-directory-override
              (not (imoogi-project--same-directory-p dir default-directory)))
    (if (and imoogi-project--context-call-root
             (imoogi-project--same-directory-p
              dir imoogi-project--context-call-root))
        ;; Every other member declined, so the root is not under VC.  This
        ;; also answers in notes buffers: `f'/`s'/`d' never consult
        ;; `imoogi-project-notes--current-source-root'.
        (cons 'transient imoogi-project--context-call-root)
      (imoogi-project--perspective-find-mapped))))

(defun imoogi-project--perspective-find-mapped ()
  "Return the current Perspective's project for the current buffer, or nil."
  ;; `persp-current-name' signals while `persp-mode' is off.
  (when-let* (((bound-and-true-p persp-mode))
              (name (persp-current-name))
              (raw-root (car (rassoc name imoogi-project-perspective-alist)))
              ((not (file-remote-p raw-root)))
              ((file-directory-p raw-root))
              ((not (run-hook-with-args-until-success
                     'imoogi-project-perspective-finder-exclude-functions))))
    (let ((key (cons name raw-root)))
      (or (cdr (assoc key imoogi-project--perspective-finder-memo))
          (let ((project (imoogi-project-resolve-root
                          (imoogi-project--canonical-root raw-root))))
            (push (cons key project) imoogi-project--perspective-finder-memo)
            project)))))

;; @MX:ANCHOR: [AUTO] every implicit `project-current' call reaches this finder
;; @MX:REASON: it runs on each tab-bar redisplay and under every project.el
;; command; an escaping error or file I/O here breaks or slows them all
(defun imoogi-project-perspective-find (dir)
  "Perspective finder for `project-find-functions'; see SPEC-PERSPPIN-001.
Resolve an implicit lookup of DIR from a buffer with no project of its own to
the project of the root the current Perspective is mapped to.  Any error
declines, because `project--find-in-directory' only absorbs
`permission-denied'."
  (condition-case nil
      (imoogi-project--perspective-find dir)
    (error nil)))

(defun imoogi-project--perspective-finder-memo-clear ()
  "Forget every project the Perspective finder memoized."
  (setq imoogi-project--perspective-finder-memo nil))

(defun imoogi-project--mark-explicit-lookup (function &optional maybe-prompt directory)
  "Call FUNCTION, `project-current', with MAYBE-PROMPT and DIRECTORY.
Bind `imoogi-project--explicit-lookup' to whether DIRECTORY was given, so the
Perspective finder declines an explicit lookup even when DIRECTORY equals
`default-directory'.  An implicit lookup nested in an explicit one binds it
back to nil."
  (let ((imoogi-project--explicit-lookup (and directory t)))
    (funcall function maybe-prompt directory)))

(defun imoogi-project-install-perspective-finder ()
  "Add the Perspective finder to `project-find-functions' at depth 95.
`add-hook' with APPEND means depth 90, and at equal depth a newer member
sorts after an older one, so 95 keeps every member added at default depth or
with APPEND, before or after this call, ahead of the finder.  95 rather than
100 leaves room for a member that deliberately wants to be last.
Also install the advice that marks explicit `project-current' lookups.
Calling this again changes nothing: both `add-hook' and `advice-add' skip a
function that is already present."
  (add-hook 'project-find-functions #'imoogi-project-perspective-find 95)
  (advice-add 'project-current :around #'imoogi-project--mark-explicit-lookup))

(defun imoogi-project-context-root ()
  "Return the root for the current Imoogi project context.

An explicitly mapped Perspective is preferred because it survives buffer
restoration and also works for study notes without a VCS root.  When no
mapping exists, fall back to Emacs' normal `project-current' detection."
  (or (imoogi-project--perspective-root)
      (when-let* ((project (and (fboundp 'project-current)
                                (project-current nil)))
                  (root (project-root project)))
        (imoogi-project--canonical-root root))))

(defun imoogi-project-context ()
  "Return the current Imoogi project context as a property list."
  (list :perspective (and (fboundp 'persp-current-name)
                          (persp-current-name))
        :root (imoogi-project-context-root)))

(defun imoogi-project-context-call (command)
  "Call interactive project COMMAND using the current Imoogi context.

The current Perspective is resolved on every invocation, so switching
Perspectives immediately changes the project used by COMMAND.  Commands that
already have a project fallback continue to use the current buffer when no
Imoogi context is mapped."
  (interactive)
  (let* ((root (plist-get (imoogi-project-context) :root))
         ;; The Perspective finder's context-call clause reads this, so
         ;; commands resolving with `(project-current t)' find ROOT even when
         ;; it is not under VC or the buffer is a notes buffer.
         (imoogi-project--context-call-root root)
         (default-directory (or root default-directory)))
    (call-interactively command)))

(defun imoogi-project--find-file-in-context (&optional include-all)
  "Find a file in the context root's project and in its notes directory.
Called by `imoogi-project-context-call'.  With INCLUDE-ALL (interactively, a
prefix argument), include every file under the directories, as
`project-find-file' does."
  (interactive "P")
  (let* ((context-root imoogi-project--context-call-root)
         ;; Route (a).  The middle branch only restates the context-call
         ;; clause for a local root; a remote root falls through to the
         ;; prompt like `s' and `d' (REQ-PIN-006).
         (project (or (project-current nil)
                      (and context-root
                           (not (file-remote-p context-root))
                           (cons 'transient context-root))
                      (project-current t)))
         (root (project-root project))
         (notes (imoogi-project--notes-directory root)))
    ;; `project-files-relative-names' stays unbound: the notes files lie
    ;; outside ROOT and must reach the reader as absolute names.
    (project-find-file-in
     (delq nil (list (and buffer-file-name
                          (project--find-default-from buffer-file-name project))
                     (thing-at-point 'filename)))
     (if notes (list root notes) (list root))
     project include-all)))

(defun imoogi-project-find-file ()
  "Find a file in the current Imoogi project context.
The candidates are the project's files plus the files of its project notes."
  (interactive)
  (imoogi-project-context-call #'imoogi-project--find-file-in-context))

(defun imoogi-project-find-regexp ()
  "Search files in the current Imoogi project context."
  (interactive)
  (imoogi-project-context-call #'project-find-regexp))

(defun imoogi-project-dired ()
  "Open Dired at the current Imoogi project context root."
  (interactive)
  (imoogi-project-context-call #'project-dired))

;;; Project transient `b': project buffers plus the project's own notes

(defun imoogi-project--notes-directory (root)
  "Return ROOT's project-notes directory, or nil.
26-project-notes.el supplies the lookup; without it, or when it fails (no
entry, a remote or missing root), the candidate lists stay project-only and
nothing is shown."
  (when (fboundp 'imoogi-project-notes-directory-for-root)
    (condition-case nil
        (imoogi-project-notes-directory-for-root root)
      (error nil))))

(defun imoogi-project--buffer-project ()
  "Return the project `b' lists buffers for.
The buffer's own project wins, then the Perspective's root (which also covers
notes buffers, where the finder declines), then the normal prompt.
Like the finder, the fallback declines for a remote buffer or a remote mapped
root, checked on the raw strings before any file operation (REQ-PIN-006)."
  (or (project-current nil)
      (when-let* (((bound-and-true-p persp-mode))
                  ((not (file-remote-p default-directory)))
                  (raw-root (car (rassoc (persp-current-name)
                                         imoogi-project-perspective-alist)))
                  ((not (file-remote-p raw-root)))
                  (root (imoogi-project--perspective-root)))
        (imoogi-project-resolve-root root))
      (project-current t)))

(defun imoogi-project-buffer-candidates (project)
  "Return the live buffers the project transient's `b' offers for PROJECT.
These are PROJECT's own buffers, as `project-switch-to-buffer' filters them,
plus the buffers visiting a file inside the project's notes directory.  The
notes buffers stay out of `project-buffers', so `k' never kills them."
  (let* ((notes (imoogi-project--notes-directory (project-root project)))
         (own (cl-remove-if
               (lambda (buffer)
                 (project--buffer-check buffer project-ignore-buffer-conditions))
               (project-buffers project)))
         (notes-buffers
          (when notes
            (cl-remove-if-not
             (lambda (buffer)
               (when-let* ((file (buffer-file-name buffer)))
                 (file-in-directory-p file notes)))
             (buffer-list)))))
    (delete-dups (append own notes-buffers))))

(defun imoogi-project-switch-to-buffer (buffer-or-name)
  "Switch to BUFFER-OR-NAME among the current project's buffers and notes.
Interactively, offer `imoogi-project-buffer-candidates' for the buffer's own
project or, in a buffer without one, for the current Perspective's project,
so scratch and notes buffers do not prompt for a project."
  (interactive
   (let* ((project (imoogi-project--buffer-project))
          (candidates (imoogi-project-buffer-candidates project))
          (other (other-buffer (current-buffer)))
          (name (read-buffer "Switch to buffer: "
                             (when (memq other candidates)
                               (buffer-name other))
                             nil
                             (lambda (entry) (memq (cdr entry) candidates)))))
     ;; Like `project-switch-to-buffer', a new name creates a buffer at the
     ;; project root.
     (list (or (get-buffer name)
               (let ((default-directory (project-root project)))
                 (get-buffer-create name))))))
  (switch-to-buffer buffer-or-name))

(defun imoogi-project--canonical-root (root)
  "Return ROOT as a canonical directory name."
  (file-name-as-directory (file-truename (expand-file-name root))))

(defun imoogi-project--path-components (root)
  "Return the non-empty path components in ROOT."
  (split-string (directory-file-name root) "/" t))

(defun imoogi-project--path-suffix (root depth)
  "Return the last DEPTH components of ROOT joined by a slash."
  (let* ((parts (imoogi-project--path-components root))
         (start (max 0 (- (length parts) depth))))
    (string-join (nthcdr start parts) "/")))

(defun imoogi-project--known-roots (root)
  "Return canonical known roots, including ROOT and persisted mappings."
  (delete-dups
   (delq nil
         (mapcar
          (lambda (candidate)
            (when (file-directory-p candidate)
              (imoogi-project--canonical-root candidate)))
          (append (list root)
                  (project-known-project-roots)
                  (mapcar #'car imoogi-project-perspective-alist))))))

(defun imoogi-project--unique-suffix (root roots)
  "Return ROOT's shortest suffix that is unique among ROOTS."
  (let ((others (delete root (copy-sequence roots))))
    (cl-loop for depth from 1 to (length (imoogi-project--path-components root))
             for suffix = (imoogi-project--path-suffix root depth)
             unless (cl-some
                     (lambda (other)
                       (string= suffix (imoogi-project--path-suffix other depth)))
                     others)
             return suffix)))

(defun imoogi-project--name-used-by-other-root-p (name root)
  "Return non-nil when NAME is already assigned to a root other than ROOT."
  (cl-some (lambda (entry)
             (and (not (equal (car entry) root))
                  (equal (cdr entry) name)))
           imoogi-project-perspective-alist))

(defun imoogi-project-perspective-name (root)
  "Return the stable Perspective name assigned to project ROOT.
The first computed mapping is persisted through `savehist' and is never
renamed merely because the known project registry changes."
  (let* ((canonical-root (imoogi-project--canonical-root root))
         (stored (assoc canonical-root imoogi-project-perspective-alist)))
    (or (cdr stored)
        (let* ((roots (imoogi-project--known-roots canonical-root))
               (basename (file-name-nondirectory
                          (directory-file-name canonical-root)))
               (same-basename
                (cl-remove-if-not
                 (lambda (candidate)
                   (equal basename
                          (file-name-nondirectory
                           (directory-file-name candidate))))
                 roots))
               (candidate
                (if (> (length same-basename) 1)
                    (or (imoogi-project--unique-suffix canonical-root roots)
                        basename)
                  basename))
               (name
                (if (imoogi-project--name-used-by-other-root-p
                     candidate canonical-root)
                    (format "%s--%s" candidate
                            (substring (secure-hash 'sha1 canonical-root) 0 8))
                  candidate)))
          (push (cons canonical-root name) imoogi-project-perspective-alist)
          name))))

(defun imoogi-project-switch-perspective (keep-perspective)
  "Select a project and open it in a matching Perspective workspace.
With KEEP-PERSPECTIVE (interactively, a prefix argument), remember and open
the selected project without leaving the current Perspective."
  (interactive "P")
  (let* ((selected-root (funcall project-prompter))
         (project
          ;; An explicit selection resolves on its own merits, even when it
          ;; equals the buffer's directory and is not under VC.
          (let ((project-prompter (lambda () selected-root))
                (imoogi-project--perspective-finder-inhibit t))
            (project-current t selected-root)))
         (root (project-root project)))
    (project-remember-project project)
    (unless keep-perspective
      (persp-switch (imoogi-project-perspective-name root)))
    (let ((default-directory root))
      (project-dired))
    ;; 07-treemacs.el owns the explorer integration.  A prefix argument is an
    ;; explicit request to keep the current, potentially multi-project,
    ;; Perspective, so leave its Treemacs workspace untouched as well.
    (when (and (not keep-perspective)
               (fboundp 'imoogi-treemacs-open-project-workspace))
      (imoogi-treemacs-open-project-workspace
       root (imoogi-project-perspective-name root)))))

(defun imoogi-persp-new (name)
  "새 작업공간 NAME 을 만들고 그리로 전환한다.

perspective 의 `persp-new' 는 interactive 가 아니라 메뉴에 직접 붙일 수 없고,
`persp-switch' 는 없는 이름을 주면 만들지만 이름만 봐서는 생성인지 전환인지
알기 어렵다. 그래서 \"만들기\"를 별도 명령으로 드러낸다."
  (interactive "s새 작업공간 이름: ")
  (persp-switch name))

(defun imoogi-perspective-state-restore ()
  "Restore saved Perspective state without making startup fragile."
  (when (file-readable-p imoogi-perspective-state-file)
    (condition-case err
        (progn
          (let ((imoogi-perspective--treemacs-visible nil))
            (persp-state-load imoogi-perspective-state-file))
          (setq imoogi-perspective--treemacs-visible
                (when (file-readable-p imoogi-perspective-treemacs-file)
                  (with-temp-buffer
                    (insert-file-contents imoogi-perspective-treemacs-file)
                    (let ((names (read (current-buffer))))
                      (when (and (listp names)
                                 (cl-every #'stringp names))
                        names)))))
          (when (file-readable-p imoogi-perspective-active-file)
            (let ((active
                   (with-temp-buffer
                     (insert-file-contents imoogi-perspective-active-file)
                     (read (current-buffer)))))
              (when (and (stringp active)
                         (member active (persp-names)))
                (persp-switch active))))
          (imoogi-perspective--restore-treemacs))
      (error
       (display-warning
        'imoogi-projects
        (format "Perspective state restore failed: %s"
                (error-message-string err))
        :warning)))
    nil))

(defun imoogi-perspective-state-save ()
  "Save Perspective state on normal Emacs exit."
  (when (bound-and-true-p persp-mode)
    (condition-case err
        (progn
          (make-directory (file-name-directory imoogi-perspective-state-file) t)
          (persp-state-save imoogi-perspective-state-file)
          (with-temp-file imoogi-perspective-active-file
            (prin1 (persp-current-name) (current-buffer)))
          (with-temp-file imoogi-perspective-treemacs-file
            (prin1 (imoogi-perspective--treemacs-visible-names)
                   (current-buffer))))
      (error
       (display-warning
        'imoogi-projects
        (format "Perspective state save failed: %s"
                (error-message-string err))
        :warning)))
    nil))

(use-package project
  :ensure nil
  :demand t
  :bind (:map project-prefix-map
              ("p" . imoogi-project-switch-perspective))
  :config
  (imoogi-project-install-perspective-finder))

;;; Perspective
(use-package perspective
  :ensure t
  :custom
  (persp-mode-prefix-key (kbd "C-x x"))
  (persp-state-default-file imoogi-perspective-state-file)
  :config
  (persp-mode 1)
  (define-key perspective-map (kbd "l") #'persp-switch-last)

  ;; [HARD] 모드라인의 작업공간 표시를 클릭 불가로 만든다.
  ;;
  ;; perspective 는 모드라인의 작업공간 이름을 클릭하면 그리로 전환되게 해 뒀다.
  ;; 그 처리는 클릭 지점의 문자열을 `posn-string' 으로 읽어 그대로
  ;; `persp-switch' 에 넘기는데, 이름 하나가 나올 것을 기대한 코드다.
  ;;
  ;; doom-modeline 은 모드라인 조각을 하나의 문자열로 합쳐 그리므로 이름이 아니라
  ;; 표시 전체가 넘어온다 — "main" 이 아니라 "[imoogi-emacs|main]" 이 넘어온다.
  ;; `persp-switch' 는 없는 이름을 받으면 그 이름으로 새로 만들므로, 클릭 한 번에
  ;; 작업공간 하나가 생기고 모드라인이 그만큼 길어진다. 다음 클릭은 그 늘어난
  ;; 문자열을 통째로 이름으로 삼아 정확히 두 배가 된다.
  ;;
  ;; 실측: 클릭 12번에 이름 40,959자짜리 작업공간까지 12개가 생겼고 모드라인
  ;; 문자열이 4,107,159자가 되어 화면을 뒤덮었다. 종료 시 상태 파일에 그대로
  ;; 저장돼(7.8MB) 다음 부팅에도 살아남았다.
  ;;
  ;; 바인딩만 떼어 낸다. advice 를 걸지 않으므로 패키지 업데이트에 끌려다니지
  ;; 않고, 클릭은 모드라인 기본 동작(창 크기 조절)으로 자연스럽게 넘어간다.
  ;; 키보드로 전환하는 길(C-x x s / l / n / p, C-c h p)은 그대로다.
  (dolist (map (list persp-mode-line-map persp-header-line-map))
    (define-key map [mode-line down-mouse-1] nil)
    (define-key map [header-line down-mouse-1] nil))

  (add-hook 'emacs-startup-hook #'imoogi-perspective-state-restore -50)
  (add-hook 'persp-before-switch-hook
            #'imoogi-perspective--remember-treemacs-visibility)
  (add-hook 'persp-switch-hook #'imoogi-perspective--restore-treemacs)
  (add-hook 'persp-switch-hook #'imoogi-project--perspective-finder-memo-clear)
  (add-hook 'kill-emacs-hook #'imoogi-perspective-state-save))

(provide 'imoogi-projects)
;;; projects.el ends here
