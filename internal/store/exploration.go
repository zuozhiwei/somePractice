package store

import "database/sql"

func ensureExploration(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS atlas_visits (
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		map_id INTEGER NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
		visited_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(user_id, map_id)
	)`)
	return err
}
