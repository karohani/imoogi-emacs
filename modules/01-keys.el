;;; keys.el --- Korean input & key translation -*- lexical-binding: t; -*-

;;; macOS: Emacs 포커스 시 시스템 입력 소스를 영문으로 강제 전환
;; 한글 입력은 Emacs 내장 input-method (S-SPC)로만 처리
(when (eq system-type 'darwin)
  (defvar imoogi--prev-input-source nil
    "Emacs 진입 전 macOS 입력 소스를 저장.")

  (defun imoogi--setup-im-select ()
    "im-select로 포커스 전환 시 입력 소스 자동 관리."
    (let ((im-select (executable-find "im-select")))
      (when im-select
        ;; Emacs 진입: 현재 입력 소스 저장 후 영문 전환
        (add-hook 'focus-in-hook
                  (lambda ()
                    (setq imoogi--prev-input-source
                          (string-trim (shell-command-to-string im-select)))
                    (call-process im-select nil nil nil "com.apple.keylayout.ABC")))
        ;; Emacs 이탈: 이전 입력 소스로 복원
        (add-hook 'focus-out-hook
                  (lambda ()
                    (when imoogi--prev-input-source
                      (call-process im-select nil nil nil imoogi--prev-input-source)))))))
  (cond
   ;; emacs-mac 포트
   ((fboundp 'mac-auto-ascii-mode)
    (mac-auto-ascii-mode 1))
   ;; NS 빌드: im-select CLI로 포커스 시 영문 전환
   ((executable-find "im-select")
    (imoogi--setup-im-select))
   ;; im-select 미설치: 망분리 원칙상 부팅 중 자동 설치(brew)는 하지 않는다.
   ;; 필요 시 사용자가 직접 설치: brew tap daipeihust/tap && brew install im-select
   (t
    (message "imoogi: im-select 미설치 — 포커스 시 입력 소스 자동 전환 비활성. (brew install im-select 로 활성화)"))))

;;; Korean input method (Shift+Space로 한영 전환)
(global-set-key (kbd "S-SPC") 'toggle-input-method)
(setq default-input-method "korean-hangul")

;;; Korean key translation (한글 입력 상태에서도 주요 키바인딩 동작)
;; 한글 두벌식 자판: 영문 키 → 한글 자모 매핑
;; a→ㅁ b→ㅠ c→ㅊ d→ㅇ e→ㄷ f→ㄹ g→ㅎ h→ㅗ i→ㅑ j→ㅓ k→ㅏ l→ㅣ m→ㅡ
;; n→ㅜ o→ㅐ p→ㅔ q→ㅂ r→ㄱ s→ㄴ t→ㅅ u→ㅕ v→ㅍ w→ㅈ x→ㅌ y→ㅛ z→ㅋ
(let ((pairs '(("ㅁ" . "a") ("ㅠ" . "b") ("ㅊ" . "c") ("ㅇ" . "d")
               ("ㄷ" . "e") ("ㄹ" . "f") ("ㅎ" . "g") ("ㅗ" . "h")
               ("ㅑ" . "i") ("ㅓ" . "j") ("ㅏ" . "k") ("ㅣ" . "l")
               ("ㅡ" . "m") ("ㅜ" . "n") ("ㅐ" . "o") ("ㅔ" . "p")
               ("ㅂ" . "q") ("ㄱ" . "r") ("ㄴ" . "s") ("ㅅ" . "t")
               ("ㅕ" . "u") ("ㅍ" . "v") ("ㅈ" . "w") ("ㅌ" . "x")
               ("ㅛ" . "y") ("ㅋ" . "z"))))
  (dolist (p pairs)
    (define-key key-translation-map (kbd (concat "C-" (car p))) (kbd (concat "C-" (cdr p))))
    (define-key key-translation-map (kbd (concat "M-" (car p))) (kbd (concat "M-" (cdr p))))))

;;; macOS Cmd 키 (복사/붙여넣기/잘라내기/되돌리기/전체선택)
(global-set-key (kbd "s-c") 'kill-ring-save)
(global-set-key (kbd "s-v") 'yank)
(global-set-key (kbd "s-x") 'kill-region)
(global-set-key (kbd "s-z") 'undo)
(global-set-key (kbd "s-a") 'mark-whole-buffer)
;; Super key 한글 매핑
(define-key key-translation-map (kbd "s-ㅊ") (kbd "s-c"))
(define-key key-translation-map (kbd "s-ㅍ") (kbd "s-v"))
(define-key key-translation-map (kbd "s-ㅌ") (kbd "s-x"))
(define-key key-translation-map (kbd "s-ㅋ") (kbd "s-z"))
(define-key key-translation-map (kbd "s-ㅁ") (kbd "s-a"))

(provide 'imoogi-keys)
;;; keys.el ends here
