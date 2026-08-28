;;; 10-theme.el --- doom-themes + doom-modeline -*- lexical-binding: t; -*-

;; 모던한 외관. doom-themes 는 treemacs/magit/ivy 등 imoogi 가 쓰는 패키지에
;; 맞춘 연동 테마를 내장한다. 모듈 로딩 순서상 treemacs(07) 뒤에 와야
;; doom-themes-treemacs-config 가 정상 동작한다.

;;; Code:

(imoogi-require "10-theme" 'doom-themes 'doom-modeline 'nerd-icons)

;;; 폰트 — 동봉 폰트 자동 설치 + 코딩 폰트 지정 (망분리 대응, 네트워크 불필요)
;; assets/fonts/ 의 모든 .ttf 를 OS 폰트 디렉터리로 복사한다. 로컬 파일 복사라
;; 인터넷이 필요 없다(M-x nerd-icons-install-fonts 는 다운로드라 폐쇄망 부적합).
;;   - NanumGothicCoding(.ttf/-Bold.ttf): 한글/영문 고정폭 코딩 폰트 (네이버, OFL 1.1)
;;   - NFM.ttf: nerd-icons 심볼 폰트 (doom-modeline 아이콘)

(defvar imoogi-font-family "NanumGothicCoding"
  "기본 코딩 폰트 패밀리. 설치돼 있을 때만 적용된다.")
(defvar imoogi-font-size 14
  "기본 폰트 크기(pt). `imoogi-font-size-by-dpi' 가 nil 이면 이 값을 그대로 쓴다.")

(defcustom imoogi-font-size-by-dpi
  '((150 . 16)
    (118 . 14)
    (0   . 13))
  "모니터 DPI 구간별 폰트 크기(pt). (최소DPI . 크기) 를 내림차순으로 둔다.

프레임이 놓인 모니터의 DPI 가 최소DPI 이상인 첫 항목의 크기를 쓴다.
nil 이면 해상도를 보지 않고 `imoogi-font-size' 를 그대로 쓴다.

방향: 여기서 계산하는 DPI 는 **논리 픽셀** 기준이라 macOS 의 화면 스케일링이
이미 반영돼 있다. 그래서 DPI 가 높을수록 같은 pt 가 물리적으로 작아 보이고,
겉보기 크기를 맞추려면 고DPI 쪽 pt 를 키워야 한다.

기본값 기준점은 27인치 2대 실측이다.
  2880x1620 (122DPI) -> 14pt   현재 쓰던 크기를 그대로 유지
  2560x1440 (109DPI) -> 13pt   14 * 109/122 = 12.5 -> 13
  150DPI 이상        -> 16pt   스케일링하지 않은 4K/5K 대비

[HARD] 이 표는 눈으로 확인해야 한다 — 계산은 겉보기 크기를 보장하지 않는다.
`M-x imoogi-font-report' 로 현재 모니터의 DPI 와 선택된 크기를 확인하고,
어색하면 이 표만 고친다(코드 수정 불필요)."
  :type '(choice (const :tag "해상도 무시" nil)
                 (alist :key-type integer :value-type integer))
  :group 'imoogi)

(defun imoogi--monitor-dpi (&optional frame)
  "FRAME 이 놓인 모니터의 가로 DPI. 구할 수 없으면 nil.

`display-mm-width' 는 다중 모니터에서 집계값을 돌려줘 못 쓴다 — 실측 예:
2880px 모니터에서 display-mm-width 가 795 로 나와 92DPI 가 계산됐지만
실제 모니터는 599mm 로 122DPI 였다(33% 오차). 그래서 모니터별 속성을 쓴다."
  (when (display-graphic-p frame)
    (let* ((attrs (frame-monitor-attributes frame))
           (geometry (alist-get 'geometry attrs))
           (mm-size (alist-get 'mm-size attrs))
           (px (nth 2 geometry))
           (mm (nth 0 mm-size)))
      (when (and (numberp px) (numberp mm) (> mm 0) (> px 0))
        (/ (* px 25.4) (float mm))))))

(defun imoogi-font-size-for-frame (&optional frame)
  "FRAME 에 쓸 폰트 크기(pt). DPI 를 못 구하면 `imoogi-font-size' 로 물러선다."
  (let ((dpi (and imoogi-font-size-by-dpi (imoogi--monitor-dpi frame))))
    (or (and dpi
             (cdr (seq-find (lambda (entry) (>= dpi (car entry)))
                            imoogi-font-size-by-dpi)))
        imoogi-font-size)))

