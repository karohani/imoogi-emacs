;;; imoogi-error.el --- Code -> user-facing message and remediation table -*- lexical-binding: t; -*-

;;; Commentary:

;; plan.md D-5's error taxonomy: every failure crossing the front end /
;; back end boundary is a typed code.  The Go binary emits codes in
;; response.errors[].code; this file is the ONLY component that turns a
;; code into prose a user reads (REQ-018).  No caller of
;; `imoogi-error-message' may show a raw Go stack trace, a raw HTTP
;; transport error, an Emacs backtrace, or a bare process exit code --
;; every message here names the missing prerequisite or the problem and
;; states a corrective action instead.
;;
;; "A code with no table entry is itself a defect" (plan.md D-5) is
;; mechanically enforced by test/imoogi-error-test.el's completeness
;; test (acceptance.md SS D.8), not merely asserted here.

;;; Code:

(defconst imoogi-error-table
  '(("binary_not_found" .
     "The imoogi-anki binary was not found at the configured path. Build it with `make build-anki' from the imoogi-emacs checkout, then make sure it is on `exec-path' (or point `imoogi-binary-path' at it).")
    ("binary_incompatible" .
     "The imoogi-anki binary speaks a different protocol version than this package expects. Rebuild it from this checkout with `make build-anki'.")
    ("sync_root_unset" .
     "No sync root is configured. Run `imoogi-anki-setup' and choose the directory imoogi should scan for sync targets.")
    ("anki_unreachable" .
     "Anki does not appear to be running -- nothing answered at the configured AnkiConnect address. Start Anki and run the sync again.")
    ("ankiconnect_missing" .
     "Anki is running, but the AnkiConnect add-on does not appear to be installed, or is not responding. Install the AnkiConnect add-on from Anki's add-on manager, restart Anki, and run the sync again.")
    ("ankiconnect_error" .
     "AnkiConnect reported an error for one of imoogi's requests. Check that Anki and the AnkiConnect add-on are both up to date, and try again.")
    ("org_parse_error" .
     "One entry's body could not be parsed as Org text, so it was skipped. Check that entry's content for malformed markup.")
    ("cloze_marker_missing" .
     "A Cloze entry has no {{cN::...}} marker anywhere in its body, so no card was created for it. Add at least one cloze marker to the entry, or change its ANKI_NOTE_TYPE to Basic.")
    ("deck_create_failed" .
     "The target Anki deck could not be created. Check the deck name for characters Anki rejects, and confirm Anki is responsive.")
    ("deck_move_failed" .
     "This note's cards could not be moved to the resolved deck. Check that the deck still exists in Anki and that Anki is responsive.")
    ("note_id_unknown" .
     "This entry's ANKI_NOTE_ID names a note that no longer exists in the collection. imoogi will treat it as unsynchronized and create a new note on the next run.")
    ("note_type_change_unsupported" .
     "This entry's ANKI_NOTE_TYPE has changed since it was last synced, and imoogi does not convert an existing note from one type to another. Delete the note in Anki, remove its ANKI_NOTE_ID property from the Org heading, then sync again to recreate it under the new type.")
    ("note_field_missing" .
     "This heading's note type has no field to put the rendered content in -- most often because the note type's fields were renamed in Anki. The message names the note type and the fields it actually has. Either rename the field back in Anki (Tools -> Manage Note Types -> Fields), or point the heading's ANKI_NOTE_TYPE at a note type that carries the expected fields. Nothing was written for this heading.")
    ("note_id_duplicated" .
     "Two or more headings carry the same ANKI_NOTE_ID -- usually because a heading was copied and pasted. One identifier means one Anki note, so imoogi cannot tell which heading owns it and wrote nothing for any of them. Delete the ANKI_NOTE_ID property from every copy except the one that should keep the existing note; the copies get their own notes on the next sync.")
    ("state_unreadable" .
     "imoogi's registry file exists but could not be read or parsed. Deletion is disabled until this is resolved. Check the registry file for corruption, or restore it from a backup.")
    ("delete_suppressed" .
     "Deletion was suppressed for this run because a safety check could not be completed -- either the scan of the sync root was incomplete, or Anki could not confirm the notes that were candidates for removal. Nothing was deleted.")
    ("delete_candidate_unowned" .
     "A note that looked like an orphan was skipped rather than deleted, because its content in Anki no longer matches what imoogi last recorded for it. Review it by hand if it should be removed."))
  "plan.md D-5's code -> user-facing message table (REQ-018).

Every value names the missing prerequisite or the problem and states a
corrective action.  No value contains a Go stack trace, a raw HTTP
transport error, an Emacs backtrace, or a bare process exit code.")

(defun imoogi-error-message (code)
  "Return the user-facing message for CODE.

When CODE has no table entry, return a generic fallback naming CODE
itself rather than signal an error -- per plan.md D-5, a code with no
table entry is a defect in this table, not a case callers must guard
against."
  (or (cdr (assoc code imoogi-error-table))
      (format "imoogi reported an error code with no message on file (%s). This is a gap in imoogi's own error table; please report it."
              code)))

(provide 'imoogi-error)
;;; imoogi-error.el ends here
