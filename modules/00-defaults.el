;;; 00-defaults.el --- 더 나은 기본값 + 세션 영속 -*- lexical-binding: t; -*-

;; minimal-emacs.d (James Cherti) 의 init.el "better defaults" 중 imoogi-emacs 에
;; 유용한 것들을 추려 모은 모듈. 외부 패키지 없이 내장 기능만 사용한다.
;; boot.el 의 모듈 리스트에서 가장 먼저 로드된다.

;;; Code:

;;; 응답 간소화 — yes/no 대신 y/n
(setq use-short-answers t
      read-answer-short t)

;;; 미니버퍼
(setq enable-recursive-minibuffers t) ; 중첩 미니버퍼 허용
;; 읽기 전용 프롬프트 영역에 커서가 들어가지 않게.
(setq minibuffer-prompt-properties
      '(read-only t intangible t cursor-intangible t face minibuffer-prompt))
(add-hook 'minibuffer-setup-hook #'cursor-intangible-mode)

;;; 잠금/백업 동작 (백업 디렉터리 자체는 boot.el 에서 설정)
;; 잠금 파일(.#filename)은 global-auto-revert-mode 로 대체 가능하므로 끈다.
(setq create-lockfiles nil)
(setq backup-by-copying t            ; 심볼릭 링크/하드링크 깨지 않게 복사로 백업
      delete-old-versions t          ; 오래된 백업 자동 정리
      version-control t              ; 번호 매긴 백업 사용
      kept-new-versions 5
      kept-old-versions 5)
;; 삭제 시 휴지통으로 이동(대화형 모드에서만).
(setq delete-by-moving-to-trash (not noninteractive))
;; 중복 kill 은 kill-ring 에 저장하지 않음.
(setq kill-do-not-save-duplicates t)

;;; 파일 처리
(setq find-file-visit-truename t       ; 심볼릭 링크를 실제 경로로 해석
      vc-follow-symlinks t)            ; VC 관리 심링크는 묻지 않고 따라감
(setq find-file-suppress-same-file-warnings t
      confirm-nonexistent-file-or-buffer nil)
(setq large-file-warning-threshold (* 100 1024 1024)) ; 100MB
(setq require-final-newline t)
;; 세로 분할 선호.
(setq split-width-threshold 170
      split-height-threshold nil)
;; 버퍼 이름 중복 시 디렉터리 경로로 구분.
(setq uniquify-buffer-name-style 'forward)

;;; 편집 / 들여쓰기
(setq-default indent-tabs-mode nil     ; 탭 대신 스페이스
              tab-width 4
              fill-column 80
              truncate-lines t         ; 기본은 줄 자르기(성능)
              word-wrap t)             ; 줄 바꿈 시 단어 단위로
(setq tab-always-indent 'complete)     ; TAB: 들여쓰기 후 보완
(setq sentence-end-double-space nil)   ; 문장 끝 두 칸 관습 폐기
;; 대용량 버퍼에서 입력 중 폰트 처리를 생략해 렉 완화.
(setq redisplay-skip-fontification-on-input t)

;;; 표시 / UI 디테일
(setq ring-bell-function #'ignore      ; 경고음/깜빡임 끄기
      visible-bell nil)
(setq truncate-string-ellipsis "…")
(setq x-underline-at-descent-line t)   ; 밑줄을 베이스라인 아래로
(setq mouse-yank-at-point t)           ; 마우스 붙여넣기를 커서 위치에
(setq highlight-nonselected-windows nil)
(defalias #'view-hello-file #'ignore)  ; hello 파일 표시 안 함
;; 줄 번호 폭 고정(점프 시 흔들림 방지).
(setq-default display-line-numbers-width 3)
;; 상대 줄 번호를 코드/텍스트/설정 버퍼에 표시 (minimal-emacs.d 추천).
(setq-default display-line-numbers-type 'relative)
(dolist (hook '(prog-mode-hook text-mode-hook conf-mode-hook))
  (add-hook hook #'display-line-numbers-mode))
;; 모드라인에 줄:열 표시, Tree-sitter 최대 하이라이트 레벨.
(setq line-number-mode t
      column-number-mode t)
(setq treesit-font-lock-level 4)

;;; 괄호 매칭
(setq show-paren-delay 0.1
      show-paren-when-point-inside-paren t
      show-paren-when-point-in-periphery t)
(setq blink-matching-paren nil)
(when (bound-and-true-p blink-cursor-mode)
  (blink-cursor-mode -1))

;;; 스크롤 (boot.el 의 중복 설정을 이리로 통합)
(setq scroll-conservatively 20         ; 과도한 recenter 방지
      scroll-margin 3
      scroll-preserve-screen-position t
      scroll-error-top-bottom t
      auto-window-vscroll nil
      fast-but-imprecise-scrolling t
      hscroll-margin 2
      hscroll-step 1)

;;; 부드러운 픽셀 스크롤 + fringe 폭 (minimal-emacs.d 추천)
;; emacs-mac 포트는 자체 스무스 스크롤이 있어 제외.
(unless (and (eq window-system 'mac)
             (bound-and-true-p mac-carbon-version-string))
  (when (fboundp 'pixel-scroll-precision-mode)
    (pixel-scroll-precision-mode 1)))
;; fringe 폭을 글자 폭에 맞춰 동적으로(diff-hl 등 표시 여유).
(when (fboundp 'fringe-mode)
  (fringe-mode (frame-char-width)))

;;; 실행 취소 한계 상향
(setq undo-limit (* 13 160000)
      undo-strong-limit (* 13 240000)
      undo-outer-limit (* 13 24000000))

;;; recentf — 최근 파일 기록
(use-package recentf
  :ensure nil
  :hook (after-init . recentf-mode)
  :config
  (setq recentf-max-saved-items 300
        recentf-max-menu-items 15))

;;; savehist — 미니버퍼 히스토리 세션 간 유지
(use-package savehist
  :ensure nil
  :hook (after-init . savehist-mode)
  :config
  (setq history-length 300
        savehist-additional-variables
        '(register-alist mark-ring global-mark-ring
                         search-ring regexp-search-ring)))

;;; saveplace — 파일 내 마지막 커서 위치 기억
(use-package saveplace
  :ensure nil
  :hook (after-init . save-place-mode)
  :config
  (setq save-place-limit 600))

;;; dired
(use-package dired
  :ensure nil
  :config
  (setq dired-dwim-target t            ; 분할 창 있으면 이동/복사 대상 자동 제안
        dired-recursive-deletes 'top
        dired-recursive-copies 'always
        dired-deletion-confirmer 'y-or-n-p
        dired-create-destination-dirs 'ask
        dired-clean-confirm-killing-deleted-buffers nil
        dired-auto-revert-buffer 'dired-directory-changed-p))

;;; eglot (Emacs 29+ 내장 LSP) — 사용 시 빠르고 조용하게
(with-eval-after-load 'eglot
  (setq eglot-autoshutdown t           ; 마지막 버퍼 종료 시 서버 종료
        eglot-sync-connect 0           ; 연결 동안 UI 블로킹 안 함
        eglot-extend-to-xref t
        eglot-events-buffer-config '(:size 0 :format short)))

;;; flymake
(with-eval-after-load 'flymake
  (setq flymake-show-diagnostics-at-end-of-line nil
        flymake-wrap-around nil))

;;; xref — 정의 후보를 미니버퍼에서 보완
(setq xref-show-definitions-function 'xref-show-definitions-completing-read
      xref-show-xrefs-function 'xref-show-definitions-completing-read)

;;; 비활성화돼 있던 유용한 명령들 활성화
(dolist (cmd '(narrow-to-region narrow-to-page upcase-region downcase-region
                                erase-buffer scroll-left dired-find-alternate-file
                                set-goal-column list-timers))
  (put cmd 'disabled nil))

(provide 'imoogi-defaults)
;;; 00-defaults.el ends here
