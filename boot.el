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
(defvar imoogi-failed-modules nil
  "부팅 중 건너뛴 모듈 이름 목록(로드 순서).
사전조건 누락이나 로딩 오류로 건너뛴 모듈이 여기 쌓인다.
`M-x describe-variable imoogi-failed-modules' 로 확인할 수 있고,
tests/assert-boot.el 이 이 값을 설치 검증의 판정 근거로 쓴다.")

;; 이 파일은 `imoogi-reload' 로 다시 로드될 수 있다. 목록을 비우지 않으면
;; 재로드마다 같은 모듈이 중복 누적돼(예: ("06-git" "06-git")) 이 변수를 판정
;; 근거로 쓰는 쪽이 잘못 읽는다. 매 로드가 그 로드의 결과만 담도록 초기화한다.
(setq imoogi-failed-modules nil)

(dolist (module '("00-defaults"
                  "01-keys"
                  "02-completion"
                  "03-which-key"
                  "04-projects"
                  "05-transient"
                  "06-git"
                  "07-treemacs"
                  "08-obsidian"
                  "09-autorevert"
                  "10-theme"
                  "11-editing"
                  "12-navigation"
                  "13-system"
                  "14-org"
                  "15-markdown"
                  "16-elisp"
                  "17-lsp"
                  "18-languages"
                  "19-folding"
                  "20-terminal"
                  "21-native-compile"))
  (condition-case err
      (load (expand-file-name (concat "modules/" module) imoogi-emacs-dir))
    (error
     (setq imoogi-failed-modules
           (append imoogi-failed-modules (list module)))
     (display-warning 'imoogi
                      (format "모듈 %s 로딩 실패(건너뜀): %s"
                              module (error-message-string err))
                      :error))))

;;; Reload
(defun imoogi-reload ()
  "설정을 다시 읽는다(Emacs 재시작 불필요).

건너뛴 모듈이 있으면 그 목록까지 함께 알린다 — 조용히 절반만 로드된 상태를
성공으로 착각하지 않도록. 전체 결과는 `imoogi-failed-modules' 에 남는다.

한계: 이미 정의된 것을 덮어쓰는 것이라 되돌리지는 못한다. 키바인딩 제거,
모드 해제, 훅에서 뺀 함수 같은 \"없앤 변경\"은 재시작해야 반영된다."
  (interactive)
  (load-file (expand-file-name "boot.el" imoogi-emacs-dir))
  (if imoogi-failed-modules
      (message "imoogi-emacs 재로드 완료 — 건너뛴 모듈 %d개: %s"
               (length imoogi-failed-modules)
               (string-join imoogi-failed-modules ", "))
    (message "imoogi-emacs 재로드 완료 — 모든 모듈 정상.")))

(provide 'imoogi-boot)
;;; boot.el ends here
