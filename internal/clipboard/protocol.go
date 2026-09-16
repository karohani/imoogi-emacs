package clipboard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

const ProtocolVersion = 1

const (
	DefaultMaxFiles      = 32
	DefaultMaxFileBytes  = int64(256 << 20)
	DefaultMaxBatchBytes = int64(512 << 20)
)

type Operation string

const (
	OperationInspect       Operation = "inspect"
	OperationCheckpoint    Operation = "checkpoint"
	OperationPaste         Operation = "paste"
	OperationImport        Operation = "import"
	OperationFinalize      Operation = "finalize"
	OperationDocumentSaved Operation = "document-saved"
	OperationCommit        Operation = "commit"
	OperationAbort         Operation = "abort"
	OperationReconcile     Operation = "reconcile"
	OperationPrune         Operation = "prune"
	OperationVersion       Operation = "version"
)

type OwnerKind string

const (
	OwnerProjectNotes OwnerKind = "project_notes"
	OwnerStandalone   OwnerKind = "standalone"
	OwnerStaging      OwnerKind = "staging"
)

type TransactionState string

const (
	StateStaged              TransactionState = "STAGED"
	StatePrepared            TransactionState = "PREPARED"
	StateDocumentSaved       TransactionState = "DOCUMENT_SAVED"
	StateCommitted           TransactionState = "COMMITTED"
	StateAborted             TransactionState = "ABORTED"
	StateNeedsReconciliation TransactionState = "NEEDS_RECONCILIATION"
)

type ErrorCode string

const (
	CodeInvalidRequest        ErrorCode = "invalid-request"
	CodeUnsupportedVersion    ErrorCode = "unsupported-version"
	CodeUnsupportedCapability ErrorCode = "unsupported-capability"
	CodeClipboardChanged      ErrorCode = "clipboard-changed"
	CodeDirectoryRejected     ErrorCode = "directory-rejected"
	CodeNonRegularRejected    ErrorCode = "non-regular-rejected"
	CodeOwnerRejected         ErrorCode = "owner-rejected"
	CodePathEscape            ErrorCode = "path-escape"
	CodeLimitExceeded         ErrorCode = "limit-exceeded"
	CodeTransactionConflict   ErrorCode = "transaction-conflict"
	CodeNeedsReconciliation   ErrorCode = "needs-reconciliation"
	CodeInternal              ErrorCode = "internal"
)

type Limits struct {
	MaxFiles      int   `json:"max_files"`
	MaxFileBytes  int64 `json:"max_file_bytes"`
	MaxBatchBytes int64 `json:"max_batch_bytes"`
}

func DefaultLimits() Limits {
	return Limits{
		MaxFiles:      DefaultMaxFiles,
		MaxFileBytes:  DefaultMaxFileBytes,
		MaxBatchBytes: DefaultMaxBatchBytes,
	}
}

type Identity struct {
	Session    string `json:"session"`
	Buffer     string `json:"buffer"`
	Generation uint64 `json:"generation"`
}

func (i Identity) Validate() error {
	if i.Session == "" || i.Buffer == "" || i.Generation == 0 {
		return errors.New("session, buffer, and positive generation are required")
	}
	return nil
}

type Owner struct {
	Kind        OwnerKind `json:"kind"`
	Document    string    `json:"document,omitempty"`
	Root        string    `json:"root,omitempty"`
	RegistryKey string    `json:"registry_key,omitempty"`
	Session     string    `json:"session,omitempty"`
	Buffer      string    `json:"buffer,omitempty"`
	Generation  uint64    `json:"generation,omitempty"`
}

func (o Owner) ValidateShape() error {
	switch o.Kind {
	case OwnerProjectNotes:
		if o.Document == "" || o.Root == "" {
			return errors.New("project_notes requires document and root")
		}
	case OwnerStandalone:
		if o.Document == "" || o.Root == "" {
			return errors.New("standalone requires document and root")
		}
	case OwnerStaging:
		if err := (Identity{Session: o.Session, Buffer: o.Buffer, Generation: o.Generation}).Validate(); err != nil {
			return fmt.Errorf("staging owner: %w", err)
		}
	default:
		return fmt.Errorf("unknown owner kind %q", o.Kind)
	}
	return nil
}

