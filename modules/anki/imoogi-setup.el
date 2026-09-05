;;; imoogi-setup.el --- The imoogi-anki-setup interactive command -*- lexical-binding: t; -*-

;;; Commentary:

;; REQ-017: verify the configured binary is present and executable,
;; verify Anki is running and the AnkiConnect add-on responds, prompt
;; for a sync root and a default deck, and write the resulting
;; configuration -- so a first-time user reaches a working
;; configuration without hand-editing any file (AC-003).
;;
;; SYNC-ROOT and DEFAULT-DECK are accepted as optional arguments so this
;; command is directly callable from tests without simulating
;; keystrokes; `imoogi-anki-setup' falls back to interactive prompts
;; only when they are omitted.
;;
;; The AnkiConnect check is a direct HTTP handshake -- imoogi.el's own
;; `requestPermission' call, mirroring internal/ankiconnect's Handshake
;; (research.md SS3.5) -- rather than a round trip through the Go
;; binary, since no `sync' subcommand invocation is appropriate before
;; a sync root and registry even exist.

;;; Code:

(require 'url)
(require 'url-http)
(require 'json)
(require 'imoogi)
(require 'imoogi-config)
(require 'imoogi-error)

;; `imoogi-binary-path' and `imoogi-anki-connect-url' are the real
;; defcustoms from imoogi.el, `require'd above -- NOT forward-declared.
;;
;; This is deliberate, not merely a preference: `imoogi-anki-setup' is
;; autoloaded, so a first-time user's very first invocation of it can
;; load THIS FILE ALONE, with imoogi.el never having loaded and its
;; defcustoms never having run.  A nil-valued forward declaration (the
;; pattern imoogi-config.el uses for its own two symbols) would silently
;; "win" that binding as nil in this exact path, since a later
;; `defcustom' does not overwrite an already-bound value -- and
;; `executable-find' signals `wrong-type-argument' on a nil path, which
;; is precisely the raw-backtrace failure REQ-018 forbids.  Requiring
;; `imoogi' here instead guarantees the real defaults are in place
;; before `imoogi-anki-setup' can run, at the one-time cost of imoogi.el
;; itself deliberately NOT requiring `imoogi-setup' back (see imoogi.el's
;; own comment on this), which would otherwise be circular.

;; url-http.el declares this buffer-local via `defvar-local' inside a
;; function body rather than at file top level, so `require'ing
;; url-http does not, by itself, teach the byte compiler that the
;; symbol is special -- this forward declaration silences that warning
;; without changing behavior.
(defvar url-http-response-status)

(defun imoogi-setup--ankiconnect-status (url)
  "Probe URL with an AnkiConnect `requestPermission' handshake.

Returns one of:
  `ok'               -- URL answered with a well-formed AnkiConnect
                         handshake response (a JSON body with a
                         \"result\" key).
  `unreachable'       -- nothing answered at URL at all (AC-005's
                         Given: Anki is very likely not running).
  `invalid-response'  -- something answered, but not with a valid
                         AnkiConnect handshake (AC-006's Given: the
                         add-on is absent or unresponsive)."
  (condition-case nil
      (let* ((url-request-method "POST")
             (url-request-extra-headers '(("Content-Type" . "application/json")))
             (url-request-data
              (json-serialize (list (cons 'action "requestPermission")
                                     (cons 'version 6))))
             (buffer (url-retrieve-synchronously url t t 5)))
        (if (null buffer)
            'unreachable
          (unwind-protect
              (with-current-buffer buffer
                (cond
                 ;; No HTTP response was ever received at all -- e.g. a
                 ;; connection-refused: url.el still returns a buffer, but
                 ;; `url-http-response-status' stays nil and the buffer is
                 ;; empty.  This IS AC-005's Given (nothing listening).
                 ((not (and (boundp 'url-http-response-status)
                            url-http-response-status))
                  'unreachable)
                 ;; Something answered, but not with HTTP 200 -- AC-006's
                 ;; Given (an add-on-shaped absence: the host answered the
                 ;; wrong thing).
                 ((not (eql url-http-response-status 200))
                  'invalid-response)
                 (t
                  (goto-char (point-min))
                  (if (search-forward "\n\n" nil t)
                      (let* ((body (buffer-substring (point) (point-max)))
                             (parsed (condition-case nil
                                         (json-parse-string body :object-type 'alist)
                                       (error nil))))
                        (if (and parsed (assq 'result parsed))
                            'ok
                          'invalid-response))
                    'invalid-response))))
            (kill-buffer buffer))))
    (error 'unreachable)))

;;;###autoload
(defun imoogi-anki-setup (&optional sync-root default-deck config-file)
  "Verify the binary and AnkiConnect, prompt for a sync root and a
default deck, and write the resulting configuration file (REQ-017).

Interactive use prompts for SYNC-ROOT and DEFAULT-DECK.  Both may be
supplied directly -- the arguments exist chiefly so this command is
callable from tests without simulating keystrokes.  CONFIG-FILE
defaults to `imoogi-config-file'.

Returns the human-readable report string (also shown via `message')."
  (interactive)
  (let* ((root (or sync-root (read-directory-name "imoogi sync root: ")))
         (deck (or default-deck (read-string "Default deck: " imoogi-default-deck)))
         (target (or config-file imoogi-config-file))
         (binary (executable-find imoogi-binary-path))
         (report
          (if (null binary)
              (imoogi-error-message "binary_not_found")
            (let ((status (imoogi-setup--ankiconnect-status imoogi-anki-connect-url)))
              (cond
               ((eq status 'ok)
                (imoogi-config-write target root deck)
                (setq imoogi-sync-root root)
                (setq imoogi-default-deck deck)
                (format "imoogi setup complete: binary check succeeded, AnkiConnect check succeeded. Config written to %s."
                        target))
               ((eq status 'unreachable) (imoogi-error-message "anki_unreachable"))
               (t (imoogi-error-message "ankiconnect_missing")))))))
    (message "imoogi: %s" report)
    report))

(provide 'imoogi-setup)
;;; imoogi-setup.el ends here
