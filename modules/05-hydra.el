;;; hydra.el --- Hydra definitions -*- lexical-binding: t; -*-

(use-package hydra
  :ensure t)

;;; ace-window
(use-package ace-window
  :ensure t
  :bind ("M-o" . ace-window)
  :custom
  (aw-keys '(?a ?s ?d ?f ?g ?h ?j ?k ?l))
  (aw-scope 'frame))

;; 창 관리
(defhydra hydra-window (:hint nil :color amaranth)
  "
  _h_: ←  _l_: →  _j_: ↓  _k_: ↑    _s_: 수평분할  _v_: 수직분할
  _H_: 축소← _L_: 확대→ _J_: 확대↓ _K_: 축소↑    _d_: 삭제  _D_: 나머지삭제
  _b_: 버퍼전환  _f_: 파일열기  _a_: ace-window  _m_: 스왑  _q_: 종료
  "
  ("h" windmove-left)
  ("l" windmove-right)
  ("j" windmove-down)
  ("k" windmove-up)
  ("H" shrink-window-horizontally)
  ("L" enlarge-window-horizontally)
  ("J" enlarge-window)
  ("K" shrink-window)
  ("s" split-window-below)
  ("v" split-window-right)
  ("d" delete-window)
  ("D" delete-other-windows :color blue)
  ("b" consult-buffer)
  ("f" find-file)
  ("a" ace-window)
  ("m" ace-swap-window)
  ("q" nil :color blue))

;; 프로젝트
(defhydra hydra-projectile (:hint nil :color blue)
  "
  _f_: 파일찾기  _s_: 검색(grep)  _b_: 버퍼  _d_: dired
  _p_: 프로젝트전환  _r_: 최근파일  _k_: 버퍼모두닫기
  _c_: 컴파일  _t_: 테스트  _q_: 종료
  "
  ("f" projectile-find-file)
  ("s" consult-ripgrep)
  ("b" consult-project-buffer)
  ("d" projectile-dired)
  ("p" projectile-switch-project)
  ("r" projectile-recentf)
  ("k" projectile-kill-buffers)
  ("c" projectile-compile-project)
  ("t" projectile-test-project)
  ("q" nil))

;; 텍스트 확대/축소
(defhydra hydra-zoom (:hint nil :color amaranth)
  "
  _i_: 확대  _o_: 축소  _0_: 초기화  _q_: 종료
  "
  ("i" text-scale-increase)
  ("o" text-scale-decrease)
  ("0" (text-scale-set 0) :color blue)
  ("q" nil :color blue))

;; Git (Magit)
(defhydra hydra-git (:hint nil :color blue)
  "
  _s_: status  _l_: log  _b_: blame  _d_: diff  _q_: 종료
  "
  ("s" magit-status)
  ("l" magit-log-current)
  ("b" magit-blame)
  ("d" magit-diff-dwim)
  ("q" nil))

;; 마스터 hydra (진입점)
(defhydra hydra-master (:hint nil :color blue)
  "
  _w_: 창관리  _p_: 프로젝트  _g_: Git  _z_: 확대/축소
  _t_: treemacs  _q_: 종료
  "
  ("w" hydra-window/body)
  ("p" hydra-projectile/body)
  ("g" hydra-git/body)
  ("z" hydra-zoom/body)
  ("t" treemacs)
  ("q" nil))

(global-set-key (kbd "C-c h") 'hydra-master/body)

(provide 'imoogi-hydra)
;;; hydra.el ends here
