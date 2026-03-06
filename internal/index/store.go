package index

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// FileInfo holds basic file metadata from the index.
type FileInfo struct {
	Path  string
	Mtime int64
	Size  int64
}

// StatsInfo holds aggregate statistics about the index.
type StatsInfo struct {
	FileCount int
}

// Store wraps a SQLite database for the vaultquery index.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) a SQLite index database at dbPath.
func Open(dbPath string) (*Store, error) {
	dsn := dbPath
	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB.
func (s *Store) DB() *sql.DB {
	return s.db
}

// BeginTx starts a new transaction.
func (s *Store) BeginTx() (*sql.Tx, error) {
	return s.db.Begin()
}

// UpsertFile inserts or updates a file record and returns its ID.
func (s *Store) UpsertFile(path string, mtime, size int64, title string) (int64, error) {
	return upsertFile(s.db, path, mtime, size, title)
}

// UpsertFileTx is the transaction-aware version of UpsertFile.
func (s *Store) UpsertFileTx(tx *sql.Tx, path string, mtime, size int64, title string) (int64, error) {
	return upsertFile(tx, path, mtime, size, title)
}

type querier interface {
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
}

func upsertFile(q querier, path string, mtime, size int64, title string) (int64, error) {
	var id int64
	err := q.QueryRow(
		`INSERT INTO files (path, mtime, size, title) VALUES (?, ?, ?, ?)
		 ON CONFLICT(path) DO UPDATE SET mtime=excluded.mtime, size=excluded.size, title=excluded.title
		 RETURNING id`,
		path, mtime, size, title,
	).Scan(&id)
	return id, err
}

// DeleteFile removes a file (and its fields via CASCADE) from the index.
func (s *Store) DeleteFile(path string) error {
	return deleteFile(s.db, path)
}

// DeleteFileTx is the transaction-aware version of DeleteFile.
func (s *Store) DeleteFileTx(tx *sql.Tx, path string) error {
	return deleteFile(tx, path)
}

func deleteFile(q querier, path string) error {
	_, err := q.Exec("DELETE FROM files WHERE path=?", path)
	return err
}

// ListFiles returns all indexed files.
func (s *Store) ListFiles() ([]FileInfo, error) {
	rows, err := s.db.Query("SELECT path, mtime, size FROM files")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var files []FileInfo
	for rows.Next() {
		var f FileInfo
		if err := rows.Scan(&f.Path, &f.Mtime, &f.Size); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// SetFields replaces all fields for a given file ID.
func (s *Store) SetFields(fileID int64, fields map[string][]string) error {
	return setFields(s.db, fileID, fields)
}

// SetFieldsTx is the transaction-aware version of SetFields.
func (s *Store) SetFieldsTx(tx *sql.Tx, fileID int64, fields map[string][]string) error {
	return setFields(tx, fileID, fields)
}

func setFields(q querier, fileID int64, fields map[string][]string) error {
	if _, err := q.Exec("DELETE FROM fields WHERE file_id=?", fileID); err != nil {
		return err
	}
	for key, values := range fields {
		for _, val := range values {
			if _, err := q.Exec("INSERT INTO fields (file_id, key, value) VALUES (?, ?, ?)", fileID, key, val); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetMeta retrieves a metadata value by key. Returns "" if not found.
func (s *Store) GetMeta(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM meta WHERE key=?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetMeta sets a metadata key-value pair.
func (s *Store) SetMeta(key, value string) error {
	_, err := s.db.Exec(
		"INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",
		key, value,
	)
	return err
}

// Stats returns aggregate statistics about the index.
func (s *Store) Stats() (*StatsInfo, error) {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM files").Scan(&count); err != nil {
		return nil, err
	}
	return &StatsInfo{FileCount: count}, nil
}

// DropAll drops all tables and recreates the schema.
func (s *Store) DropAll() error {
	for _, table := range []string{"fields", "files", "meta"} {
		if _, err := s.db.Exec("DROP TABLE IF EXISTS " + table); err != nil {
			return err
		}
	}
	return migrate(s.db)
}