type Request struct {
	ProtocolVersion     int               `json:"protocol_version"`
	Operation           Operation         `json:"operation"`
	CorrelationID       string            `json:"correlation_id"`
	Identity            *Identity         `json:"identity,omitempty"`
	Owner               *Owner            `json:"owner,omitempty"`
	ExpectedClipboardID string            `json:"expected_clipboard_id,omitempty"`
	ClipboardID         string            `json:"clipboard_id,omitempty"`
	KillGeneration      uint64            `json:"kill_generation,omitempty"`
	Paths               []string          `json:"paths,omitempty"`
	TransactionToken    string            `json:"transaction_token,omitempty"`
	TransactionID       string            `json:"transaction_id,omitempty"`
	Rewrite             map[string]string `json:"rewrite,omitempty"`
	Limits              *Limits           `json:"limits,omitempty"`
	Lease               *Lease            `json:"lease,omitempty"`
	RetentionSeconds    int64             `json:"retention_seconds,omitempty"`
}

func (r Request) Validate() error {
	if r.ProtocolVersion != ProtocolVersion {
		return fmt.Errorf("protocol version %d is unsupported", r.ProtocolVersion)
	}
	if r.CorrelationID == "" {
		return errors.New("correlation_id is required")
	}
	if !validOperation(r.Operation) {
		return fmt.Errorf("unknown operation %q", r.Operation)
	}
	if r.Owner != nil {
		if err := r.Owner.ValidateShape(); err != nil {
			return fmt.Errorf("owner: %w", err)
		}
	}
	return nil
}

func validOperation(operation Operation) bool {
	switch operation {
	case OperationInspect, OperationCheckpoint, OperationPaste, OperationImport,
		OperationFinalize, OperationDocumentSaved, OperationCommit, OperationAbort,
		OperationReconcile, OperationPrune, OperationVersion:
		return true
	default:
		return false
	}
}

type Asset struct {
	ID           string `json:"id"`
	Source       string `json:"source,omitempty"`
	Path         string `json:"path,omitempty"`
	RelativePath string `json:"relative_path,omitempty"`
	MIME         string `json:"mime"`
	SHA256       string `json:"sha256"`
	Size         int64  `json:"size"`
}

type Response struct {
	ProtocolVersion  int              `json:"protocol_version"`
	Operation        Operation        `json:"operation"`
	CorrelationID    string           `json:"correlation_id"`
	Status           string           `json:"status"`
	Code             ErrorCode        `json:"code,omitempty"`
	Message          string           `json:"message,omitempty"`
	Kind             string           `json:"kind,omitempty"`
	Capability       string           `json:"capability,omitempty"`
	ClipboardID      string           `json:"clipboard_id,omitempty"`
	Formats          []string         `json:"formats,omitempty"`
	Assets           []Asset          `json:"assets,omitempty"`
	InsertionText    string           `json:"insertion_text,omitempty"`
	TransactionToken string           `json:"transaction_token,omitempty"`
	TransactionID    string           `json:"transaction_id,omitempty"`
	State            TransactionState `json:"state,omitempty"`
	Orphans          []string         `json:"orphans,omitempty"`
	Version          string           `json:"version,omitempty"`
}

func Success(operation Operation, correlationID string) Response {
	return Response{
		ProtocolVersion: ProtocolVersion,
		Operation:       operation,
		CorrelationID:   correlationID,
		Status:          "ok",
	}
}

func Failure(operation Operation, correlationID string, code ErrorCode, message string) Response {
	return Response{
		ProtocolVersion: ProtocolVersion,
		Operation:       operation,
		CorrelationID:   correlationID,
		Status:          "error",
		Code:            code,
		Message:         message,
	}
}

type Manifest struct {
	ProtocolVersion  int               `json:"protocol_version"`
	TransactionID    string            `json:"transaction_id"`
	TransactionToken string            `json:"transaction_token"`
	Identity         Identity          `json:"identity"`
	Owner            Owner             `json:"owner"`
	State            TransactionState  `json:"state"`
	Assets           []Asset           `json:"assets"`
	Rewrite          map[string]string `json:"rewrite"`
	CreatedFiles     []string          `json:"created_files"`
	StagingFiles     []string          `json:"staging_files,omitempty"`
	Orphans          []string          `json:"orphans,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func DecodeRequest(data []byte) (Request, error) {
	var request Request
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return Request{}, fmt.Errorf("decode request: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Request{}, errors.New("decode request: multiple JSON values")
		}
		return Request{}, fmt.Errorf("decode request trailer: %w", err)
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}
