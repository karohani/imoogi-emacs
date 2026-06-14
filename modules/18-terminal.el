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
;; 수동 빌드: `M-x vterm-module-compile`.
;; 자세한 가이드는 README "vterm (터미널) 빌드" 섹션 참조.

;;; Code:

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
  (add-hook 'vterm-mode-hook #'imoogi--vterm-setup)
  (setq vterm-timer-delay 0.05
        vterm-kill-buffer-on-exit t
        vterm-max-scrollback 10000)
  ;; nil 이면 최초 실행 시 빌드 여부를 사용자에게 확인한다(타겟에서 빌드).
  (setq vterm-always-compile-module nil))

(provide 'imoogi-terminal)
;;; 18-terminal.el ends here
