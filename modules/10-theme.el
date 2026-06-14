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
  "기본 폰트 크기(pt).")

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
  "코딩 폰트가 설치돼 있으면 FRAME(또는 현재 프레임)에 적용."
  (when (and (display-graphic-p frame)
             (member imoogi-font-family (font-family-list frame)))
    (set-face-attribute 'default frame
                        :family imoogi-font-family
                        :height (* imoogi-font-size 10))))

;; 부팅 시 동봉 폰트 설치 → 현재/이후 프레임(daemon, emacsclient)에 적용
(imoogi--install-bundled-fonts)
(when (member imoogi-font-family (font-family-list))
  (imoogi--apply-font)
  (add-to-list 'default-frame-alist
               (cons 'font (format "%s-%d" imoogi-font-family imoogi-font-size))))
(add-hook 'after-make-frame-functions #'imoogi--apply-font)

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
    (setq doom-themes-treemacs-theme "doom-atom")
    (doom-themes-treemacs-config)))

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
