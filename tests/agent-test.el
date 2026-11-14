;;; agent-test.el --- imoogi-agent-* interface over agent-shell -*- lexical-binding: t; -*-

;; The imoogi-agent-* commands are the only names the rest of the config uses.
;; These tests pin each one to the agent-shell function it reaches, and fail
;; naming the missing upstream functions when an agent-shell upgrade renames one.

;;; Code:

(require 'ert)
(require 'cl-lib)

(ert-deftest imoogi-agent-vendored-agent-shell-has-every-wrapped-function ()
  "The vendored agent-shell provides every function the wrapper reaches."
  (should (equal (imoogi-agent-missing-api) nil)))

(ert-deftest imoogi-agent-missing-upstream-function-is-named ()
  "A renamed upstream function fails with its name instead of a void-function."
  (let ((imoogi-agent--upstream-api (cons 'agent-shell-renamed-away imoogi-agent--upstream-api)))
    (let ((err (should-error (imoogi-agent-toggle) :type 'user-error)))
      (should (string-match-p "agent-shell-renamed-away" (cadr err)))
      (should (string-match-p "31-agent" (cadr err))))))

(defmacro imoogi-agent-test--recording (&rest body)
  "Run BODY with every wrapped upstream command recorded instead of run."
  (declare (indent 0))
  `(let ((calls nil))
     (imoogi-agent--ensure)
     (cl-letf (((symbol-function 'call-interactively)
                (lambda (fn &rest _) (push fn calls)))
               ((symbol-function 'executable-find) (lambda (_) "/usr/bin/true")))
       ,@body)
     (nreverse calls)))

(ert-deftest imoogi-agent-commands-reach-their-agent-shell-functions ()
  (dolist (row '((imoogi-agent-open . agent-shell)
                 (imoogi-agent-toggle . agent-shell-toggle)
                 (imoogi-agent-new . agent-shell-new-shell)
                 (imoogi-agent-switch . agent-shell-switch-buffer)
                 (imoogi-agent-send-region . agent-shell-send-region-to)
                 (imoogi-agent-send-file . agent-shell-send-file)
                 (imoogi-agent-send-other-file . agent-shell-send-other-file)
                 (imoogi-agent-interrupt . agent-shell-interrupt)
                 (imoogi-agent-resume . agent-shell-resume-session)
                 (imoogi-agent-set-model . agent-shell-set-session-model)
                 (imoogi-agent-set-mode . agent-shell-set-session-mode)
                 (imoogi-agent-copy-last-output . agent-shell-copy-last-output)
                 (imoogi-agent-copy-session-id . agent-shell-copy-session-id)
                 (imoogi-agent-view-logs . agent-shell-view-acp-logs)))
    (should (equal (imoogi-agent-test--recording (funcall (car row))) (list (cdr row))))))

(ert-deftest imoogi-agent-start-picks-the-backend ()
  (should (equal (imoogi-agent-test--recording (imoogi-agent-start 'claude))
                 '(agent-shell-anthropic-start-claude-code)))
  (should (equal (imoogi-agent-test--recording (imoogi-agent-start 'codex))
                 '(agent-shell-openai-start-codex)))
  (should (equal (imoogi-agent-test--recording (imoogi-agent-start 'gemini))
                 '(agent-shell-google-start-gemini)))
  (let ((imoogi-agent-default-backend 'codex))
    (should (equal (imoogi-agent-test--recording (imoogi-agent-start))
                   '(agent-shell-openai-start-codex)))))

(ert-deftest imoogi-agent-start-explains-a-missing-acp-adapter ()
  (imoogi-agent--ensure)
  (cl-letf (((symbol-function 'executable-find) (lambda (_) nil))
            ((symbol-function 'call-interactively) (lambda (&rest _) (error "must not start"))))
    (let ((err (should-error (imoogi-agent-start 'claude) :type 'user-error)))
      (should (string-match-p "claude-agent-acp" (cadr err))))))

;;; agent-test.el ends here
