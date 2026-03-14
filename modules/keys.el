;;; keys.el --- Korean input & key translation -*- lexical-binding: t; -*-

;;; Korean input method (Shift+Space로 한영 전환)
(global-set-key (kbd "S-SPC") 'toggle-input-method)
(setq default-input-method "korean-hangul")

;;; Korean key translation (한글 입력 상태에서도 주요 키바인딩 동작)
(define-key key-translation-map (kbd "C-ㅎ") (kbd "C-g"))
(define-key key-translation-map (kbd "C-ㅓ") (kbd "C-j"))
(define-key key-translation-map (kbd "C-ㅐ") (kbd "C-o"))
(define-key key-translation-map (kbd "C-ㅈ") (kbd "C-w"))
(define-key key-translation-map (kbd "C-ㅛ") (kbd "C-y"))
(define-key key-translation-map (kbd "C-ㅊ") (kbd "C-c"))
(define-key key-translation-map (kbd "C-ㅌ") (kbd "C-x"))
(define-key key-translation-map (kbd "C-ㄴ") (kbd "C-s"))
(define-key key-translation-map (kbd "C-ㄱ") (kbd "C-r"))
(define-key key-translation-map (kbd "C-ㅁ") (kbd "C-a"))
(define-key key-translation-map (kbd "C-ㄷ") (kbd "C-e"))
;; Meta key
(define-key key-translation-map (kbd "M-ㅌ") (kbd "M-x"))
(define-key key-translation-map (kbd "M-ㅈ") (kbd "M-w"))
(define-key key-translation-map (kbd "M-ㅛ") (kbd "M-y"))
(define-key key-translation-map (kbd "M-ㄹ") (kbd "M-f"))
(define-key key-translation-map (kbd "M-ㅠ") (kbd "M-b"))
(define-key key-translation-map (kbd "M-ㅇ") (kbd "M-d"))

(provide 'imoogi-keys)
;;; keys.el ends here
