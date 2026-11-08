// Package store writes clips under the configured save root.
//
// Every filesystem operation goes through an *os.Root opened on the save
// root, so no path — including one that traverses a symlink — can resolve
// outside it. There are no string-prefix path checks; the Root is the
// security mechanism.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/karohani/obsidian-clipper/native-host/internal/protocol"
)

const (
	stateDir    = ".org-clipper"
	tmpDir      = stateDir + "/tmp"
	requestsDir = stateDir + "/requests"

	dirPerm  = 0o700
	filePerm = 0o600

	// LockTTL is how long an in-flight marker is honoured before it is
	// treated as left behind by a crashed host.
	LockTTL = 10 * time.Minute
	// maxLedgerBytes bounds how much of a ledger file is read back.
	maxLedgerBytes = protocol.MaxOutgoing
	maxSlugRunes   = 60
)

// ErrExists is returned when the final clip directory already exists.
var ErrExists = errors.New("destination already exists")

var (
	assetNameRe = regexp.MustCompile(`^[0-9a-f]{32}\.(png|jpg|webp)$`)
	errBadName  = errors.New("invalid name")
)

// Store is an open save root.
type Store struct {
	root *os.Root
}

// Open opens the save root. A missing root or a non-directory yields a
// protocol error with code root_unavailable.
func Open(rootPath string) (*Store, error) {
	r, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, &protocol.Error{Code: protocol.CodeRootUnavailable, Message: "save root is not available"}
	}
	return &Store{root: r}, nil
}

// Close releases the root.
func (s *Store) Close() { s.root.Close() }

// Slug derives a filesystem-safe, human-readable name from a title: NFC,
// lowercased, Unicode letters, digits and combining marks kept (Korean
// survives), every other run collapsed to a single "-", trimmed, at most 60
// runes. It can never contain "/", "\", "..", NUL, control characters or a
// leading ".", because only letters, digits, marks and "-" are ever emitted.
func Slug(title string) string {
	var out []rune
	dash := false
	for _, r := range norm.NFC.String(title) {
		keep := unicode.IsLetter(r) || unicode.IsDigit(r) || (unicode.IsMark(r) && len(out) > 0 && !dash)
		switch {
		case keep:
			out = append(out, unicode.ToLower(r))
			dash = false
		case len(out) > 0 && !dash:
			out = append(out, '-')
			dash = true
		}
	}
	if len(out) > maxSlugRunes {
		out = out[:maxSlugRunes]
	}
	s := strings.Trim(string(out), "-.")
	if s == "" {
		return "untitled"
	}
	return s
}

// DocBaseName is <YYYY-MM-DD from clippedAt>-<slug>: the document is saved as
// <base>.org in the category folder and its images under assets/<base>/.
func DocBaseName(clippedAt time.Time, title string) string {
	return clippedAt.Format("2006-01-02") + "-" + Slug(title)
}

// maxNameTries bounds the "-2", "-3", ... suffix search in FreeName.
const maxNameTries = 1000

// FreeName returns base, or base-2, base-3, ... — the first name for which
// neither <category>/<name>.org nor <category>/assets/<name> exists. Commit
// still fails closed if the name is taken by the time it runs.
func (s *Store) FreeName(category, base string) (string, error) {
	if !validDirName(base) || !validCategory(category) {
		return "", errBadName
	}
	for i := 1; i <= maxNameTries; i++ {
		name := base
		if i > 1 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		taken, err := s.taken(category, name)
		if err != nil {
			return "", err
		}
		if !taken {
			return name, nil
		}
	}
	return "", ErrExists
}

