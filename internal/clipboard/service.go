package clipboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	Adapter     Adapter
	StagingBase string
	Version     string
}

func (s Service) Handle(ctx context.Context, request Request) Response {
	if err := request.Validate(); err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, err.Error())
	}
	switch request.Operation {
	case OperationVersion:
		response := Success(request.Operation, request.CorrelationID)
		response.Version = s.Version
		return response
	case OperationInspect:
		return s.inspect(ctx, request)
	case OperationCheckpoint:
		return s.checkpoint(ctx, request)
	case OperationImport:
		return s.importPaths(request, request.Paths)
	case OperationPaste:
		return s.paste(ctx, request)
	case OperationFinalize:
		return s.finalize(request)
	case OperationDocumentSaved:
		return s.transition(request, StateDocumentSaved)
	case OperationCommit:
		return s.commit(request)
	case OperationAbort:
		return s.abort(request)
	case OperationReconcile:
		return s.reconcile(request)
	case OperationPrune:
		return s.prune(request)
	default:
		return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, "unsupported operation")
	}
}

func (s Service) checkpoint(ctx context.Context, request Request) Response {
	if request.Identity != nil || request.Lease != nil {
		if request.Identity == nil || request.Lease == nil {
			return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, "identity and lease must be provided together")
		}
		start, alive, err := processIdentity(request.Lease.PID)
		if err != nil || !alive {
			return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, "lease process cannot be validated")
		}
		lease := *request.Lease
		lease.ProcessStart = start
		if err := (Store{Base: s.stagingBase()}).RenewLease(*request.Identity, lease); err != nil {
			return Failure(request.Operation, request.CorrelationID, CodeInternal, err.Error())
		}
	}
	return s.inspect(ctx, request)
}

func (s Service) prune(request Request) Response {
	retention := time.Duration(request.RetentionSeconds) * time.Second
	removed, err := (Store{Base: s.stagingBase()}).Prune(retention)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeInternal, err.Error())
	}
	response := Success(request.Operation, request.CorrelationID)
	response.Message = fmt.Sprintf("pruned %d expired session(s)", removed)
	return response
}

func (s Service) commit(request Request) Response {
	directory, err := s.transactionDirectory(request)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, err.Error())
	}
	store := Store{Base: s.stagingBase()}
	manifest, err := store.Load(directory)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, err.Error())
	}
	if request.TransactionToken == "" || request.TransactionToken != manifest.TransactionToken {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, "transaction token mismatch")
	}
	if manifest.State == StateCommitted {
		return responseFromManifest(request, manifest)
	}
	if manifest.State != StateDocumentSaved && manifest.State != StateNeedsReconciliation {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict,
			fmt.Sprintf("cannot commit transaction in state %s", manifest.State))
	}
	if manifest.State == StateNeedsReconciliation {
		return Failure(request.Operation, request.CorrelationID, CodeNeedsReconciliation,
			"transaction requires explicit recovery before it can be committed")
	}
	if err := verifyPreparedManifest(manifest); err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeNeedsReconciliation, err.Error())
	}
	if orphans := compensate(manifest.StagingFiles); len(orphans) != 0 {
		manifest, transitionErr := store.Transition(directory, request.TransactionToken,
			StateNeedsReconciliation, nil, nil, orphans)
		if transitionErr != nil {
			return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, transitionErr.Error())
		}
		response := responseFromManifest(request, manifest)
		response.Status = "error"
		response.Code = CodeNeedsReconciliation
		response.Message = "failed to remove one or more staging files"
		response.Orphans = orphans
		return response
	}
	manifest, err = store.Transition(directory, request.TransactionToken, StateCommitted, nil, nil, []string{})
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, err.Error())
	}
	return responseFromManifest(request, manifest)
}

