;;; 16-languages.el --- 언어/파일타입 메이저 모드 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; 가벼운 메이저 모드 모음. 대부분 해당 확장자를 열 때만 로드된다(:mode).
;; Tree-sitter 기반(*-ts-mode)이 있으면 그쪽이 더 정확하지만, 여기서는
;; 별도 설치 없이 동작하는 전통 모드를 동봉한다.

;;; Code:

(imoogi-require "16-languages" 'git-modes 'yaml-mode 'dockerfile-mode 'gnuplot
                'lua-mode 'jinja2-mode 'csv-mode 'go-mode 'rust-mode 'crontab-mode
                'nginx-mode 'hcl-mode 'nix-mode 'fish-mode 'vimrc-mode 'jenkinsfile-mode
                'clojure-mode 'kotlin-mode 'typescript-mode 'web-mode)

(defun imoogi-eglot-server-available-p (server)
  "Return non-nil when SERVER command is available locally."
  (let ((command (if (listp server) (car server) server)))
    (and (stringp command) (executable-find command))))

(defun imoogi-eglot-register-if-available (modes server)
  "Register SERVER for MODES when its executable exists."
  (when (imoogi-eglot-server-available-p server)
    (with-eval-after-load 'eglot
      (add-to-list 'eglot-server-programs (cons modes server)))))

(defun imoogi-eglot-ensure-if-server-available (server)
  "Start Eglot only when SERVER executable exists locally."
  (when (and (imoogi-eglot-server-available-p server)
             (require 'eglot nil t))
    (eglot-ensure)))

;;; Git 관련 파일(.gitignore/.gitconfig/.gitattributes)
(use-package git-modes
  :ensure t
  :mode (("/\\.gitignore\\'"     . gitignore-mode)
         ("/\\.gitconfig\\'"     . gitconfig-mode)
         ("/\\.git/config\\'"    . gitconfig-mode)
         ("/\\.gitmodules\\'"    . gitconfig-mode)
         ("/\\.gitattributes\\'" . gitattributes-mode)))

;;; HTML — 닫는 태그 자동 삽입(내장 sgml-mode)
(use-package sgml-mode
  :ensure nil
  :hook ((html-mode mhtml-mode) . sgml-electric-tag-pair-mode))

;;; YAML
(use-package yaml-mode
  :ensure t
  :mode (("\\.ya?ml\\'" . yaml-mode)))

;;; Dockerfile
(use-package dockerfile-mode
  :ensure t
  :mode ("Dockerfile\\'" . dockerfile-mode))

;;; Gnuplot
(use-package gnuplot
  :ensure t
  :mode ("\\.gp\\'" . gnuplot-mode))

;;; Lua
(use-package lua-mode
  :ensure t
  :mode ("\\.lua\\'" . lua-mode))

;;; Jinja2 템플릿
(use-package jinja2-mode
  :ensure t
  :mode ("\\.j2\\'" . jinja2-mode))

;;; CSV (자동 열 정렬)
(use-package csv-mode
  :ensure t
  :mode ("\\.csv\\'" . csv-mode)
  :hook ((csv-mode . csv-align-mode)
         (csv-mode . csv-guess-set-separator))
  :custom
  (csv-align-max-width 100)
  (csv-separators '("," ";" " " "|" "\t")))

;;; Go
(use-package go-mode
  :ensure t
  :mode ("\\.go\\'" . go-mode))

;;; Rust
(use-package rust-mode
  :ensure t
  :mode ("\\.rs\\'" . rust-mode)
  :custom
  (rust-indent-offset 2))

;;; crontab
(use-package crontab-mode
  :ensure t
  :mode ("/crontab\\(\\.X*[[:alnum:]]+\\)?\\'" . crontab-mode))

;;; Nginx 설정
(use-package nginx-mode
  :ensure t
  :mode (("nginx\\.conf\\'" . nginx-mode)
         ("/nginx/.+\\.conf\\'" . nginx-mode)))

;;; HCL (Terraform 등)
(use-package hcl-mode
  :ensure t
  :mode ("\\.hcl\\'" . hcl-mode))

;;; Nix
(use-package nix-mode
  :ensure t
  :mode ("\\.nix\\'" . nix-mode))

;;; Fish 셸
(use-package fish-mode
  :ensure t
  :mode ("\\.fish\\'" . fish-mode))

;;; Vim 설정 파일
(use-package vimrc-mode
  :ensure t
  :mode ("\\.vim\\(rc\\)?\\'" . vimrc-mode))

;;; Jenkinsfile
(use-package jenkinsfile-mode
  :ensure t
  :mode ("Jenkinsfile\\'" . jenkinsfile-mode))

;;; Clojure / ClojureScript / EDN
(use-package clojure-mode
  :ensure t
  :mode (("\\.clj\\'"  . clojure-mode)
         ("\\.cljc\\'" . clojurec-mode)
         ("\\.cljs\\'" . clojurescript-mode)
         ("\\.edn\\'"  . edn-mode))
  :hook ((clojure-mode clojurec-mode clojurescript-mode)
         . (lambda ()
             (imoogi-eglot-ensure-if-server-available "clojure-lsp")))
  :config
  (imoogi-eglot-register-if-available
   '(clojure-mode clojurec-mode clojurescript-mode)
   '("clojure-lsp")))

;;; Java (내장 cc-mode)
(use-package cc-mode
  :ensure nil
  :mode ("\\.java\\'" . java-mode)
  :hook (java-mode . (lambda ()
                       (imoogi-eglot-ensure-if-server-available "jdtls")))
  :config
  (imoogi-eglot-register-if-available 'java-mode '("jdtls")))

;;; Kotlin
(use-package kotlin-mode
  :ensure t
  :mode (("\\.kt\\'"  . kotlin-mode)
         ("\\.kts\\'" . kotlin-mode))
  :hook (kotlin-mode . (lambda ()
                         (imoogi-eglot-ensure-if-server-available "kotlin-language-server")))
  :config
  (imoogi-eglot-register-if-available
   'kotlin-mode
   '("kotlin-language-server")))

;;; TypeScript
(use-package typescript-mode
  :ensure t
  :mode ("\\.ts\\'" . typescript-mode)
  :custom
  (typescript-indent-level 2)
  :hook (typescript-mode . (lambda ()
                             (imoogi-eglot-ensure-if-server-available "typescript-language-server")))
  :config
  (imoogi-eglot-register-if-available
   '(typescript-mode web-mode)
   '("typescript-language-server" "--stdio")))

;;; TSX/JSX 템플릿
(use-package web-mode
  :ensure t
  :mode (("\\.tsx\\'" . web-mode)
         ("\\.jsx\\'" . web-mode))
  :custom
  (web-mode-content-types-alist '(("jsx" . "\\.tsx\\'")))
  (web-mode-markup-indent-offset 2)
  (web-mode-code-indent-offset 2)
  (web-mode-css-indent-offset 2)
  :hook (web-mode . (lambda ()
                      (when (and buffer-file-name
                                 (string-match-p "\\.tsx\\'" buffer-file-name))
                        (imoogi-eglot-ensure-if-server-available "typescript-language-server")))))

(provide 'imoogi-languages)
;;; 16-languages.el ends here
