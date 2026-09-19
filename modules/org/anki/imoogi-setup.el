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
(require 'imoogi-process)
(require 'imoogi-scan)
(require 'imoogi-writeback)

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

(defun imoogi-setup--call (thunk)
  "Run THUNK, degrading any signal to nil.

The subprocess calls below reach a binary whose behavior on a
half-written stdin (an immediate exit, a closed pipe) is not this
command's business to distinguish: every failure lands on the same
user-facing outcome -- no usable response -- and the report says so.
Letting a raw signal escape would surface an Emacs backtrace, which
REQ-018 forbids outright."
  (condition-case nil (funcall thunk) (error nil)))

(defun imoogi-setup--install-report (response)
  "Render the per-type outcome of an `install-models' RESPONSE (REQ-C-002).

Names each note type and the branch it took (added or updated), states
that the type's CSS is replaced wholesale so a hand edit made inside
Anki is known to be discarded, and renders any failure through
`imoogi-error-message' -- never through the Go error's own text."
  (if (null response)
      "Note-type install: no response from the binary."
    (let* ((results (plist-get response :results))
           (errors (plist-get response :errors))
           (lines (mapconcat
                   (lambda (r) (format "\n  %s: %s" (plist-get r :key) (plist-get r :action)))
                   results ""))
           (failures (mapconcat
                      (lambda (e)
                        (format "\n  %s: %s"
                                (or (plist-get e :key) "(whole run)")
                                (imoogi-error-message (plist-get e :code))))
                      errors "")))
      (concat (format "Note-type install: %d installed or updated, %d failed."
                      (length results) (length errors))
              lines failures
              (when results
                "\n  (Each type's CSS is replaced wholesale, so an edit made by hand inside Anki is discarded.)")))))

(defun imoogi-setup--install-step (binary)
  "Read the user stylesheet and run the install step (design.md SS6).
Returns the report string."
  (let* ((user-css (imoogi-user-stylesheet-contents))
         (response (imoogi-setup--call
                    (lambda ()
                      (imoogi-process-run-install binary imoogi-anki-connect-url user-css)))))
    (imoogi-setup--install-report response)))

(defun imoogi-setup--migrate-prompt (count)
  "REQ-C-019's confirmation text: it names COUNT and the loss."
  (format (concat "%d entries are recorded on a stock note type. "
                  "Migrate them onto imoogi's own note types? "
                  "Migrating discards each migrated note's review history "
                  "and scheduling state. ")
          count))

(defun imoogi-setup--migrate-step (binary root)
  "design.md SS7.1 steps 1-4 and 11, front-end half.

Dry-run for the candidate count; zero candidates finish silently; a
non-zero count is put to the user with `y-or-n-p' naming both the count
and the scheduling loss; a decline issues NO writing `migrate' request
at all (AC-C-018c); a confirmation issues the same request document
without --dry-run and writes back both properties (REQ-C-020.3).

Returns the report string."
  (if (not (and root (file-directory-p root)))
      "Migration check: skipped -- no sync root exists yet."
    (let* ((scan (imoogi-setup--call
                  (lambda () (imoogi-scan-root root imoogi-exclude-patterns))))
           (entries (plist-get scan :entries))
           (config (list :default-deck imoogi-default-deck
                         :anki-connect-url imoogi-anki-connect-url
                         :registry-path (imoogi--registry-path root)
                         :sync-root root
                         :exclude-patterns imoogi-exclude-patterns
                         :scan-complete (plist-get scan :scan-complete)))
           (dry (imoogi-setup--call
                 (lambda ()
                   (imoogi-process-run-migrate binary config
                                               (plist-get scan :census)
                                               entries t)))))
      (cond
       ((null dry) "Migration check: no response from the binary.")
       (t
        (let ((count (length (imoogi-process-migrate-candidates dry))))
          (cond
           ;; Step 2 -- no candidates, so no prompt.  The count is still
           ;; reported: "0" is the answer to a question the user would
           ;; otherwise have to ask again next time.
           ((zerop count) "Migration candidates: 0 -- nothing to migrate.")
           ((not (y-or-n-p (imoogi-setup--migrate-prompt count)))
            (format "Migration candidates: %d -- declined, so nothing was migrated." count))
           (t
            (let ((response (imoogi-setup--call
                             (lambda ()
                               (imoogi-process-run-migrate binary config
                                                           (plist-get scan :census)
                                                           entries nil)))))
              (if (null response)
                  "Migration: no response from the binary."
                (let ((needs-save (imoogi-writeback-apply-migration root
                                                                    (plist-get response :results))))
                  (concat
                   (format "Migration: %d candidates, %d migrated, %d failed."
                           count
                           (length (plist-get response :results))
                           (length (plist-get response :errors)))
                   (mapconcat (lambda (e)
                                (format "\n  %s: %s"
                                        (or (plist-get e :key) "(whole run)")
                                        (imoogi-error-message (plist-get e :code))))
                              (plist-get response :errors) "")
                   (if needs-save
                       (format " Needs save: %s" (mapconcat #'identity needs-save ", "))
                     "")))))))))))))

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
                ;; The tail (design.md SS6 and SS7.1): install imoogi's own
                ;; note types, then offer the migration.  It is APPENDED to
                ;; the configuration line rather than replacing it -- setup
                ;; having succeeded is the thing the user came for, and a
                ;; failing install step must not read as a failing setup.
                (concat
                 (format "imoogi setup complete: binary check succeeded, AnkiConnect check succeeded. Config written to %s."
                         target)
                 "\n" (imoogi-setup--install-step binary)
                 "\n" (imoogi-setup--migrate-step binary root)))
               ((eq status 'unreachable) (imoogi-error-message "anki_unreachable"))
               (t (imoogi-error-message "ankiconnect_missing")))))))
    (message "imoogi: %s" report)
    report))

(provide 'imoogi-setup)
;;; imoogi-setup.el ends here