func (s Service) inspect(ctx context.Context, request Request) Response {
	inspection, err := s.adapter().Inspect(ctx)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, classifyError(err), err.Error())
	}
	response := Success(request.Operation, request.CorrelationID)
	response.Kind = inspection.Kind
	response.Capability = inspection.Capability
	response.ClipboardID = inspection.ClipboardID
	response.Formats = inspection.Formats
	return response
}

func (s Service) paste(ctx context.Context, request Request) Response {
	payload, err := s.adapter().Read(ctx, request.ExpectedClipboardID)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, classifyError(err), err.Error())
	}
	if payload.Inspection.Kind == "text" {
		response := Success(request.Operation, request.CorrelationID)
		response.Kind = "text"
		return response
	}
	paths := payload.Paths
	if len(payload.Image) != 0 {
		temporary, err := os.CreateTemp("", "imoogi-clipboard-*.png")
		if err != nil {
			return Failure(request.Operation, request.CorrelationID, CodeInternal, err.Error())
		}
		name := temporary.Name()
		defer os.Remove(name)
		if _, err := temporary.Write(payload.Image); err != nil {
			temporary.Close()
			return Failure(request.Operation, request.CorrelationID, CodeInternal, err.Error())
		}
		if err := temporary.Close(); err != nil {
			return Failure(request.Operation, request.CorrelationID, CodeInternal, err.Error())
		}
		paths = []string{name}
	}
	return s.importPaths(request, paths)
}

func (s Service) importPaths(request Request, paths []string) Response {
	if request.Owner == nil || request.Identity == nil {
		return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, "owner and identity are required")
	}
	resolved, err := ResolveOwner(*request.Owner, s.stagingBase())
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeOwnerRejected, err.Error())
	}
	if request.Owner.Kind == OwnerStaging &&
		(request.Owner.Session != request.Identity.Session ||
			request.Owner.Buffer != request.Identity.Buffer ||
			request.Owner.Generation != request.Identity.Generation) {
		return Failure(request.Operation, request.CorrelationID, CodeOwnerRejected, "staging owner identity mismatch")
	}
	result, err := (Publisher{Limits: limitsFor(request), ContainmentRoot: resolved.ContainmentRoot}).Publish(paths, resolved.AssetRoot)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, classifyError(err), err.Error())
	}
	if request.Owner.Kind != OwnerStaging {
		response := Success(request.Operation, request.CorrelationID)
		response.Kind = "assets"
		response.Assets = result.Assets
		response.State = StateCommitted
		return response
	}
	store := Store{Base: s.stagingBase()}
	manifest, directory, err := store.Create(*request.Identity, *request.Owner, result.Assets, result.CreatedFiles)
	if err != nil {
		orphans := compensate(result.CreatedFiles)
		if len(orphans) != 0 {
			recoveryErr := store.RecordRecovery(directory, manifest, orphans)
			return Failure(request.Operation, request.CorrelationID, CodeNeedsReconciliation,
				fmt.Sprintf("persist transaction manifest: %v; orphans=%v; recovery_manifest=%v", err, orphans, recoveryErr))
		}
		return Failure(request.Operation, request.CorrelationID, CodeInternal, err.Error())
	}
	response := Success(request.Operation, request.CorrelationID)
	response.Kind = "assets"
	response.Assets = result.Assets
	response.TransactionToken = manifest.TransactionToken
	response.TransactionID = manifest.TransactionID
	response.State = manifest.State
	return response
}

func (s Service) transition(request Request, state TransactionState) Response {
	directory, err := s.transactionDirectory(request)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, err.Error())
	}
	manifest, err := (Store{Base: s.stagingBase()}).Transition(directory, request.TransactionToken, state, request.Rewrite, nil, nil)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, err.Error())
	}
	response := Success(request.Operation, request.CorrelationID)
	response.State = manifest.State
	response.TransactionToken = manifest.TransactionToken
	response.TransactionID = manifest.TransactionID
	return response
}

