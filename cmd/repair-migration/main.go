// repair-migration archives chat membership records referencing deleted users.
// It must only be run on an offline migration copy, not a running database.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"skilltree-go/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	path := os.Getenv("SKILLTREE_MIGRATION_OUTPUT")
	if path == "" {
		return fmt.Errorf("SKILLTREE_MIGRATION_OUTPUT is required")
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	db, err := store.Open(path)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil || version < 1 || version > 3 {
		return fmt.Errorf("unsupported schema version")
	}
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS migration_orphan_members (id INTEGER PRIMARY KEY,conversation_id INTEGER NOT NULL,joined_at TEXT,last_read_at TEXT,user_id INTEGER NOT NULL,archived_at TEXT NOT NULL)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO migration_orphan_members(id,conversation_id,joined_at,last_read_at,user_id,archived_at) SELECT c.id,c.conversation_id,c.joined_at,c.last_read_at,c.user_id,datetime('now') FROM conversation_members c LEFT JOIN users u ON u.id=c.user_id WHERE u.id IS NULL`)
	if err != nil {
		return err
	}
	result, err := tx.Exec(`DELETE FROM conversation_members WHERE id IN (SELECT m.id FROM conversation_members m LEFT JOIN users u ON u.id=m.user_id WHERE u.id IS NULL)`)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if err = store.Migrate(db); err != nil {
		return fmt.Errorf("relation validation failed: %w", err)
	}
	_, err = db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"archivedMemberships": n, "schemaVersion": 3, "relationsEnforced": true})
}
