package store

import "database/sql"

func ensureMentorshipRanks(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var installed bool
	if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE type='trigger' AND name='end_mentorship_after_rank_change')`).Scan(&installed); err != nil {
		return err
	}
	if installed {
		return tx.Commit()
	}
	if _, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_mentorship_mentor ON mentorships(mentor_id,status);
		UPDATE mentorships AS m SET status='ENDED'
		WHERE status='ACTIVE' AND EXISTS (
			SELECT 1 FROM users AS mentor JOIN users AS apprentice
			WHERE mentor.id=m.mentor_id AND apprentice.id=m.apprentice_id
				AND COALESCE(apprentice.level,0)>COALESCE(mentor.level,0)
		);
		CREATE TRIGGER end_mentorship_after_rank_change
		AFTER UPDATE OF level ON users
		WHEN COALESCE(NEW.level,0) <> COALESCE(OLD.level,0)
		BEGIN
			UPDATE mentorships SET status='ENDED'
			WHERE status='ACTIVE' AND (
				(apprentice_id=NEW.id AND EXISTS (
					SELECT 1 FROM users WHERE id=mentor_id AND COALESCE(level,0)<COALESCE(NEW.level,0)
				))
				OR (mentor_id=NEW.id AND EXISTS (
					SELECT 1 FROM users WHERE id=apprentice_id AND COALESCE(level,0)>COALESCE(NEW.level,0)
				))
			);
		END;
	`); err != nil {
		return err
	}
	return tx.Commit()
}