func (s Service) finalize(request Request) Response {
	if request.Owner == nil {
		return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, "destination owner is required")
	}
	directory, err := s.transactionDirectory(request)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, err.Error())
	}
	store := Store{Base: s.stagingBase()}
	manifest, err := store.Load(directory)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, err.Error())
	}
	if request.TransactionToken == "" || request.TransactionToken != manifest.TransactionToken {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, "transaction token mismatch")
	}
	if manifest.State == StatePrepared || manifest.State == StateDocumentSaved ||
		manifest.State == StateNeedsReconciliation || manifest.State == StateCommitted {
		return responseFromManifest(request, manifest)
	}
	resolved, err := ResolveOwner(*request.Owner, s.stagingBase())
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeOwnerRejected, err.Error())
	}
	paths := make([]string, 0, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		paths = append(paths, asset.Path)
	}
	result, err := (Publisher{Limits: limitsFor(request), ContainmentRoot: resolved.ContainmentRoot}).Publish(paths, resolved.AssetRoot)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, classifyError(err), err.Error())
	}
	rewrite := make(map[string]string, len(result.Assets))
	for index := range result.Assets {
		result.Assets[index].ID = manifest.Assets[index].ID
		asset := result.Assets[index]
		rewrite["imoogi-asset:"+asset.ID] = asset.Path
	}
	manifest, err = store.Prepare(directory, request.TransactionToken, *request.Owner,
		result.Assets, rewrite, result.CreatedFiles)
	if err != nil {
		orphans := compensate(result.CreatedFiles)
		var transitionErr error
		if len(orphans) != 0 {
			_, transitionErr = store.Transition(directory, request.TransactionToken,
				StateNeedsReconciliation, nil, result.CreatedFiles, orphans)
		}
		return Failure(request.Operation, request.CorrelationID, CodeNeedsReconciliation,
			fmt.Sprintf("persist prepared state: %v; compensated=%t; orphans=%v; recovery_manifest=%v",
				err, len(orphans) == 0, orphans, transitionErr))
	}
	return responseFromManifest(request, manifest)
}

func responseFromManifest(request Request, manifest Manifest) Response {
	response := Success(request.Operation, request.CorrelationID)
	response.State = manifest.State
	response.TransactionID = manifest.TransactionID
	response.TransactionToken = manifest.TransactionToken
	response.Assets = manifest.Assets
	return response
}

func (s Service) reconcile(request Request) Response {
	directory, err := s.transactionDirectory(request)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, err.Error())
	}
	manifest, err := (Store{Base: s.stagingBase()}).Load(directory)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, err.Error())
	}
	if request.TransactionToken == "" || request.TransactionToken != manifest.TransactionToken {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, "transaction token mismatch")
	}
	if manifest.State == StatePrepared || manifest.State == StateDocumentSaved || manifest.State == StateNeedsReconciliation {
		if err := verifyPreparedManifest(manifest); err != nil {
			response := Failure(request.Operation, request.CorrelationID, CodeNeedsReconciliation, err.Error())
			response.State = manifest.State
			response.TransactionID = manifest.TransactionID
			response.TransactionToken = manifest.TransactionToken
			response.Orphans = manifest.Orphans
			return response
		}
	}
	if manifest.State == StateNeedsReconciliation {
		cleanup := manifest.Orphans
		if len(cleanup) == 0 {
			cleanup = manifest.StagingFiles
		}
		if orphans := compensate(cleanup); len(orphans) != 0 {
			manifest, _ = (Store{Base: s.stagingBase()}).Transition(directory,
				request.TransactionToken, StateNeedsReconciliation, nil, nil, orphans)
			response := Failure(request.Operation, request.CorrelationID, CodeNeedsReconciliation,
				"failed to remove one or more staging files")
			response.State = manifest.State
			response.TransactionID = manifest.TransactionID
			response.TransactionToken = manifest.TransactionToken
			response.Orphans = orphans
			return response
		}
		manifest, err = (Store{Base: s.stagingBase()}).Transition(directory,
			request.TransactionToken, StateCommitted, nil, nil, []string{})
		if err != nil {
			return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, err.Error())
		}
	}
	response := Success(request.Operation, request.CorrelationID)
	response.State = manifest.State
	response.TransactionToken = manifest.TransactionToken
	response.TransactionID = manifest.TransactionID
	response.Assets = manifest.Assets
	response.Orphans = manifest.Orphans
	return response
}

