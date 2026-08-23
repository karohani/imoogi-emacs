;;; transient-menu-test.el --- transient 메뉴 구조/동작 테스트 -*- lexical-binding: t; -*-

;; SPEC-TRANSIENT-001 의 plan.md §C 매핑 표를 기계 검증한다.
;;
;; 두 층으로 나뉜다.
;;
;; 1. 구조 — `transient--make-predicate-map' 이 만든 keymap 을 조회해 각 suffix 가
;;    실제로 어떤 pre-command 로 디스패치되는지 본다. 선언된 `:transient' 속성을
;;    읽는 것보다 강하다: transient 자신의 기본값 해석까지 거친 "실행될 함수"다.
;;      transient--do-call     열린 채 유지 (hydra 의 amaranth head)
;;      transient--do-exit     실행 후 닫힘 (hydra 의 :color blue)
;;      transient--do-stack    하위 메뉴로 진입
;;      transient--do-quit-one 종료
;;
;; 2. 동작 — `execute-kbd-macro' 로 실제 키를 눌러 팝업이 유지/종료되는지
;;    `transient--prefix' 로 관찰한다. 배치 모드에서 동작한다(실증됨).
;;    프롬프트를 띄우는 명령(project-find-file, magit-status 등)은 배치에서
;;    멈추므로 동작 층에서 제외하고 구조 층으로만 검증한다.
;;
;; @MX:NOTE transient--make-predicate-map / transient--prefix / transient--stack-zap
;; 은 transient 내부 API(double-dash)다. transient 업그레이드 시 깨질 수 있으며,
;; 그때는 transient-get-suffix 의 :transient 속성을 읽는 방식으로 낮춰 잡는다
;; (더 약한 검증이지만 공개 API).

;;; Code:

