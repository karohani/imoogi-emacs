;;; imoogi-props-test.el --- Tests for imoogi-props.el -*- lexical-binding: t; -*-

;;; Code:

(require 'ert)
(require 'org)
(require 'imoogi-props)

(defmacro imoogi-test--with-org-buffer (content &rest body)
  "Run BODY in a temp buffer visiting CONTENT as Org text."
  (declare (indent 1))
  `(with-temp-buffer
     (insert ,content)
     (org-mode)
     (goto-char (point-min))
     ,@body))

(defun imoogi-test--goto-heading (title)
  "Move point to the heading whose text is TITLE."
  (goto-char (point-min))
  (re-search-forward (format "^\\*+ %s$" (regexp-quote title))))

;; --- AC-010 shape: nearest-wins, including the empty-value case. One
;; of the four highest-risk behaviors named in the delegation prompt.

(ert-deftest imoogi-props-test-nearest-wins-and-empty-terminates ()
  (imoogi-test--with-org-buffer
      (concat
       "#+PROPERTY: ANKI_DECK FileDeck\n"
       "* Parent\n:PROPERTIES:\n:ANKI_DECK: ParentDeck\n:END:\n"
       "** W\n:PROPERTIES:\n:ANKI_DECK: OwnDeck\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody W\n"
       "** X\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody X\n"
       "** Y\n:PROPERTIES:\n:ANKI_DECK:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody Y\n"
       "* Z\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody Z\n")
    (imoogi-test--goto-heading "W")
    (should (equal (imoogi-props-resolve-deck) "OwnDeck"))
    (imoogi-test--goto-heading "X")
    (should (equal (imoogi-props-resolve-deck) "ParentDeck"))
    (imoogi-test--goto-heading "Z")
    (should (equal (imoogi-props-resolve-deck) "FileDeck"))
    (imoogi-test--goto-heading "Y")
    ;; a present-but-empty value terminates the chain -- resolve-deck
    ;; returns nil (never "" and never the inherited ParentDeck), which
    ;; is the back end's default-deck-fallback trigger.
    (should (null (imoogi-props-resolve-deck)))
    (should (equal (imoogi-props-resolve "ANKI_DECK") ""))))

;; --- The `+' accumulating form is normalized to plain replacement.
;; One of the four highest-risk behaviors named in the delegation prompt.

(ert-deftest imoogi-props-test-plus-form-only-reads-as-plain-override ()
  (imoogi-test--with-org-buffer
      (concat
       "* Parent\n:PROPERTIES:\n:ANKI_TAGS: base\n:END:\n"
       "** Child\n:PROPERTIES:\n:ANKI_TAGS+: extra\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody\n")
    (imoogi-test--goto-heading "Child")
    ;; own drawer carries only the `+' form -- read as a plain override,
    ;; never appended to the parent's "base".
    (should (equal (imoogi-props-resolve "ANKI_TAGS") "extra"))
    (should (equal (imoogi-props-resolve-tags) '("extra")))))

(ert-deftest imoogi-props-test-plus-form-same-heading-does-not-merge ()
  (imoogi-test--with-org-buffer
      (concat
       "* Heading\n:PROPERTIES:\n:ANKI_TAGS: base\n:ANKI_TAGS+: extra\n"
       ":ANKI_NOTE_TYPE: Basic\n:END:\nBody\n")
    (imoogi-test--goto-heading "Heading")
    ;; both the plain and the `+' form appear on the SAME heading's own
    ;; drawer. Org's own accessor would append ("base extra"); D-7
    ;; requires plain replacement -- the last matching line wins, never
    ;; a merge of both.
    (should (equal (imoogi-props-resolve "ANKI_TAGS") "extra"))))

(ert-deftest imoogi-props-test-tags-empty-when-absent ()
  (imoogi-test--with-org-buffer
      "* NoTags\n:PROPERTIES:\n:ANKI_NOTE_TYPE: Basic\n:END:\nBody\n"
    (imoogi-test--goto-heading "NoTags")
    (should (null (imoogi-props-resolve-tags)))))

(ert-deftest imoogi-props-test-tags-space-separated ()
  (imoogi-test--with-org-buffer
      (concat "* T\n:PROPERTIES:\n:ANKI_TAGS: geography europe\n"
              ":ANKI_NOTE_TYPE: Basic\n:END:\nBody\n")
    (imoogi-test--goto-heading "T")
    (should (equal (imoogi-props-resolve-tags) '("geography" "europe")))))

(provide 'imoogi-props-test)
;;; imoogi-props-test.el ends here
