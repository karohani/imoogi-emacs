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

;; --- SPEC-ANKICARD-002: the three card-option properties resolve through
;; the SAME chain, and the front end interprets none of their values.

;; AC-OPT-001a/b/c: own drawer beats the nearest ancestor, which beats a
;; farther ancestor, which beats the file-level keyword.
(ert-deftest imoogi-props-test-card-options-resolve-nearest-wins ()
  (imoogi-test--with-org-buffer
      (concat
       "#+PROPERTY: ANKI_SWIFT nil\n"
       "#+PROPERTY: ANKI_INCREMENTAL t\n"
       "* Grand\n:PROPERTIES:\n:ANKI_SWIFT: nil\n:ANKI_DIRECTION: ->\n:END:\n"
       "** Parent\n:PROPERTIES:\n:ANKI_SWIFT: t\n:END:\n"
       "*** Own\n:PROPERTIES:\n:ANKI_DIRECTION: <-\n:ANKI_NOTE_TYPE: imoogi-Cloze\n:END:\nBody\n"
       "*** Inherited\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Cloze\n:END:\nBody\n")
    ;; a: the heading's own drawer wins over an ancestor's.
    (imoogi-test--goto-heading "Own")
    (should (equal (imoogi-props-resolve-direction) "<-"))
    ;; b: the NEAREST ancestor wins over a farther one and over the file.
    (should (equal (imoogi-props-resolve-swift) "t"))
    ;; c: the file-level keyword is the last level, reached when no
    ;; heading in the chain carries the property.
    (should (equal (imoogi-props-resolve-incremental) "t"))
    ;; the inherited-direction case: no own value, so the grandparent's.
    (imoogi-test--goto-heading "Inherited")
    (should (equal (imoogi-props-resolve-direction) "->"))))

;; AC-OPT-001d: a present-but-empty value TERMINATES the chain and resolves
;; to no value -- never the empty string treated as a value, and never the
;; inherited value falling through. This is the per-heading opt-out that
;; makes an inherited option suppressible.
(ert-deftest imoogi-props-test-card-options-empty-value-terminates-chain ()
  (imoogi-test--with-org-buffer
      (concat
       "#+PROPERTY: ANKI_SWIFT t\n"
       "#+PROPERTY: ANKI_DIRECTION ->\n"
       "#+PROPERTY: ANKI_INCREMENTAL t\n"
       "* Suppressed\n:PROPERTIES:\n:ANKI_SWIFT:\n:ANKI_DIRECTION:\n"
       ":ANKI_INCREMENTAL:\n:ANKI_NOTE_TYPE: imoogi-Cloze\n:END:\nBody\n")
    (imoogi-test--goto-heading "Suppressed")
    (should (null (imoogi-props-resolve-swift)))
    (should (null (imoogi-props-resolve-direction)))
    (should (null (imoogi-props-resolve-incremental)))
    ;; the raw chain still reports the empty string -- the collapse to no
    ;; value is the resolver's, exactly as it is for ANKI_DECK.
    (should (equal (imoogi-props-resolve "ANKI_SWIFT") ""))))

;; AC-OPT-001e: the accumulating `+' form is normalized to plain replacement.
;;
;; This is the criterion that discriminates REUSE of the existing chain from
;; a second implementation. Sub-criteria a through d describe a nearest-wins
;; chain any competent reimplementation would also satisfy; `PROPERTY+'
;; normalization is the one rule the existing chain deliberately does NOT
;; inherit from Org's own property accessor, which appends a same-level
;; `PROPERTY+' value even with inheritance off. A resolver built on that
;; accessor fails here and passes everything else.
(ert-deftest imoogi-props-test-card-options-plus-form-is-plain-override ()
  (imoogi-test--with-org-buffer
      (concat
       "* Plus\n:PROPERTIES:\n"
       ":ANKI_DIRECTION: ->\n:ANKI_DIRECTION+: <-\n"
       ":ANKI_INCREMENTAL: nil\n:ANKI_INCREMENTAL+: t\n"
       ":ANKI_SWIFT: t\n:ANKI_SWIFT+: nil\n"
       ":ANKI_NOTE_TYPE: imoogi-Cloze\n:END:\nBody\n")
    (imoogi-test--goto-heading "Plus")
    ;; the LAST matching drawer line wins as a plain override -- never the
    ;; appended form "-> <-".
    (should (equal (imoogi-props-resolve-direction) "<-"))
    (should (equal (imoogi-props-resolve-incremental) "t"))
    (should (equal (imoogi-props-resolve-swift) "nil"))))

;; AC-OPT-002a/b: the front end interprets nothing. A malformed value is
;; carried verbatim, case is not folded, and no spelling is mapped onto
;; another. Deciding what a value MEANS is the back end's obligation, and a
;; front end that coerced first would make that decision unreachable.
(ert-deftest imoogi-props-test-card-options-are-carried-verbatim ()
  (imoogi-test--with-org-buffer
      (concat
       "* Malformed\n:PROPERTIES:\n:ANKI_DIRECTION: -->\n"
       ":ANKI_INCREMENTAL: yes\n:ANKI_NOTE_TYPE: imoogi-Cloze\n:END:\nBody\n"
       "* Cased\n:PROPERTIES:\n:ANKI_INCREMENTAL: T\n:ANKI_SWIFT: NIL\n"
       ":ANKI_DIRECTION: <->\n:ANKI_NOTE_TYPE: imoogi-Cloze\n:END:\nBody\n")
    (imoogi-test--goto-heading "Malformed")
    (should (equal (imoogi-props-resolve-direction) "-->"))
    (should (equal (imoogi-props-resolve-incremental) "yes"))
    (imoogi-test--goto-heading "Cased")
    ;; case intact on every one of them. The case-insensitivity that
    ;; decides RECOGNITION belongs to the back end, not here.
    (should (equal (imoogi-props-resolve-incremental) "T"))
    (should (equal (imoogi-props-resolve-swift) "NIL"))
    (should (equal (imoogi-props-resolve-direction) "<->"))))

;; A property absent at all three levels resolves to no value, which is the
;; wire's null.
(ert-deftest imoogi-props-test-card-options-absent-resolve-to-no-value ()
  (imoogi-test--with-org-buffer
      "* Bare\n:PROPERTIES:\n:ANKI_NOTE_TYPE: imoogi-Cloze\n:END:\nBody\n"
    (imoogi-test--goto-heading "Bare")
    (should (null (imoogi-props-resolve-direction)))
    (should (null (imoogi-props-resolve-incremental)))
    (should (null (imoogi-props-resolve-swift)))))

(provide 'imoogi-props-test)
;;; imoogi-props-test.el ends here
