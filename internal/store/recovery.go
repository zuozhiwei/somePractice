package store

import "database/sql"

func ensureAccountRecovery(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS password_recovery (
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			method TEXT NOT NULL CHECK(method IN ('email','admin')),
			digest BLOB NOT NULL, issued_at INTEGER NOT NULL, expires_at INTEGER NOT NULL,
			attempts_left INTEGER NOT NULL,
			PRIMARY KEY(user_id,method)
		);
		CREATE TABLE IF NOT EXISTS login_versions (
			user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			version INTEGER NOT NULL
		);
		-- A deleted SQLite rowid and username may be reused. New accounts
		-- start in a fresh session generation, including direct SQL inserts.
		CREATE TRIGGER IF NOT EXISTS initialize_session_after_user_insert
		AFTER INSERT ON users
		BEGIN
			INSERT INTO login_versions(user_id,version)
			VALUES(NEW.id, (random() & 4611686018427387903) + 1);
		END;
		CREATE TRIGGER IF NOT EXISTS recovery_unique_email_insert
		BEFORE INSERT ON users WHEN coalesce(trim(NEW.email),'')<>'' AND EXISTS(
			SELECT 1 FROM users WHERE lower(trim(email))=lower(trim(NEW.email)))
		BEGIN SELECT RAISE(ABORT,'UNIQUE constraint failed: users.email'); END;
		CREATE TRIGGER IF NOT EXISTS recovery_unique_email_update
		BEFORE UPDATE OF email ON users WHEN coalesce(trim(NEW.email),'')<>'' AND lower(trim(OLD.email)) IS NOT lower(trim(NEW.email)) AND EXISTS(
			SELECT 1 FROM users WHERE id<>OLD.id AND lower(trim(email))=lower(trim(NEW.email)))
		BEGIN SELECT RAISE(ABORT,'UNIQUE constraint failed: users.email'); END;
		CREATE TRIGGER IF NOT EXISTS invalidate_recovery_after_account_change
		AFTER UPDATE OF username,email,role,password ON users
		WHEN OLD.username IS NOT NEW.username OR OLD.email IS NOT NEW.email OR OLD.role IS NOT NEW.role OR OLD.password IS NOT NEW.password
		BEGIN DELETE FROM password_recovery WHERE user_id=NEW.id; END;
		CREATE TRIGGER IF NOT EXISTS invalidate_session_after_password_change
		AFTER UPDATE OF password ON users WHEN OLD.password IS NOT NEW.password
		BEGIN
			INSERT INTO login_versions(user_id,version) VALUES(NEW.id,1)
			ON CONFLICT(user_id) DO UPDATE SET version=version+1;
		END;
	`)
	if err != nil {
		return err
	}
	return tx.Commit()
}