(defun imoogi--os-font-dir ()
  "사용자 OS 폰트 디렉터리 경로(미지원 OS 면 nil)."
  (pcase system-type
    ('darwin (expand-file-name "~/Library/Fonts/"))
    ('gnu/linux (expand-file-name "~/.local/share/fonts/"))
    (_ nil)))

(defun imoogi--install-bundled-fonts ()
  "assets/fonts/*.ttf 를 OS 폰트 디렉터리에 설치(미설치분만). 네트워크 불필요.
설치 후 새로 켜는 Emacs 부터 폰트/아이콘이 인식된다."
  (let ((src-dir (expand-file-name "assets/fonts/"
                                   (bound-and-true-p imoogi-emacs-dir)))
        (font-dir (imoogi--os-font-dir))
        (installed nil))
    (when (and font-dir (file-directory-p src-dir))
      (make-directory font-dir t)
      (dolist (src (directory-files src-dir t "\\.ttf\\'"))
        (let ((dest (expand-file-name (file-name-nondirectory src) font-dir)))
          (unless (file-exists-p dest)
            (copy-file src dest t)
            (setq installed t))))
      (when installed
        ;; Linux 는 폰트 캐시 갱신 필요(있을 때만, 오프라인).
        (when (and (eq system-type 'gnu/linux) (executable-find "fc-cache"))
          (call-process "fc-cache" nil nil nil "-f" font-dir))
        (message "imoogi: 동봉 폰트 설치됨 → %s. 적용하려면 Emacs 재시작." font-dir)))))

(defun imoogi--apply-font (&optional frame)
  "코딩 폰트가 설치돼 있으면 FRAME(또는 현재 프레임)에 적용.
크기는 그 프레임이 놓인 모니터의 DPI 로 정한다(`imoogi-font-size-for-frame')."
  (when (and (display-graphic-p frame)
             (member imoogi-font-family (font-family-list frame)))
    (set-face-attribute 'default frame
                        :family imoogi-font-family
                        :height (* (imoogi-font-size-for-frame frame) 10))))

(defun imoogi--apply-font-on-move (frame)
  "FRAME 이 다른 모니터로 옮겨졌으면 폰트 크기를 다시 정한다.
`move-frame-functions' 는 창을 조금만 끌어도 불리므로, 크기가 실제로 바뀔
때만 얼굴 속성을 건드린다."
  (when (display-graphic-p frame)
    (let ((want (* (imoogi-font-size-for-frame frame) 10))
          (now (face-attribute 'default :height frame)))
      (unless (equal want now)
        (imoogi--apply-font frame)))))

(defun imoogi-font-report ()
  "현재 프레임이 놓인 모니터의 DPI 와 선택된 폰트 크기를 알린다."
  (interactive)
  (let ((dpi (imoogi--monitor-dpi)))
    (message "모니터 DPI: %s / 폰트 크기: %dpt%s"
             (if dpi (format "%.1f" dpi) "알 수 없음")
             (imoogi-font-size-for-frame)
             (if dpi "" " (DPI 를 못 구해 기본값 사용)"))))

;; 부팅 시 동봉 폰트 설치 → 현재/이후 프레임(daemon, emacsclient)에 적용
(imoogi--install-bundled-fonts)
(when (member imoogi-font-family (font-family-list))
  (imoogi--apply-font)
  (add-to-list 'default-frame-alist
               (cons 'font (format "%s-%d" imoogi-font-family imoogi-font-size))))
(add-hook 'after-make-frame-functions #'imoogi--apply-font)
;; 프레임을 다른 모니터로 옮기면 그 모니터 DPI 로 다시 맞춘다(Emacs 30+).
(when (boundp 'move-frame-functions)
  (add-hook 'move-frame-functions #'imoogi--apply-font-on-move))

;;; nerd-icons — doom-modeline 아이콘용(폰트는 위에서 동봉 설치)
(use-package nerd-icons
  :ensure t)

;;; doom-themes
(use-package doom-themes
  :ensure t
  :config
  (setq doom-themes-enable-bold t
        doom-themes-enable-italic t)
  ;; 기본 테마. 다른 변형 예: doom-nord, doom-dracula, doom-gruvbox, doom-vibrant.
  (load-theme 'doom-one t)
  ;; org-mode 헤더/블록 글꼴 보정
  (doom-themes-org-config)
  ;; treemacs 색상/아이콘을 doom 테마에 맞춤
  (with-eval-after-load 'treemacs
    (setq doom-themes-treemacs-theme "doom-colors")
    (doom-themes-treemacs-config)))

;;; hl-line — 현재 커서 라인 강하게 표시
(use-package hl-line
  :ensure nil
  :config
  (global-hl-line-mode 1)
  (set-face-attribute 'hl-line nil
                      :background "#343a46"
                      :extend t)
  (set-face-attribute 'line-number-current-line nil
                      :foreground "#ffcc66"
                      :background "#343a46"
                      :weight 'bold))

;;; Eglot 심볼 하이라이트 — 커서 올린 심볼의 다른 등장 위치 표시
;; VSCode 에서 변수에 커서를 두면 같은 변수가 전부 표시되는 그 기능이다.
;; Eglot 은 이미 이걸 한다(LSP documentHighlight → `eglot--highlight-piggyback').
;; 그런데 기본 face 가 `(:inherit bold)' 뿐이고(실측) doom-themes 는 이 face 를
;; 정의하지 않아(doom-themes-base.el 에 eglot 항목 0건), 고정폭 폰트에서는
;; 굵기 차이가 거의 안 보인다. 배경색을 주면 VSCode 처럼 눈에 띈다.
;; 색은 hl-line(#343a46) 보다 한 단계 밝게 잡아 현재 줄 위에서도 구분된다.
(with-eval-after-load 'eglot
  (set-face-attribute 'eglot-highlight-symbol-face nil
                      :background "#3f4757"
                      :weight 'normal))

;;; gnus face 상속 순환 끊기
;;
;; gnus 를 쓰지 않아도 gnus-group 이 다른 라이브러리에 딸려 로드되면, 화면을 그릴
;; 때마다 이 오류가 뜬다:
;;   Face inheritance results in inheritance cycle: gnus-group-news-low
;;
;; 두 정의가 서로를 가리켜서 생긴다 — 각각은 멀쩡한데 합쳐지면 고리가 된다.
;;   gnus 자체       : news-low       가 news-low-empty 를 상속
;;   doom-themes     : news-low-empty 가 news-low       를 상속
;;
;; doom-themes 가 news-low 에 대해 기록해 둔 스펙은 원래
;;   (:inherit gnus-group-mail-1 :foreground "#5B6268")
;; 인데, 테마가 적용된 뒤에 gnus 가 로드되면서 defface 기본값이 그 자리를 덮는다.
;; 그래서 테마의 의도대로 되돌려 놓으면 고리가 풀린다.
;;
;; 전체 face 576개를 훑어 순환은 이 한 쌍뿐임을 확인했다(실측) — 다른 gnus face 는
;; 멀쩡하므로 이 한 곳만 고치면 된다.
;;
;; [HARD] set-face-attribute 가 아니라 custom-theme-set-faces 여야 한다.
;; set-face-attribute 는 Custom 의 spec 체계 "바깥"에서 속성만 덮어쓴다. 그래서
;; 새 프레임이 생기거나 테마가 다시 계산될 때(custom-theme-recalc-face) Custom 이
;; defface 기본값 + 테마 spec 으로 다시 계산하면서 순환이 되살아나고,
;; face-spec-set-2 가 그 순간 오류를 낸다(실측: 수정을 넣은 세션에서도 재발).
;; spec 자체를 바꿔 두면 언제 다시 계산해도 순환이 만들어지지 않는다.
;;
;; 고치는 쪽은 -empty 다. doom-themes 의 다른 -empty face 들은 전부
;; gnus-group-mail-1-empty 를 상속하는데(669행 등) news-low-empty 만 news-low 를
;; 가리켜 고리를 만든다 — 형제들과 같은 모양으로 맞춘다.
(custom-theme-set-faces 'doom-one
  '(gnus-group-news-low-empty ((t (:inherit gnus-group-mail-1-empty :weight normal)))))

;;; doom-modeline — 깔끔한 모드라인
(use-package doom-modeline
  :ensure t
  :hook (after-init . doom-modeline-mode)
  :config
  (setq doom-modeline-height 28
        doom-modeline-bar-width 3
        doom-modeline-icon t
        doom-modeline-major-mode-icon t
        doom-modeline-buffer-file-name-style 'truncate-upto-project
        doom-modeline-minor-modes nil))

(provide 'imoogi-theme)
;;; 10-theme.el ends here
