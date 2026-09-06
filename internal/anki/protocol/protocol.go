// Package protocol defines the JSON documents exchanged between the Emacs Lisp
// front end and the Go back end: one request on stdin, one response on stdout,
// per sync run.
//
// No field carries omitempty. A null note_id is the add trigger and a null deck
// is the default-deck fallback trigger, so a dropped key and a null value are
// not interchangeable on the wire.
package protocol

// Version is the wire-contract version this binary speaks. The front end sends
// its own version in every request; a mismatch is reported rather than parsed
// through, because the hand-built distribution model makes version skew routine.
const Version = 1

// Result actions reported per sync target, plus one per orphan this run deleted.
const (
	ActionAdded   = "added"
	ActionUpdated = "updated"
	ActionSkipped = "skipped"
	ActionDeleted = "deleted"
	ActionFailed  = "failed"

	// ActionMigrateCandidate is the card-styling SPEC's one new action value
	// (design.md §5.1, REQ-C-018). It is emitted ONLY by `migrate --dry-run`,
	// one per candidate entry, carrying that entry's EXISTING note
	// identifier — so the candidate count the confirmation prompt needs is
	// len(results) and no response field had to be added for it. The front
	// end counts it and does nothing else: it triggers no write-back and no
	// property edit.
	//
	// A successful migration does NOT use this value. It reports `added`
	// carrying the NEW identifier, which is the value the front end's
	// existing write-back already acts on.
	ActionMigrateCandidate = "migrate_candidate"
)

// Diagnostic codes the back end emits. Each has a matching entry in the front
// end's code-to-message table, which is the only component that renders prose
// (plan.md D-5). A code with no table entry is itself a defect (D-5's own
// rationale); this const block is the single source every code the Go binary
// can emit is drawn from, so cmd/imoogi and internal/planner never invent one
// ad hoc.
const (
	CodeBinaryIncompatible        = "binary_incompatible"
	CodeAnkiUnreachable           = "anki_unreachable"
	CodeAnkiConnectMissing        = "ankiconnect_missing"
	CodeAnkiConnectError          = "ankiconnect_error"
	CodeOrgParseError             = "org_parse_error"
	CodeClozeMarkerMissing        = "cloze_marker_missing"
	CodeDeckCreateFailed          = "deck_create_failed"
	CodeDeckMoveFailed            = "deck_move_failed"
	CodeNoteTypeChangeUnsupported = "note_type_change_unsupported"
	CodeNoteFieldMissing          = "note_field_missing"
	CodeNoteIDDuplicated          = "note_id_duplicated"
	CodeStateUnreadable           = "state_unreadable"
	CodeDeleteSuppressed          = "delete_suppressed"
	CodeDeleteCandidateUnowned    = "delete_candidate_unowned"

	// The card-styling SPEC's own codes (design.md §5). They are declared
	// with the client surface they describe rather than with the milestones
	// that first raise them, because the front end's code-to-message table
	// and its contract test are keyed on this constant set: a code added
	// later than its table entry, or earlier, breaks the pairing either way.
	CodeModelInstallFailed = "model_install_failed"
	CodeMediaFileNotFound  = "media_file_not_found"
	CodeMediaUploadFailed  = "media_upload_failed"
	CodeMigrationAddFailed = "migration_add_failed"
)

// Request is the document the front end writes to the binary's stdin.
type Request struct {
	ProtocolVersion int           `json:"protocol_version"`
	Config          Config        `json:"config"`
	Census          []CensusEntry `json:"census"`
	Entries         []Entry       `json:"entries"`
}

// Config carries the effective settings for one run. The binary reads no
// configuration file of its own and consults no environment for its behavior.
type Config struct {
	DefaultDeck     string   `json:"default_deck"`
	AnkiConnectURL  string   `json:"anki_connect_url"`
	RegistryPath    string   `json:"registry_path"`
	SyncRoot        string   `json:"sync_root"`
	ExcludePatterns []string `json:"exclude_patterns"`
	// ScanComplete is true only when every .org file under the root was read,
	// excluded files included. False suppresses the entire delete path.
	ScanComplete bool `json:"scan_complete"`
}

// CensusEntry records one ANKI_NOTE_ID occurrence found anywhere under the sync
// root, excluded files included. The slice is in scan order, which is what the
// first-occurrence rule for a duplicated identifier reads.
type CensusEntry struct {
	NoteID     int    `json:"note_id"`
	SourcePath string `json:"source_path"`
}

// Entry is one sync target: a heading carrying ANKI_NOTE_TYPE in its own drawer,
// in a file exclusion does not bar. Deck and tags arrive already resolved through
// the inheritance chain; the back end performs no inheritance of its own.
type Entry struct {
	// Key correlates a result or error back to this heading. It is unique within
	// one request and deliberately not durable across runs.
	Key string `json:"key"`
	// NoteID is nil when the heading carries no identifier yet.
	NoteID     *int   `json:"note_id"`
	NoteType   string `json:"note_type"`
	SourcePath string `json:"source_path"`
	// Deck is nil when the chain resolved to no value, which selects the
	// configured default deck.
	Deck  *string  `json:"deck"`
	Tags  []string `json:"tags"`
	Title string   `json:"title"`
	// Body is raw Org text. The back end renders it; the front end never does.
	Body string `json:"body"`
}

// InstallRequest is the document the front end writes to the install-models
// subcommand's stdin (spec.md §2 "Install request document", design.md §6).
// It is a NEW document rather than a field added to Request, which is what
// keeps REQ-C-018's wire-stability clause true: the sync request and response
// documents gain nothing and Version is unchanged.
//
// The response side is deliberately NOT new — install-models answers with the
// existing Response, one Result per note type, so the front end's existing
// response reader and diagnostic table need no second shape.
//
// The endpoint key is spelled anki_connect_url, matching Config's spelling of
// the same value since the parent SPEC. design.md §6 sketches it as
// ankiconnect_url; that is read as a typo rather than as a second spelling,
// because one value carrying two spellings on one wire is itself the defect.
type InstallRequest struct {
	ProtocolVersion int    `json:"protocol_version"`
	AnkiConnectURL  string `json:"anki_connect_url"`
	// UserCSS is the user stylesheet's contents, or the empty string when no
	// such file exists (REQ-C-008). The Go binary reads no stylesheet from
	// disk itself; the front end is the only reader.
	UserCSS string `json:"user_css"`
}

// Response is the document the binary writes to stdout.
type Response struct {
	ProtocolVersion int      `json:"protocol_version"`
	OK              bool     `json:"ok"`
	Results         []Result `json:"results"`
	Errors          []Error  `json:"errors"`
}

// Result is one processed sync target, or one orphan this run deleted.
type Result struct {
	// Key echoes the request entry's key, and is nil for a deleted result, which
	// has no corresponding request entry to echo one from.
	Key    *string `json:"key"`
	Action string  `json:"action"`
	NoteID *int    `json:"note_id"`
}

// Error is one diagnostic code that fired this run.
type Error struct {
	Code string `json:"code"`
	// Message is machine-oriented detail. The front end's own table, not this
	// string, is what reaches the user.
	Message string `json:"message"`
	// Key is nil for a run-level error and set for a per-entry one.
	Key *string `json:"key"`
}
