;;; 14-org.el --- org 설정 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; org-mode(내장) 기본 설정 + org-appear.

;;; Code:

(imoogi-require "14-org" 'org 'org-appear 'hl-line 'calendar 'transient
                'mermaid-mode 'ob-mermaid)

(eval-when-compile (require 'transient))

;; Org Babel does not provide a JSON language alias.  Use the built-in
;; `js-json-mode' (which is remapped to `json-ts-mode' when available) for
;; `#+begin_src json' editing buffers.
(with-eval-after-load 'org-src
  (add-to-list 'org-src-lang-modes '("json" . js-json)))

;; `#+begin_src mermaid' already maps to `mermaid-mode' by name.  Enable
;; Babel execution for shell (sh/bash/zsh, built-in `ob-shell') and mermaid
;; (`ob-mermaid', renders via mmdc on PATH).  `org-confirm-babel-evaluate'
;; stays at its default, so each execution still asks first.
(defun imoogi-org--enable-babel-languages ()
  "Add shell and mermaid to `org-babel-load-languages', keeping existing ones."
  (let ((languages org-babel-load-languages))
    (dolist (language '(shell mermaid))
      (setf (alist-get language languages) t))
    (org-babel-do-load-languages 'org-babel-load-languages languages)))

(with-eval-after-load 'org
  (imoogi-org--enable-babel-languages))

(defun imoogi-org--default-directory ()
  "Return imoogi's default Org directory."
  (expand-file-name "~/notes/"))

(defun imoogi-org--permanent-directory ()
  "Return the directory used for permanent Org-roam notes."
  (expand-file-name "permanent/" (imoogi-org--default-directory)))

(defun imoogi-org--default-agenda-file ()
  "Return imoogi's default agenda file."
  (expand-file-name "agenda.org" (imoogi-org--default-directory)))

(defun imoogi-org--ensure-agenda-file (file)
  "Create FILE as a minimal Org agenda file when it does not exist."
  (when (file-directory-p file)
    (signal 'file-error (list "Agenda file path is a directory" file)))
  (unless (file-exists-p file)
    (with-temp-file file
      (insert "#+title: Agenda\n\n* Inbox\n* Tasks\n* Schedule\n"))))

(defun imoogi-org--agenda-storage-buffer-modified-p ()
  "Return non-nil when the agenda file-list storage buffer has unsaved edits."
  (and (stringp org-agenda-files)
       (let ((buffer (find-buffer-visiting org-agenda-files)))
         (and buffer (buffer-modified-p buffer)))))

(defun imoogi-org--current-agenda-targets ()
  "Return current agenda targets expanded against the present `org-directory'."
  (cond
   ((stringp org-agenda-files)
    (if (file-exists-p org-agenda-files)
        (mapcar #'car (org-read-agenda-file-list t))
      nil))
   ((listp org-agenda-files)
    (mapcar (lambda (file) (expand-file-name file org-directory))
            org-agenda-files))
   (t (error "Invalid value of `org-agenda-files'"))))

(defun imoogi-org--write-agenda-storage-file (file targets)
  "Write agenda TARGETS to agenda storage FILE without changing its variable."
  (let ((directory (file-name-directory (expand-file-name file))))
    (when directory
      (make-directory directory t)))
  (with-temp-file file
    (insert (mapconcat #'identity targets "\n"))
    (insert "\n")))

(defun imoogi-org--register-agenda-directory (directory)
  "Register DIRECTORY as an Org agenda target while preserving existing targets."
  (let* ((directory (file-name-as-directory (expand-file-name directory)))
         (targets (delete-dups
                   (append (imoogi-org--current-agenda-targets)
                           (list directory)))))
    (when (imoogi-org--agenda-storage-buffer-modified-p)
      (user-error "Save or kill the agenda file-list buffer before running imoogi-org-setup"))
    (if (stringp org-agenda-files)
        (imoogi-org--write-agenda-storage-file org-agenda-files targets)
      (setq org-agenda-files targets))
    directory))

(defun imoogi-org--register-default-agenda-directory-when-present ()
  "Restore the default notes target without writing agenda storage at startup."
  (let ((directory (imoogi-org--default-directory)))
    (when (and (listp org-agenda-files) (file-directory-p directory))
      (imoogi-org--register-agenda-directory directory))))

;;;###autoload
(defun imoogi-org-setup ()
  "Create ~/notes and use it as the default Org directory.
Existing notes are preserved.  This command does not run Anki setup or sync."
  (interactive)
  (require 'org)
  (let ((directory (imoogi-org--default-directory))
        (agenda-file (imoogi-org--default-agenda-file)))
    (make-directory directory t)
    (make-directory (imoogi-org--permanent-directory) t)
    (imoogi-org--ensure-agenda-file agenda-file)
    (imoogi-org--register-agenda-directory directory)
    (setq org-directory directory)
    (message "imoogi: 기본 Org 폴더: %s, agenda 파일: %s" directory agenda-file)
    directory))

;;;###autoload
(defun imoogi-org-setup-doctor ()
  "Repair the notes layout without deleting or overwriting any files.

Ensure `~/notes/`, `~/notes/permanent/`, and `agenda.org` exist, then
copy direct `.org` files from the notes root into `permanent/`.  The
scratch and agenda files are excluded.  Existing destination files are
left untouched, and every source file is preserved."
  (interactive)
  (let* ((directory (imoogi-org--default-directory))
         (permanent (imoogi-org--permanent-directory))
         (agenda-file (imoogi-org--default-agenda-file))
         (copied 0)
         (skipped 0)
         (excluded 0)
         (sources nil))
    (make-directory directory t)
    (make-directory permanent t)
    (imoogi-org--ensure-agenda-file agenda-file)
    (imoogi-org--register-agenda-directory directory)
    (setq org-directory directory)
    (dolist (source (directory-files directory t "\\.org\\'" t))
      (let ((name (file-name-nondirectory source)))
        (cond
         ((member name '("agenda.org" "scratch.org"))
          (setq excluded (1+ excluded)))
         ((file-regular-p source)
          (push source sources)))))
    (dolist (source (nreverse sources))
      (let ((destination (expand-file-name (file-name-nondirectory source)
                                           permanent)))
        (if (file-exists-p destination)
            (setq skipped (1+ skipped))
          (copy-file source destination nil nil t)
          (setq copied (1+ copied)))))
    (when (fboundp 'org-roam-db-sync)
      (condition-case err
          (org-roam-db-sync)
        (error
         (message "imoogi: org-roam DB 동기화 건너뜀: %s" (error-message-string err)))))
    (message "imoogi: notes 구조 점검 완료 — 복사 %d개, 기존 대상 보존 %d개, 제외 %d개 (원본 삭제 없음)"
             copied skipped excluded)
    (list :directory directory :permanent permanent :agenda agenda-file
          :copied copied :skipped skipped :excluded excluded)))

;;;###autoload
(defun imoogi-org-agenda ()
  "Open the standard Org agenda dispatcher."
  (interactive)
  (require 'org-agenda)
  (org-agenda nil))

(defun imoogi-org-agenda-overdue-p ()
  "Return non-nil for an unfinished entry whose deadline is before today."
  (let ((deadline (org-entry-get nil "DEADLINE")))
    (and deadline (not (org-entry-is-done-p))
         (< (org-time-string-to-absolute deadline) (org-today)))))

(defun imoogi-org-agenda-skip-not-overdue ()
  "Skip this heading unless overdue, without skipping its children."
  (unless (imoogi-org-agenda-overdue-p)
    (save-excursion (outline-next-heading) (point))))

(defun imoogi-org-agenda-skip-overdue ()
  "Skip overdue headings already included in the summary block."
  (when (imoogi-org-agenda-overdue-p)
    (save-excursion (outline-next-heading) (point))))

(defun imoogi-org-agenda-overview (&optional _match)
  "Show overdue entries once above the regular agenda."
  (interactive)
  (require 'org-agenda)
  (org-agenda-run-series
   "일정"
   '(((tags "DEADLINE<>\"\""
            ((org-agenda-overriding-header "기한 지난 항목")
             (org-agenda-skip-function #'imoogi-org-agenda-skip-not-overdue)
             (org-agenda-sorting-strategy '(deadline-up priority-down category-keep))))
      (agenda ""
              ((org-agenda-skip-function #'imoogi-org-agenda-skip-overdue)))))))

(use-package org-agenda
  :ensure nil
  :after org
  :config
  ;; Use the overview for the standard dispatcher key, preserving a user's
  ;; existing custom command if they have already assigned that key.
  (unless (assoc "a" org-agenda-custom-commands)
    (add-to-list 'org-agenda-custom-commands
                 '("a" "일정 + 기한 지난 항목" imoogi-org-agenda-overview ""))))

(defvar-local imoogi-org-calendar--fit-cookie nil)
(defvar-local imoogi-org-calendar--fit-state nil)

(defun imoogi-org-calendar-fit-window (&optional window)
  "Fit the calendar within 40% of the frame, shrinking only its display.
Keep the user's text scale intact; restore it when space becomes available.
Very small frames retain a 9-point readability floor and date text input."
  (let ((window (or window (get-buffer-window (current-buffer)))))
    (when (and (window-live-p window)
               (with-current-buffer (window-buffer window)
                 (derived-mode-p 'calendar-mode)))
      (with-current-buffer (window-buffer window)
        (let* ((frame (window-frame window))
               (cap (floor (* 0.4 (window-pixel-height (frame-root-window frame)))))
               (width (window-body-width window t))
               (scale (if (bound-and-true-p text-scale-mode)
                          (expt text-scale-mode-step text-scale-mode-amount) 1.0))
               (base-height (face-attribute 'default :height frame))
               (state (list cap width scale base-height (buffer-chars-modified-tick))))
          ;; Cache font fitting, but always repair the actual window height.
          (unless (equal state imoogi-org-calendar--fit-state)
            (setq imoogi-org-calendar--fit-state state)
            (when imoogi-org-calendar--fit-cookie
              (face-remap-remove-relative imoogi-org-calendar--fit-cookie)
              (setq imoogi-org-calendar--fit-cookie nil))
            (let* ((size (window-text-pixel-size window nil t 10000 10000))
                   (chrome (- (window-pixel-height window)
                              (window-body-height window t)))
                   (factor (min 1.0
                                (/ (float (max 1 (- width 8))) (max 1 (car size)))
                                (/ (float (max 1 (- cap chrome 4)))
                                   (max 1 (cdr size)))))
                   (floor-factor (min 1.0 (/ 90.0 (* base-height scale)))))
              (when (< factor 1.0)
                (setq imoogi-org-calendar--fit-cookie
                      (face-remap-add-relative 'default :height
                                               (max floor-factor factor))))))
          ;; Org and persisted zoom may change the window after the first fit,
          ;; even when the cached frame/content dimensions are unchanged.
          (let* ((height (cdr (window-text-pixel-size window nil t 10000 10000)))
                 (chrome (- (window-pixel-height window)
                            (window-body-height window t)))
                 (target (min cap (+ chrome height 4)))
                 (delta (- target (window-pixel-height window))))
            (unless (zerop delta)
              (let ((window-resize-pixelwise t))
                (window-resize-no-error window delta nil window t)))))
        (set-window-start window (point-min))
        (set-window-vscroll window 0)))))

(defun imoogi-org-calendar-fit-frame (frame)
  "Refit calendars after frame-wide persisted text scale restoration.
Global window hooks run after buffer-local hooks.  Persisted zoom also
suppresses `text-scale-mode-hook', so local fitting alone is too early."
  (dolist (window (window-list frame 'nomini))
    (imoogi-org-calendar-fit-window window)))

(defun imoogi-org-calendar-window-setup ()
  "Refit after persisted zoom, frame resizing, and manual zoom changes."
  (add-hook 'window-buffer-change-functions
            #'imoogi-org-calendar-fit-window 90 t)
  (add-hook 'window-size-change-functions
            #'imoogi-org-calendar-fit-window 90 t)
  (add-hook 'text-scale-mode-hook #'imoogi-org-calendar-fit-window 90 t))

(use-package calendar
  :ensure nil
  :custom (calendar-split-width-threshold nil)
  :hook ((calendar-mode . imoogi-org-calendar-window-setup)
         (calendar-initial-window . imoogi-org-calendar-fit-window))
  :config
  (require 'face-remap)
  (add-hook 'window-buffer-change-functions #'imoogi-org-calendar-fit-frame 95)
  (add-to-list 'display-buffer-alist
               '("\\`\\*Calendar\\*\\'"
                 (display-buffer-in-side-window)
                 (side . bottom) (slot . 0)))
  (when (get-buffer calendar-buffer)
    (with-current-buffer calendar-buffer
      (imoogi-org-calendar-window-setup)
      (imoogi-org-calendar-fit-window))))

(defun imoogi-org-hl-line-range ()
  "제목 줄의 색을 가리지 않도록 본문에서만 현재 줄을 강조한다."
  (unless (org-at-heading-p)
    (cons (line-beginning-position)
          (min (point-max) (1+ (line-end-position))))))

(defun imoogi-org-heading-setup ()
  "제목 배경을 줄 끝까지 적용하고 기존 버퍼의 표시도 갱신한다."
  (setq-local org-fontify-whole-heading-line t)
  (setq-local hl-line-range-function #'imoogi-org-hl-line-range)
  (setq-local org-ellipsis " ▼")
  (setq-local buffer-display-table
              (copy-sequence (or buffer-display-table (make-display-table))))
  (set-display-table-slot
   buffer-display-table 4
   (vconcat (mapcar (lambda (char) (make-glyph-code char 'org-ellipsis))
                   org-ellipsis)))
  (dolist (alias '(outline block drawer))
    (org-fold-core-set-folding-spec-property
     (org-fold-core-get-folding-spec-from-alias alias) :ellipsis org-ellipsis))
  (when (fboundp 'hl-line-unhighlight) (hl-line-unhighlight))
  (when (fboundp 'global-hl-line-unhighlight) (global-hl-line-unhighlight))
  (font-lock-flush))

;;; org-mode (내장)
(use-package org
  :ensure nil
  :mode ("\\.org\\'" . org-mode)
  :hook (org-mode . imoogi-org-heading-setup)
  :custom
  (org-directory (expand-file-name "~/notes/"))
  (org-hide-leading-stars t)
  (org-startup-indented t)
  (org-adapt-indentation nil)
  (org-edit-src-content-indentation 0)
  (org-startup-truncated t)
  (org-fontify-whole-heading-line t)
  (org-cycle-level-faces t)
  (org-ellipsis " ▼")
  :config
  (imoogi-org--register-default-agenda-directory-when-present)
  ;; 빨강 → 파랑 → 초록 → 노랑. :extend 로 제목 뒤 빈 공간까지 칠한다.
  ;; user 테마에 등록해 테마 재적용과 새 프레임에서도 유지한다.
  (let ((palette '(("#ff8c92" "#3b2930" "#a12635" "#fbe9ec")
                   ("#82b7ff" "#253449" "#205da8" "#e8f0fc")
                   ("#a5d67d" "#2c392b" "#386b23" "#edf5e7")
                   ("#f2d479" "#3d3726" "#806000" "#faf3d9"))))
    (dotimes (index 8)
      (let ((colors (nth (% index 4) palette)))
        (custom-theme-set-faces
         'user
         `(,(intern (format "org-level-%d" (1+ index)))
           ((((background light))
             (:foreground ,(nth 2 colors) :background ,(nth 3 colors)
              :weight bold :extend t))
            (t (:foreground ,(nth 0 colors) :background ,(nth 1 colors)
                :weight bold :extend t))))))))
  ;; 설정 재로드 시 이미 열려 있던 Org 버퍼도 줄 전체를 다시 칠한다.
  (dolist (buffer (buffer-list))
    (with-current-buffer buffer
      (when (derived-mode-p 'org-mode)
        (imoogi-org-heading-setup)))))

;;; org-appear — 강조표시(*굵게* 등) 마크업을 커서가 닿을 때만 표시
(use-package org-appear
  :ensure t
  :hook (org-mode . org-appear-mode))

(defun imoogi-org-open-agenda-file ()
  "Open the default agenda file, creating it safely if needed."
  (interactive)
  (imoogi-org-setup)
  (find-file (imoogi-org--default-agenda-file)))

(defun imoogi-org-agenda-heading-p ()
  "Return non-nil when the current position is on an Org heading."
  (and (derived-mode-p 'org-mode) (org-at-heading-p)))

(defun imoogi-org-set-category (category)
  "Set CATEGORY on the current heading and its inheriting children."
  (interactive
   (progn
     (unless (imoogi-org-agenda-heading-p)
       (user-error "Org 제목에서 실행하세요"))
     (list (read-string "카테고리: " (org-entry-get nil "CATEGORY" t)))))
  (unless (imoogi-org-agenda-heading-p)
    (user-error "Org 제목에서 실행하세요"))
  (when (string-empty-p (string-trim category))
    (user-error "카테고리를 입력하세요"))
  (org-set-property "CATEGORY" (string-trim category)))

(defun imoogi-org-export-context-p ()
  "Return non-nil in an Org document or agenda view."
  (derived-mode-p 'org-mode 'org-agenda-mode))

(defun imoogi-org-export ()
  "Export the current Org document or agenda view interactively."
  (interactive)
  (cond
   ((derived-mode-p 'org-agenda-mode)
    (require 'org-agenda)
    (call-interactively #'org-agenda-write))
   ((derived-mode-p 'org-mode)
    (require 'ox)
    (call-interactively #'org-export-dispatch))
   (t (user-error "Org 문서 또는 Agenda 화면에서 실행하세요"))))

;;; Org block snippets and completion

(defun imoogi-org--block-template (kind)
  "Return the yasnippet template for Org block KIND."
  (pcase kind
    ('source "#+begin_src ${1:language}\n$0\n#+end_src")
    ('example "#+begin_example\n$0\n#+end_example")
    (_ (error "지원하지 않는 Org 블록: %s" kind))))

(defun imoogi-org-insert-block (kind)
  "Insert an Org block of KIND using yasnippet fields."
  (interactive (list (intern (completing-read
                              "Org 블록: " '("source" "example") nil t))))
  (unless (derived-mode-p 'org-mode)
    (user-error "Org 버퍼에서 실행하세요"))
  (unless (require 'yasnippet nil t)
    (user-error "yasnippet을 불러올 수 없습니다"))
  (yas-expand-snippet (imoogi-org--block-template kind)))

(defun imoogi-org-insert-source-block ()
  "Insert an Org source block with a language field."
  (interactive)
  (imoogi-org-insert-block 'source))

(defun imoogi-org-insert-example-block ()
  "Insert an Org example block."
  (interactive)
  (imoogi-org-insert-block 'example))

(defun imoogi-org--register-block-snippets ()
  "Register the short Org block snippets used by `<s>' and `<e>'."
  (when (fboundp 'yas-define-snippets)
    (yas-define-snippets
     'org-mode
     `(("<s>" ,(imoogi-org--block-template 'source) "Org source block")
       ("<e>" ,(imoogi-org--block-template 'example) "Org example block")))))

(if (featurep 'yasnippet)
    (imoogi-org--register-block-snippets)
  (with-eval-after-load 'yasnippet
    (imoogi-org--register-block-snippets)))

(defun imoogi-org--block-completion-candidates ()
  "Return completion candidates for the Org block shortcuts at point."
  (when (derived-mode-p 'org-mode)
    (let* ((point (point))
           (line-start (line-beginning-position))
           (start (save-excursion
                    (when (search-backward "<" line-start t)
                      (point))))
           (end (if (and (< point (point-max))
                        (eq (char-after point) ?>))
                     (1+ point)
                   point)))
      (when (and start
                 (string-match-p "\\`<\\(?:s\\|e\\)>?\\'"
                                 (buffer-substring-no-properties start end)))
        (list start end '("<s>" "<e>")
              :exclusive 'no
              :annotation-function
              (lambda (candidate)
                (if (equal candidate "<s>") "  source block" "  example block"))
              :exit-function
              (lambda (candidate status)
                (when (and (eq status 'finished)
                           (member candidate '("<s>" "<e>")))
                  (when (search-backward candidate line-start t)
                    (delete-region (point) (+ (point) (length candidate)))
                    (imoogi-org-insert-block
                     (if (equal candidate "<s>") 'source 'example))))))))))

(defun imoogi-org--enable-block-completion ()
  "Enable Org block shortcut completion and avoid pairing `<'."
  (require 'elec-pair)
  (let ((default (default-value 'electric-pair-inhibit-predicate)))
    (setq-local electric-pair-inhibit-predicate
                (lambda (char)
                  (or (eq char ?<)
                      (and default (funcall default char))))))
  (add-hook 'completion-at-point-functions
            #'imoogi-org--block-completion-candidates nil t))

(add-hook 'org-mode-hook #'imoogi-org--enable-block-completion)

(with-eval-after-load 'imoogi-transient
  (transient-define-prefix imoogi-org-block-transient ()
    "Org source and example block insertion."
    :column-widths '(22 22)
    [["블록 ----------------"
      ("s" "source 블록" imoogi-org-insert-source-block)
      ("e" "example 블록" imoogi-org-insert-example-block)]
     ["기타 ----------------"
      ("q" "종료" transient-quit-one)]])

  (transient-define-prefix imoogi-org-agenda-transient ()
    "Org agenda and planning commands."
    :column-widths '(20 20 20)
    [["조회 -------------"
      ("a" "일정 보기" imoogi-org-agenda-overview)
      ("t" "전체 TODO" org-todo-list)
      ("m" "Agenda 메뉴" imoogi-org-agenda)
      ("b" "Org 블록" imoogi-org-block-transient)]
     ["현재 제목 ---------"
      ("s" "일정 지정" org-schedule :inapt-if-not imoogi-org-agenda-heading-p)
      ("d" "마감일 지정" org-deadline :inapt-if-not imoogi-org-agenda-heading-p)
      ("T" "TODO 상태" org-todo :inapt-if-not imoogi-org-agenda-heading-p)
      ("c" "카테고리 설정" imoogi-org-set-category :inapt-if-not imoogi-org-agenda-heading-p)]
     ["파일·설정 ---------"
      ("e" "agenda.org 열기" imoogi-org-open-agenda-file)
      ("S" "기본 폴더 설정" imoogi-org-setup)
      ("x" "내보내기" imoogi-org-export :inapt-if-not imoogi-org-export-context-p)
      ("v" "브라우저 미리보기" imoogi-org-preview :inapt-if-not (lambda () (derived-mode-p 'org-mode)))
      ("V" "미리보기 종료" imoogi-org-preview-stop :inapt-if-not (lambda () (derived-mode-p 'org-mode)))
      ("q" "종료" transient-quit-one)]])
  (transient-append-suffix 'imoogi-transient-master "t"
    '("o" "Org Agenda" imoogi-org-agenda-transient)))

(provide 'imoogi-org)
;;; 14-org.el ends here
