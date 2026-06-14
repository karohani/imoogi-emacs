;;; 18-terminal.el --- ghostel 터미널 (libghostty-vt) -*- lexical-binding: t; -*-

;; ghostel 은 libghostty-vt(Ghostty 의 VT 엔진) 기반 Emacs 터미널 에뮬레이터다.
;; vterm 대비 기능이 우수하고(kitty graphics/keyboard, OSC8, char/line/emacs 모드,
;; 더 높은 처리량 등), 무엇보다 `ghostel-ime-mode' 로 Emacs 내장 한글 입력기
;; (korean-hangul, S-SPC)가 **터미널 안에서도 동작**한다 — vterm 이 못 하던 것.
;;
;; air-gap: 네이티브 모듈을 저장소 vendor/ghostel-module/ 에 동봉한다
;; (사전빌드 바이너리, aarch64-macos). 부팅/사용 시 다운로드를 시도하지 않는다.
;; 모듈 갱신은 온라인 머신에서 `M-x ghostel-download-module' 실행 후 vendor 커밋.
;; (타겟 arch 가 다르면 해당 arch 바이너리를 동봉하거나 `M-x ghostel-module-compile'.)

;;; Code:

(imoogi-require "18-terminal" 'ghostel)

(use-package ghostel
  :ensure t
  :commands (ghostel ghostel-project)
  :bind (("C-c t" . ghostel))
  :init
  ;; 네이티브 모듈을 저장소 동봉 경로에서 로드한다. 패키지 디렉터리 밖이라
  ;; 재-vendoring 시 덮어써지지 않는다. air-gap: 자동 다운로드/빌드 비활성.
  (setq ghostel-module-directory
        (expand-file-name "vendor/ghostel-module/" imoogi-emacs-dir))
  (setq ghostel-module-auto-install nil)
  (setq ghostel-max-scrollback 10000))

;;; ghostel-ime — 터미널 안에서 Emacs 한글/CJK 입력기(quail) 동작
;; ghostel 패키지에 함께 포함되므로 :ensure nil.
(use-package ghostel-ime
  :ensure nil
  :after ghostel
  :hook (ghostel-mode . ghostel-ime-mode))

(provide 'imoogi-terminal)
;;; 18-terminal.el ends here
