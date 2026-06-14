;;; vendor.el --- 패키지 vendoring 스크립트 (온라인 전용) -*- lexical-binding: t; -*-

;; 망분리(air-gap) 대응. 인터넷이 되는 "빌드 머신"에서 실행해
;; packages.el 의 패키지 + 전이 의존성을 저장소 안 vendor/elpa/ 로 설치하고,
;; 감사·재현용 packages.lock 을 생성한다.
;;
;; 실행:
;;   emacs --batch -Q -l scripts/vendor.el           # 누락 패키지만 설치
;;   emacs --batch -Q -l scripts/vendor.el -- upgrade # 모두 최신으로 갱신
;;
;; 이후 vendor/ 와 packages.lock 변경분을 git 커밋 → 폐쇄망으로 반입.
;; 타겟 Emacs 메이저 버전을 빌드 머신과 동일하게 유지할 것(.elc 호환).

;;; Code:

(require 'cl-lib)

(defvar imoogi-vendor--root
  (file-name-directory
   (directory-file-name
    (file-name-directory (or load-file-name buffer-file-name))))
  "저장소 루트(scripts/ 의 상위).")

(defvar imoogi-vendor--upgrade
  (member "upgrade" command-line-args-left)
  "Non-nil 이면 설치된 패키지도 최신으로 갱신한다.")

;;; 매니페스트 로드
(load (expand-file-name "packages.el" imoogi-vendor--root) nil t)

;;; package.el 을 저장소 로컬 디렉터리로 향하게 설정
(require 'package)
(setq package-user-dir (expand-file-name "vendor/elpa/" imoogi-vendor--root))
(setq package-archives
      '(("gnu"    . "https://elpa.gnu.org/packages/")
        ("nongnu" . "https://elpa.nongnu.org/nongnu/")
        ("melpa"  . "https://melpa.org/packages/")))
;; 안정성 위해 GNU/NonGNU 우선, MELPA(rolling) 후순위.
(setq package-archive-priorities '(("gnu" . 99) ("nongnu" . 80) ("melpa" . 70)))

(message "==> vendoring 대상: %s" package-user-dir)
(make-directory package-user-dir t)

(package-initialize)
(message "==> 아카이브 갱신 중...")
(package-refresh-contents)

(when imoogi-vendor--upgrade
  (message "==> upgrade 모드: 설치된 패키지 갱신")
  (when (fboundp 'package-upgrade-all)
    (ignore-errors (package-upgrade-all nil))))

;;; 설치
(dolist (pkg imoogi-required-packages)
  (condition-case err
      (if (package-installed-p pkg)
          (message "    이미 설치됨: %s" pkg)
        (message "    설치: %s" pkg)
        (package-install pkg))
    (error (message "!!! %s 설치 실패: %s" pkg (error-message-string err)))))

;;; packages.lock 생성 — 설치된 모든 패키지(전이 의존성 포함)의 버전 기록
(let ((lock (expand-file-name "packages.lock" imoogi-vendor--root))
      (entries '()))
  (dolist (p package-alist)
    (let* ((name (car p))
           (desc (cadr p))
           (ver (package-version-join (package-desc-version desc)))
           (arc (or (package-desc-archive desc) "?")))
      (push (format "%-28s %-22s %s" name ver arc) entries)))
  (with-temp-file lock
    (insert ";; imoogi-emacs packages.lock — vendor/elpa/ 에 동결된 패키지 버전\n")
    (insert (format ";; 생성: emacs %s, 총 %d 패키지\n" emacs-version (length entries)))
    (insert ";; NAME                         VERSION                ARCHIVE\n")
    (dolist (line (sort entries #'string<))
      (insert line "\n")))
  (message "==> lockfile 기록: %s (%d 패키지)" lock (length entries)))

;;; nerd-icons 폰트도 저장소에 동봉(best-effort) — 폐쇄망에서 아이콘 표시용
(let ((font-dir (expand-file-name "assets/fonts/" imoogi-vendor--root)))
  (condition-case err
      (progn
        (require 'nerd-icons)
        (make-directory font-dir t)
        ;; nerd-icons 가 받는 심볼 폰트(NFM.ttf)를 저장소로 복사
        (let* ((url (concat (if (boundp 'nerd-icons-font-base-url)
                                nerd-icons-font-base-url
                              "https://raw.githubusercontent.com/rainstormstudio/nerd-icons.el/main/fonts/")
                            "NFM.ttf"))
               (dest (expand-file-name "NFM.ttf" font-dir)))
          (url-copy-file url dest t)
          (message "==> 폰트 동봉: %s" dest)))
    (error (message "!!! 폰트 동봉 건너뜀(수동 처리 필요): %s"
                    (error-message-string err)))))

(message "==> vendoring 완료. `git status`로 vendor/ · packages.lock 확인 후 커밋하세요.")

;;; vendor.el ends here
