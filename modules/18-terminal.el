;;; 18-terminal.el --- vterm 터미널 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; vterm 은 libvterm(C) 기반 고성능 터미널 에뮬레이터다. elisp 패키지는 vendor 에
;; 동봉되지만, 네이티브 모듈(vterm-module)은 타겟에서 **최초 실행 시 빌드**된다.
;;
;; 망분리 타겟 빌드 사전조건(빌드 도구가 타겟에 있어야 함):
;;   - cmake, C 컴파일러(cc/gcc), libtool
;;   - 시스템 libvterm (없으면 cmake 가 인터넷에서 받으려 하므로 폐쇄망에선 필수)
;;     · macOS : brew install cmake libtool libvterm
;;     · Debian: sudo apt install cmake libtool-bin libvterm-dev
;;
;; 최초 실행: `M-x vterm` → 모듈이 없으면 "Compile vterm-module?" 확인 후 빌드.
;; 수동 빌드: `M-x vterm-module-compile`.  사전점검: `M-x imoogi-vterm-check-deps`.
;; 자세한 가이드는 README "vterm (터미널) 빌드" 섹션 참조.

;;; Code:

(imoogi-require "18-terminal" 'vterm)

;;; 빌드 사전조건 점검 -----------------------------------------------------------
;; vterm 모듈 빌드는 cmake/make/cc 와 (시스템 libvterm 없으면) glibtool·pkg-config
;; 가 필요하다. 누락 시 cryptic 한 make 오류 대신 무엇을 설치해야 하는지 안내한다.

(defun imoogi--vterm-install-hint ()
  "OS 별 빌드 의존성 설치 명령 문자열."
  (pcase system-type
    ('darwin "brew install cmake libtool libvterm pkg-config")
    ('gnu/linux "apt install cmake libtool-bin libvterm-dev pkg-config gcc")
    (_ "cmake, libtool, libvterm, pkg-config, C 컴파일러를 설치하세요")))

(defun imoogi--vterm-system-libvterm-p ()
  "시스템 libvterm 을 pkg-config 로 찾을 수 있으면 non-nil."
  (and (executable-find "pkg-config")
       (eq 0 (call-process "pkg-config" nil nil nil "--exists" "vterm"))))

(defun imoogi--vterm-missing-deps ()
  "vterm 모듈 빌드를 막는 누락 도구 목록(없으면 nil)."
  ;; macOS 의 GNU libtool 은 `glibtool' 이름으로 설치된다.
  (let ((glibtool (if (eq system-type 'darwin) "glibtool" "libtool"))
        (missing '()))
    (unless (executable-find "cmake") (push "cmake" missing))
    (unless (executable-find "make") (push "make" missing))
    (unless (executable-find "cc") (push "C 컴파일러(cc)" missing))
    ;; 시스템 libvterm 이 있으면 그걸 링크하므로 glibtool/pkg-config 불필요.
    ;; 없으면 번들 libvterm 을 컴파일해야 해서 둘 다 필요(+ 폐쇄망이면 다운로드 실패).
    (unless (imoogi--vterm-system-libvterm-p)
      (unless (executable-find "pkg-config") (push "pkg-config" missing))
      (unless (executable-find glibtool) (push glibtool missing)))
    (nreverse missing)))

(defun imoogi-vterm-check-deps ()
  "vterm 모듈 빌드 사전조건을 점검하고 결과를 출력한다."
  (interactive)
  (let ((missing (imoogi--vterm-missing-deps)))
    (cond
     (missing
      (message "vterm 빌드 불가 — 누락: %s\n  설치: %s"
               (string-join missing ", ") (imoogi--vterm-install-hint)))
     ((imoogi--vterm-system-libvterm-p)
      (message "vterm 빌드 사전조건 충족 ✓ (시스템 libvterm 사용). M-x vterm 으로 빌드"))
     (t
      (message "vterm 빌드 가능 ✓ (번들 libvterm 컴파일 — 폐쇄망이면 시스템 libvterm 권장)")))))

(defun imoogi--vterm-precheck (&rest _)
  "vterm 모듈 빌드 전에 의존성을 점검해 누락 시 친절히 중단."
  (let ((missing (imoogi--vterm-missing-deps)))
    (when missing
      (user-error "vterm 모듈을 빌드할 수 없습니다 — 누락: %s.  설치: %s"
                  (string-join missing ", ") (imoogi--vterm-install-hint)))))

;;; vterm -----------------------------------------------------------------------

(use-package vterm
  :ensure t
  ;; 모듈 지원 빌드의 Emacs 에서만 활성화
  :if (bound-and-true-p module-file-suffix)
  :commands (vterm vterm-other-window vterm-module-compile)
  :bind (("C-c t" . vterm))
  :preface
  (when noninteractive
    ;; vterm 로드 시 vterm-module 컴파일이 트리거될 수 있다. vendoring 의
    ;; 바이트컴파일(noninteractive) 단계에서 빌드를 시도하지 않도록 막는다.
    (advice-add #'vterm-module-compile :override #'ignore))
  (defun imoogi--vterm-setup ()
    (setq mode-line-format nil)
    (setq-local hscroll-margin 0)
    (setq-local confirm-kill-processes nil))
  :init
  ;; 빌드 시도 전 의존성 점검(누락 시 cryptic make 오류 대신 안내 후 중단).
  (unless noninteractive
    (advice-add 'vterm-module-compile :before #'imoogi--vterm-precheck))
  (add-hook 'vterm-mode-hook #'imoogi--vterm-setup)
  (setq vterm-timer-delay 0.05
        vterm-kill-buffer-on-exit t
        vterm-max-scrollback 10000)
  ;; nil 이면 최초 실행 시 빌드 여부를 사용자에게 확인한다(타겟에서 빌드).
  (setq vterm-always-compile-module nil))

(provide 'imoogi-terminal)
;;; 18-terminal.el ends here
