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

;; [HARD] 런타임 문법 다운로드 차단.
;; clojure-ts-mode 는 `clojure-ts-ensure-grammars' 가 기본 t 라, 문법이 없거나
;; 자기가 못박은 리비전과 다르면 모드 진입 시 GitHub 에서 내려받아 컴파일한다
;; (실측: 부팅 중 clojure/regex/markdown-inline 3개가 ~/.emacs.d/tree-sitter/ 에
;; 설치됨). 망분리 원칙("부팅 경로에서 네트워크 접근 없음") 위반이므로 끈다.
;; 필요한 문법은 vendor/tree-sitter/ 에 정확한 리비전으로 동봉한다
;; (scripts/build-grammars.sh 의 clojure/regex/markdown-inline 항목).
(setq clojure-ts-ensure-grammars nil)

(when (require 'treesit nil t)
  (add-to-list 'treesit-extra-load-path imoogi-treesit-grammar-dir)
  (imoogi-treesit-remap-if-available 'sh-mode 'bash-ts-mode 'bash)
  (imoogi-treesit-remap-if-available 'js-mode 'js-ts-mode 'javascript)
  (imoogi-treesit-remap-if-available 'typescript-mode 'typescript-ts-mode 'typescript)
  (imoogi-treesit-auto-mode-if-available "\\.tsx\\'" 'tsx-ts-mode 'tsx)
  (imoogi-treesit-auto-mode-if-available "\\.jsx\\'" 'tsx-ts-mode 'tsx)
  (imoogi-treesit-remap-if-available 'js-json-mode 'json-ts-mode 'json)
  (imoogi-treesit-remap-if-available 'python-mode 'python-ts-mode 'python)
  (imoogi-treesit-remap-if-available 'go-mode 'go-ts-mode 'go)
  (imoogi-treesit-remap-if-available 'rust-mode 'rust-ts-mode 'rust)
  (imoogi-treesit-remap-if-available 'java-mode 'java-ts-mode 'java)
  (imoogi-treesit-remap-if-available 'yaml-mode 'yaml-ts-mode 'yaml)
  (imoogi-treesit-remap-if-available 'dockerfile-mode 'dockerfile-ts-mode 'dockerfile)
  (imoogi-treesit-remap-if-available 'html-mode 'html-ts-mode 'html)
  ;; 아래 둘은 Emacs 내장 ts-mode 가 없어 패키지가 제공한다. 패키지와 문법이
  ;; 모두 있어야 전환된다 — 헬퍼가 `fboundp' 로 모드 존재까지 확인한다.
  (imoogi-treesit-remap-if-available 'kotlin-mode 'kotlin-ts-mode 'kotlin)
  (imoogi-treesit-remap-if-available 'clojure-mode 'clojure-ts-mode 'clojure))

;;; JSON — imenu 로 키를 훑을 수 있게
;; js-json-mode 는 js-mode 에서 파생돼 JavaScript 용 인덱서
;; (`js--imenu-create-index')를 물려받는다. 그건 함수 선언·할당 같은 "코드"를
;; 찾으므로 순수 데이터인 JSON 에서는 늘 빈손이고, imenu 는
;; "No items suitable for an index found" 로 끝난다(실측).
;;
;; 그래서 인덱서를 기본 구현으로 되돌리고 키를 잡는 패턴을 준다. 둘 다 해야
;; 한다 — `imenu-generic-expression' 만 설정하면 여전히 js 인덱서가 불린다.
;;
;; tree-sitter json 문법을 반입하면 json-ts-mode 가 자체 imenu 설정
;; (treesit-simple-imenu-settings)을 갖고 있어 이 보정이 필요 없어진다.
(defun imoogi-json-setup-imenu ()
  "JSON 버퍼에서 키를 imenu 항목으로 잡는다."
  (setq-local imenu-create-index-function #'imenu-default-create-index-function)
  (setq-local imenu-generic-expression
              '((nil "^[ \t]*\"\\([^\"]+\\)\"[ \t]*:" 1))))

(add-hook 'js-json-mode-hook #'imoogi-json-setup-imenu)

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
