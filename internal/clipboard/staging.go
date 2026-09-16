package clipboard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const manifestFilename = "manifest.json"

type Store struct {
	Base string
	Now  func() time.Time
}

type Lease struct {
	PID          int       `json:"pid"`
	ProcessStart string    `json:"process_start"`
	RenewedAt    time.Time `json:"renewed_at"`
}

const DefaultRetention = 24 * time.Hour

func (s Store) Create(identity Identity, owner Owner, assets []Asset, created []string) (Manifest, string, error) {
	if err := identity.Validate(); err != nil {
		return Manifest{}, "", err
	}
	now := s.now()
	id, err := randomHex(16)
	if err != nil {
		return Manifest{}, "", err
	}
	token, err := randomHex(32)
	if err != nil {
		return Manifest{}, "", err
	}
	directory := s.transactionDir(identity, id)
	manifest := Manifest{
		ProtocolVersion:  ProtocolVersion,
		TransactionID:    id,
		TransactionToken: token,
		Identity:         identity,
		Owner:            owner,
		State:            StateStaged,
		Assets:           assets,
		Rewrite:          map[string]string{},
		CreatedFiles:     append([]string(nil), created...),
		StagingFiles:     append([]string(nil), created...),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := writeJSONAtomic(filepath.Join(directory, manifestFilename), manifest); err != nil {
		return manifest, directory, err
	}
	return manifest, directory, nil
}

func (s Store) RecordRecovery(directory string, manifest Manifest, orphans []string) error {
	manifest.State = StateNeedsReconciliation
	manifest.Orphans = append([]string(nil), orphans...)
	manifest.UpdatedAt = s.now()
	return writeJSONAtomic(filepath.Join(directory, manifestFilename), manifest)
}

func (s Store) Load(directory string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(directory, manifestFilename))
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	if manifest.ProtocolVersion != ProtocolVersion {
		return Manifest{}, fmt.Errorf("manifest protocol version %d is unsupported", manifest.ProtocolVersion)
	}
	return manifest, nil
}

func (s Store) Transition(directory, token string, next TransactionState, rewrite map[string]string, created, orphans []string) (Manifest, error) {
	manifest, err := s.Load(directory)
	if err != nil {
		return Manifest{}, err
	}
	if token == "" || token != manifest.TransactionToken {
		return Manifest{}, errors.New("transaction token mismatch")
	}
	if !allowedTransition(manifest.State, next) {
		return Manifest{}, fmt.Errorf("invalid transaction transition %s -> %s", manifest.State, next)
	}
	manifest.State = next
	if rewrite != nil {
		manifest.Rewrite = rewrite
	}
	if created != nil {
		manifest.CreatedFiles = created
	}
	if orphans != nil {
		manifest.Orphans = orphans
	}
	manifest.UpdatedAt = s.now()
	if err := writeJSONAtomic(filepath.Join(directory, manifestFilename), manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (s Store) Prepare(directory, token string, owner Owner, assets []Asset, rewrite map[string]string, created []string) (Manifest, error) {
	manifest, err := s.Load(directory)
	if err != nil {
		return Manifest{}, err
	}
	if token == "" || token != manifest.TransactionToken {
		return Manifest{}, errors.New("transaction token mismatch")
	}
	if !allowedTransition(manifest.State, StatePrepared) {
		return Manifest{}, fmt.Errorf("invalid transaction transition %s -> %s", manifest.State, StatePrepared)
	}
	manifest.Owner = owner
	manifest.Assets = assets
	manifest.Rewrite = rewrite
	manifest.CreatedFiles = created
	manifest.State = StatePrepared
	manifest.UpdatedAt = s.now()
	if err := writeJSONAtomic(filepath.Join(directory, manifestFilename), manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func allowedTransition(current, next TransactionState) bool {
	if current == next {
		return true
	}
	switch current {
	case StateStaged:
		return next == StatePrepared || next == StateAborted || next == StateNeedsReconciliation
	case StatePrepared:
		return next == StateDocumentSaved || next == StateAborted || next == StateNeedsReconciliation
	case StateDocumentSaved:
		return next == StateCommitted || next == StateNeedsReconciliation
	case StateNeedsReconciliation:
		return next == StateAborted || next == StateCommitted
	case StateCommitted, StateAborted:
		return false
	default:
		return false
	}
}

func (s Store) RenewLease(identity Identity, lease Lease) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	lease.RenewedAt = s.now()
	directory := filepath.Join(s.Base, "sessions", safeComponent(identity.Session), safeComponent(identity.Buffer))
	return writeJSONAtomic(filepath.Join(directory, "lease.json"), lease)
}

func (s Store) Prune(retention time.Duration) (int, error) {
	if retention <= 0 {
		retention = DefaultRetention
	}
	root := filepath.Join(s.Base, "sessions")
	sessions, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, session := range sessions {
		if !session.IsDir() {
			continue
		}
		buffers, err := os.ReadDir(filepath.Join(root, session.Name()))
		if err != nil {
			return removed, err
		}
		for _, buffer := range buffers {
			if !buffer.IsDir() {
				continue
			}
			directory := filepath.Join(root, session.Name(), buffer.Name())
			lease, err := readLease(filepath.Join(directory, "lease.json"))
			if err != nil {
				// Without a valid PID/start identity, age alone cannot prove that
				// an Emacs session is dead. Keep the directory for explicit recovery.
				continue
			}
			if s.now().Sub(lease.RenewedAt) < retention {
				continue
			}
			start, alive, err := processIdentity(lease.PID)
			if err != nil || (alive && start == lease.ProcessStart) {
				continue
			}
			if err := os.RemoveAll(directory); err != nil {
				return removed, err
			}
			removed++
		}
	}
	return removed, nil
}

func readLease(path string) (Lease, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Lease{}, err
	}
	var lease Lease
	if err := json.Unmarshal(data, &lease); err != nil {
		return Lease{}, err
	}
	if lease.PID <= 0 || lease.ProcessStart == "" || lease.RenewedAt.IsZero() {
		return Lease{}, errors.New("invalid lease")
	}
	return lease, nil
}

func (s Store) transactionDir(identity Identity, transactionID string) string {
	return filepath.Join(s.Base, "sessions", safeComponent(identity.Session), safeComponent(identity.Buffer), fmt.Sprint(identity.Generation), "transactions", transactionID)
}

func (s Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".json-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}

func randomHex(bytes int) (string, error) {
	value := make([]byte, bytes)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