// taken reports whether a document or asset directory named name exists.
func (s *Store) taken(category, name string) (bool, error) {
	for _, p := range []string{category + "/" + name + ".org", category + "/assets/" + name} {
		if _, err := s.root.Lstat(p); err == nil {
			return true, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
	}
	return false, nil
}

func validCategory(category string) bool {
	for _, part := range strings.Split(category, "/") {
		if !validDirName(part) || part == "assets" {
			return false
		}
	}
	return true
}

// validDirName accepts a single path component that cannot traverse.
func validDirName(n string) bool {
	if n == "" || n == "." || n == ".." || strings.HasPrefix(n, ".") || strings.ContainsAny(n, "/\\") {
		return false
	}
	for _, r := range n {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// Lookup returns the ledger entry for requestID, or nil if there is none.
func (s *Store) Lookup(requestID string) (*protocol.Result, error) {
	if !protocol.ValidRequestID(requestID) {
		return nil, errBadName
	}
	f, err := s.root.Open(requestsDir + "/" + requestID + ".json")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxLedgerBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxLedgerBytes {
		return nil, fmt.Errorf("ledger entry too large")
	}
	var r protocol.Result
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("ledger entry unreadable: %w", err)
	}
	if r.RequestID != requestID {
		return nil, fmt.Errorf("ledger entry does not match its name")
	}
	return &r, nil
}

// Acquire takes the in-flight marker for requestID with O_EXCL. busy is true
// when another host process holds a marker younger than LockTTL. An older
// marker is considered stale, removed, and taken over. release removes the
// marker and must be called on every path once acquired.
func (s *Store) Acquire(requestID string) (release func(), busy bool, err error) {
	if !protocol.ValidRequestID(requestID) {
		return nil, false, errBadName
	}
	if err := s.root.MkdirAll(requestsDir, dirPerm); err != nil {
		return nil, false, err
	}
	name := requestsDir + "/" + requestID + ".lock"
	for attempt := 0; attempt < 2; attempt++ {
		f, err := s.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, filePerm)
		if err == nil {
			fmt.Fprintf(f, "%d %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
			f.Close()
			return func() { s.root.Remove(name) }, false, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, false, err
		}
		fi, err := s.root.Lstat(name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue // released between our open and stat
			}
			return nil, false, err
		}
		if time.Since(fi.ModTime()) < LockTTL {
			return nil, true, nil
		}
		// Stale: a previous host died mid-request. Two hosts racing to take
		// over both remove and retry; O_EXCL lets exactly one win.
		if err := s.root.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, false, err
		}
	}
	return nil, true, nil
}

// InFlight reports whether a fresh (younger than LockTTL) in-flight marker
// exists for requestID.
func (s *Store) InFlight(requestID string) bool {
	if !protocol.ValidRequestID(requestID) {
		return false
	}
	fi, err := s.root.Lstat(requestsDir + "/" + requestID + ".lock")
	return err == nil && time.Since(fi.ModTime()) < LockTTL
}

// WriteLedger stores the final result atomically (temp file + rename).
func (s *Store) WriteLedger(r protocol.Result) error {
	if !protocol.ValidRequestID(r.RequestID) {
		return errBadName
	}
	if r.FailedAssets == nil {
		r.FailedAssets = []protocol.FailedAsset{}
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if err := s.root.MkdirAll(requestsDir, dirPerm); err != nil {
		return err
	}
	tmp := requestsDir + "/" + r.RequestID + ".json.tmp-" + randHex(8)
	if err := s.writeNew(tmp, b); err != nil {
		return err
	}
	if err := s.root.Rename(tmp, requestsDir+"/"+r.RequestID+".json"); err != nil {
		s.root.Remove(tmp)
		return err
	}
	s.syncDir(requestsDir)
	return nil
}

// writeNew creates name exclusively, writes b, and fsyncs it.
func (s *Store) writeNew(name string, b []byte) error {
	f, err := s.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, filePerm)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		s.root.Remove(name)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		s.root.Remove(name)
		return err
	}
	return f.Close()
}

// syncDir best-effort fsyncs a directory so a rename is durable.
func (s *Store) syncDir(name string) {
	if d, err := s.root.Open(name); err == nil {
		d.Sync()
		d.Close()
	}
}

// CleanTmp removes staging entries older than maxAge. It only ever touches
// direct children of .org-clipper/tmp, through the Root, and RemoveAll does
// not follow symlinks, so nothing outside that directory can be removed.
func (s *Store) CleanTmp(maxAge time.Duration) {
	d, err := s.root.Open(tmpDir)
	if err != nil {
		return
	}
	entries, err := d.ReadDir(-1)
	d.Close()
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !validDirName(name) {
			continue
		}
		fi, err := s.root.Lstat(tmpDir + "/" + name)
		if err != nil || time.Since(fi.ModTime()) < maxAge {
			continue
		}
		s.root.RemoveAll(tmpDir + "/" + name)
	}
}

// Staging is a private, per-request directory under .org-clipper/tmp holding
// doc.org and assets/; Commit moves them into place only once complete.
type Staging struct {
	s      *Store
	dir    string // root-relative, slash separated
	hasDoc bool
}

const stagedDoc = "doc.org"

// Stage creates <root>/.org-clipper/tmp/<requestId>-<random>/assets.
func (s *Store) Stage(requestID string) (*Staging, error) {
	if !protocol.ValidRequestID(requestID) {
		return nil, errBadName
	}
	if err := s.root.MkdirAll(tmpDir, dirPerm); err != nil {
		return nil, err
	}
	dir := tmpDir + "/" + requestID + "-" + randHex(8)
	if err := s.root.Mkdir(dir, dirPerm); err != nil { // fails if it exists
		return nil, err
	}
	if err := s.root.Mkdir(dir+"/assets", dirPerm); err != nil {
		s.root.RemoveAll(dir)
		return nil, err
	}
	return &Staging{s: s, dir: dir}, nil
}

// Dir is the staging directory, relative to the root.
func (st *Staging) Dir() string { return st.dir }

