;;; 31-agent.el --- imoogi-agent-* interface over agent-shell -*- lexical-binding: t; -*-

;; Coding agents (Claude Code, Codex, Gemini) run inside Emacs through
;; agent-shell, which talks the Agent Client Protocol (ACP) to each agent's
;; adapter process. agent-shell is pinned (packages.el `imoogi-pinned-packages')
;; but moves fast, so the rest of this config only calls the imoogi-agent-*
;; commands below. When an upgrade renames or drops a function, only this file
;; changes: `imoogi-agent--upstream-api' lists every upstream function used, and
;; both the first command and tests/agent-test.el name any that are missing.
;;
;; The Claude Code and Codex adapters (claude-agent-acp, codex-acp) are
;; toolchain components: `make toolchain-setup' installs them into .local/bin
;; on the bundled node, and M-x imoogi-doctor checks that they answer.
;;
;; agent-shell is large, so it loads on first use rather than at boot.
;; The opposite direction (an agent notifying Emacs) is 30-emacsctl.el.

;;; Code:

(imoogi-require "31-agent" 'agent-shell)

(eval-when-compile (require 'transient))

(defgroup imoogi-agent nil
  "imoogi interface to agent-shell."
  :group 'tools)

(defconst imoogi-agent--features
  '(agent-shell agent-shell-anthropic agent-shell-openai agent-shell-google)
  "agent-shell libraries the wrapper loads on first use.")

(defvar imoogi-agent--upstream-api
  '(agent-shell agent-shell-toggle agent-shell-new-shell agent-shell-switch-buffer
    agent-shell-send-region-to agent-shell-send-file agent-shell-send-other-file
    agent-shell-interrupt agent-shell-resume-session
    agent-shell-set-session-model agent-shell-set-session-mode
    agent-shell-copy-last-output agent-shell-copy-session-id agent-shell-view-acp-logs
    agent-shell-anthropic-start-claude-code agent-shell-openai-start-codex
    agent-shell-google-start-gemini)
  "Every agent-shell function the imoogi-agent-* commands call.
Checked against the loaded agent-shell before the first call.")

(defconst imoogi-agent-backends
  '((claude :label "Claude Code" :start agent-shell-anthropic-start-claude-code
            :command agent-shell-anthropic-claude-acp-command)
    (codex :label "Codex" :start agent-shell-openai-start-codex
           :command agent-shell-openai-codex-acp-command)
    (gemini :label "Gemini" :start agent-shell-google-start-gemini
            :command agent-shell-google-gemini-acp-command))
  "Agents `imoogi-agent-start' can launch.
:start is the agent-shell command, :command the variable holding the ACP
adapter command line whose program must be on PATH.")

(defcustom imoogi-agent-default-backend 'claude
  "Agent `imoogi-agent-start' launches without a prefix argument."
  :type '(choice (const claude) (const codex) (const gemini)))

(defun imoogi-agent-missing-api ()
  "Load agent-shell and return the wrapped functions it no longer defines."
  (dolist (feature imoogi-agent--features)
    (require feature nil t))
  (seq-remove #'fboundp imoogi-agent--upstream-api))

(defun imoogi-agent--ensure ()
  "Load agent-shell, or fail naming the functions an upgrade removed."
  (when-let* ((missing (imoogi-agent-missing-api)))
    (user-error "agent-shell %s no longer defines %s; update modules/development/31-agent.el"
                (if (boundp 'agent-shell--version) agent-shell--version "(unknown version)")
                (mapconcat #'symbol-name missing ", "))))

(defun imoogi-agent--run (command)
  "Run agent-shell COMMAND interactively after checking the wrapped API."
  (imoogi-agent--ensure)
  (call-interactively command))

(defmacro imoogi-agent--define (name command doc)
  "Define interactive NAME that runs agent-shell COMMAND, documented by DOC."
  `(defun ,name ()
     ,(concat doc "\n\nRuns `" (symbol-name command) "'.")
     (interactive)
     (imoogi-agent--run ',command)))

(imoogi-agent--define imoogi-agent-open agent-shell
  "Open the project's agent shell, starting one if needed (agent-shell DWIM).")
(imoogi-agent--define imoogi-agent-toggle agent-shell-toggle
  "Show or hide the current agent shell.")
(imoogi-agent--define imoogi-agent-new agent-shell-new-shell
  "Start another agent shell.")
(imoogi-agent--define imoogi-agent-switch agent-shell-switch-buffer
  "Pick one of the open agent shells.")
(imoogi-agent--define imoogi-agent-send-region agent-shell-send-region-to
  "Send the active region to an agent shell.")
(imoogi-agent--define imoogi-agent-send-file agent-shell-send-file
  "Send the current file to the agent shell.")
(imoogi-agent--define imoogi-agent-send-other-file agent-shell-send-other-file
  "Pick a file and send it to the agent shell.")
(imoogi-agent--define imoogi-agent-interrupt agent-shell-interrupt
  "Stop the agent's current turn.")
(imoogi-agent--define imoogi-agent-resume agent-shell-resume-session
  "Resume an earlier agent session.")
(imoogi-agent--define imoogi-agent-set-model agent-shell-set-session-model
  "Change the model of the current session.")
(imoogi-agent--define imoogi-agent-set-mode agent-shell-set-session-mode
  "Change the mode (for example plan or auto-edit) of the current session.")
(imoogi-agent--define imoogi-agent-copy-last-output agent-shell-copy-last-output
  "Copy the agent's last answer.")
(imoogi-agent--define imoogi-agent-copy-session-id agent-shell-copy-session-id
  "Copy the current session id.")
(imoogi-agent--define imoogi-agent-view-logs agent-shell-view-acp-logs
  "Show the ACP traffic log.")

(defun imoogi-agent--read-backend ()
  "Ask which agent to start."
  (let ((choices (mapcar (lambda (b) (cons (plist-get (cdr b) :label) (car b)))
                         imoogi-agent-backends)))
    (cdr (assoc (completing-read "Agent: " choices nil t) choices))))

(defun imoogi-agent-start (&optional backend)
  "Start a new shell for BACKEND (claude, codex or gemini).
Without BACKEND use `imoogi-agent-default-backend'; with a prefix argument ask."
  (interactive (list (and current-prefix-arg (imoogi-agent--read-backend))))
  (let* ((backend (or backend imoogi-agent-default-backend))
         (spec (or (cdr (assq backend imoogi-agent-backends))
                   (user-error "Unknown agent backend: %s" backend))))
    (imoogi-agent--ensure)
    ;; The Claude and Codex adapters ship in the toolchain bundle (.local/bin),
    ;; which 17-lsp.el puts on PATH; do it here too in case that has not run.
    (when (fboundp 'imoogi-lsp-prepend-local-bin)
      (imoogi-lsp-prepend-local-bin))
    (let* ((variable (plist-get spec :command))
           (program (car (and (boundp variable) (symbol-value variable)))))
      (unless (and program (executable-find program))
        (user-error "%s needs the ACP adapter `%s' on PATH (set by `%s'); run `make toolchain-setup', then M-x imoogi-doctor"
                    (plist-get spec :label) (or program "?") variable)))
    (call-interactively (plist-get spec :start))))

(with-eval-after-load 'imoogi-transient
  (add-to-list 'imoogi-transient--purposes
               '(imoogi-agent-transient . "agent-shell 코딩 에이전트를 시작·전환하고 문맥을 보냅니다."))
  (transient-define-prefix imoogi-agent-transient ()
    "Coding agents in Emacs (agent-shell)."
    [["시작·전환 --------"
      ("a" "열기 (프로젝트)" imoogi-agent-open)
      ("s" "에이전트 골라 시작" imoogi-agent-start)
      ("t" "보이기/숨기기" imoogi-agent-toggle)
      ("n" "새 셸" imoogi-agent-new)
      ("b" "셸 고르기" imoogi-agent-switch)]
     ["문맥 보내기 --------"
      ("r" "선택 영역" imoogi-agent-send-region)
      ("f" "현재 파일" imoogi-agent-send-file)
      ("F" "다른 파일" imoogi-agent-send-other-file)]
     ["세션 --------"
      ("k" "멈추기" imoogi-agent-interrupt)
      ("R" "이전 세션 이어가기" imoogi-agent-resume)
      ("M" "모델 바꾸기" imoogi-agent-set-model)
      ("m" "모드 바꾸기" imoogi-agent-set-mode)]
     ["복사·기록 --------"
      ("c" "마지막 답 복사" imoogi-agent-copy-last-output)
      ("i" "세션 ID 복사" imoogi-agent-copy-session-id)
      ("l" "ACP 로그" imoogi-agent-view-logs)
      ("q" "종료" transient-quit-one)]])
  (transient-append-suffix 'imoogi-transient-master "i"
    '("A" "AI 에이전트" imoogi-agent-transient)))

(provide 'imoogi-agent)
;;; 31-agent.el ends here
