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

;;; Tree-sitter 색칠 수준 — IntelliJ/VSCode 식 의미 기반 하이라이트
;; 기본값 3 은 정의·키워드·타입·상수까지만 칠한다. 4 를 켜야 아래가 붙는다
;; (go-ts-mode 의 `treesit-font-lock-feature-list' 실측):
;;   bracket delimiter error function operator property variable
;; 즉 함수 "호출", 변수 "사용처", 프로퍼티가 여기서 처음 색을 갖는다.
;; doom-themes 가 이 확장 face 를 모두 정의하므로(doom-themes-base.el —
;; font-lock-function-call-face 는 이탤릭 + 흐린 색으로 정의를 호출과 구분한다)
;; 레벨만 올리면 바로 보인다.
;;
;; [HARD] setq 가 아니라 setopt 여야 한다. 이 옵션에는
;; `:set treesit--font-lock-level-setter' 가 걸려 있어(30.2 실측), setq 로 바꾸면
;; 값만 바뀌고 `treesit-font-lock-recompute-features' 가 불리지 않아 무효다.
;;
;; 값의 형태가 버전마다 다르다 — Emacs 30 의 :type 은 `integer' 뿐이고(실측),
;; (MAJOR-MODE . LEVEL) alist 는 Emacs 31 부터다. 30 에 alist 를 주면
;; "does not match type integer" 경고가 뜬다. 그래서 버전으로 갈라 쓴다.
;; 31 에서는 YAML/JSON/HTML 같은 데이터 포맷만 3 으로 남긴다 — 이런 파일에서
;; 4 는 구분자·괄호까지 전부 칠해 오히려 읽기 어려워진다.
(when (boundp 'treesit-font-lock-level)
  (setopt treesit-font-lock-level
          (if (>= emacs-major-version 31)
              '((yaml-ts-mode . 3)
                (json-ts-mode . 3)
                (html-ts-mode . 3)
                (t            . 4))
            4)))

;; [HARD] 문법 자동 다운로드 차단 (Emacs 31+ 에서만 존재하는 옵션).
;; 31 에 없는 문법을 자동으로 내려받아 빌드하는 기능이 생겼고, 기본값이 `ask'
;; 다(31.1 실측). 망분리 원칙("부팅 경로에서 네트워크 접근 없음")상 꺼야 한다.
;;
;; 값은 반드시 `never' 여야 한다. :type 이 열거형이라(31.1 실측)
;;   (choice (const never) (const always) (const ask) (const ask-dir))
;; nil 은 허용 값이 아니고, 주면 setopt 가 타입 경고를 낸다.
;;
;; 지금은 구조적으로도 이미 막혀 있다 — `imoogi-treesit-remap-if-available' 가
;; `treesit-language-available-p' 로 먼저 거르므로, 문법이 없는 언어는 ts-mode
;; 진입 자체를 안 하고 따라서 자동 설치가 발동할 계기가 없다. 다만
;; `treesit-enabled-modes' 로 모드를 직접 켜면 그 보호가 사라지므로, 이 설정이
;; 그때의 방어선이 된다.
(when (boundp 'treesit-auto-install-grammar)
  (setopt treesit-auto-install-grammar 'never))

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
