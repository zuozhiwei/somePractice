package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"skilltree-go/internal/store"
)

type column struct{ name, typ string }
type summary struct {
	Table    string `json:"table"`
	Rows     int    `json:"rows"`
	Verified bool   `json:"verified"`
}
type report struct {
	Tables          []summary        `json:"tables"`
	Orphans         map[string]int64 `json:"orphans"`
	Duplicates      map[string]int64 `json:"duplicates"`
	Integrity       string           `json:"integrity"`
	ReadyForCutover bool             `json:"ready_for_cutover"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	dsn := os.Getenv("SKILLTREE_SOURCE_DATABASE_URL")
	dest := os.Getenv("SKILLTREE_MIGRATION_OUTPUT")
	if dsn == "" || dest == "" {
		return fmt.Errorf("SKILLTREE_SOURCE_DATABASE_URL and SKILLTREE_MIGRATION_OUTPUT are required")
	}
	dest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return err
	}
	// Reserve the final name exclusively; never overwrite an existing database.
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("reserve destination: %w", err)
	}
	f.Close()
	success := false
	defer func() {
		if !success {
			os.Remove(dest)
			os.Remove(dest + "-wal")
			os.Remove(dest + "-shm")
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	src, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open source failed")
	}
	defer src.Close()
	source, err := src.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return fmt.Errorf("source read-only transaction: %w", err)
	}
	defer source.Rollback()
	db, err := store.Open(dest)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = store.Prepare(db); err != nil {
		return err
	}
	target, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer target.Rollback()
	rows, err := source.QueryContext(ctx, `SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name`)
	if err != nil {
		return err
	}
	var names []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		names = append(names, name)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	var expected int
	if err = target.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name <> 'schema_migrations'`).Scan(&expected); err != nil {
		return err
	}
	if len(names) != expected {
		return fmt.Errorf("source has %d tables, baseline expects %d; inspect schema drift", len(names), expected)
	}
	rep := report{Tables: []summary{}, Orphans: map[string]int64{}}
	for _, name := range names {
		cols, err := columns(ctx, source, name)
		if err != nil {
			return err
		}
		targetCols, err := target.QueryContext(ctx, `SELECT name, type FROM pragma_table_info(?) ORDER BY cid`, name)
		if err != nil {
			return err
		}
		var tc []column
		for targetCols.Next() {
			var c column
			if err = targetCols.Scan(&c.name, &c.typ); err != nil {
				targetCols.Close()
				return err
			}
			tc = append(tc, c)
		}
		if err = targetCols.Err(); err != nil {
			targetCols.Close()
			return err
		}
		targetCols.Close()
		if len(tc) != len(cols) {
			return fmt.Errorf("schema drift in %s: column count mismatch", name)
		}
		for i, c := range cols {
			if c.name != tc[i].name || sqliteType(c.typ) != tc[i].typ {
				return fmt.Errorf("schema drift in %s: column mismatch", name)
			}
		}
		var qcols, marks []string
		for _, c := range cols {
			qcols = append(qcols, pgx.Identifier{c.name}.Sanitize())
			marks = append(marks, "?")
		}
		query := "SELECT " + strings.Join(qcols, ",") + " FROM " + pgx.Identifier{"public", name}.Sanitize()
		sr, err := source.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		stmt, err := target.PrepareContext(ctx, "INSERT INTO "+pgx.Identifier{name}.Sanitize()+" ("+strings.Join(qcols, ",")+") VALUES ("+strings.Join(marks, ",")+")")
		if err != nil {
			sr.Close()
			return err
		}
		var sourceHashes []string
		for sr.Next() {
			vals, err := readRow(sr, cols)
			if err != nil {
				stmt.Close()
				sr.Close()
				return err
			}
			h, err := digest(vals)
			if err != nil {
				return err
			}
			sourceHashes = append(sourceHashes, h)
			if _, err = stmt.ExecContext(ctx, vals...); err != nil {
				stmt.Close()
				sr.Close()
				return fmt.Errorf("insert %s: %w", name, err)
			}
		}
		if err = sr.Err(); err != nil {
			stmt.Close()
			sr.Close()
			return err
		}
		sr.Close()
		stmt.Close()
		tr, err := target.QueryContext(ctx, "SELECT "+strings.Join(qcols, ",")+" FROM "+pgx.Identifier{name}.Sanitize())
		if err != nil {
			return err
		}
		var targetHashes []string
		for tr.Next() {
			vals, err := readRow(tr, cols)
			if err != nil {
				tr.Close()
				return err
			}
			h, err := digest(vals)
			if err != nil {
				return err
			}
			targetHashes = append(targetHashes, h)
		}
		if err = tr.Err(); err != nil {
			tr.Close()
			return err
		}
		tr.Close()
		sort.Strings(sourceHashes)
		sort.Strings(targetHashes)
		if len(sourceHashes) != len(targetHashes) {
			return fmt.Errorf("row count mismatch: %s", name)
		}
		for i := range sourceHashes {
			if sourceHashes[i] != targetHashes[i] {
				return fmt.Errorf("row content mismatch: %s", name)
			}
		}
		rep.Tables = append(rep.Tables, summary{name, len(sourceHashes), true})
	}
	checks := map[string]string{
		"invite_codes.team_id":        `SELECT count(*) FROM invite_codes c LEFT JOIN teams p ON p.id=c.team_id WHERE c.team_id IS NOT NULL AND p.id IS NULL`,
		"tasks.skill_id":              `SELECT count(*) FROM tasks c LEFT JOIN skills p ON p.id=c.skill_id WHERE c.skill_id IS NOT NULL AND p.id IS NULL`,
		"tutorials.skill_id":          `SELECT count(*) FROM tutorials c LEFT JOIN skills p ON p.id=c.skill_id WHERE c.skill_id IS NOT NULL AND p.id IS NULL`,
		"user_skill_mastery.user_id":  `SELECT count(*) FROM user_skill_mastery c LEFT JOIN users p ON p.id=c.user_id WHERE p.id IS NULL`,
		"user_skill_mastery.skill_id": `SELECT count(*) FROM user_skill_mastery c LEFT JOIN skills p ON p.id=c.skill_id WHERE p.id IS NULL`,
		"submissions.reviewer_id":     `SELECT count(*) FROM user_task_submissions c LEFT JOIN users p ON p.id=c.reviewer_id WHERE c.reviewer_id IS NOT NULL AND p.id IS NULL`,
		"users.team_id":               `SELECT count(*) FROM users c LEFT JOIN teams p ON p.id=c.team_id WHERE c.team_id IS NOT NULL AND p.id IS NULL`,
		"map_progress.user_id":        `SELECT count(*) FROM user_map_progress c LEFT JOIN users p ON p.id=c.user_id WHERE c.user_id IS NOT NULL AND p.id IS NULL`,
		"map_progress.map_id":         `SELECT count(*) FROM user_map_progress c LEFT JOIN maps p ON p.id=c.map_id WHERE c.map_id IS NOT NULL AND p.id IS NULL`,
		"user_teams.user_id":          `SELECT count(*) FROM user_teams c LEFT JOIN users p ON p.id=c.user_id WHERE p.id IS NULL`,
		"user_teams.team_id":          `SELECT count(*) FROM user_teams c LEFT JOIN teams p ON p.id=c.team_id WHERE p.id IS NULL`,
		"conversations.owner_id":      `SELECT count(*) FROM conversations c LEFT JOIN users p ON p.id=c.owner_id WHERE c.owner_id IS NOT NULL AND p.id IS NULL`,
		"levels.map_id":               `SELECT count(*) FROM levels c LEFT JOIN maps p ON p.id=c.map_id WHERE c.map_id IS NOT NULL AND p.id IS NULL`,
		"tasks.level_id":              `SELECT count(*) FROM tasks c LEFT JOIN levels p ON p.id=c.level_id WHERE c.level_id IS NOT NULL AND p.id IS NULL`,
		"tutorials.level_id":          `SELECT count(*) FROM tutorials c LEFT JOIN levels p ON p.id=c.level_id WHERE c.level_id IS NOT NULL AND p.id IS NULL`,
		"progress.user_id":            `SELECT count(*) FROM user_level_progress c LEFT JOIN users p ON p.id=c.user_id WHERE p.id IS NULL`,
		"progress.level_id":           `SELECT count(*) FROM user_level_progress c LEFT JOIN levels p ON p.id=c.level_id WHERE p.id IS NULL`,
		"submissions.user_id":         `SELECT count(*) FROM user_task_submissions c LEFT JOIN users p ON p.id=c.user_id WHERE c.user_id IS NOT NULL AND p.id IS NULL`,
		"submissions.task_id":         `SELECT count(*) FROM user_task_submissions c LEFT JOIN tasks p ON p.id=c.task_id WHERE c.task_id IS NOT NULL AND p.id IS NULL`,
		"messages.conversation_id":    `SELECT count(*) FROM chat_messages c LEFT JOIN conversations p ON p.id=c.conversation_id WHERE p.id IS NULL`,
		"messages.sender_id":          `SELECT count(*) FROM chat_messages c LEFT JOIN users p ON p.id=c.sender_id WHERE p.id IS NULL`,
		"members.conversation_id":     `SELECT count(*) FROM conversation_members c LEFT JOIN conversations p ON p.id=c.conversation_id WHERE p.id IS NULL`,
		"members.user_id":             `SELECT count(*) FROM conversation_members c LEFT JOIN users p ON p.id=c.user_id WHERE p.id IS NULL`,
		"mentorship.mentor_id":        `SELECT count(*) FROM mentorships c LEFT JOIN users p ON p.id=c.mentor_id WHERE c.mentor_id IS NOT NULL AND p.id IS NULL`,
		"mentorship.apprentice_id":    `SELECT count(*) FROM mentorships c LEFT JOIN users p ON p.id=c.apprentice_id WHERE c.apprentice_id IS NOT NULL AND p.id IS NULL`,
	}
	rep.ReadyForCutover = true
	for name, q := range checks {
		var n int64
		if err = target.QueryRowContext(ctx, q).Scan(&n); err != nil {
			return err
		}
		rep.Orphans[name] = n
		if n > 0 {
			rep.ReadyForCutover = false
		}
	}
	rep.Duplicates = map[string]int64{}
	for name, q := range map[string]string{
		"awarded_task":         `SELECT count(*) FROM (SELECT user_id,task_id FROM user_task_submissions WHERE xp_granted=1 GROUP BY user_id,task_id HAVING count(*)>1)`,
		"map_progress":         `SELECT count(*) FROM (SELECT user_id,map_id FROM user_map_progress WHERE user_id IS NOT NULL AND map_id IS NOT NULL GROUP BY user_id,map_id HAVING count(*)>1)`,
		"conversation_members": `SELECT count(*) FROM (SELECT conversation_id,user_id FROM conversation_members GROUP BY conversation_id,user_id HAVING count(*)>1)`,
		"group_owner":          `SELECT count(*) FROM (SELECT owner_id FROM conversations WHERE type='GROUP' AND owner_id IS NOT NULL GROUP BY owner_id HAVING count(*)>1)`,
	} {
		var n int64
		if err = target.QueryRowContext(ctx, q).Scan(&n); err != nil {
			return err
		}
		rep.Duplicates[name] = n
		if n > 0 {
			rep.ReadyForCutover = false
		}
	}
	if err = target.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&rep.Integrity); err != nil {
		return err
	}
	if rep.Integrity != "ok" {
		return fmt.Errorf("SQLite integrity check failed")
	}
	if err = target.Commit(); err != nil {
		return err
	}
	if err = source.Commit(); err != nil {
		return err
	}
	if _, err = db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return err
	}
	if err = db.Close(); err != nil {
		return err
	}
	success = true
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}
func sqliteType(t string) string {
	switch t {
	case "bigint", "integer", "boolean":
		return "INTEGER"
	case "jsonb", "timestamp without time zone", "character varying", "text":
		return "TEXT"
	default:
		return "UNKNOWN"
	}
}
func columns(ctx context.Context, tx *sql.Tx, name string) ([]column, error) {
	r, err := tx.QueryContext(ctx, `SELECT column_name,data_type FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 ORDER BY ordinal_position`, name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var out []column
	for r.Next() {
		var c column
		if err = r.Scan(&c.name, &c.typ); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, r.Err()
}
func readRow(rows *sql.Rows, cols []column) ([]any, error) {
	v := make([]any, len(cols))
	p := make([]any, len(cols))
	for i := range v {
		p[i] = &v[i]
	}
	if err := rows.Scan(p...); err != nil {
		return nil, err
	}
	for i := range v {
		switch x := v[i].(type) {
		case []byte:
			v[i] = string(x)
		case bool:
			if x {
				v[i] = int64(1)
			} else {
				v[i] = int64(0)
			}
		case time.Time:
			v[i] = x.Format("2006-01-02 15:04:05.999999999")
		}
		if cols[i].typ == "jsonb" && v[i] != nil {
			var obj any
			dec := json.NewDecoder(strings.NewReader(v[i].(string)))
			dec.UseNumber()
			if err := dec.Decode(&obj); err != nil {
				return nil, err
			}
			b, err := json.Marshal(obj)
			if err != nil {
				return nil, err
			}
			v[i] = string(b)
		}
	}
	return v, nil
}
func digest(v []any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}