// WriteAsset stores image bytes as assets/<name>. name must be the
// content-derived <32 hex>.<png|jpg|webp> form; an existing file of the same
// name is identical content (the name is its hash) and is kept.
func (st *Staging) WriteAsset(name string, data []byte) error {
	if !assetNameRe.MatchString(name) {
		return errBadName
	}
	err := st.s.writeNew(st.dir+"/assets/"+name, data)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	return err
}

// WriteDoc stores the Org document.
func (st *Staging) WriteDoc(content string) error {
	if err := st.s.writeNew(st.dir+"/"+stagedDoc, []byte(content)); err != nil {
		return err
	}
	st.hasDoc = true
	return nil
}

// Commit places the document at <category>/<name>.org and, when any image was
// saved, the images at <category>/assets/<name>/, then returns the
// root-relative document path. It never overwrites: a taken name fails closed
// with ErrExists. The asset directory moves first and the document last, so a
// visible document always has its images.
func (st *Staging) Commit(category, name string) (string, error) {
	if !validDirName(name) || !st.hasDoc || !validCategory(category) {
		return "", errBadName
	}
	s := st.s
	if taken, err := s.taken(category, name); err != nil {
		return "", err
	} else if taken {
		return "", ErrExists
	}
	entries, err := s.readDir(st.dir + "/assets")
	if err != nil {
		return "", err
	}
	if err := s.root.MkdirAll(category, dirPerm); err != nil {
		return "", err
	}
	assetsFinal := ""
	if len(entries) > 0 {
		s.syncDir(st.dir + "/assets")
		if err := s.root.MkdirAll(category+"/assets", dirPerm); err != nil {
			return "", err
		}
		assetsFinal = category + "/assets/" + name
		// @MX:WARN: [AUTO] check-then-rename is not atomic
		// @MX:REASON: os.Root.Rename exposes no RENAME_EXCL. rename(2) onto an existing EMPTY directory replaces it; onto a non-empty one it fails with ENOTEMPTY. The window between taken() and Rename only matters if another writer creates an empty directory with this exact name in that instant; the per-request lock serialises host processes for one requestId.
		if err := s.root.Rename(st.dir+"/assets", assetsFinal); err != nil {
			if errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, fs.ErrExist) {
				return "", ErrExists
			}
			return "", err
		}
		s.syncDir(category + "/assets")
	}
	docFinal := category + "/" + name + ".org"
	// A hard link creates the name exclusively (EEXIST if taken), unlike rename.
	if err := s.root.Link(st.dir+"/"+stagedDoc, docFinal); err != nil {
		if assetsFinal != "" {
			s.root.Rename(assetsFinal, st.dir+"/assets")
		}
		if errors.Is(err, fs.ErrExist) {
			return "", ErrExists
		}
		return "", err
	}
	s.syncDir(category)
	s.root.RemoveAll(st.dir)
	st.dir = ""
	return docFinal, nil
}

func (s *Store) readDir(name string) ([]fs.DirEntry, error) {
	d, err := s.root.Open(name)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return d.ReadDir(-1)
}

// Discard removes the staging directory if it has not been committed.
func (st *Staging) Discard() {
	if st.dir != "" {
		st.s.root.RemoveAll(st.dir)
		st.dir = ""
	}
}

// Classify maps a storage error to a protocol error code.
func Classify(err error) string {
	var pe *protocol.Error
	switch {
	case err == nil:
		return ""
	case errors.As(err, &pe):
		return pe.Code
	case errors.Is(err, syscall.ENOSPC), errors.Is(err, syscall.EDQUOT):
		return protocol.CodeDiskFull
	case errors.Is(err, syscall.EACCES), errors.Is(err, syscall.EPERM), errors.Is(err, fs.ErrPermission),
		errors.Is(err, syscall.ELOOP), isEscape(err):
		return protocol.CodePermissionDenied
	case errors.Is(err, fs.ErrNotExist):
		return protocol.CodeRootUnavailable
	}
	return protocol.CodeInternal
}

// isEscape recognises os.Root's refusal to follow a path out of the root.
// The sentinel is unexported in package os, so the message is matched.
func isEscape(err error) bool {
	return err != nil && strings.Contains(err.Error(), "path escapes from parent")
}

// Message returns fixed, host-authored text for an error code. It never
// includes a path (the root must not be disclosed).
func Message(code string) string {
	switch code {
	case protocol.CodeDiskFull:
		return "No space left on device"
	case protocol.CodePermissionDenied:
		return "Permission denied while writing the clip"
	case protocol.CodeRootUnavailable:
		return "The save root is not available"
	case protocol.CodeNotConfigured:
		return "The native host has no save root configured; run org-clipper-host setup"
	}
	return "Internal error while saving the clip"
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
