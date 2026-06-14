;;; 13-system.el --- 시스템 통합 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; exec-path-from-shell(환경변수 동기화), Emacs server, buffer-terminator,
;; persist-text-scale.

;;; Code:

(imoogi-require "13-system" 'exec-path-from-shell 'buffer-terminator 'persist-text-scale)

;;; exec-path-from-shell — 셸 환경변수를 GUI Emacs 로 동기화 (macOS 필수)
;; GUI/데몬으로 띄운 Emacs 는 로그인 셸의 PATH 등을 물려받지 못한다.
;; 로컬 셸 호출이라 네트워크 불필요(망분리 안전).
(use-package exec-path-from-shell
  :ensure t
  :if (and (or (display-graphic-p) (daemonp))
           (eq system-type 'darwin))
  :demand t
  :config
  (dolist (var '("TMPDIR"
                 "SSH_AUTH_SOCK" "SSH_AGENT_PID"
                 "GPG_AGENT_INFO"
                 "LANG" "LC_CTYPE"))
    (add-to-list 'exec-path-from-shell-variables var))
  (exec-path-from-shell-initialize))

;;; Emacs server — emacsclient 로 기존 세션에 파일 열기
(use-package server
  :ensure nil
  :if (not (daemonp))
  :hook (after-init . imoogi--server-start)
  :preface
  (defun imoogi--server-start ()
    "서버가 떠 있지 않으면 시작한다."
    (require 'server)
    (unless (server-running-p)
      (server-start))))

;;; buffer-terminator — 오래 비활성인 버퍼 자동 정리(보이는/수정된 버퍼는 보호)
(use-package buffer-terminator
  :ensure t
  :custom
  (buffer-terminator-verbose nil)
  (buffer-terminator-inactivity-timeout (* 30 60)) ; 30분
  (buffer-terminator-interval (* 10 60))           ; 10분마다
  :config
  (buffer-terminator-mode 1))

;;; persist-text-scale — 텍스트 확대/축소 상태를 세션 간 유지
(use-package persist-text-scale
  :ensure t
  :hook (after-init . persist-text-scale-mode))

(provide 'imoogi-system)
;;; 13-system.el ends here
