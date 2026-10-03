package notecache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type fileRow struct {
	path        string
	hash        string
	size        int64
	modTimeNS   int64
	document    *Document
	occurrences []OccurrenceRecord
	links       []Link
}

type store struct {
	db *sql.DB
}

func openStore(ctx context.Context, cacheDir, scopeKey string) (*store, error) {
	if cacheDir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		cacheDir = filepath.Join(base, "imoogi-emacs", "notes")
	}
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(cacheDir); err != nil {
		return nil, err
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("cache_dir must be a real directory: %s", cacheDir)
	}
	path := filepath.Join(cacheDir, scopeKey+".sqlite")
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("cache database must not be a symbolic link: %s", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return openStorePath(ctx, path, true)
}

func openStorePath(ctx context.Context, path string, retry bool) (*store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, `PRAGMA busy_timeout=5000`); err != nil {
		_ = db.Close()
		if retry && recoverableCacheDatabaseError(err) {
			_ = os.Remove(path)
			return openStorePath(ctx, path, false)
		}
		return nil, err
	}
	s := &store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		if retry && recoverableCacheDatabaseError(err) {
			_ = os.Remove(path)
			return openStorePath(ctx, path, false)
		}
		return nil, err
	}
	return s, nil
}

func (s *store) close() error {
	return s.db.Close()
}

func recoverableCacheDatabaseError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code() & 0xff
	return code == sqlite3.SQLITE_CORRUPT || code == sqlite3.SQLITE_NOTADB
}

