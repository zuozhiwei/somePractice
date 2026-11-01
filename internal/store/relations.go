package store

import (
	"database/sql"
	"fmt"
)

type relation struct{ child, column, parent string }

// Only actual ID references are listed; historical created_by values include
// system sentinels and are intentionally not treated as foreign keys.
var relations = []relation{
	{"users", "team_id", "teams"}, {"invite_codes", "team_id", "teams"},
	{"levels", "map_id", "maps"}, {"tasks", "level_id", "levels"}, {"tasks", "skill_id", "skills"},
	{"tutorials", "level_id", "levels"}, {"tutorials", "skill_id", "skills"},
	{"mentorships", "mentor_id", "users"}, {"mentorships", "apprentice_id", "users"},
	{"user_level_progress", "user_id", "users"}, {"user_level_progress", "level_id", "levels"},
	{"user_map_progress", "user_id", "users"}, {"user_map_progress", "map_id", "maps"},
	{"user_teams", "user_id", "users"}, {"user_teams", "team_id", "teams"},
	{"user_skill_mastery", "user_id", "users"}, {"user_skill_mastery", "skill_id", "skills"},
	{"user_task_submissions", "user_id", "users"}, {"user_task_submissions", "task_id", "tasks"}, {"user_task_submissions", "reviewer_id", "users"},
	{"conversations", "owner_id", "users"}, {"chat_messages", "sender_id", "users"}, {"chat_messages", "conversation_id", "conversations"},
	{"conversation_members", "user_id", "users"}, {"conversation_members", "conversation_id", "conversations"},
}

func enforceRelations(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		return err
	}
	if version >= 2 {
		return tx.Commit()
	}
	for _, r := range relations {
		var n int
		q := fmt.Sprintf(`SELECT count(*) FROM "%s" c LEFT JOIN "%s" p ON p.id=c."%s" WHERE c."%s" IS NOT NULL AND p.id IS NULL`, r.child, r.parent, r.column, r.column)
		if err = tx.QueryRow(q).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			return fmt.Errorf("relation %s.%s contains %d orphan rows; repair the migration before starting", r.child, r.column, n)
		}
		for _, operation := range []string{"INSERT", "UPDATE"} {
			name := fmt.Sprintf("rel_%s_%s_%s", r.child, r.column, operation)
			ddl := fmt.Sprintf(`CREATE TRIGGER "%s" BEFORE %s ON "%s" WHEN NEW."%s" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "%s" WHERE id=NEW."%s") BEGIN SELECT RAISE(ABORT,'invalid relation %s.%s'); END`, name, operation, r.child, r.column, r.parent, r.column, r.child, r.column)
			if _, err = tx.Exec(ddl); err != nil {
				return err
			}
		}
		name := fmt.Sprintf("rel_%s_%s_DELETE", r.child, r.column)
		ddl := fmt.Sprintf(`CREATE TRIGGER "%s" BEFORE DELETE ON "%s" WHEN EXISTS (SELECT 1 FROM "%s" WHERE "%s"=OLD.id) BEGIN SELECT RAISE(ABORT,'referenced by %s.%s'); END`, name, r.parent, r.child, r.column, r.child, r.column)
		if _, err = tx.Exec(ddl); err != nil {
			return err
		}
		name = fmt.Sprintf("rel_%s_%s_PARENT_UPDATE", r.child, r.column)
		ddl = fmt.Sprintf(`CREATE TRIGGER "%s" BEFORE UPDATE OF id ON "%s" WHEN NEW.id<>OLD.id AND EXISTS (SELECT 1 FROM "%s" WHERE "%s"=OLD.id) BEGIN SELECT RAISE(ABORT,'referenced by %s.%s'); END`, name, r.parent, r.child, r.column, r.child, r.column)
		if _, err = tx.Exec(ddl); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`INSERT INTO schema_migrations VALUES(2,datetime('now'))`); err != nil {
		return err
	}
	return tx.Commit()
}

func enforceUniqueBusinessKeys(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		return err
	}
	if version >= 3 {
		return tx.Commit()
	}
	statements := []string{
		`CREATE UNIQUE INDEX uq_awarded_task ON user_task_submissions(user_id,task_id) WHERE xp_granted=1`,
		`CREATE UNIQUE INDEX uq_map_progress ON user_map_progress(user_id,map_id)`,
		`CREATE UNIQUE INDEX uq_conversation_member ON conversation_members(conversation_id,user_id)`,
		`CREATE UNIQUE INDEX uq_group_owner ON conversations(owner_id) WHERE type='GROUP'`,
	}
	for _, q := range statements {
		if _, err = tx.Exec(q); err != nil {
			return fmt.Errorf("duplicate business keys prevent migration: %w", err)
		}
	}
	if _, err = tx.Exec(`INSERT INTO schema_migrations VALUES(3,datetime('now'))`); err != nil {
		return err
	}
	return tx.Commit()
}
