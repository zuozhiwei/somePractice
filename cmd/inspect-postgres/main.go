// inspect-postgres reads only metadata and counts, without dumping user data.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type column struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Nullable bool    `json:"nullable"`
	Default  *string `json:"default,omitempty"`
}
type table struct {
	Name    string   `json:"name"`
	Rows    int64    `json:"rows"`
	Columns []column `json:"columns"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	dsn := os.Getenv("SKILLTREE_SOURCE_DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("SKILLTREE_SOURCE_DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database failed")
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return fmt.Errorf("begin read-only transaction: %w", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name`)
	if err != nil {
		return err
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return err
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	result := make([]table, 0, len(names))
	for _, name := range names {
		t := table{Name: name, Columns: []column{}}
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM "+pgx.Identifier{"public", name}.Sanitize()).Scan(&t.Rows); err != nil {
			return err
		}
		cr, err := tx.QueryContext(ctx, `SELECT column_name, data_type, is_nullable='YES', column_default FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 ORDER BY ordinal_position`, name)
		if err != nil {
			return err
		}
		for cr.Next() {
			var c column
			if err := cr.Scan(&c.Name, &c.Type, &c.Nullable, &c.Default); err != nil {
				cr.Close()
				return err
			}
			t.Columns = append(t.Columns, c)
		}
		if err := cr.Err(); err != nil {
			cr.Close()
			return err
		}
		cr.Close()
		result = append(result, t)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
