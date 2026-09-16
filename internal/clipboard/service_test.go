package clipboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeAdapter struct {
	inspection Inspection
	payload    ClipboardPayload
	err        error
}

func (f fakeAdapter) Inspect(context.Context) (Inspection, error)            { return f.inspection, f.err }
func (f fakeAdapter) Read(context.Context, string) (ClipboardPayload, error) { return f.payload, f.err }

func TestServiceInspect(t *testing.T) {
	service := Service{Adapter: fakeAdapter{inspection: Inspection{ClipboardID: "id:1", Formats: []string{"public.png"}, Kind: "image", Capability: "native"}}}
	response := service.Handle(context.Background(), Request{ProtocolVersion: 1, Operation: OperationInspect, CorrelationID: "c"})
	if response.Status != "ok" || response.ClipboardID != "id:1" || response.Kind != "image" {
		t.Fatalf("response = %#v", response)
	}
}

func TestServiceUnsupportedCapability(t *testing.T) {
	service := Service{Adapter: fakeAdapter{err: errors.New("unsupported-capability")}}
	response := service.Handle(context.Background(), Request{ProtocolVersion: 1, Operation: OperationInspect, CorrelationID: "c"})
	if response.Code != CodeUnsupportedCapability {
		t.Fatalf("response = %#v", response)
	}
}

func TestServiceCheckpointRenewsValidatedLease(t *testing.T) {
	staging := t.TempDir()
	service := Service{StagingBase: staging, Adapter: fakeAdapter{inspection: Inspection{ClipboardID: "id:1", Kind: "text"}}}
	response := service.Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationCheckpoint, CorrelationID: "lease",
		Identity: &Identity{Session: "s", Buffer: "b", Generation: 1},
		Lease:    &Lease{PID: os.Getpid(), ProcessStart: "client-assertion-is-not-trusted"},
	})
	if response.Status != "ok" {
		t.Fatalf("response = %#v", response)
	}
	lease, err := readLease(filepath.Join(staging, "sessions", "s", "b", "lease.json"))
	if err != nil {
		t.Fatal(err)
	}
	want, alive, err := processIdentity(os.Getpid())
	if err != nil || !alive || lease.ProcessStart != want {
		t.Fatalf("lease = %#v, want process start %q", lease, want)
	}
}

func TestServiceImport(t *testing.T) {
	source := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	documentRoot := t.TempDir()
	document := filepath.Join(documentRoot, "a.org")
	service := Service{StagingBase: t.TempDir()}
	response := service.Handle(context.Background(), Request{
		ProtocolVersion: 1,
		Operation:       OperationImport,
		CorrelationID:   "c",
		Identity:        &Identity{Session: "s", Buffer: "b", Generation: 1},
		Owner:           &Owner{Kind: OwnerStandalone, Document: document, Root: filepath.Join(documentRoot, "a.assets")},
		Paths:           []string{source},
	})
	if response.Status != "ok" || len(response.Assets) != 1 || response.State != StateCommitted || response.TransactionID != "" {
		t.Fatalf("response = %#v", response)
	}
}

func TestServiceRejectsForgedStagingIdentity(t *testing.T) {
	source := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	response := (Service{StagingBase: t.TempDir()}).Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationImport, CorrelationID: "forged",
		Identity: &Identity{Session: "s", Buffer: "b", Generation: 1},
		Owner:    &Owner{Kind: OwnerStaging, Session: "other", Buffer: "b", Generation: 1},
		Paths:    []string{source},
	})
	if response.Code != CodeOwnerRejected {
		t.Fatalf("response = %#v", response)
	}
}

func TestServiceFinalizesStagingIntoStandaloneOwner(t *testing.T) {
	staging := t.TempDir()
	source := filepath.Join(t.TempDir(), "screen.png")
	if err := os.WriteFile(source, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	identity := &Identity{Session: "s", Buffer: "b", Generation: 1}
	service := Service{StagingBase: staging}
	imported := service.Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationImport, CorrelationID: "import",
		Identity: identity,
		Owner:    &Owner{Kind: OwnerStaging, Session: "s", Buffer: "b", Generation: 1},
		Paths:    []string{source},
	})
	if imported.Status != "ok" {
		t.Fatalf("import = %#v", imported)
	}
	documentRoot := t.TempDir()
	document := filepath.Join(documentRoot, "note.org")
	finalized := service.Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationFinalize, CorrelationID: "finalize",
		Identity: identity, TransactionID: imported.TransactionID,
		TransactionToken: imported.TransactionToken,
		Owner: &Owner{Kind: OwnerStandalone, Document: document,
			Root: filepath.Join(documentRoot, "note.assets")},
	})
	if err := os.WriteFile(document, []byte(finalized.Assets[0].RelativePath), 0o644); err != nil {
		t.Fatal(err)
	}
	if finalized.Status != "ok" || finalized.State != StatePrepared || len(finalized.Assets) != 1 {
		t.Fatalf("finalize = %#v", finalized)
	}
	canonicalRoot, err := filepath.EvalSymlinks(documentRoot)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(finalized.Assets[0].Path) != filepath.Join(canonicalRoot, "note.assets") {
		t.Fatalf("final path = %q", finalized.Assets[0].Path)
	}
}

