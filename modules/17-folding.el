;;; 17-folding.el --- 코드 폴딩 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; kirigami(통합 폴드 인터페이스) + 내장 outline/hs-minor-mode + outline-indent.
;; treesit-fold 는 언어별 tree-sitter 문법(별도 설치/빌드)이 필요해 air-gap 에서는
;; 제외한다. 필요하면 문법을 갖춘 환경에서 packages.el 에 추가.

;;; Code:

;;; kirigami — 폴드 열기/닫기 통합 인터페이스 (C-c z 접두)
(use-package kirigami
  :ensure t
  :bind (("C-c z o" . kirigami-open-fold)            ; 현재 폴드 열기
         ("C-c z O" . kirigami-open-fold-rec)        ; 재귀적으로 열기
         ("C-c z r" . kirigami-open-folds)           ; 전부 열기
         ("C-c z c" . kirigami-close-fold)           ; 현재 폴드 닫기
         ("C-c z m" . kirigami-close-folds)          ; 전부 닫기
         ("C-c z a" . kirigami-toggle-fold)))        ; 토글

;;; outline-minor-mode (내장) — 헤딩/구조 기반 폴딩
(use-package outline
  :ensure nil
  :hook ((emacs-lisp-mode . outline-minor-mode)
         (lisp-mode . outline-minor-mode)
         (conf-mode . outline-minor-mode)
         (markdown-mode . outline-minor-mode)
         (diff-mode . outline-minor-mode)
         (outline-minor-mode
          . (lambda ()
              ;; 접힌 텍스트 표시를 "..." 대신 " ▼" 로
              (let* ((dt (or buffer-display-table (make-display-table)))
                     (off (* (face-id 'shadow) (ash 1 22)))
                     (val (vconcat (mapcar (lambda (c) (+ off c)) " ▼"))))
                (set-display-table-slot dt 'selective-display val)
                (setq buffer-display-table dt))))))

;;; hs-minor-mode (내장) — 중괄호/블록 기반 폴딩
(dolist (hook '(c-mode-hook c++-mode-hook java-mode-hook sh-mode-hook html-mode-hook))
  (add-hook hook #'hs-minor-mode))

;;; outline-indent — 들여쓰기 기반 폴딩(Python/YAML 등)
(use-package outline-indent
  :ensure t
  :commands outline-indent-minor-mode
  :custom
  (outline-indent-ellipsis " ▼")
  :init
  (dolist (hook '(python-mode-hook python-ts-mode-hook
                                   yaml-mode-hook yaml-ts-mode-hook))
    (add-hook hook #'outline-indent-minor-mode)))

(provide 'imoogi-folding)
;;; 17-folding.el ends here
