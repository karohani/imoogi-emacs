;;; anki-src-open-test.el --- SPEC-ANKISRC-001 open side: the Emacs receiver -*- lexical-binding: t; -*-

;;; Commentary:

;; AC-ASRC-009..013 and the Elisp half of AC-ASRC-014: `imoogi-anki-open-id',
;; the fixed function the `imoogi-anki open' handler evaluates through
;; emacsclient.  Every test isolates the Org ID cache to temporary state and
;; stubs the frame-focus function so its calls can be counted.

;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'org)
(require 'org-id)
(require 'imoogi)
(require 'imoogi-open)

(defvar imoogi-anki-src-open-test--focus 0
  "How many times the receiver raised the Emacs frame.")

(defvar imoogi-anki-src-open-test--messages nil
  "Messages the receiver showed, newest first.")

(defmacro imoogi-anki-src-open-test--with-root (&rest body)
  "Run BODY with BASE, ROOT (the legacy sync root) and isolated Org ID state."
  (declare (indent 0))
  `(let* ((base (file-name-as-directory (make-temp-file "imoogi-src-open" t)))
          (root (file-name-as-directory (expand-file-name "notes" base)))
          (locations (expand-file-name "state/org-id-locations" base))
          (org-id-locations-file locations)
          (org-id-locations (make-hash-table :test 'equal))
          (org-id-files nil)
          (org-id-track-globally t)
          (imoogi-targets-file (expand-file-name "state/targets.json" base))
          (imoogi-sync-root root)
          (imoogi-exclude-patterns nil)
          (imoogi-anki-src-open-test--focus 0)
          (imoogi-anki-src-open-test--messages nil)
          (prev-window-buffer (window-buffer (selected-window)))
          (prev-current-buffer (current-buffer))
          (start-buffer (generate-new-buffer " *imoogi-open-start*")))
     (make-directory root t)
     (make-directory (file-name-directory locations) t)
     (unwind-protect
         (cl-letf (((symbol-function 'imoogi-anki-open--focus-frame)
                    (lambda () (cl-incf imoogi-anki-src-open-test--focus)))
                   ((symbol-function 'message)
                    (lambda (format-string &rest args)
                      (when format-string
                        (push (apply #'format-message format-string args)
                              imoogi-anki-src-open-test--messages)))))
           (set-window-buffer (selected-window) start-buffer)
           (set-buffer start-buffer)
           ,@body)
       (dolist (buffer (buffer-list))
         (let ((file (buffer-file-name buffer)))
           (when (and file (file-in-directory-p file base))
             (with-current-buffer buffer (set-buffer-modified-p nil))
             (kill-buffer buffer))))
       (when (get-buffer "*imoogi-anki-open*") (kill-buffer "*imoogi-anki-open*"))
       ;; Hand the selected window and the current buffer back as found, so
       ;; later tests do not inherit whatever the receiver displayed.
       (when (buffer-live-p prev-window-buffer)
         (set-window-buffer (selected-window) prev-window-buffer))
       (when (buffer-live-p prev-current-buffer)
         (set-buffer prev-current-buffer))
       (kill-buffer start-buffer)
       (delete-directory base t))))

(defun imoogi-anki-src-open-test--write (dir relative id &optional title)
  "Write a one-card Org file at RELATIVE under DIR whose heading has ID."
  (let ((file (expand-file-name relative dir)))
    (make-directory (file-name-directory file) t)
    (with-temp-file file
      (insert "#+TITLE: fixture\n"
              "* Unrelated\nText.\n"
              (format "* %s\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ID: %s\n:END:\nBody\n"
                      (or title "Card") id)))
    file))

(defun imoogi-anki-src-open-test--cache (id file)
  "Put ID -> FILE into the in-memory location cache, Org's way."
  (puthash id (abbreviate-file-name file) org-id-locations))

(defun imoogi-anki-src-open-test--at-heading-p (file id)
  "Non-nil when the current buffer visits FILE and point is on ID's heading."
  (and buffer-file-name
       (equal (file-truename buffer-file-name) (file-truename file))
       (org-at-heading-p)
       (equal (org-entry-get (point) "ID" nil) id)))

(defun imoogi-anki-src-open-test--saved-path (id)
  "Return the truename `org-id-locations-file' records for ID, or nil."
  (when (file-exists-p org-id-locations-file)
    (let ((data (with-temp-buffer
                  (insert-file-contents org-id-locations-file)
                  (read (current-buffer))))
          found)
      (dolist (item data)
        (when (member id (cdr item))
          (setq found (file-truename (expand-file-name (car item))))))
      found)))

;; --- AC-ASRC-009: a cache hit opens the heading.

(ert-deftest imoogi-anki-src-open-cache-hit-opens-heading ()
  (imoogi-anki-src-open-test--with-root
    (let ((file (imoogi-anki-src-open-test--write root "a.org" "id-a")))
      (imoogi-anki-src-open-test--write root "b.org" "id-b")
      (imoogi-anki-src-open-test--cache "id-a" file)
      (should (equal (imoogi-anki-open-id "id-a") "ok"))
      (should (imoogi-anki-src-open-test--at-heading-p file "id-a"))
      (should (= imoogi-anki-src-open-test--focus 1)))))

(ert-deftest imoogi-anki-src-open-cached-file-outside-targets-is-accepted ()
  (imoogi-anki-src-open-test--with-root
    (let ((outside (imoogi-anki-src-open-test--write base "elsewhere/o.org" "id-o")))
      (imoogi-anki-src-open-test--write root "a.org" "id-a")
      (imoogi-anki-src-open-test--cache "id-o" outside)
      (should (equal (imoogi-anki-open-id "id-o") "ok"))
      (should (imoogi-anki-src-open-test--at-heading-p outside "id-o"))
      (should (= imoogi-anki-src-open-test--focus 1)))))

;; --- AC-ASRC-010: a moved file is found again and re-registered.

(ert-deftest imoogi-anki-src-open-moved-file-is-rescanned-and-recached ()
  (imoogi-anki-src-open-test--with-root
    (let ((moved (imoogi-anki-src-open-test--write root "new/a.org" "id-m")))
      (imoogi-anki-src-open-test--cache "id-m" (expand-file-name "old/a.org" root))
      (should (equal (imoogi-anki-open-id "id-m") "ok"))
      (should (imoogi-anki-src-open-test--at-heading-p moved "id-m"))
      (should (equal (imoogi-anki-src-open-test--saved-path "id-m") (file-truename moved))))))

;; --- AC-ASRC-011: nothing found -> a message, no file, no buffer.

(ert-deftest imoogi-anki-src-open-missing-id-creates-nothing ()
  (imoogi-anki-src-open-test--with-root
    (let ((gone (expand-file-name "deleted.org" root))
          (buffers (buffer-list)))
      (imoogi-anki-src-open-test--write root "a.org" "id-a")
      (dolist (case '(("id-gone" . t) ("id-never" . nil)))
        (setq imoogi-anki-src-open-test--messages nil)
        (when (cdr case)
          (imoogi-anki-src-open-test--cache (car case) gone))
        (should (equal (imoogi-anki-open-id (car case)) "not-found"))
        (let ((shown (string-join imoogi-anki-src-open-test--messages "\n")))
          (should (string-match-p (regexp-quote (car case)) shown))
          (should (string-match-p "imoogi-sync" shown))))
      (should-not (file-exists-p gone))
      (should-not (find-buffer-visiting gone))
      (should-not (cl-set-difference (buffer-list) buffers))
      (should (eq (current-buffer) (get-buffer " *imoogi-open-start*")))
      (should (= imoogi-anki-src-open-test--focus 2)))))

;; --- AC-ASRC-012: a duplicate ID lists the candidates and moves nowhere.

(ert-deftest imoogi-anki-src-open-duplicate-lists-candidates ()
  (dolist (cached '(nil t))
    (imoogi-anki-src-open-test--with-root
      (let ((one (imoogi-anki-src-open-test--write root "one.org" "id-d" "First copy"))
            (two (imoogi-anki-src-open-test--write root "sub/two.org" "id-d" "Second copy")))
        (when cached (imoogi-anki-src-open-test--cache "id-d" one))
        (with-current-buffer " *imoogi-open-start*" (insert "abc") (goto-char 2))
        (should (equal (imoogi-anki-open-id "id-d") "duplicate"))
        (should (eq (current-buffer) (get-buffer " *imoogi-open-start*")))
        (should (= (point) 2))
        (let ((listing (with-current-buffer "*imoogi-anki-open*" (buffer-string))))
          (should (string-match-p (regexp-quote (file-truename one)) listing))
          (should (string-match-p (regexp-quote (file-truename two)) listing))
          (should (string-match-p "First copy" listing))
          (should (string-match-p "Second copy" listing)))
        (should (= imoogi-anki-src-open-test--focus 1))))))

;; O4: a property line with trailing whitespace still counts as a match, so
;; the duplicate is not missed.
(ert-deftest imoogi-anki-src-open-trailing-whitespace-still-counts ()
  (imoogi-anki-src-open-test--with-root
    (imoogi-anki-src-open-test--write root "one.org" "id-w")
    (let ((two (expand-file-name "two.org" root)))
      (with-temp-file two
        (insert "* Copy\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:ID: id-w \t\n:END:\nBody\n"))
      (should (equal (imoogi-anki-open-id "id-w") "duplicate")))))

;; --- AC-ASRC-013: the user's unsaved edits are left exactly as they were.

(ert-deftest imoogi-anki-src-open-keeps-unsaved-user-buffer-untouched ()
  (imoogi-anki-src-open-test--with-root
    (let* ((file (imoogi-anki-src-open-test--write root "a.org" "id-u"))
           (disk (with-temp-buffer (insert-file-contents file) (buffer-string)))
           (buffer (find-file-noselect file))
           text)
      (with-current-buffer buffer
        (goto-char (point-min))
        (forward-line 1)
        (insert "* Unsaved heading the user is writing\n")
        (setq text (buffer-string)))
      (set-buffer " *imoogi-open-start*")
      (should (equal (imoogi-anki-open-id "id-u") "ok"))
      (should (eq (current-buffer) buffer))
      (should (imoogi-anki-src-open-test--at-heading-p file "id-u"))
      (should (buffer-modified-p buffer))
      (should (equal (buffer-string) text))
      (should (equal (with-temp-buffer (insert-file-contents file) (buffer-string)) disk)))))

;; --- AC-ASRC-014 (Elisp half): the receiver re-validates its input.

(ert-deftest imoogi-anki-src-open-rejects-ids-outside-the-character-set ()
  (imoogi-anki-src-open-test--with-root
    (let ((buffers (buffer-list)))
      (imoogi-anki-src-open-test--write root "a.org" "id-a")
      (dolist (bad '("x\")(shell-command \"touch pwn\"" "a b" "" "a/b"))
        (should-not (member (imoogi-anki-open-id bad) '("ok" "not-found" "duplicate"))))
      (should-not (cl-set-difference (buffer-list) buffers))
      (should (eq (current-buffer) (get-buffer " *imoogi-open-start*"))))))

;; --- REQ-ASRC-011.7: the receiver loads with the Anki module itself.

(ert-deftest imoogi-anki-src-open-receiver-is-required-by-the-module ()
  (should (fboundp 'imoogi-anki-open-id))
  (let ((module (seq-find #'file-readable-p
                          (list (expand-file-name "modules/org/24-anki.el" default-directory)
                                (and (boundp 'imoogi-emacs-dir)
                                     (expand-file-name "modules/org/24-anki.el" imoogi-emacs-dir))))))
    (skip-unless module)
    (with-temp-buffer
      (insert-file-contents module)
      (should (re-search-forward "^(require 'imoogi-open)" nil t)))))

(provide 'anki-src-open-test)
;;; anki-src-open-test.el ends here