func TestServiceAbortCannotDeleteCommittedAssets(t *testing.T) {
	staging := t.TempDir()
	source := filepath.Join(t.TempDir(), "screen.png")
	if err := os.WriteFile(source, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	identity := &Identity{Session: "s", Buffer: "b", Generation: 1}
	service := Service{StagingBase: staging}
	imported := service.Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationImport, CorrelationID: "import",
		Identity: identity, Owner: &Owner{Kind: OwnerStaging, Session: "s", Buffer: "b", Generation: 1}, Paths: []string{source},
	})
	documentRoot := t.TempDir()
	finalized := service.Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationFinalize, CorrelationID: "finalize",
		Identity: identity, TransactionID: imported.TransactionID, TransactionToken: imported.TransactionToken,
		Owner: &Owner{Kind: OwnerStandalone, Document: filepath.Join(documentRoot, "note.org"), Root: filepath.Join(documentRoot, "note.assets")},
	})
	document := filepath.Join(documentRoot, "note.org")
	if err := os.WriteFile(document, []byte(finalized.Assets[0].RelativePath), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []Operation{OperationDocumentSaved, OperationCommit} {
		response := service.Handle(context.Background(), Request{
			ProtocolVersion: 1, Operation: operation, CorrelationID: string(operation), Identity: identity,
			TransactionID: imported.TransactionID, TransactionToken: imported.TransactionToken,
		})
		if response.Status != "ok" {
			t.Fatalf("%s = %#v", operation, response)
		}
	}
	retried := service.Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationFinalize, CorrelationID: "finalize-retry", Identity: identity,
		TransactionID: imported.TransactionID, TransactionToken: imported.TransactionToken,
		Owner: &Owner{Kind: OwnerStandalone, Document: document, Root: filepath.Join(documentRoot, "note.assets")},
	})
	if retried.Status != "ok" || retried.State != StateCommitted || len(retried.Assets) != 1 || retried.Assets[0].Path != finalized.Assets[0].Path {
		t.Fatalf("committed finalize retry = %#v", retried)
	}
	if _, err := os.Stat(imported.Assets[0].Path); !os.IsNotExist(err) {
		t.Fatalf("staging asset remains after commit: %v", err)
	}
	aborted := service.Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationAbort, CorrelationID: "abort", Identity: identity,
		TransactionID: imported.TransactionID, TransactionToken: imported.TransactionToken,
	})
	if aborted.Code != CodeTransactionConflict {
		t.Fatalf("abort = %#v", aborted)
	}
	if _, err := os.Stat(finalized.Assets[0].Path); err != nil {
		t.Fatalf("committed asset was removed: %v", err)
	}
}

func TestServiceReconcileRetriesCleanupAndCommitsVerifiedDocument(t *testing.T) {
	staging := t.TempDir()
	identity := Identity{Session: "s", Buffer: "b", Generation: 1}
	store := Store{Base: staging}
	documentRoot := t.TempDir()
	document := filepath.Join(documentRoot, "note.org")
	assetPath := filepath.Join(documentRoot, "note.assets", "screen.png")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte("png")
	if err := os.WriteFile(assetPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	asset := Asset{ID: "asset", Path: assetPath, RelativePath: "note.assets/screen.png", Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])}
	if err := os.WriteFile(document, []byte(asset.RelativePath), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, directory, err := store.Create(identity, Owner{Kind: OwnerStaging, Session: "s", Buffer: "b", Generation: 1}, []Asset{asset}, nil)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err = store.Prepare(directory, manifest.TransactionToken,
		Owner{Kind: OwnerStandalone, Document: document, Root: filepath.Dir(assetPath)}, []Asset{asset}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err = store.Transition(directory, manifest.TransactionToken, StateDocumentSaved, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err = store.Transition(directory, manifest.TransactionToken, StateNeedsReconciliation, nil, nil, []string{filepath.Join(staging, "already-gone")})
	if err != nil {
		t.Fatal(err)
	}
	finalized := (Service{StagingBase: staging}).Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationFinalize, CorrelationID: "finalize-needs", Identity: &identity,
		TransactionID: manifest.TransactionID, TransactionToken: manifest.TransactionToken,
		Owner: &manifest.Owner,
	})
	if finalized.Status != "ok" || finalized.State != StateNeedsReconciliation {
		t.Fatalf("finalize reconciliation state = %#v", finalized)
	}
	response := (Service{StagingBase: staging}).Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationReconcile, CorrelationID: "reconcile", Identity: &identity,
		TransactionID: manifest.TransactionID, TransactionToken: manifest.TransactionToken,
	})
	if response.Status != "ok" || response.State != StateCommitted {
		t.Fatalf("reconcile = %#v", response)
	}
}

func TestServiceFinalizeKeepsDistinctStableIDsForEqualContent(t *testing.T) {
	staging := t.TempDir()
	sources := t.TempDir()
	paths := []string{filepath.Join(sources, "one.png"), filepath.Join(sources, "two.png")}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("same"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	identity := &Identity{Session: "s", Buffer: "b", Generation: 1}
	service := Service{StagingBase: staging}
	imported := service.Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationImport, CorrelationID: "import", Identity: identity,
		Owner: &Owner{Kind: OwnerStaging, Session: "s", Buffer: "b", Generation: 1}, Paths: paths,
	})
	if imported.Assets[0].ID == imported.Assets[1].ID {
		t.Fatal("equal content reused an opaque asset id")
	}
	root := t.TempDir()
	finalized := service.Handle(context.Background(), Request{
		ProtocolVersion: 1, Operation: OperationFinalize, CorrelationID: "finalize", Identity: identity,
		TransactionID: imported.TransactionID, TransactionToken: imported.TransactionToken,
		Owner: &Owner{Kind: OwnerStandalone, Document: filepath.Join(root, "note.org"), Root: filepath.Join(root, "note.assets")},
	})
	for i := range imported.Assets {
		if finalized.Assets[i].ID != imported.Assets[i].ID {
			t.Fatalf("asset %d id changed", i)
		}
	}
}
