;;; 17-lsp.el --- Eglot LSP 및 xref 공통 설정 -*- lexical-binding: t; no-native-compile: t; -*-

;; [HARD] no-native-compile: Emacs 31.1 의 네이티브 컴파일러가 이 파일의 아래쪽
;; `with-eval-after-load 'imoogi-transient' + transient-define-prefix' 조합을
;; 깨뜨린다. .eln 이 로드되면 모듈 전체가
;;   Symbol's value as variable is void: imoogi-transient-lsp
;; 로 실패하고, imoogi-require 의 degradation 때문에 이 모듈만 조용히 빠진다
;; — 증상이 "31 에서 LSP 가 안 켜진다" 로 나타나 원인을 찾기 어렵다.
;;
;; 바이트 컴파일(.elc)은 멀쩡하다. 실측으로 분리했다: .eln 두 개만 지우면
;; .elc 를 그대로 둔 채로 정상 부팅한다. 30.2 는 네이티브 컴파일이 없어 무관하며,
;; 이 파일은 설정 배선이라 네이티브 컴파일로 얻는 성능도 없다.

;; 공통 LSP 수명주기와 xref 진입점만 관리한다. 언어별 서버 연결과
;; workspace 설정은 modules/lsp/*.el 에서 독립적으로 로드한다.

;;; Code:

(imoogi-require "17-lsp" 'eglot 'flymake 'xref 'seq)

(defvar eglot-server-programs)

(defvar imoogi-lsp-language-config-dir
  (expand-file-name "modules/lsp/" imoogi-emacs-dir)
  "Directory containing language-specific LSP configurations.")

(defun imoogi-lsp-local-bin-dir ()
  "Return active repository-local toolchain bin directory, or nil.

The activation path must be the relative .local/bin symlink created by the
toolchain setup command.  Missing, broken, absolute, or non-toolchain links are
ignored so Emacs startup stays side-effect free."
  (let* ((local-root (expand-file-name ".local/" imoogi-emacs-dir))
         (link (expand-file-name "bin" local-root))
         (target (file-symlink-p link)))
    (when (and target
               (not (file-name-absolute-p target))
               (string-prefix-p "toolchains/" target)
               (file-directory-p link))
      (let ((toolchains-root (file-truename
                              (expand-file-name "toolchains/" local-root)))
            (bin-root (file-truename link)))
        (when (file-in-directory-p bin-root toolchains-root)
          (file-name-as-directory link))))))

(defun imoogi-lsp-prepend-local-bin ()
  "Prepend active repository-local toolchain bin to process lookup paths."
  (when-let ((local-bin (imoogi-lsp-local-bin-dir)))
    (setq exec-path
          (cons local-bin
                (seq-remove (lambda (entry)
                              (imoogi-lsp-same-directory-p entry local-bin))
                            exec-path)))
    (let* ((current-path (getenv "PATH"))
           (entries (if current-path
                        (split-string current-path path-separator t)
                      nil)))
      (setenv "PATH"
              (mapconcat #'identity
                         (cons local-bin
                               (seq-remove
                                (lambda (entry)
                                  (imoogi-lsp-same-directory-p entry local-bin))
                                entries))
                         path-separator)))))

(defun imoogi-lsp-same-directory-p (a b)
  "Return non-nil when A and B name the same directory string."
  (and (stringp a)
       (stringp b)
       (string= (directory-file-name a) (directory-file-name b))))

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

(defun imoogi-eglot-ensure-if-any-server-available (&rest servers)
  "Start Eglot when any command in SERVERS is available locally."
  (when (and (seq-some #'imoogi-eglot-server-available-p servers)
             (require 'eglot nil t))
    (eglot-ensure)))

(use-package eglot
  :ensure nil
  :commands (eglot eglot-ensure eglot-rename eglot-code-actions
                   eglot-code-action-organize-imports eglot-shutdown)
  :custom
  (eglot-autoshutdown t)
  (eglot-sync-connect 0)
  (eglot-extend-to-xref t)
  (eglot-events-buffer-config '(:size 0 :format short)))

(use-package flymake
  :ensure nil
  :custom
  (flymake-show-diagnostics-at-end-of-line nil)
  (flymake-wrap-around nil))

(defvar-keymap imoogi-lsp-map
  :doc "LSP와 xref 코드 탐색 명령의 공통 접두 맵."
  "d" #'xref-find-definitions
  "r" #'xref-find-references
  "a" #'xref-find-apropos
  "b" #'xref-go-back
  "f" #'xref-go-forward
  "e" #'eglot
  "R" #'eglot-rename
  "c" #'eglot-code-actions
  "o" #'eglot-code-action-organize-imports
  "q" #'eglot-shutdown)

(use-package xref
  :ensure nil
  :config
  (global-set-key (kbd "C-c l") imoogi-lsp-map))

(with-eval-after-load 'which-key
  (which-key-add-key-based-replacements "C-c l" "LSP/xref"))

(defun imoogi-lsp-eglot-active-p ()
  "현재 버퍼가 Eglot 관리 하에 있으면 non-nil.
Eglot 이 아직 로드되지 않았을 수도 있으므로 `fboundp' 로 먼저 막는다."
  (and (fboundp 'eglot-managed-p) (eglot-managed-p)))

;; transient 메뉴는 05-transient 가 로드된 경우에만 만든다. 이 모듈은 transient
;; 를 사전조건(`imoogi-require')으로 요구하지 않으므로, 05 가 건너뛰어졌으면
;; 메뉴만 없고 위 `imoogi-lsp-map'(C-c l)은 그대로 동작한다.
;; @MX:NOTE 마스터 메뉴에 자기 항목을 스스로 등록하는 방식(transient-append-suffix).
;; 05-transient 가 이 모듈을 알 필요가 없고, 이 모듈이 로드 실패하면 항목도 함께
;; 사라진다 — imoogi-require 의 degradation 모델과 같은 결. 등록 키가 다른 모듈과
;; 겹치면 나중 등록이 조용히 이기므로, tests/transient-menu-test.el 이 마스터의
;; 키 집합을 단언해 충돌을 잡는다.
(with-eval-after-load 'imoogi-transient
  (transient-define-prefix imoogi-transient-lsp ()
    "LSP·코드 탐색 메뉴."
    :column-widths '(16 18)
    [["탐색 (xref) --"
      ("d" "정의로" xref-find-definitions)
      ("r" "참조 찾기" xref-find-references)
      ("a" "apropos" xref-find-apropos)
      ("b" "뒤로" xref-go-back :transient t)
      ("f" "앞으로" xref-go-forward :transient t)]
     ["Eglot ---------"
      ("e" "연결" eglot)
      ;; 아래 넷은 Eglot 이 붙은 버퍼에서만 의미가 있다. 숨기지 않고 흐리게
      ;; 표시해(:inapt-*) 기능의 존재는 알리되 지금은 못 쓴다는 걸 보여준다.
      ("R" "이름 변경" eglot-rename :inapt-if-not imoogi-lsp-eglot-active-p)
      ("c" "코드 액션" eglot-code-actions :inapt-if-not imoogi-lsp-eglot-active-p)
      ("o" "임포트 정리" eglot-code-action-organize-imports
       :inapt-if-not imoogi-lsp-eglot-active-p)
      ;; 접두 맵에서는 C-c l q 지만, transient 에서 q 는 종료 관례라 K 로 옮겼다.
      ("K" "서버 종료" eglot-shutdown :inapt-if-not imoogi-lsp-eglot-active-p)
      ("q" "종료" transient-quit-one)]])

  (transient-append-suffix 'imoogi-transient-master "t"
    '("l" "LSP" imoogi-transient-lsp)))

(imoogi-lsp-prepend-local-bin)

(defun imoogi-lsp-load-language-configurations ()
  "Load every language-specific LSP configuration with failure isolation."
  (when (file-directory-p imoogi-lsp-language-config-dir)
    (dolist (file (directory-files imoogi-lsp-language-config-dir t
                                   "\\`[[:alnum:]-]+\\.el\\'"))
      (condition-case err
          (load (file-name-sans-extension file))
        (error
         (display-warning
          'imoogi-lsp
          (format "LSP 언어 설정 %s 로딩 실패(건너뜀): %s"
                  (file-name-base file) (error-message-string err))
          :error))))))

(imoogi-lsp-load-language-configurations)

(provide 'imoogi-lsp)
;;; 17-lsp.el ends here
