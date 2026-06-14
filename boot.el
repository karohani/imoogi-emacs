;;; boot.el --- imoogi-emacs entry point -*- lexical-binding: t; -*-

(setq package-enable-at-startup nil)

;;; imoogi-emacs directory
(defvar imoogi-emacs-dir
  (file-name-directory (or load-file-name buffer-file-name))
  "Root directory of imoogi-emacs configuration.")

;;; 패키지 — 망분리(air-gap) 대응 vendoring
;; 모든 패키지를 저장소 안 vendor/elpa/ 에 동봉한다. 저장소를 클론해 들고
;; 들어가면 인터넷 없이 그대로 동작한다(부팅 경로에서 네트워크 접근 없음).
;; vendor/ 채우기/갱신은 온라인 머신에서: emacs --batch -Q -l scripts/vendor.el
(require 'package)
(setq package-user-dir (expand-file-name "vendor/elpa/" imoogi-emacs-dir))
;; 아래 아카이브는 온라인 vendoring 시에만 의미가 있다. 런타임에는
;; package-refresh-contents 를 호출하지 않으므로 네트워크에 접근하지 않는다.
(setq package-archives '(("gnu"    . "https://elpa.gnu.org/packages/")
                         ("nongnu" . "https://elpa.nongnu.org/nongnu/")
                         ("melpa"  . "https://melpa.org/packages/")))
(package-initialize)

;;; use-package
(require 'use-package)
;; 오프라인 원칙: 누락 패키지를 네트워크로 받지 않는다. 동봉된 vendor/ 만 사용.
;; (개별 모듈의 명시적 :ensure t 는 이미 설치돼 있으면 무시된다)
(setq use-package-always-ensure nil)

;;; Backup & auto-save files → ~/.emacs.d/.cache/ 로 모으기
(let ((backup-dir  (expand-file-name ".cache/backups/"  user-emacs-directory))
      (autosave-dir (expand-file-name ".cache/autosaves/" user-emacs-directory)))
  (make-directory backup-dir t)
  (make-directory autosave-dir t)
  (setq backup-directory-alist         `(("." . ,backup-dir))
        auto-save-file-name-transforms `((".*" ,autosave-dir t))
        lock-file-directory             autosave-dir))

;; 스크롤 등 편집 기본값은 modules/00-defaults.el 에서 통합 관리한다.

;;; 모듈 사전조건 점검 헬퍼
;; 각 모듈 파일은 맨 위에서 (imoogi-require "NN-name" 'pkg ...) 로 필요 라이브러리가
;; 로드 가능한지(vendor 에 동봉됐는지/내장인지) 확인한다. 하나라도 없으면 error 를
;; 시그널하고, 아래 모듈 로더가 condition-case 로 받아 그 모듈만 건너뛴다.
(defun imoogi-require (module &rest packages)
  "MODULE 이 요구하는 PACKAGES 가 모두 로드 가능한지 확인한다.
누락 시 error 를 시그널해 해당 모듈 로딩을 중단시킨다(나머지 모듈은 계속)."
  (let ((missing (seq-remove (lambda (p) (locate-library (symbol-name p)))
                             packages)))
    (when missing
      (error "[%s] 누락 패키지 %s — packages.el 에 추가 후 재-vendoring 필요"
             module missing))))

;;; Load modules
(dolist (module '("00-defaults"
                  "01-keys"
                  "02-completion"
                  "03-which-key"
                  "04-projects"
                  "05-hydra"
                  "06-git"
                  "07-treemacs"
                  "08-obsidian"
                  "09-autorevert"
                  "10-theme"
                  "11-editing"
                  "12-navigation"
                  "13-system"
                  "14-org-markdown"
                  "15-elisp"
                  "16-languages"
                  "17-folding"
                  "18-terminal"
                  "19-native-compile"))
  (condition-case err
      (load (expand-file-name (concat "modules/" module) imoogi-emacs-dir))
    (error
     (display-warning 'imoogi
                      (format "모듈 %s 로딩 실패(건너뜀): %s"
                              module (error-message-string err))
                      :error))))

;;; Reload
(defun imoogi-reload ()
  "Reload imoogi-emacs configuration."
  (interactive)
  (load-file (expand-file-name "boot.el" imoogi-emacs-dir))
  (message "imoogi-emacs reloaded."))

(provide 'imoogi-boot)
;;; boot.el ends here
