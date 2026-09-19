;;; imoogi-flashcards-org.el --- Org card extraction -*- lexical-binding: t; -*-

;;; Code:

(require 'cl-lib)
(require 'org)
(require 'seq)
(require 'subr-x)
(require 'imoogi-flashcards-core)

(defconst imoogi-flashcards-id-property "IMOOGI_FLASHCARD_ID")
(defconst imoogi-flashcards-kind-property "IMOOGI_FLASHCARD_KIND")
(defconst imoogi-flashcards-supported-kinds '("Basic" "Cloze"))

(defun imoogi-flashcards--uuid ()
  "Return a random UUID string."
  (if (fboundp 'uuidgen-4)
      (uuidgen-4)
    (md5 (format "%s-%s-%s" (current-time) (random) (user-uid)))))

(defun imoogi-flashcards--normalize-kind (kind)
  "Normalize KIND to a supported card kind or nil."
  (cond
   ((null kind) nil)
   ((string-equal-ignore-case kind "basic") "Basic")
   ((string-equal-ignore-case kind "front/back") "Basic")
   ((string-equal-ignore-case kind "front-back") "Basic")
   ((string-equal-ignore-case kind "cloze") "Cloze")
   (t kind)))

(defun imoogi-flashcards--at-heading ()
  "Move to the current Org heading or signal a user error."
  (unless (derived-mode-p 'org-mode)
    (user-error "imoogi flashcards: Org 버퍼에서만 쓸 수 있습니다"))
  (org-back-to-heading t))

(defun imoogi-flashcards--entry-body ()
  "Return current Org subtree body after heading metadata."
  (save-excursion
    (org-end-of-meta-data t)
    (let ((start (point))
          (end (save-excursion
                 (or (outline-next-heading) (point-max))
                 (point))))
      (string-trim (buffer-substring-no-properties start end)))))

(defun imoogi-flashcards--cloze-numbers (text)
  "Return distinct cloze numbers present in TEXT, sorted ascending."
  (let (numbers)
    (with-temp-buffer
      (insert text)
      (goto-char (point-min))
      (while (re-search-forward "{{c\\([0-9]+\\)::" nil t)
        (cl-pushnew (string-to-number (match-string 1)) numbers)))
    (sort numbers #'<)))

(defun imoogi-flashcards--cloze-front (text cloze-no)
  "Return TEXT with CLOZE-NO hidden."
  (replace-regexp-in-string
   (format "{{c%d::\\([^}]+\\)}}" cloze-no)
   "[...]"
   text nil nil))

(defun imoogi-flashcards--cloze-back (text)
  "Return TEXT with all cloze markers revealed."
  (replace-regexp-in-string "{{c[0-9]+::\\([^}]+\\)}}" "\\1" text nil nil))

(defun imoogi-flashcards--cards-at-point (source-file)
  "Return card structs for the Org heading at point.
SOURCE-FILE is the canonical source file stored in the projection."
  (let* ((kind (imoogi-flashcards--normalize-kind
                (org-entry-get (point) imoogi-flashcards-kind-property nil)))
         (note-id (org-entry-get (point) imoogi-flashcards-id-property nil)))
    (when kind
      (unless note-id
        (error "Missing %s on flashcard heading: %s"
               imoogi-flashcards-id-property
               (org-get-heading t t t t)))
      (unless (member kind imoogi-flashcards-supported-kinds)
        (error "Unsupported flashcard kind %S on heading: %s"
               kind (org-get-heading t t t t)))
      (let* ((heading (org-get-heading t t t t))
             (body (imoogi-flashcards--entry-body))
             (raw (string-trim (if (string-empty-p body)
                                   heading
                                 (concat heading "\n\n" body)))))
        (pcase kind
          ("Basic"
           (list (make-imoogi-flashcards-card
                  :key note-id
                  :note-id note-id
                  :kind kind
                  :source-file source-file
                  :heading heading
                  :front heading
                  :back body)))
          ("Cloze"
           (let ((numbers (imoogi-flashcards--cloze-numbers raw)))
             (unless numbers
               (error "Cloze flashcard has no {{cN::...}} marker: %s" heading))
             (mapcar
              (lambda (n)
                (let ((front (imoogi-flashcards--cloze-front raw n))
                      (back (imoogi-flashcards--cloze-back raw)))
                  (make-imoogi-flashcards-card
                   :key (format "%s::c%d" note-id n)
                   :note-id note-id
                   :kind kind
                   :cloze-no n
                   :source-file source-file
                   :heading heading
                   :front front
                   :back back)))
              numbers))))))))

(defun imoogi-flashcards-scan-file (file)
  "Scan Org FILE for flashcards.  Stored source paths remain absolute."
  (let* ((source-file (expand-file-name file))
         cards occurrences)
    (with-temp-buffer
      (insert-file-contents file)
      (delay-mode-hooks (org-mode))
      (goto-char (point-min))
      (org-map-entries
       (lambda ()
         (let ((note-id (org-entry-get (point) imoogi-flashcards-id-property nil))
               (kind (org-entry-get (point) imoogi-flashcards-kind-property nil)))
           (when note-id
             (push (list :note-id note-id
                         :source-file source-file
                         :heading (org-get-heading t t t t))
                   occurrences))
           (when kind
             (setq cards (nconc cards
                                (imoogi-flashcards--cards-at-point
                                 source-file))))))))
    (let ((duplicates (imoogi-flashcards--duplicate-note-ids occurrences)))
      (when duplicates
        (setq cards (seq-remove
                     (lambda (card)
                       (member (imoogi-flashcards-card-note-id card) duplicates))
                     cards)))
      (list :cards cards
            :occurrences (nreverse occurrences)
            :duplicate-note-ids duplicates))))

(defun imoogi-flashcards--duplicate-note-ids (occurrences)
  "Return duplicate note IDs from OCCURRENCES."
  (let (seen duplicates)
    (dolist (occ occurrences)
      (let ((id (plist-get occ :note-id)))
        (if (assoc id seen)
            (push id duplicates)
          (push (cons id occ) seen))))
    (delete-dups duplicates)))

(defun imoogi-flashcards--org-files (root)
  "Return sorted Org files under ROOT."
  (let (files)
    (dolist (name (sort (directory-files root nil "\\`[^.]") #'string<))
      (let ((path (expand-file-name name root)))
        (cond
         ((file-directory-p path)
          (setq files (nconc files (imoogi-flashcards--org-files path))))
         ((string-suffix-p ".org" path)
          (push path files)))))
    (nreverse files)))

(defun imoogi-flashcards-scan-root (root)
  "Scan ROOT for flashcards and duplicate note-id conflicts."
  (let (cards occurrences)
    (dolist (file (imoogi-flashcards--org-files root))
      (let ((result (imoogi-flashcards-scan-file file)))
        (setq cards (nconc cards (plist-get result :cards)))
        (setq occurrences (nconc occurrences (plist-get result :occurrences)))))
    (let ((duplicates (imoogi-flashcards--duplicate-note-ids occurrences)))
      (when duplicates
        (setq cards (seq-remove
                     (lambda (card)
                       (member (imoogi-flashcards-card-note-id card) duplicates))
                     cards)))
      (list :cards cards
            :occurrences occurrences
            :duplicate-note-ids duplicates))))

(defun imoogi-flashcards-resolve-card (source-file key)
  "Resolve current Org content for card KEY from SOURCE-FILE.
SQLite stores only source identifiers; this function re-reads Org as the
canonical card content at display time."
  (let* ((scan (imoogi-flashcards-scan-file source-file))
         (card (seq-find
                (lambda (candidate)
                  (equal (imoogi-flashcards-card-key candidate) key))
                (plist-get scan :cards))))
    (unless card
      (user-error "imoogi flashcards: Org 원본에서 카드를 찾을 수 없습니다: %s" key))
    card))

(defun imoogi-flashcards-mark-basic ()
  "Mark current Org heading as a Basic local flashcard."
  (interactive)
  (imoogi-flashcards--at-heading)
  (unless (org-entry-get (point) imoogi-flashcards-id-property nil)
    (org-set-property imoogi-flashcards-id-property (imoogi-flashcards--uuid)))
  (org-set-property imoogi-flashcards-kind-property "Basic")
  (message "imoogi flashcards: Basic 카드로 표시했습니다"))

(defun imoogi-flashcards-mark-cloze ()
  "Mark current Org heading as a Cloze local flashcard."
  (interactive)
  (imoogi-flashcards--at-heading)
  (unless (org-entry-get (point) imoogi-flashcards-id-property nil)
    (org-set-property imoogi-flashcards-id-property (imoogi-flashcards--uuid)))
  (org-set-property imoogi-flashcards-kind-property "Cloze")
  (message "imoogi flashcards: Cloze 카드로 표시했습니다"))

(defun imoogi-flashcards-unmark ()
  "Remove local flashcard kind from current Org heading.
The identity property is left in place so review history remains attached
if the card is marked again later."
  (interactive)
  (imoogi-flashcards--at-heading)
  (org-delete-property imoogi-flashcards-kind-property)
  (message "imoogi flashcards: 카드 표시를 해제했습니다"))

(defun imoogi-flashcards--next-cloze-number ()
  "Return the next cloze number for the current subtree."
  (save-excursion
    (imoogi-flashcards--at-heading)
    (let ((end (save-excursion (org-end-of-subtree t t)))
          (highest 0))
      (while (re-search-forward "{{c\\([0-9]+\\)::" end t)
        (setq highest (max highest (string-to-number (match-string 1)))))
      (1+ highest))))

(defun imoogi-flashcards-cloze-region (beg end &optional number)
  "Wrap region BEG..END as a cloze deletion.
NUMBER overrides the automatically selected cloze number."
  (interactive "r\nP")
  (unless (use-region-p)
    (user-error "imoogi flashcards: 빈칸으로 만들 영역을 먼저 선택하세요"))
  (let* ((n (if number
                (prefix-numeric-value number)
              (imoogi-flashcards--next-cloze-number)))
         (text (buffer-substring-no-properties beg end)))
    (delete-region beg end)
    (insert (format "{{c%d::%s}}" n text))
    (save-excursion
      (imoogi-flashcards-mark-cloze))
    (message "imoogi flashcards: c%d 빈칸을 만들었습니다" n)))

(provide 'imoogi-flashcards-org)
;;; imoogi-flashcards-org.el ends here
