;;; 13-system.el --- 시스템 통합 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; exec-path-from-shell(환경변수 동기화), Emacs server, buffer-terminator,
;; persist-text-scale.

;;; Code:

(imoogi-require "13-system" 'exec-path-from-shell 'buffer-terminator
                'persist-text-scale 'gnutls 'json 'transient)

(eval-when-compile (require 'transient))

(defcustom imoogi-system-config-file
  (expand-file-name "imoogi-system.json" user-emacs-directory)
  "Persistent non-secret Emacs system configuration."
  :type 'file)

(defcustom imoogi-ca-certificate-file nil
  "Optional PEM CA bundle added to Emacs' global GnuTLS trust store."
  :type '(choice (const :tag "Use system trust store only" nil) file))

(defun imoogi-system--write-config ()
  "Persist the non-secret system configuration atomically."
  (make-directory (file-name-directory imoogi-system-config-file) t)
  (let ((temporary (make-temp-file
                    (expand-file-name ".imoogi-system-"
                                      (file-name-directory
                                       imoogi-system-config-file))
                    nil ".json")))
    (unwind-protect
        (progn
          (with-temp-file temporary
            (insert (json-serialize
                     `((ca_certificate_file . ,imoogi-ca-certificate-file))
                     :null-object nil))
            (insert "\n"))
          (set-file-modes temporary #o600)
          (rename-file temporary imoogi-system-config-file t))
      (when (file-exists-p temporary) (delete-file temporary)))))

(defun imoogi-system-configure-ca-certificate ()
  "Apply `imoogi-ca-certificate-file' to Emacs' global GnuTLS trust store."
  (when imoogi-ca-certificate-file
    (let ((file (expand-file-name imoogi-ca-certificate-file)))
      (unless (file-readable-p file)
        (user-error "Emacs 사설 CA 파일을 읽을 수 없습니다: %s" file))
      (require 'gnutls)
      (add-to-list 'gnutls-trustfiles file)
      file)))

(defun imoogi-system-set-ca-certificate (file)
  "Select and persist FILE as Emacs' global private CA bundle."
  (interactive (list (read-file-name "사설 CA PEM 파일: " nil nil t)))
  (setq imoogi-ca-certificate-file (expand-file-name file))
  (imoogi-system-configure-ca-certificate)
  (imoogi-system--write-config)
  (message "Emacs 전역 사설 CA를 설정했습니다: %s" imoogi-ca-certificate-file))

(defun imoogi-system-clear-ca-certificate ()
  "Remove the persisted private CA bundle from Emacs' global trust store."
  (interactive)
  (when imoogi-ca-certificate-file
    (setq gnutls-trustfiles
          (delete (expand-file-name imoogi-ca-certificate-file)
                  gnutls-trustfiles)))
  (setq imoogi-ca-certificate-file nil)
  (imoogi-system--write-config)
  (message "Emacs 전역 사설 CA 설정을 해제했습니다"))

(when (file-readable-p imoogi-system-config-file)
  (condition-case err
      (with-temp-buffer
        (insert-file-contents imoogi-system-config-file)
        (let ((data (json-parse-buffer :object-type 'alist)))
          (setq imoogi-ca-certificate-file
                (alist-get 'ca_certificate_file data))))
    (error
     (display-warning 'imoogi-system
                      (format "시스템 설정을 읽지 못했습니다: %s"
                              (error-message-string err))
                      :warning))))
(imoogi-system-configure-ca-certificate)

(with-eval-after-load 'imoogi-transient
  (add-to-list 'imoogi-transient--purposes
               '(imoogi-system-transient . "Emacs 전역 네트워크와 인증서를 설정합니다."))
  (transient-define-prefix imoogi-system-transient ()
    "Emacs global system settings."
    [["네트워크·인증서 --------"
      ("P" "사설 CA PEM 설정" imoogi-system-set-ca-certificate)
      ("X" "사설 CA 설정 해제" imoogi-system-clear-ca-certificate)
      ("q" "종료" transient-quit-one)]])
  (transient-append-suffix 'imoogi-transient-master "R"
    '("s" "시스템 설정" imoogi-system-transient)))

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
