;;; 18-languages.el --- 언어/파일타입 메이저 모드 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; 가벼운 메이저 모드 모음. 대부분 해당 확장자를 열 때만 로드된다(:mode).
;; Tree-sitter 문법이 있으면 내장 *-ts-mode 로 자동 전환하고, 없으면
;; 별도 설치 없이 동작하는 전통 모드를 그대로 쓴다.

;;; Code:

(imoogi-require "18-languages" 'git-modes 'yaml-mode 'dockerfile-mode 'gnuplot
                'lua-mode 'jinja2-mode 'csv-mode 'go-mode 'rust-mode 'crontab-mode
                'nginx-mode 'hcl-mode 'nix-mode 'fish-mode 'vimrc-mode 'jenkinsfile-mode
                'clojure-mode 'kotlin-mode 'typescript-mode 'web-mode)

(defvar imoogi-treesit-grammar-dir
  (expand-file-name "vendor/tree-sitter/" imoogi-emacs-dir)
  "Directory for vendored tree-sitter grammar libraries.")

(defun imoogi-treesit-remap-if-available (from-mode to-mode language)
  "Remap FROM-MODE to TO-MODE when LANGUAGE grammar is available."
  (when (and (fboundp 'treesit-available-p)
             (treesit-available-p)
             (treesit-language-available-p language)
             (fboundp to-mode))
    (add-to-list 'major-mode-remap-alist (cons from-mode to-mode))))

(defun imoogi-treesit-auto-mode-if-available (regexp mode language)
  "Use MODE for REGEXP when LANGUAGE grammar is available."
  (when (and (fboundp 'treesit-available-p)
             (treesit-available-p)
             (treesit-language-available-p language)
             (fboundp mode))
    (setq auto-mode-alist
          (let (entries)
            (dolist (entry auto-mode-alist (nreverse entries))
              (unless (and (consp entry)
                           (stringp (car entry))
                           (string= (car entry) regexp))
                (push entry entries)))))
    (push (cons regexp mode) auto-mode-alist)))

(when (require 'treesit nil t)
  (add-to-list 'treesit-extra-load-path imoogi-treesit-grammar-dir)
  (imoogi-treesit-remap-if-available 'sh-mode 'bash-ts-mode 'bash)
  (imoogi-treesit-remap-if-available 'js-mode 'js-ts-mode 'javascript)
  (imoogi-treesit-remap-if-available 'typescript-mode 'typescript-ts-mode 'typescript)
  (imoogi-treesit-auto-mode-if-available "\\.tsx\\'" 'tsx-ts-mode 'tsx)
  (imoogi-treesit-auto-mode-if-available "\\.jsx\\'" 'tsx-ts-mode 'tsx)
  (imoogi-treesit-remap-if-available 'go-mode 'go-ts-mode 'go)
  (imoogi-treesit-remap-if-available 'rust-mode 'rust-ts-mode 'rust)
  (imoogi-treesit-remap-if-available 'java-mode 'java-ts-mode 'java)
  (imoogi-treesit-remap-if-available 'yaml-mode 'yaml-ts-mode 'yaml)
  (imoogi-treesit-remap-if-available 'dockerfile-mode 'dockerfile-ts-mode 'dockerfile)
  (imoogi-treesit-remap-if-available 'html-mode 'html-ts-mode 'html))

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

;;; Shell script (내장 sh-mode)
(use-package sh-script
  :ensure nil
  :mode ("\\.sh\\'" . sh-mode))

;;; JavaScript (내장 js-mode)
(use-package js
  :ensure nil
  :mode (("\\.js\\'"  . js-mode)
         ("\\.mjs\\'" . js-mode)
         ("\\.cjs\\'" . js-mode)))

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
         ("\\.edn\\'"  . edn-mode)))

;;; Java (내장 cc-mode)
(use-package cc-mode
  :ensure nil
  :mode ("\\.java\\'" . java-mode))

;;; Kotlin
(use-package kotlin-mode
  :ensure t
  :mode (("\\.kt\\'"  . kotlin-mode)
         ("\\.kts\\'" . kotlin-mode)))

;;; TypeScript
(use-package typescript-mode
  :ensure t
  :mode ("\\.ts\\'" . typescript-mode)
  :custom
  (typescript-indent-level 2))

;;; TSX/JSX 템플릿
(use-package web-mode
  :ensure t
  :mode (("\\.tsx\\'" . web-mode)
         ("\\.jsx\\'" . web-mode))
  :custom
  (web-mode-content-types-alist '(("jsx" . "\\.tsx\\'")))
  (web-mode-markup-indent-offset 2)
  (web-mode-code-indent-offset 2)
  (web-mode-css-indent-offset 2))

;; Keep TSX/JSX tree-sitter entries ahead of web-mode's fallback entries.
(when (require 'treesit nil t)
  (imoogi-treesit-auto-mode-if-available "\\.tsx\\'" 'tsx-ts-mode 'tsx)
  (imoogi-treesit-auto-mode-if-available "\\.jsx\\'" 'tsx-ts-mode 'tsx))

(provide 'imoogi-languages)
;;; 18-languages.el ends here