func (s Service) abort(request Request) Response {
	directory, err := s.transactionDirectory(request)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeInvalidRequest, err.Error())
	}
	store := Store{Base: s.stagingBase()}
	manifest, err := store.Load(directory)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, err.Error())
	}
	if request.TransactionToken == "" || request.TransactionToken != manifest.TransactionToken {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, "transaction token mismatch")
	}
	if manifest.State == StateAborted {
		return responseFromManifest(request, manifest)
	}
	if manifest.State != StateStaged && manifest.State != StatePrepared && manifest.State != StateNeedsReconciliation {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict,
			fmt.Sprintf("cannot abort transaction in state %s", manifest.State))
	}
	created := append(append([]string(nil), manifest.CreatedFiles...), manifest.StagingFiles...)
	orphans := compensate(created)
	next := StateAborted
	if len(orphans) != 0 {
		next = StateNeedsReconciliation
	}
	manifest, err = store.Transition(directory, request.TransactionToken, next, nil, nil, orphans)
	if err != nil {
		return Failure(request.Operation, request.CorrelationID, CodeTransactionConflict, err.Error())
	}
	response := responseFromManifest(request, manifest)
	response.Orphans = orphans
	if len(orphans) != 0 {
		response.Status = "error"
		response.Code = CodeNeedsReconciliation
		response.Message = "failed to remove one or more transaction-created files"
	}
	return response
}

func verifyPreparedManifest(manifest Manifest) error {
	if manifest.Owner.Document == "" {
		return errors.New("prepared transaction has no destination document")
	}
	document, err := os.ReadFile(manifest.Owner.Document)
	if err != nil {
		return fmt.Errorf("read saved document: %w", err)
	}
	for _, asset := range manifest.Assets {
		data, err := os.ReadFile(asset.Path)
		if err != nil {
			return fmt.Errorf("read final asset %q: %w", asset.Path, err)
		}
		sum := sha256.Sum256(data)
		if int64(len(data)) != asset.Size || hex.EncodeToString(sum[:]) != asset.SHA256 {
			return fmt.Errorf("final asset verification failed for %q", asset.Path)
		}
		if !strings.Contains(string(document), asset.RelativePath) &&
			!strings.Contains(string(document), url.PathEscape(asset.RelativePath)) {
			return fmt.Errorf("saved document does not reference final asset %q", asset.RelativePath)
		}
	}
	return nil
}

func (s Service) transactionDirectory(request Request) (string, error) {
	if request.Identity == nil || request.TransactionID == "" {
		return "", errors.New("identity and transaction_id are required")
	}
	if err := request.Identity.Validate(); err != nil {
		return "", err
	}
	return (Store{Base: s.stagingBase()}).transactionDir(*request.Identity, safeComponent(request.TransactionID)), nil
}

func (s Service) adapter() Adapter {
	if s.Adapter != nil {
		return s.Adapter
	}
	return NewAdapter()
}

func (s Service) stagingBase() string {
	if s.StagingBase != "" {
		return s.StagingBase
	}
	return filepath.Join(os.TempDir(), "imoogi-clip")
}

func limitsFor(request Request) Limits {
	if request.Limits != nil {
		return *request.Limits
	}
	return DefaultLimits()
}

func classifyError(err error) ErrorCode {
	message := err.Error()
	for _, code := range []ErrorCode{CodeUnsupportedCapability, CodeClipboardChanged, CodeDirectoryRejected, CodeNonRegularRejected, CodePathEscape, CodeLimitExceeded, CodeNeedsReconciliation} {
		if strings.Contains(message, string(code)) {
			return code
		}
	}
	return CodeInternal
}
