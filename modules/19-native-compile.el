;;; 19-native-compile.el --- compile-angel 네이티브 컴파일 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; compile-angel 은 로드되는 .el 을 바이트/네이티브 컴파일해 실행 성능을 높인다.
;; `compile-angel-on-load-mode' 는 이후 로드뿐 아니라 이미 로드된 패키지도
;; 소급해 컴파일하므로 마지막 모듈로 로드해도 전부 처리된다.
;;
;; 주의(air-gap): vendor 에는 .elc 를 동봉하지만 .eln(네이티브)은 머신별 캐시라
;; 동봉하지 않는다. 따라서 새 타겟의 "첫 부팅"에서 compile-angel 이 다량을
;; 네이티브 컴파일하느라 수십 초~수 분 걸릴 수 있다(오프라인, 1회성). 이후
;; 부팅은 빨라진다. 네트워크는 필요 없다.

;;; Code:

(imoogi-require "19-native-compile" 'compile-angel)

(use-package compile-angel
  :ensure t
  :demand t
  :config
  ;; 패키지 설치 시 컴파일은 compile-angel 이 담당하므로 끈다.
  (setq package-native-compile nil)
  (setq compile-angel-verbose nil)
  ;; init 계열 파일은 컴파일 대상에서 제외(경로 suffix 매칭).
  (dolist (suffix '("/init.el" "/early-init.el" "/boot.el"))
    (push suffix compile-angel-excluded-path-suffixes))
  (compile-angel-on-load-mode 1))

(provide 'imoogi-native-compile)
;;; 19-native-compile.el ends here
