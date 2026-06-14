;;; early-init.el --- imoogi-emacs early init -*- lexical-binding: t; -*-

;; Emacs 27+ 는 패키지/프레임 초기화 이전에 이 파일을 먼저 로드한다.
;; minimal-emacs.d (James Cherti) 의 스타트업 최적화 기법을 imoogi-emacs 에 맞게
;; 추려 옮긴 것이다. 무거운 패키지 설정은 boot.el / modules/ 에서 담당한다.

;;; Code:

;;; 가비지 컬렉션 — 스타트업 동안 GC 를 미뤘다가 부팅 후 합리적인 값으로 복원
;; 부팅 중 잦은 GC 가 체감 시작 시간을 크게 늘린다. 임계값을 최대로 올려두고
;; emacs-startup-hook 에서 32MB 로 되돌린다.
(defvar imoogi--gc-cons-threshold (* 32 1024 1024)
  "부팅 완료 후 복원할 `gc-cons-threshold' 값.")

(setq gc-cons-threshold most-positive-fixnum
      gc-cons-percentage 1.0)

(add-hook 'emacs-startup-hook
          (lambda ()
            (setq gc-cons-threshold imoogi--gc-cons-threshold
                  gc-cons-percentage 0.1))
          105)

;;; file-name-handler-alist 최적화
;; 부팅 중 로드되는 모든 파일은 이 alist 의 핸들러를 순회한다. 부팅 동안 비워두면
;; 파일/패키지 로드가 빨라진다. 부팅 후 원래 값과 병합해 복원한다.
(defvar imoogi--file-name-handler-alist
  (default-toplevel-value 'file-name-handler-alist))

(unless (or (daemonp) noninteractive init-file-debug)
  (set-default-toplevel-value 'file-name-handler-alist nil)
  (add-hook 'emacs-startup-hook
            (lambda ()
              (set-default-toplevel-value
               'file-name-handler-alist
               (delete-dups (append file-name-handler-alist
                                    imoogi--file-name-handler-alist))))
            101))

;;; 네이티브 컴파일 / 바이트 컴파일 — 경고 소음 줄이기
(setq native-comp-async-report-warnings-errors 'silent
      native-comp-warning-on-missing-source nil)
(setq byte-compile-warnings nil
      warning-minimum-level :error)
;; defvaralias / lexical-binding 누락 경고는 서드파티 패키지에서 흔하고 손쓸 수
;; 없으므로 억제한다.
(setq warning-suppress-types '((defvaralias) (lexical-binding))
      warning-inhibit-types '((files missing-lexbind-cookie)))
;; 구식 advice API 의 재정의 경고도 끈다.
(setq ad-redefinition-action 'accept)

;;; 프로세스 / I-O 성능
;; LSP(eglot 등) 같은 대용량 출력 프로세스의 처리량을 높인다.
(setq read-process-output-max (* 4 1024 1024) ; 4MB
      process-adaptive-read-buffering nil)
;; auto-mode-alist 를 대소문자 무시로 한 번 더 훑는 동작 제거.
(setq auto-mode-case-fold nil)
;; 도메인처럼 보이는 문자열에 핑 보내지 않기.
(setq ffap-machine-p-known 'reject)
;; 새 init 파일을 우선 사용.
(setq load-prefer-newer t)

;;; 언어 환경
(set-language-environment "UTF-8")
;; set-language-environment 가 건드리는 기본 입력기는 원치 않으므로 비활성화.
(setq default-input-method nil)

;;; 프레임 / 폰트 렌더링 성능
(setq frame-inhibit-implied-resize t   ; 부팅 시 프레임 자동 리사이즈 방지
      frame-resize-pixelwise t
      inhibit-compacting-font-caches t) ; 폰트 캐시 압축 비용 회피(메모리↑)
;; 양방향 텍스트(BiDi) 스캔 비활성화로 재표시 속도 소폭 향상.
(setq-default bidi-display-reordering 'left-to-right
              bidi-paragraph-direction 'left-to-right)
(setq bidi-inhibit-bpa t)

;;; 스타트업 화면 / 메시지 억제
(setq inhibit-startup-screen t
      inhibit-splash-screen t
      inhibit-startup-buffer-menu t
      inhibit-startup-echo-area-message user-login-name
      initial-buffer-choice nil
      inhibit-x-resources t)
;; "For information about GNU Emacs..." 에코 메시지 및 스타트업 스크린 무력화.
(advice-add 'display-startup-echo-area-message :override #'ignore)
(advice-add 'display-startup-screen :override #'ignore)

;;; UI 요소 — 프레임 파라미터로 끄기(모드 함수 호출보다 리드로우 비용이 적다)
;; macOS 에서는 메뉴바가 시스템 상단바에 표시돼 비용이 없으므로 그대로 둔다.
(push '(tool-bar-lines . 0) default-frame-alist)
(push '(vertical-scroll-bars) default-frame-alist)
(push '(horizontal-scroll-bars) default-frame-alist)
(setq tool-bar-mode nil
      scroll-bar-mode nil)
(unless (memq window-system '(mac ns))
  (push '(menu-bar-lines . 0) default-frame-alist)
  (setq menu-bar-mode nil))
;; 시스템마다 들쭉날쭉한 GUI 다이얼로그 대신 미니버퍼를 쓴다.
(setq use-file-dialog nil
      use-dialog-box nil)

;;; 보안 — TLS/GnuTLS 강화
(setq gnutls-verify-error t
      gnutls-min-prime-bits 3072)
(setq tls-checktrust gnutls-verify-error)

;;; use-package / package.el 성능 플래그
;; (실제 부트스트랩과 아카이브 설정은 boot.el 에서 수행한다)
(setq use-package-expand-minimally t
      use-package-enable-imenu-support t)
(setq package-enable-at-startup nil)

(provide 'imoogi-early-init)
;;; early-init.el ends here