(require 'ert)
(require 'transient)

(defun imoogi-test--predicate-map (prefix)
  "PREFIX 를 setup 한 뒤 pre-command 디스패치 keymap 을 돌려준다."
  (unwind-protect
      (progn (transient-setup prefix)
             (transient--make-predicate-map))
    (transient--stack-zap)))

(defun imoogi-test--dispatch (prefix command)
  "PREFIX 안에서 COMMAND 가 디스패치될 pre-command 를 돌려준다."
  (lookup-key (imoogi-test--predicate-map prefix) (vector command)))

;;; 1. 구조 — plan.md §C 40행

(defconst imoogi-test--suffix-table
  '((imoogi-transient-window                      ; §C.1 (17)
     (windmove-left                    transient--do-call)
     (windmove-right                   transient--do-call)
     (windmove-down                    transient--do-call)
     (windmove-up                      transient--do-call)
     (shrink-window-horizontally       transient--do-call)
     (enlarge-window-horizontally      transient--do-call)
     (enlarge-window                   transient--do-call)
     (shrink-window                    transient--do-call)
     (split-window-below               transient--do-call)
     (split-window-right               transient--do-call)
     (delete-window                    transient--do-call)
     (delete-other-windows             transient--do-exit)
     (imoogi-consult-perspective-buffer transient--do-call)
     (find-file                        transient--do-call)
     (ace-window                       transient--do-call)
     (ace-swap-window                  transient--do-call)
     (transient-quit-one               transient--do-quit-one))
    ;; §C.2 확장 — 원래 8개(열기+관리)에 "작업공간 전환" 그룹을 더했다.
    ;; persp-next/prev 만 훑어보기용이라 열린 채 유지된다.
    (imoogi-transient-project
     (imoogi-project-switch-perspective transient--do-exit)
     (project-find-file                transient--do-exit)
     (project-find-regexp              transient--do-exit)
     (project-dired                    transient--do-exit)
     (project-switch-to-buffer         transient--do-exit)
     (persp-switch-last                transient--do-exit)
     (persp-switch                     transient--do-exit)
     (persp-next                       transient--do-call)
     (persp-prev                       transient--do-call)
     (persp-switch-by-number           transient--do-exit)
     (project-compile                  transient--do-exit)
     (project-kill-buffers             transient--do-exit)
     (persp-rename                     transient--do-exit)
     (persp-kill                       transient--do-exit)
     (project-forget-project           transient--do-exit)
     (transient-quit-one               transient--do-quit-one))
    (imoogi-transient-zoom                        ; §C.3 (4)
     (text-scale-increase              transient--do-call)
     (text-scale-decrease              transient--do-call)
     (imoogi-transient-zoom-reset      transient--do-exit)
     (transient-quit-one               transient--do-quit-one))
    (imoogi-transient-git                         ; §C.4 (5)
     (magit-status                     transient--do-exit)
     (magit-log-current                transient--do-exit)
     ;; magit-blame 은 그 자체가 transient prefix 라서 do-exit 이 될 수 없다.
     ;; transient 는 prefix 인 suffix 를 항상 stack/recurse 로 디스패치한다
     ;; (transient.el `transient--make-predicate-map', prefix/nil → do-stack).
     ;; hydra :color blue 와의 유일한 동작 차이 — ARCHITECTURE.md 참조.
     (magit-blame                      transient--do-stack)
     (magit-diff-dwim                  transient--do-exit)
     (transient-quit-one               transient--do-quit-one))
    (imoogi-transient-master                      ; §C.5 (6)
     (imoogi-transient-window          transient--do-stack)
     (imoogi-transient-project         transient--do-stack)
     (imoogi-transient-git             transient--do-stack)
     (imoogi-transient-zoom            transient--do-stack)
     (imoogi-transient-code            transient--do-stack)
     (imoogi-transient-tab             transient--do-stack)
     (imoogi-treemacs-toggle-file-tree transient--do-exit)
     (imoogi-reload                    transient--do-exit)
     (transient-quit-one               transient--do-quit-one)))
  "plan.md §C 의 (prefix (command expected-pre-command)...) 전개.")

(ert-deftest imoogi-transient-suffix-dispatch-matches-spec-table ()
  "모든 suffix 가 §C 표대로 유지/종료 디스패치되는지 확인한다."
  (dolist (entry imoogi-test--suffix-table)
    (let* ((prefix (car entry))
           (map (imoogi-test--predicate-map prefix)))
      (dolist (row (cdr entry))
        (should (eq (lookup-key map (vector (nth 0 row)))
                    (nth 1 row)))))))

(ert-deftest imoogi-transient-every-prefix-binds-q-explicitly ()
  "5개 prefix 모두 명시적 q suffix 를 갖는다 (AC-008)."
  (dolist (prefix '(imoogi-transient-window imoogi-transient-project
                    imoogi-transient-zoom imoogi-transient-git
                    imoogi-transient-master))
    (should (eq (plist-get (cdr (transient-get-suffix prefix "q")) :command)
                #'transient-quit-one))))

(ert-deftest imoogi-transient-master-is-bound-at-C-c-h ()
  (should (eq (lookup-key global-map (kbd "C-c h"))
              #'imoogi-transient-master)))

(defun imoogi-test--layout-bindings (prefix)
  "PREFIX 레이아웃이 선언한 (키 . 명령) 쌍 목록(키 기준 정렬).

키만 모으면 충돌을 놓친다: `transient-append-suffix' 가 이미 있는 키를 만나면
키 집합은 그대로 둔 채 명령만 조용히 교체하기 때문이다(실측). 명령까지 함께
단언해야 덮어쓰기가 드러난다."
  (let (pairs)
    (letrec ((walk (lambda (node)
                     (cond
                      ((vectorp node) (mapc walk (append node nil)))
                      ((listp node)
                       (let* ((pl (cdr-safe node))
                              (k (plist-get pl :key)))
                         (if k
                             (push (cons k (plist-get pl :command)) pairs)
                           (mapc walk (seq-filter #'sequencep node)))))))))
      (funcall walk (get prefix 'transient--layout)))
    (sort pairs (lambda (a b) (string< (car a) (car b))))))

;; 모듈이 `transient-append-suffix' 로 마스터에 항목을 등록하는 구조라, 두 모듈이
;; 같은 키를 쓰면 나중 등록이 조용히 이긴다. 키와 명령을 함께 단언해 그 덮어쓰기를
;; 여기서 잡는다. 새 항목을 의도적으로 추가할 때는 이 기대값도 함께 갱신한다.
(ert-deftest imoogi-transient-master-bindings-have-no-collisions ()
  (should (equal (imoogi-test--layout-bindings 'imoogi-transient-master)
                 '(("R" . imoogi-reload)
                   ("T" . imoogi-transient-tab)
                   ("c" . imoogi-transient-code)
                   ("g" . imoogi-transient-git)
                   ("l" . imoogi-transient-lsp)
                   ("p" . imoogi-transient-project)
                   ("q" . transient-quit-one)
                   ("t" . imoogi-treemacs-toggle-file-tree)
                   ("w" . imoogi-transient-window)
                   ("z" . imoogi-transient-zoom)))))

;;; 모듈이 스스로 등록한 LSP 메뉴 (modules/17-lsp.el)

(ert-deftest imoogi-transient-lsp-is-registered-on-master ()
  (should (eq (plist-get (cdr (transient-get-suffix 'imoogi-transient-master "l"))
                         :command)
              #'imoogi-transient-lsp))
  (should (eq (imoogi-test--dispatch 'imoogi-transient-master 'imoogi-transient-lsp)
              #'transient--do-stack)))

(ert-deftest imoogi-transient-lsp-bindings-match-prefix-map ()
  "메뉴 항목이 C-c l 접두 맵과 같은 명령을 가리킨다.
q 만 예외 — 접두 맵에서는 eglot-shutdown 이지만 transient 에서 q 는 종료
관례라 K 로 옮겼다."
  (should (equal (imoogi-test--layout-bindings 'imoogi-transient-lsp)
                 '(("K" . eglot-shutdown)
                   ("R" . eglot-rename)
                   ("a" . xref-find-apropos)
                   ("b" . xref-go-back)
                   ("c" . eglot-code-actions)
                   ("d" . xref-find-definitions)
                   ("e" . eglot)
                   ("f" . xref-go-forward)
                   ("o" . eglot-code-action-organize-imports)
                   ("q" . transient-quit-one)
                   ("r" . xref-find-references))))
  ;; 접두 맵과 어긋나지 않는지 교차 확인. K/q 는 아래에서 따로 단언한다.
  (dolist (pair (imoogi-test--layout-bindings 'imoogi-transient-lsp))
    (unless (member (car pair) '("q" "K"))
      (should (eq (lookup-key imoogi-lsp-map (kbd (car pair)))
                  (cdr pair)))))
  ;; 의도된 유일한 차이: 접두 맵의 q(서버 종료)가 메뉴에서는 K 로 옮겨졌고,
  ;; 메뉴의 q 는 종료다. 접두 맵 쪽은 손대지 않았음을 함께 고정한다.
  (should (eq (lookup-key imoogi-lsp-map (kbd "q")) #'eglot-shutdown))
  (should-not (lookup-key imoogi-lsp-map (kbd "K"))))

(defun imoogi-test--inapt-keys (prefix)
  (unwind-protect
      (progn (transient-setup prefix)
             (let (keys)
               (dolist (obj transient--suffixes)
                 (when (and (slot-boundp obj 'key) (oref obj inapt))
                   (push (oref obj key) keys)))
               (sort keys #'string<)))
    (transient--stack-zap)))

(ert-deftest imoogi-transient-lsp-greys-out-eglot-commands-when-inactive ()
  "Eglot 이 안 붙은 버퍼에서는 서버 의존 명령만 흐려진다(숨기지 않는다)."
  (with-temp-buffer
    (should (equal (imoogi-test--inapt-keys 'imoogi-transient-lsp)
                   '("K" "R" "c" "o")))
    (cl-letf (((symbol-function 'eglot-managed-p) (lambda (&rest _) t)))
      (should-not (imoogi-test--inapt-keys 'imoogi-transient-lsp)))))

;;; 2. 동작 — 실제 키 입력

(defun imoogi-test--active-prefix ()
  (and transient--prefix (oref transient--prefix command)))

(ert-deftest imoogi-transient-zoom-stays-open-then-exits-on-reset ()
  "i 는 팝업을 유지하고 0 은 배율을 되돌리며 닫는다 (AC-005, AC-006)."
  (with-temp-buffer
    (unwind-protect
        (progn
          (transient-setup 'imoogi-transient-zoom)
          (execute-kbd-macro (kbd "i"))
          (should (eq (imoogi-test--active-prefix) 'imoogi-transient-zoom))
          (should (= text-scale-mode-amount 1))
          (execute-kbd-macro (kbd "i"))
          (should (eq (imoogi-test--active-prefix) 'imoogi-transient-zoom))
          (should (= text-scale-mode-amount 2))
          (execute-kbd-macro (kbd "0"))
          (should-not (imoogi-test--active-prefix))
          (should (= text-scale-mode-amount 0)))
      (transient--stack-zap))))

(ert-deftest imoogi-transient-window-stays-open-then-exits-on-delete-others ()
  "h 는 팝업을 유지하고 D 는 다른 창을 지우며 닫는다 (AC-003, AC-004)."
  (let ((config (current-window-configuration)))
    (unwind-protect
        (progn
          (delete-other-windows)
          (split-window-right)
          (other-window 1)
          (transient-setup 'imoogi-transient-window)
          (execute-kbd-macro (kbd "h"))
          (should (eq (imoogi-test--active-prefix) 'imoogi-transient-window))
          (should (> (length (window-list)) 1))
          (execute-kbd-macro (kbd "D"))
          (should-not (imoogi-test--active-prefix))
          (should (= (length (window-list)) 1)))
      (transient--stack-zap)
      (set-window-configuration config))))

(ert-deftest imoogi-transient-master-opens-submenu-and-q-closes-it ()
  "마스터에서 z 는 하위 메뉴로 진입하고, q 는 그 메뉴를 닫는다 (AC-007)."
  (with-temp-buffer
    (unwind-protect
        (progn
          (transient-setup 'imoogi-transient-master)
          (execute-kbd-macro (kbd "z"))
          (should (eq (imoogi-test--active-prefix) 'imoogi-transient-zoom))
          (execute-kbd-macro (kbd "q"))
          (should-not (eq (imoogi-test--active-prefix) 'imoogi-transient-zoom)))
      (transient--stack-zap))))

;;; 설정 재로드 (C-c h R)

(ert-deftest imoogi-reload-does-not-accumulate-failed-modules ()
  "재로드를 반복해도 실패 모듈 목록이 누적되지 않는다.

`boot.el' 은 재로드될 수 있으므로 목록을 매 로드마다 비워야 한다. 비우지 않으면
같은 모듈 이름이 두 번 세 번 쌓여, 이 변수를 판정 근거로 쓰는
tests/assert-boot.el 이 잘못 읽는다(실제로 발생했던 버그).

건강한 트리에서는 실패 모듈이 없어 누적이 드러나지 않으므로, 실패를 일부러
주입한다 — 주입 없이 before/after 만 비교하면 초기화를 빼도 통과하는
공허한 테스트가 된다(확인함)."
  (let ((orig (symbol-function 'locate-library)))
    (unwind-protect
        (cl-letf (((symbol-function 'locate-library)
                   (lambda (lib &rest args)
                     (unless (equal lib "magit") (apply orig lib args)))))
          (imoogi-reload)
          (let ((first-run (copy-sequence imoogi-failed-modules)))
            ;; 주입이 실제로 먹었는지 먼저 확인(안 먹으면 아래 비교가 공허해진다)
            (should (member "06-git" first-run))
            (imoogi-reload)
            (should (equal imoogi-failed-modules first-run))))
      ;; 다른 테스트에 영향이 가지 않도록 정상 상태로 되돌린다.
      (imoogi-reload))))

(ert-deftest imoogi-reload-survives-menu-registration ()
  "재로드 후에도 마스터 메뉴와 전역 키가 그대로다.
모듈이 transient-append-suffix 로 등록하는 구조라, 재로드가 항목을 중복시키거나
날려버리지 않는지 확인한다."
  (let ((before (imoogi-test--layout-bindings 'imoogi-transient-master)))
    (imoogi-reload)
    (should (equal (imoogi-test--layout-bindings 'imoogi-transient-master) before))
    (should (eq (lookup-key global-map (kbd "C-c h")) #'imoogi-transient-master))))

;;; 코드 이해 5단계 메뉴 (C-c h c)

(ert-deftest imoogi-transient-code-bindings-match-the-five-levels ()
  (should (equal (imoogi-test--layout-bindings 'imoogi-transient-code)
                 '(("?" . imoogi-code-capability-report)
                   ("I" . consult-imenu-multi)
                   ("d" . xref-find-definitions)
                   ("f" . imoogi-treemacs-toggle-file-tree)
                   ("g" . imoogi-code-capability-report)
                   ("i" . consult-imenu)
                   ("m" . eglot-find-implementation)
                   ("q" . transient-quit-one)
                   ("r" . xref-find-references)
                   ("s" . imoogi-treemacs-toggle-structure)
                   ("t" . treesit-explore-mode)
                   ("y" . eglot-find-typeDefinition)))))

(ert-deftest imoogi-transient-code-greys-out-unavailable-levels ()
  "갖춰지지 않은 단계만 흐려지고, 각 단계는 독립적으로 켜진다."
  (with-temp-buffer
    ;; 일반 버퍼: 파서 없음 + Eglot 없음 + 그래프 미구현
    (should (equal (imoogi-test--inapt-keys 'imoogi-transient-code)
                   '("g" "m" "t" "y")))
    ;; L3 만 갖춰지면 t 만 켜진다 (다른 단계는 그대로 흐림)
    (cl-letf (((symbol-function 'imoogi-code--syntax-available-p) (lambda () t)))
      (should (equal (imoogi-test--inapt-keys 'imoogi-transient-code)
                     '("g" "m" "y"))))
    ;; L4 만 갖춰지면 m/y 만 켜진다
    (cl-letf (((symbol-function 'imoogi-code--relations-available-p) (lambda () t)))
      (should (equal (imoogi-test--inapt-keys 'imoogi-transient-code)
                     '("g" "t"))))))

(ert-deftest imoogi-code-capability-report-names-every-level ()
  "보고서가 5단계를 모두 짚고, 안 되는 항목에는 이유가 붙는다."
  (with-temp-buffer
    (imoogi-code-capability-report))
  (with-current-buffer "*imoogi 코드 이해 가용성*"
    (let ((text (buffer-substring-no-properties (point-min) (point-max))))
      (dolist (level '("L1" "L2" "L3" "L4" "L5"))
        (should (string-match-p level text)))
      ;; 미지원 항목은 이유 없이 나열되면 안 된다
      (should (string-match-p "hierarchy" text))
      (should (string-match-p "vendor/tree-sitter/" text)))))

;;; 탭 메뉴 (tmux window 층, modules/22-tabs.el)

(ert-deftest imoogi-tab-prefix-map-is-bound-outside-treemacs-territory ()
  "Emacs 표준 C-x t 는 treemacs 가 이미 쓰므로 C-c w 를 쓴다."
  (should (keymapp (lookup-key global-map (kbd "C-c w"))))
  (should (eq (lookup-key imoogi-tab-map (kbd "c")) #'tab-bar-new-tab))
  ;; treemacs 자리를 빼앗지 않았는지 함께 고정한다.
  (should (eq (lookup-key global-map (kbd "C-x t t"))
              #'imoogi-treemacs-toggle-file-tree)))

(ert-deftest imoogi-tab-quick-switch-keys-are-bound ()
  (should (eq (key-binding (kbd "s-[")) #'tab-bar-switch-to-prev-tab))
  (should (eq (key-binding (kbd "s-]")) #'tab-bar-switch-to-next-tab)))

(ert-deftest imoogi-transient-tab-bindings-are-stable ()
  (should (equal (imoogi-test--layout-bindings 'imoogi-transient-tab)
                 '(("c" . tab-bar-new-tab)
                   ("k" . tab-bar-close-tab)
                   ("l" . tab-bar-switch-to-recent-tab)
                   ("n" . tab-bar-switch-to-next-tab)
                   ("p" . tab-bar-switch-to-prev-tab)
                   ("q" . transient-quit-one)
                   ("r" . tab-bar-rename-tab)
                   ("s" . tab-bar-switch-to-tab)
                   ("t" . tab-bar-mode)
                   ("u" . tab-bar-undo-close-tab)))))

(provide 'transient-menu-test)
;;; transient-menu-test.el ends here