func (s *store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS files(
  path TEXT PRIMARY KEY,
  hash TEXT NOT NULL,
  size INTEGER NOT NULL,
  mtime_ns INTEGER NOT NULL,
  is_document INTEGER NOT NULL,
  document_id TEXT NOT NULL,
  title TEXT NOT NULL,
  kind TEXT NOT NULL,
  identity_error TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS ids(
  id TEXT NOT NULL,
  file TEXT NOT NULL,
  position INTEGER NOT NULL,
  PRIMARY KEY(id, file, position)
);
CREATE TABLE IF NOT EXISTS links(
  file TEXT NOT NULL,
  source_id TEXT NOT NULL,
  target_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  position INTEGER NOT NULL,
  PRIMARY KEY(file, source_id, target_id, kind, position)
);
CREATE INDEX IF NOT EXISTS ids_id_idx ON ids(id);
CREATE INDEX IF NOT EXISTS links_target_idx ON links(target_id);
`)
	return err
}

func (s *store) parserVersionMatches(ctx context.Context) (bool, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key='parser_version'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return raw == fmt.Sprintf("%d", ParserVersion), nil
}

func (s *store) generation(ctx context.Context) (int64, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key='generation'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var generation int64
	_, _ = fmt.Sscanf(raw, "%d", &generation)
	return generation, nil
}

func txGeneration(ctx context.Context, tx *sql.Tx) (int64, error) {
	var raw string
	err := tx.QueryRowContext(ctx, `SELECT value FROM meta WHERE key='generation'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var generation int64
	_, _ = fmt.Sscanf(raw, "%d", &generation)
	return generation, nil
}

func (s *store) bumpGeneration(ctx context.Context, tx *sql.Tx) (int64, error) {
	next := time.Now().UnixNano()
	_, err := tx.ExecContext(ctx, `INSERT INTO meta(key,value) VALUES('generation',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, fmt.Sprintf("%d", next))
	return next, err
}

func (s *store) load(ctx context.Context, path, hash string) (*fileRow, bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT hash,size,mtime_ns,is_document,document_id,title,kind,identity_error FROM files WHERE path=?`, path)
	var storedHash, docID, title, kind, identityError string
	var size, modTime int64
	var isDocument int
	if err := row.Scan(&storedHash, &size, &modTime, &isDocument, &docID, &title, &kind, &identityError); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if storedHash != hash {
		return nil, false, nil
	}
	result := &fileRow{path: path, hash: hash, size: size, modTimeNS: modTime}
	if isDocument != 0 {
		result.document = &Document{File: path, ID: docID, Title: title, Kind: kind, IdentityError: identityError}
	}
	idRows, err := s.db.QueryContext(ctx, `SELECT id,position FROM ids WHERE file=? ORDER BY id,position`, path)
	if err != nil {
		return nil, false, err
	}
	defer idRows.Close()
	for idRows.Next() {
		var occurrence OccurrenceRecord
		occurrence.File = path
		if err := idRows.Scan(&occurrence.ID, &occurrence.Position); err != nil {
			return nil, false, err
		}
		result.occurrences = append(result.occurrences, occurrence)
	}
	if err := idRows.Err(); err != nil {
		return nil, false, err
	}
	linkRows, err := s.db.QueryContext(ctx, `SELECT source_id,target_id,kind,position FROM links WHERE file=? ORDER BY position`, path)
	if err != nil {
		return nil, false, err
	}
	defer linkRows.Close()
	for linkRows.Next() {
		link := Link{File: path}
		if err := linkRows.Scan(&link.SourceID, &link.TargetID, &link.Kind, &link.Position); err != nil {
			return nil, false, err
		}
		result.links = append(result.links, link)
	}
	return result, true, linkRows.Err()
}

func (s *store) save(ctx context.Context, scanned map[string]struct{}, rows []fileRow, invalidated []string) (bool, int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, err
	}
	changed := false
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	parserMatches := false
	var parserRaw string
	if metaErr := tx.QueryRowContext(ctx, `SELECT value FROM meta WHERE key='parser_version'`).Scan(&parserRaw); metaErr == nil {
		parserMatches = parserRaw == fmt.Sprintf("%d", ParserVersion)
	} else if errors.Is(metaErr, sql.ErrNoRows) {
		parserMatches = false
	} else {
		return false, 0, metaErr
	}
	if !parserMatches {
		changed = true
	}
	for _, path := range invalidated {
		if err = deletePath(ctx, tx, path); err != nil {
			return false, 0, err
		}
		changed = true
	}
	existing, err := tx.QueryContext(ctx, `SELECT path FROM files`)
	if err != nil {
		return false, 0, err
	}
	for existing.Next() {
		var path string
		if scanErr := existing.Scan(&path); scanErr != nil {
			_ = existing.Close()
			return false, 0, scanErr
		}
		if _, ok := scanned[path]; !ok {
			if err = deletePath(ctx, tx, path); err != nil {
				_ = existing.Close()
				return false, 0, err
			}
			changed = true
		}
	}
	if err = existing.Close(); err != nil {
		return false, 0, err
	}
	for _, row := range rows {
		isDocument := 0
		docID, title, kind, identityError := "", "", "", ""
		if row.document != nil {
			isDocument = 1
			docID, title, kind, identityError = row.document.ID, row.document.Title, row.document.Kind, row.document.IdentityError
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO files(path,hash,size,mtime_ns,is_document,document_id,title,kind,identity_error)
VALUES(?,?,?,?,?,?,?,?,?)
ON CONFLICT(path) DO UPDATE SET hash=excluded.hash,size=excluded.size,mtime_ns=excluded.mtime_ns,is_document=excluded.is_document,document_id=excluded.document_id,title=excluded.title,kind=excluded.kind,identity_error=excluded.identity_error`,
			row.path, row.hash, row.size, row.modTimeNS, isDocument, docID, title, kind, identityError); err != nil {
			return false, 0, err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM ids WHERE file=?`, row.path); err != nil {
			return false, 0, err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM links WHERE file=?`, row.path); err != nil {
			return false, 0, err
		}
		for _, occurrence := range row.occurrences {
			if _, err = tx.ExecContext(ctx, `INSERT INTO ids(id,file,position) VALUES(?,?,?)`, occurrence.ID, occurrence.File, occurrence.Position); err != nil {
				return false, 0, err
			}
		}
		for _, link := range row.links {
			if _, err = tx.ExecContext(ctx, `INSERT INTO links(file,source_id,target_id,kind,position) VALUES(?,?,?,?,?)`, link.File, link.SourceID, link.TargetID, link.Kind, link.Position); err != nil {
				return false, 0, err
			}
		}
		changed = true
	}
	generation, err := txGeneration(ctx, tx)
	if err != nil {
		return false, 0, err
	}
	if changed {
		if _, err = tx.ExecContext(ctx, `INSERT INTO meta(key,value) VALUES('parser_version',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, fmt.Sprintf("%d", ParserVersion)); err != nil {
			return false, 0, err
		}
		generation, err = s.bumpGeneration(ctx, tx)
		if err != nil {
			return false, 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return false, 0, err
	}
	return changed, generation, nil
}

func deletePath(ctx context.Context, tx *sql.Tx, path string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM files WHERE path=?`, path); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM ids WHERE file=?`, path); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM links WHERE file=?`, path); err != nil {
		return err
	}
	return nil
}
