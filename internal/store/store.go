package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

func Open(path string) (*sql.DB, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(abs)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	u := url.URL{Scheme: "file", Path: uriPath}
	q := u.Query()
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Prepare creates the baseline schema without enabling relation enforcement.
// It is used by the importer while rows are copied in arbitrary table order.
func Prepare(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	var version int
	if err = tx.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		return err
	}
	if version > 3 {
		return fmt.Errorf("database schema version %d is newer than supported version 3", version)
	}
	if version == 0 {
		if _, err = tx.Exec(schema); err != nil {
			return fmt.Errorf("create baseline: %w", err)
		}
		if _, err = tx.Exec(`INSERT INTO schema_migrations VALUES (1,datetime('now'))`); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func Migrate(db *sql.DB) error {
	if err := Prepare(db); err != nil {
		return err
	}
	if err := enforceRelations(db); err != nil {
		return err
	}
	return enforceUniqueBusinessKeys(db)
}
