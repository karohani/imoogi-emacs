;;; 16-languages.el --- 언어/파일타입 메이저 모드 (minimal-emacs.d 추천) -*- lexical-binding: t; -*-

;; 가벼운 메이저 모드 모음. 대부분 해당 확장자를 열 때만 로드된다(:mode).
;; Tree-sitter 기반(*-ts-mode)이 있으면 그쪽이 더 정확하지만, 여기서는
;; 별도 설치 없이 동작하는 전통 모드를 동봉한다.

;;; Code:

(imoogi-require "16-languages" 'git-modes 'yaml-mode 'dockerfile-mode 'gnuplot
                'lua-mode 'jinja2-mode 'csv-mode 'go-mode 'rust-mode 'crontab-mode
                'nginx-mode 'hcl-mode 'nix-mode 'fish-mode 'vimrc-mode 'jenkinsfile-mode)

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

(provide 'imoogi-languages)
;;; 16-languages.el ends here
