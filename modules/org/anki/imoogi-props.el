;;; imoogi-props.el --- Nearest-wins property resolution -*- lexical-binding: t; -*-

;;; Commentary:

;; Resolves ANKI_DECK / ANKI_TAGS through the nearest-wins chain: the
;; current heading's own PROPERTIES drawer, then the nearest ancestor
;; heading's own drawer, then the file-level `#+PROPERTY:' keyword.
;; First hit wins; values are never merged across levels.
;;
;; Two rules the chain must honor (plan.md D-7):
;;
;;  - A present-but-empty value at any level TERMINATES the chain with
;;    an empty result -- it does not fall through to the next level.
;;  - Org's own accumulating `+' form (PROPERTY+) is normalized to plain
;;    replacement.  Org's built-in property accessor silently appends a
;;    `PROPERTY+' value to a same-level `PROPERTY' value even when
;;    inheritance is off, so this module reads the drawer text directly
;;    rather than delegating that comparison to `org-entry-get'.
;;
;; ANKI_NOTE_TYPE never inherits at all -- it is read from the target's
;; own drawer only, by `imoogi-scan.el', and this module is never
;; consulted for it.

;;; Code:

(require 'org)

(defun imoogi-props--own-value (property)
  "Return a cons (PRESENT . VALUE) for PROPERTY in the entry at point's
own PROPERTIES drawer only -- no inheritance, no walking to ancestors.

Reads the drawer text directly rather than `org-entry-get', because
Org's own accessor merges a same-level PROPERTY and PROPERTY+ into one
appended value even with inheritance off, which is exactly the
behavior D-7 requires this module NOT to exhibit.  When the drawer
contains PROPERTY, PROPERTY+, or both, the last matching line in the
drawer supplies the value as a plain override -- never an append.

PRESENT is non-nil when the property (either form) appears in the
drawer, even with an empty VALUE.  Returns (nil . nil) when neither
form appears."
  (let ((block (org-get-property-block)))
    (if (null block)
        (cons nil nil)
      (save-excursion
        (goto-char (car block))
        (let ((end (cdr block))
              (regexp (format "^[ \t]*:%s\\+?:[ \t]*\\(.*?\\)[ \t]*$"
                               (regexp-quote property)))
              (found nil)
              (value nil))
          (while (re-search-forward regexp end t)
            (setq found t)
            (setq value (match-string 1)))
          (if found (cons t value) (cons nil nil)))))))

(defun imoogi-props--ancestor-value (property)
  "Walk from point's parent heading upward, returning the first
ancestor's own-drawer PRESENT cons for PROPERTY, or (nil . nil) if no
ancestor's own drawer carries it."
  (save-excursion
    (let (result)
      (while (and (not result) (org-up-heading-safe))
        (let ((v (imoogi-props--own-value property)))
          (when (car v)
            (setq result v))))
      (or result (cons nil nil)))))

(defun imoogi-props--file-value (property)
  "Return a cons (PRESENT . VALUE) for the file-level
`#+PROPERTY: PROPERTY value' keyword, or (nil . nil)."
  (let* ((keywords (org-collect-keywords '("PROPERTY")))
         (entries (cdr (assoc "PROPERTY" keywords)))
         (regexp (format "\\`%s\\+?[ \t]+\\(.*\\)\\'" (regexp-quote property))))
    (catch 'imoogi-found
      (dolist (entry entries)
        (when (string-match regexp entry)
          (throw 'imoogi-found (cons t (match-string 1 entry)))))
      (cons nil nil))))

(defun imoogi-props-resolve (property)
  "Resolve PROPERTY at point via the nearest-wins chain: own drawer,
nearest ancestor's own drawer, file-level `#+PROPERTY:' keyword.

Returns the resolved string value, the empty string when a
present-but-empty value terminated the chain, or nil when PROPERTY is
absent at every level."
  (let ((own (imoogi-props--own-value property)))
    (if (car own)
        (cdr own)
      (let ((ancestor (imoogi-props--ancestor-value property)))
        (if (car ancestor)
            (cdr ancestor)
          (let ((file (imoogi-props--file-value property)))
            (if (car file)
                (cdr file)
              nil)))))))

(defun imoogi-props-resolve-deck ()
  "Resolve ANKI_DECK at point.

Returns the resolved string, or nil when absent everywhere or
terminated by a present-but-empty value -- either case selects the
back end's configured default deck (REQ-005), never resolved here."
  (let ((v (imoogi-props-resolve "ANKI_DECK")))
    (if (and v (not (string-empty-p v))) v nil)))

(defun imoogi-props-resolve-tags ()
  "Resolve ANKI_TAGS at point into a list of tag strings.

Returns nil (an empty tag set) when absent everywhere or terminated by
a present-but-empty value -- REQ-006's absent-value fallback."
  (let ((v (imoogi-props-resolve "ANKI_TAGS")))
    (if (and v (not (string-empty-p (string-trim v))))
        (split-string (string-trim v))
      nil)))

(provide 'imoogi-props)
;;; imoogi-props.el ends here
